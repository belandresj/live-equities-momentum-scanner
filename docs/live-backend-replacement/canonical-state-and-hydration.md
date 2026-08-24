# Canonical state and hydration

**Status:** Owner-approved focused replacement specification, 2026-08-23;
owner-revised to restore bounded parallel live hydration on 2026-08-24. The
[delivery program](delivery-program.md) is the sole mutable status ledger.

**Parent:** [Live backend replacement architecture](../live-backend-replacement.md).

## 1. Outcome and authority

This contract gives the live engine one compact aggregate state per bound
symbol and installs fresh or recovery hydration into that same state. It owns
aggregate identity, precedence, correction, coverage consequence, and the
hydration generation/fence consumer. It does not own qualification formulas,
selection, T/Q, provider-frame decoding, API mapping, or runtime cutover.

Allocated parent requirements are `LBR-ARCH-05`, `LBR-ARCH-06`, and
`LBR-ARCH-07`. Cross-cutting sole-owner conformance under `LBR-ARCH-01` is
owned by the integration contract; this contract supplies its canonical-
aggregate implementation obligation. Allocated retained product semantics are
`PG-UNIVERSE-01`, `PG-REFERENCE-01`, `PG-REFERENCE-02`, `PG-RANK-02`,
`PG-FEATURE-07`, `PG-OPS-01`, `PG-OPS-02`, and `PG-OBS-02` from
[`product-goals.md`](../product/product-goals.md). This document routes only
their canonical-state and hydration consequences; it does not restate their
formulas.

The compatible historical semantics routed here are:

- `DTE-SESSION-01`, `DTE-SESSION-02`, `DTE-CLOCK-01`, `DTE-WINDOW-01`,
  `DTE-WINDOW-02`, `DTE-WINDOW-04`, `DTE-EVENT-03`, `DTE-AGG-01` through `DTE-AGG-04`,
  `DTE-HYDRATE-01`, `DTE-HYDRATE-02`, and `DTE-MERGE-01` through
  `DTE-MERGE-05` from the
  [data/time/event contract](../architecture/data-time-and-event-contract.md);
- `DTE-RECOVERY-01` through `DTE-RECOVERY-05` for interval, catch-up, empty,
  and fence meaning, but not the old state representation; and
- `LIFE-MODEL-01` through `LIFE-MODEL-04`, `LIFE-INIT-01`, `LIFE-INIT-02`,
  `LIFE-INIT-04`, `LIFE-INIT-05`, `LIFE-HYDRATE-01` through
  `LIFE-HYDRATE-07`, and `LIFE-RECOVER-02` through `LIFE-RECOVER-06` from the
  [engine lifecycle](../architecture/scanner-state-engine-lifecycle.md), with
  connection-attempt execution allocated to
  [`live-ingress.md`](live-ingress.md).

Replay and checkpoint sections of those documents are not routed here.

## 2. Boundary and dependencies

This is the first replacement capability and has no dependency on another
focused replacement spec. It consumes the retained immutable
`reference.Binding`, normalized aggregates, hydration terminal results, and an
ordered ingress-fence fact. Existing producers may adapt to these inputs until
Capability D replaces the ingress producer; the engine-owned acceptance rules
below do not change.

Downstream contracts consume immutable views only:

| Output | Consumer | Exact meaning |
| --- | --- | --- |
| `SelectionStateView` | [evaluation/publication](evaluation-and-publication.md) | Symbol/reference scalars, trusted mark, coverage class, compact proof inputs, and a canonical revision. No writable aggregate or map. |
| `SelectedAggregateView` | evaluation/publication | Read-only as-of access to current-product aggregate measurements for one selected symbol. |
| hydration/recovery view | evaluation/publication and integration | Active generation, purpose, exact work identity, fence state, and coverage consequences owned by the engine. |
| normalized aggregate and hydration/fence input contracts | [live ingress](live-ingress.md) and existing adapters | Bounded facts whose binding, epoch, generation, position, and interval are validated before mutation. |

The dependencies are acyclic: this contract defines the owner/consumer seam;
evaluation consumes it, T/Q consumes evaluation, ingress implements the final
producer seam, and integration consumes all four.

