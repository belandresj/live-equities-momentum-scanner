function element(document, tag, className = "") {
  const node = document.createElement(tag); if (className) node.className = className; return node;
}
function textElement(document, tag, value, className = "") { const node = element(document, tag, className); node.textContent = value; return node; }
function statusItem(document, label, value, state = "") {
  const item = element(document, "div"); item.append(textElement(document, "span", label)); const strong = textElement(document, "strong", value); if (state) strong.dataset.state = state; item.append(strong); return item;
}
const COLUMN_DEFINITIONS = [
  ["SYMBOL", "Ticker symbol for the listed stock."],
  ["FLOAT", "Estimated number of publicly tradable shares."],
  ["VOLUME", "Total shares traded during the current scanner session."],
  ["LAST", "Latest trusted price."],
  ["FROM CLOSE %", "Percent change from the adjusted previous close."],
  ["FROM OPEN %", "Percent change from the first eligible session price."],
  ["DAY RANGE", "Current price position between today's low and high."],
  ["ACTIVITY 30s", "Recent share-volume activity compared with the prior five-minute baseline."],
  ["MOVE 30s", "Signed price change over the last 30 seconds."],
  ["TAPE SPEED", "Qualified trades per second over the last five seconds."],
  ["SPREAD", "Difference between the current bid and ask, shown in cents and basis points."],
];
function percentageWeight(position) { return `${Math.round(position * 100) / 100}%`; }
function cell(document, value, state, detail, overallCurrent, band = 0, focusKey = "", palette = "", position = null, textColor = "") {
  const td = element(document, "td"); td.dataset.state = overallCurrent ? state : "retained"; td.dataset.fieldState = state;
  td.dataset.band = String(band); if (palette) td.dataset.palette = palette;
  if (palette === "heat" && position !== null) { td.dataset.heatPosition = String(position); td.dataset.heatWeight = percentageWeight(position); }
  if (palette === "day" && position !== null) { td.dataset.dayColor = String(position); td.dataset.dayWeight = percentageWeight(position * 100); }
  if (palette === "from-open" && position !== null) { td.dataset.fromOpenColor = String(position); td.dataset.fromOpenWeight = percentageWeight(position * 100); }
  if (palette === "volume-turnover" && position !== null) td.dataset.volumeTurnoverWeight = percentageWeight(position * 100);
  if (palette === "float" && position !== null) td.dataset.floatWeight = percentageWeight(position * 100);
  const primary = textElement(document, "span", value, "primary");
  if (overallCurrent && textColor) primary.style.color = textColor;
  td.append(primary);
  if (detail) { td.tabIndex = 0; td.dataset.focusKey = focusKey; td.setAttribute("aria-label", `${value}. ${detail}`); }
  return td;
}
const TABLE_COLUMN_CLASSES = [
  "context-column", "context-column", "context-column", "context-column",
  "location-column", "location-column", "location-column",
  "momentum-column", "momentum-column",
  "execution-column", "execution-column",
];
function buildTable(document, model) {
  const shell = element(document, "div", "table-shell");
  const table = element(document, "table"); table.id = "scanner-table"; table.dataset.publicationState = model.rowsCurrent ? "current" : "noncurrent";
  const columnGroup = element(document, "colgroup");
  for (const className of TABLE_COLUMN_CLASSES) columnGroup.append(element(document, "col", className));
  table.append(textElement(document, "caption", model.partial ? "Server-ranked top 20 trusted marks by From Close %, qualification not asserted" : "Server-ranked top 20 qualifying equities"), columnGroup);
  const thead = element(document, "thead"), groups = element(document, "tr", "group-header"), header = element(document, "tr", "leaf-header");
  for (const [label, span] of [["CONTEXT", 4], ["LOCATION", 3], ["CURRENT MOMENTUM", 2], ["EXECUTION", 2]]) {
    const th = textElement(document, "th", label); th.setAttribute("scope", "colgroup"); th.setAttribute("colspan", String(span)); groups.append(th);
  }
  for (const [label, description] of COLUMN_DEFINITIONS) { const th = textElement(document, "th", label); th.setAttribute("scope", "col"); th.setAttribute("data-tooltip", description); header.append(th); }
  thead.append(groups, header); table.append(thead);
  const tbody = element(document, "tbody"); tbody.id = "rows";
  for (const row of model.rows) {
    const tr = element(document, "tr"); tr.dataset.publicationState = model.rowsCurrent ? "current" : "noncurrent"; tr.setAttribute("aria-label", `Server rank ${row.rank}, ${row.symbol}`);
    const key = row.symbol;
    tr.append(cell(document, row.symbol, "current", `server rank ${row.rank}`, model.rowsCurrent, 0, `${key}:symbol`),
      cell(document, row.float.text, row.float.state, row.float.detail, model.rowsCurrent, 0, `${key}:float`, row.float.cyanIntensity === null ? "" : "float", row.float.cyanIntensity), cell(document, row.volume.text, row.volume.state, row.volume.detail, model.rowsCurrent, 0, `${key}:volume`, row.volume.colorIntensity === null ? "" : "volume-turnover", row.volume.colorIntensity), cell(document, row.last, "current", `mark age ${row.markAgeMS} ms`, model.rowsCurrent, 0, `${key}:last`),
      cell(document, row.day, "current", "From Close %: server-published ranking value; color is relative to the displayed snapshot", model.rowsCurrent, 0, `${key}:day`, "day", row.dayColor), cell(document, row.fromOpen.text, row.fromOpen.state, row.fromOpen.detail, model.rowsCurrent, row.fromOpen.band, `${key}:fromopen`, row.fromOpen.colorPosition === null ? "" : "from-open", row.fromOpen.colorPosition), cell(document, row.dayRange.text, row.dayRange.state, row.dayRange.detail, model.rowsCurrent, row.dayRange.band, `${key}:dayrange`, "", null, row.dayRange.textColor),
      cell(document, row.activity.text, row.activity.state, row.activity.detail, model.rowsCurrent, row.activity.band, `${key}:activity`, "", null, row.activity.textColor), cell(document, row.move.text, row.move.state, row.move.detail, model.rowsCurrent, row.move.band, `${key}:move`, "", null, row.move.textColor),
      cell(document, row.tape.primary, row.tape.state, row.tape.detail, model.rowsCurrent, 0, `${key}:tape`, "", null, row.tape.textColor), cell(document, row.spread.primary, row.spread.state, row.spread.detail, model.rowsCurrent, 0, `${key}:spread`, "", null, row.spread.textColor));
    tbody.append(tr);
  }
  table.append(tbody); shell.append(table); return shell;
}
export function renderDashboard(document, event, options = {}) {
  const focusKey = document.activeElement?.dataset?.focusKey || "";
  const main = element(document, "main"), model = event.model;
  const header = element(document, "header"), title = element(document, "div"); title.append(textElement(document, "p", "LIVE EQUITIES", "eyebrow"), textElement(document, "h1", "Momentum Scanner"));
  const replayState = event.transport === "refresh_delayed" ? " · REFRESH DELAYED" : event.transport === "disconnected" ? " · FROZEN · DISCONNECTED" : "";
  const phaseLabel = !model ? "" : ({ connecting: "CONNECTING", subscribing: "SUBSCRIBING", hydrating: "HYDRATING", preparing_hydration: "PREPARING HYDRATION", finalizing_hydration: "FINALIZING HYDRATION", reconnecting: "RECONNECTING", resubscribing: "RESUBSCRIBING", preparing_recovery: "PREPARING RECOVERY", recovering: "RECOVERING", retrying: "RETRYING RECOVERY", finalizing_recovery: "FINALIZING RECOVERY" })[model.phase] || "";
  const primaryState = !model ? "" : model.replay ? `HISTORICAL · NONLIVE${replayState}` : model.current ? "CURRENT" : model.partial ? "PARTIAL · CURRENT DATA" : event.transport === "refresh_delayed" ? "REFRESH DELAYED" : event.transport === "disconnected" ? "FROZEN · DISCONNECTED" : model.backendReady && model.rankingMode === "degraded_bootstrap" ? "DEGRADED" : phaseLabel || "NONCURRENT";
  const workProgress = model && event.transport === "connected" && ["hydrating", "recovering", "retrying", "finalizing_hydration", "finalizing_recovery"].includes(model.phase) ? ` · generation ${model.recoveryGeneration} · ${model.hydrationProgress}${model.hydrationIssueText}` : "";
  const live = textElement(document, "div", model ? `${primaryState}${workProgress} · ${model.rows.length} ranked` : event.error ? `Scanner API disconnected: ${event.error}` : "Connecting to scanner API", "status-live");
  live.id = "status-live"; live.dataset.state = model?.current && !model.replay ? "current" : "warning"; header.append(title, live); main.append(header);
  const grid = element(document, "section", "status-grid"); grid.setAttribute("aria-label", "Scanner status");
  const tqRecovery = !model || model.tqPressure === "normal" ? "" : model.tqPressureSampleObserved
    ? ` · ${model.tqPressureCause || "pressure"} · oldest ${model.tqOldestWaitingFrameAgeMS} ms · recovery ${model.tqRecoveryHealthySamples}/${model.tqRecoveryRequiredSamples}`
    : ` · ${model.tqPressureCause || "pressure"} · recovery sample unavailable`;
  grid.append(
    statusItem(document, "Backend", model ? model.backendReady ? "ready" : phaseLabel ? `${phaseLabel.toLowerCase()}${workProgress}` : model.readinessReason || "not ready" : "unknown", model?.backendReady ? "current" : "warning"),
    statusItem(document, "Ranking", model ? `${model.rankingMode}${model.rankingReason ? ` · ${model.rankingReason}` : ""}` : "unknown"),
    statusItem(document, "T/Q", model ? `${model.tqPressure}${model.tqAggregateOnly ? " · aggregate only" : ""}${model.tqRetainedBoundHit ? " · retained bound" : ""}${tqRecovery} · ${model.tqUnknown} unknown` : "unknown", model?.tqPressure === "normal" && !model?.tqRetainedBoundHit && model?.tqUnknown === 0 ? "current" : "warning"),
  );
  main.append(grid);
  if (model?.lifecycle === "suppressed") {
    const suppressed = element(document, "div", "message");
    suppressed.dataset.state = "suppressed";
    suppressed.textContent = `SCANNER SUPPRESSED · ${model.lifecycleReason || "integrity failure"} · ${model.suppression || "restart required"}${model.integrityFailure ? ` · ${model.integrityFailure.category} at engine sequence ${model.integrityFailure.engine_sequence}` : ""}`;
    main.append(suppressed);
  }
  if (model?.partial) {
    const partial = element(document, "div", "message");
    partial.dataset.state = "partial";
    partial.textContent = `PARTIAL RANKING · current trusted marks ordered by From Close % · qualification is not asserted · ${model.rankingReason || "symbol-local uncertainty"}`;
    main.append(partial);
  }
  const message = element(document, "div", "message"); message.id = "message";
  if (!model) message.textContent = "No valid scanner snapshot is available.";
  else if (model.rows.length === 0) message.textContent = model.rowsCurrent && model.rankingMode === "qualified_current" ? "No symbols currently qualify." : phaseLabel ? `${phaseLabel.toLowerCase()}${workProgress}. Ranking remains noncurrent until the aggregate fence reconciles.` : `No rows in retained noncurrent publication · ${model.rankingMode}${model.rankingReason ? ` · ${model.rankingReason}` : ""}.`;
  else message.hidden = true;
  main.append(message);
  if (model) main.append(buildTable(document, model));
  options.beforeCommit?.(main);
  let mount = document.getElementById?.("app");
  if (!mount) { mount = element(document, "div"); mount.id = "app"; document.body.append(mount); }
  let announcer = document.getElementById?.("announcer");
  if (!announcer) { announcer = element(document, "div", "sr-only"); announcer.id = "announcer"; announcer.setAttribute("role", "status"); announcer.setAttribute("aria-live", "polite"); document.body.append(announcer); }
  mount.replaceChildren(main);
  announcer.textContent = live.textContent;
  if (focusKey) {
    try {
      const restored = mount.querySelector?.(`[data-focus-key="${CSS.escape(focusKey)}"]`);
      restored?.focus({ preventScroll: true });
    } catch { /* full coherent view is already committed */ }
  }
  return main;
}
