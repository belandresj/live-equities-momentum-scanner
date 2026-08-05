# Scanner State Engine lifecycle

**Status:** Phase 1 scaffold. This document will be drafted from the owner-
reviewed decisions in `../plans/phase-1-decisions-brief.md`.

This specification will define:

- the `ScannerStateEngine` responsibilities and prohibited responsibilities;
- its internal state and event inputs;
- initialization, aggregate acknowledgement, checkpoint loading, hydration,
  live operation, recovery, suppression, and session end;
- the single-writer ordering rule;
- state-transition and terminal-outcome tables;
- readiness and publication boundaries; and
- aggregate, T/Q, checkpoint, replay, API, and UI failure isolation.

There will be no terminal state that both releases recovery work and prevents
ordinary live evaluation without an event capable of exiting that state.
