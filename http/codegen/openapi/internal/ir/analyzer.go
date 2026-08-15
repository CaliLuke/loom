package ir

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

type (
	schemaUsage int

	// AnalyzerOption customizes analyzer behavior.
	AnalyzerOption func(*Analyzer)

	// Analyzer converts expr types into IR schemas and endpoint body models.
	Analyzer struct {
		schemas              map[string]*Schema
		schemaFingerprints   map[string]string
		schemasByFingerprint map[string][]schemaRef
		unionBranchSchemas   map[string]string
		closeObjects         bool
		rand                 *expr.ExampleGenerator
		exampleValue         func(*expr.AttributeExpr, any) (any, bool)
		suppressExamples     func(*expr.AttributeExpr, bool) bool
	}

	schemaRef struct {
		ref          string
		explicitName string
	}
)

const (
	schemaUsageNeutral schemaUsage = iota
	schemaUsageRequest
	schemaUsageResponse
)

// WithExampleValue projects raw expr examples into OpenAPI-safe values.
func WithExampleValue(fn func(*expr.AttributeExpr, any) (any, bool)) AnalyzerOption {
	return func(a *Analyzer) {
		a.exampleValue = fn
	}
}

// WithExampleSuppression controls whether examples should be omitted.
func WithExampleSuppression(fn func(*expr.AttributeExpr, bool) bool) AnalyzerOption {
	return func(a *Analyzer) {
		a.suppressExamples = fn
	}
}

// NewAnalyzer creates a schema analyzer.
func NewAnalyzer(rand *expr.ExampleGenerator, closeObjects bool, options ...AnalyzerOption) *Analyzer {
	a := &Analyzer{
		schemas:              make(map[string]*Schema),
		schemaFingerprints:   make(map[string]string),
		schemasByFingerprint: make(map[string][]schemaRef),
		unionBranchSchemas:   make(map[string]string),
		closeObjects:         closeObjects,
		rand:                 rand,
	}
	for _, opt := range options {
		opt(a)
	}
	return a
}

// Components returns the analyzed component schemas.
func (a *Analyzer) Components() map[string]*Schema {
	return a.schemas
}

// SchemaFingerprints returns component fingerprints keyed by component name.
func (a *Analyzer) SchemaFingerprints() map[string]string {
	return a.schemaFingerprints
}

// AnalyzeSchema builds an IR schema for the given attribute.
func (a *Analyzer) AnalyzeSchema(attr *expr.AttributeExpr, noref ...bool) *Schema {
	if attr == nil || attr.Type == expr.Empty {
		return nil
	}
	if t, ok := attr.Type.(expr.UserType); ok {
		return a.analyzeUserType(attr, t, len(noref) > 0)
	}

	s, note := a.analyzeInlineType(attr)
	a.applySchemaAttributeDetails(s, attr, note)
	return s
}

// Uniquify returns a stable unique component name.
func (a *Analyzer) Uniquify(name, fingerprint string) string {
	if _, ok := a.schemas[name]; !ok {
		return name
	}
	candidate := name + "_" + fingerprint[:16]
	if _, ok := a.schemas[candidate]; !ok {
		return candidate
	}
	for i := 2; ; i++ {
		fallback := fmt.Sprintf("%s_%s_%d", name, fingerprint[:16], i)
		if _, ok := a.schemas[fallback]; !ok {
			return fallback
		}
	}
}

// ClaimExplicitName reserves an explicit component name or panics if it conflicts.
func (a *Analyzer) ClaimExplicitName(name, fingerprint string) string {
	if existingFingerprint, ok := a.schemaFingerprints[name]; ok && existingFingerprint != fingerprint {
		panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values", name))
	}
	return name
}

// FingerprintAttribute computes the canonical structural fingerprint for the attribute.
func (a *Analyzer) FingerprintAttribute(att *expr.AttributeExpr) string {
	return fingerprintAttribute(att, a.closeObjects)
}

func mergeStreamingBodyNote(req, streaming *Schema) *Schema {
	if streaming == nil {
		return req
	}
	var note string
	if streaming.Ref != "" {
		note = streaming.Ref
	} else {
		note = streaming.Type
	}
	if req == nil {
		req = streaming
		if req.Description != "" {
			req.Description += "\n"
		}
		req.Description += "Streaming body."
		return req
	}
	if req.Description != "" {
		req.Description += "\n"
	}
	req.Description += fmt.Sprintf("Streaming body: %s", note)
	return req
}

