# Repository guidance

## Repository goal

This repository implements a private/local live equities momentum scanner. The
supported path is a fresh-start live scanner that hydrates historical aggregate
coverage, consumes live aggregate and selected-row trade/quote data, maintains
one canonical state, publishes immutable snapshots through a local API, and
renders the dashboard independently.

The current product scope is described in `README.md` and
`docs/current-state.md`. Replay and checkpoint packages have been removed. Public service deployment
and credentialed market-hours validation are outside ordinary agent work.

## How to approach work

Read `README.md` first, then only the product, architecture, operational, and
implementation documents directly relevant to the task. Do not load every file
under `docs/` by default. Older design and delivery documents may explain why
the code looks the way it does, but they are reference material rather than a
required workflow.

For implementation work:

1. Establish the current behavior from the relevant code, tests, and current
   documentation.
2. Identify the smallest coherent change that solves the requested problem
   while preserving the invariants below.
3. Implement it without unrelated cleanup, speculative abstractions, or a
   second state path.
4. Run focused verification that distinguishes the intended behavior, then run
   the ordinary repository command when proportionate to the change.
5. Report what changed, what the evidence proves, important limitations, and
   the next practical decision.

Do not create extra process or planning documents unless the owner explicitly
asks for one. A short implementation brief is appropriate when a change has
meaningful ordering, ownership, persistence,
external-input, or market-semantics risk; it should contain only the objective,
design boundary, important invariants, non-scope, and focused verification.

When evidence invalidates an assumption, correct the relevant code or current
document and rerun the narrowest useful proof. Do not preserve an obsolete test,
fixture, benchmark, document, or implementation merely because it was accepted
earlier.

## Authority

Use this order when sources disagree:

1. the current owner request;
2. `docs/product/product-goals.md`;
3. the architecture documents under `docs/architecture/`;
4. the current delivery or implementation brief relevant to the task;
5. current operational documentation, tests, and code; and
6. historical design and delivery records.

Product behavior controls what the scanner means. Architecture controls state
ownership, ordering, time, lifecycle, and publication boundaries. Lower-level
documents and code must be corrected when they conflict with those meanings.

## Canonical system model

Use domain language that describes the running scanner:

- reference data and session binding;
- live transport, frame classification, and normalization;
- historical aggregate hydration and recovery;
- the `ScannerStateEngine` and canonical symbol state;
- aggregate features, qualification, and ranking;
- selected-row trade/quote coverage and enrichment;
- committed aggregate watermark and readiness;
- immutable snapshot publication and the snapshot API;
- dashboard presentation; and
- operational diagnostics and failure containment.

Avoid delivery-era numbering or internal planning terminology in new code and
documentation. Name behavior by the market fact, state transition, boundary, or
user-visible capability it represents.

## Core invariants

- There is one authoritative `ScannerStateEngine` and one canonical symbol
  state. Only the engine's ordered execution path mutates scanner state.
- Provider adapters normalize bounded facts. They do not own ranking,
  readiness, canonical merge decisions, or lifecycle transitions.
- REST and live aggregates use the same canonical identity and merge rules.
- Live item order is explicit through connection epoch, frame sequence, and
  array index. Goroutine completion and map iteration are not ordering
  authorities.
- Aggregate ranking and aggregate readiness never depend on trade/quote
  availability or health.
- T/Q may be rejected or shed only through the approved observable containment
  paths. A mixed frame must continue to preserve later aggregate and control
  facts.
- Successful empty hydration is explicit no-print evidence, not a fabricated
  mark or unfinished work.
- Every primary population and work counter participates in a documented
  accounting identity. Overlapping diagnostic dimensions remain separately
  labelled.
- A published snapshot is immutable and coherent at one committed aggregate
  watermark. API and UI readers do not observe partially applied state.
- Fresh reference resolution and aggregate hydration are the restart path.
  Replay and checkpoint restoration are not implemented.
- The UI remains independently runnable from the scanner backend.
- Inputs, queues, retries, waits, retained state, and external work are bounded.
- Do not add a database, service split, generic event bus, runtime plugin
  system, generalized framework, worker pool for canonical mutation, or another
  mutable owner without an explicit architectural need.

## Provider and evidence boundaries

Do not access provider credentials or make live provider requests unless the
owner explicitly authorizes that exact execution. A documented market-hours
procedure is not itself authorization.

Do not implement a nontrivial provider edge case without at least one of:

- provider documentation;
- a captured or owner-approved fixture;
- a production observation;
- a product or mathematical invariant;
- an existing regression; or
- explicit owner direction.

The only predecessor checkout permitted by default is:

```text
../Momentum-Equities-Live-Scanner-v2 (relative to the original project checkout)
```

Inspect it only when a concrete question cannot be answered more directly in
the current repository. Do not inspect the older Version 1 checkout without an
explicit owner-approved reason. Predecessor code is evidence, never authority;
adapt behavior into the current ownership model rather than transplanting its
structure.

## Engineering conventions

- Use Go 1.26 and the module path
  `github.com/belandresj/live-equities-momentum-scanner`.
- Prefer the Go standard library. Add a third-party dependency only when it is
  clearly required by the requested behavior.
- Prefer the minimum number of mutable states, representations, owners, and
  handoffs.
- Keep blocking network, disk, and UI work outside the engine's ordered path.
- Preserve unrelated user changes in a dirty worktree. Never use destructive
  Git operations to make the tree look clean.
- Do not stage, commit, push, rebase, amend, or rewrite history unless the owner
  asks for that Git action.

## Verification

The ordinary repository command is:

```text
go test -short -timeout 2m ./...
```

During correction, run the narrowest affected proof first. Use the smallest
deterministic fixture that proves the behavior. Validate fixture shape and work
before interpreting timing.

Capacity, benchmark, soak, race, and live work must be explicitly selected and
must be skipped by `testing.Short()` when long-running. Bound loops, retries,
channel waits, polling, and generators. Give every long command and performance
trial an explicit timeout; no local acceptance command should exceed 15 minutes
without a specific reason.

Separate these claims in both tests and reporting:

- measurement and accounting correctness;
- descriptive path or state quality;
- predictive evidence about future market behavior; and
- validated executable trading expectancy.

Passing tests establishes only the behavior those tests exercise. It does not
establish live-provider capacity, latency distributions, market-hours behavior,
or trading edge unless the evidence directly measures those claims.

## Reviews and collaboration

Use another read-only reviewer when the change has material concurrency,
persistence, security, external-evidence, sole-owner, or cross-boundary risk, or
when the owner requests one. Do not require a ceremonial review for every
change.

When multiple agents are used, keep one write-capable agent at a time. Reviewers
remain read-only. Preserve unrelated work and coordinate before touching shared
files.

## Communication

Communicate as a technically sophisticated software engineer and quantitative
trader. Lead with the actual conclusion and why it matters. Use exact behavior,
numbers, functions, and comparisons instead of project-management labels.

For quantitative work, explain what a signal measures, when it becomes knowable
in real time, what causes it to fire, how it compares with a relevant baseline,
and whether apparent edge is broad or concentrated. State uncertainty,
selection bias, costs, latency, and what remains unproven.

For implementation work, start with the capability or behavior that changed.
Explain important design choices and tradeoffs, then report verification in
terms of what it proves. Avoid long file inventories and chronological tool
summaries unless requested.
