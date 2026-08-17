package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
)

func TestTypeInitSectionStructuredDeclaration(t *testing.T) {
	section := typeInitSection("viewed-result-type-to-service-result-type", &InitData{
		Name:          "NewWidget",
		Description:   "NewWidget builds a widget.",
		ReturnTypeRef: "*Widget",
		Args: []*InitArgData{
			{Name: "name", Ref: "string"},
		},
		Code: `
res := &Widget{Name: name}
return res`,
	})

	code := codegen.SectionCode(t, section)
	require.Contains(t, code, "func NewWidget(name string) *Widget {")
	testutil.AssertGo(t, "testdata/golden/sections_type_init.go.golden", code)
}

func TestViewedResultInitSectionsReturnErrors(t *testing.T) {
	resultSection := typeInitSection("viewed-result-type-to-service-result-type", &InitData{
		Name:          "NewWidget",
		Description:   "NewWidget builds a widget.",
		ReturnTypeRef: "*Widget",
		ReturnsError:  true,
		Args: []*InitArgData{
			{Name: "vres", Ref: "*widgetviews.Widget"},
		},
		Code: `
var res *Widget
switch vres.View {
case "default", "":
	res = NewWidgetFromWidgetView(vres.Projected)
default:
	return res, loom.InvalidEnumValueError("view", vres.View, []any{"default"})
}
return res, nil`,
	})

	resultCode := codegen.SectionCode(t, resultSection)
	require.Contains(t, resultCode, "func NewWidget(vres *widgetviews.Widget) (*Widget, error) {")
	require.NotContains(t, resultCode, "panic(")
	testutil.AssertGo(t, "testdata/golden/sections_viewed_result_init_result.go.golden", resultCode)

	viewedSection := typeInitSection("service-result-type-to-viewed-result-type", &InitData{
		Name:          "NewViewedWidget",
		Description:   "NewViewedWidget builds a viewed widget.",
		ReturnTypeRef: "*widgetviews.Widget",
		ReturnsError:  true,
		Args: []*InitArgData{
			{Name: "res", Ref: "*Widget"},
			{Name: "view", Ref: "string"},
		},
		Code: `
var vres *widgetviews.Widget
switch view {
case "default", "":
	vres = &widgetviews.Widget{Projected: ProjectWidget(res), View: "default"}
default:
	return vres, loom.InvalidEnumValueError("view", view, []any{"default"})
}
return vres, nil`,
	})

	viewedCode := codegen.SectionCode(t, viewedSection)
	require.Contains(t, viewedCode, "func NewViewedWidget(res *Widget, view string) (*widgetviews.Widget, error) {")
	require.NotContains(t, viewedCode, "panic(")
	testutil.AssertGo(t, "testdata/golden/sections_viewed_result_init_viewed.go.golden", viewedCode)
}

func TestConvertSectionsStructuredDeclarations(t *testing.T) {
	convertCode := codegen.SectionCode(t, convertSection("convert-to", convertData{
		Name:            "ToExternal",
		ReceiverTypeRef: "*Widget",
		TypeRef:         "*external.Widget",
		TypeName:        "external.Widget",
		Code: `
v := &external.Widget{Name: t.Name}`,
	}))
	require.Contains(t, convertCode, "func (t *Widget) ToExternal() *external.Widget {")
	testutil.AssertGo(t, "testdata/golden/sections_convert_to.go.golden", convertCode)

	createCode := codegen.SectionCode(t, createSection("create-from", convertData{
		Name:            "FromExternal",
		ReceiverTypeRef: "*Widget",
		TypeRef:         "*external.Widget",
		TypeName:        "external.Widget",
		Code: `
temp := &Widget{Name: v.Name}`,
	}))
	require.Contains(t, createCode, "func (t *Widget) FromExternal(v *external.Widget) {")
	testutil.AssertGo(t, "testdata/golden/sections_create_from.go.golden", createCode)

	helperCode := codegen.SectionCode(t, transformHelperSection("convert-create-helper", &codegen.TransformFunctionData{
		Name:          "widgetToAlias",
		ParamTypeRef:  "*Widget",
		ResultTypeRef: "*Alias",
		Code: `
res := &Alias{Name: v.Name}`,
	}))
	require.Contains(t, helperCode, "func widgetToAlias(v *Widget) *Alias {")
	testutil.AssertGo(t, "testdata/golden/sections_transform_helper.go.golden", helperCode)
}

