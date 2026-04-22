package ir

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func responseComponentBase(operationID, status string) string {
	if base := standardErrorResponseComponentBase(status); base != "" {
		return base
	}
	return componentNameFromOperation(operationID) + responseStatusComponentSuffix(status) + "Response"
}

func requestBodyComponentBase(usage requestBodyUsage, schemas map[string]*Schema) string {
	if usage.ref != nil && usage.ref.Value != nil && strings.TrimSpace(usage.ref.Value.ComponentName) != "" {
		return strings.TrimSpace(usage.ref.Value.ComponentName)
	}
	if base := reusableRequestBodyComponentBase(usage.ref, schemas); base != "" {
		return base
	}
	return usage.base
}

func reusableRequestBodyComponentBase(ref *RequestBodyRef, schemas map[string]*Schema) string {
	if ref == nil || ref.Value == nil || len(ref.Value.Content) != 1 {
		return ""
	}
	contentType := orderedStringKeys(ref.Value.Content)[0]
	mediaType := ref.Value.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	if !ok {
		return ""
	}
	suffix := mediaTypeComponentSuffix(contentType)
	if strings.HasSuffix(schemaName, "RequestBody") {
		if suffix == "" {
			return schemaName
		}
		return schemaName + suffix
	}
	return schemaName + suffix + "RequestBody"
}

func reusableResponseComponentBase(ref *ResponseRef, status string, schemas map[string]*Schema) string {
	if ref == nil || ref.Value == nil {
		return ""
	}
	if strings.TrimSpace(ref.Value.ComponentName) != "" {
		return strings.TrimSpace(ref.Value.ComponentName)
	}
	if base := reusableErrorResponseComponentBase(ref.Value, status, schemas); base != "" {
		return base
	}
	if base := standardErrorResponseComponentBase(status); base != "" {
		return base
	}
	if base := genericEmptyResponseComponentBase(ref.Value, status); base != "" {
		return base
	}
	if len(ref.Value.Content) != 1 {
		return ""
	}
	contentType := orderedStringKeys(ref.Value.Content)[0]
	mediaType := ref.Value.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	if !ok {
		return ""
	}
	return schemaName + mediaTypeComponentSuffix(contentType) + responseStatusComponentSuffix(status) + "Response"
}

func reusableErrorResponseComponentBase(response *Response, status string, schemas map[string]*Schema) string {
	if response == nil || !isErrorSchemaResponse(response, schemas) {
		return ""
	}
	if code := responseErrorCode(response.Description); code != "" {
		if generic := genericErrorCodeForStatus(status); generic != "" && code == generic {
			return standardErrorResponseComponentBase(status)
		}
		return semanticErrorComponentBase(code)
	}
	return ""
}

func isErrorSchemaResponse(response *Response, schemas map[string]*Schema) bool {
	if response == nil || len(response.Content) != 1 {
		return false
	}
	contentType := orderedStringKeys(response.Content)[0]
	mediaType := response.Content[contentType]
	schemaName, ok := componentSchemaNameFromMediaType(mediaType, schemas)
	return ok && schemaName == "Error"
}

func responseErrorCode(description string) string {
	desc := strings.TrimSpace(description)
	if desc == "" {
		return ""
	}
	code, _, ok := strings.Cut(desc, ":")
	if !ok {
		return ""
	}
	code = strings.TrimSpace(code)
	if code == "" {
		return ""
	}
	for _, r := range code {
		if !(r == '_' || r == '-' || (r >= '0' && r <= '9') || (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z')) {
			return ""
		}
	}
	return code
}

func semanticErrorComponentBase(code string) string {
	base := codegen.Goify(code, true)
	if base == "" {
		return "Error"
	}
	if strings.HasSuffix(base, "Error") {
		return base
	}
	return base + "Error"
}

func genericEmptyResponseComponentBase(response *Response, status string) string {
	if response == nil || len(response.Content) != 0 || len(response.Headers) != 0 {
		return ""
	}
	if !isDefaultResponseDescription(response.Description, status) {
		return ""
	}
	text := strings.TrimSpace(http.StatusText(statusCodeValue(status)))
	if text == "" {
		return ""
	}
	return codegen.Goify(text, true) + "Response"
}

func isDefaultResponseDescription(description, status string) bool {
	code := statusCodeValue(status)
	if code == 0 {
		return false
	}
	return description == fmt.Sprintf("%s response.", http.StatusText(code))
}

func statusCodeValue(status string) int {
	trimmed := strings.TrimSpace(status)
	if !isDigits(trimmed) {
		return 0
	}
	code, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0
	}
	return code
}

func componentSchemaNameFromMediaType(mediaType *MediaType, schemas map[string]*Schema) (string, bool) {
	if mediaType == nil || mediaType.Schema == nil {
		return "", false
	}
	name, ok := schemaComponentName(mediaType.Schema.Ref)
	if !ok {
		return "", false
	}
	return canonicalComponentSchemaName(name, schemas), true
}

func canonicalComponentSchemaName(name string, schemas map[string]*Schema) string {
	base, ok := duplicateAliasBase(name)
	if !ok || schemas[base] == nil || schemas[name] == nil {
		return name
	}
	cache := map[string]string{}
	if schemaHashByName(base, schemas, cache, map[string]struct{}{}) == schemaHashByName(name, schemas, cache, map[string]struct{}{}) {
		return base
	}
	return name
}

func standardErrorResponseComponentBase(status string) string {
	switch strings.TrimSpace(status) {
	case "400":
		return "BadRequestError"
	case "401":
		return "UnauthorizedError"
	case "403":
		return "ForbiddenError"
	case "404":
		return "NotFoundError"
	case "409":
		return "ConflictError"
	case "422":
		return "UnprocessableEntityError"
	case "429":
		return "TooManyRequestsError"
	default:
		return ""
	}
}

func genericErrorCodeForStatus(status string) string {
	switch strings.TrimSpace(status) {
	case "400":
		return "bad_request"
	case "401":
		return "unauthorized"
	case "403":
		return "forbidden"
	case "404":
		return "not_found"
	case "409":
		return "conflict"
	case "422":
		return "unprocessable_entity"
	case "429":
		return "rate_limited"
	default:
		return ""
	}
}
