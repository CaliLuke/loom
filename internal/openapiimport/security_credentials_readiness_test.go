package openapiimport

import (
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pb33f/libopenapi"
	"github.com/stretchr/testify/require"

	exec "github.com/CaliLuke/loom/internal/testprocess"
)

func TestSecurityCredentialReadinessAllocation(t *testing.T) {
	bearer := SecurityScheme{Name: "Bearer", Type: "http", Scheme: "bearer"}
	oauth := SecurityScheme{Name: "OAuth", Type: "oauth2"}
	basic := SecurityScheme{Name: "Basic", Type: "http", Scheme: "basic"}
	key := func(name, location, wire string) SecurityScheme {
		return SecurityScheme{Name: name, Type: "apiKey", In: location, ParameterName: wire}
	}
	for _, tc := range []struct {
		name    string
		schemes []SecurityScheme
		and     bool
		want    int
	}{
		{"bearer OAuth OR", []SecurityScheme{bearer, oauth}, false, 1},
		{"bearer OAuth AND", []SecurityScheme{bearer, oauth}, true, 1},
		{"Basic bearer OAuth OR", []SecurityScheme{basic, bearer, oauth}, false, 3},
		{"two bearer roles", []SecurityScheme{bearer, {Name: "Second", Type: "http", Scheme: "bearer"}}, false, 1},
		{"two Basic roles", []SecurityScheme{basic, {Name: "Second", Type: "http", Scheme: "basic"}}, false, 2},
		{"same header case folded", []SecurityScheme{key("One", "header", "X-Key"), key("Two", "header", "x-key")}, true, 1},
		{"same cookie", []SecurityScheme{key("One", "cookie", "key"), key("Two", "cookie", "key")}, false, 1},
		{"same query", []SecurityScheme{key("One", "query", "key"), key("Two", "query", "key")}, true, 1},
		{"different header", []SecurityScheme{key("One", "header", "X-One"), key("Two", "header", "X-Two")}, true, 2},
		{"different location", []SecurityScheme{key("One", "query", "key"), key("Two", "cookie", "key")}, true, 2},
		{"query case sensitive", []SecurityScheme{key("One", "query", "key"), key("Two", "query", "Key")}, true, 2},
		{"cookie case sensitive", []SecurityScheme{key("One", "cookie", "key"), key("Two", "cookie", "Key")}, true, 2},
		{"different framing", []SecurityScheme{basic, bearer}, false, 3},
		{"raw and bearer remain distinct", []SecurityScheme{key("Raw", "header", "Authorization"), bearer}, false, 2},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var requirements SecurityRequirements
			for _, scheme := range tc.schemes {
				ref := SecurityRequirementScheme{Name: scheme.Name}
				if tc.and && len(requirements) > 0 {
					requirements[0].Schemes = append(requirements[0].Schemes, ref)
				} else {
					requirements = append(requirements, SecurityRequirement{Schemes: []SecurityRequirementScheme{ref}})
				}
			}
			r := renderer{document: &Document{SecurityDefined: true, Security: requirements, Components: Components{SecuritySchemes: tc.schemes}}}
			got, err := r.securityParameters(&Operation{}, nil, "#/security")
			require.NoError(t, err)
			require.Len(t, got, tc.want, "physical credential fields must not be allocated independently per semantic scheme")
		})
	}
}

func TestSecurityCredentialReadinessScopeControls(t *testing.T) {
	scheme := SecurityScheme{Name: "Global", Type: "http", Scheme: "bearer"}
	r := renderer{document: &Document{SecurityDefined: true, Security: SecurityRequirements{{Schemes: []SecurityRequirementScheme{{Name: "Global"}}}}, Components: Components{SecuritySchemes: []SecurityScheme{scheme}}}}
	ordinary := []renderedParameter{{field: "globalToken", parameter: Parameter{Name: "q", In: "query", Schema: &Schema{Type: "string"}}}}
	for _, tc := range []struct {
		name      string
		operation Operation
		want      int
	}{
		{"inherit", Operation{}, 2},
		{"explicit none", Operation{SecurityDefined: true}, 1},
		{"anonymous alternative", Operation{SecurityDefined: true, Security: SecurityRequirements{{}}}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := r.securityParameters(&tc.operation, ordinary, "#/security")
			require.NoError(t, err)
			require.Len(t, got, tc.want)
			require.Equal(t, ordinary[0], got[0])
			if len(got) > 1 {
				require.NotEqual(t, ordinary[0].field, got[1].field)
			}
		})
	}
}

