# ScannerStateEngine, canonical symbol state, and feature-boundary implementation details

**Status:** Finally approved 2026-08-05 after `S1`–`S4` implementation, the
required independent reviews and corrections, and the separate read-only final
component review. This manifest is the authoritative Component 2 contract. No
acceptance item remains open.

**Owner boundary approval:** approved 2026-08-05 in the owning Codex task

**Owner contract/reuse/test/slice-plan approval:** approved 2026-08-05 in the
owning Codex task after adversarial final review and owner-approved corrections

**Owner correction-horizon decision:** `H = 16 minutes`, accepted 2026-08-05
and reaffirmed with complete-contract approval

**Modular-layout migration approval:** owner-accepted 2026-08-05 through the
explicit instruction to delete the verified legacy key, monolith, and section
mirrors. The migration changes document layout and context routing only; it
does not change a requirement, reuse decision, proof, slice allocation,
approval claim, or implementation authorization.

**S1 implementation acceptance:** owner-accepted 2026-08-05 after all three
allocated primary proofs, package/race/repository verification, correction of
the independent review's FIFO lifecycle-ordering finding, and successful
targeted re-review. `S2` was subsequently authorized and accepted through its
separate bounded assignment.

**S2 implementation acceptance:** owner-accepted 2026-08-05 after the complete
`ENG-AGG-01` validation/merge/retention matrix, rerun of all accepted S1
proofs, package/race/repository verification, correction of the independent
review's lifecycle-bypass, integrity-reentry, multi-identity historical-result,
and stale historical-authority findings, and successful final targeted
re-review. No S2 review or acceptance item remains open. That acceptance did
not itself authorize `S3`; `S3` was later separately authorized and accepted.

**S3 implementation acceptance:** owner-accepted 2026-08-05 after the three
allocated `ENG-TIME-01`, `ENG-COMMIT-01`, and `ENG-LIFE-01` primary proofs;
rerun of all accepted S1 proofs and the complete S2 `ENG-AGG-01` matrix;
package/race/repository verification; correction of the independent review's
post-suppression binding-install and illegal-timer target-mutation findings;
and successful final targeted re-review. No S3 review or acceptance item
remains open. `S4` was subsequently separately authorized and accepted.

**S4 implementation acceptance:** owner-accepted 2026-08-05 after all five
allocated `ENG-INPUT-01`, `ENG-PUBLISH-01`, `ENG-MODULE-01`, `ENG-FAIL-01`,
and `ENG-OBS-01` primary proofs; rerun of every accepted S1–S3 primary proof;
package/race/repository/vet verification; correction of the independent
review's duplicated internal publication path and unknown-input fallthrough
findings; and successful clean targeted re-review. No S4 review or acceptance
item remains open. This acceptance completes the implementation-slice sequence
but does not replace the required separate read-only final Component 2 review
or authorize Component 3 implementation.

**Final-review correction:** owner-directed 2026-08-05 after the first complete
read-only review returned four blockers. The implementation, routed proof text,
and allocated tests now (1) carry a closed suppression disposition and preserve
the actual cause in unavailable publication, (2) seal and drain the captured
FIFO on canonical integrity failure, (3) copy a bounded per-identity historical
decision context without retaining caller maps, and (4) charge both lazy
session bitmaps at a combined 1,440,000,000-byte raw ceiling for 100,000 active
symbols. All twelve non-cached `ENG-*` primary tests and the complete required
verification set pass.

**Final component review:** owner-confirmed complete 2026-08-05 after the
corrected separate read-only re-review. No blocking conformance, proof,
boundedness, predecessor-coupling, ownership, or scope finding remains.

