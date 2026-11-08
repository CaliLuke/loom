package expr

import "fmt"

// HTTPInformationalExpr declares a named, independently typed interim response.
type HTTPInformationalExpr struct {
	// Name identifies the emitter and observer within its endpoint.
	Name string
	// HTTPResponseExpr retains the selector, headers and status source.
	*HTTPResponseExpr
	// Value is the native informational value type.
	Value *AttributeExpr
}

// EvalName identifies this informational declaration in DSL diagnostics.
func (i *HTTPInformationalExpr) EvalName() string {
	return fmt.Sprintf("HTTP Informational %q", i.Name)
}
