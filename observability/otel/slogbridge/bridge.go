package slogbridge

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/log"
)

type (
	// Config configures a [slog.Handler] that bridges records into the
	// OpenTelemetry Logs API.
	Config struct {
		// ScopeName is the OpenTelemetry instrumentation scope name attached to
		// bridged log records.
		ScopeName string
		// ScopeVersion is the optional instrumentation scope version.
		ScopeVersion string
		// SchemaURL is the optional OpenTelemetry schema URL for the
		// instrumentation scope.
		SchemaURL string
		// LoggerProvider overrides the OpenTelemetry logger provider used by
		// the bridge. When nil, the global OpenTelemetry logger provider is
		// used.
		LoggerProvider log.LoggerProvider
		// Level reports the minimum slog.Level handled by the bridge. When nil,
		// no local minimum level filter is applied and Enabled delegates solely
		// to the OpenTelemetry logger.
		Level slog.Leveler
		// AddSource attaches OpenTelemetry code source attributes
		// (code.file.path, code.function.name, code.line.number) derived from
		// slog.Record.PC when true.
		AddSource bool
		// ReplaceAttr rewrites or drops each resolved non-group attribute
		// before OpenTelemetry conversion. The first argument is the read-only
		// slice of enclosing group names (nil at the root). Returning a zero
		// slog.Attr drops the attribute.
		ReplaceAttr func(groups []string, a slog.Attr) slog.Attr
		// FormatError formats non-nil error values encountered in attributes
		// before OpenTelemetry conversion. Typed-nil pointer errors are
		// emitted as "<nil>" without invoking FormatError. When nil, error
		// values are formatted with fmt.Sprint. Returning a zero slog.Value
		// drops the attribute.
		//
		// When FormatError is set, raw error values are never forwarded to
		// log.Record.SetErr or formatted directly; if FormatError itself
		// returns a value containing an error, that error value is dropped so
		// sanitization fails closed.
		FormatError func(err error) slog.Value
	}

	handler struct {
		logger      log.Logger
		level       slog.Leveler
		addSource   bool
		replaceAttr func(groups []string, a slog.Attr) slog.Attr
		formatError func(err error) slog.Value
		goa         *groupOrAttrs
	}

	groupOrAttrs struct {
		group string
		attrs []slog.Attr
		next  *groupOrAttrs
	}
)

const (
	// slogToOTelSeverityOffset shifts slog.Level values so LevelDebug (-4),
	// LevelInfo (0), LevelWarn (4), and LevelError (8) align with
	// SeverityDebug (5), SeverityInfo (9), SeverityWarn (13), and
	// SeverityError (17) respectively.
	slogToOTelSeverityOffset = int(log.SeverityInfo) - int(slog.LevelInfo)
	minMappedLevel           = slog.Level(int(log.SeverityTrace1) - slogToOTelSeverityOffset)
	maxMappedLevel           = slog.Level(int(log.SeverityFatal4) - slogToOTelSeverityOffset)

	codeFilePathKey     = "code.file.path"
	codeFunctionNameKey = "code.function.name"
	codeLineNumberKey   = "code.line.number"
)

// NewHandler returns a [slog.Handler] that emits records directly to the
// OpenTelemetry Logs API.
func NewHandler(cfg Config) slog.Handler {
	opts := make([]log.LoggerOption, 0, 2)
	if cfg.ScopeVersion != "" {
		opts = append(opts, log.WithInstrumentationVersion(cfg.ScopeVersion))
	}
	if cfg.SchemaURL != "" {
		opts = append(opts, log.WithSchemaURL(cfg.SchemaURL))
	}

	var logger log.Logger
	if cfg.LoggerProvider != nil {
		logger = cfg.LoggerProvider.Logger(cfg.ScopeName, opts...)
	} else {
		logger = otel.Logger(cfg.ScopeName, opts...)
	}

	return &handler{
		logger:      logger,
		level:       cfg.Level,
		addSource:   cfg.AddSource,
		replaceAttr: cfg.ReplaceAttr,
		formatError: cfg.FormatError,
	}
}

func convertLevel(level slog.Level) log.Severity {
	switch {
	case level <= minMappedLevel:
		return log.SeverityTrace1
	case level >= maxMappedLevel:
		return log.SeverityFatal4
	default:
		return log.Severity(int(level) + slogToOTelSeverityOffset)
	}
}

func appendGroup(groups []string, name string) []string {
	out := make([]string, len(groups)+1)
	copy(out, groups)
	out[len(groups)] = name
	return out
}

