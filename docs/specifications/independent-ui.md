# Independent UI

**Status:** Completed C11 contract is the current executable plan under the
Version 1 Release Program; implementation pending

**Boundary approval:** Approved 2026-08-07 by the owner through the Version 1
Release Program revision

**Completed-contract authority:** V1 Release Program orchestrator; lower-level
UI, fixture, and slice decisions remain revisable until final V1 acceptance

**Controlling Phase 1 requirements:** `PG-RANK-04`, `PG-FEATURE-01`,
`PG-FEATURE-02`, `PG-FEATURE-03`, `PG-FEATURE-04`, `PG-AVAIL-01`,
`PG-AVAIL-02`, `PG-AVAIL-03`, `PG-UI-01`, `PG-UI-02`, `PG-OBS-03`,
`ARCH-OWN-03`, `ARCH-OWN-04`, `DTE-CLOCK-05`, `DTE-CLOCK-06`,
`DTE-COMMIT-04`, `LIFE-PUBLISH-01`, `LIFE-PUBLISH-02`, and
`LIFE-PUBLISH-03`

**Approved dependencies:** Finally accepted Component 10 API and its stable V1
product meanings

## Contract document map and delivery ledger

**Layout:** Compact single-file contract. This file owns the C11 boundary plan;
Sections 8-19 are completed here after just-in-time V2 UI reconnaissance.

| Document | Exclusive normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This parent | Complete C11 boundary, detailed contract, proofs, slices, and sole delivery ledger | Sections 1-19 | Every C11 task | Component 10 and V1 program |

| Item | State | Evidence | Next action |
| --- | --- | --- | --- |
| Boundary/reconnaissance plan | `accepted` | Direct owner V1 program revision; exact source list recorded before bounded content inspection | Complete |
| Completed contract | `accepted_current_plan` | Focused review corrections add full T/Q trust disclosure, fail-closed semantic coherence, additive compatibility, C10-matching response bounds, exact accessibility assertions, and delayed-refresh containment; focused re-review clean | Begin C11-S1 |
| `C11-S1` API/view-state integration | `pending` | `P-C11-STATE` | Begin after completed-contract review |
| `C11-S2` visual/interaction/accessibility | `pending` | `P-C11-VISUAL` | Begin only after S1 acceptance |
| Final component review | `pending` | Mandatory read-only review after both proofs and verification | Then integrated V1 RC review |

## 1-4. Outcome, scope, ownership, and settled boundary

C11 delivers an independently runnable Chrome-desktop dashboard that adapts
the useful V2 scanner layout, visual character, information density, and
interactions while presenting only C10-owned product meaning.

In scope:

- rank, symbol, Last, Day %, From 4AM %, HOD drawdown, session/30m/60m range
  position, Activity, Tape Rate, and NBBO Spread;
- process/backend/ranking status, stale/unavailable/invalid/warming field
  states and reasons, aggregate versus T/Q health, coverage, and bounded
  operational context;
- fewer-than-20 rows and exact API order without client resorting;
- a high-fidelity Chrome-desktop adaptation of useful V2 layout, density,
  visual character, scanning hierarchy, and interactions;
- keyboard/focus, semantic table/status, contrast, reduced-motion where
  relevant, loading/error/reconnect behavior, and an independent local run
  against configured C10 origin.

Not in scope: browser-owned market calculations, ranking, readiness, field-
status inference, state reconciliation, direct backend internals, mobile,
multi-browser certification, public deployment, authentication/TLS, PWA/offline
mode, or a generalized design system.

The browser owns presentation, interaction, transient view preferences, and
transport retry display. C10 owns every market value, order, publication
identity, readiness, coverage, and field status. UI deployment and restart
cannot restart or relink the backend.

The component introduces no ranking key, market-time/window interpretation,
readiness rule, or mutable scanner owner.

## 5-7. Evidence questions, reconnaissance, and delivery plan

The detailed contract must identify the useful V2 screen regions,
interactions, visual assets/tokens, density, field formatting, accessibility
gaps, obsolete status semantics, backend coupling, build/runtime boundary, and
the smallest deterministic screenshot/state fixture set.

After C10 final acceptance, reconnaissance may inspect the V2 UI specification,
UI code, assets, and focused UI tests only. This exact category is owner-
approved. Reject browser calculations/readiness, obsolete lifecycle/field
states, direct backend coupling, unrelated application screens, deployment
infrastructure, credentials, and server code beyond C10 fixtures.

