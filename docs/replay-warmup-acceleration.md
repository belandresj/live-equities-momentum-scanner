# Replay warm-up acceleration

**Status:** Implemented and semantically accepted locally; the 30-second warm-up
target passes, while the 90-second total-preparation target remains unmet
because the accepted artifact boundary performs two complete validation scans.

**Authorized:** 2026-08-14.

**Purpose:** Make cached aggregate replay reach its CLI-declared observation
start quickly, then continue automatically at the existing 1x wall schedule
through the unchanged snapshot API and dashboard.

This correction is separate from the live feature-set MVP and does not make
replay a gate for live operation. It reopens only the replay warm-up
performance and then-current-feature equivalence claims exposed by the retained
2026-08-07 artifact. The owner has explicitly waived the repository's full
focused-component-template and assignment ceremony for this correction.

## Conclusion and implementation size

This is a medium backend refactor, not a replay rewrite. The replay coordinator
already has the correct high-level schedule: `[S,O0]` is unpaced and
`(O0,O1]` is paced from one monotonic wall anchor. The scale defect is that the
engine repeatedly performs full-population feature/ranking projection during
the hidden warm-up, and the coordinator copies a complete warming capture after
each logical second.

Expected implementation footprint:

- two sequential slices;
- changes concentrated in `internal/engine` and `internal/replaymode`, with
  focused tests in those packages;
- approximately 3-6 production files and 3-6 test files;
- approximately 300-700 production lines and 600-1,200 proof/benchmark lines;
- no API schema, dashboard asset, replay artifact schema, provider, or live-mode
  change.

The uncertainty is not file count. It is whether qualification and retained
feature evidence can be advanced incrementally using the existing per-symbol
state, or whether a small explicit warm-up accumulator is required. The first
slice must settle that question with a profile and an equivalence proof before
the production seam is finalized.

## Operator outcome

The operator uses the existing commands and the same dashboard:

```text
go run ./cmd/scanner \
  --run-mode replay \
  --replay-artifact /absolute/path/to/aggregate-replay.jsonl \
  --reference-dir /absolute/path/to/reference-cache \
  --observation-start 09:25:00 \
  --observation-end 09:35:00 \
  --api-address 127.0.0.1:8080 \
  --allow-origin http://127.0.0.1:4173

go run ./cmd/dashboard \
  --address 127.0.0.1:4173 \
  --api-origin http://127.0.0.1:8080 \
  --assets ui
```

The backend validates the cached artifact, folds every aggregate and coverage
group from 04:00 through 09:25 without wall pacing, publishes the first exact
09:25 replay observation, and immediately continues at 1 logical second per
wall second. The dashboard continues polling `GET /api/v2/snapshot`; it may be
opened during warm-up or after observation begins. There is no Play button,
pause/seek control, alternate frontend, or browser-owned clock.

"Streaming" in this workflow means the existing once-per-second immutable
snapshot sequence. The browser does not receive or reconstruct raw aggregate
events.

## Controlling behavior

This correction preserves the compatible intersection of:

- `PG-RANK-01` through `PG-RANK-04`: unchanged qualification-before-ranking
  and exact Day-%/symbol ordering;
- `PG-FEATURE-01` through `PG-FEATURE-07`: current Volume, From Open, Day
  Range, Activity 30s, and Move 30s meanings and all-symbol evidence;
- `DTE-CLOCK-04` through `DTE-CLOCK-06`, `DTE-TIMER-01`, and
  `DTE-COMMIT-01` through `DTE-COMMIT-04`: one simulated clock, ordered timer
  facts, and one committed watermark;
- `DTE-REPLAY-01` through `DTE-REPLAY-03`: validated artifact identity,
  canonical record/group order, and speed-independent logical results;
- `LIFE-REPLAY-01` through `LIFE-REPLAY-03`: one replay engine, ordinary
  evaluator semantics, honest nonlive publication, and exact completion; and
- `C12-WINDOW-01`, `C12-SCHEDULE-01`, `C12-API-01`, and
  `C12-CONTAIN-01`: hidden warm-up, cumulative 1x observation scheduling,
  atomic replay context, and source-owned cancellation.

