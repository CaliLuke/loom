package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	loom "github.com/CaliLuke/loom/pkg"
)

func TestRequestBodyPolicy(t *testing.T) {
	tests := []struct {
		name      string
		maxBytes  int64
		body      string
		wantValue string
		wantCode  string
	}{
		{name: "below limit", maxBytes: 32, body: `{"value":"ok"}`, wantValue: "ok"},
		{name: "at limit", maxBytes: 14, body: `{"value":"ok"}`, wantValue: "ok"},
		{name: "above limit", maxBytes: 13, body: `{"value":"ok"}`, wantCode: loom.RequestBodyTooLarge},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			policy, err := NewRequestBodyPolicy(test.maxBytes)
			require.NoError(t, err)
			require.Equal(t, test.maxBytes, policy.MaxBytes())

			var got string
			var decodeErr error
			var gotLimit int64
			handler := policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
				gotLimit = RequestBodyLimit(request.Context())
				var payload struct {
					Value string `json:"value"`
				}
				decodeErr = RequestDecoder(request).Decode(&payload)
				got = payload.Value
			}))

			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			handler.ServeHTTP(httptest.NewRecorder(), request)

			require.Equal(t, test.maxBytes, gotLimit)
			if test.wantCode == "" {
				require.NoError(t, decodeErr)
				require.Equal(t, test.wantValue, got)
				return
			}
			var serviceErr *loom.ServiceError
			require.ErrorAs(t, decodeErr, &serviceErr)
			require.Equal(t, test.wantCode, serviceErr.Name)
		})
	}
}

func TestNewRequestBodyPolicyRejectsNonPositiveLimits(t *testing.T) {
	for _, limit := range []int64{-1, 0} {
		_, err := NewRequestBodyPolicy(limit)
		require.Error(t, err)
	}
}

func TestRequestBodyPolicyLimitsRawReaders(t *testing.T) {
	policy, err := NewRequestBodyPolicy(3)
	require.NoError(t, err)

	var readErr error
	handler := policy.Handler(http.HandlerFunc(func(_ http.ResponseWriter, request *http.Request) {
		_, readErr = io.ReadAll(request.Body)
	}))
	handler.ServeHTTP(
		httptest.NewRecorder(),
		httptest.NewRequest(http.MethodPost, "/", strings.NewReader("four")),
	)

	var serviceErr *loom.ServiceError
	require.ErrorAs(t, readErr, &serviceErr)
	require.Equal(t, loom.RequestBodyTooLarge, serviceErr.Name)
}

func TestResponseCookiePolicyMiddleware(t *testing.T) {
	expires := time.Date(2030, time.January, 2, 3, 4, 5, 0, time.UTC)
	policy := ResponseCookiePolicy(func(_ context.Context, cookie *http.Cookie) error {
		cookie.Domain = "example.test"
		cookie.Secure = false
		cookie.Expires = expires
		return nil
	})
	handler := policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := SetResponseCookie(request.Context(), response, &http.Cookie{
			Name:     "session",
			Value:    "secret",
			Path:     "/",
			Secure:   true,
			HttpOnly: true,
		})
		require.NoError(t, err)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	require.Equal(t, "example.test", cookies[0].Domain)
	require.False(t, cookies[0].Secure)
	require.Equal(t, expires, cookies[0].Expires)
}

func TestResponseCookiePolicyFailureDoesNotWriteCookie(t *testing.T) {
	wantErr := errors.New("deployment cookie policy is invalid")
	policy := ResponseCookiePolicy(func(context.Context, *http.Cookie) error {
		return wantErr
	})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	handler := policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		err := SetResponseCookie(request.Context(), response, &http.Cookie{Name: "session", Value: "secret"})
		require.ErrorIs(t, err, wantErr)
	}))
	handler.ServeHTTP(recorder, request)

	require.Empty(t, recorder.Header().Values("Set-Cookie"))
}

func TestSetResponseCookieRejectsNilCookie(t *testing.T) {
	err := SetResponseCookie(context.Background(), httptest.NewRecorder(), nil)
	require.Error(t, err)
}

