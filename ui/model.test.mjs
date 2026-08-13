import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { buildViewModel, PollController, readBoundedJSON, validateSnapshot } from "./model.js";
import { renderDashboard } from "./render.js";
import { replaySnapshotFixture, snapshotFixture } from "./test-fixture.js";

const clone = value => structuredClone(value);
function clearPartialTQ(snapshot) {
  snapshot.tq.desired_symbols = []; snapshot.tq.known_present = 0;
  for (const row of snapshot.rows) {
    row.tq_membership = { desired: false, provider_present: false, provider_membership_unknown: false };
    row.tape_rate = { status: "unselected", reason: "", trade_coverage: false, one_second: { status: "unselected", reason: "", trades_per_second: null }, five_second: { status: "unselected", reason: "", trades_per_second: null }, timestamp_basis: "", lifecycle_records_observed: false };
    row.spread = { status: "unselected", reason: "", quote_coverage: false, cents: null, basis_points: null, valid_duration_ms: 0, quality: "" };
  }
}

test("P-C11-STATE preserves server order, units, zero, and TQ trust", () => {
  const snapshot = snapshotFixture(20);
  snapshot.sample.id = "9007199254740999";
  snapshot.rows[0].symbol = `<img src=x onerror=alert(1)>`;
  snapshot.tq.desired_symbols[0] = snapshot.rows[0].symbol;
  const model = buildViewModel(snapshot);
  assert.equal(model.current, true);
  assert.equal(model.sampleID, "9007199254740999");
  assert.deepEqual(model.rows.map(row => row.rank), Array.from({ length: 20 }, (_, index) => index + 1));
  assert.equal(model.rows[0].symbol, `<img src=x onerror=alert(1)>`);
  assert.equal(model.rows[0].day, "2.15%");
  assert.equal(model.rows[0].from4am.text, "0.00%");
  assert.equal(model.rows[0].dayRange.text, "50%");
  assert.equal(model.rows[0].activity.text, "80%");
  assert.equal(model.rows[0].tape.primary, "1.4/s");
  assert.equal(model.rows[0].spread.primary, "15.0 bps / 1.50¢");
  assert.match(model.rows[0].tape.detail, /coverage yes; timestamp mixed; lifecycle records observed/);
  assert.match(model.rows[0].spread.detail, /coverage yes; duration 5000 ms; quality reviewed_ordinary/);
	assert.doesNotMatch(model.rows[0].tape.detail, /burst|one-second/);
  assert.match(model.rows[0].tape.detail, /membership desired yes, provider present, unknown no/);
  assert.equal(model.tqKnownPresent, 20); assert.equal(model.tqUnknown, 0); assert.equal(model.tqRetainedBoundHit, false);
  assert.ok(model.diagnostics.some(([name, value]) => name === "operations.deliveries" && value === "10"));
  assert.deepEqual(model.rows.map(row => row.symbol), snapshot.rows.map(row => row.symbol));
  assert.ok(model.rows.every((row, index) => index === 0 || Number.parseFloat(model.rows[index - 1].day) >= Number.parseFloat(row.day)), "visual fixture must be plausible Day-% descending server order");
});

