# Focused component specification template

**Status:** Owner-approved mandatory template.

**Approved:** 2026-08-05

**Revised:** 2026-08-07

Use this file as the parent for each focused component contract. A compact
contract may keep all sections in the parent. A larger contract may keep the
parent high-level and route cohesive low-level details to subordinate specs in a
directory named for the parent's filename stem. Both layouts represent one
component contract and use the same two stages:

1. **Phase 1 contract skeleton:** complete the content required by Sections 1–7
   without opening version 2 code, tests, or fixtures, then stop for boundary
   and reconnaissance scope approval by the applicable authority.
2. **Detailed component contract:** after that approval, perform the bounded
   version 2 reconnaissance, complete the content required by Sections 8–19,
   and stop for contract/reuse/test/slice-plan approval by the applicable
   authority. When this is the
   one-component lookahead skeleton, boundary approval records the future
   reconnaissance scope but Sections 8–19 wait until the preceding component
   passes final review unless the owner records an exact stable-interface
   exception.

The skeleton prevents predecessor structure from redefining Phase 1. The
reconnaissance prevents the detailed contract from ignoring working,
live-informed implementation evidence. Cite Phase 1 instead of restating it,
and omit private helper names or exact file layouts unless correctness or an
approved reuse decision requires them.

## Choosing the document layout

Use a modular contract when the detailed contract has multiple cohesive
semantic, trust, proof, or delivery concerns and loading unrelated sections is
making drafting, review, or implementation inefficient. Context size and edit
latency are valid reasons to modularize once those boundaries exist. Do not
split an arbitrary number of lines, headings, or tokens, and do not create
fragments so small that most tasks must load all of them.

A component with multiple implementation slices or large reconnaissance,
trust-boundary, edge-case, or proof ledgers should normally be modular unless
those details are compact and must repeatedly be reasoned about together.

File size is a context-routing alarm, not a reason to shard arbitrary prose.
Reassess the map when the mandatory parent is likely to exceed about 2,500
words or one normative detail is likely to exceed about 5,000 words. A larger
file requires an approval note explaining why its material is one inseparable
concern that every routed task must load. Otherwise extract cohesive semantic,
trust, proof, reconnaissance, or delivery details. Never create numbered parts
or split only to satisfy the word target.

### Compact contract form

The compact form preserves this template's two approval stages and all required
claims while reducing headings, repeated tables, and empty boilerplate. Use it
only when the component has one cohesive outcome and ownership boundary, is
expected to need one implementation slice, has a small reconnaissance and
trust-boundary ledger, and can remain below the parent context target above.
If detailed evidence later creates independently routable concerns or multiple
slices, convert the draft to the modular form before completed-contract
approval.

A compact contract may group the mandatory content under these headings:

1. `Sections 1–4 — Outcome, scope, ownership, and settled boundary`;
2. `Sections 5–7 — Evidence questions, reconnaissance scope, likely proof,
   and boundary checkpoint`;
3. after boundary approval, `Section 8 — Reconnaissance and reuse`;
4. `Sections 9–14 — Inputs/state, behavior, trust/failure outcomes,
   accounting/bounds, and evidenced edges`;
5. `Sections 15–17 — Primary proof, one-slice assignment, discretion,
   prohibitions, and escalation`; and
6. `Sections 18–19 — Acceptance checklist and drift audit`.

Within those grouped sections, omit an inapplicable table rather than filling
it with repetitive `none` rows, but state the inapplicability and reason in one
sentence. Keep exact requirement IDs, success/failure evidence, proof claims
and limitations, approvals, whitelist, slice assignment, and drift results.
Compact form is not permission to omit a gate or weaken a contract. The parent
document map records `Compact single-file contract, Sections 1–19`.

For a modular contract:

- the parent remains at the path listed in `docs/specification-map.md` and is
  the mandatory starting document;
- the parent retains status and approvals, Sections 1–4, the component-wide
  invariants, the single delivery-state ledger, and the contract document map
  below;
- Sections 5–19 may be placed in one or more subordinate specs according to
  cohesive concerns, with each required section and ledger routed exactly once;