The 2026-08-14 live feature-set MVP remains authoritative for live operation.
This correction neither changes its acceptance boundary nor enables checkpoint
mode.

## Current evidence and failure

The first retained-artifact product run validated the complete 2026-08-07
artifact and exact binding, entered `warming`, then completed only 107 of
20,101 groups in approximately 107 seconds. That projects to about 5.5 hours
before a 09:30 observation. A 09:25 observation requires 19,501 warm-up groups
and has the same unacceptable behavior.

The coordinator contains no warm-up wall wait. The visible repeated work is:

1. Replay aggregate changes with an established committed `T` invoke
   `stageAggregateEvaluationAtLocked` across the population.
2. Every replay group/timer performs full-symbol maintenance and another full
   population projection.
3. Every pre-`O0` group calls `PublishReplay`, which observes and defensively
   clones a complete engine snapshot even though API mapping suppresses warm-up
   rows.

These are strong code-level suspects, not a substitute for a production-scale
profile. Slice RW-S1 records CPU and allocation attribution before choosing the
smallest correction.

## Required design

### One canonical fast-forward path

Acceleration must remain inside the existing replay source, engine, and
coordinator composition. It may add one typed engine-internal distinction
between hidden warm-up groups and observable replay groups. It must not add a
second state owner, evaluator, clock, artifact reader, or ranking
implementation.

For every warm-up group through `O0`, the engine must still:

- admit every normalized replay record in artifact ordinal order;
- apply every aggregate disposition representable by the validated replay
  input, plus coverage/no-print meaning, exactly as ordinary replay;
- admit the exact group/timer fact at its logical whole second;
- advance replay source accounting, logical clock, coverage, and lifecycle in
  the same order;
- preserve path-dependent qualification latches, session Volume, open/high/low,
  retained 330-second evidence plus predecessor mark, Activity state, and
  correction consequences; and
- fail or suppress on the same invalid evidence.

Hidden warm-up may avoid materializing a complete immutable ranking
publication at each intermediate logical second. At `O0`, it must perform one
ordinary full-population projection using production evaluation delay `D=4s`,
commit the exact target `clamp(floor_second(O0-D),S,E)`, and publish the first
observable replay snapshot. No row or value from an intermediate fast-forward
boundary may be exposed as the `O0` result.

Simply skipping evaluation until `O0` is not sufficient if doing so loses an
earlier qualification pass or evicts evidence needed at `O0`. The correction
must either reuse existing incremental per-symbol state or introduce the
smallest engine-owned warm-up accumulator that makes those path-dependent
facts equivalent. Replaymode may select the declared `O0`; it may not compute
market state.

### Warming publication

The API may retain one initial `warming` capture so the unchanged dashboard can
show a noncurrent historical process. Per-second full snapshot cloning during
fast-forward is not required. If progress updates are retained, they must be
bounded by wall time or coarse logical intervals and must not trigger full row
materialization. Progress is diagnostic; it cannot become another market
watermark or correctness owner.

### Automatic observation

After the exact `O0` group and projection complete, replaymode captures `W0`
from the existing monotonic wall clock and immediately publishes `observing`.
Every later group retains the current deadline:

```text
deadline(t) = W0 + (t - O0)
```

Processing lateness remains explicit schedule lag. No warm-up optimization may
reset the anchor, skip a group, weaken artifact suffix validation, or alter
requested-end completion.

### Validation boundary

Full artifact validation remains unchanged in this correction. The retained
2.584 GB artifact has previously required roughly 40 seconds for its optimized
reader validation, so initial end-to-end preparation will not be literally
instant even after warm-up is fixed. A content-addressed validation cache or
artifact index is a separate follow-up requiring its own mutation/trust model.

## Proofs and performance acceptance

### RW-EQUIV — exact logical-boundary equivalence

Run the same complete deterministic artifact through:

1. the existing ordinary per-group projection schedule retained as a
   test/reference path; and
2. accelerated hidden warm-up through `O0`.

