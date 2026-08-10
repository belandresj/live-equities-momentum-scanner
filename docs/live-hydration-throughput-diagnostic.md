# Live hydration throughput diagnostic

**Purpose:** Identify the smallest workload that makes the live aggregate
consumer fall behind during a midday fresh start. This is a focused diagnostic,
not a new component contract, production feature, or release gate.

## Question to answer

Can this Mac keep up with the current full-universe Massive aggregate stream:

1. with no REST hydration work;
2. with one REST hydration worker; and
3. as REST concurrency increases to two, four, and, only if still safe, eight
   workers?

The result must distinguish a live-path bottleneck from cross-source
contention and from an excessive hydration-worker setting.

## Minimal harness

Add one opt-in, non-short diagnostic test in `internal/operations`. Reuse the
production session binding, C5 adapter, C6 REST worker, engine, queue limits,
normalization, and metrics. Do not add a production flag or alternate state
owner.

The harness must:

- use one connection attempt per trial and never reconnect;
- subscribe to `A.*`, consume the acknowledgement, and continuously deliver
  the live tail through the ordinary engine path;
- start REST hydration immediately after acknowledgement for worker trials;
- cancel and join REST, WebSocket, and engine work at the trial boundary;
- run trials sequentially with no other scanner connection active;
- persist only bounded numeric summaries under `var/`; and
- never persist or print the credential, authorization data, URLs, provider
  bodies, raw frames, symbols, or market-data rows.

The test must skip unless explicitly enabled. Credential retrieval remains a
shell boundary using the existing macOS Keychain item; the test reads only
`MASSIVE_API_KEY` from its environment. Writing this spec does not itself
authorize a provider run.

## Four tests

Use the current production universe, queue limits, engine configuration, and
30 seconds measured from aggregate acknowledgement. Sample once per second,
including a sample immediately after acknowledgement. The first ten seconds
must remain individually visible because the observed failure occurred after
approximately four seconds.

### T1 — live-only baseline

Connect and drain the production `A.*` stream for 30 seconds without starting
REST hydration. The engine may remain non-ready; readiness and rankings are not
the subject of this test.

This answers whether the production live path alone can keep up with the
provider's immediate post-subscription burst and continuing stream.

### T2 — live plus one REST worker

Repeat with a fresh hydration plan and exactly one REST worker. Continue
draining live input while historical chunks and terminal results use the
ordinary engine path.

This answers whether any concurrent historical work is enough to make live
ingress unstable.

### T3 — hydration concurrency sweep

Repeat with two workers, then four workers. Run eight workers only if four
workers remains stable. Stop the sweep at the first saturated result; do not
rerun a known-higher unsafe concurrency.

This identifies the current Mac's hydration concurrency boundary without
repeating the already observed uncontrolled eight-worker failure.

### T4 — bottleneck classification

Compare the per-second samples and classify exactly one result:

| Observation | Conclusion |
| --- | --- |
| T1 falls behind | The live normalization/admission/evaluation path itself needs batching or optimization before hydration scheduling matters. |
| T1 is stable but T2 falls behind | REST and live work contend at the single engine boundary; live-priority admission and reduced hydration-side work are required. |
| T1 and T2 are stable but a higher worker count falls behind | Cap or dynamically throttle hydration concurrency at the measured boundary. |
| Every tested level is stable | The earlier failure depends on another production-start interaction; preserve these results and isolate that interaction before changing architecture. |

Do not implement a production fix in this diagnostic task.

## Measurements

Record one row per second and one summary per trial:

- elapsed time and configured hydration workers;
- live frames read, admitted, dispositioned, queued, and fenced;
- capacity/receipt/oversize rejections;
- current and maximum queued frames and bytes;
- oldest live-frame age;
- live delivery count, mean delay, one-second maximum delay, and overall
  maximum delay;
- engine queue occupancy;
- aggregate consumed/inserted/revised/rejected/fenced counts;
- hydration open/completed/failed/canceled/fenced symbols and rows consumed;
- hydration rows per second;
- heap allocation/in-use and goroutine count; and
- Go runtime CPU-seconds delta divided by wall time, reported as average CPU
  cores used.

The final comparison table is:

| Workers | Duration | Live frames in/s | Live frames out/s | Max queued | Max oldest age | Rejections | Max delivery delay | Hydration rows/s | Avg CPU cores | Outcome |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |

## Safety stops

End the current trial immediately and close/join the attempt when any of these
occurs:

- any live-frame rejection;
- ingress-integrity or connection terminal;
- queued frames reach 384, which is 75% of the 512-frame limit;
- oldest live-frame age reaches two seconds;
- accounting stops reconciling; or
- the 30-second trial deadline expires.