test("P-C11-STATE distinguishes exact empty, fewer, noncurrent, and independent TQ", () => {
  const fewer = buildViewModel(snapshotFixture(3));
  assert.equal(fewer.rows.length, 3);
  const emptySnapshot = snapshotFixture(0);
  const empty = buildViewModel(emptySnapshot);
  assert.equal(empty.current, true); assert.deepEqual(empty.rows, []);

  const degradedSnapshot = snapshotFixture(1);
  degradedSnapshot.ranking.mode = "degraded_bootstrap"; degradedSnapshot.ranking.reason = "incomplete_population";
  clearPartialTQ(degradedSnapshot);
  const degraded = buildViewModel(degradedSnapshot);
  assert.equal(degraded.current, false, "backend-ready degraded output is never presented as qualified current");
  assert.equal(degraded.backendReady, true, "legal degraded bootstrap preserves the server readiness fact");
  assert.equal(degraded.rankingMode, "degraded_bootstrap");
  const degradedDocument = new FakeDocument(); renderDashboard(degradedDocument, { transport: "connected", model: degraded });
  assert.match(degradedDocument.body.textContent, /DEGRADED.*Backendready.*Rankingdegraded_bootstrap · incomplete_population/s);
  assert.doesNotMatch(degradedDocument.body.textContent, /NONCURRENT/);

  const partialSnapshot = snapshotFixture(1);
  partialSnapshot.ranking.mode = "degraded_current"; partialSnapshot.ranking.reason = "qualification_incomplete";
  partialSnapshot.ranking.total_passers = 0;
  partialSnapshot.accounting.qualification.provisional = 0; partialSnapshot.accounting.qualification.unresolved = 1;
  partialSnapshot.accounting.uncertainty.local_invalid = 1;
  assert.throws(() => buildViewModel(partialSnapshot), /partial ranking promoted TQ membership/, "partial mode must reject retained qualified T/Q state");
  clearPartialTQ(partialSnapshot);
  const partial = buildViewModel(partialSnapshot);
  assert.equal(partial.backendReady, true); assert.equal(partial.current, false); assert.equal(partial.partial, true); assert.equal(partial.rowsCurrent, false);
  const partialDocument = new FakeDocument(); renderDashboard(partialDocument, { transport: "connected", model: partial });
  assert.match(partialDocument.body.textContent, /PARTIAL · CURRENT DATA.*Backendready.*Rankingdegraded_current · qualification_incomplete.*PARTIAL RANKING · current trusted marks ordered by Day % · qualification is not asserted/s);
  assert.match(partialDocument.body.textContent, /Server-ranked top 20 trusted marks by Day %, qualification not asserted/);
  assert.equal(partialDocument.getElementById("scanner-table").dataset.publicationState, "noncurrent");

  const ended = snapshotFixture(1);
  ended.publication.lifecycle = "ended"; ended.publication.lifecycle_reason = "session_end"; ended.status.backend_ready = false; ended.status.readiness_reason = "lifecycle_not_ready";
  assert.equal(buildViewModel(ended).lifecycle, "ended");

  for (const [mode, reason] of [["stale", "qualification_incomplete"], ["suppressed", "global_suppression"], ["unavailable", "no_committed_watermark"]]) {
    const state = snapshotFixture(1); state.status.backend_ready = false; state.status.ranking_current = false; state.status.readiness_reason = mode === "suppressed" ? "suppressed" : "ranking_noncurrent"; state.ranking.mode = mode; state.ranking.reason = reason;
    if (mode === "suppressed") { state.publication.lifecycle = "suppressed"; state.publication.suppression = "restart_required"; }
    if (mode === "unavailable") { state.publication.committed_t = null; state.status.watermark_lag_ms = null; }
    assert.equal(buildViewModel(state).current, false, mode);
  }

  const suppressed = snapshotFixture(0);
  suppressed.publication.lifecycle = "suppressed"; suppressed.publication.lifecycle_reason = "accounting_integrity"; suppressed.publication.suppression = "restart_required";
  suppressed.status.backend_ready = false; suppressed.status.ranking_current = false; suppressed.status.readiness_reason = "suppressed";
  suppressed.ranking.mode = "suppressed"; suppressed.ranking.reason = "global_suppression";
  suppressed.operations.integrity_failure = { category: "support_contradiction", engine_sequence: "10725", candidate_time: "2026-08-08T15:59:58Z", expected_time: "2026-08-08T15:59:58Z", first_field: "support", first_reason: "contradictory_evaluator_support" };
  const suppressedModel = buildViewModel(suppressed), suppressedDocument = new FakeDocument();
  renderDashboard(suppressedDocument, { transport: "connected", model: suppressedModel });
  assert.equal(suppressedModel.transport, "connected"); assert.equal(suppressedModel.rows.length, 0);
  assert.match(suppressedDocument.body.textContent, /Transportconnected.*SCANNER SUPPRESSED · accounting_integrity · restart_required · support_contradiction at engine sequence 10725/s);
  assert.doesNotMatch(suppressedDocument.body.textContent, /FROZEN · DISCONNECTED/);
  const unknownDiagnostic = clone(suppressed); unknownDiagnostic.operations.integrity_failure.category = "provider_guess";
  assert.throws(() => validateSnapshot(unknownDiagnostic), /category is unknown/);

  const fields = snapshotFixture(1); fields.rows[0].from_4am_change = { status: "warming", reason: "rolling_warmup", value_ratio: null }; fields.rows[0].hod_drawdown = { status: "unavailable", reason: "history_incomplete", value_ratio: null }; fields.rows[0].activity = { status: "invalid", reason: "invalid_input", value_ratio: null };
  const fieldModel = buildViewModel(fields); assert.equal(fieldModel.rows[0].from4am.state, "warming"); assert.equal(fieldModel.rows[0].hod.state, "unavailable"); assert.equal(fieldModel.rows[0].activity.state, "invalid");

  const pressure = snapshotFixture(1);
  pressure.status.tq_pressure_mode = "aggregate_only"; pressure.status.tq_shed = true; pressure.tq.pressure_mode = "aggregate_only"; pressure.tq.aggregate_only = true; pressure.tq.shed = true;
  pressure.rows[0].tape_rate = { ...pressure.rows[0].tape_rate, status: "pressure_shed", reason: "pressure", trade_coverage: false, timestamp_basis: "", one_second: { status: "pressure_shed", reason: "pressure", trades_per_second: null }, five_second: { status: "pressure_shed", reason: "pressure", trades_per_second: null } };
  pressure.rows[0].spread = { ...pressure.rows[0].spread, status: "pressure_shed", reason: "pressure", quote_coverage: false, cents: null, basis_points: null, valid_duration_ms: 0, quality: "" };
  const pressureModel = buildViewModel(pressure);
  assert.equal(pressureModel.current, true, "TQ pressure must not change aggregate readiness");
  assert.equal(pressureModel.tqAggregateOnly, true); assert.equal(pressureModel.rows[0].tape.state, "pressure_shed"); assert.equal(pressureModel.rows[0].spread.state, "pressure_shed");
});

