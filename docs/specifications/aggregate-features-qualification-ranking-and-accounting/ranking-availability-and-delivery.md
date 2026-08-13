# Aggregate features, qualification, ranking, and accounting — ranking, availability, and delivery

**Parent contract:** [Aggregate features, qualification, ranking, and accounting](../aggregate-features-qualification-ranking-and-accounting.md)

**Normative responsibility:** Rankability, exact ordering/top-20 selection,
qualified/degraded/unavailable projections, independent row fields, atomic
single-evaluator integration, contract-wide proof ledger, delivery plan,
implementation discretion, acceptance checklist, and drift audit

**Controlling requirements:** `C3-RANK-01`, `C3-PROJ-01`, `C3-EVAL-01`;
`PG-RANK-01`–`PG-RANK-05`, `PG-AVAIL-01`–`PG-AVAIL-03`,
`PG-OPS-02`, `PG-REPLAY-01`, `PG-OBS-03`, `PG-TAQ-03`,
`ARCH-OWN-01`, `ARCH-OWN-03`, `ARCH-OWN-04`, `DTE-CLOCK-05`,
`DTE-MERGE-05`, `DTE-RECOVERY-05`, `DTE-COMMIT-02`–`DTE-COMMIT-04`,
`DTE-TQ-02`, `DTE-TQ-03`, `DTE-REPLAY-01`, `DTE-REPLAY-03`,
`LIFE-MODEL-01`–`LIFE-MODEL-03`, `LIFE-HYDRATE-03`,
`LIFE-HYDRATE-04`, `LIFE-HYDRATE-06`, `LIFE-LIVE-01`,
`LIFE-LIVE-02`, `LIFE-RECOVER-05`, `LIFE-TQ-01`, `LIFE-TQ-03`,
`LIFE-REPLAY-02`, `LIFE-PUBLISH-02`, `LIFE-PUBLISH-03`,
`LIFE-T11`, `LIFE-T12`, `LIFE-T15`, `LIFE-T19`, `LIFE-T20`, and
`LIFE-T23`

**Allocated slices:** `C3-S4` owns this document's original requirements;
owner-approved corrective slice `C3-R1` amends the affected proofs and atomic
boundary; owner-approved `C3-R2` amends the Activity retained-state/proof
premise; this document owns the authoritative delivery plan

**Document dependencies:** Parent; aggregate feature mathematics;
qualification and corrections; population accounting; Component 2 S3 central
commit and S4 contributor/publication boundaries

**Approval state:** Inherits the parent contract approval; not independently approved

**Delivery state:** See the authoritative parent delivery-state ledger; this
detail does not copy mutable acceptance status

## 5–8. Resolved questions, approved reconnaissance, and reuse assessment

Day return is a full-precision finite `float64` computed by `C3-FEAT-01`.
Equality for ranking is exact equality of that unrounded computed value; the
only tie-breaker is exact case-sensitive symbol ascending. Selection uses a
20-element worst-first heap plus final exact sort, giving `O(U log 20)` time
and `O(20)` selection memory while still counting every passer.

The implemented Component 2 seam is sufficient. `Engine.transition` has one
explicit fixed-order contributor insertion point after canonical disposition
and before `finishTransitionLocked`; `commitTargetLocked` is the sole `T`
assignment path; `completePublicationDecisionLocked` is the sole visibility
path. Component 3 must extend those private functions with staged evaluation,
not add a public interface or second pass.

Scoped predecessor inspection at commit
`5f92a151dd850002578a33a81ad90dea096c63b6` found:

| Exact source | SHA-256 | Finding and limitation | Decision | Coupling removed / required proof |
| --- | --- | --- | --- | --- |
| `internal/scanner/state.go`: `evaluateGainers`, `gainerCandidate` | `a1742749f3be63e60e0d60dc8064a12cb8931307326174a2e90af5aeb40b6ae9` | Correctly filters qualification before Day-return/symbol sorting. It scans a competing state owner, returns all passers, reports fractional returns, and owns watermark/evaluation separately. | Behavior evidence only | Reimplement inside Component 2 with contracted percentages, bounded top 20, and atomic commit. |
| `internal/scanner/aggregate_product_state_test.go`: `TestAggregateProductStateFilterBeforeRankBackfillsTop20AndUsesExactTies` | `2fbab060f95ef4edc24bc7411e71b7f3df80a00fab819cee08363ad34c6957c8` | Strong filter-before-sort and exact-symbol tie inputs. Despite its name it expects 22 returned rows, so it does not prove truncation. | Adapt fixture; reject expected cardinality | Add explicit 0/1/19/20/21+ row and total-passer assertions. |
| `internal/scanner/state_test.go`: `TestTopGainersPureRankingStaleMarksAndCorrectionAwareHOD` | `fc3474a37f116d3300bf9f493c819d3fae9688d040b8fa66d701c9e73152ecfd` | Shows pure Day-return order, below-price/missing-prior filtering, mark age, and correction reevaluation. It accepts arbitrarily old marks and uses fraction-scale fields. | Behavior evidence with stale/scale expectations rejected | Component 8 supplies freshness; Component 3 enforces nonnegative age and exact formula scale. |
| `internal/scanner/activity_product_state_test.go`: `TestPhase2BActivityProductStateIndependentOfTAQSelection`, `TestPhase2BActivityProductStateGlobalBoundFailsClosedWithoutChangingRank` | `9c06b2e48fdb9dc60723764afc84c61d8164a83723cd7762fd3139212bbec960` | Useful evidence that T/Q and display-field failure do not alter aggregate rank. | Behavior evidence | Prove independence through the new immutable contribution; do not port T/Q/snapshot types. |

No scoped v2 implementation/test distinguished exact qualified,
`degraded_bootstrap`, unavailable, stale, or suppressed projections with the
required covered/unresolved population and no-T/Q-promotion semantics. No v2
evaluator invalidation scheme can be reused without old orchestration.

**Proposed v2 implementation whitelist:** no production code. Only the named
filter/tie/rankability and field-independence test inputs may be adapted.
`state.go` evaluation/sort/watermark code, `types.go`, snapshots, readiness,
recovery, T/Q, owner/orchestration, API, and UI are rejected.

## 9–10. Semantic contract and required behavior

### Inputs, output, and engine-owned state

| Item | Exact meaning | Bound / owner |
| --- | --- | --- |
| Evaluation projection | Immutable binding population/prior facts; canonical mark/context; feature/qualification state; candidate/current `T`; exact coverage/fence/lifecycle facts. | Copied/read-only inside the one transition. |
| Staged result | Population counters, qualification completeness, ranking mode/reason, total passers or known-rankable count, and up to 20 private rows. | Ephemeral until the same central commit accepts it; otherwise discarded. |
| Current result | Latest applied Component 3 result at exactly committed `T`. | One bounded engine-owned value; no result history or second population copy. |
| Private row | Symbol, trusted Last/mark age, unrounded Day %, and optional aggregate fields with independent status/reason. | At most 20; deep-copied into Component 2 publication. Not Component 10 schema. |

Construction has one statically named `aggregateEvaluatorState` inside the
engine, one pure staging function, one owner apply point, and no queue,
goroutine, callback, registry, clock, watermark setter, publisher, or writable
alias. Runtime validation checks row uniqueness/order/cardinality, population
identities, finite values, mode predicates, and staged `T` identity before
apply.

### `C3-RANK-01` — rankability, filtering, ordering, and top 20

A symbol can enter qualified ranking only when population accounting classifies
it `trusted_rankable_mark`, its mark has nonnegative age at `T`, and its
qualification state is provisional or finalized. A display feature being
warming/unavailable/invalid never affects membership.

Filter qualification before selection. Define `a precedes b` iff:

```text
a.day_percent > b.day_percent
or (a.day_percent == b.day_percent and a.symbol < b.symbol)
```

No epsilon, display rounding, Activity/range/TQ value, mark age, source, or
other key participates. Iterate the complete population once, increment the
exact total qualified-passer count, retain only the best 20 in a worst-first
heap, then sort those rows by `precedes` and assign ranks 1..N. Zero through 19
passers produce exactly that many rows; 20 produces 20; more produces exactly
20 without duplicates or holes.

### `C3-PROJ-01` — exact projection modes and independent availability

The private ranking mode is one of:

