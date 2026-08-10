# Version 1 Release Program

**Status:** Owner-approved C7-C11 delivery authority.

**Approved:** 2026-08-07

**Scope:** Autonomous correction and sequential completion of Components 7
through 11 into a private/local Version 1 Release Candidate (V1 RC).

This document replaces the former C7-C11 unattended-program authority. It is
an execution authority below the approved Phase 1 product and architecture
contracts. It cannot change market semantics or ownership, but it can revise
every lower-level C7-C11 delivery decision named below until final V1
acceptance.

## 1. Release outcome

Done means one locally verified private V1 RC provides:

1. a runnable Go scanner for the complete eligible U.S. common-stock and ADR
   universe, using 6,000 symbols as the normal local reference population;
2. correction-aware aggregate processing and exact ranking across duplicate,
   late, corrected, overlapping REST/live, and replay inputs;
3. session, 30-minute, and 60-minute range position, HOD drawdown, Activity,
   Tape Rate, and NBBO Spread with independent field status;
4. live trade and quote ingestion that can degrade to zero coverage without
   changing aggregate ranking or backend readiness;
5. bounded operational measurements for throughput, processing delay, queue
   growth, memory, rejected messages, and dropped or pressure-shed messages;
6. checkpoint restart and same-process recovery with explicit stale or
   unavailable output whenever completeness cannot be proved;
7. a private versioned read-only HTTP API with publication identity, coherent
   snapshots, loopback binding by default, and explicit CORS allow-listing;
8. an independently runnable Chrome-desktop dashboard that presents the V1
   scanner fields and data-health state without browser-owned market logic;
9. compact deterministic replay and fake-provider evidence sufficient to
   finish the RC while the market is closed; and
10. the separate market-hours procedure in
   [`market-hours-validation.md`](market-hours-validation.md), recorded as
   pending unless the owner later authorizes credentialed observation.

The RC claim is local and evidence-bounded. It does not claim public
availability, production scale, live-provider validation, trading edge, or
executable expectancy.

### 1.1 C7-S3 stop diagnosis

The current S3 proof used a 5,500-symbol, 30-second-gap fixture and a 4,393,853-
byte checkpoint. Its first recorded trial measured projection 62.8 ms, encode
11.3 ms, write 9.92 s, load 9.82 s, install 63.3 ms, acknowledgement 0.12 ms,
catch-up/fence 12.64 s, and end-to-end checkpoint restart 22.53 s. The command
ran all five checkpoint/fresh trials and failed after 339.48 seconds.

The hard product-facing recovery claim did not fail: 22.53 seconds is inside
the current 60-second C7 end-to-end target and materially below the product's
rejected approximately 130-second fresh-reconstruction precedent. The failure
was the C7 contract's inherited V2 premise that local discovery/decode/
validation/install complete within five seconds; load alone took 9.82 seconds.
This identifies a real lower-level decoder/storage inefficiency, but not a
product-semantic failure.

The former authority converted that evidence into an owner-interruption path in
four steps:

1. `C7-OBJECTIVE-01` and `P-C7-OBJECTIVE` made the five-second premise a hard
   slice-acceptance target and fixed a five-run fixture/proof design.
2. The C7 implementation-discretion section froze JSON, the reference fixture,
   target, proof allocation, and three-slice plan.
3. The former `AGENTS.md` and implementation process made any post-contract
   fixture, proof, slice, whitelist, interface, advancement, or threshold
   change manual.
4. S1/S2 acceptance and the old drift/advancement rules prevented reopening the
   decoder/codec decision even though downstream evidence implicated it.

The proof also did unnecessary work: it had no `testing.Short()` separation,
no per-trial deadline, repeated five full-population checkpoint/fresh paths,
and reported the decisive segment only after completing the entire 339-second
command. The 100,000-symbol structural case was correctly finite but belongs in
an isolated capacity/acceptance command, not ordinary verification or the
normal restart benchmark.

