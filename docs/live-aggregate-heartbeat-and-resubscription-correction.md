# Live aggregate heartbeat and resubscription correction

**Status:** Accepted 2026-08-17. HR-S1 and HR-S2 are complete; final focused
review and required repository verification passed.

**Requested:** 2026-08-17, after owner observation of repeated aggregate
heartbeat termination, incomplete bootstrap hydration, and failed automatic
resubscription.

**Purpose:** Correct the narrow live aggregate transport/recovery path that can
terminate an actively delivering socket, cancel hydration, discard the reason
later handshakes failed, and repeatedly reopen exhausted recovery batches. This
is a correction to accepted lower-level runtime behavior, not a new component
or product feature.

## 1. Conclusion and observed evidence

The current incident is not a REST hydration-capacity failure. Aggregate epoch
1 ended on `heartbeat/heartbeat_failure_unclassified` while the live reader was
still receiving and dispositioning frames. The resulting bootstrap-generation
cancellation was correct; the subsequent resubscription policy was not.

The 2026-08-17 12:35 PDT diagnostic establishes:

- hydration generation 1 had 5,522 planned symbols, 54 terminal successes, no
  provider failure, and 5,468 open requests when the epoch was lost;
- the queue had zero waiting frames at the cause, peaked at 78 of 32,768 frame
  slots and 34,549 of 134,217,728 bytes, and recorded no capacity rejection;
- frames read and dispositioned advanced from 2,177 at 19:35:20.187Z to 2,734
  at 19:35:23.188Z and then to 2,786 at the 19:35:24.506Z heartbeat terminal;
  therefore inbound transport progress continued during the failed heartbeat
  operation; and
- after that loss, the running process opened greater epochs in immediate
  five-attempt batches, reached `suppressed/recovery_exhausted`, cleared the
  attempt count on the scheduled transition, and repeated. Observation stopped
  after epoch 75 without another aggregate acknowledgement.

Four earlier persisted incidents on 2026-08-14 and 2026-08-17 have the same
heartbeat terminal. Three occurred after all hydration work was terminal and
two occurred during bootstrap. This predates the current UI-only changes.

The local evidence inputs are the five bounded JSON artifacts under
`var/diagnostics/` named:

```text
live-ingress-20260814T234647.563819000Z.json
live-ingress-20260817T174857.380095000Z.json
live-ingress-20260817T181410.124293000Z.json
live-ingress-20260817T183411.191682000Z.json
live-ingress-20260817T193524.507839000Z.json
```

The accepted 2026-08-17 recovery-observability correction remains valid: it
accurately presents active generation work and preserves the first ingress
cause. Its stated limitation also remains exact: it did not identify the
heartbeat cause or prove a transport SLA. This correction uses the newly
observed inbound-progress evidence and changes only the transport/retry
consequence.

## 2. Controlling authority and corrected lower-level decisions

This correction implements the strict compatible intersection of:

- `PG-OPS-02`;
- `ARCH-FLOW-03`, `ARCH-FLOW-04`, and the aggregate-connection-loss failure
  boundary in the system overview;
- `LIFE-HYDRATE-07`;
- `LIFE-RECOVER-01` through `LIFE-RECOVER-06`;
- `LIFE-SUPPRESS-01` through `LIFE-SUPPRESS-03`;
- `TQR-HANDSHAKE-01`, `TQR-RECOVER-01`, `TQR-RUNTIME-01`, and `TQR-DIAG-01`;
  and
- the live-MVP fresh-start and same-process aggregate-gap boundary.

Higher authority requires a scheduled next action and finite recovery budget.
Repeated epoch loss must reach `suppressed` or `ended`; it must not create an
infinite inactive or repeatedly reset recovery stage. Accordingly, this spec
narrowly revises these accepted lower-level decisions:

1. `scheduled_recovery` does not reset the consecutive connection-attempt
   budget. Only successful reconciliation of the current hydration generation
   and ingress fence resets it.
