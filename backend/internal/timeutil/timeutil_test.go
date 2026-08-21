package timeutil

import (
	"strings"
	"testing"
	"time"
)

func TestFormatRoundTrip(t *testing.T) {
	raw := "2026-08-21 20:15:00"
	got, err := Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if Format(got) != raw {
		t.Fatalf("round trip %s != %s", Format(got), raw)
	}
	if got.Location().String() != "CST" {
		t.Fatalf("tz %s", got.Location())
	}
}

func TestNowStringShape(t *testing.T) {
	s := NowString()
	if _, err := Parse(s); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(s, "T") || strings.Contains(s, "Z") {
		t.Fatalf("user-facing time must be Beijing layout, got %s", s)
	}
}

func TestFormatZero(t *testing.T) {
	if Format(time.Time{}) != "" {
		t.Fatal("zero should be empty")
	}
}
