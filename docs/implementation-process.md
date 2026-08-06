# Implementation process

**Status:** Owner-approved operational process.

**Approved:** 2026-08-05

**Revised:** 2026-08-06

**Scope:** The controlling operational process for component research,
specification, predecessor reuse, proof allocation, implementation assignment,
implementation, integration, and final validation.

This document controls how work advances after Phase 1. It does not redefine
scanner behavior. The owner-approved product and architecture contracts remain
authoritative; if this process or a focused component contract appears to
conflict with them, stop and resolve the conflict with the owner.

## 1. Contract-first rules

Before drafting a focused component contract, the assigned agent must:

1. read `AGENTS.md`, `README.md`, `docs/specification-map.md`, every relevant
   approved Phase 1 authority, and approved dependency specifications;
2. enumerate the exact Phase 1 requirement IDs governing the component;
3. state the component's ownership and explicit non-scope;
4. prefer the simplest design that satisfies all cited requirements;
5. introduce no new product rule, competing state owner, watermark, evaluator,
   T/Q-to-ranking dependency, changed time-window meaning, speculative edge
   case, or duplicated responsibility;
6. choose the initial single-file or modular layout and create a short Phase 1
   contract skeleton without inspecting version 2; the skeleton states the
   outcome, controlling IDs, ownership/non-scope, settled semantic boundary and
   invariants, unresolved evidence questions, proposed version 2
   reconnaissance scope, and the proposed document map when modular;
7. stop for owner boundary approval before opening version 2 code, tests, or
   fixtures;
8. after approval, inspect only the approved relevant version 2 sources and use
   the findings to complete the focused component contract;
9. confirm the approved document layout still fits the reconnaissance findings,
   stopping for owner review before changing its map or evidence routing;
10. define one bounded implementation slice or a sequential slice plan that
   assigns every requirement and primary proof exactly once; and
11. stop for owner contract/reuse/test/slice-plan approval before an
    implementation assignment is prepared or authorized.

Phase 1 already controls the shared domain through the product goals, system
overview, data/time/event contract, and `ScannerStateEngine` lifecycle. A
focused contract cites their exact requirement IDs and defines only its own
ownership, interfaces, behavior, failure and terminal outcomes, accounting,
evidence, and primary proof. Do not create a shared core-domain spec, repeat a
Phase 1 contract in new words, or add a contract layer merely to rename an
existing interface.

Simplicity is a first-class acceptance criterion:

- minimize mutable owners, state representations, and handoffs;
- add no abstraction that has only a hypothetical use;
- add no database, service split, generic event bus, plugin system, or
  generalized framework unless an approved requirement needs it;
- add no edge-case machinery without component-local evidence;
- reuse behavior or code only when it is narrower and safer than implementing
  the approved contract directly; and
- prefer one understandable complete data path over isolated infrastructure.

### 1.1 Focused contract layout and context routing

A focused component contract may use either of two document layouts:

1. a single specification file for a contract that remains efficient to load,
   edit, review, and implement; or
2. one compact parent specification plus cohesive subordinate detail specs for
   a contract whose concerns can be routed independently.

Within the single-file layout, a cohesive one-slice component with small
evidence, trust, and proof ledgers may use the compact grouped-section form
defined by the mandatory template. Compact form changes presentation only: it
keeps both approval stages, exact requirements, evidence/reuse decisions,
trust-boundary claims, proof allocation, implementation assignment, acceptance,
and drift audit.

Document decomposition is not implementation slicing. Decomposition reduces
the normative context needed for a particular contract task. Slicing divides
implementation into sequential observable behaviors and primary proofs. A
component may need either, both, or neither.

Use the modular layout when the detailed contract has multiple stable semantic,
trust, proof, or delivery concerns and ordinary work repeatedly requires loading
large unrelated sections. Context size and edit latency are valid reasons to
stop extending a monolith once cohesive boundaries exist. Do not split at
arbitrary line, heading, or token counts, and do not create tiny fragments that
force every task to reopen the whole set.

