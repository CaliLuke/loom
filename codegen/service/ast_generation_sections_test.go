package service

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
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
	require.Contains(t, code, "return res")
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
	require.Contains(t, convertCode, "return v")

	createCode := codegen.SectionCode(t, createSection("create-from", convertData{
		Name:            "FromExternal",
		ReceiverTypeRef: "*Widget",
		TypeRef:         "*external.Widget",
		TypeName:        "external.Widget",
		Code: `
temp := &Widget{Name: v.Name}`,
	}))
	require.Contains(t, createCode, "func (t *Widget) FromExternal(v *external.Widget) {")
	require.Contains(t, createCode, "*t = *temp")

	helperCode := codegen.SectionCode(t, transformHelperSection("convert-create-helper", &codegen.TransformFunctionData{
		Name:          "widgetToAlias",
		ParamTypeRef:  "*Widget",
		ResultTypeRef: "*Alias",
		Code: `
res := &Alias{Name: v.Name}`,
	}))
	require.Contains(t, helperCode, "func widgetToAlias(v *Widget) *Alias {")
	require.Contains(t, helperCode, "return res")
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
	require.Contains(t, authCode, `return ctx, fmt.Errorf("not implemented")`)

	endpointCode := codegen.SectionCode(t, exampleEndpointSection(&basicEndpointData{
		MethodData: &MethodData{
			Name:                         "Do",
			VarName:                      "Do",
			Description:                  "Do runs the widget method.",
			Result:                       "Widget",
			SkipResponseBodyEncodeDecode: true,
		},
		ServiceVarName: "widget",
		ResultFullName: "Widget",
		ResultFullRef:  "*Widget",
		ResultIsStruct: true,
	}))
	require.Contains(t, endpointCode, "func (s *widgetsrvc) Do(ctx context.Context) (res *Widget, resp io.ReadCloser, err error) {")
	require.Contains(t, endpointCode, `resp = io.NopCloser(strings.NewReader("Do"))`)

	streamCode := codegen.SectionCode(t, jsonrpcHandleStreamSection(&Data{
		VarName: "widget",
		PkgName: "widgetsvc",
	}))
	require.Contains(t, streamCode, "func (s *widgetsrvc) HandleStream(ctx context.Context, stream widgetsvc.Stream) error {")
	require.Contains(t, streamCode, `log.Printf(ctx, "widget.HandleStream")`)
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
	require.Contains(t, code, "func NewWidgetServerInterceptors() *WidgetServerInterceptors {")
	require.Contains(t, code, "func (i *WidgetServerInterceptors) Trace(ctx context.Context, info *widgetsvc.TraceInfo, next loom.Endpoint) (any, error) {")
}
