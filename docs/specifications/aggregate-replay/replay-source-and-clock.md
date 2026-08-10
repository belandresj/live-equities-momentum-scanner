# Aggregate replay — Source, simulated clock, and lifecycle

**Parent contract:** [Aggregate replay](../aggregate-replay.md)

**Normative responsibility:** Validated-artifact playback, simulated clock and
whole-second timers, typed engine replay evidence, replay lifecycle/completion,
and run-local failure/cancellation containment.

**Controlling requirements:** `PG-REPLAY-01`, `PG-REPLAY-02`, `PG-OBS-03`,
`ARCH-OWN-01`–`ARCH-OWN-04`, `ARCH-FLOW-01`–`ARCH-FLOW-04`,
`DTE-CLOCK-02`–`DTE-CLOCK-06`, `DTE-WINDOW-04`, `DTE-EVENT-01`,
`DTE-EVENT-04`, `DTE-TIMER-01`, `DTE-MERGE-01`, `DTE-MERGE-02`,
`DTE-MERGE-05`, `DTE-COMMIT-01`–`DTE-COMMIT-04`,
`DTE-REPLAY-01`–`DTE-REPLAY-03`, `DTE-REJECT-01`,
`LIFE-MODEL-01`, `LIFE-MODEL-02`, `LIFE-MODEL-04`, `LIFE-INIT-01`,
`LIFE-REPLAY-01`–`LIFE-REPLAY-03`, `LIFE-SUPPRESS-01`–`LIFE-SUPPRESS-03`,
`LIFE-END-01`–`LIFE-END-03`, `LIFE-PUBLISH-02`, `LIFE-T03`,
`LIFE-T23`–`LIFE-T25`, `LIFE-T28`, `LIFE-T29`; `C4-SCHED-01`,
`C4-RUN-01`, `C4-FAIL-01`, and `C4-PREFIX-END-01`; owner-approved
`C4-BOUNDED-CANCEL-01`

**Allocated slices:** accepted `C4-S3`, `C4-S5`; owner-approved and queued
`C4-S6`

**Document dependencies:** [Parent](../aggregate-replay.md),
[REST normalization and artifact trust](rest-normalization-and-artifact.md),
Component 1 [binding contract](../reference-data-and-session-binding.md),
Component 2 [engine contract](../scanner-state-engine-and-canonical-state.md),
and Component 3
[evaluator contract](../aggregate-features-qualification-ranking-and-accounting.md)

**Approval state:** Approved 2026-08-06 as part of the complete parent contract;
the additive C4-S6 plan was owner-approved 2026-08-09 and remains queued. This
detail has no independent delivery state.

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-replay.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance used version 2 commit
`5f92a151dd850002578a33a81ad90dea096c63b6`; exact inspected bytes are named
below. Both inspected replay-named files were untracked worktree files, so they
are evidence at the recorded hashes and are not attributed to that commit. The
approved file inventory contained no replay compiler, codec,
validated artifact source, simulated clock, replay controls, or deterministic
aggregate-core integration.

| Exact source | File SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `docs/refactor/phase-5-market-replay-outline.md` | `a42dcd0e2b58eaa35d0ce90485f7260436c9d2ebd5d3166a54f0760228e1036f` | A 23-line deferred outline says future replay should use the production decoder/owner, preserve chronology, distinguish replay from live, and resolve credentials/licensing/raw-data handling. It defines no artifact or runtime behavior and includes live/TQ aspirations outside version 1. | Reject as implementation; retain only behavior cautions | Use the approved normalized-artifact boundary, not a captured live decoder or fake transport. Exclude T/Q and live chronology; preserve the nonlive label and secret/provider-data safeguards. | `P-C4-SCHED`; `P-C4-RUN`; `P-C4-CORE` |
| `internal/massive/phase0_lifecycle_test.go`: exact function `TestPhase0DeterministicReplay` only | `7ade3a3eedee10b7ac77eeb4635ac5640a2a18501bd4fb4c96557dbdd38ca1cd` | Calls a deterministic helper twice and compares two fixed legacy structs. It does not read an artifact, simulate time, deliver timers, exercise the current engine/evaluator, prove order/coverage, or traverse replay lifecycle. | Reject | Do not reuse the test name or result as replay evidence. Build the Phase 1 trace through Components 1–3. | `P-C4-CORE` |