File size is a routing alarm, not a semantic shard boundary. Reassess the
document map before approval when a mandatory parent is likely to exceed about
2,500 words or any one normative detail is likely to exceed about 5,000 words.
Exceeding those targets is permitted only when the material is one genuinely
inseparable concern and the approval record explains why every routed task
needs it. Otherwise split at the smallest real semantic, trust, proof, or
delivery boundaries. Never create `part-1`/`part-2` files or divide a document
only to satisfy a word target.

A component with multiple implementation slices or large reconnaissance,
trust-boundary, edge-case, or proof ledgers should normally use the modular
layout because those concerns can be routed to distinct drafting and
implementation tasks. Keep it single-file only when those details are compact
and repeatedly need to be reasoned about together.

The parent remains at the path recorded in `specification-map.md` and is the
only entry point a caller must know. Subordinate specs live under a directory
with the parent's filename stem. The parent contains:

- component status and both owner-approval records;
- one authoritative delivery-state ledger containing advancement mode, slice
  state, evidence/review result, and current component state;
- the high-level outcome and user/operator consequence;
- the single ownership boundary, dependencies, and explicit non-scope;
- the cross-cutting Phase 1 semantic invariants;
- a normative document map stating what each subordinate spec exclusively
  controls, its requirement/proof/slice coverage, when it must be read, and its
  dependencies; and
- compact contract-wide coverage and acceptance state, linked to detailed
  ledgers rather than duplicated from them.

Each subordinate spec owns one cohesive detail boundary and begins by naming
its parent, normative responsibility, controlling requirement IDs, component
requirements, declared document dependencies, and allocated slices. Detail
specs may live directly under the component directory or in nested concern
directories when that makes a real subconcern independently routable. Prefer a
shallow layout; directory depth does not create authority or approval gates,
and every nested file remains listed explicitly in the parent document map. A
detail spec is part of the parent's component contract. It is not a new
component, another architecture layer, an independently approved authority, or
a second owner of cross-cutting state.

Across the complete set, every normative decision, component requirement,
evidence/reuse decision, primary proof, and implementation-slice allocation has
one authoritative home. Parent summaries are navigational and link to that
home. Cross-document dependencies must be explicit and acyclic; a detail spec
may not silently reinterpret its parent or another detail spec. Missing routes,
duplicate normative ownership, broken links, or contradictory meanings block
approval.

Mutable delivery status has exactly one authoritative home: the parent ledger.
Subordinate specs name their allocated slices and link to that ledger but do not
copy slice state, acceptance dates, proof results, reviewer results, or next-
step authorization. The specification map records only coarse component
sequence and completion; update it mechanically when a component completes,
not after every slice.

Context loading follows the parent map. For localized drafting, review, or
implementation, read the parent, the routed detail specs, and their declared
dependencies; do not load unrelated subordinate specs by default. Read the
complete manifest-listed set for boundary approval, completed-contract
approval, cross-cutting changes, and final component review. An implementation
assignment names the exact detail specs needed for its slice. Adding, removing,
or changing a subordinate spec's normative responsibility after approval is a
component-contract change and requires explicit owner review.

An already approved single-file contract may be modularized through a bounded,
owner-authorized documentation migration without reopening its substantive
decisions. Before moving text, approve the proposed parent map and an exact
section-to-document move plan. During the migration, preserve normative meaning
and do not change requirements, reuse decisions, evidence, proofs, slices, or
approval claims. The original monolith remains authoritative until the complete
new set has no missing or duplicate normative ownership, all links resolve, the
coverage ledgers reconcile, and the owner accepts the migration. If a
substantive correction is discovered, separate it from the layout migration and
use the normal contract-revision gate.

## 2. Repeated component workflow

Every component follows the same approval and delivery path:

```text
approved Phase 1 requirement(s)
  -> short Phase 1 contract skeleton or indexed skeleton set drafted without version 2
  -> owner boundary and reconnaissance-scope approval
  -> narrow version 2 code/test/fixture reconnaissance
  -> completed focused contract or indexed contract set and reuse/proof assessment
  -> owner contract/reuse/test/slice-plan and advancement-mode approval
  -> begin one bounded implementation slice
  -> implement slice and run its allocated primary proof
  -> record delegated acceptance and continue, or stop on a manual/failed gate
  -> final component review
  -> record delegated final conformance, or stop on a failed gate
  -> vertical integration milestone
  -> final operations, shadow, and cutover validation
```

An approval applies only to the reviewed artifact and scope. Boundary approval
confirms the Phase 1 constraints and permits only the named version 2
reconnaissance; it does not approve detailed behavior, reuse, or implementation.
Contract/reuse/test/slice-plan approval fixes the completed component contract,
exact reuse whitelist, proof plan, and sequential delivery boundaries. It does
not permit concurrent slices or changes to their boundaries. Unless the owner
marks a component or slice `manual`, that approval conditionally pre-authorizes
the approved slices in sequence: only one is active, and the next begins only
after the current slice satisfies the delegated acceptance gate in Section 2.6.
Manual gates still wait for an explicit owner decision.

### 2.1 Targeted research and evidence

Research answers only unresolved questions needed by the current component.
Record the question, why the approved contracts do not already settle it, the
evidence, and the resulting constraint or remaining uncertainty in that
component's spec.

Allowed evidence classes are:

- provider documentation;
- existing version 2 JSON or fake-provider fixtures;
- controlled live observations explicitly authorized for that task;
- product or mathematical invariants;
- demonstrated predecessor regressions; and
- explicit owner decisions.

Evidence is component-local at this stage. Do not build an up-front global
evidence registry, broad predecessor catalog, or speculative edge-case list.
Evidence may be an embedded payload fixture or diagnostic/review record; it is
not assumed to be a raw WebSocket recording. A global evidence/test index may
later be assembled mechanically from approved component specs if it becomes
useful.

Before boundary approval, use the Phase 1 authorities and non-predecessor
evidence. Version 2 code, tests, and fixtures are component-local evidence, but
they may be opened only during the approved reconnaissance pass. This keeps the
skeleton independent of predecessor structure without forcing the detailed
contract to ignore working, live-informed implementation evidence.

Live provider calls and credential access require explicit owner authorization
for the current task. Without it, use documentation, existing fixtures, and
offline evidence only.

### 2.2 Phase 1 contract skeleton and boundary approval

Use the
[`focused component specification template`](specifications/focused-component-spec-template.md).
The initial artifact is a short contract skeleton, not a completed component
contract. It may be one file or an indexed skeleton set under Section 1.1,
but its parent remains sufficient to understand and route the boundary. Across
the skeleton set it contains:

- the outcome and user/operator consequence;
- exact controlling Phase 1 requirement IDs;
- component ownership, dependencies, scope, and explicit non-scope;
- the settled semantic input/output boundary and invariants that version 2
  cannot change;
- unresolved behavior, provider, capacity, or proof questions;
- the proposed categories of version 2 code, tests, and fixtures to inspect and
  the question each category should answer; and
- an initial view of the likely primary proof boundaries.

Owner boundary approval confirms that this is the correct Phase 1-derived
component boundary and approves the proposed reconnaissance scope. It is not
approval of detailed behavior or reuse. Until it passes, do not inspect version
2 code, tests, or fixtures.

### 2.3 Narrow version 2 reconnaissance and reuse assessment

The only default predecessor is:

```text
/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2
```

The older version 1 checkout is out of scope unless the owner later authorizes
a precise exception. There is no broad predecessor harvest.

After boundary approval, inspect only code, tests, and fixtures within the
approved reconnaissance scope. Locate exact relevant files/functions and add a
reuse assessment to its one authoritative location in the focused contract for
each candidate containing:

- source path and relevant function/type/fixture;
- commit and file hash when provenance matters;
- decision: `direct port`, `adapt`, `behavior evidence`, or `reject`;
- behavior preserved because it satisfies the new contract;
- predecessor coupling or ownership to remove; and
- proof required for the reused behavior.

Version 2 is evidence, never authority. The owner approves the exact reuse and
fixture whitelist before implementation. Approved per-component decisions may
later accumulate into a repository-wide reuse index; unreviewed candidates do
not.

The reconnaissance should recover useful implementation knowledge rather than
only search for copyable code. It records validated algorithms, provider
behavior, regression cases, bounded-state techniques, existing proof quality,
and places where old orchestration or ownership must be removed. If findings
would change an approved Phase 1 boundary or expand the reconnaissance scope,
stop for owner review instead of allowing version 2 to drive the contract.

### 2.4 Completed contract, proof allocation, and owner approval

After reconnaissance, complete the focused component contract. In a modular
layout, keep the parent high-level and put each detailed concern in the one
subordinate spec assigned by its document map. Across the set, define semantic
inputs/outputs and owned state, required behavior, failure and terminal
outcomes, accounting/observability, boundedness, evidenced edge cases,
implementation discretion, acceptance criteria, and—when needed—a sequential
implementation-slice plan. Use version 2 findings where they provide stronger
implementation or behavior evidence, while keeping every normative decision
subordinate to the cited Phase 1 requirements.

Proof planning occurs in the focused contract before implementation. Each
requirement receives one primary proof. A second test at another layer is
allowed only when it proves a distinct boundary, and the contract names that
boundary. Do not multiply the same scenario across unit, component,
integration, replay, and live tests.

One coherent test, trace, fixture, or construction inspection may serve as the
primary proof for multiple tightly coupled requirements. It must name and
assert each requirement's distinct claim, dangerous counterexample, observable
result, and limitation so a failure remains diagnosable. Do not create separate
test functions merely to mirror requirement IDs, and do not use proof sharing
to hide an unexercised branch or distinct implementation path.

Passing tests is necessary but not sufficient evidence of contract
conformance. Before allocating proofs, make each consequential external,
persisted, or cross-component trust boundary explicit:

- the exact evidence accepted into the success path;
- malformed, stale, contradictory, or otherwise invalid evidence that is
  rejected;
- whether failure is global, symbol-/field-local, or persistence-only;
- the most dangerous false-success case: invalid evidence that could otherwise
  look complete, current, or valid; and
- important invalid states prevented by API, type, ownership, or construction
  rather than by runtime tests.

Do this only for the component's real trust and ownership boundaries. Terms
such as `valid`, `strict`, `complete`, `current`, `immutable`, and `atomic` are
not proof plans by themselves; define the concrete conditions that matter to
the component. Do not expand the spec with a generic repository-wide mutation
matrix.

For every primary proof, record:

1. the exact claim;
2. the dangerous counterexample or boundary condition it exercises;
3. the observable result that distinguishes conformance from false success;
4. any distinct implementation paths participating in the claim; and
5. what the proof intentionally does not establish.

The proof form remains component-specific. Deterministic mathematics usually
needs exact boundary or permutation tests; provider normalization needs
documented fixtures including ambiguous/failure envelopes; cache and
filesystem behavior needs focused fault injection and path-safety inspection;
identity needs golden encoding plus semantic mutation/permutation cases;
ordered state needs event traces with invariants checked after each input; and
lifecycle/recovery needs progress, terminal-disposition, containment, and
accounting scenarios. Select only the forms justified by the component.

Use the narrowest applicable proof form:

| Proof form | What it establishes |
| --- | --- |
| Construction/ownership proof | API, type, visibility, and ownership design make an important invalid state or competing mutation path unavailable. |
| Formula test | Exact deterministic mathematics, boundaries, invalid-input behavior, and tie/order rules. |
| Provider-normalization fixture | Documented or captured payload-to-normalized-fact mapping and bounded rejection. |
| Focused fault-injection scenario | Failure at a named I/O or persistence stage produces the contracted containment and terminal result. |
| Focused lifecycle scenario | State transition, progress/exit event, failure containment, or terminal accounting. |
| Deterministic aggregate replay | Shared normalized aggregate/state/evaluator path and market-time invariance across playback speeds. |
| Checkpoint/restart equivalence | Coherent `T0` projection plus catch-up reproduces uninterrupted aggregate-derived state. |
| Differential replay | New and approved reference behavior agree on a controlled corpus; disagreement is investigated rather than hidden. |
| Authorized shadow observation | Production-shaped transport, timing, capacity, and operational behavior under explicit live-data authorization. |
| Cutover evidence | Operational readiness, rollback, observability, and acceptance criteria in the intended deployment. |

Speculative edge cases require provider documentation, an existing fixture, a
controlled observation, an invariant, a demonstrated regression, or explicit
owner approval. Agreement proves scanner correctness only. It is not evidence
of predictive trading edge, entries/exits, slippage, or executable expectancy.

Before owner approval, begin with the presumption that the component is one
bounded assignment. Add each further slice only by naming the independently
observable behavior, distinct proof family, provider-I/O/canonical-mutation
boundary, ownership/package boundary, dependency order, or reviewability
problem that prevents one coherent assignment. A different helper, file, or
requirement ID is not by itself a reason to split. Do not split by estimated
hours, lines, or token count. Split at behavior and proof boundaries, and do not
force one slice when the named boundaries would make review or rollback
ambiguous.

Each slice must:

- produce one coherent observable behavior or tightly related behavior set;
- own an exact subset of component requirements and their primary proofs;
- leave the repository buildable with its allocated proofs passing;
- preserve the approved ownership and semantic boundaries;
- avoid unused scaffolding or a temporary competing state path;
- state deferred behavior honestly; and
- be reviewable before the next slice begins.

Slice acceptance is provisional evidence that the approved next slice may
proceed; final component review remains the complete conformance decision. At
the slice gate, passing allocated proofs are accompanied by a compact
conformance walkthrough stating:

- which important invalid states are prevented by construction;
- what external or persisted evidence can reach the success return;
- how the main failure paths are contained and terminally reported;
- which distinct implementations participate in a shared requirement and how
  each was proved; and
- any material claim still supported only by inspection or intentionally left
  unproved.

This is a short review artifact, not a second specification or broad test
inventory. Its result and evidence references are written once to the parent
delivery ledger. Subordinate detail specs retain normative allocation only and
do not receive acceptance-status edits.

Every component requirement and primary proof belongs to exactly one slice.
If a requirement concerns interaction among slices, allocate its primary proof
to the last slice needed to make that interaction real. A later milestone may
add proof only for a distinct cross-component boundary. Owner
contract/reuse/test/slice-plan approval fixes the complete component contract,
predecessor whitelist, approved fixtures, primary proofs, slice ordering, and
any intentionally deferred secondary evidence. It also fixes
`advancement_mode` as `delegated` by default or `manual` for the whole component
or named slices.

A focused component contract that has not yet received that approval must
conform to the current mandatory template, including its document-map rules
when modular, before approval even if its draft began under an earlier template
revision. Do not silently retrofit or repartition an already approved component
contract; revise it only through explicit owner review.

### 2.5 Bounded implementation-slice assignment

One implementation assignment covers exactly one approved slice and must
contain:

1. authoritative documents and exact controlling requirement IDs;
2. the slice identifier, coherent outcome, and exact requirements assigned;
3. approved component scope, slice scope, and explicit non-scope;
4. approved dependencies, semantic interfaces, and ownership boundary;
5. allowed files/packages or a precise code-ownership boundary;
6. approved version 2 source and fixture whitelist;
7. approved fixtures and other evidence inputs;
8. the primary proof for each assigned requirement;
9. required verification and compact acceptance record;
10. behavior explicitly deferred to later slices;
11. routine decisions delegated to the implementer;
12. prohibited changes; and
13. stop/escalation conditions; and
14. the component-specific trust-boundary counterexamples, construction
    guarantees, and proof limitations allocated to the slice; and
15. the inherited `advancement_mode` and exact conditions that force a manual
    owner decision.

The assignment must be implementable from a compact context bundle:

- `AGENTS.md`;
- the approved parent focused component contract;
- only the subordinate detail specs routed to the slice, plus their declared
  dependencies;
- exact cited Phase 1 sections and requirement IDs;
- approved dependency specs or interface excerpts;
- approved version 2 sources/fixtures; and
- the one approved slice, its required verification, and its deferred behavior.

The agent should not need the full project discussion history. Implementation
discretion includes private helper names, exact internal file layout, and
routine algorithms unless the approved contract makes one necessary for
correctness. Stop if implementation exposes a missing contract, requires an
unapproved dependency or predecessor file, or would change an owner-approved
behavior. If the assignment no longer has one coherent outcome and reviewable
proof, stop and split it rather than allowing the task to expand.

### 2.6 Slice implementation, delegated acceptance, and component review

Implement only the assignment's authorized boundary and primary proofs. Keep
provider adapters fact-returning, canonical mutation inside the sole engine
owner, and concurrent consumers on immutable views. Preserve bounded queues,
retention, diagnostics, and failure outcomes specified for the component.

At the end of every slice, produce a compact acceptance record containing:

- the coherent behavior now available;
- exact requirements completed and behavior still deferred;
- files, ownership, or interfaces changed;
- the approved proof design, result, and what it establishes;
- deviations, failed assumptions, or newly discovered evidence;
- the compact construction and success/failure-path conformance walkthrough;
- proof limitations and claims supported only by inspection;
- whether the slice freezes a consequential trust, persistence, identity,
  ownership, ordering, concurrency, or cross-component interface boundary;
- whether the next approved slice remains valid; and
- the delegated-gate result: accepted and continue, or stopped for the smallest
  manual decision.

An independent slice review is not mandatory for every slice. Request one
before dependent work proceeds when the slice freezes a consequential external
trust, persistence/atomicity, market-identity/currentness, global-versus-local
failure, sole-mutation/ordering, concurrency, or cross-component interface
boundary and the allocated proof plus construction argument does not make the
risk straightforward to assess. The review should target that boundary rather
than repeat the full component review. A read-only independent final component
review remains required after all slices.

When an independent review is delegated to an agent, follow the trigger,
model/effort, correction, and focused re-review policy in `AGENTS.md`.

Under `advancement_mode: delegated`, mark the slice accepted in the parent
ledger and continue to the next already-approved slice without requesting an
owner response only when:

1. every allocated proof and required verification passes;
2. each required independent review is clean after focused correction;
3. scope, dependencies, interfaces, document map, and predecessor whitelist are
   unchanged;
4. the conformance walkthrough contains no deviation, failed assumption,
   unresolved success-invalidating inspection claim, or drift-audit `yes`; and
5. the next slice remains exactly valid as approved.

Send a concise progress update when advancing; it is informational and requires
no reply. Under `manual` mode, or when any condition above fails, record the
evidence without marking acceptance and stop for the smallest owner decision.
A failing proof or implementation discovery may require revising the component
contract or remaining slice plan; delegated advancement never authorizes that
change inside the current assignment.

After all slices pass, final component review reads the complete
manifest-listed contract set and verifies:

- conformance to every cited requirement and approved interface;
- passage and diagnostic quality of the allocated primary proofs;
- exact accounting and terminal outcomes where applicable;
- absence of unapproved predecessor coupling or scope expansion;
- bounded memory, queues, retries, cardinality, and diagnostics; and
- whether the complete local data path is simpler and clearer than the
  alternatives actually considered.

