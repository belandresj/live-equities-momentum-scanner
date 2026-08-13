# Live fence finalization and burst-capacity correction

**Status:** S2 retained-tail correction implemented and locally accepted on
2026-08-12. Queue, incident, timing, and semantic evidence remain accepted;
the corrected full-universe cost, 60-cycle latency, and credential-free
216-frame/s capacity proofs pass. Another live-provider run remains separately
authorized and is not authorized by this correction.

**Boundary and completed-contract authority:** Direct owner request after the
2026-08-12 D3 retry selected `protocol/frame_slot_capacity` during hydration
fence finalization.

**Standing program decisions:** The
[`Version 1 Release Program`](v1-release-program.md) correction, containment,
verification-cost, and evidence-reuse rules apply. Accepted C2/C3/C5/C6/C8/C10
semantics remain authoritative; this document revises only their lower-level
delivery settings, diagnostics, and fence-finalization mechanics.

**Advancement mode:** Continuous, one write-capable slice at a time. Failed or
ambiguous proof reopens the lowest implicated slice and invokes the correction
loop; it does not create an owner-response gate.

**Controlling requirements:** `PG-OPS-02`, `PG-OBS-01`–`03`, `ARCH-FLOW-01`–
`04`, `DTE-EVENT-02`, `DTE-RECOVERY-02`–`05`, `DTE-COMMIT-02`–`04`,
`LIFE-HYDRATE-03`–`07`, `LIFE-RECOVER-01`–`06`, `C5-BOUND-01`,
`C6-FENCE-01`, `C6-START-01`, `C6-INTEGRATION-01`, `C8-MEASURE-01`, and
`C8-CAPACITY-01`.

**Approved dependencies:** Accepted C2/C3 engine/evaluator; C5 adapter FIFO;
C6 hydration/fence; C8 runtime/metrics; C10 immutable snapshot; accepted D1/D2
capacity diagnostics and D3 population correction.

## Contract map and sole delivery ledger

This is a compact single-file correction contract owning Sections 1–19. It is
the implementation entry point for this correction and depends on
[`live-scanner-recovery-narrow-fix.md`](live-scanner-recovery-narrow-fix.md)
for the broader recovery ledger and
[`live-ingress-first-cause-diagnostic.md`](live-ingress-first-cause-diagnostic.md)
for D1–D3 evidence. No Version 2 inspection is authorized or needed: the
current implementation, accepted cached fixture, and two recorded provider
observations answer the implementation questions directly.

**Layout note:** This contract is approximately 2,900 words, modestly above the
template's 2,500-word routing alarm. The excess is accepted because the queue
headroom is deliberately conditional on the same measured fence stall, and the
incident projection, timing identity, semantic-equality oracle, burst proof,
and two sequential slices must be evaluated together to prevent an implementer
from treating 8,192 slots as a substitute for correcting the critical section.
Splitting those coupled acceptance conditions would make every S1/S2 task load
both files and would create duplicate mutable status. If implementation evidence
adds another independently routable concern, convert this contract to a modular
set before expanding it further.

| Item | State | Evidence and required review | Next action |
| --- | --- | --- | --- |
| Correction contract | `accepted_current_plan` | Direct owner authorization plus the incident facts below | Implement S1 |
| `S1` — attribution and durable last-coherent evidence | `accepted` | `P-FENCE-ATTRIBUTION`; focused concurrency/diagnostic review clean after three correction rounds | Preserve in live incident evidence |
| `S2` — bounded fence correction and provisional burst headroom | `accepted_after_retained_tail_correction` | The retained-tail cause was corrected without changing canonical aggregates or evaluation semantics. Three exact full-universe boundaries staged in 322.106–397.037ms and applied in 0.797–1.658ms; 60 advancing cycles averaged 399.871ms and peaked at 601.020ms. A paced 60-second C5 proof sustained 216 frames/s with queue high-water 6, final queue zero, and no rejection. Focused semantics, differential/checkpoint proofs, ordinary/race/vet/diff, and the required final review are recorded below. | Local correction complete. Await a separately authorized provider retry; do not infer provider latency or a live SLA. |
| Market-hours retry | `not_authorized_by_this_contract` | Requires corrected local cost/capacity acceptance and a separate exact-date owner authorization | No provider request during this diagnostic/correction. |

## Sections 1–4 — outcome, scope, ownership, and fixed boundary

### Outcome

A checkpoint-off full-universe startup must be able to finish its exact ingress
fence without starving the live-frame consumer long enough to lose a raw
provider message. If any later terminal still occurs, one bounded incident must
retain both the exact transport cause and the last coherent market/accounting
state, including the D3 population-transition ledger.

### In scope

- Attribute final-fence time to hydration coverage installation/verification,
  per-symbol maintenance, staged full-population evaluation, apply, and
  publication.
