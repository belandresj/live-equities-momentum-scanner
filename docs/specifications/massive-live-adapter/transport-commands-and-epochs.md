# Massive live adapter — transport, commands, and epochs

**Parent contract:** [Massive live adapter](../massive-live-adapter.md)
**Normative responsibility:** One-socket transport, connection epochs, bounded
frame/queue behavior, commands, acknowledgements, loss/reconnect facts, and
containment
**Controlling requirements:** `C5-ENGINE-01`, `C5-TRANSPORT-01`,
`C5-COMMAND-01`, `C5-BOUND-01`, `C5-RECONNECT-01`, `C5-LIVE-01`;
parent-routed
`PG-TAQ-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-FLOW-01`–`04`, `DTE-EVENT-01`, `DTE-EVENT-02`,
`DTE-CONTROL-01`, `DTE-MERGE-03`, `DTE-TQ-01`–`03`,
`LIFE-MODEL-01`–`04`, `LIFE-INIT-04`, `LIFE-INIT-05`,
`LIFE-HYDRATE-01`, `LIFE-HYDRATE-03`, `LIFE-HYDRATE-07`,
`LIFE-LIVE-02`, `LIFE-RECOVER-01`–`03`, `LIFE-RECOVER-06`,
`LIFE-TQ-02`, `LIFE-TQ-03`, `LIFE-END-01`–`03`, and the parent-listed
`LIFE-T*` transitions
**Allocated slices:** `C5-S2`, `C5-S3`; see the parent delivery ledger
**Document dependencies:** Parent; provider classification and normalization;
Component 2 S1/S2/S3/S4 contracts
**Approval state:** Inherits the parent completed-contract approval; not an
independent component authority
**Delivery state:** See the authoritative parent delivery-state ledger; do not
copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance used v2 commit
`5f92a151dd850002578a33a81ad90dea096c63b6`. Only the listed function roles
and focused offline tests were inspected.

