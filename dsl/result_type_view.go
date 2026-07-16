package dsl

import (
	"fmt"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

const (
	viewDefinitionMetaKey = "loom:view:definition"
	viewOptionalMetaKey   = "loom:view:optional"
)

// View has two usages:
//
// - when used inside a ResultType DSL function it defines a view for the result
// type. A view lists a subset of the result type attributes that are used when
// marshalling responses.
//
// - when used inside a Result DSL function it defines the view used to marshal
// the result type returned by the method.
//
// Note that the view used to render a response can also be set dynamically by
// the method code in which case the result function should not specify a view
// in the design.  The attribute names listed in a view must be identical to
// existing attributes in the result type on which the view is defined. If an
// attribute is itself a result type then the view may specify which view to use
// when marshaling the attribute using the View function recursively, see
// example below. All result types must have a view called "default" which is
// the view used to marshal results when no specific view is specified.
//
// View must appear in a ResultType or a Result expression.
//
// View accepts two arguments for the first usage: the view name and its
// defining DSL.  View accepts a single argument for the second usage: the view
// name used to render the result.
//
// Examples:
//
//	// MyResultType defines 2 views.
//	var MyResultType = ResultType("application/vnd.loom.my", func() {
//	    Attributes(func() {
//	        Attribute("id", String)
//	        Attribute("name", String)
//	        Attribute("origin", OriginResult)
//	    })
//
//	    View("default", func() {
//	        // "id" and "name" must be result type attributes
//	        Attribute("id")
//	        Attribute("name")
//	    })
//
//	    View("extended", func() {
//	        Attribute("id")
//	        Attribute("name")
//	        Attribute("origin", func() {
//	            // Use view "extended" to render attribute "origin"
//	            View("extended")
//	        })
//	    })
//	})
//
//	// MyMethod uses the extended view of MyResultType to marshal the
//	// response.
//	var _ = Service("MyService", func() {
//	    Method("MyMethod", func() {
//	        Result(MyResultType, func() { View("extended") })
//	        GRPC(func() {})
//	    })
//	})
func View(name string, adsl ...func()) {
	if adsl == nil {
		switch e := eval.Current().(type) {
		case *expr.ResultTypeExpr:
			e.AddMeta(expr.ViewMetaKey, name)
		case *expr.AttributeExpr:
			e.AddMeta(expr.ViewMetaKey, name)
		default:
			eval.IncompatibleDSL()
		}
		return
	}
	rt, ok := eval.Current().(*expr.ResultTypeExpr)
	if !ok {
		eval.IncompatibleDSL()
		return
	}
	if rt.View(name) != nil {
		eval.ReportError("view %q is defined multiple times in result type %q", name, rt.TypeName)
		return
	}
	at := &expr.AttributeExpr{Meta: expr.MetaExpr{viewDefinitionMetaKey: nil}}
	ok = false
	if len(adsl) > 0 {
		ok = eval.Execute(adsl[0], at)
	} else if a, ok := rt.Type.(*expr.Array); ok {
		// inherit view from collection element if present
		if elem := a.ElemType; elem != nil {
			if pa, ok2 := elem.Type.(*expr.ResultTypeExpr); ok2 {
				if v := pa.View(name); v != nil {
					at = v.AttributeExpr
					rt = pa
				} else {
					eval.ReportError("unknown view %#v", name)
					return
				}
			}
		}
	}
	if ok {
		view, err := buildView(name, rt, at)
		if err != nil {
			eval.ReportError(err.Error())
			return
		}
		rt.Views = append(rt.Views, view)
	}
}

// ViewRequired marks fields included in the current result view as required,
// overriding optional canonical fields for that view.
func ViewRequired(names ...string) {
	at, ok := eval.Current().(*expr.AttributeExpr)
	if !ok || at.Meta == nil {
		eval.IncompatibleDSL()
		return
	}
	if _, ok := at.Meta[viewDefinitionMetaKey]; !ok {
		eval.IncompatibleDSL()
		return
	}
	if at.Validation == nil {
		at.Validation = &expr.ValidationExpr{}
	}
	at.Validation.AddRequired(names...)
}

// ViewOptional marks fields included in the current result view as optional,
// overriding required canonical fields for that view.
func ViewOptional(names ...string) {
	at, ok := eval.Current().(*expr.AttributeExpr)
	if !ok || at.Meta == nil {
		eval.IncompatibleDSL()
		return
	}
	if _, ok := at.Meta[viewDefinitionMetaKey]; !ok {
		eval.IncompatibleDSL()
		return
	}
	at.Meta[viewOptionalMetaKey] = append(at.Meta[viewOptionalMetaKey], names...)
}

// buildView builds a view expression given an attribute and a corresponding
// result type. The attribute must be an object listing the child attributes
// that make up the view.
func buildView(name string, mt *expr.ResultTypeExpr, at *expr.AttributeExpr) (*expr.ViewExpr, error) {
	if at.Type == nil {
		return nil, fmt.Errorf("invalid view DSL")
	}
	o := expr.AsObject(at.Type)
	if o == nil {
		return nil, fmt.Errorf("invalid view DSL")
	}
	requiredOverrides := viewRequiredOverrides(at)
	optionalOverrides := at.Meta[viewOptionalMetaKey]
	selected := make(map[string]struct{}, len(*o))
	for _, nat := range *o {
		selected[nat.Name] = struct{}{}
	}
	if err := validateViewRequirednessOverrides(requiredOverrides, optionalOverrides, selected); err != nil {
		return nil, err
	}
	for _, nat := range *o {
		n := nat.Name
		cat := nat.Attribute
		if existing := mt.Find(n); existing != nil {
			dup := expr.DupAtt(existing)
			if v, ok := cat.Meta.Last(expr.ViewMetaKey); ok {
				dup.AddMeta("view", v)
			}
			o.Set(n, dup)
		} else if n != "links" {
			return nil, fmt.Errorf("unknown attribute %#v", n)
		}
	}
	at.Validation = effectiveViewValidation(mt.AttributeExpr, selected, requiredOverrides, optionalOverrides)
	delete(at.Meta, viewDefinitionMetaKey)
	delete(at.Meta, viewOptionalMetaKey)
	return &expr.ViewExpr{
		AttributeExpr: at,
		Name:          name,
		Parent:        mt,
	}, nil
}

func viewRequiredOverrides(at *expr.AttributeExpr) []string {
	if at.Validation == nil {
		return nil
	}
	return append([]string(nil), at.Validation.Required...)
}

func validateViewRequirednessOverrides(required, optional []string, selected map[string]struct{}) error {
	requiredSet := make(map[string]struct{}, len(required))
	for _, name := range required {
		if _, ok := selected[name]; !ok {
			return fmt.Errorf("view requiredness override references unknown or unselected attribute %q", name)
		}
		requiredSet[name] = struct{}{}
	}
	for _, name := range optional {
		if _, ok := selected[name]; !ok {
			return fmt.Errorf("view requiredness override references unknown or unselected attribute %q", name)
		}
		if _, ok := requiredSet[name]; ok {
			return fmt.Errorf("view attribute %q cannot be both required and optional", name)
		}
	}
	return nil
}

func effectiveViewValidation(
	canonical *expr.AttributeExpr,
	selected map[string]struct{},
	requiredOverrides, optionalOverrides []string,
) *expr.ValidationExpr {
	required := make(map[string]struct{}, len(selected))
	if canonical.Validation != nil {
		for _, name := range canonical.Validation.Required {
			if _, ok := selected[name]; ok {
				required[name] = struct{}{}
			}
		}
	}
	for _, name := range optionalOverrides {
		delete(required, name)
	}
	for _, name := range requiredOverrides {
		required[name] = struct{}{}
	}
	ordered := make([]string, 0, len(required))
	for _, nat := range *expr.AsObject(canonical.Type) {
		if _, ok := required[nat.Name]; ok {
			ordered = append(ordered, nat.Name)
		}
	}
	if canonical.Validation == nil && len(ordered) == 0 {
		return nil
	}
	validation := &expr.ValidationExpr{}
	if canonical.Validation != nil {
		validation = canonical.Validation.Dup()
	}
	validation.Required = ordered
	return validation
}