func TestExampleSectionsStructuredDeclarations(t *testing.T) {
	authCode := codegen.SectionCode(t, exampleSecurityAuthSection(&Data{
		Name:    "Widget",
		VarName: "widget",
		Schemes: SchemesData{
			{Type: "Basic", SchemeName: "basic"},
		},
	}))
	require.Contains(t, authCode, "func (s *widgetsrvc) BasicAuth(ctx context.Context, user, pass string, scheme *security.BasicScheme) (context.Context, error) {")
	testutil.AssertGo(t, "testdata/golden/sections_example_security_auth.go.golden", authCode)

	multiAPIKeyCode := codegen.SectionCode(t, exampleSecurityAuthSection(&Data{
		Name:    "Widget",
		VarName: "widget",
		Schemes: SchemesData{
			{Type: "APIKey", SchemeName: "first"},
			{Type: "APIKey", SchemeName: "second"},
		},
	}))
	require.Equal(t, 1, strings.Count(multiAPIKeyCode, "func (s *widgetsrvc) APIKeyAuth("))
	require.Contains(t, multiAPIKeyCode, "APIKey security scheme type.")

	endpointCode := codegen.SectionCode(t, exampleEndpointSection(&basicEndpointData{
		MethodData: &MethodData{
			Name:        "Do",
			VarName:     "Do",
			Description: "Do runs the widget method.",
			MethodResultData: MethodResultData{
				Result: "Widget",
			},
			MethodTransportData: MethodTransportData{
				SkipResponseBodyEncodeDecode: true,
			},
		},
		ServiceVarName: "widget",
		ResultFullName: "Widget",
		ResultFullRef:  "*Widget",
		ResultIsStruct: true,
	}))
	require.Contains(t, endpointCode, "func (s *widgetsrvc) Do(ctx context.Context) (res *Widget, resp io.ReadCloser, err error) {")
	testutil.AssertGo(t, "testdata/golden/sections_example_endpoint.go.golden", endpointCode)

	streamCode := codegen.SectionCode(t, jsonrpcHandleStreamSection(&Data{
		VarName: "widget",
		PkgName: "widgetsvc",
	}))
	require.Contains(t, streamCode, "func (s *widgetsrvc) HandleStream(ctx context.Context, stream widgetsvc.Stream) error {")
	testutil.AssertGo(t, "testdata/golden/sections_jsonrpc_handle_stream.go.golden", streamCode)
}

func TestExampleInterceptorSectionStructuredDeclaration(t *testing.T) {
	code := codegen.SectionCode(t, exampleInterceptorSection("example-server-interceptor", map[string]any{
		"ServiceName": "Widget",
		"StructName":  "Widget",
		"PkgName":     "widgetsvc",
		"ServerInterceptors": []*InterceptorData{
			{Name: "Trace", Description: "Trace logs request flow."},
		},
		"ClientInterceptors": []*InterceptorData{},
	}, true))

	require.Contains(t, code, "type WidgetServerInterceptors struct{}")
	require.Contains(t, code, "func (i *WidgetServerInterceptors) Trace(ctx context.Context, info *widgetsvc.TraceInfo, next loom.Endpoint) (any, error) {")
	testutil.AssertGo(t, "testdata/golden/sections_example_interceptor.go.golden", code)
}

func TestUnionSectionStructuredDeclarations(t *testing.T) {
	code := codegen.SectionCode(t, unionTypeSection("service-union-type", &UnionTypeData{
		Name:     "Selection",
		KindName: "SelectionKind",
		TypeKey:  "type",
		ValueKey: "value",
		Fields: []*UnionFieldData{
			{
				Name:               "text",
				KindConst:          "SelectionKindText",
				FieldName:          "Text",
				FieldType:          "SelectionText",
				EmitPrimitiveAlias: true,
				PrimitiveAliasType: "string",
				TypeTag:            "text",
			},
			{
				Name:               "count",
				KindConst:          "SelectionKindCount",
				FieldName:          "Count",
				FieldType:          "SelectionCount",
				EmitPrimitiveAlias: true,
				PrimitiveAliasType: "int",
				TypeTag:            "count",
			},
		},
	}))

	require.Contains(t, code, "type SelectionText string")
	require.Contains(t, code, "type SelectionCount int")
	testutil.AssertGo(t, "testdata/golden/sections_union_type.go.golden", code)
}
