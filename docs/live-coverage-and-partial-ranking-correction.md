# Live coverage and partial-ranking correction

**Status:** Owner-directed executable correction plan, revised 2026-08-11.
Slice 1 is accepted after invariant-level correction, complete focused proof,
and clean read-only re-review. Slice 2 remains planned and depends on the
accepted Slice 1 boundary.

## Decision and outcome

The scanner must be exact about what it knows without requiring perfect
provider data from every symbol before it can operate.

This correction preserves exact `qualified_current` output and adds a current,
explicitly partial fallback for symbol-local aggregate uncertainty after
bootstrap. A symbol-local conflict, missing historical identity, invalid
qualification contributor, or unusable qualification history must not make an
otherwise current scanner publish zero rows. Global binding, transport, fence,
clock, canonical-integrity, or zero-trusted-mark failures remain unavailable.

This document records the owner's correction to the prior strict
`PG-RANK-05` interpretation. It does not change `PG-RANK-01` through
`PG-RANK-04`: trusted current marks, the approved qualification formula, exact
Day-% ordering, symbol tie-breaking, and the 20-row limit remain unchanged.
Qualification is never fabricated. When exact qualification is unavailable,
the partial projection is raw known-rankable Day-% order and is visibly not the
qualified table.

## Authority, ownership, and scope

Controlling requirements are `PG-RANK-01` through `PG-RANK-05`, `PG-OBS-01`
through `PG-OBS-03`, `DTE-WINDOW-01`, `DTE-MERGE-04`, `DTE-RECOVERY-01`
through `DTE-RECOVERY-04`, `DTE-TIMER-01`, `DTE-COMMIT-02`,
`DTE-COMMIT-03`, and `LIFE-LIVE-01` through `LIFE-LIVE-03`.

The `ScannerStateEngine` remains the sole canonical aggregate, coverage,
qualification, ranking-mode, and publication owner. Provider adapters only
normalize facts. Runtime derives readiness from one sealed publication; the
API maps it; the UI presents it. No second evaluator, readiness owner,
watermark, service, database, event bus, or generalized quality framework is
introduced.

In scope:

- ordinary post-bootstrap live coverage through accepted FIFO fences;
- exact print/no-print evidence for quiet symbols;
- retained structural-invalid evidence that prevents false absence or stale-mark
  trust;
- containment of REST/live conflicts and other symbol-local history defects;
- a current partial ranking projection after bootstrap; and
- exact accounting, API/readiness mapping, UI status, and deterministic
  production-composition proof for those behaviors.

Out of scope:

- changing the qualification formula or Day-% ordering;
- silently filling unknown aggregates, price, volume, or ATS values;
- a second coverage representation or generalized rejected-identity history;
- provider requests, credentials, live-market validation, queue resizing,
  checkpoint format/terminal work, or unrelated operational corrections.

## Canonical evidence and time model

The engine owns one evidence model. Coverage is not inferred from a timestamp
alone and no downstream evaluator repairs contradictory identity facts.

For each bound symbol and one-second aggregate identity:

1. Accepted canonical presence and proven absence are mutually exclusive.
2. Proven absence may be installed only when an accepted wildcard-stream fence
   covers that identity and no accepted presence, retained structural-invalid
   evidence, or unresolved historical conflict prevents the absence claim.
3. A retained structural-invalid identity is observed but unusable evidence. It
   remains retained, prevents absence for that identity, and prevents trust in
   an older mark when it is newer than that mark and before the evaluation
   watermark.
4. A historical conflict may coexist with the accepted live canonical value.
   It does not erase that value, but it prevents exactness for every dependent
   historical field or qualification fact whose interval includes the conflict.
5. When structural-invalid evidence is retained for an identity previously
   marked absent, the engine revokes that absence immediately. An ordinary
   fence must not reinstall it while the invalid evidence remains applicable.

All coverage and evaluation intervals are half-open. For an advancement from
`T0` to `T1`, only identities with `T0 <= window_start < T1` participate in the
new interval. Evidence with `window_start == T1` remains retained but belongs
to the next state: it does not invalidate a publication at `T1`, and it becomes
applicable when a later committed watermark advances past `T1`.

The canonical presence/absence bitmaps remain the one coverage truth.
Structural-invalid and historical-conflict state are retained evidence that can
prevent an absence bit or make dependent coverage non-exact; they are not a
second coverage map. `aggregateEvaluator.coverage` remains a derived,
symbol-level consequence carrying no-print or a closed uncertainty origin. It
must never contradict the canonical identity evidence as of `T`.

