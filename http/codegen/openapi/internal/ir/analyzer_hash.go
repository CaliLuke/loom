package ir

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

type canonicalSchemaEncoder struct {
	bytes.Buffer
	active  map[expr.DataType]int
	depth   int
	checked *schemaFingerprintAnalysis
}

// schemaFingerprintVersion identifies Loom's canonical OpenAPI schema encoding.
// The encoding uses length-prefixed fields, sorted object keys, set semantics
// for enum and required values, and last-write-wins object member semantics.
const (
	schemaFingerprintVersion = "loom-openapi-schema-v3"
	exampleContextVersion    = "loom-openapi-example-context-v1"
)

func fingerprintAttribute(att *expr.AttributeExpr, closeObjects bool) string {
	return newCanonicalSchemaEncoder().fingerprint(att, closeObjects)
}

func exampleGeneratorForAttribute(
	generator *expr.ExampleGenerator,
	att *expr.AttributeExpr,
	closeObjects bool,
	context ...string,
) *expr.ExampleGenerator {
	if generator == nil || generator.Randomizer == nil {
		return generator
	}
	faker, ok := generator.Randomizer.(*expr.FakerRandomizer)
	if !ok {
		return generator
	}
	identity := fingerprintAttribute(att, closeObjects)
	if len(context) > 0 && context[0] != "" {
		identity = context[0] + ":" + identity
	}
	return expr.NewRandom(faker.Seed + ":openapi-example:" + identity)
}

func exampleContext(parts ...string) string {
	encoder := newCanonicalSchemaEncoder()
	encoder.writeString(exampleContextVersion)
	for _, part := range parts {
		encoder.writeString(part)
	}
	sum := sha256.Sum256(encoder.Bytes())
	return hex.EncodeToString(sum[:])
}

func childExampleContext(parent string, parts ...string) string {
	return exampleContext(append([]string{parent}, parts...)...)
}

func attributeExampleContext(
	att *expr.AttributeExpr,
	closeObjects bool,
	parts ...string,
) string {
	parts = append(parts, "schema", fingerprintAttribute(att, closeObjects))
	if att != nil {
		if userType, ok := att.Type.(expr.UserType); ok {
			parts = append(parts, "user-type", userType.ID())
		}
	}
	return exampleContext(parts...)
}

func fingerprintString(value string) string {
	encoder := newCanonicalSchemaEncoder()
	encoder.writeString(schemaFingerprintVersion)
	encoder.writeString("text")
	encoder.writeString(value)
	sum := sha256.Sum256(encoder.Bytes())
	return hex.EncodeToString(sum[:])
}

func (e *canonicalSchemaEncoder) fingerprint(att *expr.AttributeExpr, closeObjects bool) string {
	e.writeString(schemaFingerprintVersion)
	e.writeAttribute(att, closeObjects, e.captureInput(att))
	sum := sha256.Sum256(e.Bytes())
	return hex.EncodeToString(sum[:])
}

func newCanonicalSchemaEncoder() *canonicalSchemaEncoder {
	return &canonicalSchemaEncoder{active: make(map[expr.DataType]int)}
}

func (e *canonicalSchemaEncoder) writeAttribute(att *expr.AttributeExpr, closeObjects bool, input schemaFingerprintInput) {
	if att == nil {
		e.writeString("nil-attribute")
		return
	}
	if _, ok := att.Type.(expr.UserType); ok {
		e.writeType(att, closeObjects, input)
		return
	}
	e.writeString("attribute")
	e.writeTitle(att.Title)
	e.writeType(att, closeObjects, input)
	if att.Type != nil && (att.Type.Kind() == expr.UserTypeKind || att.Type.Kind() == expr.ResultTypeKind) {
		return
	}
	e.writeValidation(att, input)
	if att.Nullable {
		e.writeString("nullable")
	}
}

func hasUserTypeValidationOverlay(attr, base *expr.AttributeExpr) bool {
	if attr.Validation == nil {
		return false
	}
	left := newCanonicalSchemaEncoder()
	left.writeValidation(attr, schemaFingerprintInput{})
	right := newCanonicalSchemaEncoder()
	right.writeValidation(base, schemaFingerprintInput{})
	return !bytes.Equal(left.Bytes(), right.Bytes())
}

func (e *canonicalSchemaEncoder) writeTitle(title string) {
	if title == "" {
		return
	}
	e.writeString("title")
	e.writeString(title)
}

func (e *canonicalSchemaEncoder) writeType(att *expr.AttributeExpr, closeObjects bool, input schemaFingerprintInput) {
	t := att.Type
	if t == nil {
		e.writeString("nil-type")
		return
	}
	if depth, ok := e.active[t]; ok {
		e.writeString("recursive")
		e.writeInt(depth)
		return
	}
	e.active[t] = e.depth
	defer delete(e.active, t)
	if t.Kind() != expr.UserTypeKind {
		e.depth++
		defer func() {
			e.depth--
		}()
	}

	e.writeTypeShape(att, t, closeObjects, input)
}

