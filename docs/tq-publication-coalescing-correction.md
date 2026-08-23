# T/Q publication coalescing correction

**Status:** Owner-directed, implementation-ready specification. This is step 2
of the scanner-stability correction sequence and starts only after the live
evaluation-cycle correction is offline complete.

**Development constraint:** The market is closed. Use deterministic T/Q traces
and production-shape load; do not access credentials or make provider requests.
Live confirmation is deferred to the single combined market-hours soak after
step 3.

**Implementer starting context:** Read this specification, the accepted/offline
completed [step-1 correction](live-evaluation-cycle-coalescing-correction.md),
and the accepted
[`T/Q data-confirmed subscription correction`](tq-data-confirmed-subscription-correction.md).
Preserve unrelated worktree changes and use one write-capable implementation
agent.

## Outcome

Every admitted trade, quote, T/Q drop, command result, and pressure fact must
continue to mutate or classify canonical T/Q state synchronously in engine
sequence order. Those facts must no longer rebuild and validate the complete
global immutable scanner publication one by one.

Ordinary high-rate T/Q mutations are accumulated into the next already-required
one-second publication boundary. Coverage loss, first data confirmation,
quarantine, retention-bound, membership, and pressure-mode transitions remain
immediately visible because they change whether a displayed field can be
trusted or whether T/Q is being shed.

The result should reduce global publication rate from market-event rate to
approximately aggregate evaluation cadence plus a small number of meaningful
T/Q transitions, while retaining one atomic aggregate-and-T/Q publication.

## Current brittleness

`tqPublicationInput` currently includes timers, T/Q command results,
quarantine, trades, quotes, drops, pressure results, and pressure ticks. Each
such input increments `tq.revision`. That revision participates in the global
publication fingerprint, so even an ordinary trade or quote can clone the
aggregate evaluation, clone the complete T/Q view, validate the combined
publication, allocate a new publication ID, and replace the atomic cell.

During the observed live run, publication IDs advanced at roughly 127 per
second while the dashboard requested one snapshot per second. This work has no
consumer-visible cadence benefit and increases allocator/GC pressure on the
same serialized engine path that must keep the aggregate watermark current.

## Fixed boundaries

Preserve all of the following:

- the engine is the only mutable T/Q owner;
- every T/Q input receives its exact existing applied/rejected/duplicate/
  fenced/pressure-shed disposition and accounting in engine order;
- T/Q coverage is per symbol and channel, begins only from valid current-
  generation post-boundary data, and closes immediately on a gap;
- Tape Speed and Spread formulas, windows, stale rules, retention bounds, and
  data-confirmed membership semantics;
- all displayed rows are still selected only by aggregate qualification and
  Day-% ordering;
- pressure and T/Q failure never gate aggregate readiness or ranking;
- one combined immutable publication and one atomic publication store; and
- publication IDs remain strictly increasing and independent of watermark.

The controlling meanings are `PG-AVAIL-01`, `PG-AVAIL-03`, `PG-TAQ-01`
through `PG-TAQ-03`, `PG-OBS-03`, `ARCH-OWN-01` through `ARCH-OWN-04`,
`DTE-TQ-01` through `DTE-TQ-03`, `DTE-MERGE-05`, `LIFE-TQ-01` through
`LIFE-TQ-03`, and `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`. The accepted
data-confirmed subscription correction remains authoritative.

## Required design

### 1. Separate canonical mutation from public projection

Maintain two concepts inside the existing engine-owned T/Q state:

- **canonical mutation progress:** changes for every classified T/Q fact and
  supports exact internal/accounting behavior; and
- **public T/Q projection dirty/revision state:** changes only when a combined
  immutable publication must be replaced.

Names and representation are implementation discretion. A pending bit plus a
public projection revision is sufficient. Do not add another T/Q state owner,
atomic T/Q publication, queue, worker, callback registry, or API-side merge.

An ordinary trade/quote mutation marks the public projection dirty but does not
by itself change the global publication fingerprint. The canonical mutation
and completion result still finish synchronously before the next engine input.

### 2. Cadence publication uses the existing aggregate boundary

Any global publication built after T/Q reconciliation contains the latest
canonical T/Q view at that engine sequence. Therefore an accepted live-coverage
fence, aggregate ingress fence, evaluation timer fallback, or other ordinary
global replacement also flushes the pending T/Q projection included in it.

With step 1 installed:

- a successful live fence is normally the one-second aggregate-and-T/Q
  publication;
- its following maintenance-only timer must not create a second publication
  merely because ordinary post-fence T/Q events are dirty; those events may
  wait for the next cadence boundary; and
- when no live fence can publish, the evaluation/maintenance timer may flush
  pending T/Q state at most once for that cadence.

Time-derived T/Q changes—Tape windows becoming current/zero, quote age crossing
the stale boundary, and retention pruning—must remain visible at the first
cadence boundary at or after their exact existing deadline. They do not require
per-event publication.

When any combined publication includes current T/Q state, clear only the dirty
work represented by that publication. Because publication is decided under the
sole engine lock, a later engine input cannot be accidentally cleared.

