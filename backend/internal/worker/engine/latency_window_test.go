package engine_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gorhino/internal/shared/histogram"
	"gorhino/internal/worker/engine"
)

func TestRunEmitsLatencySamplesWithRequests(t *testing.T) {
	var ticks []engine.Tick
	err := engine.New().Run(context.Background(), engine.Spec{
		TaskID:   "latency-window",
		Method:   http.MethodGet,
		URL:      "://invalid-target",
		VU:       2,
		Duration: 50 * time.Millisecond,
	}, func(tick engine.Tick) {
		ticks = append(ticks, tick)
	})
	if err != nil {
		t.Fatal(err)
	}

	var requests, samples int64
	for _, tick := range ticks {
		requests += tick.Requests
		window, err := histogram.Decode(tick.Histogram)
		if err != nil {
			t.Fatalf("decode latency histogram: %v", err)
		}
		samples += window.Count()
	}
	if requests == 0 {
		t.Fatal("engine attempted no requests")
	}
	if samples != requests {
		t.Fatalf("latency sample count = %d, request count = %d", samples, requests)
	}
}
