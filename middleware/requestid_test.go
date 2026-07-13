package middleware

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRequestIDOptions(t *testing.T) {
	cases := []struct {
		name       string
		options    []RequestIDOption
		wantUse    bool
		wantHeader string
	}{
		{
			name:       "defaults",
			options:    nil,
			wantUse:    false,
			wantHeader: "",
		},
		{
			name:       "use request ID enabled",
			options:    []RequestIDOption{UseRequestIDOption(true)},
			wantUse:    true,
			wantHeader: "X-Request-Id",
		},
		{
			name:       "use request ID disabled has no header side effect",
			options:    []RequestIDOption{UseRequestIDOption(false)},
			wantUse:    false,
			wantHeader: "",
		},
		{
			name:       "custom header enables use",
			options:    []RequestIDOption{RequestIDHeaderOption("X-Custom-Id")},
			wantUse:    true,
			wantHeader: "X-Custom-Id",
		},
		{
			name: "custom header overrides default header",
			options: []RequestIDOption{
				UseRequestIDOption(true),
				RequestIDHeaderOption("X-Custom-Id"),
			},
			wantUse:    true,
			wantHeader: "X-Custom-Id",
		},
		{
			name: "disable preserves configured header",
			options: []RequestIDOption{
				RequestIDHeaderOption("X-Custom-Id"),
				UseRequestIDOption(false),
			},
			wantUse:    false,
			wantHeader: "X-Custom-Id",
		},
		{
			name:       "limit alone does not enable use",
			options:    []RequestIDOption{RequestIDLimitOption(3)},
			wantUse:    false,
			wantHeader: "",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			o := NewRequestIDOptions(c.options...)
			require.NotNil(t, o)
			assert.Equal(t, c.wantUse, o.IsUseRequestID())
			assert.Equal(t, c.wantHeader, o.RequestIDHeader())
		})
	}
}

func TestGenerateRequestID(t *testing.T) {
	cases := []struct {
		name    string
		options []RequestIDOption
		ctxID   string
		// wantID is the exact expected ID; empty means a generated
		// 8-character short ID is expected instead.
		wantID string
	}{
		{
			name:    "generates ID by default",
			options: nil,
			ctxID:   "",
			wantID:  "",
		},
		{
			name:    "ignores context ID by default",
			options: nil,
			ctxID:   "ctx-id",
			wantID:  "",
		},
		{
			name:    "uses context ID when enabled",
			options: []RequestIDOption{UseRequestIDOption(true)},
			ctxID:   "ctx-id",
			wantID:  "ctx-id",
		},
		{
			name:    "generates ID when enabled but context has none",
			options: []RequestIDOption{UseRequestIDOption(true)},
			ctxID:   "",
			wantID:  "",
		},
		{
			name: "truncates context ID beyond limit",
			options: []RequestIDOption{
				UseRequestIDOption(true),
				RequestIDLimitOption(3),
			},
			ctxID:  "too long for limit",
			wantID: "too",
		},
		{
			name: "keeps context ID within limit",
			options: []RequestIDOption{
				UseRequestIDOption(true),
				RequestIDLimitOption(10),
			},
			ctxID:  "short",
			wantID: "short",
		},
		{
			name: "zero limit means no truncation",
			options: []RequestIDOption{
				UseRequestIDOption(true),
				RequestIDLimitOption(0),
			},
			ctxID:  "not truncated at all",
			wantID: "not truncated at all",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			if c.ctxID != "" {
				ctx = context.WithValue(ctx, RequestIDKey, c.ctxID) // nolint: staticcheck
			}
			o := NewRequestIDOptions(c.options...)

			ctx = GenerateRequestID(ctx, o)

			id, ok := ctx.Value(RequestIDKey).(string)
			require.True(t, ok, "request ID not stored in context")
			if c.wantID != "" {
				assert.Equal(t, c.wantID, id)
				return
			}
			assert.Len(t, id, 8)
			if c.ctxID != "" {
				assert.NotEqual(t, c.ctxID, id)
			}
		})
	}
}

func TestGenerateRequestIDUnique(t *testing.T) {
	o := NewRequestIDOptions()
	seen := make(map[string]struct{})
	for range 100 {
		ctx := GenerateRequestID(context.Background(), o)
		id, ok := ctx.Value(RequestIDKey).(string)
		require.True(t, ok)
		if _, dup := seen[id]; dup {
			t.Errorf("duplicate generated request ID: %q", id)
		}
		seen[id] = struct{}{}
	}
}
