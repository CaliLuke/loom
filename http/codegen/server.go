package codegen

import (
	"fmt"
	"path"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// ServerFiles returns the generated HTTP server files.
func ServerFiles(genpkg string, data *ServicesData) []*codegen.File {
	files := make([]*codegen.File, 0, len(data.Expressions.Services)*3)
	for _, svc := range data.Expressions.Services {
		files = append(files, serverFile(genpkg, svc, data))
		if f := websocketServerFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
		if f := sseServerFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	for _, svc := range data.Expressions.Services {
		if f := ServerEncodeDecodeFile(genpkg, svc, data); f != nil {
			files = append(files, f)
		}
	}
	return files
}

// serverFile returns the file implementing the HTTP server.
func serverFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "http", svcName, "server", "server.go")
	title := fmt.Sprintf("%s HTTP server", svc.Name())
	sections := []codegen.Section{
		codegen.Header(title, "server", serverImports(genpkg, svcName, data)),
	}
	sections = append(sections, serverBaseSections(data)...)
	for _, e := range data.Endpoints {
		sections = append(sections, serverEndpointSections(e)...)
	}
	if section := mappedFilesSection(data); section != nil {
		sections = append(sections, section)
	}
	for _, s := range data.FileServers {
		sections = append(sections, fileServerSection(s))
	}

	return &codegen.File{Path: fpath, Sections: sections}
}

// ServerEncodeDecodeFile returns the file defining the HTTP server encoding and
// decoding logic.
func ServerEncodeDecodeFile(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "http", svcName, "server", "encode_decode.go")
	title := fmt.Sprintf("%s HTTP server encoders and decoders", svc.Name())
	sections := make([]codegen.Section, 0, 1+len(data.Endpoints)*4+len(data.ServerTransformHelpers))
	sections = append(sections, codegen.Header(title, "server", serverEncodeDecodeImports(genpkg, svcName, data)))
	for _, e := range data.Endpoints {
		sections = append(sections, serverEncodeDecodeSections(svc, services, e)...)
	}
	for _, h := range data.ServerTransformHelpers {
		sections = append(sections, transformHelperSection("server-transform-helper", h))
	}

	// If all endpoints use skip encoding and decoding of both payloads and
	// results and define no error then this file is irrelevant.
	if len(sections) == 1 {
		return nil
	}

	return &codegen.File{Path: path, Sections: sections}
}

func serverImports(genpkg, svcName string, data *ServiceData) []*codegen.ImportSpec {
	imports := []*codegen.ImportSpec{
		{Path: "bufio"},
		{Path: "context"},
		{Path: "fmt"},
		{Path: "io"},
		{Path: "net/http"},
		{Path: "path"},
		{Path: "strings"},
		{Path: "github.com/gorilla/websocket"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	}
	if hasCustomMultipartDecoder(data) {
		imports = append(imports, &codegen.ImportSpec{Path: "mime/multipart"})
	}
	return imports
}

func serverBaseSections(data *ServiceData) []codegen.Section {
	sections := []codegen.Section{
		serverStructSection(data),
		mountPointStructSection(data),
	}
	for _, e := range data.Endpoints {
		if e.MultipartRequestDecoder != nil {
			sections = append(sections, multipartRequestDecoderTypeSection(e.MultipartRequestDecoder))
		}
	}
	return append(sections,
		serverInitSection(data),
		serverServiceSection(data),
		serverUseSection(data),
		serverMethodNamesSection(data),
		serverMountSection(data),
	)
}

func serverEndpointSections(e *EndpointData) []codegen.Section {
	return []codegen.Section{
		serverHandlerSection(e),
		&codegen.SectionTemplate{Name: "server-handler-init", Source: serverHandlerInitSource, FuncMap: serverTemplateFuncs(), Data: e},
	}
}

func serverTemplateFuncs() map[string]any {
	return map[string]any{
		"join":                strings.Join,
		"hasWebSocket":        HasWebSocket,
		"isWebSocketEndpoint": IsWebSocketEndpoint,
		"isSSEEndpoint":       IsSSEEndpoint,
		"viewedServerBody":    viewedServerBody,
		"mustDecodeRequest":   mustDecodeRequest,
		"addLeadingSlash":     addLeadingSlash,
		"dir":                 path.Dir,
		"isObject":            expr.IsObject,
	}
}

func mappedFilesSection(data *ServiceData) codegen.Section {
	if len(data.FileServers) == 0 {
		return nil
	}
	mappedFiles := map[string]string{}
	for _, fs := range data.FileServers {
		if fs.IsDir {
			continue
		}
		for _, p := range fs.RequestPaths {
			baseFilePath := "/" + filepath.Base(fs.FilePath)
			baseRequestPath := "/" + filepath.Base(p)
			if baseFilePath != baseRequestPath {
				mappedFiles[baseRequestPath] = baseFilePath
			}
		}
	}
	return appendFSSection(mappedFiles)
}

func serverEncodeDecodeImports(genpkg, svcName string, data *ServiceData) []*codegen.ImportSpec {
	imports := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "errors"},
		{Path: "fmt"},
		{Path: "io"},
		{Path: "net/http"},
		{Path: "strconv"},
		{Path: "strings"},
		{Path: "encoding/json"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	}
	if hasCustomMultipartDecoder(data) {
		imports = append(imports, &codegen.ImportSpec{Path: "mime/multipart"})
	}
	return imports
}

