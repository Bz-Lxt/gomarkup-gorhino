package histogram

import (
	"math"
	"sort"
	"testing"
)

func exactPercentile(samples []int64, q float64) int64 {
	if len(samples) == 0 {
		return 0
	}
	cp := append([]int64(nil), samples...)
	sort.Slice(cp, func(i, j int) bool { return cp[i] < cp[j] })
	idx := int(math.Ceil(q/100.0*float64(len(cp)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(cp) {
		idx = len(cp) - 1
	}
	return cp[idx]
}

func TestRecordAndPercentile(t *testing.T) {
	w := New()
	var samples []int64
	for i := int64(1); i <= 1000; i++ {
		us := i * 100
		w.Record(us)
		samples = append(samples, us)
	}
	got := w.ValueAt(99)
	want := exactPercentile(samples, 99)
	rel := math.Abs(float64(got-want)) / float64(want)
	if rel > 0.01 {
		t.Fatalf("p99 error %.3f got=%d want=%d", rel, got, want)
	}
}

func TestMergeAcrossWorkers(t *testing.T) {
	a := New()
	b := New()
	for i := int64(1); i <= 500; i++ {
		a.Record(i * 10)
		b.Record(10000 + i*10)
	}
	raw, err := b.Encode()
	if err != nil {
		t.Fatal(err)
	}
	if err := MergeEncoded(a, raw); err != nil {
		t.Fatal(err)
	}
	if a.Count() != 1000 {
		t.Fatalf("count %d", a.Count())
	}
	if a.ValueAt(50) == 0 {
		t.Fatal("p50")
	}
}

func TestResetKeepsCapacity(t *testing.T) {
	w := New()
	w.Record(42)
	w.Reset()
	if w.Count() != 0 {
		t.Fatal("reset")
	}
	w.Record(100)
	if w.Count() != 1 {
		t.Fatal("reuse")
	}
}

func TestEmptyEncodeDecode(t *testing.T) {
	w := New()
	b, err := w.Encode()
	if err != nil {
		t.Fatal(err)
	}
	d, err := Decode(b)
	if err != nil {
		t.Fatal(err)
	}
	if d.Count() != 0 {
		t.Fatal("empty")
	}
}
