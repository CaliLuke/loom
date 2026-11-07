package service

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/uniongen"
	"github.com/CaliLuke/loom/codegen/unionjson"
	"github.com/CaliLuke/loom/dsl"
)

func TestUnionValidationImportOwnership(t *testing.T) {
	const body = "err = loom.InvalidLengthError(\"name\", utf8.RuneCountInString(v), 2, true)"
	for _, tc := range []struct {
		name                   string
		matcher, method, field string
		want                   map[string]bool
	}{
		{"matcher", body, "", "", map[string]bool{"types/choice.go": true}},
		{"method", "", body, "", map[string]bool{"types/cat.go": true}},
		{"both files", body, body, "", map[string]bool{"types/choice.go": true, "types/cat.go": true}},
		{"no dependency", "", "", "", map[string]bool{}},
		{"unemitted field cache", "", "", body, map[string]bool{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			union := &UnionTypeData{Type: uniongen.Type{
				Name: "Choice", Untagged: true,
				Loc:    &codegen.Location{FilePath: "types/choice.go", RelImportPath: "types"},
				Fields: []*uniongen.Field{{ValidateCode: tc.field}},
				JSON:   &unionjson.Union{Branches: []*unionjson.Branch{{Validate: tc.matcher}}},
			}}
			if tc.method != "" {
				union.validations = []*unionValidationData{{
					loc:  &codegen.Location{FilePath: "types/cat.go", RelImportPath: "types"},
					data: &ValidateData{Name: "ValidateCat", Validate: tc.method},
				}}
			}
			sections := collectServiceTypeSections(&Data{unions: []*UnionTypeData{union}}, "example/service.go")
			require.Equal(t, tc.want, sections.validationUTF8Paths)
		})
	}
}

func TestUnionMatcherValidationImports(t *testing.T) {
	for _, tc := range []struct {
		name, shape    string
		custom, length bool
	}{
		{"custom object length", "object", true, true},
		{"custom primitive length", "primitive", true, true},
		{"custom nested length", "nested", true, true},
		{"ordinary object length", "object", false, true},
		{"custom no length", "object", true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, unionValidationImportsDSL(tc.shape, tc.custom, tc.length))
			services := NewServicesData(root)
			files := Files("example.com/imports/gen", root.Services[0], services, make(map[string][]string))
			wantPath := filepath.Join("gen", "imports", "service.go")
			if tc.custom {
				wantPath = filepath.Join("gen", "types", "choice.go")
			}
			var matched bool
			for _, file := range files {
				if file.Path != wantPath {
					continue
				}
				matched = true
				var output bytes.Buffer
				for _, section := range file.AllSections() {
					require.NoError(t, section.Write(&output))
				}
				require.Contains(t, output.String(), "matchJSON(data []byte)")
				if tc.length && tc.shape != "nested" {
					require.Contains(t, output.String(), "utf8.RuneCountInString")
					require.Contains(t, output.String(), `"unicode/utf8"`)
				} else {
					require.NotContains(t, output.String(), `"unicode/utf8"`)
				}
			}
			require.True(t, matched)
		})
	}
}

func unionValidationImportsDSL(shape string, custom, length bool) func() {
	return func() {
		dsl.API("imports", func() {})
		textRules := func() {
			if length {
				dsl.MinLength(2)
				dsl.MaxLength(8)
			}
		}
		var branch any = dsl.String
		if shape != "primitive" {
			branch = dsl.Type("Cat", func() {
				if custom {
					dsl.Meta("struct:pkg:path", "types")
				}
				dsl.Attribute("name", dsl.String, textRules)
				dsl.Required("name")
			})
			if shape == "nested" {
				child := branch
				branch = dsl.Type("Nested", func() {
					if custom {
						dsl.Meta("struct:pkg:path", "types")
					}
					dsl.Attribute("cat", child)
					dsl.Required("cat")
				})
			}
		}
		choice := dsl.Type("Choice", dsl.OneOf(func() {
			dsl.Untagged()
			if shape == "primitive" {
				dsl.Attribute("cat", branch, textRules)
			} else {
				dsl.Attribute("cat", branch)
			}
			dsl.Attribute("dog", dsl.Int)
		}), func() {
			if custom {
				dsl.Meta("struct:pkg:path", "types")
			}
		})
		dsl.Service("imports", func() {
			dsl.Method("write", func() {
				dsl.Payload(choice)
			})
		})
	}
}
