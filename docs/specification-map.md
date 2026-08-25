# Specification map

**Status:** Approved authority index; the 2026-08-23 live backend replacement
is the current sequential delivery program, owner-revised on 2026-08-24 to
insert bounded parallel hydration `LBR-A3` and the accepted narrow watermark-
diagnostic preservation correction before D1.

**Phase 1 approved:** 2026-08-05

**Product feature-set revision approved:** 2026-08-14

**Day-%/From-Open presentation, header-tooltip, compact dashboard-structure,
trader-facing status, and compact Rank-column revisions approved:** 2026-08-17

**Live feature-set MVP program approved:** 2026-08-14

**Live backend replacement architecture and delivery program approved:**
2026-08-23; E2 owner-revised to exactly one 10-minute deterministic run on
2026-08-23; bounded `1|2|4|8` live hydration restored on 2026-08-24

**Phase 2 sequence approved:** 2026-08-05

**Version 1 Release Program approved:** 2026-08-07; replaces the former
C7-C11 unattended program

**C12 follow-on sequence approved:** 2026-08-09; retained as historical
authority and not active under the live backend replacement

This map identifies authoritative documents, component entry points,
dependencies, and coarse completion state. Current replacement delivery state
lives in the
[`live backend replacement program`](live-backend-replacement/delivery-program.md);
historical component evidence remains routed through each component parent.

## Authority

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved delivery-program authority for its stated scope;
4. focused component contracts; and
5. implementation assignments, plans, tests, fixtures, benchmarks, and code.

For the replacement scope, the
[live backend replacement architecture](live-backend-replacement.md) controls
live topology, ownership handoffs, state shape, evaluation, resource policy,
and removal. The approved
[data/time/event contract](architecture/data-time-and-event-contract.md) and
[engine lifecycle](architecture/scanner-state-engine-lifecycle.md) remain
compatible semantic evidence until their live requirements are routed into the
five focused replacement contracts. None may weaken the
[product contract](product/product-goals.md).

When a lower-level artifact conflicts with these authorities, revise the lower-
level artifact. For C7-C11, apparent Phase 1 tension uses the V1 program's
authority order, strict compatible intersection, simplest design, and honest
unavailable-output rule without an in-goal owner question. Historical drafting
records are never tie-breakers.

## Phase 1 approved product and architecture

| Document | Status and role |
| --- | --- |
| [`product/product-goals.md`](product/product-goals.md) | Approved product contract, 2026-08-05 and owner-revised 2026-08-14; highest product authority. The revised feature set supersedes conflicting lower-level formulas and displayed-field contracts pending sequential reconciliation. |
| [`live-backend-replacement.md`](live-backend-replacement.md) | Approved 2026-08-23 and owner-revised 2026-08-24 current replacement architecture; controls one state owner/handoff, bounded parallel hydration, compact state, two-phase evaluation, resource policy, and deletion boundary. |
| [`architecture/system-overview.md`](architecture/system-overview.md) | Historical approved architecture and compatible evidence; superseded for conflicting replacement topology, state, queue, evaluation, replay/checkpoint-live-core, and resource decisions. |
| [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md) | Approved semantic architecture and compatible evidence; focused replacement specs route retained live session, identity, ordering, merge, coverage, and watermark meaning. Replay/checkpoint sections are non-gating. |
| [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md) | Approved lifecycle semantics and compatible evidence; focused replacement specs route retained live startup, hydration, recovery, currentness, suppression, session-end, and shutdown meaning. Replay/checkpoint states are non-gating. |
| [`glossary.md`](glossary.md) | Approved vocabulary; controlling product/architecture text wins when more specific. |

Phase 1 and the 2026-08-14 owner revision settle the eligible universe,
prior-close basis, unchanged qualification, exact qualified Day-% ranking/top
20, revised displayed fields, one engine/state/watermark/evaluator, 04:00-20:00
New York session, half-open windows, REST/live identity and precedence,
no-print proof, lifecycle, failure domains, accounting, API/UI ownership, and
independent availability. Historical checkpoint and replay architecture remains
semantic evidence only; its implementation has been removed from the supported
scanner and is not a current operating or acceptance path. Focused contracts
cite retained live meanings; they do not create a shared replacement layer.

## 2026-08-14 product feature-set revision

The owner replaced the prior displayed feature set with:

```text
RANK | SYMBOL | FLOAT | VOLUME | LAST || FROM CLOSE % | FROM OPEN % | DAY RANGE ||
ACTIVITY 30s | MOVE 30s || TAPE SPEED | SPREAD
```

