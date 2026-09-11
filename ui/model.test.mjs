import assert from "node:assert/strict";
import { readFile } from "node:fs/promises";
import test from "node:test";
import { activityColor, buildViewModel, dayRangeColor, floatCyanIntensity, formatShares, interpolateRGBGradient, moveColor, MUTED_NEGATIVE_TEXT_COLOR, NEUTRAL_TEXT_COLOR, PollController, RankMovementHistory, readBoundedJSON, signedValueTextColor, spreadColor, tapeColor, validateSnapshot, volumeTurnoverIntensity } from "./model.js";
import { renderDashboard, scannerRowHeight } from "./render.js";
import { snapshotFixtureV2 } from "./test-fixture-v2.js";

const clone = value => structuredClone(value);
test("P-MVP-UI fits rows to measured table space with a 50px cap", () => {
  assert.equal(scannerRowHeight(816, 54, 20), 38.05);
  assert.equal(scannerRowHeight(1116, 54, 20), 50);
  assert.equal(scannerRowHeight(616, 54, 20), 28.05);
  assert.equal(scannerRowHeight(816, 54, 10), 50);
  assert.equal(scannerRowHeight(816, 54, 0), null);
});
function clearTQ(snapshot) {
  snapshot.tq.desired_symbols = []; snapshot.tq.known_present = 0;
  for (const row of snapshot.rows) {
    row.tq_membership = { desired: false, provider_present: false, provider_membership_unknown: false };
    row.tape_5s = { status: "unselected", reason: "", trade_coverage: false, trades_per_second: null, timestamp_basis: "", lifecycle_records_observed: false };
    row.spread = { status: "unselected", reason: "", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  }
}

function recoverySnapshot({ active = false, connected = false, acknowledged = false, lifecycle = "recovering", generation = "1", open = "0", completedValue = "2", completedEmpty = "0", canceled = "0" } = {}) {
  const snapshot = snapshotFixtureV2(0); clearTQ(snapshot);
  snapshot.publication.lifecycle = lifecycle; snapshot.publication.lifecycle_reason = lifecycle === "hydrating" ? "aggregate_acknowledged_at_session_start" : "aggregate_epoch_lost"; snapshot.publication.suppression = "";
  snapshot.publication.connection_epoch = connected ? "3" : "2"; snapshot.publication.connection_active = connected; snapshot.publication.aggregate_acknowledged = acknowledged;
  snapshot.publication.aggregate_ack_position = acknowledged ? { connection_epoch: snapshot.publication.connection_epoch, frame_sequence: "3", array_index: 0 } : { connection_epoch: "0", frame_sequence: "0", array_index: 0 };
  snapshot.publication.hydration_fence = { reconciled: false, connection_epoch: "0", through_frame_sequence: "0", marker_ordinal: "0", supported_through: null };
  snapshot.status.backend_ready = false; snapshot.status.readiness_reason = "lifecycle_not_ready"; snapshot.status.ranking_current = false;
  snapshot.ranking.mode = "unavailable"; snapshot.ranking.reason = "no_committed_watermark";
  snapshot.recovery = { generation_active: active, purpose: lifecycle === "hydrating" ? "fresh_bootstrap" : active ? "gap_recovery" : "fresh_bootstrap", generation, start: "2026-08-08T15:58:00Z", end: "2026-08-08T15:59:00Z", supported_through: "2026-08-08T15:59:58Z", fence_reconciled: false, policy_waiting: false,
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

test("P2 accepts independently unconfirmed T/Q channels without invalidating aggregate ranking", () => {
  const snapshot = snapshotFixtureV2(2);
  snapshot.tq.known_present = 0; snapshot.tq.unknown = 2;
  snapshot.rows[0].tape_5s = { status: "unavailable", reason: "channel_unconfirmed", trade_coverage: false, trades_per_second: null, timestamp_basis: "", lifecycle_records_observed: false };
  snapshot.rows[0].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
  snapshot.rows[1].spread = { status: "unavailable", reason: "channel_unconfirmed", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  snapshot.rows[1].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };

  const model = buildViewModel(snapshot);
  assert.equal(model.current, true);
  assert.equal(model.rows[0].tape.state, "unavailable");
  assert.equal(model.rows[0].spread.state, "current");
  assert.equal(model.rows[1].tape.state, "current");
  assert.equal(model.rows[1].spread.state, "unavailable");
});

test("P3 accepts explicit T/Q control-error containment without invalidating aggregate ranking", () => {
  const snapshot = snapshotFixtureV2(1);
  snapshot.rows[0].tape_5s = { status: "unavailable", reason: "control_error", trade_coverage: true, trades_per_second: null, timestamp_basis: "", lifecycle_records_observed: false };
  snapshot.rows[0].spread = { status: "unavailable", reason: "control_error", quote_coverage: true, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };

  const model = buildViewModel(snapshot);
  assert.equal(model.current, true);
  assert.equal(model.rows[0].tape.state, "unavailable");
  assert.equal(model.rows[0].spread.state, "unavailable");
});

function sampledAt(snapshot, seconds) {
  snapshot.sample.id = String(10 + seconds);
  snapshot.sample.sampled_at = new Date(Date.parse("2026-08-08T16:00:00Z") + seconds * 1000).toISOString().replace(".000Z", "Z");
  return snapshot;
}

function moveSymbol(snapshot, symbol, rank) {
  const index = snapshot.rows.findIndex(row => row.symbol === symbol);
  const [row] = snapshot.rows.splice(index, 1); snapshot.rows.splice(rank - 1, 0, row);
  snapshot.rows.forEach((candidate, candidateIndex) => { candidate.rank = candidateIndex + 1; });
  snapshot.tq.desired_symbols = snapshot.rows.map(candidate => candidate.symbol);
  return snapshot;
}

function movement(previousRank, currentRank) {
  const history = new RankMovementHistory();
  history.apply(buildViewModel(sampledAt(snapshotFixtureV2(10), 0)));
  const symbol = `S${String(previousRank).padStart(2, "0")}`;
  const current = moveSymbol(sampledAt(snapshotFixtureV2(10), 60), symbol, currentRank);
  return history.apply(buildViewModel(current)).rows.find(row => row.symbol === symbol).rankMovement;
}

test("P-MVP-UI assigns displayed ranks and computes capped rolling 60-second movement", () => {
  assert.deepEqual(buildViewModel(snapshotFixtureV2(6)).rows.map(row => row.rank), [1, 2, 3, 4, 5, 6]);
  assert.deepEqual(movement(9, 4), { direction: "up", text: "↑5", label: "up 5 ranks from 60 seconds ago" });
  assert.deepEqual(movement(10, 4), { direction: "up", text: "↑5+", label: "up 6 ranks from 60 seconds ago" });
  assert.deepEqual(movement(4, 6), { direction: "down", text: "↓2", label: "down 2 ranks from 60 seconds ago" });
  assert.deepEqual(movement(2, 10), { direction: "down", text: "↓5+", label: "down 8 ranks from 60 seconds ago" });
  assert.deepEqual(movement(7, 7), { direction: "none", text: "", label: "unchanged from 60 seconds ago" });
});

test("P-MVP-UI distinguishes rank-history warmup from a new top-20 entrant", () => {
  const history = new RankMovementHistory();
  const initial = history.apply(buildViewModel(sampledAt(snapshotFixtureV2(10), 0)));
  assert.ok(initial.rows.every(row => row.rankMovement.text === ""), "startup labeled rows as new");
  const warmup = history.apply(buildViewModel(sampledAt(snapshotFixtureV2(10), 59)));
  assert.ok(warmup.rows.every(row => row.rankMovement.text === ""), "sub-60-second history produced movement");
  const entrant = sampledAt(snapshotFixtureV2(10), 60); entrant.rows[3].symbol = "NEW"; entrant.tq.desired_symbols[3] = "NEW";
  const current = history.apply(buildViewModel(entrant));
  assert.deepEqual(current.rows[3].rankMovement, { direction: "up", text: "↑ NEW", label: "new to the top 20 compared with 60 seconds ago" });
});

test("P-MVP-UI applies every new row order immediately and bounds rank history", () => {
  const history = new RankMovementHistory();
  for (let second = 0; second < 200; second++) {
    const snapshot = sampledAt(snapshotFixtureV2(10), second);
    if (second % 2 === 1) moveSymbol(snapshot, "S10", 1);
    const model = history.apply(buildViewModel(snapshot));
    assert.equal(model.rows[0].symbol, second % 2 === 1 ? "S10" : "S01", "rank order was smoothed or delayed");
  }
  assert.equal(history.size, 70, "rank history did not remain at its fixed bound");
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
  assert.deepEqual(buildViewModel(tied).rows.map(row => row.dayColor), [1, 1, 1, 1]);
  assert.equal(buildViewModel(snapshotFixtureV2()).rows[0].dayColor, 1);
  assert.deepEqual(buildViewModel(snapshotFixtureV2(0)).rows, []);
});

test("P-MVP-UI normalizes FROM OPEN % by eligible displayed value rather than rank or zero", () => {
  const snapshot = snapshotFixtureV2(3);
  [snapshot.rows[0].from_open_change.value_ratio, snapshot.rows[1].from_open_change.value_ratio, snapshot.rows[2].from_open_change.value_ratio] = [1, .2, -.1];
  const model = buildViewModel(snapshot);
  assert.deepEqual(model.rows.map(row => row.symbol), snapshot.rows.map(row => row.symbol));
  assert.deepEqual(model.rows.map(row => row.fromOpen.text), ["+100.00%", "+20.00%", "-10.00%"]);
  assert.equal(model.rows[0].fromOpen.colorPosition, 1);
  assert.equal(model.rows[1].fromOpen.colorPosition, 0);
  assert.equal(model.rows[2].fromOpen.colorPosition, null);
  assert.equal(model.rows[1].fromOpen.colorPosition, 0, "positive scale used the positive subset");

  const fewer = snapshotFixtureV2(3);
  fewer.rows.forEach((row, index) => { row.from_open_change.value_ratio = [.3, .2, .1][index]; });
  const fewerColors = buildViewModel(fewer).rows.map(row => row.fromOpen.colorPosition);
  assert.equal(fewerColors[0], 1); assert.ok(Math.abs(fewerColors[1] - .5) < 1e-12); assert.equal(fewerColors[2], 0);

  const singleton = snapshotFixtureV2(1); singleton.rows[0].from_open_change.value_ratio = .4;
  assert.deepEqual(buildViewModel(singleton).rows.map(row => row.fromOpen.colorPosition), [1]);
  const equal = snapshotFixtureV2(3); equal.rows.forEach(row => { row.from_open_change.value_ratio = .2; });
  assert.deepEqual(buildViewModel(equal).rows.map(row => row.fromOpen.colorPosition), [1, 1, 1]);
  const negative = snapshotFixtureV2(3);
  negative.rows.forEach((row, index) => { row.from_open_change.value_ratio = [-.05, -.1, -.2][index]; });
  const negativeColors = buildViewModel(negative).rows.map(row => row.fromOpen.colorPosition);
  assert.deepEqual(negativeColors, [null, null, null]);
  const tied = snapshotFixtureV2(4);
  tied.rows.forEach((row, index) => { row.from_open_change.value_ratio = [.5, .2, .2, .1][index]; });
  assert.deepEqual(buildViewModel(tied).rows.map(row => row.fromOpen.colorPosition), [1, .25, .25, 0]);
  assert.deepEqual(buildViewModel(snapshotFixtureV2(0)).rows, []);
});

test("P-MVP-UI makes FROM CLOSE % and FROM OPEN % sign-aware with positive-only normalization", () => {
  const values = [1.6344, .3522, .016, 0, -.0099, -.1079];
  const snapshot = snapshotFixtureV2(values.length);
  snapshot.rows.forEach((row, index) => {
    row.day_change_ratio = values[index];
    row.from_open_change.value_ratio = values[index];
  });
  const model = buildViewModel(snapshot);
  const middlePosition = (values[1] - values[2]) / (values[0] - values[2]);
  assert.deepEqual(model.rows.map(row => row.dayColor), [1, middlePosition, 0, null, null, null]);
  assert.deepEqual(model.rows.map(row => row.fromOpen.colorPosition), [1, middlePosition, 0, null, null, null]);
  assert.deepEqual(model.rows.map(row => row.dayTextColor), [null, null, null, NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);
  assert.deepEqual(model.rows.map(row => row.fromOpen.textColor), [null, null, null, NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);
  assert.equal(signedValueTextColor(Number.NaN), null);
  assert.equal(signedValueTextColor(0), NEUTRAL_TEXT_COLOR);
  assert.equal(signedValueTextColor(-.005), MUTED_NEGATIVE_TEXT_COLOR);

  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model });
  const dayCells = find(document.body, node => node.dataset.focusKey?.endsWith(":day"));
  const fromOpenCells = find(document.body, node => node.dataset.focusKey?.endsWith(":fromopen"));
  const middleWeight = `${Math.round(middlePosition * 10000) / 100}%`;
  assert.deepEqual(dayCells.map(cell => cell.dataset.palette), ["day", "day", "day", undefined, undefined, undefined]);
  assert.deepEqual(dayCells.map(cell => cell.dataset.dayWeight), ["100%", middleWeight, "0%", undefined, undefined, undefined]);
  assert.deepEqual(fromOpenCells.map(cell => cell.dataset.palette), ["from-open", "from-open", "from-open", undefined, undefined, undefined]);
  assert.deepEqual(fromOpenCells.map(cell => cell.dataset.fromOpenWeight), ["100%", middleWeight, "0%", undefined, undefined, undefined]);
  assert.deepEqual(dayCells.map(cell => cell.children[0].style.color), [undefined, undefined, undefined, NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);
  assert.deepEqual(fromOpenCells.map(cell => cell.children[0].style.color), [undefined, undefined, undefined, NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);
  assert.deepEqual(dayCells.map(cell => cell.textContent), ["+163.44%", "+35.22%", "+1.60%", "0.00%", "-0.99%", "-10.79%"]);

  const noPositive = snapshotFixtureV2(3);
  noPositive.rows.forEach((row, index) => {
    row.day_change_ratio = [0, -.01, -.2][index];
    row.from_open_change.value_ratio = [0, -.01, -.2][index];
  });
  const noPositiveModel = buildViewModel(noPositive);
  assert.deepEqual(noPositiveModel.rows.map(row => row.dayColor), [null, null, null]);
  assert.deepEqual(noPositiveModel.rows.map(row => row.fromOpen.colorPosition), [null, null, null]);
  assert.deepEqual(noPositiveModel.rows.map(row => row.dayTextColor), [NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);
  assert.deepEqual(noPositiveModel.rows.map(row => row.fromOpen.textColor), [NEUTRAL_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR, MUTED_NEGATIVE_TEXT_COLOR]);

  const equalPositive = snapshotFixtureV2(2);
  equalPositive.rows.forEach(row => { row.day_change_ratio = .05; row.from_open_change.value_ratio = .05; });
  assert.deepEqual(buildViewModel(equalPositive).rows.map(row => row.dayColor), [1, 1]);
  assert.deepEqual(buildViewModel(equalPositive).rows.map(row => row.fromOpen.colorPosition), [1, 1]);
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
  const anchors = [[0, "#8F9AA3"], [5, "#8F9AA3"], [15, "#B8793E"], [50, "#E98212"], [100, "#FF8A00"]];
  for (const [value, color] of anchors) assert.equal(tapeColor(value), color);
  assert.deepEqual([tapeColor(2.5), tapeColor(10), tapeColor(32.5), tapeColor(75)], ["#8F9AA3", "#A48A71", "#D17E28", "#F48609"]);
  assert.notEqual(tapeColor(14), tapeColor(15)); assert.notEqual(tapeColor(15), tapeColor(16));
  assert.notEqual(tapeColor(40), tapeColor(50)); assert.notEqual(tapeColor(50), tapeColor(60));
  assert.equal(tapeColor(101), "#FF8A00"); assert.equal(tapeColor(5000), "#FF8A00");
});

test("P-MVP-UI interpolates Spread through exact absolute-bps RGB control points", () => {
  const anchors = [[0, "#8F9AA3"], [20, "#8F9AA3"], [35, "#B08A5A"], [60, "#D1843E"], [100, "#E05A32"], [200, "#E9342B"], [400, "#FC0000"]];
  for (const [value, color] of anchors) assert.equal(spreadColor(value), color);
  assert.deepEqual([spreadColor(10), spreadColor(27.5), spreadColor(47.5), spreadColor(80), spreadColor(150), spreadColor(300)], ["#8F9AA3", "#A0927F", "#C1874C", "#D96F38", "#E5472F", "#F31A16"]);
  assert.equal(spreadColor(19.9), "#8F9AA3");
  assert.notEqual(spreadColor(34), spreadColor(35)); assert.notEqual(spreadColor(35), spreadColor(36));
  assert.notEqual(spreadColor(59), spreadColor(60)); assert.notEqual(spreadColor(60), spreadColor(61));
  assert.notEqual(spreadColor(198), spreadColor(200)); assert.notEqual(spreadColor(200), spreadColor(202));
  assert.equal(spreadColor(401), "#FC0000"); assert.equal(spreadColor(1000), "#FC0000");
});

test("P-MVP-UI excludes non-current FROM OPEN % states from the relative scale", () => {
  const snapshot = snapshotFixtureV2(5);
  snapshot.rows[0].from_open_change.value_ratio = .4;
  snapshot.rows[1].from_open_change = { status: "warming", reason: "before_first_print", value_ratio: null };
  snapshot.rows[2].from_open_change = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  snapshot.rows[3].from_open_change = { status: "invalid", reason: "historical_conflict", value_ratio: null };
  snapshot.rows[4].from_open_change.value_ratio = -.2;
  const model = buildViewModel(snapshot);
  assert.deepEqual(model.rows.map(row => row.fromOpen.colorPosition), [1, null, null, null, null]);
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
  snapshot.rows[0].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
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
  pressure.rows[0].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
  const model = buildViewModel(pressure); assert.equal(model.current, true); assert.equal(model.tqAggregateOnly, true);
  const partial = snapshotFixtureV2(); partial.ranking.mode = "degraded_current"; partial.ranking.reason = "qualification_incomplete"; clearTQ(partial); const partialModel = buildViewModel(partial); assert.equal(partialModel.partial, true); assert.equal(partialModel.rowsCurrent, false);
});

test("P-MVP-TQ-RECOVERY renders the accepted pressure sample and recovery progress", () => {
  const snapshot = snapshotFixtureV2(); snapshot.status.tq_pressure_mode = "taq_degraded"; snapshot.status.tq_shed = true; snapshot.tq.pressure_mode = "taq_degraded"; snapshot.tq.pressure_cause = "oldest_waiting_frame"; snapshot.tq.shed = true;
  snapshot.tq.pressure_sample = { observed: true, waiting_frames: 12, frame_capacity: 32768, waiting_bytes: 4096, byte_capacity: 134217728, oldest_waiting_frame_age_ms: 612, aggregate_watermark_lag_ms: 1000, recovery_healthy: true };
  snapshot.tq.pressure_recovery = { healthy_samples: 3, required_samples: 5 };
  snapshot.rows[0].tape_5s = { ...snapshot.rows[0].tape_5s, status: "pressure_shed", reason: "pressure", trade_coverage: false, trades_per_second: null, timestamp_basis: "" };
  snapshot.rows[0].spread = { status: "pressure_shed", reason: "pressure", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  snapshot.rows[0].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
  const model = buildViewModel(snapshot); assert.equal(model.current, true); assert.equal(model.tqOldestWaitingFrameAgeMS, 612); assert.equal(model.tqRecoveryHealthySamples, 3);
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model });
  assert.equal(document.getElementById("status-live").textContent, "LIVE");
  assert.match(document.body.textContent, /TAPE \/ QUOTES DEGRADED.*Trades1 shed.*Quotes1 shed/s);
  const missReset = clone(snapshot); missReset.tq.pressure_recovery.healthy_samples = 0;
  const resetDocument = new FakeDocument(); renderDashboard(resetDocument, { transport: "connected", model: buildViewModel(missReset) });
  assert.equal(resetDocument.getElementById("status-live").textContent, "LIVE");
  assert.match(resetDocument.body.textContent, /TAPE \/ QUOTES DEGRADED/);
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
  assert.equal(document.getElementById("status-live").textContent, "RECOVERING");
  assert.match(document.getElementById("aggregate-status").textContent, /Repairing aggregate gap.*5,099 \/ 5,522/);
  assert.equal(find(document.body, node => node.className === "message").length, 0);

  const hydrating = recoverySnapshot({ active: true, connected: true, acknowledged: true, lifecycle: "hydrating", generation: "1", open: "5505", completedValue: "17" });
  const hydratingModel = buildViewModel(hydrating);
  assert.equal(hydratingModel.phase, "hydrating");
  const hydratingDocument = new FakeDocument(); renderDashboard(hydratingDocument, { transport: "connected", model: hydratingModel });
  assert.equal(hydratingDocument.getElementById("status-live").textContent, "WARMING UP");
  assert.match(hydratingDocument.getElementById("aggregate-status").textContent, /Syncing history.*17 \/ 5,522/);
  assert.equal(find(hydratingDocument.body, node => node.className === "message").length, 0);

  const reconnectDocument = new FakeDocument(); renderDashboard(reconnectDocument, { transport: "connected", model: reconnecting });
  assert.equal(reconnectDocument.getElementById("status-live").textContent, "RECOVERING");
  assert.match(reconnectDocument.getElementById("aggregate-status").textContent, /Reconnecting aggregate stream/); assert.doesNotMatch(reconnectDocument.body.textContent, /2 \/ 2/);

  const priorV2 = recoverySnapshot({ connected: true, acknowledged: true }); delete priorV2.recovery.generation_active;
  const compatible = buildViewModel(priorV2); assert.equal(compatible.phase, "preparing_recovery"); assert.equal(compatible.recoveryGenerationActive, false);
});

test("P-MVP-UI bounded poller skips overlap, freezes, resamples, replaces, and recovers", async () => {
  const updates = []; let resolveFirst; const deferred = new Promise(resolve => { resolveFirst = resolve; });
  const controller = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", requestTimeoutMilliseconds: 30, fetchImpl: () => deferred, onUpdate: update => updates.push(update) });
  const first = controller.tick(); await controller.tick(); assert.equal(updates.at(-1).transport, "refresh_delayed"); resolveFirst(new Response(JSON.stringify(snapshotFixtureV2()))); await first;
  const same = snapshotFixtureV2(); same.sample.id = "11"; same.sample.sampled_at = "2026-08-08T16:00:01Z"; controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(same))); await controller.tick(); assert.equal(updates.at(-1).model.sampleID, "11");
  const next = snapshotFixtureV2(); next.publication.id = "21"; next.rows[0].symbol = "REPLACED"; next.tq.desired_symbols[0] = "REPLACED"; controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(next))); await controller.tick(); assert.equal(updates.at(-1).model.rows[0].symbol, "REPLACED");
  controller.fetchImpl = () => Promise.resolve(new Response("{}", { status: 503 })); await controller.tick(); assert.equal(updates.at(-1).transport, "api_unavailable"); assert.equal(updates.at(-1).model.publicationID, "21"); controller.fetchImpl = () => Promise.resolve(new Response(JSON.stringify(snapshotFixtureV2()))); await controller.tick(); assert.equal(updates.at(-1).transport, "connected");
});

test("P5 poller separates self-timeout, API failure, and stop cancellation", async () => {
  const validResponse = () => new Response(JSON.stringify(snapshotFixtureV2()));
  const timeoutUpdates = []; let timeoutMode = "timeout";
  const timeoutController = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", requestTimeoutMilliseconds: 5,
    fetchImpl: (_, options) => timeoutMode === "success" ? Promise.resolve(validResponse()) : new Promise((_, reject) => options.signal.addEventListener("abort", () => reject(new Error("aborted")), { once: true })),
    onUpdate: update => timeoutUpdates.push(update) });
  await timeoutController.tick(); assert.equal(timeoutUpdates.at(-1).transport, "refresh_delayed");
  const timeoutDocument = new FakeDocument(); renderDashboard(timeoutDocument, timeoutUpdates.at(-1));
  assert.equal(timeoutDocument.getElementById("status-live").textContent, "DELAYED");
  assert.ok(timeoutUpdates.every(update => update.transport !== "disconnected"));
  timeoutMode = "success"; await timeoutController.tick(); assert.equal(timeoutUpdates.at(-1).transport, "connected");
  timeoutMode = "timeout"; await timeoutController.tick();
  assert.equal(timeoutUpdates.at(-1).transport, "refresh_delayed"); assert.equal(timeoutUpdates.at(-1).model.current, true);
  const retainedDocument = new FakeDocument(); renderDashboard(retainedDocument, timeoutUpdates.at(-1));
  assert.equal(retainedDocument.getElementById("status-live").textContent, "DELAYED"); assert.equal(retainedDocument.getElementById("scanner-table").dataset.publicationState, "noncurrent");
  assert.ok(find(retainedDocument.body, node => node.tagName === "TD").every(cell => cell.dataset.state === "retained"));
  timeoutMode = "success"; await timeoutController.tick(); assert.equal(timeoutUpdates.at(-1).transport, "connected");
  const recoveredDocument = new FakeDocument(); renderDashboard(recoveredDocument, timeoutUpdates.at(-1));
  assert.equal(recoveredDocument.getElementById("status-live").textContent, "LIVE"); assert.equal(recoveredDocument.getElementById("scanner-table").dataset.publicationState, "current");

  const failureUpdates = []; let failureMode = "network";
  const failureController = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", fetchImpl: () => failureMode === "network" ? Promise.reject(new Error("network down")) : validResponse(), onUpdate: update => failureUpdates.push(update) });
  await failureController.tick(); assert.equal(failureUpdates.at(-1).transport, "api_unavailable");
  const failureDocument = new FakeDocument(); renderDashboard(failureDocument, failureUpdates.at(-1));
  assert.equal(failureDocument.getElementById("status-live").textContent, "UNAVAILABLE");
  assert.ok(failureUpdates.every(update => update.transport !== "disconnected"));
  failureMode = "success"; await failureController.tick(); assert.equal(failureUpdates.at(-1).transport, "connected");

  const unavailableUpdates = []; let unavailableMode = "503";
  const unavailableController = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", fetchImpl: () => unavailableMode === "503" ? Promise.resolve(new Response("{}", { status: 503 })) : Promise.resolve(validResponse()), onUpdate: update => unavailableUpdates.push(update) });
  await unavailableController.tick(); assert.equal(unavailableUpdates.at(-1).transport, "api_unavailable"); assert.ok(unavailableUpdates.every(update => update.transport !== "disconnected"));
  unavailableMode = "success"; await unavailableController.tick(); assert.equal(unavailableUpdates.at(-1).transport, "connected");

  const recoveringDocument = new FakeDocument(); renderDashboard(recoveringDocument, { transport: "connected", model: buildViewModel(recoverySnapshot({ active: true, connected: true, acknowledged: true, open: "5520" })) });
  assert.equal(recoveringDocument.getElementById("status-live").textContent, "RECOVERING");

  const stopUpdates = [];
  let stoppedSignal;
  const stopController = new PollController({ url: "http://127.0.0.1/api/v2/snapshot", fetchImpl: (_, options) => {
    stoppedSignal = options.signal;
    return new Promise((_, reject) => options.signal.addEventListener("abort", () => reject(new Error("stopped")), { once: true }));
  }, onUpdate: update => stopUpdates.push(update) });
  const stopped = stopController.tick(); stopController.stop(); await stopped;
  assert.equal(stoppedSignal.aborted, true); assert.equal(stopUpdates.length, 0);
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
  assert.equal(find(document.body, node => node.className === "status-grid")[0], undefined);
  const status = document.getElementById("system-status"), statusSummary = document.getElementById("status-summary");
  const statusDetails = find(document.body, node => node.className === "status-detail");
  assert.equal(status.tagName, "DETAILS"); assert.equal(statusSummary.tagName, "SUMMARY");
  assert.deepEqual(statusDetails.map(item => item.children[0].textContent), ["Scanner", "Aggregates", "Trades", "Quotes"]);
  for (const label of ["Backend", "Ranking mode", "Dashboard feed", "Transport", "Process", "Ops sample", "Watermark", "Sample"]) assert.doesNotMatch(document.body.textContent, new RegExp(`\\b${label}\\b`, "i"));
  assert.match(document.body.textContent, /CONTEXT.*LOCATION.*MOMENTUM.*TAPE \/ EXECUTION/s);
  assert.equal(document.getElementById("status-live").textContent, "LIVE");
  assert.match(document.getElementById("scanner-status").textContent, /exact qualified ranking/);
  assert.equal(document.getElementById("trade-status").textContent, "2/2 current");
  assert.equal(document.getElementById("quote-status").textContent, "2/2 current");
  assert.doesNotMatch(document.body.textContent, /\b2 ranked\b|publication \d+/i);
  const headers = find(document.body, node => node.tagName === "TH"); const leafHeaders = headers.filter(node => node.attributes.scope === "col"); assert.deepEqual(leafHeaders.map(node => node.textContent), ["RANK", "SYMBOL", "FLOAT", "VOLUME", "LAST", "FROM CLOSE %", "FROM OPEN %", "DAY RANGE", "ACTIVITY 30s", "MOVE 30s", "TAPE SPEED", "SPREAD"]); assert.deepEqual(leafHeaders.map(node => node.attributes["data-tooltip"]), ["Current scanner rank and movement compared with approximately 60 seconds ago.", "Ticker symbol for the listed stock.", "Estimated number of publicly tradable shares.", "Total shares traded during the current scanner session.", "Latest trusted price.", "Percent change from the adjusted previous close.", "Percent change from the first eligible session price.", "Current price position between today's low and high.", "Recent share-volume activity compared with the prior five-minute baseline.", "Signed price change over the last 30 seconds.", "Qualified trades per second over the last five seconds.", "Difference between the current bid and ask, shown in cents and basis points."]); assert.ok(leafHeaders.every(node => node.tabIndex === undefined && node.title === undefined)); const cells = find(document.body, node => node.tagName === "TD"); assert.ok(cells.every(cell => cell.title === undefined)); assert.equal(find(document.body, node => node.tagName === "TBODY")[0].children[0].children.length, 12);
  const rankCells = find(document.body, node => node.className === "rank-cell"); assert.deepEqual(rankCells.map(cell => cell.children[0].children[0].textContent), ["#1", "#2"]); assert.ok(rankCells.every(cell => cell.children[0].children[1].textContent === ""));
  const committed = document.body.children[0], tableHead = find(document.body, node => node.tagName === "THEAD")[0], originalRows = document.getElementById("rows").textContent; status.open = true;
  assert.throws(() => renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }, { beforeCommit: () => { throw new Error("render failed"); } })); assert.equal(document.body.children[0], committed); assert.equal(document.getElementById("rows").textContent, originalRows);
  const focused = find(document.body, node => node.dataset.focusKey?.endsWith(":spread"))[0]; focused.focus(); const key = focused.dataset.focusKey; [snapshot.rows[0], snapshot.rows[1]] = [snapshot.rows[1], snapshot.rows[0]]; snapshot.rows.forEach((row, index) => { row.rank = index + 1; }); snapshot.tq.desired_symbols = snapshot.rows.map(row => row.symbol); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) }); assert.equal(document.activeElement.dataset.focusKey, key);
  assert.equal(find(document.body, node => node.tagName === "THEAD")[0], tableHead, "poll replaced the header tooltip anchors"); assert.equal(document.getElementById("status-summary"), statusSummary); assert.equal(status.open, true, "poll closed the status disclosure");
  renderDashboard(document, { transport: "connected", model: buildViewModel(snapshotFixtureV2()) }); assert.equal(document.getElementById("diagnostics"), null); assert.doesNotMatch(document.body.textContent, /Operational details/); const source = await readFile(new URL("./render.js", import.meta.url), "utf8"); assert.doesNotMatch(source, /innerHTML|outerHTML|insertAdjacentHTML|document\.write/);
});

