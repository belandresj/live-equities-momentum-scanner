# Phase 1 decisions brief

**Status:** Historical Phase 1 drafting record; non-authoritative.

**Synchronized:** 2026-08-05

This brief preserves the inputs and alternatives used to draft Phase 1. The
approved [`product contract`](../product/product-goals.md) and approved
architecture contracts—[`system overview`](../architecture/system-overview.md),
[`data/time/event`](../architecture/data-time-and-event-contract.md), and
[`Scanner State Engine lifecycle`](../architecture/scanner-state-engine-lifecycle.md)—
control wherever this history is less specific or stale. The brief cannot
override them.

## Current resolution index

The former open decisions in section 16 now classify as follows. No item in the
historical list remains a blocker to Phase 1 owner approval.

| # | Current classification | Controlling contract or named destination |
| ---: | --- | --- |
| 1 | Resolved in Phase 1 | Universe and exact adjusted-prior-close policy: `PG-UNIVERSE-01`, `PG-REFERENCE-01`, and `PG-RANK-01` in the [product contract](../product/product-goals.md). Provider retrieval/cache mechanics remain in the universe/session-binding specification. |
| 2 | Resolved in Phase 1 | Qualification formula and correction-aware session latch: `PG-RANK-03`. |
| 3 | Resolved in Phase 1 at product level | Version 1 fields, formulas, ranking effects, and availability independence: `PG-FEATURE-*` and `PG-AVAIL-*`. Reference-block eligibility, condition fixtures, quote validation, and retained algorithms remain in the aggregate-feature and T/Q specifications. |
| 4 | Resolved in Phase 1 | `degraded_bootstrap` is a permanent, explicitly partial product projection under `PG-RANK-05`; it is not a diagnostic-only substitute for the qualified table. |
| 5 | Resolved in Phase 1 | Process live, backend ready, ranking current, and field/T/Q currentness are distinct under `PG-OBS-03` and `LIFE-PUBLISH-*`. Exact timing and HTTP mappings remain in the readiness/operations specification. |
| 6 | Intentionally deferred | Checkpoint cadence, encoding, retained representation, and restart objective belong to the checkpoint/restart specification; version 1 checkpoint support and coherent `T0` semantics are already mandatory. |
| 7 | Intentionally deferred | Polling, SSE, or WebSocket snapshot delivery belongs to the snapshot API/UI specification. |
| 8 | Intentionally deferred | Predecessor UI reuse versus replacement follows approval of the snapshot API/UI specification. |
| 9 | Intentionally deferred | T/Q pressure thresholds, hysteresis, and restoration timing belong to the T/Q coverage/pressure specification; aggregate-first priority is settled. |
| 10 | Intentionally deferred | Implementation language and build/deployment tooling are lower-level implementation-plan choices unless a consequential tradeoff requires an ADR. |
| 11 | Intentionally deferred | Replay artifact storage, retention, privacy, and provider-license policy belong to the aggregate replay specification and evidence registry before data is downloaded or committed. |

## 1. Purpose of the new repository

Build a production-grade live equities momentum scanner in a new repository
with a deliberately reviewable architecture and commit history.

The predecessor is retained as a reference for:

- approved or useful product semantics;
- Massive REST and WebSocket behavior;
- observed market-data and operational edge cases;
- provider fixtures;
- formulas and bounded data structures worth preserving; and
- differential regression evidence.

Do not copy its accumulated owner/recovery/planner architecture by default. A
component should be reused only when its behavior is needed, understandable,
bounded, and pinned by focused tests. Over-engineered components should be
reduced to the required behavior rather than transplanted intact.

The new repository and its commit sequence should let an engineering reviewer
understand how product goals became contracts, architectural decisions, tests,
and implementation.

## 2. Product mission

The scanner serves a discretionary U.S. equities momentum trader. It identifies
sufficiently active eligible stocks with the highest return from the adjusted
previous regular-session close and displays current-session context useful for
short-horizon decisions.

