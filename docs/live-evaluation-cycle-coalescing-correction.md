# Live evaluation-cycle coalescing correction

**Status:** Step 1 offline complete on 2026-08-19. Live stability remains
unconfirmed pending the single combined market-hours soak after all three
scanner-stability corrections.

**Development constraint:** The market is closed. Complete and accept this
change from deterministic production-shape evidence without credentials or
provider requests. Live stability remains explicitly unconfirmed until the
single combined market-hours soak after all three corrections.

**Implementer starting context:** Read this specification, the
[`live feature-set MVP program`](live-feature-mvp-program.md), and the accepted
[`live fence finalization correction`](live-fence-finalization-and-burst-capacity-correction.md).
Preserve unrelated worktree changes and use one write-capable implementation
agent.

## Outcome

In ordinary `live` operation, one scheduler iteration must not perform the
same full-universe aggregate evaluation twice merely because the runtime first
completed a live-coverage fence and then admitted its normal timer.

An accepted live-coverage fence is the ordinary aggregate evaluation boundary.
The following timer still advances non-aggregate time-dependent work, but is
maintenance-only for aggregate evaluation. A full timer evaluation remains
available when no fence evaluated that scheduler iteration or when an explicit
same-target deadline/session-end condition requires it.

This removes redundant O(universe) work and its associated allocation/GC burst
without changing the committed watermark, coverage proof, qualification,
ranking, or single-engine ownership model.

## Why this is the first correction

The current one-second runtime path is serial:

```text
capture live-coverage fence
  -> wait for engine completion
  -> admit timer
  -> wait for engine completion
```

Both an applied `inputLiveCoverageFence` and an applied `inputTimer` are
currently eligible to call `stageAggregateEvaluationAtLocked`. A supported
same-target timer deliberately remains valid for strict correction-horizon
finalization. That rule is correct in isolation, but it means the normal
post-fence timer can repeat the full roughly 5,700-symbol pass even when the
fence already evaluated the same canonical prefix and no new deadline requires
another pass.

The observed live symptoms are consistent with this deadline miss: the
watermark repeatedly moves between two and three seconds behind the causal
target, T/Q pressure is selected by oldest waiting frame despite low queue
occupancy, and the provider connection remains active. Prior retained-tail
proofs measured one isolated full pass; they did not exercise the production
fence-plus-timer cycle under ordinary GC.

## Fixed boundaries

Preserve all of the following:

- one `ScannerStateEngine`, canonical symbol state, committed aggregate
  watermark, evaluator, and immutable publication;
- the exact live ingress fence and per-symbol coverage-extension semantics;
- accepted aggregate/correction mutation in engine sequence order;
- exact qualification finalization strictly beyond the correction horizon;
- timer-driven quiet-window, mark-age, field-state, T/Q, lifecycle, and
  session-end behavior;
- same-target reevaluation when genuinely required to expose a correction or
  finalize state and no successful fence already supplied that evaluation;
- replay behavior and checkpoint-off production operation; and
- readiness tolerance. This correction removes work; it does not hide lag by
  increasing the tolerance or evaluation delay.

The controlling meanings are `PG-RANK-03` through `PG-RANK-05`, `PG-OPS-02`,
`PG-OBS-03`, `ARCH-OWN-01` through `ARCH-OWN-04`, `DTE-TIMER-01`,
`DTE-COMMIT-01` through `DTE-COMMIT-04`, and `LIFE-LIVE-02` through
`LIFE-LIVE-05`.

## Required design

### 1. One aggregate projection opportunity per ordinary cycle

The runtime must distinguish these two timer policies without adding another
state owner or event clock:

- **evaluation timer:** may run the existing aggregate target/evaluator gate;
- **maintenance-only timer:** consumes the same ordered timer fact and runs all
  non-aggregate timer behavior, but cannot stage a full-universe aggregate
  evaluation.

The simplest expected implementation is one private policy bit or closed enum
on the existing timer input, with two engine admission methods if that makes
misuse harder. Do not add a second timer event kind, evaluator, goroutine-owned
market state, or runtime-owned watermark decision.

For each runtime evaluation scheduler iteration:

1. Attempt the existing live-coverage fence.
2. If the fence disposition is `live_coverage_fence_applied`, admit a
   maintenance-only timer.
3. If no fence was issuable, the fence was canceled/fenced/rejected, or the
   runtime is outside ordinary live fencing, admit an evaluation timer.
4. The runtime must wait for the chosen ordered facts exactly as it does now;
   it must not race the fence and timer.

An applied fence remains eligible for exactly one full aggregate stage even
when its target equals the committed watermark, because a correction or a
strict engine-time finalization may change the same-`T` result. Its immediately
following maintenance timer must not stage again.

### 2. Deferred post-fence facts remain honest

Aggregates admitted after the fence marker are outside the publication's
supporting ingress prefix. If they arrive before the maintenance timer, their
canonical mutations and accounting remain synchronous, but their pending
aggregate projection may wait for the next accepted fence. The maintenance
timer must not consume that pending flag.

This is not silent loss or false currentness: the already-published snapshot is
bound to the earlier fence, and the later facts are still pending in canonical
state. The next fence must consume them. A terminal/session-end fallback must
flush pending same-target work when no later ordinary fence can occur.

### 3. Timer fallback is explicit, not periodic rescanning

An evaluation timer may stage only when the existing support rules hold and at
least one of these conditions is true:

- a later supported target can advance `T` and no applied live fence already
  evaluated this scheduler iteration;
- aggregate projection is pending at the committed target and no ordinary live
  fence is available to consume it;
- an engine-owned strict same-target deadline can change qualification or
  another aggregate-derived result; or
- session-end/final evaluation must flush the last supported boundary.

Do not preserve a generic rule that every supported same-target timer performs
a full scan. The engine may track the earliest same-target maintenance deadline
while it is already scanning the universe; it must not scan the universe merely
to discover whether a scan is due.

### 4. Missed cadence ticks are coalesced

If one evaluation cycle runs longer than its cadence, the runtime must not run
one immediate catch-up cycle for every missed tick. After completion, schedule
the next future cadence boundary and use the newest then-current causal target.
Intermediate missed wall-clock ticks create no market facts and may be skipped.

Keep pressure sampling independent and bounded. This change does not weaken or
rescale T/Q pressure thresholds.

### 5. Fixed-cardinality evidence

Expose enough internal/test observation to count aggregate evaluation starts by
source (`live_coverage_fence`, `timer`, `aggregate_ingress_fence`, `replay`) and
target. Reuse the existing evaluation timing view where practical. Do not add
symbol labels, event payloads, an unbounded history, or a required API-v2 field.

## Implementation boundary

Expected production changes are limited to:

- `internal/operations/runtime.go` and focused runtime tests;
- `internal/engine/engine.go`, `feature_price_range.go`, and the smallest
  timer/evaluation state needed to enforce the policy;
- `internal/engine/evaluation_coalescing_test.go` and production-composition
  or retained-tail capacity tests; and
- this document's acceptance record.

Do not change provider transport, hydration coverage meaning, aggregate merge
rules, feature formulas, T/Q pressure constants, readiness thresholds, API/UI,
checkpoint/replay claims, queue capacity, or process supervision.

## Required deterministic proofs

### P1 — fence plus timer performs one full pass

Drive the real runtime/engine composition through at least 60 advancing
one-second cycles at the existing retained-tail production population. In each
cycle, complete a real live-coverage fence and its following timer.

Prove:

- exactly one full-universe aggregate stage per cycle;
- the source is the accepted live fence;
- the timer still performs T/Q/time/lifecycle maintenance;
- watermark and evaluation target are exact and nondecreasing;
- no admitted aggregate, work accounting, or publication fence is lost; and
- no immediate catch-up burst occurs after an intentionally delayed cycle.

Record stage, apply, publication, end-to-end cycle time, allocations, heap
before/after, queue high-water, and maximum oldest-frame age. On the current
development host, require evaluator/cycle p99 below 750 ms and maximum below
one second for this deterministic fixture. This is a local regression bound,
not a provider or production SLA.

### P2 — post-fence aggregate waits for the next fence

