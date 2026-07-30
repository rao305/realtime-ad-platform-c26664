#!/usr/bin/env bash
set -euo pipefail

BASE="${ADSERVER_URL:-http://localhost:18080}"

echo "== health =="
curl -sf "$BASE/healthz"
echo

echo "== serve =="
RESP=$(curl -sf -X POST "$BASE/serve" \
  -H 'content-type: application/json' \
  -d '{"request_id":"demo-1","user_id":"user-42","context":"r/technology","interests":["gadgets","ai"],"country":"US"}')
echo "$RESP"

CAMPAIGN=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["campaign_id"])' <<<"$RESP")
CREATIVE=$(python3 -c 'import json,sys; print(json.load(sys.stdin)["creative_id"])' <<<"$RESP")

if [[ -z "$CAMPAIGN" ]]; then
  echo "no campaign served" >&2
  exit 1
fi

echo "== click =="
curl -sf -X POST "$BASE/click" \
  -H 'content-type: application/json' \
  -d "{\"request_id\":\"demo-1\",\"campaign_id\":\"$CAMPAIGN\",\"creative_id\":\"$CREATIVE\",\"user_id\":\"user-42\"}"
echo
echo "demo complete"
