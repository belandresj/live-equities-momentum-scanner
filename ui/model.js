const MAX_RESPONSE_BYTES = 1 << 20;
const MAX_SAFE = Number.MAX_SAFE_INTEGER;
const DECIMAL = /^(0|[1-9][0-9]*)$/;
const TRADING_DATE = /^(\d{4})-(\d{2})-(\d{2})$/;

const KNOWN_FIELD_STATUS = new Set(["warming", "current", "unavailable", "invalid"]);
const KNOWN_FLOAT_STATUS = new Set(["current", "stale", "unavailable", "invalid"]);
const KNOWN_FIELD_REASON = new Set(["", "before_first_print", "history_incomplete", "prior_close_unavailable", "no_aggregate_in_target", "rolling_warmup", "reference_warmup", "zero_width", "historical_conflict", "invalid_input", "state_bound_exceeded"]);
const KNOWN_TQ_STATUS = new Set(["unselected", "warming", "current", "stale", "unavailable", "invalid", "pressure_shed"]);
const KNOWN_TQ_REASON = new Set(["", "coverage", "channel_unconfirmed", "control_error", "coverage_warming", "five_second_warming", "qualifying_original_prints", "unequal_repeat", "one_sided_quote", "crossed_quote", "stale_quote", "insufficient_coverage", "pressure", "replay_unavailable"]);
const CURRENT_LIFECYCLE = new Set(["live", "hydrating"]);
const BACKEND_READY_RANKING_MODE = new Set(["qualified_current", "degraded_bootstrap", "degraded_current"]);
const TIMESTAMP_BASIS = new Set(["", "none", "participant", "sip_fallback", "mixed"]);
const SPREAD_QUALITY = new Set(["", "reviewed_ordinary", "known_special", "unclassified"]);
const PRESSURE_CAUSE = new Set(["", "waiting_frames", "waiting_bytes", "oldest_waiting_frame", "aggregate_watermark_lag", "capacity_drop", "tq_retention_bound", "transport_accounting_loss"]);
const EVALUATOR_INTEGRITY_CATEGORY = new Set(["candidate_target_mismatch", "support_contradiction", "population_accounting", "qualification_accounting", "uncertainty_accounting", "feature_accounting", "ranking_projection", "ranking_row", "tq_intent", "unknown_evaluator_integrity"]);
const LIFECYCLE = new Set(["initializing", "awaiting_session", "awaiting_aggregate_ack", "hydrating", "live", "recovering", "replaying", "suppressed", "ended"]);
const LIFECYCLE_REASON = new Set(["", "binding_before_session", "binding_in_session", "binding_after_session", "session_start_without_aggregate_ack", "session_end", "controlled_stop", "sequence_exhaustion", "clock_regression", "canonical_integrity", "publication_integrity", "accounting_integrity", "closed", "replay_start", "replay_end", "replay_requested_end", "replay_failure", "aggregate_acknowledged", "aggregate_acknowledged_at_session_start", "aggregate_epoch_lost", "ingress_integrity", "hydration_complete", "recovery_exhausted", "scheduled_recovery"]);
const SUPPRESSION = new Set(["", "same_binding_recovery_allowed", "clean_reinitialization_required", "restart_required", "terminal_replay_failure"]);
const READINESS_REASON = new Set(["", "runtime_unavailable", "binding_mismatch", "not_live_mode", "lifecycle_not_ready", "suppressed", "aggregate_unacknowledged", "fence_pending", "ranking_noncurrent", "watermark_missing", "watermark_stale", "accounting_invalid"]);
const RANKING_MODE = new Set(["unavailable", "qualified_current", "degraded_bootstrap", "degraded_current", "stale", "suppressed"]);
const RANKING_REASON = new Set(["", "no_committed_watermark", "no_trusted_marks", "incomplete_population", "qualification_incomplete", "global_suppression", "replay_warming"]);

function fail(message) { throw new Error(message); }
function object(value, name) {
  if (value === null || typeof value !== "object" || Array.isArray(value)) fail(`${name} must be an object`);
  return value;
}
function fields(value, names, label) {
  object(value, label);
  for (const name of names) if (!Object.hasOwn(value, name)) fail(`${label}.${name} is required`);
}
function string(value, name) { if (typeof value !== "string") fail(`${name} must be a string`); }
function bool(value, name) { if (typeof value !== "boolean") fail(`${name} must be a boolean`); }
function uint(value, name) { if (!Number.isSafeInteger(value) || value < 0) fail(`${name} must be an exact unsigned integer`); }
function finite(value, name) { if (typeof value !== "number" || !Number.isFinite(value)) fail(`${name} must be finite`); }
function decimal(value, name) { string(value, name); if (!DECIMAL.test(value) || BigInt(value) > 18446744073709551615n) fail(`${name} must be uint64 decimal`); }
function timestamp(value, name) {
  string(value, name);
  const match = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2}):(\d{2})(?:\.(\d{1,9}))?Z$/.exec(value);
  if (!match || match[7]?.endsWith("0") || !calendar(Number(match[1]), Number(match[2]), Number(match[3])) || Number(match[4]) > 23 || Number(match[5]) > 59 || Number(match[6]) > 59) fail(`${name} must be canonical UTC RFC3339Nano`);
}
function optionalTimestamp(value, name) { if (value !== null) timestamp(value, name); }
function sumDecimal(total, ...parts) { return BigInt(total) === parts.reduce((sum, value) => sum + BigInt(value), 0n); }
function calendar(year, month, day) {
  if (month < 1 || month > 12 || day < 1) return false;
  const leap = year % 4 === 0 && (year % 100 !== 0 || year % 400 === 0);
  const days = [31, leap ? 29 : 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31];
  return day <= days[month - 1];
}

const POPULATION_FIELDS = ["universe_total", "valid_prior_close", "invalid_or_missing_prior_close", "trusted_rankable_mark", "trusted_below_price_mark", "no_print_through_t", "invalid_mark", "unknown_due_failure_or_fence", "covered_population", "unresolved_population"];
const QUALIFICATION_FIELDS = ["not_yet_passed", "provisional", "finalized", "unresolved"];
const UNCERTAINTY_FIELDS = ["bootstrap_origin", "post_bootstrap_gap", "local_invalid"];
const POPULATION_TRANSITION_FIELDS = ["bootstrap_unknown", "trusted_by_later_live_mark", "no_later_eligible_mark", "latest_mark_not_live_authority", "no_strictly_older_localized_conflict", "conflict_at_or_after_mark", "invalid_at_or_after_mark", "incomplete_post_mark_coverage"];
const WORK_FIELDS = ["planned", "open", "completed_value", "completed_empty", "failed", "canceled", "fenced"];
const RECOVERY_ROW_FIELDS = ["consumed", "inserted", "duplicate", "conflict_or_withdrawal", "rejected", "fenced", "integrity"];
const FACT_FIELDS = ["consumed", "applied", "duplicate", "rejected", "fenced", "pressure_shed", "integrity", "normalized_trades", "normalized_quotes", "applied_trades", "applied_quotes", "pressure_shed_trades", "pressure_shed_quotes"];
const COMMAND_FIELDS = ["issued", "pending_write", "written", "failed", "fenced", "result_fenced"];
const CHECKPOINT_DECIMAL_FIELDS = ["eligible", "pressure_deferred", "projection_started", "projection_in_progress", "projected", "projection_rejected", "submit_rejected", "submitted", "outstanding", "in_progress", "pending", "completed", "failed", "canceled", "superseded", "artifact_bytes"];
const CHECKPOINT_UINT_FIELDS = ["usable_age_ms", "projection_total_ms", "projection_lock_ms", "write_ms", "encode_ms", "reopen_validation_ms"];
const OPERATION_UINT_FIELDS = ["queue_capacity_frames", "queue_current_frames", "queue_high_frames", "queue_current_bytes", "queue_high_bytes", "mean_processing_delay_ms", "max_processing_delay_ms", "max_processing_delay_one_second_ms", "goroutines"];
const OPERATION_DECIMAL_FIELDS = ["deliveries", "consumer_deferred", "heap_alloc_bytes", "heap_in_use_bytes", "connection_recovery_attempts"];

