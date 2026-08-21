package api

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"gorhino/internal/master/store"
	"gorhino/internal/shared/model"
	"gorhino/internal/shared/validate"
)

type Service interface {
	Health() map[string]any
	Nodes() []model.Node
	Whitelist() ([]string, error)
	ReplaceWhitelist([]string) error
	CreateTask(model.TaskSpec) (*model.Task, error)
	ListTasks() ([]model.Task, error)
	GetTask(id string) (*model.Task, []model.Snapshot, error)
	StartTask(id string) (*model.Task, error)
	StopTask(id string) (*model.Task, error)
	ListReports() ([]model.Report, error)
	GetReport(id string) (*model.Report, error)
	HandleWS(http.ResponseWriter, *http.Request)
}

type Server struct {
	log *slog.Logger
	svc Service
}

func New(log *slog.Logger, svc Service) *Server {
	return &Server{log: log, svc: svc}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/v1/health", s.health)
	mux.HandleFunc("GET /api/v1/nodes", s.nodes)
	mux.HandleFunc("GET /api/v1/whitelist", s.getWL)
	mux.HandleFunc("PUT /api/v1/whitelist", s.putWL)
	mux.HandleFunc("POST /api/v1/tasks", s.createTask)
	mux.HandleFunc("GET /api/v1/tasks", s.listTasks)
	mux.HandleFunc("GET /api/v1/tasks/{id}", s.getTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/start", s.startTask)
	mux.HandleFunc("POST /api/v1/tasks/{id}/stop", s.stopTask)
	mux.HandleFunc("GET /api/v1/reports", s.listReports)
	mux.HandleFunc("GET /api/v1/reports/{id}", s.getReport)
	mux.HandleFunc("GET /api/v1/ws/live", s.svc.HandleWS)
	return withRecover(s.log, mux)
}

func withRecover(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic", "err", rec, "path", r.URL.Path)
				writeErr(w, http.StatusInternalServerError, "INTERNAL", "internal error")
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, s.svc.Health())
}

func (s *Server) nodes(w http.ResponseWriter, _ *http.Request) {
	writeOK(w, map[string]any{"nodes": s.svc.Nodes()})
}

func (s *Server) getWL(w http.ResponseWriter, _ *http.Request) {
	p, err := s.svc.Whitelist()
	if err != nil {
		writeErr(w, 500, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"patterns": p})
}

func (s *Server) putWL(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Patterns []string `json:"patterns"`
	}
	if err := decodeStrict(r, &body); err != nil {
		writeErr(w, 400, "VALIDATION_FAILED", err.Error())
		return
	}
	ps, err := validate.Patterns(body.Patterns)
	if err != nil {
		writeErr(w, 400, "VALIDATION_FAILED", err.Error())
		return
	}
	if err := s.svc.ReplaceWhitelist(ps); err != nil {
		writeErr(w, 500, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"patterns": ps})
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request) {
	var spec model.TaskSpec
	if err := decodeStrict(r, &spec); err != nil {
		writeErr(w, 400, "VALIDATION_FAILED", err.Error())
		return
	}
	spec = validate.NormalizeSpec(spec)
	if issues := validate.TaskSpec(spec); len(issues) > 0 {
		writeErr(w, 400, "VALIDATION_FAILED", validate.Join(issues).Error())
		return
	}
	t, err := s.svc.CreateTask(spec)
	if err != nil {
		mapErr(w, err)
		return
	}
	writeOKStatus(w, http.StatusCreated, t)
}

func (s *Server) listTasks(w http.ResponseWriter, _ *http.Request) {
	items, err := s.svc.ListTasks()
	if err != nil {
		writeErr(w, 500, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"items": items})
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request) {
	t, snaps, err := s.svc.GetTask(r.PathValue("id"))
	if err != nil {
		mapErr(w, err)
		return
	}
	writeOK(w, map[string]any{"task": t, "series": snaps})
}

func (s *Server) startTask(w http.ResponseWriter, r *http.Request) {
	t, err := s.svc.StartTask(r.PathValue("id"))
	if err != nil {
		mapErr(w, err)
		return
	}
	writeOK(w, t)
}

func (s *Server) stopTask(w http.ResponseWriter, r *http.Request) {
	t, err := s.svc.StopTask(r.PathValue("id"))
	if err != nil {
		mapErr(w, err)
		return
	}
	writeOK(w, t)
}

func (s *Server) listReports(w http.ResponseWriter, _ *http.Request) {
	items, err := s.svc.ListReports()
	if err != nil {
		writeErr(w, 500, "INTERNAL", err.Error())
		return
	}
	writeOK(w, map[string]any{"items": items})
}

func (s *Server) getReport(w http.ResponseWriter, r *http.Request) {
	rep, err := s.svc.GetReport(r.PathValue("id"))
	if err != nil {
		mapErr(w, err)
		return
	}
	writeOK(w, rep)
}

func decodeStrict(r *http.Request, dst any) error {
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

func mapErr(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		writeErr(w, 404, "TASK_NOT_FOUND", err.Error())
	case err == store.ErrConflict:
		if strings.Contains(err.Error(), "running") {
			writeErr(w, 409, "TASK_ALREADY_RUNNING", err.Error())
			return
		}
		if strings.Contains(err.Error(), "worker") {
			writeErr(w, 409, "NO_WORKERS", err.Error())
			return
		}
		writeErr(w, 409, "TASK_ALREADY_RUNNING", err.Error())
	default:
		msg := err.Error()
		if strings.Contains(msg, "whitelist") || strings.Contains(msg, "SSRF") || strings.Contains(msg, "validation") {
			writeErr(w, 400, "VALIDATION_FAILED", msg)
			return
		}
		if strings.Contains(msg, "no alive workers") {
			writeErr(w, 409, "NO_WORKERS", msg)
			return
		}
		if strings.Contains(msg, "already running") {
			writeErr(w, 409, "TASK_ALREADY_RUNNING", msg)
			return
		}
		writeErr(w, 500, "INTERNAL", msg)
	}
}

func writeOK(w http.ResponseWriter, data any) {
	writeOKStatus(w, 200, data)
}

func writeOKStatus(w http.ResponseWriter, code int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	b, _ := json.Marshal(map[string]any{"ok": true, "data": data})
	_, _ = w.Write(b)
}

func writeErr(w http.ResponseWriter, status int, code, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	b, _ := json.Marshal(map[string]any{"ok": false, "error": model.APIError{Code: code, Message: msg}})
	_, _ = w.Write(b)
}
