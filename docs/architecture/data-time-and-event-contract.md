# Data, time, and event contract

**Status:** Approved architecture contract.

**Prepared:** 2026-08-05

**Approved:** 2026-08-05

**Scope:** Repository-wide market-data meaning, clocks, interval boundaries,
normalized event shapes, causal ordering, reconciliation, committed time,
replay time, and checkpoint cutoffs. Exact provider wire decoding, capacity
limits, retry policy, storage encoding, and feature formulas belong in focused
specifications.

## 1. Authority and purpose

[`../product/product-goals.md`](../product/product-goals.md) is the product-level
authority. [`system-overview.md`](system-overview.md) defines ownership and
topology. This document makes their data and time semantics precise. A
component specification may add implementation detail but may not invent a
different clock, event identity, interval convention, or definition of
coverage.

The contract has one central rule:

> Every published fact must identify the market interval or causal boundary
> through which it is true. Receipt, recovery, checkpoint, and T/Q progress may
> describe their own work, but none of them may silently become the ranking
> clock.

Normative requirements use `DTE-*` identifiers. Examples are explanatory and
do not weaken those requirements.

## 2. Foundational model

### DTE-MODEL-01 — one canonical event path

Live WebSocket data, historical REST data, and offline replay data are decoded
by source-specific adapters and converted into provider-independent normalized
events. They then enter the same ordered `ScannerStateEngine` acceptance and
state-transition path.

An adapter may describe how a fact arrived. It may not create a second meaning
for price, volume, event time, aggregate identity, qualification, or feature
state.

```text
provider payload or replay record
  -> decoded source record
  -> normalized event
  -> admitted engine input
  -> canonical acceptance or explicit rejection
  -> state transition
  -> evaluation at committed T
  -> immutable snapshot
```

### DTE-MODEL-02 — stages are distinct

The implementation must not use the word “accepted” for every stage:

1. **Decoded** means the source payload was parsed.
2. **Normalized** means required fields were converted into a valid canonical
   type and basic structural invariants passed.
3. **Admitted** means the bounded engine input accepted ownership of the item.
4. **Canonically accepted** means the engine validated the active session,
   generation or connection epoch, interval, causal position, and merge rules,
   then applied the market fact.
5. **Published** means a later immutable snapshot exposes state derived from
   that accepted fact at its committed watermark.

A decoded WebSocket frame can therefore be accepted by the transport while an
individual event inside it is rejected by the engine. Rejection must have a
bounded reason and must never be represented as successful market-data
acceptance.

### DTE-MODEL-03 — canonical values and delivery evidence

Market values and delivery evidence are separate:

- canonical values include symbol, market interval, prices, volume, trade
  fields, or quote fields;
- delivery evidence includes source, connection epoch, request generation,
  causal position, receipt time, and diagnostics.

Delivery evidence does not change aggregate identity. It is retained where
needed for fencing, conflict resolution, latency measurement, and diagnostics.

## 3. Session calendar and time representation

### DTE-SESSION-01 — trading date

`trading_date` is the exchange-local calendar date in
`America/New_York`. It is not derived by truncating a UTC timestamp.

The exchange schedule determines:

- whether the date is a valid trading date;
- the immediately preceding completed regular trading session used for prior
  close; and
- the UTC instants corresponding to the current session bounds.

Weekend, holiday, regular-session early-close, and daylight-saving behavior
comes from that schedule. Components must not infer it from weekday arithmetic.
An early regular close affects prior-session identification; it does not
silently shorten the product's separately defined 04:00–20:00 scanner session.

### DTE-SESSION-02 — scanner session bounds

For an active trading date, the scanner market-data session is the half-open
interval:

```text
S = trading_date at 04:00:00 America/New_York
E = trading_date at 20:00:00 America/New_York
session = [S, E)
```

The instant `E` is not part of that session. The final eligible one-second
aggregate is `[E-1s,E)`. A trade or quote whose effective event time equals
`E` is outside the session.

The session binding contains at least the trading date, `S`, `E`, universe
identity, required prior-session date, prior-close adjustment policy, and a
stable binding identifier. All canonical same-session market state belongs to
exactly one binding.

### DTE-SESSION-03 — timestamp encoding

