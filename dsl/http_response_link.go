package dsl

import (
	"github.com/CaliLuke/loom/v3/eval"
	"github.com/CaliLuke/loom/v3/expr"
)

// Link declares an OpenAPI response link on the enclosing HTTP response.
//
// Link must appear in an HTTP Response expression.
//
// The target operation is configured with LinkOperation or LinkOperationRef and
// request mappings are configured with LinkParam and LinkRequestBody.
func Link(name string, fn func()) {
	response, ok := eval.Current().(*expr.HTTPResponseExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	link := &expr.HTTPResponseLinkExpr{Name: name}
	response.Links = append(response.Links, link)
	if fn != nil {
		eval.Execute(fn, link)
	}
}

// LinkOperation sets the target operation for the enclosing response link.
//
// LinkOperation must appear in a Link expression.
func LinkOperation(name string) {
	link, ok := eval.Current().(*expr.HTTPResponseLinkExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	link.Operation = name
}

// LinkOperationRef sets the explicit OpenAPI operationRef for the enclosing
// response link.
//
// LinkOperationRef must appear in a Link expression.
func LinkOperationRef(ref string) {
	link, ok := eval.Current().(*expr.HTTPResponseLinkExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	link.OperationRef = ref
}

// LinkParam maps a target operation parameter to an OpenAPI runtime expression.
//
// LinkParam must appear in a Link expression.
func LinkParam(name string, expression string) {
	link, ok := eval.Current().(*expr.HTTPResponseLinkExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if link.Parameters == nil {
		link.Parameters = make(map[string]string)
	}
	link.Parameters[name] = expression
}

// LinkRequestBody sets the OpenAPI runtime expression used to build the target
// operation request body.
//
// LinkRequestBody must appear in a Link expression.
func LinkRequestBody(expression string) {
	link, ok := eval.Current().(*expr.HTTPResponseLinkExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	link.RequestBody = expression
}
