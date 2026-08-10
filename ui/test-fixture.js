export function snapshotFixture(rowCount = 1) {
  const populationSize = Math.max(rowCount, 2);
  const measurement = value => ({ status: "current", reason: "", value_ratio: value });
  const rows = Array.from({ length: rowCount }, (_, index) => ({
    rank: index + 1, symbol: `S${String(index + 1).padStart(2, "0")}`, last_usd: 10 + index, day_change_ratio: .0025 + (rowCount - index - 1) / 1000, mark_age_ms: 250,
    from_4am_change: measurement(0), hod_drawdown: measurement(-.01), day_range_position: measurement(.5), range_30m_position: measurement(.25), range_60m_position: measurement(.75), activity: measurement(.8),
    tape_rate: { status: "current", reason: "qualifying_original_prints", trade_coverage: true, one_second: { status: "current", reason: "qualifying_original_prints", trades_per_second: 2 }, five_second: { status: "current", reason: "qualifying_original_prints", trades_per_second: 1.4 }, timestamp_basis: "mixed", lifecycle_records_observed: true },
    spread: { status: "current", reason: "", quote_coverage: true, cents: 1.5, basis_points: 15, valid_duration_ms: 5000, quality: "reviewed_ordinary" },
    tq_membership: { desired: true, provider_present: true, provider_membership_unknown: false },
  }));
  const desired = rows.map(row => row.symbol);
  return {
    schema_version: "scanner.snapshot.v1",
    sample: { id: "10", sampled_at: "2026-08-08T16:00:00Z" },
    publication: { id: "20", binding_identity: "binding", trading_date: "2026-08-08", run_mode: "live", lifecycle: "live", lifecycle_reason: "hydration_complete", suppression: "", generated_at: "2026-08-08T16:00:00Z", committed_t: "2026-08-08T15:59:58Z", last_engine_sequence: "30", connection_epoch: "4", connection_active: true, aggregate_acknowledged: true, aggregate_ack_position: { connection_epoch: "4", frame_sequence: "8", array_index: 1 }, hydration_fence: { reconciled: true, connection_epoch: "4", through_frame_sequence: "8", marker_ordinal: "2", supported_through: "2026-08-08T15:59:58Z" } },
    status: { process_live: true, backend_ready: true, readiness_reason: "", ranking_current: true, causal_target: "2026-08-08T15:59:58Z", watermark_lag_ms: 0, accounting_valid: true, tq_pressure_mode: "normal", tq_shed: false },
    ranking: { mode: "qualified_current", reason: "", total_passers: rowCount, known_rankable_count: rowCount, day_invalid_rankable: 0, qualified_day_invalid: 0 },
    rows,
    accounting: { population: { universe_total: populationSize, valid_prior_close: populationSize, invalid_or_missing_prior_close: 0, trusted_rankable_mark: rowCount, trusted_below_price_mark: 0, no_print_through_t: populationSize - rowCount, invalid_mark: 0, unknown_due_failure_or_fence: 0, covered_population: populationSize, unresolved_population: 0 }, qualification: { not_yet_passed: populationSize - rowCount, provisional: rowCount, finalized: 0, unresolved: 0 }, uncertainty: { bootstrap_origin: 0, post_bootstrap_gap: 0, local_invalid: 0 } },
    recovery: { purpose: "fresh_bootstrap", generation: "1", start: "2026-08-08T15:58:00Z", end: "2026-08-08T15:59:00Z", supported_through: "2026-08-08T15:59:58Z", fence_reconciled: true, policy_waiting: false, work: { planned: String(populationSize), open: "0", completed_value: String(rowCount), completed_empty: String(populationSize - rowCount), failed: "0", canceled: "0", fenced: "0" }, rows: { consumed: String(rowCount), inserted: String(rowCount), duplicate: "0", conflict_or_withdrawal: "0", rejected: "0", fenced: "0", integrity: "0" } },
    tq: { desired_symbols: desired, pressure_mode: "normal", aggregate_only: false, shed: false, retained_bound_hit: false, pressure_misses: 0, pressure_transitions: "0", pressure_fenced: "0", known_present: rowCount, known_absent: 0, unknown: 0, retained_trades: rowCount, retained_quotes: rowCount, retained_fingerprints: rowCount, facts: { consumed: String(rowCount * 2), applied: String(rowCount * 2), duplicate: "0", rejected: "0", fenced: "0", pressure_shed: "0", integrity: "0" }, commands: { issued: String(rowCount), pending: "0", acknowledged: String(rowCount), failed: "0", fenced: "0", result_fenced: "0" } },
    checkpoint: { installed: false, submitted: "0", in_progress: "0", pending: "0", completed: "0", failed: "0", canceled: "0", superseded: "0" },
    operations: { sample_accounting_valid: true, queue_capacity_frames: 64, queue_current_frames: 0, queue_high_frames: 2, queue_current_bytes: 0, queue_high_bytes: 2048, deliveries: "10", consumer_deferred: "0", mean_processing_delay_ms: 1, max_processing_delay_ms: 2, max_processing_delay_one_second_ms: 2, heap_alloc_bytes: "1024", heap_in_use_bytes: "2048", goroutines: 8, connection_recovery_attempts: "0" },
  };
}

