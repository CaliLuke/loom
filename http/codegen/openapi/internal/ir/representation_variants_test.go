package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

func TestByteRepresentationsOwnCompleteComponents(t *testing.T) {
	var firstNamed map[string]*Schema
	for _, reversed := range []bool{false, true} {
		root := codegen.RunDSL(t, func() {
			blob := dsl.Type("Blob", dsl.Bytes, func() {
				dsl.Meta("openapi:typename", "PublicBlob")
				dsl.Meta("openapi:typename:canonical", "true")
				dsl.MinLength(2)
				dsl.MaxLength(2)
			})
			dsl.Service("storage", func() {
				names := []string{"json", "multipart", "query"}
				if reversed {
					names = []string{"query", "multipart", "json"}
				}
				for _, name := range names {
					dsl.Method(name, func() {
						dsl.Payload(func() {
							dsl.Attribute("data", blob)
						})
						dsl.HTTP(func() {
							dsl.POST("/" + name)
							switch name {
							case "multipart":
								dsl.MultipartRequest()
							case "query":
								dsl.Param("data")
							}
						})
					})
				}
			})
		})
		doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
		require.NoError(t, err)
		jsonBody := representationSchema(t, doc, doc.Paths["/json"].Operations["POST"].RequestBody.Value.Content["application/json"].Schema)
		multipartBody := representationSchema(t, doc, doc.Paths["/multipart"].Operations["POST"].RequestBody.Value.Content["multipart/form-data"].Schema)
		jsonField := jsonBody.Properties["data"]
		require.Equal(t, "#/components/schemas/PublicBlob", jsonField.Ref)
		jsonBytes := representationSchema(t, doc, jsonField)
		require.Equal(t, "base64", jsonBytes.ContentEncoding)
		require.Empty(t, jsonBytes.Format)
		require.Nil(t, jsonBytes.MaxLength, "decoded byte bounds must not remain encoded string bounds")
		multipartBytes := representationSchema(t, doc, multipartBody.Properties["data"])
		require.Empty(t, multipartBytes.ContentEncoding)
		require.Equal(t, "binary", multipartBytes.Format)
		require.Equal(t, 2, *multipartBytes.MaxLength)
		query := doc.Paths["/query"].Operations["POST"].Parameters[0].Value
		require.NotNil(t, query)
		queryBytes := representationSchema(t, doc, query.Schema)
		require.Empty(t, queryBytes.ContentEncoding)
		require.Equal(t, 2, *queryBytes.MaxLength)
		// The parent renderer allocates anonymous body annotations from the first
		// encountered body: JSONRequestBody forward, MultipartRequestBody reverse.
		// Representation splitting must preserve that authority for both variants.
		bodyExample := "Znk="
		if reversed {
			bodyExample = "YW8="
		}
		for _, body := range []*Schema{jsonBody, multipartBody} {
			require.Equal(t, map[string]any{"data": bodyExample}, body.Example)
		}
		named := make(map[string]*Schema)
		for name, schema := range doc.Components.Schemas {
			if strings.HasPrefix(name, "PublicBlob") {
				named[name] = schema
			}
		}
		if firstNamed == nil {
			firstNamed = named
		} else {
			require.Equal(t, firstNamed, named, "registration order cannot choose the named public representation")
		}
	}
}

func TestByteRepresentationKeepsCustomSibling(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Service("storage", func() {
			dsl.Method("store", func() {
				dsl.Payload(func() {
					dsl.Attribute("builtin", dsl.Bytes, func() {
						dsl.MaxLength(2)
					})
					dsl.Attribute("custom", dsl.Bytes, func() {
						dsl.Meta("struct:field:type", "custom.Blob", "example.com/custom")
						dsl.MaxLength(2)
					})
				})
				dsl.HTTP(func() {
					dsl.POST("/")
				})
			})
		})
	})
	doc, err := BuildDocument(root.API, root.Types, root.ResultTypes)
	require.NoError(t, err)
	body := representationSchema(t, doc, doc.Paths["/"].Operations["POST"].RequestBody.Value.Content["application/json"].Schema)
	require.Equal(t, "base64", body.Properties["builtin"].ContentEncoding)
	require.Empty(t, body.Properties["custom"].ContentEncoding)
	require.Equal(t, 2, *body.Properties["custom"].MaxLength)
}

func representationSchema(t *testing.T, doc *Document, schema *Schema) *Schema {
	t.Helper()
	seen := make(map[string]bool)
	for schema != nil && schema.Ref != "" {
		require.False(t, seen[schema.Ref], "unexpected alias-only component cycle")
		seen[schema.Ref] = true
		schema = doc.Components.Schemas[strings.TrimPrefix(schema.Ref, "#/components/schemas/")]
	}
	require.NotNil(t, schema)
	return schema
}
