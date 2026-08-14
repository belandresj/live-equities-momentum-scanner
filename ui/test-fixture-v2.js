export function snapshotFixtureV2(rowCount = 1) {
  const populationSize = Math.max(rowCount, 2);
  const ratio = value => ({ status: "current", reason: "", value_ratio: value });
  const rows = Array.from({ length: rowCount }, (_, index) => ({
    rank: index + 1, symbol: `S${String(index + 1).padStart(2, "0")}`,
    float: { status: index === 1 ? "stale" : "current", reason: index === 1 ? "cached_fallback" : "", value_shares: 12_000_000 + index * 1_000_000, percent_ratio: null, provider: "massive-stocks-float-experimental", effective_date: "2026-08-07", retrieved_at: "2026-08-08T15:55:00Z", provenance: index === 1 ? "cache" : "fresh" },
    volume: { status: "current", reason: "", value_shares: index === 0 ? 0 : 1_250_000 + index * 50_000 },
    last_usd: 10 + index, day_change_ratio: .0025 + (rowCount - index - 1) / 1000, mark_age_ms: 250,
    from_open_change: ratio(0), day_range_position: ratio(.5), activity_30s: ratio(index % 2 === 0 ? .8 : .15), move_30s: ratio(index % 2 === 0 ? .0125 : -.0075),
    tape_5s: { status: "current", reason: "qualifying_original_prints", trade_coverage: true, trades_per_second: index % 2 === 0 ? 1.4 : 18, timestamp_basis: "mixed", lifecycle_records_observed: true },
    spread: { status: "current", reason: "", quote_coverage: true, cents: 1.5, basis_points: 15, quote_age_ms: 500, quality: "reviewed_ordinary" },
    tq_membership: { desired: true, provider_present: true, provider_membership_unknown: false },
  }));
  return {
    schema_version: "scanner.snapshot.v2",
    sample: { id: "10", sampled_at: "2026-08-08T16:00:00Z" },
    publication: { id: "20", binding_identity: "binding", trading_date: "2026-08-08", run_mode: "live", lifecycle: "live", lifecycle_reason: "hydration_complete", suppression: "", generated_at: "2026-08-08T16:00:00Z", committed_t: "2026-08-08T15:59:58Z", last_engine_sequence: "30", connection_epoch: "4", connection_active: true, aggregate_acknowledged: true, aggregate_ack_position: { connection_epoch: "4", frame_sequence: "8", array_index: 1 }, hydration_fence: { reconciled: true, connection_epoch: "4", through_frame_sequence: "8", marker_ordinal: "2", supported_through: "2026-08-08T15:59:58Z" } },
    status: { process_live: true, backend_ready: true, readiness_reason: "", ranking_current: true, causal_target: "2026-08-08T15:59:58Z", watermark_lag_ms: 0, accounting_valid: true, tq_pressure_mode: "normal", tq_shed: false },
    ranking: { mode: "qualified_current", reason: "", total_passers: rowCount, known_rankable_count: rowCount, day_invalid_rankable: 0, qualified_day_invalid: 0 }, rows,
    accounting: {
      population: { universe_total: populationSize, valid_prior_close: populationSize, invalid_or_missing_prior_close: 0, trusted_rankable_mark: rowCount, trusted_below_price_mark: 0, no_print_through_t: populationSize - rowCount, invalid_mark: 0, unknown_due_failure_or_fence: 0, covered_population: populationSize, unresolved_population: 0 },
      qualification: { not_yet_passed: populationSize - rowCount, provisional: rowCount, finalized: 0, unresolved: 0 }, uncertainty: { bootstrap_origin: 0, post_bootstrap_gap: 0, local_invalid: 0 },
      population_transition_diagnostic: { bootstrap_unknown: 0, trusted_by_later_live_mark: 0, no_later_eligible_mark: 0, latest_mark_not_live_authority: 0, no_strictly_older_localized_conflict: 0, conflict_at_or_after_mark: 0, invalid_at_or_after_mark: 0, incomplete_post_mark_coverage: 0 },
    },
    recovery: { purpose: "fresh_bootstrap", generation: "1", start: "2026-08-08T15:58:00Z", end: "2026-08-08T15:59:00Z", supported_through: "2026-08-08T15:59:58Z", fence_reconciled: true, policy_waiting: false, work: { planned: String(populationSize), open: "0", completed_value: String(rowCount), completed_empty: String(populationSize - rowCount), failed: "0", canceled: "0", fenced: "0" }, rows: { consumed: String(rowCount), inserted: String(rowCount), duplicate: "0", conflict_or_withdrawal: "0", rejected: "0", fenced: "0", integrity: "0" } },
    tq: { desired_symbols: rows.map(row => row.symbol), pressure_mode: "normal", pressure_cause: "", aggregate_only: false, shed: false, retained_bound_hit: false, pressure_misses: 0, pressure_transitions: "0", pressure_fenced: "0", known_present: rowCount, known_absent: 0, unknown: 0, retained_trades: rowCount, retained_quotes: rowCount, retained_fingerprints: rowCount, facts: { consumed: String(rowCount * 2), applied: String(rowCount * 2), duplicate: "0", rejected: "0", fenced: "0", pressure_shed: "0", integrity: "0", normalized_trades: String(rowCount), normalized_quotes: String(rowCount), applied_trades: String(rowCount), applied_quotes: String(rowCount), pressure_shed_trades: "0", pressure_shed_quotes: "0" }, commands: { issued: String(rowCount), pending: "0", acknowledged: String(rowCount), failed: "0", fenced: "0", result_fenced: "0" } },
    checkpoint: { installed: false, eligible: "0", pressure_deferred: "0", projection_started: "0", projection_in_progress: "0", projected: "0", projection_rejected: "0", submit_rejected: "0", submitted: "0", outstanding: "0", in_progress: "0", pending: "0", completed: "0", failed: "0", canceled: "0", superseded: "0", usable_age_ms: 0, projection_total_ms: 0, projection_lock_ms: 0, artifact_bytes: "0", write_ms: 0, encode_ms: 0, reopen_validation_ms: 0 },
    operations: { sample_accounting_valid: true, queue_capacity_frames: 64, queue_current_frames: 0, queue_high_frames: 2, queue_current_bytes: 0, queue_high_bytes: 2048, deliveries: "10", consumer_deferred: "0", mean_processing_delay_ms: 1, max_processing_delay_ms: 2, max_processing_delay_one_second_ms: 2, delivery_latency_attribution: { aggregate: "10", tq: "0", control: "0", hydration_fence: "0", checkpoint: "0", timer: "0", unknown: "0", maximum_ms: 2, maximum_family: "aggregate" }, heap_alloc_bytes: "1024", heap_in_use_bytes: "2048", goroutines: 8, connection_recovery_attempts: "0" },
  };
}
