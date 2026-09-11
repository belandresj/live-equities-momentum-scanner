# Product goals

This document defines the scanner's user-visible market meaning. Architecture
and implementation may choose how to realize it but may not silently change the
measurements, ranking, availability, or trust claims below.

## Product statement

The Live Equities Momentum Scanner is a real-time market-data discovery tool for
a discretionary U.S. equities momentum trader. It answers:

1. Which eligible stocks have demonstrated the required aggregate tape and
   gained the most from the adjusted previous regular-session close?
2. What are each displayed stock's Float, cumulative session Volume, latest
   price, and session location?
3. Is recent share-volume participation unusual, and what signed 30-second move
   accompanied it?
4. For displayed leaders, how fast are qualifying original trades printing and
   what spread is currently quoted?

It does not produce a prediction, recommendation, entry, exit, order, or claim
of executable expectancy.

## Product priorities

When behavior conflicts, use this order:

1. Preserve aggregate ranking correctness.
2. Represent availability and uncertainty honestly.
3. Continue or recover without fabricating market state.
4. Preserve aggregate-derived context and momentum independently.
5. Degrade T/Q enrichment before aggregate processing.
6. Keep presentation deployable independently from the backend.

## Session, universe, and reference facts

The scanner session is `[04:00,20:00)` America/New_York on one exchange-declared
trading date.

The eligible universe contains active provider records with `market=stocks`,
`locale=us`, and type `CS` or `ADRC`. ETFs, ETNs, preferreds, units, warrants,
rights, funds, inactive records, non-U.S. records, and unknown types are
excluded.

Ranking requires the finite positive adjusted close from the immediately
preceding completed regular trading session. A missing or invalid prior close
makes only that symbol unrankable and must remain visible in accounting. Do not
substitute the current open, an unadjusted close, or an unidentified older
close.

Float is provider-reported public free float in shares. It is optional,
display-only reference enrichment and carries source, effective date, retrieval
time, and fresh/cache provenance. It is not shares outstanding. Missing,
invalid, ambiguous, or stale Float cannot change qualification, rank, readiness,
or live processing.

## Qualification and rank

At committed aggregate watermark `T`, the latest trusted mark is the Close of
the latest accepted aggregate strictly before `T`. It must be from the current
session, finite, at least USD 0.25, and have nonnegative age at `T`.

A fully reconciled empty historical interval establishes resolved no-print
evidence. It does not create a price and is not unfinished work. A later valid
live aggregate creates the first real mark and triggers normal reevaluation.

Qualification is evaluated before ranking. Over `[T-60s,T)` define:

```text
N    = number of distinct accepted one-second aggregates
G    = longest run of missing whole-second slots, including both edges
A60  = sum(Volume / AverageTradeSize) over 60 seconds
A5   = sum(Volume / AverageTradeSize) over the final 5 seconds
DV60 = sum(Volume * VWAP) over 60 seconds
C    = maximum one-second Volume / total 60-second Volume
```

A passing window requires:

```text
latest_price >= USD 0.25
N >= 45
G <= 3 seconds
A60 available and >= 1,000
DV60 >= USD 250,000
A5 available and >= 100
C <= 0.50
```

`A60` and `A5` are unavailable when a contributing aggregate lacks a finite
positive Average Trade Size. Missing seconds are inactivity; they are not
fabricated bars.

The first passing window creates a same-session qualification latch. A
correction may revoke the sole passing proof while it remains inside the
16-minute aggregate correction horizon. Once a proof is finalized beyond that
horizon, later quiet tape does not revoke it.

Qualified rankable symbols are ordered by:

1. From Close % descending; then
2. exact symbol ascending.

The first 20 are displayed. Fewer than 20 passers produce fewer than 20 rows.
No display field, T/Q fact, pressure state, or presentation value may become an
implicit filter or secondary ranking key.

When full population or qualification completeness is not yet proved, the
backend may publish a visibly partial ordering of currently trusted rankable
marks. It must not call that view qualified, imply that omitted symbols cannot
outrank displayed rows, or use it to request T/Q subscriptions.

## Display contract

The dashboard columns are:

```text
CONTEXT                       LOCATION                     MOMENTUM            TAPE / EXECUTION
RANK SYMBOL FLOAT VOLUME LAST | FROM CLOSE % FROM OPEN % DAY RANGE | ACTIVITY 30s MOVE 30s | TAPE SPEED SPREAD
```

Rank is the server order. The client may show bounded movement relative to the
closest valid displayed snapshot approximately 60 seconds earlier, but that
presentation-local history cannot sort, smooth, filter, or delay server rows.

### From Close %

```text
100 * (Last / adjusted_prior_close - 1)
```

This is the sole numeric ranking key after qualification.

### From Open %

```text
100 * (Last / first_session_aggregate_open - 1)
```

The base is the Open of the earliest accepted aggregate beginning in `[S,T)`.
It does not depend on selected-row T/Q and is unavailable until its own history
is trustworthy.

### Day Range

```text
100 * (Last - session_low) / (session_high - session_low)
```

The extrema cover accepted aggregates in `[S,T)`. A zero-width, incomplete, or
invalid range is unavailable or invalid rather than clamped to a false value.

### Volume

```text
sum(aggregate Volume_i), for window_start in [S,T)
```

This is correction-aware economic share volume. Proven no-print seconds
contribute zero without creating bars. Unknown coverage or conflict prevents a
current complete value.

### Activity 30s

The current target is the mean share volume per second over `[T-30s,T)`.
Reference endpoints are:

