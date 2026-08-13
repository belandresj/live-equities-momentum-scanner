# Checkpoints and restart — proof and delivery plan

**Parent contract:** [checkpoints-and-restart.md](../checkpoints-and-restart.md)
**Normative responsibility:** Exact reconnaissance/reuse ledger, complete
requirement/proof allocation, sequential slice plan, independent-review
triggers, V1 correction/orchestration protocol, discretion, completed-
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
| `C7-REPLAY-01` | `P-C7-REPLAY`: checkpointed versus full replay differential at two playback paces | Exact artifact cutoff and ordinary C4 delivery reproduce full replay; repeated/pre-`T0` or gapped continuation cannot pass. Counterexamples: duplicated prefix double-counts Activity/qualification, or a foreign/forged public fact authorizes a missing prefix while final ranking appears plausible. | Full-from-`S` and engine-owned checkpoint-from-`T0` outputs/counters equal at logical checkpoints and end at both paces; bad/foreign cutoff rejects and complete-from-`S` fallback succeeds only under fresh-session authority. Does not prove live epochs, latency, or T/Q. | Accepted C4 artifact/runner fixtures plus C7 state fixture; no V2 reuse | `C7-S3` |
| `C7-OBJECTIVE-01` | `P-C7-OBJECTIVE`: validated, individually bounded three-trial 6,000-symbol reference benchmark with equivalent fresh controls | Under the current program-selected settings, every checkpoint trial is <=60s end to end and checkpoint median is at least 20% faster than fresh recovery under the same binding, `R`, provider fixture, worker limits, and host. Recorded measured evidence may revise those settings, but restart must remain materially faster than equivalent fresh recovery and below the product's rejected approximately 130-second precedent. Counterexample: a tiny/mismatched fixture, mocked C6 work, unbounded trial, or local segment threshold masquerades as the product recovery claim. | Fixture schema/hash/bytes/cardinalities/corrections/state families/planned work/host and every segment are validated before timing and recorded; local load/install is diagnostic, not a five-second gate. Does not prove live-provider or 100,000-symbol deployed capacity. | Final 2026-08-12 correction run: unchanged 1,674,157-byte artifact, 961 corrections, all required state families, fixed two-worker comparison control; checkpoint median 524.029ms versus fresh median 1.381s (62.1% faster). The later launcher-default revision does not rewrite this measured baseline, and the prior eight-worker host-scaled control failure remains preserved in the hot-path correction. | `C7-S3` |

No secondary proof layer duplicates these claims. `P-C7-LIVE` is intentionally
cross-component rather than a second install proof: it establishes C5/C6
handoff and current-epoch reconciliation that `P-C7-INSTALL` cannot.

The compact `P-C7-LIVE` fixture uses one hydration worker for its two semantic
requests. Two repeated runs of the prior concurrent loopback fixture failed in
the local TLS handshake before either request could distinguish checkpoint
semantics; reducing only that proof's worker count removes the transport flake
without changing `[T0,R)`, empty-evidence, correction, or equivalence claims.
The selected 6,000-symbol objective proof retains and preflights its explicit
multiworker workload, so this correction does not weaken the accepted capacity
or timing evidence.

## 16. Sequential implementation-slice plan