function validateMeasurement(value, name, field) {
  fields(value, ["status", "reason", "value_ratio"], name);
  string(value.status, `${name}.status`); string(value.reason, `${name}.reason`);
  if (value.value_ratio !== null) finite(value.value_ratio, `${name}.value_ratio`);
  const reasons = {
    from_open: { warming: ["before_first_print"], unavailable: ["history_incomplete"] },
    day_range: { warming: [], unavailable: ["before_first_print", "history_incomplete", "zero_width"] },
    activity_30s: { warming: ["reference_warmup"], unavailable: ["history_incomplete"] },
    move_30s: { warming: ["before_first_print", "rolling_warmup"], unavailable: ["history_incomplete", "no_aggregate_in_target"] },
  }[field];
  if (!KNOWN_FIELD_STATUS.has(value.status) || !KNOWN_FIELD_REASON.has(value.reason) || !reasons) fail(`${name} unknown status/reason`);
  const legal = value.status === "current" ? value.reason === "" && value.value_ratio !== null
    : value.value_ratio === null && (value.status === "invalid" ? ["historical_conflict", "invalid_input", "state_bound_exceeded"].includes(value.reason) : reasons[value.status].includes(value.reason));
  if (!legal) fail(`${name} status/reason/value conflict`);
  if ((field === "day_range" || field === "activity_30s") && value.value_ratio !== null && (value.value_ratio < 0 || value.value_ratio > 1)) fail(`${name} value outside [0,1]`);
}

function validateShares(value, name) {
  fields(value, ["status", "reason", "value_shares"], name);
  string(value.status, `${name}.status`); string(value.reason, `${name}.reason`);
  if (value.value_shares !== null) finite(value.value_shares, `${name}.value_shares`);
  const legal = value.status === "current" ? value.reason === "" && value.value_shares !== null && value.value_shares >= 0
    : value.value_shares === null && (value.status === "unavailable" && value.reason === "history_incomplete" || value.status === "invalid" && ["historical_conflict", "invalid_input", "state_bound_exceeded"].includes(value.reason));
  if (!legal) fail(`${name} status/reason/value conflict`);
}

function validateFloat(value, name) {
  fields(value, ["status", "reason", "value_shares", "percent_ratio", "provider", "effective_date", "retrieved_at", "provenance"], name);
  string(value.status, `${name}.status`); string(value.reason, `${name}.reason`); string(value.provider, `${name}.provider`); string(value.provenance, `${name}.provenance`);
  if (value.value_shares !== null) finite(value.value_shares, `${name}.value_shares`);
  if (value.percent_ratio !== null) finite(value.percent_ratio, `${name}.percent_ratio`);
  if (value.effective_date !== null) { string(value.effective_date, `${name}.effective_date`); const date = TRADING_DATE.exec(value.effective_date); if (!date || !calendar(Number(date[1]), Number(date[2]), Number(date[3]))) fail(`${name}.effective_date invalid`); }
  optionalTimestamp(value.retrieved_at, `${name}.retrieved_at`);
  const available = value.status === "current" || value.status === "stale";
  if (!KNOWN_FLOAT_STATUS.has(value.status)) fail(`${name} unknown status`);
  const empty = value.value_shares === null && value.percent_ratio === null && value.provider === "" && value.effective_date === null && value.retrieved_at === null && value.provenance === "";
  const legalAvailable = value.value_shares !== null && value.value_shares > 0 && value.provider === "massive-stocks-float-experimental" && value.retrieved_at !== null && (value.percent_ratio === null || value.percent_ratio >= 0 && value.percent_ratio <= 1) &&
    (value.status === "current" ? value.reason === "" && value.provenance === "fresh" : value.reason === "cached_fallback" && value.provenance === "cache");
  const legalEmpty = empty && (value.status === "unavailable" && value.reason === "not_available" || value.status === "invalid" && value.reason === "invalid_provenance");
  if (available ? !legalAvailable : !legalEmpty) fail(`${name} provenance/status/value conflict`);
}

function validateTape(value, name) {
  fields(value, ["status", "reason", "trade_coverage", "trades_per_second", "timestamp_basis", "lifecycle_records_observed"], name);
  string(value.status, `${name}.status`); string(value.reason, `${name}.reason`); bool(value.trade_coverage, `${name}.trade_coverage`); string(value.timestamp_basis, `${name}.timestamp_basis`); bool(value.lifecycle_records_observed, `${name}.lifecycle_records_observed`);
  if (value.trades_per_second !== null) { finite(value.trades_per_second, `${name}.trades_per_second`); if (value.trades_per_second < 0) fail(`${name} negative rate`); }
  if (!TIMESTAMP_BASIS.has(value.timestamp_basis)) fail(`${name} unknown timestamp basis`);
  if (!KNOWN_TQ_STATUS.has(value.status) || !KNOWN_TQ_REASON.has(value.reason)) fail(`${name} unknown status/reason`);
  let legal = false;
  switch (value.status) {
    case "unselected": legal = !value.trade_coverage && value.reason === ""; break;
    case "warming": legal = value.trade_coverage && ["coverage_warming", "five_second_warming"].includes(value.reason); break;
    case "current": legal = value.trade_coverage && value.reason === "qualifying_original_prints"; break;
    case "unavailable": legal = ["coverage", "replay_unavailable"].includes(value.reason); break;
    case "invalid": legal = value.trade_coverage && value.reason === "unequal_repeat"; break;
    case "pressure_shed": legal = value.reason === "pressure"; break;
  }
  legal &&= (value.status === "current") === (value.trades_per_second !== null);
  if (!legal) fail(`${name} trust tuple conflict`);
}

function validateSpread(value, name, replay) {
  fields(value, ["status", "reason", "quote_coverage", "cents", "basis_points", "quote_age_ms", "quality"], name);
  string(value.status, `${name}.status`); string(value.reason, `${name}.reason`); bool(value.quote_coverage, `${name}.quote_coverage`); uint(value.quote_age_ms, `${name}.quote_age_ms`); string(value.quality, `${name}.quality`);
  if (!SPREAD_QUALITY.has(value.quality)) fail(`${name} unknown quality`);
  if ((value.cents === null) !== (value.basis_points === null)) fail(`${name} value pair conflict`);
  if (value.cents !== null) {
    finite(value.cents, `${name}.cents`); finite(value.basis_points, `${name}.basis_points`);
    if (value.cents < 0 || value.basis_points < 0) fail(`${name} status/value conflict: negative spread`);
  }
  if ((value.status === "current" || value.status === "stale") !== (value.cents !== null)) fail(`${name} status/value conflict`);
  if (!KNOWN_TQ_STATUS.has(value.status) || !KNOWN_TQ_REASON.has(value.reason)) fail(`${name} unknown status/reason`);
  let legal = false;
  switch (value.status) {
    case "unselected": legal = !value.quote_coverage && value.reason === "" && value.quote_age_ms === 0 && value.quality === ""; break;
    case "warming": legal = value.quote_coverage && value.reason === "coverage_warming"; break;
    case "current": legal = value.quote_coverage && value.reason === ""; break;
    case "stale": legal = value.quote_coverage && value.reason === "stale_quote"; break;
    case "invalid": legal = value.quote_coverage && value.reason === "crossed_quote"; break;
    case "unavailable": legal = value.reason === "coverage" || value.reason === "replay_unavailable" || value.quote_coverage && value.reason === "one_sided_quote"; break;
    case "pressure_shed": legal = value.reason === "pressure"; break;
  }
  if (!legal) fail(`${name} trust tuple conflict`);
}

