# Checkpoints and restart — checkpoint state and installation

**Parent contract:** [checkpoints-and-restart.md](../checkpoints-and-restart.md)
**Normative responsibility:** Exact coherent projection, schema semantics,
compatibility/validation, atomic install, live/replay restart handoff, and
engine-owned checkpoint facts
**Controlling requirements:** `C7-STATE-01`, `C7-INSTALL-01`, `C7-LIVE-01`,
`C7-REPLAY-01`; `PG-OPS-01`, `PG-OPS-02`, `PG-REPLAY-01`,
`ARCH-OWN-01`–`04`, `DTE-SESSION-02`, `DTE-CLOCK-05`,
`DTE-COMMIT-02`–`04`, `DTE-CHECKPOINT-01`–`03`, `LIFE-INIT-03`–`05`,
`LIFE-HYDRATE-01`–`06`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`
**Allocated slices:** `C7-S1` owns `C7-STATE-01`/`C7-INSTALL-01`;
`C7-S3` owns `C7-LIVE-01`/`C7-REPLAY-01`
**Document dependencies:** Parent; Components 1–4 and 6 as routed by the parent
**Approval state:** Inherits the parent contract approval; not independently approved
**Delivery state:** See the authoritative parent delivery-state ledger; do not copy mutable status here

## 9. Detailed semantic inputs, outputs, and owned state

### 9.1 Engine-owned projection

`ScannerStateEngine` may form a checkpoint view only while its installed
binding is complete, its lifecycle permits checkpointing, and its current
aggregate evaluation names the same nonzero committed `T0`. The view is a deep,
immutable, semantic copy of state required to reproduce `[S,T0)`. It is not a
serialization of private Go structs and cannot retain aliases to the engine.

The projection is explicitly **as of `T0`**. Forward sufficient state admitted
after `T0` while the commit gate was stalled is not checkpoint state. The
projector must clip or reconstruct each affected canonical, coverage, feature,
qualification, and accounting structure at `T0`; copying a current map whose
contents happen to include post-`T0` evidence is invalid.

| Item | Required semantic content | Bounds and ownership |
| --- | --- | --- |
| Header | Schema version; `producer_mode=live_aggregate`; exact Component 1 binding identity and all compatibility-bearing binding fields; `T0`; `checkpoint_created_at >= T0`; sequence; exact population and structure counts | One immutable header; engine constructs binding and time facts; valid for same-binding live or aggregate-replay load |
| Canonical correction state | Every canonical aggregate identity `< T0` still represented in Component 2's correction tail, with original finite OHLC/volume/VWAP/ATS value and ATS provenance; the retained older mark and committed-latest mark needed at `T0` | Existing `<=961` tail records plus one older mark per active symbol; no synthetic aggregate or flat OHLC reconstruction |
| Slot evidence | Exact present, hydration-proven-absent, and historical-conflict bits in `[S,T0)`; uncertainty origin needed by current consequences | Three lazy session bitmaps per applicable symbol, clipped at `T0`; bitmap allocation alone proves nothing |
| Price/range state | Earliest-open/finalized-through and rolling floor, prefix/session extrema facts, monotone extrema sequences, invalid/bound flags, and other Component 3 sufficient inputs needed to produce the result at `T0` and continue after it | Existing Component 3 mathematical bounds; semantic ordered slices, not private containers |
| Activity state | Final reference summaries, mutable exact contributions, folded-target blocks and identity-positioned contributions, pruning floor, and invalid/bound flags needed for every permitted later nonaligned target | Existing 119-reference, 33-mutable, 1,920-block, and 57,600-contribution per-symbol bounds |
| Qualification state | Finalized gate bars, mutable proofs, dirty identities, accounted/finalization boundaries, finalized latch and proof end, unresolved origin, and invalid/bound flags | Existing 961-proof/dirty and finalized-overlap bounds; provisional success is never inferred from a Boolean alone |
| Evaluator support | Success-bearing per-symbol invalid-mark evidence and coverage consequence needed to regenerate the exact population and field dimensions at `T0` | At most one bounded record of each kind per bound symbol |
| Rebuilt values | Feature/qualification result caches, evaluator counters, ordered rows, total passers, and immutable publication | Not persisted; rebuilt and validated by the ordinary Component 3 evaluator during installation |

Engine-run lifecycle, live connection epoch/acknowledgement, active hydration or
recovery generation/ledger/fence, replay scheduler state, admission/engine
sequence, publication identity, writer state, cumulative runtime diagnostics,
T/Q membership/coverage/windows, credentials, URLs, paths, raw provider frames,
and REST bodies/pages are prohibited payload. Restart-local diagnostic counters
begin at zero; semantic population/accounting support is restored or rebuilt.
Version 1 produces checkpoints only from the live cadence. Replay may read a
same-binding `live_aggregate` image but does not create another checkpoint
writer/cadence path.

### 9.2 Candidate and installed facts

The decoder returns one detached candidate with no writable engine reference.
It carries parsed semantic values, exact counts, integrity result, and artifact
identity. A candidate is single-use: install consumes it or rejects it.

Successful install produces one engine-owned installed-checkpoint fact:
`(binding_identity, schema_version, T0, artifact_sequence, checksum)`. This fact
selects Component 6's already-approved checkpoint plan after a new current-epoch
aggregate acknowledgement fixes `R`; it does not itself prove `[T0,R)`, live
coverage, currentness, or readiness.

### 9.3 Construction and runtime validation

Construction prevents a writer/decoder from mutating engine state, projection
from sampling two transitions, a partial candidate from being installed, maps
or interfaces with unbounded/dynamic field vocabulary in the artifact, and
restored process-local source positions from impersonating a new epoch.

Runtime validation must still reject unsupported schema/run mode; any binding
field or identity mismatch; zero, non-whole-second, out-of-session, or
incoherent `T0`; `checkpoint_created_at < T0`; unsorted/duplicate/foreign/missing
symbols; counts beyond the binding or dependency bounds; aggregate identity,
finite-value, ATS-provenance, half-open-window, bitmap, feature, qualification,
or accounting contradictions; post-`T0` contributions; and any regenerated
candidate whose Component 3 semantic validator fails.

## 10. Required behavior

| Requirement | Behavior | Controlling authority and evidence |
| --- | --- | --- |
| `C7-STATE-01` | From one fully committed/evaluated boundary, project a detached, bounded, complete semantic image of Components 1–3 exactly at `T0`, excluding post-`T0`, ephemeral transport/TQ, cached-output, and runtime-only state. Projection failure changes no canonical state or `T` and emits no writable request. | `PG-OPS-01`, `ARCH-OWN-01`–`04`, `DTE-CHECKPOINT-01`, `LIFE-LIVE-04`; current C2/C3 invariants; V2 detached-image behavior evidence with its incomplete derived-only payload rejected |
| `C7-INSTALL-01` | Validate the whole candidate against a newly installed exact binding, rebuild dependency-owned candidate state and ordinary evaluation at `T0`, then atomically install all of it through the sole engine owner or install none. Restored canonical values are a restart baseline, not old live/historical causal positions. | `DTE-SESSION-02`, `DTE-COMMIT-02`–`04`, `DTE-CHECKPOINT-01`–`03`, `LIFE-INIT-03`; V2 all-or-none detached install adapted without its owner/token architecture |
| `C7-LIVE-01` | A valid install at `T0` selects Component 6 `checkpoint_catchup`; the new acknowledgement fixes `R`, C6 covers exactly `[T0,R)`, live evidence reconciles through its fence, and the ordinary evaluator reproduces the uninterrupted aggregate-derived state. T/Q and all live epochs start empty/new. | `PG-OPS-01`, `PG-OPS-02`, `DTE-CHECKPOINT-02`, `LIFE-HYDRATE-01`–`06`; finally accepted C6 interface |
| `C7-REPLAY-01` | Replay may install the same mode-compatible semantic image, require the replay artifact's first continuation identity to be exactly at or after `T0`, and continue via the ordinary Component 4 path. If the checkpoint is rejected, replay restarts at `S` only when the artifact proves the complete `[S,E)` input required by Component 4. | `PG-REPLAY-01`, `LIFE-REPLAY-01`, `LIFE-REPLAY-02`; accepted Component 4 artifact/order contract; no dedicated V2 replay-checkpoint evidence found |

### 10.1 Restored canonical authority

Checkpoint provenance is an installation class, not a provider source class or
persisted causal position. On a new run:

1. exact duplicate input against a restored identity is an ordinary duplicate;
2. a valid new-epoch live unequal value may revise a restored identity while
   Component 2's inclusive correction rule still permits it;
3. Component 6 historical catch-up starts at `T0`, so it cannot overwrite the
   restored prefix; and
4. replay rejects a continuation that repeats or precedes the restored prefix.

Old connection epochs, live frame positions, hydration request positions, and
replay delivery positions are never compared with new-run positions. Persisting
or reactivating them would allow arrival ordering from a dead process to reject
valid new evidence and is prohibited.

### 10.2 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Engine projection | One locked transition proves binding, lifecycle, nonzero `T0`, evaluator-at-`T0`, complete clipped sufficient state, and all counts/bounds | Projection failure is persistence-only; live evaluation continues | A stalled engine copies a newer tail or folded contribution beside the older committed evaluation |
| Decoded candidate | Codec integrity is valid and the engine independently validates every compatibility/semantic invariant and regenerated evaluation before visibility | Reject candidate and try older artifact/fresh path; no partial engine mutation | Valid checksum over internally inconsistent state is treated as canonical success |
| Restored baseline | Values are real accepted normalized aggregates with exact identity/value/provenance and no old process causal position | Reject synthetic/derived mark reconstruction and any reactivated epoch/request position | A persisted close is expanded to fake OHLC/volume/ATS and later qualifies a symbol |
| Live continuation | Exact installed fact, new acknowledgement, C6 `[T0,R)` terminal accounting, and reconciled live fence reach the ordinary evaluator | C6 owns local/global failure consequence; installed prefix remains noncurrent | File load alone is labelled live/current or a historical result overwrites restored prefix |
| Replay continuation | Binding/schema/mode match and artifact cutoff proves no event `<T0` will be delivered | Reject checkpoint; fall back only with complete-from-`S` evidence | Duplicate pre-`T0` replay events silently revise or double count restored state |

## 11. Failure and terminal behavior

Projection has exactly `projected` or `rejected(reason)` terminal disposition.
Candidate installation has `installed`, `incompatible`, or `invalid`; absence,
I/O/cancellation, and unsupported schema are loader dispositions and never
reach partial install. Every candidate is consumed once.

A failed projection, encoding, or write is persistence-only and cannot suppress
ranking or block `T`. An invalid newest artifact permits deterministic attempt
of the previous candidate. If neither candidate installs, initialization uses
the ordinary fresh path with a bounded reason. A checkpoint contradiction
does not invalidate an independently constructed current binding; only an
existing Component 1/engine binding ambiguity uses its pre-existing global
integrity path.

After installation, later catch-up failure is Component 6 failure—not a reason
to roll back to another checkpoint. The installed prefix remains canonical but
noncurrent according to the existing lifecycle/evaluator consequences.

## 12. Accounting and observability

The checkpoint semantic population obeys:

```text
binding_symbols = serialized_symbol_records
serialized_symbol_records = records_with_state + empty_state_records
```

Every structure header count equals the sum of the exact per-symbol lengths
decoded and validated. Omitted sparse allocation is distinct from omitted
semantic state: an empty record is explicit and a missing bound symbol is
invalid. Format, storage, and writer accounting is owned by the codec detail.

Bounded reasons distinguish projection ineligible, projection bound/invariant,
unsupported schema, binding/mode mismatch, integrity/structure, semantic
invariant, regenerated-evaluation mismatch, replay-cutoff mismatch, and install
success. Reasons are fixed enums; symbols, paths, checksums, and provider text
are not metric labels.

## 13. Simplicity and boundedness

- The only active mutable owner remains `ScannerStateEngine`; projection and
  install are two explicit owner stages over detached immutable state.
- Persist semantic sufficient state once. Do not persist evaluator output and
  also trust it, retain raw session history, or create a second restart model.
- Existing Component 2/3 cardinality bounds are the schema bounds. Their
  theoretical maxima prove finite decoding, not production memory/capacity;
  Component 8 must later accept deployed size/latency.
- Installation may use a detached builder and scratch candidate graph so
  validation precedes the sole atomic swap/apply. Private layouts and whether
  empty records share immutable zero values remain implementation discretion.
- V2's detached-builder/all-or-none-install technique is adapted. Its derived-
  only state, opaque prefix, fabricated aggregate mark, old owner/token/fence,
  and 16-minute-lag checkpoint model are rejected.
- No migration between schema versions is required in version 1. An unsupported
  version falls back. A migration or replacement format is a lower-level C7
  contract revision allowed by the V1 correction loop when evidence requires
  it; it cannot weaken semantic completeness or invalid-artifact rejection.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| Committed `T0` stalls while forward state advances | Accepted C3-R1/R2 regressions | Project exactly at `T0`; post-`T0` tail/fold/proof state cannot contaminate restore | `P-C7-STATE` |
| Provisional proof is revised; finalized latch survives | Accepted C3 qualification proof | Persist proof/dirty/finalization inputs; install regenerates the same provisional or final result | `P-C7-STATE` |
| Valid checksum but contradictory semantic state | Product invariant and V2 structural validation limitation | Engine rejects before any visible mutation | `P-C7-INSTALL` |
| New live correction targets restored tail | Component 2 correction contract; process-position invariant | New-epoch live evidence can revise within `H`; dead-process positions have no authority | `P-C7-INSTALL`, `P-C7-LIVE` |
| Empty/sparse `[T0,R)` catch-up | Accepted C6 startup/no-print proofs | Complete empty remains explicit no-print; failure remains unknown; ordinary evaluator resumes | `P-C7-LIVE` |
| Replay artifact begins before or after expected cutoff | Accepted C4 identity/order contract | Exact `T0` continuation succeeds; repeated/gapped prefix rejects or uses complete-from-`S` fallback | `P-C7-REPLAY` |