func (e *canonicalSchemaEncoder) writeTypeShape(att *expr.AttributeExpr, t expr.DataType, closeObjects bool, input schemaFingerprintInput) {
	if e.writePrimitiveType(t.Kind()) {
		return
	}
	switch t.Kind() {
	case expr.BytesKind:
		e.writeString("bytes")
		e.writeInt(len(att.Bases))
		for _, base := range att.Bases {
			attribute := &expr.AttributeExpr{Type: base}
			e.writeAttribute(attribute, closeObjects, e.captureInput(attribute))
		}
	case expr.ArrayKind:
		e.writeString("array")
		element := expr.AsArray(t).ElemType
		e.writeAttribute(element, closeObjects, e.childInput(element, input.occurrence.Element()))
	case expr.MapKind:
		e.writeString("map")
		element := expr.AsMap(t).ElemType
		e.writeAttribute(element, closeObjects, e.childInput(element, input.occurrence.Element()))
	case expr.ObjectKind:
		e.writeObject(att, expr.AsObject(t), closeObjects, input)
	case expr.UserTypeKind:
		original := t.(expr.UserType).Attribute()
		base := *original
		derived := false
		if att.Nullable && !expr.IsNullable(&base) {
			base.Nullable = true
			derived = true
		}
		if att.Title != "" && att.Title != base.Title {
			base.Title = att.Title
		}
		if hasUserTypeValidationOverlay(att, &base) {
			base.Validation = att.Validation
			derived = true
		}
		underlying := e.childInput(original, input.occurrence.Underlying())
		if derived {
			underlying = e.captureInput(&base)
		}
		e.writeAttribute(&base, closeObjects, underlying)
	case expr.ResultTypeKind:
		e.writeString("result-type")
		resultType := t.(*expr.ResultTypeExpr)
		view, ok := att.Meta.Last(expr.ViewMetaKey)
		if !ok {
			view, _ = resultType.Meta.Last(expr.ViewMetaKey)
		}
		e.writeString(view)
		e.writeAttribute(resultType.Attribute(), closeObjects, e.childInput(resultType.Attribute(), input.occurrence.Underlying()))
	case expr.UnionKind:
		e.writeUnion(t.(*expr.Union), closeObjects, input)
	default:
		panic(fmt.Sprintf("openapi: cannot fingerprint type %T", t))
	}
}

func (e *canonicalSchemaEncoder) writePrimitiveType(kind expr.Kind) bool {
	schemaType, format, ok := primitiveSchemaTypeFormat(kind)
	if !ok {
		return false
	}
	e.writePrimitive(schemaType, format)
	return true
}

func (e *canonicalSchemaEncoder) writePrimitive(schemaType, format string) {
	e.writeString("primitive")
	e.writeString(schemaType)
	e.writeString(format)
}

func (e *canonicalSchemaEncoder) writeObject(att *expr.AttributeExpr, object *expr.Object, closeObjects bool, input schemaFingerprintInput) {
	e.writeString("object")
	members := make(map[string]*expr.AttributeExpr, len(*object))
	captured := e.objectMembers(object, input)
	var occurrences map[string]expr.ValueOccurrence
	if captured != nil {
		occurrences = make(map[string]expr.ValueOccurrence, len(captured))
	}
	for index, member := range *object {
		name := expr.JSONFieldName(expr.ElementName(member.Name), member.Attribute)
		if name != "-" && openapi.MustGenerate(member.Attribute.Meta) {
			members[name] = member.Attribute
			if captured != nil {
				occurrences[name] = captured[index].Occurrence
			}
		}
	}
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)
	e.writeInt(len(names))
	for _, name := range names {
		e.writeString(name)
		member := members[name]
		e.writeAttribute(member, closeObjects, e.childInput(member, occurrences[name]))
	}
	explicitAdditionalProperties := openapi.AdditionalPropertiesFromExpr(att.Meta)
	e.writeBool(explicitAdditionalProperties == nil && closeObjects)
	if explicit, ok := explicitAdditionalProperties.(bool); ok {
		e.writeString("explicit-additional-properties")
		e.writeBool(explicit)
	} else {
		e.writeString("implicit-additional-properties")
	}
}

func (e *canonicalSchemaEncoder) writeUnion(union *expr.Union, closeObjects bool, input schemaFingerprintInput) {
	e.writeString("union")
	e.writeString(union.GetTypeKey())
	e.writeBool(union.Untagged)
	e.writeString(union.GetValueKey())
	values := e.unionInputs(union, input)
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].branch.Name != values[j].branch.Name {
			return values[i].branch.Name < values[j].branch.Name
		}
		left := fingerprintAttribute(values[i].branch.Attribute, closeObjects)
		right := fingerprintAttribute(values[j].branch.Attribute, closeObjects)
		return left < right
	})
	e.writeInt(len(values))
	for _, value := range values {
		e.writeString(expr.UnionVariantTag(value.branch))
		e.writeAttribute(value.branch.Attribute, closeObjects, value.input)
	}
}

