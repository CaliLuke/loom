package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
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
						codegen.LoomImport(""),
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
						codegen.LoomImport(""),
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
		receiverType := structName + strings.Title(mode) + "Interceptors"

		stmt.Comment(fmt.Sprintf("%s implements the %s for the %s service.", receiverType, implements, serviceName)).Line()
		stmt.Type().Id(receiverType).StructFunc(func(*jen.Group) {})
		stmt.Line()
		stmt.Comment(fmt.Sprintf("New%s creates a new %s interceptor for the %s service.", receiverType, mode, serviceName)).Line()
		stmt.Func().Id("New" + receiverType).Params().Op("*").Id(receiverType).Block(
			jen.Return(jen.Op("&").Id(receiverType).Values()),
		)
		stmt.Line()
		for _, interceptor := range interceptors {
			if interceptor.Description != "" {
				codegen.Doc(stmt, interceptor.Description)
			}
			responseAction := "Received response"
			if server {
				responseAction = "Response"
			}
			stmt.Func().
				Params(jen.Id("i").Op("*").Id(receiverType)).
				Id(interceptor.Name).
				Params(
					jen.Id("ctx").Add(codegen.TypeRef("context.Context")),
					jen.Id("info").Add(codegen.TypeRef("*"+pkgName+"."+interceptor.Name+"Info")),
					jen.Id("next").Add(codegen.TypeRef("loom.Endpoint")),
				).
				Params(jen.Any(), jen.Error()).
				BlockFunc(func(group *jen.Group) {
					group.Add(codegen.Expr(fmt.Sprintf(`log.Printf(ctx, "[%s] %s: %%v", info.RawPayload())`, interceptor.Name, action)))
					group.Id("resp").Op(",").Id("err").Op(":=").Id("next").Call(jen.Id("ctx"), jen.Id("info").Dot("RawPayload").Call())
					group.If(jen.Id("err").Op("!=").Nil()).Block(
						codegen.Expr(fmt.Sprintf(`log.Printf(ctx, "[%s] Error: %%v", err)`, interceptor.Name)),
						jen.Return(jen.Nil(), jen.Id("err")),
					)
					group.Add(codegen.Expr(fmt.Sprintf(`log.Printf(ctx, "[%s] %s: %%v", resp)`, interceptor.Name, responseAction)))
					group.Return(jen.Id("resp"), jen.Nil())
				})
			stmt.Line()
		}
	})
}
