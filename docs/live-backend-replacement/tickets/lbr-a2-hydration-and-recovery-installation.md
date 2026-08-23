# LBR-A2 implementation assignment — hydration and recovery installation

**Status:** Draft for independent cross-check. This ticket is not a delivery
ledger and does not authorize implementation.

**Activation gate:** `LBR-P1` and `LBR-A1` must be accepted in the
[delivery program](../delivery-program.md), which must mark only `LBR-A2`
active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), and the complete
[canonical-state contract](../canonical-state-and-hydration.md). This ticket
implements only `LBR-A2` and `P-LBR-A2-HYDRATION`.

Controlling requirements are `LBR-ARCH-06`, `PG-OPS-01`, `PG-OPS-02`,
`PG-OBS-02`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `DTE-RECOVERY-01` through
`DTE-RECOVERY-05`, `LIFE-HYDRATE-01` through `LIFE-HYDRATE-07`, and
`LIFE-RECOVER-02` through `LIFE-RECOVER-06`, plus `LIFE-INIT-01`,
`LIFE-INIT-02`, `LIFE-INIT-04`, and `LIFE-INIT-05`. The immutable binding
storage, merge, and window requirements implemented by accepted `LBR-A1`
remain mandatory dependencies.

## 2. Outcome, owner, scope, and interfaces

Fresh `[S,R)` and exact gap hydration enter the accepted compact canonical
state while the subscribed live tail remains active. The engine owns one
generation/request ledger, coverage consequence, fence acceptance, and recovery
completion. One bounded REST worker owns requests, retries, pagination,
normalization, and immutable terminal results; it owns no symbol state or
currentness.

Inputs are accepted aggregate acknowledgement/epoch, engine-captured interval
and generation, bounded worker results, and the ordered ingress-fence fact.
Outputs are canonical mutations plus immutable hydration/recovery accounting,
coverage, lifecycle, and fence views for Capability B. Final decoded-fence
production is deferred to `LBR-D2`; the current adapter may implement the same
accepted producer seam temporarily.

## 3. Allowed implementation boundary

Allowed paths are hydration/recovery/generation ownership in `internal/engine`;
REST worker, result, and fence-conversion seams in `internal/massive`; and the
smallest `internal/operations` composition/configuration seam required to run
one worker. Focused tests may use those packages. `cmd/scanner` may change only
if construction cannot otherwise enforce the parent one-worker live topology;
record that need before editing.

Do not modify ranking/formulas, T/Q, decoder/queue topology, API/UI, launcher,
or checkpoint/replay implementations. Do not use credentials or provider
requests.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only current approved fixtures and the
[`REST acquisition`](../../specifications/aggregate-rest-hydration-and-recovery/rest-acquisition-and-terminal-outcomes.md)
and [`engine hydration`](../../specifications/aggregate-rest-hydration-and-recovery/engine-hydration-reconciliation-and-recovery.md)
behavior evidence. Preserve existing request shape, at most two pages, at most
three bounded attempts per page, page/row/byte limits, and one terminal outcome;
reject old ledgers, state shape, checkpoint catch-up, and worker topology.

## 5. Required implementation behavior

- Acknowledged aggregate coverage precedes engine capture of exact `R` and one
  active generation.
- The supported run mode is explicitly live. A valid installed binding before
  `S` enters `awaiting_session`; within `[S,E)` it enters
  `awaiting_aggregate_ack`; at or after `E` it enters terminal `ended` and
  cannot start hydration. Pre-session preconnect never fabricates hydration or
  current progress.
- Invalid or globally ambiguous binding evidence prevents initialization and
  cannot create partial symbol state, provider work, or a current claim.
  In-session initialization exposes bounded progress and terminal failure/
  exhaustion honestly before or while entering ordinary hydration.
- Plan every valid-prior symbol once for fresh or exact gap purpose and keep the
  live tail subscribed throughout historical work.
- Every planned request terminates exactly once as value, empty, failed,
  canceled, or fenced with binding/generation/token/symbol/interval evidence.
- Merge value rows only through the accepted A1 path; complete empty proves
  no-print only for its exact interval. Transport success and row acceptance
  remain separate.
- Replacement epoch/generation fences old rows, terminals, and markers while
  preserving already accepted canonical facts.
- Request the ingress fence only after all planned requests are terminal; the
  fence must be behind all frames already read for the accepted epoch.
- Currentness and retry-budget reset remain impossible until that exact fence
  is accepted and the ordinary evaluator succeeds.
- Work and row accounting identities reconcile before completion can be
  trusted. Recovery cannot leave an inactive terminal state freezing ordinary
  accepted live evaluation.

## 6. Primary proof and acceptance distinction

`P-LBR-A2-HYDRATION` composes a real engine and bounded fake REST/fence
producers through value, exact empty, malformed/rejected row, request failure,
cancellation, superseded generation, epoch replacement, post-live gap recovery,
REST/live delivery permutations, invalid/global-ambiguity rejection without
partial state, exact before-`S` wait/preconnect, in-session
`awaiting_aggregate_ack`, at/after-`E` terminal behavior, bounded initialization
progress/failure/exhaustion, and a marker behind already-read live work.

Observe exact work/row identities, canonical projection, local coverage
consequence, lifecycle/currentness, and absence of a current publication before
fence plus evaluation. It also observes explicit live mode, exact schedule-
dependent lifecycle/reasons, no hydration after `E`, no mutation under invalid
global binding, and honest bounded initialization progress. The dangerous
counterexample is a complete-looking initialization/generation that starts in
an illegal schedule/binding state or whose work/live tail is not reconciled.
The proof does not establish provider availability, connection retry execution,
or throughput targets.

## 7. Verification and timeout policy

Run the narrow hydration/engine proof, affected short tests, then affected race
packages with a timeout no greater than five minutes. Run focused vet,
`git diff --check`, and `go test -count=1 -short -timeout 2m ./...` at the gate.
All loops, retries, channels, and generators are bounded. Reuse unchanged
expensive evidence; no live/provider tier is authorized.

## 8. Removal and handoff

Acceptance makes old hydration proof maps, alternate aggregate backing,
checkpoint/replay-driven live installation branches, and any multiworker
ordinary-live construction removable from the supported path. Source deletion
versus separated unsupported tooling remains `LBR-E1`. Handoff records the
one-worker composition, result/fence interfaces, accounting, old paths now
removable, proof limitation, and whether Capability B remains valid.

## 9. Implementer discretion and prohibited changes

Private ledger types, result chunk size, direct owner-local result mechanics,
bounded request scheduling, cancellation wiring, and file layout are delegated.
Do not change REST/live precedence, successful-empty meaning, session/window or
fence semantics, add a hydration deadline unrelated to bounded request work,
create another recovery/currentness owner, or repair replay/checkpoints.

## 10. Containment, review, and correction

One active generation and closed terminal enums should be construction
guarantees. Runtime validation contains foreign generation/token/interval,
incomplete pagination, row conflict, epoch loss, and marker mismatch. Any
representable path from false terminal success to current output triggers a
narrow independent review. Findings or failed proofs reopen the smallest
contract/ticket through the delivery-program correction loop. Only the
orchestrator records acceptance and commits after quiescence.
