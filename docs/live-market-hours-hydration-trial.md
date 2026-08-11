# Live market-hours aggregate and hydration trial

**Status:** focused implementation and execution specification; execution
requires separate authorization for one exact trading date and credential.

## Goal

Finish the minimum market-hours evidence needed to start a controlled live
scanner after 04:00 ET:

1. prove the corrected scanner keeps up with the actual live aggregate stream
   when REST hydration is disabled;
2. prove one production REST worker completes the current-date full-universe
   hydration while that live stream remains current; and
3. if both pass, launch the ordinary scanner with exactly one hydration worker
   and observe it reaching honest backend readiness.

This is a focused diagnostic and launch gate, not a component contract,
provider SLA, capacity study, trading-signal study, or authorization to access
credentials.

## Why both trials are required

A scanner started after 04:00 ET needs REST hydration over `[04:00,R)` for all
valid-prior-close symbols, where `R` is the acknowledged live handoff. Live
aggregates after the acknowledgement preserve the tail but cannot reconstruct
the earlier session. Therefore:

- **live-only** proves actual WebSocket burst shape, normalization, queueing,
  engine admission, timer evaluation, and publication cost after the recent
  corrections; it is expected to remain not ready because hydration was
  intentionally omitted; and
- **one worker** proves actual REST latency, rate limits, response shapes,
  empty/value outcomes, canonical merge, aggregate ingress fence, and the
  transition to current ranking while the same live path remains active.

The earlier 2026-08-10 provider live-only trial ran before evaluation
coalescing and failed at 290 queued frames and 2.089 seconds oldest-frame age.
The corrected live-only and live-plus-hydration paths now pass deterministic
T1 tests, but neither corrected path has been observed against the provider.

## Minimal implementation changes

### 1. Select exactly one diagnostic trial

Change `TestLiveHydrationThroughputDiagnostic` to require:

```text
LIVE_HYDRATION_TRIAL=live_only
```

or:

```text
LIVE_HYDRATION_TRIAL=one_worker
```

Each invocation runs exactly one fresh runtime, adapter, engine, WebSocket
attempt, artifact, and—only for `one_worker`—one hydration generation with one
REST worker. Remove the automatic `0,1,2,4,8` loop from this diagnostic. Reject
missing or unknown trial values before credential use or provider connection.

`live_only` must never call `Runtime.hydrate`, create a hydration plan, or make
an aggregate REST request. `one_worker` must use the production planner,
worker, acquisition/normalization, chunk, terminal, canonical merge, and real
aggregate ingress-fence path. Do not add a diagnostic scanner implementation.

### 2. Use an unintrusive safety monitor

The former 10-millisecond `Runtime.Metrics()` probe must not return. It calls
`runtime.ReadMemStats` and takes the engine observation lock, so it can change
the behavior being measured.

Use two bounded sampling paths:

- every 50 milliseconds, read only lightweight C5 queue accounting and stop
  on a frame rejection, 384 queued frames, or two-second oldest-frame age;
- once per second, capture the existing bounded numeric runtime sample for the
  retained artifact and check engine integrity, suppression, aggregate and
  hydration accounting, delivery delay, watermark lag, CPU, memory, and
  readiness.

If the one-second sample is delayed behind engine work, the independent queue
probe must remain able to cancel the trial. Every loop, channel wait, provider
operation, and join remains context-bounded. Do not raise queue limits or
parallelize engine mutation.

### 3. Give the two modes correct completion semantics

`live_only` runs for 30 seconds after aggregate acknowledgement and then stops
cleanly. It passes or fails solely on live-path stability and accounting; lack
of hydration/readiness is expected and must not fail it.

`one_worker` runs until either:

- hydration is fully terminal, its ingress fence is reconciled, backend
  readiness becomes true, and ten subsequent one-second samples remain
  stable; or
- eight minutes elapse after aggregate acknowledgement.

The existing outer command timeout remains ten minutes and cleanup must join
REST, live delivery, adapter, timer, and engine work within ten seconds after
success, failure, or cancellation.

Write one bounded numeric artifact per invocation. Use separate output
directories so the second trial cannot overwrite the first. Artifacts retain
no credential, header, URL, raw frame, symbol, provider body, or price/volume
row.

### 4. Make the production launch use the tested worker count

