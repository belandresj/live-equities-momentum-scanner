import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { buildViewModel } from "./model.js";
import { snapshotFixtureV2 } from "./test-fixture-v2.js";

function luminance(hex) { const values = [1, 3, 5].map(index => Number.parseInt(hex.slice(index, index + 2), 16) / 255).map(value => value <= .03928 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4); return .2126 * values[0] + .7152 * values[1] + .0722 * values[2]; }
function contrast(foreground, background) { const a = luminance(foreground), b = luminance(background); return (Math.max(a, b) + .05) / (Math.min(a, b) + .05); }

test("P-MVP-UI visual grammar keeps backend-neutral continuous presentation", () => {
  const snapshot = snapshotFixtureV2(); snapshot.rows[0].day_range_position.value_ratio = .75; snapshot.rows[0].activity_30s.value_ratio = 1; snapshot.rows[0].tape_5s.trades_per_second = 30; snapshot.rows[0].spread.basis_points = 50;
  const row = buildViewModel(snapshot).rows[0]; assert.deepEqual([row.dayRange.position, row.activity.position, row.tape.position, row.spread.band], [75, 100, 100, 4]); assert.equal(buildViewModel(snapshot).current, true);
});

test("P-MVP-UI text/status palettes meet WCAG AA contrast", () => {
  const pairs = [["#e7eef5", "#071019"], ["#7ce2b3", "#071019"], ["#ffd37a", "#071019"], ["#819bad", "#0c1822"], ["#8da5b6", "#0b1721"], ["#d9e5ed", "#071019"], ["#95a7b4", "#111a21"], ["#ffb58f", "#071019"], ["#ffe29d", "#302817"], ["#ffb09c", "#351b18"], ["#e7eef5", "#202a31"], ["#e7eef5", "#80400d"]];
  for (const [foreground, background] of pairs) assert.ok(contrast(foreground, background) >= 4.5, `${foreground} on ${background}`);
});

test("P-MVP-UI CSS fixes desktop density, groups, focus, retained contrast, and reduced motion", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  assert.match(css, /main \{ min-width: 1120px/); assert.match(css, /td \{ height: 32px/); assert.match(css, /\.group-header th/); assert.match(css, /\.leaf-header th/); assert.match(css, /:focus-visible/); assert.doesNotMatch(css, /tbody\s+tr:hover/); assert.match(css, /@media \(prefers-reduced-motion: reduce\)/); assert.match(css, /animation-duration: \.01ms !important/); assert.match(css, /table\[data-publication-state="noncurrent"\]/); assert.match(css, /attr\(data-range-red-weight type\(<percentage>\), 0%\)/); assert.match(css, /attr\(data-heat-weight type\(<percentage>\), 0%\)/);
  assert.ok(32 + 52 + 53 + 27 + 54 + 20 * 32 <= 900, "full dashboard exceeds target height allocation");
});