Only the current repository's
[`S2 canonical aggregate`](../specifications/scanner-state-engine-and-canonical-state/s2-canonical-aggregates.md),
[`REST acquisition`](../specifications/aggregate-rest-hydration-and-recovery/rest-acquisition-and-terminal-outcomes.md),
and [`engine hydration`](../specifications/aggregate-rest-hydration-and-recovery/engine-hydration-reconciliation-and-recovery.md)
contracts and their directly named fixtures are reuse evidence. Their merge,
terminal, fence, bounded `1|2|4|8` worker-pool, cancellation, and joined-result
outcomes are retained; their representations, ledgers, checkpoint/replay paths,
and accepted implementation are not. No predecessor checkout or provider
access is part of this contract.

## 3. Owned state and invariants

The `ScannerStateEngine` live loop is the only mutable owner. Installation of
one immutable session binding creates a fixed-index `SymbolState` array. A
symbol has one logical aggregate representation:

- immutable reference facts from the binding;
- a sealed prefix containing sufficient current-product state through a
  whole-second fold boundary;
- a sparse canonical tail for mutable aggregate identities in the inclusive
  16-minute correction horizon, plus the strictly older predecessor mark when
  needed by a boundary calculation; and
- compact coverage and invalid/conflict evidence sufficient to distinguish
  present, proven absent, unknown/fenced, and locally invalid intervals.

The prefix carries the cumulative session-volume effect, first eligible open,
session extrema, latest sealed eligible mark, fold boundary, and the compact
qualification facts allocated by the evaluation contract. The tail carries
each canonical aggregate once. Any index is bounded, rebuildable from this
state, and nonauthoritative. It may not retain another mutable aggregate copy.

An identity remains `(symbol, window_start)`. Accepted windows are exact
one-second, half-open, session-bound intervals. The tail retains at most the
961 mutable second identities implied by the 16-minute horizon plus one
predecessor mark; sparse absence does not allocate a full aggregate record.
Once an identity is strictly outside the correction horizon, its effect folds
once into the prefix and its mutable record is discarded. No ordinary later
input can reopen it.

The compact coverage evidence retains whether folded presence has independent
sealed-live support. Exact value comparison needed by a historical request may
exist only as bounded reconciliation evidence owned by that exact active
request; it is consumed by the matching row and purged on terminal, fence,
cancellation, loss, or generation replacement. It is not permanent symbol
history, a rebuildable index, or another canonical aggregate copy.

## 4. Aggregate behavior

For every aggregate input, the engine validates schema, binding, canonical
symbol, source, interval, values, source position, delivery evidence, lifecycle,
and correction horizon before mutation. It then produces exactly one bounded
disposition and accounting consequence.

- Equal values for an existing identity are an exact duplicate and do not
  mutate canonical values.
- A greater current-epoch live position revises that identity; a lesser live
  position is nonprecedent.
- Historical evidence under a validated current request token may fill a
  missing identity anywhere inside that request's exact session interval,
  including identities older than the live correction horizon. It never
  reopens live correction for a sealed identity. Live evidence wins regardless
  of delivery order.
- Compacted sealed-live presence is independently supported canonical evidence
  even when its discarded values predate the current historical request. The
  current request may reconcile through that identity without reopening it.
  When request-scoped exact comparison evidence exists, equality is an exact
  duplicate and inequality is a live-preserving local conflict. When the live
  values compacted before the request and equality is therefore no longer
  reconstructible, any returned historical row is conservatively classified
  as live-preserving conflict evidence; it cannot poison the independently
  supported live coverage, replace the live fact, or create a false current
  claim. Compacted presence without sealed-live support remains ambiguous and
  cannot authorize recovery completion.
- Unequal historical rows claiming one identity within one result withdraw any
  historical-only value installed from that result and leave the affected
  historical coverage unknown. A historical/live mismatch retains the live
  canonical value and its independently supported trust, records one bounded
  local conflict, and makes only the dependent historical coverage or fields
  unknown where the mismatch prevents their proof.
- An ordinary accepted revision applies one inverse/forward delta to the
  prefix or tail-derived sufficient state and notifies evaluation of only the
  affected symbol and proof windows.
- Evidence later than engine time or outside the accepted epoch/generation is
  rejected or fenced without mutation. A live first delivery or revision
  beyond the correction horizon is rejected as too late. Validated current-
  generation historical fill follows the source-aware rule above: an older
  missing identity may be installed and folded into the prefix after result-
  local conflict resolution, but it cannot revise a sealed live identity.

