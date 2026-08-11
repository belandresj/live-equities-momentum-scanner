# Live backend contention reproducer

**Purpose:** Determine whether the live-only backlog observed on 2026-08-10
was caused by the scanner pipeline or by the diagnostic's own 10-millisecond
metrics probe. This is a focused test specification, not a component contract,
production redesign, benchmark, or release gate.

## Question

Can the ordinary C5 adapter-to-engine path consume a deterministic fake
WebSocket stream shaped like the failed T1 trial when it runs:

1. without the diagnostic probe; and
2. with the exact heavyweight probe that T1 ran every 10 milliseconds?

Do not optimize the normalizer, admission path, engine, or publication cadence
before this comparison. The test exists to reproduce and localize the failure,
not to prove a proposed fix.

## Why this test is needed

The two existing results conflict:

- Live T1 received 882 frames in 6.140 seconds, dispositioned 592, queued 290,
  reached a 2.089-second oldest-frame age, and used 5.15 average CPU cores.
- The deterministic cost-attribution test sent the same approximate density
  through the real C5 queue, `LiveAttempt`, engine, feature, publication, and
  validation path at 16,905 frames/second.

T1 also called `Runtime.Metrics()` every 10 milliseconds. That call takes the
engine lock through `ObserveOperational()`, clones publication state, takes
adapter/metrics locks, and calls `runtime.ReadMemStats`. The one-second sampler
itself later blocked for increasing intervals. The probe is therefore a
specific possible cause, not neutral instrumentation.

The retained numeric T1 artifact supplies the fake-stream envelope without
retaining or recreating real market data:

| Interval | Frames received | Approximate input rate |
| --- | ---: | ---: |
| 0.000-1.000 s | 130 | 130 frames/s |
| 1.000-2.035 s | 149 | 144 frames/s |
| 2.035-3.508 s | 312 | 212 frames/s |
| 3.508-6.140 s | 291 | 111 frames/s |

The dispositioned live frames contained 1.99 aggregate deliveries per frame.
The queued frames averaged approximately 382 bytes, and 27.8% of consumed
aggregates were rejected while 72.2% were inserted.

## Minimal harness

Add one opt-in, non-short test in `internal/operations`:

```text
TestLiveBackendContentionReproducer
  control
  legacy_probe
```

The subtests run sequentially and reuse one immutable generated stream. Each
subtest creates fresh state and uses:

- a 6,000-symbol production binding;
- the ordinary fake WebSocket connection and C5 `LiveAdapter`;
- the production 512-frame/64-MiB live queue;
- the ordinary handshake, `LiveAttempt`, `DeliverNextToEngine`, engine,
  feature, publication, validation, and delivery-observation paths;
- the production runtime configuration and timer loop; and
- bounded cancellation and joined adapter/engine/runtime cleanup.

Do not use the normalization-only attribution facade. Do not add a production
flag, alternate queue, batching path, profiling framework, or new state owner.
The test must not use credentials, issue provider requests, retain raw provider
data, or inspect the predecessor repository.

This first comparison deliberately starts no REST hydration, matching T1. It
tests the acknowledged live tail rather than claiming complete startup or
readiness.

## Fixed synthetic stream

Generate exactly 882 frames over 6.140 seconds using the four interval counts
above. Pace frames evenly inside each interval; do not send the whole fixture
as an instantaneous burst.

Each frame contains exactly two structurally valid `A` objects with event times
spanning the corresponding six session seconds. Use deterministic symbols and
values. Exactly 1,274 aggregates (72.22%) name distinct valid bound identities
and 490 (27.78%) name deterministic unknown symbols so the engine insert/reject
mix matches T1. Keep every encoded frame between 350 and 450 bytes using
harmless fixed-shape additive fields if necessary.

Before timing, assert:

- 6,000 bound symbols;
- 882 frames and 1,764 aggregate objects;
- exact per-interval frame counts;
- two aggregates per frame;
- exactly 1,274 valid and 490 unknown-symbol aggregates;
- every frame is within the production byte limit and target size band; and
- both subtests consume byte-identical frames in the same schedule.

