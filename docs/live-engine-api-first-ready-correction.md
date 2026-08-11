# Live engine-to-API first-ready correction

Status: accepted locally after correction and focused independent re-review,
2026-08-11. This is a narrow C8/C10 correction under the Version 1 Release
Program, not a new component or a change to market semantics.

## Conclusion

The retained evidence does not prove that the 2026-08-11 HTTP 503 rejected the
same publication that the last operational status line reported as ready. The
status sample was recorded at `12:11:56.891Z`; the failed HTTP request was at
`12:11:57.293Z`, about 402 milliseconds later. The failed capture and mapper
error were not retained, so the exact live rejection predicate is unknowable
from that run.

Scale and the first degraded publication are not sufficient causes. A
deterministic production-runtime proof selected a prefix containing at least
167,046 source records and applied exactly 166,984 registered hydration rows
for a 5,691-symbol binding, with 58 REST/live conflicts at the hydration fence.
The same proof then admitted the first ordinary post-fence live aggregate,
admitted the engine timer that publishes its projection, and mapped the new
bound, current publication. A small ordinary proof traverses the real engine,
Runtime capture, mapper, loopback listener, `/api/v1/snapshot`, `/readyz`,
server shutdown, and Runtime shutdown.

The correction therefore does two things that are justified by evidence:

1. fixes a reproducible adjacent lifecycle defect in which a quiet timer after
   recoverable ingress suppression erased the fixed `ingress_integrity` cause;
   and
2. retains and immediately reports a bounded private mapper failure record, so
   a future HTTP rejection identifies the exact invariant instead of collapsing
   all diagnosis into the public `snapshot_unavailable` response.

Public HTTP errors and product semantics remain unchanged.

## Authority and boundary

Controlling requirements are `PG-OPS-02`, `PG-OBS-03`, `ARCH-OWN-01`,
`ARCH-OWN-03`, `ARCH-FLOW-02`, `DTE-RECOVERY-02` through
`DTE-RECOVERY-05`, `LIFE-HYDRATE-03` through `LIFE-HYDRATE-06`,
`LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, and `LIFE-PUBLISH-01`
through `LIFE-PUBLISH-03`.

The engine remains the sole lifecycle, canonical-state, evaluation, and
publication owner. Runtime still captures one sealed publication and derives
readiness from that capture. C10 still owns only representation and loopback
transport. Aggregate ranking remains independent of T/Q health.

In scope are first-ready and immediate post-ready transport, recoverable
suppression visibility, mapper failure classification, private reporting, and
graceful API/runtime shutdown. Provider access, credentials, another live run,
UI changes, market formulas, queue sizing, timeout increases, and service
splitting are out of scope.

## Reproduction and correction

`TestIngressSuppressionSnapshotIsServableBoundAndNoncurrent` starts from a
current bound publication, enters same-binding recoverable ingress suppression,
then admits the dangerous later timer before requesting the product snapshot.
Before correction the response remained HTTP 200 with zero rows and suppressed
ranking, but `publication.lifecycle_reason` was empty rather than
`ingress_integrity`. A reasonless legal self-transition is progress, not new
lifecycle evidence; the engine now preserves the transition that established
suppression.

Mapper failures now use closed invariant identifiers such as
`capture_coherence`, `lifecycle_reason`, `population_mark_identity`, and
`hydration_fence`. The handler records only the invariant, route, publication
ID, last engine sequence, lifecycle, and ranking mode. One latest record and
one pending notification are retained, so repeated failed requests consume
constant memory. The scanner drains the notification immediately to stderr.
The public bodies remain `snapshot_unavailable` and
`publication_unavailable`.

## Proof allocation and results

- Ordinary end-to-end startup proof:
  `TestLiveFirstReadyDegradedPublicationIsTransportable`. It creates real
  engine-owned degraded/incomplete-population rows, captures them through
  `operations.Runtime`, serves a real loopback listener, obtains HTTP 200 from
  the snapshot and readiness endpoints, and joins server and Runtime shutdown.
- Warm-up and suppression proof:
  `TestLiveWarmupSnapshotIsServableAndBound` and
  `TestIngressSuppressionSnapshotIsServableBoundAndNoncurrent`. They prove
  bound noncurrent warm-up and bound recoverable suppression without retained
  ranking rows.
- Diagnostic proof:
  `TestMappingDiagnosticsIdentifyInvariantAndRemainBounded`. It distinguishes
  a specific population identity failure and proves that 100 reports retain
  only the latest value and one notification.
  `TestMappingFailureTraversesHTTPServerDiagnostics` sends a test-only invalid
  sealed capture through a real listener, proves the public generic 503 body,
  and verifies the exact retained/notified private invariant.
- Retained scale proof:
  `TestRetainedArtifactLiveBootstrapIntegrity`, explicitly selected with the
  sealed 2026-08-07 artifact and exact cached reference directory. It validates
  7,671,171 artifact records and a 5,691-symbol binding, selects at least
  167,046 source records, applies exactly 166,984 registered rows through the
  production hydration ledger with 58 conflicts, and maps the first degraded
  ready publication. It then admits the first post-fence live aggregate and an
  evaluator timer, requires a newer publication ID, and maps the subsequent
  bound current publication. No provider request or credential is used.

The fresh retained-scale rerun passed in 164.02 seconds. Focused race checks for
the affected engine/API/scanner paths, `go vet ./...`, `git diff --check`, and
the final serial `go test -count=1 -short -timeout 2m ./...` pass.

One non-authoritative verification attempt ran the ordinary suite concurrently
with a broad race command. The competing load caused an existing two-second
launcher-start assertion to expire and existing operations capacity harnesses
to reach their 512-frame safety stop. The failed packages passed immediately
when rerun alone, and the authoritative full ordinary command then passed
serially. No production correction or bound change was made from the
load-contaminated attempt.

Two independent read-only reviews initially found proof gaps. One found that
the retained test admitted a post-fence aggregate without the timer required to
publish its projection. The corrected proof now requires that timer and a
strictly newer publication ID. The other found that mapper diagnostics were
tested below the production wiring. A test-only corrupted sealed capture now
crosses the real listener/handler/mapper path, proves the unchanged generic 503
body, and verifies exact server retention/notification; the scanner diagnostic
encoder has a separate exact-output proof. Focused re-reviews found both P2
items closed and no remaining P1/P2.

## Dangerous counterexample and limitation

The dangerous counterexample is a valid engine publication being replaced
between the last status log and the HTTP capture by a state that the mapper
rejects, leaving only a generic 503. The new ordinary and retained tests cover
the first-ready publication, the first post-fence aggregate, warm-up,
recoverable suppression, and shutdown. The private diagnostic makes any other
rejection attributable to an exact invariant.

The original 2026-08-11 mapper predicate remains unproven because no failed
capture was persisted. This correction must not be described as reconstructing
that missing fact, and deterministic acceptance is not live-provider
confirmation.
