# CLI event-time replay correction

**Status:** Owner-approved correction direction, 2026-08-12

**Component:** Reopens Component 12 and only the exact accepted C2-C4 mechanics
that its proof shows must change. This is not a new replay component.

**Supersedes:** The mandatory 1x observation schedule, replay HTTP/API
retention, dashboard replay presentation, Chrome proof, and `C12-S2` delivery
in [`historical-replay-product-mode.md`](specifications/historical-replay-product-mode.md).
Earlier C12 implementation and evidence remain diagnostic evidence, not the
current executable product plan where they conflict with this correction.

**Preserves:** `PG-REPLAY-01`, `PG-REPLAY-02`, `PG-OBS-01`-`03`,
`ARCH-OWN-01`-`04`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, `ARCH-FLOW-04`,
`DTE-MODEL-01`-`03`, `DTE-CLOCK-04`-`06`, `DTE-WINDOW-01`-`04`,
`DTE-EVENT-04`, `DTE-TIMER-01`, `DTE-COMMIT-01`-`04`,
`DTE-REPLAY-01`-`03`, `DTE-REJECT-01`, `LIFE-INIT-01`,
`LIFE-REPLAY-01`-`03`, `LIFE-SUPPRESS-01`-`03`, and `LIFE-END-02`-`03`.

## 1. Conclusion and operator behavior

Replay is a finite CLI event-time run over stored normalized one-second
aggregates. It is not a simulated live transport and it is not a visual
playback product.

The operator supplies one complete replay artifact, its exact-date reference
cache, and New York start/end clock times. The artifact supplies the trading
date. The process creates one clean replay-mode `ScannerStateEngine`, consumes
every aggregate and required whole-second timer from session start `S=04:00`
through the requested end, and runs without wall-clock pacing. It processes
`[S,O0)` as hidden warm-up, emits scanner state at `O0`, then emits after each
completed market second through `O1`. The resulting boundary path is
`O0,O0+1s,...,O1`: exactly `seconds(O1-O0)+1` snapshot records. The `O1`
snapshot contains the effect of aggregates for the last selected market second
`[O1-1s,O1)`. The process then emits one terminal result after exact
requested-end completion, joins, and exits. Every emitted snapshot names both
replay logical time and the ordinary committed watermark `T`; production
`D=4s` is unchanged.

The intended command is:

```text
go run ./cmd/scanner \
  --run-mode replay \
  --replay-artifact /absolute/private/path/session.replay \
  --reference-dir /absolute/private/path/reference \
  --replay-start 09:30:00 \
  --replay-end 10:00:00
```

Replay stdout is NDJSON suitable for redirection. No snapshot before `O0`
reaches stdout. `O1` is the inclusive endpoint of the emitted boundary path
and the exact logical completion boundary. Bounded operator progress goes to
stderr at phase transitions (validation, hidden warm-up, observation, suffix
finalization, terminal result) so a long file scan cannot look like a hung
process; it never contains symbols, rows, artifact identity, or the private
path.

Every stdout record is a closed, bounded `scanner.replay.cli.v1` envelope with
`schema="scanner.replay.cli.v1"` and `kind="snapshot"|"result"`; every encoded
record is at most 1 MiB.

- A `snapshot` contains `logical_time`, nullable `committed_t`, `run_mode`
  (`replay`), engine `lifecycle`, `publication_id`, exact engine-owned ranking
  mode/reason, population accounting, and the immutable ordered top-20 rows
  with existing V1 aggregate fields and availability meanings. Tape Rate and
  Spread are `unavailable/replay_unavailable`. It contains no C4 source
  counters, artifact ID, or path.
- The sole `result` contains outcome, completion, bounded reason, final logical
  time, committed `T`, and exact final C4 source accounting. It contains no
  rows, artifact ID, or path. It is successful only after C4 and engine
  terminal success reconcile. Any stdout encoding failure cancels and joins
  the source, exits nonzero, and must not emit a success result.

## 2. Ownership, inputs, and non-scope

The accepted C4 artifact validator/source owns persisted trust, record order,
the single simulated clock, group timers, cancellation, requested-end suffix
validation, and terminal accounting. `ScannerStateEngine` remains the only
canonical state/evaluator/watermark/publication owner. C12 owns only CLI
configuration, hidden-versus-emitted output selection, bounded encoding, and
joined process composition.

