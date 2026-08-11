# Cached hydration fence-finalization correction

Status: deterministic correction accepted 2026-08-10; separately authorized
provider confirmation remains pending. This is a narrow live-startup
correction, not a new component contract or architecture.

## Conclusion

Use the sealed 2026-08-07 full-session aggregate artifact as the repeatable,
no-network scale input for the production hydration/fence boundary. Ordinary
replay is not the proof because it bypasses the C6 hydration ledger and live
ingress fence.

The local harness must run the real hydration plan, historical chunk and
terminal admissions, concurrent bounded live-aggregate queue, actual ingress
fence, feature/qualification evaluation, ranking publication, and readiness
projection. Profile the failing fence before selecting another correction.

Only after this cached proof passes may one current-date provider launch be
used to confirm the external REST/WebSocket composition.

## Fixed evidence

Use these files in place; do not copy the 2.4 GiB artifact into this worktree.

- Artifact:
  `/Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/aggregate-replay/aggregate-replay-fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a.jsonl`
- Artifact identity:
  `sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`
- Declared complete-session population: 5,691 coverage entries, including 172
  successful empty symbols.
- Declared aggregate records: 7,671,171 over
  `[2026-08-07T08:00:00Z, 2026-08-08T00:00:00Z)`.
- Exact-date reference directory:
  `/Users/joshuabelandres/Dev/live-equities-momentum-scanner/var/reference`
- Its prior-close accounting contains 5,502 valid and 189 missing symbols.

The harness must fully validate the artifact seal and exact binding before
timing. The hydration population is the engine plan's 5,502 valid-prior-close
symbols, not all 5,691 artifact coverage entries. During preflight, calculate
and record the exact artifact row and successful-empty counts belonging to the
plan interval and registered tokens. Those derived counts become the terminal
and row-accounting oracle.

## Behavior that must remain unchanged

This correction remains controlled by `LIFE-HYDRATE-05`,
`LIFE-HYDRATE-06`, `DTE-COMMIT-02` through `DTE-COMMIT-04`,
`C6-FENCE-01`, `C6-INTEGRATION-01`, and `P-C8-RUNTIME`.

- One engine remains the canonical state, lifecycle, fence, and readiness
  owner.
- Every hydration request has exactly one terminal result.
- Successful empty remains explicit complete no-print evidence.
- Every classified live item through the fence is consumed or explicitly
  rejected before readiness.
- Aggregate ranking never depends on T/Q availability or recovery.
- Readiness is published only after the exact fence and coherent projection.
- Market formulas, correction horizon, ranking order, population accounting,
  and the maximum 20 displayed rows do not change.

Do not solve the failure by increasing the 512-frame live queue, enlarging a
timeout, suppressing metrics, weakening accounting, skipping features,
dropping symbols, relabeling 2026-08-07 data as another date, or publishing
readiness before finalization completes.

## Required cached harness

Add one explicitly selected, non-short acceptance harness. It may add small
test-only adapters around existing production seams, but it must not add a
second engine or replay-owned readiness path.

1. Load the exact cached binding and validate the artifact using the existing
   replay-artifact trust boundary.
2. Select a late-session `R` within the artifact, initially 17:15 ET, so the
   row volume is comparable to the failed 2026-08-10 starts while the session
   is still active.
3. Establish the ordinary live epoch and acknowledgement, then obtain the real
   fresh-bootstrap hydration plan derived by the engine.
4. Stream the artifact once in canonical time order. Buffer at most one
   production-sized partial chunk per registered symbol, admit historical rows
   under their actual request tokens, and admit the corresponding terminal
   result after the artifact interval is exhausted. Ignore symbols outside the
   engine plan without assigning them hydration outcomes.
5. Concurrently run a deterministic paced aggregate stream through the same
   bounded adapter-to-engine queue used by live operations. Include overlapping
   symbols and keep producing before, during, and after the final terminal and
   fence. Use the observed late-session frame rate and a second run at twice
   that rate; do not fabricate T/Q as a ranking prerequisite.
6. Capture and consume the real engine-issued ingress-fence command/fact. Do
   not call a test-only readiness shortcut or forge coverage.
7. Record phase timings and a CPU profile for at least: terminal completion,
   coverage installation, compaction/maintenance, qualification/features,
   ranking projection, fence disposition, and first ready publication.
8. If the run fails, correct the lowest measured dominant cost and rerun this
   cached harness. Do not repeat a provider download to diagnose local engine
   work.

The artifact reader may interleave chunks for different symbols. It must keep
per-token chunk order exact and bound resident partial chunks. No temporary
artifact rewrite is required.

## Cached acceptance

One run passes only when all of the following are true:

- artifact identity, binding identity, interval, plan population, derived row
  count, and empty/value terminal counts validate before timing;
- every registered request reaches exactly one correct terminal outcome and
  hydration row/work accounting reconciles exactly;
