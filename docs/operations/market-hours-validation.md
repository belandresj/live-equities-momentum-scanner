# Market-hours validation

Market-hours validation is a separately authorized observation of the private
scanner against the live provider. It is not part of ordinary development,
offline acceptance, or documentation cleanup.

## Authorization boundary

Before any run, obtain explicit owner authorization for the exact date,
duration, entry point, and intended evidence. The existence of local
credentials, cached data, this procedure, or passing offline tests is not
authorization.

Do not inspect or print the credential. Do not capture raw provider payloads or
write provider URLs, headers, symbols, or credentials into an incident. Stop if
the requested observation would exceed the authorized scope.

## Preconditions

- The ordinary short test suite passes for the intended revision.
- The repository changes and revision under observation are recorded.
- Local ports and disk budget are available.
- The exchange trading date and observation interval are explicit.
- The expected process, readiness, watermark, queue, hydration, T/Q, and
  accounting measurements are named before the run.
- Every command and observation window has a bounded timeout.

## Observation

Use the supported private launcher unless the authorization explicitly names a
lower-level diagnostic entry point. Observe only bounded aggregate metrics and
closed reasons from local liveness, readiness, snapshot, and permitted incident
outputs.

Record:

- actual session/date and wall-clock interval;
- build revision and relevant runtime options;
- universe and hydration work counts, terminal outcomes, and fence status;
- time to liveness and readiness;
- committed-watermark lag over time;
- decoded-batch queue maximum slots/bytes/age, capacity drops, and terminal accounting;
- pressure transitions and T/Q coverage separately from aggregate readiness;
- publication/accounting validity; and
- controlled shutdown outcome.

For a market-open throughput claim, distinguish offered provider rate from
scanner service rate and check that queue growth is not merely hidden by later
drain or T/Q shedding.

## Stop conditions

Stop and contain the run on credential exposure, unbounded output, repeated
capacity terminal, invalid accounting, unexplained suppression, uncontrolled
process behavior, or evidence outside the authorized boundary. Do not change
queue, byte, pressure, readiness, or fidelity settings during the run to make
the result appear successful.

## Interpretation

Report what occurred under the exact observed market load. Compare it with the
relevant deterministic baseline and state whether behavior was broad or driven
by a short burst or small symbol set. A successful run does not prove future
capacity, provider guarantees, predictive value, or executable trading
expectancy.

Live evidence may motivate a focused correction, but the correction returns to
offline deterministic proof before another separately authorized live run.