function validateRow(row, index, seen) {
  const name = `rows[${index}]`;
  fields(row, ["rank", "symbol", "float", "volume", "last_usd", "day_change_ratio", "mark_age_ms", "from_open_change", "day_range_position", "activity_30s", "move_30s", "tape_5s", "spread", "tq_membership"], name);
  uint(row.rank, `${name}.rank`); string(row.symbol, `${name}.symbol`); finite(row.last_usd, `${name}.last_usd`); finite(row.day_change_ratio, `${name}.day_change_ratio`); uint(row.mark_age_ms, `${name}.mark_age_ms`);
  if (row.rank !== index + 1 || row.symbol === "" || seen.has(row.symbol)) fail(`${name} rank/symbol invalid`);
  seen.add(row.symbol);
  validateFloat(row.float, `${name}.float`); validateShares(row.volume, `${name}.volume`);
  validateMeasurement(row.from_open_change, `${name}.from_open_change`, "from_open");
  validateMeasurement(row.day_range_position, `${name}.day_range_position`, "day_range");
  validateMeasurement(row.activity_30s, `${name}.activity_30s`, "activity_30s");
  validateMeasurement(row.move_30s, `${name}.move_30s`, "move_30s");
  validateTape(row.tape_5s, `${name}.tape_5s`);
  validateSpread(row.spread, `${name}.spread`, false);
  fields(row.tq_membership, ["desired", "provider_present", "provider_membership_unknown"], `${name}.tq_membership`);
  bool(row.tq_membership.desired, `${name}.tq_membership.desired`); bool(row.tq_membership.provider_present, `${name}.tq_membership.provider_present`); bool(row.tq_membership.provider_membership_unknown, `${name}.tq_membership.provider_membership_unknown`);
  if (row.tq_membership.provider_present !== (row.tape_5s.trade_coverage && row.spread.quote_coverage) || row.tq_membership.provider_membership_unknown !== (row.tq_membership.desired && !row.tq_membership.provider_present)) fail(`${name}.tq_membership confirmation conflict`);
}

