# Live bootstrap integrity, suppression visibility, and operator output correction

**Status:** Reopened — deterministic containment complete; awaiting one typed
current-date live diagnostic observation and the root invariant correction it
may identify

**Authorized:** 2026-08-11

**Outcome:** Diagnose and correct the post-04:00 fresh-bootstrap integrity
failure without weakening evaluator invariants; keep the actual suppressed state
servable to the UI; replace default one-second stdout JSON with concise
operator-oriented progress; prove the correction offline before a coordinated
live observation.

This is one bounded cross-component correction, not a new component or a new
state owner. It reopens only the claims invalidated by the 2026-08-11 live
observation. Unaffected C2/C3/C6/C8/C10/C11 evidence remains reusable.
When implementation begins, this earlier-boundary correction becomes the one
active slice; pause later C12 implementation until it is accepted. Preserve
the current uncommitted C12 work and do not mix it into correction commits.

## 1. Why this correction exists

A fresh live start after 04:00 ET connected and hydrated successfully but failed
at the bootstrap reconciliation boundary:

- the Massive adapter connected on its first attempt and the aggregate
  subscription was acknowledged;
- all `5,540` hydration requests became terminal: `1,742` value and `3,798`
  empty, with zero failed/canceled/fenced work;
- `68,154` rows reconciled, including `24` REST/live conflicts or withdrawals;
- the ingress fence reconciled and recorded `supported_through=09:56:12Z`;
- the last valid UI watermark was `09:56:11Z`;
- the engine then recorded one integrity failure, entered
  `suppressed/accounting_integrity/restart_required`, and sealed at engine
  sequence `10,725`; and
- the provider connection remained active and live frames continued arriving.

The engine currently discards the specific error returned by aggregate
evaluation validation and reports the broad `accounting_integrity` cause. The
suppression cell then has publication ID zero, blank binding/date, and zero
generation time. Component 10 rejects that cell, returns `503
snapshot_unavailable`, and Component 11 freezes the previous response as local
API transport `disconnected`. Readiness also checks blank sentinel binding
before suppression, producing the misleading `binding_mismatch` reason.

There are therefore two defects to correct:

1. the real fence/evaluator invariant failure; and
2. loss and misclassification of the resulting suppression state across
   engine, readiness, API, and UI.

## 2. Controlling behavior

The correction preserves these existing requirements:

- `PG-OPS-02`: terminal hydration must return to ordinary evaluation rather
  than freezing accepted live work;
- `PG-OBS-03`: aggregate transport, backend readiness, ranking currentness, and
  field/T/Q currentness remain distinct;
- `ARCH-OWN-01`, `ARCH-OWN-03`, and `ARCH-FLOW-02`: one engine owner, one
  immutable publication boundary, and no silently lost accepted work;
- `DTE-RECOVERY-02` through `DTE-RECOVERY-05`: live catch-up, exact ingress
  fencing, empty terminal success, and local containment where possible;
- `LIFE-HYDRATE-03` through `LIFE-HYDRATE-06`: the live tail remains active,
  terminal hydration reconciles through the fence, and completion leaves
  `hydrating` exactly once;
- `LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`: genuine global integrity
  failure remains explicit and fail-closed;
- `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`: process liveness is distinct and
  legal terminal startup returns to ordinary publication; and
- the accepted C3 population/evaluator, C6 merge/fence/start, C8 readiness and
  measurements, C10 coherent snapshot, and C11 disconnected/suppressed display
  contracts.

No validation rule may be deleted, relaxed, or converted to an ordinary
unknown state merely to make the failing input pass.

## 3. Scope and non-scope

### In scope

- typed, bounded evaluator-integrity diagnostics that survive suppression;
- deterministic reproduction of fresh hydration plus concurrent live-tail
  merge and final ingress fence;
- direct correction of the evidenced invariant violation;
- a valid, coherent suppressed snapshot for installed-binding integrity
  failures;
