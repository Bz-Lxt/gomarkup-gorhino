package config

import (
	"testing"
	"time"
)

func TestSplitCSV(t *testing.T) {
	got := splitCSV(" a, b,,c ")
	if len(got) != 3 || got[0] != "a" || got[2] != "c" {
		t.Fatalf("%v", got)
	}
}

func TestDurationEnv(t *testing.T) {
	t.Setenv("GORHINO_REAP_AFTER", "15s")
	if durationEnv("GORHINO_REAP_AFTER", time.Second) != 15*time.Second {
		t.Fatal("parse duration")
	}
	t.Setenv("GORHINO_REAP_AFTER", "12")
	if durationEnv("GORHINO_REAP_AFTER", time.Second) != 12*time.Second {
		t.Fatal("int seconds")
	}
	t.Setenv("GORHINO_REAP_AFTER", "nope")
	if durationEnv("GORHINO_REAP_AFTER", 3*time.Second) != 3*time.Second {
		t.Fatal("fallback")
	}
}

func TestMasterFromEnv(t *testing.T) {
	t.Setenv("GORHINO_HTTP_ADDR", ":18080")
	t.Setenv("GORHINO_DEFAULT_WHITELIST", "target,http://target:8088/echo")
	m := MasterFromEnv()
	if m.HTTPAddr != ":18080" || len(m.Whitelist) != 2 {
		t.Fatalf("%+v", m)
	}
}

func TestIntEnv(t *testing.T) {
	t.Setenv("X", "9")
	if IntEnv("X", 1) != 9 {
		t.Fatal("int")
	}
	if IntEnv("MISSING", 4) != 4 {
		t.Fatal("def")
	}
}
