# Readiness and operations

**Status:** Component 8 reaccepted 2026-08-13 after the fixed-cardinality
engine-delivery latency attribution correction; runtime/readiness, capacity,
checkpoint, and prior measurement evidence remain accepted

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
| `C8-S1` runtime/lifecycle | `accepted_after_correction` | 2026-08-10 live-start evidence invalidated the use of one 60-second wall-clock deadline for both connection establishment and full-population hydration. Corrected `P-C8-RUNTIME` proves the connection deadline is handshake-only, finite C6 hydration can outlive it, the subscribed live tail is consumed concurrently, and every causal predecessor is drained through the ingress fence before readiness; affected race clean. | Complete |
| `C8-S1` live response-bound/failure containment | `accepted_after_correction` | A 2026-08-10 two-worker launch reached 4,558/5,517 terminal symbols without live-queue loss, then exposed an undersized 512 MiB cumulative REST transfer bound and a proof-byte accounting defect that converted bounded exhaustion into global suppression. The accepted correction raised production to 2 GiB; the later owner-selected pre-retry setting is 4 GiB. The proof byte is excluded from accepted terminal accounting, and terminal engine suppression exits the live supervisor instead of reconnect-spinning. Focused provider/engine/supervisor proofs, ordinary verification, and race verification pass. | Complete |
| Owner-selected live-retry headroom | `accepted_after_correction_review` | The 2026-08-12 owner directed loosening only self-imposed delivery limits before the next run. Production selects 32,768 C5 frame slots under the unchanged 128-MiB/8-MiB byte guards, raises the nonresident cumulative REST transfer allowance from 2 to 4 GiB, and allows five recovery attempts instead of three. Review exposed and correction removed quadratic slice dequeue and a scheduler-dependent saturation fixture. Full-ceiling wrap/saturation/drain race and an 8,192-slot Runtime capacity control/failure race now pass; the latter proves cross-component behavior without claiming a full 32,768-slot provider composition. The isolated production-configuration 216-frame/s proof dispositioned 12,960/12,960 frames and 25,920 aggregates with queue high-water 2, final depth zero, and no rejection. Two hydration workers, resident-record bounds, provider page/request bounds, readiness/currentness, accounting, and terminal semantics remain unchanged. Final `gpt-5.6-sol` medium re-review returned CLEAN/PASS. | Complete locally; use the new settings for the separately authorized owner-run observation. |
| `C8-S1` hydration-pin/fence finalization | `accepted_after_deterministic_correction` | The request-symbol index, progressive canonical folding, bounded feature maintenance, and coalesced aggregate projection preserve the exact fence/readiness semantics while removing the measured dominant costs. Final-source full 1x passed all 7,581,690 rows and 28,377 frames with zero rejection, queue high-water 293, 490.875 ms fence, 650.808 ms drain, and ready/accounting true. | Complete locally; separately authorized provider confirmation remains pending |
| 2026-08-11 engine-to-API correction | `accepted_after_correction_review` | [`live-engine-api-first-ready-correction.md`](../live-engine-api-first-ready-correction.md) proves small and 5,691-symbol first-ready/post-fence publications map through the production Runtime/API boundary and fixes a reasonless timer erasing the fixed `ingress_integrity` suppression cause. The initial post-fence-publication P2 and diagnostic-wiring P2 are closed; focused re-reviews found no remaining P1/P2. No provider or credential was accessed. | Complete locally; original failed live capture and provider confirmation remain unavailable/deferred |
| Cached hydration fence correction | `accepted_deterministically` | [`live-fence-finalization-cached-hydration-correction.md`](../live-fence-finalization-cached-hydration-correction.md) records the sealed-artifact full 1x pass, a second current-source full 1x pass, the strengthened 3,072,000-row bounded 2x burst pass, and the preserved honest sustained-2x failure. No provider or credential was accessed. | Complete; do not rerun sustained full 2x |
| Checkpoint hot-path measurement composition | `finally_accepted` | [`live-checkpoint-hot-path-correction.md`](../live-checkpoint-hot-path-correction.md) adds fixed-cardinality projection/submit/terminal/timing/byte facts, returns writer terminals to engine accounting, and proves a real completed/discoverable artifact. Three real-consumer dense 6,000-symbol boundaries limited observation delay to 55.523ms and aggregate delivery to 66.874ms; focused re-review is clean. | Preserve for V1 integration; provider capacity remains unproved |
| Engine-delivery latency attribution | `accepted_after_empty_window_correction` | The prior proof established cumulative seven-family accounting and atomic winning-delivery attribution but did not cross the production `syncTQPressure` reset into an immediate C10 mapping. It incorrectly treated cumulative deliveries as evidence that the reset current window was nonempty. The resulting `0 / unknown` window was coherent but failed mapping whenever cumulative `unknown` remained zero. The corrected production-path regression crosses record/reset/capture/map/next-record, while focused C8/C9/C10, repository short, affected race, vet, and diff gates pass. | Complete; current-window occupancy is lock-copied with the maximum pair and remains independent of lifetime family counts. |
| `C8-S2` measurements/load | `accepted_after_correction` | `P-C8-LOAD`: 6,000-symbol bound population, 100 active aggregate symbols, 20 corrections, 20 duplicates, 20 rejects, and 60 normalized T/Q facts correctly fenced without acknowledged membership; 53.67 s end-to-end and exact accounting | Complete |
| Final component review | `accepted_after_2026-08-10_correction` | Focused re-review confirmed the hydration-deadline/live-tail correction plus synchronized concurrent-terminal diagnostics and an explicit active-attempt shutdown join. No P1/P2 finding remains; uncached ordinary, focused race, vet, UI-model, and diff checks pass. Unchanged non-short load evidence is reused. | Complete |
| Private daily operational finalization | `accepted_after_final_review` | `./scripts/run-private-scanner` supplies the exact two-worker/persistent-path workflow, isolates credentials, gates dashboard startup on `/livez`, reports engine `/readyz`, and contains both processes. Public-wrapper/fake-process tests, current-source capacity evidence, ordinary/race/vet/UI checks, and correction/re-review are clean. | Complete locally; exact-date provider observation remains separately authorized |

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

