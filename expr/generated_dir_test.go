package expr_test

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestGeneratedDirCollisions(t *testing.T) {
	const deseret = "\U00010428" // DESERET SMALL LETTER LONG I, escaped as U00010428
	cases := []struct {
		Name     string
		Servers  []string
		Services []string
		Errors   []string
	}{
		{Name: "distinct servers", Servers: []string{"calc", "adder"}},
		{Name: "accented server and its ASCII spelling", Servers: []string{"Café", "cafe"}},
		{Name: "distinct escaped servers", Servers: []string{"Café", "Cafè"}},
		{
			Name:    "escaped server and its escape",
			Servers: []string{"Café", "cafu00e9"},
			Errors:  []string{`Server cafu00e9: servers "Café" and "cafu00e9" both use the generated directory name "cafu00e9" (as in cmd/cafu00e9); rename one of them`},
		},
		{
			Name:    "space and underscore servers",
			Servers: []string{"calc server", "calc_server"},
			Errors:  []string{`Server calc_server: servers "calc server" and "calc_server" both use the generated directory name "calc_server" (as in cmd/calc_server); rename one of them`},
		},
		{
			Name:    "servers that differ in ASCII case",
			Servers: []string{"Calc", "calc"},
			Errors:  []string{`servers "Calc" and "calc" both use the generated directory name "calc"`},
		},
		{
			Name:    "servers whose directories differ only in case",
			Servers: []string{deseret, "u00010428"},
			Errors:  []string{`Server u00010428: servers "` + deseret + `" and "u00010428" use the generated directory names "U00010428" and "u00010428" (as in cmd/U00010428 and cmd/u00010428), which differ only in case; case-insensitive file systems merge them and the Go toolchain rejects import paths that differ only in case, so rename one of them`},
		},
		{
			Name:    "three colliding servers",
			Servers: []string{"calc server", "calc_server", "CalcServer"},
			Errors: []string{
				`servers "calc server" and "calc_server" both use the generated directory name "calc_server"`,
				`servers "calc server" and "CalcServer" both use the generated directory name "calc_server"`,
			},
		},
		{Name: "distinct services", Services: []string{"calc", "calc2"}},
		{Name: "accented service and its ASCII spelling", Services: []string{"Café", "cafe"}},
		{
			Name:     "escaped service and its escape",
			Services: []string{"Café", "cafu00e9"},
			Errors:   []string{`service "cafu00e9": services "Café" and "cafu00e9" both use the generated directory name "cafu00e9" (as in gen/cafu00e9); rename one of them`},
		},
		{
			Name:     "space and underscore services",
			Services: []string{"calc service", "calc_service"},
			Errors:   []string{`services "calc service" and "calc_service" both use the generated directory name "calc_service" (as in gen/calc_service)`},
		},
		{
			Name:     "reserved word service and its suffixed spelling",
			Services: []string{"string", "string_"},
			Errors:   []string{`services "string" and "string_" both use the generated directory name "string_"`},
		},
		{
			Name:     "services whose directories differ only in case",
			Services: []string{deseret, "u00010428"},
			Errors:   []string{`service "u00010428": services "` + deseret + `" and "u00010428" use the generated directory names "U00010428" and "u00010428" (as in gen/U00010428 and gen/u00010428), which differ only in case`},
		},
		{
			Name:     "server and service with the same directory",
			Servers:  []string{"calc"},
			Services: []string{"calc"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			design := generatedDirDSL(c.Servers, c.Services)
			if len(c.Errors) == 0 {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.Error(t, err)
			for _, want := range c.Errors {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

func generatedDirDSL(servers, services []string) func() {
	if len(services) == 0 {
		services = []string{"svc"}
	}
	return func() {
		API("dirs", func() {
			for _, name := range servers {
				Server(name)
			}
		})
		for _, name := range services {
			Service(name, func() {
				Method("show", func() {})
			})
		}
	}
}

// TestGeneratedCLIDirCollisions covers a service whose directory is cli, the
// directory of the client CLI packages of a transport. The generators place
// the transport packages of the service in gen/<transport>/cli/server, client
// and, for gRPC, pb, and the client CLI package of each server that hosts a
// service of the transport in gen/<transport>/cli/<server>.
func TestGeneratedCLIDirCollisions(t *testing.T) {
	cases := []struct {
		Name       string
		Servers    []string
		API        string
		Transports []string
		Hosted     []string
		ServerOnly bool
		Errors     []string
	}{
		{Name: "cli service", Servers: []string{"calc"}, Transports: []string{"http", "grpc"}},
		{Name: "JSON-RPC cli service", Servers: []string{"calc"}, Transports: []string{"jsonrpc", "grpc"}},
		{Name: "cli service and cli server", Servers: []string{"cli"}, Transports: []string{"http", "grpc"}},
		{Name: "pb server without gRPC", Servers: []string{"pb"}, Transports: []string{"http"}},
		{Name: "pb server without gRPC over JSON-RPC", Servers: []string{"pb"}, Transports: []string{"jsonrpc"}},
		{Name: "cli service without transport", Servers: []string{"server"}},
		{
			Name:       "HTTP server package",
			Servers:    []string{"server"},
			Transports: []string{"http"},
			Errors:     []string{`service "cli": service "cli" and server "server" both use the generated directory gen/http/cli/server: the HTTP server package of the service and the HTTP client CLI package of the server; rename the service or the server`},
		},
		{
			Name:       "HTTP client package",
			Servers:    []string{"Client"},
			Transports: []string{"http"},
			Errors:     []string{`service "cli" and server "Client" both use the generated directory gen/http/cli/client: the HTTP client package of the service and the HTTP client CLI package of the server`},
		},
		{
			Name:       "gRPC pb package",
			Servers:    []string{"pb"},
			Transports: []string{"grpc"},
			Errors:     []string{`service "cli" and server "pb" both use the generated directory gen/grpc/cli/pb: the gRPC pb package of the service and the gRPC client CLI package of the server`},
		},
		{
			Name:       "JSON-RPC server package",
			Servers:    []string{"server"},
			Transports: []string{"jsonrpc"},
			Errors:     []string{`service "cli" and server "server" both use the generated directory gen/jsonrpc/cli/server: the JSON-RPC server package of the service and the JSON-RPC client CLI package of the server`},
		},
		{
			Name:       "server that does not host the cli service",
			Servers:    []string{"server"},
			Transports: []string{"grpc"},
			Hosted:     []string{"other"},
			Errors:     []string{`service "cli" and server "server" both use the generated directory gen/grpc/cli/server: the gRPC server package of the service and the gRPC client CLI package of the server`},
		},
		{
			Name:       "default server",
			API:        "server",
			Transports: []string{"grpc"},
			Errors:     []string{`service "cli" and server "server" both use the generated directory gen/grpc/cli/server`},
		},
		{
			Name:       "server-only HTTP generation",
			Servers:    []string{"calc"},
			Transports: []string{"http"},
			ServerOnly: true,
			Errors:     []string{`service "cli": service "cli" uses the generated directory gen/http/cli, which Meta("http:generate", "server") removes as stale HTTP client CLI output; rename the service`},
		},
		{
			Name:       "server-only HTTP generation without an HTTP cli service",
			Servers:    []string{"calc"},
			Transports: []string{"grpc"},
			ServerOnly: true,
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			design := generatedCLIDirDSL(c.API, c.Servers, c.Hosted, c.Transports, c.ServerOnly)
			if len(c.Errors) == 0 {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.Error(t, err)
			for _, want := range c.Errors {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

// generatedCLIDirDSL returns a design with a service named cli and a service
// named other, both exposed over transports, and servers that host the hosted
// services, both when hosted is empty.
func generatedCLIDirDSL(api string, servers, hosted, transports []string, serverOnly bool) func() {
	if api == "" {
		api = "dirs"
	}
	if len(hosted) == 0 {
		hosted = []string{"cli", "other"}
	}
	return func() {
		API(api, func() {
			if serverOnly {
				Meta("http:generate", "server")
			}
			for _, name := range servers {
				Server(name, func() {
					Services(hosted...)
				})
			}
		})
		for _, name := range []string{"cli", "other"} {
			Service(name, func() {
				if slices.Contains(transports, "jsonrpc") {
					JSONRPC(func() {
						POST("/" + name + "/rpc")
					})
				}
				Method("show", func() {
					Payload(func() {
						Attribute("id", Int, func() {
							Meta("rpc:tag", "1")
						})
					})
					for _, transport := range transports {
						switch transport {
						case "http":
							HTTP(func() {
								POST("/" + name)
							})
						case "grpc":
							GRPC(func() {})
						case "jsonrpc":
							JSONRPC(func() {})
						}
					}
				})
			})
		}
	}
}
