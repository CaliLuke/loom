package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type serverTypeSections struct {
	sections       []codegen.Section
	initData       []*InitData
	validatedTypes []*TypeData
	seenValidated  map[string]struct{}
	seenInits      map[string]struct{}
}

// ServerTypeFiles returns the HTTP transport type files.
func ServerTypeFiles(genpkg string, data *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, len(data.Expressions.Services))
	for i, r := range data.Expressions.Services {
		fw[i] = serverType(genpkg, r, data)
	}
	return fw
}

// serverType return the file containing the type definitions used by the HTTP
// transport for the given service server.
//
// Below are the rules governing whether values are pointers or not. Note that
// the rules only applies to values that hold primitive types, values that hold
// slices, maps or objects always use pointers either implicitly - slices and
// maps - or explicitly - objects.
//
//   - The payload struct fields (if a struct) hold pointers when not required
//     and have no default value.
//
//   - Request body fields (if the body is a struct) always hold pointers to
//     allow for explicit validation.
//
//   - Request header, path and query string parameter variables hold pointers
//     when not required. Request header, body fields and param variables that
//     have default values are never required (enforced by DSL engine).
//
//   - The result struct fields (if a struct) hold pointers when not required
//     or have a default value (so generated code can set when null)
//
//   - Response body fields (if the body is a struct) and header variables hold
//     pointers when not required and have no default value.
func serverType(genpkg string, svc *expr.HTTPServiceExpr, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "http", svcName, "server", "types.go")
	sections := newServerTypeSections(codegen.Header(svc.Name()+" HTTP server types", "server", serverTypeImports(genpkg, svcName, data)))

	for _, a := range svc.HTTPEndpoints {
		sections.appendEndpointTypes(data.Endpoint(a.Name()), data)
	}

	for _, tdata := range data.ServerBodyAttributeTypes {
		sections.appendTypeData("server-body-attributes", tdata, false, data)
	}

	for _, u := range data.UnionTypes {
		sections.sections = append(sections.sections, unionTypeSection("server-union-type", u))
	}

	sections.appendBodyInitSections()

	for _, adata := range data.Endpoints {
		sections.appendPayloadInitSections(adata)
	}

	sections.appendValidateSections()

	return &codegen.File{Path: path, Sections: sections.sections}
}

func serverTypeImports(genpkg, svcName string, data *ServiceData) []*codegen.ImportSpec {
	return []*codegen.ImportSpec{
		{Path: "encoding/json"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "unicode/utf8"},
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "goahttp"),
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	}
}

func newServerTypeSections(header codegen.Section) *serverTypeSections {
	return &serverTypeSections{
		sections:      []codegen.Section{header},
		seenValidated: make(map[string]struct{}),
		seenInits:     make(map[string]struct{}),
	}
}

func (s *serverTypeSections) appendEndpointTypes(adata *EndpointData, data *ServiceData) {
	s.appendTypeData("request-body-type-decl", adata.Payload.Request.ServerBody, false, data)
	if adata.ServerWebSocket != nil && !adata.Method.IsJSONRPC {
		s.appendTypeData("request-stream-payload-type-decl", adata.ServerWebSocket.Payload, false, data)
	}
	for _, resp := range adata.Result.Responses {
		for _, tdata := range resp.ServerBody {
			s.appendTypeData("response-server-body", tdata, true, data)
		}
	}
	for _, gerr := range adata.Errors {
		for _, herr := range gerr.Errors {
			for _, tdata := range herr.Response.ServerBody {
				s.appendTypeData("error-body-type-decl", tdata, true, data)
			}
		}
	}
}

func (s *serverTypeSections) appendTypeData(section string, tdata *TypeData, trackInit bool, data *ServiceData) {
	if tdata == nil {
		return
	}
	if generated, ok := data.ServerTypeNames[tdata.Name]; ok && generated {
		return
	}
	if tdata.Def != "" {
		s.sections = append(s.sections, typeDeclSection(section, tdata))
		data.ServerTypeNames[tdata.Name] = true
	}
	if trackInit {
		s.appendBodyInitData(tdata.Init)
	}
	if tdata.ValidateDef != "" {
		recordValidatedType(tdata, s.seenValidated, &s.validatedTypes)
	}
}

func (s *serverTypeSections) appendBodyInitData(init *InitData) {
	if init == nil {
		return
	}
	s.initData = append(s.initData, init)
}

func (s *serverTypeSections) appendBodyInitSections() {
	for _, init := range s.initData {
		s.sections = append(s.sections, bodyInitSection("server-body-init", init, false))
	}
}

func (s *serverTypeSections) appendPayloadInitSections(adata *EndpointData) {
	s.appendPayloadInitSection(adata.Payload.Request.PayloadInit)
	if IsWebSocketEndpoint(adata) && adata.ServerWebSocket.Payload != nil {
		s.appendPayloadInitSection(adata.ServerWebSocket.Payload.Init)
	}
}

func (s *serverTypeSections) appendPayloadInitSection(init *InitData) {
	if init == nil {
		return
	}
	if _, ok := s.seenInits[init.Name]; ok {
		return
	}
	s.seenInits[init.Name] = struct{}{}
	s.sections = append(s.sections, typeInitSection("server-payload-init", init, false))
}

func (s *serverTypeSections) appendValidateSections() {
	for _, data := range s.validatedTypes {
		s.sections = append(s.sections, validateSection("server-validate", data))
	}
}

// fieldCode returns the code to initialize the return struct fields. It is
// used only in templates.
func fieldCode(init *InitData, typ string) string {
	varn := "res"
	if init.ReturnTypeAttribute == "" {
		varn = "v"
	}
	args := init.ServerArgs
	if typ == "client" {
		args = init.ClientArgs
	}
	initArgs := make([]*codegen.InitArgData, len(args))
	for i, arg := range args {
		initArgs[i] = &codegen.InitArgData{
			Name:         arg.VarName,
			Pointer:      arg.Pointer,
			Type:         arg.Type,
			FieldName:    arg.FieldName,
			FieldPointer: arg.FieldPointer,
			FieldType:    arg.FieldType,
		}
	}
	// We can ignore the transform helpers as there won't be any generated
	// because the headers and params cannot be user types.
	c, _, err := codegen.InitStructFields(initArgs, varn, "", init.ReturnTypePkg)
	if err != nil {
		panic(err) // bug
	}
	return c
}
