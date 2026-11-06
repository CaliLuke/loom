package transportir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func formFieldCodecs(body *expr.AttributeExpr, form bool) map[string]httpencoding.Codec {
	if !form || body == nil {
		return nil
	}
	body = expr.UnwrapInlineHTTPBody(body)
	object := expr.AsObject(body.Type)
	if object == nil || len(*object) == 1 && expr.IsUnion((*object)[0].Attribute.Type) {
		return nil
	}
	codecs := make(map[string]httpencoding.Codec, len(*object))
	for _, field := range *object {
		name := expr.ElementName(field.Name)
		codec, _ := parameterCodec(httpencoding.FormField, name, field.Attribute, expr.HTTPParameterSerialization(body, "openapi:encoding:"+name+":"))
		codecs[name] = codec
	}
	return codecs
}

func parameterCodec(location httpencoding.Location, name string, attribute *expr.AttributeExpr, source httpencoding.Serialization) (httpencoding.Codec, httpencoding.KeySet) {
	codec, keys, err := httpencoding.Resolve(location, name, expr.HTTPValueShape(attribute), source)
	if err != nil {
		panic(fmt.Errorf("validated parameter %q: %w", name, err))
	}
	return codec, keys
}