Apply fence F1, admit a valid aggregate/correction after F1's marker, then run
the maintenance timer and fence F2.

Prove the maintenance timer performs no aggregate stage and does not consume
pending projection; F2 evaluates the mutation exactly once, preserves engine
sequence/fence identity, and publishes the same canonical result as a direct
single evaluation at F2.

### P3 — fallback and finalization remain live

Cover all of these separately:

- no live attempt/fence available: evaluation timer advances a supported
  target;
- canceled/rejected/fenced live fence: timer retains the existing safe
  fallback without fabricating coverage;
- supported same-target qualification proof becomes final strictly after the
  correction horizon without requiring target advancement;
- pending correction at session end is flushed exactly once; and
- quiet/no-print time still changes the appropriate aggregate and T/Q field
  states.

The existing `supported same-target timer preserves strict correction-horizon
finalization` regression must remain meaningful; do not delete or weaken it to
make coalescing pass.

### P4 — semantic differential

For a small exact trace containing insertion, revision, withdrawal, no-print
coverage, qualification transition, quiet seconds, and session end, compare
the corrected coalesced path with an oracle that performs one evaluation from
the same canonical state at every required target. Assert identical committed
watermarks, qualification state, rows/order, feature states/reasons, population
accounting, T/Q desired membership, and final publication meaning. Publication
IDs may differ only by the removed redundant replacements and must remain
strictly increasing.

## Verification and acceptance

During correction, run the narrow engine/runtime tests first. Final offline
acceptance requires:

```text
go test -count=1 -short -timeout 2m ./internal/engine ./internal/operations
go test -count=1 -short -timeout 2m ./...
go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations
go vet ./internal/engine ./internal/operations
git diff --check
```

Run the 60-cycle production-shape proof separately with an explicit timeout no
greater than 15 minutes. Record the exact fixture population, retained-tail
shape, Go version, host, cycle distribution, evaluation count, allocations,
heap, and limitation.

The step is **offline complete** when P1-P4 and the affected verification pass,
one read-only concurrency/ordering review is clean, and the code walkthrough
shows no path from an applied fence to a redundant same-cycle timer scan.

Offline completion does not claim the scanner is live-stable. Do not run a
provider test after this step; proceed directly to step 2, preserving the
market-hours test for the combined three-change build.

## Explicit non-scope

- incremental/dirty-symbol evaluation;
- parallel or sharded evaluators;
- changing the 4-second evaluation delay or 2-second readiness tolerance;
- increasing queues or hydration workers;
- T/Q publication coalescing, which is step 2;
- snapshot/API/UI isolation, which is step 3;
- replay or checkpoint repair; and
- hosting or public deployment.

## Step 1 offline acceptance record — 2026-08-19

The runtime now waits for the live-coverage fence disposition and admits a
maintenance-only timer only after an applied fence. Any absent, canceled,
rejected, or fenced fence selects the ordinary evaluation timer. Applied live
and hydration-ingress fences retain one full-universe evaluation opportunity;
post-fence facts remain pending for the next fence. The timer retains canonical
compaction, T/Q/time/quiet-window/lifecycle work, supported later-target and
pending same-target fallback, strict-after qualification finalization, and the
one terminal same-target evaluation needed to expose session end or flush
pending work. The scheduler uses a one-shot cadence timer and advances to the
next future boundary after delay; it never replays missed ticks.

Evaluation observation is fixed cardinality: the latest target/source and four
counters for `live_coverage_fence`, `timer`, `aggregate_ingress_fence`, and
`replay`. No event history or second evaluation owner was added. The capacity
correction also removed two costs exposed by the required retained-tail proof:
the three nested price-range windows now share one low pass and one high pass,
and the existing symbol-owned MVP measurement state maintains an ordered,
cumulative derived tail index. The canonical aggregate map remains the sole
mutable aggregate record. Insert/revision, withdrawal, compaction, and
checkpoint reconstruction maintain or rebuild both derived views.

### Deterministic proofs

