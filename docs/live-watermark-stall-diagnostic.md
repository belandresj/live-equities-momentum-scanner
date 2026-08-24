# Live watermark-stall diagnostic

**Status:** Accepted baseline correction, semantically forward-ported into the
live-backend-replacement architecture after accepted `LBR-A3` and before
inactive `LBR-D1` on 2026-08-24. Offline replacement proofs and final read-only
review pass. No replacement-branch provider run has exercised the corrected
diagnostic.

**Parent authority:** The current
[live backend replacement program](live-backend-replacement/delivery-program.md),
the [replacement architecture](live-backend-replacement.md), and compatible
accepted scanner-stability evidence. This document owns only the bounded causal
evidence and private persistence described here; it does not create a new
component, readiness owner, watermark, evaluator, or publication path.

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
- The runtime retains one fixed-cardinality active-cycle view while an ordinary
  cycle is in progress. It distinguishes live-coverage capture/enqueue,
  ordered coverage completion, timer admission/completion, and combines that
  phase with the engine's fixed-cardinality active aggregate-evaluation phase
  (`maintenance`, `stage`, `apply`, or `publication`). When the readiness path
  retains a stale crossing it also retains the then-observed active view with
  that view's own observation timestamp. The status sample and active
  observation are ordered but intentionally do not pretend to be one atomic
  engine snapshot. The view retains no event, symbol, goroutine stack, or
  unbounded history.
- A completed ordinary live-coverage cycle correlates evaluation timing with
  the live-coverage disposition's engine sequence. Its following
  maintenance-only timer sequence is not the evaluation owner.
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

### Replacement forward-port after `LBR-A3`

The corrected baseline behavior was preserved in commit
`e88eef766940bcc953fec2e4b1bc8f213c3ef7a0` on
`codex/live-backend-stability-baseline`, whose parent is replacement baseline
`0d043c16cea43e3739abe11c766e3496b3ea0fcd`. The replacement branch uses that
commit as semantic provenance rather than merging or cherry-picking it.

The forward-port retains one fixed-cardinality active observation and maps its
phases onto the accepted replacement evaluator: owner-local symbol/qualification
maintenance, compact selection plus selected-row enrichment staging, candidate
application, and the sole immutable publication. Completed live-coverage timing
is sealed with the disposition's actual engine sequence before the following
maintenance-only timer can overwrite singleton timing. The runtime separately
records live-coverage enqueue and ordered-completion durations and retains the
exact ready-to-`watermark_stale` crossing through the existing shared latch.

This is diagnostic-only. It adds no canonical aggregate copy, qualification
clone, full-population legacy evaluator, second publication owner, ingress
handoff, readiness input, or D1 decoder/interface change. The B3 source-
exclusion proof permits only the diagnostic atomic pointer and package-private
phase-pause seam while continuing to reject superseded evaluator/state owners.
Final review found that live aggregate trust-correction cycles could still run
the replacement full-population selection/enrichment path without an active
view while ordered live-coverage completion waited. The correction now starts
an explicit `trust_correction` active stage before that production scan and
tracks its apply/publication phases. Focused re-review then found the adjacent
same-`T` selected-row trust-closure candidate had the same omission; it now
uses the same source/phase path. Deterministic production regressions hold and
observe stage/apply/publication for both the full selection repair and selected-
row trust closure. A second focused re-review found that the first selected-row
timing seal also advanced B's semantic trust-correction coalescing identity.
Timing capture is now separate from that revision/latch, and the production
regression proves both remain unchanged. Replacement short, repository, race, vet,
and diff proofs pass, and focused re-review reports `CLEAN/PASS` with no
remaining finding.

The 2026-08-24 incident below remains historical evidence from baseline commit
`0d043c16cea4`, not evidence produced by the replacement architecture. Its
ignored runtime JSON remains outside Git. No provider request or private
scanner run exercised the forward-ported replacement binary, so an actual
replacement ready-to-`watermark_stale` attribution remains unconfirmed.

### 2026-08-24 live-evidence correction

An owner-authorized ordinary scanner run on revision `0d043c16cea4`, explicit
eight-worker hydration, completed all 5,566 work items and reached ready. In a
four-minute post-hydration observation it crossed
`ready -> watermark_stale -> ready`: the causal-target lag reached three
seconds, the provider connection stayed on epoch 1, recovery attempts remained
zero, accounting remained valid, and the scanner recovered without
intervention after approximately four seconds. The protected incident is
`var/diagnostics/watermark-stale-20260824T172145.288951000Z.json`.

That evidence invalidates two lower-level diagnostic claims while preserving
the transition latch and bounded persistence claims. First, the final retained
cycle completed at `17:21:42.734195Z`, while the stale observation occurred at
`17:21:45.288951Z`; the in-progress cycle overlapping the crossing was absent
because the ring records only completions. Second, all 87 retained ordinary
cycles reported `evaluation_observed=false`: correlation compared the
live-coverage-owned evaluation with the later maintenance timer or historical
ingress-fence sequence instead of the live-coverage disposition sequence.
Consequently the accepted file proved a delayed ordinary cycle but could not
separate ordered coverage wait from evaluator stage/apply/publication.

The narrow correction retains the active runtime and engine evaluation views
as one coherent bounded observation with its own timestamp after the status
sample; it does not claim that status and phase are one atomic snapshot. It
maps that observation into the existing one-shot protected document and
correlates completed live-coverage timing with its actual disposition. It does
not change the cycle, queue, evaluator, watermark, readiness tolerance,
publication, provider, API, or UI. Primary correction proofs deterministically
hold each active phase, require timestamped phase attribution when the
readiness path retains the incident, require live-coverage
maintenance/stage/apply/publication timing to be observed, and preserve the
existing exclusion, fixed-capacity, concurrency, and nonblocking proofs.

The first independent correction review found four distinguishing
counterexamples. The readiness sample timestamp could predate a later phase
load; full-universe symbol maintenance occurred before the active stage;
singleton evaluation timing could be overwritten between live-coverage
completion and the following timer; and the retained aggregate-ingress fence
timing fields had been inadvertently zeroed. The correction now timestamps the
active observation independently, marks maintenance before its production
scan, seals evaluation timing into the live-coverage disposition at engine
completion, and preserves the prior ingress-fence timing fields alongside the
new live-coverage enqueue/completion durations. Deterministic regressions cover
preemption between snapshot sampling and readiness observation, an actually
held production maintenance scan, overwritten later singleton timing, and the
retained fields.

Focused short tests passed for `internal/engine`, `internal/operations`, and
`cmd/scanner`, including the active maintenance/stage/apply/publication,
ordered-wait, preemption, sealed-correlation, persistence, exclusion, and
bounded-ring regressions. The complete affected packages passed the short tier;
the repository-wide `go test -count=1 -short -timeout 2m ./...` tier passed;
the affected race tier passed with a five-minute timeout; focused vet and
`git diff --check` passed. The first independent review's three P1 and one P2
findings were corrected; focused re-review then reported `CLEAN/PASS`. These
proofs establish diagnostic attribution and noninterference offline. They do
not establish that the live latency source has been identified until another
separately authorized incident or clean observation exercises the corrected
binary.

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
