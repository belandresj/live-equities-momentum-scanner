# Massive live adapter — proof and delivery plan

**Parent contract:** [Massive live adapter](../massive-live-adapter.md)
**Normative responsibility:** Complete requirement/proof ledger, sequential
slice plan, whitelist, implementation discretion, completed-contract
checklist, and drift audit
**Controlling requirements:** `C5-CLASS-01`, `C5-AGG-01`, `C5-TRADE-01`,
`C5-QUOTE-01`, `C5-STATUS-01`, `C5-ENGINE-01`, `C5-TRANSPORT-01`,
`C5-COMMAND-01`, `C5-BOUND-01`, `C5-RECONNECT-01`, `C5-LIVE-01`, and
all parent-routed Phase 1 requirements
**Allocated slices:** `C5-S1`, `C5-S2`, `C5-S3`; see the parent delivery
ledger
**Document dependencies:** Parent; provider classification and normalization;
transport, commands, and epochs; Components 1–4 as routed by the parent
**Approval state:** Inherits the parent completed-contract approval; not an
independent component authority
**Delivery state:** See the authoritative parent delivery-state ledger; do not
copy mutable status here

## 15. Primary proof allocation and complete requirement ledger

Each Component 5 requirement has exactly one primary proof and one slice. A
shared fixture file or test harness may host several named proof cases, but
each row below must retain its distinct claim, counterexample, observable
result, and limitation.

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result and intentional limitation | Approved fixture/evidence | Slice |
| --- | --- | --- | --- | --- | --- |
| `C5-CLASS-01` | `P-C5-CLASS`: strict mixed-frame classification fixture | Every examined item receives exact `(epoch,frame,array_index)` order and one closed family/disposition. Counterexamples: duplicate `ev`, unknown/missing family between valid items, trailing JSON, or a T/Q item before a later A/control under shed mode. | Earlier valid results remain ordered; first unclassifiable position emits ingress ambiguity and exact fenced remainder; recognized shed T/Q does not hide later A/control. Does not prove socket delivery or provider-current schema. | Adapted offline payload shapes from v2 `decode_test.go`, plus repository-owned duplicate/trailing/shed cases. | `C5-S1` |
| `C5-AGG-01` | `P-C5-AGG`: live aggregate normalization fixture | Exact A mapping, whole one-second milliseconds, `dv` preference/`v` fallback, fractional volume, live `z`/provenance, and structure. Counterexamples: duplicate/missing numeric, nonexact time, wrong interval, nonfinite value, or REST ATS substitution. | One exact `engine.AggregateInput` or atomic bounded rejection with zero value. Does not prove binding membership acceptance, transport latency, revision precedence, provider SLA, or live/REST ATS equality. | V2 `aggregateJSON`/schema test shapes; Component 2 aggregate type and Component 4 structural precedent. | `C5-S1` |
| `C5-TRADE-01` | `P-C5-TRADE`: trade normalization table | Complete identity, fractional economic size, participant/SIP fallback, bounded conditions/TRF/sequence/tape/lifecycle evidence. Counterexamples: `ds`/`s` mismatch, participant after SIP, future beyond 250 ms, incomplete TRF pair, unknown conditions, or structured correction evidence. | Exact immutable trade fact or local bounded rejection; unknowns remain unclassified and no feature eligibility is claimed. Does not prove actual provider lifecycle semantics, Tape Rate, coverage, or condition policy. | V2 `decodeProductTrade`, decimal-size test, offline lifecycle outcomes; repository-owned boundary mutations. | `C5-S1` |
| `C5-QUOTE-01` | `P-C5-QUOTE`: quote normalization table | Exact SIP/receipt/position, two-sided prices, ancillary presence, bounded condition/indicator/sequence/tape evidence. Counterexamples: crossed/locked quote, absent size, scalar/array/unknown metadata, or future timestamp. | Structural fact and quality evidence remain distinct; malformed core rejects locally; no zero is fabricated. Does not establish NBBO, Spread validity/time-weighting, or provider-current metadata policy. | V2 `decodeProductQuote` and offline quote disposition assertions; repository-owned boundaries. | `C5-S1` |
| `C5-STATUS-01` | `P-C5-STATUS`: status normalization/correlation fixture | Only exact phase/count/epoch context produces a status success fact. Counterexamples: provider prose URL, wrong phase, partial/extra/wrong-epoch success, duplicate recognized fields. | Bounded redacted success/failure/ambiguity fact with causal position; no provider prose. Does not prove write/ack race or engine lifecycle consequence. | V2 `DecodeStatusCount`, status/ack/privacy tests. | `C5-S1` |
| `C5-ENGINE-01` | `P-C5-ENGINE`: Component 2 connection/control lifecycle trace and construction inspection | Closed typed facts enter the one FIFO and only the engine accepts epochs/transitions. Counterexamples: adapter direct lifecycle setter, stale ack, A before ack, second active epoch, ack intent/write-only, connection loss in awaiting/hydrating/live/recovering, T/Q-only status failure. | Exact engine sequence/disposition/lifecycle record for every case; stale facts fence; post-ack A can enter canonical path; no second state/transition/publication path exists. Does not prove real socket, hydration completion, successful `T`, readiness, or Component 9 coverage. | Phase 1 transition table; accepted Component 2 proofs; repository-owned ordered trace. | `C5-S2` |
| `C5-TRANSPORT-01` | `P-C5-TRANSPORT`: offline fake-WebSocket handshake/reader/heartbeat scenario | One engine-commanded attempt executes connected/auth/A.* strictly, reads text/binary, redacts auth failure, and joins workers. Counterexamples: out-of-phase status, step/overall timeout, unsupported message, secret-bearing provider error, heartbeat failure, hung close. | Ordered facts and one bounded terminal result; no goroutine remains; credential absent from all output. Does not prove real network/provider availability, production duration values, entitlement, or capacity. | Adapted v2 fake-server tests; private connection interface. | `C5-S3` |
| `C5-COMMAND-01` | `P-C5-COMMAND`: deterministic command/write/ack interleaving scenario | A.* and paired T/Q commands serialize with one pending token and exact count. Counterexamples: ack before write return, write failure after ack, partial/extra/wrong-epoch/wrong-phase result, loss while pending, stale close. | Only both write success and exact ordered statuses complete; every other path has one failed/ambiguous/fenced outcome and no coverage claim. Does not prove Component 9 desired membership or provider entitlement. | V2 paired-ack, write-race, and fake command tests. | `C5-S3` |
| `C5-BOUND-01` | `P-C5-BOUND`: bounded FIFO concurrency/accounting scenario | Copy-on-admission, per-epoch sequence, count/byte/frame ceilings, close/gate wakeups, and accepted-work drain. Counterexamples: post-call byte mutation, count/byte saturation, oversize, regressing receipt, cancellation around linkage, queue full at terminal marker, optional T/Q before later aggregate/control. | Queue/accounting identities hold at pause points; failures leave queue unchanged and create epoch-loss evidence; all admitted frames disposition/fence. Does not establish production throughput, OOM survival, or Component 8 thresholds. | Adapted v2 queue tests plus Component 2 S1 concurrency proof style. | `C5-S3` |
| `C5-RECONNECT-01` | `P-C5-RECONNECT`: epoch termination/reopen lifecycle scenario | One first cause terminates an epoch, marker follows admitted frames, workers join, and only a later engine-issued open command causes allocation of a greater adapter epoch. Counterexamples: old reader after new open, duplicate terminal cause, controlled close aimed at future epoch, heartbeat ambiguity, canceled close. | Old facts fence, no ack/command state crosses, new epoch starts A.* only, and final cancel terminates within bounds. Does not prove retry/backoff/exhaustion policy or recovery gap correctness. | V2 consecutive-epoch/heartbeat/controlled-disconnect tests; Phase 1 lifecycle. | `C5-S3` |
| `C5-LIVE-01` | `P-C5-LIVE`: Components 1–5 offline fake-provider integration trace | Component 1 binding + C5 A.* handshake/status/frame + Component 2 FIFO/canonical state + Component 3 evaluator use one path. Counterexamples: aggregate at/before ack, mixed failed T/Q status plus A, stale epoch after reconnect, adapter-owned health/evaluator/publication, or T/Q fact changing rank. | Current ack transitions through the legal engine guard; only post-ack live aggregate receives canonical disposition with exact live position; mixed T/Q/control is explicit and aggregate result unchanged; source inspection finds one engine owner/evaluator. Does not prove hydration completion, live committed `T`, qualified/current ranking, readiness, REST reconciliation, actual provider behavior, or T/Q coverage/features. | Components 1–4 accepted interfaces; v2 fake-provider/mixed-frame evidence; repository-owned offline socket fixture. | `C5-S3` |

