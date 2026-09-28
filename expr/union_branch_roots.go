package expr

import "strconv"

type unionBranchRootCollector struct {
	roots []unionBranchRoot
}

func (r *RootExpr) unionBranchRoots(anchors map[UserType]string) []unionBranchRoot {
	collector := &unionBranchRootCollector{}
	for _, typ := range r.Types {
		path := unionBranchPath("0", "type", typ.Name())
		anchors[typ] = path
		collector.add(path, &AttributeExpr{Type: typ})
	}
	for _, typ := range r.ResultTypes {
		path := unionBranchPath("0", "result", typ.Identifier)
		anchors[typ] = path
		collector.add(path, &AttributeExpr{Type: typ})
	}
	collector.errors("1", r.Errors)
	for _, service := range r.Services {
		path := unionBranchPath("2", "service", service.Name)
		collector.errors(path, service.Errors)
		for _, method := range service.Methods {
			methodPath := unionBranchPath(path, "method", method.Name)
			collector.add(methodPath+"/payload", method.Payload)
			collector.add(methodPath+"/result", method.Result)
			collector.add(methodPath+"/streaming-payload", method.StreamingPayload)
			collector.add(methodPath+"/streaming-result", method.StreamingResult)
			collector.errors(methodPath, method.Errors)
		}
	}
	for _, interceptor := range r.Interceptors {
		path := unionBranchPath("3", "interceptor", interceptor.Name)
		collector.add(path+"/read-payload", interceptor.ReadPayload)
		collector.add(path+"/write-payload", interceptor.WritePayload)
		collector.add(path+"/read-result", interceptor.ReadResult)
		collector.add(path+"/write-result", interceptor.WriteResult)
		collector.add(path+"/read-streaming-payload", interceptor.ReadStreamingPayload)
		collector.add(path+"/write-streaming-payload", interceptor.WriteStreamingPayload)
		collector.add(path+"/read-streaming-result", interceptor.ReadStreamingResult)
		collector.add(path+"/write-streaming-result", interceptor.WriteStreamingResult)
	}
	if r.API != nil {
		collector.http("4/http", r.API.HTTP)
		if r.API.JSONRPC != nil {
			collector.http("4/jsonrpc", &r.API.JSONRPC.HTTPExpr)
		}
		collector.grpc("4/grpc", r.API.GRPC)
	}
	return collector.roots
}

func (c *unionBranchRootCollector) add(path string, attribute *AttributeExpr) {
	if attribute != nil {
		c.roots = append(c.roots, unionBranchRoot{path: path, attribute: attribute})
	}
}

func (c *unionBranchRootCollector) mapped(path string, attribute *MappedAttributeExpr) {
	if attribute != nil {
		c.add(path, attribute.AttributeExpr)
	}
}

func (c *unionBranchRootCollector) errors(path string, errors []*ErrorExpr) {
	for _, err := range errors {
		c.add(unionBranchPath(path, "error", err.Name), err.AttributeExpr)
	}
}

func (c *unionBranchRootCollector) http(path string, root *HTTPExpr) {
	if root == nil {
		return
	}
	c.mapped(path+"/params", root.Params)
	c.mapped(path+"/headers", root.Headers)
	c.mapped(path+"/cookies", root.Cookies)
	c.httpErrors(path, root.Errors)
	for _, service := range root.Services {
		servicePath := unionBranchPath(path, "service", service.Name())
		c.mapped(servicePath+"/params", service.Params)
		c.mapped(servicePath+"/headers", service.Headers)
		c.mapped(servicePath+"/cookies", service.Cookies)
		c.httpErrors(servicePath, service.HTTPErrors)
		for _, endpoint := range service.HTTPEndpoints {
			endpointPath := unionBranchPath(servicePath, "method", endpoint.Name())
			c.mapped(endpointPath+"/params", endpoint.Params)
			c.mapped(endpointPath+"/headers", endpoint.Headers)
			c.mapped(endpointPath+"/cookies", endpoint.Cookies)
			c.add(endpointPath+"/body", endpoint.Body)
			c.add(endpointPath+"/openapi-body", endpoint.OpenAPIRequestBody)
			c.add(endpointPath+"/streaming-body", endpoint.StreamingBody)
			for i, response := range endpoint.Responses {
				c.httpResponse(unionBranchPath(endpointPath, "response", strconv.Itoa(i)), response)
			}
			c.httpErrors(endpointPath, endpoint.HTTPErrors)
		}
	}
}

func (c *unionBranchRootCollector) httpErrors(path string, errors []*HTTPErrorExpr) {
	for _, err := range errors {
		c.httpResponse(unionBranchPath(path, "error", err.Name), err.Response)
	}
}

func (c *unionBranchRootCollector) httpResponse(path string, response *HTTPResponseExpr) {
	if response == nil {
		return
	}
	c.mapped(path+"/headers", response.Headers)
	c.add(path+"/body", response.Body)
	c.add(path+"/openapi-body", response.OpenAPIBody)
	for i, cookie := range response.Cookies {
		c.mapped(unionBranchPath(path, "cookie", strconv.Itoa(i)), cookie.MappedAttributeExpr)
	}
}

func (c *unionBranchRootCollector) grpc(path string, root *GRPCExpr) {
	if root == nil {
		return
	}
	c.grpcErrors(path, root.Errors)
	for _, service := range root.Services {
		servicePath := unionBranchPath(path, "service", service.Name())
		c.grpcErrors(servicePath, service.GRPCErrors)
		for _, endpoint := range service.GRPCEndpoints {
			endpointPath := unionBranchPath(servicePath, "method", endpoint.Name())
			c.add(endpointPath+"/request", endpoint.Request)
			c.add(endpointPath+"/streaming-request", endpoint.StreamingRequest)
			c.mapped(endpointPath+"/metadata", endpoint.Metadata)
			c.grpcResponse(endpointPath+"/response", endpoint.Response)
			c.grpcErrors(endpointPath, endpoint.GRPCErrors)
		}
	}
}

func (c *unionBranchRootCollector) grpcErrors(path string, errors []*GRPCErrorExpr) {
	for _, err := range errors {
		c.grpcResponse(unionBranchPath(path, "error", err.Name), err.Response)
	}
}

func (c *unionBranchRootCollector) grpcResponse(path string, response *GRPCResponseExpr) {
	if response == nil {
		return
	}
	c.add(path+"/message", response.Message)
	c.mapped(path+"/headers", response.Headers)
	c.mapped(path+"/trailers", response.Trailers)
}
