# Runbook: Spend reconciliation

## Symptoms
- Campaign `spent_today` disagrees with analytics impressions * bid/1000
- Duplicate suspicion after consumer restarts

## Invariants
- Each `event_id` is claimed once in `processed_events`
- Each billed impression has one `spend_ledger` row keyed by `event_id`
- Kafka delivery is at-least-once; correctness comes from the claim table

## Checks
```sql
-- Orphan claims without ledger (should be clicks or failed mid-tx)
SELECT p.event_id, p.event_type
FROM processed_events p
LEFT JOIN spend_ledger s ON s.event_id = p.event_id
WHERE p.event_type = 'impression' AND s.event_id IS NULL;

-- Analytics vs ledger for a campaign
SELECT c.id, c.spent_today,
       (SELECT COALESCE(SUM(amount),0) FROM spend_ledger WHERE campaign_id = c.id) AS ledger
FROM campaigns c;
```

## Mitigations
1. If ledger < spent_today, investigate non-ledger updates (should not happen).
2. If analytics > ledger, look for unprocessed impressions still in Kafka.
3. Never "fix" spend by double-applying events — replay only unclaimed ids.
