# ScannerStateEngine and canonical state — S1 engine kernel

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Atomic Component 1 binding installation, bounded
immutable admission, one FIFO owner/consumer, complete S1 disposition order,
and the S1 trust/accounting/proof boundary

**Controlling requirements:** `ENG-RUN-01`, `ENG-ADMIT-01`, `ENG-ORDER-01`;
`ARCH-OWN-01`–`ARCH-OWN-04`, `ARCH-FLOW-01`–`ARCH-FLOW-04`,
`DTE-SESSION-02`, `DTE-SESSION-04`, `DTE-EVENT-01`–`DTE-EVENT-04`,
`LIFE-MODEL-01`, `LIFE-MODEL-02`, `LIFE-INIT-01`, `LIFE-INIT-02`,
`LIFE-T01`, `LIFE-T02`, `LIFE-T05`

**Allocated slices:** `S1` only

**Document dependencies:** Parent; approved Component 1
[`reference.Binding`](../reference-data-and-session-binding.md#9-detailed-semantic-inputs-outputs-and-owned-state)

**Approval state:** Inherits the parent contract approval; not independently approved

**Implementation status:** `S1` implemented and owner-accepted 2026-08-05
after its three allocated primary proofs, required independent review,
correction of the review's FIFO lifecycle-ordering finding, targeted re-review,
race verification, and repository-wide verification. `S2` was subsequently
authorized, implemented, independently reviewed, and owner-accepted through
its separate bounded assignment.

## S1 outcome

One engine kernel accepts exactly one trusted immutable session binding,
installs all copied reference/symbol state atomically, and owns a bounded
immutable FIFO admission path. Successful FIFO linkage is the ownership and
admission-result linearization point. One consumer assigns contiguous engine
sequence and gives every admitted S1 fact exactly one complete disposition.

S1 does not expose the final immutable publication boundary, accept aggregates
or timers, implement the remaining lifecycle graph, or own `T`. Those are real
deferred behaviors, not placeholders.

## Semantic inputs, state, and construction guarantees

| Item | Normative meaning | Bound/owner |
| --- | --- | --- |
| Construction | Explicit `live` or `replay` run mode, injected clock, and finite `C/R` with `1 <= R < C`. `H=16m` is fixed; `D` has no production default. | Immutable per engine; invalid construction returns no engine. |
| Unbound shell | Queue synchronization, clock reference, run mode, empty fixed counters, private lifecycle `initializing`, no binding, symbols, `T`, or market claim. | Controlled close remains possible. The externally readable initial/unavailable view is added and proved by S4. |
| Binding evidence | Exact exported concrete `reference.Binding` constructed by Component 1, with private fields and defensive accessors. An exact value copy is the same fact; DTO/interface/JSON/map/caller digest is not. | Accepted at most once while open, unbound, and `initializing`; envelope and payload identity must match. |
| Install candidate | Complete engine-private copy of sorted symbols, prior-close facts/accounting, exact-symbol index, and zeroed core records, built after validation and off active state. | One owner pointer/state swap installs it; any failure discards it with no reachable partial state. Process OOM is not claimed recoverable. |
| Closed S1 payload family | Binding-install and bounded engine control/close variants only. Every node is a self-contained immutable value with versioned kind/schema, required identity/context, and admission-time sample. | No raw/opaque bytes, mutable aliases, credentials, URLs, errors, or unbounded batches. Later families require approved compiled variants and bounds. |
| Admission transfer | Before linkage the caller owns/can cancel. Under one serialization, validate size/family, deep-copy all reference-bearing fields, reserve a future sequence, sample the clock once, link the node, and commit `admitted`. | Link plus result commitment is the linearization point. Mutation/cancel after it cannot retract or alter the node. |
| Admission result | Exactly `admitted`, `not_admitted_invalid`, `not_admitted_canceled`, `not_admitted_closed`, `pressure_shed_optional`, or `sequence_budget_exhausted`. | Required facts block cancellation-aware and may use all `C`; future optional T/Q may use only `C-R`. Nonadmission gets no node or engine sequence. |
| FIFO/order | Linkage order alone is mutation order. Dequeue assigns the next positive `engine_sequence`; one transition at a time. | Worker completion, event/receipt time, caller/source sequence across unlike facts, map order, and scheduling cannot reorder. |
| Sequence lifetime | At most `math.MaxUint64-1` external admissions; `math.MaxUint64` is reserved for the internal fail-closed exhaustion transition. | No wrap/reuse. Each admitted node owns a reserved future slot. |
| Seal/drain | Close, controlled stop, and later session end atomically prevent new linkage and capture the greatest admitted ordinal. | Every node through the boundary completes a disposition. Blocked admission wakes; later results return exact nonadmission. |

The concrete binding type, off-state candidate, private mutable graph,
self-contained queue nodes, single linkage point, and single consumer prevent
external binding reconstruction, partial install visibility, post-admission
caller mutation, node retraction, a second sequence authority, or an alternate
mutation path by construction. Runtime must still reject representable
zero/partial concrete values, wrong identity, invalid accounting, unsupported
schema, illegal state, cancellation, close, and exhaustion.

## Binding installation acceptance

One precommit validation must establish all of the following:

1. the payload is the exact concrete `reference.Binding`, not a reconstructed
   interface/DTO/decoded artifact;
2. its `session-binding-v1:` identity equals the envelope identity;
3. trading date, `S/E`, prior-session date/close, schedule schema/version/hash,
   policies, and source parameters are present, UTC-normalized, internally
   consistent, and within Component 1 bounds;
4. the universe is nonempty, exact-symbol sorted/unique, at most 100,000
   symbols and 64 bytes per symbol, and its closed accounting reconciles;
5. exactly one same-order prior-close fact exists per symbol; only `valid`
   carries a finite positive close, and missing/invalid carry only fixed reasons;
6. prior-close accounting and the Component 1 portion of `PG-OBS-01` reconcile;
7. recomputed universe, prior-close, and binding identities match the stored
   identities;
8. complete copied candidate construction succeeds; and
9. the engine remains open, unbound, and `initializing`.

A digest match is necessary but not sufficient. Candidate construction mutates
nothing. The single commit installs binding plus all symbol records and applies
the live admission-time lifecycle route in the same transition:

- admission time `<S`: `LIFE-T01` to `awaiting_session`;
- admission time in `[S,E)`: `LIFE-T02` to `awaiting_aggregate_ack`;
- admission time `>=E` or controlled stop: `LIFE-T05` to `ended`.

A replay binding stays `initializing` until Component 4 supplies validated
artifact evidence for `LIFE-T03`. Failed/second/cross-binding installation
cannot replace the binding or expose partial state. S4 later owns the readable
initial and failed-install unavailable publication.

## Requirements

### `ENG-RUN-01` — one atomically bound engine

Construct one live/replay shell with no canonical session state. Accept and
fully revalidate only the concrete Component 1 binding above, atomically install
the copied candidate, and apply the required binding-time live route. Reject a
failed, partial, reconstructed, identity-only, mutable, second, or cross-binding
substitute without partial/replacement state. A changed binding requires a new
engine.

### `ENG-ADMIT-01` — bounded ownership transfer

Transfer ownership and commit admission only at successful FIFO linkage after
complete family copy/freeze, clock sample, and sequence reservation. Required
facts block with cancellation and wake on close; future optional T/Q cannot
consume the reserve and sheds explicitly. Cancellation before/after linkage,
close/seal, exhaustion, and pressure have the exact results above. No admitted
node is retractable or silently lost, and sealing drains the captured boundary.

### `ENG-ORDER-01` — one complete FIFO disposition

Reserve one nonwrapping sequence slot per admitted node, assign it at FIFO
consumption, and complete one disposition only after every transition stage
installed in the current slice completes. Every applied, rejected, fenced,
duplicate, terminal, or integrity input has exactly one sequence/disposition.
S4 extends the completion boundary through final publication decision without
changing FIFO order or prior dispositions.

## Trust, failure, and accounting

| Boundary/condition | Accept or exact result | False success prevented |
| --- | --- | --- |
| Binding provenance/install | Concrete private-field type plus complete semantic/identity/accounting validation and off-state commit; invalid gets `rejected_binding_invalid`, second gets `rejected_binding_already_installed`. | A syntactically valid digest/reconstructed value or partial candidate looking like a trusted complete binding. |
| FIFO admission | Complete deep copy, reserved slot, clock sample, successful linkage, and result commitment together. | Linked node omitted from admitted accounting, post-link mutation/retraction, or nonlinked call reported admitted. |
| Cancellation/close | Before linkage: precise nonadmission, no sample/node/sequence. After linkage: node remains admitted. Close wakes callers and drains captured nodes. | Silent loss during cancellation, capacity pressure, or shutdown. |
| Sequence exhaustion | Seal, drain reserved nodes, use reserved terminal sequence, suppress/end. | Wrap, zero/repeated sequence, or already-admitted loss. |
| Binding validation failure | Discard unreachable candidate; remain unbound/initializing; later valid install can succeed. | Partially populated canonical state or irreversible contamination. |

At a coherent admission boundary:

```text
admission_calls_started
  = admission_calls_in_progress
  + admission_results_committed

admission_results_committed
  = admitted_external
  + not_admitted_invalid
  + not_admitted_canceled
  + not_admitted_closed
  + pressure_shed_optional
  + sequence_budget_exhausted

admitted_external
  = queued_not_started
  + owner_transition_in_progress
  + completed_external_input_transitions
```

Linkage, ownership transfer, `admitted_external`, and result commitment become
visible together. A blocked call remains in progress. The proof samples after
result commitment/before caller resumption and after dequeue/before completion.
The reserved internal exhaustion transition is separate from external
admission.

Binding dependency facts also remain exact:

```text
universe_total
  = valid_prior_close
  + missing_prior_close
  + invalid_prior_close

invalid_or_missing_prior_close
  = missing_prior_close
  + invalid_prior_close
```

These are not Component 3 mark/ranking populations.

## Bounds and evidenced edge cases

- Binding: `1..100,000` sorted exact symbols, `1..64` bytes, one prior fact
  each. Test zero, over-limit, disorder, duplicate, accounting mismatch, and
  injected candidate-build failure; do not claim a 100,000-record load test.
- FIFO: finite `C/R`, optional occupancy `<=C-R`, total `<=C`, one bounded node
  per fact. Component 8 selects production values.
- Sequence: checked `uint64`, last value reserved, no wrap/reuse.
- Canonical symbol records: exactly `universe_total`; no later symbol insertion.
- No aggregate tail, timer, contributor, publication store, retry loop, or
  later component state is introduced in S1.

Evidenced cases are binding/envelope mismatch, second/cross-binding install,
partial candidate failure, concurrent producer completion differing from FIFO
linkage, caller mutation after admission, pre/post-link cancellation, close
while blocked or queued, optional reserve pressure, and near-exhaustion. These
derive from Component 1 identity/immutability, Phase 1 ownership/flow, and the
documented v2 caller-sequencing defect; S1 ports no v2 implementation.

## Primary proofs and dangerous false-success cases

| Requirement | Primary proof | Dangerous case and observable distinction | Construction/limitation |
| --- | --- | --- | --- |
| `ENG-RUN-01` | Binding-install/lifecycle atomicity scenario | `DF-01`, `DF-02`, binding `DF-12`: DTO/zero/partial/identity mismatch, injected partial-build failure, second install, and omitted time route. Valid live input installs all facts once and reaches the exact time-region state; replay remains initializing; invalid cases have one rejection and zero partial/replacement state. | Private dependency fields, defensive accessors, candidate/single commit. No `unsafe`/malicious in-package/OOM defense; readable initial/failure view waits for S4. |
| `ENG-ADMIT-01` | Bounded ownership/closure concurrency scenario | `DF-03`, `DF-04`, plus linkage `DF-15`: post-link mutation, cancel on both sides of linearization, blocked close, optional shed, exhaustion/drain. One exact result, stable copied bytes, every linked ordinal dispositioned, occupancy bounds and identities hold at both pause points. | Self-contained node, one linkage/result point, one FIFO. Small `C/R`; no production capacity or future-family immutability claim. |
| `ENG-ORDER-01` | Ordered-transition trace | `DF-05`: vary producer completion while holding FIFO linearization; mixed success/reject/fence/integrity near limit yields contiguous unique sequences and deterministic S1 state. Reversing linkage is explicitly a different trace. | One consumer/no sequence setter. Does not preordain concurrent calls before linearization, prove provider order, or include S4 publication completion. |

Race mode is secondary evidence of data-race absence, not the semantic proof.

## S1 implementation assignment boundary

**Outcome:** one atomically bound engine kernel with bounded immutable FIFO
admission and complete S1 order.

**Allowed package:** new `internal/engine` and its focused tests only; read-only
import of `internal/reference`. No Component 1 edit or adapter/later package.

**Approved v2 use:** no implementation port. Only whitelisted
`state_test.go` input-construction ideas may inform fixtures; `state.go` and
`types.go` are forbidden.

**Review artifact:** behavior/diff; the three primary proof designs/results;
construction guarantees; admitted evidence-to-success and rejection/drain
walkthrough; race result where used; inspection-only claims; limitations;
confirmation that deferred behavior is absent; and whether S2 remains valid.

**Explicitly deferred:** aggregate/timer/later variants, remaining lifecycle,
`T`, immutable publication, contributors, and Components 3–11.

**Independent review trigger:** binding provenance/atomic installation plus
sole FIFO ownership/order must receive a narrow independent review before S2.

Stop for any Component 1 interface change, need outside `internal/engine`, new
fact family, unapproved v2 source, caller-selected sequence/time/transition,
silent loss/reorder, failed accounting identity, or inability to leave the
repository buildable with all S1 proofs passing.
