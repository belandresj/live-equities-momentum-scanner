# Versioned snapshot API

**Status:** Finally accepted 2026-08-08 under the Version 1 Release Program

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
schema and proof decisions remain revisable until final V1 acceptance

**Controlling Phase 1 requirements:** `PG-AVAIL-01`, `PG-AVAIL-02`,
`PG-AVAIL-03`, `PG-UI-01`, `PG-UI-02`, `PG-OBS-01`, `PG-OBS-02`,
`PG-OBS-03`, `ARCH-OWN-02`, `ARCH-OWN-03`, `ARCH-OWN-04`,
`ARCH-FLOW-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`, `DTE-COMMIT-04`,
`DTE-CHECKPOINT-03`, `LIFE-LIVE-05`, `LIFE-PUBLISH-01`,
`LIFE-PUBLISH-02`, and `LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Components 1-9 immutable
publication, field-status, accounting, readiness, checkpoint, and T/Q meanings

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the complete C10
boundary, schema/trust rules, proofs, slices, and sole delivery ledger.

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `accepted` | Owner V1 program revision; C9 finally accepted; recorded V2 scope inspected | Complete |
| Completed contract | `accepted_current_plan` | Focused publication/schema/HTTP trust review clean after exact-schema, immutable-T/Q, sample-identity, liveness, fence, accounting, and percentage-point conversion corrections | Complete; remains revisable through the correction loop |
| `C10-S1` immutable schema mapping | `accepted_corrected` | C11 contract work exposed that engine feature values are percentage points while the V1 wire schema is ratios; the mapper now divides every aggregate percentage field by 100 and the boundary proof distinguishes zero, extrema, and values above 100 percentage points; focused ordinary and race verification plus independent re-review are clean | Complete |
| `C10-S2` HTTP/CORS/runtime composition | `accepted` | `P-C10-HTTP`; ordinary and affected race clean; focused final correction re-review clean | Complete |
| 2026-08-11 first-ready/API diagnostic correction | `accepted_after_correction_review` | [`live-engine-api-first-ready-correction.md`](../live-engine-api-first-ready-correction.md) records the missing failed-capture limitation, real loopback first-ready proof, retained 5,691-symbol first-ready/post-fence proof, closed mapper invariant diagnostics, unchanged generic HTTP errors, clean ordinary/focused race verification, and two focused re-reviews with no remaining P1/P2. | Complete locally; live-provider confirmation is not claimed |
| Final component review | `accepted` | Mandatory read-only review found one P2 proof-matrix gap; focused correction re-review clean | Complete |

## Sections 1-4 — outcome, scope, ownership, and settled boundary

C10 exposes the latest immutable scanner publication through a private,
versioned, read-only HTTP API sufficient for the independent V1 dashboard.
The API preserves one exact product state; it never derives ranking, readiness,
coverage, feature availability, or provider membership.

In scope are one explicit V1 JSON schema and compatibility rule; publication,
binding/session, causal-time, ranking/readiness, row/field, aggregate/T/Q
coverage, symbol/work accounting, checkpoint, and bounded C8 operational
representations; one coherent source read per response; process-live and
backend-ready HTTP mappings; polling; loopback-only listening; explicit CORS
allow-listing; and bounded server/client behavior.

Authentication, TLS, hosting, public/nonloopback exposure, service splitting,
database/session state, streaming transports, browser calculations, UI assets,
and production cutover are non-scope. API unavailability or a slow client
cannot change engine lifecycle, block its ordered mutation path, or stop
checkpointing.

The `ScannerStateEngine` remains sole product-state and publication owner. S1
extends that publication with a defensive C9 T/Q projection and T/Q revision:
every product-visible T/Q state, pressure, command, coverage, or accounting
transition replaces publication identity even when committed `T` is unchanged.
Joining a later `ObserveTQ` or operational engine read is prohibited.

One Runtime capture mutex assigns a response-sample sequence, reads the clock
and process-live atomics once, loads one immutable engine publication once, and
samples fixed-cardinality nonengine metrics without rereading engine state. It
then derives readiness once from that captured publication, time, and process
fact and returns an operations-owned sealed bundle. Other packages may inspect
only a detached copy; they cannot reconstruct a capture or submit replaced
T/Q, readiness, metrics, or publication members to the public mapper. T/Q
repeats the publication ID and status repeats the sample time as construction
checks. Publication identity identifies only engine facts; `(sample_id,
publication identity if present)` identifies the whole response. Later mutable
facts are excluded, never overlaid. The C10 mapper owns only stable
representation; handlers own method, route, CORS, and transport outcomes.

These component requirements apply:

- `C10-IDENTITY-01`: engine-publication identity is the tuple of binding
  identity and a positive publication sequence represented as an exact decimal
  string. It is independent of committed `T`; corrections, T/Q-only changes,
  and nonmarket transitions may produce a new identity at the same `T`.
  Response-sample identity is a separate positive decimal sequence. Exact
  accepted-ingress support is represented by last engine sequence plus
  acknowledged connection/fence facts.
- `C10-SCHEMA-01`: `scanner.snapshot.v1` preserves every V1 row and independent
  field status/reason from one publication. Unavailable numeric/time values are
  JSON `null`, never fabricated zero; genuine zero remains numeric zero.
  All nonnegative duration-to-millisecond conversions use integer truncation
  (floor), so sub-millisecond values encode as genuine zero rather than rounding.
- `C10-STATUS-01`: process-live, backend-ready, ranking-current/mode/reason, and
  T/Q pressure/coverage remain distinct. T/Q health never changes backend
  readiness. Replay, stale, suppressed, ended, or unfenced state cannot be
  labeled live-ready. Liveness sampling is valid without binding or a positive
  publication; readiness and the product snapshot are not.
- `C10-HTTP-01`: read-only bounded endpoints expose the captured schema without
  handler-side state merging or calculation; slow/canceled clients are isolated.
- `C10-CORS-01`: requests without `Origin` are permitted for local tools;
  browser origins require exact configured matches. Wildcard, `null`, malformed,
  or unlisted origins receive no data.
- `C10-BOUNDS-01`: responses contain at most 20 rows, fixed-cardinality status
  families, and at most 1 MiB encoded JSON. Headers and server waits are bounded.

## Sections 5-8 — questions, approved reconnaissance, and reuse

The approved reconnaissance asked which V2 handler mechanics, JSON-null tests,
readiness behavior, and update transport were reusable without importing a
second state owner. Only these dirty-worktree files at predecessor HEAD
`5f92a151dd850002578a33a81ad90dea096c63b6` were inspected; hashes are the exact
provenance:

| V2 source and SHA-256 | Finding | Decision and required proof |
| --- | --- | --- |
| `internal/httpapi/httpapi.go`, `8a050d7182ff02062c83e1436cf785c148631eec845837cf9e62f062015d2edc` | Useful method/HEAD parity, no-store headers, response-size check, and standard-library server deadlines. Its `effectiveSnapshot` double-samples mutable health/queue stores and recalculates readiness. It also serves coupled UI assets. | Adapt bounded HTTP mechanics only. Reject state overlay, readiness calculation, legacy schema/routes, and asset serving. Prove one-source identity and handler nonmutation. |
| `internal/httpapi/httpapi_test.go`, `ab095c4605e17727bee275ec37e024e04682c3aeb3bba2c90347e13422a225a1` | Method/HEAD, unavailable-null, startup empty-array, response-bound, and concurrent clone isolation cases are useful. Many assertions depend on V2 health ownership. | Adapt only named transport/serialization cases against current engine authority. Reject V2 readiness and embedded-dashboard expectations. |
| `internal/httpapi/taq_api_contract_test.go`, `f64a2019afbe2a336d28d36df2ab343605d7241e17e8ff26d8ba530b567d9a9` | Explicit T/Q status/reason plus null-versus-genuine-zero is useful. Lifecycle/audit fields reflect predecessor shapes rather than accepted C9. | Adapt state/value mutation corpus; map only current C9 fields and proof identities. |

No dashboard assets, other V2 source, older V1 checkout, credentials, or live
provider evidence were inspected or whitelisted. Polling is selected because a
one-second dashboard cadence needs no second delivery protocol or state buffer.

## Sections 9-14 — schema, behavior, trust, and bounds

The only product route is `GET|HEAD /api/v1/snapshot`. Its schema is normative
below. `d` means a base-10 digit string with no sign or leading zero except
`"0"`; `u` means a nonnegative JSON integer no greater than `2^53-1`; `f` means
a finite JSON number; `t` means a UTC RFC3339Nano string; `?` permits JSON
`null`. Every listed property is required, no additional map/label families are
emitted, and every array is present even when empty.

| Object/path | Required properties and exact JSON type/unit |
| --- | --- |
| root | `schema_version:string="scanner.snapshot.v1"`; `sample:object`; `publication:object`; `status:object`; `ranking:object`; `rows:array` (0..20); `accounting:object`; `recovery:object`; `tq:object`; `checkpoint:object`; `operations:object` |
| `sample` | `id:d` (Runtime capture sequence); `sampled_at:t` |
| `publication` | `id:d` (positive engine publication sequence); `binding_identity:string` (nonempty); `trading_date:string` (`YYYY-MM-DD`); `run_mode:string enum`; `lifecycle:string enum`; `lifecycle_reason:string enum`; `suppression:string enum or ""`; `generated_at:t`; `committed_t:t?`; `last_engine_sequence:d`; `connection_epoch:d`; `connection_active:bool`; `aggregate_acknowledged:bool`; `aggregate_ack_position:position`; `hydration_fence:object` |
| `position` | `connection_epoch:d`; `frame_sequence:d`; `array_index:u`. All are zero only when the associated acknowledgement is absent. |
| `publication.hydration_fence` | `reconciled:bool`; `connection_epoch:d`; `through_frame_sequence:d`; `marker_ordinal:d`; `supported_through:t?`. The numeric coordinates are zero and time null exactly when unreconciled. |
| `status` | `process_live:bool`; `backend_ready:bool`; `readiness_reason:string enum`; `ranking_current:bool`; `causal_target:t`; `watermark_lag_ms:u?`; `accounting_valid:bool`; `tq_pressure_mode:string enum`; `tq_shed:bool`. Lag is null exactly when `committed_t` is null; otherwise it is nonnegative and genuine zero remains `0`. |
| `ranking` | `mode:string enum`; `reason:string enum`; `total_passers:u`; `known_rankable_count:u`; `day_invalid_rankable:u`; `qualified_day_invalid:u` |
| `rows[]` base | `rank:u` (1..20); `symbol:string`; `last_usd:f`; `day_change_ratio:f`; `mark_age_ms:u`; `from_4am_change:ratio`; `hod_drawdown:ratio`; `day_range_position:ratio`; `range_30m_position:ratio`; `range_60m_position:ratio`; `activity:ratio`; `tape_rate:object`; `spread:object`; `tq_membership:object` |
| `ratio` | `status:string field-status`; `reason:string field-reason`; `value_ratio:f?`. Value is nonnull only for `current`; genuine zero is `0`. Ratio `0.125` means 12.5%, including range positions and Activity's dimensionless score. The mapper converts the engine's owner-authoritative percentage-point representation exactly once by dividing by 100; it never exposes percentage points under a ratio property. |
| `rows[].tape_rate` | `status:string T/Q-status`; `reason:string T/Q-reason`; `trade_coverage:bool`; `one_second:rate`; `five_second:rate`; `timestamp_basis:string enum`; `lifecycle_records_observed:bool` |
| `rate` | `status:string T/Q-status`; `reason:string T/Q-reason`; `trades_per_second:f?`, nonnull only for `current`; genuine covered zero is `0` |
| `rows[].spread` | `status:string T/Q-status`; `reason:string T/Q-reason`; `quote_coverage:bool`; `cents:f?`; `basis_points:f?`; `valid_duration_ms:u`; `quality:string enum`. Both values are nonnull only for `current`; genuine locked spread is numeric zero. |
| `rows[].tq_membership` | `desired:bool`; `provider_present:bool`; `provider_membership_unknown:bool` |
| `accounting` | `population:object`; `qualification:object`; `uncertainty:object` |
| `accounting.population` | `universe_total:u`; `valid_prior_close:u`; `invalid_or_missing_prior_close:u`; `trusted_rankable_mark:u`; `trusted_below_price_mark:u`; `no_print_through_t:u`; `invalid_mark:u`; `unknown_due_failure_or_fence:u`; `covered_population:u`; `unresolved_population:u` |
| `accounting.qualification` | `not_yet_passed:u`; `provisional:u`; `finalized:u`; `unresolved:u` |
| `accounting.uncertainty` | `bootstrap_origin:u`; `post_bootstrap_gap:u`; `local_invalid:u` |
| `recovery` | `purpose:string enum or ""`; `generation:d`; `start:t?`; `end:t?`; `supported_through:t?`; `fence_reconciled:bool`; `policy_waiting:bool`; `work:object`; `rows:object` |
| `recovery.work` | `planned:d`; `open:d`; `completed_value:d`; `completed_empty:d`; `failed:d`; `canceled:d`; `fenced:d` |
| `recovery.rows` | `consumed:d`; `inserted:d`; `duplicate:d`; `conflict_or_withdrawal:d`; `rejected:d`; `fenced:d`; `integrity:d` |
| `tq` | `desired_symbols:array<string>` (0..20, rank order); `pressure_mode:string enum`; `aggregate_only:bool`; `shed:bool`; `retained_bound_hit:bool`; `pressure_misses:u`; `pressure_transitions:d`; `pressure_fenced:d`; `known_present:u`; `known_absent:u`; `unknown:u`; `retained_trades:u`; `retained_quotes:u`; `retained_fingerprints:u`; `facts:object`; `commands:object` |
| `tq.facts` | `consumed:d`; `applied:d`; `duplicate:d`; `rejected:d`; `fenced:d`; `pressure_shed:d`; `integrity:d` |
| `tq.commands` | `issued:d`; `pending:d`; `acknowledged:d`; `failed:d`; `fenced:d`; `result_fenced:d` |
| `checkpoint` | `installed:bool`; `submitted:d`; `in_progress:d`; `pending:d`; `completed:d`; `failed:d`; `canceled:d`; `superseded:d` |
| `operations` | `sample_accounting_valid:bool`; `queue_capacity_frames:u`; `queue_current_frames:u`; `queue_high_frames:u`; `queue_current_bytes:u`; `queue_high_bytes:u`; `deliveries:d`; `consumer_deferred:d`; `mean_processing_delay_ms:u`; `max_processing_delay_ms:u`; `max_processing_delay_one_second_ms:u`; `heap_alloc_bytes:d`; `heap_in_use_bytes:d`; `goroutines:u`; `connection_recovery_attempts:d` |

The current producer enum domains are exact: run mode `live|replay`; lifecycle
`initializing|awaiting_session|awaiting_aggregate_ack|hydrating|live|recovering|
replaying|suppressed|ended`; ranking mode
`unavailable|qualified_current|degraded_bootstrap|stale|suppressed`; ranking
reason `""|no_committed_watermark|no_trusted_marks|incomplete_population|
qualification_incomplete|global_suppression`; readiness reason
`""|runtime_unavailable|binding_mismatch|not_live_mode|lifecycle_not_ready|
suppressed|aggregate_unacknowledged|fence_pending|ranking_noncurrent|
watermark_missing|watermark_stale|accounting_invalid`; field status
`warming|current|unavailable|invalid`; field reason
`""|before_first_print|history_incomplete|prior_close_unavailable|
no_aggregate_in_target|rolling_warmup|reference_warmup|zero_width|
historical_conflict|invalid_input|state_bound_exceeded`; T/Q status
`unselected|warming|current|stale|unavailable|invalid|pressure_shed`; T/Q reason
`""|coverage|coverage_warming|five_second_warming|qualifying_original_prints|
unequal_repeat|one_sided_quote|crossed_quote|stale_quote|insufficient_coverage|
pressure`; pressure mode `normal|taq_degraded|aggregate_only`; timestamp basis
`""|none|participant|sip_fallback|mixed`; spread quality
`""|reviewed_ordinary|known_special|unclassified`; hydration purpose
`""|fresh_bootstrap|checkpoint_catchup|gap_recovery`; suppression
`""|same_binding_recovery_allowed|clean_reinitialization_required|
restart_required|terminal_replay_failure`. Lifecycle reason is
`""|binding_before_session|binding_in_session|binding_after_session|
session_start_without_aggregate_ack|session_end|controlled_stop|
sequence_exhaustion|clock_regression|canonical_integrity|
publication_integrity|accounting_integrity|closed|replay_start|replay_end|
replay_failure|aggregate_acknowledged|aggregate_acknowledged_at_session_start|
aggregate_epoch_lost|ingress_integrity|hydration_complete|
recovery_exhausted`. The mapper copies enum values without translation.

The primary identities are asserted in every encoded snapshot:

```text
universe_total = valid_prior_close + invalid_or_missing_prior_close
valid_prior_close = trusted_rankable_mark + trusted_below_price_mark
                  + no_print_through_t + invalid_mark
                  + unknown_due_failure_or_fence
