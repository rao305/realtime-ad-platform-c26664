# ADR 0004: Local Postgres analytics + optional BigQuery

## Status
Accepted

## Context
BigQuery is the right warehouse for large-scale reporting, but requiring GCP
credentials for every local/dev run blocks the learning loop.

## Decision
- Default sink: PostgreSQL `analytics_events` (`local_sink.py`).
- Optional sink: BigQuery (`bq_sink.py`) behind `ANALYTICS_SINK=bigquery` + ADC.
- Same Kafka topic, independent consumer groups.

## Consequences
Zero-credential demos and e2e tests. Production can dual-write or cut over to BQ
without changing the adserver.
