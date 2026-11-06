package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

// TestGeneratedAdditionalAttributesRuntime is the decisive generated
// regression for objects with typed additional members: it decodes, validates,
// converts and encodes through the generated HTTP code. A required member that
// only the typed policy describes must be present, admits zero, and rejects
// null, directly and through Extend. Undeclared members keep their values,
// respect the policy, keep exact names under case-insensitive matching, and
// cannot collide with declared names on output. User types and validations
// that only additional members reach are generated and enforced, recursively
// and on both sides. A member bound to a path parameter or header is neither
// required in the body nor admitted as an undeclared body member: the server
// answers 400 and the client rejects the response.
func TestGeneratedAdditionalAttributesRuntime(t *testing.T) {
	const modulePath = "example.com/additionalit"

	root := RunHTTPDSL(t, testdata.AdditionalAttributesTypedDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, modulePath, root)

	serviceCode := readGeneratedGo(t, filepath.Join(dir, "gen", "records"))
	require.Contains(t, serviceCode, "AdditionalAttributes map[string]int `json:\",embed\"`")
	require.NotContains(t, serviceCode, "`json:\"count,case:strict\"`")
	require.Contains(t, serviceCode, "type Item struct", "a type reachable only through AdditionalAttributes is generated")

	serverCode := readGeneratedGo(t, filepath.Join(dir, "gen", "http", "records", "server"))
	for _, want := range []string{
		"type ItemRequestBody struct",
		"func ValidateItemRequestBody(",
		"type ItemResponseBody struct",
		"ValidateListsRequestBody(&actual, validationPath+\".lists\")",
		"ValidateNodeRequestBody(actual, validationPath+\"[key]\")",
		"loom.InvalidAdditionalMemberError(validationPath+\".id\")",
		"loom.CheckAdditionalMembers(res.AdditionalAttributes, \"PutResponseBody\", \"id\")",
	} {
		require.Contains(t, serverCode, want)
	}
	require.NotContains(t, serverCode, `CheckAdditionalMembers(res.AdditionalAttributes, "res"`, "collision paths name the object, not a Go variable")
	require.NotRegexp(t, `type BoundRequestBody struct \{[^}]*\bID\b`, serverCode, "a path-bound required member is not a body field")
	clientCode := readGeneratedGo(t, filepath.Join(dir, "gen", "http", "records", "client"))
	require.Contains(t, clientCode, "loom.CheckAdditionalMembers(p.AdditionalAttributes, \"OptRequestBody\", \"name\", \"id\")")
	require.Contains(t, clientCode, "loom.InvalidAdditionalMemberError(validationPath+\".id\")")

	require.NoError(t, os.WriteFile(filepath.Join(dir, "additional_attributes_test.go"), []byte(additionalAttributesHarness), 0o600))
	harness, err := os.ReadFile(filepath.Join("testdata", "additional_attributes_http_harness.go.txt"))
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "additional_attributes_http_test.go"), harness, 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "./...")
}

const additionalAttributesHarness = `package additionalit_test

import (
	"encoding/json/v2"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"

	records "example.com/additionalit/gen/records"
	server "example.com/additionalit/gen/http/records/server"
)

func decode(t *testing.T, raw string, options ...json.Options) (*server.PutRequestBody, error) {
	t.Helper()
	var body server.PutRequestBody
	if err := json.Unmarshal([]byte(raw), &body, append([]json.Options{loom.JSONOptions()}, options...)...); err != nil {
		return nil, err
	}
	if err := server.ValidatePutRequestBody(&body); err != nil {
		return nil, err
	}
	return &body, nil
}

func TestGeneratedAdditionalAttributes(t *testing.T) {
	for _, tc := range []struct {
		name, raw string
		ok        bool
	}{
		{"missing required extra", ` + "`" + `{"labels":{"id":"a"}}` + "`" + `, false},
		{"zero required extra", ` + "`" + `{"labels":{"id":"a","count":0}}` + "`" + `, true},
		{"null required extra", ` + "`" + `{"labels":{"id":"a","count":null}}` + "`" + `, false},
		{"extra member", ` + "`" + `{"labels":{"id":"a","count":1,"other":2}}` + "`" + `, true},
		{"extra wrong type", ` + "`" + `{"labels":{"id":"a","count":1,"other":"x"}}` + "`" + `, false},
		{"extra null", ` + "`" + `{"labels":{"id":"a","count":1,"other":null}}` + "`" + `, false},
		{"extra violates minimum", ` + "`" + `{"labels":{"id":"a","count":1,"other":-1}}` + "`" + `, false},
		{"inherited missing extra", ` + "`" + `{"labels":{"id":"a","count":1},"tagged":{"id":"b"}}` + "`" + `, false},
		{"inherited zero extra", ` + "`" + `{"labels":{"id":"a","count":1},"tagged":{"id":"b","count":0,"total":3,"x":4}}` + "`" + `, true},
		{"empty extras", ` + "`" + `{"labels":{"id":"a","count":1}}` + "`" + `, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			body, err := decode(t, tc.raw)
			if tc.ok != (err == nil) {
				t.Fatalf("ok=%v err=%v", tc.ok, err)
			}
			if err != nil {
				return
			}
			payload, err := server.NewPutPayload(body)
			if err != nil {
				t.Fatal(err)
			}
			out, _ := json.Marshal(payload, json.Deterministic(true))
			t.Logf("payload=%s labels.count=%d extras=%v", out, payload.Labels.AdditionalAttributes["count"], payload.Labels.AdditionalAttributes)
		})
	}
	t.Run("case-insensitive input stays exact", func(t *testing.T) {
		body, err := decode(t, ` + "`" + `{"labels":{"id":"a","count":1,"ID":5}}` + "`" + `, json.MatchCaseInsensitiveNames(true))
		if err != nil {
			t.Fatal(err)
		}
		if *body.Labels.ID != "a" {
			t.Fatalf("id=%v", *body.Labels.ID)
		}
		if v, ok := body.Labels.AdditionalAttributes["ID"].Value(); !ok || v != 5 {
			t.Fatalf("extras=%v", body.Labels.AdditionalAttributes)
		}
	})
	t.Run("outbound collision", func(t *testing.T) {
		_, err := server.NewPutResponseBody(&records.Labels{ID: "a", AdditionalAttributes: map[string]int{"count": 1, "id": 9}})
		if err == nil {
			t.Fatal("expected collision error")
		}
		t.Log(err)
		res, err := server.NewPutResponseBody(&records.Labels{ID: "a", AdditionalAttributes: map[string]int{"count": 0, "z": 9}})
		if err != nil {
			t.Fatal(err)
		}
		out, _ := json.Marshal(res, json.Deterministic(true))
		if string(out) != ` + "`" + `{"id":"a","count":0,"z":9}` + "`" + ` {
			t.Fatalf("encoded %s", out)
		}
	})
}
`