- **P1 — sealed production-shape runtime:**
  `CACHED_HYDRATION_ACCEPTANCE=1 go test -count=1 -run
  '^TestCachedHydrationFenceAcceptance$' -timeout 15m -v
  ./internal/operations` passed in 276.705 seconds. The local sealed artifact
  was
  `sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`
  at 2,584,011,150 bytes. Its binding contained 5,691 symbols; the selected
  retained-tail prefix contained 7,581,690 rows across 5,502 symbols (5,439
  value and 63 explicit empty/no-print terminals) over
  `[2026-08-07T08:00:00Z, 2026-08-07T21:15:00Z)`. The real local adapter and
  runtime accounted for 26,125/26,125/26,125/26,125 frames
  sent/read/admitted/dispositioned with zero rejection. Sixty automatic
  advancing cycles produced exactly
  `{LiveCoverageFence:60 Timer:0 AggregateIngressFence:0 Replay:0}` full
  evaluation starts. End-to-end p99 was 461.9685 ms and maximum was
  629.175458 ms. Stage ranged from 109.205958 ms to 455.486917 ms, apply from
  12.643708 ms to 29.563583 ms, and publication from 10.167 microseconds to
  84.75 microseconds. Per-cycle allocation ranged from 22,924,088 to
  29,015,328 bytes (median 28,818,592; 1,587,324,616 total). Heap was
  16,533,152 bytes before the run, 2,765,244,192 at the hydration fence, and
  1,607,765,608 after final GC. Queue high-water was 81 frames and maximum
  oldest-frame age was zero. The separate 5,694-symbol by 961-record
  retained-tail evaluator proof (5,471,934 canonical tail records) passed 60
  advancing seconds with 276.078961 ms mean and 409.990375 ms maximum. The
  missed-cadence regression proves a cycle delayed to `anchor+3.25s` schedules
  at the next future boundary after 750 ms rather than immediately catching
  up.
- **P2 — pending post-fence work:**
  `TestLiveAggregateEvaluationCoalescing/post-fence_correction_remains_pending_through_maintenance_and_is_consumed_by_next_fence`
  proves F1 evaluation, a post-marker correction, no maintenance-timer stage
  or pending-work consumption, and exactly one F2 evaluation with the revised
  canonical close and preserved fence/source counters.
- **P3 — fallback/finalization:** focused engine and runtime proofs cover the
  no-attempt evaluation-timer path; rejected cancellation and fenced stale
  fence commands followed by same-target pending fallback; strict-after
  engine-owned qualification deadline; recovering-lifecycle same-target
  invalidation; session-end maintenance evaluation and exactly-once pending
  flush; and quiet/no-print advancement. No fallback fabricates coverage.
- **P4 — semantic differential:**
  `TestEvaluationCoalescingSemanticDifferential` covers historical
  withdrawal/conflict, live insertion and revision, exact no-print coverage,
  strict qualification finalization, quiet seconds, and session end. At every
  required target the coalesced result equals a direct staged oracle for the
  committed watermark, qualification, rows/order, feature status/reason,
  population and uncertainty accounting, T/Q desired membership, and final
  publication meaning. Publication IDs remain strictly increasing; only the
  redundant replacements are absent.

### Verification and review

Final verification on Go 1.26.5, macOS 26.2 (`darwin/arm64`), Apple M1,
8 GiB RAM:

```text
PASS  go test -count=1 -short -timeout 2m ./internal/engine ./internal/operations
PASS  go test -count=1 -short -timeout 2m ./...
PASS  go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations
PASS  go vet ./internal/engine ./internal/operations
PASS  git diff --check
```

The required independent read-only concurrency/ordering review used
`gpt-5.6-sol` with medium reasoning and returned **CLEAN — no P1/P2
findings**. It confirmed the serial fence disposition before timer selection,
maintenance-only exclusion and terminal exception, supported fallback and
strict deadline ordering, next-future cadence scheduling, replay preservation,
fixed-cardinality observation, and derived-index ownership/rebuild paths.

All acceptance was deterministic and offline. The adapter endpoint was a local
test server; no provider credentials were read and no live provider request was
made. This evidence does not establish market-hours stability, provider
latency, replay repair, or checkpoint compatibility beyond rebuilding the new
nonpersisted derived index. Unrelated worktree changes were preserved.
