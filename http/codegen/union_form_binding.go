package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/representation"
)

// bindHTTPUnionDeclaration separates shared name allocation from package-owned physical
// field and validation descriptors. Client and server can use different presence
// carriers under the same allocated names; neither publication mutates the other.
func bindHTTPUnionDeclaration(allocated *uniongen.Type, union *expr.Union, context *codegen.AttributeContext) *uniongen.Type {
	result := *allocated
	result.Fields = make([]*uniongen.Field, len(allocated.Fields))
	result.HasScalarFormBranch = false
	if allocated.JSON != nil {
		binding := *allocated.JSON
		binding.Branches = make([]*unionjson.Branch, len(allocated.JSON.Branches))
		result.JSON = &binding
	}
	for index, source := range allocated.Fields {
		field := *source
		attribute := union.Values[index].Attribute
		branchContext := context.Dup()
		representation.ApplyLayout(branchContext, attribute)
		uniongen.ConfigureFormField(&field, attribute, branchContext)
		if union.Untagged {
			if _, named := attribute.Type.(expr.UserType); named && !expr.IsNullable(attribute) && !expr.IsPrimitive(attribute.Type) && !expr.IsUnion(attribute.Type) {
				field.ValidateRef = unionBranchValidateRef(field.FieldType)
				field.ValidateCode = "err = " + field.ValidateRef
			} else {
				validationContext, _ := branchContext.UnionBranchContext(attribute)
				field.ValidateCode = codegen.ValidationCode(attribute, nil, validationContext, true, expr.IsAlias(attribute.Type), false, "v")
			}
			branch := *allocated.JSON.Branches[index]
			branch.Validate = field.ValidateCode
			result.JSON.Branches[index] = &branch
		}
		result.Fields[index] = &field
		result.HasScalarFormBranch = result.HasScalarFormBranch || (!field.FlatFormObject && field.FormError == "")
	}
	return &result
}

func unionBranchValidateRef(fieldType string) string {
	typeName := strings.TrimPrefix(fieldType, "*")
	return "Validate" + typeName + "(v)"
}
