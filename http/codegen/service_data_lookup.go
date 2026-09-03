package codegen

import (
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

// NewServicesData creates a new ServicesData instance for the given service data.
func NewServicesData(services *service.ServicesData, expressions *expr.HTTPExpr) *ServicesData {
	return &ServicesData{
		ServicesData:         services,
		Expressions:          expressions,
		HTTPData:             make(map[string]*ServiceData),
		serviceImportAliases: newServiceImportAliases(expressions),
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
