package generator

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	d "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
)

func TestGenerateDefaultMaterializationDiagnostic(t *testing.T) {
	for _, required := range []bool{false, true} {
		name := "optional materialized"
		if required {
			name = "required unused"
		}
		t.Run(name, func(t *testing.T) {
			t.Cleanup(func() {
				generatorLoader = generators
			})
			generatorLoader = func(string) ([]genfunc, error) {
				return []genfunc{Service, Transport}, nil
			}
			codegen.RunDSL(t, func() {
				d.Service("defaults", func() {
					d.Method("read", func() {
						d.Payload(func() {
							d.Attribute("values", d.MapOf(d.String, d.ArrayOf(d.String)), func() {
								d.Meta("struct:field:type", "url.Values", "net/url")
								d.Default(map[string][]string{"q": {"one"}})
							})
							if required {
								d.Required("values")
							}
						})
						d.HTTP(func() {
							d.POST("/")
						})
					})
				})
			})
			outputs, err := Generate(t.TempDir(), "gen", "github.com/CaliLuke/loom/codegen/generator", false)
			if required {
				require.NoError(t, err)
				return
			}
			require.Nil(t, outputs)
			require.ErrorContains(t, err, "unsupported opaque default materialization")
			require.ErrorContains(t, err, "default_diagnostics_test.go:")
			require.ErrorContains(t, err, "service defaults, method read")
			var materialization *codegen.DefaultValueError
			require.ErrorAs(t, err, &materialization)
			require.NotContains(t, err.Error(), "panic:")
		})
	}
}

func TestGeneratePreservesProgrammerPanics(t *testing.T) {
	for _, failure := range []any{"programmer bug", codegen.NewError(nil, nil, errors.New("attributed programmer bug"))} {
		t.Run("panic", func(t *testing.T) {
			t.Cleanup(func() {
				generatorLoader = generators
			})
			codegen.RunDSL(t, func() {
				d.API("defaults", func() {
				})
			})
			generatorLoader = func(string) ([]genfunc, error) {
				return []genfunc{func(string, []eval.Root) ([]*codegen.File, error) {
					panic(failure)
				}}, nil
			}
			require.PanicsWithValue(t, failure, func() {
				_, err := Generate(t.TempDir(), "gen", "github.com/CaliLuke/loom/codegen/generator", false)
				require.NoError(t, err)
			})
		})
	}
}
