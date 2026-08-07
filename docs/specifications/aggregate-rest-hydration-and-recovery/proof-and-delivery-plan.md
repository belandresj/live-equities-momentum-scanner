# Aggregate REST hydration and recovery — proof and delivery plan

**Parent contract:** [Aggregate REST hydration and recovery](../aggregate-rest-hydration-and-recovery.md)

**Normative responsibility:** Complete Component 6 requirement/proof ledger,
exact proposed V2 whitelist, sequential implementation slices, implementation
discretion/prohibitions, completed-contract checklist, and drift audit.

**Controlling requirements:** all exact Phase 1 IDs enumerated in the
[parent](../aggregate-rest-hydration-and-recovery.md), routed here only for
proof/delivery allocation; `C6-REST-01`, `C6-WORKER-01`, `C6-BOUND-01`,
`C6-PLAN-01`, `C6-LEDGER-01`, `C6-MERGE-01`, `C6-FENCE-01`,
`C6-START-01`, `C6-NOPRINT-01`, `C6-RECOVER-01`, and
`C6-INTEGRATION-01`

**Allocated slices:** Approved `C6-S1`–`C6-S4`

**Document dependencies:** [Parent](../aggregate-rest-hydration-and-recovery.md),
[REST acquisition detail](rest-acquisition-and-terminal-outcomes.md),
[engine hydration/recovery detail](engine-hydration-reconciliation-and-recovery.md),
and the approved Components 1–5 interfaces routed by the parent

**Approval state:** Owner-approved 2026-08-06 as part of the complete modular
Component 6 contract; this detail is not an independent authority

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-rest-hydration-and-recovery.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 15. Primary proof allocation and complete requirement ledger

