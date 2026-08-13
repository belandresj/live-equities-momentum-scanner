# Partial-session live test mode

**Status:** Owner-requested implementation specification; planning complete,
not yet implemented or accepted

**Requested:** 2026-08-12

**Purpose:** Make repeated owner-authorized live-provider tests materially
faster by hydrating only from a configured same-day test boundary, while
preserving the real 04:00–20:00 New York session and making every partial claim
unmistakable.

**Authority:** The direct owner request permits a distinct diagnostic/test
behavior. It does not revise ordinary scanner semantics. The fixed full-session
meaning in the product and architecture documents remains authoritative for
ordinary live mode.

**Controlling requirements:** `PG-RANK-01`–`PG-RANK-05`, `PG-FEATURE-01`,
`PG-FEATURE-02`, `PG-AVAIL-01`, `PG-AVAIL-02`, `PG-AVAIL-03`, `PG-OPS-01`,
`PG-OPS-02`, `PG-OBS-01`–`PG-OBS-03`, `PG-UI-01`, `PG-UI-02`,
`DTE-SESSION-01`–`DTE-SESSION-04`, `DTE-WINDOW-01`, `DTE-WINDOW-02`,
`DTE-RECOVERY-01`–`DTE-RECOVERY-05`, `DTE-COMMIT-01`–`DTE-COMMIT-04`,
`DTE-CHECKPOINT-01`–`DTE-CHECKPOINT-03`, `DTE-TQ-01`–`DTE-TQ-03`,
`LIFE-MODEL-01`–`LIFE-MODEL-04`, `LIFE-HYDRATE-01`–`LIFE-HYDRATE-07`,
`LIFE-LIVE-01`–`LIFE-LIVE-05`, `LIFE-RECOVER-01`–`LIFE-RECOVER-06`, and
`LIFE-PUBLISH-01`–`LIFE-PUBLISH-03`.

**Approved dependencies:** Accepted Components 1–3, 5–11 and the current live
recovery corrections. No Version 2 source was inspected or is needed.

**Layout:** Compact single-file focused specification, Sections 1–19.

## Sections 1–4 — Outcome, scope, ownership, and settled boundary

### Outcome

An operator may start the real live transport at time `R` but choose a
whole-second New York test evidence boundary `X`, where `S <= X < R <= E`.
Fresh REST hydration covers `[X,R)` instead of `[S,R)`. The live tail remains
active during hydration and the ordinary ingress fence still reconciles every
admitted frame through the captured boundary.

After hydration and fencing, the scanner may publish an explicitly labeled
partial-session test ranking. It is exact only for this proposition:

> Among the complete bound universe, rank by ordinary Day % the symbols that
> demonstrate the ordinary 60-second qualification gate at or after `X+60s`,
> using only exact aggregate evidence in `[X,T)`.

This ranking is useful for exercising transport, hydration, reconciliation,
canonical merge, current marks, Day %, qualification, top-20 ordering, T/Q,
API, UI, recovery, and operational behavior. It is not the ordinary
full-session qualified table: a symbol that qualified before `X` but not after
`X` can be absent.

Ordinary live mode is byte-for-byte and semantically unchanged when the test
flag is absent. Its session remains `[S,E)`, fresh hydration remains `[S,R)`,
and its normal readiness and checkpoint behavior remain intact.

### Single ownership boundary

`ScannerStateEngine` remains the only canonical-state, lifecycle, coverage,
qualification, ranking, watermark, T/Q-intent, and publication owner. The
test boundary is one immutable run configuration copied into that owner before
binding installation. Component 6 still asks the engine for a hydration plan;
it never supplies `X` on an individual request. Component 8 derives test
readiness from one immutable publication. Components 10 and 11 only map and
label that state.

This is an extension of the existing evaluator and hydration planner, not a
second engine, evaluator, watermark, queue, population owner, or readiness
store.

### Explicit non-scope

- Do not change the binding's real `S=04:00` or `E=20:00`, aggregate identity,
  prior-close meaning, Day-% formula, gate thresholds, correction horizon,
  exact ordering, or top-20 limit.
- Do not claim that `[S,X)` was empty, covered, hydrated, qualified, or
  checkpointed.
- Do not rename a value computed from `X` as From 4AM, session HOD, session
  range, or ordinary Activity.
- Do not load or write ordinary checkpoints in test mode.
- Do not add a database, separate service, alternate market-state graph,
  browser calculation, provider-specific shortcut, dynamic moving start, or
  automatic credentialed run.
