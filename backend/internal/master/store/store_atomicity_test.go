package store_test

import (
	"reflect"
	"testing"

	"gorhino/internal/master/store"
)

func TestReplaceWhitelistErrorPreservesExistingPatterns(t *testing.T) {
	s, err := store.Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	original := []string{"api.internal", "worker.internal"}
	if err := s.SeedWhitelist(original); err != nil {
		t.Fatal(err)
	}
	if err := s.ReplaceWhitelist([]string{"new.internal", "new.internal"}); err == nil {
		t.Fatal("expected duplicate whitelist entry to reject the replacement")
	}

	got, err := s.ListWhitelist()
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, original) {
		t.Fatalf("failed replacement changed whitelist: got %v, want %v", got, original)
	}
}