The current exact reconnaissance list was selected from a filename-only V2 UI
inventory before opening file contents:

- `docs/dashboard-implementation-spec.md`: identify the intended desktop
  hierarchy, density, responsive floor, interaction, and accessibility rules;
- `web/index.html`: identify the implemented layout, CSS tokens, client fetch/
  retry behavior, formatting, and any browser-owned calculations or obsolete
  API coupling that must be removed;
- `web/embed_test.go`: identify the predecessor's narrow static-asset/runtime
  proof mechanics, if any are reusable;
- `docs/live-run-2026-08-04-ui-handoff.md`: inspect only recorded visual/usability
  observations, not provider values, credentials, or live correctness claims.

The working notes, refactor/runtime-decoupling spec, Go server/embed wiring,
other tests, unrelated screens, deployment files, and all credentials/provider
paths remain excluded unless a recorded in-component correction expands this
list before inspection.

Likely primary proofs:

1. one deterministic API-to-view state matrix covering complete, fewer-than-20,
   exact empty, stale, unavailable, warming, invalid, T/Q-degraded, recovery,
   API error, and publication change without client recomputation; and
2. one Chrome-desktop visual/interaction/independence acceptance covering the
   approved layout baseline, density, key interactions, accessibility, and
   backend-continuity during UI restart.

**Provisional slices:** At most two: `C11-S1` C10 integration and status/view
states; `C11-S2` high-fidelity visual/interaction/accessibility completion.
Combine them if one coherent implementation and proof remains reviewable.

**Boundary checkpoint:** Exact Phase 1 IDs, outcome, ownership/non-scope,
settled invariants, evidence questions, V2 scope/exclusions, likely proofs, and
provisional slice outcomes are recorded. No V2 source was inspected for this
plan. Direct owner approval makes an independent skeleton review unnecessary.

## 8. Reconnaissance result and reuse decisions

The following exact V2 sources were inspected after the scope above was
recorded. They are evidence, not authority:

| Source and SHA-256 | Reuse decision |
| --- | --- |
| `docs/dashboard-implementation-spec.md`, `74733ee7f80b0734187dd9fe0714b3b472849fe9ffe54ddc29ab719fa6b40f52` | Adapt the dense dark desktop hierarchy, fixed status bands, 12-column order, stable rows, explicit unavailable reasons, and keyboard-accessible detail. Reject historical owner gates, browser-owned currentness, obsolete fields, and old units. |
| `web/index.html`, `ac11e6944cda6c6affab416b8c6e5c31121188417f79e24cf0aca5d75ad08377` | Adapt palette, information hierarchy, one-second nonoverlapping fetch, inactive-state styling, and safe text presentation. Replace obsolete endpoint/schema, embedded deployment, inferred readiness, HTML-string rendering, and client market calculations. |
| `web/embed_test.go`, `b5099e85a6b81c8e81647d0b070ecd3d2da6917d8341b8e857fbe9cb9ac42457` | Reuse only the idea of deterministic asset/runtime boundary tests. Do not embed assets into the scanner or reuse predecessor server wiring. |
| `docs/live-run-2026-08-04-ui-handoff.md`, `7f1ec32bdb1b63e207077c9e4f18753b0817eb1b5c887f188a80c5e37b2b25df` | Reuse observations that independent failure, genuine zero, server-owned states, and one-screen density matter. Reuse no provider value, live-market claim, credential, or release gate. |

The inspection supports a high-fidelity adaptation, not a port. No excluded V2
source was opened. C11 adds no third-party runtime or browser framework.

## 9. Exact C10 input and display semantics

The UI accepts only `scanner.snapshot.v1` from `GET /api/v1/snapshot`. It
preserves `rows` order and server rank exactly and rejects more than 20 rows,
nonconsecutive ranks, duplicate/empty symbols, missing required properties,
nonfinite numbers, or incompatible schema. It never sorts, filters, recomputes
rank, derives readiness, or joins another response to repair a snapshot.

Display conversion is presentation only:

- `last_usd` is USD with two to four decimals as magnitude requires;
- `day_change_ratio`, every status-bearing aggregate `value_ratio`, and
  Activity are ratios, multiplied by 100 only for a `%` label. Activity 0.80 is
  displayed as 80%, not 0.8% or 80.00 points;
- range positions use the same ratio-to-percent formatting; they are not
  clamped by the browser;