The engine's current-market claim includes `qualified_current`,
`degraded_bootstrap`, and the owner-approved `degraded_current` partial mode.
Operations does not infer those modes from counters: it accepts the sealed
engine claim, then independently requires live process/lifecycle, acknowledged
transport, reconciled fence, current watermark, and valid operational
accounting. T/Q remains outside readiness for every aggregate ranking mode.

The runtime admits one engine timer per sampler interval. Before an ordinary
live timer, the engine issues an unforgeable current-binding/current-epoch
coverage command; C5 appends its fact after the concrete raw-frame tail, and
the engine consumes it through the FIFO before the timer. Only that accepted
fact extends continuous live/no-print support. Replayed, foreign, regressing,
pre-ack, post-admission, and stale-epoch facts are fenced. This realizes
`LIFE-LIVE-02`/`03` without adding a watermark or adapter-owned currentness.

Current local settings are: engine capacity 8,192 with 128 required-input
reserve; C5 queue 32,768 frames, 128 MiB total, 8 MiB
per frame; four-second evaluation delay; one-second sampler cadence; two-second
readiness tolerance past the exact causal target; five recovery attempts; 60
seconds to establish and acknowledge each connection attempt; C6's finite
full-population hydration plan with two workers in the supported launcher (one,
two, four, or eight are valid), at most two pages and three
15-second attempts per page, explicit byte/record limits, and parent/shutdown
cancellation; and ten-second controlled shutdown. Hydration has no unrelated
whole-plan wall-clock success deadline. Constructors reject nonpositive,
inverted, or above-component-hard-limit settings. These are V1 delivery
settings and may be revised from measured evidence without changing readiness
meaning.

## 10. Required behavior

