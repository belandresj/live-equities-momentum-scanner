# Scanner output and API continuity correction

**Status:** Proposed direct implementation specification

**Owner:** Scanner backend implementer

**Why this is separate:** This work changes failure handling inside the scanner
process. It requires context on `cmd/scanner`, `internal/snapshotapi`, runtime
shutdown, and API publication. It does not require private-launcher child
supervision changes.

## Outcome

After the live scanner has started, failure of terminal output or the snapshot
API listener must not cancel, shut down, replace, or rehydrate the current
`operations.Runtime` or its engine.

The scanner must continue consuming aggregates, evaluating ranking, updating
the committed watermark, and maintaining T/Q state while:

- stdout or stderr is closed or returns a write error;
- an operator-status render fails;
- a snapshot-mapping diagnostic cannot be written; or
- the running snapshot API server terminates unexpectedly.

The API may be recreated against the same runtime and immutable snapshot source.
No engine replacement is part of this correction.

## Confirmed current defects

`cmd/scanner/main.go` currently performs joined scanner/API shutdown when:

1. the initial operator render fails;
2. a periodic operator render fails;
3. encoding a snapshot-mapping diagnostic fails; or
4. `snapshotapi.Server.Done()` reports an unexpected terminal result.

These consequences conflict with the existing architecture rule that API/UI and
field-local failures cannot freeze aggregate evaluation.

## Required behavior

### Terminal output

1. Operator and diagnostic output is best-effort observability. A write error
   must never call `joinAndShutdown`, cancel `runCtx`, or cause `run` to return.
2. A failed stdout and a failed stderr are tracked independently. After a sink
   fails, the scanner must stop repeatedly writing to that sink.
3. The scanner may make one best-effort attempt to report a sink failure through
   the other still-healthy sink. Failure of that report is also nonfatal and
   must not recurse.
4. The scanner process must not terminate from `SIGPIPE` caused by its stdout or
   stderr disappearing. The implementation must convert this condition into an
   ordinary failed write or otherwise isolate the scanner from the fragile
   terminal.
5. Failure to persist an ingress incident remains distinct from terminal-output
   failure. It may be retained in shutdown diagnostics, but it must not stop an
   otherwise-running scanner merely because its explanatory terminal write also
   failed.
6. Failure to capture or validate an operational snapshot is not automatically
   an output failure. Existing integrity handling for an invalid runtime or
   publication may remain terminal.
7. This correction adds no API-v2 field for terminal health. Bounded internal
   state or counters are sufficient.

### Snapshot API

1. Initial API configuration and bind failure remains a startup failure. In
   particular, an occupied configured port must not cause the scanner to run
   behind an unknown endpoint.
2. Once the API has successfully started, an unexpected server termination must
   not shut down or replace the runtime.
3. The scanner must recreate the API server with the same address, allowed
   origins, runtime capture source, and schema behavior.
4. Restart attempts must be serialized and bounded. Make at most five
   replacement attempts after one unexpected server termination, with delays of
   250 milliseconds, 500 milliseconds, 1 second, 2 seconds, and 4 seconds
   before attempts one through five. Do not create overlapping listeners,
   retry goroutines, or timers. A successfully started replacement resets this
   budget for a later distinct server termination.
5. While API restart is pending or exhausted, the runtime continues normally.
   The dashboard may observe connection failure; no stale response may be
   fabricated.
6. If the bounded restart policy is exhausted, keep the runtime alive and mark
   the API unavailable in scanner-local operational diagnostics. Do not exit the
   process solely because the API is unavailable.
7. A later controlled stop, session end, or genuine runtime/engine terminal
   result still shuts down the currently installed API server and joins all
   scanner work through the existing bounded shutdown path.
8. A controlled API shutdown initiated by scanner termination must not be
   mistaken for an unexpected API failure or schedule a restart.
9. Each replacement API instance owns a new `Done()` and mapping-failure
   channel. Events from retired instances must be fenced and cannot affect the
   active instance.
10. Snapshot mapping failure remains fail-closed for that HTTP response. This
    correction changes only failure to write the diagnostic, not schema or
    mapping validation.

## Ownership and allowed implementation area

The implementation agent may change:

- `cmd/scanner/main.go`;
- focused tests under `cmd/scanner`;
- `internal/snapshotapi/server.go` and its tests only if a small lifecycle seam
  is needed; and
- small scanner-local helper files where they reduce lifecycle complexity.

Do not change engine state, aggregate/T/Q semantics, readiness derivation,
snapshot schema, dashboard behavior, provider retry policy, checkpoint mode, or
the private launcher in this assignment.

Prefer a scanner-local API supervisor/state machine over a new service,
framework, or generic restart abstraction. The existing runtime remains the
single capture source for every API generation.

## Required proofs

One focused deterministic suite must prove all of the following:

1. Initial operator output failure leaves the runtime and API running.
2. Periodic stdout failure is latched, is not retried every tick, and aggregate
   evaluation continues across later ticks.
3. Stderr failure while encoding a mapping diagnostic does not stop the runtime
   or API and does not recurse through the other sink.
4. A broken-pipe output condition does not terminate the scanner process.
5. Unexpected post-start API termination creates exactly one replacement at a
   time; the replacement serves a coherent snapshot from the same runtime and
   publication sequence continues rather than resetting.
6. Failed API restart attempts follow the exact finite backoff policy. Exhaustion
   leaves the engine running with no fabricated API availability.
7. A controlled stop during API backoff cancels the timer, joins the active or
   retired server exactly once, shuts down the runtime, and leaves no goroutine
   blocked on an obsolete `Done()` channel.
8. A runtime/engine terminal result still exits through the current shutdown
   path; output/API containment cannot conceal genuine engine termination.
9. Initial address conflict remains fail-fast and never starts live market
   processing.

Tests must use injected writers/listeners or server factories. They must not use
provider credentials, live requests, real market time, or arbitrary sleeps.

## Acceptance

The correction is complete when the focused proofs pass, the ordinary short
repository suite passes, and a code walkthrough demonstrates these exact
properties:

- output failure cannot reach runtime cancellation;
- post-start API failure cannot reach runtime shutdown;
- every API replacement reads the same runtime until genuine scanner shutdown;
- retry and cleanup work is finite; and
- no engine restart, second engine, second canonical state, or API schema change
  was introduced.

## Explicit non-scope

- automatic scanner or engine restart;
- hot-swapping an engine behind the API;
- changing same-binding aggregate recovery or recovery exhaustion;
- public hosting, authentication, TLS, or a scanner/API service split;
- dashboard child supervision;
- log rotation, remote logging, or a general observability subsystem; and
- replay or checkpoint behavior.
