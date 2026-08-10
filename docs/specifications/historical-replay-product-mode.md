# Historical replay product mode

**Status:** Owner-approved implementation-ready contract. Sequential execution
is authorized after C11 and the integrated private V1 RC are finally accepted.

**Boundary approval authority:** Owner

**Boundary approval:** Approved by the owner on 2026-08-09 for the historical
observation-window workflow recorded below

**Completed-contract approval authority:** Owner

**Contract/reuse/test/slice-plan approval:** Owner-approved 2026-08-09 after
the required independent review and focused re-reviews were clean

**Standing program decisions:** The existing
[Version 1 Release Program](../v1-release-program.md) controls C7-C11 only. It
does not authorize C12 approval, implementation, credential access, or changes
to accepted C4 acquisition semantics. The owner's 2026-08-09 C12 approval
separately authorizes the exact C4-S6, C12-S1, and C12-S2 assignments,
proportionate reviews, in-scope lower-level corrections, local milestone
commits, and the retained B4 read described below. It does not authorize new
product semantics, provider calls, credential access, public deployment,
artifact copying, pushing, or history rewriting.

**Advancement mode:** `delegated_sequential` for C4-S6, C12-S1, C12-S2, and
their required reviews after C11/private V1 RC acceptance. One write-capable
slice runs at a time. Routine corrections within the approved requirements,
whitelists, trust boundaries, and deferrals are recorded and continue without
another owner message; fixed-authority conflict, new source scope, provider or
credential action, destructive/external action, or public deployment stops for
the smallest owner decision.

**Controlling Phase 1 requirements:** `PG-REPLAY-01`, `PG-REPLAY-02`,
`PG-UI-01`, `PG-UI-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-04`, `DTE-CLOCK-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-EVENT-04`, `DTE-TIMER-01`, `DTE-COMMIT-01`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-REPLAY-01`, `DTE-REPLAY-02`,
`DTE-REPLAY-03`, `LIFE-INIT-01`, `LIFE-LIVE-05`, `LIFE-REPLAY-01`,
`LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`,
`LIFE-PUBLISH-02`, `LIFE-END-02`, and `LIFE-END-03`

**Dependencies:** Finally accepted C4 aggregate replay including accepted
`C4-S5` prefix-validation/requested-end completion, and requiring approved queued
`C4-S6` bounded manual-driver cancellation to be accepted before C12-S1 may start; accepted C8
operations runtime and C10 snapshot API contracts; and the current C11 dashboard API/view-
state boundary. C7 checkpoint continuation is an accepted future-compatible
seam but is not exposed by the first C12 product workflow. C11 final Chrome
acceptance remains separate. Completed-contract work may use the stable current
C11 seam, but implementation additionally requires finally accepted C11 and an
accepted integrated private V1 RC. The owner-approved specification-map
revision now activates C12 immediately after those prerequisites.

## Contract document map

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C12 outcome, ownership, non-scope, semantic/runtime/API/UI contract, evidence ledger, proofs, assignments, and approval state | Sections 1-19; `C12-S1`, `C12-S2`; all C12 requirements and proofs | Every C12 task | C4, C8, C10, and current C11 contracts; C7 compatibility only |
| [REST feasibility benchmark](../replay-rest-feasibility-benchmark.md) | Non-authoritative provider/host/plan-specific completed evidence record and future rerun protocol | `B-C12-REST`; no product requirement or acceptance proof | REST benchmark evidence or a separately authorized future rerun | C4 downloader and this skeleton |

**Layout:** Single-file C12 contract, Sections 1-19. The file exceeds the
template's approximate parent routing target because the one observation-run
state machine, its atomic API mapping, and its two sequential assignments must
be reasoned about together to prevent mixed phase/market publications. The
separate benchmark remains evidence rather than product behavior. If later
implementation makes backend or UI work independently routable, change the
document map only through an owner-approved C12 contract revision.

**Routing rule:** C12 owns only accepted-artifact-to-local-product composition.
C4 continues to own historical acquisition, normalization, artifact trust, and
replay source/clock. Accepted `C4-S5` now validates an exact requested prefix end
and supplies truthful `requested_end` completion while still validating/sealing
the complete same-open source artifact. C12 consumes that opaque C4 fact but may
not reimplement it in its wall scheduler, forge artifact-end evidence, or become
a second replay source/clock.

**Contract-wide coverage and acceptance:** The owner approved the Sections 1-7
boundary and, after clean completed-contract review/re-reviews, approved
Sections 8-19, the current-code/empty-V2 whitelist, six primary proofs, two
slices, exact assignments, delegated sequential advancement, and the retained
B4 S2 read on 2026-08-09. This approval makes no V1 completion claim.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `implementation_active` | Owner-approved boundary and delegated sequence are active. C11/private V1 RC, C4-S6, and C12-S1 are accepted in order; the retained B4 read remains restricted to the now-authorized C12-S2 assignment. | 2026-08-09 | Execute C12-S2, then final component review/conformance |
| `B-C12-REST` | `complete` | B0-B4 completed under exact owner authorizations. B4 produced one independently reopened complete 2026-08-07 artifact: 5,691/5,691 symbols terminal-complete, 7,671,171 records, 2,584,011,150 artifact bytes; compile 12:28.835 plus reopen validation 1:32.697. REST is retained for C12; the 11.58 GB peak Go heap is a known C4 routine-use limitation, not a C12 semantic change. Licensed bytes remain ignored and uncommitted. | 2026-08-09 | Use the retained validated artifact as the first local C12 acceptance input; do not hard-code its path or identity |
| C4 requested-end correction | `complete` | Accepted `C4-S5` supplies the C4-owned requested-end seam: exact prefix application, full same-open suffix integrity, distinct engine lifecycle disposition, and intentional-suffix accounting; focused ordinary/short/race/vet verification and clean independent re-review are recorded in the C4 parent | 2026-08-09 | Consume only through the accepted C4 boundary during future C12 contract work; do not duplicate it in C12 |
| C4 bounded manual-driver correction | `accepted` | Commit `7d550f7` records accepted `C4-BOUNDED-CANCEL-01`, `P-C4-BOUNDED-CANCEL`, and `C4-S6`: context-bounded candidate/full validation, source-owned idempotent manual cancellation, retained terminal ordering, exact accounting, and the explicit `Run` result/error seam passed focused short/race proof and clean correction re-review. | 2026-08-09 | C12 consumes the accepted C4 boundary without editing C4 packages |
| Completed-contract review | `clean` | Initial required read-only `gpt-5.6-sol` medium review found one P1 and two P2s; the first focused re-review closed those and found one P2 authority-placement defect. C4 now owns the approved requirement/proof/S6 correction; the same reviewer found the focused correction clean with no remaining P1/P2 issue. | 2026-08-09 | Reuse this evidence unless implementation reopens a reviewed premise |
| Implementation | `in_progress` | Commit `3644ed1` accepted C11 and the integrated private V1 RC; commit `7d550f7` accepted C4-S6; C12-S1 is accepted below. Sequential implementation has advanced to C12-S2. | 2026-08-09 | Complete S2 and final C12 acceptance |
| `C12-S1` backend observation composition/API | `accepted` | `P-C12-WINDOW`, `P-C12-DETERMINISM`, `P-C12-RUNTIME`, and `P-C12-CONTAINMENT` pass. Focused review found terminal-publication identity, output-shutdown, and containment-proof defects; corrections retain C4's exact terminal publication, withdraw failed sentinel capture, join every CLI/API exit, and add active-Step/source-engine/API/CLI/timeout cases. The same reviewer found no remaining P1/P2. Full short, changed-package short race, vet, and diff checks pass. | 2026-08-09 | C12-S2 is authorized; retain backend boundary unless S2 evidence reopens it |
| `C12-S2` dashboard/real-artifact acceptance | `authorized_active` | Accepted S1 supplies the replay API. The exact assignment now authorizes opening the retained B4 artifact in place for the first local acceptance; copying, path/identity/row persistence, credentials, and provider calls remain prohibited. Chrome/B4 evidence is not yet run. | 2026-08-09 | Implement replay presentation, run `P-C12-INDEPENDENCE` and the one bounded `P-C12-B4`, then obtain focused review |

### C12-S1 acceptance record

The cache-only replay command now fully validates one same-open C4 artifact and
exact date/universe/prior-close binding before engine mutation, reconstructs
`[S,O0]` unpaced, and observes `(O0,O1]` against cumulative monotonic wall
deadlines. One immutable C8 capture binds the engine publication to warming,
observing, finalizing, retained, or non-success replay context; the additive
C10 schema rejects mixed phase/publication/accounting identities. Complete and
canceled terminal paths retain C4's exact terminal publication identity.
Because C4 intentionally exposes only publication ID zero for a replay-failure
sentinel, failure removes the prior capture and makes the API unavailable
instead of fabricating a C10-valid publication.

The four allocated proofs cover nonzero evaluation delay, `O0=S`, `O1<R`,
`O1=R`, quiet seconds, cache/header/artifact rejection, cross-schedule semantic
identity and post-disposition lag, atomic phase/API retention, paced-wait and
active-Step cancellation, finalization, source/engine sentinel failure, API
and CLI-output failure, and bounded join timeout. The implementation calls only
C4 `Cancel(ctx)` for replay containment and branches into replay before live
credential, provider, hydration, checkpoint, or T/Q construction. Focused
short tests, `go test -short -timeout 2m ./...`, changed-package
`go test -race -short -count=1 -timeout 5m`, `go vet ./...`, and
`git diff --check` pass. The focused `gpt-5.6-sol` medium correction re-review
is clean. Compact synthetic proof does not establish B4 scale, Chrome behavior,
provider performance, market edge, or executable expectancy; those first two
claims remain allocated to S2.

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

This approved mode is a product-review and engineering tool. It makes no claim
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
| What is the smallest additive C4 prefix-validation/requested-end seam that can use a complete `[S,R)` artifact for `O1<R` without applying later records, forging artifact end, skipping full-source integrity validation, or moving replay-source/clock ownership into C12? | The skeleton identified that accepted baseline C4 treated `O1<R` as cancellation | Resolved by accepted `C4-S5`, `C4-PREFIX-END-01`, and `P-C4-PREFIX-END` | Consume the opaque requested-end fact directly; no C12 terminal-evidence path |
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
- Arbitrary `O1<R` uses accepted `C4-S5`; full same-open suffix integrity,
  requested-end admission at exact logical `O1`, completion disposition, and
  cancellation linearization remain owned by C4.

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
| `P-C4-PREFIX-END` requested-end correction | `PG-REPLAY-01`, `ARCH-OWN-01`-`04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-01`-`03` | A complete `[S,R)` artifact truthfully supports requested `O1<R` only after exact prefix coverage/order and complete-source integrity validation; later rows are not applied; mutation, missing prefix evidence, end contradiction, and attempted C12 ownership fail without false completion | Complete and accepted in C4; reused, not re-proved by C12 |
| `P-C12-WINDOW` observation-boundary trace | `PG-REPLAY-01`, `ARCH-OWN-01`-`04`, `DTE-TIMER-01`, `DTE-COMMIT-01`-`04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-01`-`03` | `[S,O0)` reconstructs without wall waits; no intermediate ranking is presented as the observation; first observable publication at logical `O0` has exact ordinary `T=clamp(floor_second(O0-D),S,E)`; `[O0,O1)` progresses in order on a cumulative 1x target | Current source/pacer, configured-delay, and publication seam reconnaissance |
| `P-C12-RUNTIME` replay-to-snapshot scenario | `PG-REPLAY-01`, `PG-UI-01`-`02`, `ARCH-OWN-01`-`04`, `LIFE-REPLAY-01`-`03`, `LIFE-PUBLISH-01`-`02` | The validated real artifact drives changing replay/nonlive API snapshots at the selected window, then serves a retained immutable terminal snapshot without another owner | C8/C10 seam reconnaissance and the retained local B4 artifact |
| `P-C12-DETERMINISM` cross-schedule API trace | `PG-REPLAY-01`, `DTE-REPLAY-03`, `DTE-COMMIT-04` | Different warm-up wall schedules and diagnostic observation paces expose equivalent publication meaning at the same logical checkpoints | Exact capture comparison design |
| `P-C12-CONTAINMENT` runtime failure boundary | `ARCH-FLOW-04`, `DTE-REPLAY-01`-`03`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-01`-`02`, `LIFE-END-02`-`03` | Same-open artifact mutation, ordinal/group/clock/disposition/requested-end failure, and cancellation cannot leave the API/UI serving the prior ranking as replay-current or successfully ended; suppressed, canceled, retained-success, and shutdown outcomes remain exact and joined | C4 failure seams plus C8/C10/C11 terminal mapping reconnaissance |
| `P-C12-INDEPENDENCE` dashboard restart | `PG-UI-01`, `ARCH-OWN-03`, `LIFE-LIVE-05` | UI-only restart leaves replay/API progress alive and reconnects without browser recomputation | Chrome control and deterministic artifact fixture |