- Tape Rate uses five-second trades/second as the primary value and one-second
  burst as secondary context. Keyboard/focus detail always exposes trade
  coverage, participant/SIP/mixed timestamp basis, lifecycle-record observation,
  and the exact status/reason so a current number cannot imply fully corrected
  consolidated tape;
- Spread uses basis points as the primary value and cents as secondary context.
  Keyboard/focus detail always exposes quote coverage, valid duration, quality,
  and exact status/reason;
- UTC timestamps are rendered in the browser locale only after valid parsing;
  durations/counters retain the API units and decimal strings are never coerced
  through an unsafe JavaScript integer.

Genuine numeric zero remains visible. A non-current status hides its numeric
slot and shows its exact status/reason; color never overrides that state.
Presentation bands are stable and deliberately non-semantic: return magnitude
at 0, 1, 2.5, 5, and 10 percentage points; range/Activity at 0, 25, 50, 75,
and 100%; Tape at 0, 1, 5, 15, and 30 trades/second; Spread at 0, 5, 10, 25,
and 50 bps. Values outside a visual band remain numerically visible. These
bands do not create alerts, qualification, ranking, readiness, or capacity
claims.

## 10. Authoritative view-state matrix

One captured response produces one render model. Before state selection, the
client enforces the C10 normative type/null, rank/order, decimal-string,
full-accounting-identity, publication/fence, lifecycle/status, and readiness
relationships. Unknown additive root or nested properties are ignored for
within-major compatibility; required properties and known meanings remain
mandatory. Any contradictory known facts reject the whole response atomically.
In particular, overall current requires `process_live`, `backend_ready`, and
`ranking_current` true, `ranking.mode=qualified_current`, a present committed
watermark/lag, valid accounting, and a C10-permitted live/hydrating lifecycle.
Any known noncurrent fact takes fail-closed precedence; a response claiming
ready/current simultaneously with ended, suppressed, stale, unavailable,
missing-watermark, invalid accounting, or false process-live is invalid rather
than green. The following legal states are distinguishable without browser
inference:

| C10 fact | Required UI consequence |
| --- | --- |
| All current-coherence predicates above | Current status; render 0..20 rows in exact order. |
| Qualified current with zero rows | Explicit “No symbols currently qualify”; never loading or error. |
| Qualified current with 1..19 rows | Render only those rows; no placeholders or client backfill. |
| `ranking.mode=degraded_bootstrap` | Prominent incomplete-population band; rows may be shown only as degraded, never current. |
| `stale`, `suppressed`, `ended`, or `unavailable` ranking/lifecycle | Exact server mode/reason in a persistent noncurrent band; retained rows, if supplied, remain visibly noncurrent. |
| Aggregate field `warming`, `unavailable`, or `invalid` | Per-cell status token and keyboard/focus-accessible reason; no fabricated zero. |
| T/Q `warming`, `unavailable`, `invalid`, `pressure_shed`, or uncovered | Tape/Spread cell independently noncurrent; aggregate rank and aggregate fields remain intact. Tape detail still states coverage, timestamp basis, and lifecycle observation; Spread detail still states coverage, duration, and quality. |
| `tq.aggregate_only`, degraded pressure, unknown membership, or retained-bound hit | Separate T/Q health band and affected cell state; never downgrade aggregate ranking unless C10 says so. |
| Recovery/checkpoint/operations facts | Bounded diagnostics in the status/details region; never inputs to browser readiness. |
| HTTP/network failure or invalid schema | Keep the last valid render only as an explicitly frozen snapshot, mark transport disconnected, show its original `sampled_at`, and never call it current. With no valid prior snapshot, show an error state and no rows. |
| A later valid response, same publication ID | Replace sample/status/operations display atomically while engine facts may remain equal. |
| A later valid response, new publication ID | Replace the entire render model atomically; no row-level merge or animation implies continuity. |

Unknown future enum values are displayed as `unknown` with the original bounded
token available in detail and are treated noncurrent. They do not crash the
poller or silently inherit a known visual meaning.

## 11. Transport, independence, and trust boundary

The dashboard is a separate loopback process serving static assets. Its
configured API origin must be an exact `http` or `https` loopback origin with
no credentials, path, query, or fragment. Production defaults are UI
`127.0.0.1:4173`, API `http://127.0.0.1:8080`, one-second polling, and a bounded
three-second request timeout. The scanner must explicitly allow the UI origin
through C10 CORS.

