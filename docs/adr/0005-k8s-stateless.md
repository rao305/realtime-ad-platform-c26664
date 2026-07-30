# ADR 0005: Stateless K8s; managed data plane

## Status
Accepted

## Context
Running Kafka/Postgres/Redis as long-lived stateful sets in app manifests
invites operational debt for a capstone/demo.

## Decision
Kubernetes manifests cover only stateless services. Data stores are Compose for
local and managed services for production (documented in `platform/k8s/README.md`).

## Consequences
Clear boundary between app deploy and data plane. Local `docker compose up`
remains the full-stack path.