**Delivery assessment:** Exactly two sequential slices after owner approval:
backend observation-window composition and compatible C10 API mapping first;
then C11 replay view states, Chrome independence, and the retained B4 artifact
acceptance. The split protects the engine/artifact/order boundary from the
separate browser presentation and licensed-real-input acceptance boundary.

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
finding with no remaining P1/P2 issue. The later accepted `C4-S5` correction
now satisfies that prerequisite without transferring any meaning to C12.

**Skeleton drift audit:** No competing state owner, fake transport, changed
market semantics, T/Q-to-ranking dependency, or fabricated live readiness is
introduced. C12 remains outside the V1 RC gate. The owner's later completed-
contract decision supplies a separate map activation and delegated in-scope
correction authority without expanding the C7-C11 V1 program.

**Approval decision:** Owner-approved Sections 1-7 boundary. That approval
authorized the routed current-dependency reconnaissance completed below. It
authorizes no provider access or implementation slice. Sections 8-19 require
their recorded completed-contract review before the owner's separate approval.

## Section 8 — current-dependency reconnaissance and reuse

No Version 2 source, fixture, or test was inspected. Reconnaissance stayed at
repository commit `1b2b800a35fcb9c32ffcf9371af7a1b5d04a068c` and only inside
the approved C4, C8, C10, and current C11 seams. The B4 artifact itself and its
licensed rows were not opened, copied, or inspected.

