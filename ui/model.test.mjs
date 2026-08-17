import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { activityColor, buildViewModel, dayRangeColor, floatCyanIntensity, formatShares, interpolateRGBGradient, moveColor, PollController, readBoundedJSON, spreadColor, tapeColor, validateSnapshot, volumeTurnoverIntensity } from "./model.js";
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

function recoverySnapshot({ active = false, connected = false, acknowledged = false, generation = "1", open = "0", completedValue = "2", completedEmpty = "0", canceled = "0" } = {}) {
  const snapshot = snapshotFixtureV2(0); clearTQ(snapshot);
  snapshot.publication.lifecycle = "recovering"; snapshot.publication.lifecycle_reason = "aggregate_epoch_lost"; snapshot.publication.suppression = "";
  snapshot.publication.connection_epoch = connected ? "3" : "2"; snapshot.publication.connection_active = connected; snapshot.publication.aggregate_acknowledged = acknowledged;
  snapshot.publication.aggregate_ack_position = acknowledged ? { connection_epoch: snapshot.publication.connection_epoch, frame_sequence: "3", array_index: 0 } : { connection_epoch: "0", frame_sequence: "0", array_index: 0 };
  snapshot.publication.hydration_fence = { reconciled: false, connection_epoch: "0", through_frame_sequence: "0", marker_ordinal: "0", supported_through: null };
  snapshot.status.backend_ready = false; snapshot.status.readiness_reason = "lifecycle_not_ready"; snapshot.status.ranking_current = false;
  snapshot.ranking.mode = "unavailable"; snapshot.ranking.reason = "no_committed_watermark";
  snapshot.recovery = { generation_active: active, purpose: active ? "gap_recovery" : "fresh_bootstrap", generation, start: "2026-08-08T15:58:00Z", end: "2026-08-08T15:59:00Z", supported_through: "2026-08-08T15:59:58Z", fence_reconciled: false, policy_waiting: false,
    work: { planned: active ? "5522" : "2", open, completed_value: completedValue, completed_empty: completedEmpty, failed: "0", canceled, fenced: "0" },
    rows: { consumed: "0", inserted: "0", duplicate: "0", conflict_or_withdrawal: "0", rejected: "0", fenced: "0", integrity: "0" } };
  return snapshot;
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

test("P-MVP-UI maps absolute share-volume turnover to cyan intensity and keeps raw Volume", () => {
  const floatShares = 10_000_000;
  const cases = [
    [1_000_000, 0], [2_499_999, 0], [2_500_000, .2], [4_999_999, .2],
    [5_000_000, .4], [9_999_999, .4], [10_000_000, .6], [19_999_999, .6],
    [20_000_000, .8], [49_999_999, .8], [50_000_000, 1], [100_000_000, 1],
  ];
  for (const [volume, intensity] of cases) assert.equal(volumeTurnoverIntensity(volume, floatShares), intensity, `${volume} / ${floatShares}`);
  assert.equal(volumeTurnoverIntensity(50_000_000, 0), null);
  assert.equal(volumeTurnoverIntensity(50_000_000, -1), null);
  assert.equal(volumeTurnoverIntensity(Number.NaN, floatShares), null);

  const snapshot = snapshotFixtureV2(cases.length);
  snapshot.rows.forEach((row, index) => {
    row.float = { status: "current", reason: "", value_shares: floatShares, percent_ratio: null, provider: "massive-stocks-float-experimental", effective_date: "2026-08-07", retrieved_at: "2026-08-08T15:55:00Z", provenance: "fresh" };
    row.volume = { status: "current", reason: "", value_shares: cases[index][0] };
  });
  const rows = buildViewModel(snapshot).rows;
  assert.deepEqual(rows.map(row => row.volume.colorIntensity), cases.map(([, intensity]) => intensity));
  assert.equal(rows[2].volume.text, "2.5M");
  assert.equal(rows[10].volume.text, "50M");
  assert.notEqual(rows[10].volume.text, "5x");

  const invalidFloats = [
    { status: "unavailable", reason: "not_available", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" },
    { status: "invalid", reason: "invalid_provenance", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" },
    { status: "stale", reason: "cached_fallback", value_shares: floatShares, percent_ratio: null, provider: "massive-stocks-float-experimental", effective_date: "2026-08-07", retrieved_at: "2026-08-08T15:55:00Z", provenance: "cache" },
  ];
  for (const float of invalidFloats) {
    const data = snapshotFixtureV2(); data.rows[0].volume.value_shares = 50_000_000; data.rows[0].float = float;
    const row = buildViewModel(data).rows[0];
    assert.equal(row.volume.colorIntensity, null, `${float.status} Float must not color Volume`);
    assert.equal(row.volume.text, "50M");
  }
});

test("P-MVP-UI maps Float to an absolute cyan intensity scale", () => {
  const values = [1_000_000, 2_000_000, 5_000_000, 10_000_000, 20_000_000, 50_000_000, 100_000_000];
  const expected = [1, .85, .65, .45, .2, 0, 0];
  const current = value_shares => ({ status: "current", reason: "", value_shares });
  assert.deepEqual(values.map(value_shares => floatCyanIntensity(current(value_shares))), expected);
  assert.equal(floatCyanIntensity(current(1_999_999)), 1);
  assert.equal(floatCyanIntensity({ status: "stale", reason: "cached_fallback", value_shares: 1_000_000 }), null);
  assert.equal(floatCyanIntensity({ status: "invalid", reason: "invalid_provenance", value_shares: null }), null);
  assert.equal(floatCyanIntensity({ status: "unavailable", reason: "not_available", value_shares: null }), null);
  assert.equal(floatCyanIntensity({ status: "current", reason: "", value_shares: 0 }), null);
  assert.equal(floatCyanIntensity(null), null);

  const snapshot = snapshotFixtureV2(values.length + 1);
  values.forEach((value_shares, index) => { snapshot.rows[index].float = { ...snapshot.rows[index].float, status: "current", reason: "", value_shares, provenance: "fresh" }; });
  snapshot.rows.at(-1).float = { status: "unavailable", reason: "not_available", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" };
  assert.deepEqual(buildViewModel(snapshot).rows.map(row => row.float.cyanIntensity), [...expected, null]);

  const single = snapshotFixtureV2(); single.rows[0].float = { ...single.rows[0].float, value_shares: 5_000_000 };
  const peers = snapshotFixtureV2(2); peers.rows[0].float = { ...peers.rows[0].float, value_shares: 5_000_000 }; peers.rows[1].float = { ...peers.rows[1].float, status: "current", reason: "", provenance: "fresh", value_shares: 1_000_000 };
  assert.equal(buildViewModel(single).rows[0].float.cyanIntensity, .65);
  assert.equal(buildViewModel(peers).rows[0].float.cyanIntensity, .65);
});

test("P-MVP-UI normalizes DAY % by displayed value rather than rank", () => {
  const snapshot = snapshotFixtureV2(20);
  snapshot.rows.forEach((row, index) => { row.day_change_ratio = .2 - index * .1 / 18; });
  snapshot.rows[0].day_change_ratio = 1;
  snapshot.rows[1].day_change_ratio = .2;
  snapshot.rows[19].day_change_ratio = .1;
  const model = buildViewModel(snapshot);
  assert.deepEqual(model.rows.map(row => row.symbol), snapshot.rows.map(row => row.symbol));
  assert.equal(model.rows[0].dayColor, 1);
  assert.ok(Math.abs(model.rows[1].dayColor - 1 / 9) < 1e-12, `second row color ${model.rows[1].dayColor}`);
  assert.equal(model.rows[19].dayColor, 0);

  const fewer = snapshotFixtureV2(3);
  [fewer.rows[0].day_change_ratio, fewer.rows[1].day_change_ratio, fewer.rows[2].day_change_ratio] = [.3, .2, .1];
  const fewerColors = buildViewModel(fewer).rows.map(row => row.dayColor);
  assert.equal(fewerColors[0], 1); assert.ok(Math.abs(fewerColors[1] - .5) < 1e-12); assert.equal(fewerColors[2], 0);

  const tied = snapshotFixtureV2(4);
  tied.rows.forEach(row => { row.day_change_ratio = .25; });
  assert.deepEqual(buildViewModel(tied).rows.map(row => row.dayColor), [.5, .5, .5, .5]);
  assert.equal(buildViewModel(snapshotFixtureV2()).rows[0].dayColor, .5);
  assert.deepEqual(buildViewModel(snapshotFixtureV2(0)).rows, []);
});

test("P-MVP-UI normalizes FROM OPEN % by eligible displayed value rather than rank or zero", () => {
  const snapshot = snapshotFixtureV2(3);
  [snapshot.rows[0].from_open_change.value_ratio, snapshot.rows[1].from_open_change.value_ratio, snapshot.rows[2].from_open_change.value_ratio] = [1, .2, -.1];
  const model = buildViewModel(snapshot);
  assert.deepEqual(model.rows.map(row => row.symbol), snapshot.rows.map(row => row.symbol));
  assert.deepEqual(model.rows.map(row => row.fromOpen.text), ["+100.00%", "+20.00%", "-10.00%"]);
  assert.equal(model.rows[0].fromOpen.colorPosition, 1);
  assert.ok(Math.abs(model.rows[1].fromOpen.colorPosition - 3 / 11) < 1e-12, `middle From Open color ${model.rows[1].fromOpen.colorPosition}`);
  assert.equal(model.rows[2].fromOpen.colorPosition, 0);
  assert.notEqual(model.rows[1].fromOpen.colorPosition, .5, "middle value was assigned by rank");

  const fewer = snapshotFixtureV2(3);
  fewer.rows.forEach((row, index) => { row.from_open_change.value_ratio = [.3, .2, .1][index]; });
  const fewerColors = buildViewModel(fewer).rows.map(row => row.fromOpen.colorPosition);
  assert.equal(fewerColors[0], 1); assert.ok(Math.abs(fewerColors[1] - .5) < 1e-12); assert.equal(fewerColors[2], 0);

  const singleton = snapshotFixtureV2(1); singleton.rows[0].from_open_change.value_ratio = .4;
  assert.deepEqual(buildViewModel(singleton).rows.map(row => row.fromOpen.colorPosition), [.5]);
  const equal = snapshotFixtureV2(3); equal.rows.forEach(row => { row.from_open_change.value_ratio = .2; });
  assert.deepEqual(buildViewModel(equal).rows.map(row => row.fromOpen.colorPosition), [.5, .5, .5]);
  const negative = snapshotFixtureV2(3);
  negative.rows.forEach((row, index) => { row.from_open_change.value_ratio = [-.05, -.1, -.2][index]; });
  const negativeColors = buildViewModel(negative).rows.map(row => row.fromOpen.colorPosition);
  assert.equal(negativeColors[0], 1); assert.ok(Math.abs(negativeColors[1] - 2 / 3) < 1e-12); assert.equal(negativeColors[2], 0);
  const tied = snapshotFixtureV2(4);
  tied.rows.forEach((row, index) => { row.from_open_change.value_ratio = [.5, .2, .2, .1][index]; });
  assert.deepEqual(buildViewModel(tied).rows.map(row => row.fromOpen.colorPosition), [1, .25, .25, 0]);
  assert.deepEqual(buildViewModel(snapshotFixtureV2(0)).rows, []);
});

test("P-MVP-UI interpolates Day Range through exact RGB control points", () => {
  const anchors = [[0, "#FC0000"], [.25, "#FF5A5A"], [.5, "#8F9AA3"], [.75, "#73FF63"], [1, "#2CFF05"]];
  for (const [value, color] of anchors) assert.equal(dayRangeColor(value), color);
  assert.deepEqual([dayRangeColor(.1), dayRangeColor(.37), dayRangeColor(.63), dayRangeColor(.92)], ["#FD2424", "#C9797D", "#80CF82", "#43FF23"]);
  assert.notEqual(dayRangeColor(.24), dayRangeColor(.25)); assert.notEqual(dayRangeColor(.26), dayRangeColor(.25)); assert.notEqual(dayRangeColor(.24), dayRangeColor(.26));
  assert.equal(dayRangeColor(-.1), "#FC0000"); assert.equal(dayRangeColor(1.1), "#2CFF05");
  const stops = [[0, "#000000"], [1, "#FFFFFF"]];
  assert.equal(interpolateRGBGradient(.5, stops), "#808080");
});

test("P-MVP-UI interpolates Activity through exact RGB control points without overstating ordinary percentiles", () => {
  const anchors = [[0, "#8F9AA3"], [.5, "#8F9AA3"], [.75, "#C87932"], [.9, "#E98212"], [.97, "#F98A05"], [1, "#FF8A00"]];
  for (const [value, color] of anchors) assert.equal(activityColor(value), color);
  assert.deepEqual([activityColor(.6), activityColor(.82), activityColor(.94), activityColor(.99)], ["#A68D76", "#D77D23", "#F2870B", "#FD8A02"]);
  assert.deepEqual([activityColor(0), activityColor(.2), activityColor(.4), activityColor(.5)], ["#8F9AA3", "#8F9AA3", "#8F9AA3", "#8F9AA3"]);
  assert.notEqual(activityColor(.6), activityColor(.75)); assert.notEqual(activityColor(.82), activityColor(.9)); assert.notEqual(activityColor(.94), activityColor(.97)); assert.notEqual(activityColor(.99), activityColor(1));
  assert.equal(activityColor(-.1), "#8F9AA3"); assert.equal(activityColor(1.1), "#FF8A00");
});

test("P-MVP-UI interpolates Move 30s continuously through signed RGB control points", () => {
  const anchors = [[-.07, "#FC0000"], [-.05, "#EF3030"], [-.02, "#C46B6B"], [0, "#8F9AA3"], [.02, "#70B873"], [.05, "#45E532"], [.07, "#2CFF05"]];
  for (const [value, color] of anchors) assert.equal(moveColor(value), color);
  assert.deepEqual([moveColor(-.06), moveColor(-.03), moveColor(.01), moveColor(.035), moveColor(.06)], ["#F61818", "#D25757", "#80A98B", "#5BCF53", "#39F21C"]);
  assert.notEqual(moveColor(-.051), moveColor(-.05)); assert.notEqual(moveColor(-.049), moveColor(-.05));
  assert.notEqual(moveColor(.049), moveColor(.05)); assert.notEqual(moveColor(.051), moveColor(.05));
  assert.equal(moveColor(-.08), "#FC0000"); assert.equal(moveColor(.08), "#2CFF05");
});

test("P-MVP-UI interpolates Tape 5s through exact absolute-rate RGB control points", () => {
  const anchors = [[0, "#8F9AA3"], [50, "#8F9AA3"], [100, "#B8793E"], [250, "#E98212"], [500, "#FF8A00"]];
  for (const [value, color] of anchors) assert.equal(tapeColor(value), color);
  assert.deepEqual([tapeColor(25), tapeColor(75), tapeColor(175), tapeColor(400)], ["#8F9AA3", "#A48A71", "#D17E28", "#F68707"]);
  assert.notEqual(tapeColor(99), tapeColor(100)); assert.notEqual(tapeColor(100), tapeColor(102));
  assert.notEqual(tapeColor(245), tapeColor(250)); assert.notEqual(tapeColor(250), tapeColor(260));
  assert.equal(tapeColor(501), "#FF8A00"); assert.equal(tapeColor(5000), "#FF8A00");
});

test("P-MVP-UI interpolates Spread through exact absolute-bps RGB control points", () => {
  const anchors = [[0, "#8F9AA3"], [10, "#8F9AA3"], [25, "#B06F6F"], [50, "#D84A4A"], [75, "#EE2525"], [100, "#FC0000"]];
  for (const [value, color] of anchors) assert.equal(spreadColor(value), color);
  assert.deepEqual([spreadColor(5), spreadColor(18), spreadColor(40), spreadColor(65), spreadColor(90)], ["#8F9AA3", "#A18387", "#C85959", "#E53434", "#F60F0F"]);
  assert.notEqual(spreadColor(24), spreadColor(25)); assert.notEqual(spreadColor(25), spreadColor(26));
  assert.notEqual(spreadColor(49), spreadColor(50)); assert.notEqual(spreadColor(50), spreadColor(51));
  assert.equal(spreadColor(101), "#FC0000"); assert.equal(spreadColor(1000), "#FC0000");
});

test("P-MVP-UI excludes non-current FROM OPEN % states from the relative scale", () => {
  const snapshot = snapshotFixtureV2(5);
  snapshot.rows[0].from_open_change.value_ratio = .4;
  snapshot.rows[1].from_open_change = { status: "warming", reason: "before_first_print", value_ratio: null };
  snapshot.rows[2].from_open_change = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  snapshot.rows[3].from_open_change = { status: "invalid", reason: "historical_conflict", value_ratio: null };
  snapshot.rows[4].from_open_change.value_ratio = -.2;
  const model = buildViewModel(snapshot);
  assert.deepEqual(model.rows.map(row => row.fromOpen.colorPosition), [1, null, null, null, 0]);
  assert.deepEqual(model.rows.map(row => row.fromOpen.state), ["current", "warming", "unavailable", "invalid", "current"]);
  assert.deepEqual(model.rows.map(row => row.fromOpen.text), ["+40.00%", "—", "—", "—", "-20.00%"]);

  const empty = snapshotFixtureV2(3);
  empty.rows[0].from_open_change = { status: "warming", reason: "before_first_print", value_ratio: null };
  empty.rows[1].from_open_change = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  empty.rows[2].from_open_change = { status: "invalid", reason: "historical_conflict", value_ratio: null };
  assert.deepEqual(buildViewModel(empty).rows.map(row => row.fromOpen.colorPosition), [null, null, null]);
});

test("P-MVP-UI retains only exact stale Float/Spread tuples and exposes provenance", () => {
  const snapshot = snapshotFixtureV2(); snapshot.rows[0].float = { ...snapshot.rows[0].float, status: "stale", reason: "cached_fallback", provenance: "cache" };
  snapshot.rows[0].spread = { ...snapshot.rows[0].spread, status: "stale", reason: "stale_quote", quote_age_ms: 45 * 60 * 1000 };
  const row = buildViewModel(snapshot).rows[0]; assert.equal(row.float.text, "12M · stale"); assert.equal(row.spread.primary, "15.0 bps / 1.50¢ · stale"); assert.equal(row.spread.textColor, null); assert.match(row.spread.detail, /quote age 2700000 ms/);
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
  const edits = [value => { value.schema_version = "scanner.snapshot.v1"; }, value => { value.rows[0].rank = 2; }, value => { value.rows[0].symbol = ""; }, value => { value.rows[0].activity_30s.value_ratio = null; }, value => { value.rows[0].activity_30s.reason = "future_reason"; }, value => { value.rows[0].volume.value_shares = -1; }, value => { value.rows[0].tape_5s.trades_per_second = null; }, value => { value.accounting.population.covered_population = 0; value.accounting.population.unresolved_population = 2; }, value => { value.recovery.generation_active = true; }, value => { value.tq.pressure_recovery.required_samples = 4; }, value => { value.tq.pressure_recovery.healthy_samples = 6; }, value => { value.tq.pressure_sample.waiting_frames = 1; }];
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

test("P-MVP-TQ-RECOVERY renders the accepted pressure sample and recovery progress", () => {
  const snapshot = snapshotFixtureV2(); snapshot.status.tq_pressure_mode = "taq_degraded"; snapshot.status.tq_shed = true; snapshot.tq.pressure_mode = "taq_degraded"; snapshot.tq.pressure_cause = "oldest_waiting_frame"; snapshot.tq.shed = true;
  snapshot.tq.pressure_sample = { observed: true, waiting_frames: 12, frame_capacity: 32768, waiting_bytes: 4096, byte_capacity: 134217728, oldest_waiting_frame_age_ms: 612, aggregate_watermark_lag_ms: 1000, recovery_healthy: true };
  snapshot.tq.pressure_recovery = { healthy_samples: 3, required_samples: 5 };
  snapshot.rows[0].tape_5s = { ...snapshot.rows[0].tape_5s, status: "pressure_shed", reason: "pressure", trade_coverage: false, trades_per_second: null, timestamp_basis: "" };
  snapshot.rows[0].spread = { status: "pressure_shed", reason: "pressure", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  const model = buildViewModel(snapshot); assert.equal(model.current, true); assert.equal(model.tqOldestWaitingFrameAgeMS, 612); assert.equal(model.tqRecoveryHealthySamples, 3);
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model });
  assert.match(document.body.textContent, /T\/Qtaq_degraded · oldest_waiting_frame · oldest 612 ms · recovery 3\/5 · 0 unknown/);
  const missReset = clone(snapshot); missReset.tq.pressure_recovery.healthy_samples = 0;
  const resetDocument = new FakeDocument(); renderDashboard(resetDocument, { transport: "connected", model: buildViewModel(missReset) });
  assert.match(resetDocument.body.textContent, /oldest 612 ms · recovery 0\/5/);
  const impossibleComplete = clone(snapshot); impossibleComplete.tq.pressure_recovery.healthy_samples = 5;
  assert.throws(() => validateSnapshot(impossibleComplete), /TQ pressure recovery conflict/);
});

test("P-MVP-UI accepts the engine-owned scheduled recovery lifecycle", () => {
  const snapshot = snapshotFixtureV2(0);
  snapshot.publication.lifecycle = "recovering"; snapshot.publication.lifecycle_reason = "scheduled_recovery"; snapshot.publication.suppression = "";
  snapshot.status.backend_ready = false; snapshot.status.readiness_reason = "lifecycle_not_ready"; snapshot.status.ranking_current = false;
  snapshot.ranking.mode = "unavailable"; snapshot.ranking.reason = "no_committed_watermark"; clearTQ(snapshot);
  const model = buildViewModel(snapshot);
  assert.equal(model.lifecycle, "recovering"); assert.equal(model.lifecycleReason, "scheduled_recovery"); assert.equal(model.current, false);
});

test("P-MVP-RECOVERY-OBS distinguishes reconnect, active retry, and final fence without stale progress", () => {
  const reconnecting = buildViewModel(recoverySnapshot());
  assert.equal(reconnecting.phase, "reconnecting"); assert.equal(reconnecting.recoveryGenerationActive, false); assert.equal(reconnecting.recovering, true);

  const active = recoverySnapshot({ active: true, connected: true, acknowledged: true, generation: "3", open: "423", completedValue: "909", completedEmpty: "4190" });
  const activeModel = buildViewModel(active);
  assert.equal(activeModel.phase, "recovering"); assert.equal(activeModel.hydrationProgress, "5,099 / 5,522 · 92.3%");

  active.recovery.work.open = "0"; active.recovery.work.completed_value = "1332";
  const finalizing = buildViewModel(active); assert.equal(finalizing.phase, "finalizing_recovery"); assert.equal(finalizing.finalizing, true);

  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: activeModel });
  assert.match(document.body.textContent, /RECOVERING.*generation 3.*5,099 \/ 5,522.*Ranking remains noncurrent/s);
  const reconnectDocument = new FakeDocument(); renderDashboard(reconnectDocument, { transport: "connected", model: reconnecting });
  assert.match(reconnectDocument.body.textContent, /RECONNECTING/); assert.doesNotMatch(reconnectDocument.body.textContent, /2 \/ 2/);

  const priorV2 = recoverySnapshot({ connected: true, acknowledged: true }); delete priorV2.recovery.generation_active;
  const compatible = buildViewModel(priorV2); assert.equal(compatible.phase, "preparing_recovery"); assert.equal(compatible.recoveryGenerationActive, false);
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
  constructor(tag, document = null) { this.tagName = tag.toUpperCase(); this.ownerDocument = document; this.children = []; this.dataset = {}; this.attributes = {}; this.style = {}; this.hidden = false; this.open = false; this.id = ""; this.className = ""; this._text = ""; }
  set textContent(value) { this._text = String(value); this.children = []; } get textContent() { return this._text + this.children.map(child => child.textContent).join(""); } append(...nodes) { this.children.push(...nodes); } replaceChildren(...nodes) { this._text = ""; this.children = nodes; } setAttribute(name, value) { this.attributes[name] = String(value); }
  querySelector(selector) { const match = /^\[data-focus-key="(.*)"\]$/.exec(selector); return match ? find(this, node => node.dataset.focusKey === match[1])[0] || null : null; } focus() { if (this.ownerDocument) this.ownerDocument.activeElement = this; }
}
class FakeDocument { constructor() { this.body = new FakeNode("body", this); this.activeElement = null; } createElement(tag) { return new FakeNode(tag, this); } getElementById(id) { return find(this.body, node => node.id === id)[0] || null; } }
function find(node, predicate, result = []) { if (predicate(node)) result.push(node); for (const child of node.children) find(child, predicate, result); return result; }
globalThis.CSS ??= { escape: value => String(value).replace(/["\\]/g, "\\$&") };

test("P-MVP-UI renders exact groups/columns safely and atomically with keyed focus", async () => {
  const document = new FakeDocument(), snapshot = snapshotFixtureV2(2); snapshot.rows[0].symbol = "<img src=x onerror=owned()>"; snapshot.tq.desired_symbols[0] = snapshot.rows[0].symbol; renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  assert.equal(buildViewModel(snapshot).diagnostics, undefined);
  assert.equal(document.getElementById("diagnostics"), null);
  assert.doesNotMatch(document.body.textContent, /Operational details/);
  const statusGrid = find(document.body, node => node.tagName === "SECTION" && node.className === "status-grid")[0];
  assert.deepEqual(statusGrid.children.map(item => item.children[0].textContent), ["Backend", "Ranking", "T/Q"]);
  for (const label of ["Transport", "Process", "Ops sample", "Watermark", "Sample"]) assert.doesNotMatch(document.body.textContent, new RegExp(`\\b${label}\\b`, "i"));
  assert.match(document.body.textContent, /CONTEXT.*LOCATION.*CURRENT MOMENTUM.*EXECUTION/s);
  assert.match(document.body.textContent, /CURRENT · 2 ranked/); assert.doesNotMatch(document.body.textContent, /publication \d+/i);
  const headers = find(document.body, node => node.tagName === "TH"); const leafHeaders = headers.filter(node => node.attributes.scope === "col"); assert.deepEqual(leafHeaders.map(node => node.textContent), ["SYMBOL", "FLOAT", "VOLUME", "LAST", "FROM CLOSE %", "FROM OPEN %", "DAY RANGE", "ACTIVITY 30s", "MOVE 30s", "TAPE SPEED", "SPREAD"]); assert.deepEqual(leafHeaders.map(node => node.attributes["data-tooltip"]), ["Ticker symbol for the listed stock.", "Estimated number of publicly tradable shares.", "Total shares traded during the current scanner session.", "Latest trusted price.", "Percent change from the adjusted previous close.", "Percent change from the first eligible session price.", "Current price position between today's low and high.", "Recent share-volume activity compared with the prior five-minute baseline.", "Signed price change over the last 30 seconds.", "Qualified trades per second over the last five seconds.", "Difference between the current bid and ask, shown in cents and basis points."]); assert.ok(headers.every(node => node.textContent !== "Rank")); assert.ok(leafHeaders.every(node => node.tabIndex === undefined && node.title === undefined)); const cells = find(document.body, node => node.tagName === "TD"); assert.ok(cells.every(cell => cell.title === undefined)); assert.equal(find(document.body, node => node.tagName === "TBODY")[0].children[0].children.length, 11);
  const committed = document.body.children[0]; assert.throws(() => renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }, { beforeCommit: () => { throw new Error("render failed"); } })); assert.equal(document.body.children[0], committed);
  const focused = find(document.body, node => node.dataset.focusKey?.endsWith(":spread"))[0]; focused.focus(); const key = focused.dataset.focusKey; [snapshot.rows[0], snapshot.rows[1]] = [snapshot.rows[1], snapshot.rows[0]]; snapshot.rows.forEach((row, index) => { row.rank = index + 1; }); snapshot.tq.desired_symbols = snapshot.rows.map(row => row.symbol); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) }); assert.equal(document.activeElement.dataset.focusKey, key);
  renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }); assert.equal(document.getElementById("diagnostics"), null); assert.doesNotMatch(document.body.textContent, /Operational details/); const source = await readFile(new URL("./render.js", import.meta.url), "utf8"); assert.doesNotMatch(source, /innerHTML|outerHTML|insertAdjacentHTML|document\.write/);
});

