# Implementation process

**Status:** Owner-approved operational process.

**Approved:** 2026-08-05

**Revised:** 2026-08-23 for the live backend replacement program and its exact
one-run 10-minute E2 acceptance boundary

**Scope:** Capability/component research, contracts, predecessor reuse, proof
allocation, implementation assignments, correction, acceptance, integration,
and private/local release validation.

This process does not redefine scanner behavior. The product and architecture
contracts remain authoritative. The current
[`live backend replacement delivery program`](live-backend-replacement/delivery-program.md)
supplies the active capability sequence, delivery scope, resource-target
policy, and completion boundary. The former live-feature MVP and Version 1
program remain historical accepted evidence for their scopes.

## 1. Authority and correction model

The operational rule is:

```text
fixed product/architecture meaning
  -> current capability/component plan
  -> implementation and evidence
  -> accept, or diagnose and revise the lowest unsuitable artifact
  -> continue in the current program order
```

A lower-level contract, fixture, test, benchmark, whitelist, slice, interface,
review route, or implementation does not become coequal with Phase 1 because it
was previously approved. If evidence exposes a mistake, revise it. An accepted
slice/capability/component may be reopened before final program acceptance.

For C7-C11 there is no component-local owner-response state. All correctable
failures use the program's required correction loop; excluded external/live
work, user/Git overlap, and tool/reviewer availability use its containment and
fallback rules.

Outside the V1 program, the applicable owner approval rules recorded in that
component remain in force.

## 2. Contract-first component planning

Before implementation, the component plan must:

1. read `AGENTS.md`, `README.md`, the specification map, the current delivery
   program, every relevant Phase 1 authority, and accepted dependency
   contracts;
2. enumerate exact controlling `PG-*`, `ARCH-*`, `DTE-*`, and `LIFE-*` IDs;
3. state one ownership boundary, inputs/outputs, dependencies, and explicit
   non-scope;
4. choose the simplest design satisfying those authorities;
5. introduce no new product rule, state owner, watermark, evaluator,
   T/Q-to-ranking dependency, time-window meaning, speculative edge case, or
   duplicated responsibility;
6. record a short Phase 1 boundary and exact V2 reconnaissance scope before
   opening V2 code, tests, or fixtures;
7. complete the detailed contract after reconnaissance, including trust
   boundaries, requirements, one primary proof per requirement, verification
   tiers, sequential slices, review triggers, implementation discretion,
   deferrals, and correction conditions; and
8. record the completed contract in the parent ledger as the current executable
   plan.

C8-C11 already have owner-approved boundary/reconnaissance plans at the parent
paths in the specification map. Their detailed contracts are completed just in
time after the preceding component's interface is finally accepted. Under the
V1 program the orchestrator may revise and record the boundary or completed
contract without another owner message. If a completed contract introduces or
changes a consequential trust, persistence, identity, concurrency, ownership,
ordering, or cross-component interface boundary, it receives one focused
independent review before implementation. Other completed contracts do not
need a ceremonial independent review.

### 2.1 Layout and routing

Use one compact file when the component has one cohesive ownership boundary,
small evidence/trust ledgers, and at most two readily reviewable slices. Use a
compact indexed parent plus subordinate details only when distinct semantic,
trust, proof, or delivery concerns would otherwise force unrelated context into
every task.

The parent remains at the specification-map path and owns:

- outcome, single boundary, dependencies, and non-scope;
- cross-cutting Phase 1/V1 invariants;
- current contract state and sole mutable delivery ledger; and
- a normative document map routing every requirement, proof, decision, and
  slice exactly once.

Each detail owns one cohesive boundary, declares parent/dependencies/
requirements/slices, and copies no mutable status. Cross-document dependencies
are explicit and acyclic. A localized task reads the parent and routed details;
final component review reads the complete set.

Within C7-C11, layout and document-map responsibility can be revised through
the V1 correction loop. Record the before/after map and ensure no missing,
duplicate, or contradictory normative ownership. Outside the program, use the
applicable owner gate.

### 2.2 Version 2 reconnaissance and reuse

The only default predecessor is:

```text
/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2
```

