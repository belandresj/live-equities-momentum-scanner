# Live backend replacement delivery program

**Status:** Owner-approved current delivery authority, 2026-08-23; owner-revised
E2 to exactly one 10-minute deterministic acceptance run on 2026-08-23,
inserted bounded parallel live hydration `LBR-A3` before D1, and accepted the
narrow diagnostic-preservation correction between A3 and D1 on 2026-08-24;
owner-revised on 2026-08-25 to defer E2 as non-gating and permit E3 next under
exact provider authorization.

**Parent:** [`../live-backend-replacement.md`](../live-backend-replacement.md).

**Baseline:** `0d043c1 stabilize live evaluation and local continuity`.

**Implementation branch/worktree:** `codex/live-backend-replacement` in the
dedicated replacement worktree. `main`, `origin/main`, and unrelated work in
the original checkout remain outside this program.

## 1. Outcome

Done means the ordinary private/local fresh-start live scanner runs exclusively
through the replacement architecture and:

- preserves every product meaning routed by the parent contract;
- hydrates and reconciles the complete population through the subscribed live
  tail without fabricated marks or unfinished successful-empty work;
- maintains one compact canonical symbol state and one ordered mutation path;
- updates qualification incrementally, selects exact Day-%/symbol top 20 once
  per causally supported second, and enriches only selected rows;
- keeps aggregate ranking independent from T/Q health;
- publishes the existing API v2 contract to the finished dashboard;
- removes the old live state/evaluator/handoff and superseded feature
  machinery from the supported path;
- passes accepted semantic, recovery, concurrency, and ordinary verification;
- records bounded real-provider stability only when separately owner-authorized
  or owner-executed; and
- records deterministic whole-composition capacity separately only if deferred
  E2 is later reactivated and passes.

This program delivers a private/local scanner, not a public service, trading
signal, replay product, checkpoint restart claim, or provider SLA.

## 2. Authority and execution boundary

After owner approval, authority for this program descends as:

1. [`product-goals.md`](../product/product-goals.md);
2. the [replacement parent architecture](../live-backend-replacement.md);
3. this delivery program;
4. the five focused replacement specifications routed by the parent; and
5. one-slice implementation assignments, proofs, reviews, benchmarks, and
   code.

Existing architecture/component/correction documents remain reusable evidence
for preserved behavior. Where they conflict with the approved replacement
topology or deletion boundary, the parent and this program control. Historical
acceptance does not require preserving an old queue, representation, package,
feature state, replay path, checkpoint shape, test, or benchmark.

Focused contract drafting proceeds first. A focused spec may not begin
implementation until its contract and dependency interfaces are complete and
reviewed as required below.

## 3. Fixed behavior, program choices, and non-scope

### Fixed

- All parent-routed product semantics and accounting identities.
- One `ScannerStateEngine`, canonical symbol state, watermark, ranking path,
  and immutable publication.
- One persistent aggregate socket with selected-row T/Q on the same supported
  provider connection.
- Existing fresh/gap hydration policy and exact live ingress fence.
- One bounded live REST hydration pool accepting `1|2|4|8`, defaulting to 8,
  with workers limited to immutable fact production and one engine-owned
  generation/fence/currentness path.
- Existing API v2 field meanings and finished dashboard behavior.
- The owner-approved T/Q late/duplicate horizon: events more than 30 seconds
  older than committed `T` are ignored with accounting, and duplicate evidence
  is retained for 30 seconds after receipt.
- Checkpoint mode off and replay unclaimed for the supported path.
- No provider credentials or live requests without exact execution approval.

### Program-revisable

The orchestrator may revise focused document layout, internal interfaces,
algorithms, queue slots/batch shape, private state encoding, proof allocation,
fixtures, thresholds stricter than the parent resource ceiling, slice
boundaries, review routing, and deletion mechanics. Revisions must remain
inside fixed behavior, be recorded before dependent work, and invoke the
correction loop rather than an owner stop.

### Non-scope

- New product fields, qualification/ranking changes, or UI redesign.
- Database, microservice, generic bus, plugin system, or public deployment.
- Replay repair, checkpoint compatibility, or alternate restart paths.
- Provider protocol inventions without documentation, fixture, observation,
  predecessor regression, product invariant, or owner decision.
- Broad repository renaming or historical-document rewriting before the live
  replacement is accepted.

## 4. Program structure

The program has one specification gate, five sequential implementation
capabilities, and one integrated acceptance boundary:

| Order | Capability | Focused contract | Slices |
| ---: | --- | --- | --- |
| 0 | Authority, focused contracts, and baseline characterization | Complete replacement set | `LBR-P0`, `LBR-P1` |
| 1 | Canonical state and hydration | `canonical-state-and-hydration.md` | `LBR-A1`, `LBR-A2`, owner-inserted `LBR-A3` before D1 |
| 2 | Evaluation and publication | `evaluation-and-publication.md` | `LBR-B1`, `LBR-B2`, `LBR-B3` |
| 3 | Compact selected-row T/Q | `tq-state.md` | `LBR-C1`, `LBR-C2` |
| 4 | Single-pass live ingress and one handoff | `live-ingress.md` | `LBR-D1`, `LBR-D2` |
| 5 | Cutover, deletion, and integrated acceptance | `integration-removal-and-acceptance.md` | `LBR-E1`, `LBR-E2`, `LBR-E3` |

Keep one active write-capable slice. A later capability cannot begin while the
prior capability has unaccepted behavior on an interface it consumes.

## 5. Specification and baseline gate

### `LBR-P0` — parent/program approval

**Outcome:** Accepted 2026-08-23. The owner approved the architecture outcome,
document map, delivery sequence, 30-second T/Q policy, and target-versus-hard-
acceptance resource policy.

Update the specification map and repository guide in this planning milestone
without changing implementation.

### `LBR-P1` — focused contracts and characterization

**Outcome:** The five focused specs are complete, cross-reviewed, and provide
enough information to generate one-slice assignments without conversation
history.

The specification set must:

- route every `LBR-ARCH-*` requirement, parent-retained semantic, primary
  proof, and implementation slice exactly once;
- define acyclic interfaces and one ownership path;
- state the malformed/late/duplicate/overflow trust policies;
- name what each accepted slice makes removable;
- preserve only evidence required to compare old and replacement behavior;
  and
- avoid implementation chronologies or duplicated acceptance ledgers.

Characterization freezes, without changing production behavior:

- approved aggregate merge/coverage/qualification/ranking/API fixtures;
- the latest durable queue, heartbeat, hydration, cycle, and memory facts;
- one mature-population deterministic manifest;
- current API-v2/UI golden behavior; and
- the commands/configuration required to compare semantic and resource
  results.

Existing expensive evidence is reused when its code/configuration/premise is
unchanged. Missing current CPU/RSS evidence is recorded as unknown rather than
backfilled from an incomparable historical run.

The completed-contract independent cross-review on 2026-08-23 returned
`CLEAN/PASS` with no P1, P2, or P3 finding after the source-aware hydration,
historical/live trust, cross-cutting allocation, retry-ordinal, and generic
stability-acceptance corrections. This completes the review portion of
`LBR-P1`. The owner accepted the reviewed five-spec set on 2026-08-23. Baseline
characterization remains before `LBR-P1` can authorize implementation.

## 6. Capability A — canonical state and hydration

### `LBR-A1` — compact canonical aggregate state

Replace the current duplicated/pointer-heavy canonical aggregate backing with
the parent-defined sealed prefix and sparse correction tail. Install the
current-product sufficient state needed by the existing projection while
leaving any still-required legacy derived projection compatibility explicitly
temporary until `LBR-B3`; it cannot remain a canonical aggregate fallback.

Primary proof distinguishes insert, duplicate, revision, withdrawal, conflict,
ordinary out-of-order live delivery, 16-minute horizon boundaries, no-print,
coverage uncertainty, and REST/live precedence from a state that merely
matches the latest price. It compares complete semantic projections with the
approved baseline, not private struct equality.

The slice must leave one production canonical state. Test-only differential
oracles may read the same immutable fixtures but cannot become a runtime
fallback or second owner.

### `LBR-A2` — hydration and recovery installation

Route fresh and gap hydration through the compact state while retaining one
terminal outcome per request, the subscribed live tail, generation fencing,
and ingress reconciliation.

Primary proof covers value/empty/failure/cancellation, replacement epoch during
hydration, post-live gap recovery, live-over-REST precedence, and currentness
only after the exact fence. Acceptance makes old aggregate state and
checkpoint/replay-driven live installation removable from the supported path.

### `LBR-A3` — bounded parallel live hydration restoration

Restore the previously accepted ordinary-live `1|2|4|8` REST worker surface
with default 8. Workers perform only blocking acquisition and normalization;
they return immutable bounded chunks/terminals to A2's one engine-owned
generation ledger. Preserve the subscribed live tail, exact REST/live merge,
one terminal per request, fence-after-all-terminal rule, currentness, and all
accepted B/C behavior.

Primary proof runs 1, 2, 4, and 8 workers with deliberately unequal request
latency while live aggregates continue. It proves bounded active/resident work,
out-of-order token correctness, no dropped or duplicated terminal, cancellation
and joined shutdown, no sustained live-queue growth, one exact fence, and
semantic/publication equivalence. Provider speedup remains a separately
authorized observation, not deterministic acceptance.

The 2026-08-24 observation at `5abaf2d` is motivation rather than acceptance:
one-worker hydration reached 1,761/5,566 symbols (31.6%) in approximately five
minutes while 113,037 live aggregate deliveries progressed, queue high-water
remained 165/32,768, accounting stayed valid, and the connection did not
restart. The historical failures motivating the backend replacement occurred
after successful multiworker hydration, so serial acquisition is not a valid
containment for that later failure domain.

Capability A final review focuses on canonical uniqueness, correction
equivalence, hydration false-success, memory bounds, and absence of writable
aliases.

## 7. Capability B — evaluation and publication

### `LBR-B1` — incremental qualification and selection

Replace per-cycle qualification cloning with symbol-local incremental proof
maintenance. At each supported second, scan only compact selection/accounting
state and retain exact qualified Day-%/symbol top 20.

Primary proof covers the approved gate, mutable proof revocation, strict
finalization, first later print, fewer-than-20, ties, degraded/unavailable
population cases, and T/Q/display-field independence.

### `LBR-B2` — selected-row enrichment and combined publication

Compute/read final aggregate display fields for selected rows from the same
preselection canonical state, join selected-row T/Q, validate accounting, and
replace one immutable API-v2-compatible snapshot.

Primary proof keeps a symbol unselected for more than 330 seconds, selects it
solely by Day %, and requires immediate exact Volume, From Open, Day Range,
Activity 30s, and Move 30s while Tape and Spread alone warm. One-publication
identity and field-local failures remain exact.

### `LBR-B3` — remove superseded evaluation

Delete active HOD drawdown, rolling 30/60-minute range, old Activity composite,
redundant aggregate-derived state, and their implementation-only proofs.
Mechanically retained replay/checkpoint compilation may not keep those states
inside the live owner. Superseded T/Q/Tape state is owned by Capability C.

Primary proof is the product/API exclusion plus resource differential: removal
cannot change qualification, Day-%/symbol ordering, final fields, availability,
or API v2, and the mature evaluation must meet its allocated latency/allocation
budget.

Capability B final review focuses on exact ordering, selection-independent
history, accounting, one publication, and demonstrated removal rather than
dead fallback code.

## 8. Capability C — compact selected-row T/Q

### `LBR-C1` — bounded Tape/Spread state

