package log

import (
	"encoding/json/v2"
	"errors"
	"fmt"
	"math"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	formatLineStringer string
	formatTypedArray   []any
	formatInvalidJSON  struct {
	}
)

func TestFormatJSONControlCharacters(t *testing.T) {
	for value := range 32 {
		t.Run(fmt.Sprintf("U+%04X", value), func(t *testing.T) {
			input := "before" + string(rune(value)) + "after"
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "value", V: input}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Equal(t, input, decoded["value"])
			require.Equal(t, 1, strings.Count(string(out), "\n"))
		})
	}
}

func TestFormatJSONNamesAndUTF8(t *testing.T) {
	for _, key := range []string{"quote\"", "backslash\\", "line\n", "control\x01"} {
		t.Run(fmt.Sprintf("key_%q", key), func(t *testing.T) {
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: key, V: "retained"}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Equal(t, "retained", decoded[key])
		})
	}
	for _, tc := range []struct {
		name  string
		key   string
		value string
	}{
		{"value", "key", "bad\xffvalue"},
		{"key", "bad\xffkey", "value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: tc.key, V: tc.value}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Equal(t, strings.ToValidUTF8(tc.value, "\ufffd"), decoded[strings.ToValidUTF8(tc.key, "\ufffd")])
		})
	}
}

func TestFormatJSONNonfiniteValues(t *testing.T) {
	for _, value := range []any{math.NaN(), math.Inf(1), math.Inf(-1), float32(math.NaN()), float32(math.Inf(1)), float32(math.Inf(-1)), []any{1, math.NaN()}, []float64{1, math.Inf(1)}} {
		t.Run(fmt.Sprintf("%T_%v", value, value), func(t *testing.T) {
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "before", V: 1}, {K: "value", V: value}, {K: "after", V: true}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Equal(t, float64(1), decoded["before"])
			require.Equal(t, true, decoded["after"])
			require.Contains(t, decoded, "value", "unsupported value must not erase the field or entry")
			require.Equal(t, 1, strings.Count(string(out), "\n"))
		})
	}
}

func TestTextFormattersEscapePresentedValues(t *testing.T) {
	for _, formatter := range []struct {
		name   string
		format FormatFunc
	}{
		{"text", FormatText},
		{"terminal", FormatTerminal},
	} {
		for _, value := range []any{"first\nforged", []string{"first\nforged"}, []any{"first\nforged"}, map[string]string{"value": "first\nforged"}, formatLineStringer("first\nforged")} {
			t.Run(fmt.Sprintf("%s/%T", formatter.name, value), func(t *testing.T) {
				out := formatter.format(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "value", V: value}}})
				require.Equal(t, 1, strings.Count(string(out), "\n"), "one physical line per entry: %q", out)
				require.Contains(t, string(out), `\n`, "retain the line break as escaped data")
			})
		}
	}
}

func TestFormatJSONRetainsFieldSequence(t *testing.T) {
	out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "same", V: 1}, {K: "same", V: 2}, {K: "last", V: uint64(math.MaxUint64)}}})
	require.Contains(t, string(out), `,"same":1,"same":2,"last":18446744073709551615}`)
}

func TestFormatJSONValueSemantics(t *testing.T) {
	for _, tc := range []struct {
		name  string
		value any
		want  any
	}{
		{"NaN", math.NaN(), "NaN"},
		{"positive infinity", math.Inf(1), "Infinity"},
		{"negative infinity", float32(math.Inf(-1)), "-Infinity"},
		{"nested nonfinite", []any{math.NaN(), math.Inf(1)}, []any{"NaN", "Infinity"}},
		{"nil", nil, nil},
		{"bytes", []byte{1, 2, 3}, "AQID"},
		{"nil interface array", []any(nil), []any{}},
		{"nil typed array", []string(nil), []any{}},
		{"large finite", 1e20, float64(1e20)},
		{"small finite", 1e-9, float64(1e-9)},
		{"smallest finite", math.SmallestNonzeroFloat64, math.SmallestNonzeroFloat64},
		{"largest finite", math.MaxFloat64, math.MaxFloat64},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "value", V: tc.value}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Equal(t, tc.want, decoded["value"])
		})
	}
}

