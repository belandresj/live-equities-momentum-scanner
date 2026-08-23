import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { buildViewModel } from "./model.js";
import { snapshotFixtureV2 } from "./test-fixture-v2.js";

function luminance(hex) { const values = [1, 3, 5].map(index => Number.parseInt(hex.slice(index, index + 2), 16) / 255).map(value => value <= .03928 ? value / 12.92 : ((value + .055) / 1.055) ** 2.4); return .2126 * values[0] + .7152 * values[1] + .0722 * values[2]; }
function contrast(foreground, background) { const a = luminance(foreground), b = luminance(background); return (Math.max(a, b) + .05) / (Math.min(a, b) + .05); }

test("P-MVP-UI visual grammar keeps backend-neutral continuous presentation", () => {
  const snapshot = snapshotFixtureV2(3); snapshot.rows[0].day_range_position.value_ratio = .75; snapshot.rows[0].activity_30s.value_ratio = 1; snapshot.rows[0].tape_5s.trades_per_second = 30; snapshot.rows[0].spread.basis_points = 50;
  [snapshot.rows[0].from_open_change.value_ratio, snapshot.rows[1].from_open_change.value_ratio, snapshot.rows[2].from_open_change.value_ratio] = [1, .2, -.1];
  const row = buildViewModel(snapshot).rows[0]; assert.deepEqual([row.dayColor, row.fromOpen.colorPosition, row.dayRange.position, row.activity.position, row.tape.textColor, row.spread.textColor], [1, 1, 75, 100, "#8F9AA3", "#C48649"]); assert.equal(buildViewModel(snapshot).current, true);
  assert.equal(buildViewModel(snapshot).rows[1].fromOpen.colorPosition, 0);
});

test("P-MVP-UI text/status palettes meet WCAG AA contrast", () => {
  const pairs = [["#e7eef5", "#071019"], ["#7ce2b3", "#071019"], ["#ffd37a", "#071019"], ["#819bad", "#0c1822"], ["#8da5b6", "#0b1721"], ["#d9e5ed", "#071019"], ["#95a7b4", "#111a21"], ["#ffb58f", "#071019"], ["#ffe29d", "#302817"], ["#ffb09c", "#351b18"], ["#e7eef5", "#202a31"], ["#e7eef5", "#80400d"], ["#4a965d", "#071019"], ["#2cff05", "#071019"], ["#00ffff", "#071019"]];
  for (const [foreground, background] of pairs) assert.ok(contrast(foreground, background) >= 4.5, `${foreground} on ${background}`);
});