Internal instants are timezone-aware, nanosecond-capable UTC values and preserve
the precision supplied by the source. Persisted and public timestamps use a
precision-preserving RFC 3339 representation with an explicit `Z` or zero
offset. A timestamp without a timezone is invalid at an external, persistence,
or component boundary.

Integer epoch timestamps received from a provider are converted according to
the provider field’s documented unit before normalization. Code must not infer
milliseconds versus nanoseconds from magnitude.

Durations are explicit duration types or unit-suffixed fields. Bare integers
must not carry an implicit time unit.

### DTE-SESSION-04 — exact comparison and rounding

Timestamp comparisons are exact after normalization. Time is rounded only at a
named boundary using one of these operations:

```text
floor_second(t) = greatest whole UTC second <= t
ceil_second(t)  = least whole UTC second >= t
```

Adapters must not silently clamp a future timestamp, shift a bar into the
session, or round an event into a feature window. Any provider clock tolerance
must be explicit in the provider-adapter specification, observable, and must
not alter the normalized event time.

## 4. Clock taxonomy

### DTE-CLOCK-01 — schedule time

Schedule time supplies `trading_date`, `S`, `E`, and the required prior regular
session. It is configuration, not evidence that market data has arrived.

### DTE-CLOCK-02 — provider event time

Provider event time describes when the market event occurred according to the
source:

- aggregate event time is its exact one-second `[window_start,window_end)`;
- trade event time uses valid participant time, otherwise valid SIP time;
- quote event time uses SIP time.

Provider event time determines session and feature-window membership. It does
not determine the order in which concurrent inputs mutate canonical state.

### DTE-CLOCK-03 — receipt time

`received_at` is the live process time at which the scanner accepted the source
payload from its I/O boundary. For replay it is the artifact’s logical delivery
time. Receipt time supports latency and operational diagnostics; it does not
move an event into or out of a market-time window.

### DTE-CLOCK-04 — engine time

The engine receives time through an injected clock:

- a live clock in production; or
- a simulated clock during replay and deterministic tests.

Product state and feature code must not read the operating-system wall clock
directly. Engine time may not regress within a session. A regression is a
global clock-integrity failure: ranking publication is suppressed until the
clock is restored or the process restarts under a valid session binding.

Monotonic elapsed time may be used separately for network deadlines, retries,
and pressure controls. Such operational duration never becomes market time.

### DTE-CLOCK-05 — committed watermark

The committed aggregate watermark `T` is the single market-time boundary for
ranking, aggregate features, mark age, and snapshot coherence. State at `T`
contains accepted aggregate facts whose windows begin in `[S,T)` and the
explicit coverage, failure, or exclusion state required for the claims made by
that snapshot.

`T` is nondecreasing and satisfies `S <= T <= E`. It is not a receipt time,
WebSocket cursor, REST progress time, checkpoint time, or T/Q time.

### DTE-CLOCK-06 — generated time

`generated_at` is engine time when an immutable snapshot is created. It must be
greater than or equal to the snapshot watermark, but it carries no claim that
market state is complete through `generated_at`.

## 5. Interval convention and feature membership

### DTE-WINDOW-01 — all market windows are half-open

All session, hydration, aggregate, coverage, and feature intervals use
`[start,end)`. A boundary event belongs to exactly one adjacent interval.

At committed watermark `T`:

| Meaning | Interval |
| --- | --- |
| Current-session aggregate history | `[S,T)` |
| Qualification window | `[T-60s,T)` |
| Qualification short window | `[T-5s,T)` |
| Rolling 30-minute range | `[T-30m,T)` intersected with `[S,T)` |
| Rolling 60-minute range | `[T-60m,T)` intersected with `[S,T)` |
| Tape Rate, trailing 5 seconds | `[T-5s,T)` while T/Q coverage is valid |
| Tape Rate, trailing 1 second | `[T-1s,T)` while T/Q coverage is valid |

Where product formulas use a different named reference time for an
instantaneous T/Q projection, that specification must state it explicitly and
must not redefine aggregate `T`.

### DTE-WINDOW-02 — aggregate membership

A one-second aggregate belongs to `[a,b)` when:

```text
a <= window_start < b
```

An accepted aggregate has exact whole-second UTC boundaries:

