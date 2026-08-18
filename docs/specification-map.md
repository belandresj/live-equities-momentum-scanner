# Specification map

**Status:** Approved authority index; the 2026-08-14 live feature-set MVP is
the current sequential delivery program.

**Phase 1 approved:** 2026-08-05

**Product feature-set revision approved:** 2026-08-14

**Day-%/From-Open presentation, header-tooltip, compact dashboard-structure,
trader-facing status, and compact Rank-column revisions approved:** 2026-08-17

**Live feature-set MVP program approved:** 2026-08-14

**Phase 2 sequence approved:** 2026-08-05

**Version 1 Release Program approved:** 2026-08-07; replaces the former
C7-C11 unattended program

**C12 follow-on sequence approved:** 2026-08-09; retained as historical
authority and not active under the live feature-set MVP

This map identifies authoritative documents, component entry points,
dependencies, and coarse completion state. Current MVP delivery state lives in
the [`Live feature-set MVP program`](live-feature-mvp-program.md); historical
component evidence remains routed through each component parent.

## Authority

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved delivery-program authority for its stated scope;
4. focused component contracts; and
5. implementation assignments, plans, tests, fixtures, benchmarks, and code.

Within Phase 1, the [system overview](architecture/system-overview.md) controls
topology/ownership, the
[data/time/event contract](architecture/data-time-and-event-contract.md)
controls identity/clocks/order/coverage/reconciliation, and the
[engine lifecycle](architecture/scanner-state-engine-lifecycle.md) controls
legal operational transitions. None may weaken the
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
| [`architecture/system-overview.md`](architecture/system-overview.md) | Approved architecture; runtime topology, sole ownership, components, failure containment, checkpoint/API/UI boundaries. |
| [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md) | Approved architecture; session, identity, clocks, causal order, merge/reconciliation, watermark, replay, and checkpoint cutoff. |
| [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md) | Approved architecture; legal live/replay lifecycle, progress/exit, publication, recovery, suppression, session end, and shutdown. |
| [`glossary.md`](glossary.md) | Approved vocabulary; controlling product/architecture text wins when more specific. |

Phase 1 and the 2026-08-14 owner revision settle the eligible universe,
prior-close basis, unchanged qualification, exact qualified Day-% ranking/top
20, revised displayed fields, one engine/state/watermark/evaluator, 04:00-20:00
New York session, half-open windows, REST/live identity and precedence,
no-print proof, lifecycle, failure domains, accounting, API/UI ownership, and
independent availability. Existing checkpoint and replay architecture remains
available for future work, but neither path is a current MVP operating or
acceptance requirement. Focused contracts cite these meanings; they do not
create a shared replacement layer.

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
evidence, not proof of conformance to the revised product. The current work is
organized by the smallest affected capabilities—live backend measurements and
Float enrichment, snapshot API v2, final dashboard, and integrated live-MVP
acceptance—rather than by reopening every numbered historical component.
Offline replay and checkpoint persistence are specifically not reopened.
Until that sequential reconciliation and integrated acceptance are complete,
the repository implementation and the previously accepted private V1 RC
implement the superseded field set and must not be described as the revised
product.

Historical files under [`history/`](history/) are non-authoritative drafting
and review records.

## Operational authorities

