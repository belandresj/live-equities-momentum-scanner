# Aggregate replay

**Status:** Finally re-accepted 2026-08-11 after the narrow production-reader
resource correction. The exact 2,584,011,150-byte artifact validates in 40.53
seconds with approximately 20.30 MB peak reader RSS; schema, persisted trust,
replay lifecycle, `C4-S1`--`C4-S6`, and their eleven primary proofs remain
preserved, and the focused persisted-trust review found no P1/P2.

**Owner boundary approval:** approved 2026-08-06 in the owning Codex task;
includes the Phase 1 boundary, modular document map, and exact future version 2
reconnaissance scope

**Owner contract/reuse/test/slice-plan approval:** the original nine-proof,
four-slice plan was approved 2026-08-06 in the owning Codex task. The owner
approved the additive tenth proof and `C4-S5` correction on 2026-08-09; the
owner approved the additive eleventh proof and `C4-S6` correction plan later
on 2026-08-09; the version 2 whitelist remains unchanged.

**Advancement mode:** `delegated` for `C4-S1`–`C4-S6` and required reviews.
`C4-S6` activates only after C11 and the integrated private V1 RC are accepted.
Routine in-scope corrections continue without another owner message; a fixed-
authority conflict, new source scope, provider/credential action, or destructive
external action still stops for the smallest owner decision.

