package valuecontract

import (
	"encoding/json/jsontext"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
)

type (
	referenceAliasValue struct {
		Origin              uint64                       `json:"origin"`
		Semantic            uint64                       `json:"semantic"`
		Number              *referenceDecimal            `json:"number"`
		SatisfiedPredicates []referencePredicateIdentity `json:"satisfiedPredicates"`
	}
	referencePredicateIdentity struct {
		Kind      string `json:"kind"`
		Predicate uint64 `json:"predicate"`
	}
	referencePredicateClause struct {
		Kind      string `json:"kind"`
		Predicate uint64 `json:"predicate"`
		Origin    uint64 `json:"origin"`
	}
	referenceAliasNumeric struct {
		Minimum          *referenceDecimal `json:"minimum"`
		ExclusiveMinimum *referenceDecimal `json:"exclusiveMinimum"`
		Maximum          *referenceDecimal `json:"maximum"`
		ExclusiveMaximum *referenceDecimal `json:"exclusiveMaximum"`
	}
	referenceAliasRequired struct {
		Declaration uint64 `json:"declaration"`
		Field       uint64 `json:"field"`
		Origin      uint64 `json:"origin"`
	}
	requiredObservation struct {
		field  uint64
		origin uint64
	}
	predicateObservation struct {
		kind   string
		value  string
		origin uint64
	}
	referenceAliasLayer struct {
		Declaration  uint64                     `json:"declaration"`
		Enumeration  *[]referenceAliasValue     `json:"enumeration"`
		DefaultValue *referenceAliasValue       `json:"defaultValue"`
		Numeric      referenceAliasNumeric      `json:"numeric"`
		Predicates   []referencePredicateClause `json:"predicates"`
		Required     []referenceAliasRequired   `json:"required"`
		name         string
	}
	referenceEffectiveAlias struct {
		Enumeration  *[]referenceAliasValue     `json:"enumeration"`
		DefaultValue *referenceAliasValue       `json:"defaultValue"`
		Numeric      []referenceAliasNumeric    `json:"numericLayers"`
		Predicates   []referencePredicateClause `json:"predicates"`
		Required     []referenceAliasRequired   `json:"required"`
	}
	referenceAliasOutcome struct {
		OK    *referenceEffectiveAlias `json:"ok"`
		Error jsontext.Value           `json:"error"`
	}
	independentAliasLayer struct {
		name      string
		attribute *expr.AttributeExpr
	}
	independentSemanticClass struct {
		id    uint64
		value independentResolvedValue
	}
	independentPredicateClass struct {
		identity referencePredicateIdentity
		value    string
	}
	independentAliasExtractor struct {
		t          *testing.T
		attribute  *expr.AttributeExpr
		classes    []independentSemanticClass
		predicates []independentPredicateClass
		nextOrigin uint64
	}
)

func TestIndependentAliasContractExtractionPreservesRawLayers(t *testing.T) {
	minimum, exclusiveMinimum := 5.0, 1.0
	base := aliasNamed("Base", expr.Float64, &expr.ValidationExpr{
		Values: []any{1.0, 2.0}, Minimum: &minimum, Pattern: "base", Format: expr.FormatUUID,
		PatternClauses: []string{"base", "base-carrier"},
		FormatClauses:  []expr.ValidationFormat{expr.FormatUUID, expr.FormatDateTime},
		Required:       []string{"missing"},
	})
	derived := aliasNamed("Derived", base, &expr.ValidationExpr{
		Values: []any{2.0, 2.0}, ExclusiveMinimum: &exclusiveMinimum, Pattern: "derived",
		PatternClauses: []string{"derived", "derived-carrier"},
		FormatClauses:  []expr.ValidationFormat{expr.FormatUUID},
	})
	derived.Attribute().DefaultValue = 2.0
	layers, _ := extractIndependentAliasLayers(t, &expr.AttributeExpr{Type: derived})
	require.Len(t, layers, 3)
	require.Equal(t, []string{"Base", "Derived", "attribute"}, []string{layers[0].name, layers[1].name, layers[2].name})
	require.NotNil(t, layers[0].Enumeration)
	require.NotNil(t, layers[1].Enumeration)
	require.Len(t, *layers[1].Enumeration, 2, "duplicate authored enum entries must remain distinct")
	require.NotNil(t, layers[0].Numeric.Minimum)
	require.Nil(t, layers[0].Numeric.ExclusiveMinimum)
	require.Nil(t, layers[1].Numeric.Minimum)
	require.NotNil(t, layers[1].Numeric.ExclusiveMinimum)
	require.Equal(t, []referencePredicateClause{
		{Kind: "pattern", Predicate: 1, Origin: 1},
		{Kind: "pattern", Predicate: 1, Origin: 1},
		{Kind: "pattern", Predicate: 2, Origin: 1},
		{Kind: "format", Predicate: 1, Origin: 1},
		{Kind: "format", Predicate: 1, Origin: 1},
		{Kind: "format", Predicate: 2, Origin: 1},
	}, layers[0].Predicates)
	require.Equal(t, []referencePredicateClause{
		{Kind: "pattern", Predicate: 3, Origin: 2},
		{Kind: "pattern", Predicate: 3, Origin: 2},
		{Kind: "pattern", Predicate: 4, Origin: 2},
		{Kind: "format", Predicate: 1, Origin: 2},
	}, layers[1].Predicates)
	require.NotNil(t, layers[1].DefaultValue)
	require.Nil(t, layers[2].Enumeration, "absence must not become an empty declaration")
}

