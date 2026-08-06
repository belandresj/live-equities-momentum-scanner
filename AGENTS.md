# Repository instructions

## Authority

Read `README.md`, `docs/specification-map.md`, and the relevant approved product,
architecture, and component contracts before changing code or contracts.

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved focused component contracts, whether represented by one
   specification file or one indexed parent plus subordinate detail specs; and
4. implementation assignments, plans, tests, and code.

If two authorities conflict, stop and report the conflict. Do not silently pick
one.

## Current phase

Phase 1 product and architecture specification is complete and owner-approved.
The repository is now in Phase 2 focused component-contract design.

Do not add production code, provider integrations, runtime infrastructure, or
tests until the applicable focused component contract, including every detail
spec routed to the slice, has completed the approval gates in
[`docs/implementation-process.md`](docs/implementation-process.md) and the
owner-approved completed contract and advancement mode permit the bounded
implementation-slice assignment.

Implementation is sequential in the order recorded in
[`docs/specification-map.md`](docs/specification-map.md). Keep at most one
implementation slice active. A focused component contract may divide its
delivery into multiple sequential slices approved as one contract plan; each
slice implements one coherent behavior and its allocated primary proof. The
approved contract's advancement mode controls whether clean evidence-based
slice gates advance automatically or wait for an owner decision. While
component N is being implemented slice by slice, component N+1 may advance only
through its short Phase 1 skeleton and owner boundary/reconnaissance-scope
approval. Do not inspect version 2 or draft N+1 Sections 8–19 until N passes
final component review, unless the owner explicitly approves earlier work
against an already stable dependency interface. N+1 completed-contract
approval and implementation always wait for N final review unless that same
exception explicitly covers the later gate.

## Before completing a focused component contract

Contract-first drafting is mandatory:

1. Read this file, `README.md`, `docs/specification-map.md`, the approved Phase 1
   authorities relevant to the component, and any approved dependency specs.
2. Enumerate the exact Phase 1 requirement IDs that control the component.
3. State the component's single ownership boundary and explicit non-scope.
4. Choose the simplest design that satisfies every cited requirement.
5. Do not introduce a new product rule, competing state owner, watermark,
   evaluator, T/Q-to-ranking dependency, changed time-window meaning,
   speculative edge case, or duplicated responsibility.
6. Choose the initial document layout and create a short Phase 1 contract
   skeleton before inspecting version 2. It must state the outcome, controlling
   IDs, ownership/non-scope, settled semantic boundary and invariants,
   unresolved evidence questions, proposed version 2 reconnaissance scope, and
   the proposed document map when modular.
7. Stop for owner boundary approval. This approves the Phase 1 constraints and
   exact reconnaissance scope, not the detailed component contract or reuse.
8. After boundary approval, inspect only the approved relevant version 2 code,
   tests, and fixtures. Use the findings to complete the focused contract
   and reuse/proof assessment. For a one-component lookahead skeleton, boundary
   approval records the future reconnaissance scope but does not permit this
   step until the preceding component passes final review, unless the owner
   explicitly approves a stable-interface exception.
9. Confirm the approved document layout still fits the evidence. Use a compact
   indexed parent plus subordinate detail specs when the detailed contract
   contains multiple cohesive concerns or routinely loading unrelated sections
   makes drafting, review, or implementation inefficient. A multiple-slice
   component or a component with large reconnaissance/proof ledgers should
   normally be modular. Do not keep a monolith merely to preserve a one-file
   convention, and do not shard arbitrary line or token ranges without a
   semantic boundary. Stop for owner review before changing the approved
   document map or reconnaissance routing.
10. Decide whether the component fits one bounded implementation assignment. If
   it contains multiple independently provable behaviors, proof families,
   provider/canonical boundaries, or reviewable changes, define sequential
   implementation slices in the focused contract. Assign every requirement and
   its primary proof to exactly one slice.
11. Stop for owner contract/reuse/test/slice-plan approval before preparing or
    authorizing an implementation assignment. That approval also fixes the
    component's advancement mode; unless the owner marks a slice or component
    `manual`, evidence-based slice and final-component acceptance are delegated
    under the rules below.

Use the mandatory
[`focused component specification template`](docs/specifications/focused-component-spec-template.md).
Phase 1 already controls shared engine, state, event, time, and lifecycle
semantics. Do not create a redundant shared architecture or core-domain spec.

### Modular focused component contracts

