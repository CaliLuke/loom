package representation

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestResponseMetadataCarrierAuthority(t *testing.T) {
	for _, backend := range []bool{false, true} {
		name := "schema"
		if backend {
			name = "emitter"
		}
		t.Run(name, func(t *testing.T) {
			root := codegen.RunDSL(t, responseMetadataDSL)
			owner := service.NewSemanticData(root)
			method := owner.Method(root.Services[0].Methods[0])
			var native *service.Data
			if backend {
				native = owner.Services().Get("events")
				method = native.Method("watch")
			}
			prepared, err := PrepareService(root.API.HTTP.Services[0], native, owner)
			require.NoError(t, err)
			endpoint := prepared.Endpoints[0]
			opening := endpoint.Response.Responses[0]
			require.NotNil(t, opening.Headers[0].Value)
			require.NoError(t, opening.Headers[0].Value.Error)
			value := method.HTTPMetadataValues[opening.Occurrence]
			require.NotNil(t, value)
			require.Equal(t, value.Occurrence.ID(), opening.Headers[0].Value.Source.Occurrence.ID())
			require.NotEqual(t, endpoint.Stream.ResponseValue.Source.Occurrence.ID(), value.Occurrence.ID())
			require.Equal(t, method.StreamingResultValue.Occurrence.ID(), endpoint.Stream.ResponseValue.Source.Occurrence.ID())
			require.NotNil(t, opening.StatusField.Value)
			require.Equal(t, value.Occurrence.ID(), opening.StatusField.Value.Source.Occurrence.ID())
			require.Nil(t, opening.BodyValue)
			info := endpoint.Response.Informational[0].Response
			require.NoError(t, info.Headers[0].Value.Error)
			require.Equal(t, method.HTTPMetadataValues[info.Occurrence].Occurrence.ID(), info.Headers[0].Value.Source.Occurrence.ID())
			require.NotEqual(t, value.Occurrence.ID(), info.Headers[0].Value.Source.Occurrence.ID())
		})
	}
}

func TestResponseMetadataSchemaPreparation(t *testing.T) {
	root := codegen.RunDSL(t, responseMetadataDSL)
	root.API.ExampleGenerator = expr.NewRandom("metadata-schema-only")
	control := expr.NewRandom("metadata-schema-only")
	prepared, err := PrepareService(root.API.HTTP.Services[0], nil, nil)
	require.NoError(t, err)
	for _, status := range []*expr.HTTPResponseExpr{root.API.HTTP.Services[0].HTTPEndpoints[0].Responses[0], root.API.HTTP.Services[0].HTTPEndpoints[0].Informational[0].HTTPResponseExpr} {
		var value *service.ValueData
		for _, response := range responseOccurrences(prepared.Endpoints[0].Response) {
			if response.Occurrence == status {
				value = response.Headers[0].Value.Source
			}
		}
		require.NotNil(t, value)
		require.Zero(t, value.Example.Outcome())
	}
	require.Equal(t, control.Int(), root.API.ExampleGenerator.Int())
}

func responseMetadataDSL() {
	opening := dsl.Type("Opening", func() {
		dsl.Attribute("status", dsl.Int)
		dsl.Attribute("session", dsl.String)
		dsl.Required("status", "session")
	})
	hints := dsl.Type("Hints", func() {
		dsl.Attribute("session", dsl.String)
		dsl.Required("session")
	})
	dsl.Service("events", func() {
		dsl.Method("watch", func() {
			dsl.StreamingResult(func() {
				dsl.Attribute("status", dsl.String)
				dsl.Attribute("session", dsl.Int)
				dsl.Required("status", "session")
			})
			dsl.HTTP(func() {
				dsl.GET("/events")
				dsl.ServerSentEvents()
				dsl.Informational("hints", dsl.StatusEarlyHints, hints, func() {
					dsl.Header("session:X-Hint")
				})
				dsl.Response(dsl.StatusCreated, opening, func() {
					dsl.StatusField("status")
					dsl.Header("session:X-Session")
				})
			})
		})
	})
}
