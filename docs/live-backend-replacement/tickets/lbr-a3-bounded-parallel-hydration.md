# LBR-A3 implementation assignment — bounded parallel live hydration

**Status:** Owner-directed implementation assignment, added 2026-08-24 and
inactive. Activation occurs only through the
[delivery program](../delivery-program.md); this file is not a mutable ledger.

**Activation gate:** A1/A2 and Capabilities B–C retain their accepted evidence.
The [delivery program](../delivery-program.md) must mark only `LBR-A3` active
after its mandatory pre-assignment audit. D1 remains inactive until A3 is
accepted.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the complete
[canonical/hydration contract](../canonical-state-and-hydration.md), accepted
A2 handoff, accepted B/C handoffs, and every production hydration constructor,
consumer, launcher, CLI, budget, cancellation, result-admission, and fence
path before activation.

This ticket implements only `LBR-A3` and
`P-LBR-A3-PARALLEL-HYDRATION`. Controlling requirements are `LBR-ARCH-01`,
`LBR-ARCH-02`, `LBR-ARCH-06`, `PG-OPS-01`, `PG-OPS-02`, `PG-OBS-02`,
`DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `DTE-RECOVERY-01` through
`DTE-RECOVERY-05`, and `LIFE-HYDRATE-01` through `LIFE-HYDRATE-07`. A1/A2
canonical, merge, generation, terminal, recovery, fence, and currentness
meanings remain mandatory and unchanged.

## 2. Outcome, owner, scope, and interfaces

The ordinary private scanner accepts exactly `1`, `2`, `4`, or `8` live REST
hydration workers and defaults to `8`, restoring the previously accepted
bounded interface. Workers concurrently perform blocking per-symbol REST
acquisition and normalization, then return immutable bounded chunks and one
terminal fact per request to the existing engine admission seam.

The `ScannerStateEngine` remains the sole generation/request-ledger, canonical
state, coverage, lifecycle, fence, currentness, ranking, T/Q, and publication
owner. Worker concurrency creates no mutable symbol alias, alternate merge,
completion owner, market-event queue, or readiness decision.

## 3. Allowed implementation boundary

Allowed production changes are the existing live hydration plan constructor
under `internal/massive`, worker-count validation and result composition under
`internal/operations`, `cmd/scanner` worker/budget flags, the private launcher,
wrapper script, active runbook/README, and focused tests. Reuse the current
worker pool; do not build another pool, scheduler, result bus, or state layer.

Do not change canonical representation, product formulas, aggregate/TQ ingress,
API/UI semantics, reference resolution, page/retry policy, queue sizes,
connection recovery, replay, checkpoints, D1/D2, or E1/E2 behavior except the
worker-count configuration consumed by their existing manifests.

## 4. Evidence and source whitelist

Permitted current-repository evidence is:

- baseline `0d043c1` worker validation and budget wiring in `cmd/scanner`,
  `internal/operations`, `internal/massive`, `internal/privatelauncher`, and
  `scripts/run-private-scanner`;
- current `HydrationWorker.Run`, `NewHydrationWorkerPlan`, A2 admission/fence
  paths, and their directly named tests;
- [`live-rest-hydration-progression.md`](../../live-rest-hydration-progression.md)
  for deterministic concurrent-live acceptance conditions;
- [`private-live-scanner-operational-finalization.md`](../../private-live-scanner-operational-finalization.md)
  for the formerly accepted `1|2|4|8`, default-8 launcher surface; and
- the bounded 2026-08-24 observation recorded in the delivery program solely
  as motivation and limitation evidence.

The predecessor checkout whitelist remains empty. No credential access or
provider request is authorized by this ticket.

## 5. Required implementation behavior and bounds

- Wrapper, launcher, scanner help/validation, live components, and worker-plan
  construction accept exactly `1|2|4|8`; default is `8`; invalid counts fail
  before credential access or runtime startup.
- At most the configured count performs blocking acquisition. One work item is
  owned by one worker at a time; results carry the original binding,
  generation, request token, symbol, interval, chunk ordinal, and terminal
  identity.
- The existing at-most-two-page, at-most-three-attempt, per-page/request row/
  byte limits remain unchanged. The plan-wide normalized-record and 4 GiB
  cumulative response bounds remain population/interval bounds. Maximum
  resident normalized records is exactly `workers * 57,600` and is validated
  without overflow before work starts.
- Out-of-order chunks and terminals are serialized only by the engine owner.
  Every planned request becomes terminal exactly once; worker completion order
  cannot alter merge precedence, accounting, or symbol outcome.
- The subscribed live tail remains actively consumed during hydration.
  Hydration result pressure is cancellation-aware and bounded; workers cannot
  create a completion backlog that stalls live aggregate consumption.
- Fence creation remains impossible until every exact request token is
  terminal and work/row accounting reconciles. The fence remains behind all
  already-read work and currentness still requires fence acceptance plus one
  ordinary evaluation.
- Cancellation, epoch/generation replacement, terminal engine state, or
  shutdown stops new scheduling, contains late results, cancels provider work,
  and joins every worker. No goroutine, request, chunk, or terminal waiter may
  survive the live attempt.
- Accepted A1/A2 canonical projections and B/C ranking, publication, T/Q
  membership/trust, API bytes, and aggregate independence are identical for
  one and multiple workers.

## 6. Primary proof and acceptance distinction

`P-LBR-A3-PARALLEL-HYDRATION` runs the ordinary production composition with
`1`, `2`, `4`, and `8` workers against one deterministic latency-controlled
REST fixture while a real fake WebSocket advances mixed live aggregates. A
barrier proves exactly the configured active-worker ceiling rather than
inferring concurrency from elapsed time; unequal releases force result and
terminal order to differ from plan order.

The proof observes exact work/row/terminal identities, bounded resident and
cumulative budgets, one canonical projection, live frames continuing without
sustained queue growth or rejection, fence placement only after all terminals,
one current publication, exact B/C/API equivalence, and joined cancellation at
every supported count. It composes wrapper through launcher through scanner
validation so default 8 and every explicit supported value reach the live plan
exactly once.

Dangerous counterexamples are an early fence after the first worker finishes,
a duplicated/missing terminal under out-of-order completion, retained mutable
result aliases, worker-owned merge/currentness, an unbounded completed-result
backlog, or parallel REST work starving the sole live consumer. This proof
does not establish Massive speedup, provider rate-limit behavior, market-hours
readiness time, whole-process E2 plateaus, or a provider SLA.

## 7. Verification and timeout policy

Run the primary proof first, then direct worker/plan/budget/cancellation tests,
affected engine/Massive/operations/scanner/launcher short suites, and affected
race suites under five minutes. Run focused vet, `git diff --check`, wrapper
argument tests, and exact ordinary `go test -count=1 -short -timeout 2m ./...`
on final bytes. Reuse unchanged A1–C2 and expensive evidence. No live/provider
tier is authorized.

Because this restores concurrent production acquisition across cancellation,
terminal accounting, and fence eligibility, one final read-only focused review
is required before A3 acceptance. It reviews only worker ownership, bounds,
out-of-order completion, live-consumer progress, join behavior, flag/default
wiring, and absence of B/C semantic change.

## 8. Removal and handoff

Acceptance supersedes A2's one-worker-only live constructor and validation
surface, not its state or recovery behavior. Handoff records supported/default
counts, exact budgets, maximum observed active workers, terminal/fence/live-tail
proof results, cancellation/join result, resource limitation, and the unchanged
D1 input interface. D1 becomes next but remains inactive until its own audit.

## 9. Implementer discretion and prohibited changes

Worker scheduling order, private bounded channels, and deterministic barrier
mechanics are delegated. Prefer the current worker-pool implementation and the
smallest configuration restoration. Do not parallelize engine mutation,
introduce per-worker canonical state, change REST/live precedence, increase
queue or response limits, add automatic worker tuning, infer provider behavior,
or combine A3 with ingress replacement.

## 10. Containment, review, and correction

One engine owner, one generation ledger, finite worker/resident bounds, and one
terminal identity per request should be construction guarantees. Runtime
validation contains invalid counts, foreign tokens, duplicate/late results,
budget exhaustion, cancellation, and generation replacement. Any path from
partial worker completion to false fence/currentness, or any concurrency that
can freeze the live consumer, triggers the correction loop and reopens only
A3. The orchestrator alone activates, records acceptance, stages, and commits
after all workers/reviewers are quiescent.
