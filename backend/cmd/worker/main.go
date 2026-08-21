package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gorhino/internal/config"
	"gorhino/internal/logger"
	"gorhino/internal/worker/client"
)

func main() {
	cfg := config.WorkerFromEnv()
	log := logger.New(cfg.LogLevel)
	if cfg.NodeID != "" {
		_ = os.Setenv("GORHINO_NODE_ID", cfg.NodeID)
	}
	w := client.New(log, cfg.MasterGRPC)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	log.Info("worker starting", "master", cfg.MasterGRPC)
	if err := w.Run(ctx); err != nil && err != context.Canceled {
		log.Error("worker exit", "err", err)
		os.Exit(1)
	}
}
