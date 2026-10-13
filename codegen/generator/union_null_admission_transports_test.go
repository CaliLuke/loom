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
	object := dsl.Type("NullableObject", func() {
		dsl.Attribute("text", dsl.String, func() {
			dsl.MinLength(1)
		})
		dsl.Required("text")
		dsl.Nullable()
	})
	nested := dsl.Type("Nested", dsl.OneOf(dsl.String, dsl.Int))
	inheritedNested := dsl.Type("NullableNested", nested, func() {
		dsl.Nullable()
	})
	holder := dsl.Type("Holder", func() {
		dsl.OneOf("branch", func() {
			dsl.Attribute("nullable", dsl.String, func() {
				dsl.Nullable()
			})
			dsl.Attribute("inherited", inherited)
			dsl.Attribute("object", object)
			dsl.Attribute("nested", nested, func() {
				dsl.Nullable()
			})
			dsl.Attribute("inherited_nested", inheritedNested)
			dsl.Attribute("anything", dsl.Any)
			dsl.Attribute("text", dsl.String)
		})
		dsl.Required("branch")
	})
	report := dsl.ResultType("application/vnd.nulladmission", "Report", func() {
		dsl.Extend(holder)
		dsl.Attribute("label", dsl.String)
		dsl.View("default", func() {
			dsl.Attribute("branch")
			dsl.Attribute("label")
		})
		dsl.View("tiny", func() {
			dsl.Attribute("label")
		})
	})
	collection := dsl.CollectionOf(report)
	envelope := dsl.ResultType("application/vnd.nulladmission.envelope", "Envelope", func() {
		dsl.Attribute("report", report)
		dsl.View("default", func() {
			dsl.Attribute("report", func() {
				dsl.View("tiny")
			})
		})
	})
	dsl.Service("admission", func() {
		dsl.Method("list", func() {
			dsl.Result(collection)
			dsl.HTTP(func() {
				dsl.GET("/list")
			})
		})
		dsl.Method("family", func() {
			dsl.Result(envelope)
			dsl.HTTP(func() {
				dsl.GET("/family")
			})
		})
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
		dsl.Method("list", func() {
			dsl.Result(collection)
			dsl.JSONRPC(func() {
			})
		})
		dsl.Method("family", func() {
			dsl.Result(envelope)
			dsl.JSONRPC(func() {
			})
		})
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
	for _, branch := range []string{"nullable", "inherited", "object", "nested", "inherited_nested", "anything", "text"} {
		values := []string{"null", " \n null \t", "\"value\"", "42", ""}
		switch branch {
		case "anything":
			values = append(values, "{\"nested\":[true,null]}")
		case "object":
			values = append(values, "{\"text\":\"value\"}")
		case "nested", "inherited_nested":
			values = append(values, "{\"type\":\"String\",\"value\":\"value\"}")
		}
		for _, value := range values {
			valid := value != "" && (branch == "anything" ||
				((branch == "nullable" || branch == "inherited" || branch == "text") && value == "\"value\"") ||
				(branch == "object" && value == "{\"text\":\"value\"}") ||
				((branch == "nested" || branch == "inherited_nested") && value == "{\"type\":\"String\",\"value\":\"value\"}") ||
				(branch != "text" && (value == "null" || value == " \n null \t")))
			envelope := "{\"type\":\"" + branch + "\""
			if value != "" {
				envelope += ",\"value\":" + value
			}
			wire := []byte("{\"branch\":" + envelope + "}}")
			for _, target := range []struct {
				name  string
				value any
			}{
				{"service payload", &service.Holder{}},
				{"service result", &service.Report{}},
				{"projected result", &views.ReportView{}},
				{"HTTP client result", &client.ShowResponseBody{}},
				{"HTTP server payload", &server.ShowRequestBody{}},
				{"JSONRPC service payload", &rpcservice.Holder{}},
				{"JSONRPC service result", &rpcservice.Report{}},
				{"JSONRPC projected result", &rpcviews.ReportView{}},
				{"JSONRPC client result", &rpcclient.ShowResponseBody{}},
				{"JSONRPC server payload", &rpcserver.ShowRequestBody{}},
				{"HTTP projected result", &client.ProjectedResponseBody{}},
				{"JSONRPC projected result", &rpcclient.ProjectedResponseBody{}},
			} {
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

func TestSelectedViewNullableBranchConstraints(t *testing.T) {
	for _, target := range []struct {
		name      string
		construct func([]byte, string) error
	}{
		{"HTTP result", func(wire []byte, view string) error {
			var value service.Report
			if err := json.Unmarshal(wire, &value); err != nil {
				return err
			}
			_, err := service.NewViewedReport(&value, view)
			return err
		}},
		{"HTTP collection", func(wire []byte, view string) error {
			var value service.Report
			if err := json.Unmarshal(wire, &value); err != nil {
				return err
			}
			_, err := service.NewViewedReportCollection(service.ReportCollection{&value}, view)
			return err
		}},
		{"JSONRPC result", func(wire []byte, view string) error {
			var value rpcservice.Report
			if err := json.Unmarshal(wire, &value); err != nil {
				return err
			}
			_, err := rpcservice.NewViewedReport(&value, view)
			return err
		}},
		{"JSONRPC collection", func(wire []byte, view string) error {
			var value rpcservice.Report
			if err := json.Unmarshal(wire, &value); err != nil {
				return err
			}
			_, err := rpcservice.NewViewedReportCollection(rpcservice.ReportCollection{&value}, view)
			return err
		}},
	} {
		for _, tc := range []struct {
			name  string
			value string
			view  string
			valid bool
		}{
			{"null", "null", "default", true},
			{"valid object", "{\"text\":\"value\"}", "default", true},
			{"invalid object", "{\"text\":\"\"}", "default", false},
			{"excluded invalid object", "{\"text\":\"\"}", "tiny", true},
			{"unknown view", "null", "unknown", false},
		} {
			t.Run(target.name+"/"+tc.name, func(t *testing.T) {
				wire := []byte("{\"label\":\"ok\",\"branch\":{\"type\":\"object\",\"value\":" + tc.value + "}}")
				err := target.construct(wire, tc.view)
				if tc.valid {
					require.NoError(t, err)
				} else {
					require.Error(t, err)
				}
			})
		}
	}
}

func TestNestedSelectedViewHelper(t *testing.T) {
	wire := []byte("{\"label\":\"ok\",\"branch\":{\"type\":\"object\",\"value\":{\"text\":\"\"}}}")
	var value service.Report
	require.NoError(t, json.Unmarshal(wire, &value))
	projected, err := service.NewViewedEnvelope(&service.Envelope{Report: &value}, "default")
	require.NoError(t, err)
	require.Empty(t, projected.Projected.Report.Branch.Kind())
	var rpcValue rpcservice.Report
	require.NoError(t, json.Unmarshal(wire, &rpcValue))
	rpcProjected, err := rpcservice.NewViewedEnvelope(&rpcservice.Envelope{Report: &rpcValue}, "default")
	require.NoError(t, err)
	require.Empty(t, rpcProjected.Projected.Report.Branch.Kind())
}
`
