package store

import (
	"testing"

	"gorhino/internal/shared/model"
)

func TestTaskCRUD(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.SeedWhitelist([]string{"target", "http://target:8088/echo"}); err != nil {
		t.Fatal(err)
	}
	wl, err := s.ListWhitelist()
	if err != nil || len(wl) != 2 {
		t.Fatalf("wl %v %v", wl, err)
	}
	task, err := s.CreateTask(model.TaskSpec{
		Method: "GET", URL: "http://target:8088/echo", VU: 10, DurationSec: 5, VersionTag: "v1",
		Headers: map[string]string{"X-T": "1"},
	})
	if err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != task.URL || got.Status != model.StatusDraft || got.Headers["X-T"] != "1" {
		t.Fatalf("%+v", got)
	}
	if err := s.MarkRunning(task.ID); err != nil {
		t.Fatal(err)
	}
	run, err := s.RunningTask()
	if err != nil || run == nil || run.ID != task.ID {
		t.Fatalf("running %v %v", run, err)
	}
	if err := s.InsertSnapshot(model.Snapshot{
		TaskID: task.ID, TS: "2026-08-21 20:00:00", RPS: 100, P99MS: 2, Codes: map[string]int{"2xx": 100},
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.FinishTask(task.ID, model.StatusCompleted); err != nil {
		t.Fatal(err)
	}
	if err := s.UpsertReport(model.Report{Task: model.Task{ID: task.ID}, FinalRPS: 100, P99MS: 2, TotalRequests: 100, Codes: map[string]int{"2xx": 100}}); err != nil {
		t.Fatal(err)
	}
	rep, err := s.GetReport(task.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Series) != 1 || rep.FinalRPS != 100 {
		t.Fatalf("%+v", rep)
	}
	list, err := s.ListReports()
	if err != nil || len(list) != 1 {
		t.Fatalf("list %v %v", list, err)
	}
}

func TestMarkRunningConflict(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	task, err := s.CreateTask(model.TaskSpec{Method: "GET", URL: "http://x", VU: 1, DurationSec: 1, VersionTag: "a"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRunning(task.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.MarkRunning(task.ID); err == nil {
		t.Fatal("expected conflict")
	}
}

func TestListReportsNoNestedQuery(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	for i := 0; i < 3; i++ {
		task, err := s.CreateTask(model.TaskSpec{Method: "GET", URL: "http://x", VU: 1, DurationSec: 1, VersionTag: "v"})
		if err != nil {
			t.Fatal(err)
		}
		if err := s.MarkRunning(task.ID); err != nil {
			t.Fatal(err)
		}
		if err := s.FinishTask(task.ID, model.StatusCompleted); err != nil {
			t.Fatal(err)
		}
		if err := s.UpsertReport(model.Report{Task: model.Task{ID: task.ID}, FinalRPS: float64(i), TotalRequests: int64(i)}); err != nil {
			t.Fatal(err)
		}
	}
	list, err := s.ListReports()
	if err != nil || len(list) != 3 {
		t.Fatalf("%v %v", list, err)
	}
}

func TestMissingTask(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if _, err := s.GetTask("nope"); err != ErrNotFound {
		t.Fatalf("%v", err)
	}
}
