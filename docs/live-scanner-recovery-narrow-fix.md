# Live scanner recovery narrow fix

**Status:** Owner-requested executable correction contract; implementation not
started.

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

**Sequencing correction, 2026-08-14:** The owner-requested
[`live T/Q resilience correction`](live-tq-resilience-correction.md) now
precedes this implementation. Its corrected T/Q-versus-aggregate failure
domains and explicit bounded same-binding recovery control this document where
older terminal-guard wording could otherwise imply unconditional process exit.
This contract retains the prohibition on implicit reconnect loops; it does not
override the explicit recovery event or T/Q independence.

**Decision:** After that dependency is accepted, compose the demonstrated
evaluation coalescing and corrected lifecycle guard, then correct only the
measured late-session Activity hotspot. Do not merge either dirty worktree
wholesale or redesign ranking.

This is one C3/C7/C8/C10 correction. Pause C12 and preserve its files.

## Contract map and delivery ledger

| Document | Normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This file | Compact single-file correction contract, Sections 1-19 | `NARROW-*`, `P-NARROW-*`, S1/S2 | Every implementation, proof, review, or correction | Accepted dependencies named above |

This is the sole ledger; prior corrections are evidence.

| Item | State | Evidence/review | Next action |
| --- | --- | --- | --- |
| Correction contract | `contract_recorded_waiting_dependency` | Owner-directed boundary, current-worktree evidence, two proofs, and two slices recorded; `live-tq-resilience-correction.md` must be accepted first; focused evaluator/lifecycle review required after implementation | Resume S1 after T/Q resilience acceptance |
| S1 — stable live path | `pending` | `P-NARROW-LIVE` | Compose and prove coalescing, terminality, visibility, and checkpoint-off startup |
| S2 — measured mature cost | `pending` | `P-NARROW-MATURE` | Begin only after S1 acceptance |

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
| API-correction `internal/operations/live.go` | Adapt | Preserve the check that forbids opening a socket directly from `suppressed`/`ended`; replace unconditional return for `same_binding_recovery_allowed` with the preceding correction's opaque engine-commanded scheduled recovery. Exclude unrelated composition changes. |
| Main-tree evaluator/publication/operations/snapshot regression paths named in S1 | Adapt | Preserve typed first failure and coherent warming/suppression; exclude replay/C12 changes. |
| Current `internal/engine/feature_activity.go` plus accepted C3 Activity proofs | Behavior and oracle evidence | Measure first; change only reference lookup if dominant. |

The contract changes evaluator scheduling and terminal cross-component
behavior, but its deterministic coalescing, suppression, lifecycle, race, and
diff-hygiene proofs are the acceptance evidence. Completion or reopening of
this boundary does not independently trigger read-only review.

## Sections 9-17 — Required behavior, proofs, slices, and bounds

### Slice 1 — compose the stable live path

Port the API-correction tree's live-mode coalescing and adapt its terminal guard
to the accepted T/Q resilience boundary. Preserve these exact effects:

1. accepted changes update only affected canonical/support state; every live
   aggregate disposition marks one pending accounting prefix;
2. no ordinary nonintegrity aggregate admission performs a full-universe
   projection or replaces the immutable publication;
3. the next timer or hydration ingress fence evaluates the pending prefix once,
   publishes its cumulative accounting even if rows are unchanged, and clears
   the pending bit only after successful validation/apply;
4. replay retains its deterministic per-input behavior unless an existing
   replay-group boundary already coalesces it; and
5. a suppressed/ended lifecycle cannot enter an implicit reconnect loop;
   `restart_required`/ended remains terminal, while
   `same_binding_recovery_allowed` waits for the explicit bounded recovery
   event owned by the preceding T/Q resilience correction.

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
| Terminal runtime | Engine observation is nonterminal or an explicit lifecycle recovery transition authorized another attempt | Open no socket while suppressed; wait for the opaque scheduled command when same-binding recovery is allowed, and return only for ended/restart-required state | Reconnect hot loop or process exit that bypasses the explicit recovery disposition |
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
./...` and `git diff --check`. The deterministic proofs and conformance
walkthrough must confirm one owner/evaluator/watermark, no weakened validation,
no accepted aggregate loss, correct suppression visibility, and no
session-length work on aggregate admission. Do not invoke an independent
reviewer merely because this C9 boundary was reopened or these gates are green.

Record slice acceptance in this document with exact commands, durations,
manifest, stage/apply attribution, queue/rejection/accounting results, review
findings if a risk-triggered review occurred, and limitations. The correction
is accepted for the fresh-start live milestone after Gates A-E pass. Gate F
remains a separate market-hours confirmation and may be pending; a failure
reopens only the typed boundary it distinguishes.

Completion also requires resolved links, one owner/evaluator/publication path,
whitelist conformance, slice walkthroughs, and a clean drift audit:

| Question | Answer | Evidence |
| --- | --- | --- |
| New product rule, owner, watermark, evaluator, or T/Q ranking gate? | No | Scheduling and derived representation only; fixed C3 truth remains authoritative. |
| Changed time, window, correction, merge, readiness, or suppression meaning? | No | Existing timer/fence, Activity, two-second tolerance, and lifecycle rules are unchanged. |
| Unevidenced edge behavior or predecessor authority? | No | Every edge comes from the recorded live incident, deterministic regression, accepted invariant, or current timing. No predecessor is used. |
| Duplicated responsibility or unnecessary machinery? | No | One pending prefix, optional construction of existing checkpoint components, and two bounded Activity value collections; no new service, queue, or framework. |
