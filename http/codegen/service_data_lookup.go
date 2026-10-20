package codegen

import (
	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// NewServicesData creates a new ServicesData instance for the given service data.
func NewServicesData(services *service.ServicesData, expressions *expr.HTTPExpr) *ServicesData {
	jsonrpc := services.Root.API.JSONRPC != nil && expressions == &services.Root.API.JSONRPC.HTTPExpr
	return &ServicesData{
		ServicesData:         services,
		Expressions:          expressions,
		HTTPData:             make(map[string]*ServiceData),
		serviceImportAliases: newServiceImportAliases(services.Root, expressions),
		jsonrpc:              jsonrpc,
	}
}

// Get retrieves the transport data for the service with the given name
// computing it if needed. It returns nil if there is no service with the given
// name.
func (sds *ServicesData) Get(name string) *ServiceData {
	if data, ok := sds.HTTPData[name]; ok {
		return data
	}
	svc := sds.Expressions.Service(name)
	if svc == nil {
		return nil
	}
	sds.HTTPData[name] = sds.analyze(svc)
	return sds.HTTPData[name]
}

// ServicesWithEndpoints returns the services whose analyzed transport data
// contains at least one method endpoint, preserving design order.
func (sds *ServicesData) ServicesWithEndpoints() []*expr.HTTPServiceExpr {
	var services []*expr.HTTPServiceExpr
	for _, service := range sds.Expressions.Services {
		data := sds.Get(service.Name())
		if data != nil && len(data.Endpoints) > 0 {
			services = append(services, service)
		}
	}
	return services
}

// servicesForGeneration returns the configured HTTP services, or the
// endpoint-bearing services when analyzing JSON-RPC transport data.
func (sds *ServicesData) servicesForGeneration() []*expr.HTTPServiceExpr {
	if !sds.jsonrpc {
		return sds.Expressions.Services
	}
	return sds.ServicesWithEndpoints()
}

// FileData returns the transport data of the service name that renders the
// code of a generated file that lists imports, and the imports of the file.
// The file imports the struct:pkg:path packages whose names clash with other
// imports under aliases, and the returned data qualifies their types with
// these aliases, see service.Data.FileImports. When no name clashes, FileData
// returns the data of Get and imports.
func (sds *ServicesData) FileData(name string, imports []*codegen.ImportSpec) (*ServiceData, []*codegen.ImportSpec) {
	names, imports := sds.Get(name).Service.FileImports(imports)
	services := sds.ServicesData.WithPackageNames(names)
	if services == sds.ServicesData {
		return sds.Get(name), imports
	}
	renamed, ok := sds.renamed[services]
	if !ok {
		renamed = &ServicesData{
			ServicesData:         services,
			Expressions:          sds.Expressions,
			HTTPData:             make(map[string]*ServiceData),
			serviceImportAliases: sds.serviceImportAliases,
			jsonrpc:              sds.jsonrpc,
		}
		if sds.renamed == nil {
			sds.renamed = make(map[*service.ServicesData]*ServicesData)
		}
		sds.renamed[services] = renamed
	}
	return renamed.Get(name), imports
}

// Endpoint returns the service method transport data for the endpoint with the
// given name, nil if there isn't one.
func (svc *ServiceData) Endpoint(name string) *EndpointData {
	for _, e := range svc.Endpoints {
		if e.Method.Name == name {
			return e
		}
	}
	return nil
}
