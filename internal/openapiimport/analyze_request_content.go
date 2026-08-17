package openapiimport

import (
	"reflect"

	v3 "github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml4 "go.yaml.in/yaml/v4"
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
			a.mediaTypeParserGapDiagnostics(media, mediaPath)
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

// mediaTypeParserGapDiagnostics inspects official OpenAPI 3.2 fields that the
// current libopenapi MediaType model does not expose. Keeping this check at the
// normalized-model boundary prevents a parser gap from becoming silent loss.
func (a *analyzer) mediaTypeParserGapDiagnostics(media *v3.MediaType, path string) {
	if media == nil || media.GoLow() == nil {
		return
	}
	root := media.GoLow().GetRootNode()
	if root == nil || root.Kind != yaml4.MappingNode {
		return
	}
	for index := 0; index+1 < len(root.Content); index += 2 {
		switch root.Content[index].Value {
		case "description":
			a.unsupported(
				"media-type-description",
				path+"/description",
				"media type descriptions are omitted because the Loom HTTP DSL has no media-level description",
			)
		case "prefixEncoding":
			a.unsupported(
				"media-prefix-encoding",
				path+"/prefixEncoding",
				"prefixEncoding is not in the strict import subset",
			)
		}
	}
}
