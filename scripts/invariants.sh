#!/usr/bin/env bash
set -euo pipefail

BASE="${ADSERVER_URL:-http://localhost:18080}"
RUN_ID="$(date +%s%N)"

post_serve() {
  curl -sf -X POST "$BASE/serve" \
    -H 'content-type: application/json' \
    -d "$1"
}

NO_TARGET=$(post_serve \
  "{\"request_id\":\"target-$RUN_ID\",\"user_id\":\"target-user-$RUN_ID\",\"interests\":[\"gadgets\"],\"country\":\"DE\"}")
python3 -c 'import json,sys; assert not json.load(sys.stdin)["served"]' <<<"$NO_TARGET"
echo "targeting: unmatched country returned no-ad"

SERVED=0
for i in $(seq 1 6); do
  RESP=$(post_serve \
    "{\"request_id\":\"cap-$RUN_ID-$i\",\"user_id\":\"cap-user-$RUN_ID\",\"interests\":[\"gadgets\"],\"country\":\"US\"}")
  IS_SERVED=$(python3 -c 'import json,sys; print(int(json.load(sys.stdin)["served"]))' <<<"$RESP")
  SERVED=$((SERVED + IS_SERVED))
done
test "$SERVED" -eq 5
echo "frequency cap: served 5/6 requests for one user"

REQUEST_ID="duplicate-$RUN_ID"
BODY="{\"request_id\":\"$REQUEST_ID\",\"user_id\":\"duplicate-user-$RUN_ID\",\"interests\":[\"gadgets\"],\"country\":\"US\"}"
post_serve "$BODY" >/dev/null
post_serve "$BODY" >/dev/null
EVENT_ID="imp:$REQUEST_ID:camp-gadgets"

for _ in $(seq 1 100); do
  IFS='|' read -r CLAIMS LEDGER AMOUNT <<<"$(docker compose exec -T postgres \
    psql -U ads -d ads -Atc \
    "SELECT
       (SELECT COUNT(*) FROM processed_events WHERE event_id='$EVENT_ID'),
       (SELECT COUNT(*) FROM spend_ledger WHERE event_id='$EVENT_ID'),
       COALESCE((SELECT SUM(amount) FROM spend_ledger WHERE event_id='$EVENT_ID'), 0)")"
  if [[ "$CLAIMS" -eq 1 && "$LEDGER" -eq 1 ]]; then
    python3 -c "assert abs(float('$AMOUNT') - 0.0025) < 1e-9"
    echo "idempotency: duplicate event produced one claim and one 0.0025 ledger charge"
    exit 0
  fi
  sleep 0.1
done

echo "idempotency invariant did not settle" >&2
exit 1
