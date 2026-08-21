package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"gorhino/internal/master/agg"
	"gorhino/internal/master/api"
	"gorhino/internal/master/cluster"
	"gorhino/internal/master/store"
	"gorhino/internal/master/ws"
	"gorhino/internal/proto"
	"gorhino/internal/shared/model"
	"gorhino/internal/shared/validate"
	"gorhino/internal/timeutil"

	"google.golang.org/grpc"
)

var (
	ErrAlreadyRunning = fmt.Errorf("%w: a task is already running", store.ErrConflict)
	ErrNoWorkers      = fmt.Errorf("%w: no alive workers", store.ErrConflict)
	ErrNotDraft       = fmt.Errorf("%w: task is not draft", store.ErrConflict)
)

type App struct {
	log     *slog.Logger
	store   *store.Store
	cluster *cluster.Cluster
	agg     *agg.Aggregator
	hub     *ws.Hub
	mu      sync.Mutex
	runID   string
	stopper context.CancelFunc
}

func New(log *slog.Logger, st *store.Store) *App {
	a := &App{
		log:     log,
		store:   st,
		cluster: cluster.New(log, "master-1"),
		agg:     agg.New(),
		hub:     ws.New(log),
	}
	a.cluster.SetStatsHandler(a.agg.Ingest)
	return a
}

func (a *App) Cluster() *cluster.Cluster { return a.cluster }

func (a *App) StartLoops(ctx context.Context) {
	go a.tickLoop(ctx)
	go a.reapLoop(ctx)
}

func (a *App) tickLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.tick()
		}
	}
}

func (a *App) reapLoop(ctx context.Context) {
	t := time.NewTicker(time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			a.cluster.Reap(10 * time.Second)
		}
	}
}

func (a *App) tick() {
	a.mu.Lock()
	id := a.runID
	a.mu.Unlock()
	if id == "" {
		return
	}
	sn := a.agg.Flush(a.cluster.AliveCount())
	if sn == nil {
		return
	}
	if err := a.store.InsertSnapshot(*sn); err != nil {
		a.log.Error("snapshot", "err", err)
	}
	a.hub.Broadcast(sn)
	task, err := a.store.GetTask(id)
	if err != nil {
		return
	}
	if a.agg.Elapsed() >= time.Duration(task.DurationSec)*time.Second {
		a.finish(id, model.StatusCompleted)
	}
}

func (a *App) finish(id, status string) {
	a.mu.Lock()
	if a.runID != id {
		a.mu.Unlock()
		return
	}
	a.runID = ""
	if a.stopper != nil {
		a.stopper()
		a.stopper = nil
	}
	a.mu.Unlock()
	a.cluster.DispatchStop(id)
	if err := a.store.FinishTask(id, status); err != nil {
		a.log.Error("finish task", "err", err)
	}
	if rep := a.agg.Final(status); rep != nil {
		if err := a.store.UpsertReport(*rep); err != nil {
			a.log.Error("report", "err", err)
		}
		rep.Status = status
		sn := a.agg.Flush(a.cluster.AliveCount())
		if sn != nil {
			sn.Status = status
			a.hub.Broadcast(sn)
		}
	}
	a.log.Info("task finished", "id", id, "status", status)
}

func (a *App) Health() map[string]any {
	return map[string]any{
		"service": "master",
		"time":    timeutil.NowString(),
		"workers": a.cluster.AliveCount(),
	}
}

func (a *App) Nodes() []model.Node { return a.cluster.List() }

func (a *App) Whitelist() ([]string, error) { return a.store.ListWhitelist() }

func (a *App) ReplaceWhitelist(p []string) error { return a.store.ReplaceWhitelist(p) }

func (a *App) CreateTask(spec model.TaskSpec) (*model.Task, error) {
	patterns, err := a.store.ListWhitelist()
	if err != nil {
		return nil, err
	}
	if err := validate.CheckURL(spec.URL, patterns); err != nil {
		return nil, err
	}
	return a.store.CreateTask(spec)
}

func (a *App) ListTasks() ([]model.Task, error) { return a.store.ListTasks() }

func (a *App) GetTask(id string) (*model.Task, []model.Snapshot, error) {
	t, err := a.store.GetTask(id)
	if err != nil {
		return nil, nil, err
	}
	snaps, err := a.store.ListSnapshots(id)
	if err != nil {
		return nil, nil, err
	}
	return t, snaps, nil
}

func (a *App) StartTask(id string) (*model.Task, error) {
	running, err := a.store.RunningTask()
	if err != nil {
		return nil, err
	}
	if running != nil {
		return nil, ErrAlreadyRunning
	}
	t, err := a.store.GetTask(id)
	if err != nil {
		return nil, err
	}
	if t.Status != model.StatusDraft {
		return nil, ErrNotDraft
	}
	if err := a.store.MarkRunning(id); err != nil {
		return nil, err
	}
	if a.cluster.AliveCount() == 0 {
		return nil, ErrNoWorkers
	}
	patterns, err := a.store.ListWhitelist()
	if err != nil {
		return nil, err
	}
	if err := validate.CheckURL(t.URL, patterns); err != nil {
		return nil, err
	}
	t, err = a.store.GetTask(id)
	if err != nil {
		return nil, err
	}
	a.agg.Begin(t)
	n, err := a.cluster.DispatchStart(t)
	if err != nil {
		return nil, err
	}
	a.mu.Lock()
	a.runID = id
	a.mu.Unlock()
	a.log.Info("task started", "id", id, "workers", n)
	return t, nil
}

func (a *App) StopTask(id string) (*model.Task, error) {
	t, err := a.store.GetTask(id)
	if err != nil {
		return nil, err
	}
	if t.Status != model.StatusRunning {
		return t, nil
	}
	a.finish(id, model.StatusStopped)
	return a.store.GetTask(id)
}

func (a *App) ListReports() ([]model.Report, error) { return a.store.ListReports() }

func (a *App) GetReport(id string) (*model.Report, error) { return a.store.GetReport(id) }

func (a *App) HandleWS(w http.ResponseWriter, r *http.Request) { a.hub.Handle(w, r) }

func Listen(ctx context.Context, log *slog.Logger, httpAddr, grpcAddr string, a *App) error {
	httpSrv := api.New(log, a)
	hs := &http.Server{Addr: httpAddr, Handler: cors(httpSrv.Handler())}
	lis, err := net.Listen("tcp", grpcAddr)
	if err != nil {
		return err
	}
	gs := grpc.NewServer()
	proto.RegisterControlServer(gs, a.cluster)
	errCh := make(chan error, 2)
	go func() {
		log.Info("http listen", "addr", httpAddr)
		errCh <- hs.ListenAndServe()
	}()
	go func() {
		log.Info("grpc listen", "addr", grpcAddr)
		errCh <- gs.Serve(lis)
	}()
	select {
	case <-ctx.Done():
		shctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = hs.Shutdown(shctx)
		gs.GracefulStop()
		return nil
	case err := <-errCh:
		return err
	}
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(204)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func DefaultWhitelist() []string {
	raw := os.Getenv("GORHINO_DEFAULT_WHITELIST")
	if raw == "" {
		return []string{"target", "target:8088", "http://target:8088"}
	}
	return strings.Split(raw, ",")
}

func IsConflict(err error) bool {
	return errors.Is(err, store.ErrConflict)
}
