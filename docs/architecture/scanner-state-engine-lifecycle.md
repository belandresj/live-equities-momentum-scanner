# Scanner state engine lifecycle

One `ScannerStateEngine` instance owns one session binding, one canonical
state graph, and one immutable publication cell. Live batches arrive through
an unbuffered handoff after the decoded-batch FIFO; other typed admissions
share the same engine owner. Lifecycle changes occur only on that owner path.

## States

| State | Meaning |
| --- | --- |
| `initializing` | Engine exists but the immutable binding has not completed installation. |
| `awaiting_session` | Binding is valid, but engine time is before session start. |
| `awaiting_aggregate_ack` | Session is active and the live aggregate epoch has not been established. |
| `hydrating` | A fresh-start aggregate-history generation is active while the live tail continues. |
| `live` | Startup hydration and ingress fence are terminal; ordinary timers/fences may publish current ranking. |
| `recovering` | A same-process aggregate gap is being reacquired and reconciled. |
| `suppressed` | A global integrity/currentness claim cannot be made; an explicit recovery disposition is published. |
| `ended` | Session end, controlled stop, or engine closure is terminal for this instance. |

Transport status, ranking mode, T/Q pressure, field availability, API health,
and process liveness are separate dimensions; none creates a hidden lifecycle
state.

## Initialization

The binding installation independently validates schedule, universe, prior
closes, identities, and accounting. Depending on engine time, a valid live
binding enters `awaiting_session`, `awaiting_aggregate_ack`, or `ended` if the
session has already passed.

The live launcher opens the provider connection near 03:55 New York time.
Before `S`, timers preserve `awaiting_session`. At `S`, the engine waits for a
current-epoch aggregate subscription boundary. Connected/authenticated status
alone is insufficient.

## Startup hydration

The current-epoch aggregate acknowledgment establishes the live handoff and
starts one fresh hydration generation. The engine allocates exact symbol
requests over `[S,R)`, where `R` is the captured handoff target. The WebSocket
reader and decoded-batch FIFO remain active throughout REST work.

Every planned request reaches one terminal work state. Successful empty results
are terminal. Failed, canceled, and fenced work remain explicit. After worker
completion, the adapter captures the aggregate-ingress fence in decoded-batch order.

Fence reconciliation installs exact present/no-print/conflict consequences and
evaluates the population. Startup must then leave `hydrating`: success enters
`live`; a recoverable epoch loss enters recovery; unreconcilable global
integrity enters suppression. There is no terminal “hydration complete but
ranking frozen” substate.

## Ordinary live progress

In `live`:

- live aggregates, trades, quotes, T/Q drops, and control facts continue through
  the FIFO;
- live-coverage fences and one-second timers provide supported evaluation
  targets;
- accepted aggregate changes stage and apply through the single evaluator;
- current qualified rows drive T/Q intent;
- ordinary T/Q mutations may be coalesced to cadence publication while trust
  transitions publish immediately; and
- API/UI readers cannot delay engine progress.

A quiet market still advances through timers/fences when coverage is proven. A
first print after no-print changes only that symbol and follows normal
evaluation.

## Aggregate loss and recovery

Loss of the active aggregate epoch immediately invalidates aggregate currentness
and closes all T/Q coverage. The engine records the failed epoch, schedules a
bounded same-binding reconnect, and prevents stale-epoch facts from mutating
state.

A recovered aggregate epoch starts a gap-hydration generation over the exact
uncovered interval. The same REST worker, canonical merge, work ledger, and
ingress-fence process used at startup applies. A reconciled fence returns to
`live`; exhausted retries or unreconciled global loss enters `suppressed` with
an explicit same-binding, clean-reinitialization, or restart disposition.

Recovery delays and attempts are bounded by runtime policy; lifecycle
transitions remain engine-owned. Merely sleeping or observing suppression
does not authorize a second socket or state owner.

## T/Q interaction

T/Q desire exists only for rows in an exact `qualified_current` ranking. A
subscribe write records requested membership and a causal boundary. The first
post-boundary trade and first post-boundary quote independently activate their
coverage.

T/Q errors are subordinate:

- invalid trades/quotes are rejected or fenced within T/Q;
- command ambiguity quarantines T/Q for the epoch;
- pressure can enter `taq_degraded` or `aggregate_only` and close coverage;
- retained-state bounds force aggregate-only containment; and
- later healthy evidence restores T/Q through fresh subscriptions and warm-up.

A stale aggregate watermark alone masks T/Q values at API capture while
subscription and ingestion continue. The recovery visibility hold is defined
in [trade/quote enrichment](../features/trade-quote-enrichment.md).

None of these transitions changes top-level lifecycle or aggregate rank unless
the underlying shared transport also loses aggregate/control integrity.

## Suppression

Suppression is entered for global conditions such as:

- engine sequence exhaustion;
- engine clock regression;
- repeated unequal aggregate data at the same authoritative position;
- ingress ambiguity that may have lost aggregate/control facts;
- failed population, transition, or publication accounting;
- publication construction/validation failure; or
- exhausted aggregate recovery without a trustworthy current boundary.

The publication names the reason and one of:

- `same_binding_recovery_allowed`;
- `clean_reinitialization_required`;
- `restart_required`.

Suppression never fabricates an empty or stale-ready table. A same-binding
recovery transition must be an ordered engine action; otherwise the instance
ends and a new engine performs fresh initialization.

## Session end and shutdown

At `E`, controlled stop, or sealed engine closure, the owner fences/finishes
admitted work within bounded shutdown and enters `ended`. `ended` is terminal.
The API may retain the final immutable publication while reporting the process
or session state accurately.

Dashboard failure does not stop the scanner. Scanner failure stops the daily
launcher after bounded containment; it is not automatically restarted into the
same unexplained state.

## Publication and readiness

`process_live` means the backend is running. It does not imply market
currentness.

Backend readiness requires all of:

- matching session binding;
- a lifecycle allowed to make the current market claim;
- active acknowledged aggregate connection;
- reconciled startup/recovery fence;
- current ranking at the causal target within readiness tolerance; and
- valid closed accounting.

T/Q pressure or field unavailability alone does not make the backend unready.
A legitimate exact zero-row qualified result can be ready. Partial ranking can
be current as an explicitly degraded view but is not an exact qualified result.

## Progress and accounting invariants

1. Every waiting state names a bounded external or timer progress event.
2. Every planned hydration request reaches one terminal disposition.
3. Every admitted external input receives one completed transition category.
4. Every transition receives one publication decision: no exposed change,
   replacement, or integrity failure.
5. Aggregate changes cannot become permanently invisible after hydration or
   recovery becomes terminal.
6. T/Q, API, UI, or field-local failure cannot freeze aggregate
   evaluation.
7. `ended` is terminal for one engine instance and binding.
