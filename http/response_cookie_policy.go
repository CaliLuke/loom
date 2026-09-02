package http

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"strings"
)

type (
	// ResponseCookiePolicy applies deployment-specific attributes to a modeled
	// response cookie immediately before Loom writes it. Policies may set values
	// such as Domain, Secure, and Expires without moving cookie values out of the
	// generated response contract.
	ResponseCookiePolicy func(context.Context, *http.Cookie) error

	responseCookiePolicyContextKey struct{}
)

// Handler makes p available to modeled response cookies written by next.
// Apply it to one generated method handler or through the generated server's
// Use method. Handler panics when p is nil.
func (p ResponseCookiePolicy) Handler(next http.Handler) http.Handler {
	if p == nil {
		panic("loom: response cookie policy cannot be nil")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), responseCookiePolicyContextKey{}, p)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// SetResponseCookie applies the active runtime policy, validates the resulting
// cookie, and writes it to w. Generated response encoders call this function.
func SetResponseCookie(ctx context.Context, w http.ResponseWriter, cookie *http.Cookie) error {
	if cookie == nil {
		return fmt.Errorf("response cookie cannot be nil")
	}
	if ctx != nil {
		if policy, ok := ctx.Value(responseCookiePolicyContextKey{}).(ResponseCookiePolicy); ok && policy != nil {
			designed := *cookie
			if err := policy(ctx, cookie); err != nil {
				return fmt.Errorf("apply response cookie policy to %q: %w", designed.Name, err)
			}
			if responseCookieContractChanged(&designed, cookie) {
				return fmt.Errorf(
					"response cookie policy for %q may change only Domain, Secure, and Expires",
					designed.Name,
				)
			}
		}
	}
	if err := cookie.Valid(); err != nil {
		return fmt.Errorf("validate response cookie %q: %w", cookie.Name, err)
	}
	if err := validateRuntimeResponseCookieSecurity(cookie); err != nil {
		return err
	}
	http.SetCookie(w, cookie)
	return nil
}

func responseCookieContractChanged(before, after *http.Cookie) bool {
	return before.Name != after.Name ||
		before.Value != after.Value ||
		before.Quoted != after.Quoted ||
		before.Path != after.Path ||
		before.RawExpires != after.RawExpires ||
		before.MaxAge != after.MaxAge ||
		before.HttpOnly != after.HttpOnly ||
		before.SameSite != after.SameSite ||
		before.Partitioned != after.Partitioned ||
		before.Raw != after.Raw ||
		!slices.Equal(before.Unparsed, after.Unparsed)
}

func validateRuntimeResponseCookieSecurity(cookie *http.Cookie) error {
	if strings.HasPrefix(cookie.Name, "__Host-") {
		if cookie.Domain != "" {
			return fmt.Errorf("response cookie %q cannot set Domain because its name uses the %q prefix", cookie.Name, "__Host-")
		}
		if cookie.Path != "/" {
			return fmt.Errorf("response cookie %q requires Path=/ because its name uses the %q prefix", cookie.Name, "__Host-")
		}
	}
	if !cookie.Secure {
		switch {
		case strings.HasPrefix(cookie.Name, "__Host-"):
			return fmt.Errorf("response cookie %q requires Secure because its name uses the %q prefix", cookie.Name, "__Host-")
		case strings.HasPrefix(cookie.Name, "__Secure-"):
			return fmt.Errorf("response cookie %q requires Secure because its name uses the %q prefix", cookie.Name, "__Secure-")
		case cookie.SameSite == http.SameSiteNoneMode:
			return fmt.Errorf("response cookie %q requires Secure when SameSite is None", cookie.Name)
		}
	}
	return nil
}