- Do not alter reference-data retrieval. The same current-date universe and
  exact adjusted prior closes are required.
- Do not make partial-session mode a production default or let an ordinary
  launcher invocation inherit a prior test boundary.

### Fixed meanings preserved

`S` and `E` remain the Component 1 schedule bounds. `X` is named
`evidence_start`, not `session_start`. `R` remains the acknowledgement receipt
boundary. `T` remains the sole committed aggregate watermark. All windows are
half-open. Live/historical identity and precedence are unchanged. Successful
empty REST work proves only no print in its exact requested interval.

`backend_ready`, `ranking_current`, `qualified_current`, `/readyz=200`, From
4AM, session HOD/range, ordinary Activity, and ordinary checkpoint compatibility
retain their existing full-session meanings. Test mode must not emit any of
those full-session success claims merely because `[X,T)` is exact.

## Sections 5–8 — Evidence questions and reconnaissance

The relevant implementation seams are already present in this repository:

- `cmd/scanner/main.go` owns live CLI validation and composition;
- `scripts/run-private-scanner` owns the supported private launcher;
- `cmd/scanner/checkpoint_mode.go` selects checkpoint-on/off composition;
- `internal/operations/runtime.go`, `live.go`, and `status.go` compose the sole
  engine and derive operational status;
- `internal/engine/hydration.go` derives fresh/checkpoint/gap intervals;
- `internal/engine/qualification.go`, `feature_price_range.go`,
  `feature_activity.go`, and `evaluator.go` implement the one ordinary
  aggregate evaluation path;
- `internal/engine/publication.go` and `replay_view.go` seal immutable state;
- `internal/engine/tq.go` derives selected-symbol T/Q intent from ranking;
- `internal/engine/checkpoint.go` and `checkpoint_cadence.go` own checkpoint
  projection/install/cadence;
- `internal/snapshotapi/schema.go` and `mapper.go` own the V1 wire mapping; and
- `ui/model.js` and `ui/render.js` own validation and explicit presentation.

No predecessor inspection is justified. The questions are semantic rather
than structural: which claims remain mathematically valid with evidence only
from `X`, and where must the existing full-session claims remain unavailable?
The decisions below answer them directly from the current authority and code.

## Sections 9–14 — Inputs, behavior, trust, and bounds

### Immutable run configuration and CLI

Add one optional live-only flag:

```text
--live-test-start HH:MM:SS
```

The value is parsed in `America/New_York` on `--trading-date`. It must identify
an exact whole second, satisfy `S <= X < E`, use the current New York trading
date, and not be later than the current whole second when startup validates it.
DST conversion comes from the already accepted schedule/timezone, not a fixed
UTC offset. Duplicate values, malformed values, replay use, a date other than
today, pre-session/post-session execution, or a future `X` fail before opening
the WebSocket or starting hydration.

No minimum lookback is required. If `X` is fewer than 60 seconds before `R`,
the scanner simply shows test qualification warming until `T >= X+60s`.

The supported launcher accepts and forwards the same flag:

```bash
./scripts/run-private-scanner \
  --trading-date 2026-08-12 \
  --live-test-start 14:30:00 \
  --open
```

Before starting the binary, the launcher prints a prominent line containing
`PARTIAL-SESSION LIVE TEST`, the trading date, `X`, and
`ordinary backend readiness disabled`. It does not prompt interactively.

Represent the configuration internally as an immutable closed value, for
example:

```text
SessionScope {
    kind: full_session | partial_session_test
    evidence_start: X // S in full_session
}
```

Pass it through `operations.New...` into `engine.Config`; validate it again
when the binding is installed. Do not add `X` to `HydrationPlanInput`, because
that would let the coordinator select a new boundary per generation.

### Hydration, fencing, and recovery

Add one engine-owned hydration purpose, `partial_session_test_bootstrap`, or
equivalently retain fresh-bootstrap purpose with immutable scope attached to
the plan. The observable purpose must distinguish the test interval.

For the initial generation:

```text
start = X
end   = R
requests = every valid-prior-close symbol when X < R
```

All existing request tokens, row validation, terminal accounting, REST/live
precedence, live-tail pumping, budget checks, and ingress-fence rules apply.
The fence may establish exact test coverage only for `[X,T)`. It must not set
ordinary full-session `no_print_through(T)` or ordinary full-session
complete-population coverage. Instead it records a separate fixed-cardinality
test partition:

```text
valid_prior_close
  = test_trusted_rankable_mark
  + test_trusted_below_price_mark
  + no_print_since_test_start
  + test_invalid_mark
  + test_unknown
```

The existing ordinary population identity remains honest; symbols lacking
`[S,X)` evidence remain unknown in that full-session accounting. The test
partition is the evidence used only by the partial-session test mode.

If the live epoch is lost during bootstrap, supersede the generation normally
and retry from the same immutable `X`; never move `X` forward. After a test
ranking becomes current, same-process gap recovery remains the ordinary exact
`[T_supported,R)` recovery because all required history begins no earlier than
the already established `X`. Recovery failure removes `test_ready` and follows
the existing bounded retry/suppression path.

### Ranking and field decision table

| Output | Partial-session behavior | Why |
| --- | --- | --- |
| Last / mark age | Current when the latest accepted mark in `[X,T)` is valid and causally supported. | It requires no pre-`X` price history. |
| Day % | Current and used as the sole numeric ranking key. | Adjusted prior close is independently exact; `100*(Last/prior-1)` needs no `[S,X)` aggregates. |
| Qualification | `test_qualified` only. Evaluate the unchanged 60s/5s gate for proof ends `P >= X+60s`; every contributing identity lies in `[X,P)`. Preserve provisional correction and finalization mechanics, but never call this the full-session qualification latch. | A post-`X` passing proof is mathematically exact; a possible pre-`X` proof is unknown. |
| Ranking | New mode `partial_session_test`, ordered by exact unrounded Day % then symbol, after test qualification, across the complete test-covered universe. | Exact within the stated test proposition, not exact for the ordinary all-session passer set. |
| From 4AM % | Unavailable, reason `partial_session_history`. | The first aggregate after `X` is not necessarily the first aggregate after 04:00. |
| HOD drawdown | Unavailable, reason `partial_session_history`. | A higher price may exist in `[S,X)`. |
| Session range | Unavailable, reason `partial_session_history`. | Session low/high require `[S,T)`. |
| 30-minute range | Warming until `T >= X+30m`; then current only with exact `[T-30m,T)` coverage and ordinary validity. | Once the entire mathematical window is after `X`, no omitted history participates. |
| 60-minute range | Warming until `T >= X+60m`; then current only with exact `[T-60m,T)` coverage and ordinary validity. | Same reasoning as the 30-minute range. |
| Activity | Unavailable, reason `partial_session_history`. | Ordinary Activity compares against eligible blocks from 04:00; replacing that baseline with `X` would be a different metric. |
| Tape Rate / Spread | Use the ordinary T/Q acknowledgement, coverage, warm-up, gap, and pressure rules for displayed test rows. | These measurements depend on post-subscription causal coverage, not pre-`X` aggregates, and never affect ranking. |

The existing evaluator performs one population scan and stages one candidate.
It receives the immutable scope and branches only where the proof boundary
differs: test population classification, earliest qualification proof end,
full-session feature unavailability, and the new ranking mode. There is no
second test evaluator or separately mutable test-ranking cache.

Test rows may create ordinary selected-row T/Q intent because their aggregate
selection is current within the explicitly labeled test scope. `tq.go` must
accept `partial_session_test` as intent-eligible without treating T/Q health as
a test or production readiness predicate.

### Readiness, publication, API, and UI

The immutable publication adds a test-scope projection containing:

```text
kind = partial_session_test
evidence_start = X
test_ready
test_readiness_reason
test_ranking_current
test population accounting
```

`test_ready=true` requires all of: process live; exact binding identity;
partial-session scope; lifecycle `hydrating` or `live`; current acknowledged
aggregate epoch; reconciled startup/recovery fence; exact test population with
`test_unknown=0`; current `partial_session_test` ranking; nonstale `T` under
the existing C8 target/tolerance; valid accounting; and no suppression.

In the same snapshot, ordinary `backend_ready=false` and
`ranking_current=false`, with readiness reason `partial_session_test`.
`/readyz` remains HTTP 503. `/livez` and `/api/v1/snapshot` retain their current
transport meanings and may return 200 for a coherent test snapshot.