- subordinate specs live under
  `docs/specifications/<parent-filename-stem>/` or nested concern directories
  below it, inherit the parent's approval state, and are not separate
  components or independently approved authorities; prefer a shallow layout
  and use nesting only when it makes a real subconcern independently routable;
- every normative decision, component requirement, evidence/reuse decision,
  primary proof, and slice allocation has one authoritative home; summaries
  link to that home instead of copying it; and
- document dependencies are explicit and acyclic. A localized task reads the
  parent, its routed subordinate specs, and their declared dependencies, not the
  whole set by default.

Start each subordinate spec with this routing header:

```text
# [Component name] — [Cohesive detail boundary]

**Parent contract:** [relative link to parent]
**Normative responsibility:** [the one concern controlled here]
**Controlling requirements:** [exact Phase 1 and component IDs]
**Allocated slices:** [exact slice IDs or none]
**Document dependencies:** [exact parent/detail links or none beyond parent]
**Approval state:** Inherits the parent contract approval; not independently approved
**Delivery state:** See the authoritative parent delivery-state ledger; do not copy mutable status here
```

Boundary approval and completed-contract approval review the entire
manifest-listed set. Adding, removing, or changing the normative responsibility
of a subordinate spec after approval requires explicit owner review. Document
decomposition does not alter component sequence, ownership, requirements,
proof allocation, or implementation-slice gates.

For C8-C11 under the standing unattended-program authority, both approval gates
also require the independent review, correction/re-review, parent evidence, and
separate local-commit records defined in `AGENTS.md` and
`docs/implementation-process.md`. Outside that exact program, the applicable
approval authority remains the owner.

For an approved single-file contract, do not apply this template piecemeal while
moving content. Use an owner-authorized documentation-only migration with an
approved parent map and exact move plan; keep the original authoritative until
the complete set passes link, coverage, duplication, and owner review. Separate
any substantive contract correction from that layout migration.

Delete instructional text when drafting.

---

# [Component name]

**Status:** Skeleton draft | Boundary approved | Detailed draft | Contract/
reuse/test/slice-plan approved | Approved for slice implementation

**Boundary approval authority:** [owner / exact standing program delegation]

**Boundary approval:** [date/commit/review evidence or pending]

**Completed-contract approval authority:** [owner / exact standing program delegation]

**Contract/reuse/test/slice-plan approval:** [date/commit/review evidence or pending]

**Standing program decisions:** [exact applicable decisions and manual stops,
or `none`; link to the repository program authority without weakening it]

**Advancement mode:** `delegated` (default after completed-contract approval) |
`manual` [name any manual slices or component-final gate]

**Controlling Phase 1 requirements:** [exact `PG-*`, `ARCH-*`, `DTE-*`, and
`LIFE-*` IDs]

**Approved dependencies:** [approved focused component contracts/interfaces,
or none]

## Contract document map

This table is mandatory. For a single-file contract, keep the parent row and
state that it owns Sections 1–19. For a modular contract, add every subordinate
spec before boundary approval and update its detailed coverage before completed-
contract approval. The `Read for` column is the context-routing instruction for
later models and implementers.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Component-level outcome, ownership/non-scope, cross-cutting invariants, routing, and approval state | [Component-wide IDs; Sections 1–4 and any sections retained here] | Every component task | [Approved dependency specs] |
| [Relative subordinate-spec link, or omit for single-file] | [One cohesive concern] | [Exact requirement IDs, proof IDs, slice IDs, and template sections] | [Tasks or decisions requiring this document] | [Parent plus exact subordinate dependencies] |

**Layout:** [Single-file contract / modular contract with the complete set
listed above]

**Routing rule:** [Any requirement or task not unambiguously routed by this
table stops for a parent-map correction; models do not guess among detail
specs.]

**Contract-wide coverage and acceptance:** [Compact status plus links to the
authoritative requirement/proof/slice ledgers and completed-contract checklist;
do not duplicate those ledgers here.]

## Authoritative delivery-state ledger

