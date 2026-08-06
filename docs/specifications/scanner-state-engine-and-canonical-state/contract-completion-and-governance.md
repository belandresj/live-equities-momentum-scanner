# ScannerStateEngine and canonical state — contract completion and governance

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Contract-wide implementation discretion,
prohibitions/escalation not owned by one slice, complete-manifest acceptance,
and drift audit

**Controlling requirements:** All parent-routed Phase 1 and `ENG-*`
requirements; this document adds no behavior requirement or proof

**Allocated slices:** None; read when preparing any assignment and for final
component/contract-change review

**Document dependencies:** Parent; authority/reuse; S1; S2; S3; S4

**Approval state:** Inherits the parent contract approval; not independently approved

## Documentation-only migration crosswalk

This is the exact semantic move plan from the approved Section 1–19 layout.
Rows split by requirement/proof/slice are partitioned, not copied; references
to an earlier owner are navigational summaries.

| Approved monolith content | Authoritative modular home |
| --- | --- |
| Status/approvals, Sections 1–2, Section 3 ownership conclusion, Section 4 cross-cutting boundary | Parent |
| Section 3.1 Phase 1 trace; resolved Section 5; Sections 6–8 reconnaissance/reuse/whitelist | Authority and reuse |
| Section 3.2 Component 1 dependency | Parent plus S1 concrete binding/install boundary |
| Section 3.3 ownership ledger | Parent for cross-cutting owner; each S1–S4 document for its exclusive state/seam |
| Section 9 input/output/state rows and Sections 10/10.5 requirement/trust rows | Partitioned by the requirement's approved S1–S4 allocation |
| Section 9.1 binding acceptance | S1 |
| Section 9.2 construction/ownership | S1 for binding/FIFO owner and S4 for publication/contributor/read no-alias completion |
| Section 10.1 aggregate algorithm | S2 |
| Section 10.2 clock/commit/publication order | S3 for clock/commit; S4 for creation-time publication completion |
| Section 10.3 family matrix | S4, with S1–S3 links to the already authoritative implemented-family specifics |
| Section 10.4 lifecycle staging | S3 |
| Sections 11–14 failure/accounting/bounds/edges and Section 14.1 dangerous cases | Partitioned by the exact requirement/proof/slice owner; S4 owns only final cross-path interaction |
| Section 15 proof rows | One row in the owning S1–S4 document; no requirement has two primary proofs |
| Section 16 slice rows | One owning slice document; common sequencing/review rules in this document |
| Section 16.1 assignment/review rules | Parent constraints, each slice assignment boundary, and this document's common governance |
| Section 16.2 final conformance walkthrough | S4 |
| Sections 17–19 | This document, with slice-local prohibitions/stops repeated only as routed implementation context in the owning slice |

## 17. Implementation discretion

Within one authorized slice, implementation may choose private helper names,
the exact file split inside `internal/engine`, standard-library mutex/condition/
channel mechanics realizing the one FIFO, immutable value versus privately
owned pointer representation, fixed enum encoding, map allocation strategy,
compact finalized-presence structure, error wrapping for external logs, and
equivalent standard-library algorithms. Every choice must preserve the routed
observable behavior, bounds, accounting identities, ownership, and proof.

Required adaptations are not discretionary:

- rewrite only the whitelisted v2 correction/window/long-path cases against the
  new atomic transition and engine-assigned sequence, retaining inclusive
  16-minute acceptance and strict finalization;
- reimplement exact equality and merge precedence without v2 `Apply` or
  `ApplyRecovery`;
- adapt atomic replacement/mutation isolation to the minimal private
  publication without the v2 `Snapshot`;
- adapt only fixed-cardinality enum/scalar diagnostics with Component 2 reasons
  and accounting; and
- create no contributor instance or runtime callback seam. Future contributors
  enter as explicit source-order calls with named engine-owned substate.

### Prohibited changes

- A second engine/canonical map/lifecycle owner/watermark/evaluator/clock/
  publication path or writable reader view.
- Caller/adapter/worker choice of engine sequence, `T`, lifecycle, readiness,
  merge precedence, or publication.
