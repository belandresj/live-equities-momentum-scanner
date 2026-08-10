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

**Live evidence state:** pending. No provider request or credential access was
performed because no exact trading date has yet been authorized. The
comparison table, T4 classification, and next implementation boundary remain
unresolved until that single authorized run.
