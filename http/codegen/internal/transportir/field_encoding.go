package transportir

import (
	"fmt"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/httpencoding"
)

func formBodyEncoding(body *expr.AttributeExpr, form bool) *expr.HTTPFormEncoding {
	if !form || body == nil {
		return nil
	}
	fields, err := expr.HTTPFormBodyEncoding(body)
	if err != nil {
		panic(fmt.Errorf("validated form body: %w", err))
	}
	return fields
}

func multipartBodyEncoding(body *expr.AttributeExpr, multipart bool) *expr.HTTPMultipartEncoding {
	if !multipart || body == nil {
		return nil
	}
	contract, err := expr.HTTPMultipartBodyEncoding(body)
	if err != nil {
		panic(fmt.Errorf("validated multipart body: %w", err))
	}
	return contract
}

func parameterCodec(location httpencoding.Location, name string, attribute *expr.AttributeExpr, source httpencoding.Serialization) (httpencoding.Codec, httpencoding.KeySet) {
	codec, keys, err := httpencoding.Resolve(location, name, expr.HTTPValueShape(attribute), source)
	if err != nil {
		panic(fmt.Errorf("validated parameter %q: %w", name, err))
	}
	return codec, keys
}