test("P-MVP-UI gives delayed and disconnected transport precedence over retained model state", () => {
  const currentModel = buildViewModel(snapshotFixtureV2(2));
  for (const [transport, label] of [["refresh_delayed", "DELAYED"], ["disconnected", "DISCONNECTED"]]) {
    const document = new FakeDocument(); renderDashboard(document, { transport, model: currentModel });
    const live = document.getElementById("status-live"), table = document.getElementById("scanner-table"), cells = find(document.body, node => node.tagName === "TD");
    assert.match(live.textContent, new RegExp(`^${label}`)); assert.equal(live.dataset.state, "warning"); assert.equal(table.dataset.publicationState, "noncurrent"); assert.ok(cells.every(cell => cell.dataset.state === "retained"));
  }
  const partialSnapshot = snapshotFixtureV2(2); partialSnapshot.ranking.mode = "degraded_current"; partialSnapshot.ranking.reason = "qualification_incomplete"; clearTQ(partialSnapshot);
  const partialDocument = new FakeDocument(); renderDashboard(partialDocument, { transport: "disconnected", model: buildViewModel(partialSnapshot) });
  assert.equal(partialDocument.getElementById("status-live").textContent, "DISCONNECTED");
});

test("P-MVP-UI maps scanner lifecycle to trader-facing status without a row count", () => {
  const current = buildViewModel(snapshotFixtureV2(2));
  const cases = [
    [{ transport: "connected", model: null }, "CONNECTING"],
    [{ transport: "disconnected", model: null, error: "offline" }, "DISCONNECTED"],
    [{ transport: "connected", model: { ...current, current: false, backendReady: false, lifecycle: "awaiting_session" } }, "WAITING FOR SESSION"],
    [{ transport: "connected", model: { ...current, current: false, backendReady: false, lifecycle: "initializing" } }, "WARMING UP"],
    [{ transport: "connected", model: current }, "LIVE"],
    [{ transport: "connected", model: { ...current, current: false, partial: true, rankingMode: "degraded_current" } }, "PARTIAL"],
    [{ transport: "connected", model: { ...current, current: false, backendReady: false, lifecycle: "recovering", recovering: true } }, "RECOVERING"],
    [{ transport: "connected", model: { ...current, current: false, backendReady: false, lifecycle: "suppressed" } }, "UNAVAILABLE"],
    [{ transport: "connected", model: { ...current, current: false, backendReady: false, lifecycle: "ended" } }, "SESSION ENDED"],
    [{ transport: "connected", model: { ...current, current: false, replay: true } }, "HISTORICAL"],
  ];
  for (const [event, expected] of cases) {
    const document = new FakeDocument(); renderDashboard(document, event);
    assert.equal(document.getElementById("status-live").textContent, expected);
    assert.doesNotMatch(document.getElementById("status-summary").textContent, /ranked/i);
  }
  const emptyDocument = new FakeDocument(); renderDashboard(emptyDocument, { transport: "connected", model: buildViewModel(snapshotFixtureV2(0)) });
  assert.equal(emptyDocument.getElementById("status-live").textContent, "LIVE", "exact empty ranking was not live");
});

