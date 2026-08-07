# Readiness and operations

**Status:** Component 8 finally accepted for the private/local V1 RC

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; current plan
and later lower-level decisions remain revisable through the program correction
loop

**Controlling Phase 1 requirements:** `PG-OPS-01`, `PG-OPS-02`,
`PG-OBS-01`, `PG-OBS-02`, `PG-OBS-03`, `PG-AVAIL-01`,
`ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-03`, `ARCH-OWN-04`,
`ARCH-FLOW-01`, `ARCH-FLOW-02`, `ARCH-FLOW-03`, `ARCH-FLOW-04`,
`DTE-CLOCK-03`, `DTE-CLOCK-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-COMMIT-01`, `DTE-COMMIT-02`, `DTE-COMMIT-04`,
`DTE-REJECT-01`, `LIFE-INIT-05`, `LIFE-HYDRATE-05`,
`LIFE-HYDRATE-06`, `LIFE-LIVE-01`, `LIFE-LIVE-02`,
`LIFE-RECOVER-01`, `LIFE-RECOVER-04`, `LIFE-RECOVER-05`,
`LIFE-RECOVER-06`, `LIFE-END-01`, `LIFE-END-02`, `LIFE-END-03`,
`LIFE-PUBLISH-01`, `LIFE-PUBLISH-02`, and `LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Components 1-6 and corrected/final
Component 7 interfaces

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C8 boundary plan;
Sections 8-19 are completed here after its just-in-time V2 reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C8 boundary, current detailed contract when added, proofs, slices, and sole delivery ledger | Sections 1-7 approved; Sections 8-19 pending | Every C8 task | Components 1-7 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `accepted` | Direct owner V1 program revision; C7 finally accepted; recorded V2 scope inspected | Complete |
| Completed contract | `accepted_current_plan` | Two slices and two primary proofs preserve the existing engine-owned lifecycle/readiness boundary; no new competing owner or consequential interface requires pre-implementation review | Implement `C8-S1` |
| `C8-S1` runtime/lifecycle | `accepted` | `P-C8-RUNTIME` covers initial bootstrap retry/exhaustion, two live-gap-live recoveries followed by exact post-live exhaustion, checkpoint pre-plan false-ready rejection, active-live shutdown join, and timeout without a false joined claim; affected race clean | Complete |
| `C8-S2` measurements/load | `accepted` | `P-C8-LOAD`: 6,000-symbol bound population, 100 active aggregate symbols, 20 corrections, 20 duplicates, 20 rejects, and 60 T/Q-deferred facts; 53.58 s end-to-end and exact accounting | Complete |
| Final component review | `accepted` | Focused final re-review found no remaining P1/P2 finding after the recovery, shutdown, and checkpoint false-ready corrections; ordinary and affected race suites passed uncached. Unchanged non-short load evidence was reused. | Begin C9 contract completion |

## 1-4. Outcome, scope, ownership, and settled boundary

C8 turns Components 1-7 into one runnable backend composition with explicit
configuration, lifecycle control, readiness, degradation/staleness, bounded
recovery and shutdown, and operator measurements. It completes the production
aggregate-lifecycle milestone without claiming live-provider or multi-host
capacity.

In scope:

- compose the existing binding, adapter, hydration/recovery, engine, replay,
  checkpoint, snapshot, and process-control boundaries into a runnable Go
  scanner;
- derive process-live, backend-ready, ranking-current, stale/unavailable, and
  field-status output from engine-owned facts without another readiness owner;
- choose evidence-backed evaluation/freshness, queue, retry, recovery, and
  shutdown thresholds that protect aggregate correctness and avoid false-ready
  output;
- measure aggregate/TQ ingress and disposition throughput, processing delay,
  committed-watermark lag, queue occupancy/growth, memory, drops, rejections,
  pressure shedding, recovery, and checkpoint status with bounded cardinality;
  and
- prove local V1 behavior under deterministic mixed load at 6,000 symbols.

Not in scope: T/Q feature/pressure policy owned by C9; API serialization and
HTTP status owned by C10; UI; credentials/live observation; databases,
multi-host/HA operation, public monitoring infrastructure, exhaustive soak, or
capacity claims beyond the recorded local host.

`ScannerStateEngine` remains the only readiness and lifecycle state owner.
Runtime adapters, signal/supervisor wiring, samplers, and metrics collectors
return control facts or read immutable output. They cannot advance `T`, declare
ranking current, or reinterpret recovery completion.

The component introduces no new product rule, watermark, evaluator, T/Q-to-
ranking dependency, time-window meaning, or competing state owner.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must answer:

- what measured evaluation delay, freshness, queue, recovery, checkpoint, and
  shutdown values meet the fixed trust semantics on the local host;
- which existing package seams form the smallest runnable composition without
  a second orchestrator;
- which counters can be observed without unbounded labels or hot-path
  distortion;
- what compact mixed-load fixture distinguishes throughput from queue growth
  and false-ready publication; and
- whether runtime/lifecycle and measurement/load behavior need one or two
  slices.

After C7 final acceptance, reconnaissance may inspect only V2 runtime
configuration, readiness/health derivation, bounded metrics, shutdown,
controlled-load tests, and directly used fixtures. It must reject V2 owner,
planner, health-store, polling, recovery-stage, ranking, and publication
authority. No production logs, credentials, live calls, deployment, or broad
performance suite are authorized.

Likely primary proofs:

1. a deterministic composition/lifecycle trace covering startup, ready,
   stale, recovery, terminal failure, checkpoint fallback, and bounded
   shutdown without false readiness; and
2. one controlled 6,000-symbol mixed-load acceptance that records throughput,
   processing delay, queue growth, memory, reject/drop/shedding counts, and
   confirms aggregate accounting/correctness.

**Provisional slices:** At most two: `C8-S1` runnable composition and lifecycle;
`C8-S2` measurements, thresholds, and mixed-load acceptance. Combine them if
the completed contract has one coherent proof boundary.

The boundary and exact reconnaissance scope are owner-approved. Detailed
thresholds, fixtures, proof allocation, and slice boundaries are V1 agent-
revisable. A correctable failure reopens the affected item and does not request
owner approval.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.

## 8. Reconnaissance and reuse ledger

Only the seven pre-recorded V2 runtime/health/metrics/config/load files below
were inspected. Version 2 is evidence, not authority.

| Source and SHA-256 | Decision | Preserved evidence and coupling rejected | Proof |
| --- | --- | --- | --- |
| `cmd/scanner/main.go` `d4765c...ffcd` | `adapt` | Signal-rooted cancellation, bounded ten-second join, and explicit subsystem terminals are useful. Reject the V2 owner, HTTP, health store, polling, ranking, recovery-stage, and T/Q orchestration. | `P-C8-RUNTIME` |
| `internal/scanner/metrics.go` `b198ad...9976` | `adapt` | Atomic scalar counters, maxima, `runtime.MemStats`, goroutine count, and bounded reason arrays are useful. Reject maps built from unbounded labels and V2 scanner ownership. | `P-C8-LOAD` |
| `internal/scanner/health.go` / `health_test.go` `2553e5...e22` / `a77076...a44` | `behavior evidence` | Queue-age/processing-delay comparison and stale-generation rejection identify necessary observations. Reject the competing mutable health/readiness/recovery owner. | `P-C8-RUNTIME` |
| `internal/massive/config.go` / `config_queue_test.go` `5472ad...eaa` / `e46e18...72c2` | `adapt` | Exact bounds, literal loopback validation, private paths, count/byte queue limits, gate closure, and ordered terminal markers are useful. Reject V2 product caps and HTTP/asset configuration. | both |
| `internal/massive/capacity_test.go` `1e5de9...409e` | `behavior evidence` | Deterministic seed/count manifests, correction injection, p50/p99/max segments, memory/goroutine plateau, and independent accounting constants are useful. Reject its 10,000-symbol/million-event scale and obsolete engine/formula expectations. | `P-C8-LOAD` |

No production logs, credentials, live requests, deployment artifacts, broad
performance suite, or unrecorded predecessor source were inspected.

## 9. Detailed inputs, outputs, state, and configuration

`internal/operations` owns only process composition, immutable operational
sampling, bounded counters, cancellation, and joins. It accepts the immutable
Component 1 binding and concrete Components 2/5/6/7 instances/facts. It owns no
symbol, ranking, watermark, recovery, checkpoint, or provider truth.

The engine exposes one defensive C8 operational view copied from its existing
immutable publication. The view contains publication/binding identity, run
mode, lifecycle/reason, watermark/generated time, ranking mode/reason/current
claim, queue capacity/occupancy, bounded admission/disposition accounting,
aggregate accounting, connection/ack state, hydration purpose/accounting/fence,
and suppression. C10 may map it but cannot mutate it.

The runtime admits one engine timer per sampler interval. Before an ordinary
live timer, the engine issues an unforgeable current-binding/current-epoch
coverage command; C5 appends its fact after the concrete raw-frame tail, and
the engine consumes it through the FIFO before the timer. Only that accepted
fact extends continuous live/no-print support. Replayed, foreign, regressing,
pre-ack, post-admission, and stale-epoch facts are fenced. This realizes
`LIFE-LIVE-02`/`03` without adding a watermark or adapter-owned currentness.

Current local settings are: engine capacity 8,192 with 128 required-input
reserve; C5 queue 512 frames (the accepted C5 hard limit), 64 MiB total, 8 MiB per frame; four-second
evaluation delay; one-second sampler cadence; two-second readiness tolerance
past the exact causal target; three recovery attempts; 60 seconds to establish
each attempt (C7 measured fresh bootstrap at about 36.6 seconds);
and ten-second controlled shutdown. Constructors reject nonpositive, inverted,
or above-component-hard-limit settings. These are V1 delivery settings and may
be revised from measured evidence without changing readiness meaning.

## 10. Required behavior

| Requirement | Exact behavior |
| --- | --- |
| `C8-RUNTIME-01` | One concrete runtime binds/configures the sole engine, C5 adapter, C6 hydration worker, and C7 store/writer; it emits no current claim before the engine does and never treats T/Q state as readiness. |
| `C8-READY-01` | `process_live` means the runtime has started and not joined. `backend_ready` requires live mode, lifecycle `live` or permitted exact/current `hydrating`, a current engine ranking claim, current acknowledgement, reconciled startup/recovery fence, no suppression, and watermark within two seconds of `min(floor(now)-4s,E)`. Ranking and field statuses remain separate. |
| `C8-RECOVERY-01` | Disconnect/gap facts enter only the engine-owned stale/recovery path. Each attempt and the attempt count are bounded; exhaustion produces honest suppressed/unavailable output and cannot leave an inactive recovery loop. |
| `C8-SHUTDOWN-01` | Cancellation closes ingress, drains/fences accepted work, stops the checkpoint writer/engine, and joins every owned goroutine within ten seconds; timeout is terminal and never reported clean. |
| `C8-MEASURE-01` | Fixed-cardinality observations report ingress/admission/disposition throughput, processing delay, watermark lag, queue current/high-water count/bytes, memory/goroutines, aggregate/TQ reject/drop/shed counts, hydration/recovery accounting, and checkpoint status. Counters reconcile and labels are closed enums. |
| `C8-CAPACITY-01` | A deterministic 6,000-symbol mixed aggregate/control/TQ-deferred load stays within configured queue/memory/work bounds, preserves exact aggregate accounting/ranking, and does not let T/Q pressure change readiness or ranking. |

## 11. Trust boundaries and false-success containment

The smallest dangerous false success is a running process with a stale or
fence-lagged watermark reported backend-ready. Readiness therefore derives
only from one immutable engine publication plus the sampled clock; collectors
cannot supply a watermark or lifecycle. A disconnected adapter, unreconciled
fence, suppressed lifecycle, mismatched binding, stale target, or unresolved
zero-mark bootstrap is not ready. T/Q zero coverage is recorded independently.

Metrics accept only scalar closed-family facts and monotonic counters. Invalid
counter reconciliation marks the operational sample invalid; it cannot be
silently omitted. Cancellation and timeout can only remove readiness. Queue and
goroutine bounds are construction limits; provider outcomes and completion
claims remain runtime-validated.

## 12. Primary proof allocation

| Requirement(s) | Primary proof, counterexample, observable result, limitation |
| --- | --- |
| `C8-RUNTIME-01`, `C8-READY-01`, `C8-RECOVERY-01`, `C8-SHUTDOWN-01` | `P-C8-RUNTIME`: deterministic startup/current/disconnect/recovery/exhaustion/stop trace through real engine admissions and fake C5/C6 boundaries. It rejects process-live-as-ready, stale generation success, T/Q-as-gate, and unjoined shutdown. It observes exact lifecycle/readiness reasons and all goroutines joined. It does not prove provider availability or scale. |
| `C8-MEASURE-01`, `C8-CAPACITY-01` | `P-C8-LOAD`: explicit non-short 6,000-symbol mixed-load acceptance with fixed seed/count manifest, correction and rejection classes, segmented throughput/delay/lag/queue/memory/goroutine observations, exact accounting/ranking oracle, and zero-to-nonzero T/Q diagnostic variation. It detects loss, unbounded growth, and readiness coupling. It is host/fixture evidence, not live capacity or an SLA. |

## 13. Slice plan and assignments

`C8-S1` owns `internal/operations`, the narrow engine operational projection,
and `cmd/scanner` composition/configuration. It completes `C8-RUNTIME-01`,
`C8-READY-01`, `C8-RECOVERY-01`, and `C8-SHUTDOWN-01` with `P-C8-RUNTIME`.
It may change only those paths and narrow accepted-component seams. Provider
credentials/live calls, C9 T/Q policy, C10 HTTP, and public deployment are
deferred.

`C8-S2` adds fixed-cardinality measurements and the controlled-load proof in
the same operations boundary. It completes `C8-MEASURE-01` and
`C8-CAPACITY-01` with `P-C8-LOAD`. It may revise the delivery settings from
recorded measurements but may not change Phase 1 market/readiness semantics.

## 14-19. Verification, review, discretion, deferrals, and correction

Ordinary verification is `go test -short -timeout 2m ./...`. Each slice runs
its named proof and affected race packages. `P-C8-LOAD` is non-short with a
five-minute per-trial and ten-minute command timeout; it validates its manifest
before timing. Final acceptance requires both proofs, ordinary/race checks, a
clean conformance walkthrough, and one final read-only review.

Implementation may choose private helpers, fixed arrays, atomic mechanics,
error wrapping, and exact file layout. It may not add another readiness owner,
polling health truth, watermark, evaluator, T/Q readiness predicate, database,
service split, generic event bus, or runtime plugin system. Credentialed/live
validation, HTTP/API/UI, public deployment, HA, exhaustive soak, and capacity
beyond the recorded host/fixture remain deferred. Every failure follows the V1
correction loop and updates this sole delivery ledger without an owner stop.

## 20. Slice evidence, corrections, and limitations

`C8-S1` now provides the runnable `cmd/scanner` composition, bounded binding
installation, lazy C7 latest-to-previous installation, up to three consecutive
failed connection/hydration establishment attempts with an independent
60-second deadline for each attempt, a live
connection lifetime rooted separately from those establishment deadlines,
explicit retry or exhaustion policy facts, one-second engine timers,
engine-issued C5 live-coverage fences, and a ten-second joined shutdown. The
queue setting was corrected from the provisional 4,096 frames to the accepted
C5 hard limit of 512. The establishment bound was corrected from 30 to 60
seconds because C7's accepted fresh-bootstrap median was about 36.6 seconds. A
single injected C5 clock source, defaulting to UTC wall time, keeps receipts,
deadlines, terminals, and fence capture causally coherent without changing
production time semantics.

`P-C8-RUNTIME` rejects process-live-as-ready, T/Q-as-readiness, a stale
watermark, disconnect-as-current, an establishment deadline canceling a healthy
live connection, caller-forged/replayed continuous-coverage facts, and unjoined
shutdown. The real local WebSocket/REST trace remained `live` and ready beyond
its two-second test establishment deadline because ordered live fences and
timers advanced the sole engine watermark. Focused race verification passed.

Final review reopened two claims. First, retry purpose and exhaustion were
selected from the supervisor's total attempt ordinal, so a failed initial
handshake incorrectly requested gap recovery and three connection epochs over
the whole process could consume the lifetime budget. The correction exposes
only an engine-owned installed-checkpoint flag and consecutive attempt count:
`fresh_bootstrap`, `checkpoint_catchup`, or `gap_recovery` is selected from the
current engine lifecycle/baseline, and the count resets only when hydration and
its ingress fence complete. The engine accepts the new bounded exhaustion fact
only after the exact stated count of ordered connection attempts, a current
connection-loss terminal, and no active epoch or hydration generation. The
proof now covers a failed first handshake followed by fresh bootstrap, two
failed establishments reaching `recovery_exhausted`, and two separate
successful live-disconnect/gap-hydration/live cycles under a
two-consecutive-failure budget. After both resets, the same concrete trace
forces two failed recovery connections and reaches engine-owned
`recovery_exhausted`, proving the post-live exhaustion boundary as well as the
startup boundary.
Second, `Shutdown` did not own the active live composition. Runtime now
registers exactly one live lifetime, cancels it, waits for its terminal, and
only then closes the timer, engine, and checkpoint writer. The proof calls
`Shutdown` while live without first joining and separately verifies that a
join timeout returns an error and leaves `joined=false` for a safe retry.
Focused re-review also found that a newly installed checkpoint could retain a
current T0 evaluation while the acknowledged engine was `hydrating`, before a
catch-up plan existed; the prior generation-conditional fence check could
therefore report ready inside that small window. Live readiness now always
requires a reconciled hydration/ingress fence. A checkpoint-current,
generation-zero, pre-plan trace remains ranking-current for diagnostics but is
explicitly not backend-ready with `fence_pending`.

`C8-S2` exposes a fixed-cardinality sample containing current engine admission,
transition, aggregate, connection, and hydration families; C5 queue and adapter
accounting; C7 writer accounting; delivery count/mean/max processing delay;
watermark lag; heap; and goroutines. Absolute counters plus `sampled_at` support
throughput deltas without mutable label maps. The operational projection joins
immutable readiness/publication state with a same-lock snapshot of current
nonpublishing disposition counters, preventing exact duplicates/rejections
from disappearing until a later market publication.

The accepted current-host `P-C8-LOAD` manifest is exactly 6,000 bound symbols,
100 initially active symbols, 20 same-window live corrections, 20 exact
duplicates, 20 engine rejections, 60 deferred T/Q facts, two post-handshake raw
frames, and one exact evaluator timer. The latest run measured 29.14 seconds
for the controlled stream and 53.58 seconds end-to-end, about 8 items/s,
131.24 ms mean and 317.97 ms maximum instrumented delivery delay, two frames
and 9,172 bytes at queue high-water, about 90.4 MiB heap growth, three
goroutines of growth, and zero watermark lag. Aggregate accounting was exactly
`consumed=160=100 inserted+20 revised+20 duplicate+20 rejected`; all 6,000
symbols reconciled as 100 trusted marks plus 5,900 no-print-through-T. The 60
T/Q facts changed only the deferred diagnostic and `TQAvailable` remained
false; readiness/ranking stayed current. This is bounded local fixture
evidence, not provider availability, a market-hours observation, an SLA, or
capacity beyond this host and shape.

Verification after both slices: `go test -short -timeout 2m ./...` passed; the
affected `internal/engine`, `internal/massive`, and `internal/operations` race
packages passed under a five-minute command bound; and `P-C8-LOAD` passed under
its five-minute trial/ten-minute command bounds. The initial 7,700-item active
shape was interrupted after 201 seconds when per-disposition publication cost
showed it was disproportionate; a 600-active-symbol revision then completed in
229.62 seconds but exposed an incorrect nonempty-row oracle. The final compact
shape preserves every disposition, full-population accounting, real queue
pressure, and the exact empty qualified-table oracle while completing within
the intended local cost envelope.

The required final read-only review initially found the recovery, shutdown,
and checkpoint false-ready defects recorded above. After the narrow
corrections and proofs, the same reviewer found no remaining actionable P1/P2
issue. Its uncached ordinary suite and affected race suite both passed, as did
`git diff --check`. The unchanged non-short `P-C8-LOAD` evidence was reused;
credentialed provider availability, market-hours validation, and an SLA remain
explicitly unproven and deferred.