### 15.1 Construction guarantees reviewed at acceptance

The slice acceptance walkthrough must inspect and record that:

- only Component 2 writes accepted epoch, lifecycle, canonical aggregate state,
  committed `T`, evaluation, and publication;
- normalizers have no engine pointer, mutable callback, retained raw payload,
  or product clock;
- the live runner has one raw FIFO, one active epoch, one pending command, one
  per-epoch frame-sequence writer, and no retry loop;
- all aggregate/control engine facts use required admission capacity; later T/Q
  admissions remain optional and cannot consume Component 2's reserve;
- the WebSocket dependency and credential do not escape the adapter package;
  and
- Component 9 state/types, Component 6 recovery work, and Component 8 policy
  are absent.

Claims about absence of globals, callbacks, alternate setters, goroutine leaks,
raw retention, or unauthorized imports are partly inspection-supported; the
required independent reviews target those construction boundaries.

## 16. Sequential implementation-slice plan

Three slices are required. Pure external normalization is independently
provable without concurrency; engine control/lifecycle extension is a separate
sole-owner boundary; the final socket/queue/command composition is a distinct
I/O and concurrency boundary that depends on both.

| Slice | Coherent outcome | Requirements and proofs | Dependencies/entry state | Allowed ownership/files/packages | Approved v2 whitelist/evidence | Acceptance record and gate | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `C5-S1` | Stateless strict Massive A/T/Q/control classifier and normalizers produce complete provider-independent values/rejections with exact live positions. | `C5-CLASS-01`/`P-C5-CLASS`; `C5-AGG-01`/`P-C5-AGG`; `C5-TRADE-01`/`P-C5-TRADE`; `C5-QUOTE-01`/`P-C5-QUOTE`; `C5-STATUS-01`/`P-C5-STATUS`. | Accepted Component 1 binding accessors, Component 2 aggregate types, Component 4 aggregate-value seam. | `internal/massive` pure live-normalization files/tests; only necessary provider-independent value types may be added under the exact Component 2/Phase 1 boundary. No socket, goroutine, engine mutation, or third-party dependency. | `decode.go` exact symbols in Section 8; focused payload/assertion shapes from `decode_test.go`, named offline portions of `taq_correction_test.go` and `taq_provider_gates_test.go`. `trade_conditions_fixture.json` excluded. | Run all five proof tables, affected package tests, repository build/test/vet, formatting, whitelist/ownership inspection, and required narrow external-trust review. Record complete-value/rejection walkthrough; clean delegated gate advances to S2. | Engine connection/control facts, socket I/O, commands, queues, epochs/reconnect, production integration, all T/Q coverage/features. |
| `C5-S2` | Component 2 consumes closed connection/control/ingress facts in its sole FIFO and realizes the approved ack/loss lifecycle guards without an external transition authority. | `C5-ENGINE-01`/`P-C5-ENGINE`. | Accepted S1 immutable facts and all accepted Component 2 behavior/proofs. | `internal/engine` typed connection/control input, disposition, lifecycle, accounting/publication integration and focused tests only; minimal `internal/massive` type adaptation if required. No network or retry policy. | No v2 production code port. Phase 1 transition table, Component 2 contracts, and v2 fact shapes are behavior evidence only. | Run `P-C5-ENGINE`, affected engine tests, every earlier Component 2/3 proof the closed input/lifecycle extension can affect, repository build/test/vet, race for FIFO/state changes, source/ownership inspection, and required narrow lifecycle/sole-owner review. Clean delegated gate advances to S3. | Real socket, credentials, queue, command writer, heartbeat, reconnect attempt, production duration values, T/Q consumption, hydration/recovery work. |
| `C5-S3` | One bounded fake-testable Massive socket attempt executes engine-issued A.* and T/Q commands, classifies/drains frames, reports epochs/loss, and composes the offline Components 1–5 live-aggregate path. | `C5-TRANSPORT-01`/`P-C5-TRANSPORT`; `C5-COMMAND-01`/`P-C5-COMMAND`; `C5-BOUND-01`/`P-C5-BOUND`; `C5-RECONNECT-01`/`P-C5-RECONNECT`; `C5-LIVE-01`/`P-C5-LIVE`. | Accepted S1/S2 and Components 1–4; explicit test durations and small queue configuration. | `internal/massive` transport/queue/composition and tests; exact Component 2 calls only; `go.mod`/`go.sum` for sole approved WebSocket dependency. No production executable/runtime orchestration beyond the package composition. | Only the exact symbols and named focused tests in the transport detail's Section 8 ledger: `websocket.go`; `queue.go`; queue-only `config.go` fields; the listed tests in `websocket_test.go`, `config_queue_test.go`, `phase0_lifecycle_test.go`, `phase6_controlled_disconnect_test.go`, `phase3_taq_resilience_test.go`, and `owner_test.go`; and `taq.go` command/ack symbols as behavior evidence only. `owner.go`, all unlisted test functions, and `concurrent_stability_test.go` are rejected. | Run all five S3 proofs, complete Component 5 ledger, affected package/engine tests, repository build/test/vet, race for adapter/engine/concurrency packages, dependency/credential/whitelist inspection, and required narrow concurrency/order/interface review. Record worker join, accepted-frame drain, success/failure path, limitations, and clean drift audit. Clean delegated gate proceeds to mandatory final component review. | REST hydration/recovery and successful live commit/currentness; retry/backoff/readiness/shutdown production policy; checkpoint/API/UI; T/Q desired membership, coverage, features, pressure policy; live/provider observation and cutover. |