The corrected program preserves checkpoint coherence, invalid-artifact
rejection, exact continuation, honest fallback, a current end-to-end recovery
limit, and a meaningful same-host fresh-control improvement. It removes the
independent five-second release gate, uses 6,000 symbols, validates the fixture
before timing, uses three individually bounded measured trials, separates the
100,000-symbol structural proof, and permits decoder/codec/fixture/proof/slice
correction without owner intervention.

## 2. Fixed product behavior and revisable delivery decisions

The following are fixed unless the owner changes the controlling Phase 1
authority:

- the approved product formulas, windows, eligible-universe and prior-close
  meaning, qualification latch, exact Day-%/symbol order, and top-20 limit;
- one authoritative `ScannerStateEngine`, one canonical symbol state, one
  committed aggregate watermark, and one ordinary evaluator;
- the canonical aggregate identity, correction, REST/live reconciliation,
  causal ordering, checkpoint-coherence, and replay-time rules;
- aggregate ranking independence from trade/quote health, availability, and
  pressure state;
- honest `current`, `stale`, `warming`, `unavailable`, `invalid`, no-print,
  readiness, coverage, and accounting semantics;
- the V1 outcomes in Section 1; and
- the prohibition on fabricated marks, history, completeness, readiness,
  qualification, or T/Q continuity.

The V1 orchestrator may revise the following without another owner message
when the revision remains inside those fixed boundaries:

- C7-C11 component contracts, delivery ledgers, document maps, and internal
  interfaces that do not change an accepted cross-component product meaning;
- implementation mechanics, algorithms, codecs, schemas used only for local
  persistence, package layout, and bounded queue structures;
- implementation slices, sequencing within a component, advancement state,
  and previously accepted lower-level decisions;
- Version 2 reconnaissance and implementation whitelists, provided Version 2
  remains evidence rather than authority and the scope is recorded before use;
- fixtures, reference populations, proof allocation, test scale, benchmark
  method, and review routing;
- numeric delivery thresholds subordinate to a fixed product limit, including
  measured readiness, pressure, recovery, and performance settings; and
- optional hardening deferrals that do not remove a V1 outcome.

An accepted slice or component is accepted evidence, not an immutable design.
Later correctness, integration, performance, or proof evidence may reopen it.
The ledger must name the invalidated claim, preserve still-valid evidence, and
route the correction to the lowest artifact capable of fixing it.

When a lower-level artifact conflicts with a higher authority, revise the
lower-level artifact and continue. If two controlling Phase 1 statements appear
to conflict, preserve both by applying their strict compatible intersection,
the authority order above, the simplest design, and honest unavailable output
instead of inventing success. Record the interpretation and continue; do not
ask the owner to resolve an in-scope C7-C11 delivery decision.

## 3. Zero-interruption execution policy

C7-C11 have no planned owner approval gate, component-local manual mode, or
failure state that waits for an owner response. The orchestrator records every
boundary, contract, correction, acceptance, and review decision authorized by
this program and continues through final V1 RC review. A progress report,
milestone commit, failed test, missed benchmark, unsuitable fixture, review
finding, defect in accepted work, contract or document-map revision, whitelist
change, proof or slice reallocation, implementation replacement, closed market,
or resolvable uncertainty is never a turn-ending condition.

Potentially blocking conditions are contained automatically:

1. Credentialed/live-provider work, destructive action, external publication,
   paid infrastructure, authentication/TLS/hosting, public deployment, and
   production cutover are outside this private/local RC. Substitute deterministic
   replay, fake-provider, loopback, or local evidence; record the external item
   as deferred; and continue. Do not request permission during this goal.
2. Preserve unexpected user work. Work around same-path changes with a narrow
   merge or, when safe, an isolated worktree/branch; never overwrite or
   improperly combine them. If a milestone commit cannot be made safely, leave
   the exact program work in the working tree, record the evidence, and continue:
   a local commit is preferred bookkeeping, not a release-capability prerequisite.
