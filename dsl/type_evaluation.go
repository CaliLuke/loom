package dsl

import (
	"slices"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

type (
	// typeEvaluation tracks the execution of the DSL of a type declared with
	// Type so that an attribute DSL can evaluate the type before copying it.
	typeEvaluation struct {
		userType expr.UserType
		fn       func()
		running  bool
		done     bool
		// kept lists the attributes of local copies that still reference
		// the type because its DSL had not completed when they were copied.
		kept []keptTypeReference
	}

	// keptTypeReference distinguishes a structural back-edge from a requested
	// local snapshot. Copies retain that intent through KeptReference.Source.
	keptTypeReference struct {
		attribute *expr.AttributeExpr
		owner     expr.UserType
		localized bool
	}

	// typeEvaluations indexes the type evaluations of one design root.
	typeEvaluations struct {
		root   *expr.RootExpr
		byType map[*expr.AttributeExpr]*typeEvaluation
		kept   map[*expr.AttributeExpr]keptTypeReference
	}
)

// evaluations records the type evaluations of the design being executed.
var evaluations typeEvaluations

// typeDSL returns the DSL of ut: fn, run at most once, either by the DSL
// engine or earlier when an attribute DSL needs a complete copy of ut.
func typeDSL(ut expr.UserType, fn func()) func() {
	if fn == nil {
		return nil
	}
	ev := &typeEvaluation{userType: ut, fn: fn}
	evaluations.current().byType[ut.Attribute()] = ev
	return func() {
		if ev.running || ev.done {
			return
		}
		ev.running = true
		ev.fn()
		ev.running = false
		ev.done = true
		ev.completeKeptReferences()
	}
}

// localizeAttribute replaces the type of att with a copy that an attribute
// DSL may refine without changing the shared types it references. Inside a
// DSL body, the DSL of every referenced type declared with Type runs first so
// that the copy is complete, as if the types had been declared before. A type
// that cannot be completed yet, because its DSL is running or because att is
// declared outside any DSL body where types cannot be evaluated, stays
// referenced until its DSL ends. Explicit local occurrences then receive a
// complete copy; structural self-references retain their enclosing owner.
func localizeAttribute(att, parent *expr.AttributeExpr) {
	if att.Type == nil || referencesAttribute(att.Type, parent) {
		return
	}
	_, declaring := eval.Current().(eval.TopExpr)
	incomplete := evaluateReferencedTypes(att.Type, !declaring)
	if len(incomplete) == 0 {
		att.Type = expr.Dup(att.Type)
		return
	}
	roots := localizedTypeRoots(att.Type)
	copied, kept := expr.DupKeeping(att.Type, incomplete)
	att.Type = copied
	owner := evaluations.current().enclosingType()
	for _, reference := range kept {
		evaluations.current().keep(reference, owner, roots[reference.Source])
	}
	if ut, ok := copied.(expr.UserType); ok && slices.Contains(incomplete, ut) {
		evaluations.current().keep(expr.KeptReference{Source: att, Copy: att}, owner, true)
	}
}

// localizedTypeRoots identifies named occurrences directly owned by the
// localized builtin shape. Descending into a named declaration instead reaches
// structural dependencies, whose self-references retain their enclosing owner.
func localizedTypeRoots(typ expr.DataType) map[*expr.AttributeExpr]bool {
	roots := make(map[*expr.AttributeExpr]bool)
	seen := make(map[expr.DataType]bool)
	var walk func(expr.DataType)
	var attribute func(*expr.AttributeExpr)
	attribute = func(att *expr.AttributeExpr) {
		if att == nil || roots[att] {
			return
		}
		roots[att] = true
		walk(att.Type)
		attribute(att.AdditionalAttributes)
	}
	walk = func(typ expr.DataType) {
		if typ == nil || seen[typ] {
			return
		}
		seen[typ] = true
		switch actual := typ.(type) {
		case *expr.Array:
			attribute(actual.ElemType)
		case *expr.Map:
			attribute(actual.KeyType)
			attribute(actual.ElemType)
		case *expr.Object:
			for _, field := range *actual {
				attribute(field.Attribute)
			}
		case *expr.Union:
			for _, branch := range actual.Values {
				attribute(branch.Attribute)
			}
		}
	}
	walk(typ)
	return roots
}

// evaluateReferencedTypes returns the types declared with Type that dt
// references, including through typed AdditionalAttributes, and whose DSL has
// not completed. When evaluate is true it first runs the DSL of the referenced
// types that have not been evaluated yet, so that only the types whose DSL is
// running remain incomplete.
func evaluateReferencedTypes(dt expr.DataType, evaluate bool) []expr.UserType {
	var incomplete []expr.UserType
	seen := make(map[expr.UserType]struct{})
	var walk func(expr.DataType)
	walkAttribute := func(att *expr.AttributeExpr) {
		walk(att.Type)
		if att.AdditionalAttributes != nil {
			walk(att.AdditionalAttributes.Type)
		}
	}
	walk = func(dt expr.DataType) {
		switch actual := dt.(type) {
		case expr.UserType:
			if _, ok := seen[actual]; ok {
				return
			}
			seen[actual] = struct{}{}
			if ev, ok := evaluations.current().byType[actual.Attribute()]; ok {
				if ev.running || (!ev.done && !evaluate) {
					incomplete = append(incomplete, actual)
					return
				}
				if !ev.done {
					eval.Execute(actual.Attribute().DSLFunc, actual.Attribute())
				}
			}
			walkAttribute(actual.Attribute())
		case *expr.Array:
			walkAttribute(actual.ElemType)
		case *expr.Map:
			walkAttribute(actual.KeyType)
			walkAttribute(actual.ElemType)
		case *expr.Object:
			for _, nat := range *actual {
				walkAttribute(nat.Attribute)
			}
		case *expr.Union:
			for _, nat := range actual.Values {
				walkAttribute(nat.Attribute)
			}
		}
	}
	walk(dt)
	return incomplete
}

// current returns the evaluations of the current design root.
func (e *typeEvaluations) current() *typeEvaluations {
	if e.root != expr.Root || e.byType == nil {
		*e = typeEvaluations{
			root:   expr.Root,
			byType: make(map[*expr.AttributeExpr]*typeEvaluation),
			kept:   make(map[*expr.AttributeExpr]keptTypeReference),
		}
	}
	return e
}

// enclosingType finds the authored declaration whose DSL owns this occurrence,
// including occurrences nested under inline object/collection DSL expressions.
func (e *typeEvaluations) enclosingType() expr.UserType {
	for index := len(eval.Context.Stack) - 1; index >= 0; index-- {
		var attribute *expr.AttributeExpr
		switch current := eval.Context.Stack[index].(type) {
		case *expr.AttributeExpr:
			attribute = current
		case expr.CompositeExpr:
			attribute = current.Attribute()
		}
		if evaluation := e.byType[attribute]; evaluation != nil {
			return evaluation.userType
		}
	}
	return nil
}

// keep records both source/copy correspondence and the DSL's binding intent.
// A copied explicit localization remains a snapshot, even when its target has
// the same declaration identity as the enclosing structural owner.
func (e *typeEvaluations) keep(reference expr.KeptReference, owner expr.UserType, localized bool) {
	att := reference.Copy
	ut, ok := att.Type.(expr.UserType)
	if !ok {
		return
	}
	ev := e.byType[ut.Attribute()]
	if ev == nil {
		return
	}
	if previous, ok := e.kept[reference.Source]; ok {
		localized = localized || previous.localized
	}
	if _, ok := e.kept[att]; ok {
		return
	}
	kept := keptTypeReference{attribute: att, owner: owner, localized: localized}
	e.kept[att] = kept
	ev.kept = append(ev.kept, kept)
}

// completeKeptReferences publishes complete local snapshots while retaining
// structural self-references on their enclosing declaration. Occurrence
// validations stay independent of the declaration's physical field layout.
func (ev *typeEvaluation) completeKeptReferences() {
	e := evaluations.current()
	refs := ev.kept
	ev.kept = nil
	for _, reference := range refs {
		att := reference.attribute
		delete(e.kept, att)
		if att.Type != ev.userType {
			continue
		}
		if !reference.localized && reference.owner == ev.userType {
			continue // Structural recursion stays bound to its authored owner.
		}
		copied, kept := expr.DupKeeping(ev.userType, evaluateReferencedTypes(ev.userType, true))
		att.Type = copied
		for _, k := range kept {
			e.keep(k, reference.owner, false)
		}
	}
}