- correct readiness and UI distinction between provider connection and local
  API transport;
- concise change-aware live-mode stdout, terminal/warning stderr, and an
  optional private one-second NDJSON diagnostics file;
- ordinary, focused race, cached-artifact, UI, and coordinated live proof.

### Not in scope

- changing qualification, ranking, feature, merge, watermark, or time-window
  semantics;
- treating REST/live conflict as equality or silently selecting a winner;
- adding another evaluator, publication owner, event bus, database, general
  recorder, or logging framework;
- persisting or restoring a half-reconciled hydration generation as a normal
  checkpoint;
- embedding the local licensed artifact or provider payloads in the repository;
- changing T/Q behavior except for preserving its existing independent status;
- public deployment, provider-capacity claims, or automatic credential use.

## 4. Required execution order

Implementation is sequential. Do not start with a guessed market-state fix.

### Slice 1 — expose the failed invariant and reproduce it offline

#### 4.1 Typed evaluator failure

Replace the boolean evaluator-validation result with one closed diagnostic
result. The exact internal type is discretionary, but it must preserve at least
these bounded categories:

```text
candidate_target_mismatch
support_contradiction
population_accounting
qualification_accounting
uncertainty_accounting
feature_accounting
ranking_projection
ranking_row
tq_intent
unknown_evaluator_integrity
```

The diagnostic records:

- validation category;
- transition input kind and engine sequence;
- candidate time and expected time;
- lifecycle and hydration generation/fence identity;
- the relevant fixed-cardinality totals;
- when the violation is symbol-local in construction, only the first bounded
  symbol and field/reason tuple; and
- no raw provider payload, URL, credential, stack dump, unbounded symbol list,
  or mutable map.

The engine latches the first terminal integrity diagnostic before entering
suppression. Later closed admissions cannot overwrite it. It is visible in the
operational metrics/optional NDJSON and immediate stderr failure line. It does
not become another readiness input and does not replace the existing lifecycle
reason or suppression disposition.

Expose the same bounded value through the V1 snapshot as an optional additive
`operations.integrity_failure` object. It is absent before failure. At minimum
it contains category, engine sequence, candidate/expected time, and the optional
first field/symbol tuple. This is a compatible optional C10 addition, not a new
schema version or an invitation to add arbitrary diagnostic maps.

Existing evaluator validation remains the authority. Refactor it only enough
to return a typed reason instead of discarding its error.

#### 4.2 Offline reproduction ladder

Use the first level that reproduces the failure. Preserve the failing fixture
before correcting code.

1. **Small deterministic interleaving.** Exercise the real engine hydration
   plan/chunk/terminal/fence path with a small binding. Admit live aggregates
   during hydration, include equal REST/live duplicates, unequal conflicts,
   value results, successful empty results, quiet symbols, and a fence target
   one second beyond the last valid publication. Assert either the observed
   typed failure or clean legal completion.
2. **Observed-shape synthetic scale.** If the small case is clean, generate a
   deterministic `5,540`-symbol binding and reproduce the observed work bins,
   concurrent live overlay, `24` conflict/withdrawal rows, terminal ordering
   permutations, and final fence. Use bounded generated values; do not copy
   licensed rows into source control.
3. **Existing cached aggregate artifact.** If shape-only evidence remains
   clean, run an explicit non-short local diagnostic using:

   ```text
   var/aggregate-replay/aggregate-replay-fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a.jsonl
   var/reference/universe/2026-08-07.json
   var/reference/prior-close/2026-08-06.json
   ```

   The artifact is a sealed `complete_final_bars` 2026-08-07 session with
   `5,691` symbols and `7,671,171` normalized aggregate records. Validate it
   through the existing replay-artifact reader and exact reference binding.
   Reuse a bounded session prefix as hydration facts, add a deterministic live
   overlay before selected historical facts, and complete the ordinary
   hydration ingress fence. Do not introduce another JSONL parser or REST
   normalizer. The harness must be skipped by `testing.Short()`, accept paths
   explicitly rather than hard-code developer home paths, and have a maximum
   15-minute command timeout.

