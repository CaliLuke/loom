package expr

import (
	"reflect"
	"slices"
	"strings"

	"github.com/CaliLuke/loom/internal/encodingmeta"
	"github.com/CaliLuke/loom/internal/naming"
)

type compositionJSONTags struct {
	attribute       *AttributeExpr
	bases           []*AttributeExpr
	effective       *AttributeExpr
	path, fieldName string
	optional        bool
}

func (c *designComposer) composeAttributeMetadata(attribute *AttributeExpr, bases []*AttributeExpr, path, fieldName string, optional bool) bool {
	local := attribute.Meta
	result := make(MetaExpr)
	owners := make(map[string]*AttributeExpr)
	for _, base := range bases {
		keys := make([]string, 0, len(base.Meta))
		for key := range base.Meta {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		for _, key := range keys {
			if !composeInheritedMetadata(attribute, base, result, owners, key, path) {
				return false
			}
		}
	}
	for key, values := range local {
		if !compositionIdentityMeta(key) && !compositionDocumentationMeta(key) && !attribute.removedMeta[key] {
			result[key] = append(result[key], values...)
		} else {
			result[key] = slices.Clone(values)
		}
	}
	authored := *attribute
	c.jsonTags = append(c.jsonTags, compositionJSONTags{
		attribute: &authored, bases: bases, effective: attribute,
		path: path, fieldName: fieldName, optional: optional,
	})
	attribute.Meta = result
	return true
}

func composeInheritedMetadata(
	attribute, base *AttributeExpr, result MetaExpr,
	owners map[string]*AttributeExpr, key, path string,
) bool {
	local := attribute.Meta
	values := base.Meta[key]
	identity := compositionIdentityMeta(key)
	if key == "openapi:additionalProperties" {
		if !allowsAdditionalProperties(base) {
			if _, explicit := local[key]; explicit && allowsAdditionalProperties(attribute) || attribute.removedMeta[key] {
				compositionError(attribute, "%s: cannot reopen inherited closed object from %s", path, compositionSource(base))
				return false
			}
			result[key] = []string{"false"}
		} else if _, present := result[key]; !present {
			result[key] = slices.Clone(values)
		}
		return true
	}
	if attribute.removedMeta[key] {
		if own, replaced := local[key]; identity && (!replaced || !equalCompositionMeta(key, own, values)) {
			compositionError(attribute, "%s: cannot remove inherited identity metadata %q from %s", path, key, compositionSource(base))
			return false
		}
		return true
	}
	if key == "struct:tag:json" || key == "struct:tag:json:name" {
		// Validate the resolved JSON contract after representation metadata merges.
		if _, localTag := local[key]; !localTag {
			result[key] = slices.Clone(values)
		}
		return true
	}
	if own, exists := local[key]; exists {
		if identity && !equalCompositionMeta(key, own, values) {
			compositionError(attribute, "%s: incompatible inherited metadata %q from %s", path, key, compositionSource(base))
			return false
		}
		if identity || compositionDocumentationMeta(key) {
			return true
		}
	}
	if prior, exists := result[key]; exists && !equalCompositionMeta(key, prior, values) {
		compositionError(attribute, "%s: conflicting inherited metadata %q from %s and %s", path, key, compositionSource(owners[key]), compositionSource(base))
		return false
	}
	result[key] = slices.Clone(values)
	owners[key] = base
	return true
}

func validateCompositionJSONTags(attribute *AttributeExpr, bases []*AttributeExpr, effective *AttributeExpr, path, fieldName string, optional bool) bool {
	parent := &AttributeExpr{Type: &Object{{Name: fieldName, Attribute: effective}}}
	if !optional {
		parent.Validation = &ValidationExpr{Required: []string{fieldName}}
	}
	policy := ServiceFieldLayout(parent, fieldName, effective, false, true).JSON
	tag := func(source *AttributeExpr) string {
		return encodingmeta.ResolveJSONField(source.Meta, AttributeName(fieldName), policy).Tag
	}
	var owner *AttributeExpr
	for _, base := range bases {
		if _, present := attributeJSONTag(base); !present {
			continue
		}
		if owner != nil && tag(owner) != tag(base) {
			compositionError(attribute, "%s: conflicting inherited JSON tag from %s and %s", path, compositionSource(owner), compositionSource(base))
			return false
		}
		if _, local := attributeJSONTag(attribute); local && tag(attribute) != tag(base) {
			compositionError(attribute, "%s: incompatible inherited JSON tag from %s", path, compositionSource(base))
			return false
		}
		owner = base
	}
	return true
}

func compositionIdentityMeta(key string) bool {
	return strings.HasPrefix(key, "struct:") || strings.HasPrefix(key, "security:") ||
		strings.HasPrefix(key, "oneof:") || key == "rpc:tag" || key == "proto:tag:json" ||
		key == "openapi:additionalProperties" || key == "openapi:contentEncoding" || key == "openapi:contentMediaType" || key == "openapi:format"
}

func compositionDocumentationMeta(key string) bool {
	return key == "documentation" || key == "openapi:summary" || key == "openapi:description" ||
		key == "openapi:example" || key == "openapi:externalDocs:description" || key == "openapi:externalDocs:url"
}

func equalCompositionMeta(key string, left, right []string) bool {
	if key == "struct:error:name" || strings.HasPrefix(key, "security:") {
		return true // Role markers are interpreted by presence, not values.
	}
	a, b := MetaExpr{key: left}, MetaExpr{key: right}
	switch key {
	case "struct:type:name", "proto:tag:json", "oneof:typename", "oneof:type:field", "oneof:value:field":
		av, ap := a.First(key)
		bv, bp := b.First(key)
		return ap == bp && av == bv
	case "struct:field:type":
		return encodingmeta.GoType(a) == encodingmeta.GoType(b)
	case "struct:field:proto":
		return encodingmeta.ProtoType(a) == encodingmeta.ProtoType(b)
	case "rpc:tag", "struct:pkg:path", "openapi:additionalProperties", "openapi:contentEncoding", "openapi:contentMediaType", "openapi:format":
		av, ap := a.Last(key)
		bv, bp := b.Last(key)
		return ap == bp && av == bv
	case "struct:field:name":
		return naming.GoifyAttribute("", a, true) == naming.GoifyAttribute("", b, true)
	case "struct:tag:json", "struct:tag:json:name":
		av, ap := attributeJSONTag(&AttributeExpr{Meta: a})
		bv, bp := attributeJSONTag(&AttributeExpr{Meta: b})
		return ap == bp && av == bv
	}
	return reflect.DeepEqual(left, right)
}

func composeAttributeDocumentation(attribute *AttributeExpr, bases []*AttributeExpr, path string) bool {
	inherited := &AttributeExpr{}
	for _, base := range bases {
		if (attribute.Description == "" && inherited.Description != "" && base.Description != "" && inherited.Description != base.Description) ||
			(attribute.Title == "" && inherited.Title != "" && base.Title != "" && inherited.Title != base.Title) ||
			(attribute.Docs == nil && inherited.Docs != nil && base.Docs != nil && !reflect.DeepEqual(inherited.Docs, base.Docs)) ||
			(len(attribute.UserExamples) == 0 && len(inherited.UserExamples) > 0 && len(base.UserExamples) > 0 && !equalCompositionExamples(inherited.UserExamples, base.UserExamples)) {
			compositionError(attribute, "%s: conflicting inherited documentation from %s; declare a local choice", path, compositionSources(bases))
			return false
		}
		if base.Description != "" {
			inherited.Description = base.Description
		}
		if base.Title != "" {
			inherited.Title = base.Title
		}
		if base.Docs != nil {
			inherited.Docs = base.Docs
		}
		if len(base.UserExamples) > 0 {
			inherited.UserExamples = base.UserExamples
		}
	}
	if attribute.Description == "" {
		attribute.Description = inherited.Description
	}
	if attribute.Title == "" {
		attribute.Title = inherited.Title
	}
	if attribute.Docs == nil && inherited.Docs != nil {
		docs := *inherited.Docs
		attribute.Docs = &docs
	}
	if len(attribute.UserExamples) == 0 {
		for _, example := range inherited.UserExamples {
			attribute.UserExamples = append(attribute.UserExamples, copyValueExample(example))
		}
	}
	return true
}

func equalCompositionExamples(left, right []*ExampleExpr) bool {
	return slices.EqualFunc(left, right, func(a, b *ExampleExpr) bool {
		if a == nil || b == nil {
			return a == b
		}
		ac, bc := *a, *b
		ac.valueOrigin, bc.valueOrigin = nil, nil
		return reflect.DeepEqual(ac, bc)
	})
}
