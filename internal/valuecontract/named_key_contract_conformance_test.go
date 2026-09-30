package valuecontract

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"fmt"
	"maps"
	"math"
	"math/big"
	"reflect"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	loom "github.com/CaliLuke/loom/pkg"
)

func checkNamedKeyContractConformance(t *testing.T, executable string) {
	t.Helper()
	t.Run("raw layers preserve duplicates order and lengths", func(t *testing.T) {
		baseMinimum, baseMaximum := 1, 10
		derivedMinimum, derivedMaximum := 2, 9
		currentMinimum, currentMaximum := 3, 8
		base := aliasNamed("KeyBase", expr.String, &expr.ValidationExpr{
			Values:         []any{"aaa", "aaa"},
			Pattern:        `^a`,
			PatternClauses: []string{`^a`, `a$`},
			MinLength:      &baseMinimum,
			MaxLength:      &baseMaximum,
		})
		base.Attribute().DefaultValue = "aaa"
		middle := aliasNamed("KeyMiddle", base, nil)
		derived := aliasNamed("KeyDerived", middle, &expr.ValidationExpr{
			Values:         []any{"aaa"},
			Pattern:        `^a`,
			PatternClauses: []string{`^a`},
			MinLength:      &derivedMinimum,
			MaxLength:      &derivedMaximum,
		})
		key := &expr.AttributeExpr{Type: derived, Validation: &expr.ValidationExpr{
			MinLength: &currentMinimum,
			MaxLength: &currentMaximum,
		}}
		contract, err := captureIndependentKeyContract(t, key)
		require.NoError(t, err)
		require.Len(t, contract.layers, 4)
		require.Equal(t, []string{"KeyBase", "KeyMiddle", "KeyDerived", "attribute"}, []string{
			contract.layers[0].name,
			contract.layers[1].name,
			contract.layers[2].name,
			contract.layers[3].name,
		})
		require.Len(t, contract.layers[0].Predicates, 3)
		require.Len(t, contract.layers[2].Predicates, 2)
		require.Equal(t, []referencePredicateClause{
			{Kind: "pattern", Predicate: 1, Origin: 1},
			{Kind: "pattern", Predicate: 1, Origin: 1},
			{Kind: "pattern", Predicate: 2, Origin: 1},
		}, contract.layers[0].Predicates)
		require.Equal(t, []referencePredicateClause{
			{Kind: "pattern", Predicate: 1, Origin: 3},
			{Kind: "pattern", Predicate: 1, Origin: 3},
		}, contract.layers[2].Predicates)
		require.NotNil(t, contract.layers[0].Enumeration)
		require.Len(t, *contract.layers[0].Enumeration, 2,
			"duplicate raw key enum entries must remain present")
		require.Equal(t, (*contract.layers[0].Enumeration)[0].Semantic,
			(*contract.layers[0].Enumeration)[1].Semantic)
		require.NotNil(t, contract.layers[0].DefaultValue)
		require.Nil(t, contract.layers[1].DefaultValue)
		require.Equal(t, contract.layers[0].DefaultValue.Origin,
			runIndependentKeyContract(t, executable, contract).OK.DefaultValue.Origin)
		require.Equal(t, []byteAliasBound{
			{Minimum: &baseMinimum, Maximum: &baseMaximum},
			{},
			{Minimum: &derivedMinimum, Maximum: &derivedMaximum},
			{Minimum: &currentMinimum, Maximum: &currentMaximum},
		}, contract.lengths, "raw key length layers must retain absence, order, and repeated directions")
		result := runIndependentKeyContract(t, executable, contract)
		require.NotNil(t, result.OK)
		require.Len(t, result.OK.Predicates, 2, "exact duplicate key clauses must deduplicate stably")
		require.Equal(t, uint64(3), result.OK.Predicates[0].Origin)

		minimum, exclusiveMinimum, maximum, exclusiveMaximum := 0.0, 1.0, 10.0, 9.0
		numericBase := aliasNamed("RawNumericBase", expr.Float64, &expr.ValidationExpr{
			Minimum: &minimum, Maximum: &maximum,
		})
		numericDerived := aliasNamed("RawNumericDerived", numericBase, &expr.ValidationExpr{
			ExclusiveMinimum: &exclusiveMinimum, ExclusiveMaximum: &exclusiveMaximum,
		})
		numeric, err := captureIndependentKeyContract(t, &expr.AttributeExpr{Type: numericDerived})
		require.NoError(t, err)
		require.Equal(t, referenceAliasNumeric{
			Minimum: decimalPointer(&minimum), Maximum: decimalPointer(&maximum),
		}, numeric.layers[0].Numeric)
		require.Equal(t, referenceAliasNumeric{
			ExclusiveMinimum: decimalPointer(&exclusiveMinimum),
			ExclusiveMaximum: decimalPointer(&exclusiveMaximum),
		}, numeric.layers[1].Numeric)
	})

	t.Run("plain enum", func(t *testing.T) {
		key := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a"}}}
		checkKeyQueries(t, executable, key, []keyQueryCase{
			{name: "admitted", key: "a", accepted: true},
			{name: "outside", key: "b"},
		})
	})

	t.Run("named enum equality and subset", func(t *testing.T) {
		base := aliasNamed("NamedKeyBase", expr.String, &expr.ValidationExpr{Values: []any{"a", "b"}})
		equal := aliasNamed("NamedKeyEqual", base, &expr.ValidationExpr{Values: []any{"a", "b"}})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: equal}, []keyQueryCase{
			{name: "equal admitted", key: "a", accepted: true},
		})
		subset := aliasNamed("NamedKeySubset", base, &expr.ValidationExpr{Values: []any{"b"}})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: subset}, []keyQueryCase{
			{name: "subset admitted", key: "b", accepted: true},
			{name: "ancestor excluded", key: "a"},
		})
	})

	t.Run("multi hop enum refinement", func(t *testing.T) {
		base := aliasNamed("MultiKeyBase", expr.String, &expr.ValidationExpr{Values: []any{"a", "b", "c"}})
		middle := aliasNamed("MultiKeyMiddle", base, &expr.ValidationExpr{Values: []any{"b", "c"}})
		derived := aliasNamed("MultiKeyDerived", middle, &expr.ValidationExpr{Values: []any{"c"}})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: derived}, []keyQueryCase{
			{name: "last subset", key: "c", accepted: true},
			{name: "middle member excluded", key: "b"},
		})
	})

	t.Run("widening rejected by reference and whole map", func(t *testing.T) {
		base := aliasNamed("WideKeyBase", expr.String, &expr.ValidationExpr{Values: []any{"a", "b"}})
		derived := aliasNamed("WideKeyDerived", base, &expr.ValidationExpr{Values: []any{"b", "c"}})
		key := &expr.AttributeExpr{Type: derived}
		contract, err := captureIndependentKeyContract(t, key)
		require.NoError(t, err)
		result := runIndependentKeyContract(t, executable, contract)
		require.Nil(t, result.OK)
		require.Contains(t, string(result.Error), "enumWidening")
		raw, err := singletonMap("b", "value")
		require.NoError(t, err)
		_, productionErr := resolveWholeMap(t, key, raw)
		require.ErrorContains(t, productionErr, "enum member")
	})

	t.Run("defaults validate each key prefix", func(t *testing.T) {
		base := aliasNamed("DefaultKeyBase", expr.String, &expr.ValidationExpr{Values: []any{"a", "b"}})
		base.Attribute().DefaultValue = "a"
		valid := aliasNamed("DefaultKeyValid", base, &expr.ValidationExpr{Values: []any{"b"}})
		valid.Attribute().DefaultValue = "b"
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: valid}, []keyQueryCase{
			{name: "local replacement", key: "b", accepted: true},
		})

		invalid := aliasNamed("DefaultKeyInvalid", base, &expr.ValidationExpr{Values: []any{"b"}})
		contract, err := captureIndependentKeyContract(t, &expr.AttributeExpr{Type: invalid})
		require.NoError(t, err)
		result := runIndependentKeyContract(t, executable, contract)
		require.Nil(t, result.OK)
		require.Contains(t, string(result.Error), "invalidDefault")
		_, productionErr := resolveWholeMap(t, &expr.AttributeExpr{Type: invalid}, map[string]string{"b": "value"})
		require.ErrorContains(t, productionErr, "default")

		minimum := 2
		lengthBase := aliasNamed("LengthDefaultBase", expr.String, &expr.ValidationExpr{MinLength: &minimum})
		lengthBase.Attribute().DefaultValue = "x"
		lengthDerived := aliasNamed("LengthDefaultDerived", lengthBase, nil)
		lengthContract, err := captureIndependentKeyContract(t, &expr.AttributeExpr{Type: lengthDerived})
		require.NoError(t, err)
		require.Nil(t, runIndependentKeyContract(t, executable, lengthContract).OK)
		_, productionErr = resolveWholeMap(t, &expr.AttributeExpr{Type: lengthDerived}, map[string]string{"xx": "value"})
		require.ErrorContains(t, productionErr, "default")

		validBase := aliasNamed("LengthReplacementBase", expr.String, nil)
		validBase.Attribute().DefaultValue = "x"
		replacement := aliasNamed("LengthReplacementDerived", validBase, &expr.ValidationExpr{MinLength: &minimum})
		replacement.Attribute().DefaultValue = "xx"
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: replacement}, []keyQueryCase{
			{name: "length-valid replacement", key: "xx", accepted: true},
		})
	})

	t.Run("inherited predicates and duplicate clauses", func(t *testing.T) {
		const valid = "550e8400-e29b-41d4-a716-446655440000"
		const other = "123e4567-e89b-42d3-a456-426614174000"
		base := aliasNamed("PredicateKeyBase", expr.String, &expr.ValidationExpr{
			Values: []any{valid, other}, Pattern: `^[[:xdigit:]-]+$`, Format: expr.FormatUUID,
			PatternClauses: []string{`^[[:xdigit:]-]+$`},
			FormatClauses:  []expr.ValidationFormat{expr.FormatUUID},
		})
		derived := aliasNamed("PredicateKeyDerived", base, &expr.ValidationExpr{Pattern: `^550e`})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: derived}, []keyQueryCase{
			{name: "all predicates", key: valid, accepted: true},
			{name: "derived pattern", key: other},
		})

		formatOnly := aliasNamed("FormatOnlyKey", expr.String, &expr.ValidationExpr{Format: expr.FormatUUID})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: formatOnly}, []keyQueryCase{
			{name: "format valid", key: valid, accepted: true},
			{name: "format invalid independently", key: "not-a-uuid"},
		})
	})

	t.Run("four numeric bounds", func(t *testing.T) {
		minimum, maximum := 0.0, 10.0
		exclusiveMinimum, exclusiveMaximum := 1.0, 9.0
		base := aliasNamed("NumericKeyBase", expr.Float64, &expr.ValidationExpr{
			Minimum: &minimum,
			Maximum: &maximum,
		})
		derived := aliasNamed("NumericKeyDerived", base, &expr.ValidationExpr{
			ExclusiveMinimum: &exclusiveMinimum,
			ExclusiveMaximum: &exclusiveMaximum,
		})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: derived}, []keyQueryCase{
			{name: "closed minimum independently admitted", key: float64(0), accepted: false},
			{name: "inside", key: float64(2), accepted: true},
			{name: "exclusive lower", key: float64(1)},
			{name: "exclusive upper", key: float64(9)},
			{name: "outside", key: float64(10)},
		})

		closed := aliasNamed("ClosedNumericKey", expr.Float64, &expr.ValidationExpr{
			Minimum: &minimum, Maximum: &maximum,
		})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: closed}, []keyQueryCase{
			{name: "ordinary minimum", key: float64(0), accepted: true},
			{name: "ordinary maximum", key: float64(10), accepted: true},
			{name: "below ordinary minimum", key: float64(-1)},
			{name: "above ordinary maximum", key: float64(11)},
		})

		tie := 1.0
		openTieBase := aliasNamed("OpenTieBase", expr.Float64, &expr.ValidationExpr{Minimum: &tie})
		openTie := aliasNamed("OpenTieDerived", openTieBase, &expr.ValidationExpr{ExclusiveMinimum: &tie})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: openTie}, []keyQueryCase{
			{name: "open tie rejects endpoint", key: float64(1)},
			{name: "open tie admits above", key: float64(2), accepted: true},
		})

		emptyMinimum, emptyMaximum := 2.0, 1.0
		emptyBase := aliasNamed("EmptyNumericBase", expr.Float64, &expr.ValidationExpr{Minimum: &emptyMinimum})
		empty := aliasNamed("EmptyNumericDerived", emptyBase, &expr.ValidationExpr{Maximum: &emptyMaximum})
		emptyKey := &expr.AttributeExpr{Type: empty}
		emptyContract, err := captureIndependentKeyContract(t, emptyKey)
		require.NoError(t, err)
		require.NotNil(t, runIndependentKeyContract(t, executable, emptyContract).OK,
			"contradictory numeric bounds remain a valid empty contract")
		checkKeyQueries(t, executable, emptyKey, []keyQueryCase{
			{name: "empty domain lower", key: float64(1)},
			{name: "empty domain upper", key: float64(2)},
		})
	})

	t.Run("present empty key enum is an empty domain", func(t *testing.T) {
		key := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{}}}
		contract, err := captureIndependentKeyContract(t, key)
		require.NoError(t, err)
		require.NotNil(t, contract.layers[0].Enumeration)
		require.Empty(t, *contract.layers[0].Enumeration)
		reference := runIndependentKeyContract(t, executable, contract)
		require.NotNil(t, reference.OK)
		require.NotNil(t, reference.OK.Enumeration)
		require.Empty(t, *reference.OK.Enumeration)
		checkKeyQueries(t, executable, key, []keyQueryCase{{name: "no member admitted", key: "a"}})
	})

	t.Run("string length bounds", func(t *testing.T) {
		minimum, maximum := 2, 3
		base := aliasNamed("LengthKeyBase", expr.String, &expr.ValidationExpr{MinLength: &minimum})
		derived := aliasNamed("LengthKeyDerived", base, &expr.ValidationExpr{MaxLength: &maximum})
		checkKeyQueries(t, executable, &expr.AttributeExpr{Type: derived}, []keyQueryCase{
			{name: "rune count admitted", key: "éa", accepted: true},
			{name: "too short", key: "x"},
			{name: "too long", key: "abcd"},
		})

		invalidBase := aliasNamed("InvalidLengthEnumBase", expr.String, &expr.ValidationExpr{
			Values: []any{"x", "xx"}, MinLength: &minimum,
		})
		localRepair := aliasNamed("InvalidLengthEnumDerived", invalidBase, &expr.ValidationExpr{
			Values: []any{"xx"},
		})
		invalidKey := &expr.AttributeExpr{Type: localRepair}
		invalidContract, err := captureIndependentKeyContract(t, invalidKey)
		require.NoError(t, err)
		require.Nil(t, runIndependentKeyContract(t, executable, invalidContract).OK,
			"a later enum cannot repair an invalid ancestor length declaration")
		_, productionErr := resolveWholeMap(t, invalidKey, map[string]string{"xx": "value"})
		require.ErrorContains(t, productionErr, "enum member")
	})

	t.Run("typed integer and float keys", func(t *testing.T) {
		const exact = int64(9_007_199_254_740_993)
		integer := &expr.AttributeExpr{Type: expr.Int64, Validation: &expr.ValidationExpr{Values: []any{exact}}}
		checkKeyQueries(t, executable, integer, []keyQueryCase{
			{name: "signed", key: exact, accepted: true},
			{name: "unsigned same value", key: uint64(exact), accepted: true},
			{name: "next", key: uint64(exact + 1)},
		})
		float := &expr.AttributeExpr{Type: expr.Float32, Validation: &expr.ValidationExpr{Values: []any{float64(0.1)}}}
		checkKeyQueries(t, executable, float, []keyQueryCase{
			{name: "float32", key: float32(0.1), accepted: true},
			{name: "different", key: float32(0.2)},
		})

		integerWire := projectUnconstrainedKey(t, executable, expr.Int64, exact)
		integerName := onlyObjectName(t, integerWire)
		decodedInteger, ok := productionNumericPolicies()[1].key(integerName)
		require.True(t, ok)
		require.Equal(t, exact, decodedInteger)

		floatWire := projectUnconstrainedKey(t, executable, expr.Float32, float32(0.1))
		floatName := onlyObjectName(t, floatWire)
		decodedFloat, ok := productionNumericPolicies()[4].key(floatName)
		require.True(t, ok)
		require.Equal(t, float32(0.1), decodedFloat)
		require.NotEqual(t, any(float32(0.1)), any(float64(0.1)),
			"Any key source identity must retain distinct Go numeric host types")
	})

	t.Run("Any source identity differs from canonical name", func(t *testing.T) {
		key := &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{int(1)}}}
		contract, err := captureIndependentKeyContract(t, key)
		require.NoError(t, err)
		require.True(t, queryIndependentKey(t, executable, contract, int(1)).contractOK)
		require.False(t, queryIndependentKey(t, executable, contract, "1").contractOK)

		integerWire := projectConstrainedKey(t, executable, key, int(1))
		stringWire := projectUnconstrainedKey(t, executable, expr.Any, "1")
		require.Equal(t, integerWire, stringWire,
			"equal wire names cannot replace raw Any key-domain equality")
		var decodedString map[string]string
		require.NoError(t, json.Unmarshal([]byte(integerWire), &decodedString))
		require.Equal(t, map[string]string{"1": "value"}, decodedString)
		var decodedInteger map[int]string
		require.NoError(t, json.Unmarshal([]byte(integerWire), &decodedInteger, loom.JSONOptions()))
		require.Equal(t, map[int]string{1: "value"}, decodedInteger)

		jsonSnapshot := &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{Values: []any{"1"}}}
		snapshotContract, err := captureIndependentKeyContract(t, jsonSnapshot)
		require.NoError(t, err)
		require.False(t, queryIndependentKey(t, executable, snapshotContract, int(1)).contractOK)
		require.True(t, queryIndependentKey(t, executable, snapshotContract, "1").contractOK,
			"negative control: ordinary Any JSON snapshots substitute a different source domain")

		numericAny := &expr.AttributeExpr{Type: expr.Any, Validation: &expr.ValidationExpr{
			Values: []any{float32(0.1)},
		}}
		numericContract, err := captureIndependentKeyContract(t, numericAny)
		require.NoError(t, err)
		require.True(t, queryIndependentKey(t, executable, numericContract, float32(0.1)).contractOK)
		require.False(t, queryIndependentKey(t, executable, numericContract, float64(0.1)).contractOK)
		numericWire := projectConstrainedKey(t, executable, numericAny, float32(0.1))
		var decodedFloat map[float32]string
		require.NoError(t, json.Unmarshal([]byte(numericWire), &decodedFloat, loom.JSONOptions()))
		require.Equal(t, map[float32]string{float32(0.1): "value"}, decodedFloat)
	})

	t.Run("dropping a raw key layer changes the answer", func(t *testing.T) {
		base := aliasNamed("DroppedKeyBase", expr.String, &expr.ValidationExpr{Values: []any{"a"}})
		derived := aliasNamed("DroppedKeyDerived", base, nil)
		contract, err := captureIndependentKeyContract(t, &expr.AttributeExpr{Type: derived})
		require.NoError(t, err)
		require.False(t, queryIndependentKey(t, executable, contract, "b").contractOK)
		dropped := contract
		dropped.layers = slices.Clone(contract.layers[1:])
		require.True(t, queryIndependentKey(t, executable, dropped, "b").contractOK,
			"negative control must detect a lost ancestor key enum")
	})

	t.Run("malformed ancestry and declarations reject independently", func(t *testing.T) {
		cycle := aliasNamed("CyclicKey", expr.String, nil)
		cycle.Attribute().Type = cycle
		_, err := captureIndependentKeyContract(t, &expr.AttributeExpr{Type: cycle})
		require.ErrorContains(t, err, "cyclic named key ancestry")
		missing := &expr.UserTypeExpr{TypeName: "MissingKey"}
		_, err = captureIndependentKeyContract(t, &expr.AttributeExpr{Type: missing})
		require.ErrorContains(t, err, "has no declaration")

		invalid := &expr.AttributeExpr{Type: expr.Int, Validation: &expr.ValidationExpr{Values: []any{int64(1)}}}
		_, err = captureIndependentKeyContract(t, invalid)
		require.ErrorContains(t, err, "enum")
		raw, rawErr := singletonMap(int64(1), "value")
		require.NoError(t, rawErr)
		_, productionErr := resolveWholeMap(t, invalid, raw)
		require.ErrorContains(t, productionErr, "effective contract")
	})

	t.Run("value sibling does not become a key oracle", func(t *testing.T) {
		key := &expr.AttributeExpr{Type: expr.String, Validation: &expr.ValidationExpr{Values: []any{"a"}}}
		contract, err := captureIndependentKeyContract(t, key)
		require.NoError(t, err)
		query := queryIndependentKey(t, executable, contract, "a")
		require.True(t, query.contractOK && query.lengthOK)
		valid, err := resolveWholeMap(t, key, map[string]string{"a": "value"})
		require.NoError(t, err)
		require.Equal(t, expr.ValueResolved, valid)
		invalid, err := resolveWholeMap(t, key, map[string]any{"a": 1})
		require.NoError(t, err)
		require.Equal(t, expr.ValueInvalid, invalid,
			"an invalid value sibling must not rewrite independent key acceptance")
	})
}

