package uniongen

import (
	"bytes"
	"fmt"
	"testing"

	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/internal/testingx"
)

func TestUnionFormDecodersGenerated(t *testing.T) {
	batch := testingx.NewModuleBatch(t, testingx.RepoRoot(), "")
	for _, test := range []struct {
		name           string
		textSupported  bool
		otherSupported bool
	}{
		{"rejected", false, false},
		{"mixed", true, false},
		{"supported", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fields := []*Field{
				{Name: "text", KindConst: "ChoiceText", FieldName: "Text", MethodName: "Text", FieldType: "string", TypeTag: "text"},
				{Name: "other", KindConst: "ChoiceOther", FieldName: "Other", MethodName: "Other", FieldType: "string", TypeTag: "other"},
			}
			for index, supported := range []bool{test.textSupported, test.otherSupported} {
				if !supported {
					fields[index].FormError = "unsupported " + fields[index].Name + " form contract"
				}
			}
			data := &Type{
				Name: "Choice", KindName: "ChoiceKind", TypeKey: "type", ValueKey: "value",
				Fields: fields, HasScalarFormBranch: test.textSupported || test.otherSupported,
			}
			var output bytes.Buffer
			section := &codegen.JenniferSection{Build: func(stmt *jen.Statement) {
				Emit(stmt, data)
			}}
			require.NoError(t, section.Write(&output))
			module := batch.Case(t, test.name, "example.com/unionform")
			module.WriteFile(t, "union.go", unionFormPrelude+output.String())
			module.WriteFile(t, "union_test.go", fmt.Sprintf(unionFormHarness, test.textSupported, test.otherSupported))
		})
	}
	batch.Check(t)
}

const unionFormPrelude = `package unionform

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"net/url"

	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
)

`

const unionFormHarness = `package unionform

import (
	"net/url"
	"strings"
	"testing"
)

func TestFormSelection(t *testing.T) {
	for _, test := range []struct {
		kind      string
		supported bool
	}{
		{"text", %t},
		{"other", %t},
	} {
		t.Run(test.kind, func(t *testing.T) {
			before := NewChoiceText("original")
			value := before
			err := value.UnmarshalFormValues(url.Values{"type": {test.kind}, "value": {"decoded"}}, "")
			if !test.supported {
				if err == nil || err.Error() != "unsupported "+test.kind+" form contract" || value != before {
					t.Fatalf("rejected branch changed value or error: %%#v, %%v", value, err)
				}
				return
			}
			if err != nil || string(value.Kind()) != test.kind {
				t.Fatalf("supported branch did not decode: %%#v, %%v", value, err)
			}
			decoded := value.Text
			if test.kind == "other" {
				decoded = value.Other
			}
			if decoded != "decoded" {
				t.Fatalf("lost decoded value: %%q", decoded)
			}
		})
	}
	for _, kind := range []string{"", "unknown"} {
		before := NewChoiceText("original")
		value := before
		err := value.UnmarshalFormValues(url.Values{"type": {kind}}, "")
		if err == nil || !strings.Contains(err.Error(), "type") || value != before {
			t.Fatalf("invalid discriminant changed value or lost error: %%#v, %%v", value, err)
		}
	}
}
`