Replace broad string-heavy retention with the smallest state proving Tape 5s,
Spread, exact duplicates, unequal repeats, supported ordinary out-of-order
delivery, quote age, and coverage gaps. Remove separate one-second Tape-burst
state and output assumptions while retaining only evidence needed by Tape 5s.

The focused spec must implement the approved 30-second event-time and receipt-
time boundaries without silently extending them. A bound makes T/Q unavailable;
it cannot suppress aggregate state.

### `LBR-C2` — cadence membership and trust transitions

Batch desired top-20 membership changes at the existing combined cadence while
keeping first data confirmation, coverage closure, command/control error,
pressure degradation, and restoration immediate where trust changes.

Primary proof composes rapid rank churn, subscribe/unsubscribe writes,
current-epoch data confirmation, quiet coverage, pressure shedding to zero,
restoration, and unchanged aggregate publication semantics.

Capability C final review focuses on bounded retention, provider-status
assumptions, coverage honesty, and aggregate independence.

## 9. Capability D — single-pass live ingress

### `LBR-D1` — single-pass Massive decoding

Decode one frame once into one bounded immutable batch while preserving epoch,
frame/array order, receipt time, recognized-member duplicate rejection, event
normalization, additive-field tolerance, and the focused malformed-event
policy.

Primary proof compares every accepted aggregate/T/Q/control/drop/ambiguity
result and accounting consequence with the approved fixture corpus, then
measures allocations and throughput separately from engine work.

### `LBR-D2` — one ingress handoff

Replace the raw-frame queue, intermediate adapter deliveries, and second engine
FIFO with the one decoded-batch FIFO and causal fence marker owned by the
parent architecture. Retain one-at-a-time socket retirement, inbound-aware
heartbeat, retry pacing/exhaustion, resubscription, and exact recovery.

Primary proof covers mixed frames, backpressure, safe T/Q shedding, aggregate
overflow terminal/recovery, fence ordering, heartbeat progress/no-progress,
failed handshakes, one immediate process-start dial plus five exactly paced
recovery attempts, exhaustion before recovery attempt 6, and zero silent
aggregate/control loss.

Capability D final review focuses on ordering linearization, socket/worker
joins, one queue, overflow false-success, and preserved connection semantics.

## 10. Capability E — cutover, deletion, and acceptance

### `LBR-E1` — exclusive production path and cleanup

Wire the ordinary scanner exclusively to the accepted replacement path. Remove
old live engine/queue/envelope/state implementations, compatibility flags,
superseded feature tests, and active specification routing. Apply the owner's
replay/checkpoint source/tool disposition without adding a gate to live
operation.

API v2, dashboard, launcher, reference, hydration, and operator behavior remain
compatible unless an accepted focused contract explicitly revises an internal
interface.

### `LBR-E2` — deferred deterministic resource and stability characterization

If separately reactivated, run exactly one mature 5,694-symbol deterministic 10-minute mixed-feed
acceptance composition with selected-row T/Q and one-second dashboard polling. Use a fixed
manifest at exactly 300 frames/s: 600 timed seconds, 180,000 frames, 3,416,400
base aggregates, 600 resource samples, 600 dashboard polls, and 11 ranking/
accounting checkpoints. Measure the parent CPU, heap, RSS,
allocation, goroutine, queue, cycle, capture, watermark, loss, and accounting
targets.

There is no optional, fallback, diagnostic, host-coexistence, or non-gating
repeat composition. The exact 10-minute run remains within the ordinary
15-minute local-command ceiling. Validate the manifest first, use an explicit
15-minute command timeout, keep stop conditions active, and do not shorten or
extend the timed interval.

Every numeric target is reported from that one run. If hard acceptance passes,
a numeric miss is recorded as a deviation and cannot authorize an additional
timed trial. Measurement invalidity or a hard failure reopens the lowest owning
ticket; a corrected future E2 activation again owns exactly one 10-minute run.
The orchestrator must not append optimization, coexistence, or diagnostic runs.
E2 is retained as optional future capacity evidence and is not a prerequisite
for E3 or `live_stability_confirmed`.

### `LBR-E3` — separately authorized market-hours confirmation

After accepted A–D semantics and E1 exclusive cutover, one exact-date
owner-authorized or owner-run observation confirms the final provider wiring, hydration, connection
continuity/recovery, ranking, T/Q, API, dashboard, and resource trend.

This program does not itself authorize credential access or a provider request.
E3 may proceed while E2 is deferred, but it cannot claim the synthetic
300-frames/s capacity result. Without a clean E3 observation and final
integrated program review, the implementation is not
`live_stability_confirmed`.

## 11. Proof and verification policy

Each focused requirement has one primary proof that states claim,
counterexample, observable distinction, participating path, and limitation.
Use the smallest proof that distinguishes correctness; do not rebuild the old
test volume around private structures.

Per slice, run in this order:

1. narrow primary proof and direct regressions;
2. affected short package tests;
3. affected race tests when ownership, queues, concurrency, publication, or
   immutable reads change;
4. focused vet and diff checking;
5. repository ordinary verification at the slice gate, not after every edit;
6. selected capacity/resource proof only at its allocated capability or
   integrated gate; and
7. independent review at the cadence below.

Ordinary verification remains:

```text
go test -count=1 -short -timeout 2m ./...
```

Every long command has an explicit timeout and validated manifest. No local
trial exceeds 15 minutes without a recorded reason. Live work remains separate
and explicitly authorized.

## 12. Agent orchestration and review

The orchestrator owns the program ledger, assignment generation, exact-path
staging, local commits, correction routing, and final conformance review.

For an E2 resource-target miss, the orchestrator records the deviation from the
single 10-minute run. It must reject repeated profiling, speculative tuning,
scope expansion, or another timed trial once hard acceptance passes. A target
deviation is evidence for the ledger, not an unfinished implementation state.

For implementation:

```text
orchestrator
  -> one write-capable implementer for one current slice
  -> narrow proofs and handoff
  -> independent read-only reviewer when required
  -> correction by the same implementer when practical
  -> focused re-review
  -> acceptance record and next slice
```

Before assigning a ticket, the orchestrator performs one compact read-only
counterexample audit across the current ticket, accepted dependency handoff,
and every production consumer whose defaults or configuration can construct
the changed interface. The audit must cover, where applicable:

- exact equality and one-tick boundaries around correction, expiry, cadence,
  retry, and retention horizons;
- a duration longer than every owned retention horizon while committed time is
  stalled, followed by loss/recovery;
- default wrapper/CLI/launcher/configuration values through the final runtime
  validator;
- empty, malformed, duplicate, conflict, overflow, cancellation, replacement,
  and terminal evidence that could falsely appear complete/current; and
- old code or unsupported tooling that still compiles but must not remain a
  production constructor or fallback.

This audit changes no authority and does not create a ceremonial review. If it
finds a consequential ownership, ordering, concurrency, persistence, or
false-success ambiguity that construction and the allocated primary proof do
not already resolve, use the program's existing narrow-review route before
coding rather than discovering the boundary only at capability review.

At most one write-capable worker is active. Reviewers are read-only and may not
edit, stage, commit, approve, expand scope, or preserve an implementation over
controlling authority. Independent semantic, concurrency, and performance
inspection may run in parallel only when it is read-only and the orchestrator
does not mutate the ledger/Git state until all agents are quiescent.

Review cadence:

- one completed-contract cross-review for the five focused specs;
- one final read-only review per capability;
- a narrow slice review only when a consequential ownership, ordering,
  concurrency, external trust, or false-success boundary cannot be established
  by construction and primary proof alone; and
- one final integrated program review after `LBR-E3`, incorporating focused
  live-evidence inspection, plus an independent capacity-evidence review only
  if deferred `LBR-E2` is later reactivated.

One implementation assignment covers one slice and contains the ten items
required by `AGENTS.md`. Tickets are generated from the approved focused specs;
they are not new architecture documents or authorities.

During implementation and correction, run only the narrow distinguishing proof
and direct affected tests until the bytes are stable. Run the affected race
tier, focused vet/diff gate, and repository ordinary command once on the final
candidate bytes. A later edit that changes the proved boundary invalidates only
the affected final-byte gates; it does not justify repeatedly rerunning
unchanged expensive evidence.

## 13. Correction and acceptance

A failed proof, review, benchmark, or integration run records evidence and
reopens the lowest unsuitable spec/slice. Preserve unaffected evidence, revise
the smallest artifact, run the narrow distinguishing proof, and continue in
program order.

A numeric resource-target miss alone does not reopen a capability. Accept the
single stable hard-conforming 10-minute result with its deviation. Measurement
invalidity or a hard failure reopens the lowest owning ticket; any later E2
activation again permits one exact 10-minute run, not an appended repeat.

At each accepted slice, record only:

- coherent behavior now available and behavior deferred;
- exact requirements and primary proofs passed;
- changed ownership/interfaces and old code now removable;
- counterexample, limitation, and construction guarantee;
- review result; and
- whether the next slice remains valid.

Do not append command transcripts or chronological debugging narratives. Link
bounded artifacts or commits when provenance matters.

Capability acceptance requires all allocated slices, proportionate
verification, one final read-only review, and no unresolved parent/product
conflict. Later evidence may reopen it before integrated completion.

### Accepted slice records

#### `LBR-A1` — accepted 2026-08-23

- **Coherent behavior:** one engine-owned, binding-indexed canonical aggregate
  state now has a sealed sufficient prefix, a strict inclusive 961-identity
  correction tail, explicit present/no-print/unknown/invalid/conflict evidence,
  symbol-local canonical revisions/affected-proof notifications, and detached
  `SelectionStateView`/`SelectedAggregateView` values. Current-token old
  historical fills fold directly without hydration pinning a full-session raw
  tail. Exact REST/live comparison values exist only inside the exact active
  hydration request, are capped at 961, force unknown coverage on overflow,
  and purge on row consumption, terminal, fence, cancellation, loss, or
  generation replacement.
- **Proof:** `P-LBR-A1-CANONICAL` passes insert/equal duplicate/revision,
  historical withdrawal/conflict, ordinary out-of-order live delivery,
  REST/live permutations, inside/far-old historical fill, exact 16-minute
  equality/+1-ns boundaries, no-print/invalid/unknown coverage, strict tail and
  temporary reconciliation bounds, immutable views, and production hydration
  accounting/purge. The dangerous horizon-blind/latest-only counterexamples
  cannot acquire too-late live authority, reopen sealed live state, preserve a
  withdrawn prefix effect, restore unsupported first-open/extrema, hide a later
  trusted mark, miss invalid-evidence notification, or retain a comparison
  shadow after its request.
- **Ownership/removal:** canonical merge reads only the one aggregate state.
  Existing `priceRange`, `activity`, `mvpMeasurements`, and qualification
  structs remain one-way compatibility projections, never merge inputs;
  Capability B removes them in its allocated slices. The old hydration
  raw-tail pin is removed. No writable view alias or production fallback
  exists.
- **Verification/review:** focused proof, engine short/race, focused vet, diff
  hygiene, and repository short verification pass. Narrow independent review
  first reopened too-late authority, prefix trust, sealed-live comparison, and
  invalid-notification false-success cases; both focused correction rounds
  passed re-review with no remaining P1/P2 finding.
- **Limitation/next gate:** this proves canonical merge/state behavior, not A2
  generation/fence completion, mature-population resource scale, or provider
  correctness. `LBR-A2` remains valid and is next; it must consume this compact
  state and request-scoped reconciliation seam without restoring raw-tail
  pinning.

#### `LBR-A2` — accepted after final-review correction 2026-08-23