test("P-C11-STATE presents connected hydration as bounded warm-up progress", () => {
  const snapshot = snapshotFixture(0);
  snapshot.publication.lifecycle = "hydrating"; snapshot.publication.lifecycle_reason = "fresh_bootstrap";
  snapshot.publication.hydration_fence = { reconciled: false, connection_epoch: "0", through_frame_sequence: "0", marker_ordinal: "0", supported_through: null };
  snapshot.status.backend_ready = false; snapshot.status.ranking_current = false; snapshot.status.readiness_reason = "fence_pending";
  snapshot.recovery.fence_reconciled = false;
  snapshot.recovery.work = { planned: "5517", open: "3378", completed_value: "2121", completed_empty: "18", failed: "0", canceled: "0", fenced: "0" };
  const model = buildViewModel(snapshot);
  assert.equal(model.warming, true); assert.equal(model.finalizing, false); assert.equal(model.hydrationProgress, "2,139 / 5,517 · 38.7%"); assert.equal(model.current, false);
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model });
  assert.match(document.body.textContent, /WARMING · 2,139 \/ 5,517 · 38\.7%/);
  assert.match(document.body.textContent, /Backendwarming · 2,139 \/ 5,517 · 38\.7%/);
  assert.match(document.body.textContent, /Scanner warm-up in progress · 2,139 \/ 5,517 · 38\.7%/);
  assert.doesNotMatch(document.body.textContent, /DISCONNECTED/);

  snapshot.recovery.work = { planned: "5517", open: "0", completed_value: "5490", completed_empty: "25", failed: "1", canceled: "0", fenced: "1" };
  const finalizing = buildViewModel(snapshot); assert.equal(finalizing.finalizing, true); assert.equal(finalizing.hydrationIssues, true);
  const finalDocument = new FakeDocument(); renderDashboard(finalDocument, { transport: "connected", model: finalizing });
  assert.match(finalDocument.body.textContent, /FINALIZING · 5,517 \/ 5,517 · 100\.0% · failed 1 · canceled 0 · fenced 1/);

  const disconnected = new FakeDocument(); renderDashboard(disconnected, { transport: "disconnected", model: buildViewModel(snapshot, "disconnected") });
  assert.match(disconnected.body.textContent, /FROZEN · DISCONNECTED/);
  assert.doesNotMatch(disconnected.body.textContent, /FINALIZING · 5,517/);
});

