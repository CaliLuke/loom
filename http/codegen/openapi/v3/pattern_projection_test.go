package openapiv3_test

import (
	"path/filepath"
	"testing"

	. "github.com/CaliLuke/loom/dsl"
	httpgen "github.com/CaliLuke/loom/http/codegen"
	openapiv3 "github.com/CaliLuke/loom/http/codegen/openapi/v3"
	"github.com/stretchr/testify/require"
)

func TestFilesPatternProjection(t *testing.T) {
	for _, target := range []string{"3.1", "3.2"} {
		for _, pattern := range []string{`\Afoo\z`, `^\s$`, `^.$`} {
			t.Run(target+"/"+pattern, func(t *testing.T) {
				root := httpgen.RunHTTPDSL(t, func() {
					value := Type("Value", String, func() {
						Pattern(pattern)
					})
					API("patterns", func() {
						Meta("openapi:version", target)
					})
					Service("svc", func() {
						Method("post", func() {
							NoSecurity()
							Payload(func() {
								Attribute("value", value)
							})
							HTTP(func() {
								POST("/")
							})
						})
					})
				})
				files, err := openapiv3.Files(root)
				if pattern == `^.$` {
					require.ErrorContains(t, err, "cannot preserve Go validation")
					require.Empty(t, files)
					return
				}
				require.NoError(t, err)
				for _, file := range files {
					if filepath.Ext(file.Path) != ".json" {
						continue
					}
					buffer := renderSection(t, file.AllSections()[0])
					raw := buffer.Bytes()
					require.NotContains(t, string(raw), `\\A`)
					require.NotContains(t, string(raw), `\\s`)
					if pattern == `\Afoo\z` {
						require.Contains(t, string(raw), `"pattern": "^foo$"`)
					}
					if pattern == `^\s$` {
						require.Contains(t, string(raw), `"pattern": "^[\\u0009-\\u000a\\u000c-\\u000d ]$"`)
					}
				}
			})
		}
	}
}