type keyQueryCase struct {
	name     string
	key      any
	accepted bool
}

func checkKeyQueries(
	t *testing.T,
	executable string,
	key *expr.AttributeExpr,
	cases []keyQueryCase,
) {
	t.Helper()
	contract, err := captureIndependentKeyContract(t, key)
	require.NoError(t, err)
	reference := runIndependentKeyContract(t, executable, contract)
	require.NotNil(t, reference.OK, "raw key contract rejected: %s", reference.Error)
	lengthReference := runIndependentLengths(t, executable, contract.lengths, nil)
	production, err := expr.EffectiveConstraintsFor(key)
	require.NoError(t, err)
	lowered := production.Validation().Lowered()
	require.Equal(t, lengthReference.Effective.Minimum, lowered.MinLength)
	require.Equal(t, lengthReference.Effective.Maximum, lowered.MaxLength)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			query := queryIndependentKey(t, executable, contract, tc.key)
			require.Equal(t, tc.accepted, query.contractOK && query.lengthOK,
				"synthetic key query is independent adapter evidence")
			raw, err := singletonMap(tc.key, "value")
			require.NoError(t, err)
			outcome, err := resolveWholeMap(t, key, raw)
			require.NoError(t, err)
			require.Equal(t, tc.accepted, outcome == expr.ValueResolved)
		})
	}
}

