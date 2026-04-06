package codegen

import (
	"reflect"
	"sort"
	"testing"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestJoinImportPath(t *testing.T) {
	cases := []struct {
		name   string
		genpkg string
		rel    string
		want   string
	}{
		{name: "empty rel", genpkg: "example.com/myapp/gen", rel: "", want: ""},
		{name: "gen suffix", genpkg: "example.com/myapp/gen", rel: "types", want: "example.com/myapp/gen/types"},
		{name: "without gen suffix", genpkg: "example.com/myapp", rel: "types", want: "example.com/myapp/gen/types"},
		{name: "repeated gen suffix", genpkg: "example.com/myapp/gen/gen", rel: "nested/types", want: "example.com/myapp/gen/nested/types"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := JoinImportPath(tc.genpkg, tc.rel)
			if got != tc.want {
				t.Errorf("got import path %q, expected %q", got, tc.want)
			}
		})
	}
}

func TestGetMetaTypeImports(t *testing.T) {
	testdata := []struct {
		name string
		dsl  func()
		want []string
	}{
		{
			name: "payload-primitive",
			dsl: func() {
				dsl.Method("m", func() {
					dsl.Payload(func() {
						dsl.Attribute("a", dsl.String, func() {
							dsl.Meta("struct:field:type", "CustomTypeString", "package/string")
						})
						dsl.Attribute("b", dsl.Int, func() {
							dsl.Meta("struct:field:type", "CustomTypeInt", "package/int")
						})
					})
				})
			},
			want: []string{
				"package/string",
				"package/int",
			},
		},
		{
			name: "payload-map",
			dsl: func() {
				dsl.Method("m", func() {
					dsl.Payload(func() {
						dsl.Attribute("a", dsl.MapOf(dsl.String, dsl.String, func() {
							dsl.Key(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapKey", "package/map-key")
							})
							dsl.Elem(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapElem", "package/map-elem")
							})
						}))
					})
				})
			},
			want: []string{
				"package/map-elem",
				"package/map-key",
			},
		},
		{
			name: "payload-map-map",
			dsl: func() {
				dsl.Method("m", func() {
					dsl.Payload(func() {
						dsl.Attribute("a", dsl.MapOf(dsl.String, dsl.MapOf(dsl.String, dsl.String, func() {
							dsl.Key(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapKey", "package/map-map-key")
							})
							dsl.Elem(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapElem", "package/map-map-elem")
							})
						}), func() {
							dsl.Key(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapKey", "package/map-key")
							})
							dsl.Elem(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapElem", "package/map-elem")
							})
						}))
					})
				})
			},
			want: []string{
				"package/map-key",
				"package/map-map-elem",
				"package/map-map-key",
				"package/map-elem",
			},
		},
		{
			name: "payload-array",
			dsl: func() {
				dsl.Method("m", func() {
					dsl.Payload(func() {
						dsl.Attribute("a", dsl.ArrayOf(dsl.String, func() {
							dsl.Meta("struct:field:type", "SomeCustomTypeArrayElem", "package/array-elem")
						}), func() {
							dsl.Meta("struct:field:type", "SomeCustomTypeArray", "package/array")
						})
					})
				})
			},
			want: []string{
				"package/array-elem",
				"package/array",
			},
		},
		{
			name: "result",
			dsl: func() {
				dsl.Method("m", func() {
					dsl.Result(func() {
						dsl.Attribute("a", dsl.String, func() {
							dsl.Meta("struct:field:type", "CustomTypeString", "package/result-string")
						})
						dsl.Attribute("b", dsl.ArrayOf(dsl.String, func() {
							dsl.Meta("struct:field:type", "SomeCustomTypeArrayElem", "package/result-array-elem")
						}), func() {
							dsl.Meta("struct:field:type", "SomeCustomTypeArray", "package/result-array")
						})
						dsl.Attribute("a", dsl.MapOf(dsl.String, dsl.String, func() {
							dsl.Key(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapKey", "package/result-map-key")
							})
							dsl.Elem(func() {
								dsl.Meta("struct:field:type", "CustomTypeMapElem", "package/result-map-elem")
							})
						}))
					})
				})
			},
			want: []string{
				"package/result-string",
				"package/result-array",
				"package/result-array-elem",
				"package/result-map-key",
				"package/result-map-elem",
			},
		},
	}
	for _, tt := range testdata {
		t.Run(tt.name, func(t *testing.T) {
			eval.SetupTestContext(t)
			serviceExpr := &expr.ServiceExpr{}
			eval.Execute(tt.dsl, serviceExpr)
			if eval.Context.Errors != nil {
				t.Fatalf("%s: Service DSL failed unexpectedly with %s", tt.name, eval.Context.Errors)
			}
			for _, methodExpr := range serviceExpr.Methods {
				eval.Execute(methodExpr.DSLFunc, methodExpr)
				if eval.Context.Errors != nil {
					t.Fatalf("%s: Method DSL failed unexpectedly with %s", tt.name, eval.Context.Errors)
				}
			}
			for _, methodExpr := range serviceExpr.Methods {
				payloadImports := GetMetaTypeImports(methodExpr.Payload)
				resultImports := GetMetaTypeImports(methodExpr.Result)
				got := make([]string, 0, len(payloadImports)+len(resultImports))
				for _, v := range payloadImports {
					got = append(got, v.Path)
				}
				for _, v := range resultImports {
					got = append(got, v.Path)
				}
				sort.Strings(got)
				sort.Strings(tt.want)
				if !reflect.DeepEqual(tt.want, got) {
					t.Errorf("want %+v, got %+v", tt.want, got)
				}
			}
		})
	}
}

func TestGatherAttributeImports(t *testing.T) {
	attr := &expr.AttributeExpr{
		Type: &expr.Object{
			{
				Name: "external",
				Attribute: &expr.AttributeExpr{
					Type: &expr.UserTypeExpr{
						TypeName: "ExternalType",
						AttributeExpr: &expr.AttributeExpr{
							Type: expr.String,
							Meta: expr.MetaExpr{
								"struct:pkg:path": []string{"types"},
							},
						},
					},
				},
			},
			{
				Name: "meta",
				Attribute: &expr.AttributeExpr{
					Type: expr.String,
					Meta: expr.MetaExpr{
						"struct:field:type": []string{"CustomString", "example.com/custom/string"},
					},
				},
			},
			{
				Name: "nested",
				Attribute: &expr.AttributeExpr{
					Type: &expr.Array{
						ElemType: &expr.AttributeExpr{
							Type: &expr.UserTypeExpr{
								TypeName: "NestedType",
								AttributeExpr: &expr.AttributeExpr{
									Type: expr.Int,
									Meta: expr.MetaExpr{
										"struct:pkg:path": []string{"nested/types"},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	imports := GatherAttributeImports("example.com/app/gen", attr)
	got := make([]string, 0, len(imports))
	for _, im := range imports {
		got = append(got, im.Path)
	}
	want := []string{
		"example.com/app/gen/nested/types",
		"example.com/app/gen/types",
		"example.com/custom/string",
	}
	if !reflect.DeepEqual(want, got) {
		t.Errorf("got imports %v, expected %v", got, want)
	}
}
