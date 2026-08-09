# Historical replay product mode

**Status:** Owner-approved boundary skeleton; detailed contract pending

**Boundary approval authority:** Owner

**Boundary approval:** Approved by the owner on 2026-08-09 for the historical
observation-window workflow recorded below

**Completed-contract approval authority:** Owner

**Contract/reuse/test/slice-plan approval:** Pending current-dependency
reconnaissance, focused review, completed Sections 8-19, and owner decision

**Standing program decisions:** The existing
[Version 1 Release Program](../v1-release-program.md) controls C7-C11 only. It
does not authorize C12 approval, implementation, credential access, or changes
to accepted C4 acquisition semantics.

**Advancement mode:** Owner-controlled completed-contract and implementation
gates; the approved boundary permits only the routed reconnaissance and review
needed to complete this contract

**Controlling Phase 1 requirements:** `PG-REPLAY-01`, `PG-REPLAY-02`,
`PG-UI-01`, `PG-UI-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-04`, `DTE-CLOCK-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-EVENT-04`, `DTE-TIMER-01`, `DTE-COMMIT-01`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-REPLAY-01`, `DTE-REPLAY-02`,
`DTE-REPLAY-03`, `LIFE-INIT-01`, `LIFE-LIVE-05`, `LIFE-REPLAY-01`,
`LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`,
`LIFE-PUBLISH-02`, `LIFE-END-02`, and `LIFE-END-03`

**Dependencies:** Finally accepted C4 aggregate replay plus one required
owner-approved additive C4 prefix-validation/requested-end correction described
below; accepted C8 operations runtime and C10 snapshot API contracts; and the
current C11 dashboard API/view-state boundary. C7 checkpoint continuation is an
accepted future-compatible seam but is not exposed by the first C12 product
workflow. C11 final Chrome acceptance remains separate.

## Contract document map

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | C12 outcome, ownership, non-scope, settled boundary, evidence questions, likely proofs, and approval state | Owner-approved Sections 1-7; detailed Sections 8-19 pending | Every C12 task | C4, C8, C10, and current C11 contracts; C7 compatibility only |
| [REST feasibility benchmark](../replay-rest-feasibility-benchmark.md) | Non-authoritative provider/host/plan-specific completed evidence record and future rerun protocol | `B-C12-REST`; no product requirement or acceptance proof | REST benchmark evidence or a separately authorized future rerun | C4 downloader and this skeleton |

**Layout:** Compact single-file C12 contract through the skeleton stage. Keep
the benchmark separate because measured acquisition feasibility is evidence,
not product behavior. Convert to a modular contract only if completed-contract
work reveals independently routable runtime/API/UI trust boundaries.

**Routing rule:** C12 owns only accepted-artifact-to-local-product composition.
C4 continues to own historical acquisition, normalization, artifact trust, and
replay source/clock. Accepted C4 currently permits successful completion only at
the artifact header end `R`; stopping at `O1<R` is cancellation, not replay
completion. The owner-approved arbitrary observation outcome therefore requires
an additive C4 contract correction that validates a requested prefix end and
delivers a truthful successful requested-end fact while still validating/sealing
the complete source artifact. C12 may consume that C4-owned fact but may not
implement it in a scheduling adapter, forge artifact-end evidence, or become a
second replay source/clock. Complete and obtain owner approval for that focused
C4 correction before C12 implementation approval.

**Contract-wide coverage and acceptance:** The owner approved the Sections 1-7
boundary below on 2026-08-09. No implementation slice, detailed Sections 8-19,
V2/current-code whitelist, product proof, or V1 completion claim is approved by
that boundary decision.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `boundary_approved` | Owner approved an accurate historical observation window with hidden accelerated warm-up, ordinary 1x observation, replay labeling, and retained terminal inspection; independent skeleton review and focused correction re-review are clean | 2026-08-09 | Complete bounded dependency reconnaissance and the required C4 requested-end correction contract |
| `B-C12-REST` | `complete` | B0-B4 completed under exact owner authorizations. B4 produced one independently reopened complete 2026-08-07 artifact: 5,691/5,691 symbols terminal-complete, 7,671,171 records, 2,584,011,150 artifact bytes; compile 12:28.835 plus reopen validation 1:32.697. REST is retained for C12; the 11.58 GB peak Go heap is a known C4 routine-use limitation, not a C12 semantic change. Licensed bytes remain ignored and uncommitted. | 2026-08-09 | Use the retained validated artifact as the first local C12 acceptance input; do not hard-code its path or identity |
| C4 requested-end correction | `required` | Independent skeleton review proved accepted C4 has no successful prefix-end seam: every group through artifact `R`, final `R` timer, all ordinals, and second-pass seal currently precede success | 2026-08-09 | Specify and obtain owner approval for one additive C4 prefix-validation/requested-end fact; do not implement it in C12 |
| Detailed contract | `pending` | Requires bounded current-repository reconnaissance, corrected clean skeleton review, and the routed C4 requested-end decision; REST acquisition itself is resolved | 2026-08-09 | Complete Sections 8-19 and obtain owner approval |
| Implementation | `not_authorized` | No slices or implementation assignment accepted | 2026-08-09 | Owner approves completed contract before any product change |

## Sections 1-4 — outcome, scope, ownership, and settled boundary

### Outcome and user consequence

C12 provides an independently observable local historical observation window.
An operator supplies one accepted complete aggregate artifact and selects an
exact observation interval `[O0,O1)` on its bound trading date. The backend
reconstructs canonical state from session start `S` through `O0` without wall-
time pacing, commits all required timers through logical time `O0`, and only then
exposes the requested interval through the sole production engine and evaluator
on a cumulative 1x wall schedule. The existing loopback snapshot API publishes
changing replay-labeled state for the independent dashboard from the first
observation boundary. The production evaluation delay `D` remains unchanged:
at logical replay time `O0`, the committed ranking watermark is
`T=clamp(floor_second(O0-D),S,E)`, not falsely relabeled as `O0`.

The operator can watch historical Last, Day %, From 4AM %, HOD drawdown, range
positions, Activity, qualification, and exact server ranking move second by
second while the market is closed, then compare that view with charts and other
historical evidence at the same timestamp. Replay is visibly nonlive and current
only relative to replay logical time and complete artifact coverage. Version 1
Tape Rate and Spread remain explicitly unavailable. Restarting or redeploying
only the dashboard does not stop, pause, relink, or mutate the replay backend.
At `O1`, after the C4-owned requested-end validation/fact succeeds, the engine
publishes its terminal replay snapshot and ends; the local API process retains
that immutable final observation for inspection until the operator stops the
process. Cancellation, integrity suppression, and successful requested end are
distinct terminal outcomes and never retain a prior apparently current ranking
under a false ended label.

This proposed mode is a product-review and engineering tool. It makes no claim
about live receipt latency, correction-arrival chronology, current provider
entitlements, predictive edge, or executable trading expectancy.

### Scope

In scope:

- one local replay composition from validated C4 artifact to the existing
  `ScannerStateEngine`, C8-style coherent operational capture, C10 snapshot API,
  and C11 dashboard;
- one operator-selected observation interval `[O0,O1)` interpreted in New York
  exchange time on the artifact's bound date, with `S <= O0 < O1 <= E`;
- validation that the artifact's binding, date, identity, complete-universe
  coverage, and interval support every required aggregate and absence fact in
  `[S,O1)` before canonical warm-up begins;
- accelerated deterministic warm-up over `[S,O0)`, including timers through
  logical time `O0`, with no warm-up ranking presented as the requested
  observation and with the ordinary configured evaluation delay `D` preserved;
- a cumulative 1x wall-time scheduling target over `[O0,O1)` by default, with
  bounded lag reported honestly rather than assuming every second's processing
  completes within one wall second; any non-1x override is confined to explicit
  test or diagnostic configuration and is never owned by the browser;
- explicit artifact, observation bounds, loopback API address, and exact
  allowed dashboard origin; the trading date is derived from the validated
  binding rather than accepted as a conflicting second identity;
- replay/nonlive lifecycle and status presentation without changing ranking or
  feature semantics;
- bounded validation, warm-up, observation pacing, replay completion,
  cancellation/suppression containment, retained-successful-terminal API
  service, API shutdown, dashboard-only restart independence, and joined
  process termination;
- deterministic equivalence at the same logical boundary regardless of warm-up
  wall duration or an explicitly selected diagnostic observation pace; and
- use of the owner-retained, independently validated 2026-08-07 complete
  artifact for the first real-data local acceptance run without committing,
  copying, or coupling C12 to its licensed bytes or local path.

Not in scope:

- raw-trade or flat-file acquisition, local trade-to-second aggregation, a
  second aggregate normalizer, or changed C4 artifact semantics;
- fake WebSocket transport, live epochs/acknowledgements, historical live
  receipt chronology, or a second scanner/evaluator;
- full-session or two-pass T/Q replay, historical NBBO reconstruction, or
  fabricated Tape Rate/Spread;
- operator-visible fast-forward, pause, seek, reverse, looping, multi-session
  playlists, browser-owned pace, or saved replay controls in the first mode;
- checkpoint-selected observation start or backend checkpoint-resume controls
  in the first mode; a clean deterministic warm-up from `S` establishes `O0`;
- downloading or recompiling provider data as an implicit side effect of
  starting an observation run;
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
| Replay input | One validated complete C4 artifact supplies the sole binding/date and complete `[S,O1)` coverage at the normalized-event boundary; no fake WebSocket and no implicit download. When `O1<R`, only the required C4-owned prefix-validation/requested-end seam may establish successful completion. | `PG-REPLAY-01`, `DTE-REPLAY-01`, `LIFE-REPLAY-01` |
| Observation interval | Operator selects exact exchange-time `[O0,O1)` with `S <= O0 < O1 <= E`; invalid, mismatched, subsecond, ambiguous, or insufficiently covered bounds fail before engine mutation | `DTE-CLOCK-04`-`06`, `DTE-EVENT-04`, `LIFE-REPLAY-01` |
| Warm-up and first observation | A clean engine consumes `[S,O0)` and required timers through logical time `O0` without wall waits. Warm-up status may be exposed, but no intermediate ranking may be presented as the requested observation. The first observable ranking is one coherent publication at logical replay time `O0` with the ordinary exact target `T=clamp(floor_second(O0-D),S,E)`; C12 never changes or hides `D`. | `ARCH-OWN-01`-`04`, `DTE-TIMER-01`, `DTE-COMMIT-01`-`04`, `DTE-REPLAY-02`-`03` |
| State and order | The sole engine/evaluator owns canonical state, logical order, timers, committed `T`, qualification, ranking, and immutable publications | `ARCH-OWN-01`-`04`, `DTE-REPLAY-02`-`03`, `LIFE-REPLAY-02`-`03` |
| Product currentness | Ranking may be exact/current relative to replay `T`, but the process is not production-live ready and must be visibly replay/nonlive | `PG-OBS-03`, `LIFE-PUBLISH-01`-`02` |
| T/Q | Aggregate ranking remains independent; historical T/Q fields are unavailable with replay-specific reason | `PG-REPLAY-02`, `LIFE-REPLAY-02` |
| API/UI | API reads one immutable publication; UI owns no market calculations or readiness, preserves the live table semantics, displays replay date/time and phase, and can restart independently | `PG-UI-01`-`02`, `ARCH-OWN-03`, `LIFE-LIVE-05` |
| Pace | Observation defaults to a cumulative 1x schedule mapping logical elapsed replay duration to wall elapsed duration. Pace changes scheduling only; lag is explicit and equivalent logical times produce equivalent engine facts and output. No browser control is required. | `PG-REPLAY-01`, `DTE-REPLAY-03`, `LIFE-REPLAY-03` |
| Completion and inspection | At `O1`, after the C4-owned requested-end fact and required timers complete, the engine publishes one successful terminal `ended` replay snapshot. The backend retains and serves that immutable final snapshot until operator shutdown without claiming production-live readiness. | `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`-`02`, `LIFE-END-02`-`03` |
| Failure and cancellation | Artifact mutation, invalid ordinal/group/clock/disposition, requested-end contradiction, cancellation, or API/UI failure cannot alter prior canonical history or leave a prior apparently replay-current observation served as successful retained end. Suppressed, canceled/controlled-stop, successful requested end, API-retained success, and process shutdown remain distinguishable and bounded. | `ARCH-FLOW-04`, `LIFE-REPLAY-03`, `LIFE-END-02`-`03` |

C12 introduces no new ranking rule, market window, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, event representation, or
browser calculation.

## Sections 5-7 — evidence questions, reconnaissance scope, likely proofs, and boundary checkpoint

### Evidence questions

| Question | Why current authority does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| What is the smallest additive C4 prefix-validation/requested-end seam that can use a complete `[S,R)` artifact for `O1<R` without applying later records, forging artifact end, skipping full-source integrity validation, or moving replay-source/clock ownership into C12? | Accepted C4 treats `O1<R` as cancellation and permits success only after all groups/ordinals through `R`, the final `R` timer, and the second-pass seal | Focused C4 contract correction plus current source/validator inspection and a dangerous prefix-mutation/end-contradiction proof | Truthful arbitrary observation end while preserving C4 ownership |
| Can the accepted C4 source/pacer reach logical time `O0` without wall waits and then target cumulative 1x only for `[O0,O1)` without changing logical order or the configured evaluation delay `D`? | C4 proves uniform pace invariance, not an observation cutoff with two wall-scheduling phases | Current source/pacer construction and deterministic trace inspection | Reuse/configure the C4 seam or add the smallest C12 wall-scheduling adapter that owns no source, clock, or publication meaning |
| Can C8 operational capture represent validation/warm-up, replay observation, production-live-not-ready, engine-ended/API-retained, and shutdown states without false readiness? | C8 was accepted around runnable live composition | Current type/construction/test inspection | Reuse C8 capture or revise its lower-level representation |
| Can `scanner.snapshot.v1` express observation bounds, replay date/time, warm-up, replaying, and retained terminal state compatibly? | C10 includes run mode/lifecycle but not necessarily the operator's observation boundary or retained-process distinction | Schema/mapper/validator fixture inspection for all observation phases | Compatible additive reuse or an explicit C10 contract/schema correction |
| What exact dashboard consequence distinguishes warm-up, 1x replay-current, retained-ended, live-current, degraded, frozen, and disconnected without redesigning the live table? | C11 currently treats overall currentness through live readiness | Current model/render/state-proof inspection and Chrome fixture | Minimal replay context and state presentation |
| What bounded startup progress is useful while millions of records are validated and `[S,O0)` is reconstructed? | Authority forbids false currentness but does not require a progress metric | Existing validator/source accounting and C8/C10 bounded-field inspection | Honest warming status without a second counter or estimated completion claim |
| Which diagnostic pace range and acceptance window prove invariance without making speed a product control? | The owner selected 1x for ordinary use; Phase 1 still requires speed-invariant evidence | Existing pace tests plus a compact observation-window fixture at 1x-equivalent and accelerated scheduling | Test-only configuration bound and acceptance timeout |

### Resolved evidence-dependent decisions

- `B-C12-REST` retained C4 REST acquisition. The complete artifact missed the
  approximate ten-minute preference but finished inside the authorized fifteen-
  minute compile bound and validated successfully. No alternate acquisition
  source is required for C12.
- The first product workflow always reconstructs from `S`; it exposes no
  checkpoint selector or replay-resume control.
- Ordinary observation pace is 1x. A non-1x setting exists only if required for
  deterministic proof or explicit diagnostics, and it is not a dashboard
  control.
- Observation completion retains the final immutable snapshot through the API
  until operator shutdown; engine `ended` and API process liveness are distinct.
- The observation clock preserves the production evaluation delay `D`. At
  logical replay time `O0`, the first observation watermark is the ordinary
  `T=clamp(floor_second(O0-D),S,E)`; matching the live scanner is more important than
  fabricating `T=O0`.
- Arbitrary `O1<R` requires an additive C4 prefix-validation/requested-end
  correction. That meaning remains owned by C4 and must be completed before C12
  implementation approval.

### Proposed reconnaissance scope

No Version 2 inspection is proposed. Current accepted repository boundaries are
the relevant dependencies. The owner-approved boundary and completed REST
benchmark now permit the detailed-contract task to inspect only:

- C4 replay command/source/pace, observation-cutoff feasibility, artifact
  validation, interval-coverage, second-pass seal, cancellation, requested-end,
  configured-delay, and terminal-publication interfaces;
- C8 runtime construction, status, snapshot capture, metrics, and shutdown;
- C10 capture source, mapper/schema, HTTP lifecycle, replay/ended status, and
  retained-final-snapshot tests;
- C11 validation/rendering of run mode, historical date/time, warm-up,
  observation, retained end, readiness, T/Q unavailable, disconnection, and
  dashboard-only restart; and
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
| `B-C12-REST` feasibility benchmark | C4 dependency only; no product proof | Complete provider artifact acquisition is feasible on this host; exact B4 accounting and validation are recorded in the ledger | Complete; routine compiler memory reduction remains C4 follow-up, not C12 scope |
| `P-C4-PREFIX-END` requested-end correction | `PG-REPLAY-01`, `ARCH-OWN-01`-`04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-01`-`03` | A complete `[S,R)` artifact truthfully supports requested `O1<R` only after exact prefix coverage/order and complete-source integrity validation; later rows are not applied; mutation, missing prefix evidence, end contradiction, and attempted C12 ownership fail without false completion | Focused C4 correction contract and owner approval |
| `P-C12-WINDOW` observation-boundary trace | `PG-REPLAY-01`, `ARCH-OWN-01`-`04`, `DTE-TIMER-01`, `DTE-COMMIT-01`-`04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-01`-`03` | `[S,O0)` reconstructs without wall waits; no intermediate ranking is presented as the observation; first observable publication at logical `O0` has exact ordinary `T=clamp(floor_second(O0-D),S,E)`; `[O0,O1)` progresses in order on a cumulative 1x target | Current source/pacer, configured-delay, and publication seam reconnaissance |
| `P-C12-RUNTIME` replay-to-snapshot scenario | `PG-REPLAY-01`, `PG-UI-01`-`02`, `ARCH-OWN-01`-`04`, `LIFE-REPLAY-01`-`03`, `LIFE-PUBLISH-01`-`02` | The validated real artifact drives changing replay/nonlive API snapshots at the selected window, then serves a retained immutable terminal snapshot without another owner | C8/C10 seam reconnaissance and the retained local B4 artifact |
| `P-C12-DETERMINISM` cross-schedule API trace | `PG-REPLAY-01`, `DTE-REPLAY-03`, `DTE-COMMIT-04` | Different warm-up wall schedules and diagnostic observation paces expose equivalent publication meaning at the same logical checkpoints | Exact capture comparison design |
| `P-C12-CONTAINMENT` runtime failure boundary | `ARCH-FLOW-04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`-`02`, `LIFE-END-02`-`03` | Same-open artifact mutation, ordinal/group/clock/disposition/requested-end failure, and cancellation cannot leave the API/UI serving the prior ranking as replay-current or successfully ended; suppressed, canceled, retained-success, and shutdown outcomes remain exact and joined | C4 failure seams plus C8/C10/C11 terminal mapping reconnaissance |
| `P-C12-INDEPENDENCE` dashboard restart | `PG-UI-01`, `ARCH-OWN-03`, `LIFE-LIVE-05` | UI-only restart leaves replay/API progress alive and reconnects without browser recomputation | Chrome control and deterministic artifact fixture |

**Provisional delivery assessment:** Two sequential slices after completed-
contract approval: backend observation-window composition/API first, then
replay context/dashboard state/Chrome independence. If current reconnaissance
proves no C10/C11 change is necessary, the completed contract may combine them
only when the cross-component proof remains reviewable.

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
- [x] The owner approved the observation-window boundary, 1x ordinary pace,
  retained final inspection, and use of the validated local B4 artifact.
- [x] Independent review exposed the accepted C4 requested-end gap, evaluation-
  delay oracle error, and missing runtime-failure proof; each is routed above
  without assigning C4 meaning to C12.

**Independent skeleton review:** Initial read-only review on 2026-08-09 found
one P1 (accepted C4 has no successful `O1<R` seam) and two P2s (false `T=O0`
oracle under nonzero `D`; missing consequential runtime-failure proof). The
corrections above route requested-end meaning to a required focused C4
correction, preserve the exact clamped `DTE-COMMIT-01` target, and add
`P-C12-CONTAINMENT`. Focused re-review and final clamp check closed every
finding with no remaining P1/P2 issue. The required C4 correction remains an
honest prerequisite, not an implemented or approved seam.

**Skeleton drift audit:** No competing state owner, fake transport, changed
market semantics, T/Q-to-ranking dependency, or fabricated live readiness is
proposed. Adding C12 to the V1 RC gate or authorizing automatic correction
would require an explicit owner-approved program/map revision.

**Approval decision:** Owner-approved Sections 1-7 boundary. This approval
authorizes the routed current-dependency reconnaissance and required read-only
skeleton review needed to complete Sections 8-19. It authorizes no additional
provider access, implementation slice, or V1 completion claim.
