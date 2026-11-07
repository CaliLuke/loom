package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	codegentestdata "github.com/CaliLuke/loom/codegen/testdata"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestRelocatedTypeFamiliesOwnedByDestination(t *testing.T) {
	for _, order := range [][]string{{"catalog", "catalogrpc"}, {"catalogrpc", "catalog"}} {
		t.Run(order[0], func(t *testing.T) {
			root := codegen.RunDSL(t, codegentestdata.UserTypePackageTransportsDSL)
			services := NewServicesData(root)
			var previous []string
			for _, name := range order {
				data := services.Get(name)
				declared := make(map[string]string)
				for _, typ := range data.userTypes {
					if typ.Loc != nil && typ.Loc.RelImportPath == "common" {
						declared[typ.VarName] = typ.Name
					}
				}
				var unionNames []string
				for _, union := range data.unions {
					if union.Loc == nil || union.Loc.RelImportPath != "common" {
						continue
					}
					require.NotContains(t, declared, union.KindName, "destination package companion collides with a nominal declaration")
					unionNames = append(unionNames, union.Name)
				}
				if previous == nil {
					previous = unionNames
				}
				require.ElementsMatch(t, previous, unionNames)
				aliased := services.WithPackageNames(map[string]string{"common": "commonTypes"}).Get(name)
				require.Equal(t, data.unions[0].Name, aliased.unions[0].Name)
			}
		})
	}
}

func TestRelocatedInlineUnionFamily(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		outer := dsl.Type("Outer", func() {
			dsl.Meta("struct:pkg:path", "common")
			dsl.OneOf("pick", func() {
				dsl.Attribute("Kind", dsl.String)
				dsl.Attribute("other", dsl.Int)
			})
		})
		dsl.Service("svc", func() {
			dsl.Method("use", func() {
				dsl.Payload(outer)
			})
		})
	})
	data := NewServicesData(root).Get("svc")
	names := make(map[string]bool)
	for _, typ := range data.userTypes {
		if typ.Loc != nil && typ.Loc.RelImportPath == "common" {
			names[typ.VarName] = true
		}
	}
	for _, union := range data.unions {
		if union.Loc != nil && union.Loc.RelImportPath == "common" {
			require.NotContains(t, names, union.KindName)
		}
	}
}

func TestRelocatedForcedOwnServicePackage(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		dsl.Type("Service", func() {
			dsl.Meta("struct:pkg:path", "svc")
			dsl.Meta("type:generate:force", "svc")
			dsl.Attribute("value", dsl.String)
		})
		dsl.Service("svc", func() {
			dsl.Method("use", func() {
			})
		})
	})
	data := NewServicesData(root).Get("svc")
	require.Len(t, data.userTypes, 1)
	require.Equal(t, "Service2", data.userTypes[0].VarName)
	require.Equal(t, "*Service2", data.userTypes[0].Ref)
}

