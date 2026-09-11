# Qualification and ranking

The scanner first decides whether a symbol has demonstrated the required
one-minute aggregate tape, then ranks passers by return from the adjusted prior
regular-session close. Aggregate measurements and T/Q enrichment cannot become
hidden filters.

## Evaluation boundary

All calculations use the committed aggregate watermark `T`. Windows are
half-open, so an event beginning exactly at `T` belongs to a later evaluation.
The mark is the Close of the latest trusted accepted aggregate strictly before
`T`.

A symbol is rankable only when it has a finite positive adjusted prior close
and a finite current-session mark of at least USD 0.25. Missing or invalid
reference or mark facts remain visible in population accounting.

## Qualification window

Over `[T-60s,T)`, define:

```text
N    = distinct accepted one-second aggregates
G    = longest run of missing whole seconds, including window edges
A60  = sum(Volume / AverageTradeSize)
A5   = the same sum over [T-5s,T)
DV60 = sum(Volume * VWAP)
C    = maximum one-second Volume / total Volume
```

The window passes only when:

```text
latest price >= USD 0.25
N >= 45
G <= 3 seconds
A60 is available and >= 1,000
DV60 >= USD 250,000
A5 is available and >= 100
C <= 0.50
```

`A60` or `A5` is unavailable if a contributing aggregate lacks a finite
positive Average Trade Size. A missing second is inactivity, not a fabricated
bar. Unknown coverage or a conflict prevents a complete proof for the affected
window.

## Qualification latch and corrections

The first passing window creates a same-session qualification proof. While the
sole proof remains inside the 16-minute aggregate correction horizon, an
accepted correction can revoke it. When a proof ages beyond that horizon it is
final: later quiet tape does not unqualify the symbol.

The latch records enough evidence to re-evaluate corrections without using
arrival order as market truth. A new passing window may replace a revoked
provisional proof.

## Ranking

Qualified rankable symbols are sorted by:

1. `100 * (Last / adjusted_prior_close - 1)` descending; then
2. exact canonical symbol ascending.

At most 20 rows are published. Fewer passers produce fewer rows. Float, Volume,
From Open, Day Range, Activity 30s, Move 30s, Tape 5s, Spread, T/Q membership,
field availability, pressure, and client-side rank movement do not alter this
order.

## Exact and partial views

`qualified_current` means population coverage, qualification, rankability, and
ordering are exact at `T`. Only this state drives T/Q subscription desire.

When current trusted marks exist but the full population cannot yet be proved,
the engine may publish a visibly partial ordering. It cannot label that view
qualified, imply absent symbols cannot outrank it, or use it for T/Q selection.
A legitimate exact zero-row result can be ready.

## Population accounting

The primary population identities are:

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

Qualification status and feature availability are overlapping diagnostics, not
additional mutually exclusive population buckets. The evaluator stages the
full population, validates these identities, and commits ranking and `T`
atomically.