export function validateSnapshot(snapshot) {
  fields(snapshot, ["schema_version", "sample", "publication", "status", "ranking", "rows", "accounting", "recovery", "tq", "checkpoint", "operations"], "root");
  if (snapshot.schema_version !== "scanner.snapshot.v2") fail("unsupported schema");
  fields(snapshot.sample, ["id", "sampled_at"], "sample"); decimal(snapshot.sample.id, "sample.id"); timestamp(snapshot.sample.sampled_at, "sample.sampled_at");
  if (snapshot.sample.id === "0") fail("invalid sample identity");
  const p = snapshot.publication;
  fields(p, ["id", "binding_identity", "trading_date", "run_mode", "lifecycle", "lifecycle_reason", "suppression", "generated_at", "committed_t", "last_engine_sequence", "connection_epoch", "connection_active", "aggregate_acknowledged", "aggregate_ack_position", "hydration_fence"], "publication");
  for (const name of ["id", "last_engine_sequence", "connection_epoch"]) decimal(p[name], `publication.${name}`);
  for (const name of ["binding_identity", "trading_date", "run_mode", "lifecycle", "lifecycle_reason", "suppression"]) string(p[name], `publication.${name}`);
  const date = TRADING_DATE.exec(p.trading_date);
  if (p.id === "0" || p.binding_identity === "" || !date || !calendar(Number(date[1]), Number(date[2]), Number(date[3]))) fail("invalid publication identity");
  if (p.run_mode !== "live" && p.run_mode !== "replay") fail("unknown publication run mode");
  if (!LIFECYCLE.has(p.lifecycle) || !LIFECYCLE_REASON.has(p.lifecycle_reason) || !SUPPRESSION.has(p.suppression)) fail("unknown publication lifecycle meaning");
  timestamp(p.generated_at, "publication.generated_at"); optionalTimestamp(p.committed_t, "publication.committed_t"); bool(p.connection_active, "publication.connection_active"); bool(p.aggregate_acknowledged, "publication.aggregate_acknowledged");
  fields(p.aggregate_ack_position, ["connection_epoch", "frame_sequence", "array_index"], "publication.aggregate_ack_position");
  decimal(p.aggregate_ack_position.connection_epoch, "aggregate_ack_position.connection_epoch"); decimal(p.aggregate_ack_position.frame_sequence, "aggregate_ack_position.frame_sequence"); uint(p.aggregate_ack_position.array_index, "aggregate_ack_position.array_index");
  const zeroPosition = p.aggregate_ack_position.connection_epoch === "0" && p.aggregate_ack_position.frame_sequence === "0" && p.aggregate_ack_position.array_index === 0;
  if (p.aggregate_acknowledged ? zeroPosition || p.aggregate_ack_position.connection_epoch !== p.connection_epoch : !zeroPosition) fail("aggregate acknowledgement conflict");
  const fence = p.hydration_fence;
  fields(fence, ["reconciled", "connection_epoch", "through_frame_sequence", "marker_ordinal", "supported_through"], "publication.hydration_fence");
  bool(fence.reconciled, "hydration_fence.reconciled");
  for (const name of ["connection_epoch", "through_frame_sequence", "marker_ordinal"]) decimal(fence[name], `hydration_fence.${name}`);
  optionalTimestamp(fence.supported_through, "hydration_fence.supported_through");
  const zeroFence = fence.connection_epoch === "0" && fence.through_frame_sequence === "0" && fence.marker_ordinal === "0" && fence.supported_through === null;
  const positiveFence = fence.connection_epoch !== "0" && fence.through_frame_sequence !== "0" && fence.marker_ordinal !== "0" && fence.supported_through !== null;
  if (fence.reconciled ? !positiveFence || fence.connection_epoch !== p.connection_epoch : !zeroFence) fail("hydration fence conflict");

  const status = snapshot.status;
  fields(status, ["process_live", "backend_ready", "readiness_reason", "ranking_current", "causal_target", "watermark_lag_ms", "accounting_valid", "tq_pressure_mode", "tq_shed"], "status");
  for (const name of ["process_live", "backend_ready", "ranking_current", "accounting_valid", "tq_shed"]) bool(status[name], `status.${name}`);
  string(status.readiness_reason, "status.readiness_reason"); string(status.tq_pressure_mode, "status.tq_pressure_mode"); timestamp(status.causal_target, "status.causal_target");
  if (!READINESS_REASON.has(status.readiness_reason)) fail("unknown readiness reason");
  if (status.watermark_lag_ms !== null) uint(status.watermark_lag_ms, "status.watermark_lag_ms");
  if ((p.committed_t === null) !== (status.watermark_lag_ms === null)) fail("watermark lag conflict");

  const ranking = snapshot.ranking;
  fields(ranking, ["mode", "reason", "total_passers", "known_rankable_count", "day_invalid_rankable", "qualified_day_invalid"], "ranking");
  string(ranking.mode, "ranking.mode"); string(ranking.reason, "ranking.reason");
  if (!RANKING_MODE.has(ranking.mode) || !RANKING_REASON.has(ranking.reason)) fail("unknown ranking meaning");
  for (const name of ["total_passers", "known_rankable_count", "day_invalid_rankable", "qualified_day_invalid"]) uint(ranking[name], `ranking.${name}`);

  if (!Array.isArray(snapshot.rows) || snapshot.rows.length > 20) fail("rows must contain at most 20 values");
  const replay = p.run_mode === "replay";
  const seen = new Set(); snapshot.rows.forEach((row, index) => validateRow(row, index, seen));

  fields(snapshot.accounting, ["population", "qualification", "uncertainty", "population_transition_diagnostic"], "accounting");
  const population = snapshot.accounting.population;
  fields(population, POPULATION_FIELDS, "accounting.population"); POPULATION_FIELDS.forEach(name => uint(population[name], `accounting.population.${name}`));
  if (population.universe_total !== population.valid_prior_close + population.invalid_or_missing_prior_close || population.valid_prior_close !== population.trusted_rankable_mark + population.trusted_below_price_mark + population.no_print_through_t + population.invalid_mark + population.unknown_due_failure_or_fence) fail("population accounting conflict");
  if (population.covered_population !== population.universe_total - population.unknown_due_failure_or_fence || population.unresolved_population !== population.unknown_due_failure_or_fence) fail("population coverage accounting conflict");
  fields(snapshot.accounting.qualification, QUALIFICATION_FIELDS, "accounting.qualification"); QUALIFICATION_FIELDS.forEach(name => uint(snapshot.accounting.qualification[name], `accounting.qualification.${name}`));
  fields(snapshot.accounting.uncertainty, UNCERTAINTY_FIELDS, "accounting.uncertainty"); UNCERTAINTY_FIELDS.forEach(name => uint(snapshot.accounting.uncertainty[name], `accounting.uncertainty.${name}`));
  const transition = snapshot.accounting.population_transition_diagnostic;
  fields(transition, POPULATION_TRANSITION_FIELDS, "accounting.population_transition_diagnostic"); POPULATION_TRANSITION_FIELDS.forEach(name => uint(transition[name], `accounting.population_transition_diagnostic.${name}`));
  if (transition.bootstrap_unknown !== POPULATION_TRANSITION_FIELDS.slice(1).reduce((total, name) => total + transition[name], 0)) fail("population transition accounting conflict");

  const recovery = snapshot.recovery;
  fields(recovery, ["purpose", "generation", "start", "end", "supported_through", "fence_reconciled", "policy_waiting", "work", "rows"], "recovery");
  if (recovery.generation_active !== undefined) bool(recovery.generation_active, "recovery.generation_active");
  string(recovery.purpose, "recovery.purpose"); decimal(recovery.generation, "recovery.generation"); optionalTimestamp(recovery.start, "recovery.start"); optionalTimestamp(recovery.end, "recovery.end"); optionalTimestamp(recovery.supported_through, "recovery.supported_through"); bool(recovery.fence_reconciled, "recovery.fence_reconciled"); bool(recovery.policy_waiting, "recovery.policy_waiting");
  if (!new Set(["", "fresh_bootstrap", "checkpoint_catchup", "gap_recovery"]).has(recovery.purpose)) fail("unknown recovery purpose");
  fields(recovery.work, WORK_FIELDS, "recovery.work"); WORK_FIELDS.forEach(name => decimal(recovery.work[name], `recovery.work.${name}`));
  if (!sumDecimal(recovery.work.planned, recovery.work.open, recovery.work.completed_value, recovery.work.completed_empty, recovery.work.failed, recovery.work.canceled, recovery.work.fenced)) fail("recovery work conflict");
  if (recovery.generation_active && (recovery.generation === "0" || recovery.fence_reconciled)) fail("recovery generation activity conflict");
  fields(recovery.rows, RECOVERY_ROW_FIELDS, "recovery.rows"); RECOVERY_ROW_FIELDS.forEach(name => decimal(recovery.rows[name], `recovery.rows.${name}`));
  if (!sumDecimal(recovery.rows.consumed, recovery.rows.inserted, recovery.rows.duplicate, recovery.rows.conflict_or_withdrawal, recovery.rows.rejected, recovery.rows.fenced, recovery.rows.integrity)) fail("recovery row conflict");

  const tq = snapshot.tq;
  fields(tq, ["desired_symbols", "pressure_mode", "pressure_cause", "aggregate_only", "shed", "retained_bound_hit", "pressure_misses", "pressure_transitions", "pressure_fenced", "pressure_sample", "pressure_recovery", "known_present", "known_absent", "unknown", "retained_trades", "retained_quotes", "retained_fingerprints", "facts", "commands"], "tq");
  if (!Array.isArray(tq.desired_symbols) || tq.desired_symbols.length > 20) fail("invalid desired symbols");
  const desired = new Set(); for (const symbol of tq.desired_symbols) { string(symbol, "tq.desired_symbols[]"); if (symbol === "" || desired.has(symbol)) fail("invalid desired symbol"); desired.add(symbol); }
  string(tq.pressure_mode, "tq.pressure_mode"); string(tq.pressure_cause, "tq.pressure_cause"); for (const name of ["aggregate_only", "shed", "retained_bound_hit"]) bool(tq[name], `tq.${name}`);
  for (const name of ["pressure_misses", "known_present", "known_absent", "unknown", "retained_trades", "retained_quotes", "retained_fingerprints"]) uint(tq[name], `tq.${name}`);
  decimal(tq.pressure_transitions, "tq.pressure_transitions"); decimal(tq.pressure_fenced, "tq.pressure_fenced");
  const pressureSample = tq.pressure_sample, pressureRecovery = tq.pressure_recovery;
  fields(pressureSample, ["observed", "waiting_frames", "frame_capacity", "waiting_bytes", "byte_capacity", "oldest_waiting_frame_age_ms", "aggregate_watermark_lag_ms", "recovery_healthy"], "tq.pressure_sample");
  for (const name of ["observed", "recovery_healthy"]) bool(pressureSample[name], `tq.pressure_sample.${name}`);
  for (const name of ["waiting_frames", "frame_capacity", "waiting_bytes", "byte_capacity", "oldest_waiting_frame_age_ms", "aggregate_watermark_lag_ms"]) uint(pressureSample[name], `tq.pressure_sample.${name}`);
  fields(pressureRecovery, ["healthy_samples", "required_samples"], "tq.pressure_recovery"); uint(pressureRecovery.healthy_samples, "tq.pressure_recovery.healthy_samples"); uint(pressureRecovery.required_samples, "tq.pressure_recovery.required_samples");
  if (pressureRecovery.required_samples !== 5 || pressureRecovery.healthy_samples > pressureRecovery.required_samples || pressureSample.recovery_healthy && !pressureSample.observed ||
      tq.pressure_mode !== "normal" && pressureRecovery.healthy_samples >= pressureRecovery.required_samples ||
      pressureSample.observed && (pressureSample.frame_capacity === 0 || pressureSample.waiting_frames > pressureSample.frame_capacity || pressureSample.byte_capacity === 0 || pressureSample.waiting_bytes > pressureSample.byte_capacity) ||
      !pressureSample.observed && (pressureSample.waiting_frames !== 0 || pressureSample.frame_capacity !== 0 || pressureSample.waiting_bytes !== 0 || pressureSample.byte_capacity !== 0 || pressureSample.oldest_waiting_frame_age_ms !== 0 || pressureSample.aggregate_watermark_lag_ms !== 0 || pressureSample.recovery_healthy) ||
      tq.pressure_mode !== "normal" && pressureRecovery.healthy_samples > 0 && !pressureSample.recovery_healthy || tq.pressure_mode === "normal" && pressureRecovery.healthy_samples !== 0) fail("TQ pressure recovery conflict");
  fields(tq.facts, FACT_FIELDS, "tq.facts"); FACT_FIELDS.forEach(name => decimal(tq.facts[name], `tq.facts.${name}`));
  if (!sumDecimal(tq.facts.consumed, tq.facts.applied, tq.facts.duplicate, tq.facts.rejected, tq.facts.fenced, tq.facts.pressure_shed, tq.facts.integrity)) fail("TQ fact conflict");
  if (BigInt(tq.facts.applied_trades) + BigInt(tq.facts.applied_quotes) > BigInt(tq.facts.applied) || BigInt(tq.facts.pressure_shed_trades) + BigInt(tq.facts.pressure_shed_quotes) !== BigInt(tq.facts.pressure_shed)) fail("TQ family fact conflict");
  fields(tq.commands, COMMAND_FIELDS, "tq.commands"); COMMAND_FIELDS.forEach(name => decimal(tq.commands[name], `tq.commands.${name}`));
  if (!sumDecimal(tq.commands.issued, tq.commands.pending_write, tq.commands.written, tq.commands.failed, tq.commands.fenced)) fail("TQ command conflict");
  if (!new Set(["normal", "taq_degraded", "aggregate_only"]).has(tq.pressure_mode) || !PRESSURE_CAUSE.has(tq.pressure_cause) || (tq.pressure_mode === "normal") !== (tq.pressure_cause === "") || status.tq_pressure_mode !== tq.pressure_mode || status.tq_shed !== tq.shed || tq.aggregate_only !== (tq.pressure_mode === "aggregate_only") || tq.shed !== (tq.pressure_mode !== "normal")) fail("TQ pressure conflict");
  if (ranking.mode === "degraded_bootstrap" || ranking.mode === "degraded_current") {
    if (tq.desired_symbols.length !== 0) fail("partial ranking promoted TQ membership");
    for (const row of snapshot.rows) {
      const membership = row.tq_membership, tape = row.tape_5s, spread = row.spread;
      if (membership.desired || membership.provider_present || membership.provider_membership_unknown ||
          tape.status !== "unselected" || tape.reason !== "" || tape.trade_coverage || tape.trades_per_second !== null || tape.timestamp_basis !== "" || tape.lifecycle_records_observed ||
          spread.status !== "unselected" || spread.reason !== "" || spread.quote_coverage || spread.cents !== null || spread.basis_points !== null || spread.quote_age_ms !== 0 || spread.quality !== "") fail("partial ranking exposed TQ state");
    }
  }

  const checkpointFields = ["installed", ...CHECKPOINT_DECIMAL_FIELDS, ...CHECKPOINT_UINT_FIELDS];
  fields(snapshot.checkpoint, checkpointFields, "checkpoint"); bool(snapshot.checkpoint.installed, "checkpoint.installed"); CHECKPOINT_DECIMAL_FIELDS.forEach(name => decimal(snapshot.checkpoint[name], `checkpoint.${name}`)); CHECKPOINT_UINT_FIELDS.forEach(name => uint(snapshot.checkpoint[name], `checkpoint.${name}`));
  if (!sumDecimal(snapshot.checkpoint.eligible, snapshot.checkpoint.pressure_deferred, snapshot.checkpoint.projection_started) || !sumDecimal(snapshot.checkpoint.projection_started, snapshot.checkpoint.projection_in_progress, snapshot.checkpoint.projected, snapshot.checkpoint.projection_rejected) || !sumDecimal(snapshot.checkpoint.projected, snapshot.checkpoint.submitted, snapshot.checkpoint.submit_rejected) || !sumDecimal(snapshot.checkpoint.submitted, snapshot.checkpoint.outstanding, snapshot.checkpoint.completed, snapshot.checkpoint.failed, snapshot.checkpoint.canceled, snapshot.checkpoint.superseded)) fail("checkpoint conflict");
  const operationFields = ["sample_accounting_valid", "delivery_latency_attribution", ...OPERATION_UINT_FIELDS, ...OPERATION_DECIMAL_FIELDS];
  if (snapshot.operations.integrity_failure !== undefined) operationFields.push("integrity_failure");
  fields(snapshot.operations, operationFields, "operations"); bool(snapshot.operations.sample_accounting_valid, "operations.sample_accounting_valid"); OPERATION_UINT_FIELDS.forEach(name => uint(snapshot.operations[name], `operations.${name}`)); OPERATION_DECIMAL_FIELDS.forEach(name => decimal(snapshot.operations[name], `operations.${name}`));
  const latency = snapshot.operations.delivery_latency_attribution, latencyFields = ["aggregate", "tq", "control", "hydration_fence", "checkpoint", "timer", "unknown"];
  fields(latency, [...latencyFields, "maximum_ms", "maximum_family"], "operations.delivery_latency_attribution"); latencyFields.forEach(name => decimal(latency[name], `operations.delivery_latency_attribution.${name}`)); uint(latency.maximum_ms, "operations.delivery_latency_attribution.maximum_ms"); string(latency.maximum_family, "operations.delivery_latency_attribution.maximum_family");
  if (!sumDecimal(snapshot.operations.deliveries, ...latencyFields.map(name => latency[name])) || !new Set(["aggregate", "tq", "control", "hydration_fence", "checkpoint", "timer", "unknown"]).has(latency.maximum_family)) fail("delivery latency attribution conflict");
  if (snapshot.operations.integrity_failure !== undefined) {
    const failure = snapshot.operations.integrity_failure;
    const diagnosticFields = ["category", "engine_sequence", "candidate_time", "expected_time"];
    for (const name of ["first_symbol", "first_field", "first_reason"]) if (failure[name] !== undefined) diagnosticFields.push(name);
    fields(failure, diagnosticFields, "operations.integrity_failure"); string(failure.category, "operations.integrity_failure.category"); if (!EVALUATOR_INTEGRITY_CATEGORY.has(failure.category)) fail("operations.integrity_failure.category is unknown"); decimal(failure.engine_sequence, "operations.integrity_failure.engine_sequence");
    optionalTimestamp(failure.candidate_time, "operations.integrity_failure.candidate_time"); optionalTimestamp(failure.expected_time, "operations.integrity_failure.expected_time");
    for (const name of ["first_symbol", "first_field", "first_reason"]) if (failure[name] !== undefined) string(failure[name], `operations.integrity_failure.${name}`);
  }

  if (replay) {
    if (snapshot.replay === undefined || snapshot.replay === null) fail("replay context conflict");
    fields(snapshot.replay, ["logical_time"], "replay"); timestamp(snapshot.replay.logical_time, "replay.logical_time");
    if (status.backend_ready || status.readiness_reason !== "not_live_mode") fail("contradictory replay status");
  } else if (snapshot.replay !== undefined && snapshot.replay !== null) fail("live snapshot includes replay context");
  const coherentReady = status.process_live && status.backend_ready && status.ranking_current && status.readiness_reason === "" && status.accounting_valid && BACKEND_READY_RANKING_MODE.has(ranking.mode) && p.run_mode === "live" && p.committed_t !== null && CURRENT_LIFECYCLE.has(p.lifecycle) && p.suppression === "" && p.connection_active && p.aggregate_acknowledged && fence.reconciled;
  if (status.backend_ready && !coherentReady) fail("contradictory current status");
  if (!status.backend_ready && status.readiness_reason === "") fail("missing noncurrent readiness reason");
  return snapshot;
}

