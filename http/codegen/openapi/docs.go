package openapi

import "github.com/CaliLuke/loom/expr"

// ExternalDocs represents an OpenAPI External Documentation object as defined in
// https://github.com/OAI/OpenAPI-Specification/blob/master/versions/3.0.3.md#externalDocumentationObject
type ExternalDocs struct {
	Description string `json:"description,omitzero"`
	URL         string `json:"url,omitzero"`
}

// DocsFromExpr builds a ExternalDocs from the Loom docs expression.
// The OpenAPI External Documentation object carries no Loom-owned extension
// scope, so extension metadata belongs to the object that declares it.
func DocsFromExpr(docs *expr.DocsExpr) *ExternalDocs {
	if docs == nil {
		return nil
	}
	return &ExternalDocs{
		Description: docs.Description,
		URL:         docs.URL,
	}
}