- Preserve the last coherent immutable aggregate evaluation and its D3 ledger
  in the fixed-cardinality incident record before terminal suppression replaces
  the public product publication.
- Correct the measured fence-finalization bottleneck using existing canonical
  state, bitmaps, derived summaries, and the sole engine/evaluator path.
- Provisionally increase the production raw-frame queue from 1,024 to **8,192
  slots**, while retaining **128 MiB total queued bytes** and **8 MiB maximum
  per frame**.
- Prove exact causal ordering, accepted-frame accounting, bounded memory, and
  fail-closed saturation after the change.
- For the next owner-run live observation, revise the lower-level production
  selection to **32,768 slots** under the unchanged 128-MiB total and 8-MiB
  per-frame guards. This is additional transient headroom after the 8,192-slot
  setting itself saturated in the retained-tail run; it changes no prior proof
  result or product semantic.

### Explicit non-scope

- No dropped, sampled, coalesced, reordered, or selectively ignored raw frame.
- No second queue, state owner, watermark, evaluator, publication path, or
  readiness rule.
- No weakened hydration/fence, coverage, population, qualification, D3,
  suppression, or terminal semantics.
- No autonomous retry, silent restart, dynamic queue resizing, database,
  telemetry framework, or unbounded diagnostic label/history.
- No provider credential access or live request during implementation or local
  acceptance.
- No promise that 32,768 is a sustained-rate capacity claim or final minimum
  production slot count. Later evidence may tighten it.

### Ownership

The C5 adapter remains sole raw-frame FIFO/epoch owner. The C2 engine remains
sole ordered canonical, lifecycle, evaluator, watermark, and publication
owner. C6 owns hydration work/fence facts, not market state. C8 samples and
composes fixed-cardinality diagnostics; it does not infer readiness or repair
facts. C10 maps only the already-sealed immutable capture.

The correction introduces no new market rule, mark, coverage fact, product
population, T/Q-to-ranking dependency, or competing mutable owner.

## Sections 5–8 — observed evidence, questions, and reuse

### Evidence of record

The 2026-08-12 retry artifact is:

```text
log:
  var/run-private-scanner/d3-retry-20260812-143529.log
  sha256 10d5d349e4341feee48deebe37aa782787c6b732c21686dadb10144d267036c3

incident:
  var/diagnostics/live-ingress-20260812T184459.448917000Z.json
  sha256 25a7e8eeb452f5cd454031a20f33afde7861766bf6f231c0721af173e8797e0a
```

The run completed `5,511/5,511` hydration requests: `5,424` value, `87`
successful empty, zero failed/canceled/fenced, `5,119,255` rows consumed, and
`694` conflict/withdrawal rows. Immediately before fence finalization the raw
queue held seven frames and its prior high-water was 166. In the next scheduled
one-second sample the reader had accepted 1,195 additional messages while only
179 messages were dispositioned; the queue reached 1,024 and rejected the next
2,804-byte message. Queue bytes were 400,909 of 134,217,728 (0.30%). The active
delivery was `aggregate_ingress_fence` for at least 6.235631875 seconds; oldest
frame age reached 6.028534 seconds.

The fence ultimately reconciled through frame 77,435 and briefly published
`live/degraded_bootstrap/incomplete_population` at watermark 14:44:48 EDT,
then the ordered terminal suppressed the engine. The terminal incident retained
the prior lifecycle/ranking summary but not its population, qualification,
uncertainty, feature accounting, or D3 transition ledger. The operator's
`0.0/s` terminal rate compared two frozen post-rejection samples and did not
represent the preceding burst.

The earlier authorized D2 run reached 813/1,024 frames and only 424,594 bytes
without a capacity terminal. Together these observations prove count pressure,
not byte pressure, and invalidate 1,024 as adequate production burst headroom.

### Exact questions and decisions

| Question | Evidence needed | Decision enabled |
| --- | --- | --- |
| Which ordered fence substage owns the 6.2-second stall? | Production-path full-scale cached fixture with monotonic substage timings whose sum reconciles to the observed fence delivery | The narrow algorithmic correction; no speculative rewrite |
| Did D3 resolve the prior 486-symbol mechanism before suppression? | Last coherent immutable evaluation retained in the incident, including complete population/qualification/uncertainty/D3 counters | Distinguish D3 success, a specific predicate-rejection bin, and the separate `local_invalid` residual on the next authorized run |
| How much provisional burst headroom is justified? | Observed `1,196 messages/s` burst times a conservative `6.5s` measured-stall envelope gives 7,774 slots; next power-of-two is 8,192, still subordinate to the 128 MiB byte cap | One bounded production setting for an expensive retry; later reduction remains possible |
| Can fence cost return below the accepted responsiveness limit without changing semantics? | Exact full-scale cached 1x comparison plus dangerous counterexamples and race proof | Accept S2 or continue local correction before live work |

### Reuse decision

