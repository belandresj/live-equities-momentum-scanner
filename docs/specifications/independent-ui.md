# Independent UI

**Status:** Historical Component 11 evidence remains accepted for the
superseded v1 field set. Section 20 is the current executable MVP-S3
reconciliation contract for API v2 and the final dashboard.

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
| `C11-S1` API/view-state integration | `accepted_after_degraded_current_correction` | The validator accepts the third backend-ready mode, reserves green/current presentation for `qualified_current`, rejects any partial publication that exposes T/Q membership or measurements, and presents `degraded_current` as explicit current-data partial ranking. The exact sealed HTTP JSON from the Runtime composition passes the production UI model; focused final review is clean. | Complete |
| `C11-S2` visual/interaction/accessibility | `accepted` | `P-C11-VISUAL` passes in production Chrome at 1440x900: six required states, exact 12-column/20-row fit, computed contrast/focus/semantics, safe hostile text, delayed/disconnected freeze, exact-origin CORS, and UI-only restart all pass; final-review corrections did not invalidate those observations | Complete |
| Final component review | `accepted_after_correction` | Mandatory read-only review found the ready/degraded and sample-identity boundary defects; both were corrected with distinguishing proofs, and two focused same-reviewer re-reviews found no remaining P1/P2 | Complete; integrated private V1 RC later accepted |

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
  displayed as 80%, not 0.8% or 80.00 points. Day %, From 4AM %, and HOD
  drawdown retain two decimals; range positions and Activity use whole percents;
- range positions are not numerically clamped by the browser;
- Tape Rate shows only five-second trades/second. The API may retain the
  one-second fact, but the dashboard does not display it. Keyboard/focus detail
  exposes trade coverage, participant/SIP/mixed timestamp basis, lifecycle-record observation,
  and the exact status/reason so a current number cannot imply fully corrected
  consolidated tape;
- Spread shows basis points and cents on one slash-separated line for both
  current and stale retained valid quotes. A secondary label shows quote age
  and marks stale quotes; keyboard/focus detail always exposes quote coverage,
  age, quality, and exact status/reason;
- UTC timestamps are rendered in the browser locale only after valid parsing;
  durations/counters retain the API units and decimal strings are never coerced
  through an unsafe JavaScript integer.

Genuine numeric zero remains visible. A non-current status hides its numeric
slot and shows its exact status/reason; color never overrides that state.
Day %, From 4AM %, and HOD drawdown remain numerically uncolored because equal
magnitudes can have different symbol-relative meaning. The three range-position
columns use continuous linear color interpolation from red at 0%, through
neutral gray at 50%, to green at 100%; intermediate percentages have no
categorical cutoff or implied statistical threshold. Out-of-range numbers stay
visible while their color uses the nearest endpoint. Activity uses continuous
gray-to-orange intensity across its defined 0–100 percentile-derived scale.
Tape uses the current product-approved absolute-rate gradient from 0 through
500 five-second trades/second; rates above 500 retain their number and use the
endpoint color. Spread uses the current product-approved continuous absolute-
bps neutral-to-amber/orange-to-red text gradient through 400 bps; values above
400 retain their number and use the endpoint color. Values outside a visual
scale remain numerically visible.
These presentation scales do not create
alerts, qualification, ranking, readiness, or capacity claims.

## 10. Authoritative view-state matrix

One captured response produces one render model. Before state selection, the
client enforces the C10 normative type/null, rank/order, positive response-
sample identity, decimal-string,
full-accounting-identity, publication/fence, lifecycle/status, and readiness
relationships. Unknown additive root or nested properties are ignored for
within-major compatibility; required properties and known meanings remain
mandatory. Any contradictory known facts reject the whole response atomically.
In particular, backend-ready coherence requires `process_live`,
`backend_ready`, and `ranking_current` true, `ranking.mode` equal to
`qualified_current`, `degraded_bootstrap`, or `degraded_current`, a present committed watermark/lag,
valid accounting, and a C10-permitted live/hydrating lifecycle. Only
`qualified_current` receives the green/current presentation. A legal
backend-ready `degraded_bootstrap` response is accepted without changing the
server readiness fact, but its rows and publication remain prominently
non-qualified/degraded in the presentation. Its connected primary label is
`DEGRADED`, not the false claim `NONCURRENT`; transport delay or loss still
takes precedence as `REFRESH DELAYED` or `FROZEN · DISCONNECTED`.
An accepted `degraded_current` response uses the distinct `PARTIAL · CURRENT
DATA` label and a persistent explanation that rows are trusted current marks in
raw Day-% order and do not assert qualification. Its table never receives the
green qualified-current presentation.
Any known noncurrent fact takes fail-closed precedence; a response claiming
ready/current simultaneously with ended, suppressed, stale, unavailable,
missing-watermark, invalid accounting, or false process-live is invalid rather
than green. The following legal states are distinguishable without browser
inference:

