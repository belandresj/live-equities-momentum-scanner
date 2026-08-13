import http from "node:http";
import { validateSnapshot } from "./model.js";
import { snapshotFixture } from "./test-fixture.js";

const host = "127.0.0.1";
const port = Number.parseInt(process.env.SCANNER_FIXTURE_PORT || "18080", 10);
const allowedOrigin = process.env.SCANNER_UI_ORIGIN || "http://127.0.0.1:14173";
let state = "current";
let sampleSequence = 9n;

function fixture(name) {
  const snapshot = snapshotFixture(name === "empty" ? 0 : 20);
  snapshot.sample.id = String(++sampleSequence);
  snapshot.sample.sampled_at = new Date().toISOString().replace(/\.000Z$/, "Z").replace(/(\.\d*?[1-9])0+Z$/, "$1Z");
  if (name === "degraded") {
    snapshot.status.backend_ready = false; snapshot.status.ranking_current = false; snapshot.status.readiness_reason = "ranking_noncurrent";
    snapshot.ranking.mode = "degraded_bootstrap"; snapshot.ranking.reason = "incomplete_population";
    snapshot.accounting.population.universe_total = 21; snapshot.accounting.population.valid_prior_close = 21;
    snapshot.accounting.population.unknown_due_failure_or_fence = 1; snapshot.accounting.population.covered_population = 20; snapshot.accounting.population.unresolved_population = 1;
    snapshot.accounting.qualification.unresolved = 1; snapshot.accounting.uncertainty.bootstrap_origin = 1;
    snapshot.recovery.work.planned = "21"; snapshot.recovery.work.failed = "1";
    if (snapshot.accounting.population.unresolved_population !== 1 || snapshot.recovery.work.failed !== "1" || snapshot.ranking.known_rankable_count !== 20) throw new Error("degraded fixture lacks explicit unresolved population");
  }
  if (name === "tq") {
    snapshot.status.tq_pressure_mode = "aggregate_only"; snapshot.status.tq_shed = true;
    snapshot.tq.pressure_mode = "aggregate_only"; snapshot.tq.pressure_cause = "tq_retention_bound"; snapshot.tq.aggregate_only = true; snapshot.tq.shed = true; snapshot.tq.retained_bound_hit = true; snapshot.tq.unknown = 1; snapshot.tq.known_present = 19;
    for (const [index, row] of snapshot.rows.entries()) {
      row.tape_rate = { status: "pressure_shed", reason: "pressure", trade_coverage: false, one_second: { status: "pressure_shed", reason: "pressure", trades_per_second: null }, five_second: { status: "pressure_shed", reason: "pressure", trades_per_second: null }, timestamp_basis: "", lifecycle_records_observed: row.tape_rate.lifecycle_records_observed };
      row.spread = { status: "pressure_shed", reason: "pressure", quote_coverage: false, cents: null, basis_points: null, quote_age_ms: 0, quality: "" };
      row.tq_membership.provider_present = index !== 0;
      row.tq_membership.provider_membership_unknown = index === 0;
    }
  }
  if (name === "hostile" && snapshot.rows.length) {
    snapshot.rows[0].symbol = "<img src=x onerror=alert(1)>";
    snapshot.tq.desired_symbols[0] = snapshot.rows[0].symbol;
    snapshot.rows[0].activity = { status: "unavailable", reason: "history_incomplete", value_ratio: null };
  }
  validateSnapshot(snapshot);
  return snapshot;
}

const server = http.createServer((request, response) => {
  if (request.method === "POST" && request.url === "/__fixture/state") {
    let body = "";
    request.setEncoding("utf8"); request.on("data", chunk => { if (body.length < 64) body += chunk; });
    request.on("end", () => {
      if (!["current", "empty", "degraded", "tq", "hostile", "hang", "error"].includes(body)) { response.writeHead(400).end(); return; }
      state = body; response.writeHead(204).end();
    });
    return;
  }
  if (request.method !== "GET" || request.url !== "/api/v1/snapshot") { response.writeHead(404).end(); return; }
  response.setHeader("Access-Control-Allow-Origin", allowedOrigin); response.setHeader("Vary", "Origin"); response.setHeader("Cache-Control", "no-store"); response.setHeader("Content-Type", "application/json");
  if (state === "hang") return;
  if (state === "error") { response.writeHead(503).end('{"error":"fixture_unavailable"}'); return; }
  const body = JSON.stringify(fixture(state)); response.setHeader("Content-Length", Buffer.byteLength(body)); response.writeHead(200).end(body);
});
server.headersTimeout = 2000; server.requestTimeout = 5000; server.keepAliveTimeout = 1000;
server.listen(port, host, () => console.log(`fixture API listening on http://${host}:${port}`));

function shutdown() { server.close(() => process.exit(0)); setTimeout(() => process.exit(1), 3000).unref(); }
process.on("SIGINT", shutdown); process.on("SIGTERM", shutdown);
