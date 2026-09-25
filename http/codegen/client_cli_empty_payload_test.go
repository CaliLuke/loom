package codegen

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestClientCLIEmptyObjectPayload asserts that the client CLI builds the
// payload of an object user type without attributes with a payload builder
// that takes no flag, as the server builds it without reading the request.
// The command parser calls the builder and never refers to the service type
// itself, which its package does not import.
func TestClientCLIEmptyObjectPayload(t *testing.T) {
	root := RunHTTPDSL(t, func() {
		var E = Type("E", func() {})
		Service("svc", func() {
			Method("create", func() {
				Payload(E)
				HTTP(func() {
					POST("/")
				})
			})
		})
	})
	services := CreateHTTPServices(root)
	files := ClientCLIFiles("gen", services)
	build := codegen.SectionCode(t, findFileWithSection(t, files, "cli-build-payload").Section("cli-build-payload")[0])
	parse := codegen.SectionCode(t, findFileWithSection(t, files, "parse-endpoint").Section("parse-endpoint")[0])

	assert.Contains(t, build, "func BuildCreatePayload() (*svc.E, error) {\n\tv := &svc.E{}\n\treturn v, nil\n}\n")
	assert.Contains(t, parse, "\t\t\t\tdata, err = svcc.BuildCreatePayload()\n")
	assert.NotContains(t, parse, "var val")
	assert.NotContains(t, parse, "svcCreatePFlag")
}
