package slogbridge

import (
	"context"
	"errors"
	"log/slog"
	"math"
	"strings"
	"sync"
	"testing"
	"testing/slogtest"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
	"go.opentelemetry.io/otel/log/noop"
	sdklog "go.opentelemetry.io/otel/sdk/log"

	"github.com/CaliLuke/loom/observability/otel/internal/testkit"

	"github.com/stretchr/testify/require"
)

type (
	testValuer struct {
		val slog.Value
	}

	typedNilError struct {
		msg string
	}
)

func (v testValuer) LogValue() slog.Value {
	return v.val
}

func (e *typedNilError) Error() string {
	return e.msg
}

func TestSlogtestConformance(t *testing.T) {
	var harness *testkit.LogHarness
	slogtest.Run(t, func(t *testing.T) slog.Handler {
		t.Helper()
		harness = testkit.NewLogHarness(t)
		return NewHandler(Config{
			ScopeName:      "slogtest",
			LoggerProvider: harness.Provider,
		})
	}, func(t *testing.T) map[string]any {
		t.Helper()
		records := harness.Exporter.Records()
		require.Len(t, records, 1)
		return recordToSlogtestMap(records[0])
	})
}

func TestTraceCorrelationAndScopeMetadata(t *testing.T) {
	logHarness := testkit.NewLogHarness(t)
	traceHarness := testkit.NewTraceHarness(t)

	logger := slog.New(NewHandler(Config{
		ScopeName:      "autok.server",
		ScopeVersion:   "v1.2.3",
		SchemaURL:      "https://opentelemetry.io/schemas/1.43.0",
		LoggerProvider: logHarness.Provider,
	}))

	ctx, span := traceHarness.Provider.Tracer("test").Start(t.Context(), "bridge")
	spanCtx := span.SpanContext()
	logger.InfoContext(ctx, "hello", slog.String("project_id", "proj_123"))
	span.End()

	records := logHarness.Exporter.Records()
	require.Len(t, records, 1)
	rec := records[0]
	require.Equal(t, "hello", rec.Body().AsString())
	require.Equal(t, spanCtx.TraceID(), rec.TraceID())
	require.Equal(t, spanCtx.SpanID(), rec.SpanID())
	require.Equal(t, "autok.server", rec.InstrumentationScope().Name)
	require.Equal(t, "v1.2.3", rec.InstrumentationScope().Version)
	require.Equal(t, "https://opentelemetry.io/schemas/1.43.0", rec.InstrumentationScope().SchemaURL)
	require.Equal(t, log.SeverityInfo, rec.Severity())
	require.Equal(t, "INFO", rec.SeverityText())
	requireAttr(t, rec, attribute.String("project_id", "proj_123"))
}

