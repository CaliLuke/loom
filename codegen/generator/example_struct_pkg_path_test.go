package generator

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/internal/loomsource"
	"github.com/CaliLuke/loom/internal/testingx"
)

// TestExampleStructPkgPathTypesCompile generates the service, transport, and
// example output for services whose payloads and results use a user type
// placed in its own package with "struct:pkg:path", then builds and vets the
// module. The example service stubs and multipart.go must import that
// package, under an alias when its name clashes with another import.
func TestExampleStructPkgPathTypesCompile(t *testing.T) {
	cases := []struct {
		Name   string
		Design structPkgPathDesign
		// Stub and Multipart are the imports and qualifiers of the type
		// package in the service stub and in multipart.go, the "common"
		// package when empty.
		Stub, Multipart pkgRef
	}{
		{Name: "http", Design: structPkgPathDesign{HTTP: true, List: true}},
		{Name: "grpc", Design: structPkgPathDesign{GRPC: true, List: true}},
		{Name: "jsonrpc", Design: structPkgPathDesign{JSONRPC: true, List: true}},
		{Name: "http-and-grpc", Design: structPkgPathDesign{HTTP: true, GRPC: true, List: true}},
		{Name: "http-multipart", Design: structPkgPathDesign{HTTP: true, Multipart: true, List: true}},
		{
			Name:      "package-named-log",
			Design:    structPkgPathDesign{HTTP: true, Multipart: true, List: true, Maps: true, PkgPath: "types/log"},
			Stub:      pkgRef{Import: `log2 "example.com/catalog/gen/types/log"`, Qualifier: "log2"},
			Multipart: pkgRef{Import: `log "example.com/catalog/gen/types/log"`, Qualifier: "log"},
		},
		{
			Name:      "non-ascii-package-path",
			Design:    structPkgPathDesign{HTTP: true, Multipart: true, List: true, Maps: true, PkgPath: "tipos/menü"},
			Stub:      pkgRef{Import: `menu00fc "example.com/catalog/gen/tipos/menu00fc"`, Qualifier: "menu00fc"},
			Multipart: pkgRef{Import: `menu00fc "example.com/catalog/gen/tipos/menu00fc"`, Qualifier: "menu00fc"},
		},
		{
			Name:   "package-named-errors",
			Design: structPkgPathDesign{HTTP: true, PkgPath: "types/errors"},
			Stub:   pkgRef{Import: `errors_ "example.com/catalog/gen/types/errors"`, Qualifier: "errors_"},
		},
	}
	repoRoot, err := loomsource.RepositoryRoot(".")
	require.NoError(t, err)
	source, err := loomsource.Resolve(repoRoot, filepath.Join(t.TempDir(), "loom-pinned"))
	require.NoError(t, err)
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			codegen.RunDSL(t, c.Design.DSL)
			stubRef, multipartRef := c.Stub.orCommon(), c.Multipart.orCommon()
			dir := t.TempDir()
			goMod := fmt.Sprintf("module example.com/catalog\n\ngo 1.27\n\nrequire github.com/CaliLuke/loom v0.0.0\n\nreplace github.com/CaliLuke/loom => %s\n", source)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o600))

			_, err := Generate(dir, "gen", false)
			require.NoError(t, err)
			_, err = Generate(dir, "example", false)
			require.NoError(t, err)

			stub, err := os.ReadFile(filepath.Join(dir, "catalog.go"))
			require.NoError(t, err)
			qual := stubRef.Qualifier
			require.Contains(t, string(stub), stubRef.Import)
			require.Contains(t, string(stub), "p *"+qual+".Item) (res *"+qual+".Item, err error)")
			if c.Design.List {
				require.Contains(t, string(stub), "(res []*"+qual+".Item, err error)")
			}
			if c.Design.Maps {
				require.Contains(t, string(stub), "(res map[string]*"+qual+".Item, err error)")
				require.Contains(t, string(stub), "p map[string][]*"+qual+".Item) (res [][]*"+qual+".Item, err error)")
			}
			if c.Design.Multipart {
				multipart, err := os.ReadFile(filepath.Join(dir, "multipart.go"))
				require.NoError(t, err)
				require.Contains(t, string(multipart), multipartRef.Import)
				require.Contains(t, string(multipart), "p *"+multipartRef.Qualifier+".Item) error")
			}

			_, err = testingx.RunCmd(dir, "go", "mod", "tidy")
			require.NoError(t, err)
			output, err := testingx.RunCmd(dir, "go", "build", "./...")
			require.NoError(t, err, output)
			output, err = testingx.RunCmd(dir, "go", "vet", "./...")
			require.NoError(t, err, output)
		})
	}
}

// pkgRef is the import and qualifier of a type package in an example file.
type pkgRef struct {
	// Import is the import declaration.
	Import string
	// Qualifier qualifies the package's types.
	Qualifier string
}

// orCommon returns r, or the "common" package when r is empty.
func (r pkgRef) orCommon() pkgRef {
	if r.Import == "" {
		return pkgRef{Import: `common "example.com/catalog/gen/common"`, Qualifier: "common"}
	}
	return r
}

// structPkgPathDesign declares a "catalog" service whose "put" method payload
// and result use the "Item" type generated in a struct:pkg:path package.
type structPkgPathDesign struct {
	// HTTP, GRPC, and JSONRPC select the transports that expose the methods.
	HTTP, GRPC, JSONRPC bool
	// Multipart makes the HTTP "put" endpoint decode a multipart request.
	Multipart bool
	// List adds a "list" method that returns a collection of "Item".
	List bool
	// Maps adds HTTP methods whose payloads and results are maps and nested
	// collections of "Item".
	Maps bool
	// PkgPath is the struct:pkg:path of "Item", "common" when empty.
	PkgPath string
}

// DSL declares the design selected by d.
func (d structPkgPathDesign) DSL() {
	pkgPath := d.PkgPath
	if pkgPath == "" {
		pkgPath = "common"
	}
	dsl.API("catalog", func() {
		if d.JSONRPC {
			dsl.JSONRPC(func() {})
		}
	})
	item := dsl.Type("Item", func() {
		dsl.Meta("struct:pkg:path", pkgPath)
		dsl.Field(1, "id", dsl.String)
		dsl.Field(2, "name", dsl.String)
		dsl.Required("id")
	})
	transports := func(route func()) {
		if d.HTTP {
			dsl.HTTP(route)
		}
		if d.GRPC {
			dsl.GRPC(func() {})
		}
		if d.JSONRPC {
			dsl.JSONRPC(func() {})
		}
	}
	dsl.Service("catalog", func() {
		if d.JSONRPC {
			dsl.JSONRPC(func() {
				dsl.POST("/rpc")
			})
		}
		dsl.Method("put", func() {
			dsl.Payload(item)
			dsl.Result(item)
			transports(func() {
				dsl.POST("/items")
				if d.Multipart {
					dsl.MultipartRequest()
				}
			})
		})
		if d.List {
			dsl.Method("list", func() {
				dsl.Result(dsl.ArrayOf(item))
				transports(func() {
					dsl.GET("/items")
				})
			})
		}
		if d.Maps {
			dsl.Method("index", func() {
				dsl.Result(dsl.MapOf(dsl.String, item))
				dsl.HTTP(func() {
					dsl.GET("/index")
				})
			})
			dsl.Method("regroup", func() {
				dsl.Payload(dsl.MapOf(dsl.String, dsl.ArrayOf(item)))
				dsl.Result(dsl.ArrayOf(dsl.ArrayOf(item)))
				dsl.HTTP(func() {
					dsl.POST("/regroup")
				})
			})
		}
	})
}
