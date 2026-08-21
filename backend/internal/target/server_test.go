package target

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gorhino/internal/logger"
)

func TestTargetRoutes(t *testing.T) {
	s := New(logger.New("error"))
	ts := httptest.NewServer(s.Handler())
	defer ts.Close()

	res, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	b, _ := io.ReadAll(res.Body)
	_ = res.Body.Close()
	if res.StatusCode != 200 || !strings.Contains(string(b), "target") {
		t.Fatalf("%d %s", res.StatusCode, b)
	}

	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/echo", strings.NewReader("hi"))
	req.Header.Set("User-Agent", "ut")
	res, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(b), `"method":"POST"`) {
		t.Fatalf("%s", b)
	}

	res, err = http.Get(ts.URL + "/slow?ms=5")
	if err != nil || res.StatusCode != 200 {
		t.Fatalf("slow %v %v", err, res)
	}
	_ = res.Body.Close()

	res, err = http.Get(ts.URL + "/error?rate=1")
	if err != nil {
		t.Fatal(err)
	}
	if res.StatusCode != 500 {
		t.Fatalf("error %d", res.StatusCode)
	}
	_ = res.Body.Close()

	res, err = http.Get(ts.URL + "/stats")
	if err != nil {
		t.Fatal(err)
	}
	b, _ = io.ReadAll(res.Body)
	_ = res.Body.Close()
	if !strings.Contains(string(b), `"hits"`) {
		t.Fatalf("%s", b)
	}
}
