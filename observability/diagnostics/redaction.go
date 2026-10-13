// Package diagnostics owns credential masking for development transport previews.
// It operates on already captured data and never reads application bodies.
package diagnostics

import (
	"fmt"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"unicode"
)

type (
	// Fields identifies authored credential locations in the actual wire format.
	// Diagnostic constructors copy these slices; later caller changes have no effect.
	Fields struct {
		// Headers lists credential header names, compared without case sensitivity.
		Headers []string
		// Query lists credential query parameter names.
		Query []string
		// JSON lists credential JSON member names, masked at any nesting depth.
		JSON []string
		// Form lists credential URL-encoded form field names.
		Form []string
		// Body omits the entire body, for example when the body is a scalar credential.
		Body bool
		// Path omits the URL path and all separately printed route variables.
		Path bool
	}

	// Policy is an immutable union of authored credential locations and conventional
	// credential names. A zero Policy applies conventional masking only.
	Policy struct {
		headers map[string]bool
		query   map[string]bool
		json    map[string]bool
		form    map[string]bool
		body    bool
		path    bool
	}
)

const (
	// BodyCaptureLimit bounds development body previews without limiting wire bytes.
	BodyCaptureLimit = 64 << 10
	redactedValue    = "[REDACTED]"
)

// New copies and merges fields into an immutable policy. The same physical name
// is masked across all observed methods, even if only one method declares it secret.
func New(fields ...Fields) *Policy {
	policy := &Policy{headers: make(map[string]bool), query: make(map[string]bool), json: make(map[string]bool), form: make(map[string]bool)}
	for _, field := range fields {
		for _, name := range field.Headers {
			policy.headers[strings.ToLower(name)] = true
		}
		for _, name := range field.Query {
			policy.query[name] = true
		}
		for _, name := range field.JSON {
			policy.json[name] = true
		}
		for _, name := range field.Form {
			policy.form[name] = true
		}
		policy.body = policy.body || field.Body
		policy.path = policy.path || field.Path
	}
	return policy
}

// Headers returns a masked copy without changing the application's headers.
func (p *Policy) Headers(headers http.Header) http.Header {
	result := headers.Clone()
	for name := range result {
		if sensitiveName(name) || p.headers[strings.ToLower(name)] {
			result[name] = []string{redactedValue}
		}
	}
	return result
}

// URL returns a masked copy. An authored credential path also clears RawPath;
// credentials must not remain in an alternate escaped or opaque URL rendering.
func (p *Policy) URL(original *url.URL) *url.URL {
	if original == nil {
		return nil
	}
	result := original.Clone()
	if result.User != nil {
		result.User = url.User(redactedValue)
	}
	query, err := url.ParseQuery(result.RawQuery)
	if err != nil {
		result.RawQuery = "[invalid query omitted]"
	} else {
		for name := range query {
			if sensitiveName(name) || p.query[name] {
				query[name] = []string{redactedValue}
			}
		}
		result.RawQuery = query.Encode()
	}
	if p.path {
		result.Path, result.RawPath, result.Opaque = "/"+redactedValue, "", ""
	}
	return result
}

// PathValues returns a masked copy of separately captured route variables.
// A credential-bearing path omits every route value, matching whole-path omission.
func (p *Policy) PathValues(values map[string]string) map[string]string {
	result := make(map[string]string, len(values))
	for name, value := range values {
		if p.path || sensitiveName(name) {
			value = redactedValue
		}
		result[name] = value
	}
	return result
}

// Body returns a safe preview of captured bytes. It never reads or modifies the
// application body. Omitted, malformed or incomplete structured captures are
// represented by an omission marker. Known body credentials with an unsupported
// content type cause whole-body omission; ordinary opaque bodies remain visible.
func (p *Policy) Body(contentType string, data []byte, omitted, complete bool) []byte {
	if omitted {
		return fmt.Appendf(nil, "[body omitted after %d bytes]", BodyCaptureLimit)
	}
	if len(data) == 0 {
		return nil
	}
	if p.body {
		return []byte("[credential body omitted]")
	}
	mediaType, _, err := mime.ParseMediaType(contentType)
	isJSON := err == nil && (mediaType == "application/json" || strings.HasSuffix(mediaType, "+json"))
	isForm := err == nil && mediaType == "application/x-www-form-urlencoded"
	if !complete && (isJSON || isForm) {
		return []byte("[incomplete structured body omitted]")
	}
	switch {
	case isJSON:
		return p.jsonBody(data)
	case isForm:
		return p.formBody(data)
	case len(p.json) > 0 || len(p.form) > 0:
		return []byte("[credential body with unsupported content type omitted]")
	default:
		return append([]byte(nil), data...)
	}
}

func sensitiveName(name string) bool {
	// Normalization makes apiKey, api_key, API-KEY and spaced spellings agree.
	var normalized strings.Builder
	for _, char := range name {
		if unicode.IsLetter(char) || unicode.IsDigit(char) {
			normalized.WriteRune(unicode.ToLower(char))
		}
	}
	value := normalized.String()
	switch value {
	case "authorization", "cookie", "setcookie", "proxyauthorization", "password", "passwd", "secret", "token", "key", "apikey", "xapikey", "accesstoken", "refreshtoken", "clientsecret", "credential", "credentials", "session", "sessionid":
		return true
	}
	return strings.HasSuffix(value, "token") || strings.HasSuffix(value, "secret") || strings.HasSuffix(value, "apikey")
}

func (p *Policy) formBody(data []byte) []byte {
	values, err := url.ParseQuery(string(data))
	if err != nil {
		return []byte("[invalid form body omitted]")
	}
	for name := range values {
		if sensitiveName(name) || p.form[name] {
			values[name] = []string{redactedValue}
		}
	}
	return []byte(values.Encode())
}