This is the only mutable slice/component delivery record. Detail specs name
allocations but link here instead of copying acceptance state. Update this table
as part of the same task that satisfies a delegated gate; do not require a
separate status-update turn.

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `skeleton` / `boundary_approved` / `contract_approved` / `implementing` / `final_review` / `complete` / `stopped` | [Approval/evidence link or concise reference] | [date/commit] | [Exact next gate] |
| `S1` | `pending` / `active` / `accepted` / `manual_review` / `failed` | [Primary proof results; independent review result or `not required`] | [date/commit] | [Continue to approved slice/final review, or owner decision] |

`accepted` may be recorded automatically only under the delegated gate in
`AGENTS.md` and the implementation process. A failed or ambiguous gate records
evidence but never records acceptance.

## 1. Outcome and user consequence

[What capability or trust claim this component enables. State what the trader
or operator observes when it succeeds, degrades, or is unavailable. Do not add
a product rule.]

## 2. Scope and explicit non-scope

**In scope**

- [Owned responsibility.]

**Not in scope**

- [Adjacent responsibility and its actual owner.]
- [Phase 1 behavior this spec consumes but does not redefine.]

## 3. Ownership and dependencies

[Name the single component ownership boundary and its approved dependencies.
State which mutable state—if any—it exclusively owns. Confirm that adapters and
workers return facts and that canonical state remains with
`ScannerStateEngine` where applicable.]

## 4. Settled Phase 1 semantic boundary

Summarize only the component boundary needed to control later reconnaissance.
Do not reproduce the shared event, time, lifecycle, or state contracts.

| Boundary item | Settled meaning | Controlling Phase 1 IDs |
| --- | --- | --- |
| [Semantic input/output, owned state, or invariant] | [Meaning that version 2 cannot change] | [`PG/ARCH/DTE/LIFE-*`] |

Explicitly confirm that the component introduces no new product rule,
competing mutable owner, watermark, evaluator, T/Q-to-ranking dependency,
changed time-window meaning, or duplicated responsibility.

## 5. Unresolved questions and evidence needs

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| [Implementation/provider/capacity/proof question] | [Delegated detail] | [Provider docs, v2 code/test/fixture, invariant, authorized observation, or owner decision] | [Contract or implementation choice] |

[Exclude speculative edge cases and questions already answered by Phase 1.]

## 6. Proposed version 2 reconnaissance scope

The only default predecessor is
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Do not inspect
version 1 without an explicit owner-approved exception.

Before looking at version 2, name the narrow source areas and questions to be
investigated. Exact paths/functions are added in Section 8 after discovery.

| Proposed code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| [Component-specific area] | [Why inspection is necessary] | [Unrelated orchestration/packages/behavior] |

### 6.1 Initial proof and slicing boundaries

Record only the likely proof boundaries needed to judge whether the proposed
contract is implementable. Detailed proof allocation belongs in Sections
14–16 after reconnaissance.

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| [Formula, normalization, lifecycle, replay, restart, or other component-local boundary] | [`PG/ARCH/DTE/LIFE-*`] | [Distinct claim this proof would establish] | [None, or an unresolved item from Section 5] |

**Provisional delivery assessment:** [one coherent implementation slice /
multiple sequential slices / unresolved]

**Reason and likely slice outcomes:** [Explain the component boundary,
dependency order, and why later work would extend rather than replace earlier
ownership or behavior.]

## 7. Boundary-approval checkpoint

- [ ] Exact controlling Phase 1 IDs are enumerated.
- [ ] Outcome, ownership, dependencies, scope, and non-scope are unambiguous.
- [ ] Settled inputs/outputs/state and invariants are sufficient to prevent
      version 2 from changing the architecture.
- [ ] Unresolved questions are genuinely delegated details.
- [ ] Proposed version 2 reconnaissance is narrow and question-driven.
- [ ] No version 2 code, tests, or fixtures were opened while preparing the
      skeleton.
- [ ] Initial likely proof boundaries are identified without inventing a broad
      test matrix.
- [ ] The skeleton states whether the component is likely to need multiple
      sequential implementation slices and why; this is provisional until
      detailed evidence and proofs are complete.

**Independent skeleton review:** [not required outside a standing program /
reviewer, result, findings/corrections, and focused re-review evidence]

**Skeleton drift audit:** [no substantive `yes`, with concise evidence for the
Section 19 questions at the Phase 1 boundary / exact unresolved `yes` and stop]

