# Private Live Scanner Operational Finalization

Status: complete and accepted locally after final read-only review, 2026-08-10
Scope: private/local daily operation of the existing scanner and dashboard
Primary worktree: `/Users/joshuabelandres/Dev/live-equities-momentum-scanner-live`

## 1. Decision and intended operating model

Finalize the current scanner for a simple daily workflow in which the operator
starts it shortly before 04:00 America/New_York with one command and leaves it
running through the desired session. The launcher starts and supervises the
existing scanner backend and independently runnable dashboard; it does not
merge their ownership or introduce another market-state path.

The supported daily path is:

1. From the repository root, run `./scripts/run-private-scanner` at about 03:55
   America/New_York.
2. The launcher derives the New York trading date, loads the Massive credential
   without displaying it, uses the persistent reference and checkpoint
   directories, and starts the scanner.
3. Once the scanner HTTP service is live, the launcher starts the dashboard and
   prints both local URLs.
4. Before 04:00, ranking readiness is expected to remain false. At 04:00 the
   existing runtime begins the session, continuously incorporates aggregates,
   and writes coherent checkpoints on its existing cadence.
5. A same-day restart uses the same command and directories. The runtime—not
   the launcher—selects the latest valid compatible checkpoint, falls back to
   the previous checkpoint if necessary, catches up by REST hydration, fences
   the buffered live tail, and then publishes live state.
6. Ctrl-C stops both processes cleanly. An unexpected exit of either process
   stops the other and causes the launcher to exit nonzero.

Starting after 04:00 remains a supported recovery attempt, not the recommended
daily cold-start path. A compatible same-day checkpoint can make a restart
fast. With no usable checkpoint, the scanner may require full-session REST
hydration while live aggregates continue to arrive. The current final code has
proved this at the full 1x workload, but it has not proved sustained full 2x
capacity. The launcher and runbook must warn about that limitation; they must
not claim that any post-04:00 cold start will complete within a fixed time.

This task does **not** replace correct full-universe qualification with a top-N
shortcut. Aggregate rankability and qualification retain their current Phase 1
semantics. Starting before 04:00 avoids a large catch-up backlog because the
engine advances incrementally; it does not create different scanner results.

## 2. Controlling authority

Before implementation, read:

- `AGENTS.md`;
- `README.md`;
- `docs/specification-map.md`;
- `docs/v1-release-program.md`;
- the Phase 1 product and architecture documents routed by the specification
  map for runtime lifecycle, checkpoints, aggregate hydration, ranking, and
  dashboard independence;
- `docs/live-fence-finalization-cached-hydration-correction.md`; and
- the current C8 parent contract and delivery ledger entries affected by this
  correction.

This specification is a lower-level operational finalization of the existing
V1 behavior. It does not authorize changed ranking formulas, qualification
rules, session boundaries, checkpoint semantics, state ownership, or provider
semantics. If any instruction here conflicts with controlling Phase 1
authority, revise this document through the V1 correction loop and preserve the
fixed product meaning.

## 3. Current evidence that must be preserved

Do not restart the capacity investigation from scratch. Begin by validating the
current source and the following handoff evidence:

