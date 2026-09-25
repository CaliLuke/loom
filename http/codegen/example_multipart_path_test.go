package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestDummyMultipartFilePath checks that the example multipart file does not
// take the path of an example service implementation, which is named after
// its service.
func TestDummyMultipartFilePath(t *testing.T) {
	cases := []struct {
		Name     string
		Services []string
		Path     string
	}{
		{Name: "other-service", Services: []string{"catalog"}, Path: "multipart.go"},
		{Name: "service-named-multipart", Services: []string{"multipart"}, Path: "multipart_codecs.go"},
		{Name: "both-names-taken", Services: []string{"multipart", "multipart_codecs"}, Path: "multipart_codecs2.go"},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunHTTPDSL(t, func() {
				for _, name := range c.Services {
					Service(name, func() {
						Method("upload", func() {
							Payload(func() {
								Attribute("name", String)
							})
							HTTP(func() {
								POST("/" + name)
								MultipartRequest()
							})
						})
					})
				}
			})
			f := dummyMultipartFile("example.com/app/gen", root, CreateHTTPServices(root))
			require.NotNil(t, f)
			assert.Equal(t, c.Path, f.Path)
		})
	}
}