function knownCurrentMeasurement(field) { return field.status === "current" && field.reason === ""; }
function knownCurrentTQ(status, reason) { return status === "current" && KNOWN_TQ_REASON.has(reason); }
function tqState(status, reason) { return KNOWN_TQ_STATUS.has(status) && KNOWN_TQ_REASON.has(reason) ? status : "unknown"; }
export function formatPercent(ratio, digits = 2) { return `${(ratio * 100).toFixed(digits)}%`; }
export function formatSignedPercent(ratio, digits = 2) { return `${ratio > 0 ? "+" : ""}${formatPercent(ratio, digits)}`; }
function formatUSD(value) { return value >= 100 ? `$${value.toFixed(2)}` : `$${value.toFixed(4).replace(/0+$/, "").replace(/\.$/, "")}`; }
export function formatShares(value) {
  const units = [[1e12, "T"], [1e9, "B"], [1e6, "M"], [1e3, "K"]];
  for (let index = 0; index < units.length; index++) if (Math.abs(value) >= units[index][0]) {
    let [scale, suffix] = units[index];
    if (index > 0 && Math.abs(Number((value / scale).toFixed(1))) >= 1000) [scale, suffix] = units[index - 1];
    return `${(value / scale).toFixed(1).replace(/\.0$/, "")}${suffix}`;
  }
  return Number.isInteger(value) ? String(value) : value.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
}
export function volumeTurnoverIntensity(sessionShareVolume, floatShares) {
  if (typeof sessionShareVolume !== "number" || !Number.isFinite(sessionShareVolume) || sessionShareVolume < 0) return null;
  if (typeof floatShares !== "number" || !Number.isFinite(floatShares) || floatShares <= 0) return null;
  const turnover = sessionShareVolume / floatShares;
  if (!Number.isFinite(turnover) || turnover < 0) return null;
  if (turnover >= 5) return 1;
  if (turnover >= 2) return .8;
  if (turnover >= 1) return .6;
  if (turnover >= .5) return .4;
  if (turnover >= .25) return .2;
  return 0;
}
export function floatCyanIntensity(field) {
  if (!field || field.status !== "current" || field.reason !== "" || typeof field.value_shares !== "number" || !Number.isFinite(field.value_shares) || field.value_shares <= 0) return null;
  if (field.value_shares < 2_000_000) return 1;
  if (field.value_shares < 5_000_000) return .85;
  if (field.value_shares < 10_000_000) return .65;
  if (field.value_shares < 20_000_000) return .45;
  if (field.value_shares < 50_000_000) return .2;
  return 0;
}
const DAY_RANGE_COLOR_STOPS = [
  [0, "#FC0000"], [.25, "#FF5A5A"], [.5, "#8F9AA3"], [.75, "#73FF63"], [1, "#2CFF05"],
];
const ACTIVITY_COLOR_STOPS = [
  [0, "#8F9AA3"], [.5, "#8F9AA3"], [.75, "#C87932"], [.9, "#E98212"], [.97, "#F98A05"], [1, "#FF8A00"],
];
export const NEUTRAL_TEXT_COLOR = "#8F9AA3";
export const MUTED_NEGATIVE_TEXT_COLOR = "#C46B6B";
const MOVE_COLOR_STOPS = [
  [-.07, "#FC0000"], [-.05, "#EF3030"], [-.02, MUTED_NEGATIVE_TEXT_COLOR], [0, NEUTRAL_TEXT_COLOR], [.02, "#70B873"], [.05, "#45E532"], [.07, "#2CFF05"],
];
const TAPE_COLOR_STOPS = [
  [0, "#8F9AA3"], [50, "#8F9AA3"], [100, "#B8793E"], [250, "#E98212"], [500, "#FF8A00"],
];
const SPREAD_COLOR_STOPS = [
  [0, "#8F9AA3"], [20, "#8F9AA3"], [35, "#B08A5A"], [60, "#D1843E"], [100, "#E05A32"], [200, "#E9342B"], [400, "#FC0000"],
];
function rgbFromHex(hex) { return [1, 3, 5].map(index => Number.parseInt(hex.slice(index, index + 2), 16)); }
export function interpolateRGBGradient(value, stops) {
  if (typeof value !== "number" || !Number.isFinite(value) || !Array.isArray(stops) || stops.length === 0) return null;
  const clamped = Math.max(stops[0][0], Math.min(stops.at(-1)[0], value));
  let upperIndex = 1;
  while (upperIndex < stops.length && clamped > stops[upperIndex][0]) upperIndex++;
  if (upperIndex === stops.length) return stops.at(-1)[1].toUpperCase();
  const [lowerPosition, lowerColor] = stops[upperIndex - 1], [upperPosition, upperColor] = stops[upperIndex], span = upperPosition - lowerPosition;
  const weight = span === 0 ? 0 : (clamped - lowerPosition) / span;
  const lowerRGB = rgbFromHex(lowerColor), upperRGB = rgbFromHex(upperColor);
  return `#${lowerRGB.map((channel, index) => Math.round(channel + (upperRGB[index] - channel) * weight).toString(16).padStart(2, "0")).join("")}`.toUpperCase();
}
export function signedValueTextColor(value) {
  if (typeof value !== "number" || !Number.isFinite(value)) return null;
  if (value < 0) return MUTED_NEGATIVE_TEXT_COLOR;
  if (value === 0) return NEUTRAL_TEXT_COLOR;
  return null;
}
export function dayRangeColor(value) { return interpolateRGBGradient(value, DAY_RANGE_COLOR_STOPS); }
export function activityColor(value) { return interpolateRGBGradient(value, ACTIVITY_COLOR_STOPS); }
export function moveColor(value) { return interpolateRGBGradient(value, MOVE_COLOR_STOPS); }
export function tapeColor(value) { return interpolateRGBGradient(value, TAPE_COLOR_STOPS); }
export function spreadColor(value) { return interpolateRGBGradient(value, SPREAD_COLOR_STOPS); }
function band(value, stops) { const magnitude = Math.abs(value); let result = 0; for (let index = 1; index < stops.length; index++) if (magnitude >= stops[index]) result = index; return result; }
function fieldView(field, label, stops = null, digits = 2, signed = false) {
  const state = KNOWN_FIELD_STATUS.has(field.status) && KNOWN_FIELD_REASON.has(field.reason) ? field.status : "unknown";
  return knownCurrentMeasurement(field)
    ? { state: "current", text: signed ? formatSignedPercent(field.value_ratio, digits) : formatPercent(field.value_ratio, digits), detail: `${label}: status current; reason none`, band: stops ? band(field.value_ratio * 100, stops) : 0 }
    : { state, text: "—", detail: `${label}: status ${state}; reason ${field.reason || field.status || "unknown"}`, band: 0 };
}
function rangeFieldView(field) { const view = fieldView(field, "Day Range", null, 0); view.position = view.state === "current" ? field.value_ratio * 100 : null; view.textColor = view.state === "current" ? dayRangeColor(field.value_ratio) : null; return view; }
function activityFieldView(field) { const view = fieldView(field, "Activity 30s", null, 0); view.position = view.state === "current" ? field.value_ratio * 100 : null; view.textColor = view.state === "current" ? activityColor(field.value_ratio) : null; return view; }
function moveFieldView(field) { const view = fieldView(field, "Move 30s", null, 2, true); view.textColor = view.state === "current" ? moveColor(field.value_ratio) : null; return view; }
function groupedDecimal(value) { return value.replace(/\B(?=(\d{3})+(?!\d))/g, ","); }
function hydrationView(recovery) {
  const work = recovery.work;
  const planned = BigInt(work.planned), open = BigInt(work.open);
  const failed = BigInt(work.failed), canceled = BigInt(work.canceled), fenced = BigInt(work.fenced);
  const terminal = BigInt(work.completed_value) + BigInt(work.completed_empty) + failed + canceled + fenced;
  const issues = failed + canceled + fenced;
  const percent = planned === 0n ? null : terminal * 1000n / planned;
  const progress = planned === 0n ? "planning" : `${groupedDecimal(String(terminal))} / ${groupedDecimal(work.planned)} · ${percent / 10n}.${percent % 10n}%`;
  const issueText = issues === 0n ? "" : ` · failed ${groupedDecimal(work.failed)} · canceled ${groupedDecimal(work.canceled)} · fenced ${groupedDecimal(work.fenced)}`;
  return { planned, open, issues, progress, issueText };
}

