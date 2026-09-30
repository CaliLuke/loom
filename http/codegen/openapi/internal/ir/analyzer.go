package ir

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/internal/enumvalue"
	"github.com/CaliLuke/loom/internal/jsonkey"
)

type (
	schemaUsage int

	// AnalyzerOption customizes analyzer behavior.
	AnalyzerOption func(*Analyzer)

	// Analyzer converts expr types into IR schemas and endpoint body models.
	Analyzer struct {
		declarationRoots     map[string]*Schema
		asyncAcquisition     *asyncBaselineAcquisition
		componentAnnotations map[componentAnnotationIdentity]map[componentAnnotationPosition]any
		annotationOwner      map[componentAnnotationPosition]any
		occurrences          []schemaOccurrenceAnalysis
		registrations        []*Schema
		plan                 expr.ValuePlanNode
		constructingBaseline bool
		constructions        map[*Schema]schemaConstruction
		projectedSchemas     map[schemaProjectionKey]*Schema
		projectedComponents  map[schemaProjectionKey]string
		schemaNames          map[string]representationComponentName
		schemas              map[string]*Schema
		schemaFingerprints   map[string]string
		schemaTypeIDs        map[string]string
		schemasByFingerprint map[string][]schemaRef
		unionBranchSchemas   map[string]string
		closeObjects         bool
		rand                 *expr.ExampleGenerator

		exampleValue       func(*expr.AttributeExpr, any) (any, bool)
		customExampleValue bool
		suppressExamples   func(*expr.AttributeExpr, bool) bool
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

const projectedResultMetaKey = "loom:openapi:projected-result"

// WithExampleValue projects raw expr examples into OpenAPI-safe values.
func WithExampleValue(fn func(*expr.AttributeExpr, any) (any, bool)) AnalyzerOption {
	return func(a *Analyzer) {
		a.exampleValue = fn
		a.customExampleValue = fn != nil
	}
}

// WithExampleSuppression controls whether examples should be omitted.
func WithExampleSuppression(fn func(*expr.AttributeExpr, bool) bool) AnalyzerOption {
	return func(a *Analyzer) {
		a.suppressExamples = fn
	}
}

// WithExampleGenerator overrides the generator used for synthesized examples.
func WithExampleGenerator(generator *expr.ExampleGenerator) AnalyzerOption {
	return func(a *Analyzer) {
		a.rand = generator
	}
}

// NewAnalyzer creates a schema analyzer. Examples use OpenAPIExampleValue by
// default, including byte encoding and completeness checks. WithExampleValue
// overrides this projection when a caller needs another representation.
func NewAnalyzer(rand *expr.ExampleGenerator, closeObjects bool, options ...AnalyzerOption) *Analyzer {
	a := &Analyzer{
		declarationRoots:     make(map[string]*Schema),
		constructions:        make(map[*Schema]schemaConstruction),
		projectedSchemas:     make(map[schemaProjectionKey]*Schema),
		projectedComponents:  make(map[schemaProjectionKey]string),
		schemaNames:          make(map[string]representationComponentName),
		schemas:              make(map[string]*Schema),
		schemaFingerprints:   make(map[string]string),
		schemaTypeIDs:        make(map[string]string),
		schemasByFingerprint: make(map[string][]schemaRef),
		unionBranchSchemas:   make(map[string]string),
		closeObjects:         closeObjects,

		rand:         rand,
		exampleValue: OpenAPIExampleValue,
	}
	for _, opt := range options {
		opt(a)
	}
	return a
}

func exampleGeneratorWithOptions(
	generator *expr.ExampleGenerator,
	options ...AnalyzerOption,
) *expr.ExampleGenerator {
	settings := &Analyzer{rand: generator}
	for _, option := range options {
		option(settings)
	}
	return settings.rand
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
	context := exampleContext("schema", fingerprintAttribute(attr, a.closeObjects))
	return a.analyzeSchema(attr, context, noref...)
}

// AnalyzeSchemaWithContext builds an IR schema using a stable example occurrence context.
func (a *Analyzer) AnalyzeSchemaWithContext(attr *expr.AttributeExpr, context string, noref ...bool) *Schema {
	if context == "" {
		context = exampleContext("schema", fingerprintAttribute(attr, a.closeObjects))
	}
	return a.analyzeSchema(attr, context, noref...)
}

func (a *Analyzer) analyzeSchema(attr *expr.AttributeExpr, context string, noref ...bool) *Schema {
	if attr == nil || attr.Type == expr.Empty {
		return nil
	}
	if a.plan.Valid() && byteSchemaExcluded(attr) {
		return a.analyzeSchemaPlan(attr, context, expr.ValuePlanNode{}, noref...)
	}
	if t, ok := attr.Type.(expr.UserType); ok {
		if a.asyncAcquisition != nil && a.asyncAcquisition.inline() {
			// The inline consumer owns named shape, annotation overrides and
			// null policy before ordinary result-view and occurrence handling.
			// Its original plan remains intact for byte-only projection.
			return a.analyzeUnderlyingSchema(componentAttribute(attr, t), context, true)
		}
		s := a.analyzeUserType(attr, t, context, len(noref) > 0)
		if attr.Nullable && !expr.IsNullable(t.Attribute()) {
			a.applySchemaExample(s, attr, context)
			a.applyNullableSchema(s)
		} else if expr.AllowsNull(t.Attribute()) && len(attr.UserExamples) > 0 {
			a.applySchemaExample(s, attr, context)
		}
		return s
	}

	s, note := a.analyzeInlineType(attr, context)
	a.applySchemaAttributeDetails(s, attr, note, context)
	if expr.IsNullable(attr) {
		a.applyNullableSchema(s)
	}
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
		panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(%q, ...) values", name, "openapi:typename"))
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

func (a *Analyzer) analyzeInlineType(attr *expr.AttributeExpr, context string) (*Schema, string) {
	s := &Schema{}
	switch t := attr.Type.(type) {
	case expr.Primitive:
		a.analyzeInlinePrimitive(s, attr, t, context)
	case *expr.Array:
		a.analyzeInlineArray(s, t, context)
	case *expr.Object:
		a.analyzeInlineObject(s, attr, t, context)
	case *expr.Map:
		a.analyzeInlineMap(s, t, context)
	case *expr.Union:
		a.analyzeInlineUnion(s, t, context)
	default:
		panic(fmt.Sprintf("unknown type %T", t))
	}
	a.recordStructuralSchema(s, attr)
	return s, ""
}

func (a *Analyzer) analyzeInlinePrimitive(
	s *Schema,
	attr *expr.AttributeExpr,
	primitive expr.Primitive,
	context string,
) {
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
		a.analyzeInlineBytes(s, attr, context)
	case expr.AnyKind:
		s.Type = ""
	default:
		s.Type = primitive.Name()
	}
}

func (a *Analyzer) analyzeInlineBytes(s *Schema, attr *expr.AttributeExpr, context string) {
	if bases := attr.Bases; len(bases) > 0 {
		for index, base := range bases {
			baseContext := childExampleContext(context, "base", strconv.Itoa(index))
			s.AnyOf = append(s.AnyOf, a.analyzeSchema(&expr.AttributeExpr{Type: base}, baseContext, false))
		}
		return
	}
	s.Type = "string"
	s.Format = "binary"
}

func (a *Analyzer) analyzeInlineArray(s *Schema, arr *expr.Array, context string) {
	s.Type = string(openapi.Array)
	s.Items = a.analyzeSchemaPlan(arr.ElemType, childExampleContext(context, "items"), a.plan.Element())
}

func (a *Analyzer) analyzeInlineObject(s *Schema, attr *expr.AttributeExpr, obj *expr.Object, context string) {
	s.Type = string(openapi.Object)
	if len(*obj) > 0 {
		s.Properties = make(map[string]*Schema)
	}
	for _, nat := range *obj {
		name := expr.JSONFieldName(expr.ElementName(nat.Name), nat.Attribute)
		if name != "-" && openapi.MustGenerate(nat.Attribute.Meta) {
			s.Properties[name] = a.analyzeSchemaPlan(
				nat.Attribute,
				childExampleContext(context, "property", name), a.memberPlan(nat.Name),
			)
		}
	}
	if a.closeObjects && openapi.AdditionalPropertiesFromExpr(attr.Meta) == nil {
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(false)}
	}
}

func (a *Analyzer) analyzeInlineMap(s *Schema, m *expr.Map, context string) {
	s.Type = string(openapi.Object)
	if m.ElemType.Type == expr.Any {
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(true)}
		return
	}
	s.AdditionalProperties = &BoolOrSchema{
		Schema: a.analyzeSchemaPlan(m.ElemType, childExampleContext(context, "additional-properties"), a.plan.Element()),
	}
}

