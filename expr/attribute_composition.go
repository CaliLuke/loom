package expr

import (
	"errors"
	"fmt"
	"reflect"
	"slices"

	"github.com/CaliLuke/loom/eval"
)

type designComposer struct {
	seen         map[*AttributeExpr]bool
	active       map[*AttributeExpr]bool
	composed     []*AttributeExpr
	keep         map[copyTypeKey]struct{}
	pending      []compositionCopy
	owner        UserType
	declarations map[*AttributeExpr]UserType
	jsonTags     []compositionJSONTags
}

// additionalPolicyOverlap classifies the conjunction of typed
// additional-member policies.
type additionalPolicyOverlap int

const (
	policiesRefine additionalPolicyOverlap = iota
	policiesDisjoint
	policiesIncomparable
)

// SetDSLSource retains the authored callback for diagnostics independently of
// evaluator wrappers that execute a declaration at most once.
func (a *AttributeExpr) SetDSLSource(fn func()) {
	a.declarationDSL = fn
}

// DSLSource returns the authored callback used for declaration diagnostics.
func (a *AttributeExpr) DSLSource() func() {
	if a.declarationDSL != nil {
		return a.declarationDSL
	}
	return a.DSLFunc
}

// CompositionBases returns the named object contracts retained by Extend after
// composition. The returned slice is detached; its declarations are shared and
// must not be mutated. Before composition it returns the authored bases.
func (a *AttributeExpr) CompositionBases() []DataType {
	if a == nil {
		return nil
	}
	if len(a.compositionBases) > 0 {
		return slices.Clone(a.compositionBases)
	}
	return slices.Clone(a.Bases)
}

// composeDesign establishes object shape before any transport prepares a
// mapping or validation inspects a field set. Finalize still owns defaults,
// views and transport lowering that depend on successful validation.
func (r *RootExpr) composeDesign() {
	roots := r.unionBranchRoots(make(map[UserType]string))
	graph := newCompositionGraph(roots)
	graph.validateInheritance()
	if eval.Context.Errors != nil {
		return
	}
	c := &designComposer{seen: make(map[*AttributeExpr]bool), active: make(map[*AttributeExpr]bool), keep: graph.keep, declarations: graph.declarations}
	for _, root := range roots {
		c.walk(root.attribute, root.path)
	}
	c.detachNamedCopies()
	if eval.Context.Errors != nil {
		return
	}
	for _, attribute := range c.composed {
		constraints, err := EffectiveConstraintsFor(attribute)
		if err != nil {
			compositionError(attribute, "%s: %v", attribute.compositionName, err)
			continue
		}
		validation := constraints.Validation().Lowered()
		if values, present := constraints.EnumCandidates(); present && len(values) == 0 {
			compositionError(attribute, "%s: contradictory Extend enum constraints", attribute.compositionName)
			continue
		}
		if len(validation.Required) == 0 {
			validation.Required = nil
		}
		if reflect.DeepEqual(*validation, ValidationExpr{}) {
			validation = nil
		}
		attribute.Validation = validation
		applyEffectiveDefault(attribute, constraints)
	}
	// JSON identity depends on finalized inherited defaults as well as metadata.
	// Check the original authored declarations against the canonical service
	// carrier only after every effective constraint has been selected.
	for _, check := range c.jsonTags {
		validateCompositionJSONTags(check.attribute, check.bases, check.effective, check.path, check.fieldName, check.optional)
	}
}

func (c *designComposer) walk(attribute *AttributeExpr, path string) {
	if attribute == nil {
		return
	}
	if c.active[attribute] {
		return
	}
	if c.seen[attribute] {
		return
	}
	c.active[attribute] = true
	defer delete(c.active, attribute)
	previousOwner := c.owner
	if owner := c.declarations[attribute]; owner != nil {
		c.owner = owner
	}
	defer func() {
		c.owner = previousOwner
	}()
	if err := attribute.ResolveTypeReference(); err != nil {
		compositionError(attribute, "%s: %v", path, err)
		return
	}
	if !c.publishTemplate(attribute, path) {
		return
	}
	attribute.compositionName = path
	attribute.normalizeLegacyPresence(make(map[*AttributeExpr]struct{}))
	if named, ok := attribute.Type.(UserType); ok {
		c.walk(named.Attribute(), named.Name())
	}
	var bases []*AttributeExpr
	for _, base := range attribute.Bases {
		inherited := &AttributeExpr{Type: base}
		if named, ok := base.(UserType); ok {
			inherited = named.Attribute()
		}
		c.walk(inherited, base.Name())
		bases = append(bases, inherited)
	}
	if len(bases) > 0 {
		c.object(attribute, bases, path)
		attribute.compositionBases = slices.Clone(attribute.Bases)
		attribute.Bases = nil
	}
	switch typ := attribute.Type.(type) {
	case *Object:
		for _, field := range *typ {
			c.walk(field.Attribute, path+"."+field.Name)
		}
		c.walk(attribute.AdditionalAttributes, path+"[additional]")
	case *Array:
		c.walk(typ.ElemType, path+"[]")
	case *Map:
		c.walk(typ.KeyType, path+"[key]")
		c.walk(typ.ElemType, path+"[value]")
	case *Union:
		for _, branch := range typ.Values {
			c.walk(branch.Attribute, path+"."+branch.Name)
		}
	}
	c.seen[attribute] = true
}

