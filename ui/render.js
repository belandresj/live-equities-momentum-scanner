function element(document, tag, className = "") {
  const node = document.createElement(tag); if (className) node.className = className; return node;
}
function textElement(document, tag, value, className = "") { const node = element(document, tag, className); node.textContent = value; return node; }
function statusItem(document, label, value, state = "") {
  const item = element(document, "div"); item.append(textElement(document, "span", label)); const strong = textElement(document, "strong", value); if (state) strong.dataset.state = state; item.append(strong); return item;
}
function rangeWeights(position) {
  return { red: position < 50 ? (50 - position) * 2 : 0, green: position > 50 ? (position - 50) * 2 : 0 };
}
function percentageWeight(position) { return `${Math.round(position * 100) / 100}%`; }
function cell(document, value, state, detail, overallCurrent, band = 0, focusKey = "", palette = "", position = null) {
  const td = element(document, "td"); td.dataset.state = overallCurrent ? state : "retained"; td.dataset.fieldState = state;
  td.dataset.band = String(band); if (palette) td.dataset.palette = palette;
  if (palette === "range" && position !== null) { const weights = rangeWeights(position); td.dataset.rangePosition = String(position); td.dataset.rangeRedWeight = `${weights.red}%`; td.dataset.rangeGreenWeight = `${weights.green}%`; }
  if (palette === "heat" && position !== null) { td.dataset.heatPosition = String(position); td.dataset.heatWeight = percentageWeight(position); }
  td.append(textElement(document, "span", value, "primary"));
  if (detail) { td.tabIndex = 0; td.title = detail; td.dataset.focusKey = focusKey; td.setAttribute("aria-label", `${value}. ${detail}`); }
  return td;
}
function buildTable(document, model) {
  const shell = element(document, "div", "table-shell");
  const table = element(document, "table"); table.id = "scanner-table"; table.dataset.publicationState = model.rowsCurrent ? "current" : "noncurrent";
  table.append(textElement(document, "caption", model.partial ? "Server-ranked top 20 trusted marks by Day %, qualification not asserted" : "Server-ranked top 20 qualifying equities"));
  const thead = element(document, "thead"), groups = element(document, "tr", "group-header"), header = element(document, "tr", "leaf-header");
  for (const [label, span] of [["CONTEXT", 4], ["LOCATION", 3], ["CURRENT MOMENTUM", 2], ["EXECUTION", 2]]) {
    const th = textElement(document, "th", label); th.setAttribute("scope", "colgroup"); th.setAttribute("colspan", String(span)); groups.append(th);
  }
  for (const label of ["SYMBOL", "FLOAT", "VOLUME", "LAST", "DAY %", "FROM OPEN %", "DAY RANGE", "ACTIVITY 30s", "MOVE 30s", "TAPE 5s", "SPREAD"]) { const th = textElement(document, "th", label); th.setAttribute("scope", "col"); header.append(th); }
  thead.append(groups, header); table.append(thead);
  const tbody = element(document, "tbody"); tbody.id = "rows";
  for (const row of model.rows) {
    const tr = element(document, "tr"); tr.dataset.publicationState = model.rowsCurrent ? "current" : "noncurrent"; tr.setAttribute("aria-label", `Server rank ${row.rank}, ${row.symbol}`);
    const key = row.symbol;
    tr.append(cell(document, row.symbol, "current", `server rank ${row.rank}`, model.rowsCurrent, 0, `${key}:symbol`),
      cell(document, row.float.text, row.float.state, row.float.detail, model.rowsCurrent, 0, `${key}:float`), cell(document, row.volume.text, row.volume.state, row.volume.detail, model.rowsCurrent, 0, `${key}:volume`), cell(document, row.last, "current", `mark age ${row.markAgeMS} ms`, model.rowsCurrent, 0, `${key}:last`),
      cell(document, row.day, "current", "Day %: server-published ranking value", model.rowsCurrent, row.dayBand, `${key}:day`), cell(document, row.fromOpen.text, row.fromOpen.state, row.fromOpen.detail, model.rowsCurrent, row.fromOpen.band, `${key}:fromopen`), cell(document, row.dayRange.text, row.dayRange.state, row.dayRange.detail, model.rowsCurrent, row.dayRange.band, `${key}:dayrange`, "range", row.dayRange.position),
      cell(document, row.activity.text, row.activity.state, row.activity.detail, model.rowsCurrent, row.activity.band, `${key}:activity`, "heat", row.activity.position), cell(document, row.move.text, row.move.state, row.move.detail, model.rowsCurrent, row.move.band, `${key}:move`, row.move.palette),
      cell(document, row.tape.primary, row.tape.state, row.tape.detail, model.rowsCurrent, 0, `${key}:tape`, "heat", row.tape.position), cell(document, row.spread.primary, row.spread.state, row.spread.detail, model.rowsCurrent, row.spread.band, `${key}:spread`, "spread"));
    tbody.append(tr);
  }
  table.append(tbody); shell.append(table); return shell;
}
export function renderDashboard(document, event, options = {}) {
  const focusKey = document.activeElement?.dataset?.focusKey || "";
  const main = element(document, "main"), model = event.model;
  const header = element(document, "header"), title = element(document, "div"); title.append(textElement(document, "p", "LIVE EQUITIES", "eyebrow"), textElement(document, "h1", "Momentum Scanner"));
  const replayState = event.transport === "refresh_delayed" ? " · REFRESH DELAYED" : event.transport === "disconnected" ? " · FROZEN · DISCONNECTED" : "";
  const primaryState = !model ? "" : model.replay ? `HISTORICAL · NONLIVE${replayState}` : model.current ? "CURRENT" : model.partial ? "PARTIAL · CURRENT DATA" : event.transport === "refresh_delayed" ? "REFRESH DELAYED" : event.transport === "disconnected" ? "FROZEN · DISCONNECTED" : model.backendReady && model.rankingMode === "degraded_bootstrap" ? "DEGRADED" : model.finalizing ? "FINALIZING" : model.warming ? "WARMING" : "NONCURRENT";
  const warmupProgress = model?.warming && event.transport === "connected" ? ` · ${model.hydrationProgress}${model.hydrationIssueText}` : "";
  const live = textElement(document, "div", model ? `${primaryState}${warmupProgress} · publication ${model.publicationID} · ${model.rows.length} ranked` : event.error ? `Scanner API disconnected: ${event.error}` : "Connecting to scanner API", "status-live");
  live.id = "status-live"; live.dataset.state = model?.current && !model.replay ? "current" : "warning"; header.append(title, live); main.append(header);
  const grid = element(document, "section", "status-grid"); grid.setAttribute("aria-label", "Scanner status");
  grid.append(statusItem(document, "Transport", event.transport, event.transport === "connected" ? "current" : "warning"),
    statusItem(document, "Process", model ? model.processLive ? model.replay ? "running" : "live" : "not live" : "unknown", model?.processLive ? "current" : "warning"),
    statusItem(document, "Backend", model ? model.backendReady ? "ready" : model.warming ? `${model.finalizing ? "finalizing" : "warming"} · ${model.hydrationProgress}${model.hydrationIssueText}` : model.readinessReason || "not ready" : "unknown", model?.backendReady ? "current" : "warning"),
    statusItem(document, "Ops sample", model ? model.sampleAccountingValid ? "accounting valid" : "accounting invalid" : "unknown", model?.sampleAccountingValid ? "current" : "warning"),
    statusItem(document, "Ranking", model ? `${model.rankingMode}${model.rankingReason ? ` · ${model.rankingReason}` : ""}` : "unknown"),
    statusItem(document, model?.replay ? "Replay time" : "Watermark", model?.replayLogicalTime || (model?.committedT ? `${new Date(model.committedT).toLocaleTimeString()} · ${model.watermarkLagMS} ms` : "unavailable")),
    statusItem(document, "T/Q", model ? `${model.tqPressure}${model.tqAggregateOnly ? " · aggregate only" : ""}${model.tqRetainedBoundHit ? " · retained bound" : ""} · ${model.tqUnknown} unknown` : "unknown", model?.tqPressure === "normal" && !model?.tqRetainedBoundHit && model?.tqUnknown === 0 ? "current" : "warning"),
    statusItem(document, "Sample", model ? `${new Date(model.sampledAt).toLocaleTimeString()} · #${model.sampleID}` : "—"));
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
    partial.textContent = `PARTIAL RANKING · current trusted marks ordered by Day % · qualification is not asserted · ${model.rankingReason || "symbol-local uncertainty"}`;
    main.append(partial);
  }
  const message = element(document, "div", "message"); message.id = "message";
  if (!model) message.textContent = "No valid scanner snapshot is available.";
  else if (model.rows.length === 0) message.textContent = model.rowsCurrent && model.rankingMode === "qualified_current" ? "No symbols currently qualify." : model.warming ? `Scanner warm-up in progress · ${model.hydrationProgress}${model.hydrationIssueText}.` : `No rows in retained noncurrent publication · ${model.rankingMode}${model.rankingReason ? ` · ${model.rankingReason}` : ""}.`;
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