The revision retains the existing aggregate qualification latch, ranks
qualified passers strictly by Day % descending with exact-symbol tie-breaking,
and preserves selected-row T/Q independence. It adds public Float, cumulative
session share Volume, a five-minute-local share-volume Activity percentile,
aggregate-mark Move 30s with known-no-print carry, and an all-symbol minimum
330-second aggregate/coverage-evidence guarantee. It removes HOD drawdown,
30/60-minute ranges, the transaction/price-expansion Activity composite, and
the one-second Tape burst from the product.

The accepted pre-revision component evidence remains historical baseline
evidence. The completed live-feature MVP/API/UI/stability work established the
current product baseline. The live backend replacement now reorganizes only
the backend implementation around compact state, incremental evaluation,
bounded T/Q, and one ingress handoff; it preserves the approved product and
finished UI. Offline replay and checkpoint persistence remain outside the
supported path.

Historical files under [`history/`](history/) are non-authoritative drafting
and review records.

## Operational authorities

| Document | Status and role |
| --- | --- |
| [`live-backend-replacement/delivery-program.md`](live-backend-replacement/delivery-program.md) | Current owner-approved delivery authority, owner-revised 2026-08-24: Capabilities A–D and `LBR-E1` accepted; `LBR-E2` is next permitted but inactive; sole ledger, bounded execution, and hard integrated acceptance. |
| [`live-feature-mvp-program.md`](live-feature-mvp-program.md) | Historical accepted 2026-08-14 feature/API/UI/stability delivery evidence; superseded as the active implementation program. |
| [`live-evaluation-cycle-coalescing-correction.md`](live-evaluation-cycle-coalescing-correction.md) | Accepted historical scanner-stability evidence: one fence-owned evaluation opportunity, maintenance-only following timer, and coalesced missed ticks. Its old evaluator implementation is not replacement authority. |
| [`tq-publication-coalescing-correction.md`](tq-publication-coalescing-correction.md) | Accepted historical scanner-stability evidence: synchronous canonical T/Q mutation with one-second combined publication and immediate trust transitions. Its old T/Q representation is not replacement authority. |
| [`snapshot-capture-and-dashboard-transport-isolation-correction.md`](snapshot-capture-and-dashboard-transport-isolation-correction.md) | Accepted reusable API-isolation evidence: atomic engine capture, cached diagnostics, cancellation containment, and dashboard/API/provider failure distinction. |
| [`live-watermark-stall-diagnostic.md`](live-watermark-stall-diagnostic.md) | Accepted focused diagnostic correction and replacement forward-port after A3: bounded active/ordinary-cycle evidence and one private ready-to-`watermark_stale` incident record; diagnostic-only, no readiness or market-semantics change. The 2026-08-24 incident is baseline history; no replacement provider run is claimed. |
| [`live-aggregate-heartbeat-and-resubscription-correction.md`](live-aggregate-heartbeat-and-resubscription-correction.md) | Accepted focused correction, 2026-08-17: ordered one-at-a-time aggregate attempt retirement, individually paced finite recovery, preserved redacted handshake facts, inbound-aware heartbeat evidence, and startup/post-live return to current only through the existing hydration fence. Deterministic acceptance and final read-only review passed; credentialed provider chronology remains unobserved and separately authorized. |
| [`v1-release-program.md`](v1-release-program.md) | Accepted 2026-08-07 authority and evidence for the former C7-C11 field set. Its conflicting delivery gates are superseded; still-compatible semantic and proof decisions remain usable evidence. |
| [`implementation-process.md`](implementation-process.md) | Owner-approved process, revised 2026-08-23: contract-first replacement planning, correction/reopening, bounded verification, risk-based review, and integrated validation. |
| [`specifications/focused-component-spec-template.md`](specifications/focused-component-spec-template.md) | Mandatory template, revised 2026-08-07: compact/modular routing, current-plan and correction records, single parent ledger, proof and slice rules. |
| [`market-hours-validation.md`](market-hours-validation.md) | Procedure approved; execution pending a separate owner authorization. Not credential authority by itself. |
| [`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md) | Accepted owner-requested scanner-recovery baseline. Deterministic S1/S2, Gates A-E, the local D4 fence/capacity correction, and the D5 REST/live-precedence correction are accepted locally. D1-D4 observations remain evidence; another provider retry requires separate exact authorization. |
| [`live-tq-resilience-correction.md`](live-tq-resilience-correction.md) | Accepted 2026-08-14 T/Q resilience correction transplanted onto clean main: asynchronous status correlation, T/Q-local quarantine/accounting, exact recovery for possible aggregate loss, and engine-scheduled same-binding continuation pass focused and repository-wide deterministic proofs. Later D8 queue-pressure constants remain authoritative. Credentialed live confirmation is owner-run and not yet claimed. |
| [`live-ingress-first-cause-diagnostic.md`](live-ingress-first-cause-diagnostic.md) | D1 identified generic queue capacity after clean hydration; D2 made capacity causes decisive; D3 corrected the localized-conflict population transition and added its reason ledger. The D3 retry completed hydration but selected a long ingress-fence stall and slot saturation before retaining that ledger. |
| [`live-fence-finalization-and-burst-capacity-correction.md`](live-fence-finalization-and-burst-capacity-correction.md) | Accepted D4 correction: retained-tail full-universe evaluation now remains below the fixed responsiveness limit; terminal incidents preserve bounded last-coherent D3/accounting evidence; and owner-selected 32,768-slot retry headroom remains under the unchanged 128-MiB byte cap with fail-closed controls. |
| [`live-checkpoint-hot-path-correction.md`](live-checkpoint-hot-path-correction.md) | Finally accepted C7/C8 correction: bounded FIFO-tail projection continuations replace the multi-second monolithic owner hold, alias-isolated per-symbol transfer prevents retained writer aliases, checkpoint terminals return to engine accounting, O(1) continuation occupancy preserves saturated ingress cost, and the bounded decoder/restart objective passes; focused final re-review is clean. |
| [`v1-release-follow-on-goal.md`](v1-release-follow-on-goal.md) | Historical handoff prompt for the completed former C7-C11 program; not current execution authority. |
| [`replay-rest-feasibility-benchmark.md`](replay-rest-feasibility-benchmark.md) | Completed non-authoritative C12 acquisition evidence record and separately gated rerun protocol. It authorizes no credentials, provider request, production change, rerun, or capacity claim. |
| [`c12-implementation-goal.md`](c12-implementation-goal.md) | Historical replay follow-on handoff. Its retained implementation is unverified and non-gating for the live backend replacement. |
| [`replay-warmup-acceleration.md`](replay-warmup-acceleration.md) | Owner-authorized 2026-08-14 focused replay correction: accelerate cached 04:00-to-observation-start warm-up, preserve exact logical-boundary equivalence, then use the unchanged automatic 1x API/dashboard path. Separate from and non-gating for the live MVP. |

## Current delivery program

The
[`live backend replacement delivery program`](live-backend-replacement/delivery-program.md)
supersedes conflicting historical implementation sequences and private
representations for its scope. Keep one write-capable slice and finally review
each capability. `LBR-P1` and A1/A2/B/C evidence remain accepted. The
2026-08-24 owner revision reopened Capability A only for bounded parallel live
hydration; A3 is now accepted. The corrected watermark-stall diagnostic was
then semantically forward-ported without reopening A/B/C. D1/D2 and their
Capability D review are accepted. `LBR-E1` completed the exclusive live cutover
and owner-selected replay/checkpoint deletion; `LBR-E2` is next permitted but
inactive.

The active capability order is:

1. canonical state and hydration, including owner-inserted `LBR-A3` bounded
   parallel acquisition before D1;
2. incremental evaluation and immutable publication;
3. compact selected-row T/Q under the approved 30-second late/duplicate policy;
4. single-pass Massive ingress and one decoded-batch handoff; and
5. exclusive cutover, deletion, deterministic resource/stability acceptance,
   then separately authorized live confirmation.

The accepted live-feature MVP, scanner-stability corrections, numbered
components, and V1 program are reusable evidence. They cannot preserve a
superseded feature, queue, representation, evaluator, replay/checkpoint-live-
core path, or proof gate. The API v2 and finished UI meanings remain fixed.

Numeric resource values in the parent are design targets. E2 executes exactly
one 10-minute deterministic acceptance run; a hard-conforming target miss is
recorded as a deviation without another timed trial. Only correctness/loss,
unbounded resource growth, sustained backlog/readiness failure, or unusable
one-second API/dashboard behavior blocks completion. No optional, fallback,
diagnostic, host-coexistence, or non-gating repeat run exists.

## Historical Phase 2 focused component sequence

The path in this table is the authoritative component entry point. A modular
component keeps its parent at this path and routes details beneath a same-named
directory. Subordinate specs do not create new components or mutable delivery
ledgers.

| Sequence | Focused component | Outcome and current state |
| --- | --- | --- |
| 1 | [Reference data and session binding](specifications/reference-data-and-session-binding.md) | Finally accepted 2026-08-05. Owns immutable trading date/session bounds, eligible universe, required prior session, adjusted prior closes, and binding identity. |
| 2 | [ScannerStateEngine and canonical state](specifications/scanner-state-engine-and-canonical-state.md) | Finally accepted 2026-08-05. Owns sole ordered mutation, canonical aggregate state, committed watermark, immutable publication, and typed extension seams. |
| 3 | [Aggregate features, qualification, ranking, and accounting](specifications/aggregate-features-qualification-ranking-and-accounting.md) | Finally accepted 2026-08-06. Owns formulas, correction-aware qualification, exact order/top 20, independent availability, and population accounting. |
| 4 | [Aggregate replay](specifications/aggregate-replay.md) | Historical accepted evidence for former aggregate fields. The replacement does not depend on or claim retained replay operation. |
| 5 | [Massive live adapter](specifications/massive-live-adapter.md) | Finally accepted 2026-08-09 after the integrated V1 RC reopened and corrected blocked-dequeue T/Q acknowledgement/deadline correlation with clean focused re-review. Owns bounded A/T/Q/control classification, normalization, causal positions, epochs, commands, and acknowledgements. |
| 6 | [Aggregate REST hydration and recovery](specifications/aggregate-rest-hydration-and-recovery.md) | Finally accepted 2026-08-07. Owns production pagination/workers, fresh/checkpoint/gap plans, exact terminal outcomes, ingress fencing, and REST/live reconciliation. |
| 7 | [Checkpoints and restart](specifications/checkpoints-and-restart.md) | Historical accepted evidence for the former state schema. Checkpoints are excluded from the supported replacement live core; fresh hydration remains the restart path. |
| 8 | [Readiness and operations](specifications/readiness-and-operations.md) | Finally accepted 2026-08-07 and corrected with clean focused re-review 2026-08-10 after live-start analysis invalidated the whole-hydration 60-second deadline. Connection establishment remains bounded; finite C6 hydration continues under per-request bounds while the subscribed live tail is consumed; readiness waits for the exact ingress fence; concurrent terminal diagnostics and shutdown joins are race-safe. Runnable composition, honest readiness/staleness, engine-owned bounded recovery/exhaustion, fixed-cardinality measurements, and controlled 6,000-symbol mixed load remain accepted. |
| 9 | [Top-20 T/Q coverage and features](specifications/top-20-tq-coverage-and-features.md) | Reaccepted after the owner-directed waiting-pressure/frame-local correction. Ordinary live configuration targets all ranked displayed rows up to 20 through serialized paired acknowledgements. Global pressure uses waiting frame/byte backlog, separate capacity/accounting/bound loss, and guarded aggregate lag; active-frame age is diagnostic only, with a 500-ms frame-local T/Q budget preserving later aggregates/controls. Spread retains the latest valid numeric quote with age through quiet coverage. Required deterministic, ordinary, affected race, vet, and diff gates pass; live/provider capacity remains unclaimed. |
| 10 | [Versioned snapshot API](specifications/versioned-snapshot-api.md) | Finally accepted 2026-08-08. One sealed immutable capture maps to the exact versioned schema; loopback HTTP, exact-origin CORS, liveness/readiness, bounded transport, and joined scanner composition pass with clean focused final re-review. |
| 11 | [Independent UI](specifications/independent-ui.md) | Finally accepted 2026-08-09. Production Chrome visual/interaction/accessibility/independence proof, final correction review, and integrated private V1 RC review are clean. Independent Chrome-desktop UI has every V1 field/status and no browser-owned market logic. |
| 12 | [Historical replay product mode](specifications/historical-replay-product-mode.md) | Historical, incomplete follow-on work. Its runnable capability is unknown and it is not a replacement acceptance gate. |

The numbered component contracts remain reusable ownership, semantic, fixture,
and proof evidence. Current replacement implementation follows `LBR-P1`, then
Capabilities A-E. A historical component does not control the new private
state shape, package boundary, queue, proof allocation, or cleanup decision.

## Historical V1 vertical milestones

1. **Deterministic aggregate core:** C1-C4, already accepted.
2. **Production aggregate lifecycle:** C5-C8, including corrected checkpoint
   restart, runnable readiness/recovery, and local operational evidence.
3. **T/Q enrichment:** C9, including normal top-20 coverage and complete
   aggregate-protecting shedding/restoration.
4. **Private/local V1 RC:** Accepted 2026-08-09. C10-C11, loopback API,
   independent Chrome UI, deterministic production-path integrated evidence,
   and the final read-only review are clean.

These milestones describe the accepted former feature set. They do not prove
the revised product, replay compatibility, or checkpoint compatibility.

The replacement uses deterministic fixtures for semantics and resource work,
but the only supported operating claim remains the fresh-start live scanner.
Replay is not a fallback acceptance path. Any credentialed market-hours
observation still requires the separate authorization described in
[`market-hours-validation.md`](market-hours-validation.md). Scanner correctness
does not prove trading edge or executable expectancy.