Add a live-mode scanner flag:

```text
-hydration-workers=1
```

Accept only `1`, `2`, `4`, or `8`; default to `1`. Replay mode rejects the
flag. Pass the selected value to both `LiveComponents.Workers` and
`MaximumResidentRecords = workers * 57_600`. Do not change rows per chunk,
queue limits, deadlines, recovery policy, or any market semantics.

The 2026-08-10 controlled two-worker launch invalidated the original 512 MiB
cumulative REST response budget: hydration reached 4,558 of 5,517 terminal
symbols before the transfer bound, with no live-queue loss. The projected
complete transfer was about 625 MiB. Production therefore uses a 2 GiB
cumulative response budget. This is a transfer/accounting bound, not a heap
limit; per-page, per-request, row, resident-record, and retry bounds remain in
force. A response byte read only to prove that the cumulative bound was
crossed is excluded from accepted terminal accounting, so budget exhaustion is
a bounded provider failure and cannot manufacture an engine accounting-
integrity suppression.

This flag changes only bounded REST concurrency. It does not permit zero
workers in the ordinary scanner because a mid-session production start cannot
claim complete readiness without hydration.

## Acceptance conditions

### Live-only provider trial

The corrected live-only trial passes only when all of these hold for the full
30 seconds:

- the aggregate subscription is acknowledged and the connection remains on
  one coherent active epoch;
- every read frame is admitted or still exactly accounted in the bounded
  classifier; there are no capacity, receipt, oversize, gate, or close
  rejections;
- aggregate consumed dispositions reconcile as inserted, revised, duplicate,
  rejected, withdrawn/conflict, or fenced, with no integrity terminal;
- queued frames remain below 384 and oldest-frame age remains below two
  seconds at every lightweight probe;
- the final ten one-second samples show no persistent queue or oldest-age
  growth, and the final sample has an empty live queue;
- no engine suppression, connection fence, aggregate fence, or accounting
  divergence occurs; and
- hydration planned/open/terminal/row counts remain zero.

Backend readiness is expected to be false with a hydration/fence-pending
reason. That is not a failure and must not be mislabeled as a runnable scanner.

If this trial fails, do not run `one_worker` or launch the scanner. Preserve the
numeric artifact and correct only the lowest failing live boundary.

### One-worker provider trial

Run this only after `live_only` passes. It must satisfy every live-only
queue/integrity/accounting condition and all of the following:

- the hydration plan contains every valid-prior-close bound symbol exactly
  once and uses one worker;
- hydration makes forward progress while live frames are being delivered;
- every planned request reaches exactly one terminal value/empty/failed/
  canceled/fenced bin, and row accounting reconciles;
- for this launch gate, failed, canceled, and fenced work are all zero;
- the aggregate ingress fence is applied once after all work is terminal;
- lifecycle becomes `live`, `BackendReady` and `RankingCurrent` are true,
  suppression is empty, accounting is valid, and committed watermark lag is
  within the existing two-second readiness tolerance;
- publication, hydration, aggregate, adapter, queue, admission, and transition
  identities describe the same accepted prefix; and
- ten consecutive post-readiness samples remain within the safety thresholds
  without queue growth or loss of readiness.

Successful empty responses count as complete no-print evidence. They are not
failures and must not be fabricated into marks.

If the trial reaches terminal hydration but not readiness, or live currentness
falls behind, classify it as failed even when every HTTP request returned.

### Production launch gate

The scanner may be launched for a bounded operator-observed trial only when
both provider artifacts were produced on the same trading date, same code,
same host, same account/entitlement context, and both passed without threshold
exceptions.

Start with `-hydration-workers=1`. The launch is successful when it independently
repeats the one-worker outcome through the ordinary CLI composition:

- exact-date binding and prior closes resolve;
- aggregate acknowledgement occurs;
- hydration completes and the real fence reconciles;
- status reaches `ProcessLive=true`, `BackendReady=true`,
  `RankingCurrent=true`, and `AccountingValid=true`;
- watermark lag stays within tolerance and queue/oldest age do not grow; and
- the loopback snapshot API reports the same publication identity and status.

Stop immediately on any diagnostic safety threshold, integrity/suppression,
loss of accounting validity, repeated readiness loss, or provider entitlement
error. A clean empty or sub-20 ranking table is allowed; fabricated readiness
or rows are not.

