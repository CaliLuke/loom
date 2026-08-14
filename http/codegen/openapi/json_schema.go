package openapi

import (
	"encoding/json"
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// Schema represents an instance of a JSON schema.
	// See http://json-schema.org/documentation.html
	Schema struct {
		Schema string `json:"$schema,omitempty" yaml:"$schema,omitempty"`
		// Core schema
		ID           string             `json:"id,omitempty" yaml:"id,omitempty"`
		Title        string             `json:"title,omitempty" yaml:"title,omitempty"`
		Type         Type               `json:"type,omitempty" yaml:"type,omitempty"`
		Items        *Schema            `json:"items,omitempty" yaml:"items,omitempty"`
		Properties   map[string]*Schema `json:"properties,omitempty" yaml:"properties,omitempty"`
		Defs         map[string]*Schema `json:"$defs,omitempty" yaml:"$defs,omitempty"`
		Description  string             `json:"description,omitempty" yaml:"description,omitempty"`
		DefaultValue any                `json:"default,omitempty" yaml:"default,omitempty"`
		Example      any                `json:"example,omitempty" yaml:"example,omitempty"`

		// Hyper schema
		Media            *Media `json:"media,omitempty" yaml:"media,omitempty"`
		ReadOnly         bool   `json:"readOnly,omitempty" yaml:"readOnly,omitempty"`
		WriteOnly        bool   `json:"writeOnly,omitempty" yaml:"writeOnly,omitempty"`
		Deprecated       bool   `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
		ContentEncoding  string `json:"contentEncoding,omitempty" yaml:"contentEncoding,omitempty"`
		ContentMediaType string `json:"contentMediaType,omitempty" yaml:"contentMediaType,omitempty"`
		// ContentSchema describes decoded content after applying ContentEncoding and ContentMediaType.
		ContentSchema *Schema `json:"contentSchema,omitempty" yaml:"contentSchema,omitempty"`
		PathStart     string  `json:"pathStart,omitempty" yaml:"pathStart,omitempty"`
		Links         []*Link `json:"links,omitempty" yaml:"links,omitempty"`
		Ref           string  `json:"$ref,omitempty" yaml:"$ref,omitempty"`

		// Validation
		Enum                  []any    `json:"enum,omitempty" yaml:"enum,omitempty"`
		Format                string   `json:"format,omitempty" yaml:"format,omitempty"`
		Pattern               string   `json:"pattern,omitempty" yaml:"pattern,omitempty"`
		ExclusiveMinimum      *float64 `json:"exclusiveMinimum,omitempty" yaml:"exclusiveMinimum,omitempty"`
		Minimum               *float64 `json:"minimum,omitempty" yaml:"minimum,omitempty"`
		ExclusiveMaximum      *float64 `json:"exclusiveMaximum,omitempty" yaml:"exclusiveMaximum,omitempty"`
		Maximum               *float64 `json:"maximum,omitempty" yaml:"maximum,omitempty"`
		MinLength             *int     `json:"minLength,omitempty" yaml:"minLength,omitempty"`
		MaxLength             *int     `json:"maxLength,omitempty" yaml:"maxLength,omitempty"`
		MinItems              *int     `json:"minItems,omitempty" yaml:"minItems,omitempty"`
		MaxItems              *int     `json:"maxItems,omitempty" yaml:"maxItems,omitempty"`
		Required              []string `json:"required,omitempty" yaml:"required,omitempty"`
		AdditionalProperties  any      `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
		UnevaluatedProperties any      `json:"unevaluatedProperties,omitempty" yaml:"unevaluatedProperties,omitempty"`

		// Union
		AnyOf         []*Schema      `json:"anyOf,omitempty" yaml:"anyOf,omitempty"`
		OneOf         []*Schema      `json:"oneOf,omitempty" yaml:"oneOf,omitempty"`
		Discriminator *Discriminator `json:"discriminator,omitempty" yaml:"discriminator,omitempty"`
		XML           *XML           `json:"xml,omitempty" yaml:"xml,omitempty"`

		// Extensions defines the OpenAPI extensions.
		Extensions map[string]any `json:"-" yaml:"-"`
	}

	// Type is the JSON type enum.
	Type string

	// Media represents a "media" field in a JSON hyper schema.
	Media struct {
		BinaryEncoding string `json:"binaryEncoding,omitempty" yaml:"binaryEncoding,omitempty"`
		Type           string `json:"type,omitempty" yaml:"type,omitempty"`
	}

	// Discriminator represents an OpenAPI discriminator object.
	Discriminator struct {
		PropertyName   string            `json:"propertyName" yaml:"propertyName"`
		Mapping        map[string]string `json:"mapping,omitempty" yaml:"mapping,omitempty"`
		DefaultMapping string            `json:"defaultMapping,omitempty" yaml:"defaultMapping,omitempty"`
		// Optional records that the discriminator property may be omitted in OpenAPI 3.2.
		Optional bool `json:"-" yaml:"-"`
	}

	// XML describes how a schema maps to XML nodes.
	XML struct {
		Name      string `json:"name,omitempty" yaml:"name,omitempty"`
		Namespace string `json:"namespace,omitempty" yaml:"namespace,omitempty"`
		Prefix    string `json:"prefix,omitempty" yaml:"prefix,omitempty"`
		NodeType  string `json:"nodeType,omitempty" yaml:"nodeType,omitempty"`
		Attribute bool   `json:"attribute,omitempty" yaml:"attribute,omitempty"`
		Wrapped   bool   `json:"wrapped,omitempty" yaml:"wrapped,omitempty"`
	}

	// Link represents a "link" field in a JSON hyper schema.
	Link struct {
		Title        string  `json:"title,omitempty" yaml:"title,omitempty"`
		Description  string  `json:"description,omitempty" yaml:"description,omitempty"`
		Rel          string  `json:"rel,omitempty" yaml:"rel,omitempty"`
		Href         string  `json:"href,omitempty" yaml:"href,omitempty"`
		Method       string  `json:"method,omitempty" yaml:"method,omitempty"`
		Schema       *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
		TargetSchema *Schema `json:"targetSchema,omitempty" yaml:"targetSchema,omitempty"`
		ResultType   string  `json:"mediaType,omitempty" yaml:"mediaType,omitempty"`
		EncType      string  `json:"encType,omitempty" yaml:"encType,omitempty"`
	}

	// These types are used in marshalJSON() to avoid recursive call of json.Marshal().
	_Schema Schema
)

