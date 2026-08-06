# Reference data and exchange schedule/session binding

**Status:** Finally approved 2026-08-05 after the repeated independent final
component review; original S1–S3 and corrective C1-R1–C1-R3 are accepted. The
component is not frozen.

**Owner boundary approval:** approved 2026-08-05 in the owning Codex task

**Original owner contract/reuse/test/slice-plan approval:** approved 2026-08-05
in the owning Codex task

**Corrective contract/test/slice-plan approval:** approved 2026-08-05 in the
owning Codex task through the bounded `C1-R1` implementation assignment

**C1-R1 owner slice acceptance:** accepted 2026-08-05 in the owning Codex task

**C1-R2 owner slice acceptance:** accepted 2026-08-05 in the owning Codex task

**C1-R3 owner slice acceptance:** accepted 2026-08-05 through the successful
repeated independent final component review

**Controlling Phase 1 requirements:** `PG-UNIVERSE-01`,
`PG-REFERENCE-01`, `PG-RANK-01`, `PG-RANK-05`, `PG-OBS-01`,
`PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-04`,
`DTE-SESSION-01`, `DTE-SESSION-02`, `DTE-SESSION-03`,
`DTE-SESSION-04`, `DTE-CLOCK-01`, `DTE-CLOCK-04`, `DTE-REJECT-02`, `LIFE-MODEL-01`,
`LIFE-INIT-02`, `LIFE-INIT-03`, `LIFE-INIT-04`, `LIFE-INIT-05`, and
`LIFE-REPLAY-01`

**Approved dependencies:** none beyond the approved Phase 1 product and
architecture contracts

**Final component review:** repeated independent review passed 2026-08-05 with
no blocking conformance, proof, boundedness, predecessor-coupling, ownership,
or scope defect. Component 1 is finally approved and its dependency boundary is
available for subsequent specification decisions. Component 1 is not frozen,
and Component 2 implementation remains unauthorized without a separate owner
decision.

### Corrective-revision scope

The final review found that the original contract already controlled most of
the required behavior, but several provider, cache, filesystem, proof, and
binding details were not precise enough to prevent an implementation from
weakening the intended boundary. This revision:

- makes provider response-status, optional count, and duplicate-JSON-member
  handling explicit without globalizing attributable prior-close row failures;
- makes fresh acquisition independent from cache preparation and preserves
  exact-date cache equivalence when provider configuration is unavailable;
- fixes prior-close retrieval-time validity, ancestor-symlink rejection,
  temporary-file retention, and bounded cache-directory enumeration;
- requires final binding assembly to revalidate schedule facts rather than
  trusting a caller-owned value copy;
- fixes the bounded diagnostic result and the primary proofs that were only
  evident in code or tested for one resolver/cache class; and
- defines three sequential corrective slices. It does not authorize their
  implementation, reopen `S1`, expand predecessor reuse, or add component-2
  behavior.

## 1. Outcome and user consequence

This component produces a deterministic immutable session binding for one
scanner run. The binding contains the exchange-local trading date, the UTC
instants corresponding to the half-open 04:00–20:00 `America/New_York` scanner
session, the exact eligible-universe identity and population, the immediately
preceding completed regular-session date, the adjusted-prior-close policy and
validated per-symbol data for that date, and a stable binding identifier.

When the component succeeds with current reference facts, downstream live or
replay initialization can establish that all same-session state refers to one
coherent market date, population, and prior-close basis. A missing or invalid
prior close remains a symbol-local unrankable category. An invalid or ambiguous
global binding prevents a live-current claim. A validated prior-date universe
cache can keep the process observable but cannot support current ranking; a
validated prior-close cache for the exact required prior-session date has the
same product meaning as fresh retrieval.

## 2. Scope and explicit non-scope

**In scope**

- Resolve and validate the exchange-local trading date, scanner bounds, and
  immediately preceding completed regular session from an exchange schedule.
- Obtain, paginate, normalize, filter, validate, and identify the current-run-
  date Massive reference universe required by `PG-UNIVERSE-01`.
- Obtain and validate finite positive adjusted closes for the exact required
  prior regular-session date, preserving explicit missing and invalid results.
- Validate cache provenance, date, policy, and currentness before cached facts
  can contribute to a binding.
- Construct an immutable binding and deterministic identities from the
  validated schedule, universe, and prior-close facts.
- Return bounded success, partial per-symbol prior-close status, or terminal
  global failure facts to the initialization boundary.

**Not in scope**

- `ScannerStateEngine` lifecycle, binding installation, engine time, committed
  watermark advancement, or suppression/readiness decisions. Those remain
  owned by the engine under the Phase 1 lifecycle.
- Canonical live symbol state, aggregate acceptance or reconciliation,
  hydration/no-print proof, qualification, ranking, feature evaluation, T/Q,
  or population-accounting evaluation.
- Checkpoint discovery, compatibility decisions, installation, projection, or
  storage. This component only supplies binding facts and identity needed by
  the later checkpoint boundary.
- Snapshot/API schemas, HTTP behavior, UI presentation, or operator controls.
- Live WebSocket handling, aggregate REST hydration/recovery, replay artifact
  behavior, or a second clock, state owner, or evaluator.

## 3. Ownership and dependencies

The single ownership boundary is reference acquisition and validation through
construction of one immutable session-binding result. The component owns no
mutable scanner state. Any bounded request, pagination, retry, or cache state is
transient component-local I/O state and cannot become an alternative session,
readiness, or ranking authority.

Its external dependencies are an exchange-calendar source and Massive
reference/prior-close REST data. Source adapters and concurrent workers return
validated facts, provenance, or terminal failures. They do not install a
binding or mutate canonical symbol state. `ScannerStateEngine` remains the sole
owner of the active binding after it consumes the completed immutable result in
one ordered transition. Live and replay consumers must use the same binding
meaning; a run cannot replace it in place.

## 4. Settled Phase 1 semantic boundary

| Boundary item | Settled meaning | Controlling Phase 1 IDs |
| --- | --- | --- |
| Trading date and prior session | `trading_date` is an `America/New_York` exchange-local date. A schedule—not weekday arithmetic—validates it and identifies the immediately preceding completed regular session across weekends, holidays, early closes, and DST. An early regular close does not shorten the scanner session. | `DTE-SESSION-01`, `DTE-CLOCK-01` |
| Scanner bounds | `S` and `E` are the timezone-aware UTC instants corresponding to 04:00 and 20:00 New York time for the bound date; the scanner session is exactly `[S,E)`. The component does not create a new clock or infer market-data arrival from schedule time. | `DTE-SESSION-02` through `DTE-SESSION-04`, `DTE-CLOCK-01` |
| Eligible universe | The population contains only active provider records with `market=stocks`, `locale=us`, and type `CS` or `ADRC`. Exact current ranking requires a current-run-date universe binding; excluded and unknown records cannot silently enter it. Provider-filter exclusions are acquisition diagnostics, not additional primary states inside the bound `universe_total`. | `PG-UNIVERSE-01`, `PG-OBS-01` |
| Adjusted prior close | Each usable close is finite, positive, adjusted, and explicitly identified with the exact immediately preceding completed regular-session date and adjustment policy. No open, unadjusted close, or unidentified older close may substitute. | `PG-REFERENCE-01`, `PG-RANK-01` |
| Cache meaning | A validated prior-date universe cache is observable-only and cannot be labeled current. A validated prior-close cache is product-equivalent to fresh retrieval only when it names the exact required prior-session date and adjustment policy. Cache mechanics cannot weaken those currentness rules. | `PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-RANK-05`, `PG-OBS-03` |
| Binding contents and isolation | One immutable binding joins the trading date, `S`, `E`, universe identity/population, required prior-session date, prior-close policy/data, and stable identifier. Canonical same-session state belongs to exactly one binding; a different date, universe, or policy requires a new engine run rather than in-place replacement. | `DTE-SESSION-02`, `LIFE-MODEL-01`, `LIFE-INIT-02`, `LIFE-REPLAY-01` |
| Failure containment and accounting | A missing or invalid prior close makes only that symbol unrankable and separately countable. An invalid or ambiguous schedule/universe/global binding prevents live initialization. Neither case may be collapsed into another market-population state. | `PG-REFERENCE-01`, `PG-RANK-01`, `PG-OBS-01`, `PG-OBS-03`, `DTE-REJECT-02`, `LIFE-INIT-02`, `LIFE-INIT-05` |
| Installation and downstream ownership | Reference work may run concurrently, but it returns facts. The engine atomically installs the completed immutable binding and exclusively owns lifecycle, canonical state, accounting, evaluation, readiness, and publication. Checkpoint handling and pre-session transitions occur only after or through that engine-owned boundary. | `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-04`, `LIFE-INIT-02` through `LIFE-INIT-05` |

This component introduces no new product rule, competing mutable owner,
watermark, evaluator, T/Q-to-ranking dependency, changed time-window meaning,
or duplicated responsibility.

## 5. Unresolved questions and evidence needs

