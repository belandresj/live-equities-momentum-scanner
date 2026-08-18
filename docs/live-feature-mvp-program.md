# Live feature-set MVP program

**Status:** Owner-approved current delivery authority, 2026-08-14.

**Scope:** The smallest sequential change that makes the private/local live
scanner compute, publish, and display the revised feature set for the interview
MVP. This program does not clean up the repository, prove replay, or repair
checkpoint persistence.

**Supersedes for this scope:** Conflicting replay, checkpoint, component-number,
and former-field delivery gates in the Version 1 Release Program and accepted
lower-level component contracts. Their still-compatible market-data,
qualification, ranking, ownership, availability, live-operations, API, and UI
decisions remain evidence and authority below the revised product contract.

## 1. MVP outcome

Done means the ordinary private/local **live** scanner:

1. preserves the existing eligible universe, qualification latch, exact
   qualified Day-%/symbol ordering, top-20 limit, canonical aggregate merge,
   REST/live reconciliation, committed watermark, and T/Q independence;
2. presents exactly:

   ```text
   CONTEXT                  LOCATION                     CURRENT MOMENTUM     EXECUTION
   SYMBOL FLOAT VOLUME LAST | FROM CLOSE % FROM OPEN % DAY RANGE | ACTIVITY 30s MOVE 30s | TAPE SPEED SPREAD
   ```

3. retains all-symbol aggregate and coverage evidence sufficient for Activity
   30s and Move 30s before selection, so a newly selected row is not warmed by
   membership;
4. computes every market measurement in the backend and exposes it through a
   versioned read-only snapshot API;
5. runs with checkpoint mode off and reconstructs state through the existing
   fresh hydration path after process restart;
6. keeps replay code present but makes no claim that replay currently works,
   performs acceptably, or reproduces the revised fields; and
7. passes bounded deterministic proofs plus the ordinary repository command.

One separately authorized or owner-executed market-hours observation may
confirm live provider wiring. Its absence does not convert replay into a gate
or justify credential access by an agent.

This is an interview MVP and private/local scanner, not a public deployment,
production SLA, checkpoint-restart release, replay product, or trading-edge
claim.

## 2. Fixed behavior

The following product behavior is fixed by
[`product-goals.md`](product/product-goals.md):

- existing aggregate qualification before ranking;
- exact Day % descending and exact-symbol ascending order;
- one authoritative `ScannerStateEngine`, canonical symbol state, committed
  aggregate watermark, evaluator, and immutable publication;
- full-universe aggregate inputs for Day %, Volume, From Open, Day Range,
  Activity 30s, and Move 30s;
- at least `[max(S,T-330s),T)` aggregate/coverage evidence plus the predecessor
  mark for every display-eligible symbol;
- selected-row T/Q coverage only after aggregate selection;
- independent field availability and honest no-print/unknown/conflict meaning;
- Float as optional dated Massive reference enrichment; and
- no browser-owned formulas, composite score, T/Q-to-ranking dependency, or
  fabricated history.

The MVP does not revise these meanings to reduce implementation work.

## 3. Live-only operating claim

### MVP-LIVE-01 — supported path

The only operating path claimed by this MVP is the ordinary live scanner with
fresh REST hydration, causally fenced live aggregate intake, same-process gap
recovery, selected-row T/Q enrichment, loopback snapshot API, and independent
dashboard.

The existing deterministic unit/component tests, fake provider, HTTP fixtures,
and UI fixture server remain valid proof tools. They are not product replay.

### MVP-LIVE-02 — market-hours evidence

Automated work must not access provider credentials or make a live request
without explicit authorization for that execution. The owner may run the
private scanner during market hours and record a bounded smoke observation.
Such an observation confirms only the exercised provider wiring and displayed
behavior; it does not prove broad capacity, latency distribution, or trading
edge.

## 4. Replay and checkpoint containment

### MVP-DEFER-REPLAY — present, unverified, non-gating

