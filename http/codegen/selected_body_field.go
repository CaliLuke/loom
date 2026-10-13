package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// selectedServiceField retains both identities of a body-selected member. Name
// and Key belong to the design; GoName names the actual service declaration.
// Attribute carries the existing effective transform representation.
type selectedServiceField struct {
	Name      string
	Key       string
	GoName    string
	Attribute *expr.AttributeExpr
}

func selectBodyField(parent, body *expr.AttributeExpr) selectedServiceField {
	if body == nil || body.Type == expr.Empty {
		return selectedServiceField{Attribute: parent}
	}
	origin := body.Meta["origin:attribute"]
	if len(origin) == 0 {
		return selectedServiceField{Attribute: parent}
	}
	name := origin[0]
	key, attribute := parent.FindAttribute(name)
	return selectedServiceField{
		Name:      name,
		Key:       key,
		GoName:    codegen.GoifyAtt(attribute, name, true),
		Attribute: serviceFieldTransformAttribute(parent, name, attribute),
	}
}
