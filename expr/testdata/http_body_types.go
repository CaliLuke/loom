package testdata

import "github.com/CaliLuke/loom/expr"

var FinalizeEndpointBodyAsExtendedType = &expr.UserTypeExpr{
	AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	},
	TypeName: "FinalizeEndpointBodyAsExtendedType",
}

var FinalizeEndpointBodyAsPropWithExtendedType = &expr.UserTypeExpr{
	AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{
			{Name: "id", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "name", Attribute: &expr.AttributeExpr{Type: expr.String}},
		},
	},
	TypeName: "FinalizeEndpointBodyAsPropWithExtendedTypeDSL",
}
