# Replay UI MVP backend performance fix

## 1. Outcome and non-scope

A developer selects a five-minute interval from an existing complete aggregate artifact and exact cached reference data, starts the loopback API, fast-forwards from session start to the observation start in minutes, and sees ordinary changing one-second dashboard publications for the selected interval at approximately 1x wall pace.

This is a correction to the local visualization harness. It adds no replay product, browser controls, provider access, artifact/schema version, cache, database, service, ranking rule, or second engine/evaluator. It neither skips records nor treats prior validation as permanent trust.

## 2. Current root causes with exact functions

Validation performs a header-only `ProbeCandidateHeader`, then three complete artifact passes. `replayartifact.OpenValidatedContext` calls `validateOpenFile` for pre-engine binding/schema/canonical/order/coverage/count/digest trust. `Source.Start` calls `Handle.BeginPlaybackThroughContext`; `playback.NewContext` then drives a first cursor through every group and `EndContext` only to validate the same open file and learn the record count. Its returned second cursor is the third pass: `Source.Step` streams the applied prefix and `Cursor.RequestedEndContext` validates the unapplied suffix through summary/seal. The first and streaming passes are required; the middle construction pass is redundant.

Per aggregate, `ReadBytes` allocates a line; `lineKind`/`kindOf` separately decode JSON; strict decoding allocates a decoder/reader and parses three timestamps; canonical comparison re-encodes JSON. `strictCanonicalAggregateLine` re-encodes an aggregate twice. Both aggregate encoders repeatedly construct `bytes.Buffer`/`json.Encoder` instances for individual strings. A 512-symbol synthetic profile attributed 43,692 allocations to `replayartifact.appendJSONString` and 23,894 to `playback.appendJSONString`; fixture TLS/download work prevents treating those shares as retained-artifact percentages. The retained artifact previously reopened in 1:32.697 and the latest observed validation was about three minutes, consistent with CPU/allocation sensitivity rather than disk alone.

Warm-up is worse than the prior “one snapshot per second” diagnosis. The exact path is `Runtime.run` -> `Source.Step` -> `Cursor.NextRecordContext` -> `Engine.AdmitReplayRecord` -> `Engine.transition` -> `runAggregateFeatureContributorLocked` -> `stageAggregateEvaluationAtLocked` -> `runAggregateEvaluatorLocked` -> `applyAggregateCandidateLocked` -> private publication, repeated serially for each changed record once `committedT` exists. Each replay group then calls `AdmitReplayGroup` -> `applyReplayGroupLocked`/`classifyReplaySlotLocked` -> timer contributor/evaluator/apply/publication. With 7,671,171 records and 5,691 symbols, record-triggered staging plus apply admits an algorithmic upper scale above 87 billion symbol visits, before timer work. `Runtime.publish` also calls `ReplayRuntime.PublishReplay` after each warm-up group, but it copies at most 20 ranking/TQ rows, not 5,691 symbols. `CaptureSnapshot` and API mapping occur only on HTTP polling; status JSON is emitted only when phase changes. Record parsing/admission and per-second slot classification remain material, but full-population record-triggered evaluation is the dominant defect.

## 3. Recommended minimal design for validation

Make `playback` own one shared full validator. `playback.ValidateContext(file, plan)` performs the current pre-engine scan and returns an opaque `ValidatedArtifact` whose fields bind the exact `*os.File`, plan/identity, total records, size, and modification time. `OpenValidatedContext` stores that evidence in its private `Handle` and derives `Metadata` through read-only accessors. `BeginPlaybackThroughContext` passes the evidence to `playback.NewContext`; after exact file/plan/stat checks and rewind, `NewContext` creates the streaming cursor directly instead of scanning.

The streaming cursor still decodes and canonical-compares every line, enforces ordinal/group/order/coverage/count/byte budgets, hashes header through summary, matches the original artifact ID and record count, requires the seal to be final, and compares final size/modification time. `RequestedEndContext` still reads the entire suffix without exposing suffix records to the engine. Thus same-path replacement is blocked by the retained open descriptor, and same-file mutation/corruption is detected by canonical/order/reconciliation/digest/stat checks.

Use one line-kind dispatch and one strict aggregate decode plus one normalized canonical re-encode. Reuse a record buffer/encoder where safe; do not replace exact canonical-byte comparison with semantic JSON equality.

