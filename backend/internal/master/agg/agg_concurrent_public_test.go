package agg_test

import (
	"sync"
	"testing"

	"gorhino/internal/master/agg"
	"gorhino/internal/shared/model"
)

func TestConcurrentIngestPreservesReportTotals(t *testing.T) {
	const (
		workers   = 32
		perWorker = 1000
	)

	a := agg.New()
	a.Begin(&model.Task{ID: "concurrent-report"})

	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			for j := 0; j < perWorker; j++ {
				a.Ingest(model.WorkerSnap{
					TaskID:   "concurrent-report",
					Requests: 1,
					Errors:   1,
					SumLatUS: 1000,
				})
			}
		}()
	}
	close(start)
	wg.Wait()

	report := a.Final(model.StatusCompleted)
	want := int64(workers * perWorker)
	if report.TotalRequests != want {
		t.Fatalf("total requests = %d, want %d", report.TotalRequests, want)
	}
	if report.TotalErrors != want {
		t.Fatalf("total errors = %d, want %d", report.TotalErrors, want)
	}
	if report.AvgMS != 1 {
		t.Fatalf("average latency = %v ms, want 1 ms", report.AvgMS)
	}
}
