# Massive live adapter — provider classification and normalization

**Parent contract:** [Massive live adapter](../massive-live-adapter.md)
**Normative responsibility:** A/T/Q/control frame classification,
provider-field mapping, structural rejection, receipt evidence, and live causal
positions
**Controlling requirements:** `C5-CLASS-01`, `C5-AGG-01`, `C5-TRADE-01`,
`C5-QUOTE-01`, `C5-STATUS-01`; parent-routed `ARCH-FLOW-01`–`04`,
`DTE-MODEL-01`–`03`, `DTE-SESSION-02`–`04`, `DTE-CLOCK-02`–`04`,
`DTE-WINDOW-01`–`03`, `DTE-EVENT-01`, `DTE-EVENT-02`, `DTE-AGG-01`–`03`,
`DTE-TRADE-01`, `DTE-TRADE-02`, `DTE-QUOTE-01`, `DTE-QUOTE-02`,
`DTE-CONTROL-01`, and `DTE-REJECT-01`
**Allocated slices:** `C5-S1`; see the parent delivery ledger
**Document dependencies:** Parent; Component 1 binding; Component 2 S1/S2/S4
input and aggregate boundaries; Component 4 provider-independent aggregate
value seam
**Approval state:** Inherits the parent completed-contract approval; not an
independent component authority
**Delivery state:** See the authoritative parent delivery-state ledger; do not
copy mutable status here

## 8. Version 2 reconnaissance and reuse assessment

Reconnaissance used v2 commit
`5f92a151dd850002578a33a81ad90dea096c63b6`. Only the symbols and focused
tests below were inspected. V2 behavior is evidence, never authority.

