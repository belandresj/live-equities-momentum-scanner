# Aggregate hydration and recovery

Historical aggregate REST work establishes exact coverage behind the live
tail. It runs concurrently with live ingestion, then reconciles through an
ordered ingress fence. Fresh hydration is the supported start and restart
path.

## Planning

After the current aggregate subscription acknowledgment, the engine captures a
handoff target `R` and plans bounded symbol intervals over `[S,R)`. A plan fixes
the generation, binding, symbols, intervals, chunk budgets, worker count,
maximum response bytes, normalized-record bound, and resident-record bound.

Both the private launcher and the direct scanner command default to eight
workers, with `1|2|4|8` as the supported choices. Workers perform network and decoding work;
they do not mutate canonical state.

## Worker results

For each planned request, `internal/massive` obtains bounded REST pages,
validates response shape and query identity, normalizes rows, and returns sealed
chunks to the engine. Every request terminates exactly once as:

- completed with values;
- completed empty;
- failed;
- canceled; or
- fenced.

A successful empty result is positive no-print evidence for the requested
interval. A failed, canceled, or fenced result remains unknown. Transport
success does not imply canonical acceptance: every row still passes the common
aggregate validation and merge path.

## Canonical reconciliation

REST and live aggregates use the same `(binding, symbol, window_start)`
identity. Historical data may fill an absent identity but cannot overwrite
accepted live authority. Unequal historical rows for the same identity create a
localized conflict rather than an arbitrary winner.

The WebSocket reader remains active during REST work. Once every planned
request is terminal, the adapter inserts an aggregate-ingress fence after all
previously decoded batches. When the engine consumes the fence, it has final
dispositions for the exact live prefix through that boundary.

Reconciliation then combines:

- accepted historical values;
- exact live values and revisions through the fence;
- completed-empty no-print evidence;
- localized conflicts; and
- explicit unknown intervals from non-success outcomes.

Only then may the engine evaluate and commit the corresponding aggregate
watermark.

## Same-process gap recovery

Loss of the active aggregate epoch closes currentness. A successful reconnect
starts another bounded generation for the exact uncovered gap while the new
live tail remains active. It uses the same worker, merge, terminal ledger, and
ingress-fence semantics as startup.

A reconciled generation returns the engine to `live`. Exhausted retry or a
global integrity failure enters explicit suppression. Recovery cannot create a
second socket owner, second canonical state, or untracked background mutation.

## Work accounting

The primary hydration identity is:

```text
planned
  = open
  + completed_value
  + completed_empty
  + failed
  + canceled
  + fenced
```

Chunk and row accounting separately reconcile received, normalized, admitted,
accepted, duplicate, rejected, conflicted, and terminal work. Coverage,
qualification, Float availability, and ranking are overlapping consequences;
they are not forced into the terminal work partition.

## Restart boundary

Process restart resolves a fresh reference binding and hydrates session history.
Replay and checkpoint restoration are not implemented. The scanner does not
resume mutable state from a prior process.
