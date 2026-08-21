package cluster

import (
	"testing"
	"time"

	"gorhino/internal/logger"
	"gorhino/internal/shared/model"
	"gorhino/internal/timeutil"
)

func TestEmptyCluster(t *testing.T) {
	c := New(logger.New("error"), "m1")
	if c.AliveCount() != 0 {
		t.Fatal("alive")
	}
	if len(c.List()) != 0 {
		t.Fatal("list")
	}
	n, err := c.DispatchStart(&model.Task{ID: "t", VU: 10, DurationSec: 1})
	if err != nil || n != 0 {
		t.Fatalf("%d %v", n, err)
	}
	c.DispatchStop("t")
}

func TestReapStale(t *testing.T) {
	c := New(logger.New("error"), "m1")
	c.mu.Lock()
	c.nodes["dead"] = &node{info: model.Node{
		ID: "dead", State: model.NodeAlive, LastHeartbeat: timeutil.Format(timeutil.Now().Add(-time.Minute)),
	}}
	c.nodes["fresh"] = &node{info: model.Node{
		ID: "fresh", State: model.NodeAlive, LastHeartbeat: timeutil.NowString(),
	}}
	c.mu.Unlock()
	c.Reap(10 * time.Second)
	ids := map[string]bool{}
	for _, n := range c.List() {
		ids[n.ID] = true
	}
	if ids["dead"] || !ids["fresh"] {
		t.Fatalf("%v", ids)
	}
}