A safety stop is valid diagnostic evidence, not a reason to retry. The harness
must not enter the production recovery loop. If cancellation or cleanup does
not join within ten seconds, fail the harness and do not start another trial.

## Minimal verification and deliverable

Before any live execution, run only:

```text
go test -short -timeout 2m ./internal/operations ./cmd/scanner
go test -race -run TestLiveHydrationThroughputHarness -timeout 2m ./internal/operations
```

The race test uses fake HTTP/WebSocket sources and proves bounded sampling,
safety-stop behavior, cancellation, and joins. It does not simulate production
rates.

After the owner authorizes the exact trading date, the implemented live harness
must run with one command of this shape:

```zsh
LIVE_HYDRATION_DIAGNOSTIC=1 \
LIVE_TRADING_DATE=YYYY-MM-DD \
LIVE_DIAGNOSTIC_OUTPUT="$PWD/var/live-hydration-diagnostic" \
MASSIVE_API_KEY="$(security find-generic-password \
  -a joshuabelandres \
  -s momentum-scanner-massive-api \
  -w)" \
go test -count=1 -run '^TestLiveHydrationThroughputDiagnostic$' \
  -timeout 10m ./internal/operations
```

The live deliverable is the per-second numeric artifact, the five-row-or-fewer
comparison table, the T4 classification, and the narrowest next implementation
boundary. Do not run replay, the dashboard, repository-wide acceptance,
benchmarks, soak tests, repeated live trials, or retained-artifact work.

## Implementation record

**Harness state:** implemented and pre-live verification complete on 2026-08-10.

`TestLiveHydrationThroughputDiagnostic` in
`internal/operations/live_hydration_diagnostic_test.go` owns only test-time
orchestration. Each level creates a fresh production runtime, adapter, and REST
worker; opens exactly one aggregate attempt; measures from the accepted
aggregate acknowledgement; stops without entering `RunLive`'s recovery loop;
and joins hydration, delivery, adapter, and engine work within one ten-second
cleanup boundary. The live-only level leaves hydration intentionally unstarted.

The implementation uses the production 512-frame/64-MiB C5 queue, C6 budgets
and chunking, C8 engine configuration, one-second samples, and a separate
10-millisecond scalar safety probe so a dangerous queue/age/accounting change
does not wait for the next persisted sample. Output is one create-only,
maximum-1-MiB `summary.json` with at most five trials and 31 numeric samples per
trial. It contains no credential, URL, provider body, raw frame, symbol, or
market-data row. Classification and outcome values are bounded numeric enums;
the test log renders the required comparison values without provider data.

Reference binding is cache-only: the live test requires validated current
same-date universe and prior-close caches under `var/reference` and makes no
reference-data provider request. Thus each measured trial contains only the
one WebSocket attempt and, for worker levels, ordinary aggregate REST
hydration.

Pre-live proof passed exactly as prescribed:

```text
go test -short -timeout 2m ./internal/operations ./cmd/scanner
go test -race -run TestLiveHydrationThroughputHarness -timeout 2m ./internal/operations
```

The fake-source race proof covers bounded sampling, every safety predicate,
numeric-only serialization, cancellation of an active REST request, and joined
REST/WebSocket/engine cleanup. It does not establish live throughput.

**Live evidence state:** completed for the authorized 2026-08-10 run. T1 hit
the oldest-frame safety stop after 6.140 seconds with no REST hydration active,
so the harness correctly stopped without executing T2 or any higher worker
level.

The first authorized command on 2026-08-10 stopped during cache-only preflight,
before any WebSocket or REST connection, because the Go test process resolves
relative paths from `internal/operations`. The harness now derives
`var/reference` from the already validated absolute output path. This was a
test-harness path defect and consumed no live trial; the local gates were rerun
before continuing with the still-unused authorized provider run.

The next command exposed the complementary preflight defect: output
containment still anchored `var/` to the package working directory. It also
stopped before a provider connection. After two failures from the same
relative-working-directory premise, the harness replaced that premise with a
bounded upward search for the repository's exact Go module declaration; the
fake-source proof now checks module-root discovery directly. Reference and
output paths both derive from that single validated root.

## 2026-08-10 live result

| Workers | Duration | Live frames in/s | Live frames out/s | Max queued | Max oldest age | Rejections | Max delivery delay | Hydration rows/s | Avg CPU cores | Outcome |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 0 | 6.140 s | 143.65 | 96.42 | 290 | 2.089 s | 0 | 2.622 s | 0.00 | 5.146 | saturated: oldest-frame stop |

