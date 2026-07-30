# Service Level Objectives

| SLO | Target | SLI | Local verification |
|-----|--------|-----|--------------------|
| Serving availability | 99.95% / 30 days | `1 - rate(cache_error+timeout)/rate(all)` | Prometheus alert `ServingErrorBudgetBurn` |
| Decision latency (p99) | ≤ 30 ms | `histogram_quantile(0.99, adserver_decision_latency_seconds)` | `make load` + Grafana panel |
| Event pipeline freshness | ≤ 5 s lag | Time from emit to spend/analytics row | `make e2e` settles within ~30s (budget 5s steady-state) |
| Spend accuracy | exact | Ledger vs claimed events | Idempotent `processed_events` + `spend_ledger` |

**Error budget:** 99.95% availability allows ~21.6 min of unavailability per 30 days.
We spend that budget on risk (deploys, experiments). If we burn it, freeze risky
changes and shore up reliability.

## Failure policies (intentional)

| Dependency | Serving behavior | Why |
|------------|------------------|-----|
| Redis targeting down | Return no-ad (`cache_unavailable`) | Wrong targeting is worse than an empty slot |
| Redis frequency cap down | Fail open + metric | Prefer slight over-delivery to hard outage |
| Kafka emit failure | Still return the ad; log/metric | User-visible latency must not wait on the write path |
| Postgres processor down | Serving continues on stale pacing | Budgets catch up when the processor recovers |

Local SLO numbers from `make load` are **measured on Compose**, not production guarantees.
