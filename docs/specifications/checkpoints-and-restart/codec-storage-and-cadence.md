# Checkpoints and restart — codec, storage, and cadence

**Parent contract:** [checkpoints-and-restart.md](../checkpoints-and-restart.md)
**Normative responsibility:** Bounded encoding/decoding, durable atomic local
storage and discovery, asynchronous write-result identity, cadence,
final-checkpoint policy, and restart objective
**Controlling requirements:** `C7-CODEC-01`, `C7-STORE-01`,
`C7-CADENCE-01`, `C7-OBJECTIVE-01`; `PG-OPS-01`, `ARCH-OWN-02`,
`ARCH-FLOW-01`–`04`, `DTE-CHECKPOINT-01`, `DTE-CHECKPOINT-03`,
`DTE-EVENT-01`, `DTE-EVENT-04`, `DTE-REJECT-01`, `LIFE-INIT-05`,
`LIFE-LIVE-04`, `LIFE-END-02`, `LIFE-END-03`
**Allocated slices:** `C7-S2` owns `C7-CODEC-01`/`C7-STORE-01`/
`C7-CADENCE-01`; `C7-S3` owns `C7-OBJECTIVE-01`
**Document dependencies:** Parent; [checkpoint state and installation](checkpoint-state-and-installation.md);
Components 1–3 as routed by the parent
**Approval state:** Inherits the parent contract approval; not independently approved
**Delivery state:** See the authoritative parent delivery-state ledger; do not copy mutable status here

## 9. Detailed semantic inputs, outputs, and owned state

### 9.1 Current Version 1 format

The accepted S2 implementation currently uses a standard-library JSON envelope named
`scanner-checkpoint-v1`. It contains a fixed header, the raw JSON semantic
payload, and a lowercase SHA-256 digest of the exact payload bytes. On-disk
payloads use structs and deterministically ordered slices only; maps and
polymorphic/dynamic fields are prohibited. Symbols and all identity-bearing
subrecords use their dependency-defined canonical order, making equal semantic
images byte-identical except for explicitly variable header fields such as
sequence and creation time.

The decoder reads through an `io.LimitedReader`, rejects duplicate object keys
at every depth, disallows unknown fields, validates declared lengths before
allocating, checks every dependency cardinality and aggregate numeric bound,
requires exactly one complete JSON value plus trailing whitespace, and verifies
the payload digest and manifest size/digest. It incrementally builds a detached
candidate; it never retains both an unbounded byte buffer and a fully decoded
object graph.

There is no in-place schema migration. Unknown schema or manifest versions are
`incompatible`, not corrupt, and selection proceeds to the previous candidate
or fresh initialization. JSON and the current decoder are lower-level V1
delivery choices, not Phase 1 requirements. They may be optimized or replaced
through the V1 correction loop if the replacement preserves complete semantic
state, strict bounded validation, invalid-artifact containment,
latest/previous fallback, and deterministic compatibility behavior.

### 9.2 Storage and manifest

The store is one configured absolute, canonical, local directory. Construction
rejects an empty/relative path, symlink directory, non-directory, or bindingless
configuration. It creates private directories as `0700` and files as `0600`.
No path from artifact content is trusted.

Immutable generation files and one manifest are used:

```text
checkpoint-<20-digit-sequence>.json
manifest.json
```

The versioned manifest contains exactly `latest` and optional `previous`
entries. Each entry carries a generated basename, byte length, payload SHA-256,
binding identity, `T0`, and positive sequence. Basenames must match the exact
generation filename grammar; separators, absolute paths, symlinks, hard links
with unexpected link count, nonregular files, and directory escape reject the
candidate. Discovery never guesses from modification time or scans generation
files to find an unmanifested checkpoint.

Load attempts `latest`, then `previous`. Missing/invalid/incompatible latest
does not mask a usable previous candidate. Missing/invalid manifest produces
`unavailable`; orphan/temp files never become authority.

### 9.3 Write protocol and asynchronous identity

For one immutable request `(binding identity, request ID, artifact sequence,
T0)`, the single sequential writer:

1. creates a same-directory unique temporary file with `0600`;
2. streams the bounded envelope while computing byte count and SHA-256;
3. syncs and closes it;
4. reopens and completely validates it through the same loader path;
5. renames it to its immutable generation basename and syncs the directory;
6. writes, syncs, closes, reopens, and validates a temporary manifest naming
   new latest and old latest as previous;
7. atomically renames the manifest and syncs the directory; and
8. performs bounded cleanup that can never alter manifest-referenced files.

