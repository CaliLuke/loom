package middleware_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	httpm "github.com/CaliLuke/loom/http/middleware"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type ctxTestKey string

func TestRequestContext(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxTestKey("key"), "value")

	var got context.Context
	handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		got = r.Context()
	})

	req := httptest.NewRequest("GET", "/", nil)
	httpm.RequestContext(ctx)(handler).ServeHTTP(httptest.NewRecorder(), req)

	require.NotNil(t, got, "wrapped handler was not invoked")
	assert.Equal(t, "value", got.Value(ctxTestKey("key")))
}

func TestRequestContextKeyVals(t *testing.T) {
	cases := []struct {
		name    string
		keyvals []any
		want    map[ctxTestKey]any
	}{
		{
			name:    "no keyvals",
			keyvals: nil,
			want:    map[ctxTestKey]any{},
		},
		{
			name:    "single pair",
			keyvals: []any{ctxTestKey("a"), "1"},
			want:    map[ctxTestKey]any{"a": "1"},
		},
		{
			name:    "multiple pairs",
			keyvals: []any{ctxTestKey("a"), "1", ctxTestKey("b"), 2},
			want:    map[ctxTestKey]any{"a": "1", "b": 2},
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var got context.Context
			handler := http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
				got = r.Context()
			})

			req := httptest.NewRequest("GET", "/", nil)
			httpm.RequestContextKeyVals(c.keyvals...)(handler).ServeHTTP(httptest.NewRecorder(), req)

			require.NotNil(t, got, "wrapped handler was not invoked")
			for k, v := range c.want {
				assert.Equal(t, v, got.Value(k))
			}
		})
	}
}

func TestRequestContextKeyValsOddCountPanics(t *testing.T) {
	require.PanicsWithValue(t,
		"initctx: invalid number of key/value elements, must be an even number",
		func() {
			//lint:ignore SA5012 Odd argument count is the behavior under test.
			httpm.RequestContextKeyVals(ctxTestKey("a"))
		})
}