func TestSetResponseCookieAlwaysValidatesCookie(t *testing.T) {
	recorder := httptest.NewRecorder()
	err := SetResponseCookie(context.Background(), recorder, &http.Cookie{
		Name:  "invalid name",
		Value: "secret",
	})

	require.Error(t, err)
	require.Empty(t, recorder.Header().Values("Set-Cookie"))
}

func TestResponseCookiePolicyCannotChangeContractIdentity(t *testing.T) {
	tests := []struct {
		name   string
		policy ResponseCookiePolicy
	}{
		{
			name: "name",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Name = "other"
				return nil
			},
		},
		{
			name: "value",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Value = "other"
				return nil
			},
		},
		{
			name: "secure prefix",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Secure = false
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := test.policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				err := SetResponseCookie(request.Context(), response, &http.Cookie{
					Name:   "__Secure-session",
					Value:  "secret",
					Secure: true,
				})
				require.Error(t, err)
			}))

			handler.ServeHTTP(recorder, request)

			require.Empty(t, recorder.Header().Values("Set-Cookie"))
		})
	}
}

func TestResponseCookiePolicyCannotChangeDesignedAttributes(t *testing.T) {
	tests := []struct {
		name   string
		policy ResponseCookiePolicy
	}{
		{
			name: "path",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Path = "/other"
				return nil
			},
		},
		{
			name: "HTTP only",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.HttpOnly = false
				return nil
			},
		},
		{
			name: "same site",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.SameSite = http.SameSiteNoneMode
				return nil
			},
		},
		{
			name: "max age",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.MaxAge = 60
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := test.policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				err := SetResponseCookie(request.Context(), response, &http.Cookie{
					Name:     "session",
					Value:    "secret",
					Path:     "/",
					MaxAge:   3600,
					Secure:   true,
					HttpOnly: true,
					SameSite: http.SameSiteLaxMode,
				})
				require.Error(t, err)
			}))

			handler.ServeHTTP(recorder, request)

			require.Empty(t, recorder.Header().Values("Set-Cookie"))
		})
	}
}

func TestResponseCookiePolicyPreservesHostPrefixRules(t *testing.T) {
	tests := []struct {
		name   string
		policy ResponseCookiePolicy
	}{
		{
			name: "domain",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Domain = "example.test"
				return nil
			},
		},
		{
			name: "path",
			policy: func(_ context.Context, cookie *http.Cookie) error {
				cookie.Path = "/session"
				return nil
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			handler := test.policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
				err := SetResponseCookie(request.Context(), response, &http.Cookie{
					Name:   "__Host-session",
					Value:  "secret",
					Path:   "/",
					Secure: true,
				})
				require.Error(t, err)
			}))

			handler.ServeHTTP(recorder, request)

			require.Empty(t, recorder.Header().Values("Set-Cookie"))
		})
	}
}

func TestResponseNegotiationPolicy(t *testing.T) {
	policy, err := NewResponseNegotiationPolicy("application/json")
	require.NoError(t, err)

	tests := []struct {
		name       string
		accept     string
		wantStatus int
		wantCalled bool
	}{
		{name: "missing", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "exact", accept: "application/json", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "parameters", accept: "application/json; charset=utf-8", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "type wildcard", accept: "application/*", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "any wildcard", accept: "*/*", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "weighted match", accept: "text/html, application/json;q=0.5", wantStatus: http.StatusNoContent, wantCalled: true},
		{name: "explicit rejection wins over wildcard", accept: "application/json;q=0, */*;q=1", wantStatus: http.StatusNotAcceptable},
		{name: "unsupported", accept: "text/html", wantStatus: http.StatusNotAcceptable},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			called := false
			handler := policy.Handler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				called = true
				response.WriteHeader(http.StatusNoContent)
			}))
			request := httptest.NewRequest(http.MethodGet, "/", nil)
			if test.accept != "" {
				request.Header.Set("Accept", test.accept)
			}
			recorder := httptest.NewRecorder()

			handler.ServeHTTP(recorder, request)

			require.Equal(t, test.wantStatus, recorder.Code)
			require.Equal(t, test.wantCalled, called)
			require.Contains(t, recorder.Header().Values("Vary"), "Accept")
			if test.wantStatus == http.StatusNotAcceptable {
				require.Equal(t, ProblemJSONContentType, recorder.Header().Get("Content-Type"))
				require.Contains(t, recorder.Body.String(), `"code":"not_acceptable"`)
			}
		})
	}
}

