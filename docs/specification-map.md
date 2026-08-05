# Specification map

**Status:** Approved Phase 1 specification index and owner-approved Phase 2
sequential roadmap.

**Phase 1 approved:** 2026-08-05

**Phase 2 roadmap approved:** 2026-08-05

This map identifies the authoritative Phase 1 documents and the plain component
sequence for Phase 2. It does not grant approval to a focused specification or
authorize implementation. The controlling work gates are in
[`implementation-process.md`](implementation-process.md).

## Authority

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved focused component specifications; and
4. implementation assignments, plans, tests, and code.

Within the approved architecture set, the
[`system overview`](architecture/system-overview.md) controls topology and
ownership, the
[`data, time, and event contract`](architecture/data-time-and-event-contract.md)
controls identity, clocks, ordering, coverage, and reconciliation, and the
[`Scanner State Engine lifecycle`](architecture/scanner-state-engine-lifecycle.md)
controls legal operational transitions. None may weaken the
[`product contract`](product/product-goals.md). If two approved documents
conflict substantively, work stops for the smallest owner decision; historical
drafting records are never tie-breakers.

This repository does not use an architecture decision record workflow.
Consequential component choices belong in the applicable focused component
spec. Repository-wide mechanical conventions belong in `AGENTS.md` or the
implementation process. Routine implementation choices are delegated within a
bounded assignment.

## Phase 1: approved product and architecture context

| Document | Current status | Role and controlling links |
| --- | --- | --- |
| [`product/product-goals.md`](product/product-goals.md) | Approved product contract, 2026-08-05 | Product promises, formulas, version 1 scope, priorities, and non-goals; highest authority. |
| [`architecture/system-overview.md`](architecture/system-overview.md) | Approved architecture contract, 2026-08-05 | Runtime topology, sole state ownership, component boundaries, failure containment, checkpoint/snapshot/API/UI boundaries, and focused-specification list. Depends on the product contract. |
| [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md) | Approved architecture contract, 2026-08-05 | Session binding, normalized-event meaning, half-open windows, causal order, live/REST merge, hydration evidence, committed watermark, replay order, and checkpoint cutoff. Depends on product behavior and system ownership. |
| [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md) | Approved architecture contract, 2026-08-05 | Legal live/replay lifecycle states and transitions, progress/exit conditions, publication permissions, recovery, suppression, session end, and shutdown. Depends on the product, system, and data/time contracts. |
| [`glossary.md`](glossary.md) | Approved shared vocabulary, 2026-08-05 | Concise definitions only. The controlling product or architecture contract wins if a summary is less specific. |

### Non-authoritative Phase 1 history

| Document | Current status | Role |
| --- | --- | --- |
| [`history/phase-1-decisions-brief.md`](history/phase-1-decisions-brief.md) | Historical drafting record; non-authoritative | Preserves design history only. |
| [`history/phase-1-consistency-review.md`](history/phase-1-consistency-review.md) | Accepted Phase 1 completion review; not an authority | Records the Phase 1 synchronization review; it does not control implementation. |

## Phase 2: approved procedural controls

These documents control how focused contracts and bounded implementations are
prepared and approved. They do not define scanner product behavior or compete
with Phase 1 architecture.

| Document | Current status | Role |
| --- | --- | --- |
| [`implementation-process.md`](implementation-process.md) | Owner-approved operational process, 2026-08-05 | Operational gates from targeted research through final validation. |
| [`specifications/focused-component-spec-template.md`](specifications/focused-component-spec-template.md) | Owner-approved mandatory template, 2026-08-05 | Required structure for each Phase 2 focused component contract. |

## Concerns already settled by Phase 1

Phase 1 already controls:

- the eligible universe, adjusted-prior-close basis, qualification gate and
  session latch, exact Day-%/symbol ordering, top-20 cardinality, and permanent
  `degraded_bootstrap` behavior;
- version 1 displayed fields, default displayed-row T/Q coverage, mandatory
  coherent checkpoints, aggregate-only replay, and independent UI deployment;
- one authoritative `ScannerStateEngine`, one canonical per-symbol state, one
  committed aggregate watermark, one ordinary evaluator, and fact-returning
  adapter/worker boundaries;
- the 04:00–20:00 New York session binding, half-open intervals, event/receipt/
  engine/replay time, source positions, epochs/generations, REST/live identity
  and precedence, no-print proof, snapshot identity, and checkpoint cutoff;
- legal initialization, bootstrap, live, recovery, replay, suppression,
  session-end, and shutdown transitions, including terminal hydration returning
  to ordinary evaluation; and
- aggregate/T/Q/checkpoint/API/UI/replay failure domains, global versus local
  integrity handling, mutually exclusive primary accounting, terminal work
  accounting, independent availability, and bounded observability.

Do not create `docs/specifications/core-domain-and-engine-boundaries.md` or an
equivalent shared layer. Focused specs cite exact Phase 1 requirement IDs and
define only component-specific ownership, interfaces, behavior, failures,
accounting, evidence, boundedness, and primary proof.

## Phase 2: focused component sequence