Slice rules from the parent process apply: one slice active, buildable repository,
one parent-ledger acceptance record, later slices extend rather than replace
earlier ownership, and no unused alternative path or temporary owner.

### 16.1 Independent review triggers

- `C5-S1`: required narrow `gpt-5.6-sol` medium review of the consequential
  external provider trust boundary, strict JSON/numeric/time normalization,
  rejection localization, and absence of feature/engine ownership.
- `C5-S2`: required narrow `gpt-5.6-sol` medium review of sole engine
  ownership, FIFO order, lifecycle guards, stale-epoch fencing, accounting, and
  publication atomicity.
- `C5-S3`: required narrow `gpt-5.6-sol` medium review of socket/command
  concurrency, acknowledgement linearization, frame/terminal-marker order,
  worker cancellation/join, credential containment, and Component 2 seam.
- After all slices, the mandatory separate read-only final Component 5 review
  reads the complete manifest and proof ledger. A clean slice review is not
  repeated merely to obtain another clean result.

## 17. Implementation discretion

Delegated routine choices include private file/helper names, strict JSON
decoder mechanics, immutable value construction, fixed enum representation,
private connection-interface method names, queue container/ring layout,
condition-variable/channel mechanics, error wrapping that preserves closed
reasons, and fake-socket harness structure.