func TestFormatJSONContainsCodecErrors(t *testing.T) {
	for _, value := range []any{make(chan int), formatInvalidJSON{}, []any{formatInvalidJSON{}}} {
		t.Run(fmt.Sprintf("%T", value), func(t *testing.T) {
			out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "bad", V: value}, {K: "retained", V: true}}})
			var decoded map[string]any
			require.NoError(t, json.Unmarshal(out, &decoded))
			require.Contains(t, string(out), "<invalid: ")
			require.Equal(t, true, decoded["retained"])
			require.Equal(t, 1, strings.Count(string(out), "\n"))
		})
	}
}

func TestTextFormattersEscapeNamesAndControls(t *testing.T) {
	for _, formatter := range []struct {
		name   string
		format FormatFunc
	}{
		{"text", FormatText}, {"terminal", FormatTerminal},
	} {
		for _, input := range []string{"forged\nentry", "control\x01", "escape\x1b", "bad\xffUTF8", `a="b"`} {
			t.Run(fmt.Sprintf("%s/%q", formatter.name, input), func(t *testing.T) {
				out := string(formatter.format(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: input, V: input}}}))
				require.Equal(t, 1, strings.Count(out, "\n"))
				require.NotContains(t, out, input, "both the name and value must escape unsafe text")
			})
		}
	}
}

func TestFormatJSONContainsDepthErrors(t *testing.T) {
	for _, depth := range []int{9998, 9999, 10000, 10001} {
		for _, typed := range []bool{false, true} {
			t.Run(fmt.Sprintf("depth=%d/typed=%t", depth, typed), func(t *testing.T) {
				var value any = "leaf"
				for range depth {
					value = []any{value}
				}
				if typed {
					value = formatTypedArray(value.([]any))
				}
				out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "before", V: 1}, {K: "deep", V: value}, {K: "after", V: true}}})
				var decoded map[string]any
				require.NoError(t, json.Unmarshal(out, &decoded))
				require.Equal(t, float64(1), decoded["before"])
				require.Equal(t, true, decoded["after"])
				leaf := decoded["deep"]
				for {
					array, ok := leaf.([]any)
					if !ok {
						break
					}
					require.Len(t, array, 1)
					leaf = array[0]
				}
				if depth < 10000 {
					require.Equal(t, "leaf", leaf)
				} else {
					require.Contains(t, leaf, "<invalid: ")
				}
			})
		}
	}
}

func TestFormatJSONContainsRecursiveArray(t *testing.T) {
	value := make([]any, 1)
	value[0] = value
	out := FormatJSON(&Entry{Time: testTime, Severity: SeverityInfo, KeyVals: kvList{{K: "cycle", V: value}, {K: "after", V: true}}})
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(out, &decoded))
	require.Equal(t, true, decoded["after"])
	require.Contains(t, string(out), "<invalid: ")
}

func BenchmarkFormatterSerialization(b *testing.B) {
	for _, formatter := range []struct {
		name   string
		format FormatFunc
	}{
		{"text", FormatText}, {"JSON", FormatJSON}, {"terminal", FormatTerminal},
	} {
		for _, tc := range []struct {
			name   string
			values kvList
		}{
			{"ordinary", kvList{{K: "msg", V: "request completed"}, {K: "status", V: 200}, {K: "elapsed", V: 1.25}}},
			{"structured", kvList{{K: "msg", V: "with arrays"}, {K: "values", V: []any{"a b", 1, true}}, {K: "typed", V: []string{"first", "second"}}}},
		} {
			b.Run(formatter.name+"/"+tc.name, func(b *testing.B) {
				entry := &Entry{Time: testTime, Severity: SeverityInfo, KeyVals: tc.values}
				b.ReportAllocs()
				for b.Loop() {
					formatter.format(entry)
				}
			})
		}
	}
}

// String returns the raw value to exercise fallback presentation escaping.
func (s formatLineStringer) String() string {
	return string(s)
}

// MarshalJSON returns a codec error containing unsafe diagnostic text.
func (formatInvalidJSON) MarshalJSON() ([]byte, error) {
	return nil, errors.New("invalid\nvalue\x01")
}
