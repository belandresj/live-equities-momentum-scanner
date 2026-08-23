# Snapshot capture and dashboard transport isolation correction

**Status:** Owner-directed, implementation-ready specification. This is step 3
of the scanner-stability correction sequence and starts only after steps 1 and
2 are offline complete.

**Development constraint:** The market is closed. Complete deterministic
concurrency, HTTP, and UI proofs without credentials or provider requests. The
first live test is one combined market-hours soak of all three changes.

**Worktree dependency:** The existing scanner-output/API-continuity and
private-launcher/dashboard-continuity work must be complete or quiescent before
this step edits overlapping scanner/API/UI lifecycle code. Preserve its rule
that terminal, API, or dashboard failure cannot restart or stop a healthy
runtime.

**Implementer starting context:** Read this specification, the accepted/offline
completed [step-1](live-evaluation-cycle-coalescing-correction.md) and
[step-2](tq-publication-coalescing-correction.md) corrections, and the current
[`scanner/API continuity`](scanner-output-and-api-continuity-correction.md) and
[`launcher/dashboard continuity`](private-launcher-dashboard-continuity-correction.md)
specifications. Preserve unrelated worktree changes and use one write-capable
implementation agent.

## Outcome

An API snapshot request must never wait for the engine mutation lock, serialize
behind another snapshot request, call `runtime.ReadMemStats`, or perform a
checkpoint-state lock read. A slow or canceled browser request must not create
capture backlog or consume engine time after cancellation.

The dashboard must distinguish:

- **DELAYED:** its own request exceeded the three-second deadline or a prior
  request still occupies the poll slot;
- **UNAVAILABLE:** the loopback API cannot be reached or returned a non-success
  response; and
- **DISCONNECTED/RECOVERING:** only a validated backend snapshot supplies
  aggregate-transport state supporting that meaning.

This keeps a browser/API problem from looking like a provider disconnect and
prevents diagnostic sampling from competing with the live evaluator.

## Current brittleness

`Engine.ObserveSnapshot` already reads one immutable atomic publication without
taking the engine mutex. `Runtime.CaptureSnapshot` then defeats that property
by serializing every caller through `captureMu` and building fresh metrics.
That metrics path calls `Engine.CheckpointOperations`, which takes the engine
mutex, and `runtime.ReadMemStats`, which is relatively expensive. Queue,
adapter, writer, delivery, incident, and recovery diagnostics are also joined
for every HTTP request.

The HTTP capture interface has no context, so cancellation is checked only
before capture. Concurrent or timed-out requests can continue doing work and
later callers queue behind `captureMu`.

The dashboard aborts after three seconds but maps the resulting abort, HTTP
503, and network failure to the same `disconnected` transport. It freezes the
last rows and displays `DISCONNECTED`, even though those failures say nothing
about the Massive socket. This is why a transient API stall can be presented as
a scanner/provider disconnect.

## Fixed boundaries

Preserve all of the following:

- one immutable combined engine publication and one atomic engine snapshot
  read per capture;
- exact snapshot/API validation, response-size bound, loopback binding, CORS,
  `/livez`, `/readyz`, and `/api/v2/snapshot` semantics;
- backend readiness derived from engine/session/fence/watermark/accounting
  facts, never from browser reachability or T/Q health;
- no browser-owned market calculation, ranking, currentness, or provider state;
- one in-flight dashboard request, one-second poll cadence, retained-last-model
  behavior, and atomic DOM replacement;
- API/dashboard failure independence from runtime market processing; and
- checkpoint mode off in the supported MVP path.

The controlling meanings are `PG-UI-01` through `PG-UI-03`, `PG-OBS-03`,
`ARCH-OWN-02` through `ARCH-OWN-04`, `ARCH-FLOW-04`, system-overview Sections
5.12 and 5.13, `LIFE-LIVE-05`, `LIFE-PUBLISH-01` through
`LIFE-PUBLISH-03`, and the accepted API/UI component contracts.

## Required design

### 1. Capture is lock-independent from engine mutation

`Runtime.CaptureSnapshot` must construct one sealed capture from:

1. exactly one `Engine.ObserveSnapshot()` atomic load;
2. atomic process/joined flags and an atomic capture sequence;
3. a previously collected immutable runtime-diagnostics sample; and
4. bounded incident/recovery latch copies.