| Requirement | Exact behavior |
| --- | --- |
| `C8-RUNTIME-01` | One concrete runtime binds/configures the sole engine, C5 adapter, C6 hydration worker, and C7 store/writer; it emits no current claim before the engine does and never treats T/Q state as readiness. |
| `C8-READY-01` | `process_live` means the runtime has started and not joined. `backend_ready` requires live mode, lifecycle `live` or permitted exact/current `hydrating`, a current engine ranking claim, current acknowledgement, reconciled startup/recovery fence, no suppression, and watermark within two seconds of `min(floor(now)-4s,E)`. Ranking and field statuses remain separate. |
| `C8-RECOVERY-01` | Disconnect/gap facts enter only the engine-owned stale/recovery path. Each attempt and the attempt count are bounded; exhaustion produces honest suppressed/unavailable output and cannot leave an inactive recovery loop. |
| `C8-SHUTDOWN-01` | Cancellation closes ingress, drains/fences accepted work, stops the checkpoint writer/engine, and joins every owned goroutine within ten seconds; timeout is terminal and never reported clean. |
| `C8-MEASURE-01` | Fixed-cardinality observations report ingress/admission/disposition throughput, processing delay, watermark lag, queue current/high-water count/bytes, memory/goroutines, aggregate/TQ reject/drop/shed counts, hydration/recovery accounting, and checkpoint status. Engine-delivery counts partition exactly into aggregate, T/Q, control, hydration/fence, checkpoint-related, timer, or unknown. Current one-second-window occupancy is represented independently from those cumulative counts. The one-second maximum duration and family are one atomic winning-delivery record; an empty window is `0 / unknown`, equal durations use the fixed listed-family order, and absent or cross-family completion evidence is unknown. Counters reconcile and labels are closed enums. |
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
| `C8-MEASURE-01` | `P-C8-DELIVERY-ATTRIBUTION`: compact operations/C10 trace proving the exact seven-family delivery identity, atomic duration/family maximum, fixed tie order, unknown/mixed containment, unchanged duration threshold/ranking/watermark/readiness, the attributed-winner fact consumed only by C9 recovery, and bounded coherent schema output. Counterexamples are a split maximum pair, count mismatch, dynamic family, mixed completion labeled specific, or an exact family changing a pressure threshold. It identifies the measured input family, not the sole CPU/queueing cause. |
| `C8-CAPACITY-01` | `P-C8-LOAD`: explicit non-short 6,000-symbol mixed-load acceptance with fixed seed/count manifest, correction and rejection classes, segmented throughput/delay/lag/queue/memory/goroutine observations, exact accounting/ranking oracle, and zero-to-nonzero T/Q diagnostic variation. It detects loss, unbounded growth, and readiness coupling. It is host/fixture evidence, not live capacity or an SLA. |

## 13. Slice plan and assignments

`C8-S1` owns `internal/operations`, the narrow engine operational projection,
and `cmd/scanner` composition/configuration. It completes `C8-RUNTIME-01`,
`C8-READY-01`, `C8-RECOVERY-01`, and `C8-SHUTDOWN-01` with `P-C8-RUNTIME`.
It may change only those paths and narrow accepted-component seams. Provider
credentials/live calls, C9 T/Q policy, C10 HTTP, and public deployment are
deferred.

`C8-S2` adds fixed-cardinality measurements and the controlled-load proof in
the same operations boundary. It completes `C8-MEASURE-01` and
`C8-CAPACITY-01` with `P-C8-DELIVERY-ATTRIBUTION` and `P-C8-LOAD`
respectively. It may revise the delivery settings from recorded measurements
but may not change Phase 1 market/readiness semantics.

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
installation, lazy C7 latest-to-previous installation, up to five consecutive
failed recovery attempts, a 60-second connection establishment/acknowledgement
deadline, finite C6 hydration whose lifetime is rooted in the live operation
rather than that connection deadline, concurrent subscribed-live-tail delivery
during hydration, explicit retry or exhaustion policy facts, one-second engine
timers, engine-issued C5 live-coverage fences, and a ten-second joined shutdown. The
queue setting was revised by the owner-authorized D4 correction from 1,024 to
8,192 frames, then by the owner's pre-retry headroom direction to 32,768 frames,
under the unchanged 128 MiB byte ceiling. The original establishment bound was corrected from 30
to 60 seconds using C7's accepted fresh-bootstrap median of about 36.6 seconds,
but 2026-08-10 evidence showed that benchmark did not justify a production
full-universe hydration deadline. The deadline now ends after the WebSocket
handshake; C6's per-request retries/deadlines and finite plan bound hydration.
A single injected C5 clock source, defaulting to UTC wall time, keeps receipts,
deadlines, terminals, and fence capture causally coherent without changing
production time semantics.

