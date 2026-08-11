# Live aggregate plus REST hydration progression

## Goal

Determine whether the corrected scanner can stay current on the T1 live
aggregate envelope while full-universe REST hydration progresses and completes.
This is the next focused test toward running the scanner live. It is not a
component contract, architecture exercise, public capacity claim, or trading
signal study.

The test must answer two practical questions:

1. Does one hydration worker complete full-universe startup without making the
   live aggregate tail fall behind?
2. If so, how much additional REST concurrency is safe on this Mac before live
   latency or queue growth becomes unacceptable?

## Evidence already established

The post-coalescing deterministic control passes the aggregate-only T1
envelope through the real fake-WebSocket, C5 queue, `LiveAttempt`, engine,
timer, publication, and accounting paths:

- 6,000 bound symbols;
- 882 frames and 1,764 aggregates over 6.140 seconds;
- 1,274 inserted aggregates and 490 expected unknown-symbol rejections;
- all frames and aggregates dispositioned;
- maximum queue depth 1, oldest-frame age 0, and drained tail; and
- no per-aggregate full-population evaluation in the CPU profile.

This proves the corrected live-only path for the modeled T1 rate. It does not
yet prove concurrent REST hydration, complete startup, provider behavior, or a
6,000-aggregate-per-second extreme.

## Test 1: deterministic live plus one hydration worker

Add one opt-in, non-short subtest in `internal/operations` that reuses the
production runtime and existing fake-source infrastructure. Do not add a
production option or alternate scanner path.

Use:

- a 6,000-symbol binding with a valid prior close for every symbol;
- the existing paced T1 frame shape and rates, extended only as long as needed
  to overlap full hydration and its ingress fence;
- the production C5 queue, normalization, delivery, engine, one-second timer,
  C6 hydration planner, REST acquisition/normalization, chunking, terminal
  facts, and fence path;
- a fake REST server receiving work for every planned symbol; and
- exactly one hydration worker.

The fake REST fixture should be cheap but semantically complete: deterministic
successful-value and successful-empty responses, bounded response bodies, no
retry delay, and exact request/terminal accounting. It need not imitate real
network latency. Its purpose is to expose CPU, allocation, lock, admission,
merge, evaluation, and publication contention across the real scanner paths.

Start the live tail immediately after aggregate acknowledgement. Start the
fresh hydration plan at the ordinary production point and continue the paced
live stream until all hydration work is terminal and the real aggregate
ingress fence has been consumed. Then stop the stream and allow at most 250
milliseconds for its admitted tail to drain.

### One-worker acceptance

The one-worker test passes only when all of the following are true:

- every generated live frame is read, admitted, and dispositioned;
- every generated aggregate reaches its expected inserted, revised, rejected,
  or fenced disposition, with no unexplained loss;
- there is no frame rejection, ingress-integrity terminal, connection fence,
  hydration-integrity terminal, or engine suppression;
- maximum live queue depth stays below 384 frames;
- maximum oldest-frame age stays below two seconds;
- the final admitted live tail drains within 250 milliseconds;
- hydration shows forward progress during live delivery, every one of the
  6,000 symbols reaches exactly one terminal result, and its row/work
  accounting reconciles;
- the aggregate ingress fence is applied exactly once after all registered
  hydration work is terminal;
- after the fence, lifecycle/readiness, committed watermark, ranking
  population, publication sequence, and aggregate/hydration accounting all
  describe the same accepted prefix; and
- adapter, queue, engine admission, transition, publication, aggregate, and
  hydration identities all reconcile at shutdown.

The test must assert these conditions. A diagnostic log saying `stable` is not
enough.

## Test 2: bounded hydration concurrency sweep

Run this only after the one-worker test passes without weakening an acceptance
condition.

Repeat the byte-identical live and REST fixtures with two workers, then four.
Run eight workers only if four workers passes. Use fresh runtime, adapter,
engine, fake servers, and hydration generation for every level. Stop at the
first failed level; do not continue to a known-higher load.

The safe local setting is the highest worker count that passes every
one-worker acceptance condition. Do not select a worker count merely because
hydration finishes faster. Prefer the lower count when adjacent levels have
materially similar completion time but the higher level increases live queue
age, delivery delay, or CPU substantially.

Record this compact comparison:

| Workers | Hydration completion | Live frames/s | Aggregates/s | Max queued | Max oldest age | Max delivery delay | Tail drain | Fence applied | Accounting | Outcome |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |

No production worker default changes as part of the test. The result supplies
evidence for a separate small configuration decision.

## Measurement and safety

Measure live queue depth/age only at the paced interval boundaries and with a
lightweight queue-accounting read. Do not restore the legacy 10-millisecond
`Runtime.Metrics()`/`runtime.ReadMemStats` probe. A one-second full metrics
sample is allowed if it does not block delivery; omit it if it materially
changes the control.

Bound the deterministic test to two minutes per worker level and join the REST
workers, live delivery loop, adapter, and engine within ten seconds after any
success or failure. Cancel immediately on frame rejection, integrity terminal,
384 queued frames, two-second oldest age, or irreconcilable accounting.

