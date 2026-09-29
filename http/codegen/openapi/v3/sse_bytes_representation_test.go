package openapiv3

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type sseNamedBytes []byte

func TestSSEMappedBytesRetainRawRepresentation(t *testing.T) {
	for _, test := range []struct {
		name     string
		named    bool
		nullable bool
		whole    bool
		value    any
		encoded  string
		base64   bool
	}{
		{name: "native mapped", value: []byte("hi"), encoded: "hi"},
		{name: "named mapped", named: true, value: sseNamedBytes("hi"), encoded: `"aGk="`, base64: true},
		{name: "nullable mapped", nullable: true, value: loom.NullableValue([]byte("hi")), encoded: `"aGk="`, base64: true},
		{name: "native whole event", whole: true, value: []byte("hi"), encoded: `"aGk="`, base64: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := codegen.RunDSL(t, sseBytesDesign(test.named, test.nullable, test.whole))
			encode := loomhttp.EncodeSSEData
			if test.whole {
				encode = loomhttp.EncodeSSEJSONData
			}
			encoded, err := encode(test.value)
			require.NoError(t, err)
			require.Equal(t, test.encoded, encoded)
			spec := New(root)
			require.NotNil(t, spec)
			data := ssePhysicalDataSchema(t, spec)
			if test.nullable {
				require.Len(t, data.AnyOf, 2)
				require.Equal(t, openapi.Type(openapi.Null), data.AnyOf[1].Type)
				data = data.AnyOf[0]
			}
			if test.base64 {
				require.Equal(t, "base64", data.ContentEncoding)
				require.Empty(t, data.Format)
				require.Nil(t, data.MaxLength)
			} else {
				require.Empty(t, data.ContentEncoding, "raw SSE data must not acquire a JSON base64 constraint")
				require.Equal(t, "binary", data.Format)
				require.Equal(t, 2, *data.MaxLength)
			}
		})
	}
}

func sseBytesDesign(named, nullable, whole bool) func() {
	return func() {
		var bytesType expr.DataType = expr.Bytes
		if named {
			bytesType = dsl.Type("SSEBlob", dsl.Bytes, func() {
				dsl.Meta("openapi:typename", "SSEBlob")
				dsl.Meta("openapi:typename:canonical", "true")
				dsl.MaxLength(2)
			})
		}
		dsl.Service("raw-byte-events", func() {
			dsl.Method("watch", func() {
				if whole {
					dsl.StreamingResult(dsl.Bytes, func() {
						dsl.MaxLength(2)
					})
				} else {
					dsl.StreamingResult(func() {
						dsl.Attribute("data", bytesType, func() {
							if !named {
								dsl.MaxLength(2)
							}
							if nullable {
								dsl.Nullable()
							}
						})
						dsl.Attribute("event", dsl.String)
						dsl.Required("data", "event")
					})
				}
				dsl.HTTP(func() {
					dsl.GET("/events")
					dsl.ServerSentEvents(func() {
						if !whole {
							dsl.SSEEventData("data")
							dsl.SSEEventType("event")
						}
					})
				})
			})
		})
	}
}

func ssePhysicalDataSchema(t *testing.T, spec *OpenAPI) *openapi.Schema {
	t.Helper()
	response := spec.Paths["/events"].Get.Responses["200"]
	if response.Ref != "" {
		response = spec.Components.Responses[response.Ref[len("#/components/responses/"):]]
	}
	media := response.Value.Content["text/event-stream"]
	require.NotNil(t, media.ItemSchema)
	data := media.ItemSchema.Properties["data"].ContentSchema
	require.NotNil(t, data)
	if data.Ref != "" {
		data = spec.Components.Schemas[data.Ref[len("#/components/schemas/"):]]
	}
	return data
}
