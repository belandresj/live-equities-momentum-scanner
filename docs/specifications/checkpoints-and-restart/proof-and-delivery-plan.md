# Checkpoints and restart — proof and delivery plan

**Parent contract:** [checkpoints-and-restart.md](../checkpoints-and-restart.md)
**Normative responsibility:** Exact reconnaissance/reuse ledger, complete
requirement/proof allocation, sequential slice plan, independent-review
triggers, unattended goal-orchestration protocol, discretion, completed-
contract checklist, and drift audit
**Controlling requirements:** All Phase 1 and Component 7 requirements routed
by the parent
**Allocated slices:** `C7-S1`, `C7-S2`, `C7-S3`
**Document dependencies:** Parent; [checkpoint state and installation](checkpoint-state-and-installation.md);
[codec, storage, and cadence](codec-storage-and-cadence.md); Components 1–6 as
routed by the parent
**Approval state:** Inherits the parent contract approval; not independently approved
**Delivery state:** See the authoritative parent delivery-state ledger; do not copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance began only after Component 6 passed final review and stayed
inside the owner-approved scope. The predecessor HEAD was
`5f92a151dd850002578a33a81ad90dea096c63b6`; its worktree was heavily dirty and
all relevant checkpoint files below were untracked, so the worktree SHA-256 is
the exact provenance rather than HEAD content.

| Exact source path and function/type/test/fixture | Worktree SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `internal/scanner/checkpoint_phase2b.go`: payload/dependency types, `ExportCheckpoint`, detached builder, payload validation, `InstallCheckpoint` | `f70387588514991360fd9269f99023760260068bb5520228e37d6cb6932b8b9e` | Sorted semantic records, explicit field status, count preflight, detached building, and all-or-none install are useful. V2 deliberately sealed at `created-H`, persisted derived/optional state only, omitted current canonical/coverage/folded/proof state, and reconstructed a mark as synthetic flat OHLC with fabricated volume/ATS. | `adapt` detached-builder/validation technique; `reject` schema and restored values | Use current C2/C3 sufficient state at exact `T0`; no opaque 16-minute prefix, optional success-bearing omission, fabricated aggregate, old owner/token, or cached ranking authority | `P-C7-STATE`, `P-C7-INSTALL` |
| `internal/scanner/phase2b_checkpoint_bootstrap_test.go`: `TestPhase2BCheckpointDetachedAtomicInstall`, `TestPhase2BCheckpointPartialInstallReportIsGlobal`, persisted dependency/activity boundary cases | `47216b4b6b0b7c3f884192b4f63e3b73d7970a8444ffe9387c12e22668045a68` | Strong detached/atomic and several feature-boundary cases, but expectations accept V2's incomplete derived schema and warming omissions rather than exact restart equivalence. | `behavior evidence` | Retain atomic rejection and boundary mutations; replace payload/result assertions with current uninterrupted-oracle equality and exact complete state | `P-C7-STATE`, `P-C7-INSTALL` |
| `internal/massive/checkpoint_phase2b.go`: strict decoder, `CheckpointStore`, `CheckpointWriter`, manifest/load/write/cleanup paths | `086d307a3e100aa87230d65858495c66b3a56e8e7cf9c912f44bb6dab21fe914` | Credible standard-library strict JSON, SHA-256, limited streaming, latest/previous manifest, reopen validation, temp/sync/rename/directory-sync publication, bounded cleanup, one coalescing slot, and terminal step facts. It is coupled to predecessor scanner tokens/owner polling and assumes 64 MiB/5 seconds as production defaults. | `adapt` algorithms and bounds evidence | Replace tokens/polling with C7 request/result identity and engine-owned outstanding accounting; make artifact/operation configuration explicit under C7 absolute caps; use new schema | `P-C7-CODEC`, `P-C7-STORE`, `P-C7-CADENCE` |
| `internal/massive/phase2b_checkpoint_test.go`: `TestPhase2BCheckpointMaximumShape192MiB`, `TestPhase2BCheckpointLatestAndPreviousFallback`, duplicate-key, detached-install/fence, exact-step crash authority, streaming cancellation, cleanup/plateau tests | `869842fbcf05210c9cb5bbf254b1e4e0491ede46305342a51f62fb06de0b5775` | Strong maximum-shape allocation, nested duplicate-key, fallback, exact durability-step, cancellation, and bounded-retention evidence. Fresh-bootstrap/readiness assertions use obsolete orchestration and are outside reuse. | `behavior evidence`; adapt named storage/codec cases only | Rebuild fixtures with current schema and loader/engine seam; reject V2 owner/readiness/fresh-bootstrap outcomes | `P-C7-CODEC`, `P-C7-STORE`, `P-C7-CADENCE` |
| `docs/refactor/phase-2b-checkpoint-bootstrap-implementation-spec.md`: Sections 6–7 and 16 only | `d5eb0c52594d180f48a71588bc9ea946769486c6abc053edcd643f6593929f07` | Records the 30-second cadence, 5-second local-operation safety budget, latest/previous policy, strict bounds, and 5-minute catch-up ceiling. The last is explicitly a safety ceiling, not a readiness SLA; no measured normal restart target exists. | `behavior evidence` only | Use cadence/local budget as reference premises; add the missing measured C7 target and reject V2's 16-minute checkpoint lag/5-minute normality implication | `P-C7-OBJECTIVE` |
| Replay/checkpoint packages and named files in the approved optional scope | no dedicated source found | V2 contains no direct replay-from-checkpoint proof suitable for reuse. | `none` | Use accepted Component 4 artifact identity/order plus a new C7 fixture | `P-C7-REPLAY` |