```text
window_end = window_start + 1 second
floor_second(window_start) = window_start
floor_second(window_end) = window_end
S <= window_start < window_end <= E
```

### DTE-WINDOW-03 — trade and quote membership

A trade or quote belongs to `[a,b)` when its effective event time satisfies
`a <= event_time < b`. Receipt order and receipt time do not change membership.

An event delivered late may revise a still-mutable result if its event type and
correction rules permit it. Otherwise it is explicitly rejected as too late or
retained only as a diagnostic; it is never shifted into the current window.

### DTE-WINDOW-04 — absent seconds

An absent one-second aggregate slot is absence, not a zero-volume synthetic
bar. Product formulas may explicitly treat absence as inactivity or calculate
a missing-run length, but canonical storage must not manufacture OHLC, VWAP,
volume, or Average Trade Size for that second.

## 6. Common normalized event envelope

### DTE-EVENT-01 — required envelope

Every engine input has a versioned envelope containing:

- event kind and schema version;
- active session binding identifier;
- source kind: live, historical, replay, or system;
- source position appropriate to that source;
- receipt or logical delivery time; and
- a source-specific payload.

The engine assigns a monotonically increasing `engine_sequence` when it
consumes an admitted input. This sequence defines mutation order and is not a
market timestamp.

### DTE-EVENT-02 — live source position

A live item is ordered by the tuple:

```text
(connection_epoch, frame_sequence, array_index)
```

- `connection_epoch` is a positive, process-local, monotonically increasing
  identifier assigned to each WebSocket connection attempt that can deliver
  data;
- `frame_sequence` is positive and increases when a frame is admitted at the
  bounded transport boundary; and
- `array_index` is zero-based provider order inside a frame.

Connection/control events and end-of-frame or gap markers also receive causal
positions. Map iteration, worker completion order, and goroutine scheduling are
not ordering authorities.

### DTE-EVENT-03 — historical source position

A historical item identifies:

```text
(hydration_generation, request_id, symbol, requested_interval,
 page_or_result_ordinal)
```

`hydration_generation` is positive and increases for each logically new fresh
bootstrap, checkpoint catch-up, or gap-recovery operation. A request token
binds the session, generation, symbol, and exact interval. Results with a stale
binding or generation are fenced and cannot mutate canonical state.

### DTE-EVENT-04 — replay and system positions

A replay item is ordered by `(artifact_id, record_ordinal)`. A system timer or
engine-control event has a process- or run-local monotonically increasing
system sequence. Neither position impersonates a provider sequence.

## 7. Normalized market events

### 7.1 One-second aggregate

#### DTE-AGG-01 — identity

Within one session binding, aggregate identity is:

```text
(symbol, window_start)
```

`window_end`, source, receipt time, and source position are not identity
fields. A symbol is normalized to the exact canonical symbol present in the
bound universe before the event can be accepted.

#### DTE-AGG-02 — canonical fields

A normalized aggregate contains:

- symbol;
- `window_start` and `window_end`;
- open, high, low, close;
- economic volume, preserving provider-supported fractional quantity;
- VWAP;
- Average Trade Size; and
- Average Trade Size provenance.

The two version 1 Average Trade Size mappings are:

| Source | Canonical mapping | Provenance |
| --- | --- | --- |
| Live second aggregate | Provider second-aggregate Average Trade Size field | `live_provider_average` |
| Historical REST aggregate | `floor(volume / transaction_count)` when transaction count is positive | `rest_floor_volume_over_transactions` |

REST requests used for same-session parity are unadjusted. The REST transaction
count is an input to normalization only; it is not exposed as an economically
exact trade count or stored as an independent ranking feature.

Prior empirical comparison found the resulting historical Average Trade Size
differences from the live provider field minimal and negligible for the
version 1 qualification and Activity use cases. The historical mapping is
therefore an approved approximation, not an unresolved parity defect. Its
provenance remains explicit so the scanner never claims that the two source
fields are economically or numerically identical.

Average Trade Size may be zero structurally. Product calculations that divide
by it require a finite positive value and are unavailable when that condition
does not hold.

#### DTE-AGG-03 — structural validity

An aggregate is structurally valid only when:

- all required fields are present;
- price fields and VWAP are finite and positive;
- volume is finite and nonnegative;
- Average Trade Size is an integer greater than or equal to zero;
- `low <= open <= high`, `low <= close <= high`, and `low <= high`; and
- the exact interval and session rules in `DTE-WINDOW-02` pass.

