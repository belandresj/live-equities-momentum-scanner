# Selected-row T/Q state

**Status:** Owner-approved focused replacement specification, 2026-08-23. The
[delivery program](delivery-program.md) is the sole mutable status ledger.

**Parent:** [Live backend replacement architecture](../live-backend-replacement.md).

## 1. Outcome and authority

This contract gives the sole engine owner the smallest bounded selected-row
state that can produce Tape 5s and Spread honestly. It owns desired T/Q
membership, local request generations, per-channel confirmation/coverage,
deduplication, Tape/Spread state, pressure mode, and T/Q projection. It never
owns aggregate qualification, ranking, watermark, readiness, provider socket
I/O, or API/UI interpretation.

No `LBR-ARCH-*` requirement is allocated here. The integration contract owns
cross-cutting `LBR-ARCH-01`/`02`/`11` conformance, while this focused contract
supplies their engine-owned T/Q, bounded-state, and T/Q-local containment
obligations. Allocated retained product semantics are
`PG-FEATURE-05`, `PG-FEATURE-06`, `PG-AVAIL-03`, `PG-TAQ-01`,
`PG-TAQ-02`, and `PG-TAQ-03` from
[`product-goals.md`](../product/product-goals.md).

The compatible historical semantics routed here are `DTE-CLOCK-02`,
`DTE-WINDOW-03`, `DTE-TRADE-01`, `DTE-TRADE-02`, `DTE-QUOTE-01`,
`DTE-QUOTE-02`, and `DTE-TQ-01` through `DTE-TQ-03` from the
[data/time/event contract](../architecture/data-time-and-event-contract.md),
plus `LIFE-TQ-01` through `LIFE-TQ-03` from the
[engine lifecycle](../architecture/scanner-state-engine-lifecycle.md).
The provider-status correction in
[`tq-data-confirmed-subscription-correction.md`](../tq-data-confirmed-subscription-correction.md)
controls over earlier status-count assumptions. No replay T/Q meaning is
routed.

## 2. Boundary and dependencies

This capability depends on the accepted aggregate selection/publication seam
from [`evaluation-and-publication.md`](evaluation-and-publication.md). The
desired set is exactly the ordered rows of a current `qualified_current`
snapshot, at most 20. No other output creates additions; losing that state
closes desired coverage immediately.

It defines typed command, normalized T/Q, drop, pressure-sample, and
write-result seams that [`live-ingress.md`](live-ingress.md) implements. The
current adapter may temporarily implement those seams until Capability D.
This direction keeps dependencies acyclic: T/Q specifies engine-owned facts;
ingress later supplies or executes them.

| Input | Required evidence |
| --- | --- |
| desired ranking revision | binding, epoch, publication/revision, ordered symbols, and qualified-current claim |
| normalized trade/quote | binding/date, epoch, frame/array position, symbol, event and receipt times, structural/classification evidence, and bounded canonical identity/value fields |
| command write result | private command token, action, symbol generations, epoch, success/failure, receipt time, and boundary `B` for successful subscribe writes |
| pressure sample/drop | private sample identity, fixed queue/lag/accounting scalars, or exact affected channel/symbol and causal position |

Outputs are one private command at a time, early-shed policy for ingress, and
one immutable rank-ordered `SelectedTQView` joined by publication. T/Q trust
closures notify publication immediately; ordinary measurement changes may
coalesce to the next one-second snapshot.

Reuse evidence is limited to the current repository's
[`top-20 T/Q contract`](../specifications/top-20-tq-coverage-and-features.md),
the data-confirmed correction cited above, and their directly named condition,
membership, pressure, and API fixtures. Tape/Spread meanings and dangerous
counterexamples are retained; broad retention, status acknowledgement,
historical ledgers, replay state, and private implementation shapes are
rejected. No predecessor checkout or provider request is part of this work.

## 3. Membership and causal coverage

Each desired symbol has one local generation in the current connection epoch.
Trade and quote channel states are independent:

`not_requested -> requested_unconfirmed -> confirmed -> closed`.

A successful paired subscribe write is request evidence only. Immediately
after the write returns, ingress captures `B`, the greatest complete provider
frame sequence already read. Only a structurally valid matching event with
frame sequence strictly greater than `B`, current epoch/generation, and still-
desired symbol confirms that channel. The entire frame at `B` is excluded.
Feature-ineligible trades and unusable but structurally identifiable quotes may
confirm delivery without contributing a measurement.

Generic post-authentication `success` status is informational. Its presence,
absence, count, or timing never completes a command or opens coverage. A failed
write opens nothing. Silence leaves a requested channel unconfirmed and causes
no retry. A post-authentication provider `error` closes T/Q for the epoch and
blocks further additions without changing aggregates; authentication failure
remains a connection terminal owned by ingress.