The V2 `phase2b_race_enabled_test.go` and `phase2b_race_disabled_test.go` files
were identified but contain no distinct checkpoint proof needed by this
contract and are not whitelisted. No V2 fixture or live observation is needed.

**Proposed implementation whitelist:** only the exact functions/types and named
tests in the first four rows above. The refactor document is evidence, not a
source port. No other V2 file, no version 1 source, and no credentials/live
provider access is approved. Implementers may adapt algorithms or test inputs;
they may not import the predecessor module or copy its scanner-owner/readiness
architecture.

## 15. Primary proof allocation

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result and proof limitation | Approved fixture/evidence | Slice |
| --- | --- | --- | --- | --- | --- |
| `C7-STATE-01` | `P-C7-STATE`: committed-`T0` semantic projection and in-memory round-trip differential | Projection contains every restart-required C2/C3 input exactly at `T0` across corrected tail, present/absent/conflict, stalled forward state, ranges, Activity mutable/folded nonaligned state, provisional/final qualification, and invalid support. Counterexample: copying post-`T0` forward state or omitting dirty/folded state still yields plausible rows. | Installed candidate plus ordinary evaluator equals an uninterrupted full-history oracle at `T0` and after identical continuations; byte/view mutation cannot affect engine. Does not prove bytes/files, live provider, or capacity. | Accepted C2/C3 distinguishing regressions plus adapted V2 detached case | `C7-S1` |
| `C7-INSTALL-01` | `P-C7-INSTALL`: semantic mutation matrix and atomic-install ownership trace | Exact binding/schema/count/time/value/provenance/invariant validation and regenerated evaluator precede one owner apply. Counterexamples: valid checksum with mixed `T0`, synthetic mark, partial symbol install, cached result trusted, or old epoch position rejecting a valid new live correction. | Every mutation installs nothing and leaves a fresh engine observably unchanged; valid candidate installs all state/`T0`, duplicate is stable, and new-epoch correction is accepted under C2 rules. Inspection proves one install/apply path. Does not prove disk integrity or C6 catch-up. | Product/DTE invariants; adapted V2 atomic failure cases | `C7-S1` |
| `C7-CODEC-01` | `P-C7-CODEC`: golden deterministic round trip, strict corruption corpus, cancellation, and maximum-shape allocation trace | Complete schema encodes deterministically and bounded decode rejects truncation, trailing value, unknown/duplicate fields, length/count overflow, digest mismatch, nonfinite/numeric/state preflight, and cancellation. Counterexample: malformed bytes allocate from attacker length or default into a valid candidate. | Equal images produce equal payload bytes; every corrupt case produces no candidate; measured transient/retained allocation plateaus within declared bounds. Does not prove filesystem durability or engine semantic install. | Adapted V2 strict JSON/maximum-shape cases; current schema bounds | `C7-S2` |
| `C7-STORE-01` | `P-C7-STORE`: latest/previous discovery plus exact durability-step fault injection | Only manifest-authorized complete generations load; every failure before manifest publication retains old authority, and post-publication ambiguity cannot expose partial content. Counterexamples: orphan/mtime authority, symlink escape, corrupt latest hiding valid previous, or failed cleanup deleting referenced state. | Candidate selection and exact terminal step match the protocol for every injection; latest/previous contents/checksums remain correct; unsafe paths fail persistence-only. Does not prove power-loss behavior beyond target filesystem primitives or live deployment storage. | Adapted V2 fallback, path, exact-step, cancellation, cleanup tests | `C7-S2` |
| `C7-CADENCE-01` | `P-C7-CADENCE`: engine/writer concurrency and terminal-accounting trace | Session-aligned 30-second committed-time eligibility never blocks evaluation; one writing plus one pending is the maximum; replacement terminals the old pending request; stale/duplicate/late results cannot become current success. Counterexample: writer backlog aliases engine state, drops a request from accounting, or file success advances `T`. | Under paused writer and advancing/correcting engine, `submitted=in_progress+pending+terminal`, occupancy bounds hold, ordinary `T` advances, views stay immutable, and end/cancel terminates or fences every result. Does not prove wall latency. | V2 coalescing technique; Component 2 admission/accounting proof style | `C7-S2` |
| `C7-LIVE-01` | `P-C7-LIVE`: Components 1–7 checkpoint-restart equivalence trace | Valid install at `T0`, new C5 ack `R`, exact C6 `[T0,R)` value/empty work, overlapping live correction, fence, and ordinary C3 evaluator equal uninterrupted processing; T/Q/old epochs are absent. Counterexample: load alone claims current, catch-up begins at `T0+1s`, history overwrites prefix, empty fabricates a bar, or dead epoch rejects live correction. | Canonical identities, coverage classes, feature/qualification state, population counters, rows, lifecycle, and publication match the uninterrupted oracle after catch-up; T/Q state is empty. No live-provider/SLA/readiness claim. | Accepted C1–C6 fake socket/HTTP/engine seams; new C7 checkpoint fixture | `C7-S3` |
| `C7-REPLAY-01` | `P-C7-REPLAY`: checkpointed versus full replay differential at two playback paces | Exact artifact cutoff and ordinary C4 delivery reproduce full replay; repeated/pre-`T0` or gapped continuation cannot pass. Counterexample: checkpoint plus duplicated prefix double-counts Activity/qualification while final ranking appears plausible. | Full-from-`S` and checkpoint-from-`T0` outputs/counters equal at logical checkpoints and end at both paces; bad cutoff rejects and complete-from-`S` fallback succeeds. Does not prove live epochs, latency, or T/Q. | Accepted C4 artifact/runner fixtures plus C7 state fixture; no V2 reuse | `C7-S3` |
| `C7-OBJECTIVE-01` | `P-C7-OBJECTIVE`: recorded segmented five-run reference benchmark with fresh control | Compatible local load/install is <=5s and 5,500-symbol/30-second-gap restart is <=60s and faster than fresh control. Counterexample: benchmark silently omits success-bearing state, mocks C6 work, uses a tiny universe, or reports only a best run. | Fixture/cardinalities/bytes/host and five projection-through-catch-up segments are recorded; every run meets both targets and checkpoint median beats fresh median. Does not prove live-provider or 100,000-symbol deployed capacity. | V2 timing/cadence/5,500-symbol evidence; product 130s rejection; actual C7 implementation | `C7-S3` |

