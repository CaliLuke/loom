package codegen

import (
	"path/filepath"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

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
	var (
		path    string
		data    = services.Get(svc.Name())
		svcName = data.Service.PathName
	)
	path = filepath.Join(codegen.Gendir, "http", svcName, "server", "types.go")
	imports := []*codegen.ImportSpec{
		{Path: "encoding/json"},
		{Path: "fmt"},
		{Path: "net/url"},
		{Path: "unicode/utf8"},
		{Path: genpkg + "/" + svcName, Name: data.Service.PkgName},
		codegen.GoaImport(""),
		codegen.GoaNamedImport("http", "goahttp"),
		{Path: genpkg + "/" + svcName + "/" + "views", Name: data.Service.ViewsPkg},
	}
	header := codegen.Header(svc.Name()+" HTTP server types", "server", imports)

	var (
		initData       []*InitData
		validatedTypes []*TypeData

		sections = []codegen.Section{header}
	)

	// request body types
	for _, a := range svc.HTTPEndpoints {
		adata := data.Endpoint(a.Name())
		if data := adata.Payload.Request.ServerBody; data != nil {
			if data.Def != "" {
				sections = append(sections, typeDeclSection("request-body-type-decl", data))
			}
			if data.ValidateDef != "" {
				validatedTypes = append(validatedTypes, data)
			}
		}
		if adata.ServerWebSocket != nil && !adata.Method.IsJSONRPC {
			if data := adata.ServerWebSocket.Payload; data != nil {
				if data.Def != "" {
					sections = append(sections, typeDeclSection("request-stream-payload-type-decl", data))
				}
				if data.ValidateDef != "" {
					validatedTypes = append(validatedTypes, data)
				}
			}
		}
	}

	// response body types
	for _, a := range svc.HTTPEndpoints {
		adata := data.Endpoint(a.Name())
		for _, resp := range adata.Result.Responses {
			for _, tdata := range resp.ServerBody {
				if generated, ok := data.ServerTypeNames[tdata.Name]; ok && !generated {
					if tdata.Def != "" {
						sections = append(sections, typeDeclSection("response-server-body", tdata))
					}
					if tdata.Init != nil {
						initData = append(initData, tdata.Init)
					}
					if tdata.ValidateDef != "" {
						validatedTypes = append(validatedTypes, tdata)
					}
					data.ServerTypeNames[tdata.Name] = true
				}
			}
		}
	}

	// error body types
	for _, a := range svc.HTTPEndpoints {
		adata := data.Endpoint(a.Name())
		for _, gerr := range adata.Errors {
			for _, herr := range gerr.Errors {
				for _, tdata := range herr.Response.ServerBody {
					// Check if this error type has already been generated
					if generated, ok := data.ServerTypeNames[tdata.Name]; ok && generated {
						continue
					}

					if tdata.Def != "" {
						sections = append(sections, typeDeclSection("error-body-type-decl", tdata))
						// Mark this type as generated
						data.ServerTypeNames[tdata.Name] = true
					}
					if tdata.Init != nil {
						initData = append(initData, tdata.Init)
					}
					if tdata.ValidateDef != "" {
						validatedTypes = append(validatedTypes, tdata)
					}
				}
			}
		}
	}

	// body attribute types
	for _, tdata := range data.ServerBodyAttributeTypes {
		if tdata.Def != "" {
			sections = append(sections, typeDeclSection("server-body-attributes", tdata))
		}

		if tdata.ValidateDef != "" {
			validatedTypes = append(validatedTypes, tdata)
		}
	}

	// union sum types
	for _, u := range data.UnionTypes {
		sections = append(sections, unionTypeSection("server-union-type", u))
	}

	// body constructors
	for _, init := range initData {
		sections = append(sections, bodyInitSection("server-body-init", init, false))
	}

	for _, adata := range data.Endpoints {
		// request to method payload
		if init := adata.Payload.Request.PayloadInit; init != nil {
			sections = append(sections, typeInitSection("server-payload-init", init, false))
		}
		if IsWebSocketEndpoint(adata) && adata.ServerWebSocket.Payload != nil {
			if init := adata.ServerWebSocket.Payload.Init; init != nil {
				sections = append(sections, typeInitSection("server-payload-init", init, false))
			}
		}
	}

	// validate methods
	for _, data := range validatedTypes {
		sections = append(sections, validateSection("server-validate", data))
	}

	return &codegen.File{Path: path, Sections: sections}
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
