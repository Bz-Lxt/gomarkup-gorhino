"""API smoke against builtin Target. Cost ¥0."""
from __future__ import annotations

import json
import os
import sys
import time
import urllib.error
import urllib.request

BASE = os.environ.get("GORHINO_API", "http://127.0.0.1:37102")


def req(method: str, path: str, body=None, expect=200):
    data = None if body is None else json.dumps(body).encode()
    r = urllib.request.Request(BASE + path, data=data, method=method)
    if body is not None:
        r.add_header("Content-Type", "application/json")
    try:
        with urllib.request.urlopen(r, timeout=10) as resp:
            raw = resp.read()
            code = resp.status
    except urllib.error.HTTPError as e:
        raw = e.read()
        code = e.code
    payload = json.loads(raw.decode())
    if code != expect:
        raise SystemExit(f"{method} {path} -> {code} want {expect}: {payload}")
    return payload


def main() -> int:
    h = req("GET", "/api/v1/health")
    assert h["ok"] and h["data"]["service"] == "master"
    wl = req("GET", "/api/v1/whitelist")
    assert "target" in "".join(wl["data"]["patterns"])
    bad = req(
        "POST",
        "/api/v1/tasks",
        {
            "method": "GET",
            "url": "http://example.com/",
            "vu": 1,
            "duration_sec": 1,
            "qps": 0,
            "version_tag": "smoke",
        },
        expect=400,
    )
    assert bad["error"]["code"] == "VALIDATION_FAILED"
    created = req(
        "POST",
        "/api/v1/tasks",
        {
            "method": "GET",
            "url": "http://target:8088/echo",
            "vu": 4,
            "duration_sec": 3,
            "qps": 20,
            "version_tag": "smoke-v1",
            "headers": {"X-Smoke": "1"},
        },
        expect=201,
    )
    tid = created["data"]["id"]
    started = req("POST", f"/api/v1/tasks/{tid}/start")
    assert started["ok"]
    time.sleep(1.2)
    live = req("GET", f"/api/v1/tasks/{tid}")
    assert live["ok"]
    print("SMOKE_OK", tid)
    return 0


if __name__ == "__main__":
    sys.exit(main())
