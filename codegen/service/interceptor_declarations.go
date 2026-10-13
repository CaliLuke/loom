package service

type (
	// interceptorDeclarations assigns each shared Info and its complete accessor
	// set to one output file. Invocation records remain local to each side.
	interceptorDeclarations struct {
		server []*InterceptorData
		client []*InterceptorData
	}
)

// buildInterceptorDeclarations covers both invocation sides without changing
// their method lists or allocating new accessor identities. Shared signatures have
// already been admitted across both sides; physical bindings remain method-local.
func buildInterceptorDeclarations(methods []*MethodData, server, client []*InterceptorData) interceptorDeclarations {
	var result interceptorDeclarations
	declarations := make(map[string]*InterceptorData)
	coverage := make(map[*InterceptorData]map[string]*MethodInterceptorData)
	for side, interceptors := range [][]*InterceptorData{server, client} {
		for _, interceptor := range interceptors {
			declaration := declarations[interceptor.DesignName]
			if declaration == nil {
				shared := *interceptor
				shared.Methods = nil
				declaration = &shared
				declarations[interceptor.DesignName] = declaration
				coverage[declaration] = make(map[string]*MethodInterceptorData)
				if side == 0 {
					result.server = append(result.server, declaration)
				} else {
					result.client = append(result.client, declaration)
				}
			}
			for _, method := range interceptor.Methods {
				if coverage[declaration][method.MethodName] == nil {
					coverage[declaration][method.MethodName] = method
				}
			}
		}
	}
	for declaration, covered := range coverage {
		for _, method := range methods {
			if accessor := covered[method.VarName]; accessor != nil {
				declaration.Methods = append(declaration.Methods, accessor)
			}
		}
	}
	return result
}
