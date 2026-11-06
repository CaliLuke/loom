package selvedge_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	sv "github.com/CaliLuke/selvedge"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/execution"
	adapter "github.com/CaliLuke/loom/execution/selvedge"
	"github.com/CaliLuke/loom/execution/selvedge/examples/durable/contract"
	"github.com/CaliLuke/loom/internal/testprocess"
)

type (
	workflowProcess struct {
		command *testprocess.Cmd
		done    chan struct{}
		err     error
		logs    *workflowProcessLog
	}
	workflowProcessLog struct {
		mu     sync.Mutex
		buffer bytes.Buffer
	}
	workflowJobView struct {
		// ID identifies the durable job.
		ID string
		// State reports the native lifecycle status.
		State sv.WorkflowStatus
		// Output contains completed metadata.
		Output contract.Metadata
		// Error contains a recorded job failure, when present.
		Error string
	}
)

func (l *workflowProcessLog) Write(b []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.Write(b)
}
func (l *workflowProcessLog) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.buffer.String()
}

func buildWorkflowExamples(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	binaries := make([]string, 0, 2)
	for _, name := range []string{"producer", "worker"} {
		binary := filepath.Join(dir, name)
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		output, err := testprocess.CommandContext(ctx, "go", "build", "-o", binary, "./examples/durable/"+name).CombinedOutput()
		cancel()
		require.NoError(t, err, "%s", output)
		binaries = append(binaries, binary)
	}
	return binaries[0], binaries[1]
}

func startWorkflowProcess(t *testing.T, binary, database string) *workflowProcess {
	t.Helper()
	p := &workflowProcess{command: testprocess.CommandContext(context.Background(), binary, "-database", database, "-lease", "2s", "-poll", "20ms"), done: make(chan struct{}), logs: &workflowProcessLog{}}
	output, err := p.command.StdoutPipe()
	require.NoError(t, err)
	p.command.Stderr = p.logs
	require.NoError(t, p.command.Start())
	go func() {
		p.err = p.command.Wait()
		close(p.done)
	}()
	t.Cleanup(func() {
		select {
		case <-p.done:
			return
		default:
		}
		err := p.command.Process.Kill()
		if err != nil && !errors.Is(err, os.ErrProcessDone) {
			t.Errorf("kill worker: %v", err)
		}
		select {
		case <-p.done:
		case <-time.After(10 * time.Second):
			t.Error("worker failed to exit")
		}
	})
	ready := make(chan error, 1)
	go func() {
		scanner := bufio.NewScanner(output)
		for scanner.Scan() {
			if scanner.Text() == "ready" {
				ready <- nil
				return
			}
		}
		if err := scanner.Err(); err != nil {
			ready <- fmt.Errorf("read worker readiness: %w", err)
			return
		}
		ready <- errors.New("worker exited before readiness")
	}()
	select {
	case err := <-ready:
		require.NoError(t, err, "worker never became ready: %s", p.logs.String())
	case <-time.After(15 * time.Second):
		t.Fatalf("worker readiness timeout: %s", p.logs.String())
	}
	return p
}

func runWorkflowProducer(t *testing.T, binary, database string, args ...string) []byte {
	t.Helper()
	output, err := workflowProducerOutput(binary, database, args...)
	require.NoError(t, err)
	return output
}

func workflowProducerOutput(binary, database string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	command := testprocess.CommandContext(ctx, binary, append([]string{"-database", database}, args...)...)
	var stdout, stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err := command.Run()
	if err != nil {
		return nil, fmt.Errorf("producer: %w: %s", err, stderr.String())
	}
	return stdout.Bytes(), nil
}

func workflowTwoProcess(t *testing.T, database string) {
	t.Helper()
	producer, worker := buildWorkflowExamples(t)
	var calls atomic.Int32
	entered := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if calls.Add(1) == 1 {
			close(entered)
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Length", "42")
		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)
	id := t.Name()
	var admitted adapter.Admission
	require.NoError(t, json.Unmarshal(runWorkflowProducer(t, producer, database, "-action", "submit", "-id", id, "-url", server.URL), &admitted))
	require.True(t, admitted.Created)
	var pending workflowJobView
	require.NoError(t, json.Unmarshal(runWorkflowProducer(t, producer, database, "-action", "lookup", "-id", id), &pending))
	require.Equal(t, sv.StatusRunning, pending.State)
	require.Zero(t, calls.Load(), "producer must not execute the activity")
	first := startWorkflowProcess(t, worker, database)
	select {
	case <-entered:
	case <-time.After(15 * time.Second):
		t.Fatalf("activity did not reach remote object: %s", first.logs.String())
	}
	require.NoError(t, first.command.Process.Kill())
	<-first.done
	require.Error(t, first.err)
	second := startWorkflowProcess(t, worker, database)
	require.EventuallyWithT(t, func(check *assert.CollectT) {
		var job workflowJobView
		output, err := workflowProducerOutput(producer, database, "-action", "lookup", "-id", id)
		require.NoError(check, err)
		require.NoError(check, json.Unmarshal(output, &job))
		require.Equal(check, sv.StatusCompleted, job.State, "worker: %s", second.logs.String())
		require.Equal(check, contract.Metadata{Length: 42, ContentType: "application/octet-stream"}, job.Output)
		require.Empty(check, job.Error)
	}, 20*time.Second, 100*time.Millisecond)
	var again adapter.Admission
	require.NoError(t, json.Unmarshal(runWorkflowProducer(t, producer, database, "-action", "submit", "-id", id, "-url", server.URL), &again))
	require.Equal(t, adapter.Admission{ID: id}, again)
	require.GreaterOrEqual(t, calls.Load(), int32(2), "interrupted activity must resume")
}

func TestWorkflowTwoProcess(t *testing.T) {
	workflowTwoProcess(t, filepath.Join(t.TempDir(), "workflows.db"))
}

func TestWorkflowSingleProcess(t *testing.T) {
	w, err := adapter.NewWorker(sv.WithDatabase(filepath.Join(t.TempDir(), "single.db")), sv.WithMaxPollInterval(10*time.Millisecond))
	require.NoError(t, err)
	wf := execution.Workflow[string, string]{Name: t.Name()}
	activity := execution.Activity[string, string]{Name: t.Name() + ".read"}
	var calls atomic.Int32
	require.NoError(t, adapter.Register(w.Activities(), activity, func(_ context.Context, in string) (string, error) {
		calls.Add(1)
		return in, nil
	}, sv.ActivityOptions{}))
	require.NoError(t, adapter.RegisterWorkflow(w, wf, func(ctx *sv.WorkflowContext, in string) (string, error) {
		return activity.Execute(ctx.Context(), w.Activities().Scope(ctx), in, execution.ActivityOptions{})
	}))
	require.NoError(t, w.Start(context.Background()))
	t.Cleanup(func() {
		require.NoError(t, w.Shutdown(context.Background()))
	})
	admitted, err := adapter.Submit(context.Background(), w, wf, "single", "typed")
	require.NoError(t, err)
	require.True(t, admitted.Created)
	require.EventuallyWithT(t, func(check *assert.CollectT) {
		job, err := adapter.Lookup(context.Background(), w, wf, "single")
		require.NoError(check, err)
		require.Equal(check, sv.StatusCompleted, job.State)
		require.Equal(check, "typed", job.Output)
	}, 10*time.Second, 10*time.Millisecond)
	require.EqualValues(t, 1, calls.Load())
	again, err := adapter.Submit(context.Background(), w, wf, "single", "typed")
	require.NoError(t, err)
	require.False(t, again.Created)
}
