# Follow-on goal: implement historical replay product mode

Complete the remaining private/local V1 RC work and then implement, prove,
review, and finally accept the historical replay product mode through C12 in
one continuous sequential goal. Start from the approval milestone containing
this prompt on `codex/c12-historical-replay-contract`; if branch creation is in
scope, use `codex/c12-implementation`. Do not push, rebase, amend, rewrite
history, or modify unrelated user work.

First read `AGENTS.md`, `README.md`, `docs/specification-map.md`,
`docs/v1-release-program.md`, `docs/implementation-process.md`, this prompt,
the complete C11 contract, the complete routed C4 contract, and the complete
C12 contract. The focused contracts and Phase 1 authorities control if this
handoff summary is less precise.

Keep one active write-capable slice and one authoritative implementation path.
Independent reviewers are read-only. Do not stop after a progress report,
local commit, correctable test failure, benchmark miss, review finding, or
in-scope contract correction. Preserve the evidence, revise the lowest
incorrect artifact, run the narrowest distinguishing proof, update the sole
component ledger, and continue. Stop only for an irreconcilable fixed-authority
conflict, a required new source/whitelist or product rule, provider/credential
access, destructive/external action, public deployment, or a hard tool/system
impossibility after exhausting the approved alternatives.

Execute these gates in order:

1. Finish C11 and the private/local V1 RC. Use the installed
   `browser:control-in-app-browser` skill to run the pending Chrome-desktop
   `P-C11-VISUAL` proof against deterministic loopback fixtures at the contract
   viewport. Correct any production-path defect, rerun the narrow proof, finish
   C11's required read-only final review, then complete the integrated V1 RC
   proof/review and update the C11 ledger/specification map. Do not use provider
   credentials or market-hours data.
2. Implement C4-S6 exactly from the C4 parent, replay-source detail, and
   deterministic-core assignment. Mark Component 4 reopened only for
   `C4-BOUNDED-CANCEL-01`, preserving accepted S1-S5 evidence. Add the bounded
   untrusted candidate-header probe, context-aware first-pass and terminal/
   suffix validation, and idempotent source-owned `Cancel(ctx)`. Run
   `P-C4-BOUNDED-CANCEL`, focused ordinary/race verification, the required
   `gpt-5.6-sol` medium persisted-source/lifecycle review, focused re-review
   after corrections, and conformance. Return C4 to accepted before C12-S1.
   Do not inspect Version 2 or the retained B4 artifact in this slice.
3. Implement C12-S1 exactly from its ten-point assignment. Add the mutually
   exclusive scanner replay CLI, one focused replay-mode coordinator, C8
   replay-only runtime/capture support, and C10's additive atomic `replay`
   mapping. Preserve the single C4 clock/source, engine state/evaluator/
   publication ownership, production `D=4s`, unpaced warm-up through `O0`, one
   monotonic cumulative 1x schedule over `(O0,O1]`, honest lag, replay-specific
   T/Q unavailability, failure containment, and immutable retained success.
   C4 packages are closed in S1. Run `P-C12-WINDOW`,
   `P-C12-DETERMINISM`, `P-C12-RUNTIME`, and `P-C12-CONTAINMENT`, ordinary
   verification, focused race, vet, conformance, and the required read-only
   review. Correct findings and accept S1 before S2. Do not read B4 rows in S1.
4. Implement C12-S2 exactly from its ten-point assignment. Extend the existing
   independent dashboard with replay-authoritative/nonlive warming, observing,
   behind-schedule, finalizing, retained-success, canceled/suppressed, and
   disconnected/frozen states without browser-owned market calculations or
   sorting. Run the pure model proof and Chrome replay-state/UI-restart proof.
   The owner authorizes the retained validated 2026-08-07 B4 artifact to be
   opened in place only after S1 acceptance for the exact
   `[09:30:00,09:35:00)` New York acceptance. Do not copy it, commit it, record
   its path/identity/rows, read credentials, or make provider requests. Validate
   only the private manifest before timing: 2,584,011,150 bytes, 5,691 symbols,
   7,671,171 records, complete mode, full 04:00-20:00 session. Bound the full
   validation/warm-up/five-minute 1x/suffix/retention/restart/shutdown proof to
   15 minutes. Record exact accounting, lag, changing publications/rows,
   terminal disposition, retained immutability, UI-only restart, and clean
   shutdown. Correct findings and accept S2.
5. Run the required final read-only C12 review and clean conformance
   walkthrough. Correct every actionable finding with the narrowest proof,
   finalize the C12 ledger/specification map/README operator workflow, and
   locally commit final C12 acceptance. Do not claim provider SLA, live
   chronology, correction-arrival fidelity, public deployment, predictive
   edge, or executable expectancy.

Ordinary verification is `go test -short -timeout 2m ./...`. Every focused
race, Chrome, replay, capacity, and B4 command has the explicit timeout in its
own contract; no ordinary test performs wall-session work. Stage exact paths
and make local commits only when all workers and reviewers are quiescent. Make
coherent local commits for C11/V1 RC closure, accepted C4-S6, accepted C12-S1,
accepted C12-S2, and final C12 acceptance. Finish with a clean worktree and a
trader-facing report that states the exact replay workflow, observation and
terminal behavior, measured B4 results, failure containment, proof limits,
review results, and commit hashes.