A modular specification set is still one focused component contract and one
authority. Its parent file remains at the path listed in
`docs/specification-map.md` and is the mandatory entry point. The parent stays
compact and contains the component outcome, single ownership boundary, explicit
non-scope, cross-cutting invariants, approval state, and a normative document
map that routes each concern, requirement family, primary-proof allocation, and
implementation slice to exactly one authoritative detail spec. The parent also
owns the single authoritative delivery-state ledger for the component.

Place subordinate specs in a directory named for the parent specification's
filename stem. Each subordinate spec must own one cohesive semantic, trust,
proof, or delivery boundary; identify its parent, controlling requirement IDs,
component requirements, dependencies, and allocated slices; and avoid
restating normative text owned elsewhere. A subordinate spec is not a new
component, shared architecture layer, independently approved authority, or way
to bypass the component approval gates.

Every normative decision, component requirement, evidence/reuse decision,
primary proof, and slice allocation has one authoritative home across the set.
The parent may summarize detail for navigation, but must link to the controlling
location and must not duplicate or weaken it. Cross-document dependencies must
be explicit and acyclic. A conflict or unrouted requirement stops work.
Slice acceptance and current delivery state are recorded only in the parent
ledger. Detail specs link to that ledger and do not copy mutable status, dates,
proof results, or review results.

For a localized task, read the parent first, then the detail specs selected by
its document map and their declared dependencies. Do not load unrelated detail
specs by default. Boundary approval, completed-contract approval, a
cross-cutting contract change, and final component review examine the complete
manifest-listed set. Adding, removing, or changing the normative responsibility
of a detail spec after approval requires explicit owner review.

An approved monolith may be converted through an owner-authorized,
documentation-only layout migration. First approve the proposed parent map and
exact section-to-document move plan. Preserve normative meaning, make no reuse,
proof, requirement, or slice change in that migration, and keep the original
file authoritative until the complete moved set passes link, coverage,
duplication, and owner review. Any substantive edit follows the normal
component-contract revision gate instead.

## Engineering rules

### Language and tooling

- Use Go 1.26 with standard Go module tooling.
- The module path is `github.com/belandresj/live-equities-momentum-scanner`.
- Prefer the Go standard library; add third-party dependencies only when an
  owner-approved component contract and implementation slice require them.

- Keep one authoritative `ScannerStateEngine` and one canonical symbol state.
- Provider adapters normalize data; they do not determine ranking readiness.
- REST and live aggregates use the same canonical identity and merge rules.
- Aggregate ranking never depends on trade/quote availability or T/Q health.
- Treat successful empty hydration as an explicit no-print outcome, not as a
  fabricated mark or unfinished work.
- Every primary population counter must participate in a documented accounting
  identity. Label overlapping feature dimensions explicitly.
- Checkpoints must represent one coherent committed timestamp.
- The UI must remain deployable independently from the scanner backend.
- Do not introduce a database, service split, generic event bus, plugin system,
  or generalized framework without an approved architectural need.
- Prefer the minimum number of mutable states and owners, and do not add an
  abstraction, edge-case mechanism, or extension point for a hypothetical use.

## Edge cases and tests

Do not implement speculative provider edge cases. A nontrivial edge case must be
supported by provider documentation, a captured fixture, a production
observation, a product invariant, a predecessor regression, or explicit owner
approval.

Each requirement should have one primary proof. Avoid duplicating the same
scenario across unit, component, integration, replay, and live tests unless the
additional layer proves a distinct boundary.

Passing allocated tests is necessary but not sufficient for slice acceptance.
For each external, persisted, or otherwise consequential trust boundary, the
focused component contract must state what is accepted, what is rejected or
contained, and the most dangerous way invalid evidence could falsely reach
success. Each primary proof must name that counterexample, its observable
result, and what the proof does not establish. Slice review also records the
important invalid states prevented by construction, a short success/failure-
path walkthrough, and any material claim supported only by inspection. Choose
these checks for the component's actual risks; do not create a broad universal
matrix or duplicate proof layers.

Never access provider credentials or make live provider requests unless the
owner explicitly authorizes that activity for the current task.

Evidence and proof plans are component-local. Do not create a broad evidence
inventory or speculative repository-wide test matrix. Evidence may be provider
documentation, an existing version 2 JSON or fake-provider fixture, a
controlled live observation, a product or mathematical invariant, a
demonstrated regression, or an explicit owner decision. Each requirement has
one primary proof; another test layer is justified only when it proves a
different boundary. Version 2 code, tests, and fixtures may be opened only after
owner approval of the Phase 1 contract skeleton and reconnaissance scope.

## Predecessor reuse

