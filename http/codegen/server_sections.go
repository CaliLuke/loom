package codegen

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/CaliLuke/loom/codegen"
)

func serverStructSection(data *ServiceData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s lists the %s service endpoint HTTP handlers.", data.ServerStruct, data.Service.Name)))
	b.Add("\n")
	b.Addf("type %s struct {\n", data.ServerStruct)
	b.Addf("\tMounts []*%s\n", data.MountPointStruct)
	for _, endpoint := range data.Endpoints {
		b.Addf("\t%s http.Handler\n", endpoint.Method.VarName)
	}
	for _, fs := range data.FileServers {
		b.Addf("\t%s http.Handler\n", fs.VarName)
	}
	b.Add("}\n")
	return codegen.MustRenderSection("server-struct", b.String)
}

func mountPointStructSection(data *ServiceData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s holds information about the mounted endpoints.", data.MountPointStruct)))
	b.Add("\n")
	b.Addf("type %s struct {\n", data.MountPointStruct)
	b.Add("\t" + codegen.Comment("Method is the name of the service method served by the mounted HTTP handler.") + "\n")
	b.Add("\tMethod string\n")
	b.Add("\t" + codegen.Comment("Verb is the HTTP method used to match requests to the mounted handler.") + "\n")
	b.Add("\tVerb string\n")
	b.Add("\t" + codegen.Comment("Pattern is the HTTP request path pattern used to match requests to the mounted handler.") + "\n")
	b.Add("\tPattern string\n")
	b.Add("}\n")
	return codegen.MustRenderSection("server-mountpoint", b.String)
}

func serverInitSection(data *ServiceData) codegen.Section {
	return codegen.MustRenderSection("server-init", func() string {
		return renderServerInit(data)
	})
}

func renderServerInit(data *ServiceData) string {
	var b sourceBuilder
	comment := fmt.Sprintf("%s instantiates HTTP handlers for all the %s service endpoints using the provided encoder and decoder. The handlers are mounted on the given mux using the HTTP verb and path defined in the design. errhandler is called whenever a response fails to be encoded. formatter is used to format errors returned by the service methods prior to encoding. Both errhandler and formatter are optional and can be nil.", data.ServerInit, data.Service.Name)
	b.Add("\n")
	b.Add(codegen.Comment(comment))
	b.Add("\n")
	b.Addf("func %s(\n", data.ServerInit)
	b.Addf("\te *%s.Endpoints,\n", data.Service.PkgName)
	b.Add("\tmux loomhttp.Muxer,\n")
	b.Add("\tdecoder func(*http.Request) loomhttp.Decoder,\n")
	b.Add("\tencoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,\n")
	b.Add("\terrhandler func(context.Context, http.ResponseWriter, error),\n")
	b.Add("\tformatter func(ctx context.Context, err error) loomhttp.Statuser,\n")
	if HasWebSocket(data) {
		b.Add("\tupgrader loomhttp.Upgrader,\n")
		b.Add("\tconfigurer *ConnConfigurer,\n")
	}
	for _, endpoint := range data.Endpoints {
		if endpoint.MultipartRequestDecoder != nil {
			b.Addf("\t%s %s,\n", endpoint.MultipartRequestDecoder.VarName, endpoint.MultipartRequestDecoder.FuncName)
		}
	}
	for _, fs := range data.FileServers {
		b.Addf("\t%s http.FileSystem,\n", fs.ArgName)
	}
	b.Addf(") *%s {\n", data.ServerStruct)
	if HasWebSocket(data) {
		b.Add("\tif configurer == nil {\n\t\tconfigurer = &ConnConfigurer{}\n\t}\n")
	}
	for _, fs := range data.FileServers {
		b.Addf("\tif %s == nil {\n\t\t%s = http.Dir(\".\")\n\t}\n", fs.ArgName, fs.ArgName)
		prefix := addLeadingSlash(fs.FilePath)
		if !fs.IsDir {
			prefix = filepath.Dir(prefix)
		}
		b.Addf("\t%s = appendPrefix(%s, %q)\n", fs.ArgName, fs.ArgName, prefix)
	}
	b.Addf("\treturn &%s{\n", data.ServerStruct)
	b.Addf("\t\tMounts: []*%s{\n", data.MountPointStruct)
	for _, endpoint := range data.Endpoints {
		for _, route := range endpoint.Routes {
			b.Addf("\t\t\t{%q, %q, %q},\n", endpoint.Method.VarName, route.Verb, route.Path)
		}
	}
	for _, fs := range data.FileServers {
		for _, requestPath := range fs.RequestPaths {
			b.Addf("\t\t\t{%q, %q, %q},\n", "Serve "+fs.FilePath, "GET", requestPath)
		}
	}
	b.Add("\t\t},\n")
	for _, endpoint := range data.Endpoints {
		b.Addf("\t\t%s: %s(e.%s, mux, ", endpoint.Method.VarName, endpoint.HandlerInit, endpoint.Method.VarName)
		if endpoint.MultipartRequestDecoder != nil {
			b.Addf("%s(mux, %s)", endpoint.MultipartRequestDecoder.InitName, endpoint.MultipartRequestDecoder.VarName)
		} else {
			b.Add("decoder")
		}
		b.Add(", encoder, errhandler, formatter")
		if IsWebSocketEndpoint(endpoint) {
			b.Addf(", upgrader, configurer.%sFn", endpoint.Method.VarName)
		}
		b.Add("),\n")
	}
	for _, fs := range data.FileServers {
		b.Addf("\t\t%s: http.FileServer(%s),\n", fs.VarName, fs.ArgName)
	}
	b.Add("\t}\n}\n")
	return b.String()
}

