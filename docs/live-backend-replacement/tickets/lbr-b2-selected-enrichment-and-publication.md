# LBR-B2 implementation assignment — selected enrichment and publication

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** Capability A and `LBR-B1` must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-B2` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical contract](../canonical-state-and-hydration.md), the complete
[evaluation contract](../evaluation-and-publication.md), and the recorded
`LBR-B1` handoff. This ticket implements only `LBR-B2` and
`P-LBR-B2-PUBLICATION`.

Controlling requirements are `LBR-ARCH-04`, the enrichment/publication portions
of `LBR-ARCH-08`, `LBR-ARCH-09`, `PG-FEATURE-01`, `PG-FEATURE-02`,
`PG-FEATURE-03`, `PG-FEATURE-04`, `PG-AVAIL-01`, `PG-AVAIL-02`,
`PG-OBS-03`, `DTE-CLOCK-06`,
`DTE-COMMIT-04`, `LIFE-LIVE-01` through `LIFE-LIVE-03`, `LIFE-LIVE-05`,
`LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, `LIFE-END-01` through
`LIFE-END-03`, and `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`.

## 2. Outcome, owner, scope, and interfaces

For one accepted B1 selection and canonical revision, the engine enriches only
selected rows with current-product aggregate fields, Float, and an immutable
selected T/Q view, validates all accounting/trust identities, and atomically
replaces one coherent API-v2-compatible publication. API capture loads that
cell once without engine mutation locks or later market/TQ joins.

Inputs are B1 cycle/selection, A1 selected aggregate views, binding Float facts,
engine lifecycle/fence/currentness, and the current bounded T/Q projection
(temporarily from the existing implementation until Capability C). Outputs are
one immutable publication and desired-symbol revision. T/Q formulas/membership,
HTTP representation, browser logic, and ingress are non-scope.

## 3. Allowed implementation boundary

Allowed production paths are current-product aggregate-field evaluation,
publication construction/validation/atomic storage, immutable engine snapshot
views, and the smallest `internal/operations` capture seam needed to keep one
atomic load and cached diagnostics. Focused `internal/snapshotapi` tests may
assert unchanged v2 mapping. The narrow canonical seam may be corrected to
return one selected symbol's immutable projection explicitly bound to candidate
`T` and the accepted preselection revision; it may not create another canonical
owner or expose writable/history aliases. Production schema/semantics may not
change. UI, massive transport, T/Q policy, replay, and checkpoint paths are
excluded.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse the current approved product fixtures,
the focused contract's named qualification/ranking/accounting evidence, and the
[`snapshot capture correction`](https://github.com/belandresj/live-equities-momentum-scanner/blob/eda0d46eccb95ac96a239e6de50bcf2f26a52e04/docs/snapshot-capture-and-dashboard-transport-isolation-correction.md).
Old aggregate feature implementations are oracles only when they implement a
still-current formula; HOD/rolling/old Activity/Tape and cross-snapshot joins
are explicitly rejected.

## 5. Required implementation behavior

- Capture one accepted selection/canonical revision and reject/fence a stale
  apply rather than joining mixed revisions.
- Evaluate/read Volume, From Open, Day Range, Activity 30s, and Move 30s only
  for at most 20 selected rows using the same preselection canonical state and
  strict as-of-`T` intervals. The selected projection remains exact when
  `PrefixFoldedThrough` is after `T`: compacted post-`T` Volume, open/extrema,
  Activity, Move, or mark evidence cannot enter the row.
- Keep aggregate sufficient history selection-independent: a symbol absent for
  more than 330 seconds enters with immediate exact aggregate-derived fields;
  only Tape/Spread may warm.
- Preserve independent current/warming/unavailable/stale/invalid states,
  reasons, genuine zero, optional Float, and full-population fixed status
  accounting without evaluating discarded display fields.
- Join one immutable T/Q projection without letting it alter aggregate ranking,
  watermark, currentness, or readiness.
- Publish at ordinary successful one-second cadence; publish an immediate new
  identity for a trust closure that would otherwise expose false currentness,
  field validity, coverage, lifecycle, or terminal state.
- Seal all members before one atomic store. Capture performs one atomic load,
  no engine lock, no per-request `ReadMemStats`/checkpoint read, and no later
  engine/TQ merge. Slow/canceled API work remains isolated.

## 6. Primary proof and acceptance distinction

`P-LBR-B2-PUBLICATION` keeps a symbol unselected for more than 330 seconds,
selects it only through Day-%, and requires exact aggregate fields while T/Q
alone warms. Compose corrections, genuine zero, no-print, independent field
failures, same-`T` trust closure, concurrent captures, and slow/canceled clients.
It also stalls committed `T` beyond the 16-minute correction horizon, compacts
accepted rows after `T`, loses the epoch, performs exact gap recovery, and
requires the old/recovered-`T` selected fields to match `[S,T)` and their exact
`T`-relative windows with all compacted future rows excluded.

The production publisher/capture trace distinguishes successful empty from a
malformed or incomplete terminal, cancellation, and binding/generation/epoch
replacement. Successful empty becomes resolved no-print only after its exact
fence and accepted evaluator cycle; every other terminal remains explicit
noncurrent/unknown and cannot publish a current complete population.

Observe one publication identity/revision/`T`, exact ordered rows/field states/
accounting, immediate closure, no duplicate same-prefix cycle, detached
captures, and engine-lock independence. The dangerous counterexample is a
current snapshot joining different revisions, future-to-`T` compacted state,
selection-warmed aggregate history, or unresolved terminal evidence. The proof
does not establish final T/Q formulas or HTTP/browser presentation.

## 7. Verification and timeout policy

Run the primary proof, direct current-feature/publication/capture regressions,
affected short packages, and affected engine/operations/snapshot race tests
under five minutes. Run focused vet, API-v2 golden tests, `git diff --check`,
and ordinary `go test -count=1 -short -timeout 2m ./...` at the gate. Do not run
provider or integrated 10-minute E2 tier.

## 8. Removal and handoff

Acceptance makes full-universe display-field evaluation, separately sampled
engine/TQ joins, mutable publication members, lock-coupled capture, and
per-request expensive diagnostics removable. Record any old feature/projection
adapter still needed solely for `LBR-B3` or Capability C, plus publication/
capture interfaces, proof limitation, and whether `LBR-B3` remains valid.

## 9. Implementer discretion and prohibited changes

Selected-row builders, fixed arrays, immutable layout, dirty/revision flags,
atomic cell mechanics, and private helpers are delegated. Do not change
product formulas/order/delay, API-v2 meaning, browser behavior, T/Q policy,
readiness, canonical merge, or add streaming transport, another snapshot
owner, replay/checkpoint compatibility, database, or service split.

## 10. Containment, review, and correction

Immutable publication aliases and caller-triggered evaluation must be prevented
by construction. Revision, accounting, clock, and publication contradictions
fail closed before atomic replacement. Any remaining mixed-revision or
false-current linearization triggers narrow review. Findings reopen the
smallest ticket/spec through the delivery loop; only the orchestrator records
acceptance and commits after all actors are quiet.
