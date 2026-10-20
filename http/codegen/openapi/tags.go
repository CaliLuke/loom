package openapi

import (
	"encoding/json/jsontext"
	"sort"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/openapimeta"
)

// Tag allows adding meta data to a single tag that is used by the Operation Object. It is
// not mandatory to have a Tag Object per tag used there.
type Tag struct {
	// Name of the tag.
	Name string `json:"name,omitzero" yaml:"name,omitempty"`
	// Summary is the short display label for the tag in OpenAPI 3.2.
	Summary string `json:"summary,omitzero" yaml:"summary,omitempty"`
	// Description is a short description of the tag.
	// GFM syntax can be used for rich text representation.
	Description string `json:"description,omitzero" yaml:"description,omitempty"`
	// Parent names the tag that contains this tag in OpenAPI 3.2.
	Parent string `json:"parent,omitzero" yaml:"parent,omitempty"`
	// Kind classifies the tag in OpenAPI 3.2.
	Kind string `json:"kind,omitzero" yaml:"kind,omitempty"`
	// ExternalDocs is additional external documentation for this tag.
	ExternalDocs *ExternalDocs `json:"externalDocs,omitzero" yaml:"externalDocs,omitempty"`
	// Extensions defines the OpenAPI extensions.
	Extensions map[string]any `json:"-" yaml:"-"`
}

// TagsFromExpr extracts the OpenAPI related metadata from the given expression.
func TagsFromExpr(mdata expr.MetaExpr) (tags []*Tag) {
	keys := make([]string, 0, len(mdata))
	for k := range mdata {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		name, field, ok, err := openapimeta.ParseTagKey(key)
		if !ok || err != nil {
			continue
		}

		var tag *Tag
		for _, t := range tags {
			if t.Name == name {
				tag = t
				break
			}
		}
		if tag == nil {
			tag = &Tag{Name: name}
			tags = append(tags, tag)
		}
		if field != "" {
			values := mdata[key]
			if len(values) != 1 {
				continue
			}
			switch field {
			case "desc":
				tag.Description = values[0]
			case "summary":
				tag.Summary = values[0]
			case "parent":
				tag.Parent = values[0]
			case "kind":
				tag.Kind = values[0]
			case "url":
				if tag.ExternalDocs == nil {
					tag.ExternalDocs = &ExternalDocs{}
				}
				tag.ExternalDocs.URL = values[0]
			case "url:desc":
				if tag.ExternalDocs == nil {
					tag.ExternalDocs = &ExternalDocs{}
				}
				tag.ExternalDocs.Description = values[0]
			default:
				tag.Extensions = tagExtensionsFromExpr(mdata, name)
			}
		}
	}

	return tags
}

func tagExtensionsFromExpr(metadata expr.MetaExpr, name string) map[string]any {
	var extensions map[string]any
	for key, values := range metadata {
		tagName, field, ok, err := openapimeta.ParseTagKey(key)
		if !ok || err != nil || tagName != name || !strings.HasPrefix(field, "extension:") || len(values) != 1 {
			continue
		}
		name := strings.TrimPrefix(field, "extension:")
		if extensions == nil {
			extensions = make(map[string]any)
		}
		raw := jsontext.Value(values[0]).Clone()
		if err := raw.Format(jsontext.ReorderRawObjects(true)); err != nil {
			extensions[name] = values[0]
			continue
		}
		extensions[name] = raw
	}
	return extensions
}

// TagNamesFromExpr computes the names of the OpenAPI tags specified in the
// given metadata expressions.
func TagNamesFromExpr(mdata expr.MetaExpr) (tagNames []string) {
	tags := TagsFromExpr(mdata)
	for _, tag := range tags {
		tagNames = append(tagNames, tag.Name)
	}
	return
}

type _tag Tag

// MarshalJSON returns the JSON encoding of t.
func (t Tag) MarshalJSON() ([]byte, error) {
	return MarshalJSON(_tag(t), t.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (t Tag) MarshalYAML() (any, error) {
	return MarshalYAML(_tag(t), t.Extensions)
}
