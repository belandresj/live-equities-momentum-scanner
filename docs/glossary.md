# Glossary

**Status:** Approved shared vocabulary.

**Approved:** 2026-08-05

This glossary is a quick reference. Normative details live in
[`product/product-goals.md`](product/product-goals.md),
[`architecture/system-overview.md`](architecture/system-overview.md),
[`architecture/data-time-and-event-contract.md`](architecture/data-time-and-event-contract.md),
and
[`architecture/scanner-state-engine-lifecycle.md`](architecture/scanner-state-engine-lifecycle.md).
If a short definition here appears less specific, the normative specification
controls.

- **Accepted event (canonically accepted market event):** A normalized market
  event that the Scanner State Engine has validated against the active binding,
  source generation or epoch, interval, causal position, and merge rules and
  then applied to canonical state. Transport receipt, decoding, normalization,
  and admission are earlier stages, not canonical acceptance. See
  `DTE-MODEL-02` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Activity:** The aggregate-derived comparison of current recent behavior
  with the symbol's own completed same-session history. It is not a ranking key
  and is distinct from qualification.

- **Admitted input:** A normalized event or control fact for which the bounded
  engine input has accepted ownership. Admission does not guarantee canonical
  acceptance.

- **Aggregate:** A provider-independent summary of market activity for one
  exact one-second interval, containing OHLC, volume, VWAP, and Average Trade
  Size with provenance.

- **Aggregate identity:** `(symbol, window_start)` within one session binding.
  Source, receipt time, and window end are not identity fields.

- **Aggregate mark:** The latest trusted accepted current-session aggregate
  close available at committed watermark `T`, together with its source window
  and age. It is never synthesized from an empty result or another session.

- **Aggregate transport coverage:** The exact live causal and market-time
  region over which the scanner can support its aggregate-ingress claims. It is
  not a guarantee that the provider will never send a later correction.

- **Average Trade Size (ATS):** The aggregate field used as the denominator in
  approved activity-count approximations. Live ATS is the provider's live
  second-aggregate field; historical ATS is `floor(volume / transaction_count)`
  and retains that provenance. Prior comparison found the difference negligible
  for version 1, but tests do not require the two source values to be equal. A
  zero ATS is structurally representable but unavailable for division.

- **Backend ready:** The live backend has a valid session binding, current
  aggregate transport and committed processing, a reconciled publication
  fence, and a coherent current snapshot that the API can serve. It is not a
  process-liveness or T/Q-availability claim; an honestly resolved empty table
  may be ready. See `PG-OBS-03` in
  [`product-goals.md`](product/product-goals.md).

- **Canonical symbol state:** The single authoritative same-session per-symbol
  representation of accepted market data, coverage, qualification, feature
  state, and accounting owned by the Scanner State Engine. There is one such
  state per bound symbol, and no adapter, worker, API handler, checkpoint writer,
  or UI owns a competing copy. See `ARCH-OWN-01` and `ARCH-OWN-02` in the
  [`system overview`](architecture/system-overview.md).

- **Canonical values:** Provider-independent economic or market fields, as
  distinct from delivery evidence such as source, receipt time, and request
  position.

- **Catch-up boundary (`R`):** The whole-second handoff between historical
  hydration and the current live aggregate tail, calculated from the current
  aggregate-subscription acknowledgement according to the data/time contract.

- **Causal position (live ingress position):** The deterministic source-local
  order of a live item: `(connection_epoch, frame_sequence, array_index)`. It
  supports acknowledgement, fencing, and precedence; it is not event time or
  receipt time. Other source kinds have their own source positions. See
  `DTE-EVENT-02` through `DTE-EVENT-04` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Checkpoint:** A persisted, validated projection of canonical aggregate and
  restart-required derived state exactly as of one coherent committed boundary
  `T0`. Creation/write time is metadata, and ephemeral connection/T/Q coverage
  does not survive restart. See `DTE-CHECKPOINT-*` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Committed watermark (`T`):** The one nondecreasing aggregate market-time
  boundary through which canonical processing and known coverage support the
  claims in a published evaluation. It is the sole ranking and
  aggregate-feature clock; recovery, checkpoint, T/Q, receipt, and generated
  times cannot replace it. See `DTE-CLOCK-05` and `DTE-COMMIT-*` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Completed empty:** A terminal successful hydration outcome proving that a
  complete provider result contained no rows for one exact requested interval.
  It is not failed work and does not by itself prove whole-session no-print.

