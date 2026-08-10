# Aggregate replay — Deterministic-core delivery

**Parent contract:** [Aggregate replay](../aggregate-replay.md)

**Normative responsibility:** Complete Component 4 requirement/proof routing,
the Components 1–4 deterministic-core milestone, sequential slice plan,
implementation discretion/prohibitions, completed-contract checklist, and
drift audit.

**Controlling requirements:** Every Phase 1 ID listed by the
[parent](../aggregate-replay.md), plus `C4-NORM-01`, `C4-COMP-NORM-01`,
`C4-DL-01`, `C4-ART-01`, `C4-ART-02`, `C4-SCHED-01`, `C4-RUN-01`,
`C4-FAIL-01`, `C4-CORE-01`, and `C4-PREFIX-END-01`

**Allocated slices:** `C4-S1`–`C4-S5`

**Document dependencies:** [Parent](../aggregate-replay.md),
[REST normalization and artifact trust](rest-normalization-and-artifact.md),
[replay source and clock](replay-source-and-clock.md), Component 1
[binding contract](../reference-data-and-session-binding.md), Component 2
[engine contract](../scanner-state-engine-and-canonical-state.md), and
Component 3
[evaluator contract](../aggregate-features-qualification-ranking-and-accounting.md)

**Approval state:** Approved 2026-08-06 as part of the complete parent contract;
this detail has no independent approval state

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-replay.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 15. Primary proof allocation

`C4-CORE-01` requires one representative complete artifact to traverse the
ordinary Component 1 binding, Component 2 replay/canonical/commit/publication,
and Component 3 feature/qualification/accounting/ranking path at two distinct
wall paces. Every logical-time state and immutable output must be equal; the
only permitted difference is wall duration. The result is explicitly replay,
T/Q-unavailable, and not production-live ready. This is the deterministic
aggregate-core milestone, not a live-parity, predictive-edge, or execution
claim (`PG-REPLAY-01`, `PG-REPLAY-02`, `PG-OBS-03`, `DTE-REPLAY-01`–`03`,
`LIFE-REPLAY-02`, `LIFE-REPLAY-03`, `LIFE-PUBLISH-02`).