const (
	// Array represents a JSON array.
	Array Type = "array"
	// Boolean represents a JSON boolean.
	Boolean = "boolean"
	// Integer represents a JSON number without a fraction or exponent part.
	Integer = "integer"
	// Number represents any JSON number. Number includes integer.
	Number = "number"
	// Null represents the JSON null value.
	Null = "null"
	// Object represents a JSON object.
	Object = "object"
	// String represents a JSON string.
	String = "string"
	// File is an extension used by OpenAPI to represent a file download.
	File = "file"
)

// SchemaRef is the JSON Schema draft 2020-12 meta-schema identifier.
const SchemaRef = "https://json-schema.org/draft/2020-12/schema"

var (
	// Definitions contains the generated JSON schema definitions
	Definitions map[string]*Schema
)

// Initialize the global variables
func init() {
	Definitions = make(map[string]*Schema)
}

// NewSchema instantiates a new JSON schema.
func NewSchema() *Schema {
	js := Schema{
		Properties: make(map[string]*Schema),
		Defs:       make(map[string]*Schema),
	}
	return &js
}

// JSON serializes the schema into JSON. It makes sure the "$schema" standard
// field is set if needed prior to delegating to the standard JSON marshaler.
func (s *Schema) JSON() ([]byte, error) {
	if s.Ref == "" {
		s.Schema = SchemaRef
	}
	return json.Marshal(s)
}

// APISchema produces the API JSON hyper schema.
func APISchema(api *expr.APIExpr, r *expr.RootExpr) *Schema {
	for _, res := range r.API.HTTP.Services {
		GenerateServiceDefinition(api, res)
	}
	href := string(api.Servers[0].Hosts[0].URIs[0])
	links := []*Link{
		{
			Href: href,
			Rel:  "self",
		},
		{
			Href:   "/schema",
			Method: "GET",
			Rel:    "self",
			TargetSchema: &Schema{
				Schema:               SchemaRef,
				AdditionalProperties: true,
			},
		},
	}
	s := Schema{
		ID:          fmt.Sprintf("%s/schema", href),
		Title:       api.Title,
		Description: api.Description,
		Type:        Object,
		Defs:        Definitions,
		Properties:  propertiesFromDefs(Definitions, "#/$defs/"),
		Links:       links,
	}
	return &s
}

