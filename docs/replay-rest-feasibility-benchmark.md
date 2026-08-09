# REST replay acquisition feasibility benchmark

**Status:** Proposed evidence plan and future assignment; not product authority,
not an implementation specification, and not credential/provider authorization

**Related proposed component:** [Historical replay product mode](specifications/historical-replay-product-mode.md)

**Accepted dependency:** [Component 4 aggregate replay](specifications/aggregate-replay.md)

## Decision this benchmark supports

Determine whether the accepted per-symbol Massive one-second aggregate
downloader is operationally suitable for routine complete-session replay on the
owner's current subscription, network, and host. The desired full eligible-
universe 04:00-20:00 New York artifact acquisition target is approximately ten
minutes. This target is provisional until the owner accepts the C12 contract.

The benchmark does not validate market semantics already owned by C4 and does
not prove provider SLA, future entitlement, public capacity, live latency,
correction-arrival chronology, or trading edge. Results are labeled with exact
date, interval, symbol-selection identity, worker count, provider plan where
known, host, and command commit.

## Current capability and benchmark gap

`cmd/aggregate-replay` accepts a shorter `--from/--to` interval but always
resolves and downloads the complete binding universe before compiling an
artifact. It has no subset flag. Adding an unreviewed production `--symbols`
option would weaken complete-binding evidence and is prohibited.

The benchmark therefore requires one narrow explicitly selected harness that
constructs a valid benchmark-only subset binding and calls the existing sealed
`massive.OfflineDownloader` directly. Subset results never publish a complete-
universe artifact or acceptance claim. The production downloader, normalizer,
request limits, retries, and accounting remain unchanged.

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

Never record an API key, Authorization header, continuation credential, raw
response body, or licensed provider rows in a committed artifact.

## Sequential trial matrix

Every stage requires the preceding result to be complete and reconciled. A
failure records evidence and stops escalation; it does not justify broadening
scope or raising limits automatically.

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

The ten-minute full-day target is the primary product preference. Before B4,
forecast conservatively from B1-B3 and report uncertainty rather than asserting
linear scaling: per-symbol request overhead, sparse results, 50,000-row
pagination, provider throttling, response bytes, and local parsing do not scale
identically.

Proposed decision bands, subject to owner acceptance:

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
requirement. The benchmark reports evidence; the owner selects the acquisition
path.

## Draft implementation assignment for the benchmark harness

This assignment becomes executable only after C12 boundary approval and an
explicit owner authorization for the exact provider trial.

1. **Authority and requirements:** Read `AGENTS.md`, `README.md`, the
   specification map, C4 parent and REST/artifact detail, the proposed C12
   skeleton, and this plan. Preserve every C4 request, normalization, terminal,
   coverage, and credential-containment rule.
2. **Outcome and owner:** Add one benchmark-only harness that measures the
   existing downloader over an authentic full or deterministic subset binding.
   The harness owns no production state or semantics.
3. **Allowed boundary:** Prefer a long-running Go test or narrowly named local
   benchmark command colocated with C4 acquisition. Do not change production
   CLI flags, downloader behavior, retry policy, worker maximum, artifact
   schema, reference caches, or scanner runtime.
4. **Evidence inputs:** Exact owner-authorized date, interval, subset size,
   `MASSIVE_API_KEY` environment access, accepted reference cache/resolver, and
   official provider endpoint documentation. No V2 or raw flat-file source.
5. **Primary evidence:** `B-C12-REST` is one staged measurement record covering
   B0-B3; B4 is a separately authorized acceptance trial. Semantic unit tests
   remain C4 evidence and are not duplicated.
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
    unexpected pagination, credential exposure, or inability to distinguish
    acquisition from compile work stops the stage and returns evidence to the
    owner. It does not authorize implementation expansion.

## Provider-execution authorization template

The owner may later authorize a stage with a message containing all of:

```text
Authorize B-C12-REST stage <B0/B1/B2/B3/B4> for trading date <YYYY-MM-DD>,
interval <04:00-08:00 or 04:00-20:00 America/New_York>, population
<deterministic N or complete binding>, workers <N>, and the recorded hard
timeout. You may read MASSIVE_API_KEY only for this trial. Do not commit provider
data or credentials and stop at the stage boundary.
```

Approval of the C12 specification does not itself authorize this execution.
