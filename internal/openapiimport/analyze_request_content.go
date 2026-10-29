package openapiimport

import (
	v3 "github.com/CaliLuke/libopenapi/datamodel/high/v3"
	"github.com/CaliLuke/libopenapi/orderedmap"
	"github.com/pb33f/go-yaml"
)

func (a *analyzer) requestContent(
	content *orderedmap.Map[string, *v3.MediaType],
	path string,
) ([]string, *Schema, []Example) {
	if orderedmap.Len(content) == 0 {
		return nil, nil, nil
	}
	contentTypes := make([]string, 0, orderedmap.Len(content))
	branches := make([]*Schema, 0, orderedmap.Len(content))
	var singleExamples []Example
	for contentType, media := range content.FromOldest() {
		contentTypes = append(contentTypes, contentType)
		mediaPath := path + "/" + escapeJSONPointer(contentType)
		var schema *Schema
		var examples []Example
		if media != nil {
			a.mediaTypeParserGapDiagnostics(media, mediaPath)
			if media.ItemSchema != nil {
				a.unsupported("media-item-schema", mediaPath+"/itemSchema", "item schemas are not in the strict import subset")
			}
			if orderedmap.Len(media.Encoding) > 0 && !isFormMediaType(contentType) && !isMultipartMediaType(contentType) || orderedmap.Len(media.ItemEncoding) > 0 {
				a.unsupported("media-encoding", mediaPath, "media encodings are not in the strict import subset")
			}
			schema = a.schema(media.Schema, mediaPath+"/schema")
			examples = a.mediaExamples(media, schema, mediaPath)
			if len(contentTypes) == 1 {
				singleExamples = examples
			}
			if orderedmap.Len(content) > 1 {
				schema = schemaWithExamples(schema, examples)
				if schema != nil {
					schema.MediaEncoding = a.mediaEncoding(media, mediaPath)
				}
			}
			a.unsupportedExtensions(mediaPath, media.Extensions)
		}
		branches = append(branches, schema)
	}
	if len(branches) == 1 {
		return contentTypes, branches[0], singleExamples
	}
	return contentTypes, &Schema{OneOf: branches, MediaTypes: contentTypes}, nil
}

// mediaTypeParserGapDiagnostics inspects official OpenAPI 3.2 fields that the
// current libopenapi MediaType model does not expose. Keeping this check at the
// normalized-model boundary prevents a parser gap from becoming silent loss.
func (a *analyzer) mediaTypeParserGapDiagnostics(media *v3.MediaType, path string) {
	if media == nil || media.GoLow() == nil {
		return
	}
	root := media.GoLow().GetRootNode()
	if root == nil || root.Kind != yaml.MappingNode {
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
