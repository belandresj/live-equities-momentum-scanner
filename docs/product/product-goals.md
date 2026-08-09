# Product goals

**Status:** Approved product contract.

**Approved:** 2026-08-05

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
2. Where is each displayed stock trading relative to its current-session and
   recent price ranges?
3. How active is the stock's recent aggregate behavior relative to its own
   current-session history?
4. For displayed leaders, how quickly are qualifying trades printing and what
   quoted spread is currently being shown?

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
  -> inspect latest price and mark age
  -> distinguish front-side strength from pullback using HOD and ranges
  -> assess recent aggregate activity
  -> inspect current Tape Rate and Spread
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
4. **Aggregate-derived context.** Preserve current-session and rolling features
   independently where their required history is trustworthy.
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

No Activity, range, T/Q, pressure, or presentation field may become an implicit
secondary ranking key.

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

| Field | User question answered | Ranking effect |
| --- | --- | --- |
| Rank | Where does this qualified symbol stand among all passers? | Output of Day-% ordering. |
| Symbol | Which exact listed security is this? | Exact symbol breaks Day-% ties. |
| Last | What is the latest trusted aggregate mark? | Required for rankability. |
| Day % | How far is Last from the adjusted previous regular-session close? | Sole numeric ranking field. |
| From 4AM % | How far is Last from the first trustworthy current-session aggregate open? | Display only. |
| HOD drawdown | How far is Last below the trustworthy current-session high? | Display only. |
| Day range position | Where is Last inside the trustworthy current-session low/high range? | Display only. |
| 30-minute range position | Where is Last inside the trailing 30-minute low/high range? | Display only. |
| 60-minute range position | Where is Last inside the trailing 60-minute low/high range? | Display only. |
| Activity | How unusual are recent aggregate-estimated transactions and price expansion relative to the stock's own current-session reference periods? | Display only. |
| Tape Rate | How quickly are accepted qualifying trades printing now? | Display only. |
| Spread | What recent quoted spread is shown for the selected leader? | Display only. |

Historical and T/Q-derived fields are not prerequisites for displaying an
otherwise trustworthy qualified Day-% row unless a later owner-approved product
change explicitly makes a field part of qualification.

### PG-FEATURE-01 — price and range formulas

At committed watermark `T`:

```text
Day % = 100 * (Last / adjusted_prior_close - 1)

From 4AM % = 100 * (Last / first_session_aggregate_open - 1)

HOD drawdown % = 100 * (Last / session_high - 1)

range_position = 100 * (Last - range_low) / (range_high - range_low)
```

`first_session_aggregate_open` is the Open of the earliest accepted aggregate
beginning at or after 04:00 ET. Range position is calculated for session
`[04:00,T)`, rolling `[T-30m,T)`, and rolling `[T-60m,T)` ranges.

A missing, incomplete, invalid, or zero-width range is unavailable rather than
clamped or reported as zero.

### PG-FEATURE-02 — Activity

Activity measures aggregate-estimated transactions and price expansion over
`[T-30s,T)`:

```text
transactions  = sum(Volume / AverageTradeSize)
expansion_bps = 10,000 * ln(max(High) / min(Low))
```

Compare both components with independently eligible completed 30-second blocks
from the 04:00 session boundary through the completed block immediately before
the target. The baseline never resets at regular-hours open or after-hours
open. Require at least 10 reference blocks, use
inclusive empirical percentiles, and report:

```text
Activity = sqrt(transaction_percentile * expansion_percentile)
```

The result is bounded to `[0,100]`. The focused feature specification must
define reference-block eligibility and exact invalid-input handling consistently
with this product meaning.

### PG-FEATURE-03 — Tape Rate

Tape Rate reports unique condition-qualified original trades per second over
the trailing five seconds and the trailing one-second burst rate. Use participant
execution time with explicit SIP-time fallback. Exclude known non-volume and
lifecycle-adjustment records from the initial qualifying-original basis, and
disclose incomplete lifecycle handling rather than implying exact corrected
tape when the required provider semantics are unavailable.

The focused feature specification must define the evidenced condition fixtures,
identity rules, and any attention threshold without changing this measurement
meaning.

### PG-FEATURE-04 — Spread

Spread reports the five-second time-weighted median validated NBBO spread in cents
and basis points. Locked quotes are valid zero spread. Crossed, one-sided,
stale, warming, or insufficient-coverage states are unavailable rather than
negative or fabricated spread.

The focused feature specification must define its evidenced quote validation,
coverage minimum, and weighting boundary without changing this measurement
meaning.