| Exact source/test/fixture and SHA-256 | Finding or validated behavior | Decision | Required adaptation or coupling to remove | Required proof |
| --- | --- | --- | --- | --- |
| `internal/massive/decode.go` — `Decoder.Process`, `decodeAggregate`, `decodeProductTrade`, `decodeProductQuote`, `decodeTradeConditions`, `decodeQuoteConditions`, `decodeQuoteIndicators`, `exactRawInt64`, `requiredFloat`, `requiredInt64`, `DecodeStatusCount`; `34cdd44e5a9c55d0632b85d24c05bc4b0bbdd08bdb14d9ad87c05404f20e42c7` | Provider discriminators are `A`, `T`, `Q`, and `status`; provider integer times are milliseconds; A uses `s/e/o/h/l/c/(dv or v)/vw/z`; trades demonstrate `x/i/p/s/ds/c/pt/t/q/z/trfi/trft/e`; quotes demonstrate `t/bx/ax/bp/ap/bs/as/c/i/q/z`. Arrays retain provider index. Timestamps tolerate at most 250 ms of positive provider skew without changing event time. | `adapt` | Replace decoder-owned `scanner.Engine`, `HealthStore`, metrics, TAQ state, direct symbol invalidation, direct recovery observation, wall-clock timing, JSON map duplicate loss, and decoder-local delivery sequence with pure complete-value-or-rejection normalization and engine-owned binding/position facts. Unknown additive fields stay allowed; duplicate recognized fields reject. | `P-C5-CLASS`, `P-C5-AGG`, `P-C5-TRADE`, `P-C5-QUOTE`, `P-C5-STATUS` |
| `internal/massive/decode_test.go` — `TestDecoderMassiveSchemaBatchOrderingAndFallback`, `TestDecoderFeedWideInvalidElementPreservesNeighbors`, `TestDecoderIdentityAndEligibleDomainFailureScopes`, `TestDecoderTopLevelAndStatusStrictness`, `TestDecoderNonUTF8AndIneligibleFiltering`, `TestDecoderBatchedAggregateTradeQuoteAndTAQValidation`, `TestDecoderTAQDecimalTradeSizeAndClosedRejectionReasons`; `7ac84310b615ddff69c53afd02a74da50c2d9c1edfa697a376a88dd0eb7dcf78` | Focused payloads establish batch order, `dv` decimal-volume preference with `v` fallback, live `z`, fractional `ds`, strict top-level/status phases, UTF-8 rejection, exact-symbol filtering, and local eligible-symbol numeric rejection. V2 applies valid items after an unclassifiable earlier element and only then fails the epoch. | `behavior evidence` | Reuse payload shapes, not scanner assertions. Preserve valid items causally before ambiguity, but stop/fence later array elements once an unclassifiable item is reached; do not apply causally later facts before the ambiguity transition. | Same five S1 proofs |
| `internal/massive/taq.go` — only `CausalPosition` uses visible through `AcknowledgeAt`, plus command/ack call sites needed to interpret decoder output; `84c3501999c5abd8a4e9dda48507e15f9e2129eec6a82a6164090cff46d9ab43` | Dynamic subscription success is evidenced as one provider status element per requested channel, and a paired `T.SYM,Q.SYM` request becomes complete only after both ordered successes. | `behavior evidence` | Do not port `TAQState`, desired/subscribed maps, coverage, features, pressure, wall clock, or snapshot state. Component 5 emits status/command facts; Component 9 later interprets them under engine ownership. | `P-C5-STATUS`; transport `P-C5-COMMAND` |
| `internal/massive/taq_correction_test.go` — `ackNext`, `TestTAQCorrectionPartialAckTimeoutIsTerminalWithoutRetry`; `de514bd341ff11d4fe4d80339c061e18b3bdbd33c167b2ec308594aa335f4cd6` | Two causal successes are required for a paired T/Q command; a partial acknowledgement cannot be called coverage and leaves provider membership ambiguous. | `behavior evidence` | Keep only the acknowledgement counterexample. Correction, ranking churn, measurements, timeouts, retry policy, and membership consequence belong to Components 8/9. | Transport `P-C5-COMMAND` |
| `internal/massive/taq_provider_gates_test.go` — offline-only `TestTAQProviderGateConditionNormalizationANDAndHash`, `TestTAQProviderGateAcknowledgementPaths`, `TestTAQProviderGateQuoteDispositionPolicy`, `TestTAQProviderGateQuoteRejectionsRemainSymbolScoped`, `TestTAQProviderGateLifecycleAndOverallClosedOutcomes`; `bcb97a6a10b89a093f1613fc96ff912d711b3675d6861455aa98d383e062ccff` | Offline assertions confirm exact paired acknowledgements, wrong/extra epoch failure, quote-shape versus feature-policy separation, symbol-local recognized T/Q rejection, and explicit inconclusive lifecycle evidence. The same file also contains live/REST/credential functions that were not opened. | `behavior evidence` | No code or whole-file test port. Exclude live gate, REST comparison, selection, artifact, feature policy, and environment behavior. | `P-C5-TRADE`, `P-C5-QUOTE`, transport `P-C5-COMMAND` |
| `internal/massive/trade_conditions.go` and `trade_conditions_fixture.json`; `7d25eaae93dce5eb2cec2eae8babd9d8257c0a8d090335ac48c812d011064034` and `290afe50faab29058cb7d98b278139b7f2946e185dd6b78509303e3d6ddb7050` | The fixture is a sealed 55-rule `updates_volume` classification dated 2026-08-03. | `reject` | Component 5 preserves bounded raw condition codes and evidence quality only. `updates_volume`, reviewed-condition inclusion, Tape Rate, and provider-condition refresh are Component 9 rules. The fixture is not whitelisted. | `P-C5-TRADE` proves unknown/unclassified preservation, not feature eligibility. |

### 8.1 Existing evidence strength and limitations

The v2 payloads are useful regression evidence for field names, integer
exactness, millisecond timestamps, decimal trade size, paired channel
acknowledgements, and failure localization. They do not prove current Massive
schema stability, entitlements, provider SLA, actual arrival latency, full
condition/lifecycle semantics, or production capacity. No raw recording or
live provider observation was authorized. The new proofs therefore use offline
fixtures and make no live-conformance claim.

V2's decoder cannot be directly ported: it mutates a predecessor engine,
health, recovery, and T/Q state; treats a JSON object as a map and therefore
cannot reject duplicate recognized members; assigns a decoder-local delivery
sequence rather than the Phase 1 live tuple; and can apply array elements
causally after an earlier ambiguity before failing the epoch. Those are
ownership or ordering conflicts, not reusable architecture.

