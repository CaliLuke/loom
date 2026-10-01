package openapiimport

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRenderClonesSharedSchemaUsedByDifferentErrors(t *testing.T) {
	tests := []struct {
		name       string
		schemas    string
		wantClones bool
		wantFields []string
	}{
		{
			name: "object",
			schemas: `    ErrorResponse:
      type: object
      properties:
        message:
          type: string`,
			wantClones: true,
			wantFields: []string{"message"},
		},
		{
			name: "base-only object",
			schemas: `    ErrorResponse:
      allOf:
        - $ref: "#/components/schemas/ErrorBase"
    ErrorBase:
      type: object
      properties:
        message:
          type: string`,
			wantClones: true,
			wantFields: []string{"message"},
		},
		{
			name: "map",
			schemas: `    ErrorResponse:
      type: object
      additionalProperties:
        type: string`,
		},
		{
			name: "alias to map",
			schemas: `    ErrorResponse:
      $ref: "#/components/schemas/ErrorValues"
    ErrorValues:
      type: object
      additionalProperties:
        type: string`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(fmt.Sprintf(`openapi: 3.1.0
info:
  title: Shared Errors
  version: "1"
paths:
  /items:
    get:
      operationId: getItems
      responses:
        "200":
          description: OK
        "401":
          description: Unauthorized
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
        "403":
          description: Forbidden
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
components:
  schemas:
%s
`, test.schemas)))
			require.NoError(t, err)
			require.Empty(t, diagnostics)

			source, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			design := string(source)
			if test.wantClones {
				require.Contains(t, design, `var ImportedStatus401Error = Type("Status401", func() {`)
				require.Contains(t, design, `var ImportedStatus403Error = Type("Status403", func() {`)
				require.Contains(t, design, `Error("Status401", ImportedStatus401Error)`)
				require.Contains(t, design, `Error("Status403", ImportedStatus403Error)`)
				require.Equal(t, 2, strings.Count(design, `Extend(ImportedErrorResponse)`))
			} else {
				require.NotContains(t, design, `Extend(ImportedErrorResponse)`)
			}
			require.Equal(t, 2, strings.Count(design, `OpenAPIBody(ImportedErrorResponse)`))
			require.NotContains(t, design, `Body("body")`)
			requireRenderedDesignEvaluates(t, source, 1)
			moduleDir := requireRenderedDesignGenerates(t, source)
			if test.wantClones {
				requireRenderedErrorFields(t, moduleDir, map[string][]string{
					"Status401": test.wantFields,
					"Status403": test.wantFields,
				})
			}
		})
	}
}

func TestRenderClonesSharedSchemaUsedBySameStatusAcrossOperations(t *testing.T) {
	document, diagnostics, err := Analyze([]byte(`openapi: 3.1.0
info:
  title: Shared Status Errors
  version: "1"
paths:
  /items:
    get:
      operationId: listItems
      responses:
        "200":
          description: OK
        "400":
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
    post:
      operationId: createItem
      responses:
        "201":
          description: Created
        "400":
          description: Invalid request
          content:
            application/json:
              schema:
                $ref: "#/components/schemas/ErrorResponse"
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        message:
          type: string
`))
	require.NoError(t, err)
	require.Empty(t, diagnostics)

	source, err := Render(document, Options{PackageName: "design"})
	require.NoError(t, err)
	design := string(source)
	for _, errorName := range []string{"ListItemsStatus400", "CreateItemStatus400"} {
		cloneName := "Imported" + errorName + "Error"
		require.Contains(t, design, `var `+cloneName+` = Type("`+errorName+`", func() {`)
		require.Contains(t, design, `Error("`+errorName+`", `+cloneName+`)`)
	}
	require.Equal(t, 2, strings.Count(design, `Extend(ImportedErrorResponse)`))
	moduleDir := requireRenderedDesignGenerates(t, source)
	requireRenderedErrorFields(t, moduleDir, map[string][]string{
		"ListItemsStatus400":  {"message"},
		"CreateItemStatus400": {"message"},
	})
	requireRenderedDesignEvaluates(t, source, 2)
}