Version 2 is evidence, never authority. The older Version 1 checkout is out of
scope without a precise owner exception.

Before inspecting V2, record the source category, question, and exclusion.
After inspection, record for each candidate:

- exact path and function/type/test/fixture;
- commit/file hash when provenance matters;
- `direct port`, `adapt`, `behavior evidence`, or `reject`;
- behavior preserved;
- predecessor ownership/coupling removed; and
- required proof.

For C7-C11, the whitelist may be revised when new evidence remains inside the
fixed component boundary. Record the expansion before opening the new source,
then update its proof obligation. Apparent broader product/architecture tension
uses the V1 program's authority-order and strict-compatible-interpretation rule.

Credentials and live calls always require a separate explicit owner
authorization.

## 3. Detailed contract and proof allocation

The completed contract defines:

- semantic inputs, outputs, owned state, and immutable/fact-returning
  boundaries;
- exact required behavior, failure/terminal outcomes, accounting,
  observability, and bounds;
- evidenced edge cases and explicitly deferred behavior;
- implementation discretion versus fixed Phase 1 meaning;
- the current V2/fixture whitelist;
- one primary proof per requirement; and
- one or more sequential implementation slices.

### 3.1 Trust and false-success boundary

For each consequential external, persisted, or cross-component boundary, state:

- exact evidence accepted into success;
- malformed, stale, contradictory, or invalid evidence rejected/contained;
- global, symbol/field-local, or persistence-only failure scope;
- the smallest dangerous case that could falsely appear complete, current,
  valid, or durable; and
- invalid states prevented by type/API/ownership/construction.

Do not create a generic repository-wide matrix. Cover the component's real
risks only.

### 3.2 Primary proofs

Each primary proof records:

1. the exact claim;
2. the dangerous counterexample/boundary;
3. the observable result distinguishing conformance from false success;
4. every distinct implementation path participating; and
5. what the proof intentionally does not establish.

One proof may cover tightly coupled requirements if every claim remains
diagnosable. A second layer is allowed only for a distinct boundary. Do not
duplicate the same scenario across unit, component, integration, replay, and
live tests.

Use the narrowest useful form: construction/ownership inspection, formula
test, provider fixture, focused fault injection, lifecycle trace, deterministic
replay, checkpoint/restart equivalence, controlled capacity evidence, or
separately authorized live observation.

### 3.3 Slice plan

Begin with one slice. Add another only for a distinct observable behavior,
proof family, provider/canonical boundary, ownership/package boundary, or
dependency order that would make one change difficult to review. For C8-C11,
use no more than two slices unless a recorded third boundary genuinely cannot
fit two.

Each slice:

- owns an exact subset of requirements and proofs;
- produces coherent behavior, not unused scaffolding;
- preserves one owner/state path;
- leaves the repository buildable with its allocated proof passing;
- states deferred behavior honestly; and
- is reviewable without beginning the next slice.

Every requirement/proof is allocated once. Cross-slice interaction belongs to
the last slice needed to make it real.

## 4. Implementation assignment

One assignment covers exactly one current slice and contains:

1. authoritative documents and exact requirements;
2. outcome, scope/non-scope, dependencies, interfaces, and ownership;
3. allowed files/packages or precise ownership boundary;
4. current V2 source/fixture whitelist and evidence inputs;
5. primary proof and dangerous counterexample per requirement;
6. verification tier, explicit timeout, and acceptance record;
7. deferred behavior and routine decisions delegated to the implementer;
8. prohibited Phase 1/product/ownership changes;
9. risk-triggered review requirement; and
10. the V1 zero-interruption/containment link and lower-level correction
    triggers.

The implementer should not need project discussion history. Private helpers,
file layout, ordinary algorithms, error wrapping, and test builders remain
implementation discretion unless correctness or current reuse evidence fixes
them.

If the slice becomes incoherent, update the current contract/slice allocation
through the V1 correction loop. Do not let the task expand silently or create a
temporary competing path.

## 5. Implementation, correction, and slice acceptance

Implement only the current slice. Keep blocking I/O outside the engine, inputs
bounded, provider adapters fact-returning, canonical mutation in the sole
engine, and readers on immutable output.

