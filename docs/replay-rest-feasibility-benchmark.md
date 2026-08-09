# REST replay acquisition feasibility benchmark

**Status:** B0-B4 completed under exact owner authorizations on 2026-08-09;
local harness verification and B4 artifact reopen validation succeeded. This
document records acquisition evidence and future rerun safeguards, not product
authority or C12 implementation approval.

**Related proposed component:** [Historical replay product mode](specifications/historical-replay-product-mode.md)

**Accepted dependency:** [Component 4 aggregate replay](specifications/aggregate-replay.md)

## Authorization boundary

The owner's initial 2026-08-09 authorization permitted only the benchmark-local
code and ordinary deterministic tests needed to construct a strict subset
binding, invoke the sealed C4 `massive.OfflineDownloader`, and report its
existing measurements honestly. It permitted no production CLI flag,
downloader/retry/worker-limit change, artifact-schema change, scanner-runtime
change, C12 product implementation, credential read, provider request,
licensed-data download, or B0-B4 execution. Separate later owner messages
authorized each completed provider stage and B4 artifact construction with
exact bounds; none created continuing credential or provider authority.

Implementation authorization and provider-execution authorization are
separate. The local harness passed fake-HTTPS tests before the C12 boundary was
approved. The completed 2026-08-09 stages authorized no continuing provider
access. Every future rerun still requires a new owner message naming the exact
stage, trading date, interval, population, workers, hard timeout, and permission
to read `MASSIVE_API_KEY`. C12 approval by itself does not supply those facts or
authorize another provider request.

## Completed evidence summary

B0-B3 established credential/entitlement operation, subset interval and worker
scaling, and exact full-binding premarket request completion. B4 then compiled
and independently reopened one complete 2026-08-07 04:00-20:00 New York
artifact:

- 5,691 of 5,691 symbols completed; zero failed or canceled;
- 5,519 nonempty and 172 successful-empty symbols;
- 7,671,171 normalized one-second aggregate records;
- 5,691 HTTP pages, 5,750 attempts, and 693,876,166 response bytes;
- 2,584,011,150 artifact bytes;
- 12:28.835 acquisition/compile wall time plus 1:32.697 independent reopen
  validation; and
- exact binding, interval, coverage, record-count, and SHA-256 identity match on
  reopen.

The run missed the approximate ten-minute preference but completed inside its
authorized fifteen-minute compile deadline. REST acquisition is accepted for
the first C12 workflow. Peak Go heap allocation reached 11.58 GB; routine daily
artifact construction on this 8 GiB host should later replace the current
memory-heavy compile path with the already-permitted bounded spill/merge design.
That is a C4 operational follow-up, not a C12 semantic dependency. Licensed
artifact bytes and exact-date caches remain private, ignored, and uncommitted.

## Decision this benchmark supports

Determine whether the accepted per-symbol Massive one-second aggregate
downloader is operationally suitable for routine complete-session replay on the
owner's current subscription, network, and host. The desired full eligible-
universe 04:00-20:00 New York artifact acquisition target is approximately ten
minutes. The owner accepted that preference as a target rather than a hard
acceptance limit; B4's bounded 12:28.835 result retains REST for C12.

The benchmark does not validate market semantics already owned by C4 and does
not prove provider SLA, future entitlement, public capacity, live latency,
correction-arrival chronology, or trading edge. Results are labeled with exact
date, interval, symbol-selection identity, worker count, provider plan where
known, host, and command commit.

## Current capability and benchmark gap

`cmd/aggregate-replay` accepts a shorter `--from/--to` interval but always
resolves and downloads the complete binding universe before compiling an
artifact. It has no subset flag, and this benchmark does not add one.

The authorized local harness derives an evenly spaced strict subset from the
sorted accepted binding, constructs a valid callback-confined subset binding,
and calls the existing sealed `massive.OfflineDownloader` directly. Its sealed
report has fixed `deterministic_subset` scope, fixed false complete-universe/
artifact/acceptance eligibility, and no binding value, normalized rows,
`DownloadResult`, artifact identity, or path. The production downloader,
normalizer, request limits, retries, accounting, CLI, and scanner remain
unchanged.

