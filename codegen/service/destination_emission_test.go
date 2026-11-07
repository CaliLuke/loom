package service

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestDestinationRawUnionEmission(t *testing.T) {
	for _, local := range []bool{false, true} {
		for _, reverse := range []bool{false, true} {
			t.Run(fmt.Sprintf("local=%t/reverse=%t", local, reverse), func(t *testing.T) {
				root := codegen.RunDSL(t, destinationRawUnionDSL(local))
				services := NewServicesData(root)
				order := slices.Clone(root.Services)
				if reverse {
					slices.Reverse(order)
				}
				var previous map[string]string
				for range 2 {
					files := renderedDestinationFiles(t, order, services)
					var count int
					for _, source := range files {
						count += strings.Count(source, "type CountOrtext struct {")
					}
					require.Equal(t, 1, count, "one physical union is emitted once across all file kinds")
					path := "gen/common/left.go"
					if reverse {
						path = "gen/common/right.go"
					}
					if local {
						path = "gen/alpha/service.go"
						if reverse {
							path = "gen/alpha/right.go"
						}
					}
					require.Contains(t, files[path], "type CountOrtext struct {", "the first existing file candidate owns emission")
					if previous != nil {
						require.Equal(t, previous, files, "a fresh generation ledger resets emission without resetting semantic data")
					}
					previous = files
				}
			})
		}
	}
}

func TestDestinationNominalEmission(t *testing.T) {
	root := codegen.RunDSL(t, destinationNominalDSL)
	files := renderedDestinationFiles(t, root.Services, NewServicesData(root))
	require.Contains(t, files["gen/left/foo_bar.go"], "type FooBar struct {")
	require.Contains(t, files["gen/right/foo_bar.go"], "type FooBar struct {")
}

func TestDestinationAuthoredGenPath(t *testing.T) {
	root := codegen.RunDSL(t, destinationGenPathDSL)
	files := renderedDestinationFiles(t, root.Services, NewServicesData(root))
	require.Contains(t, files, "gen/foo/service.go")
	require.Contains(t, files, "gen/gen/foo/service.go", "an authored gen component is part of the physical package")
	require.Contains(t, files["gen/gen/foo/service.go"], "type Service struct {")
}

func renderedDestinationFiles(t *testing.T, order []*expr.ServiceExpr, services *ServicesData) map[string]string {
	t.Helper()
	result := make(map[string]string)
	packages := make(map[string][]string)
	for _, service := range order {
		require.NoError(t, SetUserTypeImports("example.com/destination/gen", services.Get(service.Name)))
		for _, file := range Files("example.com/destination/gen", service, services, packages) {
			var source strings.Builder
			for _, section := range file.AllSections() {
				require.NoError(t, section.Write(&source))
			}
			result[file.Path] += source.String()
		}
	}
	return result
}

func destinationRawUnionDSL(local bool) func() {
	return func() {
		choice := dsl.OneOf(func() {
			dsl.Attribute("text", dsl.String, func() {
				dsl.Meta("struct:field:type", "string")
			})
			dsl.Attribute("count", dsl.Int, func() {
				dsl.Meta("struct:field:type", "int")
			})
		})
		var left expr.DataType
		if !local {
			left = dsl.Type("Left", func() {
				dsl.Meta("struct:pkg:path", "common")
				dsl.Attribute("pick", choice)
			})
		}
		right := dsl.Type("Right", func() {
			path := "common"
			if local {
				path = "alpha"
			}
			dsl.Meta("struct:pkg:path", path)
			dsl.Attribute("pick", choice)
		})
		alpha := func() {
			dsl.Service("alpha", func() {
				dsl.Method("use", func() {
					if local {
						dsl.Payload(func() {
							dsl.Attribute("pick", choice)
						})
					} else {
						dsl.Payload(left)
					}
				})
			})
		}
		beta := func() {
			dsl.Service("beta", func() {
				dsl.Method("use", func() {
					dsl.Payload(right)
				})
			})
		}
		alpha()
		beta()
	}
}

func destinationNominalDSL() {
	left := dsl.Type("foo-bar", func() {
		dsl.Meta("struct:pkg:path", "left")
		dsl.Attribute("text", dsl.String)
	})
	right := dsl.Type("foo_bar", func() {
		dsl.Meta("struct:pkg:path", "right")
		dsl.Attribute("count", dsl.Int)
	})
	dsl.Service("svc", func() {
		dsl.Method("use", func() {
			dsl.Payload(func() {
				dsl.Attribute("left", left)
				dsl.Attribute("right", right)
			})
		})
	})
}

func destinationGenPathDSL() {
	child := dsl.Type("Service", func() {
		dsl.Meta("struct:pkg:path", "gen/foo")
		dsl.Attribute("text", dsl.String)
	})
	dsl.Service("foo", func() {
		dsl.Method("use", func() {
			dsl.Payload(child)
		})
	})
}
