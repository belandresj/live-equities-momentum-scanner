# Single-pass live ingress

**Status:** Owner-approved focused replacement specification, 2026-08-23. The
[delivery program](delivery-program.md) is the sole mutable status ledger.

**Parent:** [Live backend replacement architecture](../live-backend-replacement.md).

## 1. Outcome and authority

This contract decodes each Massive WebSocket frame once, hands one immutable
decoded batch to the engine through one bounded FIFO, and retains the accepted
connection/heartbeat/retry semantics. It owns socket attempts, wire reads and
writes, provider-frame positions, normalization, the decoded-batch FIFO,
causal fence-marker production, pressure observations, and transport
terminals. It owns no market state, coverage consequence, ranking, watermark,
readiness, or publication.

Allocated parent requirements are `LBR-ARCH-03` and `LBR-ARCH-10`.
The integration contract owns cross-cutting `LBR-ARCH-02`/`11` conformance;
this contract supplies their ingress concurrency and transport-containment
obligations. No parent-routed product requirement is allocated here; ingress
faithfully transports the product facts owned by the other focused specs.

The compatible historical semantics routed here are `DTE-MODEL-01` through
`DTE-MODEL-03`, `DTE-SESSION-03`, `DTE-SESSION-04`, `DTE-CLOCK-03`,
`DTE-EVENT-01`, `DTE-EVENT-02`, `DTE-CONTROL-01`, and `DTE-REJECT-01` from the
[data/time/event contract](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/architecture/data-time-and-event-contract.md).
Connection-attempt behavior routes `LIFE-RECOVER-01` from the
[engine lifecycle](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/architecture/scanner-state-engine-lifecycle.md) and the
inbound-aware retry facts from
[`live-aggregate-heartbeat-and-resubscription-correction.md`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/live-aggregate-heartbeat-and-resubscription-correction.md).
The data-confirmed dynamic T/Q status behavior is routed from
[`tq-data-confirmed-subscription-correction.md`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/tq-data-confirmed-subscription-correction.md).

## 2. Boundary and dependencies

This capability consumes the accepted normalized aggregate/fence seam from
[`canonical-state-and-hydration.md`](canonical-state-and-hydration.md), the
timer/publication owner from
[`evaluation-and-publication.md`](evaluation-and-publication.md), and T/Q
commands/admissions/pressure meanings from [`tq-state.md`](tq-state.md).
These specifications define engine inputs first; ingress implements their
final producer/executor interface and cannot reinterpret them.

| Input from engine/runtime | Ingress action |
| --- | --- |
| open/retire attempt command | Dial, authenticate, establish aggregate subscription, or close exactly one epoch. |
| typed T/Q command | Serialize one bounded provider write; return exact write result and read boundary `B`. |
| fence request | Append a marker after the greatest complete frame already read for the current epoch. |
| pressure mode/sample request | Apply early T/Q normalization policy or return one fixed queue/accounting sample. |
| cancellation/session terminal | Stop reads/writes, emit the exact terminal, and join owned work. |

The output is one FIFO stream of immutable `DecodedBatch` values and marker/
terminal entries. The engine consumes each entry serially and assigns its own
mutation sequence. Hydration results and engine timers remain the parent-
allowed owner-local paths; they do not share or compete for provider ordering.

Reuse evidence is limited to the current repository's
[`provider normalization`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/specifications/massive-live-adapter/provider-classification-and-normalization.md),
[`transport/epoch`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/specifications/massive-live-adapter/transport-commands-and-epochs.md),
heartbeat, and data-confirmed corrections cited above, plus their directly
named provider fixtures. Wire shapes, classifications, and recovery
counterexamples are retained; the double parse, raw queue, intermediate
envelopes, old delivery ledger, and private implementation are rejected. No
predecessor checkout, credential, or live provider request is authorized.

## 3. One-pass frame model

One attempt-local reader/decoder goroutine reads a complete provider frame,
captures one UTC receipt time and monotonically increasing frame sequence,
streams its JSON array once, and constructs one immutable batch. It never
retains raw bytes after successful admission or performs a pre-scan/full
unmarshal followed by per-event unmarshal.

Every batch contains the binding/epoch, frame sequence, receipt time, encoded
byte charge, ordered event/drop/control facts, and a terminal ambiguity when
applicable. Each event carries its zero-based array index. The immutable causal
position is `(connection_epoch, frame_sequence, array_index)`; wire timestamps
never impersonate ordering.

The decoder accepts unknown additive members, rejects a duplicate recognized
member within one event, and validates the currently documented shapes for
`A`, `T`, `Q`, and status/control. A clearly identified unsupported event
family is a bounded drop. It does not require a second representation to
decide the family.

