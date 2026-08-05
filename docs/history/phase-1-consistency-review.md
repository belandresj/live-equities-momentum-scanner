# Phase 1 consistency review

**Status:** Accepted Phase 1 completion review.

**Approved:** 2026-08-05

## Conclusion

**Phase 1 is complete and owner-approved.** The approved product and
architecture contracts form one consistent answer to the Phase 1 readiness
questions: what the scanner promises, where data originates, who owns mutable
state, which clock and order are authoritative, how each live/replay lifecycle
progresses, how no-print work terminates, how aggregate ranking survives T/Q
failure, what checkpoints and snapshots mean, why the UI remains independent,
and which focused contracts still precede implementation.

No substantive conflict between approved documents was found. No focused
component detail blocks the Phase 1 architecture. Production implementation
remains unauthorized until the applicable focused specifications and required
ADRs are approved.

## Documents reviewed

| Document | Status used in this review |
| --- | --- |
| [`README.md`](../../README.md) and [`AGENTS.md`](../../AGENTS.md) | Repository context and instructions; not product/architecture authorities. |
| [`product-goals.md`](../product/product-goals.md) | Approved product contract; highest authority. |
| [`system-overview.md`](../architecture/system-overview.md) | Approved architecture contract for topology and ownership. |
| [`data-time-and-event-contract.md`](../architecture/data-time-and-event-contract.md) | Approved architecture contract for identity, time, ordering, coverage, reconciliation, replay, and checkpoint cutoffs. |
| [`scanner-state-engine-lifecycle.md`](../architecture/scanner-state-engine-lifecycle.md) | Approved architecture contract for legal operational states and transitions. |
| [`glossary.md`](../glossary.md) | Approved shared vocabulary; summary only. |
| [`specification-map.md`](../specification-map.md) | Approved Phase 1 specification index. |
| [`phase-1-decisions-brief.md`](phase-1-decisions-brief.md) | Historical drafting record; non-authoritative. |
| Architecture decision records | No ADR is currently present or accepted. |

Targeted read-only predecessor evidence was also consulted. The
`bootstrap-recovery-first-principles-architecture-review.md` investigation
supports the terminal no-print/deferred-state failure described by the system
overview and the need for one ordinary evaluator. The earlier predecessor
checkout was searched only for the historical Average Trade Size semantics.
Both predecessors remain evidence, not authority; no predecessor file was
changed, and no live process was queried.

## Consistency matrix

| Dimension | Result | Strongest evidence |
| --- | --- | --- |
| Product behavior | Consistent | `PG-UNIVERSE-01` through `PG-RANK-05` fix universe/reference policy, qualification and latch, Day-%/symbol order, top 20/fewer-than-20 behavior, and permanent `degraded_bootstrap`. `PG-TAQ-*`, `PG-OPS-*`, `PG-REPLAY-*`, and `PG-UI-*` fix version 1 T/Q, checkpoints, aggregate-only replay, and independent UI behavior. |
| State ownership | Consistent | `ARCH-OWN-*` and `LIFE-MODEL-*` define one `ScannerStateEngine`, one canonical per-symbol state, fact-returning adapters/workers, immutable snapshot/checkpoint projections, and no competing evaluator or watermark. |
| Time and ordering | Consistent | `DTE-SESSION-*`, `DTE-CLOCK-*`, `DTE-WINDOW-*`, `DTE-EVENT-*`, `DTE-MERGE-*`, `DTE-RECOVERY-*`, `DTE-REPLAY-*`, and `DTE-CHECKPOINT-*` define half-open session/time semantics, causal positions, epochs/generations, live/REST identity and precedence, committed `T`, deterministic replay order, and coherent `T0`. |
| Lifecycle | Consistent | The complete state vocabulary, state summary, input-permission matrix, `LIFE-T01`–`LIFE-T29`, and progress invariants cover fresh/checkpoint startup, hydration, live, recovery, replay, suppression, session end, and shutdown. Valid empty work is terminal, every wait has a progress event, and accepted aggregates retain a path to ordinary evaluation. |
| Failure domains | Consistent | System-overview section 8 and `LIFE-RECOVER-*`, `LIFE-TQ-*`, `LIFE-SUPPRESS-*`, and `LIFE-LIVE-04/05` isolate T/Q, checkpoint, API/UI, replay, and symbol-local failures while reserving suppression for genuinely global aggregate/integrity ambiguity. |
| Accounting and observability | Consistent | `PG-OBS-*`, system-overview section 9, and `LIFE-PUBLISH-*` keep primary symbol and terminal-work identities exact; label qualification, historical fields, Activity, and T/Q as overlapping; and separate process live, backend ready, ranking current, and field currentness. |
| Scope | Consistent | Product non-goals, system-overview sections 10/12/14, and lifecycle section 17 exclude speculative infrastructure and delegate only exact provider, capacity, storage, API, operations, and proof details to named focused specifications. |

