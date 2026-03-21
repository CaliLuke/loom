package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"goa.design/goa/v3/codegen"
)

func serverStructSection(data *ServiceData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s lists the %s service endpoint HTTP handlers.", data.ServerStruct, data.Service.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", data.ServerStruct)
	fmt.Fprintf(&b, "\tMounts []*%s\n", data.MountPointStruct)
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\t%s http.Handler\n", endpoint.Method.VarName)
	}
	for _, fs := range data.FileServers {
		fmt.Fprintf(&b, "\t%s http.Handler\n", fs.VarName)
	}
	b.WriteString("}\n")
	return codegen.NewRawSection("server-struct", b.String())
}

func mountPointStructSection(data *ServiceData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s holds information about the mounted endpoints.", data.MountPointStruct)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "type %s struct {\n", data.MountPointStruct)
	b.WriteString("\t" + codegen.Comment("Method is the name of the service method served by the mounted HTTP handler.") + "\n")
	b.WriteString("\tMethod string\n")
	b.WriteString("\t" + codegen.Comment("Verb is the HTTP method used to match requests to the mounted handler.") + "\n")
	b.WriteString("\tVerb string\n")
	b.WriteString("\t" + codegen.Comment("Pattern is the HTTP request path pattern used to match requests to the mounted handler.") + "\n")
	b.WriteString("\tPattern string\n")
	b.WriteString("}\n")
	return codegen.NewRawSection("server-mountpoint", b.String())
}

func serverInitSection(data *ServiceData) codegen.Section {
	return codegen.NewRawSection("server-init", renderServerInit(data))
}

func renderServerInit(data *ServiceData) string {
	var b strings.Builder
	comment := fmt.Sprintf("%s instantiates HTTP handlers for all the %s service endpoints using the provided encoder and decoder. The handlers are mounted on the given mux using the HTTP verb and path defined in the design. errhandler is called whenever a response fails to be encoded. formatter is used to format errors returned by the service methods prior to encoding. Both errhandler and formatter are optional and can be nil.", data.ServerInit, data.Service.Name)
	b.WriteString("\n")
	b.WriteString(codegen.Comment(comment))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(\n", data.ServerInit)
	fmt.Fprintf(&b, "\te *%s.Endpoints,\n", data.Service.PkgName)
	b.WriteString("\tmux goahttp.Muxer,\n")
	b.WriteString("\tdecoder func(*http.Request) goahttp.Decoder,\n")
	b.WriteString("\tencoder func(context.Context, http.ResponseWriter) goahttp.Encoder,\n")
	b.WriteString("\terrhandler func(context.Context, http.ResponseWriter, error),\n")
	b.WriteString("\tformatter func(ctx context.Context, err error) goahttp.Statuser,\n")
	if HasWebSocket(data) {
		b.WriteString("\tupgrader goahttp.Upgrader,\n")
		b.WriteString("\tconfigurer *ConnConfigurer,\n")
	}
	for _, endpoint := range data.Endpoints {
		if endpoint.MultipartRequestDecoder != nil {
			fmt.Fprintf(&b, "\t%s %s,\n", endpoint.MultipartRequestDecoder.VarName, endpoint.MultipartRequestDecoder.FuncName)
		}
	}
	for _, fs := range data.FileServers {
		fmt.Fprintf(&b, "\t%s http.FileSystem,\n", fs.ArgName)
	}
	fmt.Fprintf(&b, ") *%s {\n", data.ServerStruct)
	if HasWebSocket(data) {
		b.WriteString("\tif configurer == nil {\n\t\tconfigurer = &ConnConfigurer{}\n\t}\n")
	}
	for _, fs := range data.FileServers {
		fmt.Fprintf(&b, "\tif %s == nil {\n\t\t%s = http.Dir(\".\")\n\t}\n", fs.ArgName, fs.ArgName)
		prefix := addLeadingSlash(fs.FilePath)
		if !fs.IsDir {
			prefix = filepath.Dir(prefix)
		}
		fmt.Fprintf(&b, "\t%s = appendPrefix(%s, %q)\n", fs.ArgName, fs.ArgName, prefix)
	}
	fmt.Fprintf(&b, "\treturn &%s{\n", data.ServerStruct)
	fmt.Fprintf(&b, "\t\tMounts: []*%s{\n", data.MountPointStruct)
	for _, endpoint := range data.Endpoints {
		for _, route := range endpoint.Routes {
			fmt.Fprintf(&b, "\t\t\t{%q, %q, %q},\n", endpoint.Method.VarName, route.Verb, route.Path)
		}
	}
	for _, fs := range data.FileServers {
		for _, requestPath := range fs.RequestPaths {
			fmt.Fprintf(&b, "\t\t\t{%q, %q, %q},\n", "Serve "+fs.FilePath, "GET", requestPath)
		}
	}
	b.WriteString("\t\t},\n")
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\t\t%s: %s(e.%s, mux, ", endpoint.Method.VarName, endpoint.HandlerInit, endpoint.Method.VarName)
		if endpoint.MultipartRequestDecoder != nil {
			fmt.Fprintf(&b, "%s(mux, %s)", endpoint.MultipartRequestDecoder.InitName, endpoint.MultipartRequestDecoder.VarName)
		} else {
			b.WriteString("decoder")
		}
		b.WriteString(", encoder, errhandler, formatter")
		if IsWebSocketEndpoint(endpoint) {
			fmt.Fprintf(&b, ", upgrader, configurer.%sFn", endpoint.Method.VarName)
		}
		b.WriteString("),\n")
	}
	for _, fs := range data.FileServers {
		fmt.Fprintf(&b, "\t\t%s: http.FileServer(%s),\n", fs.VarName, fs.ArgName)
	}
	b.WriteString("\t}\n}\n")
	return b.String()
}

func serverServiceSection(data *ServiceData) codegen.Section {
	return codegen.NewRawSection("server-service", fmt.Sprintf("\n%s\nfunc (s *%s) %s() string { return %q }\n", codegen.Comment(fmt.Sprintf("%s returns the name of the service served.", data.ServerService)), data.ServerStruct, data.ServerService, data.Service.Name))
}

func serverUseSection(data *ServiceData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment("Use wraps the server handlers with the given middleware."))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) Use(m func(http.Handler) http.Handler) {\n", data.ServerStruct)
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\ts.%s = m(s.%s)\n", endpoint.Method.VarName, endpoint.Method.VarName)
	}
	b.WriteString("}\n")
	return codegen.NewRawSection("server-use", b.String())
}

