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

func genericEmptyResponseComponentBase(response *Response, status string) string {
	if response == nil || len(response.Content) != 0 || len(response.Headers) != 0 {
		return ""
	}
	text := strings.TrimSpace(http.StatusText(statusCodeValue(status)))
	if text == "" {
		return ""
	}
	return codegen.Goify(text, true) + "Response"
}

func responseComponentDescription(status string) string {
	if text := http.StatusText(statusCodeValue(status)); text != "" {
		return text + " response."
	}
	return fmt.Sprintf("HTTP %s response.", status)
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
	if schemaHashByName(base, schemas, cache, map[string]struct{}{}, responseAllocationHash) == schemaHashByName(name, schemas, cache, map[string]struct{}{}, responseAllocationHash) {
		return base
	}
	return name
}

func standardErrorResponseComponentBase(status string) string {
	code := statusCodeValue(status)
	if code < 400 || code > 599 {
		return ""
	}
	if text := http.StatusText(code); text != "" {
		return codegen.Goify(text, true) + "Error"
	}
	return "Status" + strings.TrimSpace(status) + "Error"
}