func TestSeverityMappingAndLevelFiltering(t *testing.T) {
	cases := []struct {
		name     string
		level    slog.Level
		expected log.Severity
		text     string
	}{
		{name: "min int clamp", level: slog.Level(math.MinInt), expected: log.SeverityTrace1, text: slog.Level(math.MinInt).String()},
		{name: "below trace clamp", level: slog.Level(-100), expected: log.SeverityTrace1, text: "DEBUG-96"},
		{name: "trace", level: slog.LevelDebug - 4, expected: log.SeverityTrace, text: "DEBUG-4"},
		{name: "debug", level: slog.LevelDebug, expected: log.SeverityDebug, text: "DEBUG"},
		{name: "debug+2", level: slog.LevelDebug + 2, expected: log.SeverityDebug3, text: "DEBUG+2"},
		{name: "info", level: slog.LevelInfo, expected: log.SeverityInfo, text: "INFO"},
		{name: "warn", level: slog.LevelWarn, expected: log.SeverityWarn, text: "WARN"},
		{name: "error", level: slog.LevelError, expected: log.SeverityError, text: "ERROR"},
		{name: "fatal", level: slog.LevelError + 4, expected: log.SeverityFatal, text: "ERROR+4"},
		{name: "above fatal clamp", level: slog.Level(100), expected: log.SeverityFatal4, text: "ERROR+92"},
		{name: "max int clamp", level: slog.Level(math.MaxInt), expected: log.SeverityFatal4, text: slog.Level(math.MaxInt).String()},
	}

	logHarness := testkit.NewLogHarness(t)
	h := NewHandler(Config{
		ScopeName:      "severity",
		LoggerProvider: logHarness.Provider,
	})
	logger := slog.New(h)

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, convertLevel(tc.level))
			logger.Log(t.Context(), tc.level, tc.name)
		})
	}

	records := logHarness.Exporter.Records()
	require.Len(t, records, len(cases))
	for i, tc := range cases {
		require.Equal(t, tc.expected, records[i].Severity())
		require.Equal(t, tc.text, records[i].SeverityText())
	}

	// Verify Leveler filtering with dynamic LevelVar.
	var levelVar slog.LevelVar
	levelVar.Set(slog.LevelWarn)
	filteredHarness := testkit.NewLogHarness(t)
	filtered := NewHandler(Config{
		ScopeName:      "filtered",
		LoggerProvider: filteredHarness.Provider,
		Level:          &levelVar,
	})
	require.False(t, filtered.Enabled(t.Context(), slog.LevelInfo))
	require.True(t, filtered.Enabled(t.Context(), slog.LevelWarn))

	levelVar.Set(slog.LevelDebug)
	require.True(t, filtered.Enabled(t.Context(), slog.LevelDebug))

	// Verify Enabled delegates to the underlying OTel logger when Level is nil.
	disabled := NewHandler(Config{
		ScopeName:      "disabled",
		LoggerProvider: noop.NewLoggerProvider(),
	})
	require.False(t, disabled.Enabled(t.Context(), slog.LevelError))
}

func TestGlobalLoggerProviderFallbackAndNilContext(t *testing.T) {
	lpBefore := otel.GetLoggerProvider()
	t.Cleanup(func() {
		otel.SetLoggerProvider(lpBefore)
	})

	harness := testkit.NewLogHarness(t)
	otel.SetLoggerProvider(harness.Provider)

	h := NewHandler(Config{ScopeName: "global"})
	require.True(t, h.Enabled(context.TODO(), slog.LevelInfo))

	rec := slog.NewRecord(time.Now(), slog.LevelInfo, "via global", 0)
	var nilCtx context.Context
	require.True(t, h.Enabled(nilCtx, slog.LevelInfo))
	require.NoError(t, h.Handle(nilCtx, rec))

	records := harness.Exporter.Records()
	require.Len(t, records, 1)
	require.Equal(t, "via global", records[0].Body().AsString())
}