| C10 fact | Required UI consequence |
| --- | --- |
| All backend-ready coherence predicates above with `ranking.mode=qualified_current` | Current status; render 0..20 rows in exact order. |
| Qualified current with zero rows | Explicit “No symbols currently qualify”; never loading or error. |
| Qualified current with 1..19 rows | Render only those rows; no placeholders or client backfill. |
| `ranking.mode=degraded_bootstrap`, including legal backend-ready output | Preserve the server backend-ready fact but show a prominent incomplete-population band; rows may be shown only as degraded, never qualified current. |
| `ranking.mode=degraded_current` with backend readiness | Preserve the server currentness fact, show `PARTIAL · CURRENT DATA`, state that qualification is not asserted, render the server's raw trusted-mark order without client filtering, and keep T/Q unselected. |
| `stale`, `suppressed`, `ended`, or `unavailable` ranking/lifecycle | Exact server mode/reason in a persistent noncurrent band; retained rows, if supplied, remain visibly noncurrent. |
| Aggregate field `warming`, `unavailable`, or `invalid` | Per-cell status token and keyboard/focus-accessible reason; no fabricated zero. |
| T/Q `warming`, `unavailable`, `invalid`, `pressure_shed`, or uncovered | Tape/Spread cell independently noncurrent; aggregate rank and aggregate fields remain intact. Tape detail still states coverage, timestamp basis, and lifecycle observation; Spread detail still states coverage, duration, and quality. |
| `tq.aggregate_only`, degraded pressure, unknown membership, or retained-bound hit | Separate T/Q health band and affected cell state; never downgrade aggregate ranking unless C10 says so. While pressure is nonnormal, show its engine-owned cause plus the last accepted oldest-waiting age and `healthy/required` recovery progress, or explicitly say the recovery sample is unavailable. Do not recompute pressure in the browser. |
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
Rank, Symbol, Last, Day %, From 4AM %, HOD DD %, Day Range %, 60 MIN Range %, 30
MIN Range %, Activity, Tape Rate, Spread. Stable compact rows use tabular numerals,
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

`C11-S1` implements one pure `scanner.snapshot.v1` validator/view model, one
bounded nonoverlapping poller, one text-only atomic renderer, and a separate
loopback static server. `P-C11-STATE` passes the full named corpus, including
ratio/Activity units, exact order/count, all field/TQ states and trust details,
currentness contradictions, every wire accounting identity, additive fields,
malicious strings, response bounds, delayed/timeout/frozen/reconnect behavior,
and same/new-publication replacement. The server proof rejects public bind,
nonloopback/path-bearing API origins, unknown routes, symlink escape, oversize
assets, and unjoined shutdown while proving exact known assets/config/CSP/HEAD.
Ordinary repository and affected race verification pass. The coherent S1
success path is one bounded C10 body to one validated detached view; malformed,
incoherent, unsafe, over-bound, or timed-out inputs cannot partially replace it.
S2 visual density, computed accessibility/contrast, Chrome screenshots, and UI-
restart evidence remain deferred exactly as allocated.

The initial S1 implementation review rejected known-invalid T/Q tuples and
hydration fences reaching current, current-looking frozen rows/empty claims,
dropped process/membership/bound facts, sequential partial DOM mutation, and
loose calendar/timestamp parsing. Corrections enforce exact known aggregate/TQ
trust tuples while preserving unknown-reason compatibility, require positive
same-epoch reconciled fences, downgrade the entire retained table and empty
message, expose every bounded V1 family plus row membership/mark age, build one
detached document before a single swap and acknowledge snapshots only after the
swap, and parse canonical UTC RFC3339Nano/calendar dates. The expanded proof
injects each dangerous counterexample, actual detached DOM failure, readiness-
expired/refresh-delayed/disconnected styling, and unrendered-snapshot failure.
Focused re-review then found operations-sample accounting incorrectly vetoed
C10 readiness, unknown Tape meanings still exposed inner rates, and fractional
timestamps accepted noncanonical trailing zeros. The final correction renders
operations accounting as diagnostics only, gates both rates on known-current
outer Tape authority, and matches C10 canonical RFC3339Nano. All 14 focused
tests, the server proof, ordinary repository verification, and affected race
verification pass; final focused re-review found no remaining P1/P2. S1 is
accepted and its S2 boundary remains valid.

A 2026-08-09 manual production-Chrome attempt then reopened S1's transport
claim: before sending its first request, `PollController` stored native
`Window.fetch` and invoked it as a controller method, so Chrome supplied the
controller as the receiver and rejected every poll with `Illegal invocation`.
The fixture API and generated API origin were healthy, distinguishing this
from CORS, replay, and backend availability. The production default now binds
`globalThis.fetch` to `globalThis`; an injected fetch remains unchanged for
deterministic boundaries. A new receiver-sensitive regression would fail under
the original method call and proves a connected current snapshot under the
correct receiver. All 20 focused tests pass. This correction changes no API,
polling cadence, state meaning, or market logic; corrected real-Chrome
observation remains allocated to `P-C11-VISUAL`.

Mandatory final review on 2026-08-09 reopened S1 again after distinguishing two
cross-component boundary cases that the prior corpus omitted. First, the
validator treated `qualified_current` as the only legal backend-ready ranking
mode, although `LIFE-PUBLISH-02` permits a causally current, backend-ready
`degraded_bootstrap` publication. The corrected validator accepts exactly those
two ready ranking modes; the view model still reserves green/current
presentation for `qualified_current`, preserves the backend-ready server fact,
and renders ready/degraded rows under the distinct `DEGRADED` primary label with
`degraded_bootstrap · incomplete_population`. Second, the response-sample
validator now rejects identity `0`, matching C10's positive decimal sequence
contract rather than merely its uint64 shape. Direct regressions exercise both
dangerous counterexamples. All 20 focused tests, `go test -short -timeout 2m
./...`, `go test -race -short -timeout 5m ./internal/ui ./cmd/dashboard
-count=1`, and `git diff --check` pass. The retained Chrome proof is unaffected:
these findings concern omitted wire-validity distinctions, not its measured
layout, accessibility, transport, CORS, or restart observations.