On a failure, capture one CPU and blocking profile for the lowest failing
worker count. Do not increase queue limits, parallelize engine mutation, or
continue the sweep before identifying the contention point.

Write no raw market-data artifact. Optional results under `var/` must be
bounded numeric summaries without credentials, URLs, raw frames, symbols, or
price/volume rows.

## Verification commands

The implemented deterministic progression should run with:

```text
go test -short -timeout 2m ./internal/engine ./internal/operations ./cmd/scanner

go test -race -count=1 \
  -run '^(TestLiveAggregateEvaluationCoalescing|TestLiveRESTHydrationProgressionHarness)$' \
  -timeout 2m ./internal/engine ./internal/operations

go test -count=1 \
  -run '^TestLiveRESTHydrationProgression$/^workers_1$' \
  -timeout 2m ./internal/operations
```

After one worker passes, run the sweep subtests sequentially. Do not run all
levels in parallel.

## Deterministic implementation result

**State:** passed on 2026-08-10. The production-path one-worker test completed
all 6,000 REST requests as exactly 3,000 successful-value and 3,000
successful-empty terminals while a full 882-frame / 1,764-aggregate paced T1
envelope ran. At the paced interval boundaries, maximum queued frames and
oldest-frame age were both zero. The admitted tail drained in 2.302
milliseconds, the hydration ingress fence reconciled, lifecycle/ranking were
current, and adapter, queue, engine, aggregate, hydration, and publication
accounting reconciled.

The first one-worker execution correctly failed. REST work finished before the
hydration fence became eligible, but `Runtime.hydrate` stopped its live-delivery
pump during that eligibility wait, allowing the socket reader to fill the C5
queue. Keeping the sole consumer active through the wait exposed a second
serialized cost: the fence installed and verified eight hours of exact
coverage one second at a time for every symbol. The correction retains the same
single engine/fence path, uses exact fixed-bitmap word ranges for coverage, and
fast-forwards qualification only when the complete retained bar population is
strictly below the 45-second gate minimum. Focused conflict/presence/range and
sparse-qualification regressions cover the dangerous false-success cases.

| Workers | Hydration completion | Live frames/s | Aggregates/s | Max queued | Max oldest age | Max delivery delay | Tail drain | Fence applied | Accounting | Outcome |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | --- |
| 1 | 5.806 s | 141.6 | 283.1 | 0 | 0 | 809.389 ms | 2.302 ms | yes | reconciled | pass |
| 2 | 5.809 s | 141.6 | 283.2 | 0 | 0 | 811.122 ms | 1.152 ms | yes | reconciled | pass |
| 4 | 5.811 s | 141.5 | 283.1 | 0 | 0 | 813.243 ms | 2.304 ms | yes | reconciled | pass |
| 8 | 5.809 s | 141.6 | 283.1 | 0 | 0 | 810.749 ms | 2.354 ms | yes | reconciled | pass |

The safe local setting selected by this deterministic fixture is **one worker**.
Additional workers did not reduce completion time because the common fence-
eligibility boundary dominated, while the two- and four-worker trials consumed
materially more CPU in this single-trial comparison. This is test evidence for
a later configuration decision; no production worker default changed. The
race harness uses the same production composition and full paced frame envelope
with 128 symbols so race instrumentation does not masquerade as the separate
6,000-symbol capacity claim.

The deterministic result does not authorize credentials or provider requests.
Actual REST latency/rate limits, live provider burst shape, pagination, sparse
symbol distribution, corrections, and shared network behavior remain pending
under the separately authorized procedure below.

## Market-hours validation after deterministic success

Deterministic success authorizes no credential or provider request. The next
step is a separately authorized market-hours run for one exact trading date,
using the existing live hydration diagnostic with these corrections:

- use the unprobed control measurement path from the contention reproducer;
- run live-only first, then live plus one REST worker;
- continue to two/four/eight workers only while the preceding level is stable;
- keep the current cache-only reference-data preflight and redaction rules;
- make only one trial per permitted worker level; and
- stop immediately at the first safety threshold or integrity failure.

That run answers what the fake sources cannot: actual provider burst shape,
REST latency and rate limits, pagination, sparse symbol distribution, live
corrections, and shared network behavior. It still does not establish a public
provider SLA or an extreme 6,000-aggregates-per-second capacity claim.

## Completion decision

The scanner is ready for a bounded live operating trial when:

1. deterministic live plus one-worker full-universe hydration passes;
2. a safe local worker count is selected from the conditional sweep; and
3. the separately authorized market-hours live-only and one-worker trials pass
   without queue growth, stale currentness, false readiness, or accounting
   divergence.

If deterministic one-worker hydration fails, fix only the lowest failing
combined path and rerun that level. If deterministic tests pass but the real
one-worker trial fails, use the retained numeric evidence to distinguish
provider/network latency from local CPU, queue, or engine contention before
changing production behavior.
