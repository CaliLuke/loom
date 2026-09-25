package service

import (
	"fmt"
	"os"
	"path"
	"path/filepath"

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
	_, svcQual := exampleInterceptorsImports(genpkg, sdata)
	data := map[string]any{
		"ServiceName":        sdata.Name,
		"StructName":         sdata.StructName,
		"PkgName":            svcQual,
		"ServerInterceptors": sdata.ServerInterceptors,
		"ClientInterceptors": sdata.ClientInterceptors,
	}

	var files []*codegen.File

	// Generate server interceptor if needed and file doesn't exist
	if len(sdata.ServerInterceptors) > 0 {
		serverPath := filepath.Join("interceptors", sdata.PathName+"_server.go")
		if _, err := os.Stat(serverPath); os.IsNotExist(err) {
			serverImports, _ := exampleInterceptorsImports(genpkg, sdata)
			files = append(files, &codegen.File{
				Path: serverPath,
				Sections: []codegen.Section{
					codegen.Header(fmt.Sprintf("%s example server interceptors", sdata.Name), "interceptors", serverImports),
					exampleInterceptorSection("example-server-interceptor", data, true),
				},
			})
		}
	}

	// Generate client interceptor if needed and file doesn't exist
	if len(sdata.ClientInterceptors) > 0 {
		clientPath := filepath.Join("interceptors", sdata.PathName+"_client.go")
		if _, err := os.Stat(clientPath); os.IsNotExist(err) {
			clientImports, _ := exampleInterceptorsImports(genpkg, sdata)
			files = append(files, &codegen.File{
				Path: clientPath,
				Sections: []codegen.Section{
					codegen.Header(fmt.Sprintf("%s example client interceptors", sdata.Name), "interceptors", clientImports),
					exampleInterceptorSection("example-client-interceptor", data, false),
				},
			})
		}
	}

	return files
}

// exampleInterceptorsImports returns the imports of the example interceptor
// files of the service described by sdata and the name under which they
// import the service package, which declares the interceptor Info types. The
// service package is imported under an alias when its name clashes with
// another import of the files.
func exampleInterceptorsImports(genpkg string, sdata *Data) ([]*codegen.ImportSpec, string) {
	svcImport := &codegen.ImportSpec{Path: path.Join(genpkg, sdata.PathName), Name: sdata.PkgName}
	imports := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "fmt"},
		{Path: "github.com/CaliLuke/loom/clue/log"},
		codegen.LoomImport(""),
		svcImport,
	}
	if alias, ok := codegen.AliasClashingImports(imports, []string{svcImport.Path})[svcImport.Path]; ok {
		svcImport.Name = alias
	}
	return imports, svcImport.Name
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
		receiverType := structName + codegen.Goify(mode, true) + "Interceptors"

		stmt.Comment(codegen.LineComment(fmt.Sprintf("%s implements the %s for the %s service.", receiverType, implements, serviceName))).Line()
		stmt.Type().Id(receiverType).StructFunc(func(*jen.Group) {})
		stmt.Line()
		stmt.Comment(codegen.LineComment(fmt.Sprintf("New%s creates a new %s interceptor for the %s service.", receiverType, mode, serviceName))).Line()
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
					jen.Id("info").Op("*").Add(codegen.PkgQual(pkgName, interceptor.Name+"Info")),
					jen.Id("next").Add(codegen.TypeRef("loom.Endpoint")),
				).
				Params(jen.Any(), jen.Error()).
				BlockFunc(func(group *jen.Group) {
					group.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(
						jen.Id("ctx"),
						jen.Lit("["+interceptor.Name+"] "+action+": %v"),
						jen.Id("info").Dot("RawPayload").Call(),
					)
					group.Id("resp").Op(",").Id("err").Op(":=").Id("next").Call(jen.Id("ctx"), jen.Id("info").Dot("RawPayload").Call())
					group.If(jen.Id("err").Op("!=").Nil()).Block(
						jen.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(
							jen.Id("ctx"),
							jen.Lit("["+interceptor.Name+"] Error: %v"),
							jen.Id("err"),
						),
						jen.Return(jen.Nil(), jen.Id("err")),
					)
					group.Qual("github.com/CaliLuke/loom/clue/log", "Printf").Call(
						jen.Id("ctx"),
						jen.Lit("["+interceptor.Name+"] "+responseAction+": %v"),
						jen.Id("resp"),
					)
					group.Return(jen.Id("resp"), jen.Nil())
				})
			stmt.Line()
		}
	})
}
