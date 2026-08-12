# Live scanner recovery narrow fix

**Status:** Reopened after the single Gate E run was externally terminated
before preload; S1 accepted and S2 deterministic correction passes A-D.

**Boundary and completed-contract authority:** Direct owner request on
2026-08-11 to use the read-only recovery review and specify the narrow fix.

**Standing program decisions:** The
[`Version 1 Release Program`](v1-release-program.md) controls C7-C10 correction
and verification. The direct request authorizes the narrow C3 mechanics change,
not changed C3 market semantics.

**Advancement mode:** Sequential correction with one active slice. Failures use
the V1 correction loop; there is no lower-level owner-response gate.

**Approved dependencies:** Accepted C2/C3 engine/evaluator, C5/C6 live and
hydration, C7 checkpoint, C8 runtime, C10 API, and C11 UI contracts.

**Decision:** Compose the demonstrated evaluation coalescing and terminal
suppression behavior, then correct only the measured late-session Activity
hotspot. Do not merge either dirty worktree wholesale or redesign ranking.

This is one C3/C7/C8/C10 correction. Pause C12 and preserve its files.

## Contract map and delivery ledger

| Document | Normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This file | Compact single-file correction contract, Sections 1-19 | `NARROW-*`, `P-NARROW-*`, S1/S2 | Every implementation, proof, review, or correction | Accepted dependencies named above |

This is the sole ledger; prior corrections are evidence.

| Item | State | Evidence/review | Next action |
| --- | --- | --- | --- |
| Correction contract | `reopened` | Planning milestone `6381f6c`; Gate E failed before preload, so A-E acceptance and focused final review are not available | Preserve the one-run failure; do not run Gate F or claim acceptance |
| S1 — stable live path | `accepted` | `P-NARROW-LIVE` and Gate B passed 2026-08-11; exact evidence below | Measure the S2 17:15 boundary before any Activity change |
| S2 — measured mature cost | `proof_incomplete` | Activity correction and synthetic A-D evidence pass; the single Gate E run was terminated in production-reader validation before any cached row or cycle | Preserve implementation/evidence as a correction milestone; no final acceptance |

### S1 acceptance evidence — 2026-08-11

S1 preserves immediate canonical aggregate mutation while coalescing the
full-population evaluator and immutable publication to the next live timer or
hydration ingress fence. It also preserves the first typed evaluator failure,
serves installed-binding warming and suppressed captures with HTTP 200 while
`/readyz` returns 503, stops `RunLive` before any connection attempt in a
terminal lifecycle, and adds a default-on live checkpoint switch whose off
path constructs no store/writer and submits no checkpoint work.

Gate A passed in fast-to-long order:

```text
go test -timeout 90s ./internal/engine -run 'TestLiveAggregateEvaluationCoalescing|TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect|TestC3POP02AccountingIntegrityAndOverlap' -count=1  # 2.31s
go test -timeout 90s ./internal/operations -run '^TestSuppressedIngressCannotEnterReconnectHotLoop$' -count=1  # 0.89s
go test -timeout 90s ./internal/snapshotapi -run 'TestLiveWarmupSnapshotIsServableAndBound|TestSuppressedEvaluatorIntegrityRemainsServable' -count=1  # 0.59s
go test -timeout 90s ./cmd/scanner -run 'TestLiveOperator|TestInitialOperator' -count=1  # 0.58s
perl -e 'alarm 90; exec @ARGV' node --test ui/model.test.mjs  # 0.34s, 18/18
go test -timeout 90s ./internal/engine -run 'TestC3ACT|TestFreshHydrationActivityOptimizationsMatchValueOracle' -count=1  # 1.08s
go test -timeout 90s ./cmd/scanner -run '^TestCheckpoint' -count=1  # 0.56s
go test -timeout 90s ./internal/operations -run '^TestProductionCompositionCoalescesThousandAcceptedAggregates$' -count=1  # 0.58s
```

The 1,000-event composition admitted all inputs with one insert and 999
revisions, zero rejected/canceled/closed/sequence-exhausted admission deltas,
zero publication replacements before the timer, and exactly one replacement
at the timer. The resulting population was universe `1`, valid prior `1`,
trusted rankable `1`, covered `1`, unresolved `0`; aggregate/admission
identities reconciled and queue occupancy returned to zero.