## 4. Recommended minimal design for warm-up

Pass `observationStart` as a replay-only fast-forward boundary when `operations.NewReplay` constructs the engine. Restrict optimization to validated `complete_final_bars`; partial/correction-capable replay retains its existing transition behavior.

For every complete-artifact record, continue parsing, hashing, ordinal/order validation, engine admission, canonical merge/disposition, local mutable-block maintenance, accounting, and the disposition wait. Do not stage/apply a population evaluation or replace the private publication for that record: a final-bar record delivered in group `G` cannot change the already committed target before `G`. The group remains the sole evaluation boundary.

For each group before `observationStart`, continue simulated-clock advance, group proof, all-symbol present/absent slot classification, timer/lifecycle/target semantics, compaction, Activity maintenance, and qualification advancement through the ordinary delayed target. Suppress the full ranking/accounting projection, feature-result materialization, committed-watermark application, and private/API publication. At the `observationStart` group, run the existing ordinary contributor, evaluator, candidate apply, and publication once; this produces the first observation snapshot. Every later group uses that same ordinary path, so one-second ranking publications resume at `observationStart` and continue through `observationEnd`.

Coalesce warming `PublishReplay` calls to at most once per wall-clock second, plus the initial warming capture and mandatory observation-start capture. This is the bounded UI/status progress signal. It is not market state and cannot change pacing or engine facts.

## 5. Exact files/functions expected to change

- `internal/replayartifact/validate.go`: `Handle`, `OpenValidatedContext`, `BeginPlaybackThroughContext`.
- `internal/replayartifact/playback/playback.go`: shared validator/opaque evidence, `NewContext`, line dispatch/canonical aggregate path, streaming end checks.
- `internal/engine/engine.go`, `replay.go`, `feature_price_range.go`, `evaluator.go`: replay-only boundary, record coalescing, fast-forward maintenance, and ordinary boundary resumption.
- `internal/operations/replay.go` and `internal/replaymode/runtime.go`: pass the boundary and rate-limit warming captures.
- Focused tests in `internal/replayartifact/artifact_test.go`, `internal/replay/replay_test.go`, `internal/engine/*_test.go`, `internal/replaymode/runtime_test.go`, and existing UI model tests.

## 6. Construction invariants

One validated same-open file, cursor, simulated clock, engine, canonical symbol state, evaluator, committed watermark, and publication cell remain authoritative. Every prefix record receives exactly one engine disposition; suffix records receive none. Complete no-print coverage, identity/correction rules, half-open windows, four-second evaluation delay, qualification latches, feature formulas, exact ranking, and requested-end accounting remain unchanged. No fast-forward snapshot may claim current ranking. The first observation snapshot must equal the unoptimized semantic state—canonical records, coverage, fields, qualification status/final latch, ranking/accounting, committed time, and logical time—allowing only operational publication IDs/counters to differ.

## 7. Focused tests and one practical acceptance measurement

Retain/regress noncanonical/unknown-field/signed-zero aggregates, wrong ordinal/order, wrong binding or incomplete coverage/count, truncation/extra bytes, digest/seal mismatch, mutation before playback, mutation during prefix, and mutated requested-end suffix. Prove playback construction performs no full scan after opaque validation.

Add a compact dense/quiet/no-print fixture that runs baseline and fast-forward modes with the production delay and compares the first observation semantic snapshot exactly. Assert every record/group disposition and terminal accounting match. Assert at least three consecutive observation publications change through the API/UI model and warm-up progress never exceeds 1 Hz.

After stabilization, run the retained 2026-08-07 artifact once for 04:00->09:30 ET and report five segments. On the current host the local acceptance envelope is: initial validation <=3:00; playback construction <=0:05; warm-up <=3:00; five-minute observation 5:00-5:15 with at least three changing publications and no cumulative schedule failure; requested-end finalization <=3:00. This preserves the prior 15-minute bounded local workflow while replacing the observed ~27-hour warm-up. Also run `go test -short -timeout 2m ./...` and `git diff --check`.

## 8. Explicitly deferred alternatives

Defer binary artifacts, sidecar indexes, persistent validation caches, parallel/out-of-order engine admission, incremental-ranking redesign, browser calculation, checkpoints for seeking, and generalized evaluator batching. Reconsider only if the single retained-artifact measurement misses the envelope after this correction.
