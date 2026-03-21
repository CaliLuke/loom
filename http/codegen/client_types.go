package codegen

import (
	"path/filepath"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

// ClientTypeFiles returns the HTTP transport client types files.
func ClientTypeFiles(genpkg string, data *ServicesData) []*codegen.File {
	fw := make([]*codegen.File, len(data.Expressions.Services))
	for i, svc := range data.Expressions.Services {
		fw[i] = clientType(genpkg, svc, make(map[string]struct{}), data)
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
	var (
		path    string
		data    = services.Get(svc.Name())
		svcName = data.Service.PathName
	)
	path = filepath.Join(codegen.Gendir, "http", svcName, "client", "types.go")
	imports := []*codegen.ImportSpec{
		{Path: "encoding/json"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "unicode/utf8"},
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
		codegen.GoaImport(""),
		codegen.GoaNamedImport("http", "goahttp"),
	}
	header := codegen.Header(svc.Name()+" HTTP client types", "client", imports)

	var (
		initData       []*InitData
		validatedTypes []*TypeData
		seenValidated  = make(map[string]struct{}) // Track validated types to avoid duplicates
		seenInit       = make(map[string]struct{}) // Track init functions to avoid duplicates

		sections = []codegen.Section{header}
	)

	appendTypeData := func(section string, data *TypeData, trackInit bool) {
		if data == nil {
			return
		}
		if _, ok := seen[data.Ref]; ok {
			return
		}
		seen[data.Ref] = struct{}{}
		if data.Def != "" {
			sections = append(sections, typeDeclSection(section, data))
		}
		if trackInit && data.Init != nil {
			if _, ok := seenInit[data.Init.Name]; !ok {
				seenInit[data.Init.Name] = struct{}{}
				initData = append(initData, data.Init)
			}
		}
		if data.ValidateDef != "" {
			recordValidatedType(data, seenValidated, &validatedTypes)
		}
	}

	for _, a := range svc.HTTPEndpoints {
		adata := data.Endpoint(a.Name())
		appendTypeData("client-request-body", adata.Payload.Request.ClientBody, true)
		if adata.ClientWebSocket != nil {
			appendTypeData("client-request-body", adata.ClientWebSocket.Payload, true)
		}
		for _, resp := range adata.Result.Responses {
			appendTypeData("client-response-body", resp.ClientBody, false)
		}
		for _, gerr := range adata.Errors {
			for _, herr := range gerr.Errors {
				appendTypeData("client-error-body", herr.Response.ClientBody, false)
			}
		}
	}

	for _, data := range data.ClientBodyAttributeTypes {
		appendTypeData("client-body-attributes", data, false)
	}

	// union sum types
	for _, u := range data.UnionTypes {
		sections = append(sections, unionTypeSection("client-union-type", u))
	}

	// body constructors
	for _, init := range initData {
		sections = append(sections, bodyInitSection("client-body-init", init, true))
	}

	// Track generated init functions to avoid duplicates
	seenInits := make(map[string]struct{})

	for _, adata := range data.Endpoints {
		// response to method result (client)
		for _, resp := range adata.Result.Responses {
			if init := resp.ResultInit; init != nil {
				if _, ok := seenInits[init.Name]; !ok {
					seenInits[init.Name] = struct{}{}
					sections = append(sections, typeInitSection("client-result-init", init, true))
				}
			}
		}

		// error response to method result (client)
		for _, gerr := range adata.Errors {
			for _, herr := range gerr.Errors {
				if init := herr.Response.ResultInit; init != nil {
					if _, ok := seenInits[init.Name]; !ok {
						seenInits[init.Name] = struct{}{}
						sections = append(sections, typeInitSection("client-error-result-init", init, true))
					}
				}
			}
		}
	}

	// body attribute types
	// validate methods
	for _, data := range validatedTypes {
		sections = append(sections, validateSection("client-validate", data))
	}
	return &codegen.File{Path: path, Sections: sections}
}

func recordValidatedType(data *TypeData, seenValidated map[string]struct{}, validatedTypes *[]*TypeData) {
	if _, ok := seenValidated[data.Name]; ok {
		return
	}
	seenValidated[data.Name] = struct{}{}
	*validatedTypes = append(*validatedTypes, data)
}