func componentAttribute(attr *expr.AttributeExpr, t expr.UserType) *expr.AttributeExpr {
	componentAttr := expr.DupAtt(t.Attribute())
	if attr == nil {
		return componentAttr
	}
	if componentAttr.Meta == nil {
		componentAttr.Meta = make(expr.MetaExpr)
	}
	for key, values := range attr.Meta {
		componentAttr.Meta[key] = append([]string(nil), values...)
	}
	if attr.Description != "" {
		componentAttr.Description = attr.Description
	}
	if attr.Validation != nil {
		componentAttr.Validation = attr.Validation
	}
	if attr.DefaultValue != nil {
		componentAttr.DefaultValue = attr.DefaultValue
	}
	if len(attr.UserExamples) > 0 {
		componentAttr.UserExamples = attr.UserExamples
	}
	return componentAttr
}

func (a *Analyzer) analyzeInlineType(attr *expr.AttributeExpr) (*Schema, string) {
	s := &Schema{}
	switch t := attr.Type.(type) {
	case expr.Primitive:
		a.analyzeInlinePrimitive(s, attr, t)
	case *expr.Array:
		a.analyzeInlineArray(s, t)
	case *expr.Object:
		a.analyzeInlineObject(s, attr, t)
	case *expr.Map:
		a.analyzeInlineMap(s, t)
	case *expr.Union:
		a.analyzeInlineUnion(s, t)
	default:
		panic(fmt.Sprintf("unknown type %T", t))
	}
	return s, ""
}

func (a *Analyzer) analyzeInlinePrimitive(s *Schema, attr *expr.AttributeExpr, primitive expr.Primitive) {
	switch primitive.Kind() {
	case expr.IntKind, expr.UIntKind, expr.Int64Kind, expr.UInt64Kind:
		s.Type = "integer"
		s.Format = "int64"
	case expr.Int32Kind, expr.UInt32Kind:
		s.Type = "integer"
		s.Format = "int32"
	case expr.Float32Kind:
		s.Type = "number"
		s.Format = "float"
	case expr.Float64Kind:
		s.Type = "number"
		s.Format = "double"
	case expr.BytesKind:
		a.analyzeInlineBytes(s, attr)
	case expr.AnyKind:
		s.Type = ""
	default:
		s.Type = primitive.Name()
	}
}

func (a *Analyzer) analyzeInlineBytes(s *Schema, attr *expr.AttributeExpr) {
	if bases := attr.Bases; len(bases) > 0 {
		for _, base := range bases {
			s.AnyOf = append(s.AnyOf, a.AnalyzeSchema(&expr.AttributeExpr{Type: base}, false))
		}
		return
	}
	s.Type = "string"
	s.Format = "binary"
}

func (a *Analyzer) analyzeInlineArray(s *Schema, arr *expr.Array) {
	s.Type = string(openapi.Array)
	s.Items = a.AnalyzeSchema(arr.ElemType)
}

func (a *Analyzer) analyzeInlineObject(s *Schema, attr *expr.AttributeExpr, obj *expr.Object) {
	s.Type = string(openapi.Object)
	if len(*obj) > 0 {
		s.Properties = make(map[string]*Schema)
	}
	for _, nat := range *obj {
		if openapi.MustGenerate(nat.Attribute.Meta) {
			s.Properties[nat.Name] = a.AnalyzeSchema(nat.Attribute)
		}
	}
	if a.closeObjects && openapi.AdditionalPropertiesFromExpr(attr.Meta) == nil {
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(false)}
	}
}

func (a *Analyzer) analyzeInlineMap(s *Schema, m *expr.Map) {
	s.Type = string(openapi.Object)
	if m.ElemType.Type == expr.Any {
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(true)}
		return
	}
	s.AdditionalProperties = &BoolOrSchema{Schema: a.AnalyzeSchema(m.ElemType)}
}

func (a *Analyzer) analyzeInlineUnion(s *Schema, union *expr.Union) {
	values := sortedUnionValues(union)
	s.Type = string(openapi.Object)
	s.Discriminator = &Discriminator{
		PropertyName: union.GetTypeKey(),
		Mapping:      make(map[string]string, len(values)),
	}
	if a.closeObjects {
		s.UnevaluatedProperties = &BoolOrSchema{Bool: boolPtr(false)}
	}
	for _, val := range values {
		ref := a.ensureUnionBranchSchema(union, val)
		s.OneOf = append(s.OneOf, &Schema{Ref: ref})
		s.Discriminator.Mapping[expr.UnionVariantTag(val)] = ref
	}
}