Reuse the accepted cached full-population hydration/fence fixture and existing
queue/incident accounting. Do not inspect Version 2 or invent a new fixture if
the accepted exact artifact is available. If that artifact is unavailable,
the agent may use the smallest deterministic generated 5,500–6,000-symbol
fixture that preserves full session coverage, conflict, no-print, later-live
mark, and burst shapes, but must record the substitution and limitation before
timing it.

## Sections 9–14 — required behavior, trust, accounting, and bounds

### `FENCE-ATTRIBUTION-01` — production-owned substage timing

Measure one ordered fence delivery with a monotonic production clock at these
nonoverlapping boundaries:

```text
coverage_finalization
  + symbol_maintenance
  + evaluation_stage
  + evaluation_apply
  + publication
  + residual_ordered_overhead
  = aggregate_ingress_fence_delivery
```

The timing family is fixed-cardinality and bound to engine sequence, hydration
generation, fence epoch/through-frame/marker ordinal, target `T`, and
publication ID when one is produced. It contains no symbol, payload, provider
text, URL, credential, stack, or arbitrary label. Production timing is always
available for this boundary; it is not a test-only armed clock.

Timing may observe but never decide ordering, coverage, lifecycle, evaluation,
publication, readiness, or terminal outcomes. A missing/nonmonotonic/overflowed
timing sample is diagnostic invalidity, not permission to publish false-ready.

### `FENCE-INCIDENT-01` — durable last coherent market projection

The first terminal incident must retain a bounded copy of the last coherent
pre-terminal immutable evaluation/publication sufficient to answer:

- publication identity, committed `T`, lifecycle and ranking mode/reason;
- primary population bins and covered/unresolved totals;
- qualification bins;
- uncertainty-origin bins;
- D3 `population_transition_diagnostic` denominator and all seven reason bins;
- evaluator accounting validity; and
- the exact latest fence timing identity/durations when applicable.

It may reuse the existing defensive replay/publication view or a smaller typed
projection, but may not retain rows, symbol identities, feature values,
provider data, or mutable aliases. The incident record remains one per process,
create-without-overwrite, owner-only, atomically published, and bounded. The
terminal fact and last coherent projection need not be globally atomic across
owners; each carries its owner identity/time, and no cross-owner equality is
inferred. The operator should render the pre-terminal D3 ledger when the final
suppressed publication no longer contains it.

Terminal rate rendering must select the most recent pair of history samples
with positive elapsed time and at least one relevant counter delta before the
cause. If no such pair exists, print `unavailable`, not `0.0/s`.

### `FENCE-COST-01` — bounded ordered finalization

The agent must first measure the current full-scale fence with the new substage
timings. Correct only the measured dominant stage(s). Permitted mechanics
include eliminating repeated bitmap/tail reconstruction, maintaining bounded
derived summaries already implied by canonical state, moving semantics-neutral
maintenance earlier while hydration/live inputs are already ordered, and
reusing a validated staged result at the same target and support identity.

If measurement selects an algorithmic/representation change, that correction
must preserve:

- the marker remains behind every raw frame through its captured sequence;
- every admitted raw item through the marker receives its existing engine
  disposition before coverage finalizes;
- the same fence input produces the same canonical state, consequence map,
  field statuses/reasons, qualification, D3 ledger, ranking rows/order, `T`,
  lifecycle, and publication identity semantics as the unoptimized oracle;
- later raw frames remain after the marker and cannot affect that fence result;
- no calculation uses provider receipt time as market time; and
- no work is deferred past a ready/current publication if it can change that
  publication.

If the measured path is already below the target and no algorithm or market
representation changes, the accepted existing C2/C3/C6 semantic proofs plus
the unchanged production-path fixture replace a vacuous before/after oracle;
record that proof-allocation correction explicitly. The accepted responsiveness
target is **less than 2.0 seconds** total ordered fence delivery on the exact
current-host full-scale cached 1x fixture. Record each substage, total,
allocation/heap effect, and frame backlog growth/drain. For an actual
optimization, also record the before/after semantic equality oracle. If the host cannot meet two seconds,
do not weaken the target or readiness: retain the 8,192-slot safety change,
record the exact residual stage, and continue the correction loop locally.

### `FENCE-BURST-01` — provisional 8,192-slot production headroom

Revise the C5/C8 delivery ceiling and production selection to:

```text
frame_slots = 8,192
max_frame_bytes = 8 MiB
total_frame_bytes = 128 MiB
```

This is an owner-authorized provisional operational bound based on the recorded
burst/stall, not a new provider or product semantic. The byte cap remains the
primary resident-payload bound. Queue storage must not preallocate 8,192 × 8
MiB; descriptor storage may be bounded to 8,192 and payload bytes remain
limited by the actual 128 MiB aggregate cap. Constructors still reject zero,
above-ceiling, frame-over-total, and overflow shapes.

