# Live backend replacement architecture

**Status:** Owner-approved replacement architecture, 2026-08-23; owner-revised
E2 to exactly one 10-minute deterministic acceptance run on 2026-08-23 and
restored bounded parallel live hydration on 2026-08-24; owner-revised on
2026-08-25 to defer E2 as non-gating and permit E3 next under exact provider
authorization.

**Baseline:** `0d043c1 stabilize live evaluation and local continuity` on
`codex/live-backend-replacement`.

**Scope:** Replace the private/local scanner's live ingress handoff, canonical
state representation, evaluation path, and selected-row T/Q state with the
simplest bounded design that preserves the approved product. Retain the proven
reference, hydration-policy, connection-recovery, snapshot API v2, dashboard,
and private-launcher outcomes.

**Companion delivery authority:**
[`live-backend-replacement/delivery-program.md`](live-backend-replacement/delivery-program.md).

This document is the architecture entry point for the replacement scope
described in Section 2. It does not change a product formula, aggregate market-
data meaning, provider credential authority, or public operating claim.

## 1. Decision and evidence

The replacement is a constrained live-core rewrite, not a clean-room backend
rewrite.

The current design's strongest decisions remain correct: one authoritative
state owner, one canonical aggregate path, causally fenced hydration, bounded
serialized reconnects, T/Q independence, immutable publication, a versioned
read-only API, and a presentation-only UI.

The current implementation nevertheless retains unnecessary work and state:

- the mature live evaluator scans the full population and clones qualification
  state every second;
- superseded HOD, rolling-range, transaction/range-expansion Activity, and old
  Tape state still participate in mutation or evaluation;
- a provider frame is decoded through two JSON passes and several event
  representations;
- a raw-frame queue feeds a second engine queue even though one owner consumes
  both synchronously; and
- T/Q duplicate state is materially broader than its five-second product
  output.

Durable 2026-08-20 diagnostics distinguish this cost from a transport-capacity
failure. The latest connection incident admitted and dispositioned all
1,127,006 frames, rejected none, peaked at 1,723 of 32,768 frame slots and
659,669 of 134,217,728 queued bytes, then terminated only after inbound frames
stopped and the heartbeat deadline observed no progress. Separate 120-cycle
records measured approximately 333-382 ms median evaluation cycles, up to
1.11-2.10 seconds maximum, and roughly 1.28-2.25 GB heap in use. The replacement
therefore corrects state/evaluation/allocation cost first and preserves the
accepted inbound-aware heartbeat behavior.

## 2. Authority and supersession

[`product/product-goals.md`](product/product-goals.md) remains the highest
authority and is not revised by this architecture.

Approval of this document supersedes lower-level architecture and component
delivery decisions only where they require or preserve:

- the current raw-frame-to-adapter-to-engine double handoff;
- current pointer-heavy or duplicated live state representations;
- full-universe evaluation of display-only fields;
- active implementation of removed product features;
- replay/checkpoint state in the supported live core; or
- historical component boundaries that prevent the replacement topology.

The approved document effect is:

| Existing authority | Replacement effect |
| --- | --- |
| `architecture/system-overview.md` | Superseded for the supported live-backend topology, ownership handoffs, queues, state shape, evaluation, and runtime resource boundary |
| `architecture/data-time-and-event-contract.md` | Preserved for the live session/event/merge/watermark semantics named below until each is routed into an accepted focused replacement spec; replay/checkpoint sections create no replacement requirement |
| `architecture/scanner-state-engine-lifecycle.md` | Preserved for compatible fresh/hydrating/live/recovering/currentness/termination meaning until routed into focused specs; replay/checkpoint and conflicting lower-level transition machinery are superseded |
| `live-feature-mvp-program.md` | Historical accepted feature/UI/stability evidence; superseded as active implementation authority by the approved replacement delivery program |
| Numbered component and focused correction contracts | Reusable evidence only where compatible; no old representation, proof allocation, or delivery gate controls the replacement |

The following existing meanings remain fixed unless a later owner-approved
product change says otherwise:

| Meaning | Controlling requirements |
| --- | --- |
| Eligible universe, adjusted prior close, and optional Float | `PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-REFERENCE-02` |
| Trusted mark, qualification, exact Day-%/symbol ordering, and top 20 | `PG-RANK-01` through `PG-RANK-05` |
| Volume, From Open, Day Range, Activity 30s, Move 30s, Tape 5s, Spread, and retained aggregate evidence | `PG-FEATURE-01` through `PG-FEATURE-07` |
| Independent field and T/Q availability | `PG-AVAIL-01` through `PG-AVAIL-03`, `PG-TAQ-01` through `PG-TAQ-03` |
| Fresh restart, live recovery, and no frozen ordinary evaluation | `PG-OPS-01`, `PG-OPS-02` |
| Backend/API/UI ownership | `PG-UI-01`, `PG-UI-02` |
| Population/work accounting and trader-visible trust | `PG-OBS-01` through `PG-OBS-03` |

Existing session bounds, half-open windows, normalized aggregate identity,
16-minute correction horizon, REST/live precedence, successful-empty meaning,
committed-watermark meaning, causal fencing, and currentness rules remain
compatible evidence from the approved data/time/event and lifecycle contracts.
The focused replacement specifications must route each retained semantic to one
new authoritative home before the corresponding implementation slice begins.

Replay remains unsupported and checkpoint persistence remains disabled during
the replacement. Neither may add state, proof, or compatibility gates to the
supported fresh-start live path.

## 3. Architectural outcome

The supported scanner remains one backend process plus the independently
runnable dashboard:

```text
schedule + reference data
  -> immutable SessionBinding

Massive WebSocket reader + single-pass decoder
  -> one bounded decoded-batch ingress ring
  -> ScannerStateEngine live loop

one bounded REST hydration pool (1, 2, 4, or 8 workers; default 8)
  -> bounded hydration results
  -> the same ScannerStateEngine and canonical aggregate merge

ScannerStateEngine
  -> compact canonical SymbolState[]
  -> incremental aggregate/qualification mutation
  -> one-second causally fenced selection
  -> selected-row enrichment
  -> one immutable API-v2-compatible snapshot

snapshot API v2
  -> existing dashboard
```

There is no database, service split, generic event bus, runtime plugin system,
per-symbol worker hierarchy, browser-owned market logic, or second mutable
scanner state.

## 4. Ownership model

### `LBR-ARCH-01` — one mutable market-state owner

One `ScannerStateEngine` live loop exclusively owns:

- the installed session binding and top-level live lifecycle;
- accepted connection epoch and hydration/recovery generation;
- canonical aggregate and selected-row T/Q state;
- coverage, correction, qualification, availability, and accounting;
- committed aggregate watermark and exact top-20 selection;
- T/Q desired membership and trust state; and
- immutable snapshot creation.

No adapter, worker, timer, API handler, diagnostic, or UI path may mutate those
facts or publish a competing interpretation.

### `LBR-ARCH-02` — concurrency only at blocking boundaries

The justified concurrent work is:

- one WebSocket reader/single-pass decoder;
- one heartbeat operation while a socket is active;
- one bounded REST hydration pool of 1, 2, 4, or 8 fact-producing workers
  while historical work is active;
- loopback HTTP request handling over immutable snapshots; and
- private launcher/dashboard process supervision.

These components return bounded facts or immutable reads. They do not receive
writable symbol state. Additional goroutines, queues, samplers, or workers
require measured blocking or capacity evidence in the owning focused spec.

### `LBR-ARCH-03` — one live ingress ordering buffer

Decoded provider batches and causal fence markers share one bounded FIFO owned
by the live ingress boundary. The adapter owns admission and raw causal
position; the engine consumes batches serially. The FIFO is a bounded handoff,
not a market-state owner. A batch is the smallest handoff across that boundary;
its events retain connection epoch, frame sequence, array order, and receipt
time.

There is no second general-purpose engine FIFO for those same live facts.
Engine-owned timers and bounded hydration results may use direct owner-local
control or a small dedicated result channel, but cannot become another market
event bus or ordering authority.

Queue saturation never silently loses possible aggregate/control work. Safe
T/Q shedding occurs first. If an aggregate/control-bearing batch cannot be
admitted, the epoch closes and exact gap recovery is required.

### `LBR-ARCH-04` — one immutable publication boundary

The engine replaces one immutable snapshot cell after a complete evaluation.
API capture performs one atomic snapshot read and no market-state lock,
`runtime.ReadMemStats`, checkpoint-state read, or cross-snapshot join. Slow,
canceled, or unavailable HTTP/UI work cannot delay canonical mutation.

## 5. Canonical state

### `LBR-ARCH-05` — compact prefix and correction tail

Each bound symbol has one logical state:

```text
SymbolState
  static reference
    symbol | adjusted prior close | Float fact

  sealed session prefix
    covered-through | cumulative volume | first open
    session high/low | latest sealed mark | finalized qualification

  mutable correction tail
    sparse canonical aggregates within the 16-minute horizon
    compact present/no-print/invalid/unknown coverage
    latest eligible mark | provisional qualification evidence

  selected T/Q state, only while needed
    current generation coverage | qualifying trade window
    bounded duplicate evidence | latest and latest-valid quote
```

The structure may differ in Go, but an accepted aggregate cannot be retained as
several independently mutable copies for canonical, price-range, Activity,
Volume, replay, and checkpoint purposes. Derived indexes are permitted only
when bounded, rebuildable from canonical state, and proven cheaper than the
direct alternative.

When an aggregate passes the correction horizon, its required effect folds
once into the sealed prefix and the raw mutable record is discarded. A rare
correction may perform bounded work for that symbol; routine one-second
evaluation must not clone or rescan per-symbol session history.

### `LBR-ARCH-06` — hydration enters the same state

The existing hydration policy is retained:

1. establish and acknowledge persistent aggregate coverage;
2. capture the exact catch-up boundary `R`;
3. hydrate `[S,R)` while the live tail remains active;
4. give every planned symbol one terminal value/empty/failed/canceled/fenced
   outcome;
5. merge historical and live aggregates by the same canonical identity and
   precedence rules; and
6. return current only after the live ingress fence behind already-read work is
   applied and the ordinary evaluator succeeds.

Successful empty remains explicit no-print evidence. The hydrator owns
requests, pagination, retries, response bounds, and normalization; it never
owns symbol state, ranking, readiness, or recovery completion.

## 6. Mutation, evaluation, and publication

### `LBR-ARCH-07` — accepted facts update only affected state

An accepted aggregate synchronously updates its symbol's canonical tail,
prefix deltas, mark, coverage, and affected qualification windows. Duplicate,
revision, withdrawal, conflict, invalid, and late evidence remains explicitly
classified and accounted.

Qualification is incremental and symbol-local. A correction may revisit only
the bounded proof windows it can affect. Routine one-second work does not clone
qualification maps across the universe.

### `LBR-ARCH-08` — two-phase one-second evaluation

At one causally supported boundary per ordinary second:

1. **Selection phase:** scan compact scalar state for the complete population,
   apply approved qualification, compute Day %, maintain primary accounting,
   and retain exact Day-%/symbol top 20.
2. **Enrichment phase:** compute or read Volume, From Open, Day Range, Activity
   30s, Move 30s, Tape 5s, Spread, Float, and field states only for the selected
   rows, while separately maintaining required full-population feature
   accounting without evaluating discarded product features.

Selection cannot create or warm aggregate evidence. A symbol outside the table
for more than 330 seconds must enter with the same aggregate-derived values it
would have had if continuously selected. T/Q alone warms after selection.

The timer advances quiet-window, age, lifecycle, T/Q, and session-end work. A
completed ingress fence and its immediately following timer cannot repeat the
same full-population selection for the same canonical prefix.

### `LBR-ARCH-09` — publication cadence and immediate trust changes

Ordinary aggregate and T/Q mutations may coalesce into the next one-second
combined snapshot. A trust transition that closes currentness, T/Q coverage,
or field validity publishes immediately when required to prevent false-current
output. Every snapshot contains one publication ID, one committed watermark,
one ordered row set, and internally coherent aggregate/TQ/status/accounting
views.

## 7. Connection and data-quality policy

### `LBR-ARCH-10` — retain bounded connection recovery

Use one Massive Stocks WebSocket. Only one attempt may dial, handshake, run, or
clean up at a time. The process-start dial is immediate and is not a recovery
ordinal. After a failed process-start dial or any lost epoch, recovery attempts
1 through 5 wait exactly 1, 2, 4, 8, and 16 seconds respectively. Failure or
loss of recovery attempt 5 reaches stable noncurrent exhaustion with no recovery
attempt 6. Only a successfully reconciled hydration/recovery fence resets the
recovery budget.

A failed heartbeat with later supported inbound frame progress is diagnostic
and nonterminal. A read failure, independent heartbeat transport failure, or
five-second heartbeat deadline without inbound progress closes currentness,
T/Q coverage, and the epoch, then enters exact recovery.

### `LBR-ARCH-11` — common failure containment