3. Do not repair Git corruption by rewriting user history. Continue verification
   and artifact completion without the unsafe Git operation, leaving an exact
   working-tree handoff if necessary.
4. Use available local tools and reviewer models, record substitutions, reduce
   expensive experiments to finite distinguishing cases, and retry deferred
   review when capacity returns. No process wording can override a later direct
   owner instruction, reconcile literally contradictory controlling authority,
   or execute beyond a hard system/tool/usage impossibility; those are external
   termination facts, not manual program gates, and the orchestrator must first
   exhaust the containment and fallback rules above.

The orchestrator never solicits an owner decision for in-scope uncertainty.
When evidence cannot establish an optional hardening or external claim, it
records that claim as not established, preserves honest field/readiness status,
and completes the bounded RC outcome that does not depend on it.

## 4. Required correction loop

For every correctable failure:

1. preserve the failing evidence and identify the exact claim it tested;
2. classify the cause as product/architecture conflict, implementation defect,
   fixture defect, measurement defect, or an unnecessary lower-level rule;
3. revise the lowest applicable C7-C11 contract, plan, fixture, proof, test, or
   implementation decision;
4. reopen only the affected accepted item and preserve unrelated clean proof;
5. implement the correction and run the narrowest proof that distinguishes
   success from the observed failure;
6. obtain only the risk-triggered focused review described in Section 8;
7. update the component parent ledger with the new premise and result; and
8. continue in component order.

A failed correction attempt is new diagnostic evidence. It does not freeze the
attempted mechanism or restore an unsuitable prior benchmark as authority.
Never rerun an unchanged failed command. After two failed attempts using the
same mechanism or premise, abandon it, return to the simplest design consistent
with fixed authority, reduce the experiment to a compact deterministic case
that distinguishes the failure, and rebuild scale only after that case passes.

## 5. Component plan and V1 non-scope

Implementation remains sequential: finish C7, then C8, C9, C10, and C11. Keep
one active implementation slice. Contract and reconnaissance work may be
revised just in time; implementation of component N+1 waits for component N's
accepted interface and final review.

| Component | Minimum V1 outcome | Planned delivery shape | Deferred hardening |
| --- | --- | --- | --- |
| C7 checkpoints/restart | Coherent `T0`, invalid-checkpoint rejection, corrected aggregate tail and all required canonical state, exact `[T0,R)` continuation, replay continuation, bounded restart materially faster than equivalent fresh recovery | Retain the three historical slices because S1/S2 already exist; correct and finish S3, reopening S1/S2 mechanics if evidence implicates them | Remote storage, replication, migrations not needed by the local RC, 100,000-symbol operational capacity |
| C8 readiness/operations | Runnable composition, configuration/lifecycle, honest readiness/degradation/staleness, bounded recovery/shutdown, and throughput/delay/queue/memory/reject/drop measurements under controlled mixed load | Compact contract; at most two slices: runtime/lifecycle, then measurements and mixed-load acceptance | HA, multi-host claims, exhaustive soak, public monitoring stack, live-provider capacity claim |
| C9 top-20 T/Q | Selected-row subscriptions, acknowledgement-based continuous coverage, Tape Rate, NBBO Spread, field status, aggregate-first shedding to zero, and current-rank-order restoration | Compact contract; at most two slices: coverage/features, then pressure/shedding/restoration | Full-universe T/Q, full-session T/Q replay, Tape Rate attention threshold, unevidenced lifecycle reconstruction |
| C10 API | Private versioned read-only HTTP API, coherent snapshot mapping, publication identity, scanner/field status, loopback default, explicit CORS allow-list | Compact contract; one or two slices depending on whether schema identity and HTTP trust need separate proof | Authentication, TLS, public hosting, service split, production cutover, optional streaming transport |
| C11 UI | Independent Chrome-desktop dashboard with high-fidelity useful V2 layout/density/interactions, all V1 fields, and explicit health/coverage states | Compact contract; at most two slices: API/view-state integration, then visual/interaction/accessibility acceptance | Mobile, multi-browser certification, offline/PWA behavior, browser market calculations, broad design-system work |