Each Component 4 requirement has exactly one primary proof below. A proof may
exercise accepted Component 1–3 behavior, but it does not re-prove those
components' formulas or ownership. Tests use fake HTTP, the approved small
version 2 fixture adaptation, and repository-owned synthetic inputs. No
credential or live provider request is part of contract or slice acceptance.

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result and proof limitation | Approved fixture/evidence | Allocated slice |
| --- | --- | --- | --- | --- | --- |
| `C4-NORM-01` | `P-C4-NORM` — shared REST normalization table and construction inspection | Every evidenced `t,o,h,l,c,v,vw,n` row maps once to exact canonical values, one-second window, normalized signed zero, checked floor ATS, and REST provenance through one stateless mapper. Counterexample: missing/fractional `n` or `t`, invalid OHLC/VWAP, overflow, or signed-zero spelling silently defaults/rounds into success or changes the value. | Happy fractional/sparse/signed-zero rows equal exact values; every malformed variant returns its bounded class and no aggregate; source/package inspection finds one mapper and no request, coverage, replay, recovery, or engine authority. It does not prove provider availability, a compiler consumer, Component 6 request fencing/reuse, or live ATS parity. | Adapted `rest-second-bars.json`; Phase 1 formula/validity; official Massive fields; repository-owned invalid rows. | `C4-S1` |
| `C4-COMP-NORM-01` | `P-C4-COMP-NORM` — compiler consumer and sole-mapping construction proof | Every Massive row that can reach artifact compilation is decoded exactly once by the accepted S1 mapper, and only its successful immutable value can enter the artifact compiler. Counterexample: the downloader/compiler or a fixture adapter redecodes raw fields, defaults an invalid row, or constructs a parallel REST value. | Fake-provider and direct compiler inputs show the same mapper rejection/value boundary; source/package inspection finds no other production REST field mapping and no raw-row-to-codec path. It does not prove Component 6 production reuse, which its later contract must allocate after that consumer exists. | Accepted S1 mapper/proof; adapted fake-provider rows; repository-owned invalid rows. | `C4-S2` |
| `C4-DL-01` | `P-C4-DL` — bounded fake-provider download/accounting scenario | Exact fixed query, auth containment, pagination, retry/deadline/body limits, successful empty, cancellation, and symbol terminal identity hold. Counterexample: foreign/error/truncated work becomes complete empty or one failed worker still publishes. | Fake server observes exact requests and bounded attempts; redirect/foreign/third page and malformed responses fail; `planned=complete+failed+canceled`; all-complete including empty is the only compiler input. No live service capacity or provider SLA is proved. | Scoped v2 REST tests/fixture adapted; official endpoint/limit; fake HTTP only. | `C4-S2` |
| `C4-ART-01` | `P-C4-ART-BYTES` — golden compiler permutation and canonical-byte proof | Canonically equal normalized input sets under different worker/map/run order, temporary-run partition, and signed-zero spelling produce the exact normative JSONL members/encodings, golden bytes, ordinals, coverage, and artifact ID; partial synthetic mode cannot masquerade as complete final bars. Counterexample: completion/map order, alternate JSON encoding, or `-0` changes identity or sparse seconds are filled. | Golden byte/digest equality, exact member order, strict re-encoding, coverage lines including empty symbol, and mode restrictions pass across permutations. It does not prove disk atomicity or runtime playback. | Normative minimal golden artifact; repository-owned multi-symbol normalized corpus plus adapted sparse rows. | `C4-S2` |
| `C4-ART-02` | `P-C4-ART-TRUST` — whole-file validation, lease, and persistence fault/linearization matrix | Only a canonically sealed, binding-compatible, within-budget artifact with reconciled exact coverage reaches a validated handle, and only the stated atomic-no-replace/directory-sync outcomes reach their exact terminal classification. Counterexample: truncation, corruption, missing symbol, wrong binding, digest/count lie, partial mode, destination race, cancellation race, crash remnants, or sync uncertainty appears complete, overwrites existing bytes, or permits unbounded retained files. | Pre-publication faults/cancellation yield no new final path/handle; atomic no-replace plus successful directory sync completes; an existing-name race validates rather than overwrites; directory-sync uncertainty returns no handle but permits later adoption only after full validation; conflicting content is preserved; lease/remnant bounds prevent further file accumulation; valid same-open artifact passes twice. It does not prove cryptographic authorship, provider licensing, power-loss behavior beyond the filesystem guarantees, or process-OOM cleanup. | Repository-owned canonical/corrupt artifacts, normative golden artifact, filesystem/lease/cancellation fault seam; persistence invariant. | `C4-S2` |
| `C4-SCHED-01` | `P-C4-SCHED` — sparse delivery/clock/quiet-timer trace at two paces | Every logical second `S..R` has exactly one ordered group/timer; equal-time records precede timer; recordless seconds remain empty; clock and engine inputs are identical at unpaced and finite pace. Counterexample: wall lateness skips/coalesces a timer, changes order, or advances time before completion. | Exact logical trace, aggregate/timer dispositions, system/engine positions, simulated times, and per-group publications match; wall elapsed is excluded. It does not prove Component 3 output breadth or production timing accuracy. | Repository-owned sparse multi-symbol complete artifact; injected/short finite pacer. | `C4-S3` |
| `C4-RUN-01` | `P-C4-RUN` — replay lifecycle/coverage/commit/end scenario | Valid complete evidence alone moves `initializing -> replaying`, admits through canonical/evaluator path, classifies each finished slot for every binding symbol as present or artifact-proved absent, opens the existing replay commit gate after group completion, and ends after final seal/timer; partial synthetic evidence installs no complete absence and stays commit-ineligible while ordinary duplicate/revision semantics work. Counterexample: caller boolean, unclassified/partial coverage, or timer alone advances `T` or counts an empty symbol, or replay uses a second evaluator/publisher. | Complete run shows exact `LIFE-T03/T23/T24`, presence/absence accounting, nondecreasing supported `T`, Component 3 publication at `T`, replay/TQ-unavailable labels, and terminal ended; partial trace has no complete commit claim. Ownership inspection finds one engine/evaluator/publication. It does not prove live no-print tokens/readiness, checkpoint starts, or full capacity. | Small complete Component 1 binding/artifact with a successful-empty symbol and explicit partial synthetic correction trace. | `C4-S3` |
| `C4-FAIL-01` | `P-C4-FAIL` — artifact/clock/engine/cancellation containment matrix | Every consequential contradiction produces failed/suppressed terminal replay with no later restoration; cancellation produces canceled/controlled stop without artifact-end success. Counterexample: edited second-pass bytes, wrong ordinal/clock, engine rejection, EOF, or cancellation reports complete. | Exact bounded reason/last boundary, `terminal_replay_failure` for failures, no later admission/current claim, fresh-engine requirement, and distinct canceled result pass. It does not prove process restart orchestration or Component 8 shutdown policy. | Mutated same-open-file artifact, source/clock fault seams, accepted engine containment, cancellation boundaries. | `C4-S3` |
| `C4-CORE-01` | `P-C4-CORE` — Components 1–4 deterministic aggregate-core comparison | One representative complete artifact and identical binding/engine configuration produce identical logical results at unpaced and finite accelerated playback through the sole Component 1–3 path. Counterexample: pace/goroutine timing changes lifecycle, `T`, canonical state, qualification/features/accounting/rank, publication, or terminal result. | At every logical group, compare lifecycle transition, engine/system sequence, committed `T`, canonical proof view, Component 3 feature/qualification/population/ranking state, publication identity/content including simulated `generated_at`, and final result; require at least one qualified ranked symbol, one sparse symbol, and one successful-empty symbol. It does not establish live/REST ATS parity, provider chronology/latency, T/Q, trading edge, or executable expectancy. | Repository-owned representative complete artifact built through S1/S2; accepted Components 1–3 proof views; unpaced plus finite accelerated pace. | `C4-S4` |
| `C4-PREFIX-END-01` | `P-C4-PREFIX-END` — complete-source requested-end trust/lifecycle matrix | A complete `[S,R)` artifact can finish successfully at exact `O1`, `S < O1 <= R`, while only records/timers through `O1` reach the engine and full same-open suffix integrity still gates success. Counterexample: a later record is applied, corrupt/missing suffix or prefix evidence passes, a mismatched or wrong-clock end is admitted, cancellation is relabeled, or rejected ordinal/group/clock/engine work retains completion. | Prove exact prefix state and requested-end disposition for `O1<R`; `O1=R` byte/trace behavior and artifact-end disposition parity; no post-`O1` engine aggregate/timer; mutated suffix failure; missing/invalid prefix and requested-end contradiction failure; cancellation before terminal linkage remains canceled while cancellation after linkage cannot displace the accepted terminal result; rejection/suppression has no complete result; and the extended accounting identity closes with an exact intentional suffix count. It does not prove adversarial filesystem immutability, C12 API/UI composition, provider access, retained-B4 behavior, or large-artifact throughput. | Small repository-owned complete artifacts and package-private mutation/admission/terminal-link fault seams; owner authorization. | `C4-S5` |

