package service

import (
	"path/filepath"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
)

// InterceptorsFiles returns the interceptors files for the given service.
func InterceptorsFiles(genpkg string, service *expr.ServiceExpr, services *ServicesData) []*codegen.File {
	var files []*codegen.File
	svc := services.Get(service.Name)

	// Generate service-specific interceptor files
	if len(svc.ServerInterceptors) > 0 {
		files = append(files, interceptorFile(services, svc, true))
	}
	if len(svc.ClientInterceptors) > 0 {
		files = append(files, interceptorFile(services, svc, false))
	}

	// Generate wrapper file if this service has any interceptors
	if len(svc.ServerInterceptors) > 0 || len(svc.ClientInterceptors) > 0 {
		files = append(files, wrapperFile(genpkg, services, svc))
	}

	return files
}

// interceptorFile returns the file defining the interceptors.
// This method is called twice, once for the server and once for the client.
func interceptorFile(services *ServicesData, svc *Data, server bool) *codegen.File {
	svc, imports := services.fileData(svc.Name, append([]*codegen.ImportSpec{
		{Path: "context"},
		codegen.LoomImport(""),
	}, svc.UserTypeImports...))
	filename := "client_interceptors.go"
	desc := "Client Interceptors"
	if server {
		filename = "service_interceptors.go"
		desc = "Server Interceptors"
	}
	desc = svc.Name + desc
	path := filepath.Join(codegen.Gendir, svc.PathName, filename)

	interceptors := svc.interceptorDeclarations.server
	if !server {
		interceptors = svc.interceptorDeclarations.client
	}

	sections := []codegen.Section{codegen.Header(desc, svc.PkgName, imports)}
	if server {
		sections = append(sections, serverInterceptorsInterfaceSection(svc))
	} else {
		sections = append(sections, clientInterceptorsInterfaceSection(svc))
	}
	if len(interceptors) > 0 {
		sections = append(sections, interceptorTypesSection(interceptors))
	}
	sections = append(sections, endpointInterceptionSections(svc, server)...)

	if len(interceptors) > 0 {
		sections = append(sections, interceptorsSection(interceptors, server))
	}

	return &codegen.File{Path: path, Sections: sections}
}

// endpointInterceptionSections renders the allocated chains for each invocation
// boundary without repeating applicability or naming decisions.
func endpointInterceptionSections(svc *Data, server bool) []codegen.Section {
	var sections []codegen.Section
	for _, m := range svc.Methods {
		if server {
			if len(m.ServerInterceptors) == 0 {
				continue
			}
			wrappers := make([]string, len(m.ServerInterceptors))
			for i, name := range m.ServerInterceptors {
				wrappers[i] = m.symbols.interceptors[name].server
			}
			sections = append(sections, endpointWrapperSection(true, m, m.ServerInterceptorWrapper, wrappers))
			continue
		}
		for _, plan := range m.clientInterception.endpoints() {
			if plan.wrapper == "" {
				continue
			}
			wrappers := make([]string, len(plan.bindings))
			for i, binding := range plan.bindings {
				wrappers[i] = binding.wrapper
			}
			sections = append(sections, endpointWrapperSection(false, m, plan.wrapper, wrappers))
		}
	}

	return sections
}

// wrapperFile returns the file containing the interceptor wrappers.
func wrapperFile(genpkg string, services *ServicesData, svc *Data) *codegen.File {
	path := filepath.Join(codegen.Gendir, svc.PathName, "interceptor_wrappers.go")

	svc, imports := services.fileData(svc.Name, append([]*codegen.ImportSpec{
		{Path: "context"},
		{Path: "fmt"},
		codegen.LoomImport(""),
		{Path: genpkg + "/" + svc.PathName + "/views", Name: svc.ViewsPkg},
	}, svc.UserTypeImports...))
	var sections []codegen.Section
	sections = append(sections, codegen.Header("Interceptor wrappers", svc.PkgName, imports))

	// Generate any interceptor stream wrapper struct types first
	var wrappedServerStreams, wrappedClientStreams []*StreamInterceptorData
	if len(svc.ServerInterceptors) > 0 {
		wrappedServerStreams = collectWrappedStreams(svc.ServerInterceptors, true)
		if len(wrappedServerStreams) > 0 {
			sections = append(sections, streamWrapperTypesSection("server-interceptor-stream-wrapper-types", wrappedServerStreams, true))
		}
	}
	if len(svc.ClientInterceptors) > 0 {
		wrappedClientStreams = collectWrappedStreams(svc.ClientInterceptors, false)
		if len(wrappedClientStreams) > 0 {
			sections = append(sections, streamWrapperTypesSection("client-interceptor-stream-wrapper-types", wrappedClientStreams, false))
		}
	}

	// Generate the interceptor wrapper functions next (only once)
	if len(svc.ServerInterceptors) > 0 {
		sections = append(sections, serverInterceptorWrappersSection(svc.Name, svc.ServerInterceptors))
	}
	if len(svc.ClientInterceptors) > 0 {
		sections = append(sections, clientInterceptorWrappersSection(svc.Name, svc.ClientInterceptors))
	}

	// Generate any interceptor stream wrapper struct methods last
	if len(wrappedServerStreams) > 0 {
		sections = append(sections, streamWrappersSection("server-interceptor-stream-wrappers", wrappedServerStreams, true))
	}
	if len(wrappedClientStreams) > 0 {
		sections = append(sections, streamWrappersSection("client-interceptor-stream-wrappers", wrappedClientStreams, false))
	}

	return &codegen.File{
		Path:     path,
		Sections: sections,
	}
}

// hasPrivateImplementationTypes returns true if any of the interceptors have
// private implementation types.
func hasPrivateImplementationTypes(interceptors []*InterceptorData) bool {
	for _, intr := range interceptors {
		if intr.ReadPayload != nil || intr.WritePayload != nil || intr.ReadResult != nil || intr.WriteResult != nil || intr.ReadStreamingPayload != nil || intr.WriteStreamingPayload != nil || intr.ReadStreamingResult != nil || intr.WriteStreamingResult != nil {
			return true
		}
	}
	return false
}

// hasEndpointStruct returns a function that returns true if the method has an endpoint struct
// if server is true, otherwise it returns false.
func hasEndpointStruct(server bool) func(*MethodInterceptorData) bool {
	if !server {
		return func(*MethodInterceptorData) bool { return false }
	}
	return func(m *MethodInterceptorData) bool {
		return m.ServerStream != nil && m.ServerStream.EndpointStruct != ""
	}
}

// collectWrappedStreams returns a slice of streams to be wrapped by interceptor wrapper functions.
func collectWrappedStreams(interceptors []*InterceptorData, server bool) []*StreamInterceptorData {
	var (
		streams     []*StreamInterceptorData
		streamNames = make(map[string]struct{})
	)
	for _, intr := range interceptors {
		if intr.HasStreamingPayloadAccess || intr.HasStreamingResultAccess {
			for _, method := range intr.Methods {
				if server {
					if method.ServerStream.serverPlan.endpoint == "" {
						continue
					}
					if _, ok := streamNames[method.ServerStream.Interface]; !ok {
						streams = append(streams, method.ServerStream)
						streamNames[method.ServerStream.Interface] = struct{}{}
					}
				} else {
					if _, ok := streamNames[method.ClientStream.Interface]; !ok {
						streams = append(streams, method.ClientStream)
						streamNames[method.ClientStream.Interface] = struct{}{}
					}
				}
			}
		}
	}
	return streams
}