func TestSecurityCredentialReadinessCandidateGenerated(t *testing.T) {
	source, err := os.ReadFile("testdata/security_credentials/candidate.go.txt")
	require.NoError(t, err)
	dir := requireRenderedDesignGenerates(t, source)
	harness, err := os.ReadFile("testdata/security_credentials/wire.go.txt")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "wire_test.go"), harness, 0o600))
	for _, args := range [][]string{{"test", "-mod=mod", "-v", "-run", "^TestCandidateAuthWire$", "./..."}, {"build", "./..."}, {"vet", "./..."}} {
		cmd := exec.CommandContext(t.Context(), "go", args...)
		cmd.Dir = dir
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, string(output))
		t.Log(string(output))
	}
	generated, err := os.ReadFile(filepath.Join(dir, "gen", "http", "openapi.json"))
	require.NoError(t, err)
	parsed, err := libopenapi.NewDocument(generated)
	require.NoError(t, err)
	_, err = parsed.BuildV3Model()
	require.NoError(t, err)
	var contract map[string]any
	require.NoError(t, json.Unmarshal(generated, &contract))
	for path, want := range map[string]any{
		"/either":      []any{map[string]any{"Basic": []any{}}, map[string]any{"Bearer": []any{}}, map[string]any{"OAuth": []any{"read"}}},
		"/both":        []any{map[string]any{"Bearer": []any{}, "OAuth": []any{}}},
		"/keys":        []any{map[string]any{"One": []any{}, "Two": []any{}}},
		"/scoped_both": []any{map[string]any{"Bearer": []any{}, "OAuth": []any{"read"}}},
	} {
		require.Equal(t, want, operationFromImportedSpec(t, contract, path, "get")["security"])
	}

	cmd := exec.CommandContext(t.Context(), "go", "test", "-run", "^TestCandidateMixedScopeSourceContract$", "-v", ".")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
	t.Log("Per-scheme source scope contract:", string(output))
}

func TestSecurityCredentialReadinessScopeAdmission(t *testing.T) {
	for _, tc := range []struct {
		requirement string
		unsupported bool
	}{
		{"{Bearer: [], OAuth: [read]}", true},
		{"{OAuth: [read], Bearer: []}", true},
		{"{OAuth: [], Other: [read]}", true},
		{"{OAuth: [read], Other: []}", true},
		{"{OAuth: [read], Other: [read]}", false},
		{"{Bearer: [], OAuth: []}", false},
		{"{Bearer: []}, {OAuth: [read]}", false},
	} {
		t.Run(tc.requirement, func(t *testing.T) {
			source := []byte(`openapi: 3.2.0
info: {title: Scope fidelity, version: "1"}
security: [REQUIREMENT]
paths:
 /secure:
  get:
   responses: {"204": {description: done}}
components:
 securitySchemes:
  Bearer: {type: http, scheme: bearer}
  OAuth:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read}}
  Other:
   type: oauth2
   flows:
    clientCredentials: {tokenUrl: https://example.com/token, scopes: {read: Read}}
`)
			source = []byte(strings.ReplaceAll(string(source), "REQUIREMENT", tc.requirement))
			_, diagnostics, err := Analyze(source)
			require.NoError(t, err)
			if tc.unsupported {
				require.Len(t, diagnostics, 1)
				require.Equal(t, "security-requirement", diagnostics[0].Code)
				require.Equal(t, "#/security/0", diagnostics[0].Path)
				require.Contains(t, diagnostics[0].Message, "use different scopes")
				fatal, warnings := diagnostics.Classify(true)
				require.Equal(t, diagnostics, fatal)
				require.Empty(t, warnings)
			} else {
				require.Empty(t, diagnostics)
			}
		})
	}
}

func TestSecurityCredentialRoles(t *testing.T) {
	document := &Document{
		SecurityDefined: true,
		Security: SecurityRequirements{
			{Schemes: []SecurityRequirementScheme{{Name: "JWT"}}},
			{Schemes: []SecurityRequirementScheme{{Name: "OAuth", Scopes: []string{"read"}}}},
			{Schemes: []SecurityRequirementScheme{{Name: "JWT"}, {Name: "OAuth"}}},
		},
		Components: Components{SecuritySchemes: []SecurityScheme{
			{Name: "JWT", GoName: "JWT", Type: "http", Scheme: "bearer"},
			{Name: "OAuth", GoName: "OAuth", Type: "oauth2"},
		}},
	}
	before, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	r := renderer{document: document}
	fields, err := r.securityParameters(&Operation{}, nil, "#/security")
	require.NoError(t, err)
	require.Len(t, fields, 1)
	require.Equal(t, "jWTtoken", fields[0].field)
	require.Equal(t, []credentialRole{{scheme: "JWT", kind: "token"}, {scheme: "OAuth", kind: "accessToken"}}, fields[0].securityRoles)
	require.NoError(t, r.payload(fields, nil, "#/payload"))
	require.NoError(t, r.securityRequirements(document.Security))
	require.Contains(t, r.builder.String(), `Token("jWTtoken", String, func() {`)
	require.Contains(t, r.builder.String(), `Meta("security:accesstoken")`)
	require.Contains(t, r.builder.String(), "Security(JWT)\nSecurity(OAuth, func()")
	require.Contains(t, r.builder.String(), "Security(JWT, OAuth)")
	after, err := json.Marshal(document, json.Deterministic(true))
	require.NoError(t, err)
	require.Equal(t, before, after)
}
