# Live checkpoint hot-path correction

**Status:** Reopened and reaccepted 2026-08-13 after live API evidence exposed
the missing in-flight projection operand; bounded projection, alias isolation,
live responsiveness, and checkpoint persistence semantics remain unchanged

**Recorded:** 2026-08-12 under the
[`Version 1 Release Program`](v1-release-program.md) correction loop

**Boundary and completed-contract approval authority:** Version 1 Release
Program

**Standing program decisions:** Phase 1 checkpoint coherence and normal restart
support remain fixed. C7 implementation mechanics, persistence schema, cadence
below the fixed product limit, proof allocation, and performance settings are
revisable under the V1 correction loop.

**Controlling requirements:** `PG-OPS-01`, `PG-OPS-02`, `ARCH-OWN-01`–`04`,
`ARCH-FLOW-01`–`04`, `DTE-CHECKPOINT-01`–`03`, `DTE-COMMIT-02`–`04`,
`DTE-RECOVERY-02`–`03`, `LIFE-INIT-03`, `LIFE-LIVE-02`, `LIFE-LIVE-04`,
`LIFE-END-02`–`03`, `C7-STATE-01`, `C7-CODEC-01`, `C7-STORE-01`,
`C7-CADENCE-01`, `C7-OBJECTIVE-01`, `C8-MEASURE-01`

**Approved dependencies:**
[`Checkpoints and restart`](specifications/checkpoints-and-restart.md),
[`checkpoint state and installation`](specifications/checkpoints-and-restart/checkpoint-state-and-installation.md),
[`codec, storage, and cadence`](specifications/checkpoints-and-restart/codec-storage-and-cadence.md),
[`Readiness and operations`](specifications/readiness-and-operations.md), and the
accepted Components 1–6 boundaries.

## Contract document map and delivery ledger

| Document | Exclusive responsibility | Read for |
| --- | --- | --- |
| This document | Compact correction contract, Sections 1–19: observed failure, revised live projection mechanics, observability, proofs, and two sequential slices | Any checkpoint hot-path correction |
| C7 parent and details linked above | Persisted semantic state, compatibility, install, codec/store durability, restart objective, and still-valid accepted proofs | Any schema, installation, persistence, or objective change |
| C8 parent linked above | Runtime processing-delay, queue, watermark, heap, and readiness meanings | Operational measurement or API projection |

**Layout:** Compact single-file correction contract. This document reopens only
the lower-level C7 projection/cadence mechanics and their C8 measurement
composition. It does not create another component or delivery ledger.

The shared C7 parent ledger, specification map, and active live-correction
parent are currently modified by a concurrent REST/live reconciliation
correction. The implementing task must merge this correction into those three
authoritative maps before code changes; it must not overwrite the concurrent
work.

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Correction contract | `finally_accepted` | Authorized live run, two rejected monolithic measurements, bounded FIFO-continuation correction, alias-isolated writer transfer, real-consumer mature proof, current-host restart evidence, clean focused re-review, and final verification below | Preserve for V1 integration; provider validation remains separately authorized |
| `CKHOT-S1` projection ownership and observability | `accepted` | Existing complete semantic round trip remains clean; production projection no longer restores/evaluates a scratch engine; one explicit per-symbol defensive isolation copy prevents retained aliases, including after accepted submit to a paused writer; exact writer/engine accounting passes | Preserve while S2/final review complete |
| `CKHOT-S2` live responsiveness and restart composition | `accepted` | Three real-consumer dense boundaries held the owner for at most 20.870 ms, delivered aggregates within 66.874 ms, served observations within 55.523 ms, and retained a stable post-GC heap; final rerun restart median was 524.029 ms versus 1.381 s fresh | Preserve for V1 integration |
| In-flight API accounting correction | `accepted 2026-08-13` | A live run showed `/readyz` and snapshot mapping failures with `checkpoint_projection_identity` during each incremental projection. The engine now exposes the bounded `projection_in_progress` gauge, and the exact identity is `projection_started = projection_in_progress + projected + projection_rejected`. All activation/success/rejection/builder-failure/shutdown paths maintain the gauge under the engine lock; fixed failure reasons align with the mapper. Focused Engine/API/UI proofs, `go test -short -timeout 2m ./...`, affected race, affected vet, `git diff --check`, and required `gpt-5.6-sol` medium final focused review pass with no remaining P1/P2. | Preserve the additive operand and keep checkpoint work outside readiness ownership |

