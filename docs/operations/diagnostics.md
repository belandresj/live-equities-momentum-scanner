# Operational diagnostics

Diagnostics explain currentness, pressure, accounting, and bounded failures
without exposing provider credentials, URLs, symbols, or raw payloads. They are
evidence for debugging, not a second source of scanner state.

## First questions

When the scanner appears wrong or slow, separate these conditions first:

1. Is the process live?
2. Is the aggregate backend ready, and if not, what lifecycle/readiness reason
   is published?
3. Is the committed watermark advancing relative to the causal target?
4. Is hydration/recovery work closed and its ingress fence reconciled?
5. Are population, admission, transition, queue, and publication identities
   valid?
6. Is the problem aggregate-wide or limited to T/Q fields?
7. Is pressure caused by queue slots, queued bytes, oldest unread age,
   watermark lag, capacity loss, retention, or accounting?

`/livez`, `/readyz`, and `/api/v2/snapshot` answer these from one capture path.
Do not infer backend readiness from the dashboard process or T/Q cells alone.

## Important measurements

| Measurement | Interpretation |
| --- | --- |
| Committed `T` and causal target | Market-time progress and currentness boundary. |
| Watermark lag | Aggregate evaluation lag, not WebSocket round-trip latency. |
| Waiting frames/bytes and oldest age | Decoded-batch backlog and service deficit. |
| Frame/event dispositions | Whether every classified input reached a closed outcome. |
| Hydration work/rows/fence | Whether historical acquisition and live-tail reconciliation completed. |
| T/Q pressure mode and cause | Subordinate enrichment containment state. |
| Retained T/Q identities | Bound health; not a measure of ranking quality. |
| Publication and evaluation counts | Whether accepted facts become observable and whether projection is being coalesced. |

A zero final queue does not prove adequate burst service if the queue grew and
drained later. Compare offered duration, wall-clock input rate, service rate,
maximum occupancy, growth rate, drain time, and all terminal accounting.

## Bounded incident files

The scanner may write bounded local incidents under `var/diagnostics` (or the
explicit diagnostic directory) for invariant failures and selected offline/live
observations. Incident schemas use closed reasons and bounded numeric evidence.
They must not contain credentials, provider endpoints, raw frames, or an
unbounded symbol list.

Historical incident and performance interpretations live under
`docs/history/`. Those reports describe the exact fixture and host claim they
measured; they are not current architecture authority.

## Offline performance work

Use deterministic local providers and explicit opt-in tests. Validate frame
count, event mix, interval, queue capacities, normalized work, and accounting
before timing. Capture CPU profiles and execution traces only for bounded runs.
Separate fake-server goroutines from scanner goroutines and distinguish
normalization time, queue wait, engine owner wait, state mutation, maintenance,
and publication.

The [implementation status](../current-state.md) distinguishes the retained
extended-hours observation from deferred deterministic capacity measurement.
Older baseline throughput measurements do not establish capacity for the
replacement backend.

## Claim limits

- Unit and fake-provider tests prove only the exercised semantics.
- A local capacity trial proves only that fixture, host, build, and duration.
- A live observation proves provider behavior only for its authorized time and
  conditions.
- None of these proves predictive edge, fill quality, costs, or executable
  expectancy.

Never run credentialed diagnostics or market-hours trials without explicit
authorization for that exact run.
