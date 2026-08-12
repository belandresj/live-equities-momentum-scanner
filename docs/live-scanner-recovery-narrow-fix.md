# Live scanner recovery narrow fix

**Status:** Accepted for the fresh-start live milestone after corrected Gate E
passed, and re-accepted after a closed-market vertical proof exposed and closed
the first-session Activity/checkpoint equivalence defect. Final independent
reviews found no actionable P1/P2 or authority drift. S1, S2, Gates A-E, the
reader/playback prerequisites, and the credential-free private scanner vertical
are accepted. Gate F remains a separate, unexecuted market-hours validation
requiring exact authorization for its trading date, credentials, duration, and
diagnostic path.

**Boundary and completed-contract authority:** Direct owner request on
2026-08-11 to use the read-only recovery review and specify the narrow fix.

**Standing program decisions:** The
[`Version 1 Release Program`](v1-release-program.md) controls C7-C10 correction
and verification. The direct request authorizes the narrow C3 mechanics change,
not changed C3 market semantics.

**Advancement mode:** Sequential correction with one active slice. Failures use
the V1 correction loop; there is no lower-level owner-response gate.

**Approved dependencies:** Accepted C2/C3 engine/evaluator, C5/C6 live and
hydration, C7 checkpoint, C8 runtime, C10 API, and C11 UI contracts.

**Decision:** Compose the demonstrated evaluation coalescing and terminal
suppression behavior, then correct only the measured late-session Activity
hotspot. Do not merge either dirty worktree wholesale or redesign ranking.

This is one C3/C7/C8/C10 correction. Pause C12 and preserve its files.

## Contract map and delivery ledger

| Document | Normative responsibility | Coverage | Read for | Depends on |
| --- | --- | --- | --- | --- |
| This file | Compact single-file correction contract, Sections 1-19 | `NARROW-*`, `P-NARROW-*`, S1/S2 | Every implementation, proof, review, or correction | Accepted dependencies named above |

This is the sole ledger; prior corrections are evidence.

| Item | State | Evidence/review | Next action |
| --- | --- | --- | --- |
| Correction contract | `accepted` | The owner authorized one further full Gate E execution after the exact cycle-1 cause was proven and corrected. Gate E passed, the final `gpt-5.6-sol` medium review returned CLEAN/PASS, and final verification is recorded below | Preserve accepted A-E evidence; Gate F remains separate and unexecuted |
| S1 — stable live path | `accepted` | `P-NARROW-LIVE` and Gate B passed 2026-08-11; exact evidence below | Preserve; no S1 rerun is required by S2-P |
| S2 — measured mature cost | `accepted` | Activity correction and synthetic A-D evidence pass. Corrected Gate E completed 7,581,690 hydration rows, then ten exact runtime-owned cycles in 73.53-122.18ms each against the two-second limit, with monotonically increasing publication IDs | Complete for the fresh-start live milestone; this is current-host cached-data evidence, not provider latency or an SLA |
| S2-R — production-reader resource diagnosis | `accepted_prerequisite` | Corrected exact reader validates all 2,584,011,150 bytes and 7,671,171 records in 40.53s with 20.30 MB peak RSS; compact trust/scale/repository/race/vet/review evidence is clean | Complete; reused by S2-P and any later authorized composition |
| S2-P — production-playback preparation correction | `accepted_prerequisite` | Milestone `3ab1e1b`; one trusted requested-prefix manifest and reused same-open cursor reduce five strict parses to three. Exact reader-only proof validates 7,581,690 selected rows and 5,439/63 value/empty symbols in 2m6.65s; focused trust/resource, short/race/vet, and required review are clean. The new Gate E independently completed preparation in 1m21.202s and hydration in 2m27.780s | Complete; the new failure is downstream of playback and hydration |
| S2-E — corrected cached composition | `accepted` | Newly authorized corrected Gate E passed in 233.93s: exact prefix/row/value/empty counts, queue high-water 320/512, zero rejection, 26,963 sent/read/admitted/dispositioned frames, ten coherent automatic cycles, live/qualified-current lifecycle, 20 ranking and T/Q rows, valid accounting/readiness, and deterministic shutdown | Complete; do not rerun unchanged evidence |
| Post-E closed-market vertical | `accepted_after_correction` | `TestPV1RCVertical` initially rejected checkpoint projection because the first accelerated Activity lookup evaluated a nonexistent pre-session block and diverged from checkpoint rebuild. Both accelerated loops now clamp to the first valid 04:00:30 reference end. The direct boundary regression, checkpoint round trip, full scanner vertical through live A/T/Q/API/UI/shutdown, focused races, short suite, vet/diff, and focused review are clean | Complete; preserves Gate E and closes checkpoint-on launch composition without claiming provider behavior |

### S1 acceptance evidence — 2026-08-11

S1 preserves immediate canonical aggregate mutation while coalescing the
full-population evaluator and immutable publication to the next live timer or
hydration ingress fence. It also preserves the first typed evaluator failure,
serves installed-binding warming and suppressed captures with HTTP 200 while
`/readyz` returns 503, stops `RunLive` before any connection attempt in a
terminal lifecycle, and adds a default-on live checkpoint switch whose off
path constructs no store/writer and submits no checkpoint work.

Gate A passed in fast-to-long order:

```text
go test -timeout 90s ./internal/engine -run 'TestLiveAggregateEvaluationCoalescing|TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect|TestC3POP02AccountingIntegrityAndOverlap' -count=1  # 2.31s
go test -timeout 90s ./internal/operations -run '^TestSuppressedIngressCannotEnterReconnectHotLoop$' -count=1  # 0.89s
go test -timeout 90s ./internal/snapshotapi -run 'TestLiveWarmupSnapshotIsServableAndBound|TestSuppressedEvaluatorIntegrityRemainsServable' -count=1  # 0.59s
go test -timeout 90s ./cmd/scanner -run 'TestLiveOperator|TestInitialOperator' -count=1  # 0.58s
perl -e 'alarm 90; exec @ARGV' node --test ui/model.test.mjs  # 0.34s, 18/18
go test -timeout 90s ./internal/engine -run 'TestC3ACT|TestFreshHydrationActivityOptimizationsMatchValueOracle' -count=1  # 1.08s
go test -timeout 90s ./cmd/scanner -run '^TestCheckpoint' -count=1  # 0.56s
go test -timeout 90s ./internal/operations -run '^TestProductionCompositionCoalescesThousandAcceptedAggregates$' -count=1  # 0.58s
```

The 1,000-event composition admitted all inputs with one insert and 999
revisions, zero rejected/canceled/closed/sequence-exhausted admission deltas,
zero publication replacements before the timer, and exactly one replacement
at the timer. The resulting population was universe `1`, valid prior `1`,
trusted rankable `1`, covered `1`, unresolved `0`; aggregate/admission
identities reconciled and queue occupancy returned to zero.

The ordinary Gate B command `go test -short -timeout 2m ./...` passed in
11.61 seconds. Orchestrator review rejected an initial proof that synthesized a
sealed API capture with `unsafe`. The corrected proof used the production path
from a real evaluator accounting fault through suppression, sealed runtime
capture, and HTTP mapping; focused snapshot, engine, and UI checks passed in
1.92 seconds, 1.63 seconds, and 0.43 seconds, and the repeated Gate B passed in
8.65 seconds. `git diff --check` was clean.

Construction prevents a pending accepted prefix from disappearing, clears it
only after successful evaluator validation/apply, latches rather than replaces
the first evaluator diagnostic, preserves deterministic replay behavior,
prevents a suppressed/ended runtime from opening a socket, and makes
checkpoint-off status explicit with zero checkpoint work. This proves S1
semantics and ordinary conformance, not late-session capacity, provider
behavior, or Gate F.