Loading the exact-date universe/prior-close cache constructs the immutable
session binding; it is not aggregate hydration. Replay must branch before and
must not read credentials or construct live WebSocket, REST aggregate
hydration/recovery, checkpoint, T/Q, HTTP API, dashboard, or browser work.

There is no UI, HTTP listener, 1x or other wall pacer, pause, seek, reverse,
loop, playlist, checkpoint start, implicit download/compile, database, second
event type, second engine, second clock, or second evaluator. Historical T/Q
remains explicitly unavailable. Replay proves neither live arrival latency nor
correction-arrival chronology, predictive edge, or executable expectancy.

## 3. Exact requirements

| ID | Required behavior |
| --- | --- |
| `C12R-CONFIG-01` | Replay requires exactly artifact, reference directory, `HH:MM:SS` start, and later end. It derives and validates the date from the artifact, rejects live/API/UI flags and duplicate scalar flags, and fails before engine mutation on invalid cache, artifact, or bounds. |
| `C12R-PATH-01` | One C4 source admits every stored normalized record and whole-second group through the same engine aggregate/timer path. There is no fake WebSocket or aggregate rehydration. |
| `C12R-WINDOW-01` | Process `S..O1` unpaced. Suppress only stdout snapshots before `O0`; do not skip market facts, absent-second evidence, timers, ordinary group evaluations/publications, qualification progression, feature state, correction semantics, or the production evaluation delay. Emit exactly `seconds(O1-O0)+1` immutable boundary snapshots at `O0..O1`, inclusive. |
| `C12R-GROUP-01` | For validated `complete_final_bars`, record admissions mutate and account for their canonical symbols without triggering a redundant full-population projection. Every completed logical-second group from `S` through `O1` invokes the ordinary full evaluator and publication path exactly once. Partial/correction-capable replay keeps its existing behavior. No observation-start-owned evaluator, qualification algorithm, feature algorithm, or alternate canonical path is permitted. |
| `C12R-TRUST-01` | Perform one pre-engine whole-artifact validation and bind it opaquely to the same open file. Playback construction must not perform another full scan. Streaming prefix and unapplied suffix still check canonical bytes, order, counts, coverage, digest, seal, and same-file mutation before success. |
| `C12R-OUTPUT-01` | Encode only `scanner.replay.cli.v1` records under 1 MiB. Snapshots contain engine-owned publication/population facts but no C4 source accounting. The sole result contains exact final C4 accounting and no rows. Encoding failure cannot emit success and produces a bounded nonzero joined exit. |
| `C12R-END-01` | Requested end uses C4's accepted terminal fact and full suffix validation. Success requires exact source/engine/accounting reconciliation; cancellation or integrity failure emits no success, joins bounded work, and exits nonzero. Successful completion emits one result after the `O1` snapshot and exits zero without retaining an API process. |
| `C12R-PERF-01` | On the retained 2026-08-07 complete artifact (2,584,011,150 bytes, 5,691 symbols, 7,671,171 records), exact `[09:30:00,09:35:00)` New York—including warm-up, 301 snapshot encodes, suffix validation, and shutdown—must finish within one explicit 15-minute command bound on the current host. Record segment durations; do not convert them into an SLA. |

## 4. Correction rationale and implementation constraints

The accepted C12 wrapper deliberately imposed cumulative 1x pacing and retained
HTTP/dashboard state. That behavior conflicts with the current CLI-only,
maximum-speed requirement. Independently, the retained-day diagnosis found:

- an upper scale above 87 billion symbol visits because a full-population
  evaluator ran after nearly every changed aggregate;
- three complete artifact passes because playback construction rescanned an
  already validated file; and
- no completed retained-day C12 acceptance evidence.

An uncommitted patch on `codex/replay-ui-mvp` is evidence only. Its shared
opaque validator, group-boundary coalescing, and compact equivalence tests may
be adapted. Its 1x schedule, API/UI retention, replay presentation, and any
`ReplayObservationStart`, fast-forward evaluator/qualification/feature branch,
or other alternate observation-start semantics are rejected. Preserve
unrelated live-scanner work exactly.