The owner-approved boundary plans are the component entry points linked from
[`specification-map.md`](specification-map.md). Detailed contracts may change
their tentative slice count, but a third slice requires a distinct proof or
ownership boundary that cannot be reviewed coherently in two.

## 6. V1 capability and primary-proof matrix

This matrix routes release claims without duplicating already accepted lower-
layer proofs. Exact proof names for C8-C11 are finalized in their detailed
contracts.

| V1 capability | Owning component(s) | Primary proof or release evidence | What it does not prove |
| --- | --- | --- | --- |
| Complete eligible population near 6,000 symbols | C1, C8 | Accepted C1 binding proofs plus C8 6,000-symbol runtime/accounting acceptance | Current live provider population until market-hours validation |
| Duplicate/late/corrected/overlapping aggregate correctness and exact ranking | C2, C3, C6 | Existing canonical merge, feature/qualification, ranking/accounting, and REST/live integration primary proofs | Live arrival-frequency distribution or trading edge |
| Session/30m/60m ranges, HOD drawdown, Activity | C3 | Existing C3 formula/correction/availability proofs | T/Q behavior or production latency |
| Coherent checkpoint restart and aggregate replay continuation | C7 | `P-C7-STATE`, `P-C7-INSTALL`, `P-C7-LIVE`, `P-C7-REPLAY`, and corrected `P-C7-OBJECTIVE` | Live-provider SLA or 100,000-symbol capacity |
| Runnable lifecycle, readiness, bounded recovery/shutdown, and operational measurements | C8 | One deterministic runtime/recovery trace and one controlled 6,000-symbol mixed-load acceptance, each asserting distinct semantic and capacity boundaries | Capacity beyond the recorded host/fixture or live transport behavior |
| Top-20 Tape Rate, NBBO Spread, continuous coverage, shedding/restoration | C9 | One T/Q normalization/feature/coverage trace and one aggregate-first pressure trace | Full-universe T/Q or complete trade lifecycle reconstruction |
| Versioned read-only product schema and snapshot consistency | C10 | Schema golden/mutation proof plus HTTP binding/CORS/coherence proof if those are distinct boundaries | Public security, TLS, or availability SLA |
| Independent Chrome-desktop product behavior | C11 | Browser integration/visual-state proof against deterministic C10 fixtures, plus independent-run proof | Other browsers, mobile, or backend correctness already owned below |
| Closed-market release completion | C4, C5-C11 | Compact replay/fake-provider vertical V1 RC scenario that proves only cross-component wiring not already covered below | Actual provider timing, entitlements, or current universe |
| Market-hours observation | Separate procedure | Authorized observation record from `market-hours-validation.md` | Not required for the private/local RC; remains pending and proves no deterministic RC claim |

Each requirement still has one primary proof. The final V1 scenario proves
only cross-component wiring, field/status preservation, and independent UI/API
operation; it does not rerun every lower-layer counterexample.

## 7. Verification tiers and cost limits

The normal repository command is:

```text
go test -short -timeout 2m ./...
```

| Tier | Purpose and required bounds |
| --- | --- |
| Ordinary | Deterministic unit/component checks using the smallest population and event stream that proves the claim. Must pass the command above. No capacity, soak, live, or long benchmark work. |
| Acceptance | Explicit component or vertical proof commands skipped by `testing.Short()`. Use named tests, compact fixtures, and an explicit command timeout. No single local acceptance command exceeds 15 minutes without a component-specific recorded justification. |
| Capacity | Separate host-specific command using the smallest scale that crosses the claimed boundary—6,000 symbols for the normal V1 reference unless another fixed limit requires more. Record host, Go version, fixture manifest, trial segmentation, and limitations. Never run repeatedly in the ordinary suite. |
| Replay | Compact deterministic logical-time streams; do not simulate a wall-clock session. Run at multiple playback rates only where rate invariance is the claim. Replay can complete the RC while the market is closed. |
| Live validation | Execute only under separate owner authorization. Record actual universe/provider/transport/latency evidence and limitations. It is pending, not a V1 RC blocker. |

