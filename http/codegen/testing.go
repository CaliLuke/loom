package codegen

import (
	"testing"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// RunHTTPDSL returns the HTTP DSL root resulting from running the given DSL.
func RunHTTPDSL(t *testing.T, dsl func()) *expr.RootExpr {
	// reset all roots and codegen data structures
	root := expr.RunDSL(t, dsl)
	return root
}

// CreateHTTPServices creates a new ServicesData instance for testing.
func CreateHTTPServices(root *expr.RootExpr) *ServicesData {
	return NewServicesData(service.NewServicesData(root), root.API.HTTP)
}