func convertUint64(u uint64) attribute.Value {
	if u <= math.MaxInt64 {
		return attribute.Int64Value(int64(u))
	}
	return attribute.StringValue(strconv.FormatUint(u, 10))
}

func isTypedNilPointer(v any) bool {
	rv := reflect.ValueOf(v)
	return rv.Kind() == reflect.Pointer && rv.IsNil()
}

func convertSliceAny(v any) (attribute.Value, bool) {
	switch x := v.(type) {
	case []byte:
		return attribute.ByteSliceValue(slices.Clone(x)), true
	case []string:
		return attribute.StringSliceValue(slices.Clone(x)), true
	case []int:
		return attribute.IntSliceValue(slices.Clone(x)), true
	case []int64:
		return attribute.Int64SliceValue(slices.Clone(x)), true
	case []float64:
		return attribute.Float64SliceValue(slices.Clone(x)), true
	case []bool:
		return attribute.BoolSliceValue(slices.Clone(x)), true
	default:
		return attribute.Value{}, false
	}
}

func (h *handler) Enabled(ctx context.Context, level slog.Level) bool {
	if h.level != nil && level < h.level.Level() {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	return h.logger.Enabled(ctx, log.EnabledParameters{
		Severity: convertLevel(level),
	})
}

func (h *handler) Handle(ctx context.Context, rec slog.Record) error {
	if ctx == nil {
		ctx = context.Background()
	}

	var logRec log.Record
	if !rec.Time.IsZero() {
		logRec.SetTimestamp(rec.Time)
	}
	logRec.SetSeverity(convertLevel(rec.Level))
	logRec.SetSeverityText(rec.Level.String())
	logRec.SetBody(attribute.StringValue(rec.Message))

	var attrs []attribute.KeyValue
	if h.addSource && rec.PC != 0 {
		attrs = h.appendSourceAttrs(attrs, rec.PC)
	}

	nodes := h.collectNodes()
	attrs = h.appendChain(attrs, nodes, rec, nil)
	if len(attrs) > 0 {
		logRec.AddAttributes(attrs...)
	}

	h.logger.Emit(ctx, logRec)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	if len(attrs) == 0 {
		return h
	}
	clone := *h
	clone.goa = &groupOrAttrs{
		attrs: slices.Clone(attrs),
		next:  h.goa,
	}
	return &clone
}

func (h *handler) WithGroup(name string) slog.Handler {
	if name == "" {
		return h
	}
	clone := *h
	clone.goa = &groupOrAttrs{
		group: name,
		next:  h.goa,
	}
	return &clone
}

func (h *handler) collectNodes() []*groupOrAttrs {
	if h.goa == nil {
		return nil
	}
	var count int
	for n := h.goa; n != nil; n = n.next {
		count++
	}
	nodes := make([]*groupOrAttrs, count)
	for n := h.goa; n != nil; n = n.next {
		count--
		nodes[count] = n
	}
	return nodes
}

func (h *handler) appendSourceAttrs(out []attribute.KeyValue, pc uintptr) []attribute.KeyValue {
	frames := runtime.CallersFrames([]uintptr{pc})
	frame, _ := frames.Next()
	if frame.File != "" {
		out = h.appendAttr(out, nil, slog.String(codeFilePathKey, frame.File))
	}
	if frame.Function != "" {
		out = h.appendAttr(out, nil, slog.String(codeFunctionNameKey, frame.Function))
	}
	if frame.Line != 0 {
		out = h.appendAttr(out, nil, slog.Int(codeLineNumberKey, frame.Line))
	}
	return out
}

func (h *handler) appendChain(
	out []attribute.KeyValue,
	nodes []*groupOrAttrs,
	rec slog.Record,
	groups []string,
) []attribute.KeyValue {
	for i, n := range nodes {
		if n.group != "" {
			childGroups := appendGroup(groups, n.group)
			children := h.appendChain(nil, nodes[i+1:], rec, childGroups)
			if len(children) > 0 {
				out = append(out, attribute.KeyValue{
					Key:   attribute.Key(n.group),
					Value: attribute.MapValue(children...),
				})
			}
			return out
		}
		for _, attr := range n.attrs {
			out = h.appendAttr(out, groups, attr)
		}
	}
	rec.Attrs(func(attr slog.Attr) bool {
		out = h.appendAttr(out, groups, attr)
		return true
	})
	return out
}

func (h *handler) appendAttr(
	out []attribute.KeyValue,
	groups []string,
	a slog.Attr,
) []attribute.KeyValue {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return out
	}

	if a.Value.Kind() == slog.KindGroup {
		groupAttrs := a.Value.Group()
		if len(groupAttrs) == 0 {
			return out
		}
		if a.Key == "" {
			for _, ga := range groupAttrs {
				out = h.appendAttr(out, groups, ga)
			}
			return out
		}
		childGroups := appendGroup(groups, a.Key)
		var children []attribute.KeyValue
		for _, ga := range groupAttrs {
			children = h.appendAttr(children, childGroups, ga)
		}
		if len(children) > 0 {
			out = append(out, attribute.KeyValue{
				Key:   attribute.Key(a.Key),
				Value: attribute.MapValue(children...),
			})
		}
		return out
	}

	if h.replaceAttr != nil {
		var groupArg []string
		if len(groups) > 0 {
			groupArg = groups[:len(groups):len(groups)]
		}
		a = h.replaceAttr(groupArg, a)
		a.Value = a.Value.Resolve()
		if a.Equal(slog.Attr{}) {
			return out
		}
		if a.Value.Kind() == slog.KindGroup {
			return h.appendConvertedGroup(out, a.Key, a.Value.Group(), true)
		}
	}

	val, ok := h.convertValue(a.Value, true)
	if !ok {
		return out
	}
	return append(out, attribute.KeyValue{
		Key:   attribute.Key(a.Key),
		Value: val,
	})
}