Correctness-fixed choices are the Phase 1 field meanings, exact timestamp
units, 250 ms T/Q future tolerance, direct Component 2 aggregate type,
per-epoch causal tuple, strict duplicate recognized-member rejection, earliest
ambiguity/fenced remainder order, one socket, `A.*`-only epoch establishment,
paired deterministic T/Q channels, one command in flight, write+exact-ack
completion, engine-authorized/adapter-allocated epochs, one-attempt runner,
queue ceilings, and
Component 2 sole ownership.

The implementation may pin a compatible released version of
`github.com/coder/websocket` when `C5-S3` begins. The exact version and module
checksum become part of that slice diff and acceptance record. Any different
runtime dependency, a second dependency, or a public generic transport
interface requires owner review before use.

**Prohibited changes**

- No adapter-owned readiness/currentness, recovery plan/generation,
  `no_print_through`, T/Q desired/covered membership, feature state, ranking,
  watermark, evaluator, snapshot, or lifecycle transition API.
- No second normalized aggregate value, fake-WebSocket replay, second socket,
  generic event bus/payload/registry, callback-based scanner mutation,
  database/journal, service split, or plugin framework.
- No initial combined A/T/Q subscription, autonomous reconnect/backoff loop,
  caller-selected active epoch, silent status correlation, whole mixed-frame
  drop, raw provider text retention, timestamp-unit inference, or silent
  duplicate-member choice.
