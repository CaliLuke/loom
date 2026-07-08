package codegen

import (
	"fmt"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"

	gocodegen "github.com/CaliLuke/loom/codegen"
)

const clientOperationGroupThreshold = 6

type (
	clientOperationGroup struct {
		Name      string
		FieldName string
		Endpoints []*EndpointData
	}
)

func clientOperationGroups(data *ServiceData) []clientOperationGroup {
	if len(data.Endpoints) <= clientOperationGroupThreshold {
		return nil
	}
	byName := make(map[string][]*EndpointData)
	names := make([]string, 0)
	reserved := clientOperationGroupReservedFields(data)
	for _, endpoint := range data.Endpoints {
		name := clientOperationGroupName(endpoint)
		if _, ok := reserved[name]; ok {
			name += "Operations"
		}
		if _, ok := byName[name]; !ok {
			names = append(names, name)
		}
		byName[name] = append(byName[name], endpoint)
	}
	sort.Strings(names)
	groups := make([]clientOperationGroup, 0, len(names))
	for _, name := range names {
		groups = append(groups, clientOperationGroup{
			Name:      name + "Client",
			FieldName: name,
			Endpoints: byName[name],
		})
	}
	return groups
}

func clientOperationGroupReservedFields(data *ServiceData) map[string]struct{} {
	reserved := map[string]struct{}{
		"RestoreResponseBody": {},
		"scheme":              {},
		"host":                {},
		"encoder":             {},
		"decoder":             {},
		"dialer":              {},
		"configurer":          {},
	}
	for _, endpoint := range data.Endpoints {
		reserved[endpoint.Method.VarName+"Doer"] = struct{}{}
	}
	return reserved
}

func clientOperationGroupName(endpoint *EndpointData) string {
	if len(endpoint.Routes) == 0 {
		return endpoint.Method.VarName
	}
	path := strings.Trim(endpoint.Routes[0].Path, "/")
	if path == "" {
		return "Root"
	}
	for _, segment := range strings.Split(path, "/") {
		if segment == "" || strings.HasPrefix(segment, "{") {
			continue
		}
		return gocodegen.Goify(segment, true)
	}
	return "Root"
}

func clientOperationGroupSection(data *ServiceData) gocodegen.Section {
	return gocodegen.NewJenniferSection("client-operation-groups", func(stmt *jen.Statement) {
		for _, group := range clientOperationGroups(data) {
			gocodegen.Doc(stmt, fmt.Sprintf("%s groups %s HTTP client operations.", group.Name, group.FieldName))
			stmt.Type().Id(group.Name).Struct(
				jen.Id("client").Op("*").Id(data.ClientStruct),
			)
			stmt.Line()
			for _, endpoint := range group.Endpoints {
				addClientGroupEndpointMethod(stmt, group.Name, endpoint)
				if endpoint.HasMixedResults {
					stream := *endpoint
					stream.EndpointInit = endpoint.EndpointInit + "Stream"
					addClientGroupEndpointMethod(stmt, group.Name, &stream)
				}
				addClientGroupRequestMethod(stmt, group.Name, endpoint)
			}
		}
	})
}

func addClientGroupEndpointMethod(stmt *jen.Statement, groupName string, endpoint *EndpointData) {
	gocodegen.Doc(stmt, fmt.Sprintf("%s returns an endpoint that makes HTTP requests to the %s service %s server.", endpoint.EndpointInit, endpoint.ServiceName, endpoint.Method.Name))
	fn := stmt.Func().Params(jen.Id("g").Op("*").Id(groupName)).Id(endpoint.EndpointInit)
	if endpoint.MultipartRequestEncoder != nil {
		fn.Params(jen.Id(endpoint.MultipartRequestEncoder.VarName).Id(endpoint.MultipartRequestEncoder.FuncName))
	} else {
		fn.Params()
	}
	fn.Id("loom").Dot("Endpoint").BlockFunc(func(body *jen.Group) {
		body.Return(jen.Id("g").Dot("client").Dot(endpoint.EndpointInit).CallFunc(func(args *jen.Group) {
			if endpoint.MultipartRequestEncoder != nil {
				args.Id(endpoint.MultipartRequestEncoder.VarName)
			}
		}))
	})
	stmt.Line()
}

func addClientGroupRequestMethod(stmt *jen.Statement, groupName string, endpoint *EndpointData) {
	init := endpoint.RequestInit
	gocodegen.Doc(stmt, init.Description)
	fn := stmt.Func().
		Params(jen.Id("g").Op("*").Id(groupName)).
		Id(init.Name).
		ParamsFunc(func(params *jen.Group) {
			params.Id("ctx").Qual("context", "Context")
			for _, arg := range init.ClientArgs {
				params.Id(arg.VarName).Add(gocodegen.TypeRef(arg.TypeRef))
			}
		}).
		Params(jen.Op("*").Qual("net/http", "Request"), jen.Error())
	fn.BlockFunc(func(body *jen.Group) {
		body.Return(jen.Id("g").Dot("client").Dot(init.Name).CallFunc(func(args *jen.Group) {
			args.Id("ctx")
			for _, arg := range init.ClientArgs {
				args.Id(arg.VarName)
			}
		}))
	})
	stmt.Line()
}
