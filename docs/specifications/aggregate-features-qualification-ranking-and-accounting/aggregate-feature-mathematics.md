# Aggregate features, qualification, ranking, and accounting — aggregate feature mathematics

**Parent contract:** [Aggregate features, qualification, ranking, and accounting](../aggregate-features-qualification-ranking-and-accounting.md)

**Normative responsibility:** Day %, From 4AM %, HOD drawdown, session and
rolling range positions, Activity, field-local aggregate-history availability,
correction equivalence, and bounded retained mathematics

**Controlling requirements:** `C3-FEAT-01`, `C3-FEAT-02`, `C3-ACT-01`,
`C3-ACT-02`; `PG-FEATURE-01`, `PG-FEATURE-02`, `PG-FEATURE-05`,
`PG-AVAIL-01`, `PG-AVAIL-02`, `PG-REPLAY-01`, `DTE-CLOCK-05`,
`DTE-WINDOW-01`, `DTE-WINDOW-04`, `DTE-AGG-02`–`DTE-AGG-04`,
`DTE-MERGE-02`, `DTE-MERGE-04`, `DTE-COMMIT-03`, `DTE-TIMER-01`,
`DTE-REPLAY-03`, `DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-02`,
`LIFE-LIVE-02`, and `LIFE-T23`

**Allocated slices:** `C3-S1` owns `C3-FEAT-01`/`02`; `C3-S2` owns
`C3-ACT-01`/`02`; `C3-R1` and `C3-R2` own their approved corrective
amendments

**Document dependencies:** Parent; Component 1 binding; Component 2 S2
canonical aggregates and S4 contributor/publication boundary

**Approval state:** Inherits the parent contract approval; not independently approved

**Delivery state:** See the authoritative parent delivery-state ledger; this
detail does not copy mutable acceptance status

## 5–8. Resolved questions, approved reconnaissance, and reuse assessment

The approved reconnaissance settled representation, block eligibility,
availability, and bounded-state questions without changing Phase 1. Internal
market values and derived values use `float64`; every input and intermediate
operation must remain finite. Product formulas are evaluated at full internal
precision. Display rounding belongs to Component 10 and never feeds comparison,
percentiles, qualification, or a stored correction summary.

Only these v2 sources were opened for this concern, at predecessor commit
`5f92a151dd850002578a33a81ad90dea096c63b6`:

| Exact source | SHA-256 | Finding and limitation | Decision | Coupling removed / required proof |
| --- | --- | --- | --- | --- |
| `internal/scanner/state.go`: `finalizedRange`, `foldState`, `gainerCandidate`, `populateAggregateProductFields`, `rangePosition` | `a1742749f3be63e60e0d60dc8064a12cb8931307326174a2e90af5aeb40b6ae9` | Monotone extrema and prefix summaries survive a long corrected path. The implementation mixes owner, recovery, ranking, and feature state and reports return fractions rather than contracted percentages. | Adapt technique only | Move state into the named Component 3 substate; use Component 2 correction/fold events; prove exact formulas and equivalence. |
| `internal/scanner/aggregate_product_state_test.go`: `TestAggregateProductStateFrom4AMAndCorrectionAwareRanges`, `TestAggregateProductStateWarmingSparseZeroRangesJSONAndBounds` | `2fbab060f95ef4edc24bc7411e71b7f3df80a00fab819cee08363ad34c6957c8` | Credible half-open, earliest-open, correction, zero-width, and long-path cases. JSON/snapshot assertions are outside scope; expected returns omit the factor of 100 and elapsed-30m/60m warm-up conflicts with Phase 1's session-intersected windows. | Behavior evidence | Retain only named market-state cases and replace expected formula scale/availability with this contract. |
| `internal/scanner/state_test.go`: `TestRollingSparseSlotsAndExactFeatures`, `TestFiniteAggregatesCannotProduceNonfiniteDerivedFacts`, `TestCorrectionAwareHODAndFinalizedPrefix`, `TestLongPathEquivalenceAfterPrefixFolding` | `fc3474a37f116d3300bf9f493c819d3fae9688d040b8fa66d701c9e73152ecfd` | Finite accumulation, missing-slot absence, prefix-HOD, and differential long-path checks are useful. Several tests target superseded momentum features. | Behavior evidence | Adapt only finite/correction/prefix cases; reject unrelated pressing/volatility logic. |
| `internal/scanner/state.go`: `activityComponents`, `recomputeActivityBlock`, `passesActivityReferenceEligibility`, `populateActivity`, `empiricalPercentile` | state hash above | Establishes session-aligned 30-second blocks, ATS invalidity, an inclusive 100-transaction reference floor, inclusive ties, and correction insert/remove/replace. It incorrectly retains all-session references, uses a five-second maintenance delay, and permits 1,920 references per symbol. | Adapt arithmetic and eligibility; reject retention/delay | Apply the rolling-60-minute interval and exact bounds below; prove target/reference and correction equivalence. |
| `internal/scanner/activity_product_state_test.go`: all eleven focused Activity tests | `9c06b2e48fdb9dc60723764afc84c61d8164a83723cd7762fd3139212bbec960` | Strong arithmetic, 10-reference, tie, correction, independence, and field-local bound evidence. `AllSessionReference` and five-second-delay expectations conflict with Phase 1. | Behavior evidence, with conflicting cases rejected | Reuse only named arithmetic/tie/correction/TQ-independence inputs; replace all-session/delay expectations. |