| Mode | Exact acceptance evidence and result |
| --- | --- |
| `qualified_current` | Current binding, aggregate transport/watermark and ingress fence; reconciled population with `unknown_due_failure_or_fence=0`; qualification evaluated completely through `T`. Rows use `C3-RANK-01`; zero passers is an exact empty table. |
| `degraded_bootstrap` | Current binding, transport/watermark and fence; exact population counters; at least one trusted rankable mark; every unresolved population/qualification cause is `bootstrap_origin`. Rows are raw known-rankable Day-%/symbol top 20 without qualification filtering. Expose every primary count, `covered_population`, `unresolved_population`, known-rankable count, and bounded origin counts; state that omitted symbols may qualify or outrank. Never promote T/Q. |
| `degraded_current` | Live lifecycle with current binding, aggregate transport/watermark and ingress fence; at least one trusted rankable mark; remaining uncertainty is symbol-local. The uncertainty may retain `bootstrap_origin` after startup or arise as `post_bootstrap_gap`/`local_invalid`; origin does not extend the startup-only mode. Rows are raw known-rankable Day-%/symbol top 20 without qualification filtering. Symbols whose current mark could be displaced by unresolved evidence are excluded; trusted-mark symbols with unresolved qualification may appear but are never called qualified. Expose exact population, qualification, and origin counts. Never promote T/Q. |
| `unavailable` | No committed `T`, zero trusted marks for degraded output, incomplete global binding/fence, or another unmet current/degraded predicate. No contracted rows. |
| `stale` | A later Component 8 currentness fact says an otherwise coherent prior result is no longer current. Component 3 chooses no age threshold and creates no current/degraded claim. |
| `suppressed` | Component 2 global integrity/lifecycle path controls the publication; Component 3 contributes no contracted rows. |

Neither partial mode is used after exact evidence merely because zero
symbols qualify. Row fields carry `warming/current/unavailable/invalid` from
Components 3/9 and the later Component 8 `stale` overlay independently. A
field absence is never numeric zero. T/Q state cannot alter a mode, row, rank,
aggregate field, `T`, or backend-readiness predicate; only
`qualified_current` rows may later create T/Q desired membership.
`degraded_current` is a current causal claim but an explicitly non-qualified
ranking claim; a genuine transport, binding, clock, fence, or canonical-global
failure remains unavailable rather than being relabeled symbol-local.

Uncertainty origin is the closed enum `bootstrap_origin`,
`post_bootstrap_gap`, or `local_invalid`. Bootstrap origin means startup
hydration or explicit partial replay initialization before the first ordinary
live entry; it may persist after transition to live. A later live/recovery gap
is `post_bootstrap_gap` and invalid/bound evidence is `local_invalid`.
Active recovery projects stale/unavailable under lifecycle authority; a
terminal ranking-affecting later gap suppresses; a transiently live later gap
is unavailable. Neither may be relabeled degraded bootstrap.

### `C3-EVAL-01` — one atomic evaluator and path equivalence

The fixed owner transition order is:

```text
Component 2 validate/classify/canonical mutation and bounded forward update
  -> stage complete Component 3 candidate with one-symbol scratch
  -> validate exact candidate T, field pairs, counters, rows, and bounds
  -> validate run support, contributor predicates, fence, and accounting
  -> on failure apply no committed-T/result/evaluator-current state
  -> on success atomically apply committed T, per-symbol results, evaluator current
  -> build/validate publication, including evaluation.at == watermark
  -> one Component 2 cell swap
```

Evaluation is pure with respect to the supplied projection until owner apply.
A failed commit cannot leave candidate-T feature, qualification, accounting,
ranking, or availability state visible. A same-`T` accepted correction or
coverage/status change reevaluates current `T` and may replace publication with
a new publication ID. A future bar relative to current `T` can update mutable
support state but cannot enter the current evaluation.

A candidate-`T` mismatch is accounting integrity failure, not a silent
discard. Ordinary lack of future run support leaves the previous current
publication unchanged. A deterministic locked second apply pass may recompute
the already validated per-symbol result with one-symbol scratch; it consumes no
new evidence and has no rejecting branch.

Aggregate insert/revision/withdrawal, ordered timer, exact coverage/no-print/
failure/fence change, lifecycle currentness change, and later coherent
checkpoint install are evaluator causes. T/Q event/health, checkpoint-write
result, API/UI activity, and receipt/generated time are not ranking causes.
Dirty-set optimization is discretionary only if it is observationally
equivalent to complete evaluation; the initial design performs bounded
incremental feature updates and a complete `O(U log 20)` population scan.