// GenerateServiceDefinition produces the JSON schema corresponding to the given
// service. It stores the results in Definitions.
func GenerateServiceDefinition(api *expr.APIExpr, res *expr.HTTPServiceExpr) {
	s := NewSchema()
	s.Description = res.Description()
	s.Type = Object
	s.Title = res.Name()
	Definitions[res.Name()] = s
	for _, a := range res.HTTPEndpoints {
		var requestSchema *Schema
		if a.MethodExpr.Payload.Type != expr.Empty {
			requestSchema = AttributeTypeSchema(api, a.MethodExpr.Payload)
			requestSchema.Description = a.Name() + " payload"
		}
		var targetSchema *Schema
		var identifier string
		for _, resp := range a.Responses {
			dt := resp.Body.Type
			if mt := dt.(*expr.ResultTypeExpr); mt != nil {
				if identifier == "" {
					identifier = mt.Identifier
				} else {
					identifier = ""
				}
				switch {
				case targetSchema == nil:
					targetSchema = TypeSchemaWithPrefix(api, mt, a.Name())
				case targetSchema.AnyOf == nil:
					firstSchema := targetSchema
					targetSchema = NewSchema()
					targetSchema.AnyOf = []*Schema{firstSchema, TypeSchemaWithPrefix(api, mt, a.Name())}
				default:
					targetSchema.AnyOf = append(targetSchema.AnyOf, TypeSchemaWithPrefix(api, mt, a.Name()))
				}
			}
		}
		for i, r := range a.Routes {
			for j, href := range toSchemaHrefs(r) {
				link := Link{
					Title:        a.Name(),
					Rel:          a.Name(),
					Href:         href,
					Method:       r.Method,
					Schema:       requestSchema,
					TargetSchema: targetSchema,
					ResultType:   identifier,
				}
				if i == 0 && j == 0 {
					if ca := a.Service.CanonicalEndpoint(); ca != nil {
						if ca.Name() == a.Name() {
							link.Rel = "self"
						}
					}
				}
				s.Links = append(s.Links, &link)
			}
		}
	}
}

// ResultTypeRef produces the JSON reference to the media type definition with
// the given view.
func ResultTypeRef(api *expr.APIExpr, mt *expr.ResultTypeExpr, view string) string {
	return ResultTypeRefWithPrefix(api, mt, view, "")
}

// ResultTypeRefWithPrefix produces the JSON reference to the media type definition with
// the given view and adds the provided prefix to the type name
func ResultTypeRefWithPrefix(api *expr.APIExpr, mt *expr.ResultTypeExpr, view, prefix string) string {
	projected, err := expr.Project(mt, view)
	if err != nil {
		panic(codegen.NewError(nil, mt, fmt.Errorf("failed to project media type %#v: %w", mt.Identifier, err)))
	}
	var metaName string
	if n, ok := mt.Meta["openapi:typename"]; ok {
		metaName = codegen.Goify(n[0], true)
	}
	if metaName != "" {
		projected.TypeName = metaName
	}
	if _, ok := Definitions[projected.TypeName]; !ok {
		projected.TypeName = codegen.Goify(prefix, true) + codegen.Goify(projected.TypeName, true)
		if metaName != "" {
			projected.TypeName = metaName
		}
		GenerateResultTypeDefinition(api, projected, expr.DefaultView)
	}
	return fmt.Sprintf("#/$defs/%s", projected.TypeName)
}

// TypeRef produces the JSON reference to the type definition.
func TypeRef(api *expr.APIExpr, ut *expr.UserTypeExpr) string {
	return TypeRefWithPrefix(api, ut, "")
}

