package openapiv3

import (
	"encoding/json/v2"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/openapi"
)

func TestAsyncSchemaReferenceLifecycle(t *testing.T) {
	for _, direction := range []string{"inbound", "outbound"} {
		for _, shared := range []bool{false, true} {
			name := direction + "/async-only"
			if shared {
				name = direction + "/shared-ordinary"
			}
			t.Run(name, func(t *testing.T) {
				misleading := map[string]any{"$ref": toSchemaComponentRef("DataOnly")}
				root := &openapi.Schema{
					Ref: toSchemaComponentRef("FirstAlias"), Description: "Retained occurrence annotation.",
					Not: &openapi.Schema{Ref: toSchemaComponentRef("ForbiddenAlias")}, Example: misleading,
				}
				op := &Operation{Extensions: map[string]any{
					openAPIAsyncExtension: map[string]any{"messages": map[string]any{
						direction:                   map[string]any{"schema": root, "example": misleading},
						"not-a-framework-direction": map[string]any{"schema": &openapi.Schema{Ref: toSchemaComponentRef("DataOnly")}},
					}},
					"x-user-data": map[string]any{"schema": &openapi.Schema{Ref: toSchemaComponentRef("DataOnly")}},
				}}
				if shared {
					op.RequestBody = &RequestBodyRef{Value: &RequestBody{Content: map[string]*MediaType{
						"application/json": {Schema: &openapi.Schema{Ref: toSchemaComponentRef("FirstAlias")}},
					}}}
				}
				paths := map[string]*PathItem{"/stream": {Get: op}}
				schemas := map[string]*openapi.Schema{
					"FirstAlias":     {Ref: toSchemaComponentRef("SecondAlias")},
					"SecondAlias":    {Ref: toSchemaComponentRef("Message")},
					"ForbiddenAlias": {Ref: toSchemaComponentRef("Forbidden")},
					"Forbidden":      {Type: openapi.String, Enum: []any{"forbidden"}},
					"DataOnly":       {Type: openapi.String},
					"Message": {Type: openapi.Object, Properties: map[string]*openapi.Schema{
						"next": {Ref: toSchemaComponentRef("Message")},
						"data": {Type: openapi.String, ContentEncoding: "base64"},
					}},
				}
				collapseSchemaAliases(paths, schemas, reusableComponents{})
				require.Equal(t, toSchemaComponentRef("Message"), root.Ref)
				require.Equal(t, toSchemaComponentRef("Forbidden"), root.Not.Ref)
				require.Equal(t, "Retained occurrence annotation.", root.Description)
				require.Equal(t, toSchemaComponentRef("DataOnly"), misleading["$ref"], "example data is not a schema edge")
				if shared {
					require.Equal(t, root.Ref, op.RequestBody.Value.Content["application/json"].Schema.Ref)
				}
				retained := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{})
				require.Len(t, retained, 2)
				require.Contains(t, retained, "Message")
				require.Contains(t, retained, "Forbidden")
				require.Equal(t, toSchemaComponentRef("Message"), retained["Message"].Properties["next"].Ref)
				require.Equal(t, "base64", retained["Message"].Properties["data"].ContentEncoding)
			})
		}
	}
}

func TestAsyncPureReferenceSerialization(t *testing.T) {
	ref := toSchemaComponentRef("Message")
	typed, err := json.Marshal(&openapi.Schema{Ref: ref})
	require.NoError(t, err)
	legacy, err := json.Marshal(map[string]any{"$ref": ref})
	require.NoError(t, err)
	require.Equal(t, legacy, typed)
}