One reconnaissance variance must be acknowledged at completed-contract
approval: a targeted range read of `taq_correction_test.go` displayed
non-command Component 9 correction/ranking tests adjacent to the authorized
acknowledgement helpers. Those declarations were not used as evidence,
whitelisted, or allowed to change this contract. No credential, live function,
network call, or additional file was involved. The scope-conformance checklist
therefore remains open pending the owner's explicit acknowledgement.

## 9. Detailed semantic inputs, outputs, and owned state

### 9.1 Pure frame classifier

The classifier accepts one immutable raw frame value containing:

- exact Component 1 binding identity;
- positive adapter-allocated process-local connection epoch already reported to
  the engine as the current attempt;
- positive per-epoch frame sequence assigned when the raw frame enters the
  bounded adapter FIFO;
- one UTC `received_at` captured immediately after the socket read completes;
  and
- a copied UTF-8 byte sequence no larger than the configured hard frame bound.

It returns a finite provider-order sequence of closed classified results. Each
result carries `(epoch,frame_sequence,array_index)`, where `array_index` is the
zero-based JSON-array position. The result is exactly one of normalized A, T,
Q, provider status/control, attributable recognized-item rejection, or global
ingress ambiguity. There is no generic payload, mutable map, callback, engine
reference, or independent sequence.

The top level must be one JSON array with no trailing value. An empty array is
a classified empty frame with a completed no-item disposition. Every element
must be an object with one unambiguous string `ev`. Unknown members inside a
recognized event are additive. A duplicate recognized member is rejected; a
map-decoder's silent last-member-wins behavior is prohibited.

Classification is causal. Recognized attributable rejections allow later
array elements to continue. A non-UTF-8 frame, invalid/non-array/trailing JSON,
nonobject element, missing/malformed `ev`, or unknown `ev` creates an ingress-
ambiguity result at the earliest knowable position and stops classification of
later elements in that frame. Earlier complete results remain ordered facts;
later bytes are fenced because aggregate/control loss cannot be excluded.

### 9.2 Normalized aggregate

An `A` result is one `engine.AggregateInput` with:

| Item | Exact mapping |
| --- | --- |
| Envelope | `engine.AggregateSchemaV1`, binding identity, `AggregateSourceLive`, exact symbol, frame receipt as delivery time, and `engine.LivePosition{epoch,frame_sequence,array_index}`. |
| Time | `s` and `e` are required exact JSON integers documented by the v2 evidence as Unix milliseconds. They must decode exactly, be UTC whole seconds, satisfy `e=s+1000ms`, and lie in bound `[S,E)`. Units are never inferred by magnitude. |
| Price/value | Required finite positive `o/h/l/c/vw`; required finite nonnegative economic volume from decimal-string `dv` when present, otherwise numeric `v`; required exact integer `z >= 0` as live Average Trade Size; provenance `engine.ATSLiveProviderAverage`. |
| Structure | `low <= open,close <= high` and `low <= high`; signed zero normalizes only for equality; no clamp, synthesized bar, inferred mark, or adjusted REST field. |

Unknown/ineligible symbol remains an engine-bound rejection because only the
installed binding is authoritative. Missing/malformed symbol is an ingress
ambiguity: it cannot be attributed to a symbol-local aggregate rejection.
Malformed numeric structure for an exact bound symbol is a symbol-local
normalization rejection and cannot invalidate unrelated array items.

### 9.3 Normalized trade

A `T` result preserves the `DTE-TRADE-01` value/delivery evidence without
performing Tape Rate eligibility:

- exact bound symbol; positive exact integer exchange `x`; nonempty trade ID
  `i` of at most 128 bytes; finite positive price `p` no greater than `1e9`;
- integer base size `s >= 0`; optional decimal string `ds` of at most 32 bytes
  is finite, `0 < ds <= 1e12`, and `floor(ds) == s`; without `ds`, `s` must be
  positive; economic size preserves `ds` when present;