At evaluation time the engine derives these separate symbol facts:

1. **Current mark:** trusted, absent/no-print, invalid, or unknown because an
   unresolved identity before `T` could hide a newer mark.
2. **Qualification history:** exact, locally uncertain, or unusable.
3. **Row eligibility:** exactly qualified, raw-partial rankable, or excluded.

A later accepted live mark can restore current-mark trust when no still-newer
unresolved identity precedes `T`. It does not automatically make an earlier
qualification interval exact. Current-mark trust and qualification-history
quality are evaluated separately before row eligibility is derived.

Source disagreement is not automatically whole-symbol uncertainty. The
canonical merge rule still gives accepted live evidence authority over
overlapping REST evidence. A resolved conflict remains a bounded
historical-quality diagnostic and affects current-mark trust only when the
unresolved evidence could hide a newer mark.

Ranking modes are:

| Mode | Required evidence and rows |
| --- | --- |
| `qualified_current` | Current binding, transport, watermark, and fence; no primary unknowns; qualification exact through `T`. Publish the exact qualified top 20. |
| `degraded_bootstrap` | Existing startup-only partial mode. Every unresolved cause is bootstrap-origin. Publish raw known-rankable top 20 and do not promote T/Q. |
| `degraded_current` | Binding, aggregate transport, watermark, and ingress fence are current; at least one trusted rankable mark exists; remaining uncertainty is symbol-local and cannot establish global canonical ambiguity. Publish raw known-rankable top 20, expose exact excluded/uncertain counts, label the table partial, and do not promote T/Q. |
| `unavailable` | No committed `T`, no trusted mark for a partial view, or unresolved binding, transport, fence, clock, or global canonical integrity. Publish no rows. |

`degraded_current` is backend-ready because its causal target and limitations
are current and explicit. It is not qualified and must not be presented as an
exact complete-population result. Primary population accounting remains exact;
qualification and historical quality remain separate overlapping dimensions.

## Slice 1 — invariant-preserving ordinary live coverage

**Outcome:** Advancing an accepted live-coverage fence from `T0` to `T1`
installs only evidence justified for `[T0,T1)`. For every bound symbol, each
identity in that interval has accepted canonical presence, justified proven
absence, or retained evidence that keeps the interval explicitly non-exact. A
quiet symbol does not become qualification-unresolved merely because the global
`supportedThrough` timestamp advanced.

Required behavior:

- preserve the FIFO rule that the live-coverage fence follows every admitted
  predecessor through its target position and retain all existing causal-fence
  validation;
- make structural-invalid retention and exact-coverage installation preserve
  the canonical invariant by construction: retaining invalid evidence revokes
  conflicting absence, and coverage installation never writes absence over
  applicable invalid evidence;
- install absence for `[T0,T1)` without overwriting accepted aggregate presence
  or historical conflicts and without fabricating then repairing a false fact;
- apply the same half-open predicate wherever invalid evidence affects coverage,
  primary classification, evaluator validation, or stale-mark trust;
- keep an invalid identity at `window_start == T1` retained but outside the
  publication at `T1`; a subsequent advancement past `T1` must make it
  applicable;
- derive `coverageNoPrintThroughT` only when canonical coverage is exact through
  `T` and no applicable invalid evidence or conflict contradicts it;
- preserve a pre-existing closed uncertainty origin when its unresolved
  interval remains relevant, and assign `post_bootstrap_gap` to a newly
  discovered ordinary-live hole; never leave an ordinary unresolved result with
  origin `none`;
- preserve an older accepted mark canonically while refusing to trust it across
  a newer unresolved invalid identity before `T`;
- retain existing timer, qualification-window, correction-horizon, watermark,
  `generated_at`, publication-identity, and suppression meanings;
- keep `ScannerStateEngine` as the only mutable owner and introduce no second
  coverage representation; and
- make no `degraded_current`, API, UI, T/Q, checkpoint, or Slice 2 behavior
  change in this slice.

Primary proofs:

1. **Engine invariant and half-open boundary proof.** Use production aggregate
   admission and an accepted ordinary fence to cover, for both no-older-mark and
   older-accepted-mark states:
   - structural-invalid evidence at `T0` or strictly inside `[T0,T1)` remains
     retained, is not absent, makes the interval non-exact, receives
     `post_bootstrap_gap`, and prevents stale-mark trust;
   - structural-invalid evidence at `T1` remains retained but does not affect
     the publication at `T1`; the timer publishes coherently without evaluator
     or accounting-integrity suppression;
   - the next accepted fence past `T1` makes that retained boundary identity
     applicable and produces the correct closed uncertainty consequence; and
   - late structural-invalid evidence for an identity already marked absent
     immediately revokes that absence and cannot coexist with false
     `no_print_through_T`.

   Assert exact population, qualification, uncertainty-origin, aggregate,
   admission, and transition accounting at each publication. Use a nonzero
   evaluation delay for the right-open endpoint case so a completed identity at
   `T1` can causally precede a fence whose target is `T1`.

2. **Runtime production-composition proof.** Use bounded deterministic
   `Runtime.RunLive`, the production hydration worker, and a fake live adapter
   with at least three symbols: one printing, one quiet, and one with an
   overlapping REST/live identity. Complete hydration and its ingress fence,
   then cross at least two ordinary live-coverage fences and timers. Require
   committed `T` to advance, quiet seconds to be exact no-print, no new
   origin-`none` qualification uncertainty, preserved accepted live authority
   and conflict evidence, exact accounting, and no regression from a current
   projection solely because the symbols were quiet.

The Engine proof owns the identity/time invariant and dangerous invalid-mark
counterexamples. The Runtime proof owns production composition, FIFO delivery,
and quiet/presence/conflict preservation; it need not duplicate every Engine
counterexample.

Slice 1 acceptance requires both focused proofs, ordinary
`go test -short -timeout 2m ./...`, `git diff --check`, and a narrow read-only
review of coverage mutation, half-open time advancement, evaluator validation,
qualification consequences, and production composition. Record the proof
result and limitation in the ledger before Slice 2 begins.

## Slice 2 — local uncertainty containment and current partial output

**Dependency:** Accepted Slice 1 canonical evidence, half-open coverage, and
closed-origin behavior. Slice 2 consumes those facts; it does not repair,
reinterpret, or duplicate coverage.

**Outcome:** Symbol-local historical or qualification uncertainty no longer
forces an otherwise current scanner to `unavailable`. Exact qualification stays
exact; the new `degraded_current` projection provides a usable, visibly partial
raw Day-% table when only local uncertainty remains.

Required behavior:

- derive current-mark trust and qualification-history quality separately from
  the Slice 1 evidence model as of the same committed `T`;
- exclude a symbol when an unresolved identity before `T` could hide a newer
  current mark; evidence at `window_start == T` belongs to the next state;
- permit a later accepted live mark to restore primary current-mark trust only
  when no still-newer unresolved identity precedes `T`, while retaining any
  dependent historical or qualification limitation;
- treat accepted live-over-REST reconciliation as one canonical value plus a
  bounded conflict diagnostic, not a permanent whole-symbol population
  override;
- admit a symbol to raw-partial ranking only when its current mark and prior
  close are trusted; never infer qualification from uncertain history;
- select `degraded_current`, not `unavailable`, when system-level currentness is
  proved, at least one trusted rankable mark exists, and every remaining defect
  is symbol-local;
- keep `unavailable` for systemic binding, transport, fence, clock, canonical
  integrity, or zero-trusted-mark failures;
- preserve mutually exclusive primary population accounting and report
  qualification and historical-quality dimensions separately;
- map `degraded_current` to backend-ready HTTP 200, expose its exact population
  and quality counters, show an explicit partial-data UI state, and prohibit
  new T/Q membership exactly as for `degraded_bootstrap`; and
- preserve the exact qualified path and all existing ranking formulas.

Primary proof:

- Extend the production-composition scenario with equivalent overlap,
  conflicting overlap, a later live correction/mark, an isolated unknown
  historical identity, an unusable qualification contributor, and a genuine
  global stream gap. Prove separately that a later trusted mark can restore
  current-mark trust without erasing an earlier qualification limitation. The
  symbol-local cases must retain a current `qualified_current` or
  `degraded_current` publication with mode-consistent rows; only systemic
  ambiguity or zero trusted marks may make the scanner unavailable. Prove the
  snapshot/readiness mapping and production UI model consume the same sealed
  publication.

