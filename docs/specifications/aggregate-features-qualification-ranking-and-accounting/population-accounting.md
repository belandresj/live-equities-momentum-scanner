# Aggregate features, qualification, ranking, and accounting — population accounting

**Parent contract:** [Aggregate features, qualification, ranking, and accounting](../aggregate-features-qualification-ranking-and-accounting.md)

**Normative responsibility:** Exact mutually exclusive symbol population
partition, overlapping qualification/feature dimensions, local/global
containment, and accounting-integrity consequence

**Controlling requirements:** `C3-POP-01`, `C3-POP-02`; `PG-RANK-01`,
`PG-RANK-02`, `PG-RANK-05`, `PG-OBS-01`, `PG-OBS-03`,
`DTE-MERGE-04`, `DTE-RECOVERY-04`, `DTE-RECOVERY-05`,
`DTE-COMMIT-02`–`DTE-COMMIT-04`, `DTE-REJECT-02`,
`LIFE-HYDRATE-06`, `LIFE-LIVE-03`, `LIFE-SUPPRESS-01`,
`LIFE-PUBLISH-03`, and `LIFE-T12`

**Allocated slices:** `C3-S4`; `C3-R1` owns the approved corrective amendments

**Document dependencies:** Parent; Component 1 binding; qualification and
corrections; Component 2 S4 accounting/publication boundary

**Approval state:** Inherits the parent contract approval; not independently approved

**Delivery state:** See the authoritative parent delivery-state ledger; this
detail does not copy mutable acceptance status

## 5–8. Resolved questions, approved reconnaissance, and reuse assessment

Primary classification is a closed precedence table over immutable binding
facts, Component 2 canonical mark/context, and later Component 6 exact
no-print/failure/fence facts. `invalid_mark` means resolved, attributable local
invalid mark evidence; it is not a bucket for arbitrary rejected input. An
unresolved failure capable of hiding a later mark takes `unknown` precedence.

Scoped v2 inspection at commit
`5f92a151dd850002578a33a81ad90dea096c63b6` found no reusable complete
population evaluator:

| Exact source | SHA-256 | Finding and limitation | Decision | Coupling removed / required proof |
| --- | --- | --- | --- | --- |
| `internal/scanner/state.go`: `gainerCandidate`, `evaluateGainers`, `MarkedMissingPriorCloseCount`, `RetentionStats` | `a1742749f3be63e60e0d60dc8064a12cb8931307326174a2e90af5aeb40b6ae9` | Filters missing prior/mark/below-price symbols but does not assign every bound symbol to the `PG-OBS-01` partition; no-print/failure and degraded covered/unresolved accounting are absent. | Reject as accounting implementation | Build the closed table below in the one evaluator; ranking behavior is assessed separately. |
| `internal/scanner/aggregate_product_state_test.go`: focused rank/range tests | `2fbab060f95ef4edc24bc7411e71b7f3df80a00fab819cee08363ad34c6957c8` | Demonstrates local filtering, not a balancing identity or failure/no-print transitions. | Evidence limitation only | No accounting fixture is whitelisted from this file. |
| `internal/scanner/metrics.go`: enum-indexed counters | `b198addd39a77343dcd5df49557036b328a868ebfb0e4096bcb3b9ed54bc9976` | Fixed-cardinality arrays avoid symbol labels, but the reason set and owner are transport-specific. | Adapt technique only | Define Component 3 categories and prove reconciliation; do not port metrics owner/backend. |

No scoped v2 test established `completed_empty`, earlier-mark-plus-empty,
failure/fence, or later-first-print population transitions. Their authority is
Phase 1 and their production evidence owner remains Component 6.

**Proposed v2 implementation whitelist:** only the fixed enum-indexed scalar
counter technique from `metrics.go`; no v2 state, accounting, recovery,
readiness, snapshot, or test fixture is whitelisted.

## 9–10. Semantic contract and required behavior

### Inputs, outputs, and state

| Item | Exact meaning | Bound / owner |
| --- | --- | --- |
| Bound population | Component 1 exact ordered universe and per-symbol prior-close status. | Immutable for the engine lifetime. |
| Mark context | Latest Component 2 canonical aggregate with `window_start<T`, plus exact local historical-conflict/coverage status. | Read-only at evaluation; no copied history. |
| Local invalid-mark evidence | Component 2 exact bound-symbol/session/window aggregate disposition rejected for structural market values after attribution is unambiguous. | Component 3 retains at most one fixed reason/identity boundary per printed/invalid symbol; a later trusted accepted mark supersedes it. |
| Later coverage fact | Component 6 typed per-symbol `no_print_through(T)` or unknown failure/fence consequence, each naming binding, interval, generation, and reconciled fence. | Absent until Component 6; Component 3 defines consequences, not producer/schema mechanics. |
| Primary result | Seven counters forming the two identities below plus `covered_population`. | One fixed struct per publication; no per-symbol retained classification table is required. |
| Overlapping dimensions | Qualification state counts and status/reason counts for each Component 3 feature. | Fixed arrays/scalars; future T/Q coverage is added by Component 9 and never balances these identities. |

Construction uses one enumeration function with one return category per bound
symbol. No caller supplies counters and no post-ranking second pass may adjust
them to balance. Runtime validation checks every count for overflow, both
identities, covered/unresolved derivation, and row membership against category.

### `C3-POP-01` — complete primary partition and precedence