Three slices are required because in-memory canonical ownership, persisted
untrusted bytes/files/concurrency, and cross-component live/replay restart are
distinct trust and proof boundaries. Combining them would make atomic-state
defects, filesystem defects, and continuation-interface defects difficult to
review or roll back independently.

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies / entry state | Allowed ownership or packages | Approved V2 whitelist | Acceptance record | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `C7-S1` | A bound engine can project one complete immutable semantic image at committed `T0`, validate/install it all-or-nothing in a fresh engine, regenerate the ordinary evaluation, and accept valid new-run correction authority | `C7-STATE-01` / `P-C7-STATE`; `C7-INSTALL-01` / `P-C7-INSTALL` | Finally accepted C1–C6; completed C7 approval; no C7 code | `internal/engine` and a C7-private semantic model under `internal/checkpoint` if needed; no filesystem, goroutine, CLI, or provider code | V2 scanner checkpoint source and named scanner tests only | Exact state coverage/diff; install mutation results; owner/apply inspection; affected engine tests and race; repository compile/test/vet; construction/success/failure walkthrough; required review result; next-slice validity | On-disk codec/store/writer/cadence; live/replay composition; timing target |
| `C7-S2` | The S1 image has one strict bounded persisted codec and private latest/previous local store; one asynchronous writer obeys cadence, coalescing, exact request accounting, and failure containment without blocking engine progress | `C7-CODEC-01` / `P-C7-CODEC`; `C7-STORE-01` / `P-C7-STORE`; `C7-CADENCE-01` / `P-C7-CADENCE` | Accepted `C7-S1`; semantic completeness/install authority remains fixed, while codec mechanics are reopenable | C7-owned codec/store/writer packages plus the narrow `internal/engine` request/result/cadence seam; current standard-library JSON may be optimized/replaced under the V1 correction loop | Current recorded V2 Massive checkpoint sources/tests; expand only after recording the new in-boundary source and proof | Preserve golden/corrupt/durability/queue correctness; rerun only affected codec/store proofs and race if S3 correction changes mechanics; keep the 100,000-symbol case outside short tests | Live C5/C6 restart composition; replay runner continuation; corrected objective benchmark |
| `C7-S3` | Compatible latest/previous checkpoint selection drives real C6 `[T0,R)` startup and optional C4 replay continuation with uninterrupted-state equivalence, and the corrected 6,000-symbol reference objective passes | `C7-LIVE-01` / `P-C7-LIVE`; `C7-REPLAY-01` / `P-C7-REPLAY`; `C7-OBJECTIVE-01` / `P-C7-OBJECTIVE` | Accepted/reopenable `C7-S1`/`S2`; accepted C4/C5/C6 meanings remain unchanged | Narrow engine/C4 replay-runner/C5-C6 fake-boundary wiring and C7 integration/benchmark artifacts; any codec correction stays inside C7; no C8 runtime/readiness code | None unless a recorded C7 codec correction requires an in-boundary whitelist revision | Live/replay differential outputs; validated three-trial 6,000-symbol benchmark and equivalent fresh controls with per-trial deadlines; affected regressions/short suite/race; narrow identity/fence review; final-review readiness | C8 production config/readiness/capacity/shutdown; C9 T/Q; live-provider validation |

Each slice uses the repository slice-acceptance tier. Historical S1/S2 race and
review evidence remains valid unless a correction touches its boundary. S3
runs its three allocated proofs, affected regressions, short repository check,
and implicated race paths; it does not rerun every expensive C7 proof after
each correction. The complete current ledger runs once after S3 stabilizes for
final component review.

### 16.1 Required independent reviews

- `C7-S1`: narrow sole-owner, as-of-`T0`, semantic completeness, restored-
  authority, and atomic-install review before S2.
- `C7-S2`: narrow untrusted-decoder, filesystem publication/durability,
  path-safety, cancellation, writer-concurrency, and request-accounting review
  before S3.
- `C7-S3`: narrow C4/C5/C6 identity/cutoff/fence and uninterrupted-equivalence
  review before final. Benchmark premises/results are part of the primary proof
  record, not a second broad review.
- Final: mandatory separate read-only complete-component review.

Use `gpt-5.6-sol` with medium reasoning as required by `AGENTS.md`. Send in-scope
findings to the active implementer, then use the same reviewer for focused
re-review when practical. A reviewer supplies evidence; it does not edit,
stage, commit, or expand scope. Under the V1 program the orchestrator may revise
the lower-level contract, whitelist, proof, or slice and records that correction.

### 16.2 Version 1 single-goal execution protocol

The owner revised this contract and replaced the former unattended authority
with the [Version 1 Release Program](../../v1-release-program.md). A future goal
resumes from `C7-S3` and may revise C7 lower-level decisions without another
owner message:

1. Preserve the existing dirty S3 code and the measured failed-proof record in
   the parent ledger.
2. Classify each failure and revise the lowest unsuitable contract, whitelist,
   fixture, benchmark, proof, slice, or implementation decision.
3. Reopen S1/S2 only when the new evidence implicates their behavior; preserve
   unaffected correctness, durability, and review evidence.
4. Keep at most one write-capable implementation worker. Reviewers are read-
   only and run only after the writer is quiescent.
5. During correction, run the failed proof and direct regressions only. After
   S3 stabilizes, run its complete acceptance tier once, then one final C7
   read-only review and focused finding re-review if needed.
6. Record final C7 acceptance, synchronize the specification map, and continue
   sequentially through the owner-approved C8-C11 boundary plans.
7. Complete each future contract just in time, using at most two slices unless
   a distinct consequential boundary requires a third. Contract review is
   risk-triggered; final component review is always required once.
8. Stage exact paths and make local commits at coherent correction/planning
   milestones, accepted slices, final components, and vertical milestones only
   while all workers/reviewers are quiescent. Never push or rewrite history.

