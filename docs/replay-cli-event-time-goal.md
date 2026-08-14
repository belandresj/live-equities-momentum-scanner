# Goal handoff: implement CLI event-time replay

Use this exact slash command in Codex:

```text
/goal Implement and finally accept the CLI-only, fully unpaced event-time replay correction. Follow docs/replay-cli-event-time-goal.md and docs/replay-cli-event-time-correction.md as the current executable instructions; do not stop at diagnosis, a partial patch, a commit, or a correctable failure.
```

The objective is a finite command that feeds the retained day's normalized
one-second aggregates through the sole production `ScannerStateEngine` as fast
as the host can process them. It reconstructs state from 04:00 to a requested
New York start without emitting scanner rows, emits immutable CLI boundary
snapshots at start and after every selected market second through the requested
end, validates exact terminal accounting, joins, and exits. A five-minute
window therefore emits 301 snapshots (`start..end`, inclusive) followed by one
result. There is no replay UI, HTTP server, 1x pacing,
live hydration, checkpoint, provider call, credential read, T/Q replay, second
clock, second evaluator, or second market-state path.

First read `AGENTS.md`, `README.md`, `docs/specification-map.md`,
`docs/v1-release-program.md`, `docs/implementation-process.md`, the correction
spec above, the complete C12 parent, the complete routed C4 contract, and the
replay sections of the approved product/architecture authorities. Treat the
correction spec as the owner's current C12 direction and reconcile lower-level
C12 text/ledger/map before implementation.

Git safety is the first action after read-only inspection. The current
`codex/replay-ui-mvp` worktree has mixed uncommitted live and replay changes.
Do not commit, discard, or carry them wholesale. Create an isolated worktree
and branch `codex/replay-cli-event-time` from `28281df` unless inspection finds
a later clean replay-only milestone. Read the original replay diff as evidence
and port only the replay changes justified by the correction spec. Preserve all
unrelated user/live work. Because the correction documents themselves are
currently uncommitted, copy their exact contents and the parent/map routing
edits into the isolated worktree as the first correction-plan commit; apply
only the replay-specific map hunk, not the unrelated live-scanner map change.
Do not push, rebase, amend, rewrite history, delete branches, or use
destructive resets.

Keep one write-capable slice at a time. First reproduce the compact path and
record the exact accepted claims reopened. Then implement the smallest
corrections in this order: complete-final-bar group-boundary evaluation with
exact baseline equivalence; one opaque pre-engine validation with no redundant
construction scan; finite unpaced CLI configuration/output/termination. Remove
or bypass the old C12 1x scheduler, replay HTTP retention, and dashboard path;
do not generalize them. Every completed second from 04:00 through the end must
still invoke the ordinary evaluator/publication path; do not port the dirty
patch's observation-start evaluator/qualification/feature branch. Preserve
C4's source, simulated clock, record/group order, same-open prefix/suffix
validation, requested-end fact, cancellation, and accounting. Preserve
production `D=4s` and all ordinary market semantics.

Implement `P-C12R-CONFIG`, `P-C12R-EQUIV`, `P-C12R-GROUP`,
`P-C12R-OUTPUT`, and `P-C12R-TRUST` with compact deterministic fixtures. The
CLI envelope is `scanner.replay.cli.v1`, bounded to 1 MiB per NDJSON record;
snapshots contain engine-owned publication/population facts, while only the
terminal result contains exact C4 source accounting. Run narrow proofs while
correcting, then `go test -short -timeout 2m ./...`, focused changed-package
race under five minutes, `go vet ./...`, and `git diff --check`. Distinguish
pre-existing failures in the original dirty worktree from failures on the
isolated branch.

Use the retained 2026-08-07 artifact and exact caches for `P-C12R-B4` without
provider requests or credential access. Locate it only from existing private
configuration or its approved manifest; never print or commit its path or
artifact identity. Verify 2,584,011,150 bytes, 5,691 symbols, 7,671,171 records,
complete 04:00-20:00 session, then run `[09:30:00,09:35:00)` entirely unpaced
with a hard 15-minute command timeout. Rows are intentional CLI output, but the
acceptance harness must consume them in an ephemeral private `0600` sink and
must not echo them into test/task logs, commit them, or retain the sink. If the
run misses, preserve segment timings, correct the lowest proven hotspot, and
rerun only the distinguishing proof before one final acceptance attempt.

Do not invoke a reviewer solely because the persisted/order/batching claims are
reopened or C12 is complete; use the allocated deterministic proofs and
conformance walkthroughs for reacceptance. Reopen and reaccept the exact C2/C3
parent-ledger claims touched by group evaluation and the exact C4 parent-ledger
claim touched by validation; preserve unaffected evidence. Update the C12 sole
ledger, specification map, README command, and operator behavior. Make coherent
local commits only while workers/reviewers are quiescent. Finish with a clean
isolated worktree and report the exact CLI, retained-day segment timings,
output/terminal accounting, limitations, branch, and commit hashes. Do not
claim live chronology, provider SLA, predictive edge, or executable expectancy.
