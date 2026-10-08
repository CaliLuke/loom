package codegen

import (
	"fmt"
	"path/filepath"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/naming"
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
			cmds = append(cmds, buildCommandData(data.Ctx, sd))
			svcs = append(svcs, svc)
		}
	}
	files := make([]*codegen.File, 0, len(data.Root.API.Servers)*2) // preallocate for CLI files
	for _, svr := range data.Root.API.Servers {
		if !hostsTransportService(svr, data.Expressions) {
			continue
		}
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

// buildCommandData returns the CLI command of the service sd, which has
// endpoints.
func buildCommandData(ctx *codegen.Context, sd *ServiceData) *commandData {
	command := &commandData{
		CommandData: cli.BuildCommandData(sd.Service),
		NeedDialer:  HasWebSocket(sd),
	}
	for _, e := range sd.Endpoints {
		sub := buildSubcommandData(ctx, sd, e)
		command.Subcommands = append(command.Subcommands, sub)
		command.CommandData.Subcommands = append(command.CommandData.Subcommands, sub.SubcommandData)
	}
	for _, sub := range command.Subcommands {
		if sub.Example != "" {
			command.Example = sub.Example
			break
		}
	}
	return command
}

func buildSubcommandData(ctx *codegen.Context, sd *ServiceData, e *EndpointData) *subcommandData {
	flags, buildFunction := buildFlags(ctx, sd, e)

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
	pkg := naming.ServerDir(svr.Name)
	path := filepath.Join(codegen.Gendir, transport.PathName, "cli", pkg, "cli.go")
	title := fmt.Sprintf("%s %s client CLI support package", svr.Name, transport.DisplayName)
	specs := []*codegen.ImportSpec{
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "errors"},
		{Path: "flag"},
		{Path: "fmt"},
		{Path: "net/http"},
		{Path: "os"},
		{Path: "strconv"},
		codegen.LoomImport(""),
		codegen.LoomNamedImport("http/cli", "loomhttpcli"),
		codegen.LoomNamedImport("http", "loomhttp"),
	}
	clientPaths := make([]string, 0, len(data))
	for _, sv := range svr.Services {
		svc := root.Service(sv)
		sd := services.Get(svc.Name)
		if sd == nil || len(sd.Endpoints) == 0 {
			continue
		}
		clientPaths = append(clientPaths, genpkg+"/"+transport.PathName+"/"+sd.Service.PathName+"/client")
		// Add interceptors import if service has client interceptors
		if len(sd.Service.ClientInterceptors) > 0 {
			specs = append(specs, &codegen.ImportSpec{
				Path: genpkg + "/" + sd.Service.PathName,
				Name: sd.Service.PkgName,
			})
		}
	}
	data = allocateHTTPCommandIdentifiers(data, specs, transport)
	for i, clientPath := range clientPaths {
		specs = append(specs, &codegen.ImportSpec{Path: clientPath, Name: data[i].PkgName})
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

func allocateHTTPCommandIdentifiers(
	commands []*commandData,
	imports []*codegen.ImportSpec,
	transport ClientCLITransport,
) []*commandData {
	reserved := []string{
		"UsageCommands", "UsageExamples", "ParseEndpoint", "commandLine",
		"scheme", "host", "doer", "enc", "dec", "restore", "dialer",
		"command", "args", "svcn", "epn", "path", "err", "data", "endpoint", "c", "value",
	}
	for _, spec := range imports {
		reserved = append(reserved, naming.ImportName(spec.Path, spec.Name))
	}
	common := make([]*cli.CommandData, len(commands))
	for i, command := range commands {
		common[i] = command.CommandData
		if command.NeedDialer {
			reserved = append(reserved, transport.StreamingConfigurerName(command.VarName))
		}
		if command.Interceptors != nil {
			reserved = append(reserved, command.Interceptors.VarName)
		}
		for _, subcommand := range command.Subcommands {
			if subcommand.MultipartVarName != "" {
				reserved = append(reserved, subcommand.MultipartVarName)
			}
		}
	}
	allocated := cli.AllocateCommandIdentifiers(common, reserved)
	out := make([]*commandData, len(commands))
	for i, command := range commands {
		copyCommand := *command
		copyCommand.CommandData = allocated[i]
		copyCommand.Subcommands = make([]*subcommandData, len(command.Subcommands))
		for j, subcommand := range command.Subcommands {
			copySubcommand := *subcommand
			copySubcommand.SubcommandData = allocated[i].Subcommands[j]
			if subcommand.StreamFlag != nil {
				for _, flag := range copySubcommand.Flags {
					if flag.FullName == subcommand.StreamFlag.FullName {
						copySubcommand.StreamFlag = flag
						break
					}
				}
			}
			copyCommand.Subcommands[j] = &copySubcommand
		}
		out[i] = &copyCommand
	}
	return out
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
	fileSD, imports := services.FileData(svc.Name(), clientCLIImports(genpkg, sd))
	if fileSD != sd {
		data = buildCommandData(services.Ctx, fileSD).CommandData
	}
	sections := []codegen.Section{
		codegen.Header(title, "client", imports),
	}
	for _, sub := range data.Subcommands {
		if sub.BuildFunction != nil {
			sections = append(sections, cli.PayloadBuilderSection(sub.BuildFunction, fileSD.Service.Scope))
		}
	}

	return &codegen.File{Path: path, Sections: sections}
}

// clientCLIImports returns the imports of the client CLI support file of the
// service.
func clientCLIImports(genpkg string, sd *ServiceData) []*codegen.ImportSpec {
	jsonCLI := codegen.LoomNamedImport("http/cli", "loomhttpcli")
	imports := append([]*codegen.ImportSpec{
		{Path: "encoding/json/v2", Name: "json"},
		{Path: "fmt"},
		{Path: "net/http"},
		{Path: "os"},
		{Path: "strconv"},
		{Path: "unicode/utf8"},
		codegen.LoomImport(""),
		jsonCLI,
		codegen.LoomNamedImport("http", "loomhttp"),
		{Path: genpkg + "/" + sd.Service.PathName, Name: sd.Service.PkgName},
	}, sd.Service.UserTypeImports...)
	if alias, ok := codegen.AliasClashingImports(imports, []string{jsonCLI.Path})[jsonCLI.Path]; ok {
		jsonCLI.Name = alias
	}
	return imports
}

// buildFlags builds the flag data and build function for an endpoint.
func buildFlags(ctx *codegen.Context, svc *ServiceData, e *EndpointData) ([]*cli.FlagData, *cli.BuildFunctionData) {
	var (
		flags         []*cli.FlagData
		buildFunction *cli.BuildFunctionData
	)

	svcn := svc.Service.Name
	en := e.Method.Name
	if e.Payload != nil {
		if init := e.Payload.Request.PayloadInit; init != nil &&
			(len(init.ClientArgs)+len(init.CLIArgs) > 0 || init.ReturnIsStruct) {
			// An object payload without arguments, such as an object type
			// without attributes, gets a builder without flags: the command
			// parser cannot refer to the service type itself.
			args := init.ClientArgs
			args = append(args, init.CLIArgs...)
			flags, buildFunction = makeFlags(e, args, e.Payload.Request.PayloadType, cliJSONPackageName(svc)+".UnmarshalJSON")
		} else if e.Payload.Ref != "" {
			example := e.Method.PayloadEx
			usesJSONBody := cli.IsJSONFlagType(e.Method.PayloadRef) && e.Payload.Request.ClientBody != nil
			if usesJSONBody {
				example = (cliExampleProjector{
					context:   ctx,
					attribute: e.valueTransport.Request.Body,
					service:   svcn,
					method:    en,
				}).retainedJSON(e.Payload.Request.ClientBody.Value)
			}
			flag := cli.NewFlagData(svcn, en, "p", e.Method.PayloadRef, e.Method.PayloadDesc, true, example, e.Method.PayloadDefault)
			if usesJSONBody && example == nil {
				flag.Example = ""
			}
			if containsBooleanMapKeys(e.Payload.Request.PayloadType) {
				flag.Unmarshal = "loomhttpcli.UnmarshalJSON"
			}
			flags = append(flags, flag)
		}
	}
	if e.Method.SkipRequestBodyEncodeDecode {
		flags = append(flags, streamFlag(svcn, en))
	}

	return flags, buildFunction
}

// makeFlags creates flag data and build function from endpoint arguments.
func makeFlags(e *EndpointData, args []*InitArgData, payload expr.DataType, jsonDecoder string) ([]*cli.FlagData, *cli.BuildFunctionData) {
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

		f := makeFlagData(e, arg, jsonDecoder)
		flags[i] = f
		params[i] = f.FullName
		if arg.FieldName == "" && arg.VarName != "body" && expr.IsObject(payload) {
			continue
		}
		var code *jen.Statement
		var chek bool
		if arg.IsTextUnmarshaler {
			code, chek = textFieldLoadCode(f, arg, e.Payload.Ref)
		} else {
			code, chek = cli.FieldLoadCode(f, arg.VarName, arg.TypeName, arg.Validate, arg.DefaultValue, payload, e.Payload.Ref)
		}
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

	pInit := payloadInitData(e, flagFullName(flags, args, "body"), pInitArgs)

	return flags, &cli.BuildFunctionData{
		Name:         "Build" + e.Method.VarName + "Payload",
		ActualParams: params,
		FormalParams: params,
		ServiceName:  e.ServiceName,
		MethodName:   e.Method.Name,
		ResultType:   e.Payload.Ref,
		Fields:       fdata,
		PayloadInit:  pInit,
		CheckErr:     check,
	}
}

func makeFlagData(e *EndpointData, arg *InitArgData, jsonDecoder string) *cli.FlagData {
	flagType := arg.TypeName
	if arg.IsTextUnmarshaler {
		flagType = "string"
	}
	flag := cli.NewFlagData(e.ServiceName, e.Method.Name, arg.VarName, flagType, arg.Description, arg.Required, arg.Example, arg.DefaultValue)
	if arg.VarName == "body" && flag.Type == "JSON" && arg.Example == nil {
		flag.Example = ""
	}
	if !arg.IsTextUnmarshaler && containsBooleanMapKeys(arg.Type) {
		flag.Unmarshal = jsonDecoder
	}
	return flag
}

// payloadInitData returns the data of the code of the CLI payload builder
// of e that initializes the payload from the flags. bodyFlag is the build
// function parameter holding the body flag, if any, and args the payload
// constructor arguments.
func payloadInitData(e *EndpointData, bodyFlag string, args []*codegen.InitArgData) *cli.PayloadInitData {
	init := e.Payload.Request.PayloadInit
	var initCode *jen.Statement
	if init.ClientCode != "" {
		initCode = codegen.Expr(init.ClientCode)
	}
	if !init.ReturnIsOptionalBody {
		// An empty flag leaves a union or a type with explicit presence nil
		// or absent by itself. The builder sets any other optional body
		// attribute only when its flag is set.
		bodyFlag = ""
	}
	return &cli.PayloadInitData{
		Code:                          initCode,
		ReturnTypeAttributeFlag:       bodyFlag,
		ReturnTypeAttribute:           init.ReturnTypeAttribute,
		ReturnTypeAttributePointer:    init.ReturnIsPrimitivePointer,
		ReturnTypeAttributeUnionValue: init.ReturnIsUnionValue,
		ReturnIsStruct:                init.ReturnIsStruct,
		ReturnTypeName:                init.ReturnTypeName,
		ReturnTypePkg:                 init.ReturnTypePkg,
		Args:                          args,
	}
}

// flagFullName returns the full name of the flag of the argument of args
// named varName, or the empty string if there is none. flags holds the flag
// of each argument.
func flagFullName(flags []*cli.FlagData, args []*InitArgData, varName string) string {
	for i, arg := range args {
		if arg.VarName == varName {
			return flags[i].FullName
		}
	}
	return ""
}

// streamFlag returns the flag used to specify the upload file for endpoints
// that use SkipRequestBodyEncodeDecode.
func streamFlag(svcn, en string) *cli.FlagData {
	return cli.NewFlagData(svcn, en, "stream", "string", "path to file containing the streamed request body", true, "loom.bin", nil)
}

// hostsTransportService reports whether svr hosts at least one of the
// services exposed by the transport expression transport. Client CLI files
// are generated only for such servers: the example client main does not call
// into the CLI package of a transport the server does not host.
func hostsTransportService(svr *expr.ServerExpr, transport *expr.HTTPExpr) bool {
	for _, name := range svr.Services {
		if transport.Service(name) != nil {
			return true
		}
	}
	return false
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