func TestAttributeConversionsAndEdgeCases(t *testing.T) {
	harness := testkit.NewLogHarness(t)
	h := NewHandler(Config{
		ScopeName:      "attrs",
		LoggerProvider: harness.Provider,
		AddSource:      true,
	})
	logger := slog.New(h)

	ts := time.Date(2026, 11, 21, 18, 0, 0, 123456789, time.UTC)
	rawBytes := []byte("secret-bytes")
	strs := []string{"a", "b"}
	var nilErr *typedNilError

	logger.InfoContext(t.Context(), "kinds",
		slog.Bool("b", true),
		slog.Int64("i64", -42),
		slog.Uint64("u64_small", 42),
		slog.Uint64("u64_large", math.MaxUint64),
		slog.Float64("f64", 3.5),
		slog.String("s", "hello"),
		slog.Duration("dur", 250*time.Millisecond),
		slog.Time("ts", ts),
		slog.Any("nil_val", nil),
		slog.Any("err_val", errors.New("boom")),
		slog.Any("typed_nil_err", nilErr),
		slog.Any("bytes", rawBytes),
		slog.Any("strings", strs),
		slog.Any("ints", []int{1, 2}),
		slog.Any("int64s", []int64{3, 4}),
		slog.Any("floats", []float64{1.25, 2.5}),
		slog.Any("bools", []bool{true, false}),
		slog.Any("uint_large", uint(math.MaxUint64)),
		slog.Any("valuer_group", testValuer{
			val: slog.GroupValue(slog.String("nested", "yes")),
		}),
		slog.Group("", slog.String("inlined", "ok")),
		slog.Group("empty_group"),
	)

	// Mutate caller slices after logging to ensure defensive copies.
	rawBytes[0] = 'X'
	strs[0] = "mutated"

	records := harness.Exporter.Records()
	require.Len(t, records, 1)
	attrs := attrsMap(records[0])

	require.Equal(t, attribute.BoolValue(true), attrs["b"])
	require.Equal(t, attribute.Int64Value(-42), attrs["i64"])
	require.Equal(t, attribute.Int64Value(42), attrs["u64_small"])
	require.Equal(t, attribute.StringValue("18446744073709551615"), attrs["u64_large"])
	require.Equal(t, attribute.Float64Value(3.5), attrs["f64"])
	require.Equal(t, attribute.StringValue("hello"), attrs["s"])
	require.Equal(t, attribute.Int64Value(int64(250*time.Millisecond)), attrs["dur"])
	require.Equal(t, attribute.StringValue(ts.Format(time.RFC3339Nano)), attrs["ts"])
	require.Equal(t, attribute.Value{}, attrs["nil_val"])
	require.Equal(t, attribute.StringValue("boom"), attrs["err_val"])
	require.Equal(t, attribute.StringValue("<nil>"), attrs["typed_nil_err"])
	require.Equal(t, attribute.ByteSliceValue([]byte("secret-bytes")), attrs["bytes"])
	require.Equal(t, attribute.StringSliceValue([]string{"a", "b"}), attrs["strings"])
	require.Equal(t, attribute.IntSliceValue([]int{1, 2}), attrs["ints"])
	require.Equal(t, attribute.Int64SliceValue([]int64{3, 4}), attrs["int64s"])
	require.Equal(t, attribute.Float64SliceValue([]float64{1.25, 2.5}), attrs["floats"])
	require.Equal(t, attribute.BoolSliceValue([]bool{true, false}), attrs["bools"])
	require.Equal(t, attribute.StringValue("18446744073709551615"), attrs["uint_large"])
	require.Equal(t, attribute.MapValue(attribute.String("nested", "yes")), attrs["valuer_group"])
	require.Equal(t, attribute.StringValue("ok"), attrs["inlined"])
	require.NotContains(t, attrs, "empty_group")

	// Verify AddSource emitted OpenTelemetry stable code.* attributes.
	require.Contains(t, attrs[codeFilePathKey].AsString(), "bridge_test.go")
	require.Contains(t, attrs[codeFunctionNameKey].AsString(), "TestAttributeConversionsAndEdgeCases")
	require.Positive(t, attrs[codeLineNumberKey].AsInt64())
}

func TestReplaceAttrAndFailClosedFormatError(t *testing.T) {
	harness := testkit.NewLogHarness(t)

	var seenGroups [][]string
	rawSecretErr := errors.New("postgres://admin:supersecret@db:5432")
	dropErr := errors.New("drop-me")

	h := NewHandler(Config{
		ScopeName:      "hooks",
		LoggerProvider: harness.Provider,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			seenGroups = append(seenGroups, append([]string(nil), groups...))
			if a.Key == "secret_token" {
				return slog.Attr{}
			}
			// Normalize camelCase/uppercase keys to lowercase.
			a.Key = strings.ToLower(a.Key)
			if a.Key == "err" {
				a.Key = "error"
			}
			return a
		},
		FormatError: func(err error) slog.Value {
			if errors.Is(err, dropErr) {
				return slog.Value{}
			}
			return slog.GroupValue(
				slog.String("kind", "sanitized"),
				slog.String("message", "internal error"),
			)
		},
	})

	logger := slog.New(h).WithGroup("req").With("RequestID", "req_1")
	logger.ErrorContext(t.Context(), "db failed",
		slog.Any("ERR", rawSecretErr),
		slog.Any("dropped_err", dropErr),
		slog.String("secret_token", "tok_123"),
		slog.Group("only_dropped", slog.String("secret_token", "tok_456")),
	)

	records := harness.Exporter.Records()
	require.Len(t, records, 1)
	attrs := attrsMap(records[0])

	reqMap := kvSliceToMap(attrs["req"].AsMap())
	require.Equal(t, attribute.StringValue("req_1"), reqMap["requestid"])
	require.NotContains(t, reqMap, "secret_token")
	require.NotContains(t, reqMap, "dropped_err")
	require.NotContains(t, reqMap, "only_dropped")

	errMap := kvSliceToMap(reqMap["error"].AsMap())
	require.Equal(t, attribute.StringValue("sanitized"), errMap["kind"])
	require.Equal(t, attribute.StringValue("internal error"), errMap["message"])

	// Fail-closed check: raw secret must not appear anywhere in exported record.
	records[0].WalkAttributes(func(kv attribute.KeyValue) bool {
		require.NotContains(t, kv.Value.String(), "supersecret")
		return true
	})

	require.Contains(t, seenGroups, []string{"req"})
	require.Contains(t, seenGroups, []string{"req", "only_dropped"})
}

