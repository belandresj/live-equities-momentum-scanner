# Top-20 T/Q coverage and features

**Status:** Component 9 finally reaccepted 2026-08-13 after the owner-selected
current-host heap-gate correction; membership, features, non-heap pressure
gates, and prior evidence remain accepted

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
| `C9-S1` membership/coverage/features | `accepted_after_owner_revision` | Owner shortened Spread to a five-second weighted median with four-of-five valid-duration coverage; focused boundary/window proof, ordinary/race verification, and focused read-only review pass | Complete |
| `C9-S2` pressure/shedding/restoration | `accepted` | `P-C9-PRESSURE`; ordinary and affected race gates; focused corrections and re-review clean | Complete |
| `C9-S2-H` current-host heap gates | `accepted_after_correction_review` | The 2026-08-12 preserved live snapshot measured 1,785,959,440 bytes of Go heap while T/Q was `aggregate_only`; the 512-MiB degraded, 1.25-GiB aggregate-only, and 384-MiB recovery gates made normal T/Q and recovery structurally unreachable for that aggregate baseline. Owner direction on 2026-08-13 selected 2.5-GiB recovery, 3.25-GiB degraded, and 4-GiB aggregate-only heap gates for the private 8-GiB Apple M1 host. Exact boundary proof, ordinary/race verification, vet/diff, and focused final review pass. | Complete locally; current-host live behavior remains to be observed without claiming portable capacity. |
| Final component review | `accepted_after_heap_correction` | Original reviews remain valid; the 2026-08-13 focused read-only re-review found no P1/P2 and one P3 unit mismatch, corrected before reacceptance. | Complete; reopen only through the V1 correction loop |

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
- implement five-second time-weighted median validated NBBO spread in cents and
  basis points, including locked-zero and unavailable crossed/one-sided/stale/
  insufficient-coverage behavior;
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

`C9-SPREAD-01` — Q-channel events are provider NBBO observations. Positive
bid/ask with `ask >= bid` is feature-valid; locked quotes are valid zero and
crossed quotes are an invalid timeline state. Bounded condition/indicator
metadata is retained as reviewed-ordinary, known-special, or unclassified
quality evidence but is not guessed into a disallow rule without a reviewed
source. Each valid quote carries from its SIP time until the earlier of the
next quote, two seconds, or `T`. Within `[T-5s,T)`, separately compute cents
`100*(ask-bid)` and basis points
`10000*(ask-bid)/((ask+bid)/2)`. The weighted median is the first sorted value
whose cumulative duration is at least half of total valid duration. Spread is
current only when the latest timeline state at `T` is valid and no older than
two seconds and at least four seconds of the five-second window has valid quote
duration. Less than four seconds is `warming` before four seconds of channel
coverage and `unavailable/insufficient_coverage` afterward. A latest crossed
state is `invalid`; an expired valid quote is `stale`. Cents and basis points
are absent in every non-current state.

Every quote-duration segment is additionally intersected with the current
acknowledged Q-channel coverage interval. A causally post-ack late quote whose
SIP time precedes the acknowledgement can be retained as event-time evidence,
but contributes no duration before `max(SIP,Q-coverage-start)` and can never
help satisfy coverage outside that intersection. `Q-coverage-start` is the
normalized UTC receipt time of the correlated successful acknowledgement,
recorded atomically with its causal position and clamped to the session
interval `[S,E]`. The causal position remains the ordering boundary for which
items are covered; the clamped receipt time is the distinct market-time
boundary used for quote duration and warm-up. No caller or quote timestamp may
substitute for it.

Causally ordered quote observations may arrive out of SIP event-time order.
An observation with `SIP < T-10s` is rejected as too late for the mutable
Spread window. Otherwise it is inserted into the retained 12-second timeline
and may revise weighted duration on the next projection. Timeline order is
`(SIP time, live causal position)`; at the same SIP boundary the causally later
observation supersedes the earlier state. No observation is shifted to receipt
time or into a later window.

`C9-STATUS-01` — Membership and each channel/field independently report a
bounded state and reason. At minimum: `unselected`, `warming`, `current`,
`stale`, `unavailable`, `invalid`, and `pressure_shed`, with reasons for
subscription pending, coverage warming, insufficient coverage, stale quote,
crossed quote, condition/identity ambiguity, incomplete lifecycle, epoch/control
failure, state bound, and pressure. One field may be current while the other is
unavailable. None of these states gates aggregate readiness.

`C9-PRESSURE-01` — The engine consumes fixed-cardinality samples of current C5
queue occupancy/oldest-frame age, one-second maximum engine-delivery delay,
heap allocation, goroutine count, and decoder accounting. It owns three states:

Each sample is requested by one private engine-issued command containing the
current binding, monotonically increasing sample sequence, and engine issue
time. Exactly one command may be outstanding. The returned scalars are
accepted only for that command within two seconds; stale, duplicate,
out-of-order, foreign-binding, and superseded samples are fenced without
changing pressure. Persistence and recovery dwell use serialized engine
admission time, never caller sample time or the count of queued samples. A
delayed burst of healthy results therefore cannot manufacture 30 seconds of
recovery.

An engine timer expires an outstanding sample command two seconds after its
engine issue time. Expiry is an engine-owned terminal transition: it clears the
outstanding command so the next timer can issue a fresh sequence, fences every
later result for the expired sequence, and conservatively enters
`taq_degraded` on the first consecutive miss. A second consecutive expiry
enters `aggregate_only`; only a timely admitted sample resets the consecutive-
expiry count. Missing results or results delayed by the congestion being
measured therefore cannot leave pressure `normal` or disable subsequent
sampling.