Extend `scanner.snapshot.v1` with one optional root `test_session` object,
present only in this mode. It contains the fields above and exact test
accounting. This is additive; ordinary snapshots remain unchanged. Update the
Go mapper validator and browser validator together. If compatibility review
finds any existing consumer that rejects additive root properties, use
`scanner.snapshot.v2` and `/api/v2/snapshot` instead of weakening validation.

The operator and UI render a persistent `PARTIAL-SESSION TEST` banner showing
`X`. When test-ready, the status is `TEST CURRENT · NOT PRODUCTION READY`.
The table caption is `Test-qualified top 20 since HH:MM:SS ET`. Rows are
visually current within the test scope, while unavailable full-session fields
remain em dashes with the explicit partial-history reason. Neither the word
`CURRENT` alone nor a green ordinary-ready indicator may appear.

### Checkpoints

Partial-session test mode forces checkpoint composition off:

- reject `--checkpoint-mode=on` when `--live-test-start` is present;
- construct no store/writer;
- attempt no discovery/install;
- submit no projection/write; and
- report checkpoint installed/submitted/in-progress/pending as zero/false.

This is the simplest safe policy. An ordinary checkpoint contains full-session
meaning and a test checkpoint would require a distinct compatibility identity,
retention contract, and restart purpose. Neither is needed to shorten the
current live-test loop. Ordinary checkpoint behavior is unchanged outside test
mode.

### Bounds and failure behavior

Reuse all current queue, frame-byte, hydration-worker, request/page/attempt,
response-byte, engine-capacity, evaluation-delay, retry, and shutdown bounds.
Do not add an arbitrary maximum lookback or minimum test duration. The shorter
interval naturally reduces possible normalized rows and REST transfer.

Invalid scope construction fails before market transport. A failed request,
unknown row consequence, unreconciled fence, stale watermark, accounting
failure, or suppression keeps `test_ready=false`; it cannot fall back to a
rank-biased subset. Zero test-qualified passers is a valid exact empty test
table only after the test population and qualification trace are complete.

The smallest dangerous false success is a snapshot labeled test-current after
hydrating `[X,R)` while silently presenting full-session qualification or
session features. The second is a no-print classification that treats missing
`[S,X)` as empty. Construction and snapshot validation must reject both.

## Sections 15–17 — Requirements, proofs, and implementation slices

### Requirements and primary proofs

| Requirement | Primary proof and dangerous counterexample | Acceptance evidence / limitation |
| --- | --- | --- |
| `PSTM-CONFIG-01` | `P-PSTM-CONFIG`: table-driven CLI/scope proof across malformed, duplicate, replay, wrong-date, future, `S`, `E`, and DST cases. Counterexample: `14:30 ET` interpreted with a fixed UTC offset or ordinary mode inheriting it. | Exact `X` or rejection before transport; ordinary no-flag config unchanged. No provider call. |
| `PSTM-HYDRATE-01` | `P-PSTM-HYDRATE`: fake socket/HTTP composition where only `[X,R)` is requested for every valid-prior symbol, live overlap reconciles, empty becomes `no_print_since_test_start`, and a missing terminal prevents test readiness. Counterexample: planner requests `[S,R)` or promotes partial empty to ordinary no-print. | Exact tokens, rows, test accounting, fence identity, and lifecycle. No live-provider latency claim. |
| `PSTM-RANK-01` | `P-PSTM-RANK`: one differential table with a symbol that passed before `X` only, one that passes after `X`, ties, correction revocation/finalization, zero passers, and one unknown symbol. Counterexample: pre-`X` latch enters the test table, raw Day-% bypasses the post-start gate, or an unknown symbol still permits test-ready. | Exact test qualification, order, cardinality, and ready state through the ordinary evaluator. Does not claim the ordinary full-session table. |
| `PSTM-FIELDS-01` | `P-PSTM-FIELDS`: boundary evaluation at `X+29:59`, `X+30m`, `X+59:59`, and `X+60m`, with a hidden pre-`X` high/open/reference block. Counterexample: From 4AM/HOD/session range/Activity appears current or a rolling range warms before its full interval is covered. | Exact status/reason/value table; T/Q is independently current after its own coverage. |
| `PSTM-TRUST-01` | `P-PSTM-TRUST`: immutable engine-to-operations-to-API-to-UI trace. Counterexample: `backend_ready=true`, `/readyz=200`, ordinary CURRENT styling, missing `X`, checkpoint work, or a browser-derived test label. | One coherent publication shows test current and production not ready; ordinary snapshot golden remains unchanged. |
| `PSTM-RECOVER-01` | `P-PSTM-RECOVER`: disconnect during bootstrap and after test-ready. Counterexample: retry moves `X`, skips the gap, restores a checkpoint, or retains current T/Q across a gap. | Same `X` on bootstrap retry; exact ordinary gap recovery after readiness; bounded exhaustion removes test-ready. |