A deterministic burst proof must hold the consumer at the real fence boundary,
enqueue at least **7,774 small raw messages** (the measured 1,196/s × 6.5s
envelope) without rejection, then release it and prove FIFO order, fence
linearization, exact frame/byte accounting, full drain, bounded heap, and joined
goroutines. A dangerous control adds one message beyond a deliberately smaller
configured slot cap and must still produce the exact fail-closed
`frame_slot_capacity` terminal. A byte-cap control must still select
`frame_byte_capacity` independently.

Do not claim that 8,192 survives arbitrary sustained imbalance. After one
successful authorized live observation, a later correction may reduce the slot
ceiling only from retained peak, arrival-rate, fence-time, and safety-margin
evidence; no automatic shrink is added here.

### Accounting and validation

Existing C5/C6/C8 identities remain unchanged. The new timing is overlapping
diagnostic accounting, not another work or symbol population. Its durations
must reconcile to total delivery without negative residual. The incident's
copied evaluation must independently satisfy the existing population,
qualification, uncertainty, and D3 identities before persistence; if it does
not, persist the terminal cause with an explicit bounded
`last_coherent_projection_invalid` diagnostic rather than serializing
incoherent market accounting.

### Evidenced edge cases

| Edge | Required behavior |
| --- | --- |
| 1,196-message burst during a 6.2s fence | 8,192-slot/128-MiB queue admits the conservative envelope while preserving order; fence optimization remains independently required |
| Small messages exhaust slots with ample bytes | Exact `frame_slot_capacity`; no byte-pressure inference |
| Large messages exhaust bytes below slot count | Exact `frame_byte_capacity`; no slot-pressure inference |
| Terminal immediately after a coherent degraded publication | Incident preserves its complete accounting/D3 ledger before suppression replacement |
| Terminal before any valid publication | Incident explicitly has no coherent market projection; no fabricated zero ledger |
| Later frame after the fence marker | It remains ordinary queued work and cannot enter the fenced prefix |
| D3 historical conflict plus later trusted mark | Population may resolve while feature conflict and qualification uncertainty remain exactly as accepted |
| Diagnostic clock failure | Market behavior unchanged; bounded diagnostic invalidity is explicit |

## Sections 15–17 — proofs, sequential slices, and implementation rules

### Primary proofs

| Proof | Claim and dangerous counterexample | Observable result and limitation |
| --- | --- | --- |
| `P-FENCE-ATTRIBUTION` | Real fence delivery is partitioned into the six nonoverlapping stages, and a terminal after a coherent publication retains that publication's valid D3/accounting projection. Counterexamples are double-counted time, a test-only clock, mutable alias, terminal replacement erasing evidence, and frozen-history `0.0/s`. | Exact timing identity/reconciliation; forced terminal JSON and operator line preserve the prior ledger; a no-publication control remains explicitly absent. Local deterministic proof, not provider timing. |
| `P-FENCE-COST` | Full-scale cached 1x fence is below 2s total. If an algorithm/representation changes, it is semantically identical before/after. Counterexamples are marker bypass, later-frame leakage, changed canonical/coverage/fields/qualification/D3/ranking, second evaluator, or work deferred past current publication. | Validated manifest, per-stage timing, allocations/heap, queue/drain, total <2s. An actual optimization additionally requires before/after state hash/typed equality; an unchanged already-conforming path reuses accepted C2/C3/C6 semantic proofs. Current-host fixture evidence, not an SLA. |
| `P-FENCE-RETAINED-TAIL` | A generated 5,694-symbol state preserves the observed Activity status and D3-transition bins while changing only 961 canonical tail records per symbol into equivalent folded-presence evidence. Counterexamples are a shallow tail, warmed invalid lookup, changed evaluation, and fixture construction attributed as evaluator work. | Folded/retained staged results are deeply equal; stage/apply and allocations are separate and CPU samples carry folded/retained stage/apply labels. This establishes local algorithmic causality, not provider timing or post-correction capacity. |
| `P-FENCE-BURST` | The 8,192-slot/128-MiB queue absorbs at least 7,774 observed-shape messages while the real fence consumer is held, then drains exactly. Smaller slot and byte controls remain fail-closed. | Zero rejection in envelope; exact FIFO/fence/accounting; bounded heap/goroutines; exact typed terminal in both dangerous controls. Synthetic observed-shape evidence, not arbitrary sustained capacity. |

### Sequential slices

| Slice | Coherent outcome | Allowed boundary | Acceptance and deferral |
| --- | --- | --- | --- |
| `S1` | Exact production fence attribution and durable pre-terminal D3/accounting evidence | `internal/engine`, `internal/operations`, `cmd/scanner`, incident schema/tests, and the three recovery docs/map only | `P-FENCE-ATTRIBUTION`, affected ordinary/race, focused concurrency/diagnostic review. No queue increase or optimization until accepted. |
| `S2` | Measured bottleneck corrected below 2s and provisional 8,192-slot production queue proven against the conservative burst | `internal/engine`, `internal/massive`, `internal/operations`, `cmd/scanner`, focused capacity/cached-fixture tests, and contract evidence | `P-FENCE-COST`, `P-FENCE-BURST`, ordinary/race/vet/diff, final focused review. Live/provider retry remains separately authorized and unexecuted. |

