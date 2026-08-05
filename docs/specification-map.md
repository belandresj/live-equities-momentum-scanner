# Specification map

**Status:** Approved Phase 1 specification index.

**Approved:** 2026-08-05

This map reports the owner-approved Phase 1 document set and the required
pre-implementation specification sequence. Document status remains explicit;
this index does not grant approval to a later focused specification.

## Authority

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. accepted architecture decision records (ADRs);
4. approved focused component specifications; and
5. implementation plans and code.

Within the approved architecture set, the
[`system overview`](architecture/system-overview.md) controls topology and
ownership, the
[`data, time, and event contract`](architecture/data-time-and-event-contract.md)
controls identity, clocks, ordering, coverage, and reconciliation, and the
[`Scanner State Engine lifecycle`](architecture/scanner-state-engine-lifecycle.md)
controls legal operational transitions. None may weaken the
[`product contract`](product/product-goals.md). If two approved documents
conflict substantively, implementation stops for the smallest owner decision;
the drafting brief is never used as a tie-breaker.

## Phase 1: shared product and architecture context

| Document | Current status | Role and controlling links |
| --- | --- | --- |
| [`product/product-goals.md`](product/product-goals.md) | Approved product contract, 2026-08-05 | Product promises, formulas, version 1 scope, priorities, and non-goals; highest authority. |
| [`architecture/system-overview.md`](architecture/system-overview.md) | Approved architecture contract, 2026-08-05 | Runtime topology, sole state ownership, component boundaries, failure containment, checkpoint/snapshot/API/UI boundaries, and deeper-specification list. Depends on the product contract. |
| [`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md) | Approved architecture contract, 2026-08-05 | Session binding, normalized-event meaning, half-open windows, causal order, live/REST merge, hydration evidence, committed watermark, replay order, and checkpoint cutoff. Depends on product behavior and system ownership. |
| [`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md) | Approved architecture contract, 2026-08-05 | Legal live/replay lifecycle states and transitions, progress/exit conditions, publication permissions, recovery, suppression, session end, and shutdown. Depends on the product, system, and data/time contracts. |
| [`glossary.md`](glossary.md) | Approved shared vocabulary, 2026-08-05 | Concise definitions only. The controlling product or architecture contract wins if a summary is less specific. |
| [`history/phase-1-decisions-brief.md`](history/phase-1-decisions-brief.md) | Historical drafting record; non-authoritative | Preserves design history and indexes each former open decision as resolved, deliberately deferred, or blocking. |
| [`history/phase-1-consistency-review.md`](history/phase-1-consistency-review.md) | Accepted Phase 1 completion review; not an authority | Records the synchronization review, corrections, conflicts, deferrals, and completion checklist. |

No ADR is currently present or accepted. The ADR directory should be created
when the first consequential architecture decision is proposed.

## Concerns settled by Phase 1

The approved contracts settle:

- the eligible universe, adjusted-prior-close basis, qualification gate and
  session latch, exact Day-%/symbol ordering, top-20 cardinality, and permanent
  `degraded_bootstrap` behavior;
- version 1 displayed fields, default displayed-row T/Q coverage, mandatory
  coherent checkpoints, aggregate-only replay, and independent UI deployment;
- one authoritative `ScannerStateEngine`, one canonical per-symbol state, one
  committed aggregate watermark, one ordinary evaluator, and fact-returning
  adapter/worker boundaries;
- the 04:00–20:00 New York session binding, half-open intervals, event/receipt/
  engine/replay time, source positions, epochs/generations, REST/live identity
  and precedence, no-print proof, snapshot identity, and checkpoint cutoff;
- legal initialization, bootstrap, live, recovery, replay, suppression,
  session-end, and controlled-shutdown transitions, including the rule that
  terminal hydration always returns to ordinary evaluation; and
- aggregate/T/Q/checkpoint/API/UI/replay failure domains, global versus local
  integrity handling, mutually exclusive primary accounting, terminal work
  accounting, independent availability, and bounded observability.

## Later component specifications

Phase 1 authorizes further specification, not implementation. Focused contracts
must be approved in this dependency order. Add substantive Phase 2 drafts under
`docs/specifications/` as they are started; this map remains their status and
dependency index, so the directory does not need a placeholder README. The
order 6 test strategy belongs with those focused contracts unless its approved
evidence-registry design later requires a separate documentation area.

| Order | Focused specification | Requires | May unlock or run in parallel |
| --- | --- | --- | --- |
| 1A | Universe, exchange schedule, session binding, and adjusted prior close | Phase 1 contracts | Can be written in parallel with 1B. |
| 1B | Exact normalized-event and engine-input/output interfaces | Phase 1 data/time and lifecycle contracts | Can be written in parallel with 1A; unlocks provider, hydration, T/Q, and replay specifications. |
| 2A | Canonical symbol state and feature-extension boundary | 1A and 1B | Establishes the shared correction/coverage, currentness, selected-row, and checkpoint-input interfaces. |
| 2B | Massive live adapter, control protocol, and provider fixtures | 1A and 1B | Can be written in parallel with 2A once normalized interfaces are fixed. |
| 3A | Hydration, gap recovery, terminal outcomes, and REST/live merge policy | 1A, 1B, and 2A | Can proceed in parallel with 3B and 3C; constrains checkpoint restart and readiness. |
| 3B | Aggregate features, qualification, ranking, and exact accounting | 2A shared interfaces | Can proceed in parallel with 3A and 3C; must remain subordinate to product formulas. |
| 3C | Top-20 T/Q coverage, conditions, measurements, pressure, and warm-up | 1B and 2A shared interfaces | Can proceed in parallel with 3A and 3B. |
| 4A | Checkpoint contents, validation, cadence, atomic storage, and restart target | 1A, 2A, 3A, and restart requirements from 3B | Can be authored alongside replay; approval waits for all retained feature inputs. |
| 4B | Deterministic aggregate replay artifact, clock controls, retention, and provider-data restrictions | 1A, 1B, and 2A; product-output proof later uses 3B | Can be authored alongside checkpoint and T/Q work. |
| 5A | Readiness thresholds, bounded diagnostics, operations, and shutdown | 2B, 3A, 3C, and 4A status interfaces | Supplies the final backend-ready and operational reason contract. |
| 5B | Versioned snapshot API and independently deployed UI integration | 3B, 3C, and 5A publication meanings | API schema precedes UI transport/reuse decisions; UI work can then proceed independently. |
| 6 | Test strategy, evidence registry, differential replay, shadowing, and cutover | All applicable focused contracts | Proof planning should accompany every specification; final acceptance/cutover criteria close after the interfaces above. |

After 1A, 1B, and 2A approve the shared binding, normalized-event, engine, and
canonical-state interfaces, 2B, 3A, 3B, 3C, and 4B may be written in parallel.
The checkpoint contract 4A can be drafted alongside them but cannot close until
3A and 3B identify every recovery and retained-feature input. Central
engine/event/API contracts retain one integration owner. Implementation
language, exact packages, build tooling, queue sizes, private data structures,
and other delegated mechanics remain lower-level choices unless a focused
contract or accepted ADR makes them consequential.

These specifications should constrain observable behavior, state ownership,
time semantics, boundedness, terminal outcomes, and failure policy. They should
not prescribe private helper names, exact file layouts, mechanical error
wrapping, or speculative provider behavior.

## Architecture decision records

When needed, `docs/decisions/` will contain short ADRs only for consequential
choices whose alternatives and tradeoffs should remain visible in the
repository history. Routine implementation choices do not require ADRs.