2. `recovery_exhausted` does not automatically schedule another batch of
   connection attempts. The process and API remain live and honestly
   suppressed. A future explicit operator control may authorize same-binding
   recovery; absent that control, the current private-MVP recovery is a fresh
   process restart.
3. The heartbeat remains a bounded transport observation, but a failed ping
   concurrent with proven inbound frame progress is not evidence that aggregate
   coverage was lost.
4. A failed handshake's already-classified causal facts are delivered before
   the attempt terminal. They are not discarded because the overall handshake
   returned an error.

No accepted market, ranking, hydration-interval, canonical merge, T/Q
independence, or readiness meaning changes.

## 3. Ownership boundary and non-scope

The Massive adapter owns one socket attempt, strict connected/authentication/
`A.*` handshake classification, bounded heartbeat observation, causal frame
positions, terminal cleanup, and redacted attempt diagnostics. It does not own
retry policy, lifecycle, readiness, hydration planning, or ranking.

The engine remains the sole owner of epoch admission, lifecycle, recovery
budget, suppression, hydration generation, ingress fence, committed watermark,
readiness, and publication. Operations executes only engine-authorized work
after the engine-supplied deadline.

This correction does not change:

- UI behavior or presentation;
- API v2 schema or route behavior;
- T/Q desired membership, quarantine, pressure shedding, or features;
- REST page/request/concurrency/byte/resident-record limits;
- frame-slot or byte capacity;
- checkpoint or replay behavior;
- qualification, Day %, ordering, field calculations, or session time;
- provider endpoint, entitlement, or credential handling; or
- public deployment or credentialed live validation.

No predecessor checkout inspection is required or authorized.

The expected current-code seams are deliberately narrow:

- `internal/massive/live_transport.go`: `performHandshake`,
  `awaitHandshakeStatus`, `heartbeatWorker`, terminal cleanup, and bounded
  adapter accounting;
- `internal/operations/live.go`: `runLive`, `openAttempt`, retry scheduling,
  and joined attempt retirement;
- `internal/engine/recovery_control.go`: scheduled-recovery admission and
  attempt-budget ownership;
- `internal/engine/hydration.go`: the existing successful-fence reset boundary,
  which should be reused rather than duplicated;
- `internal/operations/ingress_diagnostic.go` and runtime metrics: immutable
  first cause plus bounded latest attempt outcome; and
- `cmd/scanner/operator.go`: redacted recovery-attempt rendering.

Tests in those package boundaries may change with the production behavior.
Snapshot/API/UI files are outside the slice unless a failing existing proof
exposes an actual contract conflict.

## 4. Required behavior

### HR-01 — one attempt and ordered retirement

At most one stocks WebSocket attempt may be dialing, handshaking, active, or
cleaning up. A greater epoch cannot dial until the prior attempt has:

1. closed queue admission;
2. canceled and joined its reader and heartbeat workers;
3. closed the socket;
4. enqueued and delivered or fenced its terminal marker; and
5. reconciled adapter and queue accounting.

The retry delay begins no earlier than completion of that retirement. Provider
connection-limit rejection must not cause overlapping sockets or bypass this
ordering.

### HR-02 — every failed establishment is paced

The first process-start establishment may begin immediately. After any failed
or lost epoch, every greater-epoch establishment is individually authorized and
delayed. The existing exponential policy remains one second, doubling to a
30-second cap. No branch may open several attempts back-to-back and then apply
one delay only after the batch fails.

For the current five-attempt production budget, the observable recovery delays
before recovery ordinals 1 through 5 are 1s, 2s, 4s, 8s, and 16s. A failed dial,
connected-status phase, authentication phase, aggregate-subscription phase,
or post-ack epoch consumes one attempt under the same policy.

### HR-03 — finite consecutive budget and reset boundary

The attempt counter measures consecutive connection epochs since the most
recent successfully reconciled fresh-bootstrap or gap-recovery ingress fence.
Neither dial success, `connected`, `auth_success`, aggregate acknowledgement,
entry into `hydrating`, suppression, nor admission of `scheduled_recovery`
resets it.