export function replaySnapshotFixture(sequence = 0) {
  const snapshot = snapshotFixture(2);
  const logicalSecond = String(sequence).padStart(2, "0");
  const logicalTime = `2026-08-08T16:00:${logicalSecond}Z`;
  snapshot.sample = { id: String(100 + sequence), sampled_at: logicalTime };
  snapshot.publication = {
    ...snapshot.publication,
    id: String(200 + sequence), run_mode: "replay", lifecycle: "replaying", lifecycle_reason: "replay_start",
    generated_at: logicalTime, committed_t: logicalTime, last_engine_sequence: String(300 + sequence), connection_epoch: "0",
    connection_active: false, aggregate_acknowledged: false,
    aggregate_ack_position: { connection_epoch: "0", frame_sequence: "0", array_index: 0 },
    hydration_fence: { reconciled: false, connection_epoch: "0", through_frame_sequence: "0", marker_ordinal: "0", supported_through: null },
  };
  snapshot.status = { ...snapshot.status, backend_ready: false, readiness_reason: "not_live_mode", ranking_current: true, causal_target: logicalTime };
  snapshot.recovery = {
    purpose: "", generation: "0", start: null, end: null, supported_through: null, fence_reconciled: false, policy_waiting: false,
    work: { planned: "0", open: "0", completed_value: "0", completed_empty: "0", failed: "0", canceled: "0", fenced: "0" },
    rows: { consumed: "0", inserted: "0", duplicate: "0", conflict_or_withdrawal: "0", rejected: "0", fenced: "0", integrity: "0" },
  };
  snapshot.tq = {
    ...snapshot.tq, desired_symbols: [], known_present: 0, known_absent: 0, unknown: 0, retained_trades: 0, retained_quotes: 0, retained_fingerprints: 0,
    facts: { consumed: "0", applied: "0", duplicate: "0", rejected: "0", fenced: "0", pressure_shed: "0", integrity: "0" },
    commands: { issued: "0", pending: "0", acknowledged: "0", failed: "0", fenced: "0", result_fenced: "0" },
  };
  for (const row of snapshot.rows) {
    row.tape_rate = { status: "unavailable", reason: "replay_unavailable", trade_coverage: false, one_second: { status: "unavailable", reason: "replay_unavailable", trades_per_second: null }, five_second: { status: "unavailable", reason: "replay_unavailable", trades_per_second: null }, timestamp_basis: "", lifecycle_records_observed: false };
    row.spread = { status: "unavailable", reason: "replay_unavailable", quote_coverage: false, cents: null, basis_points: null, valid_duration_ms: 0, quality: "" };
    row.tq_membership = { desired: false, provider_present: false, provider_membership_unknown: false };
  }
  snapshot.replay = {
    phase: "observing",
    artifact_id: "sha256:0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
    artifact_end: "2026-08-08T20:00:00Z",
    observation_start: "2026-08-08T16:00:00Z",
    observation_end: "2026-08-08T16:00:03Z",
    logical_time: logicalTime,
    completion: "",
    schedule_lag_ms: 0,
    source: {
      artifact_records: "3", completed_record_dispositions: String(sequence + 1), intentionally_unapplied_suffix_records: "0", unread_records: String(2 - sequence),
      planned_groups: "4", completed_groups: String(sequence + 1), active_group: "0", remaining_groups: String(3 - sequence), completed_runs: "0", failed_runs: "0", canceled_runs: "0",
    },
    window: {
      warmup_groups_planned: "1", warmup_groups_completed: "1", warmup_group_active: "0", warmup_groups_remaining: "0",
      observation_seconds_planned: "3", observation_seconds_completed: String(sequence), observation_second_active: "0", observation_seconds_remaining: String(3 - sequence), observation_boundaries_published: String(sequence + 1),
    },
  };
  return snapshot;
}
