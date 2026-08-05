# Live Equities Momentum Scanner

Live Equities Momentum Scanner is a planned production-grade, real-time U.S.
equities scanner for a discretionary momentum trader. It will rank qualified
stocks by return from the adjusted previous regular-session close, display
aggregate-derived context, and provide trade- and quote-derived measurements
for the displayed top 20.

This repository is intentionally starting with product and architecture review.
Production implementation is not yet authorized.

## Current status

**Phase 1: high-level product and architecture specification.**

The authoritative drafting input is
[`docs/plans/phase-1-decisions-brief.md`](docs/plans/phase-1-decisions-brief.md).
The planned specification hierarchy is in
[`docs/specification-map.md`](docs/specification-map.md).

## Design direction

The scanner will have:

- one authoritative `ScannerStateEngine`;
- normalized aggregate, trade, and quote events;
- one canonical per-symbol session state;
- explicit bootstrap, checkpoint catch-up, live, and recovery lifecycles;
- one qualification and ranking path;
- default trade and quote coverage for the displayed top 20, with aggregate-
  protecting load shedding;
- coherent version 1 checkpoints;
- deterministic offline aggregate replay;
- a versioned read-only backend API; and
- an independently deployable UI.

The predecessor repository is retained separately as a source of product
contracts, Massive protocol behavior, observed edge cases, fixtures, and
regression evidence. Its orchestration architecture is not the starting point
for this implementation.

## Documentation

- [Product goals](docs/product/product-goals.md)
- [System overview](docs/architecture/system-overview.md)
- [Data, time, and event contract](docs/architecture/data-time-and-event-contract.md)
- [Scanner State Engine lifecycle](docs/architecture/scanner-state-engine-lifecycle.md)
- [Glossary](docs/glossary.md)
- [Specification map](docs/specification-map.md)
- [Phase 1 decisions brief](docs/plans/phase-1-decisions-brief.md)