| Evidence | Current result | Meaning for this task |
| --- | --- | --- |
| Full cached-hydration 1x preserved artifact | Passed: 7,581,690 REST rows; 5,439 value symbols and 63 successful-empty symbols; 30,483 frames read/admitted/dispositioned/fenced; zero rejection; maximum queue depth 247; fence 526.912 ms; tail drain 364.965 ms; 20 qualified-current ranked rows; ready/accounting true | The existing design can complete the required full late-start workload at the accepted 1x input rate. Preserve `var/cached-hydration-fence/final-1x/run.log`. |
| Full cached-hydration sustained 2x preserved artifact | Failed honestly: 56,757 frames read; 56,756 admitted, 56,244 dispositioned, and 512 fenced; one capacity rejection after 7,078,180 hydration rows had been consumed | Do not present sustained full 2x as supported. Preserve `var/cached-hydration-fence/final-2x/run.log` and do not repeatedly rerun this expensive known-failing trial. |
| Focused full-retention 2x burst handoff | Reported as passing twice, including a 3,072,000-row, 23,193-frame trial with maximum queue depth 75 and 3.46 ms drain | The live-tail path reportedly absorbs the tested bounded 2x burst. Reconcile this claim with a saved test artifact or rerun only the bounded focused proof; it is not equivalent to sustained full 2x capacity. |
| Current-source full cached-hydration 1x confirmation | Passed: the same 7,581,690 REST rows and 5,439 value/63 successful-empty terminals; 28,377 frames read/admitted/dispositioned/fenced; zero rejection; queue high-water 293; fence 490.875 ms; tail drain 650.808 ms; 20 ranked rows; ready/accounting true | Confirms production engine changes after the original artifact did not regress the accepted 1x boundary. Preserve separately at `var/cached-hydration-fence/final-source-1x/run.log`; do not combine its pacing/timing metrics with the earlier trial. |
| Current-source focused full-retention 2x burst | Passed: 7,745,536-row preload plus all 12,000 measured chunks/3,072,000 rows; 23,448 frames read/dispositioned; zero rejection; queue high-water 38; drain 2.194 ms; accounting true | Reconciles the unsaved handoff with a fresh bounded proof at the same semantic boundary. Preserve `var/cached-hydration-fence/final-source-bounded-2x/run.log`; it does not establish sustained full 2x. |
| Provider/live-market execution | Not run | This task does not authorize credentials access or provider requests during implementation or verification. |

An earlier task handoff quoted different 1x and 2x frame/timing counts while
pointing to the two preserved `final-*` logs above. Treat the logs as the
current evidence of record. Before final acceptance, determine whether the
different handoff counts came from later unsaved trials. Either save and cite
the later complete output or correct the handoff/ledger; never combine metrics
from different trials into one result.

Before treating the code as frozen, confirm that no transient experiment made
after the passing full 1x trial remains in production source. Preserve unrelated
user changes in the dirty worktree. Record the exact commit/diff state from
which the final evidence is claimed.

## 4. Required outcome

Completion requires all of the following:

1. One stable daily operator command: `./scripts/run-private-scanner`.
2. Deterministic daily defaults suitable for a private macOS workstation.
3. Correct startup, supervision, shutdown, and failure containment for the two
   existing processes.
4. Secure credential acquisition with no credential output or persistence.
5. Explicit pre-04:00, post-04:00, restart, and after-session behavior.
6. A concise runbook describing startup, status interpretation, recovery, and
   the proven capacity boundary.
7. Final verification of the already-correct hydration/fence implementation,
   without an open-ended performance-tuning loop.
8. One final read-only review and a clean conformance walkthrough.

This task is complete when a new operator can use the scanner on an ordinary
trading day with the one command and can distinguish “process is alive,”
“scanner is ready,” “hydrating,” and “honestly suppressed” without reading the
source code.

## 5. Non-scope and prohibited redesigns

Do not implement any of the following as part of this task:

- top-N-only ranking, qualification, hydration, or feature computation;
- changed liquidity, day-percent, ranking, or filter formulas;
- a second state engine, second canonical symbol representation, or launcher-
  owned readiness calculation;
- queue-size, timeout, or capacity changes made only to chase the known full 2x
  failure;
- a database, daemon, `launchd` installation, container deployment, public
  deployment, service split, or process manager framework;
- automatic order entry, alerts, or trade/quote dependence for aggregate
  ranking;
- new provider requests, market-hours validation, credential inspection, or
  Keychain access during tests;
- deletion of checkpoints after an error;
- broad retry loops or a launcher that silently restarts a failing scanner;
- UI redesign beyond the documentation or minimal status text required to make
  existing lifecycle state intelligible.

An automatic scheduled start may be considered later. V1 daily operation is an
explicit one-command foreground launch so failures remain visible to the
operator.

## 6. Existing behavior to reuse

The implementation must wrap, not duplicate, the current behavior:

- `cmd/scanner` remains the scanner entry point and sole owner of the live
  runtime.
- `cmd/dashboard` remains independently runnable and consumes the scanner API.
- `internal/operations/live.go` remains responsible for installing the latest
  or previous valid checkpoint, choosing fresh bootstrap versus checkpoint
  catch-up, hydrating, and transitioning through the live fence.
- `internal/engine/checkpoint_cadence.go` remains responsible for coherent
  session-aligned checkpoint submission.
- `/livez` reports that the scanner HTTP process is alive.
- `/readyz` reports authoritative scanner readiness. The launcher may display
  this result but may not synthesize or override it.
- `/api/v1/snapshot` remains the dashboard data boundary.

The implementation agent must inspect the exact current flags and API behavior
before writing the launcher. If names or defaults differ from this document,
record and correct this lower-level specification rather than adding a
translation layer that obscures the real interface.

## 7. One-command launcher contract

### 7.1 Public interface

The required interface is:

```text
./scripts/run-private-scanner [--trading-date YYYY-MM-DD] [--hydration-workers 1|2|4|8] [--open]
```

No argument is required for ordinary daily use. Unknown arguments, repeated
single-value arguments, invalid dates, unsupported worker counts, or positional
arguments fail before any child starts. Hydration defaults to eight workers;
the explicit override changes only the scanner's existing bounded C6 worker
setting. `--help` prints the supported interface without reading a credential
or starting a process.

The launcher must resolve the repository root from its own location rather than
from the current directory. The short relative command above is the ordinary
repository-root workflow; invoking the script by its absolute path must also
work from another directory.

The implementation mechanism is agent-discretion within this behavioral
contract. Prefer the smallest auditable shell wrapper if it can prove correct
signal forwarding and child reaping on the supported macOS environment. Use a
small Go supervisor only if the shell implementation cannot reliably meet the
process-lifecycle tests. Do not refactor the scanner and dashboard into one
process merely to simplify launching.

### 7.2 Fixed daily defaults

The launcher supplies these settings explicitly rather than relying on
incidental CLI defaults:

| Setting | Daily value |
| --- | --- |
| Scanner run mode | `live` |
| Trading date | Current date in `America/New_York`, unless explicitly overridden |
| REST hydration workers | `2` |
| Reference directory | `<repo>/var/reference` |
| Checkpoint directory | `<repo>/var/checkpoints` |
| Scanner API address | `127.0.0.1:8080` |
| Dashboard address | `127.0.0.1:4173` |
| Dashboard scanner origin | `http://127.0.0.1:8080` |
| Scanner allowed dashboard origin | `http://127.0.0.1:4173` |

All other provider pagination, timeout, REST, WebSocket, and engine values stay
at their production CLI defaults unless the current executable contract
already requires an explicit value. The daily launcher is deliberately not a
general tuning interface.

The launcher creates required local runtime directories with private
permissions where practical. It does not alter existing reference files or
checkpoints beyond what the scanner itself is authorized to write.

### 7.3 Trading date and wall-clock behavior

All time-of-day decisions use `America/New_York`, independent of the Mac's
local timezone.

- Before 04:00: start normally and print that readiness is expected only when
  the authoritative runtime reaches the session and completes its required
  lifecycle work.
- From 04:00 through the live session: start normally but print a prominent
  warning before credential acquisition and child startup. State that a usable
  checkpoint can make restart fast, while a checkpoint-less cold start must
  hydrate elapsed aggregates and is proved only to the accepted full 1x
  workload.
- After the controlling session end: fail by default with an explanation that
  this command is for the live session. A future explicit historical or replay
  workflow is outside this launcher.
- On weekends, holidays, or invalid schedule dates: let the scanner's
  authoritative schedule validation reject the date. The launcher must surface
  the error and stop the dashboard; it must not invent a trading calendar.

`--trading-date` exists for an explicit operator correction and deterministic
testing. It does not change New York session semantics or authorize replaying a
past date through the live provider path.

### 7.4 Credential handling

Credential precedence is:

