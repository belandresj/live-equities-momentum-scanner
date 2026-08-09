# Checkpoints and restart

**Status:** Component 7 reaccepted under the Version 1 Release Program after
the 2026-08-09 `P-C7-LIVE` fixture correction; `C7-S1` through `C7-S3`, all
eight primary proofs, cumulative verification, and the mandatory focused final
read-only review are clean and reopenable by later V1 integration evidence

**Owner boundary and reconnaissance approval:** Pre-approved 2026-08-07 by the
owner in the initiating Component 7 task, subject to the exact Sections 1–7
boundary, document map, and reconnaissance limits below

**Owner stable-interface exception:** None was needed. Component 6 passed final
component review on 2026-08-07 before version 2 reconnaissance began.

**Owner contract/reuse/test/slice-plan revision:** Revised 2026-08-07 through
the Version 1 Release Program. The eight semantic requirements remain, while
the V2 whitelist, fixtures, codec/mechanics, proof allocation, benchmark,
slice boundaries, reviews, and accepted lower-level decisions are revisable
when evidence requires correction.

**Advancement mode:** V1 continuous correction/acceptance. Correctable failures
reopen the affected item and continue; this component has no owner-response
state.

**V1 program authority:** The owner-approved
[`Version 1 Release Program`](../v1-release-program.md) replaces the former
C7-C11 unattended authority. It explicitly permits correction of this C7
contract, document map, whitelist, fixtures, codec/mechanics, proofs, slices,
thresholds, reviews, and accepted S1/S2 implementation decisions while Phase 1
checkpoint coherence and restart semantics remain fixed. The current local
host is the benchmark reference; C7 makes no broader capacity claim.

**Standing downstream decisions:** The minimum C8-C11 scope and deferrals are
owned by the V1 program and their component boundary plans. No credentialed
live-provider observation is authorized. Program completion is a locally
verified private release candidate, not live validation or deployment.