function operationalPhase(snapshot, hydration) {
  const lifecycle = snapshot.publication.lifecycle;
  if (lifecycle !== "hydrating" && lifecycle !== "recovering") return "";
  if (!snapshot.publication.connection_active) return lifecycle === "recovering" ? "reconnecting" : "connecting";
  if (!snapshot.publication.aggregate_acknowledged) return lifecycle === "recovering" ? "resubscribing" : "subscribing";
  if (snapshot.recovery.policy_waiting) return "retrying";
  if (!snapshot.recovery.generation_active) return lifecycle === "recovering" ? "preparing_recovery" : "preparing_hydration";
  if (hydration.planned > 0n && hydration.open === 0n) return lifecycle === "recovering" ? "finalizing_recovery" : "finalizing_hydration";
  return lifecycle === "recovering" ? "recovering" : "hydrating";
}

export function relativeColorPositions(rows, valueForRow, { positiveOnly = false } = {}) {
  const values = rows.map(row => {
    const value = valueForRow(row);
    return typeof value === "number" && Number.isFinite(value) ? value : null;
  });
  const eligible = values.filter(value => value !== null && (!positiveOnly || value > 0));
  if (eligible.length === 0) return values.map(value => positiveOnly ? null : value);
  let minimum = eligible[0], maximum = minimum;
  for (let index = 1; index < eligible.length; index++) {
    minimum = Math.min(minimum, eligible[index]);
    maximum = Math.max(maximum, eligible[index]);
  }
  const spread = maximum - minimum;
  if (spread === 0) return values.map(value => value === null || positiveOnly && value <= 0 ? null : positiveOnly ? 1 : .5);
  return values.map(value => value === null || positiveOnly && value <= 0 ? null : Math.max(0, Math.min(1, (value - minimum) / spread)));
}