| Current accepted seam and exact source | SHA-256 | Finding | Decision and required adaptation | Required C12 proof |
| --- | --- | --- | --- | --- |
| C4 offline command: `cmd/aggregate-replay/main.go` | `3620d6db...79073` | Cached exact-date Component 1 binding construction already disables fresh acquisition when credentials are empty; validated replay requires explicit queue/reserve/delay and one uniform pace. | Adapt cache-only binding construction into replay product startup. Do not call the compiler/downloader and do not inherit the engineering command's uniform-pace CLI. | `P-C12-WINDOW` |
| C4 source: `internal/replay/replay.go` | `71ce267a...166ab` | `Start`/`Step`/`Finish`, the one simulated clock, requested-end completion, exact source accounting, cancellation linkage, and cumulative rational pacer are accepted. One source pace begins at `S`, so it cannot directly express unpaced `[S,O0]` followed by 1x `(O0,O1]`. The manual exported path has no bounded source-owned cancellation operation; current cancellation accounting is private to uniform `Run`. | Reuse the unpaced source and opaque terminal facts only after owner-approved C4-S6 is implemented, proved, reviewed, and accepted. C4-S6 owns the smallest `Cancel(ctx)` seam; C12 then places only a monotonic wall-deadline coordinator around sequential `Step` calls. C12 never edits C4, advances logical time, stops the engine directly, or constructs a C4 terminal fact. | C4 `P-C4-BOUNDED-CANCEL`; C12 `P-C12-WINDOW`, `P-C12-DETERMINISM`, `P-C12-CONTAINMENT` |
| C4 artifact validator/cursor: `internal/replayartifact/validate.go`, `internal/replayartifact/playback/playback.go` | `23721b2d...c40dfb`, `42bdba80...d5fd11d` | `OpenValidated` requires an already assembled binding and exact header interval, retains the same read-only file description, and later revalidates all prefix/suffix bytes. The artifact header holds the only candidate trading date, but no exported bounded header-probe fact exists. Current validation and terminal/suffix cursor scans do not accept cancellation contexts. | Owner-approved C4-S6, not C12, owns the read-only candidate-header probe and context-aware first-pass and terminal/suffix validation. The probe remains untrusted and exposes no records, coverage success, artifact identity, or playback handle. C12 consumes only the accepted seam. | C4 `P-C4-BOUNDED-CANCEL`; C12 `P-C12-WINDOW`, `P-C12-CONTAINMENT` |
| C8 runtime/capture: `internal/operations/runtime.go`, `snapshot.go`, `status.go` | `42277185...43777`, `69bd6795...50b16`, `29e68f10...37717` | Current construction is live-only, starts wall timers/TQ pressure immediately, and derives `backend_ready=false/not_live_mode` for replay. Its sealed capture proves one publication/time/process sample and its ten-second joined shutdown is reusable. | Add a replay-specific runtime construction with no live adapter, wall evaluation timer, T/Q pressure loop, hydration, or checkpoint writer. Preserve sealed one-capture semantics and false production readiness. C12 owns only phase/schedule/result facts included atomically with that capture. | `P-C12-RUNTIME`, `P-C12-CONTAINMENT` |
| C10 schema/mapper/HTTP: `internal/snapshotapi/schema.go`, `mapper.go`, `http.go` | `f736609e...c349c`, `632689e6...a6650`, `a83042ea...20a8c` | The API already maps immutable replay publications and serves ended snapshots with HTTP 200. It currently rejects the accepted `replay_requested_end` lifecycle reason, has no observation bounds/phase/schedule context, and maps replay T/Q as `unselected`. | Keep `scanner.snapshot.v1` and add one optional root `replay` object that is mandatory for C12 replay responses and absent for live responses. Accept `replay_requested_end`; map T/Q as `unavailable/replay_unavailable`; suppress warm-up rows at representation time. No second endpoint or post-capture join. | `P-C12-RUNTIME`, `P-C12-CONTAINMENT` |
| Scanner composition: `cmd/scanner/main.go` | `62f0b31c...e1e38` | The existing command is live-only, requires `MASSIVE_API_KEY`, starts C5/C6/C7, and joins API/live/runtime on cancellation. | Add a mutually exclusive replay run mode to the same backend command. Replay never reads the credential environment variable and constructs none of the live provider/checkpoint path. Reuse loopback API and joined shutdown. | `P-C12-WINDOW`, `P-C12-RUNTIME`, `P-C12-CONTAINMENT` |
| C11 model/render/poller: `ui/model.js`, `render.js`, `app.js` and focused tests | `06750701...28a6`, `2589407a...878c`, `e89b32cc...aad0` | C11 intentionally rejects replay from the live-current predicate and renders every backend-not-ready response as retained noncurrent. Polling, atomic replacement, field rendering, transport freeze, and UI-only restart independence are reusable. | Add a distinct replay-authoritative state derived only from the C10 `replay` object plus existing engine facts. It is never live/current green and never changes row order/values. Preserve one endpoint, one in-flight request, and atomic whole-model replacement. | `P-C12-INDEPENDENCE`, `P-C12-B4` |

**Implementation whitelist:** No Version 2 source or fixture. Future C12
implementation may change only the current packages/files named in the two
assignments below, their focused tests/fixtures, this parent ledger, and the
minimum README/specification-map text needed to expose the approved workflow.
The owner-approved C4-S6 has its own C4-owned whitelist, proof, assignment, and
ledger; it is not part of either C12 slice.
Any new source area requires a C12 owner-approved contract revision before
inspection or use.

The accepted B4 evidence is reused only as a manifest and future local
acceptance input: trading date 2026-08-07, complete 04:00-20:00 New York
interval, 5,691 symbols, 7,671,171 records, and 2,584,011,150 artifact bytes.
Its local path, artifact identity, and licensed rows are deliberately absent
from this contract.

## Sections 9-14 — exact inputs/state, behavior, trust/failure outcomes, accounting/bounds, and evidenced edges

### 9. Detailed semantic inputs, outputs, and owned state

| Item | Exact meaning and provenance | Bound or owner |
| --- | --- | --- |
| Replay CLI mode | `cmd/scanner --run-mode replay` is mutually exclusive with the existing default `live` mode. Replay requires one artifact path, one cache directory, and exact observation clock times; it never reads `MASSIVE_API_KEY` or constructs C5/C6/C7 live/checkpoint components. | C12 process configuration; one run per process. |
| Candidate artifact header | A context-aware C4-owned read-only probe returns only schema/mode, candidate trading date, candidate `[S,R)`, and the minimum identities needed to select a cache. It is not validation, coverage, artifact identity, or success evidence. | At most one canonical header line from the exact path under the artifact byte bound; discarded on cancellation or any later mismatch. |
| Immutable binding and validated handle | The candidate date selects exact-date Component 1 caches with acquisition disabled. Context-aware `OpenValidated` then checks the complete C4 artifact against that assembled binding, `Start=S`, header `End=R`, complete-final-bars mode, every symbol/record/coverage/summary/digest byte, and configured byte/record limits. It checks cancellation between bounded line reads and immediately before handle success. | C1 owns binding; C4 owns handle/artifact trust. Same read-only file description remains open through completion. |
| Observation bounds | CLI `HH:MM:SS` values are interpreted in `America/New_York` on the validated binding date and converted once to exact UTC whole seconds `O0,O1`. Require `S <= O0 < O1 <= min(R,E)`. | Immutable C12 configuration; no date or UTC offset is accepted separately. |
| Engine/source | One clean replay-mode `ScannerStateEngine`, C8 delivery settings (`capacity=8192`, `required reserve=128`, `D=4s`), one C4 simulated clock initialized at `S`, and one `NewSourceThrough(..., Unpaced(), O1)`. C4 terminal/suffix validation observes the `Finish(ctx)` context between bounded line reads. C4 additionally exposes one idempotent bounded `Cancel(ctx)` for the manual driver: before terminal linkage it performs the existing controlled-stop/source-accounting path and waits only on the supplied shutdown context; after accepted terminal linkage it returns the already sealed complete result; suppression remains failed. | Engine/C4 own all canonical, logical-time, timer, evaluation, publication, order, cancellation, and completion facts. No checkpoint. C12 never calls engine stop/close directly. |
| Observation wall schedule | After the C4 group timer at `O0` completes, capture monotonic wall anchor `W0`. Before each later C4 group `t` in `(O0,O1]`, wait cancelably until `deadline(t)=W0+(t-O0)`. If late, wait zero; never reset the anchor or skip/reorder a group. Immediately after the group/timer disposition returns, sample the same monotonic clock `C(t)` and report `schedule_lag(t)=max(0,C(t)-deadline(t))`. | C12 owns only wall deadlines and lag. Positive lag remains authoritative relative to sealed replay logical time/`T`; it is visibly behind schedule and nonlive. Pace is fixed at 1 logical second per wall second. |
| Operational phase | Closed run-local enum `warming`, `observing`, `finalizing`, `retained_success`, `canceling`, `suppressed`, `shutting_down`. `validating_artifact` is emitted only to the local CLI because no trusted binding/publication yet exists. | C12 owns one monotonic phase cell; it cannot change engine lifecycle or ranking meaning. |
| Immutable observation capture | One atomic value contains the C12 phase/bounds/schedule/source-accounting fact and exactly one sealed C8/C10 engine/process capture. It is replaced only after a completed C4 group or phase transition. | C12 composition owns the cell; API handlers receive defensive immutable copies only. |
| Replay API context | Optional additive root `replay` object in `scanner.snapshot.v1`, present exactly for C12 replay responses and absent for live. It names phase, artifact ID, `R`, `O0`, `O1`, logical time, completion disposition, nullable schedule lag, and exact C4/C12 accounting. | C10 representation owner; at most one object, no path or licensed bytes. |
| Retained successful end | Exact final engine publication plus `completion=artifact_end|requested_end`, phase `retained_success`, and reconciled terminal accounting remain immutable and HTTP-readable until operator shutdown. | C12 retains one capture only; engine is already terminal and accepts no facts. |

