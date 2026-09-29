package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestEffectiveErrorBinding(t *testing.T) {
	for _, test := range []struct {
		name, mapping, local, want string
	}{
		{"API reference", "method", "", "api"},
		{"service reference", "method", "service", "service"},
		{"method reference", "method", "method", "method"},
		{"API mapping shadows local", "api", "method", "api"},
		{"service mapping shadows local", "service", "method", "service"},
	} {
		for _, emitter := range []bool{false, true} {
			mode := "schema"
			if emitter {
				mode = "emitter"
			}
			t.Run(test.name+"/"+mode, func(t *testing.T) {
				root := codegen.RunDSL(t, effectiveErrorDSL(test.mapping, test.local))
				var semantic *service.Data
				var existing *service.ValueData
				if emitter {
					semantic = service.NewServicesData(root).Get("errors")
					existing = semantic.Method("show").ErrorValues["failure"]
				}
				root.API.ExampleGenerator = expr.NewRandom("error-capture")
				control := expr.NewRandom("error-capture")
				registered := len(*expr.GeneratedResultTypes)
				prepared, err := PrepareService(root.API.HTTP.Services[0], semantic)
				require.NoError(t, err)
				require.Equal(t, control.Int(), root.API.ExampleGenerator.Int())
				require.Len(t, *expr.GeneratedResultTypes, registered)
				response := prepared.Endpoints[0].Response.ErrorResponses[0]
				require.NotNil(t, response.BodyValue)
				require.NoError(t, response.BodyValue.Error)
				source := response.BodyValue.Source
				require.NotNil(t, source)
				members := source.Occurrence.Members()
				names := make([]string, 0, len(members))
				for _, member := range members {
					names = append(names, member.Name)
				}
				require.Contains(t, names, test.want+"_marker")
				if test.mapping == "method" {
					require.Equal(t, []string{"data"}, response.BodyValue.Selection)
				} else {
					require.Empty(t, response.BodyValue.Selection)
				}
				require.Same(t, source, response.Headers[0].Value.Source)
				require.NoError(t, response.Headers[0].Value.Error)
				require.Same(t, source, response.Cookies[0].Value.Source)
				require.NoError(t, response.Cookies[0].Value.Error)
				require.Same(t, source, response.DocumentValue.Source)
				require.NoError(t, response.DocumentValue.Error)
				for _, target := range response.BodyValues {
					require.Same(t, source, target.Source)
					require.NoError(t, target.Error)
				}
				if emitter {
					require.Same(t, existing, semantic.Method("show").ErrorValues["failure"])
					if test.want == test.local {
						require.Same(t, existing, source)
					} else {
						require.Zero(t, source.Example.Outcome(), "missing error capture must not sample")
						if existing != nil {
							require.Same(t, existing.Context, source.Context)
						}
					}
				} else {
					require.Zero(t, source.Example.Outcome())
				}
			})
		}
	}
}

func effectiveErrorDSL(mapping, local string) func() {
	return func() {
		api := effectiveErrorType("api")
		serviceError := effectiveErrorType("service")
		methodError := effectiveErrorType("method")
		dsl.API("error-source", func() {
			dsl.Error("failure", api)
			if mapping == "api" {
				dsl.HTTP(func() {
					effectiveErrorResponse(false)
				})
			}
		})
		dsl.Service("errors", func() {
			if local == "service" || mapping == "service" {
				dsl.Error("failure", serviceError)
			}
			if mapping == "service" {
				dsl.HTTP(func() {
					effectiveErrorResponse(false)
				})
			}
			dsl.Method("show", func() {
				if local == "method" {
					dsl.Error("failure", methodError)
				}
				dsl.HTTP(func() {
					dsl.GET("/")
					if mapping == "method" {
						effectiveErrorResponse(true)
					}
				})
			})
		})
	}
}

func effectiveErrorType(name string) expr.UserType {
	return dsl.Type(name+"Failure", func() {
		dsl.Attribute("data", dsl.Bytes)
		dsl.Attribute("header", dsl.String)
		dsl.Attribute("cookie", dsl.String)
		dsl.Attribute(name+"_marker", dsl.Boolean)
	})
}

func effectiveErrorResponse(selected bool) {
	dsl.Response("failure", dsl.StatusBadRequest, func() {
		if selected {
			dsl.Body("data")
		}
		dsl.Header("header:X-Failure")
		dsl.Cookie("cookie:failure")
	})
}
