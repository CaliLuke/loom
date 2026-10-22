package expr

import "iter"

type (
	// ObjectFieldFacts contains scalar facts for one logical object field.
	// It retains no attribute, raw default value or mutable graph.
	ObjectFieldFacts struct {
		// ParentPresent distinguishes an absent parent from an unknown field.
		ParentPresent bool
		// Required records normalized requiredness, including dangling names.
		Required bool
		// HasDefault records selected default presence without materializing it.
		HasDefault bool
		// NativePrimitive records eligibility for native primitive pointer storage.
		NativePrimitive bool
		// ElementName is the mapped transport name, empty for an unknown field.
		ElementName string
		// OriginalKey is the winning field's key in the original object.
		OriginalKey string
		// Index is the winning field's original position, or -1 when absent.
		Index int
	}

	// FieldFactsProvider supplies field facts from an immutable snapshot or the
	// current state of a mutable mapped object. Builtin nil providers are absent.
	FieldFactsProvider interface {
		// FieldFacts returns scalar facts for the requested logical field name.
		FieldFacts(name string) ObjectFieldFacts
	}

	// ObjectFields owns scalar facts and logical field order captured from an
	// object. Concurrent reads are safe; construction requires a stable input.
	ObjectFields struct {
		fields map[string]ObjectFieldFacts
		order  []string
	}
)

// NewObjectFields captures the mapped interpretation of an object's immediate
// fields without copying descendants. Nil is an empty present object; a nonobject
// panics. Exact keys first collapse at their first position with their last
// attribute, then logical names collapse by the same rule;
// a later unqualified key does not erase an earlier explicit element mapping.
// Later input mutations do not affect the snapshot.
func NewObjectFields(attribute *AttributeExpr) *ObjectFields {
	result := &ObjectFields{fields: make(map[string]ObjectFieldFacts)}
	if attribute == nil {
		return result
	}
	object := AsObject(attribute.Type)
	if object == nil {
		panic("cannot capture object field facts from a non object attribute")
	}
	// Match mapped construction: DupAtt collapses exact keys before Remap
	// collapses logical names. Keep only scalar positions in this temporary index.
	positions := make(map[string][2]int, len(*object))
	for index, field := range *object {
		position, exists := positions[field.Name]
		if !exists {
			position[0] = index
		}
		position[1] = index
		positions[field.Name] = position
	}
	for originalIndex, original := range *object {
		position := positions[original.Name]
		if originalIndex != position[0] {
			continue
		}
		index := position[1]
		field := (*object)[index]
		name := AttributeName(field.Name)
		prior, exists := result.fields[name]
		element := name
		if !exists {
			result.order = append(result.order, name)
		} else {
			element = prior.ElementName
		}
		if name != field.Name {
			element = ElementName(field.Name)
		}
		result.fields[name] = ObjectFieldFacts{
			ParentPresent:   true,
			HasDefault:      field.Attribute.effectiveDefault() != nil,
			NativePrimitive: primitivePointerEligible(field.Attribute),
			ElementName:     element, OriginalKey: field.Name, Index: index,
		}
	}
	for _, required := range attribute.AllRequired() {
		name := AttributeName(required)
		facts := result.FieldFacts(name)
		facts.Required = true
		result.fields[name] = facts
	}
	return result
}

// PrimitivePointer reports whether an optional primitive uses pointer storage
// under the selected default policy. Transport-wide pointer flags are separate.
func (f ObjectFieldFacts) PrimitivePointer(useDefault bool) bool {
	return f.NativePrimitive && !f.Required && (!f.HasDefault || !useDefault)
}

// FieldFacts returns captured facts by logical name. Unknown names retain parent
// presence and dangling requiredness, but have no physical index or element name.
func (f *ObjectFields) FieldFacts(name string) ObjectFieldFacts {
	if f == nil {
		return ObjectFieldFacts{Index: -1}
	}
	if facts, found := f.fields[name]; found {
		return facts
	}
	return ObjectFieldFacts{ParentPresent: true, Index: -1}
}

// Fields yields scalar selections in first logical-name appearance order.
// Exact keys collapse before logical names; each selection identifies the
// winning original attribute under those two ordered collapses.
func (f *ObjectFields) Fields() iter.Seq2[string, ObjectFieldFacts] {
	return func(yield func(string, ObjectFieldFacts) bool) {
		if f == nil {
			return
		}
		for _, name := range f.order {
			if !yield(name, f.fields[name]) {
				return
			}
		}
	}
}

// FieldFacts queries the current mutable mapping without caching it. It observes
// explicit Map overrides and later mutations; nil means an absent parent.
func (ma *MappedAttributeExpr) FieldFacts(name string) ObjectFieldFacts {
	if ma == nil {
		return ObjectFieldFacts{Index: -1}
	}
	facts := ObjectFieldFacts{ParentPresent: true, Required: ma.IsRequired(name), Index: -1}
	for index, field := range *AsObject(ma.Type) {
		if field.Name != name {
			continue
		}
		facts.HasDefault = field.Attribute.effectiveDefault() != nil
		facts.NativePrimitive = primitivePointerEligible(field.Attribute)
		facts.OriginalKey, facts.Index = name, index
		facts.ElementName = name
		if element, mapped := ma.nameMap[name]; mapped {
			facts.ElementName = element
		}
		break
	}
	return facts
}

func primitivePointerEligible(attribute *AttributeExpr) bool {
	if attribute == nil || !IsPrimitive(attribute.Type) {
		return false
	}
	kind := unalias(attribute.Type).Kind()
	return kind != BytesKind && kind != AnyKind
}
