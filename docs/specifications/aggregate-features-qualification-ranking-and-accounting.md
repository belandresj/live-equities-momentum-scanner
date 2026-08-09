# Aggregate features, qualification, ranking, and accounting

**Status:** Finally accepted 2026-08-06 after `C3-S1`–`C3-S4`, corrective
cycles `C3-R1`/`C3-R2`, their required focused reviews, and the mandatory
separate read-only final component review. No Component 3 acceptance item
remains open.

**Owner boundary approval:** approved 2026-08-05 in the owning Codex task

**Owner contract/reuse/test/slice-plan approval:** approved 2026-08-05 in the
owning Codex task; the owner-approved 2026-08-06 process revision delegates the
remaining objective slice and final-component gates unless a manual-decision
condition occurs

**Owner corrective-contract approval:** approved 2026-08-06 in the owning
Codex task for the exact-coverage, uncertainty-origin, covered-empty Activity,
qualification-accounting, committed-`T` retention, and atomic-candidate
corrections recorded as `C3-R1`

**Owner `C3-R2` corrective-contract approval:** approved 2026-08-06 in the
owning Codex task for the exact non-aligned Activity target reconstruction,
folded-target retained-state bound, checkpoint-reproduction obligation, and
corrective-proof amendments recorded as `C3-R2`

**Advancement mode:** `delegated`

**Controlling Phase 1 requirements:** `PG-RANK-01`, `PG-RANK-02`,
`PG-RANK-03`, `PG-RANK-04`, `PG-RANK-05`, `PG-FEATURE-01`,
`PG-FEATURE-02`, `PG-FEATURE-05`, `PG-AVAIL-01`, `PG-AVAIL-02`,
`PG-AVAIL-03`, `PG-OPS-01`, `PG-OPS-02`, `PG-REPLAY-01`, `PG-OBS-01`,
`PG-OBS-03`, `PG-TAQ-03`, `ARCH-OWN-01`, `ARCH-OWN-03`,
`ARCH-OWN-04`, `DTE-CLOCK-05`, `DTE-WINDOW-01`, `DTE-WINDOW-04`,
`DTE-AGG-02`, `DTE-AGG-03`, `DTE-AGG-04`, `DTE-MERGE-02`,
`DTE-MERGE-04`, `DTE-MERGE-05`, `DTE-RECOVERY-04`,
`DTE-RECOVERY-05`, `DTE-COMMIT-02`, `DTE-COMMIT-03`,
`DTE-COMMIT-04`, `DTE-REJECT-02`, `DTE-TIMER-01`, `DTE-TQ-02`,
`DTE-TQ-03`, `DTE-REPLAY-01`, `DTE-REPLAY-03`,
`DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-02`, `LIFE-MODEL-01`,
`LIFE-MODEL-02`, `LIFE-MODEL-03`, `LIFE-HYDRATE-03`,
`LIFE-HYDRATE-04`, `LIFE-HYDRATE-06`, `LIFE-LIVE-01`,
`LIFE-LIVE-02`, `LIFE-LIVE-03`, `LIFE-RECOVER-05`, `LIFE-TQ-01`,
`LIFE-TQ-03`, `LIFE-REPLAY-02`, `LIFE-SUPPRESS-01`,
`LIFE-PUBLISH-02`, `LIFE-PUBLISH-03`, `LIFE-T11`, `LIFE-T12`,
`LIFE-T15`, `LIFE-T19`, `LIFE-T20`, and `LIFE-T23`

