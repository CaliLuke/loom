package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// ClientFiles returns the generated HTTP client files.
func ClientFiles(genpkg string, data *ServicesData) []*codegen.File {
	files := make([]*codegen.File, 0, len(data.Expressions.Services)*3) // preallocate for client files
	for _, svc := range data.Expressions.Services {
		files = append(files, clientFile(genpkg, svc, data))
		if f := websocketClientFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
		if f := sseClientFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	for _, svc := range data.Expressions.Services {
		if f := ClientEncodeDecodeFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// ClientEncodeDecodeFile returns the file containing the HTTP client encoding
// and decoding logic.
func ClientEncodeDecodeFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "http", svcName, "client", "encode_decode.go")
	title := fmt.Sprintf("%s HTTP client encoders and decoders", svc.Name())
	sections := make([]codegen.Section, 0, 1+len(data.Endpoints)*4+len(data.ClientTransformHelpers))
	sections = append(sections, codegen.Header(title, "client", clientEncodeDecodeImports(genpkg, svcName, data)))
	for _, e := range data.Endpoints {
		sections = append(sections, clientEncodeDecodeSections(svc, services, e)...)
	}
	for _, h := range data.ClientTransformHelpers {
		sections = append(sections, transformHelperSection("client-transform-helper", h))
	}

	return &codegen.File{Path: path, Sections: sections}
}

func clientEncodeDecodeImports(genpkg, svcName string, data *ServiceData) []*codegen.ImportSpec {
	return []*codegen.ImportSpec{
		{Path: "bytes"},
		{Path: "context"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "fmt"},
		{Path: "io"},
		{Path: "mime/multipart"},
		{Path: "net/http"},
		{Path: "net/url"},
		{Path: "os"},
		{Path: "strconv"},
		{Path: "strings"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	}
}

func clientEncodeDecodeSections(svc *expr.HTTPServiceExpr, services *ServicesData, e *EndpointData) []codegen.Section {
	sections := []codegen.Section{requestBuilderSection(e)}
	if section := clientRequestEncoderSection(svc, services, e); section != nil {
		sections = append(sections, section)
	}
	if e.MultipartRequestEncoder != nil {
		sections = append(sections, multipartRequestEncoderSection(e.MultipartRequestEncoder))
	}
	if section := clientResponseDecoderSection(svc, services, e); section != nil {
		sections = append(sections, section)
	}
	if e.Method.SkipRequestBodyEncodeDecode {
		sections = append(sections, buildStreamRequestSection(e))
	}
	return sections
}

func clientRequestEncoderSection(svc *expr.HTTPServiceExpr, services *ServicesData, e *EndpointData) codegen.Section {
	if e.RequestEncoder == "" || e.Payload.Ref == "" {
		return nil
	}
	return codegen.NewTextTemplateSection("request-encoder", requestEncoderSource, clientRequestTemplateFuncs(svc, services), e)
}

func clientRequestTemplateFuncs(svc *expr.HTTPServiceExpr, services *ServicesData) map[string]any {
	return map[string]any{
		"typeConversionData": typeConversionData,
		"mapConversionData":  mapConversionData,
		"goTypeRef": func(dt expr.DataType) string {
			return services.ServicesData.Get(svc.Name()).Scope.GoTypeRef(&expr.AttributeExpr{Type: dt})
		},
		"isBearer":    isBearer,
		"aliasedType": fieldType,
		"isAlias": func(dt expr.DataType) bool {
			_, ok := dt.(expr.UserType)
			return ok
		},
		"underlyingType": func(dt expr.DataType) expr.DataType {
			if ut, ok := dt.(expr.UserType); ok {
				return ut.Attribute().Type
			}
			return dt
		},
		"requestStructPkg": requestStructPkg,
	}
}

func clientResponseDecoderSection(svc *expr.HTTPServiceExpr, services *ServicesData, e *EndpointData) codegen.Section {
	if e.Result == nil && len(e.Errors) == 0 {
		return nil
	}
	fm := transDecoderTmplFuncs(svc, services)
	fm["buildResponseData"] = buildResponseData
	return codegen.NewTextTemplateSection(
		"response-decoder",
		responseDecoderSource,
		fm,
		e,
	)
}

// clientFile returns the client HTTP transport file
func clientFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "http", svcName, "client", "client.go")
	title := fmt.Sprintf("%s client HTTP transport", svc.Name())
	sections := []codegen.Section{
		codegen.Header(title, "client", []*codegen.ImportSpec{
			{Path: "context"},
			{Path: "fmt"},
			{Path: "io"},
			{Path: "mime/multipart"},
			{Path: "net/http"},
			{Path: "strconv"},
			{Path: "strings"},
			{Path: "time"},
			{Path: "github.com/gorilla/websocket"},
			codegen.LoomImport(""),
			codegen.LoomNamedImport("http", "loomhttp"),
			{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
			{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
		}),
	}
	sections = append(sections, clientStructSection(data))
	if len(clientOperationGroups(data)) > 0 {
		sections = append(sections, clientOperationGroupSection(data))
	}

	for _, e := range data.Endpoints {
		if e.MultipartRequestEncoder != nil {
			sections = append(sections, multipartRequestEncoderTypeSection(e.MultipartRequestEncoder))
		}
	}
	sections = append(sections, clientInitSection(data))
	for _, e := range data.Endpoints {
		sections = append(sections, clientEndpointSections(e)...)
	}

	return &codegen.File{Path: path, Sections: sections}
}

// typeConversionData produces the template data suitable for executing the
// "header_conversion" template.
func typeConversionData(dt, ft expr.DataType, varName, target string) map[string]any {
	ut, isut := ft.(expr.UserType)
	if isut {
		ft = ut.Attribute().Type
	}
	return map[string]any{
		"Type":      dt,
		"FieldType": ft,
		"VarName":   varName,
		"Target":    target,
		"IsAliased": isut,
	}
}

func mapConversionData(dt, ft expr.DataType, varName, sourceVar, sourceField string, newVar bool) map[string]any {
	ut, isut := ft.(expr.UserType)
	if isut {
		ft = ut.Attribute().Type
	}
	return map[string]any{
		"Type":        dt,
		"FieldType":   ft,
		"VarName":     varName,
		"SourceVar":   sourceVar,
		"SourceField": sourceField,
		"NewVar":      newVar,
		"IsAliased":   isut,
	}
}

// buildResponseData produces the template data suitable for executing the
// "single_response" partial template.
func buildResponseData(data *ResponseData, serviceName string, method *service.MethodData) map[string]any {
	return map[string]any{
		"Data":        data,
		"ServiceName": serviceName,
		"Method":      method,
	}
}

func fieldType(ft expr.DataType) expr.DataType {
	ut, isut := ft.(expr.UserType)
	if isut {
		return ut.Attribute().Type
	}
	return ft
}

// isBearer returns true if the security scheme uses a Bearer scheme.
func isBearer(schemes []*service.SchemeData) bool {
	for _, s := range schemes {
		if s.Name != "Authorization" {
			continue
		}
		if s.Type == "JWT" || s.Type == "OAuth2" {
			return true
		}
	}
	return false
}

func requestStructPkg(m *service.MethodData, def string) string {
	if m.PayloadLoc != nil {
		return m.PayloadLoc.PackageName()
	}
	return def
}

func responseStructPkg(m *service.MethodData, def string) string {
	if m.ResultLoc != nil {
		return m.ResultLoc.PackageName()
	}
	return def
}
