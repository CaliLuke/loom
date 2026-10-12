package transportir

import (
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// URLShape resolves an evaluated attribute to the shared URL codec category.
func URLShape(attribute *expr.AttributeExpr) httpencoding.Shape {
	if attribute == nil {
		return httpencoding.UnsupportedShape
	}
	switch {
	case expr.AsMap(attribute.Type) != nil:
		return httpencoding.Map
	case expr.AsObject(attribute.Type) != nil || expr.IsUnion(attribute.Type):
		return httpencoding.Object
	case expr.AsArray(attribute.Type) != nil:
		return httpencoding.Sequence
	default:
		return httpencoding.Scalar
	}
}

func formFieldModes(body *expr.AttributeExpr, form bool) map[string]httpencoding.Mode {
	if !form || body == nil {
		return nil
	}
	body = expr.UnwrapInlineHTTPBody(body)
	object := expr.AsObject(body.Type)
	if object == nil || len(*object) == 1 && expr.IsUnion((*object)[0].Attribute.Type) {
		return nil // A single union is the established unwrapped constructor form.
	}
	modes := make(map[string]httpencoding.Mode, len(*object))
	for _, field := range *object {
		modes[expr.ElementName(field.Name)] = httpencoding.Select(httpencoding.FormField, URLShape(field.Attribute))
	}
	return modes
}
