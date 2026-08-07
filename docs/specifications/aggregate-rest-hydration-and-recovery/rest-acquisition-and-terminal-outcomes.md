# Aggregate REST hydration and recovery — REST acquisition and terminal outcomes

**Parent contract:** [Aggregate REST hydration and recovery](../aggregate-rest-hydration-and-recovery.md)

**Normative responsibility:** Production Massive aggregate REST request and
response trust, reuse of the single Component 4 row mapper, bounded worker
execution, sealed result chunking, and immutable provider terminal facts.

**Controlling requirements:** `ARCH-OWN-02`, `ARCH-FLOW-01`–`ARCH-FLOW-04`,
`DTE-MODEL-01`–`DTE-MODEL-03`, `DTE-WINDOW-01`, `DTE-WINDOW-02`,
`DTE-EVENT-03`, `DTE-AGG-01`–`DTE-AGG-04`, `DTE-HYDRATE-01`,
`DTE-HYDRATE-02`, `DTE-RECOVERY-01`, `DTE-REJECT-01`; `C6-REST-01`,
`C6-WORKER-01`, and `C6-BOUND-01`

**Allocated slices:** Approved and owner-authorized early `C6-S1`

**Document dependencies:** [Parent](../aggregate-rest-hydration-and-recovery.md),
Component 1 [binding contract](../reference-data-and-session-binding.md), and
Component 4 [REST normalizer/artifact trust](../aggregate-replay/rest-normalization-and-artifact.md)

**Approval state:** Owner-approved 2026-08-06 as part of the complete modular
Component 6 contract. The owner separately authorized `C6-S1` to run
concurrently with Component 5 under the exact parent exception; this detail is
not an independent authority.

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-rest-hydration-and-recovery.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance used V2 commit `5f92a151dd850002578a33a81ad90dea096c63b6`
with unrelated worktree changes. Every inspected file below was modified or
untracked except where the V2 status says otherwise, so the SHA-256 identifies
the exact bytes inspected and does not attribute dirty behavior to the commit.
No credential value, network call, live test, version 1 source, or unrelated
provider endpoint was opened.

| Exact V2 source | File SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `internal/massive/recovery_rest.go`: `RESTRecoveryFetcher`, `Stream`, `validateRequest`, `fetchSymbol`, `fetchPage`, `reserveGenerationWire`, and `validRetryAfter` | `83eb8fd07c3d776a2a9ce79bc135bad81095b93d60feb0abe8d08742074b5264` | Fixed unadjusted ascending second-aggregate request, same-origin pagination, redirect refusal, 8 workers, 2 pages, 3 attempts, 16 MiB/page, 50,000 rows/page, per-generation bytes, exact interval/duplicate/order checks, cancellation, and per-symbol progress/terminal streaming are useful. The fetcher buffers one complete symbol before returning it. | `behavior evidence` and `adapt` | Use Component 4's stronger existing HTTPS configuration, strict envelope identity/status/count decoder, exact endpoint-path continuation, end-minus-one-millisecond request, credential source, mapper, and reasons. Remove API key fields, random jitter, generation/lifecycle policy, duplicate provider structs, string-matched errors, batch IDs, and whole-batch fallback. | `P-C6-REST`, `P-C6-WORKER`, `P-C6-BOUND` |
| Same file: `normalizeRecoveryBar`, `recoveryRESTBar`, `exactJSONInt64`, and `finiteJSONFloat` | same | Duplicates the provider row mapping that Component 4 has already replaced with stricter exact JSON, structural validation, checked ATS, and provenance. | `reject` | Every row must call the accepted `NormalizeRESTSecondAggregate`; no second field map or ATS path may compile. | `P-C6-REST` construction inspection |
| `internal/massive/recovery_rest_test.go`: `TestRESTRecoveryContractSparseNormalizationAndBounds`, `TestRESTRecoveryPaginationRetriesRedirectsAndRedaction`, `TestRESTRecoveryRejectsMalformedOversizedAndSlowResponses`, `TestRESTRecoveryConcurrencyAndSymbolBounds`, and `TestPhase2ARESTStreamingRetryAfterAndGenerationBounds` | `b868d6970feddf9179aac953bebbd22f0745a5c777f5e113c6569d8cc31360ce` | Focused fake HTTP cases prove request shape, sparse seconds, two-page/retry/redirect/body/deadline/concurrency bounds, progress facts, decoded-row limit, and Retry-After parsing. | `behavior evidence` | Adapt only the named cases. V2 fixtures omit required `status`, `ticker`, and `adjusted` envelope evidence and infer empty/value from a weak response; repository tests must use Component 4's strict envelope and closed terminal facts. Numeric generation durations are not adopted. | All three S1 proofs |
| `internal/massive/recovery.go`: `RecoveryRequest`, `RecoveryInterval`, `RecoverySymbolOutcome`, `RecoverySymbolResult`, and `RecoveryStreamer` | `0a65213c5c9ed982bdc8863194a70068c7b2ad564c3970b6ade46b8253e683f9` | Immutable binding/generation/epoch/symbol/interval evidence and per-symbol streaming are useful shapes. | `behavior evidence` | Replace mutable slices/maps and pointer budget with copied immutable plan values. Replace `completed` with distinct provider `completed_value`/`completed_empty`; add `fenced` only as an engine disposition, exact request/result/chunk ordinals and counts, and eliminate `Fetch` whole-batch compatibility. | `P-C6-WORKER`, `P-C6-BOUND` |
| Same file: `CheckpointBoundary`, `ObserveCheckpointBoundary`, and checkpoint outcome vocabulary | same | Couples checkpoint discovery to the recovery adapter. | `reject` | Component 7 owns checkpoint load/install. C6 accepts only an engine-owned future `T0` in a checkpoint-catch-up plan. | None |
| `internal/massive/config.go`: recovery-related values reached from `Config`/`Validate` | `5472ad07aea7175ba1d445b84be2906436364719327263641c22d2aac07a1eaa` | Confirms predecessor numeric defaults but mixes credentials, URLs, queues, readiness, checkpoint, and API configuration. | `reject` | Reuse Component 4 hard REST ceilings and injected per-plan budgets. Component 8 later owns deployed concurrency/capacity/deadline policy. | `P-C6-BOUND` |

