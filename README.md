# Live Equities Momentum Scanner

Live Equities Momentum Scanner is a planned production-grade, real-time U.S.
equities scanner for a discretionary momentum trader. It ranks qualified stocks
by return from the adjusted previous regular-session close, presents
aggregate-derived session context, and provides trade- and quote-derived
measurements for the displayed top 20.

The scanner is a market-data measurement and discovery product. It does not
produce forecasts, recommendations, entries, exits, orders, or claims of
executable trading expectancy.

## Product behavior

The scanner is designed to:

- bind each run to an exchange schedule, eligible U.S. common-stock/ADRC
  universe, and exact adjusted prior closes;
- consume live and historical one-second aggregates through one canonical event
  and state path;
- qualify symbols using the approved same-session aggregate-tape gate, then rank
  passers by Day % descending with exact-symbol tie-breaking;
- publish at most 20 rows with independently available price, range, Activity,
  Tape Rate, and Spread measurements;
- recover from fresh start, checkpoint restart, and same-process aggregate gaps
  without fabricating marks or treating successful empty hydration as unfinished
  work; and
- expose readiness, coverage, population accounting, and field availability
  honestly through a versioned read-only API.

The complete product contract is
[`docs/product/product-goals.md`](docs/product/product-goals.md).

## Architecture

The design has:

- one authoritative `ScannerStateEngine` and one canonical per-symbol session
  state;
- normalized aggregate, trade, quote, control, hydration, replay, and timer
  inputs;
- one committed aggregate watermark and one qualification/ranking path;
- explicit bootstrap, checkpoint catch-up, live, recovery, replay, suppression,
  session-end, and shutdown lifecycles;
- default T/Q coverage for displayed rows, with T/Q degraded before aggregate
  correctness;
- coherent checkpoints and deterministic offline aggregate replay;
- immutable snapshots served by the scanner backend; and
- an independently deployable UI that owns presentation, not market state.

The architecture deliberately excludes a database, microservice split, generic
event bus, runtime plugin system, and browser-owned scanner logic unless a future
approved requirement establishes a concrete need.

## Repository guide

| Document | Purpose |
| --- | --- |
| [`AGENTS.md`](AGENTS.md) | Repository rules, authority order, engineering invariants, and agent-assignment requirements. |
| [`docs/product/product-goals.md`](docs/product/product-goals.md) | Highest product authority: user-facing behavior, formulas, version 1 scope, and non-goals. |
| [`docs/architecture/system-overview.md`](docs/architecture/system-overview.md) | Runtime topology, component boundaries, state ownership, and failure containment. |
| [`docs/architecture/data-time-and-event-contract.md`](docs/architecture/data-time-and-event-contract.md) | Session, clock, event, ordering, coverage, reconciliation, replay, and checkpoint-cutoff semantics. |
| [`docs/architecture/scanner-state-engine-lifecycle.md`](docs/architecture/scanner-state-engine-lifecycle.md) | Legal engine states, transitions, publication permissions, recovery, and termination. |
| [`docs/glossary.md`](docs/glossary.md) | Shared vocabulary; the controlling product or architecture contract wins when more specific. |
| [`docs/specification-map.md`](docs/specification-map.md) | Current component sequence, document status, dependencies, and implementation milestones. |
| [`docs/implementation-process.md`](docs/implementation-process.md) | Contract-first research, approval, predecessor-reuse, proof, slice, integration, and release workflow. |
| [`docs/specifications/focused-component-spec-template.md`](docs/specifications/focused-component-spec-template.md) | Mandatory template and modular-layout rules for focused component contracts. |
| [`docs/history/`](docs/history/) | Non-authoritative Phase 1 drafting and review history. |

## How work advances

Phase 1 product and architecture contracts are owner-approved. Current component
status and the sequential implementation roadmap are maintained only in the
[`specification map`](docs/specification-map.md).

Focused component contracts follow the contract-first
[`implementation process`](docs/implementation-process.md): establish the Phase
1-derived boundary, obtain approval for narrow version 2 reconnaissance,
complete the component contract and proof allocation, then implement one
owner-authorized slice at a time. A contract may be one file or a compact
indexed parent with cohesive subordinate detail specs; either layout remains
one component authority and passes the same approval gates. Production
implementation is not authorized merely because a specification draft exists.

### Authorized C7-through-C11 local program

The owner approved the completed Component 7 checkpoint contract on
2026-08-07 and granted standing, conditional authority for one sequential
unattended C7-through-C11 program on `codex/c7-c11-program`. Components 8-11
still require separate Phase 1 skeleton and completed-contract gates, but the
future goal orchestrator may record each gate without another owner message
only after mandatory independent review, correction/re-review as needed,
complete proof allocation, a clean drift audit, and conformance to the approved
product, architecture, dependency, and program decisions. The exact delegation,
manual stops, single-writer rule, and local-commit policy are in
[`AGENTS.md`](AGENTS.md#c7-through-c11-unattended-program-authority) and the
[`implementation process`](docs/implementation-process.md#28-c7-through-c11-unattended-program).

The target is a completely reviewed and locally verified private release
candidate through Component 11. This program does not authorize credentialed
live-provider observation, a public capacity claim, public deployment,
authentication/TLS/hosting, production cutover, pushing, or history rewriting.
Chrome desktop is the required UI target, and Component 11 is a high-fidelity
adaptation of the useful V2 scanner UI without V2 browser-owned calculations,
readiness logic, obsolete state semantics, or backend coupling.

## Predecessor evidence

The version 2 predecessor is retained separately as a source of Massive protocol
behavior, observed edge cases, fixtures, algorithms, and regression evidence. It
is evidence, never authority. It may be inspected only within a narrow,
owner-approved reconnaissance scope after the new component boundary has been
derived from Phase 1. Its orchestration architecture is not the starting point
for this implementation.