- **Connection epoch:** A positive process-local identifier that increases for
  each live connection capable of delivering data. Data and acknowledgements
  from an old epoch cannot extend current coverage. See `DTE-EVENT-02` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Correction horizon (`H`):** The bounded recent aggregate interval in which
  an accepted later observation may revise an existing aggregate identity and
  recompute dependent state.

- **Coverage:** The exact market interval or causal boundary over which an
  input or derived measurement is trustworthy. Coverage is tracked separately
  for aggregates, historical ranges, trades, quotes, and individual fields;
  row presence or subscription intent is not coverage. See the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Decoded record:** A source payload that has been parsed. It has not
  necessarily passed normalization or engine acceptance.

- **Degraded bootstrap (`degraded_bootstrap`):** The product-approved,
  explicitly partial raw-Day-% projection available while bootstrap-origin
  completeness remains unresolved and all safeguards in the product contract
  hold. It is not the qualified table and cannot promote T/Q subscriptions. It
  is a ranking status, not a top-level lifecycle state. See `PG-RANK-05` in
  [`product-goals.md`](product/product-goals.md).

- **Delivery evidence:** Source, epoch or generation, causal position, receipt
  time, request interval, and diagnostics describing how a fact reached the
  scanner. Delivery evidence does not change aggregate identity.

- **Effective event time:** The timestamp used for feature-window membership:
  the aggregate interval for aggregates, valid participant time with SIP
  fallback for trades, and SIP time for quotes.

- **Engine sequence:** The monotonically increasing order assigned when the
  Scanner State Engine consumes an admitted input. It defines mutation order,
  not market time.

- **Engine time:** Time supplied by the injected live or simulated clock. It
  drives watermark targets and publication without allowing product code to
  read wall time directly.

- **Event time:** Provider-originated time describing when a market event
  occurred. It determines session and feature-window membership, not mutation
  order. See `DTE-CLOCK-02` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Exact duplicate:** A later aggregate observation with the same aggregate
  identity and equal canonical values. It is counted but does not create a
  second bar.

- **Field availability:** The independent status of one published value:
  unavailable, warming, current, stale, or invalid. Row presence does not make
  every field current.

- **Field currentness:** The field-level claim that one measurement has the
  complete trustworthy coverage required for its own window at the published
  boundary. `current` is distinct from `warming`, `stale`, `unavailable`, and
  `invalid`, and is independent of process liveness, backend readiness, and
  aggregate-ranking currentness. See `PG-AVAIL-01` and `PG-OBS-03` in
  [`product-goals.md`](product/product-goals.md).

- **Fence:** A captured causal boundary through which admitted live input must
  be consumed or explicitly rejected before a dependent coverage or
  publication claim is made.

- **Fenced result:** A terminal result that no longer belongs to the active
  session binding or hydration generation. It is observable but cannot mutate
  canonical state or prove empty coverage.

- **Generated time (`generated_at`):** Engine time when an immutable snapshot
  is created. It is not the ranking watermark and makes no completeness claim
  through that instant.

- **Global suppression versus symbol-local failure:** Global suppression is a
  top-level `suppressed` lifecycle state used only when binding, clock, ingress,
  canonical, accounting, or unrecovered aggregate ambiguity prevents a
  trustworthy current ranking claim. A symbol-local failure changes only that
  symbol or dependent fields and their completeness/accounting consequence.
  See `LIFE-SUPPRESS-*` in the
  [`lifecycle contract`](architecture/scanner-state-engine-lifecycle.md).

- **Hydration:** Bounded historical aggregate loading for fresh bootstrap,
  checkpoint catch-up, or recovery of a known gap. Workers return normalized
  values and one terminal result per planned symbol; only the engine reconciles
  those facts into canonical state. See `DTE-RECOVERY-*` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Hydration or recovery generation:** A positive identifier for one logically
  new hydration or recovery operation. Stale generations are fenced from
  canonical state. See `DTE-EVENT-03` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Hydration interval:** The half-open aggregate window-start interval
  `[start,end)` named by a hydration request.

