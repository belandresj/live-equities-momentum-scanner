function element(document, tag, className = "") {
  const node = document.createElement(tag); if (className) node.className = className; return node;
}
function textElement(document, tag, value, className = "") { const node = element(document, tag, className); node.textContent = value; return node; }
const COLUMN_DEFINITIONS = [
  ["RANK", "Current scanner rank and movement compared with approximately 60 seconds ago."],
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
const TABLE_COLUMN_CLASSES = [
  "rank-column", "symbol-column", "context-column", "context-column", "context-column",
  "location-column", "location-column", "location-column",
  "momentum-column", "momentum-column",
  "execution-column", "execution-column",
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
function rankCell(document, row, overallCurrent) {
  const movement = row.rankMovement || { direction: "none", text: "", label: "60-second rank movement unavailable" };
  const td = element(document, "td", "rank-cell"); td.dataset.state = overallCurrent ? "current" : "retained"; td.dataset.fieldState = "current";
  td.tabIndex = 0; td.dataset.focusKey = `${row.symbol}:rank`;
  td.setAttribute("aria-label", `Current rank ${row.rank}. ${movement.label}.`);
  const metadata = element(document, "span", "rank-metadata");
  metadata.append(textElement(document, "span", `#${row.rank}`, "current-rank"));
  const movementNode = textElement(document, "span", movement.text, "rank-movement"); movementNode.dataset.direction = movement.direction; metadata.append(movementNode);
  td.append(metadata);
  return td;
}
function statusDetail(document, label, id) {
  const item = element(document, "div", "status-detail"); item.append(textElement(document, "span", label));
  const value = textElement(document, "strong", "unknown"); value.id = id; item.append(value); return item;
}
function buildShell(document) {
  const main = element(document, "main", "dashboard"); main.id = "dashboard-root";
  const header = element(document, "header", "app-bar"); header.append(textElement(document, "h1", "MOMENTUM SCANNER"));
  const status = element(document, "details", "system-status"); status.id = "system-status";
  const summary = element(document, "summary", "status-summary"); summary.id = "status-summary";
  const live = textElement(document, "span", "Connecting to scanner API", "status-primary"); live.id = "status-live"; live.setAttribute("role", "status"); live.setAttribute("aria-live", "polite");
  const alert = textElement(document, "span", "", "status-alert"); alert.id = "status-alert";
  const chevron = textElement(document, "span", "⌄", "status-chevron"); chevron.setAttribute("aria-hidden", "true");
  summary.append(live, alert, chevron); status.append(summary);
  const panel = element(document, "div", "status-panel"); panel.setAttribute("aria-label", "Scanner status details");
  const scannerItem = statusDetail(document, "Scanner", "scanner-status"), aggregatesItem = statusDetail(document, "Aggregates", "aggregate-status"), tradesItem = statusDetail(document, "Trades", "trade-status"), quotesItem = statusDetail(document, "Quotes", "quote-status");
  panel.append(scannerItem, aggregatesItem, tradesItem, quotesItem); status.append(panel); header.append(status); main.append(header);
  const messages = element(document, "div", "message-stack"); messages.id = "messages"; main.append(messages);
  const scannerPanel = element(document, "section", "scanner-panel"); scannerPanel.id = "scanner-panel"; scannerPanel.setAttribute("aria-label", "Momentum scanner results");
  const shell = element(document, "div", "table-shell"), table = element(document, "table"); shell.id = "table-shell"; table.id = "scanner-table";
  const caption = textElement(document, "caption", "Server-ranked top 20 qualifying equities"); caption.id = "scanner-caption";
  const columnGroup = element(document, "colgroup"); for (const className of TABLE_COLUMN_CLASSES) columnGroup.append(element(document, "col", className)); table.append(caption, columnGroup);
  const thead = element(document, "thead"), groups = element(document, "tr", "group-header"), leafHeader = element(document, "tr", "leaf-header");
  for (const [label, span] of [["CONTEXT", 5], ["LOCATION", 3], ["MOMENTUM", 2], ["TAPE / EXECUTION", 2]]) { const th = textElement(document, "th", label); th.setAttribute("scope", "colgroup"); th.setAttribute("colspan", String(span)); groups.append(th); }
  for (const [label, description] of COLUMN_DEFINITIONS) { const th = textElement(document, "th", label); th.setAttribute("scope", "col"); th.setAttribute("data-tooltip", description); leafHeader.append(th); }
  thead.append(groups, leafHeader); table.append(thead);
  const rows = element(document, "tbody"); rows.id = "rows"; table.append(rows); shell.append(table); scannerPanel.append(shell); main.append(scannerPanel);
  return { main, status, summary, live, alert, scanner: scannerItem.children[1], aggregates: aggregatesItem.children[1], trades: tradesItem.children[1], quotes: quotesItem.children[1], messages, scannerPanel, tableShell: shell, table, caption, thead, rows };
}
function existingShell(document) {
  const main = document.getElementById?.("dashboard-root"); if (!main) return null;
  return { main, status: document.getElementById("system-status"), summary: document.getElementById("status-summary"), live: document.getElementById("status-live"), alert: document.getElementById("status-alert"), scanner: document.getElementById("scanner-status"), aggregates: document.getElementById("aggregate-status"), trades: document.getElementById("trade-status"), quotes: document.getElementById("quote-status"), messages: document.getElementById("messages"), scannerPanel: document.getElementById("scanner-panel"), tableShell: document.getElementById("table-shell"), table: document.getElementById("scanner-table"), caption: document.getElementById("scanner-caption"), thead: document.getElementById("scanner-table")?.querySelector?.("thead"), rows: document.getElementById("rows") };
}
export function scannerRowHeight(shellHeight, headerHeight, rowCount, cap = 50) {
  if (![shellHeight, headerHeight, rowCount, cap].every(Number.isFinite) || shellHeight < 0 || headerHeight < 0 || rowCount <= 0 || cap <= 0) return null;
  return Math.min(cap, Math.max(0, shellHeight - headerHeight - 1) / rowCount);
}
export function fitScannerRows(document) {
  const refs = existingShell(document); if (!refs?.tableShell || !refs.thead || !refs.rows) return null;
  const height = scannerRowHeight(refs.tableShell.clientHeight, refs.thead.getBoundingClientRect().height, refs.rows.children.length);
  if (height === null) refs.tableShell.style.removeProperty("--scanner-row-height");
  else refs.tableShell.style.setProperty("--scanner-row-height", `${height}px`);
  return height;
}
function buildRows(document, model, rowsCurrent) {
  if (!model) return [];
  const rows = [];
  for (const row of model.rows) {
    const tr = element(document, "tr"); tr.dataset.publicationState = rowsCurrent ? "current" : "noncurrent"; tr.setAttribute("aria-label", `Server rank ${row.rank}, ${row.symbol}`); const key = row.symbol;
    tr.append(rankCell(document, row, rowsCurrent), cell(document, row.symbol, "current", `server rank ${row.rank}`, rowsCurrent, 0, `${key}:symbol`),
      cell(document, row.float.text, row.float.state, row.float.detail, rowsCurrent, 0, `${key}:float`, row.float.cyanIntensity === null ? "" : "float", row.float.cyanIntensity), cell(document, row.volume.text, row.volume.state, row.volume.detail, rowsCurrent, 0, `${key}:volume`, row.volume.colorIntensity === null ? "" : "volume-turnover", row.volume.colorIntensity), cell(document, row.last, "current", `mark age ${row.markAgeMS} ms`, rowsCurrent, 0, `${key}:last`),
      cell(document, row.day, "current", "From Close %: server-published ranking value; color is relative to the displayed snapshot", rowsCurrent, 0, `${key}:day`, row.dayColor === null ? "" : "day", row.dayColor, row.dayTextColor), cell(document, row.fromOpen.text, row.fromOpen.state, row.fromOpen.detail, rowsCurrent, row.fromOpen.band, `${key}:fromopen`, row.fromOpen.colorPosition === null ? "" : "from-open", row.fromOpen.colorPosition, row.fromOpen.textColor), cell(document, row.dayRange.text, row.dayRange.state, row.dayRange.detail, rowsCurrent, row.dayRange.band, `${key}:dayrange`, "", null, row.dayRange.textColor),
      cell(document, row.activity.text, row.activity.state, row.activity.detail, rowsCurrent, row.activity.band, `${key}:activity`, "", null, row.activity.textColor), cell(document, row.move.text, row.move.state, row.move.detail, rowsCurrent, row.move.band, `${key}:move`, "", null, row.move.textColor),
      cell(document, row.tape.primary, row.tape.state, row.tape.detail, rowsCurrent, 0, `${key}:tape`, "", null, row.tape.textColor), cell(document, row.spread.primary, row.spread.state, row.spread.detail, rowsCurrent, 0, `${key}:spread`, "", null, row.spread.textColor));
    rows.push(tr);
  }
  return rows;
}
function buildMessages(document, model, phaseLabel, workProgress) {
  const messages = [];
  if (model?.lifecycle === "suppressed") { const suppressed = element(document, "div", "message"); suppressed.dataset.state = "suppressed"; suppressed.textContent = `SCANNER SUPPRESSED · ${model.lifecycleReason || "integrity failure"} · ${model.suppression || "restart required"}${model.integrityFailure ? ` · ${model.integrityFailure.category} at engine sequence ${model.integrityFailure.engine_sequence}` : ""}`; messages.push(suppressed); }
  if (model?.partial) { const partial = element(document, "div", "message"); partial.dataset.state = "partial"; partial.textContent = `PARTIAL RANKING · current trusted marks ordered by From Close % · qualification is not asserted · ${model.rankingReason || "symbol-local uncertainty"}`; messages.push(partial); }
  if (!model || model.rows.length === 0) {
    if (workProgress) return messages;
    const message = element(document, "div", "message"); message.id = "message";
    if (!model) message.textContent = "No valid scanner snapshot is available.";
    else message.textContent = model.rowsCurrent && model.rankingMode === "qualified_current" ? "No symbols currently qualify." : phaseLabel ? `${phaseLabel.toLowerCase()}${workProgress}. Ranking remains noncurrent until the aggregate fence reconciles.` : `No rows in retained noncurrent publication · ${model.rankingMode}${model.rankingReason ? ` · ${model.rankingReason}` : ""}.`;
    messages.push(message);
  }
  return messages;
}

function traderStatus(event, model) {
  if (!model) return event.error
    ? { label: "DISCONNECTED", scanner: "Disconnected · no current scanner snapshot", aggregates: "Status unavailable" }
    : { label: "CONNECTING", scanner: "Connecting to scanner", aggregates: "Establishing connection" };
  if (event.transport === "disconnected") return { label: "DISCONNECTED", scanner: "Disconnected · displayed rows are frozen", aggregates: "Status retained from the last update" };
  if (event.transport === "refresh_delayed") return { label: "DELAYED", scanner: "Delayed · displayed rows are retained", aggregates: "Status retained from the last update" };
  if (model.replay) return { label: "HISTORICAL", scanner: "Historical data · not live", aggregates: "Historical aggregate playback" };
  if (model.lifecycle === "ended") return { label: "SESSION ENDED", scanner: "Session ended · final snapshot retained", aggregates: "Session ended" };
  if (model.lifecycle === "suppressed") return { label: "UNAVAILABLE", scanner: "Unavailable · integrity protection active", aggregates: "Aggregate ranking unavailable" };
  if (model.lifecycle === "awaiting_session") return { label: "WAITING FOR SESSION", scanner: "Waiting for the scanner session", aggregates: "Waiting for session start" };
  if (model.current) return { label: "LIVE", scanner: "Live · exact qualified ranking", aggregates: "Streaming · ranking current" };
  if (model.partial || model.backendReady && ["degraded_bootstrap", "degraded_current"].includes(model.rankingMode)) return { label: "PARTIAL", scanner: "Partial · qualification is incomplete", aggregates: "Streaming · partial population" };
  if (model.lifecycle === "recovering" || model.recovering) return { label: "RECOVERING", scanner: "Recovering · displayed ranking is not current", aggregates: aggregateProgress(model) };
  if (["initializing", "awaiting_aggregate_ack", "hydrating"].includes(model.lifecycle) || model.phase) return { label: "STARTING", scanner: "Starting · ranking is not yet current", aggregates: aggregateProgress(model) };
  return { label: "UNAVAILABLE", scanner: "Unavailable · no current ranking", aggregates: "Aggregate ranking unavailable" };
}

function aggregateProgress(model) {
  const progress = model.hydrationProgress ? ` · ${model.hydrationProgress}${model.hydrationIssueText}` : "";
  return ({
    connecting: "Connecting to aggregate stream",
    subscribing: "Subscribing to aggregates",
    preparing_hydration: "Preparing history sync",
    hydrating: `Syncing history${progress}`,
    finalizing_hydration: `Finalizing history sync${progress}`,
    reconnecting: "Reconnecting aggregate stream",
    resubscribing: "Resubscribing to aggregates",
    preparing_recovery: "Preparing aggregate recovery",
    recovering: `Repairing aggregate gap${progress}`,
    retrying: `Retrying aggregate recovery${progress}`,
    finalizing_recovery: `Finalizing aggregate recovery${progress}`,
  })[model.phase] || "Establishing aggregate ranking";
}

function coverageSummary(model, field) {
  if (!model) return "Status unavailable";
  if (model.partial) return "Not active for partial ranking";
  if (model.rows.length === 0) return model.current ? "No selected symbols" : "Not active";
  const counts = new Map();
  for (const row of model.rows) {
    const state = row[field].state;
    counts.set(state, (counts.get(state) || 0) + 1);
  }
  const selected = model.rows.length - (counts.get("unselected") || 0);
  if (selected === 0) return "Not active";
  const labels = [["current", "current"], ["warming", "warming"], ["stale", "stale"], ["pressure_shed", "shed"], ["unavailable", "unavailable"], ["invalid", "invalid"], ["unknown", "unknown"]];
  const parts = [];
  for (const [state, label] of labels) if (counts.get(state)) parts.push(state === "current" ? `${counts.get(state)}/${selected} ${label}` : `${counts.get(state)} ${label}`);
  return parts.join(" · ") || "Not active";
}

export function renderDashboard(document, event, options = {}) {
  const focusKey = document.activeElement?.dataset?.focusKey || "", model = event.model;
  const phaseLabel = !model ? "" : ({ connecting: "CONNECTING", subscribing: "SUBSCRIBING", hydrating: "HYDRATING", preparing_hydration: "PREPARING HYDRATION", finalizing_hydration: "FINALIZING HYDRATION", reconnecting: "RECONNECTING", resubscribing: "RESUBSCRIBING", preparing_recovery: "PREPARING RECOVERY", recovering: "RECOVERING", retrying: "RETRYING RECOVERY", finalizing_recovery: "FINALIZING RECOVERY" })[model.phase] || "";
  const workProgress = model && event.transport === "connected" && ["hydrating", "recovering", "retrying", "finalizing_hydration", "finalizing_recovery"].includes(model.phase) ? ` · generation ${model.recoveryGeneration} · ${model.hydrationProgress}${model.hydrationIssueText}` : "";
  const statusView = traderStatus(event, model), liveText = statusView.label;
  const tqAttention = Boolean(model && (model.tqPressure !== "normal" || model.tqAggregateOnly || model.tqRetainedBoundHit || model.tqUnknown > 0));
  const rowsCurrent = Boolean(model?.rowsCurrent && event.transport === "connected"), nextRows = buildRows(document, model, rowsCurrent), nextMessages = buildMessages(document, model, phaseLabel, workProgress);
  let refs = existingShell(document), mount = document.getElementById?.("app"); const isInitial = !refs; if (!refs) refs = buildShell(document);
  options.beforeCommit?.(refs.main);
  refs.live.textContent = liveText; refs.live.dataset.state = model?.current && !model.replay && event.transport === "connected" ? "current" : "warning";
  refs.alert.textContent = tqAttention ? "TAPE / QUOTES DEGRADED" : ""; refs.alert.hidden = !tqAttention; refs.alert.dataset.state = "warning";
  refs.summary.setAttribute("aria-label", `${liveText}${tqAttention ? ". Tape and quotes degraded" : ""}. Activate for scanner status details.`);
  refs.scanner.textContent = statusView.scanner; refs.scanner.dataset.state = model?.current && event.transport === "connected" ? "current" : "warning";
  refs.aggregates.textContent = statusView.aggregates; refs.aggregates.dataset.state = model?.current && event.transport === "connected" ? "current" : "warning";
  refs.trades.textContent = coverageSummary(model, "tape"); refs.trades.dataset.state = tqAttention ? "warning" : "current";
  refs.quotes.textContent = coverageSummary(model, "spread"); refs.quotes.dataset.state = tqAttention ? "warning" : "current";
  refs.messages.replaceChildren(...nextMessages); refs.scannerPanel.hidden = !model; refs.table.dataset.publicationState = rowsCurrent ? "current" : "noncurrent";
  refs.caption.textContent = model?.partial ? "Server-ranked top 20 trusted marks by From Close %, qualification not asserted" : "Server-ranked top 20 qualifying equities"; refs.rows.replaceChildren(...nextRows);
  if (!mount) { mount = element(document, "div"); mount.id = "app"; document.body.append(mount); } if (isInitial) mount.replaceChildren(refs.main);
  let announcer = document.getElementById?.("announcer"); if (!announcer) { announcer = element(document, "div", "sr-only"); announcer.id = "announcer"; announcer.setAttribute("role", "status"); announcer.setAttribute("aria-live", "polite"); document.body.append(announcer); }
  announcer.textContent = tqAttention ? `${liveText}. Tape and quotes degraded.` : liveText;
  if (focusKey) { try { refs.main.querySelector?.(`[data-focus-key="${CSS.escape(focusKey)}"]`)?.focus({ preventScroll: true }); } catch { /* coherent view is already committed */ } }
  return refs.main;
}
