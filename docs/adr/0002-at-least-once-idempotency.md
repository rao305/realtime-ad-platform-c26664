# ADR 0002: At-least-once Kafka with idempotent claims

## Status
Accepted

## Context
Async emission after `/serve` means we optimize for user latency, not exactly-once
broker delivery. Kafka (and Redpanda) give at-least-once to consumers.

## Decision
- Every event carries a stable `event_id`.
- Consumers `INSERT ... ON CONFLICT DO NOTHING RETURNING` into `processed_events`.
- Only the first claim accrues spend / analytics side effects.
- Offsets commit after successful processing.

## Consequences
Duplicates are cheap no-ops. Spend accuracy stays exact without exactly-once brokers.
Replays are safe for already-claimed ids.
