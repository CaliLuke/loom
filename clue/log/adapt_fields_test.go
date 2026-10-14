package log

import (
	"bytes"
	"errors"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
)

func TestLogrArgumentNormalization(t *testing.T) {
	for _, mode := range []string{"info", "verbose", "error", "bound"} {
		for _, tc := range []struct {
			name string
			args []any
			want string
		}{
			{name: "empty"},
			{name: "pair", args: []any{"key", "value"}, want: "key=value"},
			{name: "orphan", args: []any{"orphan"}, want: "orphan=<no-value>"},
			{name: "pair and orphan", args: []any{"key", "value", "orphan"}, want: "key=value orphan=<no-value>"},
			{name: "repeated keys", args: []any{"same", 1, "same", 2}, want: "same=1 same=2"},
			{name: "non-string key", args: []any{7, "value"}, want: "7=value"},
			{name: "nil value", args: []any{"nothing", nil}, want: "nothing=<nil>"},
		} {
			t.Run(mode+"/"+tc.name, func(t *testing.T) {
				var buf bytes.Buffer
				logger := logr.New(ToLogrSink(newTestContext(&buf, WithDebug())))
				require.NotPanics(t, func() {
					switch mode {
					case "info":
						logger.Info("message", tc.args...)
					case "verbose":
						logger.V(2).Info("message", tc.args...)
					case "error":
						logger.Error(errors.New("failed"), "message", tc.args...)
					case "bound":
						logger.WithValues(tc.args...).Info("message", "event", "value")
					}
				})
				require.Contains(t, buf.String(), "msg=message")
				require.Contains(t, buf.String(), tc.want)
				if mode == "error" {
					require.Contains(t, buf.String(), "err=failed")
				}
			})
		}
	}
}

func TestLogrBoundFieldsRemainIndependent(t *testing.T) {
	var buf bytes.Buffer
	logger := logr.New(ToLogrSink(newTestContext(&buf, WithDebug()))).WithName("parent")
	args := []any{"bound", "orphan"}
	child := logger.WithValues(args...).WithValues("missing").WithName("child")
	args[0], args[1] = "changed", "changed"
	child.Info("child", "event", "value")
	require.Contains(t, buf.String(), "bound=orphan missing=<no-value>")
	require.Contains(t, buf.String(), "event=value")
	require.Contains(t, buf.String(), "log=parent/child")
	require.NotContains(t, buf.String(), "changed")
	buf.Reset()
	logger.Info("parent")
	require.NotContains(t, buf.String(), "bound=")
	require.NotContains(t, buf.String(), "missing=")
	require.NotContains(t, buf.String(), "child")
}