Slice 2 acceptance requires focused Engine/Runtime/API/UI tests, ordinary
verification, `git diff --check`, and a narrow read-only review of current-mark
trust, qualification-history containment, population accounting, ranking-mode
selection, readiness, and API compatibility.

## Combined acceptance and runnable claim

After both slices, one deterministic command must exercise the production
reference-binding path with validated local input, real Runtime and Engine,
REST hydration, concurrent fake live ingestion, overlap/conflict
reconciliation, the exact startup fence, at least two ordinary post-fence
publications, current qualified or `degraded_current` ranking, zero through 20
mode-consistent rows, snapshot HTTP 200, readiness HTTP 200, dashboard
consumption of the same publication, exact accounting, no mapper diagnostic,
and graceful scanner/API/dashboard shutdown.

The combined scenario proves cross-component wiring and mode-consistent
consumption. It does not replace the Slice 1 Engine boundary proof or rerun
every identity-level counterexample.

Deterministic acceptance does not prove provider behavior, trading edge, or
live-market capacity. Credentialed live confirmation remains separately
authorized under `docs/market-hours-validation.md`.

## Correction record and delivery ledger

The first Slice 1 implementation established ordinary quiet-symbol coverage
but failed blocking review. Its fence repair handled retained invalid evidence
strictly inside `[T0,T1)` but left evaluator validation insensitive to the
right-open boundary: invalid evidence at `window_start == T1` could coexist with
valid no-print coverage through `T1`, be misclassified as contradictory support,
and trigger accounting-integrity suppression. The invariant-level correction
made retention revoke contradictory absence immediately, excluded retained
invalid evidence from absence installation, and made evaluator validation use
the same as-of-`T` predicate.

The required review then found a second ordering counterexample: a retained
invalid identity at `T1` could prevent a later-arriving applicable invalid
identity at `T0` from replacing the bounded newest-evidence entry, and the old
implementation also skipped revoking `T0` absence. The correction now applies
identity-local absence revocation and closes applicable coverage for every
attributable structural-invalid input independently of whether that input
becomes the one bounded newest retained identity. This preserves the prohibition
on generalized rejected-identity history while preventing false no-print.

| Item | Status | Evidence / correction record |
| --- | --- | --- |
| Parent decision and two-slice plan | accepted, revised | Owner directed robustness over perfect all-symbol history on 2026-08-11. The outcome and two-slice sequence remain; the evidence/time model and both slice boundaries are revised after the blocking Slice 1 review. |
| Slice 1 invariant-preserving ordinary live coverage | accepted 2026-08-11 | `go test -short -timeout 30s ./internal/engine -run '^TestSlice1' -count=1` passes. The proof covers no older mark and older accepted mark; invalid identity at `T0`, strictly inside `[T0,T1)`, and exactly at `T1` with nonzero evaluation delay; coherent unsuppressed publication at `T1`; next-fence applicability; endpoint-first then late-interior invalid ordering; immediate absence revocation; exact population, qualification, uncertainty-origin, aggregate, admission, and transition accounting. `go test -short -timeout 30s ./internal/operations -run '^TestSlice1RuntimeRunLiveInstallsOrdinaryLiveCoverage$' -count=1` passes production `Runtime.RunLive` hydration, FIFO fences, printing/quiet symbols, and REST/live overlap/conflict preservation. `go test -short -timeout 2m ./...` and `git diff --check` pass. The focused evidence is deterministic and does not establish live-provider timing, entitlements, current population, or Slice 2 partial-ranking behavior. |
| Slice 1 read-only review | accepted 2026-08-11 | Required `gpt-5.6-sol` medium review first found the endpoint-first/late-interior ordering counterexample above. After correction and the new regression, focused re-review found the P1 resolved and no remaining P1/P2 in coverage mutation, half-open advancement, evaluator validation, stale-mark trust, accounting, or Runtime production composition. |
| Slice 2 local containment and `degraded_current` | planned | Depends on accepted Slice 1 evidence/time behavior and consumes it without coverage repair. |
| Slice 2 read-only review | planned | Required before combined acceptance. |
| Combined production-composition acceptance | planned | No runnable claim until both slices pass. |
| Final integrated read-only review | planned | Required after combined acceptance. |

If later evidence invalidates an accepted claim, mark only that item reopened,
preserve unaffected proof, revise this compact plan if necessary, and run the
narrowest distinguishing proof before continuing.