| Question | Why Phase 1 does not settle it | Evidence needed | Decision enabled |
| --- | --- | --- | --- |
| Which exchange-calendar source and version/update policy should supply trading dates and completed regular-session schedules, and what cached schedule evidence remains acceptable? | Phase 1 fixes schedule semantics but delegates the source and retrieval/cache policy. | Version 2 calendar selection code/tests, library or source documentation, and existing boundary fixtures. | Calendar dependency, update/currentness checks, and terminal behavior when schedule evidence is unavailable or invalid. |
| How does Massive reference pagination map records into the exact `CS`/`ADRC` universe, including duplicate symbols, response bounds, retry behavior, and cache provenance? | Phase 1 fixes eligibility but not provider fields, pagination, request mechanics, or evidenced conflict handling. | Version 2 universe-loading code, JSON/fake-provider fixtures, and provider documentation where fixtures do not establish the mapping. | Exact provider mapping, pagination completion evidence, bounds, retry/cache contract, and rejection reasons. |
| How are adjusted prior closes requested and mapped, which provider adjustment flags are required, and how is the exact prior-session date proved on every accepted value or dataset? | Phase 1 fixes economic meaning and exact-date validation but not the REST endpoint/fields or request mechanics. | Version 2 prior-close code/tests/fixtures and relevant provider documentation. | Retrieval and normalization contract, adjustment/date validation, and response-level failure policy. |
| What cache keys, metadata, validation, atomicity, and expiration/update mechanics preserve the already-settled universe and prior-close currentness meanings? | Phase 1 defines when cached facts are current or observable-only, not their storage representation or refresh mechanics. | Version 2 cache code/tests plus invariants for date, policy, source, and content identity. | Cache acceptance rules, bounded retention, and safe fallback outcomes. |
| Which canonical encoding and digest/version scheme makes universe and binding identities deterministic across pagination order, retries, and process restarts while changing whenever a binding-defining fact changes? | Phase 1 requires stable identities but delegates their representation and construction. | Version 2 binding/identity behavior if present, checkpoint compatibility needs visible without opening unrelated checkpoint code, and deterministic-encoding invariants. | Exact identity contract consumed by later components. |
| What bounded concurrency, retry, timeout, cancellation, and terminal failure vocabulary should reference acquisition expose? | Phase 1 requires bounded scheduled progress and terminal disposition but delegates numeric bounds and component result shapes. | Version 2 request/retry behavior and tests, provider limits/documentation, and later capacity evidence where exact values remain unknown. | Bounded work model and explicit global versus per-symbol terminal outcomes. |
| Which existing fixtures can serve as the primary evidence for schedule, universe, prior-close, identity, containment, and cache proofs without duplicating the same claim at multiple layers? | Phase 1 names required proof scenarios but not the component-local fixture allocation. | Narrow review of the proposed version 2 tests/fixtures and their provenance/limitations. | Detailed primary-proof allocation and final sequential-slice plan. |

Original post-reconnaissance status: none of these remained an unanswered owner
question when the first contract was approved. Final component review later
identified the narrower provider-ambiguity, cache-independence, filesystem,
diagnostic, and binding-validation gaps recorded in the corrective-revision
scope and Sections 10–16. Those corrections use current official provider
documentation, the implemented failure scenarios, and Phase 1 invariants; they
do not reopen predecessor reconnaissance. Exact numeric request bounds retain
the already-tested version 2 limits. The corrective 64-entry cache-directory
bound is an owner-review decision derived from the existing 14-date retention
plus demonstrated interrupted/unexpected-entry accumulation; it may not be
silently tuned after approval.

## 6. Proposed version 2 reconnaissance scope

The only predecessor proposed for inspection after owner approval is
`/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2`. Reconnaissance
will remain limited to the areas and questions below; exact paths and symbols
will be recorded in Section 8 only after discovery.

| Proposed code/test/fixture area | Question it should answer | Explicit exclusion |
| --- | --- | --- |
| Schedule/calendar selection code and its focused tests or date fixtures | Which source/library and conversions already handle valid dates, preceding sessions, holidays, early closes, and DST correctly? | Scanner lifecycle/orchestration, runtime clock advancement, recovery, and unrelated scheduling packages. |
| Massive reference-universe loader, pagination code, and JSON/fake-provider fixtures | Which provider fields, page-completion rules, eligibility mappings, duplicate handling, bounds, retries, and cache metadata have evidence? | Live WebSocket handling, aggregate REST hydration, ranking, canonical state, and unrelated provider endpoints. |
| Adjusted-prior-close retrieval/normalization code and focused fixtures | Which endpoint, adjustment flags, date fields, value validation, pagination/batching, and missing-result behavior are evidenced? | Aggregate mark/feature computation, hydration/no-print logic, qualification, and ranking. |
| Reference-data cache validation and tests | Which keys, provenance, exact-date/currentness checks, write/read safeguards, and stale-cache behaviors are worth preserving? | Checkpoint storage/installation, general persistence frameworks, databases, and unrelated caches. |
| Session-binding or deterministic identity code and tests | Is there existing order-independent identity behavior or only coupling that should be rejected, and what downstream compatibility assumption is evidenced locally? | Opening checkpoint, API, UI, replay-artifact, or orchestration packages merely to discover additional consumers. |
| Focused reference request/retry/concurrency tests | Which terminal outcomes and numeric/request bounds are evidenced rather than speculative? | Scanner recovery ownership, global operations policy, and broad load or integration suites. |

Scanner orchestration, `ScannerStateEngine`, canonical live symbol state,
WebSocket handling, aggregate acceptance/reconciliation, hydration and recovery
ownership, qualification, ranking, readiness, checkpoints, replay execution,
API, UI, version 1, and unrelated packages are explicitly excluded.

### 6.1 Initial proof and slicing boundaries

| Likely proof boundary | Controlling requirement IDs | What it would establish | Evidence still needed |
| --- | --- | --- | --- |
| Schedule selection across a weekend, exchange holiday, and early-close predecessor, with a DST-offset case | `DTE-SESSION-01`, `DTE-CLOCK-01` | The selected trading/prior dates come from the exchange schedule and the correct New York offset is used rather than weekday or fixed-UTC arithmetic. | Exact calendar source and focused fixtures. |
| 04:00-inclusive and 20:00-exclusive scanner bounds converted to UTC | `DTE-SESSION-02` through `DTE-SESSION-04` | `S` is included, `E` is excluded, the final second is `[E-1s,E)`, and persisted/component timestamps retain explicit timezone and exact precision. | Calendar/time library behavior and any existing boundary fixtures. |
| Eligible `CS`/`ADRC` filtering with deterministic universe identity and exclusion accounting | `PG-UNIVERSE-01`, `PG-OBS-01` | Every considered reference record is deterministically included once or assigned one bounded top-level exclusion result, only the included current-run-date population defines `universe_total`, and input/page order cannot change its identity. | Provider mapping, pagination, duplicate behavior, and fixtures. |
| Exact prior-session and adjusted-close validation | `PG-REFERENCE-01`, `PG-RANK-01`, `DTE-SESSION-01` | Only finite positive adjusted closes naming the required schedule-derived date become valid prior-close facts; older, unadjusted, nonpositive, or nonfinite data is rejected. | Endpoint/flag/date mapping and representative fixtures. |
| Deterministic complete binding identity | `DTE-SESSION-02`, `LIFE-MODEL-01`, `LIFE-INIT-02`, `LIFE-REPLAY-01` | Equivalent validated inputs yield the same binding identity across order/retry/restart, while any binding-defining date, bound, universe, prior-close dataset, or policy change yields a different identity. | Identity representation and downstream compatibility evidence. |
| Missing/invalid prior-close containment | `PG-REFERENCE-01`, `PG-RANK-01`, `PG-OBS-01`, `DTE-REJECT-02`, `LIFE-INIT-02` | Per-symbol prior-close failure remains explicit and does not invalidate an otherwise coherent global binding or fabricate a substitute value. | Terminal reason model and fixtures for missing/invalid results. |
| Cache acceptance versus observable-only stale behavior | `PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-RANK-05`, `PG-OBS-03`, `LIFE-INIT-05` | Exact required-date/policy cached prior closes can contribute current facts, while a prior-date universe cache cannot support current ranking and invalid cache evidence cannot silently initialize a binding. | Existing cache metadata/validation behavior and focused fixtures. |

**Provisional delivery assessment:** multiple sequential slices are likely

**Reason and likely slice outcomes:** Schedule/session derivation and stable
identity are deterministic, mostly provider-independent behaviors with calendar
boundary proofs. Universe acquisition/filtering and adjusted-prior-close
acquisition each cross a provider I/O/normalization/cache boundary and require
different fixtures and failure proofs. A likely progression is therefore:
(1) schedule-derived session configuration and deterministic binding identity,
(2) current-run-date eligible-universe acquisition and cache validation, and
(3) exact adjusted-prior-close acquisition, symbol-local containment, and
complete binding assembly. This is not a finalized slice plan; reconnaissance
may show that identity belongs with final assembly or that provider mechanics
can be reviewed coherently in fewer slices. No private helper, package, or file
layout is prescribed.

## 7. Boundary-approval checkpoint

