package validate

import (
	"net"
	"testing"

	"gorhino/internal/shared/model"
)

func TestBlockedIPTable(t *testing.T) {
	cases := []struct {
		ip   string
		want bool
	}{
		{"169.254.169.254", true},
		{"169.254.0.1", true},
		{"8.8.8.8", false},
		{"10.0.0.5", false},
		{"172.16.0.2", false},
		{"192.168.1.1", false},
		{"127.0.0.1", false},
		{"0.0.0.0", true},
		{"224.0.0.1", true},
	}
	for _, c := range cases {
		if got := blockedIP(net.ParseIP(c.ip)); got != c.want {
			t.Fatalf("%s blocked=%v want %v", c.ip, got, c.want)
		}
	}
}

func TestRejectNonHTTP(t *testing.T) {
	if err := CheckURL("ftp://target/x", []string{"target"}); err == nil {
		t.Fatal("ftp")
	}
	if err := CheckURL("http:///nohost", []string{"target"}); err == nil {
		t.Fatal("no host")
	}
}

func TestHeaderAndBodyLimits(t *testing.T) {
	h := map[string]string{}
	for i := 0; i < model.MaxHeaderPairs+1; i++ {
		h[string(rune('a'+i%26))+string(rune('0'+i%10))+string(rune('A'+i%26))] = "v"
	}
	issues := TaskSpec(model.TaskSpec{
		Method: "GET", URL: "http://t", VU: 1, DurationSec: 1, VersionTag: "v", Headers: h,
	})
	found := false
	for _, is := range issues {
		if is.Field == "headers" {
			found = true
		}
	}
	if !found {
		t.Fatalf("%v", issues)
	}
	big := make([]byte, model.MaxBodyBytes+1)
	issues = TaskSpec(model.TaskSpec{
		Method: "POST", URL: "http://t", VU: 1, DurationSec: 1, VersionTag: "v", Body: string(big),
	})
	found = false
	for _, is := range issues {
		if is.Field == "body" {
			found = true
		}
	}
	if !found {
		t.Fatal("body limit")
	}
}

func TestNormalize(t *testing.T) {
	s := NormalizeSpec(model.TaskSpec{Method: " post ", URL: " http://x ", VersionTag: " t "})
	if s.Method != "POST" || s.URL != "http://x" || s.VersionTag != "t" || s.Headers == nil {
		t.Fatalf("%+v", s)
	}
}