1. an already exported `MASSIVE_API_KEY`; otherwise
2. the existing private macOS Keychain generic-password item with account
   `joshuabelandres` and service `momentum-scanner-massive-api`.

If neither is available, fail before starting either child and print setup
instructions that do not contain the credential. Do not log the credential,
include it in command arguments, persist it to a file, enable shell tracing
around it, or expose it in error output. Pass it only through the scanner child
environment. The dashboard must not receive it.

Tests use an injected fake credential and fake child commands or processes.
They must not execute the real Keychain lookup. The production test seam must
be narrow, clearly named as test-only, and unable to change market semantics.

### 7.5 Preflight

Before reading a credential or starting a child, the launcher must:

1. resolve and validate the repository root and required command packages;
2. validate arguments and the New York trading date;
3. verify required build/runtime tools are available;
4. verify that `127.0.0.1:8080` and `127.0.0.1:4173` are available; and
5. verify that required directories are accessible.

If a port is occupied, report the exact port and fail. Never kill an existing
process automatically.

### 7.6 Build and startup sequence

The one command may build the two current commands as an internal startup step.
If it does, build into a task-specific private temporary or ignored runtime
directory and execute the resulting binaries directly. Do not use a `go run`
process tree if it prevents reliable signal forwarding, exit-status
attribution, or child cleanup.

Startup order is:

1. start the scanner with the fixed daily settings;
2. wait up to 5 minutes for `/livez` to succeed, while also watching for early
   scanner exit; the scanner resolves the reference binding before opening the
   API, so this bound must not assume an instant cached startup;
3. if `/livez` does not succeed, stop and report a scanner startup failure;
4. start the dashboard;
5. confirm that the dashboard HTTP listener is live within 30 seconds;
6. print the dashboard URL, scanner snapshot URL, `/livez`, and `/readyz` URLs;
7. continue to report the transition of `/readyz` without treating pre-session
   or in-progress hydration as a launcher failure.

The launcher may optionally open the dashboard only when `--open` is supplied.
Opening a browser is never required for scanner correctness and its failure
must not stop healthy services.

### 7.7 Process supervision and shutdown

The launcher stays in the foreground and remains the parent/supervisor for the
session.

- Ctrl-C and SIGTERM are forwarded to both children.
- The scanner receives a bounded graceful-shutdown interval sufficient to
  execute its existing shutdown path. The dashboard is then stopped and both
  children are reaped.
- If either child exits unexpectedly, stop the other child, reap both, identify
  which process failed, and return nonzero.
- Never use broad process-name matching, unresolved globs, or `kill -9` as the
  ordinary shutdown path.
- After every tested success or failure path, no scanner, dashboard, wrapper,
  or temporary build process may remain orphaned.
- The launcher does not automatically restart either child. A restart is an
  explicit operator invocation of the same command, allowing the scanner's
  existing checkpoint recovery to run.

Persistent log capture is optional for this slice. If added, it must use a
private ignored directory, must not contain credentials, and must have a stated
retention or size bound. Terminal output alone is acceptable for V1 because the
scanner already exposes structured operational state.

## 8. Operator runbook requirements

Update `README.md` with a short link and put the detailed runbook in either this
document or one clearly linked operational document. It must include:

1. prerequisites: supported macOS environment, Go version, reference data,
   writable checkpoint directory, local ports, and credential setup;
2. the single ordinary command and the recommended approximately 03:55 New
   York start time;
3. exact local URLs and the optional `--open` behavior;
4. the distinction between:
   - `/livez` successful: the scanner HTTP process is running;
   - `/readyz` successful: the scanner itself declares the published snapshot
     ready;
   - hydrating/fencing: process healthy but ranking publication not yet ready;
   - honestly suppressed: an observed capacity or integrity condition prevents
     a false-ready snapshot;
5. why a pre-04:00 start is preferred: no large elapsed-session backlog, not a
   different ranking algorithm;
6. late-start and same-day restart behavior, including latest/previous
   checkpoint recovery and REST catch-up;