const RANK_LOOKBACK_MILLISECONDS = 60_000;
const RANK_HISTORY_LIMIT = 70;
const RANK_COMPARISON_TOLERANCE_MILLISECONDS = 5_000;

function rankMovement(previousRank, currentRank) {
  if (previousRank === undefined) return { direction: "up", text: "↑ NEW", label: "new to the top 20 compared with 60 seconds ago" };
  const delta = previousRank - currentRank;
  if (delta === 0) return { direction: "none", text: "", label: "unchanged from 60 seconds ago" };
  const magnitude = Math.abs(delta);
  return {
    direction: delta > 0 ? "up" : "down",
    text: `${delta > 0 ? "↑" : "↓"}${Math.min(magnitude, 5)}${magnitude > 5 ? "+" : ""}`,
    label: `${delta > 0 ? "up" : "down"} ${magnitude} rank${magnitude === 1 ? "" : "s"} from 60 seconds ago`,
  };
}

const NO_RANK_MOVEMENT = Object.freeze({ direction: "none", text: "", label: "60-second rank movement unavailable" });

export class RankMovementHistory {
  constructor({ lookbackMilliseconds = RANK_LOOKBACK_MILLISECONDS, comparisonToleranceMilliseconds = RANK_COMPARISON_TOLERANCE_MILLISECONDS, maximumSnapshots = RANK_HISTORY_LIMIT } = {}) {
    this.lookbackMilliseconds = lookbackMilliseconds;
    this.comparisonToleranceMilliseconds = comparisonToleranceMilliseconds;
    this.maximumSnapshots = maximumSnapshots;
    this.snapshots = [];
    this.bindingKey = "";
  }
  get size() { return this.snapshots.length; }
  apply(model) {
    const currentRows = model.rows.map((row, index) => ({ ...row, rank: index + 1, rankMovement: NO_RANK_MOVEMENT }));
    if (!model.rowsCurrent) return { ...model, rows: currentRows };
    const timestamp = Date.parse(model.sampledAt);
    const bindingKey = `${model.bindingIdentity}\u0000${model.tradingDate}`;
    if (!Number.isFinite(timestamp)) return { ...model, rows: currentRows };
    if (this.bindingKey !== bindingKey || this.snapshots.length > 0 && timestamp < this.snapshots.at(-1).timestamp) {
      this.snapshots = [];
      this.bindingKey = bindingKey;
    }
    const target = timestamp - this.lookbackMilliseconds;
    let comparison = null;
    if (this.snapshots.length > 0 && this.snapshots[0].timestamp <= target) {
      for (const snapshot of this.snapshots) {
        if (comparison === null || Math.abs(snapshot.timestamp - target) < Math.abs(comparison.timestamp - target)) comparison = snapshot;
      }
      if (comparison !== null && Math.abs(comparison.timestamp - target) > this.comparisonToleranceMilliseconds) comparison = null;
    }
    const rows = comparison === null ? currentRows : currentRows.map(row => ({ ...row, rankMovement: rankMovement(comparison.ranks.get(row.symbol), row.rank) }));
    const ranks = new Map(currentRows.map(row => [row.symbol, row.rank]));
    const retained = { timestamp, ranks };
    if (this.snapshots.at(-1)?.timestamp === timestamp) this.snapshots[this.snapshots.length - 1] = retained;
    else this.snapshots.push(retained);
    while (this.snapshots.length > this.maximumSnapshots) this.snapshots.shift();
    return { ...model, rows };
  }
}