### 5.1 Correction cycle

When a test, benchmark, review, fixture, or implementation fails:

1. preserve the observed command, configuration, fixture manifest, and result;
2. identify the exact claim and classify the cause;
3. revise the lowest unsuitable current-program artifact or implementation;
4. mark any affected accepted slice/component `reopened` while preserving
   unrelated clean evidence;
5. run the narrowest distinguishing proof and direct regressions;
6. request focused re-review only when the correction changes the consequential
   boundary originally reviewed; and
7. update the parent ledger and continue.

Do not request owner approval for a new codec, corrected fixture, benchmark,
proof allocation, slice boundary, contract map, threshold, or replacement
implementation inside the fixed V1 boundary.

Never rerun an unchanged failed command. After two failed attempts using the
same mechanism or premise, abandon it for the simplest fixed-authority-
compliant alternative, prove that alternative with a compact deterministic
distinguishing case, and only then rebuild scale. Progress reports and local
commits are evidence checkpoints, not turn-ending conditions.

### 5.2 Slice acceptance record

Record:

- coherent behavior now available;
- exact requirements/proofs completed and behavior deferred;
- files/ownership/interfaces changed;
- proof design, result, limitation, and counterexample;
- construction guarantees and success/failure-path walkthrough;
- deviations, failed assumptions, and inspection-only claims;
- review trigger/result; and
- whether the next slice is valid or needs an in-program revision.

Under the current program, a clean evidence-based gate records `accepted` and
continues. A correctable failure records `reopened` or `correction_active` and
continues through Section 5.1. Historical C7-C11 work does not record a
component-local waiting-for-owner state.

### 5.3 Final component review

After all slices pass, one independent read-only review examines the complete
component set and verifies:

- conformance to fixed Phase 1 and accepted dependency meaning;
- passage/diagnostic quality of allocated proofs;
- exact accounting and terminal outcomes where applicable;
- no unrecorded predecessor coupling or scope expansion;
- bounded memory, queues, waits, retries, work, and diagnostics; and
- one clear ownership/data path.

Correct findings and request focused re-review. Do not repeat the broad review.
When clean, record final capability/component acceptance and update the
specification map. Later integration evidence may still reopen it before final
current-program acceptance.

## 6. Verification tiers and cost policy

### 6.1 Ordinary

The normal repository command is:

```text
go test -short -timeout 2m ./...
```

Ordinary unit/component tests use the smallest deterministic population and
event stream proving the requirement. Tests skipped by `testing.Short()` must
be explicit acceptance, capacity, benchmark, soak, race, or live-validation
commands.

### 6.2 Acceptance

Run the current slice's primary proofs, affected package tests, a repository
short compile/test check, and only earlier proofs whose owned state/interface
can be affected. Add race detection when concurrency, queues, synchronization,
mutable ownership, publication, or reader isolation changes.

Acceptance commands are explicitly selected and timed out. No single local
acceptance command exceeds 15 minutes without a component-specific recorded
reason.

### 6.3 Capacity

Use the smallest population crossing the claimed boundary. For the current
MVP, a controlled 6,000-symbol measurement runs only when ordinary evidence or
live observation suggests the revised full-universe calculation threatens
accepted responsiveness. Theoretical 100,000-symbol structural admission is a
separate bounded proof and not an operating-capacity claim.

Record OS/architecture/CPU, Go version, configuration, fixture bytes and
structure, trial deadlines/segments, throughput, processing delay, queue high-
water/growth, memory, rejected/dropped/shedded counts, and limitations. Run the
capacity proof once after the implementation is stable, not in ordinary or
repeated correction loops.

### 6.4 Deterministic fixtures and retained replay

For the live backend replacement, use compact deterministic event traces,
fake-provider inputs, API fixtures, and UI fixtures. These may prove formulas,
state transitions, and presentation while the market is closed, but they are
not product replay. Retained replay code has unknown current capability and is
not a replacement proof or acceptance gate.

### 6.5 Live validation

Live observation is separate, credentialed, and owner-authorized per execution.
Follow [`market-hours-validation.md`](market-hours-validation.md). Pending live
validation does not block deterministic replacement acceptance and does
not authorize replay as a substitute operating claim.

