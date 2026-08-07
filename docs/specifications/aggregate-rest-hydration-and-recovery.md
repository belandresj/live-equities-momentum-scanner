# Aggregate REST hydration and recovery

**Status:** Finally accepted 2026-08-07 after delegated acceptance of
`C6-S1`–`C6-S4`, complete final-tier verification, and clean mandatory
independent final review with focused re-review; no Component 6 acceptance item
remains open

**Owner boundary approval:** Pre-approved 2026-08-06 by the owner in the
initiating Component 6 task, subject to the exact Sections 1–7 boundary,
document map, and reconnaissance limits below

**Owner stable-interface exception:** Approved 2026-08-06 for version 2
reconnaissance and Sections 8–19 before Component 5 final review. The exception
is limited to Component 4's accepted REST aggregate value/normalizer seam,
Component 5's approved aggregate-acknowledgement, epoch/loss, and live-ingress
fact meanings, and Components 1–3's finally accepted interfaces. It does not
authorize dependence on unfinished Component 5 implementation details or any
Component 6 implementation.

**Owner contract/reuse/test/slice-plan approval:** Approved 2026-08-06. The
approval covers completed Sections 8–19, the behavior-only V2 whitelist, all
eleven primary proofs, `C6-S1`–`C6-S4`, required narrow reviews, and the
Component 6-owned C5 capture-command/marker extension. It permits this
completed-contract approval before Component 5 final review but does not by
itself authorize early Component 6 implementation.

**Owner early-implementation exception:** Approved 2026-08-06 for `C6-S1`
only. The owner explicitly overrides the roadmap's component-order and
one-active-implementation-slice rules solely to allow one active Component 5
slice and `C6-S1` to proceed concurrently. `C6-S1` has no Component 5
dependency: it may use only the accepted Components 1 and 4 interfaces, this
contract's approved behavior-only V2 evidence, and its allocated REST worker
files/proofs. It may not inspect, modify, adapt to, or make acceptance depend on
unfinished C5 implementation details. A file/semantic collision with C5, a
required C5 change, or verification that cannot be attributed to one coherent
source snapshot stops `C6-S1` without acceptance. This exception does not
authorize `C6-S2`, `C6-S3`, or `C6-S4` before Component 5 final review.

**Advancement mode:** `delegated` for `C6-S1`–`C6-S4` and final component
review once each slice is authorized. The owner expressly pre-authorizes the
implementer to write the compact `C6-S1` acceptance record and change its sole
parent-ledger state to `accepted` without another owner message when every S1
proof, required verification, narrow independent review, walkthrough, and
drift check is clean. Component 5 has since passed final review, so an accepted
S1 activates `C6-S2` through the normal approved delegated sequence without a
new owner message.

**Controlling Phase 1 requirements:** `PG-RANK-02`, `PG-RANK-05`,
`PG-FEATURE-05`, `PG-AVAIL-01`, `PG-AVAIL-02`, `PG-OPS-01`, `PG-OPS-02`,
`PG-OBS-01`, `PG-OBS-02`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, `ARCH-FLOW-03`,
`ARCH-FLOW-04`, `DTE-MODEL-01`, `DTE-MODEL-02`, `DTE-MODEL-03`,
`DTE-SESSION-01`, `DTE-SESSION-02`, `DTE-SESSION-03`, `DTE-SESSION-04`,
`DTE-CLOCK-03`, `DTE-CLOCK-05`, `DTE-WINDOW-01`, `DTE-WINDOW-02`,
`DTE-WINDOW-04`, `DTE-EVENT-01`, `DTE-EVENT-03`, `DTE-AGG-01`,
`DTE-AGG-02`, `DTE-AGG-03`, `DTE-AGG-04`, `DTE-HYDRATE-01`,
`DTE-HYDRATE-02`, `DTE-MERGE-01`, `DTE-MERGE-02`, `DTE-MERGE-03`,
`DTE-MERGE-04`, `DTE-MERGE-05`, `DTE-RECOVERY-01`,
`DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-RECOVERY-04`,
`DTE-RECOVERY-05`, `DTE-COMMIT-02`, `DTE-COMMIT-03`,
`DTE-COMMIT-04`, `DTE-CHECKPOINT-02`, `DTE-CHECKPOINT-03`,
`DTE-REJECT-01`, `DTE-REJECT-02`, `LIFE-MODEL-01`, `LIFE-MODEL-02`,
`LIFE-MODEL-03`, `LIFE-MODEL-04`, `LIFE-INIT-03`, `LIFE-INIT-04`,
`LIFE-HYDRATE-01`, `LIFE-HYDRATE-02`, `LIFE-HYDRATE-03`,
`LIFE-HYDRATE-04`, `LIFE-HYDRATE-05`, `LIFE-HYDRATE-06`,
`LIFE-HYDRATE-07`, `LIFE-LIVE-01`, `LIFE-LIVE-02`, `LIFE-LIVE-03`,
`LIFE-RECOVER-01`, `LIFE-RECOVER-02`, `LIFE-RECOVER-03`,
`LIFE-RECOVER-04`, `LIFE-RECOVER-05`, `LIFE-RECOVER-06`,
`LIFE-END-01`, `LIFE-END-02`, `LIFE-END-03`, `LIFE-PUBLISH-02`,
`LIFE-PUBLISH-03`, `LIFE-T06`, `LIFE-T08`, `LIFE-T11`, `LIFE-T12`,
`LIFE-T13`, `LIFE-T14`, `LIFE-T16`, `LIFE-T18`, `LIFE-T19`,
`LIFE-T20`, `LIFE-T21`, `LIFE-T22`, `LIFE-T26`, and `LIFE-T29`