The ordinary Gate B command `go test -short -timeout 2m ./...` passed in
11.61 seconds. Orchestrator review rejected an initial proof that synthesized a
sealed API capture with `unsafe`. The corrected proof used the production path
from a real evaluator accounting fault through suppression, sealed runtime
capture, and HTTP mapping; focused snapshot, engine, and UI checks passed in
1.92 seconds, 1.63 seconds, and 0.43 seconds, and the repeated Gate B passed in
8.65 seconds. `git diff --check` was clean.

Construction prevents a pending accepted prefix from disappearing, clears it
only after successful evaluator validation/apply, latches rather than replaces
the first evaluator diagnostic, preserves deterministic replay behavior,
prevents a suppressed/ended runtime from opening a socket, and makes
checkpoint-off status explicit with zero checkpoint work. This proves S1
semantics and ordinary conformance, not late-session capacity, provider
behavior, or Gate F.

### S2 correction and Gate E failure evidence — 2026-08-11

The mandatory pre-change 6,000-symbol 17:15 measurement had exactly 1,589
reference blocks plus one target block per symbol. Three trials reported:

| Trial | Stage | Apply | Publication | Total lock | Allocated bytes |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 10.268257667s | 9.082167ms | 519.125µs | 10.2778605s | 6,497,812,568 |
| 2 | 11.541589417s | 4.965542ms | 775.333µs | 11.547334875s | 6,492,435,720 |
| 3 | 10.663651875s | 5.47725ms | 83.458µs | 10.669216125s | 6,492,435,736 |

Mean lock time was about 10.83147 seconds and every trial exceeded two
seconds. Stage time and about 6.49 GB of allocations per publication isolated
the session-length Activity traversal; apply and publication did not justify a
ranking index or broader evaluator change.

The triggered correction keeps the existing at-most-1,920 block summaries as
semantic and checkpoint authority and adds only two derived sorted finite-value
collections, each bounded by 1,920. A corrected eligible block removes its old
values, recomputes from canonical evidence, and inserts its new values.
Percentiles retain inclusive `<= target` ties. Whole-interval trust is checked
before lookup use, and checkpoint installation rebuilds the derived lookup
through ordinary deterministic apply; nothing is persisted in checkpoint
schema.

Post-correction Gate C passed with the same exact manifest:

| Trial | Stage | Apply | Publication | Total lock | Allocated bytes |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 287.817333ms | 1.369833ms | 94.792µs | 289.295333ms | 14,692,504 |
| 2 | 62.610167ms | 429.458µs | 38.459µs | 63.078417ms | 9,315,656 |
| 3 | 60.465416ms | 431.5µs | 68.167µs | 60.965375ms | 9,315,640 |

Mean lock time was 137.779708 ms and maximum was 289.295333 ms. Gate D's
generated 60 one-second evaluation cycles reported stage total 4.767944542s,
apply total 304.075541ms, publication total 1.700374ms, mean lock 84.562449ms,
maximum 359.461709ms, zero aggregate rejection, zero queue growth, and lookup
cardinality at most 1,920. Checkpoint-off production composition separately
made ten successful HTTP snapshot polls while checkpoint submitted/outstanding
work remained zero. The Gate D fixture directly measures staged/apply/
publication cycles; it does not instantiate HTTP or admit a fresh aggregate
correction in each measured cycle, so those are separate proofs rather than one
fully joined D trace.

Activity differential proofs cover insert, correction, withdrawal containment,
the exact 100-transaction eligibility boundary, restoration, inclusive target
ties, stalled `T`, horizon folding, and checkpoint continuation/rebuild. The
focused Activity/checkpoint suite passed in 3.00 seconds; checkpoint-off HTTP
composition passed in 2.27 seconds; `go test -short -timeout 2m ./...` passed
in 7.99 seconds before Gate E and 8.93 seconds after it.

The corrected Gate E preflight passed without a full parse: the compact
production-reader fixture passed in 0.01 seconds; the exact private regular
artifact was 2,584,011,150 bytes with file SHA-256
`e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189`;
the exact reference files were `prior-close/2026-08-06.json` and
`universe/2026-08-07.json`; reference/binding validation completed in about
3.30 seconds; and the recorded manifest named binding
`session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d`,
7,671,171 records, 5,691 symbols, and the complete session interval
`[2026-08-07T08:00:00Z,2026-08-08T00:00:00Z)`.