### 3. Immediate publication is limited to trust transitions

The following transitions must remain immediately publishable when their
public result changes:

| Transition | Why immediate |
| --- | --- |
| First valid post-request trade or quote confirms a channel generation | Changes coverage from unavailable to warming/current. |
| Desired/requested/provider membership changes or coverage closes | Changes whether displayed T/Q fields are trustworthy. |
| T/Q drop, rank removal, connection/epoch change, or command failure closes coverage | Retained values must not look current across a gap. |
| Pressure mode, pressure cause, aggregate-only, shedding, or recovery-mode transition | Trader must see protection state promptly. |
| Quarantine, explicit provider-control error, accounting integrity, or retention bound | Changes the failure domain or field availability. |

An ordinary qualifying trade, nonqualifying trade, ordinary quote update,
duplicate, bounded rejection counter increment, unchanged pressure sample, or
unchanged timer prune is not an immediate global-publication trigger.

If one input causes both an immediate transition and ordinary numeric changes,
publish once after the complete input transition; never publish an intermediate
state.

### 4. Combined-publication validity remains fail-closed

`validTQPublication` continues to validate the T/Q view against the same
aggregate evaluation and publication ID. Coalescing must not weaken any tuple,
membership, coverage, pressure, accounting, or row validation.

If a combined candidate is invalid, retain the existing fail-closed integrity
path. Do not recover performance by skipping validation, omitting accounting,
serving an independently newer T/Q view, or letting the API join two cells.

### 5. Fixed-cardinality observation

Add or reuse bounded counters sufficient to distinguish:

```text
canonical T/Q mutations
T/Q projection dirtied
T/Q projection flushed by cadence publication
immediate T/Q trust-transition publications
coalesced ordinary T/Q mutations
```

The identities must reconcile. They need not be added to API v2 unless an
existing operations field can carry them without schema ambiguity. No symbol-
level or event-level retained diagnostic history is permitted.

## Implementation boundary

Expected production changes are limited to:

- `internal/engine/engine.go`, `publication.go`, `tq.go`, `tq_pressure.go`, and
  focused tests;
- the smallest runtime test updates required to compose with step 1; and
- snapshot mapper/UI fixtures only if an existing test incorrectly assumes
  one publication per T/Q fact. The public v2 field meanings should not change.

Do not change provider subscription commands, data-confirmation boundaries,
trade/quote normalization, pressure thresholds, aggregate evaluator behavior,
readiness, UI poll cadence, API capture mechanics, checkpoint/replay claims, or
process supervision.

## Required deterministic proofs

### P1 — high-rate facts do not create high-rate publications

Drive the real engine with a qualified 20-symbol selected set, confirmed trade
and quote channels, at least 100,000 valid mixed T/Q facts, and 60 one-second
aggregate publication cycles.

Prove:

- every fact reaches its exact canonical disposition and accounting identity;
- ordinary facts do not increment publication ID individually;
- total replacements are bounded by the 60 cadence publications plus the exact
  finite trust transitions deliberately exercised by the fixture;
- publication count is O(cadence + transitions), not O(T/Q facts);
- final retained trade/quote/fingerprint counts respect existing bounds; and
- aggregate watermark, qualification, ranking rows/order, readiness, and
  aggregate accounting equal an A-only control.

Record total facts, event rate, publication replacements by cause, allocations,
heap before/peak/after-GC, processing latency, queue high-water, and proof
duration. The fixture must finish under 15 minutes.

### P2 — cadence values equal an eager oracle

At every one-second publication boundary, compare Tape, Spread, membership,
coverage, pressure, and all T/Q accounting with a deterministic oracle that
applies the same ordered facts eagerly but observes only at that boundary.

Include qualifying/nonqualifying trades, duplicates, unequal repeats, valid/
one-sided/crossed quotes, quiet covered time, stale quote transition, rank
churn, and generation fencing. The values and status/reason tuples must be
identical at every observed boundary.

### P3 — trust transitions publish immediately and once

Separately prove first trade confirmation, first quote confirmation, coverage
drop, rank removal, T/Q control error/quarantine, entry to degraded, entry to
aggregate-only, and recovery to normal. Each changed transition must replace
the combined publication exactly once before its completion is observed; an
unchanged repeated sample/fact must not replace it again.

Every variant must preserve aggregate results byte-for-byte relative to the
A-only control.

### P4 — post-fence dirty state waits safely

Apply a cadence fence/publication, then ordinary trades and quotes, then the
step-1 maintenance-only timer. Prove no extra publication occurs and the dirty
state remains pending. The next cadence fence must publish the exact latest
T/Q state once. Repeat with an immediate coverage-closing event and prove that
the close does publish immediately rather than waiting.

### P5 — publication integrity and API snapshot

Map cadence and immediate-transition publications through the real snapshot
API. Prove each response contains one coherent publication ID shared by
aggregate and T/Q views, valid strict-v2 status tuples, exact server row order,
and no acknowledgement-era membership claim. Inject one invalid T/Q tuple and
prove the existing fail-closed containment remains active.

