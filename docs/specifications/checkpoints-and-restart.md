# Checkpoints and restart

**Status:** Completed Component 7 contract approved for sequential slice
implementation; implementation has not begun

**Owner boundary and reconnaissance approval:** Pre-approved 2026-08-07 by the
owner in the initiating Component 7 task, subject to the exact Sections 1–7
boundary, document map, and reconnaissance limits below

**Owner stable-interface exception:** None was needed. Component 6 passed final
component review on 2026-08-07 before version 2 reconnaissance began.

**Owner contract/reuse/test/slice-plan approval:** Approved 2026-08-07 as
written, including the exact V2 whitelist, fixtures/evidence, eight primary
proofs, three slices, required independent reviews, and
`advancement_mode: delegated`

**Advancement mode:** `delegated` for every slice and the final component gate;
no manual gate is designated

**C7-C11 program authority:** The same 2026-08-07 owner decision authorizes the
standing unattended program recorded in
[`AGENTS.md`](../../AGENTS.md#c7-through-c11-unattended-program-authority), the
[`implementation process`](../implementation-process.md#28-c7-through-c11-unattended-program),
and the [specification map](../specification-map.md#c7-c11-standing-program-decisions).
For C7, this adds no behavior, whitelist, proof, slice, target, or review change.
The current local host is the benchmark reference, and C7 makes no capacity
claim beyond its recorded evidence.

**Standing downstream decisions:** C8 uses conservative evidence-backed
thresholds and avoids false-ready results; C9 has no Tape Rate attention
threshold, uses continuous warm-up and current-rank-order restoration, permits
complete T/Q shedding, and protects aggregates first; C10 is a private
versioned read-only loopback HTTP API with explicit field status, publication
identity, and an explicit CORS allow-list, with public auth/TLS/hosting/cutover
deferred; C11 targets Chrome desktop and narrowly inspects the V2 UI
specification, code, assets, and focused tests for a high-fidelity adaptation.
The adaptation preserves useful layout, visual character, density, and
interactions subject to current semantics, accessibility, independent
deployment, and C10, and rejects browser-owned calculations/readiness, obsolete
state semantics, and backend coupling. No credentialed live-provider
observation is authorized, and program completion is a locally verified
private release candidate rather than live validation or production deployment.

**Controlling Phase 1 requirements:** `PG-OPS-01`, `PG-OPS-02`,
`PG-REPLAY-01`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-03`, `ARCH-FLOW-04`, `DTE-SESSION-02`, `DTE-CLOCK-05`,
`DTE-WINDOW-01`, `DTE-EVENT-01`, `DTE-EVENT-04`,
`DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-CHECKPOINT-01`,
`DTE-CHECKPOINT-02`, `DTE-CHECKPOINT-03`, `DTE-REJECT-01`,
`LIFE-MODEL-01`, `LIFE-MODEL-02`, `LIFE-MODEL-04`, `LIFE-INIT-03`,
`LIFE-INIT-04`, `LIFE-INIT-05`, `LIFE-HYDRATE-01`,
`LIFE-HYDRATE-02`, `LIFE-HYDRATE-03`, `LIFE-HYDRATE-04`,
`LIFE-HYDRATE-05`, `LIFE-HYDRATE-06`, `LIFE-LIVE-02`,
`LIFE-LIVE-04`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`, `LIFE-END-02`,
`LIFE-END-03`, `LIFE-PUBLISH-02`, `LIFE-T01`, `LIFE-T02`, `LIFE-T03`,
`LIFE-T04`, `LIFE-T06`, `LIFE-T08`, `LIFE-T11`, `LIFE-T12`,
`LIFE-T15`, and `LIFE-T29`

**Approved dependencies:** the finally accepted Component 1 immutable
[`reference.Binding`](reference-data-and-session-binding.md), Component 2
[`ScannerStateEngine` and canonical-state contract](scanner-state-engine-and-canonical-state.md),
Component 3
[`aggregate feature/evaluator contract`](aggregate-features-qualification-ranking-and-accounting.md),
Component 4 [`aggregate replay contract`](aggregate-replay.md), and Component 5
[`Massive live-adapter contract`](massive-live-adapter.md). The owner-approved
finally accepted Component 6
[`aggregate hydration/recovery contract`](aggregate-rest-hydration-and-recovery.md)
supplies the installed-`T0`/`[T0,R)` interface.

## Contract document map

The modular layout separates the coherent engine-owned state boundary from the
external durable-storage boundary and from end-to-end proof/delivery. All files
listed here form one Component 7 contract and pass the same approval gates.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Outcome, single ownership boundary, explicit non-scope, cross-cutting invariants, Sections 1–7, routing, approvals, and sole delivery ledger | All controlling Phase 1 IDs; Sections 1–7 | Every Component 7 task | Components 1–6 interfaces named above |
| [Checkpoint state and installation](checkpoints-and-restart/checkpoint-state-and-installation.md) | Exact coherent projection, schema semantics, compatibility/validation, atomic install, live/replay restart handoff, and engine-owned checkpoint facts | Sections 9–14; `C7-STATE-01`, `C7-INSTALL-01`, `C7-LIVE-01`, `C7-REPLAY-01`; `P-C7-STATE`, `P-C7-INSTALL`, `P-C7-LIVE`, `P-C7-REPLAY`; `C7-S1`, `C7-S3` | Projection, semantic validation, installation, or restart-state work | Parent; Components 1–4 and 6 |
| [Codec, storage, and cadence](checkpoints-and-restart/codec-storage-and-cadence.md) | Bounded encoding/decoding, durable atomic storage and discovery, asynchronous write-result identity, cadence, final-checkpoint policy, and restart objective | Sections 9–14; `C7-CODEC-01`, `C7-STORE-01`, `C7-CADENCE-01`, `C7-OBJECTIVE-01`; `P-C7-CODEC`, `P-C7-STORE`, `P-C7-CADENCE`, `P-C7-OBJECTIVE`; `C7-S2`, `C7-S3` | Codec, filesystem, persistence failure, cadence, or restart-target work | Parent; preceding state/install detail; Components 1–3 |
| [Proof and delivery plan](checkpoints-and-restart/proof-and-delivery-plan.md) | Exact reconnaissance/reuse ledger, complete requirement/proof allocation, sequential slice plan, independent-review triggers, unattended goal-orchestration protocol, discretion, completed-contract checklist, and drift audit | Sections 8 and 15–19; all eight Component 7 requirements/proofs; `C7-S1`–`C7-S3` | Completed-contract approval, assignments, acceptance, and final review | Parent and both preceding details; Components 1–6 |

**Layout:** Modular contract with this parent and the three routed detail specs
above. Projection/install, external durable storage, and end-to-end restart
equivalence are independently reviewable trust boundaries and require three
sequential slices. This parent is about 3,400 words, above the
template's approximate 2,500-word routing alarm, because its advance-approved
Sections 1–7 must keep the exact boundary, reconnaissance limits, dependency
handoffs, and timing restriction together. All reconnaissance, detailed
requirements, trust/proof ledgers, and delivery material are routed out rather
than extending it.

**Routing rule:** Any requirement, evidence decision, proof, slice, or task not
unambiguously routed by this table stops for a parent-map correction. Models do
not guess among detail specs or load unrelated dependency details.

**Contract-wide coverage and acceptance:** Sections 1–7 below remain the
approved boundary. The routed details now own the exact eight-requirement,
eight-primary-proof, three-slice contract and the completed-contract checklist.
The owner-approved set authorizes its sequential slices only after a future goal
is explicitly created; this preparation task does not start implementation.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `contract_approved` | Owner approved the complete four-document contract as written on 2026-08-07, including the exact V2 whitelist, fixtures/evidence, eight proofs, three slices, required reviews, and `advancement_mode: delegated` | 2026-08-07; governance commit records the approval | Begin `C7-S1` only when the future C7-C11 goal is explicitly created and launched |
| `C7-S1` | `pending` | `P-C7-STATE`, `P-C7-INSTALL`; narrow ownership/atomic-install review required | 2026-08-07 | Wait for explicit future goal launch |
| `C7-S2` | `pending` | `P-C7-CODEC`, `P-C7-STORE`, `P-C7-CADENCE`; narrow persistence/atomicity/concurrency review required | 2026-08-07 | Wait for accepted `C7-S1` |
| `C7-S3` | `pending` | `P-C7-LIVE`, `P-C7-REPLAY`, `P-C7-OBJECTIVE`; narrow cross-component identity/restart-equivalence review required | 2026-08-07 | Wait for accepted `C7-S2` |
| Final component review | `pending` | Mandatory separate read-only review after `C7-S1`–`C7-S3` pass | 2026-08-07 | Wait for all slices |

## 1. Outcome and user consequence

Component 7 makes an ordinary process restart reproduce the prior trustworthy
aggregate-derived scanner state at one committed boundary `T0`, then retrieve
only `[T0,R)` before returning to the same ordinary evaluator. It prevents a
restart from silently losing a valid qualification latch, reviving corrected
evidence, combining feature state from different times, or requiring a normal
full-session rebuild.

For the operator, a compatible complete checkpoint shortens restart while
remaining visibly noncurrent until catch-up evidence is sufficient. A missing,
incompatible, or corrupt checkpoint selects fresh hydration with a bounded
reason and no partial state. Projection, encoding, or write failure leaves live
ranking advancing and retains the previous complete checkpoint.

## 2. Scope and explicit non-scope

**In scope**

- Define one versioned, bounded checkpoint artifact and its exact compatibility
  identity for one immutable Component 1 session binding.
- Define the minimum sufficient canonical aggregate, coverage/no-print/unknown/
  conflict, feature, qualification, accounting-support, correction-tail, and
  committed state needed to reproduce Components 2–3 at exactly `T0`.
- Add one engine-owned coherent projection operation that copies immutable
  checkpoint state only after the ordinary commit/evaluation boundary is
  complete.
- Validate structural bounds, binding/schema compatibility, semantic
  invariants, completeness, and cross-field `T0` coherence before any restored
  value becomes canonical.
- Atomically install one validated checkpoint in a new engine, or install none;
  expose the resulting `T0` to Component 6's already-defined `[T0,R)` startup
  plan and the ordinary Component 3 evaluator.
- Encode, decode, discover, select, write, and atomically replace local
  checkpoint files outside the ordered engine path while retaining the last
  complete usable artifact on failure.
- Set an evidenced cadence and normal restart target, bound in-flight work and
  retained artifacts, identify stale write outcomes, and return bounded
  checkpoint status/reasons to the engine.
- Support the Phase 1 optional compatible-checkpoint start for aggregate replay
  without changing replay ordering or transport claims.

**Not in scope**

- Session/reference acquisition or binding construction; Component 1 owns it.
- Canonical aggregate identity/merge/correction rules, committed-time meaning,
  feature mathematics, qualification, accounting, ranking, or another
  evaluator; Components 2–3 own them.
- Replay artifact acquisition/format/order/clock, live WebSocket epochs or
  acknowledgements, REST pagination, hydration work, no-print proof, ingress
  fencing, or the mechanics of `[T0,R)` catch-up; Components 4–6 own them.
- Production freshness/readiness/capacity thresholds, retry and shutdown
  budgets, supervisor policy, or HTTP readiness mapping; Component 8 owns them.
  Component 7 owns only its evidenced checkpoint cadence/restart objective,
  hard safety bounds, and facts consumed by later operations policy.
- T/Q membership, causal coverage, ephemeral T/Q windows, or persistence of
  any T/Q measurement; Component 9 owns new-epoch warm-up and coverage.
- Public API schema, UI behavior, database, remote/object storage, replication,
  journal/event sourcing, generic serialization framework, plugin system,
  service split, credentials, or live provider access.

## 3. Ownership and dependencies

The single Component 7 ownership boundary begins when the engine seals an
immutable projection from one fully committed/evaluated boundary `T0`, or when
initialization supplies one bounded candidate checkpoint for validation. It
ends when an external writer returns a matching terminal outcome, or when the
engine atomically installs the entire validated state and exposes installed
`T0` as the initial boundary for live or replay continuation.

`ScannerStateEngine` remains the sole mutable owner of the active binding,
canonical state, feature/qualification support, lifecycle, `T`, evaluation,
accounting, and checkpoint cadence/request identity. The projector is one
explicit engine-owned stage. Codec, loader, and writer code own only bounded
immutable bytes, request-local filesystem state, and terminal facts; they
receive no writable engine reference and cannot decide compatibility,
canonical success, lifecycle, ranking currentness, or committed time.

Component 7 may choose a compact persisted representation, but it cannot
reinterpret dependency semantics. Installation must reconstruct the exact
dependency-owned state those contracts require or reject the artifact before
visibility. Component 6 alone plans and reconciles catch-up after a valid `T0`;
Component 7 cannot treat file load as present-day coverage.

## 4. Settled Phase 1 semantic boundary

| Boundary item | Settled meaning version 2 cannot change | Controlling Phase 1 IDs |
| --- | --- | --- |
| Binding and lifetime | A checkpoint belongs to one exact immutable session binding and one engine-run mode-compatible continuation. A different trading date, session bounds, universe identity, required prior session, prior-close policy/data identity, or incompatible schema cannot be installed. | `DTE-SESSION-02`, `LIFE-MODEL-01`, `LIFE-INIT-03`, `LIFE-REPLAY-01` |
| Coherent cutoff | `T0` is a committed aggregate watermark. The artifact represents all restart-required mutable contributions exactly as of `[S,T0)`—canonical identities, coverage/no-print/unknown/conflict evidence, derived sufficient state, correction status, and accounting support—or explicitly fails validation. `checkpoint_created_at >= T0` is persistence metadata; creation time, file modification time, `R`, and generated time cannot substitute for `T0`. | `PG-OPS-01`, `DTE-CLOCK-05`, `DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-03` |
| Restart equivalence | Equal binding, complete checkpoint state at `T0`, and the same continuation facts must reproduce the same dependency-owned canonical and aggregate-derived evaluation as uninterrupted processing. Current selected rows are not checkpoint authority; the ordinary evaluator regenerates output. | `PG-OPS-01`, `PG-OPS-02`, `DTE-COMMIT-02`, `DTE-COMMIT-03`, `DTE-CHECKPOINT-01` |
| Projection ownership | Only the engine may seal a checkpoint view, after the committed/evaluated transition is coherent. The view is immutable and detached from live maps; no separately sampled state or writer read can form a checkpoint. | `ARCH-OWN-01`–`ARCH-OWN-04`, `DTE-CHECKPOINT-01`, `LIFE-LIVE-04` |
| Persistence boundary | Encoding, self-validation, temporary-file writing, durability operations, and atomic replacement occur outside the ordered engine path. Requests and terminal results have versioned system envelopes, monotonic system positions, and exact request/binding identity. A matching terminal result is an operational fact only; success or failure cannot advance `T`, mutate canonical state, or block ordinary evaluation. | `ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-CHECKPOINT-03`, `LIFE-LIVE-02`, `LIFE-LIVE-04` |
| Atomic startup install | Compatibility and complete structural/semantic validation finish before canonical visibility. A rejected candidate installs nothing. Missing/incompatible/corrupt state ordinarily falls back to an older compatible checkpoint or fresh hydration; only ambiguity that also invalidates the binding uses global suppression. | `PG-OBS-03`, `DTE-REJECT-01`, `LIFE-INIT-03`, `LIFE-T04` |
| Live continuation | A valid installed checkpoint selects `checkpoint_catchup`; after the current aggregate acknowledgement fixes `R`, Component 6 covers `[T0,R)` and reconciles the live tail through its fence. The aggregate with `window_start=T0` is continuation input, not checkpoint state. | `DTE-WINDOW-01`, `DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-02`, `LIFE-HYDRATE-05` |
| T/Q reset | Connection epoch, subscription membership, causal coverage, and ephemeral T/Q feature windows never survive restart, even if aggregate qualification and history do. | `DTE-CHECKPOINT-02`, `LIFE-INIT-03`, `LIFE-PUBLISH-02` |
| Replay continuation | Replay may install one compatible coherent checkpoint at `T0` and continue from its exact artifact cutoff; rejection falls back to `S` only when the replay artifact contains the complete required continuation. Live epochs/latency/TQ claims remain absent. | `PG-REPLAY-01`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`, `LIFE-T03` |
| End and final projection | A controlled stop may request a final view only from an already coherent committed boundary and cannot wait indefinitely for disk I/O. Late writer results cannot mutate an ended engine. | `LIFE-END-02`, `LIFE-END-03`, `LIFE-T29` |

This component introduces no new product rule, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, changed time-window meaning,
or duplicated responsibility.

## 5. Unresolved questions and evidence needs

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| What exact sufficient state must be projected for Components 2–3, including compacted presence/conflict facts, correction tail, folded Activity targets, feature dirty/fold metadata, qualification proof/dirty state, final latch, and classification support? | Phase 1 fixes coherent restart equivalence; approved component contracts state reproduction obligations but intentionally do not choose checkpoint fields. | Dependency-contract inventory, current accepted implementation invariants after C6 final review, and scoped V2 projection/round-trip evidence. | Exact semantic schema and construction/validation invariants without persisting redundant evaluator output. |
| Which values should be persisted directly versus deterministically rebuilt during atomic installation? | Both can satisfy equivalence if complete, bounded, and validated; Phase 1 does not choose the tradeoff. | Size/latency evidence, dependency constructors, V2 restore behavior, and differential uninterrupted-versus-restart proofs. | Install representation, validation order, and bounded startup cost. |
| What artifact version/compatibility policy and integrity envelope safely distinguish unsupported, corrupt, truncated, and semantically inconsistent checkpoints? | Schema evolution, checksum/framing, numeric bounds, and compatibility are delegated. | Scoped V2 codecs/fixtures/tests plus standard-library format capabilities and construction invariants. | Version envelope, decoder limits, compatibility matrix, rejection reasons, and migration policy if any. |
| How should discovery choose among current, previous, temporary, corrupt, and incompatible local artifacts? | Phase 1 requires a compatible checkpoint or clean fallback but does not define filenames, retention, or candidate order. | Scoped V2 storage/load tests and filesystem guarantees. | Deterministic discovery/selection and bounded retention policy. |
| What exact temporary-write, flush, rename, directory-durability, permission, and stale-result protocol preserves the last complete checkpoint across process or write failure? | Architecture requires atomic replacement but delegates concrete local storage mechanics. | Scoped V2 writer tests/fixtures, target-platform filesystem semantics, and failure injection. | Durable writer contract and matching request/result identity. |
| What in-flight/coalescing rule keeps projection and writer work bounded when `T` advances faster than persistence? | Phase 1 prohibits blocking/unbounded work but does not choose skip, replace-pending, or finite queue behavior. | Measured artifact size/encode/write latency and V2 cadence/backpressure evidence. | Hard in-flight bound, cadence eligibility, stale-result handling, and observable skipped/failed outcomes. |
| What cadence and restart target satisfy normal production continuity, and is a final controlled-stop checkpoint useful within the shutdown boundary? | `PG-OPS-01` requires evidenced values and rejects approximately 130 seconds of normal fresh reconstruction but supplies no numeric target. | V2 timing/configuration evidence, post-C6 representative offline benchmark data, artifact size/write cost, and later Component 8 capacity context. | Concrete cadence, restart objective, benchmark premise, and final-view policy. |
| Which live and replay restart scenarios distinguish correct installation/catch-up from an apparently successful mixed or incomplete restore? | Phase 1 states the outcome but delegates primary proof structure. | Scoped V2 restart tests/fixtures, C3 differential oracles, Component 6 accepted catch-up proofs, and replay continuation fixtures. | Minimal primary-proof families and provisional slice boundaries. |

These are component-local representation, storage, boundedness, and proof
questions. They do not reopen Phase 1 time, state, feature, ranking, recovery,
or lifecycle semantics.

All eight questions are resolved by the detailed state/install and
codec/storage/cadence contracts and the Section 8 evidence ledger. No evidence
question remains open before completed-contract review; a later implementation
finding that invalidates one of those resolutions triggers the Section 17
owner-escalation gate rather than implementer discretion.

## 6. Approved and completed version 2 reconnaissance scope

The only predecessor is
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Version 1,
credentials, and live provider requests remain prohibited. This scope is
approved and was used only after Component 6 passed final
review. Exact findings, hashes, and reuse decisions are routed to
[Section 8](checkpoints-and-restart/proof-and-delivery-plan.md#8-version-2-reconnaissance-and-reuse-assessment).

| Proposed code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| Checkpoint model, projection, codec, decoder, validation, and focused unit fixtures | Which persisted fields and invariants were useful; where did V2 permit mixed-as-of state, omit restart-required state, trust derived output, or lack hard decode bounds? | Broad scanner state/ranking/feature implementation except exact declarations reached from checkpoint code; no assumption that V2 fields or schema are authoritative. |
| Local checkpoint discovery, loader, writer, atomic-replacement, and focused filesystem-failure tests | What candidate selection, corruption containment, temporary-file, rename/durability, permission, and previous-checkpoint preservation behavior has credible evidence? | Cache/reference storage, replay artifacts, remote/object storage, database, deployment, backup, and unrelated filesystem helpers. |
| Exact checkpoint restart/install and checkpoint-catch-up integration tests and their local fake fixtures | Which tests distinguish atomic valid install, invalid fallback, same-state reproduction, `[T0,R)` boundary behavior, live-tail reconciliation, and empty/sparse continuation? | V2 owner/planner/readiness/recovery architecture and production source ports; inspect only the minimum test-local call path needed to understand evidence. |
| Exact feature/qualification checkpoint round-trip tests and state declarations they directly exercise | Which correction-tail, Activity, range, coverage/conflict, dirty/fold, proof, and latch states were preserved or lost across restart? | General formula/ranking reconnaissance already owned by Component 3 and unlisted feature tests. |
| Checkpoint cadence/configuration declarations plus focused size, latency, or restart measurements if present | What empirical premise did V2 use, and is it strong enough to inform a new cadence/restart target or only identify a benchmark gap? | Readiness thresholds, operations orchestration, production logs, credentials, live observations, and broad performance suites. |
| Optional replay-from-checkpoint tests/fixtures, only if directly present in checkpoint or replay packages | Does V2 contain useful evidence for exact artifact continuation at `T0` without live transport claims? | General replay downloader/compiler/source code already owned by Component 4. |

Reconnaissance will record exact paths/functions/tests/fixtures, commit and file
hashes where provenance matters, behavior strength, coupling to remove, reuse
decision, and required proof in the routed proof/delivery detail. Discovery
that requires an unrelated source area, new product rule, changed dependency
interface, or broader orchestration inspection stops for owner review.

### 6.1 Initial proof and slicing boundaries

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| Coherent projection and in-memory round trip | `PG-OPS-01`, `DTE-CHECKPOINT-01`, `DTE-COMMIT-02`, `DTE-COMMIT-03` | One owner-sealed `T0` projection contains sufficient bounded dependency state; install plus identical continuation matches an uninterrupted oracle, including corrected/dirty/folded/final-latch cases. | Exact state inventory, rebuild choices, and V2 round-trip limitations. |
| Compatibility/semantic validation and atomic install | `DTE-SESSION-02`, `DTE-CHECKPOINT-01`–`03`, `LIFE-INIT-03` | Wrong binding/schema, truncation, corrupt bounds, mixed `T0`, or incomplete state cannot partially mutate a new engine or create false success. | Exact envelope, invariant matrix, and corrupt/incompatible fixtures. |
| Bounded codec and durable atomic replacement | `ARCH-OWN-02`, `ARCH-FLOW-01`, `ARCH-FLOW-04`, `LIFE-LIVE-04` | Encode/decode and filesystem failure never expose a partial artifact, lose the last complete checkpoint, alias engine state, block `T`, or accept a stale writer result as current success. | Format/storage choice, filesystem semantics, V2 failure evidence, and artifact-size bounds. |
| Live checkpoint restart composition | `PG-OPS-02`, `DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-02`–`06` | Valid install at `T0`, exact `[T0,R)` hydration, concurrent live tail, terminal fence, and the ordinary evaluator reproduce uninterrupted aggregate-derived output while T/Q starts empty. | Component 6 final accepted interface/evidence and exact cross-component fixture. |
| Replay continuation and invalid fallback | `PG-REPLAY-01`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02` | A compatible checkpoint resumes at the exact replay cutoff; rejection restarts at `S` only with complete artifact evidence; playback scheduling cannot alter the result. | Scoped replay/checkpoint evidence and Component 4 fixture suitability. |
| Cadence, in-flight bound, and restart objective | `PG-OPS-01`, `ARCH-FLOW-01`, `LIFE-LIVE-04`, `LIFE-END-02` | The chosen policy is finite, does not impede ordinary evaluation, and meets a measured normal restart objective materially better than fresh reconstruction. | Representative post-C6 benchmark and any credible V2 timing evidence. |

**Final delivery assessment:** Three sequential implementation slices.

**Reason and slice outcomes:** `C7-S1` establishes coherent engine projection,
semantic validation, and atomic in-memory install. `C7-S2` adds the independently
provable persisted trust boundary: bounded codec, local latest/previous store,
asynchronous write results, and cadence. `C7-S3` composes those accepted paths
with Component 6 live catch-up and Component 4 replay continuation, then proves
the restart objective. Each slice leaves usable behavior and extends rather
than replaces the preceding owner/path.

## 7. Boundary-approval checkpoint

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, ownership, dependencies, scope, and non-scope are unambiguous.
- [x] Settled inputs/outputs/state and invariants prevent version 2 from
      changing the architecture.
- [x] Unresolved questions are genuinely delegated representation, storage,
      boundedness, cadence, target, and proof details.
- [x] Future version 2 reconnaissance is narrow, question-driven, and excludes
      broad predecessor orchestration.
- [x] No version 2 code, tests, or fixtures were opened while preparing this
      skeleton.
- [x] Initial likely proof boundaries are identified without a broad duplicate
      test matrix.
- [x] The likely need for multiple sequential slices and their dependency order
      are explicit and provisional.

**Owner decision:** Advance-approved 2026-08-07 for the exact boundary,
document map, and reconnaissance scope above. Component 6 passed final review
before that scope was exercised.

**Detailed-contract gate:** Passed by explicit owner approval on 2026-08-07.
Sections 8–19 are complete in the routed details, and the exact V2 whitelist,
proof allocation, three-slice plan, required reviews, and delegated advancement
are fixed. No implementation begins during this preparation task; the future
C7-C11 goal starts at `C7-S1`.