The accepted Component 2/3 implementation reconnaissance found the exact seam
the detailed contract must extend:

- `engine.Config` already fixes `RunModeReplay`, the injected `Clock`, queue
  capacity/reserve, and explicit evaluation delay;
- `AggregateInput` already has replay source, artifact ID, record ordinal,
  logical delivery time, and REST ATS provenance;
- aggregate identity/merge, FIFO engine sequence, timer system sequence,
  publication, evaluator, and failure containment already have one owner;
- replay aggregates currently have no production proof admission; replay
  binding remains `initializing`; and the central commit gate deliberately
  holds `validatedReplayArtifact`, coverage/order, and completed-group
  predicates false for Component 4; and
- Component 3 is source-blind and already runs inside the same transition and
  private publication path. No Component 4 evaluator or publisher is needed.

These current interfaces are accepted dependency evidence, not predecessor
reuse. They require a narrow typed extension inside the existing engine rather
than a scheduler-owned watermark or mutable state copy.

**Approved implementation whitelist:** no version 2 source or fixture for this
boundary. The outline and legacy test are evidence to reject, not copy.

## 9. Detailed semantic inputs, outputs, and owned state

| Item | Meaning and required provenance/identity | Bounds or ownership |
| --- | --- | --- |
| Validated artifact handle | An immutable, binding-checked first-pass result and rewound read-only cursor over the same already-open file description produced only by the artifact validator. It carries the exact canonical wire schema/ID/mode, `[S,R)`, record count, coverage class, and sealed group/ordinal evidence. It never reopens the pathname between validation and playback. | Component 4 owns the handle, file description, and cursor. Complete evidence is nonforgeable through ordinary construction outside its validator; a second pass rechecks the same bytes/digest during playback. |
| Replay binding/config | One immutable Component 1 binding plus existing Component 2 replay mode, explicit evaluation delay, capacity, and reserve. Playback pace is separate and excluded from engine correctness configuration. | One engine instance and artifact per run; no binding or mode switch. Component 8 later owns production defaults, so this contract supplies none. |
| Simulated clock | Exact UTC whole-second logical engine time, initialized at `S`, readable through Component 2's injected clock, and advanced only by the replay source. | One writer, nonregressing, bounded to `[S,R]`. It is not `T`, receipt wall time, lifecycle, or a second scheduling clock. |
| Playback pace | `unpaced` or a checked positive rational logical-seconds/wall-second setting. It controls only cancelable wall waiting before a logical advance. | Replay-source state only. Wall clock/wait results never enter artifact records, engine facts, source positions, snapshots, or comparison keys. |
| Replay-start evidence | Typed opaque evidence that the exact binding, canonical artifact bytes, provenance, policy, order, and coverage mode passed validation. It names artifact ID, `[S,R)`, total records, and whether complete-binding commit support is permitted. | One accepted start per initializing replay engine. Only this fact may satisfy `LIFE-T03`; partial synthetic mode enters replaying with commit support permanently false. |
| Replay aggregate | Existing Component 2 normalized aggregate envelope with source `replay`, exact binding/artifact ID, positive immutable artifact ordinal, artifact logical delivery time, and no live/historical position. | Admitted only through the typed Component 4 replay path while `replaying`. The engine remains canonical/merge/evaluator owner. |
| Replay group/timer evidence | One typed timer admission for each whole second `t` from `S` through the selected application end (`R` normally, `O1` for requested-end replay), after all artifact records with logical time `t` have completed. It names the validator-produced expected last ordinal for that group. For complete final bars at `t>S`, it also proves that every binding symbol's slot `[t-1s,t)` is represented by its accepted bar or by artifact-proved absence. The engine assigns admission time and system sequence. | Exactly `(selected_end-S)/1s+1` timers. It reuses the engine timer transition and central commit gate; the caller cannot provide a different timer timestamp/sequence or a partial-mode absence claim. |
| Replay-end evidence | Validator/cursor proof that the second pass reached the canonical seal, recomputed the same artifact ID, consumed exactly all records, and completed the timer at `R`. | Exactly one artifact-end terminal fact when selected end is `R`; the engine owns `LIFE-T24`, final publication, and `ended`. |
| Replay-failure evidence | Bounded artifact/source reason and the last supported artifact/logical/ordinal boundary. It can only remove claims. | At most one terminal failure cause. The engine owns suppression and `terminal_replay_failure`; the instance is never repaired in place. |
| Run result | Immutable local result: complete, failed, or canceled; binding/artifact IDs when validated; last logical time/ordinal; exact bounded counters/reason; and completion of engine drain. | Component 4 report, not a public snapshot/API schema and not a live-readiness claim. Component 10 remains the public schema owner. |
| Requested replay end | Optional exact UTC whole-second `O1` with `S < O1 <= R` for a validated `complete_final_bars` artifact whose header remains `[S,R)`. It bounds engine application, not artifact identity or coverage. | Validated before replay start. `O1=R` is the existing artifact-end path; `O1<R` selects requested-end completion and cannot be used with partial evidence. |
| Requested-end evidence | Opaque validator/cursor proof naming artifact `R`, requested `O1`, the last prefix ordinal applied through `O1`, the full artifact record count, and successful same-open validation of every unapplied suffix record plus coverage, summary, digest, final seal, and unchanged-file identity. | Exactly one C4 producer. It authorizes only the ordinary engine requested-end transition; it is neither artifact-end evidence nor a controlled stop. |
| Replay source state | Artifact cursor, expected next ordinal/group, simulated clock, pacer, terminal outcome, and bounded counters. | One sequential driver; no record queue, worker pool, copied canonical graph, rank state, or publication cell. |

