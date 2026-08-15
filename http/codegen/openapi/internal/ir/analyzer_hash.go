package ir

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
)

type canonicalSchemaEncoder struct {
	bytes.Buffer
	active map[expr.DataType]int
	depth  int
}

// schemaFingerprintVersion identifies Loom's canonical OpenAPI schema encoding.
// The encoding uses length-prefixed fields, sorted object keys, set semantics
// for enum and required values, and last-write-wins object member semantics.
const schemaFingerprintVersion = "loom-openapi-schema-v1"

func fingerprintAttribute(att *expr.AttributeExpr, closeObjects bool) string {
	encoding := canonicalAttributeEncoding(att, closeObjects)
	sum := sha256.Sum256(encoding)
	return hex.EncodeToString(sum[:])
}

func fingerprintString(value string) string {
	encoder := newCanonicalSchemaEncoder()
	encoder.writeString(schemaFingerprintVersion)
	encoder.writeString("text")
	encoder.writeString(value)
	sum := sha256.Sum256(encoder.Bytes())
	return hex.EncodeToString(sum[:])
}

func canonicalAttributeEncoding(att *expr.AttributeExpr, closeObjects bool) []byte {
	encoder := newCanonicalSchemaEncoder()
	encoder.writeString(schemaFingerprintVersion)
	encoder.writeAttribute(att, closeObjects)
	return encoder.Bytes()
}

func newCanonicalSchemaEncoder() *canonicalSchemaEncoder {
	return &canonicalSchemaEncoder{active: make(map[expr.DataType]int)}
}

func (e *canonicalSchemaEncoder) writeAttribute(att *expr.AttributeExpr, closeObjects bool) {
	if att == nil {
		e.writeString("nil-attribute")
		return
	}
	if att.Type != nil && att.Type.Kind() == expr.UserTypeKind {
		e.writeType(att, closeObjects)
		return
	}
	e.writeString("attribute")
	e.writeType(att, closeObjects)
	if att.Type != nil && (att.Type.Kind() == expr.UserTypeKind || att.Type.Kind() == expr.ResultTypeKind) {
		return
	}
	e.writeValidation(att)
}

func (e *canonicalSchemaEncoder) writeType(att *expr.AttributeExpr, closeObjects bool) {
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

	e.writeTypeShape(att, t, closeObjects)
}

func (e *canonicalSchemaEncoder) writeTypeShape(att *expr.AttributeExpr, t expr.DataType, closeObjects bool) {
	if e.writePrimitiveType(t.Kind()) {
		return
	}
	switch t.Kind() {
	case expr.BytesKind:
		e.writeString("bytes")
		e.writeInt(len(att.Bases))
		for _, base := range att.Bases {
			e.writeAttribute(&expr.AttributeExpr{Type: base}, closeObjects)
		}
	case expr.ArrayKind:
		e.writeString("array")
		e.writeAttribute(expr.AsArray(t).ElemType, closeObjects)
	case expr.MapKind:
		e.writeString("map")
		e.writeAttribute(expr.AsMap(t).ElemType, closeObjects)
	case expr.ObjectKind:
		e.writeObject(att, expr.AsObject(t), closeObjects)
	case expr.UserTypeKind:
		e.writeAttribute(t.(expr.UserType).Attribute(), closeObjects)
	case expr.ResultTypeKind:
		e.writeString("result-type")
		resultType := t.(*expr.ResultTypeExpr)
		view, ok := att.Meta.Last(expr.ViewMetaKey)
		if !ok {
			view, _ = resultType.Meta.Last(expr.ViewMetaKey)
		}
		e.writeString(view)
		e.writeAttribute(resultType.Attribute(), closeObjects)
	case expr.UnionKind:
		e.writeUnion(t.(*expr.Union), closeObjects)
	default:
		panic(fmt.Sprintf("openapi: cannot fingerprint type %T", t))
	}
}

func (e *canonicalSchemaEncoder) writePrimitiveType(kind expr.Kind) bool {
	switch kind {
	case expr.BooleanKind:
		e.writePrimitive("boolean", "")
	case expr.IntKind, expr.UIntKind, expr.Int64Kind, expr.UInt64Kind:
		e.writePrimitive("integer", "int64")
	case expr.Int32Kind, expr.UInt32Kind:
		e.writePrimitive("integer", "int32")
	case expr.Float32Kind:
		e.writePrimitive("number", "float")
	case expr.Float64Kind:
		e.writePrimitive("number", "double")
	case expr.StringKind:
		e.writePrimitive("string", "")
	case expr.AnyKind:
		e.writePrimitive("", "")
	default:
		return false
	}
	return true
}

func (e *canonicalSchemaEncoder) writePrimitive(schemaType, format string) {
	e.writeString("primitive")
	e.writeString(schemaType)
	e.writeString(format)
}

func (e *canonicalSchemaEncoder) writeObject(att *expr.AttributeExpr, object *expr.Object, closeObjects bool) {
	e.writeString("object")
	members := make(map[string]*expr.AttributeExpr, len(*object))
	for _, member := range *object {
		if openapi.MustGenerate(member.Attribute.Meta) {
			members[member.Name] = member.Attribute
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
		e.writeAttribute(members[name], closeObjects)
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

func (e *canonicalSchemaEncoder) writeUnion(union *expr.Union, closeObjects bool) {
	e.writeString("union")
	e.writeString(union.GetTypeKey())
	e.writeString(union.GetValueKey())
	values := append([]*expr.NamedAttributeExpr(nil), union.Values...)
	sort.SliceStable(values, func(i, j int) bool {
		if values[i].Name != values[j].Name {
			return values[i].Name < values[j].Name
		}
		left := fingerprintAttribute(values[i].Attribute, closeObjects)
		right := fingerprintAttribute(values[j].Attribute, closeObjects)
		return left < right
	})
	e.writeInt(len(values))
	for _, value := range values {
		e.writeString(expr.UnionVariantTag(value))
		e.writeAttribute(value.Attribute, closeObjects)
	}
}

func (e *canonicalSchemaEncoder) writeValidation(att *expr.AttributeExpr) {
	validation := att.Validation
	if validation == nil {
		e.writeString("no-validation")
		return
	}
	e.writeString("validation")
	e.writeValueSet(validation.Values)
	e.writeString(string(validation.Format))
	e.writeString(validation.Pattern)
	e.writeOptionalFloat(validation.ExclusiveMinimum)
	e.writeOptionalFloat(validation.Minimum)
	e.writeOptionalFloat(validation.Maximum)
	e.writeOptionalFloat(validation.ExclusiveMaximum)
	e.writeOptionalInt(validation.MinLength)
	e.writeOptionalInt(validation.MaxLength)

	required := make([]string, 0, len(validation.Required))
	for _, name := range validation.Required {
		if child := att.Find(name); child != nil && !openapi.MustGenerate(child.Meta) {
			continue
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

func (e *canonicalSchemaEncoder) writeOptionalFloat(value *float64) {
	if value == nil {
		e.writeBool(false)
		return
	}
	e.writeBool(true)
	e.writeString(strconv.FormatFloat(*value, 'g', -1, 64))
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
	encoded, err := json.Marshal(toStringMap(value))
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