The milestone comparison deliberately includes nontrivial output. A merely empty
or fixed-struct run would not show that Component 3's qualification, features,
population accounting, ordering, and immutable publication are on the replay
path. Conversely, this is engineering determinism evidence, not evidence that
the scanner predicts returns or has positive expectancy after costs.

## 16. Sequential implementation-slice plan

The accepted baseline used four slices because the component crossed four separately
reviewable boundaries in dependency order: a future-shared external row
mapping, provider I/O plus persisted artifact trust, engine clock/order/lifecycle
integration, and the runnable Components 1–4 milestone. Combining them would
make false provider/persistence success difficult to distinguish from engine
determinism. The owner-approved additive `C4-S5` is a fifth, focused correction
because requested-end persisted integrity and lifecycle completion must be
proved without reopening the accepted provider/compiler or milestone slices.

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies/entry state | Allowed ownership or files/packages | Approved v2 whitelist/fixtures | Acceptance record | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `C4-S1` | One stateless Massive REST second-aggregate mapper returns the exact provider-independent aggregate value or bounded rejection through the stable seam reserved for replay compilation and future Component 6. | `C4-NORM-01`; `P-C4-NORM`. | Finally accepted Components 1–3; approved Component 4 contract. | New focused `internal/massive` row/wire-normalization source and tests; only the existing `internal/engine` aggregate value/provenance types needed by the seam. No HTTP, artifact, replay, recovery, consumer integration, or engine mutation. | Behavior reference to the named `normalizeRecoveryBar` region/test; adapted exact `rest-second-bars.json` only. | Exact field/formula/signed-zero results; invalid table; construction/ownership inspection; affected and repository build/tests; narrow review result; deviations and S2 validity. | Downloader/compiler consumption, credentials/client, artifact, replay clock/lifecycle, milestone, and all Component 6 ownership and production-reuse proof. |
| `C4-S2` | A bounded offline fake-testable downloader uses the accepted S1 mapper, compiles the exact canonical versioned artifact, and atomically validates/publishes it with exact complete/empty coverage and partial-synthetic separation. | `C4-COMP-NORM-01`, `C4-DL-01`, `C4-ART-01`, `C4-ART-02`; `P-C4-COMP-NORM`, `P-C4-DL`, `P-C4-ART-BYTES`, `P-C4-ART-TRUST`. | Accepted S1 mapper and unchanged Components 1/2 interfaces. | Focused `internal/massive` offline client, new `internal/replayartifact` (or owner-approved equivalent), compile-mode wiring under `cmd/aggregate-replay`, and their focused tests/fixtures. No engine replay state. | Exact behavior regions/tests and adaptable JSON fixture named in Section 8; no other v2 file. | Sole-mapper consumer trace; query/failure/accounting trace; golden byte/digest/coverage result; persistence/lease/cancellation trust matrix; bounds; secret/path inspection; affected/repository tests plus required race; narrow review; deviations and S3 validity. | Engine start/groups/timers/commit/end, replay command execution, milestone, live calls, and Component 6 hydration/recovery and production-reuse proof. |
| `C4-S3` | A sequential validated-artifact source and single simulated clock drive typed replay start/aggregate/group-timer/end/failure through the existing engine, including artifact-proved per-slot presence/absence, successful replay `T`, final lifecycle, and containment. | `C4-SCHED-01`, `C4-RUN-01`, `C4-FAIL-01`; `P-C4-SCHED`, `P-C4-RUN`, `P-C4-FAIL`. | Accepted S2 artifact handle; accepted Component 2/3 engine/evaluator. | New focused `internal/replay` source/pacer and the minimum named replay evidence/admission/coverage state extension in `internal/engine`, with focused tests. No new evaluator, publication cell, or public API. | None; v2 runtime materials are rejected evidence. | Logical trace; presence/absence plus lifecycle/commit/end trace; failure/cancel matrix; sequence/state bounds; owner-path walkthrough; affected/repository tests plus required race; narrow review; deviations and S4 validity. | User-facing replay command completion, end-to-end representative milestone, live/TQ/checkpoint/readiness/API/UI and Component 6 no-print/work ownership. |
| `C4-S4` | The offline tool drives a representative complete binding/artifact through Components 1–3 and proves identical nontrivial immutable output at different paces, completing the deterministic aggregate-core milestone. | `C4-CORE-01`; `P-C4-CORE`. | Accepted S1–S3 and their proofs/reviews; no interface reinterpretation. | Replay-mode wiring under `cmd/aggregate-replay` and narrowly routed integration/proof tests in the Component 4/engine boundary. Production semantics may only compose accepted S1–S3 behavior. | None. | Runnable path; exact cross-speed comparison; nontrivial ranking/accounting output; all Component 4 proofs, repository build/test/race/vet, drift audit, deviations, and readiness for mandatory final review. | Provider credential use/live calls, production capacity/readiness, checkpoints, live/TQ, public API/UI, and any claim of predictive edge. |
| `C4-S5` | A validated complete artifact truthfully finishes at an exact operator-requested `O1<=R` without applying its suffix, while full source integrity, ordinary engine semantics, and cancellation/failure distinctions remain exact. | `C4-PREFIX-END-01`; `P-C4-PREFIX-END`. | Accepted S1–S4; owner-approved 2026-08-09 additive correction. | `internal/replayartifact/playback`, its `replayartifact.Handle` seam, `internal/replay`, and the minimum typed requested-end extension in `internal/engine`, plus focused tests and necessary local command flag wiring only. No C12 package/API/UI work. | None. | Exact dangerous-counterexample matrix; focused ordinary tests; repository short suite; proportionate race; required persisted-source/lifecycle read-only review and clean focused re-review. | C12 composition, provider requests/credentials, licensed-data reads, retained B4 artifact, live/TQ/API/UI changes, and capacity claims. |