Structural validity does not by itself prove current generation, trusted
coverage, rankability, or feature availability.

#### DTE-AGG-04 — Average Trade Size parity and tests

Tests must verify each source mapping exactly, but must not assert that a REST-
derived Average Trade Size equals the live provider Average Trade Size for the
same second. A REST-compiled replay may consequently produce slightly different
qualification timing, Activity values, or other ATS-dependent statistics than
a captured live run. That difference is accepted version 1 source behavior and
is not replay nondeterminism.

Replay determinism means that the same normalized artifact, configuration, and
logical event order produce the same output at every playback speed. It does
not mean that artifacts built from different source fields must reproduce a
live run exactly.

### 7.2 Trade

#### DTE-TRADE-01 — canonical fields

A normalized trade retains the minimum stable data needed for current Tape
Rate and future T/Q-derived features:

- trading date, symbol, exchange, trade identifier;
- Trade Reporting Facility identifier and an explicit presence flag when the
  provider supplies one;
- price and economic size, preserving fractional quantity;
- normalized condition codes and their provider classification evidence;
- participant timestamp, SIP timestamp, effective event time, and timestamp
  basis;
- receipt time, optional provider sequence, tape, and live causal position;
  and
- lifecycle action, referenced trade identifier, and completeness evidence
  when supplied.

Unknown condition codes and incomplete lifecycle evidence are represented
explicitly. They are not guessed into a qualifying classification.

#### DTE-TRADE-02 — identity and effective time

When all identity inputs are present, trade identity is:

```text
(trading_date, symbol, exchange, trf_present, trf_id, trade_id)
```

The participant timestamp is the effective event time when it is valid and is
not later than the valid SIP timestamp. Otherwise the valid SIP timestamp is
used and the basis is recorded as `sip_fallback`. If neither produces a valid
same-session time, the trade is rejected.

Trade corrections and cancellations may affect a feature only to the degree
that the provider supplies sufficient lifecycle identity and the focused T/Q
feature specification defines the behavior. Absence of that evidence must be
observable; the engine must not claim complete lifecycle reconstruction.

### 7.3 Quote

#### DTE-QUOTE-01 — canonical fields

A normalized quote contains:

- trading date and symbol;
- bid and ask exchange, price, and size with explicit presence;
- quote conditions, indicators, and metadata-quality evidence;
- SIP event time and receipt time;
- optional provider sequence and tape; and
- live causal position.

Quote event time is SIP time. In version 1, source position is the observation
identity used for causal application; no provider quote identity is invented.

#### DTE-QUOTE-02 — structural versus feature validity

A structurally normalized quote need not be usable for Spread. One-sided,
nonpositive, crossed, disallowed-condition, or otherwise non-NBBO-valid states
remain distinguishable from malformed input. A validated locked NBBO is valid
zero spread as required by the product contract. The Spread specification owns
the remaining inclusion rules and time-weighting; normalization owns only
faithful fields, time, and quality evidence.

### 7.4 Connection and control

#### DTE-CONTROL-01 — connection facts

Normalized connection/control events contain the connection epoch, causal
position, receipt time, provider channel or command, affected symbols when
applicable, command token, and acknowledged or failed status.

Connection establishment, aggregate subscription acknowledgement, T/Q
subscription acknowledgement, unsubscribe acknowledgement, provider errors,
and connection loss are facts delivered to the engine. The live adapter may
not directly declare ranking current or mutate T/Q coverage.

### 7.5 Hydration result

#### DTE-HYDRATE-01 — one terminal outcome

Every planned symbol request produces exactly one terminal outcome:

```text
completed_value | completed_empty | failed | canceled | fenced
```

- `completed_value` means the provider request and all pagination completed
  successfully and returned at least one normalized aggregate row;
- `completed_empty` means the complete successful result contained zero rows;
- `failed` means the requested interval was not established;
- `canceled` means work ended under explicit cancellation; and
- `fenced` means the result no longer belongs to the active binding or
  generation.

Progress events are observable but nonterminal and do not prove coverage.
Terminal results include the request token, exact interval, pagination
completion evidence, row count, and any bounded validation/conflict summary.