The only predecessor permitted by default is
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Do not inspect
the older version 1 checkout without an explicit owner-approved exception for a
precise purpose.

Version 2 is evidence, never authority. Inspect it only after owner boundary
approval of the Phase 1 contract skeleton, and only within the approved
reconnaissance scope. Record the exact code/test/fixture source, reuse decision,
preserved behavior, coupling to remove, and required proof. Implementation may
use only owner-approved, whitelisted version 2 sources and fixtures.

## Agent assignments

An implementation assignment covers exactly one approved slice. It must
identify the slice outcome, authoritative documents and exact requirement IDs,
approved scope and non-scope, dependencies and interfaces, allowed
files/packages or ownership boundary, approved version 2 whitelist and
fixtures, primary proof, required verification and review artifact, explicitly
deferred behavior, delegated decisions, prohibited changes, and
stop/escalation conditions.

At the end of every slice, produce the compact acceptance record required by
the implementation process. Under delegated advancement, if every allocated
proof and required independent review passes, the conformance walkthrough is
clean, there is no contract deviation or escalation condition, and the next
slice remains valid, update the parent delivery ledger to `accepted` and begin
the next already-approved slice without requesting another owner message. Send
the owner a concise progress update, not an approval question. Under manual
advancement, or whenever a delegated gate is not clean, stop for the owner.

If a proposed slice cannot be expressed as one coherent outcome with a
reviewable primary proof, split it before implementation through the applicable
contract-change gate; delegated advancement cannot change the approved plan.

Implementers make routine lower-level choices within those bounds. A
consequential component choice is resolved in the applicable component spec;
repository-wide mechanical conventions belong here or in the implementation
process. This repository does not use an ADR workflow.

### Independent agent-review execution

Owner authority and independent agent review are different. An independent
agent supplies read-only review evidence; it does not change the contract or
implementation. A clean required review can satisfy an objective delegated
acceptance gate, but it cannot approve new scope, revised behavior, a new
whitelist, or another consequential decision reserved to the owner.

Do not spawn an independent reviewer automatically after every slice. Spawn one
only when:

- the approved component contract marks that slice's consequential trust,
  persistence, identity, ownership, ordering, concurrency, or dependency-
  interface boundary as requiring narrow independent review;
- implementation evidence exposes a new risk in one of those categories and
  an in-scope narrow review is needed before delegated advancement; or
- all slices have passed and the mandatory read-only final component review is
  due.

When an independent review is delegated and the model is available, use
`gpt-5.6-sol` with medium reasoning by default. Use high reasoning only when the
owner requests it or a medium review leaves concrete unresolved uncertainty
about concurrency linearization, persistence/atomicity, invalid external
evidence reaching false success, sole-owner enforcement, or a cross-component
contract conflict. Record any model substitution when the default is not
available.

After findings are corrected, reuse the same reviewer when practical and
request a focused re-review of the findings and affected boundary. Do not
repeat the complete review unless the correction changes a cross-cutting
contract, ownership path, or proof premise. A clean review is not repeated only
to obtain another clean result.

Development and acceptance verification follow the tiered cadence in
[`docs/implementation-process.md`](docs/implementation-process.md). A component
contract or bounded assignment may require stronger verification for its
specific risk; routine implementation must not silently expand every slice to
the most expensive repository-wide verification tier.

### Explicit owner decisions and delegated advancement

Explicit owner approval remains required for:

- the Phase 1 boundary and version 2 reconnaissance scope;
- the completed contract, reuse/fixture whitelist, proof allocation, slice
  plan, and any slice explicitly marked `manual`;
- a substantive contract, requirement, document-map responsibility,
  cross-component interface, or approved-whitelist change;
- live-provider credentials or observations, destructive/external actions, and
  production cutover; and
- any failed or ambiguous delegated gate whose resolution could change behavior
  or scope.

Approval of a completed contract delegates its objective evidence-based slice
and final-component gates by default. Delegated acceptance requires all of the
following:

1. every allocated proof and required verification passes;
2. every required independent review is clean after any focused corrections;
3. the implementation and evidence remain inside the approved scope and
   whitelist;
4. the acceptance walkthrough reports no deviation, failed assumption,
   unresolved inspection-only claim capable of invalidating success, or drift-
   audit `yes`; and
5. the next slice or final-component claim remains exactly as approved.

When those conditions hold, the implementing agent records acceptance in the
component parent's delivery ledger and advances. When they do not, it records
the failed evidence without marking acceptance and stops for the smallest owner
decision. The owner may set `advancement_mode: manual` for any component or
slice and may revoke delegation prospectively at any time.
