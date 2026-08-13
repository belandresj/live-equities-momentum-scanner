# ScannerStateEngine and canonical state — S2 canonical aggregates

**Parent contract:** [ScannerStateEngine, canonical symbol state, and feature-boundary implementation details](../scanner-state-engine-and-canonical-state.md)

**Normative responsibility:** Aggregate fact and canonical-state shape,
aggregate-specific validation, identity/equality/precedence, correction and
historical-fill behavior, finalization/presence, exact bounds, and S2 proof

**Controlling requirements:** `ENG-AGG-01`; `PG-RANK-02`,
`DTE-WINDOW-01`, `DTE-WINDOW-02`, `DTE-WINDOW-04`, `DTE-AGG-01`–`DTE-AGG-03`,
`DTE-MERGE-01`–`DTE-MERGE-05`, `DTE-RECOVERY-01`

**Allocated slices:** `S2` only

**Document dependencies:** Parent; accepted S1; [authority and reuse](authority-and-reuse.md)

**Contract approval state:** Inherits the parent contract approval; not an
independent component authority

**Implementation status:** `S2` implemented and owner-accepted 2026-08-05
after its complete `ENG-AGG-01` primary proof, all accepted S1 proof reruns,
package/race/repository verification, correction of every independent-review
finding, and successful final targeted re-review. No S2 review or acceptance
item remains open. That acceptance did not itself authorize `S3`; `S3` was
later separately authorized and owner-accepted.

## S2 outcome

The S1 transition becomes the sole owner of normalized one-second aggregate
acceptance and canonical aggregate state. Every input is validated before
affected mutation, maps to one `(symbol,window_start)` identity, produces one
explicit aggregate disposition, and obeys live/replay causal precedence,
historical fill-only semantics, the inclusive 16-minute correction horizon,
strict finalization, and exact bounded retention.

S2 does not implement provider normalization, hydration/recovery planning,
active production ledgers, feature formulas, ranking, timers/`T`, or immutable
publication.

## Semantic input and owned state

| Item | Normative meaning | Bound/owner |
| --- | --- | --- |
| Normalized aggregate | Binding, source kind/context, exact symbol, one-second `[window_start,window_end)`, OHLC, economic volume, VWAP, Average Trade Size and ATS provenance, receipt/logical-delivery evidence, and source position. | One immutable self-contained value. Symbol must be an exact installed member and `<=64` bytes. |
| Identity/equality | Identity is `(symbol,window_start)` within the binding. Equality compares every normalized canonical numeric value, ATS, and ATS provenance exactly; finite signed zero is numeric zero. | Binding/symbol/start define context/identity. Window end is fixed by one-second validity. Delivery/context metadata never changes equality. |
| Canonical aggregate record | Canonical values, immutable first-acceptance evidence, current authority/precedence evidence, optional greatest live causal support, and bounded conflict status. | Engine-owned. Equal later evidence may advance causal support without rewriting values or first acceptance. |
| Per-symbol core | Copied reference fact, accepted same-session mark/no-mark, separate lazy 57,600-slot presence and historical-conflict bitmaps, lazy correction tail keyed by identity, one latest mark, and fixed scalar status. | Exactly one logical record per bound symbol. Absence remains absence; no writable alias escapes. |
| Future ingress/reconciliation seam | Accepted epoch/positions, positive generation/tokens, exact intervals/fences/outcomes and bounded active ledgers only after Components 5/6 define them. | S2 retains presence needed to reject false missing overlap, but creates no provider/hydration/no-print/recovery placeholder or production active ledger. |
| Historical-result proof context | A package-private, bounded immutable per-identity decision snapshot used only to prove the pure canonical decision for one admitted historical row. Admission reads caller-side setup synchronously, copies only the target record/conflict and result cardinality into the FIFO node, and retains neither map; canonical records never retain the context and the engine never mutates caller-owned maps. | Proof-only: one optional canonical record plus fixed metadata per queued proof input, with source-result cardinality bounded by the exact requested interval. Component 6 owns production ledger registration, accumulation, terminality, and cross-row ownership. |
| Correction boundary | After source-context checks, live/replay is mutable iff `0 <= admission_engine_time-window_end <= H`, inclusive. Future end is rejected; strictly older first/revision is too late. | Fixed `H=16m`; at most 961 whole-second full records per active symbol plus one older latest mark. Current-token historical fill may be older but cannot reopen live correction. |

