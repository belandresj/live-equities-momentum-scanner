# UI design decisions and follow-ups

**Status:** Non-authoritative UI backlog; implementation is intentionally
deferred while backend work continues.

This file records concrete presentation improvements discovered while using the
live scanner. It does not change the product, API, ranking, readiness,
availability, or market-state contracts. The accepted UI contract remains
[`docs/specifications/independent-ui.md`](specifications/independent-ui.md).

When a follow-up is implemented, update its status and record the verification
result here. Keep each item focused on one observable UI decision.

## 1. Add subtle row striping to the scanner table

**Status:** Proposed  
**Priority:** High  
**Observed:** 2026-08-13 during live desktop use

### Observation

Adjacent rows currently share nearly the same dark base background, and the
thin row dividers are not sufficient to track a row horizontally. This is most
noticeable in the Rank, Symbol, Last, Day %, From 4AM %, and HOD DD % columns,
where there is little or no cell-level color. The range, Activity, Tape Rate,
and Spread fills on the right side help those individual cells but do not make
the full row boundary obvious.

### Decision

Use low-contrast alternating row backgrounds (zebra striping) in the table
body. The two fills should be adjacent dark neutrals: distinct enough to
separate neighboring rows at a glance, but restrained enough to preserve the
dense dark-scanner aesthetic.

The row base fill is presentation-only. Existing range, Activity, Tape Rate,
and Spread cell backgrounds remain visible and continue to communicate their
existing measurement scales and field states. Current, degraded, frozen, and
noncurrent states must remain distinguishable through their existing status
labels and text/state styling; row striping must not imply freshness,
qualification, or signal strength.

Striping follows the rendered table row index, not symbol identity. It must not
animate or otherwise suggest continuity when the server publishes a new rank
order. A separate hover treatment may be evaluated later, but it is not part
of this decision.

### Revisit notes

- Primary implementation surface is likely [`ui/styles.css`](../ui/styles.css);
  no backend, API, model, ranking, or row-order change is required.
- Check the leftmost columns first, then verify that the existing heatmap and
  status-cell fills are not flattened or made unreadable.
- Verify in Chrome at the required 1440x900 desktop viewport with 20 rows,
  fewer rows, a noncurrent/frozen snapshot, and a replay snapshot.
- Preserve the existing semantic-table behavior, visible keyboard focus, and
  WCAG AA text/background contrast requirements.

### Acceptance signal

At normal desktop zoom, a user can follow any adjacent row from Rank through
Spread without relying only on the thin divider lines. No displayed value,
rank, availability state, or currentness claim changes, and the specialized
cell-level visualizations remain legible.

## 2. Increase the range-position gradient intensity

**Status:** Proposed  
**Priority:** High  
**Observed:** 2026-08-13 during live desktop use

### Observation

The current range-position fills are visually muted at their endpoints. A
value near 0% does not read as strongly low-range, and a value near 100% does
not stand out as strongly high-range, especially against the dark table and
the surrounding uncolored columns.

### Decision

Make the continuous range-position gradient more vivid at its endpoints: use a
brighter, more prominent red at 0% and a brighter, more prominent green at
100%. Keep the neutral midpoint around 50% and preserve a continuous
interpolation between the endpoints rather than introducing categorical bands
or a new threshold.

This applies to the shared range-position palette used by Day Range %, 60 MIN
Range %, and 30 MIN Range %. Day Range % is the immediate motivation, but the
three columns should not develop inconsistent visual semantics. The change is
presentation-only: it must not alter the numeric value, range formula,
availability state, ranking, qualification, readiness, or any backend/API
behavior. The colors emphasize low/high position visually; they are not trade
signals or alerts.

“Brighter” should mean more vivid and easier to notice, not necessarily a
lighter fill that reduces text contrast. Preserve the existing numeric labels,
status styling, out-of-range endpoint behavior, and WCAG AA text/background
contrast. The endpoint colors must remain distinguishable from the row-striping
decision above once both are implemented.

### Revisit notes

- Primary implementation surface is likely the range-palette rules in
  [`ui/styles.css`](../ui/styles.css); no backend, API, model, or ranking
  change is required.
- Review 0%, 25%, 50%, 75%, and 100% values together so the gradient reads as
  monotonic and the midpoint remains visibly neutral.
- Verify current, degraded, frozen/noncurrent, and replay states; noncurrent
  state styling must still take precedence over a vivid current-value palette.
- Check white/text contrast on the brightest red and green endpoints at the
  required Chrome 1440x900 viewport.

### Acceptance signal

At normal desktop zoom, 0% and 100% range-position cells are immediately
recognizable as the low/high endpoints, the 50% cell remains neutral, and
intermediate values still read as a continuous gradient. No semantic value or
state changes, and the existing cell text and status reasons remain legible.

## 3. Show rank movement with up/down arrows

**Status:** Proposed  
**Priority:** High  
**Observed:** 2026-08-13 during live desktop use

### Observation

The table shows each symbol's absolute server rank, but a trader must compare
two publications mentally to notice that a row moved up or down. A compact
movement cue would make rank swaps visible without requiring a second screen or
manual memorization.

### Decision

Add a small movement glyph inside the existing Rank cell so the fixed 12-column
table does not gain a new column. Use a green up arrow (`▲`) when a symbol
moves toward rank 1 and a red/coral down arrow (`▼`) when it moves toward rank
20. Include the absolute one-publication rank delta: `▲2` means the symbol
moved up two places, while `▼1` means it moved down one place. A symbol with no
rank change gets no arrow in the initial version to avoid adding noise.

