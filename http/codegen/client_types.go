package codegen

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type clientTypeSections struct {
	sections       []codegen.Section
	initData       []*InitData
	validatedTypes []*TypeData
	seenTypes      map[string]struct{}
	seenValidated  map[string]struct{}
	seenInits      map[string]struct{}
}

// ClientTypeFiles returns the HTTP transport client types files.
func ClientTypeFiles(genpkg string, data *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, 0, len(data.Expressions.Services))
	for _, svc := range data.Expressions.Services {
		file := clientType(genpkg, svc, make(map[string]struct{}), data)
		svcData := data.Get(svc.Name())
		svcName := svcData.Service.PathName
		title := svc.Name() + " HTTP client types"
		imports := clientTypeImports(genpkg, svcName, svcData)
		fw = append(fw, splitTypeFileIfLarge(file, title, "client", imports)...)
	}
	return fw
}

// clientType return the file containing the type definitions used by the HTTP
// transport for the given service client. seen keeps track of the names of the
// types that have already been generated to prevent duplicate code generation.
//
// Below are the rules governing whether values are pointers or not. Note that
// the rules only applies to values that hold primitive types, values that hold
// slices, maps or objects always use pointers either implicitly - slices and
// maps - or explicitly - objects.
//
//   - The payload struct fields (if a struct) hold pointers when not required
//     and have no default value.
//
//   - Request and response body fields (if the body is a struct) always hold
//     pointers to allow for explicit validation.
//
//   - Request header, path and query string parameter variables hold pointers
//     when not required. Request header, body fields and param variables that
//     have default values are never required (enforced by DSL engine).
//
//   - The result struct fields (if a struct) hold pointers when not required
//     or have a default value (so generated code can set when null).
//
//   - Response header variables hold pointers when not required and have no
//     default value.
func clientType(genpkg string, svc *expr.HTTPServiceExpr, seen map[string]struct{}, services *ServicesData) *codegen.File {
	data := services.Get(svc.Name())
	svcName := data.Service.PathName
	path := filepath.Join(codegen.Gendir, "http", svcName, "client", "types.go")
	sections := newClientTypeSections(codegen.Header(svc.Name()+" HTTP client types", "client", clientTypeImports(genpkg, svcName, data)))

	for _, a := range svc.HTTPEndpoints {
		sections.appendEndpointTypes(data.Endpoint(a.Name()), seen)
	}

	sections.appendBodyAttributeTypes(data.ClientBodyAttributeTypes, seen)

	for _, u := range data.UnionTypes {
		sections.sections = append(sections.sections, unionTypeSection("client-union-type", u))
	}

	sections.appendBodyInitSections()

	for _, adata := range data.Endpoints {
		sections.appendResultInitSections(adata)
	}

	sections.appendValidateSections()
	return &codegen.File{Path: path, Sections: sections.sections}
}

func clientTypeImports(genpkg, svcName string, data *ServiceData) []*codegen.ImportSpec {
	return []*codegen.ImportSpec{
		{Path: "encoding/json/jsontext"},
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "unicode/utf8"},
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
	}
}

func newClientTypeSections(header codegen.Section) *clientTypeSections {
	return &clientTypeSections{
		sections:      []codegen.Section{header},
		seenTypes:     make(map[string]struct{}),
		seenValidated: make(map[string]struct{}),
		seenInits:     make(map[string]struct{}),
	}
}

func (s *clientTypeSections) appendEndpointTypes(adata *EndpointData, seen map[string]struct{}) {
	s.appendTypeData("client-request-body", adata.Payload.Request.ClientBody, true, seen)
	if adata.ClientWebSocket != nil {
		s.appendTypeData("client-request-body", adata.ClientWebSocket.Payload, true, seen)
	}
	for _, resp := range adata.Result.Responses {
		s.appendTypeData("client-response-body", resp.ClientBody, false, seen)
	}
	for _, gerr := range adata.Errors {
		for _, herr := range gerr.Errors {
			s.appendTypeData("client-error-body", herr.Response.ClientBody, false, seen)
		}
	}
}

func (s *clientTypeSections) appendBodyAttributeTypes(types []*TypeData, seen map[string]struct{}) {
	for _, data := range types {
		s.appendTypeData("client-body-attributes", data, false, seen)
	}
}

func (s *clientTypeSections) appendTypeData(section string, data *TypeData, trackInit bool, seen map[string]struct{}) {
	if data == nil {
		return
	}
	if _, ok := seen[data.Ref]; ok {
		if trackInit {
			s.appendInitData(data.Init)
		}
		if data.ValidateDef != "" {
			recordValidatedType(data, s.seenValidated, &s.validatedTypes)
		}
		return
	}
	seen[data.Ref] = struct{}{}
	if data.Def != "" {
		s.sections = append(s.sections, typeDeclSection(section, data))
	}
	if trackInit {
		s.appendInitData(data.Init)
	}
	if data.ValidateDef != "" {
		recordValidatedType(data, s.seenValidated, &s.validatedTypes)
	}
}

func (s *clientTypeSections) appendInitData(init *InitData) {
	if init == nil {
		return
	}
	if _, ok := s.seenInits[init.Name]; ok {
		return
	}
	s.seenInits[init.Name] = struct{}{}
	s.initData = append(s.initData, init)
}

func (s *clientTypeSections) appendBodyInitSections() {
	for _, init := range s.initData {
		s.sections = append(s.sections, bodyInitSection("client-body-init", init, true))
	}
}

func (s *clientTypeSections) appendResultInitSections(adata *EndpointData) {
	for _, resp := range adata.Result.Responses {
		s.appendTypeInitSection("client-result-init", resp.ResultInit)
	}
	for _, gerr := range adata.Errors {
		for _, herr := range gerr.Errors {
			s.appendTypeInitSection("client-error-result-init", herr.Response.ResultInit)
		}
	}
}

func (s *clientTypeSections) appendTypeInitSection(section string, init *InitData) {
	if init == nil {
		return
	}
	if _, ok := s.seenTypes[init.Name]; ok {
		return
	}
	s.seenTypes[init.Name] = struct{}{}
	s.sections = append(s.sections, typeInitSection(section, init, true))
}

func (s *clientTypeSections) appendValidateSections() {
	for _, data := range s.validatedTypes {
		s.sections = append(s.sections, validateSection("client-validate", data))
	}
}

func recordValidatedType(data *TypeData, seenValidated map[string]struct{}, validatedTypes *[]*TypeData) {
	if _, ok := seenValidated[data.Name]; ok {
		return
	}
	seenValidated[data.Name] = struct{}{}
	*validatedTypes = append(*validatedTypes, data)
}
