package codegen

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/CaliLuke/loom/codegen/unionnames"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/representation"
)

type (
	// DefaultValueOptions selects materialization timing without changing the
	// context's declared pointer, presence, or collection representation.
	DefaultValueOptions struct {
		// MaterializeMembers fills absent members from their selected defaults
		// after the containing source value has been admitted. It never creates
		// an absent outer value or changes required-input validation.
		MaterializeMembers bool
	}

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
		values             *expr.ValueContext
		active             map[expr.ValueIdentity]bool
		errorAware         bool
		materializeMembers bool
	}
)

// DefaultValue resolves authored input under the attribute's effective constraints
// and renders it in this context's physical Go layout. It does not decide when a
// default applies. Unrepresentable members return an error rather than being
// dropped. Opaque values remain owned by an explicitly selected custom or raw
// JSON representation; they are never reflected into guessed object fields.
func (a *AttributeContext) DefaultValue(att *expr.AttributeExpr, value any, options ...DefaultValueOptions) (code DefaultValueCode, err error) {
	defer func() {
		if err != nil {
			err = &DefaultValueError{Attribute: att, Err: err}
		}
	}()
	if len(options) > 1 {
		return DefaultValueCode{}, fmt.Errorf("DefaultValue accepts at most one options value")
	}
	context, occurrence, resolved, err := resolveDefaultValue(att, value)
	if err != nil {
		return DefaultValueCode{}, err
	}
	renderer := &defaultRenderer{values: context, active: make(map[expr.ValueIdentity]bool)}
	if len(options) == 1 {
		renderer.materializeMembers = options[0].MaterializeMembers
	}
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
		if expr.IsUnion(child.Type) && facts.ServiceLayout(selected, context.Pointer, context.UseDefault).Pointer {
			return defaultUnionPointer(context.Scope.Ref(child, context.Pkg(child)), literal)
		}
		if expr.IsPrimitive(child.Type) && expr.IsPrimitive(selected.Type) && facts.ServiceLayout(selected, context.Pointer, context.UseDefault).Pointer {
			ref := context.Scope.Ref(child, context.Pkg(child))
			return "func() *" + ref + " {\n v := " + ref + "(" + literal + ")\n return &v\n}()"
		}
	}
	return literal
}

func (r *defaultRenderer) render(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	context = context.ForAttribute(att)
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
		return opaqueDefaultLiteral(att, source, path)
	}
	if expr.IsNullable(att) {
		return r.nullable(att, occurrence, value, context, path)
	}
	switch actual := unalias(att.Type).(type) {
	case *expr.Object:
		return r.object(att, occurrence, value, context, path)
	case *expr.Array:
		return r.array(att, actual, occurrence, value, context, path)
	case *expr.Map:
		return r.mapping(att, actual, occurrence, value, context, path)
	case *expr.Union:
		return r.union(att, actual, occurrence, value, context, path)
	default:
		scalar, ok := value.Scalar()
		if !ok {
			return "", fmt.Errorf("%s: default has no native scalar representation", path)
		}
		return formatAttributeGoLiteral(att, scalar), nil
	}
}

func defaultUnionPointer(ref, literal string) string {
	return "func() " + ref + " { v := " + literal + "; return &v }()"
}

// union lowers the branch already selected by the value owner. Constructor
// identity belongs to the same allocated declaration family as the emitter.
func (r *defaultRenderer) union(att *expr.AttributeExpr, actual *expr.Union, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	_, selected, payload, ok := value.Union()
	if !ok {
		return "", fmt.Errorf("%s: default has no selected union branch", path)
	}
	context = transformPhysicalContext(att, context)
	for _, branch := range occurrence.Branches() {
		if branch.ID != selected {
			continue
		}
		for _, declaration := range actual.Values {
			if declaration.Name != branch.Name {
				continue
			}
			attribute := declaration.Attribute
			branchContext, pointer := context.UnionBranchContext(attribute)
			literal, err := r.render(attribute, branch.Occurrence, payload, branchContext, path+"."+branch.Name)
			if err != nil {
				return "", err
			}
			if pointer && expr.IsUnion(attribute.Type) && !expr.IsNullable(attribute) {
				literal = defaultUnionPointer(branchContext.Scope.Ref(attribute, branchContext.Pkg(attribute)), literal)
			}
			primary := context.Scope.Scope().GoValueTypeName(att)
			constructor := unionnames.PackageDeclarations(actual, primary).Branches[branch.Name].Constructor
			if pkg := context.Pkg(att); pkg != "" {
				constructor = pkg + "." + constructor
			}
			return constructor + "(" + literal + ")", nil
		}
	}
	return "", fmt.Errorf("%s: selected union branch has no native declaration", path)
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
	context = transformPhysicalContext(att, context)
	ref := context.CollectionElementRef(att, false)
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
		if context.CollectionElementPresence && !expr.ArrayElementsAllowNull(actual) {
			literal = "loom.OptionalValue[" + context.CollectionElementRef(actual.ElemType, false) + "](" + literal + ")"
		}
		items[index] = literal
	}
	return ref + "{" + strings.Join(items, ", ") + "}", nil
}

func (r *defaultRenderer) mapping(att *expr.AttributeExpr, actual *expr.Map, occurrence expr.ValueOccurrence, value expr.ResolvedValue, context *AttributeContext, path string) (string, error) {
	context = transformPhysicalContext(att, context)
	ref := context.CollectionElementRef(att, false)
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
		if context.CollectionElementPresence && !expr.MapValuesAllowNull(actual) {
			child = "loom.OptionalValue[" + context.CollectionElementRef(actual.ElemType, false) + "](" + child + ")"
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
			literal, err := r.additionalMember(att, occurrence, field, context, path)
			if err != nil {
				return "", err
			}
			extras = append(extras, literal)
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
		fields = append(fields, expr.AdditionalAttributesFieldName+": "+context.CollectionElementRef(carrier, false)+"{"+strings.Join(extras, ", ")+"}")
	}
	ref := context.Scope.Name(att, context.Pkg(att), context)
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
		if raw == nil || !r.materializeMembers && (!context.UseDefault || context.Pointer) {
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

// additionalMember renders a literal-key entry of the undeclared-member map.
func (r *defaultRenderer) additionalMember(att *expr.AttributeExpr, occurrence expr.ValueOccurrence, field expr.ResolvedField, context *AttributeContext, path string) (string, error) {
	declaration := att.TypedObjectDeclaration()
	additional, typed := occurrence.Additional()
	if declaration == nil || !typed {
		return "", fmt.Errorf("%s.%s: authored default member has no native Go field", path, field.Name)
	}
	value, err := r.render(declaration.AdditionalAttributes, additional, field.Value, context, path+"."+field.Name)
	if err != nil {
		return "", err
	}
	return strconv.Quote(field.Name) + ": " + value, nil
}

// resolveDefaultValue is the strict effective-value owner shared by literal
// rendering and the bounded native-scalar default equivalence proof.
func resolveDefaultValue(att *expr.AttributeExpr, value any) (*expr.ValueContext, expr.ValueOccurrence, expr.ResolvedValue, error) {
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(att)
	if err != nil {
		return nil, expr.ValueOccurrence{}, expr.ResolvedValue{}, fmt.Errorf("default representation: %w", err)
	}
	result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: value}), expr.ValueRoleDefault)
	resolved, ok := result.RepresentationValue()
	if !ok {
		return nil, expr.ValueOccurrence{}, expr.ResolvedValue{}, fmt.Errorf("default representation: cannot resolve value: %v", result.Diagnostics())
	}
	return context, occurrence, resolved, nil
}