**Controlling Phase 1 requirements:** `PG-OPS-01`, `PG-OPS-02`,
`PG-REPLAY-01`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`,
`ARCH-FLOW-03`, `ARCH-FLOW-04`, `DTE-SESSION-02`, `DTE-CLOCK-05`,
`DTE-WINDOW-01`, `DTE-EVENT-01`, `DTE-EVENT-04`,
`DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, `DTE-COMMIT-04`, `DTE-CHECKPOINT-01`,
`DTE-CHECKPOINT-02`, `DTE-CHECKPOINT-03`, `DTE-REJECT-01`,
`LIFE-MODEL-01`, `LIFE-MODEL-02`, `LIFE-MODEL-04`, `LIFE-INIT-03`,
`LIFE-INIT-04`, `LIFE-INIT-05`, `LIFE-HYDRATE-01`,
`LIFE-HYDRATE-02`, `LIFE-HYDRATE-03`, `LIFE-HYDRATE-04`,
`LIFE-HYDRATE-05`, `LIFE-HYDRATE-06`, `LIFE-LIVE-02`,
`LIFE-LIVE-04`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`, `LIFE-END-02`,
`LIFE-END-03`, `LIFE-PUBLISH-02`, `LIFE-T01`, `LIFE-T02`, `LIFE-T03`,
`LIFE-T04`, `LIFE-T06`, `LIFE-T08`, `LIFE-T11`, `LIFE-T12`,
`LIFE-T15`, and `LIFE-T29`

**Approved dependencies:** the finally accepted Component 1 immutable
[`reference.Binding`](reference-data-and-session-binding.md), Component 2
[`ScannerStateEngine` and canonical-state contract](scanner-state-engine-and-canonical-state.md),
Component 3
[`aggregate feature/evaluator contract`](aggregate-features-qualification-ranking-and-accounting.md),
Component 4 [`aggregate replay contract`](aggregate-replay.md), and Component 5
[`Massive live-adapter contract`](massive-live-adapter.md). The owner-approved
finally accepted Component 6
[`aggregate hydration/recovery contract`](aggregate-rest-hydration-and-recovery.md)
supplies the installed-`T0`/`[T0,R)` interface.

## Contract document map

The modular layout separates the coherent engine-owned state boundary from the
external durable-storage boundary and end-to-end proof/delivery. All files
listed here form one Component 7 contract. Their responsibilities may be
revised through the V1 correction loop when evidence requires it.

| Document | Exclusive normative responsibility | Requirement/proof/slice coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Outcome, single ownership boundary, explicit non-scope, cross-cutting invariants, Sections 1–7, routing, approvals, and sole delivery ledger | All controlling Phase 1 IDs; Sections 1–7 | Every Component 7 task | Components 1–6 interfaces named above |
| [Checkpoint state and installation](checkpoints-and-restart/checkpoint-state-and-installation.md) | Exact coherent projection, schema semantics, compatibility/validation, atomic install, live/replay restart handoff, and engine-owned checkpoint facts | Sections 9–14; `C7-STATE-01`, `C7-INSTALL-01`, `C7-LIVE-01`, `C7-REPLAY-01`; `P-C7-STATE`, `P-C7-INSTALL`, `P-C7-LIVE`, `P-C7-REPLAY`; `C7-S1`, `C7-S3` | Projection, semantic validation, installation, or restart-state work | Parent; Components 1–4 and 6 |
| [Codec, storage, and cadence](checkpoints-and-restart/codec-storage-and-cadence.md) | Bounded encoding/decoding, durable atomic storage and discovery, asynchronous write-result identity, cadence, final-checkpoint policy, and restart objective | Sections 9–14; `C7-CODEC-01`, `C7-STORE-01`, `C7-CADENCE-01`, `C7-OBJECTIVE-01`; `P-C7-CODEC`, `P-C7-STORE`, `P-C7-CADENCE`, `P-C7-OBJECTIVE`; `C7-S2`, `C7-S3` | Codec, filesystem, persistence failure, cadence, or restart-target work | Parent; preceding state/install detail; Components 1–3 |
| [Proof and delivery plan](checkpoints-and-restart/proof-and-delivery-plan.md) | Exact reconnaissance/reuse ledger, current requirement/proof allocation, sequential slice plan, risk-based review, V1 correction/orchestration protocol, discretion, checklist, and drift audit | Sections 8 and 15–19; all eight Component 7 requirements/proofs; `C7-S1`–`C7-S3` | Contract revisions, assignments, correction, acceptance, and final review | Parent and both preceding details; Components 1–6 |

**Layout:** Modular contract with this parent and the three routed detail specs
above. Projection/install, external durable storage, and end-to-end restart
equivalence are independently reviewable trust boundaries and require three
sequential slices. This parent is about 3,400 words, above the
template's approximate 2,500-word routing alarm, because its advance-approved
Sections 1–7 must keep the exact boundary, reconnaissance limits, dependency
handoffs, and timing restriction together. All reconnaissance, detailed
requirements, trust/proof ledgers, and delivery material are routed out rather
than extending it.

**Routing rule:** Any requirement, evidence decision, proof, slice, or task not
unambiguously routed by this table requires a recorded parent-map correction
and then continues. Models do not guess among details or load unrelated
dependency material.

**Contract-wide coverage and acceptance:** Sections 1–7 retain the fixed Phase
1 boundary. All eight requirements and primary proofs pass after the
2026-08-09 `P-C7-LIVE` fake-provider ordering correction. The corrected S3
objective and focused final reviews are clean, so Component 7 is reaccepted.
This remains accepted evidence rather than a frozen design and may be reopened
by later V1 integration evidence.

## Authoritative delivery-state ledger

| Item | State | Evidence and required review | Recorded at | Next action |
| --- | --- | --- | --- | --- |
| Component contract | `accepted_reopenable` | The 2026-08-09 correction re-established deterministic `P-C7-LIVE` input without changing production behavior; focused repetition, dangerous-path rejection, affected race, ordinary verification, and final review pass | 2026-08-09 | Complete; reopen only if later evidence invalidates a claim |
| `C7-S1` | `accepted_reopenable` | `P-C7-STATE` and `P-C7-INSTALL` pass; S3 preserved their semantic/ownership evidence | 2026-08-07 | Complete |
| `C7-S2` | `accepted_reopenable` | `P-C7-CODEC`, `P-C7-STORE`, and `P-C7-CADENCE` pass. S3 added bounded lazy latest/previous selection without changing durability semantics; the isolated 100,000-symbol maximum-shape proof passes. | 2026-08-07 | Complete |
| `C7-S3` | `accepted_reopenable` | `P-C7-REPLAY` and `P-C7-OBJECTIVE` evidence remains valid. Corrected `P-C7-LIVE` passed 500 consecutive focused runs; the explicit descending-response counterexample passed 20 runs and preserved exact provider-failed terminal handling. | 2026-08-09 | Complete |
| Final component review | `accepted` | `gpt-5.6-sol` medium found no implementation or proof defect. It confirmed the failure signature comes from unordered fake-provider rows, equivalence assertions are unchanged, descending input still fails closed, and fixed checkpoint/C6 meanings are preserved. Its documentation-state finding was corrected and focused re-review was clean. | 2026-08-09 | Complete |

### 2026-08-09 `C7-LIVE-01` reopening record

The reopened claim is that `P-C7-LIVE` is a deterministic checkpoint-catch-up
equivalence proof. The observed failure is preserved by
`go test -short -timeout 2m -run '^TestC7LIVE01CheckpointCatchupEquivalence$' -count=100 ./internal/massive`:
14 runs reported two started items partitioned as one `completed_empty` and one
`provider_failed`, so those runs never reached the checkpoint-versus-oracle
comparison. The prior worker-count reduction therefore did not remove the
failure.

The root cause is a test-fixture defect, not HTTP terminal handling or
checkpoint catch-up. The fake endpoint advertised `sort=asc` behavior but
serialized two AAA rows by iterating a Go map. When the fresh-control interval
contained both timestamps, the map could emit the later bar first. The shared
C4/C6 acquisition path correctly rejected that response with
`symbol_interval_order_duplicate` and emitted one failed terminal. Local
`httptest` TLS handshake messages seen in the broader run do not explain the
repeatable producer accounting signature and are not checkpoint evidence.

The correction sorts the filtered fake-provider timestamps before encoding the
response; it does not change worker count, retries, production HTTP code,
terminal outcomes, engine ownership, `T0`, `[T0,R)`, merge behavior, or any
market semantic. A focused C6 regression supplies the dangerous counterexample:
an explicitly descending two-row response must emit no chunks, exactly one
`provider_failed` terminal with `symbol_interval_order_duplicate`, and no
value/empty success.

The distinguishing proof passed 500 consecutive corrected runs under the
two-minute bound. The descending-response counterexample passed 20 consecutive
runs. Affected race verification passed for `P-C7-LIVE` and the shared C6 REST
contract, and ordinary verification passed with
`go test -short -timeout 2m ./...`. The correction changes test code only, so
no production concurrency, HTTP, terminal, or checkpoint race claim was
reopened. The limitation remains that `P-C7-LIVE` uses deterministic local fake
TLS/HTTP and proves checkpoint/catch-up equivalence, not provider availability,
live transport latency, or deployed capacity. The focused final review found no
implementation/proof defect; its documentation-state finding was corrected and
cleanly re-reviewed.

### C7-S1 acceptance record

`C7-S1` adds one detached semantic checkpoint image and two closed typed engine
inputs. The existing FIFO consumer projects Components 1–3 exactly at a
committed/evaluated `T0`, or rejects without a writable request; installation
deep-copies and validates one candidate, rebuilds the ordinary evaluator in a
scratch graph, and performs one whole-state owner apply or none. Restored
aggregates carry installation origin without dead-process causal positions, so
the first valid new-run correction has ordinary Component 2 authority.

`P-C7-STATE` distinguishes committed state from forward state across
correction-horizon compaction, presence/absence/conflict, older and committed
marks, folded/mutable/invalid Activity including real `ATS=0`, provisional and
dirty qualification, the final latch, invalid-mark/coverage support, detached
mutation, standard JSON finiteness, and identical continuation. `P-C7-INSTALL`
rejects binding/schema/mode/time/sequence/count/value/provenance/price/Activity/
qualification/coverage contradictions without partial mutation; it also proves
candidate detachment, duplicate stability, new-run correction authority, one
whole-state apply, and that late accounting/publication failures cannot return
stale success. The required independent review initially found finite Activity,
Activity cross-field validation, final-completion, and proof-coverage gaps; all
were corrected in scope and the same reviewer returned a clean final focused
re-review.

Verification passed with `go test -race ./internal/engine ./internal/checkpoint`,
`go build ./...`, `go test ./...`, `go vet ./...`, focused uncached `TestC7`
proof runs, and `git diff --check`. Inspection confirms one checkpoint whole-
state apply and no filesystem, codec/store, new goroutine, provider, CLI,
live/replay composition, or timing implementation. Those deferred behaviors
remain allocated to `C7-S2`/`C7-S3`; the reviewed semantic image and typed seam
leave `C7-S2` valid as approved. There is no deviation, failed assumption,
success-invalidating inspection-only claim, escalation condition, or
substantive drift-audit `yes`.

### C7-S2 acceptance record

`C7-S2` adds the strict `scanner-checkpoint-v1` standard-library JSON codec,
one private manifest-authorized latest/previous local store, one sequential
writer with one replaceable pending request, and the narrow engine-owned
30-second committed-time cadence/result seam. Decode streams exact raw payload
bytes through private `0600` spooling, SHA-256, duplicate/unknown/trailing and
structural-path checks, context-aware semantic preflight, and one detached
decode without co-retaining the full artifact and graph. The store publishes
only reopened/revalidated immutable generations through the approved temp,
sync, close, rename, directory-sync, manifest, and bounded-cleanup sequence.

`P-C7-CODEC` proves deterministic payload bytes and raw digest, cancellation,
strict corruption and semantic rejection, exact required fixed-array presence
and length, dynamic cardinality preflight, a complete S1 image round trip, and
a literal 100,000-symbol combined-largest-valid allocation trace. That trace
includes maximum valid extrema, Activity references/mutable/targets and 57,600
contributions, qualification bars, proofs, and dirty state; 57,601 distinct
gate bars are structurally bounded but semantically impossible in the 57,600-
second session and are explicitly rejected. `P-C7-STORE` proves manifest-only
authority, latest-to-previous fallback, private path/file/link constraints,
every durability-step consequence, cancellation, and 256/16 bounded cleanup.
`P-C7-CADENCE` proves exact aligned versus nonaligned eligibility, ordinary
committed-time progress while persistence is paused, one-writing/one-pending
coalescing, bounded undrained shutdown, and terminal identity/accounting with
invalid, wrong, stale, and duplicate results fenced before consumption.

The required independent review initially found cadence arithmetic, fixed-array
truncation/defaulting, terminal-consumption, incremental-allocation, shutdown,
cleanup, permission, semantic-preflight, cancellation, and proof-shape gaps.
All findings were corrected in scope; the final focused re-review is clean.
Verification passed with checkpoint/engine race tests, repository build and
tests, `go vet ./...`, focused uncached proofs including the 85-second literal-
maximum codec run, and `git diff --check`. Inspection confirms exactly one C7
writer goroutine, standard-library-only dependencies, filesystem ownership
confined to the C7 codec/store/writer, and no provider, CLI, readiness,
live/replay composition, or objective benchmark. `C7-S3` remains valid as
approved. There is no deviation, failed assumption, success-invalidating
inspection-only claim, escalation condition, or substantive drift-audit `yes`.

### C7-S3 acceptance record

`C7-S3` composes one compatible manifest-authorized checkpoint with the real
C5 acknowledgement and C6 `[T0,R)` worker/ledger/fence path, and separately
continues the ordinary C4 runner from an engine-owned cutoff. Latest and
previous candidates are decoded one at a time under the store deadline; the
previous generation is opened only after latest storage or engine rejection.
Replay start authority distinguishes an explicitly scoped C4 artifact interval
from fresh-session recovery and checkpoint continuation: fresh recovery
requires `start == S`, while continuation requires the target engine's own
binding and installed `T0`. A foreign or forged public fact cannot authorize a
missing prefix.

The corrected objective preflights and byte-compares the exact timed artifact:
6,000 sorted symbols, 961 correction-tail records, every restart-required state
family, 1,674,157 bytes, SHA-256
`425f997f4ed161a0faf3a3c44cf0e2b5d577af55ae5c0ec5111898e5edd4c03c`,
6,000 checkpoint requests/3,000 normalized records, and an equivalent
6,000-request/180,000-record fresh control. Separate two-minute contexts bound
each checkpoint and fresh trial through binding, install, handshake, hydration,
fence, projection, close, and engine wait. Checkpoint restart measured 10.159,
10.290, and 10.286 seconds; fresh recovery measured 36.498, 36.641, and 36.668
seconds. The checkpoint median is 71.9% faster than fresh and the maximum is
below the 60-second setting.

The first final review found four consequential false-success/boundedness
defects; all were corrected and the focused re-review was clean. Full short
verification, the isolated maximum-shape proof, and focused race verification
pass. A single provider-failed terminal appeared only in one parallel
multi-package race run and did not reproduce in either the exact race test or
the complete Massive race package rerun; no semantic claim relies on that
failed run. Market-hours validation remains pending and is not a C7 or private
V1 RC prerequisite.

## 1. Outcome and user consequence

Component 7 makes an ordinary process restart reproduce the prior trustworthy
aggregate-derived scanner state at one committed boundary `T0`, then retrieve
only `[T0,R)` before returning to the same ordinary evaluator. It prevents a
restart from silently losing a valid qualification latch, reviving corrected
evidence, combining feature state from different times, or requiring a normal
full-session rebuild.

For the operator, a compatible complete checkpoint shortens restart while
remaining visibly noncurrent until catch-up evidence is sufficient. A missing,
incompatible, or corrupt checkpoint selects fresh hydration with a bounded
reason and no partial state. Projection, encoding, or write failure leaves live
ranking advancing and retains the previous complete checkpoint.

## 2. Scope and explicit non-scope

**In scope**

- Define one versioned, bounded checkpoint artifact and its exact compatibility
  identity for one immutable Component 1 session binding.
- Define the minimum sufficient canonical aggregate, coverage/no-print/unknown/
  conflict, feature, qualification, accounting-support, correction-tail, and
  committed state needed to reproduce Components 2–3 at exactly `T0`.
- Add one engine-owned coherent projection operation that copies immutable
  checkpoint state only after the ordinary commit/evaluation boundary is
  complete.
- Validate structural bounds, binding/schema compatibility, semantic
  invariants, completeness, and cross-field `T0` coherence before any restored
  value becomes canonical.
- Atomically install one validated checkpoint in a new engine, or install none;
  expose the resulting `T0` to Component 6's already-defined `[T0,R)` startup
  plan and the ordinary Component 3 evaluator.
- Encode, decode, discover, select, write, and atomically replace local
  checkpoint files outside the ordered engine path while retaining the last
  complete usable artifact on failure.
- Set an evidenced cadence and normal restart target, bound in-flight work and
  retained artifacts, identify stale write outcomes, and return bounded
  checkpoint status/reasons to the engine.
- Support the Phase 1 optional compatible-checkpoint start for aggregate replay
  without changing replay ordering or transport claims.

**Not in scope**

- Session/reference acquisition or binding construction; Component 1 owns it.
- Canonical aggregate identity/merge/correction rules, committed-time meaning,
  feature mathematics, qualification, accounting, ranking, or another
  evaluator; Components 2–3 own them.
- Replay artifact acquisition/format/order/clock, live WebSocket epochs or
  acknowledgements, REST pagination, hydration work, no-print proof, ingress
  fencing, or the mechanics of `[T0,R)` catch-up; Components 4–6 own them.
- Production freshness/readiness/capacity thresholds, retry and shutdown
  budgets, supervisor policy, or HTTP readiness mapping; Component 8 owns them.
  Component 7 owns only its evidenced checkpoint cadence/restart objective,
  hard safety bounds, and facts consumed by later operations policy.
- T/Q membership, causal coverage, ephemeral T/Q windows, or persistence of
  any T/Q measurement; Component 9 owns new-epoch warm-up and coverage.
- Public API schema, UI behavior, database, remote/object storage, replication,
  journal/event sourcing, generic serialization framework, plugin system,
  service split, credentials, or live provider access.

## 3. Ownership and dependencies

The single Component 7 ownership boundary begins when the engine seals an
immutable projection from one fully committed/evaluated boundary `T0`, or when
initialization supplies one bounded candidate checkpoint for validation. It
ends when an external writer returns a matching terminal outcome, or when the
engine atomically installs the entire validated state and exposes installed
`T0` as the initial boundary for live or replay continuation.

`ScannerStateEngine` remains the sole mutable owner of the active binding,
canonical state, feature/qualification support, lifecycle, `T`, evaluation,
accounting, and checkpoint cadence/request identity. The projector is one
explicit engine-owned stage. Codec, loader, and writer code own only bounded
immutable bytes, request-local filesystem state, and terminal facts; they
receive no writable engine reference and cannot decide compatibility,
canonical success, lifecycle, ranking currentness, or committed time.

Component 7 may choose a compact persisted representation, but it cannot
reinterpret dependency semantics. Installation must reconstruct the exact
dependency-owned state those contracts require or reject the artifact before
visibility. Component 6 alone plans and reconciles catch-up after a valid `T0`;
Component 7 cannot treat file load as present-day coverage.

## 4. Settled Phase 1 semantic boundary

| Boundary item | Settled meaning version 2 cannot change | Controlling Phase 1 IDs |
| --- | --- | --- |
| Binding and lifetime | A checkpoint belongs to one exact immutable session binding and one engine-run mode-compatible continuation. A different trading date, session bounds, universe identity, required prior session, prior-close policy/data identity, or incompatible schema cannot be installed. | `DTE-SESSION-02`, `LIFE-MODEL-01`, `LIFE-INIT-03`, `LIFE-REPLAY-01` |
| Coherent cutoff | `T0` is a committed aggregate watermark. The artifact represents all restart-required mutable contributions exactly as of `[S,T0)`—canonical identities, coverage/no-print/unknown/conflict evidence, derived sufficient state, correction status, and accounting support—or explicitly fails validation. `checkpoint_created_at >= T0` is persistence metadata; creation time, file modification time, `R`, and generated time cannot substitute for `T0`. | `PG-OPS-01`, `DTE-CLOCK-05`, `DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-03` |
| Restart equivalence | Equal binding, complete checkpoint state at `T0`, and the same continuation facts must reproduce the same dependency-owned canonical and aggregate-derived evaluation as uninterrupted processing. Current selected rows are not checkpoint authority; the ordinary evaluator regenerates output. | `PG-OPS-01`, `PG-OPS-02`, `DTE-COMMIT-02`, `DTE-COMMIT-03`, `DTE-CHECKPOINT-01` |
| Projection ownership | Only the engine may seal a checkpoint view, after the committed/evaluated transition is coherent. The view is immutable and detached from live maps; no separately sampled state or writer read can form a checkpoint. | `ARCH-OWN-01`–`ARCH-OWN-04`, `DTE-CHECKPOINT-01`, `LIFE-LIVE-04` |
| Persistence boundary | Encoding, self-validation, temporary-file writing, durability operations, and atomic replacement occur outside the ordered engine path. Requests and terminal results have versioned system envelopes, monotonic system positions, and exact request/binding identity. A matching terminal result is an operational fact only; success or failure cannot advance `T`, mutate canonical state, or block ordinary evaluation. | `ARCH-FLOW-01`–`ARCH-FLOW-04`, `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-CHECKPOINT-03`, `LIFE-LIVE-02`, `LIFE-LIVE-04` |
| Atomic startup install | Compatibility and complete structural/semantic validation finish before canonical visibility. A rejected candidate installs nothing. Missing/incompatible/corrupt state ordinarily falls back to an older compatible checkpoint or fresh hydration; only ambiguity that also invalidates the binding uses global suppression. | `PG-OBS-03`, `DTE-REJECT-01`, `LIFE-INIT-03`, `LIFE-T04` |
| Live continuation | A valid installed checkpoint selects `checkpoint_catchup`; after the current aggregate acknowledgement fixes `R`, Component 6 covers `[T0,R)` and reconciles the live tail through its fence. The aggregate with `window_start=T0` is continuation input, not checkpoint state. | `DTE-WINDOW-01`, `DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-02`, `LIFE-HYDRATE-05` |
| T/Q reset | Connection epoch, subscription membership, causal coverage, and ephemeral T/Q feature windows never survive restart, even if aggregate qualification and history do. | `DTE-CHECKPOINT-02`, `LIFE-INIT-03`, `LIFE-PUBLISH-02` |
| Replay continuation | Replay may install one compatible coherent checkpoint at `T0` and continue from its exact artifact cutoff; rejection falls back to `S` only when the replay artifact contains the complete required continuation. Live epochs/latency/TQ claims remain absent. | `PG-REPLAY-01`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`, `LIFE-T03` |
| End and final projection | A controlled stop may request a final view only from an already coherent committed boundary and cannot wait indefinitely for disk I/O. Late writer results cannot mutate an ended engine. | `LIFE-END-02`, `LIFE-END-03`, `LIFE-T29` |