## Verification and acceptance

Run narrow T/Q/publication proofs during correction. Final offline acceptance
requires:

```text
go test -count=1 -short -timeout 2m ./internal/engine ./internal/operations ./internal/snapshotapi
go test -count=1 -short -timeout 2m ./...
go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi
go vet ./internal/engine ./internal/operations ./internal/snapshotapi
git diff --check
```

Run P1 separately with an explicit timeout no greater than 15 minutes. One
read-only review must focus on publication linearization, dirty-state clearing,
immediate coverage closure, combined-publication validation, and the absence of
a second T/Q publication authority.

The step is **offline complete** when P1-P5 pass, affected verification and
review are clean, and a code walkthrough shows that an ordinary trade or quote
cannot replace the global publication on its own.

Do not perform a provider test after this step. Continue directly to step 3.

## Explicit non-scope

- changing Tape Speed or Spread formulas;
- weakening data-confirmed membership or coverage;
- changing pressure thresholds or recovery hysteresis;
- sampling, dropping, batching, or reordering canonical T/Q inputs;
- creating a separate T/Q snapshot/API endpoint;
- incremental aggregate evaluation;
- snapshot-capture/API/UI isolation, which is step 3;
- replay/checkpoint repair; and
- hosting or public deployment.

## Step 2 offline acceptance record — 2026-08-20

The engine now keeps canonical T/Q mutation progress and public projection
dirty state separately. Trades, quotes, drops, command results, pressure
facts, and timer maintenance still complete synchronously in engine order, but
ordinary mutations no longer advance the global publication fingerprint.
Accepted live fences, aggregate/timer publication boundaries, and other
combined replacements include the latest canonical T/Q view and clear only the
dirty work present under the engine lock. First data confirmation, membership
and rank changes, coverage closure, control quarantine/recovery, retention
bound, and pressure-mode changes advance the immediate projection revision and
replace the combined publication once.

The fixed-cardinality accounting identity is:

```text
canonical T/Q mutations = immediate trust-transition mutations
                         + coalesced ordinary mutations
```

### Deterministic proofs

- **P1 — high-rate facts:**
  `TestTQPublicationCoalescingP1HighRate` passed with 100,000 mixed facts and
  60 cadence cycles in 39.413 seconds. It recorded 60 cadence replacements,
  42 finite immediate-transition mutations/publications, 100,102 canonical
  mutations, 100,060 coalesced ordinary mutations, zero queue occupancy, and
  bounded retained trade/quote/fingerprint counts. The fixture's unchanged
  aggregate baseline is the A-only control for this T/Q isolation proof.
- **P2 — cadence oracle:**
  `TestTQPublicationCoalescingCadenceOracle` passed with two deterministic
  engines. The coalesced immutable cell matched the eager canonical T/Q
  observation at each boundary across qualifying, duplicate, unequal-repeat,
  one-sided and crossed quotes, stale aging, pressure sampling, generation
  fencing, quiet time, and rank removal.
- **P3 — immediate transitions:**
  `TestTQPublicationCoalescingImmediateTrustTransitions`,
  `TestTQPublicationCoalescingFamilyDropAndQuarantineRecovery`, and
  `TestTQPublicationCoalescingPressureTransitions` passed. First trade/quote
  confirmation, coverage close, family-drop edge triggering, rank removal,
  quarantine and greater-epoch recovery, degraded/aggregate-only entry, and
  normal recovery each produced one immediate replacement; repeated unchanged
  inputs did not republish.
- **P4 — post-fence dirty state:**
  `TestTQPublicationCoalescingPostFenceDirtyState` passed the real live-fence
  sequence: ordinary post-fence trade/quote facts remained dirty through the
  maintenance-only timer, the next live fence published the latest T/Q view
  once, and a subsequent coverage close published immediately.
- **P5 — publication/API integrity:**
  The coalescing proofs assert one publication ID for the aggregate and T/Q
  views and exact equality between the stored T/Q cell and the engine's
  canonical view at flush. Existing engine publication-integrity tests and
  snapshot API/mapper/HTTP tests continued to pass, including invalid tuple,
  accounting, pressure, and fail-closed paths.

### Verification and review

```text
PASS  go test -count=1 -short -timeout 2m ./internal/engine ./internal/operations ./internal/snapshotapi
PASS  go test -count=1 -short -timeout 2m ./...
PASS  go test -count=1 -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi
PASS  go vet ./internal/engine ./internal/operations ./internal/snapshotapi
PASS  git diff --check
PASS  TestTQPublicationCoalescingP1HighRate (-timeout 15m)
```

The focused read-only review used `gpt-5.6-sol` with medium reasoning. Its
initial findings on repeated family drops and quarantine recovery accounting
were corrected with targeted implementation changes and regression proofs; the
follow-up review returned **no findings**. All acceptance was deterministic and
offline. No provider credentials were read and no live provider request was
made. Unrelated worktree changes were preserved.
