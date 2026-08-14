# Live T/Q protocol resilience and aggregate continuity correction

**Status:** `TQR-S1` and `TQR-S2` accepted from their deterministic proofs

**Boundary and completed-contract authority:** Direct owner request on
2026-08-14 following the owner-reported live T/Q observation; lower-level
C5/C8/C9 decisions are revisable under the
[`Version 1 Release Program`](v1-release-program.md).

**Standing program decisions:** The V1 program's
[fixed meanings](v1-release-program.md#2-fixed-product-behavior-and-revisable-delivery-decisions),
[zero-interruption policy](v1-release-program.md#3-zero-interruption-execution-policy),
and [correction loop](v1-release-program.md#4-required-correction-loop) apply.

**Advancement mode:** Sequential correction with one active implementation
slice. This correction precedes implementation of
[`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md);
C12 remains paused. No credentialed provider request is authorized.

**Controlling Phase 1 requirements:** `PG-RANK-05`, `PG-AVAIL-01`,
`PG-AVAIL-03`, `PG-TAQ-01` through `PG-TAQ-03`, `PG-OPS-02`, `PG-OBS-02`,
`PG-OBS-03`, `ARCH-OWN-01` through `ARCH-OWN-04`, `ARCH-FLOW-01` through
`ARCH-FLOW-04`, `DTE-CONTROL-01`, `DTE-RECOVERY-02` through
`DTE-RECOVERY-05`, `DTE-COMMIT-02` through `DTE-COMMIT-04`, `DTE-TQ-01`
through `DTE-TQ-03`, `DTE-REJECT-01`, `LIFE-LIVE-01`, `LIFE-LIVE-02`,
`LIFE-RECOVER-01` through `LIFE-RECOVER-06`, `LIFE-TQ-01` through
`LIFE-TQ-03`, `LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, and
`LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`.

**Approved dependencies:** Accepted C2 engine/lifecycle, C5 live adapter, C6
gap hydration, C8 runtime/recovery, and C9 T/Q membership/coverage contracts.

## Contract document map and delivery ledger

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This file | Compact single-file correction contract, Sections 1-19 | `TQR-*`, `P-TQR-*`, `TQR-S1`, `TQR-S2` | Every implementation, proof, review, or correction | Approved dependencies above |

**Layout:** Compact single-file contract. Its modest size above the template's
context target is approved because provider-status scope and the resulting
recovery consequence are one inseparable trust boundary: either half alone can
still turn optional T/Q evidence into a false global ranking failure.

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Correction contract | `accepted` | Owner direction, live incident, current-code review, official provider documentation/client behavior, both primary proofs, affected verification, and the completed narrow trust/ordering review | 2026-08-14 | Complete; reopen only through the V1 correction loop |
| `TQR-S1` — status and T/Q containment | `accepted` | `P-TQR-STATUS`: fake-WebSocket framing variants, engine aggregate-equivalence/quarantine trace, and operations accounting-partition trace pass; C5/C8/C9 ledgers updated | 2026-08-14 | Complete |
| `TQR-S2` — aggregate recovery and runtime continuity | `accepted_after_review_correction` | `P-TQR-CONTINUITY`; review-discovered quiet-socket lifecycle and parent-deadline defects corrected and re-proved | 2026-08-14 | Complete |

## Sections 1-4 — Outcome, scope, ownership, and settled boundary

The scanner will continue publishing exact aggregate rankings when subscription,
entitlement, acknowledgement, decoding, coverage, or accounting uncertainty is
provably T/Q-local. A delayed, split, duplicate, extra, error, or unsolicited
post-handshake status may make Tape Rate and Spread unavailable and may disable
further T/Q commands for that connection epoch; it cannot suppress ranking.

When evidence shows an aggregate frame may actually have been lost, the scanner
stops claiming currentness, closes the epoch, hydrates the exact gap through the
existing C6 path, and resumes after its causal fence. A recoverable suppression
does not terminate the process or API: an explicit bounded scheduled event may
enter `recovering`. Canonical contradiction, restart-required suppression,
session end, and controlled shutdown remain terminal.

In scope are C5 status framing/correlation and event-family classification; a
typed T/Q-control quarantine fact; C9 epoch-local command/coverage containment;
C5/C8 accounting failure-domain separation; recoverable transport-loss routing;
bounded scheduled same-binding recovery; and diagnostic evidence needed to
distinguish these outcomes.

Not in scope are Tape Rate or Spread formulas, aggregate qualification or
ordering, another ranking mode, silent approximate-current output, another
socket or service, public deployment, credentials, live reruns, provider SLA,
or C12. During a true aggregate gap, retained rows may be exposed only under
their existing stale/noncontracted meaning; `qualified_current` remains exact.

The adapter owns socket I/O, frame positions, classification, and one pending
wire command. It returns bounded facts. The `ScannerStateEngine` remains the
only owner of lifecycle, aggregate coverage, desired/provider T/Q membership,
T/Q quarantine, field availability, watermark, and ranking. Operations owns
bounded scheduling/composition but cannot clear suppression or declare
currentness.

| Boundary | Settled meaning | Authority |
| --- | --- | --- |
| T/Q independence | Every T/Q-only failure changes only membership, coverage, pressure, and dependent fields. | `PG-AVAIL-03`, `LIFE-TQ-02`, `LIFE-TQ-03` |
| Status evidence | Generic provider statuses have causal positions but no provider command token; correlation is valid only while the sole pending command remains unambiguous. | `DTE-CONTROL-01`, `DTE-TQ-01` |
| Aggregate loss | Known or possible aggregate loss closes currentness and uses exact gap recovery before suppression. | `DTE-RECOVERY-02`-`05`, `LIFE-RECOVER-01`-`06` |
| Suppression | Only material global ambiguity can suppress; same-binding recovery requires an explicit event. | `LIFE-SUPPRESS-01`-`03` |
| Publication | Process liveness, aggregate ranking currentness, and T/Q field currentness remain distinct. | `PG-OBS-03`, `LIFE-PUBLISH-01`-`03` |

This correction adds no product rule, mutable owner, watermark, evaluator,
T/Q-to-ranking dependency, changed market-time window, or duplicated canonical
state.

## Sections 5-7 — Evidence, reconnaissance boundary, proof shape, and checkpoint

The 2026-08-14 incident preserved this first cause:

```text
adapter_terminal / protocol/status_ambiguous
epoch 1 position (1,79920,0)
lifecycle suppressed / ingress_integrity
```

The preceding samples remained `qualified_current`, with 20 rows and a
one-second aggregate watermark lag. The retained diagnostic does not include
the raw provider status, so it establishes the failure domain and consequence,
not the exact provider message.

Current code review establishes that:

- a status with no pending T/Q command becomes global ingress integrity;
- expected acknowledgement cardinality is evaluated per frame, so two paired
  successes split across frames become partial then unsolicited;
- handshake steps require the next frame to contain exactly one status;
- a structurally explicit but unfamiliar `ev` is treated like unknowable event
  identity;
- frame capacity/oversize/receipt failures bypass gap recovery and suppress;
  and
- combined adapter accounting can promote T/Q-command accounting failure to
  global ingress integrity.

The current Massive
[WebSocket quickstart](https://massive.com/docs/websocket/quickstart) documents
asynchronous array batching but no per-channel or same-frame subscription-ack
guarantee. At commit `eef5a9ae787a117b0d0701c211b209226e478ebe`, the official
[Go client status handler](https://github.com/massive-com/client-go/blob/eef5a9ae787a117b0d0701c211b209226e478ebe/websocket/client.go#L456-L479)
logs generic `success`, `error`, and unknown statuses and treats only
`auth_failed` as fatal. That client is behavior evidence, not an implementation
dependency or authority for scanner coverage.

No new Version 2 inspection is needed or permitted. Accepted C5/C9
reconnaissance already records the predecessor's generic status and paired
command evidence; the live incident and current official sources invalidate
the lower-level same-frame/global-failure interpretation. The implementation
whitelist is current repository code under `internal/massive`,
`internal/engine`, `internal/operations`, `cmd/scanner`, their focused tests,
and this contract. No predecessor code is added.

The boundary and detailed contract are recorded under the direct owner request
and V1 correction authority. The skeleton checklist is satisfied: exact IDs,
ownership, non-scope, evidence, two proof boundaries, and two sequential slices
are explicit; no Version 2 source was opened for this correction. An independent
skeleton review is not required. The drift audit is clean.

## Section 8 — Reuse and correction decisions

| Evidence/source | Decision | Preserved behavior | Lower-level rule removed | Proof |
| --- | --- | --- | --- | --- |
| Current C5 classifier, cursor, FIFO, and causal positions | Adapt | Bounded streaming JSON, exact `(epoch,frame,index)`, causal-prefix preservation, credential redaction | Per-frame command completion and every-orphan-status global failure | `P-TQR-STATUS` |
| Current C9 opaque command and unknown membership | Adapt | One engine-issued command, no write-as-coverage, fresh warm-up, explicit unknown | Cleanup command after the epoch's generic status correlation is contaminated | `P-TQR-STATUS` |
| Current C6/C8 gap recovery | Reuse and compose | Freeze currentness, exact hydration plan, live ingress fence, bounded attempts | Direct suppression for representable recoverable frame loss | `P-TQR-CONTINUITY` |
| Official Massive docs/client at provenance above | Behavior evidence | Statuses are generic/asynchronous; authentication failure is distinct | Assumed provider token, per-channel ack, same-frame guarantee | Both |

## Sections 9-14 — State, required behavior, trust, failures, and bounds

The adapter may retain one bounded pending-command accumulator containing the
engine command identity, write boundary, deadline, expected count, cumulative
success count, last status position, and failure/contamination state. It retains
no arbitrary provider prose. The engine adds at most one epoch-local T/Q-control
quarantine state and bounded reason. Quarantine closes all T/Q coverage, marks
unresolved provider membership unknown, prevents every later T/Q command in
that epoch, and clears only on a greater acknowledged connection epoch or
terminal lifecycle. For `same_binding_recovery_allowed`, the engine exposes at
most one opaque one-shot recovery command containing the binding, failed epoch,
retry ordinal, and earliest eligible process time. Operations may return only
that command after its deadline; callers cannot directly set `recovering`.

Construction prevents two pending commands, caller-created acknowledgement or
coverage, orphan reuse, adapter-owned lifecycle mutation, and supervisor-owned
recovery transitions. Runtime validation remains necessary for frame syntax,
phase, epoch, causal position, cumulative status count, deadline, explicit
event family, accounting partition, recovery-command identity, and retained
aggregate boundary.

| Requirement | Required behavior |
| --- | --- |
| `TQR-STATUS-01` | Accumulate correlated success elements across frames after write success. Interleaved A/T/Q elements retain their causal dispositions. Complete success only at the exact expected cumulative count before deadline. |
| `TQR-STATUS-02` | A T/Q failure, partial deadline, extra, late, duplicate, unsolicited, wrong-phase, or unknown post-handshake status never emits aggregate ingress integrity. Emit a bounded T/Q failure/quarantine fact, close coverage, and keep aggregates live. Never correlate an orphan status to a later command. |
| `TQR-HANDSHAKE-01` | Connected, authentication, and A-subscribe phases consume classified elements until their bounded terminal evidence. Batching does not require a singleton status frame. Pre-ack market elements cannot establish coverage; causally post-A-ack aggregates use the ordinary path. `auth_failed` or connection loss ends the attempt. |
| `TQR-CLASS-01` | Missing, duplicate, malformed, or non-string event identity remains genuine raw ambiguity. A valid explicit unfamiliar event family is an observable bounded unsupported-family rejection and does not imply aggregate loss. |
| `TQR-ACCOUNT-01` | Aggregate/control transport accounting and optional T/Q command/normalization accounting have separate identities and consequences. T/Q-local contradiction enters quarantine/`aggregate_only`; only aggregate/control ambiguity enters aggregate recovery or suppression. |
| `TQR-RECOVER-01` | Queue capacity, oversize, receipt regression, or unclassifiable raw loss that may contain aggregates closes the epoch and enters exact gap recovery when a safe retained boundary exists. Suppress only when construction cannot name a recovery boundary, recovery fails/exhausts, or canonical integrity is contradictory. |
| `TQR-RUNTIME-01` | `same_binding_recovery_allowed` prohibits an immediate socket reopen but does not end the process. The engine issues one opaque recovery command; a bounded scheduler returns it no earlier than its finite backoff and the engine alone enters `recovering`. API/liveness remain available. `restart_required`, session end, and controlled stop remain terminal. |
| `TQR-PRESSURE-01` | Pressure observation must shed T/Q before the configured raw queue can reach its fatal capacity under the accepted deterministic burst. Threshold/cadence values are measured lower-level decisions; no adapter-owned ranking or pressure policy is added. |
| `TQR-DIAG-01` | Persist a bounded first-cause class, phase, epoch/position, pending-command counts/deadline outcome, quarantine/recovery disposition, and aggregate/TQ accounting partitions. Credential and arbitrary provider prose remain excluded. |

| Trust boundary | Accept as success only when | Contain | Dangerous false success |
| --- | --- | --- | --- |
| T/Q acknowledgement | Write succeeded and the sole command accumulated exactly its expected successes before deadline without contamination | Fail/quarantine T/Q only | A late success opens coverage for the next command |
| Status scope | Authentication/aggregate-handshake status occurs in its active phase; post-handshake generic status is T/Q-local unless socket/aggregate loss is separately proved | Fence or quarantine by phase | Generic `error` suppresses healthy aggregates |
| Raw loss | The last committed aggregate boundary and new epoch/fence can define exact recovery | Recover; suppress if no safe proof or recovery exhausts | Continue labeling ranking current across a lost aggregate frame |
| Same-binding restart | Engine admits an explicit scheduled recovery event after its bound/backoff | Remain suppressed/stale without opening a socket | Supervisor reconnect hot loop bypasses lifecycle ownership |

On the success path, the aggregate handshake establishes its causal boundary,
split T/Q statuses complete only their pending command, coverage warms from the
final acknowledgement position, and rankings advance independently. On a
T/Q-control failure, coverage closes and the epoch is quarantined while A
continues. On possible A loss, the epoch closes, exact hydration and its ingress
fence restore currentness, or bounded failure reaches an honest suppression
whose opaque scheduled command is the only same-binding continuation.

Primary accounting remains partitioned. Raw frames reconcile to admitted or one
terminal rejection; admitted frames reconcile to dispositioned or fenced.
Status elements reconcile to handshake, pending-command cumulative evidence,
T/Q quarantine diagnostic, or bounded rejection. T/Q commands reconcile to
pending, acknowledged, failed, quarantined/fenced. Aggregate admissions,
watermark, qualification, and ranking counters must equal an A-only control for
every T/Q-only trace. Reasons and diagnostics are closed and fixed-cardinality.

All queues retain existing hard bounds. One command and one accumulator exist
per epoch. Scheduled recovery has one timer, one outstanding event, finite
per-attempt deadlines, bounded exponential delay capped at a contract-selected
value, and no tight loop. S2 must choose pressure cadence/thresholds from a
compact deterministic burst below the 512-frame hard limit. It may not enlarge
the queue as the primary fix.

Evidenced cases are the observed orphan status; split, partial, extra, and
write/ack races already representable in current tests; official asynchronous
status/batching behavior; explicit provider event families; existing queue
capacity/oversize tests; and existing C6 recovery traces. No speculative status
message parser or undocumented provider-message inference is authorized.

## Sections 15-17 — Primary proofs, slices, discretion, and correction

`P-TQR-STATUS` is one deterministic fake-WebSocket differential trace. It runs
the same valid A stream with: paired successes in one frame and split across
frames; A/T/Q interleaving; delayed, duplicate, extra, unsolicited `success`,
`error`, unknown status, and `auth_failed`; explicit unfamiliar `ev`; malformed
T/Q; and command-accounting contamination. Every T/Q-only variant must produce
the same aggregate watermark, qualification, rank order, rows, and readiness
as the A-only control while exposing exact T/Q coverage/quarantine. The
authentication case must end the epoch and request recovery, not claim current
coverage. The proof does not establish live provider chronology or entitlement.

`P-TQR-CONTINUITY` is one deterministic runtime trace with a small exact
binding. A T/Q burst crosses the selected early-shed boundary without losing A.
Separate injected capacity/oversize/unclassifiable losses freeze currentness,
close the epoch, hydrate `[T,R)`, reconcile the live fence, and resume the exact
A-only ranking. Recovery failure reaches `same_binding_recovery_allowed`, opens
no socket until an explicit bounded scheduled event, keeps API/liveness
available, and never hot-loops. Restart-required suppression still exits. The
proof does not claim production saturation, provider SLA, or permissible
approximate-current ranking.

| Slice | Coherent outcome | Requirements/proof | Allowed boundary | Deferred |
| --- | --- | --- | --- | --- |
| `TQR-S1` | Provider status framing is asynchronous and T/Q-local ambiguity quarantines only T/Q while exact aggregate ranking continues. | `TQR-STATUS-01`, `TQR-STATUS-02`, `TQR-HANDSHAKE-01`, `TQR-CLASS-01`, `TQR-ACCOUNT-01`, `TQR-DIAG-01`; `P-TQR-STATUS` | `internal/massive`, typed T/Q/control seams in `internal/engine`, focused `internal/operations` accounting, tests, and this ledger | Recoverable raw loss routing, scheduler, measured early pressure |
| `TQR-S2` | Actual aggregate loss recovers before suppression; recoverable suppression keeps the process/API alive and pressure sheds T/Q before hard capacity. | `TQR-RECOVER-01`, `TQR-RUNTIME-01`, `TQR-PRESSURE-01`; `P-TQR-CONTINUITY` | `internal/massive`, `internal/engine`, `internal/operations`, `cmd/scanner`, focused API status tests, and this ledger | Live-provider confirmation and any new approximate ranking mode |

Implementation may choose private accumulator helpers, typed enum names,
bounded backoff constants, and equivalent test decomposition. It may revise
existing lower-level C5 `C5-STATUS-01`/`C5-COMMAND-01`, C8 recovery settings,
and C9 `C9-COVER-01` cleanup mechanics only as required here and must record
the affected evidence in their parent ledgers at slice acceptance.

Prohibited changes include parsing arbitrary provider prose as trusted command
identity, treating write success as coverage, correlating an orphan status to a
new command, allowing T/Q to affect ranking/readiness, continuing a current
claim across actual aggregate loss, adding a second socket/state owner, changing
feature formulas, or accessing credentials/provider services.

The consequential external-status and cross-component recovery interfaces
trigger one narrow `gpt-5.6-sol` medium read-only review after both slices
stabilize: invalid external evidence could otherwise open false T/Q coverage or
suppress valid aggregate ranking, and the new opaque recovery command crosses
engine/runtime ownership. A failure records evidence, reopens only the
implicated slice/parent claim, and follows the V1 correction loop without an
owner stop.

## Sections 18-19 — Acceptance and drift audit

### TQR-S1 acceptance record

The adapter now correlates the sole written T/Q command with cumulative status
evidence across frames. It completes only at the exact expected success count;
partial deadlines, extra or duplicate statuses, provider errors, unfamiliar
statuses, wrong-phase evidence, write failure, and unsolicited or delayed
statuses emit a bounded T/Q-control quarantine fact instead of aggregate
ingress integrity. The accumulator retains only token/count/deadline/position
evidence and never provider prose. A generic status is never reused for a later
command.

Handshake phases now consume classified array elements until their bounded
terminal status rather than requiring one singleton status frame. Market and
rejection elements retain exact array order: pre-A-ack aggregates remain before
the acknowledgement, post-A-ack aggregates remain after it, and malformed T/Q
cannot hide a later aggregate. `auth_failed` ends the connection epoch through
ordinary connection-loss recovery. A syntactically valid unfamiliar `ev` is a
bounded unsupported-family rejection; missing, duplicated, malformed, or
non-string `ev` remains raw ambiguity.

The engine owns one epoch-local quarantine. It retires the pending command,
closes all T/Q coverage, converts unresolved membership to unknown, prevents
later commands in that epoch, exposes the bounded first cause/counts/deadline,
and clears only after a greater aggregate-acknowledged epoch. Adapter command
and status identities and operations checks are partitioned so optional T/Q
accounting contradiction enters this quarantine while queue and non-T/Q
transport contradiction retain the aggregate-ingress path.

`P-TQR-STATUS` is decomposed into
`TestPTQRStatusAsyncCorrelationAndContainment`,
`TestPTQRStatusQuarantinePreservesAggregateState`, and
`TestPTQRAccountingPartitionQuarantinesOnlyTQ`. Together they cover one-frame
and split success, handshake batching, A/T/Q interleaving, partial/extra/
duplicate/delayed/unsolicited/error/unknown/wrong-phase/authentication status,
unsupported `ev`, malformed T/Q, redacted fixed-cardinality diagnostics, fresh
epoch reset, exact command/status accounting, and byte-for-byte-equivalent
canonical aggregate/evaluation/watermark state across T/Q quarantine. The
focused proof, its affected-package race execution, and the complete
`internal/massive` short suite pass.

The required repository-wide short command was run and remains red only in
three pre-existing, out-of-slice dirty-worktree regressions:
`TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect`,
`TestSuppressedIngressCannotEnterReconnectHotLoop`, and
`TestLiveWarmupSnapshotIsServableAndBound`. They concern the separately queued
recovery/suppression and warm-up snapshot work; TQR-S1 neither changes their
paths nor claims them as passing. The engine and operations failures were
observed in the pre-edit baseline; the snapshot regression file was already an
untracked user change and is likewise outside this slice. No TQR-S1 proof or
Massive package test fails. The risk-triggered independent review remains deferred as
contracted until S2 stabilizes; S2's raw-loss recovery, scheduler, and measured
early-pressure work remain untouched.

### TQR-S2 acceptance record

Possible raw loss now has two exact outcomes. If live canonical state has a
committed supported boundary, queue capacity, oversize, receipt regression, or
unclassifiable raw loss closes the epoch, freezes the current claim, retains
that boundary, and enters the existing `recovering` gap-hydration/fence path.
If construction cannot name that boundary, or bounded recovery exhausts, the
engine publishes `same_binding_recovery_allowed`; it does not permit an
immediate reconnect.

For recoverable suppression the engine owns one opaque scheduled command with
binding identity, failed epoch, monotonic retry ordinal, issue time, and
earliest eligible process time. The finite exponential delay is one second,
doubling to a 30-second cap. Operations can return only the issued command
after its delay; early, stale, foreign, duplicate, or caller-created evidence
cannot select `recovering`. `RunLive` remains joined and the API remains
servable while waiting. Session end, controlled stop, and restart-required
suppression are unchanged.

Pressure sampling is now 100 milliseconds. A 25%-full raw queue sample enters
`taq_degraded` immediately, and 60% enters `aggregate_only`; the 512-frame hard
limit is unchanged. The paired proofs admit a real 128-frame burst to the raw
queue without capacity loss, leaving 384 slots, and separately apply the exact
128/512 engine sample that activates shedding. They preserve the existing
mixed-frame aggregate/control classification guarantee; they do not claim a
host-saturation or sampler-versus-burst race frontier.

`P-TQR-CONTINUITY` is decomposed into
`TestPTQRRecoverableIngressLossRoutesToExactGapRecovery`,
`TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect`,
`TestPTQRScheduledRecoveryIsTheOnlySameBindingContinuation`,
`TestPTQRQuietLiveSocketCannotMaskSessionEnd`,
`TestPTQRQuietLiveSocketCannotMaskRestartRequiredSuppression`,
`TestPTQRQuietLiveSocketHonorsParentDeadline`,
`TestPTQRPressureBurstLeavesHardCapacityHeadroom`,
`TestPTQRPressureShedsAtQuarterQueueBeforeHardCapacity`, and
`TestPTQRRecoverableSuppressionKeepsAPIAndLivenessAvailable`, plus the existing
C6/C8 disconnect-to-gap-hydration-to-live traces. Together they prove the safe
loss split, exact retained boundary, engine-only delayed continuation, no
pre-deadline socket, no reconnect hot loop, process/API continuity, early T/Q
shedding, and unchanged exact recovery-to-live behavior. The former engine and
operations regressions named in the S1 record are resolved; the unrelated
pre-existing `TestLiveWarmupSnapshotIsServableAndBound` dirty-worktree
regression remains outside TQR-S2. Live provider chronology, production
saturation, and approximate-current ranking remain unproved and deferred.

The allocated medium-reasoning read-only review found that the first live-loop
implementation observed terminal lifecycle only between connection attempts.
The correction bounds a quiet delivery wait to 100 milliseconds, closes and
joins the active attempt on session end or restart-required suppression, and
returns recoverable states to the engine-issued scheduler. Focused re-review
then found that a parent deadline could be mistaken for the internal polling
deadline; continuation now also requires the parent context to remain live.
The three quiet-socket tests above distinguish all terminal/cancellation paths,
and the same reviewer reported the findings resolved with no remaining P1/P2.

Each slice first runs its named focused proof under two minutes, then
`go test -short -timeout 2m ./...`. After both stabilize, run affected
`internal/massive`, `internal/engine`, `internal/operations`, and `cmd/scanner`
race tests under five minutes, `go vet ./...`, and `git diff --check`. A compact
non-short burst may be added only if ordinary scale cannot cross the selected
pressure boundary; it remains below 15 minutes. Credentialed live validation
is separately authorized and not an acceptance gate.

Acceptance requires exact aggregate equivalence for every T/Q-only variant,
fresh T/Q warm-up after quarantine, no orphan reuse, exact accounting, no
reconnect hot loop, successful loss-to-recovery-to-live traversal, coherent
API/liveness throughout recoverable suppression, resolved links, updated C5/C8/C9
parent ledgers, and no unresolved fixed-authority conflict.

| Drift question | Answer | Evidence |
| --- | --- | --- |
| New product rule? | No | Implements existing T/Q independence and explicit recovery; approximate-current ranking is excluded. |
| New owner, watermark, evaluator, or T/Q ranking gate? | No | Adapter returns facts; engine retains all market/lifecycle authority. |
| Changed session, event-time, correction, or feature windows? | No | Only provider-control correlation and failure containment change. |
| Unevidenced behavior? | No | Live incident, current regressions, official provider sources, and accepted recovery invariants cover every case. |
| Duplicated responsibility or unnecessary machinery? | No | One accumulator, one engine quarantine state, and one bounded existing-runtime scheduler; no new service or framework. |
| Version 2 authority? | No | No new predecessor inspection or implementation reuse. |