- **Terminal hydration outcome (terminal symbol outcome):** Exactly one of
  `completed_value`, `completed_empty`, `failed`, `canceled`, or `fenced` for
  every planned symbol request. It accounts for work completion; its canonical
  coverage consequence is evaluated separately. See `DTE-HYDRATE-01` and
  `DTE-HYDRATE-02` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Ingress fence:** The greatest admitted live causal position that the engine
  must reconcile before making the associated state claim. An empty queue is
  not by itself an ingress fence.

- **Live tail:** Accepted live aggregate observations beginning at the catch-up
  boundary and reconciled with historical or checkpoint state.

- **Logical delivery time:** The receipt-time equivalent encoded in a replay
  artifact. It orders replay delivery but does not claim historical live
  latency.

- **Market-time window:** A half-open interval `[start,end)` whose membership is
  determined by effective event time. The end instant is excluded.

- **`no_print_through(T)`:** Complete evidence that a symbol with no existing
  same-session mark produced no accepted aggregate across successfully
  reconciled coverage from session start through `T`. It is a resolved,
  currently unrankable symbol state, not a price and not unfinished work. A
  later accepted aggregate returns it to ordinary mark evaluation. See
  `DTE-RECOVERY-04` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Normalized event:** A provider-independent, structurally valid aggregate,
  trade, quote, connection/control, hydration, replay, or timer fact. It still
  requires engine acceptance.

- **Receipt time (`received_at`):** When the live I/O boundary received a
  payload, or the logical delivery time during replay. It supports diagnostics
  and ordering evidence but does not determine market-window membership. See
  `DTE-CLOCK-03` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Population accounting:** The exact partition of the bound universe into
  mutually exclusive prior-close, mark, no-print, failure/fence, and price
  states required by the product contract. Qualification, Activity, historical
  availability, T/Q, and feature diagnostics are separate overlapping
  dimensions, not additional primary bins.

- **Prior close:** The finite positive adjusted close from the immediately
  preceding completed regular trading session identified by the exchange
  schedule and session binding.

- **Process live:** The backend process is running and can expose status. It
  makes no claim about binding validity, market-data coverage, committed
  currentness, ranking correctness, or individual fields. See `PG-OBS-03` in
  [`product-goals.md`](product/product-goals.md).

- **Immutable snapshot (published snapshot):** A read-only API view of rows,
  independent field availability, readiness, population accounting, and
  bounded diagnostics at one committed watermark and publication identifier.
  API/UI readers cannot mutate or reconstruct engine state. See `ARCH-OWN-03`
  and `ARCH-OWN-04` in the
  [`system overview`](architecture/system-overview.md).

- **Qualification:** The approved same-session aggregate tape gate evaluated
  before ranking. Its first finalized passing proof latches for the session;
  mutable proof remains correction-aware. See `PG-RANK-03` in
  [`product-goals.md`](product/product-goals.md).

- **`qualified_current`:** The ranking status in which complete-population
  evidence at current committed `T` supports the exact qualification filter,
  Day-%/symbol ordering, and top-20 truncation. A resolved zero-row result may
  be `qualified_current`; `degraded_bootstrap` cannot. Only current qualified
  rows may create desired T/Q membership. See `PG-OBS-03` in
  [`product-goals.md`](product/product-goals.md) and `LIFE-PUBLISH-02` in the
  [`lifecycle contract`](architecture/scanner-state-engine-lifecycle.md).

- **Rankable:** A symbol has a valid bound prior close and a trusted accepted
  current-session mark satisfying core mark, price, age, and coverage rules.
  Rankable does not mean qualified, selected, or predictive. See `PG-RANK-01`
  and `PG-RANK-02` in [`product-goals.md`](product/product-goals.md).

- **Ranking current:** The published evidence supports its explicitly named
  ranking status at the current committed watermark. `qualified_current` has
  complete-population evidence and is exact; `degraded_bootstrap` can be current
  only as a disclosed partial raw-Day-% view. Ranking currentness is distinct from
  process liveness, backend readiness, and field currentness. See `PG-OBS-03` in
  [`product-goals.md`](product/product-goals.md).

- **Reconciliation:** Deterministically merging current historical results,
  checkpoint state, and accepted live-tail observations by identity,
  precedence, generation, and ingress fence before making a coverage claim.