Invalid states prevented by construction: replay and live composition cannot
coexist; the artifact date is the only date identity; only C4 can construct
start/group/end evidence; only the engine can mutate market state; stdout sees
no pre-start snapshot; and no HTTP/UI object exists in replay mode.

Runtime rejection remains required for malformed/mutated artifacts, cache or
binding mismatch, invalid bounds, rejected dispositions, nonmonotonic logical
time, accounting contradiction, output failure, cancellation, and shutdown
timeout.

## 5. Primary proofs and acceptance

1. `P-C12R-CONFIG`: command matrix proves replay never reads credentials or
   constructs live, hydration, checkpoint, API, or UI work; all contradictory
   or invalid inputs fail before engine mutation.
2. `P-C12R-EQUIV`: on a compact dense/quiet/no-print fixture with `D=4s`, run
   the ordinary baseline and optimized unpaced driver. At every group from `S`
   through `O1`, compare ordinary evaluation/publication identity; at every
   emitted boundary `O0..O1`, compare canonical records, absent evidence,
   qualification latch, features, ranking/order, population accounting,
   committed `T`, record/group dispositions, and terminal identity. The
   optimized run emits nothing before `O0` and emits exactly
   `seconds(O1-O0)+1` snapshots.
3. `P-C12R-GROUP`: many same-second symbols prove group-boundary batching is
   identical to the baseline at observable boundaries; a partial/correction
   artifact proves the optimization cannot enter that path.
4. `P-C12R-OUTPUT`: golden and mutation cases prove the closed envelope,
   inclusive boundary times/count, 1 MiB limit, snapshot/result separation,
   replay-specific T/Q unavailability, and joined nonzero behavior on encoding
   failure without a success result.
5. `P-C12R-TRUST`: retain noncanonical/unknown-field/signed-zero, ordinal/order,
   truncation/extra-byte, digest/seal, mutation-before-playback, prefix
   mutation, and suffix mutation cases. Instrument construction to prove no
   redundant full scan.
6. `P-C12R-B4`: once, without credentials/provider calls or path/identity/row
   persistence, validate the retained manifest and run exact
   `[09:30:00,09:35:00)` New York fully unpaced under 15 minutes. Consume the
   intentional row output through an ephemeral private `0600` sink; do not
   echo rows into test/task logs or commit/preserve the sink. Record validation,
   construction, hidden warm-up, observation encoding, suffix finalization,
   exact accounting, and joined exit.

Run narrow proofs during correction, then `go test -short -timeout 2m ./...`,
focused race with an explicit five-minute timeout for changed Go packages,
`go vet ./...`, and `git diff --check`. Existing unrelated dirty-tree failures
must be isolated and reported, never hidden or folded into replay acceptance.

The persisted/order/engine batching boundary is accepted from its allocated
deterministic trust, ordering, batching, race, and retained-day proofs. Do not
invoke a reviewer solely because a lower-component claim was reopened or C12
is complete. Accept only after the retained-day proof passes; one day
establishes functional and local performance evidence, not market generality.

## 6. Delivery and Git

Keep one write-capable slice: first record the contract/ledger correction;
then correct group-boundary replay evaluation; then remove the redundant
construction scan; then replace the C12 wrapper with the finite CLI; finally
run retained-day acceptance. The group change must reopen the
exact C2 engine-transition and C3 evaluator/publication claims in their parent
ledgers; the validation change must reopen the exact C4 persisted-artifact/
playback-construction claim in the C4 parent. Preserve unaffected proofs,
rerun the narrow reacceptance proof, and return each component claim to accepted
when its proof and conformance record are clean. C12 must not silently own
these lower-component semantics.

The current worktree contains mixed uncommitted live and replay changes. Do not
branch in place, stage them together, or commit unrelated work. Create an
isolated worktree and `codex/replay-cli-event-time` branch from commit
`28281df` unless a later clean replay milestone is demonstrably safer. Inspect
the current replay diff as read-only evidence and port only reviewed replay
changes. The first isolated-worktree milestone applies and reviews only this
correction document, its goal handoff, and the replay-specific C12 parent/map
hunks; exclude the unrelated live-scanner map hunk and commit the correction
plan before code. Make local coherent commits; do not push, rebase, amend, rewrite
history, delete branches, or use destructive reset operations.