func (a *Analyzer) analyzeUserType(attr *expr.AttributeExpr, t expr.UserType, noRef bool) *Schema {
	if resultType, ok := t.(*expr.ResultTypeExpr); ok {
		view, hasView := attr.Meta.Last(expr.ViewMetaKey)
		if !hasView {
			view, hasView = resultType.Meta.Last(expr.ViewMetaKey)
		}
		if hasView {
			projected, err := expr.Project(resultType, view)
			if err != nil {
				panic(codegen.NewError(nil, resultType, fmt.Errorf("project OpenAPI result view %q: %w", view, err)))
			}
			projectedAttr := expr.DupAtt(attr)
			projectedAttr.Type = projected
			projectedAttr.Validation = projected.Validation
			delete(projectedAttr.Meta, expr.ViewMetaKey)
			return a.AnalyzeSchema(projectedAttr, noRef)
		}
	}
	metaName, canonical := schemaTypeNaming(attr, t)
	if expr.IsAlias(t) && !canonical {
		return a.AnalyzeSchema(t.Attribute())
	}

	s := &Schema{}
	fingerprint := a.FingerprintAttribute(attr)

	refs, ok := a.schemasByFingerprint[fingerprint]
	if !noRef && ok {
		if ref := findMatchingSchemaRef(refs, metaName, canonical); ref != "" {
			s.Ref = ref
			return s
		}
	}

	typeName := codegen.Goify(schemaTypeName(t, metaName), true)
	if canonical {
		if metaName != "" {
			typeName = metaName
		}
		typeName = a.ClaimExplicitName(typeName, fingerprint)
	} else {
		typeName = a.Uniquify(typeName, fingerprint)
	}
	s.Ref = toRef(typeName)
	a.registerSchemaRef(fingerprint, s.Ref, metaName)
	if _, ok := a.schemas[typeName]; !ok {
		a.schemaFingerprints[typeName] = fingerprint
		componentAttr := componentAttribute(attr, t)
		a.schemas[typeName] = a.AnalyzeSchema(componentAttr, true)
	}
	return s
}

func (a *Analyzer) applySchemaAttributeDetails(s *Schema, attr *expr.AttributeExpr, note string) {
	s.Description = attr.Description
	if note != "" {
		s.Description += "\n" + note
	}
	s.DefaultValue = toStringMap(attr.DefaultValue)

	suppress := false
	if a.suppressExamples != nil {
		suppress = a.suppressExamples(attr, a.closeObjects)
	}
	if !suppress {
		raw := attr.Example(a.rand)
		if a.exampleValue != nil {
			if example, ok := a.exampleValue(attr, raw); ok {
				s.Example = example
			}
		} else if raw != nil {
			s.Example = raw
		}
	}
	s.Extensions = openapi.ExtensionsFromExpr(attr.Meta)
	applySchemaOpenAPIMetadata(s, attr.Meta)
	if ap := openapi.AdditionalPropertiesFromExpr(attr.Meta); ap != nil {
		if explicit, ok := ap.(bool); ok {
			s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(explicit)}
		}
	}

	val := attr.Validation
	if val == nil {
		return
	}
	s.Enum = val.Values
	if val.Format != "" {
		s.Format = string(val.Format)
	}
	s.Pattern = val.Pattern
	s.ExclusiveMinimum = val.ExclusiveMinimum
	s.Minimum = val.Minimum
	s.ExclusiveMaximum = val.ExclusiveMaximum
	s.Maximum = val.Maximum
	if val.MinLength != nil {
		if _, ok := attr.Type.(*expr.Array); ok {
			s.MinItems = val.MinLength
		} else {
			s.MinLength = val.MinLength
		}
	}
	if val.MaxLength != nil {
		if _, ok := attr.Type.(*expr.Array); ok {
			s.MaxItems = val.MaxLength
		} else {
			s.MaxLength = val.MaxLength
		}
	}
	for _, required := range val.Required {
		if child := attr.Find(required); child != nil && !openapi.MustGenerate(child.Meta) {
			continue
		}
		s.Required = append(s.Required, required)
	}
}

