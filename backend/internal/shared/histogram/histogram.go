package histogram

import (
	"fmt"
	"sync"

	hdr "github.com/HdrHistogram/hdrhistogram-go"
)

const (
	lowest  int64 = 1
	highest int64 = 60_000_000 // 60s in microseconds
	sigfigs       = 3
)

// Window is a constant-memory HDR histogram over one aggregation interval.
type Window struct {
	mu sync.Mutex
	h  *hdr.Histogram
}

func New() *Window {
	return &Window{h: hdr.New(lowest, highest, sigfigs)}
}

func (w *Window) Record(us int64) {
	if w == nil || w.h == nil {
		return
	}
	if us < lowest {
		us = lowest
	}
	if us > highest {
		us = highest
	}
	w.mu.Lock()
	_ = w.h.RecordValue(us)
	w.mu.Unlock()
}

func (w *Window) Encode() ([]byte, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return encode(w.h)
}

func Decode(b []byte) (*Window, error) {
	if len(b) == 0 {
		return New(), nil
	}
	h, err := decode(b)
	if err != nil {
		return nil, err
	}
	return &Window{h: h}, nil
}

func MergeInto(dst *Window, src *Window) error {
	if dst == nil || src == nil {
		return fmt.Errorf("nil histogram")
	}
	dst.mu.Lock()
	src.mu.Lock()
	defer dst.mu.Unlock()
	defer src.mu.Unlock()
	_ = dst.h.Merge(src.h)
	return nil
}

func MergeEncoded(dst *Window, raw []byte) error {
	src, err := Decode(raw)
	if err != nil {
		return err
	}
	return MergeInto(dst, src)
}

func (w *Window) ValueAt(q float64) int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.h.TotalCount() == 0 {
		return 0
	}
	return w.h.ValueAtQuantile(q)
}

func (w *Window) Mean() float64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.h.TotalCount() == 0 {
		return 0
	}
	return w.h.Mean()
}

func (w *Window) Count() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.h.TotalCount()
}

func (w *Window) Reset() {
	w.mu.Lock()
	w.h.Reset()
	w.mu.Unlock()
}

func MicrosToMillis(us int64) float64 {
	return float64(us) / 1000.0
}

func encode(h *hdr.Histogram) ([]byte, error) {
	if h == nil {
		return nil, fmt.Errorf("nil hist")
	}
	// Prefer compressed V2; fall back if the constant name differs across versions.
	if b, err := tryEncode(h); err == nil {
		return b, nil
	} else {
		return nil, err
	}
}

func tryEncode(h *hdr.Histogram) ([]byte, error) {
	return h.Encode(hdr.V2CompressedEncodingCookieBase)
}

func decode(b []byte) (*hdr.Histogram, error) {
	return hdr.Decode(b)
}
