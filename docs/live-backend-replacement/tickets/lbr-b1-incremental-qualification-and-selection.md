# LBR-B1 implementation assignment — incremental qualification and selection

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** The [delivery program](../delivery-program.md) must record
`LBR-P1` and Capability A accepted and mark only `LBR-B1` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical dependency](../canonical-state-and-hydration.md), and the complete
[evaluation contract](../evaluation-and-publication.md). This ticket implements
only `LBR-B1` and `P-LBR-B1-SELECTION`.

Controlling requirements are `LBR-ARCH-08` selection-phase obligations,
`PG-RANK-01`, `PG-RANK-03`, `PG-RANK-04`, `PG-RANK-05`, `PG-OBS-01`,
`DTE-CLOCK-04`, `DTE-CLOCK-05`, `DTE-TIMER-01`, `DTE-COMMIT-01` through
`DTE-COMMIT-03`, `DTE-REJECT-02`, and the system-position portion of
`DTE-EVENT-04`; replay position semantics remain excluded.

## 2. Outcome, owner, scope, and interfaces

The sole engine owner maintains qualification proof state incrementally and,
at most once per unique causally supported second/revision, scans compact
selection scalars for the complete population to produce exact Day-%/symbol
top 20 and primary accounting. It does not clone qualification maps, scan
session history, or evaluate display-only fields.

Inputs are accepted A1 canonical revisions, affected-proof notifications,
coverage/currentness facts, engine time, and immutable binding/reference facts.
Outputs are bounded per-symbol proof scalars, exact selection/accounting result,
desired ranked symbols for Capability C, and the cycle identity consumed by
`LBR-B2`. Enrichment/publication, T/Q, ingress, and API/UI are deferred.

## 3. Allowed implementation boundary

Allowed production paths are qualification, evaluator selection stage, timer/
watermark cycle control, and narrow canonical-state proof fields in
`internal/engine`. New private proof-index/selection files and focused engine
tests are allowed. Do not edit display-field implementations, T/Q, massive
transport/hydration, snapshot API/UI, launcher, replay, or checkpoint code.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only the current approved
[`qualification/corrections`](../../specifications/aggregate-features-qualification-ranking-and-accounting/qualification-and-corrections.md),
[`ranking/availability`](../../specifications/aggregate-features-qualification-ranking-and-accounting/ranking-availability-and-delivery.md),
and [`population accounting`](../../specifications/aggregate-features-qualification-ranking-and-accounting/population-accounting.md)
fixtures/oracles. Old cloned maps, full evaluator, private state, delivery
ledger, replay, and checkpoint cases are rejection evidence, not requirements.

## 5. Required implementation behavior

- Maintain bounded provisional/final proof evidence per symbol; accepted
  mutation/correction/coverage/timer updates only endpoints it can affect.
- Preserve the approved gate, mutable proof revocation, strict finalization,
  first-later-print behavior, and quiet-time advancement without invented
  market evidence.
- Schedule at most one full-population selection for a unique binding,
  canonical revision, and candidate `T`; a fence plus timer cannot duplicate it.
- Preserve one engine-owned monotonically increasing system position for timer
  and owner-control work; callers cannot select or reuse that position.
- Scan the fixed symbol array once, classify every primary population and
  qualification bin, select the latest trusted mark strictly before candidate
  `T`, compute approved Day-%, and keep exact top 20 by Day-% descending then
  symbol ascending. A mark whose window starts at or after `T` cannot rank or
  displace an older eligible mark retained by compact canonical state.
- Preserve that as-of-`T` mark rule when committed time stalls longer than the
  correction horizon, accepted post-`T` rows compact, and loss/gap recovery
  evaluates the still-older or recovered `T`; selection cannot substitute the
  globally latest accepted mark for the eligible pre-`T` mark.
- Preserve exact fewer-than-20, tie, exact-empty, degraded/unavailable, invalid
  prior/mark, and incomplete-population behavior.
- T/Q, Float, and aggregate display-field state do not enter qualification,
  Day-%, order, watermark, or backend readiness.
- Retain fixed counters/scalars only; no population-sized map clone, per-cycle
  history copy, or selection-created evidence is allowed.

## 6. Primary proof and acceptance distinction

`P-LBR-B1-SELECTION` drives gate boundaries, multiple provisional proofs,
correction revocation at strict/equality edges, finalization, first later print,
quiet timers, fewer-than-20, exact ties, invalid prior/mark, incomplete
population, absent T/Q/display fields, and monotonic nonreused system positions
across timer/control work through the production path and approved semantic
oracle. It also stalls committed `T` longer than the 16-minute correction
horizon, compacts accepted rows on both sides of `T`, performs loss and exact
gap recovery, and proves that a mark with `window_start == T` or after `T`
cannot rank or displace the retained latest eligible mark strictly before `T`.

Observe proof classes, Day-%, order, watermark candidate/commit, accounting,
affected symbols, and exact count of full-population cycles. The dangerous
counterexamples are cloned/rescanned proof state, caller-selected/reused system
position, a duplicated same-prefix cycle, future-to-`T` selection after a long
stall/recovery, or T/Q/display-field-dependent ranking. The proof does not
establish enrichment, immutable capture, or mature resource acceptance.

## 7. Verification and timeout policy

Run the primary proof and direct qualification/ranking tests, affected short
engine tests, affected engine race under five minutes, focused vet,
`git diff --check`, then `go test -count=1 -short -timeout 2m ./...` at the
slice gate. Any generated population validates cardinality and proof events
before timing. No provider or non-short resource tier is authorized here.

## 8. Removal and handoff

Acceptance makes qualification-map cloning, full-history qualification scans,
and duplicate full-population selection triggers removable. Handoff records
the compact proof shape, cycle identity, exact accounting/order evidence,
temporary compatibility seam if any, proof limitation, and whether `LBR-B2`
remains valid.

## 9. Implementer discretion and prohibited changes

Proof indexes, fixed arrays/heaps, dirty sets, revision guards, timer
coalescing, and private file layout are delegated. Do not change product gate
or ordering formulas, semantic delay, watermark/currentness, aggregate merge,
field availability, T/Q independence, or introduce another evaluator/owner,
generic scheduler, database, replay, or checkpoint work.

## 10. Containment, review, and correction

Caller-selected proof state, selection membership, or watermark must be
prevented by private types/owner-local execution. Runtime validation contains
revision mismatch, clock regression, invalid accounting, and unsupported
coverage. A remaining correction/finalization or cycle-linearization false
success triggers narrow review. Failed evidence reopens the smallest ticket or
focused contract through the program loop; only the orchestrator records
acceptance and commits after quiescence.
