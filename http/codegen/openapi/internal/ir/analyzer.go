package ir

import (
	"encoding/binary"
	"fmt"
	"hash"
	"hash/fnv"
	"strconv"
	"strings"

	"github.com/gohugoio/hashstructure"

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
		schemas            map[string]*Schema
		schemaHashes       map[string]uint64
		hashes             map[uint64][]schemaRef
		canonicalNames     map[string]uint64
		unionBranchSchemas map[string]string
		closeObjects       bool
		rand               *expr.ExampleGenerator
		exampleValue       func(*expr.AttributeExpr, any) (any, bool)
		suppressExamples   func(*expr.AttributeExpr, bool) bool
	}

	schemaRef struct {
		ref      string
		explicit bool
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
		schemas:            make(map[string]*Schema),
		schemaHashes:       make(map[string]uint64),
		hashes:             make(map[uint64][]schemaRef),
		canonicalNames:     make(map[string]uint64),
		unionBranchSchemas: make(map[string]string),
		closeObjects:       closeObjects,
		rand:               rand,
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

// SchemaHashes returns the tracked component hashes keyed by component name.
func (a *Analyzer) SchemaHashes() map[string]uint64 {
	return a.schemaHashes
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
func (a *Analyzer) Uniquify(name string, h uint64) string {
	if _, ok := a.schemas[name]; !ok {
		return name
	}
	candidate := fmt.Sprintf("%s_%016x", name, h)
	if _, ok := a.schemas[candidate]; !ok {
		return candidate
	}
	for i := 2; ; i++ {
		fallback := fmt.Sprintf("%s_%016x_%d", name, h, i)
		if _, ok := a.schemas[fallback]; !ok {
			return fallback
		}
	}
}

// ClaimExplicitName reserves an explicit component name or panics if it conflicts.
func (a *Analyzer) ClaimExplicitName(name string, h uint64) string {
	if existingHash, ok := a.schemaHashes[name]; ok && existingHash != h {
		panic(fmt.Sprintf("openapi: explicit component name %q is claimed by multiple different schemas; use distinct Meta(\"openapi:typename\", ...) values", name))
	}
	a.canonicalNames[name] = h
	return name
}

// HashAttribute computes a structural hash for the given attribute.
func (*Analyzer) HashAttribute(att *expr.AttributeExpr, h hash.Hash64) uint64 {
	return *hashAttribute(att, h, make(map[string]*uint64))
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
	if expr.IsAlias(t) {
		return a.AnalyzeSchema(t.Attribute())
	}

	s := &Schema{}
	h := a.HashAttribute(attr, fnv.New64())
	metaName, canonical := schemaTypeNaming(attr, t)
	metaRef := toRef(metaName)

	refs, ok := a.hashes[h]
	if !noRef && ok {
		if ref := findMatchingSchemaRef(refs, metaRef, metaName != ""); ref != "" {
			s.Ref = ref
			return s
		}
	}

	typeName := codegen.Goify(schemaTypeName(t, metaName), true)
	if canonical {
		typeName = a.ClaimExplicitName(typeName, h)
	} else {
		typeName = a.Uniquify(typeName, h)
	}
	s.Ref = toRef(typeName)
	a.registerSchemaRef(h, s.Ref, metaName != "")
	if _, ok := a.schemas[typeName]; !ok {
		a.schemaHashes[typeName] = h
		if canonical {
			a.canonicalNames[typeName] = h
		}
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
	hash := hashString(key, fnv.New64())
	name = a.Uniquify(name, hash)
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
	a.schemaHashes[name] = hash
	a.schemas[name] = branchSchema
	return toRef(name)
}

func (a *Analyzer) unionBranchSchemaKey(union *expr.Union, val *expr.NamedAttributeExpr) string {
	hash := a.HashAttribute(val.Attribute, fnv.New64())
	return strings.Join([]string{
		union.GetTypeKey(),
		union.GetValueKey(),
		expr.UnionVariantTag(val),
		strconv.FormatUint(hash, 10),
	}, ":")
}

func (a *Analyzer) registerSchemaRef(h uint64, ref string, explicit bool) {
	for _, existing := range a.hashes[h] {
		if existing.ref == ref && existing.explicit == explicit {
			return
		}
	}
	a.hashes[h] = append(a.hashes[h], schemaRef{ref: ref, explicit: explicit})
}


func toRef(name string) string {
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

func hashAttribute(att *expr.AttributeExpr, h hash.Hash64, seen map[string]*uint64) *uint64 {
	t := att.Type
	if cached, ok := seen[t.Hash()]; ok {
		return cached
	}
	var (
		res uint64
		ptr = &res
	)
	seen[t.Hash()] = ptr

	hv := hashValidation(att.Validation, h)
	switch t.Kind() {
	case expr.ObjectKind:
		o := expr.AsObject(t)
		for _, member := range *o {
			if !openapi.MustGenerate(member.Attribute.Meta) {
				continue
			}
			kh := hashString(member.Name, h)
			vh := hashAttribute(member.Attribute, h, seen)
			*ptr ^= orderedHash(kh, *vh, h)
		}
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.ArrayKind:
		kh := hashString("[]", h)
		vh := hashAttribute(expr.AsArray(t).ElemType, h, seen)
		*ptr = orderedHash(kh, *vh, h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.MapKind:
		m := expr.AsMap(t)
		kh := hashAttribute(m.KeyType, h, seen)
		vh := hashAttribute(m.ElemType, h, seen)
		*ptr = orderedHash(*kh, *vh, h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	case expr.UserTypeKind:
		*ptr = *hashAttribute(t.(expr.UserType).Attribute(), h, seen)
	case expr.ResultTypeKind:
		rt := t.(*expr.ResultTypeExpr)
		*ptr = hashString(rt.Identifier, h)
		if view, ok := rt.Meta.Last(expr.ViewMetaKey); ok {
			*ptr = orderedHash(*ptr, hashString(view, h), h)
		}
	default:
		*ptr = hashString(t.Name(), h)
		if hv != 0 {
			*ptr = orderedHash(*ptr, hv, h)
		}
	}
	return ptr
}

func hashValidation(val *expr.ValidationExpr, h hash.Hash64) uint64 {
	if val == nil {
		return 0
	}
	res, err := hashstructure.Hash(val, &hashstructure.HashOptions{
		Hasher:          h,
		ZeroNil:         false,
		IgnoreZeroValue: true,
		SlicesAsSets:    true,
	})
	if err != nil {
		return 0
	}
	return res
}

func hashString(s string, h hash.Hash64) uint64 {
	h.Reset()
	if _, err := h.Write([]byte(s)); err != nil {
		panic(err)
	}
	return h.Sum64()
}

func orderedHash(aVal, bVal uint64, h hash.Hash64) uint64 {
	h.Reset()
	if err := binary.Write(h, binary.LittleEndian, aVal); err != nil {
		panic(err)
	}
	if err := binary.Write(h, binary.LittleEndian, bVal); err != nil {
		panic(err)
	}
	return h.Sum64()
}

func boolPtr(v bool) *bool {
	return &v
}