| Requirement | One primary proof | Claim and dangerous counterexample | Observable result and limitation | Approved fixture/evidence | Approved slice |
| --- | --- | --- | --- | --- | --- |
| `C6-REST-01` | `P-C6-REST`: strict shared-client fake-provider matrix plus sole-mapper construction inspection | Both offline and hydration consumers use one request/envelope/pagination core and one row mapper. Counterexamples: HTTP 200 non-OK, foreign ticker, adjusted response, duplicate envelope member, count mismatch, foreign/cyclic/third continuation, invalid last row, or second mapper. | Exact request/result reason and zero exposed rows on failure; instrumentation/source inspection finds one page decoder and `NormalizeRESTSecondAggregate` as the sole wire-row value constructor for both consumers. Does not prove provider availability/current schema or engine consumption. | Accepted Component 4 tests/contract; named V2 REST fake-HTTP shapes. | `C6-S1` |
| `C6-WORKER-01` | `P-C6-WORKER`: sealed-result/chunk/terminal table | A worker exposes no rows until full provider completion, then contiguous copied chunks and one exact value/empty terminal. Counterexamples: page two fails, result mutates after emission, missing/duplicate chunk, empty inferred from error, cancellation between last chunk and terminal. | Failed result emits zero chunks; success row/chunk counts reconcile; immutable mutation probes fail to affect facts; exactly one producer terminal fact. Does not establish engine applicability/fenced disposition or production throughput. | V2 streaming/progress cases adapted to strict C4 responses; construction invariant. | `C6-S1` |
| `C6-BOUND-01` | `P-C6-BOUND`: worker/backpressure/cancellation accounting scenario | Worker/page/attempt/body/record/resident/chunk bounds hold under concurrent completion and blocked engine admission. Counterexamples: ninth worker, response budget race, oversize last page, cancel during I/O/buffer/chunk/terminal admission, or worker goroutine outliving operation. | Maximums and worker joins are observed; every accepted item produces terminal or explicit closed-input cleanup evidence; no unbounded queue/result retention. Does not choose deployed capacity or whole-generation retry policy. | Component 4 bounds tests; V2 concurrency/cancel/generation-wire cases. | `C6-S1` |
| `C6-PLAN-01` | `P-C6-PLAN`: deterministic mode/interval/population construction table | Exact fresh/checkpoint/gap intervals and every valid-prior symbol are planned independent of rank; invalid context cannot partially register. Counterexamples: pre-session `R=S`, `T0=R`, V2 `E+1s`, V2 `T-16m`, invalid/missing prior symbol included, rank/top-20 omission, duplicate symbol/request, overflow, or compacted-present overlap. | Exact sorted tokens/intervals or atomic plan rejection; empty interval produces zero requests and terminal generation progress; no ranking input is callable. Does not prove HTTP or lifecycle exit. | Phase 1 formulas, Component 1 binding facts, Component 2 registration invariant, V2 planning counterexamples. | `C6-S2` |
| `C6-LEDGER-01` | `P-C6-LEDGER`: ordered chunk/terminal/supersession concurrency trace | One engine ledger validates token/result sequence and assigns exactly one terminal bin despite producer permutation, cancellation, repeated terminal, stale generation, or late result. | After every input, `planned=terminal+open`; at generation end the five terminal bins equal planned; stale/late evidence cannot change canonical/coverage/lifecycle or double count. Does not prove REST authenticity or no-print meaning. | V2 5,500-symbol, cancellation, stale-binding/generation cases; Component 2 FIFO/accounting proof style. | `C6-S2` |
| `C6-MERGE-01` | `P-C6-MERGE`: production historical-result canonical-disposition matrix | Every row uses existing Component 2 fill-only decisions and terminal provider success stays distinct from usable coverage. Counterexamples: historical overwrites live, unequal historical arrival-order winner, caller mutation, completed value after rejected row claiming coverage, or compacted presence relabeled missing. | Exact Component 2 disposition/values/conflict/coverage consequence and one C6 row bin; one engine map/evaluator path by inspection. Does not prove full lifecycle/fence or provider request correctness. | Accepted `ENG-AGG-01`; V2 conflict/remediation shapes. | `C6-S2` |
| `C6-FENCE-01` | `P-C6-FENCE`: raw-frame/marker/engine-FIFO linearization scenario | A marker appended after raw frame N prevents finalization until every item through N is classified and consumed. Counterexamples: frame before capture delayed, frame after capture, queue full, mixed frame, ingress ambiguity, stale token/epoch, loss before marker, or polling queue empty. | Matching complete marker is consumed after all required dispositions; later frames remain ordinary; every failure blocks old-generation completion and has exact cancel/fence consequence. Does not prove real socket/provider behavior or readiness policy. | Phase 1 fence rule; approved C5 raw queue/terminal marker; V2 pre/post/final-disconnect fence cases. | `C6-S3` |
| `C6-START-01` | `P-C6-START`: fresh and checkpoint-interface lifecycle trace | Live tail/timers remain active while complete-population work terminates; every terminal combination exits hydration exactly once. Counterexamples: sparse/empty population, local failure, zero qualified rows, degraded then exact, `R=S`, future `T0`, epoch loss, old late result, session end, or old `deferred`/120-second freeze. | Exact transitions, publication status, work/accounting/fence state, and continued ordinary timer/aggregate evaluation. Does not prove checkpoint validation/storage, deployed deadlines, or provider availability. | Phase 1 lifecycle scenarios 1–8/20; V2 fresh/interleaving/remediation regressions. | `C6-S3` |
| `C6-NOPRINT-01` | `P-C6-NOPRINT`: coverage composition matrix at exact boundaries | Empty becomes no-print only with `[S,R)` historical success plus continuous fenced live coverage to `T`. Counterexamples: HTTP error empty body, partial interval, failed/canceled/fenced work, pre-fence live print, earlier accepted mark, conflict, first print after no-print, and exact `R/T/E` boundaries. | Exact primary symbol category and work state; no synthetic aggregate/mark; later print triggers ordinary evaluation. Does not prove provider absence beyond scanner evidence or public API schema. | Product/data-time no-print invariant; V2 empty limitation as negative evidence. | `C6-S3` |
| `C6-RECOVER-01` | `P-C6-RECOVER`: same-process loss/reack/gap/retry/exhaustion lifecycle trace | Recovery requests `[T_supported,R)`, preserves state, closes currentness/T/Q, accepts new tail, supersedes old generations, and exits through ordinary evaluation or explicit suppression/termination. Counterexamples: request `T-H`, old epoch tail, second disconnect at each finalization point, local/global terminal failure, retry fact missing next action, exhausted inactive wait, or second evaluator. | Exact transitions/five-bin accounting/fence/currentness and equivalence to uninterrupted control when evidence matches; failed global claim suppresses. Does not select Component 8 retry numbers/backoff/readiness thresholds. | Phase 1 recovery scenarios 10–12; V2 control/supersession/final-disconnect tests. | `C6-S4` |
| `C6-INTEGRATION-01` | `P-C6-INTEGRATION`: Components 1–6 offline fake socket/HTTP aggregate-lifecycle trace and ownership inspection | One binding, C5 ack/tail/fence, C6 REST result, Component 2 canonical merge, Component 3 evaluator, and one publication compose. Counterexamples: REST second mapper, adapter/worker lifecycle mutation, historical overwrite, fence bypass, separate recovery ranking, or terminal empty freezing timers. | Exact canonical rows/categories/transitions and one owner/mapper/`T`/evaluator/publication path; all C6 proof IDs are reachable. Does not prove live Massive behavior, actual latency/capacity, checkpoint implementation, readiness, or cutover. | Components 1–5 accepted interfaces; V2 fake flows adapted offline. | `C6-S4` |