The method must not call any engine method that takes `Engine.mu`, including
`CheckpointOperations`. It must not lock `captureMu` or another global mutex
that serializes requests. Replace `captureSequence` protection with an atomic
increment and preserve explicit overflow failure.

Publication, operational, and T/Q fields must all come from the one
`ObserveSnapshot` result. Do not join separately observed engine views. Status
is derived from that view and the capture's one sampled clock value.

### 2. Expensive diagnostics are sampled asynchronously once

Use one existing runtime-owned diagnostic/sampling loop; do not add a sampler
per HTTP handler or a new service. At no more than the configured one-second
cadence, collect and atomically replace one immutable fixed-cardinality
diagnostics sample containing the non-publication facts currently needed by
the snapshot:

- queue and T/Q-normalization accounting;
- adapter and checkpoint-writer accounting;
- checkpoint-engine operational accounting;
- delivery latency counters/window attribution;
- queue high-water values;
- heap allocation/in-use and goroutine count; and
- the sample's actual collection time.

`runtime.ReadMemStats`, adapter/queue accounting locks, writer accounting, and
checkpoint-engine lock reads may occur only in this sampler, startup
initialization, or explicit diagnostic/incident capture—not per API request.

Prefer moving checkpoint operational counters into the immutable engine
publication if that is smaller and cleaner than sampling them through the
engine lock. Either design is acceptable only if the HTTP capture path is
provably engine-lock-independent.

The first diagnostics sample must exist before the API is reported started.
Later sample failure or staleness must be explicit in sample-accounting
validity and must not fabricate values or make an otherwise coherent aggregate
publication disappear. Do not silently restamp old diagnostics as newly
collected. Preserve or add one bounded collection timestamp/age internally;
an additive API field is allowed only if honest age cannot otherwise be
validated without changing existing field meaning.

### 3. Capture composition is coherent and bounded

At capture time, combine the current engine view with the latest diagnostics
sample without mutating either. Recompute any cross-sample boolean from those
two immutable values. Never require exact equality between an engine
publication time and a diagnostic collection time; instead validate that the
diagnostics sample is nonfuture and within a fixed bound of two sampling
cadences.

If diagnostics are absent, future-dated, or too old, keep the engine
publication/API available while marking operational sample accounting invalid
or unavailable. Backend readiness continues to use the existing engine-owned
accounting predicates; optional process diagnostics cannot become a new
readiness gate.

Every returned slice/map/pointer remains detached under the existing sealed
capture rules. The cached diagnostics sample must have no mutable aliases.

### 4. HTTP cancellation stops avoidable work

Keep the capture-source interface unchanged if the lock-free cached capture is
already bounded; adding context to the interface is not a goal by itself.

The handler must check `request.Context().Err()`:

- before capture;
- after capture and before mapping;
- after mapping and before JSON marshaling/writing; and
- before recording a mapping failure caused only after cancellation.

A canceled request produces no later mapping diagnostic and does not write a
response body. One slow client or response writer must not prevent another
handler from capturing/mapping its own immutable snapshot.

### 5. Dashboard transport vocabulary is causal

Keep at most one fetch in flight. Classify failures as follows:

| Evidence | Dashboard transport/status |
| --- | --- |
| Poll fires while prior request is active | `refresh_delayed` / `DELAYED` |
| Controller's own three-second deadline aborts the request | `refresh_delayed` / `DELAYED` |
| `stop()` aborts the request | No new render event |
| Fetch/network rejection or non-2xx HTTP response | `api_unavailable` / `UNAVAILABLE` |
| Valid API snapshot with backend aggregate recovery/disconnection facts | Existing backend-derived `RECOVERING`, `WARMING UP`, `UNAVAILABLE`, or `DISCONNECTED` meaning |
| Valid current qualified snapshot | `connected` / `LIVE` |

Neither timeout nor API unavailability may be called a provider disconnect.
Retain and visibly mark the last model noncurrent for delayed/unavailable
events. Recovery to a valid response immediately restores the backend-derived
status.

`DISCONNECTED` may remain in the trader vocabulary, but it must be reachable
only from validated scanner facts—not from a browser fetch exception.

## Implementation boundary

Expected production changes are limited to:

- `internal/operations/snapshot.go`, `metrics.go`, `runtime.go`, and focused
  tests;