If that mandatory independent review is clean, complete verification passes,
and no manual-decision condition is present, delegated mode marks final
component conformance accepted in the parent ledger. Update the specification
map's coarse component status mechanically and continue only as permitted by
the approved sequence and milestone dependencies. No separate owner acceptance
message is required. A failed or ambiguous final review stops without claiming
completion.

A completed component is not integrated merely because its slices pass in
isolation. It advances when final component review and its vertical milestone
demonstrate the shared boundary with real adjacent components or deterministic
substitutes.

### 2.7 Verification cadence

Verification is cumulative at acceptance boundaries, not maximally broad after
every edit. The approved component contract and slice assignment may require a
stronger check when their risk demands it; otherwise use these tiers:

1. **Development loop:** run the allocated primary proof under active
   development and the smallest affected package or fixture checks. Do not
   repeatedly run unrelated long proofs while editing.
2. **Slice acceptance:** run every primary proof allocated to the slice, the
   affected package tests, a repository compile/ordinary-test check, and only
   earlier proofs whose owned state or interface the slice can affect. Run race
   detection when the slice changes concurrency, queues, synchronization,
   mutable ownership, publication, or reader isolation. Run other expensive
   static, fault, or long-path checks only when allocated by the contract or
   implicated by the change.
3. **Focused correction re-review:** rerun the failed proof, the proof for the
   affected boundary, and direct regressions. The component-final tier still
   follows; a local correction does not automatically restart every clean
   independent review or long proof.
4. **Final component review:** rerun the complete component proof ledger,
   repository build/test and static checks, required race checks, and the
   read-only conformance review across the complete manifest-listed contract.
5. **Vertical milestone and release:** run the named cross-component path,
   milestone failure cases, and final-validation evidence. Reuse reviewed
   lower-level results; duplicate them only when the milestone establishes a
   distinct integration boundary.

Long-running, live, credentialed, destructive, or environment-sensitive checks
remain separately authorized where required. Verification commands and results
must state what claim they establish; command volume is not evidence quality.

## 3. Focused component sequence

