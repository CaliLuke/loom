package log

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
)

// testTime is a fixed timestamp used by formatter tests.
var testTime = time.Date(2022, time.January, 9, 20, 29, 45, 0, time.UTC)

func TestFormatText(t *testing.T) {
	prefix := "time=2022-01-09T20:29:45Z level=info"
	cases := []struct {
		name    string
		keyvals kvList
		want    string
	}{
		{
			name: "no keyvals",
			want: prefix + "\n",
		},
		{
			name:    "plain string",
			keyvals: kvList{{K: "k", V: "hello"}},
			want:    prefix + " k=hello\n",
		},
		{
			name:    "string with space is quoted",
			keyvals: kvList{{K: "k", V: "hello world"}},
			want:    prefix + " k=\"hello world\"\n",
		},
		{
			name:    "string with quote and backslash is escaped",
			keyvals: kvList{{K: "k", V: `say "hi"\now`}},
			want:    prefix + ` k="say \"hi\"\\now"` + "\n",
		},
		{
			name:    "control characters stay on one logfmt line",
			keyvals: kvList{{K: "k", V: "first\nsecond\rthird\tfourth"}},
			want:    prefix + ` k="first\nsecond\rthird\tfourth"` + "\n",
		},
		{
			name:    "empty string is quoted",
			keyvals: kvList{{K: "k", V: ""}},
			want:    prefix + ` k=""` + "\n",
		},
		{
			name:    "string with equal sign is quoted",
			keyvals: kvList{{K: "k", V: "a=b"}},
			want:    prefix + ` k="a=b"` + "\n",
		},
		{
			name:    "int",
			keyvals: kvList{{K: "k", V: 42}},
			want:    prefix + " k=42\n",
		},
		{
			name:    "bool",
			keyvals: kvList{{K: "k", V: true}},
			want:    prefix + " k=true\n",
		},
		{
			name:    "float64",
			keyvals: kvList{{K: "k", V: 1.5}},
			want:    prefix + " k=1.5\n",
		},
		{
			name:    "array",
			keyvals: kvList{{K: "k", V: []any{"a b", 1, true}}},
			want:    prefix + ` k=["a b" 1 true]` + "\n",
		},
		{
			name:    "fallback to fmt for other types",
			keyvals: kvList{{K: "k", V: 3 * time.Second}},
			want:    prefix + " k=3s\n",
		},
		{
			name:    "multiple keyvals keep order",
			keyvals: kvList{{K: "a", V: 1}, {K: "b", V: "two"}},
			want:    prefix + " a=1 b=two\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &Entry{Time: testTime, Severity: SeverityInfo, KeyVals: tc.keyvals}
			require.Equal(t, tc.want, string(FormatText(e)))
		})
	}
}

func TestFormatTextSeverities(t *testing.T) {
	cases := []struct {
		sev  Severity
		want string
	}{
		{SeverityDebug, "level=debug"},
		{SeverityInfo, "level=info"},
		{SeverityWarn, "level=warn"},
		{SeverityError, "level=error"},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			e := &Entry{Time: testTime, Severity: tc.sev}
			require.Contains(t, string(FormatText(e)), tc.want)
		})
	}
}

