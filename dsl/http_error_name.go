package dsl

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

// ErrorNameFromStatus reconstructs a custom error's required ErrorName field
// from the declared error name when the HTTP client selects its response status.
// It may appear only inside an HTTP error Response. That status must identify
// exactly one response on every endpoint that inherits the mapping.
//
// The field is omitted from the implicit body and must not also be mapped to an
// explicit body, header or cookie. The server does not emit Loom-Error for this
// response. Existing header mappings do not acquire a missing-header fallback.
//
//	Response("not_found", StatusNotFound, func() {
//	    ErrorNameFromStatus()
//	})
func ErrorNameFromStatus() {
	response, ok := eval.Current().(*expr.HTTPResponseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if response.ErrorName == "" {
		eval.ReportError("ErrorNameFromStatus must appear in an HTTP error Response")
		return
	}
	response.ErrorNameFromStatus = true
}