**Approval decision:** [owner or standing-program orchestrator; approved
boundary and reconnaissance scope / revisions]

**Stop here until approval.** Boundary approval permits only the named version
2 reconnaissance. It does not approve detailed behavior, reuse, tests, or
implementation. A standing-program approval is valid only after its required
independent review is clean and the approval is committed separately.

## 8. Version 2 reconnaissance and reuse assessment

Complete this section only after recording boundary approval above.

| Exact source path and function/type/test/fixture | Commit/file hash when relevant | Finding or validated behavior | Decision (`direct port` / `adapt` / `behavior evidence` / `reject`) | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| [v2 source] | [provenance] | [What the code/test/fixture establishes] | [decision] | [Old ownership, orchestration, dependency, or limitation] | [proof] |

Record existing test strength and limitations. If discovery requires unrelated
source areas, a new product rule, or a change to the approved skeleton, stop for
the applicable approval authority before expanding scope. Any post-completed-
contract change remains an owner decision.

**Proposed implementation whitelist:** [exact version 2 sources and fixtures,
or none]

Version 2 is evidence, never authority. Preserve validated behavior where it
satisfies Phase 1; do not transplant predecessor orchestration or competing
ownership.

## 9. Detailed semantic inputs, outputs, and owned state

| Item | Meaning and required provenance/identity | Bounds or ownership |
| --- | --- | --- |
| [Input/output/state] | [Semantic contract, informed by approved evidence] | [Limit/lifetime/owner] |

[Cite exact Phase 1 requirements. Specify concrete interface details needed by
dependent components, but do not create a second normalized-event, time,
lifecycle, or canonical-state contract.]

**Construction guarantees:** [Important invalid states or competing mutation
paths prevented by API, type, visibility, ownership, or immutable construction.
Keep this to component-defining guarantees.]

**Runtime validation still required:** [Important invalid states that remain
representable and therefore require explicit rejection or containment.]

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `[component ID]` | [Observable behavior and correctness-relevant design constraint] | [`PG/ARCH/DTE/LIFE-*`; approved evidence] |

Use v2 findings to avoid rediscovering validated algorithms and provider
behavior. Prescribe an exact algorithm or structure only when correctness,
boundedness, or approved reuse requires it; otherwise leave the mechanism to
implementation discretion.

### 10.1 Consequential trust-boundary acceptance

Include only external, persisted, or cross-component boundaries where invalid
evidence could falsely appear complete, current, valid, or safely contained.
Do not repeat ordinary private helper validation or create a universal mutation
matrix.

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| [Provider payload, cache artifact, persisted state, exported fact, ordered input, etc.] | [Concrete required evidence] | [Concrete invalid/contradictory evidence and global/local/persistence-only effect] | [Smallest invalid input or failure that could otherwise look successful] |

## 11. Failure and terminal behavior

[Define bounded failure outcomes, containment, progress/exit events, retry or
deadline terminal disposition, fencing/cancellation, and state exposed to the
engine/operator. Successful empty and failed/unknown outcomes remain distinct
where applicable.]

## 12. Accounting and observability

[Give each primary counter's accounting identity or state why the component has
no primary population. Label overlapping feature/diagnostic dimensions. Define
bounded reasons, metrics/cardinality, and evidence needed for currentness
without inventing exact metric names unless correctness requires them.]

## 13. Simplicity and boundedness

- Mutable owners/state representations: [minimum set and why each is needed].
- Queues, retained history, retries, work, diagnostics, and concurrency:
  [explicit bounds or the evidence needed to select them].
- V2 mechanisms retained: [why each is narrower/safer than reimplementation].
- V2 mechanisms rejected: [obsolete coupling, unnecessary machinery, or
  conflict with the approved boundary].
- Other credible alternatives: [why the chosen design is the simplest complete
  data path satisfying the contract].
- Confirm no hypothetical abstraction, generic event bus/plugin framework,
  database/service split, or unevidenced edge-case machinery was added.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| [Case] | [Provider doc, approved v2 fixture/test, authorized observation, invariant, demonstrated regression, or owner decision] | [Contract] | [Proof] |

