package target

import (
	"encoding/json"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"strconv"
	"sync/atomic"
	"time"
)

type Server struct {
	log   *slog.Logger
	hits  atomic.Int64
	slow  atomic.Int64
	errN  atomic.Int64
	echo  atomic.Int64
	start time.Time
}

func New(log *slog.Logger) *Server {
	return &Server{log: log, start: time.Now()}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("/echo", s.echoH)
	mux.HandleFunc("/slow", s.slowH)
	mux.HandleFunc("/error", s.errorH)
	mux.HandleFunc("GET /stats", s.stats)
	return mux
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"service":"target"}`))
}

func (s *Server) echoH(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	s.echo.Add(1)
	body, _ := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	out := map[string]any{
		"method": r.Method,
		"path":   r.URL.Path,
		"query":  r.URL.RawQuery,
		"ua":     r.Header.Get("User-Agent"),
		"bytes":  len(body),
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}

func (s *Server) slowH(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	s.slow.Add(1)
	ms, _ := strconv.Atoi(r.URL.Query().Get("ms"))
	if ms < 0 {
		ms = 0
	}
	if ms > 5000 {
		ms = 5000
	}
	time.Sleep(time.Duration(ms) * time.Millisecond)
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true,"slept_ms":` + strconv.Itoa(ms) + `}`))
}

func (s *Server) errorH(w http.ResponseWriter, r *http.Request) {
	s.hits.Add(1)
	rate, _ := strconv.ParseFloat(r.URL.Query().Get("rate"), 64)
	if rate < 0 {
		rate = 0
	}
	if rate > 1 {
		rate = 1
	}
	if rand.Float64() < rate {
		s.errN.Add(1)
		http.Error(w, `{"ok":false}`, http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write([]byte(`{"ok":true}`))
}

func (s *Server) stats(w http.ResponseWriter, _ *http.Request) {
	out := map[string]any{
		"hits":    s.hits.Load(),
		"echo":    s.echo.Load(),
		"slow":    s.slow.Load(),
		"errors":  s.errN.Load(),
		"uptime":  time.Since(s.start).Seconds(),
		"service": "target",
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(out)
}