func serverEncodeDecodeSections(svc *expr.HTTPServiceExpr, services *ServicesData, e *EndpointData) []codegen.Section {
	sections := []codegen.Section{}
	if e.Redirect == nil && (!IsWebSocketEndpoint(e) || e.Method.IsJSONRPC) {
		sections = append(sections, &codegen.SectionTemplate{
			Name:    "response-encoder",
			FuncMap: transTmplFuncs(svc, services),
			Source:  responseEncoderSource,
			Data:    e,
		})
	}
	if mustDecodeRequest(e) {
		fm := transDecoderTmplFuncs(svc, services)
		sections = append(sections, &codegen.SectionTemplate{
			Name:    "request-decoder",
			Source:  requestDecoderSource,
			FuncMap: fm,
			Data:    e,
		})
	}
	if e.MultipartRequestDecoder != nil {
		fm := transDecoderTmplFuncs(svc, services)
		sections = append(sections, &codegen.SectionTemplate{
			Name:    "multipart-request-decoder",
			Source:  multipartRequestDecoderSource,
			FuncMap: fm,
			Data:    e.MultipartRequestDecoder,
		})
	}
	if len(e.Errors) > 0 {
		sections = append(sections, &codegen.SectionTemplate{
			Name:    "error-encoder",
			Source:  errorEncoderSource,
			FuncMap: transTmplFuncs(svc, services),
			Data:    e,
		})
	}
	return sections
}

func transDecoderTmplFuncs(s *expr.HTTPServiceExpr, services *ServicesData) map[string]any {
	fm := transTmplFuncs(s, services)
	fm["mapQueryDecodeData"] = mapQueryDecodeData
	return fm
}

func transTmplFuncs(s *expr.HTTPServiceExpr, services *ServicesData) map[string]any {
	return map[string]any{
		"goTypeRef": func(dt expr.DataType) string {
			return services.ServicesData.Get(s.Name()).Scope.GoTypeRef(&expr.AttributeExpr{Type: dt})
		},
		"isAliased": func(dt expr.DataType) bool {
			_, ok := dt.(expr.UserType)
			return ok
		},
		"conversionData":       conversionData,
		"headerConversionData": headerConversionData,
		"printValue":           printValue,
		"viewedServerBody":     viewedServerBody,
	}
}

func hasCustomMultipartDecoder(data *ServiceData) bool {
	for _, e := range data.Endpoints {
		if e.MultipartRequestDecoder != nil {
			return true
		}
	}
	return false
}

// mustDecodeRequest returns true if the Payload type is not empty.
func mustDecodeRequest(e *EndpointData) bool {
	return e.Payload.Ref != ""
}

// conversionData creates a template context suitable for executing the
// "type_conversion" template.
func conversionData(varName, name string, dt expr.DataType) map[string]any {
	return map[string]any{
		"VarName": varName,
		"Name":    name,
		"Type":    dt,
	}
}

// headerConversionData produces the template data suitable for executing the
// "header_conversion" template.
func headerConversionData(dt expr.DataType, varName string, required bool, target string) map[string]any {
	return map[string]any{
		"Type":     dt,
		"VarName":  varName,
		"Required": required,
		"Target":   target,
	}
}

// printValue generates the Go code for a literal string containing the given
// value. printValue panics if the data type is not a primitive or an array.
func printValue(dt expr.DataType, v any) string {
	switch actual := dt.(type) {
	case *expr.Array:
		val := reflect.ValueOf(v)
		elems := make([]string, val.Len())
		for i := 0; i < val.Len(); i++ {
			elems[i] = printValue(actual.ElemType.Type, val.Index(i).Interface())
		}
		return strings.Join(elems, ", ")
	case expr.Primitive:
		return fmt.Sprintf("%v", v)
	default:
		panic("unsupported type value " + dt.Name()) // bug
	}
}

// viewedServerBody returns the type data that uses the given view for
// rendering.
func viewedServerBody(sbd []*TypeData, view string) *TypeData {
	for _, v := range sbd {
		if v.View == view {
			return v
		}
	}
	panic("view not found in server body types: " + view)
}

func addLeadingSlash(s string) string {
	if s == "" || s[0] != '/' {
		return "/" + s
	}
	return s
}

func mapQueryDecodeData(dt expr.DataType, varName string, inc int) map[string]any {
	return map[string]any{
		"Type":      dt,
		"VarName":   varName,
		"Loop":      string(rune(97 + inc)),
		"Increment": inc + 1,
		"Depth":     codegen.MapDepth(expr.AsMap(dt)),
	}
}
