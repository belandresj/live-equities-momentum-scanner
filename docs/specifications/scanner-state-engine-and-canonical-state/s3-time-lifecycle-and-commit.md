# ScannerStateEngine and canonical state — S3 time, lifecycle, and commit

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Injected-clock and timer authority, remaining
currently realizable Phase 1 lifecycle edges/waits, and the single private
run-specific base commit gate that cannot yet advance `T`

**Controlling requirements:** `ENG-TIME-01`, `ENG-COMMIT-01`, `ENG-LIFE-01`;
`DTE-CLOCK-02`–`DTE-CLOCK-06`, `DTE-TIMER-01`, `DTE-COMMIT-01`–`DTE-COMMIT-04`,
`DTE-REPLAY-02`, `DTE-REPLAY-03`, complete lifecycle contract and
`LIFE-T01`–`LIFE-T29`

**Allocated slices:** `S3` only

**Document dependencies:** Parent; accepted S1; accepted S2

**Approval state:** Inherits the parent contract approval; not independently approved

**Implementation status:** `S3` implemented and owner-accepted 2026-08-05
after all three allocated primary proofs, all accepted S1 proof reruns, the
complete accepted S2 `ENG-AGG-01` matrix, package/race/repository verification,
correction of the independent review's post-suppression binding-install and
illegal-timer target-mutation findings, and successful final targeted
re-review. No S3 review or acceptance item remains open. This acceptance does
not authorize `S4`.

## S3 outcome

The same S1/S2 owner uses only an injected clock and explicit FIFO timer facts
for deterministic time-driven progress, owns the only lifecycle transition
function, and installs one private run-specific commit gate. Component 2 proves
that `T` cannot regress or advance through false evidence; because no approved
live or replay support producer exists yet, the base gate remains closed and
S3 makes no successful-advancement claim.

S3 does not add publication-time `generated_at`, provider acknowledgement,
hydration/recovery/replay artifact facts, production timing/capacity policy,
ranking/readiness, checkpoint, T/Q, or a generic test hook for any of them.

## Semantic time, timer, lifecycle, and `T` state

| Item | Normative meaning | Bound/owner |
| --- | --- | --- |
| Clock authority | Product input time comes only from the injected clock. Every actual read is serialized and checked against the previous actual read. | Immutable clock dependency; product packages do not read wall time directly. |
| Admission sample | After capacity exists and a copied node is ready, admission samples once and commits `admission_engine_time` with linkage/result. Blocked/nonadmitted calls preserve no earlier sample. | FIFO-stored samples compare only in FIFO linkage order. Consumption reads no clock. |
| Timer fact | System source, positive run-local `system_sequence`, binding, and the admission-sampled engine time; no caller timestamp or market payload. | Same immutable FIFO path. Component 8 owns live cadence; Component 4 owns replay clock advancement/scheduling. |
| Publication clock seam | S4 later samples `generated_at` after completed transition. That later actual read is serialized but metadata-only and cannot invalidate an older already-linked admission sample. | Owned/proved by S4; absent as an external publication path in S3. |
| Lifecycle | Exactly one approved Phase 1 state plus latest bounded transition record. Only the engine has transition authority. | No module transition method, writable reference, generic fact hook, or placeholder. `ended` terminal. |
| Committed state | Optional private nondecreasing `T` in `[S,E]`, candidate target, future supporting ingress fence, gap status, and explicit named predicates. | Exactly one per binding; no setter/callback/fixture production hook or alternate watermark. |

## `ENG-TIME-01` — injected time and explicit timer

Admission records the serialized clock sample only at successful linkage.
Actual admission reads and stored FIFO admission samples must be nonregressing
under their correct comparison orders. Timer callers supply only intent/system
position. A timer can drive deterministic evaluation during quiet market
seconds but creates no aggregate, trade, quote, mark, receipt, or coverage.

Clock regression gives the triggering input
`integrity_failure_clock_regression` before its fact mutation, retains the last
supported `T`, replaces current claims through the later S4 failure/publication
path, and enters `suppressed`. In a live run its disposition is
`restart_required`; in replay it is `terminal_replay_failure`. Both close
admission and drain/fence the already-linked FIFO. An ordinary later timer
cannot be admitted to restore or advance the terminated instance.

Replay uses the same path. Component 4 must advance simulated time one logical
delivery group at a time and wait until that group's admitted records/timer and
later publications complete before advancing beyond it. S3 does not prove that
scheduler or playback-speed invariance.

## `ENG-COMMIT-01` — one closed base gate

A timer calculates:

```text
target = clamp(floor_second(admission_engine_time - D), S, E)
```

`D` is required nonnegative construction configuration but has no Component 2
production value. One central owner function is the only path that may assign
`T`, and `T` is nondecreasing and bounded by `[S,E]`.

Common future predicates are valid binding, monotonic clock, no unresolved
global ambiguity, and coherent completed engine accounting. Live success also
requires the applicable accepted aggregate epoch, a consumed-through ingress
fence, no prior global transport gap, and named contributor predicates. Replay
success instead requires Component 4 validated artifact coverage/order and
completion of the logical delivery group; it never requires a live epoch.

Component 2 has no production fact capable of satisfying either complete
branch. Therefore target calculation is reachable but `T` remains absent. No
caller, aggregate receipt time, checkpoint `T0`, recovery end `R`, hydration
progress, T/Q boundary/coverage, queue emptiness, `generated_at`, wall speed,
fixture mutation, boolean `ready`, or runtime callback can select or advance
`T`. Later components may add named state only inside this gate. T/Q can never
gate aggregate `T` or ranking.

Component 4 owns the first successful replay proof; Components 5/6 own the
first successful live proof. Component 8 supplies production `D` and
currentness policy. At `E`, only later proved support can commit `E`.

## `ENG-LIFE-01` — sole staged lifecycle authority

The Phase 1 vocabulary/graph is exhaustive. S1 already owns binding-time
`LIFE-T01`, `LIFE-T02`, and `LIFE-T05`. S3 owns remaining currently realizable
timer, controlled-stop, sequence-exhaustion, clock-integrity, canonical-
integrity, explicit illegal/uninstalled-input, suppression, and terminal edges.
S4 completes publication-integrity observation. Later components add exact
closed facts and guards inside the same private transition function and receive
no transition API.

| State | Required progress/exit evidence and Component 2 boundary |
| --- | --- |
| `initializing` | Valid binding, fixed initialization integrity, stop, or bound time `>=E`; checkpoint/replay artifact inputs wait for Components 7/4. |
| `awaiting_session` | Timer reaches `S`, binding/clock failure, or stop; pre-session acknowledgement comes from Component 5. |
| `awaiting_aggregate_ack` | Current-epoch ack, finite establishment-policy result, session end, or stop; state/illegal containment exists now, facts/policy come from Components 5/8. |
| `hydrating` | Live/control/timer, generation progress/terminal, captured fence, epoch loss, integrity, session end/stop. Components 5/6/8 add facts/policy; terminal evidence must exit to `live`, `suppressed`, or `ended`. |
| `live` | Ordinary facts/timers, exact coverage loss, integrity, session end/stop. S3 owns timer/integrity authority; later components add evaluator/coverage/checkpoint/TQ facts. |
| `recovering` | New-epoch ack, current generation/live tail/timer/retry, reconciled fence, exhaustion, session end/stop. Later facts cannot leave an inactive wait. |
| `replaying` | Component 4 ordered aggregate/timer, artifact end, integrity, stop; corrected input requires a new engine. |
| `suppressed` | Explicit same-binding recovery, clean restart/reinitialization, terminal replay disposition, session end/stop. Ordinary timer cannot restore. |
| `ended` | No exit; admission fails terminally and no later fact mutates product state. |

Illegal admitted input receives `rejected_illegal_lifecycle`, never silent
ignore. Uninstalled later families have no constructible success path.
`suppressed` always has an explicit bounded disposition. Every reachable wait
names scheduled/external progress or terminal evidence.

Component 2's currently constructible suppression mapping is closed and
deterministic:

| Run/cause | Suppression disposition | Admission boundary |
| --- | --- | --- |
| Live canonical integrity | `clean_reinitialization_required` | Seal and drain/fence captured FIFO; discard this instance before a new engine is initialized. |
| Live clock, sequence, accounting, or publication integrity | `restart_required` | Seal and drain/fence captured FIFO. |
| Any replay integrity failure | `terminal_replay_failure` | Seal and drain/fence the failed run; corrected input uses a new engine. |

`same_binding_recovery_allowed` remains part of the Phase 1 closed vocabulary
for a later explicitly proved recoverable ingress/gap cause; Component 2 has no
constructible cause that maps to it and introduces no recovery producer.

## Trust, failure, and bounded observability

