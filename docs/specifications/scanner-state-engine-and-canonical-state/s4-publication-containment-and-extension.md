# ScannerStateEngine and canonical state — S4 publication, containment, and extension

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Final common/cross-family validation interaction,
immutable publication and owner-sampled `generated_at`, compiled extension
construction rule, cross-path failure containment, completed accounting and
bounded observability, S4 proof, and final Component 2 conformance walkthrough

**Controlling requirements:** `ENG-INPUT-01`, `ENG-PUBLISH-01`,
`ENG-MODULE-01`, `ENG-FAIL-01`, `ENG-OBS-01`; `ARCH-OWN-01`–`ARCH-OWN-04`,
`ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-MODEL-01`–`DTE-MODEL-03`,
`DTE-CLOCK-06`, `DTE-MERGE-05`, `DTE-COMMIT-04`, `DTE-REJECT-01`,
`DTE-REJECT-02`, lifecycle publication/suppression/end requirements

**Allocated slices:** `S4` only

**Document dependencies:** Parent; accepted S1; accepted S2; accepted S3;
[authority and reuse](authority-and-reuse.md)

**Approval state:** Inherits the parent contract approval; not independently approved

**Implementation status:** `S4` implemented and owner-accepted 2026-08-05
after all five allocated primary proofs, all accepted S1–S3 proof reruns,
package/race/repository/vet verification, correction of both independent-review
findings, and a clean targeted re-review. The separate read-only final
Component 2 review remains required; this acceptance does not authorize
Component 3 implementation.

## S4 outcome

Every admitted Component 2 input completes only after its common validation,
family disposition, canonical and lifecycle/commit work, completed accounting,
and one publication decision. Changed output becomes one new private immutable
publication with independent ID and owner-sampled `generated_at`; unchanged
output records no exposed change; publication failure swaps a preconstructed
unavailable sentinel. Readers can observe only a complete old view, complete
new view, or unavailable sentinel.

S4 also freezes the rule for future engine-contained contributors without
creating any contributor instance, registry, callback, placeholder state, or
generic bus. It completes Component 2 failure containment and accounting but
implements none of Components 3–11.

## Semantic publication, extension, and read boundaries

| Item | Normative meaning | Bound/owner |
| --- | --- | --- |
| Internal publication | Private schema, independent checked publication ID, optional binding/date, run mode, lifecycle/reason, last completed engine sequence, optional `T`, owner-sampled `generated_at`, supporting fence, Component 2 accounting, and bounded aggregate/integrity status. Ranking/readiness/rows/TQ/checkpoint/public compatibility fields do not exist. | Built after completed transition into private containers; unexported fields expose values/defensive copies; one atomic cell retains latest. Not Component 10 schema. |
| Initial/sentinel views | Construction installs immutable `initializing/unavailable` with no binding, `T`, or market claim. A fixed preconstructed unavailable sentinel for each constructible integrity cause preserves that exact cause and its deterministic suppression disposition while exposing no current claim or mutable container. | Always readable; five fixed cause variants, one atomic cell, zero publication history. |
| Publication clock | After transition work completes, take one serialized nonregressing injected-clock read for `generated_at` immediately before private construction. | Metadata only: cannot determine membership, order, precedence, target, `T`, or invalidate an older already-linked FIFO admission sample. |
| Contributor rule | A later approved module is an explicit statically named source-order call. It receives a copied/read-only transition projection and only its named engine-owned substate; executes synchronously; returns a bounded typed result copied/applied by owner. | No callback/registry/interface list now. No retained reference, goroutine, queue, clock, watermark, publisher, cross-module mutation, or mutable returned alias. |
| Checkpoint projection seam | Component 7 may later add one explicit owner stage/method copying state committed through exactly `T0` into a purpose-specific immutable projection before async writing. | No payload, validator, install, cadence, codec, retry, or storage in Component 2. Writer outcome cannot advance `T`. |
| Read boundary | Readers atomically load only the private immutable publication or later purpose-specific projection. | No canonical map/pointer/slice alias or callback; slow I/O never blocks mutation. |

The completed transition order is:

```text
freeze/copy + clock-sample + FIFO-link
  -> consume + engine_sequence
  -> validate full closed input/context
  -> classify exactly one disposition
  -> mutate permitted canonical state
  -> run named installed contributors in fixed order
  -> complete accounting + lifecycle + central commit decision
  -> sample generated_at; build/validate immutable publication if changed
  -> atomically replace one cell, record no change, or fail closed to sentinel
```

No external caller can invoke validation, apply, evaluation, or publication as
separate authoritative steps. S4 extends S1's completion boundary through the
publication decision without changing earlier FIFO order or dispositions.

## `ENG-INPUT-01` — closed common and cross-family validation

Every compiled variant passes common schema/kind, binding, run/source,
lifecycle, date/session, symbol/container, and position/context checks before
affected mutation. S2 owns aggregate-specific epoch/generation/token/interval/
precedence branches; S4 reruns aggregate paths only to prove the distinct final
accounting/publication interaction.

| Family | Success evidence | Containment and implementation boundary |
| --- | --- | --- |
| Binding | Exact concrete Component 1 type plus S1 provenance/semantic/identity/accounting checks in unbound initializing state. | Pre-link invalid; post-link fixed invalid/already-installed. S1 owns install/routes; S4 owns readable initial/failure view. |
| Aggregate | Supported schema/binding/symbol/source/run/lifecycle/one-second interval/structure and applicable context/position; compacted-present overlap excluded. | Reject/fence/integrity before partial mutation. S2 owns exact merge; Components 4–6 own normalization, epochs/tokens/work. |
| Timer | System schema, installed binding, positive system position, legal run intent, admission-sampled time. | Reject caller time/context; regression is global. S3 owns timer; Components 4/8 own replay/live scheduling and production `D`. |
| Engine control | Closed fixed kind/reason, applicable binding, legal lifecycle, run-internal authorization. | Unknown/illegal/wrong binding rejects; stop seals/drains; acquisition failure cannot impersonate binding. Only Component 2 stop/close/exhaustion/session/integrity controls exist. |
| Connection/aggregate ack | Future Component 5 exact binding, positive epoch, causal position, command/channel/token and status. | Absent now. Later stale context fences; aggregate/control-loss ambiguity global; adapter cannot declare currentness. |
| Hydration/recovery | Future Component 6 exact binding/generation/token/symbol/interval/purpose/position and exactly one terminal outcome per registered item. | Absent now. Later stale fences; contradiction contains/rejects; empty terminal is not fabricated no-print. Finite planner/ledger bounds required first. |
| Checkpoint load/projection/write | Future Component 7 exact binding/schema/coherent `T0`, validated complete contents, or matching request/result identity. | Absent now. Reject mixed/incompatible before visibility; persistence failure cannot alter `T`/canonical/lifecycle. |
| Trade/quote/TQ ack/pressure | Future Components 5/9 exact binding/symbol/epoch/selected acknowledged causal coverage, identity/time/conditions/position, bounded command token/intent. | Absent now. Stale/local invalid/optional shed cannot affect aggregate `T`, rank, readiness; quote structural acceptance differs from Spread validity. |
| Replay end/artifact control | Future Component 4 validated binding/artifact schema/provenance/ordinals/logical times/coverage. Aggregate/timer reuse existing variants. | Absent artifact/end scheduler. Corruption/wrong ordinal/regressing simulated clock fails replay; never creates live epoch/latency evidence. |
| API/UI | No market-state input. | Readers only; writable callback/reference is unavailable by construction. |

Unsupported family/kind cannot be encoded into opaque `unknown` success.
Before linkage it is `not_admitted_invalid`; a representable admitted version
mismatch gets one completed `rejected_unsupported_schema` transition.

## `ENG-PUBLISH-01` — atomic immutable publication

Every observation follows completed disposition, canonical/withdrawal work,
contributor reevaluation, accounting, lifecycle, and commit. When output
changes, serialize/check the publication clock read, assign a nonwrapping ID
independent of `T`, build and validate a fully private view off-cell, then
atomically replace the one cell. Corrections, lifecycle/control, or future
availability changes at unchanged `T` get a new ID. A duplicate/no-output
change records `no_exposed_change` and does not create an identity merely for
internal work.