**Approved dependencies:** the finally approved Component 1 immutable
[`reference.Binding`](reference-data-and-session-binding.md#9-detailed-semantic-inputs-outputs-and-owned-state),
the finally approved Component 2
[`ScannerStateEngine` contract](scanner-state-engine-and-canonical-state.md),
the finally accepted Component 3
[`aggregate evaluator contract`](aggregate-features-qualification-ranking-and-accounting.md),
Component 4's finally accepted
[`Massive REST row normalizer and provider-independent aggregate value seam`](aggregate-replay/rest-normalization-and-artifact.md),
and only the approved Component 5 aggregate-acknowledgement, connection-epoch,
connection-loss, and live-ingress fact meanings in the
[`Massive live adapter contract`](massive-live-adapter.md). Component 6 may
specify the engine-owned production hydration/recovery extension but may not
reinterpret or duplicate these dependencies.

## Contract document map

The modular layout separates the external REST trust boundary from the
engine-owned generation/reconciliation boundary and from proof/delivery
allocation. All files listed here will form one Component 6 contract and pass
the same two approval gates.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Outcome, single ownership boundary, explicit non-scope, cross-cutting Phase 1 invariants, Sections 1–7, routing, approvals, and sole delivery ledger | All controlling Phase 1 IDs; Sections 1–7 | Every Component 6 task | Components 1–5 stable interfaces named above |
| [REST acquisition and terminal outcomes](aggregate-rest-hydration-and-recovery/rest-acquisition-and-terminal-outcomes.md) | Production request construction, pagination, response trust, reuse of the one Component 4 row mapper, bounded concurrency/retry/cancellation, and exactly one terminal result per requested symbol | Sections 8–14; `C6-REST-01`, `C6-WORKER-01`, `C6-BOUND-01`; three proofs; `C6-S1` | REST client, provider trust, normalization consumption, and worker accounting | Parent; Components 1 and 4 |
| [Engine hydration, reconciliation, and recovery](aggregate-rest-hydration-and-recovery/engine-hydration-reconciliation-and-recovery.md) | Engine-owned plans/generations/tokens, startup/catch-up/gap intervals, historical admission, live ingress fences, terminal consequence/no-print proof, lifecycle exits, and exact recovery | Sections 8–14; `C6-PLAN-01`, `C6-LEDGER-01`, `C6-MERGE-01`, `C6-FENCE-01`, `C6-START-01`, `C6-NOPRINT-01`, `C6-RECOVER-01`, `C6-INTEGRATION-01`; eight proofs; `C6-S2`–`C6-S4` | Engine extension, hydration/recovery lifecycle, reconciliation, no-print, and integration | Parent; preceding REST detail; Components 1–5 stable interfaces |
| [Proof and delivery plan](aggregate-rest-hydration-and-recovery/proof-and-delivery-plan.md) | Complete requirement/proof ledger, exact reuse whitelist, sequential slice plan, discretion, completed-contract checklist, and drift audit | Sections 15–19; all eleven Component 6 requirements/proofs; `C6-S1`–`C6-S4` | Assignment preparation, completed-contract approval, and final review | Parent and both preceding details; Components 1–5 stable interfaces |

**Layout:** Modular contract with this parent and the three routed details
above. This parent is about 3,450 words, above the template's approximate
2,500-word routing alarm, because its owner-approved Sections 1–7 are one
inseparable boundary, exact-scope, and stable-interface-exception record. All
reconnaissance, detailed requirements, trust/proof ledgers, and delivery
material are routed out rather than extending it. Changing that map or moving
the approved boundary record requires owner review.

**Routing rule:** Any requirement, evidence decision, proof, slice, or task not
unambiguously routed by this table stops for a parent-map correction. Models do
not guess among detail specs or load unrelated dependency details.

**Contract-wide coverage and acceptance:** Sections 1–7, the exact early
reconnaissance exception, and the detailed
[eleven-requirement/proof ledger, exact approved whitelist, four-slice plan,
checklist, and drift audit](aggregate-rest-hydration-and-recovery/proof-and-delivery-plan.md)
were owner-approved as one completed Component 6 contract on 2026-08-06.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `approved` | Owner-approved complete modular contract, behavior-only V2 whitelist, eleven proofs, four slices, required reviews, C5 capture-marker extension, delegated advancement, and exact early-S1 exception | 2026-08-06 | Preserve the approved contract and exception bounds |
| `C6-S1` | `accepted` | `C6-REST-01`, `C6-WORKER-01`, and `C6-BOUND-01`; all three primary proofs, unchanged C4 mapper/downloader/compiler/artifact proofs, repository build/test/vet, focused worker race, formatting/source/scope/secret inspections, clean drift audit, and required `gpt-5.6-sol` medium external-trust focused re-review all clean | 2026-08-07 | Complete |
| `C6-S2` | `accepted` | `C6-PLAN-01`, `C6-LEDGER-01`, and `C6-MERGE-01`; all three primary proofs, affected Components 2–4 and accepted S1 regressions, repository build/test/vet, engine/Massive race, formatting/source/scope inspections, clean drift audit, and required `gpt-5.6-sol` medium sole-owner focused re-review all clean | 2026-08-07 | Complete |
| `C6-S3` | `accepted` | `C6-FENCE-01`, `C6-START-01`, and `C6-NOPRINT-01`; all three primary proofs, affected Components 2/3/5 and accepted S1/S2 regressions, repository build/test/vet, adapter/engine race, formatting/source/scope inspections, clean drift audit, and required `gpt-5.6-sol` medium fence/lifecycle focused re-review all clean | 2026-08-07 | Complete |
| `C6-S4` | `accepted` | `C6-RECOVER-01` and `C6-INTEGRATION-01`; both primary proofs, complete eleven-proof C6 ledger, affected C2–C5 regressions, repository build/test/vet, full engine/Massive race, formatting/source/scope inspections, clean drift audit, and required `gpt-5.6-sol` medium recovery/interface focused re-review all clean | 2026-08-07 | Complete |
| Final component review | `accepted` | Complete eleven-proof ledger, repository build/test/vet, full engine/Massive race, formatting/source/dependency/scope/drift inspections, and separate `gpt-5.6-sol` medium final review plus focused cancellation re-review all clean | 2026-08-07 | Complete; specification map synchronized |

**C6-S1 acceptance record:** One private strict Massive second-aggregate
acquisition core now serves both the accepted offline downloader and the new
production hydration worker. The worker copies immutable work identity, seals
the complete provider result before emitting contiguous copied chunks, and
returns exactly one provider value, empty, failed, or canceled terminal per
accepted item. Plan-wide response-byte, normalized-row, resident-row, chunk,
interval, work-item, and `1..8` worker bounds are validated before I/O; worker
and blocked-admission cancellation join all goroutines. A still-open engine
input receives exactly one canceled terminal; only explicit closed-input
evidence retains an unadmitted terminal as bounded cleanup, while unexpected
open-input rejection is a distinct admission-integrity failure.

`P-C6-REST`, `P-C6-WORKER`, and `P-C6-BOUND` pass together with the affected
Component 4 mapper, downloader, compiler, artifact, and aggregate-replay
proofs. Repository build/tests/vet, the focused hydration-worker race run,
formatting, diff, dependency, secret-containment, sole-decoder/mapper/client,
and ownership/scope inspections are clean. The required independent
`gpt-5.6-sol` medium external-trust review found blocked terminal admission and
missing second-page-failure proof coverage; the implementation added the
cancellation-aware terminal path and both regressions, and the same reviewer
reported the focused re-review clean with no remaining blocking or nonblocking
finding. Success is limited to strict offline fake-provider evidence; provider
availability/current schema, engine applicability/fencing, canonical coverage,
deployed capacity, and cross-generation policy remain unproved exactly as
deferred.

Construction prevents a second mapper/page decoder, mutable caller aliases,
worker engine-state authority, partial-page success, detached workers, and
credential-bearing facts. Success is strict complete acquisition, copied
chunks, then a reconciled value/empty terminal. Any later page/row/envelope or
budget failure discards the complete buffer and emits no chunks; cancellation
stops further chunks and yields an admitted canceled terminal, or explicit
unadmitted cleanup only after engine input closes. There is no contract,
dependency, interface, document-map,
whitelist, or slice-boundary deviation, no failed assumption or
success-invalidating inspection-only claim, and every implementation drift
question remains `no`. With Component 5 finally accepted, the approved S2
dependencies and assignment remain valid; the clean delegated gate accepts S1
and activates S2 without another owner decision.

**C6-S2 acceptance record:** The sole `ScannerStateEngine` now derives exact
fresh, future-checkpoint, and gap plan intervals from its installed binding,
acknowledgement, and retained engine state; allocates one positive generation
and immutable sorted request token for every valid-prior-close symbol; and
owns one finite active ledger. Massive S1 facts convert through a narrow
provider-independent seam into defensively copied engine chunk and terminal
inputs. The one FIFO validates token/result/chunk/row sequence, routes every
historical row through Component 2's existing fill-only canonical merge and
Component 3's ordinary contributor/evaluator, keeps provider terminal success
distinct from canonical coverage, and maintains exact open/five-terminal-bin
and per-row accounting after every input.

`P-C6-PLAN`, `P-C6-LEDGER`, and `P-C6-MERGE` pass with all affected existing
engine, Component 3, C4, and C6-S1 proofs. Repository build/tests/vet, focused
engine and Massive race runs, formatting, diff/dependency, ownership, bounds,
whitelist, and scope inspections are clean. The required independent
`gpt-5.6-sol` medium sole-owner review found that integrity suppression could
strand open work and that impossible terminal counts could reach provider
success. The implementation now terminally fences every remaining open item
before integrity suppression and strictly reconciles state-specific page,
attempt, byte, interval-row, normalized-row, and emitted-count evidence. The
same reviewer reports the focused re-review clean with no remaining blocking
or nonblocking finding.

Construction prevents caller-selected intervals/IDs, rank-dependent planning,
partial plan registration, caller aliases, a second engine/FIFO/canonical map/
evaluator/publication path, and Massive identity authority. Invalid plans
install nothing; stale and late facts fence without mutation; sequence or
terminal contradictions fence every open item once before suppression; live
authority wins unequal history; unequal historical evidence has no arrival-
order winner; provider completion cannot convert conflict/rejection into usable
coverage. C5 ingress fencing, lifecycle exit, final no-print consequence,
checkpoint activation/storage, gap retry/recovery policy, deployed capacity,
readiness/runtime orchestration, and provider-current/live evidence remain
unproved exactly as deferred. There is no contract, dependency, interface,
document-map, whitelist, or slice-boundary deviation, no failed assumption or
success-invalidating inspection-only claim, and every implementation drift
question remains `no`. The accepted S1/S2 facts and finally accepted C5 raw
FIFO leave the approved S3 assignment unchanged; the clean delegated gate
accepts S2 and activates S3 without another owner decision.

**C6-S3 acceptance record:** The accepted C5 raw FIFO now carries a distinct
zero-payload aggregate-ingress capture marker behind every previously admitted
raw frame without consuming a live frame sequence or aliasing the epoch
terminal marker. The only production drain serializes dequeue, Component 2
admission, and disposition completion, so a marker cannot overtake a
pre-capture aggregate or ambiguity. Capture waits cancellation-aware under raw
slot pressure; cancellation or epoch loss sends an ordered canceled fence,
deactivates the generation, and fences any later completion. The engine
accepts only the exact current binding/epoch/generation/token and a
nonregressing through-frame after every request is terminal, then installs the
bounded coverage consequence and leaves `hydrating` in the same transition.

`P-C6-FENCE`, `P-C6-START`, and `P-C6-NOPRINT` pass with all affected C2/C3/C5
and accepted C6-S1/S2 proofs. Repository build/tests/vet, adapter and engine
race runs, formatting, diff/dependency, sole-owner, no-polling, no-deferred,
and scope inspections are clean. The required independent `gpt-5.6-sol`
medium review found optional cross-FIFO ordering, missing capture-cancellation
consequence, timer-manufactured support beyond the captured boundary, and
ambiguous full-slot marker semantics. Production direct dequeue was removed;
capture failure now closes the generation synchronously; timers remain pinned
to captured `T`; marker admission and accounting now prove bounded waiting,
sequential markers, and terminal coexistence. The same reviewer reports the
focused re-review clean with no remaining blocking or nonblocking finding.

Construction prevents marker/terminal aliasing, fabricated raw sequence gaps,
marker overtaking, terminal-only finalization, stale identity completion,
queue-empty success, quiet-timer no-print extension, synthetic marks, and an
indefinite hydration stage after capture failure. Successful exact empty
coverage may establish no-print only through the reconciled fence; failed,
canceled, conflicted, prefix-incomplete, or lost evidence remains unknown, and
a later accepted live print clears no-print through the ordinary evaluator.
Checkpoint validation/storage, same-process gap recovery and retry action,
deployed currentness/readiness/orchestration, live-provider behavior, C9 T/Q,
and public API remain unproved exactly as deferred. There is no contract,
dependency, interface, document-map, whitelist, or slice-boundary deviation,
no failed assumption or success-invalidating inspection-only claim, and every
implementation drift question remains `no`. The approved S4 recovery and
integration assignment remains unchanged, so the clean delegated gate accepts
S3 and activates S4 without another owner decision.

**C6-S4 acceptance record:** On live aggregate loss, the sole engine now
retains exact committed `T_supported`, makes the current evaluation and T/Q
intent noncurrent in the same publication, preserves accepted canonical facts,
and cancels the active generation exactly once. A greater acknowledged epoch
plans the complete valid-prior population over exact `[T_supported,R)` with no
fixed backward overlap. Old epochs, generations, terminals, policies, and
fences cannot mutate the new attempt. Closed immutable retry, exhaust, cancel,
and stop policy facts are accepted only for one exact active decision point;
there is no autonomous retry count, delay, backoff, or exhaustion choice.

`P-C6-RECOVER` and `P-C6-INTEGRATION` pass with all nine preceding Component 6
proofs and affected C2–C5 regressions. The integration uses the accepted fake
C5 socket handshake/acknowledgement, strict C6 fake HTTP acquisition and C4
mapper, S1 worker conversions, Component 2 FIFO/merge, a pre-capture live frame
overlapping historical identity, the real C5 marker, and Component 3's ordinary
evaluator/publication. It proves live revision authority and exact
qualified-current population, qualification, ranking, current-market, and
publication results. Repository build/tests/vet, full engine/Massive race,
formatting, diff/dependency, ownership, whitelist, no-deferred, and source
inspections are clean.

The required independent `gpt-5.6-sol` medium recovery/interface review found
pre-validation stop sealing, engine-inferred exhaustion without a policy fact,
an integration proof that omitted evaluator/category results, and blank
explicit-exhaustion disposition fields. Stop now seals only after ordered
identity/state validation; unresolved fenced recovery publishes an explicit
policy-wait state and remains noncurrent until a policy fact; the integration
asserts overlapping live-over-history authority and the full ordinary
evaluation; and exhaustion publishes its exact disposition/reason. The same
reviewer reports the focused re-review clean with no remaining blocking or
nonblocking finding.

Construction prevents loss-boundary drift, a recovery watermark/evaluator,
old-generation promotion, repeated policy decisions, stale stop sealing,
successful-work exhaustion, inferred retry/exhaustion, unobservable inactive
recovery, double terminal accounting, and historical overwrite of live
authority. Failed local evidence stays explicit; an unresolved complete fence
waits visibly for policy; explicit exhaustion suppresses with its fixed
same-binding recovery disposition; successful terminal work plus the real
fence returns through the ordinary evaluator. Fake socket/HTTP evidence does
not prove live-provider behavior, deployed latency/capacity, checkpoint
storage/validation, Component 8 policy values/readiness/orchestration, C9 T/Q
coverage, API/UI, or cutover. There is no contract, dependency, interface,
document-map, whitelist, or slice-boundary deviation, no failed assumption or
success-invalidating inspection-only claim, and every implementation drift
question remains `no`. The clean delegated gate accepts S4 and activates the
mandatory separate final component review.

**Final Component 6 acceptance record:** All eleven allocated primary proofs
pass on one source snapshot: strict shared REST acquisition, sealed worker
terminality/bounds, engine-owned plan/ledger/merge, raw ingress fencing,
startup/no-print lifecycle, exact same-process recovery, and the offline
Components 1–6 socket-plus-HTTP path. Repository build and all ordinary tests,
vet, full engine/Massive race detection, formatting, diff, dependency,
secret-containment, whitelist, sole-owner, mapper, boundedness, no-polling,
no-deferred, and drift inspections are clean. Work accounting reconciles one
open or five terminal bins; row and raw-queue accounting reconcile; one
current policy action decides unresolved recovery; no late fact can promote or
double-terminal an old identity.

The mandatory separate `gpt-5.6-sol` medium final reviewer found one cross-
slice cancellation defect: provider-work cancellation also canceled the
terminal-admission context, allowing cleanup-only evidence while engine input
remained open. The worker now separates work and admission lifetimes, retries
one canceled terminal after an interrupted terminal admission, treats only
explicit input closure as unadmitted cleanup, and records unexpected open-input
rejection as admission integrity. The same reviewer repeated the focused proof
and race boundary and reports no remaining finding; the final delegated gate
is satisfied.

The complete success path is one immutable binding, one strict C4 mapper and
private REST client, copied sealed worker facts, one Component 2 FIFO and
generation ledger, the existing historical fill-only merge with live
authority, one causal C5 raw marker, and one Component 3 evaluator/publication.
Failures remain bounded failed/canceled/fenced/unknown or explicit suppression;
they do not fabricate marks, absence, currentness, retry policy, or recovery
success. Residual limitations are exactly contracted: offline fake providers
only; no live-provider/schema-currentness claim, deployed capacity/latency,
checkpoint implementation, Component 8 policy values/readiness/runtime
orchestration, C9 T/Q coverage, API/UI, or cutover. There is no deviation,
failed assumption, success-invalidating inspection-only claim, or drift-audit
`yes`. Component 6 is finally accepted and the coarse specification-map status
is mechanically synchronized.

## 1. Outcome and user consequence

Component 6 establishes the production historical aggregate path that lets a
fresh start, checkpoint restart, or same-process aggregate gap become a finite,
causally reconciled operation rather than a second scanner truth or an
indefinite freeze.

For every requested binding/symbol/interval, the operator receives exactly one
terminal value, empty, failed, canceled, or fenced outcome. The engine may
derive `no_print_through(T)` only from complete successful historical coverage
joined to continuous accepted live coverage through an explicit ingress fence.
When all work is terminal, startup or recovery leaves its active state and the
ordinary evaluator resumes even if results are sparse. Failures remain honest
unknown or gap evidence; they never become a fabricated mark or false empty.

## 2. Scope and explicit non-scope

**In scope**

- Plan fresh `[S,R)`, later checkpoint `[T0,R)`, and exact same-process gap
  hydration from engine-owned binding, epoch, handoff, and supported-boundary
  evidence.
- Own positive hydration generations, immutable request tokens, finite
  per-generation work ledgers, cancellation/supersession, and exactly one
  terminal disposition per planned symbol request.
- Execute bounded Massive aggregate REST requests outside the engine, including
  exact request identity, pagination, retry/deadline/body/concurrency limits,
  response validation, and successful-empty distinction.
- Reuse Component 4's one accepted Massive REST row mapper and aggregate value;
  add no second provider-field mapping.
- Return immutable historical rows and terminal facts with exact binding,
  generation, request, symbol, interval, ordinal, pagination, count, and
  validation evidence for ordered engine admission.
- Reconcile historical fill-only values with current live authority, capture
  and consume an explicit live ingress fence, derive interval/no-print
  consequences, reconcile exact work/symbol accounting, and leave `hydrating`
  or the current recovery attempt on every terminal path.

**Not in scope**

- Massive WebSocket framing, authentication, subscription commands,
  acknowledgement parsing, connection epochs, raw live queues, or reconnect
  transport. Component 5 owns those facts and mechanics.
- A second REST aggregate mapper, replay artifact/downloader/compiler/source,
  simulated clock, or replay lifecycle. Component 4 owns them.
- Canonical aggregate identity, value meaning, live/historical precedence,
  correction horizon, feature formulas, qualification, ordering, or another
  evaluator. Components 2 and 3 own them.
- Checkpoint schema, persistence, validation, selection, installation, cadence,
  or restart target. Component 7 will own them; Component 6 only accepts a
  future valid installed `T0` and plans the already-settled `[T0,R)` interval.
- Production evaluation delay, freshness/readiness/capacity thresholds,
  cross-generation retry policy, reconnect exhaustion, or shutdown budgets.
  Component 8 owns those policies; Component 6 owns hard safety bounds and
  exposes finite facts needed by that policy.
- T/Q coverage/features/pressure, public API/UI schemas, databases, raw-event
  journals, generic event infrastructure, plugins, services, credentials,
  live provider requests, version 1 predecessor inspection, or broad version 2
  orchestration harvest.

## 3. Ownership and dependencies

The single Component 6 ownership boundary begins when the engine, after a
valid current aggregate acknowledgement or explicit same-binding recovery
decision, constructs an immutable historical-work plan from its installed
binding and supported boundaries. It ends when every planned request has one
ordered engine-consumed terminal disposition and the engine has reconciled the
generation's historical rows and live ingress fence into canonical coverage,
accounting, no-print/unknown consequences, and the required lifecycle exit.

The engine exclusively owns the active generation, request-token registry,
terminal-work ledger, accepted historical consequences, live fence,
hydration/recovery step, lifecycle, canonical state, `T`, accounting,
evaluation, and publication. REST workers own only bounded request-local I/O,
pagination, retry-attempt, decode, normalized-row, and terminal-result state.
They receive immutable plans and return immutable facts through Component 2's
one bounded FIFO; they receive no writable engine reference and do not decide
coverage, no-print, currentness, readiness, lifecycle, or retry of a whole
generation.

The stable-interface exception allows this contract to rely on Component 5's
approved fact meanings, not its unfinished implementation shape. If C5-S2/S3
or final review changes the approved acknowledgement, epoch/loss, or live
ingress interface semantically, detailed Component 6 drafting stops for an
owner decision. Routine private implementation differences do not block the
contract.

## 4. Settled Phase 1 semantic boundary

| Boundary item | Settled meaning version 2 cannot change | Controlling Phase 1 IDs |
| --- | --- | --- |
| One canonical path and owner | REST workers return normalized rows and terminal facts; only the engine validates binding/generation/token/interval, installs historical values, owns coverage/lifecycle/`T`, evaluates, and publishes. | `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-04`, `ARCH-FLOW-04`, `DTE-MODEL-01`–`DTE-MODEL-03`, `LIFE-MODEL-01`, `LIFE-MODEL-02` |
| Exact historical identity | A request token binds one positive generation, binding, purpose, symbol, and exact half-open interval. Historical source position also carries request identity and page/result ordinal; goroutine completion and map order have no authority. | `DTE-WINDOW-01`, `DTE-EVENT-01`, `DTE-EVENT-03`, `DTE-RECOVERY-01`, `ARCH-FLOW-03` |
| One shared REST value mapping | Every provider row uses Component 4's accepted unadjusted second-aggregate mapper and its `rest_floor_volume_over_transactions` provenance. Component 6 wraps that value in production historical evidence but does not decode the eight provider fields again. | `DTE-AGG-01`–`DTE-AGG-04`, `DTE-MODEL-03`; accepted `C4-NORM-01` and `C4-COMP-NORM-01` seam obligation |
| Live handoff and startup intervals | A current-epoch aggregate acknowledgement fixes `R = clamp(ceil_second(ack_received_at),S,E)`. Fresh hydration covers `[S,R)`; future checkpoint catch-up covers `[T0,R)`. Live facts after the acknowledgement remain active while REST runs. | `DTE-RECOVERY-02`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-01`–`LIFE-HYDRATE-03`, `LIFE-T06`, `LIFE-T08`, `LIFE-T11` |
| Historical/live reconciliation | Historical rows fill only missing identities under a current token and never overwrite accepted live authority. Conflicts remain bounded evidence and localize dependent uncertainty where possible. Every classified live item through the captured fence is consumed or explicitly rejected before terminal consequences support publication. | `DTE-MERGE-01`–`DTE-MERGE-05`, `DTE-RECOVERY-03`, `DTE-RECOVERY-05`, `LIFE-HYDRATE-03`, `LIFE-HYDRATE-05` |
| Empty, no-print, and failure | Complete provider success with zero normalized rows is terminal `completed_empty`, not pending. It proves only its exact interval. `no_print_through(T)` additionally requires continuous accepted live coverage through `T`; failed/canceled/fenced work remains unknown and no synthetic aggregate is created. | `PG-RANK-02`, `PG-OPS-02`, `PG-OBS-01`, `PG-OBS-02`, `DTE-WINDOW-04`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `DTE-RECOVERY-04` |
| Exact work accounting and terminality | Per generation, `planned = completed_value + completed_empty + failed + canceled + fenced`. Supersession/cancellation completes open work once; later delivery is fenced evidence, not another terminal outcome. | `PG-OBS-02`, `DTE-HYDRATE-01`, `LIFE-HYDRATE-05`–`LIFE-HYDRATE-07`, `LIFE-RECOVER-04`, `LIFE-RECOVER-06` |
| Startup exit and ordinary evaluation | Terminal startup always leaves `hydrating` in the same ordered transition. Sparse/empty/local-failure outcomes do not create `deferred` or freeze ordinary timers. The same evaluator consumes resulting canonical state. | `PG-OPS-02`, `LIFE-HYDRATE-05`, `LIFE-HYDRATE-06`, `LIFE-LIVE-01`–`LIFE-LIVE-03`, `LIFE-PUBLISH-03`, `LIFE-T12`–`LIFE-T14` |
| Same-process exact gap recovery | Aggregate coverage loss closes currentness/T/Q, preserves supported canonical state, establishes a new acknowledged epoch, hydrates every identity needed to join the supported boundary to the new live tail, reconciles a fence, and returns to ordinary evaluation or explicit suppression/termination. | `LIFE-RECOVER-01`–`LIFE-RECOVER-06`, `DTE-RECOVERY-05`, `LIFE-T16`, `LIFE-T18`–`LIFE-T22`, `LIFE-T26` |
| Bounded work and containment | All plans, queues, pages, attempts, bodies, rows, workers, retained ledgers, reasons, and waits are finite. Admitted rows/results are processed, explicitly rejected/fenced, or enter the global integrity path; blocking REST never runs on the engine path. | `ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-REJECT-01`, `LIFE-END-01`–`LIFE-END-03`, `LIFE-T29` |

Component 6 introduces no new product rule, competing mutable owner, second
watermark/evaluator, T/Q dependency, changed interval meaning, duplicate REST
mapping, replay path, checkpoint authority, readiness policy, or speculative
provider edge-case mechanism.

## 5. Unresolved questions and evidence needs

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| Which production REST client mechanics can be shared with Component 4 while preserving request identity, same-origin pagination, bounded retry/body/time, successful-empty semantics, and credential containment? | Phase 1 fixes trust consequences; Component 4 proves an offline compiler consumer but not a production hydration consumer or its ownership. | Accepted Component 4 implementation/contracts plus the scoped V2 REST client and focused fake-HTTP tests. | Exact reuse/import decision, hard request bounds, terminal result shape, and sole-mapper proof. |
| What immutable plan/token/result shapes and bounded ledger state support one terminal outcome per symbol while preventing stale generation, wrong interval, duplicate terminal, worker-order, or caller-alias false success? | Phase 1 fixes identities/accounting but delegates concrete types, registration, and memory bounds. | Scoped V2 planner/generation/token code and focused unit/concurrency tests; Component 2 ownership invariants. | Engine extension, construction guarantees, registration limits, and accounting proof. |
| What exact same-process gap interval and evidenced overlap join the last supported live boundary to the new acknowledged live tail without omission or double authority? | `LIFE-RECOVER-03` delegates exact gap start and overlap while requiring complete coverage. | Scoped V2 gap planner/recovery tests plus Phase 1 half-open/merge invariants. | Exact recovery plan and boundary proof; any unresolved semantic choice returns to owner. |
| How are historical rows accumulated and terminalized so a multi-page/multi-row result cannot install partial success, silently choose conflicting rows, or claim empty after decode/pagination failure? | Phase 1 fixes row precedence and terminal outcomes, not worker/result batching or validation order. | Scoped V2 response/result integration and fault tests; Component 4 mapper/downloader evidence. | Atomic versus streamed admission design, conflict handling, and dangerous false-success proof. |
| How is the live ingress fence requested/captured/consumed across the engine and Component 5 fact seam, including queue saturation, epoch loss, cancellation, and a late old-generation result? | Phase 1 fixes the fence meaning, not the concrete command/fact handshake or finite ledger. | Scoped V2 fence/recovery integration tests and accepted Component 2/C5 contracts, without relying on unfinished private C5 code. | Exact engine-owned fence protocol and startup/recovery exit proofs. |
| Which predecessor mechanisms correctly treat successful empty, failed/canceled/fenced work, sparse results, re-acknowledgement, retry/supersession, and recovery exhaustion—and which recreate the known `deferred` freeze? | Phase 1 deliberately supersedes the predecessor's lifecycle ownership but V2 can still supply regressions and useful fact shapes. | Scoped V2 hydration/recovery tests and source, especially sparse/terminal accounting cases. | Reuse/reject ledger, explicit regressions, and slice/proof boundaries. |
| What hard component-local limits are justified for request concurrency, pages, attempts, bytes, rows, active generations/tokens, and diagnostic retention without preempting Component 8 production policy? | Phase 1 requires boundedness but delegates values and operations policy. | Component 4 accepted bounds, scoped V2 config/tests, and mathematical 16-hour/universe bounds. | Constructor/config validation and boundedness proof; production tuning remains deferred. |

No live observation, credential access, or provider request is authorized. If
offline evidence cannot establish a required current provider behavior, the
detailed contract records the limitation and stops rather than accessing a
network or secret.

## 6. Approved version 2 reconnaissance scope

The only permitted predecessor is:

```text
/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2
```

File-name discovery may identify exact candidates after this skeleton is
recorded. Content inspection is limited to declarations, call sites, focused
tests, and fixtures serving the roles below. Section 8 must record exact
paths/functions/tests/fixtures and hashes before any reuse decision.

| Approved code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| Massive aggregate REST request, envelope, pagination, normalization-call, and fake-HTTP tests | Which request/query/auth/continuation/retry/deadline/body/row rules and successful-empty cases are credible for a production hydration consumer of Component 4's mapper? | No reference/prior-close REST, WebSocket, live provider tests, credentials, replay artifact/compiler, or unrelated provider endpoints. |
| Hydration plan, generation, request-token, worker-pool, result, cancellation, and terminal-accounting declarations/tests | Which bounded value shapes and concurrency rules yield exact per-symbol terminal outcomes and fence stale/duplicate work? | No generic job framework, broad owner orchestration, readiness/health policy, checkpoints, API/UI, or database/event-bus design. |
| Fresh bootstrap and checkpoint-catch-up interval/fence integration tests | Which existing regressions demonstrate `[S,R)`, `[T0,R)`, live-tail concurrency, ingress-fence reconciliation, sparse/empty completion, epoch loss, and ordinary exit? | Checkpoint schema/storage/install mechanics; only the already-settled installed `T0` interface may be observed. |
| Same-process aggregate-gap planner/recovery declarations and focused offline tests | Which exact gap/overlap, generation supersession, new-epoch tail, fence, retry/exhaustion, and local/global consequence behavior is useful evidence? | No adapter-owned reconnect, recovery-owned ranker/watermark/readiness, T/Q features, production shutdown policy, or broad phase suite. |
| Focused failure, accounting, and predecessor-regression fixtures/tests | Which cases distinguish completed empty from failed/fenced/pending, prevent double terminal outcomes, contain historical/live or historical/historical conflict, and reproduce the old terminal `deferred` freeze? | No broad repository harvest, version 1, Git history, external docs not needed for an unresolved mapping, or unlisted integration behavior. |
| Component-local hard-bound configuration and tests | Which page/attempt/body/row/worker/token/generation ceilings are already exercised and separable from operations thresholds? | No readiness/freshness/capacity policy, performance claims, or production tuning without Component 8 evidence. |

The reconnaissance may follow a call only when necessary to understand one of
these named roles and must record the added exact declaration in Section 8.
Encountering a semantic dependency on unfinished Component 5 internals, an
unlisted subsystem, a new product rule, or a changed document responsibility
stops expansion for owner review.

### 6.1 Initial proof and slicing boundaries

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| Production REST consumer and terminal-result trust fixture | `DTE-AGG-01`–`DTE-AGG-04`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `ARCH-FLOW-01`–`ARCH-FLOW-04` | Exact requests and bounded validated pagination feed every row through Component 4's mapper; only complete terminal evidence can become value or empty. | V2 client/fake-HTTP evidence and Component 4 implementation reuse assessment. |
| Generation/token/terminal-accounting concurrency scenario | `DTE-EVENT-03`, `PG-OBS-02`, `LIFE-MODEL-01`, `LIFE-MODEL-02`, `LIFE-HYDRATE-05`, `LIFE-HYDRATE-07` | Registration, worker completion, cancellation, supersession, duplicates, and late results preserve exact identity and one terminal outcome without external mutation. | V2 planner/ledger code/tests and final bounds. |
| Fresh/checkpoint hydration and no-print reconciliation trace | `PG-RANK-02`, `PG-OPS-01`, `PG-OPS-02`, `DTE-RECOVERY-02`–`DTE-RECOVERY-04`, `LIFE-HYDRATE-01`–`LIFE-HYDRATE-07` | `[S,R)`/`[T0,R)` work and concurrent live tail reconcile through a real fence; empty and failures produce distinct symbol/work consequences; terminal work always exits hydration. | V2 startup/fence regressions and final engine seam. |
| Same-process gap recovery lifecycle trace | `DTE-RECOVERY-01`, `DTE-RECOVERY-03`, `DTE-RECOVERY-05`, `LIFE-RECOVER-01`–`LIFE-RECOVER-06` | Exact missing coverage, new epoch tail, retries/supersession, terminal fence, and ordinary evaluator restore currentness or explicitly suppress/terminate without a frozen state. | V2 gap planner/tests and exact overlap decision. |
| Components 1–6 offline production-aggregate lifecycle proof | `DTE-MODEL-01`–`DTE-MODEL-03`, `DTE-MERGE-04`, `DTE-COMMIT-02`, `LIFE-T11`, `LIFE-T12`, `LIFE-T19`, `LIFE-T20` | One binding, C5 ack/tail facts, C6 REST outcomes, Component 2 canonical path, and Component 3 evaluator compose without a second mapper, owner, watermark, or evaluator. | Completed contract interfaces and focused fake socket/HTTP composition; no live-provider claim. |

**Initial delivery assessment (later approved in Section 16):** Multiple
sequential slices are required.

**Reason and likely slice outcomes:** The external provider/REST trust boundary,
the engine-owned generation and terminal-accounting boundary, and startup/gap
reconciliation lifecycle are independently consequential and provable. A
likely sequence is: (1) reusable production REST acquisition and immutable
terminal facts; (2) engine-owned plan/generation/token/work ledger; (3) fresh
and future checkpoint hydration with fence/no-print consequences; and (4)
same-process recovery plus the offline Components 1–6 path. Reconnaissance may
refine or combine these outcomes but may not create concurrent slices or a new
ownership boundary.

## 7. Boundary-approval checkpoint

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, single ownership boundary, dependencies, scope, and explicit
      non-scope are unambiguous.
- [x] Settled identity, interval, normalization, precedence, fence, no-print,
      lifecycle-exit, accounting, and boundedness invariants prevent V2 from
      changing the architecture.
- [x] Unresolved questions are provider/client/concurrency/gap/proof details,
      not new product rules.
- [x] The modular document map routes three cohesive nonoverlapping concerns.
- [x] Version 2 scope is limited to the default checkout and exact named roles;
      no credentials, network, version 1, or broad orchestration are allowed.
- [x] No version 2 code, tests, or fixtures were opened while preparing this
      skeleton.
- [x] Likely proofs and multiple sequential delivery boundaries are identified
      without approving detailed behavior or implementation.

**Owner decision:** Approved 2026-08-06 by explicit pre-approval in the
initiating task. The owner authorized the recorded V2 reconnaissance before
Component 5 finishes unless C-6 requires its exact implementation. This
skeleton limits the early dependency to already-approved C5 acknowledgement,
epoch/loss, and live-ingress fact semantics, so finished C5 private
implementation is not required. The same message therefore records the exact
stable-interface exception for early Sections 8–19 drafting. Any semantic
change to those approved interfaces or need to inspect unfinished C5 private
implementation stops for owner review.

This boundary approval permitted only the recorded offline reconnaissance and
detailed contract drafting. It did not itself approve reuse, fixtures,
detailed Component 6 requirements, proof allocation, slices, advancement mode,
implementation, live provider access, or credentials. The separate completed-
contract approval recorded at the top of this parent now approves all of those
contract items except implementation, live provider access, and credentials.
