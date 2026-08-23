package operations

import (
	"context"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

// This production composition proof keeps the runtime timer quiescent, admits
// one 1,000-event live prefix through public APIs, and then opens exactly one
// evaluation/publication boundary.
func TestProductionCompositionCoalescesThousandAcceptedAggregates(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(10 * time.Second)
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := run.Shutdown(ctx); err != nil {
			t.Fatal(err)
		}
	})
	owner := run.Engine()
	ackAt := now.Add(-config.EvaluationDelay)
	applyControl(t, owner, binding, engine.ConnectionAttempt, 1, 1, engine.LivePosition{}, ackAt)
	applyControl(t, owner, binding, engine.AggregateCommandWriteResult, 1, 2, engine.LivePosition{}, ackAt)
	applyControl(t, owner, binding, engine.AggregateSubscriptionResult, 1, 2, engine.LivePosition{ConnectionEpoch: 1, FrameSequence: 1}, ackAt)
	completeHydration(t, owner, binding, engine.HydrationFreshBootstrap, 1, now)
	before := owner.ObserveSnapshot()
	beforeOps := owner.ObserveOperational()
	windowStart := binding.SessionStart()
	for index := 0; index < 1_000; index++ {
		closeValue := 10 + float64(index)/10_000
		input := engine.AggregateInput{SchemaVersion: engine.AggregateSchemaV1, BindingIdentity: binding.Identity(), Source: engine.AggregateSourceLive, Symbol: "AAA",
			WindowStart: windowStart, WindowEnd: windowStart.Add(time.Second), DeliveryTime: now,
			Values: engine.AggregateValues{Open: closeValue, High: closeValue, Low: closeValue, Close: closeValue, Volume: 100, VWAP: closeValue, AverageTradeSize: 10, ATSProvenance: engine.ATSLiveProviderAverage},
			Live:   engine.LivePosition{ConnectionEpoch: 1, FrameSequence: uint64(index + 2), ArrayIndex: 1}}
		admission, completion := owner.AdmitAggregate(context.Background(), input)
		if admission != engine.AdmissionAdmitted || completion == nil {
			t.Fatalf("aggregate %d admission=%s", index, admission)
		}
		result := <-completion
		want := engine.DispositionAggregateRevised
		if index == 0 {
			want = engine.DispositionAggregateInserted
		}
		if result.Code != want {
			t.Fatalf("aggregate %d disposition=%+v", index, result)
		}
	}
	middle := owner.ObserveSnapshot()
	middleOps := owner.ObserveOperational()
	if middle.Publication.PublicationID != before.Publication.PublicationID || middle.Publication.LastEngineSequence != before.Publication.LastEngineSequence {
		t.Fatalf("accepted prefix published early before=%d/%d after=%d/%d", before.Publication.PublicationID, before.Publication.LastEngineSequence, middle.Publication.PublicationID, middle.Publication.LastEngineSequence)
	}
	if middleOps.Aggregates.Consumed-beforeOps.Aggregates.Consumed != 1_000 || middleOps.Aggregates.Inserted-beforeOps.Aggregates.Inserted != 1 || middleOps.Aggregates.Revised-beforeOps.Aggregates.Revised != 999 || !middleOps.Aggregates.Reconciles() {
		t.Fatalf("prefix accounting before=%+v after=%+v", beforeOps.Aggregates, middleOps.Aggregates)
	}
	if middleOps.Admissions.Invalid != beforeOps.Admissions.Invalid || middleOps.Admissions.Canceled != beforeOps.Admissions.Canceled || middleOps.Admissions.Closed != beforeOps.Admissions.Closed || middleOps.Admissions.SequenceExhausted != beforeOps.Admissions.SequenceExhausted {
		t.Fatalf("queue rejection changed: before=%+v after=%+v", beforeOps.Admissions, middleOps.Admissions)
	}
	run.runEvaluationCycle(context.Background())
	after := owner.ObserveSnapshot()
	afterOps := owner.ObserveOperational()
	timing := owner.ObserveEvaluationTiming()
	population := after.Publication.AggregateEvaluation.Population
	if after.Publication.PublicationID != middle.Publication.PublicationID+1 || population.UniverseTotal != 1 || population.ValidPriorClose != 1 || population.TrustedRankableMark != 1 || population.CoveredPopulation != 1 || population.UnresolvedPopulation != 0 || !afterOps.Admissions.Reconciles(afterOps.QueueOccupancy) || !afterOps.Aggregates.Reconciles() {
		t.Fatalf("timer publication=%+v operations=%+v", after.Publication, afterOps)
	}
	if timing.Source != engine.AggregateEvaluationTimer || timing.Starts.Timer != 1 {
		t.Fatalf("no-fence runtime fallback timing=%+v", timing)
	}
}

func TestEvaluationCadenceSkipsMissedTicks(t *testing.T) {
	anchor := time.Unix(100, 0)
	cadence := time.Second
	for _, tc := range []struct {
		now  time.Time
		want time.Duration
	}{
		{anchor, time.Second},
		{anchor.Add(500 * time.Millisecond), 500 * time.Millisecond},
		{anchor.Add(3*time.Second + 250*time.Millisecond), 750 * time.Millisecond},
	} {
		if got := nextEvaluationCadenceDelay(anchor, cadence, tc.now); got != tc.want {
			t.Fatalf("now=%s delay=%s want=%s", tc.now, got, tc.want)
		}
	}
}
