package engine

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestEngineReportsWrappedTimeoutAsTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(75 * time.Millisecond)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	var requests, errors, timeouts, other int64
	err := New().Run(context.Background(), Spec{
		TaskID:   "slow-target",
		Method:   http.MethodGet,
		URL:      ts.URL,
		VU:       1,
		Duration: 90 * time.Millisecond,
		Timeout:  10 * time.Millisecond,
	}, func(tick Tick) {
		requests += tick.Requests
		errors += tick.Errors
		timeouts += tick.Timeouts
		other += tick.Other
	})
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if requests == 0 || errors != requests {
		t.Fatalf("requests=%d errors=%d, want every request to time out", requests, errors)
	}
	if timeouts != requests || other != 0 {
		t.Fatalf("requests=%d timeouts=%d other=%d, want all failures classified as timeouts", requests, timeouts, other)
	}
}
