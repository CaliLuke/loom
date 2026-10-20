package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestJSONBodyInputRole(t *testing.T) {
	child := &expr.UserTypeExpr{TypeName: "Child", UID: "Child", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "flag", Attribute: &expr.AttributeExpr{Type: expr.Boolean, DefaultValue: true}},
	}}}
	parent := &expr.UserTypeExpr{TypeName: "Parent", UID: "Parent", AttributeExpr: &expr.AttributeExpr{Type: &expr.Object{
		{Name: "child", Attribute: &expr.AttributeExpr{Type: child}},
	}}}
	attribute := &expr.AttributeExpr{Type: parent}
	sd := &ServiceData{Service: &service.Data{Name: "values"}, Scope: codegen.NewNameScope()}
	// A pre-existing encoder declaration owns the unsuffixed candidate name.
	collision := &expr.UserTypeExpr{TypeName: "CLIParent", UID: "collision", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	require.Equal(t, "CLIParent", sd.Scope.GoTypeName(&expr.AttributeExpr{Type: collision}))
	input := newJSONBodyInput(attribute, "read", sd, false, make(map[bodyTypeKey]string), make(map[string]*uniongen.Type))
	require.NotSame(t, attribute, input.attribute)
	require.Equal(t, "Parent", parent.Name())
	require.Equal(t, "Child", child.Name())
	require.Equal(t, "CLIParent2", input.root.VarName)
	require.True(t, input.context.JSONPresence)
	require.True(t, input.context.CollectionElementPresence)
	require.True(t, input.context.Pointer)
	require.False(t, input.context.UseDefault)
	require.Contains(t, input.root.Def, "loom.Optional[CLIChild]")
	require.Len(t, input.types, 1)
	require.Contains(t, input.types[0].Def, "loom.Optional[bool]")
	require.Empty(t, sd.ClientJSONPresenceTypes)
	require.Empty(t, sd.ServerJSONPresenceTypes)
	require.Equal(t, "CLIParent", sd.Scope.GoTypeName(&expr.AttributeExpr{Type: collision}))
}

func TestCLICollectionInputKeepsServerInitialization(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadBodyPrimitiveArrayStringValidateDSL)
	services := CreateHTTPServices(root)
	sd := services.Get(root.Services[0].Name)
	require.Len(t, sd.Endpoints, 1)
	request := sd.Endpoints[0].Payload.Request
	require.NotNil(t, request.cliInput)
	require.NotSame(t, request.PayloadInit, request.cliPayloadInit())
	require.Equal(t, "p", request.cliInput.argument.VarName)
	require.Contains(t, request.cliInput.argument.TypeName, "loom.Optional[string]")
	require.NotEmpty(t, request.cliInput.argument.Validate)
	require.Equal(t, request.cliInput.init, request.cliPayloadInit())
}