func TestRenderSharedErrorCloneNameAvoidsComponentCollision(t *testing.T) {
	tests := []struct {
		name          string
		component     string
		cloneGoName   string
		cloneTypeName string
	}{
		{
			name:          "Go identifier",
			component:     "Status401Error",
			cloneGoName:   "ImportedStatus401Error2",
			cloneTypeName: "Status401",
		},
		{
			name:          "generated service type",
			component:     "Status401",
			cloneGoName:   "ImportedStatus4012Error",
			cloneTypeName: "Status4012",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			document, diagnostics, err := Analyze([]byte(fmt.Sprintf(`openapi: 3.1.0
info: {title: Collision, version: "1"}
paths:
  /items:
    get:
      operationId: getItems
      responses:
        "200": {description: OK}
        "401":
          description: Unauthorized
          content:
            application/json:
              schema: {$ref: "#/components/schemas/ErrorResponse"}
        "403":
          description: Forbidden
          content:
            application/json:
              schema: {$ref: "#/components/schemas/ErrorResponse"}
components:
  schemas:
    ErrorResponse:
      type: object
      properties:
        message: {type: string}
    %s:
      type: string
`, test.component)))
			require.NoError(t, err)
			require.Empty(t, diagnostics)

			source, err := Render(document, Options{PackageName: "design"})
			require.NoError(t, err)
			design := string(source)
			require.Contains(t, design, fmt.Sprintf(`var %s = Type(%q, func() {`, test.cloneGoName, test.cloneTypeName))
			require.Contains(t, design, `Error("Status401", `+test.cloneGoName+`)`)
			moduleDir := requireRenderedDesignGenerates(t, source)
			requireRenderedErrorFields(t, moduleDir, map[string][]string{
				"Status401": {"message"},
				"Status403": {"message"},
			})
		})
	}
}

func requireRenderedErrorFields(t *testing.T, moduleDir string, expected map[string][]string) {
	t.Helper()
	designDir := filepath.Join(moduleDir, "design")

	entries := make([]string, 0, len(expected))
	for name, fields := range expected {
		sort.Strings(fields)
		quoted := make([]string, len(fields))
		for index, field := range fields {
			quoted[index] = strconv.Quote(field)
		}
		entries = append(entries, fmt.Sprintf("%q: {%s}", name, strings.Join(quoted, ", ")))
	}
	sort.Strings(entries)
	testSource := fmt.Sprintf(`package design

import (
	"slices"
	"sort"
	"testing"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestImportedErrorFields(t *testing.T) {
	if err := expr.RegisterDefaultRoots(); err != nil {
		t.Fatal(err)
	}
	if err := eval.RunDSL(); err != nil {
		t.Fatal(err)
	}
	expected := map[string][]string{%s}
	for name, want := range expected {
		var failure *expr.ErrorExpr
		for _, service := range expr.Root.Services {
			for _, method := range service.Methods {
				if failure = method.Error(name); failure != nil {
					break
				}
			}
			if failure != nil {
				break
			}
		}
		if failure == nil {
			t.Fatalf("missing error %%q", name)
		}
		object := expr.AsObject(failure.Type)
		if object == nil {
			t.Fatalf("error %%q is not an object", name)
		}
		got := make([]string, 0, len(*object))
		for _, field := range *object {
			got = append(got, field.Name)
		}
		sort.Strings(got)
		if !slices.Equal(got, want) {
			t.Errorf("error %%q fields = %%v, want %%v", name, got, want)
		}
	}
}
`, strings.Join(entries, ", "))
	require.NoError(t, os.WriteFile(filepath.Join(designDir, "design_test.go"), []byte(testSource), 0o600))

	cmd := exec.Command("go", "test", "-mod=mod", "./design")
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, string(output))
}
