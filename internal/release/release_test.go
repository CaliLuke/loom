package release

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestChooseAlpha(t *testing.T) {
	for _, tc := range []struct {
		name string
		tags []tag
		sha  string
		want string
		skip bool
	}{
		{name: "first", sha: "b", want: "v1.10.0-alpha.1"},
		{name: "new commit", tags: []tag{{"v1.10.0-alpha.5", "a"}}, sha: "b", want: "v1.10.0-alpha.6"},
		{name: "retry same commit", tags: []tag{{"v1.10.0-alpha.5", "a"}}, sha: "a", want: "v1.10.0-alpha.5"},
		{name: "stable already published", tags: []tag{{"v1.10.0", "a"}}, sha: "a", skip: true},
		{name: "same source on another train", tags: []tag{{"v1.9.0-alpha.9", "a"}}, sha: "a", skip: true},
		{name: "closed train", tags: []tag{{"v1.10.0", "a"}}, sha: "b", skip: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := chooseAlpha("v1.10.0", tc.sha, tc.tags)
			require.NoError(t, err)
			require.Equal(t, tc.want, got)
			require.Equal(t, tc.skip, got == "")
		})
	}
}

func TestPromotionVersion(t *testing.T) {
	for _, tc := range []struct {
		alpha, stable string
		good          bool
	}{
		{"v1.10.0-alpha.6", "v1.10.0", true},
		{"v1.10.0-alpha.6", "v1.11.0", false},
		{"v1.10.0", "v1.10.0", false},
		{"v1.10.0-beta.1", "v1.10.0", false},
	} {
		t.Run(tc.alpha+tc.stable, func(t *testing.T) {
			err := validatePromotion(tc.alpha, tc.stable)
			require.Equal(t, tc.good, err == nil)
		})
	}
}