This component introduces no new product rule, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, changed time-window meaning,
or duplicated responsibility.

## 5. Unresolved questions and evidence needs

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| What exact sufficient state must be projected for Components 2–3, including compacted presence/conflict facts, correction tail, folded Activity targets, feature dirty/fold metadata, qualification proof/dirty state, final latch, and classification support? | Phase 1 fixes coherent restart equivalence; approved component contracts state reproduction obligations but intentionally do not choose checkpoint fields. | Dependency-contract inventory, current accepted implementation invariants after C6 final review, and scoped V2 projection/round-trip evidence. | Exact semantic schema and construction/validation invariants without persisting redundant evaluator output. |
| Which values should be persisted directly versus deterministically rebuilt during atomic installation? | Both can satisfy equivalence if complete, bounded, and validated; Phase 1 does not choose the tradeoff. | Size/latency evidence, dependency constructors, V2 restore behavior, and differential uninterrupted-versus-restart proofs. | Install representation, validation order, and bounded startup cost. |
| What artifact version/compatibility policy and integrity envelope safely distinguish unsupported, corrupt, truncated, and semantically inconsistent checkpoints? | Schema evolution, checksum/framing, numeric bounds, and compatibility are delegated. | Scoped V2 codecs/fixtures/tests plus standard-library format capabilities and construction invariants. | Version envelope, decoder limits, compatibility matrix, rejection reasons, and migration policy if any. |
| How should discovery choose among current, previous, temporary, corrupt, and incompatible local artifacts? | Phase 1 requires a compatible checkpoint or clean fallback but does not define filenames, retention, or candidate order. | Scoped V2 storage/load tests and filesystem guarantees. | Deterministic discovery/selection and bounded retention policy. |
| What exact temporary-write, flush, rename, directory-durability, permission, and stale-result protocol preserves the last complete checkpoint across process or write failure? | Architecture requires atomic replacement but delegates concrete local storage mechanics. | Scoped V2 writer tests/fixtures, target-platform filesystem semantics, and failure injection. | Durable writer contract and matching request/result identity. |
| What in-flight/coalescing rule keeps projection and writer work bounded when `T` advances faster than persistence? | Phase 1 prohibits blocking/unbounded work but does not choose skip, replace-pending, or finite queue behavior. | Measured artifact size/encode/write latency and V2 cadence/backpressure evidence. | Hard in-flight bound, cadence eligibility, stale-result handling, and observable skipped/failed outcomes. |
| What cadence and restart target satisfy normal production continuity, and is a final controlled-stop checkpoint useful within the shutdown boundary? | `PG-OPS-01` requires evidenced values and rejects approximately 130 seconds of normal fresh reconstruction but supplies no numeric target. | V2 timing/configuration evidence, post-C6 representative offline benchmark data, artifact size/write cost, and later Component 8 capacity context. | Concrete cadence, restart objective, benchmark premise, and final-view policy. |
| Which live and replay restart scenarios distinguish correct installation/catch-up from an apparently successful mixed or incomplete restore? | Phase 1 states the outcome but delegates primary proof structure. | Scoped V2 restart tests/fixtures, C3 differential oracles, Component 6 accepted catch-up proofs, and replay continuation fixtures. | Minimal primary-proof families and provisional slice boundaries. |

