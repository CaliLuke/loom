package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllocateDesignNamesEntryStable(t *testing.T) {
	for _, tc := range []struct {
		name, entry, external, wantEntry, wantExternal string
		scheme                                         bool
	}{
		{"schemaWords", "pet_store", "PetStore", "PetStore", "PetStore2", false},
		{"schemaSecurityWords", "Foo_Security", "FooSecurity", "FooSecurity", "FooSecurity2", false},
		{"schemeVariable", "Foo", "FooSecurity", "ImportedFooSecurity", "FooSecurity2", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document := &Document{OpenAPIVersion: "3.2.0"}
			if tc.scheme {
				document.Components.SecuritySchemes = []SecurityScheme{{Name: tc.entry, Type: "http", Scheme: "bearer"}}
			} else {
				document.Components.Schemas = []NamedSchema{{Name: tc.entry, Schema: &Schema{Type: "string"}}}
			}
			require.NoError(t, allocateDesignNames(document))
			document.Components.Schemas = append(document.Components.Schemas, NamedSchema{Name: tc.external, Schema: &Schema{Type: "string"}, origin: declarationOrigin{introduced: true, order: 1}})
			for range 2 {
				require.NoError(t, allocateDesignNames(document))
				for _, schema := range document.Components.Schemas {
					if schema.origin.introduced {
						require.Equal(t, tc.wantExternal, schema.GoName)
					} else {
						require.Equal(t, tc.wantEntry, schema.GoName)
					}
				}
				if tc.scheme {
					require.Equal(t, tc.wantEntry, document.Components.SecuritySchemes[0].GoName)
				}
			}
		})
	}
}

func TestAllocateDesignNamesErrorClones(t *testing.T) {
	document := nameInventoryErrors()
	require.NoError(t, allocateDesignNames(document))
	want := document.names.clones[cloneIdentity{"GET", "/items", "401"}]
	require.Equal(t, cloneNames{typeName: "Status401", variable: "ImportedStatus401Error"}, want)
	document.Components.Schemas = append(document.Components.Schemas,
		NamedSchema{Name: "Status401Error", Schema: &Schema{Type: "string"}, origin: declarationOrigin{introduced: true, order: 1}},
		NamedSchema{Name: "Status401", Schema: &Schema{Type: "string"}, origin: declarationOrigin{introduced: true, order: 2}},
	)
	require.NoError(t, allocateDesignNames(document))
	require.Equal(t, want, document.names.clones[cloneIdentity{"GET", "/items", "401"}])
	got := make(map[string]string)
	for _, schema := range document.Components.Schemas {
		got[schema.Name] = schema.GoName
	}
	require.Equal(t, "Status401Error2", got["Status401Error"])
	require.Equal(t, "Status4012", got["Status401"])
}

func TestAllocateDesignNamesNoDuplicates(t *testing.T) {
	document := nameInventoryErrors()
	document.Components.SecuritySchemes = []SecurityScheme{{Name: "Status401Error", Type: "http", Scheme: "bearer"}}
	document.Components.Schemas = append(document.Components.Schemas, NamedSchema{Name: "Status401ErrorSecurity", Schema: &Schema{Type: "string"}, origin: declarationOrigin{introduced: true, order: 1}})
	require.NoError(t, allocateDesignNames(document))
	types, variables := make(map[string]bool), make(map[string]bool)
	claim := func(namespace map[string]bool, name string) {
		t.Helper()
		key := strings.ToLower(name)
		require.False(t, namespace[key], "duplicate declaration %s", name)
		namespace[key] = true
	}
	for _, schema := range document.Components.Schemas {
		claim(types, schema.GoName)
		claim(variables, "Imported"+schema.GoName)
	}
	for _, scheme := range document.Components.SecuritySchemes {
		claim(variables, scheme.GoName)
	}
	for _, clone := range document.names.clones {
		claim(types, clone.typeName)
		claim(variables, clone.variable)
	}
}

func TestErrorCloneInventoryResponseOrigins(t *testing.T) {
	document := nameInventoryErrors()
	document.Components.Responses = []NamedResponse{{Name: "Denied", Response: document.Operations[0].Responses[1].Response}}
	document.Operations[0].Responses[1].Response = Response{Ref: "#/components/responses/Denied"}
	document.bundle = &bundledDescription{origins: provenance{
		"#/components/responses/Denied": {Document: "root:errors.yaml", Pointer: "#/Denied"},
	}}
	document.bundle.indexDeclarationOrigins()
	inventory, err := errorCloneInventory(document)
	require.NoError(t, err)
	require.Len(t, inventory, 2)
	for _, clone := range inventory {
		require.True(t, clone.origin.introduced, "all responses contributing to the clone decision determine its origin")
	}
}

func TestErrorCloneInventoryMatchesRenderPlan(t *testing.T) {
	document := nameInventoryErrors()
	require.NoError(t, allocateDesignNames(document))
	plan := planDocument(document)
	require.Empty(t, plan.diagnostics)
	inventory, err := errorCloneInventory(document)
	require.NoError(t, err)
	require.Len(t, inventory, 2)
	for _, clone := range inventory {
		found := false
		for i, operation := range plan.operations {
			for _, response := range operation.failures {
				identity := cloneIdentity{document.Operations[i].Method, document.Operations[i].Path, response.status}
				if identity == clone.identity {
					require.True(t, response.cloneErrorType)
					require.Equal(t, clone.name, response.errorName)
					found = true
				}
			}
		}
		require.True(t, found)
	}
}

func TestErrorCloneInventorySyntheticHeader(t *testing.T) {
	document := nameInventoryErrors()
	// A media-range response adds Content-Type even without an authored header.
	// That header means the response owns a wrapper rather than an error clone.
	for i := 1; i < len(document.Operations[0].Responses); i++ {
		document.Operations[0].Responses[i].Response.Schema.MediaTypes = []string{"application/*"}
	}
	inventory, err := errorCloneInventory(document)
	require.NoError(t, err)
	require.Empty(t, inventory)
}

func nameInventoryErrors() *Document {
	return &Document{OpenAPIVersion: "3.2.0", Components: Components{Schemas: []NamedSchema{
		{Name: "Failure", Schema: &Schema{Type: "object", Properties: []NamedProperty{{Name: "message", Schema: &Schema{Type: "string"}}}}},
	}}, Operations: []Operation{{Method: "GET", Path: "/items", GoName: "List", Responses: []StatusResponse{
		{Status: "200", Response: Response{Description: "OK"}},
		{Status: "401", Response: Response{Description: "Unauthorized", ContentType: "application/json", Schema: &Schema{Ref: "#/components/schemas/Failure"}}},
		{Status: "403", Response: Response{Description: "Forbidden", ContentType: "application/json", Schema: &Schema{Ref: "#/components/schemas/Failure"}}},
	}}}}
}
