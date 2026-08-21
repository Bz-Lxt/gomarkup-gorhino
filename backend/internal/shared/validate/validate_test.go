package validate

import (
	"testing"

	"gorhino/internal/shared/model"
)

func TestTaskSpecOK(t *testing.T) {
	issues := TaskSpec(model.TaskSpec{
		Method: "get", URL: "http://target:8088/echo", VU: 10,
		DurationSec: 5, VersionTag: "v1",
	})
	if len(issues) != 0 {
		t.Fatalf("%v", issues)
	}
}

func TestTaskSpecBounds(t *testing.T) {
	issues := TaskSpec(model.TaskSpec{Method: "TRACE", URL: "", VU: 0, DurationSec: 0, VersionTag: ""})
	if len(issues) < 4 {
		t.Fatalf("expected multiple issues, got %v", issues)
	}
}

func TestMatchWhitelist(t *testing.T) {
	p := []string{"target", "target:8088", "http://target:8088/echo"}
	if !MatchWhitelist("http://target:8088/echo", p) {
		t.Fatal("prefix")
	}
	if !MatchWhitelist("http://target:8088/slow", p) {
		t.Fatal("host")
	}
	if MatchWhitelist("http://evil.example/x", p) {
		t.Fatal("should deny")
	}
}

func TestSSRFLinkLocal(t *testing.T) {
	err := CheckURL("http://169.254.169.254/latest/meta-data", []string{"169.254.169.254", "http://169.254.169.254/latest/meta-data"})
	if err == nil {
		t.Fatal("metadata must be blocked even if whitelisted")
	}
}

func TestSSRFUserinfo(t *testing.T) {
	err := CheckURL("http://a:b@target:8088/echo", []string{"target"})
	if err == nil {
		t.Fatal("userinfo")
	}
}

func TestPatterns(t *testing.T) {
	got, err := Patterns([]string{" target ", "target", ""})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("%v", got)
	}
	if _, err := Patterns(nil); err == nil {
		t.Fatal("empty")
	}
}
