package log

import (
	"bytes"
	"context"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	lineageOutput struct {
		mu     sync.Mutex
		buffer bytes.Buffer
	}
	lineageSerialOutput struct {
		active  atomic.Int64
		overlap atomic.Bool
		writes  atomic.Int64
	}
)

func TestBufferLineageFlushesWholeScopeOnce(t *testing.T) {
	for _, first := range []string{"parent", "child", "sibling"} {
		t.Run(first, func(t *testing.T) {
			var output bytes.Buffer
			parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
			Info(parent, KV{K: "msg", V: "ancestor"})
			child := With(parent, KV{K: "child", V: "one"})
			sibling := With(parent, KV{K: "child", V: "two"})
			Info(child, KV{K: "msg", V: "descendant"})
			Info(sibling, KV{K: "msg", V: "sibling"})
			contexts := map[string]context.Context{"parent": parent, "child": child, "sibling": sibling}
			FlushAndDisableBuffering(contexts[first])
			require.Equal(t, 1, strings.Count(output.String(), "msg=ancestor"))
			require.Equal(t, 1, strings.Count(output.String(), "msg=descendant"))
			require.Equal(t, 1, strings.Count(output.String(), "msg=sibling"))
			lines := strings.Split(strings.TrimSpace(output.String()), "\n")
			require.Len(t, lines, 3)
			require.NotContains(t, lines[0], "child=")
			require.Contains(t, lines[1], "child=one")
			require.Contains(t, lines[2], "child=two")
			for _, ctx := range []context.Context{parent, child, sibling} {
				FlushAndDisableBuffering(ctx)
			}
			require.Len(t, strings.Split(strings.TrimSpace(output.String()), "\n"), 3)
		})
	}
}

func TestBufferLineageNoDuplicateAncestor(t *testing.T) {
	var output bytes.Buffer
	parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	Info(parent, KV{K: "msg", V: "ancestor"})
	child := With(parent)
	Error(child, nil, KV{K: "msg", V: "child error"})
	Error(parent, nil, KV{K: "msg", V: "parent error"})
	require.Equal(t, 1, strings.Count(output.String(), "msg=ancestor"))
}

func TestBufferLineageAfterConcurrentFlush(t *testing.T) {
	var output lineageOutput
	parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	const children = 20
	var ready, done sync.WaitGroup
	ready.Add(children)
	release := make(chan struct{})
	for index := range children {
		child := With(parent, KV{K: "child", V: index})
		done.Go(func() {
			Info(child, KV{K: "msg", V: fmt.Sprintf("before-%d", index)})
			ready.Done()
			<-release
			Info(child, KV{K: "msg", V: fmt.Sprintf("after-%d", index)})
		})
	}
	ready.Wait()
	FlushAndDisableBuffering(parent)
	close(release)
	done.Wait()
	FlushAndDisableBuffering(parent)
	text := output.String()
	require.Len(t, strings.Split(strings.TrimSpace(text), "\n"), children*2)
	for index := range children {
		require.Equal(t, 1, strings.Count(text, fmt.Sprintf("msg=before-%d\n", index)))
		require.Equal(t, 1, strings.Count(text, fmt.Sprintf("msg=after-%d\n", index)))
	}
}

func TestRequestErrorFlushesEndpointEntries(t *testing.T) {
	for _, protocol := range []string{"HTTP", "unary", "streaming"} {
		t.Run(protocol, func(t *testing.T) {
			var output bytes.Buffer
			root := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
			invokeRequestScope(t, protocol, root, context.Background(), true, false, func(ctx context.Context) {
				endpoint := Endpoint(func(ctx context.Context, _ any) (any, error) {
					Info(ctx, KV{K: "msg", V: "endpoint detail"})
					return nil, nil
				})
				serviceContext := context.WithValue(ctx, loom.ServiceKey, "scope_service")
				serviceContext = context.WithValue(serviceContext, loom.MethodKey, "scope_method")
				_, err := endpoint(serviceContext, nil)
				require.NoError(t, err)
				Error(ctx, nil, KV{K: "msg", V: "request error"})
			})
			require.Contains(t, output.String(), "endpoint detail")
			require.Contains(t, output.String(), LoomServiceKey+"=scope_service")
			require.Contains(t, output.String(), LoomMethodKey+"=scope_method")
			require.Equal(t, 1, strings.Count(output.String(), "endpoint detail"))
		})
	}
}

func TestBufferLineageRetainsEntryOutputs(t *testing.T) {
	for _, reconfigure := range []string{"child", "same context"} {
		t.Run(reconfigure, func(t *testing.T) {
			var original, changed bytes.Buffer
			parent := Context(context.Background(), WithOutputs(Output{Writer: &original, Format: FormatText}))
			Info(parent, KV{K: "msg", V: "before"})
			child := parent
			if reconfigure == "child" {
				child = With(parent)
			}
			child = Context(child, WithOutputs(Output{Writer: &changed, Format: FormatJSON}))
			Info(child, KV{K: "msg", V: "after"})
			FlushAndDisableBuffering(child)
			FlushAndDisableBuffering(parent)
			require.Contains(t, original.String(), "msg=before")
			require.NotContains(t, original.String(), "after")
			require.Contains(t, changed.String(), `"msg":"after"`)
			require.NotContains(t, changed.String(), "before")
			require.Equal(t, 1, strings.Count(original.String(), "\n"))
			require.Equal(t, 1, strings.Count(changed.String(), "\n"))
		})
	}
}

func TestBufferLineageFlushTriggers(t *testing.T) {
	for _, trigger := range []string{"debug", "option", "derivation callback"} {
		t.Run(trigger, func(t *testing.T) {
			var output bytes.Buffer
			parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
			child := With(parent)
			Info(child, KV{K: "msg", V: "before"})
			switch trigger {
			case "debug":
				parent = Context(parent, WithDebug())
				Debug(parent, KV{K: "msg", V: "debug trigger"})
			case "option":
				Context(parent, WithDisableBuffering(func(context.Context) bool {
					return true
				}))
			case "derivation callback":
				var active bool
				parent = Context(parent, WithDisableBuffering(func(context.Context) bool {
					return active
				}))
				active = true
				With(parent)
			}
			Info(child, KV{K: "msg", V: "after"})
			require.Equal(t, 1, strings.Count(output.String(), "msg=before"))
			require.Equal(t, 1, strings.Count(output.String(), "msg=after"))
		})
	}
}

func TestBufferLineageSerializesSiblingWrites(t *testing.T) {
	var output lineageSerialOutput
	parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	FlushAndDisableBuffering(parent)
	var done sync.WaitGroup
	for range 100 {
		child := With(parent)
		done.Go(func() {
			for range 10 {
				Info(child, KV{K: "msg", V: "concurrent"})
			}
		})
	}
	done.Wait()
	require.False(t, output.overlap.Load(), "shared buffer must serialize sibling output callbacks")
	require.Equal(t, int64(1000), output.writes.Load())
}

// Write appends output under a lock for concurrent logger controls.
func (w *lineageOutput) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.Write(p)
}

// String returns a synchronized output snapshot.
func (w *lineageOutput) String() string {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.buffer.String()
}

// Write detects overlapping calls without serializing them for the logger.
func (w *lineageSerialOutput) Write(p []byte) (int, error) {
	if w.active.Add(1) > 1 {
		w.overlap.Store(true)
	}
	runtime.Gosched()
	w.writes.Add(1)
	w.active.Add(-1)
	return len(p), nil
}