func projectUnconstrainedKey(
	t *testing.T,
	executable string,
	keyType expr.DataType,
	key any,
) string {
	t.Helper()
	raw := map[any]string{key: "value"}
	attribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  &expr.AttributeExpr{Type: keyType},
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}
	tc := productionProjectionCase{
		name: "Any canonical key", attribute: attribute, raw: raw,
	}
	var actual, expected productionProjected
	usedNumericTargets := false
	if primitive, ok := keyType.(expr.Primitive); ok {
		policyIndex := map[expr.Primitive]int{
			expr.Int32: 0, expr.Int64: 1, expr.UInt32: 2, expr.UInt64: 3,
			expr.Float32: 4, expr.Float64: 5,
		}
		if index, found := policyIndex[primitive]; found {
			actual, expected = productionProjectPairWithNumericTargets(
				t, executable, tc, []productionNumericPolicy{productionNumericPolicies()[index]},
			)
			usedNumericTargets = true
		}
	}
	if !usedNumericTargets {
		actual, expected = productionProjectPair(t, executable, tc, expr.ValuePlanRuntime)
	}
	require.Equal(t, expected, actual)
	require.Equal(t, "emitted", actual.outcome)
	return actual.wire
}

func projectConstrainedKey(
	t *testing.T,
	executable string,
	key *expr.AttributeExpr,
	rawKey any,
) string {
	t.Helper()
	contract, err := captureIndependentKeyContract(t, key)
	require.NoError(t, err)
	query := queryIndependentKey(t, executable, contract, rawKey)
	require.True(t, query.contractOK && query.lengthOK,
		"raw key-domain reference must admit the constrained projection input")
	attribute := &expr.AttributeExpr{Type: &expr.Map{
		KeyType:  key,
		ElemType: &expr.AttributeExpr{Type: expr.String},
	}}
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(attribute)
	require.NoError(t, err)
	resolved := context.Resolve(
		occurrence,
		context.SupplyValue(expr.ValueInput{Raw: map[any]string{rawKey: "value"}}),
		expr.ValueRoleExample,
	)
	require.Equal(t, expr.ValueResolved, resolved.Outcome())
	plan, err := context.NewValuePlan(occurrence, expr.ValuePlanRequest{
		Target: attribute, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanRuntime,
	})
	require.NoError(t, err)
	actual := productionProjectionOutput(t, context.ProjectJSON(resolved, plan))
	require.Equal(t, "emitted", actual.outcome)
	expected := projectUnconstrainedKey(t, executable, contract.extractor.attribute.Type, rawKey)
	require.Equal(t, expected, actual.wire,
		"independent key-domain admission and key encoding jointly define the constrained wire")
	return actual.wire
}