- [x] Exact controlling Phase 1 IDs are enumerated.
- [x] Outcome, ownership, dependencies, scope, and non-scope are unambiguous.
- [x] Settled inputs/outputs/state and invariants are sufficient to prevent
      version 2 from changing the architecture.
- [x] Unresolved questions are genuinely delegated details.
- [x] Proposed version 2 reconnaissance is narrow and question-driven.
- [x] No version 2 code, tests, or fixtures were opened while preparing the
      skeleton.
- [x] Initial likely proof boundaries are identified without inventing a broad
      test matrix.
- [x] The skeleton states whether the component is likely to need multiple
      sequential implementation slices and why; this is provisional until
      detailed evidence and proofs are complete.

**Owner decision:** approved 2026-08-05; proceed only with the version 2
reconnaissance scope named in Section 6

Boundary approval above authorized only the named version 2 reconnaissance.
Sections 8–19 record the resulting detailed draft; they do not authorize
implementation until the separate contract/reuse/test/slice-plan approval.

## 8. Version 2 reconnaissance and reuse assessment

The approved predecessor was inspected at commit
`5f92a151dd850002578a33a81ad90dea096c63b6`. Its worktree contained unrelated
user changes, but none of the inspected files below was modified. File hashes
pin the exact evidence reviewed.

| Exact source path and function/type/test/fixture | Commit/file hash when relevant | Finding or validated behavior | Decision (`direct port` / `adapt` / `behavior evidence` / `reject`) | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- | --- |
| `internal/scanner/schedule.go`: `LoadSchedule`, `PreviousTradingDate`, `SessionBounds` | SHA-256 `26900d82b5f8b6c6b81ee7418256e156448efb942994993d98fa81ae4b63acaa` | Strictly decodes a checksummed embedded NYSE artifact, uses embedded `America/New_York` tzdata, selects prior trading dates from declared rows, preserves early-close metadata, and derives 04:00/20:00 civil bounds. | `adapt` | Keep deterministic artifact validation and schedule lookup; accept an explicit requested trading date rather than reading wall time, use civil-time construction for `S`/`E`, and return facts without lifecycle decisions. | Schedule artifact validation and weekend/holiday/early-close/DST boundary proof. |
| `internal/scanner/schedule_test.go` | SHA-256 `083961def222ad44fbbe4ae8511437234db1933e0429abc713e9fdce9e926dc8` | Covers declared holidays, weekends, early close, DST UTC offsets, coverage rejection, and prior-session selection. The isolated focused test passed during reconnaissance. | `behavior evidence` | Split artifact-integrity and session-selection claims so each has one primary proof. Add exact 20:00 exclusion/final-second assertions. | `REF-SCHEDULE-01` and `REF-SCHEDULE-02` proofs. |
| `internal/scanner/nyse_trading_days.json` and `docs/nyse-schedule-provenance.json` | Artifact SHA-256 `3fb957d2c41f53e64883699d6324f378fd896755fa299a4bd209d1e60d190784`; provenance SHA-256 `1fc9ca73d052b8752c769fefa1302d070071a1350ef4f13109afa237c27bd03d` | A 502-row, 2026–2027 reviewed NYSE schedule with source URL, review date, three early closes, schema/version, coverage, and checksum. Current NYSE documentation now also publishes 2028, confirming that artifact renewal is an explicit release-time responsibility. | `adapt` | Preserve the reviewed immutable artifact model; validate provenance and exceptional closures, and require replacement before an unsupported run date or after an official revision. Do not add a runtime calendar service. | Artifact schema, checksum, coverage, ordering, official-source provenance, and renewal-boundary proof. |
| `internal/massive/universe.go`: `UniverseResolver`, `fetch`, pagination/cache functions | SHA-256 `46922ae4912822af9366089c351825b61f976cfcc6a978b3fe1d5ff5b950bc1a` | Uses `/v3/reference/tickers` with exact run date, `active=true`, `market=stocks`, 1,000-row pages, sorted pagination, `CS`/`ADRC` filtering, closed exclusion counts, duplicate/cycle/origin checks, three attempts, strict caches, and stale fallback labeling. | `adapt` | Keep request, mapping, bounds, accounting, cache hardening, and same-date fallback. Remove runtime `/tickers/types` validation, return a prior-date cache only as observable stale evidence, remove mutable set/map ownership from the result, and add deterministic universe identity. | Provider-normalization/pagination fixture plus order-invariant identity and cache-currentness proofs. |
| `internal/massive/universe_test.go` | SHA-256 `d80829422bca6426088a1944a51857ee6ebef3bf6f3b77db0d4df9de556ca5ae` | Exercises exact query parameters, two-page pagination, every eligibility exclusion, case-sensitive sorting, cache publication/fallback, object/array type responses, redirect rejection, response-read retry, duplicate/cycle/date/credential rejection, and strict cache mutations. | `behavior evidence` | Retain only component-local cases. Replace dynamic type-discovery coverage with fixed documented mapping, add page-order identity invariance, and separate stale observable output from current binding assembly. | `REF-UNIVERSE-01`, `REF-UNIVERSE-02`, and cache proofs. |
| `internal/massive/prior_close.go`: `PriorCloseResolver`, `fetch`, cache functions | SHA-256 `6827407e3971b149604a53ed82b12e3ff8ffbeda62a3e07a96fb8800aa6e2025` | Uses the grouped daily endpoint with `adjusted=true`, `include_otc=false`, validates response adjustment/count/date/value, filters exact case-sensitive eligible symbols, retries bounded failures, and accepts only an exact-date cache. | `adapt` | Preserve the grouped request and source metadata. Treat valid empty data as all symbols missing; convert attributable invalid/duplicate/wrong-date rows into symbol-local invalid states; keep only envelope/source ambiguity global; preserve fresh data when cache persistence fails; cache normalized valid/invalid dataset evidence. | Exact adjustment/date fixture, per-symbol containment/accounting, same-date cache, and persistence-degradation proofs. |
| `internal/massive/prior_close_test.go` | SHA-256 `d0c6df76c3abe107fdd0b250aca63e1fa0776d9bfe4a509024f69af17e7076ef` | Demonstrates exact-symbol case, missing counts, bearer credential redaction, atomic same-date fallback, and rejection of duplicate, zero, and wrong-date rows. | `behavior evidence` | Change the expected duplicate/invalid-row outcome from global rejection to invalid only for the attributable symbol; add valid empty, nonfinite, partial-valid, cache-persistence, and identity cases. | `REF-PRIOR-01` and `REF-PRIOR-02` proofs. |
| `internal/massive/privatefs.go`: private directory/file validation | SHA-256 `45efd341ec37e08dfc73407a69ece737c55d04a20550bf89a38f053b9e15bbab` | Rejects symlinked/nonprivate paths and requires `0700` directories and `0600` regular files. | `direct port` | Keep the behavior as a small reference-cache boundary; do not generalize it into a storage framework. | Cache mutation matrix including permission/symlink and interrupted-write cases. |
| `internal/scanner/types.go`: `MaximumTickerBytes`, `MaximumCacheBytes`, policy/readiness constants only | SHA-256 `260b2203edd281585ce95c0edc0ba8114994f67f18a92ef8b46790b710127318` | Supplies evidenced 64-byte symbols, 16 MiB caches, explicit universe policy versions, and current/stale/persistence statuses. | `behavior evidence` | Move component-specific policy/currentness meaning to the reference contract; do not port unrelated scanner types or make the reference loader a readiness owner. | Boundary/value tests and binding-assembly currentness proof. |

