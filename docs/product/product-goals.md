# Product goals

**Status:** Approved product contract; feature-set revision approved 2026-08-14;
Day-%/From-Open presentation revision approved 2026-08-17.

**Approved:** 2026-08-05

**Revised:** 2026-08-14 — preserves the eligible universe, aggregate
qualification latch, exact qualified Day-% ranking, top-20 limit, lifecycle,
availability, recovery, and ownership model while replacing the displayed
context/location/momentum feature set. Lower-level component contracts and the
implementation still describe the previously accepted feature set until they
are reconciled through the current capability sequence.

**Presentation revised:** 2026-08-17 — Day % and From Open % use value-relative
green scales computed from the exact displayed snapshot; the dashboard labels
Day % as `FROM CLOSE %` and Tape 5s as `TAPE SPEED`, with short pointer-hover
header descriptions and no data-cell hover tooltips. These changes affect no
market value, qualification, ranking, API field, readiness fact, or backend
ownership.

**Current delivery profile:** The owner-approved
[`Live feature-set MVP program`](../live-feature-mvp-program.md) makes the
ordinary fresh-start live scanner the only required operating path for the
interview MVP. Replay remains present but unverified and non-gating;
checkpoint persistence remains present but disabled and non-gating. Neither
may fabricate revised-feature compatibility.

**Scope:** User-facing behavior, product priorities, version 1 outcomes, and
developer-facing product constraints. Provider mappings, internal component
design, and deployment mechanics belong in later specifications.

This document is the highest product-level authority in the repository.
Architecture and component specifications may explain how to implement these
decisions but may not silently change them.

## 1. Product statement

Live Equities Momentum Scanner is a real-time market-data discovery tool for a
discretionary U.S. equities momentum trader. It answers:

1. Which eligible stocks with a sufficiently demonstrated aggregate tape have
   gained the most from the adjusted previous regular-session close?
2. What are each displayed stock's public float, cumulative session share
   volume, and latest price?
3. Where is each displayed stock relative to its first session print and
   current-session range?
4. Is 30-second share-volume participation accelerating, and is price moving
   directionally with it?
5. For displayed leaders, how quickly are condition-qualified original trades
   printing and what quoted spread is currently being shown?

The scanner provides measurements and situational awareness. It does not
produce a recommendation, forecast, entry, exit, order, or claim of executable
expectancy.

## 2. Intended user and workflow

The intended user is a discretionary momentum trader monitoring U.S. equities
from premarket through the regular and postmarket sessions, 04:00 through 20:00
America/New_York.

The user's primary workflow is:

```text
find the qualified Day-% leaders
  -> inspect Float, cumulative session Volume, Last, and mark age
  -> distinguish gap context and front-side/backside location using From Open
     and Day Range
  -> compare Activity 30s with signed Move 30s
  -> inspect Tape 5s and Spread
  -> form an independent trading decision using charts, news, and other tools
```

The scanner must present the measurements needed for this workflow without
turning them into directional labels or trading instructions.

## 3. Product priorities

When goals conflict, version 1 uses this priority order:

1. **Aggregate ranking correctness.** Preserve accepted one-second aggregates,
   qualification, Day-% ordering, and the committed ranking clock.
2. **Honest availability.** Never present partial, stale, warming, or unknown
   data as complete and current.
3. **Operational continuity.** Recover quickly from process and transport
   failures without fabricating market state.
4. **Aggregate-derived context and momentum.** Preserve session Volume, From
   Open, Day Range, Activity 30s, and Move 30s independently where their
   required history is trustworthy.
5. **Trade/quote enrichment.** Normally provide T/Q-derived features for all
   displayed rows, but degrade them before risking aggregate correctness.
6. **Presentation continuity.** Permit UI development and deployment without
   interrupting the scanner backend.

## 4. Core table contract

### PG-UNIVERSE-01 — eligible securities

The eligible universe contains only active provider records with
`market=stocks`, `locale=us`, and security type `CS` or `ADRC`. Exclude ETFs,
ETNs, preferreds, units, warrants, rights, funds, inactive records, non-U.S.
records, and unknown types.

Exact current ranking requires a current-run-date universe binding. A validated
prior-date universe cache may keep the process observable but cannot be labeled
current.

