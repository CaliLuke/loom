package service

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
)

// TestStructPkgPathStreamingTypes checks that the streaming payloads and
// results of a type generated in a struct:pkg:path package are referenced
// from that package, generated in that package only, and imported.
func TestStructPkgPathStreamingTypes(t *testing.T) {
	const genpkg = "example.com/app/gen"
	root := codegen.RunDSL(t, func() {
		plain := dsl.Type("Plain", func() {
			dsl.Meta("struct:pkg:path", "menu")
			dsl.Attribute("name", dsl.String)
		})
		local := dsl.Type("Local", func() {
			dsl.Attribute("id", dsl.String)
		})
		dsl.Service("svc", func() {
			dsl.Method("upload", func() {
				dsl.StreamingPayload(plain)
				dsl.Result(plain)
			})
			dsl.Method("relay", func() {
				dsl.Payload(local)
				dsl.StreamingPayload(plain)
				dsl.StreamingResult(plain)
			})
			dsl.Method("mixed", func() {
				dsl.Result(local)
				dsl.StreamingResult(plain)
			})
			dsl.Method("list", func() {
				dsl.StreamingPayload(dsl.ArrayOf(plain))
				dsl.StreamingResult(local)
			})
		})
	})
	services := NewServicesData(root)
	data := services.Get("svc")
	require.NoError(t, SetUserTypeImports(genpkg, data))
	loc := &codegen.Location{FilePath: filepath.Join("menu", "plain.go"), RelImportPath: "menu"}
	cases := []struct {
		Method              string
		StreamingPayloadRef string
		StreamingResultRef  string
		SendRef, RecvRef    string
	}{
		{Method: "upload", StreamingPayloadRef: "*menu.Plain", SendRef: "*menu.Plain", RecvRef: "*menu.Plain"},
		{Method: "relay", StreamingPayloadRef: "*menu.Plain", SendRef: "*menu.Plain", RecvRef: "*menu.Plain"},
		{Method: "mixed", StreamingResultRef: "*menu.Plain", SendRef: "*menu.Plain"},
		{Method: "list", StreamingPayloadRef: "[]*menu.Plain", SendRef: "*Local", RecvRef: "[]*menu.Plain"},
	}
	for _, c := range cases {
		t.Run(c.Method, func(t *testing.T) {
			m := data.Method(c.Method)
			require.NotNil(t, m)
			assert.Equal(t, c.StreamingPayloadRef, m.StreamingPayloadRef)
			assert.Equal(t, c.StreamingResultRef, m.StreamingResultRef)
			assert.Equal(t, c.SendRef, m.ServerStream.SendTypeRef)
			assert.Equal(t, c.RecvRef, m.ServerStream.RecvTypeRef)
			if c.StreamingPayloadRef == "*menu.Plain" {
				assert.Equal(t, loc, m.StreamingPayloadLoc)
			}
			if c.StreamingResultRef != "" {
				assert.Equal(t, loc, m.StreamingResultLoc)
			}
		})
	}
	assert.Equal(t, []*codegen.ImportSpec{{Name: "menu", Path: genpkg + "/menu"}}, data.UserTypeImports)

	files := Files(genpkg, root.Services[0], services, make(map[string][]string))
	var typeFile *codegen.File
	for _, f := range files {
		if f.Path == filepath.Join(codegen.Gendir, "menu", "plain.go") {
			typeFile = f
		}
	}
	require.NotNil(t, typeFile, "the Plain type must be generated in its struct:pkg:path package")
	svcCode := codegen.SectionsCode(t, files[0].AllSections()[1:])
	assert.NotContains(t, svcCode, "type Plain struct")
	assert.NotContains(t, svcCode, "(*Plain")
	assert.Contains(t, codegen.SectionsCode(t, typeFile.AllSections()[1:]), "type Plain struct")
}

