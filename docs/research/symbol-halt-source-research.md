# Symbol halt source research

> Research notes retained for reference. Proposed data sources and scanner extensions are not implemented capabilities of the live scanner.


**Researched:** 2026-09-01

## Conclusion

The strongest solution is a two-source halt model:

1. Treat Nasdaq Trader's Trade Halt RSS feed as the authoritative bootstrap and
   reconciliation source for exchange-listed symbols. It supplies the halt
   reason and scheduled quote and trade resumption times for both Nasdaq-listed
   and other exchange-listed securities, including a symbol that was already
   halted before the scanner connected.
2. Treat Massive's `LULD` WebSocket channel as a low-latency transition source.
   Indicator `17` reports a suspended/halt/pause transition and `18` reports a
   reopening transition, but Massive explicitly limits those two messages to
   Nasdaq-listed securities and does not include the reason or resumption
   schedule.

If delivery is staged, implement the Nasdaq RSS path first. It directly fixes
the observed GPRO failure mode and produces the information the trader needs to
interpret a frozen row. Add Massive `LULD` next to reduce Nasdaq halt-detection
latency below the RSS feed's one-minute update cadence.

Do not infer `halted` from a stale aggregate mark, missing trades, missing
quotes, or unconfirmed T/Q coverage. Those observations are consistent with a
halt, but also with illiquidity, a late subscription, provider loss, or a
symbol-specific data problem.

## What each source actually provides

| Source | Detection latency | Listing coverage | Startup/reconnect state | Reason and schedule | Correct role |
| --- | --- | --- | --- | --- | --- |
| Massive `LULD` WebSocket | Event-driven | Price-band events across multiple U.S. exchanges; explicit halt/reopen indicators `17`/`18` only for Nasdaq-listed securities | No documented replay or current-state response on subscribe | No | Fast Nasdaq transition evidence and optional LULD-band context |
| Nasdaq Trader Trade Halt RSS | Updated once a minute | Nasdaq-listed and other exchange-listed securities | Yes: the response is a current halt snapshot and supports date queries | Yes: reason code, halt time, quote-resumption time, and trade-resumption time | Canonical bootstrap and periodic reconciliation |
| Massive quotes or absence of A/T/Q | Event-driven | Only subscribed/received symbols | No retained halt snapshot | Quote conditions can corroborate some states; absence has no halt meaning | Corroboration only |

