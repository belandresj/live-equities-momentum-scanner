# Reference data and session binding

Reference resolution creates the immutable market context for one scanner run.
It is deliberately completed outside the state engine before live state can be
interpreted.

## Session schedule

`internal/session` validates an embedded NYSE schedule and resolves:

- the requested exchange trading date;
- the regular-session close, including early closes;
- the immediately preceding completed regular session; and
- the scanner interval `[04:00,20:00)` America/New_York.

The scanner interval is the product's premarket-through-after-hours window. It
does not redefine the prior close: that reference still comes from the previous
regular session selected by the exchange calendar.

An unsupported date, malformed calendar entry, invalid timezone transition, or
missing prior session prevents a trustworthy binding. Weekends and exchange
holidays are not silently advanced to another requested trading date.

## Eligible universe

`internal/reference` resolves provider reference records for the exact trading
date. A symbol is eligible only when the record is active and has:

```text
market = stocks
locale = us
type = CS or ADRC
```

Provider pagination is bounded and validated. Duplicate or conflicting symbol
facts cannot silently create two identities. The normalized symbol set is
deterministic and becomes immutable for the run.

## Adjusted prior close

Every eligible symbol is joined to a finite positive adjusted close from the
immediately preceding regular session. The join is exact by canonical symbol
identity and trading date.

A missing or invalid prior close makes that symbol unrankable, but it remains in
population accounting. The scanner never substitutes a current-session open,
an unadjusted close, or a close from an unidentified older date.

## Float

Float is optional public-free-float enrichment. It carries the provider value,
source, effective date, retrieval time, and whether it came from a fresh
response or cache.

Float is not shares outstanding and does not affect universe eligibility,
qualification, rank, hydration, readiness, or T/Q selection. Missing,
ambiguous, invalid, or stale Float remains visibly unavailable.

## Cache behavior

Reference caches are exact-date evidence, not a general stale-data fallback.
Each cache entry is validated for schema, query identity, trading date, content
identity, bounds, and completeness before use. A valid exact-date cache may
contain an empty result when emptiness is a legitimate completed response.

Float cache fallback is field-local. It must not turn a failed universe or
prior-close acquisition into an apparently valid binding.

Cache I/O is bounded and occurs outside the engine owner. Provider credentials,
URLs, and raw response bodies are not copied into engine state or public
diagnostics.

## Binding

The completed binding contains the trading date, session boundaries,
schedule provenance, universe identities, prior-close facts, Float facts, and
closed reference accounting. The engine validates and installs it once.

The binding is immutable after installation. A different trading date,
universe, or prior-close basis requires a new engine instance. Live events,
hydration rows, commands, and fences must match the binding
identity before they can affect canonical state.

