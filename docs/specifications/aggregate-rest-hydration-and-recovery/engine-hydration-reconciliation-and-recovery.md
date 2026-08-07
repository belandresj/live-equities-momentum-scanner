# Aggregate REST hydration and recovery — engine hydration, reconciliation, and recovery

**Parent contract:** [Aggregate REST hydration and recovery](../aggregate-rest-hydration-and-recovery.md)

**Normative responsibility:** Engine-owned hydration plans, generations,
request/result ledgers, historical admission and consequences, mid-epoch live
ingress fences, startup/no-print reconciliation, and same-process exact-gap
recovery through ordinary evaluation.

**Controlling requirements:** `PG-RANK-02`, `PG-RANK-05`, `PG-FEATURE-05`,
`PG-AVAIL-01`, `PG-AVAIL-02`, `PG-OPS-01`, `PG-OPS-02`, `PG-OBS-01`,
`PG-OBS-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-04`,
`ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-MODEL-01`–`DTE-MODEL-03`,
`DTE-CLOCK-05`, `DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04`,
`DTE-EVENT-01`, `DTE-EVENT-03`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`,
`DTE-MERGE-01`–`DTE-MERGE-05`, `DTE-RECOVERY-01`–`DTE-RECOVERY-05`,
`DTE-COMMIT-02`–`DTE-COMMIT-04`, `DTE-CHECKPOINT-02`,
`DTE-CHECKPOINT-03`, `DTE-REJECT-01`, `DTE-REJECT-02`,
`LIFE-MODEL-01`–`LIFE-MODEL-04`, `LIFE-INIT-03`, `LIFE-INIT-04`,
`LIFE-HYDRATE-01`–`LIFE-HYDRATE-07`, `LIFE-LIVE-01`–`LIFE-LIVE-03`,
`LIFE-RECOVER-01`–`LIFE-RECOVER-06`, `LIFE-END-01`–`LIFE-END-03`,
`LIFE-PUBLISH-02`, `LIFE-PUBLISH-03`, `LIFE-T06`, `LIFE-T08`,
`LIFE-T11`–`LIFE-T14`, `LIFE-T16`, `LIFE-T18`–`LIFE-T22`, `LIFE-T26`,
`LIFE-T29`; `C6-PLAN-01`, `C6-LEDGER-01`, `C6-MERGE-01`,
`C6-FENCE-01`, `C6-START-01`, `C6-NOPRINT-01`, `C6-RECOVER-01`, and
`C6-INTEGRATION-01`

**Allocated slices:** Approved `C6-S2`, `C6-S3`, and `C6-S4`

**Document dependencies:** [Parent](../aggregate-rest-hydration-and-recovery.md),
[REST acquisition detail](rest-acquisition-and-terminal-outcomes.md), Components
1–3 accepted contracts, Component 4's shared mapper/value seam, and only the
approved Component 5 acknowledgement/epoch/loss/ingress meanings named by the
parent

**Approval state:** Owner-approved 2026-08-06 as part of the complete modular
Component 6 contract; this detail is not an independent authority

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-rest-hydration-and-recovery.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance used V2 commit `5f92a151dd850002578a33a81ad90dea096c63b6`
and exact worktree bytes identified below. All listed engine/recovery sources
were modified or untracked. No V2 claim overrides Phase 1 or Components 1–5.

| Exact V2 source | File SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `internal/scanner/recovery_binding.go`: `RecoveryBinding` and `AggregateWorkToken` only | `ea22afd247f3b190e0bbf6037d2485a18d3a71181bcb1b4a9683c33e3655f1ec` | Binding/generation token stability and mutation cases are useful identity evidence. | `behavior evidence` | Use the accepted Component 1 binding identity directly; do not duplicate binding/product/checkpoint hashes or JSON. Add one engine-allocated request ID and exact interval/purpose. | `P-C6-PLAN`, `P-C6-LEDGER` |
| `internal/scanner/recovery_phase2a.go`: `BeginFreshBootstrapBound`, `BeginCheckpointCatchupBound`, `BeginSameProcessRecoveryBound`, `RecoveryPlanningView`, `PlanRecovery`, and `recoveryIntervalStart` | `5a747c54f92e29c0cfb4d5ebd8f05f3fd73bc794ef04eab48d9cb4723cbea07e` | V2 computes acknowledgement catch-up boundaries, supports the three modes and batched full-population work, and demonstrates a 16-minute same-process overlap. It also ranks/prioritizes work, owns field debt/evaluation, allows an `E+1s` gap, forces at least `[S,S+1s)` at pre-session handoff, and can stop planning after a qualification frontier. | `reject` production code; `behavior evidence` for interval counterexamples | Plan every valid-prior-close binding symbol independently of rank/feature debt. Use exact Phase 1 `R`, including empty `[S,S)`; never request beyond `E`; same-process base interval is `[T_supported,R)`, not predecessor `T-16m`; no recovery evaluator or prioritized omission. | `P-C6-PLAN`, `P-C6-START`, `P-C6-RECOVER` |
| Same file: `PrepareRecoverySymbol`, `CommitPreparedRecovery`, `AdvancePreparedRecoveryTrust`, `TerminalRecoverySymbol`, and `EvaluateRecovery` | same | Atomic per-symbol preparation, duplicate/conflict checks, post-fence trust advancement, and local field containment are useful regressions. The code creates a second merge/evaluator/trust graph and infers empty from `len(bars)==0`. | `reject` production code; `behavior evidence` for counterexamples | Use Component 2's sole historical aggregate acceptance and Component 3's sole evaluator. Terminal result and coverage consequences live in the C6 engine ledger; empty is an explicit terminal outcome. | `P-C6-MERGE`, `P-C6-NOPRINT`, `P-C6-INTEGRATION` |
| `internal/scanner/phase2a_recovery_test.go`: `TestPhase2AFrontierProof`, `TestPhase2ARecoveryPlanningDeterministicAndBatched`, `TestRecoveryPlannerStopsQualificationOnlyWorkAfterFrontierProof`, `TestPhase2ARecoveryMergeAndIndependentFields`, `TestPhase2AGlobalVersusLocalPredicates`, `TestPhase2ARecoveryStateTransitions` | `1a6c23db6a84297c03e38641a887e0044904ef9051195d2332ca19210e5b5fe6` | Provides deterministic planning, merge/conflict, local/global, and transition scenarios. Frontier-driven work stopping is precisely unsuitable for Phase 1 complete-population hydration. | `behavior evidence` | Keep identity/permutation/conflict/failure shapes; invert rank/frontier omission cases so every planned symbol reaches a terminal result. Replace V2 stages with Phase 1 lifecycle. | `P-C6-PLAN`, `P-C6-MERGE`, `P-C6-RECOVER` |
| `internal/scanner/phase2a_acceptance_remediation_test.go`: binding stability and terminal requirement cases | `517a963b1640dafd5f800795a09e55861eeabc0cd5238a1139c2c245ec5e35fd` | Demonstrates stable identity and that field-local historical failure need not revoke an independently current mark. | `behavior evidence` | Reuse Component 1 identity and Component 3 independent availability; no V2 trust structs or serialization assertions. | `P-C6-LEDGER`, `P-C6-MERGE` |
| `internal/massive/owner.go`: recovery fields and only initialization/checkpoint-fallback handoff, `processRecovery`, `planRecovery`, `startPlannedRecoveryBatch`, `applyRecoverySymbol`, `completeRecoveryBatchBoundary`, `completeRecoveryHealth`, `deferRecovery`, `suppressRecovery`, `beginRecoveryGeneration` | `f3e68f7b233a6ad6ab82e61889e60e37b128f5d747d3bf826380af622d091700` | Pre/post `GreatestAccepted` fences, stale binding/generation checks, per-symbol streaming, cancellation, worker join, and final-disconnect interleavings are useful. The outer owner also mutates recovery, health, evaluation, readiness, rows, and publication; uses `deferred`; polls queue/decoder state; hardcodes generation policy; and directly commits recovery outside one FIFO. | `reject` production code; `behavior evidence` for fence/terminal regressions | One engine owns all state. Add a typed Component 6 capture-fence command/fact at the Component 5 raw FIFO seam; consume chunks, terminals, fence, lifecycle and publication in the Component 2 FIFO. Remove `Owner`, health store, queue polling, `deferred`, direct evaluator, and recovery-specific publication. | `P-C6-LEDGER`, `P-C6-FENCE`, `P-C6-START`, `P-C6-RECOVER`, `P-C6-INTEGRATION` |
| `internal/massive/recovery_test.go`: `TestShortDisconnectRecoveryMatchesUninterruptedControl`, `TestRecoveryProviderFailureRemainsLocalAndObservable`, `TestSecondDisconnectCancelsStaleRecoveryGeneration`, `TestRecoveryTotalDeadlineCancelsWorkAndFailsClosed` | `f5ef1fc3ef0a4ea11a6d7dfbd36d2e528d6406c530d751c1aeef5edbc58be264` | Good control-equivalence, local failure, supersession, and bounded cancellation scenarios. The overlap assertion and old lifecycle/readiness owner are not authoritative. | `behavior evidence` | Compare the exact `[T_supported,R)` join to uninterrupted canonical state; use injected retry/exhaustion facts rather than adopting 120 seconds. | `P-C6-RECOVER`, `P-C6-INTEGRATION` |
| `internal/massive/phase2a_recovery_test.go`: `TestPhase2AAugust4Recovery5500Symbols`, `TestPhase2ARecoveryFences`, `TestPhase2AOperatorCancellationIsIdempotentAndNotProviderFailure`, `TestPhase2AGlobalAndLocalFailureDomains` | `ba14c9d66956b31949dc952ac21202431560756967e90ac15adadc214ec5019a` | Establishes 5,500-symbol finite batches/accounting, stale-row retention, pre/post fence ordering, idempotent cancellation, and local/global reason cases. It also expects degraded output and a recovery-owned `Ready` stage. | `behavior evidence` | Preserve finite full-population/accounting/fence/cancel cases. Use one engine lifecycle and ordinary publication; remove `Ready`, rank-prioritized planning, and provider batch semantics from scanner state. | `P-C6-LEDGER`, `P-C6-FENCE`, `P-C6-START` |
| `internal/massive/phase2a_acceptance_remediation_test.go`: `TestPhase2AAcceptanceRemediationCurrentMarkRequiresAcceptedCurrentEpochApply`, `TestPhase2AAcceptanceRemediationBindingRejectsStaleGenerationWork`, `TestPhase2AAcceptanceRemediationTerminalSymbolFailureIsFieldLocal`, `TestPhase2AAcceptanceRemediationFinalPromotionRejectsNewDisconnect` | `912038dc1c4ad67a9104352763ae1e72844629caa0c75e135d02da3379c933ad` | Strong stale epoch/generation/binding and before/during/after-final-fence interleavings; shows a committed earlier fact may remain while stale generation cannot advance coverage/promotion. | `behavior evidence` | Re-express through closed Component 2 inputs and generation/fence disposition, without V2 hooks, health store, ranking owner, or private decoder sequence. | `P-C6-MERGE`, `P-C6-FENCE`, `P-C6-RECOVER` |
| `internal/massive/hydration_remediation_test.go`: `TestHydrationModeBudgetsProgressWatchdogAndFormer120SecondBoundary` | `c68d9d01e9c539e1fe4e4e22cbe585400f90bf85fecff57902f008712d80735f` | Records the predecessor regression in which a 120-second same-process deadline was incorrectly reused for fresh hydration, and terminally moves leftovers to `deferred`. | `behavior evidence` for the regression; reject outcomes/numbers | C6 exposes progress/terminal facts and accepts policy commands; Component 8 owns mode deadlines/watchdog. Exhaustion must leave hydration/recovery in a Phase 1 legal state and classify every open request canceled/failed/fenced, never deferred. | `P-C6-START`, `P-C6-RECOVER` |
| `internal/massive/phase2b_checkpoint_test.go`: only `TestPhase2BFreshBootstrapOwnerFourStartTimes` and `TestPhase2BFreshBootstrapFakeRESTPromotesQualified` | `869842fbcf05210c9cb5bbf254b1e4e0491ede46305342a51f62fb06de0b5775` | Useful fresh-start timing and REST/live interleaving shapes. The fake REST envelope is weaker than Component 4 and surrounding file owns checkpoint behavior. | `behavior evidence` | Adapt only the two named flows; make the fake response strict and leave all checkpoint mechanics to Component 7. | `P-C6-START`, `P-C6-INTEGRATION` |

### 8.1 Reconnaissance conclusion

V2 contains valuable adversarial interleavings, but its production recovery
shape is the architecture Phase 1 replaced. It has an outer `Owner`, separate
health/recovery/evaluation/publication authorities, rank-prioritized discovery,
duplicate binding types, direct queue polling, and terminal `deferred` state.
No production engine/recovery source is suitable for direct port.

The simplest conforming design registers one complete-population engine plan,
runs independent per-symbol REST work outside the engine, streams only sealed
result chunks and terminal facts through the existing FIFO, captures one
adapter marker through the C5 raw-frame tail, and lets the ordinary Component
2/3 path derive consequences and publish.

**Approved V2 implementation whitelist:** Only the exact named tests and
declaration roles above as behavior evidence. No V2 production source is
approved for direct copying. The V2 `Owner`, recovery state/evaluator/planner,
duplicate binding, `deferred` stages, checkpoint code, health/readiness, and
all unlisted tests/functions remain excluded. Production builds/tests may not
depend on the V2 checkout.

## 9. Detailed semantic inputs, outputs, and owned state

### 9.1 Engine-owned plan and ledger

| Item | Meaning and required provenance/identity | Bound or ownership |
| --- | --- | --- |
| Hydration purpose | Closed `fresh_bootstrap`, `checkpoint_catchup`, or `gap_recovery`. | One purpose per generation; never selects another evaluator. |
| Base interval | Fresh `[S,R)`; checkpoint `[T0,R)`; same-process `[T_supported,R)`, where `T_supported` is the committed watermark atomically retained on entry to recovery and `R` is the new acknowledgement boundary. Empty intervals are complete without REST work. | Exact whole-second half-open bounds within `[S,E]`. No V2 `T-16m` overlap and no `E+1s`. |
| Planned population | Every Component 1 binding symbol whose prior-close fact is `valid`. Missing/invalid prior-close symbols remain their Component 1/3 primary category and create no rankability/history debt. | Immutable exact-symbol-sorted population; at most Component 1's 100,000-symbol ceiling. |
| Generation | Positive checked engine-owned identifier, purpose, binding, dependent current epoch/ack, base interval, plan budgets, and generation state. | At most one active hydration/recovery generation. No reuse/wrap. |
| Request token | Binding ID, generation, positive request ID, purpose, exact symbol and interval. | Exactly one per planned symbol when base interval is nonempty. Immutable and engine allocated. |
| Active request ledger | Expected result ID/chunk/row sequence, producer progress, canonical row dispositions, coverage consequence, one terminal work bin, and bounded reason. | One entry per planned request; fixed scalar state plus current bounded chunk context. No raw response or session-long duplicate values. |
| Aggregate reconciliation pin | Exact currently retained live authority for any identity in the registered interval, held until the request/generation terminates so historical overlap cannot overwrite or outlive its evidence. | Only existing exact records in the base interval; no new session bitmap or copied value history. Registration rejects a compacted-present identity it cannot prove. |
| Generation fence | Exact current binding/generation/epoch plus a C5 adapter marker proving classification through the greatest raw frame admitted before the capture command linearized. | One pending/current terminal fence per generation attempt; no queue-empty inference. |
| Historical coverage consequence | Per-symbol exact successful intervals, conflicts/invalid rows, and unknown intervals sufficient to derive mark/no-print/history availability at `T`. | Engine owned; may reuse already-approved Component 2 presence/conflict structures but cannot add a competing canonical map. |

Plans are outcome-blind: request membership/order does not depend on raw Day %,
published rank, current top 20, qualification frontier, or feature debt.
Deterministic exact-symbol order is scheduling input only and has no mutation
authority; the FIFO orders facts.

### 9.2 Closed Component 2 input extensions

Component 6 adds these statically typed variants to the existing FIFO:

```text
hydration_chunk
hydration_terminal
hydration_progress       // diagnostic only; optional to admit
aggregate_ingress_fence
hydration_policy_action  // retry, exhaust, cancel, or stop fact from Component 8 later
```

`hydration_chunk` carries the immutable identity/count fields in the REST
detail and normalized Component 4 values. Each row is sent through Component
2's existing historical aggregate decision; the C6 transition owns the request
ledger update around that decision. `hydration_terminal` cannot complete work
until every expected preceding chunk/row has been consumed and accounted.
Provider completion and canonical coverage remain separate.

No exported lifecycle setter, generation setter, terminal counter mutation,
no-print setter, evaluator, or publication hook is added.

### 9.3 Component 5 ingress-fence extension

Component 6 extends the approved Component 5 command/fact boundary with one
narrow pair; this is the roadmap-owned ingress-fencing behavior, not a
reinterpretation of existing C5 facts:

```text
capture_aggregate_ingress_fence command
  = binding + current_epoch + generation + positive command_token

