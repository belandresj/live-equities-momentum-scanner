# LBR-C1 implementation assignment — bounded Tape and Spread state

**Status:** Draft for independent cross-check; not an active ledger entry or
implementation authorization.

**Activation gate:** `LBR-P1` and Capabilities A/B must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-C1` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md) and
[evaluation](../evaluation-and-publication.md) contracts, their recorded
handoffs, and the complete [T/Q contract](../tq-state.md). This ticket
implements only `LBR-C1` and `P-LBR-C1-TQ-STATE`.

Controlling requirements are `PG-FEATURE-05`, `PG-FEATURE-06`,
`PG-AVAIL-03`, `DTE-CLOCK-02`, `DTE-WINDOW-03`,
`DTE-TRADE-01`, `DTE-TRADE-02`, `DTE-QUOTE-01`, `DTE-QUOTE-02`,
`DTE-TQ-01`, and `DTE-TQ-02`, including the owner-approved strict 30-second
late-event and receipt-time duplicate horizons.

## 2. Outcome, owner, scope, and interfaces

The sole engine owner retains bounded selected-symbol trade/quote coverage,
Tape 5s contribution and duplicate evidence, and O(1) Spread quote state.
Trade and quote channels confirm independently from post-write data. T/Q never
affects aggregate qualification, ranking, watermark, or readiness.

Inputs are the accepted B desired-symbol revision and typed normalized T/Q,
drop, write-boundary, epoch, and engine-time facts defined by the focused
contract. The current ingress adapter may supply them until Capability D.
Output is an immutable selected T/Q projection plus immediate trust-closure
notification for B2 publication. Membership batching and global pressure
transitions are deferred to `LBR-C2`.

## 3. Allowed implementation boundary

Allowed production paths are T/Q feature/coverage/retention state in
`internal/engine`, condition fixtures, and the narrow immutable publication
projection seam. `internal/snapshotapi` may be touched only to preserve the
accepted live v2 mapping. Do not edit provider decoder/queue/retry machinery,
aggregate evaluator, UI, launcher, replay, or checkpoint paths.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only current
[`top-20 T/Q`](../../specifications/top-20-tq-coverage-and-features.md),
[`data-confirmed subscription`](../../tq-data-confirmed-subscription-correction.md),
trade-condition, Tape/Spread, and API fixtures. Reject old 16-minute T/Q
fingerprints, generic-status coverage, one-second Tape output, historical
ledgers, and private retention shapes.

## 5. Required implementation behavior and bounds

- Maintain independent per-symbol trade/quote states
  `not_requested/requested_unconfirmed/confirmed/closed` for the current epoch
  and local generation.
- Confirm a channel only with structurally valid matching data whose complete
  frame sequence is strictly greater than successful-write boundary `B`;
  status messages and silence prove nothing.
- Ignore/count event time strictly older than `T-30s`; accept equality. Before
  `T`, confirmation may occur but a field cannot become current.
- Retain exact trade identity/fingerprint evidence through receipt+30s inclusive
  and evict only after that boundary. Exact repeats contribute once; an unequal
  repeat makes Tape invalid for the generation.
- Retain only five-second Tape contributions/lifecycle evidence; covered quiet
  time becomes numeric zero after warm-up. Remove one-second Tape-burst state.
- Keep latest observed and latest valid quote only; preserve locked zero,
  one-sided/crossed quality, quiet aging/staleness, and gap clearing.
- Enforce contribution limits 10,000/symbol and 100,000 global; duplicate
  limits 25,000/symbol and 400,000 global; combined T/Q charge 64 MiB; at most
  two quote records/desired symbol and bounded cleanup liabilities.
- A symbol bound closes only its fields and requests cleanup; a global bound
  enters aggregate-only intent. Neither changes aggregate state/readiness.

## 6. Primary proof and acceptance distinction

`P-LBR-C1-TQ-STATE` covers strict `B`, independent confirmation, Tape warm/
current/covered-zero, locked/one-sided/crossed/stale Spread, exact/unequal
duplicates, lifecycle disclosure, out-of-order events at `T-30s` and one tick
older, duplicate expiry at receipt+30s and one tick later, gaps/removal, and
every count/byte bound.

Observe exact fields/reasons, coverage/generation, membership, retained charges,
counters, trust publication, and byte-for-byte aggregate projection
equivalence. Dangerous counterexamples are status-created coverage, a duplicate
counted twice, or a bound leaving current output. It does not prove socket
writes, membership churn, or queue pressure.

## 7. Verification and timeout policy

Run the primary proof, direct condition/Tape/Spread tests, affected engine/API
short tests, affected race under five minutes, focused vet,
`git diff --check`, and ordinary `go test -count=1 -short -timeout 2m ./...` at
the gate. Bound tests use scaled deterministic limits plus exact production
constructor assertions; no provider access or broad capacity tier.

## 8. Removal and handoff

Acceptance makes old one-second Tape state/output assumptions, 16-minute T/Q
fingerprints, broad string-heavy trade retention, and separate mutable Tape/
Spread representations removable. Handoff records exact retained charges,
channel/generation and immutable-view seams, old code now removable, proof
limitation, and whether `LBR-C2` remains valid.

## 9. Implementer discretion and prohibited changes

Compact exact identity encoding with collision resolution, rings/deques,
eviction indexes, fixed arrays, and private enums are delegated. Do not extend
either 30-second boundary, invent lifecycle/identity evidence, change Tape/
Spread formulas or quality rules, create provider acknowledgements/retries,
make T/Q rank/readiness input, or add T/Q workers, replay, or checkpoint state.

## 10. Containment, review, and correction

Private generations/boundaries prevent caller-created coverage. Runtime
validation fences stale epoch/generation/position, malformed time/value/
identity, unexpected symbol, late evidence, and bound failures. Any ambiguous
identity that can look current triggers narrow review. Failed proof/review
reopens the smallest ticket/spec through the program loop; only the orchestrator
records acceptance and commits after quiescence.
