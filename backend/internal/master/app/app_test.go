package app

import (
	"testing"

	"gorhino/internal/logger"
	"gorhino/internal/master/store"
	"gorhino/internal/shared/model"
	"gorhino/internal/shared/validate"
)

func TestCreateRejectsOffWhitelist(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SeedWhitelist([]string{"target", "http://target:8088/echo"}); err != nil {
		t.Fatal(err)
	}
	a := New(logger.New("error"), st)
	_, err = a.CreateTask(model.TaskSpec{
		Method: "GET", URL: "http://example.com/", VU: 1, DurationSec: 1, VersionTag: "x",
	})
	if err == nil {
		t.Fatal("expected whitelist fail")
	}
}

func TestStartWithoutWorkers(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SeedWhitelist([]string{"target"}); err != nil {
		t.Fatal(err)
	}
	a := New(logger.New("error"), st)
	// URL check will DNS target — skip create if offline by injecting already stored task
	task, err := st.CreateTask(validate.NormalizeSpec(model.TaskSpec{
		Method: "GET", URL: "http://target:8088/echo", VU: 2, DurationSec: 2, VersionTag: "x",
	}))
	if err != nil {
		t.Fatal(err)
	}
	_, err = a.StartTask(task.ID)
	if err != ErrNoWorkers {
		t.Fatalf("got %v", err)
	}
}

func TestHealthAndReportsEmpty(t *testing.T) {
	st, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	a := New(logger.New("error"), st)
	h := a.Health()
	if h["service"] != "master" {
		t.Fatalf("%v", h)
	}
	items, err := a.ListReports()
	if err != nil || len(items) != 0 {
		t.Fatalf("%v %v", items, err)
	}
}

func TestConflictHelpers(t *testing.T) {
	if !IsConflict(ErrAlreadyRunning) || !IsConflict(ErrNoWorkers) {
		t.Fatal("conflict wrap")
	}
}
