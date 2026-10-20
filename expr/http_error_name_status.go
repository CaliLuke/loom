package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
)

// validateStatusErrorName binds the explicit reconstruction source and rejects
// competing wire sources. Endpoint admission separately checks status uniqueness.
func (e *HTTPErrorExpr) validateStatusErrorName(source *AttributeExpr, errors *eval.ValidationErrors) {
	r := e.Response
	r.StatusErrorName = ""
	if !r.ErrorNameFromStatus {
		return
	}
	key := TaggedAttribute(source, "struct:error:name")
	if IsDefaultErrorResult(source.Type) || key == "" {
		errors.Add(r, "ErrorNameFromStatus requires a custom error with an ErrorName field")
		return
	}
	_, field := source.FindAttribute(key)
	r.StatusErrorName = key
	if statusErrorNameMapped(r, key) {
		errors.Add(r, "ErrorNameFromStatus field %q cannot also appear in a response body, header or cookie", key)
	}
	constraints, err := EffectiveConstraintsFor(field)
	if err != nil {
		errors.Add(r, "ErrorNameFromStatus field %q: %s", key, err)
		return
	}
	normalized := *field
	normalized.Validation = constraints.Validation().Lowered()
	if !exampleMatchesValidation(&normalized, e.Name) {
		errors.Add(r, "ErrorNameFromStatus value %q does not satisfy ErrorName field %q constraints", e.Name, key)
	}
}

func statusErrorNameMapped(r *HTTPResponseExpr, key string) bool {
	logical := AttributeName(key)
	if r.Headers != nil {
		for _, field := range *AsObject(r.Headers.Type) {
			if AttributeName(field.Name) == logical || strings.EqualFold(r.Headers.ElemName(field.Name), "loom-error") {
				return true
			}
		}
	}
	for _, cookie := range r.Cookies {
		if cookie.AttributeName() == logical {
			return true
		}
	}
	for _, body := range []*AttributeExpr{r.Body, r.OpenAPIBody} {
		if body == nil {
			continue
		}
		if origin, ok := body.Meta.Last("origin:attribute"); ok {
			if AttributeName(origin) == logical {
				return true
			}
			continue
		}
		if object := AsObject(body.Type); object != nil {
			if _, field := objectAttribute(object, key); field != nil {
				return true
			}
		}
	}
	return false
}

func (e *HTTPEndpointExpr) validateStatusErrorNames(errors *eval.ValidationErrors) {
	counts := make(map[int]int)
	for _, response := range e.Responses {
		counts[response.StatusCode]++
	}
	for _, failure := range e.HTTPErrors {
		counts[failure.Response.StatusCode]++
	}
	for _, failure := range e.HTTPErrors {
		response := failure.Response
		if !response.ErrorNameFromStatus {
			continue
		}
		if e.IsJSONRPC() {
			errors.Add(response, "ErrorNameFromStatus is HTTP only")
		}
		if counts[response.StatusCode] != 1 {
			errors.Add(response, "ErrorNameFromStatus requires a unique response status; status %d selects multiple responses", response.StatusCode)
		}
	}
}