func TestFormatErrorAdversarialAndReplaceAttrGroups(t *testing.T) {
	harness := testkit.NewLogHarness(t)
	rawSecret := errors.New("raw-secret-password")
	echoErr := errors.New("echo-raw-secret")
	nestedErr := errors.New("nested-secret-cause")
	var nilErr *typedNilError

	var sourceSeenAtRoot bool
	h := NewHandler(Config{
		ScopeName:      "adversarial",
		LoggerProvider: harness.Provider,
		AddSource:      true,
		ReplaceAttr: func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == codeFilePathKey && len(groups) == 0 {
				sourceSeenAtRoot = true
			}
			if a.Key == "expand_named" {
				return slog.Group("expanded", slog.Any("inner_err", rawSecret), slog.String("tag", "ok"))
			}
			if a.Key == "expand_inline" {
				return slog.Group("", slog.Any("inline_err", rawSecret), slog.String("inline_tag", "ok"))
			}
			return a
		},
		FormatError: func(err error) slog.Value {
			// Call err.Error() directly to verify typed-nil pointers never reach here.
			msg := err.Error()
			if errors.Is(err, echoErr) {
				// Misbehaving formatter returning the raw error directly.
				return slog.AnyValue(err)
			}
			if errors.Is(err, nestedErr) {
				// Misbehaving formatter embedding a raw error inside a returned group.
				return slog.GroupValue(
					slog.String("safe", "kept"),
					slog.Any("leaked_cause", err),
				)
			}
			return slog.StringValue("sanitized:" + strings.Split(msg, "-")[0])
		},
	})

	logger := slog.New(h).WithGroup("svc")
	logger.ErrorContext(t.Context(), "edge cases",
		slog.Any("typed_nil", nilErr),
		slog.Any("echo_err", echoErr),
		slog.Any("nested_err", nestedErr),
		slog.Any("valuer_err", testValuer{val: slog.AnyValue(rawSecret)}),
		slog.Group("grp", slog.Any("grp_err", rawSecret)),
		slog.String("expand_named", "trigger"),
		slog.String("expand_inline", "trigger"),
	)

	records := harness.Exporter.Records()
	require.Len(t, records, 1)
	require.True(t, sourceSeenAtRoot, "expected AddSource code.file.path at root groups")

	attrs := attrsMap(records[0])
	require.Contains(t, attrs[codeFilePathKey].AsString(), "bridge_test.go")

	svcMap := kvSliceToMap(attrs["svc"].AsMap())
	require.Equal(t, attribute.StringValue("<nil>"), svcMap["typed_nil"])
	require.NotContains(t, svcMap, "echo_err")

	nestedMap := kvSliceToMap(svcMap["nested_err"].AsMap())
	require.Equal(t, attribute.StringValue("kept"), nestedMap["safe"])
	require.NotContains(t, nestedMap, "leaked_cause")

	require.Equal(t, attribute.StringValue("sanitized:raw"), svcMap["valuer_err"])

	grpMap := kvSliceToMap(svcMap["grp"].AsMap())
	require.Equal(t, attribute.StringValue("sanitized:raw"), grpMap["grp_err"])

	expandedMap := kvSliceToMap(svcMap["expanded"].AsMap())
	require.Equal(t, attribute.StringValue("sanitized:raw"), expandedMap["inner_err"])
	require.Equal(t, attribute.StringValue("ok"), expandedMap["tag"])

	require.Equal(t, attribute.StringValue("sanitized:raw"), svcMap["inline_err"])
	require.Equal(t, attribute.StringValue("ok"), svcMap["inline_tag"])

	records[0].WalkAttributes(func(kv attribute.KeyValue) bool {
		require.NotContains(t, kv.Value.String(), "secret")
		return true
	})
}

