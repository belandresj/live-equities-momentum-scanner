# Live Equities Momentum Scanner

Market and measurement vocabulary for the private live U.S. equities scanner.
Implementation ownership and algorithms belong in the architecture documents.

## Session and reference language

**Trading date**: The exchange-local date selected from the NYSE schedule.

**Scanner session**: The half-open interval from 04:00 through 20:00 New York
time on the selected trading date, including premarket and after-hours.

**Prior close**: The adjusted close of the immediately preceding completed
regular trading session, including calendar and early-close handling.

**Eligible universe**: Active U.S. common shares and common-stock American
depositary receipts admitted by the reference policy.

**Float**: Provider-reported public free float in shares, with source and date
provenance. It is not shares outstanding.

## Observation language

**Aggregate**: One second of reported OHLC prices, share volume, VWAP, and
average trade size for one stock.

**Committed watermark (`T`)**: The market-time boundary through which one
coherent population evaluation has been committed. It is not wall-clock time
or the latest event received.

**Coverage**: Evidence that each second in an interval has an accepted aggregate
or is explicitly proven absent, without unresolved conflict.

**No-print evidence**: Complete coverage showing no accepted aggregate for a
stock in an interval. It does not imply a zero price or a synthetic bar.

**Hydration**: Acquisition of historical aggregates to establish coverage for
live startup or an observed gap. It is not historical session replay.

## Selection and measurements

**Qualification**: Evidence that a stock passed the 60-second activity criteria
during the session. Correctable evidence can be revoked; finalized evidence
persists through later quiet trading.

**Qualified ranking**: Up to 20 qualified, rankable stocks ordered by From Close
percentage descending, with symbol order breaking ties.

**Partial ranking**: An explicitly incomplete ordering of currently trusted
marks when full population or qualification completeness is not proved.

**From Close**: Percentage return from the adjusted prior regular-session close.
_Avoid_: Day return when the reference price is ambiguous.

**From Open**: Percentage return from the open of the first accepted aggregate
in the scanner's extended session.

**Day Range**: The latest mark's relative position between session low and high.

**Activity 30s**: The empirical percentile of current 30-second share-volume
activity against 55 preceding reference windows. It is not a probability.

**Move 30s**: Signed percentage return between trusted marks 30 seconds apart.

**Tape Speed (Tape 5s)**: Distinct qualifying original trades per second over
the preceding five seconds.

**Spread**: The latest valid two-sided bid–ask separation, expressed in cents
and basis points. A quote older than two seconds at `T` is stale.

**Backend readiness**: The ability to publish a current, reconciled aggregate
ranking with valid accounting. Trade/quote enrichment may be unavailable
independently.
