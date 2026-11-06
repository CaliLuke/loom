package loom

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCheckAdditionalMembers(t *testing.T) {
	cases := []struct {
		name     string
		members  map[string]int
		declared []string
		want     string
	}{
		{name: "nil members", members: nil, declared: []string{"id"}},
		{name: "no declared names", members: map[string]int{"id": 1}},
		{name: "distinct names", members: map[string]int{"other": 1}, declared: []string{"id", "count"}},
		{name: "exact names only", members: map[string]int{"ID": 1}, declared: []string{"id"}},
		{name: "collision", members: map[string]int{"count": 1}, declared: []string{"id", "count"}, want: `Labels: additional member "count" collides with a declared member`},
		{name: "first declared collision", members: map[string]int{"id": 1, "count": 2}, declared: []string{"id", "count"}, want: `Labels: additional member "id" collides with a declared member`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := CheckAdditionalMembers(tc.members, "Labels", tc.declared...)
			if tc.want == "" {
				require.NoError(t, err)
				return
			}
			require.EqualError(t, err, tc.want)
			var serviceError *ServiceError
			require.False(t, errors.As(err, &serviceError), "an encoding collision is not a validation error")
		})
	}
}

func TestInvalidAdditionalMemberError(t *testing.T) {
	err := InvalidAdditionalMemberError("body.id")
	var serviceError *ServiceError
	require.ErrorAs(t, err, &serviceError)
	require.Equal(t, InvalidFieldType, serviceError.Name)
	require.NotNil(t, serviceError.Field)
	require.Equal(t, "body.id", *serviceError.Field)
	require.False(t, serviceError.Fault)
	require.Equal(t, `invalid member "body.id"; a declared member cannot appear as an additional member`, serviceError.Message)
	require.NotNil(t, serviceError.Remedy)
	require.Equal(t, serviceError.Message, serviceError.Remedy.SafeMessage)
}