test("P-MVP-UI summarizes trade and quote coverage separately", () => {
  const snapshot = snapshotFixtureV2(3);
  snapshot.rows[1].tape_5s = { status: "warming", reason: "five_second_warming", trade_coverage: true, trades_per_second: null, timestamp_basis: "mixed", lifecycle_records_observed: true };
  snapshot.rows[2].spread = { ...snapshot.rows[2].spread, status: "stale", reason: "stale_quote", quote_age_ms: 9000 };
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  assert.equal(document.getElementById("trade-status").textContent, "2/3 current · 1 warming");
  assert.equal(document.getElementById("quote-status").textContent, "2/3 current · 1 stale");
  assert.equal(document.getElementById("status-live").textContent, "LIVE");
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
  const rates = [0, 5, 15, 50, 100, 125];
  const snapshot = snapshotFixtureV2(rates.length);
  rates.forEach((rate, index) => { snapshot.rows[index].tape_5s.trades_per_second = rate; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":tape"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), rates.map(tapeColor));
  assert.deepEqual(cells.map(cell => cell.textContent), ["0.0/s", "5.0/s", "15.0/s", "50.0/s", "100.0/s", "125.0/s"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));

  const unavailable = snapshotFixtureV2(); unavailable.rows[0].tape_5s = { status: "unavailable", reason: "coverage", trade_coverage: false, trades_per_second: null, timestamp_basis: "none", lifecycle_records_observed: false };
  unavailable.rows[0].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
  const unavailableDocument = new FakeDocument(); renderDashboard(unavailableDocument, { transport: "connected", model: buildViewModel(unavailable) });
  const unavailableCell = find(unavailableDocument.body, node => node.dataset.focusKey?.endsWith(":tape"))[0];
  assert.equal(unavailableCell.children[0].style.color, undefined); assert.equal(unavailableCell.dataset.fieldState, "unavailable"); assert.equal(unavailableCell.style.background, undefined); assert.equal(unavailableCell.dataset.palette, undefined);
});