func (a *Analyzer) analyzeInlineUnion(s *Schema, union *expr.Union, context string) {
	values := sortedUnionValues(union)
	if union.Untagged {
		for _, val := range values {
			s.OneOf = append(s.OneOf, a.analyzeSchemaPlan(
				val.Attribute,
				childExampleContext(context, "branch", expr.UnionVariantTag(val)), a.branchPlan(expr.UnionVariantTag(val)),
			))
		}
		return
	}
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

func (a *Analyzer) applySchemaAttributeDetails(s *Schema, attr *expr.AttributeExpr, note, context string) {
	_, err := expr.EffectiveConstraintsFor(attr)
	if err != nil {
		panic(fmt.Sprintf("invalid effective constraints during OpenAPI analysis: %v", err))
	}
	a.applySchemaAnnotations(s, attr, note, context)
	applySchemaValidation(s, attr, attr.Validation)
}

func (a *Analyzer) applySchemaAnnotations(s *Schema, attr *expr.AttributeExpr, note, context string) {
	s.Title = attr.Title
	s.Description = attr.Description
	if note != "" {
		s.Description += "\n" + note
	}
	s.DefaultValue = toStringMap(normalizeOpenAPIExampleForAttribute(attr, projectOpenAPIExample(attr, enumvalue.Normalize(attr, attr.DefaultValue))))

	a.applySchemaExample(s, attr, context)
	s.Extensions = openapi.MergeExtensions(
		openapi.ExtensionsFromExpr(attr.Meta),
		openapi.ScopedExtensionsFromExpr(attr.Meta, "schema"),
	)
	applySchemaOpenAPIMetadata(s, attr.Meta)
	if ap := openapi.AdditionalPropertiesFromExpr(attr.Meta); ap != nil {
		if explicit, ok := ap.(bool); ok {
			s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(explicit)}
		}
	}
}

