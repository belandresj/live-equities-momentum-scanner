import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { buildViewModel, formatShares, PollController, readBoundedJSON, validateSnapshot } from "./model.js";
import { renderDashboard } from "./render.js";
import { snapshotFixtureV2 } from "./test-fixture-v2.js";

const clone = value => structuredClone(value);
function clearTQ(snapshot) {
  snapshot.tq.desired_symbols = []; snapshot.tq.known_present = 0;
  for (const row of snapshot.rows) {
    row.tq_membership = { desired: false, provider_present: false, provider_membership_unknown: false };
    row.tape_5s = { status: "unselected", reason: "", trade_coverage: false, trades_per_second: null, timestamp_basis: "", lifecycle_records_observed: false };
    row.spread = { status: "unselected", reason: "", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  }
}

test("P-MVP-UI preserves v2 server order, exact rows, signed units, compact shares, and genuine zero", () => {
  const snapshot = snapshotFixtureV2(20); snapshot.sample.id = "9007199254740999";
  [snapshot.rows[0], snapshot.rows[1]] = [snapshot.rows[1], snapshot.rows[0]]; snapshot.rows.forEach((row, index) => { row.rank = index + 1; }); snapshot.tq.desired_symbols = snapshot.rows.map(row => row.symbol);
  const model = buildViewModel(snapshot);
  assert.equal(model.schemaVersion, "scanner.snapshot.v2"); assert.equal(model.sampleID, "9007199254740999");
  assert.deepEqual(model.rows.map(row => row.symbol), snapshot.rows.map(row => row.symbol), "client reordered server rows");
  assert.equal(model.rows[0].float.text, "13M · stale"); assert.match(model.rows[0].float.detail, /effective date 2026-08-07.*provenance cache/);
  assert.equal(model.rows[1].volume.text, "0"); assert.equal(model.rows[1].day, "+2.15%"); assert.equal(model.rows[1].fromOpen.text, "0.00%");
  assert.equal(model.rows[1].move.text, "+1.25%"); assert.equal(model.rows[0].move.text, "-0.75%"); assert.equal(model.rows[1].tape.primary, "1.4/s");
  assert.deepEqual([buildViewModel(snapshotFixtureV2(3)).rows.length, buildViewModel(snapshotFixtureV2(0)).rows.length], [3, 0]);
  assert.deepEqual([formatShares(999), formatShares(1_250), formatShares(999_999), formatShares(999_999_999), formatShares(12_000_000), formatShares(2_500_000_000)], ["999", "1.3K", "1M", "1B", "12M", "2.5B"]);
});

test("P-MVP-UI retains only exact stale Float/Spread tuples and exposes provenance", () => {
  const snapshot = snapshotFixtureV2(); snapshot.rows[0].float = { ...snapshot.rows[0].float, status: "stale", reason: "cached_fallback", provenance: "cache" };
  snapshot.rows[0].spread = { ...snapshot.rows[0].spread, status: "stale", reason: "stale_quote", quote_age_ms: 45 * 60 * 1000 };
  const row = buildViewModel(snapshot).rows[0]; assert.equal(row.float.text, "12M · stale"); assert.equal(row.spread.primary, "15.0 bps / 1.50¢ · stale"); assert.match(row.spread.detail, /quote age 2700000 ms/);
  for (const mutate of [value => { value.rows[0].float.status = "stale"; value.rows[0].float.reason = "cached_fallback"; }, value => { value.rows[0].float.value_shares = null; }, value => { value.rows[0].spread.reason = "stale_quote"; }, value => { value.rows[0].spread.cents = null; }]) {
    const invalid = clone(snapshotFixtureV2()); mutate(invalid); assert.throws(() => validateSnapshot(invalid), /conflict/);
  }
});

test("P-MVP-UI handles independent field states without fabricating zero", () => {
  const snapshot = snapshotFixtureV2(6);
  snapshot.rows[0].from_open_change = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  snapshot.rows[1].day_range_position = { status: "invalid", reason: "historical_conflict", value_ratio: null };
  snapshot.rows[2].activity_30s = { status: "warming", reason: "reference_warmup", value_ratio: null };
  snapshot.rows[3].move_30s = { status: "warming", reason: "rolling_warmup", value_ratio: null };
  snapshot.rows[4].volume = { status: "unavailable", reason: "history_incomplete", value_shares: null };
  snapshot.rows[5].float = { status: "unavailable", reason: "not_available", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" };
  snapshot.rows[0].tape_5s = { ...snapshot.rows[0].tape_5s, status: "pressure_shed", reason: "pressure", trade_coverage: false, trades_per_second: null, timestamp_basis: "" };
  snapshot.rows[0].spread = { status: "warming", reason: "coverage_warming", quote_coverage: true, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  const rows = buildViewModel(snapshot).rows;
  assert.deepEqual([rows[0].fromOpen.state, rows[1].dayRange.state, rows[2].activity.state, rows[3].move.state, rows[4].volume.state, rows[5].float.state], ["unavailable", "invalid", "warming", "warming", "unavailable", "unavailable"]);
  assert.ok([rows[0].fromOpen.text, rows[1].dayRange.text, rows[2].activity.text, rows[3].move.text, rows[4].volume.text, rows[5].float.text].every(text => text === "—"));
  assert.equal(rows[0].tape.state, "pressure_shed"); assert.equal(rows[0].spread.state, "warming");
});

test("P-MVP-UI fails closed on v1, bad rank/symbol/bound, and contradictory tuples", () => {
  const edits = [value => { value.schema_version = "scanner.snapshot.v1"; }, value => { value.rows[0].rank = 2; }, value => { value.rows[0].symbol = ""; }, value => { value.rows[0].activity_30s.value_ratio = null; }, value => { value.rows[0].activity_30s.reason = "future_reason"; }, value => { value.rows[0].volume.value_shares = -1; }, value => { value.rows[0].tape_5s.trades_per_second = null; }, value => { value.accounting.population.covered_population = 0; value.accounting.population.unresolved_population = 2; }];
  for (const edit of edits) { const snapshot = snapshotFixtureV2(); edit(snapshot); assert.throws(() => validateSnapshot(snapshot)); }
  const duplicate = snapshotFixtureV2(2); duplicate.rows[1].symbol = duplicate.rows[0].symbol; assert.throws(() => validateSnapshot(duplicate), /rank\/symbol/);
  const tooMany = snapshotFixtureV2(20); tooMany.rows.push(clone(tooMany.rows[19])); tooMany.rows[20].rank = 21; assert.throws(() => validateSnapshot(tooMany), /at most 20/);
});

test("P-MVP-UI preserves status hierarchy and T/Q-independent current rows", () => {
  const pressure = snapshotFixtureV2(); pressure.status.tq_pressure_mode = "aggregate_only"; pressure.status.tq_shed = true; pressure.tq.pressure_mode = "aggregate_only"; pressure.tq.pressure_cause = "tq_retention_bound"; pressure.tq.aggregate_only = true; pressure.tq.shed = true;
  pressure.rows[0].tape_5s = { ...pressure.rows[0].tape_5s, status: "pressure_shed", reason: "pressure", trade_coverage: false, trades_per_second: null, timestamp_basis: "" }; pressure.rows[0].spread = { status: "pressure_shed", reason: "pressure", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  const model = buildViewModel(pressure); assert.equal(model.current, true); assert.equal(model.tqAggregateOnly, true);
  const partial = snapshotFixtureV2(); partial.ranking.mode = "degraded_current"; partial.ranking.reason = "qualification_incomplete"; clearTQ(partial); const partialModel = buildViewModel(partial); assert.equal(partialModel.partial, true); assert.equal(partialModel.rowsCurrent, false);
});

test("P-MVP-UI bounded poller skips overlap, freezes, resamples, replaces, and recovers", async () => {
  const updates = []; let resolveFirst; const deferred = new Promise(resolve => { resolveFirst = resolve; });
  const controller = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", requestTimeoutMilliseconds: 30, fetchImpl: () => deferred, onUpdate: update => updates.push(update) });
  const first = controller.tick(); await controller.tick(); assert.equal(updates.at(-1).transport, "refresh_delayed"); resolveFirst(new Response(JSON.stringify(snapshotFixtureV2()))); await first;
  const same = snapshotFixtureV2(); same.sample.id = "11"; same.sample.sampled_at = "2026-08-08T16:00:01Z"; controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(same))); await controller.tick(); assert.equal(updates.at(-1).model.sampleID, "11");
  const next = snapshotFixtureV2(); next.publication.id = "21"; next.rows[0].symbol = "REPLACED"; next.tq.desired_symbols[0] = "REPLACED"; controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(next))); await controller.tick(); assert.equal(updates.at(-1).model.rows[0].symbol, "REPLACED");
  controller.fetchImpl = () => Promise.resolve(new Response("{}", { status: 503 })); await controller.tick(); assert.equal(updates.at(-1).transport, "disconnected"); assert.equal(updates.at(-1).model.publicationID, "21"); controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(snapshotFixtureV2()))); await controller.tick(); assert.equal(updates.at(-1).transport, "connected");
});

