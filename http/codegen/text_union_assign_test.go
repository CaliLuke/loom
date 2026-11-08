package codegen

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

// TestTextUnionAssignClassification requires every generated text-branch
// assignment to report a failure of its closure as an ordinary candidate
// mismatch, (false, nil), because the public TextBranch runtime returns Assign
// errors unchanged. Admission limits that closure to conversion and validation
// of supplied text, so a generated Assign must not return its raw error.
func TestTextUnionAssignClassification(t *testing.T) {
	root := RunHTTPDSL(t, textUnionAssignDSL)
	dir := t.TempDir()
	renderHTTPModuleFiles(t, dir, "example.com/picks/gen", root)
	tails := make([][]string, 0, 3)
	for _, name := range []string{"server/encode_decode.go", "client/encode_decode.go", "client/cli.go"} {
		path := filepath.Join(dir, "gen", "http", "picks", name)
		data, err := os.ReadFile(path)
		require.NoError(t, err)
		tails = append(tails, regexp.MustCompile(`Assign: func\(raw loomhttp\.ParamValue\) \(bool, error\) \{[\s\S]*?\}\(\); err != nil \{\s*return ([^\n]+)\n`).FindAllStringSubmatch(string(data), -1)...)
		require.Empty(t, returnArityMismatches(t, path), name)
	}
	require.NotEmpty(t, tails, "the union is matched by generated text-branch assignments")
	for _, tail := range tails {
		require.NotEqual(t, "false, err", tail[1], "a supplied-value mismatch must not become a fatal assignment error")
	}
}

func textUnionAssignDSL() {
	long := Type("Long", func() {
		Attribute("name", String, func() {
			MinLength(5)
		})
		Attribute("tag", String, func() {
			Default("t")
		})
		Required("name")
	})
	short := Type("Short", func() {
		Attribute("name", String, func() {
			MaxLength(4)
		})
		Required("name")
	})
	pick := Type("Pick", OneOf(func() {
		Untagged()
		Attribute("long", long)
		Attribute("short", short)
	}))
	Service("picks", func() {
		Method("pick", func() {
			Payload(func() {
				Attribute("pick", pick)
			})
			HTTP(func() {
				GET("/pick")
				Param("pick")
				Response(StatusNoContent)
			})
		})
	})
}