## Implementation proof

Extend the existing fake-source harness rather than adding another framework.
It must prove:

1. trial parsing rejects missing/unknown modes before trial construction;
2. `live_only` performs zero hydration work and exits successfully despite
   expected not-ready status;
3. `one_worker` creates exactly one worker/generation, reaches the real fence
   and readiness, and waits for the post-ready samples;
4. timeout, queue threshold, oldest-age threshold, rejection, integrity, and
   accounting failure each cancel and join all work;
5. the 50-millisecond safety path does not call `Runtime.Metrics()` or
   `runtime.ReadMemStats`;
6. the two artifacts cannot overwrite one another and serialize only bounded
   numeric fields; and
7. scanner flag validation/default/propagation produces worker/resident bounds
   of `1/57,600`, `2/115,200`, `4/230,400`, and `8/460,800`, while replay and
   zero/unsupported values are rejected.

Required local commands before credential use:

```text
go test -short -timeout 2m ./internal/operations ./cmd/scanner

go test -race -count=1 \
  -run '^(TestLiveHydrationThroughputHarness|TestLiveRESTHydrationProgressionHarness)$' \
  -timeout 2m ./internal/operations

go test -short -timeout 2m ./...
go vet ./...
git diff --check
```

## Authorized execution sequence

Execution still requires explicit authorization for the exact date and
credential. Use the existing cache-only reference preflight and macOS Keychain
lookup. Never print or persist the credential.

### 1. Live-only

```zsh
LIVE_HYDRATION_DIAGNOSTIC=1 \
LIVE_HYDRATION_TRIAL=live_only \
LIVE_TRADING_DATE=YYYY-MM-DD \
LIVE_DIAGNOSTIC_OUTPUT="$PWD/var/live-hydration-diagnostic/live-only" \
MASSIVE_API_KEY="$(security find-generic-password \
  -a joshuabelandres \
  -s momentum-scanner-massive-api \
  -w)" \
go test -count=1 -run '^TestLiveHydrationThroughputDiagnostic$' \
  -timeout 10m ./internal/operations
```

Review the numeric summary. Do not continue unless its single row is
`workers=0`, the outcome is stable, and every live-only acceptance condition
passes.

### 2. Live plus one worker

```zsh
LIVE_HYDRATION_DIAGNOSTIC=1 \
LIVE_HYDRATION_TRIAL=one_worker \
LIVE_TRADING_DATE=YYYY-MM-DD \
LIVE_DIAGNOSTIC_OUTPUT="$PWD/var/live-hydration-diagnostic/one-worker" \
MASSIVE_API_KEY="$(security find-generic-password \
  -a joshuabelandres \
  -s momentum-scanner-massive-api \
  -w)" \
go test -count=1 -run '^TestLiveHydrationThroughputDiagnostic$' \
  -timeout 10m ./internal/operations
```

Review the single `workers=1` row, terminal work/fence/readiness evidence, and
post-ready stability. Do not run higher worker levels as part of this gate.

### 3. Controlled scanner launch

Use the same exact trading date and one worker:

```zsh
MASSIVE_API_KEY="$(security find-generic-password \
  -a joshuabelandres \
  -s momentum-scanner-massive-api \
  -w)" \
go run ./cmd/scanner \
  -run-mode=live \
  -trading-date=YYYY-MM-DD \
  -hydration-workers=1 \
  -reference-dir="$PWD/var/reference" \
  -checkpoint-dir="$PWD/var/checkpoints" \
  -api-address=127.0.0.1:8080 \
  -allow-origin=http://127.0.0.1:3000
```

Observe startup status and metrics until readiness is stable. Keep the first
operating trial bounded and stop with `Ctrl-C`; shutdown must join cleanly.
Changing worker count, retry behavior, response budget, queue bounds, or
provider endpoints requires a separate decision based on the retained
evidence.

## 2026-08-10 controlled-launch correction

