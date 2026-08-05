# System overview

**Status:** Phase 1 scaffold. This document will be drafted from the owner-
reviewed decisions in `../plans/phase-1-decisions-brief.md`.

This specification will be the primary architectural entry point. It will
describe:

- every external data source;
- the live, bootstrap, restart, recovery, and offline replay paths;
- the high-level components and their responsibilities;
- the owner of every mutable state domain;
- how components interconnect;
- aggregate/T/Q failure independence;
- checkpoint and UI boundaries; and
- links to deeper component specifications.

The complete runtime model should be understandable from this document without
replaying implementation details.
