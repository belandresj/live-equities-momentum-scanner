package snapshotapi

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"
	"unsafe"

	"github.com/belandresj/live-equities-momentum-scanner/internal/operations"
)

type fixedCaptureSource struct {
	capture operations.SnapshotCapture
}

func (source fixedCaptureSource) CaptureSnapshot() (operations.SnapshotCapture, error) {
	return source.capture, nil
}

// These mirrors exist only to construct the dangerous invalid-seal
// counterexample. Production packages still cannot construct or mutate a
// SnapshotCapture, and this test immediately sends the corrupted capture
// through the actual mapper/handler/server path.
type snapshotCaptureMirror struct {
	sealed *sealedSnapshotCaptureMirror
}

type sealedSnapshotCaptureMirror struct {
	view operations.SnapshotCaptureView
}

func TestMappingFailureTraversesHTTPServerDiagnostics(t *testing.T) {
	runtime, binding, now := newSnapshotRuntime(t)
	makeSnapshotReady(t, runtime.Engine(), binding, *now)
	t.Cleanup(func() { shutdownSnapshotRuntime(t, runtime) })
	capture, err := runtime.CaptureSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	mirror := (*snapshotCaptureMirror)(unsafe.Pointer(&capture))
	if mirror.sealed == nil {
		t.Fatal("runtime returned an unsealed capture")
	}
	mirror.sealed.view.Engine.Publication.AggregateEvaluation.Population.ValidPriorClose++

	server, err := Listen(fixedCaptureSource{capture: capture}, ServerConfig{Address: "127.0.0.1:0"})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			t.Error(err)
		}
	})
	client := &http.Client{Timeout: 2 * time.Second}
	response, err := client.Get("http://" + server.Address() + "/api/v2/snapshot")
	if err != nil {
		t.Fatal(err)
	}
	body, readErr := io.ReadAll(response.Body)
	_ = response.Body.Close()
	if readErr != nil || response.StatusCode != http.StatusServiceUnavailable || string(body) != `{"error":"snapshot_unavailable"}` {
		t.Fatalf("public mapper failure=%d %s read=%v", response.StatusCode, body, readErr)
	}
	var public map[string]json.RawMessage
	if err := json.Unmarshal(body, &public); err != nil || len(public) != 1 {
		t.Fatalf("public mapper body exposed diagnostics: %s err=%v", body, err)
	}

	latest, ok := server.LatestMappingFailure()
	if !ok || latest.Invariant != "population_prior_close_identity" || latest.Route != "/api/v2/snapshot" || latest.PublicationID == "0" ||
		latest.LastEngineSequence == "0" || latest.Lifecycle != "live" || latest.RankingMode == "" {
		t.Fatalf("private mapper diagnostic=%+v ok=%t", latest, ok)
	}
	select {
	case update := <-server.MappingFailures():
		if update != latest {
			t.Fatalf("mapper notification=%+v latest=%+v", update, latest)
		}
	case <-time.After(time.Second):
		t.Fatal("mapper diagnostic notification was not emitted")
	}
}