Publisher, reader, future contributor, checkpoint writer, and API code receive
no writable alias. Mutation of any returned container cannot affect canonical,
current, or later views. Builder/semantic validation failure never swaps the
candidate; it atomically installs the `publication_integrity`/
`restart_required` unavailable sentinel, closes admission,
and completes `integrity_failure_publication`. Readers never continue seeing a
stale current success as if failure had not occurred. Atomic pointer replacement
has no failure return; process OOM and `unsafe` corruption are not recoverable
claims.

Before Component 3, publication may report aggregate processing/lifecycle but
must label ranking, population evaluation, and readiness unavailable. It may
not manufacture empty ranking/current product output because base state is
coherent.

## `ENG-MODULE-01` — compiled extension construction rule

Component 2 exposes no runtime registration, callback, reducer interface,
generic payload, production feature instance, writable engine/substate, or
independently mutable result. A later focused spec may add only an explicit
source-order call and named bounded engine-owned substate under the projection/
copy rules above. Qualification/ranking, readiness, recovery, checkpoint, T/Q,
replay, and API components extend the same owner and cannot bypass it.

Construction/source inspection is material evidence: Go cannot type-system
prove absence of malicious globals, `unsafe`, goroutine creation, or reference
retention. Each actual later contributor gets its own algorithm/integration
proof; Component 2 does not preapprove it.

## `ENG-FAIL-01` — smallest valid containment and terminal drain

Fixed typed reasons, never arbitrary retained strings, determine containment:

Every global failure uses the S3 closed cause-to-disposition mapping. The
triggering completion, latest lifecycle transition, and immutable unavailable
publication carry the same disposition. The unavailable view carries its
actual clock/canonical/sequence/accounting/publication cause; it never replaces
one cause with a shared `publication_integrity` label.

| Condition | Mutation/visibility | Lifecycle/currentness consequence |
| --- | --- | --- |
| Binding invalid/second | S1 exact rejection; invalid candidate unreachable; S4 publishes bounded unavailable progress for failed first install. | Remain observable unbound initializing; second leaves original run unchanged. |
| Pre-link invalid/canceled/closed/shed/exhausted | Exact admission result; no node/sample/sequence. | No transition; close wakes callers. |
| Post-link cancel/mutation | Node remains immutable/admitted and completes once. | Normal disposition. |
| Wrong/stale context | One fence, no affected canonical/work/lifecycle change; late terminal cannot complete twice. | Local unless global aggregate support becomes unknowable. |
| Unsupported/malformed/illegal/lower position | One fixed pre-mutation rejection. | Local unless raw ingress ambiguity means aggregate/control loss cannot be classified. |
| Aggregate duplicate/conflict/future/late | Exact S2 outcome; greater equal-live support may advance; historical ambiguity withdraws/recomputes; registration over compacted presence rejects. | Local/interval unknown where containable; a required unresolved gap later blocks currentness/`T`. |
| Empty/failed future work | Future exact token gets one terminal `completed_empty`/failure/cancel/fence; no mark/no-print fabrication and no double terminal. | Future terminal evidence must exit active wait; no production work family exists now. |
| Clock regression | Fixed integrity result before fact mutation; retain last supported `T`; cause-accurate unavailable publication; close admission for Component 2's terminal disposition. | Explicit suppression; ordinary timer cannot restore. |
| Canonical/ingress/order/sequence/accounting contradiction | Reject contradictory candidate; record exact cause/disposition; close admission when the mapped disposition requires termination; drain/fence admitted work; cause-accurate unavailable publication. | Global suppression/termination. A later approved recoverable ingress cause may use the Phase 1 `same_binding_recovery_allowed` path without sealing. |
| Publication build/validation failure | Candidate never swaps; `publication_integrity`/`restart_required` sentinel swaps; transition is integrity failure. | Global unavailable suppression/termination; no stale current success. |
| Local checkpoint/TQ/API/UI/symbol-field failure | Future owning bounded status only. | Cannot freeze aggregate progress absent Phase 1 shared aggregate/control ambiguity. |
| Stop/session end | Seal/capture/drain; terminally cancel registered work once; publish final supported nonlive state. | `ended`; no later mutation and no wait for external I/O. |