The required engine extension is a closed set of typed replay admissions:
start, replay aggregate, group timer, artifact end, requested end, and failure. The artifact validator
is the only success-evidence producer. Types or unexported construction must
prevent an arbitrary caller from building complete-coverage/group/terminal proof;
source/package ownership inspection must show exactly one producer. A failure
fact may be callable by the runner because it can only suppress claims.

The engine stores only the minimum run evidence needed to validate the next
fact: artifact ID, complete/partial class, `[S,R)`, total/next ordinal, next
whole-second group, complete covered-through boundary, last completed group,
and terminal flag. Replay aggregate
consumption requires global ordinal continuity, exact artifact/binding/source,
`DeliveryTime` equal to the sampled simulated clock, and a delivery time no
earlier than `window_end`. A group timer requires exact next whole second and
the validator-certified last ordinal to equal the engine-consumed last
ordinal. For a complete final-bar group, the engine then records exact presence
or proven absence for every binding symbol's just-finished slot before applying
the timer candidate/evaluator. That coverage mutation remains inside the sole
engine and is artifact evidence only; it is not Component 6's live/historical
`no_print_through(T)` token or terminal-work ledger. This state fills Component
2's existing replay branch; it does not create another watermark.

**Construction guarantees:** one sequential source owns logical-clock writes;
only a fully validated handle can create success evidence; playback pace is not
representable in an engine fact; partial mode can never set complete commit
support; replay aggregates cannot carry live/historical positions; timers keep
engine-owned time/system sequence; and replay cannot mutate a live-mode engine.