### Two sequential slices

**`PSTM-S1 — engine and live composition`**

Implement immutable scope parsing/validation, checkpoint-off enforcement,
engine-owned `[X,R)` planning, test coverage/accounting, scoped qualification,
field availability, partial ranking, same-process recovery, T/Q intent, sealed
publication, and operator status. Primary proofs: `P-PSTM-CONFIG`,
`P-PSTM-HYDRATE`, `P-PSTM-RANK`, `P-PSTM-FIELDS`, and `P-PSTM-RECOVER`.

Expected paths are `cmd/scanner`, `scripts/run-private-scanner`,
`internal/operations`, and the narrow existing seams in `internal/engine` named
above. Do not edit reference acquisition, provider normalization, queue
semantics, or ordinary checkpoint format.

**`PSTM-S2 — public representation and dashboard`**

Add the optional test-session API projection and explicit dashboard state,
then complete `P-PSTM-TRUST`. Expected paths are `internal/snapshotapi` and
`ui`, plus focused fixtures. Do not add browser market calculations or a
second endpoint unless compatibility evidence requires the explicit V2 route.

After S2, run one fake-provider vertical from CLI parsing through UI model,
ordinary short verification, affected Go/JS tests, focused race tests for
engine/operations/snapshot capture, `go vet`, and one final read-only review.
A live-provider run is not an implementation proof and requires its own owner
authorization.

### Implementation discretion and correction

Implementers may choose private type/helper names and whether the wire object
is optional V1 or a V2 route after checking actual compatibility. They may
combine status fields if the same facts remain explicit. They may not change
the decision table, move `X`, use test state as an ordinary checkpoint, claim
ordinary readiness, or add another evaluator/owner.

If the first implementation reveals that scoping the existing qualification
state is materially more invasive than expected, keep the same two slices and
revise the internal representation—not the semantics. The simplest acceptable
fallback is still one evaluator pass with test qualification state inside the
existing canonical symbol owner. Do not replace qualified test ranking with a
raw-Day-% shortcut merely to reduce code changes.

## Sections 18–19 — Acceptance and drift audit

Implementation is complete only when:

- ordinary no-flag live behavior and API/UI goldens remain unchanged;
- the fake-provider path proves the requested interval is exactly `[X,R)`;
- all valid-prior symbols receive one terminal test outcome;
- test qualification cannot use or claim a proof before `X+60s`;
- Day % is exact and sole ordering input;
- full-session-dependent fields are unavailable and rolling fields warm at
  their exact boundaries;
- test readiness is independently true while ordinary backend readiness and
  `/readyz` remain false;
- checkpoints are neither loaded nor written;
- disconnect/retry/recovery preserve the immutable `X` and close T/Q coverage;
- ordinary short, focused race, API/UI, vet, and diff checks pass; and
- final read-only review finds no false full-session claim, second owner, or
  ordinary-mode regression.

Drift audit:

| Question | Required answer |
| --- | --- |
| Changed the real session or ordinary live semantics? | No. `S=04:00`, `E=20:00`, and ordinary `[S,R)` hydration remain fixed. |
| Fabricated missing coverage or no-print evidence? | No. Test coverage begins at `X`; full-session accounting stays unknown where `[S,X)` is absent. |
| Added a second evaluator/readiness owner? | No. One engine evaluator and one immutable publication carry both the ordinary nonready claim and test-current claim. |
| Changed Day %, gate arithmetic, ordering, or T/Q independence? | No. Only the earliest eligible test proof and scope label differ. |
| Persisted incompatible state? | No. All checkpoint activity is disabled in test mode. |
| Created avoidable constraints? | No. Any whole-second `X` from session start through the current second is allowed; existing safety/resource bounds are reused. |
| Claimed production or provider validation? | No. This mode accelerates observations and remains visibly diagnostic. |

The next decision after implementation is practical: if a partial-session run
reaches `test_ready` quickly and remains stable, it establishes transport and
post-start scanner behavior without waiting for 04:00 hydration. A later
ordinary full-session or checkpoint-backed run is still required for the real
qualified table and full-session fields.
