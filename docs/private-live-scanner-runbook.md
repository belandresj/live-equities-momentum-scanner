# Private live scanner runbook

This is the supported private/local daily workflow. Start the foreground
launcher at approximately 03:55 America/New_York from the repository root:

```text
./scripts/run-private-scanner
```

Starting several minutes before 04:00 gives preflight and compilation time to
finish before the scanner session begins. It is an operational recommendation,
not a different ranking mode: the same full-universe aggregate qualification,
Day-% ranking, hydration, and engine-owned readiness rules apply at every start
time.

## Prerequisites

- macOS with the repository's Go 1.26 toolchain available as `go`;
- an exact-date reference cache under `var/reference`, or ordinary provider
  access capable of resolving it during scanner startup;
- writable `var/reference` and `var/checkpoints` directories;
- loopback ports `127.0.0.1:8080` and `127.0.0.1:4173` available; and
- a Massive credential supplied through one of the mechanisms below.

An already exported `MASSIVE_API_KEY` takes precedence. Otherwise, create a
macOS Keychain generic-password item with account `joshuabelandres` and service
`momentum-scanner-massive-api`. This command prompts for the password because
`-w` is last; it does not place the credential in shell history:

```text
security add-generic-password -U -a joshuabelandres -s momentum-scanner-massive-api -w
```

The launcher never prints or writes the credential, never includes it in a
command argument, and removes it from the dashboard, browser opener, and Go
build environments. It reaches only the scanner child environment.

## Commands and fixed daily configuration

The public command shape is:

```text
./scripts/run-private-scanner [--trading-date YYYY-MM-DD] [--hydration-workers 1|2|4|8] [--open]
```

`--open` asks macOS to open the dashboard after both listeners are healthy. A
browser-open failure is nonfatal. `--trading-date` is for an explicit date
correction; it does not authorize historical replay through the live provider
path. The scanner remains the trading-calendar authority and rejects weekends,
holidays, and unsupported dates. Hydration defaults to eight workers. The
worker override changes only bounded REST hydration concurrency; it does not
change the universe, interval, merge rules, ranking, readiness, or T/Q path.

The launcher invokes the scanner with these exact settings:

```text
--run-mode live
--trading-date <current America/New_York date or explicit override>
--hydration-workers <8 by default; explicit 1, 2, 4, or 8>
--reference-dir <repo>/var/reference
--checkpoint-dir <repo>/var/checkpoints
--checkpoint-mode off
--diagnostic-dir <repo>/var/diagnostics (scanner default)
--api-address 127.0.0.1:8080
--allow-origin http://127.0.0.1:4173
```

For the 2026-08-12 owner-run retry, the scanner's internal bounded delivery
settings are 32,768 raw-frame slots, 128 MiB total queued payload, 8 MiB per
frame, a 4-GiB cumulative (not resident) hydration-transfer allowance, and five
finite connection/recovery attempts. These settings add containment headroom
only; exact readiness,
coverage, accounting, page/request limits, and fail-closed terminal behavior
are unchanged.

It invokes the independent dashboard with:

```text
--address 127.0.0.1:4173
--api-origin http://127.0.0.1:8080
--assets <repo>/ui
```

The script resolves the repository from its own location, so its absolute path
also works from another directory. It builds private runtime binaries under
the ignored `var/run-private-scanner/bin` directory and then remains in the
foreground supervising both processes.

The scanner retains at most one fixed-cardinality ingress incident per process
and, on a typed terminal, writes a create-without-overwrite JSON record under
`var/diagnostics` with owner-only directory/file permissions. It contains
absolute queue/accounting operands and at most 60 one-second samples; it does
not contain credentials, provider URLs/prose, symbols, or raw payloads.

## URLs and status interpretation

After the scanner's `/livez` succeeds, the launcher starts the dashboard and
prints:

- dashboard: `http://127.0.0.1:4173`;
- scanner snapshot: `http://127.0.0.1:8080/api/v1/snapshot`;
- process liveness: `http://127.0.0.1:8080/livez`; and
- authoritative readiness: `http://127.0.0.1:8080/readyz`.

Interpret them precisely:

- `/livez` HTTP 200 means the scanner HTTP process is running. It does not mean
  ranking is current.
- `/readyz` HTTP 200 with `backend_ready=true` means the scanner engine declares
  the published snapshot ready through its committed watermark and reconciled
  ingress fence.
