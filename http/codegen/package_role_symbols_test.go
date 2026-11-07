package codegen

import (
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/stretchr/testify/require"
)

func TestHTTPPackageRoleSymbols(t *testing.T) {
	for _, collision := range []bool{false, true} {
		name, names := "ordinary", []string{"Choice"}
		if collision {
			name, names = "colliding", packageRoleCollisionNames()
		}
		t.Run(name, func(t *testing.T) {
			root := RunHTTPDSL(t, packageBodySymbolDSL(names, false, false, true))
			data := CreateHTTPServices(root).Get("carrier")
			dir := t.TempDir()
			renderHTTPModuleFiles(t, dir, "example.com/carrier/gen", root)
			for _, pkg := range []string{"server", "client"} {
				if duplicates := packageSymbolDuplicates(t, filepath.Join(dir, "gen", "http", "carrier", pkg)); len(duplicates) > 0 {
					t.Errorf("%s package duplicates: %v", pkg, duplicates)
				}
			}
			require.Greater(t, len(data.Endpoints), clientOperationGroupThreshold)
			require.Len(t, data.clientGroups, 1)
			require.Equal(t, "Items", data.clientGroups[0].FieldName)
			groupName := "ItemsClient"
			if collision {
				groupName += "2"
			}
			require.Equal(t, groupName, data.clientGroups[0].Name)
			for _, endpoint := range data.Endpoints {
				switch endpoint.Method.Name {
				case "read":
					require.Len(t, endpoint.Routes, 2)
					require.Equal(t, "ReadCarrierPath", endpoint.Routes[0].PathInit.Name)
					require.Equal(t, "ReadCarrierPath2", endpoint.Routes[1].PathInit.Name)
				case "upload":
					require.NotNil(t, endpoint.MultipartRequestDecoder)
					require.NotNil(t, endpoint.MultipartRequestEncoder)
					require.Equal(t, "CarrierUploadDecoderFunc", endpoint.MultipartRequestDecoder.FuncName)
					require.Equal(t, "NewCarrierUploadDecoder", endpoint.MultipartRequestDecoder.InitName)
					require.Equal(t, "CarrierUploadEncoderFunc", endpoint.MultipartRequestEncoder.FuncName)
					require.Equal(t, "NewCarrierUploadEncoder", endpoint.MultipartRequestEncoder.InitName)
				}
			}
		})
	}
}

func packageRoleCollisionNames() []string {
	return []string{
		"ItemsClient", "ReadCarrierPath", "ReadCarrierPath2",
		"CarrierUploadDecoderFunc", "NewCarrierUploadDecoder",
		"CarrierUploadEncoderFunc", "NewCarrierUploadEncoder",
	}
}

// packageRoleMethods reuses the opaque file-array carrier exercised by
// testdata.CustomMultipartBindingsDSL, alongside enough ordinary endpoints to
// enable operation groups. read owns the two path-builder roles in the caller.
func packageRoleMethods() {
	Method("upload", func() {
		Payload(func() {
			Attribute("files", ArrayOf(Bytes), func() {
				Meta("struct:field:type", "[][]byte")
			})
			Required("files")
		})
		HTTP(func() {
			POST("/items/upload")
			MultipartRequest()
			Response(StatusNoContent)
		})
	})
	for _, name := range []string{"a", "b", "c", "d", "e", "f"} {
		Method(name, func() {
			HTTP(func() {
				GET("/items/" + name)
				Response(StatusNoContent)
			})
		})
	}
}
