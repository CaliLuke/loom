package exactnum

import (
	"encoding/json/v2"
	"math"
	"math/big"
	"math/rand/v2"
	"testing"

	"github.com/stretchr/testify/require"
	"go.yaml.in/yaml/v3"
)

func TestSourceAdmission(t *testing.T) {
	for _, source := range []string{"0.1", "1e-1000001", "0x1p-1000001", "0x1.4p0", "+.5", "1.", "1_000"} {
		_, err := ParseDSL(source)
		require.NoError(t, err, source)
	}
	for _, source := range []string{"1e309", "NaN", "Inf", "1/2", " 1", "1 "} {
		_, err := ParseDSL(source)
		require.Error(t, err, source)
	}
	for _, source := range []string{"1e1000001", "1e-1000001", "-0", "1.25"} {
		_, err := ParseJSON(source)
		require.NoError(t, err, source)
	}
	for _, source := range []string{"+1", "0x1p0", "1/2", "1_000", ".5", "1.", "01", "NaN", `"42"`} {
		_, err := ParseJSON(source)
		require.Error(t, err, source)
	}
}

func TestExactIdentityAndOwnership(t *testing.T) {
	for _, test := range []struct {
		left, right string
		order       int
	}{
		{"0x1.4p0", "1.25", 0},
		{"-0x1p-2", "-.25", 0},
		{"-0", "0", 0},
		{"9007199254740993", "9007199254740992", 1},
		{"1e-1000001", "0", 1},
		{"0x1p-1000000000000000000000000000000", "1e-1000000000000000000000000000000", 1},
		{"10e-1000000000000000000000000000001", "1e-1000000000000000000000000000000", 0},
	} {
		left, err := ParseDSL(test.left)
		require.NoError(t, err)
		right, err := ParseDSL(test.right)
		require.NoError(t, err)
		leftKey, rightKey := left.Key(), right.Key()
		require.Equal(t, test.order, left.Cmp(right))
		require.Equal(t, -test.order, right.Cmp(left))
		if test.order == 0 {
			require.Equal(t, leftKey, rightKey)
		}
		require.Equal(t, leftKey, left.Key(), "comparison must not mutate shared storage")
		require.Equal(t, rightKey, right.Key())
	}
	var zero Number
	require.Equal(t, 0, zero.Sign())
	require.True(t, zero.IsInt())
	require.Equal(t, "0", zero.String())
}

func TestFloatMeaningAndEmission(t *testing.T) {
	decimal, err := ParseJSON("0.1")
	require.NoError(t, err)
	f64, err := FromFloat64(0.1)
	require.NoError(t, err)
	f32, err := FromFloat64(float64(float32(0.1)))
	require.NoError(t, err)
	require.Positive(t, f64.Cmp(decimal))
	require.Positive(t, f32.Cmp(f64))
	encoded, err := json.Marshal(f64)
	require.NoError(t, err)
	oracle, ok := new(big.Rat).SetString(string(encoded))
	require.True(t, ok)
	require.Zero(t, oracle.Cmp(new(big.Rat).SetFloat64(0.1)))
	for _, value := range []float64{math.NaN(), math.Inf(1), math.Inf(-1)} {
		_, err := FromFloat64(value)
		require.Error(t, err)
	}
	for _, value := range []uint64{0, 1, 9007199254740993, math.MaxUint64} {
		n := FromUint64(value)
		round, err := ParseJSON(n.String())
		require.NoError(t, err)
		require.Zero(t, n.Cmp(round))
	}
}

func TestIntegrality(t *testing.T) {
	for _, test := range []struct {
		source string
		want   bool
	}{
		{"0e-1000001", true}, {"1e1000001", true}, {"1e-1000001", false},
		{"1.25", false}, {"100e-2", true}, {"5e-1", false},
	} {
		n, err := ParseJSON(test.source)
		require.NoError(t, err)
		require.Equal(t, test.want, n.IsInt(), test.source)
	}
}

func TestComparisonRationalOracle(t *testing.T) {
	random := rand.New(rand.NewPCG(681, 755))
	for range 10000 {
		left := canonical(big.NewInt(int64(random.IntN(200001)-100000)), big.NewInt(int64(random.IntN(801)-400)), big.NewInt(int64(random.IntN(801)-400)))
		right := canonical(big.NewInt(int64(random.IntN(200001)-100000)), big.NewInt(int64(random.IntN(801)-400)), big.NewInt(int64(random.IntN(801)-400)))
		l, ok := new(big.Rat).SetString(left.String())
		require.True(t, ok)
		r, ok := new(big.Rat).SetString(right.String())
		require.True(t, ok)
		require.Equal(t, l.Cmp(r), left.Cmp(right))
	}
	coefficient := new(big.Int).Exp(big.NewInt(10), big.NewInt(1000), nil)
	coefficient.Add(coefficient, big.NewInt(1))
	left := canonical(coefficient, new(big.Int), new(big.Int))
	right, err := ParseJSON("1e1000")
	require.NoError(t, err)
	coefficient.SetInt64(0)
	require.Positive(t, left.Cmp(right), "construction owns input integers; close values require refinement")
}

func TestYAMLRetainsNumericScalar(t *testing.T) {
	for _, source := range []string{"9007199254740993", "18446744073709551615", "1e-1000001", "1.25"} {
		number, err := ParseJSON(source)
		require.NoError(t, err)
		wire, err := yaml.Marshal(map[string]Number{"minimum": number})
		require.NoError(t, err)
		var document yaml.Node
		require.NoError(t, yaml.Unmarshal(wire, &document))
		scalar := document.Content[0].Content[1]
		require.Contains(t, []string{"!!int", "!!float"}, scalar.Tag)
		require.Equal(t, number.String(), scalar.Value)
	}
}
