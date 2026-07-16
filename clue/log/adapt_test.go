package log

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/aws/smithy-go/logging"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStdLoggerPrint(t *testing.T) {
	cases := []struct {
		name string
		log  func(l *StdLogger)
		want string
	}{
		{
			name: "print",
			log: func(l *StdLogger) {
				l.Print("hello ", 42)
			},
			want: `msg="hello 42"`,
		},
		{
			name: "printf",
			log: func(l *StdLogger) {
				l.Printf("count=%d", 7)
			},
			want: `msg="count=7"`,
		},
		{
			name: "println",
			log: func(l *StdLogger) {
				l.Println("line")
			},
			want: `msg="line\n"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := AsStdLogger(newTestContext(&buf))
			tc.log(l)
			require.Contains(t, buf.String(), tc.want)
		})
	}
}

func TestStdLoggerFatal(t *testing.T) {
	cases := []struct {
		name string
		log  func(l *StdLogger)
		want string
	}{
		{
			name: "fatal",
			log: func(l *StdLogger) {
				l.Fatal("boom")
			},
			want: "msg=boom",
		},
		{
			name: "fatalf",
			log: func(l *StdLogger) {
				l.Fatalf("boom %d", 2)
			},
			want: `msg="boom 2"`,
		},
		{
			name: "fatalln",
			log: func(l *StdLogger) {
				l.Fatalln("boom")
			},
			want: `msg="boom\n"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			code := stubOSExit(t)
			var buf bytes.Buffer
			l := AsStdLogger(newTestContext(&buf))
			tc.log(l)
			require.Equal(t, 1, *code)
			require.Contains(t, buf.String(), tc.want)
		})
	}
}

func TestStdLoggerPanic(t *testing.T) {
	cases := []struct {
		name      string
		log       func(l *StdLogger)
		wantPanic string
		wantLog   string
	}{
		{
			name: "panic",
			log: func(l *StdLogger) {
				l.Panic("boom")
			},
			wantPanic: "boom",
			wantLog:   "msg=boom",
		},
		{
			name: "panicf",
			log: func(l *StdLogger) {
				l.Panicf("boom %d", 3)
			},
			wantPanic: "boom 3",
			wantLog:   `msg="boom 3"`,
		},
		{
			name: "panicln",
			log: func(l *StdLogger) {
				l.Panicln("boom")
			},
			wantPanic: "boom\n",
			wantLog:   `msg="boom\n"`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := AsStdLogger(newTestContext(&buf))
			require.PanicsWithValue(t, tc.wantPanic, func() {
				tc.log(l)
			})
			require.Contains(t, buf.String(), tc.wantLog)
		})
	}
}

func TestAWSLogger(t *testing.T) {
	cases := []struct {
		name           string
		classification logging.Classification
		want           string
	}{
		{"warn logs at warn", logging.Warn, "level=warn"},
		{"debug logs at debug", logging.Debug, "level=debug"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer
			l := AsAWSLogger(newTestContext(&buf, WithDebug()))
			l.Logf(tc.classification, "aws %s", "message")
			require.Contains(t, buf.String(), tc.want)
			require.Contains(t, buf.String(), `msg="aws message"`)
		})
	}
}

func TestAWSLoggerWithContext(t *testing.T) {
	type contextKey string
	const key contextKey = "key"

	var buf bytes.Buffer
	original := context.WithValue(newTestContext(&buf, WithDebug()), key, "original")
	replacement := context.WithValue(newTestContext(&bytes.Buffer{}), key, "replacement")
	l := AsAWSLogger(original)
	l2 := l.WithContext(replacement)

	// The returned logger keeps the original clue logger attached to the new
	// context.
	awsl, ok := l2.(*AWSLogger)
	require.True(t, ok)
	require.Equal(t, "original", l.Value(key))
	require.Equal(t, "replacement", awsl.Value(key))
	awsl.Logf(logging.Warn, "relogged")
	require.Contains(t, buf.String(), "msg=relogged")
}

func TestLogrSink(t *testing.T) {
	var buf bytes.Buffer
	sink := ToLogrSink(newTestContext(&buf, WithDebug()))
	sink.Init(logr.RuntimeInfo{})
	require.True(t, sink.Enabled(0))
	require.True(t, sink.Enabled(9))

	cases := []struct {
		name string
		log  func()
		want []string
	}{
		{
			name: "info level zero",
			log: func() {
				sink.Info(0, "hello", "k", "v")
			},
			want: []string{"level=info", "msg=hello", "k=v"},
		},
		{
			name: "info level nonzero logs debug",
			log: func() {
				sink.Info(2, "verbose", "k2", 42)
			},
			want: []string{"level=debug", "msg=verbose", "k2=42"},
		},
		{
			name: "error",
			log: func() {
				sink.Error(errors.New("boom"), "failed", "k3", true)
			},
			want: []string{"level=error", "err=boom", "msg=failed", "k3=true"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			tc.log()
			for _, want := range tc.want {
				assert.Contains(t, buf.String(), want)
			}
		})
	}
}

func TestLogrSinkWithValues(t *testing.T) {
	var buf bytes.Buffer
	sink := ToLogrSink(newTestContext(&buf, WithDebug()))
	child := sink.WithValues("bound", "value")
	child.Info(0, "hi")

	require.Contains(t, buf.String(), "bound=value")
	require.Contains(t, buf.String(), "msg=hi")

	buf.Reset()
	named := sink.WithName("outer").(*LogrSink)
	nested := named.WithValues("bound", "value").(*LogrSink).WithName("inner")
	nested.Info(0, "nested")
	require.Contains(t, buf.String(), "log=outer/inner")
	require.Contains(t, buf.String(), "bound=value")
}

func TestLogrSinkWithName(t *testing.T) {
	var buf bytes.Buffer
	sink := ToLogrSink(newTestContext(&buf, WithDebug()))
	named := sink.WithName("outer")
	named.Info(0, "first")
	require.Contains(t, buf.String(), "log=outer")

	buf.Reset()
	nested := named.(*LogrSink).WithName("inner")
	nested.Info(0, "second")
	require.Contains(t, buf.String(), "log=outer/inner")

	buf.Reset()
	firstSibling := sink.WithName("first")
	secondSibling := sink.WithName("second")
	firstSibling.Info(0, "first sibling")
	secondSibling.Info(0, "second sibling")
	require.Contains(t, buf.String(), "log=first")
	require.Contains(t, buf.String(), "log=second")
	require.NotContains(t, buf.String(), "log=first/second")
}
