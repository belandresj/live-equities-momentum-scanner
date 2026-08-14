import { PollController } from "/model.js";
import { renderDashboard } from "/render.js";

const config = globalThis.SCANNER_UI_CONFIG;
if (!config || typeof config.apiOrigin !== "string") throw new Error("dashboard configuration unavailable");

const poller = new PollController({
  url: `${config.apiOrigin}/api/v2/snapshot`,
  pollMilliseconds: config.pollMilliseconds,
  requestTimeoutMilliseconds: config.requestTimeoutMilliseconds,
  onUpdate: event => renderDashboard(document, event),
});
poller.start();
globalThis.addEventListener("pagehide", () => poller.stop(), { once: true });
