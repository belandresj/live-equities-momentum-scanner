# Market-hours validation procedure

**Status:** Procedure approved and next permitted as `LBR-E3`; execution remains
inactive pending separate exact owner authorization.

**Purpose:** Record current provider behavior and production-path measurements
for the live-backend replacement after accepted A–D semantics and E1 exclusive
cutover. Deferred synthetic E2 capacity characterization is not a prerequisite.
This procedure does not authorize credential access by itself.

## Authorization and safety boundary

Before execution, the owner must explicitly authorize the exact date, provider
credentials, account/entitlement context, and observation duration. The run is
read-only and local. It must not publish externally, trade, modify provider
configuration, create paid infrastructure, or make a capacity claim beyond the
recorded host and observation.

Redact credentials, authorization headers, signed URLs, and provider prose that
may contain secrets. Preserve bounded aggregate metrics and small reviewed
payload shapes only where provider terms and repository evidence policy permit.

## Preconditions

- Replacement Capabilities A–D and `LBR-E1` remain accepted; E2 is explicitly
  deferred/non-gating.
- The current branch and exact commit are recorded, with a clean inventory of
  unrelated user changes.
- The session schedule, trading date, local host, Go version, configuration,
  queue/capacity bounds, and expected observation interval are fixed before
  startup.
- Per-attempt, recovery, observation, and shutdown deadlines are explicit.
- A rollback/stop action has been rehearsed without provider credentials.

## Observation record

Record at least:

1. provider reference-record count, eligible `CS`/`ADRC` count, invalid or
   missing prior closes, and the exact observed universe identity;
2. WebSocket authentication and A/T/Q command/acknowledgement behavior,
   connection epochs, provider status classes, and entitlement limitations;
3. sampled aggregate duplicates, corrections, out-of-order delivery, REST/live
   overlap, no-print outcomes, and any rejected schema shape without inventing
   unevidenced handling;
4. aggregate and T/Q ingress rate, processing delay distribution, committed-
   watermark lag, queue high-water marks/growth, memory high-water mark,
   rejected/dropped/pressure-shed counts, and bounded reason totals;
5. if naturally observed or explicitly authorized for this exact run, one
   controlled brief-interruption recovery: stale transition, exact gap,
   terminal work accounting, restored currentness, and duration;
6. T/Q normal coverage, warm-up, field status, aggregate-first shedding, full
   shedding only if naturally observed or explicitly authorized for this exact
   run, and current-rank-order restoration;
7. fresh-start hydration/catch-up duration, terminal/fence accounting, and
   retained diagnostic artifact bounds without checkpoint/replay claims; and
8. API publication identity/snapshot coherence and dashboard status behavior
   during normal, stale, recovery, T/Q-degraded, and shutdown transitions.

Do not induce destructive provider behavior or overload merely to fill a row.
Unobserved cases remain unobserved and continue to rely on deterministic proof.

## Result classification

The result is one of:

- `live_stability_confirmed`: the bounded observation and final integrated
  program review show fresh hydration/fence, sustained honest current output,
  usable API/dashboard, bounded observed resource/queue/readiness behavior, and
  retained one-owner/path conformance;
- `observed_limitation`: an entitlement, market condition, or observation
  window prevented the required live claim without disproving accepted
  deterministic component semantics;
- `correction_required`: observed behavior exposes an implementation,
  fixture, threshold, or lower-level replacement-contract problem; reopen the
  lowest owning capability through the replacement correction loop; or
- `authority_required`: the evidence would require changing a fixed Phase 1
  product/architecture rule or taking another action in the exhaustive manual-
  stop list.

Market-hours evidence may improve or invalidate lower-level replacement
decisions. It
does not silently change product semantics and does not convert scanner
correctness into evidence of deterministic 300-frames/s capacity, provider SLA,
trading edge, or executable expectancy.