### 15.1 Construction guarantees reviewed at acceptance

- Only the engine constructs generations/tokens, closes terminal bins, derives
  coverage/no-print, changes lifecycle/`T`, invokes evaluation, or publishes.
- Worker result/fact types are immutable copies with closed states and no
  engine pointer/callback.
- Only one Massive second-aggregate page decoder and one row mapper serve C4
  and C6; no V2 mapper survives.
- Plans receive the valid-prior population from the installed immutable
  binding and have no rank/top-20/feature input.
- Historical chunks can reach canonical state only through Component 2's FIFO
  and existing aggregate decision.
- The ingress marker uses C5's one raw FIFO; C6 owns no raw/live queue or frame
  counter.
- At most one active generation and one terminal bin per request are
  representable; no `deferred` enum or recovery watermark/evaluator exists.

### 15.2 Required independent reviews

- `C6-S1`: narrow `gpt-5.6-sol` medium review of external REST trust, secret
  containment, sole-mapper/client reuse, sealed result/chunk terminality, and
  worker cancellation/bounds.
- `C6-S2`: narrow `gpt-5.6-sol` medium review of sole engine ownership,
  generation/token identity, terminal accounting linearization, caller-alias
  containment, and historical merge integration.
- `C6-S3`: narrow `gpt-5.6-sol` medium review of C5 marker/C2 FIFO ordering,
  lifecycle exit, no-print false-success prevention, and epoch-loss fencing.
- `C6-S4`: narrow `gpt-5.6-sol` medium review of exact gap semantics,
  retry/supersession/exhaustion progression, one evaluator/watermark, and the
  Components 1–6 integration boundary.
- After all accepted slices, the mandatory separate final read-only Component
  6 review covers the complete manifest. Reuse the same reviewer for focused
  corrections where practical.

## 16. Sequential implementation-slice plan

Normally only one repository implementation slice may be active. The owner
explicitly overrides that rule and component order for `C6-S1` only: one active
C5 slice and `C6-S1` may proceed concurrently because S1 has no C5 dependency.
At most one C6 slice may be active, and `C6-S2`–`C6-S4` remain unauthorized
before Component 5 final review under the current exception.

