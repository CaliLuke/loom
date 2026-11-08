// Package formgen lowers captured form codecs into generated typed adapters.
// Callers supply physical declarations and selectors; this package never infers
// authored media policy from the generated Go representation.
package formgen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

type (
	// Field binds a captured codec to one typed form value.
	Field struct {
		// Codec preserves the occurrence's style and value-versus-item application.
		Codec httpencoding.Codec
		// AllowsNull preserves explicit null admission for JSON values.
		AllowsNull bool
	}
	// Access supplies expressions at one emission site.
	Access struct {
		// Values and Name identify the raw form map and encoded field name.
		Values, Name string
		// Value is the allocated Go selector, without an address operator.
		Value string
		// Presence is an optional physical presence expression for encoding.
		Presence string
		// Required rejects absent values or zero emitted items where keys are evidence.
		Required bool
		// ErrorPrefix supplies other return values before an error (for example "\"\", ").
		ErrorPrefix string
		// Label names the field in diagnostics.
		Label string
	}
)

// Application renders the already resolved codec application.
func Application(application httpencoding.Application) string {
	if application == httpencoding.Items {
		return "loomhttp.FieldItems"
	}
	return "loomhttp.FieldValue"
}

// EncodeCall renders one JSON or native leaf call, retaining captured policy.
func (field Field) EncodeCall(values, name, value string) string {
	if field.Codec.Style == httpencoding.JSON {
		return fmt.Sprintf("loomhttp.EncodeJSONField(%s, %s, %s, %s)", values, name, value, Application(field.Codec.Application))
	}
	return fmt.Sprintf("loomhttp.EncodeFormValue(%s, %s, %s)", values, name, value)
}

// DecodeCall renders one typed leaf decode without reflection-based policy inference.
func (field Field) DecodeCall(values, name, value string) string {
	if field.Codec.Style == httpencoding.JSON {
		return fmt.Sprintf("loomhttp.DecodeJSONField(%s, %s, &%s, %t, %s)", values, name, value, field.AllowsNull, Application(field.Codec.Application))
	}
	return fmt.Sprintf("loomhttp.DecodeFormValue(%s, %s, &%s)", values, name, value)
}

// Encode renders presence and required-item checks around a captured leaf call.
func (field Field) Encode(access Access) string {
	var code strings.Builder
	if access.Presence != "" {
		fmt.Fprintf(&code, "if %s {\n", access.Presence)
	}
	call := field.EncodeCall(access.Values, access.Name, access.Value)
	if access.Required && field.Codec.Application == httpencoding.Items {
		fmt.Fprintf(&code, "if parameterPresent, parameterError := %s; parameterError != nil {\nreturn %sparameterError\n} else if !parameterPresent {\nreturn %sfmt.Errorf(%q)\n}\n", call, access.ErrorPrefix, access.ErrorPrefix, "required form field "+access.Label+" has no encoded items")
	} else {
		fmt.Fprintf(&code, "if _, parameterError := %s; parameterError != nil {\nreturn %sparameterError\n}\n", call, access.ErrorPrefix)
	}
	if access.Presence != "" {
		if access.Required {
			fmt.Fprintf(&code, "} else {\nreturn %sfmt.Errorf(%q)\n", access.ErrorPrefix, "required form field "+access.Label+" is absent")
		}
		code.WriteString("}\n")
	}
	return code.String()
}

// Presence renders the physical declaration's presence predicate. Empty values
// remain present; codec policy never follows Go wrapper shape inference.
func Presence(context *codegen.AttributeContext, parent *expr.AttributeExpr, key string, physical *expr.AttributeExpr, value string) string {
	if context.FieldPresence(expr.NewObjectFields(parent), key, physical) != codegen.NativePresence {
		return value + ".Present()"
	}
	if expr.AsUnion(physical.Type) != nil && expr.HasNativeJSONPresence(physical) {
		return value + ".Kind() != \"\""
	}
	return ""
}
