# Independent UI

**Status:** Owner-approved V1 boundary and reconnaissance plan; detailed
contract pending after Component 10 final acceptance

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
UI, fixture, and slice decisions remain revisable until final V1 acceptance

**Controlling Phase 1 requirements:** `PG-RANK-04`, `PG-FEATURE-01`,
`PG-FEATURE-02`, `PG-FEATURE-03`, `PG-FEATURE-04`, `PG-AVAIL-01`,
`PG-AVAIL-02`, `PG-AVAIL-03`, `PG-UI-01`, `PG-UI-02`, `PG-OBS-03`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-COMMIT-04`, `LIFE-PUBLISH-01`, `LIFE-PUBLISH-02`, and
`LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Component 10 API and its stable V1
product meanings

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C11 boundary plan;
Sections 8-19 are completed here after just-in-time V2 UI reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C11 boundary, current detailed contract when added, proofs, slices, and sole delivery ledger | Sections 1-7 approved; Sections 8-19 pending | Every C11 task | Component 10 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `boundary_approved` | Direct owner V1 program revision, 2026-08-07 | Wait for C10 final acceptance |
| Completed contract | `pending` | Focused contract review only if a consequential C10 compatibility or ownership issue is introduced | Record exact V2 whitelist, UI states, visual baseline, proofs, and at most two slices |
| Implementation | `pending` | One final read-only component review and then one integrated V1 RC review | Begin after completed contract |

## 1-4. Outcome, scope, ownership, and settled boundary

C11 delivers an independently runnable Chrome-desktop dashboard that adapts
the useful V2 scanner layout, visual character, information density, and
interactions while presenting only C10-owned product meaning.

In scope:

- rank, symbol, Last, Day %, From 4AM %, HOD drawdown, session/30m/60m range
  position, Activity, Tape Rate, and NBBO Spread;
- process/backend/ranking status, stale/unavailable/invalid/warming field
  states and reasons, aggregate versus T/Q health, coverage, and bounded
  operational context;
- fewer-than-20 rows and exact API order without client resorting;
- a high-fidelity Chrome-desktop adaptation of useful V2 layout, density,
  visual character, scanning hierarchy, and interactions;
- keyboard/focus, semantic table/status, contrast, reduced-motion where
  relevant, loading/error/reconnect behavior, and an independent local run
  against configured C10 origin.

Not in scope: browser-owned market calculations, ranking, readiness, field-
status inference, state reconciliation, direct backend internals, mobile,
multi-browser certification, public deployment, authentication/TLS, PWA/offline
mode, or a generalized design system.

The browser owns presentation, interaction, transient view preferences, and
transport retry display. C10 owns every market value, order, publication
identity, readiness, coverage, and field status. UI deployment and restart
cannot restart or relink the backend.

The component introduces no ranking key, market-time/window interpretation,
readiness rule, or mutable scanner owner.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must identify the useful V2 screen regions,
interactions, visual assets/tokens, density, field formatting, accessibility
gaps, obsolete status semantics, backend coupling, build/runtime boundary, and
the smallest deterministic screenshot/state fixture set.

After C10 final acceptance, reconnaissance may inspect the V2 UI specification,
UI code, assets, and focused UI tests only. This exact category is owner-
approved. Reject browser calculations/readiness, obsolete lifecycle/field
states, direct backend coupling, unrelated application screens, deployment
infrastructure, credentials, and server code beyond C10 fixtures.

Likely primary proofs:

1. one deterministic API-to-view state matrix covering complete, fewer-than-20,
   exact empty, stale, unavailable, warming, invalid, T/Q-degraded, recovery,
   API error, and publication change without client recomputation; and
2. one Chrome-desktop visual/interaction/independence acceptance covering the
   approved layout baseline, density, key interactions, accessibility, and
   backend-continuity during UI restart.

**Provisional slices:** At most two: `C11-S1` C10 integration and status/view
states; `C11-S2` high-fidelity visual/interaction/accessibility completion.
Combine them if one coherent implementation and proof remains reviewable.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.
