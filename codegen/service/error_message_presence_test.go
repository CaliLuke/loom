package service

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/testingx"
)

const errorMessagePresenceHarness = `package errormessages_test

import (
	json "encoding/json/v2"
	"testing"

	svc "example.com/errormessages/gen/faults"
	loom "github.com/CaliLuke/loom/pkg"
)

type loomError interface {
	Error() string
	LoomErrorName() string
}

func TestErrorMessages(t *testing.T) {
	text, empty, reason := "explicit", "", "reason text"
	var nilNullable *svc.NullableRequired
	for _, test := range []struct {
		name, want, errorName string
		err                   loomError
	}{
		{"plain value", "explicit", "plain", &svc.Plain{Message: "explicit"}},
		{"plain empty", "plain fallback", "plain", &svc.Plain{}},
		{"optional value", "explicit", "optional", &svc.Optional{Message: &text}},
		{"optional empty", "optional fallback", "optional", &svc.Optional{Message: &empty}},
		{"optional absent", "optional fallback", "optional", &svc.Optional{}},
		{"nullable value", "explicit", "nullable_required", &svc.NullableRequired{Message: loom.NullableValue("explicit")}},
		{"nullable empty", "nullable required fallback", "nullable_required", &svc.NullableRequired{Message: loom.NullableValue("")}},
		{"nullable null", "nullable required fallback", "nullable_required", &svc.NullableRequired{Message: loom.NullValue[string]()}},
		{"nullable absent", "nullable required fallback", "nullable_required", &svc.NullableRequired{}},
		{"nullable nil receiver", "nullable required fallback", "nullable_required", nilNullable},
		{"optional nullable value", "explicit", "nullable_optional", &svc.NullableOptional{Message: loom.NullableValue("explicit"), Reason: &reason}},
		{"optional nullable null uses next field", "reason text", "nullable_optional", &svc.NullableOptional{Message: loom.NullValue[string](), Reason: &reason}},
		{"optional nullable absent uses next field", "reason text", "nullable_optional", &svc.NullableOptional{Reason: &reason}},
		{"optional nullable null", "nullable optional fallback", "nullable_optional", &svc.NullableOptional{Message: loom.NullValue[string]()}},
		{"optional nullable absent", "nullable optional fallback", "nullable_optional", &svc.NullableOptional{}},
		{"named nullable value", "explicit", "named_nullable", &svc.NamedNullable{Detail: loom.NullableValue(svc.Detail("explicit"))}},
		{"named nullable empty", "named nullable fallback", "named_nullable", &svc.NamedNullable{Detail: loom.NullableValue(svc.Detail(""))}},
		{"named nullable null", "named nullable fallback", "named_nullable", &svc.NamedNullable{Detail: loom.NullValue[svc.Detail]()}},
		{"named nullable absent", "named nullable fallback", "named_nullable", &svc.NamedNullable{}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := test.err.Error(); got != test.want {
				t.Errorf("Error() = %q, want %q", got, test.want)
			}
			if got := test.err.LoomErrorName(); got != test.errorName {
				t.Errorf("LoomErrorName() = %q, want %q", got, test.errorName)
			}
		})
	}
}

func TestNullableMessagePayload(t *testing.T) {
	for _, test := range []struct {
		name, payload, want string
		present, null       bool
	}{
		{"value", ` + "`{\"message\":\"explicit\"}`" + `, "explicit", true, false},
		{"null", ` + "`{\"message\":null}`" + `, "nullable optional fallback", true, true},
		{"absent", "{}", "nullable optional fallback", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var decoded svc.NullableOptional
			if err := json.Unmarshal([]byte(test.payload), &decoded, loom.JSONOptions()); err != nil {
				t.Fatalf("decode: %v", err)
			}
			if decoded.Message.Present() != test.present || decoded.Message.IsNull() != test.null {
				t.Errorf("presence = (%v, %v), want (%v, %v)", decoded.Message.Present(), decoded.Message.IsNull(), test.present, test.null)
			}
			if got := decoded.Error(); got != test.want {
				t.Errorf("Error() = %q, want %q", got, test.want)
			}
			encoded, err := json.Marshal(&decoded, loom.JSONOptions())
			if err != nil {
				t.Fatalf("encode: %v", err)
			}
			if string(encoded) != test.payload {
				t.Errorf("encoded = %s, want %s", encoded, test.payload)
			}
		})
	}
}
`

// errorMessagePresenceDSL declares error types whose conventional message
// fields use native, pointer and Nullable storage.
func errorMessagePresenceDSL() {
	detail := dsl.Type("Detail", dsl.String)
	plain := dsl.Type("Plain", func() {
		dsl.Description("plain fallback")
		dsl.Attribute("message", dsl.String)
		dsl.Required("message")
	})
	optional := dsl.Type("Optional", func() {
		dsl.Description("optional fallback")
		dsl.Attribute("message", dsl.String)
	})
	nullableRequired := dsl.Type("NullableRequired", func() {
		dsl.Description("nullable required fallback")
		dsl.Attribute("message", dsl.String, func() {
			dsl.Nullable()
		})
		dsl.Required("message")
	})
	nullableOptional := dsl.Type("NullableOptional", func() {
		dsl.Description("nullable optional fallback")
		dsl.Attribute("message", dsl.String, func() {
			dsl.Nullable()
		})
		dsl.Attribute("reason", dsl.String)
	})
	namedNullable := dsl.Type("NamedNullable", func() {
		dsl.Description("named nullable fallback")
		dsl.Attribute("detail", detail, func() {
			dsl.Nullable()
		})
		dsl.Required("detail")
	})
	dsl.Service("faults", func() {
		dsl.Method("show", func() {
			dsl.Error("plain", plain)
			dsl.Error("optional", optional)
			dsl.Error("nullable_required", nullableRequired)
			dsl.Error("nullable_optional", nullableOptional)
			dsl.Error("named_nullable", namedNullable)
		})
	})
}

func TestErrorMessagePresenceGeneratedModule(t *testing.T) {
	const module = "example.com/errormessages"
	root := codegen.RunDSL(t, errorMessagePresenceDSL)
	dir := t.TempDir()
	services := NewServicesData(root)
	for _, service := range root.Services {
		for _, file := range Files(module+"/gen", service, services, make(map[string][]string)) {
			_, err := file.Render(dir)
			require.NoError(t, err, file.Path)
		}
	}
	mod := fmt.Sprintf("module %s\n\ngo 1.27.0\nrequire github.com/CaliLuke/loom v1.0.0\nreplace github.com/CaliLuke/loom => %s\n", module, testingx.RepoRoot())
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(mod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "error_test.go"), []byte(errorMessagePresenceHarness), 0o600))
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "./gen/..."}, {"vet", "./..."}, {"test", "-count=1", "./..."}} {
		output, err := testingx.RunCmd(dir, "go", args...)
		require.NoError(t, err, output)
	}
}