func serverServiceSection(data *ServiceData) codegen.Section {
	return codegen.MustRenderSection("server-service", func() string {
		return fmt.Sprintf("\n%s\nfunc (s *%s) %s() string { return %q }\n", codegen.Comment(fmt.Sprintf("%s returns the name of the service served.", data.ServerService)), data.ServerStruct, data.ServerService, data.Service.Name)
	})
}

func serverUseSection(data *ServiceData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment("Use wraps the server handlers with the given middleware."))
	b.Add("\n")
	b.Addf("func (s *%s) Use(m func(http.Handler) http.Handler) {\n", data.ServerStruct)
	for _, endpoint := range data.Endpoints {
		b.Addf("\ts.%s = m(s.%s)\n", endpoint.Method.VarName, endpoint.Method.VarName)
	}
	b.Add("}\n")
	return codegen.MustRenderSection("server-use", b.String)
}

func serverMethodNamesSection(data *ServiceData) codegen.Section {
	return codegen.MustRenderSection("server-method-names", func() string {
		return fmt.Sprintf("\n%s\nfunc (s *%s) MethodNames() []string { return %s.MethodNames[:] }\n", codegen.Comment("MethodNames returns the methods served."), data.ServerStruct, data.Service.PkgName)
	})
}

func serverMountSection(data *ServiceData) codegen.Section {
	return codegen.MustRenderSection("server-mount", func() string {
		return renderServerMount(data)
	})
}

func renderServerMount(data *ServiceData) string {
	var b sourceBuilder
	comment := codegen.Comment(fmt.Sprintf("%s configures the mux to serve the %s endpoints.", data.MountServer, data.Service.Name))
	b.Add("\n")
	b.Add(comment)
	b.Add("\n")
	b.Addf("func %s(mux loomhttp.Muxer, h *%s) {\n", data.MountServer, data.ServerStruct)
	for _, endpoint := range data.Endpoints {
		b.Addf("\t%s(mux, h.%s)\n", endpoint.MountHandler, endpoint.Method.VarName)
	}
	for _, fs := range data.FileServers {
		if fs.Redirect != nil {
			b.Addf("\t%s(mux, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {\n", fs.MountHandler)
			b.Addf("\t\thttp.Redirect(w, r, %q, %s)\n", fs.Redirect.URL, fs.Redirect.StatusCode)
			b.Add("\t}))\n")
			continue
		}
		for _, requestPath := range fs.RequestPaths {
			stripped := addLeadingSlash(requestPath)
			if !fs.IsDir {
				stripped = filepath.Dir(stripped)
			}
			if stripped == "/" {
				b.Addf("\t%s(mux, h.%s)\n", fs.MountHandler, fs.VarName)
			} else {
				b.Addf("\t%s(mux, http.StripPrefix(%q, h.%s))\n", fs.MountHandler, stripped, fs.VarName)
			}
		}
	}
	b.Add("}\n\n")
	b.Add(comment)
	b.Add("\n")
	b.Addf("func (s *%s) %s(mux loomhttp.Muxer) {\n", data.ServerStruct, data.MountServer)
	b.Addf("\t%s(mux, s)\n", data.MountServer)
	b.Add("}\n")
	return b.String()
}

