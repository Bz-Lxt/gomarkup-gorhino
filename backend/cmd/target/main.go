package main

import (
	"net/http"
	"os"

	"gorhino/internal/config"
	"gorhino/internal/logger"
	"gorhino/internal/target"
)

func main() {
	cfg := config.TargetFromEnv()
	log := logger.New(cfg.LogLevel)
	addr := cfg.Addr
	s := target.New(log)
	log.Info("target listening", "addr", addr)
	if err := http.ListenAndServe(addr, s.Handler()); err != nil {
		log.Error("target exit", "err", err)
		os.Exit(1)
	}
}