**Runtime validation still required:** binding/artifact compatibility, ordinal
continuity, second-pass canonical bytes/digest, group membership and completion,
clock equality/nonregression including requested-end admission at exact `O1`,
aggregate/timer dispositions, terminal evidence, cancellation races, queue
closure, and engine suppression remain representable and fail the run when
observed.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C4-SCHED-01` | Starting at `S`, the source processes every whole-second logical group through the selected application end (`R` for ordinary artifact-end replay, `O1` for requested-end replay): apply cancelable wall pacing, advance the single simulated clock to the group time, sequentially admit and await every artifact record at that time in ordinal order, then admit and await exactly one typed group timer. Recordless seconds still receive the timer. It cannot advance to the next group until every disposition/publication in the current group completes. Pace changes no engine input or logical result. | `DTE-CLOCK-03`–`06`, `DTE-TIMER-01`, `DTE-REPLAY-02`, `DTE-REPLAY-03`, `LIFE-REPLAY-03`, `LIFE-T23`; accepted Component 2 injected clock/FIFO. |
| `C4-RUN-01` | After binding installation and full artifact validation, replay start moves the same engine from `initializing` to `replaying`. Typed replay aggregates use the existing canonical merge/evaluator/publication path. At each complete-final-bar group, the engine atomically classifies every binding symbol's finished slot as accepted presence or artifact-proved absence before applying the timer; only then may the existing replay commit predicates advance `T`. Partial synthetic evidence never installs complete absence or supports commit. After the validated seal and final `R` timer, replay end performs final evaluation/publication and transitions to `ended`. Output is replay-labeled and T/Q-unavailable, never production-live ready. | `PG-REPLAY-01`, `PG-REPLAY-02`, `ARCH-OWN-01`–`04`, `DTE-WINDOW-04`, `DTE-COMMIT-01`–`04`, `DTE-REPLAY-01`, `LIFE-T03`, `LIFE-REPLAY-01`–`03`, `LIFE-T24`; accepted Components 2/3 seams. |
| `C4-FAIL-01` | Schema/binding/coverage/order/digest/clock/source-position/unexpected-disposition/canonical/end contradiction produces one bounded failed run and the engine's replay-only suppression with `terminal_replay_failure`; no later ordinary fact restores it. Corrected input uses a new engine. Caller cancellation is distinct: before terminal-fact linkage it stops future source work, waits already-linked nonterminal work, admits controlled stop when possible, returns canceled/ended, and never claims completion; an already-linked terminal fact instead owns the sealed terminal result. | `DTE-REJECT-01`, `LIFE-MODEL-04`, `LIFE-SUPPRESS-01`–`03`, `LIFE-END-02`, `LIFE-END-03`, `LIFE-T25`, `LIFE-T28`, `LIFE-T29`. |
| `C4-PREFIX-END-01` | For a validated complete artifact `[S,R)`, an operator-requested exact `O1` with `S < O1 <= R` is checked before engine mutation. The source delivers every accepted record and exactly one timer per whole-second group through `O1`, never admits a later record or timer, then validates the entire unapplied suffix on the same-open second pass before admitting a distinct engine requested-end fact. That admission must be sampled by the engine at exact logical time `O1`. `O1=R` preserves the existing artifact-end path. Successful requested end publishes and ends through the sole engine while preserving configured `D`; invalid/mismatched/insufficient/ambiguous ends, suffix mutation, missing prefix evidence, wrong admission time, or requested-end contradiction cannot report success. | Owner-approved 2026-08-09 C4 correction; `PG-REPLAY-01`, `ARCH-OWN-01`–`04`, `DTE-CLOCK-04`–`06`, `DTE-TIMER-01`, `DTE-COMMIT-01`–`04`, `DTE-REPLAY-01`–`03`, `LIFE-REPLAY-01`–`03`, `LIFE-END-02`–`03`. |
| `C4-BOUNDED-CANCEL-01` | A bounded candidate-header probe reads at most the canonical header line under the artifact byte limit and returns only candidate schema/mode/date/interval/reference identities; it is never a validated handle, identity, coverage, or success fact. First-pass validation and terminal/suffix playback accept a context, check it between bounded line reads and immediately before returning success, and return no handle/end fact when cancellation wins before that success linearization point. A manually driven source exposes one idempotent `Cancel(ctx)`: before terminal-fact linkage it stops new source facts and owns the existing controlled-stop, source outcome/accounting, and engine close/drain path, waiting only within `ctx`; after terminal linkage, the sealed successful or failed terminal disposition wins. Existing noncontext entry points may remain compatibility wrappers over `context.Background()`. | Owner-approved 2026-08-09 C12 prerequisite; `ARCH-OWN-01`–`04`, `ARCH-FLOW-04`, `DTE-REPLAY-01`–`03`, `LIFE-SUPPRESS-01`–`03`, `LIFE-END-02`–`03`, `LIFE-T28`, `LIFE-T29`. Not accepted behavior until `C4-S6` implementation and acceptance. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Validated handle to engine start | Opaque evidence matches the installed binding, replay engine, schema, artifact ID, provenance/policy, interval, record count, order, and explicit coverage class. | Wrong/reused/mismatched/partial-as-complete evidence fails in `initializing`; no `replaying` or `T` claim. | A caller constructing `complete=true` without reading every coverage entry. |
| Artifact cursor to aggregate admission | Second-pass line remains canonical; artifact/binding IDs and next ordinal match; logical time equals current simulated clock; values pass existing engine validation. | Any mismatch admits failure/contains the run; no skipped invalid record can be counted delivered. | Skipping, duplicating, swapping, or editing a record after first-pass validation while the run still ends successfully. |
| Completed group to coverage/commit support | All records certified for the exact next whole second have received dispositions; actual consumed last ordinal equals group proof; complete-final-bar coverage accounts for every binding symbol as present or absent for the finished slot; timer time equals group time; and existing engine accounting/ambiguity gates pass. | Partial mode, unclassified symbol/slot, missing/extra record, wrong time/ordinal, engine rejection, or accounting/integrity failure leaves coverage/advancement unsupported and fails or suppresses as applicable. | Advancing `T` or counting an empty symbol after a timer without complete per-symbol slot evidence. |
| Artifact end to terminal success | Second-pass digest/seal, total ordinal, final `R` group/timer, engine drain, and final publication all complete exactly once. | Truncation, appended bytes, unread record, missing timer, failed disposition, or canceled context cannot produce complete. | Treating EOF or caller cancellation as a valid artifact-end fact. |
| Requested end to terminal success | The complete artifact's same-open second pass validates the exact prefix through `O1`, scans but never admits all later records/groups, reconciles full coverage/summary/digest/seal and unchanged-file identity, and the engine accepts the opaque requested-end fact against its last applied group/ordinal at sampled engine time `O1`. | Invalid range/precision/mode fails before start. Missing prefix evidence, suffix corruption, ordinal/group/clock contradiction, engine rejection, pre-link cancellation, or requested-end mismatch fails/suppresses and cannot retain complete. | Treating a prefix EOF, a previously validated but subsequently corrupted suffix, or a caller-supplied end time as successful completion. |
| Bounded manual cancellation | Candidate probe and complete validation return no trusted handle when their context wins. After manual source start, `Cancel(ctx)` is the sole shutdown entry and returns one reconciled complete/failed/canceled result only after the terminal-link ordering and source/engine accounting close within `ctx`. | Timeout returns a bounded non-success error and no fabricated terminal result; a repeated call cannot admit a second stop or change a sealed outcome. C12 never stops/closes the engine directly. | A canceled scan returning a handle, an unbounded suffix read or drain, a second cancellation owner, or cancellation relabeling an already-linked successful terminal fact. |
| Replay result to operator/test | Outcome, mode, artifact ID, logical boundary, counters, and terminal disposition agree; complete is possible only after engine `ended` via artifact-end or requested-end completion. | Failed/canceled is explicit and carries no complete/live/currentness claim beyond last engine publication's own supported state. | A suppressed or partially replayed run being reported as a deterministic successful replay. |

## 11. Failure and terminal behavior

The artifact receives a complete validation pass before replay start. Playback
uses the same open read-only file and validates canonical bytes and SHA-256
again while reading; path replacement cannot switch the file. The second pass
detects changes in bytes it actually rereads and final size/modification-time
changes. This proof assumes a trusted local filesystem: it does not prove an
immutable snapshot against an adversarial writer that changes already-buffered
or already-reread bytes while restoring size and modification time. If a
failure is discovered before `replaying`, the bound replay engine records
replay failure from `initializing`. If discovered after start, the next
typed failure transition enters `suppressed`, seals the run, and stops future
artifact delivery. Already linked FIFO nodes drain; no failed output is
upgraded to success.

For a compiled final-bar artifact, only `aggregate_inserted` is a successful
aggregate disposition. A partial synthetic trace may additionally expect the
ordinary `aggregate_revised` or `aggregate_exact_duplicate` paths implied by
its ordered values. Rejection, fencing, repeated-position inequality,
accounting/publication integrity failure, unexpected withdrawal, or any other
disposition fails the replay. Every group timer must complete as
`timer_applied`; start and end must complete with their exact replay lifecycle
dispositions.

Cancellation is observed before a group and during cancelable wall/admission
waits. If no group fact has linked, no part of that group is claimed complete.
After linked nonterminal work, the source stops adding market facts, waits those
completions, and uses the existing ordered controlled stop when the engine still
accepts it; that outcome remains canceled even if the last logical group
finished. Artifact-end or requested-end admission is the terminal
linearization point. Cancellation before that admission remains canceled; once
the terminal fact is admitted, the sealed engine owns it, drains it, and its
accepted success or failure disposition wins over later context cancellation.
Failure also takes precedence when the engine has already suppressed/sealed.

Successful completion requires the timer at every second from `S` through the
selected application end, a verified second-pass seal, exactly one matching
artifact-end or requested-end transition, engine drain, and terminal `ended`.
With nonzero existing evaluation delay, final committed `T` may be less than
that selected end; Component 4 does not silently change `D`. The deterministic
core proof uses an explicit tested delay and compares only like configuration.

## 12. Accounting and observability

The replay source has these exact identities at every observation:

```text
artifact_records
  = completed_record_dispositions
  + intentionally_unapplied_suffix_records
  + unread_records