// productionProjectPairWithNumericTargets uses the existing audited target
// codec table to close the numeric key decoder requests exposed by the
// reference. It does not invent a projection result.
func productionProjectPairWithNumericTargets(
	t *testing.T,
	executable string,
	tc productionProjectionCase,
	policies []productionNumericPolicy,
) (productionProjected, productionProjected) {
	t.Helper()
	context := expr.NewValueContext()
	occurrence, err := context.NewOccurrence(tc.attribute)
	require.NoError(t, err)
	input := newProductionInput(t)
	graph := newProductionGraph(t, input)
	root := graph.capture(tc.attribute, occurrence)
	graph.ranks()
	targets := productionProjectionTargets(t, graph, tc)
	applyNumericMapKeyTargetPolicy(t, targets, policies)
	targets, targetRoot := productionProjectionRoot(t, graph, occurrence, targets, tc.selected)
	supplied := input.encode(tc.raw)
	codecs := input.codecs()
	request := map[string]any{
		"declarations": graph.declarations,
		"root":         root,
		"supplied": map[string]any{
			"source": map[string]any{"occurrence": root, "origin": 1, "role": "authoredExample"},
			"value":  supplied,
		},
		"codecs": codecs,
		"checks": []any{},
		"projection": map[string]any{
			"root": targetRoot, "use": "runtime", "targets": targets,
		},
	}
	command := referenceConstructor("evaluate", map[string]any{"request": request})
	actual := referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	knownTexts := []string{}
	for iteration := 0; iteration < 8 && actual["projected"] == nil; iteration++ {
		changed := false
		if requests, present := actual["decimalRequests"]; present {
			changed = mergeRequestedDecimalSpellings(
				t, codecs, requests, numericNativeKeyCandidates(t, tc.raw, knownTexts, policies),
			) || changed
		}
		texts := requestedNumericTexts(t, actual)
		if len(texts) > 0 {
			knownTexts = appendUniqueStrings(knownTexts, texts...)
			for name, rows := range productionTargetCodecs(t, knownTexts, policies) {
				if name == "literalSpellings" || name == "literalReadings" {
					continue
				}
				changed = mergeNumericCodecRows(t, codecs, name, rows.([]any)) || changed
			}
		}
		require.True(t, changed, "numeric codec request made no progress: %s", productionJSON(t, actual))
		actual = referenceDecode[map[string]jsontext.Value](t, runReference(t, executable, []any{command})[0])
	}
	require.Contains(t, actual, "projected", "numeric codec table must discharge all requests: %s", productionJSON(t, actual))

	resolved := context.Resolve(occurrence, context.SupplyValue(expr.ValueInput{Raw: tc.raw}), expr.ValueRoleExample)
	plan, err := context.NewValuePlan(occurrence, productionProjectionPlan(tc, expr.ValuePlanRuntime))
	require.NoError(t, err)
	output := productionProjectionOutput(t, context.ProjectJSON(resolved, plan))
	emitted := referenceDecode[map[string]map[string]jsontext.Value](t, actual["projected"])
	require.Contains(t, emitted, "emitted")
	expected := productionProjected{
		outcome: "emitted",
		wire:    productionCanonical(t, productionReferenceWire(t, emitted["emitted"]["value"])),
	}
	return output, expected
}

