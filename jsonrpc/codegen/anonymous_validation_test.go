package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestAnonymousValidationGenerated(t *testing.T) {
	for _, nullable := range []bool{false, true} {
		for _, enum := range []bool{false, true} {
			t.Run(fmt.Sprintf("nullable=%t/enum=%t", nullable, enum), func(t *testing.T) {
				root := RunJSONRPCDSL(t, testdata.AnonymousValidationDSL("jsonrpc", nullable, enum))
				dir := t.TempDir()
				if output := os.Getenv("LOOM_VALIDATION_CARRIER_OUTPUT"); output != "" {
					dir = filepath.Join(output, "jsonrpc", fmt.Sprintf("nullable-%t-enum-%t", nullable, enum))
					require.NoError(t, os.MkdirAll(dir, 0o700))
				}
				renderJSONRPCModule(t, dir, "example.com/anonymousvalidation", root)
				if os.Getenv("LOOM_VALIDATION_CARRIER_GENERATE_ONLY") == "1" {
					return
				}
				runGoJSONRPCTestCommand(t, dir, "mod", "edit", "-require=github.com/CaliLuke/loom@v1.10.0-alpha.9.0.20261012214204-66a9dc9090a0")
				harness := strings.ReplaceAll(anonymousValidationHarness, "NULLABLE", fmt.Sprint(nullable))
				harness = strings.ReplaceAll(harness, "ENUM", fmt.Sprint(enum))
				require.NoError(t, os.WriteFile(filepath.Join(dir, "validation_test.go"), []byte(harness), 0o600))
				for _, args := range [][]string{{"mod", "tidy"}, {"build", "./..."}, {"vet", "./..."}, {"test", "-race", "-count=1", "./..."}} {
					runGoJSONRPCTestCommand(t, dir, args...)
				}
			})
		}
	}
}

const anonymousValidationHarness = `package validation_test

import (
	"context"
	"encoding/json/v2"
	"net/http/httptest"
	"strings"
	"testing"

	server "example.com/anonymousvalidation/gen/jsonrpc/objects/server"
	svc "example.com/anonymousvalidation/gen/objects"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestAnonymousObject(t *testing.T) {
	cases := []struct{ name, body, code, detail string }{
		{"absent", "{}", "", ""},
		{"valid", ` + "`" + `{"value":{"child":{"text":"good"}}}` + "`" + `, "", ""},
		{"null", ` + "`" + `{"value":null}` + "`" + `, "", ""},
	}
	if !NULLABLE {
		cases[2].code = "decode_payload"
		cases[2].detail = "invalid request body"
	}
	if !ENUM {
		cases = append(cases, struct{ name, body, code, detail string }{"invalid", ` + "`" + `{"value":{"child":{"text":"BAD"}}}` + "`" + `, "invalid_pattern", "validation error"},
			struct{ name, body, code, detail string }{"nested", ` + "`" + `{"value":{"inline":{"text":"BAD"}}}` + "`" + `, "invalid_pattern", "validation error"},
			struct{ name, body, code, detail string }{"array", ` + "`" + `{"value":{"children":[{"text":"BAD"}]}}` + "`" + `, "invalid_pattern", "validation error"},
			struct{ name, body, code, detail string }{"map", ` + "`" + `{"value":{"entries":{"key":{"text":"BAD"}}}}` + "`" + `, "invalid_pattern", "validation error"},
			struct{ name, body, code, detail string }{"missing", ` + "`" + `{"value":{"child":{}}}` + "`" + `, "missing_field", "Missing required field: text"},
			struct{ name, body, code, detail string }{"empty", ` + "`" + `{"value":{}}` + "`" + `, "", ""})
	} else {
		cases = append(cases, struct{ name, body, code, detail string }{"enum", ` + "`" + `{"value":{"child":{"text":"other"}}}` + "`" + `, "invalid_enum_value", ""})
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			calls := 0
			endpoint := func(_ context.Context, payload any) (any, error) {
				calls++
				require.NotNil(t, payload)
				if tt.name == "valid" {
					data, err := json.Marshal(payload)
					require.NoError(t, err)
					require.Contains(t, string(data), "good")
				}
				return nil, nil
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(&svc.Endpoints{Send: endpoint}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
			body := tt.body
			path := "/send"
			if "jsonrpc" == "jsonrpc" {
				path = "/rpc"
				body = ` + "`" + `{"jsonrpc":"2.0","id":1,"method":"send","params":` + "`" + ` + body + ` + "`" + `}` + "`" + `
			}
			req := httptest.NewRequest("POST", path, strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, req)
			if "jsonrpc" == "jsonrpc" {
				require.Equal(t, 200, response.Code, response.Body.String())
				var result map[string]any
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &result))
				if tt.code != "" {
					require.NotNil(t, result["error"], response.Body.String())
					failure := result["error"].(map[string]any)
					require.Equal(t, float64(-32602), failure["code"])
					if tt.detail != "" && tt.code != "decode_payload" {
						require.Equal(t, tt.detail, failure["message"])
					}
				} else {
					require.Nil(t, result["error"], response.Body.String())
				}
			} else if tt.code != "" {
				require.Equal(t, 400, response.Code, response.Body.String())
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, tt.code, problem.Code)
				if tt.detail != "" {
					require.Equal(t, tt.detail, problem.Detail)
				}
			} else {
				require.Equal(t, 204, response.Code, response.Body.String())
			}
			expected := 1
			if tt.code != "" {
				expected = 0
			}
			require.Equal(t, expected, calls)
		})
	}
}
`
