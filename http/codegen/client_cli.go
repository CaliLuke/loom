package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
)

// ClientCLITransport describes the generated transport-specific client CLI
// package.
type ClientCLITransport struct {
	// PathName is the transport directory used below the generated package.
	PathName string
	// DisplayName is the transport name used in generated file titles.
	DisplayName string
	// StreamingConfigurerName returns the generated streaming configurer
	// parameter name for a service variable.
	StreamingConfigurerName func(string) string
	// StreamingConfigurerType returns the generated streaming configurer type
	// for a service package.
	StreamingConfigurerType func(string) string
}

// commandData wraps the common CommandData and adds HTTP-specific fields.
type commandData struct {
	*cli.CommandData
	// Subcommands is the list of endpoint commands.
	Subcommands []*subcommandData
	// NeedDialer if true initializes the websocket dialer.
	NeedDialer bool
}

// commandData wraps the common SubcommandData and adds HTTP-specific fields.
type subcommandData struct {
	*cli.SubcommandData
	// MultipartFuncName is the name of the function used to render a multipart
	// request encoder.
	MultipartFuncName string
	// MultipartFuncName is the name of the variable used to render a multipart
	// request encoder.
	MultipartVarName string
	// StreamFlag is the flag used to identify the file to be streamed when
	// the endpoint uses SkipRequestBodyEncodeDecode.
	StreamFlag *cli.FlagData
	// BuildStreamPayload is the name of the generated function that builds the
	// request data structure that wraps the payload and the file stream for
	// endpoints that use SkipRequestBodyEncodeDecode.
	BuildStreamPayload string
}

// ClientCLIFiles returns the client HTTP CLI support file.
func ClientCLIFiles(genpkg string, data *ServicesData) []*codegen.File {
	return ClientCLIFilesForTransport(genpkg, data, httpClientCLITransport())
}

// ClientCLIFilesForTransport returns client CLI support files configured for
// transport. It is used by transports that share the HTTP command model.
func ClientCLIFilesForTransport(
	genpkg string,
	data *ServicesData,
	transport ClientCLITransport,
) []*codegen.File {
	if len(data.Expressions.Services) == 0 {
		return nil
	}
	transport = normalizeClientCLITransport(transport)
	var (
		cmds []*commandData
		svcs []*expr.HTTPServiceExpr
	)
	for _, svc := range data.Expressions.Services {
		sd := data.Get(svc.Name())
		if len(sd.Endpoints) > 0 {
			command := &commandData{
				CommandData: cli.BuildCommandData(sd.Service),
				NeedDialer:  HasWebSocket(sd),
			}

			for _, e := range sd.Endpoints {
				sub := buildSubcommandData(sd, e)
				command.Subcommands = append(command.Subcommands, sub)
				command.CommandData.Subcommands = append(command.CommandData.Subcommands, sub.SubcommandData)
			}

			command.Example = command.Subcommands[0].Example

			cmds = append(cmds, command)
			svcs = append(svcs, svc)
		}
	}
	files := make([]*codegen.File, 0, len(data.Root.API.Servers)*2) // preallocate for CLI files
	for _, svr := range data.Root.API.Servers {
		var svrData []*commandData
		for _, name := range svr.Services {
			for i, svc := range svcs {
				if svc.Name() == name {
					svrData = append(svrData, cmds[i])
				}
			}
		}
		files = append(files, endpointParser(genpkg, data.Root, svr, svrData, data, transport))
	}
	for i, svc := range svcs {
		files = append(files, payloadBuilders(genpkg, svc, cmds[i].CommandData, data, transport))
	}
	return files
}

func httpClientCLITransport() ClientCLITransport {
	return ClientCLITransport{
		PathName:    "http",
		DisplayName: "HTTP",
		StreamingConfigurerName: func(serviceVar string) string {
			return serviceVar + "Configurer"
		},
		StreamingConfigurerType: func(servicePkg string) string {
			return "*" + servicePkg + ".ConnConfigurer"
		},
	}
}

func normalizeClientCLITransport(transport ClientCLITransport) ClientCLITransport {
	defaults := httpClientCLITransport()
	if transport.PathName == "" {
		transport.PathName = defaults.PathName
	}
	if transport.DisplayName == "" {
		transport.DisplayName = defaults.DisplayName
	}
	if transport.StreamingConfigurerName == nil {
		transport.StreamingConfigurerName = defaults.StreamingConfigurerName
	}
	if transport.StreamingConfigurerType == nil {
		transport.StreamingConfigurerType = defaults.StreamingConfigurerType
	}
	return transport
}

