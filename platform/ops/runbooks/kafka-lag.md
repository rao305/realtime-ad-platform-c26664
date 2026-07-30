# Runbook: Kafka lag / emit failures

## Symptoms
- `adserver_events_emitted_total{result="error"}` increasing
- Spend/analytics not updating after serves
- Consumer group lag (check Redpanda/Kafka tooling)

## Impact
- Serving continues (async emit)
- Budgets and pacing freeze → potential overspend until catch-up

## Mitigations
1. Check broker health and topic existence (`ad-events`, `ad-events-dlq`).
2. Inspect `spend-processor` / `analytics-sink` logs for poison messages.
3. Invalid payloads go to DLQ; fix producers or replay after repair.
4. Scale processor replicas if lag is pure throughput.

## Verify
```bash
make demo
make e2e
```