The one permitted Gate E command was:

```text
/usr/bin/time -p env CACHED_HYDRATION_ACCEPTANCE=1 CACHED_HYDRATION_RATE_MULTIPLIER=1 go test -timeout 8m ./internal/operations -run '^TestCachedHydrationFenceAcceptance$' -count=1 -v
```

Its internal context was 7m30s. It emitted only
`=== RUN TestCachedHydrationFenceAcceptance`, then the process was terminated
after approximately 28 seconds inside the initial
`replayartifact.OpenValidatedContext` full-artifact validation, without a Go
failure, PASS, exit status, or `/usr/bin/time` trailer. Preload did not begin:
zero rows were applied and zero of ten planned cycles ran. The command was not
rerun. This distinguishes local process/resource termination in the existing
production reader from Activity semantics, cached-data results, provider
behavior, or a Gate E capacity result.

Because Gate E is not green, the correction remains reopened. The conditional
final read-only review was not spawned, no acceptance commit is authorized,
and Gate F remains both separately authorized and unexecuted. No provider
credential or request was accessed.

Final local checks after preserving the failure passed:

```text
go test -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner  # 31s
go vet ./...  # 1s
git diff --check
```

The diff contains no replay, replayartifact, replaymode, checkpoint codec/
store/writer, partial-ranking, or C12 change. Since Gate E failed, these checks
prove only local race/static/diff conformance of the deterministic correction;
they do not convert the missing cached-data result into acceptance.

## Sections 1-4 — Outcome, scope, ownership, and settled boundary

The narrow fix is complete when a fresh-start live scanner:

- applies each accepted aggregate to canonical state immediately but performs
  full-population qualification/ranking only at the next one-second timer or
  hydration ingress fence;
- never starts another connection attempt after the engine reaches
  `suppressed` or `ended` unless an explicit lifecycle recovery event permits
  it;
- serves coherent warming and suppressed snapshots instead of turning those
  states into an apparent API disconnect;
- sustains a representative 17:15 ET publication cadence with mean evaluator
  work below one second and no result above the existing two-second readiness
  tolerance; and
- can run the first diagnostic live milestone with checkpoint discovery and
  cadence explicitly disabled, using bounded fresh REST hydration for recovery.

Checkpoint restart remains a V1 requirement. Disabling it for this milestone
neither deletes C7 nor proves checkpoint recovery; it isolates the first live-
currentness proof from unmeasured checkpoint projection cost.

The exact controlling Phase 1 requirements are:

- `PG-RANK-02` through `PG-RANK-05`, `PG-FEATURE-02`, `PG-FEATURE-05`,
  `PG-AVAIL-01` through `PG-AVAIL-03`, `PG-OPS-01`, `PG-OPS-02`, and
  `PG-OBS-01` through `PG-OBS-03`;
- `ARCH-OWN-01`, `ARCH-OWN-03`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, and
  `ARCH-FLOW-04`;
- `DTE-TIMER-01`, `DTE-MERGE-01` through `DTE-MERGE-05`,
  `DTE-RECOVERY-02` through `DTE-RECOVERY-05`, `DTE-COMMIT-02` through
  `DTE-COMMIT-04`, and `DTE-CHECKPOINT-01`;