| Exact source/test and SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- |
| `internal/massive/websocket.go` — `WebSocketConnection`, `Coordinator.Run`, `dialAndHandshake`, `dynamicWriter`, `readPhaseCount`, `explicitStatus`, `explicitFailureStatus`, `writeStep`, `read`, `heartbeat`, `randomBackoffDelay`, `closeBounded`, `classifyTermination`, and controlled-disconnect helpers; `f31dac57cb1ec4229c510efa936dc98b3b297b7a916652c56c300020c46a7fda` | One socket, strict connected/auth/subscription phases, read limit, text/binary UTF-8 frames, heartbeat, per-connection epoch, bounded writes/closes, serialized dynamic writer, exact status counts, end marker before next epoch, and redacted error vocabulary are useful behavior. | `adapt` | Reject API-key fields in facts, exported mutable coordinator fields, direct `HealthStore`/metrics/TAQ callbacks, combined initial A/T/Q subscription, adapter-selected lifecycle/currentness, automatic infinite reconnect loop, random backoff, global frame sequence, and background worker result ownership. Retain adapter allocation of monotonic attempt epochs, but only engine consumption makes one active; all commands/results are immutable. | `P-C5-TRANSPORT`, `P-C5-COMMAND`, `P-C5-BOUND`, `P-C5-RECONNECT` |
| `internal/massive/websocket_test.go` — `TestWebSocketHandshakeHeartbeatTextAndBinary`, `TestAuthenticationFailureIsTerminalAndRedacted`, `TestOutOfPhaseSuccessStatusIsProtocolAmbiguity`, `TestHandshakeStepAndCompleteTimeouts`, `TestPingTimeoutLatchesCoverageAndCancelsHeartbeat`, `TestPlannedCloseIsForceBounded`, `TestReconnectMarkerPrecedesNewEpoch`, `TestFakeProviderEndToEndSelectedTAQ`; `95073220c77debfc03d1f70d59c09d0822ea95b7b391b6dfab509167124a26fd` | Fake servers establish handshake ordering, text/binary acceptance, phase ambiguity, timeout/close behavior, credential redaction, epoch marker order, and `A.*,T.SYM,Q.SYM`/paired dynamic commands. | `behavior evidence` | Reuse offline protocol shapes only. Initial Component 5 command is `A.*` alone; T/Q promotion remains an engine-issued later delta. Replace health/snapshot assertions with typed fact, FIFO, and engine disposition assertions. | Same four transport proofs; `P-C5-LIVE` |
| `internal/massive/queue.go` — `Frame`, `EnqueueFailure`, `FrameQueue`, `NewFrameQueue`, `TryEnqueue`, `EnqueueEndMarker`, `Pop`, `TryPop`, `CloseAcceptanceGate`, `CloseProducer`, `Gauge`; `cc2294d340ca3f8d42e1c5a936c38c2ec6f2d8f162f4b1f266efa1b473831123` | Count/byte/frame bounds, copy-on-admission, receipt-regression rejection, FIFO, acceptance fence, required epoch marker waiting for capacity, close/wakeup, and bounded latest gauge are strong reusable techniques. | `adapt` | Frame sequence becomes positive and per epoch, not one global ingress number. Remove scanner metrics dependency and linear-scan gauges. The engine owns the accepted epoch/fence consequence. Queue saturation/oversize is an explicit epoch-loss result, not a readiness mutation. | `P-C5-BOUND`, `P-C5-RECONNECT` |
| `internal/massive/config.go` — `Config` queue fields, `ProductionQueueFrames`, and queue validation only; `5472ad07aea7175ba1d445b84be2906436364719327263641c22d2aac07a1eaa` | V2 used 512 frames, 64 MiB total bytes, and 8 MiB per frame, with positive and `frame<=total` validation. | `behavior evidence` | Do not port environment, URLs, credentials, REST, readiness, TAQ-cap, listen, data-directory, or snapshot configuration. Treat the values as Component 5 hard safety ceilings; production chosen values and pressure/currentness thresholds remain Components 8/9. | `P-C5-BOUND` |
| `internal/massive/config_queue_test.go` — `TestFrameQueueCountBytesOrderingGateAndClock`, `TestEndMarkerWaitsForCapacityAndOrdersEpochs`; `e46e18c70fecb3ae95815ffbdd0a7c88010938983ed762fc9bb09e097dbc72c2` | Offline tests prove byte/count saturation leaves state unchanged, FIFO, copy/receipt/fence behavior, and end marker ordering. | `adapt` | Replace global sequence/currentness assertions with per-epoch frame sequence and terminal epoch fact. | `P-C5-BOUND`, `P-C5-RECONNECT` |
| `internal/massive/taq.go` — `BeginEpochAt`, `InitialAcknowledgedAt`, `ReserveNextDelta`, `DynamicWriteSucceeded`, `DynamicWriteFailed`, `channels`, `HasPending`, `AcknowledgeAt`, `ConnectionLostWithCause`; `84c3501999c5abd8a4e9dda48507e15f9e2129eec6a82a6164090cff46d9ab43` | V2 demonstrates one initial wildcard command, deterministic paired T/Q channel text, reservation before write completion, one pending dynamic command, exact success count, wrong epoch rejection, and loss clearing pending state. | `behavior evidence` | Do not port `TAQState`, desired/subscribed/member/coverage/feature/pressure/snapshot ownership. Component 5 transports engine-issued immutable command intent and returns write/status facts; Component 9 later owns membership and coverage. | `P-C5-COMMAND` |
| `internal/massive/phase0_lifecycle_test.go` — `TestPhase0ConsecutiveEpochLifecycleUsesProductionCoordinator`, `TestPhase0HeartbeatTerminationClasses`, `TestPhase0HeartbeatFailuresUseProductionWorker`, adapter portion of `TestPhase0CancellationPartialFailureAndIdempotentShutdown`; `7ade3a3eedee10b7ac77eeb4635ac5640a2a18501bd4fb4c96557dbdd38ca1cd` | Offline connections prove distinct consecutive epochs, one termination class, pre-session disconnect not pretending session coverage, heartbeat provenance, idempotent quiescence, and first termination not overwritten. | `behavior evidence` | Remove predecessor health/recovery/orchestration. Retry/backoff count and lifecycle consequence remain engine/Component 8 decisions. | `P-C5-RECONNECT` |
| `internal/massive/phase6_controlled_disconnect_test.go` — `TestControlledDisconnectRejectionsAndConfiguration`, `TestPhase6ControlledDisconnectProductionPath`; `8d5fb79d25b8edc76996f8f8ea8bdd6f25232a31af9f4605f0e142953105603e` | An epoch-bound close request cannot close a future epoch; accepted close produces one epoch termination/marker, reconnects without canceling root context, and joins workers on final cancellation. | `behavior evidence` | Operator-only one-shot feature/config is not Component 5 product scope. Preserve only epoch-bound close fencing and worker join behavior. | `P-C5-RECONNECT` |
| `internal/massive/phase3_taq_resilience_test.go` — `TestPhase3PairedAcknowledgementCausality`, `TestPhase3HeartbeatProvenance`, `TestPhase3StatusFailurePreservesAggregateAndSnapshotPrivacy`; `759e16929d991f7c5648779f757f232a920e496278359d04396c6ff65777d170` | Ack-before-write interleaving, extra success, failed write, retained provider rejection, bounded termination provenance, and T/Q status failure coexisting with a same-frame aggregate are important counterexamples. | `behavior evidence` | No TAQ state/feature/pressure/snapshot port. The adapter reports both ordered facts; the engine owns aggregate/control consequence and Component 9 later owns T/Q coverage. | `P-C5-COMMAND`, `P-C5-LIVE` |
| `internal/massive/owner.go` — `Owner.Run` queue-drain loop and `finishPlanned`; `f3e68f7b233a6ad6ab82e61889e60e37b128f5d747d3bf826380af622d091700` | V2 drains accepted frames before planned completion and avoids timer starvation, but also owns evaluation, recovery, T/Q, readiness, and publication. | `reject` | Component 2 already owns FIFO consumption, lifecycle, evaluation, and publication. Do not port a second owner or outer scanner loop. | `P-C5-LIVE` source/ownership inspection |
| `internal/massive/owner_test.go` — `TestSignalShutdownDrainsAcceptedFramesWithoutGoroutineLeak`, `TestAcceptanceFenceProcessesImmediatelyPrecedingFrame`; `62456a0b168f964c797dcf63b2c88f44b13876d3a1e97ba482faf5c520a32ab1` | Accepted work drain and fence order are useful failure scenarios. | `behavior evidence` | Re-express only accepted-frame drain/fence behavior at the adapter-to-Component-2 seam. Readiness/ticks/publication are excluded. | `P-C5-BOUND`, `P-C5-LIVE` |
| `internal/massive/concurrent_stability_test.go`; `5db936713a420a6d72938634f06bf06d0a2fa53fc1b9aae7bc7901f294867d4b` | The only named test couples socket processing to HTTP polling/readiness and provides no isolated Component 5 proof. | `reject` | API/readiness and broad orchestration are excluded. | None |

