# Readiness and operations

**Status:** Owner-approved V1 boundary and reconnaissance plan; detailed
contract pending after Component 7 final acceptance

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
| Boundary/reconnaissance plan | `boundary_approved` | Direct owner V1 program revision, 2026-08-07 | Wait for C7 final acceptance, then perform the recorded narrow reconnaissance |
| Completed contract | `pending` | Risk-triggered focused contract review only if the completed plan changes a consequential lifecycle/concurrency/interface boundary | Define exact thresholds, proofs, and at most two slices |
| Implementation | `pending` | One final read-only component review after all proofs | Begin only after completed contract is recorded |

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