The zero-interruption and containment policy in
[`v1-release-program.md`](../../v1-release-program.md#3-zero-interruption-execution-policy)
controls. Failed/ambiguous proofs, benchmark misses, review findings,
post-approval contract changes, whitelist changes, and accepted implementation
defects are correction inputs, not owner gates.

## 17. Implementation discretion

Implementers may choose private helper/type names, C7-private package/file
layout, scratch-builder representation, zero-value sharing, buffer sizes inside
hard bounds, error wrapping, diagnostics, fault-injection hooks, and fixture
builders. The V1 correction loop also permits optimizing or replacing the
current JSON/decoder mechanics, revising the V2 whitelist, benchmark fixture,
proof allocation, and slice boundary when measured evidence requires it.

The following semantic outcomes are fixed: complete restart-required state at
one exact `T0`; no persisted old causal positions/TQ/runtime state; ordinary
evaluator regeneration; whole-candidate validation/install or none;
latest/previous complete fallback or fresh recovery; nonblocking bounded
persistence; exact `[T0,R)`/replay continuation; 30-second cadence unless later
evidence selects another cadence consistent with `PG-OPS-01`; and honest
fallback. JSON, decoder passes, local segment timing, fixture construction,
proof mechanics, and the historical three-slice delivery plan are lower-level
decisions.

**Prohibited changes**

- No second engine owner, canonical map, watermark, evaluator, publication,
  recovery planner, replay path, or writer access to mutable engine state.
- No synthetic aggregate, optional omission of success-bearing state, trust in
  cached ranking output, or old live/historical/replay source-position restore.
- No change to C1 binding, C2 merge/correction, C3 formula/qualification/
  accounting, C4 artifact/order, C5 epoch/fence, or C6 plan/terminal semantics.
- No unrecorded V2 source, predecessor dependency, older Version 1 source, live
  credentials/provider request, database, remote storage, journal, generic
  plugin/event framework, public API/UI, or Component 8/9 policy.
- No silent fixture reduction, target waiver, semantic truncation, mismatched
  fresh control, or unbounded performance trial. Recorded proof/slice/
  whitelist revisions are permitted by the V1 program.

**Correction and containment conditions**

- Projection/install, codec/format, filesystem, C4/C6 continuation, fixture,
  benchmark, proof, slice, whitelist, or implementation evidence that fails
  while Phase 1 meaning remains satisfiable enters the V1 correction loop.
- Apparent tension with fixed Phase 1 meaning uses the program's authority
  order, strict compatible intersection, simplest design, and honest fallback.
- Credentials/live observation and destructive/external action remain deferred;
  user/Git overlap and tool/reviewer limits use the program's automatic
  containment and fallback rules.

## 18. Completed-contract acceptance checklist

- [x] The parent map lists the complete four-document contract and exclusive responsibilities.
- [x] The parent owns the sole mutable delivery ledger; details copy no status.
- [x] V1 delegated correction/acceptance and risk-based review are explicit.
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
- [x] Implementation discretion, prohibitions, correction triggers, and the zero-interruption/containment link support bounded assignments.
- [x] The drift audit below has no substantive `yes`.
- [x] Owner revised the contract under the Version 1 Release Program on 2026-08-07; fixed Phase 1 meaning is preserved and lower-level delivery decisions are explicitly revisable.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | The corrected benchmark measures the existing `PG-OPS-01` restart outcome; it removes a non-product five-second gate and leaves C8 readiness/capacity ownership unchanged. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | Engine projects/installs; detached codec/store/writer own only immutable bytes/files/results; ordinary C3 evaluator regenerates output. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | T/Q is explicitly prohibited from payload and starts empty after live restart. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | Artifact state is clipped to existing `[S,T0)` and continuation starts exactly at `T0`; restored provenance removes only dead-process ordering authority. |
| Did this add behavior without component-local evidence or explicit approval? | No | Phase 1/C2-C6 invariants, the exact failed S3 measurement, and the owner V1 correction support the revised proof/threshold. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | C7 owns only projection/schema/install/storage/cadence/restart composition; dependency semantics remain cited. |
| Did this add machinery without an approved need? | No | The revision adds no runtime machinery; it permits measured simplification/replacement of expensive lower-level mechanics. |
| Did version 2 drive the Phase 1 boundary instead of informing the detailed contract? | No | The preapproved Sections 1–7 preceded inspection; V2's conflicting seal/schema/owner/readiness behavior is rejected. |

Any later lower-level substantive `yes` enters the V1 correction loop. Apparent
tension with fixed Phase 1 meaning uses the program's authority order, strict
compatible intersection, simplest design, and honest fallback.