test("P-MVP-UI colors only current Spread text and leaves invalid or unavailable cells neutral", () => {
  const bps = [0, 20, 35, 60, 100, 450];
  const snapshot = snapshotFixtureV2(bps.length);
  bps.forEach((value, index) => { snapshot.rows[index].spread.basis_points = value; });
  const document = new FakeDocument(); renderDashboard(document, { transport: "connected", model: buildViewModel(snapshot) });
  const cells = find(document.body, node => node.dataset.focusKey?.endsWith(":spread"));
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), bps.map(spreadColor));
  assert.deepEqual(cells.map(cell => cell.textContent), ["0.0 bps / 1.50¢", "20.0 bps / 1.50¢", "35.0 bps / 1.50¢", "60.0 bps / 1.50¢", "100.0 bps / 1.50¢", "450.0 bps / 1.50¢"]);
  assert.ok(cells.every(cell => cell.style.background === undefined && cell.dataset.palette === undefined));

  const invalid = snapshotFixtureV2(2);
  invalid.rows[0].spread = { status: "invalid", reason: "crossed_quote", quote_coverage: true, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  invalid.rows[1].spread = { status: "unavailable", reason: "coverage", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
  invalid.rows[1].tq_membership = { desired: true, provider_present: false, provider_membership_unknown: true };
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
  assert.deepEqual(cells.map(cell => cell.dataset.palette), ["from-open", "from-open", undefined]);
  assert.deepEqual(cells.map(cell => cell.dataset.fromOpenWeight), ["100%", "0%", undefined]);
  assert.deepEqual(cells.map(cell => cell.children[0].style.color), [undefined, undefined, MUTED_NEGATIVE_TEXT_COLOR]);
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
  assert.deepEqual(cells.map(cell => cell.dataset.fromOpenWeight), ["100%", "0%", undefined]);
});

test("P-MVP-UI assets contain only v2 and final-field representations", async () => {
  const assets = await Promise.all(["app.js", "model.js", "render.js", "index.html"].map(name => readFile(new URL(`./${name}`, import.meta.url), "utf8"))); const all = assets.join("\n"); assert.match(all, /api\/v2\/snapshot/); assert.doesNotMatch(all, /api\/v1\/snapshot|scanner\.snapshot\.v1|from_4am|hod_drawdown|range_30m|range_60m|tape_rate/); assert.match(assets[2], /data-tooltip/); assert.doesNotMatch(assets[2], /\.title\s*=/);
  assert.doesNotMatch(assets[3], /<span>(?:Transport|Process|Ops sample|Watermark|Sample)<\/span>/i);
  assert.doesNotMatch(assets[3], /Operational details/i);
});