No secondary proof layer duplicates these claims. `P-C7-LIVE` is intentionally
cross-component rather than a second install proof: it establishes C5/C6
handoff and current-epoch reconciliation that `P-C7-INSTALL` cannot.

## 16. Sequential implementation-slice plan

Three slices are required because in-memory canonical ownership, persisted
untrusted bytes/files/concurrency, and cross-component live/replay restart are
distinct trust and proof boundaries. Combining them would make atomic-state
defects, filesystem defects, and continuation-interface defects difficult to
review or roll back independently.

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies / entry state | Allowed ownership or packages | Approved V2 whitelist | Acceptance record | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `C7-S1` | A bound engine can project one complete immutable semantic image at committed `T0`, validate/install it all-or-nothing in a fresh engine, regenerate the ordinary evaluation, and accept valid new-run correction authority | `C7-STATE-01` / `P-C7-STATE`; `C7-INSTALL-01` / `P-C7-INSTALL` | Finally accepted C1–C6; completed C7 approval; no C7 code | `internal/engine` and a C7-private semantic model under `internal/checkpoint` if needed; no filesystem, goroutine, CLI, or provider code | V2 scanner checkpoint source and named scanner tests only | Exact state coverage/diff; install mutation results; owner/apply inspection; affected engine tests and race; repository compile/test/vet; construction/success/failure walkthrough; required review result; next-slice validity | On-disk codec/store/writer/cadence; live/replay composition; timing target |
| `C7-S2` | The S1 image has one strict bounded JSON codec and private latest/previous local store; one asynchronous writer obeys cadence, coalescing, exact request accounting, and failure containment without blocking engine progress | `C7-CODEC-01` / `P-C7-CODEC`; `C7-STORE-01` / `P-C7-STORE`; `C7-CADENCE-01` / `P-C7-CADENCE` | Accepted `C7-S1`; its semantic schema/install interface is fixed | C7-owned codec/store/writer packages plus the narrow `internal/engine` request/result/cadence seam; standard library only | V2 Massive checkpoint source and named Massive checkpoint tests only | Golden/corrupt/maximum-shape results; exact-step authority table; queue/accounting trace; codec/store/engine tests and race; repository build/test/vet; path/dependency/ownership inspection; required review; next-slice validity | Live C5/C6 restart composition; replay runner continuation; objective benchmark |
| `C7-S3` | Compatible latest/previous checkpoint selection drives real C6 `[T0,R)` startup and optional C4 replay continuation with uninterrupted-state equivalence, and the recorded reference objective passes | `C7-LIVE-01` / `P-C7-LIVE`; `C7-REPLAY-01` / `P-C7-REPLAY`; `C7-OBJECTIVE-01` / `P-C7-OBJECTIVE` | Accepted `C7-S1`/`S2`; accepted C4/C5/C6 interfaces remain unchanged | Narrow engine/C4 replay-runner/C5-C6 fake-boundary wiring and C7 integration/benchmark artifacts; no production runtime/readiness code | None | Full live/replay differential outputs; segmented benchmark/control; complete C7 proofs and affected C1–C6 regressions; repository build/test/vet/race/diff; required review; final-review readiness | Component 8 production config/readiness/capacity/shutdown; C9 T/Q; live-provider cutover |

