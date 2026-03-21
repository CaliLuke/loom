package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
)

// ExampleInterceptorsFiles returns the files for the example server and client interceptors.
func ExampleInterceptorsFiles(genpkg string, r *expr.RootExpr, services *ServicesData) []*codegen.File {
	var fw []*codegen.File
	for _, svc := range r.Services {
		if f := exampleInterceptorsFile(genpkg, svc, services); f != nil {
			fw = append(fw, f...)
		}
	}
	return fw
}

// exampleInterceptorsFile returns the example interceptors for the given service.
func exampleInterceptorsFile(genpkg string, svc *expr.ServiceExpr, services *ServicesData) []*codegen.File {
	sdata := services.Get(svc.Name)
	data := map[string]any{
		"ServiceName":        sdata.Name,
		"StructName":         sdata.StructName,
		"PkgName":            "interceptors",
		"ServerInterceptors": sdata.ServerInterceptors,
		"ClientInterceptors": sdata.ClientInterceptors,
	}

	var files []*codegen.File

	// Generate server interceptor if needed and file doesn't exist
	if len(sdata.ServerInterceptors) > 0 {
		serverPath := filepath.Join("interceptors", sdata.PathName+"_server.go")
		if _, err := os.Stat(serverPath); os.IsNotExist(err) {
			files = append(files, &codegen.File{
				Path: serverPath,
				Sections: []codegen.Section{
					codegen.Header(fmt.Sprintf("%s example server interceptors", sdata.Name), "interceptors", []*codegen.ImportSpec{
						{Path: "context"},
						{Path: "fmt"},
						{Path: "goa.design/clue/log"},
						codegen.GoaImport(""),
						{Path: path.Join(genpkg, sdata.PathName), Name: sdata.PkgName},
					}),
					exampleInterceptorSection("example-server-interceptor", data, true),
				},
			})
		}
	}

	// Generate client interceptor if needed and file doesn't exist
	if len(sdata.ClientInterceptors) > 0 {
		clientPath := filepath.Join("interceptors", sdata.PathName+"_client.go")
		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			files = append(files, &codegen.File{
				Path: clientPath,
				Sections: []codegen.Section{
					codegen.Header(fmt.Sprintf("%s example client interceptors", sdata.Name), "interceptors", []*codegen.ImportSpec{
						{Path: "context"},
						{Path: "fmt"},
						{Path: "goa.design/clue/log"},
						codegen.GoaImport(""),
						{Path: path.Join(genpkg, sdata.PathName), Name: sdata.PkgName},
					}),
					exampleInterceptorSection("example-client-interceptor", data, false),
				},
			})
		}
	}

	return files
}

func exampleInterceptorSection(name string, data map[string]any, server bool) codegen.Section {
	return codegen.NewJenniferSection(name, func(stmt *jen.Statement) {
		structName := data["StructName"].(string)
		serviceName := data["ServiceName"].(string)
		pkgName := data["PkgName"].(string)
		mode := "client"
		implements := "client interceptors"
		action := "Sending request"
		interceptors := data["ClientInterceptors"].([]*InterceptorData)
		if server {
			mode = "server"
			implements = "server interceptor"
			action = "Processing request"
			interceptors = data["ServerInterceptors"].([]*InterceptorData)
		}

		stmt.Add(codegen.Expr(fmt.Sprintf("// %s%sInterceptors implements the %s for the %s service.\ntype %s%sInterceptors struct {\n}", structName, strings.Title(mode), implements, serviceName, structName, strings.Title(mode))))
		stmt.Line()
		stmt.Add(codegen.Expr(fmt.Sprintf("// New%s%sInterceptors creates a new %s interceptor for the %s service.\nfunc New%s%sInterceptors() *%s%sInterceptors {\nreturn &%s%sInterceptors{}\n}", structName, strings.Title(mode), mode, serviceName, structName, strings.Title(mode), structName, strings.Title(mode), structName, strings.Title(mode))))
		stmt.Line()
		for _, interceptor := range interceptors {
			if interceptor.Description != "" {
				codegen.Doc(stmt, interceptor.Description)
			}
			responseAction := "Received response"
			if server {
				responseAction = "Response"
			}
			stmt.Add(codegen.Expr(fmt.Sprintf(`func (i *%s%sInterceptors) %s(ctx context.Context, info *%s.%sInfo, next goa.Endpoint) (any, error) {
log.Printf(ctx, "[%s] %s: %%v", info.RawPayload())
resp, err := next(ctx, info.RawPayload())
if err != nil {
	log.Printf(ctx, "[%s] Error: %%v", err)
	return nil, err
}
log.Printf(ctx, "[%s] %s: %%v", resp)
return resp, nil
}`, structName, strings.Title(mode), interceptor.Name, pkgName, interceptor.Name, interceptor.Name, action, interceptor.Name, interceptor.Name, responseAction)))
			stmt.Line()
		}
	})
}
