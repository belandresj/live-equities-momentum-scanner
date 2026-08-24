# LBR-P1 baseline characterization

**Status:** Frozen deterministic comparison input for `LBR-P1`, 2026-08-23;
owner-revised to the exact 10-minute E2 manifest on 2026-08-23.

**Authority:** [delivery program](delivery-program.md), Section 5, and
[integration/removal/acceptance](integration-removal-and-acceptance.md),
Section 6. The delivery program remains the sole mutable status ledger.

**Production baseline:** commit `0d043c1`, tree
`cd67a918d5ad58c47367112a3e0dc0d13457bd9c`.

This artifact changes no production behavior, makes no provider request, and
does not reinterpret historical measurements. It freezes the exact semantic
corpus, durable baseline facts, and mature deterministic workload used to
compare the replacement with the accepted live baseline.

## 1. Provenance and comparison rule

The code, fixture, launcher, API, and UI inputs below are read from
`0d043c1`. Commits `9601603`, `c43c7b2`, `8148d4b`, and `81c391e` change
planning documents only; `git diff 0d043c1..81c391e -- '*.go' '*.json' ui
scripts` is empty. The combined SHA-256
over each listed path followed by a NUL and its exact baseline bytes is:

```text
5e892be8610f8e36195c67220760a3e2a7ce8ced5d376e73d33174a8bb59f2b4
```

Later comparisons use semantic projections and public outputs, never private
struct equality. An expensive result is reusable only when commit, command,
configuration, fixture checksum, and premise are unchanged. CPU, latency, heap,
and RSS from different fixtures or process boundaries are not substituted for
one another.

## 2. Approved semantic and compatibility corpus

| Boundary | Frozen primary inputs at `0d043c1` | Expected result |
| --- | --- | --- |
| aggregate merge/coverage | `TestENGAGG01ValidationMergeRetentionMatrix`; `TestC6INTEGRATION01OfflineComponentsOneThroughSixAggregateLifecycle`; `internal/massive/testdata/component4-rest-second-bars.json` | Exact identity, duplicate/revision/withdrawal, source precedence, coverage, empty, terminal work, and fence assertions pass. |
| qualification/ranking/current fields | `TestC3QUAL01ExactGateBoundaryMatrix`; `TestC3QUAL02CorrectionAndStrictFinalizationTrace`; `TestPMVPVolumeActivityMoveExactCorrectionAndPermutation`; `TestPMVPRankSelectionAndPreselectionHistory` | Exact gate boundaries, revocation/finalization, Day-%/symbol order, preselection history, Volume, Activity 30s, and Move 30s assertions pass. |
| selected-row T/Q | `TestTQDataConfirmationSeparatesWriteFromCoverage`; `TestPC9ExactWaitingPressureBoundaries`; all other non-skipped `TestPC9*` pressure/bound tests; `internal/engine/trade_conditions_fixture.json` | Post-write data confirmation, quiet/unconfirmed distinction, exact pressure boundaries, accounting, containment, and aggregate independence pass. The skipped acknowledgement-created `TestPC9TAQ` trace is not evidence. |
| decoder/control/heartbeat | `TestPC5ClassStrictMixedFrameClassification`; `TestPHRHeartbeatInboundProgressAndQuietDeadline`; `TestLiveSustainedAggregateCapacity` | Mixed-frame dispositions, control semantics, inbound-aware heartbeat, and the accepted 216-frames/s capacity assertions pass. |
| API v2 | `TestPC10SchemaGoldenIdentityAndSemanticMutations`, the complete `internal/snapshotapi` short suite, and its accepted mapper corpus | `scanner.snapshot.v2`, row order, availability, accounting, readiness, and rejection behavior pass. |
| UI | `ui/test-fixture-v2.js`; `node --test ui/model.test.mjs ui/visual.test.mjs` | The accepted 12-column model, status, ordering, color, and presentation assertions pass. |
| launcher | `internal/privatelauncher/launcher_test.go`; `scripts/run-private-scanner` | Checkpoint-off fresh-start composition, process independence, and launcher behavior pass. |

Frozen corpus group digests are:

