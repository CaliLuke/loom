package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/mod/module"
)

func TestServicePackageNames(t *testing.T) {
	cases := []struct {
		Name         string
		Service      string
		ExpectedPath string
		ExpectedPkg  string
	}{
		{"ascii", "calc service", "calc_service", "calcservice"},
		{"ascii acronym", "MyAPIService", "my_api_service", "myapiservice"},
		{"latin accent", "Café", "cafu00e9", "cafu00e9"},
		{"cjk", "日本語", "u65e5u672cu8a9e", "u65e5u672cu8a9e"},
		{"astral letter", "𝒜lpha", "U0001d49clpha", "U0001d49clpha"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			path := DirName(c.Service)
			assert.Equal(t, c.ExpectedPath, path)
			assert.NoError(t, module.CheckImportPath("example.com/gen/"+path))
			assert.Equal(t, c.ExpectedPkg, PackageBaseName(c.Service))
		})
	}
}