| Slice | Coherent outcome | Requirements/proofs | Dependencies | Allowed ownership/files | Approved V2 whitelist | Verification and acceptance record | Explicitly deferred |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `C6-S1` | One strict shared Massive aggregate REST acquisition core serves the accepted offline downloader and a new bounded production worker that returns sealed immutable chunks/terminal facts. | `C6-REST-01`/`P-C6-REST`; `C6-WORKER-01`/`P-C6-WORKER`; `C6-BOUND-01`/`P-C6-BOUND` | Accepted Component 1 binding accessors and Component 4 mapper/downloader/tests. No C5 dependency. | `internal/massive` REST acquisition/downloader/hydration worker values and focused tests; minimal `internal/replayartifact` test adaptation only if the private refactor requires it. No engine/lifecycle change or third-party dependency. | Behavior-only exact `recovery_rest.go`, `recovery_rest_test.go`, and `recovery.go` entries in the REST detail; no direct production port. | Run all three proofs; existing C4 mapper/downloader/compiler/artifact proofs; affected packages; repository build/test/vet; race for REST worker tests; formatting; dependency/secret/sole-mapper inspection; required external-trust review. Record unchanged C4 behavior and worker success/failure/cancel walkthrough. | Engine generation/ledger, historical admissions, fences, lifecycle, no-print, gap recovery, production policy/executable. |
| `C6-S2` | Component 2's one FIFO/engine owns complete-population hydration plans, immutable generations/tokens, chunk/terminal consumption, exact work accounting, and production historical merge consequences. | `C6-PLAN-01`/`P-C6-PLAN`; `C6-LEDGER-01`/`P-C6-LEDGER`; `C6-MERGE-01`/`P-C6-MERGE` | Accepted S1 facts, Components 1–3, accepted Component 2 extension rule and aggregate merge. | `internal/engine` C6 typed inputs/private state/plan/ledger/coverage consequence and focused tests; minimal `internal/massive` conversion glue. No raw-frame/fence adapter or lifecycle completion yet. | Behavior-only exact V2 binding/planning/merge/accounting sources/tests in the engine detail; no production port. | Run all three proofs; every earlier Component 2/3 proof affected by new state/input/merge; accepted C6-S1 and C4 proofs; repository build/test/vet; race for FIFO/ledger; formatting; ownership/bounds/whitelist inspection; required sole-owner review. Record invalid plan, chunk/terminal, stale/cancel, and merge walkthrough. | C5 fence command/marker, hydration/recovery transitions, no-print final proof, policy retry, integration executable. |
| `C6-S3` | A real C5 raw-FIFO marker and Component 2 lifecycle extension complete fresh/future-checkpoint hydration through causal fencing, honest no-print/unknown consequences, and mandatory exit to ordinary operation or explicit suppression/termination. | `C6-FENCE-01`/`P-C6-FENCE`; `C6-START-01`/`P-C6-START`; `C6-NOPRINT-01`/`P-C6-NOPRINT` | Accepted S1/S2; finally accepted Component 5 implementation; Components 1–3. | `internal/massive` only the C6 capture-marker extension on the accepted C5 queue/command seam; `internal/engine` fence/startup/no-print/lifecycle logic and focused tests. No change to existing C5 wire protocol meanings. | Behavior-only named V2 fence/fresh/hydration/remediation tests; no owner/recovery code port. | Run all three proofs; all affected C2/C3/C5 and C6-S1/S2 proofs; repository build/test/vet; race for adapter/engine marker interleavings; formatting; no-deferred/one-owner/source inspection; required fence/lifecycle review. Record fence linearization, empty/failure consequence, epoch-loss, and exit walkthrough. | Checkpoint validation/storage, same-process retry policy, full gap recovery, readiness, production orchestration. |
| `C6-S4` | Same-process aggregate loss recovers exact `[T_supported,R)` coverage through new epochs, policy facts, terminal work and one fence, then composes the offline Components 1–6 production aggregate lifecycle with one ordinary evaluator. | `C6-RECOVER-01`/`P-C6-RECOVER`; `C6-INTEGRATION-01`/`P-C6-INTEGRATION` | Accepted S1–S3 and Components 1–5. Test-only injected retry/exhaustion actions. | `internal/engine` recovery transitions/policy-fact consumption and `internal/massive`/integration composition tests; exact accepted seams only. No production Component 8 policy or scanner executable. | Behavior-only named V2 control/supersession/final-disconnect/fake-flow tests; no production source port. | Run both proofs and the complete eleven-proof C6 ledger; affected C2–C5 proofs; repository build/test/vet; race for adapter/worker/engine packages; formatting; complete conformance/whitelist/ownership/no-deferred/drift inspection; required recovery/interface review. Clean delegated gate would proceed to mandatory final component review. | Component 7 checkpoints, Component 8 deployed budgets/retries/readiness/shutdown/runtime composition, C9 T/Q, API/UI, live observation/cutover. |

### 16.1 Primary proof ownership is unique