[Keep evidence component-local. Embedded payload fixtures and diagnostic/review
evidence are valid; a raw WebSocket recording is not presumed. Exclude
speculative cases.]

## 15. Primary proof allocation

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result and proof limitation | Approved fixture/evidence | Allocated slice |
| --- | --- | --- | --- | --- | --- |
| [ID] | [Construction/ownership proof, formula test, normalization fixture, fault-injection or lifecycle scenario, deterministic replay, checkpoint/restart equivalence, differential replay, authorized shadow observation, or cutover evidence] | [Exact claim plus the invalid evidence/boundary condition that could otherwise falsely pass] | [Result distinguishing conformance; what this proof intentionally does not establish] | [Source] | [`S1`, etc.] |

[Add a second proof layer only when it establishes a different boundary, and
name that boundary. If one claim spans distinct implementations, prove each or
require one shared proved mechanism. Scanner agreement is correctness evidence,
not trading-edge or executable-expectancy validation.]

[One coherent test, trace, fixture, or construction inspection may be primary
for multiple tightly coupled requirement IDs when it names and asserts each
claim, dangerous counterexample, result, and limitation separately. Do not
create test functions solely to mirror IDs, and do not share a proof across an
unexercised branch or distinct implementation path.]

## 16. Sequential implementation-slice plan

Begin with the presumption that the entire component is one slice. Keep one
slice when it has one coherent outcome, ownership boundary, tightly related
proof family, and reviewable change set. Add each further slice only by naming
the independently observable behavior, distinct proof family, provider-I/O/
canonical-mutation boundary, ownership/package boundary, dependency order, or
reviewability problem that prevents one coherent assignment.

Split when the component contains multiple independently observable behaviors,
distinct proof families, provider I/O plus canonical mutation, separately
provable parts, multiple ownership/package boundaries, or a change set that
would be difficult to review as one unit. Do not split by estimated time, line
count, or token count. A different helper, file, or requirement ID is not by
itself a reason to split; do not force one slice when the named boundaries would
make review or rollback ambiguous.

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies/entry state | Allowed ownership or files/packages | Approved v2 whitelist/fixtures | Acceptance record | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `S1` | [One observable behavior or tightly related behavior set] | [Exact IDs and proofs from Section 15] | [Approved prior component/slice behavior] | [Precise boundary] | [Exact sources or none] | [Behavior, diff, proof design/result, deviations, next-slice validity] | [What remains unavailable until later slices] |

Slice rules:

- One implementation assignment covers exactly one slice.
- Each component requirement and primary proof is allocated to exactly one
  slice.
- A slice leaves the repository buildable and its allocated proofs passing.
- A slice produces coherent behavior, not unused scaffolding or a temporary
  competing state path.
- Later slices consume or extend earlier approved behavior; they do not replace
  its ownership or silently reinterpret its interface.
- Every slice states deferred behavior honestly and remains reviewable without
  starting the next slice.
- If a slice cannot be expressed as one coherent outcome with a reviewable
  proof, split it before implementation.
- After each slice, write one acceptance record to the parent delivery ledger.
  In delegated mode, continue when every objective gate passes; in manual mode
  or on any failed/ambiguous gate, stop for the owner.

Passing tests is necessary but not the whole slice-acceptance claim. The
acceptance record also includes a compact list of construction guarantees, the
main evidence-to-success and failure-containment paths, proof limitations or
inspection-only claims, and whether any shared requirement spans distinct
implementations.

**Independent slice-review trigger:** [None, with reason / exact consequential
trust, persistence, identity, ownership, ordering, concurrency, or dependency
interface boundary requiring a narrow independent review before dependent work.
Do not request a full repeated component review by default.]

[An owner response is not required for a clean delegated slice. When
independent agent review is required, use the execution and model/effort policy
in repository `AGENTS.md`.]

If a component requirement concerns interaction among slices, allocate its
primary proof to the last slice needed to make that interaction real. A later
milestone may add proof only for a distinct cross-component boundary; do not
duplicate slice-level primary proofs.

## 17. Implementation discretion

[List private helpers, internal layout, routine algorithms, error wrapping, and
equivalent mechanics delegated to implementation. Separately list exact v2
adaptations or algorithms required for correctness, and choices that must return
to owner review because they would alter the approved contract.]