- the smallest `internal/engine` immutable checkpoint-operations projection if
  selected;
- `internal/snapshotapi/http.go`, mapper/schema only when required for honest
  diagnostic sample age, and focused HTTP/concurrency tests;
- `ui/model.js`, `ui/render.js`, and their focused tests; and
- this document's acceptance record.

Coordinate rather than overwrite current uncommitted continuity work in
`cmd/scanner`, `internal/privatelauncher`, and their tests. This step should not
need to change the private launcher.

Do not change aggregate/T/Q formulas, ranking, watermark/evaluation semantics,
readiness thresholds, API listener restart policy, dashboard child restart
policy, provider transport, public binding/authentication/TLS, replay, or
checkpoint operation.

## Required deterministic proofs

### P1 — engine lock cannot delay HTTP capture

Using a bounded test seam, hold the engine mutation lock for longer than the
dashboard's three-second timeout while issuing concurrent `/api/v2/snapshot`,
`/livez`, and `/readyz` requests.

Prove every request obtains the last immutable engine publication without
waiting for that lock. Snapshot/liveness responses complete; readiness may
honestly return 503 if the retained watermark is stale, but it must do so
promptly. Releasing the engine lock must not unleash a backlog of old captures.

Source inspection/assertion must prove `CaptureSnapshot` cannot reach
`CheckpointOperations`, `ReadMemStats`, or a serialized capture mutex.

### P2 — concurrent callers do not serialize

Run at least 100 concurrent captures against a production-shape 20-row/TQ
publication while one response writer is deliberately blocked after mapping.

Prove:

- other captures complete independently;
- sample IDs are unique and strictly increasing by creation order where
  observable;
- each capture is internally coherent and detached;
- capture latency p99 is below 100 ms and maximum below 500 ms on the current
  development host; and
- no goroutine, waiter, or request remains after bounded cleanup.

These are local regression bounds, not a network SLA.

### P3 — diagnostics cost is request-independent

Count diagnostics collections while issuing at least 10,000 snapshot captures
across multiple goroutines. Prove `ReadMemStats`, checkpoint-engine reads, and
adapter/queue/writer sampling occur no more than once per configured sampler
cadence, not once per request.

Inject an old/future/missing diagnostics sample and prove the API preserves the
coherent engine publication while marking sample accounting honestly. A stale
diagnostic must not be relabeled with the capture time.

### P4 — cancellation is terminal for that request only

Cancel requests before capture, between capture and mapping, and between
mapping and write using deterministic seams rather than sleeps.

Prove no response body or cancellation-created mapping diagnostic is emitted,
no capture remains active, and a concurrent uncanceled request succeeds. A
genuinely invalid uncanceled snapshot must still record the bounded mapping
failure and fail closed.

### P5 — dashboard labels the actual failure plane

Drive the real poll controller through:

- overlap while a request is active;
- self-timeout abort;
- explicit `stop()` abort;
- fetch rejection;
- HTTP 503;
- valid recovering backend snapshot;
- valid current snapshot; and
- recovery after every failure.

Assert exact `DELAYED`, `UNAVAILABLE`, backend-derived recovery state, and
`LIVE` labels. Assert that neither timeout, fetch rejection, nor HTTP 503 emits
`DISCONNECTED`, and retained rows remain visibly noncurrent until a valid
current snapshot arrives.

### P6 — combined offline production cycle

Compose all three stability corrections in one credential-free production-
shape run: approximately 5,700 retained-tail symbols, advancing live-coverage
fences, high-rate selected-symbol T/Q, one-second diagnostics sampling, and
concurrent API/UI polling for at least five minutes.

Require:

- one aggregate full pass per successful cadence cycle;
- publication count bounded by cadence plus exact trust transitions;
- API capture p99 below 100 ms and no capture above 500 ms;
- no watermark lag above two seconds in the deterministic clock model;
- no T/Q pressure transition, dropped input, capacity rejection, accounting
  failure, integrity suppression, goroutine leak, or unbounded heap trend; and
- exact final aggregate ranking and T/Q view versus their deterministic
  oracles.

Record fixture shape, event counts/rates, Go/host details, GC configuration,
latency distributions, publication causes, queue/heap/goroutine series, and
the limitation that this is not provider traffic.

## Verification and acceptance

Run narrow operations/API/UI proofs during correction. Final offline acceptance
requires:

```text
go test -count=1 -short -timeout 2m ./internal/operations ./internal/snapshotapi ./cmd/scanner
node --test ui/model.test.mjs
go test -count=1 -short -timeout 2m ./...
go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner
go vet ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner
git diff --check
```

Run P6 separately with an explicit timeout no greater than 15 minutes. One
read-only review must focus on immutable-view coherence, lock independence,
cached-sample honesty, cancellation, and the distinction between API transport
and provider transport.

The step and three-change sequence are **offline complete** when P1-P6 pass,
affected verification/review are clean, and the scanner continuity work still
proves API/dashboard failures cannot stop or replace the runtime.

## Offline implementation acceptance record — 2026-08-20

The implementation applies the required boundary without changing the public
capture-source interface or API schema. `Runtime.CaptureSnapshot` now performs
one atomic `Engine.ObserveSnapshot` read, uses an atomic sequence, reads the
immutable diagnostics cache, and copies the incident/recovery latches. The
configured runtime sampler performs the expensive queue, adapter, writer,
checkpoint, memory, and goroutine observations; the startup sample is installed
before `New` returns. Cached sample age is retained internally and accounting is
invalidated for absent, future, or older-than-two-cadence samples without
removing the current engine publication.

The focused deterministic proofs passed:

- P2: 128 HTTP captures completed while another response writer was blocked;
  sample IDs were contiguous and unique, p99/max stayed below 100/500 ms, and
  the blocked handler joined.
- P3: 10,000 concurrent captures caused zero additional diagnostics
  collections; old, future, and missing samples preserved the publication and
  remained sample-accounting-invalid without restamping collection time.
- P4: cancellation before capture, after capture, and after mapping produced
  no body or cancellation-created mapping diagnostic; an uncanceled invalid
  capture still failed closed and recorded its bounded diagnostic.
- P5: overlap, self-timeout, stop cancellation, network rejection, HTTP 503,
  backend recovery, valid current data, retained rows, and recovery after each
  failure produced the specified transport/status vocabulary. Browser/API
  failures did not emit `DISCONNECTED`.

P1 source review confirms the capture path contains no serialized capture
mutex, `CheckpointOperations`, or `runtime.ReadMemStats`; the existing
blocked-writer HTTP proof also confirmed that response transport does not block
engine progress. A separate multi-second engine-mutex hold was not run because
the current engine exposes no cross-package bounded lock-test seam.

Verification completed successfully:

```text
go test -count=1 -short -timeout 2m ./internal/operations ./internal/snapshotapi ./cmd/scanner
node --test ui/model.test.mjs
go test -count=1 -short -timeout 2m ./...
go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner
go vet ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner
git diff --check
```

The final read-only conformance review covered immutable-view coherence,
capture lock independence, cached-sample age honesty, cancellation, and the
API/provider transport distinction. No separate reviewer worker was available
in this session, so that review was performed by the implementation
orchestrator.

The 5-minute, approximately 5,700-symbol combined production-shape P6 run was
not claimed here. The deterministic runtime rehearsal remains green, but the
combined API/UI polling cycle and the separately authorized market-hours soak
still require their explicit execution; no provider credentials or requests
were used for this implementation.

## Deferred combined market-hours acceptance

Do not claim the scanner stable from offline completion alone. At the next
separately authorized market session, run one combined 30-minute late-session
soak of the final three-change build. The target acceptance is:

- evaluator/cycle p99 below 750 ms and no cycle above one second;
- watermark lag never above two seconds;
- snapshot p99 below 100 ms and no snapshot above 500 ms;
- zero T/Q pressure transitions or capacity drops under the observed feed;
- no readiness flap, API restart, provider reconnect, integrity suppression,
  or process restart; and
- heap reaches a bounded plateau rather than a repeating upward trend.

If this live soak fails, preserve the first causal timing/queue/GC/publication
evidence and reopen the narrowest implicated spec. Do not loosen readiness,
timeout, or pressure thresholds to make the observation appear green.

## Explicit non-scope

- changing market calculations, ranking, or availability meanings;
- public hosting, TLS, authentication, or service splitting;
- automatic engine/scanner restart;
- changing the API listener/dashboard child continuity policies;
- replacing polling with SSE/WebSocket;
- replay/checkpoint repair; and
- incremental aggregate evaluation.