The accounting identity is always:

```text
planned = completed_value + completed_empty + failed + canceled + fenced
```

#### DTE-HYDRATE-02 — terminal transport success is not automatic acceptance

A successful provider result can still contain rows that the engine rejects or
conflicts with accepted live data. The engine records both the terminal work
outcome and the canonical coverage consequence. `completed_empty` is evidence
about exactly the requested interval; it does not automatically mean the
symbol has never printed in the session.

### 7.6 Timer and evaluation event

#### DTE-TIMER-01 — explicit time advancement

Live and replay runs deliver explicit timer/evaluation events through the
ordered engine path. A timer event contains its engine time and system
sequence. It creates no market fact. Its purpose is to make target-watermark
calculation, stale transitions, empty-window behavior, and publication
deterministic even when no market event arrives during a second.

## 8. Aggregate merge and correction rules

### DTE-MERGE-01 — exact duplicate

Two aggregates with the same identity and equal canonical market values are an
exact duplicate. The later observation increments bounded diagnostics but does
not replace the canonical values or retroactively change the first accepted
delivery evidence.

### DTE-MERGE-02 — revision

An aggregate with an existing identity and different canonical market values
is a revision, not a second bar. A valid accepted revision replaces that
identity’s canonical values and recomputes every still-mutable dependent fact,
including qualification proof and aggregate features.

The exact aggregate correction horizon `H` is defined in the aggregate-state
component specification. `H` must be long enough to preserve the product’s
correction-aware qualification latch and short enough to bound memory.

### DTE-MERGE-03 — live causal precedence

For live observations in the active epoch, the greatest valid live causal
position wins within the correction horizon. A first delivery or revision that
is outside the retained correction horizon is explicitly rejected as too late
and counted; it is not installed into an approximate slot.

An out-of-order aggregate whose identity is new but still within retained
history may be accepted. Event-time order chooses its historical slot; causal
source order chooses precedence among observations of the same identity.

### DTE-MERGE-04 — historical and live precedence

Historical data may populate a missing aggregate identity only under a current
request token whose interval contains that identity. Historical data never
overwrites an already accepted live value for the same identity.

When historical and live values conflict:

- retain the live canonical value;
- record one bounded conflict diagnostic with both provenances;
- do not double-count the bar; and
- mark dependent historical coverage or fields unknown when the conflict means
  their correctness cannot be established.

A current trusted live mark may remain trusted even when an older historical
feature interval is unknown. One conflict must not indiscriminately erase
independent current state.

Conflicting historical rows for one identity within a single completed result
invalidate that row’s coverage claim unless a focused provider specification
defines a deterministic, evidenced resolution. Silent last-row-wins behavior
is forbidden.

### DTE-MERGE-05 — corrections and publication

An accepted correction can produce a new snapshot at the same committed `T`;
the watermark need not advance for values to change. The snapshot carries a
new `generated_at` and version. Consumers must not treat watermark alone as a
unique snapshot identifier.

## 9. Hydration, reconciliation, and no-print proof

### DTE-RECOVERY-01 — interval meaning

A hydration request covers aggregate identities whose `window_start` is in its
half-open interval `[a,b)`. The token records the purpose: fresh bootstrap,
checkpoint catch-up, or known gap recovery.

Workers may execute concurrently, but only the engine installs results in
engine-sequence order after validating the current binding, generation, symbol,
and interval.

### DTE-RECOVERY-02 — live catch-up boundary

For fresh bootstrap or checkpoint restart, define `R` as the first whole-second
boundary at or after receipt of the successful current-epoch `A.*` aggregate
subscription acknowledgement, clamped to `[S,E]`:

```text
R = clamp(ceil_second(aggregate_ack_received_at), S, E)
```

Historical hydration covers `[S,R)` on a fresh start or `[T0,R)` after a
checkpoint at `T0`. The live tail supplies observations beginning at `R`.
Intentional overlap may be used for safety and is reconciled by aggregate
identity and the merge rules; it does not produce two canonical bars.

The acknowledgement receipt boundary is an operational handoff, not a claim
about provider latency. Exact provider subscription protocol behavior belongs
in the live-adapter specification.

### DTE-RECOVERY-03 — reconciliation fence