func (c *designComposer) object(attribute *AttributeExpr, bases []*AttributeExpr, path string) {
	// Resolved explicit Extend contributors declare an otherwise untyped object.
	if attribute.Type == nil {
		attribute.Type = &Object{}
	}
	object := AsObject(attribute.Type)
	if object == nil {
		compositionError(attribute, "%s: Extend requires an object", path)
		return
	}
	for _, base := range bases {
		if base.effectiveObjectOpenness().AdmitsUndeclared() {
			continue
		}
		if attribute.reopensClosedObject() {
			compositionError(attribute, "%s: cannot reopen inherited closed object from %s", path, compositionSource(base))
			return
		}
		attribute.CloseObject()
	}
	// Every consumer owns its inherited fields, including nested mutable data.
	fields := &Object{}
	local := make(map[string]*AttributeExpr, len(*object))
	for _, field := range *object {
		fields.Set(field.Name, field.Attribute)
		local[field.Name] = field.Attribute
	}
	inherited, order := inheritedCompositionFields(bases)
	c.refineLocalMembers(attribute, object, bases, inherited, path)
	for _, name := range order {
		parents := slices.Concat(inherited[name], undeclaredMemberPolicies(bases, name))
		optional := optionalCompositionField(attribute, bases, name)
		if own := local[name]; own != nil {
			c.refine(own, parents, path+"."+name, true, name, optional)
			continue
		}
		field := c.copyAttribute(parents[0], c.owner)
		if len(parents) > 1 {
			// The first inherited declaration is a peer, never an authored
			// override that could resolve an ambiguous inherited default.
			field = &AttributeExpr{Type: field.Type}
			field.SetDSLSource(attribute.DSLSource())
			c.refine(field, parents, path+"."+name, false, name, optional)
		}
		fields.Set(name, field)
	}
	if !closedBasesDeclare(attribute, bases, fields, path) {
		return
	}
	c.composeAdditionalAttributes(attribute, bases, path)
	if named, ok := attribute.Type.(UserType); ok && named != Empty {
		owned := named.Dup(c.copyAttribute(named.Attribute(), c.owner))
		owned.Attribute().Type = fields
		attribute.Type = owned
	} else {
		attribute.Type = fields
	}
	if len(attribute.UserExamples) == 0 {
		for _, example := range bases[0].UserExamples {
			attribute.UserExamples = append(attribute.UserExamples, copyValueExample(example))
		}
	}
	c.retainConstraints(attribute, bases, path)
}

// refineLocalMembers applies the typed policy of each base to the members
// only the derived type declares: a typed base constrains every member it does
// not declare.
func (c *designComposer) refineLocalMembers(attribute *AttributeExpr, object *Object, bases []*AttributeExpr, inherited map[string][]*AttributeExpr, path string) {
	for _, field := range *object {
		if _, shared := inherited[field.Name]; shared {
			continue
		}
		if policies := undeclaredMemberPolicies(bases, field.Name); len(policies) > 0 {
			c.refine(field.Attribute, policies, path+"."+field.Name, true, field.Name, optionalCompositionField(attribute, bases, field.Name))
		}
	}
}

// optionalCompositionField reports whether neither the derived attribute nor
// any base requires the member name.
func optionalCompositionField(attribute *AttributeExpr, bases []*AttributeExpr, name string) bool {
	if attribute.IsRequired(name) {
		return false
	}
	for _, base := range bases {
		if base.IsRequired(name) {
			return false
		}
	}
	return true
}

