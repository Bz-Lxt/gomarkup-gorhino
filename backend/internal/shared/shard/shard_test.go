package shard

import "testing"

func TestVUsEven(t *testing.T) {
	got := VUs(10, 2)
	if len(got) != 2 || got[0] != 5 || got[1] != 5 {
		t.Fatalf("%v", got)
	}
}

func TestVUsRemainder(t *testing.T) {
	got := VUs(11, 3)
	if Sum(got) != 11 {
		t.Fatalf("sum %v", got)
	}
	if got[0] != 4 || got[1] != 4 || got[2] != 3 {
		t.Fatalf("remain %v", got)
	}
}

func TestVUsZeroWorkers(t *testing.T) {
	if VUs(10, 0) != nil {
		t.Fatal("nil")
	}
}

func TestQPSUnlimited(t *testing.T) {
	got := QPS(0, 3)
	if len(got) != 3 || Sum(got) != 0 {
		t.Fatalf("%v", got)
	}
}

func TestQPSShard(t *testing.T) {
	got := QPS(100, 3)
	if Sum(got) != 100 {
		t.Fatalf("%v", got)
	}
}
