package cluster

import (
	"io"
	"log/slog"
	"sync"
	"time"

	"gorhino/internal/proto"
	"gorhino/internal/shared/model"
	"gorhino/internal/shared/shard"
	"gorhino/internal/timeutil"

	"google.golang.org/grpc/peer"
)

type outbound struct {
	ch chan *proto.MasterMsg
}

type node struct {
	info   model.Node
	out    *outbound
	cancel func()
}

type Cluster struct {
	proto.UnimplementedControlServer
	log     *slog.Logger
	mu      sync.Mutex
	nodes   map[string]*node
	onStats func(model.WorkerSnap)
	master  string
}

func New(log *slog.Logger, masterID string) *Cluster {
	return &Cluster{
		log:    log,
		nodes:  map[string]*node{},
		master: masterID,
	}
}

func (c *Cluster) SetStatsHandler(fn func(model.WorkerSnap)) {
	c.mu.Lock()
	c.onStats = fn
	c.mu.Unlock()
}

func (c *Cluster) Connect(stream proto.Control_ConnectServer) error {
	first, err := stream.Recv()
	if err != nil {
		return err
	}
	reg := first.GetRegister()
	if reg == nil || reg.GetNodeId() == "" {
		return io.ErrUnexpectedEOF
	}
	host := reg.GetHostname()
	if p, ok := peer.FromContext(stream.Context()); ok && host == "" {
		host = p.Addr.String()
	}
	out := &outbound{ch: make(chan *proto.MasterMsg, 16)}
	n := &node{
		info: model.Node{
			ID:            reg.GetNodeId(),
			Hostname:      host,
			CPUCount:      int(reg.GetCpuCount()),
			State:         model.NodeAlive,
			LastHeartbeat: timeutil.NowString(),
		},
		out: out,
	}
	c.mu.Lock()
	c.nodes[n.info.ID] = n
	c.mu.Unlock()
	c.log.Info("worker registered", "id", n.info.ID, "host", n.info.Hostname)

	errCh := make(chan error, 2)
	go func() {
		_ = stream.Send(&proto.MasterMsg{Body: &proto.MasterMsg_Welcome{Welcome: &proto.Welcome{MasterId: c.master}}})
		for msg := range out.ch {
			if err := stream.Send(msg); err != nil {
				errCh <- err
				return
			}
		}
	}()
	go func() {
		for {
			msg, err := stream.Recv()
			if err != nil {
				errCh <- err
				return
			}
			c.handle(n.info.ID, msg)
		}
	}()
	err = <-errCh
	c.drop(n.info.ID)
	close(out.ch)
	return err
}

func (c *Cluster) handle(id string, msg *proto.WorkerMsg) {
	switch b := msg.Body.(type) {
	case *proto.WorkerMsg_Heartbeat:
		c.mu.Lock()
		if n, ok := c.nodes[id]; ok {
			n.info.LastHeartbeat = timeutil.NowString()
			n.info.State = model.NodeAlive
		}
		c.mu.Unlock()
	case *proto.WorkerMsg_Stats:
		st := b.Stats
		c.mu.Lock()
		fn := c.onStats
		c.mu.Unlock()
		if fn != nil {
			fn(model.WorkerSnap{
				NodeID:     id,
				TaskID:     st.GetTaskId(),
				Seq:        st.GetSeq(),
				Histogram:  st.GetHistogram(),
				Requests:   st.GetRequests(),
				Errors:     st.GetErrors(),
				Timeouts:   st.GetTimeoutCount(),
				Other:      st.GetOtherCount(),
				SumLatUS:   st.GetSumLatencyUs(),
				StatusCode: st.GetStatusCodes(),
			})
		}
	}
}

func (c *Cluster) drop(id string) {
	c.mu.Lock()
	if n, ok := c.nodes[id]; ok {
		n.info.State = model.NodeLost
		delete(c.nodes, id)
	}
	c.mu.Unlock()
	c.log.Warn("worker lost", "id", id)
}

func (c *Cluster) Reap(maxAge time.Duration) {
	cutoff := time.Now().Add(-maxAge)
	c.mu.Lock()
	defer c.mu.Unlock()
	for id, n := range c.nodes {
		ts, err := timeutil.Parse(n.info.LastHeartbeat)
		if err != nil || ts.Before(cutoff) {
			n.info.State = model.NodeLost
			delete(c.nodes, id)
			c.log.Warn("worker heartbeat timeout", "id", id)
		}
	}
}

func (c *Cluster) List() []model.Node {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]model.Node, 0, len(c.nodes))
	for _, n := range c.nodes {
		out = append(out, n.info)
	}
	return out
}

func (c *Cluster) AliveIDs() []string {
	c.mu.Lock()
	defer c.mu.Unlock()
	ids := make([]string, 0, len(c.nodes))
	for id, n := range c.nodes {
		if n.info.State == model.NodeAlive {
			ids = append(ids, id)
		}
	}
	return ids
}

func (c *Cluster) DispatchStart(task *model.Task) (int, error) {
	c.mu.Lock()
	ids := make([]string, 0, len(c.nodes))
	for id, n := range c.nodes {
		if n.info.State == model.NodeAlive {
			ids = append(ids, id)
		}
	}
	vus := shard.VUs(task.VU, len(ids))
	qps := shard.QPS(task.QPS, len(ids))
	for i, id := range ids {
		n := c.nodes[id]
		n.info.AssignedVU = 0
		if i < len(vus) {
			n.info.AssignedVU = vus[i]
		}
		msg := &proto.MasterMsg{Body: &proto.MasterMsg_Start{Start: &proto.StartTask{
			TaskId:      task.ID,
			Method:      task.Method,
			Url:         task.URL,
			Headers:     task.Headers,
			Body:        []byte(task.Body),
			Vu:          int32(n.info.AssignedVU),
			DurationSec: int32(task.DurationSec),
			Qps:         0,
		}}}
		if i < len(qps) {
			msg.GetStart().Qps = int32(qps[i])
		}
		select {
		case n.out.ch <- msg:
		default:
			c.log.Error("start queue full", "id", id)
		}
	}
	alive := len(ids)
	c.mu.Unlock()
	return alive, nil
}

func (c *Cluster) DispatchStop(taskID string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	msg := &proto.MasterMsg{Body: &proto.MasterMsg_Stop{Stop: &proto.StopTask{TaskId: taskID}}}
	for _, n := range c.nodes {
		n.info.AssignedVU = 0
		select {
		case n.out.ch <- msg:
		default:
		}
	}
}

func (c *Cluster) AliveCount() int {
	return len(c.AliveIDs())
}