`C11-S2` now implements the fixed presentation bands, compact 20-row desktop
allocation, eight-part status hierarchy, retained/noncurrent suppression,
WCAG-AA token palette, reduced-motion override, semantic table, persistent
polite announcer, native disclosure, and stable symbol/field focus restoration.
Its deterministic fixture API is a separate loopback proof process; all current,
empty, degraded, T/Q-degraded, hostile, hang, and error bodies pass the
production validator before serving, sample identity advances independently of
unchanged publication identity, and the degraded corpus explicitly reconciles
20 known rows plus one unresolved bootstrap failure. Static proof and focused
correction re-review are clean after fixing alpha-composed retained contrast,
status-strip wrapping, fixture population, persistent interaction state, rank-
stable focus, and vanished-row fallback.

Owner visual review on 2026-08-09 revised the lower-level palette without
changing any market meaning: Day %, From 4AM %, and HOD drawdown no longer
receive magnitude fills; Day, 60 MIN, and 30 MIN range positions use a dedicated
continuous red-to-neutral-to-green directional scale with no intermediate
cutoffs; and the synthetic current fixture is
Day-% descending so it no longer visually contradicts the server-ranking
contract. A follow-up owner revision makes Activity and five-second Tape Rate
continuous gray-to-orange intensity scales, renders only the five-second Tape
Rate and inline Spread bps/cents, and removes false decimal precision
from range position and Activity. Spread now uses the current continuous
absolute-bps neutral-to-amber/orange-to-red text gradient rather than cell
bands. Focused
model, palette, contrast, and fixture
proofs distinguish these presentation-only decisions; C10 remains the sole
ranking and value owner.

`P-C11-VISUAL` passed on 2026-08-09 through the production static server and
poller in current Chrome at the exact 1440x900 CSS viewport and 100% zoom. The
current-20 surface rendered all 12 columns and 20 rows inside one viewport:
the table occupied `x=[20,1420]`, `y=[148,819]`, the table shell had equal
1,400-pixel client/scroll widths, and the document had no horizontal or
vertical overflow. Exact-empty, aggregate-degraded, T/Q-degraded,
`refresh_delayed`, disconnected/frozen, and hostile-string states were captured
from the deterministic loopback C10 fixture. Genuine current zero remained
`0.00%`; pressure-shed Tape/Spread values were hidden as em dashes while their
complete coverage, timestamp/duration/quality, lifecycle, membership, status,
and reason details remained focus-accessible. The hostile symbol was literal
text, produced no image/HTML node, and its unavailable Activity remained an
explicit dash plus `history_incomplete`.

Browser-computed inspection found one caption, 12 scoped column headers, 20
consecutive server ranks, a persistent polite announcer, native keyboard-
operable disclosure, and visible two-pixel focus outlines on the disclosure and
Tape/Spread cells. All 521 visible text/token samples met WCAG AA under their
computed colors; the minimum recorded contrast was 6.18:1. The reduced-motion
media rule was present, and the current surface had no nonzero animation,
transition, or smooth-scroll duration. During a hung fixture response, the
surface first became `refresh_delayed` and atomically retained noncurrent rows,
then became disconnected/frozen after the bounded request timeout without
changing the retained sample. Stopping only the UI server left the fixture API
reachable with exact-origin CORS, schema `scanner.snapshot.v1`, publication 20,
and 20 rows; restarting and reloading only the UI reconnected from sample 176
to sample 204 while preserving publication 20 and all 20 rows. Chrome logged no
warning or error.

The focused `node --test ui/model.test.mjs ui/visual.test.mjs` run passes all 20
tests; `go test -short -timeout 2m ./...` and `go test -race -short -timeout 5m
./internal/ui ./cmd/dashboard -count=1` are clean. The proof is deterministic
local Chrome/loopback evidence only: it establishes neither another browser,
mobile layout, provider/market-hours behavior, public-network behavior, formal
accessibility certification, nor an SLA. No production code, C10 meaning,
market calculation, ranking order, poll cadence, or ownership changed. C11-S2
is accepted. The mandatory final review and focused correction re-reviews are
clean; Component 11 is finally accepted. The later integrated private V1 RC
production-path proof and read-only review are also accepted.

## 20. Current MVP-S3 reconciliation contract — 2026-08-14

This section is the current executable UI contract under
[`live-feature-mvp-program.md`](../live-feature-mvp-program.md). It supersedes
the v1 schema, field list, column layout, fixture, and acceptance statements in
Sections 9, 12, 14-19 only. The independent process boundary, bounded
nonoverlapping transport, atomic validation/rendering, safe text handling,
desktop density, keyboard/focus preservation, accessibility, and failure
containment above remain applicable unless this section says otherwise.

**Controlling requirements:** `PG-AVAIL-01` through `PG-AVAIL-03`,
`PG-UI-01` through `PG-UI-03`, and the dashboard/accounting acceptance bullets
in Product Goals Section 14. The accepted
[`versioned-snapshot-api.md`](versioned-snapshot-api.md) is the sole wire
contract. MVP-S3 changes presentation only; it cannot change qualification,
Day-%/symbol ordering, backend readiness, field availability, T/Q membership,
or any market calculation.

**Outcome and non-scope:** Reconcile the existing independently runnable
Chrome-desktop dashboard to `scanner.snapshot.v2` and
`GET /api/v2/snapshot`, primarily by retaining the accepted shell, status
hierarchy, polling, interaction, accessibility, and dense-table implementation
while removing superseded columns and adding the approved semantic column
groups. The secondary `Operational details` disclosure is intentionally not
rendered; backend operations/accounting facts remain available through the
snapshot/API for readiness, incident analysis, and future tooling. Do not add
sorting, filtering, alerts, Float turnover, browser thresholds, browser
market-state inference, a UI framework, mobile or multi-browser certification,
public hosting, authentication, provider access, MVP-S4 integration claims,
replay work, checkpoint work, or scanner/backend production changes.