### 8.1 Reconnaissance conclusion

The simplest conforming design is not a Coordinator/Decoder/Owner port. It is
one engine-commanded socket attempt, one bounded raw-frame FIFO, one sequential
classifier/normalizer, and direct typed admissions/facts. Component 5 owns no
reconnect policy loop, health store, readiness evaluator, T/Q membership,
coverage, or snapshot.

V2's `github.com/coder/websocket` API gives a narrow, context-aware connector
and is the proposed sole third-party runtime dependency. The exact module
version must be pinned in `go.mod` by `C5-S3`; changing to another WebSocket
library or importing any predecessor module requires owner review. The package
interface is hidden behind one private adapter-owned connection interface for
offline tests, not exposed as a general transport abstraction.

Targeted range reads also displayed colocated excluded declarations in
`config.go` (environment/REST/listen/data configuration), `owner.go` and
`owner_test.go` (readiness/evaluation/publication portions),
`phase3_taq_resilience_test.go` (T/Q feature/pressure assertions), and trailing
test helpers in `phase6_controlled_disconnect_test.go`. No environment value or
credential was read, no network/live function was invoked, and none of that
material is evidence or whitelist input here. This was an inspection-scope
variance against Section 6's declaration-role filter and requires explicit
owner acknowledgement at the completed-contract gate; it is not silently
marked conforming.