func TestImmutabilityAndConcurrency(t *testing.T) {
	harness := testkit.NewLogHarness(t)
	base := NewHandler(Config{
		ScopeName:      "concurrent",
		LoggerProvider: harness.Provider,
	})

	// Empty WithAttrs / WithGroup must be no-ops.
	require.Same(t, base, base.WithAttrs(nil))
	require.Same(t, base, base.WithGroup(""))

	// Caller mutating the slice passed to WithAttrs must not affect the handler.
	input := []slog.Attr{slog.String("k", "original")}
	derived := base.WithAttrs(input)
	input[0] = slog.String("k", "mutated")

	const workers = 16
	var wg sync.WaitGroup
	wg.Add(workers)
	for i := range workers {
		go func(id int) {
			defer wg.Done()
			l := slog.New(derived.WithGroup("worker").WithAttrs([]slog.Attr{slog.Int("id", id)}))
			l.InfoContext(t.Context(), "tick", slog.Int("seq", id))
		}(i)
	}
	wg.Wait()

	records := harness.Exporter.Records()
	require.Len(t, records, workers)
	for _, rec := range records {
		attrs := attrsMap(rec)
		require.Equal(t, attribute.StringValue("original"), attrs["k"])
		workerMap := kvSliceToMap(attrs["worker"].AsMap())
		require.Contains(t, workerMap, "id")
		require.Contains(t, workerMap, "seq")
		require.Equal(t, workerMap["id"], workerMap["seq"])
	}
}

func requireAttr(t *testing.T, record sdklog.Record, expected attribute.KeyValue) {
	t.Helper()
	attrs := attrsMap(record)
	got, ok := attrs[string(expected.Key)]
	require.True(t, ok, "missing log attribute %q", expected.Key)
	require.Equal(t, expected.Value, got)
}

func attrsMap(record sdklog.Record) map[string]attribute.Value {
	out := make(map[string]attribute.Value, record.AttributesLen())
	record.WalkAttributes(func(kv attribute.KeyValue) bool {
		out[string(kv.Key)] = kv.Value
		return true
	})
	return out
}

func kvSliceToMap(kvs []attribute.KeyValue) map[string]attribute.Value {
	out := make(map[string]attribute.Value, len(kvs))
	for _, kv := range kvs {
		out[string(kv.Key)] = kv.Value
	}
	return out
}

func recordToSlogtestMap(rec sdklog.Record) map[string]any {
	m := map[string]any{
		slog.MessageKey: rec.Body().AsString(),
		slog.LevelKey:   rec.SeverityText(),
	}
	if !rec.Timestamp().IsZero() {
		m[slog.TimeKey] = rec.Timestamp()
	}
	rec.WalkAttributes(func(kv attribute.KeyValue) bool {
		m[string(kv.Key)] = otelValueToAny(kv.Value)
		return true
	})
	return m
}

func otelValueToAny(v attribute.Value) any {
	switch v.Type() {
	case attribute.MAP:
		out := make(map[string]any, len(v.AsMap()))
		for _, kv := range v.AsMap() {
			out[string(kv.Key)] = otelValueToAny(kv.Value)
		}
		return out
	case attribute.SLICE:
		vals := v.AsSlice()
		out := make([]any, len(vals))
		for i, elem := range vals {
			out[i] = otelValueToAny(elem)
		}
		return out
	default:
		return v.AsInterface()
	}
}
