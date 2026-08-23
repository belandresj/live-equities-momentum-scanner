# Integration, removal, and acceptance

**Status:** Owner-approved focused replacement specification, 2026-08-23. The
[delivery program](delivery-program.md) is the sole mutable status ledger.

**Parent:** [Live backend replacement architecture](../live-backend-replacement.md).

## 1. Outcome and authority

This contract cuts the ordinary private scanner over to one replacement live
path, removes the superseded live path, preserves API-v2/dashboard/launcher
behavior, and performs deterministic resource/stability acceptance plus a
separately authorized live observation. It owns final runtime composition,
source/dependency exclusion, compatibility verification, measurement, and
integrated evidence. It owns no market formula, alternate state engine,
provider policy, UI redesign, or mutable component status.

Allocated parent requirements are `LBR-ARCH-01`, `LBR-ARCH-02`, `LBR-ARCH-11`,
`LBR-ARCH-12`, and `LBR-ARCH-13`. `LBR-ARCH-01`/`02`/`11` are cross-cutting
conformance requirements: their component-local behavior remains in
Capabilities A through D, while this contract owns the final proof that the
ordinary composition has one owner, only justified bounded concurrency, and
the common failure domains. Allocated retained product semantics are
`PG-UI-01` and `PG-UI-02` from
[`product-goals.md`](../product/product-goals.md). The exact API/UI behavior is
reused from the accepted
[`versioned-snapshot-api.md`](../specifications/versioned-snapshot-api.md),
[`independent-ui.md`](../specifications/independent-ui.md),
[`readiness-and-operations.md`](../specifications/readiness-and-operations.md), and the
[`snapshot capture correction`](../snapshot-capture-and-dashboard-transport-isolation-correction.md);
their historical delivery ledgers and replay/checkpoint acceptance narratives
are not routed.

## 2. Boundary and dependencies

This capability depends on accepted Capabilities A through D in this order:

1. [`canonical-state-and-hydration.md`](canonical-state-and-hydration.md);
2. [`evaluation-and-publication.md`](evaluation-and-publication.md);
3. [`tq-state.md`](tq-state.md); and
4. [`live-ingress.md`](live-ingress.md).

It changes no dependency interface unless integrated evidence reopens the
lowest unsuitable focused spec through the delivery program. The dependency
graph is therefore acyclic and the supported production composition is:

```text
reference binding + one REST worker + one decoded-batch ingress
  -> one ScannerStateEngine live owner
  -> one immutable publication
  -> loopback snapshot API v2
  -> independent dashboard and private launcher
```

The API and dashboard are readers. Launcher supervision may restart its UI
child under the accepted policy but cannot replace a healthy scanner because
of API/UI failure. Reference acquisition, caches, schedule, credential
isolation, loopback security, and operator workflows retain their accepted
behavior.

Those accepted current-repository contracts, current API/UI goldens, launcher
tests, and the parent-cited durable diagnostics are the complete integration
reuse set. They are semantic/operational evidence, not authority for old
runtime topology, thresholds, checkpoint/replay state, or private structures.
No predecessor checkout or provider access is part of `LBR-E1`/`LBR-E2`.

## 3. Exclusive cutover and deletion rule

The ordinary scanner configuration constructs the replacement state, evaluator,
T/Q, and ingress directly. It has no runtime flag, environment setting,
interface fallback, build-tag fallback, error fallback, or shadow owner that
can select the old live path. Test-only differential oracles may remain only
outside production packages and may not own goroutines, mutable state, or
runtime selection.

`LBR-E1` verifies and removes the following categories after their replacement
slices are accepted:

| Superseded category | Removal boundary |
| --- | --- |
| canonical/evaluation | Old `symbolAggregateState` pointer/map/bitmap backing, qualification cloning, parallel price-range/Activity/Volume histories, legacy full-population evaluator, projection adapters, HOD drawdown, rolling 30/60-minute ranges, old Activity composite, and old Tape burst code/tests. |
| T/Q | Broad/16-minute duplicate retention, status-count/deadline acknowledgement machinery, per-symbol ordinary churn chain, obsolete command quarantine, and separately joined mutable T/Q publication. |
| ingress | Double-pass/per-event raw JSON representations, `internal/massive/live_queue.go` raw-frame queue, intermediate adapter deliveries, second general engine live FIFO, and duplicate fence/terminal paths. |
| runtime/API | Cross-snapshot market/TQ joins, lock-coupled snapshot capture, replay conditionals in the live mapper, and engine-owned checkpoint metrics/state in the supported configuration. Neutral checkpoint-off representation may remain in API v2 without a live engine checkpoint owner. |
| tests/docs | Private-structure proofs, superseded feature fixtures/goldens, fallback-mode tests, and active specification-map routes that claim the removed live path. Historical documents remain evidence and are not broadly rewritten. |

Source deletion follows dependency evidence rather than filenames alone. A
package may remain only when a retained production outcome still imports it or
the owner chooses an unsupported standalone tool below. No retained source may
force the live engine to carry replay/checkpoint fields, modes, lifecycle
branches, projections, queues, or proofs.

## 4. Replay and checkpoint disposition

The supported scanner is live-only, fresh-start only, and checkpoint mode off.
It does not accept replay flags or a checkpoint-enabled launch. The live API
omits the optional replay object and, while API-v2 compatibility requires its
checkpoint object, maps one fixed honest disabled/not-installed value from
runtime configuration rather than sampling engine checkpoint state.

Before `LBR-E1`, the owner chooses one source/tool disposition:

- **Recommended:** delete `cmd/aggregate-replay`, `internal/replay*`,
  `internal/replayartifact*`, `internal/replaymode`, `internal/checkpoint`, and
  their live runtime/engine/API branches and tests; or
- retain selected replay/checkpoint commands as clearly unsupported standalone
  tooling with a dependency graph that cannot reach the ordinary scanner or
  replacement engine.

Either choice removes replay/checkpoint participation from `cmd/scanner`,
`internal/engine`, `internal/operations`, `internal/massive`, and live snapshot
capture. Retention creates no compatibility, performance, correctness, or
acceptance claim and no Capability E gate beyond proving separation.

## 5. API, dashboard, and launcher compatibility

The cutover preserves `scanner.snapshot.v2`, `GET|HEAD /api/v2/snapshot`,
`/livez`, `/readyz`, exact loopback binding/CORS/response bounds, one atomic
publication read, and the accepted response validation. Market values,
availability, readiness, recovery, T/Q, accounting, and row order originate in
one publication. Capture remains independent from the engine lock and from
per-request memory/checkpoint sampling.

The dashboard preserves the accepted 12-column grouped view, server row order,
one-second nonoverlapping polling, three-second request bound, retained-last-
model behavior, causal `DELAYED`/`UNAVAILABLE` versus backend-derived recovery
labels, atomic safe rendering, focus/accessibility behavior, and 1440x900
desktop target. It performs no market calculation, ranking, readiness, or
provider inference. A backend/API/dashboard fault cannot stop or restart a
healthy engine; a scanner terminal remains visible without the API fabricating
freshness.

Compatibility means semantic equality of the accepted live API/UI corpus, not
private Go struct or removed replay/checkpoint equality. An intentional neutral
checkpoint-off source change is acceptable only if the same API-v2 live value
and validation behavior remain observable.

## 6. Deterministic acceptance manifest

`LBR-P1` freezes one versioned manifest before implementation changes behavior.
The manifest contains the approved 5,694-symbol mature binding/reference
facts, session bounds, hydration value/empty/failure cases, aggregate
correction/duplicate/invalid classes, rapid top-20 churn, selected T/Q
confirmation/quiet/gap/pressure cases, exact seed, batch composition, expected
accounting/ranking/API/UI digests, and stop conditions. It also records Go/OS/
architecture, host, GC settings, queue/TQ bounds, and fixture checksum. The
semantic/resource comparison baseline is code commit `0d043c1`; planning commit
`9601603` changes no baseline code.

The frozen comparison routes are:

