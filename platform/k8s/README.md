# Kubernetes notes

Stateless workloads (adserver, spend-processor, cache-sync, analytics-sink) are
declared under `platform/k8s/`.

**Managed dependencies (recommended in production):**
- PostgreSQL: Cloud SQL / RDS / AlloyDB
- Redis: Memorystore / ElastiCache
- Kafka: Confluent Cloud / MSK / Redpanda Cloud
- BigQuery: optional analytics sink when `ANALYTICS_SINK=bigquery`

Do not embed stateful databases in these manifests for production. Point
`adplatform-config` / `adplatform-secrets` at managed endpoints and rotate
credentials via your secret manager.

Apply order:
```bash
kubectl apply -f platform/k8s/namespace-config.yaml
kubectl apply -f platform/k8s/adserver-deployment.yaml
kubectl apply -f platform/k8s/workers.yaml
```
