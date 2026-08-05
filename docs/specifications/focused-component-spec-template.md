# Focused component specification template

**Status:** Owner-approved mandatory template.

**Approved:** 2026-08-05

Copy this file for each focused component specification. It is completed in two
stages:

1. **Phase 1 contract skeleton:** complete Sections 1–7 without opening version
   2 code, tests, or fixtures, then stop for owner boundary and reconnaissance
   scope approval.
2. **Detailed component contract:** after that approval, perform the bounded
   version 2 reconnaissance, complete Sections 8–19, and stop for owner
   contract/reuse/test/slice-plan approval.

The skeleton prevents predecessor structure from redefining Phase 1. The
reconnaissance prevents the detailed contract from ignoring working,
live-informed implementation evidence. Cite Phase 1 instead of restating it,
and omit private helper names or exact file layouts unless correctness or an
approved reuse decision requires them.

Delete instructional text when drafting.

---

# [Component name]

**Status:** Skeleton draft | Boundary approved | Detailed draft | Contract/
reuse/test/slice-plan approved | Approved for slice implementation

**Owner boundary approval:** [date/link or pending]

**Owner contract/reuse/test/slice-plan approval:** [date/link or pending]

**Controlling Phase 1 requirements:** [exact `PG-*`, `ARCH-*`, `DTE-*`, and
`LIFE-*` IDs]

**Approved dependencies:** [approved focused specs/interfaces, or none]

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

**Owner decision:** [approved boundary and reconnaissance scope / revisions]

**Stop here until owner approval.** Boundary approval permits only the named
version 2 reconnaissance. It does not approve detailed behavior, reuse, tests,
or implementation.

## 8. Version 2 reconnaissance and reuse assessment

Complete this section only after recording owner boundary approval above.

| Exact source path and function/type/test/fixture | Commit/file hash when relevant | Finding or validated behavior | Decision (`direct port` / `adapt` / `behavior evidence` / `reject`) | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| [v2 source] | [provenance] | [What the code/test/fixture establishes] | [decision] | [Old ownership, orchestration, dependency, or limitation] | [proof] |

Record existing test strength and limitations. If discovery requires unrelated
source areas, a new product rule, or a change to the approved skeleton, stop for
owner review before expanding scope.

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

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `[component ID]` | [Observable behavior and correctness-relevant design constraint] | [`PG/ARCH/DTE/LIFE-*`; approved evidence] |

Use v2 findings to avoid rediscovering validated algorithms and provider
behavior. Prescribe an exact algorithm or structure only when correctness,
boundedness, or approved reuse requires it; otherwise leave the mechanism to
implementation discretion.

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

| Requirement | One primary proof | Distinct boundary proved | Approved fixture/evidence | Allocated slice |
| --- | --- | --- | --- | --- |
| [ID] | [Formula test, normalization fixture, lifecycle scenario, deterministic replay, checkpoint/restart equivalence, differential replay, authorized shadow observation, or cutover evidence] | [Exact claim] | [Source] | [`S1`, etc.] |

[Add a second proof layer only when it establishes a different boundary, and
name that boundary. Scanner agreement is correctness evidence, not trading-edge
or executable-expectancy validation.]

## 16. Sequential implementation-slice plan

First decide whether the entire component has one coherent outcome, one
ownership boundary, one primary proof family, and one reviewable change set. If
so, define one slice. Otherwise split it at behavior and proof boundaries.

Split when the component contains multiple independently observable behaviors,
distinct proof families, provider I/O plus canonical mutation, separately
provable parts, multiple ownership/package boundaries, or a change set that
would be difficult to review as one unit. Do not split by estimated time, line
count, or token count.

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies/entry state | Allowed ownership or files/packages | Approved v2 whitelist/fixtures | Owner-review artifact | Explicitly deferred behavior |
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
- After each slice, stop for owner review and authorization before starting the
  next slice.

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

- [ ] Owner-approved boundary and reconnaissance scope are recorded.
- [ ] Version 2 inspection stayed inside that scope, or expansions received
      explicit owner approval.
- [ ] Exact version 2 sources, decisions, adaptations, fixtures, and proof
      obligations are recorded.
- [ ] Inputs, outputs, owned state, bounds, required behavior, and terminal
      outcomes are complete without duplicating Phase 1.
- [ ] Primary accounting identities and overlapping dimensions are explicit.
- [ ] Every nontrivial edge case has evidence or owner approval.
- [ ] Every component requirement has one primary proof; duplicate layers name
      a distinct boundary.
- [ ] Every requirement and primary proof is allocated to exactly one
      implementation slice.
- [ ] Every slice has one coherent outcome, precise scope, explicit deferred
      behavior, a review artifact, and an owner-review stop.
- [ ] Slice ordering is acyclic; no slice requires an interface or behavior
      defined only by a later slice.
- [ ] Later slices extend rather than replace earlier ownership and behavior.
- [ ] Implementation discretion, prohibited changes, and escalation conditions
      are clear enough for one bounded slice assignment at a time.
- [ ] The drift audit below has no unresolved substantive **yes**.
- [ ] Exact version 2 implementation and fixture whitelist received owner
      contract/reuse/test/slice-plan approval.

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
