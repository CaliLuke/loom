package ir

import (
	"fmt"

	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func analyzeMediaVariants(a *Analyzer, variants []*transportir.MediaVariant, usage schemaUsage) map[string]*Schema {
	schemas := make(map[string]*Schema, len(variants))
	for _, variant := range variants {
		attribute := attributeForSchemaUsage(variant.Body, usage)
		context := attributeExampleContext(attribute, a.closeObjects, "media-variant", variant.ContentType)
		schemas[variant.ContentType] = a.analyzePreparedOccurrence(attribute, context, variant.Value)
	}
	return schemas
}

func buildMediaVariantContent(variants []*transportir.MediaVariant, schemas map[string]*Schema, closeObjects bool, usage schemaUsage) (map[string]*MediaType, error) {
	content := make(map[string]*MediaType, len(variants))
	for _, variant := range variants {
		attribute := attributeForSchemaUsage(variant.Body, usage)
		media := buildMediaType(attribute, schemas[variant.ContentType], closeObjects, false, variant.Value)
		media.Metadata = mediaTypeMetadata(variant.Metadata)
		if err := applyMediaFieldEncoding(&transportir.Request{Body: variant.Body, FormEncoded: variant.Form != nil, Form: variant.Form, Multipart: variant.Multipart != nil, MultipartEncoding: variant.Multipart}, media); err != nil {
			return nil, fmt.Errorf("media type %q: %w", variant.ContentType, err)
		}
		content[variant.ContentType] = media
	}
	return content, nil
}