- A healthy process can remain not ready before 04:00, during REST hydration,
  or while the buffered live tail is being fenced. The launcher reports the
  engine's reason and continues supervising; it does not synthesize readiness.
- "Honestly suppressed" means a capacity or integrity condition prevents the
  engine from proving a current result. The scanner publishes no false-ready
  claim; inspect the first scanner error, lifecycle/suppression reason, and
  `/readyz` reason before considering a restart.

The dashboard may show field-level warming, stale, or unavailable states even
when aggregate ranking is ready. In particular, T/Q availability is
independent of aggregate readiness and ranking.

## Cold starts, late starts, and restarts

Before 04:00, the runtime waits for the bound session and then advances
incrementally. This avoids a large elapsed-session REST backlog; it does not
skip any symbol, use top-N hydration, or alter qualification/ranking.

From 04:00 through 20:00 New York time, the launcher prints a warning and still
starts normally. The private launcher defaults checkpoint mode off, so an
ordinary same-day restart hydrates the full elapsed session, consumes the
buffered live tail, applies the exact ingress fence, and only then reports
ready. An explicit scanner launch with `--checkpoint-mode on` may instead
install the latest valid compatible checkpoint, fall back to the previous
valid candidate, and hydrate `[checkpoint T0, live handoff R)`.

There is no promised late-start completion time. A compatible recent
checkpoint can make recovery fast; a checkpoint-less cold start must process
all elapsed aggregate history. The launcher refuses an invocation at or after
20:00 New York time because historical/replay operation is a separate mode.

## Proven capacity boundary

The accepted no-network full cached-hydration 1x trials used the real hydration
ledger, canonical engine, bounded live queue, ingress fence, evaluator, and
readiness path. The preserved evidence-of-record trial passed with:

- 7,581,690 REST rows across 5,502 planned symbols: 5,439 value terminals and
  63 successful-empty/no-print terminals;
- 30,483 frames read, admitted, dispositioned, and fenced with zero rejection;
- maximum live queue depth 247 of 512;
- 526.912 ms fence time and 364.965 ms tail drain;
- lifecycle `live`, ranking `qualified_current`, 20 rows, and reconciled
  accounting/readiness.

The later current-source confirmation kept those semantic outcomes separate
from its own timing/sample metrics: the same 7,581,690 rows and 5,439/63
terminals; 28,377 frames read/admitted/dispositioned/fenced; queue high-water
293; 490.875 ms fence; 650.808 ms drain; zero rejection; and ready/accounting
true. The different frame and timing counts are separate paced trials and must
not be combined.

The final-source focused full-retention 2x burst proof completed a 7,745,536-row
preload followed by all 12,000 measured chunks/3,072,000 rows. It read and
dispositioned 23,448 frames with zero rejection, queue high-water 38, a 2.194 ms
drain, and reconciled accounting. This tests bounded live-tail absorption, not
a complete sustained late-start run.

The preserved sustained full 2x trial is an honest failure: 56,757 frames were
read, 56,756 admitted, 56,244 dispositioned, and 512 fenced; one capacity
rejection occurred after 7,078,180 hydration rows. Sustained full 2x is
unsupported. Do not infer that every checkpoint-less post-04:00 cold start will
finish within a fixed time, and do not repeatedly rerun the known-failing trial.

These deterministic results prove the stated local fixture/rate boundary. They
do not prove provider availability, a market-hours session, an SLA, trading
edge, or executable expectancy.

## Shutdown and failure response

Press Ctrl-C once. The launcher forwards SIGINT; a directed SIGTERM is likewise
forwarded exactly. It allows the scanner's existing bounded graceful-shutdown
path to finish, then stops and reaps the dashboard. An unexpected exit of
either process stops the other and makes the launcher exit nonzero. It never
silently restarts a failed child and never kills a process merely because a
required port is occupied.

Checkpoints are periodic coherent committed states. Shutdown does not promise
a forced final checkpoint. On failure:

1. preserve terminal output and the existing `var/checkpoints` contents;
2. do not delete checkpoints or repeatedly restart the same failing state;
3. inspect the first scanner error plus the lifecycle, suppression, and
   readiness reasons; and
4. retry the same launcher command only after identifying whether the issue is
   a port conflict, reference/credential problem, invalid schedule date,
   checkpoint fallback, hydration in progress, or honest suppression.

Credentialed provider observation remains a separate activity. It requires
explicit owner authorization for the exact trading date and validation task;
this runbook and the existence of a configured credential do not authorize a
test or live-provider request by an implementation agent.
