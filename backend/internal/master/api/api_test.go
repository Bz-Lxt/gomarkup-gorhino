package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"gorhino/internal/logger"
	"gorhino/internal/master/store"
	"gorhino/internal/shared/model"
)

type fake struct {
	tasks []*model.Task
}

func (f *fake) Health() map[string]any { return map[string]any{"service": "master"} }
func (f *fake) Nodes() []model.Node    { return nil }
func (f *fake) Whitelist() ([]string, error) {
	return []string{"target"}, nil
}
func (f *fake) ReplaceWhitelist([]string) error { return nil }
func (f *fake) CreateTask(spec model.TaskSpec) (*model.Task, error) {
	t := &model.Task{ID: "tsk_1", Method: spec.Method, URL: spec.URL, VU: spec.VU, DurationSec: spec.DurationSec, VersionTag: spec.VersionTag, Status: model.StatusDraft, Headers: spec.Headers}
	f.tasks = append(f.tasks, t)
	return t, nil
}
func (f *fake) ListTasks() ([]model.Task, error) {
	out := make([]model.Task, 0, len(f.tasks))
	for _, t := range f.tasks {
		out = append(out, *t)
	}
	return out, nil
}
func (f *fake) GetTask(id string) (*model.Task, []model.Snapshot, error) {
	for _, t := range f.tasks {
		if t.ID == id {
			return t, nil, nil
		}
	}
	return nil, nil, store.ErrNotFound
}
func (f *fake) StartTask(id string) (*model.Task, error) {
	if id == "none" {
		return nil, errors.New("no alive workers")
	}
	return f.tasks[0], nil
}
func (f *fake) StopTask(id string) (*model.Task, error) {
	t, _, err := f.GetTask(id)
	return t, err
}
func (f *fake) ListReports() ([]model.Report, error)    { return nil, nil }
func (f *fake) GetReport(id string) (*model.Report, error) {
	return nil, store.ErrNotFound
}
func (f *fake) HandleWS(http.ResponseWriter, *http.Request) {}

func TestCreateAndGetTask(t *testing.T) {
	s := New(logger.New("error"), &fake{})
	body := []byte(`{"method":"GET","url":"http://target:8088/echo","vu":10,"duration_sec":5,"qps":0,"version_tag":"v1","headers":{"X":"1"}}`)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader(body))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 201 {
		t.Fatalf("code %d body %s", rec.Code, rec.Body.String())
	}
	var env map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil || env["ok"] != true {
		t.Fatalf("%s", rec.Body.String())
	}

	req = httptest.NewRequest(http.MethodGet, "http://x/api/v1/tasks/tsk_1", nil)
	rec = httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 200 {
		t.Fatalf("get %d %s", rec.Code, rec.Body.String())
	}
}

func TestValidationRejects(t *testing.T) {
	s := New(logger.New("error"), &fake{})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewReader([]byte(`{"method":"TRACE","url":"","vu":0,"duration_sec":0,"version_tag":""}`)))
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 400 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}

func TestNotFound(t *testing.T) {
	s := New(logger.New("error"), &fake{})
	req := httptest.NewRequest(http.MethodGet, "http://x/api/v1/reports/nope", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 404 {
		t.Fatalf("%d", rec.Code)
	}
}

func TestNoWorkersMapped(t *testing.T) {
	s := New(logger.New("error"), &fake{})
	req := httptest.NewRequest(http.MethodPost, "http://x/api/v1/tasks/none/start", nil)
	rec := httptest.NewRecorder()
	s.Handler().ServeHTTP(rec, req)
	if rec.Code != 409 {
		t.Fatalf("%d %s", rec.Code, rec.Body.String())
	}
}