### S2 correction and Gate E failure evidence — 2026-08-11

The mandatory pre-change 6,000-symbol 17:15 measurement had exactly 1,589
reference blocks plus one target block per symbol. Three trials reported:

| Trial | Stage | Apply | Publication | Total lock | Allocated bytes |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 10.268257667s | 9.082167ms | 519.125µs | 10.2778605s | 6,497,812,568 |
| 2 | 11.541589417s | 4.965542ms | 775.333µs | 11.547334875s | 6,492,435,720 |
| 3 | 10.663651875s | 5.47725ms | 83.458µs | 10.669216125s | 6,492,435,736 |

Mean lock time was about 10.83147 seconds and every trial exceeded two
seconds. Stage time and about 6.49 GB of allocations per publication isolated
the session-length Activity traversal; apply and publication did not justify a
ranking index or broader evaluator change.

The triggered correction keeps the existing at-most-1,920 block summaries as
semantic and checkpoint authority and adds only two derived sorted finite-value
collections, each bounded by 1,920. A corrected eligible block removes its old
values, recomputes from canonical evidence, and inserts its new values.
Percentiles retain inclusive `<= target` ties. Whole-interval trust is checked
before lookup use, and checkpoint installation rebuilds the derived lookup
through ordinary deterministic apply; nothing is persisted in checkpoint
schema.

Post-correction Gate C passed with the same exact manifest:

| Trial | Stage | Apply | Publication | Total lock | Allocated bytes |
| ---: | ---: | ---: | ---: | ---: | ---: |
| 1 | 287.817333ms | 1.369833ms | 94.792µs | 289.295333ms | 14,692,504 |
| 2 | 62.610167ms | 429.458µs | 38.459µs | 63.078417ms | 9,315,656 |
| 3 | 60.465416ms | 431.5µs | 68.167µs | 60.965375ms | 9,315,640 |

Mean lock time was 137.779708 ms and maximum was 289.295333 ms. Gate D's
generated 60 one-second evaluation cycles reported stage total 4.767944542s,
apply total 304.075541ms, publication total 1.700374ms, mean lock 84.562449ms,
maximum 359.461709ms, zero aggregate rejection, zero queue growth, and lookup
cardinality at most 1,920. Checkpoint-off production composition separately
made ten successful HTTP snapshot polls while checkpoint submitted/outstanding
work remained zero. The Gate D fixture directly measures staged/apply/
publication cycles; it does not instantiate HTTP or admit a fresh aggregate
correction in each measured cycle, so those are separate proofs rather than one
fully joined D trace.

Activity differential proofs cover insert, correction, withdrawal containment,
the exact 100-transaction eligibility boundary, restoration, inclusive target
ties, stalled `T`, horizon folding, and checkpoint continuation/rebuild. The
focused Activity/checkpoint suite passed in 3.00 seconds; checkpoint-off HTTP
composition passed in 2.27 seconds; `go test -short -timeout 2m ./...` passed
in 7.99 seconds before Gate E and 8.93 seconds after it.

The corrected Gate E preflight passed without a full parse: the compact
production-reader fixture passed in 0.01 seconds; the exact private regular
artifact was 2,584,011,150 bytes with file SHA-256
`e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189`;
the exact reference files were `prior-close/2026-08-06.json` and
`universe/2026-08-07.json`; reference/binding validation completed in about
3.30 seconds; and the recorded manifest named binding
`session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d`,
7,671,171 records, 5,691 symbols, and the complete session interval
`[2026-08-07T08:00:00Z,2026-08-08T00:00:00Z)`.

The one permitted Gate E command was:

```text
/usr/bin/time -p env CACHED_HYDRATION_ACCEPTANCE=1 CACHED_HYDRATION_RATE_MULTIPLIER=1 go test -timeout 8m ./internal/operations -run '^TestCachedHydrationFenceAcceptance$' -count=1 -v
```

Its internal context was 7m30s. It emitted only
`=== RUN TestCachedHydrationFenceAcceptance`, then the process was terminated
after approximately 28 seconds inside the initial
`replayartifact.OpenValidatedContext` full-artifact validation, without a Go
failure, PASS, exit status, or `/usr/bin/time` trailer. Preload did not begin:
zero rows were applied and zero of ten planned cycles ran. The command was not
rerun. This distinguishes local process/resource termination in the existing
production reader from Activity semantics, cached-data results, provider
behavior, or a Gate E capacity result.

Because Gate E is not green, the correction remains reopened. The conditional
final read-only review was not spawned, no acceptance commit is authorized,
and Gate F remains both separately authorized and unexecuted. No provider
credential or request was accessed.

Final local checks after preserving the failure passed:

```text
go test -race -short -timeout 5m ./internal/engine ./internal/operations ./internal/snapshotapi ./cmd/scanner  # 31s
go vet ./...  # 1s
git diff --check
```

The diff contains no replay, replayartifact, replaymode, checkpoint codec/
store/writer, partial-ranking, or C12 change. Since Gate E failed, these checks
prove only local race/static/diff conformance of the deterministic correction;
they do not convert the missing cached-data result into acceptance.

### S2-R production-reader resource diagnosis — active 2026-08-11

The owner reopened only the prerequisite that the accepted production artifact
reader can fully validate the exact Gate E input within a bounded local resource
envelope. This is a diagnostic/correction allocation under `C4-ART-02` and
`C4-BOUNDED-CANCEL-01`; it does not change `aggregate-replay-jsonl-v1`, any
persisted trust meaning, replay lifecycle, the scanner evaluator, Activity, or
Gate E authorization. Accepted compact trust and cancellation evidence remains
valid unless the diagnosis directly invalidates it.

`P-NARROW-READER-RESOURCE` is the sole new primary proof. It uses
`replayartifact.OpenValidatedContext`, valid artifacts produced by the accepted
compiler/codec, fixed-cardinality progress, and host/process measurements to
report records and bytes reached, validation phase, wall/CPU time, current and
peak heap/RSS, total allocation, and GC count. The smallest scale ladder that
distinguishes behavior is used; representative targets are 100,000, 500,000,
1,000,000, 2,000,000, and the exact 7,671,171-record artifact. Every rung has
an explicit timeout/resource bound and validated fixture identity. Scaling
stops when the mechanism is distinguished, and an unchanged failed full reader
command is never repeated.

The implementation assignment is limited initially to test-only bounded
instrumentation and fixture generation in `internal/replayartifact` plus this
ledger. A production edit inside `internal/replayartifact` is permitted only
after measurements demonstrate redundant parsing, re-encoding, allocation,
retention, unbounded-line behavior, or another bounded reader defect. No
predecessor source or fixture is authorized. `internal/engine`, Activity,
ranking, hydration, C12, provider, checkpoint, API/UI, and artifact schema or
format changes are excluded. The writer does not stage or commit.

Any correction must preserve exact rejection of noncanonical bytes/numerics,
unknown or duplicate fields, signed zero, wrong ordinal/order, duplicate
complete identity, binding/date/session/interval/symbol mismatch, incomplete or
duplicate coverage, count/body-byte/digest/seal mismatch, truncation, bytes
after seal, same-open-file mutation, cancellation, and byte/record budget
violations. The dangerous false success is any such input reaching a validated
handle because a performance change skipped or approximated existing trust
work. Construction or proof must keep strict closed-schema decoding and exact
canonical byte comparison; approximate validation is prohibited.

Before any exact full-artifact reader attempt, the compact production trust
corpus and the narrow scale/resource proof must pass. If production code
changes, ordinary verification, affected focused race, vet, diff checking, and
the required read-only persisted-trust review/re-review apply. A successful
full-reader result authorizes only the report and proposed replacement Gate E
command. Gate E composition and Gate F remain unexecuted pending their separate
owner authorizations.