Replay implementation, commands, packages, schemas, and tests may remain in
the repository. This program allocates no work to repair, benchmark, redesign,
delete, or demonstrate them. Replay compatibility changes are permitted only
when mechanically required to keep the repository buildable after the live
feature changes.

No MVP acceptance statement may claim that replay:

- starts successfully on a retained artifact;
- completes within a useful time;
- reproduces the revised aggregate fields;
- exercises live T/Q behavior; or
- is a supported user workflow.

A failed or unknown replay path does not block this live MVP.

### MVP-DEFER-CHECKPOINT — retained code, disabled operation

Checkpoint code remains present and checkpoint mode remains off by default.
The MVP does not project, encode, restore, benchmark, repair, delete, or claim
restart equivalence for revised Volume, Activity 30s, Move 30s, Float, or their
retained evidence.

The private MVP launch uses `--checkpoint-mode off`. Once the revised feature
set is active, an attempted checkpoint-enabled launch must fail clearly unless
the installed checkpoint schema explicitly proves compatibility with every
revised field. It must never restore an old checkpoint and present incomplete
new state as current.

Process restart uses fresh hydration. Same-process aggregate-gap recovery
remains required and is not checkpoint behavior.

## 5. Implementation strategy and non-scope

The MVP uses explicit typed product fields in the existing backend. It may
calculate Activity 30s and Move 30s in the current full-universe evaluator;
post-selection materialization is permitted but not required. Performance
evidence, not stylistic preference, decides whether a two-phase evaluator is
needed later.

This document is the current executable delivery plan. Do not write a second
general refactor map or reopen every historical component before MVP-S1. Each
slice assignment must cite the relevant accepted architecture/dependency
contracts and may add only the narrow lower-level detail discovered to be
necessary for that slice.

The previous HOD drawdown, 30/60-minute range, composite Activity, and Tape
one-second output must not appear in the revised API or dashboard. Their
unreachable implementation and historical tests may remain temporarily when
removal is not required for correctness, capacity, compilation, or an
unambiguous public contract.

Out of scope:

- generic feature or plugin frameworks;
- engine/package reorganization;
- component-number renaming or historical-document cleanup;
- checkpoint performance or schema migration;
- replay correctness, performance, artifacts, CLI, or UI;
- public deployment, authentication, TLS, hosting, or production cutover;
- mobile or broad cross-browser work;
- Float Turnover or another derived/composite score; and
- unevidenced Spread or attention thresholds.

## 6. Sequential slices

Keep one write-capable implementation slice active. Each slice leaves the
repository buildable and records a final read-only review before capability
acceptance as required by `AGENTS.md`.

| Slice | Controlling product requirements |
| --- | --- |
| MVP-S1 | `PG-REFERENCE-02`, `PG-RANK-01` through `PG-RANK-04`, `PG-FEATURE-01` through `PG-FEATURE-07`, `PG-AVAIL-01` through `PG-AVAIL-03`, `PG-OPS-01`, `PG-OPS-02`, and `PG-OBS-01` through `PG-OBS-03` |
| MVP-S2 | `PG-AVAIL-01` through `PG-AVAIL-03`, `PG-UI-01`, and `PG-UI-02` |
| MVP-S3 | `PG-UI-01` through `PG-UI-03` and the exact displayed-field contract in product-goals Section 5 |
| MVP-S4 | Product-level acceptance in product-goals Section 14, bounded by `MVP-LIVE-01`, `MVP-LIVE-02`, `MVP-DEFER-REPLAY`, and `MVP-DEFER-CHECKPOINT` |

### MVP-S1 — live backend measurements and reference enrichment

**Outcome:** The engine/publication path can produce the final aggregate and
reference fields without changing qualification or rank.

In scope:

- bounded Massive Float list retrieval, pagination, validation, dated cache,
  and immutable lookup, proved with local HTTP fixtures;