// TypeRefWithPrefix produces the JSON reference to the type definition and adds the provided prefix
// to the type name
func TypeRefWithPrefix(api *expr.APIExpr, ut *expr.UserTypeExpr, prefix string) string {
	typeName := ut.TypeName
	if prefix != "" {
		typeName = codegen.Goify(prefix, true) + codegen.Goify(ut.TypeName, true)
	}
	if n, ok := ut.Meta["openapi:typename"]; ok {
		typeName = codegen.Goify(n[0], true)
	}
	if _, ok := Definitions[typeName]; !ok {
		GenerateTypeDefinitionWithName(api, ut, typeName)
	}
	return fmt.Sprintf("#/$defs/%s", typeName)
}

// GenerateResultTypeDefinition produces the JSON schema corresponding to the
// given media type and given view.
func GenerateResultTypeDefinition(api *expr.APIExpr, mt *expr.ResultTypeExpr, view string) {
	if _, ok := Definitions[mt.TypeName]; ok {
		return
	}
	s := NewSchema()
	s.Title = fmt.Sprintf("Mediatype identifier: %s", mt.Identifier)
	Definitions[mt.TypeName] = s
	buildResultTypeSchema(api, mt, view, s)
}

// GenerateTypeDefinition produces the JSON schema corresponding to the given
// type.
func GenerateTypeDefinition(api *expr.APIExpr, ut *expr.UserTypeExpr) {
	GenerateTypeDefinitionWithName(api, ut, ut.TypeName)
}

// GenerateTypeDefinitionWithName produces the JSON schema corresponding to the given
// type with provided type name.
func GenerateTypeDefinitionWithName(api *expr.APIExpr, ut *expr.UserTypeExpr, typeName string) {
	if _, ok := Definitions[typeName]; ok {
		return
	}
	s := NewSchema()

	s.Title = typeName
	Definitions[typeName] = s
	buildAttributeSchema(api, s, ut.AttributeExpr)
}

// TypeSchema produces the JSON schema corresponding to the given data type.
func TypeSchema(api *expr.APIExpr, t expr.DataType) *Schema {
	return TypeSchemaWithPrefix(api, t, "")
}

// TypeSchemaWithPrefix produces the JSON schema corresponding to the given data type
// and adds the provided prefix to the type name
func TypeSchemaWithPrefix(api *expr.APIExpr, t expr.DataType, prefix string) *Schema {
	s := NewSchema()
	switch actual := t.(type) {
	case expr.Primitive:
		buildPrimitiveTypeSchema(s, actual)
	case *expr.Array:
		buildArrayTypeSchema(api, s, actual)
	case *expr.Object:
		buildObjectTypeSchema(api, s, actual)
	case *expr.Map:
		buildMapTypeSchema(api, s, actual)
	case *expr.Union:
		buildUnionTypeSchema(api, s, actual, prefix)
	case *expr.UserTypeExpr:
		s.Ref = TypeRefWithPrefix(api, actual, prefix)
	case *expr.ResultTypeExpr:
		// Use "default" view by default
		s.Ref = ResultTypeRefWithPrefix(api, actual, expr.DefaultView, prefix)
	}
	return s
}

func buildPrimitiveTypeSchema(s *Schema, primitive expr.Primitive) {
	s.Type = Type(primitive.Name())
	switch primitive.Kind() {
	case expr.AnyKind:
		s.Type = Type("")
	case expr.IntKind, expr.Int64Kind, expr.UIntKind, expr.UInt64Kind:
		s.Type = Type("integer")
		s.Format = "int64"
	case expr.Int32Kind, expr.UInt32Kind:
		s.Type = Type("integer")
		s.Format = "int32"
	case expr.Float32Kind:
		s.Type = Type("number")
		s.Format = "float"
	case expr.Float64Kind:
		s.Type = Type("number")
		s.Format = "double"
	case expr.BytesKind:
		s.Type = Type("string")
		s.Format = "byte"
	}
}

func buildArrayTypeSchema(api *expr.APIExpr, s *Schema, arr *expr.Array) {
	s.Type = Array
	s.Items = NewSchema()
	buildAttributeSchema(api, s.Items, arr.ElemType)
}

func buildObjectTypeSchema(api *expr.APIExpr, s *Schema, obj *expr.Object) {
	s.Type = Object
	for _, nat := range *obj {
		if !MustGenerate(nat.Attribute.Meta) {
			continue
		}
		prop := NewSchema()
		buildAttributeSchema(api, prop, nat.Attribute)
		s.Properties[nat.Name] = prop
	}
}

