# Runbook: Redis unavailable

## Symptoms
- `/readyz` returns 503
- `adserver_decisions_total{outcome="cache_error"}` rising
- `adserver_freq_fail_open_total` rising if only frequency path fails

## Impact
- Targeting path fails closed → empty ads
- Frequency caps may fail open if HGETALL works but INCR fails (rare)

## Mitigations
1. Check Redis pod/service health and network policies.
2. Confirm `cache-sync` is publishing (`cache_sync_total{result="ok"}`).
3. If Redis is unhealthy, scale down adserver traffic or fail traffic to a
   degraded house-ad path (not implemented here — return empty).
4. After Redis recovers, wait one `CACHE_SYNC_INTERVAL_SEC` for indexes to rebuild.

## Verify
```bash
curl -sf localhost:8080/readyz
docker compose exec redis redis-cli GET cache:generation
```