It is a market-data measurement and discovery product. It does not produce
orders, entries, exits, forecasts, recommendations, or claims of executable
expectancy.

The intended operating session is 04:00 through 20:00 America/New_York. The
eligible security policy begins with active U.S.-listed common stocks and ADR
common stocks. Exact universe policy remains subject to Phase 1 confirmation.

## 3. Version 1 required capabilities

Version 1 is not a throwaway prototype. It must include:

1. current eligible universe and adjusted prior closes;
2. persistent live one-second aggregate coverage;
3. canonical correction-aware per-symbol session state;
4. explicit fresh bootstrap, checkpoint catch-up, ordinary live operation, and
   same-process recovery;
5. qualification, exact ranking, and top-20 publication;
6. the approved aggregate-derived display features;
7. live trade and quote streaming for all displayed top-20 symbols;
8. Tape Rate and Spread for selected symbols;
9. a pressure policy that sacrifices T/Q before aggregate correctness;
10. coherent checkpoints and fast restart catch-up;
11. deterministic full-session aggregate download and offline replay;
12. exact symbol and work accounting;
13. a versioned read-only backend API; and
14. an independently deployable UI that can change while the scanner remains
    live.

## 4. Explicitly deferred or excluded from version 1

- Full-session or two-pass T/Q replay is deferred. Preserve it in documentation
  as a possible extension if later T/Q features justify its data and complexity.
- A generic feature-plugin system is excluded. Features should be modular and
  easy to extend without introducing a framework before one is needed.
- Full-universe T/Q-derived ranking is excluded. Version 1 T/Q begins after a
  symbol enters the aggregate-qualified top 20.
- A database, generalized event bus, raw-event journal, microservice split,
  worker hierarchy, and per-symbol goroutines are excluded unless later
  evidence establishes a concrete need.
- Trading or order execution is excluded.

## 5. High-level data flow

```text
session schedule + product configuration
  -> eligible universe + adjusted prior closes
  -> session binding

Massive live WebSocket A/T/Q    Massive REST aggregates    replay aggregates
              \                       |                       /
               -> provider normalization and validation <-
                                  |
                                  v
                         ScannerStateEngine
                                  |
                   canonical per-symbol session state
                                  |
                 features + qualification + exact ranking
                                  |
                         immutable snapshot
                          /       |       \
                checkpoint     API/UI     top-20 T/Q intent
```

## 6. Proposed high-level components

### 6.1 Product configuration and deterministic session clock

Own the trading date, timezone, session bounds, evaluation delay, aggregate
correction horizon, ranking limit, T/Q desired count, and clock behavior shared
by live and replay operation.

No component may independently invent wall-clock or watermark semantics.

### 6.2 Universe and prior-close binding

Load and validate eligible symbols, adjusted previous regular-session closes,
reference dates, policy versions, and deterministic identities. Produce one
immutable session binding used by state, recovery, checkpoints, and publication.

### 6.3 Market-data sources and offline replay tools

Support live WebSocket events, historical REST aggregates, and cached replay
aggregates behind explicit source boundaries. Version 1 offline tooling
downloads second aggregates, normalizes them, and replays them under a simulated
clock.

### 6.4 Provider normalization and validation

Translate provider-specific messages into normalized aggregate, trade, quote,
control, connection, and per-symbol hydration-result events. Downstream product
state must not parse Massive payloads.

### 6.5 Scanner State Engine

The `ScannerStateEngine` is the only ordered owner of changing scanner state. It
validates event generation and coverage, applies accepted events to canonical
state, advances committed time, triggers evaluation, and publishes immutable
snapshots.

It orchestrates component interactions but does not contain provider JSON,
REST pagination, feature formulas, T/Q measurement algorithms, HTTP rendering,
or UI behavior.

### 6.6 Canonical per-symbol market and feature state

Store the minimum bounded state needed for current features and straightforward
extension:

```text
SymbolState
  core identity, prior close, mark, coverage, and integrity
  aggregate state and correction tail
  trade state and coverage
  quote state and coverage
  qualification state
  feature-specific bounded derived state
```

New aggregate-, trade-, quote-, or mixed-input display features should be
addable without changing lifecycle ownership. A feature must define its inputs,
retained state, time window, update semantics, coverage, availability,
boundedness, ranking effect, checkpoint behavior, and tests.

Do not build a generic plugin framework in version 1.

### 6.7 Bounded hydration and recovery

Handle fresh bootstrap, checkpoint catch-up, and known same-process coverage
gaps using explicit intervals, generations, bounded pagination/concurrency, and
one terminal result per requested symbol:

```text
completed_value | completed_empty | failed | canceled | fenced
```

A successful fully reconciled empty result establishes `no_print_through(T)`.
It completes work and makes the symbol currently unrankable. It does not create
a price. A later accepted aggregate changes the symbol to a real mark and
triggers ordinary evaluation.

Hydration completion must transition to ordinary live evaluation even when the
candidate population is sparse. There is no eternal debt for a proven no-print
symbol and no terminal deferred state that freezes committed time.

### 6.8 Feature calculation and qualification

Version 1 aggregate-derived fields are expected to include Day return, From 4AM
return, HOD drawdown, session/30-minute/60-minute range position, and Activity.
Version 1 T/Q-derived fields include Tape Rate and Spread.

Each field has independent current, warming, stale, unavailable, or invalid
semantics. Historical-field incompleteness does not globally block a trustworthy
ranking. Feature formulas remain subject to explicit Phase 1 product review.

The aggregate minimum-tape qualification is distinct from display features and
filters table membership before ranking.

### 6.9 Ranking and immutable publication

At one committed watermark `T`, classify the complete prior-close-eligible
population, update qualification, filter nonpassers, calculate Day return, sort
by Day return descending and exact symbol ascending, and publish at most 20
rows.

The same evaluation path is used after fresh bootstrap, checkpoint catch-up,
ordinary live ingress, and same-process recovery. Published snapshots are
immutable.

### 6.10 Default top-20 T/Q coverage and aggregate protection

Normal operation requests trades and quotes for every displayed top-20 symbol
on the existing Stocks WebSocket. T/Q is a first-class user feature, but a
separate failure domain from aggregate ranking.

Pressure policy has three conceptual levels:

```text
normal
  process T/Q for all displayed top 20

taq_degraded
  continue reading the socket and preserve aggregate/control events,
  but discard T/Q before expensive feature processing

aggregate_only
  unsubscribe T/Q to reduce incoming traffic and preserve A.*
```

After any T/Q coverage gap, affected measurements warm again from fresh events.
They must not silently bridge discarded or unsubscribed intervals. Exact
thresholds require capacity evidence and are not settled in this brief.

### 6.11 Coherent version 1 checkpoints

Persist the minimum derived aggregate/session state needed for fast correct
restart. A checkpoint describes one committed timestamp: mark, qualification,
session extrema, rolling structures, Activity, and accounting may not represent
different effective times.

Startup validates binding and format, installs the last compatible complete
checkpoint, REST-catches up the missing interval, reconciles live ingress, and
returns to ordinary evaluation. Corrupt or incompatible checkpoints fail safely
to bounded fresh hydration.

Write replacement must be atomic. Export and write failures need bounded reason
codes. Ephemeral T/Q coverage and instantaneous measurements normally warm from
fresh connection data instead of surviving restart.

### 6.12 Versioned snapshot API

Expose immutable ranking, features, coverage, readiness, accounting, and bounded
diagnostics through a read-only versioned contract. Internal Go or implementation
structs do not automatically define the public schema.

### 6.13 Independently deployable UI

The UI is a separate application and lifecycle. It consumes the versioned API,
owns presentation only, and can be rebuilt or deployed without restarting the
live scanner backend.

## 7. Scanner State Engine lifecycle direction