test("P-MVP-UI colors only current Move 30s text and leaves its cell neutral", () => {
  const values = [-.07, -.05, -.02, 0, .02, .05, .07];
  const snapshot = snapshotFixtureV2(values.length);
  values.forEach((value, index) => { snapshot.rows[index].move_30s.value_ratio = value; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":move"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), values.map(moveColor));
  assert.deepEqual(cells.map(cell => cell.textContent), ["-7.00%", "-5.00%", "-2.00%", "0.00%", "+2.00%", "+5.00%", "+7.00%"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));

  const missing = snapshotFixtureV2(); missing.rows[0].move_30s = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  const missingDocument = new FakeDocument(); renderDashboard(missingDocument, { transport: "connected", model: buildViewModel(missing) });
  const missingCell = find(missingDocument.body, node => node.dataset.focusKey?.endsWith(":move"))[0];
  assert.equal(missingCell.children[0].style.color, undefined); assert.equal(missingCell.dataset.fieldState, "unavailable"); assert.equal(missingCell.style.background, undefined);
});

test("P-MVP-UI colors only current Tape 5s text and leaves its cell neutral", () => {
  const rates = [0, 50, 100, 250, 500];
  const snapshot = snapshotFixtureV2(rates.length);
  rates.forEach((rate, index) => { snapshot.rows[index].tape_5s.trades_per_second = rate; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":tape"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), rates.map(tapeColor));
  assert.deepEqual(cells.map(cell => cell.textContent), ["0.0/s", "50.0/s", "100.0/s", "250.0/s", "500.0/s"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));

  const unavailable = snapshotFixtureV2(); unavailable.rows[0].tape_5s = { status: "unavailable", reason: "coverage", trade_coverage: false, trades_per_second: null, timestamp_basis: "none", lifecycle_records_observed: false };
  const unavailableDocument = new FakeDocument(); renderDashboard(unavailableDocument, { transport: "connected", model: buildViewModel(unavailable) });
  const unavailableCell = find(unavailableDocument.body, node => node.dataset.focusKey?.endsWith(":tape"))[0];
  assert.equal(unavailableCell.children[0].style.color, undefined); assert.equal(unavailableCell.dataset.fieldState, "unavailable"); assert.equal(unavailableCell.style.background, undefined); assert.equal(unavailableCell.dataset.palette, undefined);
});

