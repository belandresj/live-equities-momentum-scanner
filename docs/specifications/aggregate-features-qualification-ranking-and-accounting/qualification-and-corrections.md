# Aggregate features, qualification, ranking, and accounting — qualification and corrections

**Parent contract:** [Aggregate features, qualification, ranking, and accounting](../aggregate-features-qualification-ranking-and-accounting.md)

**Normative responsibility:** Exact same-session aggregate-tape gate,
pre-ranking qualification, provisional proof, correction revocation, finalized
session latch, and bounded qualification state

**Controlling requirements:** `C3-QUAL-01`, `C3-QUAL-02`; `PG-RANK-02`,
`PG-RANK-03`, `PG-FEATURE-05`, `DTE-CLOCK-05`, `DTE-WINDOW-01`,
`DTE-WINDOW-04`, `DTE-AGG-02`–`DTE-AGG-04`, `DTE-MERGE-02`,
`DTE-COMMIT-03`, `DTE-TIMER-01`, `DTE-CHECKPOINT-01`,
`DTE-CHECKPOINT-02`, `LIFE-LIVE-02`, and `LIFE-T23`

**Allocated slices:** `C3-S3`; `C3-R1` owns the approved corrective amendments

**Document dependencies:** Parent; aggregate-feature mathematics for shared
finite arithmetic only; Component 2 S2 canonical aggregate and S4 contributor
boundaries

**Approval state:** Inherits the parent contract approval; not independently approved

**Delivery state:** See the authoritative parent delivery-state ledger; this
detail does not copy mutable acceptance status

## 5–8. Resolved questions, approved reconnaissance, and reuse assessment

Gate arithmetic uses full-precision finite `float64` with no display rounding.
Qualification proof identity is the whole-second window end `P`; one bit means
the exact gate passed at `[P-60s,P)`. Multiple mutable proofs coexist. A proof
becomes permanent only when engine time is strictly later than `P+H`, because
the bar ending at `P` remains revisable at equality.

Scoped v2 inspection used predecessor commit
`5f92a151dd850002578a33a81ad90dea096c63b6`:

| Exact source | SHA-256 | Finding and limitation | Decision | Coupling removed / required proof |
| --- | --- | --- | --- | --- |
| `internal/scanner/state.go`: `gateAccumulator`, `passesMinimumAggregateTapeGate`, `calculateAggregateGateFacts` | `a1742749f3be63e60e0d60dc8064a12cb8931307326174a2e90af5aeb40b6ae9` | Correctly models 60 exact slots, edge gaps, ATS availability, dollar volume, concentration, and inclusive thresholds. It is embedded in a competing evaluator/owner. | Adapt arithmetic only | Consume Component 2 canonical state in the one contributor and prove every boundary. |
| `state.go`: `markQualificationProofsDirty`, `revalidateDirtyQualificationProofs`, qualification portion of `foldState` | state hash above | A bounded set of proof ends supports targeted correction revocation and strict finalization. Old maintenance/watermark ownership is rejected. | Adapt representation | Timer/correction execution stays inside Component 2; prove multi-proof and horizon behavior. |
| `internal/scanner/aggregate_product_state_test.go`: `TestMinimumAggregateTapeGateBoundariesAndSparseTime`, `TestAggregateProductStateQualificationLatchCorrectionFinalizationAndDateReset` | `2fbab060f95ef4edc24bc7411e71b7f3df80a00fab819cee08363ad34c6957c8` | Strong exact-threshold, leading/interior/trailing-gap, ATS, revocation, quiet-latch, equality, and date-reset evidence. It uses a configurable test horizon and old evaluator. | Behavior evidence | Fix production semantics at `H=16m`; run through the Component 2 transition. |
| `internal/scanner/state_test.go`: `TestInclusiveRevisionAndStrictFinalizationBoundaries` | `fc3474a37f116d3300bf9f493c819d3fae9688d040b8fa66d701c9e73152ecfd` | Confirms inclusive correction and strict fold behavior already used by Component 2. | Behavior evidence | Shared boundary is not reimplemented; Component 3 proves its dependent proof fold. |