Failure is localized at the earliest trustworthy boundary:

- malformed `A` with trustworthy symbol/window identity emits a local invalid
  aggregate fact; without usable identity it is aggregate ambiguity;
- malformed `T` or `Q` with trustworthy family and symbol emits a local T/Q
  rejection/drop; known T/Q family with no trustworthy symbol closes T/Q
  trust globally for the epoch but not aggregate currentness;
- malformed/unknown additive status after the established aggregate handshake
  is bounded control/T/Q diagnostic unless it can conceal aggregate loss;
- a missing, malformed, or duplicate event discriminator, truncated array, or
  element that could conceal aggregate/control work emits ingress ambiguity;
  earlier complete elements remain ordered before the terminal and later
  bytes are not silently accepted; and
- unknown additive JSON members and clearly identified unsupported families
  never trigger recovery by themselves.

Normalization preserves the exact aggregate, trade, quote, time, condition,
structural validity, and source-position fields required by downstream
contracts. Provider prose is reduced to closed classes and bounded diagnostics;
it never becomes retained product state.

## 4. One FIFO and saturation policy

The initial decoded-batch ring has 4,096 entries and a 64 MiB total retained
charge, including bounded decoded strings/slices. A source frame is at most
8 MiB, a batch at most 65,536 elements and 32 MiB retained charge, and eight
entry/64 KiB capacity is reserved for fence/control/terminal markers. The ring
stores values or sole-owned immutable backing arrays; enqueue transfers
ownership and dequeue releases it. Constructors reject nonpositive or
incoherent bounds.

These queue values are program-revisable before Capability D from the frozen
characterization, but a revision must keep finite slot/per-frame/batch/byte
limits, marker reserve, and the failure behavior below. The parent resource
targets remain diagnostic rather than independent gates.

There is no second general engine FIFO for these events. Dequeue calls the
owner loop synchronously for the complete batch, so frame/array order and
terminal position are linearized once. Engine completions used only for
diagnostics cannot accumulate another market-event queue.

When T/Q pressure is active or a frame exceeds its 500 ms classification
budget, the decoder still classifies every remaining element but omits
expensive T/Q normalization and emits ordered T/Q shed facts. Aggregate and
control work is preserved. If a T/Q-only batch cannot fit, every affected T/Q
channel is closed and the drop is accounted. If any aggregate/control-bearing
batch or required fence/terminal cannot be admitted within the reserved
construction bound, the attempt closes, currentness closes, and exact gap
recovery is required. No retry path treats the lost batch as successfully
consumed.

Pressure sampling reads ring count/bytes/capacities, oldest queued receipt,
capacity drops, and closed accounting scalars directly; it adds no sampler
goroutine and owns no pressure decision.

## 5. Handshake, commands, and fence production

Only one socket attempt exists at a time. It validates connection and
authentication statuses, writes the persistent aggregate subscription, and
delivers the documented aggregate acknowledgement position before live
coverage can start. Post-handshake generic `success` is informational and is
never correlated with dynamic T/Q commands. Authentication failure and a
documented connection-fatal error terminate the attempt.

Socket writes are serialized through one attempt-local write critical section;
there is no command queue. At most one dynamic T/Q command is in flight.
Successful subscribe write completion captures `B`, the greatest frame
sequence whose read has completed at that instant, and returns `B` with the
engine's private command identity. Failed writes carry no `B`. Unsubscribe
write success proves only local write completion.

A fence request captures the greatest complete frame sequence already read and
appends one marker after all admitted batches through that sequence. Marker
identity includes binding, epoch, hydration generation, command token, and
ordinal. It cannot be forged by the hydrator. Queue closure, attempt retirement,
or epoch replacement fences any marker not consumed by the engine.

## 6. Heartbeat, recovery, and shutdown

One heartbeat operation may be active for the socket, serialized with other
writes. An attempted heartbeat records the inbound frame sequence at issue.
A heartbeat write failure followed by supported inbound progress before the
five-second deadline is diagnostic and nonterminal. A read failure, an
independent transport-fatal heartbeat result, or deadline expiry without
inbound progress retires the attempt and closes currentness/T/Q coverage.

Attempt retirement is joined before another dial begins. The process-start dial
is immediate and is not a recovery ordinal. After a failed process-start dial
or any lost epoch, recovery attempts 1 through 5 wait exactly 1, 2, 4, 8, and
16 seconds respectively. Failure or loss of recovery attempt 5 produces stable
exhaustion with no recovery attempt 6. Thus a never-established process may
perform one immediate dial plus at most five delayed recovery dials.
Only engine acceptance of a successfully reconciled hydration/recovery fence
resets the budget. Data receipt, handshake, command write, or partial hydration
does not.