- **Coherent behavior:** fresh `[S,R)` and exact post-live gap hydration now
  use the A1 compact state, one engine generation/request ledger, one supported
  REST worker, exact terminal/row accounting, and one ordered ingress fence.
  The supported `RunLive` composition rejects checkpoint storage and worker
  counts other than one; the Massive live-plan seam accepts only fresh/gap
  work. Generic worker construction remains reachable only by separated
  historical tooling pending the E1 source decision.
- **Proof:** `P-LBR-A2-HYDRATION` passes across the real engine, Massive worker
  seam, and operations composition. It covers before-session wait, in-session
  acknowledgement, at/after-session terminal behavior, invalid-binding
  atomicity, bounded initialization exhaustion, value/empty/failure/cancel/
  fenced terminals, malformed rows, generation/epoch replacement, preserved
  accepted facts, current-token fencing, REST/live precedence, terminal-before-
  fence ordering, a marker behind already-read work, retry-reset/currentness
  only after the fence-target evaluator applies, exact gap recovery, and later
  ordinary live advancement. Work and row identities reconcile before fence
  trust; successful empty proves only its exact interval.
- **Ownership/removal:** blocking REST work returns immutable bounded facts and
  owns no symbol/currentness state. The engine remains the sole generation,
  coverage, fence, retry-reset, and lifecycle owner. Ordinary-live multiworker
  configuration, checkpoint-backed `LiveComponents`, checkpoint-catchup in the
  live worker seam, and old hydration proof/backing assumptions are now
  removable; physical replay/checkpoint disposition remains E1.
- **Verification/review:** named proof, affected engine/Massive/operations
  short and race suites, focused vet, diff hygiene, and repository short
  verification pass. An initial generic worker-count change broke retained
  replay compilation; the corrected layering enforces one worker only in the
  supported live composition and the distinguishing replay regression passes.
- **Limitation/next gate:** deterministic evidence does not establish provider
  availability, connection retry chronology, or resource throughput. Both A
  slices are ready for the required Capability A final read-only review;
  Capability B remains inactive until that review is clean and Capability A is
  finally accepted.
- **Final-review correction:** the seconds-long recovery proof missed a
  stalled watermark whose accepted post-`T` live rows compact before loss;
  source-blind registration then rejected the required exact gap. The review
  also found that the retained launcher still emitted its historical default
  of eight workers into a live composition that now requires one. Source-aware
  planning now admits compacted presence only with sealed-live support; a
  1,022-row/>16-minute stalled-`T` recovery trace passes through exact gap,
  conservative REST/live conflict accounting, ordered fence/evaluation, and
  later ordinary advancement without raw repinning or permanent comparison
  history. Wrapper, launcher, scanner, operations, README, and runbook now
  accept/emit exactly one worker. Focused re-review returned `CLEAN/PASS` with
  no remaining P1/P2/P3 finding.

#### Capability A — finally accepted 2026-08-23

Both A1 and A2 primary proofs, affected short/race/vet/diff gates, repository
short verification, correction re-reviews, and the final capability review are
clean. The accepted handoff is one compact canonical aggregate owner plus one
worker fresh/gap hydration, exact generation/work/row/fence accounting, bounded
request-local reconciliation, immutable downstream views, and source-aware
long-stall recovery. It does not claim provider availability, live retry
chronology, mature resource targets, replay/checkpoint compatibility, or
trading edge. `LBR-B1` is the next permitted write-capable slice.

Post-acceptance specification clarification records the already-proven behavior
without changing code or acceptance: the canonical/hydration contract now
states the conservative REST disposition and recovery legality for live values
that compacted before an exact gap request; the then-current integration
contract recorded the one-worker invariant through wrapper, launcher, scanner,
operations, and runbook. The program orchestration section now requires a pre-
assignment cross-boundary counterexample audit and final-byte-only expensive
verification. The following owner revision supersedes only that delivery
restriction.

The owner revision on 2026-08-24 preserves every accepted A1/A2 semantic and
proof result but reopens Capability A only for the one-worker delivery
restriction. `LBR-A3` restores bounded parallel acquisition before D1; accepted
B/C behavior remains valid and does not reopen.

#### `LBR-B1` — accepted 2026-08-23

- **Coherent behavior:** the sole engine owner now advances bounded symbol-local
  qualification proof state before selection and scans fixed population scalars
  once per unique supported candidate/revision. Exact Day-%/symbol top 20 uses
  one candidate-relative mark strictly before `T`; ordinary `T` to `T+1`
  advancement is constant work per symbol, while initial/non-unit recovery and
  accepted withdrawal use one bounded symbol-local repair across at most the
  961-record correction tail. Timer and real connection-control work share one
  positive engine-assigned nonreused system sequence.
- **Proof:** `P-LBR-B1-SELECTION` passes exact qualification gates, correction
  equality and strict finalization, first-later-print, quiet timers, candidate-
  middle advancement, revisions, two consecutive withdrawals, fewer-than-20,
  exact ties, incomplete/invalid population, T/Q/display-field independence,
  unchanged-prefix coalescing, and control/timer/control sequencing. The real
  A2 trace stalls committed `T` beyond 16 minutes, compacts rows on both sides,
  loses and exactly recovers coverage, then proves the post-fence old-`T`
  evaluator row/order/accounting excludes marks beginning at or after `T`.
- **Ownership/removal:** live selection reads fixed mark/qualification scalars
  and retains at most top 20; it neither clones qualification maps nor scans a
  canonical tail. Rare correction/jump repair remains owner-local and bounded.
  Replay-only projection cloning/tail lookup is explicitly mode-isolated pending
  E1 disposition. Old live qualification cloning, full-history selection scans,
  and duplicate same-prefix triggers are now removable.
- **Verification/review:** the final candidate passes the composed B1/A2 proof,
  affected engine short and race suites, focused vet, diff hygiene, and ordinary
  repository short verification. Pre-assignment review corrected the missing
  as-of-`T` recovery discriminator. Implementation review then reopened the
  test-only control proof, candidate-relative mark maintenance, repeated
  withdrawal repair, live Day-% mark coherence, and composed recovery proof;
  focused corrections passed final re-review `CLEAN/PASS`.
- **Limitation/next gate:** B1 does not prove selected-row enrichment, immutable
  combined publication, T/Q behavior, provider traffic, or mature resources.
  Temporary full-population display adapters remain until B2/B3. `LBR-B2`
  remains valid and is next but inactive pending its pre-assignment audit.

#### `LBR-B2` — accepted 2026-08-23

- **Coherent behavior:** one revision- and `T`-bound enrichment pass computes
  Volume, From Open, Day Range, Activity 30s, and Move 30s for at most the 20
  retained B1 rows. Full-population maintenance derives only fixed field
  status/reason scalars; supported live display formulas do not run for
  discarded symbols. One validated evaluation plus immutable T/Q projection is
  sealed under the owner and atomically replaces the existing publication cell;
  Runtime/API capture performs one load with cached diagnostics.
- **Proof:** `P-LBR-B2-PUBLICATION` composes one-engine 331-second unselected
  history followed by Day-%-only entry with immediate exact aggregate fields and
  T/Q warming; more than 16 minutes of future-to-`T` compaction, epoch loss,
  exact-gap recovery, and old-`T` field exclusion; correction/revision fencing;
  genuine zero/no-print; and malformed, incomplete/failed, successful-empty
  before/after exact fence, canceled, late, and replaced evidence through
  Runtime capture and API mapping. Concurrent enriched captures, blocked writer,
  and slow/canceled HTTP preserve one publication identity and row bytes.
- **Trust/cadence:** selected field degradation, qualification revocation, and
  applicable structural coverage/rankability loss publish coherent same-`T`
  closure immediately. Value-only changes coalesce. Rare full trust restages
  have an explicit source/count/timing identity; unchanged following timer/live
  fence work deduplicates, while recovery ingress and ended lifecycle cannot.
  Right-open invalid evidence at `T`/`T+1` is neutral until its later boundary
  and remains bounded to the 961-identity correction horizon.
- **Ownership/removal:** a candidate-bound selected view zeros and marks prefix
  scalars unusable when its folded prefix is ahead of `T`. Live values are
  selected-only, stale revision/mark/accounting fails closed, and T/Q cannot
  alter aggregate rank/watermark/readiness. Full-universe display evaluation,
  separately sampled engine/TQ joins, mutable publication members, and lock-
  coupled/per-request-expensive capture are now removable.
- **Verification/review:** composed primary proof, affected short/race packages,
  focused vet, API-v2 golden, diff hygiene, and ordinary repository verification
  pass on final bytes. Focused review reopened future-prefix seam safety,
  selected-only formula enforcement, same-`T` qualification/coverage/field
  closure, correction-cycle identity, right-open structural evidence, recovery/
  lifecycle dedupe, and literal publisher/capture composition; all focused
  corrections passed final re-review `CLEAN/PASS`.
- **Limitation/next gate:** B2 does not establish final T/Q formulas, provider
  traffic, browser presentation beyond the unchanged API-v2 golden, or mature
  resources. Legacy HOD/rolling/old Activity state/formulas and explicit
  replay/checkpoint/test-tool seams remain for B3/E1 removal. `LBR-B3` remains
  valid and is next but inactive pending its pre-assignment audit.

#### `LBR-B3` — accepted 2026-08-23

- **Coherent behavior/removal:** the supported live engine retains only B1
  qualification/ranking and B2 selected enrichment for Volume, From Open, Day
  Range, Activity 30s, and Move 30s. HOD drawdown, rolling 30/60-minute ranges,
  the old Activity owner/formulas, qualification cloning, replay tail-scanned
  selection, full-population formula fallback, and temporary selected-prefix
  adapters are deleted. T/Q, Tape 5s, Spread, and separate one-second Tape-burst
  state remain unchanged for C1.
- **Tool separation:** replay/checkpoint commands still compile without a
  correctness claim or E1 disposition. Legacy checkpoint Activity, Highs,
  Lows, and RollingFloor payloads are independently rejected by bounded
  validation, direct install, and admission without changing binding,
  evaluation, installed-checkpoint state, or publication. Unsupported tooling
  cannot reconstruct a removed live owner/evaluator/fallback.
- **Proof:** `P-LBR-B3-REMOVAL` parses production and test sources, traverses
  named live/tool import graphs bidirectionally, closes engine/config/symbol
  ownership shapes and evaluation/publication entrypoints, verifies live call
  ordering, and rejects neutral renamed/transitive/test-only selector, owner,
  fallback, temporary-adapter, and direct-publication counterexamples. The
  complete B1/B2/current-field/API-v2 corpus remains exact.
- **Resource evidence:** the then-current pre-E2-duration-revision manifest SHA
  `3a70dda5bc040d1a77fbe3e114d788d3fc04c93fbb51ff2e5eec646f4c10e6e3`
  validates 5,694 symbols and 5,470,012 represented rows. Initial measurement
  was 8.655 ms and 15,484,824 allocated bytes. The single permitted profile/
  correction/rerun removed candidate boxing and corrected diagnostic coverage;
  the accepted rerun is 7.578 ms and 5,119,312 bytes (4.882 MiB/cycle), versus
  the frozen 399.871 ms mean/601.020 ms maximum baseline with baseline
  allocations unknown.
- **Verification/review:** exclusion/current-product/API proofs, affected short
  and race packages, focused vet, diff hygiene, and ordinary repository
  verification pass. Focused review reopened tooling-root/dependency proof,
  production test wrappers, structural renamed-fallback detection, and
  independent checkpoint-field rejection; all corrections passed final
  re-review `CLEAN/PASS`.
- **Limitation/next gate:** replay/checkpoint correctness remains intentionally
  unvalidated and its source disposition remains E1 owner-gated. This is not E2
  whole-process stability or provider evidence. Capability B now requires its
  final read-only review; C1 remains inactive.

