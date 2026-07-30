# ADR 0001: PostgreSQL is truth; Redis is a rebuildable serving view

## Status
Accepted

## Context
The ad server must decide in ≤30ms. Hitting PostgreSQL on every request cannot
meet that budget under load, but billing and campaign configuration need ACID
semantics.

## Decision
- PostgreSQL stores campaigns, spend, and idempotency claims.
- Redis stores the derived serving view (campaign hashes + targeting indexes + pacing).
- `cache-sync` rebuilds Redis from Postgres on a short interval.

## Consequences
Serving can continue briefly on stale pacing if the processor is down.
Spend never lies: Redis is disposable and can be wiped/rebuilt.