The initial diagnosis passed the compact persisted-trust corpus in 0.460
seconds of package time. Valid partial-synthetic artifacts were emitted by the
production canonical encoder and validated by `OpenValidatedContext` in fresh
child processes so fixture construction did not contaminate the reader
measurements:

| Records | Artifact bytes | Wall | CPU | Total allocation | GC | Peak heap | Peak RSS |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 100,000 | 29,689,864 | 0.938s | 1.000s | 766,812,064 B | 224 | 3.85 MB | 22.50 MB |
| 500,000 | 148,889,866 | 4.813s | 5.147s | 3,833,637,544 B | 1,129 | 3.87 MB | 23.00 MB |

Both rungs reached the exact final ordinal, byte count, summary, seal, and
`validated` phase. Heap after validation remained about 1.7--2.0 MB and
post-GC retained heap about 0.34 MB. The curve is linear at approximately
7,667 allocated bytes and 9.69 microseconds per record, with no retained-state
or RSS growth. Extrapolation to 7,671,171 records is approximately 54.8 GiB of
transient allocation, 17,354 garbage collections, and 74.3 seconds for the
synthetic line shape; this is diagnostic scaling, not a full-artifact result.

The 100,000-record allocation profile attributes 637.1 MB cumulative (86.25%)
to strict canonical aggregate validation and 523.1 MB (70.8%) to canonical
re-encoding. Inspection identifies the exact mechanism: every aggregate line
is decoded once to discover its kind, then strict-decoded and canonically
re-encoded, after which aggregate signed-zero normalization performs a second
canonical re-encoding. Canonical string emission also creates a fresh
`bytes.Buffer` and `json.Encoder` for each string. This is a demonstrated
production-reader implementation defect; it is not required trust work and it
does not indicate unbounded retained state.

The authorized correction may dispatch from the exact canonical kind prefix,
decode each aggregate once, normalize signed zero, and perform exactly one
canonical byte comparison. Canonical string emission may be made
allocation-light only with differential proof that it remains byte-identical
to Go 1.26 `encoding/json` with HTML escaping disabled, including controls,
invalid UTF-8 replacement, and U+2028/U+2029. Closed-schema decoding, unknown/
duplicate-field rejection, and the exact final byte comparison remain the
trust authority. No parser approximation, schema change, or skipped check is
authorized.

The implemented correction retains that exact authority. Prefix matching only
selects the closed line type; it cannot admit a line. Each aggregate receives
one `DisallowUnknownFields` decode, EOF/trailing check, signed-zero
normalization in a copy, and one exact canonical whole-line comparison.
Canonical string emission is allocation-light and differential tests match Go
1.26 `encoding/json` with HTML escaping disabled for ordinary UTF-8, quotes,
backslashes, all control bytes, invalid UTF-8 replacement, and U+2028/U+2029.
Adversarial tests reject unknown/duplicate fields, signed zero, alternate
numeric spelling, member reordering, leading whitespace, and duplicate kind.

The same valid fixed-cardinality fixtures measured:

| Records | Wall | CPU | Total allocation | GC | Peak heap | Peak RSS |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 100,000 | 0.437s | 0.469s | 328,107,704 B | 96 | 3.82 MB | 21.64 MB |
| 500,000 | 2.241s | 2.406s | 1,640,127,560 B | 484 | 3.82 MB | 22.27 MB |
| 1,000,000 | 4.462s | 4.771s | 3,280,134,816 B | 966 | 3.83 MB | 22.58 MB |
| 2,000,000 | 9.202s | 9.792s | 6,560,159,248 B | 1,939 | 3.83 MB | 22.33 MB |

At 500,000 records, wall time fell 53.44%, CPU 53.25%, allocation 57.22%,
and garbage collections 57.13%; final bytes/ordinal/summary/seal remained
exact and retained heap/RSS stayed flat. Corrected scaling is approximately
3,280 allocated bytes and 4.615 microseconds per record. The same synthetic
shape extrapolates to about 23.43 GiB cumulative allocation and 35.3 seconds at
7,671,171 records; the exact artifact remains the required distinguishing
proof rather than an inferred pass.

The required `gpt-5.6-sol` medium read-only persisted-trust review found no
P1/P2 false-success regression. It confirmed that closed-schema decode, exact
canonical byte equality, binding/session/symbol/order/coverage/accounting,
digest/seal/EOF, budgets, and cancellation remain the admission path. The
review repeated the focused trust/cancellation corpus and a 100,000-record
rung. The limitation remains that the scale ladder is one-symbol
`partial_synthetic`; it exercises the changed aggregate hot path but is not
complete-mode, exact-artifact, playback, or Gate E evidence.

Pre-full verification passed: focused canonical/adversarial plus compact trust
tests, package short, `go test -short -timeout 2m ./...` in 9.24 seconds,
`go test -race -short -timeout 5m ./internal/replayartifact -count=1` in 4.08
seconds, `go vet ./...` in 0.48 seconds, and `git diff --check`. The next action
is one reader-only validation of the exact artifact with the exact binding and
resource/progress reporting. It is not Gate E composition.

The exact reader-only command then passed once:

```text
/usr/bin/time -lp env REPLAYARTIFACT_EXACT_READER=1 go test -timeout 3m ./internal/replayartifact -run '^TestOpenValidatedContextExactArtifactResource$' -count=1 -v
```

The parent preflight recomputed file SHA-256
`e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189`
and re-established the exact 5,691-symbol binding and session interval. A fresh
child then used only `OpenValidatedContext` in `complete_final_bars` mode and
reported:

| Measurement | Exact result |
| --- | ---: |
| Reader wall / child total / command total | 40.530s / 42.31s / 44.35s |
| User CPU / system CPU | 45.802s / 2.995s |
| Artifact bytes / records | 2,584,011,150 / 7,671,171 |
| Coverage / empty symbols | 5,691 / 172 |
| Peak Go heap / retained heap | 3,966,976 B / 370,936 B |
| Reader current RSS before / after / peak | 19,496,960 B / 17,743,872 B / 20,299,776 B |
| Cumulative allocation / GC | 25,848,661,336 B / 12,178 |

The final phase was `validated`, final ordinal was 7,671,171, final progress
bytes equaled the file size, and artifact ID was exactly
`sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`.
No playback cursor, preload, engine, operations runtime, Activity evaluation,
Gate E cycle, provider, credential, or live request participated.

The historical 27.8-second disappearance did not have a recoverable signal or
macOS diagnostic. It removed the whole command session: neither Go, the shell,
nor the parent `/usr/bin/time` could report status. A 35-second idle child and a
35-second CPU-bound child both completed through the same execution supervisor,
and the corrected exact reader completed in 44.35 seconds, excluding a general
approximately-28-second supervisor CPU/wall limit. The scale ladder and exact
run also keep live heap/RSS near 20--23 MB, excluding an unbounded retained-
reader-state or ordinary process-OOM explanation. The strongest supported
classification is therefore a one-off external process-group/session
termination; whether its initiating trigger was host resource enforcement or
execution-supervisor state is no longer recoverable from available evidence.
The demonstrated reader defect was transient allocation/GC amplification, not
proved as the direct kill mechanism.

`P-NARROW-READER-RESOURCE` is accepted for the production-reader prerequisite.
Recommended reader bounds remain `MaximumBytes=3<<30`,
`MaximumRecords=8_000_000`, a 150-second direct-reader context, and a
three-minute reader-proof command. At this prerequisite gate, the proposed
replacement Gate E remained the original bounded composition command below; it
was not yet authorized by this diagnostic result or executed:

```text
/usr/bin/time -p env CACHED_HYDRATION_ACCEPTANCE=1 CACHED_HYDRATION_RATE_MULTIPLIER=1 go test -timeout 8m ./internal/operations -run '^TestCachedHydrationFenceAcceptance$' -count=1 -v
```

### S2-P replacement Gate E diagnosis and playback correction — 2026-08-11

The owner authorized exactly one replacement Gate E execution. The unchanged
command reached production hydration and then failed exactly at the internal
7m30s deadline:

```text
cached_hydration_fence_acceptance_test.go:134: hydration chunk not admitted
--- FAIL: TestCachedHydrationFenceAcceptance (450.01s)
FAIL github.com/belandresj/live-equities-momentum-scanner/internal/operations 450.396s
real 451.89
user 471.19
sys 65.81
```

The engine admission implementation classifies an already-expired context as
`not_admitted_canceled`; the harness had collapsed that exact result into the
generic message. This was not a malformed chunk disposition, queue-capacity
rejection, Activity-cycle failure, terminal/fence failure, or market-semantic
contradiction. No hydration terminal, ingress fence, readiness transition, or
one of the ten measured cycles was reached.

Inspection identified a proof-mechanics defect. `OpenValidatedContext` parsed
the complete 2,584,011,150-byte artifact once. Each of the harness's two
`BeginPlaybackThroughContext` calls then performed a complete validation pass
and a complete same-open streaming/suffix pass. Gate E therefore performed
five strict full-artifact parses around a 7,581,690-row synchronous hydration
preload, even though the prior harness used a direct parser and a 14-minute
internal context. The accepted 40.53-second reader proof measured only the
first of those five passes and did not establish the combined eight-minute
composition.

The correction keeps every C4 trust decision and removes only redundant work.
The playback first pass now captures one immutable requested-prefix count per
binding symbol, tied to artifact ID, binding ID, and requested end. Gate E uses
that bounded planning evidence to construct exact chunk totals, then reuses the
same cursor for hydration and final suffix/digest/seal/file-identity
validation. The planning evidence cannot represent terminal success, and no
hydration terminal, fence, or readiness result is possible until the second
pass returns valid requested-end evidence. Returned count maps are detached.
Partial artifacts receive no requested-prefix evidence.

The playback canonical hot path now uses exact canonical kind-prefix dispatch
and the allocation-light JSON string encoder already differentially proved for
the production validator. Closed-schema decode, duplicate/unknown rejection,
signed-zero normalization, exact whole-line canonical comparison, ordinal and
group order, coverage, digest/seal/EOF, cancellation, and same-open mutation
containment remain mandatory. Gate E diagnostics now report fixed-cardinality
progress and exact symbol/chunk/emitted rows/admission/context/queue/operational
state instead of the generic failure.

Bounded playback measurements reported:

| Records | Prepare | Stream | Allocation per pass |
| ---: | ---: | ---: | ---: |
| 100,000 | 458.890 ms | 439.355 ms | about 281.6 MB |
| 500,000 | 2.200 s | 2.225 s | about 1.408 GB |

The exact reader-only proof—no engine, hydration runtime, live adapter, fence,
publication, or Gate E cycle—then passed once. It produced a trusted 17:15
prefix of 7,587,384 artifact records, including the exact valid-prior subset of
7,581,690 rows, 5,439 value symbols, and 63 successful-empty symbols.
Preparation took 1m23.484s, the reused same-open streaming/suffix pass took
43.161s, and total test work took 2m6.65s (`real 128.73`, `user 141.11`,
`sys 8.89`). The corrected design uses three strict parses rather than five.

Focused canonical/adversarial, immutable-prefix, requested-end/replay, and
package tests pass. `go test -short -timeout 2m ./...`, focused short race for
playback/replayartifact/replay/operations, `go vet ./...`, and
`git diff --check` are clean. The required `gpt-5.6-sol` medium read-only
persisted-trust review found no P1/P2. Its residual limitation is explicit:
this proves the corrected exact reader/preparation boundary, not hydration
throughput, the ten cycles, or Gate E's eight-minute composition. The consumed
replacement Gate E was not rerun, and Gate F, credentials, provider requests,
deployment, push, rebase, and amend remain unexecuted.

### S2-E newly authorized corrected Gate E execution — 2026-08-11

The owner authorized exactly one new Gate E execution against the corrected
three-pass playback implementation. Before execution, the complete diff was
classified as only the recorded S2-P playback preparation, proof, diagnostics,
and ledger correction. The required persisted-trust review remained recorded
clean with no P1/P2. Focused canonical/adversarial and requested-prefix/end
proofs, the ordinary short suite, focused short race for playback,
replayartifact, replay, and operations, vet, and diff checking were clean. The
unchanged exact 2m6.65s prepared-playback result was reused rather than rerun.
The exact correction was committed as clean milestone `3ab1e1b`.

The compact production trust proof then passed. Exact preflight revalidated:

| Field | Exact value |
| --- | --- |
| Artifact path | `/Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/aggregate-replay/aggregate-replay-fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a.jsonl` |
| Bytes / file SHA-256 | `2,584,011,150` / `e7e33c7981ed55cb12408ee1145f78e69c11caca52fc34b3c08484a1857db189` |
| Artifact / binding | `sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a` / `session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d` |
| Reference paths | `prior-close/2026-08-06.json`; `universe/2026-08-07.json` under the recorded reference root |
| Interval / artifact records / symbols | `[2026-08-07T08:00:00Z,2026-08-08T00:00:00Z)` / `7,671,171` / `5,691` |

The authorized command ran exactly once:

```text
/usr/bin/time -p env CACHED_HYDRATION_ACCEPTANCE=1 CACHED_HYDRATION_RATE_MULTIPLIER=1 go test -timeout 8m ./internal/operations -run '^TestCachedHydrationFenceAcceptance$' -count=1 -v
```

Corrected preparation completed in `1m21.202046333s` with the exact requested
prefix: 7,587,384 prefix records, 7,581,690 selected valid-prior rows, 5,439
value symbols, and 63 successful-empty symbols. Hydration then reported:

| Selected rows reached | Elapsed | Frames read/admitted/dispositioned | Queued/classifying | Rejections |
| ---: | ---: | ---: | ---: | ---: |
| 500,000 | 5.338652250s | 1,039 / 1,039 / 1,039 | 0 / 0 | 0 |
| 1,000,000 | 10.755792333s | 2,082 / 2,082 / 2,082 | 0 / 0 | 0 |
| 1,500,000 | 17.191512958s | 3,290 / 3,290 / 3,290 | 0 / 0 | 0 |
| 2,000,000 | 24.532217500s | 4,681 / 4,681 / 4,681 | 0 / 0 | 0 |
| 2,500,000 | 30.902669875s | 5,886 / 5,886 / 5,886 | 0 / 0 | 0 |
| 3,000,000 | 38.170984500s | 7,249 / 7,249 / 7,249 | 0 / 0 | 0 |
| 3,500,000 | 45.645396958s | 8,666 / 8,666 / 8,666 | 0 / 0 | 0 |
| 4,000,000 | 53.446065750s | 10,148 / 10,148 / 10,148 | 0 / 0 | 0 |
| 4,500,000 | 1m2.301471750s | 11,801 / 11,801 / 11,801 | 0 / 0 | 0 |
| 5,000,000 | 1m11.965345916s | 13,616 / 13,616 / 13,616 | 0 / 0 | 0 |
| 5,500,000 | 1m22.247946083s | 15,511 / 15,511 / 15,511 | 0 / 0 | 0 |
| 6,000,000 | 1m33.152342333s | 17,545 / 17,545 / 17,545 | 0 / 0 | 0 |
| 6,500,000 | 1m48.108407458s | 20,248 / 20,248 / 20,248 | 0 / 0 | 0 |
| 7,000,000 | 1m59.386353166s | 22,390 / 22,390 / 22,390 | 0 / 0 | 0 |
| 7,500,000 | 2m11.294447708s | 24,598 / 24,598 / 24,598 | 0 / 0 | 0 |