| Boundary/condition | Exact result | Dangerous false success prevented |
| --- | --- | --- |
| Injected clock | Nonregressing actual reads and FIFO admission samples; regression suppresses and cannot self-clear. | A regressing clock followed by a normal timer looking current (`DF-11`). |
| Timer | Closed system fact, owner-sampled time, legal lifecycle guard. | Caller timestamp or quiet timer manufacturing a bar/mark/`T`. |
| Commit gate | Private run-specific named predicates only; base live/replay branches unreachable in S3. | Caller/checkpoint/recovery/TQ/generated/receipt/wall-speed/fixture evidence advancing `T` (`DF-10`). |
| Lifecycle | Compiled fact, exact current-state guard, fixed transition reason. | Binding omitting its mandatory route or illegal/uninstalled input silently succeeding (`DF-12/17`). |
| Stop/end | Seal admission, drain captured nodes, terminally close future work through later registered mechanisms, and reach `ended`. | External I/O or an inactive placeholder preventing termination. |

The latest transition retains only previous/next state, fixed reason, binding,
applicable epoch/generation, engine time/sequence, and last `T`; there is no
history or arbitrary reason string. State/reason/source are overlapping
diagnostic dimensions, not primary populations.

## Evidenced edge cases

- A timer during a second with no aggregate advances evaluation only.
- A blocked/canceled call contributes no earlier clock sample.
- Equal time is legal; a lower actual/FIFO admission sample triggers integrity.
- An ordinary timer after regression cannot restore or advance.
- Target boundaries at/before `S`, inside session, and at/after `E` clamp exactly.
- Every listed nonauthority leaves `T` absent; live and replay predicates remain
  distinct and cannot be fixture-written.
- Binding-time lifecycle routes remain only in S1; S3 covers all remaining
  realizable timer/stop/integrity paths and rejects uninstalled later families.
- Any input after `ended` has a terminal admission result and no mutation.

No provider-specific future-time tolerance or clock quirk is introduced.

## Primary proofs

| Requirement | Primary proof | Observable result and limitation |
| --- | --- | --- |
| `ENG-TIME-01` | Injected-clock/timer integrity scenario covering `DF-11`, post-capacity sampling, blocked/canceled admission, equal/regressing samples, quiet timer, suppression, and later timer. | Only linked nodes retain samples; timer creates no market fact; regression has fixed integrity result and cannot clear. Does not prove S4 `generated_at`, OS accuracy, production cadence, or Component 4 scheduling. |
| `ENG-COMMIT-01` | Closed-base-gate/nonauthority scenario covering `DF-10`, exact `S/E` target math, all nonauthorities, live/replay separation, and source/API inspection for setters/hooks/second assignment paths. | Target is exact but S3 never advances `T`; private predicates cannot be fixture-written. This proves ownership/false-advance prevention, not successful advancement, production `D`, coverage/no-print, or ranking currentness. |
| `ENG-LIFE-01` | Remaining staged lifecycle-owner scenario covering remaining `DF-12` and `DF-17`: closed enum, sole authority, currently realizable timer/stop/sequence/clock/canonical edges, illegal/uninstalled input, waits, suppression, terminal `ended`. | Legal edges reach exact state/reason; illegal input rejects; timer cannot restore; ended is terminal. Binding routes are proved by S1, publication integrity by S4, later producer semantics by their owning components. |

Construction inspection must find no caller time/`T`/lifecycle setter, direct
product wall-clock read, generic fact/callback/test hook, second commit path, or
later placeholder. Race mode remains secondary only.

## S3 implementation assignment boundary

**Outcome:** the accepted owner uses admission-sampled time and explicit timers,
owns all currently realizable lifecycle edges, and has one closed central base
commit gate.

**Allowed package:** `internal/engine`, focused tests, and a standard-library
clock interface only.

**Approved v2 use:** scoped interval/structural test evidence only; v2 time,
watermark, lifecycle, maintenance, and owner implementations are rejected.

**Review artifact:** exact clock/target/lifecycle tables; three proof results;
source/ownership inspection; suppression/termination walk; limitations;
confirmation of absent later facts/hooks; and whether S4 remains valid.

**Explicitly deferred:** successful replay advancement to Component 4;
successful live advancement to Components 5/6; S4 publication/`generated_at`;
production `D/C/R`; provider/hydration/recovery/replay facts; ranking/readiness,
checkpoint, T/Q, API/UI.

**Independent review trigger:** clock authority, nonregression, sole lifecycle
transition ownership, and the unreachable base commit success path require a
narrow independent review before S4.

Stop for any caller-selected time/`T`/transition, reachable fixture-ready gate,
new fact family/later policy, direct wall-time dependency, lifecycle conflict,
or inability to prove every assignment to `T` enters the central function.
