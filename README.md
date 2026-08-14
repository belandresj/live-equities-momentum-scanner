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
- consume live one-second aggregates through one canonical event and state
  path;
- qualify symbols using the approved same-session aggregate-tape gate, then rank
  passers by Day % descending with exact-symbol tie-breaking;
- publish at most 20 rows with independently available Float, session share
  Volume, Last, Day %, From Open %, Day Range, Activity 30s, Move 30s, Tape 5s,
  and Spread;
- retain at least 330 seconds of aggregate/coverage evidence plus the
  predecessor mark for every display-eligible symbol, independent of top-20
  membership;
- recover from fresh start and same-process aggregate gaps without fabricating
  marks or treating successful empty hydration as unfinished work; and
- expose readiness, coverage, population accounting, and field availability
  honestly through a versioned read-only API.

The complete product contract is
[`docs/product/product-goals.md`](docs/product/product-goals.md).

## Architecture

The design has:

- one authoritative `ScannerStateEngine` and one canonical per-symbol session
  state;
- normalized aggregate, trade, quote, control, hydration, and timer inputs;
- one committed aggregate watermark and one qualification/ranking path;
- explicit bootstrap, live, recovery, suppression, session-end, and shutdown
  lifecycles;
- default T/Q coverage for displayed rows, with T/Q degraded before aggregate
  correctness;
- retained optional checkpoint and replay code that is outside the current MVP
  operating claim;
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
| [`docs/live-feature-mvp-program.md`](docs/live-feature-mvp-program.md) | Current delivery authority: incremental backend, API, UI, and integrated-live feature cutover; replay unverified and checkpoints disabled. |
| [`docs/architecture/system-overview.md`](docs/architecture/system-overview.md) | Runtime topology, component boundaries, state ownership, and failure containment. |
| [`docs/architecture/data-time-and-event-contract.md`](docs/architecture/data-time-and-event-contract.md) | Session, clock, event, ordering, coverage, reconciliation, replay, and checkpoint-cutoff semantics. |
| [`docs/architecture/scanner-state-engine-lifecycle.md`](docs/architecture/scanner-state-engine-lifecycle.md) | Legal engine states, transitions, publication permissions, recovery, and termination. |
| [`docs/glossary.md`](docs/glossary.md) | Shared vocabulary; the controlling product or architecture contract wins when more specific. |
| [`docs/specification-map.md`](docs/specification-map.md) | Current component sequence, document status, dependencies, and implementation milestones. |
| [`docs/implementation-process.md`](docs/implementation-process.md) | Contract-first research, approval, predecessor-reuse, proof, slice, integration, and release workflow. |
| [`docs/v1-release-program.md`](docs/v1-release-program.md) | Historical accepted delivery authority and reusable evidence for the former feature set; superseded where it conflicts with the current MVP. |
| [`docs/c12-implementation-goal.md`](docs/c12-implementation-goal.md) | Historical replay follow-on authority; not active and not an MVP gate. |
| [`docs/market-hours-validation.md`](docs/market-hours-validation.md) | Separately authorized post-RC live-provider observation procedure; pending by default. |
| [`docs/specifications/focused-component-spec-template.md`](docs/specifications/focused-component-spec-template.md) | Mandatory template and modular-layout rules for focused component contracts. |
| [`docs/history/`](docs/history/) | Non-authoritative Phase 1 drafting and review history. |

## How work advances

Phase 1 product and architecture contracts are owner-approved. Current component
status and the sequential implementation roadmap are maintained only in the
[`specification map`](docs/specification-map.md).

The current incremental plan is the
[`Live feature-set MVP program`](docs/live-feature-mvp-program.md): implement
the backend measurements and Float enrichment, cut the snapshot API to v2,
finish the dashboard, then verify the integrated live composition. Keep one
write-capable slice active. The numbered component contracts remain reusable
evidence; this MVP does not rename or comprehensively refactor them.

### Current live feature-set MVP

The owner approved the
[`Live feature-set MVP program`](docs/live-feature-mvp-program.md) on
2026-08-14. The only supported operating path in this cut is the ordinary
fresh-start live scanner. Deterministic unit, fake-provider, API, and UI
fixtures remain the normal proof and weekend-development tools; they are not a
claim that product replay works.

Replay implementation remains in the repository with unknown current
capability and no MVP acceptance role. Checkpoint persistence also remains but
is disabled; the MVP restarts through fresh hydration. An old checkpoint must
not be restored and presented as complete revised-feature state. The program
does not authorize credential access or provider requests, public deployment,
or any claim of trading edge.

## Private local dashboard

For ordinary private daily operation on macOS, use the supervised one-command
workflow documented in the
[`private live scanner runbook`](docs/private-live-scanner-runbook.md):

```text
./scripts/run-private-scanner
```

Start it at approximately 03:55 America/New_York. It builds and supervises the
existing scanner and dashboard and reports authoritative liveness and readiness
without moving market-state ownership into the launcher. For the current MVP,
checkpoint mode stays off and restart uses fresh hydration. Hydration defaults
to eight workers;
`--hydration-workers 1|2|4|8` selects a lower supported concurrency when
needed. The manual commands below remain useful for development and
independent-process inspection.

The dashboard is an independent loopback process. Start the scanner API with
the dashboard origin explicitly allowed, then start the static UI server from
the repository root:

```text
go run ./cmd/scanner --trading-date YYYY-MM-DD --api-address 127.0.0.1:8080 --allow-origin http://127.0.0.1:4173
go run ./cmd/dashboard --address 127.0.0.1:4173 --api-origin http://127.0.0.1:8080 --assets ui
```

Open `http://127.0.0.1:4173` in Chrome. The UI polls `GET /api/v2/snapshot`
once per second with one request in flight. Restarting the dashboard does not stop
or relink the scanner. This is a private/local configuration; it does not add
public binding, authentication, TLS, hosting, or credentialed live validation.

### Retained replay tooling

Historical replay code and commands remain in the repository, but their current
runnable capability has not been verified and they are not supported by the
live feature-set MVP. Use deterministic fixtures to develop formulas, the API,
and the UI while the market is closed. Do not describe fixture-driven tests as
product replay or use unknown replay behavior as an MVP acceptance gate.

## Predecessor evidence

The version 2 predecessor is retained separately as a source of Massive protocol
behavior, observed edge cases, fixtures, algorithms, and regression evidence. It
is evidence, never authority. It may be inspected only within a narrow,
owner-approved reconnaissance scope after the new component boundary has been
derived from Phase 1. Its orchestration architecture is not the starting point
for this implementation.
