package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Master struct {
	HTTPAddr   string
	GRPCAddr   string
	DBPath     string
	LogLevel   string
	Whitelist  []string
	ReapAfter  time.Duration
	TickEvery  time.Duration
}

type Worker struct {
	MasterGRPC string
	LogLevel   string
	NodeID     string
}

type Target struct {
	Addr     string
	LogLevel string
}

func getenv(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func MasterFromEnv() Master {
	return Master{
		HTTPAddr:  getenv("GORHINO_HTTP_ADDR", ":8080"),
		GRPCAddr:  getenv("GORHINO_GRPC_ADDR", ":9090"),
		DBPath:    getenv("GORHINO_DB_PATH", "/tmp/gorhino.db"),
		LogLevel:  getenv("GORHINO_LOG_LEVEL", "info"),
		Whitelist: splitCSV(getenv("GORHINO_DEFAULT_WHITELIST", "target,target:8088,http://target:8088")),
		ReapAfter: durationEnv("GORHINO_REAP_AFTER", 10*time.Second),
		TickEvery: durationEnv("GORHINO_TICK_EVERY", time.Second),
	}
}

func WorkerFromEnv() Worker {
	return Worker{
		MasterGRPC: getenv("GORHINO_MASTER_GRPC", "127.0.0.1:9090"),
		LogLevel:   getenv("GORHINO_LOG_LEVEL", "info"),
		NodeID:     getenv("GORHINO_NODE_ID", ""),
	}
}

func TargetFromEnv() Target {
	return Target{
		Addr:     getenv("GORHINO_TARGET_ADDR", ":8088"),
		LogLevel: getenv("GORHINO_LOG_LEVEL", "info"),
	}
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

func durationEnv(k string, def time.Duration) time.Duration {
	raw := strings.TrimSpace(os.Getenv(k))
	if raw == "" {
		return def
	}
	if d, err := time.ParseDuration(raw); err == nil {
		return d
	}
	if n, err := strconv.Atoi(raw); err == nil {
		return time.Duration(n) * time.Second
	}
	return def
}

func IntEnv(k string, def int) int {
	raw := strings.TrimSpace(os.Getenv(k))
	if raw == "" {
		return def
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return n
}