- Unbounded/per-symbol/priority queues, reordering admitted facts, a generic
  bus/reducer/plugin/framework, database, service split, or third-party state
  machine/queue/immutable collection.
- Changing `H=16m`, inclusive/strict boundary, one-second/half-open meaning,
  identity, live-over-historical precedence, or absence/no-print semantics
  without an owner-approved revision.
- Production defaults for `D`, `C/R`, retry, freshness, or shutdown before the
  owning Component 8/9 evidence and approval.
- Feature/ranking/accounting formulas, provider I/O/decoding, replay artifact
  scheduling, hydration/recovery/no-print planning, checkpoint contents/install/
  storage, readiness, T/Q, public API, UI, or a placeholder for any of them.
- Editing Component 1 or `internal/reference` from a Component 2 assignment.
- V2 `state.go`, `types.go`, owner/orchestration, recovery, readiness,
  checkpoint, API/UI, unapproved fixtures/sources; any v1 access; credentials;
  or live provider requests.
- Raw frames, session-long raw aggregate history, unbounded event/publication/
  error history, URLs/credentials, or high-cardinality metric labels.

### Stop/escalation conditions

Stop for owner review when:

- Component 1 differs from the approved immutable binding interface or has not
  passed its required dependency gate;
- a Phase 1 authority conflicts, or a later component needs to reinterpret
  binding, identity, correction, clock, lifecycle, `T`, publication, or owner;
- Component 3 shows the 16-minute horizon cannot preserve an approved
  correction-aware feature/latch, or the S2 boundary/equivalence proof fails;
- Components 5/6 require source-position/future-clock/merge/reconciliation/
  active-ledger behavior outside the contract or cannot provide a finite bound;
- any requested source/evidence is outside the approved whitelist;
- FIFO/reserve cannot avoid silent loss/reorder, any admitted input lacks one
  disposition, or any accounting identity fails;
- publication requires an unbounded graph, exposes a writable alias/partial
  transition, or needs a second publisher;
- implementation needs deferred production policy; or
- a slice cannot remain one buildable coherent behavior with its primary
  proofs passing.

## Owner decisions and slice sequence

The owner approved on 2026-08-05 the corrected Component 2 logic, exact
`H=16m`, whitelist, twelve `ENG-*` requirement/proof allocations, four-slice
sequence, and independent slice-review triggers. No substantive Component 2
decision is open. The owner accepted the verified modular migration on
2026-08-05 by explicitly directing deletion of the legacy key, monolith, and
section mirrors. The owner accepted `S1` implementation on 2026-08-05 after
its allocated proofs, full verification, correction of the independent
review's FIFO lifecycle-ordering finding, and successful targeted re-review.
The owner accepted `S2` implementation on 2026-08-05 after its complete
`ENG-AGG-01` proof, all S1 proof reruns, full verification, correction of every
independent-review finding, and successful final targeted re-review. No S2
review or acceptance item remains open. The owner accepted `S3` implementation
on 2026-08-05 after its three allocated primary proofs, all accepted S1 proof
reruns, the complete accepted S2 proof, full verification, correction of the
independent review's post-suppression binding-install and illegal-timer
target-mutation findings, and successful final targeted re-review. No S3
review or acceptance item remains open. The owner accepted `S4` implementation
on 2026-08-05 after its five allocated primary proofs, every accepted S1–S3
proof rerun, full verification, correction of the independent review's
duplicated internal publication path and unknown-input fallthrough findings,
and a clean targeted re-review. No slice acceptance item remains open.
The owner confirmed the separate read-only final Component 2 review complete
on 2026-08-05 with no remaining blocking finding. Component 2 is finally
approved. That completion clears Component 3's dependency gate but does not
authorize Component 3 implementation.

The exact sequence is S1 binding/admission/order; S2 canonical aggregate merge;
S3 time/lifecycle/closed commit gate; S4 publication/common validation/failure/
accounting/extension. One slice assignment and owner review at a time. S1
cannot split binding from FIFO without a temporary unordered install path; S2
is one merge proof family; S3 timer/lifecycle/commit share guards; S4 is the
first complete visibility/cross-path boundary.