Panics are not dispositions. Detectable invariant failures use the integrity
path. Provider retry/deadline/shutdown grace remains Component 8 policy.

## `ENG-OBS-01` — completed accounting and bounded observability

S1 owns admission/linkage identities and S2 owns aggregate partitions. S4
completes an external transition only after the publication decision:

```text
completed_external_input_transitions
  = applied_market
  + applied_nonmarket
  + exact_duplicate
  + rejected
  + fenced
  + terminal_work_fact
  + integrity_failure

completed_transition_publication_decisions
  = no_exposed_change
  + publication_replaced
  + publication_integrity_failure
```

Every completed external or internal transition has one publication decision.
No completion increments at dequeue, validation, canonical mutation, lifecycle,
or contributor stage. Publication failure is not also a successful replacement.
An accounting mismatch is global integrity and counters are never adjusted to
make a view balance.

Future Component 6 registered work must satisfy:

```text
planned_work
  = open_work
  + completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Terminal generation requires `open_work=0`; registration increments planned
and open together, and one accepted terminal moves one item once. This state is
absent until finite Component 6 bounds are approved.

Retained observability is only fixed enum-indexed scalar counts/maxima, current
queue configuration/occupancy, current epoch/generation/fence once installed,
latest transition, current lifecycle/`T`, correction-tail occupancy/maxima,
future active-work counts, and one atomic cell. There is no transition or
publication history, raw payload, symbol/URL/credential label, arbitrary error,
or unbounded reason. Exact metrics/alerts/histograms/freshness belong to
Component 8.

## S4 bounds and dangerous false-success cases

- Publications: exactly one atomic cell containing one latest immutable view or
  one of five fixed preconstructed cause/disposition sentinels; zero history. Checked nonwrapping publication ID shares the
  sequence lifetime discipline.
- Diagnostics: one latest transition and finite enum-indexed `uint64`
  scalars/maxima; no per-event/per-symbol retained diagnostics.
- Contributors/later ledgers/retries: absent. Future constructors must declare
  and prove exact finite retention before compiling their variant/call.
- S4 reruns S1/S2 bounds to inspect alternate success/failure/close/conflict
  reachability; no path may retain uncharged state.

Primary dangerous cases are:

- `DF-09`: correction changes core without contributor reevaluation/new
  publication, or backlog causes creation-time sampling to reuse/invalidate
  admission time;
- `DF-13`: contributor retains writable state or publishes independently;
- `DF-14`: missing prebinding view, publication alias, or builder failure leaves
  stale current success;
- `DF-15`: linked/in-progress/completed accounting falls between identities;
- `DF-16`: ordinary fixture passes while future/finalized/failure/conflict/
  diagnostic paths retain uncharged state;
- `DF-17`: generic placeholder/test hook prematurely implements later behavior;
  and cross-path portions of `DF-03/04/06`–`DF-12` where completed publication,
  accounting, or failure containment is distinct from the earlier proof.

## Primary proofs

| Requirement | One primary proof | Observable distinction and limitation |
| --- | --- | --- |
| `ENG-INPUT-01` | Closed common/cross-family validation scenario covering `DF-06/12/17` across all compiled variants through S4 and aggregate participation. | One rejection/fence at sequence; coherent canonical/lifecycle/contributor/`T`/accounting/publication; unsupported family cannot succeed. Aggregate-specific rules are not duplicated; future payload/producers unproved. |
| `ENG-PUBLISH-01` | Atomic immutable-publication/clock scenario covering `DF-09/14`: initial/failed views, reads at every private stage, same-`T`, mutation, backlog clock ordering, ID boundary, builder failure. | Readers see complete initial/old/new/sentinel only; ID and `generated_at` are exact; no alias; later publication read does not invalidate old queued sample. Race mode only proves race absence; OOM/unsafe/public serialization unproved. |
| `ENG-MODULE-01` | Compiled extension construction/source proof covering `DF-13/17`. | No registration/engine pointer/setter/publisher/dummy contributor; unsupported later state/fact absent. Principal evidence is API/dependency/global/goroutine/reference inspection; later algorithms unproved. |
| `ENG-FAIL-01` | Cross-path fault-containment/terminal-drain scenario covering local, fence, every constructible global cause/disposition, canonical captured-FIFO drain, closure, sequence exhaustion, and publication failure without stale success. | Local preserves unrelated state; stale fences; global becomes cause-accurately unavailable; terminating dispositions close admission and every already-admitted node completes; later families have no success path. Does not prove later retries/work/recovery/checkpoint/TQ/API/shutdown policies. |
| `ENG-OBS-01` | Cross-stage accounting/cardinality scenario covering `DF-15/16` after each success/reject/fence/conflict/close/publication failure and long trace. | Identities reconcile at linkage/dequeue pause points and only after publication decision; all retained sets, including both lazy session bitmaps, obey the charged S1/S2/S4 bounds; mismatch takes integrity path. Mathematical maxima prove finiteness, not production capacity. |

Construction inspection also confirms no product wall-time read, second FIFO/
publisher, map-controlled dispatch/merge order, mutable payload/publication
alias, raw/high-cardinality retention, later Component 3–11 state, or unapproved
predecessor dependency.

## S4 implementation assignment boundary

**Outcome:** completed transitions publish one immutable private view; common
validation, cross-path containment/accounting, and no-registry extension rule
complete Component 2.

**Allowed package:** `internal/engine` publication/store/diagnostic code and
focused tests only. No public JSON/API, actual contributor, checkpoint schema,
operations backend, or later package.

**Approved v2 use:** adapt scoped validation cases,
`store.go`/`store_test.go` atomic and mutation-isolation behavior, and
`metrics.go` scalar/enumeration technique. Do not port `Snapshot`,
`CloneSnapshot`, v2 reason sets, `state.go`, or `types.go`.

**Review artifact:** five primary proof results; construction/success/failure
walkthrough below; race result where used; inspection-only claims; all
limitations; proof that later behavior is absent; full prior-proof rerun; and
final component-review request.

**Explicitly deferred:** every Component 3–11 behavior, successful `T`
advancement, production hydration ledger, and operations/capacity policy.

**Independent review trigger:** immutable publication/clock identity, failure
sentinel, cross-stage accounting, and extension/reader ownership require narrow
independent review before final component review.

## Final Component 2 conformance walkthrough

The S4 review must record this concrete path, not merely test command names:

1. Concrete Component 1 binding is copied into a FIFO node, fully revalidated,
   built off-state, committed once, and routed by admission time; replay waits
   for Component 4 artifact evidence.
2. Every normalized fact is frozen and clock-stamped at linkage; FIFO
   consumption assigns engine sequence and validates its whole context.
3. Exactly one disposition results. Aggregate mutation obeys identity,
   precedence, inclusive horizon, equal-live authority, historical fill-only,
   withdrawal/recompute, and finalized presence.
4. Accounting, lifecycle, and commit guard finish; Component 2 base gate still
   cannot advance `T`.
5. Owner samples `generated_at`, builds a private view with a new independent
   ID, and swaps one cell. Later clock sampling does not invalidate an older
   queued admission sample; readers cannot alias state.

Failure walkthrough must separately show pre-mutation rejection, stale-context
fencing, symbol/interval-local containment, global integrity drain/unavailable
sentinel/suppression, pre/post-link cancellation and closure, and publication
builder failure. It must identify claims supported only by source inspection
and state that `unsafe`, memory corruption, malicious in-package code, process
OOM, provider conformance, production capacity/currentness, all later component
behavior, and public schema compatibility remain unproved.

Stop for any writable alias/partial view, unbounded copied graph, second
publisher/clock/mutation path, accounting mismatch, new later state/family,
unapproved v2 source, or inability to cover every participating implementation
path in the allocated proof.