Only the existing successful hydration-complete/fence transition resets the
counter. This ensures that an acknowledged epoch which dies during hydration
does not erase the failures that preceded it.

After the fifth consecutive failed/lost epoch:

- no sixth automatic dial is authorized;
- lifecycle is `suppressed/recovery_exhausted` with
  `same_binding_recovery_allowed`;
- no automatic scheduled-recovery command remains pending;
- liveness and the read-only API remain available;
- readiness remains false and no current ranking is published; and
- shutdown/session-end behavior remains unchanged.

### HR-04 — failed handshake facts are not discarded

Handshake phases continue to classify provider frames strictly and redact
arbitrary provider prose. If the overall handshake fails, every already
classified delivery is admitted to the engine in causal order before the
terminal connection-loss fact.

The retained phase outcome distinguishes at least:

```text
dial_failed
connected_deadline | connected_failed | connected_ambiguous
authentication_deadline | authentication_failed | authentication_ambiguous
aggregate_subscribe_deadline | aggregate_subscribe_failed |
aggregate_subscribe_ambiguous
reader_closed
heartbeat_failure_with_inbound_progress
heartbeat_deadline_without_inbound_progress
heartbeat_transport_failure
```

The implementation must not infer a more precise ping-write-versus-pong-wait
cause when the WebSocket dependency cannot distinguish it. Credential values,
provider messages, URLs containing secrets, and raw payloads remain absent
from errors, metrics, files, API values, and operator output.

### HR-05 — heartbeat failure with inbound progress is nonterminal

Before starting a heartbeat operation, the adapter captures the current
process-local frame-read sequence and start time. If the heartbeat operation
returns an error:

- when a supported raw frame was received after the captured boundary, record
  `heartbeat_failure_with_inbound_progress` as a bounded nonterminal
  diagnostic and keep the aggregate epoch active;
- when no frame was received after the boundary and the operation reached its
  deadline, terminate once as
  `heartbeat_deadline_without_inbound_progress`; and
- when the socket or heartbeat operation fails independently of its deadline,
  terminate once as `heartbeat_transport_failure` or the already-known reader
  terminal, whichever won the existing first-cause race.

Inbound progress proves only that the subscribed aggregate receive path is
alive at that causal boundary. It does not prove that a pending T/Q command
succeeded, change T/Q quarantine, establish a new subscription, advance the
watermark, or make readiness current. The existing 15-second interval and
five-second deadline remain unchanged in this correction.

The nonterminal diagnostic is fixed-cardinality: latest occurrence time,
captured/read frame sequence, occurrences, and consecutive occurrences. A
later successful heartbeat clears only the consecutive count, not lifetime
accounting.

### HR-06 — startup and post-live recovery retain existing semantics

Epoch loss before first live entry continues to execute `LIFE-HYDRATE-07`:
every open request receives one canceled outcome, later old-generation results
are fenced, accepted canonical facts remain, and the next acknowledged epoch
creates exactly one fresh-bootstrap generation for the still-required
coverage.

Epoch loss after a committed watermark continues to execute
`LIFE-RECOVER-01` through `LIFE-RECOVER-06`: currentness closes at the supported
boundary, the next acknowledged epoch creates one exact gap-recovery
generation, its live tail remains admitted, and currentness returns only after
the ingress fence and ordinary evaluation complete.

Retry pacing and diagnostics cannot create a second hydration owner, reuse a
canceled result as coverage, or present an acknowledged-but-unreconciled epoch
as current.

### HR-07 — first cause plus latest recovery outcome

The existing immutable first-ingress incident remains unchanged. In addition,
operations retains one bounded latest recovery-attempt outcome containing:

- epoch and recovery ordinal;
- phase and redacted outcome;
- attempt start, terminal, cleanup-complete, and next-eligible times;
- whether any current-epoch aggregate acknowledgement occurred;
- current consecutive-attempt count and configured budget; and
- the engine lifecycle/suppression consequence.

