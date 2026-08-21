package shard

import "testing"

func TestManyWorkers(t *testing.T) {
	got := VUs(1000, 7)
	if len(got) != 7 || Sum(got) != 1000 {
		t.Fatalf("%v", got)
	}
	// remainder goes to the first rem workers
	if got[0] < got[len(got)-1] {
		t.Fatalf("remainder order %v", got)
	}
}

func TestSingleWorkerGetsAll(t *testing.T) {
	got := VUs(333, 1)
	if len(got) != 1 || got[0] != 333 {
		t.Fatalf("%v", got)
	}
}

func TestMoreWorkersThanVUs(t *testing.T) {
	got := VUs(2, 5)
	if Sum(got) != 2 {
		t.Fatalf("%v", got)
	}
	ones := 0
	for _, v := range got {
		if v == 1 {
			ones++
		}
		if v > 1 {
			t.Fatalf("overflow %v", got)
		}
	}
	if ones != 2 {
		t.Fatalf("ones %d %v", ones, got)
	}
}

func TestQPSMatchesVUShape(t *testing.T) {
	vu := VUs(17, 4)
	q := QPS(17, 4)
	if len(vu) != len(q) {
		t.Fatal("len")
	}
	for i := range vu {
		if vu[i] != q[i] {
			t.Fatalf("%v vs %v", vu, q)
		}
	}
}

func TestNegativeInputs(t *testing.T) {
	if VUs(-1, 2) != nil || VUs(5, -2) != nil {
		t.Fatal("neg")
	}
	if Sum(nil) != 0 {
		t.Fatal("sum nil")
	}
}
