package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// analyze creates the data necessary to render the code of the given service.
// It records the user types needed by the service definition in userTypes.
func (sds *ServicesData) analyze(httpSvc *expr.HTTPServiceExpr) *ServiceData {
	svc := sds.ServicesData.Get(httpSvc.ServiceExpr.Name)
	irService := transportir.BuildService(httpSvc)
	scope := newHTTPAnalysisScope(svc)
	sd := newHTTPServiceData(svc, scope)
	sd.FileServers = sds.buildFileServersData(httpSvc, scope)
	for _, httpEndpoint := range irService.Endpoints {
		sd.Endpoints = append(sd.Endpoints, sds.buildEndpointDataFromIR(httpEndpoint, svc, sd, scope))
	}
	for _, endpointIR := range irService.Endpoints {
		sds.collectEndpointBodyAttributeTypes(endpointIR, sd)
	}
	sd.UnionTypes = sds.collectEndpointUnionTypes(httpSvc, sd.Scope)

	return sd
}

func newHTTPAnalysisScope(svc *service.Data) *codegen.NameScope {
	scope := codegen.NewNameScope()
	scope.Unique("c") // 'c' is reserved as the client's receiver name.
	scope.Unique("v") // 'v' is reserved as the request builder payload argument name.
	scope.Unique("websocket")
	scope.Unique(svc.PkgName)
	return scope
}

func newHTTPServiceData(svc *service.Data, scope *codegen.NameScope) *ServiceData {
	return &ServiceData{
		Service:          svc,
		ServerStruct:     "Server",
		MountPointStruct: "MountPoint",
		ServerInit:       "New",
		MountServer:      "Mount",
		ServerService:    "Service",
		ClientStruct:     "Client",
		ServerTypeNames:  make(map[string]bool),
		ClientTypeNames:  make(map[string]bool),
		Scope:            scope,
	}
}

func (sds *ServicesData) buildFileServersData(httpSvc *expr.HTTPServiceExpr, scope *codegen.NameScope) []*FileServerData {
	fileServers := make([]*FileServerData, 0, len(httpSvc.FileServers))
	for _, server := range httpSvc.FileServers {
		paths := make([]string, len(server.RequestPaths))
		for i, path := range server.RequestPaths {
			idx := strings.LastIndex(path, "/{")
			switch {
			case idx == 0:
				paths[i] = "/"
			case idx > 0:
				paths[i] = path[:idx]
			default:
				paths[i] = path
			}
		}
		var pathParam string
		if server.IsDir() {
			pathParam = expr.ExtractHTTPWildcards(server.RequestPaths[0])[0]
		}
		fileServers = append(fileServers, &FileServerData{
			MountHandler: scope.Unique(fmt.Sprintf("Mount%s", codegen.Goify(server.FilePath, true))),
			RequestPaths: paths,
			FilePath:     server.FilePath,
			IsDir:        server.IsDir(),
			PathParam:    pathParam,
			Redirect:     httpRedirectData(server.Redirect),
			VarName:      scope.Unique(codegen.Goify(server.FilePath, true)),
			ArgName:      scope.Unique(fmt.Sprintf("fileSystem%s", codegen.Goify(server.FilePath, true))),
		})
	}
	return fileServers
}

func (sds *ServicesData) buildEndpointDataFromIR(endpointIR *transportir.Endpoint, svc *service.Data, sd *ServiceData, scope *codegen.NameScope) *EndpointData {
	method := svc.Method(endpointIR.MethodName)
	routes := sds.buildEndpointRoutes(endpointIR, method, svc, sd)
	payload := sds.buildPayloadDataFromIR(endpointIR, sd)
	reqs, hsch, bosch, qsch, basch := sds.buildRequirementSchemes(endpointIR)
	requestInit := sds.buildClientRequestInit(endpointIR, method, svc, routes)

	endpoint := &EndpointData{
		Method:          method,
		ServiceName:     svc.Name,
		ServiceVarName:  svc.VarName,
		ServicePkgName:  svc.PkgName,
		Payload:         payload,
		Result:          sds.buildResultDataFromIR(endpointIR, sd),
		Errors:          sds.buildErrorsDataFromIR(endpointIR, sd),
		HeaderSchemes:   hsch,
		BodySchemes:     bosch,
		QuerySchemes:    qsch,
		BasicScheme:     basch,
		Routes:          routes,
		MountHandler:    fmt.Sprintf("Mount%sHandler", method.VarName),
		HandlerInit:     fmt.Sprintf("New%sHandler", method.VarName),
		RequestDecoder:  fmt.Sprintf("Decode%sRequest", method.VarName),
		ResponseEncoder: fmt.Sprintf("Encode%sResponse", method.VarName),
		ErrorEncoder:    fmt.Sprintf("Encode%sError", method.VarName),
		ClientStruct:    "Client",
		EndpointInit:    method.VarName,
		RequestInit:     requestInit,
		HasMixedResults: endpointIR.Response.HasMixedResults,
		RequestEncoder:  endpointRequestEncoderName(method, payload, basch),
		ResponseDecoder: fmt.Sprintf("Decode%sResponse", method.VarName),
		Requirements:    reqs,
	}
	sds.applyStreamingEndpointData(endpoint, endpointIR, sd)
	sds.applyMultipartEndpointData(endpoint, endpointIR, method, svc, scope)
	endpoint.Redirect = transportRedirectData(endpointIR.Redirect)
	return endpoint
}

func (sds *ServicesData) applyStreamingEndpointData(endpoint *EndpointData, endpointIR *transportir.Endpoint, sd *ServiceData) {
	if !endpointIR.Stream.IsStreaming {
		return
	}
	sds.initWebSocketData(endpoint, endpointIR, sd)
	initSSEData(endpoint, endpointIR, sd)
}

func (sds *ServicesData) applyMultipartEndpointData(endpoint *EndpointData, endpointIR *transportir.Endpoint, method *service.MethodData, svc *service.Data, scope *codegen.NameScope) {
	sds.initEndpointMultipartData(endpoint, endpointIR, method, svc)
	if endpointIR.Request.SkipBodyEncode {
		endpoint.BuildStreamPayload = scope.Unique("Build" + codegen.Goify(method.Name, true) + "StreamPayload")
	}
}

func httpRedirectData(redirect *expr.HTTPRedirectExpr) *RedirectData {
	if redirect == nil {
		return nil
	}
	return &RedirectData{
		URL:        redirect.URL,
		StatusCode: statusCodeToHTTPConst(redirect.StatusCode),
	}
}

func transportRedirectData(redirect *transportir.Redirect) *RedirectData {
	if redirect == nil {
		return nil
	}
	return &RedirectData{
		URL:        redirect.URL,
		StatusCode: statusCodeToHTTPConst(redirect.StatusCode),
	}
}
