package agg

import (
	"testing"
	"time"

	"gorhino/internal/shared/histogram"
	"gorhino/internal/shared/model"
)

func TestFlushAndFinal(t *testing.T) {
	a := New()
	task := &model.Task{ID: "tsk_1", DurationSec: 10, VU: 4, URL: "http://target:8088/echo", Method: "GET", Status: model.StatusRunning}
	a.Begin(task)
	w := histogram.New()
	for i := int64(1); i <= 100; i++ {
		w.Record(i * 100)
	}
	raw, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	a.Ingest(model.WorkerSnap{
		TaskID: "tsk_1", Requests: 90, Errors: 10, SumLatUS: 90000,
		Histogram: raw, StatusCode: map[int32]int64{200: 80, 500: 10, -1: 5, -2: 5},
	})
	sn := a.Flush(2)
	if sn == nil {
		t.Fatal("snapshot")
	}
	if sn.RPS != 90 || sn.Workers != 2 {
		t.Fatalf("%+v", sn)
	}
	if sn.Codes["2xx"] != 80 || sn.Codes["5xx"] != 10 || sn.Codes["timeout"] != 5 {
		t.Fatalf("codes %+v", sn.Codes)
	}
	if sn.ErrorRate < 0.1-1e-9 {
		t.Fatalf("err rate %f", sn.ErrorRate)
	}
	if sn.P99MS <= 0 {
		t.Fatal("p99")
	}
	rep := a.Final(model.StatusCompleted)
	if rep == nil || rep.TotalRequests != 90 || rep.TotalErrors != 10 {
		t.Fatalf("%+v", rep)
	}
}

func TestIgnoreForeignTask(t *testing.T) {
	a := New()
	a.Begin(&model.Task{ID: "a", DurationSec: 3})
	a.Ingest(model.WorkerSnap{TaskID: "b", Requests: 99})
	sn := a.Flush(1)
	if sn.RPS != 0 {
		t.Fatalf("leaked %+v", sn)
	}
}

func TestBucket(t *testing.T) {
	cases := map[int32]string{200: "2xx", 301: "3xx", 404: "4xx", 503: "5xx", -1: "timeout", -2: "other", 0: "other"}
	for c, want := range cases {
		if got := bucket(c); got != want {
			t.Fatalf("%d -> %s want %s", c, got, want)
		}
	}
}

func TestElapsed(t *testing.T) {
	a := New()
	if a.CurrentTaskID() != "" {
		t.Fatal("empty")
	}
	a.Begin(&model.Task{ID: "z", DurationSec: 2})
	time.Sleep(20 * time.Millisecond)
	if a.CurrentTaskID() != "z" || a.Elapsed() <= 0 {
		t.Fatal("elapsed")
	}
}