func (e *canonicalSchemaEncoder) writeValidation(att *expr.AttributeExpr, input schemaFingerprintInput) {
	validation := att.Validation
	var enums [][]any
	if e.checked != nil {
		enums = e.checked.annotations(input)
	} else if validation != nil {
		for _, clause := range validation.Enums() {
			enums = append(enums, projectOpenAPIValues(att, clause))
		}
	}
	if validation == nil {
		if len(enums) == 0 {
			e.writeString("no-validation")
			return
		}
		validation = &expr.ValidationExpr{}
	}
	e.writeString("validation")
	e.writeValidationEnums(enums)
	e.writeValidationStrings(validation)
	e.writeOptionalBound(validation.ExclusiveMinimum)
	e.writeOptionalBound(validation.Minimum)
	e.writeOptionalBound(validation.Maximum)
	e.writeOptionalBound(validation.ExclusiveMaximum)
	e.writeOptionalInt(validation.MinLength)
	e.writeOptionalInt(validation.MaxLength)
	e.writeValidationRequired(att, validation.Required)
}

func (e *canonicalSchemaEncoder) writeValidationEnums(enums [][]any) {
	if len(enums) == 0 {
		e.writeValueSet(nil)
	} else {
		e.writeValueSet(enums[0])
		if len(enums[0]) == 0 {
			e.writeString("empty-enum-domain")
		}
	}
	if len(enums) > 1 {
		e.writeString("enum-clauses")
		e.writeInt(len(enums) - 1)
		for _, values := range enums[1:] {
			e.writeValueSet(values)
		}
	}
}

func (e *canonicalSchemaEncoder) writeValidationStrings(validation *expr.ValidationExpr) {
	e.writeString(string(validation.Format))
	e.writeString(validation.Pattern)
	formats := validation.Formats()
	if validation.Format != "" && len(formats) > 0 {
		formats = formats[1:]
	}
	if len(formats) > 0 {
		e.writeString("format-clauses")
		e.writeInt(len(formats))
		for _, format := range formats {
			e.writeString(string(format))
		}
	}
	patterns := validation.Patterns()
	if validation.Pattern != "" && len(patterns) > 0 {
		patterns = patterns[1:]
	}
	if len(patterns) > 0 {
		e.writeString("pattern-clauses")
		e.writeInt(len(patterns))
		for _, pattern := range patterns {
			e.writeString(pattern)
		}
	}
}

func (e *canonicalSchemaEncoder) writeValidationRequired(att *expr.AttributeExpr, names []string) {
	required := make([]string, 0, len(names))
	for _, name := range names {
		if child := att.Find(name); child != nil {
			if !openapi.MustGenerate(child.Meta) {
				continue
			}
			name = expr.JSONFieldName(expr.ElementName(name), child)
			if name == "-" {
				continue
			}
		}
		required = append(required, name)
	}
	sort.Strings(required)
	required = compactSortedStrings(required)
	e.writeInt(len(required))
	for _, name := range required {
		e.writeString(name)
	}
}

func (e *canonicalSchemaEncoder) writeValueSet(values []any) {
	encoded, _ := canonicalSetValues(values)
	e.writeInt(len(encoded))
	for _, value := range encoded {
		e.writeString(value)
	}
}

func canonicalSetValues(values []any) ([]string, []any) {
	byEncoding := make(map[string]any, len(values))
	for _, value := range values {
		byEncoding[canonicalJSON(value)] = value
	}
	encoded := make([]string, 0, len(byEncoding))
	for value := range byEncoding {
		encoded = append(encoded, value)
	}
	sort.Strings(encoded)
	canonical := make([]any, 0, len(encoded))
	for _, value := range encoded {
		canonical = append(canonical, byEncoding[value])
	}
	return encoded, canonical
}

func (e *canonicalSchemaEncoder) writeOptionalBound(value *expr.NumericBound) {
	if value == nil {
		e.writeBool(false)
		return
	}
	e.writeBool(true)
	e.writeString(value.Key())
}

func (e *canonicalSchemaEncoder) writeOptionalInt(value *int) {
	if value == nil {
		e.writeBool(false)
		return
	}
	e.writeBool(true)
	e.writeInt(*value)
}

func (e *canonicalSchemaEncoder) writeString(value string) {
	if err := binary.Write(&e.Buffer, binary.BigEndian, uint64(len(value))); err != nil {
		panic(err)
	}
	if _, err := e.Buffer.WriteString(value); err != nil {
		panic(err)
	}
}

func (e *canonicalSchemaEncoder) writeBool(value bool) {
	if value {
		e.writeString("true")
		return
	}
	e.writeString("false")
}

func (e *canonicalSchemaEncoder) writeInt(value int) {
	e.writeString(strconv.Itoa(value))
}

func canonicalJSON(value any) string {
	encoded, err := json.Marshal(toStringMap(value), json.Deterministic(true))
	if err != nil {
		panic(fmt.Sprintf("openapi: cannot fingerprint schema value: %v", err))
	}
	return string(encoded)
}

func compactSortedStrings(values []string) []string {
	if len(values) < 2 {
		return values
	}
	writeIndex := 1
	for _, value := range values[1:] {
		if value == values[writeIndex-1] {
			continue
		}
		values[writeIndex] = value
		writeIndex++
	}
	return values[:writeIndex]
}

func boolPtr(v bool) *bool {
	return &v
}
