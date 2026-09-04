package openapiv3_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/json/v2"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"testing"
	"unicode"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

const (
	openAPIDeterminismHelperDir    = "LOOM_OPENAPI_DETERMINISM_HELPER_DIR"
	openAPIDeterminismHelperTarget = "LOOM_OPENAPI_DETERMINISM_HELPER_TARGET"
	openAPIDeterminismHelperPrefix = "LOOM_OPENAPI_DETERMINISM_HELPER_PREFIX"
	openAPIDeterminismHelperIndent = "LOOM_OPENAPI_DETERMINISM_HELPER_INDENT"
)

func TestFilesSynthesizedExamplesAreStableAcrossRepeatedGeneration(t *testing.T) {
	first := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	second := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))

	require.Equal(t, first, second)
}

func TestFilesAreByteStableAcrossIndependentProcesses(t *testing.T) {
	if outputDir := os.Getenv(openAPIDeterminismHelperDir); outputDir != "" {
		root := httpgen.RunHTTPDSL(t, testdata.OpenAPIReusableComponentsDSL)
		if root.API.Meta == nil {
			root.API.Meta = make(expr.MetaExpr)
		}
		if target := os.Getenv(openAPIDeterminismHelperTarget); target != "" {
			root.API.Meta["openapi:version"] = []string{target}
		}
		if prefix := os.Getenv(openAPIDeterminismHelperPrefix); prefix != "" {
			root.API.Meta["openapi:json:prefix"] = []string{prefix}
		}
		if indent := os.Getenv(openAPIDeterminismHelperIndent); indent != "" {
			root.API.Meta["openapi:json:indent"] = []string{indent}
		}
		files, err := openapiv3.Files(root)
		require.NoError(t, err)
		for _, file := range files {
			_, err = file.Render(outputDir)
			require.NoError(t, err)
		}
		return
	}

	tests := []struct {
		name         string
		target       string
		prefix       string
		indent       string
		wantVersion  string
		wantJSONHash string
		wantYAMLHash string
	}{
		{
			name:         "OpenAPI 3.2 default formatting",
			wantVersion:  openapiv3.OpenAPIVersion,
			wantJSONHash: "b423879a80566c3a25e49a3c5f9a306ec40d246cd8d3764193ee99de31c3ba0e",
			wantYAMLHash: "21bd46034012dc8de7329178f9ec7bcf7ad3c66add8f41d337fc3e2c4732b614",
		},
		{
			name:         "OpenAPI 3.1 compatibility",
			target:       "3.1",
			wantVersion:  openapiv3.OpenAPICompatibilityVersion,
			wantJSONHash: "aa26b861cfecc45f7006c178f06a740956d97d2f628c0c07d3489d77e6ef7b72",
			wantYAMLHash: "758bcec9135e92ad27e6861e7327670c3ac27d43536ee3a9fae40245ef1f3bee",
		},
		{
			name:         "OpenAPI 3.2 configured formatting",
			prefix:       " ",
			indent:       "\t",
			wantVersion:  openapiv3.OpenAPIVersion,
			wantJSONHash: "9b18396f5c38f5d3c364fab25564b299fe1aca60b58c9d77e013075a9a4bda1f",
			wantYAMLHash: "21bd46034012dc8de7329178f9ec7bcf7ad3c66add8f41d337fc3e2c4732b614",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			dirs := []string{t.TempDir(), t.TempDir(), t.TempDir()}
			for _, dir := range dirs {
				cmd := exec.Command(os.Args[0], "-test.run=^TestFilesAreByteStableAcrossIndependentProcesses$")
				cmd.Env = append(
					os.Environ(),
					openAPIDeterminismHelperDir+"="+dir,
					openAPIDeterminismHelperTarget+"="+test.target,
					openAPIDeterminismHelperPrefix+"="+test.prefix,
					openAPIDeterminismHelperIndent+"="+test.indent,
				)
				output, err := cmd.CombinedOutput()
				require.NoErrorf(t, err, "isolated OpenAPI generation failed:\n%s", output)
			}

			for _, artifact := range []struct {
				name     string
				wantHash string
			}{
				{name: "openapi.json", wantHash: test.wantJSONHash},
				{name: "openapi.yaml", wantHash: test.wantYAMLHash},
			} {
				first, err := os.ReadFile(filepath.Join(dirs[0], "gen", "http", artifact.name))
				require.NoError(t, err)
				for _, dir := range dirs[1:] {
					repeated, readErr := os.ReadFile(filepath.Join(dir, "gen", "http", artifact.name))
					require.NoError(t, readErr)
					require.Equal(t, first, repeated, "%s differs across isolated generations", artifact.name)
				}
				require.Equal(t, artifact.wantHash, fmt.Sprintf("%x", sha256.Sum256(first)))
				if filepath.Ext(artifact.name) == ".json" {
					require.True(t, bytes.HasSuffix(first, []byte("\n")), "generated JSON must retain its final newline")
					require.Contains(t, string(first), fmt.Sprintf("\"openapi\": \"%s\"", test.wantVersion))
					if test.indent == "" {
						require.True(t, bytes.HasPrefix(first, []byte("{\n  \"openapi\"")))
					} else {
						require.True(t, bytes.HasPrefix(first, []byte("{\n 	\"openapi\"")))
					}
				}
			}
		})
	}
}