One write-capable implementer owns one slice at a time. Reviewers are read-only.
The orchestrator alone updates this ledger and stages or commits after all
workers/reviewers are quiescent. Preserve unrelated dirty-worktree changes and
do not stage, commit, push, rebase, amend, or create a PR unless separately
requested.

### Implementation discretion

The implementer may choose private timing/projection types, monotonic-clock
plumbing, derived-summary representation, and equivalent allocation mechanics.
The exact correction algorithm is deliberately selected only after S1 measures
the dominant stage. The implementer may revise lower-level proof mechanics or
split S2 if measurement reveals two genuinely distinct ownership boundaries;
record the revision here before proceeding.

**Prohibited changes:** increasing above 32,768/128 MiB/8 MiB without a new
recorded evidence decision; removing either count or byte admission checks;
blocking the socket reader; dropping/coalescing raw frames; concurrent canonical
mutation; changing evaluator, coverage, readiness, D3, lifecycle, terminal, or
publication semantics; retaining symbols/payloads in diagnostics; using live
provider access as a development loop.

**Correction triggers:** any semantic mismatch, fence still at or above 2s,
heap/resident growth inconsistent with the byte cap, burst rejection below the
approved envelope, race/order failure, incoherent incident projection, or
review P1/P2 reopens the implicated slice. Use the narrowest distinguishing
proof and preserve unaffected expensive evidence.

## Sections 18–19 — acceptance, review, and drift audit

S1 requires a focused `gpt-5.6-sol` medium read-only review because it crosses
the adapter/runtime terminal and immutable-publication boundary. S2 requires a
focused final review of fence linearization, sole-owner enforcement, memory
bounds, and false-ready containment. Reuse the same reviewer for focused
re-review when practical.

Verification order:

1. narrow affected proof after each correction, every wait/retry bounded;
2. S1 or S2 primary proofs with explicit command timeout;
3. `go test -short -timeout 2m ./...`;
4. affected `-race -short -timeout 5m` packages;
5. `go vet` for affected packages and `git diff --check`;
6. required focused review/re-review; and
7. update this sole ledger with exact results, dangerous counterexamples,
   limitations, success/failure walkthrough, and next-slice validity.

No local command may exceed 15 minutes without a recorded specific
justification. Validate fixture bytes, symbol count, interval, correction count,
and planned work before timing. Do not repeat unchanged expensive artifact
parsing when trusted reusable evidence can be opened once and streamed.

Final acceptance requires all three primary proofs, the 32,768/128-MiB/8-MiB
constructor and production settings, exact terminal containment, clean ordinary
and affected race verification, both required reviews clean after correction,
and no unresolved fixed-authority conflict. It authorizes no provider request.

**Drift audit:** The contract changes only diagnostic representation,
performance mechanics, and an owner-authorized bounded delivery setting. It
does not change the eligible universe, aggregate identity, time windows,
REST/live merge, no-print proof, population/qualification/ranking semantics,
readiness, T/Q independence, lifecycle, checkpoint meaning, API ratio units,
or public deployment scope. Version 2 was not inspected. The exact observed
live artifacts are evidence, not a reusable raw-provider fixture.

## Implementation and acceptance record — 2026-08-12

### S1 attribution and incident evidence

The engine now records one monotonic fixed-cardinality fence partition bound to
engine sequence, hydration generation, command token, epoch, through-frame,
marker, target, and publication ID. The total begins when C5 starts delivery,
so engine admission/owner wait is included. Coverage finalization, symbol
maintenance, evaluation stage, evaluation apply, publication, and residual
ordered overhead reconcile exactly; invalid clock/overflow/overlap evidence is
explicit and cannot affect market behavior.

The engine retains the last coherent normal immutable publication separately
from a later suppressed replacement. The incident copies only its bounded
publication/accounting projection: no rows, symbols, payloads, or mutable
aliases. Population, qualification, uncertainty, feature, and D3 transition
identities are revalidated before persistence; invalid accounting produces
`last_coherent_projection_invalid`. Adapter terminal observation remains first
cause across concurrent API sampling, and direct engine suppression lazily
records the engine terminal when no adapter delivery can own the cause. Rate
selection uses the newest pre-cause sample pair with positive time and a
relevant delta, otherwise `unavailable`.