```text
aggregate/qualification/hydration  b0e523d969ddadbec3568470c8b3b67a04966243f8eb609be7113d79ee138d57
T/Q                                1e438a41d01675ac956346fbe6ba5c072585d84a4142eb1ec65d587dedcaf3b8
decoder/connection/capacity        da543879620b0afafab66ceec50d29af8f6605cfe022abb73ee67dec6a0abcb5
API mapper corpus                    a7aa66c0d7d94fa05e9de87c03bb968a6afa48dc9ceea320035137144156d249
UI fixture and tests                 c6242ded9cff7accc9d6d3cef3a6b99f79102f6442ddf0d572ed82aa74ba4699
```

The focused semantic baseline was rechecked during characterization with
bounded package commands, and the API/UI commands passed. The 60-second
capacity and mature evaluator trials are reused from the accepted evidence in
Section 3 rather than rerun.

## 3. Durable resource and lifecycle facts

These observations retain their original scope and limitation:

| Evidence | Frozen fact | What it does not establish |
| --- | --- | --- |
| [replacement parent](../live-backend-replacement.md), Section 1 | The latest incident admitted and dispositioned 1,127,006 frames, rejected zero, reached 1,723/32,768 raw-frame slots and 659,669/134,217,728 queued bytes, then ended only after inbound progress stopped and the heartbeat deadline expired. | It is not a replacement capacity run and supplies no comparable CPU/RSS. |
| [heartbeat correction](../live-aggregate-heartbeat-and-resubscription-correction.md) | A failed heartbeat with supported inbound progress is nonterminal; five seconds without inbound progress, a read failure, or independent transport failure retires the epoch. Deterministic retry/hydration/fence proofs passed. | No new provider chronology or provider SLA. |
| [scanner recovery record](../live-scanner-recovery-narrow-fix.md), Gate F D4 | The exact 5,694-symbol/961-record fixture produced 60 advancing cycles with 399.871 ms mean and 601.020 ms maximum. | It is evaluator-only historical evidence, not whole-process latency. |
| same, Gate F D4 and pre-retry headroom | The credential-free run dispositioned 12,960/12,960 frames at 216 frames/s, 25,920 aggregates, final queue zero, high-water 6 (later production-cap rerun: 2), and zero rejection. | It is a 60-second ingress composition, not the exact 10-minute E2 resource/stability run. |
| same, corrected cached composition | 5,511/5,511 hydration requests completed in the cited live observation; the earlier deterministic cached composition completed 5,502 symbols/7,581,690 rows and ten coherent cycles. Successful empty, exact fence, and post-live recovery semantics remain approved. | Historical populations and cached bytes are not silently resized into the 5,694-symbol manifest. |
| parent Section 1 durable evaluation records | Approximate 333-382 ms median cycles, 1.11-2.10 s maxima, and 1.28-2.25 GB heap in use on the former implementation. | Different incident/configuration ranges are diagnostic only, not a single comparable trial. |

Comparable baseline average CPU, one-second CPU p95, and backend RSS are
**unknown**. Reader-only RSS and checkpoint/resource figures are different
process boundaries or premises and are not backfilled. The replacement run
must measure all parent targets directly.

## 4. Mature deterministic manifest

### 4.1 Identity, host, and runtime

```text
manifest schema:            lbr-mature-v2
seed:                       0x4c42522d50312d31
binding identity:           lbr-mature-v2:2026-08-12
trading date:               2026-08-12
session:                    [2026-08-12T08:00:00Z,2026-08-13T00:00:00Z)
timed interval:             [2026-08-12T21:15:00Z,2026-08-12T21:25:00Z)
population:                 5,694 symbols, exact names S0000..S5693
symbol-list SHA-256:        201aca43d3fbf758aa1a9859a0c2e86d76531d547139e69adb8b5b8037704866
prior close:                valid finite 10.0 for every symbol
prior-close SHA-256:        f6292b973934402b07bdd6151287d79649140dc5bbe9729701ab63312aa4bf0f
host:                       Apple M1, 8 logical/8 physical CPUs, 8 GiB RAM
OS/architecture:            Darwin 25.2.0, arm64
Go:                         go1.26.5 darwin/arm64
GC:                         GOGC=100 default (unset); GOMEMLIMIT=off (unset)
```

