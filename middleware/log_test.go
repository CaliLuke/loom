package middleware

import (
	"bytes"
	"log"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoggerLog(t *testing.T) {
	cases := []struct {
		name    string
		keyvals []any
		want    string
	}{
		{
			name:    "single pair",
			keyvals: []any{"key", "value"},
			want:    "key=value\n",
		},
		{
			name:    "multiple pairs",
			keyvals: []any{"id", "abc", "status", 200},
			want:    "id=abc status=200\n",
		},
		{
			name:    "odd count appends MISSING",
			keyvals: []any{"orphan"},
			want:    "orphan=MISSING\n",
		},
		{
			name:    "no keyvals",
			keyvals: nil,
			want:    "\n",
		},
		{
			name:    "non-string values",
			keyvals: []any{"ok", true, "count", 42},
			want:    "ok=true count=42\n",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var buf bytes.Buffer
			logger := NewLogger(log.New(&buf, "", 0))
			require.NotNil(t, logger)

			err := logger.Log(c.keyvals...)

			require.NoError(t, err)
			assert.Equal(t, c.want, buf.String())
		})
	}
}