### PG-REFERENCE-01 — adjusted prior close

Use the finite positive adjusted close from the immediately preceding completed
regular trading session determined by the exchange schedule. The prior-close
dataset must identify that exact session date and adjustment policy.

A missing or invalid close makes only that symbol unrankable and separately
counted. Never substitute today's open, an unadjusted close, or an older
unidentified session. A validated cache for the exact required prior-session
date is equivalent in product meaning to a fresh retrieval.

### PG-REFERENCE-02 — public free float

Float means public free float in shares: shares considered available for public
trading after excluding strategic, controlling, restricted, locked-up, and
other non-tradable holdings under the provider's documented methodology. It is
not `share_class_shares_outstanding`, weighted shares outstanding, or another
outstanding-share proxy.

The initial source is Massive's dedicated
[Stocks Float dataset](https://massive.com/docs/rest/stocks/fundamentals/float).
Each usable fact retains exact symbol, finite positive `free_float`,
`effective_date` when supplied, provider/source identity, retrieval time, and
cache provenance. The source is experimental and disclosure-derived rather
than real-time; the scanner must preserve that provenance and must not imply
intraday freshness.

Float is optional, display-only reference enrichment. A bounded startup
retrieval may populate an immutable symbol lookup and may fall back to the last
validated cached value when refresh fails. Missing, malformed, duplicate,
ambiguous, or unavailable Float affects only that symbol's Float field. It
cannot change the session binding's market meaning, qualification, rank,
backend readiness, or live processing. A retained older value must remain
visibly dated or stale rather than silently current.

### PG-RANK-01 — adjusted-prior-close basis

Day return is based on the valid adjusted previous regular-session close.
Today's open is not a substitute because it omits the overnight and premarket
move.

A symbol without the required prior close is not rankable. Missing or invalid
prior-close state must be visible in population accounting rather than silently
folded into another exclusion.

### PG-RANK-02 — current-session mark

A rankable symbol requires a trusted accepted current-session aggregate mark at
the committed ranking watermark. The scanner must not synthesize a mark,
forward-fill a price from another session, or infer a price from an empty
historical response.

The mark must have a finite price of at least USD 0.25, trustworthy same-session
aggregate context, and nonnegative causal age at the committed watermark.

A fully reconciled successful empty historical interval establishes
`no_print_through(T)`. That is a resolved, currently unrankable state at `T`, not
unfinished global recovery work. A later accepted live aggregate gives the
symbol a real mark and triggers ordinary reevaluation.

### PG-RANK-03 — qualification before ranking

The contracted table contains symbols that have passed the approved
same-session aggregate qualification. Qualification is evaluated before
ordering and truncation; a high raw-Day-% symbol that has not qualified cannot
occupy a row or leave an avoidable hole above a qualified lower-return symbol.

At committed aggregate watermark `T`, evaluate `[T-60s,T)` and define:

```text
N    = accepted distinct one-second aggregate count
G    = longest run of missing whole-second slots, including both edges
A60  = sum(Volume / AverageTradeSize) over [T-60s,T)
A5   = sum(Volume / AverageTradeSize) over [T-5s,T)
DV60 = sum(Volume * VWAP) over [T-60s,T)
C    = maximum one-second Volume / total Volume over [T-60s,T)
```

Qualification requires all of:

```text
latest_price >= USD 0.25
N >= 45
G <= 3 seconds
A60 available and >= 1,000
DV60 >= USD 250,000
A5 available and >= 100
C <= 0.50
```

`A60` and `A5` are available only when every contributing aggregate has a
finite positive `AverageTradeSize`. Missing market seconds are inactivity and
are not fabricated. Invalid inputs fail the gate.

The first passing window creates a same-session qualification latched through
20:00 ET. A correction may revoke a still-mutable sole proof. Once a passing
proof is finalized beyond the approved aggregate correction horizon, later
quiet tape does not revoke it. Core rankability continues to apply.

### PG-RANK-04 — exact ordering and cardinality

All qualified rankable symbols are ordered by Day return descending, with exact
symbol ascending as the deterministic tie-breaker. The scanner publishes the
first 20.

If fewer than 20 symbols qualify, the table contains fewer than 20 rows. It must
not fabricate rows, weaken qualification, or duplicate symbols to fill the
table.

No Float, Volume, From Open, Day Range, Activity 30s, Move 30s, T/Q, pressure,
or presentation field may become an implicit qualification rule or secondary
ranking key.

### PG-RANK-05 — complete-population honesty

The scanner distinguishes exact qualified ranking, explicitly partial degraded
ranking, and unavailable or globally suppressed ranking.

`degraded_bootstrap` is a permanent user-facing fallback that is visually and
semantically distinct from the qualified table. It may publish only when:

- the universe and prior-close binding is current;
- aggregate transport and the committed watermark are current;
- every returned row has a trusted mark and valid prior close; and
- publication is causally fenced through accepted ingress.

It orders currently known rankable marks by raw Day % descending and exact
symbol ascending and exposes the exact covered population and unresolved
categories. It must state that omitted symbols may still qualify or outrank a
displayed row. It cannot be labeled qualified, cannot establish qualification
completeness, and cannot permit new T/Q subscriptions.

Zero trusted marks or global binding/canonical ambiguity remains unavailable
rather than an empty degraded table.

## 5. Version 1 displayed information

Version 1 includes the following fields with the product meanings and ranking
effects defined here. Focused feature specifications may settle subordinate
measurement and provider-mapping details but cannot change these meanings.

The visible columns appear in this exact order and grouping:

```text
CONTEXT                  LOCATION                     CURRENT MOMENTUM     EXECUTION
SYMBOL FLOAT VOLUME LAST | FROM CLOSE % FROM OPEN % DAY RANGE | ACTIVITY 30s MOVE 30s | TAPE SPEED SPREAD
```

| Field | User question answered | Ranking effect |
| --- | --- | --- |
| Symbol | Which exact listed security is this? | Exact symbol breaks Day-% ties. |
| Float | How constrained is the stock's effective publicly tradable share supply? | Display only. |
| Volume | How many shares have traded during the scanner session? | Display only. |
| Last | What is the latest trusted aggregate mark? | Required for rankability. |
| From Close % | How far is Last from the adjusted previous regular-session close? | Sole numeric ordering field after qualification. |
| From Open % | How far is Last from the first eligible session trade represented by canonical aggregates? | Display only. |
| Day Range | Where is Last inside the trustworthy current-session low/high range? | Display only. |
| Activity 30s | How unusual is current 30-second share-volume participation versus the immediately preceding five-minute regime? | Display only. |
| Move 30s | What signed price return occurred over the trailing 30 seconds? | Display only. |
| Tape Speed | How quickly are distinct condition-qualified original trades printing now? | Display only. |
| Spread | What is the latest valid quoted spread for the displayed leader, and how old is that quote? | Display only. |

Row order itself expresses rank. A versioned API may carry an explicit rank
ordinal for validation and accessibility, but the dashboard does not add a
visible Rank column to the table above.

Float, aggregate context/momentum, and T/Q-derived fields are not prerequisites
for displaying an otherwise trustworthy qualified Day-% row unless a later
owner-approved product change explicitly makes a field part of qualification.

The product deliberately keeps participation, direction, immediate tape tempo,
execution cost, and session location separate: Activity 30s measures relative
share-volume participation; Move 30s measures signed price response; Tape 5s
measures immediate absolute qualifying-print tempo; Spread measures execution
friction; and Day Range measures front-side/backside location. None is a hidden
composite score.

### PG-FEATURE-01 — price and session-location formulas

At committed watermark `T`:

```text
Day % = 100 * (Last / adjusted_prior_close - 1)

From Open % = 100 * (Last / first_session_aggregate_open - 1)

Day Range = 100 * (Last - session_low) / (session_high - session_low)
```

`first_session_aggregate_open` is the Open of the earliest accepted aggregate
beginning in `[S,T)`, where `S` is 04:00 America/New_York. It is the
provider-independent full-universe representation of the first eligible trade
in that one-second aggregate; the scanner does not wait for selected-row raw
T/Q coverage or substitute an assumed 04:00 price. Day Range uses trustworthy
session extrema over `[S,T)`.

A missing, incomplete, invalid, or zero-width session range is unavailable
rather than clamped or reported as zero. From Open and Day Range are
independent of qualification and ranking after their own required history is
established.

### PG-FEATURE-02 — cumulative session Volume

Volume is cumulative accepted canonical aggregate economic share volume over
the scanner session:

```text
Session Volume(T) = sum(aggregate Volume_i), for window_start in [S,T)
```

Known no-print seconds contribute zero without creating synthetic bars.
Unknown coverage, a historical conflict, or invalid aggregate volume makes the
field unavailable or invalid rather than reporting a partial session total.
Accepted inserts, revisions, and withdrawals update the result exactly.
Internal precision preserves provider-supported fractional economic quantity;
display formatting may use compact share units but cannot feed back into state.

### PG-FEATURE-03 — Activity 30s

Activity 30s measures relative share-volume participation only. It does not
use Average Trade Size, transaction estimates, range expansion, volatility, or
price direction.

At committed watermark `T`, define the target rate:

```text
V30(T) = sum(aggregate Volume over [T-30s,T)) / 30
```

Define 55 reference endpoints at five-second spacing:

```text
r_k = T - 30s - 5s*k, for k = 0..54
V30(r_k) = sum(aggregate Volume over [r_k-30s,r_k)) / 30
```

The reference inputs therefore cover exactly `[T-330s,T-30s)` and never
overlap the current target `[T-30s,T)`. With the complete reference set:

```text
Activity 30s = 100 * count(V30(r_k) <= V30(T)) / 55
```

Ties are inclusive. Known no-print seconds are genuine zero share volume for
this product calculation without becoming fabricated aggregate bars. The
field is `warming` until the complete target and five-minute reference regime
can be evaluated, `unavailable` when required coverage is unknown, and
`invalid` for conflicting or invalid contributing evidence. The finite result
is bounded to `[0,100]` and is a descriptive empirical percentile, not a
probability.

### PG-FEATURE-04 — Move 30s

Move 30s is a signed aggregate-mark return, not volatility:

```text
Move 30s = 100 * (P(T) / P(T-30s) - 1)
```

`P(b)` is the latest trusted accepted canonical aggregate Close strictly
before boundary `b`. A completely known no-print interval carries the last
actual mark to the boundary for this calculation; it does not manufacture a
bar or a trade. If no prior actual mark exists, or an unknown/conflicted
interval could hide a later mark before either boundary, Move 30s is warming,
unavailable, or invalid as appropriate. Accepted aggregate corrections update
both boundary marks exactly.

Move 30s uses full-universe aggregate history. It must not use selected-row BBO
midpoint or T/Q coverage, because selection-dependent input would make the
field unavailable merely because a symbol had just entered the top 20.

### PG-FEATURE-05 — Tape 5s

Tape 5s reports distinct condition-qualified original trades per second over
`[T-5s,T)`. Use participant execution time with explicit SIP-time fallback.
Exclude exact duplicates, incomplete identities, unknown or unreviewed
conditions, known non-volume records, and lifecycle-adjustment records from the
qualifying-original basis. Unequal repeats or incomplete lifecycle semantics
must be disclosed rather than treated as exact corrected tape.

```text
Tape 5s = distinct qualifying original trade count in [T-5s,T) / 5
```

Genuine silence during continuous acknowledged trade coverage is numeric zero.
The product does not expose a separate one-second Tape burst field.

The focused feature specification must define the evidenced condition fixtures,
identity rules, and any attention threshold without changing this measurement
meaning.

### PG-FEATURE-06 — Spread

Spread reports the latest valid two-sided non-crossed NBBO quote spread in
cents and basis points together with quote age. Locked quotes are valid zero
spread. During acknowledged continuous quote coverage, quiet periods retain
the last valid numeric spread and increasing age; an old quote is explicitly
`stale` rather than replaced by an absent value. A latest one-sided quote is
unavailable and a latest crossed quote is invalid, so neither can fabricate a
valid spread.

The focused feature specification must define its evidenced quote validation
and stale-age boundary without changing this measurement meaning.

### PG-FEATURE-07 — corrections, coverage gaps, and preselection retention

Accepted aggregate corrections and out-of-order inserts inside the approved
correction horizon update every dependent aggregate feature exactly. T/Q
coverage gaps clear current T/Q measurements and require fresh warm-up.

Every symbol capable of entering a qualified or permitted degraded displayed
set retains canonical aggregate and exact coverage evidence sufficient to
evaluate at least `[max(S,T-330s),T)` before selection. Ranking membership must
not initiate, backfill, or alter aggregate history. Consequently, a symbol
that enters the top 20 after spending more than five minutes outside it can
immediately evaluate Activity 30s and Move 30s when its all-symbol aggregate
coverage is trustworthy; only its newly selected T/Q fields warm from fresh
acknowledged coverage.

Retention means accepted aggregates plus exact present, proven-absent,
unknown, and conflict evidence for the interval, together with the latest
trusted actual predecessor mark at or before the interval floor when one
exists. It does not require 330 printed aggregates and does not permit
fabricated zero-volume bars. The focused canonical-state contract must
preserve this guarantee across compaction, a stalled committed watermark,
fresh hydration, same-process recovery, and ordinary live evaluation.
Optional replay or checkpoint support cannot claim compatibility until it
separately proves the same guarantee for its path.

The full-universe pass owns qualification, trusted marks, Day %, and exact
top-20 selection. Display-only Activity 30s and Move 30s may be materialized
only after selection, but their result must be identical to evaluation from the
same preselection canonical state and committed `T`. Selection cannot create,
discard, backfill, warm, or otherwise change their aggregate inputs.

## 6. Independent field availability

### PG-AVAIL-01 — explicit states

Unavailable, warming, current, stale, and invalid are distinct product states.
None may be represented as a genuine numeric zero.

Each field must explain its own absence or degradation using a bounded,
user-meaningful reason. A field may be current while another field for the same
symbol is warming or unavailable.

### PG-AVAIL-02 — reference and aggregate-history independence

Missing or stale Float, or incomplete history for Volume, From Open, Day Range,
Activity 30s, or Move 30s, affects only the dependent field. It does not
globally block a trusted Last and Day-% row, qualification, or exact Day-%
ordering.

### PG-AVAIL-03 — T/Q independence

T/Q entitlement, subscription, decoding, measurement, coverage, or capacity
failure affects T/Q-derived fields only. It cannot suppress or reorder
aggregate ranking or invalidate aggregate-derived fields.

After dropped or unsubscribed T/Q coverage, affected measurements warm again
from new covered events. The product must not silently bridge the gap. Quiet
acknowledged coverage is not a gap: Tape 5s advances to genuine numeric zero
when its trailing window contains no qualifying trades, while Spread retains
the last valid quote with increasing age and stale status.

## 7. Trade and quote product behavior

### PG-TAQ-01 — normal coverage

Under normal operating capacity, the scanner requests trades and quotes for
every displayed top-20 symbol. If the table contains fewer than 20 rows, it
requests T/Q for every displayed row.

T/Q is therefore a normal version 1 product capability, not an experimental or
normally disabled subsystem.

### PG-TAQ-02 — aggregate-protecting degradation

If direct shared-feed consumption evidence shows that processing load threatens
aggregate timeliness or correctness, the scanner degrades T/Q before aggregate
processing:

1. It may stop expensive T/Q feature processing while continuing to read and
   classify the shared stream.
2. Under sustained pressure, it may unsubscribe some or all T/Q coverage to
   reduce incoming data.
3. It preserves aggregate and provider-control processing throughout.
4. It restores T/Q gradually after aggregate processing has recovered, with
   fresh measurement warm-up.

The T/Q component specification must set thresholds and restoration timing from
queue occupancy, oldest-unread-frame age, actual capacity loss, T/Q retention
bounds, and transport/accounting loss evidence while preserving this priority
order. Heap size, goroutine count, generic engine-delivery latency, delivery-
family attribution, and quiet/no-message sampling windows are diagnostics only;
they cannot enter, prolong, or reset T/Q pressure state.

### PG-TAQ-03 — selected-population scope

Version 1 T/Q-derived features apply to symbols already selected by aggregate
qualification and ranking. T/Q is not used to choose or order the full-universe
top 20.

## 8. Extensible feature product goal

The scanner must permit focused new features derived from aggregates, trades,
quotes, or combinations of them without redesigning the central scanner
lifecycle.

Every proposed feature must state:

- the trader question it answers;
- the market-data inputs and coverage it requires;
- its window and currentness semantics;
- its behavior across corrections and data gaps;
- whether it affects qualification/ranking or is display-only; and
- its bounded retained-state requirements.

This is an extensibility requirement, not approval for a generic plugin system.
Features remain explicit product decisions. A T/Q-derived feature available
only for selected symbols cannot silently become a full-universe ranking input.

## 9. Production continuity and optional checkpoints

### PG-OPS-01 — fresh-start restart support

The required process-restart path for the live feature-set MVP is fresh
reference resolution and aggregate hydration through the ordinary ingress
fence. Until that work is terminal, the scanner reports warming, unavailable,
or degraded state honestly; it cannot present partial session Volume, From
Open, Day Range, Activity 30s, Move 30s, qualification, or population coverage
as current.

Checkpoint persistence is optional and disabled by default. Existing
checkpoint code may remain for later evaluation, but checkpoint performance,
repair, migration, and revised-feature restart equivalence are not version 1
or live-MVP gates. A checkpoint-enabled launch must fail clearly unless its
schema and installation proof cover every active product field at one coherent
committed timestamp. It may not combine a newer mark with older supposedly
complete session context or silently omit revised state.

Fresh reconstruction latency must be measured honestly when claimed, but this
product contract sets no checkpoint-relative restart target.

### PG-OPS-02 — recovery does not freeze ordinary evaluation

Fresh bootstrap and same-process gap recovery feed the same canonical state
and ranking path used during ordinary operation. Once all
required hydration work is terminal—including successful empty results—the
scanner returns to ordinary live evaluation even if the candidate population is
sparse.

There must be no terminal recovery state that indefinitely freezes the ranking
watermark while accepted aggregate data continues arriving.

## 10. Optional, unverified aggregate replay

### PG-REPLAY-01 — retained implementation without an MVP claim

Replay code and tooling may remain available for later evaluation, but the
live feature-set MVP makes no claim that replay currently starts, completes
within a useful time, reproduces the revised fields, or provides a supported
user workflow. Replay repair, performance, deletion, and revised-feature
equivalence are explicitly outside the MVP.

Deterministic in-memory event traces, fake-provider inputs, HTTP fixtures, and
UI fixtures remain valid engineering proofs. They are not product replay and
do not imply that the replay runtime works.

If replay is later claimed, it requires a separate owner-approved contract and
proof for the then-current feature set. Historical replay cannot by itself
prove live receipt latency, transport behavior, correction-arrival chronology,
or T/Q behavior.

### PG-REPLAY-02 — T/Q replay deferred

Full-session and two-pass T/Q replay are not version 1 requirements. The future
design idea may be retained for later T/Q feature development, but it must not
expand the version 1 build or test scope.

## 11. API and UI product boundary

### PG-UI-01 — independent deployment

The backend computes market state, ranking, features, coverage, and readiness
without a browser. The UI consumes a versioned read-only API and owns
presentation and interaction only.

Compatible UI changes must be deployable while the scanner backend remains
live. Market calculations, ranking decisions, and readiness cannot be owned by
the browser.

### PG-UI-02 — stable product meaning

The API and UI must preserve the product meaning of fields and availability
states across compatible releases. Internal implementation structures do not
automatically define the public API.

The UI update transport and the amount of predecessor UI code to reuse remain
architecture and implementation decisions after the snapshot contract is
approved.

### PG-UI-03 — visual hierarchy

The dashboard uses color sparingly and consistently with the column groups:

- Symbol, Float, Volume, and Last are predominantly neutral context.
- Day % uses a readable value-relative green treatment across the exact rows in
  the displayed snapshot. For `n` displayed rows, let `D_max` and `D_min` be
  the maximum and minimum displayed Day-% values. When `D_max > D_min`, row
  `i` has presentation position `c_i = (D_i-D_min)/(D_max-D_min)`. The scale is
  value-relative, not rank-relative: a row close in value to the displayed
  minimum remains visually close to that endpoint even when its ordinal rank
  is high. `c=0` maps to restrained dark green `#4A965D`; `c=1` maps to neon
  green `#2CFF05`. If all displayed values are identical, every row uses
  `c=0.5`. Tied values always receive the same position. Fewer than 20 rows use
  only their own extrema; empty snapshots have no scale.
- From Open % uses the same continuous displayed-set-relative green scale as
  Day %. For the displayed rows whose `from_open_change.status` is `current`
  and whose `value_ratio` is finite, let `F_max` and `F_min` be the maximum and
  minimum ratios and assign `c_i = (F_i-F_min)/(F_max-F_min)` when the range is
  nonzero. The position is value-relative, not rank-relative or zero-relative:
  it compares the displayed From Open values with one another. `c=0` is
  restrained dark green `#4A965D`; `c=1` is neon green `#2CFF05`; interpolation
  is continuous in CSS `oklab`. Fewer than 20 rows use only the eligible values
  actually displayed, ties receive the same position, and one eligible value or
  an identical eligible range receives `c=0.5`. Warming, unavailable, invalid,
  and other non-current values do not participate and receive no position; an
  empty eligible set receives no positions. A degraded, disconnected, frozen,
  retained, or otherwise noncurrent table keeps its existing noncurrent color
  even when a current field position exists in the view model. The displayed
  text and underlying ratio are unchanged; this is a frontend presentation
  transform with no API, backend, ranking, readiness, or market-measurement
  ownership change.
- Day Range is the strongest location cue, progressing from low/red-neutral to
  high/green.
- Activity 30s uses a gray-to-amber-to-bright-orange attention scale as its
  relative participation percentile rises. Tape 5s uses a separate continuous
  absolute-rate neutral-to-orange RGB gradient: `0` and `50` trades/s map to
  `#8F9AA3`, `100` to `#B8793E`, `250` to `#E98212`, and `500` to `#FF8A00`;
  RGB channels are linearly interpolated between anchors, rates above `500`
  clamp to `#FF8A00`, and the scale is not normalized against displayed rows.
- Move 30s uses a continuous signed red-to-gray-to-green RGB gradient based on
  the displayed 30-second return. The percentage-point anchors are `-7%`
  `#FC0000`, `-5%` `#EF3030`, `-2%` `#C46B6B`, `0%` `#8F9AA3`, `+2%`
  `#70B873`, `+5%` `#45E532`, and `+7%` `#2CFF05`; each RGB channel is
  linearly interpolated between adjacent anchors, values outside the endpoints
  clamp to the nearest endpoint, and only the numeric text is colored.
- Spread uses a continuous absolute-bps neutral-to-neon-red RGB gradient for
  execution friction: `0` and `10` bps map to `#8F9AA3`, `25` to `#B06F6F`,
  `50` to `#D84A4A`, `75` to `#EE2525`, and `100` to `#FC0000`; RGB channels
  are linearly interpolated between anchors, values above `100` bps clamp to
  `#FC0000`, and only the numeric text is colored. Tight spreads remain
  neutral rather than green.

The visual grammar is: green means favorable location, orange means something
is happening now, and red means execution friction or unfavorable state. Color never changes ranking,
field availability, or the displayed numeric value, and every meaning remains
available without color alone.

Day-% and From-Open normalization are presentation transforms over one already
validated, server-ordered snapshot. They must not sort, filter, backfill, or
otherwise alter the displayed rows, and they are not added to the snapshot API.
The client clamps only computed presentation positions to `[0,1]` for floating-
point residue; it does not clamp, round, or rewrite the underlying ratios or
displayed percentages. A missing or nonfinite Day % remains an invalid product
row under the existing ranking and API contract, while a non-current From Open
field is omitted from the From Open visual scale without losing its state,
reason, or em-dash presentation.

## 12. Accounting and observability goals

### PG-OBS-01 — complete symbol accounting

Every symbol in the configured universe must be attributable to one mutually
exclusive primary state at publication time:

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

Qualification, Float availability, session Volume/history availability,
Activity 30s, Move 30s, T/Q coverage, and location diagnostics are overlapping
dimensions and must be reported separately rather than used to break the
primary identity.

### PG-OBS-02 — complete work accounting

Historical work must satisfy:

```text
planned
  = completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Successful empty work is not reported as pending or unresolved.

### PG-OBS-03 — trader-visible trust

The product must expose enough bounded status to distinguish:

- connected transport from current committed evaluation;
- current aggregate ranking from stale ranking;
- aggregate correctness from T/Q availability;
- normal no-print sparsity from failed/unknown data; and
- successful fresh hydration or recovery from incomplete work.

The product exposes four separate concepts:

1. **Process live:** the backend is running and can report status. This makes no
   claim about market-data correctness.
2. **Backend ready:** the session binding is valid, aggregate transport and
   committed processing are current, accepted ingress is processed through the
   publication fence, and the API can serve a coherent current snapshot. A
   resolved legitimately empty qualified table may be ready.
3. **Ranking current:** the published ranking watermark matches the current
   causal target and its integrity is trustworthy for its explicitly named
   status. `qualified_current` is exact. `degraded_bootstrap` may be current as
   a raw partial view but is not qualified.
4. **Field/T/Q current:** the individual measurement has complete trustworthy
   coverage for its own required window. Independent warming or unavailable
   historical/T/Q fields do not make an otherwise current aggregate ranking or
   backend unready.

Disconnected, stale, fence-lagged, binding-mismatched, globally suppressed, or
unresolved zero-mark bootstrap state is not backend ready. T/Q pressure or
unavailability alone does not change backend readiness. Architecture and
operations specifications must define exact evidenced timing thresholds and
HTTP mappings while preserving these meanings.

## 13. Version 1 non-goals

Version 1 does not include:

- order entry, broker integration, automated execution, or trade management;
- predictive signals, price targets, entries/exits, or expectancy claims;
- full-universe trade/quote ranking inputs;
- full-session or two-pass T/Q replay;
- a supported or performance-validated aggregate replay workflow;
- checkpoint persistence, checkpoint performance, or checkpoint restart
  equivalence as a release requirement;
- visible Float Turnover or another Float-derived score;
- the superseded transaction/price-expansion Activity composite, HOD drawdown,
  30-minute range position, 60-minute range position, or one-second Tape burst;
- a generic user-installed feature/plugin framework;
- a database or raw-event journal without a separately approved need;
- a generalized event bus, microservice architecture, or per-symbol worker
  hierarchy;
- silent fallbacks that fabricate marks, history, qualification, or T/Q
  continuity; or
- speculative edge-case behavior unsupported by evidence or explicit approval.

## 14. Product-level acceptance

Version 1 is product-complete only when reviewed evidence demonstrates that:

- the full eligible population is accounted for without residual categories;
- successful no-print hydration does not block live ranking;
- qualification filters before exact Day-% ordering and top-20 truncation;
- fewer-than-20 and exact-symbol tie behavior are deterministic;
- accepted live aggregates continue advancing the committed ranking during
  sparse premarket conditions;
- Float and every aggregate context/momentum field degrade independently from
  Last, qualification, and Day-% ordering;
- session Volume equals the correction-aware sum of accepted aggregate share
  volume from `S` through `T` and never presents partial history as complete;
- a symbol outside the displayed set for more than five minutes retains exact
  all-symbol evidence, then enters the top 20 with immediately correct Activity
  30s and Move 30s while Tape 5s and Spread alone warm from selected coverage;
- known no-print intervals carry the last actual aggregate mark for Move 30s
  without creating a synthetic bar;
- missing Float is unavailable and a cached Float retains its effective date
  and stale/provenance status without substituting shares outstanding;
- all displayed rows normally receive T/Q coverage;
- T/Q overload can degrade to zero without changing aggregate ranking;
- a fresh process start reconstructs the revised aggregate-derived state
  through exact hydration and does not claim readiness before the ingress fence;
- checkpoint-on operation cannot silently claim compatibility with the revised
  feature set, while checkpoint-off is the supported MVP configuration;
- the visible dashboard contains only Symbol, Float, Volume, Last, From Close %,
  From Open %, Day Range, Activity 30s, Move 30s, Tape Speed, and Spread in the
  contracted grouping/order;
- the dashboard's neutral/green/orange/red hierarchy communicates context,
  location, current activity, and execution friction without browser-owned
  market classification or color-only meaning;
- the backend operates without the UI and compatible UI releases do not restart
  it; and
- unknown replay capability remains disclosed and does not block or falsely
  strengthen the live-product acceptance claim;
- no outcome is described as trading edge or executable expectancy merely
  because scanner correctness tests pass.
