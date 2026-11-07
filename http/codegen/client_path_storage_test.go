package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestClientPathNativeStorage(t *testing.T) {
	for _, storage := range []string{"uint64", "int8", "time.Duration"} {
		t.Run(storage, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				value := Type("Value", Int, func() {
					if storage == "time.Duration" {
						Meta("struct:field:type", storage, "time")
					} else {
						Meta("struct:field:type", storage)
					}
				})
				Service("paths", func() {
					Method("read", func() {
						Payload(func() {
							Attribute("id", value)
							Required("id")
						})
						HTTP(func() {
							GET("/{id}")
						})
					})
				})
			})
			endpoint := CreateHTTPServices(root).Get("paths").Endpoint("read")
			arg := endpoint.Routes[0].PathInit.ClientArgs[0]
			require.Equal(t, storage, arg.TypeRef)
			require.Equal(t, arg.TypeRef, arg.ServiceTypeRef, "conversion must target the actual path argument declaration")
			require.Contains(t, endpoint.RequestInit.ClientCode, storage+"(p.ID)")
		})
	}
}