The discovery search incidentally named `phase2a_acceptance_remediation_test.go`;
it was not read as evidence, is outside the approved recovery-ownership scope,
and is not whitelisted.

**Proposed v2 implementation whitelist:** no whole-file or production-code
port. `state.go` may inform only the finite-add/multiply, prefix scalar, and
monotone-deque techniques. The named aggregate-product, Activity, and
`state_test.go` cases above may be copied as input/expected-behavior evidence
after conforming expected values to this contract. No v2 owner, recovery,
checkpoint, readiness, API, snapshot, or provider type is allowed.

## 9–10. Semantic contract and required behavior

### Inputs, outputs, and owned state

| Item | Exact meaning | Bound / owner |
| --- | --- | --- |
| Aggregate input | Component 2 canonical aggregate values and ATS provenance for identities with `window_start < T`, plus insert/revision/withdraw/finalization and localized coverage/conflict facts. | Read-only transition projection; no adapter payload. |
| Mark and prior close | Latest trusted accepted same-session close before `T`; Component 1 finite positive adjusted prior close. | Values copied for evaluation; owners remain Components 1/2. |
| Price/range state | Earliest trustworthy session aggregate open, finalized session high/low scalars, and two timestamped finalized monotone deques (high and low) shared by the 30m/60m queries. | Lazily allocated per printed symbol inside the engine. At most `2*3600 = 7,200` deque points; tails are Component 2 state, not duplicated. |
| Activity state | Session-aligned eligible 30-second reference summaries, mutable-block state, and a separate Activity-only folded-target projection retaining identity-positioned `(Volume/ATS, High, Low, validity)` contributions. | At most 1,920 aligned summaries, 33 mutable block identifiers, 119 evaluation references, 1,920 lazily allocated folded-target blocks, and 57,600 folded target contributions per printed symbol; no canonical raw bar copy. |
| Feature result | Optional finite numeric value plus one field status/reason for each aggregate field. | Latest engine-owned result copied into the private publication contribution. |

Construction prevents a second clock, raw-history owner, unrounded-versus-
rounded dual value, T/Q input, independently mutable result, or per-symbol
goroutine. Runtime validation still rejects nonfinite arithmetic, invalid or
contradictory coverage, impossible extrema, and retention overflow.

### `C3-FEAT-01` — exact price/range formulas and availability

At committed `T`, use full-precision `float64` and compute exactly:

```text
Day %                = 100 * (Last / adjusted_prior_close - 1)
From 4AM %           = 100 * (Last / first_session_aggregate_open - 1)
HOD drawdown %       = 100 * (Last / session_high - 1)
range position       = 100 * (Last - low) / (high - low)
```

There is no clamp. A valid HOD result lies in `[-100,0]`; a valid range result
lies in `[0,100]`, including genuine zero and 100. A result outside its
mathematical domain is `invalid`, not rounded into range. Zero-width range is
`unavailable/zero_width`; missing required value is unavailable rather than
numeric zero.

Day % needs only a trusted mark and valid prior close. From 4AM needs the Open
of the earliest accepted aggregate in `[S,T)` and trustworthy coverage proving
that an earlier unknown aggregate cannot exist. HOD and session range need
trustworthy history over `[S,T)`; rolling ranges need trustworthy history over
`[max(S,T-30m),T)` or `[max(S,T-60m),T)`. Proven absent seconds contribute
nothing. An unresolved interval or localized historical conflict clears only
dependent fields; it does not clear Last/Day %. The rolling intervals are
intersected with `[S,T)`, so elapsed session duration alone does not create a
30m/60m warm-up rule. `warming` means the exact required intersected history is
not yet complete. Component 8 alone may later convert a current value to
`stale` under an approved freshness policy.

Coverage is exact per second, not inferred from aggregate acceptance. The
required intervals are `[S,first_print_start)` for From 4AM, `[S,T)` for HOD
and session range, and `[max(S,T-width),T)` for each rolling range. Every
second must be canonically present or explicitly proven absent and a localized
conflict overrides both; allocating any bitmap proves nothing. Unknown
coverage is `unavailable/history_incomplete`. A missing/invalid prior close
makes only Day % `unavailable/prior_close_unavailable` and the symbol
unrankable; From 4AM, HOD, ranges, and Activity remain independent.