Massive documents the `LULD.*` subscription and the event fields `T` (symbol),
`h`/`l` (bands), `i` (indicators), `z` (tape), `t` (timestamp), and `q`
(per-symbol sequence). The same page says halt and resumption messages are only
available for Nasdaq-listed securities and currently shows individual access on
Stocks Advanced
([Massive LULD WebSocket documentation](https://massive.com/docs/websocket/stocks/luld)).
Massive's indicator glossary distinguishes routine band updates (`15`, `16`),
limit-state transitions (`23`-`30`), suspended/halt/pause (`17`), and reopening
(`18`)
([Massive conditions and indicators](https://massive.com/glossary/conditions-indicators)).
Therefore, approaching or entering a price band is not itself proof that
trading has halted.

Massive quote messages expose condition and indicator arrays, and the same
glossary defines conditions such as closed, resume, news pending, and LULD
trading pause. A quote can therefore corroborate a transition, but the quote
channel is event-driven and its schema contains no retained halt state or
scheduled resumption times
([Massive quotes WebSocket documentation](https://massive.com/docs/websocket/stocks/quotes)).

Massive's published Stocks WebSocket inventory contains no separate
symbol-level security-status or trading-status channel
([Massive Stocks WebSocket overview](https://massive.com/docs/websocket/stocks/overview)).
Its published Stocks REST inventory contains no symbol-level halt endpoint and
no REST LULD endpoint
([Massive Stocks REST overview](https://massive.com/docs/rest/stocks)). The
documented `GET /v1/marketstatus/now` endpoint describes overall markets,
exchanges, and session hours rather than an individual symbol's status
([Massive market-status documentation](https://massive.com/docs/rest/stocks/market-operations/market-status)).
Massive REST therefore does not provide a documented way to reconstruct a halt
missed before WebSocket subscription.

Nasdaq calls its Trade Halt RSS feed a free service containing the same halt
and pause information as its Trading Halts page for Nasdaq-listed and other
exchange-listed securities. Nasdaq says the data updates once per trading-day
minute and instructs consumers not to query more frequently than once a minute;
the page also documents current, halt-date, and resumption-date queries
([Nasdaq Trader Trade Halt RSS](https://www.nasdaqtrader.com/Trader.aspx?id=TradeHaltRSS)).
The RSS field contract includes halt date/time, symbol, market, reason code,
pause threshold, resumption date, scheduled quote-resumption time, and scheduled
trade-resumption time. Nasdaq's reason table distinguishes news and regulatory
halts from volatility pauses and identifies `T3` as news disseminated with
resumption times, while `T7` means quotations have resumed but trading remains
paused
([Nasdaq Trader halt fields and codes](https://www.nasdaqtrader.com/Trader.aspx?id=TradeHaltCodes)).

## Recommended scanner behavior

### Provider boundary

- Start a bounded Nasdaq RSS poller outside the engine's ordered mutation path.
  Fetch once at startup, after prolonged source failure, and no more than once
  per minute during the scanner session. Apply HTTP timeout, response-byte,
  item-count, XML-depth, and timestamp bounds. A failed or malformed poll makes
  halt coverage stale; it must not clear known state.
- Normalize each RSS item into a closed `HaltFact` containing source, symbol,
  listing market, halt identity `(halt_date, halt_time, symbol)`, reason code,
  halt time, optional scheduled quote/trade resumption times, source publication
  time, and retrieval time. Repeated polls revise the same halt identity because
  its reason and resumption schedule can change.
- Subscribe to Massive `LULD.*` from the beginning of each live connection, not
  only after a symbol reaches the displayed top 20. A selected-row subscription
  can miss the event that caused the pre-selection silence. Normalize indicators
  `17` and `18` as transition facts; ordinary band updates should be dropped or
  retained only under a separately bounded LULD feature requirement.
- The current Massive normalizer recognizes only `A`, `T`, `Q`, and transport
  `status`; subscribing before adding an explicit `LULD` family would classify
  those messages as unsupported
  ([live_normalization.go](../../internal/massive/live_normalization.go)). The
  full-market LULD stream must participate in raw-frame capacity accounting and
  must not weaken aggregate-priority containment
  ([live-market-data architecture](../architecture/live-market-data.md)).

### Canonical state and publication

- Admit normalized halt facts through the `ScannerStateEngine` FIFO so there is
  still one mutable owner. Halt state is display and operational context only:
  it must not qualify, exclude, reorder, advance the aggregate watermark, or
  make aggregate readiness depend on either external source.
- Publish a per-row state such as `halted`, `quotation_only`,
  `resumption_scheduled`, `resumption_due`, `resumed`, or `unknown`, plus reason,
  halt time, optional quote/trade times, source, and source freshness. Keep
  `unknown` distinct from `not_halted`; source failure is not negative evidence.
- A scheduled resumption time is a schedule, not proof that data actually
  resumed. At the scheduled trade time, change `halted` to `resumption_due` and
  clear the halt only on Massive indicator `18`, an authoritative later status
  revision, or valid post-halt live market evidence under an explicit rule.
  Quotes after their scheduled time can establish `quotation_only`; they do not
  prove that trading resumed.
- The dashboard should replace the misleading blank interpretation with an
  explicit badge and the most useful schedule, for example `HALTED · T3`,
  `QUOTES 09:55`, and `TRADING 10:00`. Tape and Spread keep their independent
  availability states; the halt explains their absence but must not fabricate
  values.

## Important limitations and proofs required

- Nasdaq documents exchange-listed coverage, not every possible OTC security.
  A symbol outside that source's declared coverage must remain `unknown` unless
  another authoritative status source is added.
- RSS is too slow to be the sole low-latency risk signal, while Massive `LULD`
  is incomplete for non-Nasdaq explicit halt/reopen events. Neither source alone
  meets the full product need.
- Massive's LULD page labels `t` as Unix milliseconds, but its sample is a
  19-digit value. Timestamp units must be proven with an owner-approved captured
  fixture before implementation rather than guessed from the sample.
- Focused verification should cover: a halt already active at startup; a live
  `17` transition; RSS revision from news-pending to scheduled resumption; quote
  resumption before trade resumption; duplicate polls; reconnect without replay;
  malformed/stale RSS; non-Nasdaq RSS-only coverage; and continued aggregate
  delivery through mixed frames and LULD load.
- This research establishes source semantics and a design direction. It does
  not establish Massive entitlement on the configured account or observed live
  LULD frame shape; checking either would require separately authorized
  credentialed provider work.
