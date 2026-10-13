package generator

import (
	"testing"

	dsl "github.com/CaliLuke/loom/dsl"
)

// TestUnionNullAdmissionTransports covers branch contracts not present in the
// physical-carrier specimen: inherited Nullable, Any and projected results.
func TestUnionNullAdmissionTransports(t *testing.T) {
	runDesignHarness(t, "example.com/nulladmission", unionNullAdmissionDSL, unionNullAdmissionHarness)
}

func unionNullAdmissionDSL() {
	dsl.API("nulladmission", func() {
		dsl.JSONRPC(func() {
		})
	})
	inherited := dsl.Type("NullableText", dsl.String, func() {
		dsl.Nullable()
	})
	holder := dsl.Type("Holder", func() {
		dsl.OneOf("branch", func() {
			dsl.Attribute("nullable", dsl.String, func() {
				dsl.Nullable()
			})
			dsl.Attribute("inherited", inherited)
			dsl.Attribute("anything", dsl.Any)
			dsl.Attribute("text", dsl.String)
		})
		dsl.Required("branch")
	})
	// Explicit Nullable projected declarations have a separate naming defect
	// tracked in #829. Any covers projected null admission here; Holder retains
	// explicit and inherited Nullable across the ordinary transport carriers.
	report := dsl.ResultType("application/vnd.nulladmission", "Report", func() {
		dsl.OneOf("branch", func() {
			dsl.Attribute("anything", dsl.Any)
			dsl.Attribute("text", dsl.String)
		})
		dsl.Required("branch")
		dsl.View("default", func() {
			dsl.Attribute("branch")
		})
	})
	dsl.Service("admission", func() {
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.HTTP(func() {
				dsl.POST("/show")
			})
		})
		dsl.Method("projected", func() {
			dsl.Result(report)
			dsl.HTTP(func() {
				dsl.GET("/projected")
			})
		})
	})
	dsl.Service("admissionrpc", func() {
		dsl.JSONRPC(func() {
			dsl.POST("/rpc")
		})
		dsl.Method("show", func() {
			dsl.Payload(holder)
			dsl.Result(holder)
			dsl.JSONRPC(func() {
			})
		})
		dsl.Method("projected", func() {
			dsl.Result(report)
			dsl.JSONRPC(func() {
			})
		})
	})
}

const unionNullAdmissionHarness = `package nulladmission

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	service "example.com/nulladmission/gen/admission"
	views "example.com/nulladmission/gen/admission/views"
	rpcservice "example.com/nulladmission/gen/admissionrpc"
	rpcviews "example.com/nulladmission/gen/admissionrpc/views"
	client "example.com/nulladmission/gen/http/admission/client"
	server "example.com/nulladmission/gen/http/admission/server"
	rpcclient "example.com/nulladmission/gen/jsonrpc/admissionrpc/client"
	rpcserver "example.com/nulladmission/gen/jsonrpc/admissionrpc/server"
)

func TestSemanticBranchAdmission(t *testing.T) {
	for _, branch := range []string{"nullable", "inherited", "anything", "text"} {
		for _, value := range []string{"null", " \n null \t", "\"value\"", "42", "{\"nested\":[true,null]}", ""} {
			valid := value != "" && (branch == "anything" || value == "\"value\"" || (branch != "text" && (value == "null" || value == " \n null \t")))
			envelope := "{\"type\":\"" + branch + "\""
			if value != "" {
				envelope += ",\"value\":" + value
			}
			wire := []byte("{\"branch\":" + envelope + "}}")
			for _, target := range []struct {
				name      string
				value     any
				projected bool
			}{
				{"service payload", &service.Holder{}, false},
				{"service result", &service.Report{}, true},
				{"projected result", &views.ReportView{}, true},
				{"HTTP client result", &client.ShowResponseBody{}, false},
				{"HTTP server payload", &server.ShowRequestBody{}, false},
				{"JSONRPC service payload", &rpcservice.Holder{}, false},
				{"JSONRPC service result", &rpcservice.Report{}, true},
				{"JSONRPC projected result", &rpcviews.ReportView{}, true},
				{"JSONRPC client result", &rpcclient.ShowResponseBody{}, false},
				{"JSONRPC server payload", &rpcserver.ShowRequestBody{}, false},
				{"HTTP projected result", &client.ProjectedResponseBody{}, true},
				{"JSONRPC projected result", &rpcclient.ProjectedResponseBody{}, true},
			} {
				if target.projected && (branch == "nullable" || branch == "inherited") {
					continue
				}
				t.Run(target.name+"/"+branch+"/"+value, func(t *testing.T) {
					err := json.Unmarshal(wire, target.value)
					if !valid {
						require.Error(t, err)
						return
					}
					require.NoError(t, err)
					encoded, err := json.Marshal(target.value)
					require.NoError(t, err)
					require.JSONEq(t, string(wire), string(encoded))
				})
			}
		}
	}
}
`
