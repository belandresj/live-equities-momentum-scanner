# Evaluation and publication

**Status:** Owner-approved focused replacement specification, 2026-08-23. The
[delivery program](delivery-program.md) is the sole mutable status ledger.

**Parent:** [Live backend replacement architecture](../live-backend-replacement.md).

## 1. Outcome and authority

This contract replaces full-population history evaluation with incremental
qualification, one compact selection scan, selected-row enrichment, and one
immutable combined publication. It owns the aggregate watermark, selection,
current-product aggregate fields, aggregate accounting, readiness-bearing
publication facts, and snapshot replacement. It does not own canonical merge,
T/Q retention/membership, provider ingress, API representation, or UI logic.

Allocated parent requirements are `LBR-ARCH-04`, `LBR-ARCH-08`, and
`LBR-ARCH-09`. Allocated retained product semantics are `PG-RANK-01`,
`PG-RANK-03`, `PG-RANK-04`, `PG-RANK-05`, `PG-FEATURE-01`,
`PG-FEATURE-02`, `PG-FEATURE-03`, `PG-FEATURE-04`, `PG-AVAIL-01`,
`PG-AVAIL-02`, `PG-OBS-01`, and `PG-OBS-03` from
[`product-goals.md`](../product/product-goals.md). Product formulas and wire
fields remain defined there and in the accepted API contract; this document
allocates their evaluation and publication implementation once.