func checkEffectiveAliasContractConformance(t *testing.T, executable string) {
	t.Helper()
	t.Run("pattern format conjunction order and dedup", func(t *testing.T) {
		const uuidPattern = `^[[:xdigit:]-]+$`
		const currentPattern = `^550e`
		const validUUID = "550e8400-e29b-41d4-a716-446655440000"
		const otherUUID = "123e4567-e89b-42d3-a456-426614174000"
		base := aliasNamed("PredicateBase", expr.String, &expr.ValidationExpr{
			Values: []any{validUUID, otherUUID}, Pattern: uuidPattern, Format: expr.FormatUUID,
		})
		middle := aliasNamed("PredicateMiddle", base, &expr.ValidationExpr{
			Pattern: uuidPattern, Format: expr.FormatUUID,
		})
		derived := aliasNamed("PredicateDerived", middle, &expr.ValidationExpr{
			Values: []any{validUUID}, Pattern: currentPattern, Format: expr.FormatUUID,
		})
		derived.Attribute().DefaultValue = validUUID
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("pattern and format same text remain distinct", func(t *testing.T) {
		base := aliasNamed("PredicateKindBase", expr.String, &expr.ValidationExpr{Pattern: "uuid"})
		derived := aliasNamed("PredicateKindDerived", base, &expr.ValidationExpr{Format: expr.FormatUUID})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("enum rejected by inherited pattern", func(t *testing.T) {
		base := aliasNamed("PatternEnumBase", expr.String, &expr.ValidationExpr{Pattern: `^a`})
		derived := aliasNamed("PatternEnumDerived", base, &expr.ValidationExpr{
			Values: []any{"buzz"}, Pattern: `z$`,
		})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidEnum", "enum member")
	})

	t.Run("default rejected by inherited pattern", func(t *testing.T) {
		base := aliasNamed("PatternDefaultBase", expr.String, &expr.ValidationExpr{Pattern: `^a`})
		derived := aliasNamed("PatternDefaultDerived", base, &expr.ValidationExpr{Pattern: `z$`})
		derived.Attribute().DefaultValue = "buzz"
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value")
	})

	t.Run("enum rejected by inherited format", func(t *testing.T) {
		base := aliasNamed("FormatEnumBase", expr.String, &expr.ValidationExpr{Format: expr.FormatUUID})
		derived := aliasNamed("FormatEnumDerived", base, &expr.ValidationExpr{
			Values: []any{"2026-09-29T12:00:00Z"}, Format: expr.FormatDateTime,
		})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidEnum", "enum member")
	})

	t.Run("default rejected by inherited format", func(t *testing.T) {
		base := aliasNamed("FormatDefaultBase", expr.String, &expr.ValidationExpr{Format: expr.FormatUUID})
		derived := aliasNamed("FormatDefaultDerived", base, &expr.ValidationExpr{Format: expr.FormatDateTime})
		derived.Attribute().DefaultValue = "2026-09-29T12:00:00Z"
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value")
	})

	t.Run("valid narrowing and replacement are not retroactive", func(t *testing.T) {
		base := aliasNamed("PrefixValidBase", expr.String, &expr.ValidationExpr{
			Values: []any{"ax", "abz"}, Pattern: `^a`,
		})
		base.Attribute().DefaultValue = "ax"
		derived := aliasNamed("PrefixValidDerived", base, &expr.ValidationExpr{
			Values: []any{"abz"}, Pattern: `z$`,
		})
		derived.Attribute().DefaultValue = "abz"
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("invalid superseded ancestor enum remains rejected", func(t *testing.T) {
		base := aliasNamed("InvalidAncestorEnumBase", expr.String, &expr.ValidationExpr{
			Values: []any{"xb", "ab"}, Pattern: `^a`,
		})
		derived := aliasNamed("InvalidAncestorEnumDerived", base,
			&expr.ValidationExpr{Values: []any{"ab"}})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidEnum", "enum member")
	})

	t.Run("inherited enum composes with derived predicate", func(t *testing.T) {
		base := aliasNamed("InheritedEnumBase", expr.String, &expr.ValidationExpr{
			Values: []any{"ab", "ax"}, Pattern: `^a`,
		})
		derived := aliasNamed("InheritedEnumDerived", base, &expr.ValidationExpr{Pattern: `b$`})
		attribute := &expr.AttributeExpr{Type: derived}
		checkAliasSuccess(t, executable, attribute)
		checkAliasResolutionOutcome(t, attribute, "ab", expr.ValueResolved)
		checkAliasResolutionOutcome(t, attribute, "ax", expr.ValueInvalid)
	})

	t.Run("locally authored enum is checked by derived predicate", func(t *testing.T) {
		base := aliasNamed("LocalEnumBase", expr.String, &expr.ValidationExpr{
			Values: []any{"ab", "ax"}, Pattern: `^a`,
		})
		derived := aliasNamed("LocalEnumDerived", base, &expr.ValidationExpr{
			Values: []any{"ab", "ax"}, Pattern: `b$`,
		})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidEnum", "enum member")
	})

	t.Run("inherited default is checked by derived predicate", func(t *testing.T) {
		base := aliasNamed("InheritedPredicateDefaultBase", expr.String, &expr.ValidationExpr{
			Values: []any{"ab", "ax"}, Pattern: `^a`,
		})
		base.Attribute().DefaultValue = "ax"
		derived := aliasNamed("InheritedPredicateDefaultDerived", base,
			&expr.ValidationExpr{Pattern: `b$`})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value")
	})

	t.Run("local default replaces inherited value under derived predicate", func(t *testing.T) {
		base := aliasNamed("ReplacedPredicateDefaultBase", expr.String, &expr.ValidationExpr{
			Values: []any{"ab", "ax"}, Pattern: `^a`,
		})
		base.Attribute().DefaultValue = "ax"
		derived := aliasNamed("ReplacedPredicateDefaultDerived", base,
			&expr.ValidationExpr{Pattern: `b$`})
		derived.Attribute().DefaultValue = "ab"
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("invalid superseded ancestor default remains rejected", func(t *testing.T) {
		base := aliasNamed("InvalidAncestorDefaultBase", expr.String, &expr.ValidationExpr{Pattern: `^a`})
		base.Attribute().DefaultValue = "xb"
		derived := aliasNamed("InvalidAncestorDefaultDerived", base, nil)
		derived.Attribute().DefaultValue = "ab"
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value")
	})

	t.Run("invalid middle inherited default remains rejected", func(t *testing.T) {
		base := aliasNamed("InvalidMiddleBase", expr.String, &expr.ValidationExpr{Pattern: `^a`})
		base.Attribute().DefaultValue = "ax"
		middle := aliasNamed("InvalidMiddle", base, &expr.ValidationExpr{Pattern: `z$`})
		derived := aliasNamed("InvalidMiddleDerived", middle, nil)
		derived.Attribute().DefaultValue = "abz"
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value")
	})

	t.Run("declared equality subset default and numeric", func(t *testing.T) {
		minimum, exclusiveMinimum, maximum, exclusiveMaximum := 0.0, 1.0, 10.0, 8.0
		base := aliasNamed("Base", expr.Float64, &expr.ValidationExpr{
			Values: []any{1.0, 2.0}, Minimum: &minimum, Maximum: &maximum,
		})
		base.Attribute().DefaultValue = 1.0
		derived := aliasNamed("Derived", base, &expr.ValidationExpr{
			Values: []any{2.0}, ExclusiveMinimum: &exclusiveMinimum,
			ExclusiveMaximum: &exclusiveMaximum,
		})
		derived.Attribute().DefaultValue = 2.0
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("inherited default fallback", func(t *testing.T) {
		base := aliasNamed("FallbackBase", expr.Int, &expr.ValidationExpr{Values: []any{1, 2}})
		base.Attribute().DefaultValue = 2
		derived := aliasNamed("FallbackDerived", base, &expr.ValidationExpr{Values: []any{2}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("bytes declared normalization", func(t *testing.T) {
		base := aliasNamed("BlobBase", expr.Bytes, &expr.ValidationExpr{Values: []any{[]byte("ok")}})
		derived := aliasNamed("BlobDerived", base, &expr.ValidationExpr{Values: []any{"ok"}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("exact integer semantic classes", func(t *testing.T) {
		const exactSigned = int64(9_007_199_254_740_993)
		const exactUnsigned = uint64(math.MaxUint64)
		for _, contract := range []struct {
			name       string
			primitive  expr.Primitive
			baseValue  any
			childValue any
		}{
			{"signed", expr.Int64, exactSigned, uint64(exactSigned)},
			{"unsigned", expr.UInt64, exactUnsigned, exactUnsigned},
		} {
			base := aliasNamed(contract.name+"Base", contract.primitive,
				&expr.ValidationExpr{Values: []any{contract.baseValue}})
			derived := aliasNamed(contract.name+"Derived", base,
				&expr.ValidationExpr{Values: []any{contract.childValue}})
			checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
		}
	})

	t.Run("float32 declared precision", func(t *testing.T) {
		base := aliasNamed("FloatBase", expr.Float32, &expr.ValidationExpr{Values: []any{float32(0.1)}})
		derived := aliasNamed("FloatDerived", base, &expr.ValidationExpr{Values: []any{float64(0.1)}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("any deep equality", func(t *testing.T) {
		baseValue := map[string]any{"count": int64(1), "items": []string{"x", "y"}}
		derivedValue := map[string]any{"items": [2]string{"x", "y"}, "count": uint64(1)}
		base := aliasNamed("AnyBase", expr.Any, &expr.ValidationExpr{Values: []any{baseValue}})
		derived := aliasNamed("AnyDerived", base, &expr.ValidationExpr{Values: []any{derivedValue}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("opaque root enum and default", func(t *testing.T) {
		value := independentOpaqueFixture{Name: "root", Version: 1}
		attribute := &expr.AttributeExpr{
			Type:         expr.Any,
			Validation:   &expr.ValidationExpr{Values: []any{value}},
			DefaultValue: independentOpaqueFixture{Name: "root", Version: 1},
		}
		checkAliasSuccess(t, executable, attribute)

		invalid := &expr.AttributeExpr{
			Type:         expr.Any,
			Validation:   &expr.ValidationExpr{Values: []any{value}},
			DefaultValue: independentOpaqueFixture{Name: "different", Version: 1},
		}
		checkAliasRejection(t, executable, invalid, "invalidDefault", "default value")
	})

	t.Run("opaque multi-hop refinement and inherited default", func(t *testing.T) {
		same := &independentOpaqueFixture{Name: "same", Version: 1}
		alternative := &independentOpaqueFixture{Name: "alternative", Version: 1}
		base := aliasNamed("OpaqueBase", expr.Any, &expr.ValidationExpr{Values: []any{same, alternative}})
		base.Attribute().DefaultValue = same
		middle := aliasNamed("OpaqueMiddle", base, nil)
		valid := aliasNamed("OpaqueValid", middle, &expr.ValidationExpr{
			Values: []any{&independentOpaqueFixture{Name: "same", Version: 1}},
		})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: valid})

		widened := aliasNamed("OpaqueWidened", middle, &expr.ValidationExpr{
			Values: []any{&independentOpaqueFixture{Name: "outside", Version: 1}},
		})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: widened}, "enumWidening", "enum member")

		inheritedMismatch := aliasNamed("OpaqueInheritedMismatch", middle, &expr.ValidationExpr{
			Values: []any{&independentOpaqueFixture{Name: "alternative", Version: 1}},
		})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: inheritedMismatch}, "invalidDefault", "default value")

		localMismatch := aliasNamed("OpaqueLocalMismatch", middle, &expr.ValidationExpr{
			Values: []any{&independentOpaqueFixture{Name: "same", Version: 1}},
		})
		localMismatch.Attribute().DefaultValue = &independentOpaqueFixture{Name: "different", Version: 1}
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: localMismatch}, "invalidDefault", "default value")
	})

	t.Run("opaque object leaf keeps known sibling semantics", func(t *testing.T) {
		const exact = int64(9_007_199_254_740_993)
		object := &expr.Object{
			{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.Bytes}},
			{Name: "count", Attribute: &expr.AttributeExpr{Type: expr.Int64}},
			{Name: "opaque", Attribute: &expr.AttributeExpr{Type: expr.Any}},
		}
		baseValue := map[string]any{
			"blob:b": []byte("ok"), "count": exact,
			"opaque": independentOpaqueFixture{Name: "same", Version: 1},
		}
		base := aliasNamed("OpaqueObjectBase", object, &expr.ValidationExpr{Values: []any{baseValue}})
		validValue := map[string]any{
			"b": "ok", "count": uint64(exact),
			"opaque": independentOpaqueFixture{Name: "same", Version: 1},
		}
		valid := aliasNamed("OpaqueObjectValid", base, &expr.ValidationExpr{Values: []any{validValue}})
		valid.Attribute().DefaultValue = validValue
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: valid})

		for _, mismatch := range []struct {
			name  string
			value map[string]any
		}{
			{"opaque", map[string]any{"b": "ok", "count": uint64(exact), "opaque": independentOpaqueFixture{Name: "different", Version: 1}}},
			{"bytes", map[string]any{"b": "different", "count": uint64(exact), "opaque": independentOpaqueFixture{Name: "same", Version: 1}}},
			{"integer", map[string]any{"b": "ok", "count": uint64(exact + 1), "opaque": independentOpaqueFixture{Name: "same", Version: 1}}},
		} {
			t.Run(mismatch.name+" mismatch", func(t *testing.T) {
				derived := aliasNamed("OpaqueObjectMismatch", base,
					&expr.ValidationExpr{Values: []any{mismatch.value}})
				checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "enumWidening", "enum member")
			})
		}
	})

	t.Run("opaque array leaf keeps known sibling semantics", func(t *testing.T) {
		array := &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.Any}}
		baseValue := []any{int32(1), &independentOpaqueFixture{Name: "same", Version: 1}}
		base := aliasNamed("OpaqueArrayBase", array, &expr.ValidationExpr{Values: []any{baseValue}})
		validValue := []any{int(1), &independentOpaqueFixture{Name: "same", Version: 1}}
		valid := aliasNamed("OpaqueArrayValid", base, &expr.ValidationExpr{Values: []any{validValue}})
		valid.Attribute().DefaultValue = validValue
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: valid})

		for _, mismatch := range []struct {
			name  string
			value []any
		}{
			{"opaque", []any{int(1), &independentOpaqueFixture{Name: "different", Version: 1}}},
			{"integer", []any{int(2), &independentOpaqueFixture{Name: "same", Version: 1}}},
		} {
			t.Run(mismatch.name+" mismatch", func(t *testing.T) {
				derived := aliasNamed("OpaqueArrayMismatch", base,
					&expr.ValidationExpr{Values: []any{mismatch.value}})
				checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "enumWidening", "enum member")
			})
		}
	})

	t.Run("array nil and empty equality", func(t *testing.T) {
		array := &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}}
		base := aliasNamed("ArrayBase", array, &expr.ValidationExpr{Values: []any{[]string(nil)}})
		derived := aliasNamed("ArrayDerived", base, &expr.ValidationExpr{Values: []any{[]string{}}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("map order independent equality", func(t *testing.T) {
		mapping := &expr.Map{
			KeyType:  &expr.AttributeExpr{Type: expr.String},
			ElemType: &expr.AttributeExpr{Type: expr.Int64},
		}
		base := aliasNamed("MapBase", mapping,
			&expr.ValidationExpr{Values: []any{map[string]int64{"one": 1, "two": 2}}})
		derived := aliasNamed("MapDerived", base,
			&expr.ValidationExpr{Values: []any{map[string]uint64{"two": 2, "one": 1}}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("object authored and wire names", func(t *testing.T) {
		object := &expr.Object{
			{Name: "blob:b", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}
		base := aliasNamed("ObjectValueBase", object,
			&expr.ValidationExpr{Values: []any{map[string]any{"blob:b": "value"}}})
		derived := aliasNamed("ObjectValueDerived", base,
			&expr.ValidationExpr{Values: []any{map[string]any{"b": "value"}}})
		derived.Attribute().DefaultValue = map[string]any{"b": "value"}
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("nullable enum member", func(t *testing.T) {
		base := aliasNamed("NullableBase", expr.String, &expr.ValidationExpr{Values: []any{nil, "set"}})
		base.Attribute().Nullable = true
		derived := aliasNamed("NullableDerived", base, &expr.ValidationExpr{Values: []any{nil}})
		derived.Attribute().Nullable = true
		attribute := &expr.AttributeExpr{Type: derived, Nullable: true}
		checkAliasSuccess(t, executable, attribute)
	})

	t.Run("required identity and provenance union", func(t *testing.T) {
		object := &expr.Object{
			{Name: "first:one", Attribute: &expr.AttributeExpr{Type: expr.String}},
			{Name: "second:two", Attribute: &expr.AttributeExpr{Type: expr.String}},
		}
		base := aliasNamed("ObjectBase", object, &expr.ValidationExpr{Required: []string{"first:one"}})
		derived := aliasNamed("ObjectDerived", base, &expr.ValidationExpr{Required: []string{"second:two", "first:one"}})
		checkAliasSuccess(t, executable, &expr.AttributeExpr{Type: derived})
	})

	t.Run("widening rejected", func(t *testing.T) {
		base := aliasNamed("EnumBase", expr.Int, &expr.ValidationExpr{Values: []any{1, 2}})
		derived := aliasNamed("EnumDerived", base, &expr.ValidationExpr{Values: []any{2, 3}})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "enumWidening", "enum member 3")
	})

	t.Run("disjoint enum rejected", func(t *testing.T) {
		base := aliasNamed("DisjointBase", expr.Int, &expr.ValidationExpr{Values: []any{1, 2}})
		derived := aliasNamed("DisjointDerived", base, &expr.ValidationExpr{Values: []any{3, 4}})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "enumWidening", "enum member 3")
	})

	t.Run("inherited default rejected after narrowing", func(t *testing.T) {
		base := aliasNamed("DefaultBase", expr.Int, &expr.ValidationExpr{Values: []any{1, 2}})
		base.Attribute().DefaultValue = 1
		derived := aliasNamed("DefaultDerived", base, &expr.ValidationExpr{Values: []any{2}})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value 1")
	})

	t.Run("inherited default rejected by tighter numeric contract", func(t *testing.T) {
		baseMinimum := 0.0
		base := aliasNamed("NumericDefaultBase", expr.Float64, &expr.ValidationExpr{Minimum: &baseMinimum})
		base.Attribute().DefaultValue = 1.0
		derivedMinimum := 2.0
		derived := aliasNamed("NumericDefaultDerived", base, &expr.ValidationExpr{Minimum: &derivedMinimum})
		checkAliasRejection(t, executable, &expr.AttributeExpr{Type: derived}, "invalidDefault", "default value 1")
	})
}

func checkAliasSuccess(t *testing.T, executable string, attribute *expr.AttributeExpr) {
	t.Helper()
	layers, extractor := extractIndependentAliasLayers(t, attribute)
	result := runAliasReference(t, executable, layers)
	require.NotNil(t, result.OK, "reference rejected independently extracted layers: %s", result.Error)
	require.Empty(t, result.Error)

	production, err := expr.EffectiveConstraintsFor(attribute)
	require.NoError(t, err)
	effectiveValidation := production.Validation()
	validation := effectiveValidation.Lowered()

	wantEnum := semanticIDs(result.OK.Enumeration)
	allEnums := validation.Enums()
	wantEnumPresent := result.OK.Enumeration != nil
	_, candidatePresent := production.EnumCandidates()
	require.Equal(t, wantEnumPresent, candidatePresent)
	require.Equal(t, wantEnumPresent, len(allEnums) > 0)
	gotEnum := []uint64{}
	if len(allEnums) > 0 {
		gotEnum = make([]uint64, len(allEnums[0]))
		for index, raw := range allEnums[0] {
			gotEnum[index] = extractor.semantic(raw).Semantic
		}
	}
	require.Equal(t, wantEnum, gotEnum)

	wantPredicates := make([]predicateObservation, len(result.OK.Predicates))
	for index, clause := range result.OK.Predicates {
		wantPredicates[index] = predicateObservation{
			kind: clause.Kind, value: extractor.predicateValue(clause.Kind, clause.Predicate),
			origin: clause.Origin,
		}
	}
	gotPredicates := make([]predicateObservation, len(effectiveValidation.Clauses()))
	for index, clause := range effectiveValidation.Clauses() {
		gotPredicates[index] = predicateObservation{
			kind: productionPredicateKind(t, clause.Kind), value: clause.Value,
			origin: layerIDByName(layers, clause.Provenance.Declaration),
		}
	}
	require.Equal(t, wantPredicates, gotPredicates)
	checkLoweredValidationRoundTrip(t, extractor, effectiveValidation, wantPredicates)

	wantDefault := result.OK.DefaultValue
	actualDefault, present := production.Default()
	require.Equal(t, wantDefault != nil, present)
	if present {
		require.Equal(t, wantDefault.Semantic, extractor.semantic(actualDefault).Semantic)
	}

	lower, upper := independentEffectiveBounds(attribute)
	requireEffectiveBoundEqual(t, lower, production.LowerBound())
	requireEffectiveBoundEqual(t, upper, production.UpperBound())

	wantRequired := make([]requiredObservation, len(result.OK.Required))
	for index, field := range result.OK.Required {
		wantRequired[index] = requiredObservation{field: field.Field, origin: field.Origin}
	}
	gotRequired := make([]requiredObservation, len(production.Required()))
	for index, field := range production.Required() {
		fieldID := independentFieldID(attribute, field.Name)
		gotRequired[index] = requiredObservation{
			field:  fieldID,
			origin: layerIDByName(layers, field.Provenance.Declaration),
		}
	}
	require.Equal(t, wantRequired, gotRequired)
}

func checkAliasRejection(t *testing.T, executable string, attribute *expr.AttributeExpr, referenceError, productionError string) {
	t.Helper()
	layers, _ := extractIndependentAliasLayers(t, attribute)
	result := runAliasReference(t, executable, layers)
	require.Nil(t, result.OK)
	require.Contains(t, string(result.Error), referenceError)
	_, err := expr.EffectiveConstraintsFor(attribute)
	require.ErrorContains(t, err, productionError)
}

func checkAliasResolutionOutcome(
	t *testing.T,
	attribute *expr.AttributeExpr,
	raw any,
	want expr.ValueOutcome,
) {
	t.Helper()
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	result := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: raw}), expr.ValueRoleExample)
	require.Equal(t, want, result.Outcome(), "%v", result.Diagnostics())
}

func runAliasReference(t *testing.T, executable string, layers []referenceAliasLayer) referenceAliasOutcome {
	t.Helper()
	command := referenceConstructor("aliasContract", map[string]any{"layers": layers})
	result := runReference(t, executable, []any{command})[0]
	return referenceDecode[referenceAliasOutcome](t, result)
}

func extractIndependentAliasLayers(t *testing.T, attribute *expr.AttributeExpr) ([]referenceAliasLayer, *independentAliasExtractor) {
	t.Helper()
	chain := rawAliasAncestry(attribute)
	leaf := chain[len(chain)-1].attribute
	semanticAttribute := &expr.AttributeExpr{Type: leaf.Type, Nullable: leaf.Nullable}
	extractor := &independentAliasExtractor{t: t, attribute: semanticAttribute}

	slices.Reverse(chain)
	for _, raw := range chain {
		if validation := raw.attribute.Validation; validation != nil {
			for _, predicate := range independentRawPredicates(validation) {
				extractor.predicate(predicate.Kind, predicate.Value)
			}
		}
	}
	layers := make([]referenceAliasLayer, len(chain))
	for index, raw := range chain {
		layerID := uint64(index + 1)
		layer := referenceAliasLayer{Declaration: layerID, name: raw.name}
		validation := raw.attribute.Validation
		if validation != nil {
			for _, predicate := range independentRawPredicates(validation) {
				identity := extractor.predicate(predicate.Kind, predicate.Value)
				layer.Predicates = append(layer.Predicates, referencePredicateClause{
					Kind: identity.Kind, Predicate: identity.Predicate, Origin: layerID,
				})
			}
			if validation.Values != nil {
				values := make([]referenceAliasValue, len(validation.Values))
				for valueIndex, value := range validation.Values {
					values[valueIndex] = extractor.semantic(value)
				}
				layer.Enumeration = &values
			}
			layer.Numeric = rawNumeric(validation)
			for _, name := range validation.Required {
				field := independentFieldID(attribute, name)
				if field != 0 {
					extractor.nextOrigin++
					layer.Required = append(layer.Required, referenceAliasRequired{
						Declaration: field, Field: field, Origin: layerID,
					})
				}
			}
		}
		if raw.attribute.DefaultValue != nil {
			value := extractor.semantic(raw.attribute.DefaultValue)
			layer.DefaultValue = &value
		}
		layers[index] = layer
	}
	return layers, extractor
}

type independentRawPredicate struct {
	Kind  string
	Value string
}

func independentRawPredicates(validation *expr.ValidationExpr) []independentRawPredicate {
	var result []independentRawPredicate
	appendAuthored := func(kind, value string) {
		if value == "" {
			return
		}
		result = append(result, independentRawPredicate{Kind: kind, Value: value})
	}
	appendAuthored("pattern", validation.Pattern)
	for _, pattern := range validation.PatternClauses {
		appendAuthored("pattern", pattern)
	}
	appendAuthored("format", string(validation.Format))
	for _, format := range validation.FormatClauses {
		appendAuthored("format", string(format))
	}
	return result
}

func rawAliasAncestry(attribute *expr.AttributeExpr) []independentAliasLayer {
	layers := []independentAliasLayer{{name: "attribute", attribute: attribute}}
	current := attribute
	seen := make(map[expr.UserType]bool)
	for {
		named, ok := current.Type.(expr.UserType)
		if !ok || seen[named] {
			return layers
		}
		seen[named] = true
		current = named.Attribute()
		layers = append(layers, independentAliasLayer{name: named.Name(), attribute: current})
	}
}

func (e *independentAliasExtractor) semantic(raw any) referenceAliasValue {
	e.t.Helper()
	e.nextOrigin++
	value, err := independentResolve(e.attribute, raw)
	require.NoError(e.t, err, "independent semantic normalization failed")
	for _, class := range e.classes {
		if independentResolvedEqual(value, class.value) {
			return referenceAliasValue{
				Origin: e.nextOrigin, Semantic: class.id, Number: independentDecimal(value),
				SatisfiedPredicates: e.satisfiedPredicates(value),
			}
		}
	}
	id := uint64(len(e.classes) + 1)
	e.classes = append(e.classes, independentSemanticClass{id: id, value: value})
	return referenceAliasValue{
		Origin: e.nextOrigin, Semantic: id, Number: independentDecimal(value),
		SatisfiedPredicates: e.satisfiedPredicates(value),
	}
}

func (e *independentAliasExtractor) predicate(kind, value string) referencePredicateIdentity {
	e.t.Helper()
	for _, predicate := range e.predicates {
		if predicate.identity.Kind == kind && predicate.value == value {
			return predicate.identity
		}
	}
	index := uint64(1)
	for _, predicate := range e.predicates {
		if predicate.identity.Kind == kind && predicate.identity.Predicate >= index {
			index = predicate.identity.Predicate + 1
		}
	}
	identity := referencePredicateIdentity{Kind: kind, Predicate: index}
	e.predicates = append(e.predicates, independentPredicateClass{identity: identity, value: value})
	return identity
}

func (e *independentAliasExtractor) predicateValue(kind string, id uint64) string {
	e.t.Helper()
	for _, predicate := range e.predicates {
		if predicate.identity.Kind == kind && predicate.identity.Predicate == id {
			return predicate.value
		}
	}
	e.t.Fatalf("unknown independent predicate %s/%d", kind, id)
	return ""
}

func (e *independentAliasExtractor) satisfiedPredicates(value independentResolvedValue) []referencePredicateIdentity {
	e.t.Helper()
	text, ok := value.scalar.(string)
	if !ok {
		return nil
	}
	var result []referencePredicateIdentity
	for _, predicate := range e.predicates {
		if independentPredicateAllows(e.t, predicate.identity.Kind, predicate.value, text) {
			result = append(result, predicate.identity)
		}
	}
	return result
}

func independentPredicateAllows(t *testing.T, kind, predicate, value string) bool {
	t.Helper()
	switch kind {
	case "pattern":
		compiled, err := regexp.Compile(predicate)
		require.NoError(t, err, "reference pattern must compile independently")
		return compiled.MatchString(value)
	case "format":
		switch expr.ValidationFormat(predicate) {
		case expr.FormatUUID:
			matched, err := regexp.MatchString(
				`^[[:xdigit:]]{8}-[[:xdigit:]]{4}-[1-5][[:xdigit:]]{3}-[89abAB][[:xdigit:]]{3}-[[:xdigit:]]{12}$`,
				value,
			)
			require.NoError(t, err)
			return matched
		case expr.FormatDateTime:
			_, err := time.Parse(time.RFC3339, value)
			return err == nil
		default:
			t.Fatalf("independent predicate adapter does not model format %q", predicate)
			return false
		}
	default:
		t.Fatalf("unknown independent predicate kind %q", kind)
		return false
	}
}

func decimalFromInteger(value int64) *referenceDecimal {
	decimal := referenceDecimal{Coefficient: jsontext.Value(fmt.Sprintf("%d", value))}
	return &decimal
}

func decimalFromUnsigned(value uint64) *referenceDecimal {
	decimal := referenceDecimal{Coefficient: jsontext.Value(fmt.Sprintf("%d", value))}
	return &decimal
}

func rawNumeric(validation *expr.ValidationExpr) referenceAliasNumeric {
	return referenceAliasNumeric{
		Minimum: decimalPointer(validation.Minimum), ExclusiveMinimum: decimalPointer(validation.ExclusiveMinimum),
		Maximum: decimalPointer(validation.Maximum), ExclusiveMaximum: decimalPointer(validation.ExclusiveMaximum),
	}
}

func decimalPointer(value *float64) *referenceDecimal {
	if value == nil {
		return nil
	}
	decimal := referenceBinaryDecimal(*value)
	return &decimal
}

func independentEffectiveBounds(attribute *expr.AttributeExpr) (expr.EffectiveNumericBound, expr.EffectiveNumericBound) {
	var lower, upper expr.EffectiveNumericBound
	for _, layer := range rawAliasAncestry(attribute) {
		validation := layer.attribute.Validation
		if validation == nil {
			continue
		}
		for _, bound := range []struct {
			value     *float64
			exclusive bool
			minimum   bool
		}{{validation.Minimum, false, true}, {validation.ExclusiveMinimum, true, true},
			{validation.Maximum, false, false}, {validation.ExclusiveMaximum, true, false}} {
			if bound.value == nil {
				continue
			}
			target := &upper
			if bound.minimum {
				target = &lower
			}
			if !target.Present || (bound.minimum && *bound.value > target.Value) ||
				(!bound.minimum && *bound.value < target.Value) ||
				(*bound.value == target.Value && bound.exclusive && !target.Exclusive) {
				*target = expr.EffectiveNumericBound{
					Value: *bound.value, Exclusive: bound.exclusive, Present: true,
					Provenance: expr.EffectiveConstraintSource{Declaration: layer.name},
				}
			}
		}
	}
	return lower, upper
}

func independentFieldID(attribute *expr.AttributeExpr, name string) uint64 {
	current := attribute
	for {
		named, ok := current.Type.(expr.UserType)
		if !ok {
			break
		}
		current = named.Attribute()
	}
	object := independentObject(current.Type)
	if object == nil {
		return 0
	}
	for index, field := range *object {
		if field.Name == name || independentAttributeName(field.Name) == independentAttributeName(name) {
			return uint64(index + 1)
		}
	}
	return 0
}

func independentObject(dataType expr.DataType) *expr.Object {
	for {
		switch actual := dataType.(type) {
		case expr.UserType:
			dataType = actual.Attribute().Type
		case *expr.Object:
			return actual
		default:
			return nil
		}
	}
}

func independentAttributeName(name string) string {
	attribute, _, _ := strings.Cut(name, ":")
	return attribute
}

func independentElementName(name string) string {
	_, element, present := strings.Cut(name, ":")
	if !present {
		return name
	}
	element, _, _ = strings.Cut(element, ":")
	return element
}

func layerIDByName(layers []referenceAliasLayer, name string) uint64 {
	for _, layer := range layers {
		if layer.name == name {
			return layer.Declaration
		}
	}
	return 0
}

func semanticIDs(values *[]referenceAliasValue) []uint64 {
	if values == nil {
		return []uint64{}
	}
	result := make([]uint64, len(*values))
	for index, value := range *values {
		result[index] = value.Semantic
	}
	return result
}

func requireEffectiveBoundEqual(t *testing.T, expected, actual expr.EffectiveNumericBound) {
	t.Helper()
	require.Equal(t, expected.Value, actual.Value)
	require.Equal(t, expected.Exclusive, actual.Exclusive)
	require.Equal(t, expected.Present, actual.Present)
	require.Equal(t, expected.Provenance.Declaration, actual.Provenance.Declaration)
}

func aliasNamed(name string, underlying expr.DataType, validation *expr.ValidationExpr) *expr.UserTypeExpr {
	return &expr.UserTypeExpr{TypeName: name, UID: name, AttributeExpr: &expr.AttributeExpr{
		Type: underlying, Validation: validation,
	}}
}