### `C3-FEAT-02` — exact correction and bounded price/range state

Every accepted insert, revision, withdrawal, and timer/finalization event
recomputes the affected mutable contribution before Component 2 can discard
its full aggregate. Equivalent canonical aggregate sets and coverage facts at
the same `T` must produce bit-identical feature values/statuses regardless of
permitted delivery order. A localized conflict invalidates only summaries whose
required interval contains it. The 16-minute mutable tail, applied-`T` scalars,
7,200 rolling deque points, and two cutoff-bearing session-extrema chains of at
most 57,600 values each are the complete retained representation. A post-`T`
record may enter the forward chains but every as-of query excludes it until a
candidate at or after its second commits. Session-long canonical raw history is
forbidden.

### `C3-ACT-01` — exact Activity statistic

The target is `[T-30s,T)`. Missing seconds are absence. The target is valid
only when its coverage is trustworthy, at least one aggregate exists, every
contributing ATS is finite positive, and all sums/extrema remain finite:

```text
transactions  = sum(Volume / AverageTradeSize)
expansion_bps = 10,000 * ln(max(High) / min(Low))
```

Reference blocks are session-aligned
`[S+30n,S+30(n+1))`. A reference is included only when its entire interval is
inside `[max(S,T-60m),T-30s)`, its coverage is trustworthy, it contains at
least one aggregate, all contributing ATS values are positive, arithmetic is
finite, and `transactions >= 100`. This inclusive floor is delegated product
detail supported by the scoped v2 tests; it is not the qualification gate.
There are therefore at most 119 eligible reference positions at one `T`.

Require at least 10 eligible references. For each component:

```text
percentile(x) = 100 * count(reference <= x) / reference_count
Activity      = sqrt(transaction_percentile * expansion_percentile)
```

Ties are inclusive; a flat but nonempty block has zero expansion and is valid.
The result must be finite in `[0,100]`. Insufficient references are `warming`;
unknown coverage is `unavailable/history_incomplete`; nonpositive ATS or
invalid arithmetic is `invalid/invalid_input`. None affects ranking membership.

Trustworthy exact target coverage with zero aggregates is
`unavailable/no_aggregate_in_target`, including when an older session print
exists. It is neither `before_first_print` nor `history_incomplete`. Every
target and considered aligned reference block independently requires exact
coverage; an exactly covered empty reference block is simply ineligible.

### `C3-ACT-02` — Activity correction and bounded equivalence

The owner recomputes only aligned blocks touched by an accepted correction and
expires references older than the committed-boundary rolling floor on aggregate or timer
transitions. A correction may insert, replace, or remove an eligible reference.
At equality with the Component 2 horizon, affected aggregate evidence remains
mutable; folding occurs only strictly beyond it. Differential recomputation
from the canonical accepted set is the required equivalence oracle. A global
Activity-retention invariant failure makes Activity invalid/unavailable with a
fixed reason and enters Component 2 integrity containment only when state
ownership/cardinality itself is ambiguous; it never changes rank.

While committed `T` is stalled, forward Activity summaries are retained rather
than deleted merely because they are later than the current evaluation window.
The retained ceiling is 1,920 aligned block summaries per symbol; evaluation
still consumes at most 119 prior reference positions.

Every strictly folded aggregate also installs one Activity-only contribution
at its exact session-second identity in a lazily allocated session-aligned
target block. A target block has 30 transaction, High, and Low slots plus
presence and invalidity masks. Evaluation reconstructs every whole-second
`[T-30s,T)` by visiting its exact seconds in order and combining folded
contributions with nonoverlapping Component 2 tail records. A successful
candidate apply may discard folded target seconds strictly before `T-30s`;
while committed `T` stalls, later target blocks remain. Component 2-permitted
tail revisions are resolved before fold. A historical-conflict withdrawal
removes the folded identity and its conflict coverage prevents false success.

At most 1,920 target blocks and 57,600 folded target contributions exist per
printed symbol. Three 30-element `float64` arrays plus two 30-bit masks charge
728 raw payload bytes per allocated block: at most 1,397,760 bytes per printed
symbol and 139,776,000,000 bytes at the 100,000-symbol ceiling, before
container overhead. These are explicit finiteness bounds, not Component 8
capacity acceptance. The projection is not canonical raw history: it retains
no Open, Close, Volume, VWAP, ATS value/provenance, source authority, or
delivery metadata.

### Consequential trust boundaries

