# Snapshot API v2

The snapshot API is a private loopback, read-only view of one immutable engine
publication plus scanner-local operational status. It performs validation and
mapping; it does not calculate market features or feed state back into the
engine.

## Routes

| Route | Meaning |
| --- | --- |
| `GET /api/v2/snapshot` | Full `scanner.snapshot.v2` document. |
| `GET /livez` | Process and capture-path liveness as `scanner.liveness.v1`. |
| `GET /readyz` | Aggregate backend readiness as `scanner.readiness.v1`. |

`HEAD` is supported with the same status and headers and no body. Other methods
are rejected. Query strings and request bodies are rejected. There is no
supported `/api/v1` route.

The server binds to loopback by default. Browser access uses an exact configured
origin allowlist; CORS does not accept wildcard origins, credentials, or custom
request headers. Responses are JSON, `Cache-Control: no-store`,
`X-Content-Type-Options: nosniff`, and bounded to 1 MiB.

## Capture and coherence

One request obtains one immutable `operations.SnapshotCapture`. Mapping checks
the capture, engine publication, population identities, row/ranking agreement,
T/Q view, and operations metadata as one coherent sample. It rejects a mixed or
invalid join rather than returning a partially mapped success.

Client cancellation may stop capture or mapping before the response is written.
It cannot mutate the engine or replace the current publication. Mapping failures
are recorded as bounded scanner-local diagnostics without provider payloads or
credentials.

## Snapshot shape

The top-level schema is:

```text
schema_version = scanner.snapshot.v2
sample
publication
status
ranking
rows[]
accounting
recovery
tq
checkpoint
operations
```

`checkpoint` is a fixed disabled compatibility object in API v2; it does not
represent a checkpoint package, writer, or restart capability.

`sample` identifies the API capture time. `publication` identifies the immutable
engine publication, binding, lifecycle, committed `T`, engine sequence,
connection epoch, acknowledgment, and hydration fence. These are different
clocks and identifiers.

`status` separates process liveness, backend readiness, ranking currentness,
watermark lag, accounting validity, and T/Q pressure. T/Q degradation alone
does not make the aggregate backend unready.

`ranking` states whether the row order is exact qualified ranking or a visibly
degraded partial view. `accounting`, `recovery`, `tq`, and `operations` expose
closed identities as decimal-safe fields where counts may exceed JavaScript's
exact integer range.

## Rows

Each row contains:

- server rank and canonical symbol;
- Float, Volume, Last, From Close, and mark age;
- From Open, Day Range, Activity 30s, and Move 30s;
- Tape 5s and Spread; and
- desired/provider T/Q membership.

Measurements with independent availability use a status, reason, and nullable
numeric value. A missing number is not encoded as zero. `DayChangeRatio` and
`LastUSD` are present only for rankable rows and are the exact backend values
used for the published ordering.

## HTTP status meaning

`/livez` returns success when the scanner runtime and capture source are alive;
it does not claim market readiness. `/readyz` returns success only when the
mapped snapshot says the aggregate backend is ready. A snapshot may still be
served while warming, partial, suppressed, or T/Q-degraded so the dashboard can
explain the state.

A capture or mapping failure returns a bounded unavailable error. The handler
never substitutes an empty table for an invalid publication.

The Go structs in `internal/snapshotapi/schema.go` are the exact wire field
inventory. Any schema change requires coordinated mapper, validator, dashboard,
fixture, and documentation changes and a new version when compatibility cannot
be preserved.