## A/B variants

### Control

Run the delivery loop without `captureLiveDiagnosticSample` and without any
periodic call to `Runtime.Metrics()` or `runtime.ReadMemStats`.

At only the four interval boundaries, read the existing live-queue accounting
and delivery counters. These reads may not clone engine publication or sample
runtime memory. After the last frame, allow at most 250 milliseconds for the
already-admitted tail to drain.

### Legacy probe

Run the identical stream and delivery loop while invoking the same
`Runtime.Metrics()`-based probe used by T1 every 10 milliseconds. The probe may
discard its samples except for bounded numeric maxima. Do not stop early at the
oldest-frame threshold; preserve the full 6.140-second comparison unless a
frame rejection, ingress-integrity terminal, or 384-frame queue safety limit
requires immediate cancellation.

## Measurements

For each variant, record only:

- frames sent, read, admitted, dispositioned, queued, fenced, and rejected;
- aggregates consumed, inserted, rejected, revised, and fenced;
- queue depth and oldest-frame age at the four interval boundaries;
- maximum queue depth and oldest-frame age;
- delivery count, mean delay, and maximum delay;
- elapsed wall time, aggregate/frame rates, and average CPU cores; and
- whether all queue, adapter, engine, and transition accounting reconciles.

The control variant may take one `Runtime.Metrics()` sample only after its
timed delivery and drain boundary. CPU measurement must not call
`runtime.ReadMemStats` during the control trial.

Write no market-data artifact. A bounded numeric comparison may be written
under `var/live-backend-contention/` and must contain no credential, URL, raw
frame, symbol, price, volume, or provider text.

## Profiles and command

Run each variant once with separate CPU and blocking profiles:

```text
mkdir -p var/live-backend-contention/control var/live-backend-contention/legacy-probe

go test -count=1 -run '^TestLiveBackendContentionReproducer$/^control$' \
  -timeout 2m \
  -cpuprofile var/live-backend-contention/control/cpu.out \
  -blockprofile var/live-backend-contention/control/block.out \
  ./internal/operations

go test -count=1 -run '^TestLiveBackendContentionReproducer$/^legacy_probe$' \
  -timeout 2m \
  -cpuprofile var/live-backend-contention/legacy-probe/cpu.out \
  -blockprofile var/live-backend-contention/legacy-probe/block.out \
  ./internal/operations
```

Inspect `pprof -top` for both profiles. Compare the control critical path with
probe time under `Runtime.Metrics`, `Engine.ObserveOperational`, publication
cloning, `runtime.ReadMemStats`, mutex waits, GC, and scheduler/runtime work.
Background timer, heartbeat, server, and profile-shutdown waits must not be
misclassified as aggregate-delivery blocking.

## Outcome rules

`kept_up` means no rejection or terminal, accounting reconciles, no safety
threshold is reached, and the final admitted tail drains within 250
milliseconds. `saturated` means a rejection/terminal occurs, queued frames
reach 384, oldest-frame age reaches two seconds, or the tail fails to drain.

| Control | Legacy probe | Conclusion and next action |
| --- | --- | --- |
| kept_up | saturated | T1 was measurement-induced. Reopen its causal classification, replace the 10 ms heavyweight probe, and do not optimize the scanner from T1. |
| saturated | saturated or kept_up | The failure is reproducible without the probe. Use the control profiles to select the smallest scanner implementation boundary. |
| kept_up | kept_up | The retained numeric envelope is insufficient. Preserve the result and next test the complete `RunLive` startup with fake REST hydration before considering another provider run. |

Any accounting failure, stream-manifest mismatch, cleanup timeout, or unequal
A/B fixture is a harness failure, not a throughput result.

## Deliverable

