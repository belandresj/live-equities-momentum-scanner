# Specification map

**Status:** Phase 1 scaffold; contents and authority remain subject to owner
review.

## Phase 1: shared product and architecture context

| Document | Purpose |
| --- | --- |
| `product/product-goals.md` | User-facing goals, developer-facing goals, product behavior, and non-goals. |
| `architecture/system-overview.md` | Complete data flow, component boundaries, ownership, and dependencies. |
| `architecture/data-time-and-event-contract.md` | Normalized events, clocks, watermarks, session boundaries, and ordering vocabulary. |
| `architecture/scanner-state-engine-lifecycle.md` | Authoritative lifecycle, state transitions, and failure-domain boundaries. |
| `glossary.md` | One shared definition for repository terms. |
| `plans/phase-1-decisions-brief.md` | Approved and proposed decisions that seed Phase 1 drafting. |

## Later component specifications

Phase 1 documents will link to focused specifications for:

1. universe and adjusted prior close;
2. Massive live market-data normalization;
3. canonical symbol state and feature extension;
4. hydration and recovery;
5. qualification and ranking;
6. top-20 T/Q coverage and pressure protection;
7. checkpoint and restart behavior;
8. deterministic aggregate replay;
9. readiness, accounting, and operations;
10. snapshot API and independently deployed UI; and
11. test strategy, evidence registry, differential replay, and cutover.

These specifications should constrain observable behavior, state ownership,
time semantics, boundedness, terminal outcomes, and failure policy. They should
not prescribe private helper names, exact file layouts, mechanical error
wrapping, or speculative provider behavior.

## Architecture decision records

`docs/decisions/` will contain short ADRs only for consequential choices whose
alternatives and tradeoffs should remain visible in the repository history.
Routine implementation choices do not require ADRs.
