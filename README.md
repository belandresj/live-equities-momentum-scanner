# Live Equities Momentum Scanner

Live Equities Momentum Scanner is an in-progress real-time U.S. equities
scanner for a discretionary momentum trader. It ranks qualified stocks
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
| [`docs/v1-release-program.md`](docs/v1-release-program.md) | Owner-approved C7-C11 authority, fixed/revisable decisions, zero-interruption correction and containment, capability/proof matrix, test tiers, and private V1 RC completion. |
| [`docs/c12-implementation-goal.md`](docs/c12-implementation-goal.md) | Owner-approved handoff prompt to close C11/private V1 RC, then implement C4-S6 and C12 sequentially. |
| [`docs/market-hours-validation.md`](docs/market-hours-validation.md) | Separately authorized post-RC live-provider observation procedure; pending by default. |
| [`docs/specifications/focused-component-spec-template.md`](docs/specifications/focused-component-spec-template.md) | Mandatory template and modular-layout rules for focused component contracts. |
| [`docs/history/`](docs/history/) | Non-authoritative Phase 1 drafting and review history. |

## How work advances

Phase 1 product and architecture contracts are owner-approved. Current component
status and the sequential implementation roadmap are maintained only in the
[`specification map`](docs/specification-map.md).

Focused component contracts follow the contract-first
[`implementation process`](docs/implementation-process.md): establish the Phase
1-derived boundary, record a narrow Version 2 reconnaissance scope, complete
the component contract/proof allocation, and implement one sequential slice at
a time. A contract may be one file or a compact indexed parent with cohesive
details; either remains one component authority. Under the V1 program, lower-
level C7-C11 decisions remain revisable when evidence exposes a defect.

### Authorized Version 1 Release Program

The owner replaced the former C7-C11 unattended authority on 2026-08-07 with
the [`Version 1 Release Program`](docs/v1-release-program.md). It permits a
future goal on `codex/c7-c11-program` to correct C7, complete C7-C11
sequentially, and revise lower-level contracts, fixtures, proofs, thresholds,
slices, whitelists, reviews, and accepted implementation decisions without
another owner message. The approved Phase 1 market semantics and architecture
remain fixed. C7-C11 have no planned owner-response gate: lower-level failures
are corrected, excluded external/live work is deferred, and user/Git/tool
friction uses recorded containment and fallback while the goal continues.

The target is a reviewed and locally verified private release candidate through
Component 11. Deterministic replay and fake-provider evidence may finish it
while the market is closed. The program does not authorize credentialed live-
provider observation, a public capacity claim, public deployment,
authentication/TLS/hosting, production cutover, pushing, or history rewriting.
Chrome desktop is the required UI target, and Component 11 is a high-fidelity
adaptation of the useful V2 scanner UI without V2 browser-owned calculations,
readiness logic, obsolete state semantics, or backend coupling.

After C11 and the integrated private V1 RC are finally accepted, the owner-
approved [C12 implementation goal](docs/c12-implementation-goal.md) reopens C4
only for its bounded manual-driver prerequisite, then implements the historical
observation replay backend/API and dashboard in two sequential slices. That
follow-on authorizes no provider request, credential access, public deployment,
or claim of trading edge.

## Private local dashboard

The dashboard is an independent loopback process. Start the scanner API with
the dashboard origin explicitly allowed, then start the static UI server from
the repository root:

```text
go run ./cmd/scanner --trading-date YYYY-MM-DD --api-address 127.0.0.1:8080 --allow-origin http://127.0.0.1:4173
go run ./cmd/dashboard --address 127.0.0.1:4173 --api-origin http://127.0.0.1:8080 --assets ui
```

Open `http://127.0.0.1:4173` in Chrome. The UI polls the versioned snapshot once
per second with one request in flight. Restarting the dashboard does not stop
or relink the scanner. This is a private/local configuration; it does not add
public binding, authentication, TLS, hosting, or credentialed live validation.

## Predecessor evidence

The version 2 predecessor is retained separately as a source of Massive protocol
behavior, observed edge cases, fixtures, algorithms, and regression evidence. It
is evidence, never authority. It may be inspected only within a narrow,
owner-approved reconnaissance scope after the new component boundary has been
derived from Phase 1. Its orchestration architecture is not the starting point
for this implementation.