// applyNumericMapKeyTargetPolicy completes the projection adapter's target
// rules from the independently authored concrete key primitive. The generic
// production graph intentionally uses conservative mathematical target rules;
// map-key projection has the concrete emitted native type needed here.
func applyNumericMapKeyTargetPolicy(
	t *testing.T,
	targets []any,
	policies []productionNumericPolicy,
) {
	t.Helper()
	require.Len(t, policies, 1, "a concrete map key has exactly one numeric policy")
	policy := policies[0]
	updated := false
	for _, raw := range targets {
		declaration := raw.(map[string]any)
		constructor, ok := declaration["target"].(map[string]any)
		if !ok {
			continue
		}
		rawMap, ok := constructor["map"]
		if !ok {
			continue
		}
		fields := rawMap.(map[string]any)
		keyRules := maps.Clone(fields["keyRules"].(map[string]any))
		keyRules["numericFormat"] = policy.decimalFormat()
		keyRules["integerFormat"] = policy.integerFormat()
		fields["keyRules"] = keyRules
		updated = true
	}
	require.True(t, updated, "numeric projection target must contain a map")
}

func requestedNumericTexts(t *testing.T, result map[string]jsontext.Value) []string {
	t.Helper()
	seen := make(map[string]bool)
	var texts []string
	for _, field := range []string{"numericRequests", "integerReadingRequests", "decimalReadingRequests"} {
		raw, present := result[field]
		if !present {
			continue
		}
		if field == "numericRequests" {
			for _, text := range referenceDecode[[]string](t, raw) {
				if !seen[text] {
					seen[text], texts = true, append(texts, text)
				}
			}
			continue
		}
		for _, row := range referenceDecode[[][]jsontext.Value](t, raw) {
			require.Len(t, row, 2)
			text := referenceDecode[string](t, row[1])
			if !seen[text] {
				seen[text], texts = true, append(texts, text)
			}
		}
	}
	return texts
}

