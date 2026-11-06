package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

type (
	// DefaultValueCode binds a default expression to its runtime error channel.
	// Error-aware expressions record the first materialization failure through
	// transformErr, a nonnil *error owned by the enclosing conversion boundary.
	DefaultValueCode struct {
		// Expression constructs the value in the selected physical Go layout.
		Expression string
		// ErrorAware reports whether Expression requires transformErr.
		ErrorAware bool
	}

	defaultRenderer struct {
		values     *expr.ValueContext
		active     map[expr.ValueIdentity]bool
		errorAware bool
	}
)

// DefaultValue resolves authored input under the attribute's effective constraints
// and renders it in this context's physical Go layout. It does not decide when a
// default applies. Unrepresentable members return an error rather than being
// dropped. Opaque values remain owned by an explicitly selected custom or raw
// JSON representation; they are never reflected into guessed object fields.
func (a *AttributeContext) DefaultValue(att *expr.AttributeExpr, value any) (DefaultValueCode, error) {
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(att)
	if err != nil {
		return DefaultValueCode{}, fmt.Errorf("default representation: %w", err)
	}
	result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: value}), expr.ValueRoleDefault)
	resolved, ok := result.RepresentationValue()
	if !ok {
		return DefaultValueCode{}, fmt.Errorf("default representation: cannot resolve value: %v", result.Diagnostics())
	}
	renderer := &defaultRenderer{values: context, active: make(map[expr.ValueIdentity]bool)}
	expression, err := renderer.render(att, occurrence, resolved, a, "default")
	return DefaultValueCode{Expression: expression, ErrorAware: renderer.errorAware}, err
}

func defaultFieldLiteral(child, selected *expr.AttributeExpr, facts expr.ObjectFieldFacts, context *AttributeContext, presence PresenceKind, literal string) string {
	switch presence {
	case OptionalPresence:
		ref := presenceValueTypeRef(child, context)
		if expr.IsObject(child.Type) {
			literal = "*(" + literal + ")"
		}
		return "loom.OptionalValue[" + ref + "](" + literal + ")"
	case NativePresence:
		if expr.IsPrimitive(child.Type) && expr.IsPrimitive(selected.Type) && facts.ServiceLayout(selected, context.Pointer, context.UseDefault).Pointer {
			ref := context.Scope.Ref(child, context.Pkg(child))
			return "func() *" + ref + " {\n v := " + ref + "(" + literal + ")\n return &v\n}()"
		}
	}
	return literal
}

func (r *defaultRenderer) render(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	if meta, _ := GetMetaType(att); meta != "" || isRawJSONValue(att) && !expr.IsNullable(att) {
		source, ok := value.SourceValue()
		if !ok {
			return "", fmt.Errorf("%s: missing default source", path)
		}
		if isRawJSONValue(att) {
			literal, err := formatRawJSONLiteral(source)
			if err != nil {
				return "", fmt.Errorf("%s: %w", path, err)
			}
			return literal, nil
		}
		if text, ok := source.(string); ok && representation.CustomString(att) {
			r.errorAware = true
			return renderCustomStringDefault(att, text, context, path), nil
		}
		return formatAttributeGoLiteral(att, source), nil
	}
	if expr.AllowsNull(att) {
		return r.nullable(att, occurrence, value, context, path)
	}
	switch actual := unalias(att.Type).(type) {
	case *expr.Object:
		return r.object(att, occurrence, value, context, path)
	case *expr.Array:
		return r.array(att, actual, occurrence, value, context, path)
	case *expr.Map:
		return r.mapping(att, actual, occurrence, value, context, path)
	default:
		scalar, ok := value.Scalar()
		if !ok {
			return "", fmt.Errorf("%s: default has no native scalar representation", path)
		}
		return formatAttributeGoLiteral(att, scalar), nil
	}
}

func (r *defaultRenderer) nullable(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	concrete := concretePresenceAttribute(att)
	ctx := presenceValueContext(context, concrete)
	ref := presenceValueTypeRef(att, ctx)
	if value.Presence() == expr.ValueNull {
		return "loom.NullValue[" + ref + "]()", nil
	}
	literal, err := r.render(concrete, occurrence, value, ctx, path)
	if err != nil {
		return "", err
	}
	if expr.IsObject(concrete.Type) {
		literal = "*(" + literal + ")"
	}
	return "loom.NullableValue[" + ref + "](" + literal + ")", nil
}

func (r *defaultRenderer) array(att *expr.AttributeExpr, actual *expr.Array, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	ref := collectionTypeRef(att, context, false)
	if value.Presence() == expr.ValueNil {
		return ref + "(nil)", nil
	}
	elements := value.Elements()
	items := make([]string, len(elements))
	for index, child := range elements {
		literal, err := r.render(actual.ElemType, occurrence.Element(), child, context, fmt.Sprintf("%s[%d]", path, index))
		if err != nil {
			return "", err
		}
		items[index] = literal
	}
	return ref + "{" + strings.Join(items, ", ") + "}", nil
}