- Exact aggregate duplicates are no-ops with accounting.
- Ordinary out-of-order aggregate changes inside the correction horizon use
  the existing causal/REST-live precedence rules.
- Recognized malformed aggregate evidence with usable identity is contained to
  that symbol/identity and cannot fabricate a value.
- Malformed T/Q evidence is T/Q-local when family and symbol are trustworthy.
- An unclassifiable frame or element that could hide aggregate/control work is
  aggregate-ingress ambiguity and requires recovery rather than silent drop.
- Unsupported but clearly identified event families and unknown additive JSON
  members are bounded drops.
- A T/Q state bound closes affected T/Q fields and membership without changing
  aggregate ranking or backend readiness.
- An unrecoverable global binding, clock, coverage, or canonical ambiguity is
  explicit noncurrent/suppressed state, never partial currentness.

Focused specifications own exact validation shapes and reasons. They may
simplify lower-level machinery but may not weaken these failure domains.

## 8. Resource architecture

### `LBR-ARCH-12` — bounded private/local operation

If E2 is resumed, measure the replacement against the exact mature 5,694-symbol
manifest with representative selected-row T/Q and dashboard polling. E2 is a
deferred deterministic capacity characterization, not a prerequisite for the
authorized E3 real-provider observation or live-stability confirmation. The
values below remain design targets and diagnostic thresholds, not independent
completion gates.

| Metric | Initial design target |
| --- | ---: |
| Average backend CPU over the exact 10-minute deterministic acceptance run | `<= 0.75` core |
| One-second-sampled CPU p95 | `<= 1.5` cores |
| Heap in use after hydration | `<= 512 MiB` |
| Heap in use during hydration | `<= 768 MiB` |
| Backend RSS | `<= 1 GiB` |
| Steady allocation rate | `<= 5 MiB/s` |
| Steady backend goroutines | `<= 12` |
| Hydrating backend goroutines | `<= 20` |
| Selection/enrichment cycle p99 / maximum | `< 100 ms / < 250 ms` |
| API snapshot capture p99 / maximum | `< 25 ms / < 100 ms` |
| Additional processing beyond the fixed four-second semantic delay, p99 | `< 500 ms` |
| Aggregate/control loss or capacity rejection | `0` |

Hard E2 resource acceptance, when resumed, requires:

- heap, RSS, retained state, queues, and goroutines reach a bounded plateau;
- the characterized ordinary feed and accepted stress composition show no
  sustained queue growth or silent aggregate/control loss;
- normal operation has no recurring watermark-stale/readiness flapping;
- snapshot/API work remains usable at the one-second dashboard cadence.

Missing a numeric design target is a recorded deviation, not automatic
rejection. It triggers one bounded response: verify the measurement, profile
once, make at most one focused correction for demonstrated unnecessary work,
and rerun once. If hard acceptance then passes, record the measured result and
continue. The orchestrator may not repeat optimization solely to reach an
aspirational number. Only correctness/loss, unbounded growth, sustained
backlog/readiness failure, or unusable API/dashboard behavior blocks
completion.

The delivery program may revise queue slots, byte division, fixture pacing,
measurement mechanics, or the reported target deviation when evidence
requires it. It may not loosen product correctness, readiness, market-time, or
bounded-plateau requirements to pass a benchmark.

`LBR-E2`, if separately reactivated, executes exactly one 10-minute
deterministic characterization run. No fallback, diagnostic, host-coexistence,
or repeat run is authorized. Deferral does not weaken that future proof; it
changes only program ordering and completion status. `LBR-E3` may proceed first
under exact provider authorization and can establish bounded real-provider
operation, but it cannot establish the synthetic 300-frames/s capacity claim.

## 9. Retain, replace, and remove

| Area | Decision |
| --- | --- |
| Schedule, universe, prior close, Float, caches, and binding | Retain behavior and adapt interfaces only when required |
| REST hydration policy, bounded `1|2|4|8` worker interface with default 8, terminal accounting, and ingress fence | Retain |
| One socket, handshake facts, inbound-aware heartbeat, serialized retry, and exhaustion | Retain |
| Aggregate identity, correction horizon, duplicate/revision/withdrawal, and REST/live precedence | Retain |
| Snapshot API v2, loopback security, dashboard, and launcher continuity | Retain |
| Canonical symbol representation and qualification/evaluation implementation | Replace |
| Double-pass live decoder | Replace with one bounded pass |
| Raw-frame queue plus second engine FIFO and intermediate event envelopes | Replace with one decoded-batch handoff |
| Selected-row T/Q retention and membership mechanics | Replace while preserving product meanings |
| HOD drawdown, rolling 30/60-minute ranges, old Activity composite, and old Tape burst | Remove from active state, evaluation, API mapping, tests, and contracts |
| Replay and checkpoint participation in the live core | Remove; final source/tool disposition is an owner decision routed to integration |
| Historical correction ledgers | Preserve as evidence but remove from the active replacement reading path |

