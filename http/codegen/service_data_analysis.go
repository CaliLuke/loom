package codegen

import (
	"fmt"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// transportGeneratedImportNames is the union of local names from literal
// ImportSpec and Loom import constructors in generated files that also emit a
// dynamic transport import. TestTransportImportAliasReservationsCoverGeneratedImports
// keeps it aligned with those constructors. Service, view, and user-type
// imports are dynamic design data and therefore allocate their own aliases.
var transportGeneratedImportNames = []string{
	"atomic",
	"bufio",
	"bytes",
	"context",
	"debug",
	"errors",
	"flag",
	"fmt",
	"http",
	"io",
	"json",
	"jsonrpc",
	"log",
	"loom",
	"loomhttp",
	"loomhttpcli",
	"loomtransport",
	"middleware",
	"multipart",
	"os",
	"path",
	"strconv",
	"strings",
	"sync",
	"testing",
	"time",
	"url",
	"utf8",
	"websocket",
}

// transportGeneratedLocalNames are local identifiers that coexist with
// service package references in generated transport functions.
var transportGeneratedLocalNames = []string{
	"body",
	"c",
	"ctx",
	"err",
	"v",
}

// analyze creates the data necessary to render the code of the given service.
// It records the user types needed by the service definition in userTypes.
// Panics are wrapped with DSL attribution (service, endpoint, source location)
// and re-panicked so opaque failures surface with navigable context.
func (sds *ServicesData) analyze(httpSvc *expr.HTTPServiceExpr) (sd *ServiceData) {
	svc := sds.ServicesData.Get(httpSvc.ServiceExpr.Name)
	ctx := sds.ServicesData.Ctx.WithService(httpSvc.ServiceExpr)
	defer func() {
		if err := codegen.RecoverPanic(recover()); err != nil {
			panic(codegen.NewError(ctx, nil, err))
		}
	}()
	ctx.Debug("analyzing HTTP service",
		"endpoints", len(httpSvc.HTTPEndpoints),
		"file_servers", len(httpSvc.FileServers),
	)
	irService := transportir.BuildService(httpSvc)
	svc, scope := newHTTPAnalysisService(svc, sds.serviceImportAliases[httpSvc.Name()])
	sd = newHTTPServiceData(svc, scope)
	sd.CORS = buildCORSData(httpSvc)
	sd.FileServers = sds.buildFileServersData(httpSvc, scope)
	for _, httpEndpoint := range irService.Endpoints {
		epCtx := ctx.WithMethod(httpSvc.ServiceExpr.Method(httpEndpoint.MethodName))
		epCtx.Debug("analyzing HTTP endpoint",
			"verb", endpointVerb(httpEndpoint),
			"path", endpointPath(httpEndpoint),
			"has_body", httpEndpoint.Request != nil && httpEndpoint.Request.Body != nil && httpEndpoint.Request.Body.Type != expr.Empty,
			"path_params", len(httpEndpoint.Request.PathParams),
			"query_params", len(httpEndpoint.Request.QueryParams),
			"error_responses", len(httpEndpoint.Response.ErrorResponses),
		)
		sd.Endpoints = append(sd.Endpoints, sds.buildEndpointDataWithContext(epCtx, httpEndpoint, svc, sd, scope))
	}
	for _, endpointIR := range irService.Endpoints {
		sds.collectEndpointBodyAttributeTypes(endpointIR, sd)
	}
	sd.UnionTypes = sds.collectEndpointUnionTypes(httpSvc, sd.Scope)

	return sd
}

func buildCORSData(httpSvc *expr.HTTPServiceExpr) *CORSData {
	cors := httpSvc.CORS
	if cors == nil {
		cors = httpSvc.Root.CORS
	}
	if cors == nil {
		return nil
	}
	out := &CORSData{Origins: make([]*CORSOriginData, 0, len(cors.Origins)), Runtime: cors.Runtime}
	for _, origin := range cors.Origins {
		out.Origins = append(out.Origins, &CORSOriginData{
			Pattern:     origin.Pattern,
			Regex:       origin.Regex,
			Methods:     append([]string(nil), origin.Methods...),
			Headers:     append([]string(nil), origin.Headers...),
			Expose:      append([]string(nil), origin.Expose...),
			MaxAge:      origin.MaxAge,
			Credentials: origin.Credentials,
		})
	}
	return out
}

func (sds *ServicesData) buildEndpointDataWithContext(
	ctx *codegen.Context,
	endpointIR *transportir.Endpoint,
	svc *service.Data,
	sd *ServiceData,
	scope *codegen.NameScope,
) (endpoint *EndpointData) {
	defer func() {
		if err := codegen.RecoverPanic(recover()); err != nil {
			panic(codegen.NewError(ctx, nil, err))
		}
	}()
	return sds.buildEndpointDataFromIR(endpointIR, svc, sd, scope)
}

func endpointVerb(ep *transportir.Endpoint) string {
	if ep == nil || len(ep.Routes) == 0 {
		return ""
	}
	return ep.Routes[0].Method
}

func endpointPath(ep *transportir.Endpoint) string {
	if ep == nil || len(ep.Routes) == 0 {
		return ""
	}
	return ep.Routes[0].Path
}

func newHTTPAnalysisService(svc *service.Data, importAlias string) (*service.Data, *codegen.NameScope) {
	scope := codegen.NewNameScope()
	scope.Unique("c") // 'c' is reserved as the client's receiver name.
	scope.Unique("v") // 'v' is reserved as the request builder payload argument name.
	scope.Unique("websocket")

	httpSvc := *svc
	httpSvc.PkgName = importAlias
	scope.Unique(httpSvc.PkgName)
	return &httpSvc, scope
}

func newServiceImportAliases(expressions *expr.HTTPExpr) map[string]string {
	scope := codegen.NewNameScope()
	for _, name := range transportGeneratedImportNames {
		scope.Unique(name)
	}
	for _, name := range transportGeneratedLocalNames {
		scope.Unique(name)
	}

	aliases := make(map[string]string, len(expressions.Services))
	for _, httpSvc := range expressions.Services {
		aliases[httpSvc.Name()] = scope.Unique(httpServicePackageName(httpSvc.ServiceExpr), "svc")
	}
	return aliases
}

func httpServicePackageName(svc *expr.ServiceExpr) string {
	scope := codegen.NewNameScope()
	scope.Unique("Use")
	scope.Unique("websocket")
	return scope.HashedUnique(svc, strings.ToLower(codegen.Goify(svc.Name, false)), "svc")
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
			prefix, _, ok := strings.CutLast(path, "/{")
			switch {
			case ok && prefix == "":
				paths[i] = "/"
			case ok:
				paths[i] = prefix
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
	responseContractCases, responseContractWarnings := buildResponseContractCaseData(endpointIR)

	endpoint := &EndpointData{
		Method:                    method,
		ServiceName:               svc.Name,
		ServiceVarName:            svc.VarName,
		ServicePkgName:            svc.PkgName,
		Payload:                   payload,
		Result:                    sds.buildResultDataFromIR(endpointIR, sd),
		Errors:                    sds.buildErrorsDataFromIR(endpointIR, sd),
		HeaderSchemes:             hsch,
		BodySchemes:               bosch,
		QuerySchemes:              qsch,
		BasicScheme:               basch,
		Routes:                    routes,
		MountHandler:              fmt.Sprintf("Mount%sHandler", method.VarName),
		HandlerInit:               fmt.Sprintf("New%sHandler", method.VarName),
		RequestDecoder:            fmt.Sprintf("Decode%sRequest", method.VarName),
		ResponseEncoder:           fmt.Sprintf("Encode%sResponse", method.VarName),
		ErrorEncoder:              fmt.Sprintf("Encode%sError", method.VarName),
		ClientStruct:              "Client",
		EndpointInit:              method.VarName,
		RequestInit:               requestInit,
		HasMixedResults:           endpointIR.Response.HasMixedResults,
		ResponseContractCasesInit: fmt.Sprintf("%sResponseContractCases", method.VarName),
		ResponseContractCases:     responseContractCases,
		ResponseContractWarnings:  responseContractWarnings,
		RequestEncoder:            endpointRequestEncoderName(method, payload, basch),
		ResponseDecoder:           fmt.Sprintf("Decode%sResponse", method.VarName),
		Requirements:              reqs,
		CORS:                      sd.CORS,
	}
	sds.applyStreamingEndpointData(endpoint, endpointIR, sd)
	sds.applyMultipartEndpointData(endpoint, endpointIR, method, svc, scope)
	endpoint.Redirect = transportRedirectData(endpointIR.Redirect)
	return endpoint
}

func buildResponseContractCaseData(endpoint *transportir.Endpoint) ([]*ResponseContractCaseData, []string) {
	analysis := transportir.AnalyzeResponseContractCases(endpoint)
	if !analysis.Supported() {
		return nil, responseContractLimitationWarnings(endpoint, analysis.Limitations)
	}

	cases := make([]*ResponseContractCaseData, 0, len(analysis.Cases))
	for _, contractCase := range analysis.Cases {
		data := &ResponseContractCaseData{
			ID:         contractCase.ID,
			IsError:    contractCase.Kind == transportir.ResponseContractError,
			Transport:  string(contractCase.Transport),
			StatusCode: contractCase.StatusCode,
			ErrorName:  contractCase.ErrorName,
		}
		if contractCase.SSE != nil {
			data.SSE = &SSEResponseContractData{
				Direction:         contractCase.SSE.Direction,
				MessageType:       contractCase.SSE.MessageType,
				DataField:         contractCase.SSE.DataField,
				DataEncoding:      contractCase.SSE.DataEncoding,
				IDField:           contractCase.SSE.IDField,
				EventField:        contractCase.SSE.EventField,
				RetryField:        contractCase.SSE.RetryField,
				IDRequired:        contractCase.SSE.IDRequired,
				EventTypeRequired: contractCase.SSE.EventTypeRequired,
				EventTypes:        append([]string(nil), contractCase.SSE.EventTypes...),
				Terminal:          contractCase.SSE.Terminal,
			}
		}
		if contractCase.HasBody {
			data.ContentTypes = append([]string(nil), contractCase.ContentTypes...)
		}
		for _, header := range contractCase.Headers {
			if header.Required {
				data.RequiredHeaders = append(data.RequiredHeaders, header.HTTPName)
			}
		}
		for _, cookie := range contractCase.Cookies {
			if cookie.Required {
				data.RequiredCookies = append(data.RequiredCookies, cookie.HTTPName)
			}
		}
		cases = append(cases, data)
	}
	return cases, nil
}

func responseContractLimitationWarnings(endpoint *transportir.Endpoint, limitations []transportir.ResponseContractLimitation) []string {
	if endpoint == nil || endpoint.Service == nil || len(limitations) == 0 {
		return nil
	}

	prefix := fmt.Sprintf("response contract omitted for %s.%s", endpoint.Service.Name, endpoint.MethodName)
	warnings := make([]string, 0, len(limitations))
	for _, limitation := range limitations {
		warnings = append(warnings, fmt.Sprintf("%s: %s: %s", prefix, limitation.Code, limitation.Detail))
	}
	return warnings
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