## 9. Detailed semantic inputs, outputs, and owned state

### 9.1 Engine-issued commands

The engine owns the command intent and accepted epoch. The adapter accepts only
these immutable command families:

| Command | Required fields and meaning | Bound |
| --- | --- | --- |
| `open_aggregate_epoch` | Binding identity, positive command token, and explicit finite dial/handshake/heartbeat/close durations. It authorizes exactly one attempt; the adapter allocates the next positive process-local epoch before dialing and reports it in `connection_attempt`. | One active/opening attempt per adapter; token never zero/reused and allocated epoch strictly increases without wrap/reuse. |
| `change_tq` | Binding identity, current open epoch, positive token, `subscribe` or `unsubscribe`, sorted unique exact bound symbols, and paired `T.SYM,Q.SYM` channels. Component 9 later creates this intent. | At most 20 symbols and 40 channels; one dynamic command in flight. |
| `close_epoch` | Binding identity, exact current epoch, positive token, and bounded cause `session_end`, `controlled_stop`, `superseded`, or `integrity_loss`. | Idempotent per epoch; cannot close a different/future epoch. |

Credentials are an adapter construction dependency, not a command field. A
credential is used only to construct the provider auth write and may never be
copied into a fact, error, log, metric, fixture, or retained command record.
This contract does not authorize reading a real credential during design or
verification.

The adapter validates command shape and binding before I/O. It neither creates
T/Q desired membership nor decides a reconnect. `open_aggregate_epoch` for a
larger epoch is issued only after the engine has consumed the prior terminal
epoch fact and selected the legal lifecycle path.

### 9.2 Adapter-returned facts

Every returned fact is immutable, bounded, and contains binding identity,
epoch, command token where applicable, receipt/completion time, and one closed
result/reason:

- connection attempt started, connected, authentication acknowledged/failed,
  aggregate subscription write succeeded/failed, aggregate subscription
  acknowledged/failed/ambiguous;
- T/Q command write succeeded/failed and T/Q acknowledgement
  complete/rejected/ambiguous, including expected/observed status-element count
  and last acknowledgement causal position;
- normalized A/T/Q item or attributable normalization rejection;
- raw-frame admission/classification integrity failure; and
- connection lost/closed with source `reader`, `writer`, `heartbeat`,
  `protocol`, or `engine_close` and one bounded reason.

Adapter facts do not contain `ready`, `current`, `covered`, `subscribed`,
`recovering`, a committed watermark, or feature state. The engine decides
whether an ack establishes the aggregate handoff, whether a loss causes
recovery/suppression, and whether a stale epoch is fenced. Component 9 later
decides T/Q coverage from command and causal facts.

### 9.3 Component 2 extension seam

`C5-S2` extends Component 2 with statically typed connection/control admissions
and no public transition setter. The exact closed fact kinds are:

```text
connection_attempt
connection_established
authentication_result
aggregate_command_write_result
aggregate_subscription_result
tq_command_write_result
tq_subscription_result
connection_lost
ingress_integrity_failure
```