All tests and data generators are finite. Channel waits, retries, polling,
loops, and trials have explicit bounds. Performance trials have individual
deadlines in addition to the command timeout. Before expensive work, validate
symbol count, fixture bytes, interval, correction count, expected state-family
coverage, and planned request/record counts. Semantic correctness and capacity
measurements are separate claims.

During correction cycles, run only the failed proof and direct regressions.
Reuse expensive evidence when code, configuration, fixture, and proof premise
are unchanged. Run cross-component, race, capacity, and full component
acceptance once after the affected implementation stabilizes.

## 8. Review cadence

- A completed component contract receives one focused independent review only
  when it introduces or changes a consequential trust, persistence, identity,
  concurrency, ownership, ordering, or cross-component interface boundary.
- An implementation slice receives a narrow independent review only for one of
  those consequential boundaries when its primary proof and construction
  argument do not make the risk straightforward.
- Every completed component receives one final read-only review.
- Correct findings, then request focused re-review of the finding and affected
  boundary; do not repeat a clean broad review.
- After C11, run one final integrated V1 RC review.
- Never request review solely to reconfirm an unchanged primary proof.

A review finding enters the correction loop when it changes a lower-level
premise or exposes a defect. If the preferred reviewer model is unavailable,
record a substitution. If reviewer capacity is temporarily unavailable,
complete all non-review work, retry later, and do not ask the owner to unblock
the process.

## 9. Closed-market and market-hours completion

The private/local V1 RC can be completed while the market is closed using the
accepted aggregate replay path, compact deterministic T/Q fixtures, fake
provider transport, controlled load, loopback API, and local Chrome dashboard.

Market-hours validation remains a separately authorized follow-up. Its absence
must be visible in the release record but cannot make deterministic V1 evidence
fail. The procedure may not access credentials until the owner authorizes that
specific execution.

## 10. Orchestration and Git

Keep at most one write-capable implementation worker active. Independent
reviewers are read-only. A ledger or Git update occurs only while workers and
reviewers are quiescent. The orchestrator stages exact paths and preserves all
unrelated user changes.

Make local commits at coherent planning/correction milestones, accepted
implementation slices, final component acceptance, and distinct vertical
milestones when safe. A progress update or commit is not a reason to end the
goal. Do not push, rebase, amend, rewrite history, delete branches, or use
destructive reset operations. A failed gate is recorded before correction; it
does not justify combining unrelated work. Apply Section 3's containment rule
when safe exact-path committing is unavailable and continue to the RC outcome.

## 11. Private/local V1 RC acceptance ledger

**State:** accepted 2026-08-09 after the production vertical proof, corrected
C5 concurrency boundary, complete verification, and clean focused final-review
re-review.

`P-V1-RC-VERTICAL` is the compact acceptance-only scenario
`TestPV1RCVertical`. One synthetic eligible symbol is advanced through the sole
live scanner engine and projected to the real private checkpoint store. A new
`operations.NewWithCheckpoint` runtime then uses production `Runtime.RunLive`,
`LiveAdapter`, `HydrationWorker`, automatic candidate discovery/install, and
production hydration-purpose selection against deterministic local WebSocket
and HTTPS fixtures. The automatic checkpoint catch-up preserves qualified
aggregate ranking, acquires post-ack Trade/Quote coverage and current Tape
Rate/Spread, seals one immutable C10 sample, serves it through the loopback HTTP
route with exact-origin CORS, and passes those exact response bytes to C11's
production validator/view model.
The final view is current, checkpoint-installed, and contains the server-ranked
`AAA` row with five-second Tape Rate `0.2/s` and Spread `16.7 bps / 2.00¢`.

