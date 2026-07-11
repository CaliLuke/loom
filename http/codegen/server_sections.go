package codegen

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/dave/jennifer/jen"

	"github.com/CaliLuke/loom/codegen"
)

func serverStructSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-struct", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s lists the %s service endpoint HTTP handlers.", data.ServerStruct, data.Service.Name))
		stmt.Type().Id(data.ServerStruct).StructFunc(func(group *jen.Group) {
			group.Id("Mounts").Index().Op("*").Id(data.MountPointStruct)
			for _, endpoint := range data.Endpoints {
				group.Id(endpoint.Method.VarName).Qual("net/http", "Handler")
			}
			for _, fs := range data.FileServers {
				group.Id(fs.VarName).Qual("net/http", "Handler")
			}
		})
		stmt.Line()
	})
}

func mountPointStructSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-mountpoint", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s holds information about the mounted endpoints.", data.MountPointStruct))
		stmt.Type().Id(data.MountPointStruct).StructFunc(func(group *jen.Group) {
			group.Comment("Method is the name of the service method served by the mounted HTTP handler.")
			group.Id("Method").String()
			group.Comment("Verb is the HTTP method used to match requests to the mounted handler.")
			group.Id("Verb").String()
			group.Comment("Pattern is the HTTP request path pattern used to match requests to the mounted handler.")
			group.Id("Pattern").String()
		})
		stmt.Line()
	})
}

func serverInitSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-init", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s instantiates HTTP handlers for all the %s service endpoints using the provided encoder and decoder. The handlers are mounted on the given mux using the HTTP verb and path defined in the design. errhandler is called whenever a response fails to be encoded. formatter is used to format errors returned by the service methods prior to encoding. Both errhandler and formatter are optional and can be nil.", data.ServerInit, data.Service.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().
			Id(data.ServerInit).
			ParamsFunc(func(group *jen.Group) {
				group.Id("e").Op("*").Add(codegen.TypeRef(data.Service.PkgName + ".Endpoints"))
				group.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer"))
				group.Id("decoder").Func().Params(jen.Op("*").Qual("net/http", "Request")).Add(codegen.TypeRef("loomhttp.Decoder"))
				group.Id("encoder").Func().Params(
					jen.Id("ctx").Qual("context", "Context"),
					jen.Id("w").Qual("net/http", "ResponseWriter"),
				).Add(codegen.TypeRef("loomhttp.Encoder"))
				group.Id("errhandler").Func().Params(
					jen.Id("ctx").Qual("context", "Context"),
					jen.Id("w").Qual("net/http", "ResponseWriter"),
					jen.Id("err").Error(),
				)
				group.Id("formatter").Func().Params(jen.Id("ctx").Qual("context", "Context"), jen.Id("err").Error()).Add(codegen.TypeRef("loomhttp.Statuser"))
				if HasWebSocket(data) {
					group.Id("upgrader").Add(codegen.TypeRef("loomhttp.Upgrader"))
					group.Id("configurer").Add(codegen.TypeRef("*ConnConfigurer"))
				}
				for _, endpoint := range data.Endpoints {
					if endpoint.MultipartRequestDecoder != nil {
						group.Id(endpoint.MultipartRequestDecoder.VarName).Add(codegen.TypeRef(endpoint.MultipartRequestDecoder.FuncName))
					}
				}
				for _, fs := range data.FileServers {
					group.Id(fs.ArgName).Qual("net/http", "FileSystem")
				}
			}).
			Op("*").Id(data.ServerStruct).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderServerInitBody(data))
			})
		stmt.Line()
	})
}

func renderServerInitBody(data *ServiceData) string {
	var b sourceBuilder
	if HasWebSocket(data) {
		b.Add("\tif configurer == nil {\n\t\tconfigurer = &ConnConfigurer{}\n\t}\n")
	}
	for _, fs := range data.FileServers {
		b.Addf("\tif %s == nil {\n\t\t%s = http.Dir(\".\")\n\t}\n", fs.ArgName, fs.ArgName)
		prefix := addLeadingSlash(fs.FilePath)
		if !fs.IsDir {
			prefix = path.Dir(prefix)
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
	b.Add("\t}\n")
	return b.String()
}

func serverServiceSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-service", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s returns the name of the service served.", data.ServerService))
		stmt.Func().Params(jen.Id("s").Op("*").Id(data.ServerStruct)).Id(data.ServerService).Params().String().Block(
			jen.Return(jen.Lit(data.Service.Name)),
		)
		stmt.Line()
	})
}

func serverUseSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-use", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "Use wraps the server handlers with the given middleware.")
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("Use").
			Params(jen.Id("m").Func().Params(jen.Qual("net/http", "Handler")).Qual("net/http", "Handler")).
			BlockFunc(func(group *jen.Group) {
				for _, endpoint := range data.Endpoints {
					group.Id("s").Dot(endpoint.Method.VarName).Op("=").Id("m").Call(jen.Id("s").Dot(endpoint.Method.VarName))
				}
			})
		stmt.Line()
	})
}

func serverMethodNamesSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-method-names", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "MethodNames returns the methods served.")
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id("MethodNames").
			Params().
			Index().String().
			Block(
				jen.Return(codegen.Expr(data.Service.PkgName + ".MethodNames[:]")),
			)
		stmt.Line()
	})
}

func serverMountSection(data *ServiceData) codegen.Section {
	return codegen.NewJenniferSection("server-mount", func(stmt *jen.Statement) {
		comment := fmt.Sprintf("%s configures the mux to serve the %s endpoints.", data.MountServer, data.Service.Name)
		codegen.Doc(stmt, comment)
		stmt.Func().
			Id(data.MountServer).
			Params(jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")), jen.Id("h").Op("*").Id(data.ServerStruct)).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderServerMountBody(data, true))
			})
		stmt.Line()
		codegen.Doc(stmt, comment)
		stmt.Func().
			Params(jen.Id("s").Op("*").Id(data.ServerStruct)).
			Id(data.MountServer).
			Params(jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer"))).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderServerMountBody(data, false))
			})
		stmt.Line()
	})
}

func renderServerMountBody(data *ServiceData, standalone bool) string {
	var b sourceBuilder
	if standalone {
		if data.CORS != nil {
			for _, route := range corsPreflightRoutes(data) {
				b.Addf("\tmux.Handle(%q, %q, func(w http.ResponseWriter, r *http.Request) {\n", "OPTIONS", route.Path)
				b.Addf("\t\tloomhttp.HandleCORSPreflight(w, r, %s, []string{%s})\n", renderCORSPolicy(data.CORS), quotedStringList(route.Methods))
				b.Add("\t})\n")
			}
		}
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
					stripped = path.Dir(stripped)
				}
				if stripped == "/" {
					b.Addf("\t%s(mux, h.%s)\n", fs.MountHandler, fs.VarName)
				} else {
					b.Addf("\t%s(mux, http.StripPrefix(%q, h.%s))\n", fs.MountHandler, stripped, fs.VarName)
				}
			}
		}
		return b.String()
	}
	b.Addf("\t%s(mux, s)\n", data.MountServer)
	return b.String()
}

func serverHandlerSection(data *EndpointData) codegen.Section {
	return codegen.NewJenniferSection("server-handler", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s configures the mux to serve the %q service %q endpoint.", data.MountHandler, data.ServiceName, data.Method.Name))
		stmt.Func().
			Id(data.MountHandler).
			Params(jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")), jen.Id("h").Qual("net/http", "Handler")).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderServerHandlerBody(data))
			})
		stmt.Line()
	})
}

func renderServerHandlerBody(data *EndpointData) string {
	var b sourceBuilder
	b.Add("\tf, ok := h.(http.HandlerFunc)\n")
	b.Add("\tif !ok {\n")
	b.Add("\t\tf = func(w http.ResponseWriter, r *http.Request) {\n\t\t\th.ServeHTTP(w, r)\n\t\t}\n\t}\n")
	if data.CORS != nil {
		b.Addf("\tf = loomhttp.CORSHandler(%s, f)\n", renderCORSPolicy(data.CORS))
	}
	for _, route := range data.Routes {
		b.Addf("\tmux.Handle(%q, %q, f)\n", route.Verb, route.Path)
	}
	return b.String()
}

type corsPreflightRoute struct {
	Path    string
	Methods []string
}