7. the exact proven boundary: full 1x passed, bounded full-retention 2x burst
   passed, sustained full 2x failed honestly and is unsupported;
8. shutdown with Ctrl-C and the fact that checkpoints are periodic coherent
   committed states, not a promise of a forced final checkpoint at shutdown;
9. failure response: preserve checkpoints and output, do not delete state or
   repeatedly restart, and inspect the first scanner error/readiness reason;
10. explicit statement that live provider validation requires separate
    owner authorization for the exact date and task.

Do not document “start exactly at 04:00” as a correctness requirement. The
recommendation is to start several minutes earlier so builds and preflight
finish before the session begins.

## 9. Delivery slices

### Slice 1: freeze the corrected runtime and implement the launcher

Allowed work:

- inspect and reconcile the current dirty diff and correction ledger;
- remove only proven transient experimental production changes;
- add the launcher and its focused tests;
- make the smallest correction necessary for a failing launcher boundary;
- update the correction evidence record.

Do not change engine, ranking, hydration, checkpoint, provider, or API behavior
unless a deterministic proof exposes a concrete correctness defect. If that
happens, mark the affected earlier claim reopened, name the invalidated claim,
preserve unaffected evidence, apply the narrowest correction, and rerun the
distinguishing proof. Do not resume general optimization.

Slice 1 is accepted when the launcher contract is proved with fake processes,
the runtime source is reconciled with the passing evidence, and no child or
credential boundary can falsely appear successful.

### Slice 2: operational documentation and final acceptance

Allowed work:

- complete the runbook and README entry;
- run the proportionate verification matrix below;
- record remaining limitations and deferred live validation;
- obtain the required final read-only review;
- perform the final conformance walkthrough and exact-path commit if the
  worktree permits it without touching unrelated user changes.

Slice 2 is accepted when the documented one-command workflow matches the tested
behavior and the final review has no unresolved fixed-authority or consequential
trust-boundary finding.

## 10. Primary proofs and verification budget

### 10.1 Launcher proofs

Use deterministic fake child programs or an equivalent injected command seam.
No test may contact Massive, read the real Keychain entry, or depend on the
market being open.

The focused launcher suite must prove:

1. no-argument invocation derives the New York date and supplies every fixed
   scanner/dashboard setting exactly once;
2. `--trading-date` overrides only the date, and `--hydration-workers` accepts
   exactly 1, 2, 4, or 8 with default 8; malformed/repeated values fail before
   preflight;
3. the post-04:00 warning appears before child startup and accurately states
   the capacity limitation;
4. help and preflight failures do not read credentials or start children;
5. the environment credential takes precedence and reaches only the scanner;
6. a simulated Keychain credential is never printed or written;
7. scanner starts before dashboard and dashboard waits for scanner `/livez`;
8. `/readyz` remaining false before session or during hydration does not kill
   healthy children or produce a false-ready message;
9. scanner early exit, scanner live timeout, dashboard startup failure, and
   either child's later failure each stop and reap the other process and return
   nonzero;
10. Ctrl-C and SIGTERM leave no child or wrapper process running;
11. occupied ports fail without killing the occupying process; and
12. `--open` failure is nonfatal after both services are healthy.

Every wait, poll, and child process in tests must be bounded. The focused suite
should complete in seconds, not minutes.

### 10.2 Runtime/capacity proofs

Perform only the remaining work necessary to freeze the current correction:

1. validate and preserve the existing passing final-code full 1x artifact;
2. run one second final-code full cached-hydration 1x trial if the current
   correction spec requires two final trials and that trial has not already
   completed;
3. run or reuse the required focused full-retention 2x burst proof on the final
   source;
4. do not rerun sustained full 2x unless a later source change directly affects
   the previously failing admission/fence boundary and the correction spec
   requires the distinguishing trial;
5. record sustained full 2x as an honest unsupported boundary, not an open
   blocker to private 1x daily operation.

Every performance command has an explicit timeout no greater than the limit in
the V1 release program. Validate fixture bytes, population, corrections,
interval, frames, rows, and planned work before timing. Do not launch multiple
full-capacity trials concurrently.