### 20.1 Exact grouped table contract

The table has a two-row semantic header and exactly these 12 leaf columns in
this order. A compact Rank column immediately precedes Symbol; there is no
separate rank-delta column. API row order is preserved exactly, and rank remains
a validated server-owned ordering fact. Rank's fixed-width metadata area shows
the displayed array position as `#1` through `#N` and, when available, rolling
60-second movement. Symbol contains only the ticker beneath its own header.

| Group | Columns | Presentation meaning |
| --- | --- | --- |
| `CONTEXT` | `RANK`, `SYMBOL`, `FLOAT`, `VOLUME`, `LAST` | Predominantly neutral ordering, identity, tradable-supply context, cumulative session participation, and current aggregate mark. |
| `LOCATION` | `FROM CLOSE %`, `FROM OPEN %`, `DAY RANGE` | Where the symbol is relative to the adjusted prior close, session open, and session range. From Close % is the presentation label for the canonical Day-% field and uses the product-approved displayed-set-relative dark-to-neon green scale; Day Range retains its red-neutral-green location scale. |
| `MOMENTUM` | `ACTIVITY 30s`, `MOVE 30s` | Recent aggregate participation and price movement. Activity uses gray-to-orange attention; Move uses the product-approved continuous signed red-to-gray-to-green RGB scale, with color applied only to the numeric text. |
| `TAPE / EXECUTION` | `TAPE SPEED`, `SPREAD` | Selected-row transaction activity and NBBO friction. Tape Speed is the presentation label for canonical Tape 5s and uses the product-approved continuous absolute-rate gradient from neutral gray at 0–50 trades/s through restrained/bright orange at 100/250 trades/s to `#FF8A00` at 500 trades/s and above. Spread uses the product-approved continuous absolute-bps neutral-to-amber/orange-to-red gradient through `#FC0000` at 400 bps; both treatments color only numeric text. |

Group labels, visible separators, scoped leaf headers, and non-color text
meaning must make these four scanning questions apparent without adding data
columns. At the 1440x900, 100%-zoom Chrome target, the grouped header, status
strip, and up to 20 rows fit without pagination or document-level horizontal
or vertical overflow. The accepted 1120px content floor and below-target
horizontal-scroll nonclaim remain unchanged.

Formatting is presentation-only. Reuse the existing safe formatters and
tabular-number treatment where compatible: compact Float and Volume share
counts; magnitude-sensitive Last USD; From Close %, From Open %, and Move 30s
as signed percentages; Day Range and Activity 30s as percentages; Tape Speed
as trades/second; and Spread as bps/cents. Genuine numeric zero is visible.
Current values render normally. Stale Float or Spread may render retained
numeric values only when API v2 supplies their exact valid stale tuple and must
carry an explicit stale label plus provenance/age detail. Warming,
unavailable, invalid, and pressure-shed values never become zero. Every field
status/reason, Float provider/effective date/retrieval/provenance, Tape coverage
and timestamp/lifecycle qualifiers, and Spread coverage/age/quality remains
available through the existing semantic cell labels and keyboard focus; they
are not rendered as hover tooltips on data cells.

Each leaf header carries one short plain-language description in a
pointer-hover-only custom tooltip. The renderer uses a `data-tooltip` value and
CSS pseudo-element with an 80 ms visual transition; it does not use the browser
native `title` tooltip, does not make headers tabbable, and does not show a
tooltip when a data cell is hovered. The header labels are `FROM CLOSE %` for
the canonical Day-% field and `TAPE SPEED` for canonical Tape 5s; API field
names and measurement semantics remain unchanged.

After validating one complete API v2 response, the view-model transformation
uses one shared displayed-set-relative normalization helper over that response's
exact `rows` array. The Day-% selector returns `row.day_change_ratio`; the From
Open selector returns `row.from_open_change.value_ratio` only when that field's
status is `current`, otherwise it returns `null`. The helper retains only finite
positive selected values for the green extrema and returns `null` positions for
zero, negative, invalid, or excluded values. For positive Day % its formula is
`(D_i-D_min)/(D_max-D_min)`; for positive From Open it is
`(F_i-F_min)/(F_max-F_min)`, clamped only for floating-point residue. The
current qualified table interpolates positive values from `#4A965D` at zero to
`#2CFF05` at one using the same continuous CSS `color-mix(in oklab, ...)`
palette. A singleton or zero-width positive range maps every positive value to
the maximum endpoint `1`; equal values and ties share one position; fewer-than-
20 snapshots use only their displayed positive extrema; and no positive From
Open values produce no From Open positions. Current zero values use neutral
gray `#8F9AA3`, current negative values use fixed muted red `#C46B6B`, and
warming, unavailable, invalid, and other non-current From Open fields retain
their state, reason, text, and no-color presentation.
The transform does not sort, filter, backfill, or recheck server order, and it
does not retain extrema across snapshots. Noncurrent/retained table styling
continues to take precedence over either relative-green palette. Missing or
nonfinite Day % fails the existing whole-response validation; the numeric field
text and underlying ratios are never altered by this presentation metadata.

The poll controller retains at most 70 current-ranking samples containing only
`sampled_at` and a map from displayed symbols to their 1-based array positions.
For each new current snapshot it chooses the retained sample closest to
`sampled_at - 60s`, provided history reaches that target and the selected
sample is within five seconds. Movement is `prior rank - current rank`:
`1..5` renders `↑1..↑5` or `↓1..↓5`, larger magnitudes render `↑5+` or `↓5+`,
and zero renders nothing. Absence from a valid comparison snapshot renders
`↑ NEW`; lack of a valid comparison during startup or a history gap renders
nothing. Binding/date changes and timestamp regression clear the history. The
metadata calculation consumes the validated order after every successful poll
and does not sort, debounce, smooth, delay, or suppress that order.

