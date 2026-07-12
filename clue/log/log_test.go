package log

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestContext returns a log context that writes text-formatted entries to
// buf. Extra options are applied after the output option.
func newTestContext(buf *bytes.Buffer, opts ...LogOption) context.Context {
	options := make([]LogOption, 0, len(opts)+1)
	options = append(options, WithOutputs(Output{Writer: buf, Format: FormatText}))
	options = append(options, opts...)
	return Context(context.Background(), options...)
}

// stubShortID replaces the request ID generator with one returning id and
// restores the original when the test completes.
func stubShortID(t *testing.T, id string) {
	t.Helper()
	old := shortID
	shortID = func() string {
		return id
	}
	t.Cleanup(func() {
		shortID = old
	})
}

// stubTimeSince makes durations deterministic and restores the original when
// the test completes.
func stubTimeSince(t *testing.T, d time.Duration) {
	t.Helper()
	old := timeSince
	timeSince = func(time.Time) time.Duration {
		return d
	}
	t.Cleanup(func() {
		timeSince = old
	})
}

// stubOSExit captures calls to os.Exit and restores the original when the
// test completes. It returns a pointer to the last recorded exit code, -1 if
// no exit was recorded.
func stubOSExit(t *testing.T) *int {
	t.Helper()
	code := -1
	old := osExit
	osExit = func(c int) {
		code = c
	}
	t.Cleanup(func() {
		osExit = old
	})
	return &code
}

func TestLogTruncatesEntryKeyvals(t *testing.T) {
	var entries []*Entry
	maxsize := len(truncationSuffix)
	ctx := Context(context.Background(),
		WithMaxSize(maxsize),
		WithOutputs(Output{
			Writer: io.Discard,
			Format: func(e *Entry) []byte {
				copied := *e
				copied.KeyVals = append(kvList(nil), e.KeyVals...)
				entries = append(entries, &copied)
				return nil
			},
		}),
	)

	keyvals := make([]Fielder, 0, maxsize+2)
	for i := range maxsize + 2 {
		keyvals = append(keyvals, KV{K: "key", V: i})
	}
	Print(ctx, keyvals...)

	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if got, want := len(entries[0].KeyVals), maxsize+1; got != want {
		t.Fatalf("expected truncated keyvals length %d, got %d: %#v", want, got, entries[0].KeyVals)
	}
	if got, want := entries[0].KeyVals[maxsize], (KV{K: "log", V: truncationSuffix}); got != want {
		t.Fatalf("expected truncation marker %#v, got %#v", want, got)
	}
}

func TestSeverityStringCodeColor(t *testing.T) {
	cases := []struct {
		name  string
		sev   Severity
		str   string
		code  string
		color string
	}{
		{"debug", SeverityDebug, "debug", "DEBG", ColorSeverityDebug},
		{"info", SeverityInfo, "info", "INFO", ColorSeverityInfo},
		{"warn", SeverityWarn, "warn", "WARN", ColorSeverityWarn},
		{"error", SeverityError, "error", "ERRO", ColorSeverityError},
		{"invalid", Severity(42), "<INVALID>", "<INVALID>", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.str, tc.sev.String())
			assert.Equal(t, tc.code, tc.sev.Code())
			assert.Equal(t, tc.color, tc.sev.Color())
		})
	}
}

func TestSeverityFunctions(t *testing.T) {
	cases := []struct {
		name     string
		debug    bool
		log      func(ctx context.Context)
		contains []string
		empty    bool
	}{
		{
			name: "printf writes immediately",
			log: func(ctx context.Context) {
				Printf(ctx, "hello %s", "world")
			},
			contains: []string{`msg="hello world"`, "level=info"},
		},
		{
			name: "infof is buffered",
			log: func(ctx context.Context) {
				Infof(ctx, "buffered")
			},
			empty: true,
		},
		{
			name: "warnf is buffered",
			log: func(ctx context.Context) {
				Warnf(ctx, "warned")
			},
			empty: true,
		},
		{
			name: "errorf flushes buffer and logs error",
			log: func(ctx context.Context) {
				Infof(ctx, "buffered")
				Errorf(ctx, errors.New("boom"), "failed %d", 42)
			},
			contains: []string{"msg=buffered", "err=boom", `msg="failed 42"`, "level=error"},
		},
		{
			name: "error with nil error omits err key",
			log: func(ctx context.Context) {
				Error(ctx, nil, KV{K: "k", V: "v"})
			},
			contains: []string{"level=error k=v"},
		},
		{
			name: "warnf after flush is written",
			log: func(ctx context.Context) {
				FlushAndDisableBuffering(ctx)
				Warnf(ctx, "warned")
			},
			contains: []string{"level=warn", "msg=warned"},
		},
		{
			name: "debugf dropped without debug mode",
			log: func(ctx context.Context) {
				Debugf(ctx, "hidden")
				FlushAndDisableBuffering(ctx)
			},
			empty: true,
		},
		{
			name:  "debugf written in debug mode",
			debug: true,
			log: func(ctx context.Context) {
				Debugf(ctx, "visible")
			},
			contains: []string{"level=debug", "msg=visible"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			opts := []LogOption{}
			if tc.debug {
				opts = append(opts, WithDebug())
			}
			ctx := newTestContext(&buf, opts...)
			tc.log(ctx)
			if tc.empty {
				assert.Empty(t, buf.String())
				return
			}
			for _, want := range tc.contains {
				assert.Contains(t, buf.String(), want)
			}
		})
	}
}