func (a *Analyzer) applySchemaExample(s *Schema, attr *expr.AttributeExpr, context string) {
	if a.annotationOwner != nil {
		position := componentAnnotationPosition{context, fingerprintAttribute(attr, a.closeObjects)}
		if example, found := a.annotationOwner[position]; found {
			s.Example = copyComponentAnnotation(example)
			return
		}
		defer func() {
			a.annotationOwner[position] = copyComponentAnnotation(s.Example)
		}()
	}
	suppress := false
	if a.suppressExamples != nil {
		suppress = a.suppressExamples(attr, a.closeObjects)
	}
	if !suppress {
		generator := exampleGeneratorForAttribute(a.rand, attr, a.closeObjects, context)
		if a.customExampleValue {
			raw := attr.Example(generator)
			if example, ok := a.exampleValue(attr, raw); ok {
				s.Example = example
			}
			return
		}
		source := synthesizedOpenAPIExample(attr, generator)
		if !source.present {
			return
		}
		if a.exampleValue == nil {
			if source.declared {
				s.Example = source.value
			} else {
				s.Example = expr.CanonicalizeExample(attr, source.value)
			}
			return
		}
		if source.declared {
			if example, ok := openAPIDeclaredExampleValue(attr, source.value); ok {
				s.Example = example
			}
			return
		}
		if example, ok := a.exampleValue(attr, source.value); ok {
			s.Example = example
		}
	}
}

