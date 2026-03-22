package testdata

import "github.com/CaliLuke/loom/expr"

var FinalizeEndpointBodyAsExtendedType = &expr.UserTypeExpr{
	AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{
			{"id", &expr.AttributeExpr{Type: expr.String}},
			{"name", &expr.AttributeExpr{Type: expr.String}},
		},
	},
	TypeName: "FinalizeEndpointBodyAsExtendedType",
}

var FinalizeEndpointBodyAsPropWithExtendedType = &expr.UserTypeExpr{
	AttributeExpr: &expr.AttributeExpr{
		Type: &expr.Object{
			{"id", &expr.AttributeExpr{Type: expr.String}},
			{"name", &expr.AttributeExpr{Type: expr.String}},
		},
	},
	TypeName: "FinalizeEndpointBodyAsPropWithExtendedTypeDSL",
}