#### Capability B — finally accepted 2026-08-23

All B1/B2/B3 primary proofs, affected short/race/vet/diff/API gates, ordinary
repository verification, focused correction re-reviews, and the required final
read-only capability review are clean. The accepted interface is one exact
candidate-relative qualification/ranking pass, selected-only five-field
enrichment for at most 20 rows, one immutable evaluation/TQ publication and
one-load capture, with old HOD/rolling/Activity/full-population fallback state
deleted. The final review returned `CLEAN/PASS` with no unresolved product,
architecture, ownership, ordering, trust, accounting, publication, API, or
deletion conflict.

Capability B does not establish final T/Q formulas/membership, ingress
replacement, whole-process E2 stability, replay/checkpoint support, provider
availability, live stability, or trading expectancy. `LBR-C1` is the next
permitted ticket and remains inactive for a fresh task.

#### `LBR-C1` — accepted 2026-08-23

- **Coherent behavior:** the sole engine owner now retains only five-second
  Tape contribution/lifecycle evidence, a receipt-bounded exact duplicate
  ledger, and latest/latest-valid O(1) quote state for selected symbols.
  Post-write trade/quote channels confirm independently only from complete
  frames strictly after `B`; one-second Tape state and projection fields are
  deleted. T/Q mutations and containment remain unable to change aggregate
  canonical state, qualification, ranking, watermark, or readiness.
- **Bounds and trust:** event time exactly `T-30s` is accepted and one
  nanosecond older is counted/rejected. Duplicate evidence remains through
  first receipt+30s equality and is evicted only later by monotonic engine
  cleanup. Production limits are 10,000/100,000 retained contributions,
  25,000/400,000 duplicate identities, two quotes per desired symbol, a 4 MiB
  symbol charge, and a 64 MiB global charge. Symbol bounds close only that
  symbol; global bounds enter aggregate-only intent. Unequal repeats, quote
  quality loss, gaps, rank removal, and epoch loss close affected trust
  immediately.
- **Proof:** `P-LBR-C1-TQ-STATE` passes strict whole-frame `B`, independent
  confirmation, Tape warming/current/covered-zero, locked/one-sided/crossed/
  stale Spread, condition-qualified exact/unequal/lifecycle evidence,
  conditional TRF identity, both 30-second ties, a committed-time stall beyond
  both owned horizons, gap/removal/replacement, O(1) quotes, every count/byte
  containment branch, immediate trust publication, and byte-for-byte aggregate
  canonical/evaluation equivalence. The dangerous absent-TRF counterexample
  normalizes irrelevant TRF IDs while present TRF IDs remain distinguishing.
- **Verification/review:** primary/direct proofs, affected engine/API/
  operations/scanner short and race suites, focused vet, diff hygiene, and
  exact ordinary repository verification pass on final bytes. No narrow review
  was triggered; the compact-identity ambiguity found by orchestrator diff
  inspection was corrected by the same implementer and the full allocated
  final-byte gates were rerun.
- **Limitation/next gate:** deterministic evidence does not establish provider
  acceptance or socket writes, C2 cadence/churn/pressure policy, final ingress
  queue behavior, whole-process heap/RSS, or live-provider chronology. `LBR-C2`
  remains coherent and is next but inactive pending its pre-assignment audit.

#### `LBR-C2` — accepted then reopened by final review 2026-08-23

- **Coherent behavior:** the engine snapshots additions only at the accepted
  one-second combined cadence, issues one sorted full batch for a fresh epoch,
  closes rank removals locally and immediately, batches all wire removals
  before one additions batch, and keeps one write in flight. Pressure recovery
  waits for cadence and restores at most one highest-ranked symbol with a fresh
  generation per cadence. Generic/late success status is informational; local
  write or bounded provider error closes additions and T/Q trust for the epoch
  without changing aggregate readiness or entering pressure by invention.
- **Command/pressure ownership:** successful subscribe `B` is the greatest
  admitted complete raw-frame sequence at write return; rejected read attempts,
  array positions, and status counts cannot advance it. Engine pressure input
  now contains only waiting slots/bytes and capacities, oldest waiting age,
  cumulative slot/byte loss, guarded aggregate lag/T/Q-work, and closed
  accounting. Active-frame age, heap, goroutines, generic latency, and delivery
  attribution remain diagnostics only. The old T/Q acknowledgement/status-
  count/deadline/quarantine state, tests, counters, mapper fallback, and
  per-symbol churn sequencing are deleted; handshake-only status correlation
  remains isolated to connection/authentication/aggregate subscription.
- **Proof:** `P-LBR-C2-TQ-MEMBERSHIP` passes fresh sorted batching, rapid churn,
  immediate local closure, unsubscribe-before-subscribe continuation, write
  failure, generic/late status, bounded provider error, quiet post-write data
  confirmation, aggregate-only membership to zero, exact 10%/25% entry and
  strict-below-1% plus 749/750-ms recovery ties, guarded watermark and capacity/
  accounting loss, five-sample cadence-gated restoration, reconnect fencing,
  one-in-flight accounting, and 500-ms mixed-frame T/Q shedding with aggregate/
  control classification preserved. The admitted-frame proof forces raw read
  attempts above `B`, excludes frame `B`, and confirms independently at `B+1`.
- **Verification/review:** primary/direct engine, Massive, operations, and API
  proofs; affected short suites; engine/Massive/operations race; focused vet;
  source exclusion; diff hygiene; API golden/HTTP; and exact ordinary repository
  verification pass on final executable bytes. A final comment-only removal of
  superseded status tests reran affected Massive/vet/diff and ordinary gates;
  unchanged race/direct evidence was preserved. No narrow review was triggered
  because command, pressure, and mixed-frame linearization were resolved by
  construction and the allocated proof.
- **Limitation/next gate:** deterministic evidence does not prove provider
  acceptance, credentialed behavior, final D2 queue replacement, whole-process
  resources, or live-market chronology. Both C slices are ready for the
  required Capability C final read-only review; `LBR-D1` remains inactive.
- **Final-review reopening:** the required Capability C review found that the
  production live loop could deliver raw frame `B+1` concurrently after the
  successful write captured boundary `B` but before the command result reached
  the engine. That frame was then fenced as unrequested and could leave an
  otherwise valid quiet channel unconfirmed. This invalidates only the claimed
  command-result/data linearization and its split proof; C1, C2 cadence,
  pressure, accounting, status removal, aggregate independence, and all
  unrelated final-byte evidence remain accepted. Reacceptance requires the
  same implementer to serialize write, exact `B` capture, and command-result
  engine completion against raw dequeue while still allowing reads to queue,
  plus a deterministic `B`/`B+1` overtaking proof and affected final-byte
  verification. `LBR-D1` remains inactive.
- **Correction and reacceptance:** the transport now holds the same delivery
  lock across the socket write, exact greatest-admitted-frame `B` capture, and
  command-result engine completion that raw dequeue already uses. The read
  worker remains independent and may continue filling the bounded queue, but
  no `B+1` frame can reach the engine before the result establishing its
  generation and boundary. The distinguishing proof pauses after `B` capture,
  queues `B+1`, starts concurrent raw delivery, and observes command result,
  fenced `B`, then applied `B+1`; `B+1` independently opens only Trade
  coverage, while aggregate/control state and adapter/queue accounting remain
  unchanged. The narrow proof, affected Massive/operations short and race
  suites, focused vet, diff hygiene, and exact ordinary repository suite pass
  on final bytes. The reopened C2 claim is reaccepted subject to focused
  re-review by the same final reviewer; all other preserved evidence remains
  valid and `LBR-D1` remains inactive.
- **Focused re-review reopening:** the production correction passed focused
  inspection, including lock ownership, reader independence, sole runtime
  path, cancellation, failure, and lock-order behavior. The adversarial test
  did not, however, prove that its raw-delivery goroutine had actually started
  contending before command release; a late-scheduled goroutine could let the
  pre-correction split path pass. Only the deterministic dangerous-
  counterexample claim is reopened. Add an explicit raw-call rendezvous and
  hook-based proof that raw engine delivery cannot enter while the command is
  paused, then retain every ordering, coverage, aggregate/control, and
  accounting assertion. Production code and all unrelated evidence remain
  accepted; `LBR-D1` remains inactive.
- **Deterministic-proof correction and reacceptance:** the raw-delivery
  goroutine now rendezvous-signals immediately before calling the production
  dequeue path. With the command paused after exact `B` capture and before
  engine completion, its Trade engine-entry hook remains unreachable for a
  bounded contention interval; after command release, the test requires that
  hook and raw completion. The retained assertions establish command result,
  fenced `B`, then applied `B+1`, independent Trade-only confirmation,
  unchanged aggregate/control state, and reconciled adapter/queue accounting.
  Production bytes are unchanged. The narrow proof, Massive short/race,
  focused vet, diff hygiene, and exact ordinary repository suite pass; the
  proof claim is reaccepted subject to final focused re-review. `LBR-D1`
  remains inactive.

#### Capability C — finally accepted 2026-08-23

- **Accepted capability:** C1 and C2 together provide bounded selected-row
  Tape/Spread state, exact event/receipt horizons, honest independently
  confirmed Trade/Quote coverage, immediate trust closure, one-second combined
  membership publication, removal-before-addition batching, cadence-gated
  pressure recovery, and exact command/data ordering. Aggregate ranking and
  readiness remain byte-for-byte independent of T/Q availability and health.
- **Final review:** the required read-only Capability C review first reopened
  the production `B+1` overtaking seam, then rejected a nondeterministic first
  version of its counterexample. The same implementer corrected each narrow
  claim sequentially. Final focused re-review returned `CLEAN/PASS` with no
  model substitution: the corrected proof passed 20 repetitions and focused
  race detection, while production remained byte-identical to the approved
  linearization correction.
- **Evidence and limitations:** all C1/C2 primary proofs, affected short/race
  suites, focused vet, source exclusion, diff hygiene, API golden/HTTP checks,
  and exact ordinary repository verification pass on their recorded final
  bytes. Deterministic local evidence does not establish provider acceptance,
  credentialed behavior, whole-process heap/RSS under E2 load, or live-market
  chronology; those claims were neither made nor required here.
- **D1 handoff:** Capability D may consume one ordered Massive ingress whose
  T/Q command result is installed before any admitted frame strictly after its
  exact boundary. D1 must preserve aggregate/control classification and loss
  accounting, selected-row T/Q bounds and trust closure, aggregate readiness
  independence, and the accepted one-second combined cadence. D1 remains
  inactive and requires its own fresh pre-assignment audit before activation.

### Owner-approved bounded parallel hydration revision — 2026-08-24

Stop before D1. The ordinary scanner must regain the previously working
bounded `1|2|4|8` hydration surface with default 8 through `LBR-A3`. This is a
blocking-acquisition pool, not another engine owner or mutation queue. A2's
generation, terminal, merge, fence, and recovery meanings remain accepted;
only its one-worker composition restriction is superseded. D1 stays inactive
until A3 is implemented, verified, reviewed when triggered, accepted, and
committed.

#### `LBR-A3` pre-assignment audit and activation — 2026-08-24

- **Production constructor audit:** the public wrapper, private launcher,
  scanner flag/budget construction, `operations.LiveComponents`, and Massive
  live-plan seam form one ordinary-live path. They currently enforce the A2
  one-worker restriction; the separated generic worker constructor remains
  reachable only by historical/offline tooling and is not a production
  fallback.
