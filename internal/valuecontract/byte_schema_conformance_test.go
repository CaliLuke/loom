package valuecontract

import (
	"encoding/json/jsontext"
	"fmt"
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/internal/byteschema"
)

// checkByteSchemaConformance compares the real shared helper with the proved
// clamping/omission arithmetic. Regex execution and decoder language are checked
// separately by actual generated HTTP/JSON-RPC requests and the Ajv schema gate.
func checkByteSchemaConformance(t *testing.T, executable string) {
	t.Helper()
	limits := []*int{nil}
	for _, value := range []int{-3, -1, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 12, 100, int(^uint(0) >> 1)} {
		copy := value
		limits = append(limits, &copy)
	}
	if ^uint(0)>>63 != 0 {
		for _, value := range []int64{1<<53 - 4, 1<<53 - 1, 1 << 53} {
			copy := int(value)
			limits = append(limits, &copy)
		}
	}
	cases := make([]byteAliasBound, 0, len(limits)*len(limits))
	commands := make([]any, 0, len(limits)*len(limits))
	for _, minimum := range limits {
		for _, maximum := range limits {
			bounds := byteAliasBound{Minimum: minimum, Maximum: maximum}
			cases = append(cases, bounds)
			commands = append(commands, referenceConstructor("byteLengthSchema", map[string]any{"bounds": bounds}))
		}
	}
	results := runReference(t, executable, commands)
	limit := new(big.Int).SetUint64(min(uint64(^uint(0)>>1), uint64(1<<53-1)))
	patterns := []string{
		"^([A-Za-z0-9+/]{4})*$",
		"^([A-Za-z0-9+/]{4})*[A-Za-z0-9+/]{2}==$",
		"^([A-Za-z0-9+/]{4})*[A-Za-z0-9+/]{3}=$",
	}
	for index, bounds := range cases {
		t.Run(fmt.Sprintf("bounds_%d", index), func(t *testing.T) {
			projected := referenceDecode[[]jsontext.Value](t, results[index])
			require.Len(t, projected, 3)
			var expected []byteschema.Branch
			overflow := false
			for residue, raw := range projected {
				if string(raw) == "null" {
					continue
				}
				branch := referenceDecode[map[string]jsontext.Value](t, raw)
				parse := func(raw jsontext.Value) *int {
					if string(raw) == "null" {
						return nil
					}
					value, ok := new(big.Int).SetString(string(raw), 10)
					require.True(t, ok, "reference bound must retain arbitrary precision")
					require.NotEqual(t, -1, value.Sign(), "emitted keyword must be nonnegative")
					if value.Cmp(limit) > 0 {
						overflow = true
						return nil
					}
					number := int(value.Int64())
					return &number
				}
				minimum, maximum := parse(branch["minimum"]), parse(branch["maximum"])
				if !overflow {
					require.NotNil(t, minimum)
					expected = append(expected, byteschema.Branch{Pattern: patterns[residue], MinLength: *minimum, MaxLength: maximum})
				}
			}
			actual, err := byteschema.Project(bounds.Minimum, bounds.Maximum)
			if overflow {
				require.Error(t, err, "unsafe encoded bound must not wrap, disappear or become unbounded")
				return
			}
			require.NoError(t, err)
			require.Equal(t, len(expected) == 0, actual.Unsatisfiable)
			require.Equal(t, expected, actual.Branches)
			if len(expected) > 0 {
				require.Equal(t, "[^A-Za-z0-9+/=]", actual.ForbiddenPattern)
			}
		})
	}
}
