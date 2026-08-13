# Live coverage and partial-ranking correction

**Status:** Accepted correction, 2026-08-13. Slice 1 is reaccepted after live
evidence exposed and correction removed an evaluator-scheduling gap. Slice 2
now contains symbol-local uncertainty in `degraded_current`; focused, ordinary,
race, exact-wire UI, and production-composition proofs pass and final focused
read-only review is clean. No live-provider request was made for Slice 2.

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
4. A REST/live discrepancy may coexist with the accepted live canonical value
   only as a bounded diagnostic. Valid live precedence establishes that
   identity's correctness, so the discrepancy does not install historical-
   conflict state or prevent exact dependent fields or qualification. An
   unresolved historical/historical conflict remains fail-closed.
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
canonical merge rule gives accepted live evidence authority over overlapping
REST evidence. That resolved discrepancy remains a bounded diagnostic and does
not change coverage, fields, qualification, or current-mark trust. Only
unresolved evidence that could hide a newer mark has those consequences.

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

Primary proof is a matched allocation across the existing sole-owner
boundaries, not one monolithic fixture:

- the production `RunLive` composition proves normalized valid/quiet facts,
  equivalent and conflicting REST/live overlap, hydration, and ordinary live
  fences reach the sole Engine path;
- Engine admission proofs cover the later live correction/mark, isolated
  unknown historical identity, unusable qualification contributor, zero
  trusted marks, replay isolation, and the global-gap counterexamples;
- a real Runtime/Engine composition installs a trusted mark plus an
  attributable local defect, then proves the sealed snapshot, readiness 200,
  exact counters/rows, no T/Q, and subsequent transport-loss readiness 503;
  and
- the exact JSON bytes emitted from that sealed capture pass the production UI
  validator/model, while the renderer proof asserts the partial label,
  non-qualified caption, retained styling, and T/Q rejection.

Malformed aggregate websocket frames are intentionally normalized into
`DeliveryNormalizationDrop` and do not become Engine canonical/local-defect
facts. The `RunLive` proof must not fabricate an adapter path that production
contains before Engine admission; Slice 2 begins at the Engine evidence
boundary for attributable structural-invalid input. Together the matched
proofs require symbol-local cases to retain a current `qualified_current` or
`degraded_current` publication with mode-consistent rows; only systemic
ambiguity or zero trusted marks may make the scanner unavailable.

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

The 2026-08-13 live run reopened the production-advancement claim. An accepted
ordinary live-coverage fence advanced `supportedThrough`, but
`inputLiveCoverageFence` was not an evaluator cause. Runtime waits for that
fence and admits the timer afterward; when fence capture and timer admission
straddled a whole-second boundary, the timer targeted one unsupported second
later and committed `T` remained frozen. The prior runtime proof advanced its
injected clock in exact one-second steps and therefore did not distinguish this
production chronology.

The correction carries the accepted fence's exact target into the existing sole
target/evaluator path and stages and applies that target on the fence
transition. The dangerous counterexample uses fence target `T+1` with admission
clock `T+2`: the fence must commit exactly `T+1`, and the unsupported later
timer must not advance it. Supported same-target timers remain evaluator causes
because processing time can cross the strict correction horizon while market
`T` is unchanged, especially at the session-end cap; a separate regression
requires provisional qualification to finalize in that case. Replay,
hydration-ingress fences, ranking modes, checkpoint mechanics, and market
semantics are unchanged.

| Item | Status | Evidence / correction record |
| --- | --- | --- |
| Parent decision and two-slice plan | accepted, revised | Owner directed robustness over perfect all-symbol history on 2026-08-11. The outcome and two-slice sequence remain; the evidence/time model and both slice boundaries are revised after the blocking Slice 1 review. |
| Slice 1 invariant-preserving ordinary live coverage | reaccepted after correction 2026-08-13 | The 2026-08-13 live run invalidated the claim that every accepted ordinary fence can advance committed `T`: fence capture and timer admission crossing a second left the timer unsupported. The exact chronology regression in `TestLiveAggregateEvaluationCoalescing` now proves the fence commits its captured supported target and the following unsupported timer cannot advance it. A second regression proves a supported same-target timer still performs strict correction-horizon qualification finalization at session-end-capped `T`. Focused Engine and Runtime Slice 1 proofs pass; `go test -short -timeout 2m ./...`, focused race proofs, `go vet ./internal/engine ./internal/operations`, and `git diff --check` pass. Preserve the prior identity/coverage proofs; they were not invalidated. |
| Slice 1 read-only review | accepted 2026-08-11; focused correction re-review clean 2026-08-13 | The required `gpt-5.6-sol` medium correction review found that a proposed generic same-target skip could omit strict correction-horizon maintenance and same-`T` hydration revisions. The skip and obsolete proof claim were removed, and the strict-horizon regression was added. Focused re-review found the implementation issue resolved and no remaining P1/P2 in the live-fence evaluator correction. Review was read-only and made no provider request. |
| Slice 2 local containment and `degraded_current` | accepted 2026-08-13 | Live lifecycle now publishes raw Day-% rows from trusted marks when remaining defects are symbol-local; potentially displaced marks stay excluded, qualification uncertainty stays explicit, and partial modes cannot create or expose T/Q. Later accepted live marks restore primary trust when post-mark coverage is exact without erasing historical/qualification limitations. `degraded_bootstrap` is startup-hydration-only; replay is unchanged. The real Runtime/Engine proof carries a local defect through a sealed snapshot to `/readyz` and snapshot 200, feeds those exact JSON bytes through the production UI model, then proves transport loss returns readiness 503. Focused Engine/Operations/API/UI tests, `go test -short -timeout 2m ./...`, affected `-race`, affected `go vet`, and `git diff --check` pass. |
| Slice 2 read-only review | accepted after correction 2026-08-13 | Required `gpt-5.6-sol` medium review found four P2s and then a P1 validator contradiction: later-mark trust depended on an unrelated conflict bit, bootstrap mode lacked a startup boundary, UI accepted partial T/Q, production/exact-wire proof was incomplete, and legal retained-bootstrap `degraded_current` could be rejected. All received distinguishing corrections. Final same-reviewer re-review found no remaining P1/P2 implementation or acceptance gap. |
| Combined production-composition acceptance | accepted 2026-08-13 | Matched proofs cover `RunLive` normalization/hydration/fences, Engine current-mark and qualification containment, Runtime readiness/API/transport failure, and exact sealed-wire UI consumption without crossing or duplicating ownership boundaries. This is a local runnable-code claim, not a market-hours/provider observation. |
| Final integrated read-only review | planned | Required after combined acceptance. |

If later evidence invalidates an accepted claim, mark only that item reopened,
preserve unaffected proof, revise this compact plan if necessary, and run the
narrowest distinguishing proof before continuing.
