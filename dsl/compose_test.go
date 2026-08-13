package dsl

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComposeDSL(t *testing.T) {
	var calls []string
	first := func() {
		calls = append(calls, "first")
	}
	next := func() {
		calls = append(calls, "next")
	}

	tests := []struct {
		name string
		fn   func()
		want []string
	}{
		{name: "both", fn: composeDSL(first, next), want: []string{"first", "next"}},
		{name: "first nil", fn: composeDSL(nil, next), want: []string{"next"}},
		{name: "next nil", fn: composeDSL(first, nil), want: []string{"first"}},
		{name: "both nil", fn: composeDSL(nil, nil)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls = nil
			if test.fn != nil {
				test.fn()
			}
			require.Equal(t, test.want, calls)
		})
	}
}