Before manifest rename, any failure leaves the old manifest authoritative.
After manifest rename, the new manifest is the visible authority even if the
final directory sync reports an ambiguous durability failure; it still names
the previous complete generation and the loader accepts the new generation only
after full validation. The result reports the exact failed/ambiguous step.

Temporary files are removed on the request's ordinary failure/cancellation
path. A generation renamed before manifest publication is an orphan and is
ignored. Cleanup examines at most 256 matching entries and removes at most 16
unreferenced, nonactive files per attempt. More matching entries or an unsafe
entry fails persistence closed; it does not trigger an unbounded scan or affect
live evaluation.

The writer owns only request-local bytes/files and a FIFO of capacity one in
addition to the single in-progress request. Submitting while one request is in
progress may fill the pending slot; a newer valid submission replaces that
pending request and returns an explicit `superseded` terminal outcome for the
old request. Thus at most two detached views exist outside the engine: one
writing and one pending. No request is canceled merely because a newer `T0`
exists.

Every terminal outcome is one immutable fact carrying the exact request and
binding identity: `completed`, `failed(step,reason)`, `canceled`, or
`superseded`. The engine accepts operational status only for a known outstanding
request and records each request terminal once. Duplicate, unknown, wrong-
binding, or ended-engine results are fenced diagnostics and cannot alter `T`,
canonical/evaluator state, lifecycle, or checkpoint selection.

### 9.4 Bounds and configuration

| Bound | Contract value |
| --- | ---: |
| Manifest candidates | 2 |
| Manifest bytes | 64 KiB |
| Absolute artifact bytes | Configured positive limit, hard ceiling 4 GiB; no C7 production default |
| Symbol records | Exact binding population; 6,000 normal V1 reference, 100,000 structural-admission ceiling only |
| Symbol bytes | Component 1 maximum 64 |
| Semantic structure lengths | Exact Component 2/3 bounds in the state/install detail |
| Writer concurrency | 1 in progress |
| Pending/coalescing capacity | 1 replaceable request |
| Matching-directory entries inspected/removed per cleanup | 256 / 16 |
| Per-operation deadline | Configured positive duration, hard ceiling 60 seconds; cancellation checked during streaming and durability steps |
| Checkpoint cadence | At most once per 30 seconds of committed aggregate time |

