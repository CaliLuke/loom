package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestOutgoingStatusNativePlan(t *testing.T) {
	for _, pointer := range []bool{false, true} {
		name := "value"
		if pointer {
			name = "pointer"
		}
		t.Run(name, func(t *testing.T) {
			field := &expr.AttributeExpr{Type: expr.Int, Meta: expr.MetaExpr{"struct:field:type": {"int64"}}}
			object := &expr.AttributeExpr{Type: &expr.Object{{Name: "status", Attribute: field}}}
			if !pointer {
				object.Validation = &expr.ValidationExpr{Required: []string{"status"}}
			}
			carrier := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Opening", UID: "opening", AttributeExpr: object}}
			response := &ResponseData{Selector: expr.HTTPStatusSelector{Class: expr.Status2XX}, Domain: []int{200, 201}, Carrier: carrier, statusBinding: &transportir.StatusBinding{Name: "status", Attribute: field}}
			plan := buildOutgoingStatusPlan(response, carrier, serviceContext("svc", codegen.NewNameScope()), "meta", "statusCode")
			require.True(t, plan.ErrorAware)
			require.Contains(t, plan.Code, "meta == nil")
			require.Contains(t, plan.Code, "[]int{200, 201}")
			require.Contains(t, plan.Code, "statusCode = ownedStatus")
			require.NotContains(t, plan.Code, "int(meta.Status)")
			require.Contains(t, plan.Code, "loomhttp.StatusViolationError")
			if pointer {
				require.Contains(t, plan.Code, "meta.Status == nil")
			} else {
				require.Contains(t, plan.Code, "uint64(meta.Status)")
			}
		})
	}
}

func TestOutgoingStatusLiteralPlan(t *testing.T) {
	response := &ResponseData{Selector: expr.HTTPStatusSelector{Code: 201}, Domain: []int{201}}
	plan := buildOutgoingStatusPlan(response, nil, nil, "meta", "statusCode")
	require.False(t, plan.ErrorAware)
	require.Equal(t, "statusCode = 201\n", plan.Code)
}

func TestOutgoingStatusNativeConstraints(t *testing.T) {
	field := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{Values: []any{201}}}
	object := &expr.AttributeExpr{Type: &expr.Object{{Name: "status", Attribute: field}}, Validation: &expr.ValidationExpr{Required: []string{"status"}}}
	carrier := &expr.AttributeExpr{Type: &expr.UserTypeExpr{TypeName: "Opening", UID: "opening", AttributeExpr: object}}
	response := &ResponseData{Selector: expr.HTTPStatusSelector{Class: expr.Status2XX}, Domain: []int{200, 201}, Carrier: carrier, statusBinding: &transportir.StatusBinding{Name: "status", Attribute: field}}
	plan := buildOutgoingStatusPlan(response, carrier, serviceContext("svc", codegen.NewNameScope()), "meta", "statusCode")
	require.Contains(t, plan.Code, "InvalidEnumValueError", "owned status 200 must still fail the authored enum")
	require.Contains(t, plan.Code, "meta.Status == 201")
	require.Contains(t, plan.Code, "if *transformErr == nil")
}

func TestOutgoingStatusLiteralRequiresCapturedDomain(t *testing.T) {
	for _, status := range []int{100, 201} {
		response := &ResponseData{Selector: expr.HTTPStatusSelector{Code: status}}
		plan := buildOutgoingStatusPlan(response, nil, nil, "meta", "statusCode")
		require.True(t, plan.ErrorAware)
		require.Contains(t, plan.Code, "if *transformErr == nil")
		require.Contains(t, plan.Code, "loomhttp.StatusViolationError")
		require.NotContains(t, plan.Code, "statusCode =")
	}
}
