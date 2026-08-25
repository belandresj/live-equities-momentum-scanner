# Repository instructions

## Authority

Read `README.md`, `docs/specification-map.md`, and the relevant approved
product, architecture, delivery-program, and component documents before
changing code or contracts. The current program is
[`docs/live-backend-replacement/delivery-program.md`](docs/live-backend-replacement/delivery-program.md).

Authority descends in this order:

1. owner-approved product specifications;
2. owner-approved architecture specifications;
3. owner-approved delivery-program authority for its stated scope;
4. focused component contracts; and
5. implementation assignments, plans, tests, fixtures, benchmarks, and code.

When a lower-level artifact conflicts with a higher-level authority, revise the
lower-level artifact and continue. Do not preserve a test, fixture, benchmark,
component decision, or accepted implementation merely because it was approved
earlier. For C7-C11, apply the V1 program's zero-interruption policy and strict
compatible interpretation of controlling Phase 1 authority; never invent a
component-local owner gate.

## Current phase

The product contract and the
[`live backend replacement architecture`](docs/live-backend-replacement.md)
are owner-approved. The accepted feature-MVP, numbered components, stability
corrections, and private/local baseline remain reusable evidence, not the
active implementation plan. Replay is unverified and checkpoint persistence is
disabled; neither is a replacement gate.

The private/local
[`live backend replacement`](docs/live-backend-replacement/delivery-program.md)
is complete for its defined scope: Capabilities A–E, the exclusive E1 cutover,
the corrected E3 provider observation, and the final integrated review are
accepted. The invalid synthetic E2 execution is preserved, while E2
deterministic capacity characterization remains deferred and non-gating.

Any new implementation scope must retain one active write-capable slice and
the applicable final review. Public deployment, replay repair, and checkpoint
compatibility remain outside current authority. Every future credentialed
provider request requires a new exact owner authorization.

## Replacement fixed and revisable decisions

For the current replacement, fixed behavior, revisable delivery choices,
resource-target policy, proof/review allocation, and completion boundaries in
the [parent architecture](docs/live-backend-replacement.md) and
[delivery program](docs/live-backend-replacement/delivery-program.md) control.
The former live-feature MVP and V1 material remains compatible historical
evidence; it does not preserve an old representation, queue, feature state,
replay path, checkpoint shape, or delivery gate.