- `LIFE-HYDRATE-03` through `LIFE-HYDRATE-06`, `LIFE-LIVE-01`,
  `LIFE-LIVE-02`, `LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, and
  `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`; and
- accepted component requirements `C3-ACT-01`, `C3-ACT-02`, `C3-EVAL-01`,
  `C7-CADENCE-01`, `C8-RUNTIME-01`, `C8-READY-01`, `C8-RECOVERY-01`,
  `C8-MEASURE-01`, `C8-CAPACITY-01`, `C10-STATUS-01`, and `C10-HTTP-01`.

No formula, qualification threshold, ranking order, merge rule, watermark,
readiness tolerance, or suppression meaning changes.

The `ScannerStateEngine` remains the only mutable owner of canonical facts,
pending evaluation work, derived Activity state, lifecycle, watermark, ranking,
and publication. Operations may decide whether to construct checkpoint
components and must stop attempts when the engine is terminal. The API/UI only
consume immutable observations. This correction adds no product rule, mutable
owner, watermark, evaluator, T/Q-to-ranking dependency, changed interval, or
duplicated responsibility.

In scope are live evaluation coalescing; terminal lifecycle guarding;
warming/suppressed visibility; fixed-cardinality stage/apply/publication timing;
a measurement-triggered Activity lookup correction; and a live-only checkpoint
switch whose default preserves existing behavior.

Out of scope are `degraded_current` or another ranking mode; conflict,
qualification, formula, or ranking changes; replay/C12; a ranking index unless
the compact scalar scan remains the measured blocker; larger queues or looser
timeouts; and checkpoint schema/codec/storage changes.

## Sections 5-8 — Evidence, reconnaissance decision, and reuse

Create a clean correction branch/worktree from
`4cf302986c20e80116e16c82f310566db2d5d921`. Treat both dirty worktrees as
read-only patch sources until exact hunks are classified:

1. `/Users/joshuabelandres/Dev/live-equities-momentum-scanner-api-correction`
   supplies coalescing, the terminal guard, and bounded evaluator proofs.
2. `/Users/joshuabelandres/Dev/live-equities-momentum-scanner` supplies typed
   integrity, warming/suppressed visibility, operator output, and regressions.

The following existing observations justify the correction:

- the main tree evaluates the universe after every accepted aggregate, which
  scales as aggregate arrival rate times universe size and can fill the
  512-frame live queue;
- the API-correction tree coalesces work, passes the short suite, and completed
  60 sparse two-hour cycles in about 29 seconds; staging averaged 243.9 ms and
  peaked at 515.7 ms;
- Activity currently walks every eligible session-to-date 30-second reference
  block during both candidate staging and deterministic apply, implying about
  19.1 million reference-block visits per 17:15 publication at 6,000 symbols;
- the main regressions reproduce suppression-reason loss, a reconnect hot loop,
  and an unservable warm-up snapshot; and
- the prior 300-cycle/full-artifact harness failed repeatedly before producing
  accepted mature-tail evidence. Those runs are diagnostic history, not a
  reason to rerun the same expensive shape.

Partial-ranking, C12/replay, checkpoint-format, acceptance-export, and Gate D
changes are excluded. No predecessor, credential, or provider request is
authorized.

No predecessor reconnaissance is needed or permitted. Current repository code,
the two observed worktrees, deterministic regressions, and recorded timing
answer the implementation questions. The exact reuse whitelist is:

| Source | Decision | Preserved behavior / excluded coupling |
| --- | --- | --- |
| API-correction `internal/engine/evaluator.go` and `evaluation_coalescing_test.go` | Adapt | Preserve live timer/fence coalescing; exclude partial ranking, checkpoint-format, and Gate D changes. |
| API-correction `internal/operations/live.go` | Adapt | Preserve `terminalLiveStateError` before each connection attempt; exclude unrelated composition changes. |
| Main-tree evaluator/publication/operations/snapshot regression paths named in S1 | Adapt | Preserve typed first failure and coherent warming/suppression; exclude replay/C12 changes. |
| Current `internal/engine/feature_activity.go` plus accepted C3 Activity proofs | Behavior and oracle evidence | Measure first; change only reference lookup if dominant. |

The contract changes evaluator scheduling and terminal cross-component
behavior, so implementation requires focused read-only review after proof.

## Sections 9-17 — Required behavior, proofs, slices, and bounds

### Slice 1 — compose the stable live path

Port the API-correction tree's live-mode coalescing and terminal guard into the
clean target. Preserve these exact effects:

1. accepted changes update only affected canonical/support state; every live
   aggregate disposition marks one pending accounting prefix;
2. no ordinary nonintegrity aggregate admission performs a full-universe
   projection or replaces the immutable publication;
3. the next timer or hydration ingress fence evaluates the pending prefix once,
   publishes its cumulative accounting even if rows are unchanged, and clears
   the pending bit only after successful validation/apply;
4. replay retains its deterministic per-input behavior unless an existing
   replay-group boundary already coalesces it; and
5. a suppressed/ended lifecycle is terminal to `RunLive`; it cannot enter an
   implicit reconnect loop.

Port the main tree's typed integrity and warming/suppressed visibility only
after coalescing is green. Preserve the first diagnostic, exact lifecycle
reason/disposition, binding identity, empty current rows, and HTTP 200 coherent
snapshot for warming or installed-binding suppression. `/readyz` remains 503
for both. Do not port C12/replay changes.

Add a live-only checkpoint switch. With checkpoint mode off, do not construct a
store/writer, discover/install a checkpoint, or submit cadence projections.
Report checkpoint status honestly as not installed with zero work. Default
behavior remains checkpoint mode on.

Primary proof `P-NARROW-LIVE` combines:

- `TestLiveAggregateEvaluationCoalescing`;
- `TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect`;
- `TestSuppressedIngressCannotEnterReconnectHotLoop`;
- `TestLiveWarmupSnapshotIsServableAndBound`; and
- `TestSuppressedEvaluatorIntegrityRemainsServable`, the existing operator
  output suite, and focused UI suppression model test; plus
- one production-composition test proving 1,000 accepted aggregates create no
  full-population evaluation/publication until one timer, then reconcile exact
  admission/disposition/population accounting with zero queue rejection.

It rejects a ready snapshot that hides accepted work and a suppressed reconnect
loop. It does not establish late-session capacity or provider behavior.

### Slice 2 — remove only the measured mature-session cost

First measure one 17:15 ET stage/apply over a generated 6,000-symbol state with
1,590 session blocks per symbol: 1,589 references plus the target. Report stage,
apply, publication, allocations, and total lock duration separately.

If mean work is below one second and every trial is below two seconds, make no
Activity representation change. If Activity reference traversal dominates,
replace only that traversal with maintained correction-aware reference
summaries and two bounded ordered value collections, one for transactions and
one for expansion. A touched 30-second block removes its old eligible values,
recomputes from canonical evidence, and inserts its new eligible values.
Percentile lookup counts values `<= target` with inclusive ties. The existing
1,920-block summaries remain the semantic and checkpoint authority; ordered
collections are derived state and are rebuilt/validated after checkpoint
install rather than persisted as a second truth.

This keeps the simple full-universe scalar ranking scan. A deterministic second
apply pass is acceptable only when it no longer repeats a session-length walk.
Do not add a `U`-sized mutable candidate image or a ranking index to avoid that
pass.

Primary proof `P-NARROW-MATURE` requires:

- full-history-oracle equality for Activity after insert, correction,
  withdrawal, exact tie, eligibility loss/restoration, stalled `T`, and
  checkpoint rebuild;
- one generated 6,000-symbol 17:15 publication with stage/apply attribution;
- 60 exact one-second cycles with mean total evaluator work below one second,
  maximum below two seconds, zero aggregate rejection, no queue growth, and no
  retained-state growth with publication count; and
- checkpoint-off production composition with API polling and honest checkpoint
  status.

This is current-host evidence for the stated shape, not provider latency, an
SLA, or trading edge.

### Failure, trust, accounting, and boundedness

| Boundary | Accept into success only when | Contain/reject | Dangerous false success |
| --- | --- | --- | --- |
| Pending aggregate prefix | Every accepted aggregate is in canonical state and its final disposition appears in the next timer/fence publication | Validation failure retains pending work and enters existing integrity containment | Ready publication silently omits already accepted work |
| Terminal runtime | Engine observation is nonterminal or an explicit lifecycle recovery transition authorized another attempt | `suppressed`/`ended` returns without opening a socket | Reconnect hot loop after fail-closed suppression |
| Activity index | Rebuilt values exactly equal eligible block summaries and oracle output at the same `T` | Mismatch makes Activity unavailable/invalid under existing containment; it never changes rank | Fast percentile from stale pre-correction values |
| Checkpoint-off mode | No store/writer exists and all checkpoint counters remain zero | Reject contradictory configuration at startup | UI implies checkpoint recovery while none ran |

Existing identities remain authoritative: aggregate consumed dispositions,
population partitions, hydration terminal work, and checkpoint writer accounting
must each reconcile exactly. Activity timing, reference count, lookup work,
queue high-water, and API latency are fixed-cardinality overlapping diagnostics,
not new primary populations. Pending work is one boolean/prefix boundary;
Activity retains no more than the existing 1,920 block summaries plus two
derived collections of at most 1,920 finite values per symbol. All waits,
trials, cycles, queues, and commands use explicit bounds.

### Mandatory fast-to-long proof ladder

Every gate must pass before the next. A failed gate is reduced and corrected;
an unchanged failed command is never rerun.

| Gate | Maximum processing time | Proof and release condition |
| --- | ---: | --- |
| A — focused semantics | `< 2m` per command | Run the four direct regressions, coalescing, operator/UI, Activity oracle, and checkpoint-off proof. |
| B — ordinary repository | `2m` | `go test -short -timeout 2m ./...`. Long/capacity/live tests must skip under `testing.Short()`. |
| C — mature synthetic single boundary | `< 2m` | Build the 6,000-symbol/17:15 generated state, validate exact symbol/reference/target counts, then time three stage/apply trials. |
| D — mature synthetic 60 cycles | `< 2m` | Run only after Gate C meets the one-/two-second limits. Validate all planned corrections, timers, API polls, and expected publications before timing. |
| E — cached real-data composition | `<= 8m`, one run | Only after A-D are green, validate the sealed artifact path, byte size, digest, binding, reference dates, interval, and expected record/symbol counts. Preload once to 17:15, execute ten exact cycles, and measure the same segments. Do not rerun after presentation-only changes or unchanged failure. |
| F — current-date live observation | `15m`, separately authorized | Only after A-E and final review. Run checkpoint-off fresh bootstrap once, stop at the first typed integrity/capacity/readiness failure, preserve the diagnostic, and do not retry in the same session. |

Gate E may use the existing sealed aggregate artifact and current reader only.
It must not add another JSONL parser. Before its potentially longer parse, a
sub-two-minute preflight checks exact file/reference paths, byte size, recorded
manifest, and a compact artifact fixture through the production reader.

Gate F requires explicit authorization for its trading date, credentials,
duration, and diagnostic path under `docs/market-hours-validation.md`. The
command must use the new checkpoint-off option. Success requires terminal
hydration, an exact reconciled ingress fence, advancing one-second watermark,
zero queue-capacity rejection, no suppression, API/UI reachability throughout,
and honest independently available T/Q fields.

## Sections 18-19 — Acceptance and drift audit

Production edits are limited to the narrow paths needed in:

```text
internal/engine       coalescing, terminal diagnostic, Activity cost, fixed metrics
internal/operations   terminal lifecycle guard, checkpoint-off composition
internal/snapshotapi  warming/suppressed coherent mapping
cmd/scanner           checkpoint switch and existing operator output
ui                    suppression/API-transport presentation only
docs                  correction ledger and authority map
```

Tests may extend the corresponding package test files. Do not edit replay,
replayartifact, replaymode, checkpoint codec/store/writer, partial-ranking, or
C12 paths.

After deterministic behavior stabilizes, run the narrowest affected race
packages with an explicit timeout no greater than five minutes, then `go vet
./...` and `git diff --check`. A focused read-only review is required because
the correction touches atomic evaluator scheduling and terminal lifecycle
containment. Review must confirm one owner/evaluator/watermark, no weakened
validation, no accepted aggregate loss, correct suppression visibility, and no
session-length work on aggregate admission.

Record slice acceptance in this document with exact commands, durations,
manifest, stage/apply attribution, queue/rejection/accounting results, review
findings, and limitations. The correction is accepted for the fresh-start live
milestone only after Gates A-E and the focused review pass. Gate F remains a
separate market-hours confirmation and may be pending; a failure reopens only
the typed boundary it distinguishes.

Completion also requires resolved links, one owner/evaluator/publication path,
whitelist conformance, slice walkthroughs, and a clean drift audit:

| Question | Answer | Evidence |
| --- | --- | --- |
| New product rule, owner, watermark, evaluator, or T/Q ranking gate? | No | Scheduling and derived representation only; fixed C3 truth remains authoritative. |
| Changed time, window, correction, merge, readiness, or suppression meaning? | No | Existing timer/fence, Activity, two-second tolerance, and lifecycle rules are unchanged. |
| Unevidenced edge behavior or predecessor authority? | No | Every edge comes from the recorded live incident, deterministic regression, accepted invariant, or current timing. No predecessor is used. |
| Duplicated responsibility or unnecessary machinery? | No | One pending prefix, optional construction of existing checkpoint components, and two bounded Activity value collections; no new service, queue, or framework. |