// closedBasesDeclare applies each closed base per source: the base rejects
// every wire key it does not declare, including keys contributed by the
// derived type or by peer bases. The derived type itself semantically declares
// all composed fields, so its own closure never forbids an inherited field.
func closedBasesDeclare(attribute *AttributeExpr, bases []*AttributeExpr, fields *Object, path string) bool {
	for _, base := range bases {
		if base.effectiveObjectOpenness() != ObjectClosed {
			continue
		}
		declared := make(map[string]bool)
		if object := AsObject(base.Type); object != nil {
			for _, field := range *object {
				declared[JSONFieldName(ElementName(field.Name), field.Attribute)] = true
			}
		}
		for _, field := range *fields {
			name := JSONFieldName(ElementName(field.Name), field.Attribute)
			if name != "-" && !declared[name] {
				compositionError(attribute, "%s: closed Extend base %s does not declare %q; leave the base open and close the derived type instead",
					path, compositionSource(base), name)
				return false
			}
		}
	}
	return true
}

func inheritedCompositionFields(bases []*AttributeExpr) (map[string][]*AttributeExpr, []string) {
	inherited := make(map[string][]*AttributeExpr)
	var order []string
	for _, base := range bases {
		if fields := AsObject(base.Type); fields != nil {
			for _, field := range *fields {
				if _, found := inherited[field.Name]; !found {
					order = append(order, field.Name)
				}
				inherited[field.Name] = append(inherited[field.Name], field.Attribute)
			}
		}
	}
	return inherited, order
}

func (c *designComposer) refine(attribute *AttributeExpr, bases []*AttributeExpr, path string, authored bool, fieldName string, optional bool) {
	for _, base := range bases {
		if !refinesType(attribute.Type, base.Type, make(map[[2]DataType]bool)) {
			// Peers may choose the most-derived declared type, provided it
			// retains every earlier peer contract.
			if !authored && refinesType(base.Type, attribute.Type, make(map[[2]DataType]bool)) {
				attribute.Type = base.Type
			} else {
				compositionError(attribute, "%s: incompatible Extend types %q and %q from %s", path, attribute.Type.Name(), base.Type.Name(), compositionSource(base))
				return
			}
		}
	}
	allNullable, allAllowNull := true, true
	for _, base := range bases {
		allNullable = allNullable && IsNullable(base)
		allAllowNull = allAllowNull && AllowsNull(base)
	}
	if AllowsNull(attribute) && !allAllowNull {
		compositionError(attribute, "%s: incompatible nullable Extend refinement from %s", path, compositionSources(bases))
		return
	}
	attribute.Nullable = attribute.Nullable || allNullable
	if !c.composeAttributeMetadata(attribute, bases, path, fieldName, optional) {
		return
	}
	if !composeAttributeDocumentation(attribute, bases, path) {
		return
	}
	switch typ := attribute.Type.(type) {
	case *Array:
		owned := c.copyType(typ).(*Array)
		parents := make([]*AttributeExpr, len(bases))
		for i, base := range bases {
			parents[i] = AsArray(base.Type).ElemType
		}
		c.refine(owned.ElemType, parents, path+"[]", authored, "", false)
		attribute.Type = owned
	case *Map:
		owned := c.copyType(typ).(*Map)
		keys, values := make([]*AttributeExpr, len(bases)), make([]*AttributeExpr, len(bases))
		for i, base := range bases {
			keys[i], values[i] = AsMap(base.Type).KeyType, AsMap(base.Type).ElemType
		}
		c.refine(owned.KeyType, keys, path+"[key]", authored, "", false)
		c.refine(owned.ElemType, values, path+"[value]", authored, "", false)
		attribute.Type = owned
	case *Object:
		c.object(attribute, bases, path)
		return
	}
	c.retainConstraints(attribute, bases, path)
}

func (c *designComposer) retainConstraints(attribute *AttributeExpr, bases []*AttributeExpr, path string) {
	local := c.copyAttribute(attribute, nil)
	local.constraintParents = nil
	local.constraintLocal = nil
	attribute.constraintLocal = local
	attribute.constraintParents = make([]*AttributeExpr, len(bases))
	for i, base := range bases {
		attribute.constraintParents[i] = c.copyAttribute(base, nil)
	}
	attribute.compositionName = path
	c.composed = append(c.composed, attribute)
}

