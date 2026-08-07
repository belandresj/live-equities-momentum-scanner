# Top-20 T/Q coverage and features

**Status:** Owner-approved V1 boundary and reconnaissance plan; detailed
contract pending after Component 8 final acceptance

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
decisions remain revisable through the program correction loop

**Controlling Phase 1 requirements:** `PG-FEATURE-03`, `PG-FEATURE-04`,
`PG-FEATURE-05`, `PG-AVAIL-01`, `PG-AVAIL-03`, `PG-TAQ-01`,
`PG-TAQ-02`, `PG-TAQ-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-03`, `DTE-WINDOW-01`, `DTE-WINDOW-03`, `DTE-TRADE-01`,
`DTE-TRADE-02`, `DTE-QUOTE-01`, `DTE-QUOTE-02`, `DTE-CONTROL-01`,
`DTE-TQ-01`, `DTE-TQ-02`, `DTE-TQ-03`, `DTE-REJECT-01`,
`LIFE-TQ-01`, `LIFE-TQ-02`, `LIFE-TQ-03`, `LIFE-PUBLISH-02`, and
`LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Components 1-8, including C5
normalized T/Q/control facts and C8 aggregate/runtime pressure measurements

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C9 boundary plan;
Sections 8-19 are completed here after just-in-time V2 reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C9 boundary, current detailed contract when added, proofs, slices, and sole delivery ledger | Sections 1-7 approved; Sections 8-19 pending | Every C9 task | Components 1-8 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `boundary_approved` | Direct owner V1 program revision, 2026-08-07 | Wait for C8 final acceptance |
| Completed contract | `pending` | Focused review expected for T/Q causal coverage, ordering, and aggregate-independence boundaries | Finalize conditions, NBBO validity/coverage, pressure thresholds, proofs, and at most two slices |
| Implementation | `pending` | Narrow review only for consequential ordering/concurrency boundaries; one final read-only review | Begin after completed contract |

## 1-4. Outcome, scope, ownership, and settled boundary

C9 normally enriches every current qualified displayed row with continuous
trade and quote coverage, Tape Rate, and NBBO Spread. When load threatens
aggregate processing, it sheds all T/Q work if necessary and later restores
coverage in current aggregate rank order with fresh warm-up.

In scope:

- derive desired membership from current `qualified_current` rows only;
- correlate paired T/Q commands and acknowledgements with current epoch and
  begin per-channel causal coverage only after acknowledged boundaries;
- implement Tape Rate's five-second and one-second qualifying-original trade
  rates with evidenced conditions and explicit lifecycle limitations;
- implement 10-second time-weighted median validated NBBO spread in cents and
  basis points, including locked-zero and unavailable crossed/one-sided/stale/
  insufficient-coverage behavior;
- clear measurements across gaps, maintain continuous warm-up, report coverage
  and field status, and add no Tape Rate attention threshold;
- reject expensive T/Q before aggregate/control, permit complete T/Q
  unsubscription, and restore gradually in current rank order.

Not in scope: aggregate qualification/ranking/readiness, full-universe T/Q,
full-session T/Q replay, public API/UI, an unevidenced complete trade-lifecycle
claim, a Tape Rate trading label, or provider credentials/live observation.

The engine remains the sole owner of desired membership, acknowledged causal
coverage, feature state, pressure state, and field availability. The C5 adapter
continues to normalize/read/control the shared socket and returns facts. No T/Q
fact can change aggregate `T`, qualification, rank, or backend readiness.

The component introduces no new ranking key, threshold-based trading signal,
watermark, evaluator, time-window meaning, or competing mutable owner.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must resolve the exact trade-condition fixture,
lifecycle completeness disclosure, NBBO quote-validity and coverage minimum,
time-weighting boundary, pressure measurements/thresholds/hysteresis, command
restoration pace, bounded retention, and compact proofs.

After C8 final acceptance, reconnaissance may inspect only V2 T/Q condition
fixtures and classification, Tape Rate/Spread calculators, acknowledgement and
coverage tests, pressure/shedding/restoration behavior, and directly used
focused fake-provider fixtures. Reject full-universe selection, T/Q-to-ranking
coupling, browser calculations, old health/readiness owners, broad replay,
credentials, and live requests.

Likely primary proofs:

1. one deterministic selected-membership/acknowledgement/gap/feature trace that
   proves Tape Rate, Spread, continuous coverage, independent status, and
   current-rank membership; and
2. one controlled pressure trace showing classification preserves aggregate/
   control facts, T/Q can shed to zero, aggregate state/rank/readiness remain
   unchanged, and restoration follows current rank order with fresh warm-up.

**Provisional slices:** At most two: `C9-S1` membership, coverage, and features;
`C9-S2` pressure, complete shedding, and restoration. These can be revised
without owner approval when evidence remains inside the fixed boundary.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.
