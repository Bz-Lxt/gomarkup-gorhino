package logger

import (
	"testing"
)

func TestEnvLevelDefault(t *testing.T) {
	t.Setenv("GORHINO_LOG_LEVEL", "")
	if EnvLevel() != "info" {
		t.Fatal(EnvLevel())
	}
	t.Setenv("GORHINO_LOG_LEVEL", "debug")
	if EnvLevel() != "debug" {
		t.Fatal(EnvLevel())
	}
}

func TestNewDoesNotPanic(t *testing.T) {
	for _, lv := range []string{"debug", "info", "warn", "warning", "error", "weird"} {
		if New(lv) == nil {
			t.Fatalf("%s", lv)
		}
	}
}