func TestLogNoopWithoutLogger(t *testing.T) {
	// Must not panic when the context has no logger.
	ctx := context.Background()
	Print(ctx, KV{K: "k", V: "v"})
	Info(ctx, KV{K: "k", V: "v"})
	Error(ctx, errors.New("boom"))
	FlushAndDisableBuffering(ctx)
}

func TestWithAddsKeyValues(t *testing.T) {
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	ctx = With(ctx, KV{K: "request", V: "abc"})
	Printf(ctx, "hi")

	require.Contains(t, buf.String(), "request=abc")
	require.Contains(t, buf.String(), "msg=hi")
}

func TestWithoutLoggerReturnsSameContext(t *testing.T) {
	ctx := context.Background()
	got := With(ctx, KV{K: "k", V: "v"})
	require.Equal(t, ctx, got)
}

func TestWithChildDoesNotAffectParent(t *testing.T) {
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	_ = With(ctx, KV{K: "child", V: "only"})
	Printf(ctx, "parent")

	require.Contains(t, buf.String(), "msg=parent")
	require.NotContains(t, buf.String(), "child=only")
}

func TestFatal(t *testing.T) {
	code := stubOSExit(t)
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	Fatal(ctx, errors.New("fatal error"), KV{K: "k", V: "v"})

	require.Equal(t, 1, *code)
	require.Contains(t, buf.String(), "err=\"fatal error\"")
}

func TestFatalf(t *testing.T) {
	code := stubOSExit(t)
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	Fatalf(ctx, errors.New("boom"), "reason %d", 7)

	require.Equal(t, 1, *code)
	require.Contains(t, buf.String(), "err=boom")
	require.Contains(t, buf.String(), `msg="reason 7"`)
}

func TestTruncateLongValues(t *testing.T) {
	cases := []struct {
		name      string
		value     any
		truncated bool
	}{
		{"long string", strings.Repeat("x", 20), true},
		{"short string", "short", false},
		{"int untouched", 123456789012345, false},
		{"uint untouched", uint8(5), false},
		{"bool untouched", true, false},
		{"float untouched", 3.14159, false},
		{"nil untouched", nil, false},
		{"long slice", []any{"aaaa", "bbbb", "cccc", "dddd"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kvs := truncate([]KV{{K: "k", V: tc.value}}, 10)
			require.Len(t, kvs, 1)
			if tc.truncated {
				s, ok := kvs[0].V.(string)
				require.True(t, ok, "truncated value should be a string")
				assert.True(t, strings.HasSuffix(s, truncationSuffix), "got %q", s)
			} else {
				assert.Equal(t, tc.value, kvs[0].V)
			}
		})
	}
}

func TestFieldsFielder(t *testing.T) {
	var buf bytes.Buffer
	ctx := newTestContext(&buf)
	Print(ctx, Fields{"single": "value"})

	require.Contains(t, buf.String(), "single=value")
}

func TestKVLogFields(t *testing.T) {
	kv := KV{K: "k", V: "v"}
	require.Equal(t, []KV{kv}, kv.LogFields())

	kvs := kvList{{K: "a", V: 1}, {K: "b", V: 2}}
	require.Equal(t, []KV(kvs), kvs.LogFields())
}

func TestWithFlushesWhenBufferingDisabled(t *testing.T) {
	type markerKey struct{}
	var buf bytes.Buffer
	ctx := newTestContext(&buf, WithDisableBuffering(func(ctx context.Context) bool {
		return ctx.Value(markerKey{}) != nil
	}))
	Infof(ctx, "buffered")
	require.Empty(t, buf.String())

	// With flushes the buffered entries when the disable-buffering function
	// returns true for the new context.
	ctx = With(context.WithValue(ctx, markerKey{}, true), KV{K: "k", V: "v"})
	require.Contains(t, buf.String(), "msg=buffered")

	Infof(ctx, "direct")
	require.Contains(t, buf.String(), "msg=direct")
	require.Contains(t, buf.String(), "k=v")
}