Every Component 6 requirement appears exactly once in Section 15 and exactly
one slice above. Re-running earlier proofs in later slices is regression
verification, not a second primary proof. The integration proof demonstrates a
distinct cross-component composition boundary rather than duplicating each
unit claim.

### 16.2 Advancement mode

Approved advancement mode is `delegated` for `C6-S1`–`C6-S4` and final
component review. The owner may revoke delegation prospectively or mark a later
gate manual. The 2026-08-06 early-implementation exception authorizes `C6-S1`
now and pre-authorizes its implementer to write the compact acceptance record
and mark only the parent-ledger S1 row `accepted` when all three S1 proofs,
allocated verification, narrow external-trust review, conformance walkthrough,
and drift audit pass with no deviation or ambiguous concurrent-C5 failure. The
implementer then stops; delegated advancement does not start `C6-S2` until C5
passes final review or the owner grants another exact exception.

### 16.3 Parallel-work isolation

`C6-S1` remains confined to the REST acquisition/downloader/worker boundary
allocated in its slice row and minimal focused test adaptation. It may share an
`internal/massive` package build with C5, but it may not edit C5-owned live
normalization, connection, command, raw-frame, classifier, engine, or lifecycle
files. Use an isolated worktree or otherwise guarantee that no file is edited
by both efforts. Acceptance verification must run against one identified,
stable source snapshot. Any overlapping file, need to understand unfinished C5
code, moving C5 interface, or failure whose ownership cannot be established is
an escalation condition rather than a waived failure.

## 17. Implementation discretion, prohibitions, and escalation

Implementers may choose private Go type names, chunk representation, compact
ledger encoding, goroutine/channel arrangement within the approved bounds,
and whether optional provider page-progress facts enter the engine or remain
worker diagnostics. They may mechanically refactor Component 4's accepted
private downloader code into a two-consumer acquisition core if every earlier
proof remains unchanged.

The following are fixed and not delegated:

- one Component 1 binding, Component 2 FIFO/engine/canonical state/`T`,
  Component 3 evaluator, Component 4 mapper, and Component 5 raw FIFO;
- complete valid-prior population planning independent of rank/features;
- exact mode intervals, including `R=S` empty work and same-process
  `[T_supported,R)` with no V2 16-minute backward overlap;
- sealed complete provider result before row chunks, followed by one terminal
  fact; no partial-page canonical exposure;
- exactly one engine terminal bin per request and exact five-bin accounting;
- historical fill-only merge and provider-terminal versus canonical-coverage
  separation;
- real C5 raw-FIFO capture marker, never queue-empty/polling evidence;
- no `deferred`, second recovery evaluator/watermark/publication path, or
  adapter/worker lifecycle/currentness decision; and
- no production retry/deadline/readiness/shutdown defaults before Component 8.

No V2 production code is a direct-copy source. No version 1 source, live
provider call, credential, checkpoint implementation, readiness/health owner,
T/Q feature, API/UI, database, journal, service, generic event bus, plugin
system, or generalized worker/HTTP framework is allowed.

Stop for the smallest owner decision if:

- Component 5 final review changes the approved acknowledgement, epoch/loss,
  raw-FIFO marker feasibility, or causal-position meaning needed here;
- the C5 capture-marker extension requires changing an approved C5 product or
  transport behavior rather than adding the narrow C6 command/fact;
- the existing C4 client cannot be shared without changing an accepted request,
  trust, persistence, or proof premise;
- a plan needs rank/feature-dependent omission, an interval before
  `T_supported`, a compacted-present overwrite, another state owner/evaluator,
  or a second REST mapper;
- chunk/terminal admission cannot guarantee one disposition without unbounded
  buffering or engine blocking;
- a failed/local result cannot be contained without changing product behavior;
  or
- any primary proof/review is failed or ambiguous, whitelist scope expands, a
  contract/document-map responsibility changes, or a drift-audit answer becomes
  `yes`.

## 18. Completed-contract acceptance checklist

- [x] The parent enumerates exact Phase 1 IDs, one ownership boundary, explicit
      non-scope, stable-interface exception, and one authoritative delivery
      ledger.
- [x] The modular map routes external REST trust, engine reconciliation, and
      proof/delivery concerns exactly once with acyclic dependencies.
