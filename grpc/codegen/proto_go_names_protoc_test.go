package codegen

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/compiler/protogen"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/pluginpb"

	"github.com/CaliLuke/loom/expr"
)

func TestProtoGoNamesMatchProtoc(t *testing.T) {
	field := func(name string) *expr.NamedAttributeExpr {
		return &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: expr.String}}
	}
	oneof := func(name string, branches ...string) *expr.NamedAttributeExpr {
		values := make([]*expr.NamedAttributeExpr, len(branches))
		for i, name := range branches {
			values[i] = field(name)
		}
		return &expr.NamedAttributeExpr{Name: name, Attribute: &expr.AttributeExpr{Type: &expr.Union{TypeName: name, Values: values}}}
	}
	for _, tc := range []struct {
		name     string
		fields   expr.Object
		required []string
	}{
		{"methods", expr.Object{field("reset"), field("descriptor"), field("string"), field("proto_message"), field("marshal"), field("unmarshal"), field("extension_range_array"), field("extension_map")}, nil},
		{"reflection method", expr.Object{field("proto_reflect"), field("proto_reflect_field")}, nil},
		{"reflection oneof", expr.Object{oneof("proto_reflect", "code", "count")}, nil},
		{"getters", expr.Object{field("label"), field("get_label"), field("get_get_label")}, nil},
		{"reversed getters", expr.Object{field("get_get_label"), field("get_label"), field("label")}, nil},
		{"initialisms", expr.Object{field("OAuth2Token"), field("api_id"), field("x1ID")}, nil},
		{"synthetic optional oneof", expr.Object{field("name"), field("x_name")}, nil},
		{"required has no synthetic oneof", expr.Object{field("name"), field("x_name")}, []string{"name"}},
		{"oneof getter before", expr.Object{field("get_pick"), oneof("pick", "code", "count")}, nil},
		{"oneof getter after", expr.Object{oneof("pick", "code", "count"), field("get_pick")}, nil},
		{"branch getter collision", expr.Object{field("label"), oneof("pick", "get_label", "reset")}, nil},
		{"reserved oneof", expr.Object{oneof("reset", "descriptor", "proto_message")}, nil},
		{"multiple oneofs", expr.Object{field("get_pick"), oneof("pick", "code", "count"), oneof("next", "get_code", "get_next"), field("get_next")}, nil},
		{"map entry wrapper", expr.Object{{Name: "foo", Attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}}, oneof("pick", "foo_entry", "count")}, nil},
		{"map entry digits", expr.Object{{Name: "foo_2", Attribute: &expr.AttributeExpr{Type: &expr.Map{KeyType: &expr.AttributeExpr{Type: expr.String}, ElemType: &expr.AttributeExpr{Type: expr.String}}}}, oneof("pick", "foo2_entry", "count")}, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			att := &expr.AttributeExpr{Type: &tc.fields, Validation: &expr.ValidationExpr{Required: tc.required}}
			names := newProtoMessageNames(att)
			message := protoNamesOracle(t, att, names)
			fields := make(map[string]*protogen.Field)
			for _, field := range message.Fields {
				fields[string(field.Desc.Name())] = field
			}
			oneofs := make(map[string]*protogen.Oneof)
			for _, oneof := range message.Oneofs {
				oneofs[string(oneof.Desc.Name())] = oneof
			}
			for _, nat := range tc.fields {
				wire := names.field(nat.Name)
				if !expr.IsUnion(nat.Attribute.Type) {
					require.Equal(t, fields[wire].GoName, names.goField(nat.Name), wire)
					continue
				}
				require.Equal(t, oneofs[wire].GoName, names.goField(nat.Name), wire)
				selectors, wrappers := names.goBranches(nat.Name), names.goBranchTypes(nat.Name)
				for i, branch := range names.oneofFields(nat.Name) {
					require.Equal(t, fields[branch].GoName, selectors[i], branch)
					require.Equal(t, strings.TrimPrefix(fields[branch].GoIdent.GoName, "Message_"), wrappers[i], branch)
				}
			}
		})
	}
}

// protoNamesOracle asks protoc to construct descriptors (including synthetic
// oneofs and map entries), then uses the actual Go plugin's naming API.
func protoNamesOracle(t *testing.T, att *expr.AttributeExpr, names *protoMessageNames) *protogen.Message {
	t.Helper()
	var source strings.Builder
	source.WriteString("syntax = \"proto3\";\noption go_package = \"example.com/names\";\nmessage Message {\n")
	tag := 1
	for _, nat := range *expr.AsObject(att.Type) {
		wire := names.field(nat.Name)
		if union := expr.AsUnion(nat.Attribute.Type); union != nil {
			fmt.Fprintf(&source, "oneof %s {\n", wire)
			for _, branch := range names.oneofFields(nat.Name) {
				fmt.Fprintf(&source, "string %s = %d;\n", branch, tag)
				tag++
			}
			source.WriteString("}\n")
			continue
		}
		typ := protoBufOptionalField(nat) + "string"
		if expr.IsMap(nat.Attribute.Type) {
			typ = "map<string, string>"
		}
		fmt.Fprintf(&source, "%s %s = %d;\n", typ, wire, tag)
		tag++
	}
	source.WriteString("}\n")
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "names.proto"), []byte(source.String()), 0o600))
	ctx, cancel := context.WithTimeout(t.Context(), 30*time.Second)
	defer cancel()
	args := append(append([]string(nil), defaultProtocCmd[1:]...), "--proto_path="+dir, "--descriptor_set_out="+filepath.Join(dir, "names.pb"), "names.proto")
	cmd := exec.CommandContext(ctx, defaultProtocCmd[0], args...)
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "%s", out)
	data, err := os.ReadFile(filepath.Join(dir, "names.pb"))
	require.NoError(t, err)
	var descriptors descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(data, &descriptors))
	plugin, err := (protogen.Options{}).New(&pluginpb.CodeGeneratorRequest{ProtoFile: descriptors.File, FileToGenerate: []string{"names.proto"}})
	require.NoError(t, err)
	return plugin.Files[0].Messages[0]
}