## Sections 1–4 — Outcome, scope, ownership, and settled boundary

### Outcome and operator consequence

A live checkpoint remains a usable restart boundary without periodically
freezing aggregate processing, watermark advancement, or snapshot reads. A
provider disconnect in the same process continues to use retained canonical
state plus exact gap recovery; a process restart uses the latest compatible
checkpoint plus `[T0,R)` catch-up. Checkpoint failure remains subordinate and
visible: it may leave an older checkpoint in force, but it cannot make live
ranking stale or consume seconds of the ordered engine path.

### Scope

In scope:

- coherent projection at one already committed and evaluated `T0`;
- projection validation placement, immutable ownership transfer, cloning and
  allocation behavior, cadence eligibility, and pressure deferral;
- bounded projection/writer diagnostics sufficient to distinguish no attempt,
  projection rejection, submission rejection, persistence failure, and usable
  completion;
- deterministic mature-state responsiveness and real persist/load/install/
  catch-up proof; and
- a measured cadence revision only if the corrected 30-second mechanism still
  cannot satisfy the live-path bound.

Not in scope:

- REST/live merge policy, feature formulas, qualification, ranking, canonical
  identity, correction horizon, committed-time meaning, or readiness semantics;
- eliminating checkpoints, weakening startup validation, persisting selected
  rows as authority, or treating an invalid artifact as usable;
- same-process gap recovery mechanics, T/Q persistence, a database, journal,
  remote storage, or public deployment; and
- checkpoint schema compaction unless S1 measurements identify retained image
  construction or bytes—not scratch validation/copying—as the remaining
  dominant cost.

### Ownership

`ScannerStateEngine` remains the sole mutable owner and alone seals `T0`.
Projection produces a detached immutable value; after accepted handoff the
checkpoint writer exclusively owns that value until terminal outcome. The
writer never reads engine state or decides market correctness. Startup alone
may install a fully validated candidate atomically. Operations and API layers
publish bounded facts only.

### Fixed boundary

- A checkpoint represents all restart-required state coherently at `[S,T0)`.
- The ordinary evaluator regenerates output after install; persisted rows are
  not ranking authority.
- Encoding, reopen validation, filesystem sync, and manifest replacement occur
  outside the ordered mutation path.
- Missing, stale, invalid, or incompatible artifacts fall back to previous or
  fresh hydration without partial install.
- Projection, submission, or write failure cannot freeze `T`, leave `live`, or
  fabricate restart protection.
- T/Q state never survives restart.

This correction introduces no product rule, second state owner, watermark,
evaluator, changed window, or T/Q-to-ranking dependency.

## Sections 5–8 — Failure evidence, questions, and implementation reconnaissance

### Observed authorized-run evidence

The 2026-08-13 owner run exposed a distinct observability defect after the hot
path itself was stable. Roughly every 30 seconds, while the incremental
projection copied the bound population, `/readyz` and `/api/v1/snapshot`
rejected otherwise-current publications with `checkpoint_projection_identity`.
The aggregate connection, qualified ranking, and watermark continued normally.
The mapper had modeled only terminal projection states:
`projection_started = projected + projection_rejected`. The live mechanism
also has exactly one bounded active state, so the complete identity is
`projection_started = projection_in_progress + projected +
projection_rejected`, with `projection_in_progress` constrained to `0|1`.
This was an API accounting false negative, not a provider disconnect or market-
state failure.

The 2026-08-12 private live run supplied a mature 5,694-symbol state after
7,175,675 historical rows:

- ingress queue high-water was only 195 of 32,768 frames, excluding queue
  capacity as the first cause;
- mean processing delay was 68 ms while maximum delay reached 17.886 seconds;
- one snapshot request failed after 12.551 seconds and six consecutive
  one-second attempts timed out before recovery;
