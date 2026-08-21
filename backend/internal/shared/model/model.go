package model

import "encoding/json"

const (
	StatusDraft     = "draft"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusStopped   = "stopped"
	StatusFailed    = "failed"

	NodeAlive = "alive"
	NodeLost  = "lost"

	MaxHeaderPairs = 32
	MaxHeaderBytes = 8 << 10
	MaxBodyBytes   = 1 << 20
	MinVU          = 1
	MaxVU          = 100000
	MinDuration    = 1
	MaxDuration    = 86400
	MaxQPS         = 1000000
	MaxURLLen      = 2048
	MaxTagLen      = 64
)

type TaskSpec struct {
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	VU          int               `json:"vu"`
	DurationSec int               `json:"duration_sec"`
	QPS         int               `json:"qps"`
	VersionTag  string            `json:"version_tag"`
}

type Task struct {
	ID          string            `json:"id"`
	Method      string            `json:"method"`
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Body        string            `json:"body"`
	VU          int               `json:"vu"`
	DurationSec int               `json:"duration_sec"`
	QPS         int               `json:"qps"`
	VersionTag  string            `json:"version_tag"`
	Status      string            `json:"status"`
	CreatedAt   string            `json:"created_at"`
	StartedAt   string            `json:"started_at,omitempty"`
	EndedAt     string            `json:"ended_at,omitempty"`
}

type Snapshot struct {
	TaskID       string         `json:"task_id"`
	TS           string         `json:"ts"`
	RPS          float64        `json:"rps"`
	P50MS        float64        `json:"p50_ms"`
	P95MS        float64        `json:"p95_ms"`
	P99MS        float64        `json:"p99_ms"`
	AvgMS        float64        `json:"avg_ms"`
	ErrorRate    float64        `json:"error_rate"`
	Codes        map[string]int `json:"codes"`
	Workers      int            `json:"workers"`
	ElapsedSec   int            `json:"elapsed_sec"`
	RemainingSec int            `json:"remaining_sec"`
	Status       string         `json:"status"`
	TotalReq     int64          `json:"total_requests,omitempty"`
	TotalErr     int64          `json:"total_errors,omitempty"`
}

type Report struct {
	Task
	FinalRPS      float64        `json:"final_rps"`
	P50MS         float64        `json:"p50_ms"`
	P95MS         float64        `json:"p95_ms"`
	P99MS         float64        `json:"p99_ms"`
	AvgMS         float64        `json:"avg_ms"`
	ErrorRate     float64        `json:"error_rate"`
	TotalRequests int64          `json:"total_requests"`
	TotalErrors   int64          `json:"total_errors"`
	Codes         map[string]int `json:"codes"`
	Series        []Snapshot     `json:"series,omitempty"`
}

type Node struct {
	ID            string `json:"id"`
	Hostname      string `json:"hostname"`
	CPUCount      int    `json:"cpu_count"`
	State         string `json:"state"`
	LastHeartbeat string `json:"last_heartbeat"`
	AssignedVU    int    `json:"assigned_vu"`
}

type WorkerSnap struct {
	NodeID     string
	TaskID     string
	Seq        int64
	Histogram  []byte
	Requests   int64
	Errors     int64
	Timeouts   int64
	Other      int64
	SumLatUS   int64
	StatusCode map[int32]int64
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Envelope struct {
	OK    bool            `json:"ok"`
	Data  json.RawMessage `json:"data,omitempty"`
	Error *APIError       `json:"error,omitempty"`
}

func HeadersJSON(h map[string]string) string {
	if h == nil {
		return "{}"
	}
	b, err := json.Marshal(h)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func ParseHeaders(s string) map[string]string {
	out := map[string]string{}
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}

func CodesJSON(c map[string]int) string {
	if c == nil {
		return "{}"
	}
	b, err := json.Marshal(c)
	if err != nil {
		return "{}"
	}
	return string(b)
}

func ParseCodes(s string) map[string]int {
	out := map[string]int{}
	if s == "" {
		return out
	}
	_ = json.Unmarshal([]byte(s), &out)
	return out
}
