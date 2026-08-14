package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/expr/testdata"
)

func TestServiceExprMethod(t *testing.T) {
	var (
		methodFoo = &expr.MethodExpr{
			Name: "foo",
		}
		methodBar = &expr.MethodExpr{
			Name: "bar",
		}
	)
	cases := map[string]struct {
		name     string
		expected *expr.MethodExpr
	}{
		"exist": {
			name:     "foo",
			expected: methodFoo,
		},
		"not exist": {
			name:     "baz",
			expected: nil,
		},
	}

	for k, tc := range cases {
		s := expr.ServiceExpr{
			Methods: []*expr.MethodExpr{
				methodFoo,
				methodBar,
			},
		}
		if actual := s.Method(tc.name); actual != tc.expected {
			t.Errorf("%s: got %#v, expected %#v", k, actual, tc.expected)
		}
	}
}

func TestServiceRejectsDuplicateMethodNames(t *testing.T) {
	err := expr.RunInvalidDSL(t, func() {
		Service("duplicate methods", func() {
			Method("show", func() {})
			Method("show", func() {})
		})
	})
	if !strings.Contains(err.Error(), `method "show" defined twice`) {
		t.Fatalf("unexpected error:\n%s", err)
	}
}

func TestServiceExprError(t *testing.T) {
	var (
		errorFoo = &expr.ErrorExpr{
			Name: "foo",
		}
		errorBar = &expr.ErrorExpr{
			Name: "bar",
		}
	)
	cases := map[string]struct {
		name     string
		expected *expr.ErrorExpr
	}{
		"exist in service": {
			name:     "foo",
			expected: errorFoo,
		},
		"exist in root": {
			name:     "bar",
			expected: errorBar,
		},
		"not exist": {
			name:     "qux",
			expected: nil,
		},
	}

	expr.Root.Errors = []*expr.ErrorExpr{
		errorBar,
	}
	s := expr.ServiceExpr{
		Errors: []*expr.ErrorExpr{
			errorFoo,
		},
	}
	for k, tc := range cases {
		t.Run(k, func(t *testing.T) {
			if actual := s.Error(tc.name); actual != tc.expected {
				t.Errorf("got %#v, expected %#v", actual, tc.expected)
			}
		})
	}
}

func TestServiceExprValidate(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"service errors", testdata.ServiceErrorDSL, `attribute: error name "a" must be required in type "ServiceError"`},
	}

	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.DSL)
			assert.EqualError(t, err, tc.Error)
		})
	}
}

func TestMethodRejectsAmbiguousEffectiveErrorTypes(t *testing.T) {
	cases := map[string]struct {
		dsl   func()
		error string
	}{
		"service and method errors": {
			dsl: func() {
				exceptionResponse := Type("ExceptionResponse", func() {
					Attribute("error", String)
					Attribute("message", String)
				})

				Service("IncodeOmniAPI", func() {
					Error("unauthorized", exceptionResponse)
					Error("forbidden", exceptionResponse)

					Method("AddDeviceFingerprint", func() {
						Error("Status400", exceptionResponse)
					})
				})
			},
			error: `attribute: type "ExceptionResponse" is used to define multiple errors and must identify the attribute containing the error name with ErrorName`,
		},
		"service errors": {
			dsl: func() {
				sharedError := Type("SharedError", func() {
					Attribute("message", String)
				})

				Service("AmbiguousServiceErrors", func() {
					Error("not_found", sharedError)
					Error("conflict", sharedError)
					Method("Show", func() {})
				})
			},
			error: `attribute: type "SharedError" is used to define multiple errors and must identify the attribute containing the error name with ErrorName`,
		},
	}

	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			assert.EqualError(t, err, tc.error)
		})
	}
}

func TestErrorExprValidate(t *testing.T) {
	cases := []struct {
		Name  string
		DSL   func()
		Error string
	}{
		{"no error", testdata.ValidErrorsDSL, ""},
		{"invalid-struct-error-name-meta", testdata.InvalidStructErrorNameDSL,
			`attribute: error name "a" must be required in type "ServiceError"
attribute: duplicate error names in type "Error"
attribute: error name "a" must be a string in type "Error"
attribute: error name "a" must be required in type "Error"
attribute: type "ErrorType" is used to define multiple errors and must identify the attribute containing the error name with ErrorName`,
		},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			if tc.Error == "" {
				expr.RunDSL(t, tc.DSL)
			} else {
				err := expr.RunInvalidDSL(t, tc.DSL)
				assert.EqualError(t, err, tc.Error)
			}
		})
	}
}