Polling permits exactly one request in flight. A tick during an active request
is skipped and immediately changes only the transport presentation to
`refresh_delayed`; the last response's sampled time and backend-owned status
remain visible but the overall surface cannot remain unqualified green/current.
At the three-second timeout the request is aborted and transport becomes
`disconnected`; the next regular tick retries. A successful validated response
returns transport to `connected`. This browser monotonic timing describes only
request observation and never recalculates backend readiness or watermark age.
There is no exponential scheduler, WebSocket, service worker, response cache,
local market-state persistence, or overlapping fetch. Every accepted response
is validated and swapped as one render model. API strings enter the DOM only
through text nodes/attributes; no response value is interpreted as HTML.

Stopping/restarting the UI server affects neither scanner process nor API.
Stopping the API makes the UI honestly disconnected but leaves the UI process
responsive. C11 neither reads backend memory nor becomes another state owner.

## 12. Desktop visual and interaction contract

The required target is current Chrome desktop at a 1440x900 CSS viewport with
100% zoom. The whole top-20 table, its header, and primary status strip fit in
one viewport without pagination or horizontal clipping at that target. A
1120px content floor may horizontally scroll below the target; mobile and
multi-browser behavior are nonclaims.

The main surface is a dense dark scanner table in this exact column order:
Rank, Symbol, Last, Day %, From 4AM %, HOD DD %, Day Range %, 60m Range %, 30m
Range %, Activity, Tape Rate, Spread. Stable compact rows use tabular numerals,
right-aligned values, sticky headers, subdued grid lines, and hue-plus-text or
shape redundancy. Row order changes only when a new response changes it; no
decorative motion occurs on polling.

The status strip exposes transport connection/delay separately from sample
time and process/backend/ranking,
committed watermark/lag, lifecycle/recovery, T/Q pressure, and accounting
validity in a scannable hierarchy. Secondary operational counters live in a
keyboard-operable disclosure, not additional table columns. Field reasons are
available on focus as well as pointer hover, including all Tape/Spread trust
details. All controls have visible focus;
the table has a caption and semantic headers; status changes use a polite live
region; text/background and status-token contrast meet WCAG AA. The stylesheet
honors `prefers-reduced-motion` and does not rely on motion to convey state.

## 13. Bounds and failure containment

Static responses are bounded assets; the UI server is loopback-only, uses
fixed header/read/write/idle timeouts, serves only known files below its asset
root, rejects traversal/symlink escape, and emits no directory listing. CSP
permits only same-origin assets and the configured loopback API connection;
`nosniff`, no-store, and a restrictive referrer policy apply.

The client reads at most 1 MiB plus one sentinel byte from the response stream
before JSON parsing, regardless of absent or dishonest `Content-Length`; a
larger body is atomically rejected. This response-wide cap is also the string/
DOM memory authority: C11 imposes no narrower local cap on C10-defined or
compatible additive strings. Ignored additive values are never retained in the
render model. Known strings remain literal, are visually clipped when needed,
and retain their bounded-by-response accessible text. The view model retains at
most one last-valid snapshot and one current error, creates at most 20 rows,
and retains no history series. Malformed/over-bound responses, unsafe API
origin, request timeout, DOM render error, and unknown schema are bounded non-
success paths and never partially replace the current render.

## 14. Primary proofs

`P-C11-STATE` is a deterministic dependency-free API-to-view proof using the
smallest `scanner.snapshot.v1` fixture corpus. It covers complete 20, fewer,
exact empty, degraded, stale, suppressed, ended, absent watermark, recovery,
field warming/unavailable/invalid, genuine zero, T/Q pressure/aggregate-only,
unknown enum, bounded-by-response malicious strings, over-bound body, invalid
schema/rank/order/duplicate/over-20, inconsistent lifecycle/readiness/null/
accounting facts, additive root/nested properties including values above any
presentation-local text length,
same-publication sample replacement, new-publication atomic replacement, one-
request scheduling, skipped-tick `refresh_delayed`, timeout/error freeze, and
reconnect. It asserts full Tape/Spread trust-detail preservation and no sort,
filter, percentage/Activity mis-scaling, readiness derivation, unsafe integer
coercion, HTML interpretation, or partial mutation. Its dangerous
counterexample is a plausible stale response or malformed row becoming a
current reordered table.

