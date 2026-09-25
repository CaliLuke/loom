package codegen

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/cli"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestClientCLIPayloadBuildersDecodeProtoJSON checks that the client CLI
// payload builders decode the JSON of the request message flag with
// protojson, which populates oneof fields, and not with encoding/json/v2,
// which leaves them nil.
func TestClientCLIPayloadBuildersDecodeProtoJSON(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"named-union-field-reuse", testdata.NamedUnionFieldReuseDSL},
		{"union-branch-union", testdata.UnionBranchUnionDSL},
		{"union-branch-collision", testdata.UnionBranchCollisionDSL},
		{"cli-protojson", testdata.CLIProtoJSONDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			code := clientCLIPayloadBuildersCode(t, c.DSL)
			assert.Contains(t, code, `protojson "google.golang.org/protobuf/encoding/protojson"`)
			assert.Contains(t, code, "err = protojson.Unmarshal([]byte(")
			assert.NotContains(t, code, "err = json.Unmarshal(")
			assert.NotContains(t, code, `"encoding/json/v2"`)
		})
	}
}

// TestClientCLIMessageExamplesProtoJSON checks that the example of the request
// message flag is in the protocol buffer JSON mapping: a union field is set
// by the name of its oneof field, a union message by the name of the oneof
// field of the message, and every field by its protocol buffer name.
func TestClientCLIMessageExamplesProtoJSON(t *testing.T) {
	examples := func(t *testing.T, dsl func()) map[string]map[string]any {
		t.Helper()
		res := make(map[string]map[string]any)
		for method, example := range clientCLIMessageExamples(t, dsl) {
			var value map[string]any
			require.NoError(t, json.Unmarshal([]byte(example), &value), "%s: %s", method, example)
			res[method] = value
		}
		return res
	}
	oneKeyOf := func(t *testing.T, value any, keys ...string) (string, any) {
		t.Helper()
		m, ok := value.(map[string]any)
		require.True(t, ok, "expected an object, got %#v", value)
		var found []string
		for _, key := range keys {
			if _, ok := m[key]; ok {
				found = append(found, key)
			}
		}
		require.Len(t, found, 1, "expected exactly one of %v in %v", keys, m)
		return found[0], m[found[0]]
	}
	keysOf := func(value any) []string {
		m, _ := value.(map[string]any)
		keys := make([]string, 0, len(m))
		for key := range m {
			keys = append(keys, key)
		}
		slices.Sort(keys)
		return keys
	}

	t.Run("named union field", func(t *testing.T) {
		ex := examples(t, testdata.NamedUnionFieldReuseDSL)
		echo := ex["echo"]
		assert.NotContains(t, echo, "choice")
		oneKeyOf(t, echo, "leaf", "other")
		oneKeyOf(t, echo["holder"], "leaf", "other")
		assert.Contains(t, echo["holder"], "label")
		for _, holder := range echo["holders"].([]any) {
			oneKeyOf(t, holder, "leaf", "other")
		}
		assert.Len(t, ex["named"], 1)
		oneKeyOf(t, ex["named"], "leaf", "other")
	})

	t.Run("union in union", func(t *testing.T) {
		ex := examples(t, testdata.UnionBranchUnionDSL)
		key, value := oneKeyOf(t, ex["wrap"], "choice", "extra")
		if key == "choice" {
			oneKeyOf(t, value, "leaf", "other")
		}
		echo := ex["echo"]
		for _, absent := range []string{"wrapped", "nested", "block", "field"} {
			assert.NotContains(t, echo, absent)
		}
		key, value = oneKeyOf(t, echo, "choice", "extra")
		if key == "choice" {
			oneKeyOf(t, value, "leaf", "other")
		}
		key, value = oneKeyOf(t, echo, "leaf", "extra_or_other")
		if key == "extra_or_other" {
			oneKeyOf(t, value, "extra", "other")
		}
		key, value = oneKeyOf(t, echo, "picked", "text")
		if key == "picked" {
			oneKeyOf(t, value, "leaf", "other")
		}
		oneKeyOf(t, echo["holder"], "leaf", "other")
		key, value = oneKeyOf(t, echo["holder"], "choice", "extra")
		if key == "choice" {
			oneKeyOf(t, value, "leaf", "other")
		}
	})

	t.Run("protocol buffer names", func(t *testing.T) {
		ex := examples(t, testdata.CLIProtoJSONDSL)
		plain := ex["plain"]
		assert.Equal(t, []string{"anything", "big", "blob", "by_id", "by_name", "ratio", "request_id", "tags"}, keysOf(plain))
		for _, leaf := range plain["by_name"].(map[string]any) {
			assert.Equal(t, []string{"leaf_name", "message_"}, keysOf(leaf))
		}
		for _, tags := range plain["tags"].([]any) {
			assert.Equal(t, []string{"field"}, keysOf(tags))
		}
		send := ex["send"]
		assert.Contains(t, send, "request_id")
		key, value := oneKeyOf(t, send, "leaf", "int")
		if key == "leaf" {
			assert.Equal(t, []string{"leaf_name", "message_"}, keysOf(value))
		}
		for _, choice := range send["choices"].([]any) {
			oneKeyOf(t, choice, "leaf", "int")
		}
		for _, choice := range send["choice_map"].(map[string]any) {
			oneKeyOf(t, choice, "leaf", "int")
		}
		oneKeyOf(t, ex["union"], "leaf", "int")
		assert.Equal(t, []string{"field"}, keysOf(ex["scalar"]))
		for _, leaf := range ex["list"]["field"].([]any) {
			assert.Equal(t, []string{"leaf_name", "message_"}, keysOf(leaf))
		}
	})
}

