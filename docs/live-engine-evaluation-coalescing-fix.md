# Live engine evaluation coalescing fix

## Outcome

Make the existing full-universe live scanner keep up with the deterministic T1
envelope before adding REST hydration load.

The change is narrow: accepted aggregate inputs continue through the ordinary
C5-to-engine path and mutate canonical state synchronously, but they no longer
run the 6,000-symbol qualification/ranking projection one time per aggregate.
The existing one-second timer and hydration ingress fence become the only live
boundaries that run and publish a full aggregate evaluation.

This is an engine scheduling correction, not a ranking redesign.

## Current defect

For every inserted, revised, or withdrawn aggregate after `committedT` exists,
the current path is:

```text
Engine.transition
  -> runAggregateFeatureContributorLocked
  -> stageAggregateEvaluationAtLocked
  -> runAggregateEvaluatorLocked
  -> applyAggregateCandidateLocked
```

Both staging and apply walk the bound population and advance qualification
state. In the paced 6,000-symbol reproducer this leaves the engine lock occupied
long enough that the backend processes about 154 of the incoming 287 aggregates
per second. The control reaches 408 queued frames and a 3.190-second oldest
frame without REST hydration or the legacy diagnostic probe.

The CPU profile attributes 88.4% of control CPU to the engine transition stack,
with 9.54 seconds in `evaluateQualificationThrough`. Queue size, WebSocket
decoding, REST concurrency, and the 10-millisecond probe are not the primary
cause.

## Minimal design

### 1. Live aggregate transitions stay synchronous and local

In live mode, an accepted aggregate still completes all work required to make
its canonical fact durable in engine memory before its disposition returns:

- canonical insert/revision/withdrawal and exact accounting;
- live position and correction precedence;
- the affected symbol's bounded price-range, Activity, and qualification
  maintenance, including dirty-proof marking; and
- integrity/suppression handling.

It must not call `stageAggregateEvaluationAtLocked`,
`stageAggregateEvaluationLocked`, or `applyAggregateCandidateLocked`, and it
must not walk `binding.symbols`.

Set one engine-owned boolean such as `aggregateProjectionPending` whenever an
aggregate transition changes canonical state or aggregate accounting that the
next snapshot must expose. Repeated aggregates only leave the bit set; they do
not allocate a batch, queue, map, or second representation.

### 2. Timer and ingress fence consume the pending work

Keep the existing production timer cadence of one second. On an accepted timer
or aggregate ingress fence:

1. choose the later `latestTarget` when the existing central support gate says
   it is valid;
2. otherwise, when aggregate projection work is pending and `committedT`
   exists, evaluate the current `committedT` so a late correction or hydration
   row behind the watermark is reflected without falsely advancing time;
3. stage, validate, and apply exactly one full-population candidate at that
   chosen `T`;
4. publish one coherent snapshot and clear the pending bit only after the
   candidate succeeds; and
5. retain the pending bit if staging, validation, or support prevents a
   coherent apply.

The existing ingress fence remains the startup boundary: all historical rows
and the subscribed live tail ahead of the fence are already ordered before it,
so one fence evaluation must include all of them.

Replay keeps its existing replay-group evaluation boundary. Do not change
replay behavior as part of this live fix.

### 3. Coalesce aggregate-driven publication

Do not replace the private market snapshot once per aggregate with rows from an
older evaluation. Aggregate dispositions and internal operational counters may
advance, but the immutable market snapshot stays at its prior coherent engine
sequence until the timer/fence consumes pending projection work.

At that boundary, publish once even when the recomputed rows are equal, so the
snapshot's engine sequence and accounting catch up to the accepted aggregate
prefix. The publication must continue to satisfy
`aggregateEvaluation.at == committedT` and must never claim that an unsupported
later target is current.

## Expected code boundary

The implementation should remain inside these existing seams:

- `internal/engine/engine.go`: one pending bit and aggregate publication
  coalescing;
- `internal/engine/feature_price_range.go`:
  `runAggregateFeatureContributorLocked` keeps affected-symbol maintenance but
  stops constructing a global candidate for `inputAggregate`;
