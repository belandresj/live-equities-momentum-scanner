# Live watermark-stall diagnostic

**Status:** Accepted focused diagnostic correction on 2026-08-20 after owner-run
evidence invalidated the original one-second trigger.

**Parent authority:** The current [Live feature-set MVP program](live-feature-mvp-program.md)
and the accepted scanner-stability corrections. This document owns only the
bounded causal evidence and private persistence described here; it does not
create a new component, readiness owner, watermark, evaluator, or publication
path.

## 1. Outcome and fixed boundary

When the ordinary live scanner observes the exact transition

```text
previous backend_ready=true
current backend_ready=false, reason=watermark_stale
```

the process records one bounded, chronological tail of recent ordinary
evaluation-cycle evidence and makes one best-effort private JSON persistence
attempt. The record is diagnostic evidence for separating late live-coverage
fence support, evaluator work, engine/queue delivery, and allocation/GC
pressure. It does not change readiness, the four-second evaluation delay, the
two-second readiness tolerance, provider behavior, ranking, publication
cadence, API capture, or UI behavior.

The fixed Phase 1 boundary is:

- `PG-OPS-02`, `PG-OBS-03`, `PG-AVAIL-03`;
- `ARCH-OWN-01` through `ARCH-OWN-04` and `ARCH-FLOW-01`, `ARCH-FLOW-02`,
  `ARCH-FLOW-04`;
- `DTE-CLOCK-03` through `DTE-CLOCK-06`, `DTE-COMMIT-01`,
  `DTE-COMMIT-02`, `DTE-COMMIT-04`, and `DTE-REJECT-01`; and
- `LIFE-LIVE-02`, `LIFE-LIVE-05`, `LIFE-END-02`, `LIFE-END-03`,
  `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`.

The accepted readiness/operations dependency remains controlling for the
meaning of `watermark_stale`, the cached process sample, fixed-cardinality
measurements, and observability-local failure containment.

## 2. Ownership, inputs, and exclusions

`internal/engine` remains the sole owner of evaluation timing/source/target and
immutable publication facts. `internal/operations.Runtime` owns the fixed
120-record cycle ring, the fixed-cardinality first-transition latch, and
combines engine boundary facts with the existing cached process sample.
`cmd/scanner` owns only latch observation and the one-shot file recorder. The
API and UI do not read or invoke the recorder.

Inputs are existing engine timing/publication/T/Q views, runtime queue and
delivery metrics, and the asynchronously cached memory/goroutine/GC sample.
The ring retains no symbols, provider frames, payloads, credentials, URLs,
environment values, or unbounded history.

Explicit non-scope: market semantics; readiness derivation; thresholds;
evaluation scheduling; provider transport; T/Q policy; API schema or capture;
UI state; replay; checkpoints; restart/recovery policy; a second engine/state
owner; a worker, queue, database, logging framework, service, public endpoint,
or live/provider validation.

## 3. Exact behavior and persistence boundary

- The ring has capacity 120, overwrites its oldest record, and snapshots in
  chronological order. A cycle record is fixed-cardinality and records cycle
  start/completion/duration, fence disposition, selected timer policy,
  evaluation source/target/stage/apply/publication durations, watermark before
  and after, causal target and lag before/after, publication and engine
  progress, queue frames/bytes/oldest age, processing-delay views, T/Q
  pressure mode/cause, and the cached heap/goroutine/GC scalars.
- Every runtime status derivation records its fixed-cardinality result in one
  lock-free diagnostic latch. This includes the exact sealed capture used by
  `/readyz`, so a crossing observed by the launcher cannot be erased by the
  scanner's independent timer phase. The atomic CAS is the observation's
  linearization point: a stale capture preempted before that CAS records after
  a newer ready observation and therefore still retains the ready-to-stale
  crossing before it can return that stale response. Capture performs one
  bounded atomic CAS;
  it does not allocate an incident tail, touch the filesystem or terminal,
  invoke the recorder, alter the response, or wait for diagnostic work.
- A dedicated scanner-local observer checks the existing lightweight status
  path and drains the shared latch every 100 milliseconds. This cadence controls
  persistence delay only; correctness does not require a tick inside the stale
  interval. The one-second operator renderer remains presentation-only.
- The observer triggers once on the exact ready-to-`watermark_stale`
  transition, never on startup, hydration, recovery, replay, controlled
  shutdown, transport failure, T/Q-only pressure, another readiness reason, or
  repeated stale samples.
- One process records at most one watermark-stale incident. The latch is set
  before persistence, so a persistence failure cannot cause retries through
  later samples or affect runtime cancellation, shutdown, readiness, or
  engine suppression.
- The file is versioned JSON at
  `var/diagnostics/watermark-stale-<UTC timestamp>.json`. Its directory is
  `0700`, its file is `0600`, publication uses create-without-overwrite after
  complete temporary-file encoding/sync, and the schema has at most 120 cycle
  records.

GC facts come from the existing cached `runtime.MemStats` sample; this
diagnostic adds no per-cycle `ReadMemStats` call.

## 4. Trust boundary and limitations

Success means the runtime observed the exact status transition, retained it
across later ready samples, and the file contains the bounded ring records
whose completion did not follow the trigger timestamp. Invalid, missing, or
failed persistence is contained as
observability failure and never becomes a readiness or lifecycle result. A
file proves timing relationships observed by this process; it does not prove
provider chronology, identify a unique root cause when planes overlap, or
establish live stability or a performance SLA.