Before publishing a post-hydration result, the engine captures a live ingress
fence: the greatest admitted live frame position that must be reconciled with
the historical result. The engine processes every classified item through that
position before deriving coverage or `no_print_through(T)` from the result.

Items admitted after the captured fence belong to subsequent ordinary
processing. “Queue currently empty” is not a fence and is not sufficient proof
of reconciliation.

### DTE-RECOVERY-04 — empty is terminal success

A valid `completed_empty` result resolves its requested historical interval as
successful empty coverage. It is not deferred work and must not keep bootstrap
globally incomplete.

For a symbol with no accepted current-session mark, `no_print_through(T)` is
established only when the engine has both:

1. complete successful historical coverage from `S` through the live handoff
   with no aggregate for the symbol; and
2. continuous accepted aggregate transport coverage, reconciled through the
   required live fence, from that handoff through `T`, also with no aggregate
   for the symbol.

Equivalent coverage may be assembled from multiple exact successful
intervals. A failed, canceled, stale, or fenced interval is unknown, not empty.

If an earlier accepted mark exists, an empty later interval means the earlier
mark remains the latest mark subject to its age and coverage rules; it does not
change the symbol to no-print. A later accepted aggregate moves a no-print
symbol into ordinary mark evaluation immediately.

### DTE-RECOVERY-05 — recovery state is local where possible

A known symbol or historical-feature gap makes only dependent symbol fields
unknown unless the gap invalidates complete-population ranking or the global
aggregate stream. A global aggregate coverage gap prevents the committed
watermark from advancing past the unsupported boundary until the gap is
recovered or the product contract permits an explicitly degraded projection.

Recovery completion does not create a second evaluation path. The same
canonical evaluator runs at the current committed `T` after accepted results
are installed.

## 10. Committed watermark and publication

### DTE-COMMIT-01 — candidate target

Let `D` be the approved aggregate evaluation delay. At an engine timer time
`now`, the candidate target is:

```text
target = clamp(floor_second(now - D), S, E)
```

`D` and any capacity-specific timing thresholds are set in the runtime or
operations specification. Changing `D` changes latency, not event identity or
window membership.

### DTE-COMMIT-02 — advancement requirements

The engine may advance `T` toward `target` only after:

- all admitted aggregate and relevant control inputs through the captured
  ingress fence have been consumed or explicitly rejected;
- active-session and connection-epoch checks have been applied;
- no unresolved global aggregate-transport gap precedes the proposed `T`; and
- the snapshot can state honest population accounting and field availability.

Committed time is proof about scanner processing and known coverage. It is not
an omniscient guarantee that the provider will never deliver a later valid
correction. The correction horizon defines how such revisions are handled.

Symbol-level historical hydration may still be incomplete while a partial
`degraded_bootstrap` projection advances, if and only if every product-level
condition for that projection is satisfied. That fallback does not excuse a
global aggregate-transport gap. Exact qualified ranking requires the
complete-population evidence in the product contract.

### DTE-COMMIT-03 — evaluation semantics

At `T`, the engine computes aggregate marks, qualification, aggregate features,
ranking, and population accounting from one coherent canonical state. A mark’s
causal age is:

```text
mark_age = T - mark.window_end
```

It must be nonnegative. A bar whose `window_start = T` belongs to the next
state and cannot be the mark for the snapshot at `T`.

The engine may publish fewer than 20 rows, but it may not use a later T/Q or
recovery boundary to fill or reorder the table.

### DTE-COMMIT-04 — snapshot identity

Each immutable snapshot contains at least:

- schema version and a publication identifier independent of market time;
- session binding identifier and trading date;
- committed watermark `T` and `generated_at`;
- the aggregate ingress fence supporting the projection;
- overall ranking/readiness state;
- ordered rows with independent field availability;
- exact population and hydration accounting; and
- bounded transport, recovery, checkpoint, and T/Q status.

Two snapshots may share `T` after a correction, availability transition, or
nonmarket operational update. Consumers must not use `T` as a unique snapshot
identifier. The API specification defines the exact publication-identifier and
compatibility representation.

## 11. T/Q causal coverage

### DTE-TQ-01 — selected enrichment has its own coverage

