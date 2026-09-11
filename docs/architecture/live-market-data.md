# Live market-data ingestion

`internal/massive` owns provider transport, classification, normalization,
subscription delivery, and bounded ingress. It supplies facts to the engine;
it does not decide canonical merge, qualification, ranking, or readiness.

## Decoding and ordering

One WebSocket reader assigns a connection epoch, frame sequence, and receipt
time. A single decoding pass classifies each element in array order and builds
a bounded immutable batch. Each event retains its original array index.
Aggregate, trade, quote, status/control, attributable T/Q drop, ambiguous, and
unsupported elements receive explicit classifications.

The decoded batch enters one FIFO bounded by slots, frame bytes, and charged
retained bytes. Causal fence and terminal markers share that ordering boundary.
Queue saturation cannot silently discard possible aggregate/control work: safe
T/Q shedding may reduce enrichment work, while an unrecoverable admission loss
closes the epoch and requires recovery.

An invalid T/Q element can be contained locally only when its family and symbol
are trustworthy. Classification must continue to preserve later aggregate and
control facts in a mixed frame. Ambiguity that could conceal their loss becomes
an ingress-integrity failure.

## Engine handoff

The consumer transfers a complete decoded batch through
`Engine.ConsumeLiveBatch`, an unbuffered handoff to the sole owner loop. The
owner applies each logical input sequentially and returns its disposition.
Live facts bypass the engine's general typed-admission FIFO; there is no second
buffer of the same provider events.

After ownership transfers, the caller waits for completion even if its context
is canceled. This prevents cancellation from losing an admitted suffix or
stranding the owner on completion. Fences apply after the preceding admitted
batches, so hydration and ordinary evaluation can reconcile a concrete live
prefix. T/Q command results and data are linearized at the ingress boundary.

## Connection recovery

Only one connection attempt may dial, handshake, run, or clean up at a time.
The initial dial is immediate. Recovery attempts wait 1, 2, 4, 8, and 16 seconds;
exhaustion remains explicitly noncurrent. Successful fence reconciliation
resets the recovery budget, not merely a connected socket or authentication.

Heartbeat failures are interpreted with inbound progress. Supported inbound
progress can make a heartbeat error diagnostic; independent transport failure
or deadline exhaustion closes the epoch and currentness. Provider status prose
is converted into bounded reasons rather than copied into public errors.

## Retention and containment

T/Q records are optional enrichment. Pressure modes `normal`, `taq_degraded`,
and `aggregate_only` control retention, shedding, and subscription recovery.
They cannot alter ranking or make unavailable data look measured.

Trade contributions and duplicate evidence are bounded by short retention
windows and explicit per-symbol/global capacities. A watermark stall cannot pin
future-to-watermark trades indefinitely. See
[trade/quote enrichment](../features/trade-quote-enrichment.md) for the exact
window and availability distinctions.

The predecessor's raw-frame queue, second live-event FIFO, and 64-element batch
optimization are not the current production topology. Historical capacity
measurements of that path do not establish replacement-backend throughput.