The accepted Component 4 implementation is stronger and nearer than V2. Its
`internal/massive/offline_downloader.go` already proves strict envelope status,
symbol, adjustment and count evidence; HTTPS same-origin exact-path
continuations; redirects and credential containment; bounded retries, bodies,
pages, workers, records, and bytes; successful empty; ordering/duplicate
rejection; and the sole `NormalizeRESTSecondAggregate` call. Component 6 should
extract the smallest private per-symbol aggregate REST acquisition core used by
both the offline compiler wrapper and the production hydrator wrapper. This is
an adaptation of accepted repository code, not V2 reuse and not a generalized
HTTP framework.

**Approved V2 implementation whitelist:** No V2 production code is approved for
direct port. Only the exact declaration roles and named focused test cases in
the table above are approved as behavior evidence. No V2 JSON fixture is
needed; Component 4's repository-owned strict fake responses and mapper fixture
are the starting evidence. Production builds/tests may not depend on the V2
checkout.

## 9. Detailed semantic inputs, outputs, and owned state

| Item | Meaning and required provenance/identity | Bound or ownership |
| --- | --- | --- |
| Immutable hydration work item | Exact Component 1 binding identity, positive generation and request ID, purpose, canonical symbol, one exact `[a,b)` interval, expected provider identity, and positive plan budgets. Connection epoch is retained as historical-generation context where the plan depends on a live handoff; it is never provider request identity. | Produced only by the engine. One request ID is never reused in a run. Symbol and interval are copied, not caller-owned aliases. |
| Massive request | `GET /v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}` with `from_ms=a`, `to_ms_inclusive=b-1ms`, `adjusted=false`, `sort=asc`, and `limit=50000`. Bearer credential comes from a private credential source. | One symbol and interval within `[S,E)`. At most two pages and three attempts/page. HTTPS configured origin only. |
| Provider page | HTTP 200, bounded single JSON object, exact `status=OK`, exact ticker, explicit `adjusted=false`, results array/null/absent under the Component 4 rule, consistent present counts, and optional validated continuation. | Maximum 50,000 results and 16 MiB per page. Unknown additive fields may be ignored only after strict contract-member handling. |
| Normalized result | One fully acquired symbol interval whose every row passed `NormalizeRESTSecondAggregate`, symbol/interval/order/duplicate validation, and aggregate/result budgets. It is either nonempty or empty; partial pages/rows are not exposed as a complete result. | Worker-local and immutable once sealed. At most one row per requested second; maximum 57,600 rows. At most the configured worker count of sealed/in-progress symbol buffers. |
| Result chunk fact | Binding/generation/request/symbol/interval, positive result ID, zero-based contiguous chunk ordinal, positive total chunks for nonempty results, row offset, exact total row count, and a copied bounded slice of normalized values. | Fixed positive `rows_per_chunk` construction value; one chunk occupies one bounded engine input. Chunks emit only after the provider result is fully sealed. |
| Provider terminal fact | Same exact identity plus `completed_value`, `completed_empty`, `failed`, or `canceled`; pages, attempts, bytes, normalized row count, emitted chunk/row counts, and one bounded reason. | Exactly one emitted terminal fact per accepted work item. `fenced` is not guessed by a worker; the engine assigns it to stale/currently inapplicable work. |
| Worker pool | Bounded jobs, active HTTP attempts, per-worker result buffer, result-to-engine admission, cancellation, and joined terminal state. | `1..8` workers; no detached goroutine. Engine FIFO backpressure may block workers cancellation-aware; it never blocks the engine. |