The additive C10 representation is exact. In the table below, `d` is a uint64
base-10 string, `u` is a nonnegative JSON integer no greater than `2^53-1`, and
`t` is canonical UTC RFC3339Nano. The existing root gains optional
`replay:object`; it is absent for every live response and required for every
C12 replay response. Unknown additive properties remain governed by C10's
within-major compatibility rule.

| Path | Required properties and exact type/meaning |
| --- | --- |
| `replay` | `phase:string`; `artifact_id:string` (`sha256:` C4 identity, never a path); `artifact_end:t`; `observation_start:t`; `observation_end:t`; `logical_time:t`; `completion:string`; `schedule_lag_ms:u?`; `source:object`; `window:object` |
| `replay.phase` | `warming|observing|finalizing|retained_success|canceling|suppressed|shutting_down` |
| `replay.completion` | `""|artifact_end|requested_end`; nonempty exactly with `phase=retained_success` and matching ended engine disposition |
| `replay.schedule_lag_ms` | `null` in warming/canceling/suppressed/shutting-down; nonnegative from the first `O0` observation capture through finalizing/retained success, preserving the last measured lag after end |
| `replay.source` | C4 decimal strings `artifact_records`, `completed_record_dispositions`, `intentionally_unapplied_suffix_records`, `unread_records`, `planned_groups`, `completed_groups`, `active_group`, `remaining_groups`, `completed_runs`, `failed_runs`, `canceled_runs` |
| `replay.window` | C12 decimal strings `warmup_groups_planned`, `warmup_groups_completed`, `warmup_group_active`, `warmup_groups_remaining`, `observation_seconds_planned`, `observation_seconds_completed`, `observation_second_active`, `observation_seconds_remaining`, `observation_boundaries_published` |

All times agree with the validated binding/artifact and satisfy
`S <= logical_time <= O1 <= artifact_end <= E`. The mapper validates the exact
source/window identities in Section 12. It adds ranking reason
`replay_warming`, T/Q reason `replay_unavailable`, and lifecycle reason
`replay_requested_end` to their compatible C10 enum domains. No existing field
name, type, unit, identity, or live-mode relationship changes.

**Construction guarantees:** run-mode parsing prevents live and replay
components from coexisting; no trading-date argument can compete with the
artifact/binding date; only C4 creates validation/group/end facts; one driver
calls C4 `Start/Step/Finish` sequentially; no API capture can join phase from
one boundary with engine rows from another; warm-up cannot expose a ranking;
the browser receives no pace control; and only successful C4 completion can
construct `retained_success`.

**Runtime validation still required:** candidate-header syntax, exact-date
cache availability/currentness, binding/header/identity/coverage/digest match,
artifact type/size/record bounds, observation conversion/range, every C4 and
engine disposition, monotonic schedule arithmetic, C4 cancellation ordering,
phase/lifecycle/accounting coherence, API encoding bound, and joined shutdown all
remain representable failures and fail closed.

### 10. Required behavior and exact operator configuration

The production operator workflow is:

```text
go run ./cmd/scanner \
  --run-mode replay \
  --replay-artifact /absolute/private/path/to/aggregate-replay.jsonl \
  --reference-dir /absolute/private/path/to/exact-date-reference-cache \
  --observation-start 09:30:00 \
  --observation-end 09:35:00 \
  --api-address 127.0.0.1:8080 \
  --allow-origin http://127.0.0.1:4173

go run ./cmd/dashboard \
  --address 127.0.0.1:4173 \
  --api-origin http://127.0.0.1:8080 \
  --assets ui
```

`--run-mode` defaults to `live`, preserving the accepted live command. Replay
requires every replay flag above and rejects `--trading-date`, explicitly set
live REST/WebSocket/checkpoint flags, positional arguments, subsecond clock
text, and duplicate scalar flags (the existing repeatable `--allow-origin` is
unchanged). Replay accepts no production pace or evaluation-delay flag:
ordinary pace is exactly 1x during observation and production
`D=4s` is reused. A package-private/test configuration may supply another
positive rational pace or a fake monotonic waiter only to prove schedule
invariance; it is never CLI or browser state. C4 validation uses the accepted
current limits of 8 GiB and 1,000,000,000 records, which contain the retained
B4 artifact. Loopback/API/CORS and ten-second shutdown defaults remain C8/C10
values.

| Requirement | Exact required behavior | Authority/evidence |
| --- | --- | --- |
| `C12-CONFIG-01` | Parse the mutually exclusive mode and exact flags above. In replay, derive the date from the candidate header then validate it through the exact binding; never read credentials or create provider/checkpoint work. | `LIFE-INIT-01`, `PG-UI-01`; current scanner/C4 CLI seams. |
| `C12-ARTIFACT-01` | Before engine mutation, fully validate one complete-final-bars artifact with `Start=S`, `R>=O1`, exact binding/universe/date/coverage/order/count/digest, regular-file/no-follow bounds, and same-open handle. Consume accepted C4-S6 candidate-header and context-aware validation seams without reimplementing them; candidate header/cache failure is not a partial replay. | `DTE-REPLAY-01`, `LIFE-REPLAY-01`; accepted C4 validator and required accepted `C4-BOUNDED-CANCEL-01`. |
| `C12-WINDOW-01` | Start one clean engine at `S`; drive C4 groups/timers unpaced through `O0`; expose no warm-up rows; atomically make the first observation capture only after the `O0` timer. Its target is exactly `clamp(floor_second(O0-4s),S,E)`. Continue the same source/engine through `O1`. | `DTE-CLOCK-04`-`06`, `DTE-TIMER-01`, `DTE-COMMIT-01`-`04`, `DTE-REPLAY-02`-`03`. |
| `C12-SCHEDULE-01` | Anchor once at `W0` after `O0`; target every later boundary cumulatively at `deadline(t)=W0+(t-O0)`. Immediately after the C4 group/timer disposition, sample `C(t)` and publish `max(0,C(t)-deadline(t))`. Processing overruns create positive visible lag and zero next wait, not drift, skipped seconds, changed engine time, lost replay authority, or a false on-time claim. | `PG-REPLAY-01`, `DTE-REPLAY-03`, accepted C4 sequential source. |
| `C12-PHASE-01` | Phase order is `validating_artifact -> warming -> observing -> finalizing -> retained_success -> shutting_down`, or the first applicable `canceling|suppressed -> shutting_down`; no backward transition exists. `finalizing` begins after the `O1` group while C4 validates terminal/suffix evidence. | `LIFE-REPLAY-03`, `LIFE-END-02`-`03`; accepted C4 requested-end linkage. |
| `C12-API-01` | Serve one atomic `scanner.snapshot.v1` with additive replay context. Warm-up maps rows to empty and ranking to `unavailable/replay_warming`. Observing/finalizing copy the exact engine ranking and rows, preserve `backend_ready=false/not_live_mode`, and map every Tape/Spread value to `unavailable/replay_unavailable`. `replay_requested_end` is a valid ended reason. | `PG-UI-01`-`02`, `PG-OBS-03`, `ARCH-OWN-03`, `LIFE-PUBLISH-01`-`02`; C10 seam. |
| `C12-TERMINAL-01` | After C4 terminal success and final engine publication, retain exactly that immutable ended snapshot with exact completion disposition until operator stop. No timer, polling sample, UI restart, or late result changes its publication ID, rows, `T`, bounds, or completion. | `LIFE-REPLAY-03`, `LIFE-END-03`, `LIFE-PUBLISH-01`-`02`. |
| `C12-CONTAIN-01` | Artifact/source/clock/disposition/accounting/API failure and pre-link cancellation atomically remove replay-authoritative presentation. The sequential driver invokes only accepted C4-S6 `Cancel(shutdownCtx)`, never engine stop/close; it publishes a bounded non-success phase if the API is still available and joins within ten seconds. C12 proves the composition result and API visibility while reusing C4's terminal-link outcome/accounting proof. Post-link successful requested end wins exactly as C4 defines. | `ARCH-FLOW-04`, `LIFE-SUPPRESS-01`-`03`, `LIFE-END-02`-`03`; `C4-FAIL-01`, required accepted `C4-BOUNDED-CANCEL-01`. |
| `C12-INDEPENDENCE-01` | The dashboard recognizes warming, replay-authoritative observation, schedule lag, finalizing, retained success, cancellation/suppression, and transport-frozen states from server facts only. Restarting only the UI reconnects to the same advancing or retained backend publication. | `PG-UI-01`-`02`, `LIFE-LIVE-05`; current C11 poll/render boundary. |
| `C12-B4-01` | The first real acceptance opens the retained validated 2026-08-07 artifact in place, uses its exact caches, observes `[09:30:00,09:35:00)` New York, and completes requested-end/full-suffix validation without download, copy, committed artifact path/identity/rows, or a provider/live/trading-edge claim. | Direct owner requirement; completed B4 manifest and accepted C4 trust boundary. |