func (h *handler) appendConvertedGroup(
	out []attribute.KeyValue,
	key string,
	groupAttrs []slog.Attr,
	allowFormatError bool,
) []attribute.KeyValue {
	if len(groupAttrs) == 0 {
		return out
	}
	if key == "" {
		for _, ga := range groupAttrs {
			out = h.appendConvertedAttr(out, ga, allowFormatError)
		}
		return out
	}
	var children []attribute.KeyValue
	for _, ga := range groupAttrs {
		children = h.appendConvertedAttr(children, ga, allowFormatError)
	}
	if len(children) > 0 {
		out = append(out, attribute.KeyValue{
			Key:   attribute.Key(key),
			Value: attribute.MapValue(children...),
		})
	}
	return out
}

func (h *handler) appendConvertedAttr(
	out []attribute.KeyValue,
	a slog.Attr,
	allowFormatError bool,
) []attribute.KeyValue {
	a.Value = a.Value.Resolve()
	if a.Equal(slog.Attr{}) {
		return out
	}
	if a.Value.Kind() == slog.KindGroup {
		return h.appendConvertedGroup(out, a.Key, a.Value.Group(), allowFormatError)
	}
	val, ok := h.convertValue(a.Value, allowFormatError)
	if !ok {
		return out
	}
	return append(out, attribute.KeyValue{
		Key:   attribute.Key(a.Key),
		Value: val,
	})
}

func (h *handler) convertValue(v slog.Value, allowFormatError bool) (attribute.Value, bool) {
	v = v.Resolve()
	switch v.Kind() {
	case slog.KindBool:
		return attribute.BoolValue(v.Bool()), true
	case slog.KindInt64:
		return attribute.Int64Value(v.Int64()), true
	case slog.KindUint64:
		return convertUint64(v.Uint64()), true
	case slog.KindFloat64:
		return attribute.Float64Value(v.Float64()), true
	case slog.KindString:
		return attribute.StringValue(v.String()), true
	case slog.KindDuration:
		return attribute.Int64Value(v.Duration().Nanoseconds()), true
	case slog.KindTime:
		return attribute.StringValue(v.Time().Format(time.RFC3339Nano)), true
	case slog.KindGroup:
		var children []attribute.KeyValue
		for _, ga := range v.Group() {
			children = h.appendConvertedAttr(children, ga, allowFormatError)
		}
		if len(children) == 0 {
			return attribute.Value{}, false
		}
		return attribute.MapValue(children...), true
	case slog.KindAny:
		return h.convertAny(v.Any(), allowFormatError)
	default:
		return attribute.StringValue(v.String()), true
	}
}

func (h *handler) convertAny(v any, allowFormatError bool) (attribute.Value, bool) {
	if v == nil {
		return attribute.Value{}, true
	}
	if err, ok := v.(error); ok {
		if isTypedNilPointer(err) {
			return attribute.StringValue("<nil>"), true
		}
		if h.formatError != nil {
			if !allowFormatError {
				return attribute.Value{}, false
			}
			formatted := h.formatError(err).Resolve()
			if formatted.Equal(slog.Value{}) {
				return attribute.Value{}, false
			}
			return h.convertValue(formatted, false)
		}
		return attribute.StringValue(fmt.Sprint(err)), true
	}
	if val, ok := convertSliceAny(v); ok {
		return val, true
	}
	return attribute.StringValue(fmt.Sprint(v)), true
}
