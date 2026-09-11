# Swing scanner and institutional-footprint research

> Research notes retained for reference. Proposed data sources and scanner extensions are not implemented capabilities of the live scanner.


**Researched:** 2026-08-23

## Conclusion

Pivoting the product toward swing discovery is the stronger decision if the
target job and users are DeepVue-style swing traders. The attached image is a
**market-breadth dashboard**, not a stock-candidate scanner: it measures the
market regime through participation and expansion/contraction counts. A useful
swing product needs both that regime layer and a separate candidate-ranking
layer.

Trade/quote data can add a technically credible experimental overlay, but it
cannot support a literal **institutional-buying detector**. The defensible claim
is narrower: estimate persistent aggressive buy pressure, passive bid-side
liquidity resilience, and the subsequent price response. Public data does not
identify the beneficial owner or parent order; institutional execution is
normally sliced across child orders and venues; NBBO size aggregates multiple
participants; hidden/reserve liquidity is omitted; and off-exchange interest is
generally observed only after execution.

A static “large bid means an institution is buying” feature would be naive and
could hurt an interview. A feature explicitly named and tested as **Flow
Persistence** or **Liquidity Resilience**, with data-quality qualifiers and no
identity claim, demonstrates the opposite: knowledge of market structure,
measurement error, real-time knowability, and out-of-sample validation.

## Why the swing pivot fits the product target

- DeepVue's own [screener description](https://deepvue.com/screener/) centers
  swing-relevant discovery: relative strength, bases and volatility
  contraction, earnings and fundamentals, price/volume context, and named
  trader presets. It distinguishes market-wide discovery from a single
  intraday trigger.