// TestGeneratedClientCLIDecodesOneofs compiles generated modules with their
// client CLI and builds payloads from CLI JSON that sets top-level union
// fields, named union fields and unions nested in unions. It checks that each
// branch reaches the service payload, that the retired shape that names the
// union attribute is rejected, and that the example JSON of every request
// message flag decodes.
func TestGeneratedClientCLIDecodesOneofs(t *testing.T) {
	cases := []struct {
		Name    string
		Service string
		DSL     func()
		Harness string
	}{
		{"named-union-field-reuse", "reuse", testdata.NamedUnionFieldReuseDSL, cliReuseHarness},
		{"union-branch-union", "nestedunion", testdata.UnionBranchUnionDSL, cliNestedUnionHarness},
		{"union-branch-collision", "collision", testdata.UnionBranchCollisionDSL, cliCollisionHarness},
		{"cli-protojson", "pbjson", testdata.CLIProtoJSONDSL, cliProtoJSONHarness},
		{"recursive-array-alias", "recursivearray", testdata.RecursiveArrayAliasDSL, cliRecursiveHarness},
	}
	source := resolveGRPCLoomSource(t)
	// The design evaluation is global, so render the modules one at a
	// time and only build and test them in parallel.
	dirs := make([]string, len(cases))
	for i, c := range cases {
		modulePath := "example.com/grpccli" + strings.ReplaceAll(c.Name, "-", "")
		root := RunGRPCDSL(t, c.DSL)
		examples := clientCLIMessageExamplesFromRoot(t, root)
		dir := t.TempDir()
		renderGRPCModule(t, dir, modulePath, root, source)
		for _, file := range ClientCLIFiles(modulePath+"/gen", CreateGRPCServices(root)) {
			_, err := file.Render(dir)
			require.NoError(t, err, file.Path)
		}
		methods := make([]string, 0, len(examples))
		for method := range examples {
			methods = append(methods, method)
		}
		slices.Sort(methods)
		var literal strings.Builder
		literal.WriteString("map[string]string{\n")
		for _, method := range methods {
			fmt.Fprintf(&literal, "\t%s: %s,\n", strconv.Quote(method), strconv.Quote(examples[method]))
		}
		literal.WriteString("}")
		testDir := filepath.Join(dir, "internal", "clitest")
		require.NoError(t, os.MkdirAll(testDir, 0o750))
		harness := fmt.Sprintf(c.Harness, modulePath, c.Service, literal.String())
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "cli_test.go"), []byte(harness), 0o600))
		dirs[i] = dir
	}
	for i, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			t.Parallel()
			runGRPCGoCommand(t, dirs[i], "mod", "tidy")
			runGRPCGoCommand(t, dirs[i], "build", "./...")
			runGRPCGoCommand(t, dirs[i], "vet", "./...")
			runGRPCGoCommand(t, dirs[i], "test", "./internal/clitest")
		})
	}
}

