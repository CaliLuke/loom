package expr

import (
	"fmt"
	"sort"
	"strings"

	"goa.design/goa/v3/eval"
)

type (
	// HTTPResponseLinkExpr describes an OpenAPI response link.
	HTTPResponseLinkExpr struct {
		// Name is the published OpenAPI link key.
		Name string
		// Description is the optional human-readable link description.
		Description string
		// Operation is the target operation name. It may be either "method" for a
		// method in the current service or "service.method" for an explicit target.
		Operation string
		// OperationRef is the explicit OpenAPI operationRef target.
		OperationRef string
		// Parameters maps target parameter names to OpenAPI runtime expressions.
		Parameters map[string]string
		// RequestBody is the optional OpenAPI runtime expression used to build the
		// target request body.
		RequestBody string
	}
)

// EvalName returns the generic definition name used in error messages.
func (l *HTTPResponseLinkExpr) EvalName() string {
	if l == nil || l.Name == "" {
		return "HTTP response link"
	}
	return fmt.Sprintf("HTTP response link %#v", l.Name)
}

// Validate validates the response link definition.
func (l *HTTPResponseLinkExpr) Validate() *eval.ValidationErrors {
	verr := new(eval.ValidationErrors)
	if l == nil {
		return verr
	}
	if strings.TrimSpace(l.Name) == "" {
		verr.Add(l, "response link name cannot be empty")
	}
	if strings.TrimSpace(l.Operation) == "" && strings.TrimSpace(l.OperationRef) == "" {
		verr.Add(l, "response link must define either a target operation or operation ref")
	}
	if strings.TrimSpace(l.Operation) != "" && strings.TrimSpace(l.OperationRef) != "" {
		verr.Add(l, "response link cannot define both a target operation and operation ref")
	}
	for _, name := range orderedLinkParameterNames(l.Parameters) {
		if strings.TrimSpace(name) == "" {
			verr.Add(l, "response link parameter name cannot be empty")
			continue
		}
		if strings.TrimSpace(l.Parameters[name]) == "" {
			verr.Add(l, "response link parameter %q expression cannot be empty", name)
		}
	}
	return verr
}

func orderedLinkParameterNames(parameters map[string]string) []string {
	if len(parameters) == 0 {
		return nil
	}
	names := make([]string, 0, len(parameters))
	for name := range parameters {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
