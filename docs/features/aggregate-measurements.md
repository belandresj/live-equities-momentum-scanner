# Aggregate-derived measurements

These fields are derived only from canonical one-second aggregates and exact
coverage evidence at committed watermark `T`. They cannot affect qualification
or ranking except where the separate qualification rules explicitly consume
the same underlying aggregates.

## Shared rules

All intervals are half-open. Accepted corrections replace prior economics for
the same second; they are not added as new volume or a second bar. Proven
no-print seconds contribute inactivity without fabricating OHLC values.

Each field reports its own status. Missing coverage, a localized conflict, an
invalid denominator, or insufficient warm-up makes only the dependent field
unavailable, warming, or invalid.

## Volume

```text
Volume = sum(aggregate Volume_i), window_start in [S,T)
```

This is correction-aware economic share volume across the scanner session.
Exact coverage is required for a complete current value.

## Last and mark age

`Last` is the Close of the latest trusted accepted aggregate strictly before
`T`. Mark age is `T` minus that aggregate's end. A proven quiet interval may
leave the actual mark unchanged; it never creates a new mark.

## From Open

```text
100 * (Last / first_session_aggregate_open - 1)
```

The base is the Open of the earliest trusted accepted aggregate beginning in
`[S,T)`. It is unavailable until the history needed to prove that earliest
aggregate is trustworthy.

## Day Range

```text
100 * (Last - session_low) / (session_high - session_low)
```

The low and high are extrema across accepted aggregates in `[S,T)`. A
zero-width or invalid range is not clamped to a false numeric value.

## Activity 30s

The target is mean share volume per second over `[T-30s,T)`. Define 55
reference endpoints spaced five seconds apart:

```text
r_k = T - 30s - 5s*k, k = 0..54
reference_k = mean share volume per second over [r_k-30s,r_k)
```

The reference history is exactly `[T-330s,T-30s)` and does not overlap the
target. With all references available:

```text
Activity 30s = 100 * count(reference_k <= target) / 55
```

Ties are inclusive. This is an empirical historical percentile, not a
probability or forecast. It warms until the exact 330-second history exists.

## Move 30s

```text
100 * (P(T) / P(T-30s) - 1)
```

`P(b)` is the latest actual trusted Close strictly before boundary `b`. Proven
quiet coverage carries that actual mark to the calculation boundary. Unknown
coverage that could hide a later mark makes the result unavailable.

## Retention

Every symbol that could enter the table retains at least the 330-second
aggregate/coverage history and preceding mark required by these features.
Ranking membership cannot initiate or backfill feature history.

The superseded HOD drawdown, rolling 30/60-minute range, and composite Activity
implementations have been removed. They are not calculated by the live backend.
