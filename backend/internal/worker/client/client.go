package client

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"os"
	"runtime"
	"sync"
	"time"

	"gorhino/internal/proto"
	"gorhino/internal/worker/engine"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Worker struct {
	log    *slog.Logger
	addr   string
	id     string
	eng    *engine.Engine
	mu     sync.Mutex
	cancel context.CancelFunc
}

func New(log *slog.Logger, addr string) *Worker {
	return &Worker{log: log, addr: addr, id: nodeID(), eng: engine.New()}
}

func nodeID() string {
	if v := os.Getenv("GORHINO_NODE_ID"); v != "" {
		return v
	}
	host, _ := os.Hostname()
	var b [4]byte
	_, _ = rand.Read(b[:])
	return host + "-" + hex.EncodeToString(b[:])
}

func (w *Worker) Run(ctx context.Context) error {
	backoff := time.Second
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		err := w.session(ctx)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		w.log.Warn("master stream closed, retry", "err", err, "backoff", backoff.String())
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(backoff):
		}
		if backoff < 15*time.Second {
			backoff *= 2
		}
	}
}

func (w *Worker) session(ctx context.Context) error {
	conn, err := grpc.NewClient(w.addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return err
	}
	defer conn.Close()
	cli := proto.NewControlClient(conn)
	stream, err := cli.Connect(ctx)
	if err != nil {
		return err
	}
	host, _ := os.Hostname()
	if err := stream.Send(&proto.WorkerMsg{Body: &proto.WorkerMsg_Register{Register: &proto.Register{
		NodeId:   w.id,
		Hostname: host,
		CpuCount: int32(runtime.NumCPU()),
	}}}); err != nil {
		return err
	}

	sess, cancel := context.WithCancel(ctx)
	defer cancel()

	go w.heartbeat(sess, stream)

	for {
		msg, err := stream.Recv()
		if err != nil {
			w.stopTask()
			return err
		}
		switch b := msg.Body.(type) {
		case *proto.MasterMsg_Welcome:
			w.log.Info("joined master", "master", b.Welcome.GetMasterId(), "id", w.id)
			backoffReset()
		case *proto.MasterMsg_Start:
			w.startTask(sess, stream, b.Start)
		case *proto.MasterMsg_Stop:
			w.stopTask()
			_ = stream.Send(&proto.WorkerMsg{Body: &proto.WorkerMsg_Ack{Ack: &proto.TaskAck{
				TaskId: b.Stop.GetTaskId(), Action: "stop",
			}}})
		}
	}
}

func backoffReset() {}

func (w *Worker) heartbeat(ctx context.Context, stream proto.Control_ConnectClient) {
	t := time.NewTicker(3 * time.Second)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			_ = stream.Send(&proto.WorkerMsg{Body: &proto.WorkerMsg_Heartbeat{Heartbeat: &proto.Heartbeat{
				NodeId: w.id, UnixMs: time.Now().UnixMilli(),
			}}})
		}
	}
}

func (w *Worker) startTask(ctx context.Context, stream proto.Control_ConnectClient, st *proto.StartTask) {
	w.stopTask()
	if st.GetVu() <= 0 {
		w.log.Info("assigned zero VU, idle", "task", st.GetTaskId())
		return
	}
	tctx, cancel := context.WithCancel(ctx)
	w.mu.Lock()
	w.cancel = cancel
	w.mu.Unlock()
	spec := engine.Spec{
		TaskID:   st.GetTaskId(),
		Method:   st.GetMethod(),
		URL:      st.GetUrl(),
		Headers:  st.GetHeaders(),
		Body:     st.GetBody(),
		VU:       int(st.GetVu()),
		Duration: time.Duration(st.GetDurationSec()) * time.Second,
		QPS:      int(st.GetQps()),
	}
	w.log.Info("start fire", "task", spec.TaskID, "vu", spec.VU, "qps", spec.QPS, "url", spec.URL)
	go func() {
		_ = w.eng.Run(tctx, spec, func(tick engine.Tick) {
			_ = stream.Send(&proto.WorkerMsg{Body: &proto.WorkerMsg_Stats{Stats: &proto.StatsReport{
				TaskId:        tick.TaskID,
				Seq:           tick.Seq,
				Histogram:     tick.Histogram,
				Requests:      tick.Requests,
				Errors:        tick.Errors,
				TimeoutCount:  tick.Timeouts,
				OtherCount:    tick.Other,
				SumLatencyUs:  tick.SumLatUS,
				StatusCodes:   tick.Codes,
			}}})
		})
		w.log.Info("engine finished", "task", spec.TaskID)
	}()
}

func (w *Worker) stopTask() {
	w.mu.Lock()
	if w.cancel != nil {
		w.cancel()
		w.cancel = nil
	}
	w.mu.Unlock()
}