### 6.6 Universal bounds

- No unbounded loop, retry, channel wait, polling interval, or generator.
- Every long command and every individual performance trial has an explicit
  timeout.
- Validate symbol count, fixture bytes, interval, correction count, expected
  state families, and planned records/requests before timing.
- Separate semantic correctness from latency/capacity.
- Do not make a noisy microbenchmark a release blocker without a stable method
  and product-derived or explicitly program-selected threshold.
- Run the narrow failed proof during correction; do not run the full repository
  after every edit.
- Reuse expensive evidence when code, configuration, fixture, and proof premise
  are unchanged.
- Run cross-component/capacity verification once after affected code is stable.

## 7. Independent review cadence

Use an independent review only for:

1. one completed-contract review when a consequential trust, persistence,
   identity, concurrency, ownership, ordering, or cross-component interface
   boundary is introduced or changed;
2. a narrow implementation review when one such boundary is difficult to
   assess from primary proof and construction alone;
3. one final read-only review per completed current-program capability; and
4. one final integrated review at current-program completion.

After correction, request focused re-review of the finding and affected
boundary. Do not repeat a complete review or ask a reviewer to reconfirm an
unchanged primary proof. A review finding is correction input. Substitute an
available reviewer model when needed; temporarily unavailable review capacity
delays only that review while other work continues.

Reviewer execution/model policy is in `AGENTS.md`.

## 8. Current capability sequence and integration milestones

The specification map and replacement delivery program identify the current
sequence:

1. **`LBR-P1`:** the cleanly cross-reviewed five focused specs are accepted;
   freeze baseline characterization;
2. **Capability A:** compact canonical state and hydration;
3. **Capability B:** incremental two-phase evaluation and publication;
4. **Capability C:** compact selected-row T/Q;
5. **Capability D:** single-pass live ingress and one handoff; and
6. **Capability E:** exclusive cutover/removal, deterministic resource/stability
   acceptance, then separately authorized live confirmation.

The focused specs allocate exact slices and primary proofs. Historical replay,
checkpoint repair, and unrelated cleanup are not milestones.

## 9. Final replacement validation

Final validation assembles the accepted capability evidence, ordinary and
allocated race/vet verification, one exact API/UI composition, exactly one
10-minute mature resource/stability run, and the final integrated read-only
review. Every numeric resource target is reported; a hard-conforming miss is a
recorded deviation and does not authorize another timed trial. Hard acceptance
depends on correctness, bounded resource plateaus, sustained feed handling
without loss/backlog/readiness flapping, and usable one-second snapshot
delivery. No optional, fallback, diagnostic, host-coexistence, or non-gating
repeat run exists.

Separately authorized live observation is recorded as pending or confirmed; it
is never inferred from deterministic evidence. The result establishes neither
replay/checkpoint support, public deployment, provider SLA, nor trading edge.

## 10. Drift audit

At contract completion, after each slice/correction, final component review,
and each milestone, ask whether the work introduced:

- a new product rule;
- another state owner, watermark, evaluator, or publication authority;
- aggregate dependence on T/Q;
- changed session/event/window/correction/watermark meaning;
- fabricated completeness/readiness/availability;
- unevidenced provider behavior;
- duplicated responsibility; or
- unnecessary machinery.

If the `yes` is in a lower-level current-program artifact, correct that
artifact and continue. If the apparent tension reaches fixed Phase 1 meaning,
apply the current program's authority order, simplest compatible design, and
honest unavailable-output rule; do not silently change product meaning.

## 11. Orchestration and Git

Keep one active implementation slice and at most one write-capable worker.
Reviewers are read-only. Ledger/Git actions occur only while workers and
reviewers are quiescent.

Stage exact paths after verifying unrelated user changes are preserved. Make
local commits at coherent planning/correction milestones, accepted slices,
final capability/component acceptance, and distinct vertical milestones when
safe. If an exact-path commit would combine unrelated work or Git repair would
rewrite user history, leave a precise working-tree handoff and continue;
commits are not capability prerequisites. Never push, rebase, amend, rewrite
history, delete branches, or use destructive reset.