The CLI emits one bounded JSON status record per phase transition and one final
run result. Records may contain phase, validated date/bounds, record/group
counts, lag, outcome, and bounded reason, but never credential material,
artifact path, symbol/row data, or provider continuation values. Before full
validation only `validating_artifact` is truthful; the API is not started and a
dashboard opened early remains in its ordinary connecting state.

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Candidate header to binding/cache selection | Canonical C4 header supplies one supported date and `[S,R)` candidate; cache loading has network/credentials disabled. | Malformed/unsupported date, missing/noncurrent cache, or any later identity mismatch fails before engine construction. | Treating a header date as trusted binding evidence or silently fetching a replacement cache. |
| Validated handle to replay runtime | Ordinary C4 validation proves complete mode, exact binding/universe/date/session interval, every symbol/record/coverage/count/digest, size/record bounds, and same read-only file description. | Any mismatch/truncation/mutation/partial artifact closes the handle and starts no engine/API. | A sparse or corrupted artifact appearing as whole-population warm-up. |
| Warm-up to first observation | Every C4 group and timer through `O0` completed in order; sampled engine logical time is `O0`; the sealed capture has exact production-delay target and no contradictory disposition. | Missing/rejected group/timer, wrong time, suppression, or accounting mismatch cannot enter `observing`; warm-up capture exposes no rows. | Publishing an `O0` label over an earlier `T`, or leaking a fast-forward ranking as the requested observation. |
| Phase plus engine capture to API | One atomic cell contains matching artifact/binding, phase, logical boundary, source accounting, engine publication, and process sample. Mapper validates every phase/lifecycle/completion relationship. | Mixed capture, invalid enum/accounting, over-bound response, or replay/live contradiction returns no product snapshot. | Joining `retained_success` from one sample to pre-end rows from another. |
| Cumulative schedule to operator | Each later group is called only after its one anchor-derived deadline or immediately when late; lag is sampled after its group/timer disposition from the same monotonic clock. Positive lag preserves exact replay authority relative to logical time/`T` while visibly stating the run is behind its wall schedule. | Overflow/regression/wait failure cancels; lag never changes logical time, rows, `T`, or completion. | Resetting the deadline after slow processing, reporting pre-processing zero lag, or treating positive lag as invalid market state. |
| C4 end to retained success | C4 opaque artifact/requested-end fact is accepted, source/engine accounting reconciles, engine publication is ended at `O1`, and completion disposition matches `O1==R` versus `O1<R`. | EOF, suffix scan alone, canceled context, suppression, rejected end, wrong `T`, or incomplete drain cannot construct the retained phase. | Retaining a controlled stop or suppressed prefix as a successfully ended observation. |

### 11. Failure, cancellation, retention, and shutdown

| Situation | Required phase/API consequence | Process consequence |
| --- | --- | --- |
| Configuration, candidate-header, cache, or first-pass validation failure | API never starts; one bounded CLI failure, no engine mutation. | Close opened descriptors; nonzero exit. |
| Cancellation during validation | No API/snapshot claim. | Close descriptor; controlled canceled result; joined exit. |
| Cancellation during warm-up/observation/finalizing before C4 terminal-fact linkage | Atomically replace any observation capture with `canceling`, `ranking_current=false`, no replay-authoritative rows, then stop serving. | The driver returns from its cancelable wait/group/finalization path and invokes C4 `Cancel(shutdownCtx)` exactly once. C4 owns controlled stop, source outcome/accounting, engine close/drain, and the ten-second-bounded wait; C12 then stops API and closes the handle/runtime. |
| Cancellation after C4 requested/artifact-end fact links and succeeds | Successful terminal fact wins; publish `retained_success`. | Remain retained until the operator issues a subsequent shutdown or the already-issued signal proceeds only after the success capture is installed; never reclassify success as canceled. |
| Artifact mutation, ordinal/group/clock/disposition/accounting contradiction | Publish `suppressed` with fixed reason and no contracted current rows if a coherent failure capture exists. | C4/engine terminal replay failure; stop API and join within ten seconds; corrected input requires a new process. |
| API startup failure | No observation is exposed. | Cancel source/runtime and join; nonzero exit. |
| API failure while replaying | Prior capture is no longer reachable as current service. | Cancel the replay and join; nonzero exit. |
| Dashboard failure/restart | API/backend phase and replay continue unchanged. Browser shows disconnected/frozen until it reconnects. | No backend lifecycle transition. |
| Successful `O1` | Exact terminal replay snapshot is `ended` with `replay_end` or `replay_requested_end`; phase is `retained_success`; rows are final relative to replay `T`, not live. | Engine/source are terminal; API and sampler remain process-live solely for read-only inspection. |
| Operator stop while retained | Retained capture remains immutable until listener closure. | Stop API, close handle/runtime resources, and join within ten seconds; clean exit. |

`process_live` means the backend/API process remains available, including while
retaining success. `backend_ready` is always false in C12 because replay is not
production live. `ranking_current` may be true relative to replay logical time
during observation/finalization; retained success is labeled final rather than
live-current. These are not contradictory claims.

### 12. Accounting and observability

C12 adds no symbol population. The engine's accepted `PG-OBS-01` population
identity remains the sole symbol accounting. C4 source accounting remains:

```text
artifact_records
  = completed_record_dispositions
  + intentionally_unapplied_suffix_records
  + unread_records

planned_groups = completed_groups + active_group + remaining_groups
completed_runs + failed_runs + canceled_runs = 1  (after terminal outcome)
```

For the C12 phase split:

```text
warmup_groups_planned = seconds(O0-S) + 1
observation_seconds_planned = seconds(O1-O0)
C4 planned_groups = warmup_groups_planned + observation_seconds_planned

warmup_groups_planned
  = warmup_groups_completed + warmup_group_active + warmup_groups_remaining
observation_seconds_planned
  = observation_seconds_completed + observation_second_active
  + observation_seconds_remaining
observation_boundaries_published = 0 before O0,
                                   1 + observation_seconds_completed after O0

runs_started = active_runs + completed_runs + failed_runs + canceled_runs = 1
```

`completed_runs=1` means C4 terminal success even while the API retains the
capture. Shutdown is process cleanup, not a second replay outcome. Artifact
bytes/records, group counts, observation seconds/boundaries, current phase,
logical time, schedule lag, API capture count, and shutdown result are bounded
diagnostics. Per-symbol, artifact path/identity as a metric label, wall-time
histories, and row histories are prohibited. No percentage-complete estimate is
shown during whole-file validation because the current validator exposes no
proved stable progress boundary.

### 13. Simplicity and boundedness

- Mutable nonmarket state is one sequential C12 driver and one immutable-capture
  cell. C4 owns the cursor/clock; the engine owns scanner truth; C10 owns wire
  representation; C11 owns rendering.
- There is no event bus, replay record queue, worker pool, duplicated snapshot,
  database, checkpoint, saved session, browser scheduler, or UI replay control.
- The only retained product object is one latest capture. API response remains
  at most 1 MiB, rows at most 20, polling one request at a time, and the browser
  retains one last-valid model.
- Warm-up work is finite: exactly `seconds(O0-S)+1` C4 groups. Observation work
  is finite: exactly `seconds(O1-O0)` later groups. Every wait is context-
  cancelable and derived from one monotonic anchor.
- Artifact limits are 8 GiB and one billion records; engine queue is 8,192 with
  128 required reserve; API/UI timeouts and shutdown remain accepted C8-C11
  bounds.
- Reusing C4 `Step` is simpler and safer than changing C4's source to own an
  operator observation phase. Reusing one additive C10 object is simpler than
  a second replay endpoint or joining API responses in the browser.
- No Version 2 mechanism, speculative provider edge case, generic abstraction,
  public service, or new persistence path is introduced.

### 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| `O0=S` | Owner-approved boundary and half-open session invariant | Warm-up consists of the `S` timer only; first observation is at `S` with clamped target `S`. | `P-C12-WINDOW` |
| `O1=R` versus `O1<R` | Accepted `C4-S5` | Use artifact-end versus requested-end completion exactly; both may retain only after matching engine success. | `P-C12-RUNTIME` |
| Nonzero production `D` | C8 accepted four-second delay and skeleton review regression | At `O0`, target is `O0-4s` clamped, never relabeled `O0`. | `P-C12-WINDOW` |
| Slow group processing | Cumulative rational pacer and owner 1x decision | Report lag and use zero wait until caught up; never reset cadence or drop logical seconds. | `P-C12-DETERMINISM` |
| Same-open suffix mutation or wrong requested-end clock | Accepted `P-C4-PREFIX-END` | Suppress/fail; no retained completion. C12 reuses the C4 proof and injects the exported failure consequence only. | `P-C12-CONTAINMENT` |
| Cancellation around terminal linkage | Accepted C4 terminal-fact FIFO proof | Pre-link is canceled; accepted post-link success wins and is retained. | `P-C12-CONTAINMENT` |
| UI restart or API disconnect | `PG-UI-01`, current C11 poll/freeze evidence | Backend continues; browser reconnects to advancing or retained publication and never computes missed states. | `P-C12-INDEPENDENCE` |
| Complete symbols with no prints | B4 has 172 successful-empty symbols; C4 coverage invariant | Preserve exact no-print population; do not fabricate rows/marks or call validation incomplete. | `P-C12-B4` |
| Retained artifact exceeds process memory preference | B4 2.584 GB artifact and 11.58 GB compile peak | Replay opens in place and streams; no copy/compile/download. C12 makes no replay-memory claim until measured. | `P-C12-B4` |

## Sections 15-17 — primary proofs, sequential slices, exact assignments, discretion, and prohibitions

### 15. Primary proof allocation

| Requirements | One primary proof | Claim and dangerous counterexample | Observable result and limitation | Evidence/fixture | Slice |
| --- | --- | --- | --- | --- | --- |
| `C12-CONFIG-01`, `C12-ARTIFACT-01`, `C12-WINDOW-01` | `P-C12-WINDOW` deterministic observation-boundary trace | Exact cache-only config/full validation, unpaced `[S,O0]`, no leaked warm-up rows, and exact delayed first target; detects a header-trusted binding, implicit download, skipped timer, or false `T=O0`. | Exact phase/capture/source/engine trace through `O1`; no provider, wall-speed, browser, or large-artifact claim. | Small repository-owned complete/invalid artifacts and fake caches | `C12-S1` |
| `C12-SCHEDULE-01` | `P-C12-DETERMINISM` cross-schedule capture trace | 1x, virtual-late, and accelerated diagnostic schedules produce identical engine/public API facts at equal logical boundaries except schedule diagnostics; each group asserts exact post-disposition `max(0,C(t)-deadline(t))`; detects deadline reset, pre-processing lag sampling, skipped group, lag removing replay authority, or pace entering market facts. | Byte/semantic comparison of publication/replay context after excluding sample wall diagnostics, plus exact lag/UI-behind-schedule oracle; no host latency/SLA claim. | Same compact artifact and fake monotonic waiter | `C12-S1` |
| `C12-PHASE-01`, `C12-API-01`, `C12-TERMINAL-01` | `P-C12-RUNTIME` loopback observation scenario | Atomic warming/observing/finalizing/retained mapping, replay-specific T/Q unavailability, accepted requested/artifact end, and immutable retained reads; detects mixed phase/publication or controlled stop labeled success. | Multiple HTTP samples plus exact final identity/rows/accounting and continued read-only service; no Chrome or real-data scale claim. | Compact complete artifact with both `O1<R` and `O1=R` cases | `C12-S1` |
| `C12-CONTAIN-01` | `P-C12-CONTAINMENT` composition fault/cancellation/shutdown matrix | Every representable C12 failure removes replay-authoritative presentation; the driver invokes only accepted C4-S6 `Cancel(ctx)` and owned work joins in ten seconds. Detects direct engine stop, a C12-local cancel/accounting path, stale prior rows served as successful end, or API failure leaving replay work active. | Exact C12 phase/API visibility, consumed C4 terminal outcome/accounting, process exit, and goroutine join for pre-step wait, active step, finalization, API fault, and shutdown. `P-C4-BOUNDED-CANCEL` owns scan internals, cancel idempotence/deadlines, and terminal-link precedence; C12 does not duplicate them. | Compact C12 clock/disposition/API/cancel composition fixtures plus accepted C4-S6 seam | `C12-S1` |
| `C12-INDEPENDENCE-01` | `P-C12-INDEPENDENCE` model plus Chrome replay-state scenario | Browser distinguishes warming, replay-authoritative, lagged, finalizing, retained, suppressed/canceled, and frozen/disconnected; UI restart cannot alter backend progress/identity. | Pure model matrix plus Chrome at 1440x900 and UI-only restart; no other-browser/public/live claim. | Deterministic S1 loopback fixture | `C12-S2` |
| `C12-B4-01` | `P-C12-B4` first real local acceptance | The retained validated 2026-08-07 complete artifact drives the exact 09:30-09:35 ET product path, changing rows/fields and successful requested end without copy/download/credentials; detects subset/cached fixture substitution or unfinished suffix validation. | Manifest match, warm-up duration, five-minute 1x observation, schedule lag, changing publication IDs/rows, exact terminal accounting, retained API, UI restart, and clean shutdown. It proves no provider SLA, live chronology/latency, correction-arrival history, edge, expectancy, or routine compiler memory. | Private retained B4 artifact and exact-date caches; path/identity/rows not committed | `C12-S2` |

