# ScannerStateEngine and canonical state — authority, reconnaissance, and reuse

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Exact Phase 1 relationship routing, resolved
skeleton questions, approved version 2 reconnaissance/provenance, reuse
decisions, and implementation whitelist

**Controlling requirements:** All Phase 1 IDs routed below; no independent
`ENG-*` behavior requirement

**Allocated slices:** Evidence is allocated to `S1`–`S4` as stated in the
whitelist; this document is not itself an implementation slice

**Document dependencies:** Parent only

**Approval state:** Inherits the parent contract approval; not independently approved

## 5. Phase 1 authority trace

Each applicable Phase 1 requirement has one relationship to Component 2.
The exact set mentioned by the approved contract is enumerated here so a
caller never has to expand shorthand ranges:

- `ARCH-FLOW-01`, `ARCH-FLOW-02`, `ARCH-FLOW-03`, `ARCH-FLOW-04`,
  `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-03`, `ARCH-OWN-04`.
- `DTE-AGG-01`, `DTE-AGG-02`, `DTE-AGG-03`, `DTE-AGG-04`,
  `DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-02`, `DTE-CHECKPOINT-03`,
  `DTE-CLOCK-01`, `DTE-CLOCK-02`, `DTE-CLOCK-03`, `DTE-CLOCK-04`,
  `DTE-CLOCK-05`, `DTE-CLOCK-06`, `DTE-COMMIT-01`, `DTE-COMMIT-02`,
  `DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-CONTROL-01`, `DTE-EVENT-01`,
  `DTE-EVENT-02`, `DTE-EVENT-03`, `DTE-EVENT-04`, `DTE-HYDRATE-01`,
  `DTE-HYDRATE-02`, `DTE-MERGE-01`, `DTE-MERGE-02`, `DTE-MERGE-03`,
  `DTE-MERGE-04`, `DTE-MERGE-05`, `DTE-MODEL-01`, `DTE-MODEL-02`,
  `DTE-MODEL-03`, `DTE-QUOTE-01`, `DTE-QUOTE-02`, `DTE-RECOVERY-01`,
  `DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-RECOVERY-04`,
  `DTE-RECOVERY-05`, `DTE-REJECT-01`, `DTE-REJECT-02`, `DTE-REPLAY-01`,
  `DTE-REPLAY-02`, `DTE-REPLAY-03`, `DTE-SESSION-01`, `DTE-SESSION-02`,
  `DTE-SESSION-03`, `DTE-SESSION-04`, `DTE-TIMER-01`, `DTE-TQ-01`,
  `DTE-TQ-02`, `DTE-TQ-03`, `DTE-TRADE-01`, `DTE-TRADE-02`,
  `DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-03`, `DTE-WINDOW-04`.
- `LIFE-END-01`, `LIFE-END-02`, `LIFE-END-03`, `LIFE-HYDRATE-01`,
  `LIFE-HYDRATE-02`, `LIFE-HYDRATE-03`, `LIFE-HYDRATE-04`,
  `LIFE-HYDRATE-05`, `LIFE-HYDRATE-06`, `LIFE-HYDRATE-07`,
  `LIFE-INIT-01`, `LIFE-INIT-02`, `LIFE-INIT-03`, `LIFE-INIT-04`,
  `LIFE-INIT-05`, `LIFE-LIVE-01`, `LIFE-LIVE-02`, `LIFE-LIVE-03`,
  `LIFE-LIVE-04`, `LIFE-LIVE-05`, `LIFE-MODEL-01`, `LIFE-MODEL-02`,
  `LIFE-MODEL-03`, `LIFE-MODEL-04`, `LIFE-PUBLISH-01`,
  `LIFE-PUBLISH-02`, `LIFE-PUBLISH-03`, `LIFE-RECOVER-01`,
  `LIFE-RECOVER-02`, `LIFE-RECOVER-03`, `LIFE-RECOVER-04`,
  `LIFE-RECOVER-05`, `LIFE-RECOVER-06`, `LIFE-REPLAY-01`,
  `LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-SUPPRESS-01`,
  `LIFE-SUPPRESS-02`, `LIFE-SUPPRESS-03`, `LIFE-TQ-01`, `LIFE-TQ-02`,
  `LIFE-TQ-03`, `LIFE-T01`, `LIFE-T02`, `LIFE-T03`, `LIFE-T04`,
  `LIFE-T05`, `LIFE-T06`, `LIFE-T07`, `LIFE-T08`, `LIFE-T09`,
  `LIFE-T10`, `LIFE-T11`, `LIFE-T12`, `LIFE-T13`, `LIFE-T14`,
  `LIFE-T15`, `LIFE-T16`, `LIFE-T17`, `LIFE-T18`, `LIFE-T19`,
  `LIFE-T20`, `LIFE-T21`, `LIFE-T22`, `LIFE-T23`, `LIFE-T24`,
  `LIFE-T25`, `LIFE-T26`, `LIFE-T27`, `LIFE-T28`, and `LIFE-T29`.