- cumulative correction-aware session Volume for all symbols;
- From Open and Day Range under their revised names/meanings;
- Activity 30s using the exact target and 55 reference windows;
- Move 30s with known-no-print mark carry;
- all-symbol 330-second aggregate/coverage evidence and predecessor mark;
- independent status/reason results and aggregate-feature accounting;
- checkpoint-on compatibility rejection for the revised feature set; and
- removal of superseded fields from the revised engine row/publication view.

Primary proofs:

| Proof | Claim and dangerous counterexample | Observable success | Limitation |
| --- | --- | --- | --- |
| `P-MVP-FLOAT` | A partial, duplicate, malformed, stale, wrong-symbol, or failed Float response cannot become silently current or gate ranking. | Exact fresh/cache/unavailable facts, provenance, bounds, and unchanged rank/readiness. | No live provider request or disclosure-timeliness claim. |
| `P-MVP-VOLUME` | Inserts, revisions, withdrawals, no-print seconds, unknown history, and conflicts cannot produce a false complete session total. | Exact full-reference equivalence and field-local status at committed `T`. | No checkpoint equivalence. |
| `P-MVP-ACTIVITY-MOVE` | A symbol outside the top 20 for more than 330 seconds cannot enter with selection-dependent aggregate warm-up; overlapping windows, boundary marks, no-print carry, unknowns, and corrections remain exact. | Same values before/after selection from the exact 55-window oracle and boundary-mark oracle. | No replay claim. |
| `P-MVP-RANK` | New fields cannot filter, reorder, or fill rows. | Existing qualification and exact Day-%/symbol rows remain identical for the same canonical state. | Does not prove provider timing. |

### MVP-S2 — snapshot API v2

**Outcome:** One sealed publication maps to a private versioned schema carrying
only the final product fields and their independent states.

The API may retain a rank ordinal for validation/accessibility, but the visible
dashboard has no Rank column. The v2 row removes HOD, rolling ranges, old
Activity, and Tape one-second output. A breaking private schema cutover need
not preserve `/api/v1/snapshot` when no supported consumer remains.

Primary proof `P-MVP-API` uses a schema golden plus invalid-status/value,
ordering, zero, null, bound, CORS, and one-publication mapping cases. It proves
the wire contract, not backend market mathematics.

#### MVP-S1/MVP-S2 acceptance record — 2026-08-14

`MVP-S1` and `MVP-S2` are accepted. The acceptance correction established one
authoritative Tape 5s status/reason tuple from the engine through API v2,
rejected contradictory aggregate-ratio, Volume, Float, Tape, and Spread tuples
without erasing genuine numeric zero, and closed the production-path proof
gaps without changing qualification or Day-%/symbol ordering.

| Proof | Accepted evidence |
| --- | --- |
| `P-MVP-FLOAT` | Deterministic local HTTP/cache cases cover pagination cycles, changed-origin continuations, exact and exceeded page/result/record/response-byte bounds, future or malformed cache metadata, fallback to the last fully validated cache after partial refresh failure, and rank/readiness independence across fresh, cached, missing, malformed, and unavailable Float. |
| `P-MVP-VOLUME` | Real engine-owner insert, same-identity revision, accepted withdrawal, known no-print, unknown, and conflict paths reach newly sealed same-`T` publications. The revision replaces rather than double-counts; after withdrawal the canonical exact pre-`T` sum is `35`, proving the removed seven-share contribution is absent before the publication becomes `invalid/historical_conflict`. Qualification, membership, Day %, readiness, and unrelated fields remain unchanged. |
| `P-MVP-ACTIVITY-MOVE` | A symbol unselected for more than 330 seconds retains two present plus 328 proven-absent seconds, no unknown/conflict evidence, the `T-31s` predecessor mark, and all 55 non-overlapping reference windows. On Day-%/symbol-driven entry it immediately publishes Activity `100*54/55` and Move `100*(20/19-1)`; only selected-row Tape 5s and Spread begin coverage warm-up. |
| `P-MVP-RANK` | Competitor price/Day-% is the sole rank-change input in the selection proof. Float freshness/provenance variants preserve qualification, symbols, ordering, and T/Q intent. |
| `P-MVP-API` | API v2 maps the sealed engine Tape tuple for below-one-second coverage, one-through-less-than-five-second warm-up, current coverage, unavailable coverage, pressure shedding, and invalid unequal-repeat state. Exact field-specific tuple validation rejects missing or contradictory reasons, Float provenance mismatch, Tape warm-up reasons on current values, current Spread with `stale_quote`, and stale Spread without both retained numeric values; route/CORS, ordering, null, bounds, one-publication mapping, and genuine zero cases pass. |

