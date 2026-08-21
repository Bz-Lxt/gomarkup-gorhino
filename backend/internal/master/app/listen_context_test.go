package app_test

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"gorhino/internal/master/app"
	"gorhino/internal/master/store"
)

func TestListenWaitsForInFlightHTTPRequestOnCancellation(t *testing.T) {
	st, err := store.Open(filepath.Join(t.TempDir(), "gorhino.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := st.SeedWhitelist([]string{"127.0.0.1"}); err != nil {
		t.Fatal(err)
	}

	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	master := app.New(log, st)
	httpAddr, grpcAddr := unusedAddresses(t)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	listenErr := make(chan error, 1)
	go func() {
		listenErr <- app.Listen(ctx, log, httpAddr, grpcAddr, master)
	}()
	waitForHTTP(t, httpAddr, listenErr)

	conn, err := net.DialTimeout("tcp", httpAddr, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	if err := conn.SetDeadline(time.Now().Add(3 * time.Second)); err != nil {
		t.Fatal(err)
	}

	body := `{"method":"GET","url":"http://127.0.0.1:8088/echo","vu":1,"duration_sec":1,"qps":0,"version_tag":"graceful"}`
	request := fmt.Sprintf("POST /api/v1/tasks HTTP/1.1\r\nHost: %s\r\nContent-Type: application/json\r\nContent-Length: %d\r\nExpect: 100-continue\r\nConnection: close\r\n\r\n", httpAddr, len(body))
	if _, err := io.WriteString(conn, request); err != nil {
		t.Fatal(err)
	}

	reader := bufio.NewReader(conn)
	status, err := reader.ReadString('\n')
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(status, " 100 Continue") {
		t.Fatalf("expected server to begin reading request body, got %q", status)
	}
	if line, err := reader.ReadString('\n'); err != nil || line != "\r\n" {
		t.Fatalf("malformed interim response: line=%q err=%v", line, err)
	}

	cancel()
	select {
	case err := <-listenErr:
		t.Fatalf("Listen returned while an HTTP request was still in flight: %v", err)
	case <-time.After(100 * time.Millisecond):
	}

	if _, err := io.WriteString(conn, body); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(reader, &http.Request{Method: http.MethodPost})
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		payload, _ := io.ReadAll(resp.Body)
		t.Fatalf("status=%d body=%s", resp.StatusCode, payload)
	}

	select {
	case err := <-listenErr:
		if err != nil {
			t.Fatal(err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("Listen did not return after the in-flight request completed")
	}
}

func unusedAddresses(t *testing.T) (string, string) {
	t.Helper()
	httpLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	grpcLis, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		_ = httpLis.Close()
		t.Fatal(err)
	}
	httpAddr := httpLis.Addr().String()
	grpcAddr := grpcLis.Addr().String()
	if err := httpLis.Close(); err != nil {
		_ = grpcLis.Close()
		t.Fatal(err)
	}
	if err := grpcLis.Close(); err != nil {
		t.Fatal(err)
	}
	return httpAddr, grpcAddr
}

func waitForHTTP(t *testing.T, addr string, listenErr <-chan error) {
	t.Helper()
	client := &http.Client{Timeout: 100 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := client.Get("http://" + addr + "/api/v1/health")
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			_ = resp.Body.Close()
			return
		}
		select {
		case err := <-listenErr:
			t.Fatalf("Listen failed before HTTP startup: %v", err)
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("HTTP server did not start")
}
