import { PollController } from "/model.js";
import { fitScannerRows, renderDashboard } from "/render.js";

const config = globalThis.SCANNER_UI_CONFIG;
if (!config || typeof config.apiOrigin !== "string") throw new Error("dashboard configuration unavailable");

let observedTableShell = null;
const rowResizeObserver = new ResizeObserver(() => fitScannerRows(document));
function renderUpdate(event) {
  renderDashboard(document, event);
  const tableShell = document.getElementById("table-shell");
  if (tableShell !== observedTableShell) {
    if (observedTableShell) rowResizeObserver.unobserve(observedTableShell);
    observedTableShell = tableShell;
    if (observedTableShell) rowResizeObserver.observe(observedTableShell);
  }
  fitScannerRows(document);
}

const poller = new PollController({
  url: `${config.apiOrigin}/api/v2/snapshot`,
  pollMilliseconds: config.pollMilliseconds,
  requestTimeoutMilliseconds: config.requestTimeoutMilliseconds,
  onUpdate: renderUpdate,
});
poller.start();
globalThis.addEventListener("pagehide", () => { rowResizeObserver.disconnect(); poller.stop(); }, { once: true });