- optional condition `c` is absent/null or an array of at most 16 exact
  integers; unknown codes are retained with `conditions_classified=false`
  unless a later approved classification source establishes them;
- participant `pt` and SIP `t` are exact Unix-millisecond integers. A value is
  time-valid only in `[S,E)` and no later than `received_at+250ms`. Participant
  time is effective only when valid and not later than valid SIP; otherwise
  valid SIP is used with `sip_fallback`. Neither valid time rejects the trade;
- optional provider sequence `q` is exact nonnegative; optional tape `z` is
  exact `1..3`; absent/malformed optional values carry explicit presence or
  unclassified evidence rather than guessed zeros;
- `trfi` and `trft` must be present together and positive exact integers to
  establish TRF presence and ID; otherwise identity is explicitly
  unclassified; and
- absent `e` means `original`. Present `e` produces
  `unsupported_lifecycle` plus bounded JSON type/evidence (scalar text at most
  32 bytes); it does not guess a correction/cancellation or referenced ID.

The 250 ms tolerance is the explicit provider-adapter tolerance evidenced by
v2. It changes only validity; it never changes the stored event timestamp or
market-window membership. Changing it requires contract revision or later
approved provider evidence.

### 9.4 Normalized quote

A `Q` result preserves the `DTE-QUOTE-01` observation without deciding Spread
validity:

- exact bound symbol and exact Unix-millisecond SIP `t` in `[S,E)` and no later
  than `received_at+250ms`;
- required finite positive bid `bp` and ask `ap`, each no greater than `1e9`;
- optional positive exact integer bid/ask exchange `bx/ax` and size `bs/as`,
  with explicit presence and unclassified evidence; absent/invalid ancillary
  metadata does not turn a complete price observation into a fabricated zero;
- `c` may be one exact integer or at most 16 exact integers; `i` may be at most
  16 exact integers. Absent, null, scalar, array, and unclassified shapes remain
  distinguishable without retaining raw JSON;
- optional nonnegative sequence `q` and tape `z` in `1..3`, with presence; and
- frame receipt and live causal position.

Normalization does not reject locked or crossed quotes, select NBBO, or label
metadata as Spread-eligible. It only reports the faithful structural facts and
quality evidence that Component 9 must interpret.

### 9.5 Normalized provider status/control

Provider `status` elements retain only bounded normalized fields: connection
epoch, causal position, receipt time, status class, applicable adapter command
kind/token supplied by transport correlation, and acknowledged/failed/
ambiguous disposition. Provider prose, URLs, credential material, and unknown
unbounded fields never enter a returned fact or diagnostic.

`connected`, `auth_success`, and `success` are distinct phase values. Success
in the wrong phase is not accepted. Any other nonempty status is a provider-
failure class, not a retriable/fatal policy decision. Exact command correlation
and acknowledgement cardinality are controlled by the transport detail.

### 9.6 Construction guarantees and runtime validation

Construction makes the event family, source, timestamp unit, and live-position
shape closed. Pure normalizers receive no engine pointer and cannot mutate
canonical state, active epoch, lifecycle, coverage, `T`, ranking, or
publication. Complete immutable values and bounded rejection enums prevent a
partial normalized object from entering the success path.

Runtime still must reject hostile JSON syntax, duplicate recognized members,
oversized strings/arrays, nonexact integers, nonfinite or structurally invalid
numbers, wrong session/symbol context, missing identity, future timestamps past
the explicit tolerance, and ambiguous status phases.

## 10. Required behavior