**Prohibited changes**

- [Product, architecture, ownership, interface, evidence, or whitelist change
  the implementation assignment may not make.]

**Stop/escalation conditions**

- [Conflict, missing evidence, unexpected predecessor coupling, interface
  change, or proof failure that requires owner review before continuing.]

## 18. Completed-contract acceptance checklist

**Independent completed-contract review:** [not required outside a standing
program / reviewer, result, findings/corrections, and focused re-review evidence]

**Approval decision:** [owner or standing-program orchestrator; exact contract,
whitelist, fixtures/evidence, proofs, slices, required reviews, advancement
mode, and local commit]

- [ ] The parent document map lists the complete contract set and gives every
      document one exclusive normative responsibility.
- [ ] The parent owns the only mutable delivery-state ledger; subordinate specs
      do not copy slice status, dates, proof results, or reviewer results.
- [ ] Advancement mode is explicit, with delegated mode as the default and any
      manual slice/final gates named.
- [ ] Compact form is used only for a cohesive one-slice component with small
      ledgers; otherwise the contract is routed modularly.
- [ ] The parent and detail sizes are below the context-routing targets, or the
      approval record explains why the larger concern is inseparable.
- [ ] Every required template section, requirement, proof, and slice is routed
      to exactly one authoritative document; parent summaries only link to
      controlling details.
- [ ] Document dependencies are explicit and acyclic, and all relative links
      resolve.
- [ ] A slice can be implemented from the parent, its routed detail specs, and
      declared dependencies without loading unrelated detail specs.
- [ ] Boundary and reconnaissance scope approval by the applicable authority is
      recorded with required independent-review evidence.
- [ ] Version 2 inspection stayed inside that scope, or expansions received
      approval from the applicable authority before inspection.
- [ ] Exact version 2 sources, decisions, adaptations, fixtures, and proof
      obligations are recorded.
- [ ] Inputs, outputs, owned state, bounds, required behavior, and terminal
      outcomes are complete without duplicating Phase 1.
- [ ] Primary accounting identities and overlapping dimensions are explicit.
- [ ] Every nontrivial edge case has evidence or approval from the applicable
      authority.
- [ ] Every component requirement has one primary proof; duplicate layers name
      a distinct boundary.
- [ ] Consequential trust boundaries define exact success evidence, rejection
      or containment, and the dangerous false-success case.
- [ ] Primary proofs name the exact claim, counterexample, observable result,
      participating implementation paths, and intentional limitation.
- [ ] Important invalid states prevented by construction are distinguished
      from representable states requiring runtime validation.
- [ ] Every requirement and primary proof is allocated to exactly one
      implementation slice.
- [ ] Every slice has one coherent outcome, precise scope, explicit deferred
      behavior, a compact acceptance record, and an exact delegated/manual gate.
- [ ] Slice ordering is acyclic; no slice requires an interface or behavior
      defined only by a later slice.
- [ ] Later slices extend rather than replace earlier ownership and behavior.
- [ ] Implementation discretion, prohibited changes, and escalation conditions
      are clear enough for one bounded slice assignment at a time.
- [ ] Each slice defines its compact conformance walkthrough and whether a
      narrow independent slice review is triggered.
- [ ] The drift audit below has no unresolved substantive **yes**.
- [ ] Exact version 2 implementation and fixture whitelist received
      contract/reuse/test/slice-plan approval from the applicable authority.
- [ ] Every applicable standing program decision and manual stop is recorded
      without weakening repository authority.
- [ ] Any required independent completed-contract review is clean after
      corrections and focused re-review, and its evidence is recorded before
      approval.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | | |
| Did this introduce another mutable state owner, watermark, or evaluator? | | |
| Did this make aggregate ranking/readiness depend on T/Q? | | |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | | |
| Did this add behavior without component-local evidence or explicit approval? | | |
| Did this duplicate an existing responsibility or Phase 1 contract? | | |
| Did this add machinery without an approved need? | | |
| Did version 2 drive the Phase 1 boundary instead of informing the detailed implementation contract? | | |

Any substantive **yes** requires explicit owner review before advancement.
