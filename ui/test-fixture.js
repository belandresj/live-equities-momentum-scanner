export function snapshotFixture(rowCount = 1) {
  const populationSize = Math.max(rowCount, 2);
  const measurement = value => ({ status: "current", reason: "", value_ratio: value });
  const rows = Array.from({ length: rowCount }, (_, index) => ({
    rank: index + 1, symbol: `S${String(index + 1).padStart(2, "0")}`, last_usd: 10 + index, day_change_ratio: .0025 + index / 1000, mark_age_ms: 250,
    from_4am_change: measurement(0), hod_drawdown: measurement(-.01), day_range_position: measurement(.5), range_30m_position: measurement(.25), range_60m_position: measurement(.75), activity: measurement(.8),
    tape_rate: { status: "current", reason: "qualifying_original_prints", trade_coverage: true, one_second: { status: "current", reason: "qualifying_original_prints", trades_per_second: 2 }, five_second: { status: "current", reason: "qualifying_original_prints", trades_per_second: 1.4 }, timestamp_basis: "mixed", lifecycle_records_observed: true },
    spread: { status: "current", reason: "", quote_coverage: true, cents: 1.5, basis_points: 15, valid_duration_ms: 10000, quality: "reviewed_ordinary" },
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
