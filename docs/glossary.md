# Glossary

**Status:** Phase 1 scaffold. Definitions will be completed and approved with
the architecture specifications.

- **Accepted event:** A normalized provider event that passed structural,
  session, generation, and canonical acceptance rules.
- **Canonical symbol state:** The single authoritative representation of a
  symbol's accepted same-session market data and derived feature state.
- **Committed watermark (`T`):** The market-time boundary through which the
  engine has incorporated all accepted work required for a published result.
- **Coverage:** The exact interval or causal boundary over which an input or
  derived measurement is trustworthy.
- **Hydration:** Bounded historical loading used for fresh bootstrap,
  checkpoint catch-up, or recovery of a known coverage gap.
- **No print through T:** Complete evidence that no accepted aggregate mark
  exists for the symbol through committed boundary `T`. It is a terminal
  unrankable state at `T`, not a synthetic price and not unfinished work.
- **Normalized event:** Provider-independent aggregate, trade, quote, control,
  connection, or hydration result consumed by the scanner.
- **Qualification:** The session-latched aggregate tape requirement that a
  symbol must pass before entering the contracted ranked table.
- **Rankable:** A symbol has the trusted inputs required to calculate Day return
  and passes non-qualification eligibility rules such as the price floor.
- **Scanner State Engine:** The single ordered owner of scanner lifecycle,
  canonical state mutation, committed time, evaluation, and publication.
- **Snapshot:** An immutable published view of ranking, features, readiness,
  accounting, and bounded diagnostics at one committed watermark.
- **T/Q:** Selected-symbol trade (`T`) and quote (`Q`) streaming used for Tape
  Rate, Spread, and future trade/quote-derived features.
