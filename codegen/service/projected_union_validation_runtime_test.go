package service

import "fmt"

func projectedUnionValidationRuntime(shape string, required, viewed bool) string {
	path, name := "gen/validation", "Choice"
	if viewed {
		path, name = path+"/views", "ChoiceView"
	}
	kind, field := "Branch", "Value"
	cases := `
		{"zero", ` + "`{\"value\":0}`" + `, false, nil},
		{"above maximum", ` + "`{\"value\":11}`" + `, false, nil},
		{"minimum", ` + "`{\"value\":1}`" + `, true, 1},
		{"maximum", ` + "`{\"value\":10}`" + `, true, 10},`
	switch shape {
	case "selected":
		kind = "UnionBranch"
		cases = `
		{"missing hidden", "{\"value\":1}", false, nil},
		{"invalid hidden", "{\"value\":1,\"hidden\":\"BAD\"}", false, nil},
		{"valid", "{\"value\":1,\"hidden\":\"good\"}", true, 1},`
	case "strings":
		cases = `
		{"pattern", ` + "`{\"value\":\"ONE\"}`" + `, false, nil},
		{"length", ` + "`{\"value\":\"o\"}`" + `, false, nil},
		{"enum", ` + "`{\"value\":\"three\"}`" + `, false, nil},
		{"valid", ` + "`{\"value\":\"one\"}`" + `, true, "one"},`
	case "nested":
		field = "Child.Value"
		cases = `
		{"missing nested", ` + "`{\"child\":{}}`" + `, false, nil},
		{"invalid nested", ` + "`{\"child\":{\"value\":0}}`" + `, false, nil},
		{"valid nested", ` + "`{\"child\":{\"value\":1}}`" + `, true, 1},`
	case "collections":
		field = "Items.0.Value"
		cases = `
		{"missing element", ` + "`{\"items\":[{}]}`" + `, false, nil},
		{"invalid element", ` + "`{\"items\":[{\"value\":0}]}`" + `, false, nil},
		{"valid element", ` + "`{\"items\":[{\"value\":1}]}`" + `, true, 1},`
	case "array":
		kind, field = "BranchList", "0.Value"
		cases = `
		{"missing element", ` + "`[{}]`" + `, false, nil},
		{"invalid element", ` + "`[{\"value\":0}]`" + `, false, nil},
		{"valid element", ` + "`[{\"value\":1}]`" + `, true, 1},`
	}
	missing := !required && (shape == "number" || shape == "default")
	source := fmt.Sprintf(`package regression

import (
	json "encoding/json/v2"
	"reflect"
	"strings"
	"testing"

	generated "example.com/unionvalidation/%s"
	loom "github.com/CaliLuke/loom/pkg"
)

func TestDecode(t *testing.T) {
	for _, test := range []struct {
		name, input string
		valid bool
		want any
	}{
		{"missing", "{}", %t, nil},
		{"null", "null", false, nil},
		{"null field", `+"`{\"value\":null}`"+`, false, nil},
		{"malformed", "{", false, nil},
%s
	} {
		t.Run(test.name, func(t *testing.T) {
			var choice generated.%s
			err := json.Unmarshal([]byte(test.input), &choice, loom.JSONOptions())
			if (err == nil) != test.valid {
				t.Fatalf("valid=%%v, error=%%v", test.valid, err)
			}
			if !test.valid {
				return
			}
			if string(choice.Kind()) != %q {
				t.Fatalf("wrong branch: %%s", choice.Kind())
			}
			value := reflect.ValueOf(choice).FieldByName(%q)
			for _, part := range strings.Split(%q, ".") {
				for value.Kind() == reflect.Pointer {
					value = value.Elem()
				}
				if part == "0" {
					value = value.Index(0)
				} else {
					value = value.FieldByName(part)
				}
			}
			for value.IsValid() && value.Kind() == reflect.Pointer {
				value = value.Elem()
			}
			var got any
			if value.IsValid() {
				got = value.Interface()
			}
			if !reflect.DeepEqual(test.want, got) {
				t.Errorf("decoded field=%%v, want %%v", got, test.want)
			}
		})
	}
}
`, path, missing, cases, name, kind, kind, field)
	if shape == "selected" {
		source += `
func TestSelectedViewValidation(t *testing.T) {
	value := 1
	branch := &generated.SelectedBranchView{Value: &value}
	if err := generated.ValidateSelectedBranchView(branch); err != nil {
		t.Fatalf("default view excludes hidden: %v", err)
	}
	if err := generated.ValidateSelectedBranchViewFull(branch); err == nil {
		t.Fatal("full view must require hidden")
	}
	invalid := "BAD"
	branch.Hidden = &invalid
	if err := generated.ValidateSelectedBranchView(branch); err != nil {
		t.Fatalf("default view does not validate hidden: %v", err)
	}
	if err := generated.ValidateSelectedBranchViewFull(branch); err == nil {
		t.Fatal("full view must validate hidden")
	}
}
`
	}
	return source
}
