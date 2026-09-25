package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
)

// TestRecursiveUnionResult checks designs where a named union used directly
// as a method payload or result is also held by a recursive type through a
// field. The message that wraps the direct union must not replace the union
// held by the field, which remains a oneof of the message of the type, so
// the designs generate, protoc accepts them and the generated modules
// compile and vet.
func TestRecursiveUnionResult(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"union result", func() { recursiveUnionDSL(false, true) }},
		{"union payload", func() { recursiveUnionDSL(true, false) }},
		{"union payload and result", func() { recursiveUnionDSL(true, true) }},
		{"streaming union result", recursiveUnionStreamingDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code := protoFileCode(t, c.DSL)
			fpath := codegen.CreateTempFile(t, code)
			require.NoError(t, protoc(defaultProtocCmd, fpath, nil), "compile proto file %q", fpath)
			dir := t.TempDir()
			renderGRPCModule(t, dir, "example.com/recursiveunion", RunGRPCDSL(t, c.DSL), resolveGRPCLoomSource(t))
			runGRPCGoCommand(t, dir, "mod", "tidy")
			runGRPCGoCommand(t, dir, "vet", "./...")
		})
	}
}

// recursiveUnionDSL declares the recursive type Node whose child field holds
// the named union U of Node and Leaf. The method uses Node or U as payload
// and result as unionPayload and unionResult select.
func recursiveUnionDSL(unionPayload, unionResult bool) {
	leaf := Type("Leaf", func() {
		Field(1, "name", String)
	})
	node := Type("Node", func() {
		Field(1, "label", String)
		Field(2, "child", "U")
	})
	u := Type("U", OneOf(node, leaf))
	Service("svc", func() {
		Method("m", func() {
			if unionPayload {
				Payload(u)
			} else {
				Payload(node)
			}
			if unionResult {
				Result(u)
			} else {
				Result(node)
			}
			GRPC(func() {})
		})
	})
}

func recursiveUnionStreamingDSL() {
	leaf := Type("Leaf", func() {
		Field(1, "name", String)
	})
	node := Type("Node", func() {
		Field(1, "label", String)
		Field(2, "child", "U")
	})
	u := Type("U", OneOf(node, leaf))
	Service("svc", func() {
		Method("m", func() {
			Payload(node)
			StreamingResult(u)
			GRPC(func() {})
		})
	})
}