export function buildViewModel(input, transport = "connected") {
  const snapshot = validateSnapshot(input);
  const current = snapshot.status.backend_ready && snapshot.ranking.mode === "qualified_current" && transport === "connected";
  const partial = snapshot.status.backend_ready && snapshot.ranking.mode === "degraded_current" && transport === "connected";
  const replay = snapshot.publication.run_mode === "replay";
  const rowsCurrent = current || replay && snapshot.status.ranking_current && snapshot.ranking.mode === "qualified_current" && transport === "connected";
  const dayColors = relativeColorPositions(snapshot.rows, row => row.day_change_ratio, { positiveOnly: true });
  const fromOpenColors = relativeColorPositions(snapshot.rows, row => row.from_open_change.status === "current" ? row.from_open_change.value_ratio : null, { positiveOnly: true });
  const rows = snapshot.rows.map((row, index) => {
    const floatAvailable = row.float.status === "current" || row.float.status === "stale";
    const tapeCurrent = knownCurrentTQ(row.tape_5s.status, row.tape_5s.reason);
    const spreadCurrent = knownCurrentTQ(row.spread.status, row.spread.reason) || row.spread.status === "stale" && row.spread.reason === "stale_quote";
    const floatState = KNOWN_FLOAT_STATUS.has(row.float.status) ? row.float.status : "unknown";
    const floatText = floatAvailable ? `${formatShares(row.float.value_shares)}${row.float.status === "stale" ? " · stale" : ""}` : "—";
    const floatIntensity = floatCyanIntensity(row.float);
    const floatDetail = `Float: status ${floatState}; reason ${row.float.reason || "none"}; provider ${row.float.provider || "none"}; effective date ${row.float.effective_date || "unknown"}; retrieved ${row.float.retrieved_at || "unknown"}; provenance ${row.float.provenance || "none"}${row.float.percent_ratio === null ? "" : `; percent ${formatPercent(row.float.percent_ratio)}`}`;
    const volumeCurrent = row.volume.status === "current";
    const volumeState = KNOWN_FIELD_STATUS.has(row.volume.status) ? row.volume.status : "unknown";
    const volumeColorIntensity = volumeCurrent && row.float.status === "current"
      ? volumeTurnoverIntensity(row.volume.value_shares, row.float.value_shares)
      : null;
    const tapeText = tapeCurrent ? `${row.tape_5s.trades_per_second.toFixed(1)}/s` : "—";
    const tapeTextColor = tapeCurrent ? tapeColor(row.tape_5s.trades_per_second) : null;
    const spreadTextColor = row.spread.status === "current" ? spreadColor(row.spread.basis_points) : null;
    const dayTextColor = signedValueTextColor(row.day_change_ratio);
    const fromOpenTextColor = knownCurrentMeasurement(row.from_open_change) ? signedValueTextColor(row.from_open_change.value_ratio) : null;
    return {
    rank: index + 1, rankMovement: NO_RANK_MOVEMENT, symbol: row.symbol, last: formatUSD(row.last_usd), day: formatSignedPercent(row.day_change_ratio), dayColor: dayColors[index], dayTextColor, markAgeMS: row.mark_age_ms,
    float: { state: floatState, text: floatText, detail: floatDetail, cyanIntensity: floatIntensity },
    volume: { state: volumeState, text: volumeCurrent ? formatShares(row.volume.value_shares) : "—", colorIntensity: volumeColorIntensity, detail: `Volume: status ${volumeState}; reason ${row.volume.reason || "none"}` },
    fromOpen: { ...fieldView(row.from_open_change, "From Open", null, 2, true), colorPosition: fromOpenColors[index], textColor: fromOpenTextColor }, dayRange: rangeFieldView(row.day_range_position), activity: activityFieldView(row.activity_30s), move: moveFieldView(row.move_30s),
    tape: { state: tqState(row.tape_5s.status, row.tape_5s.reason), primary: tapeText, textColor: tapeTextColor, detail: `Tape speed: status ${row.tape_5s.status}; reason ${row.tape_5s.reason || "none"}; coverage ${row.tape_5s.trade_coverage ? "yes" : "no"}; timestamp ${row.tape_5s.timestamp_basis || "none"}; lifecycle records ${row.tape_5s.lifecycle_records_observed ? "observed" : "not observed"}; membership desired ${row.tq_membership.desired ? "yes" : "no"}, provider ${row.tq_membership.provider_present ? "present" : "absent"}, unknown ${row.tq_membership.provider_membership_unknown ? "yes" : "no"}` },
    spread: { state: tqState(row.spread.status, row.spread.reason), primary: spreadCurrent ? `${row.spread.basis_points.toFixed(1)} bps / ${row.spread.cents.toFixed(2)}¢${row.spread.status === "stale" ? " · stale" : ""}` : "—", textColor: spreadTextColor, detail: `Spread: status ${row.spread.status}; reason ${row.spread.reason || "none"}; coverage ${row.spread.quote_coverage ? "yes" : "no"}; quote age ${row.spread.quote_age_ms} ms; quality ${row.spread.quality || "none"}; membership desired ${row.tq_membership.desired ? "yes" : "no"}, provider ${row.tq_membership.provider_present ? "present" : "absent"}, unknown ${row.tq_membership.provider_membership_unknown ? "yes" : "no"}` },
  }; });
  const hydration = hydrationView(snapshot.recovery);
  const phase = replay || !snapshot.status.process_live || snapshot.status.backend_ready ? "" : operationalPhase(snapshot, hydration);
  const warming = phase === "hydrating" || phase === "preparing_hydration";
  const recovering = ["reconnecting", "resubscribing", "preparing_recovery", "recovering", "retrying", "finalizing_recovery"].includes(phase);
  const finalizing = phase === "finalizing_hydration" || phase === "finalizing_recovery";
  return {
    schemaVersion: snapshot.schema_version, sampleID: snapshot.sample.id, sampledAt: snapshot.sample.sampled_at, publicationID: snapshot.publication.id,
    bindingIdentity: snapshot.publication.binding_identity, tradingDate: snapshot.publication.trading_date,
    transport, current, partial, rowsCurrent, replay, replayLogicalTime: replay ? snapshot.replay.logical_time : null,
    processLive: snapshot.status.process_live, backendReady: snapshot.status.backend_ready, readinessReason: snapshot.status.readiness_reason,
    lifecycle: snapshot.publication.lifecycle, rankingMode: snapshot.ranking.mode, rankingReason: snapshot.ranking.reason, committedT: snapshot.publication.committed_t,
    watermarkLagMS: snapshot.status.watermark_lag_ms, accountingValid: snapshot.status.accounting_valid, sampleAccountingValid: snapshot.operations.sample_accounting_valid, tqPressure: snapshot.tq.pressure_mode, tqPressureCause: snapshot.tq.pressure_cause, tqAggregateOnly: snapshot.tq.aggregate_only,
    tqPressureSampleObserved: snapshot.tq.pressure_sample.observed, tqOldestWaitingFrameAgeMS: snapshot.tq.pressure_sample.oldest_waiting_frame_age_ms, tqRecoveryHealthySamples: snapshot.tq.pressure_recovery.healthy_samples, tqRecoveryRequiredSamples: snapshot.tq.pressure_recovery.required_samples,
    tqUnknown: snapshot.tq.unknown, tqRetainedBoundHit: snapshot.tq.retained_bound_hit, tqKnownPresent: snapshot.tq.known_present, tqKnownAbsent: snapshot.tq.known_absent,
    connectionActive: snapshot.publication.connection_active, aggregateAcknowledged: snapshot.publication.aggregate_acknowledged,
    recoveryPurpose: snapshot.recovery.purpose, recoveryGeneration: snapshot.recovery.generation, recoveryGenerationActive: snapshot.recovery.generation_active === true, recoveryPolicyWaiting: snapshot.recovery.policy_waiting, rows,
    suppression: snapshot.publication.suppression, lifecycleReason: snapshot.publication.lifecycle_reason, integrityFailure: snapshot.operations.integrity_failure || null,
    hydrationProgress: hydration.progress, hydrationIssueText: hydration.issueText, hydrationIssues: hydration.issues !== 0n,
    phase, warming, recovering, finalizing,
  };
}

export async function readBoundedJSON(response) {
  const length = response.headers?.get?.("content-length");
  if (length !== null && length !== undefined && (!DECIMAL.test(length) || BigInt(length) > BigInt(MAX_RESPONSE_BYTES))) fail("response exceeds 1 MiB");
  if (!response.body?.getReader) fail("response body unavailable");
  const reader = response.body.getReader();
  const chunks = []; let size = 0;
  try {
    for (;;) {
      const { done, value } = await reader.read();
      if (done) break;
      size += value.byteLength;
      if (size > MAX_RESPONSE_BYTES) { await reader.cancel(); fail("response exceeds 1 MiB"); }
      chunks.push(value);
    }
  } finally { reader.releaseLock(); }
  const bytes = new Uint8Array(size); let offset = 0;
  for (const chunk of chunks) { bytes.set(chunk, offset); offset += chunk.byteLength; }
  let text;
  try { text = new TextDecoder("utf-8", { fatal: true }).decode(bytes); } catch { fail("response is not UTF-8"); }
  try { return JSON.parse(text); } catch { fail("response is not JSON"); }
}

export class PollController {
  constructor({ url, pollMilliseconds = 1000, requestTimeoutMilliseconds = 3000, fetchImpl = null, onUpdate = () => {} }) {
    this.url = url; this.pollMilliseconds = pollMilliseconds; this.requestTimeoutMilliseconds = requestTimeoutMilliseconds;
    this.fetchImpl = fetchImpl ?? globalThis.fetch.bind(globalThis); this.onUpdate = onUpdate;
    this.active = null; this.timer = null; this.lastModel = null; this.rankHistory = new RankMovementHistory(); this.stopped = false;
  }
  start() { if (this.timer !== null) return; this.stopped = false; void this.tick(); this.timer = setInterval(() => void this.tick(), this.pollMilliseconds); }
  stop() { this.stopped = true; if (this.timer !== null) clearInterval(this.timer); this.timer = null; this.active?.controller.abort(); }
  async tick() {
    if (this.active) { this.onUpdate({ kind: "transport", transport: "refresh_delayed", model: this.lastModel }); return; }
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), this.requestTimeoutMilliseconds);
    const operation = { controller }; this.active = operation;
    try {
      const response = await this.fetchImpl(this.url, { method: "GET", mode: "cors", cache: "no-store", credentials: "omit", signal: controller.signal, headers: { Accept: "application/json" } });
      if (!response.ok) fail(`snapshot HTTP ${response.status}`);
      const snapshot = await readBoundedJSON(response);
      const model = this.rankHistory.apply(buildViewModel(snapshot, "connected"));
      if (!this.stopped) { this.onUpdate({ kind: "snapshot", transport: "connected", model }); this.lastModel = model; }
    } catch (error) {
      if (!this.stopped) {
        try { this.onUpdate({ kind: "error", transport: "disconnected", error: error instanceof Error ? error.message : "request failed", model: this.lastModel }); } catch { /* detached renderer preserves the prior committed DOM */ }
      }
    } finally {
      clearTimeout(timeout); if (this.active === operation) this.active = null;
    }
  }
}