The
[`specification map`](specification-map.md#phase-2-focused-component-sequence)
is the single sequence and dependency index for the eleven real component
contracts. Each map entry points to one authoritative parent; subordinate specs
do not create extra sequence entries. This process does not maintain a second
list. The map covers reference
data/session binding, engine/canonical-state implementation details, aggregate
evaluation, aggregate replay, Massive live input, REST hydration/recovery,
checkpoints, readiness/operations, T/Q, the versioned API, and the independent
UI.

That sequence does not create a shared interface meta-spec: normalized
identity, clocks, merge rules, the authoritative engine/state owner, lifecycle,
publication meanings, and failure boundaries are already controlled by Phase
1.

Implementation follows the map sequentially. Keep one active implementation
slice. A component with several slices completes them in approved order, with a
recorded delegated or manual acceptance gate between slices. While component N
is being implemented, the short Phase 1 skeleton for component N+1 may be
drafted and receive owner boundary/reconnaissance-scope approval. That approval
records the future
reconnaissance boundary; it does not permit opening version 2 or drafting N+1
Sections 8–19 while N remains incomplete. Begin N+1 reconnaissance and detailed
contract work only after all N slices have passed their allocated primary
proofs and final component review. N+1 completed-contract approval and
implementation also wait for any milestone boundary that the map names as a
dependency. The owner may approve earlier work only when it depends exclusively
on an already approved stable interface and the exception records exactly which
reconnaissance, detailed-contract, or approval gate may advance.

This skeleton-only one-component lookahead overlaps the low-rework Phase 1
boundary work without paying for predecessor reconnaissance or detailed design
against an implementation that may still change. Keep the skeleton compact:
outcome, controlling IDs, ownership/non-scope, settled invariants, unresolved
evidence questions, proposed reconnaissance scope, proposed document map, and
likely proof boundaries only. Do not add helper names, algorithms, final data
structures, detailed proof matrices, or a committed slice count. If component N
reveals that an approved N+1 skeleton assumption is wrong, revise the skeleton
before reconnaissance; do not reopen completed component N unless an
authoritative requirement was missed.

Live-provider observation is not an up-front phase. The Massive live-adapter
and T/Q component skeletons must first name the exact question and approved
version 2 evidence. A controlled live observation may then be authorized only
when current provider behavior, entitlement, or timing remains unproved. The
aggregate adapter observation belongs with component 5; T/Q subscription and
coverage observation belongs with component 9.

## 4. Vertical integration milestones

Integration proceeds through complete data paths:

1. **Deterministic aggregate core:** replay source -> normalized aggregate ->
   `ScannerStateEngine` -> canonical symbol state -> qualification/ranking ->
   immutable snapshot. This proves deterministic market-time scanner behavior,
   not live transport.
2. **Production aggregate lifecycle:** live adapter + REST hydration/recovery +
   checkpoints + readiness around the same aggregate core. This proves fresh
   start, restart, live tail, gap recovery, failure containment, and supported
   currentness.
3. **T/Q enrichment:** qualified top-20 intent -> acknowledged subscription and
   coverage -> Tape Rate/Spread -> aggregate-protecting pressure degradation.
   This proves normal displayed-row enrichment and that complete T/Q shedding
   leaves aggregate state, watermark, ranking, and readiness unchanged.
4. **Product delivery and release validation:** versioned API + independent UI
   + release/cutover evidence. This proves read-only product meaning,
   independent deployment, and shutdown/rollback/cutover behavior against the
   readiness and operations policy already established in milestone 2.

Each milestone names its entry contracts, deterministic fixtures or authorized
environment, exact assertions, failure cases, and unresolved limitations. A
later milestone does not duplicate lower-layer proofs unless it establishes a
new cross-component boundary.

## 5. Final validation and release decision

Final validation assembles approved component proofs; it does not invent new
requirements or an after-the-fact global test strategy.

1. Re-run formula, fixture, lifecycle, replay, and checkpoint/restart primary
   proofs in their approved configurations.
2. Verify each vertical milestone, exact symbol/work accounting, stable API
   meaning, UI independence, and bounded operational diagnostics.
3. When explicitly authorized, run shadow observation to measure live
   transport behavior, latency/currentness thresholds, resource pressure,
   recovery, T/Q shedding/restoration, and checkpoint objectives without
   claiming trading performance.
4. Review cutover criteria, operational ownership, rollback, last-known-good
   checkpoint behavior, monitoring reasons, and any known limitation.
5. The owner approves cutover only when every required proof has a reviewed
   result and no unresolved item can invalidate the intended production claim.

Shadow agreement, differential agreement, and correct scanner output establish
implementation correctness within their evidence limits. They do not validate
forecast value or executable expectancy.

## 6. Drift audit

Run this audit at boundary review, after the completed contract/reuse
assessment, after every implementation slice, at final component review, and
before each integration milestone. Ask whether the work introduced:

- a new product rule;
- another mutable state owner, watermark, or evaluator;
- aggregate dependence on T/Q health, data, or availability;
- changed session, event-time, half-open-window, correction, or watermark
  semantics;
- behavior without component-local evidence or owner approval;
- duplicated ownership or responsibility;
- machinery not required by an approved contract; or
- version 2 behavior that changed the new contract rather than serving it.

Any substantive **yes** stops advancement and requires explicit owner review.
The resolution belongs in the applicable focused component contract if it changes
that component contract, or in this document/`AGENTS.md` if it is a
repository-wide mechanical convention. Routine implementation choices remain
delegated; this repository does not use an ADR workflow.