planned_groups = completed_groups + active_group + remaining_groups
terminal_outcomes = completed_runs + failed_runs + canceled_runs = 1
```

`active_group` is zero or one. An admitted failure-causing record has a
completed disposition and is included in `completed_record_dispositions`; its
reason is an overlapping diagnostic dimension. Engine admission, transition,
aggregate, evaluator, and publication accounting remain the Component 2/3
identities and are not copied into source counters.

The bounded run reasons are: artifact validation, binding/coverage, schema/
canonical encoding, ordinal/group order, clock, aggregate admission/
disposition, timer admission/disposition, engine integrity/suppression,
artifact end/digest, controlled stop/drain, and canceled. Artifact path, symbol,
record values, wall timestamps, and errors are not metric labels. Exact
artifact ID and last ordinal/time may appear in the local run report.

`intentionally_unapplied_suffix_records` becomes nonzero only after the
same-open suffix and seal validate. It is neither canceled work nor a complete
engine disposition. `planned_groups` continues to mean engine groups through
the selected successful end; suffix validation has no engine group/timer
counter. Exactly one terminal run outcome remains required, and successful
results carry the distinct completion disposition `artifact_end` or
`requested_end`.

Progress is the next record disposition or group timer; wall pacing status is
operational only. A report distinguishes `replaying`, `suppressed`, `ended`
complete, and `ended` canceled, and never maps replay to backend-ready/live.

## 13. Simplicity and boundedness

- Mutable state is one artifact cursor, one replay clock, one narrow pacer, one
  driver counter set, and the minimum named replay evidence inside the existing
  engine. Canonical/evaluator/lifecycle/publication state stays where it is.
- The source is sequential. It retains at most one decoded record and one
  completion at a time; existing engine capacity/backpressure is the only
  mutation queue. There is no replay worker pool or record channel.
- Timer count is exactly `(selected_end-S)/1s + 1`, never more than 57,601 for
  the approved 16-hour session. Records are bounded by the validated artifact/plan. Ordinal,
  group, timer, and engine sequences use checked positive counters and fail
  before wraparound.
- Wall pacing uses cumulative logical elapsed time from a fixed wall start and
  checked rational arithmetic, avoiding per-step rounding drift. Being late
  removes a wait; it never skips/coalesces a group. `unpaced` performs no wall
  wait.
- Opaque typed facts are retained because false complete coverage/order at the
  engine boundary could advance `T`. A generic event bus, callback registry,
  second loop, writable snapshot observer, or public timestamp setter is not
  permitted.
- Replaying directly from REST was rejected because it would combine provider
  availability with deterministic evidence and eliminate the validated
  artifact boundary. A fake WebSocket was rejected by Phase 1. Preloading all
  records, wall-time-derived engine clocks, and a separate replay evaluator are
  unnecessary and unsafe.
- Checkpoint start at `T0`, T/Q artifacts, live chronology, recovery, readiness,
  shutdown policy, and public snapshot serialization remain with Components
  7, 9, 5/6, 8, and 10 respectively.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Recordless whole second, including `S`, and fully empty complete artifact | `DTE-WINDOW-04`, `DTE-REPLAY-03`, `LIFE-REPLAY-03` | Emit exact timer without fabricated aggregate; at `t>S`, complete-final-bar evidence marks every missing `[t-1s,t)` slot proven absent inside the engine, supporting honest empty population and `T`. Timer `S` closes no prior slot. | `P-C4-SCHED`; `P-C4-RUN` |
| Multiple symbols at one logical time | `DTE-REPLAY-02` | Deliver immutable ordinals in `(window_start,symbol)` order, await each, then timer. | `P-C4-SCHED` |
| Synthetic duplicate, correction, or older identity delivered later | `DTE-MERGE-01`, `DTE-MERGE-02`, `DTE-MERGE-05`; explicit Phase 1 partial scenario | Preserve artifact ordinal precedence/logical time and exercise the ordinary canonical path; never relabel as compiled live chronology. | `P-C4-RUN` |
| Partial synthetic coverage | `LIFE-REPLAY-01` | May enter replaying for bounded tests but never satisfy complete replay commit support or whole-population/no-print claim. | `P-C4-RUN`; `P-C4-FAIL` |
| First-pass-valid file altered during playback | Persisted-boundary invariant | Second-pass canonical/digest check suppresses; artifact end is unavailable. | `P-C4-FAIL` |
| Skipped/swapped/repeated ordinal or wrong artifact ID | `DTE-EVENT-04`; accepted Component 2 source validation | Fail at the exact record/group boundary; no later timer/end success. | `P-C4-FAIL` |
| Simulated clock regression or pace/wall lateness | `DTE-CLOCK-04`, `DTE-REPLAY-03`; Component 2 clock integrity | Regression suppresses. Wall lateness only shortens waiting and cannot alter clock/order. | `P-C4-SCHED`; `P-C4-FAIL` |
| Nonzero evaluation delay at `R` | `DTE-COMMIT-01`; Component 2 config | Preserve configured target calculation; do not fabricate final `T=R`. | `P-C4-RUN` |
| Cancellation before or during a group | `LIFE-END-02`, `LIFE-T29` | Stop new facts, drain linked facts, controlled-stop if possible, and return canceled without artifact-end claim. | `P-C4-FAIL` |
| Requested `O1<R` with valid/corrupt suffix | Owner-approved 2026-08-09 correction; persisted-source invariant | Apply only records/timers through `O1`; scan the suffix solely for integrity. Valid full source may end with `requested_end`; any suffix mutation fails before requested-end admission. | `P-C4-PREFIX-END` |
| Cancellation during candidate probe, first pass, manual group, or terminal/suffix scan | Consequential persisted/lifecycle boundary; C12 composition need | Check context at bounded scan/admission boundaries, return no premature handle/end, and route a started source through exactly one C4-owned bounded cancel result. | `P-C4-BOUNDED-CANCEL` in C4-S6 |
| Canonical/accounting/publication integrity failure | Accepted Components 2/3 containment | Suppress only this replay engine, return failed, and require a fresh run. | `P-C4-FAIL` |

Primary proof definitions and slice allocation are authoritative in
[deterministic-core delivery](deterministic-core-delivery.md#15-primary-proof-allocation).