- DeepVue describes [breadth](https://deepvue.com/technical-analysis/market-breadth-home/)
  as a regime view built from advancers/decliners, new highs/lows, trend stages,
  sectors, and themes. That is the job performed by the attached table. The
  table's stocks-up/down thresholds, rolling ratios, percentage-expansion
  counts, 50-day participation, and universe size are regime variables; they do
  not by themselves rank entries, define stops, or establish trade expectancy.
- The current data constraint is compatible with this direction. Massive
  [Stocks Advanced](https://massive.com/pricing?product=stocks) includes
  real-time data, 20+ years of history, aggregates, trades, quotes, snapshots,
  financials, and ratios. Those inputs are enough to build breadth history,
  point-in-time technical/fundamental candidate features, and an experimental
  T/Q overlay without buying a depth-of-book feed.

DeepVue's first-party pages establish product positioning and available
workflows, not proof that a screen has predictive or executable edge. Any
remake should treat Ariel's public breadth layout as a workflow reference, not
as validated signal authority.

## What institutions hide, and what still leaks into the market

The concern about execution algorithms is correct. The SEC reports that manual
handling of institutional orders is increasingly rare: a large parent order is
typically divided into many smaller child orders, potentially sent through
multiple brokers. Broker algorithms target VWAP, TWAP, implementation
shortfall, or a percentage of market volume, and can provide or take liquidity
across exchanges, ATSs, dealer platforms, and internal risk books. See the
[SEC Staff Report on Algorithmic Trading, pp. 33-36](https://www.sec.gov/files/Algo_Trading_Report_2020.pdf).

That makes identification difficult, not theoretically impossible. Order
splitting itself can create persistent same-sign flow. An original empirical
study found that sub-hour order-flow persistence was dominated by splitting
rather than herding in its London Stock Exchange sample
([Tóth et al., *Why is order flow so persistent?*](https://arxiv.org/abs/1108.1632)).
Van Kervel and Menkveld observed HFT firms changing behavior around known large
institutional metaorders, evidence that sufficiently sophisticated participants
can sometimes detect ongoing execution
([*High-Frequency Trading around Large Institutional Orders*](https://onlinelibrary.wiley.com/doi/10.1111/jofi.12759)).

The important limitation is horizon. Detectability of an ongoing metaorder and
contemporaneous price impact do not imply positive 5-20-day returns. A study
that inferred daily institutional flow from public TAQ data required a model
benchmarked to quarterly 13F changes, not a simple large-trade rule, and found
short-run reversal as well as longer-run effects
([Campbell, Ramadorai, and Vuolteenaho](https://www.nber.org/papers/w11439.pdf)).
This is evidence that tape inference is a legitimate research problem, not
evidence that any particular footprint score has swing expectancy.

## Why a large bid or “wall” is not an institutional identifier

1. **The public quote is an aggregate.** The SEC explains that the consolidated
   tape does not reveal individual participants even anonymously. A change that
   looks like one participant posting and canceling can instead be several
   parties independently joining and leaving the same price
   ([SEC market-structure speech](https://www.sec.gov/newsroom/speeches-statements/2013-spch061813gebhtm)).
2. **Displayed size is incomplete.** Nasdaq permits a displayed order to be
   replenished from non-displayed reserve size and even allows randomized
   displayed quantities
   ([Nasdaq Equity 4, Rule 4703(h)](https://listingcenter.nasdaq.com/rulebook/nasdaq/rules/Nasdaq%20Equity%204)).
   Repeated size at a price can therefore be one reserve order, several new
   orders, or aggregate venue changes.
3. **Displayed interest can disappear.** A bid is executable while resting but
   can be canceled. The SEC defines spoofing as submitting and canceling orders
   without intent to trade to create a false impression of imbalance; this does
   not make every cancellation manipulative, but it proves that displayed size
   is not equivalent to committed directional intent
   ([SEC Algorithmic Trading Report, pp. 72-74](https://www.sec.gov/files/Algo_Trading_Report_2020.pdf)).
4. **The market is fragmented.** The SEC describes exchanges, ATSs, dealer
   platforms, and other matching systems. FINRA explains that institutions may
   prefer OTC execution for pre-trade anonymity and that OTC prints are
   reported to a Trade Reporting Facility and then the consolidated tape
   ([FINRA OTC equity trading](https://www.finra.org/investors/insights/over-the-counter-equities-trading)).
   The execution is visible after the fact, but the initiating customer's
   identity and pre-trade interest are not.

The same ambiguity applies to “strong buyers.” Every execution has a buyer and
seller. Aggressor classification estimates which side demanded immediacy; it
does not identify whether that side was an institution, market maker, retail
wholesaler, hedge, index rebalance, or arbitrage strategy.

## What Massive Stocks Advanced can and cannot measure

Massive's stock trade stream supplies tick-level price, size, exchange,
conditions, and timestamps
([trades documentation](https://massive.com/docs/websocket/stocks/trades)). Its
quote stream is explicitly the **NBBO**, with best bid/ask prices and sizes
([quotes documentation](https://massive.com/docs/websocket/stocks/quotes)).
Massive also states that its stock data combines SIP-consolidated feeds and
reported off-exchange trades
([Stocks API overview](https://massive.com/docs/rest/stocks)).

This supports consolidated trade/quote research, but not order-level book
reconstruction:

- no order ID, parent-order ID, customer identity, or beneficial-owner identity;
- no depth below the best bid or above the best ask;
- no reliable separation of new, canceled, and replenished constituent orders
  inside an aggregated NBBO-size change;
- no pre-trade view of dark/OTC liquidity; and
- no proof that a trade was institutionally initiated.

The SEC makes the data distinction explicit: full order-book analysis requires
separate proprietary feeds from individual exchanges, including posted orders,
modifications/cancellations, and executions
([SEC MIDAS](https://www.sec.gov/securities-topics/market-structure-analytics/midas-market-information-data-analytics-system)).
Therefore, do not build or advertise a Level-2 “wall detector” under the Massive
Stocks Advanced constraint. Nasdaq/NYSE auction-imbalance data would also be a
different input; Massive lists NYSE Order Imbalances as a separate dataset, not
part of Stocks Advanced.

## Defensible T/Q features for a slower scanner

Keep aggressive demand and passive absorption separate because they are
different behaviors and can oppose each other.

### 1. Estimated aggressive-flow persistence

Infer trade direction from the prevailing NBBO, then compute estimated
buy-initiated minus sell-initiated dollar volume in fixed intervals, normalized
by classified dollar volume and the symbol's time-of-day baseline. Report the
classified-volume fraction; do not silently force midpoint or stale-quote
trades into a side. The foundational Lee-Ready paper explains why direction
must be inferred and why quote timing and inside-spread trades create errors
([Lee and Ready, 1991](https://doi.org/10.1111/j.1540-6261.1991.tb02683.x)).

Useful slower summaries include the fraction of 5-minute intervals with
positive imbalance, longest same-sign run, session-cumulative imbalance, and
agreement across morning/afternoon rather than one large print.

### 2. NBBO bid-resilience proxy

Measure estimated sell-initiated volume executed near the bid while the
bid/midprice remains stable, plus repeated restoration of NBBO bid size at the
same price. Normalize by spread, displayed NBBO size, dollar volume, and the
stock's own intraday seasonality. Call this a **bid-resilience proxy**, not
confirmed iceberg absorption: the Massive feed cannot determine whether the
restoration came from one hidden reserve, multiple participants, or venue
composition changes.

### 3. Price response and retention

Pair flow with what price actually did: contemporaneous midprice response,
close versus session VWAP, closing-range location, and return retention 30-60
minutes, at the close, and next session. Cont, Kukanov, and Stoikov found a
robust relationship between short-interval price changes and best-quote order
flow imbalance, conditional on depth
([*The Price Impact of Order Book Events*](https://papers.ssrn.com/sol3/papers.cfm?abstract_id=1712822)).
Their result is primarily contemporaneous short-horizon price impact, so it
does not validate swing continuation.

### 4. Data-quality and market-context gates

Expose quote staleness, locked/crossed time, unclassified-trade fraction,
special-condition exclusions, spread, dollar liquidity, and off-exchange share.
Treat off-exchange volume as context only; an OTC print has no public directional
or institutional-intent label. Combine the flow variables with ordinary swing
context: relative volume, float turnover, ATR-normalized extension, relative
strength, base position, earnings/catalyst status, and sector/breadth regime.

## Recommended product sequence

1. **Build the swing core first.** Recreate the breadth dashboard as the market
   regime surface, then build a candidate scanner using relative strength,
   distance from highs and bases, liquidity, ATR/ADR contraction, price/volume,
   earnings/sales, and catalyst context.
2. **Preserve T/Q as a research overlay.** Add two clearly named, separately
   displayed measurements: `Estimated Aggressive Flow` and `NBBO Bid
   Resilience`. A composite can come later only if its meaning remains clear.
3. **Test incremental value, not storytelling appeal.** Compare a point-in-time
   swing baseline `B` (price, volume, relative strength, fundamentals, catalyst,
   breadth) with `B + F` (the same model plus flow features). Use walk-forward,
   purged evaluation and report 5/10/20-day ATR-normalized returns, breakout
   follow-through, maximum favorable/adverse excursion, stop/target expectancy,
   and concentration by symbol, date, liquidity, and earnings status.
4. **Respect real-time knowability.** A completed-session flow score is known
   only after the close. An intraday score must use only events received by that
   timestamp and time-of-day normalization. Define an executable entry such as
   next open or a timestamped intraday trigger before computing outcomes.
5. **Retain the overlay only for broad out-of-sample lift.** If `B + F` does not
   improve the trader-facing outcome beyond `B`, the correct result is that
   T/Q added complexity but no validated swing edge.

## Interview framing

The strongest framing is:

> I pivoted from an intraday alert engine to the swing workflow the target users
> actually need: market regime, leadership discovery, and candidate ranking. I
> reused the live trade/quote pipeline for a deliberately narrower experiment.
> It estimates persistent aggressive flow and NBBO liquidity resilience; it does
> not claim to identify institutions. I then tested whether those measurements
> add 5-20-day expectancy beyond price, volume, fundamentals, catalysts, and
> breadth.

That is more credible than cloning a breadth table alone and more
market-structure-literate than marketing big bids or block prints as direct
institutional footprints.

## Limitations

- No cited study validates these exact features on current Massive data, the
  scanner's universe, or the proposed swing horizon.
- Several foundational microstructure studies use older periods or non-U.S.
  venues with participant identifiers unavailable in the current public feed.
- Official SEC/FINRA/exchange sources establish market structure and data
  limitations; they do not establish trading alpha.
- DeepVue first-party material establishes product behavior and positioning,
  not independent evidence of edge.
- The proposal remains descriptive until point-in-time historical inputs,
  trade-condition handling, quote alignment, a baseline, and out-of-sample
  trade-model results are defined and run.
