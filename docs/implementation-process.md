# Implementation process

**Status:** Current lightweight repository workflow.

This process exists to keep changes understandable, bounded, and evidence-led.
It does not prescribe a document template or require planning artifacts before
ordinary implementation work.

## 1. Understand the behavior

Start with `README.md`, then read only the product, architecture, operational,
and implementation material directly relevant to the task. Confirm current
behavior in code and tests rather than assuming an older document still matches
the implementation.

Resolve conflicts in this order:

1. current owner direction;
2. approved product behavior;
3. approved architecture;
4. a current implementation brief or operational document for the task; and
5. tests and code.

Older design and delivery records may explain historical decisions, but they do
not control current workflow or override current product and architecture.

## 2. Choose the smallest coherent change

State the practical objective in plain language. Identify the code path that
owns the behavior and the invariants the change must preserve. Prefer one
coherent implementation over scaffolding, parallel state paths, generalized
frameworks, or unrelated cleanup.

Write a short implementation brief only when it materially reduces ambiguity or
risk. A useful brief contains:

- the behavior to add or correct;
- the narrow implementation boundary;
- important ordering, ownership, time, accounting, or external-input
  invariants;
- explicit non-scope; and
- the focused evidence needed for the next decision.

No separate plan document is required for routine work. Update current
documentation directly when behavior or operating guidance changes.

## 3. Implement in the owning path

Keep canonical mutation in the sole `ScannerStateEngine`. Provider and worker
code return bounded facts or immutable results. Blocking network, disk, and UI
work stays outside the ordered engine path.

Preserve explicit causal metadata and accounting. Do not make aggregate ranking
or readiness depend on trade/quote health. Do not create another owner, queue,
or representation merely to make an implementation convenient.

If new evidence invalidates the intended approach, revise the approach and
continue with the narrowest coherent correction. Do not preserve an obsolete
fixture, test, or document when it conflicts with current behavior.

## 4. Verify the decision

Run the narrowest affected test while iterating. Use deterministic fixtures and
validate their shape before interpreting results. Add a regression when it
distinguishes a real failure mode or consequential boundary; avoid duplicate
tests that assert the same fact at multiple layers.

The ordinary repository command is:

```text
go test -short -timeout 2m ./...
```

Use explicit opt-in commands for capacity, benchmark, soak, race, or live work.
Long-running tests must skip under `testing.Short()`. Bound all retries, waits,
polling, generators, and trial durations. Give every long command an explicit
timeout, and keep local runs below 15 minutes unless the task records a concrete
reason.

Verification should answer what the result proves, not merely whether a command
returned zero. Distinguish:

- semantic and accounting correctness;
- performance or capacity on the exercised host and fixture;
- live-provider behavior;
- predictive market evidence; and
- executable trading expectancy.

Do not infer a stronger claim than the evidence supports.

## 5. Review according to risk

Use a separate read-only review for material concurrency, persistence,
atomicity, security, external-evidence, sole-owner, or cross-boundary changes,
or when the owner asks for one. Keep review focused on the actual risk and reuse
the same reviewer for a narrow re-review when practical.

Routine localized changes do not require ceremonial review. A review finding is
evidence to correct the implementation, not a separate approval workflow.

## 6. Hand off clearly

Lead with the behavior now available and why it matters. Report:

- the important design choice and tradeoff;
- the focused verification and what it proves;
- meaningful limitations or untested conditions; and
- the next practical decision, if one remains.

Avoid chronological tool summaries, internal workflow terminology, and long
file inventories unless the owner requests them.