The proofs distinguish measurement correctness and product-path execution from
market prediction. A changing historical ranking is descriptive replay
evidence, not predictive or executable expectancy.

### 16. Two sequential implementation slices and assignments

| Slice | Coherent outcome | Requirements/proofs | Entry state and allowed boundary | Acceptance and explicit deferral |
| --- | --- | --- | --- | --- |
| `C12-S1` | One cache-only backend command validates a complete artifact, reconstructs through `O0`, serves atomic replay-labeled 1x observations through `O1`, contains failure, and retains only successful end. | `C12-CONFIG-01` through `C12-TERMINAL-01`, `C12-CONTAIN-01`; `P-C12-WINDOW`, `P-C12-DETERMINISM`, `P-C12-RUNTIME`, `P-C12-CONTAINMENT` | Finally accepted C4 including accepted `C4-S6`, C8, C10, and C11, plus owner-approved specification-map/sequence activation (or precise owner exception). May change `cmd/scanner`, add one focused C12 composition package, make only the C8 replay-capture and C10 additive-schema corrections named above, and add focused tests. C4 packages are excluded. | Record exact behavior/diff/proofs/limitations/review. Dashboard replay-state changes and B4 licensed-row run remain deferred. |
| `C12-S2` | The independent dashboard presents every replay state without market logic, and the exact retained B4 artifact completes the first real observation/retention/restart acceptance. | `C12-INDEPENDENCE-01`, `C12-B4-01`; `P-C12-INDEPENDENCE`, `P-C12-B4` | Accepted S1 API. May change `ui` model/render/styles/fixtures/tests, `cmd/dashboard` only if configuration text is required, focused C11 proof fixtures, README workflow, and this ledger. | Record model/Chrome/B4/clean-shutdown evidence and limitations. No provider requests, artifact copy, public deployment, or market-hours/trading claim. |

#### Exact `C12-S1` implementation assignment

1. **Authority and requirements:** Confirm accepted C4-S6, C11 final
   acceptance, and the owner-approved specification-map/sequence activation
   (or recorded precise exception), then read this complete parent, accepted C4
   parent/source/artifact details, C8 and C10 contracts, and the exact current
   source whitelist in Section 8. Implement only `C12-CONFIG-01` through
   `C12-TERMINAL-01` and `C12-CONTAIN-01`.
2. **Outcome/owner:** C12 owns one phase/wall-schedule/immutable-capture
   composition. C4 remains artifact/source/clock/end owner; the engine remains
   state/evaluator/publication owner; C8/C10 retain process/capture/schema/HTTP
   meanings.
3. **Allowed files/packages:** `cmd/scanner`; one narrowly named package such as
   `internal/replaymode`; exact required edits in `internal/operations` and
   `internal/snapshotapi`; focused tests/fixtures; this parent ledger. C4
   packages are prohibited because C4-S6 must already be accepted. No other
   source without owner-approved revision.
4. **Reuse/evidence:** Version 2 whitelist is none. Reuse C4 validation through
   its narrow context-aware wrapper, `NewSourceThrough`,
   `Start/Step/Finish/Cancel`, C8 configuration/shutdown/capture
   patterns, and C10 loopback mapper/handler. Tests use only repository-owned
   synthetic artifacts and caches; no B4 path or row read in S1.
5. **Primary proofs:** Implement the four exact proofs allocated above. The
   compact artifacts must include quiet seconds, nonzero `D`, `O0=S`, `O1<R`,
   `O1=R`, late schedule, invalid header/cache/artifact, C12 cancellation during
   the pre-`Step` wait, an active step, and finalization, source/engine/API
   failure, consumption of the accepted C4 outcome/accounting, and join
   timeout. Reuse `P-C4-BOUNDED-CANCEL` for scan-level cancellation and both
   sides of terminal linkage; do not duplicate that proof in C12.
6. **Verification tier/bounds:** Narrow tests first; then
   `go test -short -timeout 2m ./...`; focused race for changed Go packages with
   `-timeout 5m`; `go vet ./...`; `git diff --check`. Virtual schedules belong
   in ordinary/replay tests; no wall-session simulation. No command exceeds
   five minutes in S1.
7. **Deferred/discretion:** Private helper/type/file names, monotonic-wait
   mechanics, immutable-cell mechanism, bounded error text, and fixture builder
   shape are delegated. C11 rendering and B4 acceptance remain unavailable.
8. **Prohibited:** No provider/credential access, compile/download, artifact
   schema change, checkpoint, second clock/source/evaluator/publication, browser
   pace, live readiness, UI edit, V2 inspection, public binding, database, or
   third-party dependency.
9. **Dangerous counterexamples/limitation:** The assignment must directly reject
   a candidate header used as trust, a warm-up row leak, a reset 1x deadline, a
   mixed phase/publication, and canceled/suppressed state retained as success.
   Passing compact proofs does not establish B4 scale, Chrome behavior, provider
   performance, or trading value.
10. **Gate/review:** Owner approval and sequence activation are recorded. This
    assignment remains queued until C4-S6, C11, and the integrated private V1
    RC are finally accepted; the orchestrator then activates S1. Its source/
    cancellation/order/atomic-capture boundary
    requires a `gpt-5.6-sol` medium read-only focused slice review. Correct
    findings, rerun the narrowest proof, update this sole ledger, and continue
    under the separately approved delegated sequential advancement. No per-
    slice owner stop applies inside the exact assignment; C12 remains outside
    the C7-C11 V1 program.

#### Exact `C12-S2` implementation assignment

1. **Authority and requirements:** Read this parent, the accepted S1 record/API
   fixture, current C11 contract, and only the Section 8 C11 source whitelist.
   Implement `C12-INDEPENDENCE-01` and `C12-B4-01` only.
2. **Outcome/owner:** C11 remains presentation owner. It consumes one atomic C10
   response and adds a replay-authoritative/nonlive presentation branch; it does
   not infer phase, logical time, completion, values, row order, or readiness.
3. **Allowed files/packages:** `ui` assets/model/render/fixtures/tests;
   `cmd/dashboard` only for already-defined local configuration; focused Chrome
   harness/fixtures; README workflow; this ledger. Backend corrections reopen
   S1 rather than being patched in the browser.
4. **Reuse/evidence:** Version 2 whitelist remains none for C12. Reuse current
   C11 polling, atomic renderer, table/field formatting, transport freeze,
   accessibility, and independent server. The first real input is only the
   private retained B4 artifact/caches after explicit S2 activation authorizes
   that local licensed-row read; do not copy it or record its path/identity/rows.
5. **Primary proofs:** `P-C12-INDEPENDENCE` covers the exact pure model and
   Chrome state matrix. `P-C12-B4` uses 2026-08-07 `[09:30:00,09:35:00)` New
   York and must observe more than one publication boundary; it validates the
   B4 manifest, exact retained-success accounting, API immutability, UI-only
   restart, and joined shutdown.
6. **Verification tier/bounds:** Run focused Node/model tests and C11 static
   checks, then `go test -short -timeout 2m ./...`, changed Go-package race under
   five minutes, and `git diff --check`. Chrome proof has a five-minute bound.
   The one B4 acceptance has a 15-minute hard command bound including full
   validation, warm-up, five-minute 1x observation, suffix finalization, and
   clean shutdown. Before timing, confirm only the private manifest values:
   2,584,011,150 bytes, 5,691 symbols, 7,671,171 records, complete mode, and
   2026-08-07 full-session interval.
7. **Deferred/discretion:** Exact replay status copy, CSS token choice within
   current visual semantics, and fixture mechanics are delegated. No market
   threshold or new interaction is permitted. Broader accessibility/browser
   certification and charts remain deferred.
8. **Prohibited:** No provider request/credential read, artifact compile/copy/
   commit, path/identity/row persistence, client sorting/calculation/readiness,
   browser pace/pause/seek, backend state overlay, public deployment, or claims
   of live latency, edge, expectancy, or routine acquisition memory.