func refinesType(derived, base DataType, seen map[[2]DataType]bool) bool {
	if derived == base {
		return true
	}
	pair := [2]DataType{derived, base}
	if seen[pair] {
		return false
	}
	seen[pair] = true
	if named, ok := derived.(UserType); ok {
		if other, ok := base.(UserType); ok && named.ID() == other.ID() {
			return true
		}
		for _, ancestor := range named.Attribute().compositionBases {
			if refinesType(ancestor, base, seen) {
				return true
			}
		}
		return refinesType(named.Attribute().Type, base, seen)
	}
	if _, ok := base.(UserType); ok {
		return false
	}
	switch typ := derived.(type) {
	case *Array:
		other, ok := base.(*Array)
		return ok && (!other.NonNullableElems || typ.NonNullableElems) && refinesType(typ.ElemType.Type, other.ElemType.Type, seen)
	case *Map:
		other, ok := base.(*Map)
		return ok && refinesType(typ.KeyType.Type, other.KeyType.Type, seen) && refinesType(typ.ElemType.Type, other.ElemType.Type, seen)
	case *Object:
		other, ok := base.(*Object)
		if !ok || len(*typ) != len(*other) {
			return false
		}
		for _, field := range *other {
			own := typ.Attribute(field.Name)
			if own == nil || !refinesType(own.Type, field.Attribute.Type, seen) {
				return false
			}
		}
		return true
	}
	return false
}

func compositionSource(attribute *AttributeExpr) string {
	if file, line, ok := eval.ExpressionLocation(attribute); ok {
		return fmt.Sprintf("%q (%s:%d)", attribute.compositionName, file, line)
	}
	return fmt.Sprintf("%q", attribute.compositionName)
}

func compositionSources(attributes []*AttributeExpr) string {
	values := make([]string, len(attributes))
	for i, attribute := range attributes {
		values[i] = compositionSource(attribute)
	}
	return fmt.Sprint(values)
}

func compositionError(attribute *AttributeExpr, format string, args ...any) {
	file, line, _ := eval.ExpressionLocation(attribute)
	eval.Context.Record(&eval.Error{GoError: fmt.Errorf(format, args...), File: file, Line: line})
}

// composeAdditionalAttributes gives the derived object the conjunction of the
// typed policies of its sources for members no source declares. An unvalidated
// Any policy admits every value and does not constrain the conjunction. The
// derived type's own AdditionalAttributes refines every inherited one; without
// it, the first inherited policy is copied and refined by its peers. Policies
// whose JSON value kinds are disjoint admit no value, so the conjunction is a
// closed object, unless all are nullable: then only null remains, which no
// carrier represents. Overlapping policies whose types do not refine one another
// are rejected like incompatible Extend field types. A closed source keeps the
// object closed; reopening it was rejected earlier.
func (c *designComposer) composeAdditionalAttributes(attribute *AttributeExpr, bases []*AttributeExpr, path string) {
	inherited, unconstrained := inheritedAdditionalPolicies(bases)
	switch attribute.ObjectOpenness() {
	case ObjectClosed:
		return
	case ObjectOpen:
		if typed := slices.Concat(inherited, unconstrained); len(typed) > 0 {
			compositionError(attribute, "%s: cannot reopen inherited typed object from %s; its AdditionalAttributes still apply", path, compositionSources(typed))
			return
		}
	}
	if len(inherited) == 0 {
		// Only Any policies remain: keep typed extras, which an open object
		// would not carry.
		if len(unconstrained) > 0 && attribute.ObjectOpenness() == ObjectOpennessUnset {
			attribute.AdditionalAttributes = c.copyAttribute(unconstrained[0], c.owner)
		}
		return
	}
	if attribute.AdditionalAttributes != nil && admitsAnyValue(attribute.AdditionalAttributes) {
		attribute.AdditionalAttributes = nil
	}
	authored := attribute.AdditionalAttributes != nil
	policies := inherited
	if authored {
		policies = append([]*AttributeExpr{attribute.AdditionalAttributes}, inherited...)
	}
	if !composablePolicies(attribute, policies, path) {
		return
	}
	additional := attribute.AdditionalAttributes
	if !authored {
		additional = c.copyAttribute(inherited[0], c.owner)
		attribute.AdditionalAttributes = additional
		if len(inherited) == 1 {
			return
		}
		// As for a member that several bases declare, the first inherited
		// policy is a peer, never a local override of the others.
		additional = &AttributeExpr{Type: additional.Type}
		additional.SetDSLSource(attribute.DSLSource())
		attribute.AdditionalAttributes = additional
	}
	c.refine(additional, inherited, path+"[additional]", authored, "", true)
	// Decide emptiness now, before any type extending this one copies the
	// policy: when no value satisfies every typed policy, the object admits
	// no undeclared member.
	empty, err := emptyPolicy(additional)
	if err != nil {
		compositionError(attribute, "%s[additional]: %v", path, err)
		return
	}
	if !empty {
		return
	}
	c.composed = slices.DeleteFunc(c.composed, func(composed *AttributeExpr) bool {
		return composed == additional
	})
	attribute.AdditionalAttributes = nil
	attribute.CloseObject()
}