Record the two-row comparison, focused CPU/blocking attribution, outcome-rule
classification, and the next implementation or diagnostic boundary. Do not
change production behavior in the same task. Do not rerun Massive or access
credentials; a later live rerun requires separate authorization after this
deterministic result.

## 2026-08-10 implementation and result

`TestLiveBackendContentionReproducer` is an opt-in non-short test in
`internal/operations`. It builds one 6,000-symbol production binding and one
immutable synthetic manifest with 882 frames, 1,764 aggregate objects, 1,274
distinct bound identities, and 490 deterministic unknown identities. Every
frame is 350-450 bytes and the four paced intervals are exactly the counts and
durations specified above. Both subtests use the ordinary fake WebSocket,
production C5 queue and handshake, `LiveAttempt`, `DeliverNextToEngine`, sole
engine, feature/publication/validation path, production runtime configuration,
and timer loop. No REST hydration starts.

The control adds no diagnostic sampling during the timed interval. The legacy
variant alone calls the exact T1 `captureLiveDiagnosticSample` path every 10
milliseconds. The production timer's ordinary one-second pressure sampling is
unchanged in both variants, so the heavyweight diagnostic probe is the only
A/B difference. Each stop cancels and joins the delivery loop, fences retained
queue work through the ordinary controlled-close path, and then requires queue,
adapter, engine admission, aggregate, and transition accounting to reconcile.
No production file, credential, provider request, raw-data artifact, or market
symbol/value output was introduced.

The final separate-profile runs used Go 1.26.5 on `darwin/arm64`,
MacBookPro17,1 with eight logical CPUs:

| Variant | Timed elapsed | Frames sent / read / dispositioned / queued | Aggregates consumed / inserted / rejected | Max queued | Max oldest age | Deliveries; mean / max delay | Frames out/s | Aggregates/s | Avg CPU cores | Outcome |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| Control | 6.141 s | 881 / 881 / 472 / 408 | 946 / 683 / 263 | 408 | 3.190 s | 946; 9.500 ms / 2.998 s | 76.86 | 154.05 | 3.851 | `saturated` |
| Legacy 10 ms probe | 5.952 s | 861 / 861 / 472 / 388 | 945 / 682 / 263 | 389 | 3.001 s | 945; 9.512 ms / 5.992 s | 79.30 | 158.77 | 4.030 | `saturated` |

At the first two boundaries, both variants were nearly caught up: 123 of 129
frames were dispositioned at one second, and 266 of 278 at 2.035 seconds. At
3.508 seconds both had dispositioned 472 of 590 frames, leaving 117 queued and
the oldest frame about 557 milliseconds old. Neither variant dispositioned
another complete frame. The legacy probe observed the 384-frame safety
boundary at 5.952 seconds; it had read 861 frames and queued 388. The control
reached its fixed 6.140-second boundary with 881 frames read, 408 queued, and
the oldest frame 3.190 seconds old. In each case the mandatory queue safety
rule closed ingress before the remaining scheduled frame(s), then controlled
cleanup fenced the retained tail and reconciled all accounting. There was no
frame rejection or ingress-integrity terminal.

The lower mean delay in the legacy row is not evidence that the probe helped.
The variants completed nearly identical prefixes: 946 aggregates in control
and 945 with the probe. Delay is measured only for completed deliveries, while
the growing queued tail is represented by the oldest-frame and drain-failure
measurements. The legacy probe itself could not sample continuously while the
engine lock was occupied; its six-second maximum delivery delay is the
already-owned engine transition that had to finish after ingress closed, not a
claim that the queue remained open for that additional interval.

### CPU and blocking attribution

