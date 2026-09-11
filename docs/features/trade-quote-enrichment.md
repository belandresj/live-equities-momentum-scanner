# Trade and quote enrichment

Trade/quote data enriches exact qualified rows with Tape 5s and Spread. It is a
subordinate, pressure-sheddable path: it cannot qualify, exclude, rank, or make
the aggregate backend ready.

## Desired membership and coverage

The engine desires T/Q only for symbols in the exact `qualified_current` top
20. Subscription commands are ordered one-shot intents. A successful socket
write establishes requested provider membership and a causal boundary; it does
not establish data coverage.

Trade coverage begins with the first valid post-boundary trade. Quote coverage
begins independently with the first valid post-boundary quote. Before those
facts, the corresponding field is warming or unavailable. A drop,
unsubscribe, quarantine, pressure shed, or superseding epoch closes coverage
immediately and requires new post-boundary data to warm again.

## Tape 5s

```text
Tape 5s = distinct qualifying original trades in [T-5s,T) / 5
```

Trade time is participant execution time with explicit SIP fallback. A trade
must have a complete reviewed identity, positive economic size and price, an
allowed condition basis, and original-sale lifecycle meaning. Exact duplicates,
unknown/unreviewed conditions, corrections, cancels, and other lifecycle
adjustments do not enter the qualifying-original count.

During continuous data-confirmed trade coverage, no qualifying prints in the
window is a genuine numeric zero. Without continuous coverage it is unavailable
or warming, never a fabricated zero. The public product has no one-second Tape
field.

## Spread

Spread uses the latest received quote in accepted causal order. Quote age is
measured against `T`, with negative age clamped to zero when the quote is newer
than the committed watermark:

```text
cents        = 100 * (ask - bid)
basis_points = 10,000 * (ask - bid) / ((ask + bid) / 2)
```

A valid quote is two-sided and non-crossed. Locked quotes are valid zero spread;
one-sided quotes are unavailable; crossed quotes are invalid. A valid quote
older than two seconds at `T` is stale. This is the latest quote, not a rolling
median.

## Retention and late data

Current-epoch trades and quotes more than 30 seconds older than committed `T`
are counted and ignored. This short T/Q policy is distinct from the 16-minute
aggregate correction horizon.

Trade contributions are retained from the later of `T-5s` and monotonic engine
time minus 30 seconds. Equality at the cutoff remains retained. The engine-time
bound prevents a stalled watermark from pinning future-to-`T` trades forever.
Trade duplicate fingerprints are retained through first receipt plus 30 seconds
and removed only on a later engine tick. Quotes retain the latest and
latest-valid facts, not a full quote history. Explicit per-symbol and global
count/byte bounds apply independently of these age limits.

## Watermark-stale visibility

Aggregate-watermark staleness alone does not unsubscribe T/Q or erase its
coverage. The API masks tape and spread as warming while bounded ingestion
continues. Visibility returns after both five elapsed recovery seconds and
five seconds of committed-watermark advancement. A current spread additionally
requires a quote at or after the recovery boundary. Genuine transport loss,
pressure containment, or channel distrust still closes coverage normally.

## Pressure modes

| Mode | Behavior |
| --- | --- |
| `normal` | Desired T/Q is subscribed and processed subject to field coverage. |
| `taq_degraded` | T/Q work may be shed and coverage is withdrawn while aggregate ingestion continues. |
| `aggregate_only` | Selected-row T/Q is unsubscribed/contained; aggregates and control facts continue. |

Severe capacity loss, accounting loss, or retention-bound loss
can enter aggregate-only immediately. Queue/age pressure uses sustained samples
to avoid flapping. Restoration also requires sustained healthy evidence and
fresh subscription/data warm-up.

## Independence and accounting

Per-channel accepted, duplicate, rejected, dropped, pressure-shed, retention,
coverage, command, and projection counters are closed and observable. A mixed
frame with invalid or shed T/Q still delivers every later aggregate and control
fact. T/Q publication may be cadence-coalesced, but trust transitions publish
immediately and aggregate `T` never advances because of a T/Q event.