At a fresh accepted epoch, the engine issues one sorted paired subscribe batch
for the complete desired set. At each later combined cadence it computes one
set difference: close removals locally first, issue at most one sorted paired
unsubscribe batch, then after its write result issue at most one sorted paired
subscribe batch for additions. There is never more than one write in flight.
Rank churn is batched rather than serialized per symbol. Pressure restoration
is deliberately gradual: cleanup precedes addition and at most one highest-
ranked symbol is added per successful cadence until the desired set is restored.

Rank removal, pressure shedding, control error, connection loss/epoch change,
session end, or a state bound immediately closes affected coverage, clears
measurements, and invalidates the generation. Unsubscribe success means only
requested removal; provider absence remains unknown unless locally no request
exists. A returning symbol receives a new generation and must reconfirm both
channels.

## 4. Event-time, duplicate, and feature state

The committed aggregate `T` is an as-of input only; T/Q never advances it. For
a current-epoch desired symbol with committed `T`:

- an event is too late only when `event_time < T - 30s`; equality is accepted;
- a too-late event is counted and ignored without opening/reopening feature
  history beyond its existing confirmed coverage;
- accepted trade duplicate evidence is retained until engine receipt time is
  strictly greater than `first_receipt + 30s`; equality remains retained; and
- cleanup uses monotonic engine time and the recorded UTC receipt time, so
  caller clock or wall-clock regression cannot shorten the boundary.

Before committed `T` exists, a structurally valid post-boundary event may
confirm its channel but cannot make a field current; the ordinary bounds still
apply. No event older than the session or from a stale epoch/generation enters
state.

Trade duplicate identity and canonical fingerprint follow `DTE-TRADE-02`.
An exact repeat contributes once and is counted. An unequal repeat for the same
complete identity makes Tape invalid for that channel generation; it is not
guessed as a correction. Incomplete lifecycle evidence is disclosed and only
approved qualifying original prints contribute to Tape. The state retains
only contributions needed by the approved five-second result and the separate
30-second receipt-bounded duplicate ledger. There is no one-second Tape-burst
state or output.

Tape coverage warms from the first confirmed trade and becomes current only
after the product's continuous five-second coverage. Covered quiet time is
numeric zero. Spread evaluates the first confirmed quote immediately; it keeps
only the latest observed quote and latest valid quote, applies the approved
quality rules, and becomes stale with age without fabricating a gap. A real
coverage closure clears both records. Locked spread is genuine numeric zero;
one-sided/crossed/invalid input remains explicitly noncurrent as defined by the
product contract.

## 5. Retention and pressure bounds

At most 20 desired symbols plus at most 20 distinct cleanup liabilities exist.
Per desired symbol, contribution state retains at most 10,000 qualifying or
lifecycle records across the five-second calculation; the global contribution
limit is 100,000. Duplicate evidence retains at most 25,000 exact identity/
fingerprint entries per symbol and 400,000 globally, all within the 30-second
receipt horizon. The combined retained T/Q state has a 64 MiB hard byte charge.
Quotes remain O(1): latest observed and latest valid, at most 40 records for the
desired set. A known-absent, undesired, empty member is deleted immediately.

A per-symbol count/byte hit closes that symbol's T/Q fields and requests
cleanup. A global count/byte/accounting hit enters `aggregate_only` and removes
T/Q membership. These bounds are containment limits, not throughput claims;
integrated characterization may revise the exact private division through the
delivery program before Capability C begins, but the accepted contract must
always state finite count and byte limits and preserve the 30-second boundary.

Pressure consumes only direct fixed-cardinality ingress evidence: waiting
batch count/bytes and capacities, oldest waiting-batch age, aggregate watermark
lag while T/Q work exists, new capacity loss, queue/transport/T/Q accounting
loss, and retention-bound state. Active-frame work age, heap, goroutines,
generic delivery latency, delivery-family attribution, checkpoint state, and
missing/quiet samples are diagnostics only.

One engine-issued sample may be outstanding. Results are accepted in order
within two seconds. Entry thresholds retain the approved direct-pressure
policy: two consecutive one-second samples at at least 10% waiting slots or
bytes, or oldest waiting age at least one second, enter `taq_degraded`; three
consecutive samples at at least 25% or two seconds enter `aggregate_only`.
Watermark lag greater than two seconds for two samples while T/Q work exists,
or a new capacity/accounting/global-bound failure, enters `aggregate_only`.

`taq_degraded` closes coverage and makes ingress skip remaining expensive T/Q
normalization while continuing mixed-frame aggregate/control classification.
`aggregate_only` also removes provider T/Q membership. Recovery requires five
consecutive accepted one-second samples below 1% slots and bytes, oldest age
strictly below 750 ms, aggregate lag at most one second, coherent accounting,
and no new loss/bound. Exactly 750 ms is unhealthy. Every restoration starts a
fresh generation. Pressure never changes aggregate ranking/readiness.

