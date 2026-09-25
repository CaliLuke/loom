package expr_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestGeneratedDirCollisions(t *testing.T) {
	const deseret = "\U00010428" // DESERET SMALL LETTER LONG I, escaped as U00010428
	cases := []struct {
		Name     string
		Servers  []string
		Services []string
		Errors   []string
	}{
		{Name: "distinct servers", Servers: []string{"calc", "adder"}},
		{Name: "accented server and its ASCII spelling", Servers: []string{"Café", "cafe"}},
		{Name: "distinct escaped servers", Servers: []string{"Café", "Cafè"}},
		{
			Name:    "escaped server and its escape",
			Servers: []string{"Café", "cafu00e9"},
			Errors:  []string{`Server cafu00e9: servers "Café" and "cafu00e9" both use the generated directory name "cafu00e9" (as in cmd/cafu00e9); rename one of them`},
		},
		{
			Name:    "space and underscore servers",
			Servers: []string{"calc server", "calc_server"},
			Errors:  []string{`Server calc_server: servers "calc server" and "calc_server" both use the generated directory name "calc_server" (as in cmd/calc_server); rename one of them`},
		},
		{
			Name:    "servers that differ in ASCII case",
			Servers: []string{"Calc", "calc"},
			Errors:  []string{`servers "Calc" and "calc" both use the generated directory name "calc"`},
		},
		{
			Name:    "servers whose directories differ only in case",
			Servers: []string{deseret, "u00010428"},
			Errors:  []string{`Server u00010428: servers "` + deseret + `" and "u00010428" use the generated directory names "U00010428" and "u00010428" (as in cmd/U00010428 and cmd/u00010428), which differ only in case; case-insensitive file systems merge them and the Go toolchain rejects import paths that differ only in case, so rename one of them`},
		},
		{
			Name:    "three colliding servers",
			Servers: []string{"calc server", "calc_server", "CalcServer"},
			Errors: []string{
				`servers "calc server" and "calc_server" both use the generated directory name "calc_server"`,
				`servers "calc server" and "CalcServer" both use the generated directory name "calc_server"`,
			},
		},
		{Name: "distinct services", Services: []string{"calc", "calc2"}},
		{Name: "accented service and its ASCII spelling", Services: []string{"Café", "cafe"}},
		{
			Name:     "escaped service and its escape",
			Services: []string{"Café", "cafu00e9"},
			Errors:   []string{`service "cafu00e9": services "Café" and "cafu00e9" both use the generated directory name "cafu00e9" (as in gen/cafu00e9); rename one of them`},
		},
		{
			Name:     "space and underscore services",
			Services: []string{"calc service", "calc_service"},
			Errors:   []string{`services "calc service" and "calc_service" both use the generated directory name "calc_service" (as in gen/calc_service)`},
		},
		{
			Name:     "reserved word service and its suffixed spelling",
			Services: []string{"string", "string_"},
			Errors:   []string{`services "string" and "string_" both use the generated directory name "string_"`},
		},
		{
			Name:     "services whose directories differ only in case",
			Services: []string{deseret, "u00010428"},
			Errors:   []string{`service "u00010428": services "` + deseret + `" and "u00010428" use the generated directory names "U00010428" and "u00010428" (as in gen/U00010428 and gen/u00010428), which differ only in case`},
		},
		{
			Name:     "server and service with the same directory",
			Servers:  []string{"calc"},
			Services: []string{"calc"},
		},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			design := generatedDirDSL(c.Servers, c.Services)
			if len(c.Errors) == 0 {
				expr.RunDSL(t, design)
				return
			}
			err := expr.RunInvalidDSL(t, design)
			require.Error(t, err)
			for _, want := range c.Errors {
				assert.Contains(t, err.Error(), want)
			}
		})
	}
}

func generatedDirDSL(servers, services []string) func() {
	if len(services) == 0 {
		services = []string{"svc"}
	}
	return func() {
		API("dirs", func() {
			for _, name := range servers {
				Server(name)
			}
		})
		for _, name := range services {
			Service(name, func() {
				Method("show", func() {})
			})
		}
	}
}