| Boundary | Baseline source/command |
| --- | --- |
| aggregate merge/hydration | `TestENGAGG01ValidationMergeRetentionMatrix`, `TestC6INTEGRATION01OfflineComponentsOneThroughSixAggregateLifecycle`, and their approved fixture bytes |
| qualification/current fields | `TestC3QUAL01ExactGateBoundaryMatrix`, `TestC3QUAL02CorrectionAndStrictFinalizationTrace`, `TestPMVPVolumeActivityMoveExactCorrectionAndPermutation`, and `TestPMVPRankSelectionAndPreselectionHistory` |
| mature evaluator cost | `TestLiveRetainedTailCostAttribution` and `TestLiveRetainedTailSixtyOneSecondCycles`; compare semantics and measurements, not old private state |
| T/Q | `TestPC9TAQ`, exact pressure-boundary tests, and data-confirmed subscription fixtures |
| decoder/connection | `TestPC5ClassStrictMixedFrameClassification`, `TestPHRHeartbeatInboundProgressAndQuietDeadline`, and the existing 216-frames/s `TestLiveSustainedAggregateCapacity` ingress baseline |
| API/UI/launcher | affected Go packages plus `node --test ui/model.test.mjs ui/visual.test.mjs` against the accepted API-v2 corpus |
| ordinary repository | `go test -count=1 -short -timeout 2m ./...` |

The parent Section 1 durable queue/heartbeat/cycle/heap observations are linked
baseline evidence rather than copied here. Current comparable CPU and RSS are
unknown. Every later artifact records its exact command, configuration, and
checksum before the comparison is interpreted.

The 30-minute paced run targets approximately 300 provider frames/s—the
delivery program's documented rate, above the existing 216 frames/s fixture—
unless pre-implementation characterization records a more representative 1.5x
observed rate. Frame arrays supply complete all-symbol aggregate progress at
the product cadence plus representative selected-row T/Q; frame rate is not
mistaken for event rate. The manifest is validated before timing and cannot be
silently reduced after a failure.

Sample once per second and record CPU core-equivalents, heap in use, RSS,
allocation rate, goroutines, FIFO count/bytes/oldest age/slope, selection cycle,
API capture, additional processing delay, committed-watermark/readiness
transitions, all loss/reject/shed/accounting families, publication causes, and
dashboard poll results. CPU and RSS absent from the clean baseline remain
explicitly unknown rather than inferred from incomparable history.

Hard acceptance is:

- semantic oracles and accounting identities remain exact;
- aggregate/control loss and capacity rejection are zero;
- heap, RSS, retained state, FIFO, and goroutines reach bounded plateaus;
- the characterized feed creates no sustained backlog or recurring
  watermark-stale/readiness flap;
- one-second API/UI polling remains usable and isolated from engine mutation.

Every numeric design target in the parent is reported. A miss triggers exactly
one measurement-verification/profile/focused-correction/rerun cycle. When the
rerun satisfies hard acceptance, the measured deviation is recorded and work
continues; there is no repeated optimization merely to reach a target. A hard
failure reopens the narrowest implicated capability.

## 7. Optional host-coexistence and live boundaries

After the credential-free hard acceptance run, an optional host-coexistence
observation may repeat the same composition while any owner-chosen ordinary
local workload runs. Record scanner CPU/RSS, system memory pressure/swap, API/UI
responsiveness, readiness, and obvious host interference. This is diagnostic
evidence about practical headroom, not a named-application requirement or a
completion gate. A bad observation becomes a blocker only when it exposes a
scanner hard failure already defined in Section 6, such as unbounded growth,
sustained backlog, readiness flapping, or unusable polling.

`LBR-E3` is a separate exact-date market-hours observation only after
deterministic acceptance. It may access credentials or Massive only when the
owner executes it or grants explicit authorization for that exact run. It
observes fresh hydration/fence, sustained provider ingress, ranking/TQ/API/UI,
one recovery when naturally observed or safely owner-authorized, and bounded
resource trend. It does not create a provider SLA, public deployment, or
trading-edge claim. Without it the program can be deterministically complete
but not live-stability confirmed.

## 8. Failure and trust boundaries

