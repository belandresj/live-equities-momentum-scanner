# Versioned snapshot API

**Status:** Owner-approved V1 boundary and reconnaissance plan; detailed
contract pending after Component 9 final acceptance

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
schema and proof decisions remain revisable until final V1 acceptance

**Controlling Phase 1 requirements:** `PG-AVAIL-01`, `PG-AVAIL-02`,
`PG-AVAIL-03`, `PG-UI-01`, `PG-UI-02`, `PG-OBS-01`, `PG-OBS-02`,
`PG-OBS-03`, `ARCH-OWN-02`, `ARCH-OWN-03`, `ARCH-OWN-04`,
`ARCH-FLOW-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`, `DTE-COMMIT-04`,
`DTE-CHECKPOINT-03`, `LIFE-LIVE-05`, `LIFE-PUBLISH-01`,
`LIFE-PUBLISH-02`, and `LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Components 1-9 immutable snapshot,
field-status, accounting, readiness, checkpoint, and T/Q meanings

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C10 boundary plan;
Sections 8-19 are completed here after just-in-time reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C10 boundary, current detailed contract when added, proofs, slices, and sole delivery ledger | Sections 1-7 approved; Sections 8-19 pending | Every C10 task | Components 1-9 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `boundary_approved` | Direct owner V1 program revision, 2026-08-07 | Wait for C9 final acceptance |
| Completed contract | `pending` | Focused review expected for publication identity, schema compatibility, and HTTP trust boundary | Record the current minimum V1 schema and one/two-slice proof plan; keep lower-level details revisable through final V1 acceptance |
| Implementation | `pending` | Narrow identity/trust review if needed; one final read-only review | Begin after completed contract |

## 1-4. Outcome, scope, ownership, and settled boundary

C10 exposes the current immutable scanner publication through a private,
versioned, read-only HTTP API sufficient for the independent V1 dashboard.

In scope:

- an explicit schema version and compatibility policy;
- publication identity distinct from committed watermark, binding/session
  identity, `generated_at`, committed `T`, ranking/readiness status and reason,
  exact rows, per-field status/reason, aggregate and T/Q coverage, symbol/work
  accounting, checkpoint status, and bounded C8 operational measurements;
- snapshot-consistent serialization from one immutable publication only;
- process-live versus backend-ready HTTP behavior without recomputing state;
- loopback binding by default and a configured explicit CORS allow-list with
  deny-by-default behavior.

Not in scope: feature/ranking/readiness calculation, mutable engine access,
public authentication, TLS, hosting, public CORS, service decomposition,
database/session state, production cutover, or an optional streaming transport
unless the compact V1 dashboard demonstrably requires it.

The engine owns product state and atomically publishes the immutable snapshot.
The C10 mapper/handlers own only versioned representation and read-only HTTP
behavior. A slow or invalid client cannot block or mutate scanner state.

Committed watermark is not publication identity. Missing/stale/unavailable
fields remain explicit states rather than zero/default JSON. Internal Go
structs do not automatically become API fields. The component introduces no
market calculation, readiness owner, ranking rule, or second snapshot.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must choose the minimum JSON shape, numeric/timestamp
encoding, compatibility rule, readiness/status mapping, response/body bounds,
timeout behavior, cache semantics if any, and exact origin matching. Polling is
the default simplest update transport unless evidence requires otherwise.

After C9 final acceptance, reconnaissance may inspect only V2 scanner API
schemas/handlers, serialization tests, status/error behavior, update transport,
and the exact UI-facing fixtures needed to assess useful compatibility. Reject
V2 browser/backend coupling, browser calculations, mutable owner access,
deployment/authentication/TLS code, generic API frameworks, credentials, and
unrelated endpoints.

Likely primary proofs:

1. a golden plus semantic-mutation schema proof that preserves publication,
   binding, time, rows, accounting, readiness, coverage, and independent field
   status from one immutable snapshot; and
2. an HTTP trust-boundary proof for methods, routes, loopback default, CORS
   allow/deny behavior, response bounds, coherent concurrent reads, and client
   cancellation without engine impact.

**Provisional slices:** One if schema mapping and HTTP are a coherent change;
at most two (`C10-S1` schema/identity, `C10-S2` HTTP/CORS) if the completed
contract shows distinct trust and proof boundaries.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.
