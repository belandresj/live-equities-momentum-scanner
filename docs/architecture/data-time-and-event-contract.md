# Data, time, and event semantics

This document defines the identities and ordering rules shared by the live
adapter, hydration, engine, evaluation, publication, API, and dashboard.

## Session and intervals

For one exchange-declared trading date:

```text
S = 04:00:00 America/New_York
E = 20:00:00 America/New_York
scanner session = [S,E)
```

All internal instants are UTC. Provider timestamps are converted only after
their unit and source field are validated. Market intervals are half-open.

One-second aggregates have whole-second UTC boundaries, exactly one second of
duration, and lie entirely within `[S,E)`. Aggregate identity is
`(binding, symbol, window_start)`; source and arrival time do not create a
second identity.

The current public feature windows are:

| Measurement | Interval |
| --- | --- |
| Qualification | `[T-60s,T)` and `[T-5s,T)` |
| Activity target | `[T-30s,T)` |
| Activity references | 55 trailing 30-second windows ending every five seconds across `[T-330s,T-30s)` |
| Move 30s | latest actual mark strictly before `T-30s` to latest actual mark strictly before `T` |
| Tape 5s | `[T-5s,T)` |
| Session Volume, From Open, Day Range | `[S,T)` |

Superseded rolling-range and composite-activity implementations have been
removed. One-second Tape is not a public product field.

## Clocks

These clocks are distinct:

- **Schedule time** selects `S`, `E`, and the prior session. It makes no data
  arrival claim.
- **Provider event time** determines aggregate, trade, and quote membership.
- **Receipt time** is sampled at the scanner's provider boundary and supports
  latency, coverage start, and delivery diagnostics.
- **Engine admission time** is sampled by the engine for ordered retention and
  expiry semantics. Callers cannot supply it.
- **Committed watermark `T`** is the latest atomically applied full-population
  aggregate evaluation boundary.
- **Generated time** is sampled when an immutable publication is built.

Receipt, admission, generated, and wall-clock time cannot move an event to a
different market window. `T` cannot advance past proven aggregate support.

## Causal positions

Every live array element carries:

```text
(connection_epoch, frame_sequence, array_index)
```

Comparison is lexicographic. Connection epoch identifies the socket attempt;
frame sequence is assigned when a raw frame is read; array index preserves JSON
array order. A status or fence after a mixed frame cannot overtake an earlier
array element.

Historical hydration facts carry generation, request identity, and record
ordinal. System inputs receive an engine-owned system sequence where applicable.

Every admitted logical input receives one engine sequence. A live market-data
batch links a frame's children contiguously but does not collapse their logical
identity, order, dispositions, or accounting.

## Normalized event families

### Aggregate

A normalized aggregate contains schema, binding, source, symbol, exact
one-second window, OHLC, economic Volume, VWAP, Average Trade Size and its
provenance, delivery time, and exactly one source position.

Live Average Trade Size uses the provider's live average field. REST hydration
uses `floor(Volume / transactions)` when the provider supplies a positive
transaction count. The distinct provenance is retained; numerical equality is
not assumed.

OHLC and VWAP must be finite and positive, Volume finite and nonnegative,
Average Trade Size nonnegative, and `low <= open,close <= high`.

### Trade

A normalized trade contains binding/date/symbol, reviewed identity fields,
economic price and size, participant execution time or explicit SIP fallback,
receipt time, reviewed conditions, lifecycle classification, and live causal
position.

Trade event time controls Tape membership. Engine admission time controls the
30-second duplicate-fingerprint retention cutoff, measured from first receipt.

### Quote

A normalized quote contains binding/date/symbol, SIP time, receipt time, at
least one valid price side, reviewed condition/indicator shape, and live causal
position. One-sided and crossed quotes are structurally representable so Spread
can report unavailable or invalid rather than losing the fact.

### Control and terminals

Connection attempt, connected/authenticated/subscription results, aggregate
loss, T/Q command results, quarantines, pressure samples, fences, and terminal
causes are closed typed inputs. Provider prose, credentials, URLs, and payloads
do not enter canonical state or public diagnostics.

### Hydration

One planned symbol interval receives zero or more sealed chunks followed by
exactly one terminal outcome: completed with values, completed empty, failed,
canceled, or fenced. Transport success is not canonical acceptance; each row
still passes aggregate validation and merge.

### Timers

Callers request a timer admission, but the engine samples the relevant time and
sequence. Timers advance maintenance and evaluation; the absence of provider
messages is not itself a timer fact.

## Aggregate merge and corrections

For one canonical identity:

- Equal values are an exact duplicate. A later live duplicate may advance the
  greatest causal support without changing economics.
- A later live causal position with unequal values revises the identity while
  it is within the 16-minute correction horizon.
- An older live causal position is rejected as nonprecedent.
- Unequal values at the same authoritative live position withdraw the identity
  and enter canonical integrity protection.
- Historical data can fill an absent identity but cannot overwrite accepted
  live authority.
- Unequal historical facts for one identity create a localized historical
  conflict and withdraw that identity from trusted coverage.
- Live facts can replace historical authority for the same identity.
- Nonhistorical corrections older than the 16-minute horizon are rejected.
- Events ending after engine time are rejected as future event time.

Accepted inserts, revisions, withdrawals, and exact duplicates receive distinct
dispositions and update closed accounting.

## Coverage and no-print

Exact aggregate coverage for `[a,b)` requires every active second to be one of:

- a canonical aggregate in the mutable correction tail;
- a folded canonical-presence bit; or
- an explicit proven-absence bit;

and no localized conflict may intersect the interval.

Proven absence is installed only from a complete engine-validated hydration or
live fence. It does not create an aggregate, price, or volume. Unknown/fenced
work remains unknown.

T/Q coverage is different. A successful subscribe write establishes only a
causal boundary and requested membership. Trade coverage begins on the first
valid trade after that boundary; quote coverage begins independently on the
first valid quote. A dropped, unsubscribed, quarantined, pressure-shed, or
superseded channel closes coverage immediately and requires fresh data warm-up.

## Fences and linearization

An aggregate-ingress fence is inserted into the same decoded-batch FIFO after all
frames already read. When the engine consumes it, every earlier frame element
has reached a final disposition. Reconciliation can therefore combine terminal
REST results with the exact live prefix through the fence without racing the
reader.

A live-coverage fence similarly captures a concrete decoded-batch boundary for
ordinary evaluation. Fence markers follow earlier admitted batches. After a
batch transfers to the engine owner, caller cancellation cannot discard its
suffix; delivery completes with per-input dispositions or typed containment.

## Committed watermark and publication

The engine stages one full-population candidate at a supported target, validates
population/qualification/feature/ranking identities, applies the complete
candidate, and then advances `T`. A failed candidate cannot partially change
rows or `T`.

Accepted aggregate changes retain a route to an ordinary timer or fence
evaluation. T/Q transitions may replace or cadence-coalesce T/Q projection, but
they cannot advance aggregate `T`.

One immutable publication contains the lifecycle, `T`, ranking, T/Q projection,
accounting, and last disposition derived from the same ordered state. Snapshot
capture and API mapping reject mixed-publication joins.