- `internal/engine/evaluator.go`: remove `inputAggregate` as a global evaluator
  trigger in live mode, preserve replay behavior, and let timer/fence select
  later-`T` or same-`T` evaluation;
- focused engine tests for coalescing, correction visibility, and atomic
  publication; and
- `internal/operations/live_backend_contention_reproducer_test.go`: make the
  unprobed control fail unless it returns `kept_up`.

Do not introduce a new package, goroutine, queue, scheduler, evaluator, ranking
index, or public configuration option.

## Required proofs

### Semantic proof

Use a compact engine test with multiple bound symbols and many aggregate
changes between two timer ticks. Include:

- inserts after current `T`;
- a correction to an identity before current `T`;
- an unknown-symbol rejection; and
- an aggregate ingress-fence case representing hydration completion.

Before the next timer/fence, assert that every disposition and canonical fact
is present but no aggregate transition ran or published a full-population
candidate. After the boundary, assert:

- one coherent evaluation/publication was produced;
- the same-`T` correction is visible when later-`T` support is unavailable;
- the later target advances only when the existing support gate permits it;
- ranking, qualification, feature fields, population accounting, watermark,
  and engine sequence match the test's explicit expected final state; and
- the pending bit clears on success and survives a rejected/invalid candidate.

This proof should demonstrate behavior, not add production evaluation counters.
Publication IDs/counters may be used to show that hundreds of aggregate inputs
produce zero aggregate-driven snapshot replacements and one replacement at the
timer/fence.

### Throughput proof

Run the existing unprobed `TestLiveBackendContentionReproducer/control` through
the actual fake WebSocket, C5 queue, `LiveAttempt`, engine, timer, publication,
and accounting paths. Acceptance requires:

- all 882 frames sent, read, admitted, and dispositioned;
- all 1,764 aggregates consumed: 1,274 inserted and 490 rejected;
- no frame rejection, fencing, integrity terminal, or 384-frame safety stop;
- final queue depth zero and the admitted tail drained within 250 milliseconds;
- maximum oldest-frame age below two seconds;
- reconciled adapter, queue, engine, aggregate, transition, and publication
  accounting; and
- `kept_up` as an asserted test outcome, not merely a logged diagnostic value.

Capture one CPU and blocking profile for this post-change control. The
per-aggregate critical path must no longer contain the full-population
qualification stack. A full evaluation on a timer or fence is expected.

If this exact control still saturates, do not add parallel mutation, increase
the queue, or redesign ranking. Use the new control profile to optimize only
the remaining hottest function inside the same timer/fence evaluation boundary
until this acceptance passes.

## Verification

Run only:

```text
go test -short -timeout 2m ./internal/engine ./internal/operations ./cmd/scanner

go test -count=1 \
  -run '^TestLiveBackendContentionReproducer$/^control$' \
  -timeout 2m \
  -cpuprofile var/live-backend-contention/post-fix-control/cpu.out \
  -blockprofile var/live-backend-contention/post-fix-control/block.out \
  ./internal/operations

go tool pprof -top var/live-backend-contention/post-fix-control/cpu.out
go tool pprof -top var/live-backend-contention/post-fix-control/block.out

go test -race -count=1 \
  -run '^(TestLiveAggregateEvaluationCoalescing|TestLiveHydrationThroughputHarness)$' \
  -timeout 2m ./internal/engine ./internal/operations

git diff --check
```

Do not use the full wall-clock reproducer as a race workload. Run the compact
new semantic proof and the existing fake-source cleanup proof under `-race`
instead.

## Non-scope

Do not change C5 normalization, queue limits, REST worker count, hydration
request behavior, qualification or ranking formulas, the four-second target,
UI polling, API schema, replay, checkpoints, or provider configuration. Do not
run Massive or access credentials. Live-plus-REST testing begins only after the
deterministic unprobed control passes.

## Completion decision

This change is complete when the semantic proof passes and the unprobed paced
control is `kept_up`. The next task is then the already-defined fake REST
progression: live plus one hydration worker, followed by a bounded concurrency
sweep only if one worker remains stable.
