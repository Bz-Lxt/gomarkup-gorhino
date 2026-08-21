package cluster_test

import (
	"context"
	"net"
	"testing"
	"time"

	"gorhino/internal/logger"
	"gorhino/internal/master/cluster"
	"gorhino/internal/proto"
	"gorhino/internal/shared/model"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
)

func TestDispatchStartKeepsVUAndQPSShardsIndependent(t *testing.T) {
	lis := bufconn.Listen(1 << 20)
	srv := grpc.NewServer()
	c := cluster.New(logger.New("error"), "master-1")
	proto.RegisterControlServer(srv, c)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(context.Context, string) (net.Conn, error) { return lis.Dial() }),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	stream, err := proto.NewControlClient(conn).Connect(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := stream.Send(&proto.WorkerMsg{Body: &proto.WorkerMsg_Register{Register: &proto.Register{
		NodeId: "worker-1", Hostname: "test", CpuCount: 4,
	}}}); err != nil {
		t.Fatal(err)
	}
	if _, err := stream.Recv(); err != nil {
		t.Fatal(err)
	}

	workers, err := c.DispatchStart(&model.Task{
		ID: "task-1", Method: "GET", URL: "http://example.test", VU: 17, QPS: 5, DurationSec: 10,
	})
	if err != nil || workers != 1 {
		t.Fatalf("DispatchStart() = (%d, %v), want (1, nil)", workers, err)
	}
	msg, err := stream.Recv()
	if err != nil {
		t.Fatal(err)
	}
	start := msg.GetStart()
	if start == nil {
		t.Fatalf("received %T, want start task", msg.GetBody())
	}
	if got := start.GetVu(); got != 17 {
		t.Fatalf("worker VU = %d, want 17", got)
	}
	if got := start.GetQps(); got != 5 {
		t.Fatalf("worker QPS = %d, want 5", got)
	}
}
