# LBR-D2 implementation assignment — one ingress handoff

**Status:** Independently reviewed implementation assignment. Activation occurs
only through the [delivery program](../delivery-program.md); this file is not a
mutable ledger.

**Activation gate:** `LBR-D1` and Capabilities A–C must be accepted; the
[delivery program](../delivery-program.md) must mark only `LBR-D2` active.

## 1. Authority and exact requirements

Read [`AGENTS.md`](../../../AGENTS.md), the
[parent architecture](../../live-backend-replacement.md), the
[delivery program](../delivery-program.md), the accepted
[canonical](../canonical-state-and-hydration.md),
[evaluation](../evaluation-and-publication.md), [T/Q](../tq-state.md), and
complete [ingress](../live-ingress.md) contracts, plus the recorded D1 handoff.
This ticket implements only `LBR-D2` and `P-LBR-D2-HANDOFF`.

Controlling requirements are `LBR-ARCH-03`, `LBR-ARCH-10`,
`LIFE-RECOVER-01`, and the ingress-local concurrency/failure obligations of
integration-owned `LBR-ARCH-02` and `LBR-ARCH-11`. Preserve the heartbeat and
data-confirmed corrections routed by the focused contract.

## 2. Outcome, owner, scope, and interfaces

One bounded decoded-batch FIFO carries provider batches and causal markers from
one socket reader to the sole engine loop. The raw-frame queue, intermediate
adapter deliveries, and second general engine live FIFO disappear. Ingress
owns ordering/transport only; the engine owns every market consequence,
currentness, recovery decision, and immutable publication.

Inputs are D1 batches, engine attempt/TQ/fence/pressure/cancel commands, and
socket/heartbeat results. Outputs are serial engine consumption, write boundary
`B`, exact marker/terminal facts, and fixed queue/accounting samples. Hydration
results/timers remain only the parent-allowed owner-local paths.

## 3. Allowed implementation boundary

Allowed production paths are `internal/massive` live queue/transport/attempt/
heartbeat/command code, the general live-input queue/admission portions of
`internal/engine`, and the smallest `internal/operations` runtime/composition/
metrics seams needed to connect one ring. Delete superseded files when safe.
Focused tests may span those packages and `cmd/scanner` composition. API/UI,
launcher policy, product formulas, replay, and checkpoint are excluded.

## 4. Evidence and source whitelist

Predecessor V2 whitelist: empty. Reuse only current transport/epoch,
heartbeat/resubscription, data-confirmation, fence, pressure, capacity, and
shutdown fixtures routed by the focused spec. Old raw queue/envelopes/engine
FIFO are failure/reuse evidence only; no implementation or capacity setting is
preserved merely because historical tests accepted it. No provider access.

## 5. Required implementation behavior and bounds

- Install one 4,096-entry/64 MiB initial ring with frame <=8 MiB, batch
  <=65,536 elements/32 MiB charge, and eight-entry/64 KiB marker reserve;
  validate constructor coherence. Program correction may revise division only
  before dependent work and with finite equivalent bounds.
- Transfer immutable batches/markers by ownership, consume a complete entry
  serially, and release backing promptly. No completion backlog or second market
  queue may accumulate.
- Preserve frame/array order and append a fence only after all admitted batches
  through the greatest complete frame already read. Stale markers fence.
- Shed T/Q-only work first with exact coverage/drop facts. If aggregate/control
  batch or required marker/terminal cannot be admitted, retire the epoch,
  close currentness, and require exact gap recovery; never call it consumed.
- Serialize socket writes without a command queue; one dynamic T/Q command is
  in flight. Successful subscribe write returns exact complete-frame `B`;
  failed write returns none. Status does not complete it.
- Keep one joined socket attempt/reader and one heartbeat operation. Inbound
  progress after a failed heartbeat is diagnostic; read/fatal transport/no-
  progress deadline retires the attempt.
- Process-start dial is immediate and not a recovery ordinal. Recovery attempts
  1–5 wait exactly 1/2/4/8/16 seconds. Failure/loss of attempt 5 exhausts with
  no attempt 6. Only an accepted reconciled hydration/recovery fence resets.
- Retirement emits one exact terminal, closes/fences accepted work, cancels
  pending operations, closes socket, and joins all attempt work before redial/
  return. Queue/family/command accounting reconciles.

## 6. Primary proof and acceptance distinction

`P-LBR-D2-HANDOFF` uses a real bounded ring/engine consumer and fake socket/
clock to cover mixed order, fence placement, T/Q-first shedding, T/Q-only and
mixed saturation, aggregate/control overflow terminal/recovery, write boundary
`B`, handshake failure, heartbeat progress/no-progress, read failure, stale
epoch, one immediate plus five paced dials, no attempt 6, cancellation, and
joined shutdown.

Observe one queue/reader/attempt, exact currentness/accounting, zero silent
aggregate/control loss, marker order, retry reset only after fence, and no
goroutine/waiter. Dangerous counterexample is dropped aggregate followed by a
current fence or reset. This is deterministic transport evidence, not provider
or final 10-minute resource acceptance.

## 7. Verification and timeout policy

Run the primary proof and direct queue/transport/heartbeat/fence tests, affected
short packages, affected engine/massive/operations/cmd race under five minutes,
focused vet, `git diff --check`, and ordinary
`go test -count=1 -short -timeout 2m ./...`. Any bounded capacity trial validates
slots/bytes/events first and stays below 15 minutes. No live provider tier.

## 8. Removal and handoff

Acceptance removes `internal/massive/live_queue.go` raw-frame ownership,
adapter-delivery staging/double normalization in `live_transport.go`, the
general engine live FIFO/completion backlog, temporary D1 bridge, and duplicate
marker/terminal paths. Handoff records the final ring/interface, exact retry/
join evidence, deleted paths, bounds/limitations, and triggers Capability D
final read-only review before E1.

## 9. Implementer discretion and prohibited changes

Ring indexes/backing, socket wrapper, write critical section, fixed accounting,
and private shutdown mechanics are delegated. Do not add queues/readers/owners,
silently drop aggregate/control, change heartbeat/retry/fence/data-confirmation
semantics, weaken bounds/readiness, or add a generic bus, service split,
provider invention, replay, or checkpoint work.

## 10. Containment, review, and correction

One attempt/reader/ring, immutable ownership, and marker reserve should be
construction guarantees. Runtime validation handles stale epochs/tokens,
position regression, overflow, socket failure, and terminal duplication. Queue
saturation, marker linearization, or retirement paths that can falsely succeed
require narrow review. Findings reopen the smallest ticket/spec; final review
focuses on ordering, joins, one queue, overflow containment, and recovery. Only
the orchestrator updates the ledger or commits after quiescence.