The artifact-byte and operation limits are required constructor inputs because
Component 8 owns the deployed capacity choice. C7 supplies the absolute safety
ceilings and proves small-limit rejection plus the reference objective below.
Crossing any bound rejects/defer persistence without truncating a supposedly
complete checkpoint. A configured value that cannot contain the reference
fixture or meet its objective fails C7/C8 acceptance rather than weakening
semantic completeness.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C7-CODEC-01` | Encode and incrementally decode the complete semantic image with deterministic ordering, strict fixed schema, exact structural bounds, checksum, cancellation, and no partial candidate on any syntax/integrity/semantic-preflight failure. | `ARCH-FLOW-01`–`04`, `DTE-REJECT-01`; V2 strict JSON/digest/limited-reader and maximum-shape behavior adapted to the new complete schema |
| `C7-STORE-01` | Select only manifest-authorized latest/previous local immutable generations and publish a new generation through the exact temporary-file/validation/rename/sync protocol while retaining the previous complete candidate under every injected write step. | `PG-OPS-01`, `ARCH-OWN-02`, `DTE-CHECKPOINT-03`, `LIFE-INIT-05`; V2 store and exact-step fault tests adapted |
| `C7-CADENCE-01` | In live operation, when committed/evaluated `T` reaches the next session-aligned 30-second checkpoint boundary, project and submit without blocking ordinary evaluation. Bound external views to one writing plus one replaceable pending; every request gets one terminal result. | `ARCH-FLOW-01`–`04`, `LIFE-LIVE-04`, `LIFE-END-02`; V2 30-second cadence and one coalescing slot |
| `C7-OBJECTIVE-01` | On the recorded local host, a validated 6,000-symbol, checkpoint-age-30-seconds fixture must restart through real discovery/decode/semantic validation/install, C5 acknowledgement, C6 `[T0,R)` work/fence, and ordinary evaluation within the current 60-second setting on every measured trial. Its median must be at least 20% faster than equivalent fresh recovery for the same binding, `R`, provider fixture, worker limits, and trial conditions. Local load/install is segmented diagnostic evidence, not an independent five-second release gate. These program-selected settings are revisable only from recorded measured evidence; any replacement must remain materially faster than equivalent fresh recovery and below the product's rejected approximately 130-second precedent. | `PG-OPS-01`; the product rejects approximately 130-second normal fresh reconstruction; current S3 evidence completes restart in 22.53 seconds while the inherited five-second load premise fails; owner V1 correction requires meaningful same-host improvement |

The objective is a component release target, not a live-provider SLA or
100,000-symbol capacity claim. Before timing, validate and record OS/
architecture/CPU, Go version, fixture schema/hash/bytes, exact 6,000 symbols,
checkpoint age, correction count, required canonical-state families, planned
requests/records, worker/configuration bounds, and fresh-control equivalence.
Use one unmeasured warm-up if needed and three measured trials. Each measured
checkpoint and fresh trial has its own two-minute deadline; the complete
command has an explicit timeout no greater than 15 minutes.

Record median/maximum and projection, encode, write, load, install,
acknowledgement, catch-up/fence, end-to-end, and fresh-control segments. Every
checkpoint trial must satisfy the current 60-second end-to-end setting, and
checkpoint median must be at most 80% of fresh median. Local load/install time
is a correction/optimization input: if it causes the end-to-end or comparison
gate to fail, optimize/replace the decoder or codec and rerun the narrow proof.

The corrected reference fixture has exactly 6,000 sorted binding symbols with
valid prior-close facts, a semantic record for every symbol, and every
success-bearing state family exercised by `P-C7-STATE`. It includes at least
one maximum-bound correction tail, price/range state, Activity folded-target
state, qualification proof/dirty state, each slot-evidence class, final latch,
and invalid/localized consequence; remaining records may use explicit empty
sparse state. Its C6 continuation plans all 6,000 symbols over exactly 30 whole
seconds and uses the accepted fake acquisition/mapping/ledger/fence paths, not
a mocked completion. This is a correctness-and-orchestration reference shape,
not a claim about production print density; exact artifact cardinalities make
that limitation auditable.

The 100,000-symbol largest-valid structural admission/allocation case remains
isolated inside `P-C7-CODEC`. It is skipped by `testing.Short()`, has an
explicit deadline, runs only by an acceptance/capacity command after codec
changes, and is never repeatedly materialized across the ordinary suite or the
6,000-symbol restart proof.

### 10.1 Cadence and end behavior

Cadence is based on committed market-time `T`, not wall-clock polling, file
mtime, generated time, receipt time, or `checkpoint_created_at`. At most one
projection is created for a given `(binding,T)`. A same-`T` correction after a
request may be captured only by a later cadence boundary; it does not mutate an
already detached view.

Projection/submit failures and pending replacement are observable but do not
delay commit/evaluation. A writer success records availability only; it cannot
make a scanner current.

C7 version 1 does not force a new synchronous final checkpoint on controlled
stop. An already submitted cadence request may finish only within the later
Component 8 shutdown/drain policy. Cancellation yields a terminal outcome;
late results cannot mutate an ended engine. This is the simplest policy that
does not make filesystem latency part of the engine's end transition.

### 10.2 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Artifact bytes | Limited complete envelope, supported version, strict structure, digest, counts, and detached semantic preflight all pass | No candidate; try previous/fresh | A truncated or duplicate-key payload parses to default fields and passes checksum/shape |
| Manifest candidate | Exact safe basename, regular private file, manifest size/digest/binding/`T0`/sequence match, and full decode pass | Skip latest; try previous; unsafe directory fails persistence only | Modification time or orphan file is selected despite never-published manifest authority |
| Published write | Reopened generation validates, manifest rename is complete, and outcome names exact durability step | Old manifest remains authority before rename; post-rename ambiguity is explicit and loader revalidates | A partial generation replaces latest or a failed write deletes the only prior usable artifact |
| Writer result | Exact outstanding request/binding identity and first terminal disposition | Fence duplicate/stale/unknown/ended result | Completion for an older coalesced request is reported as the newest checkpoint |
| Objective evidence | Validated 6,000-symbol fixture/config/host, three individually bounded measured trials, equivalent fresh controls, all segments, and the current 60-second/20%-improvement settings—or a recorded measured replacement that satisfies `C7-OBJECTIVE-01`'s materially-faster-than-fresh and below-130-second bounds | A failed attempt enters the V1 correction loop; semantic state cannot be silently reduced to tune the result | A tiny sparse artifact, mismatched fresh control, unbounded trial, or mocked catch-up is presented as proof of normal restart |

## 11. Failure and terminal behavior

Load outcomes are `loaded_latest`, `loaded_previous`, `unavailable`,
`incompatible`, `invalid`, or `canceled`, with per-candidate bounded reasons.
Only a fully decoded candidate reaches engine installation; `loaded_*` does not
mean installed until `C7-INSTALL-01` succeeds.

Write step failures are exact: temp create, payload encode/limit, file sync,
file close, reopen validation, generation rename, directory sync, manifest
create/encode/sync/close/reopen validation/rename/directory sync, or cleanup.
Cleanup failure after successful manifest publication reports completed-with-
cleanup-deferred, because it cannot revoke the already authoritative complete
generation. Cancellation is checked during encode/decode and before each
durability transition; it never converts a partial file to manifest authority.

Filesystem permission/path/symlink violations, oversize artifacts, deadlines,
and writer saturation are persistence-only. They remain visible to Component 8
but never freeze aggregate evaluation or fabricate a fresh checkpoint result.

## 12. Accounting and observability

Writer accounting at every pause point is:

```text
submitted = in_progress + pending + completed + failed + canceled + superseded
in_progress <= 1
pending <= 1
```

Each request occupies exactly one terminal bin. `cadence_ineligible`,
`projection_rejected`, and `submit_rejected` are attempt dimensions, not
submitted requests, and are reported separately. Loader candidate accounting is
`manifest_candidates = loaded + incompatible + invalid + unavailable +
canceled + unattempted_after_success`, with at most two candidates.

Fixed reasons and exact step enums are bounded. Paths, symbols, checksums,
errors, and request IDs may appear in bounded diagnostic records/logs but not
metric labels. Expose last successful `T0`, artifact age/bytes, last attempt/
terminal reason, queue occupancy, and cumulative terminal totals as restart-
local operations facts; none is ranking time/currentness by itself.

## 13. Simplicity and boundedness

- One codec, one local store, one manifest, one sequential writer, and one
  capacity-one pending slot are sufficient. No database, remote storage,
  journal, generic codec registry, background scanner, or retry framework is
  introduced.
- JSON is the current accepted implementation because V2 supplied strict-
  validation/failure evidence and the standard library sufficed for S2
  correctness. It is not frozen: measured V1 evidence may justify a simpler
  decode path or another bounded format, provided persisted trust and fallback
  proofs are preserved or corrected.
- Latest plus previous gives one corruption/write fallback without unbounded
  generations. Orphans are never discovery candidates.
- V2's strict JSON, checksum, bounded reader, immutable generation/manifest,
  reopen validation, exact-step fault injection, and one coalescing slot are
  adapted. V2's 64 MiB production assumption, combined owner polling/token,
  derived-only schema, 16-minute seal lag, and universal 5-minute catch-up
  ceiling are rejected.
- A single in-progress request without a pending slot was simpler but can leave
  restart age unbounded behind one slow write. One replaceable pending view is
  the minimum finite freshness mechanism supported by V2 evidence.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Duplicate keys at nested JSON levels | V2 focused regression | Strict rejection before candidate success | `P-C7-CODEC` |
| Maximum declared lengths and transient allocation pressure | V2 192 MiB maximum-shape test; current dependency bounds | Preflight each length, enforce byte/cardinality limits, and show post-run retained plateau | `P-C7-CODEC` |
| Failure at every file/manifest durability step | V2 exact-step fault test | Before manifest publication old remains authoritative; after publication new is revalidated and previous retained | `P-C7-STORE` |
| Latest corrupt, previous valid; missing manifest | V2 fallback regressions | Load previous in first case; fresh path in second | `P-C7-STORE` |
| Cancellation during streaming write/load | V2 cancellation regression | No partial authority/candidate; one canceled terminal | `P-C7-CODEC`, `P-C7-STORE` |
| Writer slower than two cadence boundaries | V2 coalescing evidence; finite-work invariant | Keep in-progress, replace only pending, terminally supersede old pending | `P-C7-CADENCE` |
| Stop while write is pending/in progress | Lifecycle invariant | No forced final projection or unbounded wait; cancellation/late result is contained | `P-C7-CADENCE` |
| Reference checkpoint age and restart timing | Current 4.39 MB run: load 9.82 s, install 63 ms, catch-up 12.64 s, end-to-end 22.53 s; `PG-OPS-01`; owner V1 correction | Validate the 6,000-symbol fixture before timing; use bounded trials and equivalent fresh controls; meet the current 60-second/20%-improvement settings—or a recorded measured replacement that satisfies `C7-OBJECTIVE-01`'s materially-faster-than-fresh and below-130-second bounds; treat local segments as diagnostics | `P-C7-OBJECTIVE` |
