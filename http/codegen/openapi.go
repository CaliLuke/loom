package codegen

import (
	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	openapiv3 "goa.design/goa/v3/http/codegen/openapi/v3"
)

// OpenAPIFiles returns the files for the OpenAPIFile spec of the given HTTP API.
func OpenAPIFiles(root *expr.RootExpr) ([]*codegen.File, error) {
	// Only create a OpenAPI specification if there are HTTP services.
	if len(root.API.HTTP.Services) == 0 {
		return nil, nil
	}

	files, err := openapiv3.Files(root)
	if err != nil {
		return nil, err
	}
	return files, nil
}