- the scanner reached about 160% CPU, reported 4.1–5.6 GiB Go heap, reached a
  6.7 GiB process-footprint peak, and the sampled process had about 4.5 GiB
  swapped; and
- checkpoint operations reported zero submitted/completed artifacts and the
  checkpoint directory remained empty.

A five-second process sample concentrated work in checkpoint symbol projection,
scratch restoration/evaluation, Activity rebuilding, allocation, and GC. The
current path attempts every session-aligned 30 seconds. It constructs the
complete image, reconstructs and evaluates a scratch engine, compares results,
and clones the image under the engine mutex; writer submission clones it again.

This one-host diagnostic reopens only the claim that checkpoint production is
subordinate to ordinary evaluation. It does not invalidate checkpoint state,
durability, installation, or restart-equivalence proofs.

### Resolved questions and reuse

| Question | Decision |
| --- | --- |
| Must live projection reconstruct and evaluate a scratch engine? | No. Phase 1 requires a coherent engine-sealed view and strict install containment, not production-time self-restoration on every cadence. Preserve differential equivalence as a primary test and complete semantic validation on load/install. |
| May an accepted image be copied defensively at every boundary? | No. Go's public slice-bearing image cannot enforce a linear transfer against a retained shallow alias. Production therefore makes one explicit per-symbol defensive isolation copy from the engine projection value into package-private writer-owned storage, spread over the same bounded continuations. The external/test constructor makes one defensive whole-image copy. No other engine/request/writer/codec/store copy is permitted. The writer may retain at most its existing one in-progress plus one replaceable pending image. |
| Should cadence immediately widen? | No. First preserve the accepted 30-second freshness and remove known redundant work. Revise cadence only from S2 measurements, never to conceal a blocking transition. |
| Should checkpoint bytes be reduced now? | Not initially. Existing persisted state remains authoritative. Representation compaction is a correction fallback only if phase measurements show image construction/retained bytes dominate after validation and copy removal. |
| Is queue-empty-only projection sufficient? | No. It can starve checkpoint creation during sustained traffic. Pressure may defer an attempt, but bounded progress must still yield usable artifacts. If a monolithic corrected projection misses the bound, use bounded incremental projection or immutable/copy-on-write segments at the same sealed `T0`. |

No Version 2 inspection is required for the minimal correction: the current
implementation, accepted contracts/proofs, authorized run, and mathematical
ownership invariants answer the distinguishing questions. If S1 fails twice on
the same monolithic mechanism, record a narrow reconnaissance question before
inspecting only V2 checkpoint snapshot/ownership mechanics; do not import its
owner, ranker, lifecycle, or cadence assumptions.

## Sections 9–14 — Required behavior, trust, observability, bounds, and edges

### `CKHOT-01` — coherent minimal projection

At an eligible committed/evaluated `T0`, projection copies exactly the existing
C7 semantic image. Construction validates source-local invariants needed to
prevent impossible data from entering the image: binding and cutoff identity,
bounded cardinalities, canonical ordering, finite values, and cross-record
references available directly from the authoritative state.

The live projection path must not construct a second engine, restore features
from the image, rerun the evaluator, or compare a scratch evaluation. Complete
restore equivalence remains mandatory in `P-CKHOT-SEMANTIC`; semantic
validation and atomic reconstruction remain mandatory on load/install.

### `CKHOT-02` — alias-free ownership with one explicit isolation boundary

Each projected symbol is defensively isolated once into package-private
writer-request storage during its bounded projection continuation. An accepted
submission transfers that completed storage exclusively to the writer; neither
the engine nor a retained shallow alias can mutate it. Rejection releases the
builder. Pending replacement produces the existing exact superseded terminal
and releases the replaced image.

The engine, request, writer, codec, and store must not each clone the image.
Small metadata copies are permitted. The external/test `NewRequest` defensive
whole-image copy names the caller and writer as the two possible writers; the
production builder spreads equivalent isolation across per-symbol slices so it
does not add another monolithic owner hold.

