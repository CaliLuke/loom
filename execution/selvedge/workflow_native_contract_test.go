package selvedge_test

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"errors"
	"path/filepath"
	"testing"
	"time"

	sv "github.com/CaliLuke/selvedge"
)

type nativeBlockingInput struct {
	entered chan struct{}
	release chan struct{}
}

func (v nativeBlockingInput) MarshalJSON() ([]byte, error) {
	close(v.entered)
	<-v.release
	return []byte(`"held"`), nil
}
func finishNativeCleanup(t *testing.T, shutdown func(context.Context) error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	for {
		err := shutdown(ctx)
		if !errors.Is(err, sv.ErrShutdownInterrupted) {
			if err != nil {
				t.Fatal(err)
			}
			return
		}
		if ctx.Err() != nil {
			t.Fatal("cleanup did not complete", err)
		}
	}
}
func nativeWorkerCompletion(t *testing.T) {
	a := sv.NewApp(sv.WithDatabase(filepath.Join(t.TempDir(), "worker.db")), sv.WithShutdownTimeout(time.Nanosecond))
	if err := a.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	first := a.Shutdown(context.Background())
	if first != nil && (!errors.Is(first, sv.ErrShutdownInterrupted) || !errors.Is(first, context.DeadlineExceeded)) {
		t.Fatal(first)
	}
	t.Logf("first: interrupted=%t deadline=%t", errors.Is(first, sv.ErrShutdownInterrupted), errors.Is(first, context.DeadlineExceeded))
	finishNativeCleanup(t, a.Shutdown)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := a.Shutdown(ctx); err != nil {
		t.Fatal("completed result lost to cancelled waiter", err)
	}
	t.Log("worker: completed; cancelled repeat returned nil")
}
func nativeProducerCompletion(t *testing.T) {
	init, cancelInit := context.WithCancel(context.Background())
	p, err := sv.OpenProducer(init, sv.WithDatabase(filepath.Join(t.TempDir(), "producer.db")), sv.WithShutdownTimeout(time.Millisecond))
	cancelInit()
	if err != nil {
		t.Fatal(err)
	}
	raw := jsontext.Value(`{ "n": 1.00, "s": "<x>" }`)
	id, err := p.StartWorkflowEncoded(context.Background(), "unregistered", []byte(raw), sv.WithInstanceID("stable"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := p.Describe(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if d.Status != sv.StatusRunning || !bytes.Equal(d.InputData, raw) || d.ParentInstanceID != "" || d.ParentKey != "" {
		t.Fatalf("unexpected description: %+v", d)
	}
	held := nativeBlockingInput{make(chan struct{}), make(chan struct{})}
	admission := make(chan error, 1)
	go func() {
		_, err := p.StartWorkflowByName(context.Background(), "held", held, sv.WithInstanceID("held"))
		admission <- err
	}()
	<-held.entered
	first := p.Shutdown(context.Background())
	if !errors.Is(first, sv.ErrShutdownInterrupted) || !errors.Is(first, context.DeadlineExceeded) {
		t.Fatal("expected classified timeout", first)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	next := p.Shutdown(ctx)
	if !errors.Is(next, sv.ErrShutdownInterrupted) || !errors.Is(next, context.Canceled) {
		t.Fatal("expected classified cancellation", next)
	}
	close(held.release)
	if err := <-admission; err != nil {
		t.Fatal(err)
	}
	finishNativeCleanup(t, p.Shutdown)
	if err := p.Shutdown(ctx); err != nil {
		t.Fatal("completed result lost to cancelled waiter", err)
	}
	t.Log("producer: init cancellation safe; exact bytes; timeout/cancellation classified; drained; cancelled repeat returned nil")
}

func TestWorkflowNativePrerequisite(t *testing.T) {
	t.Run("worker", nativeWorkerCompletion)
	t.Run("producer", nativeProducerCompletion)
}
