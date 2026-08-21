package engine_test

import (
	"context"
	"net/http"
	"testing"
	"time"

	"gorhino/internal/worker/engine"
)

func TestEngineDurationCancelsRateLimiterWaits(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- engine.New().Run(ctx, engine.Spec{
			TaskID:   "short-rate-limited-task",
			Method:   http.MethodGet,
			URL:      "http://127.0.0.1:1",
			VU:       4,
			Duration: 100 * time.Millisecond,
			QPS:      1,
			Timeout:  25 * time.Millisecond,
		}, nil)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("run returned error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		cancel()
		<-done
		t.Fatal("run did not stop when the configured duration elapsed")
	}
}
