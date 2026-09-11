# Dashboard

The dashboard is an independently running browser client for the snapshot API.
It presents backend facts; it does not calculate qualification, market
features, coverage, readiness, or rank.

## Runtime boundary

`cmd/dashboard` serves the static assets in `ui/` on loopback, normally
`127.0.0.1:4173`. The browser reads the scanner at `127.0.0.1:8080`. The private
launcher supervises both processes, but a dashboard failure does not stop the
scanner and a dashboard restart does not reset market state.

The client polls the snapshot once per second with a three-second request
timeout. Every response is strictly validated as `scanner.snapshot.v2` before
it can replace the displayed model. The client preserves server row order.

## Display

The table has 12 leaf columns:

```text
RANK SYMBOL FLOAT VOLUME LAST FROM CLOSE % FROM OPEN % DAY RANGE
ACTIVITY 30s MOVE 30s TAPE SPEED SPREAD
```

Formatting, units, responsive layout, tooltips, keyboard focus, accessible
labels, color, and unavailable-state rendering are presentation concerns. The
browser may not turn unavailable values into numeric zero or silently omit rows
because an enrichment is absent.

Tape Speed uses an absolute continuous RGB gradient rather than discrete color
bands. Rates from 0 through 5 trades/second are neutral gray. Color then
interpolates through restrained orange at 15 trades/second and strong orange at
50 trades/second to maximum orange at 100 trades/second. Every rate at or above
100 trades/second uses the same maximum orange. The scale is not normalized to
the currently displayed rows.

## Rank movement

The backend sends current rank only. The client keeps bounded in-memory snapshot
history and compares a row with the closest valid displayed snapshot roughly 60
seconds earlier. This arrow/amount is presentation-local and resets with the
page. It cannot sort, smooth, delay, or filter server rows.

## Status behavior

The primary status distinguishes exact current operation (`LIVE`) from warming,
partial, disconnected, stale, suppressed, or ended states. An exact current
zero-row result remains `LIVE`; it is not confused with failure.

T/Q degradation is shown separately from aggregate ranking state. A T/Q warning
cannot demote an otherwise exact aggregate result, and an apparently healthy
Tape/Spread cell cannot upgrade a partial aggregate result.

When transport fails or a response is invalid, the client visibly marks the
state and may retain the last valid rows for orientation. Retained rows must not
appear current. A later valid snapshot replaces the model without replaying
intermediate browser-side market calculations.

## Verification boundary

UI tests exercise strict schema validation, row ordering, status mapping,
field formatting, movement history, keyboard/focus continuity, and failure
retention. They do not prove the correctness of the backend market formulas;
those are tested in the engine and snapshot mapper.
