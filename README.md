# Real-Time Ad Delivery & Analytics Platform

So I built this project to understand what happens after an ad request reaches a
real delivery system. The result is a working local platform: Go makes the hot
path decision, Redis serves campaign state and frequency caps, Kafka carries
events, Python updates budgets, PostgreSQL keeps transactional truth, and a
second consumer writes analytics locally or to BigQuery.

## What I learned

- The low-latency read path and durable write path need different stores:
  PostgreSQL is truth; Redis is a disposable serving view.
- Kafka's at-least-once delivery is safe only when the event claim, campaign
  lock, spend update, and ledger write happen in one transaction.
- Failure policy is a product decision: targeting fails closed, frequency caps
  fail open with telemetry, and Kafka never delays the HTTP response.
- A deadline is not just a metric. At saturation the server returns no-ad rather
  than allowing a slow decision to consume the caller's latency budget.

## Architecture

```
Client → Go /serve (≤30ms) → Redis campaign view + freq caps
                 ↘ async Kafka ad-events
                      ├─ spend-processor → Postgres spend + Redis pacing
                      └─ analytics-sink → Postgres (default) or BigQuery
Postgres ──cache-sync──→ Redis indexes
```

The complete stack runs in Docker Compose. Kubernetes manifests, Prometheus,
Grafana, SLOs, alerts, and short architecture decisions are included.

## Quick start

```bash
cp .env.example .env
make up          # build + start Compose stack
make e2e         # serve → click → spend + analytics
make invariants  # targeting, frequency cap, duplicate billing
make report      # campaign report from local analytics
make load        # concurrent latency + throughput benchmark
make down        # tear down volumes
```

Services (Compose maps the ad server to host **18080** so it doesn't collide with
other local stacks on 8080):
| URL | What |
|-----|------|
| http://localhost:18080/serve | Ad decision API |
| http://localhost:18080/metrics | Prometheus metrics |
| http://localhost:9090 | Prometheus |
| http://localhost:3000 | Grafana (anonymous viewer) |



## Measured result

Measured locally with Docker Desktop on arm64, one adserver container, 1,000
requests and concurrency 50:

```text
completed=1000/1000, served=977, transport failures=0
throughput=3,756 req/s
client RTT:      p50 10.15ms | p95 27.74ms | p99 65.02ms
server decision: p50  2.00ms | p95 17.00ms | p99 30.00ms
event pipeline freshness: 586.4ms (serve → spend + analytics)
```

Client RTT includes Docker networking and queueing. At this load the server hit
its 30ms decision budget at p99; 23 requests deliberately degraded to no-ad
instead of becoming unbounded slow responses.

The integration checks also confirmed a `5/hour` cap served exactly 5 of 6
requests, and replaying one impression created one claim and one `$0.0025`
ledger charge.


## Notes

PostgreSQL analytics is the zero-credential default; BigQuery is an optional
sink. Kubernetes deploys the stateless workloads and expects managed data
services—details are in [`platform/k8s/README.md`](platform/k8s/README.md).

**Stack:** Go, Python, Kafka/Redpanda, Redis, PostgreSQL, optional BigQuery,
Docker, Kubernetes, Prometheus, and Grafana.