### `CKHOT-03` — subordinate cadence and progress

Keep session-aligned 30-second committed-time eligibility initially. At most one
attempt occurs for `(binding,T0)`. Projection must not run merely because wall
time advanced, and same-`T0` corrections cannot mutate a detached image.

The scanner may defer an eligible attempt while the aggregate queue is already
above a measured pressure threshold or a prior checkpoint is in progress plus
pending. Deferral is observable and retries only at a later coherent boundary.
It cannot create unbounded goroutines, queued projections, or engine-owned
detached images. A quiet queue is not a correctness prerequisite.

For the mature 6,000-symbol S2 fixture, checkpoint-caused ordered-path hold is
at most 500 ms and checkpoint-caused maximum aggregate processing delay is at
most one second across three independently recorded boundaries. No boundary may
cause `watermark_stale`, API timeout, ingress rejection, or accounting failure.
These are current correction-loop delivery settings below the fixed requirement
that checkpoint work not block ordinary evaluation; they may be tightened from
evidence. A revision above them must identify a different mechanism and remain
strictly below the two-second readiness tolerance.

If the minimal S1 mechanism misses this bound twice, abandon monolithic
projection under one lock. The next mechanism must preserve one `T0` using
bounded incremental projection with mutation version checks, or immutable/
copy-on-write state segments. It may not relax coherence or silently widen
cadence as a substitute for responsiveness.

### `CKHOT-04` — usable completion and truthful diagnostics

Expose bounded cumulative and last-attempt facts for:

- eligible, pressure-deferred, projection-started, projected, projection-
  in-progress (`0|1`), rejected by fixed reason, submit-rejected, submitted,
  writer-in-progress, pending,
  superseded, completed, failed, and canceled;
- last attempted/projected/submitted/successful `T0` and usable checkpoint age;
- projection total/lock duration, semantic image bytes or encoded artifact
  bytes, encode/write/reopen-validation duration, and last fixed failure step;
  and
- projection allocation delta and post-GC retained heap in measurement mode,
  without paths, symbols, payloads, or unbounded labels.

The accounting identities remain exact. “Attempted but rejected” cannot appear
as “checkpoint off,” and zero submitted/completed artifacts cannot be presented
as restart protection. Diagnostics do not influence cadence, readiness,
ranking, or checkpoint acceptance.

### Trust and failure outcomes

| Boundary | Required success | Containment | Dangerous false success |
| --- | --- | --- | --- |
| Projection | One coherent image at committed/evaluated `T0` | Reject with fixed reason; continue live | Mixed-time feature/mark state persists quickly |
| Ownership transfer | No mutable alias after accepted submit | Reject or supersede exactly; release ownership | Writer observes engine mutation or a reused backing slice |
| Persisted artifact | Existing strict encode/digest/reopen/manifest protocol completes | Previous manifest remains authority | Fast path labels an unwritten or invalid image usable |
| Startup | Full compatibility and semantic install validation passes | Previous/fresh fallback, no partial install | Runtime validation removal leaks into startup trust |
| Pressure | Market path stays within bound and checkpoint eventually progresses | Observable defer/supersede/failure | Queue-empty policy silently prevents all checkpoints |

### Bounds and evidenced edges

- Preserve one writer, one in-progress image, one replaceable pending image,
  latest/previous artifacts, byte/deadline ceilings, and terminal accounting.
- During projection handoff, there is at most one newly detached image outside
  the writer bounds. Superseded/rejected images become unreachable promptly.
- No per-symbol goroutine, generic snapshot service, journal, or second mutable
  canonical representation is permitted.
- Prove empty/sparse symbols, maximum correction tail, price extrema, Activity
  mutable/folded/reference state, qualification provisional/dirty/finalized
  state, every coverage consequence, same-`T0` correction after detachment,
  pending replacement, projection rejection, writer failure, latest-invalid/
  previous-valid load, process restart, and same-process disconnect independence.

## Sections 15–17 — Primary proofs, slices, discretion, and correction triggers

### Primary proofs