test("P-MVP-UI colors only current Spread text and leaves invalid or unavailable cells neutral", () => {
  const bps = [0, 10, 25, 50, 75, 100];
  const snapshot = snapshotFixtureV2(bps.length);
  bps.forEach((value, index) => { snapshot.rows[index].spread.basis_points = value; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":spread"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), bps.map(spreadColor));
  assert.deepEqual(cells.map(cell => cell.textContent), ["0.0 bps / 1.50¢", "10.0 bps / 1.50¢", "25.0 bps / 1.50¢", "50.0 bps / 1.50¢", "75.0 bps / 1.50¢", "100.0 bps / 1.50¢"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));

  const invalid = snapshotFixtureV2(2);
  invalid.rows[0].spread = { status: "invalid", reason: "crossed_quote", quote_coverage: true, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  invalid.rows[1].spread = { status: "unavailable", reason: "coverage", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  const invalidDocument = new FakeDocument(); renderDashboard(invalidDocument, { transport: "connected", model: buildViewModel(invalid) });
  const invalidCells = find(invalidDocument.body, node => node.dataset.focusKey?.endsWith(":spread"));
  assert.deepEqual(invalidCells.map(cell => cell.children[0].style.color), [undefined, undefined]);
  assert.deepEqual(invalidCells.map(cell => cell.dataset.fieldState), ["invalid", "unavailable"]);
  assert.ok(invalidCells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));
});

test("P-MVP-UI renders DAY % with its snapshot-relative continuous position", () => {
  const document = new FakeDocument(), snapshot = snapshotFixtureV2(3);
  [snapshot.rows[0].day_change_ratio, snapshot.rows[1].day_change_ratio, snapshot.rows[2].day_change_ratio] = [1, .2, .1];
  renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const days = find(document.body, node => node.dataset.focusKey?.endsWith(":day"));
  assert.deepEqual(days.map(day => day.dataset.palette), ["day", "day", "day"]);
  assert.deepEqual(days.map(day => day.dataset.dayWeight), ["100%", "11.11%", "0%"]);
  assert.match(days[1].attributes["aria-label"], /relative to the displayed snapshot/);
});

test("P-MVP-UI colors only current Day Range text and leaves missing values neutral", () => {
  const snapshot = snapshotFixtureV2(6); [0, .25, .5, .75, 1, .37].forEach((value, index) => { snapshot.rows[index].day_range_position.value_ratio = value; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":dayrange"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), ["#FC0000", "#FF5A5A", "#8F9AA3", "#73FF63", "#2CFF05", "#C9797D"]);
  assert.ok(cells.every(cell => cell.style.background === undefined));

  const missing = snapshotFixtureV2(); missing.rows[0].day_range_position = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  const missingDocument = new FakeDocument(); renderDashboard(missingDocument, { transport: "connected", model: buildViewModel(missing) });
  const missingCell = find(missingDocument.body, node => node.dataset.focusKey?.endsWith(":dayrange"))[0];
  assert.equal(missingCell.children[0].style.color, undefined); assert.equal(missingCell.dataset.fieldState, "unavailable");
});

test("P-MVP-UI colors only current Activity text and leaves its cell neutral", () => {
  const snapshot = snapshotFixtureV2(6); [0, .4, .6, .82, .94, .99].forEach((value, index) => { snapshot.rows[index].activity_30s.value_ratio = value; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":activity"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), ["#8F9AA3", "#8F9AA3", "#A68D76", "#D77D23", "#F2870B", "#FD8A02"]);
  assert.deepEqual(cells.map(cell => cell.textContent), ["0%", "40%", "60%", "82%", "94%", "99%"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined && cell.dataset.heatWeight === undefined));
  assert.equal(find(document.body, node => node.dataset.focusKey?.endsWith(":tape"))[0].dataset.palette, undefined);

  const missing = snapshotFixtureV2(); missing.rows[0].activity_30s = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  const missingDocument = new FakeDocument(); renderDashboard(missingDocument, { transport: "connected", model: buildViewModel(missing) });
  const missingCell = find(missingDocument.body, node => node.dataset.focusKey?.endsWith(":activity"))[0];
  assert.equal(missingCell.children[0].style.color, undefined); assert.equal(missingCell.dataset.fieldState, "unavailable");
});

test("P-MVP-UI renders FROM OPEN % palette metadata only for eligible values", () => {
  const document = new FakeDocument(), snapshot = snapshotFixtureV2(3);
  [snapshot.rows[0].from_open_change.value_ratio, snapshot.rows[1].from_open_change.value_ratio, snapshot.rows[2].from_open_change.value_ratio] = [1, .2, -.1];
  renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":fromopen"));
  assert.deepEqual(cells.map(cell => cell.dataset.palette), ["from-open", "from-open", "from-open"]);
  assert.deepEqual(cells.map(cell => cell.dataset.fromOpenWeight), ["100%", "27.27%", "0%"]);
  assert.deepEqual(cells.map(cell => cell.textContent), ["+100.00%", "+20.00%", "-10.00%"]);

  const mixed = snapshotFixtureV2(3);
  mixed.rows[0].from_open_change = { status: "warming", reason: "before_first_print", value_ratio: null };
  mixed.rows[1].from_open_change = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  mixed.rows[2].from_open_change = { status: "invalid", reason: "historical_conflict", value_ratio: null };
  const mixedDocument = new FakeDocument(); renderDashboard(mixedDocument, { transport: "connected", model: buildViewModel(mixed) });
  const mixedCells = find(mixedDocument.body, node => node.dataset.focusKey?.endsWith(":fromopen"));
  assert.deepEqual(mixedCells.map(cell => cell.dataset.palette), [undefined, undefined, undefined]);
  assert.deepEqual(mixedCells.map(cell => cell.dataset.fieldState), ["warming", "unavailable", "invalid"]);
});

