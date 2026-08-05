# Repository instructions

## Authority

Read `README.md`, `docs/specification-map.md`, and the relevant approved product,
architecture, and component specifications before changing code or contracts.

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. accepted architecture decision records;
4. component specifications;
5. implementation plans and code.

If two authorities conflict, stop and report the conflict. Do not silently pick
one.

## Current phase

The repository is in Phase 1 documentation design. Do not add production code,
provider integrations, runtime infrastructure, or speculative tests until the
applicable specifications are approved.

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

## Agent assignments

An implementation assignment should identify the authoritative documents,
allowed scope, expected interfaces, approved fixtures, and required
verification. Make lower-level implementation decisions within those bounds and
record only consequential architectural choices as ADRs.
