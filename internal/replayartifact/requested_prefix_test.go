package replayartifact

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/belandresj/live-equities-momentum-scanner/internal/engine"
)

func TestRequestedPrefixCountsReusePreparedPlayback(t *testing.T) {
	binding := replayArtifactTestBinding(t, []string{"AAA", "BBB"})
	start := binding.SessionStart()
	end := start.Add(3 * time.Second)
	requestedEnd := start.Add(2 * time.Second)
	artifactContextValue := artifactContext{mode: CompleteFinalBars, bindingID: binding.Identity(), universeID: binding.UniverseIdentity(), date: binding.TradingDate(),
		sessionStart: binding.SessionStart(), sessionEnd: binding.SessionEnd(), replayStart: start, replayEnd: end}
	values := engine.AggregateValues{Open: 10, High: 11, Low: 9, Close: 10, Volume: 1, VWAP: 10, AverageTradeSize: 1,
		ATSProvenance: engine.ATSRESTFloorVolumeOverTrades}
	records := []canonicalRecord{
		{logicalDeliveryTime: start.Add(time.Second), symbol: "AAA", windowStart: start, windowEnd: start.Add(time.Second), values: values},
		{logicalDeliveryTime: requestedEnd, symbol: "BBB", windowStart: start.Add(time.Second), windowEnd: requestedEnd, values: values},
		{logicalDeliveryTime: end, symbol: "AAA", windowStart: requestedEnd, windowEnd: end, values: values},
	}
	artifact, metadata, err := buildCanonical(artifactContextValue, []string{"AAA", "BBB"}, records, 1<<20, 10)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "complete.jsonl")
	if err := os.WriteFile(path, artifact, 0o600); err != nil {
		t.Fatal(err)
	}
	handle, err := OpenValidatedContext(context.Background(), path, ValidationPlan{Binding: binding, Start: start, End: end, ExpectedMode: CompleteFinalBars, MaximumBytes: 1 << 20, MaximumRecords: 10})
	if err != nil {
		t.Fatal(err)
	}
	defer handle.Close()
	cursor, err := handle.BeginPlaybackThroughContext(context.Background(), requestedEnd)
	if err != nil {
		t.Fatal(err)
	}
	prefix, err := cursor.RequestedPrefix()
	if err != nil || !prefix.Valid() || prefix.ArtifactID() != metadata.ArtifactID || prefix.BindingID() != binding.Identity() ||
		prefix.RequestedEnd() != requestedEnd || prefix.PrefixRecords() != 2 {
		t.Fatalf("requested prefix=%+v err=%v", prefix, err)
	}
	counts := prefix.RecordCounts()
	if counts["AAA"] != 1 || counts["BBB"] != 1 || len(counts) != 2 {
		t.Fatalf("requested prefix counts=%v", counts)
	}
	counts["AAA"] = 99
	detached := prefix.RecordCounts()
	if detached["AAA"] != 1 {
		t.Fatalf("caller mutated requested prefix evidence: %v", detached)
	}

	started, err := cursor.StartContext(context.Background())
	if err != nil || !started.Valid() || started.RequestedEnd() != requestedEnd {
		t.Fatalf("playback start=%+v err=%v", started, err)
	}
	var streamed uint64
	for group := start; !group.After(requestedEnd); group = group.Add(time.Second) {
		for {
			record, ok, nextErr := cursor.NextRecordContext(context.Background(), group)
			if nextErr != nil {
				t.Fatal(nextErr)
			}
			if !ok {
				break
			}
			if !record.Valid() {
				t.Fatal("invalid streamed record evidence")
			}
			streamed++
		}
		if _, err := cursor.FinishGroupContext(context.Background(), group); err != nil {
			t.Fatal(err)
		}
	}
	terminal, err := cursor.RequestedEndContext(context.Background())
	if err != nil || !terminal.Valid() || terminal.PrefixRecords() != streamed || streamed != prefix.PrefixRecords() {
		t.Fatalf("requested end=%+v streamed=%d err=%v", terminal, streamed, err)
	}
}