Slice acceptance follows the parent ledger and the repository's delegated gate.
The advancement mode is `delegated` for every approved slice; no slice is
`manual`. A failed or
ambiguous proof, review, conformance walkthrough, whitelist boundary, or drift
check stops for the smallest owner decision.

Verification follows the repository tiers with these component-specific
requirements. S1 runs `P-C4-NORM`, affected package tests, and repository
build/ordinary tests; it does not require race solely for a stateless mapper.
S2 runs its four proofs, affected tests, repository build/ordinary tests, and
race detection over every new downloader/compiler package because it adds
workers and shared operation state. S3 runs its three proofs, affected engine/
replay tests, repository build/ordinary tests, and race detection across every
changed clock, queue, engine, and replay package. S4 reruns all nine Component
4 proofs, the cross-component
milestone, repository build/tests, repository
race tests, and `go vet ./...` before the separate final review. Exact command
paths may reflect the approved package layout, but none may omit the stated
claim. S5 runs `P-C4-PREFIX-END`, the repository short suite with its required
timeout, and proportionate race verification for the touched artifact/replay/
engine boundaries before focused re-review. No tier accesses credentials, a
live endpoint, licensed provider data, or the retained B4 artifact.

Every acceptance record must include:

1. the observable behavior and diff boundary completed;
2. allocated proof/verification results and their stated limitations;
3. invalid states prevented by construction versus runtime rejections;
4. one success-path and one dangerous-failure-path walkthrough;
5. boundedness/accounting and source-ownership inspection;
6. contract/whitelist deviations or `none`; and
7. whether the next approved slice remains valid.