These are component-local representation, storage, boundedness, and proof
questions. They do not reopen Phase 1 time, state, feature, ranking, recovery,
or lifecycle semantics.

All eight questions have current resolutions in the detailed contract. A later
implementation finding may invalidate one: revise the affected lower-level
resolution through the V1 correction loop and continue.

## 6. Approved and completed version 2 reconnaissance scope

The only predecessor is
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Version 1,
credentials, and live provider requests remain prohibited. This scope is
approved and was used only after Component 6 passed final
review. Exact findings, hashes, and reuse decisions are routed to
[Section 8](checkpoints-and-restart/proof-and-delivery-plan.md#8-version-2-reconnaissance-and-reuse-assessment).

| Proposed code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| Checkpoint model, projection, codec, decoder, validation, and focused unit fixtures | Which persisted fields and invariants were useful; where did V2 permit mixed-as-of state, omit restart-required state, trust derived output, or lack hard decode bounds? | Broad scanner state/ranking/feature implementation except exact declarations reached from checkpoint code; no assumption that V2 fields or schema are authoritative. |
| Local checkpoint discovery, loader, writer, atomic-replacement, and focused filesystem-failure tests | What candidate selection, corruption containment, temporary-file, rename/durability, permission, and previous-checkpoint preservation behavior has credible evidence? | Cache/reference storage, replay artifacts, remote/object storage, database, deployment, backup, and unrelated filesystem helpers. |
| Exact checkpoint restart/install and checkpoint-catch-up integration tests and their local fake fixtures | Which tests distinguish atomic valid install, invalid fallback, same-state reproduction, `[T0,R)` boundary behavior, live-tail reconciliation, and empty/sparse continuation? | V2 owner/planner/readiness/recovery architecture and production source ports; inspect only the minimum test-local call path needed to understand evidence. |
| Exact feature/qualification checkpoint round-trip tests and state declarations they directly exercise | Which correction-tail, Activity, range, coverage/conflict, dirty/fold, proof, and latch states were preserved or lost across restart? | General formula/ranking reconnaissance already owned by Component 3 and unlisted feature tests. |
| Checkpoint cadence/configuration declarations plus focused size, latency, or restart measurements if present | What empirical premise did V2 use, and is it strong enough to inform a new cadence/restart target or only identify a benchmark gap? | Readiness thresholds, operations orchestration, production logs, credentials, live observations, and broad performance suites. |
| Optional replay-from-checkpoint tests/fixtures, only if directly present in checkpoint or replay packages | Does V2 contain useful evidence for exact artifact continuation at `T0` without live transport claims? | General replay downloader/compiler/source code already owned by Component 4. |

Reconnaissance records exact paths/functions/tests/fixtures, provenance where
relevant, behavior strength, coupling to remove, reuse decision, and required
proof. Additional C7 source scope must be recorded before inspection and remain
inside the fixed boundary. Apparent fixed product/architecture tension follows
the V1 program's authority-order and strict-compatible-interpretation rule.

### 6.1 Initial proof and slicing boundaries

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| Coherent projection and in-memory round trip | `PG-OPS-01`, `DTE-CHECKPOINT-01`, `DTE-COMMIT-02`, `DTE-COMMIT-03` | One owner-sealed `T0` projection contains sufficient bounded dependency state; install plus identical continuation matches an uninterrupted oracle, including corrected/dirty/folded/final-latch cases. | Exact state inventory, rebuild choices, and V2 round-trip limitations. |
| Compatibility/semantic validation and atomic install | `DTE-SESSION-02`, `DTE-CHECKPOINT-01`–`03`, `LIFE-INIT-03` | Wrong binding/schema, truncation, corrupt bounds, mixed `T0`, or incomplete state cannot partially mutate a new engine or create false success. | Exact envelope, invariant matrix, and corrupt/incompatible fixtures. |
| Bounded codec and durable atomic replacement | `ARCH-OWN-02`, `ARCH-FLOW-01`, `ARCH-FLOW-04`, `LIFE-LIVE-04` | Encode/decode and filesystem failure never expose a partial artifact, lose the last complete checkpoint, alias engine state, block `T`, or accept a stale writer result as current success. | Format/storage choice, filesystem semantics, V2 failure evidence, and artifact-size bounds. |
| Live checkpoint restart composition | `PG-OPS-02`, `DTE-RECOVERY-02`, `DTE-RECOVERY-03`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-02`–`06` | Valid install at `T0`, exact `[T0,R)` hydration, concurrent live tail, terminal fence, and the ordinary evaluator reproduce uninterrupted aggregate-derived output while T/Q starts empty. | Component 6 final accepted interface/evidence and exact cross-component fixture. |
| Replay continuation and invalid fallback | `PG-REPLAY-01`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02` | A compatible checkpoint resumes at the exact replay cutoff; rejection restarts at `S` only with complete artifact evidence; playback scheduling cannot alter the result. | Scoped replay/checkpoint evidence and Component 4 fixture suitability. |
| Cadence, in-flight bound, and restart objective | `PG-OPS-01`, `ARCH-FLOW-01`, `LIFE-LIVE-04`, `LIFE-END-02` | The chosen policy is finite, does not impede ordinary evaluation, and meets a measured normal restart objective materially better than fresh reconstruction. | Representative post-C6 benchmark and any credible V2 timing evidence. |

**Final delivery assessment:** Three sequential implementation slices.

**Reason and slice outcomes:** `C7-S1` establishes coherent engine projection,
semantic validation, and atomic in-memory install. `C7-S2` adds the independently
provable persisted trust boundary: bounded codec, local latest/previous store,
asynchronous write results, and cadence. `C7-S3` composes those accepted paths
with Component 6 live catch-up and Component 4 replay continuation, then proves
the restart objective. Each slice leaves usable behavior and extends rather
than replaces the preceding owner/path.

## 7. Boundary-approval checkpoint

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, ownership, dependencies, scope, and non-scope are unambiguous.
- [x] Settled inputs/outputs/state and invariants prevent version 2 from
      changing the architecture.
- [x] Unresolved questions are genuinely delegated representation, storage,
      boundedness, cadence, target, and proof details.
- [x] Future version 2 reconnaissance is narrow, question-driven, and excludes
      broad predecessor orchestration.
- [x] No version 2 code, tests, or fixtures were opened while preparing this
      skeleton.
- [x] Initial likely proof boundaries are identified without a broad duplicate
      test matrix.
- [x] The likely need for multiple sequential slices and their dependency order
      are explicit and provisional.

**Owner decision:** Advance-approved 2026-08-07 for the exact boundary,
document map, and reconnaissance scope above. Component 6 passed final review
before that scope was exercised.

**Detailed-contract gate:** Revised by explicit owner authority on 2026-08-07.
Sections 8–19 remain complete current guidance, but V2 whitelist, proof
allocation, fixture, codec/mechanics, slice plan, reviews, and advancement are
agent-revisable when evidence requires correction. Fixed Phase 1 checkpoint
coherence, invalid-artifact rejection, continuation, and honest fallback are
unchanged.
