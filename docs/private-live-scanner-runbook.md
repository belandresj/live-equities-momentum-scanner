# Private live scanner runbook

This is the supported private/local daily workflow. Start the foreground
launcher at any time from the repository root:

```text
./scripts/run-private-scanner
```

Starting several minutes before 04:00 remains the shortest path and gives
preflight and compilation time to finish before the scanner session begins.
An earlier invocation enters calendar-aware overnight standby: the dashboard
runs immediately, while credential acquisition, reference requests, the
scanner process, and provider connections wait until 03:55 for the current or
next exchange-declared trading day. This is an operational timing difference,
not a different ranking mode.

The standby loop rechecks New York wall time and the exchange schedule every
30 seconds. Sleeping or pausing the Mac through a selected session therefore
cannot start that stale date: an implicit launch rolls to the next declared
session, while an explicit missed `--trading-date` exits with an error.

## Prerequisites

- macOS with the repository's Go 1.26 toolchain available as `go`;
- an exact-date reference cache under `var/reference`, or ordinary provider
  access capable of resolving it during scanner startup;
- a writable `var/reference` directory;
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

`--open` asks macOS to open the dashboard once its listener is healthy; during
standby the scanner listener is intentionally absent. A browser-open failure is
nonfatal. `--trading-date` is for an exact date correction; it does not
authorize a historical provider request. The launcher and
scanner use the same validated exchange schedule. Automatic selection skips
weekends and holidays; an explicit weekend, holiday, unsupported, or ended date
is rejected. Hydration defaults to eight workers; `--hydration-workers`
accepts exactly `1`, `2`, `4`, or `8`. This changes only bounded REST
acquisition concurrency. It does not
change the universe, interval, merge rules, ranking, readiness, or T/Q path.

The launcher invokes the scanner with these exact settings:

```text
--trading-date <current America/New_York date or explicit override>
--hydration-workers 8
--reference-dir <repo>/var/reference
--diagnostic-dir <repo>/var/diagnostics (scanner default)
--api-address 127.0.0.1:8080
--allow-origin http://127.0.0.1:4173
```

For the 2026-08-12 owner-run retry, the scanner's internal bounded delivery
settings are 4,096 decoded-batch slots, 64 MiB total queued payload, 8 MiB per
frame, a 4-GiB cumulative (not resident) hydration-transfer allowance, at most
57,600 resident normalized records per configured hydration worker, and five
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

The launcher forwards scanner and dashboard stdout/stderr to the foreground
terminal; it does not create or rotate a per-run stdout log. The binaries under
`var/run-private-scanner/bin` are rebuilt in place on the next launch, so they
are not run logs. On the first typed ingress incident, the scanner writes one
structured JSON diagnostic under `var/diagnostics` using a timestamped,
create-without-overwrite filename. Separate incident files therefore remain
available across runs. These diagnostics are failure evidence, not a copy of
stdout/stderr.

The scanner retains at most one fixed-cardinality ingress incident per process.
On the first typed incident—including one that enters recoverable gap recovery
and later succeeds—it immediately prints the bounded source/reason and process
evidence to stderr and writes a create-without-overwrite JSON record under
`var/diagnostics` with owner-only directory/file permissions. The evidence
includes connection epoch, hydration purpose/generation/activity and work
accounting, absolute queue operands, processing-delay maxima, heap allocation/
in-use bytes, goroutine count, and at most 60 one-second samples. It contains no
credentials, provider URLs/prose, symbols, or raw payloads. Later incidents in
the same process do not overwrite the first-cause record.

The scanner also retains at most 120 ordinary live evaluation-cycle records for
the separate watermark-staleness diagnostic. It writes
`watermark-stale-<UTC timestamp>.json` only when a sampled transition is exactly
`backend_ready=true` to `backend_ready=false` with reason `watermark_stale`.
The record distinguishes live-coverage fence timing, timer/evaluator timing,
publication and queue delay, T/Q pressure, and cached heap/goroutine/GC facts;
it contains no symbols, provider frames, payloads, credentials, or API capture.
One process makes at most one persistence attempt, including when that attempt
fails. A missing file therefore means this exact transition was not observed or
the best-effort diagnostic write failed; it is not evidence that the backend
remained ready. Preserve the file with the terminal output during an owner-
authorized soak, and do not treat it as a root-cause or stability proof.

## URLs and status interpretation

In immediate mode the launcher starts the dashboard after scanner `/livez`
succeeds. In overnight standby the dashboard starts first, reports the scanner
as disconnected, and keeps polling until the scanner starts at 03:55. The
dashboard prints its listener once; the launcher prints the scanner URLs once
the real API is live. A dashboard start failure, unhealthy listener, or
unexpected exit never stops or restarts a healthy scanner. The launcher waits
1, 2, then 4 seconds before at most three serialized dashboard replacements.
Each replacement uses the same loopback arguments and credential-free
environment and must pass the listener check. If all three fail, the launcher
reports dashboard unavailable and continues supervising the scanner headless;
the operator may start `cmd/dashboard` independently. It does not reopen the
browser automatically after a replacement.

- dashboard: `http://127.0.0.1:4173`;
- scanner snapshot: `http://127.0.0.1:8080/api/v2/snapshot`;
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

Before 03:55, the launcher runs only the dashboard and displays the scanner as
disconnected. At 03:55 it starts the ordinary live scanner for the resolved
trading date. From then until 04:00, the runtime waits in its engine-owned
`awaiting_session` state and then advances incrementally. This avoids both
spending finite provider retries overnight and creating a large elapsed-session
REST backlog; it does not skip any symbol, use top-N hydration, or alter
qualification/ranking.

From 04:00 through 20:00 New York time, the launcher prints a warning and still
starts normally. The current private path uses fresh hydration on every
restart: it hydrates the full elapsed session, consumes the buffered live tail,
applies the exact ingress fence, and only then reports ready.

There is no promised late-start completion time; a late start must process all
elapsed aggregate history. At or after 20:00, and on weekends or exchange
holidays, an invocation without `--trading-date` selects the next declared
trading session and enters standby. An explicit unsupported or already-ended
trading date still fails; the supported scanner remains live-only.

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
unsupported. Do not infer that every post-04:00 cold start will
finish within a fixed time, and do not repeatedly rerun the known-failing trial.

These deterministic results prove the stated local fixture/rate boundary. They
do not prove provider availability, a market-hours session, an SLA, trading
edge, or executable expectancy.

## Shutdown and failure response

Press Ctrl-C once. The launcher forwards SIGINT; a directed SIGTERM is likewise
forwarded exactly. It allows the scanner's existing bounded graceful-shutdown
path to finish, then stops and reaps the dashboard. Scanner startup/terminal
failure retains the existing bounded containment and no-auto-restart behavior.
Dashboard failure is different: it is recovered locally under the finite policy
above and never signals a healthy scanner. A shutdown during dashboard backoff
cancels that wait and still follows scanner-then-dashboard cleanup. It never
kills a process merely because a required port is occupied.

The current private path always restarts through fresh hydration. On failure:

1. preserve terminal output and any `var/diagnostics` incident file;
2. do not repeatedly restart the same failing state;
3. inspect the first scanner error plus the lifecycle, suppression, and
   readiness reasons; and
4. retry the same launcher command only after identifying whether the issue is
   a port conflict, reference/credential problem, invalid schedule date,
   hydration in progress, or honest suppression.

Credentialed provider observation remains a separate activity. It requires
explicit owner authorization for the exact trading date and validation task;
this runbook and the existence of a configured credential do not authorize a
test or live-provider request by an implementation agent.
