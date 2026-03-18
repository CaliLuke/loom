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

func headersFromAttr(attr *expr.MappedAttributeExpr, rand *expr.ExampleGenerator) map[string]*HeaderRef {
	o := expr.AsObject(attr.Type)
	if len(*o) == 0 {
		return nil
	}
	headers := make(map[string]*HeaderRef, len(*o))
	expr.WalkMappedAttr(attr, func(name, elem string, attr *expr.AttributeExpr) error { // nolint: errcheck
		header := &Header{
			Description: attr.Description,
			Required:    attr.IsRequiredNoDefault(name),
			Schema:      newSchemafier(rand).schemafy(attr),
			Example:     attr.Example(rand),
			Extensions:  openapi.ExtensionsFromExpr(attr.Meta),
		}
		initExamples(header, attr, rand)
		headers[elem] = &HeaderRef{Value: header}
		return nil
	})
	return headers
}

func responseFromExpr(r *expr.HTTPResponseExpr, bodies map[int][]*openapi.Schema, rand *expr.ExampleGenerator) *Response {
	ct := r.ContentType
	rt, ok := r.Body.Type.(*expr.ResultTypeExpr)
	if ok && ct == "" {
		ct = rt.ContentType
	}
	if ct == "" {
		// Default to application/json
		ct = "application/json"
	}
	headers := headersFromAttr(r.Headers, rand)
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
			initExamples(content[ct], r.Body, rand)
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
	if cookie.Path != "" {
		parts = append(parts, "Path="+cookie.Path)
	}
	if cookie.Domain != "" {
		parts = append(parts, "Domain="+cookie.Domain)
	}
	if cookie.MaxAge != "" {
		parts = append(parts, "Max-Age="+cookie.MaxAge)
	}
	if cookie.Secure {
		parts = append(parts, "Secure")
	}
	if cookie.HTTPOnly {
		parts = append(parts, "HttpOnly")
	}
	if cookie.SameSite != "" {
		parts = append(parts, "SameSite="+titleCase(string(cookie.SameSite)))
	}
	return strings.Join(parts, "; ")
}

func serializeResponseCookieExample(cookie *expr.HTTPResponseCookieExpr, value any) string {
	httpCookie := &http.Cookie{
		Name:     cookie.HTTPName(),
		Value:    fmt.Sprintf("%v", value),
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
	return httpCookie.String()
}

func titleCase(val string) string {
	if val == "" {
		return ""
	}
	return strings.ToUpper(val[:1]) + val[1:]
}

func isSkipResponseBodyEncodeDecode(parent eval.Expression) bool {
	ee, ok := parent.(*expr.HTTPEndpointExpr)
	return ok && ee.SkipResponseBodyEncodeDecode
}
