package openapiimport

import (
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
)

func (a *analyzer) tags(source []*base.Tag) ([]string, []Tag) {
	names := make([]string, 0, len(source))
	tags := make([]Tag, 0, len(source))
	for _, tag := range source {
		if tag == nil {
			continue
		}
		names = append(names, tag.Name)
		tagPath := "#/tags/" + escapeJSONPointer(tag.Name)
		normalized := Tag{
			Name:        tag.Name,
			Summary:     tag.Summary,
			Description: tag.Description,
			Parent:      tag.Parent,
			Kind:        tag.Kind,
			Extensions:  a.extensions(tagPath, tag.Extensions),
		}
		if !a.openAPI32() && (tag.Summary != "" || tag.Parent != "" || tag.Kind != "") {
			a.unsupported(
				"versioned-field",
				tagPath,
				"tag summary, parent, and kind require OpenAPI 3.2",
			)
		}
		if tag.ExternalDocs != nil {
			normalized.ExternalDocsURL = tag.ExternalDocs.URL
			normalized.ExternalDocsDescription = tag.ExternalDocs.Description
			if orderedmap.Len(tag.ExternalDocs.Extensions) > 0 {
				a.unsupported(
					"tag-external-docs-extensions",
					tagPath+"/externalDocs",
					"tag external documentation extensions are not in the strict import subset",
				)
			}
		}
		tags = append(tags, normalized)
	}
	return names, tags
}