func corsPreflightRoutes(data *ServiceData) []corsPreflightRoute {
	byPath := make(map[string]map[string]struct{})
	for _, endpoint := range data.Endpoints {
		for _, route := range endpoint.Routes {
			methods := byPath[route.Path]
			if methods == nil {
				methods = make(map[string]struct{})
				byPath[route.Path] = methods
			}
			methods[route.Verb] = struct{}{}
		}
	}
	paths := make([]string, 0, len(byPath))
	for path := range byPath {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	routes := make([]corsPreflightRoute, 0, len(paths))
	for _, path := range paths {
		methods := make([]string, 0, len(byPath[path]))
		for method := range byPath[path] {
			methods = append(methods, method)
		}
		sort.Strings(methods)
		routes = append(routes, corsPreflightRoute{Path: path, Methods: methods})
	}
	return routes
}

func renderCORSPolicy(cors *CORSData) string {
	var b sourceBuilder
	b.Add("loomhttp.CORSPolicy{Origins: []loomhttp.CORSOrigin{")
	for _, origin := range cors.Origins {
		b.Add("{")
		b.Addf("Pattern: %q,", origin.Pattern)
		if origin.Regex {
			b.Add("Regex: true,")
		}
		if len(origin.Methods) > 0 {
			b.Addf("Methods: []string{%s},", quotedStringList(origin.Methods))
		}
		if len(origin.Headers) > 0 {
			b.Addf("Headers: []string{%s},", quotedStringList(origin.Headers))
		}
		if len(origin.Expose) > 0 {
			b.Addf("Expose: []string{%s},", quotedStringList(origin.Expose))
		}
		if origin.MaxAge > 0 {
			b.Addf("MaxAge: %d,", origin.MaxAge)
		}
		if origin.Credentials {
			b.Add("Credentials: true,")
		}
		b.Add("},")
	}
	b.Add("}}")
	return b.String()
}

func quotedStringList(values []string) string {
	quoted := make([]string, len(values))
	for i, value := range values {
		quoted[i] = fmt.Sprintf("%q", value)
	}
	return strings.Join(quoted, ", ")
}

func appendFSSection(mappedFiles map[string]string) codegen.Section {
	return codegen.NewJenniferSection("append-fs", func(stmt *jen.Statement) {
		codegen.Doc(stmt, "appendFS is a custom implementation of fs.FS that appends a specified prefix to the file paths before delegating the Open call to the underlying fs.FS.")
		stmt.Type().Id("appendFS").Struct(
			jen.Id("prefix").String(),
			jen.Id("fs").Qual("net/http", "FileSystem"),
		)
		stmt.Line()
		codegen.Doc(stmt, "Open opens the named file, appending the prefix to the file path before passing it to the underlying fs.FS.")
		stmt.Func().
			Params(jen.Id("s").Id("appendFS")).
			Id("Open").
			Params(jen.Id("name").String()).
			Params(jen.Qual("net/http", "File"), jen.Error()).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderAppendFSOpenBody(mappedFiles))
			})
		stmt.Line()
		codegen.Doc(stmt, "appendPrefix returns a new fs.FS that appends the specified prefix to file paths before delegating to the provided embed.FS.")
		stmt.Func().
			Id("appendPrefix").
			Params(jen.Id("fsys").Qual("net/http", "FileSystem"), jen.Id("prefix").String()).
			Qual("net/http", "FileSystem").
			Block(
				jen.Return(jen.Id("appendFS").Values(jen.Id("prefix").Op(":").Id("prefix"), jen.Id("fs").Op(":").Id("fsys"))),
			)
		stmt.Line()
	})
}

func renderAppendFSOpenBody(mappedFiles map[string]string) string {
	var b sourceBuilder
	b.Add("\tswitch name {\n")
	requestedPaths := make([]string, 0, len(mappedFiles))
	for requested := range mappedFiles {
		requestedPaths = append(requestedPaths, requested)
	}
	sort.Strings(requestedPaths)
	for _, requested := range requestedPaths {
		embedded := mappedFiles[requested]
		b.Addf("\tcase %q:\n\t\tname = %q\n", requested, embedded)
	}
	b.Add("\t}\n")
	b.Add("\treturn s.fs.Open(path.Join(s.prefix, name))\n")
	return b.String()
}

func fileServerSection(data *FileServerData) codegen.Section {
	return codegen.NewJenniferSection("server-files", func(stmt *jen.Statement) {
		codegen.Doc(stmt, fmt.Sprintf("%s configures the mux to serve GET request made to %q.", data.MountHandler, strings.Join(data.RequestPaths, ", ")))
		stmt.Func().
			Id(data.MountHandler).
			Params(jen.Id("mux").Add(codegen.TypeRef("loomhttp.Muxer")), jen.Id("h").Qual("net/http", "Handler")).
			BlockFunc(func(group *jen.Group) {
				appendHTTPRawBlock(group, renderFileServerBody(data))
			})
		stmt.Line()
	})
}

func renderFileServerBody(data *FileServerData) string {
	var b sourceBuilder
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
	return b.String()
}
