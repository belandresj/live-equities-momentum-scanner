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
  Volume, Last, From Close %, From Open %, Day Range, Activity 30s, Move 30s,
  Tape Speed, and Spread;
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

The approved live-backend replacement has:

- one authoritative `ScannerStateEngine` and one canonical per-symbol session
  state;
- a compact sealed session prefix plus sparse correction tail;
- one single-pass Massive decoder and one bounded decoded-batch live handoff;
- one committed aggregate watermark and one qualification/ranking path;
- incremental qualification and one-second two-phase selection/enrichment;
- default T/Q coverage for displayed rows, with T/Q degraded before aggregate
  correctness;
- fresh/gap hydration through the same canonical state and exact ingress fence;
- replay/checkpoint state excluded from the supported live core;
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
| [`docs/live-backend-replacement.md`](docs/live-backend-replacement.md) | Current architecture authority: compact live state, two-phase evaluation, one ingress handoff, retained hydration/recovery, resource policy, and removal boundary. |
| [`docs/live-backend-replacement/delivery-program.md`](docs/live-backend-replacement/delivery-program.md) | Current delivery authority: focused-contract gate, five sequential capabilities, agent/review policy, and integrated acceptance. |
| [`docs/live-feature-mvp-program.md`](docs/live-feature-mvp-program.md) | Historical accepted feature/API/UI/stability delivery evidence; superseded as the active program. |
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

The product contract and live-backend replacement architecture are
owner-approved. Current status and the sequential roadmap live in the
[`replacement delivery program`](docs/live-backend-replacement/delivery-program.md)
and [`specification map`](docs/specification-map.md).

Capabilities A–C and the inserted `LBR-A3` bounded parallel hydration slice are
implemented and accepted for their recorded semantics. Ordinary-live hydration
accepts exactly `1|2|4|8` workers and defaults to 8. D1 is next but remains
inactive until its fresh pre-assignment audit.
Historical numbered components and stability corrections remain evidence, not
requirements to preserve their private representations.

The only supported operating path remains the ordinary fresh-start live
scanner. Replay capability is unknown and checkpoint persistence is disabled.
Deterministic fixtures remain proof tools, not product replay. No current
document authorizes credentials, provider requests, public deployment, or a
trading-edge claim.

## Private local dashboard

For ordinary private daily operation on macOS, use the supervised one-command
workflow documented in the
[`private live scanner runbook`](docs/private-live-scanner-runbook.md):

```text
./scripts/run-private-scanner
```

It may be started at any time. Before the 03:55 America/New_York preconnect
boundary it runs the dashboard in calendar-aware standby, then starts the
ordinary scanner for the current or next exchange-declared trading session.
Starting at approximately 03:55 remains the shortest path. It builds and
supervises the existing scanner and dashboard and reports authoritative
liveness and readiness without moving market-state ownership into the launcher. For the current MVP,
checkpoint mode stays off and restart uses fresh hydration. Hydration defaults
to eight workers; `--hydration-workers` accepts exactly `1`, `2`, `4`, or `8`.
The manual commands below remain useful for development and
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

The cache-only aggregate replay path has a separately authorized local
verification for the retained 2026-08-07 artifact. It fast-forwards to the
CLI-declared observation start, then automatically continues at 1x through the
same API and dashboard; there is no replay-specific frontend control. Exact
commands and measured startup behavior are recorded in
[`docs/replay-warmup-acceleration.md`](docs/replay-warmup-acceleration.md).
The accelerated warm-up passes its local bound, but total preparation still
exceeds the focused correction's target because the accepted trust path scans
the complete artifact twice; the focused correction therefore remains short of
full acceptance.

Replay remains separate from and non-gating for the live feature-set MVP, and
it does not reconstruct T/Q or prove live receipt chronology.

## Predecessor evidence

The version 2 predecessor is retained separately as a source of Massive protocol
behavior, observed edge cases, fixtures, algorithms, and regression evidence. It
is evidence, never authority. It may be inspected only within a narrow,
owner-approved reconnaissance scope after the new component boundary has been
derived from Phase 1. Its orchestration architecture is not the starting point
for this implementation.