**Controlling Phase 1 requirements:** `PG-REPLAY-01`, `PG-REPLAY-02`,
`PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-03`,
`ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, `ARCH-FLOW-03`,
`ARCH-FLOW-04`, `DTE-MODEL-01`, `DTE-MODEL-02`, `DTE-MODEL-03`,
`DTE-SESSION-01`, `DTE-SESSION-02`, `DTE-SESSION-03`,
`DTE-SESSION-04`, `DTE-CLOCK-02`, `DTE-CLOCK-03`, `DTE-CLOCK-04`,
`DTE-CLOCK-05`, `DTE-CLOCK-06`, `DTE-WINDOW-01`, `DTE-WINDOW-02`,
`DTE-WINDOW-04`, `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-AGG-01`,
`DTE-AGG-02`, `DTE-AGG-03`, `DTE-AGG-04`, `DTE-TIMER-01`,
`DTE-MERGE-01`, `DTE-MERGE-02`, `DTE-MERGE-05`, `DTE-COMMIT-01`,
`DTE-COMMIT-02`, `DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-REPLAY-01`,
`DTE-REPLAY-02`, `DTE-REPLAY-03`, `DTE-REJECT-01`, `LIFE-MODEL-01`,
`LIFE-MODEL-02`, `LIFE-MODEL-04`, `LIFE-INIT-01`, `LIFE-REPLAY-01`,
`LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-SUPPRESS-01`,
`LIFE-SUPPRESS-02`, `LIFE-SUPPRESS-03`, `LIFE-END-01`, `LIFE-END-02`,
`LIFE-END-03`, `LIFE-PUBLISH-02`, `LIFE-T03`, `LIFE-T23`, `LIFE-T24`,
`LIFE-T25`, `LIFE-T28`, and `LIFE-T29`

**Approved dependencies:** the finally approved Component 1 immutable
[`reference.Binding`](reference-data-and-session-binding.md#9-detailed-semantic-inputs-outputs-and-owned-state),
the finally approved Component 2
[`ScannerStateEngine` contract](scanner-state-engine-and-canonical-state.md),
and the finally accepted Component 3
[`aggregate evaluator contract`](aggregate-features-qualification-ranking-and-accounting.md).
Component 3 passed its mandatory final review and was finally accepted on
2026-08-06. Component 4 then completed the approved reconnaissance and routed
Sections 8–19. The owner approved the complete contract, whitelist, proofs,
slice plan, review triggers, and delegated advancement on 2026-08-06;
`C4-S1` through `C4-S6`, the original mandatory final component review, and all
focused correction re-reviews are accepted.

## Contract document map

The approved detailed-contract layout is modular because REST normalization/artifact
trust, replay scheduling/lifecycle, and deterministic-core delivery are
independently reviewable concerns. The skeleton approved the paths and
responsibilities before reconnaissance; the detailed stage has now populated
all three without changing that map.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Outcome, ownership/non-scope, cross-cutting invariants, Sections 1–7, routing, approvals, and sole delivery ledger | All controlling Phase 1 IDs; Sections 1–7 | Every Component 4 task | Components 1–3 |
| [REST normalization and artifact trust](aggregate-replay/rest-normalization-and-artifact.md) | Shared Massive REST row normalization, compiler consumption of that one mapper, offline download/compiler, and artifact trust/bounds | Sections 8–14; `C4-NORM-01`, `C4-COMP-NORM-01`, `C4-DL-01`, `C4-ART-01`, `C4-ART-02`; `P-C4-NORM`, `P-C4-COMP-NORM`, `P-C4-DL`, `P-C4-ART-BYTES`, `P-C4-ART-TRUST`; `C4-S1`, `C4-S2` | Provider mapping, future Component 6 seam obligation, compiler/artifact decisions | Parent; Components 1 and 2 |
| [Replay source and simulated clock](aggregate-replay/replay-source-and-clock.md) | Validated artifact source, simulated clock/group timers, typed engine admissions, replay lifecycle/completion including an operator-requested end, and containment | Sections 8–14; accepted `C4-SCHED-01`, `C4-RUN-01`, `C4-FAIL-01`, `C4-PREFIX-END-01`, and `C4-BOUNDED-CANCEL-01`; their routed proofs; accepted `C4-S3`, `C4-S5`, and `C4-S6` | Playback, clock, lifecycle, commit, engine integration, requested-end integrity, and the bounded manual-driver seam | Parent; preceding detail; Components 1–3 |
| [Deterministic-core delivery](aggregate-replay/deterministic-core-delivery.md) | Complete proof/requirement ledger, Components 1–4 milestone, correction slice plan, discretion, acceptance, and drift | Sections 15–19; all eleven accepted Component 4 requirements/proofs; accepted `C4-S1`–`C4-S6` | Proof allocation, assignments, completed-contract and final review | Parent and both details; Components 1–3 |

**Layout:** Approved modular contract with this parent and the three complete
detail documents listed above. The parent remains slightly above the template's
approximate 2,500-word routing target because its owner-approved Sections 1–7
are one boundary/reconnaissance record and the approved map keeps them here;
the detailed normative ledgers are routed out rather than enlarging it further.

**Routing rule:** Any requirement, evidence decision, proof, or task not
unambiguously routed by this map stops for parent-map correction. Adding,
removing, or changing a mapped detail responsibility after boundary approval
requires owner review.

**Contract-wide coverage and acceptance:** The authoritative
[proof/requirement ledger and slice plan](aggregate-replay/deterministic-core-delivery.md#15-primary-proof-allocation)
routes the nine accepted baseline requirements and the two additive corrections
to six accepted sequential slices. The original
[completed-contract checklist](aggregate-replay/deterministic-core-delivery.md#18-completed-contract-acceptance-checklist)
remains accepted for `C4-S1`–`C4-S4`; the requested-end entries, `C4-S5`,
cumulative verification, and both focused correction re-reviews are complete.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `accepted` | Owner-approved Sections 1–19, unchanged version 2 whitelist, six accepted slices, all eleven clean primary proofs, cumulative short/race/vet verification, clean drift audit, original final review, and clean requested-end/bounded-cancel focused re-reviews | 2026-08-09 | Complete |
| `C4-S1` | `accepted` | `C4-NORM-01` / `P-C4-NORM`, affected and repository build/tests, conformance/ownership/whitelist inspection, and required `gpt-5.6-sol` medium value-only shared-normalizer review all clean | 2026-08-06 | Complete |
| `C4-S2` | `accepted` | `C4-COMP-NORM-01`, `C4-DL-01`, `C4-ART-01`, and `C4-ART-02`; four allocated proofs, affected/repository build and tests, focused race, conformance/ownership/whitelist inspection, and required `gpt-5.6-sol` medium external/persisted-boundary re-review all clean | 2026-08-06 | Complete |
| `C4-S3` | `accepted` | `C4-SCHED-01`, `C4-RUN-01`, and `C4-FAIL-01`; all three allocated proofs, affected/repository build and tests, changed-package race checks, conformance/ownership/whitelist inspection, and required `gpt-5.6-sol` medium order/clock/commit/lifecycle focused re-review clean | 2026-08-06 | Complete |
| `C4-S4` | `accepted` | `C4-CORE-01` / `P-C4-CORE`, runnable offline replay, all nine Component 4 proofs, affected and repository build/tests/race/vet, conformance and drift audit, and focused final-review confirmation all clean | 2026-08-06 | Complete |
| Final component review | `accepted` | Mandatory read-only `gpt-5.6-sol` medium review found five actionable issues; all received direct regressions and corrections, the same reviewer found the complete correction set clean, and its focused review confirmed the shortened `R == E` proof preserves `C4-RUN-01` / `P-C4-RUN` without making a deferred capacity claim | 2026-08-06 | Complete |
| `C4-S5` requested-end correction | `accepted` | `C4-PREFIX-END-01` / `P-C4-PREFIX-END`; exact prefix/artifact-end parity/suffix-integrity/cancellation/clock/rejection/accounting matrix, affected and repository short tests, proportionate race, vet, conformance inspection, and required `gpt-5.6-sol` medium persisted-source/lifecycle focused re-review all clean | 2026-08-09 | Complete |
| `C4-S6` bounded manual-driver correction | `accepted` | `C4-BOUNDED-CANCEL-01` / `P-C4-BOUNDED-CANCEL`: bounded untrusted header probe; context-aware validation and both playback passes; one source-owned serialized `Cancel(ctx)`; retained nonterminal and terminal dispositions; exact terminal accounting; and an honest `(Result,error)` one-shot seam whose existing C4 command consumer supplies a separate bounded cleanup context. Thirty focused lifecycle repetitions, ten focused race repetitions, full affected/repository short/race/vet/diff/conformance checks, and required `gpt-5.6-sol` medium review/correction/re-reviews are clean. | 2026-08-09 | Complete; advance to C12-S1 |
| Production-reader resource correction | `accepted` | `P-NARROW-READER-RESOURCE`: diagnosis found linear approximately 7,667 B/record allocation and 86.25% of allocation in duplicate strict aggregate canonical work. The correction reduces the 500,000-record rung by 53.44% wall and 57.22% allocation; corrected 100,000--2,000,000 scaling is linear with flat approximately 22 MB RSS. The exact 2,584,011,150-byte complete artifact then validates 7,671,171 records, 5,691 coverage entries, and 172 empty symbols in 40.53 seconds at 20.30 MB peak reader RSS. Compact/adversarial trust proofs, repository short, focused race, vet/diff, and required `gpt-5.6-sol` persisted-trust review are clean with no P1/P2. The prior external Gate E command-session termination remains preserved and is not rewritten as a pass. | 2026-08-11 | Component 4 re-accepted; replacement Gate E composition remains separately owner-authorized and unexecuted |

**C4-S6 active correction record:** The accepted C11/private V1 RC prerequisite
completed at commit `3644ed1`, so Component 4 reopened only
`C4-BOUNDED-CANCEL-01`; all S1-S5 semantics and evidence remain unchanged. New
context entry points wrap the accepted noncontext APIs without changing artifact
schema, identity, replay order, engine facts, or terminal meanings. Candidate
header output deliberately omits artifact identity and success/coverage claims.
The source operation gate and lifetime make `Cancel(ctx)` the sole manual
shutdown owner: cancellation before terminal linkage prevents later source
facts and joins already-linked work before one controlled stop; an already
linked replay end/failure seals the original result. A timed-out cancel returns
no result and a later call joins the same operation. No V2 source, provider,
credential, C12 code, or retained B4 artifact was accessed.

The focused review exposed that the original S6 file boundary could not retain
both the existing one-shot command and caller-deadline honesty: a `Result`-only
`Run(ctx)` either fabricated a terminal result or hid cleanup past an expired
context. The correction loop therefore expanded only to the existing Component
4 `cmd/aggregate-replay` consumer and its tests. `Run` now returns
`(Result,error)`; error with no terminal result requires the caller to invoke
the same source-owned `Cancel` under a separately supplied cleanup deadline.
The command uses a two-minute bound and encodes only the reconciled result. No
C12, engine semantic, provider, or artifact scope changed.

**C4-S6 acceptance record:** `ProbeCandidateHeader` reads at most the bounded
canonical header line and exposes no artifact ID, coverage, validated handle,
or success fact. Full validation, first playback, terminal scan, and requested-
end suffix scan accept contexts, check between bounded reads and immediately
before success, and return no trusted handle/end evidence when cancellation
wins. Manual `Start`/`Step`/`Finish` operations serialize with idempotent
`Cancel(ctx)`. Cancellation before terminal linkage stops new facts, joins any
linked start/aggregate/timer disposition, admits at most one controlled stop,
closes/drains the engine, clears `ActiveGroup`, and seals exactly one canceled
result. A linked replay-end or replay-failure disposition instead wins; failure
is not sealed until exact `replay_failed` disposition and engine drain.

`P-C4-BOUNDED-CANCEL` covers candidate/validation final checks, first-pass and
suffix mid-scan cancellation, pre-start/pre-step/pacing cancellation, linked
nonterminal deadline/retry, cancellation on both sides of terminal and failure
linkage, cancel timeout/retry, post-terminal idempotence, and concurrent
`Run`/`Cancel`. Dangerous false states are either unconstructible (candidate
cannot be a handle; end evidence remains opaque) or rejected/joined at runtime
(unlinked failure, wrong disposition, stranded active accounting, duplicate
stop, or contradictory result). Success walks from validated bytes through one
terminal disposition and drain; cancellation/failure walks through the same
source owner and returns no terminal value on deadline before completion.

The required reviewer found four initial lifecycle defects: false failure
sealing without admitted failure evidence, deadline-bypassing implicit cleanup,
contradictory concurrent `Run`/`Cancel` results, and stranded active-group
accounting. After direct regressions/corrections it found one `Run` API/command-
consumer compatibility defect; the `(Result,error)` and explicit caller cleanup
correction received a clean final re-review. The proof limitation remains small
repository-owned artifacts and local process cancellation; it proves no B4-
scale throughput, provider behavior, adversarial filesystem immutability, or
process-kill cleanup. No B4 artifact, credentials, provider endpoint, or V2
source was accessed.

**Approved C4-S6 correction premise:** C12's manual `Start`/`Step`/`Finish`
composition must be cancelable without letting C12 become an engine/source
lifecycle owner. This contract assigns C4 a bounded, idempotent
`Cancel(ctx)` for a started manual source and context-aware candidate-header,
first-pass, terminal, and suffix scans. The candidate header is expressly
untrusted and can select a date-specific reference cache only; it cannot prove
identity, binding, coverage, or playback validity. Cancellation before terminal
fact linkage uses the existing C4 controlled-stop, source-outcome/accounting,
and engine close/drain path within the caller's context. Once the accepted
terminal fact links, its complete or failed disposition wins exactly as today.
No handle or terminal fact may be returned after the operation context is
observed canceled and before the corresponding success linearization point.

This correction owns no C12 wall schedule, phase, snapshot/API/UI behavior,
provider access, retained B4 execution, or licensed-data read. It does not
change accepted C4 behavior until implementation activates. `C4-S6` must be
implemented and accepted before `C12-S1`; C12 may then consume the seam but may
not re-prove or reimplement its
internal cancellation mechanics. Activation first marks Component 4 reopened
for only the bounded-manual-driver claim while preserving accepted S1-S5
evidence; clean S6 proof, verification, review/re-review, and conformance return
the component to accepted before C12 may consume it.

**C4-S5 correction premise:** Accepted C4 correctly proves artifact-end success
at `R`, but its source cannot truthfully finish at an operator-requested
`O1<R`; stopping there is cancellation. The owner approved one additive C4
correction on 2026-08-09. A validated complete artifact remains authoritative
for its unchanged header interval `[S,R)`. An exact UTC whole-second
`O1` with `S < O1 <= R` may separately bound canonical application. Records
and deterministic timers through `O1` use the ordinary engine path; later
records never do. Before requested-end success, the same-open second pass must
consume and validate the unapplied suffix through the canonical coverage,
summary, digest, final seal, and unchanged-file checks. `O1=R` continues to use
the existing artifact-end evidence and disposition. Invalid or incompatible
requested ends fail before replay start; later ordinal/group/clock/engine or
requested-end contradictions fail/suppress without a complete claim.

The terminal-fact FIFO linkage is the exact cancellation linearization point:
cancellation observed before requested-end admission remains canceled; after
admission succeeds, the sealed engine owns that accepted terminal fact and its
requested-end disposition wins. Admission must be sampled by the engine at
exactly `O1`; a different engine time fails/suppresses.

The correction owns no C12 API/UI/runtime composition and authorizes no
provider request, credential access, licensed-data read, or retained B4
artifact use. It preserves the existing artifact schema and identity, session
and half-open-window semantics, configured evaluation delay `D`, sole engine/
clock/evaluator/publication owners, and ordinary cancellation/controlled-stop
meaning.

**C4-S5 acceptance record:** `NewSourceThrough` and the validated-handle
playback seam now accept an exact UTC whole-second `O1` in `(S,R]`. At `O1<R`,
the source applies only prefix records and ordinary group timers through `O1`,
then the private cursor consumes the remaining groups solely to recheck
canonical order, coverage, summary, digest, seal, and final file identity. The
opaque requested-end fact carries full record count plus applied prefix ordinal;
the engine admits it only when artifact/binding/end, last group/ordinal, and its
own sampled clock exactly equal the started run and `O1`. It ends through the
ordinary sole lifecycle/publication owner with completion `requested_end`.
`O1=R` retains the unchanged artifact-end evidence and `artifact_end`
completion path.

`P-C4-PREFIX-END` passes in `TestC4PREFIXEND01RequestedEnd`. Its exact-prefix
success case preserves nonzero evaluation delay `D`, applies no suffix record
or timer, and closes `artifact_records = completed_record_dispositions +
intentionally_unapplied_suffix_records + unread_records`. The dangerous cases
prove that invalid/subsecond/out-of-range/partial ends fail before replay
mutation; missing prefix evidence is unavailable; a changed suffix, foreign
requested-end evidence, wrong terminal clock, ordinal/group/engine rejection,
or suppression cannot retain completion; pre-link cancellation remains
canceled; and post-link cancellation cannot displace the sealed terminal fact.
Construction keeps terminal evidence opaque and suffix groups inside the
cursor; runtime validation owns every representable identity/order/clock/
disposition contradiction. No accepted C1-C4 owner/interface, artifact schema,
version 2 whitelist, or C12/API/UI boundary changed.

Focused affected-package tests, `go test -short -timeout 2m ./...`, `go test
-race -short -timeout 5m ./internal/engine ./internal/replayartifact
./internal/replay -count=1`, `go vet ./...`, formatting, and diff checks pass.
The required read-only `gpt-5.6-sol` medium review found four issues: requested-
end admission was not bound to sampled clock `O1`, cancellation linkage was
contradictory/unproved, active contract text retained artifact-end-only baseline
statements, and the same-open proof overclaimed adversarial mutation detection.
All received direct code/proof/contract corrections; the same reviewer's
focused re-review is clean and found no false-success, suffix-to-engine,
accounting, lifecycle, concurrency, C12, or provider-scope problem.

The proof uses only small repository-owned artifacts. It neither reads the
retained B4 artifact nor establishes large-artifact throughput. Same-open
integrity assumes a trusted local filesystem: reread-byte and final size/mtime
checks do not prove an adversarial immutable snapshot if a writer restores
metadata after changing bytes already buffered or reread. No provider request,
credentials, or licensed-data read occurred. C12 remains separate and may now
consume this accepted C4 seam under its own future contract; it was not
implemented here.

**C4-S1 acceptance record:** One stateless `internal/massive` mapper now turns
exactly one raw Massive REST second-aggregate row into one value containing the
opaque exact symbol, UTC `[t,t+1s)`, Component 2 `AggregateValues`, and
`rest_floor_volume_over_transactions` provenance, or returns the zero value
with one of four bounded rejection classes. This completes `C4-NORM-01` only.
HTTP and envelope handling, compiler consumption and its sole-mapper proof,
artifact compilation/persistence, replay/recovery/coverage, engine mutation,
binding/symbol/interval membership, and Component 6 fencing/production reuse
remain deferred exactly as approved.

`P-C4-NORM` passes against the renamed byte-identical approved
`rest-second-bars.json` fixture (SHA-256
`be30ea406fd7fe5c50642a1851b5ebc63cfc98d149caac995e1c91412a1df7c5`)
and repository-owned invalid rows. It proves exact sparse windows and canonical
values, fractional-volume preservation, zero and fractional floor ATS, signed-
zero equivalence, exact-count behavior beyond `2^53`, REST provenance, additive
unknown-member handling, strict recognized-member decoding, and atomic zero-
value rejection for syntax, duplicate, missing, timestamp, numeric/count,
overflow, and structural failures. `go test ./internal/massive -run
'^TestNormalizeRESTSecondAggregate$' -count=1`, `go test ./internal/massive
-count=1`, `go test ./internal/engine -count=1`, `go build ./...`, and `go test
./... -count=1` pass. This proof does not establish provider availability,
HTTP/envelope/response-size behavior, a compiler or Component 6 consumer,
binding membership, live ATS parity, artifacts, or replay.

Construction keeps all decode/arithmetic state call-local, exposes no request,
coverage, artifact, replay, recovery, delivery-position, or engine-mutation
handle, and can return only a complete value or the zero value plus a closed
reason.
Runtime validation rejects malformed or duplicate recognized members, missing
or nonnumeric fields, non-whole/overflowing timestamps, nonfinite/invalid
canonical values, nonpositive/nonintegral/overflowing transaction counts, and
unrepresentable ATS. The success path strictly decodes all eight recognized
numbers, constructs the exact one-second window, validates canonical structure,
computes the mathematical floor over canonical `float64` volume and exact
`int64` transaction count, then returns the existing Component 2 value type.
The dangerous failure path—such as fractional/missing `n`, fractional `t`, or
an ATS overflow—terminates before any partial aggregate can escape. One call
has no retained mutable state or population accounting; source inspection finds
one production Massive REST field mapper and no I/O, goroutine, callback,
registry, state mutation, or second value schema. These statelessness,
sole-mapper, and ownership facts are inspection-supported claims.

The required read-only `gpt-5.6-sol` medium review found no actionable numeric,
timestamp, signed-zero, ATS, proof, statelessness, seam-ownership, scope, or
whitelist finding. The consequential provider-normalization/dependency seam is
therefore accepted for later consumers without accepting their behavior. There
was no contract, interface, document-map, or predecessor-whitelist deviation,
no failed assumption or newly discovered contradictory evidence, and no drift-
audit `yes`. The delegated gate is clean and `C4-S1` is accepted.

**C4-S2 acceptance record:** A bounded fake-testable `internal/massive`
downloader now requests the exact binding symbols and interval with the fixed
unadjusted ascending Massive query, bearer authentication, bounded retries,
deadlines, pages, response bytes, records, and workers. Every raw result reaches
the accepted S1 mapper exactly once; only a sealed all-symbol-complete
`DownloadResult`, including explicit successful-empty outcomes, can enter the
new `internal/replayartifact` compiler. That compiler emits the exact closed
`aggregate-replay-jsonl-v1` bytes, complete or declared-partial coverage,
summary and SHA-256 seal, validates the same already-open regular file, and
publishes by sibling hard-link no-replace plus one directory sync. The
compile-only `cmd/aggregate-replay` wiring composes the accepted Component 1
binding and warns unless the local provider-data destination is proved under
this repository's ignored `var/` root. No provider credential or live request
was used in acceptance.

`P-C4-COMP-NORM`, `P-C4-DL`, `P-C4-ART-BYTES`, and `P-C4-ART-TRUST` pass. They
prove the sole mapper/consumer path; exact query, auth containment, successful
empty, retry/deadline/page/body/global-byte/record/worker bounds, foreign/cyclic/
third continuation rejection, cancellation, and closed terminal accounting;
the normative five-line golden bytes (`body_bytes=1008`,
`sealed_bytes=1106`, artifact ID
`sha256:31ef4b0cbaa85a0b60b3bd70acbbfa4162b9386c539ded580b4ed4897d18d901`),
exact `strconv.AppendFloat(value,'g',-1,64)` numbers, signed-zero and input-order
invariance, sparse/empty coverage, and partial-mode separation; and whole-file
binding/schema/order/count/coverage/digest validation, idempotent existing-file
adoption, conflict preservation, lease/remnant bounds, FIFO/symlink rejection,
pre/post-publication cancellation, cleanup failure, and directory-sync
uncertainty. The focused package tests, `go test -race ./internal/massive
./internal/replayartifact -count=1`, `go build ./...`, `go test ./... -count=1`,
formatting, and `git diff --check` pass. These proofs do not establish provider
SLA or live capacity, licensing rights, cryptographic authorship, power-loss
behavior beyond filesystem guarantees, process-OOM cleanup, Component 6 reuse,
or replay/engine clock and lifecycle behavior.

Construction prevents raw provider rows, credentials, paths, worker order, and
request diagnostics from entering artifact bytes; a complete artifact cannot
be built from an unsealed or noncomplete download; modes and schemas are
closed; complete coverage derives from every binding symbol; canonical sort
and encoding determine ordinals and identity; and package-private persistence
fault seams have no production callback path. Runtime rejects non-200/error/
foreign/malformed/trailing/oversized responses, invalid or duplicate rows,
untrusted continuations, budget exhaustion, wrong binding or interval,
noncanonical/corrupt/truncated/incomplete artifacts, nonregular existing paths,
destination conflicts, cancellation before publication, and uncertain cleanup
or directory durability. The accounting identities
`planned=complete+failed+canceled`, `complete=nonempty+empty`, exact per-symbol
coverage counts, and one compile terminal outcome are closed and tested.

On the success path, the downloader proves every symbol outcome, the compiler
sorts only accepted immutable values within the explicit in-memory ceiling,
encodes and seals the artifact, syncs and validates its sibling temporary file,
atomically installs the content-addressed name, removes temporary/lease names,
syncs the directory once, and returns `complete`. On the dangerous failure
path, an error page, failed mapper row, missing binding coverage, FIFO or
symlink destination, conflicting bytes, or pre-link cancellation produces no
validated handle and cannot overwrite or masquerade as complete; a failure
after publication linearizes is reported only as `persistence_uncertain` and
can be adopted later only through full validation.

Source inspection finds one production Massive REST field mapper and one call
to it in the downloader, no raw-row codec bypass, no credential/raw-response
persistence, and no replay, recovery, hydration, engine mutation, trade/quote,
or Component 6 ownership. Only the approved version 2 regions and renamed
byte-identical fixture were used. The required read-only `gpt-5.6-sol` medium
review initially found four bounded path/state issues—temporary-limit overflow,
destination-warning containment, blocking/nonregular artifact opens, and
cross-worker response-budget overrun—which were corrected and directly proved;
its focused re-review found no remaining actionable issue. There is no contract,
interface, document-map, or predecessor-whitelist deviation, no unresolved
inspection-only claim capable of invalidating success, and no drift-audit
`yes`. At that delegated gate, `C4-S2` was accepted and the approved `C4-S3`
assignment remained exactly valid and became authorized, but was not
implemented by the S2 assignment.

**C4-S3 acceptance record:** A sequential `internal/replay` source now consumes
the validated artifact's same-open streaming pass, advances one source-owned
whole-second simulated clock through every group from `S` through `R`, awaits
each record disposition in global ordinal order, and applies exactly one
engine-owned timer even on quiet seconds. The existing `ScannerStateEngine`
owns replay lifecycle, system/engine sequence, canonical merge, evaluation,
publication, and committed `T`. Complete artifacts classify every binding
symbol's finished slot as present or artifact-proved absent before timer and
commit; partial artifacts may exercise insert/revision behavior but cannot
install complete support or commit. Successful end requires the final group,
second-pass digest/seal, typed engine end, and drain. Failure produces one
bounded reason and `terminal_replay_failure`; cancellation remains a distinct
controlled stop without artifact-end success. This implements only
`C4-SCHED-01`, `C4-RUN-01`, and `C4-FAIL-01`.

`P-C4-SCHED`, `P-C4-RUN`, and `P-C4-FAIL` pass via `go test
./internal/replay -run 'TestC4(SCHED01|RUN01|FAIL01)' -count=1`. They prove
unpaced/finite-pace logical trace equality across sparse groups, cumulative
rational pace including the reducible `1e9/1e9` bound, complete presence and
absence before commit, successful-empty population, partial correction without
commit, exact record/group accounting, and terminal containment for a changed
same-open file, a fully valid pre-playback artifact-identity substitution,
ordinal discontinuity, binding/coverage and schema/canonical tail mutation,
truncated EOF, engine rejection, clock regression, and cancellation. The
failure cases assert exact bounded reason and last supported logical/ordinal
boundary. `go test ./internal/replayartifact/playback ./internal/replayartifact
./internal/replay ./internal/engine`, `go build ./...`, `go test ./...
-count=1`, `go test -race ./internal/engine ./internal/replayartifact
./internal/replay -count=1`, formatting, and `git diff --check` pass. These
proofs do not establish process-restart orchestration, production wall-timing
accuracy, live-provider/T/Q/readiness behavior, checkpoints, recovery, or
`C4-S4` output breadth.

Construction makes successful start/record/group/end evidence opaque outside
the validator, binds playback to the artifact ID captured by `OpenValidated`,
keeps only one decoded record and linked completion in flight, gives only the
source a simulated-clock writer, and makes complete coverage unrepresentable
for partial evidence. Runtime validation rejects incompatible header/coverage,
noncanonical bytes, ordinal/group contradictions, wrong clock or engine
dispositions, and invalid end evidence. The success path validates the whole
same-open file, rewinds it, streams records and quiet groups through the sole
engine/evaluator/publisher path, revalidates the seal and unchanged file
identity, and ends only after drain. The dangerous failure path—rewriting the
validated inode with another valid artifact, editing a later ordinal or tail
classification, truncating the seal, or rejecting an admitted fact—suppresses
the run at its last supported boundary and cannot be repaired in place.

Source accounting preserves `artifact_records =
completed_record_dispositions + unread_records`, `planned_groups =
completed_groups + active_group + remaining_groups`, and exactly one terminal
run outcome. Inspection confirms one source writer, one committed-`T` apply
point, one evaluator path, one publication-cell store path, no replay worker
pool, and no live/T/Q intent. No Component 4 contract, Components 1–3
interface, document map, or predecessor whitelist changed; no additional
predecessor scope was inspected, and no credentials or live requests were
used. The required read-only `gpt-5.6-sol` medium review initially found
artifact-ID substitution, pace-overflow, and exact-classification proof gaps;
all were corrected and directly proved. The same reviewer's focused re-review
found no actionable issue. There is no failed assumption, unresolved
inspection-only claim capable of invalidating success, or drift-audit `yes`.
The delegated gate is clean: `C4-S3` is accepted and `C4-S4` was authorized but
not implemented by this slice.

**C4-S4 acceptance record:** The accepted slice adds the
runnable offline replay command path and `P-C4-CORE` representative
cross-pace proof. The command reads a validated artifact and exact same-date
Component 1 binding from private caches without credentials or HTTP fallback,
uses explicit replay bounds/capacity/reserve/evaluation-delay controls, installs
the binding in the sole engine, drives the sequential replay source, and emits
the terminal result. The proof compares every logical group between unpaced and
accelerated runs over qualified, sparse, and successful-empty symbols, including
lifecycle, sequences, committed `T`, canonical state, features, qualification,
population accounting, rank rows, and publication identity/content. T/Q remains
explicitly unavailable. This is the approved `C4-CORE-01` boundary; no
predecessor source, live provider, credentials, checkpoint, recovery, readiness,
API/UI, or T/Q replay behavior was added or inspected.

All nine primary proofs pass: `P-C4-NORM`, `P-C4-COMP-NORM`, `P-C4-DL`,
`P-C4-ART-BYTES`, `P-C4-ART-TRUST`, `P-C4-SCHED`, `P-C4-RUN`, `P-C4-FAIL`,
and `P-C4-CORE`. `P-C4-RUN` includes a complete successful-empty three-group
tail interval `[E-2s,E)` that proves the exact `R == E` lifecycle boundary and
typed replay-end ordering without claiming deferred full-capacity performance.
`go build ./...`, `go test ./... -count=1`, `go test -race ./... -count=1`,
`go vet ./...`, the focused affected tests, formatting, and the ownership/
whitelist/drift audit pass. The mandatory
read-only final Component 4 review used `gpt-5.6-sol` at medium reasoning. It
found five issues—fractional pace overflow, direct-finish cancellation,
rejecting-record accounting, `R == E` lifecycle termination, and pre-start
artifact identity reporting—which were corrected with direct regressions; the
same reviewer's focused re-review was clean. After the initial maximal-duration
test exposed only an unallocated verification-cost problem, the owner
authorized the contract-conformant short-tail regression; the same reviewer
confirmed it preserves `C4-RUN-01` / `P-C4-RUN`, and the exact repository-wide
race command then passed.

Construction retains one replay-source clock writer, one engine state owner,
one committed-`T` apply point, one evaluator path, and one publication-cell
store path. Runtime containment preserves the artifact-record and planned-group
accounting identities, reports the validated artifact identity on pre-start
failure, rejects overflow and invalid record dispositions, and cannot turn
cancellation, corruption, incomplete coverage, or failed end evidence into
success. The successful path remains the validated same-open stream through the
sole engine/evaluator/publisher path. There is no contract, interface,
document-map, or whitelist deviation and the drift audit has no `yes`.

There is no failed assumption, unresolved inspection-only claim capable of
invalidating success, or drift-audit `yes`. Every allocated proof and required
verification passes, the implementation remains inside the approved scope and
whitelist, and the mandatory final review is clean after focused correction.
The delegated gate therefore accepts `C4-S4` and finally accepts Component 4;
no later Component 4 slice remains.

## 1. Outcome and user consequence

Component 4 makes the approved aggregate scanner reproducible offline. It
obtains historical Massive one-second aggregate rows, maps them through the one
provider-specific REST aggregate-row normalizer later reused by Component 6,
compiles a versioned provider-independent artifact with explicit provenance and
coverage, and replays that artifact through Components 1–3 using deterministic
logical delivery and an injected simulated clock.

At different playback speeds, the same artifact must produce identical
logical-time lifecycle, committed-watermark, canonical, Component 3, and
snapshot results. This is the Components 1–4 deterministic aggregate-core
milestone:

```text
Component 1 binding
  + normalized aggregate artifact
  -> Component 4 replay source and simulated clock
  -> Component 2 canonical engine path
  -> Component 3 ordinary aggregate evaluator
  -> deterministic immutable replay output