The 2026-08-17 live recovery correction adds the already engine-owned
`scheduled_recovery` lifecycle reason to the API v2 and UI closed vocabularies.
Its production-path regression covers `suppressed/recovery_exhausted` through
the one-shot scheduled transition into `recovering/scheduled_recovery`, proving
that `/readyz` remains an honest 503 while `/api/v2/snapshot` remains a valid
HTTP 200 noncurrent publication. No recovery, ranking, readiness, or provider
behavior changed.

#### 2026-08-17 recovery observability correction

**State:** `accepted_correction`. The owner-run live observation recovered
through two aggregate epochs but exposed misleading terminal and dashboard
states; the correction now presents those states exactly and preserves the
first bounded cause even when recovery succeeds.

**Controlling requirements:** `PG-OPS-02`, `PG-OBS-02`, `PG-OBS-03`,
`PG-UI-01`, `PG-UI-02`, `LIFE-RECOVER-01` through `LIFE-RECOVER-06`, and
`LIFE-PUBLISH-02` through `LIFE-PUBLISH-03`.

**Observed evidence:** After an aggregate epoch loss from a ready publication,
the first gap-recovery generation completed 5,099 of 5,522 symbol requests and
then lost its replacement epoch. The engine correctly assigned the remaining
423 requests `canceled`, replanned the complete population under the next
acknowledged epoch, reconciled its fence, and returned to exact current
ranking. Before the first recovery plan existed, the terminal mislabeled the
completed prior bootstrap ledger as `Warm-up 5,522 / 5,522`; while either
recovery generation was active, the dashboard exposed only generic
`NONCURRENT/lifecycle_not_ready` even though API v2 carried the generation and
work counters.

**Boundary and non-scope:** The engine remains the sole owner of lifecycle,
generation activity, cancellation, coverage, fence, watermark, readiness, and
ranking. This correction exposes the already-owned active-generation fact in
the immutable operational publication/API, uses it to distinguish reconnect,
acknowledgement wait, active historical work, retry, and fence finalization in
the terminal and UI, and durably records/prints the first bounded redacted
ingress cause even when recovery succeeds. It does not change full-population
gap replanning, assemble coverage across failed epochs, tune heartbeat
deadlines, access credentials, change checkpoint/replay behavior, or alter any
market calculation.

**Primary proof `P-MVP-RECOVERY-OBS`:** The proof is a composed production-path
boundary, not a claim that a browser test owns transport semantics.
`TestC6RECOVER01SameProcessLossReackGapRetryExhaustion` supplies the
ready-to-loss, repeated-epoch, stale-fact-fencing, retry, fence, and ordinary
current-exit trace. The API regression proves the same engine-owned generation
activity reaches one immutable snapshot. Operator/UI regressions prove that an
inactive prior ledger cannot appear as 100% current work and that active retry
and fence-finalization states show the exact generation/counters. The scanner
coordinator regression sends one typed incident through terminal evidence,
one transient persistence failure, bounded retry, and the actual protected
JSON writer. Together they must preserve cancellation/retry honestly and
return to current only after the existing readiness predicates pass. The
dangerous counterexamples are treating retained completed accounting as active
work, hiding a repeated epoch loss behind generic noncurrent status, logging
credentials/provider payloads, or allowing presentation to infer currentness.
The proof establishes deterministic observability and containment, not the
cause of the live heartbeat failure or a transport SLA.

