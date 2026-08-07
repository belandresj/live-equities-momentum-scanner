# Specification map

**Status:** Approved Phase 1 specification index and owner-approved Phase 2
sequential roadmap.

**Phase 1 approved:** 2026-08-05

**Phase 2 roadmap approved:** 2026-08-05

**C7-C11 unattended program approved:** 2026-08-07

This map identifies the authoritative Phase 1 documents and the plain component
sequence for Phase 2. It does not ordinarily grant approval to a focused
component contract or authorize implementation. The explicit C7-C11 standing
program authorization below is the sole exception and remains conditional on
the controlling gates in
[`implementation-process.md`](implementation-process.md#28-c7-through-c11-unattended-program).

## Authority

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved focused component contracts, whether stored in one file or
   as one indexed parent plus subordinate detail specs; and
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
| [`implementation-process.md`](implementation-process.md) | Owner-approved operational process, revised 2026-08-07 | Operational gates from targeted research through final validation, including the C7-C11 standing delegation, independent contract gates, local-commit policy, delegated evidence-based advancement, risk-triggered review, and tiered verification. |
| [`specifications/focused-component-spec-template.md`](specifications/focused-component-spec-template.md) | Owner-approved mandatory template, revised 2026-08-07 | Required compact/modular structure, approval authority and independent-review records, single parent delivery ledger, context routing, proof allocation, and slice rules for each Phase 2 focused component contract. |

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
equivalent shared layer. Focused component contracts cite exact Phase 1
requirement IDs and define only component-specific ownership, interfaces,
behavior, failures, accounting, evidence, boundedness, and primary proof.

## C7-C11 standing program decisions

The owner approved the completed Component 7 contract as written on 2026-08-07.
For Components 8-11, the future goal orchestrator has standing authority to
draft, independently review, correct, and record approval of each Phase 1
skeleton and completed contract without another owner message when the entire
gate conforms to Phase 1 and accepted dependencies, preserves component order
and ownership, has the required complete proof allocation, passes independent
review and any focused re-review, and has no substantive drift-audit `yes`.
This delegation changes the approval actor for those clean gates only; it does
not permit concurrent components or slices, early V2 reconnaissance, or a
post-approval contract change.

The program is constrained as follows:

| Scope | Standing decision |
| --- | --- |
| Release claim | Produce a completely reviewed and locally verified private release candidate through C11. Live validation, public deployment, and production cutover are not part of done. |
| Evidence and capacity | Use the current local host as the benchmark reference. Record fixture, host, run segmentation, and results; make no capacity claim beyond that evidence. No credentialed live-provider observation is authorized, and indispensable live evidence stops the program. |
| C8 readiness/operations | Select conservative evidence-backed thresholds, prioritizing aggregate correctness and avoiding false-ready publication. |
| C9 T/Q | Add no Tape Rate attention threshold; require continuous warm-up; restore T/Q in current rank order; permit complete T/Q shedding; protect aggregate processing first. |
| C10 API | Deliver a private versioned read-only HTTP API with explicit field status, publication identity, loopback binding, and an explicit CORS allow-list. Public authentication, TLS, hosting, and cutover are deferred. |
| C11 UI | Chrome desktop is required. The approved reconnaissance category is the V2 UI specification, UI code, assets, and focused UI tests within the exact narrow scope recorded by the C11 skeleton. Deliver a high-fidelity adaptation preserving useful layout, visual character, information density, and interactions except where they conflict with current product semantics, accessibility, independent deployment, or C10. Do not port V2 browser-owned calculations, readiness logic, obsolete state semantics, or backend coupling. |

The exact manual stops, one-write-capable-worker rule, independent skeleton and
completed-contract gates, and exact-path local-commit policy are authoritative
in [`AGENTS.md`](../AGENTS.md#c7-through-c11-unattended-program-authority) and
the [implementation process](implementation-process.md#28-c7-through-c11-unattended-program).

## Phase 2: focused component sequence

These are real focused component contracts, not labels for new shared
architecture. The path in this map is always the component's authoritative
entry-point specification. A small contract may remain in that one file; a
large contract may keep that parent compact and route cohesive low-level
details to subordinate specs under a same-named directory, following the
mandatory [`template`](specifications/focused-component-spec-template.md).
Subordinate specs remain part of the same component authority and do not add
sequence entries or independent approval gates. Contract status must remain
explicit in the parent.

| Sequence | Focused component specification | What it completes and why it is next |
| --- | --- | --- |
| 1 | [Reference data and exchange schedule/session binding](specifications/reference-data-and-session-binding.md) *(finally approved 2026-08-05 after repeated independent final review; S1–S3 and C1-R1–C1-R3 accepted; component not frozen)* | Establishes the immutable trading date, session bounds, eligible universe, required prior session, adjusted prior closes, and binding identity consumed by every live and replay run. |
| 2 | [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](specifications/scanner-state-engine-and-canonical-state.md) *(finally approved 2026-08-05 after S1–S4 implementation, required independent reviews, corrections, and the separate read-only final component review; no acceptance item remains open)* | Establishes the sole ordered mutation path, canonical aggregate state, committed watermark, and immutable publication mechanism. Its compact parent routes authority/reuse plus one implementation detail document for each approved S1–S4 behavior/proof boundary. It defines direct typed seams and ordered handling for the Phase 1-known aggregate, trade, quote, timer, connection/control, hydration-result, checkpoint-load/write-result, and T/Q-acknowledgement input classes without implementing their later component behavior or introducing a generic bus. It uses component 1 and does not finalize later feature, checkpoint-content, T/Q-measurement, readiness, or public API schemas. |
| 3 | [Aggregate features, qualification, ranking, and accounting](specifications/aggregate-features-qualification-ranking-and-accounting.md) *(finally accepted 2026-08-06 after C3-S1–S4, C3-R1/C3-R2, required focused reviews, complete verification, and the clean mandatory separate final component review; no acceptance item remains open)* | Adds the Phase 1 formulas, correction-aware qualification, exact Day-%/symbol ordering, top-20 selection, independent availability, and exact population accounting to component 2's canonical path. |
| 4 | [Aggregate replay](specifications/aggregate-replay.md) *(finally accepted 2026-08-06 after `C4-S1`–`C4-S4`, all nine primary proofs, cumulative build/test/race/vet verification, clean drift audit, and clean mandatory final read-only review; no acceptance item remains open)* | Adds the offline historical downloader/compiler, versioned normalized artifact, deterministic source, and injected simulated clock that drive components 1–3. Its Massive REST aggregate row normalizer is the one reused by component 6; Component 4 proves the mapper and its compiler consumer, while Component 6 later proves its own production consumer. Offline download mechanics do not own live hydration, recovery, or terminal-work accounting. This completes the deterministic aggregate-core milestone. Full T/Q replay remains deferred. |
| 5 | [Massive live adapter](specifications/massive-live-adapter.md) *(finally accepted 2026-08-06 after `C5-S1`–`C5-S3`, all eleven primary proofs, complete final-tier verification, corrected mandatory independent final review, clean focused re-review, and no remaining acceptance item)* | Adds one bounded Stocks WebSocket adapter for A/T/Q/control classification, provider normalization, causal positions, epochs, commands, and acknowledgements. Aggregate/control integrates first; T/Q consumption remains component 9. Any live observation is narrow, question-driven, and separately authorized after v2 reconnaissance. |
| 6 | [Aggregate REST hydration and recovery](specifications/aggregate-rest-hydration-and-recovery.md) *(finally accepted 2026-08-07 after `C6-S1`–`C6-S4`, all eleven primary proofs, complete final-tier build/test/race/vet verification, clean drift audit, and clean mandatory independent final review with focused re-review; no acceptance item remains open)* | Reuses component 4's Massive REST aggregate row normalizer and adds production pagination, request concurrency/retries, fresh hydration, checkpoint catch-up inputs, exact gap recovery, explicit terminal outcomes, ingress fencing, and REST/live reconciliation. It does not create a second REST aggregate mapping or replay path. |
| 7 | [Checkpoints and restart](specifications/checkpoints-and-restart.md) *(completed contract, exact V2 whitelist, eight proofs, three slices, required reviews, and `advancement_mode: delegated` owner-approved 2026-08-07; implementation has not begun)* | Defines coherent contents at `T0`, validation, atomic storage, cadence, restart installation, and `[T0,R)` catch-up after aggregate, feature, and recovery state are known. Checkpoints remain required in version 1. |
| 8 | Readiness and operations *(standing program authority; skeleton not started)* | Fixes evidenced aggregate/runtime evaluation, freshness, and capacity thresholds, bounded diagnostics, recovery budgets, readiness reasons, and shutdown policy around components 1–7. Thresholds must be conservative and evidence-backed, protect aggregate correctness, and avoid false-ready results. The local host is the only benchmark reference and supports no broader capacity claim. T/Q pressure thresholds remain component 9. This completes the production aggregate-lifecycle milestone. |
| 9 | Top-20 T/Q coverage and T/Q features *(standing program authority; skeleton not started)* | Adds selected-row subscription intent, acknowledgement-based coverage, Tape Rate, Spread, continuous warm-up/gap behavior, current-rank-order restoration, and aggregate-first pressure degradation. It adds no Tape Rate attention threshold, permits complete T/Q shedding, and cannot change aggregate ranking or readiness. |
| 10 | Versioned snapshot API *(standing program authority; skeleton not started)* | Freezes a private versioned read-only HTTP schema after aggregate fields, T/Q fields, accounting, availability, and readiness meanings are stable. It exposes explicit field status and publication identity, binds loopback, and uses an explicit CORS allow-list. Internal structures do not automatically become the API; public authentication, TLS, hosting, and cutover are deferred. |
| 11 | Independent UI *(standing program authority; skeleton not started)* | Implements a Chrome-desktop, high-fidelity adaptation of the useful V2 scanner UI against the approved C10 API and remains deployable without restarting the backend. Its narrow reconnaissance covers the V2 UI specification, code, assets, and focused tests. Useful layout, visual character, information density, and interactions are preserved unless they conflict with current product semantics, accessibility, independent deployment, or C10. Browser-owned calculations/readiness, obsolete semantics, and backend coupling are rejected. |

Implementation is sequential. Document decomposition and implementation slicing
are separate decisions: subordinate specs reduce contract context while slices
divide delivery at coherent behavior and proof boundaries. A focused component
contract may use either or both without creating new component authorities.
Keep at most one active implementation slice. Each slice delivers one coherent
behavior and its allocated primary proof, then passes a recorded delegated or
manual acceptance gate before the next slice. While component N is being
implemented and proved slice by slice, component N+1 may advance only through a
compact Phase 1 skeleton and boundary/reconnaissance-scope approval by the
applicable authority.
Version 2 reconnaissance, Sections 8–19, completed-contract approval, and N+1
implementation wait until all N slices pass their proofs and final component
review, unless the owner records an exact stable-interface exception. This is
intentional: the skeleton captures durable authority and ownership early while
checkpoint contents follow known feature/recovery state, the public API follows
stable backend meanings, and the UI follows the approved API.

One integration owner controls the concrete engine, normalized-event, snapshot,
and API boundaries. A later component may extend an approved boundary only
through its focused component contract; it may not create a competing owner or
silently reinterpret the earlier contract.

Each component parent owns its authoritative slice-delivery ledger. This map
records only coarse component sequence and completion and is updated when a
component completes; subordinate specs do not duplicate mutable acceptance
state.

## Vertical implementation milestones

1. **Deterministic aggregate core:** replay -> normalized aggregate ->
   `ScannerStateEngine` -> symbol state -> qualification/ranking -> snapshot.
2. **Production aggregate lifecycle:** live adapter + REST hydration/recovery +
   checkpoints + readiness.
3. **T/Q:** top-20 subscription + Tape Rate/Spread + aggregate-protecting
   pressure degradation.
4. **Private product-delivery release candidate:** versioned API + independent
   Chrome-desktop UI + local release-candidate evidence against the readiness
   and operations policy established in milestone 2. Public hosting, live
   validation, and production cutover remain deferred.

Each focused component contract plans its evidence and primary proofs before
implementation.
There is no up-front global evidence registry or stand-alone test-strategy
meta-spec. Final operations, authorized shadow observation, and cutover
validation assemble the reviewed component and integration evidence under the
[`implementation process`](implementation-process.md). For the C7-C11 program,
final validation ends at the private/local release candidate recorded above;
credentialed shadow observation and public cutover are not authorized. Scanner agreement proves
correctness within its evidence limits; it does not prove trading edge or
executable expectancy.