The focused S1 review found and corrected three consequential issues: an
initial total omitted engine admission wait; direct engine terminal paths did
not all latch an incident; and rate selection could include a post-cause
sample. Re-review then found two narrower races: UTC conversion stripped the
monotonic reading, and a concurrent API sample could preempt the adapter's
exact terminal. The final implementation preserves the monotonic `time.Time`
and holds the adapter active-delivery diagnostic until operations latches the
immutable terminal. Focused re-review is clean.

### S2 measured cost and burst capacity

The accepted exact cached fixture was available: binding
`session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d`,
5,502 planned valid-prior symbols, 7,581,690 selected hydration rows, 5,439
value terminals, 63 empty terminals, and the prepared 17:15 EDT prefix from
artifact `sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`.
The first attempt exposed a fixture-driver defect: it deferred all symbol
terminals until after the complete row stream, allowing early symbols to age
past bounded retained-tail support. The proof driver was corrected to emit each
terminal immediately after that symbol's final chunk, matching production C6
progression; no market semantics changed.

The corrected exact 1x run passed in 277.31 seconds overall. Hydration took
2m53.415s, terminal completion 71.635ms, the real ingress fence 361.096ms, and
first-ready observation 19.125µs. The production timing partition was:

```text
coverage_finalization        68.744042 ms
symbol_maintenance           63.775667 ms
evaluation_stage            176.291500 ms
evaluation_apply             24.492750 ms
publication                   0.019208 ms
residual_ordered_overhead      0.033499 ms
attributed engine delivery   333.356666 ms
```

The current production path is therefore already 83.3% below the 2.0-second
limit after the accepted per-symbol hydration-finalization mechanics; no
additional algorithmic rewrite was justified. Because S2 changed no engine
algorithm or market representation after measurement, the planned
before/after semantic-equality oracle is not applicable: the unchanged
production-path fixture's accepted C2/C3/C6 semantic proofs remain the oracle,
while this proof distinguishes only current fence cost and integration
accounting. This proof-allocation correction removes no semantic assertion and
adds no alternate evaluator. It processed 32,353 live frames
with zero rejection, exact read/admit/disposition equality, peak raw queue 529,
`live/qualified_current`, 20 rows, reconciled accounting, and backend ready.
Ten later automatic cycles ranged from 80.658ms to 227.987ms total engine lock.
This is current-host cached-fixture evidence, not a provider SLA.

The accepted D4 C5 hard slot ceiling and production selection were 8,192 frames with
the unchanged 128 MiB aggregate byte cap and 8 MiB frame cap. Queue allocation
reserves descriptors only; payload residency remains actual-byte bounded. A
7,774-frame deterministic FIFO proof admitted the full conservative envelope,
placed the real fence marker after frame 7,774, drained in exact sequence, and
reconciled to zero queued frames/bytes. The production runtime composition also
held its real hydration consumer until all 8,192 slots were occupied, then
released, reached `live`/ready, and retained exact accounting. The 8,193rd-frame
control selected `protocol/frame_slot_capacity` with queue high-water 8,192;
the independent byte control selected `frame_byte_capacity` below the slot
limit. No arbitrary sustained-imbalance claim is made.

The initial final review correctly rejected the queue-only envelope and
pre-hydration runtime hold as insufficient proof of post-marker isolation. The
corrected proof dequeues a real C5 ingress marker, holds that active delivery at
the package-private adapter-to-engine proof seam, admits 7,774 later valid raw
frames behind it, then releases the marker. The fence publishes first; every
later frame receives a strictly greater engine sequence; the fence publication
ID/watermark remain unchanged until a later evaluator boundary; FIFO drains to
zero with reconciled count/byte accounting, bounded allocation, and joined
cleanup. The same review corrected the then-current C5 component text from its obsolete
512/64-MiB limits to 8,192/128 MiB and aligned `P-FENCE-COST` with the fact that
no algorithm changed after measurement. Normal and race proofs pass, and final
focused re-review reports no P1/P2 finding.

### Retained-tail causality correction — 2026-08-12

The next authorized provider run completed hydration but filled the 8,192-slot
queue during a 15.076348417s fence: stage was 8.316602042s and apply was
6.470905458s. That live evidence reopened only `P-FENCE-COST` fixture
representativeness. The cached fixture had retained one tail record per symbol,
did not reproduce the observed 837 bootstrap-unknown transitions, and did not
retain the 509 terminal-invalid Activity lookup shape.

`TestLiveRetainedTailCostAttribution` now generates 5,694 symbols with 961 tail
map entries each and the live Activity partition: 1,478 warming,
417 current, 2,370 no-target unavailable, 920 history-incomplete unavailable,
and 509 state-bound invalid. It also reproduces 190 bootstrap-unknown symbols
trusted by a later live mark and 647 rejected for incomplete post-mark
coverage. The matched folded control carries identical exact coverage and
feature operands through existing folded presence and Activity target state.
The complete staged evaluator results are deeply equal before timing is
accepted.