The controlled 2026-08-10 launch also invalidated the original 512 MiB
cumulative REST response budget. It was a whole-plan transfer bound rather
than a resident-memory limit and stopped a healthy two-worker hydration at
4,558 of 5,517 terminal symbols. Production now uses a 2 GiB cumulative bound,
more than three times the approximately 625 MiB complete transfer projected
from that observation. The owner's 2026-08-12 pre-retry direction raises this
nonresident cumulative allowance to 4 GiB to remove another avoidable
whole-plan exhaustion point; it does not raise any page or resident-memory
bound. C6's 16 MiB page, two-page, three-attempt,
15-second-per-attempt, interval-row, and worker-resident bounds remain intact.
The downloader no longer includes the single over-bound proof byte in terminal
accounting, and the engine accepts an ordinary failed terminal that exactly
consumes the remaining authorized bytes without global suppression. The live
supervisor also exits when the engine is already `suppressed` or `ended`, so a
terminal integrity event cannot become an unbounded reconnect loop.

The following paragraph records the historical state before the accepted
cached correction; it is superseded by the final-source result below. The next
controlled launch completed the full hydration plan (5,468 value and
49 successful empty terminals) but invalidated the cost of the active-generation
pin lookup. Live aggregate installation compacted each symbol tail and checked
each retained identity against the full 5,517-request plan. Near the end of the
extended session, this records-times-population loop saturated one core and
filled the 512-frame queue before the ingress fence. The generation now owns a
symbol-to-request index. While it is active, compaction returns immediately for
a pinned symbol; the fence deactivates the generation and performs the required
one-time compaction before readiness evaluation. This changes implementation
cost only: retained identity, fence ordering, ranking, and readiness semantics
are unchanged. Deterministic and race proofs pass. Two repeated provider
launches proved ordinary hydration now remains caught up through all 5,517
terminals, but reopened the fence-finalization claim: delivery stopped at 8,152
in the first and 9,421 in the chronological-fold run. In the latter, queue
occupancy rose from 157 to 483/512 in about 28 seconds before the scanner was
stopped. At that point, the remaining full-population compaction, coverage,
maintenance, and projection work still required profiling or bounded ordered
finalization; readiness could not be claimed from REST completion alone. The
accepted cached correction recorded later in this section closes that
historical condition without weakening the readiness fence.

`P-C8-RUNTIME` rejects process-live-as-ready, T/Q-as-readiness, a stale
watermark, disconnect-as-current, a connection deadline canceling finite
hydration or a healthy live connection, queued live input being skipped by the
hydration fence, caller-forged/replayed continuous-coverage facts, and unjoined
shutdown. The corrected local WebSocket/REST trace holds REST hydration beyond
its 250-millisecond connection deadline, admits a subscribed live aggregate
while hydration is still active, then becomes ready only after hydration and
the exact ingress fence reconcile. Focused race verification passed.

The focused review found two failure-path defects in that correction. First,
up to eight concurrent C6 terminal callbacks could race while recording a
rejected or fenced terminal diagnostic after connection loss. The sink now
synchronizes terminal-error and fence state, aggregates rejected terminal
diagnostics, and exposes one post-worker-join result. An eight-worker test holds
all REST requests active, drops the live epoch, and proves terminal exhaustion
plus reconciled adapter cleanup under the race detector. Second, parent
cancellation could pre-cancel `closeAndDrain`, allowing `liveDone` to close
before the C5 attempt cleanup goroutine joined. Attempt cleanup now uses an
independent bounded context and waits for the attempt; `Shutdown` independently
requires the active attempt to join before timer, engine, writer, or
`joined=true`. A second eight-worker test shuts down with every REST request
blocked and proves RunLive, requests, adapter queue, and attempt cleanup are
joined. Focused re-review found both issues resolved with no new P1/P2 finding.

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
duplicates, 20 engine rejections, 60 normalized T/Q facts, two post-handshake raw
frames, and one exact evaluator timer. The latest run measured 29.27 seconds
for the controlled stream and 53.67 seconds end-to-end, about 8 items/s,
131.82 ms mean and 255.56 ms maximum instrumented delivery delay, two frames
and 9,172 bytes at queue high-water, about 104.3 MiB heap growth, three
goroutines of growth, and zero watermark lag. Aggregate accounting was exactly
`consumed=160=100 inserted+20 revised+20 duplicate+20 rejected`; all 6,000
symbols reconciled as 100 trusted marks plus 5,900 no-print-through-T. The 60
T/Q facts were all normalized, consumed by Component 9, and fenced because the
scenario intentionally has no acknowledged selected-symbol T/Q membership;
`TQAvailable` remained false and readiness/ranking stayed current. This corrects
the pre-C9 deferred-consumer oracle without changing the load shape. This is
bounded local fixture evidence, not provider availability, a market-hours
observation, an SLA, or capacity beyond this host and shape.

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

