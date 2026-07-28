# Real-Time Ad Delivery & Analytics Platform

An advanced capstone that designs a production-grade advertising backend end to end. You serve an ad request in single-digit milliseconds from a Go ad server; decide which campaign wins through eligibility filtering, ranking, budget pacing, and frequency capping; cache hot campaign state in Redis to absorb high-traffic reads; emit every impression and click onto Kafka as an event stream; process that stream in Python to aggregate engagement and enforce budgets; keep transactional campaign truth in PostgreSQL; land analytics in BigQuery for reporting; and run the whole fleet on Kubernetes with SLOs, monitoring, and graceful degradation. It composes distributed systems, event-driven architecture, microservices, data-intensive pipelines, caching, and operational stability into one coherent system — the kind of scalable ad infrastructure Reddit Ads runs.

Built step-by-step with [KhwajaLabs Build](https://khwajalabs.com).

## Stack
- Go
- Python
- Kafka
- Redis
- PostgreSQL
- BigQuery
- Kubernetes
- Docker
