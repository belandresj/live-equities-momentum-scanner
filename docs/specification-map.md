# Specification map

**Status:** Approved Phase 1 authority index, Phase 2 sequence, accepted
private/local V1 RC, and active C12 follow-on sequence.

**Phase 1 approved:** 2026-08-05

**Phase 2 sequence approved:** 2026-08-05

**Version 1 Release Program approved:** 2026-08-07; replaces the former
C7-C11 unattended program

**C12 follow-on sequence approved:** 2026-08-09; begins only after accepted
C11/private V1 RC and accepted C4-S6

This map identifies authoritative documents, component entry points,
dependencies, and coarse completion state. Detailed delivery state lives only
in each component parent.

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
| [`product/product-goals.md`](product/product-goals.md) | Approved product contract, 2026-08-05; highest product authority. |
| [`architecture/system-overview.md`](architecture/system-overview.md) | Approved architecture; runtime topology, sole ownership, components, failure containment, checkpoint/API/UI boundaries. |
| [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md) | Approved architecture; session, identity, clocks, causal order, merge/reconciliation, watermark, replay, and checkpoint cutoff. |
| [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md) | Approved architecture; legal live/replay lifecycle, progress/exit, publication, recovery, suppression, session end, and shutdown. |
| [`glossary.md`](glossary.md) | Approved vocabulary; controlling product/architecture text wins when more specific. |

Phase 1 settles the eligible universe, prior-close basis, qualification, exact
Day-% ranking/top 20, displayed V1 fields, one engine/state/watermark/evaluator,
04:00-20:00 New York session, half-open windows, REST/live identity and
precedence, no-print proof, lifecycle, failure domains, accounting,
checkpoints, replay, API/UI ownership, and independent availability. Focused
contracts cite these meanings; they do not create a shared replacement layer.

Historical files under [`history/`](history/) are non-authoritative drafting
and review records.

## Operational authorities