The low-memory control can share two immutable timestamp/value templates, but
the acceptance profile selects `LIVE_RETAINED_TAIL_UNIQUE_RECORDS=1`. In that
mode every symbol owns a contiguous 961-record slice with its own canonical
symbol identity and independent map. This retains bounded allocation count
while reproducing the live working-set and pointer locality rather than warming
two shared pointees.

The bounded profiled command was:

```text
env LIVE_RETAINED_TAIL_UNIQUE_RECORDS=1 \
go test -v -count=1 -run '^TestLiveRetainedTailCostAttribution$' \
  -timeout 2m \
  -cpuprofile var/live-retained-tail-cost/cpu.out \
  -memprofile var/live-retained-tail-cost/mem.out \
  ./internal/engine
```

The final unique-record, phase-labelled run passed in 15.06s. The folded
control measured 66.076ms stage, 408.442ms apply, and 474.518ms total. The
retained state measured 1.791203s stage, 10.402966s apply, and 12.195134s
total: 27.11x and 25.47x inflation, respectively. Within the 1.35s of sampled
retained-stage CPU, `latestMarkBefore` owned 0.93s, price-range evaluation
owned 0.37s, and `exactAggregateCoverage` owned 0.33s. Within retained apply,
`rebuildActivityReferenceLookup` owned 7.76s and
`exactAggregateCoverage` owned 6.82s. The profile therefore establishes a
joint retained-tail stage pathology and a specific coverage/rebuild apply
pathology; it does not attribute the live stage's entire 8.317s wall duration
to coverage alone. The generated unique-record stage is still 4.64x faster
than that live wall measurement, so post-correction capacity remains a required
local proof rather than an inferred result.

The selected smallest production boundary is therefore:

1. let `latestMarkBefore` return the already-maintained global latest record in
   O(1) when that record is strictly before the requested as-of time; retain
   the current scan when the latest record is at/after that time;
2. rewrite `exactAggregateCoverage` to combine folded presence and proven
   absence word by word, then use a compact, non-persisted derived view of the
   at-most-961-second canonical correction tail; fall back to direct
   `state.tail[second]` lookups if that view cannot prove its own validity;
3. do not rebuild or advance the derived Activity reference lookup once
   `boundExceeded` has made Activity terminal-invalid; and
4. preserve canonical tail, conflict precedence, exact interval semantics,
   checkpoint schema, ordering, evaluator ownership, and queue bounds.

Before another separately authorized provider run, require semantic equality
for latest-mark selection before/equal/after `at` and exact coverage across
insert/revision/withdrawal/conflict/compaction and checkpoint restoration; zero
allocation in the coverage query; retained-tail
stage <=500ms and apply <=250ms for three boundaries; 60 cycles with mean
<=500ms and maximum <=1s; and a credential-free paced C5 run sustaining at
least 216 frames/s with nonpositive 60-second queue slope, final queue zero,
high-water <=1,024, oldest frame <=250ms, and no rejection. The accepted burst,
incident, timing, normalization, and ordinary semantic evidence is reused.

### S2 retained-tail correction and acceptance — 2026-08-12

The three selected causes were corrected at their existing ownership
boundaries. `latestMarkBefore` now uses the maintained latest record in O(1)
when it is strictly before the evaluation time, while equality and future-time
cases retain canonical lookup semantics. Exact aggregate coverage no longer
constructs a full-session bitmap or scans the retained tail for every symbol:
the canonical aggregate owner maintains a compact 16-word derived view of the
at-most-961-second correction tail. The view is rebuilt on install, revision,
withdrawal, compaction, and checkpoint restore; it is neither persisted nor a
second source of truth, and invalid shape/cardinality falls back to sparse
canonical lookups. This costs about 144 bytes per symbol, roughly 0.8 MiB for
5,694 symbols, instead of about 39 MiB for a full-session bitmap per symbol.
Activity apply also skips derived reference-lookup rebuild for bound-invalid,
unavailable, and history-incomplete results while preserving the result and
valid retained prefix.

The first read-only review found that hydration's row-at-a-time direct
compaction path bypassed `compactSymbolLocked` and therefore did not refresh
the derived view after deleting the just-installed tail record. Cardinality
validation prevented false-positive coverage, but an evaluation between chunks
could reject exact folded coverage as incomplete. The direct compaction path
now rebuilds the view immediately. Its regression proves the record is folded,
the canonical tail is empty, the compact view matches that empty tail, and the
folded second remains exactly covered.

The first three changes exposed one remaining O(tail) stage scan in price-range
evaluation. Under the correction loop, the existing checkpointed price-range
high/low indexes were narrowed to own mutable tail evidence as well as folded
session evidence. Insert/revision, withdrawal, compaction, and checkpoint
restore maintain those indexes; the evaluator falls back to the canonical tail
for a malformed or pre-correction fixture. A full differential oracle proves
the indexed and canonical paths select identical extrema and feature results.