- No inspection or import of v2 files/symbols outside the whitelist, version 1,
  live/provider-gate functions, credentials, or network calls.
- No production retry/timing/readiness/pressure values, T/Q feature policy,
  REST recovery, checkpoint, API/UI, or public schema.

**Stop/escalation conditions**

- A required Massive A/T/Q/control field cannot be established from the
  recorded evidence without provider documentation or live observation.
- The provider status protocol cannot be safely correlated with one pending
  command and exact count, or requires concurrent in-flight commands.
- Implementation needs another v2 file/symbol, runtime dependency, socket,
  queue, mutable owner, or Component 1–4 interface change beyond the exact
  approved extension.
- A raw ambiguity cannot be localized under the contract, or a required
  aggregate/control fact could be silently lost under T/Q load.
- Queue ceilings cannot be represented safely, accepted work cannot be drained
  or fenced, workers cannot be joined, or credential material can reach
  observable output.
- A primary proof/review fails or exposes a new provider edge, scope change,
  contract deviation, or drift-audit `yes`.

These conditions stop for the smallest owner decision; they do not authorize
live requests or a contract change inside implementation.

## 18. Completed-contract acceptance checklist

- [x] The parent map lists this complete four-document contract set and gives
      each document one exclusive normative responsibility.
- [x] The parent owns the only mutable delivery-state ledger; details contain
      no duplicated slice status, proof results, or acceptance dates.
- [x] Advancement is fixed as `delegated` by completed-contract approval, and
      every failed/ambiguous gate remains manual.
- [x] Modular form is used because external normalization trust, engine
      lifecycle ownership, and socket/concurrency delivery are independently
      reviewable and produce three slices.
- [x] Each detail is below the approximately 5,000-word routing target. The
      parent exceeds the approximately 2,500-word target because the owner-
      approved Sections 1–7 boundary, exact role-filtered reconnaissance scope,
      and pre-approval record are one inseparable audit artifact; Sections
      8–19 are routed out and ordinary slice work need not load unrelated
      details.
- [x] Sections 1–19, all eleven requirements/proofs, and all three slices are
      routed exactly once with explicit acyclic dependencies and resolving
      links.
- [x] A slice can load the parent, its owning detail(s), and declared
      dependencies without loading unrelated Component 1–4 details.
- [x] Owner-approved Sections 1–7 boundary and reconnaissance scope are
      recorded.
