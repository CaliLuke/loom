package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResponseAllocationPreservesEveryHistoricalContext(t *testing.T) {
	semantic := strings.Repeat("a", 40)
	uses := []responseComponentUse{
		responseAllocationUse(semantic, "new", "New", false),
		responseAllocationUse(semantic, "old-a", "First", true),
		responseAllocationUse(semantic, "old-b", "Second", true),
		responseAllocationUse(semantic, "old-c", "Third", true),
		responseAllocationUse(semantic, "new-again", "NewAgain", false),
	}
	names, allocator := responseAllocatedNames(uses)
	require.Equal(t, []string{"First", "First", "Second", "Third", "First"}, names)
	require.Same(t, uses[1].ref.Value, allocator.values["First"], "new uses cannot replace the historical producer")
	require.Same(t, uses[2].ref.Value, allocator.values["Second"])
	require.Same(t, uses[3].ref.Value, allocator.values["Third"])
}

func TestResponseAllocationKeepsRetiredRepresentativeReserved(t *testing.T) {
	first, second := strings.Repeat("a", 40), strings.Repeat("b", 40)
	uses := []responseComponentUse{
		responseAllocationUse(first, "shared", "Public", false),
		responseAllocationUse(second, "shared", "Public", false),
		responseAllocationUse(second, "shared", "Public", false),
	}
	names, allocator := responseAllocatedNames(uses)
	require.Equal(t, []string{"", "Public_bbbbbbbb", "Public_bbbbbbbb"}, names)
	require.True(t, allocator.reserved["Public"])
	require.NotContains(t, allocator.values, "Public")
}

func TestResponseAllocationReservesFutureAndAuthoredNames(t *testing.T) {
	first, second, third := strings.Repeat("a", 40), strings.Repeat("b", 40), strings.Repeat("c", 40)
	uses := []responseComponentUse{
		responseAllocationUse(first, "shared", "Public", true),
		responseAllocationUse(second, "shared", "Public", true),
		responseAllocationUse(third, "later", "Public_bbbbbbbb", true),
		// The parent old-key reuse elides this separate authored preferred name;
		// it still must not be stolen by an unrelated fresh fallback.
		responseAllocationUse(first, "shared", "Public_bbbbbbbbbbbbbbbb", true),
	}
	names, _ := responseAllocatedNames(uses)
	require.Equal(t, []string{"Public", "Public_bbbbbbbbbbbbbbbbbbbbbbbb", "Public_bbbbbbbb", "Public"}, names)
}

func TestResponseAllocationFallbackExhaustsOccupiedPrefixes(t *testing.T) {
	first, second := strings.Repeat("a", 40), strings.Repeat("b", 40)
	uses := []responseComponentUse{
		responseAllocationUse(first, "shared", "Public", true),
		responseAllocationUse(second, "shared", "Public", true),
	}
	for width := 8; width <= 40; width += 8 {
		uses = append(uses, responseAllocationUse(first, "shared", "Public_"+second[:width], true))
	}
	uses = append(uses, responseAllocationUse(first, "shared", "Public_"+second+"_2", true))
	names, _ := responseAllocatedNames(uses)
	require.Equal(t, "Public_"+second+"_3", names[1])
}

func TestResponseAllocationReusesUnassignedSemanticBindings(t *testing.T) {
	semantic := strings.Repeat("a", 40)
	uses := []responseComponentUse{
		responseAllocationUse(semantic, "first", "Public", false),
		responseAllocationUse(semantic, "second", "Other", false),
		responseAllocationUse(semantic, "third", "Another", false),
	}
	names, _ := responseAllocatedNames(uses)
	require.Equal(t, []string{"Public_aaaaaaaa", "Public_aaaaaaaa", "Public_aaaaaaaa"}, names)
}

func responseAllocationUse(semantic, allocation, base string, forced bool) responseComponentUse {
	return responseComponentUse{
		ref:  &ResponseRef{Value: &Response{Description: "same complete response"}},
		base: base, keys: responseIdentity{semantic: semantic, allocation: allocation}, forced: forced,
	}
}

func responseAllocatedNames(uses []responseComponentUse) ([]string, *responseNameAllocator) {
	allocator := newResponseNameAllocator(uses)
	names := make([]string, len(uses))
	for i, use := range uses {
		if use.forced || allocator.counts[use.keys.semantic] >= 2 {
			names[i] = allocator.name(use)
		}
	}
	return names, allocator
}