### 10.3 Repository verification

After focused tests pass, run the narrowest affected race tests, then:

```text
go test -short -timeout 2m ./...
go vet ./...
```

Run the existing dashboard/UI test and build commands required by the current
component contract. Run `git diff --check` and inspect the complete diff for
credentials, generated binaries, temporary logs, unrelated edits, and
accidental semantic changes.

Report what each tier proves and what it does not prove. Passing deterministic
tests does not claim a successful provider session.

## 11. Review requirements

One final read-only review is required because this task closes a component
correction and introduces a credential/process-supervision trust boundary. Use
the repository's preferred independent-review model and reasoning level.

The reviewer must focus on:

- whether the launcher can leave orphan processes or report false success;
- whether credentials can reach output, disk, arguments, or the dashboard;
- whether the launcher duplicates readiness, checkpoint, schedule, or state
  ownership;
- whether the daily default invokes eight workers, every explicit supported
  worker value is passed exactly once, invalid values fail before credential
  access, and the persistent checkpoint location remains fixed;
- whether the late-start wording overstates the 1x and 2x evidence;
- whether unrelated dirty-worktree changes were altered; and
- whether final documentation matches executable behavior.

The reviewer is read-only and does not expand the task into new scanner
features or another capacity-optimization cycle.

## 12. Completion record

The implementing agent's final handoff must state, concretely:

- the exact one-command workflow now available;
- the actual scanner and dashboard arguments used by that workflow;
- how credential isolation, startup ordering, failure containment, and shutdown
  were proved;
- the final full 1x results and the bounded 2x burst results;
- that sustained full 2x remains unsupported, with the recorded honest failure;
- the checkpoint/restart behavior and its proof;
- all verification results and explicit timeouts;
- the final review result;
- whether any provider request or real credential access occurred (expected:
  no); and
- the exact remaining market-hours validation that would require separate
  owner authorization.

Do not claim “private production ready” if the one-command lifecycle tests,
ordinary suite, required capacity acceptance, documentation, or final review
remain incomplete. Do not withhold private 1x operational acceptance solely
because sustained full 2x is unsupported; that boundary is documented and
honestly suppressed by the runtime.

## 13. Completion record

The accepted daily workflow is now:

```text
./scripts/run-private-scanner
```

The absolute script path works from another directory. `--trading-date
YYYY-MM-DD`, `--hydration-workers 1|2|4|8`, and `--open` may be combined in any
order; help and invalid arguments return before any build, credential lookup,
directory mutation, or child start. Hydration defaults to eight workers. The
shell bootstrap privately builds the Go supervisor under
`var/run-private-scanner/bin`, forwards INT/TERM to a build in progress, and
waits/reaps it before executing the supervisor.

The scanner receives exactly `--run-mode live`, the derived or overridden New
York `--trading-date`, `--hydration-workers 8` by default or the exact explicit
supported override, persistent absolute
`--reference-dir` and `--checkpoint-dir` paths, `--api-address
127.0.0.1:8080`, and `--allow-origin http://127.0.0.1:4173`. The dashboard
receives exactly `--address 127.0.0.1:4173`, `--api-origin
http://127.0.0.1:8080`, and the absolute repository `--assets` path.

The launcher checks both ports and required directories before credential
acquisition. An exported `MASSIVE_API_KEY` takes precedence over the named
Keychain item. The value is neither printed, persisted, nor placed in an
argument; it is removed from Go build, dashboard, and browser-opener
environments and added only to the scanner child environment. The scanner must
make `/livez` successful before the dashboard starts. A false `/readyz` remains
an engine-owned nonready state rather than a launcher failure or false-ready
claim. Exact SIGINT/SIGTERM forwarding, scanner-then-dashboard graceful stop,
bounded forced containment, unexpected child failure, startup failure, port
conflict, and browser-open failure were proved with bounded fake/real process
tests; no wrapper or child remains unjoined on a passing path.