- `PG-AVAIL-01`, `PG-AVAIL-02`, `PG-AVAIL-03`, `PG-FEATURE-01`,
  `PG-FEATURE-02`, `PG-FEATURE-03`, `PG-FEATURE-04`, `PG-FEATURE-05`,
  `PG-OBS-01`, `PG-OBS-02`, `PG-OBS-03`, `PG-OPS-01`, `PG-OPS-02`,
  `PG-RANK-01`, `PG-RANK-02`, `PG-RANK-03`, `PG-RANK-04`, `PG-RANK-05`,
  `PG-REFERENCE-01`, `PG-REPLAY-01`, `PG-REPLAY-02`, `PG-TAQ-01`,
  `PG-TAQ-02`, `PG-TAQ-03`, `PG-UI-01`, `PG-UI-02`, and
  `PG-UNIVERSE-01`.

### Directly implemented by Component 2

| Exact IDs | Component 2 responsibility |
| --- | --- |
| `PG-RANK-02` | Own current-session aggregate mark/no-mark state without synthetic or cross-session marks; Component 3 evaluates qualification/rankability. |
| `ARCH-OWN-01`–`ARCH-OWN-04` | Sole mutable authority, fact-returning concurrency, immutable publication, and engine-contained module rule. |
| `ARCH-FLOW-01`–`ARCH-FLOW-04` | Bounded direct input, complete admitted disposition, explicit order, and no blocking I/O in the owner transition. |
| `DTE-MODEL-01`–`DTE-MODEL-03` | One canonical path, distinct processing stages, and separation of canonical values from delivery evidence. |
| `DTE-SESSION-02`–`DTE-SESSION-04` | One exact `[S,E)` binding and precision-preserving boundary handling; Component 1 supplies the dates and bounds. |
| `DTE-CLOCK-02`–`DTE-CLOCK-06` | Distinct event/receipt/engine/committed/generated time and one nonregressing injected clock plus nondecreasing `T`. |
| `DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04` | Half-open aggregate/session membership and absence as absence. |
| `DTE-EVENT-01`–`DTE-EVENT-04` | Common envelope, source ordering evidence, and one engine sequence assigned at consumption. |
| `DTE-AGG-01`–`DTE-AGG-03` | Provider-independent aggregate identity/values and pre-mutation structural rejection. |
| `DTE-CONTROL-01`, `DTE-TIMER-01` | Ordered fact-returning control/timer inputs without producer state authority or fabricated market data. |
| `DTE-MERGE-01`–`DTE-MERGE-05` | Exact duplicate/revision/precedence/conflict/same-`T` publication semantics; `H=16m`. |
| `DTE-COMMIT-01`, `DTE-COMMIT-02`, `DTE-COMMIT-04` | Candidate target, proof-gated `T`, ingress fencing, and coherent publication identity. |
| `DTE-REJECT-01` | Fixed correctness-relevant rejection classes and bounded counts. |
| `LIFE-MODEL-01`–`LIFE-MODEL-04` | One binding/lifecycle, atomic ordered transitions, subordinate statuses, and bounded observable reasons. |
| `LIFE-INIT-01`, `LIFE-INIT-02`, `LIFE-INIT-05` | Immutable run mode, atomic binding installation, and observable/terminal initialization waits. |
| `LIFE-LIVE-02`, `LIFE-LIVE-05` | Timer/input access to ordinary evaluation and immutable readers. |
| `LIFE-SUPPRESS-01`–`LIFE-SUPPRESS-03` | Global suppression classification/actions/disposition without globalizing isolatable faults. |
| `LIFE-END-01`–`LIFE-END-03`, `LIFE-PUBLISH-01` | Session/controlled stop, terminal instance, and process-liveness publication. |
| `LIFE-T01`, `LIFE-T02`, `LIFE-T04`, `LIFE-T05`, `LIFE-T07`, `LIFE-T09`, `LIFE-T10`, `LIFE-T14`, `LIFE-T15`, `LIFE-T17`, `LIFE-T27`, `LIFE-T29` | Engine-owned binding/timer/control, rejection, suppression, live-self, and termination edges realizable without later fact producers. |