Each typed admission uses the existing Component 2 FIFO, required reserve for
aggregate/control, engine-assigned sequence, common binding/schema/context
validation, one disposition, accounting, lifecycle/commit evaluation, and
publication decision. A fact cannot call lifecycle or install `liveEpoch`
directly. Current-epoch aggregate acknowledgement is the only fact here that
can satisfy the Phase 1 handoff guard. A stale/wrong-epoch fact is fenced.

Normalized live aggregates continue through existing
`Engine.AdmitAggregate`; Component 5 does not create another aggregate type,
merge, state, or evaluator. Normalized T/Q values are provider-independent
adapter outputs with no Component 2 T/Q admission or canonical consequence
until Component 9 approves those direct seams. At the Component 5 boundary
they receive an explicit `consumer_deferred` disposition after classification;
this is an integration-stage fact, not claimed coverage or a silent drop.

### 9.4 One socket-attempt runner

One `open_aggregate_epoch` performs exactly:

```text
dial -> connected status -> auth write -> auth_success status
     -> subscribe A.* write -> success status -> ordinary frame read
```

Every read after dial, including handshake statuses, passes through the same
per-epoch frame-sequence allocator and strict classifier so control causal
positions are never invented outside the raw stream. The adapter may wait for
the engine disposition of the successful A.* acknowledgement before it emits
ordinary market facts as trusted post-ack data; bytes already read remain in
the bounded FIFO and preserve their positions.

Text and binary WebSocket messages are accepted only when their payload is
valid UTF-8 JSON under the classification contract. Other message types,
read-limit failure, non-UTF-8, socket read error, heartbeat failure, queue
saturation, oversized frame, receipt-time regression, or protocol ambiguity
ends that epoch with one terminal fact. It never silently resumes on the same
epoch.

The runner has reader, writer, and heartbeat work only while the epoch is
active. First terminal cause wins; cancellation joins all workers, closes the
socket under the explicit finite close duration, drains/fences every admitted
raw frame through the captured boundary, and emits one terminal epoch fact.
There is no goroutine detached from the epoch lifetime.

### 9.5 Command serialization and acknowledgements

Only one command is awaiting provider status per epoch. Handshake phases are
serial. After A.* is acknowledged, dynamic T/Q commands are serial and never
combined with A.*. A `change_tq` write uses exact deterministic comma-separated
pairs in the command's symbol order. JSON encoding is fixed to provider
`action` and `params`; arbitrary caller maps are unavailable.

Because v2 evidence shows generic provider `success` elements without an
echoed token, local correlation is valid only while exactly one command is
pending. A command is complete when the exact expected number of ordered
`success` elements has been classified for that epoch and command:

- `A.*`: exactly one;
- `T.SYM,Q.SYM`: exactly two per symbol; and
- a multi-symbol delta: exactly `2 * symbol_count`.

Status elements may share a provider array and each retains its causal
position. Wrong epoch, success before write completion, partial/extra count,
wrong phase, provider failure status, connection loss, or deadline produces a
failed/ambiguous fact. An acknowledgement may be observed before the local
write call returns; the adapter retains both bounded pieces and emits success
only after write success and exact acknowledgement evidence are both present.
This prevents the v2 race where either half alone could falsely commit.

### 9.6 Frame FIFO and hard bounds

There is one raw-frame FIFO between socket reads and sequential
classification. It uses explicit configured values with these contract
ceilings derived from v2:

```text
1 <= frame_slots <= 32,768
1 <= max_frame_bytes <= 8 MiB
max_frame_bytes <= total_frame_bytes <= 128 MiB
```

The adapter has no production default: Component 8 must select deployed values
at or below these ceilings from capacity evidence. Component 5 tests validate
the full bounds and use smaller deterministic configurations.

