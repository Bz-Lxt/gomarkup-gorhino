package app

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorhino/internal/logger"
	"gorhino/internal/master/api"
	"gorhino/internal/master/store"
	"gorhino/internal/shared/model"
)

func TestRejectedStartWithoutWorkersKeepsTaskRetryable(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SeedWhitelist([]string{"127.0.0.1"}); err != nil {
		t.Fatal(err)
	}
	task, err := st.CreateTask(model.TaskSpec{
		Method: "GET", URL: "http://127.0.0.1:8088/echo", VU: 1, DurationSec: 10, VersionTag: "retryable",
	})
	if err != nil {
		t.Fatal(err)
	}

	h := api.New(logger.New("error"), New(logger.New("error"), st)).Handler()
	start := func() (int, string) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/start", nil)
		h.ServeHTTP(rec, req)
		var env model.Envelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("decode start response: %v; body=%s", err, rec.Body.String())
		}
		if env.Error == nil {
			return rec.Code, ""
		}
		return rec.Code, env.Error.Code
	}

	if status, code := start(); status != http.StatusConflict || code != "NO_WORKERS" {
		t.Fatalf("first start = HTTP %d %s, want HTTP 409 NO_WORKERS", status, code)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+task.ID, nil)
	h.ServeHTTP(rec, req)
	var got struct {
		OK   bool `json:"ok"`
		Data struct {
			Task model.Task `json:"task"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode task response: %v; body=%s", err, rec.Body.String())
	}
	if rec.Code != http.StatusOK || !got.OK {
		t.Fatalf("get task = HTTP %d body=%s", rec.Code, rec.Body.String())
	}
	if got.Data.Task.Status != model.StatusDraft {
		t.Fatalf("task status after rejected start = %q, want %q", got.Data.Task.Status, model.StatusDraft)
	}

	if status, code := start(); status != http.StatusConflict || code != "NO_WORKERS" {
		t.Fatalf("retry start = HTTP %d %s, want HTTP 409 NO_WORKERS", status, code)
	}
}