### 20.2 API v2 view and transport boundary

The client accepts only `scanner.snapshot.v2` from `/api/v2/snapshot`. It
validates the response and swaps one detached render model atomically. It
preserves the server row order, validates consecutive rank identities, renders
current array-position metadata in Rank, rejects duplicate or empty
symbols and more than 20 rows, and does not sort, filter, join, clamp,
calculate readiness, infer availability,
or reconstruct a missing value. Exact API v2 status/reason/value and Float
provenance tuples are enforced before rendering; unknown incompatible schema
or contradictory known facts fail closed and retain the last valid snapshot as
explicitly frozen.

Retain the accepted exact-loopback origin validation, CORS assumptions,
one-second nonoverlapping polling, three-second request bound,
`refresh_delayed` and disconnected/frozen behavior, response-size bound,
sample/publication atomicity, text-node rendering, UI-only restart behavior,
status hierarchy, focus restoration, reduced-motion behavior, and WCAG-AA token
requirements. Remove v1-only field validation, formatting, the operational
details DOM/flattened view-model representation, fixtures, and headers rather
than retaining hidden parallel representations. Field-specific reasons and
provenance remain focus-accessible on their relevant table cells.

The compact top-right status control is the sole primary status surface. Its
always-visible summary translates the validated primary publication/transport
state into `CONNECTING`, `WAITING FOR SESSION`, `WARMING UP`, `LIVE`, `PARTIAL`,
`RECOVERING`, `DELAYED`, `DISCONNECTED`, `UNAVAILABLE`, `SESSION ENDED`, or
`HISTORICAL` and omits the redundant displayed-row count. `LIVE` requires
connected exact qualified-current output; an exact zero-row result remains
`LIVE`. When T/Q pressure, shedding, a retained-bound hit, or unknown selected-
row coverage makes Tape Speed or Spread incomplete, the summary additionally
exposes the trader-facing `TAPE / QUOTES DEGRADED` warning in amber without
demoting the aggregate scanner state. The native keyboard-operable disclosure
uses four trader-facing rows: Scanner, Aggregates, Trades, and Quotes. Aggregate
startup/recovery detail retains exact work progress; Trades and Quotes report
separate selected-row current/warming/stale/shed/unavailable/invalid counts.
Raw backend-readiness and ranking-mode labels and a separate dashboard-feed row
are not displayed. Connection problems remain visible in the summary's
disconnected or delayed state; normal transport is not rendered separately.
Process liveness, operational sample accounting, sample identity/time, and the
raw aggregate watermark remain available through the validated API and future
diagnostic tooling. The dashboard renders no broader Operational Details
disclosure.

The static dashboard shell, compact status control, semantic table, column
groups, leaf headers, and header-tooltip anchors are created once. Each later
validated render model is prepared off-DOM and committed synchronously by
updating status/message content and replacing the table rows. The table header
node therefore keeps identity across one-second samples, so an active custom
header tooltip is not destroyed by polling. Row replacement remains one-model
at a time, retains exact server order, and restores keyed cell focus where the
same symbol/field remains available. A preparation or render failure leaves the
previous coherent status/messages/rows intact.

During noncurrent live recovery, the status summary and Aggregates detail use the
immutable API lifecycle, connection acknowledgement,
`recovery.generation_active`, work accounting, and fence facts only for
presentation. The summary consolidates reconnect, resubscribe, preparation,
active work, retry, and fence finalization as `RECOVERING`; Aggregates names the
specific trader-facing step. Active work shows exact terminal/planned progress.
A retained inactive prior ledger is never shown as current progress, and
recovery never becomes `LIVE` until the ordinary backend readiness fact does.
The summary similarly consolidates connection, subscription, hydration, and
fence finalization as `WARMING UP`, with the specific step in Aggregates.
During an independent rolling update, a v2 response that predates
`generation_active` is conservatively treated as inactive and presented as
preparing rather than fabricated progress. These labels add no browser-owned
lifecycle or readiness decision.

### 20.3 One implementation slice and primary proof

MVP-S3 is one write-capable implementation slice because the accepted UI
owner, runtime boundary, and interaction model remain unchanged. Allowed
production boundaries are `ui/`, `internal/ui/`, and `cmd/dashboard/`; README
run instructions, deterministic fixtures/tests, and this sole ledger may be
updated as necessary. Dashboard work must not alter scanner production code or
API v2 market meaning.

`P-MVP-UI` is the primary proof. Its smallest deterministic API v2 corpus
covers 20, fewer-than-20, and exact-empty rows; current and noncurrent
publications; independent current/stale/warming/unavailable/invalid/
pressure-shed fields; genuine zero; fresh, cached, missing, malformed, and
unavailable Float; positive/negative Move; low/high Activity and Tape; retained
stale Spread; hostile literal text; delayed, disconnected, and recovered
transport; same-publication resampling; new-publication atomic replacement;
and focus preservation when rows persist, reorder, or disappear. It asserts
the exact four group labels, 12-column order, distinct Rank and Symbol columns,
compact current rank and rolling movement in Rank, compact Float/Volume formatting,
exact displayed-set-relative Day-% positions including
the `100%`, `20%`, `10%` example (`1`, approximately `.11`, `0`), exact
displayed-set-relative From-Open positions including the `100%`, `20%`, `-10%`
example (`1`, approximately `.2727`, `0`), fewer/equal/tied cases,
current-only participation, noncurrent precedence, no client sorting or
market-state calculation, and complete focus-accessible reasons/provenance.
It also asserts the renamed `FROM CLOSE %` and `TAPE SPEED` headers, all 11
plain-language header descriptions, the pointer-hover-only custom tooltip
behavior, stable header-node identity across consecutive samples, the compact
status summary/disclosure hierarchy, and the absence of native data-cell
`title` tooltips.

