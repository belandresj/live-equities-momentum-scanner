# Repository instructions

## Authority

Read `README.md`, `docs/specification-map.md`, and the relevant approved product,
architecture, and component specifications before changing code or contracts.

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved focused component specifications; and
4. implementation assignments, plans, tests, and code.

If two authorities conflict, stop and report the conflict. Do not silently pick
one.

## Current phase

Phase 1 product and architecture specification is complete and owner-approved.
The repository is now in Phase 2 focused specification design.

Do not add production code, provider integrations, runtime infrastructure, or
tests until the applicable focused component specification has completed the
approval gates in [`docs/implementation-process.md`](docs/implementation-process.md)
and the owner has authorized a bounded implementation-slice assignment.

Implementation is sequential in the order recorded in
[`docs/specification-map.md`](docs/specification-map.md). Keep at most one
implementation slice active. A focused component spec may divide its delivery
into multiple owner-approved sequential slices; each slice implements one
coherent behavior and its allocated primary proof. While component N is being
implemented slice by slice, specification work may advance for component N+1,
but N+1 contract/reuse/test/slice-plan approval and implementation wait until N
passes final component review, unless the owner explicitly approves an earlier
contract decision against an already stable dependency interface.

## Before completing a focused specification

Contract-first drafting is mandatory:

1. Read this file, `README.md`, `docs/specification-map.md`, the approved Phase 1
   authorities relevant to the component, and any approved dependency specs.
2. Enumerate the exact Phase 1 requirement IDs that control the component.
3. State the component's single ownership boundary and explicit non-scope.
4. Choose the simplest design that satisfies every cited requirement.
5. Do not introduce a new product rule, competing state owner, watermark,
   evaluator, T/Q-to-ranking dependency, changed time-window meaning,
   speculative edge case, or duplicated responsibility.
6. Create a short Phase 1 contract skeleton before inspecting version 2. It
   must state the outcome, controlling IDs, ownership/non-scope, settled
   semantic boundary and invariants, unresolved evidence questions, and the
   proposed scope of version 2 reconnaissance.
7. Stop for owner boundary approval. This approves the Phase 1 constraints and
   exact reconnaissance scope, not the detailed component contract or reuse.
8. After boundary approval, inspect only the approved relevant version 2 code,
   tests, and fixtures. Use the findings to complete the focused specification
   and reuse/proof assessment.
9. Decide whether the component fits one bounded implementation assignment. If
   it contains multiple independently provable behaviors, proof families,
   provider/canonical boundaries, or reviewable changes, define sequential
   implementation slices in the focused spec. Assign every requirement and its
   primary proof to exactly one slice.
10. Stop for owner contract/reuse/test/slice-plan approval before preparing or
    authorizing an implementation assignment.

Use the mandatory
[`focused component specification template`](docs/specifications/focused-component-spec-template.md).
Phase 1 already controls shared engine, state, event, time, and lifecycle
semantics. Do not create a redundant shared architecture or core-domain spec.

## Engineering rules

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

At the end of every slice, stop for owner review. Report the behavior now
available, requirements completed, proof design and result, changed ownership
or interfaces, deviations, and whether the next slice remains valid. Do not
begin the next slice until the owner authorizes it. If a proposed slice cannot
be expressed as one coherent outcome with a reviewable primary proof, split it
before implementation.

Implementers make routine lower-level choices within those bounds. A
consequential component choice is resolved in the applicable component spec;
repository-wide mechanical conventions belong here or in the implementation
process. This repository does not use an ADR workflow.