The symbol digest is over exact newline-terminated symbol names. The
prior-close digest is over exact symbol-sorted lines
`SYMBOL,valid,0x1.4000000000000p+3\n`, using the stated hexadecimal float
encoding for 10.0.

### 4.2 Hydration and mature state

Generation 1 plans all 5,694 symbols for `[S,21:15:00Z)` and terminates as:

```text
completed_value=5688 completed_empty=2 failed=1 canceled=1 fenced=2
```

The non-value assignment is exact: `S0000`/`S0001` are empty, `S0002` fails,
`S0003` is canceled, and `S0004`/`S0005` are fenced. Generation-1 value
symbols carry 961 exact one-second records ending immediately before the
timed interval: 5,466,168 rows. Empty results prove their exact interval.
Generation 2 replans `S0002` through `S0005` and completes four values
(3,844 rows), leaving the final mature prefix/tail input with 5,692 value
symbols, two proven-empty symbols, 5,470,012 aggregate rows, exact terminal
accounting, and a reconciled live fence before timing. This preflight exercises
failure/cancel/fence containment but begins the timed trial only after a
current ordinary evaluation.

For logical seconds `q=-961..-1`, mature value rows use the same exact formula
as the timed stream below: `close=10+((i+q) mod 5694)/8192`,
`open=close-1/8192`, `high=close+2/8192`, `low=close-2/8192`, `VWAP=close`,
`volume=1500+(i mod 17)`, and `AverageTradeSize=10`. Negative remainders wrap
into `[0,5694)`. All price operands are exactly representable binary64 values
and coverage is complete. The two empty symbols receive
their first actual live aggregates in timed second zero; they remain
unqualified until the exact 45-distinct-second gate is supported. No synthetic
bar represents their earlier empty interval.

### 4.3 Timed stream and pacing

The timed run is exactly 600 wall-clock seconds at exactly 300 complete
provider frames per second: 180,000 frames. For each logical second, ordinary
aggregates for all 5,694 symbols are partitioned in frame order: the first 294
frames contain 19 aggregates and the last six contain 18. Every frame is
released on the deterministic 1/300-second pacing boundary; batching may not
be reduced after a failure.

For logical second `q` and symbol index `i`, the accepted close-order scalar is
`(i+q) mod 5694`. Values use the exact formulas above, with window
`[21:15:00Z+q,21:15:01Z+q)`, frame sequence `300*q+f+1`, zero-based array
order, and UTC receipt time at that frame's 1/300-second pacing boundary. This
makes the exact top 20 rotate by one symbol per second without changing
qualification.

Except at the pressure second below, each second also appends one trade and one
quote for each of the 20 selected symbols to the first 20 mixed frames. Event
time is the window start plus 500 ms; trade price is `close`, size is
`100+(i mod 11)`, and the ordinary quote is
`bid=close-1/8192,ask=close+1/8192`. Coverage is confirmed through the accepted
post-write boundary; one-symbol-per-second rank churn forces removal/addition
and fresh T/Q warm-up without making T/Q a ranking input.

Within each minute, second offsets 10, 20, 30, 40, and 50 are fixed episodes:
offset 10 uses reviewed non-volume condition `15`, 20 uses a locked
quote, 30 uses a one-sided quote, 40 uses a crossed quote, and 50 replaces all
40 T/Q events with ordered pressure-shed/coverage-closure facts. Thus aggregate
progress is unchanged while quiet/quality/gap/pressure behavior is exercised.

The fixed injection counts are:

```text
frames=180000
base_aggregates=3416400
trades=11800
quotes=11800
tq_pressure_shed=400
aggregate_duplicates=600
aggregate_revisions=60
aggregate_invalid=20
aggregate_control=2
resource_samples=600
dashboard_polls=600
ranking_checkpoints=11
```

Duplicates occur once per second, revisions once per ten seconds, local invalid
evidence once per 30 seconds, and control/heartbeat facts once per 300 seconds.
The duplicate targets symbol `q mod 5694`; the revision targets
`(7*q+1) mod 5694` and adds exactly `1/8192` to close/high/VWAP; the invalid
targets `(11*q+2) mod 5694` with finite price identity but volume `-1`; and the
control fact is the accepted healthy/inbound-progress class. They are appended
after the ordinary event with a greater array or frame position and use the
approved identity/precedence/containment rules. The
canonical expected-count digest over the exact newline-terminated count lines
in this order is:

```text
895bec49c7d30fcb2e3ecb73fe724f73883c8aa437fea316272bf4daf5ab6ac8
```

### 4.4 Frozen expected identities and digests

At `q=0,60,...,600`, record the exact ordered top-20 symbols and primary
population identity. Ranking checkpoints use the unique scalar above; the
ranking SHA-256 over lines `ssss,SYMBOL,...,SYMBOL\n` is:

```text
d98e13cde289da133bb5ec0eeddd7c88f1628d4d3d646d8b9f660a0f837a529a
```

Accounting lines are
`ssss,universe=5694,valid_prior=5694,qualified=N,unqualified=M,ranked=N\n`.
At `q=0`, `N=5692`; at all one-minute checkpoints thereafter, `N=5694`.
Their SHA-256 is:

```text
3ed737914233b2dc93892c2b321caef04d898b9ff23e321795a73322ba70d8c6
```

The approved API-v2 and UI expected behavior is frozen by the API and UI
digests in Section 2. `LBR-E2` must additionally
emit a canonical full snapshot/API digest at each checkpoint and record it in
the acceptance artifact; if the replacement and baseline oracle disagree, the
semantic proof fails rather than updating this manifest.

### 4.5 Bounds, samples, stop conditions, and commands

The baseline raw FIFO is 32,768 frames, 128 MiB total raw bytes, and 8 MiB per
frame. Baseline T/Q limits are 20 symbols, 50,000 trades and 100,000
fingerprints per symbol, and 500,000/1,000,000 globally. The replacement is
measured with its approved focused-contract bounds: 4,096 decoded batches,
64 MiB total decoded charge, 8 MiB source frames, 65,536 elements and 32 MiB
per batch, eight entries/64 KiB marker reserve, and the `tq-state.md` count/
byte limits. Any program-revised division is recorded before `LBR-D1`/`D2`
and becomes part of the final command configuration.

Before timing, validate manifest schema/digests, population, session, 5,470,012
hydration rows, terminal identities, fence, frame/event totals, injection
counts, bounds, exactly 600 resource samples, exactly 600 dashboard polls,
exactly 11 ranking checkpoints, and the API/UI corpus checksum. Sample once per
second exactly as integration Section 6 requires. Stop immediately on
accounting incoherence, aggregate/control loss, unbounded queue/state growth,
repeated readiness flap, unusable polling, fixture mismatch, or the exact
10-minute deadline. A numeric target miss is recorded from the one run when
hard acceptance passes and does not alter the manifest or authorize a repeat.

Frozen comparison commands are:

```text
go test -count=1 -timeout 2m -run '<frozen semantic regex>' ./internal/engine ./internal/massive ./internal/snapshotapi
node --test ui/model.test.mjs ui/visual.test.mjs
env LIVE_RETAINED_TAIL_UNIQUE_RECORDS=1 go test -v -count=1 -timeout 2m -run '^TestLiveRetainedTailCostAttribution$' ./internal/engine
go test -v -count=1 -timeout 2m -run '^TestLiveRetainedTailSixtyOneSecondCycles$' ./internal/engine
go test -v -count=1 -timeout 110s -run '^TestLiveSustainedAggregateCapacity$' ./internal/operations
go test -count=1 -short -timeout 2m ./...
```

The first two were rerun for `LBR-P1`; the last three reuse unchanged accepted
evidence. `LBR-E2` owns the new manifest validator and exactly one authorized
10-minute timed command with an explicit 15-minute command timeout. It must name
this manifest identity and record the binary/source commit, configuration,
output checksum, and target table before interpretation. No optional, fallback,
diagnostic, host-coexistence, or non-gating repeat command exists.

## 5. Artifact checksums

Individual baseline input checksums are intentionally recoverable with
`git show 0d043c1:PATH | shasum -a 256`; the combined and group digests above
are the comparison identities. The checksum of this characterization file is
recorded in the `LBR-P1` ledger entry after its final bytes are fixed, avoiding
a recursive self-checksum field.