Coverage is evidence, not the absence of a row. A complete successful
hydration result can prove no print for its exact interval. Failed, canceled,
fenced, incomplete, or ambiguous work leaves the affected interval unknown.
Recognized invalid aggregate evidence is local to the usable identity;
unclassifiable identity or canonical ambiguity invokes the engine's explicit
noncurrent/suppression path.

## 5. Hydration and recovery installation

One engine-owned generation ledger plans every valid-prior-close symbol for
fresh hydration `[S,R)` or exact gap recovery. One bounded REST worker pool
accepts exactly 1, 2, 4, or 8 workers and defaults to 8. Workers own request
construction, at most two pages, at most three bounded attempts per page, the
existing page/row limits, normalization, and one immutable terminal result per
request. They have no access to `SymbolState`, generation completion,
currentness, ranking, or the live ingress fence.

The sequence is fixed:

1. an aggregate epoch is acknowledged and the engine accepts its exact
   connection position;
2. the engine captures `R`, starts one generation, and subscribes the live
   tail before historical work can establish completion;
3. worker results carry binding, generation, request token, symbol, exact
   interval, complete-pagination evidence, rows or explicit empty, and one
   terminal disposition;
4. the engine verifies the active ledger entry, merges rows through the same
   aggregate path, and records the terminal and coverage consequence once;
5. after every request is terminal, the engine requests an ingress fence
   behind all provider frames already read for the accepted epoch; and
6. only acceptance of that ordered fence, followed by an ordinary successful
   evaluation, can restore a current claim.

Live arrivals remain active during hydration. A replacement epoch or
generation fences old results and the old fence. Completed empty is terminal
success and proves no print only for the exact complete interval. Row rejects
or conflicts remain separate from transport success. The identities
`planned = value + empty + failed + canceled + fenced` and the associated row
disposition identity must reconcile before completion can be trusted.

Exact gap recovery remains plannable when the committed watermark has stalled
long enough for accepted post-watermark live identities to fold into the
prefix. Source-aware registration admits only folded slots with sealed-live
support, reconciles returned REST rows under the rule above, and still requires
the exact terminal accounting, ordered ingress fence, and successful ordinary
evaluation. Compaction therefore cannot strand recovery or freeze later
ordinary live evaluation.

## 6. Failure and trust boundaries

The smallest false success is a complete-looking generation whose interval is
not supported through its live fence. Generation, request token, epoch,
interval, terminal accounting, and fence coordinates are therefore validated
inside the owner before currentness is possible. A worker completion,
successful HTTP status, empty page, or progress count alone is never coverage.

Invalid aggregate states prevented by construction include writable aliases,
duplicate symbol slots, more than one active generation, reopening the sealed
prefix, and a second production canonical state. Runtime validation contains
foreign generations, nonterminal request results, malformed values,
nonprecedent evidence, conflicts, and fence mismatch. Any global accounting or
canonical integrity failure closes currentness; local invalid evidence stays
local where its identity is trustworthy.

## 7. Bounds and diagnostics

All state is bounded by the binding cardinality, 961 tail identities per
symbol plus the predecessor mark, fixed-cardinality coverage/conflict evidence,
one active hydration generation, one request per planned symbol, and the
existing bounded REST page/row/byte policy. Result delivery may be chunked,
but chunks remain owned by one terminal result and the engine retains neither
response bodies nor duplicate normalized histories after merge.

Hydration concurrency is bounded independently from engine ownership. At most
eight workers may perform blocking acquisition. The plan-wide normalized-row
and cumulative-response budgets remain fixed by population and interval; the
resident-result budget scales only with the configured worker count and stays
finite. Cancellation stops new scheduling, every started worker joins, and
out-of-order worker completion cannot complete the generation or place its
fence before every exact request token is terminal.

Diagnostics use fixed disposition families and counters. They expose tail
records, prefix fold position, coverage classes, canonical mutations,
recomputations, generation/work/row accounting, conflicts, fenced inputs, and
bound hits without symbol-keyed logs or provider prose in retained state.

## 8. Primary proofs and slices