The cached artifact proves engine/hydration behavior at realistic population
and aggregate density without another provider request. It does not prove the
2026-08-11 binding or real live arrival chronology.

#### 4.3 Slice 1 gate

Before correcting behavior, record:

- the smallest reproducing case, exact typed diagnostic, and offending
  invariant; or
- if all offline levels are clean, the exact coverage gap that requires one
  live diagnostic observation.

Do not weaken containment when offline reproduction fails. A clean offline
result means only that provider chronology or the exact current-date data is
still material.

### Slice 2 — correct behavior, suppression visibility, and operator output

#### 4.4 Root correction

Apply the narrowest change that makes the reproducer satisfy the existing
invariant. Add a regression test that fails on the pre-correction behavior and
passes only when:

- all work and row accounting still reconcile;
- equal REST/live evidence deduplicates and unequal evidence retains the
  accepted conflict/unknown consequence;
- the exact fence target is supported before the watermark advances;
- successful empty results remain terminal and do not fabricate marks;
- the engine leaves `hydrating` for `live` on valid evidence;
- ranking is `qualified_current`, permitted `degraded_bootstrap`, or honestly
  unavailable according to existing predicates; and
- no integrity diagnostic is latched on the valid case.

A test that merely changes the expected suppression reason is not a root fix.

#### 4.5 Suppression remains observable

For an installed binding, evaluator/accounting integrity failure must produce a
coherent terminal suppression observation rather than an unservable snapshot.
The implementation may revise the private terminal-publication construction,
but the externally observable result is fixed:

- `/livez` remains `200` while the process is live;
- `/readyz` is `503` with reason `suppressed` and the binding/publication
  identity when available;
- `/api/v1/snapshot` remains `200` with a schema-valid snapshot containing
  lifecycle `suppressed`, the exact lifecycle reason, `restart_required`, no
  current ranking claim, no current rows, and the latched integrity diagnostic;
- readiness selects `suppressed` before any derivative sentinel mismatch, so
  it cannot report a fabricated `binding_mismatch`;
- the UI's API transport remains `connected` while the API is reachable;
- the UI prominently shows the exact suppressed reason and restart requirement;
  and
- provider aggregate connection state remains separately visible and may still
  say active/acknowledged until shutdown drains it.

Do not reuse a prior current row set inside the new suppressed publication. A
browser may retain the previous response only on a real HTTP/schema failure,
where it remains explicitly frozen.

Use the simplest private representation that preserves one sealed capture and
the existing public schema. Do not make the API reconstruct market state from
mutable engine maps.

#### 4.6 Live-mode terminal output

Replace the unconditional one-second `Status + Metrics` JSON on live-mode
stdout with a small change-aware operator renderer. Replay-mode output is
unchanged unless a directly affected shared helper requires a mechanical
adaptation.

During fresh/checkpoint/gap hydration, print this three-line shape:

```text
Warm-up 3,481 / 5,540 · 62.8%
values 1,064 · empty 2,417 · open 2,059 · failed 0 · canceled 0 · fenced 0
aggregate live connected/acknowledged · final fence pending
```

Definitions are exact:

```text
terminal = completed_value + completed_empty + failed + canceled + fenced
percent  = 100 * terminal / planned
open     = planned - terminal
```

Use `empty`, not `no-print`, during warm-up: `completed_empty` is successful
provider work, while publication-level `no_print_through_T` is established only
after reconciliation with live evidence and the final fence.

Print immediately when lifecycle, readiness, aggregate connection/acknowledgement,
hydration purpose/generation, fence state, suppression, or recovery outcome
changes. Otherwise print at most every five seconds while warming/recovering and
every ten seconds while ready. Do not print a new line solely because a
one-second sample ID changed.

Once ready, print one concise line containing:

```text
Ready · <ranking mode> · <N> ranked · watermark <ET time> · lag <ms> · aggregate live connected
```

Warnings and terminal failures go to stderr immediately. An integrity failure
includes its typed category, lifecycle reason, engine sequence, candidate/fence
time, and the instruction that restart is required. It must not call an active
Massive connection a disconnected transport.

#### 4.7 Optional private diagnostics

Add one explicit live-mode option such as:

```text
--diagnostic-log /absolute/private/path/scanner.ndjson
```

When absent, full one-second NDJSON is disabled. When present:

- emit the existing full `Status + Metrics` sample once per second;
- create/append only to an explicitly selected regular local file with mode
  `0600` and reject unsafe group/other permissions or a non-regular target;
- never include credentials, authorization headers, raw provider frames, or
  REST URLs containing secrets;
- fail startup if the explicitly requested file cannot be opened safely;
- after startup, warn once and disable diagnostic writing on a write failure
  rather than stopping market processing; and
- close and join the writer during ordinary scanner shutdown.

This is a bounded direct writer, not an asynchronous logging subsystem. The
one-second sample is already fixed-cardinality and may be encoded synchronously
from the command's reporting loop.

## 5. Checkpoint and cached-data policy

Do not add checkpoints for half-reconciled hydration. The failed observation
had `InstalledCheckpoint=false`, `Submitted=0`, and `Completed=0`; no valid
checkpoint can accelerate its rerun.

A prior checkpoint may be used for a separate restart proof only when the
existing store validates the exact trading date and binding identity. It does
not replace the required fresh-bootstrap regression because this defect
occurred before the first checkpoint-eligible live boundary.

Prefer the existing 2026-08-07 aggregate artifact for scale diagnosis before
new REST acquisition. Any derived local diagnostic file remains under ignored
`var/`, contains no credentials, and is neither committed nor described as a
current-market or live-transport fixture.

## 6. Allowed implementation boundary

The correction may modify only the narrow paths needed under:

```text
internal/engine                 evaluator diagnostic, root fix, terminal publication, tests
internal/operations             readiness/metrics propagation and tests
internal/snapshotapi            coherent suppressed mapping/HTTP proof
cmd/scanner                     operator output and optional diagnostics
ui                              suppressed/connected presentation and tests
internal/replayartifact         test-only reuse seam only if existing public reading is insufficient
docs                            correction ledger/results
```

Avoid production changes to replay-artifact semantics. Do not inspect the V2
predecessor unless the offline reproducer exposes a precise unanswered provider
mechanics question and this document is first amended with that narrow source
scope.

## 7. Primary proofs and verification

### P-LIVE-BOOTSTRAP-INTEGRITY

The smallest reproducing hydration/live/fence trace proves the corrected
invariant and retains the pre-fix counterexample. It must assert the exact
population, work, row, conflict, fence, lifecycle, watermark, ranking, and
diagnostic outcomes. If only the cached artifact reproduces, retain a compact
synthetic regression distilled from the typed failure; do not make ordinary
tests depend on the 2.58 GB local file.

### P-SUPPRESSION-VISIBLE

Induce the actual engine evaluator/accounting failure rather than mutating a
normal mapper fixture. Capture through the real operations and snapshot API
path. Assert `livez`, `readyz`, snapshot, readiness reason, API transport, UI
band, empty current rows, binding identity, and exact failure diagnostic.

### P-OPERATOR-OUTPUT

With injected clock/samples and in-memory writers, prove:

- exact progress arithmetic and one-decimal percentage;
- immediate lifecycle/fence/failure output;
- five-/ten-second rate limits and change suppression;
- stderr routing;
- no default NDJSON on stdout;
- exact one-second NDJSON when an explicit safe diagnostic file is configured;
  and
- unsafe path/permissions and post-open write-failure behavior.

### Commands and bounds

During correction, run the narrowest affected proof after each change. Before
the live gate, require:

```text
go test -short -timeout 2m ./...
go test -race -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi
node --test ui/*.test.mjs
go vet ./...
```

Run the cached-artifact diagnostic explicitly with its artifact/reference paths
and a command timeout no greater than 15 minutes. Record artifact identity,
binding identity, prefix interval, symbol/record counts, synthetic live overlay
and conflict counts, runtime, and result. Do not rerun the 2.58 GB proof after
unrelated presentation-only edits.

Because this correction crosses canonical integrity containment and the public
snapshot boundary, perform one final read-only review after all deterministic
proofs pass. The reviewer checks that validation was not weakened, the engine
remains the sole state owner, suppression cannot appear current, and the
emergency snapshot cannot fabricate binding or market state.

## 8. Coordinated live validation

Live validation is the final step, not a discovery shortcut.

Before making any credentialed provider request, the implementing agent must:

1. finish both slices and the final deterministic review;
2. report the root cause, exact correction, cached/offline results, expected
   warm-up duration, command, diagnostic-log path, and UI/API addresses;
3. tell the owner that the scanner is ready for the live observation so the
   owner can open the UI; and
4. obtain/confirm authorization for that exact run and trading date under
   `docs/market-hours-validation.md`.

The live observation starts after 04:00 ET and exercises fresh bootstrap unless
the owner explicitly selects a separate checkpoint-restart scenario. During
the run, observe without editing the active process:

- concise warm-up progress and correct terminal arithmetic;
- active/acknowledged aggregate connection while REST work runs;
- live-tail deliveries during hydration;
- terminal work and exact ingress-fence transition;
- no evaluator-integrity diagnostic on valid evidence;
- transition to `live` with an advancing watermark;
- schema-valid API snapshots throughout;
- UI API transport remaining connected while the API is reachable;
- qualified or permitted degraded ranking and independently honest T/Q; and
- checkpoint submission only after a coherent eligible live boundary.

Success requires the UI to remain observable through warm-up and fence
completion, with no `accounting_integrity`, false `binding_mismatch`, or false
provider-disconnected presentation. If the run fails, stop after preserving the
private NDJSON and typed failure record; do not perform another live retry until
the new evidence is converted into a deterministic regression or this plan is
revised.

## 9. Completion record

The implementing agent updates this document with:

- exact root cause and why the former tests missed it;
- reproducer and diagnostic category;
- files/behavior changed;
- offline proof results and limitations;
- suppression/API/UI proof result;
- operator-output examples;
- final review result;
- coordinated live observation record, if authorized; and
- remaining uncertainty, especially anything not proven about provider
  chronology or future trading dates.

This correction is complete only when the root invariant passes deterministic
proof, actual suppression remains visible rather than becoming a local API
disconnect, default stdout is operator-oriented, and the authorized live run
passes. A new typed live failure is useful correction evidence but keeps this
plan reopened until that evidence is corrected and the live gate passes.

### 9.1 Implementation record — 2026-08-11

The three-level offline ladder did not reproduce the observed
`accounting_integrity` failure. This is a material result, not proof that the
live defect is fixed:

- the small production-path trace completed cleanly with six symbols, value
  and successful-empty work, equal REST/live duplication, unequal overlap, a
  live-only tail, and a fence one second beyond the prior publication;
- the observed-shape test completed cleanly with `5,540` symbols, `1,742`
  value results, `3,798` empty results, `68,154` hydration rows, `24`
  conflicts/withdrawals, and the final fence; and
- the retained-artifact proof validated SHA-256
  `e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189`,
  artifact binding
  `session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d`,
  all `7,671,171` artifact records and `5,691` symbols, then applied the
  valid-prior population's `68,104` records from
  `2026-08-07T08:00:00Z` through `2026-08-07T09:14:10Z` with `25` live
  overlay facts and `24` conflicts. It transitioned to `live`, reconciled the
  fence, and latched no evaluator diagnostic in `161.01s`, without REST.

