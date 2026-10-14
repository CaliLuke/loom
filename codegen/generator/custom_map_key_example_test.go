package generator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	dsl "github.com/CaliLuke/loom/dsl"
)

func TestCustomMapKeyExamplesCompile(t *testing.T) {
	dir := generateExampleModule(t, "example.com/custommapkeys", customMapKeyExamplesDSL)
	service, err := os.ReadFile(filepath.Join(dir, "gen", "mapping", "service.go"))
	require.NoError(t, err)
	require.Contains(t, string(service), "map[[16]byte]string")
}

func customMapKeyExamplesDSL() {
	dsl.API("custommapkeys", func() {
	})
	dsl.Service("mapping", func() {
		for _, shape := range []string{"automatic", "empty", "bounded", "nested", "authored"} {
			dsl.Method(shape, func() {
				dsl.Payload(func() {
					mapping := dsl.MapOf(dsl.Bytes, dsl.String, func() {
						dsl.Key(func() {
							dsl.Meta("struct:field:type", "[16]byte")
						})
					})
					if shape == "nested" {
						dsl.Attribute("values", dsl.ArrayOf(mapping))
						return
					}
					dsl.Attribute("values", mapping, func() {
						switch shape {
						case "empty":
							dsl.MaxLength(0)
						case "bounded":
							dsl.MinLength(2)
							dsl.MaxLength(2)
						case "authored":
							dsl.Example(map[[16]byte]string{})
						}
					})
				})
			})
		}
	})
}