| Slice | Primary proof | Claim, dangerous counterexample, observable distinction, limitation |
| --- | --- | --- | --- |
| `LBR-A1` | `P-LBR-A1-CANONICAL` | A deterministic approved fixture corpus drives insert, equal duplicate, revision, withdrawal/conflict, ordinary out-of-order live input, REST/live delivery permutations, current-token historical fill both inside and far older than the 16-minute horizon, the exact live 16-minute/equality boundary, no-print, invalid, and unknown coverage through both the approved baseline oracle and replacement semantic projection. It requires historical/historical conflict to withdraw only the historical result, historical/live mismatch to retain independently trusted live state while localizing historical uncertainty, and equal marks, coverage classes, sufficient-field inputs, and accounting—not private struct equality. It asserts one mutable record per identity and bounded fold state. It detects a latest-price-only, delivery-order-dependent, or horizon-blind historical implementation. It does not prove full hydration, scale, or provider correctness. |
| `LBR-A2` | `P-LBR-A2-HYDRATION` | A real engine plus bounded fake REST/fence producers composes value, exact empty, malformed row, failure, cancellation, superseded generation, epoch loss, post-live gap recovery, live-over-REST precedence, and a fence placed behind already-read live work. A long-stall trace accepts more than 16 minutes of post-watermark live rows, folds early identities, loses/replaces the epoch, plans exact `[T,R)` across sealed-live presence, reconciles equal/unequal REST rows without reopening live authority, restores current only through the fence evaluator, and then advances ordinarily. It observes exact work/row identities, coverage consequence, lifecycle/currentness, and no current publication before fence plus evaluation. It does not prove provider availability or connection retry execution. |
| `LBR-A3` | `P-LBR-A3-PARALLEL-HYDRATION` | The ordinary scanner runs the same generation and live-tail composition with 1, 2, 4, and 8 bounded workers under deliberately unequal request latency. It proves the configured active-worker ceiling, out-of-order result/token correctness, exactly one terminal per request, finite plan/resident budgets, cancellation and joined shutdown, live aggregate progress without sustained queue growth, fence eligibility only after all work is terminal, and byte-for-byte canonical/ranking/T/Q/API equivalence with the accepted one-worker result. It detects a worker-owned state mutation, early fence, dropped/duplicated terminal, unbounded result backlog, or acquisition concurrency that stalls the live consumer. It does not establish provider speedup or authorize a live request. |

`LBR-A1` implements the compact prefix/tail, canonical merge, coverage, and
affected-state mutation. Acceptance makes the old `symbolAggregateState`
maps/bitmaps and parallel `priceRange`, `activity`, and `mvpMeasurements`
aggregate copies removable as canonical authorities; temporary read-only
projection adapters may remain only until `LBR-B3`.

`LBR-A2` installs fresh/gap hydration and fence acceptance into the compact
state. Acceptance makes checkpoint/replay-driven live installation branches,
old hydration-to-aggregate proof maps, and any old aggregate backing retained
for hydration removable from the supported live path. It does not decide
whether unsupported replay/checkpoint tools are deleted; `LBR-E1` owns that
source disposition.

`LBR-A3` restores bounded parallel acquisition after the 2026-08-24 live
observation showed the accepted one-worker restriction regressed late-start
hydration without addressing the later live-ingress failure that motivated the
replacement. It changes no canonical, generation, merge, fence, readiness,
ranking, T/Q, or publication meaning. Acceptance supersedes only A2's
one-worker configuration restriction and makes D1 the next permitted slice.

## 9. Verification, review, and implementation discretion

Each slice runs its primary proof, affected short and race tests, focused vet,
`git diff --check`, and the repository ordinary command at the slice gate.
Capability review is read-only and focuses on sole ownership, merge and
coverage equivalence, false hydration success, correction-boundary ties,
memory bounds, and absence of writable aliases. A narrow slice review is
triggered only if generation/fence linearization or a conflict can still
produce a false current result after the primary proof.

The implementer may choose private value types, arrays, interval encodings,
delta indexes, result chunk size, and package layout. The implementer may not
change a product formula, source precedence, correction horizon, empty meaning,
session/window rule, reference acquisition behavior, readiness rule, number of
state owners, or add replay/checkpoint compatibility work.

Provider access, selection/enrichment, T/Q, ingress decoding/queue replacement,
API/UI semantic changes, public deployment, replay repair, and checkpoint
repair are non-scope. A3 may change only the wrapper/launcher/scanner/
operations worker-count surfaces required to restore the bounded pool. Slice
evidence and status are recorded only in the delivery program.