| Requirement | Behavior | Authority and evidence |
| --- | --- | --- |
| `C5-CLASS-01` | Classify one bounded frame into exact provider-order A/T/Q/control/rejection results with DTE live positions. Stop and emit ingress ambiguity at the first item whose family cannot be known; never silently skip or reorder an item. | `ARCH-FLOW-01`–`03`, `DTE-EVENT-01`, `DTE-EVENT-02`, `DTE-REJECT-01`; v2 `Decoder.Process` and focused tests, adapted for causal failure order and duplicate keys. |
| `C5-AGG-01` | Map each valid `A` element atomically to the existing Component 2 live aggregate input/value, including exact millisecond window, `dv`/`v`, live `z`, and ATS provenance, or one bounded attributable rejection. | `DTE-AGG-01`–`03`, `DTE-WINDOW-02`; v2 `decodeAggregate`; Component 4 value seam. |
| `C5-TRADE-01` | Map each valid `T` element to the complete Phase 1 trade fact with exact effective-time fallback, fractional economic size, identity/condition/lifecycle evidence, receipt, and causal position, or a bounded channel/symbol-local rejection. Do not decide Tape Rate eligibility. | `DTE-TRADE-01`, `DTE-TRADE-02`, `PG-AVAIL-03`; v2 `decodeProductTrade` and offline tests. |
| `C5-QUOTE-01` | Map each valid `Q` element to the complete Phase 1 quote observation and metadata-quality evidence, or a bounded channel/symbol-local rejection. Preserve locked/crossed/one-sided distinctions for Component 9; do not decide Spread. | `DTE-QUOTE-01`, `DTE-QUOTE-02`, `PG-AVAIL-03`; v2 `decodeProductQuote` and offline tests. |
| `C5-STATUS-01` | Normalize status/control elements with exact phase, current command correlation context, causal position, and bounded disposition; reject wrong phase/count/epoch and retain no arbitrary provider prose. | `DTE-CONTROL-01`, `LIFE-MODEL-04`; v2 `DecodeStatusCount`, acknowledgement tests, and privacy assertion. |

### 10.1 Consequential trust-boundary acceptance

| Boundary | Accept into success only when | Reject or contain | Dangerous false-success case |
| --- | --- | --- | --- |
| Raw provider frame | Copied bytes are within the frame bound, UTF-8, exactly one JSON array with no trailing value, and every success-bearing element is strictly decoded. | Frame syntax/family ambiguity closes aggregate-ingress trust at the earliest causal item; recognized attributable bad A/T/Q is local. | A duplicated `ev`, `sym`, time, or price member silently using last-value-wins and looking like a valid aggregate/control event. |
| Normalized market fact | Required identity/value/time fields are complete, finite/exact as applicable, session-valid, and carry binding, receipt, and live position. | Atomic zero-value rejection; no partial fact escapes. | A millisecond timestamp interpreted by magnitude as another unit, or a malformed decimal size becoming zero. |
| Status/ack fact | Recognized status is in the expected phase and correlated to the sole pending command/expected count for the same epoch. | Wrong/extra/partial/unsolicited status is failed or ambiguous, never acknowledged. | A generic `success` received after the wrong command establishing A.* or T/Q coverage. |
| T/Q quality evidence | Raw codes/shapes and presence are bounded and faithfully retained. | Unknown metadata stays unclassified; feature validity deferred. | Unknown trade condition or quote indicator guessed eligible and later appearing current. |

## 11. Failure and terminal behavior

Pure normalization has no retry or waiting state. Every classified array
position yields one complete normalized fact, one attributable rejection, or
the single earliest ingress-ambiguity result that fences the remainder of the
frame. It never logs or retains raw payloads as an error path.

An attributable malformed aggregate for a known bound symbol is symbol-local.
A recognized malformed T/Q item is channel/symbol-local because its `ev`
proves aggregate/control was not hidden. Missing/malformed aggregate identity,
unknown/missing event family, or frame syntax ambiguity is epoch-level because
the scanner cannot prove which aggregate/control fact was lost. Status failure
scope is determined by its correlated aggregate or T/Q command; the
normalizer itself does not transition lifecycle or coverage.

## 12. Accounting and observability

This detail owns no market population. For a structurally established array,
one frame classification reconciles:

```text
array_elements_examined
  = normalized_A
  + normalized_T
  + normalized_Q
  + normalized_control
  + attributable_rejection
  + ingress_ambiguity
```

If `ingress_ambiguity=1`, elements after its array index are `fenced_remainder`
and are not included in `array_elements_examined`; the frame-level identity is:

```text
declared_array_elements
  = array_elements_examined + fenced_remainder
```

