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
		kept []*expr.AttributeExpr
	}

	// typeEvaluations indexes the type evaluations of one design root.
	typeEvaluations struct {
		root   *expr.RootExpr
		byType map[*expr.AttributeExpr]*typeEvaluation
		kept   map[*expr.AttributeExpr]struct{}
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
// referenced until its DSL ends and is then replaced by a complete copy.
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
	copied, kept := expr.DupKeeping(att.Type, incomplete)
	att.Type = copied
	if ut, ok := copied.(expr.UserType); ok && slices.Contains(incomplete, ut) {
		kept = append(kept, att)
	}
	for _, k := range kept {
		evaluations.current().keep(k)
	}
}

// isKeptReference reports whether att is a kept reference. Its future copy
// of the type requires the fields that att requires.
func isKeptReference(att *expr.AttributeExpr) bool {
	_, ok := evaluations.current().kept[att]
	return ok
}

// evaluateReferencedTypes returns the types declared with Type that dt
// references and whose DSL has not completed. When evaluate is true it first
// runs the DSL of the referenced types that have not been evaluated yet, so
// that only the types whose DSL is running remain incomplete.
func evaluateReferencedTypes(dt expr.DataType, evaluate bool) []expr.UserType {
	var incomplete []expr.UserType
	seen := make(map[string]struct{})
	var walk func(expr.DataType)
	walk = func(dt expr.DataType) {
		switch actual := dt.(type) {
		case expr.UserType:
			if _, ok := seen[actual.ID()]; ok {
				return
			}
			seen[actual.ID()] = struct{}{}
			if ev, ok := evaluations.current().byType[actual.Attribute()]; ok {
				if ev.running || (!ev.done && !evaluate) {
					incomplete = append(incomplete, actual)
					return
				}
				if !ev.done {
					eval.Execute(actual.Attribute().DSLFunc, actual.Attribute())
				}
			}
			walk(actual.Attribute().Type)
		case *expr.Array:
			walk(actual.ElemType.Type)
		case *expr.Map:
			walk(actual.KeyType.Type)
			walk(actual.ElemType.Type)
		case *expr.Object:
			for _, nat := range *actual {
				walk(nat.Attribute.Type)
			}
		case *expr.Union:
			for _, nat := range actual.Values {
				walk(nat.Attribute.Type)
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
			kept:   make(map[*expr.AttributeExpr]struct{}),
		}
	}
	return e
}

// keep records att, whose type is a type whose DSL has not completed, so that
// att references a complete copy of the type once its DSL ends.
func (e *typeEvaluations) keep(att *expr.AttributeExpr) {
	ut, ok := att.Type.(expr.UserType)
	if !ok {
		return
	}
	ev, ok := e.byType[ut.Attribute()]
	if !ok {
		return
	}
	if _, ok := e.kept[att]; ok {
		return
	}
	e.kept[att] = struct{}{}
	ev.kept = append(ev.kept, att)
}

// completeKeptReferences gives each kept reference to the evaluated type its
// own complete copy of the type, which requires the fields that the reference
// requires.
func (ev *typeEvaluation) completeKeptReferences() {
	e := evaluations.current()
	refs := ev.kept
	ev.kept = nil
	for _, att := range refs {
		delete(e.kept, att)
		if att.Type != ev.userType {
			continue
		}
		copied, kept := expr.DupKeeping(ev.userType, evaluateReferencedTypes(ev.userType, true))
		if v := att.Validation; v != nil && len(v.Required) > 0 {
			ut := copied.(expr.UserType)
			if ut.Attribute().Validation == nil {
				ut.Attribute().Validation = &expr.ValidationExpr{}
			}
			ut.Attribute().Validation.AddRequired(v.Required...)
		}
		att.Type = copied
		for _, k := range kept {
			e.keep(k)
		}
	}
}