## `ENG-AGG-01` — aggregate acceptance, merge, and retention

Before mutation, validate envelope/schema, binding, run/source compatibility,
exact symbol, one-second/session interval, structural fields, and the applicable
epoch, generation/token/interval, and source position. No rejected branch may
partially change values, authority, presence, mark, installed dependents, or
aggregate accounting.

The ordered algorithm is normative:

1. Resolve the identity against the mutable tail, latest mark, finalized
   presence/conflict summary, and any future active bounded reconciliation
   ledger.
2. Compare exact canonical equality as defined above. Delivery evidence never
   creates another identity or changes value equality.
3. Equal values are `exact_duplicate`: retain canonical values and original
   first-acceptance evidence. An equal current-epoch live observation with a
   greater position must advance greatest live causal support; otherwise a
   later lower revision or historical conflict could falsely win.
4. For live/replay first observation, out-of-order new identity, or revision,
   apply the nonnegative inclusive horizon. `window_end > admission_engine_time`
   is `rejected_future_event_time`; equality at `window_end+16m` is mutable;
   one nanosecond later is `too_late`.
5. Current-epoch live unequal values may revise only at a strictly greater
   `(connection_epoch,frame_sequence,array_index)`. Lower is
   `rejected_nonprecedent`. Repeated position with unequal values is global
   canonical/ingress integrity failure; publish neither candidate as current.
6. Within one replay artifact, a strictly greater `record_ordinal` may revise
   inside `H`; lower is rejected; repeated ordinal with unequal values is
   replay-artifact integrity failure. Engine sequence still follows FIFO and
   wall speed is never precedence.
7. Historical input requires the exact current generation, request token,
   symbol, and containing interval. Equal historical/live is duplicate.
   Historical can fill only when no live authority exists. Unequal historical
   against valid live authority retains live and records bounded discrepancy
   provenance without installing historical-conflict state or making coverage
   unknown; deterministic live precedence establishes the identity's
   correctness. This includes unequal OHLCV/VWAP/ATS and source-specific ATS
   provenance. A conflict makes dependent coverage/fields unknown only when
   correctness cannot otherwise be established.
8. Against an immutable bounded result context, two unequal historical rows claiming one identity in the same result have
   no arrival-order winner. Mark the identity historical-conflicted, withdraw
   any historical-only value installed from that result, recompute latest mark
   and every installed dependent contributor, and leave affected coverage
   unknown. Existing live evidence still wins.
9. Accepted insert/revision/withdrawal updates the latest mark by greatest
   accepted event-time identity, runs every installed dependent contributor in
   fixed source order, completes accounting, then permits later publication.
   A correction may change state without advancing `T`.
10. Strict finalization sets the exact session-presence bit, folds only state
    required by an installed approved contributor, and discards exact values
    only when no active reconciliation needs their authority/conflict evidence.
11. A later request interval containing a finalized-present identity whose
    exact values were compacted is rejected at registration. Component 6 must
    split work into exact missing runs; it cannot relabel accepted data missing.

Historical rows older than `H` are accepted only through a current exact token
registered while each identity is absent or still has exact retained overlap
evidence. They may update latest mark and future finalized contributor state,
but never reopen the ordinary live tail. Any permitted overlap is charged to
the mutable tail or a later finite active ledger retaining exact values and
authority until its generation terminates.

The owner approved `H=16m` on 2026-08-05. Massive documents an initial roughly
two-second aggregate delay and a subsequent 15-minute intraday correction
buffer; measuring from `window_end` leaves 58 seconds of additional margin.
This establishes the chosen boundary rationale, not provider conformance or
end-of-day reconciliation. Changing `H` requires an owner-approved contract
revision, new evidence, and reallocated retention/contributor proofs.

