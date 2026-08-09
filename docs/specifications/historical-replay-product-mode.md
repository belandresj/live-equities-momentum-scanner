# Historical replay product mode

**Status:** Skeleton draft; owner boundary approval pending

**Boundary approval authority:** Owner

**Boundary approval:** Pending

**Completed-contract approval authority:** Owner

**Contract/reuse/test/slice-plan approval:** Pending REST feasibility evidence,
current-dependency reconnaissance, focused review, and owner decision

**Standing program decisions:** The existing
[Version 1 Release Program](../v1-release-program.md) controls C7-C11 only. It
does not authorize C12 approval, implementation, credential access, or changes
to accepted C4 acquisition semantics.

**Advancement mode:** Owner-controlled boundary, benchmark-execution, completed-
contract, and implementation gates

**Controlling Phase 1 requirements:** `PG-REPLAY-01`, `PG-REPLAY-02`,
`PG-UI-01`, `PG-UI-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-04`, `DTE-CLOCK-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-EVENT-04`, `DTE-TIMER-01`, `DTE-COMMIT-01`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-REPLAY-01`, `DTE-REPLAY-02`,
`DTE-REPLAY-03`, `LIFE-INIT-01`, `LIFE-LIVE-05`, `LIFE-REPLAY-01`,
`LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`,
`LIFE-PUBLISH-02`, `LIFE-END-02`, and `LIFE-END-03`

**Proposed dependencies:** Finally accepted C4 aggregate replay, C7 checkpoint
restart, C8 operations runtime, and C10 snapshot API contracts; the current C11
dashboard API/view-state boundary. C11 final Chrome acceptance remains separate.

## Contract document map

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | C12 outcome, ownership, non-scope, settled boundary, evidence questions, likely proofs, and approval state | Sections 1-7; detailed Sections 8-19 pending | Every C12 task | C4, C7, C8, C10, and current C11 contracts |
| [REST feasibility benchmark](../replay-rest-feasibility-benchmark.md) | Non-authoritative provider/host/plan-specific evidence plan and future benchmark assignment | `B-C12-REST`; no product requirement or acceptance proof | REST benchmark planning or execution | C4 downloader and this skeleton |

**Layout:** Compact single-file C12 contract through the skeleton stage. Keep
the benchmark separate because measured acquisition feasibility is evidence,
not product behavior. Convert to a modular contract only if completed-contract
work reveals independently routable runtime/API/UI trust boundaries.

**Routing rule:** C12 owns only accepted-artifact-to-local-product composition.
C4 continues to own historical acquisition, normalization, artifact trust, and
replay source/clock. If measured evidence requires another acquisition source,
reopen C4 through an owner-approved correction before completing C12.

**Contract-wide coverage and acceptance:** Sections 1-7 below are a proposal.
No implementation slice, V2/current-code whitelist, proof, provider benchmark,
or V1 completion claim is approved by this draft.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `skeleton` | Phase 1-derived boundary and exact evidence questions below; consequential cross-component skeleton review pending | 2026-08-09 | Owner reviews boundary and benchmark sequence |
| `B-C12-REST` | `pending_authorization` | Benchmark plan only; no credentials or provider request authorized | 2026-08-09 | After boundary approval, owner separately authorizes exact date, interval, symbols, and credential use |
| Detailed contract | `pending` | Requires benchmark result and bounded current-repository reconnaissance | 2026-08-09 | Complete Sections 8-19 only after evidence resolves the dependency choice |
| Implementation | `not_authorized` | No slices or implementation assignment accepted | 2026-08-09 | Owner approves completed contract before any product change |

## Sections 1-4 — outcome, scope, ownership, and settled boundary

### Outcome and user consequence

C12 proposes an independently observable local historical replay mode. An
operator supplies one accepted complete aggregate artifact and explicit finite
playback pace; the backend replays it through the sole production engine and
evaluator while the existing loopback snapshot API publishes changing,
replay-labeled state for the independent dashboard.

The operator can inspect historical Last, Day %, From 4AM %, HOD drawdown,
range positions, Activity, qualification, and exact server ranking while the
market is closed. Replay is visibly nonlive and current only relative to replay
logical time and complete artifact coverage. Version 1 Tape Rate and Spread
remain explicitly unavailable. Restarting or redeploying only the dashboard
does not stop, pause, relink, or mutate the replay backend.

This proposed mode is a product-review and engineering tool. It makes no claim
about live receipt latency, correction-arrival chronology, current provider
entitlements, predictive edge, or executable trading expectancy.

### Scope

In scope:

- one local replay composition from validated C4 artifact to the existing
  `ScannerStateEngine`, C8-style coherent operational capture, C10 snapshot API,
  and C11 dashboard;
- explicit trading date, artifact, interval, pace, loopback API address, and
  exact allowed dashboard origin;
- replay/nonlive lifecycle and status presentation without changing ranking or
  feature semantics;
- bounded startup, pacing, artifact completion, cancellation, API shutdown,
  dashboard-only restart independence, and joined process termination;
- deterministic equivalence at different playback speeds; and
- a provider/plan/host-specific benchmark of the existing C4 REST downloader
  before choosing whether it remains the practical acquisition dependency.

Not in scope:

- raw-trade or flat-file acquisition, local trade-to-second aggregation, a
  second aggregate normalizer, or changed C4 artifact semantics;
- fake WebSocket transport, live epochs/acknowledgements, historical live
  receipt chronology, or a second scanner/evaluator;
- full-session or two-pass T/Q replay, historical NBBO reconstruction, or
  fabricated Tape Rate/Spread;
- pause, seek, reverse, looping, multi-session playlists, browser-owned pace,
  or saved UI controls in the minimum proposed mode;
- public binding, authentication, TLS, hosting, database/storage service,
  artifact retention-policy changes, or provider credential management; and
- treating a subset benchmark as a complete-universe replay claim.

### Ownership and dependencies

C12 owns one orchestration boundary: lifecycle-safe composition of already
accepted immutable replay inputs and outputs into an independently observable
local product process. It owns no market state. `ScannerStateEngine` remains
the sole canonical state, lifecycle, watermark, evaluation, ranking, and
publication owner. C4 remains the sole accepted downloader/compiler/artifact/
replay-source owner. C8 owns readiness and operational capture meanings; C10
owns the public schema and HTTP trust boundary; C11 owns presentation.

The simplest intended implementation reuses those owners directly. It does
not translate replay through provider framing or create a second snapshot
representation. Any inability of the current C8/C10 boundaries to represent a
running, nonlive replay honestly is a contract-revision input, not permission
for C12 to infer readiness or overlay mutable state.

### Settled boundary

| Boundary item | Settled meaning | Controlling requirements |
| --- | --- | --- |
| Replay input | One validated complete C4 artifact, compatible binding and exact interval enter at the normalized-event boundary; no fake WebSocket | `PG-REPLAY-01`, `DTE-REPLAY-01`, `LIFE-REPLAY-01` |
| State and order | The sole engine/evaluator owns canonical state, logical order, timers, committed `T`, qualification, ranking, and immutable publications | `ARCH-OWN-01`-`04`, `DTE-REPLAY-02`-`03`, `LIFE-REPLAY-02`-`03` |
| Product currentness | Ranking may be exact/current relative to replay `T`, but the process is not production-live ready and must be visibly replay/nonlive | `PG-OBS-03`, `LIFE-PUBLISH-01`-`02` |
| T/Q | Aggregate ranking remains independent; historical T/Q fields are unavailable with replay-specific reason | `PG-REPLAY-02`, `LIFE-REPLAY-02` |
| API/UI | API reads one immutable publication; UI owns no market calculations or readiness and can restart independently | `PG-UI-01`-`02`, `ARCH-OWN-03`, `LIFE-LIVE-05` |
| Pace | Pace changes wall duration only; equivalent logical times produce equivalent engine facts and output | `PG-REPLAY-01`, `DTE-REPLAY-03`, `LIFE-REPLAY-03` |
| Failure | Invalid artifacts or replay ambiguity fail only the offline run; API/UI failure cannot alter canonical history | `ARCH-FLOW-04`, `LIFE-REPLAY-03`, `LIFE-END-02`-`03` |

C12 introduces no new ranking rule, market window, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, event representation, or
browser calculation.

## Sections 5-7 — evidence questions, reconnaissance scope, likely proofs, and boundary checkpoint

### Evidence questions

| Question | Why current authority does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| Can existing per-symbol REST acquisition produce a complete full-session eligible-universe artifact near the owner's ten-minute target? | C4 proves semantics and bounds, not current plan/provider/host capacity | Authorized `B-C12-REST` measurements | Retain C4 REST acquisition or reopen C4 for another source |
| Can C8 operational capture represent replay process-running, replay-current ranking, production-live-not-ready, and ended states without false readiness? | C8 was accepted around runnable live composition | Current type/construction/test inspection | Reuse C8 capture or revise its lower-level representation |
| Can `scanner.snapshot.v1` express replay transitions compatibly? | C10 includes run mode/lifecycle but its production proof is live-focused | Schema/mapper/validator fixtures for replaying and ended | Compatible reuse or explicit C10 contract/schema correction |
| What exact dashboard consequence distinguishes replay-current from live-current, degraded, frozen, and ended? | C11 currently treats overall currentness through live readiness | Current model/render/state-proof inspection and Chrome fixture | Minimal C11-compatible replay presentation |
| Does replay checkpoint continuation belong in the first product mode? | C7 proves replay continuation, but the user workflow has not yet selected CLI exposure | Existing C7 proof/interface inspection plus owner decision | Minimum CLI and restart proof allocation |
| Which finite pace bounds give useful observation without unbounded waits or changing semantics? | Phase 1 fixes invariance, not product defaults or accepted wall bounds | Existing pace implementation/tests plus controlled artifact measurements | Configuration defaults and acceptance timeout |

### Proposed reconnaissance scope

No Version 2 inspection is proposed. Current accepted repository boundaries are
the relevant dependencies. After owner boundary approval and REST benchmark,
the detailed-contract task may inspect only:

- C4 replay command/source/pace and artifact validation interfaces;
- C7 replay checkpoint continuation interfaces and primary proof;
- C8 runtime construction, status, snapshot capture, metrics, and shutdown;
- C10 capture source, mapper/schema, HTTP lifecycle, and replay-status tests;
- C11 validation/rendering of run mode, lifecycle, readiness, T/Q unavailable,
  disconnection, and dashboard-only restart; and
- the smallest existing deterministic artifacts/fixtures required to prove the
  cross-component path.

Exclude provider credentials/requests, V2, older V1, live WebSocket code,
unrelated UI redesign, raw trade/quote files, public deployment, and broad
performance work. If a missing source area becomes necessary, revise the scope
and obtain owner approval before inspection because C12 is outside the C7-C11
program's automatic correction authority.

### Likely proof and slicing boundaries

| Likely proof boundary | Requirements | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| `B-C12-REST` feasibility benchmark | C4 dependency only; no product proof | Provider/plan/host-specific acquisition time, request/page/retry/byte/row scaling and full-day forecast | Exact owner-authorized provider run |
| `P-C12-RUNTIME` replay-to-snapshot scenario | `PG-REPLAY-01`, `PG-UI-01`-`02`, `ARCH-OWN-01`-`04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-01`-`03`, `LIFE-PUBLISH-01`-`02` | Real artifact playback publishes coherent changing replay/nonlive API snapshots and clean terminal state without another owner | Current seam reconnaissance |
| `P-C12-DETERMINISM` cross-pace API trace | `PG-REPLAY-01`, `DTE-REPLAY-03`, `DTE-COMMIT-04` | Equivalent logical checkpoints expose equivalent publication meaning at two paces | Exact capture comparison design |
| `P-C12-INDEPENDENCE` dashboard restart | `PG-UI-01`, `ARCH-OWN-03`, `LIFE-LIVE-05` | UI-only restart leaves replay/API progress alive and reconnects without browser recomputation | Chrome control and deterministic artifact fixture |

**Provisional delivery assessment:** Two sequential slices after benchmark and
completed-contract approval: backend replay composition/API first, then replay
dashboard state/Chrome independence. If current reconnaissance proves no C10/
C11 change is necessary, the completed contract may combine them only when the
cross-component proof remains reviewable.

**Boundary checkpoint:**

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, single orchestration ownership boundary, dependencies, scope,
  and non-scope are explicit.
- [x] Replay input, state owner, logical time, readiness, T/Q, API/UI, pace, and
  failure invariants prevent a competing implementation.
- [x] Unresolved questions are evidence-dependent delivery decisions.
- [x] Reconnaissance is current-repository-only and question-driven; Version 2
  and provider access are excluded.
- [x] Likely proof boundaries and provisional sequential slices are identified.
- [x] No Version 2 code, tests, or fixtures were inspected for this skeleton.

**Independent skeleton review:** Required after owner accepts the proposed
outcome for review because C12 crosses replay lifecycle, operational status,
public API, and UI currentness boundaries.

**Skeleton drift audit:** No competing state owner, fake transport, changed
market semantics, T/Q-to-ranking dependency, or fabricated live readiness is
proposed. Adding C12 to the V1 RC gate or authorizing automatic correction
would require an explicit owner-approved program/map revision.

**Approval decision:** Pending owner review. This draft authorizes no benchmark,
reconnaissance, provider access, completed contract, or implementation.