test("P-MVP-UI enforces streamed response bound", async () => {
  await assert.rejects(readBoundedJSON(new Response(`{"x":"${"a".repeat((1 << 20) + 1)}"}`)), /exceeds 1 MiB/); await assert.rejects(readBoundedJSON(new Response("{}", { headers: { "content-length": String((1 << 20) + 1) } })), /exceeds 1 MiB/);
});

class FakeNode {
  constructor(tag, document = null) { this.tagName = tag.toUpperCase(); this.ownerDocument = document; this.children = []; this.dataset = {}; this.attributes = {}; this.hidden = false; this.open = false; this.id = ""; this.className = ""; this._text = ""; }
  set textContent(value) { this._text = String(value); this.children = []; } get textContent() { return this._text + this.children.map(child => child.textContent).join(""); } append(...nodes) { this.children.push(...nodes); } replaceChildren(...nodes) { this._text = ""; this.children = nodes; } setAttribute(name, value) { this.attributes[name] = String(value); }
  querySelector(selector) { const match = /^\[data-focus-key="(.*)"\]$/.exec(selector); return match ? find(this, node => node.dataset.focusKey === match[1])[0] || null : null; } focus() { if (this.ownerDocument) this.ownerDocument.activeElement = this; }
}
class FakeDocument { constructor() { this.body = new FakeNode("body", this); this.activeElement = null; } createElement(tag) { return new FakeNode(tag, this); } getElementById(id) { return find(this.body, node => node.id === id)[0] || null; } }
function find(node, predicate, result = []) { if (predicate(node)) result.push(node); for (const child of node.children) find(child, predicate, result); return result; }
globalThis.CSS ??= { escape: value => String(value).replace(/["\\]/g, "\\$&") };

test("P-MVP-UI renders exact groups/columns safely and atomically with keyed focus", async () => {
  const document = new FakeDocument(), snapshot = snapshotFixtureV2(2); snapshot.rows[0].symbol = "<img src=x onerror=owned()>"; snapshot.tq.desired_symbols[0] = snapshot.rows[0].symbol; renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  assert.match(document.body.textContent, /CONTEXT.*LOCATION.*CURRENT MOMENTUM.*EXECUTION/s);
  const headers = find(document.body, node => node.tagName === "TH"); assert.deepEqual(headers.filter(node => node.attributes.scope === "col").map(node => node.textContent), ["SYMBOL", "FLOAT", "VOLUME", "LAST", "DAY %", "FROM OPEN %", "DAY RANGE", "ACTIVITY 30s", "MOVE 30s", "TAPE 5s", "SPREAD"]); assert.ok(headers.every(node => node.textContent !== "Rank")); assert.equal(find(document.body, node => node.tagName === "TBODY")[0].children[0].children.length, 11);
  const committed = document.body.children[0]; assert.throws(() => renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }, { beforeCommit: () => { throw new Error("render failed"); } })); assert.equal(document.body.children[0], committed);
  const focused = find(document.body, node => node.dataset.focusKey?.endsWith(":spread"))[0]; focused.focus(); const key = focused.dataset.focusKey; [snapshot.rows[0], snapshot.rows[1]] = [snapshot.rows[1], snapshot.rows[0]]; snapshot.rows.forEach((row, index) => { row.rank = index + 1; }); snapshot.tq.desired_symbols = snapshot.rows.map(row => row.symbol); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) }); assert.equal(document.activeElement.dataset.focusKey, key);
  renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }); assert.equal(document.activeElement.dataset.focusKey, "diagnostics:summary"); const source = await readFile(new URL("./render.js", import.meta.url), "utf8"); assert.doesNotMatch(source, /innerHTML|outerHTML|insertAdjacentHTML|document\.write/);
});

test("P-MVP-UI renders genuine zero Move neutrally", () => {
  const document = new FakeDocument(), snapshot = snapshotFixtureV2(); snapshot.rows[0].move_30s.value_ratio = 0; renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const move = find(document.body, node => node.dataset.focusKey?.endsWith(":move"))[0]; assert.equal(move.textContent, "0.00%"); assert.equal(move.dataset.palette, undefined);
});

test("P-MVP-UI assets contain only v2 and final-field representations", async () => {
  const all = (await Promise.all(["app.js", "model.js", "render.js", "index.html"].map(name => readFile(new URL(`./${name}`, import.meta.url), "utf8")))).join("\n"); assert.match(all, /api\/v2\/snapshot/); assert.doesNotMatch(all, /api\/v1\/snapshot|scanner\.snapshot\.v1|from_4am|hod_drawdown|range_30m|range_60m|tape_rate/);
});
