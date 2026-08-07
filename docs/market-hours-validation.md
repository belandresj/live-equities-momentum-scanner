# Market-hours validation procedure

**Status:** Procedure approved; execution pending separate owner authorization.

**Purpose:** Record current provider behavior and production-path measurements
after the private/local V1 RC is complete. This observation is not part of the
RC acceptance gate and does not authorize credential access by itself.

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

- The private/local V1 RC and its deterministic final review are accepted.
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
5. one controlled brief-interruption recovery when safe: stale transition,
   exact gap, terminal work accounting, restored currentness, and duration;
6. T/Q normal coverage, warm-up, field status, aggregate-first shedding, full
   shedding if naturally or safely exercised, and current-rank-order
   restoration;
7. checkpoint age, artifact bytes, write/load/install segments, catch-up, and
   equivalent fresh-recovery control when a safe comparable observation is
   available; and
8. API publication identity/snapshot coherence and dashboard status behavior
   during normal, stale, recovery, T/Q-degraded, and shutdown transitions.

Do not induce destructive provider behavior or overload merely to fill a row.
Unobserved cases remain unobserved and continue to rely on deterministic proof.

## Result classification

The result is one of:

- `observed_consistent`: recorded behavior is consistent with the V1 contracts;
- `observed_limitation`: the RC remains correct within deterministic evidence,
  but an entitlement, market condition, or observation window prevented a live
  claim;
- `correction_required`: observed behavior exposes a C7-C11 implementation,
  fixture, threshold, or lower-level contract problem; reopen it through the
  V1 correction loop; or
- `authority_required`: the evidence would require changing a fixed Phase 1
  product/architecture rule or taking another action in the exhaustive manual-
  stop list.

Market-hours evidence may improve or invalidate lower-level V1 decisions. It
does not silently change product semantics and does not convert scanner
correctness into evidence of trading edge or executable expectancy.
