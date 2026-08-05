# Data, time, and event contract

**Status:** Phase 1 scaffold. This document will be drafted from the owner-
reviewed decisions in `../plans/phase-1-decisions-brief.md`.

This specification will define the repository-wide meaning of:

- normalized aggregate, trade, quote, control, connection, and hydration
  events;
- event time, receipt time, wall time, and replay time;
- the committed watermark;
- half-open windows and session boundaries;
- duplicate, correction, out-of-order, late, and fenced results;
- live WebSocket versus historical REST normalization; and
- deterministic replay ordering.

No component specification may invent a conflicting clock or event identity.