The intended high-level lifecycle is:

```text
initializing
  -> awaiting aggregate acknowledgement
  -> loading checkpoint or fresh hydration
  -> live
  -> recovering after a real coverage gap
  -> live
  -> ended
```

A material global binding, canonical-state, or ingress-integrity failure may
enter a separately defined suppressed state. Symbol-local failures remain
symbol-local.

The engine processes state-changing events through one ordered path. Network
reading, REST fetching, and file writing may be concurrent, but they return
events or results to the engine rather than mutating canonical state directly.

## 8. Data and time direction

Define normalized event contracts for:

- one-second aggregates;
- trades;
- quotes;
- connection and control acknowledgements;
- hydration progress and terminal symbol outcomes; and
- replay clock/timer events.

The architecture must distinguish:

- provider event time;
- local receipt time;
- deterministic replay time;
- committed evaluation watermark;
- connection epoch;
- recovery generation; and
- requested historical coverage interval.

All feature windows are causal and half-open at committed watermark `T` unless a
reviewed product specification says otherwise.

REST and live aggregates may have different wire fields, but normalize to the
same canonical aggregate identity before state mutation. REST replay proves
market-time calculations; without recorded receipt chronology it does not prove
actual network latency or live correction arrival order.

## 9. Rankability, no-print, and accounting direction

The primary population must reconcile at every publication. The target identity
is conceptually:

```text
universe_total
  = valid_prior_close + invalid_or_missing_prior_close

valid_prior_close
  = mark_value_rankable
  + mark_value_below_price
  + no_print_through_T
  + mark_invalid
  + mark_unknown_due_failure_or_fence
```

REST work separately reconciles:

```text
planned
  = completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Qualification, activity, historical-field status, and range failures are
overlapping dimensions and must not be added to the primary mark identity.

## 10. Feature-extension direction

A future feature should require a focused contract answering:

1. What user question does it answer?
2. Which normalized event types does it consume?
3. What event-time window does it use?
4. What bounded per-symbol or global state does it retain?
5. How do corrections, late data, and coverage gaps affect it?
6. When is it warming, current, stale, unavailable, or invalid?
7. Does it affect rank/qualification or only display?
8. What state, if any, belongs in a checkpoint?
9. What is the minimum distinct proof required?

T/Q-derived features for already selected top-20 symbols should fit this model
without refactoring the scanner. Using T/Q to rank the full universe would
require broader data coverage and separate product/capacity approval.

## 11. Replay direction

Version 1 requires aggregate-only offline replay. A downloader/compiler obtains
historical one-second aggregates, validates provider-specific differences, and
stores a normalized bounded replay artifact. A replay source advances a
simulated clock and feeds the same `ScannerStateEngine` event boundary used by
live normalized events.

Playback speed must not change output. Expected controls include run, pause,
step by event or second, accelerated playback, and run to completion; the exact
operator interface remains open.

Do not build full T/Q replay in version 1. Preserve this future design idea:

1. aggregate replay records selected top-20 intervals;
2. T/Q is acquired only for those symbols and intervals; and
3. a second deterministic run independently reproduces selection and emits
   cached T/Q only during simulated active coverage.

## 12. API and UI separation

The backend owns market-data connections, canonical state, features, ranking,
checkpoints, and readiness. It operates correctly with no browser connected.

The UI consumes a stable versioned snapshot API and owns formatting,
presentation, and interaction only. Compatible UI releases must be deployable
without restarting or rebuilding the scanner backend.

Whether the backend scanner and API are one process or two is an implementation
decision unless operational evidence requires separation. The UI remains a
separate build/deployment in either case.

## 13. Test and edge-case governance

Test design is reviewed with behavior, not added without bound after
implementation.

Every nontrivial edge case needs evidence from at least one of:

- provider documentation;
- a captured provider fixture;
- an observed live failure or diagnostic;
- a mathematical/product invariant;
- a predecessor regression; or
- explicit owner approval.

Every requirement receives one primary proof. Higher-level duplication is
allowed only when it proves a distinct integration boundary.

The planned proof levels are:

1. pure formula/state tests;
2. provider normalization fixtures;
3. focused lifecycle scenarios;
4. deterministic full-session aggregate replay;
5. checkpoint/restart equivalence;
6. old/new differential offline replay where the old output is trustworthy;
7. authorized non-authoritative live shadow observation; and
8. separately approved operational cutover.

Do not equate code coverage or test count with market-data correctness. Do not
call scanner agreement evidence of predictive or executable trading edge.

## 14. Specification-building process

### Phase 1: product and architecture

Draft and owner-review together:

- product goals and non-goals;
- system overview and component ownership;
- data, time, event, and glossary contracts;
- Scanner State Engine lifecycle; and
- specification map.

No production implementation begins during this phase.

### Phase 2: cross-component contracts

Settle normalized event interfaces, symbol-state boundaries, feature extension,
checkpoint boundary, snapshot API, readiness, and accounting identities before
parallel implementation.

### Phase 3: focused component specifications and proof

For each component, define behavior, inputs/outputs, invariants, terminal
outcomes, failure policy, boundedness, and the minimum approved tests. Avoid
dictating private helpers or file-by-file implementation unless correctness
depends on them.

### Phase 4: implementation assignments

Give each agent only the approved architecture documents, its focused component
specification, applicable ADRs, interfaces, fixtures, allowed scope, and required
verification. Central state/event/API contracts retain a single integration
owner. Parallelize only behind approved stable boundaries.

## 15. Reuse policy

Reuse behavior before code. Candidate reusable areas include:

- Massive payload fixtures and protocol findings;
- aggregate identity and canonical validation rules;
- correction, duplicate, late, and out-of-order semantics;
- qualification and ranking formulas;
- bounded rolling data structures after independent review;
- schedule, universe, and prior-close behavior;
- T/Q condition, acknowledgement, and measurement rules;
- checkpoint validation ideas; and
- UI presentation and styling.

Do not make the predecessor repository a runtime dependency. Port a component
only after its required contract is approved and its accidental architecture is
separated from the needed behavior.

## 16. Historical open-owner decision list

This is the original drafting list. The current classification is the resolution
index at the top of this document; these items no longer carry open status merely
because they are preserved below.

1. Confirm the exact universe eligibility and adjusted-prior-close policy.
2. Confirm the existing minimum aggregate tape qualification formula or revise
   it before it becomes the new product contract.
3. Confirm every version 1 displayed feature formula and availability behavior.
4. Decide whether named degraded raw Day-return ranking is a permanent product
   feature during incomplete hydration or only an operational diagnostic.
5. Decide the exact definition of backend `ready` versus ranking `current`.
6. Decide the checkpoint cadence, retained state boundary, and acceptable
   restart target, while keeping checkpoints mandatory in version 1.
7. Decide whether the UI consumes polling, server-sent events, WebSocket
   snapshots, or an implementation-selected compatible mechanism.
8. Decide which UI code is ported versus rewritten after the API is approved.
9. Decide T/Q pressure thresholds after reviewing existing live capacity
   evidence; the priority policy itself is settled.
10. Decide the implementation language and top-level build/deployment tooling.
11. Decide replay artifact retention, privacy, and provider-license constraints
    before downloading or committing market data.

## 17. Phase 1 completion criteria

Phase 1 is complete only when the owner can review one coherent document set and
answer:

- what the scanner promises to the trader;
- where every input comes from;
- what every high-level component owns;
- how live, bootstrap, restart, recovery, and replay interconnect;
- which state and clock are authoritative;
- how features can be added from aggregates, trades, and quotes;
- how aggregate ranking survives T/Q failure;
- what checkpoints preserve;
- how the backend and UI remain independent;
- which edge cases are evidenced; and
- which deeper specifications remain before implementation.