- [x] Version 2 stayed within the default checkout and no credential, live
      request, version 1, or unlisted file was accessed. However, targeted range
      reads displayed the colocated excluded declarations recorded in both
      Section 8 ledgers. They were not used as evidence or whitelisted. The
      owner explicitly acknowledged and accepted this display-only variance in
      the completed-contract approval on 2026-08-06.
- [x] Exact v2 commit, paths, symbols/tests, hashes, decisions, adaptations,
      exclusions, evidence limitations, and proof obligations are recorded.
- [x] Inputs, outputs, local state, bounds, required behavior, failure and
      terminal outcomes are complete without duplicating Phase 1.
- [x] Adapter queue/attempt/command accounting identities are exact; event and
      diagnostic dimensions are labeled overlapping.
- [x] Every nontrivial edge case has v2 evidence, a Phase 1 invariant, an
      accepted dependency precedent, or explicit owner scope.
- [x] Every Component 5 requirement has one primary proof; no duplicate layer
      repeats the same claim.
- [x] External provider, command/status, queue, epoch, credential, and engine
      trust boundaries define concrete success evidence, rejection/
      containment, and dangerous false success.
- [x] Proofs name exact claims, counterexamples, observable results,
      participating paths, and limitations, including the absence of a live-
      provider claim.
- [x] Construction guarantees are distinguished from runtime validation and
      inspection-supported claims.
- [x] Every requirement/proof is allocated to one slice; ordering is acyclic
      and later slices extend rather than replace earlier behavior.
- [x] Every slice has one coherent outcome, precise allowed ownership/files,
      exact whitelist, deferred behavior, verification, walkthrough, review
      trigger, and delegated/manual gate.
- [x] Implementation discretion, prohibitions, and stop conditions are
      sufficient for one bounded assignment at a time.
- [x] The drift audit below contains no substantive `yes`.
- [x] On 2026-08-06 the owner approved Sections 8–19, acknowledged and accepted
      the recorded reconnaissance variance, and approved the exact whitelist,
      sole WebSocket dependency, eleven proofs, three slices, review triggers,
      and `delegated` advancement mode. The parent approval record initially
      authorizes only `C5-S1`.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | It implements the approved one-socket live source and provider normalization only; T/Q consumption, policy, and product fields remain Component 9. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | Adapter state is bounded transient I/O only; every accepted epoch/control/aggregate consequence remains in Component 2's FIFO and Component 3 evaluator. V2 Owner/Health/TAQ ownership is rejected. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | T/Q facts/status/shedding remain subordinate and cannot gate `T`, evaluation, ranking, or readiness. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | Exact Component 1 `[S,E)`, Phase 1 milliseconds/event-time rules, Component 2 live causal precedence/H, and sole commit gate are preserved. The explicit 250 ms tolerance changes validity only, never time. |
| Did this add behavior without component-local evidence or explicit approval? | No | Wire mappings and boundary cases come from the recorded v2 code/tests; strict duplicate/earliest-ambiguity rules derive from external-trust and causal-order invariants; queue ceilings come from v2 config. No live claim is made. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | Component 4 REST normalization remains distinct; Component 2 owns state/lifecycle; Component 6 owns REST recovery; Component 8 policy; Component 9 T/Q state/features. Details cite rather than redefine shared semantics. |
| Did this add machinery without an approved need? | No | One socket attempt, one FIFO, one classifier, one pending command, and one private WebSocket wrapper are necessary for bounded live I/O and exact generic-status correlation. |
| Did version 2 drive the Phase 1 boundary instead of informing detailed implementation? | No | Sections 1–7 were written and pre-approved before contents were opened. V2 state ownership/reconnect/orchestration was rejected where it conflicted. |

No substantive completed-contract drift required owner resolution. All three
slices later passed their recorded gates; current delivery state and the final
component-review result are authoritative only in the parent ledger.