The owner's 2026-08-09 B3 authorization adds one separate download-only full-
binding report path. It invokes the same downloader over the accepted binding,
records exact binding identity and sorted-symbol digest, and seals only the
existing measurement facts. Its `complete_binding_download_only` scope is
explicitly distinct from artifact completeness and product acceptance, both
of which remain false and structurally unavailable. It cannot compile or
return an artifact, binding, `DownloadResult`, normalized row, artifact
identity, or path. B4 artifact compilation was separately authorized, completed,
and independently validated outside this harness.

### Local implementation evidence

`internal/reference/benchmark_subset.go` derives the strict subset from the
accepted binding's schedule, universe, and prior-close facts, then re-enters
the ordinary binding assembler. `internal/massive/offline_benchmark.go` applies
the fixed evenly-spaced selection or exact full binding, hard timeout, and
existing downloader, then seals only the applicable measurement report. It
contains no credential lookup or provider-stage runner.

The ordinary fake-HTTPS proof covers deterministic selection identity;
successful nonempty and empty symbols; pagination, retry attempts, response
bytes, and normalized-row counts; exact failed/canceled terminal accounting;
in-flight timeout cancellation; rejection of complete-binding scope through
the subset API before HTTPS; and full-binding download success/failure with
fixed false artifact/acceptance eligibility. Focused and repository-short
verification pass. No production downloader, provider, artifact, CLI, or
scanner behavior changes. This document persists only bounded nonsecret stage
accounting; it contains no licensed rows, artifact path, credential, header, or
continuation value.

## Required measurements

For each trial record:

- trading date and exact UTC `[from,to)` plus New York interpretation;
- full binding symbol count, selected symbol count, deterministic selection
  method, sorted-symbol digest, and any separately labeled stress symbols;
- worker count and hard context timeout;
- wall duration and, when measurable without changing the downloader,
  acquisition versus local postprocessing duration;
- planned/complete/failed/canceled and nonempty/empty symbols;
- records, HTTP pages, attempts, response bytes, and per-symbol distributions;
- observed 429, retryable 5xx, timeout, cancellation, pagination, and terminal
  reasons using bounded low-cardinality counts;
- artifact or temporary bytes only for a separately authorized complete compile;
- peak resident memory, CPU time, and host facts when the selected harness can
  measure them without credentials or payload logging; and
- whether accounting reconciles exactly.

The existing sealed downloader supports exact aggregate and per-symbol
terminal state, record, page, attempt, and response-byte measurements. It does
not expose successful-attempt HTTP status or retry-reason history, separate
network acquisition from local parsing/postprocessing time, or measure process
CPU, peak RSS, and host facts. The subset harness labels those fields
unavailable; it does not infer them from terminal reason, wall time, or response
size. Artifact and temporary bytes are also unavailable because the subset
harness cannot compile or publish an artifact.

Never record an API key, Authorization header, continuation credential, raw
response body, or licensed provider rows in a committed artifact.

## Sequential trial matrix

This matrix records the completed 2026-08-09 sequence and remains the protocol
for any future rerun. It is not continuing authorization. Every future stage
requires its own exact owner authorization and the preceding result to be
complete and reconciled. A failure records evidence and stops escalation; it
does not justify broadening scope or raising limits automatically.

| Stage | Population and interval | Workers | Hard timeout | Question answered |
| --- | --- | --- | --- | --- |
| `B0` preflight | 10 deterministic eligible symbols; 04:00-08:00 New York | 4 | 2 minutes | Are credential, entitlement, request, normalization, and accounting paths functional? |
| `B1` subset concurrency | 100 deterministic eligible symbols; 04:00-08:00 | 4 then 8 | 5 minutes each | Does concurrency improve throughput without throttling or failures? |
| `B2` interval scaling | Same 100 symbols; 04:00-20:00 | Selected B1 worker count | 10 minutes | How do rows, bytes, pagination, parsing, and wall time scale from premarket to full session? |
| `B3` full-universe request scale | Exact complete eligible universe; 04:00-08:00 | Selected count, never above accepted maximum 8 | 15 minutes | Do thousands of per-symbol requests remain bounded under real plan-level limits? |
| `B4` complete-day acceptance | Exact complete eligible universe; 04:00-20:00 and complete artifact compile/validation | Selected count | 15 minutes initially | Does the actual full product acquisition meet the owner's target? Run only after separate owner authorization based on B0-B3. |

Use an evenly spaced deterministic selection from the sorted accepted universe
for the representative subset. A separately named small high-activity stress
set may be added only to exercise row/page volume; it is not representative and
must not replace the deterministic subset.

## Interpretation and decision rule

