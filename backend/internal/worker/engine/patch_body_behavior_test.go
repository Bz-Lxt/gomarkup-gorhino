package engine_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"gorhino/internal/worker/engine"
)

func TestEngineSendsPatchBody(t *testing.T) {
	const want = `{"state":"ready"}`
	received := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		select {
		case received <- string(body):
		default:
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	err := engine.New().Run(context.Background(), engine.Spec{
		TaskID:   "patch-body",
		Method:   http.MethodPatch,
		URL:      server.URL,
		Body:     []byte(want),
		VU:       1,
		Duration: 25 * time.Millisecond,
	}, nil)
	if err != nil {
		t.Fatalf("run PATCH task: %v", err)
	}

	select {
	case got := <-received:
		if got != want {
			t.Fatalf("target received body %q, want %q", got, want)
		}
	default:
		t.Fatal("target received no PATCH request")
	}
}