test("P-C11-STATE rejects every known contradiction and accounting break", () => {
  const contradictions = [
    snapshot => { snapshot.status.process_live = false; },
    snapshot => { snapshot.publication.lifecycle = "ended"; },
    snapshot => { snapshot.status.ranking_current = false; },
    snapshot => { snapshot.status.accounting_valid = false; },
    snapshot => { snapshot.publication.run_mode = "replay"; },
    snapshot => { snapshot.publication.connection_active = false; },
    snapshot => { snapshot.publication.committed_t = null; snapshot.status.watermark_lag_ms = null; },
    snapshot => { snapshot.publication.suppression = "restart_required"; },
    snapshot => { snapshot.publication.aggregate_acknowledged = false; snapshot.publication.aggregate_ack_position = { connection_epoch: "0", frame_sequence: "0", array_index: 0 }; },
    snapshot => { snapshot.publication.hydration_fence = { reconciled: false, connection_epoch: "0", through_frame_sequence: "0", marker_ordinal: "0", supported_through: null }; },
    snapshot => { snapshot.publication.hydration_fence.marker_ordinal = "0"; },
    snapshot => { snapshot.publication.hydration_fence.connection_epoch = "5"; },
  ];
  for (const mutate of contradictions) { const snapshot = snapshotFixture(); mutate(snapshot); assert.throws(() => validateSnapshot(snapshot), /conflict|contradictory/); }

  const accounting = [
    snapshot => { snapshot.accounting.population.valid_prior_close++; },
    snapshot => { snapshot.recovery.work.failed = "1"; },
    snapshot => { snapshot.recovery.rows.rejected = "1"; },
    snapshot => { snapshot.tq.facts.rejected = "1"; },
    snapshot => { snapshot.tq.commands.failed = "1"; },
    snapshot => { snapshot.checkpoint.failed = "1"; },
  ];
  for (const mutate of accounting) { const snapshot = snapshotFixture(); mutate(snapshot); assert.throws(() => validateSnapshot(snapshot), /conflict/); }
});

test("P-C11-STATE accepts additive fields but fails closed on unknown meanings", () => {
  const snapshot = snapshotFixture();
  snapshot.future_root = "x".repeat(10_000);
  snapshot.rows[0].future_nested = { payload: "y".repeat(10_000) };
  snapshot.rows[0].activity.reason = "future_review_reason";
  const model = buildViewModel(snapshot);
  assert.equal(model.rows[0].activity.state, "unknown");
  assert.equal(model.rows[0].activity.text, "—");
  assert.equal(Object.hasOwn(model, "future_root"), false);
  assert.equal(Object.hasOwn(model.rows[0], "future_nested"), false);
});

test("P-C11-STATE rejects contradictory known field and TQ trust tuples", () => {
  const edits = [
    snapshot => { snapshot.rows[0].activity.reason = "invalid_input"; },
    snapshot => { snapshot.rows[0].tape_rate.trade_coverage = false; },
    snapshot => { snapshot.rows[0].tape_rate.timestamp_basis = "fabricated"; },
    snapshot => { snapshot.rows[0].tape_rate.five_second.reason = "pressure"; },
    snapshot => { snapshot.rows[0].spread.quote_coverage = false; },
    snapshot => { snapshot.rows[0].spread.quality = "fabricated"; },
    snapshot => { snapshot.rows[0].spread.valid_duration_ms = 3999; },
    snapshot => { snapshot.rows[0].spread.valid_duration_ms = 5001; },
    snapshot => { snapshot.rows[0].spread.cents = -0.01; },
    snapshot => { snapshot.rows[0].spread.basis_points = -0.01; },
    snapshot => { snapshot.rows[0].tape_rate.one_second.trades_per_second = -1; },
    snapshot => { snapshot.rows[0].tape_rate.five_second.trades_per_second = -1; },
  ];
  for (const edit of edits) { const snapshot = snapshotFixture(); edit(snapshot); assert.throws(() => validateSnapshot(snapshot), /conflict|unknown/); }
  const compatible = snapshotFixture(); compatible.rows[0].tape_rate.reason = "future_tape_reason";
  const compatibleModel = buildViewModel(compatible); assert.equal(compatibleModel.rows[0].tape.state, "unknown"); assert.equal(compatibleModel.rows[0].tape.primary, "—"); assert.equal(compatibleModel.rows[0].tape.secondary, "");
  const compatibleStatus = snapshotFixture(); compatibleStatus.rows[0].tape_rate.status = "future_tape_status";
  const statusModel = buildViewModel(compatibleStatus); assert.equal(statusModel.rows[0].tape.state, "unknown"); assert.equal(statusModel.rows[0].tape.primary, "—");
});