**Independent slice-review trigger:**

- `C4-S1` requires a narrow independent review of exact numeric/timestamp/ATS
  mapping, signed-zero normalization, statelessness, and whether the value-only
  seam can be consumed later without importing replay or Component 6 ownership.
- `C4-S2` requires a narrow independent review of external-response false
  success, sole compiler use of the S1 mapper, exact persisted schema/coverage/
  identity, lease/remnant bounds, and publication/cancellation/durability
  linearization.
- `C4-S3` requires a narrow independent review of replay-evidence producer
  ownership, global ordinal/group order, simulated-clock linearization,
  successful central commit gating, lifecycle end/failure, and absence of a
  second evaluator/publisher.
- `C4-S4` needs no additional slice-specific review if its proof and cumulative
  verification are clean. After S4 acceptance, the mandatory separate
  read-only final Component 4 review examines the complete contract,
  implementation, all proofs, routing, bounds, ownership, and predecessor
  coupling before the parent may become `complete`.
- `C4-S5` requires a narrow read-only persisted-source/lifecycle review of full
  suffix trust, requested-end engine admission and cancellation linearization,
  disposition/accounting separation, and false-success counterexamples, plus a
  focused clean re-review after any correction.

Required independent reviews use the model/effort and focused re-review policy
in `AGENTS.md`. They provide evidence; they cannot approve new behavior,
whitelist scope, or contract changes.

## 17. Implementation discretion

Implementation may choose private Go identifiers, helper decomposition, error
wrapping, bounded external-sort/run mechanics, temporary-file naming, JSON
token implementation, HTTP transport construction, local command flag names,
and wall-pacer timer mechanics. It may reduce configured concurrency or memory
use. It may combine production files inside an allowed package when ownership
and proof routing remain clear. Tests may construct small repository-owned
fixtures programmatically. Persisted/clock fault injection must remain a closed
package-private test seam with no production configuration or callback path.

