# System overview

The live scanner has one authoritative market-state owner. Concurrent adapters
acquire and normalize bounded facts; the `ScannerStateEngine` applies them in
order, evaluates the population, and publishes immutable snapshots. The API and
dashboard consume those snapshots independently.

## Runtime data flow

```text
NYSE schedule + Massive reference REST
  -> immutable session binding

Massive WebSocket reader and single-pass decoder
  -> bounded decoded-batch FIFO with causal fences
  -> unbuffered batch handoff
  -> ScannerStateEngine owner loop
       ^
       |
Massive aggregate REST -> bounded hydration results

ScannerStateEngine
  -> compact canonical aggregates and coverage
  -> incremental qualification
  -> population selection at one-second cadence
  -> detailed enrichment for selected rows
  -> immutable combined publication

snapshot API v2 -> independent browser dashboard
```

Live events retain connection epoch, frame sequence, array index, and receipt
time. A complete decoded batch transfers to the engine owner and is consumed
sequentially. Every logical input receives its own sequence and disposition.
The production live path does not enqueue those facts into a second general
engine FIFO. Other typed admissions, including hydration and timers, execute
on the same owner.

## Canonical state and evaluation

Each symbol has a sealed session prefix and a mutable 16-minute correction
tail. The prefix retains sufficient session facts such as cumulative volume,
first open, extrema, latest sealed mark, and finalized qualification. The tail
retains correctable aggregates and exact coverage. Bounded indexes derive from
that canonical state and cannot become a competing market history.

Accepted aggregate changes update affected symbol state and qualification
windows incrementally. At an ordinary one-second evaluation boundary, the
engine scans compact population state, reconciles accounting, and selects up
to 20 qualified stocks by return from the adjusted prior close. It then computes
detailed dashboard measurements for those selected rows. All symbols retain
the aggregate evidence needed to enter the table without rebuilding history.

The default evaluation target is delayed four seconds from engine time to
allow ordered data processing. Readiness compares the committed watermark with
that causal target; its two-second tolerance is not a feed round-trip or
end-to-end trading latency measurement.

## Hydration and recovery

Startup resolves the session and reference binding, establishes aggregate
coverage, and hydrates the elapsed session while live ingestion continues.
Both launcher and direct scanner support 1, 2, 4, or 8 hydration workers and
default to 8. Workers perform bounded REST acquisition and normalization;
canonical acceptance, coverage, and recovery completion belong to the engine.

Historical and live aggregates share one identity and precedence policy.
Completed-empty results establish no-print evidence. A causal ingress fence
reconciles terminal hydration outcomes with already-ingested live work before
the engine can claim currentness. Same-process gap recovery uses this same
merge and reconciliation path. Process restart always starts with fresh
reference resolution and hydration.

## Publication and availability

Ordinary mutations coalesce into a combined publication at cadence. Trust
transitions publish promptly when necessary to prevent false-current output.
API capture reads one immutable publication and attaches bounded operational
status; it does not join independently mutable market views or perform market
calculations. HTTP and browser work cannot block the engine's mutation path.

Trade/quote enrichment never changes aggregate qualification, ranking, or
readiness. Pressure can shed or unsubscribe T/Q while preserving aggregate and
control processing. Aggregate-watermark staleness alone masks tape and spread
in the API without discarding subscriptions or continuously ingested T/Q state.
Visibility recovers only after five seconds of both elapsed recovery time and
watermark advancement; spread also requires a quote from the recovery period.

## Package ownership

| Package | Responsibility |
| --- | --- |
| `internal/session` | Embedded exchange schedule and session boundaries. |
| `internal/reference` | Eligible universe, adjusted prior closes, optional float, caches, and immutable binding. |
| `internal/massive` | Provider transport, single-pass decoding, bounded ingress, subscription delivery, and REST acquisition. |
| `internal/engine` | Ordered canonical mutation, coverage, qualification, selection, selected-row enrichment, and immutable publication. |
| `internal/operations` | Runtime composition, timers, recovery coordination, diagnostics, and coherent API capture. |
| `internal/snapshotapi` | API v2 validation/mapping, loopback HTTP, liveness, and readiness. |
| `internal/ui`, `ui/` | Independent static dashboard serving and browser presentation. |
| `internal/privatelauncher` | Local build/start/stop supervision and credential isolation. |

Replay and checkpoint packages have been removed. The API's fixed disabled
checkpoint object remains solely for schema compatibility.

## Detailed contracts

- [Reference data and session binding](reference-data-and-session.md)
- [Data, time, and event semantics](data-time-and-event-contract.md)
- [Live ingestion](live-market-data.md)
- [Hydration and recovery](hydration-and-recovery.md)
- [Engine lifecycle](scanner-state-engine-lifecycle.md)
- [Qualification and ranking](../features/qualification-and-ranking.md)
- [Trade/quote enrichment](../features/trade-quote-enrichment.md)
- [Snapshot API](../api/snapshot-v2.md)

The completed backend replacement's design records remain under
`docs/live-backend-replacement/`. They explain implementation decisions and
retain exact evidence; their completed delivery instructions are not a new
implementation workflow.
