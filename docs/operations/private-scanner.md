# Private scanner operation

The supported operating path is one fresh-start local scanner and one optional
local dashboard for a single exchange trading date.

## Launch

From the repository root on macOS:

```text
./scripts/run-private-scanner [--trading-date YYYY-MM-DD] [--hydration-workers 1|2|4|8] [--open]
```

Without an explicit date, the launcher resolves the applicable trading session
from the embedded schedule and current New York time. It defaults to eight
hydration workers. `--open` asks macOS to open the dashboard after healthy
startup; failure to open a browser is nonfatal.

The launcher builds private runtime binaries, checks its fixed loopback ports,
and starts the dashboard during overnight standby when there is enough time to
do so safely. Scanner credential, reference, and provider work wait until five
minutes before the 04:00 New York session start. A dashboard failure during
standby does not cancel the scheduled scanner start.

## Credentials

The launcher obtains the Massive credential at scanner start from its supported
environment/Keychain boundary and passes it only to the scanner child. It does
not read the credential during overnight standby. The dashboard never receives
it. An existing nonempty `MASSIVE_API_KEY` takes precedence. On macOS, the
fallback looks up the scanner service under the current user account; the
repository does not embed an operator account name. `.env` files are ignored
by Git but are not automatically loaded by the launcher.

Do not print, inspect, copy, test, or otherwise access provider credentials
without explicit authorization. Do not make a live provider request merely
because a credential is configured or this runbook exists.

## Local endpoints

```text
dashboard  http://127.0.0.1:4173
snapshot   http://127.0.0.1:8080/api/v2/snapshot
liveness   http://127.0.0.1:8080/livez
readiness  http://127.0.0.1:8080/readyz
```

Liveness means the process/capture path is responding. Readiness means the
aggregate backend has a reconciled current publication. T/Q pressure is
reported separately and does not by itself change readiness.

## Startup expectations

Before 04:00 the scanner may be live but not market-ready. After the aggregate
subscription boundary, it hydrates `[S,R)` while buffering and processing the
live tail. Readiness becomes true only after terminal hydration work, ordered
fence reconciliation, current evaluation, and valid accounting.

Starting after 04:00 is supported: hydration covers the elapsed interval. Work
and readiness time grow with the universe, elapsed session, provider response,
host, and worker count. A completed empty interval is valid no-print evidence;
it is not stuck work.

## Supervision and shutdown

The dashboard can fail and be restarted without stopping the scanner. If the
scanner exits, the launcher contains the dashboard and returns failure. At the
session end or on an interrupt, the launcher gives both children bounded
shutdown time and escalates only when a child does not exit.

The launcher does not automatically restart a failed scanner into the same
unexplained market state. The supported recovery after process loss is a clean
new process with fresh reference resolution and hydration.

## Direct scanner command

`cmd/scanner` exposes lower-level flags for deterministic tests and controlled
operation. Both it and the private launcher default to eight hydration
workers and accept `1|2|4|8`. Replay and checkpoint flags are not supported. Provider origins, diagnostic directories, API origins, and
other lower-level flags should not be changed casually because they define
trust and containment boundaries.