The reader, heartbeat operation, and any pending write all terminate under the
attempt context. Retirement emits one terminal, drains or fences already
admitted entries according to their exact causal position, closes the socket,
and joins all attempt-owned work before returning. Process cancellation and
session end are bounded terminal paths; they cannot leave a pending command,
marker, retry timer, or socket goroutine.

## 7. Accounting and trust boundaries

Frame accounting reconciles complete reads into enqueued batches, safe T/Q-
only drops, or epoch-closing capacity/integrity terminals. Per-family decoder
accounting reconciles recognized elements into normalized, local reject,
unsupported drop, pressure shed, or ambiguity. Queue count/byte/current/high/
drop counters reconcile independently. Command writes and terminals reconcile
with the engine's private tokens.

Invalid states prevented by construction include two live attempts, two
readers, concurrent dynamic commands, mutable batch aliases, two live ordering
queues, and a fence inserted ahead of already-read admitted frames. Runtime
validation rejects stale epoch commands, frame/array regression, invalid
provider values, oversize frames/batches, write-result/token mismatch, marker
mismatch, and terminal duplication.

The smallest false success is silent aggregate/control loss followed by a
current fence or retry reset. Therefore aggregate ambiguity/capacity failure
always closes the epoch, its terminal is ordered or delivered through reserved
capacity, and only the engine's later exact hydration fence restores
currentness and retry budget.

## 8. Primary proofs and slices

| Slice | Primary proof | Claim, dangerous counterexample, observable distinction, limitation |
| --- | --- | --- | --- |
| `LBR-D1` | `P-LBR-D1-DECODE` | The approved provider fixture corpus passes once through the streaming decoder and the frozen semantic oracle. It covers mixed arrays, exact positions/receipt, every aggregate/T/Q/control/drop family, unknown additive members, duplicate recognized fields, malformed localizable elements, unsupported families, earliest ambiguity, oversize input, and status semantics. It compares normalized facts/dispositions/accounting, then records decoder allocations/throughput separately from engine work and asserts no second JSON parse/raw retention. It does not prove FIFO saturation, socket timing, or provider behavior beyond the fixtures. |
| `LBR-D2` | `P-LBR-D2-HANDOFF` | A real bounded ring, engine consumer, fake socket, and deterministic clock compose mixed-frame ordering, marker placement, T/Q-first shedding, T/Q-only/full mixed saturation, aggregate/control overflow terminal plus exact recovery, subscribe boundary `B`, handshake failure, heartbeat failure with progress/no progress, read failure, epoch fencing, one immediate process-start dial followed by exact 1/2/4/8/16-second recovery attempts 1 through 5, exhaustion with no recovery attempt 6, cancellation, and joined shutdown. It asserts one queue, zero silent aggregate/control loss, exact accounting/currentness, and retry reset only after accepted recovery fence. It is not credentialed provider or final resource evidence. |

`LBR-D1` installs the single streaming decoder and immutable batch shape behind
a temporary adapter into the accepted engine input seams. Acceptance makes the
double-pass logic in `internal/massive/live_normalization.go`, per-element raw
JSON copies, and intermediate normalization envelopes removable.

`LBR-D2` installs the decoded-batch ring directly into the owner loop and
removes the temporary adapter. Acceptance makes the raw-frame queue in
`internal/massive/live_queue.go`, adapter-delivery staging in
`internal/massive/live_transport.go`, the general engine live-input FIFO and
its completion backlog, and duplicate causal-marker paths removable. Retained
socket/retry code may be reshaped but not duplicated.

## 9. Verification, review, and discretion

Each slice runs its primary proof, affected short/race tests, focused vet,
`git diff --check`, and ordinary repository verification at the gate. The
Capability D review focuses on decoder ambiguity, FIFO/marker linearization,
socket/write concurrency, capacity false success, retry reset, and complete
joins. A narrow review is required when queue saturation, marker reserve, or
attempt retirement still permits representable aggregate loss with a current
claim.

The implementer may choose streaming-token helpers, immutable batch layout,
ring indexes, string interning with exact bounds, socket wrapper, and closed
diagnostic enums. The implementer may revise count/byte division through the
delivery program from measured characterization. The implementer may not add
another queue/reader/state owner, silently drop aggregate/control, correlate
undocumented statuses, change retry/heartbeat semantics, or introduce a
generic bus/framework.

Market formulas, canonical merge, selection, T/Q field rules, API/UI changes,
provider access, public deployment, replay/checkpoint work, and final runtime
cutover are non-scope. Slice evidence and status are recorded only in the
delivery program.
