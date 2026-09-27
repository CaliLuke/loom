package expr

import (
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/naming"
)

type goFieldNameValidator struct {
	seen   map[*AttributeExpr]bool
	errors *eval.ValidationErrors
}

// validateGoFieldNames checks service data, excluding transport mappings whose
// attributes can name distinct metadata elements without declaring Go fields.
func (r *RootExpr) validateGoFieldNames() *eval.ValidationErrors {
	v := &goFieldNameValidator{seen: make(map[*AttributeExpr]bool), errors: new(eval.ValidationErrors)}
	for _, ut := range r.Types {
		v.walk(ut.Attribute(), ut, ut.Name())
	}
	for _, rt := range r.ResultTypes {
		v.walk(rt.Attribute(), rt, rt.Name())
	}
	for _, e := range r.Errors {
		v.walk(e.AttributeExpr, e, e.Name)
	}
	for _, s := range r.Services {
		for _, e := range s.Errors {
			v.walk(e.AttributeExpr, e, e.Name)
		}
		for _, m := range s.Methods {
			v.walk(m.Payload, m, "payload")
			v.walk(m.StreamingPayload, m, "streaming payload")
			v.walk(m.Result, m, "result")
			v.walk(m.StreamingResult, m, "streaming result")
			for _, e := range m.Errors {
				v.walk(e.AttributeExpr, e, e.Name)
			}
		}
	}
	return v.errors
}

// effectiveGoFields follows the same exact-key replacement order as Extend.
// Reference fields and their metadata are selected during DSL evaluation;
// finalization does not merge unselected reference fields into the object.
func effectiveGoFields(att *AttributeExpr, active map[*AttributeExpr]bool) *Object {
	fields := &Object{}
	if att == nil || active[att] {
		return fields
	}
	active[att] = true
	defer delete(active, att)
	merge := func(other *Object) {
		for _, field := range *other {
			fields.Set(field.Name, field.Attribute)
		}
	}
	switch dt := att.Type.(type) {
	case UserType:
		merge(effectiveGoFields(dt.Attribute(), active))
	case *Object:
		merge(dt)
	}
	for _, base := range att.Bases {
		merge(effectiveGoFields(&AttributeExpr{Type: base}, active))
	}
	return fields
}

func (v *goFieldNameValidator) walk(att *AttributeExpr, parent eval.Expression, path string) {
	if att == nil || v.seen[att] {
		return
	}
	v.seen[att] = true
	switch dt := att.Type.(type) {
	case UserType:
		if len(att.Bases) > 0 && IsObject(att.Type) {
			v.walkObject(att, parent, path)
		} else {
			v.walk(dt.Attribute(), parent, path)
		}
	case *Object:
		v.walkObject(att, parent, path)
	case *Array:
		v.walk(dt.ElemType, parent, path+"[]")
	case *Map:
		v.walk(dt.KeyType, parent, path+"[key]")
		v.walk(dt.ElemType, parent, path+"[value]")
	case *Union:
		for _, branch := range dt.Values {
			v.walk(branch.Attribute, parent, branch.Name)
		}
	}
}

func (v *goFieldNameValidator) walkObject(att *AttributeExpr, parent eval.Expression, path string) {
	fields := effectiveGoFields(att, make(map[*AttributeExpr]bool))
	owners := make(map[string]string, len(*fields))
	for _, field := range *fields {
		goName := naming.GoifyAttribute(field.Name, field.Attribute.Meta, true)
		if previous, exists := owners[goName]; exists {
			v.errors.Add(parent, "attributes %q and %q in %q both generate Go field %q; rename one or set distinct struct:field:name metadata", previous, field.Name, path, goName)
		} else {
			owners[goName] = field.Name
		}
		v.walk(field.Attribute, parent, path+"."+field.Name)
	}
}