At `O0`, compare canonical per-symbol state, aggregate identities and
coverage, qualification latches and proof state, feature state, committed `T`,
ranking accounting, ordered rows and every displayed value/status/reason,
publication lifecycle, replay clock, source accounting, and next-group
continuation state. The comparison excludes only publication IDs generated by
deliberately omitted intermediate publications and wall diagnostics.

The fixture matrix must include:

- a qualification pass well before the retained 330-second tail;
- a symbol entering the top 20 at `O0` after being unselected throughout
  warm-up;
- successful empty/no-print symbols;
- sparse prints and known-no-print Move carry;
- Activity 30s across its complete 330-second evidence boundary;
- every insert/exact-duplicate/replacement disposition constructible by the
  accepted replay artifact paths. Acceleration is active only for validated
  complete-final-bars evidence; partial-synthetic replacement coverage proves
  that configuring the boundary leaves its ordinary schedule exact. Live-only
  withdrawal/correction paths remain unchanged and receive a regression proving
  the warm-up policy is unreachable from live mode;
- `O0=S`, ordinary premarket `O0`, and `O1<R` requested end; and
- cancellation or invalid evidence during accelerated warm-up.

Then advance both engines for at least 60 identical observable groups after
`O0` and compare every logical-boundary market result. Equal state only at
`O0` is insufficient if continuation diverges.

### RW-SCALE — retained-artifact discriminator

Using the existing local 2026-08-07 artifact and reference caches, with no
provider request, credential read, artifact copy, or licensed-row logging:

- confirm the existing manifest before timing;
- profile one bounded warm-up prefix before correction;
- fast-forward 04:00 through 09:25 after correction;
- require warm-up processing, excluding already measured full artifact
  validation, to finish in at most 30 seconds on the current development host;
- require total validation plus warm-up to finish in at most 90 seconds;
- record records/second, groups/second, CPU attribution, allocations, peak RSS,
  publication count, and exact source/engine accounting; and
- continue at least five minutes at 1x without cumulative schedule lag growth,
  then retain exact requested-end success.

The 30/90-second bounds are local private-workflow targets, not provider,
cross-host, or production SLAs. If profiling shows the bounds require a second
state owner, changed market semantics, unbounded memory, or artifact trust
weakening, record the evidence and revise this focused correction rather than
forcing the optimization.

### RW-UI — unchanged dashboard composition

Run the existing dashboard assets against the accelerated backend. Prove that:

- no UI file changes are required;
- the existing endpoint and polling interval are unchanged;
- warming exposes no market rows;
- the first rows are the exact `O0` observation;
- rows then change once per replay logical second under 1x pacing;
- replay remains visibly historical/nonlive and T/Q remains unavailable; and
- dashboard restart does not restart or alter the replay backend.

## Sequential slices

### RW-S1 — engine/coordinator acceleration

Profile the retained slow path, implement the smallest engine-owned
fast-forward projection policy and bounded warming capture policy, and pass
`RW-EQUIV` on deterministic fixtures. Allowed source scope is
`internal/engine`, `internal/replaymode`, and the minimum typed seam in
`internal/replay` or `internal/replayartifact/playback` only if the warm-up
boundary cannot be expressed without it. No UI, API schema, provider, live
adapter, checkpoint, or artifact-format edits.

The slice is complete when exact `O0` and 60-second continuation equivalence
pass under ordinary and race testing, cancellation still joins through the C4
source owner, and a focused read-only review finds no replay/live evaluation
cadence leak or alternate market-state path.

### RW-S2 — real artifact and unchanged UI acceptance

Run `RW-SCALE`, then `RW-UI`, for the 09:25-09:30 ET window. Correct only
evidence-backed scale defects inside the RW-S1 boundary. Update the replay
operator documentation with the exact existing commands and measured startup
behavior. No frontend modification is authorized by this slice; an unexpected
UI incompatibility reopens the backend/API compatibility claim rather than
being patched with a replay-specific frontend.

## Non-scope

