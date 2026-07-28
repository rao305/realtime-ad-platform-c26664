# Service Level Objectives

| SLO                         | Target            | Why |
|-----------------------------|-------------------|-----|
| Serving availability        | 99.95% / 30 days  | Ad slots almost always get a timely answer |
| Decision latency (p99)      | ≤ 30 ms           | The latency budget, made measurable |
| Event pipeline freshness    | ≤ 5 s lag         | Pacing reacts to spend fast enough to matter |
| Spend accuracy              | exact             | We bill from it — no tolerance |

**Error budget:** 99.95% availability allows ~21.6 min of unavailability per
30 days. We spend that budget on risk (deploys, experiments). If we burn it,
we freeze risky changes and shore up reliability. The error budget turns
"be reliable" into a number we manage.