// clientCLIPayloadBuildersCode returns the code of the payload builders file
// of the first service of the design.
func clientCLIPayloadBuildersCode(t *testing.T, dsl func()) string {
	t.Helper()
	root := RunGRPCDSL(t, dsl)
	fs := ClientCLIFiles("", CreateGRPCServices(root))
	require.Greater(t, len(fs), 1, "expected at least 2 files")
	var buf bytes.Buffer
	for _, s := range fs[1].AllSections() {
		require.NoError(t, s.Write(&buf))
	}
	return codegen.FormatTestCode(t, buf.String())
}

// clientCLIMessageExamples returns the JSON example of the message flag of
// each method of the design indexed by method name.
func clientCLIMessageExamples(t *testing.T, dsl func()) map[string]string {
	t.Helper()
	return clientCLIMessageExamplesFromRoot(t, RunGRPCDSL(t, dsl))
}

// clientCLIMessageExamplesFromRoot returns the JSON example of the message
// flag of each method of the design root indexed by method name, without the
// shell quotes of a multiline example.
func clientCLIMessageExamplesFromRoot(t *testing.T, root *expr.RootExpr) map[string]string {
	t.Helper()
	services := CreateGRPCServices(root)
	res := make(map[string]string)
	for _, svc := range root.API.GRPC.Services {
		sd := services.Get(svc.Name())
		for _, e := range sd.Endpoints {
			flags, _ := buildFlags(e)
			for _, f := range flags {
				if f.Name != "message" {
					continue
				}
				res[e.Method.Name] = unquoteCLIExample(f)
			}
		}
	}
	require.NotEmpty(t, res)
	return res
}

// unquoteCLIExample returns the JSON of the flag example without the shell
// quotes that wrap a multiline example.
func unquoteCLIExample(f *cli.FlagData) string {
	ex := f.Example
	if strings.HasPrefix(ex, "'") && strings.HasSuffix(ex, "'") {
		ex = strings.ReplaceAll(ex[1:len(ex)-1], `\'`, "'")
	}
	return ex
}

// cliExamplesHarnessHelpers is the shared part of the client CLI harnesses.
// It checks that the example of every message flag decodes.
const cliExamplesHarnessHelpers = `
var examples = %[3]s

func TestExamplesDecode(t *testing.T) {
	require.NotEmpty(t, examples)
	for method, example := range examples {
		build, ok := builders[method]
		require.True(t, ok, "no builder for method %%q", method)
		_, err := build(example)
		require.NoError(t, err, "method %%q example:\n%%s", method, example)
	}
}
`