T1 initially kept up: at one second, 130 frames had entered and 130 had been
dispositioned with an empty queue. At 2.035 seconds, ingress and disposition
still matched at 279 frames, although the maximum observed delivery delay had
risen to 752 milliseconds. By 3.508 seconds, 591 frames had entered, only 457
had been dispositioned, and 134 were queued with the oldest frame 387
milliseconds old. At the stop, 882 frames had entered, 592 had been
dispositioned, and 290 remained queued; accounting still reconciled and there
had been no frame rejection.

**T4 classification:** the live normalization/admission/evaluation path itself
falls behind. REST contention and hydration-worker concurrency are excluded as
causes of this trial because no hydration plan or REST request started. T2 and
the concurrency sweep would add provider load after the decisive condition and
were therefore prohibited by the stop rule.

**Narrowest next implementation boundary:** profile and then batch or optimize
the C5 aggregate-frame normalization through sole-engine admission/evaluation
path while preserving one canonical state owner, ordering, reconciliation,
and existing queue limits. Do not cap hydration workers as the primary fix:
zero workers already saturated.

**Measurement limitation:** the intended one-second sampler itself blocked
behind the contended production metric/engine path, producing observations at
0, 1.000, 2.035, 3.508, and 6.140 seconds rather than every exact second. This
is additional contention evidence but means the artifact is not a complete
per-second time series. The safety-stop observation, queue accounting, T1/T4
classification, and no-hydration causal distinction remain valid. Do not
repeat the live trial to fill the missing samples; the stop is already the
required diagnostic evidence.

The bounded numeric artifact is
`var/live-hydration-diagnostic/summary.json` (7,704 bytes, mode `0600`); its
directory is mode `0700`. It contains one trial and five samples, with no
credential, URL, provider body, raw frame, symbol, or market-data row.

## Next focused test: live aggregate cost attribution

This is a small diagnostic, not a component spec or production redesign. Its
only question is: where does the live-only path spend the time that caused T1
to fall behind?

Add one opt-in, non-short test in `internal/operations` using fake WebSocket
input and the ordinary C5 adapter, engine, and production queue limits. Use a
6,000-symbol binding and one fixed stream of 400 frames containing two valid
second aggregates per frame. This approximates the observed live frame density
while remaining below the 512-frame queue limit, deterministic, and
credential-free.

Run the identical aggregate values through two sequential phases:

1. **Normalization only:** decode and classify every frame through the C5 live
   normalizer, consuming every result but admitting nothing to the engine.
2. **End to end:** send the same frames through `LiveAttempt` and
   `DeliverNextToEngine`, including canonical application, feature work,
   completion waiting, publication construction, and publication validation.

For each phase, record only elapsed time, frames/s, aggregates/s, allocations,
and reconciled counts. Run the test once with CPU and blocking profiles:

```text
mkdir -p var/live-aggregate-cost
go test -count=1 -run '^TestLiveAggregateCostAttribution$' \
  -timeout 2m \
  -cpuprofile var/live-aggregate-cost/cpu.out \
  -blockprofile var/live-aggregate-cost/block.out \
  ./internal/operations
go tool pprof -top var/live-aggregate-cost/cpu.out
go tool pprof -top var/live-aggregate-cost/block.out
```

Group the result into four existing stages:

- frame decoding and normalization;
- adapter-to-engine admission, completion waiting, and channel/lock overhead;
- canonical aggregate and feature updates; and
- immutable publication construction, cloning, and validation.

Do not add timing counters, a profiling framework, a production flag, a new
queue, or a batching implementation merely to run this test. The Go profiles
and the two phase totals are sufficient.

The decision is direct:

| Evidence | Next implementation boundary |
| --- | --- |
| Normalization-only time is most of end-to-end time | Optimize the C5 decoder/normalizer without changing engine semantics. |
| Admission/waiting dominates the block profile | Replace the one-aggregate synchronous handoff with one bounded ordered admission while keeping the same state owner. |
| Canonical/feature functions dominate CPU | Optimize the identified incremental state or feature update before changing publication cadence. |
| Publication construction/validation dominates CPU | Coalesce publication work across a bounded ordered aggregate batch; do not weaken validation. |
| No stage clearly dominates | Preserve the profiles and optimize the smallest combined boundary containing the top contributors; do not guess from T1 alone. |

This test does not need live data, does not rerun the failed T1 provider trial,
and does not prove that batching is the fix. It identifies the smallest code
boundary that the next implementation should change.
