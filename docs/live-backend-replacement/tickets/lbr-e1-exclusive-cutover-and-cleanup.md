# LBR-E1 implementation assignment — exclusive cutover and cleanup

**Status:** Draft for independent cross-check; not an active ledger entry or
implementation authorization.

**Activation gate:** `LBR-P1` and Capabilities A–D, including their final
reviews, must be accepted. The owner must decide replay/checkpoint deletion
versus separated unsupported tooling. The
[delivery program](../delivery-program.md) must mark only `LBR-E1` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), [T/Q](../tq-state.md), and
[ingress](../live-ingress.md) contracts and handoffs, and the complete
[integration contract](../integration-removal-and-acceptance.md). This ticket
implements only `LBR-E1` and `P-LBR-E1-CUTOVER`.

Controlling requirements are `LBR-ARCH-01`, `LBR-ARCH-02`, `LBR-ARCH-11`,
`LBR-ARCH-13`, `PG-UI-01`, `PG-UI-02`, and every preservation/removal
obligation allocated by accepted A–D. API v2, dashboard, launcher, reference,
fresh hydration, and operator outcomes remain fixed.

## 2. Outcome, owner, scope, and dependencies

The ordinary private scanner constructs exactly one accepted replacement
binding/hydration/ingress/engine/evaluator/TQ/publication path. Remove every old
live state, evaluator, queue, envelope, fallback, compatibility selector, and
removed product feature from that production dependency graph. Preserve one
read-only API-v2/dashboard/launcher composition.

The orchestrator supplies the owner replay/checkpoint source decision before
activation. This slice applies it without adding a live gate or compatibility
claim. Resource/stability acceptance is deferred to E2; provider access to E3.

## 3. Allowed implementation boundary

This cutover may edit `cmd/scanner`, `internal/operations`, `internal/engine`,
`internal/massive`, `internal/snapshotapi`, retained reference/UI/launcher
composition tests, build/config/scripts, and delete superseded live files/tests.
It may edit replay/checkpoint packages only as required by the recorded owner
choice. Historical documents are not broadly rewritten; active maps/README may
be corrected through the sole ledger by the orchestrator.

Any production behavior change beyond wiring/deletion or an accepted A–D seam
must reopen the owning focused contract rather than be hidden in integration.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Evidence is the accepted A1–D2 primary proofs,
API-v2/UI/launcher goldens, snapshot-isolation correction, production
dependency inspection, and the baseline manifest at commit `0d043c1`. Old
code/tests are removal evidence only. No provider credentials or requests.

## 5. Required implementation behavior

- Construct the accepted live-only replacement directly; remove flags,
  environment/build-tag/error fallbacks, shadow mutation, and runtime selection
  between old/new paths.
- Prove one mutable market owner, only parent-justified bounded concurrency,
  one decoded-batch ring, one immutable publication, and component-defined
  failure domains in ordinary composition.
- Delete all A–D code declared removable, temporary adapters/oracles from
  production packages, removed field state/tests/fixtures, and obsolete active
  specification routing.
- Preserve test-only differential oracles only when isolated from production
  imports, goroutines, state, and runtime choice.
- Remove replay/checkpoint participation from ordinary scanner, live engine,
  operations, massive, and snapshot capture. If tooling is retained, prove its
  dependency graph cannot reach the replacement engine/scanner and label it
  unsupported without a claim.
- Keep checkpoint mode off. API v2 may expose its required fixed honest
  disabled/not-installed object from configuration, never engine checkpoint
  state; live replay object is absent.
- Preserve `/api/v2/snapshot`, `/livez`, `/readyz`, loopback/CORS/bounds,
  lock-independent capture, exact dashboard model/transport labels, and
  scanner independence from API/UI child failure.
- Ordinary build/test/dependency inspection must fail on any old path sentinel,
  removed field, second queue/owner/publication, or unsupported tool import.

## 6. Primary proof and acceptance distinction

`P-LBR-E1-CUTOVER` combines build/dependency/source exclusion with a real
ordinary composition trace through startup, hydration/fence, current output,
T/Q, recovery/terminal, API/UI failure, and shutdown. It asserts one accepted
owner/queue/evaluator, only allowed concurrency, exact failure domains, no old
sentinel/import/state/field, fixed checkpoint-off output, and unchanged API-v2/
UI/launcher goldens.

Dangerous counterexamples are a fallback selected after an error, a shadow
mutation, a second live queue, or unsupported tooling importing the live core.
The proof does not establish 30-minute resource stability or provider wiring.

## 7. Verification and timeout policy

Run the cutover/exclusion proof, complete A–D semantic regressions, API/UI/
launcher tests, affected short/race packages under five minutes, focused vet,
dependency/source checks, `git diff --check`, and ordinary
`go test -count=1 -short -timeout 2m ./...`. No E2 soak or provider tier. Delete
or revise lower-authority tests that exist only for removed private behavior.

## 8. Removal and handoff

Acceptance requires actual deletion or proven standalone separation, not
unreachable-looking code. Handoff lists every removed/retained package,
production dependency graph result, replay/checkpoint decision application,
one-path runtime identity, exact compatibility proof/limitation, and whether
the frozen E2 manifest remains valid.

## 9. Implementer discretion and prohibited changes

Mechanical wiring, package consolidation, config removal, sentinel/dependency
test mechanics, and exact deletion order are delegated. Do not change product
formulas, market time/precedence/currentness, provider protocol, API/UI meaning,
resource acceptance, or repair replay/checkpoints. Do not add a database,
service split, bus, plugin system, public deployment, or compatibility layer.

## 10. Containment, review, and correction

Compile-time dependencies and closed constructors should prevent alternate
production paths. Runtime composition proof contains remaining failure cases.
Any semantic/interface regression reopens the lowest A–D slice; do not patch it
locally in E1. E1 requires proportionate review for cross-cutting owner/
concurrency/failure conformance, followed later by final integrated review
after E2. Only the orchestrator records acceptance and commits after
quiescence.