The REST layer owns transient request/response/buffer state only. It owns no
active generation, work terminality after engine validation, historical
coverage, no-print state, lifecycle, `T`, retry of a whole generation,
evaluation, readiness, or publication.

**Construction guarantees:** Provider wire fields have one mapper; a result
cannot emit rows before its pages/envelope/normalization/order are complete;
worker facts are copied immutable values with closed enums; one worker owns one
work item until its terminal fact; credentials are absent from every fact and
reason; and worker count/chunk size/request/page/attempt/body/record budgets are
validated before I/O.

**Runtime validation still required:** Binding/generation/request applicability
belongs to the engine. The REST layer must still reject bad endpoint identity,
redirect/continuation, envelope/status/symbol/adjustment/count, duplicate or
out-of-order/out-of-range row, budget exhaustion, malformed data, cancellation,
and a result/chunk admission failure before reporting provider completion.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C6-REST-01` | Extract one private Massive second-aggregate acquisition core from the accepted Component 4 path. Both the existing offline downloader and the C6 production worker must call that core, and every result row must call the one accepted `NormalizeRESTSecondAggregate`. Preserve the fixed unadjusted request, strict envelope/status/symbol/adjustment/count evidence, same-origin exact-path continuation, retry/body/page/order/duplicate/budget rules, and successful-empty distinction. No second REST page decoder or row mapper may compile. | `DTE-MODEL-01`–`03`, `DTE-AGG-01`–`04`, `DTE-HYDRATE-02`; accepted `C4-NORM-01`, `C4-COMP-NORM-01`, `C4-DL-01`; V2 focused REST evidence. |
| `C6-WORKER-01` | For each immutable work item, acquire and seal the whole provider result before exposing normalized rows. On success, emit contiguous bounded chunks followed by exactly one `completed_value` or `completed_empty` fact whose counts reconcile. On provider/decode/budget failure emit no result rows and exactly one `failed`; on cancellation emit no new chunks after cancellation and exactly one `canceled` when the engine input remains open. Worker completion order has no semantic authority. | `DTE-EVENT-03`, `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, `ARCH-OWN-02`, `ARCH-FLOW-02`; V2 streamer evidence plus Phase 1 terminality. |
| `C6-BOUND-01` | Validate finite plan-wide response/normalized-row/resident-result budgets, `1..8` workers, positive bounded chunk size, two pages, three attempts/page, 16 MiB/page, and at most one row/requested second. Stop new scheduling on cancellation, join all workers, and make every accepted work item capable of reaching one engine-consumed terminal disposition without unbounded buffering or dropped accepted facts. | `ARCH-FLOW-01`–`04`, `DTE-REJECT-01`, `LIFE-END-02`; Component 4 bounds and V2 concurrency/cancellation evidence. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Hydration work item | Exact installed binding, positive generation/request, canonical symbol, one-second-aligned nonempty `[a,b)` inside `[S,E)`, and positive checked budgets. | Return a bounded construction failure before I/O; engine retains/terminalizes the registered work. | Wrong-binding or out-of-session request returning plausible bars into a current generation. |
| Massive response | Component 4's strict HTTP/envelope/identity/adjustment/count/continuation evidence and every row normalizes in strict ascending unique interval order. | Fail the complete symbol request; discard its buffered rows. | HTTP 200 error, foreign ticker, adjusted page, truncated pagination, or invalid late row being reported empty/value complete. |
| Sealed result/chunks | Full provider result complete; exact immutable identity; contiguous chunks/counts; all rows from the shared mapper. | Incomplete emission never yields a completion fact; cancellation or admission closure produces canceled/fenced cleanup evidence. | Rows from a later page failing after earlier rows already looked complete. |
| Credential | Nonempty CR/LF-free private value used only in `Authorization`; absent from URL, facts, errors, logs, and labels. | Fixed request-construction/provider-unavailable reason. | Redirect or provider prose leaking the bearer value. |

## 11. Failure and terminal behavior

