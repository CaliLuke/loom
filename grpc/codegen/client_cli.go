package codegen

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/codegen/example"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
)

// ClientCLIFiles returns the CLI files to generate a command-line client that
// makes gRPC requests. It generates a CLI support package only for the
// servers that host a gRPC service, matching the example client.
func ClientCLIFiles(genpkg string, services *ServicesData) []*codegen.File {
	if len(services.Root.API.GRPC.Services) == 0 {
		return nil
	}
	var (
		data = make([]*cli.CommandData, 0, len(services.Root.API.GRPC.Services))
		svcs = make([]*expr.GRPCServiceExpr, 0, len(services.Root.API.GRPC.Services))
	)
	for _, svc := range services.Root.API.GRPC.Services {
		if len(svc.GRPCEndpoints) == 0 {
			continue
		}
		data = append(data, buildCommandData(services.Get(svc.Name())))
		svcs = append(svcs, svc)
	}
	files := make([]*codegen.File, 0, len(services.Root.API.Servers)+len(svcs))
	servers := example.NewServersData()
	for _, svr := range services.Root.API.Servers {
		if !servers.Get(svr, services.Root).HostsGRPC() {
			continue
		}
		files = append(files, endpointParser(genpkg, services, svr, data))
	}
	for i, svc := range svcs {
		files = append(files, payloadBuilders(genpkg, svc, data[i], services))
	}
	return files
}

// buildCommandData returns the CLI command of the service sd, which has
// endpoints.
func buildCommandData(sd *ServiceData) *cli.CommandData {
	command := cli.BuildCommandData(sd.Service)
	for _, e := range sd.Endpoints {
		flags, buildFunction := buildFlags(e)
		subcmd := cli.BuildSubcommandData(sd.Service, e.Method, buildFunction, flags)
		command.Subcommands = append(command.Subcommands, subcmd)
	}
	command.Example = command.Subcommands[0].Example
	return command
}

