# Top-20 T/Q coverage and features

**Status:** Component 9 reaccepted after the 2026-08-13 owner-directed waiting-
pressure and frame-local protection correction; deterministic, ordinary,
affected race, vet, and diff verification pass

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
decisions remain revisable through the program correction loop

**Controlling Phase 1 requirements:** `PG-FEATURE-03`, `PG-FEATURE-04`,
`PG-FEATURE-05`, `PG-AVAIL-01`, `PG-AVAIL-03`, `PG-TAQ-01`,
`PG-TAQ-02`, `PG-TAQ-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-03`, `DTE-WINDOW-01`, `DTE-WINDOW-03`, `DTE-TRADE-01`,
`DTE-TRADE-02`, `DTE-QUOTE-01`, `DTE-QUOTE-02`, `DTE-CONTROL-01`,
`DTE-TQ-01`, `DTE-TQ-02`, `DTE-TQ-03`, `DTE-REJECT-01`,
`LIFE-TQ-01`, `LIFE-TQ-02`, `LIFE-TQ-03`, `LIFE-PUBLISH-02`, and
`LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Components 1-8, including C5
normalized T/Q/control facts and C8 aggregate/runtime pressure measurements

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C9 boundary plan;
Sections 8-19 are completed here after just-in-time V2 reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C9 boundary, current detailed contract, proofs, slices, and sole delivery ledger | Sections 1-19 accepted as the current executable plan | Every C9 task | Components 1-8 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `accepted` | Direct owner V1 program revision; C8 finally accepted; recorded V2 scope inspected | Complete |
| Completed contract | `accepted_current_plan` | Focused read-only review clean after corrections for ambiguous membership, accounting failure domains, bounds, out-of-order quote time, pressure authority/expiry, and the distinct acknowledged Q time/causal boundaries | Complete; remains revisable through the correction loop |
| `C9-S1` membership/coverage/features | `reaccepted_after_owner_correction` | Latest valid two-sided non-crossed quote plus age, quiet numeric stale Spread, zero Tape Rate, O(1) quote state, invalid one-sided/crossed proof, and exact 20-row desired boundary pass | Complete locally |
| `C9-S2` pressure/shedding/restoration | `reaccepted_after_owner_correction` | Waiting frame/byte/oldest-age pressure, diagnostic-only active-frame age, separate loss/accounting/bound containment, guarded aggregate lag, exact five-sample recovery, 500-ms frame-local protection, serialized restoration, and aggregate-independence proofs pass | Complete locally |
| `C9-S2-H/L/P` prior host/probe gates | `superseded` | Heap, goroutine, generic delivery latency, attribution, empty-window, and top-one gates are invalidated by the latest owner direction and are retained below only as historical correction evidence | No executable code, configuration, or current proof depends on them |
| `C9-S2-D` direct-pressure/latest-quote correction | `superseded_pressure_definition` | Owner-directed product correction and checkpoint-off evidence: acknowledged rank-1 T/Q produced Tape Rate, then false pressure shed it despite negligible queue depth | Preserved as evidence; D8/W owns current pressure semantics |
| `C9-S2-W` waiting-pressure/frame-local correction | `accepted` | Owner evidence proved normal top-20 delivery with 43/32,768 queue high-water and the active frame as the sole false pressure cause; exact focused/ordinary/race/vet/diff proofs pass | Complete locally; live/provider capacity remains unclaimed |
| Live T/Q protocol resilience correction | `accepted_on_clean_main` | [`live-tq-resilience-correction.md`](../live-tq-resilience-correction.md) adds epoch-local T/Q quarantine, fresh-epoch reset, and typed command/status diagnostics while proving canonical aggregate/evaluation/watermark equivalence. D8/W retains authority over the 32,768-frame, one-second, 10%/25% pressure policy. | Complete deterministically; owner-run live confirmation pending |
| Final correction review decision | `not_triggered` | The correction changes bounded scalar policy, projection semantics, and diagnostics but adds no mutable owner, concurrency linearization, persistence/atomicity, or external-success authority; distinguishing construction/proofs make the boundary straightforward under the V1 risk cadence | Complete; reopen only through the V1 correction loop |

## 1-4. Outcome, scope, ownership, and settled boundary

C9 normally enriches every current qualified displayed row with continuous
trade and quote coverage, Tape Rate, and NBBO Spread. When load threatens
aggregate processing, it sheds all T/Q work if necessary and later restores
coverage in current aggregate rank order with fresh warm-up.

In scope:

- derive desired membership from current `qualified_current` rows only;
- correlate paired T/Q commands and acknowledgements with current epoch and
  begin per-channel causal coverage only after acknowledged boundaries;
- implement Tape Rate's five-second and one-second qualifying-original trade
  rates with evidenced conditions and explicit lifecycle limitations;
- implement latest valid two-sided non-crossed NBBO spread in cents and basis
  points with quote age, locked-zero, retained numeric stale behavior during
  quiet continuous coverage, and explicit crossed/one-sided states;
- clear measurements across gaps, maintain continuous warm-up, report coverage
  and field status, and add no Tape Rate attention threshold;
- reject expensive T/Q before aggregate/control, permit complete T/Q
  unsubscription, and restore gradually in current rank order.

Not in scope: aggregate qualification/ranking/readiness, full-universe T/Q,
full-session T/Q replay, public API/UI, an unevidenced complete trade-lifecycle
claim, a Tape Rate trading label, or provider credentials/live observation.

The engine remains the sole owner of desired membership, acknowledged causal
coverage, feature state, pressure state, and field availability. The C5 adapter
continues to normalize/read/control the shared socket and returns facts. No T/Q
fact can change aggregate `T`, qualification, rank, or backend readiness.

The component introduces no new ranking key, threshold-based trading signal,
watermark, evaluator, time-window meaning, or competing mutable owner.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must resolve the exact trade-condition fixture,
lifecycle completeness disclosure, NBBO quote-validity and coverage minimum,
time-weighting boundary, pressure measurements/thresholds/hysteresis, command
restoration pace, bounded retention, and compact proofs.

After C8 final acceptance, reconnaissance may inspect only V2 T/Q condition
fixtures and classification, Tape Rate/Spread calculators, acknowledgement and
coverage tests, pressure/shedding/restoration behavior, and directly used
focused fake-provider fixtures. Reject full-universe selection, T/Q-to-ranking
coupling, browser calculations, old health/readiness owners, broad replay,
credentials, and live requests.

Likely primary proofs:

1. one deterministic selected-membership/acknowledgement/gap/feature trace that
   proves Tape Rate, Spread, continuous coverage, independent status, and
   current-rank membership; and
2. one controlled pressure trace showing classification preserves aggregate/
   control facts, T/Q can shed to zero, aggregate state/rank/readiness remain
   unchanged, and restoration follows current rank order with fresh warm-up.

**Provisional slices:** At most two: `C9-S1` membership, coverage, and features;
`C9-S2` pressure, complete shedding, and restoration. These can be revised
without owner approval when evidence remains inside the fixed boundary.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.

## 8. Reconnaissance result and executable decisions

Only the recorded Version 2 T/Q scope was inspected. It remains evidence, not
authority:

| Source and SHA-256 | Decision | Preserved evidence and coupling removed | Allocated proof |
| --- | --- | --- | --- |
| `internal/massive/trade_conditions.go` `7d25ea...034` and `trade_conditions_fixture.json` `290afe...050` | `adapt` | Preserve the embedded, strictly decoded 55-rule stocks fixture, its reviewed/non-volume distinction, source hash, and file hash. Move eligibility under the current engine feature owner; reject the V2 package/state owner. | `P-C9-TAQ` |
| `internal/massive/taq_product.go` `c8f8fe...3fc` | `behavior evidence` | Preserve paired post-ack causal coverage, qualifying-original disclosure, five/one-second Tape windows, locked-zero, two-second quote carry, weighted median, complete state clearing, pressure thresholds, and rank-order restoration. Adapt Spread coverage to the owner-selected four-of-five-second boundary. Replace its mutex-owned ranking/readiness/lifecycle/command state with typed facts and state in the sole current engine. Remove the unevidenced Tape Rate attention threshold and optional exact lifecycle mutation path. | Both |
| `internal/massive/taq_product_state_test.go` `c4f3ef...81d` and `phase3_taq_resilience_test.go` `759e16...170` | `fixture/oracle evidence` | Preserve pre-ack rejection, mixed-frame aggregate independence, fresh warming after gaps, locked/crossed/partial-coverage oracles, SIP fallback, deduplication, unknown/non-volume condition exclusion, write-before-ack causality, and recovery ordering. Replace V2 provider/health/capability objects with current C5 and engine boundaries. | Both |
| `internal/massive/live_taq_capacity_test.go` `977d05...d40` and `capacity_artifacts.go` `003eb9...97b` | `measurement-shape evidence` | Preserve current oldest-frame age, current processing age, queue/heap/goroutine samples, bounded manifests, and explicit headroom/limitation reporting. Do not import its credentialed procedure, selector, health owner, results claim, or 10-second acceptance limit. | `P-C9-PRESSURE` |

No HTTP, browser, checkpoint, full-universe selector, full-session T/Q replay,
credential, live-provider, V2 health/readiness owner, or unrelated source was
inspected or whitelisted.

## 9. Exact component requirements

`C9-MEMBER-01` — On every engine publication whose ranking mode is exactly
`qualified_current`, desired T/Q membership is the displayed rows in existing
rank order, capped at 20. Every other mode has empty promotable membership.
T/Q never changes aggregate values, qualification, rank, committed `T`, or
backend readiness.

`C9-COVER-01` — The engine issues at most one paired `T.symbol,Q.symbol`
command at a time for the current binding and connection epoch. Removal
precedes addition. A successful C5 write fact does not create coverage. The
correlated successful terminal acknowledgement creates both channel coverage
strictly after its final causal position; a failed, ambiguous, stale, extra,
or wrong-token outcome creates no coverage. Stale-epoch, already-completed,
duplicate, and wrong-token facts are fenced/counted and cannot mutate any
current membership, coverage, pending command, or cleanup state. Only a failed
or ambiguous terminal correlated to the current outstanding command closes its
affected coverage. Because a partial provider success may already have changed
one wire channel, that matching outcome makes the affected symbol
`provider_membership_unknown`. The only further T/Q command permitted for it
is one paired cleanup unsubscribe. A successful cleanup proves absent
membership; a failed or ambiguous cleanup leaves the unknown count nonzero and
blocks promotion until natural connection-epoch replacement. The engine never
reports zero provider membership while an unknown remains. Unsubscribe intent
closes coverage before the command leaves the engine. Connection loss/epoch
change, rank removal, intentional rejection, session end, state-bound failure,
and shutdown also close affected coverage.

`C9-TAPE-01` — Tape Rate counts distinct original trade identities
`(trading_date,symbol,exchange,trf_present,trf_id,trade_id)` whose normalized
effective event time lies in `[T-5s,T)` and whose condition set is fully
reviewed and volume-updating in the embedded fixture. An empty condition set is
eligible. Unknown, unreviewed, known non-volume, incomplete-identity, and
lifecycle-adjustment records are observed but do not count. Exact duplicates
do not count twice; unequal repeats invalidate Tape Rate until new coverage.
The one-second value is the count in `[T-1s,T)` divided by one; the five-second
value is the count in `[T-5s,T)` divided by five. One-second output warms for
one second and five-second output warms for five seconds of continuous
post-ack coverage. Genuine covered silence is numeric zero. Output explicitly
states `qualifying_original_prints`, participant/SIP-fallback/mixed timestamp
basis, and whether unapplied lifecycle records occurred; C9 does not claim an
exact corrected tape and defines no attention threshold.

`C9-SPREAD-01` — Q-channel events are provider NBBO observations. The causally
latest positive two-sided quote with `ask >= bid` is feature-valid; locked
quotes are valid zero. Compute cents as `100*(ask-bid)` and basis points as
`10000*(ask-bid)/((ask+bid)/2)`. Publish that numeric spread immediately with
nonnegative age `max(0,T-SIP)` while acknowledged Q coverage remains
continuous. Age greater than two seconds changes status to `stale` but retains
both numeric values. A causally latest one-sided quote makes Spread unavailable;
a causally latest crossed quote makes it invalid. Neither may fabricate or
replace the retained valid numeric quote, and neither exposes that numeric
value while it is the latest observed quote state.

The engine retains only the latest observed quote state and latest valid quote
per selected symbol, O(1) state independent of quote rate. The correlated
acknowledgement causal position remains the admission boundary. A real coverage
gap, unsubscribe, connection loss/epoch change, session end, or retained-state
failure clears both retained states. Quiet time and an empty delivery window do
not clear them.

`C9-STATUS-01` — Membership and each channel/field independently report a
bounded state and reason. At minimum: `unselected`, `warming`, `current`,
`stale`, `unavailable`, `invalid`, and `pressure_shed`, with reasons for
subscription pending, coverage warming, insufficient coverage, stale quote,
crossed quote, condition/identity ambiguity, incomplete lifecycle, epoch/control
failure, state bound, and pressure. One field may be current while the other is
unavailable. None of these states gates aggregate readiness.

`C9-PRESSURE-01` — The engine consumes fixed-cardinality samples of current
waiting raw-frame count/bytes and capacities, oldest waiting raw-frame age,
active/classifying-frame age, separate cumulative slot/byte capacity drops,
aggregate watermark lag, transport/T/Q accounting integrity, T/Q work
presence, and the existing retention-bound state. Waiting measurements exclude
the active/classifying frame; an empty waiting queue has zero oldest-waiting
age. Active-frame age, heap, goroutine count, generic delivery latency,
delivery-family attribution, checkpoint metrics, and checkpoint mode are not
global pressure predicates. It owns three states:

Each sample is requested by one private engine-issued command containing the
current binding, monotonically increasing sample sequence, and engine issue
time. Exactly one command may be outstanding. The returned scalars are
accepted only for that command within two seconds; stale, duplicate,
out-of-order, foreign-binding, and superseded samples are fenced without
changing pressure. Entry and recovery require consecutive accepted one-second
samples; caller time and queued-result count cannot manufacture a streak.

An engine timer expires an outstanding sample command two seconds after its
engine issue time. Expiry clears the command, counts/fences the missing result,
and permits a fresh sequence. A missing or late sample is diagnostic and cannot
shed T/Q; it also cannot advance a consecutive entry or recovery streak. Empty
waiting windows are healthy unless direct waiting, loss, bound, accounting, or
watermark evidence says otherwise.

| State | Entry | Consequence |
| --- | --- | --- |
| `normal` | Default/recovered | Selected T/Q is normalized and admitted. |
| `taq_degraded` | Any one of oldest waiting age >=1 second, waiting frames >=10% of frame capacity, or waiting bytes >=10% of byte capacity persists for two consecutive one-second samples | Close all T/Q coverage and reject T/Q elements before expensive normalization while retaining provider membership; continue classifying every mixed frame. |
| `aggregate_only` | Any one of oldest waiting age >=2 seconds, waiting frames >=25%, or waiting bytes >=25% persists for three consecutive one-second samples; aggregate watermark lag is >2 seconds for two consecutive samples while T/Q work/membership exists; or a new slot/byte capacity drop, queue/adapter/transport/T/Q accounting loss, or global retention bound occurs immediately | Keep early T/Q rejection and request paired unsubscribe for every known provider member, lowest current rank first, down to zero known membership; ambiguous members remain explicitly unknown until cleanup or epoch replacement. |

Recovery requires exactly five consecutive accepted one-second samples with
waiting frames and bytes each below 1% capacity, oldest waiting age below 250
ms, aggregate watermark lag <=1 second, no new capacity drops, coherent
queue/adapter/transport/T/Q accounting, and no global retention bound.
Active-frame age does not reset recovery. A result admitted after its two-second
pressure-command deadline is fenced before it can affect a streak. Recovery
returns to normal and issues at most one paired command at a time; each
correlated acknowledgement permits the next current-ranked symbol immediately.

Independently of global mode, one active raw frame has a 500-ms monotonic
classification budget. Once consumed, C5 continues parsing the frame but
rejects its remaining T/Q elements before expensive normalization; later
aggregate/control elements remain in exact array order. Each shed element is
counted and closes affected causal T/Q continuity, but active-frame age alone
cannot enter `taq_degraded` or `aggregate_only`. This adds no goroutine, queue,
or mutable owner.

Every restored symbol starts new coverage and warm-up. There is no restoration
sleep beyond the serialized provider acknowledgement: after one ranked paired
command succeeds, the engine may issue the next ranked paired command.

`C9-BOUNDS-01` — Per covered symbol retain only the trailing six seconds of raw
trade contributions/lifecycle timestamps, 16 minutes of compact trade-identity
fingerprints, and two quote records: the latest observed state and latest valid
state (which may be the same record). Hard per-symbol/global trade bounds are
50,000/500,000 contributions and 100,000/1,000,000 compact identities. Quote
state is O(1) per desired symbol and at most 40 records across 20 symbols.
Duplicates, incomplete identities, excluded conditions, and lifecycle records
are either counted without retention or charged to one of these bounded
families; there is no unaccounted retained slice/map. A symbol bound closes
only that symbol's affected coverage and requests unsubscription. A global
bound enters `aggregate_only`. All counters and reasons are fixed-cardinality.
The global-bound emergency transition and removal intent are part of S1's
fail-closed state-bound containment; S2 later adds sample-driven pressure and
restoration but does not weaken that invariant.

## 10. Ownership and trust boundaries

The engine is the only owner of desired/provider membership, command tokens,
acknowledged per-channel coverage, retained feature state, measurements,
pressure state, and restoration order. C5 continues to own socket I/O,
frame/array order, normalized T/Q shapes, command serialization, and status
correlation. C8 supplies scalar pressure observations and drives issued
commands; it cannot select membership, declare coverage, compute features, or
change pressure state.

Typed engine commands have private identity and are issued only for its current
binding/epoch. Typed T/Q admissions carry normalized fields and current causal
position. Invalid states prevented by construction include caller-selected
rank membership, caller-selected coverage start, a public feature setter, and
T/Q mutation of the aggregate evaluator. Runtime validation rejects foreign
bindings, stale epochs, pre-ack positions, wrong symbols/channels/tokens,
malformed times/numbers, duplicate causal positions, and commands after an
epoch-local control failure.

The smallest consequential false successes are: a pre-ack trade warming a
window, one successful status element creating paired coverage, a crossed quote
contributing negative spread, covered silence represented unavailable, a T/Q
drop leaving old current values, pressure dropping a mixed aggregate/control
frame, and T/Q changing rank/readiness. Each is an explicit primary-proof
counterexample.

## 11. Interface and state shape

The engine adds closed `TQTradeInput` and `TQQuoteInput` admissions, private
engine-issued `TQCommand` and `TQPressureCommand` values, a closed pressure
result admission that preserves the latter command, and a defensive immutable
`TQView`. The view contains pressure/control state, desired/provider counts,
command/accounting counters, and at most 20 rank-ordered symbol rows with
membership, channel coverage, feature state/reason/value, coverage duration,
timestamp basis, lifecycle-disclosure count, quote quality, and event/drop
counters. It contains no provider text or unbounded label map.

The C5 attempt accepts an engine command only through a conversion that
preserves its binding/epoch/token/action/symbol. Its normalization option is an
atomic attempt-local boolean derived from the engine pressure view; even when
set, cursor classification continues element by element and aggregate/control
delivery is unchanged. Each intentional T/Q normalization rejection is
counted. C5 exposes waiting raw-frame count/bytes/capacities and oldest waiting
age separately from diagnostic active-frame age, plus separate cumulative
slot/byte drops and T/Q classified/normalized/rejected/pressure-shed accounting,
without changing queue ownership. C8 adds a separate one-second maximum delivery-delay accumulator
that resets only when the pressure sample is admitted; its existing lifetime
maximum remains unchanged for operator reporting. The operations timer samples
pressure once per second, and command synchronization runs after timer and
delivery completions without a second owner.

C8 exposes the same one-second maximum with a closed diagnostic work-family
attribution and exact per-family delivery counts. C9 consumes the duration and
one fail-closed fact stating whether its atomic winning record has reconciled,
non-`unknown` attribution. The exact family, tie order, and attribution counts
are not pressure predicates and cannot alter thresholds, degradation,
aggregate-only entry, shedding, or restoration order.

A mixed-frame or required aggregate/control classification ambiguity is not a
scalar C9 pressure sample: C5 emits the existing aggregate-ingress integrity
fact and the engine follows recovery/suppression. Scalar queue, adapter,
transport, or T/Q normalization accounting incoherence immediately enters
`aggregate_only`; it cannot fabricate aggregate corruption or readiness.

Checkpoint and replay schemas remain unchanged: ephemeral membership,
coverage, pressure, and instantaneous measurements start empty after restart,
and T/Q is unavailable in V1 aggregate replay.

## 12. Accounting identities

For each T/Q family, `consumed = applied + duplicate + rejected + fenced +
pressure_shed + integrity`. Commands reconcile as `issued = pending +
acknowledged + failed + fenced`; provider membership partitions into known
present, known absent, and ambiguous/unknown for every affected desired or
cleanup symbol, with known present plus unknown at most 20. Desired membership
equals the current qualified rows and at most 20. The mutable membership map is
the union of at most 20 desired symbols and at most 20 distinct present/unknown
cleanup liabilities; a known-absent, non-desired, retention-empty member is
deleted immediately, so rank churn cannot retain historical universe entries.
Retained raw,
identity-fingerprint, lifecycle, and quote totals equal the sum of their
per-symbol counts and never exceed their separate stated bounds. Pressure
transitions and intentional drops are monotonic counters. Aggregate
admission/evaluation accounting must be byte-for-byte unchanged across a T/Q-
only trace except for the engine's generic nonmarket admission/transition
counters.

## 13. Primary proofs

`P-C9-TAQ` is one compact deterministic engine/C5 trace plus the exact 20-row
membership boundary. It proves desired membership, private paired command correlation,
both acknowledgement elements represented by the final C5 boundary, pre-ack
rejection, post-ack warming, exact trade identity/dedup/condition/SIP-fallback
behavior, quiet covered numeric zero, locked/crossed/one-sided/current spread,
retained numeric stale spread with increasing quote age, independent field
reasons, O(1) quote retention, the exact configured bound constants plus structurally identical overflow
containment at small private proof bounds, gap clearing, rank
removal, connection-loss clearing, and fresh resubscription. It snapshots
aggregate evaluation/rank/readiness before and after and requires exact
equality. Limitation: the reviewed fixture supports
qualifying originals but not complete provider lifecycle reconstruction or
live entitlement.

`P-C9-PRESSURE` is one controlled mixed-frame/runtime trace. It crosses exact
10%/25%, one-/two-second, two-/three-sample, byte/frame, capacity-drop,
accounting, retention, and watermark-lag boundaries; proves the active frame is
excluded from waiting pressure; proves T/Q is rejected only after element
classification; proves the 500-ms frame-local budget sheds later T/Q while a
later aggregate and control fact in the same
raw frame still apply, reaches zero provider T/Q membership, preserves
aggregate watermark/ranking/readiness, holds recovery during unhealthy/dwell
samples, then restores current symbols one-by-one in rank order with new
warming state. It separately proves a partial/ambiguous subscribe followed by
successful cleanup and a failed cleanup that remains
`provider_membership_unknown`, never false zero. It injects a T/Q-local
accounting failure into `aggregate_only`, while a mixed-frame accounting
ambiguity follows the existing global ingress-integrity path. It validates all
pressure/drop/command/accounting identities and bounds before interpreting
results. A delayed/duplicate pressure result is fenced and cannot advance
engine-time recovery dwell. Missing diagnostic results are counted/fenced and
fresh sampling continues, but silence alone cannot change pressure. Separate
distinguishing cases prove empty
windows advance five-sample recovery, heap/goroutine/generic-latency/attribution
spikes cannot change pressure, a new cumulative capacity drop enters
`aggregate_only` immediately without making the old cumulative count permanent,
and missing diagnostic samples do not create pressure. Limitation: the proof
validates policy behavior at conservative direct gates; it does not locate a
current-host saturation frontier or claim provider capacity, market-hours
behavior, or an SLA.

## 14. Verification tiers and acceptance

During correction, run the narrow engine or adapter proof under a two-minute
bound. Ordinary verification remains `go test -short -timeout 2m ./...`.
Affected engine/massive/operations race packages run under five minutes. Both
primary proofs are ordinary compact tests unless a later measured pressure
fixture needs explicit non-short selection; no local command may exceed 15
minutes. C9 final acceptance requires both proofs, exact accounting and bound
validation, a success/failure walkthrough, clean ordinary/race results, and one
final read-only review. Unchanged C1-C8 expensive evidence is reused.

## 15. Sequential slices

`C9-S1` owns engine desired membership, command authority, coverage, bounded
trade/quote state, measurements/view, fixture adaptation, C5 conversion, and
`P-C9-TAQ`. It completes `C9-MEMBER-01`, `C9-COVER-01`, `C9-TAPE-01`,
`C9-SPREAD-01`, `C9-STATUS-01`, and the feature portion of `C9-BOUNDS-01`.
It also owns the invariant emergency `aggregate_only`/unsubscribe containment
for a global retained-state bound. Ordinary pressure sampling, early shedding,
and restoration remain unavailable.

`C9-S2` adds engine pressure admissions/state, C5 early T/Q rejection, C8
sampling/command synchronization, complete unsubscription/restoration, metrics,
and `P-C9-PRESSURE`. It completes `C9-PRESSURE-01` and remaining bounds and
integration. It cannot revise feature meaning or create a second state path.

## 16. Success, failure, and shutdown walkthrough

On success, a qualified publication creates desired rows; the engine issues a
paired command; C5 writes and correlates all statuses; only the successful final
boundary opens coverage; covered normalized events mutate bounded engine state;
and timers project Tape Rate and Spread at committed `T`. Ranking can progress
before, during, or without this work.

On isolated failure, the engine rejects/fences the T/Q fact or command, closes
only affected coverage, publishes explicit field reasons, and keeps aggregate
state current. On pressure, the engine first closes all measurements and asks
C5 to shed T/Q elements, then removes known wire membership to zero if pressure
persists while retaining any ambiguous membership as explicitly unknown.
Recovery uses current ranks, not stale desired order. Connection loss,
session end, controlled stop, and Runtime shutdown close T/Q coverage and
best-effort provider membership without delaying the already bounded aggregate
termination path.

## 17-19. Corrections, review triggers, and deferrals

### C9-S1 acceptance record

The engine now owns qualified-current desired membership, opaque paired T/Q
commands, exact C5 acknowledgement correlation, per-channel post-ack causal
coverage, bounded Tape Rate and Spread state, feature projection, and
retained-state containment. C5 carries bounded raw condition, quote-quality,
side-presence, source-binding, and causal evidence without deciding feature
eligibility. Aggregate ranking, watermark, and readiness remain independent of
T/Q availability.

`P-C9-TAQ` passed through both the compact engine trace and the loopback
WebSocket/C5 composition. Its dangerous counterexamples include pre-ack and
non-increasing causal facts, exact versus semantically unequal trade repeats,
reviewed eligible/non-volume/unreviewed conditions, one-sided and out-of-order
quotes, stale/foreign command results, ambiguous membership cleanup,
per-symbol/global overflow, normalization drops that would leave old values
current, foreign-binding drops, symbol-less drops, and aggregate-only entry
with both undispatched and already-written additions. The last case proves an
undispatched subscribe is retired, while an already-written subscribe can
create only cleanup liability and never reopen coverage.

Final S1 verification passed `go test -short -timeout 2m ./... -count=1`, the
affected race gate passed for engine, Massive, and operations under five
minutes, and `git diff --check` passed. Focused independent re-review found no
remaining P1/P2. The proof is deterministic rather than scheduler stress, but
the engine lock, FIFO admission, opaque one-shot command, and terminal result
branches enforce the reviewed interleavings. Pressure sampling, early adapter
shedding, timed degradation/aggregate-only transitions, and ranked restoration
remain solely in S2; its contracted boundary remains valid.

### C9-S2 and final acceptance record

This is the historical original S2 acceptance record. Later owner correction
records below supersede its thresholds, missing-sample behavior, and restoration
pacing while preserving its still-valid ownership and aggregate-independence
evidence.

The engine now owns one fixed one-second pressure-sampling authority with
opaque, monotonically sequenced commands, a two-second terminal deadline, and
bounded missing-sample progress. The first missed sample enters
`taq_degraded`; a second enters `aggregate_only`. Timely unhealthy samples use
the contracted dwell thresholds, while recovery requires a continuous healthy
sample sequence for 30 seconds and at least 15 seconds since degradation. A
sampling blackout cannot manufacture recovery. Degraded mode closes displayed
T/Q measurements and enables C5 early T/Q shedding without dropping later
aggregate/control elements from the same frame. Aggregate-only additionally
removes provider membership; restoration follows current rank order with fresh
coverage/warming and at most one successful addition per five seconds.

`P-C9-PRESSURE` passed through the engine policy trace and the concrete
Runtime/C5 loopback composition. It proves mixed-frame aggregate/readiness
survival under T/Q shedding, deterministic low-rank removal to known zero,
unknown-membership preservation, continuous-health recovery, ranked warming,
missing and late result containment, T/Q-local accounting failure, and broad
queue/adapter ambiguity routing through the existing global ingress-integrity
path. Queue occupancy includes every slot-consuming frame/marker, raw oldest
age remains separately measured, quiescent membership is pruned, and fact,
command, pressure, retained-state, and provider-partition accounting identities
remain exact. Global retained-state containment is permanent; every
aggregate-only cause enables early shedding.

Corrections rejected the dangerous counterexamples where a 30-second sampling
blackout recovered on one healthy result, a symbol-less/global bound left C5
shedding disabled, marker saturation understated occupancy, rank churn grew an
unbounded member map, or a live-epoch reset regressed operator counters and
reused pressure sequence numbers. The final epoch proof seeds nonzero
identity-consistent counters, replaces the connection epoch, retires the old
pending T/Q command into its fenced terminal, strictly advances pressure
sequence, and fences both old opaque result types without recreating
membership. Process/binding-lifetime counters and pressure state survive while
epoch-local membership and pending authority do not.

Final verification passed `go test -short -timeout 2m ./... -count=1`,
`go test -race -short -timeout 5m ./internal/engine ./internal/massive
./internal/operations -count=1`, and `git diff --check`. The mandatory final
read-only review and focused re-reviews found no remaining P1/P2. The proof is
deterministic loopback/engine evidence rather than a scheduler soak or measured
host saturation frontier; the conservative numeric gates remain provisional
safety settings, and no provider credential, market-hours, capacity, or SLA
claim is made. Component 10 may rely on the accepted C9 view and accounting
boundary without making ranking depend on T/Q health.

### Historical C9-S2-D direct-pressure/latest-valid-quote correction — 2026-08-13

Direct owner instruction replaces the prior weighted Spread product meaning
and invalidates C9 pressure gates that did not prove shared-feed consumption
overload. Ordinary live startup now has no T/Q profile switch: desired
membership is exactly the `qualified_current` displayed rank prefix up to 20.
One paired command remains in flight, and each correlated acknowledgement
permits the next ranked addition without a restoration sleep.

Tape Rate remains current through acknowledged quiet coverage and reports
numeric `0.0` for empty trailing windows. Spread is the causally latest valid
two-sided non-crossed quote, retained as two numeric values plus nonnegative
age; age over two seconds marks the value stale without hiding it. A latest
one-sided/crossed observation remains unavailable/invalid. The engine keeps at
most the latest observed and latest valid quote record per desired symbol.

Pressure now uses only direct evidence. Queue occupancy at least 50% or oldest
unread age at least 250 ms for two continuous seconds enters `taq_degraded`.
Occupancy at least 80% or oldest unread age at least 1.5 seconds for five
continuous seconds enters `aggregate_only`. A new queue-capacity drop, a T/Q
retention-bound hit, or transport/T/Q accounting loss enters `aggregate_only`
immediately. Recovery requires five continuous seconds below 20% occupancy and
100 ms oldest unread age with no new loss. Heap, goroutines, generic delivery
latency, delivery-family attribution, empty delivery windows, and diagnostic-
sample silence cannot enter pressure or reset recovery.

The distinguishing proofs cover quiet zero Tape Rate without membership loss,
numeric stale Spread and increasing age, one-sided/crossed containment, exact
20-row desired membership, backlog degradation/unsubscription, empty-window
recovery, immediate capacity loss, diagnostics-only spikes, ranked serialized
restoration, separate normalized/applied/shed trade and quote counters, and
aggregate evaluation/watermark/readiness independence. The first full ordinary
run hit the pre-existing C8 bounded-exhaustion timing test once; its exact rerun
passed, and the subsequent full ordinary run passed. Required affected race,
vet, UI-model/visual, and diff gates also pass. No credential, provider request,
or live-capacity claim was made.

### C9-S2-W waiting-pressure/frame-local correction — 2026-08-13

The checkpoint-off owner run invalidated only the accepted pressure-definition
claim. All 20 paired subscriptions were acknowledged; 1,109 trades and 527
quotes applied; Tape Rate and Spread populated; aggregate ranking stayed
current; watermark lag was 0-1 second; queue high-water was 43/32,768 frames;
capacity drops and recovery attempts were zero; accounting and retention were
healthy; and process load remained 14-15 goroutines at substantially less than
one CPU core. The sole transition cause was `oldest_unread_frame` because the
sample treated the active/classifying frame as waiting from receipt through all
serial element admissions. Normal provider batches could therefore exceed the
old 250-ms threshold with no waiting backlog and could prevent the old below-
100-ms recovery predicate indefinitely.

This reopens only C9-S2 pressure/shedding/restoration. C5 now reports waiting
raw frames, waiting bytes, capacities, and oldest waiting age independently
from diagnostic active-frame age. The engine applies the exact owner-selected
10%/1-second/two-sample transient degradation, 25%/2-second/three-sample
aggregate-only escalation, immediate separate capacity-drop/accounting/bound
containment, and >2-second/two-sample watermark-lag guard with T/Q work. Five
exact healthy samples recover at below 1% frames/bytes, below 250-ms oldest
waiting age, and at most one-second aggregate lag. Missing samples cannot shed
or advance a streak. Active-frame age participates in none of those decisions.

An independent 500-ms monotonic budget protects one unusually large active
frame. The existing cursor continues parsing in array order, pressure-sheds
only remaining T/Q before expensive normalization, admits later aggregates and
controls, and routes each shed element through ordered engine drop accounting.
The affected coverage is closed and must re-establish through the existing
paired unsubscribe/resubscribe acknowledgement path; the frame alone does not
change the global pressure mode. No extra queue, goroutine, state owner,
top-one probe, minimum-degraded timer, or checkpoint dependency is introduced.

The correction proof allocation is the exact boundary/sample tests, waiting-
versus-active queue test, monotonic mixed-frame budget/order/accounting test,
frame-local coverage-gap test, Runtime sampling/command trace, existing T/Q
feature trace, and aggregate state/readiness equality checks. Deterministic
proof cannot establish provider arrival distributions, live saturation, or an
SLA. No credential, predecessor, provider request, process stop, live run,
checkpoint redesign, ranking change, API redesign, or UI redesign is authorized.

The allocated focused proofs pass. Repository ordinary verification passed
`go test -short -timeout 2m ./...`; affected engine, Massive, operations, and
snapshot API race verification passed under a five-minute command timeout;
affected-package `go vet` and `git diff --check` pass. No independent review was
triggered: the correction reuses the sole FIFO cursor, queue, engine owner, and
ordered drop admission, and its concurrency/order consequence is explicit by
construction and primary proof. D8 is accepted locally with live/provider
behavior still unclaimed.

### Historical, superseded C9-S2-H current-host heap-gate correction — 2026-08-13

The latest direct owner correction invalidates these gates. This subsection is
non-executable history; heap no longer participates in C9 pressure.

The preserved 2026-08-12 live snapshot invalidated only C9's provisional heap
profile. Its 1,785,959,440-byte aggregate baseline exceeded the old immediate
1.25-GiB aggregate-only gate and could never satisfy the old below-384-MiB
recovery gate. The owner selected a private 8-GiB Apple M1 profile of below
2.5 GiB for recovery, 3.25 GiB for degraded entry, and 4 GiB for immediate
aggregate-only entry. Queue occupancy, oldest-frame age, delivery delay,
goroutine, sample expiry, accounting, shedding, unsubscription, restoration,
and aggregate-independence behavior are unchanged.

`TestPC9CurrentHostHeapGateBoundaries` proves the exact configured constants,
strict ordering, the observed baseline remaining normal across timely samples,
the equal-3.25-GiB 500-ms dwell, immediate equal-4-GiB containment, rejection
of recovery at exactly 2.5 GiB, and recovery one byte below only after 30
continuous healthy seconds. The existing pressure proof still checks exact
aggregate-evaluation and committed-watermark equality across pressure changes.

Focused proof, `go test -short -timeout 2m ./... -count=1`, affected
engine/Massive/operations race verification under five minutes, `go vet ./...`,
and `git diff --check` passed. The required `gpt-5.6-sol` medium read-only final
review found no P1/P2 and one P3 documentation unit mismatch; the ledger now
states the exact 1.663-GiB baseline and approximately 1.59-GiB normal-mode
headroom. No provider request or credential access occurred. The revised
profile is current-host policy, not evidence of market-hours T/Q behavior, a
portable capacity limit, or an SLA.

### Historical, superseded C9-S2-L attributed recovery delivery-gate correction — 2026-08-13

The latest direct owner correction invalidates this gate. This subsection is
non-executable history; generic delivery latency and attribution no longer
participate in C9 pressure.

Owner direction revised only the recovery delivery predicate from below 500 ms
to an attributed one-second maximum strictly below 1 second. Runtime derives
one fail-closed boolean from C8's atomic maximum pair and fixed-cardinality
identity: the maximum duration must match, all seven family counts must
reconcile to deliveries, and the winning family must be non-`unknown`. C9 does
not consume the exact family or counts. An unknown, absent, mixed, or incoherent
winner therefore resets recovery dwell without changing degradation or
aggregate-only entry.

`TestPC9RecoveryDeliveryGateBoundariesAndAttribution` proves the configured
recovery value is exactly 1 second, exactly 1 second cannot recover, and 1
second minus 1 nanosecond recovers only after 30 continuous attributed seconds.
It also proves 31 equally low unattributed samples cannot establish dwell and a
known-attributed result admitted 1 nanosecond after the two-second command
deadline is fenced with `recoverySince` still zero. The same proof asserts the
unchanged 2-second degraded and 5-second aggregate-only delivery thresholds;
500-ms/2-second persistence dwells; 50/80/20-percent queue gates; 250-ms/1.5-s/
100-ms oldest-frame gates; 2.5/3.25/4-GiB heap gates; 48/64/128 goroutine gates;
and 30-second recovery dwell. The existing primary pressure trace again proves
aggregate evaluation and committed watermark equality across the full
degrade/unsubscribe/recover/restore path.

Focused engine/operations proof passed in 0.708/0.771 seconds. Ordinary
`go test -short -timeout 2m ./... -count=1` passed. Affected
`go test -race -short -timeout 5m ./internal/engine ./internal/massive
./internal/operations ./internal/snapshotapi ./cmd/scanner -count=1` passed;
`go vet ./...` and `git diff --check` passed. No provider request, credential,
live observation, push, or capacity/SLA claim occurred. No independent review
was triggered because this is a scalar recovery-policy revision with a
fail-closed derived predicate, not a new ownership, ordering, persistence, or
external-evidence boundary.

### Historical, removed C9-S2-P explicit top-one current-host probe — 2026-08-13

The latest direct owner correction removes this profile and restores ordinary
ranked desired membership up to 20. This subsection records the observation
that motivated the correction; it defines no current flag, profile, or proof.

The owner-authorized checkpoint-off live run invalidated the lower-level
premise that the conservative normal recovery profile could provide even a
single observable T/Q trial on the current 8-GiB host. Aggregate processing was
materially improved without checkpoints: the ordinary watermark was usually
one second behind, the queue was commonly 0-35 frames, the provider connection
remained in epoch 1 with zero recovery attempts, and checkpoint work stayed
zero. The T/Q path nevertheless acknowledged one paired subscription, consumed
15 T/Q facts, pressure-shed 13, acknowledged cleanup, and retained no trades or
quotes. Recovery samples alternated between approximately 1.70-2.15 GiB with
sub-second delivery and 2.51-2.60 GiB with 1.136-1.223-second delivery, so the
strict below-2.5-GiB/below-one-second predicates repeatedly reset the 30-second
dwell before another subscription could be issued.

The correction adds one explicit, non-default scanner startup profile named
`top-one-probe`. It is diagnostic delivery configuration, not normal product
coverage: normal remains every displayed row up to 20 under `PG-TAQ-01`. The
probe derives desired membership from exactly the first current
`qualified_current` row, preserving existing aggregate Day-%/symbol rank order;
it cannot select by Activity, T/Q arrival, or any browser fact. It retains one
paired command in flight, acknowledgement-defined causal coverage, fresh
warm-up, five-second restoration pacing, mixed-frame classification, and every
normal degradation and aggregate-only predicate. Only recovery changes to heap
strictly below 3 GiB, attributed one-second delivery maximum strictly below 1.5
seconds, and 10 continuous healthy seconds. Queue, oldest-frame age,
goroutines, attribution, accounting, and minimum-degraded predicates remain
unchanged.

`TestPC9TopOneProbeProfileIsExplicitAndBounded` is the primary correction
proof. It must prove the default profile still desires two ranked fixture rows
and retains the complete accepted normal pressure policy; the probe desires
only rank 1; exact 3-GiB and 1.5-second boundaries cannot recover; one byte and
one nanosecond below recover only after 10 continuous healthy seconds; and the
2-second degraded, 5-second aggregate-only, 3.25-GiB degraded, and 4-GiB
aggregate-only gates are identical to normal. Scanner flag parsing must reject
the probe in replay and reject unknown profiles. Existing `P-C9-PRESSURE`
continues to prove aggregate evaluation and committed-watermark independence.

This profile answers only whether one acknowledged rank-1 stream can remain
covered and produce Tape Rate/Spread evidence without making aggregates stale
on this host. It does not establish top-20 capacity, provider entitlement,
portable thresholds, or an SLA. A live failure leaves the normal product
contract unchanged and routes the next decision to resource fit, not another
subscription architecture.

The correction proof passed with the existing normal pressure regressions and
scanner configuration proof. `go test -short -timeout 2m ./... -count=1`,
affected `go test -race -short -timeout 5m ./internal/engine
./internal/operations ./cmd/scanner -count=1`, `go vet ./...`, and
`git diff --check` all passed. No independent review was triggered: the closed
profile changes scalar delivery settings and a privately validated desired-rank
prefix without changing provider acknowledgement causality, mutable ownership,
persistence, concurrency linearization, or external-evidence acceptance.

Reopen S1 if a normalized shape cannot carry the stated identity/quality facts,
paired acknowledgement cannot prove per-symbol/channel coverage, retained quote
age/value semantics become ambiguous, or any T/Q fact can change aggregate
evaluation. Reopen S2 if early rejection can hide a mixed-frame aggregate or
control item, pressure inputs are not recent/fixed-cardinality, complete
unsubscription cannot distinguish known zero from unknown membership, or
restoration bypasses rank order/warm-up.
Revise the lowest unsuitable contract, fixture, interface, threshold, or proof
and rerun the narrowest distinguishing test.

The completed contract receives a focused read-only review before S1 because
coverage/command causality and mixed-frame pressure are consequential external
ordering boundaries. Each accepted slice records its proof/counterexample,
limitations, ownership change, deferred behavior, and whether the next slice
remains valid. Full lifecycle-adjusted Tape Rate, full-universe or full-session
T/Q, T/Q replay, credentials/live provider checks, market-hours validation,
attention/trading labels, API/UI serialization, and public deployment remain
deferred.