func applySchemaOpenAPIMetadata(s *Schema, meta expr.MetaExpr) {
	if s == nil || meta == nil {
		return
	}
	if value, ok := meta.Last("openapi:readOnly"); ok {
		s.ReadOnly = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:writeOnly"); ok {
		s.WriteOnly = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:deprecated"); ok {
		s.Deprecated = metaBoolValue(value)
	}
	if value, ok := meta.Last("openapi:contentEncoding"); ok {
		s.ContentEncoding = value
	}
	if value, ok := meta.Last("openapi:contentMediaType"); ok {
		s.ContentMediaType = value
	}
	if value, ok := meta.Last("openapi:format"); ok {
		s.Format = value
	}
	if value, ok := meta.Last("openapi:discriminator:defaultMapping"); ok && s.Discriminator != nil {
		s.Discriminator.DefaultMapping = value
	}
	if value, ok := meta.Last("openapi:discriminator:optional"); ok && metaBoolValue(value) && s.Discriminator != nil {
		s.Discriminator.Optional = true
		s.Required = slices.DeleteFunc(s.Required, func(name string) bool {
			return name == s.Discriminator.PropertyName
		})
	}
	for key, assign := range map[string]func(*XML, string){
		"openapi:xml:name":      func(xml *XML, value string) { xml.Name = value },
		"openapi:xml:namespace": func(xml *XML, value string) { xml.Namespace = value },
		"openapi:xml:prefix":    func(xml *XML, value string) { xml.Prefix = value },
		"openapi:xml:nodeType":  func(xml *XML, value string) { xml.NodeType = value },
	} {
		if value, ok := meta.Last(key); ok {
			if s.XML == nil {
				s.XML = new(XML)
			}
			assign(s.XML, value)
		}
	}
}

func metaBoolValue(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "true":
		return true
	case "false":
		return false
	default:
		return true
	}
}

func (a *Analyzer) ensureUnionBranchSchema(union *expr.Union, val *expr.NamedAttributeExpr) string {
	key := a.unionBranchSchemaKey(union, val)
	if name, ok := a.unionBranchSchemas[key]; ok {
		return toRef(name)
	}

	name := deterministicUnionBranchSchemaName(union, val)
	fingerprint := fingerprintString(key)
	name = a.Uniquify(name, fingerprint)
	a.unionBranchSchemas[key] = name

	branchSchema := &Schema{
		Type:        string(openapi.Object),
		Description: syntheticUnionBranchSchemaDescription(val),
		Properties: map[string]*Schema{
			union.GetTypeKey(): {
				Type: string(openapi.String),
				Enum: []any{expr.UnionVariantTag(val)},
			},
			union.GetValueKey(): a.AnalyzeSchema(val.Attribute),
		},
		Required: []string{union.GetTypeKey(), union.GetValueKey()},
	}
	a.schemaFingerprints[name] = fingerprint
	a.schemas[name] = branchSchema
	return toRef(name)
}

func (a *Analyzer) unionBranchSchemaKey(union *expr.Union, val *expr.NamedAttributeExpr) string {
	fingerprint := a.FingerprintAttribute(val.Attribute)
	return strings.Join([]string{
		union.GetTypeKey(),
		union.GetValueKey(),
		expr.UnionVariantTag(val),
		fingerprint,
	}, ":")
}

func (a *Analyzer) registerSchemaRef(fingerprint, ref, explicitName string) {
	for _, existing := range a.schemasByFingerprint[fingerprint] {
		if existing.ref == ref && existing.explicitName == explicitName {
			return
		}
	}
	a.schemasByFingerprint[fingerprint] = append(a.schemasByFingerprint[fingerprint], schemaRef{ref: ref, explicitName: explicitName})
}

func toRef(name string) string {
	name = strings.ReplaceAll(name, "~", "~0")
	name = strings.ReplaceAll(name, "/", "~1")
	return fmt.Sprintf("#/components/schemas/%s", name)
}

func mustGenerateType(meta expr.MetaExpr) bool {
	if _, ok := meta["type:generate:force"]; ok {
		return true
	}
	if n, ok := meta.Last("openapi:typename"); ok && n != "" {
		return true
	}
	return false
}

func toStringMap(val any) any {
	switch actual := val.(type) {
	case map[any]any:
		m := make(map[string]any)
		for k, v := range actual {
			m[toString(k)] = toStringMap(v)
		}
		return m
	case []any:
		out := make([]any, len(actual))
		for i, entry := range actual {
			out[i] = toStringMap(entry)
		}
		return out
	default:
		return actual
	}
}

func toString(val any) string {
	switch actual := val.(type) {
	case string:
		return actual
	case int:
		return strconv.Itoa(actual)
	case float64:
		return strconv.FormatFloat(actual, 'f', -1, 64)
	case bool:
		return strconv.FormatBool(actual)
	default:
		panic("unexpected key type")
	}
}
