package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSchemaGoNamesAreUnique(t *testing.T) {
	document, _, err := Analyze([]byte(`openapi: 3.1.1
info: {title: t, version: "1"}
paths:
  /a:
    get:
      operationId: getUser
      parameters:
        - {name: value, in: query, schema: {type: string}}
        - {name: value2, in: query, schema: {type: string}}
        - {name: Value, in: query, schema: {type: string}}
      responses:
        "200":
          description: ok
          headers:
            Body: {schema: {type: string}}
            body2: {schema: {type: string}}
          content:
            application/json:
              schema: {$ref: '#/components/schemas/Pet'}
  /b:
    get:
      operationId: get_user
      responses: {"204": {description: ok}}
  /c:
    get:
      operationId: getUser2
      responses: {"204": {description: ok}}
components:
  schemas:
    Pet: {type: object, properties: {a: {type: string}}}
    Pet2: {type: object, properties: {a: {type: string}}}
    pet: {type: object, properties: {a: {type: string}}}
`))
	require.NoError(t, err)
	design, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	require.Equal(t, 1, strings.Count(string(design), "var ImportedPet2 ="), string(design))
	methods := map[string]string{}
	for _, operation := range document.Operations {
		other, exists := methods[operation.GoName]
		require.False(t, exists, "operations %s and %s share Go name %s", other, operation.Path, operation.GoName)
		methods[operation.GoName] = operation.Path
	}
	requireRenderedDesignGenerates(t, design)
}

func TestImportedNameConsumers(t *testing.T) {
	t.Run("operation identities", func(t *testing.T) {
		operations := []Operation{
			{Path: "/a", Method: "GET", OperationID: "getUser"},
			{Path: "/b", Method: "GET", OperationID: "get_user"},
			{Path: "/c", Method: "GET", OperationID: "getUser2"},
		}
		assignOperationNames(operations)
		require.Equal(t, "GetUser", operations[0].GoName)
		require.Equal(t, "GetUser2", operations[1].GoName)
		require.Equal(t, "GetUser22", operations[2].GoName)
	})
	t.Run("security declarations", func(t *testing.T) {
		schemes := []SecurityScheme{{Name: "Auth"}, {Name: "auth"}}
		assignSecuritySchemeNames(schemes, []NamedSchema{{GoName: "AuthSecurity2"}})
		require.Equal(t, "ImportedAuthSecurity", schemes[0].GoName)
		require.Equal(t, "ImportedAuthSecurity3", schemes[1].GoName)
	})
	t.Run("request and credential fields", func(t *testing.T) {
		r := renderer{document: &Document{
			SecurityDefined: true,
			Security:        SecurityRequirements{{Schemes: []SecurityRequirementScheme{{Name: "Auth"}}}},
			Components: Components{SecuritySchemes: []SecurityScheme{
				{Name: "Auth", Type: "apiKey", In: "header", ParameterName: "X-Auth"},
			}},
		}}
		source := make([]Parameter, 0, 5)
		for _, name := range []string{"value", "value2", "Value", "authCredential", "authCredential2"} {
			source = append(source, Parameter{Name: name, In: "query", Schema: &Schema{Type: "string"}})
		}
		parameters, err := r.parameters(source, "#/parameters")
		require.NoError(t, err)
		require.Equal(t, "value3", parameters[2].field)
		for i, parameter := range parameters {
			require.Equal(t, source[i].Name, parameter.parameter.Name, "wire names remain authored")
		}
		parameters, err = r.securityParameters(&Operation{}, parameters, "#/security")
		require.NoError(t, err)
		require.Equal(t, "authCredential3", parameters[len(parameters)-1].field)
		require.Equal(t, "X-Auth", parameters[len(parameters)-1].parameter.Name)
		_, err = r.parameters([]Parameter{
			{Name: "userID", In: "path", Required: true, Schema: &Schema{Type: "string"}},
			{Name: "user_id", In: "path", Required: true, Schema: &Schema{Type: "string"}},
		}, "#/parameters")
		require.ErrorContains(t, err, "colliding Go field name")
	})
	t.Run("response headers and body", func(t *testing.T) {
		r := renderer{}
		response, err := r.renderedResponse(StatusResponse{Status: "200", Response: Response{
			Schema: &Schema{Type: "string"}, ContentType: "application/json",
			Headers: []NamedHeader{
				{Name: "Body", Header: Header{Schema: &Schema{Type: "string"}}},
				{Name: "body2", Header: Header{Schema: &Schema{Type: "string"}}},
			},
		}}, false, "#/response")
		require.NoError(t, err)
		require.Equal(t, "body3", response.body)
		require.Equal(t, "Body", response.headers[0].name)
		require.Equal(t, "body2", response.headers[1].name)
	})
	t.Run("cloned error declarations", func(t *testing.T) {
		r := renderer{document: &Document{Components: Components{Schemas: []NamedSchema{
			{GoName: "Status400Error"}, {GoName: "Status400Error2"},
		}}}, operations: []operationPlan{{failures: []renderedResponse{
			{cloneErrorType: true, errorName: "Status400"},
		}}}}
		r.assignErrorCloneNames()
		require.Equal(t, "ImportedStatus400Error3", r.operations[0].failures[0].cloneErrorGoName)
		require.Equal(t, "Status400", r.operations[0].failures[0].cloneErrorTypeName)
	})
}

func TestImportedNameOccupancyPermutations(t *testing.T) {
	for _, bases := range [][]string{
		{"Pet", "pet", "Pet2"}, {"Pet", "Pet2", "pet"},
		{"pet", "Pet", "Pet2"}, {"pet", "Pet2", "Pet"},
		{"Pet2", "Pet", "pet"}, {"Pet2", "pet", "Pet"},
	} {
		t.Run(strings.Join(bases, "/"), func(t *testing.T) {
			used := make(map[string]int)
			seen := make(map[string]bool)
			for _, base := range bases {
				name := uniqueName(base, used)
				key := strings.ToLower(name)
				require.False(t, seen[key], "every emitted candidate owns its case-folded name")
				seen[key] = true
			}
		})
	}
}

func TestImportedNameCandidateOccupancy(t *testing.T) {
	for _, tc := range []struct {
		name        string
		bases, want []string
	}{
		{"occupied base", []string{"Pet", "Pet2", "Pet"}, []string{"Pet", "Pet2", "Pet3"}},
		{"occupied suffix", []string{"Pet", "Pet", "Pet2"}, []string{"Pet", "Pet2", "Pet22"}},
		{"case folding", []string{"Pet", "pet2", "PET"}, []string{"Pet", "pet2", "PET3"}},
		{"empty fallback", []string{"Value", "", "Value2"}, []string{"Value", "Value2", "Value22"}},
		{"ordinary", []string{"Pet", "Dog", "Pet"}, []string{"Pet", "Dog", "Pet2"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			used := make(map[string]int)
			got := make([]string, 0, len(tc.bases))
			for _, base := range tc.bases {
				got = append(got, uniqueName(base, used))
			}
			require.Equal(t, tc.want, got)
		})
	}
}
