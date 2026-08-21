# GoRhino Requirements

Status: FROZEN
Date: 2026-08-21 (GMT+8)
Command: `/pm`
Source of truth: this file defines WHAT. `docs/Roadmap.md` (Phase 1) defines WHEN.

---

## 1. Product Goal

Build a Docker-deliverable distributed HTTP load-testing platform (Mini Locust / JMeter class). Operators configure a task in a Vue 3 console. Multiple Go Worker nodes fire concurrent requests. A single Master aggregates millisecond-scale response samples into second-granularity live stats and persisted historical reports.

The platform must start with `docker compose up` and expose a browser-accessible console on localhost. Worker count scales with `docker compose up --scale worker=N`.

---

## 2. Abandonment Assessment

| Criterion | Result | Note |
|---|---|---|
| Incomplete / vague | PASS | Subject, stack, pages, and backend cores are specified. |
| Windows exclusive | PASS | Linux containers only. |
| Scale | ACCEPT (10k–40k band) | Go 5000–7500 LoC + Vue console ≈ 10k–14k LoC. Phased Roadmap is mandatory before any code. |
| External dependency | PASS (Scenario A) | Builtin target service is a complete stand-in. No live factual data (stocks / sports / traffic). |
| Specialized / paid software | PASS | gRPC, fasthttp, Vue 3, Tailwind, SQLite are all open source. |

Decision: **ACCEPT**. Do not start implementation until `docs/Roadmap.md` exists with explicit MVP / V1 / V2 boundaries.

---

## 3. Frozen User Decisions

These three answers were collected during PM and must not be reopened without a new `/pm`.

1. **Target scope**: builtin Target service plus a configurable external URL whitelist. Non-whitelist destinations are rejected. After DNS resolve, destination IPs are validated; link-local / metadata ranges (`169.254.0.0/16`, `169.254.169.254`) and other SSRF-sensitive ranges are denied.
2. **Version comparison source**: operator types a version tag or Git SHA on the task form. Reports are grouped and compared by that tag. No Git clone, no `/version` scrape.
3. **Persistence**: SQLite single file. WAL mode. Master is the only writer.

---

## 4. PM Design Rulings

These are frozen interpretations of contradictory or underspecified phrases in the original prompt.

### 4.1 Node discovery

"Self-discovery" does **not** mean Consul, etcd, or mDNS. In Docker Compose there is no need for a service registry.

Mechanism: each Worker dials Master and opens one gRPC bidirectional stream. Registration, heartbeat, task dispatch, and 1-second histogram upload all travel on that stream. Master never needs to dial Workers. Scaling is `docker compose up --scale worker=N`.

### 4.2 Percentile precision

Exact P99 requires retaining every sample. That contradicts "millisecond aggregation of tens of thousands of responses" because memory grows linearly with request count.

Mechanism: HDR Histogram (`github.com/HdrHistogram/hdrhistogram-go`). Constant memory, mergeable across Workers, ≤ 1% error at 3 significant digits. UI and docs must label P50 / P95 / P99 as **approximate**.

### 4.3 SQLite and "distributed"

"Distributed" refers to the **Worker tier** only. Master is a single process and a single SQLite writer. Workers never open the database. This is an accepted risk: Master is not horizontally scalable. Migrating to PostgreSQL is out of scope unless a later `/pm` reopens persistence.

---

## 5. Contradiction and Gap Log

| # | Contradiction / gap | Resolution |
|---|---|---|
| C1 | "Compare load-test data across code versions" — the platform cannot observe the target's source code. | Manual version tag on the task form (user decision 2). |
| C2 | Exact P99 vs millisecond aggregation of huge sample streams. | HDR Histogram approximation, error ≤ 1% (ruling 4.2). |
| C3 | "Self-discovery" vs container networking without a registry. | Worker-initiated gRPC bidi stream (ruling 4.1). |
| C4 | "Press any URL" vs Docker delivery safety / abuse / SSRF. | Builtin Target + whitelist + post-resolve IP checks (user decision 1). |
| C5 | "Distributed platform" vs SQLite single-writer. | Distribution is Worker-only; Master is a singleton (ruling 4.3). |