- **Replay artifact:** A versioned ordered file of normalized aggregate records
  with explicit provenance, logical delivery time, and immutable record
  ordinal. Version 1 does not include T/Q replay.

- **Replay time:** Engine time supplied by a simulated clock. Playback speed
  must not change replay time, event order, qualification, or canonical output.
  See `DTE-REPLAY-*` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Revision:** A later aggregate observation with the same aggregate identity
  and different canonical values. It may replace the bar only under the
  correction and source-precedence rules.

- **Scanner State Engine:** The sole ordered owner of scanner lifecycle,
  canonical mutation, committed time, evaluation, T/Q intent, publication, and
  coherent checkpoint projection. Adapters and workers return facts; snapshot,
  API, checkpoint, and UI consumers receive immutable views. See `ARCH-OWN-*` in
  the [`system overview`](architecture/system-overview.md).

- **Session binding:** Immutable identity joining the trading date, 04:00–20:00
  New York session bounds, universe, required prior session, and prior-close
  policy. Same-session state cannot cross bindings. See `DTE-SESSION-02` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Session bounds (`S`, `E`):** The UTC instants corresponding to 04:00 and
  20:00 `America/New_York` for the bound trading date. The scanner session is
  `[S,E)`.

- **Snapshot publication identifier:** A value independent of market time that
  distinguishes corrections or availability changes producing multiple
  snapshots at the same `T`. Its exact representation belongs to the API
  specification.

- **Source position:** Deterministic source-local delivery position: live
  causal tuple, hydration request/result ordinal, replay record ordinal, or
  system sequence.

- **Spread:** The quote-derived 10-second time-weighted median of valid NBBO
  spread states defined by the product and focused feature contract. It has
  quote coverage independent of aggregate ranking.

- **T/Q:** Selected-symbol trade (`T`) and quote (`Q`) data used for Tape Rate,
  Spread, and future trade/quote-derived features. T/Q normally covers all
  displayed top-20 symbols but degrades before aggregate processing.

- **T/Q coverage:** Per-symbol, per-channel causal coverage beginning strictly
  after a current-epoch subscription acknowledgement and ending on an
  unsubscribe boundary, pressure-shedding rejection, connection loss, epoch
  change, or session end.

- **T/Q lifecycle states:** Independent selected-symbol states, subordinate to
  aggregate ranking:

  - **desired** means the engine wants T/Q for a current
    `qualified_current` displayed symbol;
  - **subscribed** means the provider acknowledged current-epoch membership,
    but does not by itself prove a complete feature window;
  - **covered** means a continuous per-channel causal interval exists after the
    acknowledgement boundary;
  - **warming** means covered input is accumulating but the feature lacks its
    required history;
  - **degraded** means pressure or channel failure has reduced processing or
    subscribed coverage before aggregate correctness is risked; and
  - **unavailable** means the feature lacks trustworthy required coverage or
    valid measurement state.

  These states do not gate aggregate `T`, ranking, or backend readiness. See
  `DTE-TQ-*` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md) and
  `LIFE-TQ-*` in the
  [`lifecycle contract`](architecture/scanner-state-engine-lifecycle.md).

- **Tape Rate:** A trade-derived trailing 5-second and 1-second measurement
  based on qualifying trades and effective trade event time. It is not an
  aggregate qualification input.

- **Trading date:** The exchange-local `America/New_York` calendar date for the
  active scanner session, not a UTC date obtained by timestamp truncation.

- **Unknown due to failure or fence:** A resolved accounting category meaning
  the scanner lacks trustworthy evidence because required work failed or was
  fenced. It must not be represented as no-print or valid empty.

- **Wall time:** Operating-system time. Product logic accesses it only through
  the injected engine clock; monotonic wall-duration may separately govern I/O
  deadlines. Wall time and operational duration do not determine market-window
  membership. See `DTE-CLOCK-04` in the
  [`data/time contract`](architecture/data-time-and-event-contract.md).

- **Warming:** A field has valid coverage underway but not yet enough required
  history to calculate the contracted value.

- **Watermark target:** The bounded candidate committed time calculated from
  engine time and the approved aggregate evaluation delay. It becomes `T` only
  after the engine’s fence, coverage, and accounting conditions pass.