- **Concurrency and false-success audit:** the retained worker pool already
  bounds blocking acquisition, copies immutable chunks, joins every started
  worker, and returns one cleanup terminal per planned request. The engine
  remains the only generation/request-ledger, merge, terminal, fence, and
  currentness owner. The independent live pump remains active until all REST
  work is reconciled, and cancellation/epoch loss closes provider work and
  fences late facts. The allocated primary proof directly distinguishes an
  active-worker overshoot, out-of-order missing/duplicate terminal, early
  fence, resident-budget overflow, stalled live consumer, and unjoined
  cancellation.
- **Bounds and edge cases:** supported counts are exactly `1|2|4|8`; default is
  8; normalized capacity remains `population * 57,600`; resident capacity is
  exactly `workers * 57,600`; multiplication is checked before runtime start.
  Empty, malformed, failed, canceled, fenced, duplicate/foreign-token, and
  long-stall recovery meanings remain owned by accepted A1/A2 evidence and are
  not revised. Checkpoint/replay constructors stay unsupported by the live
  seam.
- **Activation:** no consequential ambiguity requires pre-code narrow review.
  `LBR-A3` is the sole active write-capable slice; D1 remains inactive.

#### `LBR-A3` — accepted after final-review correction 2026-08-24

- **Coherent behavior:** the ordinary wrapper, private launcher, scanner,
  operations live seam, and retained Massive blocking-acquisition pool now
  accept exactly `1|2|4|8` workers and default to 8. Normalized capacity is
  exactly `population * 57,600`; resident capacity is exactly
  `workers * 57,600`; checked multiplication and composition validation occur
  before runtime or credential access. REST responses transfer concurrently
  without surrendering the shared cumulative response-byte limit.
- **Ownership and ordering:** workers return immutable bounded chunks and one
  terminal per request. The one engine remains the sole canonical merge,
  generation/request-ledger, fence, lifecycle, and currentness owner. The live
  consumer continues independently while REST work is active; shutdown stops
  scheduling, contains late work, cancels provider calls, and joins the full
  pool.
- **Proof:** `P-LBR-A3-PARALLEL-HYDRATION` passes at `1`, `2`, `4`, and `8`
  with 32 symbols. A deterministic barrier observes maximum active REST work
  of exactly the configured count, unequal releases force both response and
  engine-terminal order away from plan order, all 32 terminal identities
  reconcile before the ingress fence, live aggregate frames advance during
  hydration, and final canonical/ranking/TQ product projections are identical.
  The API boundary separately captures the runtime, maps through the production
  v2 mapper, and compares encoded product output across all counts after
  out-of-order terminal completion. Direct proofs establish exact resident and
  cumulative budgets plus joined cancellation at every supported count.
- **Verification/review:** the focused proof matrix passed three repetitions;
  affected Massive/operations/scanner/launcher short and race suites, API-v2
  mapping proof, focused vet, wrapper argument tests, diff hygiene, and exact
  ordinary repository short verification pass on final code bytes. Final
  read-only review first reopened serialized response-body reads, a flaky fence
  observation, a permissive resident-cap seam, missing engine terminal/budget
  observation, and the API-v2 proof boundary. Corrections passed focused
  re-review `CLEAN/PASS` with no remaining P1/P2/P3 finding.
- **Limitation and handoff:** this proves bounded local composition and semantic
  equivalence, not Massive speedup, provider rate-limit behavior, provider
  availability, market-hours readiness, whole-process resource plateaus, or a
  trading edge. Accepted A1/A2 and B/C meanings are unchanged. D1 receives the
  same ordered Massive ingress/state/TQ interface and is next but inactive
  until its fresh pre-assignment audit.

### Watermark-stall diagnostic preservation correction — 2026-08-24

- **Provenance and placement:** baseline diagnostic commit
  `e88eef766940bcc953fec2e4b1bc8f213c3ef7a0`, descended from replacement
  baseline `0d043c16cea4`, corrected incomplete active-cycle attribution after
  an owner-authorized baseline incident. Its behavior was semantically
  forward-ported after accepted A3 and before D1 without merging the baseline
  branch, changing the A→B→C→D→E route, or reopening accepted A/B/C semantics.
- **Coherent behavior:** one fixed-cardinality atomic engine observation now
  follows replacement maintenance, compact selection/selected-row enrichment
  staging, candidate application, and sole immutable publication. Runtime
  evidence separately times live-coverage enqueue and ordered completion;
  completed evaluation timing is correlated with the live-coverage
  disposition's own engine sequence. The existing shared latch retains the
  exact ready-to-`watermark_stale` crossing for one protected, bounded,
  create-without-overwrite persistence attempt under `var/diagnostics`.
- **Ownership and non-scope:** the observation owns no symbol, event, payload,
  provider, readiness, or publication state. B3 exclusion remains strict: no
  removed evaluator, qualification clone, parallel feature state, second
  publication owner, legacy fallback, or D1 ingress/decoder interface was
  restored. Readiness semantics, four-second target, two-second tolerance,
  watermark ownership, ranking, API v2, T/Q policy, replay, and checkpoints are
  unchanged.
- **Proof and review:** focused engine/operations/scanner proofs, complete
  affected packages, ordinary repository short verification, affected engine/
  operations/scanner race coverage, focused vet, and diff hygiene pass. Final
  read-only review first found that a live aggregate trust-correction cycle
  could perform replacement selection/enrichment without an active view while
  ordered live-coverage completion waited. The correction now starts an
  explicit `trust_correction` stage before that production scan and tracks its
  apply/publication phases. Focused re-review found the adjacent same-`T`
  selected-row trust-closure candidate had the same omission; it now uses the
  same source/phase path. A second focused re-review found that the initial
  selected-row timing seal also advanced B's semantic trust-correction
  coalescing identity. Timing capture is now separate from that revision/latch,
  and the production regression proves both remain unchanged. All three
  deterministic counterexamples pass, and final focused re-review returned
  `CLEAN/PASS` with no remaining finding.
- **Evidence limitation and handoff:** the historical 2026-08-24 incident was
  produced by the baseline architecture and remains an ignored runtime artifact,
  not replacement live evidence. No replacement-branch provider run exercised
  the correction. D1 is the next permitted slice and remains inactive until its
  normal pre-assignment audit.

#### `LBR-D1` pre-assignment audit and activation — 2026-08-24

- **Authority and dependency audit:** `AGENTS.md`, the replacement parent and
  program, the D1 ticket, the complete ingress contract, and its routed
  canonical, evaluation/publication, T/Q, provider-normalization,
  transport/epoch, heartbeat, data-confirmed-subscription, and retained
  `DTE-*` dependencies agree on one boundary. D1 may replace only the
  double-pass Massive frame decoder with one bounded immutable decoded batch
  and a decode-free temporary bridge. The raw-frame queue, engine FIFO,
  command/write linearization, heartbeat/retry execution, fence production,
  lifecycle, and canonical/evaluation/TQ owners remain unchanged until their
  allocated later slices.
- **Production-constructor and fallback audit:** the ordinary scanner has one
  production `massive.NewLiveAdapter` constructor in `cmd/scanner`; its queue
  configuration reaches the adapter's sole final validator and caps a source
  frame at 8 MiB. The current `liveFrameCursor` is the only production frame
  classifier used by handshake and ordinary dequeue. It performs the rejected
  pre-analysis/full second decode and retains per-element `json.RawMessage`
  copies, so D1 replaces that implementation in place. No alternate decoder,
  private launcher default, unsupported tool, or old-code fallback constructs
  a second production normalization path. The existing raw queue and
  per-result adapter delivery remain explicitly temporary D2 dependencies, not
  D1 fallbacks.
- **Boundary and false-success audit:** correction, event-time, duplicate,
  cadence, and watermark horizons remain owned by accepted A/B/C state; D1
  preserves their normalized inputs exactly. The decoder owns the exact
  500-ms T/Q classification-budget equality, 8-MiB source-frame bound, 65,536
  element bound, and 32-MiB retained-batch charge. Empty input, malformed or
  truncated JSON, missing/malformed/duplicate discriminator, duplicate
  recognized members, localizable malformed A/T/Q, unsupported families,
  unknown additive members, pressure shed, and oversize/charge failure each
  receive one closed ordered disposition. The dangerous false success is an
  ambiguous or unadmitted aggregate/control suffix followed by a current
  claim; earliest ambiguity therefore preserves only its complete causal
  prefix and forces the existing epoch-integrity terminal, while pressure shed
  continues classifying later aggregate/control elements.
- **Accepted-handoff and long-stall audit:** the accepted C handoff requires
  command result installation before any frame strictly after `B`, independent
  T/Q trust closure, aggregate readiness independence, and the one-second
  combined cadence. D1 changes none of those locks, boundaries, consumers, or
  state transitions; its temporary bridge emits the same ordered normalized
  facts and terminal ambiguity. A duration beyond every aggregate/TQ retention
  horizon with stalled committed time remains owned by accepted A/B/C behavior,
  and the preserved watermark diagnostic observes the same engine completion
  sequence rather than decoder-private timing or state.
- **Proof and review decision:** `P-LBR-D1-DECODE` directly distinguishes a
  second JSON pass/raw alias, mutable batch backing, lost causal prefix,
  silently accepted ambiguous suffix, incorrect duplicate localization,
  pressure shedding that hides later aggregate/control, element/charge
  overflow, and semantic drift across the approved fixture corpus. These
  boundaries are construction- and primary-proof-resolved, so no additional
  pre-code narrow review is required. The user-required independent read-only
  D1 implementation review remains mandatory before acceptance and commit.
- **Activation:** `LBR-D1` is the sole active write-capable slice. D2,
  credentials/provider requests, the private scanner, and every market,
  readiness, T/Q, queue, state-owner, and handoff change remain inactive.

#### `LBR-D1` acceptance — 2026-08-24

- **Behavior now available:** Massive classifies each admitted source frame in
  one `json.Decoder` token pass into one private, ordered `DecodedBatch`. The
  batch owns its normalized results and envelope/accounting metadata without
  retaining the source bytes. It enforces the existing 8-MiB frame ceiling,
  the 65,536-result ceiling, and a 32-MiB retained charge that includes result
  slice capacity and dynamic string storage. Recognized fields retain their
  JSON scalar kinds; unknown additive values are structurally skipped. The
  earliest ambiguous array element preserves its causal prefix, drains only a
  structurally valid fenced suffix, and cannot be duplicated by trailing frame
  corruption. Exact 500-ms pressure sheds only T/Q and continues classifying
  later aggregate/control input. Handshake and ordinary delivery consume this
  batch through the existing one-result bridge in unchanged order.
- **Ownership and deferral:** D1 changes no canonical, evaluation, readiness,
  T/Q, event-time, watermark, lifecycle, heartbeat, recovery, queue, engine
  FIFO, command-lock, or fence ownership. The raw-frame queue, temporary
  result-at-a-time bridge, and engine FIFO are intentionally retained for D2;
  D1 neither introduces an alternate decoder nor begins the one-handoff
  cutover. The accepted A/B/C/A3 behavior and the watermark-stall diagnostic
  remain on their prior state and completion sequence.
- **Primary proof and dangerous counterexamples:** `P-LBR-D1-DECODE` covers a
  mixed A/T/Q/status/unsupported oracle, exact positions and envelope, source
  mutation after decode, duplicate recognized members, locally rejected
  malformed events, strict scalar-kind violations, valid and corrupt trailing
  input, composed array-plus-frame ambiguity, exact-budget T/Q shedding with a
  later retained aggregate/control suffix, frame/batch bounds, and production
  source exclusion of a second decoder, `json.RawMessage`, `json.Unmarshal`,
  and the old analysis pass. Invalid retained source aliases and a second
  production decode path are prevented by construction. Runtime validation
  closes malformed provider syntax, type, element-count, and charge evidence.