The cached fence correction subsequently closed the reopened deterministic
boundary without changing queue capacity, ranking, qualification, hydration,
or readiness meaning. The final-source full 1x trial validated 7,581,690 rows,
5,439 value and 63 successful-empty terminals, 28,377 fully reconciled frames,
zero rejection, queue high-water 293, a 490.875 ms ingress fence, 650.808 ms
tail drain, `live`/`qualified_current`, 20 rows, and ready/accounting true. The
focused full-retention 2x burst completed all 3,072,000 measured rows and
23,448 frames with zero rejection, queue high-water 38, 2.194 ms drain, and
reconciled accounting. The separate sustained full 2x artifact remains a
failure after 7,078,180 hydration rows and is explicitly unsupported. These
results are distinct trials and no metrics were combined.

The 2026-08-13 `P-C8-DELIVERY-ATTRIBUTION` correction replaces the duration-
only one-second record with one lock-protected maximum pair and seven scalar
cumulative counters: aggregate, T/Q, control, hydration/fence, checkpoint,
timer, and unknown. Massive delivery completions are classified only from one
unambiguous typed result; explicit runtime timer and checkpoint-terminal waits
use their closed families. Absent evidence, an unsupported family, or evidence
claiming more than one family is counted as unknown. Equal durations resolve in
the listed fixed order, so concurrent arrival order cannot choose the label.
The exact identity is `deliveries = aggregate + tq + control +
hydration_fence + checkpoint + timer + unknown`, and a nonempty maximum family
must have a nonzero matching count. Duration, family, counts, delivery total,
mean, lifetime maximum, and reset version are copied under the same mutex.

C9 receives `MaxProcessingDelayOneSecond` plus one fail-closed boolean that is
true only when the atomic winning record reconciles and its family is not
`unknown`. The exact family and counts do not enter thresholds; C9 uses the
boolean only to prevent an unattributed maximum from advancing recovery dwell.
The proof holds duration constant while changing a known winner to `unknown`
and obtains identical duration with only that boolean changed, while recording
an unchanged immutable engine snapshot and readiness view. The additive C10
object repeats the paired maximum, carries no
symbols or dynamic labels, and rejects broken count, duration/family, enum, and
winner-presence identities. This is delivery-family attribution, not proof that
the named input family was the sole CPU or queueing cause of its elapsed time.
No independent review was triggered: the change adds no ownership,
concurrency-linearization, persistence, or external-evidence authority; one
mutex and the primary identity proof make the diagnostic boundary direct.

The subsequent D6 correction invalidated only the assumption that cumulative
deliveries proved the current one-second window was nonempty. Runtime now keeps
one boolean occupancy fact under the existing delivery-window mutex. Every
record sets it while updating the cumulative family and atomic maximum pair;
an accepted `syncTQPressure` reset clears it together with the maximum only if
the sampled window version is still current. Cumulative family counts are not
reset and no `unknown` delivery is fabricated. C9 therefore continues to treat
an empty `0 / unknown` winner as unattributed recovery evidence while all
degradation, aggregate-only, and recovery thresholds remain unchanged.

`TestDeliveryLatencyEmptyWindowMapsAfterPressureReset` is the distinguishing
production-path regression. It records a known aggregate delivery, executes
the real pressure command/result reset, immediately seals and maps the empty
window, records a checkpoint delivery, and verifies its duration/family pair
plus the unchanged cumulative seven-family identity. The focused C8/C9/C10
proofs, `go test -short -timeout 2m ./... -count=1`, the affected five-package
race command under five minutes, `go vet ./...`, and `git diff --check` pass.
The proof is deterministic local composition evidence; it does not reproduce
provider timing or establish live capacity. No independent review was
triggered because this correction adds no new linearization point, owner,
persistence/atomicity boundary, external evidence admission, or cross-component
authority: the occupancy fact uses the already-proved maximum-pair mutex and
the C8-to-C10 representation is exhaustively distinguished by the primary test.