Each slice uses the repository slice-acceptance tier. Race detection is required
for all three: S1 changes mutable-owner/install isolation, S2 changes queues and
writer synchronization, and S3 composes concurrent live/recovery paths. S3's
acceptance reruns the complete eight-proof C7 ledger because it is the first
real persisted-to-live/replay vertical path. Final review then uses the final-
component tier and a separate reviewer.

### 16.1 Required independent reviews

- `C7-S1`: narrow sole-owner, as-of-`T0`, semantic completeness, restored-
  authority, and atomic-install review before S2.
- `C7-S2`: narrow untrusted-decoder, filesystem publication/durability,
  path-safety, cancellation, writer-concurrency, and request-accounting review
  before S3.
- `C7-S3`: narrow C4/C5/C6 identity/cutoff/fence, uninterrupted-equivalence,
  benchmark-validity, and deferred-operations-boundary review before final.
- Final: mandatory separate read-only complete-component review.

Use `gpt-5.6-sol` with medium reasoning as required by `AGENTS.md`. Send in-scope
findings to the active implementer, then use the same reviewer for focused
re-review when practical. A reviewer supplies evidence; it cannot approve a
contract change, whitelist expansion, revised proof, or slice boundary.

### 16.2 C7-C11 single-goal unattended execution protocol

