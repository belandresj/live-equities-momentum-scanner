# Massive WebSocket subscription acknowledgement research

**Researched:** 2026-08-18

## Conclusion

Massive's public WebSocket documentation supports comma-separated batched
subscriptions and wildcards, but it does not specify a structured
post-subscription acknowledgement contract, acknowledgement count, delivery
deadline, or retry policy. The scanner's rule that a T/Q command must receive
exactly two `success` statuses per symbol within five seconds is therefore an
application-specific assumption, not a documented Massive requirement.

Massive's first-party Python mock server emits one `success` status per topic
and puts the topic in human-readable `message` text, for example
`subscribed to: T.AAPL`. That is useful behavioral evidence, but the topic is
not carried in a documented structured field or command identifier.

## Primary-source findings

- The [WebSocket quickstart](https://massive.com/docs/websocket/quickstart)
  sends one comma-separated subscription command and proceeds directly to data
  messages. It specifies no subscription-success count, timing SLA, or retry
  rule.
- The [Stocks Trades documentation](https://massive.com/docs/websocket/stocks/trades)
  permits one ticker, a comma-separated ticker list, or `*` for all tickers.
- Massive says there is [no server-side ticker-count limit](https://massive.com/knowledge-base/article/how-many-tickers-can-you-subscribe-to-on-a-single-massive-websocket-connection)
  as long as the client can consume the data, subject to product-specific
  exceptions and command-size limits.
- The [official Python mock server](https://github.com/massive-com/client-python/blob/481e5c270ea85e8eae5e96f8b9fda34e5e2a674a/test_websocket/mock_server.py#L21-L34)
  emits one success per comma-separated topic and names that topic only in the
  `message` string.
- The [official Go control model](https://github.com/massive-com/client-go/blob/eef5a9ae787a117b0d0701c211b209226e478ebe/websocket/models/models.go#L17-L24)
  has `ev`, `status`, `message`, `action`, and `params`, but no request ID or
  structured acknowledgement topic.
- The [official Go subscribe path](https://github.com/massive-com/client-go/blob/eef5a9ae787a117b0d0701c211b209226e478ebe/websocket/client.go#L120-L167)
  records the requested subscription and queues the command without awaiting a
  status. Its [status handler](https://github.com/massive-com/client-go/blob/eef5a9ae787a117b0d0701c211b209226e478ebe/websocket/client.go#L456-L479)
  logs `success` and `error`; only authentication failure is fatal.
- The [official Python client](https://github.com/massive-com/client-python/blob/481e5c270ea85e8eae5e96f8b9fda34e5e2a674a/massive/websocket/__init__.py#L198-L214)
  likewise does not wait for or count subscription-success statuses.
- Massive's official Go client automatically reconnects and resends its locally
  retained subscriptions after an actual disconnect, but Massive documents no
  missing-status retry policy. See the [reconnect implementation](https://github.com/massive-com/client-go/blob/eef5a9ae787a117b0d0701c211b209226e478ebe/websocket/client.go#L244-L312).

## Implication for the scanner

The scanner should not interpret absence of exactly `2 * symbol_count`
statuses within five seconds as a provider-documented failure. Before using
status prose as causal coverage evidence, confirm with Massive support whether
one status per topic, the `subscribed to: <topic>` message form, and a delivery
deadline are contractual. A credentialed raw capture can establish observed
behavior for one session, but not a provider guarantee.