**Acceptance evidence:** No-cache focused short tests passed for `cmd/scanner`,
`internal/engine`, `internal/operations`, and `internal/snapshotapi`; all UI
model/render tests passed; the same four affected Go packages passed the race
tier; focused `go vet` and `git diff --check` passed. The required independent
read-only review found two P2 proof/durability gaps, both were corrected, and
focused re-review found no remaining issue. A full-repository short-tier
attempt also exposed the unchanged timing-sensitive massive transport test
`TestPC5TransportOneAttemptHandshakeHeartbeatAndContainment` subtest
`start_owns_progress_even_when_handshake_is_never_awaited`; it failed
repeatedly in isolation while no `internal/massive` source was changed here.
The one concurrent recovery-budget failure from that full run passed 10
consecutive isolated runs. Those failures do not invalidate this correction's
focused or race evidence, but the repository-wide ordinary tier is not
recorded as green.

#### 2026-08-17 T/Q recovery hysteresis and observability correction

**State:** `accepted_correction`. An owner-run market-hours observation of
the ordinary checkpoint-off scanner invalidated the claim that the accepted
T/Q recovery gate reliably restores normal selected-row coverage after a
transient host slowdown.

**Controlling requirements:** `PG-AVAIL-01`, `PG-AVAIL-03`, `PG-TAQ-01`,
`PG-TAQ-02`, `PG-OBS-03`, `LIFE-TQ-02`, and `LIFE-TQ-03`.

**Observed evidence:** The fresh process acknowledged provider membership for
all 20 desired symbols and applied 59,097 trades plus 26,025 quotes before
entering `taq_degraded` from two accepted `oldest_waiting_frame` samples at or
above the unchanged one-second entry boundary. Queue high-water was only
503/32,768 frames (1.54%), accounting remained coherent, pressure samples were
not missed, and aggregate ranking remained current. During a later bounded
20-sample observation the waiting queue was empty in six snapshots and never
exceeded 234 frames, yet pressure transitions remained exactly one and no T/Q
restoration command became eligible. This proves neither a host capacity SLA
nor the exact accepted-sample age distribution, but it does distinguish a
recovery lockout from provider subscription failure, queue-capacity loss, or a
stopped pressure sampler.

**Corrected boundary:** Keep transient degradation at oldest waiting age at
least one second for two consecutive accepted samples, severe escalation at
two seconds for three samples, the existing occupancy/watermark/loss gates,
and exact consecutive recovery. Change only the recovery oldest-waiting
predicate from below 250 ms to below 750 ms for five consecutive accepted
one-second samples. The 250-ms gap below entry retains hysteresis while
allowing a queue that repeatedly catches up after a transient stall to restore
fresh T/Q coverage. A sample at exactly 750 ms is unhealthy and resets the
recovery streak; intermittent healthy samples do not accumulate across renewed
pressure.

The engine additionally publishes the exact last accepted pressure sample
needed to explain recovery: waiting frames/capacity, waiting bytes/capacity,
oldest waiting age, aggregate watermark lag, whether that sample satisfied
every recovery predicate, the consecutive healthy count, and the required
count. API v2 carries those bounded scalars without re-evaluation. The UI T/Q
status reports the original pressure cause, accepted oldest-waiting age, and
`healthy/required` recovery progress while pressure is nonnormal. The browser
does not infer recovery or change aggregate currentness.

**Primary proof `P-MVP-TQ-RECOVERY`:** One engine trace proves below-750-ms
samples advance recovery, the exact 750-ms boundary resets it, five consecutive
healthy samples restore normal mode, and aggregate evaluation/watermark remain
unchanged. API mutation tests reject impossible sample/capacity/progress tuples,
and the UI model/render proof shows the exact engine-owned cause, age, and
progress without treating T/Q as a ranking gate. Focused engine, operations,
snapshot API, and UI tests plus ordinary verification are required. This
correction adds no queue, goroutine, mutable owner, T/Q-to-ranking dependency,
CPU/heap/delivery-latency gate, provider request, process restart, replay, or
checkpoint work.

