package engine

import (
	"context"
	"sync"
	"sync/atomic"
	"time"

	"github.com/valyala/fasthttp"
	"golang.org/x/time/rate"

	"gorhino/internal/shared/histogram"
	"gorhino/internal/shared/model"
)

type Spec struct {
	TaskID      string
	Method      string
	URL         string
	Headers     map[string]string
	Body        []byte
	VU          int
	Duration    time.Duration
	QPS         int
	Timeout     time.Duration
}

type Tick struct {
	TaskID    string
	Seq       int64
	Histogram []byte
	Requests  int64
	Errors    int64
	Timeouts  int64
	Other     int64
	SumLatUS  int64
	Codes     map[int32]int64
}

type Engine struct {
	client *fasthttp.Client
}

func New() *Engine {
	return &Engine{
		client: &fasthttp.Client{
			Name:                          "GoRhino",
			MaxConnsPerHost:               8192,
			MaxIdleConnDuration:           30 * time.Second,
			ReadTimeout:                   5 * time.Second,
			WriteTimeout:                  5 * time.Second,
			MaxConnWaitTimeout:            time.Second,
			DisableHeaderNamesNormalizing: false,
			NoDefaultUserAgentHeader:      false,
		},
	}
}

func (e *Engine) Run(ctx context.Context, spec Spec, emit func(Tick)) error {
	if spec.VU < 1 {
		return nil
	}
	if spec.Timeout <= 0 {
		spec.Timeout = 5 * time.Second
	}
	var (
		mu      sync.Mutex
		hist    = histogram.New()
		reqs    atomic.Int64
		errs    atomic.Int64
		timeouts atomic.Int64
		other   atomic.Int64
		sumUS   atomic.Int64
		codes   = map[int32]int64{}
		seq     int64
	)
	addCode := func(c int32) {
		mu.Lock()
		codes[c]++
		mu.Unlock()
	}

	var limiter *rate.Limiter
	if spec.QPS > 0 {
		limiter = rate.NewLimiter(rate.Limit(spec.QPS), max(1, spec.QPS/10))
	}

	runCtx, cancel := context.WithTimeout(ctx, spec.Duration)
	defer cancel()

	var wg sync.WaitGroup
	for i := 0; i < spec.VU; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				if runCtx.Err() != nil {
					return
				}
				if limiter != nil {
					if err := limiter.Wait(runCtx); err != nil {
						return
					}
				}
				status, lat, kind := e.do(spec)
				us := lat.Microseconds()
				if us < 1 {
					us = 1
				}
				hist.Record(us)
				sumUS.Add(us)
				reqs.Add(1)
				switch kind {
				case "ok":
					addCode(status)
					if status < 200 || status >= 300 {
						errs.Add(1)
					}
				case "timeout":
					timeouts.Add(1)
					errs.Add(1)
					addCode(-1)
				default:
					other.Add(1)
					errs.Add(1)
					addCode(-2)
				}
			}
		}()
	}

	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	flush := func() {
		hist.Reset()
		raw, err := hist.Encode()
		if err != nil {
			raw = nil
		}
		mu.Lock()
		cp := make(map[int32]int64, len(codes))
		for k, v := range codes {
			cp[k] = v
		}
		for k := range codes {
			delete(codes, k)
		}
		mu.Unlock()
		seq++
		tick := Tick{
			TaskID:    spec.TaskID,
			Seq:       seq,
			Histogram: raw,
			Requests:  reqs.Swap(0),
			Errors:    errs.Swap(0),
			Timeouts:  timeouts.Swap(0),
			Other:     other.Swap(0),
			SumLatUS:  sumUS.Swap(0),
			Codes:     cp,
		}
		if emit != nil {
			emit(tick)
		}
	}

	for {
		select {
		case <-done:
			flush()
			return nil
		case <-ticker.C:
			flush()
		}
	}
}

func (e *Engine) do(spec Spec) (int32, time.Duration, string) {
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI(spec.URL)
	req.Header.SetMethod(spec.Method)
	for k, v := range spec.Headers {
		req.Header.Set(k, v)
	}
	if len(spec.Body) > 0 && spec.Method != "GET" && spec.Method != "HEAD" {
		req.SetBody(spec.Body)
	}

	start := time.Now()
	err := e.client.DoTimeout(req, resp, spec.Timeout)
	lat := time.Since(start)
	if err != nil {
		if err == fasthttp.ErrTimeout || err == fasthttp.ErrDialTimeout {
			return 0, lat, "timeout"
		}
		return 0, lat, "other"
	}
	return int32(resp.StatusCode()), lat, "ok"
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func SpecFromTask(id string, method, url string, headers map[string]string, body []byte, vu, dur, qps int) Spec {
	_ = model.StatusDraft
	return Spec{
		TaskID:   id,
		Method:   method,
		URL:      url,
		Headers:  headers,
		Body:     body,
		VU:       vu,
		Duration: time.Duration(dur) * time.Second,
		QPS:      qps,
	}
}