```

Incomplete evidence cannot produce a whole-session or complete-population
claim. Artifact, order, clock, or canonical integrity failure terminates only
that replay run. Replay output is nonlive and makes no live latency, transport,
or correction-arrival claim.

## 2. Scope and explicit non-scope

**In scope**

- Offline acquisition of historical Massive one-second aggregate rows for an
  explicit trading date, symbol population, and market interval, followed by
  bounded compilation into a versioned normalized replay artifact.
- The single Massive REST aggregate-row normalizer that maps provider REST
  fields into the approved provider-independent aggregate values and ATS
  provenance. Component 6 must reuse this normalizer rather than create a
  second REST mapping.
- Artifact version/identity, provenance, ordinals, logical delivery times, and
  coverage needed to distinguish proved absence from missing data.
- Deterministic artifact validation/reading, bounded record delivery, injected
  simulated-clock advancement, required empty-second timers, replay completion,
  and replay-only integrity failure facts supplied to the existing engine.
- The first successful replay support through Component 2's central commit
  gate and the end-to-end deterministic-core milestone.
- Explicitly partial synthetic replay scenarios that exercise duplicates,
  out-of-order delivery, and corrections without mislabeling them as compiled
  historical arrival chronology.

**Not in scope**

- Live WebSocket decoding, epochs, acknowledgements, or framing; Component 5
  owns them.
- Production REST hydration, fresh bootstrap, checkpoint catch-up, aggregate
  gap recovery, generation/request-token ownership, ingress fencing,
  REST/live reconciliation, `no_print_through(T)` proof, or terminal-work
  accounting. Component 6 owns those behaviors while reusing this component's
  REST row normalizer.
- Checkpoint contents, validation, installation, storage, or restart policy;
  Component 7 owns them.
- Production delay, readiness/capacity, shutdown, or HTTP policy; Component 8
  owns them.
- T/Q download, normalization, artifacts, replay, coverage, features, or
  pressure; `PG-REPLAY-02` defers full-session and two-pass T/Q replay.
- Changes to Component 1 binding meaning, Component 2 canonical aggregate
  identity/merge/correction/committed-watermark/publication ownership, or
  Component 3 formulas, qualification, accounting, ranking, and availability.
- A fake WebSocket, second scanner/evaluator, database/journal, service split,
  event bus/plugin system, public API schema, or UI behavior.

## 3. Ownership and dependencies

The single ownership boundary is the offline historical aggregate
row-to-versioned-artifact-to-deterministic-replay-input path. Component 4 owns
only bounded downloader/compiler state, artifact representation/validation,
the replay cursor/grouping, playback control, and the simulated clock. These
are source/I/O states, not scanner truth.

The shared REST row normalizer returns one provider-independent aggregate value
with REST provenance or one bounded rejection. It does not select a binding,
own hydration generations/tokens, establish coverage/no-print, merge, rank, or
mutate canonical state. Component 4's compiler uses it directly; Component 6
later supplies its own production request, work, and fence context around the
same result. Component 4 proves the mapper in `C4-S1` and the compiler consumer
in `C4-S2`; Component 6 must allocate its own later proof that the production
hydration path imports this seam rather than creating another mapping.

The replay source returns immutable aggregate, timer, and replay-control facts
through Component 2's FIFO and receives no writable engine state. Component 2
retains engine sequence, canonical state, merge, lifecycle, `T`, containment,
and publication; Component 3 retains the sole evaluator. The simulated clock
uses Component 2's injected-clock seam and is not a watermark or lifecycle
authority.

Component 1 supplies the binding; Component 2 supplies admission, ordinal
precedence, the replay commit gate, lifecycle, and publication; Component 3
supplies source-blind evaluation. Component 4 may extend only those seams with
later-approved replay facts and support evidence.

## 4. Settled Phase 1 semantic boundary

| Boundary item | Settled meaning | Controlling Phase 1 IDs |
| --- | --- | --- |
| Run binding | One explicit replay run uses one immutable Component 1 binding and `[S,E)`. Artifact identity, symbols, times, policy, and coverage must be compatible before `replaying`; a different binding requires a new engine. | `DTE-SESSION-01`–`DTE-SESSION-04`, `LIFE-MODEL-01`, `LIFE-INIT-01`, `LIFE-REPLAY-01`, `LIFE-T03` |
| Shared REST normalization | REST rows use the canonical aggregate identity/fields/validity. Same-session REST is unadjusted; positive transaction count maps ATS to `floor(volume/transactions)` with REST provenance. REST/live ATS equality is not required. | `DTE-MODEL-01`–`DTE-MODEL-03`, `DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-AGG-01`–`DTE-AGG-04` |
| Artifact evidence | The versioned normalized artifact carries identity, provenance, normalization policy, ordinals, logical times, and coverage. Partial evidence may drive an explicitly partial test but cannot prove complete absence or population. | `PG-REPLAY-01`, `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-REPLAY-01`, `LIFE-REPLAY-01` |
| Order, clock, and timers | Compiled final bars use `window_end`; equal-time records order by `(window_start,symbol)`. Each complete group precedes its timer, and absent market seconds remain absence while still receiving required timers. Playback speed changes wall duration only. | `DTE-CLOCK-02`–`DTE-CLOCK-06`, `DTE-WINDOW-04`, `DTE-TIMER-01`, `DTE-MERGE-01`, `DTE-MERGE-02`, `DTE-MERGE-05`, `DTE-REPLAY-02`, `DTE-REPLAY-03`, `LIFE-T23` |
| One engine/evaluator | Replay uses Component 2's bounded ordered canonical/commit/publication path and Component 3's evaluator. Only the engine assigns `T`; validated coverage/order and completed groups may support its replay commit branch. | `ARCH-OWN-01`–`ARCH-OWN-04`, `ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-COMMIT-01`–`DTE-COMMIT-04`, `LIFE-MODEL-02`, `LIFE-REPLAY-02` |
| Determinism and containment | Equal binding/artifact/configuration produce identical logical-time lifecycle, `T`, canonical, evaluator, and snapshot results at every speed. Bad binding/artifact/order/clock/canonical evidence terminates only the replay run with a bounded reason. Output is replay-labeled, nonlive, and has no version 1 T/Q. | `PG-REPLAY-01`, `PG-REPLAY-02`, `PG-OBS-03`, `DTE-AGG-04`, `DTE-REJECT-01`, `LIFE-MODEL-04`, `LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-SUPPRESS-01`–`LIFE-SUPPRESS-03`, `LIFE-END-01`–`LIFE-END-03`, `LIFE-PUBLISH-02`, `LIFE-T24`, `LIFE-T25`, `LIFE-T28`, `LIFE-T29` |

This component introduces no new product rule, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, changed time-window meaning,
or duplicated responsibility. It adds no second REST aggregate mapping: the
Component 4 normalizer is the provider-row mapping Component 6 must reuse.

## 5. Unresolved questions and evidence needs

These questions were unresolved at boundary approval. The approved
reconnaissance and routed Sections 8–14 now answer each one; no evidence
question remains open in the approved detailed contract.

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| Which Massive endpoint/query, response, pagination, timestamp, numeric, transaction-count, and status mappings are evidenced? | Phase 1 fixes canonical values and positive-count ATS, not provider wire behavior or nonpositive/missing transaction-count treatment. | Scoped v2 downloader/normalizer tests and fixtures; provider docs only for remaining mapping gaps. | Exact request and row acceptance/rejection contract. |
| What minimal row-normalizer seam can serve the compiler and Component 6 without importing recovery ownership? | The roadmap fixes one mapping and separate owners, not the typed seam or rejection vocabulary. | Scoped v2 mapping plus Components 2/6 interface constraints. | Shared seam and proof that no second mapping/owner exists. |
| What offline request, concurrency, retry, cancellation, size, and publication bounds are warranted? | Architecture requires bounded I/O but delegates numbers; offline compilation is not production hydration. | Scoped v2 downloader tests, provider limits, and repository invariants. | Bounded downloader/compiler terminal behavior. |
| What artifact format, version/identity/integrity, atomicity, coverage granularity, size, storage/privacy/retention, and provider-data restrictions are required? | Phase 1 delegates these persisted-boundary choices. | Scoped v2 artifact code/tests/fixtures and existing repository-local policy. | Artifact trust and complete-versus-partial claim rules. |
| What buffering, group-completion handshake, playback controls, cancellation, and artifact-end fact fit the approved engine seams? | Phase 1 fixes semantic order, not source controls or bounds; Component 2 left these facts absent. | Scoped v2 source/clock tests and the finalized Component 3 interface after its final review. | Successful replay support without another clock, lifecycle, or publisher. |
| Which fixtures should own the normalization, artifact, scheduling, containment, and speed-invariance proofs? | Phase 1 names claims, not fixture quality or primary-proof allocation. | Scoped v2 fixtures plus approved Components 1–3 proofs. | One primary proof per later requirement and final slice plan. |

The Component 3 implementation was not inspected while this skeleton was
prepared. After its final acceptance, the detailed phase inspected only the
finalized Component 2/3 replay/evaluator seams needed by the approved questions;
the resulting decisions are in the routed Sections 8–14. A contract mismatch
still stops Component 4 rather than authorizing another evaluator or publisher.

## 6. Approved version 2 reconnaissance scope

This section preserves the owner-approved reconnaissance boundary. The exact
inspection performed, hashes, findings, and decisions are recorded only in the
two routed Section 8 ledgers.

With Component 3 final review complete, the approved reconnaissance scope permits only
this predecessor:
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Exact paths,
symbols, commit/hash provenance, findings, and reuse decisions belong in
Section 8 after discovery.

| Approved code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| Historical one-second REST downloader/client and focused fake-provider tests/fixtures | Provider query, pagination, decoding, bounds, retries, empty/partial behavior. | WebSocket, reference endpoints, hydration/recovery planning/work, broad integration suites. |
| Massive REST aggregate row mapper/validator and focused fixtures | Exact canonical fields, timestamps, fractional volume, ATS/provenance, and evidenced rejections. | Live/T/Q mapping, merge/no-print/ranking ownership, speculative mutations. |
| Replay compiler/codec/manifest and focused artifact tests | Versioning, identity, provenance, coverage, order, integrity, bounded/atomic I/O. | Checkpoints, journals/databases, API/UI, unrelated caches. |
| Replay source, simulated clock, controls, and deterministic scheduler tests | Grouping, timers, admission completion, cancellation/end, and speed invariance. | Live cadence/readiness, fake WebSocket, generic scheduler, recovery, T/Q replay. |
| Replay-only end-to-end fixtures/tests | Useful logical-time outputs and predecessor owner/watermark/evaluator coupling to reject. | Broad test harvest, live chronology/latency claims, checkpoint/readiness/API/UI, predecessor orchestration ports. |
| Colocated artifact storage/privacy/license metadata | Whether a concrete local policy exists. | Credentials, live calls, user data, version 1, unrelated docs. |

Only the exact version 2 source, test, and fixture reconnaissance listed above
is authorized. Broader predecessor inspection, Git history, credentials, live
endpoints, implementation, and any needed scope/boundary/map expansion remain
prohibited without the applicable owner approval.

### 6.1 Initial proof and slicing boundaries

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| Shared REST normalization table | `DTE-MODEL-01`–`DTE-MODEL-03`, `DTE-WINDOW-02`, `DTE-AGG-01`–`DTE-AGG-04`, `DTE-REJECT-01` | Each evidenced row yields the exact REST-provenance aggregate or one rejection through the seam reused by Component 6. | Provider mapping and focused fixtures. |
| Downloader/compiler and artifact-trust scenario | `PG-REPLAY-01`, `ARCH-FLOW-04`, `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-REPLAY-01`, `DTE-REPLAY-02`, `LIFE-REPLAY-01` | Equivalent inputs compile identically; corrupt, incompatible, partial, or ambiguous evidence cannot appear complete. | Bounds, format/identity/coverage, policy, fixtures. |
| Delivery/clock/quiet-timer trace | `DTE-CLOCK-03`–`DTE-CLOCK-06`, `DTE-TIMER-01`, `DTE-REPLAY-02`, `DTE-REPLAY-03`, `LIFE-T23` | Groups, recordless seconds, and timers enter exactly; wall speed changes no logical result. | Source/clock seam, controls, fixtures. |
| Replay lifecycle/containment trace | `LIFE-REPLAY-01`–`LIFE-REPLAY-03`, `LIFE-SUPPRESS-01`–`LIFE-SUPPRESS-03`, `LIFE-END-01`–`LIFE-END-03`, `LIFE-T03`, `LIFE-T24`, `LIFE-T25`, `LIFE-T28`, `LIFE-T29` | Valid evidence starts and completes; integrity failure terminates only the run and requires clean rerun. | Fact shapes, finalized engine seam, failure fixtures. |
| Components 1–4 milestone comparison | `PG-REPLAY-01`, `PG-OBS-03`, `DTE-COMMIT-01`–`DTE-COMMIT-04`, `DTE-REPLAY-01`–`DTE-REPLAY-03`, `LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-02` | Different speeds yield identical logical-time lifecycle, `T`, canonical, Component 3, and snapshot outputs through the sole path, with no live/T/Q claim. | Final Component 3 interface and representative artifacts. |

**Provisional delivery assessment:** multiple sequential slices are likely, but
the final slice count and allocation are intentionally unresolved.

**Reason and likely slice outcomes:** Normalization/artifact trust must precede
runtime replay; replay source/clock/lifecycle must precede the Components 1–4
milestone proof. Reconnaissance, the finalized Component 3 interface, and proof
design determine the final split; no count or schema is approved here.

## 7. Boundary-approval checkpoint

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, ownership, dependencies, scope, and non-scope are unambiguous.
- [x] Settled inputs, outputs, local source state, canonical-owner boundaries,
      ordering, clocks, lifecycle, completion, and determinism prevent version
      2 from changing the architecture.
- [x] The approved modular map gives each future detail one cohesive and
      nonoverlapping normative responsibility.
- [x] Unresolved questions are genuinely delegated provider, representation,
      boundedness, integration, or proof details.
- [x] The approved version 2 reconnaissance is narrow and question-driven; it
      remained blocked until Component 3 final acceptance on 2026-08-06.
- [x] No version 2 code, tests, fixtures, Git history, credentials, live
      provider resources, or current implementation code/tests were opened
      while preparing this skeleton.
- [x] Initial likely proof boundaries are identified without inventing a broad
      test matrix or allocating final component requirements.
- [x] Multiple sequential slices are likely because external normalization/
      artifact trust, runtime clock/lifecycle integration, and end-to-end
      deterministic-core proof are distinct dependencies; the final plan is
      deferred until evidence exists.

**Boundary-stage owner decision:** approved 2026-08-06 for the Phase 1 boundary,
modular document map, and exact future version 2 reconnaissance scope. At that
stage this decision did not authorize detailed behavior, reuse, proofs, slices,
or implementation; the later complete-contract approval recorded above does.

Component 3 has passed final component review. The exact approved version 2 and
finalized Component 2/3 interface reconnaissance is recorded in the detail
specifications, Sections 8–19 are complete, and the owner has approved the
complete Component 4 contract. That approval initially authorized only
`C4-S1`. `C4-S1` through `C4-S6`, the original mandatory final review, and all
focused correction re-reviews have passed their delegated gates, so Component
4 is complete and finally accepted as of 2026-08-09. Broader predecessor
inspection and credentials/live calls remain unauthorized. The S6 activation
reopened only its bounded-cancellation claim and preserved all accepted S1-S5
evidence.