- **Verification on accepted bytes:** `P-LBR-D1-DECODE` passed in 0.581s;
  Massive short passed in 1.649s; the ten direct operations regressions,
  including all A3 worker counts, passed in 27.841s; the four direct operations
  race regressions passed in 3.666s; focused vet and `git diff --check` passed;
  and ordinary repository verification
  `go test -count=1 -short -timeout 2m ./...` passed, with the longest package,
  operations, completing in 85.145s. The final isolated race rerun of the sole
  contended handshake counterexample passed in 1.533s. The isolated 256-
  aggregate decoder benchmark on Apple M1 (`20x`) measured 3,910,931 ns/op,
  7.53 MB/s, 2,481,890 B/op, and 46,870 allocs/op; it excludes engine work and
  is characterization, not an acceptance threshold.
- **Corrections and independent review:** the required read-only
  `gpt-5.6-sol` medium review first found two P2 defects: recognized scalar
  kinds were being collapsed, and trailing JSON could be silently accepted or
  charged as array work. After correction, focused re-review found one adjacent
  P2: an array ambiguity followed by trailing corruption emitted two ambiguity
  results. The corrected implementation preserves the earliest causal array
  ambiguity and records trailing corruption only in frame accounting. During
  final verification, two ordinary-suite attempts with an eight-result initial
  batch allocation timed out in accepted A3 fence reconciliation at about
  121s, although A3 passed alone in 25.069s. Reducing the private initial
  capacity to one removed seven unused union slots from common one-result
  frames; the next ordinary run passed. A concurrently loaded full Massive
  race run then hit one existing unawaited-handshake timing assertion; its
  isolated race rerun passed. Final read-only re-review checked the one-slot
  growth/charge/reserve mechanics and the complete diff and reported no
  P1/P2/P3 findings. Reviewers made no writes and performed no prohibited work.
- **Limitations and next action:** D1 proves deterministic offline decoder
  semantics and bounded ownership, not provider conformance, market-hours
  stability, end-to-end resource acceptance, or executable trading
  expectancy. No credentials or provider requests were used and the private
  scanner was not run. D2 is the next permitted slice, but remains inactive
  until its own fresh pre-assignment audit; its only permitted outcome is the
  parent-authorized decoded-batch FIFO and causal-fence handoff with preserved
  connection semantics.

#### `LBR-D2` pre-assignment audit and activation — 2026-08-24

- **Authority and dependency audit:** `AGENTS.md`, the repository guide and
  specification map, the replacement parent/program, the D2 ticket, the
  complete ingress contract, accepted canonical, evaluation/publication, and
  T/Q contracts, the D1 handoff, provider-normalization and transport/epoch
  evidence, the heartbeat and data-confirmed-subscription corrections, and the
  routed `DTE-*`/`LIFE-RECOVER-01` meanings agree on one ownership boundary.
  D2 may replace only live transport ordering and its handoff: one bounded
  decoded-batch/causal-marker FIFO transfers immutable ownership from the sole
  socket reader to serial engine consumption. The engine remains the only
  market-state, sequence, lifecycle, currentness, recovery, T/Q, watermark,
  evaluation, and publication owner. Hydration results and timers remain the
  only parent-allowed owner-local paths.
- **Production-constructor and fallback audit:** the ordinary scanner has one
  `massive.NewLiveAdapter` production constructor and one queue-default path in
  `cmd/scanner`; tests and unsupported tools construct the same adapter type
  and reach its final validator. The accepted code still constructs
  `liveFrameQueue`, copies raw bytes at admission, decodes after dequeue,
  expands a `DecodedBatch` through `liveBatchCursor` into one
  `AdapterDelivery` per result, and admits each result into the engine's
  general FIFO while waiting on a per-result completion. D2 must remove those
  three temporary stages from the supported path rather than retain a flag,
  alternate constructor, test-only production fallback, or completion
  backlog. Replay/checkpoint sources may continue to compile but cannot
  construct a second live ingress path.
- **Boundary and false-success audit:** the fixed constructor is 4,096 total
  entries/64 MiB retained charge, source frame at most 8 MiB, decoded batch at
  most 65,536 elements/32 MiB, with eight entries/64 KiB reserved for required
  markers; incoherent count/byte/reserve combinations fail before I/O. The
  exact 500-ms decoder budget, strict 749/750-ms T/Q pressure recovery tie,
  whole-frame subscribe boundary `B`, and recovery ordinals 1–5 at
  1/2/4/8/16 seconds remain unchanged. Empty, malformed, duplicate, mixed,
  T/Q-only, oversize, capacity, cancellation, replacement-epoch, heartbeat,
  handshake, read-failure, marker, and terminal paths require one closed
  disposition. The smallest false success is an aggregate/control-bearing
  batch or required marker omitted before a current fence or successful retry
  reset; construction therefore reserves marker capacity, T/Q-only work sheds
  first with exact facts, and aggregate/control admission failure retires the
  epoch and requires gap recovery.
- **Ordering, command, and long-stall audit:** a frame is decoded once
  immediately after its complete read and receives its immutable epoch/frame/
  array positions before ring admission. Dequeue consumes one complete batch
  synchronously; no later batch, fence, terminal, or `B+1` event may overtake
  its predecessor. Fence capture uses the greatest complete frame already read
  and appends behind every admitted batch through that frame. Socket writes use
  one critical section and at most one dynamic command; successful subscribe
  completion returns the greatest complete read frame `B`, failure returns no
  boundary, and generic status never completes the command. A committed-time
  stall beyond all aggregate/T/Q retention horizons followed by epoch loss
  remains an accepted A/B/C recovery case; D2 preserves its exact gap/fence
  inputs and the watermark diagnostic's engine-completion correlation.
- **Attempt, shutdown, and accounting audit:** process start performs one
  immediate dial, then recovery attempts 1–5 are individually paced after the
  prior attempt is fully retired and joined; loss/failure of attempt 5 reaches
  stable exhaustion with no attempt 6. Only an accepted reconciled hydration
  fence resets that budget. One reader and one heartbeat operation are owned by
  the attempt; inbound progress after heartbeat failure is diagnostic, while
  read/fatal/no-progress failure retires the epoch. Retirement accepts one
  first terminal, cancels pending work, closes the socket, drains or fences
  admitted causal predecessors, reconciles queue/family/command counts, and
  joins before redial or return.
- **Proof and review decision:** `P-LBR-D2-HANDOFF` must compose the real ring,
  sole engine consumer, fake socket, and deterministic clock across mixed
  ordering, marker placement, T/Q-first shedding, T/Q-only and mixed
  saturation, aggregate/control overflow recovery, `B`/`B+1`, handshake and
  heartbeat branches, read loss, stale epochs, exact retry pacing/exhaustion,
  cancellation, and joined shutdown. Queue saturation, marker linearization,
  attempt retirement, and removal of the second live FIFO are consequential
  concurrency/false-success boundaries, so the required user-requested final
  read-only Capability D review will inspect the complete D1+D2 result before
  acceptance and commit; any finding reopens the smallest D2 boundary.
- **Activation:** `LBR-D2` is the sole active write-capable slice. E1,
  credentials/provider requests, the private scanner, replay/checkpoint work,
  and every market, readiness, T/Q-formula, queue-owner, decoder-owner, or
  additional-handoff change remain inactive.

#### `LBR-D2` acceptance and Capability D final acceptance — 2026-08-24

- **Coherent behavior and ownership:** the sole socket reader now reserves one
  complete-frame causal sequence, decodes that frame once, releases the source
  bytes, and transfers one immutable `DecodedBatch` into the only buffered live
  ordering structure. The physical ring is exactly 4,096 entries/64 MiB: 4,088
  decoded entries/63.9375 MiB plus eight entries/64 KiB reserved for causal
  markers. Dequeue transfers one complete entry through an unbuffered request/
  completion rendezvous to the existing sole engine loop; every contained fact
  is consumed serially before another live entry, and none enters the engine's
  general FIFO or creates a per-result completion backlog. Startup, handshake,
  dynamic command results, hydration/live-coverage fences, capacity closure,
  and terminals use the same owner handoff.
- **Ordering, capacity, and failure behavior:** frame/array order and immutable
  positions remain exact. Fence capture waits for any active decode, records
  the greatest complete frame read, and appends behind every admitted causal
  predecessor. Successful subscribe write completion still captures whole-
  frame `B`, and the shared delivery lock keeps the command result ahead of
  `B+1`. T/Q-only ring saturation emits one compact ordered trust-closure
  marker with exact trade/quote shed accounting; aggregate/control-bearing
  saturation records the exact slot/byte cause, retires the epoch, drains every
  admitted predecessor, and requires gap recovery. The raw-frame queue,
  production per-result cursor/handshake accumulator, result-at-a-time bridge,
  and production Massive-to-engine `Admit*` live path are deleted; historical
  cursor/call-shape compatibility exists only in `_test.go` files.
- **Attempt, heartbeat, and shutdown preservation:** one pre-reserved attempt-
  owned worker owns dial and all handshake phases and joins before cleanup can
  clear the active attempt or permit replacement. Reader and heartbeat work
  remain attempt-owned and joined. Auth, aggregate subscribe, dynamic T/Q
  writes, and heartbeat ping share one attempt-local write mutex while reads
  remain independent. Heartbeat inbound-progress/no-progress behavior, one
  immediate process-start dial, exact 1/2/4/8/16-second recovery attempts 1–5,
  no attempt 6, and reset only after an accepted recovery fence remain exact.
  Cancellation during decode reconciles queue accounting; cancellation during
  dial retains `context_canceled`, genuine connector failure retains
  `dial_failed`; handshake decode ambiguity applies its valid prefix plus
  ingress-integrity fact before preserving the exact ambiguity terminal.
- **Primary proof, counterexample, and construction guarantees:**
  `P-LBR-D2-HANDOFF` is composed by `TestPLBRD2Handoff`,
  `TestPLBRD2ProductionTopology`,
  `TestPLBRD2HandoffUnbufferedBatchBypassesGeneralFIFO`, the exact dial/
  handshake ambiguity/write-serialization proofs, and retained focused
  heartbeat, `B`/`B+1`, capacity, fence, retry, recovery, exhaustion, and join
  regressions. It distinguishes the dangerous aggregate/control-loss-then-
  current-fence case, an unbounded handshake delivery slice, stuck decode
  accounting, concurrent socket writes, an unjoined old dial overlapping a
  replacement, overwritten ingress ambiguity, and cancellation mislabeled as
  dial failure. One ring, one reader, one active attempt, immutable batch
  ownership, marker reserve, and unbuffered owner handoff are construction
  guarantees; stale epochs/tokens, malformed input, bounds, transport failure,
  and terminal duplication remain runtime validations.
- **Verification on final bytes:** the direct D2/command/heartbeat/reconnect
  selection passed in engine 0.646s and Massive 0.555s; affected short engine,
  Massive, operations, and scanner packages passed during final correction;
  affected race passed in 69.806s, 4.359s, 86.553s, and 2.960s respectively;
  focused vet and `git diff --check` passed. After the final test-only cursor
  relocation, the uncached ordinary repository command
  `go test -count=1 -short -timeout 2m ./...` passed, with operations longest at
  85.410s; focused proof/vet/diff reruns also passed.
- **Independent review and corrections:** the required read-only
  `gpt-5.6-sol` medium review first found an unbounded handshake per-result
  accumulator, decode-cancel accounting leak, and unsynchronized ping/write.
  Re-review then found unjoined dial/handshake work and overwritten handshake
  ingress ambiguity, followed by one adjacent dial-cancellation first-cause
  error. Each reopened only its narrow D2 boundary; the corrections above and
  their distinguishing proofs passed focused re-review. Final complete and
  mechanical-removal re-reviews returned `CLEAN/PASS` with no P1/P2/P3
  finding. Reviewers made no writes.