Transport/read errors, HTTP 429/5xx, and request timeout receive at most the
fixed Component 4 attempt count. A valid `Retry-After` may delay within the
15-second request ceiling and caller context. Redirects, origin/path changes,
permanent HTTP/provider status, malformed/trailing/ambiguous JSON, wrong
ticker/adjustment/count, pagination cycle/third page, oversized response,
invalid/out-of-range/duplicate/nonascending row, mapper rejection, record or
resident budget exhaustion, and credential/configuration failure terminate the
work item as `failed` without exposing buffered rows.

An otherwise strict successful response with zero rows and terminal pagination
emits `completed_empty`. Missing/null/empty results under Component 4's accepted
envelope rule are not inferred from HTTP success alone. A nonempty sealed
result emits all chunks before `completed_value`. Provider success does not
claim canonical acceptance, coverage, no-print, currentness, or lifecycle exit.

Cancellation stops new jobs, cancels I/O and blocked admissions, discards
unemitted buffers, joins workers, and returns one cancellation fact for every
registered work item that the engine can still consume. If the engine FIFO has
closed, process cleanup reports the unadmitted terminal facts to the engine's
already-active shutdown/fence disposition; workers do not wait indefinitely.

## 12. Accounting and observability

Worker/provider outcomes are a producer-side diagnostic partition:

```text
worker_items_started
  = provider_completed_value
  + provider_completed_empty
  + provider_failed
  + provider_canceled
```

The authoritative Phase 1 `planned = completed_value + completed_empty +
failed + canceled + fenced` identity is engine-owned and appears in the next
detail. Producer counts do not substitute for that ledger.

For each successful nonempty item:

```text
normalized_rows = sum(chunk_rows) = terminal_row_count
emitted_chunks   = terminal_chunk_count
```

Pages, attempts, response bytes, normalized rows, active workers, blocked
admissions, and fixed reasons are overlapping diagnostics. Symbols, URLs,
payloads, credentials, and arbitrary provider text are not retained labels or
logs.

## 13. Simplicity and boundedness

- Mutable state is one private shared aggregate REST client plus bounded
  work-local page/result buffers and a finite worker pool. The client owns no
  scanner truth.
- The existing Component 4 downloader becomes one wrapper over the shared
  per-symbol acquisition core; Component 6 adds a second wrapper producing
  historical chunks/terminal facts. This two-consumer extraction is smaller
  than copying the HTTP/envelope path and preserves earlier proofs.
- V2's mapper, public mutable fetcher fields, `Fetch` compatibility path,
  batch ID, pointer budget, random jitter, string error classification,
  checkpoint seam, and generation deadlines are rejected.
- Page/attempt/body/worker ceilings remain the accepted Component 4 hard safety
  bounds. Plan-wide byte/row/resident budgets and chunk size are explicit
  construction inputs; Component 8 later selects deployed values and policies.
- There is no generic provider client, job framework, database, service,
  resumable partial result, raw response archive, or speculative edge-case
  mechanism.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Sparse successful interval | Component 4 fixture and V2 sparse test | Return only real rows; do not synthesize absent seconds; nonempty terminal count is exact. | `P-C6-REST` |
| Strict successful empty | Component 4 trust contract; V2 empty pagination shapes | Require complete strict envelope/pagination, emit no chunks then `completed_empty`. | `P-C6-WORKER` |
| Invalid last row or second page | External-trust invariant; V2 malformed/pagination tests | Discard the full buffered symbol result; emit `failed`, never partial value success. | `P-C6-REST` |
| Foreign/cyclic/third continuation or redirect | Component 4 and V2 fake HTTP tests | Reject without forwarding credentials or claiming terminal coverage. | `P-C6-REST` |
| 429/5xx/slow/oversized response | Component 4 and V2 bounds | Apply fixed bounded attempts/deadline/body limit; fail after exhaustion. | `P-C6-BOUND` |
| More than one page and result chunk | V2 progress/streaming evidence; bounded-input invariant | Provider page progress stays diagnostic; sealed normalized chunks are contiguous and terminal counts reconcile. | `P-C6-WORKER` |
| Worker completion permutation | `ARCH-FLOW-03`; V2 concurrency test | Request IDs and engine FIFO order, never goroutine order, determine state. | `P-C6-BOUND` |
| Cancellation while I/O, buffering, chunk admission, or terminal admission is active | V2 cancellation/deadline cases; Phase 1 terminality | Join workers; no new chunks after cancel; every registered item receives one engine disposition or explicit shutdown fence. | `P-C6-BOUND` |

Primary-proof allocation and slice acceptance are authoritative only in the
[proof and delivery plan](proof-and-delivery-plan.md).
