package naming

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportName(t *testing.T) {
	for _, tc := range []struct {
		path     string
		explicit string
		want     string
	}{
		{"fmt", "", "fmt"},
		{"net/http", "", "http"},
		{"gopkg.in/yaml.v3", "", "yaml"},
		{"example.com/pkg.2", "", "pkg"},
		{"example.com/go-client/v2", "client", "client"},
		{"gopkg.in/yaml.v3", "yaml3", "yaml3"},
		{"example.com/sideeffect", "_", "_"},
		{"example.com/dsl", ".", "."},
		{"", "", ""},
	} {
		t.Run(tc.path+":"+tc.explicit, func(t *testing.T) {
			require.Equal(t, tc.want, ImportName(tc.path, tc.explicit))
		})
	}
}