The following choices are not discretionary because they establish correctness
or trust:

- the shared mapper is singular and applies the exact timestamp, structural,
  floor-ATS, and provenance rules in `C4-NORM-01`;
- the Component 4 downloader/compiler consumes only successful S1 mapper values
  and contains no second REST field mapping;
- REST acquisition uses the fixed unadjusted one-second endpoint/query and the
  attempt/page/body/deadline/concurrency ceilings in the artifact detail;
- `aggregate-replay-jsonl-v1`, line-kind order, canonical-byte identity,
  complete per-symbol coverage, atomic publication, and partial-synthetic
  nonauthority are exact;
- replay uses one fully validated same-file source, one writer simulated clock,
  all whole-second groups/timers, immutable ordinals, and the typed engine facts
  defined in the replay detail;
- only the existing engine assigns sequences, mutates canonical/lifecycle/`T`,
  evaluates Component 3, and publishes; and
- complete/failed/canceled outcomes and every primary accounting identity are
  exhaustive.
- requested-end success is available only from a complete artifact after full
  same-open suffix validation; it has a distinct typed engine and result
  disposition, and intentional suffix records are explicit source accounting.

**Prohibited changes**

- No live WebSocket, T/Q replay, hydration/recovery planning, no-print token,
  generation/request ledger, ingress fence, checkpoint, readiness/capacity
  default, public API/UI, database, journal, service, generic event bus/plugin,
  or fake transport.
- No second REST row mapping, normalized aggregate schema, clock, watermark,
  canonical store, evaluator, ranking path, publication cell, or lifecycle
  transition owner.
- No adjusted same-session aggregate request, fabricated empty bar, rounded
  timestamp/count, live ATS label, worker/map/wall ordering authority, or
  partial artifact complete claim.
- No credential access, live provider request, raw provider capture, artifact
  upload/commit, or provider-license interpretation in implementation or tests
  without a separate exact owner authorization.
- No version 2 source/fixture beyond the approved whitelist and no version 1
  inspection.
- No modification of Components 1–3 behavior outside the exact Component 4
  typed replay extension already specified; a discovered mismatch returns to
  contract review.
- No reinterpretation of artifact header end, cancellation, controlled stop,
  suppression, or failure as requested-end success, and no suffix aggregate or
  timer admission after the selected requested end.

**Stop/escalation conditions**

- Massive documentation or an approved fixture contradicts the endpoint,
  pagination, timestamp, transaction-count, or empty-response behavior.
- Component 6 cannot reuse the S1 mapper without importing replay/artifact/
  recovery ownership, or a second mapper appears necessary.
- Canonical JSON Lines cannot be streamed/validated/atomically published within
  the fixed trust semantics or explicit budgets.
- The engine needs a caller-set timestamp/sequence/`T`, generic hook, second
  publisher/evaluator, or mutable artifact callback to support replay.
- Complete coverage/order/group evidence cannot be made nonforgeable enough to
  prevent an ordinary caller from falsely opening the commit gate.
- A primary proof fails, an independent review remains actionable after
  focused correction, a required interface differs from this contract, or a
  whitelist/map/slice responsibility must change.
- Verification would require credentials/live calls, provider-data sharing,
  destructive/external action, or a new operational/product rule.

## 18. Completed-contract acceptance checklist

- [x] The parent document map lists the complete four-document contract set and
      gives every document one exclusive normative responsibility.
- [x] The parent owns the only mutable delivery-state ledger; details only link
      to it.
- [x] Advancement is delegated for all slices and final review; no manual
      slice/final gate applies.
- [x] The modular layout routes independently reviewable normalization/
      persistence, runtime, and delivery concerns below the context targets.
- [x] Sections 1–19, every requirement, proof, and slice are routed exactly
      once; dependencies are explicit and acyclic.
- [x] Each slice can be implemented from the parent and its routed details
      without loading unrelated detail specifications.
- [x] Owner boundary and exact reconnaissance-scope approval is recorded.
- [x] Version 2 inspection stayed within the approved predecessor, file areas,
      functions/tests/fixtures, and exclusions.
