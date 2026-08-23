# LBR-C2 implementation assignment — cadence membership and trust transitions

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** `LBR-C1` and all A/B dependencies must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-C2` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), and complete
[T/Q](../tq-state.md) contracts, plus the recorded C1 handoff. This ticket
implements only `LBR-C2` and `P-LBR-C2-TQ-MEMBERSHIP`.

Controlling requirements are `PG-TAQ-01`, `PG-TAQ-02`, `PG-TAQ-03`,
`DTE-TQ-03`, and `LIFE-TQ-01` through `LIFE-TQ-03`, including data-confirmed
membership, direct-pressure predicates, immediate trust closure, and aggregate
independence. Accepted `LBR-C1` supplies fixed field-availability and causal-
coverage dependency behavior rather than reallocating it here.

## 2. Outcome, owner, scope, and interfaces

The engine derives desired T/Q membership only from current qualified rows,
batches ordinary removals/additions at the combined cadence, owns command
tokens/generations/pressure/restoration, and publishes every trust closure
immediately. Ingress only serializes writes, supplies `B`, normalizes data, and
returns fixed pressure/drop facts.

Inputs are accepted B selection revisions, C1 channel state, typed command
write results/data/drop facts, and engine-issued pressure samples. Outputs are
one private command at a time, early-shed/aggregate-only policy, bounded
membership/accounting view, and publication revision. Capability D later
replaces the producer/executor mechanics without changing these semantics.

## 3. Allowed implementation boundary

Allowed paths are membership/command/pressure/public-projection ownership in
`internal/engine`; the smallest current `internal/massive` command conversion,
write-result, and early-shed seam; fixed pressure sampling/composition in
`internal/operations`; and exact API mapping tests if needed. Do not replace
the decoder/raw queue in this slice, change aggregate evaluation/readiness, or
edit UI/launcher/replay/checkpoint paths.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only the focused T/Q contract's current
data-confirmation, membership, pressure, recovery, mixed-frame, accounting, and
API fixtures. Generic success-count/deadline acknowledgements, status
quarantine, serial ordinary rank-churn chains, heap/goroutine/delivery-latency
pressure predicates, replay, and historical ledgers are rejected.

## 5. Required implementation behavior

- Desired membership is exactly current qualified displayed rows, at most 20;
  noncurrent/degraded/suppressed/ended state closes additions.
- On a fresh accepted epoch, issue one sorted paired subscribe batch for the
  full desired set. Ordinary cadence computes one diff, closes removals locally,
  writes at most one sorted unsubscribe batch, then one sorted additions batch
  after its write result. One dynamic write is in flight.
- A write records requested/failed only. Per-channel post-`B` data confirms
  coverage; generic success is diagnostic, silence is not failure, and provider
  error closes T/Q additions for the epoch without touching aggregates.
- Rank removal/epoch loss/control error/pressure/bound closes local coverage and
  invalidates generations immediately. Returning symbols always reconfirm.
- Pressure samples are private, fixed-cardinality, in-order, one outstanding,
  and accepted within two seconds. Missing/late samples cannot create a streak.
- Apply exact 10%/one-second/two-sample degraded, 25%/two-second/three-sample
  aggregate-only, guarded watermark-lag, immediate loss/accounting/bound, and
  five-sample below-1%/strictly-below-750ms recovery rules from the contract.
- At degraded pressure, shed T/Q before expensive normalization but classify
  mixed frames fully. Aggregate-only removes membership. Frame-local 500 ms
  shedding cannot change global pressure.
- Restoration cleans up first and adds at most one highest-ranked symbol per
  successful cadence; every restored generation warms fresh.
- Command/fact/retention/membership accounting reconciles. Trust closure
  publishes immediately; ordinary values coalesce. Aggregate ranking,
  watermark, readiness, and admission accounting remain unchanged.

## 6. Primary proof and acceptance distinction

`P-LBR-C2-TQ-MEMBERSHIP` composes fresh batch, rapid churn, removal-before-
addition, write failure, generic/late status, data confirmation, quiet
unconfirmed channels, provider error, exact pressure entries, early mixed-frame
shedding, membership to zero, 749/750-ms recovery edge, gradual restoration,
epoch reset, and immediate trust publication.

Observe one write, bounded members, exact generations/coverage/pressure/
accounting, aggregate publication equality except T/Q revision, and zero
aggregate/control loss. Dangerous counterexamples are status-created coverage,
pressure dropping a mixed aggregate, or T/Q changing readiness/rank. It does
not prove provider acceptance or the final D2 queue.

## 7. Verification and timeout policy

Run the primary proof, direct data-confirmation/pressure/accounting tests,
affected short packages, engine/massive/operations race under five minutes,
focused vet, API golden where touched, `git diff --check`, and ordinary
`go test -count=1 -short -timeout 2m ./...`. Use deterministic sample sequences,
not sleeps. No provider/capacity/live tier.

## 8. Removal and handoff

Acceptance makes status-count/deadline coverage, obsolete status quarantine,
per-symbol ordinary churn sequencing, old membership liabilities, competing
pressure state, and separately joined mutable T/Q publication removable.
Handoff records engine/ingress interfaces, exact bounds/counters, adapter
mechanics deferred to D, proof limitation, and triggers Capability C final
read-only review before D1.

## 9. Implementer discretion and prohibited changes

Private command/set-diff representation, fixed arrays, closed enums, sample
tokens, and cadence bookkeeping are delegated. Do not invent provider ack/
retry semantics, change thresholds/horizons/formulas, make diagnostics pressure
inputs beyond the closed set, change aggregate behavior/readiness, add workers/
queues, or implement replay/checkpoint/public deployment.

## 10. Containment, review, and correction

Private tokens/generations and one in-flight command prevent caller-selected
coverage/order. Runtime validation fences stale result/sample/epoch and closes
T/Q on bound/accounting failure. A command-boundary, pressure-transition, or
mixed-frame loss linearization triggers narrow review. Findings reopen the
smallest ticket/spec; final Capability C review focuses on bounded retention,
provider assumptions, coverage honesty, and aggregate independence. Only the
orchestrator updates the ledger or commits after quiescence.
