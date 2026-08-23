# LBR-D1 implementation assignment — single-pass Massive decoding

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** `LBR-P1` and Capabilities A–C must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-D1` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), and [T/Q](../tq-state.md)
input contracts and handoffs, and the complete
[ingress contract](../live-ingress.md). This ticket implements only `LBR-D1`
and `P-LBR-D1-DECODE`.

Controlling requirements are the decoded-batch portion of `LBR-ARCH-03`,
`DTE-MODEL-01` through `DTE-MODEL-03`, `DTE-SESSION-03`, `DTE-SESSION-04`,
`DTE-CLOCK-03`, `DTE-EVENT-01`, `DTE-EVENT-02`, `DTE-CONTROL-01`, and
`DTE-REJECT-01`. Common failure containment under `LBR-ARCH-11` is an
integration-owned conformance obligation and cannot be weakened here.

## 2. Outcome, owner, scope, and interfaces

One attempt-local reader/decoder reads each complete provider frame, captures
one receipt time/frame sequence, streams the JSON array once, and produces one
bounded immutable `DecodedBatch` preserving array order and exact downstream
facts/drops/ambiguity. It performs no full-frame pre-scan plus per-element
unmarshal and retains no raw bytes after admitted transfer.

Inputs are documented provider text frames and current epoch/binding/pressure
policy. Outputs are the accepted A/B/C normalized aggregate, control, T/Q,
drop, and ambiguity facts. A temporary adapter may translate the immutable
batch into the existing handoff until `LBR-D2`; it cannot decode again or own
state.

## 3. Allowed implementation boundary

Allowed production paths are provider classification/normalization and new
immutable batch decoder files under `internal/massive`, plus the smallest
temporary batch-to-current-handoff bridge and focused tests/fixtures. Do not
replace the raw queue/engine FIFO or retry/runtime composition in D1. Do not
change engine market behavior, product formulas, API/UI, launcher, replay, or
checkpoint code.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only current
[`provider normalization`](../../specifications/massive-live-adapter/provider-classification-and-normalization.md),
[`transport/epoch`](../../specifications/massive-live-adapter/transport-commands-and-epochs.md),
data-confirmation correction, and their directly named provider fixtures.
Wire shapes/classifications are evidence; double parsing, raw event copies,
intermediate envelopes, status-count coverage, and delivery ledgers are
rejected. No credential or live request.

## 5. Required implementation behavior and bounds

- Read one complete text frame, assign epoch/frame/receipt once, and stream its
  array in exact index order through one decoder pass.
- Produce one immutable batch with sole-owned backing and causal position
  `(epoch, frame, array_index)` for each ordered fact.
- Accept unknown additive members; reject duplicate recognized members; drop
  clearly identified unsupported families with bounded accounting.
- Localize malformed `A` only with trustworthy symbol/window identity; localize
  malformed T/Q with trustworthy family/symbol; known T/Q family without symbol
  closes T/Q trust, not aggregates.
- Missing/malformed/duplicate discriminator, truncation, or any element that
  could conceal aggregate/control work records the earliest ambiguity; preserve
  earlier complete facts before the terminal and accept no hidden suffix.
- Preserve exact downstream values, structural/classification evidence,
  receipt/event times, provider position, and closed provider-status classes.
- Enforce source frame <=8 MiB, batch <=65,536 elements and <=32 MiB retained
  charge; raw bytes are released after transfer. Oversize/charge failure follows
  the contract's explicit ambiguity/capacity outcome.
- Pressure/frame-budget mode skips expensive T/Q normalization only after
  family classification and emits ordered shed facts while later aggregate/
  control elements remain processed.

## 6. Primary proof and acceptance distinction

`P-LBR-D1-DECODE` runs the approved fixture corpus through the streaming
decoder and frozen semantic oracle: mixed arrays, positions/receipt, every
recognized family/drop, unknown additive fields, duplicate recognized fields,
malformed localizable input, unsupported family, earliest ambiguity, oversize,
status, and pressure-shed suffix.

Observe normalized fact equality, ordering/dispositions/accounting, one parse,
no retained raw aliases, and allocations/throughput measured separately from
engine work. Dangerous counterexamples are a second parse or silent ambiguous
suffix containing aggregate/control work. This proof does not establish FIFO
saturation, socket timing, or provider behavior beyond fixtures.

## 7. Verification and timeout policy

Run the primary decoder corpus, direct normalization tests, affected massive
short/race tests under five minutes, focused vet, allocation benchmark with a
validated bounded fixture, `git diff --check`, and ordinary
`go test -count=1 -short -timeout 2m ./...`. Separate semantic correctness from
performance measurement. No provider/live tier.

## 8. Removal and handoff

Acceptance makes the double-pass logic in `internal/massive/live_normalization.go`,
per-element raw JSON copies, and intermediate normalization envelopes removable.
Handoff records the immutable batch schema/charge, exact temporary bridge,
fixture/accounting result, measured allocations and limitation, and whether
`LBR-D2` remains valid.

## 9. Implementer discretion and prohibited changes

Streaming-token helpers, private value layout, exact string interning with
bounds, closed enums, and file organization are delegated. Do not invent
provider fields/status semantics, add another reader/queue/owner, silently drop
aggregate/control, change T/Q/product behavior, or expand to a generic event
framework, public deployment, replay, or checkpoint work.

## 10. Containment, review, and correction

Immutable transfer and one causal position must be construction guarantees.
Runtime validation contains malformed/oversize/local input and escalates only
the exact ambiguity failure domain. A remaining parse-order or ambiguity false
success triggers narrow review. Findings reopen the smallest ticket/spec via
the delivery loop; only the orchestrator records acceptance and commits after
quiescence.
