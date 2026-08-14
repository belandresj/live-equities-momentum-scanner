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
   SYMBOL FLOAT VOLUME LAST | DAY % FROM OPEN % DAY RANGE | ACTIVITY 30s MOVE 30s | TAPE 5s SPREAD
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
positive/negative Move, Activity/Tape attention treatment, Spread friction,
keyboard/focus preservation, and no-color-only meaning.

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