func buildMapTypeSchema(api *expr.APIExpr, s *Schema, m *expr.Map) {
	s.Type = Object
	if m.KeyType.Type == expr.String && m.ElemType.Type != expr.Any {
		additionalProperties := NewSchema()
		s.AdditionalProperties = buildAttributeSchema(api, additionalProperties, m.ElemType)
		return
	}
	s.AdditionalProperties = true
}

func buildUnionTypeSchema(api *expr.APIExpr, s *Schema, union *expr.Union, prefix string) {
	typeKey := union.GetTypeKey()
	valueKey := union.GetValueKey()
	s.Type = Object
	s.Discriminator = &Discriminator{PropertyName: typeKey}
	for _, val := range union.Values {
		s.OneOf = append(s.OneOf, buildUnionBranchSchema(api, typeKey, valueKey, val, prefix))
	}
}

func buildUnionBranchSchema(api *expr.APIExpr, typeKey, valueKey string, val *expr.NamedAttributeExpr, prefix string) *Schema {
	tag := expr.UnionVariantTag(val)
	branch := NewSchema()
	branch.Type = Object
	branch.Properties = map[string]*Schema{
		typeKey: {
			Type: String,
			Enum: []any{tag},
		},
		valueKey: AttributeTypeSchemaWithPrefix(api, val.Attribute, prefix),
	}
	branch.Required = []string{typeKey, valueKey}
	return branch
}

// AttributeTypeSchema produces the JSON schema corresponding to the given attribute.
func AttributeTypeSchema(api *expr.APIExpr, at *expr.AttributeExpr) *Schema {
	return AttributeTypeSchemaWithPrefix(api, at, "")
}

// AttributeTypeSchemaWithPrefix produces the JSON schema corresponding to the given attribute
// and adds the provided prefix to the type name
func AttributeTypeSchemaWithPrefix(api *expr.APIExpr, at *expr.AttributeExpr, prefix string) *Schema {
	s := TypeSchemaWithPrefix(api, at.Type, prefix)
	initAttributeValidation(s, at)
	return s
}

// MarshalJSON returns the JSON encoding of s.
func (s *Schema) MarshalJSON() ([]byte, error) {
	return MarshalJSON((*_Schema)(s), s.Extensions)
}

// MarshalYAML returns value which marshaled in place of the original value
func (s *Schema) MarshalYAML() (any, error) {
	return MarshalYAML((*_Schema)(s), s.Extensions)
}

// Dup creates a shallow clone of the given schema.
func (s *Schema) Dup() *Schema {
	js := Schema{
		ID:                   s.ID,
		Description:          s.Description,
		Schema:               s.Schema,
		Type:                 s.Type,
		DefaultValue:         s.DefaultValue,
		Title:                s.Title,
		Media:                s.Media,
		ReadOnly:             s.ReadOnly,
		WriteOnly:            s.WriteOnly,
		Deprecated:           s.Deprecated,
		ContentEncoding:      s.ContentEncoding,
		ContentMediaType:     s.ContentMediaType,
		PathStart:            s.PathStart,
		Links:                s.Links,
		Ref:                  s.Ref,
		Enum:                 s.Enum,
		Format:               s.Format,
		Pattern:              s.Pattern,
		Minimum:              s.Minimum,
		Maximum:              s.Maximum,
		MinLength:            s.MinLength,
		MaxLength:            s.MaxLength,
		MinItems:             s.MinItems,
		MaxItems:             s.MaxItems,
		Required:             s.Required,
		AdditionalProperties: s.AdditionalProperties,
	}
	for n, p := range s.Properties {
		js.Properties[n] = p.Dup()
	}
	if s.Items != nil {
		js.Items = s.Items.Dup()
	}
	for n, d := range s.Defs {
		js.Defs[n] = d.Dup()
	}
	return &js
}

