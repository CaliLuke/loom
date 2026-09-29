package scripts_test

import (
	"encoding/json/v2"
	"testing"

	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

// TestValueContractNumericWireCollision preserves the actual codec boundary that
// invalidated the candidate model's separate JSON integer/decimal constructors.
// It is decoder evidence, not a proof of arbitrary numerical serialization.
func TestValueContractNumericWireCollision(t *testing.T) {
	integerWire, err := json.Marshal(struct {
		N int `json:"n"`
	}{N: 1})
	require.NoError(t, err)
	decimalWire, err := json.Marshal(struct {
		N float64 `json:"n"`
	}{N: 1})
	require.NoError(t, err)
	require.Equal(t, []byte(`{"n":1}`), integerWire)
	require.Equal(t, integerWire, decimalWire)

	for _, tc := range []struct {
		name         string
		wire         string
		integerValid bool
	}{
		{name: "shared canonical number", wire: string(decimalWire), integerValid: true},
		{name: "decimal lexical alias", wire: `{"n":1.0}`, integerValid: false},
		{name: "exponent lexical alias", wire: `{"n":1e0}`, integerValid: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var integerTarget struct {
				N int `json:"n"`
			}
			integerErr := json.Unmarshal([]byte(tc.wire), &integerTarget)
			if tc.integerValid {
				require.NoError(t, integerErr)
				require.Equal(t, 1, integerTarget.N)
			} else {
				require.Error(t, integerErr)
			}
			var decimalTarget struct {
				N float64 `json:"n"`
			}
			require.NoError(t, json.Unmarshal([]byte(tc.wire), &decimalTarget))
			require.Equal(t, float64(1), decimalTarget.N)
		})
	}
}

// TestValueContractMapKeyDecoding checks actual converted-key duplicate detection.
// The model must not infer decoder acceptance from canonical encoder membership.
func TestValueContractMapKeyDecoding(t *testing.T) {
	for _, tc := range []struct {
		name    string
		wire    string
		decimal bool
		valid   bool
	}{
		{name: "integer canonical", wire: `{"1":"value"}`, valid: true},
		{name: "integer signed zero", wire: `{"-0":"value"}`, valid: true},
		{name: "integer leading zero", wire: `{"01":"value"}`},
		{name: "integer plus sign", wire: `{"+1":"value"}`},
		{name: "integer duplicate decoded zero", wire: `{"0":"first","-0":"second"}`},
		{name: "decimal lexical alias", wire: `{"1.0":"value"}`, decimal: true, valid: true},
		{name: "decimal exponent alias", wire: `{"1e0":"value"}`, decimal: true, valid: true},
		{name: "decimal duplicate alias", wire: `{"1":"first","1.0":"second"}`, decimal: true},
		{name: "decimal duplicate exponent", wire: `{"1":"first","1e0":"second"}`, decimal: true},
		{name: "decimal duplicate zero", wire: `{"0":"first","-0":"second"}`, decimal: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var err error
			if tc.decimal {
				var value map[float64]string
				err = json.Unmarshal([]byte(tc.wire), &value, loom.JSONOptions())
			} else {
				var value map[int]string
				err = json.Unmarshal([]byte(tc.wire), &value, loom.JSONOptions())
			}
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