func serverHandlerSection(data *EndpointData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s configures the mux to serve the %q service %q endpoint.", data.MountHandler, data.ServiceName, data.Method.Name)))
	b.Add("\n")
	b.Addf("func %s(mux loomhttp.Muxer, h http.Handler) {\n", data.MountHandler)
	b.Add("\tf, ok := h.(http.HandlerFunc)\n")
	b.Add("\tif !ok {\n")
	b.Add("\t\tf = func(w http.ResponseWriter, r *http.Request) {\n\t\t\th.ServeHTTP(w, r)\n\t\t}\n\t}\n")
	for _, route := range data.Routes {
		b.Addf("\tmux.Handle(%q, %q, f)\n", route.Verb, route.Path)
	}
	b.Add("}\n")
	return codegen.MustRenderSection("server-handler", b.String)
}

func appendFSSection(mappedFiles map[string]string) codegen.Section {
	var b sourceBuilder
	b.Add("\n// appendFS is a custom implementation of fs.FS that appends a specified prefix\n")
	b.Add("// to the file paths before delegating the Open call to the underlying fs.FS.\n")
	b.Add("type appendFS struct {\n\tprefix string\n\tfs     http.FileSystem\n}\n\n")
	b.Add("// Open opens the named file, appending the prefix to the file path before\n")
	b.Add("// passing it to the underlying fs.FS.\n")
	b.Add("func (s appendFS) Open(name string) (http.File, error) {\n")
	b.Add("\tswitch name {\n")
	for requested, embedded := range mappedFiles {
		b.Addf("\tcase %q:\n\t\tname = %q\n", requested, embedded)
	}
	b.Add("\t}\n")
	b.Add("\treturn s.fs.Open(path.Join(s.prefix, name))\n")
	b.Add("}\n\n")
	b.Add("// appendPrefix returns a new fs.FS that appends the specified prefix to file paths\n")
	b.Add("// before delegating to the provided embed.FS.\n")
	b.Add("func appendPrefix(fsys http.FileSystem, prefix string) http.FileSystem {\n")
	b.Add("\treturn appendFS{prefix: prefix, fs: fsys}\n}\n")
	return codegen.MustRenderSection("append-fs", b.String)
}

func fileServerSection(data *FileServerData) codegen.Section {
	var b sourceBuilder
	b.Add("\n")
	b.Add(codegen.Comment(fmt.Sprintf("%s configures the mux to serve GET request made to %q.", data.MountHandler, strings.Join(data.RequestPaths, ", "))))
	b.Add("\n")
	b.Addf("func %s(mux loomhttp.Muxer, h http.Handler) {\n", data.MountHandler)
	if data.IsDir {
		for _, requestPath := range data.RequestPaths {
			suffix := ""
			if requestPath != "/" {
				suffix = "/"
			}
			b.Addf("\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath+suffix)
			b.Addf("\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath+suffix+"{*"+data.PathParam+"}")
		}
	} else {
		for _, requestPath := range data.RequestPaths {
			b.Addf("\tmux.Handle(%q, %q, h.ServeHTTP)\n", "GET", requestPath)
		}
	}
	b.Add("}\n")
	return codegen.MustRenderSection("server-files", b.String)
}