test("P-MVP-UI CSS fixes desktop density, groups, header tooltips, retained contrast, and reduced motion", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  assert.match(css, /main \{ min-width: 1120px/); assert.match(css, /\.dashboard \{[^}]*min-height: 100vh/); assert.match(css, /\.scanner-panel \{[^}]*display: flex[^}]*flex: 1[^}]*flex-direction: column/); assert.match(css, /\.table-shell \{[^}]*--scanner-row-height: 32px[^}]*container-type: size[^}]*flex: 1[^}]*min-height: 0[^}]*overflow-x: auto[^}]*overflow-y: hidden/); assert.doesNotMatch(css, /\.table-shell \{[^}]*height: 100%/); assert.match(css, /\.system-status/); assert.match(css, /\.status-chevron \{[^}]*display: inline-flex[^}]*width: 20px[^}]*height: 20px[^}]*align-items: center[^}]*justify-content: center/); assert.match(css, /\.status-panel \{[^}]*position: absolute/); assert.doesNotMatch(css, /\.status-grid/); assert.match(css, /td \{ height: var\(--scanner-row-height\)/); assert.match(css, /\.group-header th/); assert.match(css, /\.leaf-header th/); assert.match(css, /:focus-visible/); assert.doesNotMatch(css, /tbody\s+tr:hover/); assert.match(css, /@media \(prefers-reduced-motion: reduce\)/); assert.match(css, /animation-duration: \.01ms !important/); assert.match(css, /table\[data-publication-state="noncurrent"\]/); assert.doesNotMatch(css, /data-palette="range"|data-range-(?:red|green)-weight/); assert.doesNotMatch(css, /data-palette="heat"|data-heat-weight|#80400d/);
  assert.match(css, /\.leaf-header th\[data-tooltip\]::after/); assert.match(css, /content: attr\(data-tooltip\)/); assert.match(css, /transition: opacity 80ms ease/); assert.match(css, /pointer-events: none/); assert.match(css, /\.leaf-header th\[data-tooltip\]:hover::after/);
  assert.match(css, /table col\.rank-column \{ width: 7%/); assert.match(css, /\.rank-metadata \{[^}]*width: 70px[^}]*grid-template-columns: 25px 45px/); assert.match(css, /rank-movement\[data-direction="up"\][^}]*#70B873/); assert.match(css, /rank-movement\[data-direction="down"\][^}]*#EF3030/);
  assert.match(css, /#4A965D/); assert.match(css, /#2CFF05/); assert.match(css, /color-mix\(in oklab, #4A965D, #2CFF05 var\(--day-weight\)\)/); assert.match(css, /in oklab/);
  assert.match(css, /attr\(data-day-weight type\(<percentage>\), 50%\)/); assert.match(css, /attr\(data-from-open-weight type\(<percentage>\), 50%\)/); assert.match(css, /data-palette="from-open"/);
  assert.match(css, /#00FFFF/); assert.match(css, /color-mix\(in oklab, #d9e5ed, #00FFFF var\(--volume-turnover-weight\)\)/); assert.match(css, /attr\(data-volume-turnover-weight type\(<percentage>\), 0%\)/);
  assert.match(css, /data-palette="float"/); assert.match(css, /attr\(data-float-weight type\(<percentage>\), 0%\)/); assert.match(css, /color-mix\(in oklab, #d9e5ed, #00FFFF var\(--float-weight\)\)/);
  const floatRule = css.match(/data-palette="float"\]\s*\{([^}]*)\}/)?.[1] || ""; assert.match(floatRule, /color:/); assert.doesNotMatch(floatRule, /background:/);
  assert.doesNotMatch(css, /data-palette="move-(?:positive|negative)"/); assert.doesNotMatch(css, /data-palette="spread"|background: #10271f|background: #123024|background: #302817|background: #351b18/);
  assert.ok(32 + 52 + 53 + 27 + 54 + 20 * 32 <= 900, "full dashboard exceeds target height allocation");
});

test("P-MVP-UI allocates distinct compact Rank and Symbol columns without widening the table", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  assert.match(css, /table col\.rank-column \{ width: 7%; \}/);
  assert.match(css, /table col\.symbol-column, table col\.context-column \{ width: 5%; \}/);
  assert.match(css, /table col\.location-column \{ width: 9%; \}/);
  assert.match(css, /table col\.momentum-column \{ width: 9%; \}/);
  assert.match(css, /table col\.execution-column \{ width: 14%; \}/);
  assert.doesNotMatch(css, /td:first-child\s*\{[^}]*width:/);
});

test("P-MVP-UI stripes current rows by displayed position and preserves noncurrent styling", async () => {
  const css = await readFile(new URL("./styles.css", import.meta.url), "utf8");
  assert.match(css, /table\[data-publication-state="current"\] > tbody > tr:nth-child\(odd\) \{ background: #071019; \}/);
  assert.match(css, /table\[data-publication-state="current"\] > tbody > tr:nth-child\(even\) \{ background: #0A1823; \}/);
  assert.match(css, /table\[data-publication-state="noncurrent"\] tbody \{ background: #111a21; \}/);
  assert.doesNotMatch(css, /data-(?:rank|symbol)-.*background/);
});