The approved historical Average Trade Size mapping is explicitly approximate:
live and REST mappings retain different provenance, and no test may require
their numerical equality. This review did not convert that approximation into
a parity claim. Provider fixtures and quantitative impact evidence remain part
of the normalized-event/provider and aggregate-feature proof work.

## Synchronization corrections made

- Updated `README.md` so the historical decisions brief no longer appears to be
  the current authority.
- Replaced the specification-map scaffold with actual document statuses, the
  authority order, cross-links, settled Phase 1 concerns, focused-specification
  dependencies, and explicit post-interface parallelism.
- Reconciled the glossary with product, ownership, time, lifecycle, readiness,
  checkpoint/snapshot, T/Q, and failure-domain terminology. Corrected the
  population-accounting summary so overlapping dimensions are not presented as
  mutually exclusive primary bins.
- Reclassified the decisions brief as non-authoritative history, added an
  indexed resolution table, and synchronized its terminal-work names from
  `value/empty/canceled_or_deferred` to
  `completed_value/completed_empty/canceled`.
- Added this review artifact. No approved product or architecture requirement
  was changed during synchronization; the glossary edits summarize those
  controlling requirements.

## Substantive conflicts and blockers

None found. No approved clause was silently selected over another, and no owner
decision remains necessary to make the Phase 1 set internally coherent.

## Correctly deferred work

Before production implementation, focused specifications must still settle:

1. universe/schedule/session-binding retrieval and cache mechanics;
2. exact normalized-event and engine boundary schemas;
3. canonical symbol state and feature-extension interfaces;
4. Massive live/control normalization and evidenced fixtures;
5. hydration/recovery bounds, exact gap overlap, terminal results, and merge;
6. aggregate feature algorithms, qualification/ranking projection, and exact
   accounting interfaces;
7. T/Q conditions, causal coverage, feature warm-up, pressure thresholds, and
   restoration;
8. checkpoint contents, cadence, encoding, atomic storage, and restart target;
9. replay artifact/controls plus retention, privacy, and provider-license
   policy;
10. readiness thresholds, diagnostics, shutdown, and operations;
11. versioned snapshot API, update transport, schema evolution, UI integration,
    and predecessor UI reuse; and
12. the evidence registry, primary proof allocation, differential replay,
    shadowing, and cutover thresholds.

Implementation language, packages, private data structures, queue sizes, and
build tooling remain lower-level choices unless a focused contract or ADR makes
their tradeoff consequential. Deferral does not authorize changing any Phase 1
product or architecture invariant.

## Proposed completion checklist

- [x] Product goals are approved and identify version 1 behavior/non-goals.
- [x] System, data/time/event, and lifecycle architecture contracts are
  approved and mutually consistent.
- [x] Specification map, glossary, and drafting-history status are synchronized.
- [x] Phase 1 links, lifecycle vocabulary, readiness terms, terminal outcomes,
  accounting identities, and scope boundaries are reviewed.
- [x] No substantive approved-document conflict or Phase 1 blocker remains.
- [x] No production code, tests, provider integration, runtime infrastructure,
  credentials, live scanner, or predecessor file was changed.
- [x] Owner approves the synchronized Phase 1 set and this consistency
  conclusion.
- [x] The Phase 1 completion commit records the approved contracts and review.

Recommended completion commit:

```text
docs: complete phase 1 product and architecture contracts
```