type numericIdentityObservation struct {
	Value        referenceDecimal `json:"value"`
	Format       jsontext.Value   `json:"format"`
	NegativeZero bool             `json:"negativeZero"`
}

func numericNativeKeyCandidates(
	t *testing.T,
	raw any,
	texts []string,
	policies []productionNumericPolicy,
) map[string]string {
	t.Helper()
	candidates := make(map[string]string)
	add := func(value any) {
		rawValue := reflect.ValueOf(value)
		if !rawValue.IsValid() {
			return
		}
		switch rawValue.Kind() {
		case reflect.Int64, reflect.Float32, reflect.Float64:
			_, identity := referenceNumericScalar(value, 0)
			wire, err := json.Marshal(value)
			require.NoError(t, err)
			candidates[numericIdentityKey(t, identity)] = string(wire)
		}
	}
	rawMap := reflect.ValueOf(raw)
	if rawMap.IsValid() && rawMap.Kind() == reflect.Map {
		for _, rawKey := range rawMap.MapKeys() {
			for rawKey.Kind() == reflect.Interface {
				rawKey = rawKey.Elem()
			}
			add(rawKey.Interface())
		}
	}
	for _, text := range texts {
		for _, policy := range policies {
			if key, accepted := policy.key(text); accepted {
				add(key)
			}
		}
	}
	return candidates
}