**Proposed v2 implementation whitelist:** no whole-file port. The gate
accumulator arithmetic and proof-end bit-set technique may be adapted from
`state.go`; only the three named tests above may supply fixtures/expected
cases. V2 `Evaluate`, maintenance, recovery, owner, ranking, and state types are
not whitelisted.

## 9–10. Semantic contract and required behavior

### Inputs, state, and construction guarantees

| Item | Exact meaning | Bound / owner |
| --- | --- | --- |
| Gate input | Canonical accepted aggregate identities in `[T-60s,T)`, trusted Last, and correction/finalization events. | Read-only projection from Component 2; ATS provenance retained but both approved mappings use the same formula. |
| Mutable proof ring | Passing window ends plus dirty bits capable of reevaluation. | At most 961 whole-second ends per symbol for inclusive `H=16m`; engine-owned. |
| Final latch | Boolean plus earliest finalized proof end for review/restart equivalence. | One per symbol, valid only for the binding/session; clears only with a new engine. |
| Output | `unresolved`, `not_yet_passed`, `provisional`, or `finalized`, with current proof count. | Private bounded state; ranking consumes only provisional/finalized pass. |

`unresolved` also carries the bounded cause origin defined by `C3-PROJ-01`:
`bootstrap_origin`, `post_bootstrap_gap`, or `local_invalid`. This is stored in
the existing qualification consequence/result, not another population map.

Types and visibility prevent a caller-created proof, cross-session latch,
second evaluator, or correction callback. Runtime validation rejects nonfinite
arithmetic, duplicate/out-of-range proof ends, impossible dirty references,
and cardinality overflow.

### `C3-QUAL-01` — exact gate at committed `T`

For every whole-second proof end `P` in `[S,T]`, evaluate the exact interval
`[max(S,P-60s),P)` and its five-second suffix. Slots before `S` are outside the
session. A missing same-session slot is absence only when canonical presence or
installed exact coverage proves it; otherwise the proof is unresolved. Slots are never filled from another
session. The implementation may advance a mathematically equivalent sliding
accumulator, but qualification completeness through `T` means every endpoint
through `T` whose result could differ has been accounted for. Define exactly:

```text
N    = number of present canonical aggregate identities
G    = longest consecutive absent-slot run, including leading/trailing edges
A60  = sum(Volume / AverageTradeSize) over present 60-second contributors
A5   = sum(Volume / AverageTradeSize) over present five-second contributors
DV60 = sum(Volume * VWAP) over present contributors
C    = max(one-second Volume) / sum(Volume)
```

`latest_price` is the close of the latest present identity in the 60-second
window. `A60`/`A5` are unavailable when any contributor to that respective
window has nonpositive ATS. Total volume must be finite positive for `C`; all
intermediates must be finite. ATS provenance remains observable and live/REST
values are never required to be equal.

The gate passes only when every predicate is true, inclusively at its stated
boundary:

```text
latest_price >= 0.25; N >= 45; G <= 3;
A60 available and >= 1000; DV60 >= 250000;
A5 available and >= 100; C <= 0.50
```

Invalid or unavailable input fails the gate. Qualification is evaluated for a
trusted rankable mark before ranking; Activity, display ranges, T/Q, freshness,
and row capacity are not gate inputs.

### `C3-QUAL-02` — provisional proof, correction, and final latch

A passing evaluation at end `P` installs a provisional proof for `P`; an
existing proof is idempotent. A correction to identity `[s,s+1)` can affect
only proof ends `P` satisfying `s < P <= s+60s`; every retained affected proof
is marked dirty and reevaluated from canonical state before publication. A
failing reevaluation removes only that proof. The symbol remains qualified if
another provisional proof or the final latch exists.