Every publication in which population is in scope must satisfy exactly:

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

Classification proceeds once per bound symbol:

1. Missing, invalid, nonpositive, or nonfinite Component 1 prior-close status
   is `invalid_or_missing_prior_close`; no mark fact can move it into the second
   identity.
2. With valid prior close and an eligible canonical mark before `T`, an
   unresolved post-mark gap/fence capable of hiding a newer mark is
   `unknown_due_failure_or_fence`. Otherwise trustworthy close `<0.25` is
   `trusted_below_price_mark`, and trustworthy close `>=0.25` is
   `trusted_rankable_mark`.
3. With valid prior close and no eligible canonical mark, any unresolved
   failure/fence or incomplete bootstrap evidence is
   `unknown_due_failure_or_fence`.
4. If no unresolved failure remains, an exact-symbol/current-session aggregate
   with unambiguous identity but structurally invalid market values that
   prevented canonical acceptance is `invalid_mark`. Malformed or
   unattributable input is diagnostic evidence only and cannot fabricate this
   bin.
5. Only exact reconciled coverage proving no accepted or invalid same-session
   print through `T` is `no_print_through_T`. A valid empty subinterval alone is
   insufficient.

An earlier accepted mark plus a later proved empty interval remains a mark;
its price/coverage decides rankable versus below-price. A later first accepted
aggregate immediately replaces no-print/unknown/invalid local classification
with ordinary mark evaluation. Historical conflict invalidates only dependent
history when a current live mark remains independently trustworthy.

For degraded-bootstrap disclosure:

```text
covered_population = universe_total - unknown_due_failure_or_fence
unresolved_population = unknown_due_failure_or_fence
```

All primary category counts are exposed privately to the later API mapping;
`covered_population` is derived, not an eighth balancing bin.

### `C3-POP-02` — overlapping dimensions and integrity containment

Qualification counts (`unresolved`, `not_yet_passed`, `provisional`,
`finalized`) apply to trusted rankable marks, overlap the primary partition,
and reconcile exactly:

```text
trusted_rankable_mark
  = qualification_unresolved
  + qualification_not_yet_passed
  + qualification_provisional
  + qualification_finalized
```

Each of the seven aggregate features reports fixed status, reason, and
status/reason-pair counts independently. For every feature,
`sum(status)=universe_total`, `reason_none=current`, and
`sum(non-none reasons)=universe_total-current`; unknown statuses/reasons or an
illegal pair fail validation rather than defaulting into a bucket. Activity, range, and
future T/Q coverage never change a primary bin. Exact zero-qualified rows is a
valid outcome when primary accounting and qualification evidence are complete.

A symbol-attributable invalid/failure/conflict remains local and conservatively
classified. Contradictory global binding/canonical evidence, a symbol omitted
or double-counted, counter overflow, or either failed identity takes Component
2's `accounting_integrity` global suppression path before a current/degraded
publication can escape. Counters are never repaired after the fact.

### Consequential trust boundaries

| Boundary | Accept into success only when | Reject or contain | Dangerous false success |
| --- | --- | --- | --- |
| Component 1 prior status | Exact immutable binding member and explicit valid/missing/invalid status. | Binding ambiguity is global; symbol-local missing remains its bin. | A missing prior close silently entering `valid_prior_close`. |
| Component 6 no-print/failure fact | Exact binding/symbol/interval/generation and reconciled fence support the consequence through `T`. | Stale/fenced/partial evidence is unknown, never no-print. | `completed_empty` for one interval being promoted to session no-print. |
| Counter result | One category returned for every bound symbol and both checked sums hold. | Mismatch suppresses before publication. | Dropping unmarked symbols while still labeling ranking exact. |

## 11–14. Failure, observability, boundedness, and evidenced edges

Primary counts use nonwrapping integers sized for the Component 1 universe
bound; construction rejects a population that cannot be counted. One
evaluation retains only the fixed counters, at most 20 selected rows, and
fixed reason arrays. It does not retain a second universe-sized category map,
symbol diagnostic list, or arbitrary error text. The local invalid-mark state
is at most one fixed record per bound symbol and therefore at most 100,000;
allocation is lazy.

Evidenced edges derive from Phase 1 invariants: missing/invalid prior close;
trusted mark at/below the USD 0.25 boundary; future/noneligible mark exclusion;
proved no-print; partial empty; earlier mark plus later empty; local invalid
evidence; failure/fence; first print; localized historical conflict; and global
identity failure. V2 contributes no provider/recovery evidence for these cases.

## 15. Primary proof allocation

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result / limitation | Slice |
| --- | --- | --- | --- | --- |
| `C3-POP-01` | Complete partition/transition table over every primary input combination | Every symbol enters exactly one bin and empty/failure/first-print precedence is exact; counters no-print from partial empty or mark omission. | Both identities and covered/unresolved counts after every transition. Component 6 production fact validation remains unproved until its component. | `C3-S4` |
| `C3-POP-02` | Accounting-integrity/overlap plus `C3-R1` reason-pair scenario | Qualification/feature changes never rebalance primary counts; unresolved participates explicitly; every feature status/reason pair reconciles; counters omitted unresolved, contradictory reasons, silent repair, or Activity-driven exclusion. | Exact local/global result and no current publication on any mismatch. Public metrics and T/Q accounting remain unproved. | `C3-S4`; `C3-R1` |
