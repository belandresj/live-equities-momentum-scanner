# LBR-B3 implementation assignment — remove superseded evaluation

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** `LBR-B1` and `LBR-B2` must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-B3` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical contract](../canonical-state-and-hydration.md), the complete
[evaluation contract](../evaluation-and-publication.md), and recorded B1/B2
handoffs. This ticket implements only `LBR-B3` and `P-LBR-B3-REMOVAL`.

Controlling requirements are the removal obligations under `LBR-ARCH-05`,
`LBR-ARCH-08`, `LBR-ARCH-09`, and parent Section 9, while preserving all
product requirements allocated to Capability B and the exact API-v2 live
projection. `LBR-ARCH-13` final conformance remains allocated to `LBR-E1`.

## 2. Outcome, owner, scope, and dependencies

Delete the old full-population evaluator, redundant aggregate-derived state,
and every active removed-product feature path after B1/B2 provide the sole live
behavior. The supported engine retains no HOD drawdown, rolling 30/60-minute
range, transaction/range-expansion Activity, one-second Tape burst, qualification
clone, or legacy evaluation fallback.

The currently accepted T/Q/Tape 5s behavior must remain intact for later
replacement by Capability C; Capability C is not a prerequisite for `LBR-B3`.
Final runtime cutover and replay/checkpoint source disposition remain `LBR-E1`.

## 3. Allowed implementation boundary

Allowed paths are obsolete/current evaluator and feature files/tests in
`internal/engine`, the smallest snapshot/view/mapper removals required to
eliminate dead fields, and focused resource/semantic diagnostics. Delete files
when no retained behavior imports them. Do not rewrite historical documents or
change UI/API fields. Do not edit massive ingress/TQ policy except to remove an
already-unused compile dependency.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. The accepted A1/B1/B2 proofs and current API-v2
golden are authority for behavior. Baseline old-feature tests are exclusion
evidence only. Reuse the focused contract's mature evaluator diagnostic routes
for before/after measurement; do not preserve private structures just to keep
old tests compiling.

## 5. Required removal behavior

- Remove active state, mutation, evaluation, publication, mapper, tests, and
  fixtures for the four removed product feature families.
- Remove qualification cloning/full-history rescans and old full-universe
  display-field evaluation rather than leaving unreachable alternative calls.
- Remove A1 temporary legacy projections when no accepted consumer remains.
- Prove the production dependency graph cannot construct, mutate, publish, or
  select the old evaluator/state; test-only semantic oracles remain isolated.
- Preserve exact qualification, Day-%/symbol ordering, Volume, From Open, Day
  Range, Activity 30s, Move 30s, availability/accounting, publication, and API
  v2.
- Record mature cycle latency/allocation versus the frozen baseline without
  treating the numeric target as an independent gate.

## 6. Primary proof and acceptance distinction

`P-LBR-B3-REMOVAL` combines source/dependency exclusion with the complete
current-product semantic/API corpus. It must fail if the supported live build
still contains an active removed field, legacy evaluator call, qualification
clone, redundant owner, runtime compatibility selection, or temporary adapter.

Observe unchanged current-product outputs/accounting and recorded mature
cycle/allocation differential. The dangerous counterexample is dead-looking
code that still mutates or remains selectable. This proof is not the E2
30-minute whole-process resource acceptance and does not decide final
replay/checkpoint tooling.

## 7. Verification and timeout policy

Run the exclusion proof, affected current-product engine/API tests, affected
race under five minutes, focused vet, `git diff --check`, and ordinary
`go test -count=1 -short -timeout 2m ./...`. Run only the allocated bounded
non-short mature evaluator comparison with a manifest and timeout no greater
than 15 minutes. Reuse unchanged evidence and do not optimize repeatedly to a
numeric target.

## 8. Removal and handoff

This slice's outcome is deletion, not another compatibility layer. Handoff
lists deleted files/symbols/tests, retained current-product/TQ seams, source
inspection result, semantic/resource evidence and limitations, and all old
Capability B code now absent. Capability B then receives its required final
read-only review before Capability C can activate.

## 9. Implementer discretion and prohibited changes

File consolidation, private helper deletion, test-oracle isolation, and the
smallest mechanical interface cleanup are delegated. Do not change product/API
meaning, T/Q Tape 5s/Spread, canonical state, ingress, readiness, or use this
slice for broad renaming, historical-doc cleanup, replay/checkpoint repair, or
speculative performance work.

## 10. Containment, review, and correction

Compile/dependency/source assertions should prevent a hidden production
fallback by construction. Semantic and resource regressions reopen the lowest
affected A/B slice; preserve unaffected evidence. Capability B final review
focuses on ordering, selection-independent history, accounting, one immutable
publication, and actual deletion. Only the orchestrator updates the sole ledger
and commits after quiescence.
