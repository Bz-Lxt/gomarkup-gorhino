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

func TestFailedTaskReportRemainsAvailableWithoutStats(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()

	task, err := st.CreateTask(model.TaskSpec{
		Method: "GET", URL: "http://target/echo", Headers: map[string]string{},
		VU: 1, DurationSec: 30, VersionTag: "restart-case",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.MarkRunning(task.ID); err != nil {
		t.Fatal(err)
	}
	if err := st.FailStaleRunning(); err != nil {
		t.Fatal(err)
	}

	srv := api.New(logger.New("error"), app.New(logger.New("error"), st))
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/"+task.ID, nil)
	srv.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}
	var envelope struct {
		OK   bool `json:"ok"`
		Data *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &envelope); err != nil {
		t.Fatal(err)
	}
	if !envelope.OK || envelope.Data == nil {
		t.Fatalf("failed task report is missing: %s", rec.Body.String())
	}
	if envelope.Data.ID != task.ID || envelope.Data.Status != model.StatusFailed {
		t.Fatalf("unexpected report metadata: %+v", envelope.Data)
	}
}
