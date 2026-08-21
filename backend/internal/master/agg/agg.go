package agg

import (
	"sync"
	"time"

	"gorhino/internal/shared/histogram"
	"gorhino/internal/shared/model"
	"gorhino/internal/timeutil"
)

type Aggregator struct {
	mu     sync.Mutex
	task   *model.Task
	start  time.Time
	window *histogram.Window
	total  *histogram.Window
	req    int64
	err    int64
	sumUS  int64
	codes  map[string]int
	runReq int64
	runErr int64
	runSum int64
	runCod map[string]int
}

func New() *Aggregator {
	return &Aggregator{
		window: histogram.New(),
		total:  histogram.New(),
		codes:  map[string]int{},
		runCod: map[string]int{},
	}
}

func (a *Aggregator) Begin(task *model.Task) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.task = task
	a.start = timeutil.Now()
	a.window.Reset()
	a.total.Reset()
	a.req, a.err, a.sumUS = 0, 0, 0
	a.runReq, a.runErr, a.runSum = 0, 0, 0
	a.codes = map[string]int{}
	a.runCod = map[string]int{}
}

func (a *Aggregator) Ingest(s model.WorkerSnap) {
	a.mu.Lock()
	if a.task == nil || s.TaskID != a.task.ID {
		a.mu.Unlock()
		return
	}
	if len(s.Histogram) > 0 {
		_ = histogram.MergeEncoded(a.window, s.Histogram)
		_ = histogram.MergeEncoded(a.total, s.Histogram)
	}
	a.req += s.Requests
	a.err += s.Errors
	a.sumUS += s.SumLatUS
	a.runReq += s.Requests
	a.runErr += s.Errors
	a.runSum += s.SumLatUS
	for code, n := range s.StatusCode {
		key := bucket(code)
		a.codes[key] += int(n)
		a.runCod[key] += int(n)
	}
	a.mu.Unlock()
}

func bucket(code int32) string {
	switch {
	case code == -1:
		return "timeout"
	case code == -2:
		return "other"
	case code >= 200 && code < 300:
		return "2xx"
	case code >= 300 && code < 400:
		return "3xx"
	case code >= 400 && code < 500:
		return "4xx"
	case code >= 500 && code < 600:
		return "5xx"
	default:
		return "other"
	}
}

func (a *Aggregator) Flush(workers int) *model.Snapshot {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.task == nil {
		return nil
	}
	elapsed := int(timeutil.Now().Sub(a.start).Seconds())
	remain := a.task.DurationSec - elapsed
	if remain < 0 {
		remain = 0
	}
	var avg float64
	if a.req > 0 {
		avg = histogram.MicrosToMillis(a.sumUS / a.req)
	}
	var er float64
	if a.req > 0 {
		er = float64(a.err) / float64(a.req)
	}
	codes := map[string]int{}
	for k, v := range a.codes {
		codes[k] = v
	}
	sn := &model.Snapshot{
		TaskID:       a.task.ID,
		TS:           timeutil.NowString(),
		RPS:          float64(a.req),
		P50MS:        histogram.MicrosToMillis(a.window.ValueAt(50)),
		P95MS:        histogram.MicrosToMillis(a.window.ValueAt(95)),
		P99MS:        histogram.MicrosToMillis(a.window.ValueAt(99)),
		AvgMS:        avg,
		ErrorRate:    er,
		Codes:        codes,
		Workers:      workers,
		ElapsedSec:   elapsed,
		RemainingSec: remain,
		Status:       model.StatusRunning,
		TotalReq:     a.runReq,
		TotalErr:     a.runErr,
	}
	a.window.Reset()
	a.req, a.err, a.sumUS = 0, 0, 0
	a.codes = map[string]int{}
	return sn
}

func (a *Aggregator) Final(status string) *model.Report {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.task == nil {
		return nil
	}
	elapsed := timeutil.Now().Sub(a.start).Seconds()
	if elapsed < 1 {
		elapsed = 1
	}
	var avg float64
	if a.runReq > 0 {
		avg = histogram.MicrosToMillis(a.runSum / a.runReq)
	}
	var er float64
	if a.runReq > 0 {
		er = float64(a.runErr) / float64(a.runReq)
	}
	codes := map[string]int{}
	for k, v := range a.runCod {
		codes[k] = v
	}
	t := *a.task
	t.Status = status
	return &model.Report{
		Task:          t,
		FinalRPS:      float64(a.runReq) / elapsed,
		P50MS:         histogram.MicrosToMillis(a.total.ValueAt(50)),
		P95MS:         histogram.MicrosToMillis(a.total.ValueAt(95)),
		P99MS:         histogram.MicrosToMillis(a.total.ValueAt(99)),
		AvgMS:         avg,
		ErrorRate:     er,
		TotalRequests: a.runReq,
		TotalErrors:   a.runErr,
		Codes:         codes,
	}
}

func (a *Aggregator) CurrentTaskID() string {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.task == nil {
		return ""
	}
	return a.task.ID
}

func (a *Aggregator) Elapsed() time.Duration {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.task == nil {
		return 0
	}
	return timeutil.Now().Sub(a.start)
}