**Acceptance evidence:** The exact 749-ms/750-ms engine boundary, consecutive
reset, missing-sample reset, five-sample normal transition, impossible
nonnormal `5/5`, aggregate-independence, API mapping/mutation, and dashboard
status proofs pass. Focused engine, operations, and snapshot API short tests;
all 31 UI model/visual tests; affected engine/operations/snapshot API race;
focused vet; and `git diff --check` pass. The first ordinary repository run
had one unrelated timing-sensitive private-launcher bootstrap test failure; its
exact isolated rerun passed, and the final uncached ordinary repository run
passed completely. Independent review found one P2 because engine/API/UI
validators initially admitted impossible nonnormal `5/5` progress. All three
validators and exact mutation proofs were corrected; focused re-review found
no remaining P1/P2. No post-change provider observation or host-capacity claim
is made, and the already-running scanner binary was not restarted.

#### Earlier MVP-S2/S3 acceptance evidence

Verification passed with no cached test results: focused affected packages under
the two-minute short tier; the full repository with
`go test -count=1 -short -timeout 2m ./...`; and the affected race tier with
`go test -count=1 -race -short -timeout 5m ./internal/reference ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner`.
The final withdrawal correction additionally passed its isolated 45-second
production-path test. `git diff --check` passed. The existing controlled
6,000-symbol measurement is reused because these corrections do not change the
full-universe evaluator algorithm or its asymptotic work.

The required independent read-only review used `gpt-5.6-sol` with medium
reasoning. Its P2 proof findings were that the sealed API path did not exercise
all six Tape 5s states, the Volume proof did not derive backend readiness and
left the withdrawal fixture in hydration, and the retained-selection proof did
not independently enumerate the exact target plus all 55 Activity reference
windows. The corrected tests now drive every Tape state through a real
engine/runtime publication and `/api/v2/snapshot`, prove a newly sealed same-`T`
Volume revision with unchanged readiness/rank fields and restore live lifecycle
after withdrawal, and assert the target and each `k=0..54` reference endpoint
and rate. Focused re-review reported no unresolved P1 or P2 findings.

Acceptance does not authorize credentials or live provider requests. Replay
remains unverified. Checkpoints remain disabled and non-gating. Dense-market
and live-GC capacity are not established beyond the exercised deterministic
fixture.

### MVP-S3 — final dashboard

**Outcome:** The independently runnable Chrome-desktop dashboard consumes API
v2 and presents the exact final column order, grouping, units, availability,
and visual grammar without market calculations.

Deterministic API/UI fixtures are the required weekend development path when
the market is closed. Using fixtures does not claim replay or live-provider
behavior.

Primary proof `P-MVP-UI` covers exact/fewer/empty rows, current/noncurrent
publication, independent field states, compact Float/Volume formatting,
continuous signed Move, relative Activity attention, absolute Tape-rate and
Spread execution-warning gradients,
keyboard/focus preservation, and no-color-only meaning.

#### 2026-08-17 Day-% value-relative presentation revision

The owner revised `PG-UI-03` so the current dashboard colors Day % by its
relative value within the exact displayed snapshot, not by ordinal rank. This
is a presentation-only correction in the existing API-v2-to-view-model layer:
the backend continues to own Day %, qualification, ordering, top-20 membership,
and immutable publication, while API v2 remains unchanged. `P-MVP-UI` adds the
`100%`, `20%`, `10%` distinguishing example, fewer-than-20, ties, zero-width,
empty, invalid-input, endpoint, noncurrent, and unchanged-order cases. The low
endpoint is contrast-safe dark green `#4A965D`, the high endpoint is neon green
`#2CFF05`, and a zero-width displayed range uses the common midpoint.