9. **Dangerous counterexamples/limitation:** Reject a replay response presented
   as green live; a positive-lag observation that is not visibly marked behind
   schedule; a canceled/suppressed/transport-frozen capture presented replay-
   authoritative; T/Q zero shown as data; and UI restart that relinks/stops the backend. One
   historical date/window does not establish broad market behavior or a
   production SLA.
10. **Gate/review:** S2 begins only after accepted S1. The owner authorizes the
    retained B4 artifact read in place when S2 activates; that authorization
    does not permit copying, path/identity/row persistence, credentials, or
    provider calls. The replay/API/browser
    trust boundary requires a focused read-only review; final C12 acceptance
    additionally requires one final component review and delegated ledger
    acceptance.

**Independent slice-review trigger:** Both slices are triggered. S1 crosses
persisted artifact, order/clock, lifecycle, and atomic capture boundaries. S2
changes the browser's fail-closed interpretation of replay versus live and uses
licensed real input. Reviewers are read-only and edit no file.

### 17. Implementation discretion, prohibited changes, and correction conditions

Implementers may choose private helper names, package-private phase fact types,
atomic-value versus mutex mechanics, monotonic timer construction, bounded
error wrapping, fixture builders, and presentation copy/CSS within the exact
state semantics. They may not change production `D`, session/window meaning,
C4 artifact/end evidence, engine ownership, C10 existing field units/identity,
C11 market calculations, or the two-slice order.

**Prohibited changes:** another mutable market owner, alternate source/clock/
timer/evaluator, C12-owned direct engine stop/close or run-outcome accounting,
live or T/Q readiness dependency, checkpoint start, operator
pace/pause/seek, artifact download/compile side effect, new provider source,
artifact/schema rewrite, public network/security scope, database/service split,
V2 inspection, or trading-signal/expectancy claim.

**Correction and containment conditions:** Any evidence that the candidate
header can reach success, warm-up rows leak, 1x deadlines drift, C10 cannot
atomically bind phase and publication, C11 must compute a market fact, C4
requested-end/cancellation meaning is duplicated, positive lag removes replay
authority, the B4 run cannot finish within the
recorded 15-minute local bound, or a failure leaves prior rows apparently
authoritative reopens the lowest affected contract/slice. Because C12 is
outside the C7-C11 V1 program, use this contract's separate delegated authority:
revise the lowest in-scope artifact, rerun the distinguishing proof, record the
correction, and continue. Stop only if correction requires broader source
scope, a raised fixed bound, changed product semantics/architecture, provider
or credential access, or another action excluded by the approval.

## Sections 18-19 — completed-contract checklist, review, approval state, and drift audit

### 18. Completed-contract acceptance checklist

**Independent completed-contract review:** Required because C12 composes a
persisted artifact/order boundary with new operational phase and compatible
C10/C11 trust mappings. Initial read-only `gpt-5.6-sol` medium review found one
P1 and two P2 defects: the manual `Start/Step/Finish` path had no exported
bounded C4-owned cancellation operation; positive schedule lag was both
permitted and incorrectly treated as disqualifying replay authority without an
exact sampling oracle; and implementation did not require final C11 acceptance
plus owner sequence activation. Corrections add a bounded idempotent C4
`Cancel(ctx)` seam and exact cancellation proofs, define lag after each group/
timer disposition while preserving replay authority, and make C11 final plus
owner specification-map/sequence activation (or a precise exception) explicit
implementation prerequisites. Focused re-review closed those three findings
and identified one remaining P2 authority-placement defect: the new C4-owned
semantics were specified only in C12. The correction is now routed as owner-
approved `C4-BOUNDED-CANCEL-01`, `P-C4-BOUNDED-CANCEL`, and `C4-S6` in C4's parent,
source detail, proof ledger, and exact queued assignment. Accepted C4-S6 is a
hard C12-S1 prerequisite, C12's containment proof covers only composition
consequences. The same reviewer completed the final focused re-review and found
the authority placement clean with no new P1/P2 contradiction or missing
implementation-critical behavior. The reviewer remained read-only.

**Approval decision:** Approved by the owner on 2026-08-09. The approval covers
the corrected Sections 1-19 contract, current-code/empty-V2 whitelist, proofs,
two slices, exact assignments, delegated sequential implementation/reviews,
and the retained B4 read in place only after S1 acceptance. It authorizes no
provider request, credential access, artifact copy, public deployment, or
changed product semantics. Work still begins with C11/private V1 RC completion,
then C4-S6; C12-S1 cannot start until those entry conditions pass.

- [x] This parent is the complete single-file contract and sole mutable ledger;
      the separate B4 document remains non-authoritative evidence.
- [x] The larger single-file layout is justified by the inseparable phase plus
      atomic market-publication boundary and only two compact slices.
- [x] Exact C4/C8/C10/C11 dependencies and acyclic routing are recorded,
      including owner-approved C4-S6 in C4's own requirement/proof/slice ledger
      and its acceptance as a hard C12-S1 prerequisite.
- [x] No Version 2, older V1, provider credentials/requests, or B4 licensed rows
      were inspected; the implementation whitelist is `none` for Version 2.
- [x] Exact current source paths, hashes, decisions, adaptations, and proof
      obligations are recorded.
- [x] Inputs, CLI, observation interval, production delay, wall schedule,
      phase/lifecycle, API mapping, retention, failure, and shutdown are exact.
- [x] Construction-prevented invalid states and representable runtime failures
      are distinguished.
- [x] Primary symbol/source/run/group accounting identities and overlapping
      diagnostics are explicit.
- [x] Every evidenced edge is routed to one primary proof or accepted C4 proof.
- [x] Every C12 requirement has one primary proof and one slice allocation;
      the B4 layer proves a distinct private real-input product boundary.
- [x] Both slices have one coherent outcome, exact allowed boundaries,
      verification tiers, counterexamples, limitations, deferrals, and review
      triggers.
- [x] C12 implementation is sequenced after accepted C4-S6, finally accepted
      C11/private V1 RC, and the owner-approved specification-map activation
      recorded in this milestone.
- [x] S2 extends rather than replaces S1 and cannot repair backend semantics in
      the browser.
- [x] The first acceptance uses the retained B4 artifact in place without path/
      identity coupling, copy, commit, provider call, or credential access.
- [x] C12's separately owner-approved delegated correction bounds are explicit;
      no C7-C11 program authority is claimed or extended.
- [x] Required independent completed-contract review and focused re-reviews are
      clean; the final pass found no remaining P1/P2 issue.
- [x] Owner approves the completed contract, implementation whitelist/evidence,
      proofs, two slices, exact assignments, delegated sequential advancement,
      and the S2-only retained B4 read in place.

### 19. Drift audit

| Question | Yes/No | Evidence or required resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | It implements the owner-approved historical observation boundary and accepted C4 requested-end meaning. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | C12 owns only phase/wall deadlines/capture selection; C4 and the engine retain source/clock/state/evaluation/publication. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | Replay T/Q is explicitly unavailable; aggregate replay ranking remains independent and production backend readiness remains false. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | Exact `[O0,O1)`, C4 group order, and production `D=4s` use existing authority. |
| Did this add behavior without component-local evidence or explicit approval? | No | Every behavior is owner boundary, accepted dependency evidence, invariant, B4 manifest, or allocated future proof. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | C12 composes owners; candidate probing stays C4-owned and wire mapping stays C10-owned. |
| Did this add machinery without an approved need? | No | One phase/cumulative-deadline driver and one atomic capture are the minimum needed for hidden warm-up, 1x observation, and retained inspection. |
| Did Version 2 drive the Phase 1 boundary? | No | Version 2 was not inspected or whitelisted. |

There is no unresolved substantive drift. Any later substantive **yes** outside
the approved in-scope correction authority requires owner approval before
implementation continues.
