package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"gorhino/internal/config"
	"gorhino/internal/logger"
	"gorhino/internal/master/app"
	"gorhino/internal/master/store"
)

func main() {
	cfg := config.MasterFromEnv()
	log := logger.New(cfg.LogLevel)
	httpAddr := cfg.HTTPAddr
	grpcAddr := cfg.GRPCAddr
	dbPath := cfg.DBPath

	st, err := store.Open(dbPath)
	if err != nil {
		log.Error("open db", "err", err)
		os.Exit(1)
	}
	defer st.Close()
	if err := st.SeedWhitelist(cfg.Whitelist); err != nil {
		log.Error("seed whitelist", "err", err)
		os.Exit(1)
	}
	if err := st.FailStaleRunning(); err != nil {
		log.Error("reconcile stale running", "err", err)
		os.Exit(1)
	}

	a := app.New(log, st)
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	a.StartLoops(ctx)
	if err := app.Listen(ctx, log, httpAddr, grpcAddr, a); err != nil {
		log.Error("listen", "err", err)
		os.Exit(1)
	}
}