// inheritedAdditionalPolicies returns the typed policies of the bases that
// constrain values, then the unvalidated Any policies, which do not.
func inheritedAdditionalPolicies(bases []*AttributeExpr) (constraining, unconstrained []*AttributeExpr) {
	for _, base := range bases {
		additional := base.effectiveAdditionalAttributes()
		switch {
		case additional == nil:
		case admitsAnyValue(additional):
			unconstrained = append(unconstrained, additional)
		default:
			constraining = append(constraining, additional)
		}
	}
	return constraining, unconstrained
}

// composablePolicies reports whether the policies can be conjoined by
// refinement. Disjoint policies close the object instead, and policies whose
// conjunction Loom cannot carry are rejected.
func composablePolicies(attribute *AttributeExpr, policies []*AttributeExpr, path string) bool {
	switch policyOverlap(policies) {
	case policiesDisjoint:
		if allNullable(policies) {
			compositionError(attribute, "%s: unsupported Extend composition: additional attributes from %s admit only null", path, compositionSources(policies))
			return false
		}
		attribute.AdditionalAttributes = nil
		attribute.CloseObject()
		return false
	case policiesIncomparable:
		compositionError(attribute, "%s: unsupported Extend composition: additional attribute types from %s overlap but neither refines the other", path, compositionSources(policies))
		return false
	}
	return true
}

// emptyPolicy reports whether the effective constraints of a composed policy
// admit no value: contradictory bounds or an empty enum intersection. Other
// constraint errors, such as an enum member that an inherited bound rejects,
// are design errors, as they are for declared members.
func emptyPolicy(policy *AttributeExpr) (bool, error) {
	constraints, err := EffectiveConstraintsFor(policy)
	if err != nil {
		var empty emptyConstraintDomainError
		if errors.As(err, &empty) {
			return true, nil
		}
		return false, err
	}
	values, present := constraints.EnumCandidates()
	return present && len(values) == 0, nil
}

// policyOverlap classifies the conjunction of typed additional-member
// policies: every pair refines one way, some pair has disjoint JSON value
// kinds, or some pair overlaps without either type refining the other.
func policyOverlap(policies []*AttributeExpr) additionalPolicyOverlap {
	overlap := policiesRefine
	for i, left := range policies {
		for _, right := range policies[i+1:] {
			if refinesType(left.Type, right.Type, make(map[[2]DataType]bool)) ||
				refinesType(right.Type, left.Type, make(map[[2]DataType]bool)) {
				continue
			}
			if disjointJSONKinds(left.Type, right.Type) {
				return policiesDisjoint
			}
			overlap = policiesIncomparable
		}
	}
	return overlap
}

// admitsAnyValue reports whether a typed additional-member policy is Any
// without validations, so it admits every JSON value.
func admitsAnyValue(policy *AttributeExpr) bool {
	return policy.Type == Any && policy.Validation == nil
}

// disjointJSONKinds reports whether no JSON value can have both types' JSON
// kinds. Integers are numbers, bytes are strings, and maps and objects are
// both JSON objects; Any and unions are not classified.
func disjointJSONKinds(left, right DataType) bool {
	l, r := jsonValueKind(left), jsonValueKind(right)
	return l != "" && r != "" && l != r
}

func jsonValueKind(t DataType) string {
	seen := make(map[DataType]bool)
	for named, ok := t.(UserType); ok && !seen[t]; named, ok = t.(UserType) {
		seen[t] = true
		t = named.Attribute().Type
	}
	switch t.Kind() {
	case BooleanKind:
		return "boolean"
	case IntKind, Int32Kind, Int64Kind, UIntKind, UInt32Kind, UInt64Kind, Float32Kind, Float64Kind:
		return "number"
	case StringKind, BytesKind:
		return "string"
	case ArrayKind:
		return "array"
	case MapKind, ObjectKind:
		return "object"
	default:
		return ""
	}
}

func allNullable(attributes []*AttributeExpr) bool {
	for _, attribute := range attributes {
		if !IsNullable(attribute) {
			return false
		}
	}
	return true
}
