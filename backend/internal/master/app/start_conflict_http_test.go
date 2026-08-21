package app_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorhino/internal/logger"
	"gorhino/internal/master/api"
	"gorhino/internal/master/app"
	"gorhino/internal/master/store"
	"gorhino/internal/shared/model"
)

func TestStartCompletedTaskRemainsConflict(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	task, err := st.CreateTask(model.TaskSpec{
		Method: "GET", URL: "http://target:8088/echo", VU: 1, DurationSec: 1, VersionTag: "v1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.FinishTask(task.ID, model.StatusCompleted); err != nil {
		t.Fatal(err)
	}

	handler := api.New(logger.New("error"), app.New(logger.New("error"), st)).Handler()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks/"+task.ID+"/start", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusConflict {
		t.Fatalf("start completed task: status = %d, body = %s; want %d", rec.Code, rec.Body.String(), http.StatusConflict)
	}
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.OK {
		t.Fatalf("start completed task unexpectedly succeeded: %s", rec.Body.String())
	}
}
