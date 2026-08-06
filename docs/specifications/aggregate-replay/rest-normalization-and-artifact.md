# Aggregate replay — REST normalization and artifact trust

**Parent contract:** [Aggregate replay](../aggregate-replay.md)

**Normative responsibility:** The one Massive REST second-aggregate row
normalizer shared with Component 6, the offline downloader/compiler, and the
versioned replay-artifact persistence and validation boundary.

**Controlling requirements:** `PG-REPLAY-01`, `ARCH-FLOW-04`,
`DTE-MODEL-01`–`DTE-MODEL-03`, `DTE-SESSION-01`–`DTE-SESSION-04`,
`DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04`, `DTE-EVENT-01`,
`DTE-EVENT-04`, `DTE-AGG-01`–`DTE-AGG-04`, `DTE-REPLAY-01`,
`DTE-REPLAY-02`, `DTE-REJECT-01`, `LIFE-REPLAY-01`; `C4-NORM-01`,
`C4-COMP-NORM-01`, `C4-DL-01`, `C4-ART-01`, and `C4-ART-02`

**Allocated slices:** `C4-S1`, `C4-S2`

**Document dependencies:** [Parent](../aggregate-replay.md), Component 1
[binding contract](../reference-data-and-session-binding.md), and Component 2
[canonical aggregate contract](../scanner-state-engine-and-canonical-state.md)

**Approval state:** Approved 2026-08-06 as part of the complete parent contract;
this detail has no independent approval state

