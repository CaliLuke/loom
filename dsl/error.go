package dsl

import (
	"slices"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
	pkg "github.com/CaliLuke/loom/pkg"
)

const (
	// The constants below make it possible for the service specific code to
	// return error names that are consistent with the names used by the
	// generated request and response payload validation code.
	//
	// Usage:
	//
	// var _ = Service("divider", func() {
	//     Error(MissingField)
	//     Error(InvalidRange)
	//
	//     Payload(func() {
	//        Attribute("numerator", Int)
	//        Attribute("denominator", Int, func() {
	//            Minimum(1)
	//        })
	//        Required("numerator", "denominator")
	//     })
	//
	//     HTTP(func() {
	//        Response(MissingField, StatusBadRequest)
	//        Response(InvalidRange, StatusBadRequest)
	//     })
	//
	//     GRPC(func() {
	//         Response(MissingField, CodeInvalidArgument)
	//         Response(InvalidRange, CodeInvalidArgument)
	//     })
	// })

	// InvalidFieldType is the error name for invalid field type errors.
	InvalidFieldType = pkg.InvalidFieldType
	// MissingField is the error name for missing field errors.
	MissingField = pkg.MissingField
	// InvalidEnumValue is the error name for invalid enum value errors.
	InvalidEnumValue = pkg.InvalidEnumValue
	// InvalidFormat is the error name for invalid format errors.
	InvalidFormat = pkg.InvalidFormat
	// InvalidPattern is the error name for invalid pattern errors.
	InvalidPattern = pkg.InvalidPattern
	// InvalidRange is the error name for invalid range errors.
	InvalidRange = pkg.InvalidRange
	// InvalidLength is the error name for invalid length errors.
	InvalidLength = pkg.InvalidLength
)

// Error describes a method error return value. The description includes a
// unique name (in the scope of the method), an optional type, description and
// DSL that further describes the type. If no type is specified then the
// built-in RFC 9457 problem-document result type is used. The DSL syntax is identical to the
// Attribute DSL.
//
// Error must appear in the Service (to define error responses that apply to all
// the service methods) or Method expressions. Error may also appear under the API
// expression to create reusable error definitions.
//
// See Attribute for details on the Error arguments.
//
// Example:
//
//	var _ = API("calc", func() {
//	    Error("invalid_argument") // Uses type ErrorResult
//	    HTTP(func() {
//	        Response("invalid_argument", StatusBadRequest)
//	    })
//	})
//
//	var _ = Service("divider", func() {
//	    Error("invalid_argument") // Refers to error defined above.
//	                              // No need to define HTTP mapping again.
//
//	    // Method which uses the default type for its response.
//	    Method("divide", func() {
//	        Payload(DivideRequest)
//	        Error("div_by_zero", DivByZero, "Division by zero")
//	    })
//	})
func Error(name string, args ...any) {
	if len(args) == 0 {
		args = []any{expr.ErrorResult}
	}
	if len(args) == 1 {
		if description, ok := args[0].(string); ok && expr.Root.UserType(description) == nil {
			eval.ReportError(
				`error descriptions are set with a DSL function: Error(%q, func() { Description(%q) })`,
				name,
				description,
			)
			return
		}
	}
	dt, desc, fn := parseAttributeArgs(nil, args...)
	att := &expr.AttributeExpr{
		Description: desc,
		Type:        dt,
	}
	erro := &expr.ErrorExpr{AttributeExpr: att, Name: name}
	if fn != nil {
		eval.Context.Stack = append(eval.Context.Stack, erro)
		eval.Execute(fn, att)
		eval.Context.Stack = eval.Context.Stack[:len(eval.Context.Stack)-1]
	}
	if att.Type == nil {
		att.Type = expr.ErrorResult
	}
	switch actual := eval.Current().(type) {
	case *expr.APIExpr:
		expr.Root.Errors = append(expr.Root.Errors, erro)
	case *expr.ServiceExpr:
		actual.Errors = append(actual.Errors, erro)
	case *expr.MethodExpr:
		actual.Errors = append(actual.Errors, erro)
	default:
		eval.IncompatibleDSL()
	}
}

// ProblemType overrides the RFC 9457 problem "type" URI generated for the
// enclosing error.
//
// ProblemType must appear in an Error expression.
func ProblemType(uri string) {
	errExpr, ok := currentErrorExpr()
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	errExpr.AddMeta("http:problem:type", uri)
}

// ProblemTitle overrides the RFC 9457 problem "title" generated for the
// enclosing error.
//
// ProblemTitle must appear in an Error expression.
func ProblemTitle(title string) {
	errExpr, ok := currentErrorExpr()
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	errExpr.AddMeta("http:problem:title", title)
}