Independently, a decoded frame has a 500 ms monotonic classification budget.
After it expires, ingress still classifies the rest of the frame in order but
sheds T/Q elements before expensive normalization and reports each affected
coverage closure. Frame age alone cannot change the global pressure mode.

## 6. Accounting and trust boundaries

Fixed counters reconcile T/Q facts as `consumed = applied + duplicate +
rejected + fenced + pressure_shed + integrity`. Command accounting distinguishes
issued, pending write, written, failed, and fenced; no status acknowledgement
category exists. Desired, data-confirmed present, locally absent, and unknown
membership are separately observable. Retained contribution, duplicate, quote,
and byte totals equal the sum of member charges and stay within their bounds.

The dangerous false successes are a pre-`B` event confirming coverage, silence
becoming zero, an old generation warming a return, a duplicate counted twice,
a bound leaving a current value, a T/Q drop losing mixed aggregate/control
work, or T/Q changing rank/readiness. Private command/sample types prevent
caller-selected generations, boundaries, and pressure streaks. Runtime
validation fences stale tokens/epochs/positions, malformed times/values,
unexpected symbols, late evidence, and status-based confirmation.

Malformed T/Q is local only when both family and symbol are trustworthy. An
element/frame ambiguity that could conceal aggregate/control work belongs to
ingress recovery. T/Q integrity closes the affected channel or all T/Q; it
does not suppress aggregate state.

## 7. Primary proofs and slices

| Slice | Primary proof | Claim, dangerous counterexample, observable distinction, limitation |
| --- | --- | --- | --- |
| `LBR-C1` | `P-LBR-C1-TQ-STATE` | A deterministic selected-symbol trace covers the strict `B` boundary, independent trade/quote confirmation, Tape warm/current/covered-zero, Spread locked/one-sided/crossed/stale, exact duplicate, unequal repeat, lifecycle disclosure, out-of-order events at `T-30s` and one tick older, duplicate expiry at receipt+30s and one tick later, channel gap, rank removal, per-symbol/global count and byte hits. It observes exact fields, coverage, membership, counters, and aggregate publication equivalence. It detects a broad 16-minute fingerprint store or status-created coverage. It does not prove socket writes or pressure queue behavior. |
| `LBR-C2` | `P-LBR-C2-TQ-MEMBERSHIP` | Compose fresh batched membership, rapid rank churn, unsubscribe-before-subscribe, write failure, generic/late statuses, post-write data confirmation, quiet unconfirmed channels, T/Q error, two pressure entry levels, early mixed-frame shedding, removal to zero, exact five-sample 749/750-ms recovery, gradual restoration, reconnect reset, and immediate trust publication. It asserts one in-flight write, bounded members, exact accounting, no T/Q-caused aggregate/watermark/readiness change, and no aggregate/control loss. It does not prove provider acceptance of a request or final ingress queue implementation. |

`LBR-C1` replaces broad trade/string retention with the bounded contribution,
30-second duplicate, and O(1) quote state. Acceptance makes old one-second Tape
burst state, 16-minute T/Q fingerprint retention, and separate mutable Tape/
Spread representations removable.

`LBR-C2` replaces per-symbol status-correlated command sequencing and the old
membership/pressure projection with cadence-batched request generations,
data-confirmed coverage, direct-pressure state, and one immutable T/Q view.
Acceptance makes expected-status counts/deadlines, status quarantine, old
command-ack coverage, unbounded churn liabilities, and old T/Q publication
joins removable. Capability D later removes their adapter-side envelopes and
raw-queue machinery.

## 8. Verification, review, and discretion

Each slice runs its primary proof, affected short/race tests, focused vet,
`git diff --check`, and ordinary repository verification at the gate. The
Capability C review focuses on exact horizon ties, bounded retention charges,
data-confirmed coverage, command/pressure ownership, provider-status
assumptions, and aggregate independence. A narrow review is triggered for any
remaining command-boundary, pressure-transition, or mixed-frame false-success
linearization.

The implementer may choose compact identity encodings with exact collision
resolution, rings/deques, fixed arrays, eviction indexes, private enums, and
batch command representation. The implementer may not invent provider
acknowledgements/retries, extend either 30-second horizon, change product
formulas/quality rules, make T/Q a ranking/readiness input, or add a T/Q worker
hierarchy.

Aggregate selection/formulas, canonical merge/hydration, socket decoding and
retry execution, API/UI redesign, provider access, replay/checkpoint work, and
public deployment are non-scope. Slice evidence and status are recorded only
in the delivery program.