**Delivery state:** See the authoritative
[parent delivery-state ledger](../aggregate-replay.md#authoritative-delivery-state-ledger);
do not copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance was performed against version 2 commit
`5f92a151dd850002578a33a81ad90dea096c63b6`. The checkout had unrelated local
changes: both inspected REST source/test files were worktree-modified, while
the JSON fixture matched the commit. Hashes below therefore identify the exact
bytes inspected rather than attributing modified behavior to the commit. No credential,
live request, version 1 checkout, WebSocket path, recovery planner, or broad
integration suite was opened.

| Exact source | File SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `internal/massive/recovery_rest.go`: REST constants, page fetch/decode, and `normalizeRecoveryBar` only | `83eb8fd07c3d776a2a9ce79bc135bad81095b93d60feb0abe8d08742074b5264` | Uses `/v2/aggs/ticker/{symbol}/range/1/second/{from}/{to}`, `adjusted=false`, ascending order, limit 50,000, bearer auth, same-origin pagination, redirect refusal, two pages, 16 MiB/page, 15-second attempts, and two retries for 429/5xx. It parses `t,o,h,l,c,v,vw,n`; requires exact whole-second millisecond `t`, finite numerics, positive integral `n`, and computes `floor(v/n)`. | Behavior evidence and adapt | Remove recovery generation/token/planner ownership. Add strict envelope, canonical OHLC/VWAP/volume validity, explicit ATS provenance, binding/request checks outside the stateless mapper, and deterministic compiler output. Do not port the recovery client as Component 4 ownership. | `P-C4-NORM`; `P-C4-DL` |
| `internal/massive/recovery_rest_test.go`: `TestRESTRecoveryContractSparseNormalizationAndBounds`, pagination/retry/redaction test, and malformed/oversized/slow-response test only | `b868d6970feddf9179aac953bebbd22f0745a5c777f5e113c6569d8cc31360ce` | Proves the endpoint/query/auth shape, sparse seconds without fabricated bars, fractional volume, floor ATS, same-origin continuation, bounded retry/page/body/deadline behavior, redirect credential containment, and rejection of zero `n`, fractional timestamps, trailing JSON, oversized, and slow responses. | Behavior evidence and adapt | Keep only downloader/provider claims. Recovery concurrency, generation, work registration, and later tests remain excluded. Strengthen the false-success checks around response identity and artifact publication. | `P-C4-NORM`; `P-C4-DL`; `P-C4-ART-TRUST` |
| `internal/massive/testdata/rest-second-bars.json` | `be30ea406fd7fe5c50642a1851b5ebc63cfc98d149caac995e1c91412a1df7c5` | One unadjusted `OK` response for symbol `SYN` with three sparse rows at seconds 0, 1, and 3, fractional volumes, and every required row field. | Adapt fixture | Copy only under the approved implementation whitelist, rename as Component 4 evidence, retain provenance, and add repository-owned corrupt/envelope/coverage cases rather than treating one happy fixture as exhaustive. | `P-C4-NORM`; `P-C4-DL` |

The current official Massive custom-bars documentation, checked 2026-08-06,
confirms the endpoint parameters, millisecond `from`/`to`, unadjusted selector,
ascending sort, 50,000 maximum limit, response envelope, row fields, and that
no eligible trades means no aggregate bar. It documents `n` and `vw` as
optional response fields; because Phase 1 requires canonical VWAP and the REST
ATS mapping requires a positive transaction count, a row missing either cannot
normalize successfully. The documentation is provider evidence, not product
authority: [Massive custom bars](https://massive.com/docs/rest/stocks/aggregates/custom-bars).

The predecessor tests are good focused evidence for basic REST mechanics but
do not prove response-envelope identity, deterministic compilation, artifact
integrity/atomicity, exact per-symbol coverage, cancellation accounting, or
Component 6 reuse. There is no predecessor replay codec or compiler in the
approved scope.

**Approved implementation whitelist:** only the exact version 2 file regions
and fixture rows listed above. `recovery_rest.go` and its tests are behavior
references, not direct-copy sources; the JSON fixture may be adapted. No other
version 2 source or fixture is whitelisted.

## 9. Detailed semantic inputs, outputs, and owned state

| Item | Meaning and required provenance/identity | Bounds or ownership |
| --- | --- | --- |
| Offline download plan | One immutable Component 1 binding, exact interval `[S,R)` with `S < R <= E`, `complete_binding` scope, Massive provider identity, explicit worker/record/byte/temporary-file-count budgets, local destination, and an out-of-band bearer credential source. Symbols come only from the binding. | One plan per compile run and destination directory. `1 <= workers <= 8`; every size/count budget is positive and checked before I/O. Credential bytes never enter artifacts, URLs, errors, or diagnostic labels. |
| REST request | `GET /v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}` where `from_ms` is interval start and `to_ms_inclusive` is interval end minus one millisecond; `adjusted=false`, `sort=asc`, `limit=50000`. | One symbol and one subinterval no longer than the 16-hour bound session. At most two pages and three attempts per page. |
| REST response envelope | HTTP success plus provider status, exact requested ticker, unadjusted flag, result array, optional bounded count diagnostics, and optional continuation URL. | One JSON value, no trailing value. Continuation stays on the configured HTTPS origin and exact aggregate endpoint path; any returned `apiKey` query value is removed and bearer auth is applied out of band. Count fields never substitute for locally observed rows or terminal pagination. |
| REST row | Provider fields `t,o,h,l,c,v,vw,n`, decoded as exact JSON numbers before conversion. | Stateless input to the single mapper. Missing, duplicate recognized members, nonnumeric values, or unsupported magnitude reject the row. Additive unknown provider members may be ignored only after bounded JSON decoding. |
| Normalized REST aggregate | Canonical symbol, exact UTC `[window_start,window_end)`, `open/high/low/close`, fractional `volume`, `VWAP`, integer ATS `floor(volume/n)`, and `rest_floor_volume_over_transactions` provenance. It contains no replay/historical source position. | Immutable return value. Component 4 wraps it in artifact records; Component 6 later wraps the same value in its historical request/generation evidence. The mapper owns no binding, coverage, or engine mutation. |
| Symbol acquisition outcome | Exactly one terminal `complete`, `failed`, or `canceled` result for the requested symbol and interval. `complete` may contain zero rows. | Downloader-local. A complete artifact requires `complete` for every binding symbol; failed/canceled outcomes are never persisted as complete coverage. |
| Compiled artifact | A canonical UTF-8 JSON Lines file with schema `aggregate-replay-jsonl-v1`, one header, contiguous aggregate records, sorted per-symbol coverage entries, one summary, and one final seal. | One binding and `[S,R)`. Written through a sibling temporary file and atomically published without replacement only after validation, file sync, and seal. No database or journal. |
| Validated artifact handle | Immutable metadata plus a rewound cursor over the same already-open read-only file description after a complete first-pass schema/order/coverage/digest validation against the supplied binding. It never reopens by pathname. | Component 4 artifact reader owns the file descriptor/cursor only and retains it through replay. It cannot mutate engine or canonical state. Runtime semantics are owned by the replay detail. |
| Synthetic artifact input | Explicit test records with declared partial symbol/interval scope, logical times, and observations. It cannot use `complete_binding` or Massive acquisition provenance. | `partial_synthetic` mode only; bounded by the same file/record budgets and never eligible for complete replay commit support. |
| Downloader/compiler state | Bounded worker set, per-symbol page cursor, temporary normalized runs, deterministic merge cursor, coverage table, counters, SHA-256 state, and one destination-operation lease. | Component 4 only, lifetime of one operation. No state survives success except the sealed artifact. Ordinary failure removes operation-owned temporary files; crash remnants are bounded and prevent another operation from creating more files until explicitly cleared. |
| Compile operation outcome | Exactly one terminal `complete`, `failed`, `canceled`, or `persistence_uncertain` result. `complete` alone returns the published artifact identity/path and may create a validated handle in a separate open/validation operation. | `persistence_uncertain` means validated bytes may be visible at the final name after publication linearized but directory durability or temporary-name cleanup was not confirmed; it is never silently promoted to complete in the same operation. |

The JSON Lines representation is normative because it gives one streamable,
human-inspectable file without retaining the session in memory. Every line is
one closed JSON object. Line-kind order and exact object-member order are:

| Line kind | Exact members in canonical order | Exact constraints |
| --- | --- | --- |
| `header` | `kind`, `schema`, `artifact_mode`, `binding_id`, `universe_id`, `trading_date`, `session_start`, `session_end`, `replay_start`, `replay_end`, `provider`, `endpoint`, `normalization_policy`, `compile_format` | `kind="header"`; schema and compile format are `aggregate-replay-jsonl-v1`; mode is `complete_final_bars` or `partial_synthetic`. Complete mode requires the supplied Component 1 binding/universe identities, `provider="massive"`, the exact endpoint template, and `normalization_policy="massive-rest-second-aggregate-v1"`. Partial mode requires the supplied binding, `provider="synthetic"`, `endpoint=""`, and `normalization_policy="synthetic-declared-v1"`. All times and `[S,R)` agree with the binding and plan. |
| `aggregate` | `kind`, `ordinal`, `logical_delivery_time`, `symbol`, `window_start`, `window_end`, `open`, `high`, `low`, `close`, `volume`, `vwap`, `average_trade_size`, `ats_provenance` | Positive contiguous ordinal; exact binding symbol and one-second window; finite canonical values. Complete mode requires logical time equal to window end and REST ATS provenance. Partial mode permits a later logical time and either Phase 1 canonical ATS provenance, but makes no provider-acquisition claim. |
| `coverage` | `kind`, `symbol`, `start`, `end`, `class`, `record_count` | Sorted unique symbol; exact header replay interval; nonnegative count equal to aggregate records for the symbol. Complete mode requires `class="complete_interval"` and exactly every binding symbol. Partial mode requires `class="declared_partial"`, at least one declared symbol, and never establishes absence. |
| `summary` | `kind`, `aggregate_records`, `coverage_entries`, `empty_symbols`, `body_bytes` | Counts are exact. `body_bytes` is the byte length from the first header byte through the LF ending the final coverage line, excluding the summary. Empty-symbol count is zero in partial mode and otherwise equals complete coverage entries with zero records. |
| `seal` | `kind`, `artifact_id`, `sealed_bytes` | `artifact_id` is `sha256:<64 lowercase hexadecimal digits>` over the exact bytes from the first header byte through the LF ending the summary. `sealed_bytes` is that digest-input byte length. The seal is last and its LF is the final byte. |

No owned object permits an omitted, `null`, additional, duplicated, or
out-of-order member. All strings must be valid UTF-8; contract tokens and
identities are exact ASCII, while `symbol` is the exact Component 1 binding
symbol. JSON strings use the Go 1.26 `encoding/json` string escaping produced
with HTML escaping disabled. Timestamps use UTC `time.RFC3339Nano`, must contain
the canonical `Z` suffix, and must round-trip byte-for-byte; whole-second times
therefore contain no fractional part. Dates are exact `YYYY-MM-DD`.

Every `kind`, identity, mode, date, timestamp, provider, endpoint, policy,
class, symbol, and provenance member is a JSON string. `open`, `high`, `low`,
`close`, `volume`, and `vwap` are JSON numbers representing the validated
canonical `float64` values. `average_trade_size` is a nonnegative JSON integer
representable by `int64`. `ordinal`, every count, and both byte-length members
are nonnegative JSON integers no greater than `math.MaxInt64`; `ordinal` starts
at one. These wire bounds are checked before conversion or arithmetic, and all
count/byte arithmetic is checked against both its plan budget and
`math.MaxInt64`.

Unsigned and signed integers use base-10 ASCII with no leading zero except the
single byte `0`. Finite `float64` market values use
`strconv.AppendFloat(value, 'g', -1, 64)`. Every numeric zero, including a
negative signed zero received from JSON, is normalized to the single byte `0`
before aggregate construction and encoding. `NaN`, infinities, a leading plus,
nonminimal integer forms, and alternate parse-equivalent float/timestamp forms
are invalid. Objects have no insignificant whitespace, and every line ends in
one LF with no BOM or CR.

Artifact identity is SHA-256 over the exact canonical bytes from the first
header byte through the newline terminating the summary; the seal is excluded
to avoid a circular digest. The validator strictly decodes each owned schema,
re-encodes it under the rules above, and requires byte equality. Compile wall
time, worker count/order, local paths, provider request IDs, retry history, and
credential material are deliberately absent from canonical bytes.

The following five lines, followed by one final LF, are the normative minimal
complete-artifact golden encoding. `body_bytes=1008`, `sealed_bytes=1106`, and
the digest are part of the golden assertion:

```jsonl
{"kind":"header","schema":"aggregate-replay-jsonl-v1","artifact_mode":"complete_final_bars","binding_id":"session-binding-v1:example","universe_id":"universe-v1:example","trading_date":"2026-08-06","session_start":"2026-08-06T08:00:00Z","session_end":"2026-08-07T00:00:00Z","replay_start":"2026-08-06T08:00:00Z","replay_end":"2026-08-06T08:00:01Z","provider":"massive","endpoint":"/v2/aggs/ticker/{symbol}/range/1/second/{from_ms}/{to_ms_inclusive}","normalization_policy":"massive-rest-second-aggregate-v1","compile_format":"aggregate-replay-jsonl-v1"}
{"kind":"aggregate","ordinal":1,"logical_delivery_time":"2026-08-06T08:00:01Z","symbol":"SYN","window_start":"2026-08-06T08:00:00Z","window_end":"2026-08-06T08:00:01Z","open":10,"high":11,"low":9,"close":10.5,"volume":2.5,"vwap":10.25,"average_trade_size":1,"ats_provenance":"rest_floor_volume_over_transactions"}
{"kind":"coverage","symbol":"SYN","start":"2026-08-06T08:00:00Z","end":"2026-08-06T08:00:01Z","class":"complete_interval","record_count":1}
{"kind":"summary","aggregate_records":1,"coverage_entries":1,"empty_symbols":0,"body_bytes":1008}
{"kind":"seal","artifact_id":"sha256:31ef4b0cbaa85a0b60b3bd70acbbfa4162b9386c539ded580b4ed4897d18d901","sealed_bytes":1106}
```

For `complete_final_bars`, aggregate records are unique by
`(symbol,window_start)`, have `logical_delivery_time = window_end`, sort by
`(logical_delivery_time,window_start,symbol)`, and receive ordinals `1..N`.
Coverage contains every binding symbol exactly once, sorted by canonical
symbol, including zero-record symbols. `partial_synthetic` may contain repeated
identities and later logical delivery times, but ordinals and logical times
remain strictly nonregressing and its coverage is explicitly noncomplete.

**Construction guarantees:** provider rows cannot carry source position,
coverage, binding success, or canonical mutation authority; artifact modes are
closed; a complete artifact cannot encode a failed/canceled symbol; credentials
are not representable in the artifact schema; the final path is never visible
as success before the seal and whole-file validation pass; and record order is
derived from canonical values rather than worker completion or map iteration.

**Runtime validation still required:** HTTP/provider identity, page continuity,
numeric conversion, canonical structural validity, response and artifact byte
budgets, binding/session/symbol membership, unique final-bar identity,
coverage/record-count agreement, canonical byte encoding, digest, ordinals,
logical times, and existing-file identity all remain representable failures and
must be rejected before success.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C4-NORM-01` | The one stateless Massive REST row mapper decodes exact whole-second millisecond `t`; creates `[t,t+1s)`; requires finite positive OHLC/VWAP, finite nonnegative volume, valid OHLC order, and positive exact integral `n`; normalizes signed zero; computes ATS as checked `floor(volume/n)` with zero allowed; and returns `rest_floor_volume_over_transactions`. Missing/invalid fields produce one bounded rejection and no partial aggregate. It owns no request, binding, coverage, replay, recovery, or engine-mutation context. | `DTE-MODEL-01`–`03`, `DTE-AGG-01`–`04`, `DTE-REJECT-01`; official provider docs and scoped v2 mapper/fixture. |
| `C4-COMP-NORM-01` | The Component 4 downloader/compiler passes every provider row through the accepted `C4-NORM-01` mapper and can construct an aggregate artifact record only from that mapper's successful immutable value. No downloader, compiler, fixture adapter, or artifact codec contains a second REST field mapping or defaulting path. Component 6 remains required by the roadmap to import this same mapper, but its production-consumer proof belongs to the later Component 6 contract rather than Component 4 acceptance. | `DTE-MODEL-01`–`03`, `ARCH-OWN-02`, `DTE-REJECT-01`; accepted S1 seam and cross-component ownership invariant. |
| `C4-DL-01` | The offline downloader issues only the fixed unadjusted ascending query for each binding symbol and exact interval, follows at most two validated continuations, bounds attempts/body/time/workers, and records exactly one terminal symbol outcome. An `OK` terminal response with no rows is successful empty; HTTP/provider/pagination/decode failure is not. Any noncomplete symbol prevents final artifact publication. Worker completion order cannot affect output bytes. | `PG-REPLAY-01`, `DTE-WINDOW-04`, `DTE-REJECT-01`; scoped v2 REST tests and provider documentation. |
| `C4-ART-01` | The compiler emits exactly the closed `aggregate-replay-jsonl-v1` bytes above, assigns immutable ordinals and logical times, includes explicit normalization/provenance and per-symbol coverage, distinguishes complete final bars from partial synthetic traces, and produces byte-identical output/artifact identity for canonically equal normalized inputs regardless of downloader concurrency, signed-zero spelling, map order, or temporary-run partition. | `DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-REPLAY-01`, `DTE-REPLAY-02`, `LIFE-REPLAY-01`; no predecessor implementation exists. |
| `C4-ART-02` | The artifact is bounded, canonically encoded, content-addressed, atomically published, and fully validated before a replay-start fact can exist. Wrong binding/date/session/symbol, incomplete or duplicate coverage, malformed schema/order/count/digest, truncation, budget overflow, partial mode presented as complete, or a conflicting existing destination fails without exposing an authoritative artifact. | `ARCH-FLOW-04`, `DTE-SESSION-01`–`04`, `DTE-WINDOW-01`, `DTE-REPLAY-01`, `LIFE-REPLAY-01`; persisted-boundary invariant. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Massive response | HTTP 200; bounded single JSON response; provider status `OK`; exact symbol; unadjusted; validated pagination; every emitted row normalizes and belongs to the requested symbol/interval. Missing/null/empty results are successful empty only under an otherwise successful terminal response. | Wrong status/symbol/adjustment, redirect, foreign or cyclic continuation, third page, trailing JSON, oversized/slow response, invalid row, locally contradictory negative/count evidence, or out-of-range/nonascending/duplicate final row fails that symbol. | Treating an error/foreign-symbol page or truncated pagination as a successful empty/complete interval. |
| Shared row mapper | Every required canonical field and exact timestamp/count conversion passes as one value. | Return one typed bounded rejection; never return partial values and never create coverage. | Defaulting absent `vw`/`n`, rounding a fractional timestamp/count, or fabricating a zero ATS source row into success. |
| Coverage compiler | Every planned binding symbol has one successful terminal interval outcome; aggregate rows and coverage counts reconcile; empty symbols remain explicit. | No final complete artifact after failed, canceled, omitted, duplicate, or mismatched symbol work. | A worker failure or sparse result being indistinguishable from proved no-print. |
| Persisted artifact | Exact closed schema/order, canonical bytes, limits, binding compatibility, per-symbol coverage, summary identities, and recomputed SHA-256 all pass before use. | Reject the complete file before replay start; leave no partial canonical engine state. Existing same-ID bytes are accepted only after the same validation; different bytes are never overwritten. | A truncated/corrupt/partial artifact being labeled complete and advancing replay `T`. |
| Local provider data | Destination is local, normally under ignored `var/`; no credential, authorization header, signed URL, or raw response is retained; the artifact contains only required normalized data/provenance. | No built-in commit/upload/share path. A user-selected nonignored destination requires an explicit warning and remains the operator's rights/retention decision. | Secret leakage or accidental treatment of normalized licensed market data as repository-safe fixture material. |

This contract does not interpret provider licensing. Before retaining or sharing
provider-derived artifacts, the operator must have appropriate rights. Tests
use the one approved small predecessor fixture or repository-owned synthetic
data; implementation and verification do not access credentials or make live
provider calls without a separate explicit owner authorization.

## 11. Failure and terminal behavior

Transport/read failures, 429, and 5xx responses are retryable for at most two
retries after the first attempt. A valid 429 `Retry-After` may delay by at most
15 seconds and only while the caller's context remains live; malformed,
negative, or larger values do not extend the bound. All other non-200
responses, invalid redirects, decode failures, and semantic contradictions are
terminal for that symbol.
Each request attempt has a 15-second deadline and a 16 MiB response limit. A
16-hour, one-second request with limit 50,000 permits at most two pages; any
third continuation is a contract failure rather than unbounded pagination.

Cancellation stops new symbol scheduling, cancels in-flight requests, and
classifies every planned symbol exactly once as complete, failed, or canceled.
Before artifact-publication linearization, the operation returns a bounded
report and publishes no new complete artifact. Cancellation observed after
publication linearization cannot reclassify the completed local persistence
decision as canceled.
There is no resumable partial artifact in version 1; a later attempt starts a
new operation. Ordinary failure closes and removes every operation-owned
temporary file before returning, with cleanup failure retained as one bounded
persistence reason.

Compiler/validator failures are terminal and nonretrying inside one operation.
If the final content-addressed path already exists, identical fully valid bytes
make the operation idempotently successful; invalid or conflicting bytes fail
closed and are not overwritten.

Publication linearizes when the already synced, sealed, whole-file-validated
sibling temporary file is atomically installed at the content-addressed final
name without replacing an existing directory entry. The implementation uses a
same-filesystem hard link or an equivalently proved atomic no-replace primitive;
check-then-rename is forbidden. An `already exists` race re-enters the full
existing-file validation path and never overwrites bytes. Cancellation is
honored through the last check immediately before the no-replace operation.
After publication linearizes, the operation removes its temporary name and
performs exactly one directory sync without consulting caller cancellation.
Successful temporary-name removal and directory sync returns `complete`.
Temporary-name removal or directory-sync failure returns
`persistence_uncertain`: the final name may contain the fully validated
intended bytes and a bounded temporary name may remain, but the operation
returns no validated handle and makes no durability claim. A later operation
may adopt that final name only after the ordinary full existing-file validation;
invalid or different bytes still fail closed.
This is not rollback: an installed directory entry cannot be atomically undone
after an uncertain directory sync.

The destination directory permits at most one compile operation at a time.
Before download, the operation acquires an exclusive sibling lease and performs
a bounded enumeration of the closed Component 4 temporary-name prefix. Each
operation may create no more than the plan's positive `maximum_temporary_files`
and `maximum_temporary_bytes`. Any recognized crash remnants, lease conflict,
or enumeration beyond `maximum_temporary_files+1` fails before new temporary
creation and requires explicit local cleanup; the implementation does not
delete an unrecognized or potentially active file. Consequently ordinary
failure cleans up, one crashed operation can leave only its own bounded files,
and repeated attempts cannot accumulate further retained files.

## 12. Accounting and observability

The downloader has one primary population identity:

```text
planned_symbols = complete_symbols + failed_symbols + canceled_symbols
complete_symbols = nonempty_symbols + empty_symbols
```

The artifact has independent exact identities:

```text
summary.coverage_entries = count(coverage lines)
summary.aggregate_records = count(aggregate lines)
for each symbol: coverage.record_count = count(aggregate records for symbol)
complete_final_bars: coverage_entries = binding.universe_total
```

The compile operation has one terminal identity:

```text
compile_operations = complete + failed + canceled + persistence_uncertain = 1
```

Retry attempts, HTTP statuses, normalized-row rejection reasons, bytes, pages,
and worker occupancy overlap these primary outcomes and are diagnostics, not
additional symbol populations. Metrics use bounded reason/status classes;
symbol, URL, request ID, artifact path, and credential material are not metric
labels. The operation result may retain an exact failed-symbol list bounded by
the requested binding for local diagnosis.

The bounded rejection vocabulary is: request construction, transport/deadline,
HTTP retry exhausted, redirect/continuation, response size/syntax, envelope
identity/status, timestamp, numeric/count, structural aggregate, symbol/
interval/order/duplicate, plan budget, coverage, schema/canonical encoding,
summary/digest/truncation, existing destination, persistence, and canceled.

## 13. Simplicity and boundedness

- Mutable owners/state representations: one stateless mapper; one bounded
  downloader operation; one compiler/validator cursor. None owns scanner state.
- Concurrency and I/O: one to eight symbol workers; sequential pages per symbol;
  at most two pages, three attempts/page, 16 MiB/page, and 15 seconds/attempt.
  The caller supplies positive maximum normalized records, artifact bytes,
  temporary bytes, temporary files, and in-memory records. Checked
  multiplication also enforces the theoretical `symbols * seconds` final-bar
  ceiling.
- Deterministic sorting may use bounded in-memory runs plus temporary external
  merge; exact run size and private file layout are implementation discretion.
  Full-session retention in RAM is not permitted unless it is within the
  explicit in-memory-record budget.
- One exclusive destination lease plus bounded closed-prefix enumeration
  prevents concurrent publication and unbounded crash-remnant accumulation.
  Ordinary cleanup removes only operation-owned files; unknown files are never
  traversed or deleted to make progress.
- Retained predecessor mechanisms: endpoint/query, bearer redaction,
  same-origin continuation, redirect refusal, retry/page/body/deadline bounds,
  exact numeric decoding, sparse rows, and floor ATS.
- Rejected predecessor mechanisms: recovery generation/token ownership,
  recovery planning/concurrency semantics, direct construction of the old
  scanner aggregate, envelope fields ignored as trust evidence, and any claim
  that the predecessor has a replay artifact.
- A single JSON document was rejected because a large records array complicates
  bounded canonical streaming and truncation localization. A directory
  manifest plus data file was rejected because multi-file atomic publication is
  unnecessary. A database, journal, gzip dependency, signed format, and generic
  codec framework have no approved need.
- No provider edge beyond the scoped tests, official field/endpoint contract,
  and Phase 1 invariants is implemented speculatively.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Sparse seconds, including a symbol with zero rows | Phase 1 `DTE-WINDOW-04`; v2 sparse fixture/test; provider docs | Preserve absent slots; record successful empty coverage without fabricating OHLC/VWAP/volume/ATS. | `P-C4-NORM`; `P-C4-DL`; `P-C4-ART-TRUST` |
| Fractional volume and nonintegral `volume/n` | V2 fixture/test; `DTE-AGG-02` | Preserve finite fractional volume; ATS is checked mathematical floor and may be zero. | `P-C4-NORM` |
| Zero/missing/fractional/overflowing transaction count | V2 malformed test; provider field optionality; Phase 1 mapping | Reject row; do not round/default or assert live ATS parity. | `P-C4-NORM` |
| Fractional or non-second timestamp | V2 malformed test; `DTE-SESSION-04` | Reject before interval construction. | `P-C4-NORM` |
| 429/5xx, slow/oversized/trailing response | V2 focused tests | Apply only the fixed retry/deadline/body bounds; terminally fail symbol after exhaustion. | `P-C4-DL` |
| Redirect, foreign/cyclic/third continuation | V2 redirect/pagination tests plus credential containment invariant | Follow only bounded same-origin exact-endpoint continuation; never forward bearer through redirect; fail closed. | `P-C4-DL` |
| Worker completion order differs | Determinism invariant | Canonical sort yields identical bytes, ordinals, coverage order, and artifact ID. | `P-C4-ART-BYTES` |
| Provider rows reach the compiler through another decoder/mapper | One-canonical-path and Component 6 reuse invariant | Construction accepts only `C4-NORM-01` success values; raw provider rows cannot enter artifact encoding. | `P-C4-COMP-NORM` |
| Positive and negative signed zero normalize to the same volume | Component 2 canonical equality and artifact byte-identity invariant | Emit the single JSON number `0` and the same artifact identity. | `P-C4-NORM`; `P-C4-ART-BYTES` |
| Corrupt/truncated/noncanonical artifact or recomputed-looking summary mismatch | Persisted trust-boundary invariant | Whole validation fails before replay start; no complete claim or overwrite. | `P-C4-ART-TRUST` |
| Complete artifact omits a binding symbol or mislabels partial synthetic data | `LIFE-REPLAY-01` | Coverage/binding validation rejects; partial synthetic handle is never commit-supporting. | `P-C4-ART-TRUST` |
| Existing content-addressed destination | Atomic persistence invariant | Valid identical content is idempotent; invalid or unequal content fails without overwrite. | `P-C4-ART-TRUST` |
| Cancellation immediately before/after atomic no-replace publication or directory-sync failure | Persistence linearization invariant | Pre-publication cancellation leaves no new final path; post-publication cancellation does not rewrite the terminal result; sync uncertainty returns no handle and later adoption requires full validation. | `P-C4-ART-TRUST` |
| Crash leaves bounded temporary runs | Retained-state bound | The next operation refuses new temporary creation after bounded detection; ordinary cleanup or explicit local cleanup is required. | `P-C4-ART-TRUST` |

Primary proof definitions and slice allocation are authoritative in
[deterministic-core delivery](deterministic-core-delivery.md#15-primary-proof-allocation).
