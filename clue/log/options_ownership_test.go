package log

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDerivedLoggerOptionsAreIsolated(t *testing.T) {
	for _, tc := range []struct {
		name   string
		change LogOption
	}{
		{"debug", WithDebug()},
		{"size", WithMaxSize(7)},
		{"writer", WithOutput(io.Discard)},
		{"format", WithFormat(FormatJSON)},
		{"callback", WithFunc(func(context.Context) []KV {
			return []KV{{K: "derived", V: true}}
		})},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var output bytes.Buffer
			parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
			child := With(parent, KV{K: "child", V: true})
			sibling := With(parent, KV{K: "sibling", V: true})
			configured := Context(child, tc.change)
			require.Same(t, child.Value(ctxLogger), configured.Value(ctxLogger))
			require.False(t, DebugEnabled(parent), "child configuration must not change parent")
			require.False(t, DebugEnabled(sibling), "child configuration must not change sibling")
			Print(parent, KV{K: "message", V: "long unchanged parent message"})
			require.Contains(t, output.String(), "message=\"long unchanged parent message\"")
			require.NotContains(t, output.String(), "derived=true")
		})
	}
}

func TestReusableOutputOptionOwnsItsSlice(t *testing.T) {
	var original, replacement bytes.Buffer
	outputs := []Output{{Writer: &original, Format: FormatText}}
	option := WithOutputs(outputs...)
	first := Context(context.Background(), option)
	second := Context(context.Background(), option)
	Context(first, WithOutput(&replacement))
	Print(second, KV{K: "msg", V: "second"})
	require.Contains(t, original.String(), "msg=second", "reapplying one option must not share mutable output slots")
	require.Empty(t, replacement.String())
}

func TestOutputOptionDetachesAppliedCallerSlice(t *testing.T) {
	var original, replacement bytes.Buffer
	outputs := []Output{{Writer: &original, Format: FormatText}}
	ctx := Context(context.Background(), WithOutputs(outputs...))
	outputs[0].Writer = &replacement
	Print(ctx, KV{K: "msg", V: "after caller mutation"})
	require.Contains(t, original.String(), "after caller mutation", "caller slice is not live configuration")
	require.Empty(t, replacement.String())
}

func TestDerivedLoggerOptionsConcurrent(t *testing.T) {
	root := Context(context.Background(), WithOutputs(Output{Writer: io.Discard, Format: FormatText}))
	var wg sync.WaitGroup
	for range 4 {
		child := With(root)
		wg.Go(func() {
			for range 100 {
				child = Context(child, WithDebug())
				Debug(child, KV{K: "msg", V: "enabled"})
				child = Context(child, WithNoDebug())
				Info(child, KV{K: "msg", V: "disabled"})
			}
		})
	}
	wg.Wait()
	require.False(t, DebugEnabled(root))
}

func TestContextOptionUpdatePreservesBuffer(t *testing.T) {
	var output bytes.Buffer
	ctx := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	ctx = With(ctx, KV{K: "scope", V: "retained"})
	Info(ctx, KV{K: "msg", V: "buffered"})
	require.Empty(t, output.String())
	configured := Context(ctx, WithMaxSize(2048))
	require.Same(t, ctx.Value(ctxLogger), configured.Value(ctxLogger))
	Error(configured, nil, KV{K: "msg", V: "flush"})
	FlushAndDisableBuffering(ctx)
	require.Equal(t, 1, strings.Count(output.String(), "msg=buffered"))
	require.Equal(t, 1, strings.Count(output.String(), "msg=flush"))
	require.Equal(t, 2, strings.Count(output.String(), "scope=retained"))
}

func TestDerivedCallbackSlicesStayIsolated(t *testing.T) {
	var output bytes.Buffer
	root := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	for _, name := range []string{"one", "two", "three"} {
		root = Context(root, WithFunc(func(context.Context) []KV {
			return []KV{{K: name, V: true}}
		}))
	}
	first, second := With(root), With(root)
	first = Context(first, WithFunc(func(context.Context) []KV {
		return []KV{{K: "first", V: true}}
	}))
	second = Context(second, WithFunc(func(context.Context) []KV {
		return []KV{{K: "second", V: true}}
	}))
	Print(first, KV{K: "msg", V: "first entry"})
	require.Contains(t, output.String(), "first=true")
	require.NotContains(t, output.String(), "second=true")
	output.Reset()
	Print(second, KV{K: "msg", V: "second entry"})
	require.Contains(t, output.String(), "second=true")
	require.NotContains(t, output.String(), "first=true")
}

func TestReusableOutputOptionConcurrent(t *testing.T) {
	option := WithOutputs(Output{Writer: io.Discard, Format: FormatText})
	var wg sync.WaitGroup
	for range 4 {
		wg.Go(func() {
			for range 100 {
				ctx := Context(context.Background(), option)
				Print(ctx, KV{K: "msg", V: "independent"})
			}
		})
	}
	wg.Wait()
}

func TestDerivedBufferingOptionIsIsolated(t *testing.T) {
	var output bytes.Buffer
	parent := Context(context.Background(), WithOutputs(Output{Writer: &output, Format: FormatText}))
	child := With(parent)
	Context(child, WithDisableBuffering(func(context.Context) bool {
		return true
	}))
	Context(parent)
	Info(parent, KV{K: "msg", V: "buffered"})
	require.Empty(t, output.String())
	FlushAndDisableBuffering(parent)
	require.Contains(t, output.String(), "msg=buffered")
}