// endpointParser returns the file that implements the command line parser that
// builds the client endpoint and payload necessary to perform a request.
func endpointParser(genpkg string, services *ServicesData, svr *expr.ServerExpr, data []*cli.CommandData) *codegen.File {
	pkg := naming.ServerDir(svr.Name)
	fpath := filepath.Join(codegen.Gendir, "grpc", "cli", pkg, "cli.go")
	title := svr.Name + " gRPC client CLI support package"
	specs := []*codegen.ImportSpec{
		{Path: "context"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "os"},
		{Path: "strconv"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("grpc", "loomgrpc"),
		{Path: "google.golang.org/grpc", Name: "grpc"},
	}
	// Add structpb import if Any type is used
	needsAnyPb := false
	for _, svc := range services.Root.API.GRPC.Services {
		for _, e := range svc.GRPCEndpoints {
			if hasAnyType(e.MethodExpr.Payload) || hasAnyType(e.MethodExpr.Result) {
				needsAnyPb = true
				break
			}
		}
		if needsAnyPb {
			break
		}
	}
	if needsAnyPb {
		specs = append(specs,
			&codegen.ImportSpec{Path: "google.golang.org/protobuf/types/known/structpb", Name: "structpb"},
		)
	}
	for _, svc := range services.Root.API.GRPC.Services {
		sd := services.Get(svc.Name())
		if sd == nil {
			continue
		}
		svcName := sd.Service.PathName
		specs = append(specs,
			&codegen.ImportSpec{Path: path.Join(genpkg, "grpc", svcName, "client"), Name: sd.Service.PkgName + "c"},
			&codegen.ImportSpec{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: svcName + pbPkgName})
		// Add interceptors import if service has client interceptors
		if len(sd.Service.ClientInterceptors) > 0 {
			specs = append(specs, &codegen.ImportSpec{
				Path: genpkg + "/" + sd.Service.PathName,
				Name: sd.Service.PkgName,
			})
		}
	}

	sections := make([]codegen.Section, 0, 4+len(data))
	sections = append(sections,
		codegen.Header(title, "cli", specs),
		cli.UsageCommands(data),
		cli.UsageExamples(data),
		grpcParseEndpointSection(data),
	)
	for _, cmd := range data {
		sections = append(sections, cli.CommandUsage(cmd))
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

// payloadBuilders returns the file that contains the payload constructors that
// use flag values as arguments.
func payloadBuilders(genpkg string, svc *expr.GRPCServiceExpr, data *cli.CommandData, services *ServicesData) *codegen.File {
	sd := services.Get(svc.Name())
	svcName := sd.Service.PathName
	fpath := filepath.Join(codegen.Gendir, "grpc", svcName, "client", "cli.go")
	title := svc.Name() + " gRPC client CLI support package"
	specs := []*codegen.ImportSpec{
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "fmt"},
		{Path: "strconv"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("grpc", "loomgrpc"),
		{Path: "google.golang.org/protobuf/encoding/protojson", Name: "protojson"},
		{Path: path.Join(genpkg, svcName), Name: sd.Service.PkgName},
		{Path: path.Join(genpkg, "grpc", svcName, pbPkgName), Name: sd.PkgName},
	}
	specs = append(specs, sd.Service.UserTypeImports...)
	// Add structpb import if Any type is used
	needsAnyPb := false
	for _, e := range svc.GRPCEndpoints {
		if hasAnyType(e.MethodExpr.Payload) || hasAnyType(e.MethodExpr.Result) {
			needsAnyPb = true
			break
		}
	}
	if needsAnyPb {
		specs = append(specs,
			&codegen.ImportSpec{Path: "google.golang.org/protobuf/types/known/structpb", Name: "structpb"},
		)
	}
	fileSD, specs := services.fileData(svc.Name(), specs)
	if fileSD != sd {
		sd, data = fileSD, buildCommandData(fileSD)
	}
	sections := []codegen.Section{
		codegen.Header(title, "client", specs),
	}
	for _, sub := range data.Subcommands {
		if sub.BuildFunction != nil {
			sections = append(sections, cli.PayloadBuilderSection(sub.BuildFunction, sd.Service.Scope))
		}
	}
	for _, helper := range clientCLIPayloadTransformHelpers(svc, sd) {
		sections = append(sections, grpcTransformHelperSection(helper.TransformFunctionData))
	}
	return &codegen.File{Path: fpath, Sections: sections}
}

func clientCLIPayloadTransformHelpers(svc *expr.GRPCServiceExpr, sd *ServiceData) []*TransformHelperData {
	// Scan candidates in sd.transformHelpers order, never map order, so the
	// emitted helper sections are byte-stable across generations.
	candidates := make([]*TransformHelperData, 0, len(sd.transformHelpers))
	for _, helper := range sd.transformHelpers {
		if helper.Kind != validateServer {
			continue
		}
		candidates = append(candidates, helper)
	}

	var selected []*TransformHelperData
	seen := make(map[string]struct{}, len(candidates))
	var collectFromCode func(string)
	collectFromCode = func(src string) {
		for _, helper := range candidates {
			if _, ok := seen[helper.Name]; ok || !strings.Contains(src, helper.Name+"(") {
				continue
			}
			seen[helper.Name] = struct{}{}
			selected = append(selected, helper)
			collectFromCode(helper.Code)
		}
	}

	for _, grpcEndpoint := range svc.GRPCEndpoints {
		endpoint := sd.Endpoint(grpcEndpoint.Name())
		if endpoint == nil || endpoint.Request == nil || endpoint.Request.ServerConvert == nil || endpoint.Request.ServerConvert.Init == nil {
			continue
		}
		collectFromCode(endpoint.Request.ServerConvert.Init.Code)
	}
	return selected
}

func buildFlags(e *EndpointData) ([]*cli.FlagData, *cli.BuildFunctionData) {
	if e.Request != nil {
		return makeFlags(e, e.Request.CLIArgs)
	}
	return nil, nil
}

func makeFlags(e *EndpointData, args []*InitArgData) ([]*cli.FlagData, *cli.BuildFunctionData) {
	var (
		fdata     = make([]*cli.FieldData, 0, len(args))
		flags     = make([]*cli.FlagData, len(args))
		params    = make([]string, len(args))
		pInitArgs = make([]*codegen.InitArgData, len(args))
		check     bool
		pinit     *cli.PayloadInitData
	)
	for i, arg := range args {
		pInitArgs[i] = &codegen.InitArgData{
			Name:      arg.Name,
			FieldName: arg.FieldName,
			FieldType: arg.FieldType,
			Type:      arg.Type,
		}

		f := cli.NewFlagData(e.ServiceName, e.Method.Name, arg.Name, arg.TypeName, arg.Description, arg.Required, arg.Example, arg.DefaultValue)
		if arg.ProtoMessage {
			// encoding/json/v2 cannot set the oneof fields of a message.
			f.Unmarshal = "protojson.Unmarshal"
		}
		flags[i] = f
		params[i] = f.FullName
		code, chek := cli.FieldLoadCode(f, arg.Name, arg.TypeName, arg.Validate, arg.DefaultValue, e.PayloadType, e.PayloadRef)
		check = check || chek
		tn := arg.TypeRef
		if f.Type == "JSON" {
			// We need to declare the variable without
			// a pointer to be able to unmarshal the JSON
			// using its address.
			tn = arg.TypeName
		}
		fdata = append(fdata, &cli.FieldData{
			Name:    arg.Name,
			VarName: arg.Name,
			TypeRef: tn,
			Init:    code,
		})
	}
	if e.Method.PayloadRef == "" {
		return flags, nil
	}
	if e.Request.ServerConvert != nil {
		var initCode *jen.Statement
		if e.Request.ServerConvert.Init.Code != "" {
			initCode = codegen.Expr(e.Request.ServerConvert.Init.Code)
		}
		pinit = &cli.PayloadInitData{
			Code:           initCode,
			ReturnIsStruct: e.Request.ServerConvert.Init.ReturnIsStruct,
			ReturnTypePkg:  e.Request.ServerConvert.Init.ReturnTypePkg,
			Args:           pInitArgs,
			ErrorAware:     e.Request.ServerConvert.Init.ErrorAware,
		}
	}

	return flags, &cli.BuildFunctionData{
		Name:         "Build" + e.Method.VarName + "Payload",
		ActualParams: params,
		FormalParams: params,
		ServiceName:  e.ServiceName,
		MethodName:   e.Method.Name,
		ResultType:   e.PayloadRef,
		Fields:       fdata,
		PayloadInit:  pinit,
		CheckErr:     check,
	}
}