## Trust and failure containment

| Condition | Exact disposition/state effect | Dangerous false success prevented |
| --- | --- | --- |
| Wrong binding/date or stale epoch/generation/token/out-of-request work | One `fenced` result before affected mutation. | Stale context receiving a sequence and changing canonical/coverage/lifecycle before fencing (`DF-06`). |
| Structural/schema/symbol/source/run/lifecycle/interval invalidity | One fixed aggregate rejection before mutation. | Structurally plausible but context-invalid bar appearing canonical. |
| Equal observation | `exact_duplicate`; original values/first evidence stay; greater live support advances when applicable. | Equal newer live support not advancing authority, allowing later lower evidence to win (`DF-07`). |
| Lower/repeated causal evidence | Lower unequal retains authority and rejects; repeated unequal live/replay is global integrity failure. | A valid-looking lower or contradictory row replacing authoritative state (`DF-07`). |
| Future/strictly late | Reject and retain exact prior state; future never enters tail/presence. | Negative age evading retention bounds or becoming the mark. |
| Historical/live conflict | Retain live and localize unknown status to dependent historical coverage/fields. | Recovery overwrite of live state (`DF-08`). |
| Historical/historical conflict | Withdraw arrival-order-only value and recompute dependents; no arbitrary winner. | First/last/map order silently selecting canonical truth (`DF-08`). |
| Compacted-present overlap registration | Reject/split around the present slot. | Discarded live authority being relabeled missing and overwritten (`DF-08`). |

Successful empty hydration is not an S2 fact and creates no bar/mark/no-print
claim. Component 6 later proves its terminal-work/no-print boundary.

## Aggregate accounting and bounds

For completed aggregate transitions:

```text
aggregate_inputs_consumed
  = aggregate_inserted
  + aggregate_revised
  + aggregate_withdrawn_conflict
  + aggregate_exact_duplicate
  + aggregate_rejected
  + aggregate_fenced
  + aggregate_integrity_failure
```

Each triggering input occupies exactly one primary bin. Rejection subreasons,
source kind, binding/epoch/generation, lifecycle, and required/optional status
are overlapping dimensions, never new populations. A revision is not also an
insert/rejection. Historical-conflict withdrawal is the triggering input's
primary outcome; the prior input remains in its original bin.

| Resource | Exact bound/construction gate | Proof limitation |
| --- | --- | --- |
| Canonical symbols | Exactly `universe_total`; fixed core per symbol, lazy heavy state. | No symbol insertion after binding. |
| Mutable tail | Inclusive `H=16m`: `<=961` full identity records per active symbol plus one older mark; mathematical maximum `100,000*961=96,100,000` records plus 100,000 marks. | Finiteness, not acceptable production memory. Component 8 measures representation/runtime. |
| Finalized presence | First lazy 57,600-bit session bitmap per active symbol; absolute ceiling 5,760,000,000 bits = 720,000,000 raw bytes before overhead. | Presence prevents false missing. Component 7 checkpoints it; Component 8 measures lazy container cost. |
| Historical conflict | Second lazy 57,600-bit session bitmap per active symbol; absolute ceiling 5,760,000,000 bits = 720,000,000 raw bytes before overhead. | Conflict prevents an arrival-order winner after exact historical values leave the tail. Component 7/8 own checkpoint representation and measured capacity. |
| Combined session bitmaps | At most two lazy bitmaps per active symbol: 11,520,000,000 bits = 1,440,000,000 raw bytes at 100,000 active symbols, before allocation/container overhead. | This is the charged theoretical ceiling, not an acceptable-capacity claim. No third session-sized bitmap exists in S2. |
| Finalized values | No session-long raw value history. Compact only after contributors fold approved state and no reconciliation overlap needs exact evidence. | Component 3 must bound summaries/prove correction equivalence; Component 6 bounds missing-run/active-ledger state. |
| Source position | One active/last epoch position plus authority already charged to retained identities; no position history. | Component 5 later defines transport position bounds/ambiguity. |
| Historical ledger | Absent in S2. Future constructor must reject missing/overflow bounds and registration over compacted-present slots. | S2 does not claim hydration-memory capacity. |