Current official provider documentation independently confirms the v2 endpoint
shape: Massive's
[`All Tickers`](https://massive.com/docs/rest/stocks/tickers/all-tickers),
[`Ticker Types`](https://massive.com/docs/rest/stocks/tickers/ticker-types), and
[`Daily Market Summary`](https://massive.com/docs/rest/stocks/aggregates/daily-market-summary)
documents define the exact date query, 1,000-row maximum, `next_url`, ticker
fields/type codes, grouped date path, `adjusted`, `include_otc`, result count,
symbol, close, and millisecond timestamp used here. The
[`NYSE Holidays and Trading Hours`](https://www.nyse.com/trade/hours-calendars)
page is the schedule authority. These pages were read without credentials or
live data calls.

The schedule test was executed as the isolated file pair and passed. The
universe/prior-close test files were reviewed but not executed because a Go
package-level run would compile excluded, dirty v2 orchestration sources. Their
assertions are predecessor evidence, not a claim that the full v2 package is
currently clean or passing.

**Proposed implementation whitelist:**

- `internal/scanner/schedule.go`: only the schedule artifact decoding,
  validation, prior-date selection, and session-bound behavior;
- `internal/scanner/schedule_test.go` and `internal/scanner/nyse_trading_days.json`;
- `docs/nyse-schedule-provenance.json`;
- `internal/massive/universe.go`: only `UniverseResolver`, request/pagination,
  filtering/accounting, bounded HTTP, and universe-cache behavior;
- `internal/massive/universe_test.go`: only focused reference fixtures/cases;
- `internal/massive/prior_close.go`: only grouped daily request,
  normalization, bounded HTTP, and prior-close-cache behavior;
- `internal/massive/prior_close_test.go`;
- `internal/massive/privatefs.go`; and
- from `internal/scanner/types.go`, only the evidenced ticker/cache bounds and
  eligibility/prior-close policy concepts—not the file or unrelated types as a
  direct port.

No checkpoint, recovery, owner/orchestration, WebSocket, live-provider test,
aggregate, ranking, readiness, API, UI, or version 1 source is whitelisted.
Whitelist entries are provenance and reuse authorization only. Any approved
code, test behavior, or fixture must be ported into this repository and become
repository-owned source or test data. Production builds, tests, and runtime may
not import, read, symlink, execute, vendor by filesystem path, or otherwise
depend on the version 2 checkout.

## 9. Detailed semantic inputs, outputs, and owned state

| Item | Meaning and required provenance/identity | Bounds or ownership |
| --- | --- | --- |
| Binding request | One explicit exchange-local trading date supplied by the run coordinator or replay configuration. The component validates it against the schedule and never selects it from direct operating-system wall time. Run mode is not part of market-session identity. | Immutable input; caller owns lifecycle/run-mode selection. |
| Schedule artifact | Versioned official-NYSE-derived trading-day rows, regular-close times, exceptional closures, coverage interval, source/review provenance, and content checksum, interpreted with embedded `America/New_York` tzdata. | One immutable embedded artifact; component validates it before use. |
| Universe source result | Complete paginated Massive ticker result for the requested trading date with request provenance, raw count, one closed inclusion/exclusion result per record, strict sorted unique eligible symbols, policy version, and deterministic universe identity. | At most 100 pages, 100,000 records, and 4 MiB per response page. |
| Prior-close source result | Complete grouped daily Massive result for the required prior-session date with exact adjustment/source policy, valid and attributable-invalid rows, retrieval provenance, and deterministic dataset identity. | At most 100,000 rows and 4 MiB response body. |
| Per-symbol prior-close fact | For every bound eligible symbol, exactly one of `valid(finite_positive_adjusted_close)`, `missing`, or `invalid(reason)`. No untrusted numeric value accompanies missing/invalid status. | Immutable, exact-symbol, one fact per universe member. |
| Immutable session binding | Trading date; `S`; `E`; schedule version/content identity; required prior-session date and regular close; universe policy, identity, sorted population, and accounting; prior-close policy/source, dataset identity, per-symbol facts/accounting; stable binding identity. | Constructed once; no mutable maps/slices escape. Installed and owned only by `ScannerStateEngine`. |
| Observable stale-universe result | A validated prior-date universe cache with its exact reference date and age, returned only when current-date retrieval/cache fails. It cannot be assembled into a current session binding. | Maximum age seven calendar days; observable diagnostic output only. |
| Reference caches | Date-keyed, schema/policy/source-versioned universe and prior-close artifacts containing normalized facts and provenance but no credentials. Retrieval timestamps are operational metadata and are excluded from market-data identities. | Component owns bounded disk artifacts only: `0700` directories, `0600` files, 16 MiB per file, latest 14 date files per cache class. |
| Terminal result | Either a complete immutable binding, a current fresh result with cache-persistence degradation, an observable-only stale universe result, cancellation/deadline, or a fixed global invalid/unavailable reason. | One terminal result per binding request; no pending/deferred terminal state. |

The component owns transient request/pagination/retry state and its bounded
reference-cache files. It owns no active session, canonical symbol state,
lifecycle, committed time, population evaluation, or readiness state.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `REF-SCHEDULE-01` | Load one checksummed, strict-schema, sorted, versioned schedule artifact derived from the official NYSE calendar. Validate coverage, unique dates, supported regular closes, sorted unique exceptional closures, provenance identity, and timezone availability. Reject an invalid artifact or unsupported run date globally. Replace the artifact through reviewed source control before using a date outside coverage or after an official revision; do not fetch a runtime calendar. | `DTE-SESSION-01`, `DTE-CLOCK-01`; v2 artifact/loader/provenance; official NYSE calendar. |
| `REF-SCHEDULE-02` | Validate the requested trading date, choose the immediately preceding declared completed regular session, and construct exact timezone-aware UTC `S=04:00` and `E=20:00` New York civil instants. Early regular close affects prior-session evidence but never shortens `[S,E)`. Schedule time creates no market-data/currentness evidence. | `DTE-SESSION-01` through `DTE-SESSION-04`, `DTE-CLOCK-01`, `DTE-CLOCK-04`; v2 schedule tests. |
| `REF-UNIVERSE-01` | Request `/v3/reference/tickers` for the exact run date with `active=true`, `market=stocks`, `limit=1000`, `sort=ticker`, and `order=asc`. Follow only bounded same-origin HTTPS `next_url` values that contain no credential or conflicting date, reject cycles/duplicates/incomplete pages, and filter exact record fields to active `market=stocks`, `locale=us`, type `CS` or `ADRC`. A present provider response status must be exactly `OK`; a present page count must equal the number of decoded records. Duplicate contract-bearing JSON members or ambiguous ticker/eligibility fields invalidate the complete universe rather than selecting a last value. Do not make runtime ticker-type discovery a prerequisite. | `PG-UNIVERSE-01`; official Massive ticker/type docs; adapted v2 resolver; final-review malformed-envelope finding. |
| `REF-UNIVERSE-02` | Sort eligible exact symbols ascending, account every decoded reference record once as eligible or one of `inactive`, `wrong_market`, `wrong_locale`, `ineligible_type`, and compute an order-independent universe identity. Duplicate or invalid ticker identity makes the complete universe result globally unusable rather than guessed. An empty eligible result is globally invalid. | `PG-UNIVERSE-01`, `PG-OBS-01`; v2 accounting/cache tests. |
| `REF-PRIOR-01` | Request the grouped U.S. stocks daily summary for the schedule-derived prior date with `adjusted=true` and `include_otc=false`. Require provider response status exactly `OK`, a structurally unambiguous response envelope, explicit adjusted confirmation, exact result count, bounded rows/body, and request/date/source provenance. Duplicate envelope members are global ambiguity. A structurally valid zero-row `OK` response is complete and maps all eligible symbols to missing; an HTTP-success response carrying a non-`OK` or missing status cannot establish empty or value evidence. | `PG-REFERENCE-01`, `PG-RANK-01`, `DTE-SESSION-01`; official Massive daily-summary docs; adapted v2 resolver; final-review malformed-envelope finding. |
| `REF-PRIOR-02` | Map exact case-sensitive rows to the bound universe. A finite positive close with provider millisecond time identifying the required date is valid. An attributable nonpositive/nonfinite/wrong-date/duplicate value makes only that symbol invalid. An absent symbol is missing. A row containing exactly one valid bound symbol but duplicated or ambiguous close/time members is attributable and makes only that symbol invalid. A row whose symbol member is absent, malformed, or duplicated is unattributable diagnostic evidence and cannot fabricate a value. Other unattributable malformed extra rows remain bounded diagnostics; envelope/source/date ambiguity remains global. No JSON decoder may silently select the last of duplicate contract-bearing members. | `PG-REFERENCE-01`, `PG-RANK-01`, `PG-OBS-01`, `DTE-REJECT-02`; Phase 1 symbol-local invariant; v2 malformed-row regression evidence; final-review ambiguity finding. |
| `REF-CACHE-01` | Accept a universe cache into a current binding only when its schema/policy/provenance and reference date exactly match the run date. A prior-date universe cache is observable-only. Accept a prior-close cache only for the exact required prior date and exact adjusted source/policy. Strictly reject unknown or duplicate fields, trailing data, count/order/identity mismatch, oversized files, and unsupported schedule dates. Persisted timestamps use the canonical zero-offset encoding. Universe retrieval time must be possible for its reference date. Prior-close retrieval time must be at or after the schedule-derived prior regular close and not later than the injected current time. Provider credential/configuration failure does not bypass an otherwise valid permitted cache. | `PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-RANK-05`, `PG-OBS-03`, `DTE-SESSION-03`, `LIFE-INIT-05`; v2 cache matrices; final-review currentness findings. |
| `REF-CACHE-02` | Treat cache preparation, reading, publication, sync, and pruning as subordinate persistence operations. A fresh retrieval may proceed when cache preparation is unavailable; fresh validated facts remain usable after any cache preparation/publication/pruning failure with one bounded persistence reason. Write usable caches through a private temporary file, file sync, atomic rename, and directory sync. Reject a symlink in any component-owned path segment and reject nonprivate permissions; never follow an ancestor symlink to read or write cache data. Prune each cache class independently to its latest 14 dates, remove or otherwise strictly bound component-owned interrupted temporary files, and enumerate cache directories through a fixed entry bound. Never persist credentials. | `PG-OBS-03`, `ARCH-OWN-02`; v2 private filesystem and universe persistence behavior; prior-close adaptation; final-review cache-safety findings. |
| `REF-ID-01` | Compute versioned SHA-256 identities for the universe, prior-close dataset, and complete binding using a fixed canonical UTF-8 encoding: fixed field order and names, no insignificant whitespace, UTC RFC3339 nanosecond timestamps, lowercase hexadecimal IEEE-754 binary64 close bits, and exact-symbol-sorted arrays. Prefix lowercase hex digests with their identity schema (`universe-v1:`, `prior-close-v1:`, `session-binding-v1:`). Exclude retrieval time, request order, pagination, retry, cache path, and credentials. Include every binding-defining date, bound, schedule identity, policy/source parameter, universe identity, and per-symbol prior-close status/value so any semantic change changes the binding identifier. | `DTE-SESSION-02`, `LIFE-MODEL-01`, `LIFE-INIT-02`, `LIFE-REPLAY-01`; SHA-256 schedule precedent and deterministic-identity invariant. |
| `REF-FAIL-01` | Bound each reference acquisition to two minutes, each HTTP attempt to 15 seconds, each body to 4 MiB, and each request to three attempts. Retry transport/read failures, HTTP 408/429, and 5xx with cancellation-aware exponential jitter capped by the v2 250/500 ms attempt windows; do not retry permanent HTTP/provider status, credential/configuration, or semantic validation failures. Cancellation or the operation deadline stops further pages and attempts. Universe pages are sequential; prior close is one grouped request, so no provider-request concurrency is required. Return fixed terminal global/per-symbol/cache-persistence outcomes plus immutable scalar request/page/attempt counts; never retain URLs, bodies, credentials, symbols, or unbounded error text as diagnostics. | `LIFE-INIT-05`, `ARCH-OWN-02`; v2 request bounds and retry tests; final-review diagnostic/proof findings. |
| `REF-BIND-01` | Assemble a binding only after independently revalidating the supplied schedule facts against the accepted schedule selection for the exact trading date; a caller-owned value copy is not provenance. The revalidated facts, current-run-date universe, and exact-date/policy prior-close dataset must all agree. Symbol-local missing/invalid prior closes remain inside a globally coherent binding. Publish no mutable reference containers. Return the completed binding as one fact for ordered engine installation; this component does not install it or decide lifecycle/readiness. | `PG-OBS-01`, `PG-OBS-03`, `ARCH-OWN-01`, `ARCH-OWN-02`, `ARCH-OWN-04`, `DTE-SESSION-01`, `DTE-SESSION-02`, `LIFE-INIT-02` through `LIFE-INIT-05`, `LIFE-REPLAY-01`; final-review binding finding. |

The canonical identity field sequences are fixed as follows:

```text
universe-v1 = [schema, trading_date, eligibility_policy, sorted_symbols]

prior-close-v1 = [schema, prior_session_date, prior_close_policy,
                  adjusted, include_otc, locale, market,
                  sorted_per_universe_symbol(status, close_bits_or_empty)]

session-binding-v1 = [schema, trading_date, S, E,
                      schedule_schema, schedule_version,
                      schedule_artifact_sha256,
                      prior_session_date, prior_regular_close,
                      universe_identity, prior_close_identity]
```

Each sequence uses the exact encoding rules in `REF-ID-01`. Universe filter
diagnostics, provider rows outside the bound population, retrieval/persistence
metadata, and run mode do not change market-session identity.

### 10.1 Provider structural-classification precedence

Provider JSON is classified before normalized facts are constructed:

1. A malformed top-level JSON value, duplicate contract-bearing envelope
   member, non-`OK` required status, adjustment ambiguity, result-count
   mismatch, or non-array result is global source ambiguity.
2. A universe record with duplicate contract-bearing members or ambiguous
   ticker/eligibility fields makes the complete universe unusable because the
   record cannot participate exactly once in the closed filter accounting.
3. A prior-close row is attributable only when it contains exactly one valid
   exact symbol. Ambiguous value/date fields on an attributable row make that
   symbol `invalid`; they do not fail unrelated symbols.
4. A prior-close row without exactly one valid exact symbol is an
   unattributable diagnostic row. It contributes no value and does not change
   any bound symbol from `missing`.

Unknown provider fields remain forward-compatible when they do not duplicate
or contradict a contract-bearing member. Duplicate-member detection is a
small trust-boundary validator, not a generalized JSON framework.

### 10.2 Acquisition and cache decision order

For each reference class, source and cache evidence are independent inputs to
one terminal decision:

1. validate schedule and dependency facts;
2. inspect the configured cache boundary when it can be opened safely, keeping
   cache unavailability as a bounded persistence/cache diagnostic;
3. attempt fresh retrieval only when provider configuration is valid;
4. if fresh retrieval succeeds, return the fresh facts regardless of cache
   preparation/publication/pruning outcome;
5. otherwise, use only the cache outcome already permitted by `REF-CACHE-01`—
   current same-date universe, observable-only prior-date universe, or exact-
   date/policy prior close; and
6. return global failure only when neither fresh nor permitted cached evidence
   can support the requested result.

This order does not turn a prior-date universe into current evidence and does
not permit a non-exact prior-close cache. It prevents cache storage or provider
configuration from invalidating evidence that is independently sufficient.

## 11. Failure and terminal behavior

Global terminal failures are limited to conditions that prevent a coherent
binding: invalid/unsupported schedule; unavailable current-date universe;
unbounded, incomplete, duplicate-identity, or structurally ambiguous universe;
unavailable exact-date prior-close dataset; ambiguous prior-close envelope or
source adjustment/date; invalid cache with no successful source; canceled
request; or exhausted deadline/retry budget. The result names one fixed bounded
reason and cannot be installed as a current binding.

A cache path, permission, preparation, publication, sync, or pruning failure is
not global when fresh reference facts validate. A missing or invalid provider
credential is a permanent fresh-source configuration outcome and is not
retried, but it does not prevent consideration of an otherwise valid permitted
cache. A provider HTTP success carrying a non-`OK` required status is semantic
failure, not successful empty data.

Prior-close row failures with an exact symbol are local. The symbol receives
`invalid`; absence receives `missing`; other valid symbols remain valid. A
successful zero-row prior-close dataset therefore yields a coherent binding
with every eligible symbol missing, not a fabricated close or global ambiguity.

A validated prior-date universe cache is a terminal observable-only result when
no current universe exists. It preserves process diagnostics but cannot form a
current binding. A cache write, sync, rename, directory-sync, or prune failure
after fresh reference validation returns the fresh facts plus a persistence
reason; it does not invalidate their market meaning.

Cancellation and deadline exhaustion stop further pages/attempts and return
exactly one terminal result. The component has no deferred state, background
retry owner, or event capable of changing an already returned binding.

Final binding assembly rederives the expected schedule facts for the explicit
trading date and compares the complete fact value before hashing or returning a
binding. Mutation or substitution of `S`, `E`, prior-session date/close,
schedule schema/version, or artifact identity is therefore a global binding
failure even when all values are syntactically well formed.

## 12. Accounting and observability

Universe acquisition has one closed record identity:

```text
raw_reference_records
  = eligible_records
  + inactive_records
  + wrong_market_records
  + wrong_locale_records
  + ineligible_type_records
```

The precedence shown is normative and makes each decoded record mutually
exclusive. Invalid/duplicate ticker identity invalidates the whole result and
is reported as a terminal reason rather than added to a supposedly complete
population.

The binding has one exact prior-close partition:

```text
universe_total
  = valid_prior_close
  + missing_prior_close
  + invalid_prior_close

invalid_or_missing_prior_close
  = missing_prior_close
  + invalid_prior_close
```

This is the reference-data portion of `PG-OBS-01`. Mark, no-print,
qualification, feature, T/Q, and ranking dimensions do not exist in this
component. Provider rows outside the bound universe, unattributable malformed
rows, retries, pagination, cache persistence, and stale-cache age are bounded
diagnostic dimensions, not extra primary population bins.

Observable binding/reference facts include artifact/policy/schema versions,
source dates, identities, raw/accepted/rejected/missing/invalid counts, cache
source and age, request/page/attempt counts, and one fixed terminal or
persistence reason. Symbols, URLs, raw bodies, credentials, and unbounded error
strings are not metric labels or retained diagnostic sets.

The immutable terminal diagnostic shape contains only the applicable fields:

```text
source = fresh | current_same_date_cache | observable_prior_date_cache | none
terminal_reason = one fixed component reason | none
persistence_reason = cache_prepare_failed | cache_publication_failed |
                     cache_prune_failed | none
request_count >= 0
page_count >= 0
attempt_count >= 0
unattributable_prior_rows >= 0
```

Counts include work actually attempted before cancellation or terminal failure.
They are returned with success or failure through a typed immutable result; a
test-only fake-server counter is not the observable contract. Wrapped
implementation errors may be logged at the immediate call site under the later
operations policy, but they do not replace the fixed terminal reason or enter
retained binding identity.

## 13. Simplicity and boundedness

- Mutable owners/state representations: one transient resolver operation and
  two bounded date-keyed cache classes. The immutable binding is handed to the
  engine; no reference owner remains active afterward.
- Schedule: one embedded artifact, strict checksum/schema/provenance, no
  runtime service, and no weekday/holiday inference.
- Universe: sequential provider pagination, at most 100 pages/100,000 records,
  4 MiB per page, no worker pool, and no dynamic ticker-type discovery call.
- Prior close: one grouped daily request, at most 100,000 rows/4 MiB, not one
  request or worker per symbol.
- Retries: three 15-second attempts per request inside one two-minute operation;
  cancellation-aware bounded jitter only for retryable failures.
- Caches: 16 MiB per file, 14 dates per class, strict normalized content,
  private atomic files, no database, generalized repository, or checkpoint
  coupling. Directory traversal is incremental and stops at 64 entries per
  cache class. Component-owned interrupted temporary files older than the
  two-minute maximum operation duration are removed during the next safe
  preparation/prune pass; younger temporary files and other unexpected entries
  consume the bound. Exceeding the bound makes persistence unavailable rather
  than causing an unbounded scan or deletion of unrecognized user-owned
  content.
- Diagnostics: fixed reason and rejection vocabularies plus scalar counts; no
  raw-payload retention or symbol-cardinality metrics.
- V2 mechanisms retained: reviewed schedule artifact, exact-date ticker query,
  closed filter accounting, grouped adjusted daily endpoint, response/page
  bounds, retry classification, and private atomic cache mechanics.
- V2 mechanisms rejected or changed: runtime ticker-type discovery adds a
  failure without changing the approved fixed policy; mutable result maps could
  become competing state; stale universe fallback cannot be current; grouped
  prior-close row invalidity cannot fail unrelated symbols; prior-cache write
  failure cannot erase fresh valid facts; no standalone binding identity exists.
- Other credible alternatives: a runtime calendar API adds availability and
  currentness ambiguity; per-symbol prior-close calls add thousands of requests;
  a database or generalized cache adds ownership without product benefit. The
  selected path is the smallest complete deterministic reference path.

No hypothetical abstraction, event bus/plugin framework, database/service
split, or unevidenced provider edge-case mechanism is added.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Weekend and exchange holiday before a run date | V2 schedule test and official NYSE calendar | Select the immediately preceding declared trading session, never weekday arithmetic. | Schedule-selection table in `REF-SCHEDULE-02` proof. |
| Prior regular session is an early close | V2 artifact/test and official NYSE early-close rows | Retain the 13:00 regular close as prior-session evidence while current scanner bounds remain 04:00–20:00. | Schedule-selection table in `REF-SCHEDULE-02` proof. |
| DST offset changes between trading dates | V2 test around 2026-03-06/09; timezone invariant | 04:00 New York maps to the correct UTC instant on each date; no fixed UTC offset. | UTC-boundary cases in `REF-SCHEDULE-02` proof. |
| Schedule checksum/schema/order/coverage corruption | V2 strict loader/checksum and artifact provenance | Reject globally before reference requests or binding construction. | Artifact mutation matrix in `REF-SCHEDULE-01` proof. |
| Ticker type response can be object or array | V2 test and current Massive sample/docs | Do not depend on this endpoint at runtime; fixed `CS`/`ADRC` mapping is policy-versioned and fixture-proved from ticker rows. | Universe mapping fixture. |
| Pagination cycle, changed origin/date, embedded credential, redirect, duplicate ticker | V2 resolver/tests | Reject the incomplete/ambiguous universe globally and publish no cache. | Universe pagination fixture. |
| Reference response read fails transiently | V2 retry test | Retry within the three-attempt/deadline budget, then return one terminal result. | Bounded-failure proof. |
| Universe cache is stale, misdated, future-dated, unsorted, truncated, oversized, wrong policy, or count-inconsistent | V2 strict cache mutation matrix | Reject invalid cache; a valid prior-date cache is observable-only and never current. | Cache acceptance matrix. |
| Exact symbols differ only by case | V2 prior-close fixture (`AAA`, `aaa`) | Preserve both exact identities and map their closes independently. | Prior-close mapping fixture. |
| Prior-close row is duplicate, nonpositive/nonfinite, or names the wrong date | V2 rejection cases plus Phase 1 symbol-local containment | Mark only the attributable bound symbol invalid; do not choose a value or reject unrelated valid symbols. | Mixed valid/invalid prior-close fixture. |
| Eligible symbol is absent or grouped result is valid empty | `PG-REFERENCE-01`/`PG-RANK-01` invariant; v2 missing case | Mark absent symbols missing; accept valid empty as all missing; never substitute another price/date. | Prior-close containment/accounting proof. |
| Fresh facts validate but cache persistence fails | V2 universe behavior; Phase 1 currentness invariant | Return fresh facts with bounded persistence degradation; do not fall back to stale facts or fail the binding solely for persistence. | Cache persistence scenario. |
| Same semantic data arrives in different page/row order or after retry | Stable-identity invariant; v2 sorting behavior | Produce identical universe, prior-close, and binding identities. | Deterministic identity permutation proof. |
| HTTP success carries provider `status=ERROR`, inconsistent optional page count, or duplicate envelope member | Official Massive response schemas; final component review demonstrated last-value/status omission acceptance | Reject globally; never convert the response into a complete universe, values, or all-symbols-missing evidence. | Corrective provider structural-classification table. |
| Prior-close row has one exact symbol but duplicate value/time member, or has a duplicate/malformed symbol member | Phase 1 local-containment invariant; final-review ambiguity analysis | One exact symbol makes ambiguous value/time local invalid; ambiguous attribution remains diagnostic and fabricates no value. | Corrective prior-row containment table. |
| Cache preparation is unavailable while fresh retrieval succeeds | Existing fresh-fact persistence invariant; final component review found preparation preceding retrieval | Return fresh facts plus `cache_prepare_failed`; do not make persistence an acquisition prerequisite. | Corrective cache independence proof. |
| Exact prior-close cache exists while provider credential/configuration is unavailable | `PG-REFERENCE-01` exact-cache equivalence; final component review | Accept the validated exact-date/policy cache without a provider request; do not retry configuration failure. | Corrective cache/source-decision proof. |
| Prior-close cache claims retrieval before prior regular close or uses a noncanonical offset | Schedule-completion invariant and `DTE-SESSION-03`; final component review | Reject as impossible/noncanonical persisted evidence. | Corrective cache currentness matrix. |
| Configured cache path has a symlink in an ancestor component | `REF-CACHE-02` no-symlink invariant; final component review | Perform no read/write through the link; preserve fresh facts with a persistence diagnostic when available. | Corrective rooted-filesystem proof. |
| Interrupted temporary files or unexpected directory entries accumulate | Existing interrupted-write fixture plus bounded-retention invariant | Remove only old recognized component temporaries, stop incremental enumeration at 64 entries, and degrade persistence without unbounded work. | Corrective retention/pruning proof. |
| Caller mutates a copied schedule fact before final assembly | `DTE-SESSION-01`, `DTE-SESSION-02`; final component review | Revalidate against the accepted schedule and reject before hashing or returning a binding. | Corrective binding mutation table. |

## 15. Primary proof allocation

| Requirement | One primary proof | Distinct boundary proved | Approved fixture/evidence | Allocated slice |
| --- | --- | --- | --- | --- |
| `REF-SCHEDULE-01` | Schedule artifact mutation/provenance test | Only a strict, checksummed, covered, official-source artifact is usable. | V2 artifact/provenance/loader; official NYSE calendar. | `S1` |
| `REF-SCHEDULE-02` | Table-driven civil-time session-selection test | Weekend, holiday, early-close, DST, 04:00 inclusion, 20:00 exclusion, and prior-session semantics. | Adapted v2 schedule tests. | `S1` |
| `REF-UNIVERSE-01` | Fake-provider pagination/normalization fixture | Exact request, page completion/security, eligibility mapping, and terminal invalid cases. | Adapted v2 TLS fake-provider cases; official Massive docs. | `S2` |
| `REF-UNIVERSE-02` | Universe accounting and permutation test | Closed exclusion identity, exact sorted population, and order-independent universe identity. | Adapted v2 filter counts plus generated page permutations. | `S2` |
| `REF-PRIOR-01` | Grouped daily normalization fixture | Exact adjusted/date/source request and complete valid/empty dataset mapping. | Adapted v2 TLS fixture; official Massive docs. | `S3` |
| `REF-PRIOR-02` | Mixed valid/missing/invalid symbol containment test | Exact per-symbol partition and no global/fabricated consequence from attributable bad rows. | Adapted v2 duplicate/zero/wrong-date/missing cases plus Phase 1 invariant. | `S3` |
| `REF-CACHE-01` | Strict universe/prior-cache acceptance matrix | Same-date current acceptance, prior-date universe observable-only behavior, exact prior-date close acceptance, and corruption rejection. | Adapted v2 cache matrices. | `S3` |
| `REF-CACHE-02` | Private atomic persistence/failure test | Permissions, symlink rejection, interrupted write safety, pruning bound, credential exclusion, and fresh-fact survival after persistence failure. | V2 privatefs/universe behavior; adapted prior-close case. | `S3` |
| `REF-ID-01` | Canonical identity golden/permutation test | Stable digest across order/retry/cache/fresh source and digest change for every binding-defining semantic change. | SHA-256/canonical-encoding invariant; schedule hash precedent. | `S3` |
| `REF-FAIL-01` | Fake-clock/fake-transport retry/deadline test | Exact retry classes, attempts, timeout/cancellation, no page concurrency, and one terminal result. | Adapted v2 response-read/status behavior. | `S3` |
| `REF-BIND-01` | Complete binding assembly and immutability test | Only coherent current facts assemble; symbol-local prior-close failures remain contained; returned data cannot be mutated into competing state. | Outputs of `S1`–`S3`; Phase 1 ownership invariant. | `S3` |

The table above records the original approved primary allocation. Passing tests
did not completely establish `REF-UNIVERSE-01`, `REF-PRIOR-01`,
`REF-PRIOR-02`, `REF-CACHE-01`, `REF-CACHE-02`, `REF-FAIL-01`, or
`REF-BIND-01`; their corrective primary proofs are allocated exactly once
below. `REF-SCHEDULE-01`, `REF-SCHEDULE-02`, `REF-UNIVERSE-02`, and
`REF-ID-01` remain proved and receive no duplicate corrective proof.

| Corrective proof | Requirements completed | Distinct boundary proved | Allocated slice |
| --- | --- | --- | --- |
| Provider structural-classification and containment table | `REF-UNIVERSE-01`, `REF-PRIOR-01`, `REF-PRIOR-02` | Required/present status, optional count, duplicate envelope members, universe-record ambiguity, attributable prior-row ambiguity, unattributable rows, and valid `OK` empty results have exact global/local consequences without fabricated values. | `C1-R1` |
| Both-resolver bounded acquisition and terminal-diagnostics table | `REF-FAIL-01` | Universe and prior-close paths independently prove retryable transport/read/408/429/5xx behavior, permanent status/configuration no-retry, three attempts, 15-second attempt timeout, two-minute operation deadline, cancellation, sequential pages, and returned scalar diagnostics without real waiting. | `C1-R1` |
| Both-cache source/currentness and corruption matrix | `REF-CACHE-01` | Fresh/cache decision order, missing-credential exact-prior fallback, same-date versus observable-only universe behavior, prior retrieval not before regular close, canonical timestamp encoding, and strict duplicate/unknown/count/order/identity rejection are proved for both cache classes. | `C1-R2` |
| Rooted private persistence and bounded-retention table | `REF-CACHE-02` | Cache preparation/publication/prune failures preserve fresh facts; ancestor/file symlinks and nonprivate paths are not followed; atomic replacement preserves the last complete file; the two classes prune independently; temporary files and directory enumeration stay bounded; credentials are absent. | `C1-R2` |
| Schedule-revalidated complete binding table | `REF-BIND-01` | Valid facts assemble with complete immutable contents; mutation of every schedule-defining field, population/identity mismatch, or usable numeric missing/invalid close is rejected before a binding escapes. | `C1-R3` |

No live-provider call is a primary proof for this component. Current provider
documentation plus deterministic fake-provider fixtures establish the mapping;
a later authorized observation is needed only if implementation reveals an
unproved response shape or entitlement behavior.

The eleven original requirement allocations were intended to be implemented as
seven compact table-driven test groups, not eleven test files or duplicated
suites. The corrective rows above complete those same component-level groups
where the first implementation proved only one resolver/cache class or omitted
the material trust-boundary case:

| Lean test group | Requirements covered | Minimum distinct behavior |
| --- | --- | --- |
| Schedule artifact | `REF-SCHEDULE-01` | Valid reviewed artifact loads; checksum/schema corruption and unsupported coverage fail closed. |
| Session selection | `REF-SCHEDULE-02` | Weekend, holiday, early-close predecessor, DST conversion, 04:00 inclusion, and 20:00 exclusion. |
| Universe resolution | `REF-UNIVERSE-01`, `REF-UNIVERSE-02` | One paginated fake-provider path proves exact query, `CS`/`ADRC` filtering, exclusion accounting, deterministic population, documented status/optional-count handling, duplicate-member rejection, and only the material partial/ambiguous-result failures. |
| Prior-close resolution | `REF-PRIOR-01`, `REF-PRIOR-02` | One grouped-response fixture proves required `OK` status, valid, missing, symbol-local invalid, duplicate/wrong-date/ambiguous-row containment, unattributable diagnostics, and valid-empty outcomes without globalizing attributable row failure. |
| Cache policy and persistence | `REF-CACHE-01`, `REF-CACHE-02` | For both cache classes, exact-date acceptance, stale-universe observable-only behavior, exact-prior fallback without provider configuration, strict timestamp/content rejection, rooted private atomic replacement, bounded temporary/pruning behavior, and fresh-fact survival after every persistence-stage failure. |
| Identity and complete binding | `REF-ID-01`, `REF-BIND-01` | The main component happy path revalidates schedule provenance and constructs the exact immutable binding; input order/retry/cache source does not change identity, while each semantic input change does; mutated schedule facts cannot assemble. |
| Bounded failure | `REF-FAIL-01` | Both resolvers use fake transport/clock/policy seams to prove transient retry then success, permanent failure without retry, exact attempt/operation deadlines, cancellation, sequential pagination, and one typed terminal result without real waiting. |

Do not add live-provider tests, engine/ranking/checkpoint/API/UI/replay scenarios,
tests for private helpers, broad JSON mutation matrices, fuzz/property-test
infrastructure, 100,000-record load tests, duplicated test layers, or separate
cases for every HTTP status. A second layer is added only if it proves a new
cross-component boundary approved later.

## 16. Sequential implementation-slice plan

| Slice | Coherent outcome | Requirement IDs and primary proofs | Dependencies/entry state | Allowed ownership or files/packages | Approved v2 whitelist/fixtures | Owner-review artifact | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `S1` | A validated deterministic exchange schedule produces the requested trading date's exact scanner bounds and required prior-session facts. | `REF-SCHEDULE-01`, `REF-SCHEDULE-02`; artifact and civil-time proofs. | Approved Phase 1 contracts; no prior component implementation. | New repository's reference/session boundary and schedule artifact only; no engine lifecycle or provider code. | V2 schedule loader/test/artifact/provenance entries from Section 8. | Schedule behavior, artifact provenance/hash, proof results, deviations, and confirmation that `S2` inputs remain valid. | Universe, prior close, caches beyond schedule artifact, complete binding identity, and engine installation remain unavailable. |
| `S2` | A bounded exact-date Massive reference path produces an immutable eligible universe with closed exclusion accounting, deterministic identity, and current versus observable-only cache meaning. | `REF-UNIVERSE-01`, `REF-UNIVERSE-02`; fake-provider and accounting/identity proofs. | Accepted `S1` schedule validation/date facts. | Reference-provider universe adapter, normalized universe result, universe cache, and focused fixtures only. | V2 universe/privatefs sources and focused test cases; selected constants evidence. | Exact request/filter/accounting/cache behavior, proof results, deviations, and confirmation that `S3` can consume the immutable universe. | Prior-close retrieval, full binding identity/assembly, engine installation, and readiness remain unavailable. |
| `S3` | Exact adjusted prior closes and all reference facts assemble into one immutable deterministic binding with symbol-local containment, strict caches, and bounded terminal behavior. | `REF-PRIOR-01`, `REF-PRIOR-02`, `REF-CACHE-01`, `REF-CACHE-02`, `REF-ID-01`, `REF-FAIL-01`, `REF-BIND-01`; proofs from Section 15. | Accepted `S1` schedule facts and `S2` immutable universe interface. | Prior-close adapter/cache, canonical identity encoder, immutable binding assembler, and focused component tests only. | V2 prior-close/privatefs sources and focused test cases; exact whitelist in Section 8. | Complete binding behavior/accounting/identity, all allocated proof results, deviations, and final component-review readiness. | Engine lifecycle/installation, checkpoints, aggregate/replay execution, readiness/API/UI, and all later components remain unavailable. |

Slice rules from the mandatory process apply: one assignment and owner review at
a time; later slices extend earlier interfaces; no slice introduces temporary
state ownership or unused scaffolding. Interaction requirements are allocated
to `S3`, when the complete binding exists.

### 16.1 Corrective implementation-slice plan

The original `S1`–`S3` rows remain the delivery record. The following slices
are the only authorized shape of corrective implementation after this revision
receives owner contract/test/slice-plan approval:

| Slice | Coherent outcome | Requirement corrections and primary proofs | Dependencies/entry state | Allowed ownership or files/packages | Owner-review artifact | Explicitly deferred behavior |
| --- | --- | --- | --- | --- | --- | --- |
| `C1-R1` | Both provider paths accept only structurally unambiguous documented success evidence and return bounded acquisition diagnostics under the exact retry/deadline policy. | `REF-UNIVERSE-01`, `REF-PRIOR-01`, `REF-PRIOR-02`, `REF-FAIL-01`; the two corrective provider/acquisition proofs in Section 15. | Accepted `S1`; existing S2/S3 provider implementations as the review target. | `internal/reference/universe.go`, `prior_close.go`, their focused tests, and the smallest shared trust-boundary JSON/diagnostic helper inside `internal/reference`; no cache filesystem or binding changes. | Accepted/rejected response table, local/global containment, both-resolver retry/deadline/diagnostic results, deviations, and confirmation that `C1-R2` can consume the normalized facts and typed outcomes. | Cache path/currentness/persistence corrections and final binding revalidation remain unchanged until later slices. |
| `C1-R2` | Fresh and cached facts obey one independent source-decision policy while private filesystem operations, cache time, atomic replacement, and retained artifacts remain strict and bounded for both cache classes. | `REF-CACHE-01`, `REF-CACHE-02`; the two corrective cache proofs in Section 15. | Owner-accepted `C1-R1` normalized facts, fixed terminal reasons, and diagnostic shape. | `internal/reference/universe.go`, `universe_cache.go`, `prior_close.go`, `prior_close_cache.go`, the shared bounded cache helper only as required, and focused cache tests; no binding changes. | Fresh/cache decision table, retrieval-time matrix, ancestor-symlink and permissions results, atomic/interrupted-write result, independent pruning/temporary bounds, credential exclusion, deviations, and confirmation that `C1-R3` receives current validated immutable reference facts. | Binding assembly and component freeze remain unavailable. |
| `C1-R3` | Final assembly independently revalidates schedule provenance and returns the complete immutable deterministic component-1 binding or one bounded failure. | `REF-BIND-01`; corrective binding proof in Section 15 plus rerun of `REF-ID-01` golden/permutation proof. | Owner-accepted `C1-R1` and `C1-R2`; accepted `S1` schedule interface. | `internal/reference/binding.go`, `binding_test.go`, and only the minimum `internal/session` validation seam if needed; no schedule semantics, provider, cache, engine, or component-2 changes. | Complete-field and schedule-mutation table, identity/immutability/no-usable-close results, full component verification, deviations, and final-review request. | Engine installation, lifecycle, readiness, ranking, checkpoints, replay execution, API/UI, and all component-2 implementation remain unavailable. |

These slices are sequential, not parallel. `C1-R2` consumes the terminal and
diagnostic boundary fixed by `C1-R1`; `C1-R3` consumes the validated reference
facts fixed by both. Implementing them concurrently would permit overlapping
edits to the resolver interfaces and would violate the repository rule that at
most one implementation slice is active. Parallel read-only review is allowed,
but each implementation assignment starts only after owner acceptance of the
preceding slice.

## 17. Implementation discretion

Implementation may choose private helper/type names, exact package/file layout
inside the approved reference boundary, HTTP client interfaces, fake-provider
fixture encoding, error wrapping, cache JSON field layout, and retry-jitter
generator. It may choose an equivalent immutable representation and equivalent
canonical encoder implementation only if the exact Section 10 identity bytes
and digests remain unchanged. It may tighten internal bounds without changing
observable semantics, but relaxing a contract bound requires owner review.

The exact v2 adaptations required for correctness are not discretionary:
remove dynamic type-discovery as a binding prerequisite; keep prior-date
universe observable-only; contain attributable prior-close invalidity by exact
symbol; preserve fresh facts on cache-persistence failure; return immutable
containers; and implement the specified deterministic identities. The
corrective revision additionally makes provider-status and duplicate-member
classification, cache/source independence, prior-close retrieval-time bounds,
ancestor-symlink rejection, bounded temporary retention/entry enumeration,
typed scalar diagnostics, and schedule-revalidated assembly non-discretionary.

**Prohibited changes**

- Deriving a trading date from UTC truncation or direct product-code wall time.
- Weekday arithmetic, fixed UTC session offsets, or early-close shortening of
  the scanner session.
- Expanding eligibility beyond exact `CS`/`ADRC` or substituting another close.
- Treating a prior-date universe as current or a non-exact prior-close cache as
  usable.
- Treating a non-`OK` provider status, duplicate contract-bearing envelope
  member, or ambiguous universe record as successful evidence.
- Globalizing an attributable symbol prior-close failure or accepting a
  numeric value for missing/invalid status.
- Making fresh retrieval depend on writable cache storage, skipping an allowed
  exact cache solely because provider configuration is unavailable, following
  any cache-path symlink, or scanning retained cache entries without the fixed
  bound.
- Assembling a binding from merely syntactic or caller-asserted schedule facts
  without revalidation against the accepted schedule.
- Letting reference adapters install engine state, compute ranking/readiness,
  or retain a competing mutable binding.
- Introducing a database, service, generic provider/cache framework, runtime
  calendar dependency, per-symbol prior-close fan-out, or live provider call.
- Inspecting or reusing any v2 source outside the approved Section 8 whitelist.
- Importing, reading, symlinking, executing, or otherwise depending on the v2
  checkout from this repository's build, tests, or runtime; approved reuse must
  be copied or reimplemented here with provenance retained in this spec.

**Stop/escalation conditions**

- Current official provider documentation or an approved fixture conflicts
  with the endpoint/field/date semantics above.
- A provider response shape cannot be classified without opening an unapproved
  v2 area or inventing speculative behavior.
- The official NYSE schedule changes inside supported coverage or lacks the
  required prior/run date.
- A downstream approved component requires a binding field or identity change.
- The closed universe/prior-close accounting or identity permutation proof
  fails.
- Work would require lifecycle, canonical-state, checkpoint, readiness, API,
  UI, live credentials, or another excluded owner.

## 18. Completed-contract acceptance checklist

The checked items below record the original contract approval; final component
review subsequently reopened the listed corrective boundaries. They are not a
claim that the corrective revision or implementation is accepted.

- [x] Owner-approved boundary and reconnaissance scope are recorded.
- [x] Version 2 inspection stayed inside that scope, or expansions received
      explicit owner approval.
- [x] Exact version 2 sources, decisions, adaptations, fixtures, and proof
      obligations are recorded.
- [x] Inputs, outputs, owned state, bounds, required behavior, and terminal
      outcomes are complete without duplicating Phase 1.
- [x] Primary accounting identities and overlapping dimensions are explicit.
- [x] Every nontrivial edge case has evidence or owner approval.
- [x] Every component requirement has one primary proof; duplicate layers name
      a distinct boundary.
- [x] Every requirement and primary proof is allocated to exactly one
      implementation slice.
- [x] Every slice has one coherent outcome, precise scope, explicit deferred
      behavior, a review artifact, and an owner-review stop.
- [x] Slice ordering is acyclic; no slice requires an interface or behavior
      defined only by a later slice.
- [x] Later slices extend rather than replace earlier ownership and behavior.
- [x] Implementation discretion, prohibited changes, and escalation conditions
      are clear enough for one bounded slice assignment at a time.
- [x] The drift audit below has no unresolved substantive **yes**.
- [x] Exact version 2 implementation and fixture whitelist received owner
      contract/reuse/test/slice-plan approval.

### 18.1 Corrective-revision approval and completion checklist

- [x] Owner approves the corrective provider/cache/binding boundary and the
      exact three-slice sequence in Section 16.1.
- [x] `C1-R1` passes its provider-classification and both-resolver bounded-
      acquisition proofs and receives owner slice acceptance.
- [x] `C1-R2` passes its both-cache source/currentness and rooted-private-
      persistence proofs and receives owner slice acceptance.
- [x] `C1-R3` passes its schedule-revalidated binding proof, reruns the identity
      proof, and receives owner slice acceptance.
- [x] All original S1–S3 primary proofs and repository-wide verification pass
      against the corrected implementation.
- [x] Repeated independent final component review finds no blocking
      conformance, proof, boundedness, predecessor-coupling, or scope defect.
- [ ] Owner separately freezes Component 1 if desired; this review did not mark
      the component frozen.
- [ ] Owner separately authorizes any Component 2 implementation slice.

## 19. Drift audit

| Question | Yes/No | Evidence or owner resolution |
| --- | --- | --- |
| Did this introduce a new product rule? | No | Detailed decisions implement the exact Phase 1 universe, prior-close, session, cache-currentness, and binding outcomes. |
| Did this introduce another mutable state owner, watermark, or evaluator? | No | The component owns transient I/O/cache mechanics only and returns one immutable fact to the engine. |
| Did this make aggregate ranking/readiness depend on T/Q? | No | T/Q is absent from the component and its inputs/outputs. |
| Did this change session, event-time, half-open-window, correction, or committed-watermark semantics? | No | It implements schedule-derived 04:00–20:00 `[S,E)` and owns no market-event or watermark behavior. |
| Did this add behavior without component-local evidence or explicit approval? | No | Provider mechanics come from current official docs and scoped v2 evidence; per-symbol containment and deterministic identity follow approved Phase 1 invariants. |
| Did this duplicate an existing responsibility or Phase 1 contract? | No | It cites Phase 1 and specifies only reference acquisition, validation, caches, identities, and immutable assembly. |
| Did this add machinery without an approved need? | No | The design removes v2's dynamic type call and avoids runtime calendar, per-symbol prior requests, database, or generic frameworks. |
| Did version 2 drive the Phase 1 boundary instead of informing the detailed implementation contract? | No | The approved skeleton preceded inspection; v2 behavior conflicting with symbol-local containment and currentness was explicitly adapted or rejected. |

The corrective revision introduces no new product outcome. It makes existing
Phase 1 currentness, failure-containment, session-binding, fact-returning, and
boundedness requirements enforceable at the provider, cache, and final assembly
boundaries. Final-review findings are evidence for these corrections; they do
not expand predecessor scope or authorize component-2 work.

Any substantive **yes** requires explicit owner review before advancement.