The fixed product/architecture meanings and agent-revisable delivery decisions
are authoritative in
[`docs/v1-release-program.md`](docs/v1-release-program.md#2-fixed-product-behavior-and-revisable-delivery-decisions).

For C7-C11, the program orchestrator may revise component contracts, document
maps, internal interfaces, V2 whitelists, fixtures, proof allocation, test
scale, benchmarks, performance targets below a fixed product limit, slice
boundaries, review routing, implementation mechanics, and accepted lower-level
decisions. Record the revision and affected evidence in the component parent.
This authority does not permit changed market semantics, another state owner,
T/Q-to-ranking dependence, fabricated availability/readiness, or unrelated
scope.

The containment and fallback rules in
[`docs/v1-release-program.md`](docs/v1-release-program.md#3-zero-interruption-execution-policy)
are mandatory for C7-C11. A component contract, assignment, failed proof,
benchmark, review, prior approval, progress update, or milestone commit must
not create an owner-interruption gate.

## Focused component contracts

Contract-first design remains mandatory:

1. Read this file, the repository guide and map, the current delivery program,
   relevant Phase 1 authorities, and accepted dependency contracts.
2. Enumerate the exact controlling `PG-*` and `LBR-ARCH-*` IDs plus any routed
   compatible `DTE-*`/`LIFE-*` semantic dependencies.
3. State one component ownership boundary and explicit non-scope.
4. Choose the simplest design satisfying every cited requirement and current
   replacement outcome.
5. Introduce no new product rule, competing owner, watermark, evaluator,
   T/Q-to-ranking dependency, changed time-window meaning, speculative edge
   case, or duplicated responsibility.
6. Before inspecting Version 2, record a compact Phase 1 boundary and the exact
   reconnaissance questions, source categories, and exclusions. The C8-C11
   plans already supply the initial owner-approved boundary; revise the plan
   first if evidence or a stable dependency changes it.
7. Inspect only recorded Version 2 scope. Record exact source paths/functions/
   fixtures, provenance where relevant, reuse decision, preserved behavior,
   coupling to remove, and required proof. Version 2 is evidence, never
   authority.
8. Complete the focused contract with exact requirements, trust boundaries,
   primary proofs, verification tiers, sequential slices, review triggers,
   discretion, deferrals, and correction conditions.
9. Prefer one compact contract and no more than two slices for C8-C11. Add a
   third only when a distinct provider/canonical, trust, ownership, or proof
   boundary cannot be coherently implemented and reviewed in two.
10. Record the completed contract as the current executable plan. Under the
    replacement program this does not freeze it: later evidence invokes the
    correction loop rather than an owner stop.

Use the mandatory
[`focused component specification template`](docs/specifications/focused-component-spec-template.md).
The replacement parent controls shared ownership/topology; existing approved
data/time/lifecycle semantics remain compatible evidence until routed to one
focused replacement home. Do not create another shared architecture or
core-domain spec.

### Modular focused contracts

A modular set is one component contract and one authority. Its parent remains
at the path listed in the specification map and is the mandatory entry point.
It owns the outcome, single boundary, non-scope, cross-cutting invariants,
document map, and sole delivery ledger.

Each subordinate spec owns one cohesive semantic, trust, proof, or delivery
boundary; names its parent, exact requirements, dependencies, and allocated
slices; and avoids restating normative text owned elsewhere. Every requirement,
decision, evidence/reuse decision, primary proof, and slice has one
authoritative home. Dependencies are explicit and acyclic.

For localized work, read the parent and only the routed details/dependencies.
Read the complete set for cross-cutting contract changes and final component
review. Within C7-C11, a document-map change is agent-revisable under the V1
correction loop. Outside that program, follow the applicable owner approval
gate.

## Replacement correction and acceptance

Use the correction and acceptance policy in the current
[delivery program](docs/live-backend-replacement/delivery-program.md#13-correction-and-acceptance).
A failure records evidence; it does not freeze the failed premise. A numeric
resource-target miss alone receives the program's one bounded diagnostic/
correction/rerun response; it cannot create an indefinite optimization loop
when hard behavioral and bounded-plateau acceptance passes.

An accepted slice means its current proofs passed. If later evidence exposes a
defect, mark the slice/component `reopened`, name the invalidated claim, preserve
unaffected evidence, revise the lower-level artifact, and rerun the narrowest
distinguishing proof. A reopened earlier slice does not authorize concurrent
implementation or a second state path.

At each slice gate, record:

- coherent behavior now available and exact requirements/proofs completed;
- changed ownership/interfaces and behavior still deferred;
- proof design, result, dangerous counterexample, and limitation;
- important invalid states prevented by construction;
- success and failure-path walkthrough;
- deviations, failed assumptions, or inspection-only claims;
- whether a narrow independent review was triggered and its result; and
- whether the next slice remains valid or needs an in-program revision.

Final capability/component acceptance requires its allocated proofs,
proportionate verification, one final read-only review, a clean conformance
walkthrough, and no unresolved fixed-authority conflict. It may later be
reopened by integration evidence before final current-program acceptance.

## Program orchestration and Git

The current delivery goal may use one write-capable implementation worker at a
time. Independent reviewers are read-only. The orchestrator updates ledgers,
stages, or commits only while all workers/reviewers are quiescent.

The orchestrator alone stages exact paths after verifying unrelated user
changes remain untouched. Make local commits at coherent planning/correction
milestones, accepted implementation slices, final capability/component
acceptance, and distinct vertical milestones. Do not push, rebase, amend,
rewrite history, delete branches, or use destructive reset operations.

## Engineering rules

### Language and tooling

- Use Go 1.26 with standard Go module tooling.
- The module path is
  `github.com/belandresj/live-equities-momentum-scanner`.
- Prefer the Go standard library; add a third-party dependency only when the
  current focused contract and replacement slice require it.

### Architecture and implementation

- Keep one authoritative `ScannerStateEngine` and one canonical symbol state.
- Provider adapters normalize facts; they do not determine ranking readiness.
- REST and live aggregates use the same canonical identity and merge rules.
- Aggregate ranking never depends on trade/quote availability or T/Q health.
- Successful empty hydration is explicit no-print evidence, not a fabricated
  mark or unfinished work.
- Every primary population counter participates in a documented accounting
  identity. Overlapping feature dimensions are labeled separately.
- Checkpoints represent one coherent committed timestamp.
- The UI remains independently runnable from the scanner backend.
- Do not introduce a database, service split, generic event bus, runtime plugin
  system, generalized framework, or hypothetical extension point without an
  approved architectural need.
- Prefer the minimum number of mutable states, owners, representations, and
  handoffs.

## Evidence, edge cases, and tests

Do not implement a nontrivial provider edge case without provider
documentation, a captured/approved fixture, a production observation, a
product or mathematical invariant, a predecessor regression, or explicit owner
approval.

Each requirement has one primary proof. A second layer is justified only when
it proves a distinct boundary. One proof may cover tightly coupled requirements
when it separately states each claim, dangerous counterexample, observable
result, participating path, and limitation.

For an external, persisted, or consequential cross-component boundary, define
what reaches success, what is rejected/contained, the failure domain, and the
smallest invalid evidence that could falsely appear complete/current/valid.
Distinguish invalid states prevented by construction from representable states
requiring runtime validation.

The ordinary repository command and the current acceptance, capacity,
deterministic-fixture, and live tiers are authoritative in the
[`implementation process`](docs/implementation-process.md#6-verification-tiers-and-cost-policy)
as bounded by the current delivery program.
In particular:

- ordinary verification is `go test -short -timeout 2m ./...`;
- ordinary tests use the smallest deterministic population and stream that
  prove the claim;
- capacity, benchmark, soak, race, and live work is explicitly selected and
  skipped by `testing.Short()` when long-running;
- loops, retries, channel waits, polling, and generators are bounded;
- every long command and each performance trial has an explicit timeout;
- no local acceptance command exceeds 15 minutes without a recorded specific
  justification;
- validate fixture bytes, symbol count, correction count, interval, state
  families, and planned work before timing;
- run the narrowest affected proof during correction and do not rerun the full
  repository after every edit;
- reuse unchanged expensive evidence; and
- separate semantic correctness from capacity/latency measurement.

Never access provider credentials or make live provider requests without an
explicit owner authorization for that exact task. The approved procedure in
[`docs/market-hours-validation.md`](docs/market-hours-validation.md) does not
itself authorize execution.

## Predecessor reuse

The only predecessor permitted by default is:

```text
/Users/joshuabelandres/Dev/Momentum-Equities-Live-Scanner-v2
```

Do not inspect the older Version 1 checkout without an explicit owner-approved
exception for a precise purpose. Inspect Version 2 only after the current
component records its Phase 1 boundary and exact reconnaissance scope. A V1
program orchestrator may revise that scope before inspection and record the
change; it may not use unrecorded sources.

Implementation may use only the current recorded whitelist. A whitelist can be
revised inside C7-C11 when new evidence remains within the component and fixed
authority. Record the new source, reason, provenance, adaptation, and proof.

## Implementation assignments

One assignment covers exactly one current slice and states:

1. authoritative documents and exact requirements;
2. outcome, scope, non-scope, dependencies, interfaces, and owner;
3. allowed files/packages or ownership boundary;
4. current V2 source/fixture whitelist and evidence inputs;
5. one primary proof per assigned requirement;
6. verification tier, timeout, and acceptance record;
7. deferred behavior and decisions delegated to the implementer;
8. prohibited fixed-authority changes;
9. consequential trust-boundary counterexamples and proof limitations; and
10. the current-program containment reference plus in-program correction
    triggers.

Implementers make routine lower-level choices within those bounds. If the
current slice is no longer coherent, revise/split it through the current
program's correction loop, record the change, and continue sequentially. This
repository does not use an ADR workflow.

## Independent review execution

Use the risk-based cadence in the
[`implementation process`](docs/implementation-process.md#7-independent-review-cadence).
Do not spawn a reviewer merely because a slice or contract changed.

When a review is required and the model is available, use `gpt-5.6-sol` with
medium reasoning. Use high reasoning only when medium leaves concrete
uncertainty about concurrency linearization, persistence/atomicity, invalid
external evidence reaching false success, sole-owner enforcement, or a cross-
component authority conflict. Record any substitution.

After corrections, reuse the same reviewer when practical for a focused
re-review of the finding and affected boundary. Reviewers do not edit, approve,
stage, commit, or expand scope. A review finding is correction input. If the
preferred model is unavailable, record a substitution; if review capacity is
temporarily unavailable, complete other work and retry without asking the
owner to unblock it.

One final read-only review is required per completed current-program
capability, plus one final integrated review at current-program completion.