The browser compares the current displayed top-20 rank with the immediately
previous valid displayed publication for the same symbol:

- current rank lower than previous rank: up arrow;
- current rank higher than previous rank: down arrow;
- equal rank: no arrow;
- absent from the previous displayed top 20, first observation, or returning
  after leaving the top 20: no arrow because there is no direct displayed-rank
  baseline.

Movement is keyed to a new publication identity, not a sample refresh. Failed,
delayed, disconnected, degraded, or retained noncurrent responses must not
invent movement or clear the last valid indicator; the next valid qualified
publication establishes the next comparison. The arrow is transient browser
presentation state only: it does not reorder rows, recompute ranking, alter the
API, or become backend scanner state.

Color is redundant with shape and accessible text. The rank cell should expose
an accessible label or focus detail such as “rank 3, moved up from rank 5”; a
color-only arrow is insufficient. Do not animate the arrow on polling.

### Recommended transition behavior

The indicator describes only the most recent valid rank transition, not a
session-cumulative move. For example:

| Previous rank | Current rank | Display | Meaning |
| ---: | ---: | --- | --- |
| — | 5 | no arrow | No prior displayed baseline. |
| 7 | 5 | `▲2` | Moved up two places in this publication. |
| 5 | 5 | no arrow | Rank unchanged. |
| 5 | 6 | `▼1` | Moved down one place in this publication. |
| 7 → 5 | 3 | `▲2` | The next transition is 5 → 3, not cumulative `▲4`. |

The arrow and delta remain visible until the next valid qualified publication.
If that publication has no rank change, the indicator clears. There is no
separate wall-clock timeout or accumulated movement history in the first
version. This keeps the display honest at the scanner's publication cadence
and makes a two-place jump immediately explicit without creating a noisy
permanent marker.

For a concrete sequence, if the prior rank is 7 and the current publication
places the symbol at 3, the Rank cell can read `▲4 3`. If the next changed
publication places it at 2, replace that with `▲1 2`. If the following valid
publication still has rank 2, clear only the movement annotation and show rank
`2`. The stock is then simply stable at rank 2 until another observed
publication changes its rank.

The UI can report only transitions visible at publication granularity. If a
symbol moves several times between two displayed publications, the indicator
shows the net observed rank difference; it does not claim to reconstruct every
intermediate swap. If no new publication arrives, a sample refresh does not
clear the last indicator; the existing delayed/disconnected/frozen status tells
the trader that the retained view has not advanced.

### Revisit notes

- Keep the indicator in the Rank cell rather than adding a column, preserving
  the accepted table order and desktop fit.
- Retain only the previous valid displayed rank map; no client-side movement
  history is needed for the first implementation.
- Verify initial load, same-publication refresh, a two-row swap, a larger move,
  equal rank, a new symbol, a symbol leaving/returning to the top 20, and a
  disconnected/frozen interval.
- Check that the arrow remains legible alongside row striping and the stronger
  range-position colors.

### Acceptance signal

When two valid qualified publications swap rows, the displayed arrow points in
the correct direction and the absolute rank remains server-provided. No arrow
appears when the comparison is not valid, and the table never changes order or
meaning because of the indicator.

## 4. Use neutral near-white for primary table text

**Status:** Proposed  
**Priority:** High  
**Observed:** 2026-08-13 during live desktop use

### Observation

The current `data-state="current"` rule makes most current table values mint/
teal (`#7CE2B3`). That color is readable against the dark blue base, but using
it across nearly every current cell makes the whole table equally accented and
reduces the visual hierarchy between ordinary data and status/attention cues.

### Decision

Use a neutral cool off-white, approximately `#E7EEF5`, for primary table
values. Keep muted blue-gray for headers and secondary labels. Reserve mint/teal
for positive, healthy, or current status accents and upward rank arrows; use
amber/orange for warnings and a contrast-safe red/coral for downward movement
and error accents. Range, Activity, Tape Rate, and Spread cells may retain their
specialized palettes, subject to endpoint contrast review.

The current teal is not intrinsically illegible: against the table base
`#071019`, `#7CE2B3` is approximately 12.2:1 by WCAG relative-luminance
contrast, while `#E7EEF5` is approximately 16.4:1. The main improvement is
therefore hierarchy and reduced chromatic noise, with some additional
luminance headroom—not a change to market meaning or a claim that teal fails
accessibility.

### Revisit notes

- Primary implementation surface is the table-text/state selectors in
  [`ui/styles.css`](../ui/styles.css); no backend, API, or model change is
  required.
- Audit body text on the row-stripe fills and on every range/heatmap endpoint;
  contrast must remain WCAG AA at the required Chrome desktop viewport.
- Keep color redundant with labels, numeric values, arrow shape, and explicit
  status/reason text. Color must not be the only encoding of currentness,
  movement, availability, or error.
- Check the resulting hierarchy at a glance: neutral table values first,
  movement/range emphasis second, operational status accents third.

### Acceptance signal

The main table reads as a neutral high-contrast data grid, while teal remains a
deliberate accent rather than the default color of every current value. Rank
arrows, range endpoints, warnings, and field-status reasons remain visually
distinct and legible without changing any displayed data or state semantics.
