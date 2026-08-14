package openapi

import (
	"encoding/json"
	"strings"

	"github.com/CaliLuke/loom/expr"
)

// ExtensionsFromExpr generates openapi extensions from the given meta
// expression.
func ExtensionsFromExpr(mdata expr.MetaExpr) map[string]any {
	return extensionsFromExprWithPrefix(mdata, "openapi:extension:")
}

// ScopedExtensionsFromExpr returns OpenAPI extensions declared for scope.
// Scoped extension metadata uses openapi:<scope>:extension:<x-name> keys.
func ScopedExtensionsFromExpr(mdata expr.MetaExpr, scope string) map[string]any {
	return extensionsFromExprWithPrefix(mdata, "openapi:"+scope+":extension:")
}

// MergeExtensions combines extension maps, with later maps taking precedence.
func MergeExtensions(extensionSets ...map[string]any) map[string]any {
	var merged map[string]any
	for _, extensions := range extensionSets {
		for name, value := range extensions {
			if merged == nil {
				merged = make(map[string]any)
			}
			merged[name] = value
		}
	}
	return merged
}

// extensionsFromExprWithPrefix generates openapi extensions from
// the given meta expression with keys starting the given prefix.
func extensionsFromExprWithPrefix(mdata expr.MetaExpr, prefix string) map[string]any {
	if !strings.HasSuffix(prefix, ":") {
		prefix += ":"
	}
	extensions := make(map[string]any)
	for key, value := range mdata {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		name := key[len(prefix):]
		if strings.Contains(name, ":") {
			continue
		}
		if !strings.HasPrefix(name, "x-") {
			continue
		}
		val := value[0]
		ival := any(val)
		if err := json.Unmarshal([]byte(val), &ival); err != nil {
			extensions[name] = val
			continue
		}
		extensions[name] = ival
	}
	if len(extensions) == 0 {
		return nil
	}
	return extensions
}
