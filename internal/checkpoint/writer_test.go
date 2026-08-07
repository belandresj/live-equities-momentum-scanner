package checkpoint

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestC7CADENCE01WriterCoalescingAccounting(t *testing.T) {
	store, err := NewStore(StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: "binding-a", ArtifactByteLimit: 4 << 20, OperationDeadline: 5 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	store.SetStepHookForTest(func(step WriteStep) error {
		if step == StepPayloadEncode {
			select {
			case <-entered:
			default:
				close(entered)
			}
			<-release
		}
		return nil
	})
	writer, err := NewWriter(context.Background(), store)
	if err != nil {
		t.Fatal(err)
	}
	request := func(sequence uint64) Request {
		image := fixtureImage(sequence, 2)
		image.T0 = image.T0.Add(time.Duration(sequence) * 30 * time.Second)
		image.CreatedAt = image.T0.Add(time.Second)
		return Request{BindingIdentity: "binding-a", RequestID: sequence, ArtifactSequence: sequence, T0: image.T0, Image: image}
	}
	if got := writer.Submit(request(1)); got.Disposition != SubmitAccepted {
		t.Fatal(got)
	}
	select {
	case <-entered:
	case <-time.After(2 * time.Second):
		t.Fatal("writer did not enter")
	}
	if got := writer.Submit(request(2)); got.Disposition != SubmitAccepted || got.Superseded != nil {
		t.Fatal(got)
	}
	third := request(3)
	got := writer.Submit(third)
	if got.Disposition != SubmitAccepted || got.Superseded == nil || got.Superseded.RequestID != 2 || got.Superseded.Disposition != TerminalSuperseded {
		t.Fatalf("replacement=%+v", got)
	}
	third.Image.Binding.Identity = "mutated-after-submit"
	if accounting := writer.Accounting(); !accounting.Reconciles() || accounting != (WriterAccounting{Submitted: 3, InProgress: 1, Pending: 1, Superseded: 1}) {
		t.Fatalf("paused accounting=%+v", accounting)
	}
	close(release)
	seen := map[uint64]TerminalResult{}
	for len(seen) != 2 {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		result, err := writer.NextResult(ctx)
		cancel()
		if err != nil {
			t.Fatalf("terminals=%+v err=%v", seen, err)
		}
		seen[result.RequestID] = result
	}
	if seen[1].Disposition != TerminalCompleted || seen[3].Disposition != TerminalCompleted {
		t.Fatalf("terminals=%+v", seen)
	}
	if loaded := store.Load(context.Background()); loaded.Disposition != LoadedLatest || loaded.Candidate.Image.Sequence != 3 || loaded.Candidate.Image.Binding.Identity != "binding-a" {
		t.Fatalf("detached latest=%+v", loaded)
	}
	if accounting := writer.Accounting(); !accounting.Reconciles() || accounting != (WriterAccounting{Submitted: 3, Completed: 2, Superseded: 1}) {
		t.Fatalf("terminal accounting=%+v", accounting)
	}
	writer.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := writer.Wait(ctx); err != nil {
		t.Fatal(err)
	}

	t.Run("shutdown terminals in-progress and pending", func(t *testing.T) {
		cancelStore, err := NewStore(StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: "binding-a", ArtifactByteLimit: 4 << 20, OperationDeadline: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		blocked, unblock := make(chan struct{}), make(chan struct{})
		cancelStore.SetStepHookForTest(func(step WriteStep) error {
			if step == StepPayloadEncode {
				select {
				case <-blocked:
				default:
					close(blocked)
				}
				<-unblock
			}
			return nil
		})
		cancelWriter, err := NewWriter(context.Background(), cancelStore)
		if err != nil {
			t.Fatal(err)
		}
		if cancelWriter.Submit(request(4)).Disposition != SubmitAccepted {
			t.Fatal("first cancel submission")
		}
		select {
		case <-blocked:
		case <-time.After(2 * time.Second):
			t.Fatal("cancel writer did not enter")
		}
		if cancelWriter.Submit(request(5)).Disposition != SubmitAccepted {
			t.Fatal("pending cancel submission")
		}
		cancelWriter.Close()
		close(unblock)
		terminals := map[uint64]TerminalResult{}
		for len(terminals) != 2 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			result, err := cancelWriter.NextResult(ctx)
			cancel()
			if err != nil {
				t.Fatalf("cancel terminals=%+v err=%v", terminals, err)
			}
			terminals[result.RequestID] = result
		}
		if terminals[4].Disposition != TerminalCanceled || terminals[5].Disposition != TerminalCanceled {
			t.Fatalf("cancel terminals=%+v", terminals)
		}
		wait, stop := context.WithTimeout(context.Background(), 5*time.Second)
		defer stop()
		if err := cancelWriter.Wait(wait); err != nil {
			t.Fatal(err)
		}
		if accounting := cancelWriter.Accounting(); !accounting.Reconciles() || accounting != (WriterAccounting{Submitted: 2, Canceled: 2}) {
			t.Fatalf("cancel accounting=%+v", accounting)
		}
	})

	t.Run("undrained delivery is bounded and shutdown completes", func(t *testing.T) {
		boundedStore, err := NewStore(StoreConfig{Directory: filepath.Join(t.TempDir(), "checkpoints"), BindingIdentity: "binding-a", ArtifactByteLimit: 4 << 20, OperationDeadline: 5 * time.Second})
		if err != nil {
			t.Fatal(err)
		}
		boundedWriter, err := NewWriter(context.Background(), boundedStore)
		if err != nil {
			t.Fatal(err)
		}
		for sequence := uint64(1); sequence <= 256; sequence++ {
			if got := boundedWriter.Submit(request(sequence + 10)); got.Disposition != SubmitAccepted {
				t.Fatalf("submit %d=%+v", sequence, got)
			}
			deadline := time.Now().Add(5 * time.Second)
			for boundedWriter.Accounting().Completed < sequence {
				if time.Now().After(deadline) {
					t.Fatalf("completion %d accounting=%+v", sequence, boundedWriter.Accounting())
				}
				time.Sleep(time.Millisecond)
			}
		}
		if got := boundedWriter.Submit(request(1000)); got.Disposition != SubmitRejected {
			t.Fatalf("delivery saturation accepted=%+v", got)
		}
		boundedWriter.Close()
		wait, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := boundedWriter.Wait(wait); err != nil {
			t.Fatalf("undrained shutdown blocked: %v", err)
		}
		for index := 0; index < 256; index++ {
			ctx, stop := context.WithTimeout(context.Background(), time.Second)
			_, err := boundedWriter.NextResult(ctx)
			stop()
			if err != nil {
				t.Fatalf("terminal %d missing: %v", index, err)
			}
		}
		if accounting := boundedWriter.Accounting(); !accounting.Reconciles() || accounting.Completed != 256 {
			t.Fatalf("bounded accounting=%+v", accounting)
		}
	})
}