- the paced live producer remains active across finalization, with no capacity,
  receipt, oversize, gate, close, or accounting rejection;
- live delivery resumes before queue capacity is threatened, all admitted
  frames reconcile, and the queue drains after the fence;
- the ingress fence is applied exactly once and covers every causal predecessor;
- lifecycle becomes `live`, backend readiness becomes true, suppression remains
  empty, and ranking becomes current with between zero and 20 rows according to
  the actual market data;
- publication, population, aggregate, hydration, transition, adapter, and queue
  accounting remain valid;
- heap behavior and post-fence compaction are recorded; no arbitrary heap cap is
  a pass/fail gate;
- T/Q state and recovery are reported separately and do not delay aggregate
  ranking readiness; and
- the complete command finishes within the repository's 15-minute local-test
  limit.

Repeat the passing cached run once from a fresh engine. Semantic outcomes and
accounting must match; wall-clock timing may vary. Run the focused engine and
operations race proofs, `go test -short -timeout 2m ./...`, vet, UI tests, and
diff checks after the final correction.

## Final live confirmation

Do not make provider calls while iterating. After cached acceptance and local
verification pass, and only with explicit authorization for that exact date,
run the ordinary scanner once with two REST hydration workers and the real
aggregate WebSocket.

The live confirmation must reach 100% hydration, reconcile the actual ingress
fence without queue rejection, publish current ranking/readiness, expose zero
to 20 rows through the API/UI, and remain stable long enough to observe
post-fence memory and T/Q behavior. If it fails at a new external boundary,
retain the evidence and return to the cached or fake-provider proof that
distinguishes that boundary; do not repeatedly redownload unchanged history.

If it succeeds, leave the scanner and dashboard running for owner review and
report their loopback addresses, lifecycle, ranking mode, row count, queue
high-water, accounting status, heap behavior, and T/Q state.

## Completion record

Update this document and the C8 ledger with:

- preflight identities and exact derived population/row/terminal counts;
- failing and passing phase timings plus the dominant CPU profile;
- the implementation correction and why semantics are unchanged;
- cached run accounting, queue high-water, readiness/ranking, heap, and T/Q;
- verification commands/results; and
- the final live result or an honest new external limitation.

### Accepted deterministic result, 2026-08-10

The sealed preflight validated binding
`session-binding-v1:b68b50821b19ec69073cf039bc61a9d193825578b65708e49aec892aa97b7f7d`,
artifact
`sha256:fce95a904bb37c3d63bfe0a2988ac78aa0e0a2ee4c8d171bdca27fcd00f8808a`,
the exact `[2026-08-07T08:00:00Z,2026-08-07T21:15:00Z)` interval,
5,502 planned valid-prior symbols, 7,581,690 rows, 5,439 value terminals,
and 63 successful-empty terminals before timing.

The preserved evidence-of-record full 1x trial passed with 30,483 frames
read/admitted/dispositioned/fenced, zero rejection, queue high-water 247,
526.912 ms fence, 364.965 ms tail drain, `live` lifecycle,
`qualified_current` ranking with 20 rows, and ready/accounting true. Its exact
output remains at `var/cached-hydration-fence/final-1x/run.log`.

Production engine changes made after that artifact were not assumed safe. A
fresh final-source 1x confirmation revalidated the same artifact/binding/row
and terminal oracles, then passed with 28,377 frames
read/admitted/dispositioned/fenced, zero rejection, queue high-water 293,
490.875 ms fence, 650.808 ms tail drain, 52.542 microseconds to first ready,
`live`/`qualified_current`, 20 ranked rows, heap 2,576,013,024 bytes at the
fence and 1,143,073,416 bytes after GC, `aggregate_only` T/Q pressure with 20
rows, and ready/accounting true. The separate output is preserved at
`var/cached-hydration-fence/final-source-1x/run.log`; metrics from the two paced
trials are not combined.

The focused final-source full-retention 2x burst completed a 7,745,536-row
preload and all 12,000 measured chunks/3,072,000 rows. It read and
dispositioned 23,448 frames, rejected none, reached queue high-water 38/512,
drained in 2.194 ms, and reconciled accounting. The proof was strengthened to
require the complete measured chunk/row target, preventing a truncated final
burst from passing. Its output is preserved at
`var/cached-hydration-fence/final-source-bounded-2x/run.log`.

The preserved sustained full 2x trial remains an honest unsupported boundary:
56,757 frames read, 56,756 admitted, 56,244 dispositioned, 512 fenced, and one
capacity rejection after 7,078,180 hydration rows. It was not rerun. This
failure is not equivalent to the passing bounded burst and is not a blocker to
the accepted private 1x workflow.

Focused final-source race proofs, `go test -count=1 -short -timeout 2m ./...`,
`go vet ./...`, and all 22 UI model/visual tests passed. No credential, Keychain
item, or provider was accessed. The separately authorized live confirmation is
therefore still pending and is not part of this deterministic acceptance.
