package ir

import (
	"encoding/json/jsontext"
	"fmt"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/internal/examplegen"
	"github.com/CaliLuke/loom/internal/jsonkey"
)

type (
	schemaUsage int

	// AnalyzerOption customizes analyzer behavior.
	AnalyzerOption func(*Analyzer)

	// Analyzer converts expr types into IR schemas and endpoint body models.
	Analyzer struct {
		semantic                *service.SemanticData
		schemaDepth             int
		acquisitionJournal      *acquisitionJournal
		canonicalComparisons    []*canonicalComparison
		err                     error
		discriminatorReferences []discriminatorReference
		discriminatorHints      *discriminatorHintContext
		declarationRoots        map[string]*Schema
		discriminatorChildren   map[string][]string
		// discriminatorFamilies holds the direct child declarations of each
		// hinted parent root component.
		discriminatorFamilies      map[string][]expr.UserType
		declarationTypes           map[string]expr.UserType
		declarationRepresentations map[string]representationContext
		publicationContexts        map[string]string
		// declarationNeighborCache, publicationDependencyCache and
		// constructionContexts memoize immutable facts read during projection,
		// after declarations and discriminator families are recorded and before
		// representation allocation renames components, like publicationContexts.
		declarationNeighborCache   map[string][]expr.UserType
		publicationDependencyCache map[string][]expr.UserType
		constructionContexts       map[schemaPosition]string
		// contributors memoizes declarations projected into HTTP body wire
		// contexts by content identity; projectionWire is the context whose
		// contributors are being analyzed.
		contributors          map[string]*projectedContributor
		componentAcquisitions map[string]*componentAcquisition
		contributorRoot       *projectedContributor
		contributorRootDepth  int
		projectionWire        *representation.WireContext
		// projectedChildren lists, per hinted root component of a projection,
		// the children its representations publish; publishedChildren records
		// the representations already given them.
		projectedChildren          map[string][]*projectedContributor
		publishedChildren          map[[2]string]bool
		asyncAcquisition           *asyncBaselineAcquisition
		componentAnnotations       map[componentAnnotationIdentity]map[componentAnnotationPosition]componentExamples
		annotationOwner            map[componentAnnotationPosition]componentExamples
		opaqueAnnotationOwners     map[expr.DefaultValueOwner]int
		referenceAnnotationTargets map[expr.DataType]*transportir.ValueTarget
		occurrences                []schemaOccurrenceAnalysis
		registrations              []*Schema
		plan                       expr.ValuePlanNode
		examplePlan                expr.ValuePlanNode
		constructingBaseline       bool
		constructions              map[*Schema]schemaConstruction
		projectionPlans            map[schemaProjectionKey]*schemaProjection
		projectionComponents       map[schemaProjectionKey]*componentProjection
		projectionMappings         map[*Discriminator]map[string]*projectedContributor
		schemaNames                map[string]representationComponentName
		schemas                    map[string]*Schema
		schemaFingerprints         map[string]string
		schemaTypeIDs              map[string]string
		schemasByFingerprint       map[string][]schemaRef
		unionBranchSchemas         map[string]string
		closeObjects               bool
		rand                       *expr.ExampleGenerator
		preparedExamples           *transportir.ValueTarget
		parameterSources           map[expr.ValuePlanNode]*expr.AttributeExpr

		exampleValue         func(*expr.AttributeExpr, any) (any, bool)
		customExampleValue   bool
		declaredExampleValue bool
		suppressExamples     func(*expr.AttributeExpr, bool) bool
	}

	schemaRef struct {
		ref            string
		explicitName   string
		canonicalClaim string
		binding        preparedComponentBinding
	}
)

const (
	schemaUsageNeutral schemaUsage = iota
	schemaUsageRequest
	schemaUsageResponse
	projectedResultMetaKey = "loom:openapi:projected-result"
)

// WithServiceSemantics shares captured service occurrences with other backends.
func WithServiceSemantics(semantic *service.SemanticData) AnalyzerOption {
	return func(a *Analyzer) { a.semantic = semantic }
}