### `LBR-ARCH-13` — deletion is part of completion

The replacement is not complete while the supported live binary still
constructs or mutates the superseded state, uses the old handoff as a fallback,
or selects between two authoritative engines at runtime. Test-only differential
oracles may temporarily compare implementations, but production executes one
state path.

## 10. Active document map

This parent owns architecture outcome, cross-cutting invariants, the final
topology, document routing, and approval boundary. It owns no mutable delivery
ledger.

| Document | Sole responsibility | Current state |
| --- | --- | --- |
| [`delivery-program.md`](live-backend-replacement/delivery-program.md) | Capability order, slices, proof/review gates, Git/orchestration policy, and sole delivery ledger | Approved current delivery authority |
| [`canonical-state-and-hydration.md`](live-backend-replacement/canonical-state-and-hydration.md) | Aggregate prefix/tail, merge, coverage, correction, hydration, and recovery installation | Owner-approved focused contract |
| [`evaluation-and-publication.md`](live-backend-replacement/evaluation-and-publication.md) | Incremental qualification, two-phase selection/enrichment, accounting, and immutable publication | Owner-approved focused contract |
| [`tq-state.md`](live-backend-replacement/tq-state.md) | Selected membership, T/Q coverage, Tape/Spread state, deduplication, and degradation | Owner-approved focused contract |
| [`live-ingress.md`](live-backend-replacement/live-ingress.md) | Single-pass Massive decoding, decoded-batch FIFO, epochs, heartbeat, commands, and transport terminals | Owner-approved focused contract |
| [`integration-removal-and-acceptance.md`](live-backend-replacement/integration-removal-and-acceptance.md) | Production cutover, deletion, API/UI compatibility, capacity, stability, and live confirmation | Owner-approved focused contract |

Each subordinate specification names this parent, owns its allocated
requirements/proofs/slices exactly once, and copies no status. Localized work
reads this parent, the delivery program, and only its routed focused spec and
dependencies. Final integrated review reads the complete set.

## 11. Deliberately delegated decisions

Focused specs and the delivery program may select:

- exact Go packages, private types, compact encodings, and algorithms;
- queue slot count and batch shape within the resource/byte budgets;
- direct versus rebuildable derived indexes;
- hydration result chunking and owner-local control mechanics;
- fixed-cardinality diagnostic fields;
- proof fixtures and benchmark measurement mechanisms; and
- sequential slice boundaries when correction evidence requires revision.

They may not select another state owner, watermark, ranking path, product
formula, browser calculation, aggregate dependence on T/Q, silent aggregate
loss, unsupported readiness, or replay/checkpoint gate.

## 12. Approved and remaining owner decisions

1. The live core excludes replay and checkpoint state. Final source deletion
   versus retained unsupported tooling remains due before integration cleanup.
2. T/Q uses an owner-approved 30-second late/duplicate policy. A current-epoch
   trade or quote more than 30 seconds older than committed `T` is counted and
   ignored; trade duplicate evidence is retained for 30 seconds after receipt.
   Exact clock/tie/boundary mechanics belong to `tq-state.md`. This changes no
   aggregate correction horizon, ranking input, or backend-readiness rule.
3. The numeric resource values are design targets governed by the bounded
   response above. Hard acceptance is behavioral and plateau-based.
4. The supported live hydrator restores the previously accepted bounded
   `1|2|4|8` worker surface and defaults to eight. Workers own only blocking
   REST acquisition and immutable bounded results; the engine retains one
   generation/request ledger, one canonical mutation path, and the sole fence/
   currentness decision. The 2026-08-24 live observation showed that the
   one-worker restriction completed only 31.6% of a 5,566-symbol late-start
   hydration in approximately five minutes while the WebSocket path itself
   remained healthy. The historical incidents motivating this replacement
   occurred after successful multiworker hydration and do not justify making
   acquisition serial.
5. The owner separately authorizes or executes any market-hours run. E2 has
   exactly one 10-minute deterministic run and no host-coexistence repeat.