| Proof | Claim and dangerous counterexample | Observable result and limitation |
| --- | --- | --- |
| `P-CKHOT-SEMANTIC` | Uninterrupted state equals project → persist → load → install → same continuation for the complete mature state families; counters removal of live scratch validation masking an incomplete image | Bit-identical canonical/evaluator result and publication at the same continuation boundary. Deterministic fixture, not provider latency. |
| `P-CKHOT-OWNERSHIP` | Accepted transfer has no writable alias and pending replacement releases exactly one superseded image; counters mutation through shared slices or hidden third full clone | Mutation/alias adversary, allocation/copy instrumentation, and exact writer/engine accounting. Compiler/API construction proof preferred over convention. |
| `P-CKHOT-LIVE` | Three checkpoint boundaries under mature 6,000-symbol aggregate load meet 500 ms lock/one-second delivery bounds with no stale watermark, API timeout, rejection, or accounting failure; counters a fast sparse projection or proof-induced idle queue | Phase timings, queue high/drain, frame accounting, watermark/readiness, API latency, heap allocation/post-GC plateau. Local deterministic capacity evidence only. |
| `P-CKHOT-RESTART` | At least one artifact is actually completed, discovered, installed, and used for `[T0,R)` catch-up materially faster than equivalent fresh hydration; counters zero-submission success or a fast projection producing an unusable file | Three bounded trials under the existing C7 objective, plus latest-corrupt/previous-valid control. No live-provider SLA. |

### Sequential assignments

`CKHOT-S1` owns `CKHOT-01`, `CKHOT-02`, and diagnostic plumbing required to
measure them. Allowed ownership is `internal/engine` checkpoint projection and
cadence, `internal/checkpoint` request/writer ownership, focused tests, and
bounded checkpoint facts through operations/snapshot mapping. It must preserve
the current semantic image and store protocol. Run `P-CKHOT-SEMANTIC` and
`P-CKHOT-OWNERSHIP`, affected short tests, focused race, vet, and diff check.

`CKHOT-S2` owns `CKHOT-03`, `CKHOT-04`, mature live composition, and restart
objective reacceptance. It may revise the lower-level cadence only after
recording S1 phase measurements and must rerun `P-CKHOT-LIVE` and
`P-CKHOT-RESTART`. It begins only after S1 is accepted.

Implementation discretion includes the transfer type, metadata representation,
timing mechanism, measured pressure threshold, and fixture construction.
Prohibited changes include semantic-state omission, startup validation
weakening, selected-row persistence, approximate restore, fabricated
completion, readiness tolerance inflation, conflict-policy changes, or
checkpoint-off as final acceptance.

Correction triggers are: projection still reconstructs/evaluates an engine;
more than one unexplained whole-image copy; any mature boundary above the S2
limits; no usable completed artifact; allocation/GC without post-run plateau;
restart not materially faster than fresh; changed uninterrupted output; or a
review finding involving aliasing, mixed `T0`, invalid persistence, or partial
install. Preserve unaffected C7 store/codec/install evidence and correct the
lowest implicated mechanism.

## Sections 18–19 — Acceptance and drift audit

### Correction and measured evidence

The minimal no-scratch/no-clone mechanism still failed twice and is preserved
as rejected evidence. The first dense 6,000-symbol projection held the owner
for 14.493 seconds and allocated 16,056,672,928 bytes cumulatively. Preallocating
the exact slice capacities reduced cumulative allocation to 9,764,976,816
bytes but the second run held the owner for 26.963 seconds. Both images contained
9,534,000 Activity reference summaries. This selected retained-image
construction, not scratch validation or defensive copying, as the remaining
dominant cost and triggered the required mechanism change.

The accepted mechanism keeps projection in the existing sole engine consumer.
It copies one symbol, appends one internal continuation at the FIFO tail, and
therefore lets already-admitted market inputs run between projection slices.
One projection is active at a time. A pre-`T0` aggregate admitted after sealing
marks the detached image dirty and causes rejection; lifecycle/binding/sequence
change also rejects. The engine retains only request identity after the
consuming handoff. A package-private builder isolates every projected symbol
from retained aliases, then the writer exclusively owns the completed image
within its existing one-in-progress/one-replaceable-pending bound. The paused-
writer adversary mutates a retained shallow alias after accepted submission and
verifies that persisted bytes remain unchanged.