aggregate_ingress_fence fact
  = same identity + through_frame_sequence + marker causal evidence
    + captured_at + complete | failed | canceled
```

On the current epoch, the adapter linearizes capture by appending a required
zero-payload marker behind the greatest raw frame already admitted to C5's
bounded FIFO. The marker waits cancellation-aware for reserved control
capacity, exactly like the approved terminal marker. The sequential classifier
drains every earlier frame, preserves every item disposition/end-of-frame
boundary, then returns the marker fact through the engine FIFO. Frames admitted
after the marker belong to subsequent ordinary processing.

The engine considers the fence reconciled only when it consumes a matching
complete fact in FIFO order after all expected terminal work. Wrong/stale
binding, generation, epoch, token, regressing through-position, queue
saturation/ingress ambiguity, epoch loss, or cancellation cannot complete the
fence. No component polls `queue empty`, reads C5 private counters, or invents a
live frame position.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C6-PLAN-01` | On the legal engine transition, allocate one positive generation and immutable token per valid-prior-close bound symbol over the exact mode interval. Fresh is `[S,R)`, checkpoint is `[T0,R)`, and gap recovery is `[T_supported,R)`. Validate binding/epoch/ack/bounds/budgets atomically; register no interval containing compacted-present evidence unless it is split or exact authority is retained. Empty base interval terminalizes without REST. Planning is complete-population and independent of rank/features. | `PG-OPS-01`, `PG-OPS-02`, `DTE-EVENT-03`, `DTE-RECOVERY-01`, `DTE-RECOVERY-02`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-02`, `LIFE-RECOVER-03`; V2 counterexamples and Component 2 registration rule. |
| `C6-LEDGER-01` | Own one finite active ledger in the engine. Validate every chunk/progress/terminal by exact token identity and sequence; copy bounded values; assign each request exactly one of `completed_value`, `completed_empty`, `failed`, `canceled`, or `fenced`; cancellation/supersession terminalizes every open item once and fences late facts. Reconcile the Phase 1 accounting identity after every input and before generation exit. | `PG-OBS-02`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `LIFE-HYDRATE-05`, `LIFE-HYDRATE-07`, `LIFE-RECOVER-04`, `LIFE-MODEL-02`; V2 stale/cancel/accounting cases. |
| `C6-MERGE-01` | Route every historical row through Component 2's existing historical fill-only merge under its exact token/result context. Live equality/authority/conflict and historical/historical conflict have the accepted Component 2 dispositions; Component 6 records the coverage consequence without a second value map or evaluator. Provider `completed_value` may coexist with row rejection/conflict and therefore does not by itself prove complete usable historical coverage. | `DTE-MERGE-01`–`05`, `DTE-HYDRATE-02`, accepted `ENG-AGG-01`; V2 conflict/stale interleavings. |
| `C6-FENCE-01` | Add the typed capture command/marker fact above. A generation can finalize coverage only after all work is terminal and a matching current-epoch marker has reached the engine through the C5 raw FIFO/classifier and Component 2 FIFO. Epoch loss or ingress ambiguity cancels/fences the generation and cannot be relabeled as a completed fence. | `DTE-RECOVERY-03`, `ARCH-FLOW-02`, `ARCH-FLOW-03`, `LIFE-HYDRATE-05`, `LIFE-RECOVER-02`; V2 pre/post fence tests and approved C5 queue/marker semantics. |
| `C6-START-01` | After a current aggregate acknowledgement, start fresh or future checkpoint hydration while current-epoch live/control/timer inputs continue. Permit only the Phase 1 evidence-based unavailable/degraded/exact projections. Once every request is terminal and the fence/consequences/accounting reconcile, leave `hydrating` in the same ordered transition to `live`, `suppressed`, or `ended`; sparse, empty, local failure, or zero qualification never creates `deferred`. Epoch loss cancels once, fences late work, preserves accepted facts, returns to acknowledgement wait, and replans under a new generation. | `PG-RANK-05`, `PG-OPS-02`, `LIFE-HYDRATE-01`–`07`, `LIFE-T11`–`14`, `LIFE-PUBLISH-02`, `LIFE-PUBLISH-03`; V2 fresh/interleaving/regression evidence. |
| `C6-NOPRINT-01` | `completed_empty` proves only its exact historical interval. For a valid-prior-close symbol with no accepted session mark, establish/extend `no_print_through(T)` only when successful historical intervals cover from `S` through `R`, current acknowledged aggregate transport is continuous from `R` through `T`, the matching fence is reconciled, and no accepted aggregate exists. Failed/canceled/fenced/conflicted/invalid coverage remains unknown. An earlier mark survives a later empty interval; a later live aggregate immediately replaces no-print through ordinary evaluation. | `PG-RANK-02`, `PG-OBS-01`, `DTE-RECOVERY-03`–`05`, `LIFE-HYDRATE-05`, `LIFE-LIVE-03`; explicit Phase 1 no-print invariant. |
| `C6-RECOVER-01` | On aggregate coverage loss from `live`, atomically retain `T_supported`, stop currentness/advancement, close T/Q, preserve canonical state, and request a new epoch. After acknowledgement, plan `[T_supported,R)` for the complete valid-prior-close population, accept the new live tail, apply Component 8-supplied bounded retry/exhaustion actions, reconcile terminal work and one fence, then invoke the ordinary evaluator and transition to `live`, `suppressed`, or `ended`. Repeated epoch loss supersedes/cancels exactly once. No 16-minute backward overlap, recovery watermark/evaluator, or inactive terminal stage exists. | `DTE-RECOVERY-01`, `DTE-RECOVERY-03`, `DTE-RECOVERY-05`, `LIFE-RECOVER-01`–`06`, `LIFE-T16`, `LIFE-T18`–`22`, `LIFE-T26`; V2 control/supersession/final-disconnect cases. |
| `C6-INTEGRATION-01` | Compose Components 1–6 with fake socket and HTTP boundaries: one C5 acknowledgement fixes `R`; live tail and C6 historical chunks enter Component 2's FIFO; C4 mapping and Component 2 merge produce one canonical state; the C6 fence terminalizes work; Component 3 ordinary evaluation publishes the correct exact/degraded/unavailable result. Source/ownership inspection finds one mapper, engine, generation ledger, watermark, evaluator, and publication path. | `DTE-MODEL-01`–`03`, `DTE-COMMIT-02`–`04`, `LIFE-T11`, `LIFE-T12`, `LIFE-T19`, `LIFE-T20`; Components 1–5 accepted interfaces and scoped V2 fake flows. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Generation/plan | Exact installed binding, legal lifecycle/mode, current acknowledged epoch when applicable, exact `R`/`T0`/`T_supported`, complete valid-prior population, checked finite budgets, and no unprovable compacted overlap. | Atomic plan failure; no active partial ledger. Global only when required coverage cannot be represented honestly. | Rank-biased subset or wrong interval being labeled complete-population hydration. |
| Chunk/terminal fact | Exact active token/result sequence and immutable rows; terminal counts equal all consumed chunks/rows; one terminal bin still open. | Stale becomes fenced; malformed/duplicate/out-of-order fact cannot advance coverage or complete twice. | Terminal `completed_empty/value` arriving before missing chunks or from an old generation. |
| Historical canonical consequence | Existing Component 2 acceptance returns an exact disposition for every row; result context retains live authority and conflicts through terminalization. | Rejection/conflict marks only the dependent interval/field unknown unless a global invariant fails. | Provider success being equated to canonical coverage despite rejected/conflicting rows. |
| Live ingress fence | Current binding/generation/epoch/token; marker appended after a concrete raw-frame tail and consumed after every preceding classified item. | Failure/epoch loss/ambiguity cancels or fences; never queue-empty success. | Publishing no-print/current ranking while a pre-capture live frame remains unclassified. |
| Lifecycle exit | All work terminal, accounting exact, fence reconciled, consequences installed, and binding/epoch/global state coherent. | Exit to the precise Phase 1 state; no pending/deferred residue. | A sparse or failed population freezing ordinary timers after work is terminal. |

## 11. Failure and terminal behavior

Plan construction failure is atomic. After registration, every request must
receive exactly one engine terminal disposition. A stale binding/generation,
superseded epoch, cancellation, or late fact is `fenced` or `canceled` according
to the ordered event that first closed it; it is not counted again when its
producer later completes.

Malformed chunk/terminal sequencing, impossible counts, request-ID reuse,
nonmonotonic generation, repeated terminal with different evidence, or a
regressing same-epoch fence is an engine accounting/ordering integrity failure.
An attributable provider/row/canonical failure is symbol/interval local unless
it makes the global aggregate stream or complete-population claim unknowable.
The engine records provider terminal work separately from canonical coverage
consequence.

Startup epoch loss cancels all open work and returns to
`awaiting_aggregate_ack`. Recovery epoch loss cancels/fences the current
generation and remains in `recovering/awaiting_new_aggregate_ack`. Component 8
may later submit explicit retry/exhaust/cancel policy facts; C6 does not own
backoff, number of generations, production mode deadlines, or shutdown time.
Exhaustion always produces the Phase 1 `live` with provably local degradation,
`suppressed`, or `ended` disposition—never `deferred`.

At session end/stop, stop new work, cancel every open request once, capture or
explicitly fence the shutdown boundary, consume/fence admitted inputs, and let
the engine transition to `ended` without waiting indefinitely for HTTP or C5
marker completion.

## 12. Accounting and observability

The generation's primary work identity is exact after every engine input:

```text
planned
  = completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Before terminalization, `open = planned - terminal`; `open` is an internal