func applyNullableSchema(schema *Schema) {
	if schema == nil || schema.Type == string(openapi.Null) {
		return
	}
	base := *schema
	base.Title = ""
	base.Description = ""
	base.DefaultValue = nil
	base.Example = nil
	base.ReadOnly = false
	base.WriteOnly = false
	base.Deprecated = false
	base.XML = nil
	base.Extensions = nil
	*schema = Schema{
		Title:        schema.Title,
		Description:  schema.Description,
		DefaultValue: schema.DefaultValue,
		Example:      schema.Example,
		ReadOnly:     schema.ReadOnly,
		WriteOnly:    schema.WriteOnly,
		Deprecated:   schema.Deprecated,
		AnyOf: []*Schema{
			&base,
			{Type: string(openapi.Null)},
		},
		XML:        schema.XML,
		Extensions: schema.Extensions,
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
	// Synthetic envelopes retain component annotation authority. Their payloads
	// are not inline positions in the enclosing message's acquisition scope.
	acquisition := a.asyncAcquisition
	a.asyncAcquisition = nil
	defer func() {
		a.asyncAcquisition = acquisition
	}()
	logical := a.unionBranchSchemaKey(union, val)
	projection := a.byteProjection(val.Attribute, a.branchPlan(expr.UnionVariantTag(val)))
	key := logical + projection
	if name, ok := a.unionBranchSchemas[key]; ok {
		return toRef(name)
	}

	desired := deterministicUnionBranchSchemaName(union, val)
	name := desired
	fingerprint := fingerprintString(key)
	name = a.Uniquify(name, fingerprint)
	a.unionBranchSchemas[key] = name
	a.schemaNames[name] = representationComponentName{desired: desired, logical: logical, baseline: logical, json: projection != ""}

	restore := a.componentAnnotationScope(logical, logical, exampleContext("component", desired))
	defer restore()
	branchSchema := &Schema{
		Type:        string(openapi.Object),
		Description: syntheticUnionBranchSchemaDescription(val),
		Properties: map[string]*Schema{
			union.GetTypeKey(): {
				Type: string(openapi.String),
				Enum: []any{expr.UnionVariantTag(val)},
			},
			union.GetValueKey(): a.analyzeSchemaPlan(
				val.Attribute,
				childExampleContext(exampleContext("component", desired), "property", union.GetValueKey()), a.branchPlan(expr.UnionVariantTag(val)),
			),
		},
		Required: []string{union.GetTypeKey(), union.GetValueKey()},
	}
	a.constructions[branchSchema] = schemaConstruction{slots: []schemaSlot{{location: schemaProperty, name: union.GetValueKey()}}}
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

// toStringMap recursively canonicalizes map keys using the same member names
// as authored examples. Byte slices retain their JSON encoding semantics.
func toStringMap(val any) any {
	actual := reflect.ValueOf(val)
	if !actual.IsValid() {
		return nil
	}
	if jsonkey.HasCustomEncoding(actual.Type()) {
		return val
	}
	switch actual.Kind() {
	case reflect.Map:
		out := make(map[string]any, actual.Len())
		iterator := actual.MapRange()
		for iterator.Next() {
			if jsonkey.HasCustomEncoding(reflect.TypeOf(iterator.Key().Interface())) {
				return val
			}
			name, ok := jsonkey.Name(iterator.Key())
			if !ok {
				panic(fmt.Sprintf("openapi: unsupported map key %T", iterator.Key().Interface()))
			}
			out[name] = toStringMap(iterator.Value().Interface())
		}
		return out
	case reflect.Slice, reflect.Array:
		if actual.Kind() == reflect.Slice && actual.Type().Elem().Kind() == reflect.Uint8 {
			return val
		}
		out := make([]any, actual.Len())
		for index := range out {
			out[index] = toStringMap(actual.Index(index).Interface())
		}
		return out
	default:
		return val
	}
}