// buildAttributeSchema initializes the given JSON schema that corresponds to
// the given attribute.
func buildAttributeSchema(api *expr.APIExpr, s *Schema, at *expr.AttributeExpr) *Schema {
	s.Merge(TypeSchema(api, at.Type))
	if s.Ref != "" {
		s.Extensions = MergeExtensions(
			ExtensionsFromExpr(at.Meta),
			ScopedExtensionsFromExpr(at.Meta, "schema"),
		)
		applyNullableSchema(s, at.Meta)
		return s
	}
	s.DefaultValue = ToStringMap(at.DefaultValue)
	s.Description = at.Description
	s.Example = expr.CanonicalizeExample(at, at.Example(api.ExampleGenerator))
	s.Extensions = MergeExtensions(
		ExtensionsFromExpr(at.Meta),
		ScopedExtensionsFromExpr(at.Meta, "schema"),
	)
	applySchemaOpenAPIMetadata(s, at.Meta)
	if ap := AdditionalPropertiesFromExpr(at.Meta); ap != nil {
		s.AdditionalProperties = ap
	}
	initAttributeValidation(s, at)
	applyNullableSchema(s, at.Meta)

	return s
}

func applyNullableSchema(schema *Schema, meta expr.MetaExpr) {
	value, ok := meta.Last("openapi:nullable")
	if !ok || value == "false" || schema == nil || schema.Type == Null {
		return
	}
	base := *schema
	base.Extensions = nil
	*schema = Schema{
		AnyOf: []*Schema{
			&base,
			{Type: Null},
		},
		Extensions: schema.Extensions,
	}
}

// initAttributeValidation initializes validation rules for an attribute.
func initAttributeValidation(s *Schema, at *expr.AttributeExpr) {
	val := at.Validation
	if val == nil {
		return
	}
	s.Enum = val.Values
	if val.Format != "" {
		s.Format = string(val.Format)
	}
	s.Pattern = val.Pattern
	if val.ExclusiveMinimum != nil {
		s.ExclusiveMinimum = val.ExclusiveMinimum
	}
	if val.Minimum != nil {
		s.Minimum = val.Minimum
	}
	if val.ExclusiveMaximum != nil {
		s.ExclusiveMaximum = val.ExclusiveMaximum
	}
	if val.Maximum != nil {
		s.Maximum = val.Maximum
	}
	if val.MinLength != nil {
		if _, ok := at.Type.(*expr.Array); ok {
			s.MinItems = val.MinLength
		} else {
			s.MinLength = val.MinLength
		}
	}
	if val.MaxLength != nil {
		if _, ok := at.Type.(*expr.Array); ok {
			s.MaxItems = val.MaxLength
		} else {
			s.MaxLength = val.MaxLength
		}
	}
	for _, v := range val.Required {
		if a := at.Find(v); a != nil {
			if !MustGenerate(a.Meta) {
				continue
			}
		}
		s.Required = append(s.Required, v)
	}
}

// toSchemaHrefs produces hrefs that replace the path wildcards with JSON
// schema references when appropriate.
func toSchemaHrefs(r *expr.RouteExpr) []string {
	paths := r.FullPaths()
	res := make([]string, len(paths))
	for i, path := range paths {
		params := expr.ExtractHTTPWildcards(path)
		args := make([]any, len(params))
		for j, p := range params {
			args[j] = fmt.Sprintf("/{%s}", p)
		}
		tmpl := expr.HTTPWildcardRegex.ReplaceAllLiteralString(path, "%s")
		res[i] = fmt.Sprintf(tmpl, args...)
	}
	return res
}

// propertiesFromDefs creates a Properties map referencing the given definitions
// under the given path.
func propertiesFromDefs(definitions map[string]*Schema, path string) map[string]*Schema {
	res := make(map[string]*Schema, len(definitions))
	for n := range definitions {
		if n == "identity" {
			continue
		}
		s := NewSchema()
		s.Ref = path + n
		res[n] = s
	}
	return res
}

// buildResultTypeSchema initializes s as the JSON schema representing mt for the
// given view.
func buildResultTypeSchema(api *expr.APIExpr, mt *expr.ResultTypeExpr, view string, s *Schema) {
	s.Media = &Media{Type: mt.Identifier}
	projected, err := expr.Project(mt, view)
	if err != nil {
		panic(codegen.NewError(nil, mt, fmt.Errorf("failed to project media type %#v: %w", mt.Identifier, err)))
	}
	buildAttributeSchema(api, s, projected.AttributeExpr)
}