| Document | Status and role |
| --- | --- |
| [`v1-release-program.md`](v1-release-program.md) | Owner-approved C7-C11 authority, 2026-08-07: fixed/revisable decisions, zero-interruption correction/containment, minimum scope, capability/proof matrix, test tiers, and private/local V1 RC definition. |
| [`implementation-process.md`](implementation-process.md) | Owner-approved process, revised 2026-08-07: contract-first planning, revisable C7-C11 delivery, correction/reopening, bounded verification, risk-based review, integration, and final validation. |
| [`specifications/focused-component-spec-template.md`](specifications/focused-component-spec-template.md) | Mandatory template, revised 2026-08-07: compact/modular routing, current-plan and correction records, single parent ledger, proof and slice rules. |
| [`market-hours-validation.md`](market-hours-validation.md) | Procedure approved; execution pending a separate owner authorization. Not a private V1 RC gate and not credential authority by itself. |
| [`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md) | Active owner-requested correction parent and sole cross-correction ledger. Deterministic S1/S2, Gates A-E, the local D4 fence/capacity correction, and the D5 REST/live-precedence correction are accepted locally. D1-D4 observations remain evidence; another provider retry requires separate exact authorization. |
| [`live-ingress-first-cause-diagnostic.md`](live-ingress-first-cause-diagnostic.md) | D1 identified generic queue capacity after clean hydration; D2 made capacity causes decisive; D3 corrected the localized-conflict population transition and added its reason ledger. The D3 retry completed hydration but selected a long ingress-fence stall and slot saturation before retaining that ledger. |
| [`live-fence-finalization-and-burst-capacity-correction.md`](live-fence-finalization-and-burst-capacity-correction.md) | Accepted D4 correction: retained-tail full-universe evaluation now remains below the fixed responsiveness limit; terminal incidents preserve bounded last-coherent D3/accounting evidence; and owner-selected 32,768-slot retry headroom remains under the unchanged 128-MiB byte cap with fail-closed controls. |
| [`live-checkpoint-hot-path-correction.md`](live-checkpoint-hot-path-correction.md) | Finally accepted C7/C8 correction: bounded FIFO-tail projection continuations replace the multi-second monolithic owner hold, alias-isolated per-symbol transfer prevents retained writer aliases, checkpoint terminals return to engine accounting, O(1) continuation occupancy preserves saturated ingress cost, and the bounded decoder/restart objective passes; focused final re-review is clean. |
| [`v1-release-follow-on-goal.md`](v1-release-follow-on-goal.md) | Short handoff prompt for the subsequent autonomous C7-C11 goal. |
| [`replay-rest-feasibility-benchmark.md`](replay-rest-feasibility-benchmark.md) | Completed non-authoritative C12 acquisition evidence record and separately gated rerun protocol. It authorizes no credentials, provider request, production change, rerun, or capacity claim. |
| [`c12-implementation-goal.md`](c12-implementation-goal.md) | Owner-approved handoff prompt: close C11/private V1 RC, then implement and accept C4-S6, C12-S1, and C12-S2 sequentially. The focused component contracts remain authoritative. |

## Version 1 Release Program

The former C7-C11 program froze post-approval fixtures, proofs, slices,
benchmarks, interfaces, codecs, and accepted implementation decisions. The
owner replaced it with the
[`Version 1 Release Program`](v1-release-program.md): Phase 1 product and
architecture meaning remain fixed, while C7-C11 lower-level delivery decisions
remain revisable through final V1 acceptance.

Implementation stays sequential with one active slice. Correctable failures
reopen the affected item and continue. C7-C11 have no planned owner-response
gate; excluded work and operational friction use the program's automatic
containment/fallback rules. Review is risk-triggered, plus one final read-only
review per component and one final integrated V1 RC review.

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
| 4 | [Aggregate replay](specifications/aggregate-replay.md) | Finally re-accepted after the narrow production-reader and playback-preparation resource corrections. The exact 2,584,011,150-byte artifact validates all 7,671,171 records in 40.53 seconds; trusted 17:15 preparation plus same-open stream validates the 7,587,384-record prefix and 7,581,690-row valid-prior subset in 2m6.65s. Compact/adversarial trust proofs, repository short, focused race, vet/diff, and persisted-trust reviews are clean without changed schema or admission semantics. Owns offline downloader/compiler, normalized artifact, deterministic source/clock, shared aggregate path, and full-source-trusted prefix completion with distinct lifecycle/accounting. |
| 5 | [Massive live adapter](specifications/massive-live-adapter.md) | Finally accepted 2026-08-09 after the integrated V1 RC reopened and corrected blocked-dequeue T/Q acknowledgement/deadline correlation with clean focused re-review. Owns bounded A/T/Q/control classification, normalization, causal positions, epochs, commands, and acknowledgements. |
| 6 | [Aggregate REST hydration and recovery](specifications/aggregate-rest-hydration-and-recovery.md) | Finally accepted 2026-08-07. Owns production pagination/workers, fresh/checkpoint/gap plans, exact terminal outcomes, ingress fencing, and REST/live reconciliation. |
| 7 | [Checkpoints and restart](specifications/checkpoints-and-restart.md) | Finally accepted 2026-08-07 under the V1 program. S1-S3 and all eight proofs pass; the corrected 6,000-symbol restart median is 10.286 seconds versus 36.641 seconds fresh (71.9% faster), with clean focused final re-review. |
| 8 | [Readiness and operations](specifications/readiness-and-operations.md) | Finally accepted 2026-08-07 and corrected with clean focused re-review 2026-08-10 after live-start analysis invalidated the whole-hydration 60-second deadline. Connection establishment remains bounded; finite C6 hydration continues under per-request bounds while the subscribed live tail is consumed; readiness waits for the exact ingress fence; concurrent terminal diagnostics and shutdown joins are race-safe. Runnable composition, honest readiness/staleness, engine-owned bounded recovery/exhaustion, fixed-cardinality measurements, and controlled 6,000-symbol mixed load remain accepted. |
| 9 | [Top-20 T/Q coverage and features](specifications/top-20-tq-coverage-and-features.md) | Finally reaccepted 2026-08-13 after the owner-selected private 8-GiB Apple M1 heap profile corrected gates that were structurally below the observed aggregate baseline. Exact heap boundaries, aggregate-independent pressure shedding, continuous-health ranked restoration, ordinary/race verification, and focused final re-review pass; live/provider capacity remains unclaimed. |
| 10 | [Versioned snapshot API](specifications/versioned-snapshot-api.md) | Finally accepted 2026-08-08. One sealed immutable capture maps to the exact versioned schema; loopback HTTP, exact-origin CORS, liveness/readiness, bounded transport, and joined scanner composition pass with clean focused final re-review. |
| 11 | [Independent UI](specifications/independent-ui.md) | Finally accepted 2026-08-09. Production Chrome visual/interaction/accessibility/independence proof, final correction review, and integrated private V1 RC review are clean. Independent Chrome-desktop UI has every V1 field/status and no browser-owned market logic. |
| 12 | [Historical replay product mode](specifications/historical-replay-product-mode.md) | C12-S1 accepted 2026-08-09 after C11/private V1 RC and C4-S6. Cache-only full validation, deterministic warm-up, cumulative 1x observation, atomic replay API state, exact C4 terminal identity, and bounded failure containment pass allocated proofs, correction/re-review, full short, focused race, and vet. C12-S2 replay presentation and the authorized in-place B4 acceptance are active. C12 is not part of the C7-C11 V1 RC gate. |

Implementation is sequential. C8-C11 detailed contracts are completed just in
time after the preceding final interface unless an already stable dependency
permits safe earlier reconnaissance. Each future contract is compact and uses
at most two slices unless a third protects a distinct consequential boundary.
Later evidence may revise/reopen C7-C11 lower-level decisions but may not create
concurrent implementation or a competing owner.

## V1 vertical milestones

1. **Deterministic aggregate core:** C1-C4, already accepted.
2. **Production aggregate lifecycle:** C5-C8, including corrected checkpoint
   restart, runnable readiness/recovery, and local operational evidence.
3. **T/Q enrichment:** C9, including normal top-20 coverage and complete
   aggregate-protecting shedding/restoration.
4. **Private/local V1 RC:** Accepted 2026-08-09. C10-C11, loopback API,
   independent Chrome UI, deterministic production-path integrated evidence,
   and the final read-only review are clean.

The private/local V1 RC can complete while the market is closed. Credentialed
market-hours observation remains pending under
[`market-hours-validation.md`](market-hours-validation.md). Scanner correctness
does not prove trading edge or executable expectancy.