Admission copies bytes, captures UTC receipt once after `Read`, assigns the
next positive frame sequence for that epoch, and commits all queue accounting
atomically. Count/byte saturation or oversize rejects the just-read frame,
closes the epoch as an aggregate-ingress gap, and never pretends that only T/Q
was lost. An epoch terminal marker is required work, contains no raw bytes,
waits for one bounded slot while earlier admitted frames drain, and is ordered
after the greatest admitted frame of that epoch. Stop cancellation may fence
the marker only through the explicit terminal cleanup result.

There is no second classified queue. The sequential classifier emits direct
typed facts/admissions. Future Component 9 may make T/Q admission optional
through Component 2's existing required reserve rule, but it may not stop the
socket reader or skip whole-frame classification. In a future shed state, a
recognized T/Q item receives an explicit pressure-shed disposition at its
causal position; later aggregate/control items in the same array continue.

### 9.7 Reconnect boundary and injected operational durations

The adapter performs one attempt per engine command. It does not loop, sleep,
or randomize backoff. It allocates exactly one next process-local epoch for
that authorized attempt. Dial, handshake-step, whole-
handshake, heartbeat interval, heartbeat deadline, write deadline, and close
duration are explicit positive finite construction/command values with no
Component 5 production defaults. Component 8 owns their deployed values,
retry budget/backoff, exhaustion disposition, and shutdown policy.

A later engine-issued open command causes the adapter to allocate a strictly
greater positive epoch. Old worker results and frames keep the old epoch and
are fenced by Component 2.
No previous command acknowledgement or T/Q coverage crosses epochs. The new
epoch begins with A.* only; current ranking/TQ promotion resumes only through
the engine's ordinary lifecycle/evaluator evidence.

## 10. Required behavior

| Requirement | Behavior | Authority and evidence |
| --- | --- | --- |
| `C5-ENGINE-01` | Add only the closed connection/control facts above to Component 2's required FIFO. The engine validates binding/epoch/position, owns the active epoch and lifecycle edges, and fences stale facts. A current post-write A.* acknowledgement establishes the live handoff guard; no adapter method sets lifecycle/currentness/coverage. | `ARCH-OWN-01`–`04`, `DTE-CONTROL-01`, `LIFE-MODEL-01`–`02`, init/hydrate/recover transitions; Component 2 extension rule. |
| `C5-TRANSPORT-01` | Execute one context-bounded socket attempt through strict connected/auth/A.* phases, continuously read supported frames, heartbeat under injected durations, and return one terminal result while joining all workers. | `ARCH-FLOW-04`, `LIFE-INIT-05`, `LIFE-END-02`; v2 WebSocket worker and fake-server evidence. |
| `C5-COMMAND-01` | Serialize A.* and dynamic paired T/Q commands with one in flight, immutable tokens, exact write/ack interleaving, exact status count/order, and bounded failure/ambiguity facts. Intent/write alone never becomes acknowledgement or coverage. | `DTE-CONTROL-01`, `DTE-TQ-01`, `LIFE-HYDRATE-01`, `LIFE-TQ-02`; v2 paired-ack/race evidence. |
| `C5-BOUND-01` | Bound and account raw frame count/bytes, copy at admission, assign per-epoch frame order, drain every admitted frame or fence it explicitly, and treat raw saturation/oversize as epoch loss. Expose waiting raw frames/bytes/oldest age separately from the active frame and its diagnostic age. Preserve aggregate/control classification before optional T/Q shedding; after 500 ms of monotonic active-frame classification, shed only remaining T/Q elements while continuing the same ordered cursor. | `PG-TAQ-02`, `ARCH-FLOW-01`–`03`, `DTE-EVENT-02`, `DTE-TQ-03`, `LIFE-TQ-03`; direct owner evidence; v2 queue evidence. |
| `C5-RECONNECT-01` | End each epoch once with its first causal source/reason, drain/fence its boundary, clear command state, join workers, and allocate a greater epoch only for a later engine-issued open command. Adapter has no autonomous retry/lifecycle policy. | `LIFE-HYDRATE-07`, `LIFE-RECOVER-01`–`03`, `LIFE-RECOVER-06`, `LIFE-END-01`–`03`; v2 epoch/controlled-disconnect evidence. |
| `C5-LIVE-01` | Compose Component 1 binding, Component 2's sole FIFO/control/aggregate path, Component 3's installed evaluator, and the Component 5 adapter so an offline fake-provider A.* acknowledgement followed by an aggregate reaches the existing live canonical path in exact causal order. A mixed T/Q/control frame remains classified without creating Component 9 coverage or a second owner. | `DTE-MODEL-01`–`03`, `DTE-EVENT-02`, `DTE-MERGE-03`, `LIFE-HYDRATE-01`, `LIFE-HYDRATE-03`, `LIFE-LIVE-02`; Components 1–4 and v2 fake-provider evidence. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Engine command | Concrete closed command, exact binding/current epoch, positive unused token, legal action and bounded sorted symbols/durations. | Pre-I/O rejection; no socket/command state changes. | Stale desired membership closing or subscribing channels on a newer epoch. |
| Handshake/control stream | Expected status at exact phase and causal position, with write success and exact acknowledgement cardinality. | Wrong phase/count/status/epoch is failed or ambiguous; no active acknowledgement. | Generic `success` from a prior/unrelated command being treated as A.* handoff. |
| Raw-frame FIFO | Copy, count/byte capacity, nonregressing receipt, epoch-local sequence, and admission accounting commit together. | Saturation/oversize/regression closes epoch; admitted predecessors drain/fence. | A read frame omitted from both queue accounting and epoch-loss evidence. |
| Socket epoch | One adapter-allocated monotonic epoch per engine-authorized attempt, one first terminal cause, all workers joined, ordered terminal marker; engine consumption alone makes it active. | Late results remain old-epoch facts and fence. | Reconnect reusing old acknowledgement or accepting old reader bytes as new coverage. |
| Credential | Used only inside auth serialization and never observable outside private write buffer. | Fixed redacted error class; no raw provider prose. | Authentication failure returning/logging API key or request body. |

