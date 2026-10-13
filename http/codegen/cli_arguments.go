package codegen

import (
	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

type (
	// cliArgumentPlan owns dependency order for the parser and its example caller.
	// Entries retain semantic identities until each consumer allocates its names.
	cliArgumentPlan struct {
		services  []*ServiceData
		arguments []cliArgument
	}

	cliArgument struct {
		kind     cliArgumentKind
		service  *ServiceData
		endpoint *EndpointData
	}

	cliArgumentKind uint8
)

const (
	cliDialer cliArgumentKind = iota
	cliConfigurer
	cliMultipartEncoder
	cliInterceptors
)

func buildCLIArgumentPlan(server *expr.ServerExpr, services *ServicesData) *cliArgumentPlan {
	plan := &cliArgumentPlan{}
	for _, name := range server.Services {
		data := services.Get(name)
		if data != nil && len(data.Endpoints) > 0 {
			plan.services = append(plan.services, data)
		}
	}
	for _, data := range plan.services {
		if HasWebSocket(data) {
			if len(plan.arguments) == 0 {
				plan.arguments = append(plan.arguments, cliArgument{kind: cliDialer})
			}
			plan.arguments = append(plan.arguments, cliArgument{kind: cliConfigurer, service: data})
		}
	}
	for _, data := range plan.services {
		for _, endpoint := range data.Endpoints {
			if endpoint.MultipartRequestEncoder != nil {
				plan.arguments = append(plan.arguments, cliArgument{
					kind:     cliMultipartEncoder,
					service:  data,
					endpoint: endpoint,
				})
			}
		}
		if len(data.Service.ClientInterceptors) > 0 {
			plan.arguments = append(plan.arguments, cliArgument{kind: cliInterceptors, service: data})
		}
	}
	return plan
}

func appendCLIArgumentParameters(
	group *jen.Group,
	plan *cliArgumentPlan,
	commands []*commandData,
	transport ClientCLITransport,
) {
	byService := make(map[*ServiceData]*commandData, len(commands))
	byEndpoint := make(map[*EndpointData]*subcommandData)
	for _, command := range commands {
		byService[command.source] = command
		for _, subcommand := range command.Subcommands {
			byEndpoint[subcommand.source] = subcommand
		}
	}
	for _, argument := range plan.arguments {
		command := byService[argument.service]
		switch argument.kind {
		case cliDialer:
			group.Id("dialer").Add(codegen.TypeRef("loomhttp.Dialer"))
		case cliConfigurer:
			group.Id(transport.StreamingConfigurerName(command.VarName)).
				Add(codegen.TypeRef(transport.StreamingConfigurerType(command.PkgName)))
		case cliMultipartEncoder:
			subcommand := byEndpoint[argument.endpoint]
			group.Id(subcommand.MultipartVarName).Add(codegen.TypeRef(command.PkgName + "." + subcommand.MultipartFuncName))
		case cliInterceptors:
			group.Id(command.Interceptors.VarName).Add(codegen.TypeRef(command.Interceptors.PkgName + ".ClientInterceptors"))
		}
	}
}

func (plan *cliArgumentPlan) needsDialer() bool {
	return len(plan.arguments) > 0 && plan.arguments[0].kind == cliDialer
}