Trade/quote coverage is per symbol and per channel. It begins strictly after
the completed acknowledgement’s live causal position for the current
connection epoch. A trade or quote at or before that position cannot satisfy a
selected-symbol window merely because its event time is recent.

Coverage ends at the earliest of:

- the causal boundary established by a completed unsubscribe;
- an intentional pressure-shedding rejection or closed bounded input;
- connection loss or connection-epoch change; or
- session end.

Subscription intent and a sent command are not coverage. An acknowledgement
alone also does not instantly warm a trailing window; sufficient event-time
history and continuous causal coverage are both required.

### DTE-TQ-02 — gaps clear derived state

A gap closes the affected trade or quote coverage interval and changes
dependent fields to warming, stale, unavailable, or invalid as defined by the
feature contract. It must not preserve an apparently current Tape Rate or
Spread across an unknown interval. Absence of trades or quotes while the
acknowledged connection and channel coverage remain continuous is not a gap:
Tape Rate may be zero and the last valid Spread remains numeric with quote age
and stale status.

Resubscription starts new coverage after its acknowledgement and warms the
feature again. T/Q coverage, pressure state, and feature time never gate or
advance aggregate `T`, aggregate qualification, or ranking.

### DTE-TQ-03 — intentional shedding is observable

Under pressure, the transport may reject T/Q items before expensive
normalization or unsubscribe selected channels according to the approved
pressure policy. Every such boundary is counted and delivered to the engine so
coverage closes honestly. Accepted aggregate items must continue to be
processed or explicitly fail globally; they may not disappear behind T/Q load.

## 12. Deterministic aggregate replay

### DTE-REPLAY-01 — normalized artifact boundary

Version 1 replay consumes a versioned artifact of normalized one-second
aggregates. It does not open a fake WebSocket and does not maintain a second
scanner implementation. Artifact records pass through the same engine event
and canonical merge path as live and historical normalized aggregates.

The artifact records source provenance and normalization policy. REST-compiled
Average Trade Size therefore remains
`rest_floor_volume_over_transactions`; replay must not label it as the live
provider Average Trade Size field.

### DTE-REPLAY-02 — logical delivery order

For a compiled final-bar day artifact, each aggregate’s default logical
delivery time is `window_end`. Records sharing a logical delivery time are
ordered deterministically by:

```text
(window_start, symbol)
```

The compiler assigns the resulting immutable `record_ordinal`. Synthetic test
artifacts may specify later logical delivery times and multiple observations of
one identity to exercise duplicates, out-of-order delivery, and corrections;
their ordinals remain explicit.

Compiled final bars do not reproduce live provider latency or correction
chronology and must not be presented as evidence about either.

### DTE-REPLAY-03 — simulated clock algorithm

Replay advances the injected clock to the next logical time, delivers all
records at that time in artifact order, then delivers the deterministic timer
event for that time. It also emits required whole-second timer events during
market seconds with no records so empty-window and stale behavior are
reproducible.

Playback speed changes only how quickly the run finishes in wall time. It must
not change event order, committed watermarks, snapshots at equivalent logical
times, qualification, or final canonical state.

T/Q replay is deferred in version 1. Future T/Q artifacts must use the same
trade/quote schemas and explicit causal coverage rather than weakening this
contract.

## 13. Checkpoint and restart cutoffs

### DTE-CHECKPOINT-01 — coherent as-of boundary

A checkpoint sealed at `T0` represents canonical state exactly as of committed
watermark `T0`, including all accepted aggregate identities in `[S,T0)`,
qualification state, retained correction-tail state, coverage, accounting, and
other restart-required derived inputs.

A checkpoint is valid only if every mutable contribution required to reproduce
state at `T0` is included or explicitly marked incomplete. A separately read
live map is not a coherent checkpoint.

`checkpoint_created_at` is persistence metadata and satisfies
`checkpoint_created_at >= T0`; it does not change the as-of boundary.

### DTE-CHECKPOINT-02 — restart interval

After loading a valid same-binding checkpoint at `T0`, historical catch-up
covers `[T0,R)`, followed by the reconciled live tail. An aggregate whose
`window_start = T0` is not in checkpoint state and belongs to catch-up.

The current WebSocket connection epoch, T/Q causal coverage, and ephemeral T/Q
feature windows do not survive restart. They begin from the new live
connection’s acknowledgements. Persisted aggregate qualification and history
survive only when their complete restart inputs and correction status are
coherent at `T0`.