---

## 6. Architecture

```
Browser (Vue 3 + Tailwind)
    | REST (task / report / nodes)
    | WebSocket (1s live stats)
    v
Master  -- SQLite (WAL, single writer)
    ^
    | gRPC bidirectional stream
    | register / heartbeat / dispatch / histogram
    v
Worker 1 .. Worker N   (fasthttp engine, local HDR histogram)
    |
    | high-concurrency HTTP
    v
Builtin Target  (injectable latency / error rate)
    or
Whitelisted external URL
```

Directory contract (Phase 1 must follow SOP structure):

| Path | Role |
|---|---|
| `backend/` | Go workspace. Binaries: `cmd/master`, `cmd/worker`, `cmd/target`. |
| `frontend-admin/` | Vue 3 + Tailwind operator console. |
| `frontend-user/` | Not used. Do not create a stub. |
| `frontend-mp/` | Not used. Do not create a stub. |
| `docker-compose.yml` | Dev: random host ports. `/deploy` later standardizes to 8081+. |

Three processes, one Compose file:

- `master` — HTTP API + WebSocket + gRPC server + SQLite.
- `worker` — scalable replica.
- `target` — builtin victim used for demo, QA, and the RPS baseline.
- `frontend-admin` — static build served by nginx (or Vite in local-only; production path is nginx in Compose).

---

## 7. Functional Requirements

### 7.1 Task configuration (MVP)

An operator can create a task with:

- HTTP method (GET / POST / PUT / DELETE / PATCH / HEAD)
- URL (must pass whitelist + SSRF checks)
- Headers (map, size-capped)
- Body (optional, size-capped)
- Concurrent users (VU), integer ≥ 1
- Duration in seconds, integer ≥ 1
- Version tag (free text, required for later comparison; MVP stores it, V1 compares it)
- Optional QPS cap (token bucket). Empty means "as fast as VUs allow"

Business rules:

- Only one **running** task at a time (MVP). A second start is rejected with a clear error.
- Master shards VUs across currently alive Workers as evenly as possible. Remainder goes to the first N Workers.
- Start is rejected if zero Workers are alive.
- Stop is graceful: Master sends STOP on the stream; Workers drain in-flight requests up to a 5s deadline, then flush a final histogram.

### 7.2 Live monitor (MVP)

WebSocket stream, 1-second frames, containing at least:

- timestamp (Beijing time, display format `YYYY-MM-DD HH:mm:ss`)
- RPS (completed requests in the last second)
- P50 / P95 / P99 latency (microseconds internally, milliseconds on UI)
- average latency
- error rate (non-2xx plus transport errors) / total
- status-code histogram (group 2xx / 3xx / 4xx / 5xx / timeout / other)
- alive Worker count
- task id and elapsed / remaining seconds

UI: line charts for RPS and P95 / P99. Numeric tiles for the rest. Charts must remain readable at 1280px width.

### 7.3 Historical reports (MVP list + detail; V1 compare)

MVP:

- list: task id, version tag, URL, VU, duration, start/end (Beijing), final RPS, P99, error rate, status
- detail: same summary plus the 1s time series stored for the run

V1:

- multi-select two or more reports that share or differ by version tag
- side-by-side delta: RPS, P50 / P95 / P99, error rate
- UI must state percentiles are approximate

### 7.4 Master–Worker control plane (MVP + V1)

MVP:

- Worker registers on connect with node id, hostname, CPU count
- heartbeat every 3s on the same stream
- Master marks a Worker dead after 10s without a heartbeat
- Master fans task START / STOP on the stream
- Worker uploads a compressed HDR snapshot plus counters every 1s while a task is running

V1:

- if a Worker dies mid-task, Master redistributes remaining VUs to surviving Workers
- if all Workers die, task fails and the report is marked `failed`

### 7.5 High-pressure engine (MVP)

- fasthttp client with connection reuse
- one goroutine (or equivalent VU loop) per assigned virtual user
- token-bucket rate limit when QPS cap is set
- per-Worker local HDR histogram; **never** ship raw samples to Master
- Worker resident memory ≤ 256MB and must not grow with cumulative request count

V1:

- ramp-up: linear VU climb from 0 to target over N seconds

### 7.6 Builtin Target (MVP)

HTTP service used as the default press target.

- `/health` — 200
- `/echo` — echo method / selected headers / body
- `/slow?ms=` — sleep then 200 (capped, e.g. 5000ms)
- `/error?rate=` — probabilistic 500
- `/stats` — request counters for self-proof

Target is the only destination used by QA and by the RPS acceptance test.

### 7.7 Auth (V1)

Simple shared-token or single admin account login for the console and REST API. WebSocket must authenticate. No multi-tenant RBAC.

### 7.8 Cost visibility

No metered external AI / map / SMS / payment API exists. Cost-before-click UI is **not applicable**. QA cost target is ¥0 every round.

---

## 8. Non-Functional / Acceptance Baselines

These numbers are the acceptance criteria, not prose.

| ID | Metric | Baseline |
|---|---|---|
| A1 | Single Worker, 2 vCPU, press builtin Target | RPS ≥ 20,000 |
| A2 | Target-side p99 under A1 | ≤ 50ms |
| A3 | Console end-to-end freshness | ≤ 2s (1s window + 1s slack) |
| A4 | Percentile error vs exact on a recorded sample set | ≤ 1% |
| A5 | Worker RSS | ≤ 256MB and flat vs cumulative requests |
| A6 | Heartbeat interval / death | 3s / 10s |
| A7 | Master fan-in | ≥ 10 Workers reporting, no dropped 1s frames |
| A8 | Dispatch latency | START reaches all alive Workers and they begin firing within 1s |
| A9 | QA spend per round | ¥0 (builtin Target / mock only) |
| A10 | Timezone | `TZ=Asia/Shanghai`; all stored and displayed timestamps are Beijing (GMT+8) |
| A11 | Platforms | Images must pull on linux/arm64 and linux/amd64 |
| A12 | Delivery | `docker compose up --build` exposes the console on localhost with no host-side npm/go install |

---

## 9. Compatibility and Logic Check

| Topic | Result |
|---|---|
| Docker Delivery Standard | PASS. Master + Worker + Target + nginx console. WeChat Mini Program exception does not apply. |
| Native shell | Not requested. No Tauri/Electron. |
| Stack vs Docker | Go 1.25 alpine, Node 22 alpine (build), nginx alpine. Multi-arch official images. |
| Mock legitimacy | Builtin Target is a real HTTP service, not a silent fake of the engine. External whitelist is real code with a documented off-by-default deny. README §7 will describe whitelist vs Target. |
| Phase order (advisory for Architect) | **UI-First**. Console pages (config / live / reports) can be built against a schema sketch. The live chart is not a canvas/editor whose widgets are derived from a domain DSL. |

---

## 10. Tech Stack (frozen)

| Layer | Choice |
|---|---|
| Language | Go 1.25 |
| Control plane | gRPC bidirectional stream (protobuf) |
| HTTP engine | fasthttp |
| Histogram | hdrhistogram-go |
| Master HTTP | Go standard `net/http` or chi/echo — Architect picks one; no second framework later |
| Persistence | SQLite + WAL, `modernc.org/sqlite` or `mattn/go-sqlite3` — Architect picks one CGO-free option preferred |
| Frontend | Vue 3 + Vite + Tailwind + a chart lib (ECharts or lightweight equivalent) |
| Realtime | WebSocket from Master to browser |
| Time | Beijing timezone helper; never `time.Now().UTC()` in user-facing fields |