- [x] Exact version 2 commit/file hashes, decisions, adaptations, fixture, and
      proof obligations are recorded.
- [x] Inputs, outputs, owned state, bounds, required behavior, terminal
      outcomes, and complete-versus-partial claims are explicit.
- [x] `aggregate-replay-jsonl-v1` fixes every member, member order, token,
      timestamp/number encoding, signed-zero normalization, byte count, digest
      boundary, and one normative golden artifact.
- [x] Downloader, artifact, and replay accounting identities distinguish
      primary populations from overlapping diagnostics.
- [x] Publication has one achievable atomic-no-replace linearization point;
      cancellation, existing-name races, temporary cleanup, directory-sync
      uncertainty, later adoption, and crash-remnant bounds have exact outcomes.
- [x] Every nontrivial edge case has Phase 1, provider, predecessor,
      persistence, or accepted-dependency evidence.
- [x] Every Component 4 requirement has one primary proof with its dangerous
      counterexample, observable result, limitation, evidence, and slice.
- [x] Consequential external, persisted, and engine-support trust boundaries
      define success evidence and fail-closed behavior.
- [x] Construction guarantees are separated from representable runtime
      validation failures.
- [x] Every requirement/proof belongs to exactly one coherent slice; ordering
      is acyclic and later slices extend rather than replace earlier ownership.
- [x] S1 proves only mapper semantics; S2 proves compiler consumption of that
      seam; the later Component 6 contract must prove its distinct production
      consumer after that implementation exists.
- [x] Every slice defines scope, deferred behavior, acceptance record, and
      exact independent-review trigger.
- [x] Implementation discretion, fixed decisions, prohibitions, and escalation
      conditions bound one assignment at a time.
- [x] The drift audit has no substantive `Yes`.
- [x] Owner completed-contract/reuse/test/slice-plan approval is recorded.
- [x] The exact version 2 implementation/fixture whitelist and delegated
      advancement mode received that approval on 2026-08-06.
- [x] `C4-S5` proves exact prefix success, artifact-end parity, full same-open
      suffix validation, terminal-fact cancellation linearization, exact
      requested-end clock admission, and honest intentional-suffix accounting.
- [x] Focused ordinary, repository short, proportionate race, vet, and required
      independent review/re-review evidence are clean for the correction.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | It implements approved aggregate-only replay, complete/partial evidence, and deterministic time; no trader-facing formula/readiness/TQ behavior changed. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | Component 4 owns only source/I/O/clock cursor state. Component 2 retains canonical/lifecycle/`T`/publication and Component 3 retains evaluation. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | Version 1 T/Q replay remains absent and replay output is explicitly not live-ready. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | Artifact and replay facts preserve Component 1 binding and the exact Phase 1/Component 2 semantics; the new facts only supply the previously deferred replay support predicates. |
| Did this add behavior without component-local evidence or explicit approval? | No | Provider mechanics come from scoped v2 tests/fixture and official docs; exact encoding, atomic no-replace publication, cancellation linearization, and crash-remnant bounds derive from deterministic/persisted false-success invariants. No speculative provider edge was added. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | The details cite shared semantics and define only Component 4 provider mapping, representation, scheduling, facts, bounds, and proofs. |
| Did this add machinery without an approved need? | No | JSON Lines, external bounded sorting, typed replay facts, and second-pass validation directly serve deterministic/persisted false-success boundaries; database, fake transport, generic frameworks, and extra queues are rejected. |
| Did version 2 drive the Phase 1 boundary instead of informing the detailed implementation contract? | No | The owner-approved Sections 1–7 predate inspection. V2 supplied narrow REST evidence; its absent/deferred replay design and legacy deterministic test were rejected. |

The owner approved the original complete contract, exact reuse/fixture whitelist,
nine primary proofs, four-slice plan, independent-review triggers, and delegated
advancement mode on 2026-08-06, then approved the additive tenth proof and
`C4-S5` correction on 2026-08-09. Current slice acceptance and authorization are
recorded only in the authoritative
[parent delivery-state ledger](../aggregate-replay.md#authoritative-delivery-state-ledger).