func (r *defaultRenderer) mapping(att *expr.AttributeExpr, actual *expr.Map, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	ref := collectionTypeRef(att, context, false)
	if value.Presence() == expr.ValueNil {
		return ref + "(nil)", nil
	}
	items := make([]string, 0, len(value.Entries()))
	for _, entry := range value.Entries() {
		var key string
		if isRawJSONValue(actual.KeyType) {
			source, present := entry.Key.SourceValue()
			if !present {
				return "", fmt.Errorf("%s[key]: missing default key", path)
			}
			key = formatAttributeGoLiteral(actual.KeyType, source)
		} else {
			var err error
			key, err = r.render(actual.KeyType, occurrence.Key(), entry.Key, context, path+"[key]")
			if err != nil {
				return "", err
			}
		}
		child, err := r.render(actual.ElemType, occurrence.Element(), entry.Value, context, path+"["+key+"]")
		if err != nil {
			return "", err
		}
		items = append(items, key+": "+child)
	}
	sort.Strings(items)
	return ref + "{" + strings.Join(items, ", ") + "}", nil
}

func (r *defaultRenderer) object(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	context = presenceValueContext(context, att)
	members := make(map[expr.ValueIdentity]expr.ValueMember)
	for _, member := range occurrence.Members() {
		members[member.ID] = member
	}
	mapped := expr.NewObjectFields(att)
	fields := make([]string, 0, len(value.Fields()))
	var extras []string
	for _, field := range value.Fields() {
		member, known := members[field.Member]
		if !known {
			literal, extra, err := r.additionalMember(att, occurrence, field, mapped, context, path)
			if err != nil {
				return "", err
			}
			if extra {
				extras = append(extras, literal)
			} else {
				fields = append(fields, literal)
			}
			continue
		}
		literal, present, err := r.declaredMember(att, member, field.Value, mapped, context, path)
		if err != nil {
			return "", err
		}
		if present {
			fields = append(fields, literal)
		}
	}
	if len(extras) > 0 {
		sort.Strings(extras)
		carrier := att.AdditionalMembersCarrier()
		fields = append(fields, expr.AdditionalAttributesFieldName+": "+collectionTypeRef(carrier, context, false)+"{"+strings.Join(extras, ", ")+"}")
	}
	ref := context.Scope.Name(att, context.Pkg(att), context.Pointer, context.UseDefault)
	return "&" + ref + "{" + strings.Join(fields, ", ") + "}", nil
}

// declaredMember renders the default of a declared member. An absent member
// renders its own default when the context uses defaults; present reports
// whether the member renders at all.
func (r *defaultRenderer) declaredMember(att *expr.AttributeExpr, member expr.ValueMember, childValue expr.ResolvedValue, mapped *expr.ObjectFields, context *AttributeContext, path string) (literal string, present bool, err error) {
	child := expr.AsObject(att.Type).Attribute(member.Name)
	name := expr.AttributeName(member.Name)
	facts := mapped.FieldFacts(name)
	winner := (*expr.AsObject(att.Type))[facts.Index]
	expanded := false
	if childValue.Presence() == expr.ValueAbsent {
		// Query the mapped winner without copying its default or descendants.
		raw := (&expr.AttributeExpr{Type: &expr.Object{winner}}).GetDefault(winner.Name)
		if raw == nil || !context.UseDefault || context.Pointer {
			return "", false, nil
		}
		if r.active[member.Occurrence.ID()] {
			return "", false, fmt.Errorf("%s.%s: nested defaults require an infinite object value", path, name)
		}
		r.active[member.Occurrence.ID()] = true
		expanded = true
		result := r.values.Resolve(member.Occurrence, r.values.SupplyValue(expr.ValueInput{Raw: raw}), expr.ValueRoleDefault)
		var ok bool
		childValue, ok = result.RepresentationValue()
		if !ok {
			return "", false, fmt.Errorf("%s.%s: invalid nested default: %v", path, name, result.Diagnostics())
		}
	}
	literal, err = r.render(child, member.Occurrence, childValue, context, path+"."+name)
	if expanded {
		delete(r.active, member.Occurrence.ID())
	}
	if err != nil {
		return "", false, err
	}
	literal = defaultFieldLiteral(child, winner.Attribute, facts, context, context.FieldPresence(mapped, name, child), literal)
	return context.Scope.Field(child, name, true) + ": " + literal, true, nil
}

// additionalMember renders a default member that a typed object does not
// declare: a field of its own when the object requires the member, or else an
// entry of the map that carries undeclared members. extra reports the latter.
func (r *defaultRenderer) additionalMember(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, field expr.ResolvedField, mapped *expr.ObjectFields, context *AttributeContext, path string) (literal string, extra bool, err error) {
	declaration := att.TypedObjectDeclaration()
	additional, typed := occurrence.Additional()
	if declaration == nil || !typed {
		return "", false, fmt.Errorf("%s.%s: authored default member has no native Go field", path, field.Name)
	}
	value, err := r.render(declaration.AdditionalAttributes, additional, field.Value, context, path+"."+field.Name)
	if err != nil {
		return "", false, err
	}
	facts := mapped.FieldFacts(field.Name)
	if !facts.Synthesized {
		return strconv.Quote(field.Name) + ": " + value, true, nil
	}
	member := declaration.AdditionalAttributes
	value = defaultFieldLiteral(member, member, facts, context, context.FieldPresence(mapped, field.Name, member), value)
	return context.Scope.Field(member, field.Name, true) + ": " + value, false, nil
}