test("P-MVP-UI renders VOLUME turnover metadata only for current trusted Float", () => {
  const snapshot = snapshotFixtureV2(3);
  snapshot.rows.forEach((row, index) => {
    row.float = { status: "current", reason: "", value_shares: 10_000_000, percent_ratio: null, provider: "massive-stocks-float-experimental", effective_date: "2026-08-07", retrieved_at: "2026-08-08T15:55:00Z", provenance: "fresh" };
    row.volume.value_shares = [2_500_000, 10_000_000, 50_000_000][index];
  });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":volume"));
  assert.deepEqual(cells.map(cell => cell.dataset.palette), ["volume-turnover", "volume-turnover", "volume-turnover"]);
  assert.deepEqual(cells.map(cell => cell.dataset.volumeTurnoverWeight), ["20%", "60%", "100%"]);
  assert.deepEqual(cells.map(cell => cell.textContent), ["2.5M", "10M", "50M"]);

  const unavailable = snapshotFixtureV2(); unavailable.rows[0].volume.value_shares = 50_000_000; unavailable.rows[0].float = { status: "unavailable", reason: "not_available", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" };
  const unavailableDocument = new FakeDocument(); renderDashboard(unavailableDocument, { transport: "connected", model: buildViewModel(unavailable) });
  const unavailableCell = find(unavailableDocument.body, node => node.dataset.focusKey?.endsWith(":volume"))[0];
  assert.equal(unavailableCell.dataset.palette, undefined); assert.equal(unavailableCell.textContent, "50M");
});