func TestFormatJSON(t *testing.T) {
	prefix := `{"time":"2022-01-09T20:29:45Z","level":"info"`
	cases := []struct {
		name    string
		keyvals kvList
		want    string
	}{
		{
			name: "no keyvals",
			want: prefix + "}\n",
		},
		{
			name:    "string",
			keyvals: kvList{{K: "k", V: "hello"}},
			want:    prefix + `,"k":"hello"}` + "\n",
		},
		{
			name:    "string escapes",
			keyvals: kvList{{K: "k", V: "a\"b\\c\nd\te\rf\bg\fh"}},
			want:    prefix + `,"k":"a\"b\\c\nd\te\rf\bg\fh"}` + "\n",
		},
		{
			name:    "non-ascii string preserved",
			keyvals: kvList{{K: "k", V: "héllo"}},
			want:    prefix + `,"k":"héllo"}` + "\n",
		},
		{
			name:    "int",
			keyvals: kvList{{K: "k", V: 42}},
			want:    prefix + `,"k":42}` + "\n",
		},
		{
			name:    "int32",
			keyvals: kvList{{K: "k", V: int32(-7)}},
			want:    prefix + `,"k":-7}` + "\n",
		},
		{
			name:    "int64",
			keyvals: kvList{{K: "k", V: int64(1 << 40)}},
			want:    prefix + `,"k":1099511627776}` + "\n",
		},
		{
			name:    "uint",
			keyvals: kvList{{K: "k", V: uint(7)}},
			want:    prefix + `,"k":7}` + "\n",
		},
		{
			name:    "uint32",
			keyvals: kvList{{K: "k", V: uint32(8)}},
			want:    prefix + `,"k":8}` + "\n",
		},
		{
			name:    "uint64",
			keyvals: kvList{{K: "k", V: uint64(9)}},
			want:    prefix + `,"k":9}` + "\n",
		},
		{
			name:    "float32",
			keyvals: kvList{{K: "k", V: float32(1.5)}},
			want:    prefix + `,"k":1.5}` + "\n",
		},
		{
			name:    "float64",
			keyvals: kvList{{K: "k", V: 2.25}},
			want:    prefix + `,"k":2.25}` + "\n",
		},
		{
			name:    "bool",
			keyvals: kvList{{K: "k", V: false}},
			want:    prefix + `,"k":false}` + "\n",
		},
		{
			name:    "array",
			keyvals: kvList{{K: "k", V: []any{"a", 1, true}}},
			want:    prefix + `,"k":["a",1,true]}` + "\n",
		},
		{
			name:    "duration",
			keyvals: kvList{{K: "k", V: 1500 * time.Millisecond}},
			want:    prefix + `,"k":"1.5s"}` + "\n",
		},
		{
			name:    "grpc code",
			keyvals: kvList{{K: "k", V: codes.Internal}},
			want:    prefix + `,"k":"Internal"}` + "\n",
		},
		{
			name:    "fallback to json.Marshal",
			keyvals: kvList{{K: "k", V: map[string]int{"a": 1}}},
			want:    prefix + `,"k":{"a":1}}` + "\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &Entry{Time: testTime, Severity: SeverityInfo, KeyVals: tc.keyvals}
			got := string(FormatJSON(e))
			require.Equal(t, tc.want, got)
			var parsed map[string]any
			require.NoError(t, json.Unmarshal([]byte(got), &parsed), "output must be valid JSON")
			assert.Equal(t, "2022-01-09T20:29:45Z", parsed[TimestampKey])
			assert.Equal(t, "info", parsed[SeverityKey])
		})
	}
}

func TestFormatJSONSeverities(t *testing.T) {
	cases := []struct {
		sev  Severity
		want string
	}{
		{SeverityDebug, `"level":"debug"`},
		{SeverityWarn, `"level":"warn"`},
		{SeverityError, `"level":"error"`},
	}
	for _, tc := range cases {
		t.Run(tc.want, func(t *testing.T) {
			e := &Entry{Time: testTime, Severity: tc.sev}
			require.Contains(t, string(FormatJSON(e)), tc.want)
		})
	}
}

func TestFormatTerminal(t *testing.T) {
	cases := []struct {
		name    string
		sev     Severity
		elapsed time.Duration
		keyvals kvList
		want    string
	}{
		{
			name:    "no keyvals",
			sev:     SeverityDebug,
			elapsed: 0,
			want:    ColorSeverityDebug + "DEBG" + reset + "[0000]\n",
		},
		{
			name:    "info with keyvals",
			sev:     SeverityInfo,
			elapsed: 5 * time.Second,
			keyvals: kvList{{K: "msg", V: "hello"}, {K: "count", V: 42}},
			want: ColorSeverityInfo + "INFO" + reset + "[0005] " +
				ColorSeverityInfo + "msg" + reset + "=hello " +
				ColorSeverityInfo + "count" + reset + "=42\n",
		},
		{
			name:    "error",
			sev:     SeverityError,
			elapsed: 12 * time.Second,
			keyvals: kvList{{K: "err", V: "boom"}},
			want: ColorSeverityError + "ERRO" + reset + "[0012] " +
				ColorSeverityError + "err" + reset + "=boom\n",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			e := &Entry{Time: epoch.Add(tc.elapsed), Severity: tc.sev, KeyVals: tc.keyvals}
			require.Equal(t, tc.want, string(FormatTerminal(e)))
		})
	}
}

func TestTimestampFormatLayout(t *testing.T) {
	old := TimestampFormatLayout
	TimestampFormatLayout = time.Kitchen
	t.Cleanup(func() {
		TimestampFormatLayout = old
	})

	e := &Entry{Time: testTime, Severity: SeverityInfo}
	require.Equal(t, "time=8:29PM level=info\n", string(FormatText(e)))
	require.Equal(t, fmt.Sprintf(`{"time":"8:29PM","level":"info"}%s`, "\n"), string(FormatJSON(e)))
}
