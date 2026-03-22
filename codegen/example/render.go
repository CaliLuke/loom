package example

import (
	"fmt"
	"io"
	"strings"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
)

type (
	renderSection struct {
		name string
		code func() string
	}
)

func (s *renderSection) SectionName() string {
	return s.name
}

func (s *renderSection) Write(w io.Writer) error {
	_, err := io.WriteString(w, s.code())
	return err
}

func newRenderSection(name string, code func() string) codegen.Section {
	return &renderSection{name: name, code: code}
}

func renderClientMain(server *Data, root *expr.RootExpr, hasJSONRPC, hasHTTP bool) string {
	var b strings.Builder
	fmt.Fprintf(&b, "func main() {\n")
	fmt.Fprintf(&b, "\tvar (\n")
	fmt.Fprintf(&b, "\t\thostF = flag.String(%q, %q, %q)\n", "host", server.DefaultHost().Name, fmt.Sprintf("Server host (valid values: %s)", strings.Join(server.AvailableHosts(), ", ")))
	fmt.Fprintf(&b, "\t\taddrF = flag.String(%q, %q, %q)\n", "url", "", "URL to service host")
	for _, v := range server.Variables {
		fmt.Fprintf(&b, "\t\t%sF = flag.String(%q, %q, %q)\n", v.VarName, v.Name, v.DefaultValue, v.Description)
	}
	if hasJSONRPC && hasHTTP {
		fmt.Fprintf(&b, "\t\tjsonrpcF = flag.Bool(%q, false, %q)\n", "jsonrpc", "Force JSON-RPC transport")
		fmt.Fprintf(&b, "\t\tjF = flag.Bool(%q, false, %q)\n", "j", "Force JSON-RPC transport")
	}
	fmt.Fprintf(&b, "\n")
	fmt.Fprintf(&b, "\t\tverboseF = flag.Bool(%q, false, %q)\n", "verbose", "Print request and response details")
	fmt.Fprintf(&b, "\t\tvF = flag.Bool(%q, false, %q)\n", "v", "Print request and response details")
	fmt.Fprintf(&b, "\t\ttimeoutF = flag.Int(%q, 30, %q)\n", "timeout", "Maximum number of seconds to wait for response")
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\tflag.Usage = usage\n")
	fmt.Fprintf(&b, "\tflag.Parse()\n\n")

	fmt.Fprintf(&b, "\tvar (\n")
	fmt.Fprintf(&b, "\t\taddr string\n")
	fmt.Fprintf(&b, "\t\ttimeout int\n")
	fmt.Fprintf(&b, "\t\tdebug bool\n")
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\t{\n")
	fmt.Fprintf(&b, "\t\taddr = *addrF\n")
	fmt.Fprintf(&b, "\t\tif addr == \"\" {\n")
	fmt.Fprintf(&b, "\t\t\tswitch *hostF {\n")
	for _, h := range server.Hosts {
		fmt.Fprintf(&b, "\t\t\tcase %q:\n", h.Name)
		fmt.Fprintf(&b, "\t\t\t\taddr = %q\n", h.DefaultURL(server.DefaultTransport().Type))
		writeVariableReplacement(&b, h.Variables, false)
	}
	fmt.Fprintf(&b, "\t\t\tdefault:\n")
	fmt.Fprintf(&b, "\t\t\t\tfmt.Fprintf(os.Stderr, %q, *hostF)\n", fmt.Sprintf("invalid host argument: %%q (valid hosts: %s)\n", strings.Join(server.AvailableHosts(), "|")))
	fmt.Fprintf(&b, "\t\t\t\tos.Exit(1)\n")
	fmt.Fprintf(&b, "\t\t\t}\n")
	fmt.Fprintf(&b, "\t\t}\n")
	fmt.Fprintf(&b, "\t\ttimeout = *timeoutF\n")
	fmt.Fprintf(&b, "\t\tdebug = *verboseF || *vF\n")
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tvar (\n")
	fmt.Fprintf(&b, "\t\tscheme string\n")
	fmt.Fprintf(&b, "\t\thost string\n")
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\t{\n")
	fmt.Fprintf(&b, "\t\tu, err := url.Parse(addr)\n")
	fmt.Fprintf(&b, "\t\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\t\tfmt.Fprintf(os.Stderr, %q, addr, err)\n", "invalid URL %#v: %s\n")
	fmt.Fprintf(&b, "\t\t\tos.Exit(1)\n")
	fmt.Fprintf(&b, "\t\t}\n")
	fmt.Fprintf(&b, "\t\tscheme = u.Scheme\n")
	fmt.Fprintf(&b, "\t\thost = u.Host\n")
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tvar (\n")
	fmt.Fprintf(&b, "\t\tendpoint goa.Endpoint\n")
	fmt.Fprintf(&b, "\t\tpayload any\n")
	fmt.Fprintf(&b, "\t\terr error\n")
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\t{\n")
	fmt.Fprintf(&b, "\t\tswitch scheme {\n")
	for _, t := range server.Transports {
		fmt.Fprintf(&b, "\t\tcase %q, %q:\n", t.Type, string(t.Type)+"s")
		if t.Type == "http" && hasJSONRPC {
			if hasHTTP {
				fmt.Fprintf(&b, "\t\t\tif *jsonrpcF || *jF {\n")
				fmt.Fprintf(&b, "\t\t\t\tendpoint, payload, err = doJSONRPC(scheme, host, timeout, debug)\n")
				fmt.Fprintf(&b, "\t\t\t} else {\n")
				fmt.Fprintf(&b, "\t\t\t\tendpoint, payload, err = doHTTP(scheme, host, timeout, debug)\n")
				fmt.Fprintf(&b, "\t\t\t\tif err != nil && strings.HasPrefix(err.Error(), \"unknown\") {\n")
				fmt.Fprintf(&b, "\t\t\t\t\tendpoint, payload, err = doJSONRPC(scheme, host, timeout, debug)\n")
				fmt.Fprintf(&b, "\t\t\t\t}\n")
				fmt.Fprintf(&b, "\t\t\t}\n")
			} else {
				fmt.Fprintf(&b, "\t\t\tendpoint, payload, err = doJSONRPC(scheme, host, timeout, debug)\n")
			}
		} else {
			fmt.Fprintf(&b, "\t\t\tendpoint, payload, err = do%s(scheme, host, timeout, debug)\n", strings.ToUpper(t.Name))
		}
	}
	fmt.Fprintf(&b, "\t\tdefault:\n")
	fmt.Fprintf(&b, "\t\t\tfmt.Fprintf(os.Stderr, %q, scheme)\n", fmt.Sprintf("invalid scheme: %%q (valid schemes: %s)\n", strings.Join(server.Schemes, "|")))
	fmt.Fprintf(&b, "\t\t\tos.Exit(1)\n")
	fmt.Fprintf(&b, "\t\t}\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\tif err == flag.ErrHelp {\n")
	fmt.Fprintf(&b, "\t\t\tos.Exit(0)\n")
	fmt.Fprintf(&b, "\t\t}\n")
	fmt.Fprintf(&b, "\t\tfmt.Fprintln(os.Stderr, err.Error())\n")
	fmt.Fprintf(&b, "\t\tfmt.Fprintln(os.Stderr, \"run '\"+os.Args[0]+\" --help' for detailed usage.\")\n")
	fmt.Fprintf(&b, "\t\tos.Exit(1)\n")
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\tdata, err := endpoint(context.Background(), payload)\n")
	fmt.Fprintf(&b, "\tif err != nil {\n")
	fmt.Fprintf(&b, "\t\tfmt.Fprintln(os.Stderr, err.Error())\n")
	fmt.Fprintf(&b, "\t\tos.Exit(1)\n")
	fmt.Fprintf(&b, "\t}\n\n")
	fmt.Fprintf(&b, "\tif data != nil {\n")
	fmt.Fprintf(&b, "\t\tm, _ := json.MarshalIndent(data, \"\", \"    \")\n")
	fmt.Fprintf(&b, "\t\tfmt.Println(string(m))\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "}\n\n")
	return b.String()
}

func renderUsage(apiName string, server *Data, hasJSONRPC, hasHTTP bool) string {
	var b strings.Builder
	defaultUsage := string(server.DefaultTransport().Type)
	if server.DefaultTransport().Type == "http" && !hasHTTP && hasJSONRPC {
		defaultUsage = "jsonrpc"
	}
	fmt.Fprintf(&b, "func usage() {\n")
	fmt.Fprintf(&b, "\tvar usageCommands []string\n")
	for _, t := range server.Transports {
		if t.Type == "http" && hasHTTP {
			fmt.Fprintf(&b, "\tusageCommands = append(usageCommands, %sUsageCommands()...)\n", t.Type)
		}
	}
	if hasJSONRPC {
		fmt.Fprintf(&b, "\tusageCommands = append(usageCommands, jsonrpcUsageCommands()...)\n")
	}
	fmt.Fprintf(&b, "\tsort.Strings(usageCommands)\n")
	fmt.Fprintf(&b, "\tusageCommands = slices.Compact(usageCommands)\n")
	fmt.Fprintf(&b, "\tfmt.Fprintf(os.Stderr, `%s is a command line client for the %s API.\n\n", "%s", apiName)
	fmt.Fprintf(&b, "Usage:\n")
	fmt.Fprintf(&b, "    %%s [-host HOST][-url URL][-timeout SECONDS][-verbose|-v]")
	for _, v := range server.Variables {
		fmt.Fprintf(&b, "[-%s %s]", v.Name, strings.ToUpper(v.Name))
	}
	fmt.Fprintf(&b, " SERVICE ENDPOINT [flags]\n\n")
	fmt.Fprintf(&b, "    -host HOST:  server host (%s). valid values: %s\n", server.DefaultHost().Name, strings.Join(server.AvailableHosts(), ", "))
	fmt.Fprintf(&b, "    -url URL:    specify service URL overriding host URL (http://localhost:8080)\n")
	if hasJSONRPC && hasHTTP {
		fmt.Fprintf(&b, "    -jsonrpc|-j: force JSON-RPC (false)\n")
	}
	fmt.Fprintf(&b, "    -timeout:    maximum number of seconds to wait for response (30)\n")
	fmt.Fprintf(&b, "    -verbose|-v: print request and response details (false)\n")
	for _, v := range server.Variables {
		fmt.Fprintf(&b, "    -%s:    %s (%s)\n", v.Name, v.Description, v.DefaultValue)
	}
	fmt.Fprintf(&b, "\nCommands:\n%%s\nAdditional help:\n    %%s SERVICE [ENDPOINT] --help\n\nExample:\n%%s\n`, os.Args[0], os.Args[0], indent(strings.Join(usageCommands, \"\\n\")), os.Args[0], indent(%sUsageExamples()))\n", defaultUsage)
	fmt.Fprintf(&b, "}\n\n")
	fmt.Fprintf(&b, "func indent(s string) string {\n")
	fmt.Fprintf(&b, "\tif s == \"\" {\n")
	fmt.Fprintf(&b, "\t\treturn \"\"\n")
	fmt.Fprintf(&b, "\t}\n")
	fmt.Fprintf(&b, "\treturn \"    \" + strings.ReplaceAll(s, \"\\n\", \"\\n    \")\n")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

func renderServerMain(server *Data, svcData []*service.Data, apiPkg, interPkg string, hasInterceptors bool, root *expr.RootExpr) string {
	var b strings.Builder
	usesHostOverrideFlags := serverHasRenderedURIs(server)
	fmt.Fprintf(&b, "func main() {\n")
	fmt.Fprintf(&b, "\t%s\n", codegen.Comment("Define command line flags, add any other flag required to configure the service."))
	fmt.Fprintf(&b, "\tvar(\n")
	fmt.Fprintf(&b, "\t\thostF = flag.String(%q, %q, %q)\n", "host", server.DefaultHost().Name, fmt.Sprintf("Server host (valid values: %s)", strings.Join(server.AvailableHosts(), ", ")))
	if usesHostOverrideFlags {
		fmt.Fprintf(&b, "\t\tdomainF = flag.String(%q, %q, %q)\n", "domain", "", "Host domain name (overrides host domain specified in service design)")
	}
	for _, t := range server.Transports {
		fmt.Fprintf(&b, "\t\t%sPortF = flag.String(%q, %q, %q)\n", t.Type, string(t.Type)+"-port", "", fmt.Sprintf("%s port (overrides host %s port specified in service design)", t.Name, t.Name))
	}
	for _, v := range server.Variables {
		desc := v.Description
		if len(v.Values) > 0 {
			desc += fmt.Sprintf(" (valid values: %s)", strings.Join(v.Values, ", "))
		}
		fmt.Fprintf(&b, "\t\t%sF = flag.String(%q, %q, %q)\n", v.VarName, v.Name, v.DefaultValue, desc)
	}
	if usesHostOverrideFlags {
		fmt.Fprintf(&b, "\t\tsecureF = flag.Bool(%q, false, %q)\n", "secure", "Use secure scheme (https or grpcs)")
	}
	fmt.Fprintf(&b, "\t\tdbgF  = flag.Bool(%q, false, %q)\n", "debug", "Log request and response bodies")
	fmt.Fprintf(&b, "\t)\n")
	fmt.Fprintf(&b, "\tflag.Parse()\n\n")

	fmt.Fprintf(&b, "\t%s\n", codegen.Comment("Setup logger. Replace logger with your own log package of choice."))
	fmt.Fprintf(&b, "\tformat := log.FormatJSON\n")
	fmt.Fprintf(&b, "\tif log.IsTerminal() {\n\t\tformat = log.FormatTerminal\n\t}\n")
	fmt.Fprintf(&b, "\tctx := log.Context(context.Background(), log.WithFormat(format))\n")
	fmt.Fprintf(&b, "\tif *dbgF {\n\t\tctx = log.Context(ctx, log.WithDebug())\n\t\tlog.Debugf(ctx, \"debug logs enabled\")\n\t}\n")
	if len(server.Transports) > 0 {
		fmt.Fprintf(&b, "\tlog.Print(ctx, log.KV{K: %q, V: *httpPortF})\n", "http-port")
	}
	fmt.Fprintf(&b, "\n")

	writeServiceInit(&b, apiPkg, svcData)
	writeServerInterceptorInit(&b, interPkg, svcData, hasInterceptors)
	writeServerEndpointsInit(&b, svcData)

	fmt.Fprintf(&b, "\t// Create channel used by both the signal handler and server goroutines\n")
	fmt.Fprintf(&b, "\t// to notify the main goroutine when to stop the server.\n")
	fmt.Fprintf(&b, "\terrc := make(chan error)\n\n")
	fmt.Fprintf(&b, "\t// Setup interrupt handler. This optional step configures the process so\n")
	fmt.Fprintf(&b, "\t// that SIGINT and SIGTERM signals cause the services to stop gracefully.\n")
	fmt.Fprintf(&b, "\tgo func() {\n")
	fmt.Fprintf(&b, "\t\tc := make(chan os.Signal, 1)\n")
	fmt.Fprintf(&b, "\t\tsignal.Notify(c, syscall.SIGINT, syscall.SIGTERM)\n")
	fmt.Fprintf(&b, "\t\terrc <- fmt.Errorf(\"%%s\", <-c)\n")
	fmt.Fprintf(&b, "\t}()\n\n")
	fmt.Fprintf(&b, "\tvar wg sync.WaitGroup\n")
	fmt.Fprintf(&b, "\tctx, cancel := context.WithCancel(ctx)\n\n")

	fmt.Fprintf(&b, "\t%s\n", codegen.Comment("Start the servers and send errors (if any) to the error channel."))
	fmt.Fprintf(&b, "\tswitch *hostF {\n")
	for _, h := range server.Hosts {
		fmt.Fprintf(&b, "\tcase %q:\n", h.Name)
		for _, u := range h.URIs {
			if !server.HasTransport(u.Transport.Type) {
				continue
			}
			fmt.Fprintf(&b, "\t\t{\n")
			fmt.Fprintf(&b, "\t\t\taddr := %q\n", u.URL)
			writeVariableReplacement(&b, h.Variables, true)
			fmt.Fprintf(&b, "\t\t\tu, err := url.Parse(addr)\n")
			fmt.Fprintf(&b, "\t\t\tif err != nil {\n\t\t\t\tlog.Fatalf(ctx, err, %q, addr)\n\t\t\t}\n", "invalid URL %#v\n")
			fmt.Fprintf(&b, "\t\t\tif *secureF {\n\t\t\t\tu.Scheme = %q\n\t\t\t}\n", string(u.Transport.Type)+"s")
			fmt.Fprintf(&b, "\t\t\tif *domainF != \"\" {\n\t\t\t\tu.Host = *domainF\n\t\t\t}\n")
			fmt.Fprintf(&b, "\t\t\tif *%sPortF != \"\" {\n", u.Transport.Type)
			fmt.Fprintf(&b, "\t\t\t\th, _, err := net.SplitHostPort(u.Host)\n")
			fmt.Fprintf(&b, "\t\t\t\tif err != nil {\n\t\t\t\t\tlog.Fatalf(ctx, err, %q, u.Host)\n\t\t\t\t}\n", "invalid URL %#v\n")
			fmt.Fprintf(&b, "\t\t\t\tu.Host = net.JoinHostPort(h, *%sPortF)\n", u.Transport.Type)
			fmt.Fprintf(&b, "\t\t\t} else if u.Port() == \"\" {\n")
			fmt.Fprintf(&b, "\t\t\t\tu.Host = net.JoinHostPort(u.Host, %q)\n", u.Port)
			fmt.Fprintf(&b, "\t\t\t}\n")
			fmt.Fprintf(&b, "\t\t\thandle%sServer(ctx, u", strings.ToUpper(u.Transport.Name))
			for _, arg := range u.HandlerArgs {
				if arg.Endpoint != "" {
					fmt.Fprintf(&b, ", %s", arg.Endpoint)
				}
				if arg.Service != "" {
					fmt.Fprintf(&b, ", %s", arg.Service)
				}
			}
			fmt.Fprintf(&b, ", &wg, errc, *dbgF)\n")
			fmt.Fprintf(&b, "\t\t}\n")
		}
	}
	fmt.Fprintf(&b, "\tdefault:\n")
	fmt.Fprintf(&b, "\t\tlog.Fatal(ctx, fmt.Errorf(%q, *hostF))\n", fmt.Sprintf("invalid host argument: %%q (valid hosts: %s)", strings.Join(server.AvailableHosts(), "|")))
	fmt.Fprintf(&b, "\t}\n\n")

	fmt.Fprintf(&b, "\t%s\n", codegen.Comment("Wait for signal."))
	fmt.Fprintf(&b, "\tlog.Printf(ctx, \"exiting (%%v)\", <-errc)\n\n")
	fmt.Fprintf(&b, "\t%s\n", codegen.Comment("Send cancellation signal to the goroutines."))
	fmt.Fprintf(&b, "\tcancel()\n\n")
	fmt.Fprintf(&b, "\twg.Wait()\n")
	fmt.Fprintf(&b, "\tlog.Printf(ctx, \"exited\")\n")
	fmt.Fprintf(&b, "}\n")
	return b.String()
}

func serverHasRenderedURIs(server *Data) bool {
	for _, host := range server.Hosts {
		for _, uri := range host.URIs {
			if server.HasTransport(uri.Transport.Type) {
				return true
			}
		}
	}
	return false
}

func writeVariableReplacement(b *strings.Builder, vars []*VariableData, fatal bool) {
	for _, v := range vars {
		if len(v.Values) > 0 {
			fmt.Fprintf(b, "\t\t\t\tvar %sSeen bool\n", v.VarName)
			fmt.Fprintf(b, "\t\t\t\t{\n")
			fmt.Fprintf(b, "\t\t\t\t\tfor _, v := range []string{")
			for _, value := range v.Values {
				fmt.Fprintf(b, "%q,", value)
			}
			fmt.Fprintf(b, "} {\n")
			fmt.Fprintf(b, "\t\t\t\t\t\tif v == *%sF {\n", v.VarName)
			fmt.Fprintf(b, "\t\t\t\t\t\t\t%sSeen = true\n", v.VarName)
			fmt.Fprintf(b, "\t\t\t\t\t\t\tbreak\n")
			fmt.Fprintf(b, "\t\t\t\t\t\t}\n")
			fmt.Fprintf(b, "\t\t\t\t\t}\n")
			fmt.Fprintf(b, "\t\t\t\t}\n")
			fmt.Fprintf(b, "\t\t\t\tif !%sSeen {\n", v.VarName)
			if fatal {
				fmt.Fprintf(b, "\t\t\t\t\tlog.Fatal(ctx, fmt.Errorf(%q, *%sF))\n", fmt.Sprintf("invalid value for URL '%s' variable: %%q (valid values: %s)\n", v.Name, strings.Join(v.Values, ",")), v.VarName)
			} else {
				fmt.Fprintf(b, "\t\t\t\t\tfmt.Fprintf(os.Stderr, %q, *%sF)\n", fmt.Sprintf("invalid value for URL '%s' variable: %%q (valid values: %s)\n", v.Name, strings.Join(v.Values, ",")), v.VarName)
				fmt.Fprintf(b, "\t\t\t\t\tos.Exit(1)\n")
			}
			fmt.Fprintf(b, "\t\t\t\t}\n")
		}
		fmt.Fprintf(b, "\t\t\t\taddr = strings.ReplaceAll(addr, %q, *%sF)\n", fmt.Sprintf("{%s}", v.Name), v.VarName)
	}
}

func writeServiceInit(b *strings.Builder, apiPkg string, services []*service.Data) {
	if !mustInitServices(services) {
		return
	}
	fmt.Fprintf(b, "\t%s\n", codegen.Comment("Initialize the services."))
	fmt.Fprintf(b, "\tvar (\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sSvc %s.Service\n", svc.VarName, svc.PkgName)
	}
	fmt.Fprintf(b, "\t)\n")
	fmt.Fprintf(b, "\t{\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sSvc = %s.New%s()\n", svc.VarName, apiPkg, svc.StructName)
	}
	fmt.Fprintf(b, "\t}\n\n")
}

func writeServerInterceptorInit(b *strings.Builder, interPkg string, services []*service.Data, hasInterceptors bool) {
	if !mustInitServices(services) || !hasInterceptors {
		return
	}
	fmt.Fprintf(b, "\t%s\n", codegen.Comment("Initialize the interceptors."))
	fmt.Fprintf(b, "\tvar (\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 || len(svc.ServerInterceptors) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sInterceptors %s.ServerInterceptors\n", svc.VarName, svc.PkgName)
	}
	fmt.Fprintf(b, "\t)\n")
	fmt.Fprintf(b, "\t{\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 || len(svc.ServerInterceptors) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sInterceptors = %s.New%sServerInterceptors()\n", svc.VarName, interPkg, svc.StructName)
	}
	fmt.Fprintf(b, "\t}\n\n")
}

func writeServerEndpointsInit(b *strings.Builder, services []*service.Data) {
	if !mustInitServices(services) {
		return
	}
	fmt.Fprintf(b, "\t%s\n", codegen.Comment("Wrap the services in endpoints that can be invoked from other services potentially running in different processes."))
	fmt.Fprintf(b, "\tvar (\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sEndpoints *%s.Endpoints\n", svc.VarName, svc.PkgName)
	}
	fmt.Fprintf(b, "\t)\n")
	fmt.Fprintf(b, "\t{\n")
	for _, svc := range services {
		if len(svc.Methods) == 0 {
			continue
		}
		fmt.Fprintf(b, "\t\t%sEndpoints = %s.NewEndpoints(%sSvc", svc.VarName, svc.PkgName, svc.VarName)
		if len(svc.ServerInterceptors) > 0 {
			fmt.Fprintf(b, ", %sInterceptors", svc.VarName)
		}
		fmt.Fprintf(b, ")\n")
	}
	fmt.Fprintf(b, "\t}\n\n")
}