The bounded production-Chrome proof runs through the real dashboard server and
poller at 1440x900. It verifies one-viewport density, semantic grouped headers,
scoped leaf headers, table caption, polite live status, keyboard-operable
field focus, visible focus, reduced motion, WCAG-AA contrast, safe hostile text,
T/Q-only degradation, transport freeze/recovery, exact-origin CORS, and UI-only
restart while the fixture API remains reachable. This is deterministic
loopback evidence, not a live-provider, market-hours, browser-matrix, formal
accessibility-audit, public-network, or performance-SLA claim.

Implementation acceptance requires focused model/render/server tests, the
narrow Chrome proof, ordinary `go test -count=1 -short -timeout 2m ./...`, the
affected `internal/ui` and `cmd/dashboard` race tier with an explicit timeout
no greater than five minutes, `git diff --check`, and one final independent
read-only review with no unresolved P1/P2 findings. Record coherent behavior,
dangerous false-current and fabricated-zero counterexamples, limitations, and
review result here before marking MVP-S3 accepted. MVP-S4 does not begin as
part of this slice.

### 20.4 MVP-S3 acceptance record — 2026-08-14

`MVP-S3` is accepted. The independently runnable dashboard consumes only
`scanner.snapshot.v2` from `GET /api/v2/snapshot`, preserves the API row order
while validating consecutive rank ordinals, and renders no Rank column. The
two-row table header contains exactly the four `CONTEXT`, `LOCATION`, `MOMENTUM`,
and `TAPE / EXECUTION` groups and the 11 contracted leaf columns. The
accepted shell, bounded nonoverlapping poller, compact status hierarchy, safe
detached replacement, focus restoration, live region, reduced-motion rule, and
desktop density remain in the production path. The backend operations and
accounting facts remain validated and available in the snapshot, while the
secondary operational-details disclosure is absent from the dashboard.

`P-MVP-UI` passes against the deterministic v2 corpus. Twenty, fewer-than-20,
and exact-empty populations remain exact; server order is never sorted or
filtered; rank, symbol uniqueness, row bounds, schema identity, publication
coherence, population identities, and field-specific status/reason/value
tuples fail closed before model construction. Genuine Volume and measurement
zero remains visible. Noncurrent fields render an em dash rather than zero.
Fresh and cached Float carries provider, effective date, retrieval time, and
provenance; a retained cached Float and stale Spread retain their numeric value
only under the exact stale tuple and show an explicit `stale` label. Float and
Volume use compact share formatting with unit-boundary promotion. Tape uses
the fixed absolute-rate gradient through `#FF8A00` at 500 trades/s, Spread uses
the fixed absolute-bps neutral-to-amber/orange-to-red gradient through
`#FC0000` at 400 bps, and Move uses the
validated raw sign for its continuous positive/neutral/negative presentation,
so exact zero is neutral. Every status-bearing cell is keyboard-focusable and exposes
its reason plus applicable Float, Tape, Spread, and membership provenance.

The dangerous false-current counterexample mutates a ready publication so its
covered/unresolved population totals contradict the universe and unknown
counts. The client rejects it rather than replacing the last valid snapshot.
The dangerous fabricated-zero counterexamples replace unavailable values with
null-bearing warming/unavailable/invalid/pressure-shed tuples; they remain
visibly unavailable and never become numeric zero. Further mutations reject
v1 schema identity, nonconsecutive or duplicate rank/symbol rows, unknown
field meaning, invalid Float provenance, incomplete stale Spread values, and a
missing current Tape value. A renderer failure leaves the prior DOM intact;
delayed or disconnected transport keeps the last valid publication explicitly
noncurrent until a new valid sample succeeds.

The bounded production Chrome proof used the real dashboard server and v2
fixture poller with a 1440x900 CSS viewport. Its document scroll extent remained
exactly 1440x900 with all 20 rows visible. The four scoped group headers had
colspans `4/3/2/2`; the 11 scoped leaf headers, caption, polite live status,
and absence of Rank were present. Keyboard navigation produced a visible 2px
focus outline and complete Tape details. T/Q pressure left aggregate ranking
current while Tape and Spread showed `pressure_shed`; hostile symbol text
created no HTML/image node; delayed and disconnected polling retained 20
frozen rows and recovered; a `localhost` origin different from the configured
`127.0.0.1` origin was CORS-blocked; and stopping and restarting only the UI
server reconnected to the still-running fixture API.

Verification passed with fresh results: `node --test ui/model.test.mjs
ui/visual.test.mjs` (13/13); focused `go test -count=1 -short -timeout 2m
./internal/ui ./cmd/dashboard`; full `go test -count=1 -short -timeout 2m
./...`; affected `go test -count=1 -race -short -timeout 5m ./internal/ui
./cmd/dashboard`; JavaScript syntax checks for the production modules and
fixture server; and `git diff --check`.

The required final read-only review used `gpt-5.6-sol` with medium reasoning.
It found two P2 defects: population coverage identities were not enforced
before a response could appear current, and exact-zero Move inherited the
positive/orange presentation. The correction added the missing coverage
identities plus the prior false-current mutation, and moved signed presentation
selection to the validated raw Move ratio with a neutral-zero render proof.
The reviewer also noted cosmetic compact-unit rollover at just below `1M` and
`1B`; that was corrected and tested. Focused re-review found no remaining P1
or P2 issue.

