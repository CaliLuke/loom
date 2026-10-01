package ir

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func responseCookieHeader(
	cookies []*transportir.Cookie,
) *Header {
	if len(cookies) == 0 {
		return nil
	}
	header := &Header{
		Required: true,
		Schema: &Schema{
			Type: "string",
		},
	}
	if len(cookies) == 1 {
		cookie := cookies[0]
		header.Description = describeResponseCookie(cookie)
		if example, ok := responseCookieExample(cookie); ok {
			header.Example = example
		}
		return header
	}
	header.Description = describeResponseCookies(cookies)
	examples := make(map[string]*ExampleRef, len(cookies))
	for _, cookie := range cookies {
		example, ok := responseCookieExample(cookie)
		if !ok {
			continue
		}
		examples[cookie.HTTPName] = &ExampleRef{Value: &Example{
			Summary:     fmt.Sprintf("%s cookie", cookie.HTTPName),
			Description: describeResponseCookie(cookie),
			Value:       example,
		}}
	}
	if len(examples) > 0 {
		header.Examples = examples
	}
	return header
}

func responseCookieExample(cookie *transportir.Cookie) (string, bool) {
	if cookie == nil || cookie.Attribute == nil {
		return "", false
	}
	if disabled, ok := cookie.Attribute.Meta.Last("openapi:example"); ok && disabled == "false" {
		return "", false
	}
	requirePreparedExampleTarget(cookie.Value)
	if cookie.Value.Representative == nil {
		return "", false
	}
	value, ok := preparedDeclaredExampleValue(*cookie.Value.Representative)
	if !ok {
		return "", false
	}
	return serializeResponseCookieExample(cookie, value), true
}

func describeResponseCookie(cookie *transportir.Cookie) string {
	parts := []string{fmt.Sprintf("Sets the %q cookie.", cookie.HTTPName)}
	if attr := cookie.Attribute; attr != nil && attr.Description != "" {
		parts = append(parts, attr.Description)
	}
	if policy := responseCookiePolicy(cookie); policy != "" {
		parts = append(parts, "Policy: "+policy+".")
	}
	return strings.Join(parts, " ")
}

func describeResponseCookies(cookies []*transportir.Cookie) string {
	lines := make([]string, 0, 1+len(cookies))
	lines = append(lines, "Set-Cookie headers issued by the server:")
	for _, cookie := range cookies {
		lines = append(lines, "- "+describeResponseCookie(cookie))
	}
	return strings.Join(lines, "\n")
}

func responseCookiePolicy(cookie *transportir.Cookie) string {
	parts := make([]string, 0, 6)
	httpCookie := buildResponseHTTPCookie(cookie, "")
	if httpCookie.Path != "" {
		parts = append(parts, "Path="+httpCookie.Path)
	}
	if httpCookie.Domain != "" {
		parts = append(parts, "Domain="+httpCookie.Domain)
	}
	if cookie.MaxAge != "" {
		parts = append(parts, "Max-Age="+strconv.Itoa(normalizeCookieMaxAge(httpCookie.MaxAge)))
	}
	if httpCookie.Secure {
		parts = append(parts, "Secure")
	}
	if httpCookie.HttpOnly {
		parts = append(parts, "HttpOnly")
	}
	if sameSite := sameSiteString(httpCookie.SameSite); sameSite != "" {
		parts = append(parts, "SameSite="+sameSite)
	}
	return strings.Join(parts, "; ")
}

func serializeResponseCookieExample(cookie *transportir.Cookie, value any) string {
	formatted := fmt.Sprintf("%v", value)
	if bytes, ok := value.([]byte); ok {
		formatted = string(bytes)
	}
	return buildResponseHTTPCookie(cookie, formatted).String()
}

func buildResponseHTTPCookie(cookie *transportir.Cookie, value string) *http.Cookie {
	httpCookie := &http.Cookie{
		Name:     cookie.HTTPName,
		Value:    value,
		Path:     cookie.Path,
		Domain:   cookie.Domain,
		Secure:   cookie.Secure,
		HttpOnly: cookie.HTTPOnly,
	}
	if cookie.MaxAge != "" {
		if maxAge, err := strconv.Atoi(cookie.MaxAge); err == nil {
			httpCookie.MaxAge = maxAge
		}
	}
	switch cookie.SameSite {
	case expr.CookieSameSiteLax:
		httpCookie.SameSite = http.SameSiteLaxMode
	case expr.CookieSameSiteStrict:
		httpCookie.SameSite = http.SameSiteStrictMode
	case expr.CookieSameSiteNone:
		httpCookie.SameSite = http.SameSiteNoneMode
	case expr.CookieSameSiteDefault:
		httpCookie.SameSite = http.SameSiteDefaultMode
	}
	return httpCookie
}

func sameSiteString(mode http.SameSite) string {
	switch mode {
	case http.SameSiteDefaultMode:
		return "Default"
	case http.SameSiteLaxMode:
		return "Lax"
	case http.SameSiteStrictMode:
		return "Strict"
	case http.SameSiteNoneMode:
		return "None"
	default:
		return ""
	}
}

func normalizeCookieMaxAge(maxAge int) int {
	if maxAge < 0 {
		return 0
	}
	return maxAge
}