**Approved dependencies:** the finally approved Component 1 immutable
[`reference.Binding`](reference-data-and-session-binding.md#9-detailed-semantic-inputs-outputs-and-owned-state)
and the approved complete Component 2
[`ScannerStateEngine` contract](scanner-state-engine-and-canonical-state.md).
Component 2 is finally approved after its separate read-only final component
review. Component 3 is finally accepted after the complete implementation,
corrective, verification, and independent-review sequence recorded below.

## Contract document map

The documents below are one modular Component 3 contract and one authority.
This parent is the mandatory entry point; no subordinate document is
independently approved.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Component outcome, Phase 1 trace, single ownership boundary, scope/non-scope, cross-cutting time/correction/atomicity invariants, routing, and approvals | Sections 1–4 and 7; all parent-routed Phase 1 constraints | Every Component 3 task | Components 1 and 2 |
| [Aggregate feature mathematics](aggregate-features-qualification-ranking-and-accounting/aggregate-feature-mathematics.md) | Price/range and Activity formulas, availability, correction equivalence, retained mathematics, reconnaissance, trust, edges, and proofs | Sections 5–15 for `C3-FEAT-01`, `C3-FEAT-02`, `C3-ACT-01`, `C3-ACT-02`; four proofs; `C3-S1`/`C3-S2` obligations | Formula, field availability, correction, retention, and feature proofs | Parent; Components 1 and 2 |
| [Qualification and corrections](aggregate-features-qualification-ranking-and-accounting/qualification-and-corrections.md) | Exact tape gate, provisional proof, correction revocation, finalized latch, bounded state, reconnaissance, trust, edges, and proofs | Sections 5–15 for `C3-QUAL-01`, `C3-QUAL-02`; two proofs; `C3-S3` obligations | Gate/latch/proof-finalization work | Parent; feature mathematics for finite arithmetic; Component 2 |
| [Population accounting](aggregate-features-qualification-ranking-and-accounting/population-accounting.md) | Exact primary partition/precedence, overlapping dimensions, local/global containment, reconnaissance, trust, edges, and proofs | Sections 5–15 for `C3-POP-01`, `C3-POP-02`; two proofs; `C3-S4` obligations | Population/accounting integrity work | Parent; qualification; Components 1 and 2 |
| [Ranking, availability, and delivery](aggregate-features-qualification-ranking-and-accounting/ranking-availability-and-delivery.md) | Rankability/order/top 20, projection modes, field independence, atomic evaluator integration, reconnaissance, trust, complete proof ledger, slices, discretion, checklist, and drift | Sections 5–19 for `C3-RANK-01`, `C3-PROJ-01`, `C3-EVAL-01`; three proofs; authoritative `C3-S1`–`C3-S4` plus corrective `C3-R1`/`C3-R2` plans | Ranking/projection/integration, assignment preparation, and complete-manifest review | Parent; all three preceding details; Component 2 |

**Layout:** Approved modular contract. The concern boundaries are semantic and
proof-oriented, not based on document size. Formula/correction work can be
reviewed without loading degraded-ranking or accounting detail; accounting and
the final ranking projection remain separate authorities even though their
implementation is likely to land atomically in the same final slice.

**Routing rule:** A requirement, evidence question, proof, normative decision,
or proposed slice not unambiguously routed by this map stops for a parent-map
correction. Parent summaries are navigational and do not replace the owning
detail document. An ID may appear in another detail's header as a controlling
constraint or declared dependency; exclusive Component 3 normative ownership
is the single home named by this map and the relationship trace below.

**Contract-wide coverage and acceptance:** The exact eleven-requirement/proof
ledger, four-slice plan, completed-contract checklist, and drift audit are in
[ranking, availability, and delivery](aggregate-features-qualification-ranking-and-accounting/ranking-availability-and-delivery.md#15-primary-proof-allocation-and-complete-requirement-ledger).
Detailed behavior, reuse decisions, primary proofs, and the four-slice plan are
owner-approved. Current implementation state and the next gate are recorded
only in the authoritative delivery-state ledger below.

## Authoritative delivery-state ledger

This table is the only mutable Component 3 delivery record. Subordinate specs
own behavior, proof allocation, and slice boundaries but do not copy acceptance
state.

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `accepted` | Boundary and complete contract/reuse/test/four-slice plan owner-approved; all four slices and both corrective cycles accepted; mandatory final review clean | 2026-08-06 | Complete |
| `C3-S1` | `accepted` | `C3-FEAT-01`/`02`, Component 2 regressions, race, repository tests, vet, whitespace, ownership/cardinality/scope passed; independent review not required | 2026-08-06 | Complete |
| `C3-S2` | `accepted_after_owner_revision` | Owner changed the Activity baseline from prior rolling hour to all eligible completed blocks since 04:00; the 13-hour discriminator crosses both 09:30 and 16:00 without reset, checkpoint/replay and ordinary/race verification pass, and focused read-only review is clean | 2026-08-09 | Complete |
| `C3-S3` | `accepted` | `C3-QUAL-01`/`02`, accepted prior proofs and Component 2 regressions, race, repository tests, vet, whitespace, ownership and 961 proof/dirty bounds passed; required narrow qualification revocation/finalization review clean | 2026-08-06 | Complete |
| `C3-S4` | `accepted` | Owner accepted after five allocated proofs, cumulative verification, and required narrow atomicity/accounting review passed after focused corrections | 2026-08-06 | Complete |
| `C3-R1` | `accepted` | Owner-approved truth/atomicity correction implemented; complete proof/verification set passed; the mandatory final review confirmed no remaining C3-R1 conformance or proof defect | 2026-08-06 | Complete |
| `C3-R2` | `accepted` | Distinguishing oracle proof, complete C3 ledger, package/race/repository tests, vet, whitespace, bounds, assignment, ownership, and scope inspection passed; focused independent review found no code/proof defect and its sole P3 routing finding passed focused re-review after correction | 2026-08-06 | Complete |
| Final component review | `accepted_after_owner_revision` | Original final review plus the 2026-08-09 `gpt-5.6-sol` medium focused read-only review found no remaining P1/P2/P3 after the Activity baseline correction; complete affected verification passed | 2026-08-09 | Complete |

**C3-S3 acceptance record:** Exact `[P-60s,P)`/`[P-5s,P)` gate arithmetic,
ATS-local availability, revocable provisional proofs, and the strictly
finalized same-session latch now run synchronously inside the existing
Component 2 owner/contributor path. `C3-QUAL-01` proved every inclusive
threshold, half-open edge, gap boundary, canonical revision identity, ATS
provenance, and finite-arithmetic failure. `C3-QUAL-02` proved four independent
mutable proofs, affected-proof revocation under `s < P <= s+60s`, equality
remaining mutable at `P+16m`, strict finalization one nanosecond later,
quiet-tape permanence, too-late containment, and fail-closed 961-proof/
961-dirty bounds. Construction prevents a second owner, evaluator, clock,
watermark setter, queue, registry, or publisher; invalid or overflowing state
clears success-bearing mutable evidence to `unresolved`. Restart-equivalence
inputs are retained for Component 7 without defining a checkpoint codec.
Inspection-only coverage is limited to the 60-entry finalized-overlap overflow
branch and generic corrupted-state branches; the independent reviewer found no
remaining success path through either. There was no contract, whitelist,
ownership, or interface deviation, no `C3-S4` population/ranking/publication
behavior was added, and the approved `C3-S4` assignment remains valid.

**C3-S4 owner-review record:** One engine-owned evaluator now classifies every
bound symbol into the exact primary population identities, reports
qualification and seven aggregate-feature status dimensions separately,
selects exact qualified or degraded Day-%/symbol rows with a worst-first
20-element heap, and deep-copies one validated result into Component 2's sole
private immutable publication. Changes are confined to `internal/engine`; no
public interface, provider path, queue, goroutine, callback, registry, clock,
watermark setter, second evaluator/publisher, checkpoint/API schema, or
predecessor dependency was added.

All five allocated proofs pass. `C3-POP-01` covers incomplete bootstrap,
missing/invalid prior close, attributable invalid mark, failure precedence,
exact no-print, below-price/rankable marks, first print, and earlier-mark plus
later-empty transitions while reconciling both identities. `C3-POP-02` proves
overlapping qualification/feature dimensions, keeps invalid Day arithmetic
out of the primary partition, rejects counter/cardinality contradictions, and
drives the existing cause-accurate `accounting_integrity` sentinel. `C3-RANK-01`
proves 0/1/19/20/21+ passers, high-return nonpasser filtering, exact ties,
rank continuity, uniqueness, and truncation. `C3-PROJ-01` distinguishes exact,
exact-empty, both degraded causes, unavailable, stale consequence, and
suppressed output; warming/invalid fields do not remove a trustworthy row and
only qualified-current rows are T/Q-intent eligible. `C3-EVAL-01` proves
no-`T` and existing-`T` failed-advance discard, one-symbol scratch staging,
ordinary timer-to-private-publication apply, same-`T` correction staging, and
source-equivalent evaluation.

The complete `internal/engine` suite passes, including all prior C3-S1–S3 and
Component 2 proofs. `go test -race ./internal/engine`, `go test ./...`, and
`go vet ./...` pass; changed-file trailing-whitespace inspection is clean.
Source inspection finds one production `committedT` assignment, one
evaluator-current apply point, and one atomic publication-store function.

Construction stages candidate qualification on a scratch copy of one symbol's
bounded proof maps and retains only fixed counters plus the best 20 rows. Only
a result naming the exact committed `T` can replace the current evaluation.
Before the sole cell swap, validation checks both population identities,
qualification/Day identities, every feature count, exact mode/reason
predicates, order, uniqueness, nonnegative mark age, T/Q eligibility, and exact
`min(count,20)` cardinality. Local symbol/field failures remain local; a
contradictory support fact or semantic/accounting mismatch cannot publish a
current/degraded claim.

The required `gpt-5.6-sol` medium read-only review initially found
candidate-state atomicity, invalid-Day reclassification, semantic-validator,
and proof-coverage defects. Focused corrections and re-review closed every
finding. Production live/replay commit evidence remains Components 4–6; the
ordinary apply proof seeds only Component 2's absent run-support fact in
package state. Component 6 fact authenticity, Component 7 restart reproduction,
Component 8 currentness and 100,000-symbol capacity latency, Component 9 T/Q,
Component 10 serialization, provider conformance, process OOM, and trading edge
remain unproved. No contract deviation, whitelist expansion, failed assumption,
or drift-audit `yes` remains.

The owner accepted `C3-S4` on 2026-08-06. The first mandatory final Component 3
review returned blocking findings, which were resolved through the approved
`C3-R1` and `C3-R2` corrective cycles. The later complete final review was
clean and supports the final acceptance recorded in the ledger.

### Owner-approved corrective revision `C3-R1`

The mandatory read-only final review found that accepted aggregates were being
used as interval-coverage proof, post-`T` evidence could alter or evict state
needed at a stalled committed boundary, and candidate validation occurred
after applied-state mutation. The owner approved one corrective slice because
these are one truth/atomicity boundary; the accepted four-slice history remains
intact. The affected requirements are `C3-FEAT-01`, `C3-FEAT-02`,
`C3-ACT-01`, `C3-ACT-02`, `C3-QUAL-01`, `C3-QUAL-02`, `C3-POP-01`,
`C3-POP-02`, `C3-PROJ-01`, and `C3-EVAL-01`. `C3-RANK-01` is unchanged.

For every bound symbol/second the engine distinguishes canonical `present`,
explicitly `proven_absent`, `unknown`, and localized `conflicted` evidence.
Accepted structural validity proves only that record's second. Lazy bitmap
allocation proves nothing. Exact coverage of `[a,b)` requires every
in-session second to be present or proven absent and none conflicted. Only a
future Component 6 fact naming the current binding, exact symbol/interval,
generation, reconciled fence, and uncertainty origin may install production
absence; Component 3 owns only the bounded installed consequence and its use.
Component 3 adds one lazy 7,200-byte proven-absence bitmap per printed symbol;
with Component 2 presence/conflict bitmaps the raw all-allocated ceiling is
2.16 GB at 100,000 symbols. Allocation remains lazy and bitmap existence has no
semantic meaning.

Applied feature/qualification/evaluator state at committed `T` is distinct
from bounded forward sufficient state. A post-`T` record may update the latter
but cannot replace the retained mark before `T`, enter an as-of-`T` range,
expire Activity evidence needed for catch-up, or prune qualification evidence
needed before catch-up. Folding and pruning use event-time/candidate cutoffs,
not admission time alone. No session-long canonical raw history or second
owner is introduced.

The normative transition is now canonical/forward-support mutation, pure
candidate staging, exact candidate-`T`/field/counter/row/bound/run-support/
contributor/fence/accounting validation, then one locked apply of committed
`T`, per-symbol feature/qualification results, and evaluator-current state.
Failure applies none of those values. Publication is then built and validated;
ordinary private publication requires `aggregateEvaluation.at == watermark`
(or both zero) before the sole cell swap.

**C3-R1 corrective implementation record:** Exact per-second unknown/present/
proven-absent/conflicted state, bounded uncertainty origin, covered-empty
Activity, explicit qualification-unresolved accounting, committed-`T`-safe
forward sufficient state, fixed feature status/reason-pair accounting, and
prevalidated atomic candidate apply are implemented only in `internal/engine`.
The amended primary proofs distinguish sparse acceptance and allocated bitmap
false coverage, proven absence, stalled-`T` contamination/eviction, invalid
advancing candidate mutation, publication timestamp mismatch, invalid-prior
field globalization, reason-counter contradiction, later-gap degradation,
covered-empty target mislabeling, and omitted qualification unresolved.

Verification passed on 2026-08-06: `go test -count=1 ./internal/engine`,
`go test -race -count=1 ./internal/engine`, `go test -count=1 ./...`,
`go vet ./...`, `git diff --check`, and changed/untracked-file trailing-
whitespace inspection. Source inspection finds one production committed-`T`
assignment (`applyAggregateCandidateLocked`), one evaluator-current assignment
(`runAggregateEvaluatorLocked`), and one publication-cell store wrapper
(`storePublication`). One engine owner/canonical state, one candidate apply
path, one evaluator, one publication cell, defensive publication copies, and
the approved correction-tail/extrema/Activity/qualification/scratch/counter/
20-row bounds remain. No provider, replay scheduler, checkpoint schema,
readiness threshold, T/Q, API/UI, or Component 4–11 implementation was added.
The later mandatory complete final review found no remaining C3-R1 defect and
the authoritative ledger records `C3-R1` accepted.

### Owner-approved corrective revision `C3-R2`

The remaining final-review finding showed that `C3-R1` proved physical
retention but not semantic consumption: a post-committed-`T` aggregate could
fold out of Component 2's raw correction tail into one session-aligned Activity
summary, then disappear from a later non-30-second-aligned `[T-30s,T)` target.
One aligned summary cannot reconstruct an arbitrary suffix of one block plus
prefix of the next because transactions, maximum High, and minimum Low are not
invertible from whole-block totals.

`C3-R2` replaces the single applied-target folded cache with a lazily allocated,
Activity-only folded-target projection. At most 1,920 session-aligned target
blocks retain at most 30 identity-positioned contributions each, for an exact
ceiling of 57,600 folded target contributions per printed symbol. Each
contribution contains only `Volume/ATS`, High, Low, presence, and Activity
validity. It is not a second canonical aggregate history and retains no Open,
Close, Volume, VWAP, ATS value/provenance, source position, or delivery
metadata. Evaluation combines the folded projection and Component 2 tail in
exact whole-second order; committed-`T` pruning never removes forward evidence
needed for catch-up.

The correction changes `C3-ACT-02`'s retained-state technique, finite bound,
Component 7 reproduction obligation, and primary-proof premise only. It does
not change the Activity formula, half-open window, correction horizon,
Component 1/2 authority, ownership path, committed-`T` apply point, evaluator,
publication cell, or any Component 4–11 behavior. `C3-R2` and Component 3 are
not accepted by this approval record alone; the later implementation, focused
review, and final-review evidence support the acceptance recorded above.

**C3-R2 corrective implementation record:** The engine now retains one
Activity-only folded target block per touched session-aligned block and
reconstructs every target from exact whole-second identities, preferring the
canonical correction tail before the nonoverlapping folded projection. A
permitted historical-conflict withdrawal removes its folded identity. Candidate
apply prunes only seconds strictly older than the nondecreasing target floor.

`TestC3R2FoldedNonAlignedActivityTarget` records an old committed `T`, admits
and revises post-`T` evidence, advances actual engine time beyond the strict
fold boundary while the later commit gate stays closed, proves the old applied
Activity result is unchanged and the raw tail is empty, then advances to a
later non-aligned target and compares status, reason, target transactions,
target expansion, reference count, both percentiles, and final Activity with a
full-history oracle. It inspects the exact folded slot and the 728-byte target
block charge, and separately proves deep historical withdrawal containment.

Verification passed on 2026-08-06: the complete `C3-*` proof ledger,
`go test -count=1 ./internal/engine`, `go test -race -count=1
./internal/engine`, `go test -count=1 ./...`, `go vet ./...`, `git diff
--check`, and changed/untracked-file trailing-whitespace inspection. Source
inspection still finds one production committed-`T` assignment
(`applyAggregateCandidateLocked`), one evaluator-current assignment
(`runAggregateEvaluatorLocked`), and one publication-cell store wrapper
(`storePublication`). No cross-cutting contract, owner, canonical state,
evaluator, clock, queue, publication, public interface, predecessor whitelist,
or Component 4–11 path changed. The required focused independent review found
no code or proof defect; its sole low-severity contract-routing finding was
corrected and passed focused re-review. Delegated acceptance records `C3-R2`
complete.

**Final Component 3 review record:** A separate `gpt-5.6-sol` medium read-only
review examined the complete five-document authority, routed Component 1/2 and
Phase 1 dependencies, every Component 3 implementation/proof path, correction
and retention bounds, ownership/assignment paths, predecessor coupling, and
scope. It found no actionable conformance, proof, boundedness, ownership,
scope, predecessor-coupling, or contract-routing defect. The complete engine,
race, repository, vet, and diff checks passed. Exactly one production
committed-`T` assignment, one evaluator-current assignment, and one publication
cell-store wrapper remain. Delegated final acceptance is therefore clean.

Production replay/run-support, Component 6 coverage/fence authenticity and
recovery, checkpoint reproduction, measured 100,000-symbol capacity/currentness,
T/Q, public serialization, provider conformance, process OOM/unsafe corruption,
and predictive or trading edge remain explicitly unproved and owned by later
components or operations evidence.

## 1. Outcome and user/operator consequence

Component 3 extends Component 2's one canonical aggregate path with the one
ordinary aggregate evaluator. At one committed watermark `T`, it derives the
approved aggregate features, maintains correction-aware qualification, forms
the exact complete-population classification, filters qualified rankable
symbols before exact Day-% ordering, and contributes at most 20 immutable rows
with independent field availability to Component 2's publication.

When this succeeds, the trader sees the actual qualified Day-% leaders rather
than a raw-price or partially qualified list; a high-return nonpasser cannot
displace a lower-return passer, ties and fewer-than-20 populations are
deterministic, and unavailable historical context is not displayed as zero or
allowed to suppress an otherwise trustworthy Last/Day-% row. The operator can
distinguish exact qualified ranking, the explicitly partial
`degraded_bootstrap` projection, unavailable/suppressed ranking, and the exact
population categories supporting those claims.

This component establishes scanner measurement correctness only. It does not
establish predictive value, executable prices, trading expectancy, or a public
API contract.

## 2. Scope and explicit non-scope

**In scope**

- Engine-owned bounded aggregate feature state for the product-approved Day,
  session/range, and Activity measurements.
- Same-session qualification evaluation, mutable proof state, correction
  reevaluation, and finalized session latch under Component 2's approved
  16-minute aggregate correction boundary.
- Rankability classification using the immutable Component 1 prior-close fact
  and Component 2 canonical mark/coverage evidence at committed `T`.
- One complete-population evaluator for exact primary symbol accounting,
  qualification filtering, exact Day-%/symbol ordering, top-20 truncation,
  qualified/degraded/unavailable ranking projections, and independent
  aggregate-field availability.
- Bounded engine-private results incorporated into Component 2's immutable
  publication, including correction-driven replacement at unchanged `T`.
- Component-local predecessor assessment, bounded trust/edge decisions,
  primary proofs, and the final sequential slice plan for those behaviors.

**Not in scope**

- Changing Component 1 schedule, universe, adjusted-prior-close meaning,
  binding contents, or identity.
- Changing Component 2 admission, FIFO/engine sequence, aggregate identity,
  merge/precedence, `H=16m`, finalization boundary, lifecycle, committed-
  watermark owner, publication identity/cell, or extension construction rule.
- Provider decoding, provider-specific behavior, live transport, REST
  pagination, hydration/recovery planning, terminal-work accounting,
  `no_print_through(T)` proof production, or source-position authenticity.
- Aggregate replay artifact/download/scheduling, live/replay successful commit
  evidence, checkpoint contents/codec/storage/install/cadence, or production
  evaluation delay/currentness/readiness/capacity thresholds.
- Trade/quote normalization, coverage, membership, Tape Rate, Spread, or T/Q
  pressure policy. T/Q never supplies a Component 3 input.
- Public API schema/serialization, HTTP behavior, UI presentation, database,
  service split, generic event bus, runtime plugin, or speculative feature
  extension point.

## 3. Ownership and dependencies

The single ownership boundary is one statically compiled aggregate-evaluation
contributor inside the existing `ScannerStateEngine` transition. It owns only
its named bounded engine-private feature, qualification, ranking, availability,
and population-accounting substate. It receives a copied/read-only projection
of the current transition and Component 2 canonical state, executes
synchronously in the approved fixed source order, and returns a bounded typed
result that the engine copies and applies before the existing publication
decision completes.

It has no queue, goroutine, mutable state outside the engine graph, clock,
watermark setter, lifecycle transition method, publisher, reader-facing mutable
container, or runtime registration seam. Component 2 remains the sole mutable
owner, sequence authority, committed-time owner, and publisher. Component 3 is
the single ordinary evaluator named by Phase 1, not a second evaluator.

Component 1 supplies the immutable exact-symbol population and prior-close
facts. Component 2 supplies canonical accepted aggregates, mark/presence/
conflict evidence, correction/finalization events, the candidate and committed
`T` boundary, timer-driven reevaluation, lifecycle context, atomic transition,
and immutable publication mechanism. Component 3 may extend those seams but
may not reinterpret them.

The detailed contract must prove that evaluation for a candidate committed
`T` is applied atomically with Component 2's central commit decision. It may
not publish a pre-commit shadow result, mutate state for a target that fails to
commit, or add a post-commit second evaluator. If the implemented Component 2
seam cannot support that transaction, work stops for the smallest Component 2
interface decision rather than adding another path.

## 4. Settled Phase 1 semantic boundary

### Exact Phase 1 relationship trace

| Requirement | What it requires from or constrains in Component 3 | Authoritative detail |
| --- | --- | --- |
| `PG-RANK-01` | Use only valid adjusted previous regular-session close for Day return and keep missing/invalid prior close unrankable and visible. | Ranking, availability, and delivery |
| `PG-RANK-02` | Require a trusted accepted same-session mark at committed `T`, finite price at least USD 0.25, trustworthy context, and nonnegative age; never synthesize or cross-session-fill a mark. | Ranking, availability, and delivery |
| `PG-RANK-03` | Evaluate the exact 60-second/5-second gate before ranking, treat missing seconds as inactivity, make ATS-dependent sums unavailable on any nonpositive/nonfinite contributor, and maintain the correction-aware session latch. | Qualification |
| `PG-RANK-04` | Order passers by Day return descending then exact symbol ascending, publish at most 20, preserve fewer-than-20 output, and use no secondary feature key. | Ranking |
| `PG-RANK-05` | Keep exact qualified, partial `degraded_bootstrap`, and unavailable/suppressed projections distinct; degraded ranking uses only known rankable marks, exposes covered/unresolved population, and cannot promote T/Q. | Ranking, availability, and delivery |
| `PG-FEATURE-01` | Implement the exact Day, From 4AM, HOD drawdown, and session/30m/60m range formulas at `T`; invalid, incomplete, missing, or zero-width inputs are unavailable. | Aggregate feature mathematics |
| `PG-FEATURE-02` | Implement 30-second transactions/expansion Activity against independently eligible completed 30-second blocks since the 04:00 session boundary, with no RTH/after-hours reset, at least 10 blocks, inclusive empirical percentiles, geometric combination, and `[0,100]` bound. | Aggregate feature mathematics |
| `PG-FEATURE-05` | Make every dependent aggregate feature and mutable qualification proof exactly correction-aware within the approved horizon; T/Q gap behavior remains Component 9. | Parent cross-cutting correction invariant; exact feature and qualification applications are separately routed |
| `PG-AVAIL-01` | Preserve distinct unavailable, warming, current, stale, and invalid states with bounded field-local reasons; no state is encoded as numeric zero. | Parent cross-cutting availability invariant |
| `PG-AVAIL-02` | Let incomplete From 4AM, HOD, rolling-range, or Activity history affect only dependent fields, never a trustworthy Last/Day-% row globally. | Aggregate feature mathematics |
| `PG-AVAIL-03` | Keep all T/Q failures and availability outside aggregate ranking and aggregate-field validity. | Ranking |
| `PG-OPS-01` | Keep feature/qualification results coherent at one `T` and later projectable for restart; Component 7 alone decides checkpoint contents and persistence. | Parent cross-cutting downstream constraint |
| `PG-OPS-02` | Use the same canonical evaluator after fresh bootstrap, catch-up, gap recovery, successful empty work, and ordinary live input; do not create a recovery ranking path. | Ranking, availability, and delivery |
| `PG-REPLAY-01` | Use the same aggregate feature/qualification/ranking path under deterministic replay time; Component 4 owns artifact and scheduling proof. | Ranking, availability, and delivery |
| `PG-OBS-01` | Reconcile every bound symbol into the exact mutually exclusive prior-close/mark primary identity and report qualification/features as overlapping dimensions. | Population accounting |
| `PG-OBS-03` | Expose enough bounded status to distinguish exact/degraded/stale/unavailable ranking, normal no-print from failure, and independent field currentness; Component 8 owns production thresholds/readiness policy. | Ranking, availability, and delivery |
| `PG-TAQ-03` | Ensure selected-symbol T/Q consumes, but never chooses or orders, the aggregate top 20. | Ranking |
| `ARCH-OWN-01` | Keep canonical feature, qualification, ranking, availability, and accounting state exclusively inside `ScannerStateEngine`. | Parent ownership boundary |
| `ARCH-OWN-03` | Contribute only to one immutable snapshot at one committed watermark; readers never observe partial evaluation. | Ranking, availability, and delivery |
| `ARCH-OWN-04` | Implement explicit compiled modules without creating a new authority or runtime plugin mechanism. | Parent ownership boundary |
| `DTE-CLOCK-05` | Treat the one nondecreasing committed aggregate watermark `T` as the sole time boundary for aggregate features and ranking. | Parent cross-cutting time invariant |
| `DTE-WINDOW-01` | Use the approved half-open qualification, short, session, rolling-30m, and rolling-60m intervals at `T`. | Parent cross-cutting window invariant; exact applications are separately routed |
| `DTE-WINDOW-04` | Treat absent aggregate seconds as absence/inactivity, never synthetic OHLC/volume/ATS. | Parent cross-cutting absence invariant |
| `DTE-AGG-02` | Consume canonical OHLC, volume, VWAP, ATS, and ATS provenance; divide by ATS only when finite positive. | Parent cross-cutting canonical-input invariant |
| `DTE-AGG-03` | Depend on Component 2 structural acceptance without treating structural validity as proof of coverage, rankability, or feature availability. | Parent cross-cutting trust invariant |
| `DTE-AGG-04` | Preserve the approved live/REST ATS provenance distinction and do not require their numerical equality. | Parent cross-cutting source-provenance invariant |
| `DTE-MERGE-02` | Recompute every still-mutable dependent feature and qualification proof after an accepted revision. | Parent cross-cutting correction invariant |
| `DTE-MERGE-04` | Localize REST/live conflict uncertainty to dependent historical coverage/fields while preserving an independently trusted live mark where supportable. | Parent cross-cutting containment invariant |
| `DTE-MERGE-05` | Permit a correction or availability change to replace publication at unchanged `T` with a distinct publication identity. | Ranking, availability, and delivery |
| `DTE-RECOVERY-04` | Consume, but do not produce, exact `no_print_through(T)` evidence; a later real aggregate moves the symbol into ordinary evaluation immediately. | Population accounting |
| `DTE-RECOVERY-05` | Keep symbol/history gaps local where possible and invoke the same canonical evaluator after recovery. | Ranking, availability, and delivery |
| `DTE-COMMIT-02` | Require honest population accounting and field availability for the projection being committed; partial degraded ranking remains subject to its exact product predicates. | Parent cross-cutting commit invariant |
| `DTE-COMMIT-03` | Evaluate marks, qualification, aggregate features, ranking, and accounting from one coherent state at `T`, with nonnegative mark age and no use of bars beginning at `T`. | Parent cross-cutting evaluation invariant |
| `DTE-COMMIT-04` | Contribute ordered rows, independent field availability, and exact accounting to Component 2's private immutable publication without defining Component 10's public schema. | Ranking, availability, and delivery |
| `DTE-REJECT-02` | Keep prior-close, mark, no-print, failure/fence, qualification, and historical-field categories semantically distinct rather than collapsing them into “missing.” | Population accounting |
| `DTE-TIMER-01` | Reevaluate time-window expiry, missing trailing seconds, qualification-proof finalization, and availability deterministically even with no new print. | Parent cross-cutting timer invariant |
| `DTE-TQ-02` | Prevent any T/Q gap or warm-up state from gating or advancing aggregate `T`, qualification, or ranking. | Ranking |
| `DTE-TQ-03` | Keep intentional T/Q shedding observable but irrelevant to Component 3 output. | Ranking |
| `DTE-REPLAY-01` | Consume the same normalized aggregate/state path in replay, with no second scanner implementation. | Ranking, availability, and delivery |
| `DTE-REPLAY-03` | Make timer-driven feature/ranking output invariant to replay wall speed; Component 4 owns delivery scheduling. | Ranking, availability, and delivery |
| `DTE-CHECKPOINT-01` | Keep derived state at `T0` coherent and reproducible; Component 7 owns what is persisted and how it is validated. | Parent cross-cutting downstream constraint |
| `DTE-CHECKPOINT-02` | Allow persisted qualification/history to survive only when complete restart inputs and correction state are coherent; do not decide their representation here. | Parent cross-cutting downstream constraint |
| `LIFE-MODEL-01` | Keep all Component 3 mutable state and evaluation inside the one bound engine run. | Parent ownership boundary |
| `LIFE-MODEL-02` | Complete Component 3 mutation/evaluation before atomic publication of the transition. | Ranking, availability, and delivery |
| `LIFE-MODEL-03` | Keep ranking, qualification, field availability, readiness, T/Q, and lifecycle as separate dimensions owned by one engine. | Parent cross-cutting state-dimension invariant |
| `LIFE-HYDRATE-03` | Give accepted live/historical aggregates during hydration the same ordinary feature/qualification/ranking path. | Ranking, availability, and delivery |
| `LIFE-HYDRATE-04` | Produce only qualified-current, permitted degraded-bootstrap, or unavailable ranking during hydration, with no separate watermark/path. | Ranking |
| `LIFE-HYDRATE-06` | Continue ordinary evaluation after terminal hydration even when sparse, no-print, or symbol-local history failures remain. | Ranking, availability, and delivery |
| `LIFE-LIVE-01` | Derive legal live ranking outcomes from evidence rather than row count or field completeness. | Ranking |
| `LIFE-LIVE-02` | Reevaluate on accepted aggregates, corrections, timers, and coverage changes, including same-`T` replacement. | Parent cross-cutting transition invariant |
| `LIFE-LIVE-03` | Turn a later first print after proved no-print into ordinary feature, qualification, accounting, and ranking reevaluation. | Population accounting |
| `LIFE-RECOVER-05` | Return from reconciled recovery through this same evaluator or explicit suppression; no recovery-only ranking logic. | Ranking, availability, and delivery |
| `LIFE-TQ-01` | Provide qualified-current displayed membership as a later T/Q input while degraded/stale/unavailable output creates no promotion. | Ranking, availability, and delivery |
| `LIFE-TQ-03` | Preserve aggregate evaluation unchanged under every T/Q pressure state. | Ranking |
| `LIFE-REPLAY-02` | Use the same feature, qualification, accounting, ranking, and snapshot path in replay without live-readiness claims. | Ranking, availability, and delivery |
| `LIFE-SUPPRESS-01` | Treat an unreconciled accounting invariant failure as global integrity failure while containing genuinely symbol-/field-local invalidity. | Population accounting |
| `LIFE-PUBLISH-02` | Respect the legal lifecycle/ranking combinations and leave readiness thresholds/HTTP meaning to Components 8/10. | Ranking, availability, and delivery |
| `LIFE-PUBLISH-03` | Continue ordinary evaluation after terminal startup for sparse, no-print, below-price, zero-qualified, and field-local-failure populations. | Ranking, availability, and delivery |
| `LIFE-T11` | Participate in ordinary ordered evaluation during hydration without changing the `hydrating` transition owner. | Ranking, availability, and delivery |
| `LIFE-T12` | Supply exact accounting/evaluation when terminal hydration returns through the existing `live` edge. | Ranking, availability, and delivery |
| `LIFE-T15` | Participate in the existing live self-transition for accepted facts, corrections, timers, and subordinate outcomes. | Parent cross-cutting transition invariant |
| `LIFE-T19` | Process recovery facts only through the existing recovering transition; do not advance past an unsupported gap. | Ranking, availability, and delivery |
| `LIFE-T20` | Restore currentness through the ordinary evaluator after terminal reconciled recovery. | Ranking, availability, and delivery |
| `LIFE-T23` | Participate in the existing replay transition for ordered aggregates and deterministic timers. | Ranking, availability, and delivery |

### Settled semantic input/output boundary

| Boundary item | Settled meaning | Controlling authority |
| --- | --- | --- |
| Semantic inputs | Exact immutable Component 1 binding facts; Component 2 canonically accepted aggregate state and dispositions; committed/candidate `T`, timers, lifecycle, correction/finalization, coverage/conflict, no-print/failure/fence facts supplied by their owning components. Inputs are read-only transition projections, not adapter payloads or writable state. | `ARCH-OWN-01`, `ARCH-OWN-04`, `DTE-COMMIT-03`, Component 2 `ENG-AGG-01`/`ENG-MODULE-01` |
| Owned state | Only named bounded engine-private aggregate feature summaries, qualification proof/latch state, primary classification, ranking/availability state, and the latest projection inputs needed by the one evaluator. | `ARCH-OWN-01`, `LIFE-MODEL-01`, Component 2 `ENG-MODULE-01` |
| Semantic output | One bounded typed contributor result applied inside the current transition: feature values/statuses, qualification state, exact population partition/dimensions, ranking status, total passers, and at most 20 ordered private rows. | `PG-RANK-04`, `PG-OBS-01`, `DTE-COMMIT-04`, Component 2 `ENG-PUBLISH-01` |
| Time/coherence | Every output is derived for exactly one committed `T`; an accepted correction may change it at the same `T`. Evaluation cannot use receipt time, generated time, T/Q time, recovery progress, or a checkpoint boundary as ranking time. | `DTE-CLOCK-05`, `DTE-MERGE-05`, `DTE-COMMIT-03` |
| Failure containment | Invalid or incomplete history degrades only its dependent field or symbol when the primary partition and rank claim remain supportable. A failed closed accounting identity or globally ambiguous canonical input removes the current claim through Component 2's integrity path. | `PG-AVAIL-02`, `DTE-MERGE-04`, `LIFE-SUPPRESS-01` |

### Cross-cutting invariants version 2 cannot change

1. There is exactly one canonical symbol state, ordered mutation path,
   committed aggregate watermark, ordinary evaluator, and immutable
   publication path.
2. Component 3 consumes Component 2's accepted aggregate identity,
   precedence, absence, correction, strict finalization, and `H=16m` semantics
   without reinterpretation.
3. All feature and qualification windows are half-open at committed `T`;
   missing seconds are inactivity/absence, never fabricated bars.
4. Qualification precedes ordering and truncation. Day return is the sole
   numeric ranking field and exact symbol is the only tie-breaker.
5. T/Q data, coverage, health, pressure, and measurements are never inputs to
   aggregate qualification, ranking, field validity, or committed time.
6. Aggregate fields have independent availability. Missing historical context
   cannot erase a trustworthy Last/Day-% row or masquerade as numeric zero.
7. Every bound symbol participates in exactly one primary accounting category;
   qualification, feature availability, and later T/Q coverage remain separate
   overlapping dimensions.
8. `degraded_bootstrap` is an explicitly partial projection with no T/Q
   promotion, not a weakened qualified table or separate evaluator.
9. Live, hydration, recovery, and replay use the same Component 3 path; later
   components supply evidence and scheduling but not another evaluator.
10. Component 3 defines no provider mapping, public API schema, checkpoint
    contents, readiness threshold, or speculative edge-case behavior.

## 7. Boundary-approval checkpoint

- [x] The exact controlling Phase 1 IDs are enumerated with their Component 3
      consequence and routed authoritative detail.
- [x] Outcome, single ownership boundary, dependencies, scope, and non-scope
      are explicit.
- [x] Inputs, owned state, outputs, time/coherence, and cross-cutting invariants
      prevent predecessor evidence from changing the architecture.
- [x] The proposed modular map assigns each concern, evidence question, likely
      proof family, and proposed slice to one authoritative document.
- [x] Cross-document dependencies are explicit and acyclic.
- [x] No version 2 code, test, fixture, Git history, or other predecessor
      material was opened while preparing this skeleton.
- [x] The detail documents complete reconnaissance, behavior, bounds, trust,
      edge, proof, and slice decisions without authorizing implementation.
- [x] Owner approves the Component 3 boundary, modular document map, and exact
      version 2 reconnaissance scope.

**Owner decision:** approved 2026-08-05 for the Phase 1 boundary, modular
document map, and exact question-driven version 2 reconnaissance scope. This
approval does not approve detailed behavior, predecessor reuse, primary
proofs, slices, or implementation.

The boundary gate is complete. Reconnaissance remains limited to the exact
routed scope in the detail documents.
