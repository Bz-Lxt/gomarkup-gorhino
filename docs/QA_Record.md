# QA Record

Cost target: ¥0 (builtin Target only). No metered API.

## Round 1 — 2026-08-21 20:40 (GMT+8)

Cost: ¥0

- [PASS] Docker build (master / worker / target / frontend-admin, linux/arm64 host)
- [PASS] Health `GET /api/v1/health` via 37102 and nginx 37101
- [PASS] Two workers registered, heartbeat alive
- [PASS] Target `/health`
- [PASS] API smoke create+start against `http://target:8088/echo`
- [FAIL] `GET /api/v1/reports` hung; subsequent writes (`POST /tasks`) also hung. Playwright create button stuck disabled.
- Root cause: `store.ListReports` nested `QueryRow` while `rows` still open. `MaxOpenConns(1)` → self-deadlock.
- Fix: single LEFT JOIN query. Anti-flip-flop: keep this JOIN, do not revert to per-row lookup.

## Round 2 — 2026-08-21 20:44 (GMT+8)

Cost: ¥0

- [PASS] Rebuild master, recreate workers
- [PASS] `GET /reports` returns first smoke run (RPS≈18 with QPS 20, P99≈2.3ms, error_rate=0)
- [PASS] API smoke again
- [PASS] Playwright 4/4 after locator strict-mode fix (`getByRole('heading', { name: '实时监控' })`)

Commands:

```
docker compose up --build -d --scale worker=2
python3 tests/api_smoke.py
cd tests && npx playwright test
```