### PG-FEATURE-05 — corrections and coverage gaps

Accepted aggregate corrections and out-of-order inserts inside the approved
correction horizon update every dependent aggregate feature exactly. T/Q
coverage gaps clear current T/Q measurements and require fresh warm-up.

## 6. Independent field availability

### PG-AVAIL-01 — explicit states

Unavailable, warming, current, stale, and invalid are distinct product states.
None may be represented as a genuine numeric zero.

Each field must explain its own absence or degradation using a bounded,
user-meaningful reason. A field may be current while another field for the same
symbol is warming or unavailable.

### PG-AVAIL-02 — historical independence

Incomplete history for From 4AM, HOD, a rolling range, or Activity affects only
the dependent fields. It does not globally block a trusted Last and Day-% row.

### PG-AVAIL-03 — T/Q independence

T/Q entitlement, subscription, decoding, measurement, coverage, or capacity
failure affects T/Q-derived fields only. It cannot suppress or reorder
aggregate ranking or invalidate aggregate-derived fields.

After dropped or unsubscribed T/Q coverage, affected measurements warm again
from new covered events. The product must not silently bridge the gap.

## 7. Trade and quote product behavior

### PG-TAQ-01 — normal coverage

Under normal operating capacity, the scanner requests trades and quotes for
every displayed top-20 symbol. If the table contains fewer than 20 rows, it
requests T/Q for every displayed row.

T/Q is therefore a normal version 1 product capability, not an experimental or
normally disabled subsystem.

### PG-TAQ-02 — aggregate-protecting degradation

If processing load threatens aggregate timeliness or correctness, the scanner
degrades T/Q before aggregate processing:

1. It may stop expensive T/Q feature processing while continuing to read and
   classify the shared stream.
2. Under sustained pressure, it may unsubscribe some or all T/Q coverage to
   reduce incoming data.
3. It preserves aggregate and provider-control processing throughout.
4. It restores T/Q gradually after aggregate processing has recovered, with
   fresh measurement warm-up.

The T/Q component specification must set thresholds and restoration timing from
capacity evidence while preserving this priority order.

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
- its bounded retention and restart requirements.

This is an extensibility requirement, not approval for a generic plugin system.
Features remain explicit product decisions. A T/Q-derived feature available
only for selected symbols cannot silently become a full-universe ranking input.

## 9. Production continuity and checkpoints

### PG-OPS-01 — version 1 restart support

Coherent checkpoints and bounded catch-up are mandatory in version 1. A normal
process restart should restore prior trustworthy session state and retrieve
only the missing interval rather than requiring full-session reconstruction.

Every restored field must describe one coherent committed timestamp. A
checkpoint may not combine a newer mark with older supposedly complete range,
qualification, or Activity state.

The checkpoint specification must set an evidenced cadence and restart target.
Approximately 130 seconds of fresh reconstruction is not an acceptable normal
production-restart objective.

### PG-OPS-02 — recovery does not freeze ordinary evaluation

Fresh bootstrap, checkpoint catch-up, and same-process gap recovery feed the
same canonical state and ranking path used during ordinary operation. Once all
required hydration work is terminal—including successful empty results—the
scanner returns to ordinary live evaluation even if the candidate population is
sparse.

There must be no terminal recovery state that indefinitely freezes the ranking
watermark while accepted aggregate data continues arriving.

## 10. Offline aggregate replay

### PG-REPLAY-01 — version 1 aggregate replay

Version 1 includes tooling to obtain historical one-second aggregates,
normalize them to the same provider-independent aggregate event contract used
by live processing, and replay them through the same scanner state and feature
path under a deterministic simulated clock.

Replay supports product and engineering review while markets are closed. It
must reproduce the same market-time result regardless of playback speed.

Historical REST replay does not, by itself, prove actual live receipt latency,
transport behavior, or correction-arrival chronology. Those claims require
separate evidence.

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

Qualification, Activity, historical availability, T/Q coverage, and range
diagnostics are overlapping dimensions and must be reported separately rather
than used to break the primary identity.

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
- successful restart/catch-up from incomplete recovery.

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
- historical fields degrade independently from Last and Day-%;
- all displayed rows normally receive T/Q coverage;
- T/Q overload can degrade to zero without changing aggregate ranking;
- restart from a coherent checkpoint reproduces the same aggregate-derived
  state before new input and catches up within the approved target;
- aggregate replay produces the same market-time output regardless of playback
  speed;
- the backend operates without the UI and compatible UI releases do not restart
  it; and
- no outcome is described as trading edge or executable expectancy merely
  because scanner correctness tests pass.
