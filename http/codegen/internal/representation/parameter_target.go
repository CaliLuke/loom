package representation

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

// normalizeParameterTarget gives text records their DSL member names. JSON tags
// belong only to JSON occurrences, including ordinary JSON-valued form fields.
// The caller owns target; independently copy each styled field because a form
// can reference the same declaration through both a styled and a JSON field.
func normalizeParameterTarget(target *expr.AttributeExpr, codec expr.ValueCodec) {
	switch codec {
	case expr.ValueCodecText:
		normalizeTextRecord(target)
	case expr.ValueCodecForm:
		body := expr.UnwrapInlineHTTPBody(target)
		object := expr.AsObject(body.Type)
		if object == nil || len(*object) == 1 && expr.IsUnion((*object)[0].Attribute.Type) {
			return
		}
		for _, field := range *object {
			name := expr.ElementName(field.Name)
			resolved, _, err := httpencoding.Resolve(httpencoding.FormField, name, expr.HTTPValueShape(field.Attribute), expr.HTTPParameterSerialization(body, "openapi:encoding:"+name+":"))
			if err != nil {
				panic(fmt.Errorf("validated form parameter %q: %w", name, err))
			}
			if resolved.Style == httpencoding.JSON {
				continue
			}
			field.Attribute = TargetGraph(field.Attribute)
			normalizeTextRecord(field.Attribute)
		}
	}
}

func normalizeTextRecord(attribute *expr.AttributeExpr) {
	object := expr.AsObject(attribute.Type)
	if object == nil {
		return
	}
	for _, field := range *object {
		delete(field.Attribute.Meta, "struct:tag:json")
		if field.Attribute.Meta == nil {
			field.Attribute.Meta = make(expr.MetaExpr)
		}
		field.Attribute.Meta["struct:tag:json:name"] = []string{expr.ElementName(field.Name)}
	}
}