- [x] Reconnaissance records exact dirty V2 byte hashes, named declaration/test
      roles, decisions, adaptations, proof use, and limitations.
- [x] V2 production recovery/orchestration, duplicate mapper/binding,
      rank-prioritized planner, and `deferred` state are explicitly rejected.
- [x] The Component 4 shared-client/sole-mapper obligation and C5 capture-marker
      extension are concrete and reviewable.
- [x] All eleven Component 6 requirements have one primary proof and one slice;
      every proof names its dangerous counterexample, observable result, and
      limitation.
- [x] Trust boundaries distinguish provider completion, engine work terminal,
      canonical row disposition, coverage consequence, fence completion,
      no-print, and lifecycle exit.
- [x] Work, workers, pages, attempts, bodies, records, chunks, plans, ledger,
      generations, fences, diagnostics, and retained state are bounded or
      explicitly routed to Component 8 deployed policy.
- [x] The four proposed slices are sequential coherent outcomes with exact
      scope, dependencies, proof, verification, review, and deferred behavior.
- [x] Checkpoints/readiness/TQ/API/UI/live-provider/cutover behavior remains
      with later components.
- [x] Owner has approved Sections 8–19, exact reuse/fixture whitelist, eleven
      proofs, four-slice plan, review triggers, and advancement mode.
- [x] Owner has authorized `C6-S1` implementation and clean delegated
      acceptance concurrently with Component 5, overriding component order and
      the one-active-slice rule only for this pair.
- [ ] Component 5 has passed final review, which remains required before
      `C6-S2`–`C6-S4` under the current exception.

**Completed-contract owner decision:** Approved 2026-08-06. This approval fixes
the behavior-only whitelist, eleven proofs, four slices, required reviews,
delegated advancement, and C5 capture-marker extension. The separate
2026-08-06 exception authorizes early `C6-S1` implementation and clean
acceptance concurrently with C5, but no later C6 slice.

## 19. Drift audit

| Question | Answer | Evidence |
| --- | --- | --- |
| Did this add a product rule? | No | It implements Phase 1 historical work, no-print, fence, and lifecycle meanings. Planning only valid-prior symbols follows their permanent unrankable binding category; it does not change ranking. |
| Did this add a competing mutable owner, state graph, watermark, evaluator, or publication path? | No | One engine owns plan/ledger/consequence/lifecycle; V2 owner/evaluator is rejected. Workers return facts only. |
| Did this duplicate provider normalization or canonical merge? | No | The C4 mapper/client core and C2 historical merge are mandatory single paths; V2 duplicates are rejected. |
| Did this change a time-window meaning? | No | Fresh/checkpoint intervals use Phase 1 exact formulas. Same-process starts at retained supported `T`, the conservative unsupported join; V2's unnecessary `T-H` overlap is rejected. |
| Did this silently change Component 5? | No | The approved new capture command/marker is explicit under Component 6's roadmap-owned ingress-fencing responsibility and still requires finally accepted C5 implementation before coding absent a separate implementation exception. Existing C5 command/fact meanings are unchanged. |
| Did this preempt checkpoint or operations policy? | No | C7 owns checkpoint evidence; C8 supplies deployed budgets/retry/exhaustion/deadlines/readiness/shutdown. C6 owns only hard safety bounds and policy-fact execution. |
| Did this make T/Q affect aggregate ranking/readiness? | No | Recovery closes T/Q as Phase 1 requires; T/Q supplies no plan, terminal, fence, no-print, or evaluator input. |
| Did this add behavior without evidence? | No | Every edge is Phase 1/accepted dependency behavior or named scoped V2 regression evidence. No provider-current/live claim is made. |
| Did reconnaissance exceed the approved scope? | No | Reads were limited to exact REST/recovery/binding/fence/fresh/checkpoint-interface declaration and test roles. Checkpoint-only declarations were name-discovered but their contents were not used; only the two approved fresh flows were read. No live/network/credential/version 1 area was accessed. |
| Is implementation authorized? | `C6-S1` only | The owner explicitly authorized S1 concurrently with C5 and pre-authorized clean delegated acceptance. S2–S4 still wait for C5 final review or another exact exception. |