func serverMethodNamesSection(data *ServiceData) codegen.Section {
	return codegen.NewRawSection("server-method-names", fmt.Sprintf("\n%s\nfunc (s *%s) MethodNames() []string { return %s.MethodNames[:] }\n", codegen.Comment("MethodNames returns the methods served."), data.ServerStruct, data.Service.PkgName))
}

func serverMountSection(data *ServiceData) codegen.Section {
	return codegen.NewRawSection("server-mount", renderServerMount(data))
}

func renderServerMount(data *ServiceData) string {
	var b strings.Builder
	comment := codegen.Comment(fmt.Sprintf("%s configures the mux to serve the %s endpoints.", data.MountServer, data.Service.Name))
	b.WriteString("\n")
	b.WriteString(comment)
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(mux goahttp.Muxer, h *%s) {\n", data.MountServer, data.ServerStruct)
	for _, endpoint := range data.Endpoints {
		fmt.Fprintf(&b, "\t%s(mux, h.%s)\n", endpoint.MountHandler, endpoint.Method.VarName)
	}
	for _, fs := range data.FileServers {
		if fs.Redirect != nil {
			fmt.Fprintf(&b, "\t%s(mux, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n", fs.MountHandler)
			fmt.Fprintf(&b, "\t\thttp.Redirect(w, r, %q, %s)\n", fs.Redirect.URL, fs.Redirect.StatusCode)
			b.WriteString("\t}))\n")
			continue
		}
		for _, requestPath := range fs.RequestPaths {
			stripped := addLeadingSlash(requestPath)
			if !fs.IsDir {
				stripped = filepath.Dir(stripped)
			}
			if stripped == "/" {
				fmt.Fprintf(&b, "\t%s(mux, h.%s)\n", fs.MountHandler, fs.VarName)
			} else {
				fmt.Fprintf(&b, "\t%s(mux, http.StripPrefix(%q, h.%s))\n", fs.MountHandler, stripped, fs.VarName)
			}
		}
	}
	b.WriteString("}\n\n")
	b.WriteString(comment)
	b.WriteString("\n")
	fmt.Fprintf(&b, "func (s *%s) %s(mux goahttp.Muxer) {\n", data.ServerStruct, data.MountServer)
	fmt.Fprintf(&b, "\t%s(mux, s)\n", data.MountServer)
	b.WriteString("}\n")
	return b.String()
}

func serverHandlerSection(data *EndpointData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s configures the mux to serve the %q service %q endpoint.", data.MountHandler, data.ServiceName, data.Method.Name)))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(mux goahttp.Muxer, h http.Handler) {\n", data.MountHandler)
	b.WriteString("\tf, ok := h.(http.HandlerFunc)\n")
	b.WriteString("\tif !ok {\n")
	b.WriteString("\t\tf = func(w http.ResponseWriter, r *http.Request) {\n\t\t\th.ServeHTTP(w, r)\n\t\t}\n\t}\n")
	for _, route := range data.Routes {
		fmt.Fprintf(&b, "\tmux.Handle(%q, %q, f)\n", route.Verb, route.Path)
	}
	b.WriteString("}\n")
	return codegen.NewRawSection("server-handler", b.String())
}