planned = open + completed_value + completed_empty + failed + canceled + fenced
recovery.rows.consumed = inserted + duplicate + conflict_or_withdrawal
                       + rejected + fenced + integrity
tq.facts.consumed = applied + duplicate + rejected + fenced + pressure_shed + integrity
tq.commands.issued = pending + acknowledged + failed + fenced
checkpoint.submitted = in_progress + pending + completed + failed + canceled + superseded
```

Within major V1, additive optional fields are compatible. A new reason value is
compatible only because clients must treat unknown status/reason values as
unavailable presentation, never as current. Removing/renaming a field, changing
type/unit/meaning, weakening status, or changing identity semantics requires
`/api/v2` and a new `schema_version`. There is no request-driven schema
negotiation in V1.

`GET|HEAD /livez` uses the same capture but never requires binding/publication.
Its exact body is `schema_version="scanner.liveness.v1"`, `sample_id:d`,
`sampled_at:t`, `process_live:bool`, and `reason:""|runtime_unavailable`; it is
`200` whenever process-live, including initializing/no-binding, and otherwise
`503`. `/readyz` requires a valid positive publication and returns
`schema_version="scanner.readiness.v1"`, the sample fields, `process_live`,
nullable `publication_id`/`binding_identity`, `backend_ready`, and `reason`
(readiness enum plus `publication_unavailable`). It is `200` only when ready and
otherwise `503`. The snapshot route returns `200` for any valid coherent
publication, including honest not-ready/ended state; absent/invalid/over-bound
publication is `503`. Unknown routes are `404`; non-GET/HEAD methods are `405`
with `Allow: GET, HEAD`; `OPTIONS` exists only for valid CORS preflight.

Every JSON response uses `application/json`, `Cache-Control: no-store`,
`X-Content-Type-Options: nosniff`, exact `Content-Length`, and no body for HEAD.
The server binds `127.0.0.1:8080` by default and rejects nonloopback configured
addresses. Allowed origins are canonical absolute `http`/`https` origins with
no path/query/fragment/userinfo; response matching is exact and returns that
origin with `Vary: Origin`. Credentials and wildcard origins are unsupported.
Disallowed/malformed origins and invalid preflights return `403` before source
access. Valid preflight returns `204` for GET/HEAD only.

Encoded responses are built before headers and capped at 1 MiB. Server limits
are 16 KiB headers, 5-second read-header/read, 10-second write, and 30-second
idle timeouts. Source/encoding errors use a fixed bounded JSON error and never
expose internals. Handler reads use a defensive value and hold no engine lock
while encoding or writing.

## Sections 15-17 — proofs, slices, discretion, and correction

`P-C10-SCHEMA` is one golden plus semantic-mutation proof over unavailable,
qualified-current, exact-empty, degraded, pressure-shed, stale, suppressed,
and ended publications. It asserts every normative name/nesting/type/null/unit
and enum, genuine zero versus null, row order/cardinality, every displayed
field, all seven exact identities above, decimal precision above `2^53`, and
mutation isolation during concurrent replacement. A T/Q-only transition at
unchanged `T` must replace publication ID and appear in the captured immutable
projection. Separate samples at an unchanged publication exercise readiness
expiry and process termination: engine fields remain byte-identical while
sample ID/time/status change. Mutations distinguish aggregate positions with
the same frame but different array index, unequal hydration marker ordinals,
missing-watermark/null lag from exact-zero lag, and every term of both hydration
identities. Its dangerous counterexample is a plausible
mixed response whose rows come from one publication while readiness or T/Q
comes from another. Limitation: it proves representation/coherence, not HTTP or
browser rendering.

`P-C10-HTTP` uses loopback `httptest` plus one real loopback listener. It proves
route/method/HEAD/status mapping, one capture per response, no-store/security
headers, exact allowed-origin/preflight behavior, disallowed-origin no-capture,
1 MiB failure, loopback-only configuration, server timeouts, client
cancellation, concurrent publication reads, and continued engine progress with
a blocked client. It explicitly proves `/livez=200` during initializing with no
binding/publication while `/readyz` and snapshot are `503`, then proves
nonlive liveness body/status after process termination. Its dangerous
counterexamples are a disallowed origin receiving data, `/readyz` promoting
not-ready state, or a slow writer holding engine authority. Limitation: no
TLS/auth/public-host/browser or network-SLA claim.

`C10-S1` extends engine publication construction/fingerprinting with a detached
C9 T/Q projection and mandatory publication replacement for every visible
T/Q-only transition; it also owns the single-read engine snapshot projection,
Runtime response capture, stable schema/mapper, and `P-C10-SCHEMA`. It prohibits
post-capture `ObserveTQ`/`ObserveOperational` joins. `C10-S2` owns the
standard-library handlers, CORS/listener configuration, Runtime/CLI
composition, and `P-C10-HTTP`. S2 cannot add market logic or repair an
incoherent S1 input. Ordinary verification is
`go test -short -timeout 2m ./...`; affected engine/operations/API/cmd race
packages run under five minutes.

Implementers may choose private DTO/helper names, package layout, error text,
and fixture construction inside these bounds. They may not expose mutable
engine objects, serialize internal structs automatically, add a second
readiness/publication owner, serve UI assets, enable wildcard/nonloopback
access, or add streaming/framework dependencies. A schema ambiguity, mixed
publication, CORS bypass, response-bound miss, or slow-client coupling reopens
the lowest implicated slice under the V1 correction loop; it never creates an
owner-interruption gate.

## Sections 18-19 — acceptance and drift audit

Each slice records behavior, ownership/interface changes, dangerous
counterexamples, exact proof result, limitations, correction evidence, and
focused review if triggered. Final acceptance requires both primary proofs,
clean ordinary/affected-race verification, success/failure walkthrough, one
mandatory final read-only review, and no unresolved fixed-authority conflict.

This contract preserves all cited Phase 1 meanings, introduces one
representation owner but no product-state owner, keeps UI deployment
independent, and defers authentication, TLS, public hosting, streaming, and
market-hours validation. The V2 whitelist is exact and used only as evidence;
all reused behavior is re-proven against current Components 1-9.

The completed-contract review initially rejected a prose-only schema, mutable
post-publication T/Q joins, ambiguous engine-versus-sample identity, and
publication-dependent liveness. Focused corrections added the normative field
table, T/Q publication replacement, one-read capture model, and prepublication
liveness. Re-review then required full causal-fence coordinates, the hydration-
row identity, and null missing-watermark lag. The final focused re-review found
no remaining P1/P2. These findings corrected only C10 representation and proof
allocation; no accepted C1-C9 market meaning was reopened.

`C10-S1` now publishes every product-visible T/Q-only transition under a new
engine publication identity, captures publication/operational/T/Q facts from
one atomic cell, derives readiness once from the same sampled clock/process
fact, and maps the sealed capture into the exact `scanner.snapshot.v1` schema.
The primary proof covers all named publication modes, genuine zero versus null,
decimal precision, complete accounting identities, unequal causal fence
coordinates, every hydration identity term, same-`T` T/Q replacement, sealed
capture nonreconstruction, and bounded concurrent publication replacement.
The dangerous mixed-publication/mixed-sample capture is prevented by an
operations-private capture payload and rejected construction checks. Ordinary
verification and the affected engine/operations/snapshot race tier passed; the
focused implementation review required sealed capture authority, a positive
hydration marker, and complete concurrent/identity mutation evidence, and its
final re-review found no remaining P1/P2. The proof is local and deterministic;
HTTP, browser, provider, and live-market behavior remain outside S1. S2 remains
valid without revision.

The later C11 unit audit reopened only C10-S1 representation: the engine owns
percentage-point values (`0.25` means 0.25%, and Activity spans 0..100), while
the public schema owns dimensionless ratios (`0.0025` means 0.25%). The mapper
now divides Day change and every status-bearing aggregate feature by 100 exactly
once. The corrected proof distinguishes negative and above-100-point change,
range/Activity extrema, genuine zero, and non-current null. Focused ordinary
and affected race verification passed, and focused independent re-review found
no semantic mapper/proof defect. C10-S2 and C1-C9 meanings were unaffected; C11
must consume the corrected wire ratios directly.

`C10-S2` now serves the sealed capture through the three bounded loopback
routes, applies exact-origin CORS before source access, and maps liveness,
readiness, publication validity, HEAD, and error outcomes without handler-owned
market calculations. The scanner composition starts the API with explicit
loopback/origin flags and cancels and joins API/live work before Runtime
shutdown. `P-C10-HTTP` covers initializing/awaiting-ack liveness without a
valid product publication, ready and terminal states, every route/method/HEAD
branch, exact CORS/preflight acceptance and rejection, the 1 MiB response cap,
real loopback binding, fixed server deadlines, canceled clients, concurrent
publication reads, and engine progress while a response writer is blocked.
The final read-only review found no code-path correctness or ownership defect;
its sole P2 finding was missing exact assertions for several method/HEAD/CORS
branches. The expanded table-driven proof passed and focused re-review found no
remaining P1/P2. Ordinary and affected race verification pass. Evidence is
local loopback/httptest only: no browser, TLS/auth/public hosting, provider,
live-market, or network-SLA claim is made.

Final C10 conformance has one representation owner and one HTTP transport path;
the engine remains sole product-state/publication owner, operations remains the
readiness/process sampler, and T/Q health remains independent from aggregate
ranking and readiness. The success path is one sealed Runtime capture to one
validated schema/body; malformed capture, unavailable publication, invalid
origin/preflight, over-bound encoding, canceled client, and shutdown timeout
remain bounded non-success paths. No accepted C1-C9 market meaning was changed.
