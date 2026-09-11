# LBR-A1 implementation assignment — compact canonical aggregate state

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** The [delivery program](../delivery-program.md) must record
`LBR-P1` complete and `LBR-A1` active. No later slice may write concurrently.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), and the complete
[canonical-state contract](../canonical-state-and-hydration.md). This ticket
implements only `LBR-A1` and `P-LBR-A1-CANONICAL`.

Controlling requirements are `LBR-ARCH-05`, `LBR-ARCH-07`,
`PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-REFERENCE-02`, `PG-RANK-02`,
`PG-FEATURE-07`, `DTE-SESSION-01`, `DTE-SESSION-02`, `DTE-CLOCK-01`,
`DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04`, `DTE-EVENT-03`,
`DTE-AGG-01` through `DTE-AGG-04`, `DTE-MERGE-01` through `DTE-MERGE-05`,
and `LIFE-MODEL-01` through `LIFE-MODEL-04`. Final sole-owner conformance
remains allocated to `LBR-E1`; this slice must not create a second owner or
fallback. Immutable binding storage is a dependency; A2 owns complete
schedule-dependent binding initialization.

## 2. Outcome, owner, scope, and dependencies

The one `ScannerStateEngine` live loop owns one compact aggregate state per
bound symbol: immutable reference facts, a sealed sufficient prefix, a sparse
16-minute correction tail, exact coverage/invalid/conflict evidence, and only
bounded rebuildable indexes. Accepted aggregate mutation updates that state
and only the affected symbol/proof windows.

Dependency: immutable accepted `reference.Binding`. Output: the exact
`SelectionStateView`, `SelectedAggregateView`, canonical revision, and affected-
proof notification defined by the focused contract. `LBR-A2` and Capability B
are deferred consumers. Qualification formulas, hydration execution, T/Q,
ingress decoding, API/UI, replay, checkpoint, and runtime cutover are non-scope.

## 3. Allowed implementation boundary

Allowed production ownership is `internal/engine` aggregate/binding/symbol
state and the narrow read-only view seams required by the contract. Existing
files likely replaced or reduced include `aggregate.go`, aggregate portions of
`engine.go`, and redundant aggregate-derived backing in feature files. New
private compact-state files are allowed. Focused deterministic tests may be
added under `internal/engine`.

Do not edit `internal/massive` production transport/hydration, API/UI/launcher,
`internal/checkpoint`, `internal/replay*`, or runtime composition. If an
unlisted path is necessary, stop the slice and route a ticket correction
through the delivery program before editing it.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty; do not inspect it. Reuse only the current
repository's approved aggregate tests/fixtures and the
[`S2 canonical aggregate`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/specifications/scanner-state-engine-and-canonical-state/s2-canonical-aggregates.md)
behavior evidence. The baseline oracle may be compiled test-only from commit
`0d043c1`; it cannot be a production fallback or compare private struct shape.

## 5. Required implementation behavior

- Install a fixed binding-indexed symbol array with one canonical aggregate
  representation.
- Retain at most 961 mutable second identities plus the required predecessor
  mark; sparse absence allocates no fabricated aggregate.
- Represent present, proven absent, unknown/fenced, and local invalid/conflict
  evidence exactly enough for all routed product semantics.
- Fold an identity once when strictly outside the correction horizon, update
  sufficient Volume/open/extrema/latest-mark state, and discard its mutable
  record permanently. No ordinary later live input may reopen it.
- Permit a validated current-request historical row to fill a missing identity
  anywhere in its exact session interval, including far older than 16 minutes;
  resolve result-local conflicts first, fold an accepted older fill directly
  into the prefix, and never revise a sealed live identity.
- Preserve exact duplicate, revision, withdrawal/conflict, live causal
  precedence, and source-aware historical precedence. Unequal historical rows
  for one identity withdraw only that result's historical-only value/coverage;
  historical/live mismatch preserves independently trusted live state and
  localizes only unsupported historical consequences.
- Apply inverse/forward deltas for a correction and notify only affected
  symbol/proof windows.
- Install one immutable session binding once, execute canonical/lifecycle
  mutation only on the ordered engine owner, and expose closed bounded
  transition causes/reasons without treating every status dimension as a new
  top-level owner.
- Permit only bounded indexes rebuildable from canonical state; retain no
  writable aliases or parallel canonical/feature histories.
- If Capability B temporarily needs a legacy projection, make it read-only,
  one-way, explicitly removable by `LBR-B3`, and never a merge fallback.

## 6. Primary proof and acceptance distinction

`P-LBR-A1-CANONICAL` is the sole primary proof. Drive insert, equal duplicate,
revision, withdrawal/conflict, ordinary out-of-order live input, REST/live
delivery permutations, current-token historical fill both inside and far older
than the horizon, exact live 16-minute equality/one-tick boundaries, no-print,
invalid, and unknown coverage through the baseline semantic oracle and the
replacement production state. Include historical/historical conflict
withdrawal and historical/live mismatch that preserves trusted live state.

Observe complete marks, coverage classes, prefix/tail sufficient inputs,
affected-proof notifications, disposition accounting, retained record/byte
bounds, single-binding/owner-ordered transitions with bounded reasons, and
absence of a second mutable state. The dangerous counterexample is a latest-
price-equivalent or horizon-blind implementation that rejects valid old
historical fill, revises sealed live state, or loses coverage/correction
effects. The proof does not establish full hydration, scale, or provider
correctness.

## 7. Verification and timeout policy

Run the narrow primary proof first, then affected `internal/engine` short tests.
Run `go test -count=1 -race -short -timeout 5m ./internal/engine`, focused vet,
`git diff --check`, and at the slice gate
`go test -count=1 -short -timeout 2m ./...`. Validate generated fixture counts
and tail boundaries before timing. No command may access providers.

## 8. Removal and handoff

Acceptance makes the old `symbolAggregateState` map/bitmap backing and parallel
mutable `priceRange`, `activity`, and `mvpMeasurements` aggregate copies
removable as authorities. Record any temporary projection adapter precisely for
`LBR-B3`. Handoff must name the final private owner, immutable views, actual
state bounds, changed files, proof result/limitation, and whether `LBR-A2`
remains valid.

## 9. Implementer discretion and prohibited changes

Private value layout, arrays, interval encoding, delta indexes, pooling proven
necessary by measurement, error wrapping, and file organization are delegated.
Do not change product formulas, source precedence, session/window meaning,
the 16-minute horizon, reference acquisition, readiness, publication/API
meaning, or add workers, a database, generic bus, plugin system, replay, or
checkpoint compatibility.

## 10. Containment, review, and correction

Writable aliasing and a second production state must be prevented by
construction. Foreign binding/source position, malformed values, nonprecedent
input, and local conflict remain runtime-contained; global canonical/accounting
ambiguity closes currentness under the parent rule. A narrow read-only review
is triggered only if alias/ownership or correction-boundary false success is
not excluded by construction and the primary proof. Findings reopen this
ticket/spec through the delivery-program correction loop. Only the orchestrator
records acceptance, stages, or commits after all workers/reviewers are quiet.