The owner approved this completed contract, whitelist, proof allocation,
slices, reviews, and `advancement_mode: delegated` on 2026-08-07. One future
goal orchestrator may begin at `C7-S1` and execute the sequential C7-C11
program without clean-gate owner messages under the standing authority in
`AGENTS.md` and the implementation process:

1. Treat the parent delivery ledger as durable workflow state and resume from
   the first nonaccepted eligible item.
2. Keep at most one write-capable implementation subagent active. Give it only
   the current slice's routed contract/dependencies, whitelist, assignment,
   proofs, verification, prohibitions, and stop conditions.
3. After the writer is quiescent, use separate read-only subagents for the
   required narrow review and final component review; follow the model/effort
   policy above.
4. Return unambiguous in-scope findings to the implementer, rerun affected
   proofs/verification, and obtain focused re-review.
5. The orchestrator—not the reviewer—performs the conformance walkthrough and
   evaluates the objective delegated gate. When completely clean, it updates
   the parent ledger to `accepted` and immediately starts the next approved
   slice.
6. After all slices pass, run final-component verification/review. If clean,
   record C7 final acceptance and mechanically synchronize
   `docs/specification-map.md`.
7. Continue through C8-C11 in order. Each future skeleton and completed
   contract receives its separate mandatory independent review and clean
   focused re-review before the orchestrator records approval. Preserve the
   exact one-component lookahead and V2-reconnaissance ordering.
8. The orchestrator alone stages exact paths and makes separate local commits
   after every clean skeleton approval, completed-contract approval, accepted
   slice, final component acceptance, and distinct vertical milestone, and only
   after all writers and reviewers are quiescent. It never pushes, rebases,
   amends, rewrites history, deletes branches, or uses a destructive reset.

For C7, this protocol delegates only evidence-based slice and final acceptance;
it may not amend the approved contract, behavior, scope, document map,
dependency interface, V2 whitelist, proof allocation, slice boundary, or a
manual/failed/ambiguous gate. The separate standing owner decision permits the
orchestrator to approve clean independently reviewed C8-C11 skeletons and
completed contracts before implementation; it does not permit post-approval
substantive changes.