**Controlling Phase 1 requirements:** The exact, nonduplicated relationship
ledger is in [authority and reuse](scanner-state-engine-and-canonical-state/authority-and-reuse.md#5-phase-1-authority-trace).
It covers the applicable `PG-*`, `ARCH-*`, `DTE-*`, and `LIFE-*` requirements;
Component 2 directly implements only the engine-owned subset identified there.

**Approved dependency:** the finally approved Component 1
[`reference.Binding`](reference-data-and-session-binding.md#9-detailed-semantic-inputs-outputs-and-owned-state)
boundary. Component 2 may import it read-only and may not reinterpret or edit
Component 1.

## Contract document map

The documents below are one Component 2 contract and one authority. The parent
is mandatory for every task. A localized implementation task reads only its
slice row and declared dependencies; boundary approval, completed-contract
approval, cross-cutting changes, and final component review read the complete
manifest.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Component outcome, single ownership boundary, scope/non-scope, cross-cutting Phase 1 invariants, routing, status, and contract-wide prohibitions | Template Sections 1–4; all `ENG-*` requirements only as routed summaries | Every Component 2 task | Component 1 binding contract |
| [Authority and reuse](scanner-state-engine-and-canonical-state/authority-and-reuse.md) | Exact Phase 1 relationship ledger, resolved skeleton questions, approved reconnaissance, provenance, reuse decisions, and whitelist | Template Sections 5–8; all Phase 1 IDs; v2 evidence allocated to S1–S4 | Contract provenance, whitelist questions, final review; S2/S4 evidence use | Parent |
| [S1 — engine kernel](scanner-state-engine-and-canonical-state/s1-engine-kernel.md) | Atomic binding installation, bounded immutable admission, FIFO ownership/order, S1 accounting and containment | `ENG-RUN-01`, `ENG-ADMIT-01`, `ENG-ORDER-01`; `DF-01`–`DF-05`, binding `DF-12`, linkage `DF-15`; `S1` | S1 assignment, implementation, proof, and review | Parent; Component 1 |
| [S2 — canonical aggregates](scanner-state-engine-and-canonical-state/s2-canonical-aggregates.md) | Aggregate fact/state shape, validation, identity/equality, merge/precedence, correction/finalization, bounds, and proof | `ENG-AGG-01`; aggregate `DF-06`, `DF-07`, `DF-08`, aggregate-bound `DF-16`; `S2` | S2 assignment, implementation, proof, and review | Parent; S1; authority/reuse |
| [S3 — time, lifecycle, and commit](scanner-state-engine-and-canonical-state/s3-time-lifecycle-and-commit.md) | Injected-clock/timer semantics, remaining realizable lifecycle edges, and the closed base commit gate | `ENG-TIME-01`, `ENG-COMMIT-01`, `ENG-LIFE-01`; `DF-10`, `DF-11`, remaining `DF-12`, `DF-17`; `S3` | S3 assignment, implementation, proof, and review | Parent; S1; S2 |
| [S4 — publication, containment, and extension](scanner-state-engine-and-canonical-state/s4-publication-containment-and-extension.md) | Final common validation interaction, immutable publication, compiled extension rule, cross-path failure containment, completed accounting/observability, and final conformance walkthrough | `ENG-INPUT-01`, `ENG-PUBLISH-01`, `ENG-MODULE-01`, `ENG-FAIL-01`, `ENG-OBS-01`; cross-path `DF-03/04/06`–`DF-17`; `S4` | S4 assignment, implementation, proof, review, and final walkthrough | Parent; S1; S2; S3; authority/reuse |
| [Contract completion and governance](scanner-state-engine-and-canonical-state/contract-completion-and-governance.md) | Contract-wide implementation discretion, escalation rules not owned by one slice, completed-contract checklist, and drift audit | Template Sections 17–19; final review only | Assignment preparation, contract change, complete-manifest review | Parent and all preceding detail documents |

**Layout:** Modular contract. The files named in this table are the complete
normative set. The legacy key, split migration sources, and prohibited
monolith were deleted after owner acceptance and are not implementation inputs.

**Routing rule:** If a requirement, proof, trust boundary, edge case, or task
is not unambiguously routed above, work stops for a parent-map correction.
Models do not guess among detail specs or open the prohibited monolith.

**Contract-wide coverage and acceptance:** The authoritative per-requirement
proof and slice allocations are in the four slice documents. The full
acceptance state and drift audit are in
[contract completion and governance](scanner-state-engine-and-canonical-state/contract-completion-and-governance.md).

## 1. Outcome and user consequence

One `ScannerStateEngine` instance is the only ordered mutable authority for one
bound scanner run. It consumes admitted facts through one ordered transition
boundary, assigns engine sequence, applies binding and applicable context
checks, owns canonical same-session state and the single committed aggregate
watermark `T`, and publishes only immutable internally coherent views.

This prevents concurrent adapters or workers from creating a second scanner
truth, prevents readers from seeing or mutating half-applied state, and lets a
correction or status change produce a distinct publication at the same `T`.
Every stale, wrong-binding, invalid, duplicate, unsupported, or fenced fact is
classified. If binding, clock, ingress, canonical, ordering, accounting, or
publication integrity is globally unsupportable, the engine removes the
current claim instead of fabricating one.

Component 2 supplies the ownership and immutable seams for later behavior. It
does not implement aggregate qualification/ranking/features, production
readiness, hydration/recovery, checkpoints, T/Q measurements, the public API,
or the UI.

## 2. Scope and explicit non-scope

**In scope**

- Bind one engine immutably to one approved Component 1 session binding.
- Own lifecycle, engine sequence, accepted context identifiers, canonical
  same-session state, aggregate merge state, committed `T`, transition
  accounting, immutable internal publication, and coherent projection seams.
- Admit a closed family of bounded typed facts through one FIFO ownership
  transfer and one logically atomic transition path.
- Preserve distinct event, receipt, engine, replay, committed, generated, and
  checkpoint times and distinct source-position evidence.
- Apply common and family-specific validation before affected mutation.
- Enforce provider-independent aggregate identity, exact duplicate/revision
  behavior, live causal precedence, historical fill-only precedence, same-`T`
  republication, and absence without synthetic aggregates.
- Accept explicit timer/control facts without fabricating market data.
- Keep queueing, canonical retention, publications, diagnostics, and local I/O
  handoffs bounded.

**Not in scope**

- Component 1 schedule, universe, prior-close, cache, or binding construction.
- Component 3 formulas, qualification, Day-% ordering, top 20, availability,
  and symbol-population accounting.
- Component 4 artifact/download/compiler/playback behavior.
- Component 5 WebSocket/provider normalization and protocol behavior.
- Component 6 pagination, hydration/recovery planning, no-print proof, and
  terminal-work production.
- Component 7 checkpoint schema, contents, validation, codec, storage, cadence,
  or restart selection.
- Component 8 production `D`, `C/R`, readiness/currentness/capacity/retry/
  shutdown policy and operational mappings.
- Component 9 T/Q conditions, membership, coverage, features, and pressure
  policy.
- Component 10 public schema/transport and Component 11 UI.
- A generic event bus, runtime plugin, database, service split, per-symbol
  actors, browser-owned state, or speculative later-feature scaffolding.

## 3. Ownership and dependencies

There is one mutable graph, reachable only from the engine owner. Concurrent
adapters and workers may own bounded local I/O state and return immutable typed
facts; they receive no writable engine reference. A successful admission
transfers one self-contained node into the single FIFO. One consumer assigns
engine order and completes disposition, mutation, installed contributor work,
accounting, lifecycle/commit evaluation, and publication decision before the
transition is externally complete.

The engine installs exactly one concrete immutable Component 1 binding and
copies its values into engine-owned state. A changed date, bound, universe,
prior-close dataset/policy, or identity requires a new engine. Feature,
ranking, recovery, checkpoint, readiness, and T/Q logic added later must be
statically compiled into named engine-owned state and the same transition;
none receives a queue, clock, watermark, evaluator, publisher, or lifecycle
transition authority. Readers and checkpoint writers consume immutable
projections only.

## 4. Settled Phase 1 semantic boundary

| Boundary | Settled meaning | Controlling authority |
| --- | --- | --- |
| Binding/lifetime | At most one concrete valid binding per engine; every later fact names it; replacement means a new engine. | `ARCH-OWN-01`, `DTE-SESSION-02`, `DTE-EVENT-01`, `LIFE-MODEL-01`, `LIFE-INIT-02` |
| Input/ordering | One bounded closed typed FIFO; admission is not canonical acceptance; one positive nonwrapping engine sequence and one complete disposition per consumed node. | `ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-MODEL-01`, `DTE-MODEL-02`, `DTE-EVENT-01`–`DTE-EVENT-04`, `LIFE-MODEL-02` |
| Atomicity/ownership | Validation precedes affected mutation; a transition is externally visible only after all installed stages and its publication decision; no second mutable or reader path exists. | `ARCH-OWN-01`–`ARCH-OWN-04`, `LIFE-MODEL-02`, `LIFE-MODEL-04` |
| Canonical aggregate | Identity is `(symbol,window_start)` within the binding; absence is absence; exact duplicate, revision, live causal precedence, historical fill-only precedence, and conflicts have one engine-owned result. | `PG-RANK-02`, `DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04`, `DTE-AGG-01`–`DTE-AGG-03`, `DTE-MERGE-01`–`DTE-MERGE-05` |
| Time and `T` | Event time selects membership, admission-sampled engine time drives target evaluation, FIFO order drives mutation, generated time is metadata, and one nondecreasing `T` advances only through the central run-specific proved gate. | `DTE-CLOCK-02`–`DTE-CLOCK-06`, `DTE-TIMER-01`, `DTE-COMMIT-01`, `DTE-COMMIT-02`, `DTE-COMMIT-04` |
| Publication | A private immutable publication has identity independent of `T`; same-`T` corrections or status changes can replace it; readers cannot alias engine state. | `ARCH-OWN-03`, `DTE-MERGE-05`, `DTE-COMMIT-04`, `LIFE-LIVE-02` |
| Lifecycle/later modules | Only the Phase 1 lifecycle graph exists. Later approved modules extend named engine-owned substate and fixed source-order calls without a competing owner or generic registration seam. | `ARCH-OWN-04`, `LIFE-MODEL-01`–`LIFE-MODEL-04`, `LIFE-T01`–`LIFE-T29` |
| T/Q isolation | T/Q data, health, shedding, and coverage never gate aggregate `T`, qualification, ranking, or backend readiness. | `PG-TAQ-02`, `PG-TAQ-03`, `DTE-TQ-01`–`DTE-TQ-03`, `LIFE-TQ-01`–`LIFE-TQ-03` |
| Recovery/checkpoint/replay/readers | Later facts enter the same owner; replay uses the same aggregate/timer path; checkpoints and readers receive coherent immutable projections. | `PG-OPS-01`, `PG-OPS-02`, `PG-REPLAY-01`, `PG-UI-01`, `DTE-RECOVERY-05`, `DTE-REPLAY-01`, `DTE-CHECKPOINT-01`, `LIFE-LIVE-04`, `LIFE-LIVE-05` |
| Boundedness/accounting | Every primary population has a closed identity; retained state and diagnostic cardinality are finite and explicitly charged. | `PG-OBS-01`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, `DTE-REJECT-01`, `DTE-REJECT-02`, `LIFE-MODEL-04` |

This contract introduces no new product rule, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, changed interval/correction
meaning, duplicated responsibility, generic bus/plugin framework, database,
service split, or browser-owned scanner logic.

## Contract-wide implementation constraints

- Use Go 1.26 and the standard library inside `internal/engine`; Component 2
  assignments do not edit `internal/reference` or later packages.
- Keep exactly one FIFO, owner, injected clock authority, canonical map,
  lifecycle authority, central commit gate, and atomic publication cell.
- `H=16m` and its inclusive acceptance/strict finalization boundary are fixed.
- Production `D`, `C/R`, capacity, retry, freshness, and shutdown values remain
  deferred to Components 8/9.
- Do not add later feature/provider/replay-source/hydration/checkpoint/readiness/
  T/Q/API/UI behavior or a generic placeholder for it.
- No v1, credentials, live provider calls, or v2 source beyond the approved
  whitelist and exact per-slice allocation.
- Do not retain raw frames, URLs, credentials, arbitrary error strings,
  unbounded histories, high-cardinality labels, or uncharged mutable state.

Every slice stops for owner review and a narrow independent review of its
allocated ownership/trust boundary before dependent implementation proceeds.
Final Component 2 review follows S4 and does not authorize Component 3.