#### 2026-08-17 From Open value-relative presentation revision

The owner extended the same presentation-only displayed-set-relative green
scale to From Open %. MVP-S3 computes `(F_i-F_min)/(F_max-F_min)` in the
validated API-v2-to-view-model transformation using only finite rows whose
From Open status is `current`; it uses `0.5` for a singleton or zero-width
eligible range, preserves ties, uses actual extrema for fewer-than-20 rows, and
omits warming/unavailable/invalid values. The value is compared with other
displayed From Open values rather than with zero or row rank. The renderer
reuses the existing `#4A965D`/`#2CFF05` continuous CSS `oklab` palette, while
noncurrent/retained table styling remains authoritative. The row array, text,
ratio, field state/reason, API schema, backend ownership, ranking, readiness,
and publication behavior are unchanged.

`P-MVP-UI` covers the `1.00`, `0.20`, `-0.10` example (`1`, approximately
`.2727`, `0`), negative-only values, fewer rows, singleton/equal/tied/empty
sets, mixed field states, exact row order/text preservation, renderer weights,
noncurrent precedence, shared endpoint/accessibility checks, and the existing
Day-% regression. Focused JavaScript verification passed all 21 model/render/
visual proofs; the uncached ordinary repository command passed; and the
deterministic Chrome dashboard check rendered 20 rows at 1440x900 with an
exact 1440x900 document extent, monotonic From Open weights from `100%` to
`0%`, unchanged Day-% endpoints, and retained styling for degraded data. This
correction does not trigger independent review because it changes no trust,
persistence, identity, concurrency, ownership, ordering, or cross-component
interface boundary.

#### 2026-08-17 header description revision

The dashboard now presents the canonical Day-% field as `FROM CLOSE %` and the
canonical five-second Tape field as `TAPE SPEED`. Each leaf header carries a
short pointer-hover description rendered by the UI's custom tooltip treatment;
the browser-native `title` tooltip is not used. Data cells no longer show
status/reason tooltips on hover. The API fields, backend measurements, ranking,
qualification, and existing cell accessibility metadata remain unchanged.

#### 2026-08-17 live aggregate heartbeat and resubscription correction

The focused
[`live aggregate heartbeat and resubscription correction`](live-aggregate-heartbeat-and-resubscription-correction.md)
is accepted. Aggregate recovery now retires and joins each socket before the
next individually paced dial, preserves classified redacted handshake facts,
uses one finite consecutive-attempt budget, and remains stably suppressed with
no automatic dial after exhaustion. Heartbeat failure is nonterminal only when
a supported raw frame proves inbound progress across that operation's captured
boundary; a quiet deadline and an independent transport failure remain exact
terminal causes under first-cause arbitration.

Deterministic production-composition proofs cover temporary rejection through
successful fenced recovery, persistent five-attempt exhaustion, a startup
heartbeat deadline with exact old-generation cancellation and replacement
fresh bootstrap, and post-live loss of an active gap generation followed by a
replacement live tail and fence. The uncached ordinary repository tier,
affected race suite, vet, and diff checking passed. Final read-only review is
clean after correcting two missing production-proof distinctions. No
credentialed provider request or scanner restart occurred; provider chronology
remains an explicitly unverified observation, not an MVP acceptance gate. The
correction changes no replay, checkpoint, API, dashboard, ranking, or market
semantics.

#### 2026-08-17 private-launcher overnight standby correction

**State:** `accepted_deterministically`. This correction changes only the
private/local launcher's choice and timing of the next live session. It is
controlled by `PG-OPS-01`, `PG-UI-01`, `DTE-SESSION-01`, `DTE-SESSION-02`,
`LIFE-MODEL-01`, `LIFE-INIT-02` through `LIFE-INIT-05`, and
`LIFE-END-01` through `LIFE-END-03`.

