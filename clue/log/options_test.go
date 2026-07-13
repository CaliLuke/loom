package log

import (
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/trace"
)

func TestWithDebugToggles(t *testing.T) {
	cases := []struct {
		name string
		opts []LogOption
		want bool
	}{
		{"default is off", nil, false},
		{"with debug", []LogOption{WithDebug()}, true},
		{"debug then no debug", []LogOption{WithDebug(), WithNoDebug()}, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := Context(context.Background(), tc.opts...)
			require.Equal(t, tc.want, DebugEnabled(ctx))
		})
	}
}

func TestDebugEnabledWithoutLogger(t *testing.T) {
	require.False(t, DebugEnabled(context.Background()))
}

func TestWithOutputUpdatesFirstWriter(t *testing.T) {
	var buf bytes.Buffer
	o := &options{outputs: []Output{{Writer: io.Discard, Format: FormatText}}}
	WithOutput(&buf)(o)
	require.Same(t, &buf, o.outputs[0].Writer)
}

func TestWithOutputPanicsWithoutOutputs(t *testing.T) {
	require.PanicsWithValue(t, "log.WithOutput: logger outputs not initialized", func() {
		WithOutput(io.Discard)(&options{})
	})
}

func TestWithFormatUpdatesFirstFormat(t *testing.T) {
	o := &options{outputs: []Output{{Writer: io.Discard, Format: FormatText}}}
	marker := []byte("custom")
	WithFormat(func(*Entry) []byte {
		return marker
	})(o)
	require.Equal(t, marker, o.outputs[0].Format(&Entry{}))
}

func TestWithFormatPanicsWithoutOutputs(t *testing.T) {
	require.PanicsWithValue(t, "log.WithFormat: logger outputs not initialized", func() {
		WithFormat(FormatJSON)(&options{})
	})
}

func TestWithOutputsValidation(t *testing.T) {
	cases := []struct {
		name    string
		outputs []Output
		panics  string
	}{
		{
			name:   "no outputs",
			panics: "log.WithOutputs: at least one output must be provided",
		},
		{
			name:    "nil writer",
			outputs: []Output{{Writer: nil, Format: FormatText}},
			panics:  "log.WithOutputs: output writer is nil",
		},
		{
			name:    "nil format",
			outputs: []Output{{Writer: io.Discard, Format: nil}},
			panics:  "log.WithOutputs: output format is nil",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.PanicsWithValue(t, tc.panics, func() {
				WithOutputs(tc.outputs...)(&options{})
			})
		})
	}
}

func TestWithOutputsFanout(t *testing.T) {
	var text, jsons bytes.Buffer
	ctx := Context(context.Background(), WithOutputs(
		Output{Writer: &text, Format: FormatText},
		Output{Writer: &jsons, Format: FormatJSON},
	))
	Printf(ctx, "hello")

	require.Contains(t, text.String(), "msg=hello")
	require.Contains(t, jsons.String(), `"msg":"hello"`)
}

func TestWithMaxSize(t *testing.T) {
	o := &options{}
	WithMaxSize(7)(o)
	require.Equal(t, 7, o.maxsize)
}

func TestWithFileLocation(t *testing.T) {
	cases := []struct {
		name string
		log  func(context.Context)
	}{
		{name: "printf", log: func(ctx context.Context) { Printf(ctx, "located") }},
		{name: "print", log: func(ctx context.Context) { Print(ctx, KV{K: "msg", V: "located"}) }},
		{name: "info", log: func(ctx context.Context) { Info(ctx, KV{K: "msg", V: "located"}) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			ctx := newTestContext(&buf, WithFileLocation(), WithDisableBuffering(func(context.Context) bool { return true }))
			tc.log(ctx)
			require.Contains(t, buf.String(), "file=log/options_test.go:")
		})
	}
}

func TestWithFunc(t *testing.T) {
	var buf bytes.Buffer
	ctx := newTestContext(&buf, WithFunc(func(context.Context) []KV {
		return []KV{{K: "injected", V: "yes"}}
	}))
	Printf(ctx, "hi")

	require.Contains(t, buf.String(), "injected=yes")
}

func TestWithDisableBuffering(t *testing.T) {
	cases := []struct {
		name     string
		disable  bool
		buffered bool
	}{
		{"buffering disabled writes immediately", true, false},
		{"buffering enabled defers writes", false, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			disable := tc.disable
			ctx := newTestContext(&buf, WithDisableBuffering(func(context.Context) bool {
				return disable
			}))
			Infof(ctx, "entry")
			if tc.buffered {
				require.Empty(t, buf.String())
				FlushAndDisableBuffering(ctx)
			}
			require.Contains(t, buf.String(), "msg=entry")
		})
	}
}

func TestIsTracing(t *testing.T) {
	require.False(t, IsTracing(context.Background()))

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{1},
		SpanID:  trace.SpanID{1},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	require.True(t, IsTracing(ctx))
}

func TestSpan(t *testing.T) {
	require.Empty(t, Span(context.Background()))

	sc := trace.NewSpanContext(trace.SpanContextConfig{
		TraceID: trace.TraceID{0xa, 0xb},
		SpanID:  trace.SpanID{0xc, 0xd},
	})
	ctx := trace.ContextWithSpanContext(context.Background(), sc)
	kvs := Span(ctx)
	require.Len(t, kvs, 2)
	assert.Equal(t, KV{K: TraceIDKey, V: sc.TraceID().String()}, kvs[0])
	assert.Equal(t, KV{K: SpanIDKey, V: sc.SpanID().String()}, kvs[1])
}

func TestIsTerminal(t *testing.T) {
	old := isTerminal
	t.Cleanup(func() {
		isTerminal = old
	})

	isTerminal = func(int) bool {
		return true
	}
	require.True(t, IsTerminal())

	isTerminal = func(int) bool {
		return false
	}
	require.False(t, IsTerminal())
}

func TestDefaultOptions(t *testing.T) {
	old := isTerminal
	t.Cleanup(func() {
		isTerminal = old
	})

	cases := []struct {
		name       string
		isTerminal bool
		wantFormat FormatFunc
	}{
		{"non terminal uses text format", false, FormatText},
		{"terminal uses terminal format", true, FormatTerminal},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			terminal := tc.isTerminal
			isTerminal = func(int) bool {
				return terminal
			}

			o := defaultOptions()
			require.Len(t, o.outputs, 1)
			assert.Equal(t, DefaultMaxSize, o.maxsize)
			assert.NotNil(t, o.disableBuffering)
			assert.NotNil(t, o.outputs[0].Writer)
			wantPtr := reflect.ValueOf(tc.wantFormat).Pointer()
			assert.Equal(t, wantPtr, reflect.ValueOf(o.outputs[0].Format).Pointer())
		})
	}
}

func TestWithContext(t *testing.T) {
	var buf bytes.Buffer
	logCtx := newTestContext(&buf)
	parent := context.WithValue(context.Background(), struct{ k string }{"k"}, "v")

	ctx := WithContext(parent, logCtx)
	Printf(ctx, "injected")
	require.Contains(t, buf.String(), "msg=injected")

	// Without a logger in logCtx the parent is returned unchanged.
	require.Equal(t, parent, WithContext(parent, context.Background()))
}

func TestMustContainLogger(t *testing.T) {
	require.Panics(t, func() {
		MustContainLogger(context.Background())
	})
	require.NotPanics(t, func() {
		MustContainLogger(Context(context.Background()))
	})
}
