package engine

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

func TestEngineHitsLocalServer(t *testing.T) {
	var n atomic.Int64
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n.Add(1)
		_, _ = io.WriteString(w, "ok")
	}))
	defer ts.Close()

	e := New()
	var ticks []Tick
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	err := e.Run(ctx, Spec{
		TaskID:   "t1",
		Method:   "GET",
		URL:      ts.URL,
		VU:       4,
		Duration: 1200 * time.Millisecond,
		QPS:      40,
	}, func(tick Tick) { ticks = append(ticks, tick) })
	if err != nil {
		t.Fatal(err)
	}
	if n.Load() < 10 {
		t.Fatalf("hits %d", n.Load())
	}
	var sum int64
	for _, tk := range ticks {
		sum += tk.Requests
	}
	if sum == 0 {
		t.Fatal("no ticks")
	}
}

func TestEngineTimeoutClassified(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(200)
	}))
	defer ts.Close()
	e := New()
	var timeouts int64
	_ = e.Run(context.Background(), Spec{
		TaskID:   "t2",
		Method:   "GET",
		URL:      ts.URL,
		VU:       1,
		Duration: 250 * time.Millisecond,
		Timeout:  50 * time.Millisecond,
	}, func(tick Tick) { timeouts += tick.Timeouts })
	if timeouts == 0 {
		t.Fatal("expected timeout classification")
	}
}