### DTE-CHECKPOINT-03 — progress boundaries cannot override T

Checkpoint `T0`, live handoff `R`, a recovery interval end, T/Q coverage end,
and snapshot `generated_at` are named boundaries with different meanings. None
may overwrite the committed ranking watermark or be displayed as if it were
aggregate ranking time.

## 14. Rejection, diagnostics, and accounting

### DTE-REJECT-01 — required rejection classes

Adapters and the engine expose bounded counts for at least:

- malformed or structurally invalid data;
- wrong session binding or trading date;
- unknown or ineligible symbol;
- stale connection epoch or hydration generation;
- event outside requested or scanner interval;
- exact duplicate;
- conflicting revision;
- too-late event outside the correction horizon;
- canceled or fenced work; and
- pressure-shed T/Q data.

Exact metric names and cardinality controls belong in the observability
specification. Raw payloads, unbounded condition lists, and per-event logs must
not create unbounded diagnostic memory.

### DTE-REJECT-02 — market-population categories remain distinct

At a published `T`, these states must not be collapsed into “missing”:

- invalid or missing prior close;
- trusted rankable mark;
- trusted below-price mark;
- `no_print_through(T)`;
- invalid mark;
- unknown due to failure or fence;
- qualification/activity exclusion; and
- invalid or unavailable historical feature range.

The exact product accounting identities in `product-goals.md` are evaluated
from these distinct canonical categories.

## 15. Cross-component invariants

The following are mandatory implementation and test invariants:

1. Exactly one session binding owns canonical state.
2. Exactly one engine sequence orders state mutation.
3. Exactly one committed aggregate watermark governs ranking.
4. All market intervals are half-open.
5. Aggregate identity is `(symbol,window_start)` within the binding.
6. Source receipt order never changes market-window membership.
7. Empty successful hydration is terminal success for its exact interval.
8. No-print is a proved symbol state, never a synthetic mark or unresolved
   work category.
9. Historical data never overwrites an accepted live aggregate identity.
10. T/Q loss can degrade T/Q fields but cannot corrupt aggregate ranking.
11. Replay speed cannot change logical output.
12. A checkpoint boundary and a snapshot watermark retain their distinct
    meanings.
13. Every admitted aggregate is canonically accepted, explicitly rejected, or
    participates in an explicit global failure; it never silently disappears.

## 16. Required proof scenarios

Focused component and integration specifications must allocate tests that prove
at least these behaviors without duplicating the entire suite at every layer:

- 04:00 inclusive and 20:00 exclusive session boundaries in New York and UTC;
- prior-session selection across a weekend, exchange holiday, and early close;
- exact adjacency of half-open one-second and rolling windows;
- a WebSocket array whose aggregate and T/Q items receive stable causal order;
- exact duplicate, live correction, out-of-order new bar, too-late bar, and
  REST/live conflict behavior;
- exact live and historical Average Trade Size mappings without requiring
  numerical equality between those distinct source fields;
- successful empty hydration becoming terminal and later transitioning on a
  real live print;
- failed and fenced hydration remaining unknown rather than becoming empty;
- reconciliation through a captured ingress fence while new frames continue;
- correction-driven reevaluation without watermark advance;
- T/Q acknowledgement, warm-up, gap, pressure shedding, and resubscription
  without aggregate-rank changes;
- checkpoint cutoff at `T0` followed by exact `[T0,R)` catch-up; and
- identical replay results at multiple playback speeds, including market
  seconds with no aggregate records.

## 17. Deliberately delegated decisions

The following details are required later but do not alter this contract:

- provider wire-field names, timestamp units, condition maps, and explicit
  future-clock tolerance;
- aggregate evaluation delay `D` and correction horizon `H`;
- bounded queue sizes, pressure thresholds, retry budgets, and request-page
  limits;
- exact checkpoint and replay file encodings;
- per-feature trade-condition and NBBO-validity policies;
- API serialization details and compatibility policy; and
- metric names, log sampling, and diagnostic retention limits.

Those specifications must choose exact values before implementation is called
production-ready. They may tune latency and capacity but may not redefine the
session, canonical identities, causal order, coverage, no-print proof, or
committed watermark established here.