progress count, not a sixth terminal bin. Provider progress, chunks, rows,
pages, attempts, bytes, canonical inserted/duplicate/conflict/rejected rows,
purpose, and bounded reasons are overlapping diagnostics.

For each request:

```text
consumed_rows
  = canonical_inserted
  + canonical_duplicate
  + canonical_conflict_or_withdrawal
  + canonical_rejected
  + canonical_fenced
  + canonical_integrity_failure
```

The exact bin names reuse Component 2 dispositions; the equation is a C6
per-result view and does not increment Component 2's primary input counters a
second time.

Generation observability includes purpose, binding, generation, applicable
epoch, base interval, planned/open/terminal counts, fence state/through-frame,
provider and canonical summaries, retry-policy wait/action, and fixed terminal
reason. Retain no symbol list in snapshots/log labels, raw payloads, URLs,
credentials, per-row diagnostics, arbitrary errors, or generation history.

## 13. Simplicity and boundedness

- One engine-owned generation and one flat request ledger replace V2's owner,
  health store, planner batches, field-debt graph, recovery evaluator, stages,
  and publication path.
- Planning all valid-prior symbols once is simpler and more honest than
  repeatedly reprioritizing from ranks/features. The worker pool supplies
  bounded concurrency without making batches semantic state.
- `T_supported` is the same committed watermark retained at recovery entry.
  Requesting `[T_supported,R)` safely covers the unsupported join and any small
  uncommitted live overlap while avoiding V2's unnecessary 16-minute backward
  range and Component 2 compacted-presence conflict.