// WithExampleValue selects a custom projector over the original attribute and
// raw example. A nil projector uses retained declared JSON-shaped values,
// preserving Bytes as []byte. Custom projector output is the caller's
// responsibility.
func WithExampleValue(fn func(*expr.AttributeExpr, any) (any, bool)) AnalyzerOption {
	return func(a *Analyzer) {
		a.exampleValue = fn
		a.customExampleValue = fn != nil
		a.declaredExampleValue = fn == nil
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

// NewAnalyzer creates a schema analyzer. Builtin examples consume shared
// prepared values and documentation plans. WithExampleValue selects an
// explicit custom projection or retained declared values.
func NewAnalyzer(rand *expr.ExampleGenerator, closeObjects bool, options ...AnalyzerOption) *Analyzer {
	a := &Analyzer{
		declarationRoots:           make(map[string]*Schema),
		discriminatorFamilies:      make(map[string][]expr.UserType),
		declarationTypes:           make(map[string]expr.UserType),
		declarationRepresentations: make(map[string]representationContext),
		publicationContexts:        make(map[string]string),
		declarationNeighborCache:   make(map[string][]expr.UserType),
		publicationDependencyCache: make(map[string][]expr.UserType),
		constructionContexts:       make(map[schemaPosition]string),
		contributors:               make(map[string]*projectedContributor),
		componentAcquisitions:      make(map[string]*componentAcquisition),
		projectedChildren:          make(map[string][]*projectedContributor),
		publishedChildren:          make(map[[2]string]bool),
		discriminatorHints:         &discriminatorHintContext{types: make(map[string]expr.UserType)},
		constructions:              make(map[*Schema]schemaConstruction),
		projectionPlans:            make(map[schemaProjectionKey]*schemaProjection),
		projectionComponents:       make(map[schemaProjectionKey]*componentProjection),
		projectionMappings:         make(map[*Discriminator]map[string]*projectedContributor),
		schemaNames:                make(map[string]representationComponentName),
		schemas:                    make(map[string]*Schema),
		schemaFingerprints:         make(map[string]string),
		schemaTypeIDs:              make(map[string]string),
		schemasByFingerprint:       make(map[string][]schemaRef),
		unionBranchSchemas:         make(map[string]string),
		closeObjects:               closeObjects,

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

// Err returns the first unsupported schema projection encountered by this analyzer.
func (a *Analyzer) Err() error {
	if a.err != nil {
		return a.err
	}
	for _, reference := range a.discriminatorReferences {
		if a.schemas[reference.name] == nil {
			return fmt.Errorf("%s: OpenAPI discriminator defaultMapping %q does not resolve to an emitted component schema", reference.context, reference.name)
		}
	}
	return nil
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
	return a.analyzeStandaloneSchema(attr, context, noref...)
}

// AnalyzeSchemaWithContext builds an IR schema using a stable example occurrence context.
func (a *Analyzer) AnalyzeSchemaWithContext(attr *expr.AttributeExpr, context string, noref ...bool) *Schema {
	if context == "" {
		context = exampleContext("schema", fingerprintAttribute(attr, a.closeObjects))
	}
	return a.analyzeStandaloneSchema(attr, context, noref...)
}

func (a *Analyzer) analyzeStandaloneSchema(attr *expr.AttributeExpr, context string, noref ...bool) *Schema {
	target, err := representation.PrepareStandaloneValues(
		attr,
		examplegen.ForScope(a.rand, "openapi-schema", context), !a.customExampleValue,
	)
	if err != nil {
		a.err = fmt.Errorf("%s: prepare OpenAPI schema values: %w", context, err)
		return nil
	}
	if target == nil {
		return a.analyzeSchema(attr, context, noref...)
	}
	return a.analyzePreparedSchemaPositions(attr, context, target, expr.ValuePlanNode{}, noref...)
}

func (a *Analyzer) analyzeSchema(attr *expr.AttributeExpr, context string, noref ...bool) *Schema {
	defer a.schemaAnalysisScope()()
	if attr == nil || attr.Type == expr.Empty {
		return nil
	}
	if a.plan.Valid() && byteSchemaExcluded(attr) {
		return a.analyzeSchemaPositions(attr, context, expr.ValuePlanNode{}, a.examplePlan, noref...)
	}
	if t, ok := attr.Type.(expr.UserType); ok {
		if a.asyncAcquisition != nil && a.asyncAcquisition.inline() {
			// The inline consumer owns named shape, annotation overrides and
			// null policy before ordinary result-view and occurrence handling.
			// Its original plan remains intact for byte-only projection.
			return a.analyzeNamedUnderlyingSchema(componentAttribute(attr, t), t, context, true)
		}
		s := a.analyzeUserType(attr, t, context, len(noref) > 0, a.takeContributorBinding())
		// A named occurrence's refinement constrains its non-null domain;
		// inherited nullability must surround that complete refinement too.
		if expr.IsNullable(attr) && (!expr.IsNullable(t.Attribute()) ||
			hasUserTypeValidationOverlay(attr, t.Attribute()) || hasEncodingSourceOverlay(attr, t)) {
			a.applySchemaExample(s, attr, context)
			a.applyNullableSchema(s)
		} else if expr.AllowsNull(t.Attribute()) && len(attr.UserExamples) > 0 {
			a.applySchemaExample(s, attr, context)
		}
		return s
	}

	s, note := a.analyzeInlineType(attr, context)
	a.applySchemaAttributeDetails(s, attr, note, context)
	s = a.analyzeCompositionReferences(s, attr, context)
	s = a.applyEncodingSourceType(s, attr)
	if expr.IsNullable(attr) {
		a.applyNullableSchema(s)
	}
	return s
}

// Uniquify returns a stable unique component name. A name reserved for a
// component still under analysis, such as the owner of a recursive variant,
// counts as taken.
func (a *Analyzer) Uniquify(name, fingerprint string) string {
	if !a.componentNameTaken(name) {
		return name
	}
	candidate := name + "_" + fingerprint[:16]
	if !a.componentNameTaken(candidate) {
		return candidate
	}
	for i := 2; ; i++ {
		fallback := fmt.Sprintf("%s_%s_%d", name, fingerprint[:16], i)
		if !a.componentNameTaken(fallback) {
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

// FingerprintAttribute computes the canonical schema fingerprint, including
// checked enum projections retained by prepared attribute declarations.
func (a *Analyzer) FingerprintAttribute(att *expr.AttributeExpr) string {
	encoder := newCanonicalSchemaEncoder()
	encoder.checked = &schemaFingerprintAnalysis{context: expr.NewValueContext()}
	fingerprint := encoder.fingerprint(att, a.closeObjects)
	if encoder.checked.err != nil && a.err == nil {
		a.err = fmt.Errorf("fingerprint OpenAPI schema values: %w", encoder.checked.err)
	}
	return fingerprint
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
	if primitive.Kind() == expr.BytesKind {
		a.analyzeInlineBytes(s, attr, context)
		return
	}
	if schemaType, format, ok := primitiveSchemaTypeFormat(primitive.Kind()); ok {
		s.Type = schemaType
		s.Format = format
		return
	}
	s.Type = primitive.Name()
}

func primitiveSchemaTypeFormat(kind expr.Kind) (schemaType, format string, ok bool) {
	switch kind {
	case expr.BooleanKind:
		return "boolean", "", true
	case expr.IntKind, expr.Int64Kind:
		return "integer", "int64", true
	case expr.UIntKind, expr.UInt64Kind:
		return "integer", "uint64", true
	case expr.Int32Kind:
		return "integer", "int32", true
	case expr.UInt32Kind:
		return "integer", "uint32", true
	case expr.Float32Kind:
		return "number", "float", true
	case expr.Float64Kind:
		return "number", "double", true
	case expr.StringKind:
		return "string", "", true
	case expr.AnyKind:
		return "", "", true
	default:
		return "", "", false
	}
}

func (a *Analyzer) analyzeInlineBytes(s *Schema, attr *expr.AttributeExpr, context string) {
	if bases := attr.Bases; len(bases) > 0 {
		for index, base := range bases {
			baseContext := childExampleContext(context, "base", strconv.Itoa(index))
			s.AnyOf = append(s.AnyOf, a.analyzeReferenceSchema(base, baseContext, false))
		}
		return
	}
	s.Type = "string"
	s.Format = "binary"
}

func (a *Analyzer) analyzeInlineArray(s *Schema, arr *expr.Array, context string) {
	s.Type = string(openapi.Array)
	s.Items = a.analyzeSchemaPositions(
		arr.ElemType,
		childExampleContext(context, "items"),
		a.plan.Element(),
		a.examplePlan.Element(),
	)
}

func (a *Analyzer) analyzeInlineObject(s *Schema, attr *expr.AttributeExpr, obj *expr.Object, context string) {
	s.Type = string(openapi.Object)
	if len(*obj) > 0 {
		s.Properties = make(map[string]*Schema)
	}
	for _, nat := range *obj {
		name := expr.JSONFieldName(expr.ElementName(nat.Name), nat.Attribute)
		if name != "-" && openapi.MustGenerate(nat.Attribute.Meta) {
			s.Properties[name] = a.analyzeSchemaPositions(
				nat.Attribute,
				childExampleContext(context, "property", name),
				valuePlanMemberNode(a.plan, nat.Name),
				valuePlanMemberNode(a.examplePlan, nat.Name),
			)
		}
	}
	// The policy belongs to the object schema itself. Beside a $ref it would
	// see no properties and reject every member, so occurrences select a
	// component variant instead (see expr ownOccurrenceOpenness).
	switch attr.ObjectOpenness() {
	case expr.ObjectTyped:
		// A member that read/write pruning omitted is still declared, so it
		// must not fall under the typed policy. The pruned schema forbids it,
		// as a closed object's additionalProperties: false does.
		for _, name := range usageOmittedMembers(attr) {
			if s.Properties == nil {
				s.Properties = make(map[string]*Schema)
			}
			s.Properties[name] = &Schema{Not: &Schema{}}
		}
		s.AdditionalProperties = &BoolOrSchema{
			Schema: a.analyzeSchemaPositions(
				attr.AdditionalAttributes,
				childExampleContext(context, "additional-properties"),
				a.plan.Additional(),
				a.examplePlan.Additional(),
			),
		}
	case expr.ObjectClosed:
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(false)}
	case expr.ObjectOpennessUnset:
		if a.closeObjects {
			s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(false)}
		}
	}
}

func (a *Analyzer) analyzeInlineMap(s *Schema, m *expr.Map, context string) {
	s.Type = string(openapi.Object)
	if m.ElemType.Type == expr.Any {
		s.AdditionalProperties = &BoolOrSchema{Bool: boolPtr(true)}
		return
	}
	s.AdditionalProperties = &BoolOrSchema{
		Schema: a.analyzeSchemaPositions(
			m.ElemType,
			childExampleContext(context, "additional-properties"),
			a.plan.Element(),
			a.examplePlan.Element(),
		),
	}
}

func (a *Analyzer) analyzeInlineUnion(s *Schema, union *expr.Union, context string) {
	values := sortedUnionValues(union)
	if union.Untagged {
		for _, val := range values {
			s.OneOf = append(s.OneOf, a.analyzeSchemaPositions(
				val.Attribute,
				childExampleContext(context, "branch", expr.UnionVariantTag(val)),
				valuePlanBranchNode(a.plan, expr.UnionVariantTag(val)),
				valuePlanBranchNode(a.examplePlan, expr.UnionVariantTag(val)),
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
	if attr.UnionClosed() {
		s.UnevaluatedProperties = &BoolOrSchema{Bool: boolPtr(false)}
	}
	_, err := expr.EffectiveConstraintsFor(attr)
	if err != nil {
		panic(fmt.Sprintf("invalid effective constraints during OpenAPI analysis: %v", err))
	}
	annotations, err := a.schemaValueAnnotations(attr)
	if err != nil {
		if a.err == nil {
			a.err = fmt.Errorf("%s: %w", context, err)
		}
		return
	}
	if annotations.HasDefault {
		s.DefaultValue = schemaAnnotationValue(annotations.Default)
	}
	a.applySchemaAnnotations(s, attr, note, context)
	if err := applySchemaValidation(s, attr, attr.Validation, schemaAnnotationEnums(annotations.Enums)); err != nil && a.err == nil {
		a.err = fmt.Errorf("%s: %w", context, err)
	}
}

func (a *Analyzer) applySchemaAnnotations(s *Schema, attr *expr.AttributeExpr, note, context string) {
	s.Title = attr.Title
	s.Description = attr.Description
	if note != "" {
		s.Description += "\n" + note
	}

	a.applySchemaExample(s, attr, context)
	s.Extensions = openapi.MergeExtensions(
		openapi.ExtensionsFromExpr(attr.Meta),
		openapi.ScopedExtensionsFromExpr(attr.Meta, "schema"),
	)
	a.applyDiscriminatorMetadata(s, attr, context)
	applySchemaOpenAPIMetadata(s, attr.Meta)
}

func applyNullableSchema(schema *Schema) {
	if schema == nil || schema.Type == string(openapi.Null) {
		return
	}
	base := *schema
	*schema = *takeSchemaAnnotations(&base)
	schema.AnyOf = []*Schema{&base, {Type: string(openapi.Null)}}
}

// takeSchemaAnnotations moves occurrence annotations onto a new wrapper. The
// wrapped assertion retains only constraints, so null and source-type wrappers
// cannot accidentally limit annotations to a non-null or selected branch.
func takeSchemaAnnotations(schema *Schema) *Schema {
	wrapper := &Schema{
		Title:        schema.Title,
		Description:  schema.Description,
		DefaultValue: schema.DefaultValue,
		Example:      schema.Example,
		Examples:     schema.Examples,
		ReadOnly:     schema.ReadOnly,
		WriteOnly:    schema.WriteOnly,
		Deprecated:   schema.Deprecated,
		XML:          schema.XML,
		Extensions:   schema.Extensions,
	}
	schema.Title = ""
	schema.Description = ""
	schema.DefaultValue = nil
	schema.Example = nil
	schema.Examples = nil
	schema.ReadOnly = false
	schema.WriteOnly = false
	schema.Deprecated = false
	schema.XML = nil
	schema.Extensions = nil
	return wrapper
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

func toRef(name string) string {
	return "#" + string(jsontext.Pointer("/components/schemas").AppendToken(name))
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
	value, err := jsonkey.Value(val)
	if err != nil {
		panic(fmt.Errorf("openapi: %w", err))
	}
	return value
}

func (a *Analyzer) componentNameTaken(name string) bool {
	_, analyzed := a.schemas[name]
	_, reserved := a.schemaFingerprints[name]
	return analyzed || reserved
}
