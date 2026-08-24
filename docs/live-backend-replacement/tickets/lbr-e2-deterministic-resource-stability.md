# LBR-E2 implementation assignment — deterministic resource stability

**Status:** Independently reviewed implementation assignment; owner-revised to
exactly one 10-minute deterministic acceptance run on 2026-08-23. Activation
occurs only through the [delivery program](../delivery-program.md); this file is
not a mutable ledger.

**Activation gate:** `LBR-E1` must be accepted and the delivery ledger must
record the exact `LBR-P1` manifest/characterization frozen. The
[delivery program](../delivery-program.md) must mark only `LBR-E2` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), [T/Q](../tq-state.md),
[ingress](../live-ingress.md), and
[integration](../integration-removal-and-acceptance.md) contracts and all
recorded handoffs. This ticket implements only `LBR-E2` and
`P-LBR-E2-STABILITY`.

Controlling requirements are `LBR-ARCH-12`, the complete parent resource table
and hard acceptance, exact A–D semantics/accounting, and fixed API-v2/dashboard
one-second polling behavior. Numeric values are design targets; the hard gates
are correctness, zero aggregate/control loss, bounded plateaus, sustained feed
handling, stable readiness, and usable isolated polling.

## 2. Outcome, owner, scope, and dependencies

Run exactly one versioned, deterministic, mature 10-minute backend/dashboard
composition and establish whether the final exclusive scanner is semantically
exact and reaches bounded CPU/memory/state/queue/goroutine behavior without
sustained backlog/readiness flapping or unusable one-second API polling.

This slice owns fixture/measurement execution and evidence, not market
semantics. No optional, fallback, diagnostic, host-coexistence, or non-gating
repeat is authorized. Provider traffic is deferred to E3.

## 3. Allowed implementation boundary

Allowed paths are deterministic capacity/soak fixtures and tests under
`internal/operations` and directly participating packages, fixed-cardinality
metrics already required by the accepted contracts, bounded local scripts, and
evidence artifacts routed by the program. Production correction is not
preauthorized: a demonstrated target miss/hard failure first reopens the
narrowest owning A–E1 ticket and records the correction before its files change.

Do not edit product/API/UI meanings, provider transport assumptions, replay/
checkpoint, public deployment, or credentials.

## 4. Evidence and manifest whitelist

Use only the exact frozen P1 manifest: 5,694-symbol binding/reference facts,
session/hydration/correction/duplicate/invalid classes, rapid top-20 churn,
selected T/Q confirmation/quiet/gap/pressure, seed, frame/batch composition,
expected semantic/accounting/API/UI digests, host/Go/OS/GC/queue/TQ settings,
and fixture checksum. The timed manifest is exactly 600 seconds, 180,000 frames,
3,416,400 base aggregates, 600 resource samples, 600 dashboard polls, and 11
ranking/accounting checkpoints. Baseline code is `0d043c1`; parent-linked durable facts
and named comparison routes are reused only when their premise is unchanged.
No predecessor checkout or provider access.

## 5. Required measurement and acceptance behavior

- Validate manifest bytes, symbol/event/correction/TQ counts, rate, duration,
  expected work, bounds, checksums, and stop conditions before timing.
- Pace exactly 300 provider frames/s while maintaining complete all-symbol aggregate cadence,
  representative selected T/Q, hydration/fence, and one-second dashboard polls.
- Run exactly 10 minutes unless a safety stop terminates earlier. Use an
  explicit 15-minute command timeout; do not shorten or extend the timed input.
- Sample/report every parent target: CPU average/p95, heap hydration/steady,
  RSS, allocation rate, goroutines, FIFO count/bytes/age/slope, selection cycle,
  API capture, additional processing delay, watermark/readiness transitions,
  publications, loss/reject/shed, and all accounting identities.
- Require exact semantic/API/UI oracles, zero aggregate/control loss/capacity
  rejection, bounded heap/RSS/retained/FIFO/goroutine plateaus, no sustained
  backlog/readiness flap, and usable isolated polling.
- If hard acceptance passes, a numeric miss is recorded as a deviation from the
  one run and cannot authorize a second trial. Measurement invalidity or a hard
  failure reopens the lowest owning ticket; a future E2 activation again owns
  exactly one 10-minute run.
- A hard failure reopens the lowest implicated capability; never reduce fixture
  content/rate/duration, drop required T/Q, change delay/readiness, or accept a
  stale dashboard to pass.

## 6. Primary proof and acceptance distinction

`P-LBR-E2-STABILITY` is the validated exact 10-minute complete composition. Observe
all metrics above as time series/slopes and final identities/digests, with exact
publication causes and polling results. The dangerous counterexample is a short
green endpoint with growing heap/queue or recurring false-ready intervals.

The proof is local host/fixture evidence, not provider capacity, SLA, live
continuity, public deployment, or trading expectancy. No coexistence or other
repeat composition exists in this authority.

## 7. Verification and timeout policy

Before E2, reuse accepted ordinary/race/vet evidence whose code/premise is
unchanged and run narrow manifest validation. Execute the one 10-minute trial
with an explicit 15-minute command timeout and active safety stops. Run affected
short tests and `git diff --check` after measurement, then ordinary verification
at the gate. Do not execute another timed composition.

## 8. Evidence handoff and next gate

Handoff records command/config/checksum, host/Go/GC, complete target table,
hard-gate results, time-series artifact paths, semantic/accounting digests,
target deviations, proof limitation, and any reopened capability. A hard-
conforming result triggers the
required final integrated read-only review. E3 remains unauthorized until its
separate owner gate.

## 9. Implementer discretion and prohibited changes

Fixture implementation, deterministic generator, sampling/report mechanics,
fixed artifact format, and plotting/report helpers are delegated. This
discretion covers measurement representation only. Any production FIFO slot,
byte, reserve, or batch-bound change must reopen `LBR-D2` before mutation. Do
not revise market semantics, loosen hard acceptance, turn target misses into
tuning or repeat trials, invent baseline CPU/RSS, add provider requests, or add
an optional/fallback/diagnostic composition.

## 10. Containment, review, and correction

Manifest validation and safety stops prevent an incomparable or runaway trial.
Measurement-tool failure is repaired once and distinguished from scanner
failure. Every implementation correction routes to the lowest owning ticket
before mutation. Final integrated review covers sole owner/path, semantic
compatibility, deletion, plateaus/slopes/backlog/readiness, polling, and bounded
target-miss handling. Only the orchestrator records deterministic completion
and commits after all reviewers are quiet.