func buildSubcommandData(sd *ServiceData, e *EndpointData) *subcommandData {
	flags, buildFunction := buildFlags(sd, e)

	sub := &subcommandData{
		SubcommandData: cli.BuildSubcommandData(sd.Service, e.Method, buildFunction, flags),
	}
	if e.MultipartRequestEncoder != nil {
		sub.MultipartVarName = e.MultipartRequestEncoder.VarName
		sub.MultipartFuncName = e.MultipartRequestEncoder.FuncName
	}
	if e.Method.SkipRequestBodyEncodeDecode {
		sub.StreamFlag = streamFlag(sd.Service.Name, e.Method.Name)
		sub.BuildStreamPayload = e.BuildStreamPayload
	}
	return sub
}

// endpointParser returns the file that implements the command line parser that
// builds the client endpoint and payload necessary to perform a request.
func endpointParser(
	genpkg string,
	root *expr.RootExpr,
	svr *expr.ServerExpr,
	data []*commandData,
	services *ServicesData,
	transport ClientCLITransport,
) *codegen.File {
	pkg := codegen.SnakeCase(codegen.Goify(svr.Name, true))
	path := filepath.Join(codegen.Gendir, transport.PathName, "cli", pkg, "cli.go")
	title := fmt.Sprintf("%s %s client CLI support package", svr.Name, transport.DisplayName)
	specs := []*codegen.ImportSpec{
		{Path: "encoding/json"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net/http"},
		{Path: "os"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http/cli", "loomhttpcli"),
		codegen.LoomNamedImport("http", "loomhttp"),
	}
	for _, sv := range svr.Services {
		svc := root.Service(sv)
		sd := services.Get(svc.Name)
		if sd == nil {
			continue
		}
		specs = append(specs, &codegen.ImportSpec{
			Path: genpkg + "/" + transport.PathName + "/" + sd.Service.PathName + "/client",
			Name: sd.Service.PkgName + "c",
		})
		// Add interceptors import if service has client interceptors
		if len(sd.Service.ClientInterceptors) > 0 {
			specs = append(specs, &codegen.ImportSpec{
				Path: genpkg + "/" + sd.Service.PathName,
				Name: sd.Service.PkgName,
			})
		}
	}

	cliData := make([]*cli.CommandData, len(data))
	for i, cmd := range data {
		cliData[i] = cmd.CommandData
	}

	sections := make([]codegen.Section, 0, 4+len(cliData))
	sections = append(sections,
		codegen.Header(title, "cli", specs),
		cli.UsageCommands(cliData),
		cli.UsageExamples(cliData),
		parseEndpointSection(cliData, data, transport),
	)
	for _, cmd := range cliData {
		sections = append(sections, cli.CommandUsage(cmd))
	}
	return &codegen.File{Path: path, Sections: sections}
}

// payloadBuilders returns the file that contains the payload constructors that
// use flag values as arguments.
func payloadBuilders(
	genpkg string,
	svc *expr.HTTPServiceExpr,
	data *cli.CommandData,
	services *ServicesData,
	transport ClientCLITransport,
) *codegen.File {
	sd := services.Get(svc.Name())
	path := filepath.Join(codegen.Gendir, transport.PathName, sd.Service.PathName, "client", "cli.go")
	title := fmt.Sprintf("%s %s client CLI support package", svc.Name(), transport.DisplayName)
	specs := []*codegen.ImportSpec{
		{Path: "encoding/json"},
		{Path: "fmt"},
		{Path: "net/http"},
		{Path: "os"},
		{Path: "strconv"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + sd.Service.PathName, Name: sd.Service.PkgName},
	}
	sections := []codegen.Section{
		codegen.Header(title, "client", specs),
	}
	for _, sub := range data.Subcommands {
		if sub.BuildFunction != nil {
			sections = append(sections, cli.PayloadBuilderSection(sub.BuildFunction))
		}
	}

	return &codegen.File{Path: path, Sections: sections}
}

// buildFlags builds the flag data and build function for an endpoint.
func buildFlags(svc *ServiceData, e *EndpointData) ([]*cli.FlagData, *cli.BuildFunctionData) {
	var (
		flags         []*cli.FlagData
		buildFunction *cli.BuildFunctionData
	)

	svcn := svc.Service.Name
	en := e.Method.Name
	if e.Payload != nil {
		if e.Payload.Request.PayloadInit != nil {
			args := e.Payload.Request.PayloadInit.ClientArgs
			args = append(args, e.Payload.Request.PayloadInit.CLIArgs...)
			flags, buildFunction = makeFlags(e, args, e.Payload.Request.PayloadType)
		} else if e.Payload.Ref != "" {
			flags = append(flags, cli.NewFlagData(svcn, en, "p", e.Method.PayloadRef, e.Method.PayloadDesc, true, e.Method.PayloadEx, e.Method.PayloadDefault))
		}
	}
	if e.Method.SkipRequestBodyEncodeDecode {
		flags = append(flags, streamFlag(svcn, en))
	}

	return flags, buildFunction
}

// makeFlags creates flag data and build function from endpoint arguments.
func makeFlags(e *EndpointData, args []*InitArgData, payload expr.DataType) ([]*cli.FlagData, *cli.BuildFunctionData) {
	var (
		fdata     = make([]*cli.FieldData, 0, len(args)) // preallocate
		flags     = make([]*cli.FlagData, len(args))
		params    = make([]string, len(args))
		pInitArgs = make([]*codegen.InitArgData, len(args))
		check     bool
	)
	for i, arg := range args {
		pInitArgs[i] = &codegen.InitArgData{
			Name:         arg.VarName,
			Pointer:      arg.Pointer,
			FieldName:    arg.FieldName,
			FieldPointer: arg.FieldPointer,
			FieldType:    arg.FieldType,
			Type:         arg.Type,
		}

		f := cli.NewFlagData(e.ServiceName, e.Method.Name, arg.VarName, arg.TypeName, arg.Description, arg.Required, arg.Example, arg.DefaultValue)
		flags[i] = f
		params[i] = f.FullName
		if arg.FieldName == "" && arg.VarName != "body" {
			continue
		}
		code, chek := cli.FieldLoadCode(f, arg.VarName, arg.TypeName, arg.Validate, arg.DefaultValue, payload, e.Payload.Ref)
		check = check || chek
		tn := arg.TypeRef
		if f.Type == "JSON" {
			// We need to declare the variable without
			// a pointer to be able to unmarshal the JSON
			// using its address.
			tn = arg.TypeName
		}
		fdata = append(fdata, &cli.FieldData{
			Name:    arg.VarName,
			VarName: arg.VarName,
			TypeRef: tn,
			Init:    code,
		})
	}

	var initCode *jen.Statement
	if e.Payload.Request.PayloadInit.ClientCode != "" {
		initCode = codegen.Expr(e.Payload.Request.PayloadInit.ClientCode)
	}
	pInit := cli.PayloadInitData{
		Code:                          initCode,
		ReturnTypeAttribute:           e.Payload.Request.PayloadInit.ReturnTypeAttribute,
		ReturnTypeAttributePointer:    e.Payload.Request.PayloadInit.ReturnIsPrimitivePointer,
		ReturnTypeAttributeUnionValue: e.Payload.Request.PayloadInit.ReturnIsUnionValue,
		ReturnIsStruct:                e.Payload.Request.PayloadInit.ReturnIsStruct,
		ReturnTypeName:                e.Payload.Request.PayloadInit.ReturnTypeName,
		ReturnTypePkg:                 e.Payload.Request.PayloadInit.ReturnTypePkg,
		Args:                          pInitArgs,
	}

	return flags, &cli.BuildFunctionData{
		Name:         "Build" + e.Method.VarName + "Payload",
		ActualParams: params,
		FormalParams: params,
		ServiceName:  e.ServiceName,
		MethodName:   e.Method.Name,
		ResultType:   e.Payload.Ref,
		Fields:       fdata,
		PayloadInit:  &pInit,
		CheckErr:     check,
	}
}

// streamFlag returns the flag used to specify the upload file for endpoints
// that use SkipRequestBodyEncodeDecode.
func streamFlag(svcn, en string) *cli.FlagData {
	return cli.NewFlagData(svcn, en, "stream", "string", "path to file containing the streamed request body", true, "loom.bin", nil)
}

// streamingCmdExists returns true if at least one command in the list of commands
// uses stream for sending payload/result.
func streamingCmdExists(data []*commandData) bool {
	for _, c := range data {
		if c.NeedDialer {
			return true
		}
	}
	return false
}
