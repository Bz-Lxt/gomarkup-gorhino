package engine_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorhino/internal/worker/engine"
)

func TestEngineCountsRedirectResponsesAsErrors(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusFound)
	}))
	defer ts.Close()

	var requests, errors, redirects int64
	err := engine.New().Run(context.Background(), engine.Spec{
		TaskID:   "redirects",
		Method:   http.MethodGet,
		URL:      ts.URL,
		VU:       1,
		Duration: 40 * time.Millisecond,
	}, func(tick engine.Tick) {
		requests += tick.Requests
		errors += tick.Errors
		redirects += tick.Codes[http.StatusFound]
	})
	if err != nil {
		t.Fatalf("run load: %v", err)
	}
	if requests == 0 {
		t.Fatal("expected at least one request")
	}
	if redirects != requests {
		t.Fatalf("302 responses = %d, requests = %d", redirects, requests)
	}
	if errors != requests {
		t.Fatalf("errors = %d, want %d for non-2xx responses", errors, requests)
	}
}