- Existing exact live records in the interval are pinned only until the active
  request/generation terminates. No third session bitmap, duplicate canonical
  map, raw history, or recovery-only marks are introduced.
- One required marker in C5's existing raw FIFO is the minimum real ingress
  fence. There is no queue polling, periodic fence loop, or second live queue.
- A ledger entry is fixed scalar state; chunk contents live only during their
  transition. Maximum requests equal valid-prior population, maximum open
  generation is one, and retained completed-entry detail is compacted to the
  terminal bin/coverage summary immediately after the generation exits.
- Component 8 owns production budgets/retries/deadlines; Component 7 owns
  checkpoint state. C6 adds no policy defaults, generic workflow framework,
  database, journal, service, or plugin mechanism.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Pre-session acknowledged epoch yields `R=S` | Phase 1 `LIFE-INIT-04`; V2 incorrect forced second | Empty `[S,S)` plan terminalizes without provider calls; marker/lifecycle proceeds legally. | `P-C6-START` |
| Mid-session fresh start with quiet universe | Phase 1 lifecycle scenario 2/3; V2 fresh tests | Plan every valid-prior symbol; value/empty/failure are distinct; terminal work exits hydration. | `P-C6-START`, `P-C6-NOPRINT` |
| 5,500-symbol completion under worker permutations | V2 August 4 test | Flat plan/accounting remains exact, no all-universe result object/batch authority, no rank-biased omission, bounded resident state. | `P-C6-LEDGER` |
| Old binding/generation chunk or terminal | V2 remediation tests; Component 2 stale context invariant | Fence once before canonical/coverage/lifecycle mutation; later producer completion cannot double-terminal. | `P-C6-LEDGER` |
| Live frame admitted before capture but classified after REST completes | V2 pre/post fence test; DTE fence invariant | Marker waits behind it; generation cannot finalize until its item disposition precedes the marker fact in engine FIFO. | `P-C6-FENCE` |
| Live frame admitted after capture | Phase 1 `DTE-RECOVERY-03` | It remains ordinary later processing and is not required by the completed hydration fence. | `P-C6-FENCE` |
| New disconnect before/during/after final fence | V2 final-promotion interleavings | Earlier canonical facts may remain, but old generation cannot advance coverage/currentness; cancel/fence and replan new epoch. | `P-C6-RECOVER` |
| Successful empty with no mark versus with earlier mark | Phase 1 no-print proof | First can become no-print only with full live coverage/fence; second retains earlier mark and records empty later coverage. | `P-C6-NOPRINT` |
| Historical/live equality or conflict and conflicting historical rows | Component 2 proof plus V2 merge tests | Equal deduplicates; live wins conflict; dependent coverage unknown; historical arrival order never wins. | `P-C6-MERGE` |
| Provider terminal failure for one symbol | V2 local failure/remediation | Work terminates failed; independent current mark may remain; affected completeness/fields are unknown; ordinary exit still occurs when all work terminal. | `P-C6-MERGE`, `P-C6-START` |
| Same-process short gap versus uninterrupted control | V2 control test, adapted to Phase 1 interval | `[T_supported,R)` plus live tail yields equivalent canonical/evaluator result when provider evidence matches; no recovery evaluator. | `P-C6-RECOVER`, `P-C6-INTEGRATION` |
| Retry/supersession/exhaustion | V2 second disconnect/deadline tests; Phase 1 recovery | One terminal per old request, late facts fenced, policy fact schedules finite next action or exits explicitly. | `P-C6-RECOVER` |

Primary-proof allocation and slice acceptance are authoritative only in the
[proof and delivery plan](proof-and-delivery-plan.md).