func mergeRequestedDecimalSpellings(
	t *testing.T,
	codecs map[string]any,
	rawRequests jsontext.Value,
	candidates map[string]string,
) bool {
	t.Helper()
	changed := false
	for _, request := range referenceDecode[[]jsontext.Value](t, rawRequests) {
		wire, found := candidates[numericIdentityKey(t, request)]
		require.True(t, found,
			"decimal spelling request must match an actual source or decoded native key: %s", request)
		changed = mergeNumericCodecRows(t, codecs, "decimalSpellings", []any{
			map[string]any{"number": request, "text": wire},
		}) || changed
	}
	return changed
}

func mergeNumericCodecRows(
	t *testing.T,
	codecs map[string]any,
	name string,
	rows []any,
) bool {
	t.Helper()
	existing := codecs[name].([]any)
	byKey := make(map[string]string, len(existing))
	for _, row := range existing {
		byKey[numericCodecRowKey(t, name, row)] = exactJSON(t, row)
	}
	changed := false
	for _, row := range rows {
		key := numericCodecRowKey(t, name, row)
		encoded := exactJSON(t, row)
		if previous, present := byKey[key]; present {
			require.Equal(t, previous, encoded, "conflicting %s row for %s", name, key)
			continue
		}
		byKey[key] = encoded
		existing = append(existing, row)
		changed = true
	}
	codecs[name] = existing
	return changed
}

func numericCodecRowKey(t *testing.T, name string, row any) string {
	t.Helper()
	raw := referenceDecode[map[string]jsontext.Value](t, jsontext.Value(exactJSON(t, row)))
	switch name {
	case "numberReadings":
		return referenceDecode[string](t, raw["text"])
	case "integerReadings", "decimalReadings":
		return exactJSON(t, raw["format"]) + "|" + referenceDecode[string](t, raw["text"])
	case "decimalSpellings":
		return numericIdentityKey(t, raw["number"])
	default:
		t.Fatalf("unsupported numeric codec table %q", name)
		return ""
	}
}

func numericIdentityKey(t *testing.T, raw any) string {
	t.Helper()
	encoded := jsontext.Value(exactJSON(t, raw))
	identity := referenceDecode[numericIdentityObservation](t, encoded)
	coefficient, ok := new(big.Int).SetString(string(identity.Value.Coefficient), 10)
	require.True(t, ok, "invalid exact coefficient %s", identity.Value.Coefficient)
	power := new(big.Int).Exp(big.NewInt(10), big.NewInt(absInt64(identity.Value.Exponent)), nil)
	value := new(big.Rat).SetInt(coefficient)
	if identity.Value.Exponent >= 0 {
		value.Mul(value, new(big.Rat).SetInt(power))
	} else {
		value.Quo(value, new(big.Rat).SetInt(power))
	}
	return fmt.Sprintf("%s|%t|%s", identity.Format, identity.NegativeZero, value.RatString())
}

func absInt64(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}

func exactJSON(t *testing.T, value any) string {
	t.Helper()
	encoded, err := json.Marshal(value, json.Deterministic(true))
	require.NoError(t, err)
	return string(encoded)
}

func appendUniqueStrings(values []string, additions ...string) []string {
	for _, addition := range additions {
		if !slices.Contains(values, addition) {
			values = append(values, addition)
		}
	}
	return values
}

func onlyObjectName(t *testing.T, wire string) string {
	t.Helper()
	var object map[string]any
	require.NoError(t, json.Unmarshal([]byte(wire), &object))
	require.Len(t, object, 1)
	for name := range object {
		return name
	}
	return ""
}