These are real focused component specs, not labels for new shared architecture.
Drafts belong under `docs/specifications/` and use the mandatory
[`template`](specifications/focused-component-spec-template.md). Their status
must remain explicit.

| Sequence | Focused component specification | What it completes and why it is next |
| --- | --- | --- |
| 1 | Reference data and exchange schedule/session binding | Establishes the immutable trading date, session bounds, eligible universe, required prior session, adjusted prior closes, and binding identity consumed by every live and replay run. |
| 2 | `ScannerStateEngine`, canonical symbol state, and feature-boundary implementation details | Establishes the sole ordered mutation path, canonical aggregate state, committed watermark, and immutable publication mechanism. It defines direct typed seams and ordered handling for the Phase 1-known aggregate, trade, quote, timer, connection/control, hydration-result, checkpoint-load/write-result, and T/Q-acknowledgement input classes without implementing their later component behavior or introducing a generic bus. It uses component 1 and does not finalize later feature, checkpoint-content, T/Q-measurement, readiness, or public API schemas. |
| 3 | Aggregate features, qualification, ranking, and accounting | Adds the Phase 1 formulas, correction-aware qualification, exact Day-%/symbol ordering, top-20 selection, independent availability, and exact population accounting to component 2's canonical path. |
| 4 | Aggregate replay | Adds the offline historical downloader/compiler, versioned normalized artifact, deterministic source, and injected simulated clock that drive components 1–3. Its Massive REST aggregate row normalizer is the one reused by component 6; offline download mechanics do not own live hydration, recovery, or terminal-work accounting. This completes the deterministic aggregate-core milestone. Full T/Q replay remains deferred. |
| 5 | Massive live adapter | Adds one bounded Stocks WebSocket adapter for A/T/Q/control classification, provider normalization, causal positions, epochs, commands, and acknowledgements. Aggregate/control integrates first; T/Q consumption remains component 9. Any live observation is narrow, question-driven, and separately authorized after v2 reconnaissance. |
| 6 | Aggregate REST hydration and recovery | Reuses component 4's Massive REST aggregate row normalizer and adds production pagination, request concurrency/retries, fresh hydration, checkpoint catch-up inputs, exact gap recovery, explicit terminal outcomes, ingress fencing, and REST/live reconciliation. It does not create a second REST aggregate mapping or replay path. |
| 7 | Checkpoints and restart | Defines coherent contents at `T0`, validation, atomic storage, cadence, restart installation, and `[T0,R)` catch-up after aggregate, feature, and recovery state are known. Checkpoints remain required in version 1. |
| 8 | Readiness and operations | Fixes evidenced aggregate/runtime evaluation, freshness, and capacity thresholds, bounded diagnostics, recovery budgets, readiness reasons, and shutdown policy around components 1–7. T/Q pressure thresholds remain component 9. This completes the production aggregate-lifecycle milestone. |
| 9 | Top-20 T/Q coverage and T/Q features | Adds selected-row subscription intent, acknowledgement-based coverage, Tape Rate, Spread, warm-up/gap behavior, and aggregate-protecting pressure degradation without changing aggregate ranking or readiness. |
| 10 | Versioned snapshot API | Freezes the public read-only schema after aggregate fields, T/Q fields, accounting, availability, and readiness meanings are stable. Internal structures do not automatically become the API. |
| 11 | Independent UI | Implements presentation and interaction against the approved versioned API and remains deployable without restarting the backend. |

Implementation is sequential. A focused component spec may divide a large
component into owner-approved sequential implementation slices without creating
new component authorities. Keep at most one active implementation slice. Each
slice delivers one coherent behavior and its allocated primary proof, then
stops for owner review before the next slice. While component N is being
implemented and proved slice by slice, specification work may advance for
component N+1. Final N+1 contract/reuse/test/slice-plan approval and
implementation normally wait until all N slices pass their proofs and final
component review. This is intentional: checkpoint contents follow known
feature/recovery state, the public API follows stable backend meanings, and the
UI follows the approved API.

One integration owner controls the concrete engine, normalized-event, snapshot,
and API boundaries. A later component may extend an approved boundary only
through its focused specification; it may not create a competing owner or
silently reinterpret the earlier contract.

## Vertical implementation milestones

1. **Deterministic aggregate core:** replay -> normalized aggregate ->
   `ScannerStateEngine` -> symbol state -> qualification/ranking -> snapshot.
2. **Production aggregate lifecycle:** live adapter + REST hydration/recovery +
   checkpoints + readiness.
3. **T/Q:** top-20 subscription + Tape Rate/Spread + aggregate-protecting
   pressure degradation.
4. **Product delivery and release validation:** versioned API + independent UI
   + release/cutover evidence against the readiness and operations policy
   established in milestone 2.

Each focused spec plans its evidence and primary proofs before implementation.
There is no up-front global evidence registry or stand-alone test-strategy
meta-spec. Final operations, authorized shadow observation, and cutover
validation assemble the reviewed component and integration evidence under the
[`implementation process`](implementation-process.md). Scanner agreement proves
correctness within its evidence limits; it does not prove trading edge or
executable expectancy.
