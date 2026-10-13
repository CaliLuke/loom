package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

// TestServicePackageIdentityCompiles compiles generated service, transport and
// example files through the existing complete module helper.
func TestServicePackageIdentityCompiles(t *testing.T) {
	cases := []struct {
		name      string
		api       string
		service   string
		transport string
	}{
		{name: "http-service", api: "demo", service: "init", transport: "http"},
		{name: "grpc-service", api: "demo", service: "init", transport: "grpc"},
		{name: "rpc-service", api: "demo", service: "init", transport: "rpc"},
		{name: "api-package", api: "init", service: "ordinary", transport: "http"},
		{name: "api-and-service", api: "init", service: "init", transport: "http"},
		{name: "ordinary-control", api: "demo", service: "ordinary", transport: "http"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			generateExampleModule(t, "example.com/packageidentity", func() {
				dsl.API(c.api, func() {
					if c.transport == "rpc" {
						dsl.JSONRPC(func() {
						})
					}
				})
				dsl.Service(c.service, func() {
					if c.transport == "rpc" {
						dsl.JSONRPC(func() {
							dsl.POST("/rpc")
						})
					}
					dsl.Method("echo", func() {
						dsl.Payload(func() {
							dsl.Field(1, "message", dsl.String)
						})
						dsl.Result(dsl.String)
						switch c.transport {
						case "http":
							dsl.HTTP(func() {
								dsl.POST("/echo")
							})
						case "grpc":
							dsl.GRPC(func() {
							})
						case "rpc":
							dsl.JSONRPC(func() {
							})
						}
					})
				})
			})
		})
	}
}