Fresh hydration, checkpoint catch-up, ordinary live, reconciled recovery, and
replay feed this same source-blind evaluator. Equal binding, canonical state,
coverage facts, and `T` produce identical Component 3 output. Components 4–8
still own artifact scheduling, successful run-support evidence, hydration fact
production, checkpoint format, freshness, and production capacity.

Component 7 must later reproduce at one coherent `T0` all Component 3 state
needed for uninterrupted equivalence: the feature/qualification sufficient
state routed to their owning details, classification-support facts, and the
binding/committed boundary from which the same evaluator regenerates
population counters, mode, and rows after catch-up. The current selected rows
need not be checkpoint authority. This is a reproduction obligation, not a
checkpoint field, schema, or encoding decision.

### Consequential trust boundaries

| Boundary | Accept into success only when | Reject or contain | Dangerous false success |
| --- | --- | --- | --- |
| Staged candidate | Names the exact binding and candidate/current `T`; population identities, mode predicates, order, uniqueness, and bounds validate. | Discard on failed commit; contradiction uses accounting/global integrity as applicable. | Candidate ranking mutates state even though `T` did not commit. |
| Mark/rank input | Category is trusted rankable, prior close valid, eligible mark before `T`, finite unrounded Day %. | Local nonrankable/invalid/unknown stays out; global ambiguity suppresses. | Future or rounded mark displacing a valid leader. |
| Immutable contribution | Built after all transition stages and deep-copied through Component 2's sole builder. | Builder/validation failure uses Component 2 unavailable sentinel. | Reader sees new rows with old counters or old qualification. |

## 11–14. Failure behavior, observability, bounds, and evidenced edges

Local mark/history/feature invalidity affects only its category/field. Failed
ranking/accounting construction, duplicate/out-of-order rows, counter overflow,
staged-T mismatch, or cross-symbol/canonical ambiguity is global and cannot
publish stale success. Successful exact empty ranking remains distinct from
unavailable and degraded. Component 3 owns no retry or wait; later fact
producers terminate under their contracts and ordinary evaluation continues.

The complete evaluator retains: fixed scalar population/overlap/status
counters, one current result, 20 rows, and a 20-element ephemeral heap. Its
population scan is bounded by Component 1's 100,000-symbol universe maximum.
Feature and qualification bounds are in their owning details. No candidate
slice of size `U`, sort of all passers, publication history, raw events,
high-cardinality labels, or arbitrary errors are retained. Evaluation duration
may be observed as fixed scalar current/max values; Component 8 alone sets an
operational threshold.

Evidenced edges are exact equal return/symbol tie, high-return nonpasser,
missing prior, below-price mark, negative-age exclusion, 0/1/19/20/21+ passers,
duplicate prevention, exact empty, zero-mark degraded rejection, raw degraded
ordering, field independence, same-`T` correction, failed candidate commit,
and source/path equivalence. V2 supplies only the scoped filter/tie/field-
independence inputs; Phase 1 and construction provide the rest.

## 15. Primary proof allocation and complete requirement ledger

The feature, Activity, qualification, and population details own their proof
rows. The table below is a navigational coverage ledger, not a second proof
authority; each ID has one owning detail, one original owning slice, and only
the explicitly approved corrective amendments shown:

| Requirement | Primary proof / authoritative home | Slice |
| --- | --- | --- |
| `C3-FEAT-01` | Full-precision price/range boundary table — [feature mathematics](aggregate-feature-mathematics.md#15-primary-proof-allocation) | `C3-S1` |
| `C3-FEAT-02` | Correction/permutation long-path differential trace — feature mathematics | `C3-S1` |
| `C3-ACT-01` | Activity target/reference/percentile table plus `C3-R1`/`C3-R2` amendments — feature mathematics | `C3-S2`; `C3-R1`; `C3-R2` |
| `C3-ACT-02` | Activity correction/long-path differential trace plus `C3-R1`/`C3-R2` amendments — feature mathematics | `C3-S2`; `C3-R1`; `C3-R2` |
| `C3-QUAL-01` | Exact qualification gate matrix — [qualification](qualification-and-corrections.md#15-primary-proof-allocation) | `C3-S3` |
| `C3-QUAL-02` | Provisional/revoked/multiproof/finalized trace — qualification | `C3-S3` |
| `C3-POP-01` | Complete partition/transition table — [population accounting](population-accounting.md#15-primary-proof-allocation) | `C3-S4` |
| `C3-POP-02` | Accounting-integrity/overlap scenario — population accounting | `C3-S4` |
| `C3-RANK-01` | Exact qualification-filter/order/top-20 table — authoritative detailed row below | `C3-S4` |
| `C3-PROJ-01` | Qualified/degraded/unavailable/field-independence table — authoritative detailed row below | `C3-S4` |
| `C3-EVAL-01` | Single-evaluator atomic staging and path-equivalence trace — authoritative detailed row below | `C3-S4` |

| Requirement | Claim and dangerous counterexample | Observable result / intentional limitation |
| --- | --- | --- |
| `C3-RANK-01` | Only qualified rankable symbols enter exact Day-%/symbol top 20; counters high raw-return nonpasser, rounded tie, all-passer return, or hidden secondary key. | Exact order, ranks, unique row count, and total passers for boundary populations. Does not prove trading edge or capacity latency. |
| `C3-PROJ-01` | Every mode needs exact evidence; `degraded_bootstrap` is confined to startup hydration, while live-lifecycle symbol-local uncertainty uses `degraded_current` without rewriting its closed origin. Counter global uncertainty relabeled local, empty partial, T/Q promotion, or non-Day failure suppressing Last/Day %. | Exact mode/reason/rows/origin-counts/intent eligibility across startup bootstrap, live retained-bootstrap, post-bootstrap, and local-invalid evidence. Freshness thresholds and public labels remain Components 8/10. |
| `C3-EVAL-01` | Candidate applies iff exact identity and every predicate validate before the one apply; publication evaluation equals watermark. Counters mismatch mutating `T`, feature/qualification result, evaluator current, or publication; second publisher; mixed accounting; or watermark mismatch. | Invalid/mismatched candidate leaves applied state unchanged and production containment suppresses; normal publication rejects mismatched `evaluation.at`. Successful live/replay support production remains Components 4–6. |

## 16. Sequential implementation-slice plan

The original approved plan remains four accepted slices. Combining Activity with price/range would
mix independent block/percentile and extrema proof families; combining
qualification with ranking would make a permanent latch review depend on the
complete publication change. Four is therefore the smallest reviewable
sequence under the repository's proof-boundary rule. The final-review findings
require one owner-approved corrective slice because coverage, retained state,
and candidate atomicity cannot be changed independently without an intervening
false-success path.

| Slice | Coherent outcome | Requirement IDs / primary proofs | Dependencies and allowed boundary | Approved v2 use | Acceptance record | Explicitly deferred |
| --- | --- | --- | --- | --- | --- | --- |
| `C3-S1` | Exact correction-aware price/range feature state and independent availability exist inside the engine, without a ranking publication. | `C3-FEAT-01`, `C3-FEAT-02`; their two feature proofs. | Finally approved Component 2; `internal/engine` named feature state/helpers/tests only. | Named feature techniques/tests in feature Section 5–6. | Formula/availability table, differential trace, 7,200-point bound, ownership walk, deviations, S2 validity. | Activity, qualification, population/ranking projection. |
| `C3-S2` | Exact 04:00 session-to-date Activity with bounded correction-aware references extends S1 state. | `C3-ACT-01`, `C3-ACT-02`; their two Activity proofs. | Accepted S1; same package and contributor state, no alternate evaluator. | Named Activity arithmetic/tests only; five-second-delay expectations rejected. | Target/reference table, differential trace, 1,920/33 bounds, rank-independence result, deviations, S3 validity. | Qualification and population/ranking projection. |
| `C3-S3` | Exact aggregate gate owns revocable provisional proofs and the strictly finalized session latch. | `C3-QUAL-01`, `C3-QUAL-02`; gate and latch proofs. | Accepted S1/S2 and Component 2 correction/finalization seam; `internal/engine` qualification state/tests only. | Named gate/proof techniques/tests. | Boundary matrix, multiproof correction trace, 961 proof/dirty bounds, checkpoint-reproduction obligation, deviations, S4 validity. | Population, exact/degraded ranking, immutable rows. |
| `C3-S4` | One atomic evaluator classifies the full population, selects exact qualified/degraded rows, attaches independent fields, and contributes one immutable result. | `C3-POP-01`, `C3-POP-02`, `C3-RANK-01`, `C3-PROJ-01`, `C3-EVAL-01`; five owning proofs. | Accepted S1–S3; extend only Component 2 central commit/contributor/private publication in `internal/engine`. | Named ranking/independence fixtures and fixed counter technique only. | Complete partition and selection tables, mode/field table, failed/success/same-T atomic trace, construction and success/failure walkthrough, all prior proof reruns, limitations, final-review request. | Components 4–11 fact production, checkpoint schema, operations thresholds, T/Q implementation, public API/UI. |
| `C3-R1` | Exact coverage/origin, committed-`T`-safe sufficient state, truthful feature/qualification accounting, and prevalidated atomic candidate publication close the final-review findings. | Amendments to `C3-FEAT-01`/`02`, `C3-ACT-01`/`02`, `C3-QUAL-01`/`02`, `C3-POP-01`/`02`, `C3-PROJ-01`, `C3-EVAL-01`; all named false-success cases. | Accepted S1–S4; only this contract set, parent ledger, and `internal/engine`; no Component 6 producer. | No new predecessor source or fixture. | Complete C3 ledger, coverage/origin/empty-target/stalled-`T`/atomicity/reason-accounting proofs, assignment/ownership inspection, focused re-review request. | Final acceptance/spec-map update, Components 4–11, provider/recovery producer, checkpoint schema, cutover. |
| `C3-R2` | Every whole-second non-aligned Activity target remains exactly reconstructible after its aggregates fold while committed `T` stalls. | Amendments to `C3-ACT-01`/`02`; one distinguishing full-history-oracle proof. | Accepted S1–S4 and implemented C3-R1; only this contract set, parent ledger, and `internal/engine`. | No predecessor source or fixture. | Old-`T` stability, strict raw-tail fold, later non-aligned oracle equality, folded-evidence consumption, correction/withdrawal containment, revised bounds, cumulative verification, focused re-review. | Final acceptance/spec-map update, Components 4–11, checkpoint schema/codec, capacity policy, cutover. |

Every requirement and primary-proof owner remains allocated once; `C3-R1` and
`C3-R2` amend those owning proofs and do not create a second proof authority. Each slice leaves the build
passing, adds no unused alternate path, and passes the parent ledger's delegated
gate before the next assignment. S3 requires a narrow independent review of proof
revocation/finalization before S4. S4 requires a narrow independent review of
commit/evaluator/publication atomicity and accounting integrity before final
component review. S1/S2 need no separate independent reviewer unless their
primary proof or implementation evidence exposes an ownership/cardinality
defect. Mutable slice state and accepted evidence are recorded only in the
parent delivery-state ledger.

## 17. Implementation discretion

Implementation may choose private file/helper/type names, bitset versus bounded
map for proof/dirty ends, monotone-deque storage layout, 20-element heap
mechanics, scratch-array reuse, full versus correctly equivalent dirty
population scan, and error wrapping outside retained state. It may use tighter
bounds only if semantics and the differential proofs remain unchanged.

Non-discretionary choices are the full-precision formulas, Activity block/
eligibility/session-to-date interval, inclusive percentile, `H=16m`, strict proof
finalization, category precedence, exact comparator, filter-before-truncate,
mode predicates, finite bounds, and staged central-commit integration.

**Prohibited changes**

- Another owner, canonical map, evaluator, clock, `T`, queue, publisher, reader
  path, callback/registry/plugin, or mutable alias.
- T/Q-, Activity-, range-, freshness-, pressure-, or presentation-dependent
  qualification/order; rounded ranking values or another tie-breaker.
- Session-long canonical raw feature history, more than 1,920 derived Activity
  block summaries, 1,920 Activity evaluation-reference capacity, 1,920 folded-target
  blocks, or 57,600 folded target contributions, more than 20 retained
  candidate rows, or counter repair after mismatch.
- Provider decoding/mapping, hydration/recovery fact production, successful
  live/replay support claims, checkpoint contents/codec/storage, production
  thresholds/readiness, public schema/API/UI, database, service, actor, or
  speculative extension point.
- Any v2 source/test beyond the exact whitelist; any v1, Git-history,
  credential, or live-provider access.

**Stop/escalation conditions**

- The Component 2 tail/fold transition hides a correction before dependent
  state can be updated, or candidate evaluation cannot be staged/applied in
  the sole commit/publication transition.
- `H=16m` cannot preserve exact feature/qualification correction equivalence;
  a required bound is exceeded; or a primary differential proof fails.
- Component 6 needs a no-print/failure meaning incompatible with the population
  precedence table, or Component 7 cannot reproduce the stated sufficient
  state at coherent `T0` without changing Component 3 semantics.
- Provider-specific behavior, public/checkpoint schema, production freshness/
  capacity policy, unapproved predecessor evidence, or a document-map change
  is required.

## 18. Completed-contract acceptance checklist

- [x] The parent map lists the complete five-document contract and one exclusive responsibility for each.
- [x] Sections 1–19, every Phase 1 relationship, eleven `C3-*` requirements, eleven primary proofs, and four slices are routed exactly once.
- [x] Dependencies are explicit and acyclic; all relative links resolve.
- [x] Owner boundary/document-map/reconnaissance approval and Component 2 final completion are recorded.
- [x] V2 inspection stayed within the routed source/test scope; incidental excluded search output was not opened or used.
- [x] Exact v2 paths, commit/hash provenance, findings, limitations, decisions, coupling removal, proofs, and proposed whitelist are recorded.
- [x] Inputs, outputs, engine-owned state, construction guarantees, runtime validation, failure outcomes, and finite cardinality are exact.
- [x] Formula scale, Activity eligibility/window, correction/finalization, category precedence, ranking, availability, and atomic integration are settled.
- [x] Primary accounting identities and overlapping qualification/feature/future-TQ dimensions are distinct.
- [x] Every nontrivial edge has Phase 1, mathematical, Component 2, scoped v2, or explicit owner-boundary evidence; no provider edge was invented.
- [x] Each requirement has one proof naming claim, dangerous counterexample, observable distinction, participating path, and limitation.
- [x] Every requirement/proof belongs to exactly one slice; later slices extend rather than replace prior ownership.
- [x] Component 7 reproduction obligations are stated without choosing checkpoint fields/schema.
- [x] No production code, tests, provider integration, runtime infrastructure, or public API schema changed.

## 19. Drift audit

| Question | Result | Evidence |
| --- | --- | --- |
| New product rule? | No | Delegated Activity eligibility/numeric/state decisions implement exact Phase 1 formulas and scoped evidence; no trader-facing field/rank rule was added. |
| Another owner, watermark, evaluator, clock, queue, or publisher? | No | One named engine substate, pure staging function, central commit, and Component 2 cell. |
| T/Q affects aggregate qualification/ranking/readiness? | No | T/Q is absent from all predicates and cannot promote degraded output. |
| Changed session/window/correction/committed-time semantics? | No | `[start,end)`, absence, source provenance, inclusive `H=16m`, strict fold, and single `T` are preserved. |
| Local invalidity globalized? | No | Field/symbol invalidity is local; only binding/canonical/accounting ambiguity suppresses. |
| Unevidenced provider edge? | No | No provider behavior was specified; ATS mappings/provenance are consumed from Phase 1. |
| Duplicated checkpoint/API/readiness ownership? | No | Only reproduction/input obligations are stated; schemas, thresholds, and mappings remain later. |
| Unbounded or unnecessary machinery? | No | 7,200 rolling deque points, two 57,600-point cutoff chains, 1,920 retained Activity summaries/evaluation-reference capacity/33 mutable IDs plus 1,920 folded-target blocks/57,600 Activity-only contributions, 57,601 qualification summaries plus 961 proof/dirty ends, fixed counters, one current result, and heap 20; no framework/service/database. |
| V2 drove architecture or conflicting behavior survived? | No | The owner-selected session-to-date Activity baseline is implemented within the existing owner; five-second delay, fractional returns, all-passer output, and old owner/watermark/readiness remain rejected. |

Any later substantive `yes` or whitelist/interface expansion requires owner
review. The owner approved this complete contract, predecessor reuse decisions,
primary-proof plan, and four-slice sequence on 2026-08-05. Current slice state,
accepted evidence, and the next delegated gate are recorded only in the parent
delivery-state ledger.