Arrival count and run duration cannot enlarge these sets beyond the stated
cardinality. No retry/work/publication/contributor history is introduced.

## Evidenced edge cases

The primary matrix must cover all of these in one proof family:

- exact duplicate with later receipt/source evidence and mandatory greater-live
  support update;
- first/revision exactly at `window_end+16m`, one nanosecond beyond, and future
  `window_end`;
- in-horizon out-of-order new identity;
- greater/lower/repeated live position and greater/lower/repeated replay ordinal;
- current-token historical fill older than the live tail;
- historical equal/live conflict and two conflicting historical rows against a deep-copied immutable result context, including caller mutation after admission;
- finalization followed by late overlap registration over compacted presence;
- latest-mark recomputation after correction/withdrawal; and
- a long mixed trace proving the 961-record bound and separately charged
  presence state.

Evidence is the Phase 1 merge contract, the owner decision/provider rationale,
and only the whitelisted v2 `TestInclusiveRevisionAndStrictFinalizationBoundaries`,
`TestWindowAndDateValidationBoundaries`, and
`TestLongPathEquivalenceAfterPrefixFolding` behavior. V2 `Apply`,
`ApplyRecovery`, `state.go`, and `types.go` are not ports.

## Primary proof

`ENG-AGG-01` receives one complete aggregate validation/merge/retention matrix.
It must exercise aggregate `DF-06`, `DF-07`, `DF-08`, and the aggregate-bound
portion of `DF-16` across every validation/source/precedence/horizon/conflict/
finalization branch above. After every input it asserts the exact disposition,
canonical values, first evidence, greatest causal authority, conflict/coverage,
latest mark, installed-dependent recomputation trigger, presence/tail state,
and accounting bin.

Observable conformance means: no double bar or arrival-order winner; admitted
historical decision context cannot be changed through a caller-owned writable alias; equal
greater live evidence advances authority; lower/repeated/future/late evidence
has the exact containment; conflict withdrawal recomputes; compacted presence
cannot reenter missing work; and full records never exceed 961 per active
symbol. Construction inspection confirms one identity-keyed owner map,
historical fill-only branch, presence exclusion, and no alternate merge path.

The proof does not establish production historical-result ledger ownership,
registration, accumulation, or terminality; those remain Component 6. It also does not establish provider normalization/position authenticity,
provider conformance/end-of-day reconciliation, pagination/no-print planning,
future active-ledger implementation, Component 3 formulas, or acceptable
worst-case memory/latency.

## S2 implementation assignment boundary

**Outcome:** the exact S1 transition owns complete canonical aggregate
validation, merge, authority, finalization/presence, and bounded tail.

**Allowed package:** `internal/engine` and focused tests only. No provider
decoder, REST worker, feature/ranking, or public type.

**Approved v2 use:** only the named correction/window/long-path test cases as
behavior evidence; no implementation port and no `ApplyRecovery`.

**Review artifact:** merge decision table; exact proof result after each input;
construction/authority walk; withdrawal/recompute path; tail/presence bounds;
limitations; source audit; and whether S3 remains valid.

**Explicitly deferred:** Component 3 formulas/evaluator; Components 4–6
normalization/artifacts/tokens/planning/active ledger; capacity approval;
clock/`T`; publication.

**Independent review result:** satisfied 2026-08-05. The final targeted
re-review cleared aggregate identity, causal precedence, durable integrity
containment, historical fill-only and live-authority containment, the inclusive
correction boundary, compacted-presence trust, bounded proof-only historical
result evidence, and preservation of the S1 FIFO/API, with no remaining
actionable finding.

Stop for a needed source-position/future-clock/merge/reconciliation behavior
outside this contract, inability to bound a retained set, Component 3 evidence
showing `H` cannot preserve approved corrections, an unapproved v2 dependency,
or failure of the inclusive-boundary/long-path proof.
