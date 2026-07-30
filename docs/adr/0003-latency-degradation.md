# ADR 0003: Latency degradation policies

## Status
Accepted

## Context
Partial dependency failure is normal. Blindly failing closed on every error
creates total outages; blindly failing open on targeting creates bad ads.

## Decision
| Failure | Policy |
|---------|--------|
| Targeting / eligible lookup | Fail closed → no ad |
| Frequency cap INCR | Fail open + metric |
| Kafka emit | Never block HTTP response |
| Decision deadline exceeded | Return no ad immediately |

## Consequences
Empty slots under Redis targeting outages. Possible slight frequency over-delivery
during Redis blips. Serving SLOs stay enforceable independently of the write path.
