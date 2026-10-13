package service

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// errorConstructorKey identifies an emitted factory by its service-local wire
// error name, independently of nominal type identities in the same scope.
type errorConstructorKey string

func allocateErrorConstructors(service *expr.ServiceExpr, scope *codegen.NameScope) {
	for _, er := range service.Errors {
		errorConstructorName(er, scope)
	}
	for _, method := range service.Methods {
		for _, er := range method.Errors {
			errorConstructorName(er, scope)
		}
	}
}

func errorConstructorName(er *expr.ErrorExpr, scope *codegen.NameScope) string {
	if !expr.IsDefaultErrorResult(er.Type) {
		return ""
	}
	return scope.HashedUnique(errorConstructorKey(er.Name), "Make"+codegen.Goify(er.Name, true))
}

// Hash distinguishes factory identities from nominal and service scope keys.
func (k errorConstructorKey) Hash() string {
	return "_error_constructor_+" + string(k)
}