test("P-C11-STATE rejects noncanonical and impossible dates", () => {
  for (const edit of [
    snapshot => { snapshot.sample.sampled_at = "2026-02-30T00:00:00Z"; },
    snapshot => { snapshot.publication.generated_at = "2026-08-08T09:00:00-07:00"; },
    snapshot => { snapshot.status.causal_target = "2026-08-08t16:00:00z"; },
    snapshot => { snapshot.publication.trading_date = "2026-02-30"; },
    snapshot => { snapshot.publication.generated_at = "2026-08-08T24:00:00Z"; },
    snapshot => { snapshot.publication.generated_at = "2026-08-08T16:00:00.0Z"; },
    snapshot => { snapshot.publication.generated_at = "2026-08-08T16:00:00.1200Z"; },
  ]) { const snapshot = snapshotFixture(); edit(snapshot); assert.throws(() => validateSnapshot(snapshot), /RFC3339|identity/); }
});

test("P-C11-STATE operations accounting is diagnostic, not a readiness owner", () => {
  const snapshot = snapshotFixture(); snapshot.operations.sample_accounting_valid = false;
  const model = buildViewModel(snapshot); assert.equal(model.current, true); assert.equal(model.sampleAccountingValid, false);
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model });
  assert.match(document.body.textContent, /Ops sampleaccounting invalid/);
});

test("P-C11-STATE rejects malformed row/value/null boundaries", () => {
  const edits = [
    snapshot => { snapshot.schema_version = "scanner.snapshot.v2"; },
    snapshot => { delete snapshot.status.backend_ready; },
    snapshot => { snapshot.rows[0].rank = 2; },
    snapshot => { snapshot.rows.push(clone(snapshot.rows[0])); snapshot.rows[1].rank = 2; },
    snapshot => { snapshot.rows[0].symbol = ""; },
    snapshot => { snapshot.rows[0].last_usd = Infinity; },
    snapshot => { snapshot.rows[0].activity.value_ratio = null; },
    snapshot => { snapshot.rows[0].spread.cents = null; },
    snapshot => { snapshot.sample.id = "01"; },
    snapshot => { snapshot.sample.id = "0"; },
  ];
  for (const edit of edits) { const snapshot = snapshotFixture(); edit(snapshot); assert.throws(() => validateSnapshot(snapshot)); }
  const tooMany = snapshotFixture(20); tooMany.rows.push(clone(tooMany.rows[19])); tooMany.rows[20].rank = 21; assert.throws(() => validateSnapshot(tooMany), /at most 20/);
});

test("P-C11-STATE enforces the streamed 1 MiB response cap", async () => {
  const oversized = new Response(`{"x":"${"a".repeat((1 << 20) + 1)}"}`);
  await assert.rejects(readBoundedJSON(oversized), /exceeds 1 MiB/);
  const dishonest = new Response(`{"x":"${"b".repeat((1 << 20) + 1)}"}`, { headers: { "content-length": "1" } });
  await assert.rejects(readBoundedJSON(dishonest), /exceeds 1 MiB/);
  const declared = new Response("{}", { headers: { "content-length": String((1 << 20) + 1) } });
  await assert.rejects(readBoundedJSON(declared), /exceeds 1 MiB/);
});