// Synthetic collector seam: derived named wrappers need not carry explicit
// metadata; their enclosing declaration supplies the physical destination.
func TestInheritedDestinationCollectionContext(t *testing.T) {
	location := &codegen.Location{RelImportPath: "common"}
	scope := codegen.NewNameScope()
	scope.SetTypeNames(nil, func(typ expr.DataType, loc *codegen.Location) (string, bool) {
		if loc == nil {
			return "", false
		}
		require.Equal(t, "common", loc.RelImportPath)
		return typ.Name() + "2", true
	})
	nested := &expr.UserTypeExpr{TypeName: "Nested", AttributeExpr: &expr.AttributeExpr{Type: expr.String}}
	types := collectTypes(&expr.AttributeExpr{Type: nested}, scope, make(map[string]struct{}), location)
	require.Len(t, types, 1)
	require.Equal(t, "Nested2", types[0].VarName)
	require.Equal(t, "Nested2", types[0].Ref)
	union := &expr.UserTypeExpr{TypeName: "Choice", AttributeExpr: &expr.AttributeExpr{Type: &expr.Union{TypeName: "RawChoice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}}}
	unions := make(map[destinationTypeIdentity]*UnionTypeData)
	collectUnionTypes(&expr.AttributeExpr{Type: union}, scope, location, "", unions, make(map[destinationTypeIdentity]struct{}))
	require.Equal(t, "Choice2", unions[destinationTypeIdentity{path: "common", hash: union.Hash()}].Name)
	require.Equal(t, "common", unions[destinationTypeIdentity{path: "common", hash: union.Hash()}].Loc.RelImportPath)
}

func TestRelocatedAuthorizationValidation(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		outer := dsl.Type("Outer", func() {
			dsl.Meta("struct:pkg:path", "common")
			dsl.OneOf("pick", func() {
				dsl.Attribute("Kind", dsl.String, func() {
					dsl.MinLength(1)
				})
				dsl.Attribute("other", dsl.Int)
			})
		})
		access := dsl.Authorization("access.outer", outer)
		dsl.Service("svc", func() {
			dsl.Method("use", func() {
				dsl.Payload(func() {
					dsl.Attribute("outer", outer)
					dsl.Required("outer")
				})
				dsl.Authorize(access, func() {
					dsl.Bind("pick", "outer.pick")
				})
			})
		})
	})
	services := NewServicesData(root)
	data := services.Get("svc")
	var output strings.Builder
	for _, file := range Files("example.com/access/gen", root.Services[0], services, make(map[string][]string)) {
		if strings.HasSuffix(file.Path, "/authorization.go") {
			for _, section := range file.AllSections()[1:] {
				require.NoError(t, section.Write(&output))
			}
		}
	}
	rendered := output.String()
	require.Contains(t, rendered, "func validateAccessOuter(value *common.Outer")
	require.Contains(t, rendered, "value.Pick.AsKind()")
	require.Contains(t, rendered, "loom.InvalidLengthError")
	require.NotContains(t, rendered, "common.Pick(")
	var kindName string
	for _, union := range data.unions {
		if union.Name == "Pick2" {
			kindName = union.KindName
		}
	}
	require.Equal(t, "Pick2Kind", kindName, "the real promoted PickKind occupies the natural discriminator name")
}

func TestRelocatedObjectDefaultContext(t *testing.T) {
	root := codegen.RunDSL(t, relocatedObjectDefaultDSL)
	data := NewServicesData(root).Get("svc")
	child := root.Services[0].Methods[0].Payload.Find("child")
	require.NotNil(t, child)
	var unionName string
	for _, union := range data.unions {
		if union.Loc != nil && union.Loc.RelImportPath == "common" {
			unionName = union.Name
		}
	}
	require.Equal(t, "KindOrother2", unionName)
	// Object-field conversion renders a child's default with its parent's context.
	// The public default entry must derive the child's physical owner itself.
	ctx := codegen.NewAttributeContext(false, false, true, data.PkgName, data.Scope)
	value, err := ctx.DefaultValue(child, child.DefaultValue)
	require.NoError(t, err)
	require.Contains(t, value.Expression, "loom.NullValue[common."+unionName+"]()")
	require.Equal(t, data.PkgName, ctx.DefaultPkg)
}

func relocatedObjectDefaultDSL() {
	child := dsl.Type("Child", func() {
		dsl.Meta("struct:pkg:path", "common")
		dsl.Attribute("pick", dsl.OneOf(func() {
			dsl.Attribute("Kind", dsl.String)
			dsl.Attribute("other", dsl.Int)
		}), func() {
			dsl.Nullable()
		})
	})
	kind := dsl.Type("KindOrotherKind", dsl.String, func() {
		dsl.Meta("struct:pkg:path", "common")
	})
	dsl.Service("svc", func() {
		dsl.Method("use", func() {
			dsl.Payload(func() {
				dsl.Attribute("kind", kind)
				dsl.Attribute("child", child, func() {
					dsl.Default(map[string]any{"pick": nil})
				})
			})
		})
	})
}

func TestRelocatedSharedRawUnionDestinations(t *testing.T) {
	for _, reverse := range []bool{false, true} {
		root := codegen.RunDSL(t, func() {
			choice := dsl.OneOf(func() {
				dsl.Attribute("text", dsl.String, func() {
					dsl.Meta("struct:field:type", "string")
				})
				dsl.Attribute("count", dsl.Int, func() {
					dsl.Meta("struct:field:type", "int")
				})
			})
			left := dsl.Type("Left", func() {
				dsl.Meta("struct:pkg:path", "left")
				dsl.Attribute("pick", choice)
			})
			right := dsl.Type("Right", func() {
				dsl.Meta("struct:pkg:path", "right")
				dsl.Attribute("pick", choice)
			})
			dsl.Service("svc", func() {
				dsl.Method("use", func() {
					dsl.Payload(func() {
						if reverse {
							dsl.Attribute("right", right)
							dsl.Attribute("left", left)
						} else {
							dsl.Attribute("left", left)
							dsl.Attribute("right", right)
						}
					})
				})
			})
		})
		services := NewServicesData(root)
		for _, data := range []*Data{services.Get("svc"), services.WithPackageNames(map[string]string{"left": "leftAlias", "right": "rightAlias"}).Get("svc")} {
			var paths []string
			for _, union := range data.unions {
				require.Equal(t, "CountOrtext", union.Name)
				paths = append(paths, union.Loc.RelImportPath)
			}
			require.Equal(t, []string{"left", "right"}, paths, "each physical destination emits the shared raw type, in stable order")
		}
	}
}

func TestUnionDestinationIdentityNormalizesLocalPath(t *testing.T) {
	union := &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{{Name: "text", Attribute: &expr.AttributeExpr{Type: expr.String}}}}
	scope := codegen.NewNameScope()
	declarations := make(map[destinationTypeIdentity]*UnionTypeData)
	seen := make(map[destinationTypeIdentity]struct{})
	collectUnionTypes(&expr.AttributeExpr{Type: union}, scope, nil, "svc", declarations, seen)
	collectUnionTypes(&expr.AttributeExpr{Type: union}, scope, &codegen.Location{RelImportPath: "svc"}, "svc", declarations, seen)
	require.Len(t, declarations, 1, "explicit and implicit locations in one physical package are the same declaration")
}