func TestNewResponseNegotiationPolicyRejectsInvalidMediaTypes(t *testing.T) {
	tests := [][]string{nil, {""}, {"not a media type"}, {"application/*"}}
	for _, mediaTypes := range tests {
		_, err := NewResponseNegotiationPolicy(mediaTypes...)
		require.Error(t, err)
	}
}

func TestResponseNegotiationPolicyReturnsIndependentNormalizedMediaTypes(t *testing.T) {
	policy, err := NewResponseNegotiationPolicy(
		"Application/JSON; charset=utf-8",
		"application/json",
	)
	require.NoError(t, err)

	mediaTypes := policy.MediaTypes()
	require.Equal(t, []string{"application/json"}, mediaTypes)
	mediaTypes[0] = "text/plain"
	require.Equal(t, []string{"application/json"}, policy.MediaTypes())
}

func TestZeroValueTransportPoliciesPanic(t *testing.T) {
	next := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	require.Panics(t, func() {
		RequestBodyPolicy{}.Handler(next)
	})
	require.Panics(t, func() {
		ResponseCookiePolicy(nil).Handler(next)
	})
	require.Panics(t, func() {
		ResponseNegotiationPolicy{}.Handler(next)
	})
}

func TestDerivedHeadHandler(t *testing.T) {
	handler := DerivedHeadHandler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		response.Header().Add("Set-Cookie", "one=1")
		response.Header().Add("Set-Cookie", "two=2")
		response.Header().Set("Content-Type", "application/json")
		response.WriteHeader(http.StatusCreated)
		written, err := response.Write([]byte(`{"value":"ok"}`))
		require.NoError(t, err)
		require.Equal(t, 14, written)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/", nil))

	require.Equal(t, http.StatusCreated, recorder.Code)
	require.Empty(t, recorder.Body.String())
	require.Equal(t, "14", recorder.Header().Get("Content-Length"))
	require.Equal(t, []string{"one=1", "two=2"}, recorder.Header().Values("Set-Cookie"))
}

func TestDerivedHeadHandlerOmitsLengthForBodylessStatuses(t *testing.T) {
	for _, status := range []int{http.StatusNoContent, http.StatusResetContent, http.StatusNotModified} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			handler := DerivedHeadHandler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
				response.WriteHeader(status)
				_, err := response.Write([]byte("ignored"))
				require.NoError(t, err)
			}))

			recorder := httptest.NewRecorder()
			handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/", nil))

			require.Equal(t, status, recorder.Code)
			require.Empty(t, recorder.Header().Get("Content-Length"))
		})
	}
}

func TestDerivedHeadHandlerCountsReaderFromWrites(t *testing.T) {
	handler := DerivedHeadHandler(http.HandlerFunc(func(response http.ResponseWriter, _ *http.Request) {
		written, err := io.Copy(response, io.LimitReader(strings.NewReader("payload"), 7))
		require.NoError(t, err)
		require.Equal(t, int64(7), written)
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Body.String())
	require.Equal(t, "7", recorder.Header().Get("Content-Length"))
}

func TestMountDerivedHead(t *testing.T) {
	mux := NewMuxer()
	MountDerivedHead(mux, "/items/{id}", http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		require.Equal(t, "42", mux.Vars(request)["id"])
		_, err := response.Write([]byte("item"))
		require.NoError(t, err)
	}))

	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodHead, "/items/42", nil))

	require.Equal(t, http.StatusOK, recorder.Code)
	require.Empty(t, recorder.Body.String())
	require.Equal(t, "4", recorder.Header().Get("Content-Length"))
}