test("P-C11-STATE default poller fetch preserves the browser global receiver", async () => {
  const originalFetch = globalThis.fetch;
  const updates = [];
  let receiver = null;
  globalThis.fetch = function () {
    receiver = this;
    return Promise.resolve(new Response(JSON.stringify(snapshotFixture()), { status: 200 }));
  };
  let controller;
  try {
    controller = new PollController({ url: "http://127.0.0.1/api/v1/snapshot", onUpdate: update => updates.push(update) });
  } finally {
    globalThis.fetch = originalFetch;
  }
  await controller.tick();
  assert.equal(receiver, globalThis);
  assert.equal(updates.at(-1).transport, "connected");
  assert.equal(updates.at(-1).model.current, true);
});

test("P-C11-STATE poller marks skipped hung refresh, freezes, and reconnects", async () => {
  const updates = []; let resolveFirst;
  const deferred = new Promise(resolve => { resolveFirst = resolve; });
  const controller = new PollController({ url: "http://127.0.0.1/api/v1/snapshot", requestTimeoutMilliseconds: 30, fetchImpl: () => deferred, onUpdate: update => updates.push(update) });
  const first = controller.tick();
  await controller.tick();
  assert.equal(updates.at(-1).transport, "refresh_delayed"); assert.equal(updates.at(-1).model, null);
  resolveFirst(new Response(JSON.stringify(snapshotFixture()), { status: 200 })); await first;
  assert.equal(updates.at(-1).transport, "connected"); assert.equal(updates.at(-1).model.current, true);

  const samePublication = snapshotFixture(); samePublication.sample.id = "11"; samePublication.sample.sampled_at = "2026-08-08T16:00:01Z";
  controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(samePublication), { status: 200 })); await controller.tick();
  assert.equal(updates.at(-1).model.publicationID, "20"); assert.equal(updates.at(-1).model.sampleID, "11");
  const newPublication = snapshotFixture(); newPublication.sample.id = "12"; newPublication.publication.id = "21"; newPublication.rows[0].symbol = "REPLACED"; newPublication.tq.desired_symbols[0] = "REPLACED";
  controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(newPublication), { status: 200 })); await controller.tick();
  assert.equal(updates.at(-1).model.publicationID, "21"); assert.deepEqual(updates.at(-1).model.rows.map(row => row.symbol), ["REPLACED"]);

  controller.fetchImpl = (_url, options) => new Promise((_resolve, reject) => options.signal.addEventListener("abort", () => reject(new Error("aborted")), { once: true }));
  await controller.tick();
  assert.equal(updates.at(-1).transport, "disconnected"); assert.equal(updates.at(-1).model.current, false); assert.equal(updates.at(-1).model.sampleID, "12");
  controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(snapshotFixture()), { status: 200 }));
  await controller.tick(); assert.equal(updates.at(-1).transport, "connected"); assert.equal(updates.at(-1).model.current, true);
});

test("P-C11-STATE renderer never interprets API strings as HTML", async () => {
  const source = await readFile(new URL("./render.js", import.meta.url), "utf8");
  assert.doesNotMatch(source, /innerHTML|outerHTML|insertAdjacentHTML|document\.write/);
  assert.match(source, /textContent/);
});