- Play, pause, seek, speed, or replay-date controls in the dashboard;
- persisted replay seekpoints or checkpoint restoration;
- content-addressed validation caching or a new artifact index;
- replay T/Q reconstruction;
- provider acquisition, credentials, or live-provider validation;
- public deployment or authentication;
- changed qualification, feature formulas, rank ordering, evaluation delay,
  session bounds, or artifact final-bar semantics; and
- claims of market edge, executable expectancy, or live receipt chronology.

## Completion boundary

This correction is accepted only when the retained artifact reaches 09:25
within the local performance bounds, the first and continued observable states
are exactly equivalent to the reference replay, the unchanged dashboard shows
the automatic 1x sequence, and cancellation/failure cannot leave a prior
replay-current snapshot served as success. Until then, replay remains retained
but operationally unverified.

## Implementation and acceptance record — 2026-08-14

RW-S1 is accepted locally. The engine now owns one immutable replay-only
fast-forward boundary installed after binding and before replay start. Before
`O0`, records and groups still traverse the ordinary FIFO, canonical merge,
coverage, accounting, simulated clock, and lifecycle paths. Only symbols
touched in each hidden group receive their ordinary bounded feature-state
maintenance; full-population projection and immutable publication are
deferred. Records at
and after `O0` are also coalesced to their enclosing replay-group timer because
the runtime can observe replay state only after that atomic group. `O0` and
every later group therefore perform one ordinary projection rather than one
projection per record plus one per group. Live engines reject the policy, and
an active replay cannot add or replace it.

`RW-EQUIV` compares ordinary per-group replay with accelerated replay at `O0`
and at every one of the following 60 boundaries. The deterministic complete
artifact includes 460 seconds, an early qualification pass preceding the
retained 330-second feature window, a late qualifying/ranking entrant, an
empty symbol, sparse prints, complete Activity 30s evidence, and Move carry.
Canonical records and coverage, price/range state, Activity reference/mutable/
folded state, qualification proof/dirty/accounted-through state, MVP folded and
result state, committed `T`, ranking accounting/rows, lifecycle, clock, source
accounting, engine sequence, and continuation state are equal after excluding
only deliberately different publication IDs. A second partial-synthetic
regression configures the boundary on partial-synthetic evidence and directly
proves that execution remains on the ordinary schedule for an insert,
replacement, and exact duplicate. It compares the full exact engine view,
including mutable Activity and replacement-sensitive qualification/feature
state, at `O0` and every boundary through `O0+60`. Acceleration is active only
after ReplayStart validates `complete_final_bars`; the partial artifact is test
evidence for constructible engine merge dispositions, not an expansion of
runtime artifact trust. Invalid-artifact, requested-end, and cancellation
regressions remain. A focused regression proves the fast-forward policy is
rejected by live mode and immutable once configured.

The stronger internal-state comparison reopened the first implementation: it
showed that skipping hidden group maintenance preserved the displayed `O0`
result but not retained Activity/qualification state. Restoring ordinary
all-symbol maintenance repaired equivalence but measured 86.029 seconds of
warm-up. The narrow correction retains ordinary maintenance only for symbols
touched in each hidden group, then performs the ordinary all-symbol boundary
projection at `O0`; the expanded equivalence tests pass and the retained scale
returned below 30 seconds.

The coordinator now retains one initial `warming` capture and publishes no
per-second warming clone. It automatically publishes the exact `O0` capture,
anchors the existing cumulative 1x schedule, and continues through the same
API/runtime path.

RW-S2 used the retained 2026-08-07 artifact in the authorized archive without
provider access, credential reads, or artifact copying. Its confirmed manifest
remained:

| Fact | Result |
| --- | ---: |
| Artifact bytes | 2,584,011,150 |
| Artifact records | 7,671,171 |
| Coverage entries / universe | 5,691 |
| Empty symbols | 172 |
| Artifact ID | `sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a` |
| Binding ID | `session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d` |

The explicit scale rung measured:

| Phase / resource | Result |
| --- | ---: |
| Initial full validation | 40.209 s |
| Playback full revalidation | 41.341 s |
| 04:00 through 09:25 warm-up | 27.765 s |
| Total preparation | 109.316 s |
| Warm-up groups | 19,501 |
| Warm-up applied records | 321,887 |
| Warm-up throughput | 702.4 groups/s; 11,593.3 records/s |
| Warm-up allocation volume | 7,320,652,856 bytes |
| Engine publication ID at first observation | 4 |
| Five-minute terminal lag | 148 ms |
| Completed observation boundaries | 301 |
| Corrected-run maximum RSS | 634,912,768 bytes |

The warm-up target of at most 30 seconds passes. The total-preparation target
of at most 90 seconds does not: the accepted handle validation and playback
cursor each validate all artifact bytes before engine mutation, consuming
81.551 seconds before the 27.765-second fold. Removing or caching either scan
would change the artifact mutation/trust boundary and remains outside this
correction. This is the only unmet acceptance condition; it is recorded rather
than weakening validation or introducing a second state owner.

The 09:25-09:30 CLI run completed exactly at requested end with 19,801 groups,
335,258 applied prefix records, 7,335,913 intentionally unapplied suffix
records, zero unread records, one completed run, and no failed or canceled run.
Observed 1x lag samples were 234 ms, 115 ms, 142 ms, and 148 ms at completion;
there was no cumulative lag growth. The exact 09:25 state had one locally
unknown population member and therefore honestly published no rows; subsequent
exact boundaries resolved complete population evidence and published 20 rows.

RW-UI passed with no frontend changes. The existing dashboard showed
`HISTORICAL · NONLIVE`, advanced its replay timestamp/publication under the
one-second polling path, displayed 20 server-ranked rows when ranking became
current, and kept Tape 5s and Spread replay-unavailable. Restarting only the
dashboard reattached to the later backend publication without restarting or
altering replay.

Verification passed for the focused engine/replay/replaymode packages under
ordinary and race builds; `go test -count=1 -short -timeout 2m ./...`,
`go vet ./...`, and `git diff --check` passed. The first full short run exposed
one unrelated operations recovery timing failure; its isolated rerun and the
complete immediate rerun passed without a code change.

## Committed-T replay coverage correction — 2026-08-14

The first operator replay exposed a four-second periodic ranking withdrawal
while the observation remained healthy. Direct API sampling showed the exact
false transition: one newly printing symbol changed population accounting from
5,691 covered / 0 unresolved to 5,690 covered / 1 unresolved even though 3,554
trusted rankable marks remained. Four logical seconds later the population
returned to complete. The interval matched production evaluation delay `D=4s`.

The defect was point-in-time mismatch. Complete replay delivery had admitted a
new record ahead of committed `T` and removed that symbol's latest-delivery
`no_print_through_t` consequence immediately. Evaluation at the older `T`
could not yet select the future record, so it temporarily saw neither an
eligible mark nor no-print proof and withdrew the exact ranking.

Complete fresh replay now derives population coverage at evaluation time from
the validated artifact prefix through `T` and the latest mark eligible in
`[S,T)`. A delivered record newer than `T` therefore cannot erase the no-print
proof applicable at `T`; once `T` reaches the record, the symbol transitions
directly from no-print to trusted mark. Live and checkpoint-continuation
coverage consequences remain unchanged. A genuinely unresolved replay
population with other known marks is labeled `incomplete_population` rather
than the false `no_trusted_marks`; replay still publishes no partial ranking.

The deterministic regression adds a symbol that resumes printing during the
observable interval under nonzero `D` and requires 0 unresolved population and
`qualified_current` at every boundary through `O0+60`. The real retained
artifact discriminator sampled the previously failing 09:29:29-09:29:36
interval: every boundary remained 5,691/5,691 covered, zero unresolved,
`qualified_current`, and 20 rows. At 09:29:37 the symbol moved from no-print to
trusted-mark accounting without a ranking withdrawal.

Focused ordinary and race suites, snapshot/API coverage, `go vet ./...`, and
the repository short suite pass. The first repository run hit the unrelated
Massive unawaited-handshake timing test; its isolated rerun and the immediate
complete rerun passed without a code change. This correction changes no UI,
API shape, artifact trust, replay clock/order, partial-synthetic path, live
coverage behavior, or total-preparation limitation recorded above.
