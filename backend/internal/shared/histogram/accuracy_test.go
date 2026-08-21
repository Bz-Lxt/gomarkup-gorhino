package histogram

import (
	"math"
	"testing"
)

func TestThreeSigFigsMemoryBound(t *testing.T) {
	w := New()
	for i := 0; i < 200000; i++ {
		w.Record(int64((i%5000)+1) * 20)
	}
	if w.Count() != 200000 {
		t.Fatalf("count %d", w.Count())
	}
	raw, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	// Encoded snapshot must stay compact regardless of sample count.
	if len(raw) > 4096 {
		t.Fatalf("encoded size %d", len(raw))
	}
}

func TestP50P95P99Monotonic(t *testing.T) {
	w := New()
	for i := int64(1); i <= 10000; i++ {
		w.Record(i)
	}
	p50 := w.ValueAt(50)
	p95 := w.ValueAt(95)
	p99 := w.ValueAt(99)
	if !(p50 <= p95 && p95 <= p99) {
		t.Fatalf("%d %d %d", p50, p95, p99)
	}
	mean := w.Mean()
	if math.Abs(mean-5000) > 200 {
		t.Fatalf("mean %f", mean)
	}
}

func TestClampOutOfRange(t *testing.T) {
	w := New()
	w.Record(0)
	w.Record(90_000_000)
	if w.Count() != 2 {
		t.Fatal("clamp still records")
	}
}

func TestMergeEmptyBytes(t *testing.T) {
	w := New()
	if err := MergeEncoded(w, nil); err != nil {
		t.Fatal(err)
	}
}