With no explicit trading date, an invocation after the current scanner session
or on a weekend/holiday resolves the next trading day from the same validated,
embedded exchange schedule used by the scanner. The launcher starts the
independent loopback dashboard immediately, where the existing polling UI
honestly reports a disconnected scanner, but it does not acquire a credential,
resolve reference data, start the scanner, or connect to Massive until 03:55
America/New_York for that session. At the standby boundary it starts exactly
one ordinary fresh live scanner for the resolved date and thereafter uses the
unchanged liveness, readiness, supervision, and session-end behavior.

An explicit `--trading-date` remains exact: an unsupported or already-ended
date fails rather than rolling forward, while a valid future date uses the same
standby behavior. The launcher does not synthesize a snapshot or readiness
state, retain market state across dates, restart a failed scanner, change the
04:00-20:00 session, or make any provider request during deterministic proof.

Standby never relies on one overnight monotonic timer. It rechecks New York
wall time and the selected schedule facts at least every 30 seconds and after
every wake. If sleep, process suspension, or a clock change misses an implicit
session, it rolls to the next declared session and waits again; if an explicit
session is missed, it fails without acquiring a credential or starting the
scanner.

Primary proof `P-MVP-OVERNIGHT` covers a post-20:00 weekday, weekend and
exchange closure, explicit future and expired dates, dashboard failure and
operator shutdown during standby, delayed wake beyond session end, clock
movement, delayed credential-environment access, occupied scanner port at
wake-up, and the single transition into the existing ordinary scanner
supervision path. The dangerous counterexamples are binding the ended date,
inventing weekdays instead of using the schedule, relying on a sleep-sensitive
overnight timer, spending provider retries overnight, exposing a credential to
the dashboard, fabricating backend readiness, or leaving the dashboard
orphaned.

Focused session/launcher short and race suites pass, including public wrapper,
calendar, delayed-wake, credential-boundary, port-race, and child-containment
cases. The uncached ordinary repository tier passes; focused vet, shell syntax,
and diff checking pass. Final independent read-only review initially found a
stale-session wake false-success path, premature credential-environment access,
and a sleep-sensitive long timer. The corrections above close all three, and
focused re-review is clean. No provider credential was accessed and no live
request was made.

### MVP-S4 — integrated live-MVP acceptance

**Outcome:** The ordinary live composition builds and its deterministic
production-path scenario proves:

```text
full-universe aggregate/coverage state
  -> unchanged qualification and Day-% ordering
  -> newly selected top-20 symbol
  -> immediate Volume, From Open, Day Range, Activity 30s, and Move 30s
  -> fresh selected-row Tape 5s and Spread warm-up
  -> one coherent API v2 snapshot
  -> final dashboard model
```

Run the ordinary repository command and the narrow UI suite. A separately
authorized or owner-run live smoke observation is recorded when available;
unknown replay and disabled checkpoints remain explicit limitations.

## 7. Verification bounds

Ordinary verification remains:

```text
go test -short -timeout 2m ./...
```

Use the smallest deterministic streams that distinguish formulas, selection
independence, correction behavior, and trust states. A controlled 6,000-symbol
measurement is required only if ordinary evidence or live observation suggests
the revised full-universe feature calculation threatens the accepted live
responsiveness boundary. Do not begin replay or checkpoint performance work as
a substitute.

## 8. Completion and deferred cleanup

The MVP is complete when S1-S4 pass their allocated proofs, final capability
reviews are recorded, the repository is buildable, ordinary verification is
green, and the final API/UI behavior is coherent. Completion explicitly does
not establish replay support or revised-feature checkpoint compatibility.

After the interview MVP, a separate owner decision may choose among:

- repair and accept replay;
- retain replay as unsupported tooling;
- remove replay;
- repair and accept checkpoints;
- retain checkpoints disabled; or
- remove checkpoint persistence.

That follow-up may also split full-universe selection from displayed-row
enrichment, remove dead former-feature code, replace numbered historical
terminology, and simplify the package/document structure. None is silently
part of this program.
