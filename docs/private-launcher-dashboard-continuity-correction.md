# Private launcher dashboard continuity correction

**Status:** Proposed direct implementation specification

**Owner:** Private launcher implementer

**Why this is separate:** This work changes two-child process supervision and
standby behavior. It requires context on `internal/privatelauncher`, the public
launcher wrapper, and dashboard process startup. It does not require engine,
provider, snapshot-mapping, or market-state changes.

## Outcome

A dashboard process failure must never signal, cancel, shut down, or restart a
healthy scanner process.

The foreground launcher remains alive and continues supervising the scanner.
It may restart only the dashboard with a finite policy. If dashboard recovery
is exhausted, the scanner continues headless until session end, controlled
operator shutdown, or its own terminal result.

This correction does not automatically restart the scanner or engine.

## Superseded launcher rule

The older launcher behavior that treats either child exit as a reason to stop
the other child is superseded only for dashboard-originated failure.

The following existing behavior remains unchanged:

- controlled operator shutdown stops scanner first and dashboard second;
- scanner startup failure stops any dashboard started for standby;
- scanner terminal exit may end the foreground launcher and stop the dashboard;
- normal session end performs the existing bounded cleanup; and
- the launcher never owns readiness, ranking, market state, or engine recovery.

## Confirmed current defect

`internal/privatelauncher.supervise` currently handles dashboard exit by sending
`SIGTERM` to the scanner and returning an error. Dashboard start and listener
verification failure after scanner liveness also stop the scanner. These paths
violate the independent-dashboard boundary.

## Required behavior

### Dashboard failure after scanner start

1. Dashboard start failure, listener-health timeout, nonzero exit, and unexpected
   zero exit are dashboard-local failures.
2. None of those events may call `Signal`, `Kill`, or another stop operation on
   the scanner child.
3. The launcher must retain scanner supervision and continue reporting scanner
   readiness when its output sink is available.
4. The launcher may start a replacement dashboard only after the prior child is
   known exited or has been boundedly stopped and reaped. At most one dashboard
   process may exist at a time.
5. Dashboard replacement makes at most three start-and-health attempts after
   one dashboard failure, with delays of 1 second, 2 seconds, and 4 seconds
   before attempts one through three. It reuses the exact existing dashboard
   binary, address, API origin, asset root, base environment, and credential
   isolation. A healthy replacement resets this budget for a later distinct
   dashboard failure.
6. A replacement must pass the same listener-health check before being reported
   available. Browser opening is not repeated automatically.
7. Retry exhaustion is visible as a dashboard-unavailable warning but leaves the
   scanner and launcher running. The operator may start `cmd/dashboard`
   independently; no fabricated dashboard health is reported.
8. Controlled shutdown during restart backoff cancels the backoff and follows
   the existing scanner-then-dashboard shutdown order.

### Dashboard failure during overnight standby

1. Dashboard failure during standby must not cancel the scheduled scanner start,
   acquire credentials early, or change the selected exchange session.
2. The same finite dashboard-only restart policy may run during standby.
3. If dashboard restart is exhausted, standby continues without a dashboard.
   At the existing preconnect boundary, the launcher still starts exactly one
   ordinary scanner.
4. The launcher must not claim that the dashboard or scanner is available while
   the corresponding child is absent.

### Terminal-output containment in the supervisor

1. Launcher progress and warning output is best effort. Loss of the foreground
   terminal or `SIGPIPE` must not make the launcher abandon a running scanner.
2. A failed launcher output sink may be latched and disabled. It must not create
   a retry loop or be confused with scanner/dashboard child failure.
3. Child credential isolation remains exact. No output correction may add the
   credential to dashboard, build, browser-opener, arguments, or diagnostics.

## Ownership and allowed implementation area

The implementation agent may change:

- `internal/privatelauncher/launcher.go`;
- `internal/privatelauncher/launcher_test.go`;
- `cmd/private-scanner-launcher/main.go` if process-level broken-pipe handling is
  required;
- `cmd/dashboard/main.go` only if its own initial terminal write would otherwise
  create a dashboard restart loop; and
- the private scanner runbook text necessary to describe dashboard-local
  recovery.

Do not change `cmd/scanner`, `internal/operations`, `internal/engine`, provider
logic, API schema/handlers, dashboard presentation, checkpoint behavior, or
market semantics in this assignment.

Keep the policy local to the existing two-child launcher. Do not introduce a
daemon, process-manager framework, `launchd` configuration, service split, or
generic supervisor package.

## Required proofs

One focused deterministic launcher suite must prove all of the following:

1. Dashboard start failure after scanner `/livez` success sends zero signals to
   the scanner and enters the dashboard-only retry policy.
2. Dashboard listener timeout sends zero signals to the scanner.
3. Dashboard nonzero and unexpected zero exit each leave the scanner running.
4. Replacement attempts are serialized, use the unchanged arguments and
   credential-free environment, and never exceed the exact retry bound.
5. A successful replacement returns to ordinary scanner/dashboard supervision.
6. Retry exhaustion leaves the launcher and scanner alive with no dashboard
   health claim.
7. Dashboard failure and retry exhaustion during standby do not acquire a
   credential or prevent the single scanner start at the preconnect boundary.
8. Controlled shutdown during dashboard backoff stops and reaps the scanner and
   any live dashboard in the existing order.
9. Scanner failure retains the existing no-auto-restart behavior and bounded
   containment; this correction does not reinterpret it as dashboard failure.
10. Broken launcher output does not end supervision or signal either child.
11. Every path leaves no unreaped dashboard child, retry goroutine, or timer.

Tests must use the existing fake child, clock, probe, and dependency seams. They
must not launch provider connections, access credentials, wait on real wall
time, or depend on arbitrary sleeps.

## Acceptance

The correction is complete when the focused launcher and wrapper proofs pass,
the ordinary short repository suite passes, and a walkthrough demonstrates:

- every dashboard-originated failure path sends zero stop signals to the
  scanner;
- scanner failure and controlled shutdown retain their existing behavior;
- dashboard retries are single-owner, finite, and credential-free;
- standby still starts exactly one scanner at the declared boundary; and
- no scanner/engine restart or market-state ownership entered the launcher.

## Explicit non-scope

- automatic scanner or engine restart;
- API listener recovery inside the scanner;
- scanner terminal-output handling;
- changing scanner startup, hydration, provider retries, or session lifecycle;
- retaining market state across scanner processes;
- public deployment or an OS daemon; and
- dashboard UI or polling changes.
