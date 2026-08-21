package store

import (
	"testing"

	"gorhino/internal/shared/model"
)

func TestWhitelistReplace(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	if err := s.SeedWhitelist([]string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	if err := s.SeedWhitelist([]string{"c"}); err != nil {
		t.Fatal(err)
	}
	wl, _ := s.ListWhitelist()
	if len(wl) != 2 {
		t.Fatalf("seed is once-only, got %v", wl)
	}
	if err := s.ReplaceWhitelist([]string{"x", "y"}); err != nil {
		t.Fatal(err)
	}
	wl, _ = s.ListWhitelist()
	if len(wl) != 2 || wl[0] != "x" {
		t.Fatalf("%v", wl)
	}
}

func TestListTasksOrder(t *testing.T) {
	s, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	_, _ = s.CreateTask(model.TaskSpec{Method: "GET", URL: "http://a", VU: 1, DurationSec: 1, VersionTag: "a"})
	_, _ = s.CreateTask(model.TaskSpec{Method: "GET", URL: "http://b", VU: 1, DurationSec: 1, VersionTag: "b"})
	list, err := s.ListTasks()
	if err != nil || len(list) != 2 {
		t.Fatalf("%v %v", list, err)
	}
}

func TestNewIDUnique(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 64; i++ {
		id := NewID("tsk_")
		if seen[id] {
			t.Fatal("dup")
		}
		seen[id] = true
	}
}