Each attempt replaces this latest value; no unbounded history or provider prose
is retained. The ordinary terminal must print the latest outcome when recovery
is active or exhausted. This focused correction does not require an API v2
schema change or UI rendering change.

## 5. Consequential boundaries and invalid states

Construction must prevent:

- two live/opening stocks sockets;
- a greater epoch before prior cleanup joins;
- a retry before its engine-owned deadline;
- a scheduled event resetting the attempt budget;
- a sixth automatic attempt after exhaustion;
- two terminal outcomes for one epoch;
- old-generation hydration facts entering a replacement generation;
- discarded authentication or aggregate-subscription failure facts;
- heartbeat diagnostics containing secrets or arbitrary provider prose; and
- heartbeat-only state changing ranking, watermark, or readiness.

Runtime validation must reject or contain stale/foreign/duplicate scheduled
commands, nonmonotonic epochs, phase-invalid statuses, causal-position
regression, accounting contradiction, and a replacement acknowledgement that
does not belong to the active greater epoch.

The dangerous false-success cases are:

1. continuing to label ranking current across actual aggregate loss;
2. treating inbound progress as proof of a write, acknowledgement, or T/Q
   command outcome;
3. resetting the recovery budget on acknowledgement and then losing that epoch
   during hydration;
4. presenting canceled bootstrap work as completed replacement coverage; and
5. claiming recovery is bounded while automatically reopening five-attempt
   batches forever.

## 6. Sequential implementation slices

### HR-S1 — ordered attempts, finite budget, and exact failure evidence

Implement `HR-01` through `HR-04` and `HR-07` without changing heartbeat
terminal policy. The expected boundary is localized to the Massive live
transport/handshake, engine recovery control, operations live supervisor and
diagnostics, scanner operator rendering, and their focused tests.

Primary proof `P-HR-RETRY` uses a fake provider that:

1. accepts an initial epoch;
2. terminates it;
3. temporarily rejects greater connections as connection-limit/authentication
   failures;
4. later accepts one greater epoch; and
5. separately remains failed through exhaustion.

The proof must show causal handshake facts are delivered, prior attempts join
before every dial, each delay is applied individually, no counter reset occurs
before successful fence reconciliation, the successful branch starts exactly
one authorized hydration generation, and the failed branch stops after the
fifth attempt with API/liveness available.

### HR-S2 — inbound-aware heartbeat and complete recovery traversal

Implement `HR-05` and prove `HR-06` through the production composition.

Primary proof `P-HR-HEARTBEAT` blocks/fails the fake socket ping while the
reader continues delivering ordered aggregate frames. It must show the epoch
remains active, frames continue to canonical disposition, hydration is not
canceled, heartbeat accounting records inbound progress, and no readiness or
ranking rule changes. A paired quiet-socket case proves one no-progress
deadline terminates the epoch exactly once.

Primary proof `P-HR-STARTUP` executes:

```text
ack epoch 1
  -> begin fresh-bootstrap generation 1
  -> no-progress heartbeat terminal
  -> cancel every open generation-1 request exactly once
  -> paced epoch 2 acknowledgement
  -> one replacement fresh-bootstrap generation
  -> ingress fence
  -> ordinary live evaluation
```

Primary proof `P-HR-GAP` starts from a committed live publication and executes
the equivalent loss-to-exact-gap-hydration-to-fence-to-current traversal. It
must distinguish old-epoch events, canceled old-generation results, replacement
live-tail aggregates, and the final current publication.

## 7. Verification and acceptance

During correction, run the narrowest named proof first. Each accepted slice
then runs affected package tests and the ordinary repository tier:

```text
go test -short -timeout 2m ./...
```

Because this changes concurrent socket retirement, engine recovery authority,
and invalid external evidence reaching false success, final focused
verification also includes the affected `internal/massive`, `internal/engine`,
`internal/operations`, and `cmd/scanner` race tests under a five-minute command
timeout, `go vet ./...`, and `git diff --check`.

