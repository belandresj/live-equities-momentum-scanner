# Live backend replacement delivery program

**Status:** Owner-approved current delivery authority, 2026-08-23; owner-revised
E2 to exactly one 10-minute deterministic acceptance run on 2026-08-23.

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
- passes semantic, recovery, concurrency, and resource-stability
  acceptance; and
- records live-provider confirmation only when separately owner-authorized or
  owner-executed.

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
| 1 | Canonical state and hydration | `canonical-state-and-hydration.md` | `LBR-A1`, `LBR-A2` |
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

### `LBR-E2` — deterministic resource and stability acceptance

Run exactly one mature 5,694-symbol deterministic 10-minute mixed-feed
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

### `LBR-E3` — separately authorized market-hours confirmation

After deterministic acceptance, one exact-date owner-authorized or owner-run
observation confirms the final provider wiring, hydration, connection
continuity/recovery, ranking, T/Q, API, dashboard, and resource trend.

This program does not itself authorize credential access or a provider request.
Without `LBR-E3`, the implementation may be recorded
`deterministically_complete` but not `live_stability_confirmed`.

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
- one final integrated review after `LBR-E2`, plus focused live-evidence review
  after `LBR-E3` when executed.

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
that compacted before an exact gap request; the integration contract now states
the one-worker invariant through wrapper, launcher, scanner, operations, and
runbook. The program orchestration section now requires a pre-assignment
cross-boundary counterexample audit and final-byte-only expensive verification.

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
| Parent architecture | `approved_owner_revised_e2_10m` | Current replacement architecture; E2 is exactly one 10-minute deterministic run. |
| Delivery program | `approved_owner_revised_e2_10m` | Current replacement delivery authority; no optional/fallback/diagnostic repeat run exists. |
| `LBR-P1` focused contracts and characterization | `accepted_owner_revised_e2_10m` | Five-spec review/owner acceptance remain valid. Owner-revised [`baseline-characterization.md`](baseline-characterization.md), SHA-256 `5745a6eed3e25891f61f58b40b06a4e9733c7b3f7d0914d411b142fe3ad8d62d`, freezes baseline `0d043c1`, the semantic/API/UI corpus, durable evidence, and exact 5,694-symbol/600-second/180,000-frame E2 manifest with recomputed event counts and digests. Comparable baseline CPU/RSS remain explicitly unknown. |
| Capability A — canonical state and hydration | `finally_accepted` | A1/A2 proofs, verification, corrections, and Capability A final review are clean; accepted dependency for Capability B. |
| `LBR-B3` removal slice | `accepted` | Commit `f697288`; removal proof, resource evidence, focused corrections, and final focused re-review are clean. |
| Capability B — evaluation and publication | `finally_accepted` | B1/B2/B3 accepted; required final read-only review returned `CLEAN/PASS`. Next permitted ticket is C1, which is inactive. |
| Capability C — selected-row T/Q | `lbr_c2_active` | `LBR-C1` is accepted. The C2 pre-assignment audit traced the engine command/pressure owners through Runtime, Massive write/normalization/queue seams, immutable publication/API capture, and the ordinary `DefaultConfig`/CLI constructors. It corrected `B` to the greatest admitted complete raw-frame sequence rather than raw read attempts, excluded diagnostic-only metrics from pressure input/validation, and allocated exact cadence/occupancy/age ties plus rejected-frame, mixed-frame, status, failure, loss, and restoration counterexamples to the primary proof. Implement only [`LBR-C2`](tickets/lbr-c2-cadence-membership-and-trust.md). |
| Capability D — live ingress | `not_started` | Requires accepted state/TQ input interface |
| Capability E — integration/removal/acceptance | `not_started` | Requires Capabilities A-D accepted |
| E2 deterministic duration/manifest revision | `owner_approved` | Exactly one 10-minute run; 5,694 symbols, 300 frames/s, 600 polls/samples, 180,000 frames, recomputed counts/digests, 15-minute command timeout, and no repeat composition. |
| Deterministic replacement | `not_started` | Requires `LBR-E1` and `LBR-E2` |
| Live stability confirmation | `not_authorized` | `LBR-E3` requires separate exact-date authorization or owner execution |

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

`deterministically_complete` requires:

- approved parent/program and accepted focused specs;
- Capabilities A-D and `LBR-E1` accepted;
- the supported scanner executing one replacement state/handoff path;
- superseded product feature state and old fallback implementations removed;
- API v2/UI/launcher compatibility proven;
- ordinary/race/vet/diff verification green as allocated;
- `LBR-E2` hard resource/stability acceptance, with every target measured and
  deviations recorded; and
- a clean final integrated read-only review.

`live_stability_confirmed` additionally requires `LBR-E3` and a clean focused
review of its bounded evidence.

Neither state establishes replay/checkpoint support, public deployment,
provider SLA, or trading expectancy.

## 17. Owner decisions and latest responsible gates

| Decision | Recommended default | Needed by |
| --- | --- | --- |
| Replay/checkpoint source disposition | Remove from supported live binary/core; decide delete versus unsupported tooling | Before `LBR-E1` |
| Market-hours execution | One exact-date owner-run or separately authorized observation | Before `LBR-E3` |