`array_cardinality_known=true` is required for the second identity. A
non-array frame, truncated array whose remaining element count cannot be
known, or trailing frame-level value records one separate
`frame_ingress_ambiguity` and never fabricates a declared array element merely
to make the arithmetic reconcile. The causal ambiguity result is still
returned at its earliest knowable position.

Event family is mutually exclusive. Rejection reason, field presence,
condition quality, and T/Q channel are overlapping bounded diagnostic
dimensions. Reasons are closed enums; symbols and raw provider text are not
metric labels. Raw payloads are not retained. Exact metric names belong to
Component 8.

## 13. Simplicity and boundedness

- The classifier and all normalizers are stateless, call-local functions.
- One copied frame is bounded by at most 8 MiB. Strict decoding uses a bounded
  first pass to establish causal/status cardinality, then a serialized cursor
  that retains one raw-element view and one result at a time. It does not
  materialize a result slice, delivery slice, or second complete decoded tree.
- Symbol and trade ID use the stated 64/128-byte bounds; condition and indicator
  vectors contain at most 16 exact integers; correction scalar evidence is at
  most 32 bytes; status/rejection vocabularies are closed.
- V2 mapping behavior is adapted because it is narrower than rediscovery, but
  v2 stateful decoder/observer/metric/orchestration structure is rejected.
- A streaming strict array/object decoder plus direct typed outputs is simpler
  than a generic event bus, reflection registry, raw journal, or decoder-owned
  scanner facade.

No database, service split, plugin framework, generic event infrastructure,
provider-condition fetcher, or speculative wire fallback is added.

## 14. Evidenced edge cases

| Edge case | Evidence | Required behavior | Primary proof |
| --- | --- | --- | --- |
| `dv` decimal string and `v` fallback; zero live `z` | V2 `TestDecoderMassiveSchemaBatchOrderingAndFallback`; Phase 1 live ATS mapping | Preserve fractional volume; prefer valid `dv`; accept structural ATS zero with live provenance; never use REST transaction-count mapping. | `P-C5-AGG` |
| Mixed A/T/Q array | V2 `TestDecoderBatchedAggregateTradeQuoteAndTAQValidation`; DTE causal-order proof requirement | Assign one frame sequence and increasing array index; preserve exact item order and independent family outcomes. | `P-C5-CLASS` |
| Unknown/malformed family between valid items | V2 neighbor-preservation test plus Phase 1 ingress-integrity invariant | Preserve earlier valid items, emit ambiguity at its index, and fence later items rather than applying them ahead of the failure. | `P-C5-CLASS` |
| Duplicate recognized JSON member | External-trust invariant and Component 4 strict-normalization precedent | Reject rather than silently pick first/last. | `P-C5-CLASS`; family proof asserts atomic rejection |
| Fractional trade `ds` with integer base `s` | V2 decimal-size test | Preserve economic fraction only when exact shape/bounds/floor relation pass. | `P-C5-TRADE` |
| Participant later than SIP or one timestamp invalid | `DTE-TRADE-02`; v2 `decodeProductTrade` | Use valid participant only when not later than SIP; otherwise record SIP fallback; reject if neither valid. | `P-C5-TRADE` |
| Unknown condition/lifecycle evidence | Phase 1 explicit unknown requirement; v2 offline lifecycle outcomes | Preserve bounded unclassified evidence; never guess feature eligibility or lifecycle reconstruction. | `P-C5-TRADE` |
| Locked, crossed, absent ancillary quote metadata | `DTE-QUOTE-02`; v2 offline quote policy tests | Normalize structure and quality separately; do not decide Spread validity here. | `P-C5-QUOTE` |
| Partial, extra, wrong-epoch, or wrong-phase success | V2 acknowledgement/status tests | Never report completed acknowledgement; return failed/ambiguous bounded control evidence. | `P-C5-STATUS` |
| Provider error containing URL/prose | V2 status privacy test; bounded observability invariant | Retain only a fixed status class and command context. | `P-C5-STATUS` |

The proof allocations and slice acceptance criteria are authoritative in
[proof and delivery plan](proof-and-delivery-plan.md).