The exact program manual stops and standing C8-C11 product decisions are the
ones in
[`AGENTS.md`](../../../AGENTS.md#c7-through-c11-unattended-program-authority)
and the
[`implementation process`](../../implementation-process.md#28-c7-through-c11-unattended-program).
They include indispensable live evidence, unsupported capacity claims, public
deployment/cutover, post-approval substantive changes, unresolved failed or
ambiguous gates, authority drift, and unsafe Git state. Routine implementation
choices, correctable test failures, clean transitions, and unambiguous in-scope
review corrections do not stop the goal.

## 17. Implementation discretion

Implementers may choose private helper/type names, exact C7-private package/file
layout within the slice boundary, scratch-builder representation, zero-value
sharing, streaming buffer sizes inside the approved byte/heap bounds, error
wrapping, diagnostic record layout, fault-injection hooks, and test-fixture
builders. Equivalent standard-library mechanisms are allowed if they preserve
strict duplicate/unknown rejection, checksum coverage, the exact publication
protocol, and all proof observables.

The following are fixed: complete semantic rather than private-struct payload;
exact as-of-`T0` clipping; no persisted old causal positions/TQ/runtime state;
ordinary evaluator regeneration; JSON v1 envelope with raw-payload SHA-256;
latest/previous manifest authority; exact atomic write stages; one writer plus
one replaceable pending slot; 30-second committed-time cadence; no forced final
checkpoint; targets and reference fixture; and three-slice allocation.

**Prohibited changes**

- No second engine owner, canonical map, watermark, evaluator, publication,
  recovery planner, replay path, or writer access to mutable engine state.
- No synthetic aggregate, optional omission of success-bearing state, trust in
  cached ranking output, or old live/historical/replay source-position restore.
- No change to C1 binding, C2 merge/correction, C3 formula/qualification/
  accounting, C4 artifact/order, C5 epoch/fence, or C6 plan/terminal semantics.
- No V2 source outside the exact whitelist, predecessor dependency, version 1,
  live credentials/provider request, database, remote storage, journal, generic
  codec/plugin/event framework, public API/UI, or Component 8/9 policy.
- No silent fixture reduction, target waiver, semantic truncation, or proof/
  slice reassignment under delegated advancement.

**Stop/escalation conditions**

- Exact current C2/C3 state cannot be projected at `T0` without changing an
  approved dependency interface or retaining new success-bearing state.
- A complete semantic image cannot fit the approved bounded codec approach or
  meet the objective without changing format/bounds/target.
- Target filesystem semantics cannot implement the approved durability claim,
  or a fault result makes manifest authority ambiguous beyond the specified
  post-rename case.
- C4/C6 exact continuation interfaces conflict with the approved cutoff or
  restored-authority rules.
- Any proof fails in a way requiring a behavior, whitelist, fixture premise,
  primary proof, slice boundary, or operations-policy change.

## 18. Completed-contract acceptance checklist

- [x] The parent map lists the complete four-document contract and exclusive responsibilities.
- [x] The parent owns the sole mutable delivery ledger; details copy no status.
- [x] Proposed `delegated` advancement and all required reviews are explicit.
- [x] The modular layout routes the three real trust/delivery boundaries.
- [x] The parent-size exception is explained; each detail remains within its cohesive routing target.
- [x] Sections 1–19, all eight requirements/proofs, and all three slices have one authoritative home.
- [x] Document dependencies are explicit, acyclic, and links resolve.
- [x] Each slice has a compact routed context bundle.
- [x] Owner boundary/reconnaissance approval is recorded.
- [x] V2 inspection stayed inside the approved scope and after C6 final review.
- [x] Exact V2 sources, hashes, decisions, adaptations, and proof obligations are recorded.
- [x] Inputs, outputs, state, bounds, behavior, and terminal outcomes are complete without redefining Phase 1.
- [x] Primary accounting identities and overlapping diagnostics are explicit.
- [x] Every nontrivial edge has component-local evidence or a product invariant.
- [x] Every Component 7 requirement has one primary proof with a dangerous counterexample, observable result, paths, and limitation.
- [x] Consequential trust boundaries define exact success evidence and containment.
- [x] Construction guarantees and runtime validation are distinguished.
- [x] Every requirement/proof is allocated once to `C7-S1`–`C7-S3`.
- [x] Every slice is coherent, sequential, reviewable, and honest about deferred behavior.
- [x] Implementation discretion, prohibitions, and stop conditions support bounded assignments and unattended delegated gates.
- [x] The drift audit below has no substantive `yes`.
- [x] Owner approved the completed contract as written on 2026-08-07, including the exact V2 whitelist, fixtures/evidence, proofs, slices, reviews, and `advancement_mode: delegated`.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | It fixes the delegated representation/cadence/target required by `PG-OPS-01`; currentness/readiness/capacity remain C8. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | Engine projects/installs; detached codec/store/writer own only immutable bytes/files/results; ordinary C3 evaluator regenerates output. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | T/Q is explicitly prohibited from payload and starts empty after live restart. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | Artifact state is clipped to existing `[S,T0)` and continuation starts exactly at `T0`; restored provenance removes only dead-process ordering authority. |
| Did this add behavior without component-local evidence or explicit approval? | No | Every edge is tied to Phase 1/C2–C6 invariants or exact scoped V2 evidence; target becomes owner-approved with this contract. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | C7 owns only projection/schema/install/storage/cadence/restart composition; dependency semantics remain cited. |
| Did this add machinery without an approved need? | No | One semantic image, codec, manifest, sequential writer, and pending slot are the minimum complete path; alternatives are evaluated in Section 13. |
| Did version 2 drive the Phase 1 boundary instead of informing the detailed contract? | No | The preapproved Sections 1–7 preceded inspection; V2's conflicting seal/schema/owner/readiness behavior is rejected. |

Any later substantive `yes` requires owner review before advancement.