`P-C11-VISUAL` runs the production static UI against a deterministic local C10
fixture endpoint in Chrome. At 1440x900 it captures current-20, exact-empty,
aggregate-degraded, T/Q-degraded, refresh-delayed, and transport-disconnected
states; asserts
all 12 columns and 20 rows fit without pagination/horizontal clipping; checks
zero/null distinction, fixed order, complete Tape/Spread disclosures, and safe
hostile symbols/reasons. Deterministic DOM/computed-style checks assert table,
caption and scoped column headers, polite `aria-live`, disclosure keyboard
operation, visible focus, focus-accessible reasons, reduced-motion overrides,
and WCAG-AA contrast ratios for every text/status token state. It then
restarts only the UI server while the fixture API continues and proves the API
remains reachable and the reloaded UI recovers. Its dangerous counterexample
is a visually convincing green/current screen derived from stale, unavailable,
or T/Q-shed data. This is local deterministic Chrome evidence, not browser-
matrix, provider, market-hours, public-network, accessibility-audit, or SLA
evidence.

## 15. Sequential slices and assignments

`C11-S1` owns the independent static server, validated configuration, pure
schema/view model, nonoverlapping poller, safe atomic renderer, complete state
matrix, and `P-C11-STATE`. Allowed boundaries are `cmd/dashboard`, `internal/ui`
if a Go server package is useful, `ui` assets/modules/fixtures, focused tests,
this ledger, README run instructions, and the minimum ignore rule for generated
visual artifacts. It may choose ordinary file/module layout but cannot add a
framework, browser market logic, scanner asset serving, public binding, or a
second response representation.

`C11-S2` begins only after S1 acceptance. It owns high-fidelity CSS/layout,
status/details interactions, accessibility, Chrome fixtures/harness,
`P-C11-VISUAL`, runbook polish, and final C11 evidence. It may refine
presentation tokens and compact dimensions but cannot change the API model,
state meanings, poller, server ownership, or target scope.

Both assignments use only the four recorded V2 sources. A schema mismatch,
unsafe response reaching success, false-current display, inability to fit the
desktop baseline, or UI/backend lifecycle coupling reopens the lowest affected
slice and invokes the V1 correction loop without owner interruption.

## 16. Verification and review

S1 runs the narrow deterministic UI/view/server tests plus ordinary Go
verification when accepted. S2 adds the bounded Chrome proof and affected
race tests for the Go server. Every loop, timer, fetch, listener, and browser
wait is explicitly bounded; generated screenshots remain test artifacts, not
product authority. No local command exceeds 15 minutes.

The completed contract receives a focused read-only review because it fixes a
consequential browser/API trust boundary. Each slice records coherent behavior,
deferred work, proof result, dangerous counterexample, limitation, invalid
states prevented by construction, success/failure walkthrough, and any
correction. Final C11 acceptance requires both proofs, clean ordinary and
affected-race verification, one final read-only component review, and no fixed-
authority conflict. The same reviewer is reused when practical.

The completed-contract review initially found two false-success risks and four
proof/bound issues: hidden Tape/Spread trust qualifiers, contradictory known
facts with no noncurrent precedence, ambiguous additive compatibility, an
unbounded pre-parse body/text rule, incomplete deterministic accessibility
assertions, and a hung request that stayed visually current. Corrections expose
all T/Q qualifiers, reject incoherent known facts, ignore additive properties,
stream-cap responses at C10's 1 MiB limit, assert the exact semantic/accessibility
surface, and separate `refresh_delayed` from backend status. Focused re-review
then removed incompatible local per-string rejection limits in favor of the
response-wide cap and found no remaining P1/P2. These changes add no browser
market logic and reopen no C1-C10 product meaning.

## 17-19. Deferrals, discretion, and conformance

Deferred beyond private/local V1: mobile/responsive certification below the
1120px floor, Safari/Firefox/Edge certification, public hosting, authentication,
TLS, CSP for nonloopback origins, streaming, offline/PWA, customizable sorting
or filters, saved preferences, historical charts, notifications, provider/live
validation, and formal accessibility certification.

Implementer discretion is limited to file names, pure-function factoring,
exact CSS tokens within the visual baseline, safe numeric decimal precision,
and test-fixture mechanics. It cannot reinterpret C10 values/status, change
visual column order, add a market threshold, expose an unavailable value,
derive currentness from local time, or make UI process health part of backend
readiness.

The completed contract satisfies every cited Phase 1 requirement through one
presentation owner consuming one immutable C10 response. Aggregate ranking
does not depend on T/Q, browser health, or presentation. The fixed V1 outcome
remains a private/local independently runnable scanner and dashboard; public
deployment and separately authorized market-hours validation remain outside
the completion claim.