Each slice reruns prior proofs and submits the routed review artifact. Every
slice freezes a consequential boundary and therefore receives its specified
narrow independent review before dependent implementation. A separate
read-only final Component 2 review follows S4.

## 18. Completed-contract acceptance checklist

- [x] Parent map lists the complete modular contract and exclusive normative
      responsibility for every detail document.
- [x] Template Sections 1–19, Phase 1 relationships, all twelve `ENG-*`
      requirements, their primary proofs, dangerous cases, and S1–S4 are routed
      exactly once.
- [x] Document dependencies are explicit and acyclic; localized slice bundles
      do not require unrelated details or the prohibited monolith.
- [x] Boundary/reconnaissance approval, exact v2 provenance, negative findings,
      decisions, whitelist, and allocations are recorded.
- [x] Inputs, state, ownership, behavior, trust acceptance, containment,
      accounting, observability, and exact bounds are complete.
- [x] Construction guarantees are distinct from representable invalid facts
      requiring runtime containment.
- [x] Every nontrivial edge case has Phase 1, provider, v2, mathematical, or
      owner-decision evidence; speculative provider cases remain deferred.
- [x] Each requirement has one primary proof naming claim, dangerous
      counterexample, participating paths, observable distinction, construction
      argument, inspection-only claims, and intentional limitation.
- [x] Race mode is secondary evidence only; no duplicate proof layer claims the
      same boundary.
- [x] Every requirement/proof belongs to exactly one coherent slice with exact
      dependency, package boundary, whitelist, review artifact, deferred
      behavior, independent-review trigger, and owner stop.
- [x] Slice dependencies are acyclic and later slices extend rather than
      replace earlier ownership/behavior.
- [x] Global and slice-specific discretion, prohibitions, and escalation are
      sufficient for one bounded assignment at a time.
- [x] Drift audit has no substantive yes.
- [x] Original contract/reuse/test/slice-plan approval and exact v2 whitelist
      are preserved.
- [x] Owner accepted the complete documentation-only modular migration after
      link, coverage, duplication, and context-size verification; the legacy
      key, monolith, and section mirrors were then deleted.

## 19. Drift audit

| Question | Result | Evidence |
| --- | --- | --- |
| New product rule? | No | `H=16m` is the already approved `DTE-MERGE-02` delegated value; ranking/readiness/hydration/TQ/API/UI remain later. |
| Another mutable owner, watermark, or evaluator? | No | One engine owns state/lifecycle/`T`/publication; later modules are named synchronous calls only. |
| Aggregate ranking/readiness depends on T/Q? | No | Optional T/Q may later shed before admission and cannot consume reserve or gate aggregate state/`T`/rank/readiness. |
| Changed session/event-time/window/correction/`T` semantics? | No | Component 1 binding, exact half-open seconds, engine order, approved merge/horizon, and proved nondecreasing `T` remain unchanged. |
| Unevidenced behavior? | No | Aggregate/retention/publication/diagnostic decisions retain exact approved evidence; admission/time/lifecycle/accounting derive from Phase 1/math; provider cases remain deferred. |
| Duplicated responsibility/Phase 1? | No | Parent summarizes and routes; slice documents own only Component 2 implementation choices, proofs, and delivery boundaries. |
| Unneeded machinery? | No | One FIFO/owner/map/clock/cell; no registry, contributor placeholder, framework, database, or service. |
| V2 drove the Phase 1 boundary? | No | V2 architecture is rejected; only narrow tests/store/scalar techniques inform allocated implementation/proof. |
| Layout migration changed substance? | No | Requirement IDs, horizon, algorithms, trust outcomes, bounds, whitelist, proof limitations, S1–S4 allocation, approvals, and implementation gate are preserved; only normative routing and prose consolidation changed. |

Any later substantive yes requires explicit owner review. Acceptance of this
layout changes which files are authoritative; it does not authorize S1,
combine slices, waive slice reviews, or authorize Component 3/production use.