func appendFSSection(mappedFiles map[string]string) codegen.Section {
	var b strings.Builder
	b.WriteString("\n// appendFS is a custom implementation of fs.FS that appends a specified prefix\n")
	b.WriteString("// to the file paths before delegating the Open call to the underlying fs.FS.\n")
	b.WriteString("type appendFS struct {\n\tprefix string\n\tfs     http.FileSystem\n}\n\n")
	b.WriteString("// Open opens the named file, appending the prefix to the file path before\n")
	b.WriteString("// passing it to the underlying fs.FS.\n")
	b.WriteString("func (s appendFS) Open(name string) (http.File, error) {\n")
	b.WriteString("\tswitch name {\n")
	for requested, embedded := range mappedFiles {
		fmt.Fprintf(&b, "\tcase %q:\n\t\tname = %q\n", requested, embedded)
	}
	b.WriteString("\t}\n")
	b.WriteString("\treturn s.fs.Open(path.Join(s.prefix, name))\n")
	b.WriteString("}\n\n")
	b.WriteString("// appendPrefix returns a new fs.FS that appends the specified prefix to file paths\n")
	b.WriteString("// before delegating to the provided embed.FS.\n")
	b.WriteString("func appendPrefix(fsys http.FileSystem, prefix string) http.FileSystem {\n")
	b.WriteString("\treturn appendFS{prefix: prefix, fs: fsys}\n}\n")
	return codegen.NewRawSection("append-fs", b.String())
}

func fileServerSection(data *FileServerData) codegen.Section {
	var b strings.Builder
	b.WriteString("\n")
	b.WriteString(codegen.Comment(fmt.Sprintf("%s configures the mux to serve GET request made to %q.", data.MountHandler, strings.Join(data.RequestPaths, ", "))))
	b.WriteString("\n")
	fmt.Fprintf(&b, "func %s(mux goahttp.Muxer, h http.Handler) {\n", data.MountHandler)
	if data.IsDir {
		for _, requestPath := range data.RequestPaths {
			suffix := ""
			if requestPath != "/" {
				suffix = "/"
			}
			fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath+suffix)
			fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath+suffix+"{*"+data.PathParam+"}")
		}
	} else {
		for _, requestPath := range data.RequestPaths {
			fmt.Fprintf(&b, "\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath)
		}
	}
	b.WriteString("}\n")
	return codegen.NewRawSection("server-files", b.String())
}