var cliReuseHarness = `package clitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	reuse "%[1]s/gen/%[2]s"
	"%[1]s/gen/grpc/%[2]s/client"
)

var builders = map[string]func(string) (any, error){
	"echo":  func(s string) (any, error) { return client.BuildEchoPayload(s) },
	"named": func(s string) (any, error) { return client.BuildNamedPayload(s) },
}

func TestTopLevelAndNamedUnionFields(t *testing.T) {
	p, err := client.BuildEchoPayload(` + "`" + `{"id":"i","other":{"count":7},"holder":{"label":"l","leaf":{"name":"n"}},"holders":[{"other":{"count":1}},{"label":"x"}]}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, "i", *p.ID)
	other, ok := p.Choice.AsOther()
	require.True(t, ok)
	require.Equal(t, 7, *other.Count)
	require.Equal(t, "l", *p.Holder.Label)
	leaf, ok := p.Holder.Choice.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "n", *leaf.Name)
	require.Len(t, p.Holders, 2)
	other, ok = p.Holders[0].Choice.AsOther()
	require.True(t, ok)
	require.Equal(t, 1, *other.Count)
	require.Nil(t, p.Holders[1].Choice)

	p, err = client.BuildEchoPayload(` + "`" + `{"leaf":{"name":"camel"},"holder":{"other":{"count":2}}}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, reuse.ChoiceKindLeaf, p.Choice.Kind())
	require.Equal(t, reuse.ChoiceKindOther, p.Holder.Choice.Kind())
}

func TestUnionPayload(t *testing.T) {
	for _, c := range []struct {
		json string
		kind reuse.ChoiceKind
	}{
		{` + "`" + `{"leaf":{"name":"n"}}` + "`" + `, reuse.ChoiceKindLeaf},
		{` + "`" + `{"other":{"count":3}}` + "`" + `, reuse.ChoiceKindOther},
	} {
		p, err := client.BuildNamedPayload(c.json)
		require.NoError(t, err)
		require.NotNil(t, p)
		require.Equal(t, c.kind, p.Kind())
	}
}

func TestUnionAttributeNameRejected(t *testing.T) {
	_, err := client.BuildEchoPayload(` + "`" + `{"choice":{"name":"n"}}` + "`" + `)
	require.ErrorContains(t, err, "invalid JSON for message")
	_, err = client.BuildNamedPayload(` + "`" + `{"field":{"name":"n"}}` + "`" + `)
	require.ErrorContains(t, err, "invalid JSON for message")
	_, err = client.BuildNamedPayload(` + "`" + `{"leaf":{"name":"n"},"other":{"count":1}}` + "`" + `)
	require.ErrorContains(t, err, "invalid JSON for message")
}
` + cliExamplesHarnessHelpers

var cliNestedUnionHarness = `package clitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	nestedunion "%[1]s/gen/%[2]s"
	"%[1]s/gen/grpc/%[2]s/client"
)

var builders = map[string]func(string) (any, error){
	"echo": func(s string) (any, error) { return client.BuildEchoPayload(s) },
	"wrap": func(s string) (any, error) { return client.BuildWrapPayload(s) },
	"pick": func(s string) (any, error) { return client.BuildPickPayload(s) },
}

func TestUnionInUnionPayload(t *testing.T) {
	p, err := client.BuildWrapPayload(` + "`" + `{"choice":{"other":{"count":3}}}` + "`" + `)
	require.NoError(t, err)
	choice, ok := p.AsChoice()
	require.True(t, ok)
	other, ok := choice.AsOther()
	require.True(t, ok)
	require.Equal(t, 3, *other.Count)

	p, err = client.BuildWrapPayload(` + "`" + `{"extra":{"flag":true}}` + "`" + `)
	require.NoError(t, err)
	extra, ok := p.AsExtra()
	require.True(t, ok)
	require.True(t, *extra.Flag)
}

func TestUnionInUnionFields(t *testing.T) {
	p, err := client.BuildEchoPayload(` + "`" + `{
		"id": "i",
		"choice": {"leaf": {"name": "w"}},
		"extraOrOther": {"other": {"count": 5}},
		"picked": {"other": {"count": 6}},
		"holder": {"leaf": {"name": "h"}, "choice": {"other": {"count": 8}}},
		"holders": [{"extra": {"flag": true}}]
	}` + "`" + `)
	require.NoError(t, err)
	wrapped, ok := p.Wrapped.AsChoice()
	require.True(t, ok)
	leaf, ok := wrapped.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "w", *leaf.Name)
	nested, ok := p.Nested.AsExtraOrOther()
	require.True(t, ok)
	other, ok := nested.AsOther()
	require.True(t, ok)
	require.Equal(t, 5, *other.Count)
	picked, ok := p.Block.AsPicked()
	require.True(t, ok)
	other, ok = picked.AsOther()
	require.True(t, ok)
	require.Equal(t, 6, *other.Count)
	leaf, ok = p.Holder.Pick.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "h", *leaf.Name)
	outer, ok := p.Holder.Outer.AsChoice()
	require.True(t, ok)
	other, ok = outer.AsOther()
	require.True(t, ok)
	require.Equal(t, 8, *other.Count)
	require.Len(t, p.Holders, 1)
	extra, ok := p.Holders[0].Outer.AsExtra()
	require.True(t, ok)
	require.True(t, *extra.Flag)

	p, err = client.BuildEchoPayload(` + "`" + `{"extra":{"flag":false},"leaf":{"name":"l"},"text":"t"}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, nestedunion.ChoiceOrExtraKindExtra, p.Wrapped.Kind())
	require.Equal(t, nestedunion.ExtraOrOtherOrLeafKindLeaf, p.Nested.Kind())
	text, ok := p.Block.AsText()
	require.True(t, ok)
	require.Equal(t, nestedunion.BlockText("t"), text)
}

func TestPickPayload(t *testing.T) {
	p, err := client.BuildPickPayload(` + "`" + `{"other":{"count":1}}` + "`" + `)
	require.NoError(t, err)
	other, ok := p.AsOther()
	require.True(t, ok)
	require.Equal(t, 1, *other.Count)
}
` + cliExamplesHarnessHelpers