- **Limitations and next gate:** this is deterministic fake-transport and
  bounded local evidence, not credentialed provider conformance, market-hours
  continuity, the E2 whole-process resource plateau, replay/checkpoint support,
  or trading expectancy. No credentials/provider request/private scanner were
  used. D1 and D2 now complete Capability D with its required final review;
  `LBR-E1` is the next permitted slice but remains inactive pending its own
  fresh pre-assignment audit.

#### `LBR-E1` pre-assignment audit and activation — 2026-08-24

- **Authority, dependency, and owner-decision audit:** `AGENTS.md`, `README.md`,
  the specification map, replacement parent/program, all five routed focused
  contracts, the accepted A1-D2 handoffs and proofs, the frozen baseline, the
  E1 ticket, and the complete integration/removal contract agree on one
  boundary. E1 may change only production wiring, delete superseded source and
  lower-authority proofs/routes, and preserve the accepted A/B/C/A3/D1/D2 plus
  API-v2/UI/launcher behavior. The owner selected the recommended disposition:
  delete `cmd/aggregate-replay`, `internal/replay*`,
  `internal/replayartifact*`, `internal/replaymode`, `internal/checkpoint`, and
  every live engine/runtime/operations/API/test dependency. Replay/checkpoint
  is not repaired, retained as unsupported tooling, or made an acceptance gate.
- **Production-constructor and dependency audit:** at activation,
  `go list -deps ./cmd/scanner` still reaches all five replay/checkpoint package
  families. `cmd/scanner/main.go` still selects live versus replay, accepts
  replay/checkpoint flags, calls a checkpoint-aware runtime composer, and
  imports replay packages. `operations.NewWithCheckpoint`,
  `LiveComponents.Store`, checkpoint result work/metrics, engine replay mode,
  checkpoint admissions/state/cadence, checkpoint hydration purpose, snapshot
  replay mapping, and checkpoint-derived API mapping therefore survive in the
  ordinary live dependency graph. E1 must remove those branches and imports,
  not merely make them unreachable. API v2 retains one fixed honest
  disabled/not-installed checkpoint object sourced from live configuration,
  with no engine checkpoint read; its optional live replay object remains
  absent.
- **Accepted replacement topology audit:** the ordinary path constructs one
  `operations.Runtime`, which constructs one engine owner; one
  `massive.LiveAdapter` owns the accepted 4,096-entry/64-MiB decoded-batch and
  causal-marker FIFO; the unbuffered complete-batch rendezvous reaches the sole
  engine loop without the general engine FIFO; B1/B2/C1/C2 provide the sole
  evaluation, immutable publication, and selected-row T/Q path. The file
  `internal/massive/live_queue.go` now implements the accepted D2 decoded ring
  despite its historical name and must remain. The supported hydration surface
  is exactly `1|2|4|8`, default 8, through wrapper, launcher, scanner,
  operations validation, and the bounded Massive worker pool; those workers
  produce immutable facts and are not competing state owners.
- **Removal and compatibility audit:** remove replay/checkpoint command,
  package, engine state/lifecycle/admission, operations composition/metrics,
  Massive checkpoint-catchup, scanner/launcher flag, snapshot replay branch,
  private-structure proof, fixture/golden, and active README/runbook routing.
  Preserve historical specifications/correction records as evidence, with the
  specification map continuing to label replay/checkpoint contracts historical.
  Preserve the one-load immutable capture, API-v2 live schema/semantics,
  dashboard polling/rendering, launcher supervision, hydration/fence,
  readiness/recovery, diagnostics, heartbeat/retry, command ordering,
  terminals, and operator output.
- **False-success and proof decision:** the smallest false success is a green
  live composition while an error/config/build path still selects old state,
  a shadow replay/checkpoint mutation or metric remains in the engine, a second
  queue/owner/publication is constructible, deleted tooling imports the live
  core, an old product field reaches state/API, or a fixed checkpoint-off API
  value is synthesized from live checkpoint state. `P-LBR-E1-CUTOVER` must
  combine build/dependency/source exclusion with the real production
  composition trace and forced sentinel counterexamples. Because this is a
  cross-cutting owner/concurrency/removal boundary, E1 requires the program's
  final read-only review and a clean focused re-review after any correction.
- **Activation and exclusions:** `LBR-E1` is the sole active write-capable
  slice. E2 and its 10-minute manifest, E3, credentials/provider requests, the
  private scanner, product/market/readiness/T/Q semantic changes, added
  queues/owners/handoffs, replay/checkpoint redesign, and unrelated cleanup
  remain inactive.

#### `LBR-E1` acceptance — 2026-08-24

- **Coherent behavior and ownership now available:** `cmd/scanner` is live-only
  and constructs exactly one `operations.Runtime`/engine state owner, one
  Massive decoded-batch FIFO and causal-fence handoff, and one accepted
  evaluator/immutable-publication/API-v2/dashboard/launcher path. Hydration is
  the only parallel worker pool and remains bounded to `1|2|4|8`, default 8.
  The API-v2 checkpoint object is an honest fixed disabled/not-installed
  representation with no engine checkpoint owner or read.
- **Deletion and interface result:** the owner-selected deletion removed
  `cmd/aggregate-replay`, `internal/checkpoint`, `internal/replay`,
  `internal/replayartifact` (including playback), `internal/replaymode`, and
  their scanner, engine, operations, Massive, API, active-document, fixture,
  golden, and test dependencies. Engine run-mode selection, replay views and
  schema fields, checkpoint state/cadence/admission, offline acquisition and
  replay benchmarks, compatibility flags, and private deterministic-canonical
  proof surfaces are absent. Shared live REST acquisition now has a live-only
  home; the accepted D2 decoded ring remains the sole ingress queue.
- **Primary proof and dangerous counterexamples:** `P-LBR-E1-CUTOVER` combines
  production build/dependency/source exclusion with a real `Runtime.RunLive`
  composition trace over deterministic fake WebSocket/REST transports. It
  proves hydration, live-over-REST precedence, startup and ordinary causal
  fences, one current publication, watermark/population/qualification/T/Q and
  accounting results, cancellation and joined shutdown. Constructor/source
  sentinels reject fallback modes, competing owners, shadow mutation,
  additional queues/workers, removed product/schema fields, checkpoint-derived
  API state, deleted-tool imports, and duplicate scalar flags while retaining
  repeatable `allow-origin`.
- **Final-byte verification:** `go test -count=1 -run
  '^TestPLBRE1Cutover$' -timeout 90s ./cmd/scanner` passed (package 1.038s,
  wall 3.24s). API/UI/launcher regressions passed: focused Go packages (wall
  2.89s) and 48/48 Node UI tests (195.332ms test time, wall 0.46s). Affected
  short tests passed (wall 70.91s); affected race tests passed (wall 76.04s);
  focused vet passed (wall 0.77s); `git diff --check` passed (wall 0.14s); and
  `go test -count=1 -short -timeout 2m ./...` passed (wall 70.55s). Production
  commands built (wall 1.22s); scanner dependency and production-source
  exclusion inspections passed (wall 0.07s and 0.01s).
- **Independent review and correction:** the required read-only
  `gpt-5.6-sol` medium review first found that the proof used only constructor
  smoke coverage, retained replay-named schema/private views and engine mode,
  and had lost duplicate scalar-flag rejection. After correction, focused
  re-review found that the trace still assembled private pieces rather than
  calling `RunLive`, plus one tautological assertion. The final trace uses the
  real production path and independent expected values; the same reviewer
  returned `CLEAN/PASS` with no P1/P2/P3 findings. The reviewer made no edits.
- **Limitations and next gate:** the composition proof uses deterministic fake
  transports and does not establish provider conformance, sustained resource
  plateaus, market-hours continuity, or trading expectancy. The API failure
  cases are separate focused subtraces rather than socket-composition failures.
  No credentials/provider request/private scanner or E2 manifest was used.
  `LBR-E2` is the next permitted slice but remains inactive; E3 remains
  unauthorized/inactive.

#### `LBR-E2` pre-assignment audit and activation — 2026-08-25

- **Authority and accepted dependency audit:** `AGENTS.md`, `README.md`, the
  specification map, replacement parent/program, all five routed focused
  contracts, the accepted A1-D2 handoffs, the frozen `LBR-P1` characterization,
  the E2 ticket, and the complete E1 acceptance record agree on one boundary.
  E2 owns only the exact deterministic fixture, whole-composition measurement,
  acceptance evidence, and blocking correctness corrections routed through the
  narrowest owning boundary. It does not own product, market, readiness, T/Q,
  provider, replay/checkpoint, queue, worker, or state-owner semantics.
- **Frozen manifest audit:** `lbr-mature-v2:2026-08-12` remains the sole input:
  5,694 symbols, 5,470,012 mature hydration rows, 600 timed seconds at 300
  frames/s, 180,000 frames, 3,416,400 base aggregates, 600 resource samples,
  600 dashboard polls, and 11 ranking/accounting checkpoints. Its seed,
  population/prior-close/event-count/ranking/accounting/API/UI digests, fixed
  injection classes, queue/TQ bounds, stop conditions, host/Go/GC facts, and
  one-run/no-repeat rule are unchanged. Validation belongs inside the sole
  timed command before pacing begins; no preflight, alternate, shortened,
  fallback, diagnostic, coexistence, or target-miss trial is authorized.
- **Current production dependency audit:** HEAD `6af93cd303237cd555b491cceef4aac5099d9be4`
  on `codex/live-backend-replacement` has a clean worktree. The supported
  scanner dependency graph remains live-only and excludes the deleted replay/
  checkpoint packages. `cmd/scanner` constructs one `operations.Runtime`, one
  engine owner, the accepted single decoded-batch/causal-marker FIFO, one
  evaluator/publication, API v2, and the independent dashboard path. Hydration
  remains the sole worker pool at exactly `1|2|4|8`, default 8. The module has
  only `github.com/coder/websocket v1.8.15`; the audited scanner dependency-list
  SHA-256 is
  `a20b6a06a6457fd7d7b4ad9cb20061155cf6ef20ca9f61374113408746c91d42`.
- **False-success and proof decision:** `P-LBR-E2-STABILITY` must reject a
  semantically green final endpoint with aggregate/control loss, incoherent
  accounting, growing retained state/heap/RSS/FIFO/goroutines, sustained
  backlog, recurring readiness flap, or unusable isolated one-second polling.
  Every parent numeric target is reported from the one run; a numeric miss with
  all hard gates passing is a recorded deviation and never authorizes a repeat.
- **Activation and exclusions:** `LBR-E2` is the sole active write-capable
  slice. The exact 10-minute command will use `-timeout 15m` once and will not
  be rerun. Credentials/provider requests, the private scanner, E3, product/
  market/readiness/T/Q changes, replay/checkpoint restoration, added queues/
  workers/state owners, production-capacity tuning, and unrelated cleanup
  remain unauthorized/inactive.

#### `LBR-E2` invalid measurement and reopened handoff — 2026-08-25

- **Outcome:** `P-LBR-E2-STABILITY` did not reach its explicit
  `LBR-E2 timed pacing started` boundary. The final bounded invocation stopped
  after 885.01 test seconds with `aggregate connection ended before hydration
  fence reconciliation`; therefore timed seconds, provider frames, base
  aggregates, samples, dashboard polls, and checkpoints are all exactly zero.
  This is measurement invalidity, not a semantic/resource pass or numeric miss.
