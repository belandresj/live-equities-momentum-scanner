# Live backend replacement delivery program

**Status:** Owner-approved current delivery authority, 2026-08-23.

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

Run one mature approximately 5,700-symbol deterministic 30-minute mixed-feed
soak with selected-row T/Q and one-second dashboard polling. Use a fixed
manifest at approximately 300 frames/s unless characterization selects a
better documented 1.5x observed rate. Measure the parent CPU, heap, RSS,
allocation, goroutine, queue, cycle, capture, watermark, loss, and accounting
targets.

An optional generic host-coexistence observation may repeat the accepted
composition alongside any owner-chosen ordinary local workload and record
scanner/system headroom. It is diagnostic and non-gating unless it exposes a
hard scanner failure such as unbounded growth, sustained backlog/readiness
flapping, or unusable one-second polling.

The 30-minute duration is this program's explicit exception to the ordinary
15-minute local-command ceiling: the claim requires a mature heap plateau and
sustained queue slope rather than a short burst.
Validate the manifest first, keep stop conditions active, and do not extend an
individual trial beyond 30 minutes without a new recorded reason.

Every numeric target is reported. A miss invokes exactly one bounded
measurement/profile/correction/rerun cycle. If the final composition is
correct, reaches a bounded resource plateau, handles the characterized feed
without sustained backlog/loss/readiness flapping, and keeps API polling usable,
record the deviation and accept the measured result. The orchestrator must not
continue optimization only to hit a target. Block only on a hard failure or
explicit owner rejection of the measured stable result.

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

For a resource-target miss, the orchestrator owns the single bounded response
defined by `LBR-E2`. It must reject repeated profiling, speculative tuning, or
scope expansion once hard acceptance passes. A target deviation is evidence
for the ledger, not an unfinished implementation state.

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

## 13. Correction and acceptance

A failed proof, review, benchmark, or integration run records evidence and
reopens the lowest unsuitable spec/slice. Preserve unaffected evidence, revise
the smallest artifact, run the narrow distinguishing proof, and continue in
program order.

A numeric resource-target miss alone does not reopen a capability indefinitely.
After the one bounded `LBR-E2` response, accept a stable hard-conforming result
with its deviation or block on the exact hard failure. Further optimization is
a new owner-directed task.

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

## 14. Sole delivery ledger

This table is the only mutable program status. Focused specs name allocations
but copy no status.

| Item | State | Gate / next action |
| --- | --- | --- |
| Parent architecture | `approved` | Current replacement architecture |
| Delivery program | `approved` | Current replacement delivery authority |
| `LBR-P1` focused contracts and characterization | `accepted` | Five-spec independent cross-review and owner acceptance are recorded; [`baseline-characterization.md`](baseline-characterization.md), SHA-256 `3a70dda5bc040d1a77fbe3e114d788d3fc04c93fbb51ff2e5eec646f4c10e6e3`, freezes baseline `0d043c1`, the semantic/API/UI corpus, durable queue/heartbeat/hydration/cycle/memory evidence, and the exact 5,694-symbol deterministic manifest. Comparable baseline CPU/RSS remain explicitly unknown. Next: activate `LBR-A1`. |
| Capability A — canonical state and hydration | `lbr_a1_accepted_a2_next` | `LBR-A1` primary proof, affected verification, two correction rounds, and focused re-review are clean. Activate only [`LBR-A2`](tickets/lbr-a2-hydration-and-recovery-installation.md); Capability B remains inactive. |
| Capability B — evaluation and publication | `not_started` | Requires Capability A interface acceptance |
| Capability C — selected-row T/Q | `not_started` | Requires Capability B publication/selection interface |
| Capability D — live ingress | `not_started` | Requires accepted state/TQ input interface |
| Capability E — integration/removal/acceptance | `not_started` | Requires Capabilities A-D accepted |
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
