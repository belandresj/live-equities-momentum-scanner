package replaymode

import (
	"context"
	"os"
	"sort"
	"strconv"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
	"github.com/belandresj/live-equities-momentum-scanner/internal/replayartifact"
	"github.com/belandresj/live-equities-momentum-scanner/internal/snapshotapi"
)

// TestRetainedArtifactLiveBootstrapIntegrity is the third, explicitly selected
// rung of P-LIVE-BOOTSTRAP-INTEGRITY. It validates the complete retained
// artifact, distills a realistic at-least-167,046-record prefix through the existing
// playback cursor, and applies that prefix through the production hydration
// ledger and ingress fence without any provider request.
func TestRetainedArtifactLiveBootstrapIntegrity(t *testing.T) {
	if testing.Short() {
		t.Skip("retained artifact bootstrap diagnostic is not ordinary verification")
	}
	artifactPath, referenceDirectory := os.Getenv("LIVE_BOOTSTRAP_ARTIFACT"), os.Getenv("LIVE_BOOTSTRAP_REFERENCE_DIR")
	if artifactPath == "" || referenceDirectory == "" {
		t.Skip("explicit retained artifact and reference paths are required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()
	binding, err := cachedBinding(ctx, "2026-08-07", referenceDirectory)
	if err != nil {
		t.Fatal(err)
	}
	start := binding.SessionStart()
	handle, err := replayartifact.OpenValidatedContext(ctx, artifactPath, replayartifact.ValidationPlan{Binding: binding, Start: start, End: binding.SessionEnd(),
		ExpectedMode: replayartifact.CompleteFinalBars, MaximumBytes: MaximumArtifactBytes, MaximumRecords: MaximumRecords})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	metadata := handle.Metadata()
	if metadata.AggregateRecords != 7_671_171 || metadata.CoverageEntries != 5_691 || len(binding.UniverseSymbols()) != 5_691 {
		t.Fatalf("retained manifest records=%d coverage=%d population=%d", metadata.AggregateRecords, metadata.CoverageEntries, len(binding.UniverseSymbols()))
	}
	cursor, err := handle.BeginPlaybackContext(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if evidence, startErr := cursor.StartContext(ctx); startErr != nil || !evidence.Valid() || !evidence.Complete() {
		t.Fatalf("playback start: %+v err=%v", evidence, startErr)
	}
	const (
		wantedRecords          = 167_046
		expectedAppliedRecords = 166_984
		conflicts              = 58
	)
	records := make(map[string][]engine.HydrationRow, len(binding.UniverseSymbols()))
	recordCount := 0
	prefixEnd := start
	for group := start; !group.After(binding.SessionEnd()) && recordCount < wantedRecords; group = group.Add(time.Second) {
		for {
			record, ok, readErr := cursor.NextRecordContext(ctx, group)
			if readErr != nil {
				t.Fatal(readErr)
			}
			if !ok {
				break
			}
			values := record.Values()
			row, rowErr := engine.NewHydrationRow(record.Symbol(), record.WindowStart(), record.WindowEnd(), engine.AggregateValues{
				Open: values.Open, High: values.High, Low: values.Low, Close: values.Close, Volume: values.Volume, VWAP: values.VWAP,
				AverageTradeSize: values.AverageTradeSize, ATSProvenance: engine.ATSProvenance(values.ATSProvenance),
			})
			if rowErr != nil {
				t.Fatal(rowErr)
			}
			records[record.Symbol()] = append(records[record.Symbol()], row)
			recordCount++
		}
		if _, finishErr := cursor.FinishGroupContext(ctx, group); finishErr != nil {
			t.Fatal(finishErr)
		}
		prefixEnd = group.Add(time.Second)
	}
	if recordCount < wantedRecords || !start.Before(prefixEnd) {
		t.Fatalf("prefix records=%d end=%s", recordCount, prefixEnd)
	}

	now := prefixEnd
	runtimeConfig := operations.DefaultConfig()
	runtimeConfig.SampleCadence = 10 * time.Minute
	runtime, err := operations.New(ctx, binding, runtimeConfig, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		shutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()
		if shutdownErr := runtime.Shutdown(shutdown); shutdownErr != nil {
			t.Errorf("runtime shutdown: %v", shutdownErr)
		}
	}()
	owner := runtime.Engine()
	control := func(kind engine.ConnectionControlKind, position engine.LivePosition, token uint64) {
		input := engine.ConnectionControlInput{SchemaVersion: engine.ConnectionControlSchemaV1, BindingIdentity: binding.Identity(), Kind: kind,
			ConnectionEpoch: 1, Position: position, ReceiptTime: now, CommandToken: token, Outcome: engine.ControlSucceeded}
		result, completion := owner.AdmitConnectionControl(ctx, input)
		if result != engine.AdmissionAdmitted || (<-completion).Code != engine.DispositionConnectionControlApplied {
			t.Fatalf("connection control %s failed", kind)
		}
	}
	control(engine.ConnectionAttempt, engine.LivePosition{}, 1)
	control(engine.AggregateCommandWriteResult, engine.LivePosition{}, 2)
	control(engine.AggregateSubscriptionResult, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, 2)
	maximum := int64(len(binding.UniverseSymbols())) * int64(prefixEnd.Sub(start)/time.Second)
	planInput := engine.HydrationPlanInput{SchemaVersion: engine.HydrationPlanSchemaV1, BindingIdentity: binding.Identity(), Purpose: engine.HydrationFreshBootstrap, ConnectionEpoch: 1,
		Budgets: engine.HydrationPlanBudgets{Workers: 8, RowsPerChunk: 256, MaximumResponseBytes: 512 << 20, MaximumNormalizedRecords: maximum, MaximumResidentRecords: 8 * 57_600}}
	result, planned := owner.AdmitHydrationPlan(ctx, planInput)
	if result != engine.AdmissionAdmitted {
		t.Fatal("hydration plan admission failed")
	}
	plan := <-planned
	requests := plan.Plan.Requests()
	if plan.Code != engine.DispositionHydrationPlanApplied || len(requests) == 0 || len(requests) > len(binding.UniverseSymbols()) || plan.Plan.End() != prefixEnd {
		t.Fatalf("hydration plan code=%s requests=%d prefix_end=%s planned_end=%s", plan.Code, len(requests), prefixEnd, plan.Plan.End())
	}

	// Seed 58 deterministic REST/live conflicts before historical chunks and a
	// post-handoff live-only identity needed by the one-second-beyond fence.
	requestSet := make(map[string]struct{}, len(requests))
	for _, request := range requests {
		requestSet[request.Symbol()] = struct{}{}
	}
	conflictSymbols := make([]string, 0, conflicts)
	for symbol, rows := range records {
		_, requested := requestSet[symbol]
		if requested && len(rows) > 0 && !rows[len(rows)-1].WindowEnd().Before(prefixEnd.Add(-16*time.Minute)) {
			conflictSymbols = append(conflictSymbols, symbol)
		}
	}
	sort.Strings(conflictSymbols)
	conflictSymbols = conflictSymbols[:conflicts]
	now = prefixEnd
	for index, symbol := range conflictSymbols {
		row := records[symbol][len(records[symbol])-1]
		values := row.Values()
		live := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
			Symbol: symbol, WindowStart: row.WindowStart(), WindowEnd: row.WindowEnd(), Values: values, DeliveryTime: row.WindowEnd(),
			Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(index + 2)}}
		live.Values.ATSProvenance = engine.ATSLiveProviderAverage
		admitted, completed := owner.AdmitAggregate(ctx, live)
		if admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionAggregateInserted {
			t.Fatalf("live conflict seed %d failed", index)
		}
	}
	now = prefixEnd.Add(5 * time.Second)
	tailSymbol := requests[0].Symbol()
	tailValues := engine.AggregateValues{Open: 10, High: 10, Low: 10, Close: 10, Volume: 100, VWAP: 10, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage}
	tail := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: tailSymbol, WindowStart: prefixEnd, WindowEnd: prefixEnd.Add(time.Second), Values: tailValues, DeliveryTime: prefixEnd.Add(time.Second),
		Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: conflicts + 2}}
	if admitted, completed := owner.AdmitAggregate(ctx, tail); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionAggregateInserted {
		t.Fatal("live tail failed")
	}

	var fence engine.HydrationFenceCommand
	processedRecords := 0
	for _, request := range requests {
		rows := records[request.Symbol()]
		if len(rows) > 0 {
			processedRecords += len(rows)
			chunks := (len(rows) + 255) / 256
			for ordinal := 0; ordinal < chunks; ordinal++ {
				first, last := ordinal*256, min((ordinal+1)*256, len(rows))
				chunk, chunkErr := engine.NewHydrationChunkInput(request, request.ResultID(), ordinal, chunks, int64(first), int64(len(rows)), rows[first:last])
				if chunkErr != nil {
					t.Fatal(chunkErr)
				}
				admitted, completed := owner.AdmitHydrationChunk(ctx, chunk)
				if admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionHydrationChunkApplied {
					t.Fatalf("hydration chunk failed for %s", request.Symbol())
				}
			}
			terminal, terminalErr := engine.NewHydrationTerminalInput(request, request.ResultID(), engine.HydrationCompletedValue, engine.HydrationReasonNone, 1, 1, int64(len(rows))*100, int64(len(rows)), int64(chunks), int64(len(rows)))
			if terminalErr != nil {
				t.Fatal(terminalErr)
			}
			_, completed := owner.AdmitHydrationTerminal(ctx, terminal)
			fence = (<-completed).FenceCommand
		} else {
			terminal, terminalErr := engine.NewHydrationTerminalInput(request, request.ResultID(), engine.HydrationCompletedEmpty, engine.HydrationReasonNone, 1, 1, 10, 0, 0, 0)
			if terminalErr != nil {
				t.Fatal(terminalErr)
			}
			_, completed := owner.AdmitHydrationTerminal(ctx, terminal)
			fence = (<-completed).FenceCommand
		}
	}
	fenceInput, err := engine.NewAggregateIngressFenceInput(fence, engine.AggregateIngressFenceComplete, conflicts+2, 1, now)
	if err != nil {
		t.Fatal(err)
	}
	_, completedFence := owner.AdmitAggregateIngressFence(ctx, fenceInput)
	fenceResult := <-completedFence
	view := owner.ObserveOperational()
	if fenceResult.Code != engine.DispositionAggregateIngressFenceApplied || view.Lifecycle != "live" || view.Suppression != "" || view.Watermark == nil || !view.Watermark.Equal(prefixEnd.Add(time.Second)) {
		t.Fatalf("cached bootstrap fence=%+v lifecycle=%s suppression=%s watermark=%v", fenceResult, view.Lifecycle, view.Suppression, view.Watermark)
	}
	if view.Hydration.Accounting.Planned != uint64(len(requests)) || view.Hydration.Rows.Consumed != uint64(processedRecords) || view.Hydration.Rows.ConflictOrWithdrawal != conflicts || !view.Hydration.FenceReconciled {
		t.Fatalf("cached bootstrap work=%+v rows=%+v", view.Hydration.Accounting, view.Hydration.Rows)
	}
	if processedRecords != expectedAppliedRecords {
		t.Fatalf("retained applied records=%d want=%d", processedRecords, expectedAppliedRecords)
	}
	capture, captureErr := runtime.CaptureSnapshot()
	if captureErr != nil {
		t.Fatal(captureErr)
	}
	snapshot, mapErr := snapshotapi.Map(capture)
	if mapErr != nil {
		t.Fatalf("retained first-ready capture rejected: %v", mapErr)
	}
	if !snapshot.Status.BackendReady || !snapshot.Status.RankingCurrent || snapshot.Publication.Lifecycle != "live" || snapshot.Ranking.Mode != "qualified_current" || len(snapshot.Rows) == 0 {
		t.Fatalf("retained first-ready publication=%+v status=%+v ranking=%+v rows=%d", snapshot.Publication, snapshot.Status, snapshot.Ranking, len(snapshot.Rows))
	}

	// The failed live observation sampled HTTP roughly 400 ms after the last
	// retained engine status line, so prove the first ordinary post-fence live
	// publication as a separate boundary rather than assuming it was the fence
	// publication. This is the smallest deterministic transition that can
	// replace the first-ready publication before an HTTP request observes it.
	now = prefixEnd.Add(6 * time.Second)
	postFence := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive,
		Symbol: tailSymbol, WindowStart: prefixEnd.Add(time.Second), WindowEnd: prefixEnd.Add(2 * time.Second), Values: tailValues,
		DeliveryTime: prefixEnd.Add(2 * time.Second), Live: engine.LivePosition{ConnectionEpoch: 1, FrameSequence: conflicts + 3}}
	if admitted, completed := owner.AdmitAggregate(ctx, postFence); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionAggregateInserted {
		t.Fatal("post-fence live aggregate failed")
	}
	if admitted, completed := owner.AdmitTimer(ctx); admitted != engine.AdmissionAdmitted || (<-completed).Code != engine.DispositionTimerApplied {
		t.Fatal("post-fence evaluation timer failed")
	}
	postCapture, postCaptureErr := runtime.CaptureSnapshot()
	if postCaptureErr != nil {
		t.Fatal(postCaptureErr)
	}
	postSnapshot, postMapErr := snapshotapi.Map(postCapture)
	if postMapErr != nil {
		t.Fatalf("retained post-fence capture rejected: %v", postMapErr)
	}
	firstPublicationID, firstIDErr := strconv.ParseUint(snapshot.Publication.ID, 10, 64)
	postPublicationID, postIDErr := strconv.ParseUint(postSnapshot.Publication.ID, 10, 64)
	if firstIDErr != nil || postIDErr != nil || postPublicationID <= firstPublicationID {
		t.Fatalf("post-fence publication identity first=%q post=%q errors=%v/%v", snapshot.Publication.ID, postSnapshot.Publication.ID, firstIDErr, postIDErr)
	}
	if !postSnapshot.Status.BackendReady || !postSnapshot.Status.RankingCurrent || postSnapshot.Publication.Lifecycle != "live" ||
		postSnapshot.Publication.BindingIdentity != snapshot.Publication.BindingIdentity || postSnapshot.Ranking.Mode != "qualified_current" || len(postSnapshot.Rows) == 0 {
		t.Fatalf("retained post-fence publication=%+v status=%+v", postSnapshot.Publication, postSnapshot.Status)
	}
	t.Logf("retained bootstrap: artifact_records=%d binding_symbols=%d prefix=%s..%s prefix_records=%d live_overlay=%d conflicts=%d",
		metadata.AggregateRecords, len(binding.UniverseSymbols()), start.Format(time.RFC3339), prefixEnd.Format(time.RFC3339), processedRecords, conflicts+1, view.Hydration.Rows.ConflictOrWithdrawal)
}