The exact remaining coverage gap is the 2026-08-11 binding and/or actual
provider arrival chronology. Because no deterministic counterexample exists,
no guessed market-state rule was changed and no evaluator validation was
weakened. The root-invariant portion of Slice 2 remains open until one
authorized live observation either completes cleanly or supplies a typed
counterexample that can be distilled into the required regression.

Implemented containment and observation behavior:

- evaluator validation now returns one of the closed diagnostic categories and
  latches the first fixed-cardinality failure before suppression;
- installed-binding evaluator failure seals a noncurrent suppressed
  publication with no ranking rows; readiness reports `suppressed` before any
  derivative binding check; and the same diagnostic remains available through
  metrics and the snapshot API;
- the UI keeps API transport `connected` and displays a prominent exact
  suppression/restart band instead of freezing the prior response as a local
  disconnect; and
- live stdout is change-aware operator output. Full one-second `Status +
  Metrics` NDJSON is disabled by default and available only through an explicit
  absolute regular `0600` `--diagnostic-log` target. Unsafe targets fail
  startup; later write failure warns once and disables only diagnostics.

Representative operator output is:

```text
Warm-up 3,481 / 5,540 · 62.8%
values 1,064 · empty 2,417 · open 2,059 · failed 0 · canceled 0 · fenced 0
aggregate live connected/acknowledged · final fence pending
Ready · qualified_current · 17 ranked · watermark 09:30:56 EDT · lag 125ms · aggregate live connected
```

The deterministic suppression/API/UI and operator-output proofs pass.
`go test -short -timeout 2m ./...`, `node --test ui/*.test.mjs`, `go vet
./...`, and the correction-focused `go test -race -short -timeout 5m
./internal/engine ./internal/operations ./internal/snapshotapi` pass. The exact
non-short race command is not yet a clean gate: the correction's 5,540-symbol
engine diagnostic passes under race in `77.90s`, but the pre-existing C8
6,000-symbol controlled-load test reaches the five-minute package timeout
inside fence evaluation with the preserved paused C12 feature work present.
No C12 performance code was changed under this earlier-boundary correction.
This distinction must remain visible in the final review and before live-run
authorization.

The required read-only review initially found three correction-local gaps: a
symbol-local support contradiction retained its category but not the offending
symbol, initial stdout failure bypassed joined shutdown, and hydration failure
counters were neither immediate change inputs nor stderr warnings. The
correction now retains the first deterministic symbol plus exact support
reason through engine/API/private NDJSON, joins runtime/API shutdown on initial
output failure, and emits generation-scoped nonrepeating stderr on increases
in failed/fenced/row-integrity counts while treating cancellation as an
immediate stdout change only. Focused re-review found all three resolved and
no new finding in scope.

The implementation is therefore ready for one separately authorized
current-date **diagnostic** fresh-bootstrap observation, not a claim that the
root invariant or scanner live readiness is already proven. For trading date
`2026-08-11`, the proposed exact local paths are:

```text
go run ./cmd/scanner --run-mode live --trading-date 2026-08-11 --api-address 127.0.0.1:8080 --allow-origin http://127.0.0.1:4173 --diagnostic-log /Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/live-bootstrap-2026-08-11.ndjson
go run ./cmd/dashboard --address 127.0.0.1:4173 --api-origin http://127.0.0.1:8080 --assets ui
```

The UI is `http://127.0.0.1:4173`; liveness, readiness, and snapshot endpoints
are `http://127.0.0.1:8080/livez`, `/readyz`, and `/api/v1/snapshot`. The prior
failed observation did not retain a start timestamp, so it does not support a
precise expected warm-up duration. Coordinate a 15-minute observation window,
report actual warm-up duration from the change-aware output, and stop after
the first typed failure rather than retrying. Execution still requires the
owner's explicit confirmation of this trading date, credential/account
context, and duration.

No credentialed request has been made under this correction, and the
pre-existing uncommitted C12 work remains preserved and paused.
