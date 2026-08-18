# T/Q data-confirmed subscription correction

**Status:** implementation-ready specification; no implementation is authorized by this document alone

**Decision source:** owner direction on 2026-08-18, supported by Massive's
current WebSocket documentation and first-party client behavior recorded in
[`massive-websocket-subscription-ack-research.md`](massive-websocket-subscription-ack-research.md).

## 1. Outcome

Dynamic stock trade and quote subscriptions must no longer depend on receiving
one generic Massive `success` status per requested `T.SYMBOL` and `Q.SYMBOL`
within five seconds. Massive does not document that acknowledgement count or
deadline, and its first-party clients send subscription commands without
waiting for such evidence.

The scanner will instead separate three facts:

1. **requested** — the complete subscription command was written successfully;
2. **trade confirmed** — a structurally valid post-request `T.SYMBOL` event was
   received for the current symbol generation and connection epoch; and
3. **quote confirmed** — a structurally valid post-request `Q.SYMBOL` event was
   received for the current symbol generation and connection epoch.

Tape Speed and Spread remain independent. Missing `success` statuses, a quiet
symbol, or the absence of one channel must never stop aggregate processing,
ranking, watermark advancement, the snapshot API, or the scanner process.

This correction applies only to dynamic `T.*` and `Q.*` membership. It does not
change connection establishment, authentication, the initial `A.*` aggregate
subscription handoff, aggregate recovery, ranking, or the Tape/Spread formulas.

## 2. Provider evidence and resulting boundary

Massive documents explicit `connected` and `auth_success` responses. Its
subscription examples send a comma-separated command and proceed directly to
data messages. The public contract does not define a per-topic subscription
acknowledgement count, correlation identifier, delivery deadline, or retry rule.
Trade and quote messages do carry both an event family and `sym`.

Massive's Go and Python clients treat subscription as a local requested-set and
write operation. They do not wait for generic `success` statuses. The Go client
logs post-authentication `success` and `error` statuses without correlating them
to requested topics.

Therefore:

- a generic `success` is diagnostic information, not coverage evidence;
- absence of `success` is not failure evidence;
- a successful local write is request evidence, not data-delivery evidence;
- the first matching post-request event is positive evidence for that channel;
- silence is not negative evidence; and
- provider membership cannot be represented as known merely because a command
  was written.

No public provider evidence distinguishes a silently ignored subscription from
a legitimately quiet trade or quote channel. The scanner must expose that
uncertainty instead of inventing an acknowledgement guarantee.

## 3. Required state model

The engine remains the sole owner of desired membership, per-symbol request
generation, channel confirmation, feature coverage, and publication state. The
Massive adapter owns command serialization, socket writes, frame positions, and
provider-status classification.

Each selected symbol has one monotonically increasing local request generation
within the active connection epoch. Each channel has one of these states:

| State | Meaning | Coverage |
| --- | --- | --- |
| `not_requested` | The symbol is not desired or the request has not been written. | Closed |
| `requested_unconfirmed` | The paired T/Q command was written, but no qualifying post-request event has confirmed this channel. | Closed |
| `confirmed` | A structurally valid matching post-request event has been received for this epoch and generation. | Open from that event's causal position and receipt time |
| `closed` | Rank removal, pressure shedding, explicit T/Q control error, connection loss, epoch change, session end, or retained-state failure closed the channel. | Closed |

Trade and quote states transition independently even though the wire command is
paired.

### 3.1 Request boundary

For every successful subscription write, the adapter captures `B`, the greatest
complete WebSocket frame sequence already read immediately after the write
returns. The write-result fact delivered to the engine contains the binding,
epoch, command token, symbol generation(s), receipt time, and `B`.

Only a matching event whose frame sequence is greater than `B` may confirm the
new channel generation. The entire frame containing `B` is excluded; array
index ordering inside a frame must not be used to pretend that part of a frame
was read after the write.

A failed write carries no request boundary and opens no channel. At most one
wire write may be in flight so subscription and unsubscription order remains
deterministic. The next command may be written as soon as the prior write result
has been admitted; no provider-status wait exists.

### 3.2 Data confirmation

A normalized trade confirms the trade channel when all of the following hold:

- binding, trading date, connection epoch, symbol, and request generation match;
- the symbol is still desired;
- the paired request write succeeded;
- the event frame sequence is strictly greater than the request boundary; and
- the event is structurally valid enough to identify it as a real trade for the
  symbol.

Trade-condition eligibility, lifecycle eligibility, and duplicate handling are
separate feature decisions. A structurally valid but non-volume-updating trade
may confirm delivery without contributing to Tape Speed.

A normalized quote confirms the quote channel under the corresponding rules.
A one-sided, crossed, or known-special quote may confirm delivery even though
the Spread result is unavailable or invalid under its existing feature rules.

The confirming event is admitted through the ordinary feature path. Its causal
position becomes the initial greatest position for the channel and its receipt
time, clamped to the active session, becomes coverage start.

Pre-boundary, wrong-generation, stale-epoch, unselected-symbol, and already
closed events are fenced or counted as unexpected wire traffic. They cannot
reopen coverage or mutate retained feature state.

## 4. Feature behavior

### 4.1 Tape Speed

Tape coverage begins with the first confirmed trade event, not with the command
write and not with a generic status.

- Before confirmation: `unavailable / channel_unconfirmed`.
- For less than one second of confirmed coverage: both Tape windows are
  `warming / coverage_warming`.
- From one second through less than five seconds: the one-second field follows
  the existing current calculation; the five-second field remains warming.
- After five seconds of continuous confirmed coverage: both existing windows
  are valid, and a window containing no qualifying trades is numeric zero.

This intentionally refuses to label a never-observed trade channel as a
trustworthy zero. Rank removal, pressure shedding, connection loss, epoch
change, explicit T/Q control failure, or retained-state failure clears coverage
and retained Tape state. Resubscription always warms from a new generation.

### 4.2 Spread

Before quote confirmation, Spread is
`unavailable / channel_unconfirmed`. The first confirmed quote is evaluated
immediately:

- a positive two-sided quote with `ask >= bid` produces the existing numeric
  Spread result;
- a one-sided quote produces the existing unavailable result;
- a crossed quote produces the existing invalid result; and
- later quiet time uses the existing quote-age and stale rules.

Rank removal, pressure shedding, connection loss, epoch change, explicit T/Q
control failure, or retained-state failure clears quote coverage and retained
quote state.

## 5. Membership changes

### 5.1 Initial selected set

After a qualified-current ranking exists on a fresh epoch, send one paired
batch for the complete displayed set, up to 20 symbols. A successful write moves
every included channel to `requested_unconfirmed`. Symbols and channels confirm
independently as data arrives. No status count or deadline is installed.

### 5.2 Rank churn

Removal still precedes addition.

When a symbol leaves the displayed set:

1. close both local channel coverages immediately;
2. clear its retained Tape and Spread state;
3. invalidate its request generation;
4. send the paired unsubscribe command; and
5. ignore and count any later T/Q events for that unselected generation.

Unsubscribe write success means only that removal was requested. The scanner
must not claim proven provider absence. A failed unsubscribe is T/Q-local and
cannot affect aggregate ranking.

When a symbol later returns, allocate a new generation, clear any remaining
feature state, write a fresh paired subscribe, and require new post-boundary
confirmation. Events from an earlier generation cannot confirm the new one.

Unexpected T/Q traffic for unselected symbols is an observable wire-membership
drift counter. It remains locally ignored. If that traffic causes measured
queue or retention pressure, the existing T/Q pressure policy may shed T/Q or
enter aggregate-only while preserving aggregates. This specification does not
invent a silence-based retry or an undocumented repeated-unsubscribe policy.

### 5.3 Reconnect

A connection loss or greater epoch closes all T/Q request generations,
confirmations, coverage, and retained features. After the aggregate connection
and ranking are valid again, send one fresh paired batch for the current
displayed set and reconfirm every channel from data. No T/Q membership or
coverage survives an epoch.

## 6. Provider-status behavior

Status handling after the aggregate handshake is:

| Provider fact | Required behavior |
| --- | --- |
| `success` | Count as informational and discard. Do not correlate it, complete a command, open coverage, extend a deadline, or alter membership. |
| `error` | Emit one bounded T/Q-control error. Close T/Q coverage and prevent further T/Q additions for that epoch; keep `A.*`, ranking, watermark, API, and process live. Do not parse provider prose into product state. |
| late, duplicate, or unsolicited `success` | Same informational handling as any other `success`. It cannot contaminate the next command. |
| valid but unknown post-authentication status | Count as unsupported status and contain it to T/Q/control observability unless separate evidence proves aggregate transport loss. |
| `auth_failed` | Preserve the existing connection-fatal behavior. |
| malformed raw event identity or actual frame loss | Preserve the existing aggregate-integrity/recovery rules because the lost element could have been an aggregate. |

There is no T/Q status timer, expected-status count, partial-success state,
status-deadline quarantine, or status-based cleanup liability.

## 7. Failure and retry policy

- **No matching data:** remain `requested_unconfirmed`; do not retry and do not
  infer failure.
- **T/Q command write failure:** close affected T/Q state and enter the existing
  T/Q-local containment path. Continue consuming aggregates unless independent
  transport evidence closes the socket or proves aggregate loss.
- **Explicit post-authentication provider error:** contain all T/Q for the epoch
  because the error has no documented topic or request identifier. Do not stop
  aggregates.
- **Socket loss:** use the existing aggregate gap-recovery/reconnect path, then
  resubscribe the current desired set on the new epoch.
- **T/Q queue or retention pressure:** preserve the existing shed/degrade/
  aggregate-only policy and close affected coverage honestly.
- **Silence:** never drives retry, quarantine, reconnect, or aggregate failure.

This correction adds no periodic subscription retry. Repeated subscribe or
unsubscribe idempotency is not documented, and silence cannot identify which
channel needs repair. A future retry policy requires provider evidence or an
owner-approved operational assumption.

## 8. Observable and API semantics

The snapshot must not describe a successfully written command as provider-
acknowledged membership.

At minimum, revise the existing representation as follows:

- `provider_present=true` only when both trade and quote channels have been
  confirmed by post-request data for the current generation;
- `provider_membership_unknown=true` when the symbol is requested but either
  channel remains unconfirmed, or when an explicit T/Q control error makes the
  requested wire state uncertain;
- `trade_coverage` and `quote_coverage` continue to state the independent
  confirmed coverage facts;
- `known_present` counts symbols whose two channels are data-confirmed;
- `unknown` counts requested symbols that are not confirmed on both channels or
  whose control state is uncertain; and
- `known_absent` counts symbols with no active local request, not a proved
  provider-side unsubscribe result.

Rename T/Q command accounting from acknowledgement language to write language:

```text
issued = pending_write + written + failed + fenced
```

The API field `commands.acknowledged` must not continue carrying a changed
meaning. Replace it with `commands.written` in the strict v2 schema and update
the UI validator/fixtures in the same slice. Add bounded counters for
informational success statuses, provider error statuses, and unexpected T/Q
events for unselected generations if they are not already represented.

Add the field reason `channel_unconfirmed` to the engine, snapshot API, and UI
closed vocabularies. The UI should display Tape and Spread as unavailable while
their respective channel is unconfirmed; it must not describe the whole symbol
as failed when only one channel lacks evidence.

## 9. Implementation boundaries

Expected code changes are limited to:

- `internal/massive/live_transport.go`: remove T/Q expected-status accumulation
  and deadline; emit successful-write boundary facts; keep one write in flight;
- `internal/massive/live_normalization.go`: make post-handshake success
  informational, produce bounded T/Q error facts, and preserve handshake rules;
- `internal/engine/tq.go`: replace acknowledgement-created membership/coverage
  with request generations and per-channel data confirmation;
- `internal/engine/connection_control.go`: revise the T/Q write-result shape
  without changing aggregate control semantics;
- `internal/operations/runtime.go`: advance command sequencing after admitted
  write results and preserve aggregate/TQ failure separation;
- `internal/snapshotapi/schema.go` and `mapper.go`: expose honest membership,
  coverage, reasons, and write accounting;
- `ui/model.js` and fixtures/tests: accept and display the corrected strict v2
  semantics; and
- the controlling T/Q sections of
  `docs/architecture/data-time-and-event-contract.md`,
  `docs/specifications/top-20-tq-coverage-and-features.md`, and
  `docs/live-tq-resilience-correction.md` so they no longer require generic
  status acknowledgement.

Do not change aggregate ranking, feature formulas, session time, hydration,
aggregate subscription establishment, queue thresholds, checkpoint behavior,
or dashboard layout as part of this correction.