var cliCollisionHarness = `package clitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/%[2]s/client"
)

var builders = map[string]func(string) (any, error){
	"echo":  func(s string) (any, error) { return client.BuildEchoPayload(s) },
	"grow":  func(s string) (any, error) { return client.BuildGrowPayload(s) },
	"match": func(s string) (any, error) { return client.BuildMatchPayload(s) },
}
` + cliExamplesHarnessHelpers

var cliProtoJSONHarness = `package clitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/%[2]s/client"
)

var builders = map[string]func(string) (any, error){
	"plain":  func(s string) (any, error) { return client.BuildPlainPayload(s) },
	"send":   func(s string) (any, error) { return client.BuildSendPayload(s) },
	"union":  func(s string) (any, error) { return client.BuildUnionPayload(s) },
	"scalar": func(s string) (any, error) { return client.BuildScalarPayload(s) },
	"list":   func(s string) (any, error) { return client.BuildListPayload(s) },
}

func TestUnionsInCollections(t *testing.T) {
	p, err := client.BuildSendPayload(` + "`" + `{"request_id":"r","int":4,"choices":[{"leaf":{"leaf_name":"a","message_":"m"}},{"int":2}],"choice_map":{"k":{"int":9}}}` + "`" + `)
	require.NoError(t, err)
	require.Equal(t, "r", p.RequestID)
	n, ok := p.Pick.AsInt()
	require.True(t, ok)
	require.EqualValues(t, 4, n)
	require.Len(t, p.Choices, 2)
	leaf, ok := p.Choices[0].Choice.AsLeaf()
	require.True(t, ok)
	require.Equal(t, "a", leaf.LeafName)
	require.Equal(t, "m", *leaf.Message)
	n, ok = p.Choices[1].Choice.AsInt()
	require.True(t, ok)
	require.EqualValues(t, 2, n)
	n, ok = p.ChoiceMap["k"].Choice.AsInt()
	require.True(t, ok)
	require.EqualValues(t, 9, n)
}
` + cliExamplesHarnessHelpers

var cliRecursiveHarness = `package clitest

import (
	"testing"

	"github.com/stretchr/testify/require"

	"%[1]s/gen/grpc/%[2]s/client"
)

var builders = map[string]func(string) (any, error){
	"walk": func(s string) (any, error) { return client.BuildWalkPayload(s) },
}
` + cliExamplesHarnessHelpers
