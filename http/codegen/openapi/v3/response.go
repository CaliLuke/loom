package openapiv3

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"goa.design/goa/v3/eval"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
)

func headersFromAttr(attr *expr.MappedAttributeExpr, rand *expr.ExampleGenerator, closeObjects bool) map[string]*HeaderRef {
	o := expr.AsObject(attr.Type)
	if len(*o) == 0 {
		return nil
	}
	headers := make(map[string]*HeaderRef, len(*o))
	expr.WalkMappedAttr(attr, func(name, elem string, attr *expr.AttributeExpr) error { // nolint: errcheck
		header := &Header{
			Description: attr.Description,
			Required:    attr.IsRequiredNoDefault(name),
			Schema:      newSchemafier(rand, closeObjects).schemafy(attr),
			Example:     attr.Example(rand),
			Extensions:  openapi.ExtensionsFromExpr(attr.Meta),
		}
		initExamples(header, attr, rand, closeObjects)
		headers[elem] = &HeaderRef{Value: header}
		return nil
	})
	return headers
}

func responseFromExpr(r *expr.HTTPResponseExpr, bodies map[int][]*openapi.Schema, rand *expr.ExampleGenerator, closeObjects bool) *Response {
	ct := r.ContentType
	rt, ok := r.Body.Type.(*expr.ResultTypeExpr)
	if ok && ct == "" {
		ct = rt.ContentType
	}
	if ct == "" {
		// Default to application/json
		ct = "application/json"
	}
	headers := headersFromAttr(r.Headers, rand, closeObjects)
	if cookieHeader := responseCookieHeader(r.Cookies, rand); cookieHeader != nil {
		if headers == nil {
			headers = make(map[string]*HeaderRef)
		}
		headers["Set-Cookie"] = cookieHeader
	}

	var content map[string]*MediaType
	{
		if r.Body.Type != expr.Empty {
			content = make(map[string]*MediaType)
			content[ct] = &MediaType{
				Schema:     bodies[r.StatusCode][0],
				Extensions: openapi.ExtensionsFromExpr(r.Body.Meta),
			}
			initExamples(content[ct], r.Body, rand, closeObjects)
		} else if r.StatusCode != expr.StatusNoContent &&
			isSkipResponseBodyEncodeDecode(r.Parent) {
			// When SkipResponseBodyEncodeDecode is declared, the response type
			// is Empty, but the response code is not 204 and has content.
			content = make(map[string]*MediaType)
			content[ct] = &MediaType{
				Schema: &openapi.Schema{
					Type:   "string",
					Format: "binary",
				},
				Extensions: openapi.ExtensionsFromExpr(r.Body.Meta),
			}
		}
	}
	desc := r.Description
	if desc == "" {
		desc = fmt.Sprintf("%s response.", http.StatusText(r.StatusCode))
	}
	return &Response{
		Description: &desc,
		Headers:     headers,
		Content:     content,
		Extensions:  openapi.ExtensionsFromExpr(r.Meta),
	}
}

func responseCookieHeader(cookies []*expr.HTTPResponseCookieExpr, rand *expr.ExampleGenerator) *HeaderRef {
	if len(cookies) == 0 {
		return nil
	}
	header := &Header{
		Required: true,
		Schema: &openapi.Schema{
			Type: "string",
		},
	}
	if len(cookies) == 1 {
		cookie := cookies[0]
		header.Description = describeResponseCookie(cookie)
		header.Example = serializeResponseCookieExample(cookie, cookie.Attribute().Example(rand))
		return &HeaderRef{Value: header}
	}
	header.Description = describeResponseCookies(cookies)
	examples := make(map[string]*ExampleRef, len(cookies))
	for _, cookie := range cookies {
		examples[cookie.HTTPName()] = &ExampleRef{
			Value: &Example{
				Summary:     fmt.Sprintf("%s cookie", cookie.HTTPName()),
				Description: describeResponseCookie(cookie),
				Value:       serializeResponseCookieExample(cookie, cookie.Attribute().Example(rand)),
			},
		}
	}
	header.Examples = examples
	return &HeaderRef{Value: header}
}

func describeResponseCookie(cookie *expr.HTTPResponseCookieExpr) string {
	parts := []string{fmt.Sprintf("Sets the %q cookie.", cookie.HTTPName())}
	if attr := cookie.Attribute(); attr != nil && attr.Description != "" {
		parts = append(parts, attr.Description)
	}
	if policy := responseCookiePolicy(cookie); policy != "" {
		parts = append(parts, "Policy: "+policy+".")
	}
	return strings.Join(parts, " ")
}

func describeResponseCookies(cookies []*expr.HTTPResponseCookieExpr) string {
	lines := []string{"Set-Cookie headers issued by the server:"}
	for _, cookie := range cookies {
		lines = append(lines, "- "+describeResponseCookie(cookie))
	}
	return strings.Join(lines, "\n")
}

func responseCookiePolicy(cookie *expr.HTTPResponseCookieExpr) string {
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

func serializeResponseCookieExample(cookie *expr.HTTPResponseCookieExpr, value any) string {
	return buildResponseHTTPCookie(cookie, fmt.Sprintf("%v", value)).String()
}

func buildResponseHTTPCookie(cookie *expr.HTTPResponseCookieExpr, value string) *http.Cookie {
	httpCookie := &http.Cookie{
		Name:     cookie.HTTPName(),
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

func isSkipResponseBodyEncodeDecode(parent eval.Expression) bool {
	ee, ok := parent.(*expr.HTTPEndpointExpr)
	return ok && ee.SkipResponseBodyEncodeDecode
}