```text
r_k = T - 30s - 5s*k, k = 0..54
reference_k = mean share volume per second over [r_k-30s,r_k)
```

The reference inputs cover exactly `[T-330s,T-30s)` and never overlap the
target. With all 55 references available:

```text
Activity 30s = 100 * count(reference_k <= target) / 55
```

Ties are inclusive. The result is a descriptive empirical percentile in
`[0,100]`, not a probability. It warms until 330 seconds of exact coverage are
available.

### Move 30s

```text
100 * (P(T) / P(T-30s) - 1)
```

`P(b)` is the latest trusted accepted aggregate Close strictly before `b`.
Proven quiet intervals carry the last actual mark to the boundary for this
calculation without manufacturing a bar. Unknown coverage that could hide a
later mark makes the value unavailable.

### Tape 5s

```text
distinct qualifying original trades in [T-5s,T) / 5
```

Trade time uses participant execution time with explicit SIP fallback. Exact
duplicates, incomplete identities, unknown/unreviewed conditions, non-volume
records, and lifecycle-adjustment records do not enter the qualifying-original
basis. During continuous data-confirmed trade coverage, genuine silence is
numeric zero. There is no public one-second Tape field.

### Spread

Spread is the latest valid two-sided non-crossed quote's ask-minus-bid:

```text
cents = 100 * (ask - bid)
basis_points = 10,000 * (ask - bid) / midpoint
```

Locked quotes are valid zero spread. Quote age is measured against `T`; a valid
quote older than two seconds is stale. A latest one-sided quote is unavailable
and a latest crossed quote is invalid. Spread is not a rolling median.

## Independent availability

Fields use explicit states such as `warming`, `current`, `unavailable`, `stale`,
and `invalid`. Absence is never encoded as a genuine numeric zero. Each field
degrades from its own evidence:

- missing Float affects only Float;
- incomplete aggregate history affects only the dependent aggregate features
  unless it prevents rankability or qualification itself;
- T/Q entitlement, commands, coverage, decoding, retention, or pressure affect
  only Tape and Spread; and
- API/UI failure does not mutate engine market state.

Every symbol capable of entering the table retains at least 330 seconds of
aggregate and exact coverage evidence plus the preceding mark needed for Move
30s. Ranking membership cannot initiate, backfill, or change that history.

## Trade and quote behavior

For an exact qualified ranking, the scanner desires T/Q for every displayed
symbol. Trade and quote coverage begin independently with the first valid
post-write data fact after the command boundary. Command success alone does not
make either channel current.

If shared-feed consumption threatens aggregate timeliness or accounting, the
scanner may:

1. stop adding T/Q work;
2. shed T/Q facts while continuing to classify frames;
3. unsubscribe some or all selected T/Q; and
4. restore T/Q only after sustained healthy aggregate evidence, with fresh
   channel warm-up.

It must preserve aggregate and control facts throughout. T/Q can never select,
exclude, qualify, or rank a symbol.

The current implementation accepts T/Q disorder within 30 seconds of `T` and
retains duplicate fingerprints through 30 seconds after first receipt.
Aggregate-watermark staleness alone masks T/Q visibility without removing
subscriptions. Recovery requires five seconds of both elapsed time and
watermark advancement, plus a recovery-period quote for a current spread.
See [trade/quote enrichment](../features/trade-quote-enrichment.md).

## Operation and support boundary

Fresh reference resolution and aggregate hydration are the supported process
start and restart path. Until hydration and the live ingress fence reconcile,
the scanner reports warming, partial, or unavailable state honestly.

Replay and checkpoint packages have been removed from the repository. API v2
retains a fixed disabled checkpoint object for compatibility. Unit tests,
fake-provider tests, and deterministic event traces are engineering evidence;
they are not product replay and do not prove live chronology.

## API and dashboard boundary

The backend owns all market calculations, ranking, coverage, readiness, and
accounting. It publishes one versioned read-only snapshot. The browser validates
that schema and owns formatting, accessibility, status language, color, and
bounded rank-movement history only.

The dashboard can restart independently. Delayed or unavailable API transport
must visibly freeze/retain prior rows rather than make them appear current.
T/Q degradation produces a separate warning and does not demote an otherwise
exact aggregate ranking.

## Accounting identities

Every published universe satisfies:

```text
universe_total
  = valid_prior_close
  + invalid_or_missing_prior_close

valid_prior_close
  = trusted_rankable_mark
  + trusted_below_price_mark
  + no_print_through_T
  + invalid_mark
  + unknown_due_failure_or_fence
```

Hydration work satisfies:

```text
planned
  = open
  + completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Aggregate, T/Q, command, queue, transition, and publication families likewise
use closed accounting. Qualification, Float, feature availability, and T/Q
coverage are overlapping dimensions and must not be forced into the primary
population partition.

## Non-goals

- automated execution, broker integration, or trade management;
- predictive signals, price targets, or expectancy claims;
- full-universe T/Q ranking inputs;
- supported replay or checkpoint restart;
- the superseded Activity composite, HOD drawdown, rolling 30/60-minute range
  fields, or public one-second Tape;
- a database, raw-event journal, generic event bus, microservice split,
  per-symbol workers, or runtime plugin system; and
- silent fallbacks that fabricate marks, history, qualification, coverage, or
  readiness.

Passing correctness tests proves only the behavior those tests exercise. It
does not establish market-hours provider capacity, predictive value, costs,
latency distributions, or executable trading expectancy.