This evidence is deterministic private-loopback Chrome and local test evidence.
It does not establish live-provider wiring, market-hours behavior, a browser
matrix, mobile layout, a formal accessibility audit, public-network security,
performance SLA, replay support, checkpoint compatibility, or trading edge.
No credentials or provider requests were used. MVP-S4 has not started.

### 20.5 Day-% presentation correction acceptance — 2026-08-17

The dashboard now derives one continuous Day-% color position from the exact
validated rows in each render model. The distinguishing `100%`, `20%`, `10%`
fixture yields `1`, `1/9`, `0`; smaller populations use their actual extrema;
singletons and equal-value sets use the common midpoint; ties share a position;
and empty snapshots remain empty. The transformation preserves the API array
and never sorts, filters, or writes back to the snapshot. The renderer carries
the position only as presentation metadata, applies `#4A965D` through
`#2CFF05` only to current cells, and leaves degraded/noncurrent rows in the
existing retained color.

Focused JavaScript verification passed all 17 model/render/visual proofs, and
`git diff --check` passed. The ordinary repository command initially exposed a
single timing-sensitive failure in the already active operations recovery work;
the exact failed proof passed in isolation and the complete uncached repository
command then passed. A real dashboard/fixture-browser check at 1440x900 rendered
20 distinct monotonic Day-% colors with weights from `100%` to `0%`, retained
the exact 1440x900 document extent, and changed every Day-% cell to the existing
retained color under a degraded snapshot. No independent review was triggered:
this correction changes no trust, persistence, identity, concurrency,
ownership, ordering, or cross-component interface boundary.

### 20.6 From Open presentation correction acceptance — 2026-08-17

The dashboard now derives From Open % color positions in the same validated
snapshot-to-view-model pass as Day %. The shared helper uses only finite rows
with `from_open_change.status === "current"`; the `1.00`, `0.20`, `-0.10`
fixture yields `1`, approximately `0.2727272727`, and `0`. Smaller
populations use their own extrema, negative-only values remain value-relative,
singletons and equal ranges use `0.5`, ties share positions, and no eligible
values produce `null` positions. The exact API row order and displayed text are
preserved. Warming, unavailable, and invalid fields retain their state/reason
and em dash, while current field positions on degraded or retained tables stay
subordinate to the existing retained color through the current-table/current-
cell CSS guard.

The renderer exposes `from-open` palette metadata and percentage weights only
for eligible positions. It reuses the Day-% endpoint definitions and one
continuous `color-mix(in oklab, #4A965D, #2CFF05 ...)` implementation; no API,
backend, ranking, qualification, readiness, or publication behavior changed.
Focused JavaScript verification passed all 21 model/render/visual proofs, and
the requested mixed-state, no-eligible, fewer-row, singleton, equal, tie,
negative-only, order-preservation, metadata, noncurrent-precedence, Day-%
regression, endpoint, and WCAG-AA checks are covered. No independent review
regression, endpoint, and WCAG-AA checks are covered. The required uncached
`go test -count=1 -short -timeout 2m ./...` also passed. A deterministic
Chrome check against the authoritative assets rendered 20 rows at 1440x900
with an exact 1440x900 document extent, 20 monotonic From Open weights from
`100%` to `0%`, unchanged Day-% endpoint weights, no browser console errors,
and retained gray From Open cells under a degraded publication. No independent
review was triggered because this remains a presentation-only change with no
trust, ownership, ordering, or cross-component interface boundary change.

### 20.7 Header descriptions and label correction acceptance — 2026-08-17

The dashboard now uses `FROM CLOSE %` for the canonical Day-% field and
`TAPE SPEED` for canonical Tape 5s. All 11 leaf headers carry the approved
short descriptions. The custom CSS tooltip uses `data-tooltip`, appears after
an 80 ms transition, and does not use the browser-native `title` mechanism;
headers remain pointer-hover-only and are not made tabbable. Data cells no
longer expose status/reason text through native hover tooltips, while their
existing semantic labels and focus restoration remain intact.

The focused JavaScript suite passes all 36 model/render/visual proofs. The
local production dashboard and deterministic fixture were checked in the
browser: the header tooltip was visible after 150 ms with the expected text,
and a Volume data cell had no `title` attribute. No API, backend measurement,
ranking, qualification, readiness, or cross-component interface changed.

### 20.8 Compact dashboard structure and stable-header correction acceptance — 2026-08-17

The dashboard now renders the short `MOMENTUM SCANNER` title, one compact
top-right status summary/disclosure, and a framed scanner surface occupying the
remaining viewport. The permanent three-tile Backend/Ranking/T/Q band is
removed. The summary always shows exact publication/transport state and row
count; T/Q pressure, aggregate-only shedding, retained-bound state, or unknown
coverage adds the amber `TAPE / QUOTES DEGRADED` warning. Opening the native
disclosure shows backend readiness, ranking mode/reason, and the exact T/Q mode,
original pressure cause, oldest accepted waiting age, recovery progress, and
unknown count without changing aggregate currentness.

The renderer creates the dashboard shell, semantic table header, and custom
tooltip anchors once. For each later validated model it prepares complete rows
and messages off-DOM, commits those dynamic regions synchronously, and restores
keyed cell focus. Consecutive-render proof preserves exact header and status-
summary node identity and an open disclosure. The 1440x900 production fixture
kept all 20 rows in one viewport; the framed surface absorbed the remaining
height; degraded T/Q remained explicit; the native disclosure showed the exact
fixture facts; and the From Close tooltip remained visible after 1.7 seconds,
crossing a one-second poll. Browser console inspection found no warnings or
errors.