`P-CKHOT-LIVE` used the accepted dense mature fixture: 6,000 symbols, 1,589
Activity references per symbol, 9,534,000 references total, and three successive
30-second boundaries. Unlike the rejected first proof, the accepted run uses
`engine.New`, the production sole consumer, and its physical FIFO. It completed
129, 125, and 127 live exact-duplicate aggregate facts while concurrent
operational and snapshot observers made 9,083 captures. Projection totals were
14.032, 13.549, and 13.780 seconds; maximum per-symbol owner holds were 20.870,
10.658, and 17.508 ms. Worst aggregate completion delay was 66.874 ms and worst
observer delay was 55.523 ms. Cumulative allocations were 12,988,548,192,
12,987,960,328, and 12,988,508,312 bytes; post-GC heap was 3,435,158,912,
3,435,130,360, and 3,435,131,440 bytes. Every terminal returned to engine
accounting, all ingress/transition/aggregate/checkpoint identities reconciled,
and no admission was rejected. Operations composition separately captures the
production snapshot/status/metrics path during a real persist/reopen completion
and asserts sub-second capture, current readiness/watermark, and valid
accounting. Internal continuation occupancy is maintained in O(1), including a
stale continuation after rejection; the 8,192-slot near-capacity regression
proves it is never scanned as external ingress. This is local deterministic
evidence, not a provider throughput claim.

The first current-host `P-CKHOT-RESTART` rerun exposed a fixture/decoder
failure rather than an acceptable result: checkpoint median was 3.927 seconds
while an eight-worker fake fresh control took 1.374 seconds. Decode alone took
2.733-2.829 seconds. The correction merged duplicate-key and cardinality
preflight into one bounded pass, retained strict unknown-field decode and full
semantic validation, and replaced temporary-file spooling with one byte-bounded
in-memory payload. The objective now uses the supported private launcher's
exact two hydration workers instead of scaling the fake server by host CPU.

The final three trials used the unchanged 6,000-symbol, 1,674,157-byte,
961-correction artifact and all required state families. Checkpoint end-to-end
times were 507.100, 524.029, and 533.889 ms; fresh controls were 1.381, 1.370,
and 1.410 seconds. Median restart was 524.029 ms versus 1.381 seconds fresh,
62.1% faster. Load was 108.803-110.315 ms, install 2.874-3.824 ms, and
`[T0,R)` catch-up 393.295-421.538 ms. Existing latest-corrupt/previous-valid
and atomic-install controls remain clean. A production-composition regression
also completes one real artifact, discovers it, returns its terminal to engine
accounting, and exposes nonzero encode/reopen/write/byte facts.

Acceptance requires:

- both slices and all four primary proofs passing with explicit timeouts;
- ordinary `go test -short -timeout 2m ./...`, affected focused race tests,
  affected `go vet`, and `git diff --check`;
- exact before/after projection phase, API latency, queue, delivery delay, heap,
  artifact completion, age, and restart/fresh measurements;
- one focused read-only review using the required reviewer because immutable
  handoff and persisted restart correctness change a consequential concurrency
  and persistence boundary; and
- parent/map/active-correction ledger merge after the concurrent REST/live work
  is quiescent, without overwriting it.

Drift audit:

- fixed checkpoint coherence, canonical ownership, restart continuation,
  invalid-artifact fallback, and market semantics are unchanged;
- same-process disconnect recovery remains independent of checkpoint presence;
- the 30-second cadence remains the initial setting, not an excuse for blocking;
- no CPU or memory claim extends beyond the recorded host/fixture;
- no provider request or credential is authorized by this contract; and
- the required focused re-review closed two initial P2 findings and one
  correction-induced O(queue) accounting finding, then returned `CLEAN/PASS`;
  ordinary, focused race, capacity/restart, vet, and diff verification passed;
  this focused correction is finally accepted.