The ordinary two-worker scanner reached 4,558 terminal hydration results
(4,515 value, 43 empty) out of 5,517 before failing. The two active REST workers
then reported provider failures as the shared 512 MiB response budget was
exhausted; the first boundary-crossing result included its one proof byte and
was one byte greater than the engine-authorized remaining budget. The engine
correctly treated that impossible terminal accounting as `accounting_integrity`,
fenced the remaining 957 requests, and suppressed publication. A second defect
then let the live supervisor retry after terminal suppression, creating a
reconnect spin until the process was stopped.

The correction raises the production cumulative response budget to 2 GiB,
excludes the proof byte from terminal accounting, and makes `suppressed` and
`ended` engine states terminal to the live supervisor. The T/Q pressure policy
was not the cause: its heap thresholds shed optional trades/quotes while
aggregate hydration continues. That policy remains unchanged pending a
successful post-fence observation of compaction and T/Q recovery.

The corrected-budget rerun completed all 5,517 hydration terminals (5,468
value, 49 successful empty, zero failed, canceled, or fenced) in about 9
minutes 20 seconds. It then exposed a separate finalization defect before the
ingress fence could be applied. For each live aggregate, tail compaction walked
every retained record and, for every record, linearly scanned all 5,517 active
hydration requests to decide whether the identity was pinned. Late-session tail
size made that repeated work dominate one CPU core; live delivery stopped at
12,026, the bounded input queue reached 512/512, and continuity could no longer
be trusted. The scanner was stopped without claiming readiness.

The second correction indexes the active hydration plan by symbol. A symbol in
that index is known to be generation-pinned, so ordinary live installation
skips its entire tail-compaction scan while hydration is active. The ingress
fence first makes the generation inactive and then performs the one required
compaction per symbol, preserving the same pin/fence semantics without work
proportional to retained records times total population on every aggregate.
The focused semantic regression, affected race proofs, full short suite, vet,
UI tests, and diff checks pass. A repeated provider launch remains required to
observe fence latency, readiness, and post-fence T/Q recovery under the actual
late-session population.

That repeated launch again completed 5,517/5,517 terminals with zero failures,
but delivery stopped at 8,152 while the single fence operation ran; the live
queue rose monotonically from 22 to 350 before the scanner was stopped. This
proved the indexed active-hydration fast path fixed ordinary live installation,
but moving all retained-state compaction to one fence operation was still not
safe under the late-session population.

A chronological-fold correction then replaced random map-order folding with one
sort per symbol followed by ordered folding. Focused, race, and full short
verification passed. The next provider launch again completed 5,517/5,517
terminals (5,468 value, 49 successful empty, zero failed) in about 8 minutes 54
seconds. Ordinary hydration stayed healthy: deliveries advanced to 9,421 and
queue high-water was only 7. At the fence, however, delivery again stopped and
the queue rose from 157 to 483 of 512 in about 28 seconds. The scanner was
stopped before a capacity rejection.

The live evidence therefore reopens fence-finalization acceptance. Random-order
folding was a real avoidable cost, but it was not the sole cause. Full-population
compaction, coverage installation, feature maintenance, and initial projection
still execute as one synchronous engine transition whose duration exceeds the
live queue's continuity window for a late-session population. Do not increase
the queue or call this ready based on clean REST completion. The next correction
is the no-network cached-artifact profile and correction specified in
[`live-fence-finalization-cached-hydration-correction.md`](live-fence-finalization-cached-hydration-correction.md).
It must either remove the measured dominant algorithmic cost or split the same
engine-owned fence work into bounded, ordered units that interleave with live
aggregate consumption without publishing readiness early.

## Result and next decision

Record one compact row for each provider trial:

| Trial | Duration | Hydration completion | Frames/s in/out | Aggregates/s | Max queued | Max oldest | Max delivery | Watermark lag | Fence | Ready | Accounting | Outcome |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |

- If both pass, proceed immediately to the one-worker controlled scanner
  launch.
- If live-only fails, the corrected aggregate path is not ready for a scanner
  launch; hydration is not the cause.
- If live-only passes and one-worker fails, retain the two artifacts and
  distinguish REST/provider latency from local engine/queue contention before
  changing production behavior.
- If the scanner launch fails after both diagnostics pass, inspect only the
  additional CLI boundaries: reference resolution, checkpoint composition,
  API binding, T/Q commands, and long-running lifecycle.

Actual provider observations establish only behavior for the recorded date,
host, code, account, and interval. They do not establish a public SLA or any
trading edge.