The control CPU profile contains 10.90 CPU-seconds over a 15.20-second profiled
process lifetime. `Engine.transition`/`Engine.consume` accumulated 9.64 seconds
(88.4%). The same stack spent 9.64 seconds in
`stageAggregateEvaluationAtLocked`, 9.54 seconds in
`evaluateQualificationThrough`, 8.06 seconds in
`evaluateQualificationProof`, 6.78 seconds in
`qualificationCoverageTrustworthy`, and 5.37 seconds in
`exactAggregateCoverage`. The legacy profile has the same shape:
`Engine.transition` accumulated 7.19 of 8.48 CPU-seconds (84.8%), with 7.15
seconds in `evaluateQualificationThrough`. Its smaller absolute total reflects
the required earlier queue-safety stop, not a cheaper aggregate path. This is
full-population qualification/coverage work under the engine lock, not C5
normalization, GC, or memory sampling.

The control blocking profile contains 13.32 seconds of cumulative mutex wait:
8.83 seconds under `Engine.beginAdmission` (7.33 seconds specifically from
`AdmitAggregate`) and 2.99 seconds under the runtime's ordinary/final
`Metrics` calls. The legacy profile contains 17.58 seconds of mutex wait;
`Engine.beginAdmission` accounts for 7.31 seconds, while the probe's
`captureLiveDiagnosticSample` -> `Runtime.Metrics` ->
`Engine.ObserveOperational` path accounts for 7.28 seconds. Compared with the
control's 2.99-second `Runtime.Metrics` total, the legacy path adds about 4.29
seconds of cumulative metrics lock wait despite stopping earlier. Thus the
10-millisecond probe adds substantial lock competition and delayed
observations, but the unprobed aggregate caller already blocks behind the same
expensive engine transition.

`runtime.ReadMemStats`, publication cloning, GC, and scheduler/runtime work do
not appear as material CPU consumers in either top profile. The legacy block
profile does show the probe's engine-lock wait; any CPU inside its eventual
publication clone and memory read is below the profile's useful attribution
level. Large cumulative channel/select waits belong to the test server,
heartbeat, timer, delivery wait, and profile/cleanup lifetimes and are not
aggregate-delivery blocking.

The local ignored profiles are:

```text
var/live-backend-contention/control/cpu.out
var/live-backend-contention/control/block.out
var/live-backend-contention/legacy-probe/cpu.out
var/live-backend-contention/legacy-probe/block.out
```

Focused short verification and the existing compact fake-source cleanup proof
pass. A direct `-race` run of this wall-clock performance reproducer is not an
acceptance measurement: race instrumentation stretches a single known-heavy
engine transition beyond the unchanged ten-second production join bound. The
first such run exposed and corrected a test-only ordering defect by closing
adapter ingress immediately at the safety boundary before joining the owned
transition. After that correction, the race detector emitted no race report,
but the legacy performance subtest still timed out its ten-second delivery
join. The bounded `TestLiveHydrationThroughputHarness` cleanup/concurrency proof
passes under `-race`; the profiled non-race A/B runs above both pass and
reconcile.

### Classification and next boundary

**T1 was not measurement-induced.** The outcome-rule row is `saturated /
saturated`: the retained live envelope reproduces backlog, a greater-than-two-
second oldest frame, and tail-drain failure without the 10-millisecond probe.
The probe materially worsens engine-lock observation contention, so T1's exact
timing and CPU sample must not be treated as neutral instrumentation, but
removing it does not remove the failure.

The prior instantaneous 400-frame cost-attribution fixture answered decoder
cost before a production timer/evaluation cycle became material. This paced
six-second pipeline exposes the missing boundary: per-aggregate engine work
repeatedly enters full-population qualification/coverage evaluation. The
narrowest next implementation boundary is therefore
`runAggregateFeatureContributorLocked` ->
`stageAggregateEvaluationAtLocked`/`evaluateQualificationThrough`: preserve the
sole engine, canonical ordering, four-second target, exact coverage, and
publication semantics while preventing a complete 6,000-symbol qualification
proof from running for each aggregate admission. No such production change is
made by this diagnostic, and another provider run is not justified before that
boundary is corrected and deterministically re-proved.

The focused implementation specification is
[`live-engine-evaluation-coalescing-fix.md`](live-engine-evaluation-coalescing-fix.md).