## 11. Failure and terminal behavior

Dial failure, handshake read/write failure, explicit auth/A.* rejection,
handshake ambiguity/deadline, ordinary read error, unsupported message type,
heartbeat failure, dynamic writer failure, frame saturation/oversize/clock
regression, strict decoder ingress ambiguity, close, and cancellation all have
closed facts. The adapter does not label any failure retriable, exhausted,
recovering, suppressed, ready, or current.

The first terminal cause wins for an epoch and later worker errors cannot
overwrite it. The terminal process is:

```text
capture cause and greatest admitted frame
  -> stop accepting new frames/commands
  -> cancel and join reader/writer/heartbeat
  -> drain or explicitly fence accepted frames through boundary
  -> bounded socket close
  -> emit exactly one terminal epoch fact
```

An explicit engine `close_epoch` uses the same path with an engine-close reason.
At session end or controlled stop, no external network acknowledgement is
required to let the engine terminate; the adapter reports the bounded close
outcome when available. A hung socket cannot keep the engine waiting past the
injected finite close bound.

Aggregate/control ambiguity or loss is global to current aggregate transport.
A fully classified T/Q status or item failure is T/Q-local. Writer failure on a
T/Q command makes that command/membership ambiguous but does not by itself say
aggregate frames were lost; shared socket termination does close aggregate
coverage and is reported separately.

## 12. Accounting and observability

This component owns no symbol population. Its exact transport identities are:

```text
connection_attempts
  = attempts_active
  + attempts_connected
  + attempts_failed
  + attempts_canceled

frames_read
  = frames_admitted
  + frames_rejected_oversize
  + frames_rejected_capacity
  + frames_rejected_receipt
  + frames_rejected_gate_or_close

frames_admitted
  = frames_queued
  + frames_classifying
  + frames_dispositioned
  + frames_fenced

commands_started
  = commands_pending_write
  + commands_pending_ack
  + commands_acknowledged
  + commands_failed
  + commands_ambiguous
  + commands_canceled_or_fenced
```

