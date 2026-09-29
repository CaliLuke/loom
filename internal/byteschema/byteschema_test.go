package byteschema

import (
	"encoding/base64"
	"encoding/json/v2"
	"fmt"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProjectDecodedLength(t *testing.T) {
	bounds := []*int{nil, pointer(-2), pointer(-1), pointer(0), pointer(1), pointer(2), pointer(3), pointer(4), pointer(5), pointer(9)}
	for _, lower := range bounds {
		for _, upper := range bounds {
			constraint, err := Project(lower, upper)
			require.NoError(t, err)
			require.LessOrEqual(t, len(constraint.Branches), 3)
			for length := range 13 {
				wire := base64.StdEncoding.EncodeToString(make([]byte, length))
				want := (lower == nil || length >= *lower) && (upper == nil || length <= *upper)
				require.Equal(t, want, matches(t, constraint, wire), "bounds=%v/%v length=%d", lower, upper, length)
			}
		}
	}
}

func TestProjectCodecLanguage(t *testing.T) {
	constraint, err := Project(nil, nil)
	require.NoError(t, err)
	cases := []string{"", "aA==", "aGk=", "aGl=", "aGm=", "aGn=", "aGlq", "aGk", "aGk==", "aGk===", "aGk=\n", "aGk=\r", "aGk=\r\n", "aGk=\t", "aGk= ", "aGk=\x00", "aGk=\u2028", "aGk=\u2029", "aG-k", "aG_k", "=AAA", "A===", "===="}
	for _, text := range cases {
		t.Run(fmt.Sprintf("%q", text), func(t *testing.T) {
			wire, err := json.Marshal(text)
			require.NoError(t, err)
			var decoded []byte
			accepted := json.Unmarshal(wire, &decoded) == nil
			require.Equal(t, accepted, matches(t, constraint, text))
		})
	}
}

func TestProjectLargeBoundsAndOwnership(t *testing.T) {
	lower, upper := 1, 8
	constraint, err := Project(&lower, &upper)
	require.NoError(t, err)
	lower, upper = 100, 100
	require.True(t, matches(t, constraint, "aA=="))
	for _, bound := range []int{int(^uint(0) >> 1), int(^uint(0)>>1) - 2} {
		_, err := Project(nil, &bound)
		require.Error(t, err)
		_, err = Project(&bound, nil)
		require.Error(t, err)
	}
	large := 3000001
	constraint, err = Project(&large, &large)
	require.NoError(t, err)
	require.Len(t, constraint.Branches, 1)
	require.Less(t, len(constraint.Branches[0].Pattern), 80)
	require.Equal(t, 4000004, constraint.Branches[0].MinLength)
	require.Equal(t, 4000004, *constraint.Branches[0].MaxLength)
	negative := -1
	constraint, err = Project(&lower, &negative)
	require.NoError(t, err)
	require.True(t, constraint.Unsatisfiable)
	require.Empty(t, constraint.Branches)
}

func TestProjectExactSchemaIntegerBoundary(t *testing.T) {
	limit := min(uint64(^uint(0)>>1), uint64(1<<53-1))
	largest := int(limit / 4 * 3)
	constraint, err := Project(nil, &largest)
	require.NoError(t, err)
	for _, branch := range constraint.Branches {
		require.LessOrEqual(t, uint64(*branch.MaxLength), limit)
	}
	tooLarge := largest + 1
	_, err = Project(nil, &tooLarge)
	require.Error(t, err)
	lower := int(^uint(0) >> 1)
	constraint, err = Project(&lower, pointer(1))
	require.NoError(t, err)
	require.True(t, constraint.Unsatisfiable)
}

func pointer(value int) *int {
	return &value
}

func matches(t *testing.T, constraint Constraint, text string) bool {
	t.Helper()
	if constraint.Unsatisfiable {
		return false
	}
	forbidden, err := regexp.Compile(constraint.ForbiddenPattern)
	require.NoError(t, err)
	if forbidden.MatchString(text) {
		return false
	}
	for _, branch := range constraint.Branches {
		pattern, err := regexp.Compile(branch.Pattern)
		require.NoError(t, err)
		if pattern.MatchString(text) && len(text) >= branch.MinLength && (branch.MaxLength == nil || len(text) <= *branch.MaxLength) {
			return true
		}
	}
	return false
}

func TestIntersect(t *testing.T) {
	negative, lower, upper, large := -2, 2, 3, int(^uint(0)>>1)
	bounds := []Bounds{{Minimum: &negative, Maximum: &large}, {Minimum: &lower, Maximum: &upper}}
	got := Intersect(bounds)
	require.Equal(t, lower, *got.Minimum)
	require.Equal(t, upper, *got.Maximum)
	lower, upper = 7, 8
	require.Equal(t, 2, *got.Minimum)
	require.Equal(t, 3, *got.Maximum)
	require.Equal(t, Bounds{}, Intersect(nil))
	empty := Intersect([]Bounds{{Minimum: &large}, {Maximum: &negative}})
	projection, err := Project(empty.Minimum, empty.Maximum)
	require.NoError(t, err)
	require.True(t, projection.Unsatisfiable)
}
