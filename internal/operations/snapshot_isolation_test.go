package operations

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"
)

func TestP3SnapshotCaptureUsesCachedDiagnostics(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(time.Hour)
	config := DefaultConfig()
	config.SampleCadence = 10 * time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownSnapshotIsolationRuntime(t, run) })

	collectionsBefore := run.diagnosticCollections.Load()
	const total = 10_000
	const workers = 16
	ids := make(chan uint64, total)
	errorsFound := make(chan error, 1)
	var group sync.WaitGroup
	for worker := 0; worker < workers; worker++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for index := 0; index < total/workers; index++ {
				capture, captureErr := run.CaptureSnapshot()
				if captureErr != nil {
					recordSnapshotIsolationError(errorsFound, captureErr)
					return
				}
				view, ok := InspectSnapshotCapture(capture)
				if !ok || view.SampleID == 0 || view.Engine.Publication.PublicationID == 0 ||
					!view.Metrics.SampledAt.Equal(view.SampledAt) {
					recordSnapshotIsolationError(errorsFound, fmt.Errorf("invalid concurrent capture: ok=%t view=%+v", ok, view))
					return
				}
				ids <- view.SampleID
			}
		}()
	}
	group.Wait()
	close(ids)
	select {
	case err := <-errorsFound:
		t.Fatal(err)
	default:
	}

	got := make([]uint64, 0, total)
	for id := range ids {
		got = append(got, id)
	}
	if len(got) != total {
		t.Fatalf("concurrent captures=%d want=%d", len(got), total)
	}
	sort.Slice(got, func(i, j int) bool { return got[i] < got[j] })
	for index, id := range got {
		want := uint64(index + 1)
		if id != want {
			t.Fatalf("capture sequence gap at index %d: got=%d want=%d", index, id, want)
		}
	}
	if collectionsAfter := run.diagnosticCollections.Load(); collectionsAfter != collectionsBefore {
		t.Fatalf("capture path collected diagnostics: before=%d after=%d", collectionsBefore, collectionsAfter)
	}
}

func TestP3DiagnosticSampleAgeDoesNotHidePublication(t *testing.T) {
	binding := operationsBinding(t)
	now := binding.SessionStart().Add(time.Hour)
	config := DefaultConfig()
	config.SampleCadence = time.Minute
	run, err := New(context.Background(), binding, config, func() time.Time { return now })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownSnapshotIsolationRuntime(t, run) })

	initial := run.diagnostics.Load()
	if initial == nil {
		t.Fatal("startup diagnostics sample was not installed")
	}
	baseline, err := run.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	baselineView, ok := InspectSnapshotCapture(baseline)
	if !ok || baselineView.Engine.Publication.PublicationID == 0 {
		t.Fatalf("baseline capture unavailable: %+v", baselineView)
	}

	cases := []struct {
		name        string
		collectedAt time.Time
		missing     bool
	}{
		{name: "old", collectedAt: now.Add(-2*config.SampleCadence - time.Nanosecond)},
		{name: "future", collectedAt: now.Add(time.Nanosecond)},
		{name: "missing", missing: true},
	}
	for _, test := range cases {
		t.Run(test.name, func(t *testing.T) {
			if test.missing {
				run.diagnostics.Store(nil)
			} else {
				metrics := cloneMetrics(initial.metrics)
				run.diagnostics.Store(&diagnosticsSample{metrics: metrics, collectedAt: test.collectedAt})
			}

			capture, captureErr := run.CaptureSnapshot()
			if captureErr != nil {
				t.Fatal(captureErr)
			}
			view, viewOK := InspectSnapshotCapture(capture)
			if !viewOK || view.Engine.Publication.PublicationID != baselineView.Engine.Publication.PublicationID {
				t.Fatalf("diagnostics condition hid publication: ok=%t baseline=%d got=%d", viewOK, baselineView.Engine.Publication.PublicationID, view.Engine.Publication.PublicationID)
			}
			if view.Metrics.AccountingValid {
				t.Fatalf("invalid diagnostics condition reported valid accounting: %+v", view.Metrics)
			}
			if !view.Metrics.SampledAt.Equal(now) {
				t.Fatalf("capture timestamp was not composed from the capture clock: got=%s want=%s", view.Metrics.SampledAt, now)
			}
			if test.missing {
				if !view.Metrics.diagnosticSampledAt.IsZero() {
					t.Fatalf("missing sample acquired a collection time: %s", view.Metrics.diagnosticSampledAt)
				}
			} else if !view.Metrics.diagnosticSampledAt.Equal(test.collectedAt) {
				t.Fatalf("diagnostic collection time was restamped: got=%s want=%s", view.Metrics.diagnosticSampledAt, test.collectedAt)
			}
		})
	}
}

func TestCaptureSequenceOverflowFailsWithoutWrapping(t *testing.T) {
	binding := operationsBinding(t)
	run, err := New(context.Background(), binding, DefaultConfig(), func() time.Time { return binding.SessionStart().Add(time.Hour) })
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownSnapshotIsolationRuntime(t, run) })
	run.captureSequence.Store(^uint64(0))
	if _, err := run.CaptureSnapshot(); err == nil {
		t.Fatal("capture sequence overflow was accepted")
	}
	if got := run.captureSequence.Load(); got != ^uint64(0) {
		t.Fatalf("overflow changed sequence: %d", got)
	}
}

func recordSnapshotIsolationError(target chan<- error, value error) {
	select {
	case target <- value:
	default:
	}
}

func shutdownSnapshotIsolationRuntime(t *testing.T, run *Runtime) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := run.Shutdown(ctx); err != nil {
		t.Errorf("runtime shutdown: %v", err)
	}
}