### Structurally enabled here; behavior owned later

| Exact IDs | Component 2 seam and later owner |
| --- | --- |
| `PG-RANK-03`–`PG-RANK-05`, `PG-FEATURE-05` | One correction-aware evaluation/ranking attachment and atomic publication; Component 3 owns formulas and product outcomes. |
| `PG-AVAIL-01`–`PG-AVAIL-03` | Independently versioned field status without cross-field/global T/Q gating; Components 3, 8, 9. |
| `PG-TAQ-01`–`PG-TAQ-03` | Engine-owned future desired T/Q membership and coverage consequences; Component 9. |
| `PG-OPS-01`, `PG-OPS-02` | Coherent checkpoint projection and one post-recovery evaluator; Components 6–8. |
| `PG-REPLAY-01`, `PG-REPLAY-02` | Same aggregate/timer path under simulated time; Component 4; T/Q replay absent. |
| `PG-UI-01`, `PG-OBS-01`–`PG-OBS-03` | Immutable coherent internal state able to carry later accounting/readiness; Components 3, 6, 8, 10. |
| `DTE-WINDOW-03`, `DTE-TRADE-01`, `DTE-TRADE-02`, `DTE-QUOTE-01`, `DTE-QUOTE-02` | Bounded direct T/Q seams only after Component 9 defines payload/state/validity. |
| `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `DTE-RECOVERY-01`–`DTE-RECOVERY-05` | Engine-owned generation, terminal-result, fence, consequence, and evaluator seams; Component 6 owns work/proof/policy. |
| `DTE-COMMIT-03`, `DTE-REJECT-02` | Coherent evaluation/category boundary at `T`; Component 3. |
| `DTE-TQ-01`–`DTE-TQ-03` | Subordinate causal coverage and shedding without aggregate gating; Component 9. |
| `DTE-REPLAY-01`–`DTE-REPLAY-03` | Same aggregate/timer/control acceptance; Component 4 owns artifact order and delivery. |
| `DTE-CHECKPOINT-01`–`DTE-CHECKPOINT-03` | Coherent projection/install seam and distinct `T0/R/T/generated_at`; Component 7. |
| `LIFE-INIT-03`, `LIFE-INIT-04` | Checkpoint/pre-session-ack transition seams; Components 7/5. |
| `LIFE-HYDRATE-01`–`LIFE-HYDRATE-07` | Engine-owned startup hydration lifecycle and terminal exit; Components 5, 6, 8 supply facts/policy. |
| `LIFE-LIVE-01`, `LIFE-LIVE-03`, `LIFE-LIVE-04` | Live entry, no-print extension, and checkpoint projection; Components 3, 6, 7. |
| `LIFE-RECOVER-01`–`LIFE-RECOVER-06` | Recovery steps/fencing/terminal classification in the engine; Components 5, 6, 8. |
| `LIFE-TQ-01`–`LIFE-TQ-03` | Desired membership/ack/coverage/measurement/pressure remain subordinate; Component 9. |
| `LIFE-REPLAY-01`–`LIFE-REPLAY-03` | Replay initialization/progress/termination in the same owner; Component 4. |
| `LIFE-PUBLISH-02`, `LIFE-PUBLISH-03` | Legal lifecycle/readiness/publication combinations; Components 3, 6, 8, 10 supply detailed evidence/schema. |
| `LIFE-T03`, `LIFE-T06`, `LIFE-T08`, `LIFE-T11`–`LIFE-T13`, `LIFE-T16`, `LIFE-T18`–`LIFE-T26`, `LIFE-T28` | Preserve engine-owned transition destinations; later owners supply the exact checkpoint, ack, hydration, recovery, replay, or restoration fact. |

### External constraints only

| Exact IDs | External owner/constraint |
| --- | --- |
| `PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-RANK-01` | Component 1 fixes population and prior-close facts; Component 2 consumes without substitutes. |
| `PG-UI-02` | Components 10/11 own stable public meaning; the private publication is not automatically public schema. |
| `DTE-SESSION-01`, `DTE-CLOCK-01` | Component 1 schedule facts cannot be reinterpreted by market or engine time. |
| `DTE-AGG-04` | Components 4–6 normalize ATS and provenance; Component 2 retains but does not equate source mappings. |

`PG-FEATURE-01`–`PG-FEATURE-04` do not control Component 2 implementation;
their later modules must use this owner but Component 2 preselects no formula
or T/Q structure.

## 6. Resolved skeleton questions

The approved pre-reconnaissance skeleton asked about typed admission,
pre-admission refusal, minimum aggregate retention/`H`, static extension,
atomic transition shape, injected time/timers, proof supporting `T`, minimum
publication, checkpoint projection, staged lifecycle input, bounded rejection
categories, and slice count. Version 2 resolved none of the new engine's
ordering, admission, binding, lifecycle, clock/timer, watermark-proof, or
feature-extension design. The approved contract therefore derives those from
Phase 1. V2 contributes only the narrow evidence below.

The resolved choices are: one closed compiled typed FIFO; exact admission
results and required reserve; one owner transition; `H=16m`; explicit static
source-order extension only when a later component exists; injected serialized
clock reads and timer facts; a private run-specific commit gate with no base
success producer; one minimal private publication; a later purpose-specific
checkpoint projection; staged compiled lifecycle facts; fixed enum/scalar
diagnostics; and four sequential slices.

## 7. Boundary approval and reconnaissance scope

Boundary and narrow v2 reconnaissance scope were approved 2026-08-05. The
approved question-driven scope was:

| Approved area | Question | Explicit exclusion |
| --- | --- | --- |
| Smallest authoritative scanner-state owner and ordered loop | Is one transition mechanism separable, or coupled to competing orchestration that must be rejected? | Broad owner/planner/readiness/recovery orchestration and adjacent files reached only by references. |
| Admission boundary and normalized envelope/family declarations | Where do bounded ownership, explicit families, metadata, and refusal occur? | Provider decoding, generic event infrastructure, unrelated types, capacity tuning. |
| Canonical aggregate identity/merge/correction/source validation | What bounded state correctly supports duplicate/revision/precedence/context checks? | Feature formulas/ranking, REST pagination, hydration/no-print/recovery policy. |
| Clock/timer/sequence/watermark declarations | Which injected-time, nonregression, timer, order, and proof-gated progress behavior is separable? | Numeric readiness/freshness, scattered orchestration polling, replay artifact scheduling. |
| Immutable construction/publication identity/atomic replacement | Is coherent same-`T` mutation-isolated publication reusable? | Public API/HTTP/UI and unrelated broad copying. |
| Coherent checkpoint projection boundary | Is a separable immutable as-of handoff present? | Schema/contents/codec/storage/cadence/restart discovery/validation. |
| Compiled feature/state ownership seam | Can later correction-aware features remain inside one owner without runtime plugins? | Actual aggregate/TQ formulas, ranking, pressure, speculative extensions. |
| Focused tests for binding/order/disposition/time/merge/publication/bounds | Which claims have credible regression evidence and which only prove old orchestration? | Broad integration/live provider/API/UI/recovery/readiness suites, credentials, and duplicate layers. |

Provider I/O, owner/orchestration, recovery/readiness, formulas/ranking,
checkpoint contents/storage, API/UI, live calls/credentials, v1, and unrelated
packages were excluded.

The inspection remained inside that boundary. A ranking test incidentally
displayed beside an approved validation range was disclosed and was neither
used nor whitelisted. No live request or credential access occurred.

## 8. Version 2 reconnaissance, provenance, and whitelist

The predecessor commit was `5f92a151dd850002578a33a81ad90dea096c63b6` with
a heavily dirty worktree. Relevant provenance:

| Source | Inspected provenance | Decision and constraint |
| --- | --- | --- |
| `internal/scanner/state.go` (`symbolState`, `Engine`, construction/configuration) | worktree SHA-256 `a1742749f3be63e60e0d60dc8064a12cb8931307326174a2e90af5aeb40b6ae9`; uninspected HEAD `b898ed653ac0a6f365ece988f7642727f733c1a632aa4f3e95812e90c7b3216f` | Reject combined mutable reference/feature/ranking/recovery/checkpoint state and local bounds. New engine derives from concrete Component 1 binding. |
| `state.go` validation/apply/equality and clean `state_test.go` correction/window tests | test SHA-256 `fc3474a37f116d3300bf9f493c819d3fae9688d040b8fa66d701c9e73152ecfd` | Behavior evidence only: preserve explicit dispositions, exact equality, inclusive revision/strict finalization boundary. Reimplement atomic checks, binding/context/precedence, and never port `Apply`/`ApplyRecovery`. |
| `state.go` maintenance/evaluate/watermark | state hash above | Reject caller watermark, second maintenance boundary, multiple clocks, and same-`T` rejection. |
| `state.go` bounded tail plus `TestLongPathEquivalenceAfterPrefixFolding` | hashes above | Behavior evidence only: bounded mutable tail plus compact finalized state; do not reuse feature structure or caps. |
| `internal/scanner/types.go` | worktree SHA-256 `260b2203edd281585ce95c0edc0ba8114994f67f18a92ef8b46790b710127318`; uninspected HEAD `40cb42fdaf0ceddf5e4a8161ae9cfd83da85886c8c8eb886c33d17cc25edb283` | Reject types/giant snapshot. Preserve only the obligations to distinguish observation/window time and deep-copy references. |
| `internal/scanner/store.go` and `store_test.go` | SHA-256 `de33350eb3a7a40e3a41f4ed26253d03e3a6a7d5d9e94cd5ff828453089a7f1b` and `0ef94ffd32df1f54a167f04286e76e1e9f71de6271d1dcf4de2b6a04a78e1b01` | Adapt atomic replacement and mutation isolation to the minimal private publication; no public snapshot port. The isolated file set passed. |
| `internal/scanner/metrics.go` | SHA-256 `b198addd39a77343dcd5df49557036b328a868ebfb0e4096bcb3b9ed54bc9976` | Adapt fixed enum/scalar cardinality technique only; define new Component 2 reasons/accounting. |

No approved engine-local admission loop, injected clock/timer, binding envelope,
sequence assignment, lifecycle graph, commit proof, feature interface, coherent
checkpoint projection, or publication ID independent of `T` was found. Calls
and sequencing lived in excluded `internal/massive/owner.go`, which was not
opened. Untracked checkpoint/recovery files were not opened. Those negative
findings prevent architecture-by-predecessor and do not justify broader search.

**Exact approved implementation whitelist**

- `internal/scanner/state_test.go`: only input-construction helpers and the
  named correction-boundary, session/window-boundary, and long-path retention
  cases, adapted as behavior evidence; never a whole-file port.
- `internal/scanner/store.go`: atomic replacement behavior only.
- `internal/scanner/store_test.go`: deep-copy, reader-mutation, and concurrent
  publication isolation cases only.
- `internal/scanner/metrics.go`: fixed-cardinality reason/counter technique.
- `internal/scanner/state.go`: no implementation port; exact equality and
  bounded-tail observations are evidence to reimplement.

`types.go`, all `internal/massive`, owner/orchestration, recovery, readiness,
checkpoint, HTTP/API, UI, live-test, v1, and unrelated sources/fixtures are not
whitelisted. Slice allocation is narrower: S1 ports nothing; S2 may adapt only
the named aggregate test cases; S3 ports no time/watermark/lifecycle code; S4
may adapt scoped validation cases, store behavior/tests, and the metrics
technique. Any expansion requires owner review.