func checkEffectiveAliasLengthConformance(t *testing.T, executable string) {
	t.Helper()
	minimum, maximum := 1, 2
	contradictoryMinimum, contradictoryMaximum := 3, 2
	tests := []struct {
		name    string
		typ     expr.DataType
		minimum *int
		maximum *int
		values  []any
		lengths []int
	}{
		{
			name: "string rune count", typ: expr.String, minimum: &minimum, maximum: &maximum,
			values: []any{"", "é", "éa", "abc"}, lengths: []int{0, 1, 2, 3},
		},
		{
			name: "array nil empty nonempty", typ: &expr.Array{ElemType: &expr.AttributeExpr{Type: expr.String}},
			minimum: &minimum, maximum: &maximum,
			values:  []any{[]string(nil), []string{}, []string{"a"}, []string{"a", "b"}, []string{"a", "b", "c"}},
			lengths: []int{0, 0, 1, 2, 3},
		},
		{
			name: "map nil empty nonempty", typ: &expr.Map{
				KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String},
			},
			minimum: &minimum, maximum: &maximum,
			values:  []any{map[string]string(nil), map[string]string{}, map[string]string{"a": "x"}, map[string]string{"a": "x", "b": "y"}, map[string]string{"a": "x", "b": "y", "c": "z"}},
			lengths: []int{0, 0, 1, 2, 3},
		},
		{
			name: "contradictory", typ: expr.String,
			minimum: &contradictoryMinimum, maximum: &contradictoryMaximum,
			values: []any{"", "a", "ab", "abc"}, lengths: []int{0, 1, 2, 3},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			looseMinimum, looseMaximum := 0, 100
			base := aliasNamed("LengthBase", tc.typ, &expr.ValidationExpr{
				MinLength: &looseMinimum, MaxLength: &looseMaximum,
			})
			middle := aliasNamed("LengthMiddle", base, nil)
			derived := aliasNamed("LengthDerived", middle, &expr.ValidationExpr{MinLength: tc.minimum})
			attribute := &expr.AttributeExpr{Type: derived, Validation: &expr.ValidationExpr{
				MaxLength: tc.maximum,
			}}
			chain, err := checkedRawAliasAncestry(attribute)
			require.NoError(t, err)
			slices.Reverse(chain)
			bounds := make([]byteAliasBound, len(chain))
			for index, layer := range chain {
				if validation := layer.attribute.Validation; validation != nil {
					bounds[index] = byteAliasBound{
						Minimum: copyIntPointer(validation.MinLength),
						Maximum: copyIntPointer(validation.MaxLength),
					}
				}
			}
			require.Equal(t, []byteAliasBound{
				{Minimum: &looseMinimum, Maximum: &looseMaximum},
				{},
				{Minimum: tc.minimum},
				{Maximum: tc.maximum},
			}, bounds, "raw length layers must retain absent and repeated tighter directions")
			reference := runIndependentLengths(t, executable, bounds, tc.lengths)
			constraints, err := expr.EffectiveConstraintsFor(attribute)
			require.NoError(t, err)
			lowered := constraints.Validation().Lowered()
			require.Equal(t, reference.Effective.Minimum, lowered.MinLength)
			require.Equal(t, reference.Effective.Maximum, lowered.MaxLength)

			context := expr.NewValueContext()
			occurrence, err := context.NewOccurrence(attribute)
			require.NoError(t, err)
			projectionSource := expr.DupAtt(attribute)
			for current := projectionSource; ; {
				current.Validation = nil
				if named, ok := current.Type.(expr.UserType); ok {
					current = named.Attribute()
					continue
				}
				break
			}
			projectionTarget := expr.DupAtt(projectionSource)
			current := projectionTarget
			for index := len(bounds) - 1; index >= 0; index-- {
				bound := bounds[index]
				current.Validation = &expr.ValidationExpr{
					MinLength: bound.Minimum, MaxLength: bound.Maximum,
				}
				if named, ok := current.Type.(expr.UserType); ok {
					current = named.Attribute()
				}
			}
			sourceOccurrence, err := context.NewOccurrence(projectionSource)
			require.NoError(t, err)
			plan, err := context.NewValuePlan(sourceOccurrence, expr.ValuePlanRequest{
				Target: projectionTarget, Codec: expr.ValueCodecJSON, Use: expr.ValuePlanRuntime,
			})
			require.NoError(t, err)
			for index, raw := range tc.values {
				accepted := reference.Accepted[index]
				resolved := context.Resolve(
					occurrence,
					context.SupplyValue(expr.ValueInput{Raw: raw}),
					expr.ValueRoleExample,
				)
				require.Equal(t, accepted, resolved.Outcome() == expr.ValueResolved, "value %d", index)
				unconstrained := context.Resolve(
					sourceOccurrence,
					context.SupplyValue(expr.ValueInput{Raw: raw}),
					expr.ValueRoleExample,
				)
				require.Equal(t, expr.ValueResolved, unconstrained.Outcome())
				projected := context.ProjectJSON(unconstrained, plan)
				require.Equal(t, accepted, projected.Outcome() == expr.ProjectionEmitted, "projection %d", index)
			}
		})
	}
}

func TestIndependentKeyContractNegativeControls(t *testing.T) {
	const exact = int64(9_007_199_254_740_993)
	require.NotEqual(t, exact, int64(float64(exact)),
		"ordinary Float64 snapshots must not preserve exact large key identity")
	require.NotEqual(t, any(float32(0.1)), any(float64(0.1)),
		"ordinary Any equality must not collapse distinct numeric host types")
	require.NotEqual(t, math.Float64bits(0.1), math.Float64bits(float64(float32(0.1))),
		"ordinary JSON-number equality must not erase Float32 source precision")
}
