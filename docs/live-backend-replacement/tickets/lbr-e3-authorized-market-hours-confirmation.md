# LBR-E3 execution assignment — authorized market-hours confirmation

**Status:** Draft for independent cross-check. This ticket grants no credential
or provider authority and is not an active ledger entry.

**Activation gate:** `LBR-E2` and the final integrated review must be accepted.
The owner must execute the observation or grant explicit authorization naming
the exact date, account/environment, procedure, duration, and permitted
recovery action. The [delivery program](../delivery-program.md) must then mark
only `LBR-E3` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), [T/Q](../tq-state.md),
[ingress](../live-ingress.md), and
[integration](../integration-removal-and-acceptance.md) contracts and handoffs,
and [`market-hours-validation.md`](../../market-hours-validation.md). This
ticket executes only `LBR-E3` and `P-LBR-E3-LIVE`.

Controlling requirements are preserved fresh hydration/fence, bounded
connection/recovery, exact aggregate ranking/TQ/API/UI behavior, immutable
publication, resource-trend observability, and the explicit provider-access
gate. It creates no new implementation or product requirement.

## 2. Outcome, owner, scope, and interfaces

One exact-date bounded market-hours observation confirms that the accepted
ordinary scanner wiring works against the authorized Massive environment:
fresh reference/hydration/fence, continuous aggregate progression, ranking,
selected T/Q, API/dashboard polling, transport continuity/recovery evidence
when naturally observed or explicitly permitted, and bounded resource trend.

The owner/orchestrator owns authorization and claim language. The observer
operates the accepted scanner/procedure and records bounded evidence; it does
not modify production semantics, deploy publicly, or claim provider capacity/
SLA/edge.

## 3. Allowed execution and file boundary

No production-code edit is allowed. Preflight may create only ignored/private
runtime artifacts and the bounded redacted evidence destination named by the
authorization. Any discovered code defect ends the live attempt safely and
reopens the owning earlier ticket for a separate deterministic correction.
Only the orchestrator may later update the delivery ledger/evidence links and
commit after review.

## 4. Evidence and access whitelist

Use only the exact owner-approved credential source, endpoint/account,
trading date/session, launcher commands, output directories, duration, and
incident/recovery actions. Never print, persist in docs, or attach credentials,
tokens, account identifiers, raw sensitive payloads, or unbounded logs. No
predecessor checkout, alternate provider, replay, checkpoint, or ad hoc probe.

## 5. Required procedure and stop conditions

- Verify accepted commit/worktree, clean production paths, market schedule,
  private configuration, disk space, loopback endpoints, and redacted evidence
  locations before credential access.
- Start only the accepted private launcher/scanner/dashboard path. Record exact
  start/end times, binding/date, versions/configuration without secrets, and
  process identities.
- Observe terminal hydration work including explicit empty, live ingress fence,
  first and sustained current ranking, advancing committed `T`, selected T/Q
  confirmation/warm/current/degradation, one-second API/UI polling, queue/
  watermark/readiness, CPU/memory/goroutine/resource slopes, and accounting.
- Do not induce disconnect/recovery unless the authorization explicitly permits
  the exact safe action. If recovery occurs naturally, record first cause,
  attempt ordinals/pacing, gap hydration/fence, and restored/nonrestored state.
- Stop immediately on credential/auth anomaly, silent aggregate/control loss,
  accounting/integrity failure, unbounded growth/backlog, repeated false-ready,
  unusable polling, unsafe host impact, or procedure/authorization boundary.
- Use controlled shutdown and verify processes/goroutines/artifacts terminate
  as expected. Absence of a naturally occurring recovery is an explicit proof
  limitation, not a reason to create one.

## 6. Primary proof and acceptance distinction

`P-LBR-E3-LIVE` is the single authorized observation. It must distinguish a
scanner that merely authenticates from one that completes hydration/fence,
reaches and sustains honest current output, provides T/Q/API/UI, and shows
bounded resource/queue/readiness trends. It rejects silent restart, never-ready
smoke success, false-current recovery, and missing accounting.

This is bounded wiring/stability confirmation only. It does not re-prove
formulas, deterministically exercise every recovery, establish provider
capacity/SLA, authorize production/public deployment, or validate trading
expectancy.

## 7. Verification and timeout policy

All deterministic/ordinary evidence must already be accepted; do not rerun
unrelated expensive tests during market hours. The owner authorization sets a
bounded live duration and shutdown deadline. Preflight and postflight commands
use explicit timeouts. Afterward run only narrow deterministic regressions if a
captured anomaly needs classification; no live retry without new exact
authorization.

## 8. Evidence handoff and claim boundary

Handoff contains the authorization reference, redacted command/config, exact
times, hydration/fence/currentness timeline, provider/heartbeat/recovery facts,
ranking/TQ/API/UI observations, resource/queue/readiness/accounting series,
stop/shutdown result, artifact checksums, anomalies, and limitations. Without a
clean focused evidence review, do not record `live_stability_confirmed`.

## 9. Observer discretion and prohibited changes

Safe screen/log sampling cadence, redaction mechanics, and bounded artifact
format are delegated within the authorization. Do not change code/config
semantics, expand credentials/scope/date/duration, induce unapproved failure,
repair while live, expose secrets, claim a provider SLA, or infer market edge.

## 10. Containment, review, and correction

Authorization, preflight, redaction, bounded duration, stop conditions, and
controlled shutdown are mandatory containment. A defect or inconclusive run
records evidence and reopens the lowest deterministic ticket; it does not
authorize improvisation or another provider call. One focused read-only review
checks evidence/claim scope. Only the orchestrator updates the sole ledger and
commits after all live work/review is quiescent.