The ten-minute full-day target is the primary product preference. For any future
B4 rerun, forecast conservatively from preceding stages and report uncertainty
rather than asserting linear scaling: per-symbol request overhead, sparse
results, 50,000-row pagination, provider throttling, response bytes, and local
parsing do not scale identically.

Decision bands retained for future evidence interpretation:

- actual B4 complete artifact at or below ten minutes with no unresolved
  terminal failure: retain REST acquisition for the first C12 implementation;
- B3 or projected B4 near ten minutes: run B4 once before deciding and report
  sensitivity to workers/rate limits;
- persistent 429/retry/failure behavior or a credible B4 forecast above thirty
  minutes: do not tune blindly; prepare an owner-reviewed C4 flat-file source
  feasibility proposal; and
- an hours-scale forecast, inability to complete every symbol, or entitlement
  mismatch: REST is not accepted as the routine complete-day acquisition path.

The thirty-minute boundary is a proposed investigation trigger, not a product
requirement. B4's actual bounded success selected REST for the first C12 path;
future evidence may reopen that operational decision without changing replay
semantics.

## Authorized local-harness implementation assignment

This assignment is executable under the owner's 2026-08-09 message only for
local code and deterministic fake-HTTPS ordinary tests. Its presence does not
authorize any provider trial.

1. **Authority and requirements:** Read `AGENTS.md`, `README.md`, the
   specification map, C4 parent and REST/artifact detail, the proposed C12
   skeleton, and this plan. Preserve every C4 request, normalization, terminal,
   coverage, and credential-containment rule.
2. **Outcome and owner:** Add one benchmark-only harness that measures the
   existing downloader over a deterministic strict subset of an accepted
   binding. The harness owns no production state or semantics and returns no
   artifact-capable value.
3. **Allowed boundary:** Prefer a long-running Go test or narrowly named local
   benchmark command colocated with C4 acquisition. Do not change production
   CLI flags, downloader behavior, retry policy, worker maximum, artifact
   schema, reference caches, or scanner runtime.
4. **Evidence inputs:** For local implementation, repository-owned synthetic
   bindings and fake HTTPS only. A future real stage additionally requires the
   exact owner-authorized date, interval, population, workers, timeout,
   `MASSIVE_API_KEY` access, and accepted reference source. No V2 or raw
   flat-file source.
5. **Primary evidence:** Ordinary tests prove only selection identity,
   accounting/measurement preservation, bounded terminal containment, and the
   subset-only claim boundary. `B-C12-REST` begins only when B0 is separately
   authorized; B4 remains a distinct acceptance trial. Existing C4 semantics
   are consumed rather than re-proved.
6. **Verification tier:** Harness unit tests use fake HTTPS and run ordinary;
   provider trials are explicitly selected, skipped under `testing.Short()`,
   and bounded by the table timeouts. Ordinary command remains
   `go test -short -timeout 2m ./...`.
7. **Deferred behavior:** No dashboard replay, API bridge, flat-file compiler,
   T/Q replay, full-day provider run, or acquisition-source decision.
8. **Prohibited changes:** No second aggregate mapper, fake WebSocket, altered
   complete-binding meaning, credential logging, provider rows in Git,
   fabricated benchmark completion, or automatic threshold increase.
9. **Dangerous counterexamples and limitations:** A subset result presented as
   full-universe capacity, a fast empty/error response counted as success,
   hidden 429/retry cost, nonreconciling terminal accounting, or an extrapolated
   estimate presented as B4 actual must fail the evidence record. One date/plan/
   host does not prove future SLA.
10. **Correction and stop conditions:** Invalid binding/subset identity,
    provider schema mismatch, unbounded wait, repeated terminal failure,
    unexpected pagination, credential exposure, or an unsupported measurement
    stops that stage and records the limitation. It does not authorize
    production changes, invented data, or implementation expansion.

## Provider-execution authorization template

The owner may later authorize a stage with a message containing all of:

```text
Authorize B-C12-REST stage <B0/B1/B2/B3/B4> for trading date <YYYY-MM-DD>,
interval <04:00-08:00 or 04:00-20:00 America/New_York>, population
<deterministic N or complete binding>, workers <N>, and the recorded hard
timeout. You may read MASSIVE_API_KEY only for this trial. Do not commit provider
data or credentials and stop at the stage boundary.
```

Approval of the C12 specification or of the local harness does not itself
authorize this execution.