HR-S1 acceptance requires `P-HR-RETRY`, exact attempt/accounting walkthroughs,
and no automatic post-exhaustion dial. HR-S2 acceptance requires
`P-HR-HEARTBEAT`, `P-HR-STARTUP`, and `P-HR-GAP`, including the dangerous
counterexamples above. Final capability acceptance requires one read-only
review focused on socket join/epoch ordering, attempt-budget reset, heartbeat
evidence, hydration-generation fencing, and authority conformance.

Credentialed provider validation is not an implementation or acceptance gate.
An owner-run observation may later confirm provider chronology, but it cannot
replace the deterministic proofs and requires separate exact authorization.

## 8. Completion boundary and correction triggers

This correction is complete when an actively delivering socket cannot be
terminated solely because its concurrent heartbeat timed out, every real
aggregate loss has one paced and finite recovery path, failed handshakes remain
diagnosable without secrets, startup and post-live recovery both traverse back
to current through the existing fence, and persistent failure reaches stable
suppression without another automatic dial.

Reopen the correction if later evidence shows any of the following:

- a provider or dependency behavior makes inbound frame progress insufficient
  to preserve aggregate receive coverage;
- connection cleanup can complete locally while the provider still requires a
  larger minimum retry delay;
- a successful fence fails to reset the intended consecutive budget;
- recovery exhausts before the configured number of individually delayed
  attempts;
- a failed handshake still collapses to an unclassified connection loss; or
- any accepted current publication spans an unsupported aggregate gap.

## 9. Delivery and acceptance ledger

### HR-S1 — accepted 2026-08-17

The live supervisor now retires and joins the prior attempt before requesting
the engine's next one-shot recovery command. Every greater epoch is delayed
individually from cleanup completion by the existing exponential policy. The
engine no longer resets `recoveryAttempts` when admitting
`scheduled_recovery`; only the existing successful aggregate-ingress fence
does so. Exhaustion clears pending authority, enters stable
`suppressed/recovery_exhausted`, keeps the process/API live, and authorizes no
automatic sixth recovery attempt.

The adapter exposes phase-specific redacted terminal outcomes and operations
delivers every classified handshake prefix fact before its terminal, including
when the establishment context expires. The latest recovery-attempt diagnostic
is one replace-in-place value with epoch/ordinal, phase/outcome, start,
terminal, cleanup, next-eligible time, acknowledgement, consecutive count,
budget, and lifecycle consequence. It retains no provider prose, credential,
payload, or secret-bearing URL. The ordinary operator renders this value while
recovery is active or exhausted; exhaustion clears `next_eligible` rather than
implying another dial.

`P-HR-RETRY` is implemented by the `TestPHRRetry*` family. Its production-
composition provider traversal first accepts and hydrates epoch 1, then proves
temporary `auth_failed` handshakes followed by one successful greater epoch
and the existing gap-hydration fence. A separate persistent branch observes
the exact 1s, 2s, 4s, 8s, and 16s cleanup-to-next-accept lower bounds, a maximum
of one provider/adapter attempt, five consecutive failed attempts, stable
exhaustion, and no later dial. The establishment-deadline counterexample proves
`connection_attempt -> connected -> terminal` owner admission even when the
caller deadline wins. The focused command passed in 45.917s; the provider-only
traversal passed in 42.614s.

Affected package verification passed for `internal/massive`,
`internal/engine`, `internal/operations`, and `cmd/scanner` under
`go test -short -timeout 2m` (longest package 60.408s). The same four packages
passed `go test -race -short -timeout 5m` (longest package 63.047s).
`git diff --check` passed. Success-path inspection confirmed the counter reset
and return to `live` occur only after the existing fence. Persistent-failure
inspection confirmed the latest outcome becomes
`suppressed/same_binding_recovery_allowed`, `next_eligible` is empty, the
engine has no issuable scheduled command, and the runtime remains present
until controlled shutdown.

