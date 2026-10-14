package codegen

import (
	"context"
	"os"
	"runtime/debug"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/testprocess"
)

type importPluginType struct {
	*expr.UserTypeExpr
}

func TestGatherAttributeImportsRecursive(t *testing.T) {
	if os.Getenv("LOOM_TEST_RECURSIVE_IMPORT_CHILD") == "1" {
		debug.SetMaxStack(256 << 10)
		node := &expr.UserTypeExpr{TypeName: "Node", AttributeExpr: &expr.AttributeExpr{Meta: expr.MetaExpr{"struct:pkg:path": {"nodes"}}}}
		node.Type = &expr.Object{
			{Name: "next", Attribute: &expr.AttributeExpr{Type: node}},
			{Name: "value", Attribute: importMetaAttribute("value")},
		}
		require.Equal(t, []*ImportSpec{
			{Name: "nodes", Path: "example.com/gen/nodes"},
			{Path: "example.com/value"},
		}, GatherAttributeImports("example.com/gen", &expr.AttributeExpr{Type: node}))
		require.Equal(t, []*ImportSpec{{Path: "example.com/value"}}, GetMetaTypeImports(&expr.AttributeExpr{Type: node}))
		return
	}
	executable, err := os.Executable()
	require.NoError(t, err)
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	command := testprocess.CommandContext(ctx, executable, "-test.run=^TestGatherAttributeImportsRecursive$", "-test.count=1")
	command.Env = append(os.Environ(), "LOOM_TEST_RECURSIVE_IMPORT_CHILD=1")
	output, err := command.CombinedOutput()
	require.NoError(t, err, "bounded recursive import child failed: %s", output[:min(len(output), 4096)])
}

func TestGatherAttributeImportsGraph(t *testing.T) {
	shared := &expr.UserTypeExpr{
		TypeName: "Shared",
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.String,
			Meta: expr.MetaExpr{"struct:pkg:path": {"shared"}, "struct:field:type": {"custom.Shared", "example.com/shared"}},
		},
	}
	occurrence := func(name string) *expr.AttributeExpr {
		attribute := importMetaAttribute(name)
		attribute.Type = shared
		return attribute
	}
	for _, tc := range []struct {
		name      string
		attribute *expr.AttributeExpr
		expected  []*ImportSpec
	}{
		{name: "nil"},
		{name: "native", attribute: &expr.AttributeExpr{Type: expr.String}},
		{name: "metadata only", attribute: &expr.AttributeExpr{Meta: expr.MetaExpr{"struct:field:type": {"custom.Value", "example.com/metadata"}}}, expected: []*ImportSpec{{Path: "example.com/metadata"}}},
		{
			name: "shared occurrences",
			attribute: &expr.AttributeExpr{Type: &expr.Object{
				{Name: "first", Attribute: occurrence("first")},
				{Name: "second", Attribute: occurrence("second")},
			}},
			expected: []*ImportSpec{
				{Path: "example.com/first"},
				{Name: "shared", Path: "example.com/gen/shared"},
				{Path: "example.com/second"},
				{Path: "example.com/shared"},
			},
		},
		{
			name: "array map and union",
			attribute: &expr.AttributeExpr{Type: &expr.Array{ElemType: &expr.AttributeExpr{Type: &expr.Map{
				KeyType: importMetaAttribute("key"),
				ElemType: &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
					{Name: "named", Attribute: &expr.AttributeExpr{Type: shared}},
					{Name: "native", Attribute: importMetaAttribute("variant")},
				}}},
			}}}},
			expected: []*ImportSpec{
				{Name: "shared", Path: "example.com/gen/shared"},
				{Path: "example.com/key"},
				{Path: "example.com/shared"},
				{Path: "example.com/variant"},
			},
		},
		{
			name:      "plugin user type",
			attribute: &expr.AttributeExpr{Type: &importPluginType{shared}},
			expected: []*ImportSpec{
				{Name: "shared", Path: "example.com/gen/shared"},
				{Path: "example.com/shared"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.expected, GatherAttributeImports("example.com/gen", tc.attribute))
			var custom []*ImportSpec
			for _, imported := range tc.expected {
				if imported.Name == "" {
					custom = append(custom, imported)
				}
			}
			require.Equal(t, custom, append([]*ImportSpec(nil), GetMetaTypeImports(tc.attribute)...))
		})
	}
}

// TestWalkResultTypeIdentityForImports preserves the walker's existing identity
// for a result declaration and references to its underlying named type.
func TestWalkResultTypeIdentityForImports(t *testing.T) {
	body := &expr.AttributeExpr{}
	named := &expr.UserTypeExpr{TypeName: "ResultNode", AttributeExpr: body}
	result := &expr.ResultTypeExpr{UserTypeExpr: named, Identifier: "application/vnd.result-node"}
	body.Type = &expr.Object{
		{Name: "next", Attribute: &expr.AttributeExpr{Type: named}},
	}
	visits := 0
	require.NoError(t, Walk(&expr.AttributeExpr{Type: result}, func(attribute *expr.AttributeExpr) error {
		if attribute == body {
			visits++
		}
		return nil
	}))
	require.Equal(t, 1, visits)
}

func importMetaAttribute(name string) *expr.AttributeExpr {
	return &expr.AttributeExpr{Type: expr.String, Meta: expr.MetaExpr{"struct:field:type": {"custom.Value", "example.com/" + name}}}
}

func TestGetMetaTypeImportsUnionAndAliases(t *testing.T) {
	first := importMetaAttribute("shared")
	first.Meta["struct:field:type"] = []string{"first.Value", "example.com/shared", "first"}
	second := importMetaAttribute("shared")
	second.Meta["struct:field:type"] = []string{"second.Value", "example.com/shared", "second"}
	attribute := &expr.AttributeExpr{Type: &expr.Union{TypeName: "Choice", Values: []*expr.NamedAttributeExpr{
		{Name: "first", Attribute: first},
		{Name: "second", Attribute: second},
		{Name: "duplicate", Attribute: first},
	}}}
	require.Equal(t, []*ImportSpec{
		{Name: "first", Path: "example.com/shared"},
		{Name: "second", Path: "example.com/shared"},
	}, GetMetaTypeImports(attribute))
}
