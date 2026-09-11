# Implementation status

The supported project is the private/local live scanner on `main`. It uses the
replacement backend: compact canonical state, incremental qualification,
population-wide selection, selected-row enrichment, and one decoded-batch live
handoff. Historical REST hydration supports live startup and gap recovery.
Replay and checkpoint packages have been removed.

## Recorded provider observation

The retained observation ran on August 25, 2026, from 21:46:19 to 21:56:19 UTC
(17:46:19–17:56:19 New York time), after the regular close. It used the ordinary
launcher, 8 hydration workers, Massive data, and a MacBook Pro with Apple M1.
These are historical recorded results, not a new run performed for publication.

| Measurement | Recorded result |
| --- | --- |
| Reference universe | 5,698 symbols |
| Symbols with valid prior closes, planned for hydration | 5,554 |
| Hydration outcomes | 5,504 with values; 50 completed-empty; no failed/canceled/fenced work |
| Normalized hydration rows consumed | 6,592,756 |
| Full snapshot polls | 499; collection began 93 seconds after launcher start |
| Ready samples after fence reconciliation | 468 consecutive samples |
| Ready-to-not-ready transitions afterward | 0 |
| Watermark lag against the causal target | p95 1 second; maximum 2 seconds |
| Queue high-water mark | 47 of 4,096 frame slots |
| Snapshot/dashboard HTTP failures | 0 |
| Population and T/Q accounting | Valid across recorded samples |
| Controlled shutdown | Clean, with no remaining private processes or loopback listeners |

The [recorded evidence](live-backend-replacement/evidence/lbr-e3-stability-live-2026-08-25.json)
identifies code commit `0b1afe3`, the host, configuration, timestamps, checksums,
resource observations, and limitations. The 468 ready samples are operational
observations, not independent market signals. Watermark lag excludes the
configured four-second evaluation delay and does not measure order-execution
latency.

No natural disconnect, recovery, pressure shedding, or watermark-stale interval
occurred during this observation. It does not establish regular-session or
opening-burst capacity, a provider SLA, or behavior across multiple days.
The raw local capture files are not distributed with the source repository;
the retained evidence summarizes and identifies them.

## Offline verification and limits

Unit and fake-provider tests exercise canonical merge, corrections, ordering,
qualification, availability, publication coherence, recovery, API validation,
and dashboard behavior. The ordinary suite is
`go test -short -timeout 2m ./...`; browser-model tests use Node's built-in test
runner.

The planned ten-minute deterministic capacity characterization remains deferred.
An earlier invalid measurement is retained as such; it is not evidence that the
replacement sustains a specified synthetic input rate. The predecessor's
market-open batching results likewise do not measure this backend.

The project does not establish predictive value, executable trading expectancy,
public service security, or multi-user operation. Any new provider observation
requires explicit authorization for that run.

## Documentation

| Topic | Guide |
| --- | --- |
| Runtime ownership and data flow | [System architecture](architecture/system-overview.md) |
| Market definitions and selection | [Product specification](product/product-goals.md) |
| Local launch, credentials, and shutdown | [Operation guide](operations/private-scanner.md) |
| Diagnostics and evidence interpretation | [Diagnostics](operations/diagnostics.md) |
| API contract | [Snapshot API v2](api/snapshot-v2.md) |
| Browser behavior | [Dashboard](ui/dashboard.md) |

Completed replacement design and evidence records remain in
`docs/live-backend-replacement/`. Superseded predecessor specifications and
incident ledgers remain available in Git history. Research notes under
`docs/research/` describe possible extensions or historical analysis, not
additional implemented capabilities.