The dangerous false success is a file emitted for a recovery/API/T/Q event or
with provider/symbol data. Construction prevents the recorder from being
called by API capture and requires the engine-owned live mode/lifecycle and
exact readiness reason. JSON mapping uses an explicit allowlist rather than
marshalling runtime or engine structs.

## 5. Primary proofs and slices

The one implementation slice owns `internal/engine`'s fixed timing observation
reuse, `internal/operations`' ring and cycle composition, and `cmd/scanner`'s
transition/persistence path. Its primary proof `P-WATERMARK-STALL` covers:

1. fixed capacity, wraparound, chronological snapshots, and detached bounded
   records;
2. exact transition trigger, one-file bound, recovery/repeated-stale bound,
   and all excluded states;
3. deterministic slow-evaluator, late/unsupported-fence, queue/delivery-delay,
   and normal-cycle distinctions;
4. nonfatal latched persistence failure and absence of runtime/API/readiness
   side effects;
5. protected versioned create-without-overwrite JSON with no prohibited data;
   and
6. concurrent append/snapshot behavior with no partial records or engine/API
   blocking.

Focused short tests run first, followed by the repository short tier, the
affected race tier, focused vet, and diff checking. No provider/live test or
expensive retained-tail/capacity suite is part of this diagnostic.

## 6. Reconnaissance and reuse record

No Version 2 source or fixture is required. The current whitelist is the
existing `internal/engine` evaluation timing/publication view, the
`internal/operations` cached diagnostics and ingress-ring mechanics, and the
`cmd/scanner` terminal-safe diagnostic persistence convention. Reuse is by
adaptation: preserve sole engine ownership, cached sampling, protected
temporary-file publication, and nonfatal terminal output; reject ingress
incident schema, provider-facing evidence, and any second lifecycle owner.

## 7. Acceptance record

The original implementation and focused deterministic verification completed
on 2026-08-20, but its acceptance claim was invalidated by the first owner-run
observation. The launcher sampled `/readyz` on its one-second timer and printed
`ready -> watermark_stale -> ready`; the scanner-local recorder, running on an
independent one-second operator timer, sampled on a different phase and emitted
neither a file nor its persistence result. Persistence was never attempted.

The first attempted correction separated transition observation from operator
rendering but relied on a 100-millisecond sampler. Final read-only review found
that this remained probabilistic: a 50-millisecond stale interval could be
visible to the launcher while falling between observer ticks. That finding
reopened the correction and invalidated the denser-sampling proof.

The corrected design retains the first exact ready-to-stale crossing in the
runtime path that derives readiness, then lets the 100-millisecond observer
persist it later. Its deterministic phase-offset regression constructs a
50-millisecond stale interval: scanner samples at 0, 1, and 2 seconds all
remain ready, a launcher capture at 1.25 seconds observes stale, and a later
scanner observation after recovery still receives and persists exactly that
crossing. The production-path runtime regression proves `CaptureSnapshot`
records the same sealed status that `/readyz` maps. This proves the missed-
sampler defect without making persistence or recorder execution an HTTP side
effect and without a provider request.

Focused re-review then found an out-of-order concurrency counterexample: a
stale capture could be preempted before recording, a newer ready sample could
record first, and timestamp rejection could discard the stale result even
though the HTTP response would later expose it. The latch now linearizes by
atomic record arrival rather than rejecting an older sample timestamp. The
regression executes `ready(t1) -> recovered-ready(t3) -> stale(t2)` arrival
order and retains the stale crossing before the stale capture can return.

The primary proof is covered by:

- `go test -count=1 -short -timeout 2m ./internal/operations -run TestWatermarkStall`;
- `go test -count=1 -short -timeout 2m ./cmd/scanner -run TestWatermarkStale`;
- `TestWatermarkStallRingCapacityWrapAndDetachedChronology`;
- `TestWatermarkStallRingConcurrentSnapshotsNeverExposePartialRecords`;
- `TestWatermarkStallCycleEvidenceDistinguishesFailurePlanes`;
- `TestWatermarkStaleRecorderExactTransitionPersistsOneBoundedIncident`;
- `TestWatermarkStaleRecorderOneProcessBoundAcrossRepeatedStaleAndRecovery`;
- `TestWatermarkStaleRecorderExcludesNonOrdinaryOrNonWatermarkStates`;
- `TestWatermarkStaleSharedTransitionLatchCatchesPhaseOffsetTransientMissedByOneSecondScannerSamples`;
- `TestWatermarkStaleTransitionLatchRetainsShortCrossingAcrossSamplerPhases`;
- `TestC8RUNTIME01LifecycleReadinessShutdown` (sealed capture retains the exact crossing);
- `TestWatermarkStalePersistenceFailureIsNonfatalAndLatched`; and
- `TestWatermarkStalePersistenceIsProtectedAndCreateWithoutOverwrite`.

The repository short tier passed with `go test -count=1 -short -timeout 2m ./...`;
the corrected transition boundary passed
`go test -count=1 -race -short -timeout 5m ./internal/operations ./cmd/scanner`;
focused vet passed for those packages; and `git diff --check` passed. Final
read-only review first found and caused correction of the phase-width and
out-of-order-capture P1s, then reported no remaining findings.

No provider/live request, replay, checkpoint, retained-tail capacity, or
market-hours stability proof is allocated. The transition latch is in memory,
so a process crash between retention and the next at-most-100-millisecond drain
can still lose the file. A stale interval observed by neither `Runtime.Status`
nor any sealed snapshot capture remains unknowable. This diagnostic identifies
a causal failure plane but does not fix watermark delay or establish live
stability.