class FakeNode {
  constructor(tag, document = null) { this.tagName = tag.toUpperCase(); this.ownerDocument = document; this.children = []; this.dataset = {}; this.attributes = {}; this.hidden = false; this.open = false; this.id = ""; this.className = ""; this._text = ""; }
  set textContent(value) { this._text = String(value); this.children = []; }
  get textContent() { return this._text + this.children.map(child => child.textContent).join(""); }
  append(...nodes) { this.children.push(...nodes); }
  replaceChildren(...nodes) { this._text = ""; this.children = nodes; }
  setAttribute(name, value) { this.attributes[name] = String(value); }
  querySelector(selector) { const match = /^\[data-focus-key="(.*)"\]$/.exec(selector); return match ? find(this, node => node.dataset.focusKey === match[1])[0] || null : null; }
  focus() { if (this.ownerDocument) this.ownerDocument.activeElement = this; }
}
class FakeDocument {
  constructor() { this.body = new FakeNode("body", this); this.activeElement = null; }
  createElement(tag) { return new FakeNode(tag, this); }
  createDocumentFragment() { return new FakeNode("fragment", this); }
  getElementById(id) { return find(this.body, node => node.id === id)[0] || null; }
}
function find(node, predicate, result = []) { if (predicate(node)) result.push(node); for (const child of node.children) find(child, predicate, result); return result; }
globalThis.CSS ??= { escape: value => String(value).replace(/["\\]/g, "\\$&") };

test("replay UI MVP renders three changing synthetic-artifact publications", () => {
  const snapshots = [0, 1, 2].map(replaySnapshotFixture);
  [snapshots[1].rows[0], snapshots[1].rows[1]] = [snapshots[1].rows[1], snapshots[1].rows[0]];
  snapshots[1].rows.forEach((row, index) => { row.rank = index + 1; });
  snapshots[1].rows[0].last_usd = 12.5; snapshots[1].rows[0].day_change_ratio = .02;
  snapshots[2].rows[0].symbol = "S03"; snapshots[2].rows[0].last_usd = 15; snapshots[2].rows[0].day_change_ratio = .03; snapshots[2].rows[0].activity.value_ratio = 1;

  const models = snapshots.map(snapshot => buildViewModel(snapshot));
  assert.deepEqual(models.map(model => model.publicationID), ["200", "201", "202"]);
  assert.deepEqual(models.map(model => model.replayLogicalTime), ["2026-08-08T16:00:00Z", "2026-08-08T16:00:01Z", "2026-08-08T16:00:02Z"]);
  assert.deepEqual(models.map(model => model.rows.map(row => row.symbol)), [["S01", "S02"], ["S02", "S01"], ["S03", "S02"]]);
  assert.deepEqual(models.map(model => model.rows[0].last), ["$10", "$12.5", "$15"]);
  assert.ok(models.every(model => model.replay && !model.current && model.rowsCurrent));
  assert.ok(models.every(model => model.rows.every(row => row.tape.state === "unavailable" && row.spread.state === "unavailable")));

  const document = new FakeDocument();
  renderDashboard(document, { transport: "connected", model: models[2] });
  assert.match(document.body.textContent, /HISTORICAL · NONLIVE/);
  assert.match(document.body.textContent, /Replay time2026-08-08T16:00:02Z/);
  assert.equal(find(document.body, node => node.tagName === "TABLE")[0].dataset.publicationState, "current");
  assert.equal(find(document.body, node => node.dataset.palette === "heat")[0].dataset.state, "current");
});

test("P-C11-STATE detached renderer degrades retained rows and commits atomically", () => {
  const document = new FakeDocument();
  const hostile = snapshotFixture(); hostile.rows[0].symbol = "<script>owned()</script>"; hostile.tq.desired_symbols[0] = hostile.rows[0].symbol;
  renderDashboard(document, { transport: "connected", model: buildViewModel(hostile) });
  const committed = document.body.children[0];
  assert.match(document.body.textContent, /<script>owned\(\)<\/script>/);
  assert.equal(find(document.body, node => node.tagName === "TABLE")[0].dataset.publicationState, "current");
  assert.equal(find(document.body, node => node.tagName === "TD")[0].dataset.state, "current");
  const renderedCells = find(document.body, node => node.tagName === "TBODY")[0].children[0].children;
  assert.deepEqual(renderedCells.slice(3, 6).map(node => node.dataset.palette), [undefined, undefined, undefined], "return and drawdown cells received a palette");
  assert.deepEqual(renderedCells.slice(6, 9).map(node => node.dataset.palette), ["range", "range", "range"]);
  assert.deepEqual(renderedCells.slice(6, 9).map(node => node.dataset.rangePosition), ["50", "75", "25"]);
  assert.deepEqual(renderedCells.slice(6, 9).map(node => [node.dataset.rangeRedWeight, node.dataset.rangeGreenWeight]), [["0%", "0%"], ["0%", "50%"], ["50%", "0%"]]);
  assert.deepEqual(renderedCells.slice(9, 12).map(node => node.dataset.palette), ["heat", "heat", "spread"]);
  assert.deepEqual(renderedCells.slice(9, 11).map(node => node.dataset.heatWeight), ["80%", "4.67%"]);
  assert.match(document.body.textContent, /60 MIN Range %30 MIN Range %/);

  assert.throws(() => renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixture()) }, { beforeCommit: () => { throw new Error("injected render failure"); } }), /injected/);
  assert.equal(document.body.children[0], committed, "failure replaced the prior committed DOM");

  renderDashboard(document, { transport: "disconnected", model: buildViewModel(hostile, "disconnected") });
  assert.equal(find(document.body, node => node.tagName === "TABLE")[0].dataset.publicationState, "noncurrent");
  assert.ok(find(document.body, node => node.tagName === "TD").every(node => node.dataset.state === "retained"));
  assert.match(document.body.textContent, /FROZEN · DISCONNECTED/);
});