| Document | Status and role |
| --- | --- |
| [`live-feature-mvp-program.md`](live-feature-mvp-program.md) | Current owner-approved delivery authority, 2026-08-14: smallest sequential live feature cutover; explicit backend/API/UI slices; replay unverified and non-gating; checkpoints disabled and non-gating. |
| [`live-aggregate-heartbeat-and-resubscription-correction.md`](live-aggregate-heartbeat-and-resubscription-correction.md) | Accepted focused correction, 2026-08-17: ordered one-at-a-time aggregate attempt retirement, individually paced finite recovery, preserved redacted handshake facts, inbound-aware heartbeat evidence, and startup/post-live return to current only through the existing hydration fence. Deterministic acceptance and final read-only review passed; credentialed provider chronology remains unobserved and separately authorized. |
| [`v1-release-program.md`](v1-release-program.md) | Accepted 2026-08-07 authority and evidence for the former C7-C11 field set. Its conflicting delivery gates are superseded for the current live feature-set MVP; still-compatible architecture and proof decisions remain usable evidence. |
| [`implementation-process.md`](implementation-process.md) | Owner-approved process, revised 2026-08-14: contract-first capability/component planning, correction/reopening, bounded verification, risk-based review, integration, and final validation under the current delivery program. |
| [`specifications/focused-component-spec-template.md`](specifications/focused-component-spec-template.md) | Mandatory template, revised 2026-08-07: compact/modular routing, current-plan and correction records, single parent ledger, proof and slice rules. |
| [`market-hours-validation.md`](market-hours-validation.md) | Procedure approved; execution pending a separate owner authorization. Not a live-MVP gate and not credential authority by itself. |
| [`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md) | Accepted owner-requested scanner-recovery baseline. Deterministic S1/S2, Gates A-E, the local D4 fence/capacity correction, and the D5 REST/live-precedence correction are accepted locally. D1-D4 observations remain evidence; another provider retry requires separate exact authorization. |
| [`live-tq-resilience-correction.md`](live-tq-resilience-correction.md) | Accepted 2026-08-14 T/Q resilience correction transplanted onto clean main: asynchronous status correlation, T/Q-local quarantine/accounting, exact recovery for possible aggregate loss, and engine-scheduled same-binding continuation pass focused and repository-wide deterministic proofs. Later D8 queue-pressure constants remain authoritative. Credentialed live confirmation is owner-run and not yet claimed. |
| [`live-ingress-first-cause-diagnostic.md`](live-ingress-first-cause-diagnostic.md) | D1 identified generic queue capacity after clean hydration; D2 made capacity causes decisive; D3 corrected the localized-conflict population transition and added its reason ledger. The D3 retry completed hydration but selected a long ingress-fence stall and slot saturation before retaining that ledger. |
| [`live-fence-finalization-and-burst-capacity-correction.md`](live-fence-finalization-and-burst-capacity-correction.md) | Accepted D4 correction: retained-tail full-universe evaluation now remains below the fixed responsiveness limit; terminal incidents preserve bounded last-coherent D3/accounting evidence; and owner-selected 32,768-slot retry headroom remains under the unchanged 128-MiB byte cap with fail-closed controls. |
| [`live-checkpoint-hot-path-correction.md`](live-checkpoint-hot-path-correction.md) | Finally accepted C7/C8 correction: bounded FIFO-tail projection continuations replace the multi-second monolithic owner hold, alias-isolated per-symbol transfer prevents retained writer aliases, checkpoint terminals return to engine accounting, O(1) continuation occupancy preserves saturated ingress cost, and the bounded decoder/restart objective passes; focused final re-review is clean. |
| [`v1-release-follow-on-goal.md`](v1-release-follow-on-goal.md) | Historical handoff prompt for the completed former C7-C11 program; not current execution authority. |
| [`replay-rest-feasibility-benchmark.md`](replay-rest-feasibility-benchmark.md) | Completed non-authoritative C12 acquisition evidence record and separately gated rerun protocol. It authorizes no credentials, provider request, production change, rerun, or capacity claim. |
| [`c12-implementation-goal.md`](c12-implementation-goal.md) | Historical replay follow-on handoff. Its retained implementation is unverified and non-gating for the live feature-set MVP. |
| [`replay-warmup-acceleration.md`](replay-warmup-acceleration.md) | Owner-authorized 2026-08-14 focused replay correction: accelerate cached 04:00-to-observation-start warm-up, preserve exact logical-boundary equivalence, then use the unchanged automatic 1x API/dashboard path. Separate from and non-gating for the live MVP. |

## Current delivery program

The [`Live feature-set MVP program`](live-feature-mvp-program.md) supersedes
conflicting former-field, replay, checkpoint, and component-number delivery
gates for this feature cutover. Implementation remains sequential with one
active slice and final read-only review at each accepted capability boundary.
The 2026-08-17 Day-%/From-Open value-relative presentation revision is
authoritative in [`product/product-goals.md`](product/product-goals.md#pg-ui-03)
and reconciled in [`specifications/independent-ui.md`](specifications/independent-ui.md#205-from-open-presentation-correction-acceptance);
it changes no API or backend contract.
The active order is:

1. live backend measurements and Float reference enrichment;
2. snapshot API v2;
3. final dashboard; and
4. integrated live-MVP acceptance.

The [`Version 1 Release Program`](v1-release-program.md) remains an accepted
record of the former feature-set delivery and reusable evidence. It is not the
current execution plan and cannot require replay proof, checkpoint repair, or
unrelated repository cleanup before the revised live product works.

## Phase 2 focused component sequence

The path in this table is the authoritative component entry point. A modular
component keeps its parent at this path and routes details beneath a same-named
directory. Subordinate specs do not create new components or mutable delivery
ledgers.

| Sequence | Focused component | Outcome and current state |
| --- | --- | --- |
| 1 | [Reference data and session binding](specifications/reference-data-and-session-binding.md) | Finally accepted 2026-08-05. Owns immutable trading date/session bounds, eligible universe, required prior session, adjusted prior closes, and binding identity. |
| 2 | [ScannerStateEngine and canonical state](specifications/scanner-state-engine-and-canonical-state.md) | Finally accepted 2026-08-05. Owns sole ordered mutation, canonical aggregate state, committed watermark, immutable publication, and typed extension seams. |
| 3 | [Aggregate features, qualification, ranking, and accounting](specifications/aggregate-features-qualification-ranking-and-accounting.md) | Finally accepted 2026-08-06. Owns formulas, correction-aware qualification, exact order/top 20, independent availability, and population accounting. |
| 4 | [Aggregate replay](specifications/aggregate-replay.md) | Historical accepted evidence for the former aggregate fields. The live feature-set MVP does not reopen this component, depend on it, or claim that its retained implementation works with the revised fields. |
| 5 | [Massive live adapter](specifications/massive-live-adapter.md) | Finally accepted 2026-08-09 after the integrated V1 RC reopened and corrected blocked-dequeue T/Q acknowledgement/deadline correlation with clean focused re-review. Owns bounded A/T/Q/control classification, normalization, causal positions, epochs, commands, and acknowledgements. |
| 6 | [Aggregate REST hydration and recovery](specifications/aggregate-rest-hydration-and-recovery.md) | Finally accepted 2026-08-07. Owns production pagination/workers, fresh/checkpoint/gap plans, exact terminal outcomes, ingress fencing, and REST/live reconciliation. |
| 7 | [Checkpoints and restart](specifications/checkpoints-and-restart.md) | Historical accepted evidence for the former state schema. Checkpoint code is retained but disabled and non-gating for the live feature-set MVP; fresh hydration is the supported restart path, and an old schema must not be presented as compatible with revised fields. |
| 8 | [Readiness and operations](specifications/readiness-and-operations.md) | Finally accepted 2026-08-07 and corrected with clean focused re-review 2026-08-10 after live-start analysis invalidated the whole-hydration 60-second deadline. Connection establishment remains bounded; finite C6 hydration continues under per-request bounds while the subscribed live tail is consumed; readiness waits for the exact ingress fence; concurrent terminal diagnostics and shutdown joins are race-safe. Runnable composition, honest readiness/staleness, engine-owned bounded recovery/exhaustion, fixed-cardinality measurements, and controlled 6,000-symbol mixed load remain accepted. |
| 9 | [Top-20 T/Q coverage and features](specifications/top-20-tq-coverage-and-features.md) | Reaccepted after the owner-directed waiting-pressure/frame-local correction. Ordinary live configuration targets all ranked displayed rows up to 20 through serialized paired acknowledgements. Global pressure uses waiting frame/byte backlog, separate capacity/accounting/bound loss, and guarded aggregate lag; active-frame age is diagnostic only, with a 500-ms frame-local T/Q budget preserving later aggregates/controls. Spread retains the latest valid numeric quote with age through quiet coverage. Required deterministic, ordinary, affected race, vet, and diff gates pass; live/provider capacity remains unclaimed. |
| 10 | [Versioned snapshot API](specifications/versioned-snapshot-api.md) | Finally accepted 2026-08-08. One sealed immutable capture maps to the exact versioned schema; loopback HTTP, exact-origin CORS, liveness/readiness, bounded transport, and joined scanner composition pass with clean focused final re-review. |
| 11 | [Independent UI](specifications/independent-ui.md) | Finally accepted 2026-08-09. Production Chrome visual/interaction/accessibility/independence proof, final correction review, and integrated private V1 RC review are clean. Independent Chrome-desktop UI has every V1 field/status and no browser-owned market logic. |
| 12 | [Historical replay product mode](specifications/historical-replay-product-mode.md) | Historical, incomplete follow-on work. Its currently runnable capability is unknown; it is neither a supported path nor an acceptance gate for the live feature-set MVP. |

The numbered component contracts remain the source of reusable ownership and
proof evidence. They are not being renamed or comprehensively cleaned up in
the MVP. Current implementation follows MVP-S1 through MVP-S4, and only the
lower-level boundaries touched by those capabilities are reconciled.

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

The current MVP uses deterministic fixtures for formula, API, and UI work while
the market is closed, but the only supported operating claim is the fresh-start
live scanner. Replay is not a fallback acceptance path. Any credentialed
market-hours observation still requires the separate authorization described
in [`market-hours-validation.md`](market-hours-validation.md). Scanner
correctness does not prove trading edge or executable expectancy.
