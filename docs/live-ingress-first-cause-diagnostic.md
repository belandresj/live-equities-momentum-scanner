# Live ingress first-cause diagnostic and reproducer

**Status:** D1 and the offline D2 diagnostic correction completed on 2026-08-12.
A separately authorized, single checkpoint-off provider observation identified
the first owner/reason as `adapter_terminal/protocol/frame_capacity`; Section
13 records that evidence. D2 now makes a future capacity incident decisive and
provisionally raises only the bounded queue headroom. No behavioral throughput
correction was performed. A separately authorized D2 observation is recorded
in Section 15; it was run once and not retried. D3's proven hydration-conflict
population blocker now has a narrow deterministic correction, complete local
verification, and clean focused review/re-review.

**Parent and sole delivery ledger:**
[`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md).
This file is a routed subordinate specification. The parent records progress,
evidence, reopening, review, and acceptance state.

**Terminal condition:** Identify the first causal failure behind the observed
live `ingress_integrity` suppression with typed, deterministic evidence; record
the implicated boundary and the smallest correction implied by that evidence;
then stop. Do not implement the behavioral correction or run another provider
trial in this assignment.

## 1. Authority and controlling requirements

Authority descends from:

1. [`product/product-goals.md`](product/product-goals.md), especially
   `PG-OPS-02` and `PG-OBS-01` through `PG-OBS-03`;
2. [`architecture/system-overview.md`](architecture/system-overview.md),
   especially `ARCH-FLOW-01` through `ARCH-FLOW-04`;
3. [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md),
   especially `DTE-MODEL-02`, `DTE-MODEL-03`, `DTE-EVENT-02`,
   `DTE-CONTROL-01`, `DTE-RECOVERY-01` through `DTE-RECOVERY-05`,
   `DTE-COMMIT-02` through `DTE-COMMIT-04`, and `DTE-REJECT-01`;
4. [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md),
   especially `LIFE-HYDRATE-03`, `LIFE-HYDRATE-05` through
   `LIFE-HYDRATE-07`, `LIFE-RECOVER-01` through `LIFE-RECOVER-06`, and
   `LIFE-TQ-02` through `LIFE-TQ-03`;
5. [`v1-release-program.md`](v1-release-program.md), including its correction,
   containment, verification-cost, and evidence-reuse rules;
6. accepted Components 5, 6, and 8, especially
   [`specifications/massive-live-adapter/transport-commands-and-epochs.md`](specifications/massive-live-adapter/transport-commands-and-epochs.md),
   [`specifications/aggregate-rest-hydration-and-recovery.md`](specifications/aggregate-rest-hydration-and-recovery.md),
   and [`specifications/readiness-and-operations.md`](specifications/readiness-and-operations.md);
7. the active parent correction contract; and
8. the two 2026-08-12 live observations recorded below.

This assignment may improve bounded diagnostic transport and add a
deterministic production-composition reproducer. It may not change market
semantics, lifecycle meaning, queue capacity, retry policy, readiness,
hydration, ranking, T/Q behavior, or any accepted accounting identity.

## 2. Observed failure and invalidated claim

The operator-supplied pre-session run reached the session, completed hydration,
briefly published four to five ranked rows, repeatedly became noncurrent, and
finally exited with only:

```text
suppressed / ingress_integrity / same_binding_recovery_allowed
```

One separately authorized current-session rerun reproduced the same generic
terminal much earlier:

- current-date binding and `A.*` acknowledgement succeeded;
- fresh hydration planned 5,511 symbols;
- 5,176 completed with values and 334 completed empty;
- zero provider failures, cancellations, or fenced results were reported;
- exactly one hydration request remained open when suppression occurred;
- the live aggregate tail was connected and being consumed;
- the engine was still `hydrating` with its ingress fence pending;
- no ranking-driven T/Q subscriptions had begun; and
- the process exited on `ingress_integrity` before becoming ready.

A bounded sample before failure showed a 512-frame queue with high-water 24,
17,261 high-water bytes, 44,183 deliveries, 906 ms maximum processing delay,
and valid queue/accounting identities at that sample. No checkpoint was
eligible because the run never became live.

This second observation rules out T/Q pressure as the initiating cause in that
run and invalidates the operational claim that a generic
`ingress_integrity` terminal is sufficient to diagnose a failed live launch.
It does **not** prove that queue capacity, receipt regression, frame oversize,
status ambiguity, decoder ambiguity, runtime accounting, or a race between
snapshots caused the failure.

## 3. Outcome, ownership boundary, and non-scope

### Required outcome

At the end of this assignment, a technically precise incident record must name:

1. the first failing owner and typed reason;
2. the exact invariant that failed;
3. the smallest production path that deterministically reproduces it;
4. the last coherent predecessor state and first invalid transition;
5. whether the failure is a real integrity loss or a false positive caused by
   non-atomic observation/accounting;
6. the accepted contract claim invalidated by the evidence;
7. the smallest correction boundary implied by the finding; and
8. what remains unproven without another provider observation.

### Ownership

The Massive live adapter remains the sole owner of raw-frame admission,
classification, epoch termination, and its typed first terminal cause. The
`ScannerStateEngine` remains the sole lifecycle, suppression, canonical state,
watermark, and readiness owner. The operations runtime may carry and expose a
sealed diagnostic fact; it may not reinterpret or replace either owner's
decision.

No second mutable market-state owner, readiness evaluator, lifecycle state,
queue, event path, or diagnostic history is permitted. Any first-cause latch is
fixed-size, immutable after its first successful set, and diagnostic only.

### Explicit non-scope

Do not:

- fix or tune the identified behavior;
- enlarge the 512-frame or byte capacity;
- change hydration worker count, retry budgets, deadlines, evaluation cadence,
  pressure thresholds, or checkpoint behavior;
- weaken an accounting identity, validation rule, ambiguity classification,
  suppression, or readiness gate;
- add per-symbol labels, raw-frame capture, payload logging, provider prose,
  URLs, credentials, authorization headers, or unbounded event history;
- add a database, service, telemetry framework, alternate scanner path, or
  generalized fault-injection framework;
- inspect unrecorded Version 2 scope or any older Version 1 checkout;
- stage, commit, push, rebase, or open a pull request; or
- access credentials or issue another live provider request.

## 4. Exact diagnostic contract

The runtime's terminal incident must preserve the earliest causal fact and
expose one fixed-cardinality, redacted record. Fields may be represented by an
existing structure or one purpose-built immutable value, but their meanings
are fixed:

| Field | Required meaning |
| --- | --- |
| `owner` | Closed enum identifying adapter terminal, runtime accounting guard, or engine integrity transition. |
| `source` | Existing bounded terminal source or exact internal guard name; never free-form provider text. |
| `reason` | Existing bounded typed reason, including `frame_capacity`, `frame_oversize`, `receipt_regression`, `status_ambiguous`, `ingress_ambiguity`, or the exact runtime accounting identity that failed. |
| `binding` / `epoch` | Redacted binding identity and exact positive connection epoch associated with the cause. |
| `position` | Last knowable `(epoch, frame_sequence, array_index)` or explicit not-applicable marker. |
| `lifecycle` | Engine lifecycle, lifecycle reason, and suppression class after consuming the cause. |
| `hydration` | Purpose, generation, fence state, planned/open/terminal counts, and row accounting at one coherent capture boundary. |
| `queue` | Full absolute queue accounting, occupancy, capacities, and high-water values from one snapshot. |
| `adapter` | Full absolute attempt/classification/terminal accounting needed to reconcile the adapter identities. |
| `engine` | Admission, transition, aggregate, connection, and hydration counters needed to reconcile the affected identities. |
| `identity_results` | Named boolean/result for every evaluated accounting identity, with exact left/right totals on failure. |
| `captured_at` | One receipt/diagnostic instant used only for incident ordering, never as market time. |

The record must be available to the existing private operator terminal path
before process exit. It may also be exposed through the existing bounded
operations status or API mapping if that is the smallest way to preserve the
fact, but no new endpoint or public compatibility promise is required.

The first cause must survive concurrent worker cleanup, controlled close,
terminal marker disposition, runtime cancellation, final metrics collection,
and engine suppression publication. A later `controlled_stop`, context
cancellation, join error, or generic `ingress_integrity` projection cannot
overwrite or erase it.

If the runtime accounting guard is the first cause, it must report the exact
failed identity using one coherent observation. It may not compare counters
sampled before and after concurrent delivery and call the resulting mixed-time
view an integrity failure.

## 5. Production-path deterministic reproducer

Add the smallest deterministic, credential-free reproducer in
`internal/operations` that composes:

- the production `massive.LiveAdapter` and bounded live queue;
- the ordinary fake WebSocket transport and strict handshake;
- `operations.Runtime`, its delivery loop, timers, terminal handling, and
  accounting guard;
- the real engine admission, lifecycle, suppression, and observation paths;
- two-worker fresh hydration using the existing fake REST infrastructure; and
- the existing private operator/status rendering boundary when needed to prove
  the record is not lost.

The reproducer must not inject `ingress_integrity` directly into the engine or
assert only on an isolated queue helper. Each fault must enter at the earliest
production seam that can create it and travel through the same terminal,
cleanup, engine, runtime, and operator path used by `RunLive`.

Run these deterministic cases with fresh state and exact accounting oracles:

| Case | Injection | Required distinguishing result |
| --- | --- | --- |
| Healthy hydration/live-tail control | Sustained live frames while two fake REST workers advance to a controlled boundary. | No terminal or suppression; all sampled identities reconcile. |
| Frame capacity | Fill the bounded queue while preserving the real reader/admission path. | Earliest adapter terminal is `frame_capacity`; predecessor frames drain or fence exactly. |
| Frame oversize | Deliver one frame over the accepted byte ceiling through the real WebSocket connector, and separately exercise the queue-owned rejection invariant. | The production connector's equal read limit rejects the frame before queue admission as the pre-existing `reader/read_failed` recovery path; queue `frame_oversize` remains unreachable on that path and rejected bytes never appear admitted. This discovered contradiction revises the earlier expected result rather than changing behavior to manufacture a match. |
| Receipt regression | Inject a regressing receipt from the adapter's injected clock. | Earliest adapter terminal is `receipt_regression`; no later cleanup cause replaces it. |
| Status ambiguity | Deliver an out-of-phase or wrong-cardinality status through the strict handshake/control decoder. | Earliest adapter terminal is `status_ambiguous`; no acknowledgement or coverage is fabricated. |
| Ingress ambiguity | Put an unclassifiable element before otherwise valid later elements in a mixed frame. | Earliest adapter terminal is `ingress_ambiguity`; the causal remainder is fenced and not silently applied. |
| Runtime accounting guard | At a test seam, create the smallest representable mismatched accounting snapshot, then repeat under ordinary concurrent delivery without a mismatch. | A real coherent mismatch names the exact failed identity; mixed-time healthy counters never cause suppression. |

Fault construction must be bounded and local to tests. Prefer existing fake
transport, clock, queue, and runtime seams. A narrowly typed test hook is
permitted only when no existing seam can drive the production branch; it must
not exist in normal production configuration or bypass the owner whose
invariant is under test.

After individual cases pass, run one combined long-hydration/live-tail trace
using the same two-worker composition and current production limits. Select
the smallest deterministic schedule that reproduces the same owner/reason and
last-coherent/first-invalid transition as the live incident. Do not call a
candidate the cause merely because it ends in the same generic engine
suppression.

## 6. Causal decision rule

The issue is considered found only if all of the following hold:

1. the typed source/reason exists before cleanup begins;
2. the reproducer reaches that source/reason through production composition,
   not direct suppression injection;
3. all unrelated accounting identities reconcile at the immediately preceding
   coherent boundary;
4. one exact invariant first becomes false, ambiguous, or capacity-exhausted;
5. the same mechanism is consistent with every retained live fact, including
   the hydrating lifecycle, no T/Q subscriptions, low sampled queue high-water,
   one open hydration request, and absence of provider hydration failure; and
6. a healthy counterexample differing only in the causal condition does not
   fail.

If instrumentation proves that the existing live observation discarded the
only distinguishing value, do not guess. Complete the cause-preservation proof,
record the finite candidate set and why deterministic evidence does not choose
among it, and stop before another provider run. The next provider observation
would require a separate explicit authorization and must use the now-proven
diagnostic.

## 7. Consequential trust boundaries and dangerous counterexamples

| Boundary | False-success or false-failure danger | Required containment/proof |
| --- | --- | --- |
| Adapter first cause to runtime | Cleanup or context cancellation replaces the transport cause. | First write wins; deterministic concurrent cleanup preserves exact source/reason. |
| Concurrent metrics snapshot | Queue, adapter, and engine counters from different instants appear inconsistent. | One coherent capture or an identity explicitly safe across independent snapshots; healthy concurrency never suppresses. |
| Runtime to engine | Runtime invents a second lifecycle interpretation. | Runtime carries an existing typed fact; engine alone transitions lifecycle/suppression. |
| Reproducer seam | Test injects the expected terminal after bypassing the failing path. | Fault originates before the owning production check and follows ordinary delivery/terminal handling. |
| Operator output | Generic exit text drops the only causal discriminator. | Bounded terminal record is rendered before exit and contains no secret/raw payload. |
| Candidate matching | Any path ending in `ingress_integrity` is declared equivalent. | Match typed owner/reason, first-invalid invariant, predecessor state, and healthy control. |

## 8. Slice, allowed files, and primary proofs

There is exactly one active slice:

### `D1` — preserve first cause, reproduce, identify, and stop

Allowed production ownership boundaries, only when required:

```text
internal/massive      immutable typed terminal/diagnostic propagation
internal/operations   coherent capture, runtime propagation, production-path reproducer
cmd/scanner           bounded private operator rendering of the existing fact
internal/snapshotapi  only if the existing mapping is the smallest preservation path
docs                  parent ledger and final diagnostic record
```

Tests may extend corresponding package test files. Do not edit engine market
semantics, ranking/features, hydration policy, T/Q policy, replay, checkpoint
storage, UI, public schema, scripts, or provider configuration.

Primary proofs:

| Proof | Claim | Dangerous counterexample | Limitation |
| --- | --- | --- | --- |
| `P-D1-FIRST-CAUSE` | Every bounded adapter/runtime integrity source reaches engine suppression and operator exit with its exact earliest source/reason intact. | Concurrent cleanup replaces it with cancellation, controlled stop, or generic integrity. | Proves propagation and redaction, not which cause occurred live. |
| `P-D1-ACCOUNTING` | Every identity is evaluated from coherent counters; a real mismatch identifies exact operands while healthy concurrent delivery does not false-suppress. | Independently sampled counters straddle one delivery and appear unequal. | Proves the guard, not provider timing. |
| `P-D1-REPRODUCER` | The real two-worker hydration plus live-tail composition reproduces one exact causal mechanism, with a one-condition healthy control. | Direct engine suppression or isolated queue testing produces the expected label without the live path. | Offline transport scheduling cannot prove provider behavior by itself. |
| `P-D1-INCIDENT` | The final incident record distinguishes one cause consistent with all live evidence, or explicitly proves why the lost live subtype prevents distinction without another observation. | Generic `ingress_integrity` similarity is treated as root-cause evidence. | No new provider confirmation is authorized. |

## 9. Verification order and bounds

Run the narrowest proof after each diagnostic edit. The ladder is:

1. affected `internal/massive`, `internal/operations`, and `cmd/scanner`
   first-cause/accounting/operator tests, each command `-timeout 90s`;
2. the bounded composition reproducer, `-timeout 2m`;
3. `go test -short -timeout 2m ./...`;
4. affected race packages only, one command bounded by five minutes;
5. `go vet ./...`; and
6. `git diff --check` plus source inspection for credentials, raw payloads,
   unbounded labels/history, alternate state ownership, and behavioral drift.

Long, live, capacity, race, and diagnostic tests must skip under
`testing.Short()`. No local command in this assignment may exceed five minutes.
Do not rerun an unchanged failing command; reduce it to the first distinguishing
case. Existing expensive cached or live evidence is reused and not repeated.

A focused read-only `gpt-5.6-sol` medium review is required after the issue is
identified because the diagnostic crosses concurrent adapter/runtime terminal
and accounting boundaries. Review must check first-cause linearization,
coherent counter capture, redaction/cardinality, production-path fidelity, and
absence of a behavioral fix. The reviewer does not edit or expand scope.

## 10. Required final record and hard stop

Update the parent ledger and append a compact incident result to this file:

```text
first owner/reason:
failed invariant and exact operands:
last coherent state:
first invalid transition:
deterministic reproducer and healthy control:
live-evidence consistency:
invalidated accepted claim:
smallest proposed correction:
remaining uncertainty:
verification/review:
```

Then stop the task. Do not implement the proposed correction, do not tune a
limit, do not access credentials, and do not perform another live run. The
behavioral correction must begin as a separately recorded next slice after the
owner can inspect the diagnosis.

If no exact cause can be selected without another live observation, the final
record must say that plainly. A proven diagnostic plus an unresolved finite
candidate set is an honest diagnostic result; an inferred root cause without a
matching typed fact is not.

## 11. Drift checklist

- [x] One canonical engine/lifecycle/watermark/evaluator remains.
- [x] Adapter first-cause ownership remains; runtime only carries evidence.
- [x] No readiness, hydration, ranking, T/Q, queue, timeout, or retry behavior changed.
- [x] No accepted validation or suppression rule was weakened.
- [x] No raw payload, provider prose, URL, credential, symbol label, or unbounded history is retained.
- [x] Reproducer faults enter before the owning production check.
- [x] Healthy concurrent accounting cannot false-suppress.
- [x] The incident conclusion matches typed evidence, not the generic suppression label.
- [x] The parent remains the sole delivery ledger.
- [x] Work stopped before the behavioral fix or another provider request.

## 12. D1 incident result — 2026-08-12

```text
first owner/reason:
  The retained live evidence cannot identify one exact owner/reason because the
  runtime/operator boundary discarded the adapter terminal subtype and any
  runtime-guard identity before exit. The finite live-consistent set is:
  adapter_terminal/protocol/frame_capacity,
  adapter_terminal/protocol/status_ambiguous,
  adapter_terminal/protocol/ingress_ambiguity, or
  runtime_accounting_guard/sync_tq_pressure_accounting_guard/<exact failed local identity>.
  Receipt regression is excluded: the production composition reaches
  clock_regression/restart_required rather than the observed
  ingress_integrity/same_binding_recovery_allowed.

failed invariant and exact operands:
  The lost live subtype prevents selecting one invariant or its operands.
  Deterministic cases separately prove: the raw queue cannot admit a read frame
  after slot/byte capacity is exhausted; a real-connector frame over
  MaxFrameBytes is stopped by the WebSocket read limit as reader/read_failed
  before queue admission (so the queue's frame_oversize branch is not a live-
  path candidate); a success status without the one exact pending phase/cardinality is
  ambiguous; an unclassifiable array element makes the causal remainder
  ambiguous and fenced; and a coherent runtime guard mismatch reports its exact
  named identity and left/right operands. The guard proof used
  queue.frames_read left=2, right=1; its healthy control used left=1, right=1
  and did not suppress. Every unrelated queue, adapter, engine admission,
  transition, aggregate, connection, hydration-work, and hydration-row identity
  reconciled in the adapter candidate records.

last coherent state:
  The last retained live boundary remains the supplied sample: hydrating under
  the acknowledged aggregate epoch, 5,511 planned, 5,176 value, 334 empty,
  zero failed/canceled/fenced, one open request, ingress fence pending, live tail
  consumed, no ranking-driven T/Q subscription, queue high-water 24/512 and
  17,261 bytes, and all sampled identities valid. The new diagnostic cannot
  retroactively add the missing terminal subtype to that sample.

first invalid transition:
  Unknown in the retained live run. Offline production composition proves the
  exact first invalid transition for each remaining candidate before cleanup:
  queue admission capacity exhaustion; strict status phase/cardinality
  ambiguity; first unclassifiable mixed-frame element;
  or one named coherent runtime identity changing from equal to unequal. The
  adapter-owned terminal is first-write-wins and later controlled close,
  cancellation, join results, terminal marker disposition, generic engine
  suppression, and final sampling cannot replace it.

deterministic reproducer and healthy control:
  TestLiveIngressFirstCauseProductionReproducer uses Runtime.RunLive, the
  production Massive LiveAdapter and bounded queue, strict local WebSocket
  handshake, the real engine, two fake REST workers, concurrent live-tail
  delivery, terminal cleanup/join, and sealed operator capture. Every case uses
  fresh state and the production 512-frame/64-MiB/8-MiB limits. Capacity uses a
  nil-in-production diagnostic hook to hold only the consumer: the control
  admits exactly 512 frames, while the otherwise identical case sends one
  additional frame and records queue high-water 512 plus one capacity reject.
  Oversize uses exactly 8 MiB versus 8 MiB+1; receipt, status, and mixed-frame
  pairs likewise differ only in their causal condition. All traces overlap a
  deliberately open fourth REST request with two-worker hydration and live-tail
  delivery. The sustained-tail control reaches live/ready and exercises the
  accounting guard 32 times during real concurrent delivery without
  suppression. A separate latch race proves 32 concurrent cleanup attempts
  cannot overwrite the first fact.

live-evidence consistency:
  Capacity remains possible despite the earlier low high-water because that
  sample preceded the terminal. Oversize is excluded: the unchanged production
  connector rejects an over-limit WebSocket message as non-ingress
  reader/read_failed, not the observed ingress_integrity suppression. Status ambiguity remains possible after A.* ack
  even with no T/Q subscription because any later uncorrelated provider status
  is fail-closed. Ingress ambiguity remains possible in any later mixed frame.
  A real runtime local-identity mismatch also projects to the same generic
  suppression. All four are consistent with hydrating, one open REST request,
  active live tail, and no T/Q pressure. Deterministic similarity cannot select
  among them.

invalidated accepted claim:
  The C5 adapter already owned and latched the exact first terminal source and
  reason, but the accepted C5/C8 integration and PG-OBS-03 operator path did not
  carry that fact through DeliverToEngine, Runtime cleanup, sealed capture, and
  final operator output. Therefore the prior operational claim that terminal
  diagnostics were sufficient for a failed live launch is invalidated. No
  market-state, readiness, queue, or recovery semantics are invalidated.

smallest proposed correction:
  The completed diagnostic-only correction carries the existing immutable
  adapter terminal result alongside its engine disposition, seals one
  first-write-wins fixed-cardinality redacted incident in Runtime, evaluates
  runtime identities from their owner-coherent local snapshots with exact
  operands, and renders that record before exit. The eventual behavioral
  correction cannot be chosen until one of the four typed causes is observed;
  it must be scoped to that owning invariant and is not implemented here.

remaining uncertainty:
  Which of the four live-consistent causes occurred on 2026-08-12 remains
  unproved. Offline scheduling does not prove provider chronology, terminal
  payload size/content, or whether the sampled low queue later saturated. The
  next provider observation would distinguish the candidates with the now-
  preserved record, but it requires separate explicit authorization and was
  neither requested nor run.

verification/review:
  Focused Massive first-cause/control/fence tests, operations propagation,
  accounting, latch, reproducer, and scanner operator tests passed under 90s
  or 2m bounds. go test -short -timeout 2m ./... passed. The affected Massive,
  operations, and scanner packages passed -race -short under 5m. go vet ./...
  and git diff --check passed. One mistakenly broad non-short Massive package
  command ran the unrelated C7 restart capacity proof and failed its host-speed
  comparison (checkpoint median 4.002s versus fresh 1.169s); it was not rerun,
  does not exercise D1, and the subsequent narrow Massive D1 boundary passed.
  The mandated focused read-only review first rejected an oversize behavioral
  drift, post-cleanup/sampling-derived incident counters, non-production limits
  and unmatched controls, and a synthetic-only healthy guard proof. The drift
  was reverted; cause-time queue/adapter snapshots and owner-maintained queue
  high-water now pair with the immediately pre/post engine publications; all
  cases use current limits and matched controls; and the guard runs during real
  delivery. The focused re-review result is recorded in the parent.
```

Drift audit: one engine/lifecycle/watermark/evaluator remains; the adapter
still owns its terminal cause and Runtime only retains evidence; no readiness,
hydration, ranking, T/Q, queue, timeout, worker, retry, or suppression policy
changed; no payload, symbol, provider prose, URL, credential, or unbounded
history is retained; faults originate before their production owner checks;
and D1 stops here without a behavioral fix or provider request.

## 13. Separately authorized live selection — 2026-08-12

The owner subsequently authorized one current-date provider run to select the
typed candidate. The run used the existing macOS Keychain credential only in
the scanner child environment, checkpoint mode `off`, two hydration workers,
the fixed 512-frame/8-MiB-frame/64-MiB-queue limits, and a 15-minute maximum.
It ran once from 09:35:17 to 09:42:52 PDT and was not retried.

```text
first owner/reason:
  adapter_terminal/protocol/frame_capacity

failed invariant and exact operands:
  The adapter's raw-frame admission capacity conjunction first became false at
  epoch 1 after accepted frame 42,220: FramesQueued < 512 AND incoming frame
  bytes <= (64 MiB - QueuedBytes). The typed reason proves capacity rejection,
  but the private terminal renderer printed neither the retained cause-time
  FramesQueued/QueuedBytes operands nor which conjunct failed. Therefore slot
  exhaustion versus remaining-byte exhaustion cannot be selected from the
  retained output without guessing. No retry is authorized or needed to name
  the owning boundary.

last coherent state:
  Fresh hydration completed all 5,511 requests: 5,395 value, 116 successful
  empty, zero failed, zero canceled, zero fenced, and zero open. The aggregate
  epoch remained connected/acknowledged, the final ingress fence was pending,
  and the scanner briefly entered live/watermark_stale before suppression.

first invalid transition:
  The live reader attempted to admit the next raw frame after position
  (epoch=1, frame_sequence=42,220); the bounded queue rejected it as
  frame_capacity. The adapter then emitted IngressIntegrityFailure and the
  engine entered suppressed/ingress_integrity/same_binding_recovery_allowed.

invalidated claim:
  The previous unresolved four-candidate result is superseded: status
  ambiguity, ingress ambiguity, and runtime accounting were not the first
  cause in this run. The additional claim that the private terminal line made
  every retained capacity operand operator-visible is also invalidated; the
  values existed in the incident record but were not rendered before exit.

smallest proposed correction:
  Behavioral correction is deliberately not chosen here. It belongs to the
  adapter/consumer throughput and bounded-queue admission boundary, after the
  retained cause-time operands distinguish frame-slot saturation from byte-
  budget saturation. Do not tune queue size, workers, timeouts, retries,
  readiness, hydration, ranking, or T/Q behavior as a substitute for that
  evidence.

remaining uncertainty:
  Which capacity conjunct failed and which producer/consumer timing created
  the backlog remain unresolved because the terminal renderer omitted the
  cause-time queue operands. Provider chronology and raw payloads were not
  retained. The exact first owner/reason and lifecycle consequence are proven.

verification/record:
  The bounded operator log is
  var/run-private-scanner/d1-live-first-cause-2026-08-12.log (240 lines,
  13,540 bytes). It contains no credential or raw provider payload. The process
  exited after the first typed terminal; port 8080 was no longer listening.
```

## 14. D2 decisive capacity diagnostic and provisional headroom

**Status:** completed offline correction slice authorized by the owner after
the Section 13 observation proved that D1's private terminal rendering was not
decisive. No provider request was made.

The observed `frame_capacity` classification combined two different failed
invariants and printed neither one's operands. D2 must make the next run
decisive without raw-payload retention:

- classify slot exhaustion as `frame_slot_capacity` and byte-budget exhaustion
  as `frame_byte_capacity`, preserving the existing ingress-integrity lifecycle
  consequence;
- retain and render cause-time queued/classifying frames, queued bytes,
  incoming-frame bytes, both configured limits, high-water, last accepted
  causal position, and any engine delivery active at the rejection;
- retain a fixed 60-sample, one-second ring containing absolute read/admit/
  disposition/fence/rejection counts, queue frames/bytes/oldest age, delivery
  count/delay, lifecycle, and hydration progress so rates are derived from
  adjacent absolute samples rather than another mutable counter owner;
- atomically persist the bounded, redacted incident under `var/diagnostics`
  before scanner exit, using create-without-overwrite and no credential, URL,
  provider prose, symbol, or raw payload; and
- prove terminal rendering and persistence from deterministic production-path
  capacity cases before another provider request.

The owner also authorizes provisional 2x burst headroom for the next run:
1,024 frame slots and 128 MiB total queued bytes, with the per-frame ceiling
unchanged at 8 MiB. This is a revisable delivery limit, not evidence that the
larger queue fixes a sustained producer/consumer deficit. Worst-case retained
raw-frame bytes remain bounded at 128 MiB. A later measured healthy run may
reduce the limits; another saturation must be corrected at the blocking
consumer/engine operation rather than hidden by repeated queue growth.

No worker, timeout, retry, readiness, hydration, ranking, checkpoint, or T/Q
behavior changes in D2. No provider request is authorized by this section.

### D2 result

The first failing owner established by the retained live evidence remains the
Massive raw-frame admission boundary. The old record cannot be retroactively
split: its exact typed reason remains `protocol/frame_capacity`, after accepted
frame 42,220, and slot exhaustion versus byte-budget exhaustion remains
unknowable for that historical run. D2 does not relabel that evidence.

For the next incident, the first invalid transition is now represented as one
of two exact invariants:

```text
frame_slot_capacity: FramesQueued >= CapacityFrames
frame_byte_capacity: IncomingFrameBytes > CapacityBytes - QueuedBytes
```

The rejection record carries the incoming-frame size, queued/classifying
frames, queued bytes, both capacities, oldest age, owner-maintained high-water,
last accepted position, and the engine delivery kind/start/age active at the
same attempt-owner linearization point. Queue operands are copied while the
queue owner is locked. The attempt reserves the immutable terminal before
releasing its ownership lock; the adapter's internally coherent accounting is
then copied before cleanup starts. This is deliberately owner-coherent rather
than a fictitious globally atomic snapshot.

The last 60 scheduled one-second observations contain only absolute counters;
terminal and accounting-guard paths cannot insert an off-cadence sample or
evict a scheduled tick. Operator rates therefore use two adjacent scheduled
samples. The exact terminal fields remain separate from that history. The
operator line includes active delivery kind, RFC3339-nanosecond start, and age.
Before exit, the scanner writes and syncs an owner-only temporary JSON file,
atomically publishes it by no-overwrite hard link, removes the temporary name,
and syncs the directory. Readers can see either no final record or one complete
record, never a partially encoded final path.

The deterministic real-adapter/runtime/engine/two-worker reproducer selects
`adapter_terminal/protocol/frame_slot_capacity` when the 1,024-frame queue is
full: one capacity rejection, 1,024 queued/high-water frames, prior engine
state still hydrating with one REST request open, then the existing
`suppressed/ingress_integrity/same_binding_recovery_allowed` transition. Its
otherwise matched capacity control reaches live/ready with valid accounting.
Byte capacity, production-reader oversize behavior, receipt regression, status
ambiguity, ingress ambiguity, and the runtime accounting guard retain their
matched dangerous counterexamples. Healthy `RunLive` delivery exercises the
accounting guard repeatedly without false suppression.

The invalidated operational claim is narrower than D1's: retaining a generic
capacity reason and cause-time structs was not enough when the operator output
omitted the failed conjunct and operands. The smallest diagnostic correction
is the split reason plus exact operands/active delivery/history/persistence
described above. The only provisional resource correction is 1,024 slots and
128 MiB total bytes, retaining the 8 MiB frame limit. This may absorb a finite
hydration burst, but it is not evidence of adequate sustained consumer
throughput and must not justify another queue increase if measured backlog
again reaches the limit.

Remaining uncertainty is explicit: no offline proof can identify which
conjunct failed in the already completed provider run, reproduce provider
chronology, or prove the provisional headroom sufficient. A future separately
authorized provider run can now identify slot versus byte exhaustion and show
the active blocking engine delivery without retaining raw payloads. No such run
was requested or executed in D2.

Verification passed the complete matched production-composition reproducer and
runtime-guard/history/latch proofs under a 90-second bound; focused Massive
atomic-cause tests and scanner rendering/persistence tests; `go test -short
-timeout 2m ./...`; `go test -race -short -timeout 5m ./internal/massive
./internal/operations ./cmd/scanner -count=1`; `go vet ./...`; and `git diff
--check`. The first full-short correction run exposed a lock-order deadlock in
the initial atomic-capture implementation. Cause reservation was separated
from adapter accounting capture so owners are never nested during the
diagnostic snapshot; the focused test was repeated 10 times, and the complete
short and focused race suites then passed. The first D2 review also found
non-atomic active-delivery attribution, direct-to-final persistence, terminal
samples polluting the one-second ring, a missing rendered start time, and a
fixed-wall-time test error. All were corrected and sent to the same reviewer
for focused re-review. That `gpt-5.6-sol` medium re-review returned **CLEAN**
with no P1/P2 finding. It independently confirmed the two-phase first-cause
reservation has no lock inversion or cleanup race, the successful-admission
path does not manufacture a rejection snapshot, and the corrected byte-
capacity reader proof passed 50/50. That proof passed 100/100 locally; narrow
persistence/history proofs passed repeatedly. The reviewer relied on the
recorded complete short/race/vet results rather than repeating those whole
suites.

## 15. Separately authorized D2 live observation — 2026-08-12

The owner authorized one live run after D2 acceptance. It ran once from
13:24:10 through 13:39:16 America/New_York and was not retried. The run used
the current dirty worktree on branch `codex/live-scanner-recovery-narrow-fix`
at base commit `2faba3de7271734a5690dffee4983c46f7e25e65`, trading date
2026-08-12, the existing Keychain credential only in the scanner child,
checkpoint mode off, two hydration workers, 1,024 frame slots, 128 MiB total
queued bytes, an unchanged 8 MiB per-frame ceiling, and a 15-minute maximum.

```text
result classification:
  correction_required

capacity result:
  No capacity terminal or ingress-integrity suppression occurred. The queue
  reached 813/1,024 frames (79.4%) and 424,594/134,217,728 bytes (0.32%). The
  old 512-slot limit would have been crossed, while the byte budget was never
  remotely approached. This directly selects slot pressure in the new run and
  makes byte pressure implausible under comparable traffic; it cannot
  retroactively recover the missing operands from the earlier run. The
  provisional slot increase absorbed this observed burst, but one run does not
  prove sustained throughput or justify reducing the limit yet.

hydration and fence:
  Fresh hydration reconciled all 5,511 requests: 5,409 value, 102 successful
  empty, zero failed, zero canceled, zero fenced. It consumed 4,147,377 rows:
  4,146,793 inserted and 584 conflict/withdrawal, with zero rejected,
  integrity, or fenced rows. The aggregate connection stayed active and
  acknowledged at epoch 1. The ingress fence reconciled through frame 106,689
  and supported through 13:38:51 EDT.

first failing owner/reason:
  No typed terminal fired, so there is no adapter first-cause incident. The
  first observable failure after hydration/fence was the engine readiness and
  population boundary: lifecycle live/hydration_complete, ranking unavailable/
  incomplete_population, readiness ranking_noncurrent.

exact failed invariant:
  backend_ready requires a current complete population. At the final sample,
  CoveredPopulation + UnresolvedPopulation reconciled to UniverseTotal
  (5,207 + 487 = 5,694), but UnresolvedPopulation != 0. The unresolved set was
  exactly 486 bootstrap_origin uncertainties plus one local_invalid; there was
  no post_bootstrap_gap. Accounting remained valid, so this is not the runtime
  accounting guard.

last coherent state and first invalid transition:
  Hydration reached 5,511/5,511 and the fence reconciled without failure. The
  scanner then entered live/watermark_stale and next live/ranking_noncurrent.
  Its committed watermark stopped at 13:32:10 EDT even though ingestion and
  engine sequence continued through 275,317 and the fence supported through
  13:38:51. Watermark lag reached 408 seconds. The first invalid readiness
  transition is therefore hydration/fence completion leaving 487 canonical
  symbols unresolved, not a queue rejection.

remaining operational measurements:
  Final queue depth was 339 frames and 172,532 bytes. Deliveries reached
  249,701 with zero consumer-deferred; mean processing delay was 3 ms, maximum
  5,105 ms, and maximum sampled one-second delay 2,287 ms. Process RSS was
  approximately 1.04 GB near shutdown; the final snapshot reported
  1,785,959,440 heap-allocated bytes and 1,881,554,944 heap-in-use bytes. There
  were zero connection recovery attempts, no suppression, and T/Q remained
  aggregate-only/shed while ranking was unavailable.

invalidated claim and smallest proposed correction:
  The claim that queue capacity was the only remaining path to current ranking
  is invalidated. The 1,024-slot limit prevents the previously observed
  terminal in this run, but readiness still fails because fresh-bootstrap
  canonical coverage leaves 486 bootstrap-origin states unresolved plus one
  local invalid. The smallest next diagnosis is at the hydration-to-live
  canonical population transition: identify why successful value/empty
  hydration plus the reconciled ingress fence does not resolve those exact
  states, and distinguish the one local invalid. Do not grow the queue again,
  change readiness, fabricate marks, or treat unresolved population as ready.

remaining uncertainty:
  This bounded snapshot does not retain symbol identities, so it cannot tell
  whether the 486 states share a hydration result shape, aggregate chronology,
  or merge transition. It also cannot prove the 1,024-slot limit sufficient
  under a different provider burst. Another provider run is unnecessary until
  a deterministic population-transition diagnostic/fix is designed and
  verified.

artifacts and shutdown:
  var/run-private-scanner/d2-live-2026-08-12.log is 302 lines/16,032 bytes,
  SHA-256 1047f1bcaee664c3a60aa83ac3fcca01ea93db82eccb78f71407dd0e158f3983.
  var/run-private-scanner/d2-final-snapshot-2026-08-12.json is 3,292 bytes,
  SHA-256 90f8cd0181782448e36e7baa80fc58658f4dc71363cc73e05d3bbfa0cc461fa2.
  Both are owner-only and contain no credential, provider URL/prose, symbol,
  or raw payload. No incident JSON exists because no typed terminal occurred.
  One SIGINT stopped the scanner and dashboard; ports 8080 and 4173 were no
  longer listening.
```

No code was changed in response to this observation. The live result reopens
only the deterministic fresh-bootstrap population/readiness boundary for a
separately specified correction. Stop before another live request.

## 16. D3 one-symbol population-transition proof

**Status:** completed diagnostic-only proof authorized by the owner after the
Section 15 observation. No behavioral correction or provider request was
performed.

`P-D3-POPULATION-TRANSITION` uses one bound symbol and the real engine hydration,
canonical merge, ingress-fence, ordinary-live aggregate, live-coverage-fence,
timer, evaluator, and population-accounting paths. The candidate and control
differ only in the causal condition: an earlier REST hydration row is unequal
versus exactly equal to an already accepted live aggregate at the same
identity. Both then complete provider-value hydration, reconcile the initial
ingress fence, receive the same later independently trustworthy live mark, and
reconcile the same ordinary live-coverage fence.

The proof must record the first divergent transition among:

1. hydration row disposition and per-request coverage;
2. bootstrap ingress-fence coverage consequence;
3. later live-mark acceptance;
4. ordinary live-coverage consequence; and
5. final population partition and uncertainty origin.

It passes diagnostically only when the exact current behavior is distinguished
without injecting evaluator state. If the unequal overlap changes request
coverage to unknown, the ingress fence converts it to whole-symbol
`bootstrap_origin`, and later independent current evidence cannot clear that
population uncertainty while the equal control resolves, the smallest defect
boundary is proven. The proof does not decide the correction mechanics: it may
not clear a real historical conflict, fabricate absence, weaken readiness, or
change ranking. Stop after recording the result.

### D3 result

The matched one-symbol proof passes and reproduces the live blocker exactly.
Both paths use one valid-prior-close symbol, the same initial live aggregate,
successful `completed_value` hydration, the same initial ingress fence, the
same strictly later accepted live aggregate, the same ordinary live-coverage
fence, and the same evaluator timer. The sole difference is whether the REST
row at the initial live identity is equal (`duplicate` control) or unequal
(`conflict_or_withdrawal` candidate).

```text
equal control:
  hydration row = consumed 1, duplicate 1
  request coverage = candidate_complete
  bootstrap coverage consequence = none
  later live mark = aggregate_inserted
  ordinary coverage consequence = none
  population = trusted_rankable 1, covered 1, unresolved 0

unequal candidate:
  hydration row = consumed 1, conflict_or_withdrawal 1
  request coverage = unknown
  bootstrap consequence = unknown/bootstrap_origin
  later live mark = aggregate_inserted at a strictly later identity
  ordinary consequence = still unknown/bootstrap_origin
  population = trusted_rankable 0, covered 0, unresolved 1
```

The first divergent transition is
`applyHydrationChunkLocked`: one localized historical/live conflict changes
the request-wide `entry.coverage` to `hydrationCoverageUnknown`. The bootstrap
fence then maps any non-complete request to the symbol-wide
`coverageUnknownFailureOrFence`. After the later independently trustworthy
live mark, `extendOrdinaryLiveCoverageLocked` short-circuits whenever a prior
unknown consequence exists, so it cannot reconsider exact post-mark coverage.
Finally, `stageAggregateEvaluationLocked` checks the symbol-wide unknown before
classifying the later mark and puts the symbol directly into
`unknown_due_failure_or_fence`.

This proves the blocker, not merely correlation with the 584 live
`conflict_or_withdrawal` rows. The localized historical conflict itself is
valid evidence and must remain for history-dependent features. The defect is
letting that older localized conflict permanently preempt population
classification after an independently trustworthy later live mark and exact
post-mark coverage. This conflicts with the accepted population rule that a
historical conflict invalidates only dependent history when a current live mark
remains independently trustworthy.

The correction boundary is narrow: preserve the historical conflict and its
feature/qualification consequences, but derive population mark trust from the
latest accepted mark plus exact coverage after that mark rather than from a
sticky request-wide enum alone. The existing canonical mark authority, conflict
bitmap, and exact interval coverage are sufficient; no second state owner,
readiness change, hydration redesign, or foundational rewrite is indicated.
The separate one-symbol `local_invalid` observed live remains unproved and is
not folded into this cause.

Primary proof:
`TestD3OneSymbolHydrationConflictPopulationTransition` in
`internal/engine/hydration_population_transition_diagnostic_test.go`. It passed
once with the exact transition log, 100/100 ordinary repetitions, 10/10 race
repetitions, the complete engine short suite, and `git diff --check`. No
production file changed for D3.

### D3 narrow correction result

The diagnostic claim that became invalid was the implicit implementation rule
that any retained symbol-wide bootstrap unknown must always take population
precedence over a later canonical mark. That rule is stricter than
`DTE-MERGE-04`, `PG-AVAIL-02`, and `C3-POP-01`: an older localized historical
conflict must continue to invalidate dependent history, but it cannot erase an
independently trustworthy later live mark when exact coverage proves that no
newer mark can be hidden.

The correction changes only the Component 3 population projection. It does not
mutate or relabel the bootstrap coverage consequence. The evaluator may bypass
that consequence for the primary population bin only when all of this exact
predicate is true:

1. the retained consequence is specifically
   `unknown/bootstrap_origin`, not a post-bootstrap gap or local invalidity;
2. the latest eligible mark is canonically authoritative from live input and
   retains a valid live causal position/support;
3. at least one localized historical conflict exists strictly before the mark,
   and no conflict exists at or after the mark through committed `T`;
4. no structurally invalid mark evidence exists at or after the mark through
   `T`; and
5. the existing sole exact-coverage function proves every identity in
   `[mark.window_start,T)` present or explicitly absent, including the mark
   identity itself.

The consequence map and historical-conflict bitmap remain unchanged. As a
result, the unequal-overlap case ends with
`trusted_rankable_mark=1`, `covered_population=1`, and
`unresolved_population=0`, while HOD/session/rolling ranges remain
`invalid/historical_conflict`, Activity remains
`unavailable/history_incomplete`, and qualification remains
`unresolved/bootstrap_origin`. Day % remains current from the trusted mark and
valid prior close. The equal-overlap control reaches the same primary
population without any conflict.

`TestD3PopulationTrustDangerousCounterexamples` proves no false population
success for each requested boundary: a conflict at the latest mark, incomplete
post-mark coverage, no later trustworthy mark, a later provider failure/fence
represented by `post_bootstrap_gap`, structurally invalid evidence after the
mark, and a historical rather than live-authoritative mark. The positive
control differs only by an older conflict, a later live-authoritative mark, and
exact post-mark coverage.

Verification completed in this order:

```text
go test -timeout 90s ./internal/engine -run '^TestD3(OneSymbolHydrationConflictPopulationTransition|PopulationTrustDangerousCounterexamples)$' -count=100
  PASS; package 5.507s
go test -short -timeout 2m ./internal/engine -count=1
  PASS; package 4.678s
go test -short -timeout 2m ./...
  PASS; engine 5.451s, operations 7.300s, all packages green
go test -race -short -timeout 5m ./internal/engine -run '^TestD3(OneSymbolHydrationConflictPopulationTransition|PopulationTrustDangerousCounterexamples)$' -count=10
  PASS; package 2.746s
go test -race -short -timeout 5m ./internal/engine -count=1
  PASS; package 26.106s
go vet ./...
  PASS
git diff --check
  PASS
```

Remaining uncertainty is bounded. No provider run was performed, so the
correction has not observed the former 486-symbol live population directly;
it will resolve only those symbols whose retained facts satisfy the exact
predicate. The separately observed `local_invalid` symbol is unchanged and
unexplained. Generic hydration failure, later recovery failure/fence, incomplete
post-mark coverage, non-live authority, and historical-feature/qualification
trust remain conservative.

The required focused `gpt-5.6-sol` medium review found no defect in the
population-trust predicate, interval sufficiency, accounting, or ownership. It
did find two P2 correction/proof defects: the visible row's fields were being
blanket-masked from known `invalid/historical_conflict` to generic
`unavailable/history_incomplete`, and the D3 ordinary-fence proof used two
unbounded completion waits. The evaluator now preserves the already-derived
field-local conflict results for the population-trusted exception, while
qualification stays unresolved and Activity stays honestly incomplete. The
proof now requires the visible row's HOD conflict status and uses one bounded
one-second deadline for both fence completions. The same reviewer returned
**CLEAN/PASS** on focused re-review with no remaining P1/P2. It independently
reran the focused ordinary and race proofs plus scoped diff checking.

Post-review verification repeated the affected ladder: D3 plus counterexamples
passed 100/100 in 5.614 seconds; the complete short engine suite passed in
4.897 seconds; repository short passed with engine at 5.204 seconds and
operations at 7.072 seconds; focused D3 race passed 10/10 in 2.697 seconds;
complete short engine race passed in 25.560 seconds; vet and diff checking were
clean. No provider request, live run, specification-map edit, Git staging,
commit, push, or PR occurred.

## 17. D3 retry population-transition diagnostic

Before any separately authorized provider retry, the D3 predicate now emits a
fixed-cardinality reason ledger from the same evaluator pass that owns the
population decision. The denominator is valid-prior symbols with the exact
`bootstrap_origin` population consequence. Evaluation-order precedence makes
the bins mutually exclusive: trusted by a later live mark; no later eligible
mark; latest mark not independently live-authoritative; no strictly older
localized conflict; conflict at/after the mark; invalid evidence at/after the
mark; or incomplete exact `[mark,T)` coverage. Post-bootstrap uncertainty is
explicitly outside this diagnostic.

The ledger is copied into the immutable engine publication and exposed as
`accounting.population_transition_diagnostic` in `scanner.snapshot.v1`. A
non-ready operator sample prints the same bins together with
`unresolved_population` and `local_invalid`, so a bounded retry log does not
depend on a precisely timed API poll. The diagnostic retains no symbol,
provider payload, bitmap, or unbounded label family. It changes no consequence,
canonical evidence, population bin, qualification result, uncertainty origin,
ranking mode, readiness decision, or lifecycle transition.

The distinguishing behavioral proof requires the unequal-overlap correction
to publish `bootstrap_unknown=1` and `trusted_by_later_live_mark=1`, while the
equal control publishes an all-zero ledger. The dangerous-counterexample proof
selects every other reason exactly once and proves the denominator identity;
the post-bootstrap control remains uncounted. The C10 golden/mutation proof
shows the nonzero ledger on the wire and rejects an incoherent bin mutation,
and the operator proof requires the complete non-ready line. No provider or
credential access was performed. The final verification repeated the D3
behavior/reason matrix 100/100, the C10 mapping proof 10/10, and the operator
proof 10/10; repository ordinary verification and affected engine, operations,
snapshot, and scanner race suites passed. Vet and diff checking were clean.
Focused review found one P2: terminal suppression returned before the ledger
line, so a forced final sample could still lose the evidence as the API shut
down. Both terminal suppression and ordinary non-ready output now call the same
renderer, and a suppressed nonzero-counter regression proves the final line.
The same reviewer returned CLEAN/PASS with no remaining P1/P2.