The required focused `gpt-5.6-sol` medium review initially found two issues:
caller-deadline loss of a classified handshake prefix and insufficient end-to-
end retry proof. The correction joined the handshake producer and admitted its
prefix through a non-canceled local engine context, then added the complete
provider traversal above. Focused re-review passed with no unresolved finding.

The coherent HR-S1 milestone is local commit `a916241`. HR-S2 remained valid
after this gate. Credentialed provider chronology, provider retry tolerance
beyond the configured policy, and an operator recovery control remained
unproved/out of scope.

### HR-S2 and final capability — accepted 2026-08-17

The heartbeat worker now captures its process-local frame-read sequence before
each ping. A failed heartbeat with a later supported raw frame records the
fixed-cardinality nonterminal outcome
`heartbeat_failure_with_inbound_progress` and leaves the epoch, hydration,
watermark, readiness, ranking, and T/Q authority unchanged. The diagnostic
retains only latest occurrence/start times, captured/read sequences, lifetime
occurrences, and consecutive occurrences; a later successful ping clears only
the consecutive count. A deadline with no later frame terminates exactly once
as `heartbeat_deadline_without_inbound_progress`; an independent transport
failure uses `heartbeat_transport_failure`, subject to the existing immutable
first-cause arbitration. No provider prose, payload, credential, or URL is
retained.

`P-HR-HEARTBEAT` passed uncached in 0.928s. Its progressing-reader branch
blocks/fails ping while an ordered aggregate reaches the canonical engine,
keeps the epoch and hydration active, records the exact nonterminal diagnostic,
and proves a later successful ping resets only the consecutive count. Its
quiet-socket branch proves one deadline terminal. The existing timing-sensitive
Massive heartbeat regression remains enabled and passed in the full suite.

`P-HR-STARTUP` and `P-HR-GAP` passed together uncached in 3.335s. Startup now
uses a real server-side no-progress heartbeat deadline, preserves that exact
redacted first cause, cancels all four open generation-1 REST requests, and
returns current on epoch 2 only after one replacement fresh-bootstrap
generation and its ingress fence. The gap trace starts from a committed current
publication, admits an epoch-2 aggregate while gap generation 2 is active,
loses that epoch and observes the old request cancellation, admits the epoch-3
live tail while generation 3 is explicitly unreconciled and noncurrent, and
then requires the final publication watermark to equal generation 3's
`SupportedThrough` only after its fence. Both canonical live records survive.
This prevents canceled work from becoming replacement coverage and prevents an
acknowledged replacement epoch from appearing current before reconciliation.

Final primary proof evidence also includes uncached `P-HR-RETRY`: engine passed
in 0.585s and the production operations traversal passed in 42.732s. The
uncached ordinary repository command `go test -count=1 -short -timeout 2m
./...` passed with `internal/operations` longest at 60.535s. The affected
`internal/massive`, `internal/engine`, `internal/operations`, and `cmd/scanner`
packages passed `go test -count=1 -race -short -timeout 5m` with
`internal/operations` longest at 62.163s. `go vet ./...` and `git diff --check`
passed.

The required `gpt-5.6-sol` medium final review first found proof gaps: startup
used a reader close instead of the corrected heartbeat terminal, and the gap
test did not traverse active-generation loss plus replacement-tail fencing
through production composition. Both proofs were corrected as described above.
The same reviewer then ran them ten consecutive times in 30.787s and reported
no unresolved finding or observed nondeterminism. The earlier HR-S1 socket
join, recovery authority, reset/exhaustion, and redaction review also remains
clean.

No credential was accessed, no provider request was made, and the live scanner
was not restarted. Deterministic local WebSocket/REST fixtures prove the
implemented causal boundaries; actual provider ping/pong chronology and
provider tolerance of the configured retry schedule remain unobserved and
require separate exact authorization. Replay, checkpoints, API/UI behavior,
and operator recovery controls were not changed.
