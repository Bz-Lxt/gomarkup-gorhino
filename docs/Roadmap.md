# GoRhino Roadmap

Status: ACTIVE
Date: 2026-08-21 (GMT+8)
Source of WHAT: `docs/Requirements.md`
This file defines WHEN.

---

## Phase Order Decision

**UI-First.**

Rationale: the console is a CRUD + live-dashboard surface. Pages (task form, telemetry wall, report list/detail) can be built against the API schema below. The UI is not an editor / timeline / canvas whose widgets are derived from a domain DSL, so Logic-First swap is not triggered.

---

## Architect Picks (locked)

| Topic | Choice | Why |
|---|---|---|
| Master HTTP | Go 1.22+ `net/http` ServeMux | No second framework; stdlib method patterns are enough |
| SQLite driver | `modernc.org/sqlite` | CGO-free, multi-arch alpine builds stay simple |
| Chart | ECharts | line series + dual axis, no custom canvas |
| WS library | `github.com/gorilla/websocket` | mature, well-known upgrade path |
| Engine | `github.com/valyala/fasthttp` | connection reuse, high RPS |
| Histogram | `github.com/HdrHistogram/hdrhistogram-go` | mergeable, constant memory |
| Dev host ports | 37101 console / 37102 master HTTP / 37103 master gRPC / 37104 target | random 10k–60k band, avoid 1848x |

SOP `frontend-user` / `frontend-mp` directories are **not created**. Requirements §6 forbids unused stubs. This console is `frontend-admin` only.

---

## API Contract Sketch (UI builds against this)

Envelope:

```json
{ "ok": true, "data": {} }
{ "ok": false, "error": { "code": "VALIDATION_FAILED", "message": "url is not on whitelist" } }
```

| Method | Path | Purpose |
|---|---|---|
| GET | `/api/v1/health` | liveness |
| GET | `/api/v1/nodes` | alive / lost workers |
| GET | `/api/v1/whitelist` | list patterns |
| PUT | `/api/v1/whitelist` | replace patterns |
| POST | `/api/v1/tasks` | create draft |
| GET | `/api/v1/tasks` | list |
| GET | `/api/v1/tasks/{id}` | detail + snapshots |
| POST | `/api/v1/tasks/{id}/start` | dispatch to workers |
| POST | `/api/v1/tasks/{id}/stop` | graceful stop |
| GET | `/api/v1/reports` | completed/stopped/failed list |
| GET | `/api/v1/reports/{id}` | summary + 1s series |
| WS | `/api/v1/ws/live` | 1s frames for the running task |

Task body (create):

```json
{
  "method": "GET",
  "url": "http://target:8088/echo",
  "headers": { "X-Demo": "1" },
  "body": "",
  "vu": 50,
  "duration_sec": 30,
  "qps": 0,
  "version_tag": "v0.1.0"
}
```

Live frame:

```json
{
  "task_id": "tsk_...",
  "ts": "2026-08-21 20:30:01",
  "rps": 18420.0,
  "p50_ms": 1.2,
  "p95_ms": 3.4,
  "p99_ms": 5.1,
  "avg_ms": 1.6,
  "error_rate": 0.002,
  "codes": { "2xx": 18000, "5xx": 20, "timeout": 5, "other": 0 },
  "workers": 2,
  "elapsed_sec": 12,
  "remaining_sec": 18,
  "status": "running"
}
```

Percentile fields are **approximate** (HDR, ≤ 1% error).

---

## MVP Build Order (this `/auto`)

| ID | Item | Owner | Done |
|---|---|---|---|
| M0 | git init, gitignore, Compose skeleton, this Roadmap | Architect | [x] |
| M1 | DesignSpec + Vue pages (config / live / reports / nodes) | UI | [x] |
| M2 | proto + logger + timeutil + validate + shard + histogram | Logic | [x] |
| M3 | Target service | Logic | [x] |
| M4 | Worker engine + gRPC client | Logic | [x] |
| M5 | Master store / cluster / agg / REST / WS | Logic | [x] |
| M6 | Dockerfiles, nginx proxy, Compose wire-up | Logic | [x] |
| M7 | Unit tests + `docs/API.md` | Logic | [x] |
| M8 | Playwright + API smoke, QA_Record | QA | [x] |
| M9 | AuditReport + `/learn` | Auditor | [x] |

---

## V1 (not in this `/auto`)

- Version-tag multi-report diff UI
- Linear VU ramp-up
- Mid-task VU reshard on worker death
- Shared-token login for REST + WS
- Playwright journey already lands in MVP QA; V1 expands assertions
- Full error-code table polish

## V2 (out of first delivery)

- SLO assertions, CSV params, multi-step DSL, report export

---

## Directory Plan

```
backend/
  cmd/master/main.go
  cmd/worker/main.go
  cmd/target/main.go
  proto/control.proto
  internal/logger/
  internal/timeutil/
  internal/proto/          # generated
  internal/shared/{model,validate,histogram,shard}
  internal/master/{api,ws,store,cluster,agg,app}
  internal/worker/{engine,client}
  internal/target/
frontend-admin/            # Vue 3 + Vite + Tailwind + ECharts
tests/e2e_flow.spec.ts
docker-compose.yml
```

Go file budget: 25–35 files, 5000–7500 lines including tests. No empty packages.

---

## Compose (dev random ports)

| Service | Container port | Host port |
|---|---|---|
| frontend-admin | 80 | 37101 |
| master HTTP | 8080 | 37102 |
| master gRPC | 9090 | 37103 |
| target | 8088 | 37104 |
| worker | (no host publish) | scale with `--scale worker=N` |

`/deploy` later rewrites host ports to 8081+.

---

## Definition of Done (MVP)

- `docker compose up --build -d` serves the console at `http://localhost:37101`
- Creating a task against `http://target:8088/echo` and starting it produces live RPS / P95 / P99 charts
- Stopping or expiry writes a report that can be reopened
- `--scale worker=2` shows two alive nodes and shards VUs
- Unit tests cover histogram merge, VU shard, whitelist/SSRF, task CRUD
- QA round cost ¥0