Each epoch has exactly one terminal outcome once started. `attempts_connected`
is a connection fact, not a backend-ready or current-ranking count. Frame bytes
and count are primary queue dimensions; event family, failure source, command
kind, and status reason are bounded overlapping diagnostics. No symbol, URL,
provider message, credential fragment, arbitrary error, or command parameter
string is a metric label or retained log field.

## 13. Simplicity and boundedness

- Mutable adapter owners: one active epoch runner, one raw-frame FIFO, one
  per-epoch frame counter, and one pending command record. Each is necessary
  for real I/O, causal order, or generic provider ack correlation.
- Scanner truth remains one Component 2 graph. There is no HealthStore,
  owner/orchestrator state, adapter readiness, recovery generation, T/Q
  membership map, evaluator, or publisher.
- One private connection interface isolates the one approved WebSocket library
  for fake tests. It is not a generic transport framework.
- Raw retained data is at most configured `total_frame_bytes <= 128 MiB`, plus
  32,768 bounded frame descriptors and one bounded pending command. No frame,
  command, result, reason, or diagnostic history is retained after terminal
  disposition beyond fixed latest/counters owned by the engine/operations
  projection.
- Retries/backoff/hysteresis/currentness are absent rather than guessed; their
  policy owner is Component 8. T/Q coverage/features/pressure policy are absent
  rather than partially duplicated; their owner is Component 9.

This is the minimum complete live I/O path. A second socket, classified-event
queue, generic bus, callback registry, database/journal, service split, or
autonomous reconnect state machine would add owners without an approved need.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Authentication failure and provider prose containing secret/URL | V2 authentication/privacy tests | One redacted failed-auth fact; no retry classification or credential/prose retention. | `P-C5-TRANSPORT` |
| Out-of-phase or wrong-count success | V2 handshake/status tests | Command remains unacknowledged and attempt/command terminates failed or ambiguous. | `P-C5-COMMAND` |
| Ack arrives before write returns | V2 Phase 3 paired-ack race | Retain bounded ack evidence but report success only after write success also arrives. | `P-C5-COMMAND` |
| Partial/extra/wrong-epoch paired T/Q ack | V2 correction/provider-gate tests | Never establish completed command; return exact ambiguous/fenced result. | `P-C5-COMMAND` |
| Same raw frame contains failed T/Q status and valid aggregate | V2 `TestPhase3StatusFailurePreservesAggregateAndSnapshotPrivacy` | Emit both ordered facts; T/Q failure does not discard the aggregate. | `P-C5-LIVE` |
| Count/byte saturation and oversized frame | V2 queue tests/config ceilings; `ARCH-FLOW-02` | Reject atomically, close epoch as gap, and drain all earlier admitted frames. | `P-C5-BOUND` |
| End marker while queue full | V2 `TestEndMarkerWaitsForCapacityAndOrdersEpochs` | Wait cancellation-aware for bounded slot and order marker after earlier frames. | `P-C5-RECONNECT` |
| Disconnect then greater epoch | V2 consecutive-epoch and controlled-disconnect tests | One old-epoch terminal fact/marker; new reader is distinct; no old ack/coverage crosses. | `P-C5-RECONNECT` |
| Heartbeat operation cannot distinguish ping-write from pong-wait | V2 combined-heartbeat evidence | Report bounded `heartbeat_failure_unclassified`; do not invent a more precise cause. | `P-C5-RECONNECT` |
| Shutdown with accepted frames | V2 owner drain test plus Component 2 FIFO invariant | Drain/fence accepted adapter frames and engine admissions; join workers within injected bounds. | `P-C5-BOUND`, `P-C5-RECONNECT` |

The proof allocations and complete slice plan are authoritative in
[proof and delivery plan](proof-and-delivery-plan.md).