| State | Entry | Consequence |
| --- | --- | --- |
| `normal` | Default/recovered | Selected T/Q is normalized and admitted. |
| `taq_degraded` | Any of queue >=50%, oldest >=250 ms, delivery >=2 s, heap >=3.25 GiB, or goroutines >=64 persists 500 ms | Close all T/Q coverage and reject T/Q elements before expensive normalization; continue classifying every mixed frame. |
| `aggregate_only` | Queue >=80%, oldest >=1.5 s, delivery >=5 s, heap >=4 GiB, goroutines >=128, or **T/Q-local** decode/accounting failure immediately; or degraded pressure persists 2 s | Keep early T/Q rejection and request paired unsubscribe for every known provider member, lowest current rank first, down to zero known membership; ambiguous members remain explicitly unknown until cleanup or epoch replacement. |

Recovery requires queue <20%, oldest <100 ms, delivery <500 ms, heap <2.5 GiB,
goroutines <48, and coherent decoder/transport accounting continuously for 30
seconds and at least 15 seconds since degradation. It returns to `normal`, then
restores one current desired symbol at a time in rank order, no faster than one
successful acknowledgement per five seconds. Every restored symbol starts new
coverage and warm-up. The queue, age, delivery, goroutine, and accounting gates
remain the conservative provisional V1 settings relative to C5's hard queue
limits and the product's two-second readiness tolerance. The revised heap gates
are a private current-host profile: they give the observed 1.786-GB
(1.663-GiB) aggregate baseline approximately 1.59 GiB before optional T/Q
shedding, retain a further 0.75 GiB before full provider unsubscription, and
permit recovery after T/Q state is cleared. They are not a measured saturation
frontier, portable capacity claim, or SLA; current-host live observation may
revise them again below fixed aggregate-correctness limits.

`C9-BOUNDS-01` — Per covered symbol retain only the trailing six seconds of raw
trade contributions/lifecycle timestamps, 16 minutes of compact trade-identity
fingerprints, and 12 seconds of quote timeline. Hard per-symbol/global bounds
are respectively 50,000/500,000 raw trade or lifecycle contributions,
100,000/1,000,000 compact identities, and 20,000/400,000 quote observations.
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
counted. C5 adds current frame capacity, current oldest raw-frame receipt age,
and cumulative T/Q-local normalization accounting without changing queue
ownership. C8 adds a separate one-second maximum delivery-delay accumulator
that resets only when the pressure sample is admitted; its existing lifetime
maximum remains unchanged for operator reporting. The operations timer samples
pressure once per second, and command synchronization runs after timer and
delivery completions without a second owner.

A mixed-frame or required aggregate/control classification/accounting failure
is not a C9 pressure sample: C5 emits the existing aggregate-ingress integrity
fact and the engine follows recovery/suppression. Only an identity proven to
contain T/Q-local classifications and intentional drops may set the
`TQLocalAccountingHealthy=false` pressure predicate. Queue and adapter
accounting must reconcile before a sample is admitted.

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

`P-C9-TAQ` is one compact deterministic engine/C5 trace with two rank-ordered
symbols. It proves desired membership, private paired command correlation,
both acknowledgement elements represented by the final C5 boundary, pre-ack
rejection, post-ack warming, exact trade identity/dedup/condition/SIP-fallback
behavior, covered zero, locked/crossed/stale/partial/current spread, independent
field reasons, a within-horizon out-of-order quote revision and too-late quote
rejection, a post-ack quote with pre-coverage SIP that cannot backfill duration,
the exact configured bound constants plus structurally identical overflow
containment at small private proof bounds, gap clearing, rank
removal, connection-loss clearing, and fresh resubscription. It snapshots
aggregate evaluation/rank/readiness before and after and requires exact
equality. Limitation: the reviewed fixture supports
qualifying originals but not complete provider lifecycle reconstruction or
live entitlement.

`P-C9-PRESSURE` is one controlled mixed-frame/runtime trace. It crosses the
degraded and aggregate-only thresholds, proves T/Q is rejected only after
element classification, proves a later aggregate and control fact in the same
raw frame still apply, reaches zero provider T/Q membership, preserves
aggregate watermark/ranking/readiness, holds recovery during unhealthy/dwell
samples, then restores current symbols one-by-one in rank order with new
warming state. It separately proves a partial/ambiguous subscribe followed by
successful cleanup and a failed cleanup that remains
`provider_membership_unknown`, never false zero. It injects a T/Q-local
accounting failure into `aggregate_only`, while a mixed-frame accounting
ambiguity follows the existing global ingress-integrity path. It validates all
pressure/drop/command/accounting identities and bounds before interpreting
results. A delayed/duplicate healthy pressure-result trace is fenced and cannot
advance engine-time recovery dwell. Missing and more-than-two-second unhealthy
results exercise engine-owned expiry, fresh command issuance, first-miss
degradation, and second-miss aggregate-only containment; their later results
cannot restore `normal`. Limitation: the proof validates policy
behavior at conservative
provisional gates; it does not locate a current-host saturation frontier or
claim provider capacity, market-hours behavior, or an SLA.

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

### C9-S2-H current-host heap-gate correction — 2026-08-13

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

Reopen S1 if a normalized shape cannot carry the stated identity/quality facts,
paired acknowledgement cannot prove per-symbol/channel coverage, the weighted
window admits a boundary ambiguity, or any T/Q fact can change aggregate
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