The smallest integration false success is a green replacement run while the
ordinary binary can still execute old state or loses aggregate/control under
the chosen load. Production dependency inspection, source exclusions, forced
old-path sentinel failures, exact manifest accounting, and runtime identity
therefore participate in `LBR-E1`/`E2`.

Benchmark failure cannot be hidden by lowering event content, dropping T/Q,
changing product delay/readiness, omitting warm-up, shortening the 30-minute
plateau run, or accepting a stale dashboard. Measurement-tool failure is
distinguished from scanner failure and repaired once before interpretation.
API/UI failure stays outside provider transport meaning. Credential refusal or
closed market is lack of authorization/evidence, not deterministic failure.

## 9. Primary proofs and slices

| Slice | Primary proof | Claim, dangerous counterexample, observable distinction, limitation |
| --- | --- | --- | --- |
| `LBR-E1` | `P-LBR-E1-CUTOVER` | Build/dependency/source inspection plus a production composition trace proves `cmd/scanner` constructs exactly one replacement owner/ingress/evaluator, uses only the parent-allowed bounded concurrency, preserves the component-defined common failure domains, leaves old sentinels unreachable, removes old product fields from active state/API, needs no engine checkpoint state when checkpoint-off, and keeps API-v2/UI/launcher goldens exact through startup/current/recovery/terminal/API/UI-failure cases. It detects a fallback, shadow mutation, competing owner, unjustified runtime worker/queue, or unsupported tool importing the live core. It does not prove sustained resources or provider traffic. |
| `LBR-E2` | `P-LBR-E2-STABILITY` | The validated 30-minute deterministic manifest runs the complete backend/dashboard composition with one-second polling. It reports every parent target and proves exact final oracle/accounting, zero aggregate/control loss, bounded state/queue/heap/RSS/goroutine plateaus, no sustained backlog/readiness flap, and usable isolated polling. A numeric miss follows the single bounded rerun rule. An optional generic host-coexistence observation may record practical headroom but is not part of this primary proof. This is host/fixture evidence, not provider capacity or an SLA. |
| `LBR-E3` | `P-LBR-E3-LIVE` | One exact-date authorized/owner-run market-hours observation follows the frozen procedure and records hydration/fence completion, continuous provider progression, exact observable ranking/TQ/API/UI behavior, transport terminals/recovery if observed, and resource slopes. It rejects a credentialed smoke test that never reaches current or silently restarts. It is bounded operational confirmation, not deterministic formula proof, exhaustive recovery, provider SLA, deployment, or expectancy evidence. |

`LBR-E1` performs cutover and deletion after applying the owner source/tool
choice. Acceptance makes every old live engine, state, evaluator, queue,
envelope, feature path, compatibility flag, and active route removable—and
requires actual removal or proven standalone separation rather than dead code.

`LBR-E2` owns deterministic whole-process resource and stability acceptance.
`LBR-E3` owns only separately authorized live confirmation. No implementation
ticket combines these slices.

## 10. Verification, review, and owner gates

`LBR-E1` runs its proof, all API/UI/launcher regressions, affected race tests,
focused vet, `git diff --check`, and ordinary repository verification. `E2`
runs the prevalidated non-short manifest with the delivery program's explicit
30-minute exception and bounded stop conditions. It does not rerun an
unchanged expensive trial except under the single target-miss response. `E3`
uses the separately approved market-hours procedure and no credential access
is implied by this spec.

One final integrated read-only review after `E2` checks requirement allocation,
one owner/queue/publication, source removal, semantic compatibility, plateau/
backlog interpretation, target-miss handling, and remaining claims. `E3`, when
executed, receives a focused evidence review. Any correction reopens the lowest
affected focused contract or slice and preserves unrelated evidence.

The remaining owner decisions are deliberately latest-responsible:

1. replay/checkpoint source deletion versus separated unsupported tooling,
   before `LBR-E1`;
2. exact-date execution or authorization, before `LBR-E3`.

No other owner decision is required by this proposed set. Public deployment,
API/UI redesign, replay/checkpoint repair, alternate restart, new features,
provider protocol invention, trading validation, and tickets are non-scope.
Slice evidence and all status remain only in the delivery program.