On an ordered aggregate or timer transition, if any still-valid proof satisfies
`engine_time > P+16m`, the owner sets the session-final latch and may discard
all mutable proof bits. Equality remains mutable. Later quiet tape or rejected
too-late corrections cannot revoke the final latch. The latch lasts through
`E=20:00 ET` and never crosses a binding/session. Correction at unchanged
committed `T` can change provisional qualification and must trigger a distinct
publication when output changes. A symbol's qualification trace is complete
through `T` only when exact coverage/fence evidence supports every same-session
identity needed by all proof ends through `T` and the evaluator has accounted
for those ends. Sparse or quiet periods do not waive this trace; batching is
allowed only when proved equivalent to explicit per-second evaluation.

`not_yet_passed` is closed only when the complete qualification trace from `S`
through `T` has exact present/proven-absent coverage and no conflict. Unknown
coverage remains `unresolved`; allocating a conflict/absence bitmap cannot
close the trace.

### Consequential trust boundaries

| Boundary | Accept into success only when | Reject or contain | Dangerous false success |
| --- | --- | --- | --- |
| Canonical gate set | Every identity belongs to the bound symbol/session and has Component 2's final precedence. | Local invalid input fails that proof; global canonical ambiguity suppresses. | Counting a revision as a second active second. |
| Proof finalization | Proof still passes and engine time is strictly greater than `P+H`. | Equality stays provisional; dirty proof is revalidated first. | Permanently latching a proof while its last bar is still correctable. |
| Restored/projected state | Binding, `T0`, final latch, proof ends, dirty/fold state, and supporting canonical tail are coherent. | Component 7 must reject mixed/incomplete projection. | Restart revives a revoked proof or loses a valid final latch. |

## 11–14. Failure, observability, boundedness, and evidenced edges

A gate failure is an ordinary symbol result, not an error. Representable state
contradiction or proof-ring overflow is symbol-local invalid only if exact
qualification can be conservatively withheld; an ownership/accounting
contradiction that could falsely preserve qualification uses Component 2's
global integrity path. Retained diagnostics are fixed scalar counts for proof
install/revalidate/revoke/finalize and maximum ring occupancy; no proof history,
symbol labels, or arbitrary strings are retained.

The proof ring is bounded by one possible window end per second across the
inclusive 16-minute horizon: 961 proof bits and 961 dirty bits per symbol,
192,200,000 bits (24,025,000 raw bytes) at the 100,000-symbol ceiling, plus
fixed per-symbol metadata. Allocation is lazy. Gate calculation reads Component 2's already-bounded canonical tail
and cutoff-bearing derived gate summaries while catch-up is unresolved. At
most 57,601 gate summaries are retained per symbol; they are discarded after
candidate commit makes them unnecessary or the final latch closes. Admission
time alone never prunes evidence later than stalled committed `T`. These are
arithmetic summaries, not copied canonical raw history.

Evidenced edges are exact threshold adjacency, pre-`S`/leading/interior/trailing gaps,
zero total volume, ATS zero in A60 versus A5, multiple proofs, correction of a
sole proof, correction leaving another proof, quiet tape, horizon equality,
strict finalization, and new-session reset. No provider-specific case is added.

Component 7 must later reproduce the final latch, retained proof/dirty set,
fold boundary, and sufficient canonical 60-second/correction-tail state at one
coherent `T0`. This specifies restart equivalence, not checkpoint contents or
encoding.

## 15. Primary proof allocation

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result / limitation | Slice |
| --- | --- | --- | --- | --- |
| `C3-QUAL-01` | Gate boundary matrix plus `C3-R1` coverage amendment | Every formula/threshold is exact and only exact interval evidence turns missing seconds into inactivity; counters sparse bars or bitmap allocation resolving an unknown gate. | Exact facts/pass/unresolved result for adjacent boundaries, provenance, unknown, and proven absence. Provider ATS parity and Component 6 fact production remain unproved. | `C3-S3`; `C3-R1` |
| `C3-QUAL-02` | Correction/finalization plus stalled-`T` retention trace | Corrections revoke only affected proofs; strict-old proof latches; forward summaries survive stalled `T`; counters early latch, quiet revocation, or admission-time pruning before catch-up. | Proof/latch/result after every transition; 961 proof/dirty and 57,601 summary bounds. Component 7 codec and live arrival distribution remain unproved. | `C3-S3`; `C3-R1` |
