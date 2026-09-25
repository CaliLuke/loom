package naming

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"
)

func TestServerDir(t *testing.T) {
	cases := []struct {
		Name     string
		Server   string
		Expected string
	}{
		{"ascii", "calc server", "calc_server"},
		{"ascii acronym", "APIServer", "api_server"},
		{"ascii dash", "-ipIp", "ipip"},
		{"latin accent", "Café", "cafu00e9"},
		{"caseless", "サーバー", "valu30b5u30fcu30d0u30fc"},
		{"astral letter", "𝒜lpha", "U0001d49clpha"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			dir := ServerDir(c.Server)
			assert.Equal(t, c.Expected, dir)
			assert.NoError(t, module.CheckImportPath("example.com/gen/http/cli/"+dir))
			assert.NoError(t, module.CheckFilePath("cmd/"+dir+"-cli/main.go"))
		})
	}
}
