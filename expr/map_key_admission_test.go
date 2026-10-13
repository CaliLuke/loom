package expr_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/expr"
)

func TestMapKeyAdmissionRejectsNativeSlices(t *testing.T) {
	for _, shape := range []string{"bytes", "array", "nested-array", "named-bytes", "alias-bytes", "named-array", "array-override", "named-bytes-override"} {
		t.Run(shape, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				key := mapAdmissionKey(shape)
				Service("mapping", func() {
					Method("send", func() {
						Payload(func() {
							Attribute("entries", MapOf(key, String, func() {
								if shape == "array-override" || shape == "named-bytes-override" {
									Key(func() {
										Meta("struct:field:type", "custom.Key")
									})
								}
							}))
						})
					})
				})
			}, nil))
			err := eval.RunDSL()
			require.Error(t, err, "framework-owned slice keys must fail before generation")
			require.Equal(t, 1, strings.Count(err.Error(), "map keys cannot"), err.Error())
			require.Contains(t, err.Error(), "map_key_admission_test.go:")
			require.Contains(t, err.Error(), "entries")
		})
	}
}

func TestMapKeyAdmissionPreservesComparableContracts(t *testing.T) {
	for _, shape := range []string{"string", "boolean", "integer", "float", "any", "named-string", "object", "object-with-slice", "custom-string", "custom-bytes", "named-custom-bytes"} {
		t.Run(shape, func(t *testing.T) {
			root := expr.RunDSL(t, func() {
				key := mapAdmissionKey(shape)
				Service("mapping", func() {
					Method("send", func() {
						Payload(MapOf(key, String, func() {
							if shape == "custom-string" || shape == "custom-bytes" {
								Key(func() {
									Meta("struct:field:type", "custom.Key", "example.com/keys")
								})
							}
						}))
					})
				})
			})
			require.NotNil(t, expr.AsMap(root.Services[0].Methods[0].Payload.Type))
		})
	}
}

func TestMapKeyAdmissionIsOccurrenceScoped(t *testing.T) {
	for _, mapFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "value-first", true: "key-first"}[mapFirst], func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				key := Type("SharedBytes", Bytes)
				Service("mapping", func() {
					defineValue := func() {
						Method("ordinary", func() {
							Payload(key)
						})
					}
					defineMap := func() {
						Method("indexed", func() {
							Payload(MapOf(key, String))
						})
					}
					if mapFirst {
						defineMap()
						defineValue()
					} else {
						defineValue()
						defineMap()
					}
				})
			}, nil))
			err := eval.RunDSL()
			require.Error(t, err)
			require.Equal(t, 1, strings.Count(err.Error(), "map keys cannot"), err.Error())
			require.Contains(t, err.Error(), "map_key_admission_test.go:")
			require.Contains(t, err.Error(), "indexed")
		})
	}
}

func TestMapKeyAdmissionAcrossDesignRoots(t *testing.T) {
	for _, role := range []string{"result", "streaming-payload", "streaming-result", "error", "unused-type", "nested-value", "array-element"} {
		t.Run(role, func(t *testing.T) {
			expr.SetupTestDSL(t)
			require.True(t, eval.Execute(func() {
				mapping := MapOf(Bytes, String)
				if role == "unused-type" {
					Type("Unused", mapping, func() {
					})
				}
				Service("mapping", func() {
					Method("send", func() {
						switch role {
						case "result":
							Result(mapping)
						case "streaming-payload":
							StreamingPayload(mapping)
						case "streaming-result":
							StreamingResult(mapping)
						case "error":
							Error("broken", mapping)
						case "nested-value":
							Payload(MapOf(String, mapping))
						case "array-element":
							Payload(ArrayOf(mapping))
						}
					})
				})
			}, nil))
			err := eval.RunDSL()
			require.Error(t, err)
			require.Equal(t, 1, strings.Count(err.Error(), "map keys cannot"), err.Error())
			if role == "error" {
				require.Contains(t, err.Error(), "broken")
			} else {
				require.Contains(t, err.Error(), "map_key_admission_test.go:")
			}
		})
	}
}

func TestMapKeyAdmissionConstructedMap(t *testing.T) {
	expr.SetupTestDSL(t)
	mapping := &expr.AttributeExpr{Type: &expr.Map{
		KeyType: &expr.AttributeExpr{Type: &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.String},
			ElemType: &expr.AttributeExpr{Type: expr.String},
		}},
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}
	root := &expr.RootExpr{
		API:   &expr.APIExpr{Name: "mapping"},
		Types: []expr.UserType{&expr.UserTypeExpr{AttributeExpr: mapping, TypeName: "Entries"}},
	}
	err := root.Validate()
	require.Error(t, err)
	require.Equal(t, 1, strings.Count(err.Error(), "map keys cannot"), err.Error())
	require.Contains(t, err.Error(), "Entries")
}

func mapAdmissionKey(shape string) expr.DataType {
	switch shape {
	case "bytes", "custom-bytes":
		return Bytes
	case "array", "array-override":
		return ArrayOf(String)
	case "nested-array":
		return ArrayOf(ArrayOf(String))
	case "named-bytes", "named-bytes-override":
		return Type("Key", Bytes)
	case "named-custom-bytes":
		return Type("Key", Bytes, func() {
			Meta("struct:field:type", "custom.Key", "example.com/keys")
		})
	case "alias-bytes":
		return Type("Key", Type("Inner", Bytes))
	case "named-array":
		return Type("Key", ArrayOf(String))
	case "named-string":
		return Type("Key", String)
	case "object", "object-with-slice":
		return Type("Key", func() {
			Attribute("name", String)
			if shape == "object-with-slice" {
				Attribute("values", ArrayOf(String))
			}
		})
	case "any":
		return Any
	case "boolean":
		return Boolean
	case "integer":
		return Int64
	case "float":
		return Float64
	default:
		return String
	}
}