Same-day checkpoint recovery remains owned by `internal/operations/live.go`:
the latest valid compatible checkpoint is preferred, the previous valid
checkpoint is fallback evidence, and the existing REST catch-up plus exact
live ingress fence controls readiness. The launcher neither selects a
checkpoint nor calculates readiness. Ctrl-C does not promise a forced final
checkpoint.

Final capacity evidence is deliberately separated by trial:

- the preserved full 1x evidence-of-record passed 7,581,690 rows, 5,439 value
  and 63 successful-empty terminals, 30,483 fully reconciled frames, zero
  rejection, queue high-water 247, 526.912 ms fence, and 364.965 ms drain;
- the fresh current-source full 1x confirmation passed the same row/terminal
  oracles with 28,377 fully reconciled frames, zero rejection, queue high-water
  293, 490.875 ms fence, 650.808 ms drain, `live`/`qualified_current`, 20 rows,
  and ready/accounting true;
- the strengthened focused full-retention 2x burst completed the 7,745,536-row
  preload plus all 12,000 measured chunks/3,072,000 rows and 23,448 frames with
  zero rejection, queue high-water 38, 2.194 ms drain, and reconciled
  accounting; and
- sustained full 2x remains unsupported: the preserved trial read 56,757
  frames, admitted 56,756, dispositioned 56,244, fenced 512, and rejected one
  capacity admission after 7,078,180 hydration rows. It was not rerun.

Final verification passed:

- focused launcher tests under a one-minute bound, including public-shell
  parsing and bootstrap signal containment;
- the public-wrapper race proof ten consecutive times in 13.621 seconds and
  the complete launcher race suite in 2.654 seconds;
- focused final-source engine/operations race proofs under a five-minute
  command bound;
- `go test -count=1 -short -timeout 2m ./...`;
- `go vet ./...`;
- `node --test ui/model.test.mjs ui/visual.test.mjs` (22/22); and
- `sh -n scripts/run-private-scanner` plus `git diff --check`.

The required `gpt-5.6-sol` medium-reasoning read-only review initially found
bootstrap build containment/argument-ordering defects and stale ledger/signal
wording. Those were corrected and publicly tested. A focused re-review then
found one orphan-prone test failure cleanup; bounded wait ownership and
explicit fake-process cleanup corrected it. The final focused re-review found
no remaining P1/P2 issue.

The evidence source is commit
`06ff92dd8fd94a8382d3316e974149cb3d4e6526` plus the preserved dirty runtime
correction diff. The production runtime diff over `cmd/scanner`,
`internal/engine`, `internal/massive`, and `internal/operations` has SHA-256
`a6e2ead0cac1edb36a65956d190340f477b9510a4d9c5f4e09e07b85bdd73c5e`.
The launcher/script/test manifest has SHA-256
`0b236bff7536a45dd1ff9a783f99cec82df201f4e22f8b01e83c162451746afc`.
The distinct evidence-log SHA-256 values are `fc02b4eecdf2b85329ea46b0f063097d0967cc162b5aca174722802b075b4e6b`
for `final-1x`, `df88b5f918de07a7d7d0f3d540eb2e9c627f865e47262967ab8269143921b246`
for `final-source-1x`,
`7ea3b962fc18c446e1505b084db5086eb8d91dfd8eeadbe3950dbccaa6a9b152`
for `final-source-bounded-2x`, and
`fdcc6d62ec477644a27e4c62f1d1a7884fa4582102ac002f9ee53c7ee2ffb882`
for `final-2x`.
No commit was made because the worktree began with interleaved dirty runtime
work and an ownership-unknown untracked `operations.test`; exact-path staging
would not establish ownership of those pre-existing changes.

No real Keychain credential was read and no provider request was made. The only
remaining market-hours item is one separately owner-authorized, exact-date
launch with the selected worker count that observes provider binding, complete
hydration, ingress-fence reconciliation, ready/current API/UI output, queue
high-water, post-fence heap, and independent T/Q status. It is external
validation, not a prerequisite for this accepted private/local 1x workflow.