| Boundary | Accept into success only when | Reject or contain | Dangerous false success |
| --- | --- | --- | --- |
| Component 2 projection | Identity is canonical for the binding, interval is before/equal to the evaluation boundary as applicable, and correction/fold order is explicit. | Wrong binding/global ambiguity suppresses; localized conflict invalidates dependent fields. | A historical conflict silently contributing to a current range. |
| Coverage fact | Exact symbol/interval/fence evidence proves known coverage; empty slots remain absence. | Unknown/fenced interval is field-local unavailable unless globally ambiguous. | Treating missing history as a genuine narrow range or first open. |
| Derived numeric | Every operand and intermediate is finite and domain-valid. | Field becomes invalid; never emit `NaN`, infinity, or a fabricated zero. | Overflow passing a qualification/display check as current. |

## 11–14. Failure, accounting, boundedness, and evidenced edges

Feature invalidity is field-local unless a canonical/accounting contradiction
cannot be attributed. Reasons are a closed enum: `before_first_print`,
`history_incomplete`, `rolling_warmup`, `reference_warmup`, `zero_width`,
`historical_conflict`, `invalid_input`, `state_bound_exceeded`,
`prior_close_unavailable`, `no_aggregate_in_target`, and the later
Component 8 `stale` overlay. Retained observability is fixed scalar counts by
status/reason and maxima for deque/reference/dirty occupancy; symbols are not
metric labels. These dimensions do not balance `PG-OBS-01`. At the Component 1
100,000-symbol ceiling, the retained per-symbol ceilings are 7,200 rolling
deque points, two cutoff-bearing session-extrema chains of 57,600 points each,
1,920 Activity block summaries, 33 mutable Activity-block identifiers, 119
ephemeral Activity reference positions, 1,920 folded-target blocks, and 57,600
folded target contributions. Folded-target allocation is lazy by printed
symbol and block.
These are finiteness bounds,
not Component 8 capacity acceptance.

Evidenced edges are: exact `T` exclusion; genuine range 0/100; zero width;
earlier-open insert/revision; mutable extrema replacement; sparse seconds;
ATS zero; finite-input overflow; Activity empty/flat/tied/10-reference cases;
reference insert/remove/replace; horizon equality/strict fold; and long-path
equivalence. No provider-specific edge was added.

Component 7 must later project enough coherent state at exactly `T0` to
reproduce the four requirements above after catch-up: all finalized scalars and
deque/reference summaries, mutable dirty/fold metadata, the folded-target
contributions required for exact catch-up target reconstruction, field-local
coverage and invalidity facts, and their binding/`T0` identity. This is a
reproduction obligation, not a checkpoint field list, encoding, or schema
decision.

## 15. Primary proof allocation

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result / limitation | Slice |
| --- | --- | --- | --- | --- |
| `C3-FEAT-01` | Price/range boundary table plus `C3-R1` coverage/invalid-prior amendment | Exact formulas and independent availability; counters sparse acceptance or bitmap allocation proving history, invalid prior erasing non-Day truth, factor-of-100 drift, `T` inclusion, or zero-width zero. | Exact unknown/proven-absence, boundary, conflict, invalid-prior, and invalid statuses. Component 6 fact production, API formatting, and freshness remain unproved. | `C3-S1`; `C3-R1` |
| `C3-FEAT-02` | Correction/permutation plus stalled-`T` differential trace | Equivalent as-of state stays exact within 7,200 rolling points and two 57,600-point cutoff-bearing chains; counters post-`T` contamination, lost pre-`T` mark, stale extrema, or raw history. | Fold post-`T` evidence while old-`T` fields remain identical and later catch-up evidence remains. Does not prove latency. | `C3-S1`; `C3-R1` |
| `C3-ACT-01` | Target/reference table plus exact-coverage and non-aligned folded-target amendments | Exact statistics and statuses; counters unknown intervals as absence, covered-empty target as `before_first_print`, all-session leakage, target reuse, or a folded real target aggregate becoming `no_aggregate_in_target`. | Includes exact `no_aggregate_in_target` and full-history-oracle equality after folded catch-up; Component 6 facts and provider ATS parity remain unproved. | `C3-S2`; `C3-R1`; `C3-R2` |
| `C3-ACT-02` | Activity correction, stalled-`T`, long-path, and folded non-aligned catch-up trace | Equivalence within 1,920 retained summaries, 33 mutable IDs, 119 evaluation references, 1,920 folded-target blocks, and 57,600 folded target contributions; counters stalled-`T` expiry or aligned-only compaction destroying catch-up evidence. | Canonical differential output, old-`T` stability, strict raw-tail fold, later non-aligned target equality, permitted correction/withdrawal containment, retained-state ceilings, and no rank mutation. Checkpoint codec/capacity remain unproved. | `C3-S2`; `C3-R1`; `C3-R2` |