func TestMediaTypeRefHonorsDeterministicNestedMapOrdering(t *testing.T) {
	ref := &openapiv3.MediaTypeRef{Value: &openapiv3.MediaType{Example: map[string]any{
		"zulu":    "z",
		"yankee":  "y",
		"xray":    "x",
		"whiskey": "w",
		"bravo":   "b",
		"alpha":   "a",
	}}}
	const want = `{"example":{"alpha":"a","bravo":"b","whiskey":"w","xray":"x","yankee":"y","zulu":"z"}}`

	for range 20 {
		got, err := json.Marshal(ref, json.Deterministic(true))
		require.NoError(t, err)
		require.Equal(t, want, string(got))
	}
}

func TestFilesUnrelatedSchemaChangesDoNotShiftSynthesizedExamples(t *testing.T) {
	before := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	beforeJSON := decodeOpenAPIJSON(t, []byte(before[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	after := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(true, false)))
	afterJSON := decodeOpenAPIJSON(t, []byte(after[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	require.Equal(t, responseExample(t, beforeJSON, "/stable"), responseExample(t, afterJSON, "/stable"))
	require.Equal(t, componentSchema(t, beforeJSON, "StableResponseBody"), componentSchema(t, afterJSON, "StableResponseBody"))
}

func TestFilesSynthesizedExamplesDistinguishSchemaOccurrences(t *testing.T) {
	artifacts := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, false)))
	spec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	example := requireMap(t, responseExample(t, spec, "/stable"), "stable response example")
	require.NotEqual(t, example["message"], example["detail"])
	properties := requireMap(t, componentSchema(t, spec, "StableResponseBody")["properties"], "stable properties")
	message := requireMap(t, properties["message"], "message schema")
	detail := requireMap(t, properties["detail"], "detail schema")
	require.NotEqual(t, message["example"], detail["example"])
}

func TestFilesCanOmitSynthesizedExamplesWithoutRemovingAuthoredExamples(t *testing.T) {
	root := httpgen.RunHTTPDSL(t, synthesizedExampleStabilityDSL(false, true))
	originalRandomizer := root.API.ExampleGenerator.Randomizer
	artifacts := renderOpenAPIExampleArtifacts(t, root)

	jsonSpec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))
	require.NotContains(t, responseMedia(t, jsonSpec, "/stable"), "example")
	require.Equal(t, "authored-example", responseExample(t, jsonSpec, "/authored"))
	stableOperation := requireOperation(t, jsonSpec, "/stable", "get")
	require.Equal(t, "authored-header", parameterByName(t, stableOperation, "X-Authored")["example"])
	require.Equal(t, "authored-cookie", parameterByName(t, stableOperation, "authored_cookie")["example"])
	yamlSpec := decodeOpenAPIExampleYAML(
		t,
		[]byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.yaml")]),
	)
	require.NotContains(t, responseMedia(t, yamlSpec, "/stable"), "example")
	require.Equal(t, "authored-example", responseExample(t, yamlSpec, "/authored"))
	require.True(t, originalRandomizer == root.API.ExampleGenerator.Randomizer)
}

