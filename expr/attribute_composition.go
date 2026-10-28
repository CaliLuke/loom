package expr

import (
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
		if value, present := constraints.Default(); present {
			attribute.DefaultValue = value
		}
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
	if err := attribute.resolveTypeRef(); err != nil {
		compositionError(attribute, "%s: %v", path, err)
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
	object := AsObject(attribute.Type)
	if object == nil {
		compositionError(attribute, "%s: Extend requires an object", path)
		return
	}
	for _, base := range bases {
		if allowsAdditionalProperties(base) {
			continue
		}
		if _, explicit := attribute.Meta["openapi:additionalProperties"]; explicit && allowsAdditionalProperties(attribute) || attribute.removedMeta["openapi:additionalProperties"] {
			compositionError(attribute, "%s: cannot reopen inherited closed object from %s", path, compositionSource(base))
			return
		}
		if attribute.Meta == nil {
			attribute.Meta = make(MetaExpr)
		}
		attribute.Meta["openapi:additionalProperties"] = []string{"false"}
	}
	// Every consumer owns its inherited fields, including nested mutable data.
	fields := &Object{}
	local := make(map[string]*AttributeExpr, len(*object))
	for _, field := range *object {
		fields.Set(field.Name, field.Attribute)
		local[field.Name] = field.Attribute
	}
	inherited, order := inheritedCompositionFields(bases)
	for _, name := range order {
		parents := inherited[name]
		optional := !attribute.IsRequired(name)
		for _, base := range bases {
			optional = optional && !base.IsRequired(name)
		}
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