// ErrorName identifies the attribute of a custom error type used to select the
// returned error response when multiple errors of that type are defined on the
// same method. The value of the field identifies the error name as defined in
// the design. This makes it possible to define distinct transport mappings for
// the various errors (for example to return different HTTP status codes). There
// must be one and exactly one attribute defined with ErrorName on types used to
// define errors.
//
// ErrorName must appear in a Type or ResultType expression.
//
// ErrorName takes the same arguments as Attribute or Field.
//
// Example design:
//
//	   // All the methods exposed by service MyService can return the errors
//	   // "internal_error" and "bad_request". Both errors have the same type
//	   // CustomErrorType. "internal_error" is mapped to HTTP status 500 and
//	   // "bad_request" is mapped to HTTP status 400.
//	   var _ = Service("MyService", func() {
//	       Error("internal_error", CustomErrorType)
//	       Error("bad_request", CustomErrorType)
//	       HTTP(func() {
//	           Response("internal_error", StatusInternalServerError)
//	           Response("bad_request", StatusBadRequest)
//	       })
//
//	       Method("Method", func() {
//		      Payload(String)
//	           HTTP(func() {
//	               GET("/")
//	           })
//	       })
//	   })
//
//	   var CustomErrorType = Type("CustomError", func() {
//	       // The "name" attribute is used to select the error response.
//	       // name should be set to either "internal_error" or "bad_request" by
//	       // the service method returning the error.
//	       ErrorName("name", String, "Name of error.")
//	       Attribute("message", String, "Message of error.")
//	       Attribute("occurred_at", String, "Time error occurred.", func() {
//	           Format(FormatDateTime)
//	       })
//	       Required("name", "message", "occurred_at")
//	   })
//
// Example usage:
//
//	func (s *svc) Method(ctx context.Context, p string) error {
//	    // ...
//	    if err != nil {
//	         return &myservice.CustomError{
//	             Name:       "internal_error", // HTTP response status is 500.
//	             Message:    "Something went wrong",
//	             OccurredAt: time.Now().Format(time.RFC3339),
//	         }
//	    }
//	    // ...
//	    return nil
//	}
func ErrorName(args ...any) {
	if len(args) == 0 {
		eval.TooFewArgError()
		return
	}
	dsl, ok := args[len(args)-1].(func())
	if ok {
		args[len(args)-1] = func() {
			dsl()
			Meta("struct:error:name")
		}
	} else {
		args = append(args, func() {
			Meta("struct:error:name")
		})
	}
	switch actual := args[0].(type) {
	case string:
		Attribute(actual, args[1:]...)
	case int:
		if len(args) == 2 {
			eval.TooFewArgError()
			return
		}
		name, ok := args[1].(string)
		if !ok {
			eval.InvalidArgError("name", args[1])
			return
		}
		Field(actual, name, args[2:]...)
	default:
		eval.InvalidArgError("name or position", args[0])
	}
}

// Temporary qualifies an error type as describing temporary (i.e. retryable)
// errors.
//
// Temporary must appear in a Error expression.
//
// Temporary takes no argument.
//
// Example:
//
//	var _ = Service("divider", func() {
//	    Error("request_timeout", func() {
//	        Temporary()
//	    })
//	})
func Temporary() {
	attr, ok := eval.Current().(*expr.AttributeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	attr.AddMeta("loom:error:temporary")
}

// Remedy declares structured remediation guidance for an error.
//
// Remedy must appear in an Error expression.
func Remedy(fn func()) {
	errExpr, ok := currentErrorExpr()
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	remedy := &expr.ErrorRemedyExpr{}
	errExpr.Remedy = remedy
	if fn != nil {
		eval.Execute(fn, remedy)
	}
}

func currentErrorExpr() (*expr.ErrorExpr, bool) {
	if errExpr, ok := eval.Current().(*expr.ErrorExpr); ok {
		return errExpr, true
	}
	for _, v := range slices.Backward(eval.Context.Stack) {
		if errExpr, ok := v.(*expr.ErrorExpr); ok {
			return errExpr, true
		}
	}
	return nil, false
}

// RemedyCode declares a stable remediation code for an error.
//
// RemedyCode must appear in a Remedy expression.
func RemedyCode(code string) {
	remedy, ok := eval.Current().(*expr.ErrorRemedyExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	remedy.Code = code
}

// SafeMessage declares the safe, user-facing message for an error.
//
// SafeMessage must appear in a Remedy expression.
func SafeMessage(message string) {
	remedy, ok := eval.Current().(*expr.ErrorRemedyExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	remedy.SafeMessage = message
}

// RetryHint declares concise retry or correction guidance for an error.
//
// RetryHint must appear in a Remedy expression.
func RetryHint(hint string) {
	remedy, ok := eval.Current().(*expr.ErrorRemedyExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	remedy.RetryHint = hint
}

// Timeout qualifies an error type as describing errors due to timeouts, or
// sets a timeout duration on expressions implementing TimeoutHolder.
//
// When used in an Error expression, Timeout takes no argument and marks the
// error as a timeout error.
//
// When used in an expression implementing TimeoutHolder, Timeout takes a
// duration string (e.g., "30s", "1m", "500ms").
//
// Example (error timeout):
//
//	var _ = Service("divider", func() {
//	    Error("request_timeout", func() {
//	        Timeout()
//	    })
//	})
//
// Example (duration timeout):
//
//	Registry("corp", func() {
//	    URL("https://registry.corp.internal")
//	    Timeout("30s")
//	})
func Timeout(args ...string) {
	switch e := eval.Current().(type) {
	case *expr.AttributeExpr:
		if len(args) > 0 {
			eval.ReportError("Timeout in Error expression takes no arguments")
			return
		}
		e.AddMeta("loom:error:timeout")
	case expr.TimeoutHolder:
		if len(args) != 1 {
			eval.ReportError("Timeout requires a duration string argument")
			return
		}
		if err := e.SetTimeout(args[0]); err != nil {
			eval.ReportError("invalid timeout: %s", err)
		}
	default:
		eval.IncompatibleDSL()
	}
}

// Fault qualifies an error type as describing errors due to a server-side
// fault.
//
// Fault must appear in a Error expression.
//
// Fault takes no argument.
//
// Example:
//
//	var _ = Service("divider", func() {
//	     Error("internal_error", func() {
//	             Fault()
//	     })
//	})
func Fault() {
	attr, ok := eval.Current().(*expr.AttributeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	attr.AddMeta("loom:error:fault")
}
