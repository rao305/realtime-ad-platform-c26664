#!/usr/bin/env bash
set -euo pipefail

BASE="${ADSERVER_URL:-http://localhost:18080}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "waiting for adserver..."
for i in $(seq 1 60); do
  if curl -sf "$BASE/readyz" >/dev/null; then
    break
  fi
  sleep 2
  if [[ $i -eq 60 ]]; then
    echo "adserver not ready" >&2
    docker compose ps
    docker compose logs --tail=100 adserver cache-sync
    exit 1
  fi
done

REQ_ID="e2e-$(date +%s)"
START_NS=$(python3 -c 'import time; print(time.time_ns())')
RESP=$(curl -sf -X POST "$BASE/serve" \
  -H 'content-type: application/json' \
  -d "{\"request_id\":\"$REQ_ID\",\"user_id\":\"e2e-user\",\"context\":\"r/ai\",\"interests\":[\"ai\",\"gadgets\"],\"country\":\"US\"}")
echo "serve: $RESP"

python3 - <<PY
import json, sys
resp = json.loads('''$RESP''')
assert resp.get("served"), resp
open("/tmp/e2e_campaign","w").write(resp["campaign_id"])
open("/tmp/e2e_creative","w").write(resp["creative_id"])
PY

CAMPAIGN=$(cat /tmp/e2e_campaign)
CREATIVE=$(cat /tmp/e2e_creative)

curl -sf -X POST "$BASE/click" \
  -H 'content-type: application/json' \
  -d "{\"request_id\":\"$REQ_ID\",\"campaign_id\":\"$CAMPAIGN\",\"creative_id\":\"$CREATIVE\",\"user_id\":\"e2e-user\"}" >/dev/null

echo "waiting for pipeline..."
for i in $(seq 1 100); do
  IFS='|' read -r SPEND IMP CLK <<<"$(docker compose exec -T postgres \
    psql -U ads -d ads -Atc \
    "SELECT
       (SELECT spent_today FROM campaigns WHERE id='$CAMPAIGN'),
       COUNT(*) FILTER (WHERE type='impression'),
       COUNT(*) FILTER (WHERE type='click')
     FROM analytics_events
     WHERE request_id='$REQ_ID'")"
  if [[ "$IMP" -ge 1 && "$CLK" -ge 1 && $(python3 -c "print(float('$SPEND') > 0)") == "True" ]]; then
    FRESHNESS_MS=$(python3 -c "import time; print(f'{(time.time_ns()-$START_NS)/1_000_000:.1f}')")
    echo "e2e ok spend=$SPEND impressions=$IMP clicks=$CLK pipeline_freshness=${FRESHNESS_MS}ms"
    exit 0
  fi
  sleep 0.1
done

echo "pipeline did not settle" >&2
docker compose logs --tail=80 spend-processor analytics-sink adserver
exit 1