All 7,581,690 rows completed in `2m27.779621916s`. At every emitted progress
boundary, oversize, capacity, receipt, and gate/close rejections were zero;
queued bytes and terminal/fence queue counters were also zero. The harness then
advanced past terminal application, the real ingress fence, first readiness,
the producer pause, and tail drain—the cycle loop is reachable only after each
of those succeeds—but it did not emit their durations or final accounting
snapshots before failure.

Cycle 1 failed before any stage/apply/publication/total-lock/allocation or
publication-ID measurement:

```text
cached_hydration_fence_acceptance_test.go:172: Gate E cycle 1 timer admission=not_admitted_closed
cached_hydration_fence_acceptance_test.go:467: shutdown: live adapter cleanup deadline exceeded
--- FAIL: TestCachedHydrationFenceAcceptance (249.51s)
FAIL
FAIL github.com/belandresj/live-equities-momentum-scanner/internal/operations 249.938s
FAIL
real 251.51
user 236.77
sys 46.84
```

Process exit status was 1. No cycle completed, so none of the ten cycle timing
or allocation vectors exists. Final lifecycle, ranking mode/rows, queue
high-water, fence reconciliation, heap-at-fence/after-GC, T/Q state, or
checkpoint-off accounting was collected. This absence is preserved rather
than inferred as success.

The narrowest demonstrated boundary is post-hydration Gate E clock/lifecycle
coordination before cycle measurement. The harness holds its injected clock at
requested end plus five seconds while the production runtime's one-second timer
is active, then moves that clock backward to requested end plus one second for
cycle 1. A clock-regression transition requires termination and seals the
engine. The observed `not_admitted_closed` proves the engine was sealed before
the explicit cycle-1 timer could link; the trace did not emit which preceding
timer performed the sealing transition. This is not a playback, hydration
throughput, queue-capacity, Activity timing, ranking, T/Q, checkpoint, or
provider failure. The unchanged Gate E is not rerun. The conditional final
acceptance review and acceptance verification do not apply, Gate F remains
unexecuted, and no credentials, live requests, deployment, push, rebase, or
amend occurred.

### S2-E bounded continuation and stop evidence — 2026-08-11

The owner authorized at most two additional full Gate E executions, only after
a compact deterministic rehearsal, one `gpt-5.6-sol` medium harness review,
focused tests, the ordinary short suite, focused race, vet, and diff checks.
The continuation also required automatic runtime timer ownership, monotonic
clock choreography, ten measured cycles, and deterministic cleanup on every
failure exit.

The corrected harness removes manual timer admission entirely. Its injected
clock rejects regression and advances from requested end plus five seconds at
the hydration fence to plus six through plus fifteen seconds for the ten
cycles. Each cycle is accepted only from one sealed production snapshot whose
watermark equals the intended target, last disposition is `timer_applied`,
publication sequence exactly equals the measured timing sequence, and
publication/operational identities agree. The same captured snapshot supplies
the API/publication assertion. Cleanup owns the attempt from successful start,
requests a controlled close, bounds graceful terminal delivery at four
seconds, cancels and bounds the post-cancel join at one second, and preserves
the remainder of one ten-second deadline for runtime timer, adapter, engine,
and writer joins. Failed preflight validation closes its artifact handle before
ownership can transfer.

`TestCachedHydrationFenceAutomaticTimerRehearsal` uses the production runtime,
automatic timer, live adapter, successful-empty hydration terminal, real
adapter ingress fence, readiness transition, ten exact logical cycles, sealed
captures, accounting, and deterministic shutdown. Its initial run rejected a
fixture chronology that captured the fence before `end+5s`; after correction
it passed, then passed 20 consecutive focused runs. The monotonic-clock
counterexample passed, as did ten focused race repetitions. The ordinary
`go test -short -timeout 2m ./...`, `go vet ./...`, and `git diff --check` were
clean before each full execution.

The required `gpt-5.6-sol` medium read-only review initially found three P2
proof defects: independently sampled timing/publication identities, cleanup
that could exhaust its deadline before runtime joins, and a preflight handle
leak. All three were corrected and the same reviewer found no remaining P1/P2.
After full execution 1, the reviewer correctly rejected disposition-coupled
fixture backpressure as a P1 false-pass mechanism. The final containment caps
only frames written but not yet read by the production adapter; read-to-
disposition lag, the 512-frame production queue, and every capacity rejection
remain unconstrained and observable. Focused re-review found that correction
clean with no remaining P1/P2.

Full execution 1 of 2 used the authorized command unchanged. Preparation
completed in `1m20.301853291s`. Hydration reached 7,500,000 rows in
`2m8.114519125s`, then the fixture's wall-clock WebSocket writer had accumulated
an unbounded transport/kernel-buffer lead over the consumer. A late stall
filled the 512-frame production queue and rejected one frame. The engine
correctly entered `suppressed/ingress_integrity`, fenced the 5,502 hydration
requests, and a subsequent chunk returned `hydration_fenced/historical_context`.
The run failed in 225.52 seconds (`real 227.49`, `user 225.23`, `sys 40.03`).
This is harness load-shaping evidence, not an Activity, playback, artifact,
hydration-data, or product-semantics failure. The adjacent cleanup correction
also treats an already-terminal attempt's rejected second close as a reason to
cancel immediately while still requiring `Runtime.Shutdown`/`Wait`.

After the transport-only correction and repeated clean fast ladder, full
execution 2 of 2 again prepared the exact 7,587,384-record prefix with
7,581,690 selected rows, 5,439 value symbols, and 63 successful-empty symbols
in `1m20.706354959s`. All 7,581,690 hydration rows completed in
`2m18.30437725s`. Every emitted 500,000-row progress point reported zero
oversize, capacity, receipt, and gate/close rejection with zero queued frames;
the last point was 7,500,000 rows at `2m0.983517375s` with 22,778 frames read,
admitted, and dispositioned.

The final permitted run then failed before completing measured cycle 1:

```text
GATE_E_HYDRATION rows=7581690 duration=2m18.30437725s
Gate E cycle 1: automatic timer target 2026-08-07T21:15:02Z: context deadline exceeded
--- FAIL: TestCachedHydrationFenceAcceptance (233.88s)
real 235.92
user 233.36
sys 43.03
```

No cycle timing, allocation, or publication vector was accepted. The exact
correlation requirement prevents inferring a cycle from another timer,
pressure/TQ publication, or independently sampled snapshot. The two additional
full-execution budget is exhausted. Per the direct stop condition, no further
correction or Gate E execution, final acceptance review, acceptance commit, or
Gate F is performed. No credential, provider request, deployment, push,
rebase, or amend occurred.

#### Cycle-1 timeout exact-cause proof — 2026-08-11

The final timeout was not timer starvation, a blocked coverage fence, mature
evaluation cost, or failure to advance `T`. The compact rehearsal originally
used a 20 ms evaluation cadence while production uses a one-second evaluation
cadence alongside the runtime's separate one-second pressure ticker. Running
the otherwise identical compact production runtime, live adapter, hydration
terminal, ingress fence, readiness, and shutdown path with the production
one-second cadence reproduced the Gate E cycle-1 timeout under the same
five-second cycle bound.

The augmented failure evidence captured both the latest immutable publication
and the engine's latest evaluation timing. Three consecutive production-
cadence runs reported the same causal shape:

```text
automatic timer target 2026-08-07T08:20:02Z: context deadline exceeded;
last publication id=29 sequence=29 disposition=tq_applied
watermark=2026-08-07 08:20:02 +0000 UTC
timing_sequence=27 prior_sequence=9 prior_publication=9
```