The compatible historical semantics routed here are `DTE-CLOCK-04` through
`DTE-CLOCK-06`, the system-position portion of `DTE-EVENT-04`,
`DTE-TIMER-01`, `DTE-COMMIT-01` through `DTE-COMMIT-04`, and `DTE-REJECT-02` from the
[data/time/event contract](../architecture/data-time-and-event-contract.md),
plus `LIFE-LIVE-01` through `LIFE-LIVE-03`, `LIFE-LIVE-05`, and
`LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, `LIFE-END-01` through
`LIFE-END-03`, and `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03` from the
[engine lifecycle](../architecture/scanner-state-engine-lifecycle.md).
`LIFE-LIVE-04` and replay/checkpoint publication meanings are not routed.

## 2. Boundary and dependencies

This capability depends only on the accepted
[`canonical-state-and-hydration.md`](canonical-state-and-hydration.md)
read-only views and mutation notifications. It defines the selected-symbol and
publication seams consumed by [`tq-state.md`](tq-state.md); it does not depend
on the T/Q implementation. Until Capability C is accepted, a bounded adapter
may supply the existing T/Q projection, but aggregate selection and readiness
cannot branch on that adapter.

| Input | Owner | Use |
| --- | --- | --- |
| `SelectionStateView` and affected-proof notifications | canonical contract | Incremental proof maintenance and compact population scan. |
| `SelectedAggregateView` | canonical contract | Exact as-of aggregate fields for selected rows only. |
| engine time, session end, continuous aggregate coverage, hydration/fence facts | sole engine owner | Candidate target, watermark, lifecycle, field trust, and readiness. |
| `SelectedTQView` and T/Q trust revision | T/Q contract | Enrichment join and immediate false-current closure only; never ranking or readiness. |

Outputs are an ordered desired-symbol view for T/Q, one sealed internal
publication, and the existing `operations.SnapshotCaptureView`/API-v2 mapping
source. Callers receive immutable values or detached copies; no API, metrics,
or UI caller can request an evaluation or mutate publication members.

Reuse evidence is limited to the current repository's
[`qualification/corrections`](../specifications/aggregate-features-qualification-ranking-and-accounting/qualification-and-corrections.md),
[`ranking/availability`](../specifications/aggregate-features-qualification-ranking-and-accounting/ranking-availability-and-delivery.md),
[`population accounting`](../specifications/aggregate-features-qualification-ranking-and-accounting/population-accounting.md),
current-product feature fixtures, and the
[`snapshot capture correction`](../snapshot-capture-and-dashboard-transport-isolation-correction.md).
They supply semantic oracles and concurrency counterexamples only. Old feature
mathematics, full-population implementation, ledgers, replay/checkpoint paths,
and private structs are rejected. No predecessor checkout is inspected.

## 3. Incremental qualification

Qualification state belongs to each canonical symbol slot but its transition
rules belong here. Accepted aggregate mutation, correction, coverage change,
or timer may update only proof windows whose endpoint can be affected. Each
symbol retains bounded provisional proof evidence and its strict finalization
state; routine one-second work neither clones a qualification map nor scans the
symbol's session history.

The approved gate, proof-end timing, revocation, and finalization meanings are
referenced from `PG-RANK-03`. In particular, a mutable proof may be revoked by
an accepted correction in its affected interval, and finalization occurs only
after the complete correction exposure required by the product/data-time
contract. A first later trusted print can make a passing symbol rankable; a
timer can advance quiet proof/finalization state but cannot invent a print or
coverage.

Qualification state exposes fixed scalars for the selection scan: valid-prior
eligibility, qualification class, rankable mark/status, mark time, Day-%,
canonical revision, and field-trust summaries. It never exposes mutable maps.

## 4. One causally supported cycle

The engine schedules at most one ordinary full-population cycle for a unique
`(binding, canonical revision, candidate T)` per supported second. Candidate
`T`, the fixed semantic delay, continuous-coverage requirement, and committed
watermark advancement remain those of the routed contracts. An ingress fence
and immediately following timer cannot execute the same cycle twice.

The cycle has two phases over one coherent preselection revision:

1. **Selection.** Scan the fixed symbol array once. Classify each valid-prior
   symbol into the primary population and qualification identities, validate
   its trusted rankable mark, calculate approved Day-%, and retain at most 20
   rows with the exact Day-% descending/symbol ascending order. The retained
   heap or fixed array may be implementation-private, but ties and fewer-than-
   20 results are exact.
2. **Enrichment.** For those retained symbols only, read the canonical as-of
   values for Volume, From Open, Day Range, Activity 30s, and Move 30s; join
   Float and the latest selected T/Q view; preserve independent field states;
   validate publication identities; and seal one snapshot.

The selection scan may count premaintained fixed field-status families for the
complete population, but it cannot evaluate discarded display fields or walk
aggregate histories. Aggregate-derived sufficient state is maintained while
a symbol is unselected. Therefore a symbol entering the table after more than
330 seconds has the same aggregate field values and trust states it would have
had if continuously displayed; only Tape and Spread may warm after selection.

## 5. Accounting, failure, and trust

Every cycle validates the product's primary population, qualification, and
work identities before publication. The categories stay mutually exclusive;
overlapping field-status dimensions remain separately labeled. No-print,
below-price, invalid mark, unresolved coverage/fence, not-yet-qualified,
provisional, finalized, and unresolved are not collapsed into a generic
missing count.

The smallest false success is a snapshot labeled current after joining values
from different canonical revisions, an unsupported `T`, invalid accounting,
or a closed coverage/fence. The engine captures one revision, verifies it has
not changed while applying the owner-local result, and publishes only from the
accepted cycle. A correction or trust transition invalidating the current
projection publishes a noncurrent/field-invalid state immediately; ordinary
value changes coalesce into the next cadence.

A field-local invalid or unavailable input affects that field unless the
product requires the symbol to be unrankable. T/Q absence, pressure, or control
failure never changes aggregate ranking, aggregate watermark, or backend
readiness. A global clock, binding, coverage, accounting, or publication
identity failure closes currentness and follows the parent containment rule.

## 6. Immutable publication and capture

One engine-owned publication sequence advances whenever a new complete
snapshot replaces the atomic cell. A publication contains one binding/session
identity, lifecycle/currentness, committed `T`, canonical revision, ordered
rows, field statuses, T/Q projection/revision, accounting, recovery/fence
facts, and bounded operational facts required by API v2. Its internal members
cannot be mutated after the atomic store.

API capture performs exactly one atomic publication load and joins no later
market or T/Q read. It takes no engine mutation lock, runs no evaluation,
samples no checkpoint state, and calls no `runtime.ReadMemStats`. Nonmarket
diagnostics may come from the existing independently sampled immutable cache
and cannot override publication truth or backend readiness. Slow/canceled
handlers do not serialize capture or engine work.

Ordinary publication is at most the successful one-second cycle cadence.
Immediate replacement is limited to a transition that would otherwise leave
false currentness, coverage, field validity, lifecycle, or terminal state
visible. Same-`T` trust transitions receive a new publication ID. Publication
coalescing counters are diagnostic and cannot delay an immediate closure.

## 7. Bounds and resource allocation

Qualification evidence per symbol, selection scalars, population counters,
and field-status counters are fixed-cardinality. A cycle allocates no
population-sized maps or aggregate-history copies and retains at most 20 row
builders. Selected enrichment reads at most the parent windows required by the
current product; it does not materialize a second history.

The Capability B local target is the parent selection/enrichment target:
p99 below 100 ms and maximum below 250 ms on the characterized mature fixture,
with steady cycle allocations low enough to support the parent allocation-rate
target. These are design targets governed by the parent's single bounded
profile/correction/rerun policy; semantic correctness, bounded plateau, no
recurring readiness flap, and usable one-second polling are hard.

## 8. Primary proofs and slices

| Slice | Primary proof | Claim, dangerous counterexample, observable distinction, limitation |
| --- | --- | --- | --- |
| `LBR-B1` | `P-LBR-B1-SELECTION` | A deterministic engine trace covers the approved qualification gate, multiple provisional proofs, correction revocation at every strict/equality boundary, finalization, first later print, quiet timers, fewer-than-20, exact ties, invalid prior/mark, incomplete population, and T/Q/display-field absence. It compares qualification classes, Day-%, order, watermark, and primary accounting with the approved semantic oracle and counts full-population cycles. It detects cloned/rescanned proof state and T/Q-dependent ranking. It does not prove display fields or publication capture. |
| `LBR-B2` | `P-LBR-B2-PUBLICATION` | Keep a symbol unselected for more than 330 seconds while its aggregate state advances, select it solely through Day-%, and require immediate exact aggregate fields with only T/Q warming. Compose corrections, field-local failures, genuine zero, no-print, a same-`T` trust closure, concurrent immutable API captures, and slow/canceled clients. Observe one coherent publication identity, exact API-v2 semantic projection, accounting, lock independence, and no duplicated cycle. It does not prove T/Q formulas or HTTP/browser presentation. |
| `LBR-B3` | `P-LBR-B3-REMOVAL` | A source-and-behavior exclusion proof asserts the supported live build has no active HOD drawdown, rolling 30/60-minute range, old transaction/range Activity, old Tape-burst, legacy full-population evaluator, qualification clone, or redundant aggregate-derived owner. The approved current-product corpus and API-v2 golden stay unchanged, and the mature cycle records latency/allocation versus the frozen baseline. It detects dead fallback code that still mutates or publishes. It is not the final whole-process resource acceptance. |

`LBR-B1` installs incremental qualification and exact compact selection. It
makes old qualification-map cloning and full-history qualification scans
removable.

`LBR-B2` installs selected enrichment, primary accounting, cadence control,
and one immutable publication. It makes the old per-field full-universe
evaluation loop, separately sampled engine/TQ snapshot joins, and lock-coupled
capture path removable.

`LBR-B3` deletes the superseded evaluator and feature state, including active
code/tests for HOD drawdown, rolling 30/60-minute ranges, the old Activity
composite, old Tape burst, and temporary Capability A projection adapters. It
does not delete the compact T/Q implementation owned by Capability C or make
the final replay/checkpoint/tooling decision owned by `LBR-E1`.

## 9. Verification, review, and discretion

Each slice runs its primary proof, direct product/API regressions, affected
short and race tests, focused vet, `git diff --check`, and ordinary repository
verification at its gate. The Capability B review focuses on qualification
boundary ties, exact ordering, selection-independent aggregate history,
accounting identities, one-publication coherence, capture lock independence,
and demonstrated source removal. A narrow review is triggered when revision
linearization or immediate false-current closure remains representable after
the primary proof.

The implementer may choose proof indexes, heaps/fixed arrays, immutable value
layout, dirty flags, timer coalescing, private packages, and allocation-saving
mechanics. The implementer may not change formulas, ranking order, semantic
delay, watermark meaning, field availability, API-v2 meaning, T/Q independence,
or browser behavior.

Canonical merge/hydration, T/Q formula and retention, provider decoding,
connection retries, API/schema redesign, UI redesign, provider access, replay
repair, checkpoint repair, and public deployment are non-scope. Slice evidence
and status are recorded only in the delivery program.
