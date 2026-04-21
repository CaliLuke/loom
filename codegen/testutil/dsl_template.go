package testutil

import (
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

// TestAPI describes a minimal design used to synthesize a DSL function for
// tests. It covers the common "one service, a handful of endpoints" shape
// used by most codegen test inputs. For cases that exercise transport-
// specific concerns (HTTP routes, gRPC rpc:tag, JSON-RPC method names)
// callers should still hand-write the DSL — this helper keeps the default
// cases terse and skippable.
//
// Typical use:
//
//	fn := testutil.TestAPI{
//	    Name: "svc",
//	    Services: []testutil.TestService{{
//	        Name: "Foo",
//	        Methods: []testutil.TestMethod{{
//	            Name:    "GetUser",
//	            Payload: testutil.TestType{Name: "Request", Fields: []testutil.TestField{{Name: "id", Type: expr.String, Required: true}}},
//	            Result:  testutil.TestType{Name: "User"},
//	        }},
//	    }},
//	}.DSL()
//	root := codegen.RunDSL(t, fn)
type TestAPI struct {
	// Name is the API name. Defaults to "test".
	Name string
	// Services enumerated in the API.
	Services []TestService
}

// TestService describes one service within a [TestAPI].
type TestService struct {
	// Name is the service name.
	Name string
	// Methods enumerated on this service.
	Methods []TestMethod
	// Errors are service-level error names.
	Errors []string
}

// TestMethod describes one method within a [TestService].
type TestMethod struct {
	// Name is the method name.
	Name string
	// Payload is the request shape. Leave zero for a method with no payload.
	Payload TestType
	// Result is the response shape. Leave zero for a method with no result.
	Result TestType
	// Errors are method-level error names.
	Errors []string
}

// TestType describes a user-type used as a payload or result. The zero
// value means "no type".
type TestType struct {
	// Name is the type name. Empty means no type.
	Name string
	// Fields enumerated on the type. May be empty for a scalar-like or empty
	// type (the DSL will emit an empty object).
	Fields []TestField
}

// TestField describes one attribute within a [TestType].
type TestField struct {
	// Name is the attribute name.
	Name string
	// Type is the DataType of the attribute (e.g., expr.String, expr.Int).
	Type expr.DataType
	// Required reports whether the attribute should be marked Required.
	Required bool
	// Description is attached as the attribute description.
	Description string
}

// DSL returns a func suitable for passing to eval.RunDSL (or one of the
// test-helper wrappers built on top of it). It closes over the [TestAPI]
// receiver, so multiple calls produce independent DSL funcs.
func (a TestAPI) DSL() func() {
	apiName := a.Name
	if apiName == "" {
		apiName = "test"
	}
	return func() {
		dsl.API(apiName, func() {})
		for _, svc := range a.Services {
			dsl.Service(svc.Name, func() {
				for _, errName := range svc.Errors {
					dsl.Error(errName)
				}
				for _, m := range svc.Methods {
					dsl.Method(m.Name, func() {
						if m.Payload.Name != "" {
							p := m.Payload
							dsl.Payload(p.Name, func() {
								emitFields(p.Fields)
							})
						}
						if m.Result.Name != "" {
							r := m.Result
							dsl.Result(r.Name, func() {
								emitFields(r.Fields)
							})
						}
						for _, errName := range m.Errors {
							dsl.Error(errName)
						}
					})
				}
			})
		}
	}
}

func emitFields(fields []TestField) {
	var required []string
	for _, f := range fields {
		dsl.Attribute(f.Name, f.Type, func() {
			if f.Description != "" {
				dsl.Description(f.Description)
			}
		})
		if f.Required {
			required = append(required, f.Name)
		}
	}
	if len(required) > 0 {
		dsl.Required(required...)
	}
}