- **Root cause and disposition:** the deterministic websocket fixture
  completed its handshake and then waited for the timed-start signal, while the
  engine correctly required a later causal source frame to reconcile the
  generation-two hydration fence. Independent review then established that a
  proposed signal/status-frame correction remained racy and that the harness
  bypassed ordinary `Runtime.RunLive`, the eight-worker REST hydration pool,
  and the dashboard model. The invalid harness and its proof-only production
  hydration seam were removed. The 10-minute trial was not rerun.
- **Partial setup observation:** the proposed manifest validator, 5,694-symbol binding,
  5,470,012-row mature hydration expectation, exact generation-one terminal
  identity (`5,688 value + 2 empty + 1 failed + 1 canceled + 2 fenced`), and
  engine-selected generation-two replan (`S0002`–`S0005`) completed before the
  failed fence wait. These observations are not accepted manifest, production-
  composition, or semantic evidence. The harness did reject false success
  rather than reporting a green endpoint without a reconciled fence.
- **Unavailable timed evidence:** semantic/accounting/ranking/API/UI digests;
  aggregate/control loss and rejection; CPU average/p95; hydration/steady heap;
  RSS; allocation rate; hydration/steady goroutines; FIFO count/bytes/age/slope;
  selection cycle; API capture; additional processing delay; watermark and
  readiness transitions; plateau behavior; and one-second polling usability
  are all unmeasured. No inference is made from setup or earlier capability
  evidence. The durable machine-readable record is
  `docs/live-backend-replacement/evidence/lbr-e2-acceptance.json`.
- **Invocation record and deviation:** the exact command was
  `env LBR_E2_ARTIFACT=/Users/joshuabelandres/.codex/worktrees/live-backend-replacement/live-equities-momentum-scanner/docs/live-backend-replacement/evidence/lbr-e2-acceptance.json go test -v -count=1 -timeout 15m -run '^TestPLBRE2DeterministicResourceStability$' ./internal/operations`.
  Measurement-tool setup was repeatedly invoked while correcting pre-pacing
  failures; every invocation and available test-reported duration is retained
  in the artifact. None emitted the timed-start marker or consumed paced input.
  The final invocation used the fixed 15-minute timeout and is the terminating
  E2 evidence; no diagnostic, shortened, fallback, or replacement timed trial
  was run.
- **Independent integrated review:** the required read-only review returned
  `FAIL`, not clean acceptance. It found the racy fence correction, incomplete
  production composition, incomplete semantic/API/UI/accounting and plateau
  gates, incomplete manifest validation, an unauthorized production proof seam,
  and irreproducible dirty-worktree provenance. The invalid code was removed;
  the provenance limitation cannot be repaired retrospectively and remains in
  the artifact. A clean E2 acceptance review is impossible with zero timed
  seconds.
- **Final failure-handoff bytes:** all harness and engine changes identified by
  review are absent. JSON validation and `git diff --check` pass, and ordinary
  `go test -count=1 -short -timeout 2m ./...` passes in 72.71 seconds on the
  final documentation/evidence-only diff. Earlier affected/race/UI results are
  retained in the execution history but are not used to validate the removed
  harness or substitute for E2.
- **Final focused re-review:** the same independent reviewer returned
  `CLEAN/PASS` for the failure handoff after confirming the diff is documentation/
  evidence only, the invalid harness and production seam are absent, the JSON
  is valid and candid about dirty-worktree provenance and unavailable evidence,
  and E3 remains inactive. The reviewer explicitly did **not** accept E2:
  `deterministically_complete` remains unestablished. The evidence artifact
  SHA-256 is
  `b4d57ab3da41b73eeafc5ec3ebf135552e58c4ba784eff16fe311dafbba218f6`.
- **Correction/next gate:** E2 is reopened at its measurement-fixture boundary.
  A future E2 activation requires a new program/owner authorization and again
  owns exactly one 10-minute run. Capability E is not finally accepted,
  `deterministically_complete` is not established, and `LBR-E3` remains
  unauthorized/inactive.

#### Owner revision — defer E2 and permit E3 next — 2026-08-25

- **Decision:** the failed E2 harness and evidence remain preserved, but E2 is
  deferred as optional, non-gating deterministic capacity characterization. Its
  exact frozen manifest, 10-minute duration, one-run/no-repeat rule, and claims
  remain unchanged if a future owner activation resumes it. No E2 code is
  retained in production or test packages.
- **Reason:** the owner prioritizes evidence from the actual Massive REST and
  WebSocket paths, ordinary `RunLive` composition, state engine, API v2, and
  local dashboard over near-term investment in a full fake-provider composition.
  Live evidence gives stronger provider-integration confidence but cannot prove
  exact 300-frames/s deterministic capacity or forced rare-event coverage.
- **Revised ordering:** accepted A–D semantics, E1 cutover, and existing
  deterministic component/ordinary evidence are sufficient to permit E3 next.
  E3 remains inactive until an execution prompt explicitly authorizes the exact
  trading date, ordinary Massive environment and credential source, bounded
  duration, procedure, redaction, stop/shutdown behavior, and any permitted
  recovery action. The activation audit must verify a clean exact commit before
  credential access.
- **Claim boundary:** a clean E3 observation and final integrated program review
  may establish `live_stability_confirmed` for the observed private/local
  scanner path. It
  does not establish `deterministic_capacity_characterized`, provider SLA,
  public deployment, replay/checkpoint support, or trading expectancy. A future
  clean E2 run may separately establish
  `deterministic_capacity_characterized`; neither status depends on the other.
- **Next permitted slice:** `LBR-E3` is next permitted but inactive. This
  program revision is not credential authority and makes no provider request.
- **Independent authority review:** final read-only review returned
  `CLEAN/PASS` after aligning repository-wide phase routing, sole-ledger states,
  E3's final integrated review scope, exact authorization for any induced
  recovery or T/Q shedding, authority metadata, and deferred-E2 terminology.
  The reviewer confirmed that E3 can be authorized next without production-code
  changes or deletion/rewrite of E2 evidence.

### Owner-approved E2 duration and manifest revision — 2026-08-23

The owner revised E2 to exactly one 10-minute deterministic acceptance run. The
fixed population/rate/polling and
hard gates remain: 5,694 symbols, 300 frames/s, one-second dashboard polling,
all event classes, exact semantic/accounting/API/UI results, zero aggregate/
control loss, bounded CPU/memory/queue/goroutine plateaus, stable readiness,
and usable isolated polling. The owner-revised manifest is exactly 600 timed
seconds, 180,000 frames, 3,416,400 base aggregates, 600 resource samples, 600
dashboard polls, and 11 ranking/accounting checkpoints with recomputed count,
ranking, and accounting digests. No optional, fallback, diagnostic, host-
coexistence, or non-gating repeat run exists.

## 14. Sole delivery ledger

This table is the only mutable program status. Focused specs name allocations
but copy no status.

| Item | State | Gate / next action |
| --- | --- | --- |
| Parent architecture | `approved_owner_revised_e3_next_e2_deferred` | Current replacement architecture; bounded live hydration accepts `1|2|4|8`, E2 capacity characterization is deferred/non-gating, and E3 is permitted next only under exact provider authorization. |
| Delivery program | `approved_owner_revised_e3_next_e2_deferred` | A–D/E1 remain accepted; E3 is next permitted but inactive, and no fallback/diagnostic/repeat E2 composition is authorized. |
| `LBR-P1` focused contracts and characterization | `accepted_frozen_e2_deferred` | Five-spec review/owner acceptance remains valid. The frozen E2 baseline/manifest is retained for optional future reactivation and does not gate E3; comparable baseline CPU/RSS remain explicitly unknown. |
| Capability A — canonical state and hydration | `finally_accepted_lbr_a3` | A1/A2 semantics remain accepted; A3 supersedes only their one-worker ordinary-live composition restriction. |
| `LBR-A3` bounded parallel live hydration | `accepted` | Exact `1|2|4|8`, default 8, bounded concurrency/budgets, terminal/fence/live-tail/API equivalence, joined cancellation, and final focused review are clean. |
| Watermark-stall diagnostic preservation | `accepted_between_a3_d1` | Baseline commit `e88eef7` was semantically forward-ported with offline verification and clean final review; diagnostic-only, no A/B/C reopening or D1 activation. |
| `LBR-B3` removal slice | `accepted` | Commit `f697288`; removal proof, resource evidence, focused corrections, and final focused re-review are clean. |
| Capability B — evaluation and publication | `finally_accepted` | B1/B2/B3 and the required final read-only review remain accepted and unaffected by A3. |
| Capability C — selected-row T/Q | `finally_accepted` | C1/C2 remain accepted and unaffected by the A3 hydration-topology revision. |
| Capability D — live ingress | `finally_accepted` | D1/D2, one decoded-batch FIFO/owner handoff, exact connection semantics, final-byte gates, corrections, and final read-only review are clean and preserved by E1. |
| Capability E — integration/removal/acceptance | `lbr_e3_next_permitted_e2_deferred` | E1 exclusive cutover remains accepted. Invalid E2 evidence is preserved and E2 is deferred/non-gating. E3 is next permitted but inactive pending exact execution authorization. |
| E2 deterministic duration/manifest revision | `deferred_non_gating` | If reactivated: exactly one 10-minute run; 5,694 symbols, 300 frames/s, 600 polls/samples, 180,000 frames, recomputed counts/digests, 15-minute command timeout, and no repeat composition. |
| Deterministic capacity characterization | `not_established_deferred` | No timed E2 evidence exists. This optional status requires a newly authorized E2 activation and conforming run but does not gate E3. |
| Live stability confirmation | `next_permitted_not_authorized` | `LBR-E3` may proceed before E2 only after an exact-date/environment/credential/duration/recovery authorization is recorded. |

## 15. Git and milestone policy

- Work only on `codex/live-backend-replacement` in the dedicated replacement
  worktree.
- Preserve baseline `0d043c1` and do not modify `main` or `origin/main`.
- The unrelated institutional-footprint research document in the original
  checkout remains out of scope.
- Stage exact paths only while all workers/reviewers are quiescent.
- Make local commits at approved parent/program, completed focused-contract
  set, accepted slices/capabilities, deterministic integration, and distinct
  final live confirmation when executed.
- Do not push, rebase, amend, rewrite history, delete branches, or use
  destructive resets.

## 16. Completion boundary

The former combined `deterministically_complete` gate is retired for the current
ordering because it made optional whole-composition capacity evidence a
prerequisite for observing the real provider. Current statuses are independent.

`live_stability_confirmed` requires:

- approved parent/program and accepted focused specs;
- Capabilities A-D and `LBR-E1` accepted;
- the supported scanner executing one replacement state/handoff path;
- superseded product feature state and old fallback implementations removed;
- API v2/UI/launcher compatibility proven;
- ordinary/race/vet/diff verification green as allocated;
- one exactly authorized `LBR-E3` observation of the ordinary private/local
  provider path; and
- a clean final integrated program review covering its bounded, redacted live
  evidence, retained one-owner/path conformance, and claim limits.

`deterministic_capacity_characterized` separately requires the deferred exact
E2 manifest, hard resource/stability acceptance with every target measured and
deviations recorded, and its clean independent evidence review. It is not a
gate for `live_stability_confirmed`.

Neither state establishes replay/checkpoint support, public deployment,
provider SLA, or trading expectancy.

## 17. Owner decisions and latest responsible gates

| Decision | Recommended default | Needed by |
| --- | --- | --- |
| Replay/checkpoint source disposition | Remove from supported live binary/core; decide delete versus unsupported tooling | Before `LBR-E1` |
| Market-hours execution | One exact-date owner-run or separately authorized observation | Before `LBR-E3` |
