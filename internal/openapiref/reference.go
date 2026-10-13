// Package openapiref resolves local OpenAPI component reference identity.
package openapiref

import (
	"encoding/json/jsontext"
	"fmt"
	"net/url"
	"strings"
)

// Name resolves a fragment-only URI reference under a component prefix such as
// "#/components/schemas/". It decodes URI escapes once before JSON Pointer
// escapes and requires exactly one nonempty final token. Component-name admission
// is separate: valid pointer tokens need not be valid OpenAPI component keys.
func Name(ref, prefix string) (string, error) {
	if !strings.HasPrefix(ref, "#") || strings.Count(ref, "#") != 1 {
		return "", fmt.Errorf("has the wrong kind")
	}
	fragment, err := url.PathUnescape(ref[1:])
	if err != nil {
		return "", fmt.Errorf("has invalid URI fragment escaping: %w", err)
	}
	segment := strings.TrimPrefix(fragment, strings.TrimPrefix(prefix, "#"))
	if segment == fragment || segment == "" || strings.Contains(segment, "/") {
		return "", fmt.Errorf("has the wrong kind")
	}
	pointer := jsontext.Pointer("/" + segment)
	if !pointer.IsValid() {
		return "", fmt.Errorf("has invalid JSON Pointer escaping")
	}
	return pointer.LastToken(), nil
}