Independent review found one P2 inherited false-current boundary: a retained
model could outrank an explicitly delayed/disconnected render event. The
correction gives `refresh_delayed` and `disconnected` precedence over current
or partial model labels, requires connected transport for green primary state,
and forces retained table/cell styling at the render boundary. Exact current+
delayed, current+disconnected, and partial+disconnected regressions distinguish
the failure. Focused re-review found no remaining P1/P2.

Fresh verification passed all 37 model/render/visual proofs, focused
`internal/ui` and `cmd/dashboard` short tests, the full uncached repository
short tier, the affected race tier, and `git diff --check`. This correction
changes no API, polling cadence, market value, qualification, ranking,
readiness, backend ownership, provider behavior, replay, or checkpoint claim.

### 20.9 Compact rank and rolling movement correction implementation — 2026-08-17

The dashboard now renders a compact Rank column immediately before Symbol.
Each Rank cell contains a fixed-width 70px metadata block: subdued current
`#N`, then optional restrained green upward or red downward rolling movement.
The table has exactly 12 headers, with Symbol containing only the ticker beneath
its own header and no separate rank-delta column. The displayed ordinal is
assigned from the validated array position, and the current API row array is rendered immediately without
sorting, smoothing, debouncing, or delay.

`RankMovementHistory` is owned by the nonoverlapping poll controller and retains
at most 70 timestamp/rank maps. It chooses the closest current-ranking sample
within five seconds of `sampled_at - 60s`; values beyond five cap at `5+`.
Startup and history gaps remain blank, while absence from an available
comparison snapshot renders `↑ NEW`. Binding/date changes and timestamp
regression reset the presentation-local history.

The five rank-focused model/render/visual proofs pass, covering `#1..#N`, `#9 -> #4`,
`#10 -> #4`, `#4 -> #6`, `#2 -> #10`, unchanged, `NEW`, startup, alternating
immediate reorders, the 70-snapshot bound, and compact allocation. JavaScript
syntax checks, focused dashboard/server tests, and `git diff --check` pass. The
deterministic 1440x900 browser check rendered 20 rows, 12 headers, distinct
Rank and Symbol headers, identical ticker-cell start coordinates, exact viewport extent, and no
console warning/error. A concurrent user-owned Day/From-Open presentation edit
currently leaves five unrelated focused UI assertions failing, so final
correction acceptance and a whole-suite green claim remain pending worktree
convergence. The rank implementation changes no API, backend rank, ranking
algorithm, snapshot cadence, market calculation, readiness, replay, or
checkpoint claim.

### 20.10 Sign-aware Day/From Open presentation correction acceptance — 2026-08-17

The Day % and From Open % presentation scales now normalize finite positive
displayed values only. Positive minima use the existing dark-green endpoint
`#4A965D`; positive maxima use `#2CFF05`. Zero values use the existing neutral
gray `#8F9AA3`, and every negative value uses the existing muted red `#C46B6B`
without a negative gradient. A singleton or zero-width positive set maps every
positive value to the maximum endpoint; no-positive and non-current sets have
no green position. Numeric text, API values, row order, and strict Day-%
ranking remain unchanged.

The shared `relativeColorPositions` helper accepts positive-only normalization,
while the view model makes the sign-aware fixed-color decision and the renderer
applies it only to current Day/From Open cells. Focused JavaScript verification
passes all 41 model/render/visual proofs, including the requested mixed-sign
examples, positive-only extrema, no-positive and identical-positive cases,
fixed zero/negative colors, unavailable/invalid From Open states, noncurrent
precedence, unchanged text, and unchanged order. `go test -short -timeout 2m
./...` and `git diff --check` pass. No independent review is triggered because
this is a presentation-only change with no trust, ownership, ordering,
concurrency, API, or cross-component boundary change.

### 20.11 Trader-facing status vocabulary correction acceptance — 2026-08-17

The compact summary now maps validated lifecycle and transport facts to
`CONNECTING`, `WAITING FOR SESSION`, `WARMING UP`, `LIVE`, `PARTIAL`,
`RECOVERING`, `DELAYED`, `DISCONNECTED`, `UNAVAILABLE`, `SESSION ENDED`, or
`HISTORICAL`. `LIVE` requires connected exact qualified-current output and
remains correct for a resolved empty table. The displayed-row count and raw
backend/ranking vocabulary are removed. Delayed/disconnected transport retains
precedence over an otherwise current or partial model, and T/Q-only degradation
does not demote the aggregate scanner from `LIVE`.

The native disclosure contains Scanner, Aggregates, Trades, and Quotes only.
Aggregates translates the already validated connection, acknowledgement,
hydration, recovery, retry, and fence phase while preserving active work
progress. Trades and Quotes separately summarize selected-row current, warming,
stale, shed, unavailable, invalid, and unknown states. It does not add a
dashboard-feed row or reconstruct backend state from table values.

All 45 focused model/render/visual proofs pass, including exact lifecycle
mapping, an exact zero-row `LIVE` result, current/partial transport-precedence
counterexamples, T/Q-independent `LIVE`, separate coverage counts, and the
absence of the ranked-row count. The full repository short tier passes.
Deterministic browser inspection confirms the collapsed `LIVE` pill, the
four-row disclosure, and the independent `TAPE / QUOTES DEGRADED` warning with
separate shed counts; no browser warning or error was emitted. This correction
changes no API, polling cadence, backend readiness, qualification, rank,
provider behavior, replay, or checkpoint claim and does not trigger an
additional independent review.