test("P-C11-VISUAL polling preserves disclosure, keyed focus, and live announcer", () => {
  const document = new FakeDocument(), first = buildViewModel(snapshotFixture());
  renderDashboard(document, { transport: "connected", model: first });
  const announcer = document.getElementById("announcer"), details = document.getElementById("diagnostics"); details.open = true;
  const tape = find(document.body, node => node.dataset.focusKey?.endsWith(":tape"))[0]; tape.focus(); const key = tape.dataset.focusKey;
  const secondSnapshot = snapshotFixture(2); secondSnapshot.sample.id = "11";
  [secondSnapshot.rows[0], secondSnapshot.rows[1]] = [secondSnapshot.rows[1], secondSnapshot.rows[0]];
  secondSnapshot.rows[0].rank = 1; secondSnapshot.rows[1].rank = 2;
  secondSnapshot.tq.desired_symbols = secondSnapshot.rows.map(row => row.symbol);
  renderDashboard(document, { transport: "connected", model: buildViewModel(secondSnapshot) });
  assert.equal(document.getElementById("announcer"), announcer, "live region was recreated");
  assert.match(announcer.textContent, /publication 20/);
  assert.equal(document.getElementById("diagnostics").open, true);
  assert.equal(document.activeElement.dataset.focusKey, key);

  const summary = find(document.body, node => node.dataset.focusKey === "diagnostics:summary")[0]; summary.focus();
  renderDashboard(document, { transport: "connected", model: buildViewModel(secondSnapshot) });
  assert.equal(document.activeElement.dataset.focusKey, "diagnostics:summary");

  const missing = snapshotFixture(); missing.rows[0].symbol = "DIFFERENT"; missing.tq.desired_symbols[0] = "DIFFERENT";
  const oldTape = find(document.body, node => node.dataset.focusKey?.endsWith(":tape"))[0]; oldTape.focus();
  renderDashboard(document, { transport: "connected", model: buildViewModel(missing) });
  assert.equal(document.activeElement.dataset.focusKey, "diagnostics:summary", "missing row did not use the safe disclosure fallback");
});

test("P-C11-STATE noncurrent qualified empty never claims exact-current empty", () => {
  const document = new FakeDocument(), snapshot = snapshotFixture(0);
  snapshot.status.backend_ready = false; snapshot.status.readiness_reason = "watermark_stale";
  const model = buildViewModel(snapshot);
  renderDashboard(document, { transport: "connected", model });
  assert.doesNotMatch(document.body.textContent, /No symbols currently qualify/);
  assert.match(document.body.textContent, /retained noncurrent publication/);
  renderDashboard(document, { transport: "refresh_delayed", model: buildViewModel(snapshotFixture(), "refresh_delayed") });
  assert.match(document.body.textContent, /REFRESH DELAYED/);
  assert.equal(find(document.body, node => node.tagName === "TABLE")[0].dataset.publicationState, "noncurrent");
});

test("P-C11-STATE renderer failure does not acknowledge an unrendered snapshot", async () => {
  const updates = [];
  const controller = new PollController({ url: "http://127.0.0.1/api/v1/snapshot", fetchImpl: () => Promise.resolve(new Response(JSON.stringify(snapshotFixture()), { status: 200 })), onUpdate: update => { updates.push(update); if (update.kind === "snapshot") throw new Error("render failed"); } });
  await controller.tick();
  assert.equal(controller.lastSnapshot, null);
  assert.equal(updates.at(-1).kind, "error");
  assert.equal(updates.at(-1).model, null);
});
