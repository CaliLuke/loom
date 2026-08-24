package expr

import (
	"regexp/syntax"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPatgenBroadNonASCIIClassUsesBoundedAllocations(t *testing.T) {
	pattern, err := syntax.Parse(`^[\x80-\x{10FFFF}]$`, syntax.Perl)
	require.NoError(t, err)
	pattern = pattern.Simplify()
	random := &ExampleGenerator{Randomizer: boundedPatternRandomizer{value: 2}}

	allocations := testing.AllocsPerRun(5, func() {
		patgen(pattern, random)
	})

	require.Less(t, allocations, float64(10))
}

type boundedPatternRandomizer struct {
	DeterministicRandomizer
	value int
}

func (r boundedPatternRandomizer) Int() int {
	return r.value
}
