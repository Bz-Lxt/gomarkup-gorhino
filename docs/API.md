# GoRhino API

Base: `http://localhost:37101/api/v1`（经 frontend-admin nginx 反代）或 `http://localhost:37102/api/v1`（直连 Master）。
时区：所有时间字段为北京时间 `yyyy-MM-dd HH:mm:ss`。
百分位：P50 / P95 / P99 为 HDR Histogram 近似值，误差 ≤ 1%。

## 信封

成功：

```json
{ "ok": true, "data": {} }
```

失败：

```json
{ "ok": false, "error": { "code": "VALIDATION_FAILED", "message": "url: not on whitelist" } }
```

## 错误码

| code | HTTP | 含义 |
|---|---|---|
| `VALIDATION_FAILED` | 400 | 字段缺失/越界、白名单或 SSRF 校验失败 |
| `TASK_NOT_FOUND` | 404 | 任务或报告不存在 |
| `TASK_ALREADY_RUNNING` | 409 | 已有任务处于 running |
| `NO_WORKERS` | 409 | 无存活 Worker |
| `UNAUTHORIZED` | 401 | V1 鉴权预留，MVP 不返回 |
| `INTERNAL` | 500 | Master 未预期错误 |

---

## GET /health

探活。

请求：无。

响应：

```json
{ "ok": true, "data": { "service": "master", "time": "2026-08-21 20:30:00", "workers": 2 } }
```

---

## GET /nodes

Worker 列表。

响应：

```json
{
  "ok": true,
  "data": {
    "nodes": [
      {
        "id": "worker-1-ab12",
        "hostname": "worker-1",
        "cpu_count": 2,
        "state": "alive",
        "last_heartbeat": "2026-08-21 20:30:01",
        "assigned_vu": 25
      }
    ]
  }
}
```

`state`: `alive` | `lost`。

---

## GET /whitelist

列出允许压测的 URL / host 模式。

```json
{ "ok": true, "data": { "patterns": ["target", "http://target:8088/echo"] } }
```

## PUT /whitelist

整表替换。

请求：

```json
{ "patterns": ["target", "target:8088", "http://target:8088/echo"] }
```

校验：至少一条；单条 ≤ 512 字节；去重。

---

## POST /tasks

创建 draft 任务。创建时即校验 URL 白名单与 SSRF。

请求字段：

| 字段 | 类型 | 约束 |
|---|---|---|
| method | string | GET/POST/PUT/DELETE/PATCH/HEAD |
| url | string | http(s)，≤2048，命中白名单，解析 IP 不得为 169.254/8 链路本地 |
| headers | object | 键值均为 string，最多 32 对，合计 ≤8KiB |
| body | string | ≤1MiB |
| vu | int | 1–100000 |
| duration_sec | int | 1–86400 |
| qps | int | 0–1000000，0=不限速 |
| version_tag | string | 必填，≤64 |

请求示例：

```json
{
  "method": "GET",
  "url": "http://target:8088/echo",
  "headers": { "User-Agent": "GoRhino/0.1" },
  "body": "",
  "vu": 50,
  "duration_sec": 30,
  "qps": 0,
  "version_tag": "v0.1.0"
}
```

响应 `201`：

```json
{
  "ok": true,
  "data": {
    "id": "tsk_ab12cd34ef56",
    "method": "GET",
    "url": "http://target:8088/echo",
    "headers": { "User-Agent": "GoRhino/0.1" },
    "body": "",
    "vu": 50,
    "duration_sec": 30,
    "qps": 0,
    "version_tag": "v0.1.0",
    "status": "draft",
    "created_at": "2026-08-21 20:31:00"
  }
}
```

## GET /tasks

```json
{ "ok": true, "data": { "items": [ { "id": "tsk_..." } ] } }
```

## GET /tasks/{id}

```json
{ "ok": true, "data": { "task": {}, "series": [] } }
```

## POST /tasks/{id}/start

将 VU 均分到当前 alive Worker，经 gRPC 双向流下发 START。无 Worker → `NO_WORKERS`。已有 running → `TASK_ALREADY_RUNNING`。

## POST /tasks/{id}/stop

优雅停止（Worker 5s 排空）。幂等：非 running 直接返回当前任务。

---

## GET /reports

已结束任务（completed / stopped / failed）。

```json
{
  "ok": true,
  "data": {
    "items": [
      {
        "id": "tsk_ab12cd34ef56",
        "version_tag": "v0.1.0",
        "url": "http://target:8088/echo",
        "vu": 50,
        "started_at": "2026-08-21 20:31:05",
        "ended_at": "2026-08-21 20:31:35",
        "final_rps": 18420.1,
        "p99_ms": 5.12,
        "error_rate": 0.001,
        "status": "completed"
      }
    ]
  }
}
```

## GET /reports/{id}

详情 + 1 秒序列。`p50_ms` / `p95_ms` / `p99_ms` 为近似值。

---

## GET /ws/live  (WebSocket)

浏览器：`ws://localhost:37101/api/v1/ws/live`。

首帧：`{"type":"hello","data":{"note":"percentiles are HDR approximate"}}`

数据帧：

```json
{
  "type": "frame",
  "data": {
    "task_id": "tsk_ab12cd34ef56",
    "ts": "2026-08-21 20:31:10",
    "rps": 18420.0,
    "p50_ms": 1.2,
    "p95_ms": 3.4,
    "p99_ms": 5.1,
    "avg_ms": 1.6,
    "error_rate": 0.002,
    "codes": { "2xx": 18000, "5xx": 20, "timeout": 5, "other": 0 },
    "workers": 2,
    "elapsed_sec": 5,
    "remaining_sec": 25,
    "status": "running"
  }
}
```

客户端可每 10s 发送 `{"type":"ping"}` 保活。

---

## 内置 Target（非 Master API）

`http://localhost:37104`

| 路径 | 说明 |
|---|---|
| GET /health | 200 |
| ANY /echo | 回显 method / UA / body 长度 |
| GET /slow?ms= | 睡眠，ms 上限 5000 |
| GET /error?rate= | 以 rate 概率返回 500 |
| GET /stats | 计数器 |
