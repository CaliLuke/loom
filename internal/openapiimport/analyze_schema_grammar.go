package openapiimport

import (
	"reflect"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/orderedmap"
	yaml4 "go.yaml.in/yaml/v4"
)

var knownSchemaKeywords = schemaKeywordSet()

func (a *analyzer) schemaUnknownKeywordDiagnostics(proxy *base.SchemaProxy, path string) {
	if proxy == nil {
		return
	}
	node := proxy.GetValueNode()
	if node == nil || node.Kind != yaml4.MappingNode {
		return
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		keyword := node.Content[index].Value
		if _, ok := knownSchemaKeywords[keyword]; ok || strings.HasPrefix(keyword, "x-") {
			continue
		}
		a.unsupported(
			"schema-keyword-unknown",
			path+"/"+escapeJSONPointer(keyword),
			"schema keyword "+keyword+" is not recognized by the importer and cannot be preserved",
		)
	}
}

func schemaKeywordSet() map[string]struct{} {
	typeOf := reflect.TypeOf(base.Schema{})
	keywords := make(map[string]struct{}, typeOf.NumField()+1)
	keywords["$ref"] = struct{}{}
	for index := 0; index < typeOf.NumField(); index++ {
		tag := strings.Split(typeOf.Field(index).Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		keywords[tag] = struct{}{}
	}
	return keywords
}

func (a *analyzer) schemaUnsupportedKeywords(
	schema *Schema,
	source *base.Schema,
	path string,
	supportedAllOf bool,
	supportedNullableComposition bool,
	supportedNullableOneOf bool,
) {
	if len(source.AllOf) > 0 && !supportedAllOf || len(source.OneOf) > 0 && !supportedNullableOneOf ||
		len(source.AnyOf) > 0 && !supportedNullableComposition || source.Not != nil {
		schema.unsupportedComposition = true
	}
	if source.Discriminator != nil || len(source.PrefixItems) > 0 || source.Contains != nil ||
		source.MinContains != nil || source.MaxContains != nil ||
		source.If != nil || source.Then != nil || source.Else != nil ||
		orderedmap.Len(source.DependentSchemas) > 0 || orderedmap.Len(source.DependentRequired) > 0 ||
		orderedmap.Len(source.PatternProperties) > 0 || source.PropertyNames != nil || source.UnevaluatedProperties != nil {
		a.unsupported("advanced-schema", path, "advanced JSON Schema applicators are not in the strict import subset")
	}
	if source.MultipleOf != nil || source.UniqueItems != nil || source.MaxProperties != nil || source.MinProperties != nil ||
		source.Const != nil ||
		source.ContentEncoding != "" || source.ContentMediaType != "" || source.XML != nil || source.ExternalDocs != nil {
		a.unsupported("schema-keyword", path, "one or more schema keywords are not in the strict import subset")
	}
	if source.DynamicRef != "" || source.Anchor != "" || source.DynamicAnchor != "" || source.SchemaTypeRef != "" ||
		source.Id != "" || source.Comment != "" || source.ContentSchema != nil || orderedmap.Len(source.Defs) > 0 ||
		orderedmap.Len(source.Vocabulary) > 0 || source.UnevaluatedItems != nil {
		a.unsupported("schema-resource", path, "JSON Schema resource and dialect keywords are not in the strict import subset")
	}
}
