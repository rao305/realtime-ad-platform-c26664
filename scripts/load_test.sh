#!/usr/bin/env bash
set -euo pipefail

BASE="${ADSERVER_URL:-http://localhost:18080}"
N="${N:-1000}"
CONCURRENCY="${CONCURRENCY:-50}"

echo "sending $N /serve requests with concurrency=$CONCURRENCY..."
python3 - <<PY
import concurrent.futures
import json
import math
import time
import urllib.request

base = "$BASE"
total = $N
concurrency = $CONCURRENCY
run_id = time.time_ns()

def send(i):
    body = json.dumps({
        "request_id": f"load-{run_id}-{i}",
        # Unique users keep this focused on delivery rather than intentionally
        # tripping the per-user frequency cap.
        "user_id": f"load-user-{run_id}-{i}",
        "context": "r/technology",
        "interests": ["gadgets", "ai"],
        "country": "US",
    }).encode()
    req = urllib.request.Request(
        base + "/serve",
        data=body,
        headers={"content-type": "application/json"},
    )
    t0 = time.perf_counter()
    with urllib.request.urlopen(req, timeout=5) as resp:
        payload = json.loads(resp.read())
    return (time.perf_counter() - t0) * 1000, payload

started = time.perf_counter()
latencies = []
decision_ms = []
served = 0
failures = []
with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as pool:
    futures = [pool.submit(send, i) for i in range(total)]
    for future in concurrent.futures.as_completed(futures):
        try:
            latency, payload = future.result()
            latencies.append(latency)
            decision_ms.append(float(payload.get("decision_ms", 0)))
            served += int(bool(payload.get("served")))
        except Exception as exc:
            failures.append(str(exc))
elapsed = time.perf_counter() - started

latencies.sort()
decision_ms.sort()

def pct(values, p):
    return values[min(len(values) - 1, max(0, math.ceil(len(values) * p) - 1))]

completed = len(latencies)
print(f"completed={completed}/{total} served={served} failures={len(failures)}")
print(f"throughput={completed / elapsed:.1f} req/s elapsed={elapsed:.2f}s")
if latencies:
    print(
        "client_rtt "
        f"p50={pct(latencies, 0.50):.2f}ms "
        f"p95={pct(latencies, 0.95):.2f}ms "
        f"p99={pct(latencies, 0.99):.2f}ms "
        f"max={latencies[-1]:.2f}ms"
    )
    print(
        "server_decision "
        f"p50={pct(decision_ms, 0.50):.2f}ms "
        f"p95={pct(decision_ms, 0.95):.2f}ms "
        f"p99={pct(decision_ms, 0.99):.2f}ms "
        f"max={decision_ms[-1]:.2f}ms"
    )
if failures:
    print(f"first_failure={failures[0]}")
    raise SystemExit(1)
PY