The target watermark proves logical time reached the requested cycle. Timing
sequence 27 proves the automatic timer completed its full evaluator boundary.
The later publication sequence 29 with `tq_applied` proves the timer
publication was replaced, not absent. Source inspection closes the remaining
causal gap: `Runtime.runTimer` creates the evaluation and pressure tickers
together; both have a one-second cadence in production. The evaluation branch
completes the timer and then returns to the same select loop. The already-ready
pressure branch admits `TQPressureTick` and its result, whose T/Q revision
replaces the single atomic publication cell. External one-millisecond polling
cannot observe the transient sequence-27 timer publication once sequences 28
and 29 have completed synchronously in that goroutine.

As a control, the original 20 ms rehearsal passed three consecutive complete
ten-cycle runs in 0.21 seconds each. At that cadence the timer publication
normally remains visible before the next one-second pressure boundary. The
only changed variable in the distinguishing failure was restoring the
production evaluation cadence. Empty hydration and no ranked T/Q rows were
sufficient, so the 7,581,690-row state, mature Activity work, and T/Q command
acknowledgement are not necessary causes.

The failed Gate E assertion therefore tested observability of a transient
atomic-cell occupant, not whether the automatic timer ran. The smallest
justified correction is to observe and seal the timer disposition, timing, and
snapshot synchronously at the runtime-owned automatic-timer completion point,
before `syncTQCommand` or another co-ready pressure branch can replace the
publication. That observer must remain test-only/read-only, must not admit a
timer or choose time, and must preserve exact target, disposition, engine
sequence, publication ID, and sealed-snapshot correlation. Polling faster or
loosening identity would not prove the required cycle.

#### Cycle-observation correction and resumed fast ladder — 2026-08-11

The runtime now exposes one package-private, read-only automatic-timer
observation seam. It is invoked only after the completion returned by
`Runtime.runTimer`'s own `AdmitTimer` and before `syncTQCommand` or another
ticker selection. When enabled by Gate E, it seals that timer disposition, the
matching evaluation timing, and one production `CaptureSnapshot`. It does not
admit a timer, choose logical time, mutate engine state, or alter behavior when
no observer is installed.

The Gate E consumer accepts an observation only when the returned timer
disposition is `timer_applied`, its engine sequence equals both the timing
sequence and sealed publication sequence, the publication watermark equals the
requested target, publication ID advances, and operational/publication
identities agree. Its bounded size-two channel is single-producer and
nonblocking; dropping evidence can cause only an honest timeout because a later
cycle has a distinct target and cannot substitute. Deferred unregistration is
race-safe and every shutdown path still joins the runtime timer.

The first corrected production-cadence rehearsal reached cycle 9 and then
exposed an adjacent deterministic harness defect: the outer rehearsal deadline
was exactly ten seconds although ten production ticks themselves require ten
seconds. The production-cadence rehearsal budget is now twenty seconds while
each individual cycle remains bounded at five seconds. This changes no Gate E
or product timing limit.

Verification after both corrections:

- the exact one-second production-cadence rehearsal passed all ten cycles in
  10.01 seconds, including under `-race`;
- the focused rehearsal plus monotonic-clock counterexample passed 20
  consecutive repetitions;
- the focused race proof passed 10 consecutive repetitions;
- `go test -short -timeout 2m ./...` passed; and
- the same `gpt-5.6-sol` medium reviewer reported no remaining P1/P2 in the
  complete affected observation boundary.

At this point no further full Gate E execution had been performed. The owner
subsequently authorized one exact corrected execution; its acceptance evidence
follows.

#### Corrected Gate E acceptance and final review — 2026-08-11

The newly authorized exact recorded command passed once:

```text
/usr/bin/time -p env CACHED_HYDRATION_ACCEPTANCE=1 CACHED_HYDRATION_RATE_MULTIPLIER=1 go test -timeout 8m ./internal/operations -run '^TestCachedHydrationFenceAcceptance$' -count=1 -v
```

Preparation validated artifact
`sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`,
the 7,587,384-record requested prefix, 7,581,690 selected hydration rows,
5,439 value symbols, and 63 successful-empty symbols in 1m20.721s. Hydration
completed in 2m13.786s. The live fixture and production adapter reported
26,963 frames sent/read/admitted/dispositioned, zero oversize/capacity/receipt/
gate-or-close rejection, and a real queue high-water of 320 against 512 slots.

All ten automatic runtime-owned cycles produced exact correlated observations.
Publication IDs increased from 38,679 through 38,715. Total measured lock time
ranged from 73.528ms to 122.177ms, with the maximum far below the fixed
two-second limit. The final view was `live` and `qualified_current` with 20
ranking rows, aggregate-only T/Q with 20 rows, reconciled accounting, readiness
true, and a deterministic shutdown. The test passed in 233.93 seconds
(`real 236.11`, `user 232.36`, `sys 40.66`).

The required final read-only `gpt-5.6-sol` medium review inspected the complete
preparation-through-shutdown diff and returned CLEAN/PASS: no actionable P1/P2,
no changed market or product semantics, no competing timer/evaluator owner, and
no false-pass route through fixture pacing or publication correlation. Its
independent production-cadence rehearsal also passed in 10.44 seconds.

Gate E is accepted for this correction. The evidence proves local cached-data
composition on this host; it does not prove provider reachability, credentials,
current-market latency, or launch safety under live market-hours conditions.
Those claims remain allocated solely to separately authorized Gate F.

#### Post-E Activity/checkpoint vertical correction — 2026-08-12

Before recommending an unattended 04:00 scanner start, the credential-free
`TestPV1RCVertical` was run on the accepted Gate E commit. It failed before
restart, adapter, API, or UI at checkpoint projection:

```text
checkpoint_projection_rejected / projection_invariant
```

The distinguishing diagnostic compared evaluations at the same committed
timestamp. The live accelerated Activity state reported
`unavailable/history_incomplete`, while the checkpoint rebuild's authoritative
full scan reported `warming/reference_warmup` with one reference.

The exact cause was the accelerated lookup's first advance. Its lookup was
initialized at the 04:00 session boundary; subtracting the 30-second target
window produced 03:59:30. Go duration division truncates negative offsets
toward zero, so `activityBlockEnd` mapped that pre-session floor to 04:00 and
the fast path evaluated a nonexistent `[03:59:30,04:00)` reference block. The
checkpoint rebuild did not use that fast traversal and remained correct.

Both accelerated traversal sites now share a helper returning the first valid
session-aligned reference end strictly after their floor, clamped to 04:00:30.
Exact in-session block ends advance to the next block; nonaligned floors select
their containing block end. Thus an existing lookup representing ends through
the old floor neither repeats nor skips a block. Activity formulas, eligibility,
percentiles, correction behavior, bounds, market time, and checkpoint schema
are unchanged.

The new regression recreates a session-start lookup, evaluates and advances at
the first +60-second boundary, and proves repeated accelerated evaluation is
stable. It passed 20 consecutive runs. The previously failing vertical then
passed through checkpoint write/load/install, checkpoint catch-up hydration,
production live A/T/Q enrichment, snapshot HTTP serialization, the production
dashboard view model, and joined shutdown. The engine/scanner suites, ten
focused checkpoint/Activity race repetitions, three vertical race repetitions,
and ordinary short suite passed. A focused `gpt-5.6-sol` medium read-only review
returned CLEAN with no actionable P1/P2 and confirmed exactly-once block
traversal and no product or persisted-state change. Gate E was not rerun because
its 17:15 hydrated boundary never exercises this first-session lookup advance.

## Sections 1-4 — Outcome, scope, ownership, and settled boundary

The narrow fix is complete when a fresh-start live scanner:

- applies each accepted aggregate to canonical state immediately but performs
  full-population qualification/ranking only at the next one-second timer or
  hydration ingress fence;
- never starts another connection attempt after the engine reaches
  `suppressed` or `ended` unless an explicit lifecycle recovery event permits
  it;
- serves coherent warming and suppressed snapshots instead of turning those
  states into an apparent API disconnect;
- sustains a representative 17:15 ET publication cadence with mean evaluator
  work below one second and no result above the existing two-second readiness
  tolerance; and
- can run the first diagnostic live milestone with checkpoint discovery and
  cadence explicitly disabled, using bounded fresh REST hydration for recovery.

Checkpoint restart remains a V1 requirement. Disabling it for this milestone
neither deletes C7 nor proves checkpoint recovery; it isolates the first live-
currentness proof from unmeasured checkpoint projection cost.

The exact controlling Phase 1 requirements are:

- `PG-RANK-02` through `PG-RANK-05`, `PG-FEATURE-02`, `PG-FEATURE-05`,
  `PG-AVAIL-01` through `PG-AVAIL-03`, `PG-OPS-01`, `PG-OPS-02`, and
  `PG-OBS-01` through `PG-OBS-03`;
- `ARCH-OWN-01`, `ARCH-OWN-03`, `ARCH-FLOW-01`, `ARCH-FLOW-02`, and
  `ARCH-FLOW-04`;
- `DTE-TIMER-01`, `DTE-MERGE-01` through `DTE-MERGE-05`,
  `DTE-RECOVERY-02` through `DTE-RECOVERY-05`, `DTE-COMMIT-02` through
  `DTE-COMMIT-04`, and `DTE-CHECKPOINT-01`;
- `LIFE-HYDRATE-03` through `LIFE-HYDRATE-06`, `LIFE-LIVE-01`,
  `LIFE-LIVE-02`, `LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`, and
  `LIFE-PUBLISH-01` through `LIFE-PUBLISH-03`; and
- accepted component requirements `C3-ACT-01`, `C3-ACT-02`, `C3-EVAL-01`,
  `C7-CADENCE-01`, `C8-RUNTIME-01`, `C8-READY-01`, `C8-RECOVERY-01`,
  `C8-MEASURE-01`, `C8-CAPACITY-01`, `C10-STATUS-01`, and `C10-HTTP-01`.

No formula, qualification threshold, ranking order, merge rule, watermark,
readiness tolerance, or suppression meaning changes.

The `ScannerStateEngine` remains the only mutable owner of canonical facts,
pending evaluation work, derived Activity state, lifecycle, watermark, ranking,
and publication. Operations may decide whether to construct checkpoint
components and must stop attempts when the engine is terminal. The API/UI only
consume immutable observations. This correction adds no product rule, mutable
owner, watermark, evaluator, T/Q-to-ranking dependency, changed interval, or
duplicated responsibility.

In scope are live evaluation coalescing; terminal lifecycle guarding;
warming/suppressed visibility; fixed-cardinality stage/apply/publication timing;
a measurement-triggered Activity lookup correction; and a live-only checkpoint
switch whose default preserves existing behavior.

Out of scope are `degraded_current` or another ranking mode; conflict,
qualification, formula, or ranking changes; replay/C12; a ranking index unless
the compact scalar scan remains the measured blocker; larger queues or looser
timeouts; and checkpoint schema/codec/storage changes.

## Sections 5-8 — Evidence, reconnaissance decision, and reuse

Create a clean correction branch/worktree from
`4cf302986c20e80116e16c82f310566db2d5d921`. Treat both dirty worktrees as
read-only patch sources until exact hunks are classified:

1. `/Users/joshuabelandres/Dev/live-equities-momentum-scanner-api-correction`
   supplies coalescing, the terminal guard, and bounded evaluator proofs.
2. `/Users/joshuabelandres/Dev/live-equities-momentum-scanner` supplies typed
   integrity, warming/suppressed visibility, operator output, and regressions.

The following existing observations justify the correction:

- the main tree evaluates the universe after every accepted aggregate, which
  scales as aggregate arrival rate times universe size and can fill the
  512-frame live queue;
- the API-correction tree coalesces work, passes the short suite, and completed
  60 sparse two-hour cycles in about 29 seconds; staging averaged 243.9 ms and
  peaked at 515.7 ms;
- Activity currently walks every eligible session-to-date 30-second reference
  block during both candidate staging and deterministic apply, implying about
  19.1 million reference-block visits per 17:15 publication at 6,000 symbols;
- the main regressions reproduce suppression-reason loss, a reconnect hot loop,
  and an unservable warm-up snapshot; and
- the prior 300-cycle/full-artifact harness failed repeatedly before producing
  accepted mature-tail evidence. Those runs are diagnostic history, not a
  reason to rerun the same expensive shape.

Partial-ranking, C12/replay, checkpoint-format, acceptance-export, and Gate D
changes are excluded. No predecessor, credential, or provider request is
authorized.

No predecessor reconnaissance is needed or permitted. Current repository code,
the two observed worktrees, deterministic regressions, and recorded timing
answer the implementation questions. The exact reuse whitelist is:

| Source | Decision | Preserved behavior / excluded coupling |
| --- | --- | --- |
| API-correction `internal/engine/evaluator.go` and `evaluation_coalescing_test.go` | Adapt | Preserve live timer/fence coalescing; exclude partial ranking, checkpoint-format, and Gate D changes. |
| API-correction `internal/operations/live.go` | Adapt | Preserve `terminalLiveStateError` before each connection attempt; exclude unrelated composition changes. |
| Main-tree evaluator/publication/operations/snapshot regression paths named in S1 | Adapt | Preserve typed first failure and coherent warming/suppression; exclude replay/C12 changes. |
| Current `internal/engine/feature_activity.go` plus accepted C3 Activity proofs | Behavior and oracle evidence | Measure first; change only reference lookup if dominant. |

The contract changes evaluator scheduling and terminal cross-component
behavior, so implementation requires focused read-only review after proof.

## Sections 9-17 — Required behavior, proofs, slices, and bounds

### Slice 1 — compose the stable live path

Port the API-correction tree's live-mode coalescing and terminal guard into the
clean target. Preserve these exact effects:

1. accepted changes update only affected canonical/support state; every live
   aggregate disposition marks one pending accounting prefix;
2. no ordinary nonintegrity aggregate admission performs a full-universe
   projection or replaces the immutable publication;
3. the next timer or hydration ingress fence evaluates the pending prefix once,
   publishes its cumulative accounting even if rows are unchanged, and clears
   the pending bit only after successful validation/apply;
4. replay retains its deterministic per-input behavior unless an existing
   replay-group boundary already coalesces it; and
5. a suppressed/ended lifecycle is terminal to `RunLive`; it cannot enter an
   implicit reconnect loop.

Port the main tree's typed integrity and warming/suppressed visibility only
after coalescing is green. Preserve the first diagnostic, exact lifecycle
reason/disposition, binding identity, empty current rows, and HTTP 200 coherent
snapshot for warming or installed-binding suppression. `/readyz` remains 503
for both. Do not port C12/replay changes.

Add a live-only checkpoint switch. With checkpoint mode off, do not construct a
store/writer, discover/install a checkpoint, or submit cadence projections.
Report checkpoint status honestly as not installed with zero work. Default
behavior remains checkpoint mode on.

Primary proof `P-NARROW-LIVE` combines:

- `TestLiveAggregateEvaluationCoalescing`;
- `TestIngressSuppressionReasonSurvivesTimerAndRejectedReconnect`;
- `TestSuppressedIngressCannotEnterReconnectHotLoop`;
- `TestLiveWarmupSnapshotIsServableAndBound`; and
- `TestSuppressedEvaluatorIntegrityRemainsServable`, the existing operator
  output suite, and focused UI suppression model test; plus
