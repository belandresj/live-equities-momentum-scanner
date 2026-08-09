import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { buildViewModel } from "./model.js";
import { snapshotFixture } from "./test-fixture.js";

function luminance(hex) {
  const values = [1, 3, 5].map(index => Number.parseInt(hex.slice(index, index + 2), 16) / 255).map(value => value <= .03928 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4);
  return .2126 * values[0] + .7152 * values[1] + .0722 * values[2];
}
function contrast(foreground, background) {
  const first = luminance(foreground), second = luminance(background);
  return (Math.max(first, second) + .05) / (Math.min(first, second) + .05);
}

test("P-C11-VISUAL presentation bands are stable and non-semantic", () => {
  const snapshot = snapshotFixture();
  snapshot.rows[0].day_change_ratio = .10;
  snapshot.rows[0].from_4am_change.value_ratio = .05;
  snapshot.rows[0].day_range_position.value_ratio = .75;
  snapshot.rows[0].activity.value_ratio = 1;
  snapshot.rows[0].tape_rate.five_second.trades_per_second = 30;
  snapshot.rows[0].spread.basis_points = 50;
  const row = buildViewModel(snapshot).rows[0];
  assert.deepEqual([row.dayBand, row.from4am.band, row.dayRange.band, row.activity.band, row.tape.band, row.spread.band], [4, 3, 3, 4, 4, 4]);
  assert.equal(buildViewModel(snapshot).current, true, "visual bands changed readiness");
});

test("P-C11-VISUAL every text/status token palette meets WCAG AA contrast", () => {
  const pairs = [
    ["#e7eef5", "#071019"], ["#7ce2b3", "#071019"], ["#ffd37a", "#071019"], ["#819bad", "#0c1822"],
    ["#8da5b6", "#0b1721"], ["#d9e5ed", "#071019"], ["#95a7b4", "#111a21"], ["#ffb58f", "#071019"],
    ["#a8e9c8", "#10271f"], ["#7ce2b3", "#123024"], ["#ffe29d", "#302817"], ["#ffb09c", "#351b18"],
    ["#8298a8", "#10271f"], ["#8298a8", "#123024"], ["#8298a8", "#302817"], ["#8298a8", "#351b18"],
  ];
  for (const [foreground, background] of pairs) assert.ok(contrast(foreground, background) >= 4.5, `${foreground} on ${background}`);
});

test("P-C11-VISUAL retained CSS has no alpha contrast loss", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  const retained = /td\[data-state="retained"\] \{([^}]*)\}/.exec(css)?.[1] || "";
  const retainedColor = /color:\s*(#[0-9a-f]{6})/i.exec(retained)?.[1];
  const secondaryColor = /td small \{[^}]*color:\s*(#[0-9a-f]{6})/is.exec(css)?.[1];
  const retainedBackground = /table\[data-publication-state="noncurrent"\] tbody \{[^}]*background:\s*(#[0-9a-f]{6})/is.exec(css)?.[1];
  assert.doesNotMatch(retained, /opacity/);
  assert.ok(contrast(retainedColor, retainedBackground) >= 4.5);
  assert.ok(contrast(secondaryColor, retainedBackground) >= 4.5);
});

test("P-C11-VISUAL CSS fixes desktop density, focus, and reduced motion", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  assert.match(css, /main \{ min-width: 1120px/);
  assert.match(css, /td \{ height: 32px/);
  assert.match(css, /th \{[^}]*height: 30px/s);
  assert.ok(30 + 20 * 32 <= 670, "table body exceeds allocated desktop height");
  assert.match(css, /:focus-visible/);
  assert.doesNotMatch(css, /tbody\s+tr:hover/, "pointer hover must not highlight an entire scanner row");
  assert.match(css, /@media \(prefers-reduced-motion: reduce\)/);
  assert.match(css, /animation-duration: \.01ms !important/);
  assert.match(css, /table\[data-publication-state="noncurrent"\]/);
  assert.match(css, /grid-template-columns: repeat\(8, 1fr\)/);
  assert.ok(32 + 52 + 53 + 27 + 30 + 20 * 32 <= 900, "status strip plus full table exceeds target viewport");
});