test("P-MVP-UI renders Float cyan metadata by absolute threshold and keeps unavailable neutral", () => {
  const values = [1_000_000, 2_000_000, 5_000_000, 10_000_000, 20_000_000, 50_000_000, 100_000_000];
  const snapshot = snapshotFixtureV2(values.length + 1);
  values.forEach((value_shares, index) => { snapshot.rows[index].float = { ...snapshot.rows[index].float, status: "current", reason: "", value_shares, provenance: "fresh" }; });
  snapshot.rows.at(-1).float = { status: "unavailable", reason: "not_available", value_shares: null, percent_ratio: null, provider: "", effective_date: null, retrieved_at: null, provenance: "" };
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":float"));
  assert.deepEqual(cells.map(cell => cell.dataset.palette), ["float", "float", "float", "float", "float", "float", "float", undefined]);
  assert.deepEqual(cells.map(cell => cell.dataset.floatWeight), ["100%", "85%", "65%", "45%", "20%", "0%", "0%", undefined]);
  assert.deepEqual(cells.map(cell => cell.textContent), ["1M", "2M", "5M", "10M", "20M", "50M", "100M", "—"]);
});

test("P-MVP-UI keeps From Open relative metadata subordinate to retained table state", () => {
  const snapshot = snapshotFixtureV2(3);
  snapshot.rows.forEach((row, index) => { row.from_open_change.value_ratio = [1, .2, -.1][index]; });
  snapshot.ranking.mode = "degraded_current"; snapshot.ranking.reason = "qualification_incomplete"; clearTQ(snapshot);
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const table = find(document.body, node => node.tagName === "TABLE")[0];
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":fromopen"));
  assert.equal(table.dataset.publicationState, "noncurrent");
  assert.deepEqual(cells.map(cell => cell.dataset.state), ["retained", "retained", "retained"]);
  assert.deepEqual(cells.map(cell => cell.dataset.fromOpenWeight), ["100%", "27.27%", "0%"]);
});

test("P-MVP-UI assets contain only v2 and final-field representations", async () => {
  const assets = await Promise.all(["app.js", "model.js", "render.js", "index.html"].map(name => readFile(new URL(`./${name}`, import.meta.url), "utf8"))); const all = assets.join("\n"); assert.match(all, /api\/v2\/snapshot/); assert.doesNotMatch(all, /api\/v1\/snapshot|scanner\.snapshot\.v1|from_4am|hod_drawdown|range_30m|range_60m|tape_rate/); assert.match(assets[2], /data-tooltip/); assert.doesNotMatch(assets[2], /\.title\s*=/);
  assert.doesNotMatch(assets[3], /<span>(?:Transport|Process|Ops sample|Watermark|Sample)<\/span>/i);
  assert.doesNotMatch(assets[3], /Operational details/i);
});