---

## 11. Compliance with Global Memory

Copied from `knowledge-base/global.md` and bound to this project:

- [Robustness] All HTTP JSON and gRPC payloads must be validated for required fields, types, and bounds. Reject incomplete structs; do not trust "it parsed".
- [Logging] One `internal/logger` with level control. Production default hides debug. No scattered `fmt.Println` / `console.log` in shipped paths.
- [Documentation] Deliver `docs/API.md` with per-endpoint request/response examples, parameter types, and an error-code table.
- [Testing] Backend unit tests must cover histogram merge, VU sharding, whitelist/SSRF checks, and task CRUD. Web E2E via Playwright against Compose. Smoke tests run in Target/mock mode only.

---

## 12. Phased Boundaries (mandatory — 10k–40k band)

Architect must copy these boundaries into `docs/Roadmap.md` and must not write V2 code during `/auto` unless a later command expands scope.

### MVP (in-scope for first `/auto`)

- Single-task HTTP load test
- gRPC register / heartbeat / dispatch / 1s histogram upload
- Master merge + WebSocket live charts (RPS, P95, P99)
- SQLite persist + report list + report detail
- Builtin Target
- URL whitelist + SSRF IP checks
- Version tag stored (no compare UI yet)
- `docker compose up --build` and `--scale worker=N`
- Unified logger, Beijing time, health endpoints
- Unit tests for merge / shard / whitelist / CRUD
- API.md covering shipped endpoints

### V1 (same product, next increment)

- Multi-report version-tag diff
- Linear ramp-up
- Worker death → VU reshard
- Simple login for console + API + WS
- Playwright E2E of create → run → live numbers → report
- Complete error-code table

### V2 (explicitly out of first delivery)

- Assertion rules (status / latency SLO fail the run)
- CSV parameterization
- Multi-step transaction DSL
- Report export (CSV / PDF)

---

## 13. Out of Scope

- Master high availability / multi-replica / leader election
- Record-and-replay
- Multi-tenant RBAC
- Pressing arbitrary public URLs without whitelist
- Kubernetes Operator
- Protocols other than HTTP/HTTPS (gRPC-as-target, MQTT, WebSocket-as-target)
- Git integration or target `/version` scraping
- Metered third-party APIs

---

## 14. Code Volume Guardrail

Original prompt asks for ~25–35 Go files and 5000–7500 Go lines. Implementation must stay inside that band for the Go tree. Do not generate unused packages to inflate counts. Do not split one idea into many empty files.

Suggested Go layout (Architect may rename, not explode):

- `cmd/master`, `cmd/worker`, `cmd/target`
- `internal/logger`, `internal/timeutil`
- `internal/proto` (generated)
- `internal/master/{api,ws,store,cluster,agg}`
- `internal/worker/{engine,client}`
- `internal/shared/{histogram,validate,model}`
- `internal/target`

---

## 15. Error Model (minimum, to be expanded in API.md)

| Code | HTTP | Meaning |
|---|---|---|
| `VALIDATION_FAILED` | 400 | Missing/invalid field, failed whitelist or SSRF check |
| `TASK_ALREADY_RUNNING` | 409 | A task is already in `running` |
| `NO_WORKERS` | 409 | Start requested with zero alive Workers |
| `TASK_NOT_FOUND` | 404 | Unknown task / report id |
| `UNAUTHORIZED` | 401 | V1 auth only |
| `INTERNAL` | 500 | Unexpected Master error |

gRPC stream errors are logged on Master and surfaced to the console as Worker state `lost`, never as raw stack traces.

---

## 16. Handover

Requirements are frozen.

Next command: `/auto` — Chief Architect generates `docs/Roadmap.md` (MVP / V1 / V2 + UI-First phase-order decision), git init, skeleton, and random-port Compose. No application code before that Roadmap exists.