The initial final review correctly rejected a lower-level harness that manually
installed the checkpoint and injected successful recovery/T/Q facts around the
production scanner composition. It also found that helper disposition waits
used background contexts rather than the proof's 30-second deadline. Replacing
those shortcuts exposed a real C5/C8 integration race: a delivery loop already
blocked in the raw queue before a concurrent T/Q command write captured a nil
status context and rejected the later valid acknowledgement as ambiguous. The
adapter now records the raw-frame sequence observed before the command write
and re-evaluates correlation after a frame is popped; only a strictly later
frame can receive that pending context. This preserves fail-closed treatment of
pre-command statuses while accepting the causally later provider response. A
focused re-review then found the complementary no-response case: if the
provider accepted the write but sent neither an acknowledgement nor another
frame, an already-blocked dequeue could not discover the pending deadline. The
queue now has a command-state recheck signal that changes no frame or queue
accounting. `ChangeTQ` sends it after write accounting is settled, so the
blocked delivery loop installs the existing command deadline and emits a
bounded ambiguous result with zero provider position when no response arrives.
Operational command deadlines use process monotonic time rather than the
injected market/receipt clock, which may be static or deliberately displaced
in deterministic proofs.
`TestPC5CommandDeadlineWakesBlockedDequeue` starts `next` before the command,
sends no provider response under a market clock displaced to 2099, and proves
bounded completion plus reconciled command and frame accounting. All proof
helper admissions and completions now use the common bounded context.

The corrected acceptance command
`go test -v -run '^TestPV1RCVertical$' -timeout 2m ./cmd/scanner -count=1`
passes in 0.27 seconds, ten consecutive ordinary runs pass, and the same proof
under `-race` with a three-minute command timeout passes. The two focused C5
command-correlation tests pass 20 consecutive ordinary runs, and those tests
plus bounded raw-queue accounting pass ten consecutive race runs. The final
20-test C11 model/visual suite,
`go test -short -timeout 2m ./...`, `go vet ./...`, and `git diff --check` are
clean. The ordinary suite skips this cross-language acceptance path. The
production Chrome `P-C11-VISUAL` proof already establishes the distinct
renderer/layout/accessibility/transport and UI-only restart claims against C10
fixtures; it is reused unchanged. Accepted C1-C4 and C6-C10 primary proofs and
the recorded C7/C8 6,000-symbol capacity evidence are reused because their
production boundaries did not change. C5 was reopened narrowly: the two
blocked-dequeue T/Q correlation counterexamples invalidate its prior complete
concurrency claim while preserving its other ten primary proofs. The focused
C5 timeout regression, affected adapter tests, their race executions, the
production vertical proof, and the same integrated reviewer are the allocated
correction and re-acceptance evidence.

The required `gpt-5.6-sol` medium integrated read-only review first rejected
the manual checkpoint/fact shortcuts and unbounded proof waits, then found the
blocked-dequeue correlation and silent-provider deadline counterexamples. After
the production-path, causal-sequence, command-state wake, and process-clock
corrections, the same reviewer found no remaining P1/P2. Component 5 is
re-accepted for the corrected command boundary, Component 11 remains finally
accepted, and the private/local V1 RC is accepted.

Every capability remains routed exactly once through Section 6. This vertical
proof asserts only cross-component identity, status/field preservation,
checkpoint catch-up wiring, T/Q independence/enrichment, loopback API/CORS,
and production UI consumption. It does not establish current provider
population or timing, provider entitlements, live chronology, public-network
security/availability, another browser/mobile layout, trading edge, or
executable expectancy. Market-hours validation remains `pending` under
[`market-hours-validation.md`](market-hours-validation.md); no credential or
provider request was used.