## 10. Required deterministic proofs

### P1 — no-status production-path subscription

Drive the real adapter/runtime/engine/snapshot path through authentication and
aggregate establishment, then send a paired T/Q subscription batch and no
subscription success statuses.

Prove:

- the command becomes written with no pending status deadline;
- all symbols begin requested and channel-unconfirmed;
- matching T and Q events confirm only their symbol and channel;
- a valid non-qualifying trade can confirm trade delivery without increasing
  Tape Speed;
- after five seconds of confirmed trade coverage, covered silence is numeric
  zero;
- the first valid quote produces Spread immediately;
- ranking remains `qualified_current`, the aggregate watermark advances, the
  snapshot API returns HTTP 200, and readiness remains honest.

### P2 — independent and quiet channels

For at least three selected symbols, prove these simultaneous outcomes:

- both channels confirmed: Tape and Spread follow ordinary behavior;
- quote confirmed, trade unconfirmed: Spread works while Tape reports
  `channel_unconfirmed`; and
- trade confirmed, quote unconfirmed: Tape warms/current while Spread reports
  `channel_unconfirmed`.

Advancing time without data must not convert either unconfirmed channel to
failed, trigger a retry, or affect aggregate ranking.

### P3 — late statuses and explicit error

After multiple sequential membership writes, deliver delayed, split,
duplicate, and extra generic successes. Prove they are informational only and
cannot confirm channels, finish another command, or quarantine T/Q.

Then deliver an explicit post-authentication `error`. Prove all T/Q fields close
with an exact control-error reason while the aggregate-only control produces
the same ranking rows, order, qualification, watermark, and HTTP/API liveness.

### P4 — rank churn and generation fencing

Confirm both channels for one selected symbol, remove it, deliver later T/Q
events, then select it again.

Prove:

- removal closes and clears both features before the unsubscribe write;
- post-removal events are counted and ignored;
- the new selection uses a greater generation and new write boundary;
- prior-generation events cannot confirm or populate the new generation; and
- new post-boundary events confirm normally.

### P5 — reconnect and aggregate independence

Confirm a mixed selected set, lose the socket, execute the existing exact
aggregate recovery, and establish a new epoch.

Prove no T/Q state survives, the current displayed set is requested in one
fresh batch, every channel reconfirms independently, and aggregate results are
identical to an A-only control throughout every T/Q-only variant.

### P6 — API and accounting

Prove the strict snapshot schema and UI accept:

- requested-but-unconfirmed membership;
- one confirmed channel;
- both confirmed channels;
- explicit T/Q control containment; and
- the command identity `issued = pending_write + written + failed + fenced`.

Reject false provider-present claims, coverage without a matching generation
and boundary, acknowledgement terminology in the new command schema, and any
T/Q state that gates aggregate readiness.

## 11. Implementation order and acceptance

Implement in two sequential slices:

1. **Adapter/control slice:** remove T/Q status gating, add the successful-write
   frame boundary, make successes informational, contain explicit errors, and
   prove command ordering plus late-status behavior.
2. **Engine/publication slice:** add request generations and per-channel data
   confirmation, revise coverage/features/API/UI/accounting, and run P1-P6.

Run the narrowest focused tests during correction, followed by:

```text
go test -short -timeout 2m ./...
go test -race -timeout 5m ./internal/massive ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner
go vet ./...
git diff --check
```

The correction is complete only when:

- no T/Q path contains an expected generic-success count or status deadline;
- no generic success creates membership or coverage;
- every published T/Q field follows independent data-confirmed coverage;
- quiet unconfirmed channels remain honest without retries;
- late statuses cannot contaminate later commands;
- rank churn and reconnect cannot reuse old feature state;
- explicit T/Q failure never changes aggregate results or process/API liveness;
- the API contains no false acknowledgement or provider-membership claim; and
- all required proofs and ordinary verification pass.

## 12. Deferred question

The scanner cannot prove a genuinely quiet trade channel is subscribed without
a documented acknowledgement or an observed trade. This specification chooses
honest unavailability until data confirmation. If the product later requires a
numeric Tape zero before any trade is observed, that is a deliberate weaker
assumption—`successful write implies continuous delivery`—and must be approved
and documented separately rather than introduced as an implementation detail.
