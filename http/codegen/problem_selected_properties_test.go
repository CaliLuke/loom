package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

func TestProblemSelectedPropertyConstruction(t *testing.T) {
	endpoint := firstEndpointData(t, func() {
		dsl.Service("properties", func() {
			dsl.Method("selected", func() {
				dsl.Error("oops")
				dsl.HTTP(func() {
					dsl.GET("/selected")
					dsl.Response("oops", dsl.StatusBadRequest, func() {
						dsl.Body("detail")
					})
				})
			})
		})
	})
	code := endpoint.Errors[0].Errors[0].Response.ResultInit.ClientCode
	for _, assignment := range []string{"problem.Timeout = timeout", "problem.Temporary = temporary", "problem.Fault = fault"} {
		require.Contains(t, code, assignment)
	}
	require.Contains(t, code, "loomhttp.ProblemErrorFromBody(*problem)")
}

func TestProblemFieldPlanPreservesError(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		dsl.Service("problems", func() {
			dsl.Method("show", func() {
				dsl.Error("bad")
				dsl.HTTP(func() {
					dsl.GET("/")
					dsl.Response("bad", dsl.StatusBadRequest)
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	data := services.Get("problems")
	endpoint := transportir.BuildEndpoint(root.API.HTTP.Services[0].HTTPEndpoints[0])
	response := *endpoint.Response.ErrorResponses[0]
	response.Body = &expr.AttributeExpr{Type: expr.Empty}
	source := &expr.AttributeExpr{Type: expr.UInt64}
	arg := &InitArgData{attribute: source, AttributeData: &AttributeData{Name: "status", VarName: "status", Type: expr.UInt64, FieldType: expr.Int, FieldName: "Status"}}
	plan := newErrorBuilder(services, endpoint, data).buildProblemClientResult(&response, nil, []*InitArgData{arg})
	require.True(t, plan.ErrorAware)
	require.Contains(t, plan.Code, "if *transformErr == nil {\nv = loomhttp.ProblemErrorFromBody")
	require.Nil(t, arg.fieldAttribute, "problem target capture must not mutate another constructor's argument")
}
