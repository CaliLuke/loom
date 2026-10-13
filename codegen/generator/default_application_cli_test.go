package generator

import (
	"testing"

	"github.com/CaliLuke/loom/dsl"
)

const defaultApplicationCLIHarness = `package defaultcli_test

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	cli "example.com/defaultcli/gen/http/cli/default_cli"
	"example.com/defaultcli/gen/values"
)

func TestCLIValues(t *testing.T) {
	original := os.Args
	t.Cleanup(func() {
		os.Args = original
	})
	for _, tc := range []struct {
		name string
		args []string
		flag bool
		count int
	}{
		{"omitted", nil, true, 7},
		{"explicit zero", []string{"--flag", "false", "--count", "0"}, false, 0},
		{"populated", []string{"--flag", "true", "--count", "9"}, true, 9},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = append([]string{"defaults", "values", "read"}, tc.args...)
			endpoint, payload, err := cli.ParseEndpoint("http", "localhost", nil, nil, nil, false)
			require.NoError(t, err)
			require.NotNil(t, endpoint)
			actual, ok := payload.(*values.ReadPayload)
			require.True(t, ok)
			require.Equal(t, tc.flag, actual.Flag)
			require.Equal(t, tc.count, actual.Count)
		})
	}
}
`

func TestDefaultApplicationCLI(t *testing.T) {
	runDesignHarness(t, "example.com/defaultcli", defaultApplicationCLIDSL, defaultApplicationCLIHarness)
}

func defaultApplicationCLIDSL() {
	dsl.API("default_cli", func() {
	})
	dsl.Service("values", func() {
		dsl.Method("read", func() {
			dsl.Payload(func() {
				dsl.Attribute("flag", dsl.Boolean, func() {
					dsl.Default(true)
				})
				dsl.Attribute("count", dsl.Int, func() {
					dsl.Default(7)
				})
			})
			dsl.HTTP(func() {
				dsl.GET("/")
				dsl.Param("flag")
				dsl.Param("count")
			})
		})
	})
}
