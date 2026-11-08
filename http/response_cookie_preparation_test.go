package http

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPrepareResponseCookie(t *testing.T) {
	policyFailure := errors.New("deployment refused cookie")
	for _, tc := range []struct {
		name   string
		cookie *http.Cookie
		policy ResponseCookiePolicy
		want   string
		err    string
	}{
		{name: "plain", cookie: &http.Cookie{Name: "session", Value: "secret"}, want: "session=secret"},
		{name: "nil", err: "cannot be nil"},
		{name: "invalid name", cookie: &http.Cookie{Name: "bad name", Value: "secret"}, err: "validate response cookie"},
		{name: "policy attributes", cookie: &http.Cookie{Name: "session", Value: "secret", Path: "/"}, policy: func(_ context.Context, cookie *http.Cookie) error {
			cookie.Secure = true
			cookie.Domain = "example.test"
			return nil
		}, want: "session=secret; Path=/; Domain=example.test; Secure"},
		{name: "policy error", cookie: &http.Cookie{Name: "session", Value: "secret"}, policy: func(context.Context, *http.Cookie) error {
			return policyFailure
		}, err: "deployment refused cookie"},
		{name: "contract mutation", cookie: &http.Cookie{Name: "session", Value: "secret"}, policy: func(_ context.Context, cookie *http.Cookie) error {
			cookie.Value = "changed"
			return nil
		}, err: "may change only"},
		{name: "host security", cookie: &http.Cookie{Name: "__Host-session", Value: "secret", Path: "/"}, err: "requires Secure"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			calls := 0
			ctx := context.WithValue(request.Context(), struct{ name string }{"sentinel"}, "retained")
			var prepared string
			var prepareErr error
			prepare := http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				prepared, prepareErr = PrepareResponseCookie(request.Context(), tc.cookie)
			})
			if tc.policy != nil {
				policy := ResponseCookiePolicy(func(ctx context.Context, cookie *http.Cookie) error {
					calls++
					require.Equal(t, "retained", ctx.Value(struct{ name string }{"sentinel"}))
					return tc.policy(ctx, cookie)
				})
				policy.Handler(prepare).ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
				require.Equal(t, 1, calls)
			} else {
				prepare.ServeHTTP(httptest.NewRecorder(), request.WithContext(ctx))
				require.Zero(t, calls)
			}
			if tc.err != "" {
				require.ErrorContains(t, prepareErr, tc.err)
				require.Empty(t, prepared)
				if tc.name == "policy error" {
					require.ErrorIs(t, prepareErr, policyFailure)
				}
				return
			}
			require.NoError(t, prepareErr)
			require.Equal(t, tc.want, prepared)
		})
	}
}