- one production-composition test proving 1,000 accepted aggregates create no
  full-population evaluation/publication until one timer, then reconcile exact
  admission/disposition/population accounting with zero queue rejection.

It rejects a ready snapshot that hides accepted work and a suppressed reconnect
loop. It does not establish late-session capacity or provider behavior.

### Slice 2 — remove only the measured mature-session cost

First measure one 17:15 ET stage/apply over a generated 6,000-symbol state with
1,590 session blocks per symbol: 1,589 references plus the target. Report stage,
apply, publication, allocations, and total lock duration separately.

If mean work is below one second and every trial is below two seconds, make no
Activity representation change. If Activity reference traversal dominates,
replace only that traversal with maintained correction-aware reference
summaries and two bounded ordered value collections, one for transactions and
one for expansion. A touched 30-second block removes its old eligible values,
recomputes from canonical evidence, and inserts its new eligible values.
Percentile lookup counts values `<= target` with inclusive ties. The existing
1,920-block summaries remain the semantic and checkpoint authority; ordered
collections are derived state and are rebuilt/validated after checkpoint
install rather than persisted as a second truth.

This keeps the simple full-universe scalar ranking scan. A deterministic second
apply pass is acceptable only when it no longer repeats a session-length walk.
Do not add a `U`-sized mutable candidate image or a ranking index to avoid that
pass.

Primary proof `P-NARROW-MATURE` requires:

- full-history-oracle equality for Activity after insert, correction,
  withdrawal, exact tie, eligibility loss/restoration, stalled `T`, and
  checkpoint rebuild;
- one generated 6,000-symbol 17:15 publication with stage/apply attribution;
- 60 exact one-second cycles with mean total evaluator work below one second,
  maximum below two seconds, zero aggregate rejection, no queue growth, and no
  retained-state growth with publication count; and
- checkpoint-off production composition with API polling and honest checkpoint
  status.

This is current-host evidence for the stated shape, not provider latency, an
SLA, or trading edge.

### Failure, trust, accounting, and boundedness

| Boundary | Accept into success only when | Contain/reject | Dangerous false success |
| --- | --- | --- | --- |
| Pending aggregate prefix | Every accepted aggregate is in canonical state and its final disposition appears in the next timer/fence publication | Validation failure retains pending work and enters existing integrity containment | Ready publication silently omits already accepted work |
| Terminal runtime | Engine observation is nonterminal or an explicit lifecycle recovery transition authorized another attempt | `suppressed`/`ended` returns without opening a socket | Reconnect hot loop after fail-closed suppression |
| Activity index | Rebuilt values exactly equal eligible block summaries and oracle output at the same `T` | Mismatch makes Activity unavailable/invalid under existing containment; it never changes rank | Fast percentile from stale pre-correction values |
| Checkpoint-off mode | No store/writer exists and all checkpoint counters remain zero | Reject contradictory configuration at startup | UI implies checkpoint recovery while none ran |

Existing identities remain authoritative: aggregate consumed dispositions,
population partitions, hydration terminal work, and checkpoint writer accounting
must each reconcile exactly. Activity timing, reference count, lookup work,
queue high-water, and API latency are fixed-cardinality overlapping diagnostics,
not new primary populations. Pending work is one boolean/prefix boundary;
Activity retains no more than the existing 1,920 block summaries plus two
derived collections of at most 1,920 finite values per symbol. All waits,
trials, cycles, queues, and commands use explicit bounds.

### Mandatory fast-to-long proof ladder

Every gate must pass before the next. A failed gate is reduced and corrected;
an unchanged failed command is never rerun.

| Gate | Maximum processing time | Proof and release condition |
| --- | ---: | --- |
| A — focused semantics | `< 2m` per command | Run the four direct regressions, coalescing, operator/UI, Activity oracle, and checkpoint-off proof. |
| B — ordinary repository | `2m` | `go test -short -timeout 2m ./...`. Long/capacity/live tests must skip under `testing.Short()`. |
| C — mature synthetic single boundary | `< 2m` | Build the 6,000-symbol/17:15 generated state, validate exact symbol/reference/target counts, then time three stage/apply trials. |
| D — mature synthetic 60 cycles | `< 2m` | Run only after Gate C meets the one-/two-second limits. Validate all planned corrections, timers, API polls, and expected publications before timing. |
| E — cached real-data composition | `<= 8m`, one run | Only after A-D are green, validate the sealed artifact path, byte size, digest, binding, reference dates, interval, and expected record/symbol counts. Preload once to 17:15, execute ten exact cycles, and measure the same segments. Do not rerun after presentation-only changes or unchanged failure. |
| F — current-date live observation | `15m`, separately authorized | Only after A-E and final review. Run checkpoint-off fresh bootstrap once, stop at the first typed integrity/capacity/readiness failure, preserve the diagnostic, and do not retry in the same session. |

Gate E may use the existing sealed aggregate artifact and current reader only.
It must not add another JSONL parser. Before its potentially longer parse, a
sub-two-minute preflight checks exact file/reference paths, byte size, recorded
manifest, and a compact artifact fixture through the production reader.

Gate F requires explicit authorization for its trading date, credentials,
duration, and diagnostic path under `docs/market-hours-validation.md`. The
command must use the new checkpoint-off option. Success requires terminal
hydration, an exact reconciled ingress fence, advancing one-second watermark,
zero queue-capacity rejection, no suppression, API/UI reachability throughout,
and honest independently available T/Q fields.

## Sections 18-19 — Acceptance and drift audit

Production edits are limited to the narrow paths needed in:

```text
internal/engine       coalescing, terminal diagnostic, Activity cost, fixed metrics
internal/operations   terminal lifecycle guard, checkpoint-off composition
internal/snapshotapi  warming/suppressed coherent mapping
cmd/scanner           checkpoint switch and existing operator output
ui                    suppression/API-transport presentation only
docs                  correction ledger and authority map
```

Tests may extend the corresponding package test files. Do not edit replay,
replayartifact, replaymode, checkpoint codec/store/writer, partial-ranking, or
C12 paths.

After deterministic behavior stabilizes, run the narrowest affected race
packages with an explicit timeout no greater than five minutes, then `go vet
./...` and `git diff --check`. A focused read-only review is required because
the correction touches atomic evaluator scheduling and terminal lifecycle
containment. Review must confirm one owner/evaluator/watermark, no weakened
validation, no accepted aggregate loss, correct suppression visibility, and no
session-length work on aggregate admission.

Record slice acceptance in this document with exact commands, durations,
manifest, stage/apply attribution, queue/rejection/accounting results, review
findings, and limitations. The correction is accepted for the fresh-start live
milestone only after Gates A-E and the focused review pass. Gate F remains a
separate market-hours confirmation and may be pending; a failure reopens only
the typed boundary it distinguishes.

Completion also requires resolved links, one owner/evaluator/publication path,
whitelist conformance, slice walkthroughs, and a clean drift audit:

| Question | Answer | Evidence |
| --- | --- | --- |
| New product rule, owner, watermark, evaluator, or T/Q ranking gate? | No | Scheduling and derived representation only; fixed C3 truth remains authoritative. |
| Changed time, window, correction, merge, readiness, or suppression meaning? | No | Existing timer/fence, Activity, two-second tolerance, and lifecycle rules are unchanged. |
| Unevidenced edge behavior or predecessor authority? | No | Every edge comes from the recorded live incident, deterministic regression, accepted invariant, or current timing. No predecessor is used. |
| Duplicated responsibility or unnecessary machinery? | No | One pending prefix, optional construction of existing checkpoint components, and two bounded Activity value collections; no new service, queue, or framework. |