The original 60-cycle stretch target of 250ms mean and 500ms maximum was
falsified by the unique-record, full-universe heap proof after semantic work was
removed: garbage collection and immutable publication still produced about a
400ms mean. The recorded local acceptance was therefore revised to 500ms mean
and 1s maximum. That remains twice as strict as the controlling accepted
responsiveness limits of 1s mean and 2s maximum; it does not reinterpret those
product limits.

Three exact, independent full-universe retained-tail boundaries produced:

| Trial | Stage | Apply | Total |
| --- | ---: | ---: | ---: |
| 1 | 397.037ms | 1.220ms | 398.266ms |
| 2 | 322.106ms | 0.797ms | 322.913ms |
| 3 | 340.551ms | 1.658ms | 342.219ms |

The 60 advancing one-second-cycle proof completed with 16.430s total stage,
7.562s total apply, 399.871ms mean end-to-end latency, and 601.020ms maximum.
The credential-free production-composition capacity proof then paced exactly
12,960 frames over 60 seconds (216 frames/s, 25,920 aggregates) through the
8,192-slot/128-MiB/8-MiB queue. All 12,960 frames were dispositioned, midpoint
and final queue depth were zero, high-water was 6, maximum oldest-frame age was
zero, and no frame was rejected. Exact ingress/accounting reconciliation
passed. This proves local sustained headroom for the observed rate; it does not
measure provider latency or authorize a provider request.

### Owner-selected pre-retry operational headroom — 2026-08-12

Before the next owner-run observation, the owner directed loosening
unnecessarily tight delivery caps while retaining correctness boundaries. The
C5 raw FIFO ceiling and production selection are therefore 32,768 frame slots
under the unchanged 128-MiB total-payload and 8-MiB per-frame limits. The live
hydration cumulative-transfer allowance is 4 GiB, while its two-worker launcher
setting, per-worker resident-record bound, 16-MiB page bound, two-page maximum,
three attempts per page, and 15-second attempt deadline remain unchanged.
Aggregate connection recovery permits five attempts instead of three; every
failed attempt remains explicit and exhaustion still suppresses rather than
fabricating currentness.

The first full-ceiling race proof exposed that the historical slice FIFO
dequeued by shifting every remaining descriptor, making saturation drain
quadratic. The bounded FIFO now uses a fixed `frame_slots + 2` ring with O(1)
append/dequeue. Its pressure snapshot also stops at the first raw frame because
receipt times are monotonic and FIFO, avoiding an O(queue) lock while preserving
the classifying-versus-queued oldest-age minimum. A full-ceiling regression
admits 32,768 frames, rejects frame 32,769 by slot capacity, drains half, wraps
and admits another half-ceiling, then drains all 49,152 admitted frames in
exact sequence with zero retained frames/bytes and reconciled accounting. The
end-to-end first-cause proof separately composes an 8,192-slot exact
saturation/drain, fence, healthy control, and typed terminal behavior through
Runtime/adapter/engine. Thus 32,768 is proven at the concrete production C5
queue boundary, while the consequential cross-component path is proven at
8,192; no single Runtime proof claims full-ceiling provider composition.
These changes alter only bounded storage mechanics and delivery settings;
readiness, coverage, ordering, accounting, and market semantics are unchanged.
The isolated 60-second production-configuration rerun then sustained exactly
216 frames/s: all 12,960 frames and 25,920 aggregates were dispositioned,
midpoint/final queue depth were zero, queue high-water was 2, oldest age was
zero, and rejection was zero.

Final verification passed the complete ordinary repository suite, the complete
short race suites for `internal/massive` and `internal/operations`, `go vet
./...`, production scanner/dashboard/launcher builds, and `git diff --check`.
The required `gpt-5.6-sol` medium read-only review found the quadratic dequeue
and scheduler-dependent proof defects above; after correction, the reviewer
returned CLEAN/PASS with no remaining P1/P2. The 4-GiB allowance is a
configuration/boundary proof rather than a 4-GiB transfer trial, and five
attempts can extend unsuccessful recovery wall time relative to three but
remain finite and cannot make stale output ready.

Focused proofs cover latest before/equal/after boundaries, mixed folded/tail
coverage, gaps, conflict precedence, malformed records, zero query allocation,
Activity invalid/recovery behavior, price-range differential behavior,
checkpoint round trip/install, and the D3 conflict transition. Repository
verification passed `go test -short -timeout 2m ./...`, focused engine and
operations race verification, `go vet ./...`, and `git diff --check`. The final
`gpt-5.6-sol` medium read-only re-review returned CLEAN/PASS with no remaining
P1/P2 finding. It accepted the revised below-product-limit latency target and
the separate proof allocation between retained-tail evaluator cost and paced
C5 ingress capacity. No reviewer edit, provider request, or credential access
occurred.