// TestServicePackageNameAvoidsUserTypePackages checks that a service package
// is named apart from the struct:pkg:path packages of its types, which the
// transport and example files import next to it.
func TestServicePackageNameAvoidsUserTypePackages(t *testing.T) {
	cases := []struct {
		Name, Service, PkgPath string
		// Error uses the type package through an error type only, Forced
		// through a type generated with "type:generate:force" only.
		// NoTransport leaves the service without transports, so no generated
		// file imports both packages under the same name.
		Error, Forced, NoTransport bool
		PkgName                    string
	}{
		{Name: "same-name-no-transport", Service: "catalog", PkgPath: "types/catalog", NoTransport: true, PkgName: "catalog"},
		{Name: "same-name-error-no-transport", Service: "catalog", PkgPath: "types/catalog", Error: true, NoTransport: true, PkgName: "catalog"},
		{Name: "other-package", Service: "catalog", PkgPath: "types/other", PkgName: "catalog"},
		{Name: "same-name", Service: "catalog", PkgPath: "types/catalog", PkgName: "catalogsvc"},
		{Name: "same-name-error", Service: "catalog", PkgPath: "types/catalog", Error: true, PkgName: "catalogsvc"},
		{Name: "same-name-forced", Service: "catalog", PkgPath: "types/catalog", Forced: true, PkgName: "catalogsvc"},
		{Name: "service-package-forced", Service: "catalog", PkgPath: "catalog", Forced: true, PkgName: "catalog"},
		{Name: "common", Service: "common", PkgPath: "types/common", PkgName: "commonsvc"},
		{Name: "escaped", Service: "menü", PkgPath: "types/menü", PkgName: "menu00fcsvc"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				item := dsl.Type("Item", func() {
					dsl.Meta("struct:pkg:path", c.PkgPath)
					dsl.Attribute("id", dsl.String)
				})
				failure := dsl.Type("Failure", func() {
					dsl.Meta("struct:pkg:path", c.PkgPath)
					dsl.Attribute("message", dsl.String)
				})
				dsl.Type("Forced", func() {
					dsl.Meta("struct:pkg:path", c.PkgPath)
					if c.Forced {
						dsl.Meta("type:generate:force")
					}
					dsl.Attribute("id", dsl.String)
				})
				dsl.Service(c.Service, func() {
					dsl.Method("show", func() {
						if !c.NoTransport {
							dsl.HTTP(func() {
								dsl.POST("/show")
							})
						}
						if c.Error {
							dsl.Error("failed", failure)
							return
						}
						if c.Forced {
							return
						}
						dsl.Payload(func() {
							dsl.Attribute("items", dsl.MapOf(dsl.String, dsl.ArrayOf(item)))
						})
					})
				})
			})
			data := NewServicesData(root).Get(c.Service)
			require.NotNil(t, data)
			assert.Equal(t, c.PkgName, data.PkgName)
		})
	}
}

// TestSetUserTypeImportsRejectsServicePackage checks that a type that a
// service uses cannot be placed in the package generated for that service:
// the service package would import itself.
func TestSetUserTypeImportsRejectsServicePackage(t *testing.T) {
	const ownPackageError = `service "common": type "Item" has struct:pkg:path "common", the package generated for the service; the service package cannot import itself. Remove the struct:pkg:path metadata to generate the type in the service package, or use another path such as "types/common"`
	cases := []struct {
		Name, Service, PkgPath string
		// Use selects how the service uses Item: "list" returns a list,
		// "payload" takes it as payload, "error" returns it as an error and
		// "forced" only forces its generation.
		Use   string
		Error string
	}{
		{Name: "own-package", Service: "common", PkgPath: "common", Use: "list", Error: ownPackageError},
		{Name: "own-package-payload", Service: "common", PkgPath: "common", Use: "payload", Error: ownPackageError},
		{Name: "own-package-error", Service: "common", PkgPath: "common", Use: "error", Error: ownPackageError},
		{Name: "own-package-forced", Service: "common", PkgPath: "common", Use: "forced"},
		{Name: "escaped", Service: "Menü", PkgPath: "menü", Use: "list", Error: `service "Menü": type "Item" has struct:pkg:path "menü", the package generated for the service; the service package cannot import itself. Remove the struct:pkg:path metadata to generate the type in the service package, or use another path such as "types/menü"`},
		{Name: "other-service-package", Service: "catalog", PkgPath: "common", Use: "list"},
		{Name: "nested", Service: "common", PkgPath: "common/types", Use: "list"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				item := dsl.Type("Item", func() {
					dsl.Meta("struct:pkg:path", c.PkgPath)
					if c.Use == "forced" {
						dsl.Meta("type:generate:force")
					}
					dsl.Attribute("id", dsl.String)
				})
				dsl.Service(c.Service, func() {
					dsl.Method("show", func() {
						switch c.Use {
						case "list":
							dsl.Result(dsl.ArrayOf(item))
						case "payload":
							dsl.Payload(item)
						case "error":
							dsl.Error("failed", item)
						}
					})
				})
			})
			err := SetUserTypeImports("example.com/app/gen", NewServicesData(root).Get(c.Service))
			if c.Error == "" {
				assert.NoError(t, err)
				return
			}
			assert.EqualError(t, err, c.Error)
		})
	}
}
