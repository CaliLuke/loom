package expr

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestProblemResultProperties(t *testing.T) {
	for _, name := range []string{"timeout", "temporary", "fault"} {
		t.Run(name, func(t *testing.T) {
			field := ErrorResult.Find(name)
			require.NotNil(t, field)
			require.Equal(t, Boolean, field.Type)
			require.True(t, ErrorResult.IsRequired(name))
			require.NotNil(t, ErrorResult.Views[0].Find(name))
			require.False(t, AllowsNull(field))
		})
	}
}
