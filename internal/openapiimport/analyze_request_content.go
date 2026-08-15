package openapiimport

import (
	"reflect"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
)

func (a *analyzer) requestContent(
	content *orderedmap.Map[string, *v3.MediaType],
	path string,
) ([]string, *Schema, []Example) {
	if orderedmap.Len(content) == 0 {
		return nil, nil, nil
	}
	contentTypes := make([]string, 0, orderedmap.Len(content))
	var sharedSchema *Schema
	var sharedExamples []Example
	for contentType, media := range content.FromOldest() {
		first := len(contentTypes) == 0
		contentTypes = append(contentTypes, contentType)
		var schema *Schema
		var examples []Example
		mediaPath := path + "/" + escapeJSONPointer(contentType)
		if media != nil {
			if media.ItemSchema != nil {
				a.unsupported("media-item-schema", mediaPath+"/itemSchema", "item schemas are not in the strict import subset")
			}
			if orderedmap.Len(media.Encoding) > 0 || orderedmap.Len(media.ItemEncoding) > 0 {
				a.unsupported("media-encoding", mediaPath, "media encodings are not in the strict import subset")
			}
			schema = a.schema(media.Schema, mediaPath+"/schema")
			examples = a.mediaExamples(media, schema, mediaPath)
			a.unsupportedExtensions(mediaPath, media.Extensions)
		}
		if first {
			sharedSchema = schema
			sharedExamples = examples
			continue
		}
		if !reflect.DeepEqual(schema, sharedSchema) {
			a.unsupported("request-media-schema", mediaPath+"/schema", "request media types must use the same schema")
		}
		if !reflect.DeepEqual(examples, sharedExamples) {
			a.unsupported("request-media-examples", mediaPath, "request media types must use the same examples")
		}
	}
	return contentTypes, sharedSchema, sharedExamples
}
