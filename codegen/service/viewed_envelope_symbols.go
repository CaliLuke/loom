package service

import (
	"fmt"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// allocateViewedEnvelope gives the physical wrapper its own nominal identity.
// UserTypeExpr.Hash uses TypeName, not UID. A role-qualified internal name keeps
// this declaration distinct from projected types, whose names end in View. The
// scope binds that identity to the preferred public Go name before any reference.
func allocateViewedEnvelope(att, projected *expr.AttributeExpr, scope *codegen.NameScope) *expr.AttributeExpr {
	rt := att.Type.(*expr.ResultTypeExpr)
	occurrenceNullable := att.Nullable && !expr.IsNullable(&expr.AttributeExpr{Type: rt})
	identity := fmt.Sprintf("loom:viewed-envelope(%q,%q,%t)", rt.ID(), rt.Name(), occurrenceNullable)
	preferred := rt.Name()
	if occurrenceNullable {
		preferred += "Nullable"
	}
	payload := *projected
	payload.Description = "Type to project"
	typ := &expr.UserTypeExpr{
		TypeName: identity,
		UID:      identity,
		AttributeExpr: &expr.AttributeExpr{
			Type: &expr.Object{
				{Name: "projected", Attribute: &payload},
				{Name: "view", Attribute: &expr.AttributeExpr{Type: expr.String, Description: "View to render"}},
			},
			Validation: &expr.ValidationExpr{Required: []string{"projected", "view"}},
		},
	}
	scope.HashedUniqueFamily(typ, codegen.Goify(preferred, true), func(name string) []string {
		return viewTypeFamilies(typ, name)
	})
	return &expr.AttributeExpr{Type: typ, Meta: att.Meta}
}