func TestFilesPatternExamplesArePrintable(t *testing.T) {
	artifacts := renderOpenAPIExampleArtifacts(t, httpgen.RunHTTPDSL(t, patternExampleDSL))
	spec := decodeOpenAPIJSON(t, []byte(artifacts[filepath.Join(codegen.Gendir, "http", "openapi.json")]))

	example := requireMap(t, responseExample(t, spec, "/pattern"), "pattern response example")
	value := requireString(t, example["value"], "pattern value example")
	require.Regexp(t, regexp.MustCompile(`^\S+$`), value)
	for _, candidate := range value {
		require.True(t, unicode.IsPrint(candidate), "example contains non-printable rune %U", candidate)
		require.False(t, unicode.IsControl(candidate), "example contains control rune %U", candidate)
	}
}

func renderOpenAPIExampleArtifacts(t *testing.T, root *expr.RootExpr) map[string]string {
	t.Helper()
	files, err := openapiv3.Files(root)
	require.NoError(t, err)
	artifacts := make(map[string]string, len(files))
	for _, file := range files {
		var output bytes.Buffer
		for _, section := range file.AllSections() {
			require.NoError(t, section.Write(&output))
		}
		artifacts[file.Path] = output.String()
	}
	return artifacts
}

func responseExample(t *testing.T, spec map[string]any, path string) any {
	t.Helper()
	return responseMedia(t, spec, path)["example"]
}

func responseMedia(t *testing.T, spec map[string]any, path string) map[string]any {
	t.Helper()
	operation := requireOperation(t, spec, path, "get")
	responses := requireMap(t, operation["responses"], path+" responses")
	response := requireMap(t, responses["200"], path+" response")
	content := requireMap(t, response["content"], path+" content")
	return requireMap(t, content["application/json"], path+" JSON media")
}

func componentSchema(t *testing.T, spec map[string]any, name string) map[string]any {
	t.Helper()
	components := requireMap(t, spec["components"], "components")
	schemas := requireMap(t, components["schemas"], "component schemas")
	return requireMap(t, schemas[name], name+" component schema")
}

func parameterByName(t *testing.T, operation map[string]any, name string) map[string]any {
	t.Helper()
	for _, raw := range requireSlice(t, operation["parameters"], "operation parameters") {
		parameter := requireMap(t, raw, "parameter")
		if parameter["name"] == name {
			return parameter
		}
	}
	t.Errorf("parameter %q not found", name)
	return nil
}

func decodeOpenAPIExampleYAML(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &spec))
	return spec
}

func synthesizedExampleStabilityDSL(addMutableField, omitSynthesized bool) func() {
	return func() {
		API("Example Stability", func() {
			if omitSynthesized {
				Meta("openapi:example", "false")
			}
		})
		Service("Examples", func() {
			Method("Mutable", func() {
				NoSecurity()
				Result(func() {
					Attribute("value", String)
					if addMutableField {
						Attribute("count", Int)
					}
				})
				HTTP(func() {
					GET("/mutable")
					Response(StatusOK)
				})
			})
			Method("Stable", func() {
				NoSecurity()
				Payload(func() {
					Attribute("authored_header", String, func() {
						Example("authored-header")
					})
					Attribute("authored_cookie", String, func() {
						Example("authored-cookie")
					})
				})
				Result(func() {
					Attribute("message", String)
					Attribute("detail", String)
					Attribute("response_cookie", String, func() {
						MinLength(32)
						MaxLength(32)
					})
					Required("response_cookie")
				})
				HTTP(func() {
					Header("authored_header:X-Authored")
					Cookie("authored_cookie:authored_cookie")
					GET("/stable")
					Response(StatusOK, func() {
						Cookie("response_cookie:csrftoken")
					})
				})
			})
			Method("Authored", func() {
				NoSecurity()
				Result(String, func() {
					Example("authored-example")
				})
				HTTP(func() {
					GET("/authored")
					Response(StatusOK)
				})
			})
		})
	}
}

func patternExampleDSL() {
	Service("Pattern", func() {
		Method("Show", func() {
			Result(func() {
				Attribute("value", String, func() {
					Pattern(`^\S+$`)
				})
			})
			HTTP(func() {
				GET("/pattern")
				Response(StatusOK)
			})
		})
	})
}
