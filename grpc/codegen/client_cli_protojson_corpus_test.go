package codegen

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoreflect"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/dynamicpb"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

// TestClientCLIMessageExamplesDecodeWithProtoJSON checks, for every gRPC
// test design, that protojson decodes the example of each request message
// flag into the message that protoc compiles from the generated proto file.
// protojson rejects unknown fields, so the example must use the protocol
// buffer names of the fields and set each oneof by the name of a oneof field.
func TestClientCLIMessageExamplesDecodeWithProtoJSON(t *testing.T) {
	cases := []struct {
		Name string
		DSL  func()
	}{
		{"AliasValidationDSL", testdata.AliasValidationDSL},
		{"AnyErrorDSL", testdata.AnyErrorDSL},
		{"ArrayAliasDSL", testdata.ArrayAliasDSL},
		{"BidirectionalStreamingRPCDSL", testdata.BidirectionalStreamingRPCDSL},
		{"BidirectionalStreamingRPCSameTypeDSL", testdata.BidirectionalStreamingRPCSameTypeDSL},
		{"BidirectionalStreamingRPCWithErrorsDSL", testdata.BidirectionalStreamingRPCWithErrorsDSL},
		{"BidirectionalStreamingRPCWithPayloadDSL", testdata.BidirectionalStreamingRPCWithPayloadDSL},
		{"BlockOneOfFieldDSL", testdata.BlockOneOfFieldDSL},
		{"ClientStreamingNoResultDSL", testdata.ClientStreamingNoResultDSL},
		{"ClientStreamingRPCDSL", testdata.ClientStreamingRPCDSL},
		{"ClientStreamingRPCWithPayloadDSL", testdata.ClientStreamingRPCWithPayloadDSL},
		{"CLIProtoJSONDSL", testdata.CLIProtoJSONDSL},
		{"ConstructorOneOfFieldDSL", testdata.ConstructorOneOfFieldDSL},
		{"CustomMessageNameDSL", testdata.CustomMessageNameDSL},
		{"DefaultFieldsDSL", testdata.DefaultFieldsDSL},
		{"DigitServiceNamesDSL", testdata.DigitServiceNamesDSL},
		{"ElemValidationDSL", testdata.ElemValidationDSL},
		{"InlineObjectFieldsDSL", testdata.InlineObjectFieldsDSL},
		{"InterceptorsDSL", testdata.InterceptorsDSL},
		{"InvalidFirstCharacterNamesDSL", testdata.InvalidFirstCharacterNamesDSL},
		{"MapAliasDSL", testdata.MapAliasDSL},
		{"MessageArrayDSL", testdata.MessageArrayDSL},
		{"MessageMapDSL", testdata.MessageMapDSL},
		{"MessagePrimitiveDSL", testdata.MessagePrimitiveDSL},
		{"MessageResultTypeCollectionDSL", testdata.MessageResultTypeCollectionDSL},
		{"MessageResultTypeWithExplicitViewDSL", testdata.MessageResultTypeWithExplicitViewDSL},
		{"MessageResultTypeWithViewsDSL", testdata.MessageResultTypeWithViewsDSL},
		{"MessageUserTypeWithAliasMessageDSL", testdata.MessageUserTypeWithAliasMessageDSL},
		{"MessageUserTypeWithCollectionDSL", testdata.MessageUserTypeWithCollectionDSL},
		{"MessageUserTypeWithNestedUserTypesDSL", testdata.MessageUserTypeWithNestedUserTypesDSL},
		{"MessageUserTypeWithPrimitivesDSL", testdata.MessageUserTypeWithPrimitivesDSL},
		{"MessageWithMetadataDSL", testdata.MessageWithMetadataDSL},
		{"MessageWithSecurityAttrsDSL", testdata.MessageWithSecurityAttrsDSL},
		{"MessageWithServiceNameDSL", testdata.MessageWithServiceNameDSL},
		{"MessageWithValidateDSL", testdata.MessageWithValidateDSL},
		{"MetadataVarNameCollisionDSL", testdata.MetadataVarNameCollisionDSL},
		{"MethodWithAcronymDSL", testdata.MethodWithAcronymDSL},
		{"MethodWithReservedNameDSL", testdata.MethodWithReservedNameDSL},
		{"MultipleMethodsSameResultCollectionDSL", testdata.MultipleMethodsSameResultCollectionDSL},
		{"NamedOneOfFieldDSL", testdata.NamedOneOfFieldDSL},
		{"NamedUnionFieldReuseDSL", testdata.NamedUnionFieldReuseDSL},
		{"NonASCIIServiceNamesDSL", testdata.NonASCIIServiceNamesDSL},
		{"PayloadWithAliasTypeDSL", testdata.PayloadWithAliasTypeDSL},
		{"PayloadWithCustomTypePackageDSL", testdata.PayloadWithCustomTypePackageDSL},
		{"PayloadWithMixedAttributesDSL", testdata.PayloadWithMixedAttributesDSL},
		{"PayloadWithMultipleUseTypesDSL", testdata.PayloadWithMultipleUseTypesDSL},
		{"PayloadWithNestedTypesDSL", testdata.PayloadWithNestedTypesDSL},
		{"PayloadWithValidationsDSL", testdata.PayloadWithValidationsDSL},
		{"RecursiveArrayAliasDSL", testdata.RecursiveArrayAliasDSL},
		{"RecursiveArrayMessageDSL", testdata.RecursiveArrayMessageDSL},
		{"RecursiveMapMessageDSL", testdata.RecursiveMapMessageDSL},
		{"ResultWithCollectionDSL", testdata.ResultWithCollectionDSL},
		{"ServerStreamingArrayDSL", testdata.ServerStreamingArrayDSL},
		{"ServerStreamingMapDSL", testdata.ServerStreamingMapDSL},
		{"ServerStreamingResultCollectionWithExplicitViewDSL", testdata.ServerStreamingResultCollectionWithExplicitViewDSL},
		{"ServerStreamingResultWithViewsDSL", testdata.ServerStreamingResultWithViewsDSL},
		{"ServerStreamingRPCDSL", testdata.ServerStreamingRPCDSL},
		{"ServerStreamingSharedResultRPCDSL", testdata.ServerStreamingSharedResultRPCDSL},
		{"ServerStreamingUserTypeDSL", testdata.ServerStreamingUserTypeDSL},
		{"ServerStreamingWithCustomErrorsDSL", testdata.ServerStreamingWithCustomErrorsDSL},
		{"ServiceWithPackageDSL", testdata.ServiceWithPackageDSL},
		{"StructFieldNameMetaTypeDSL", testdata.StructFieldNameMetaTypeDSL},
		{"StructMetaTypeDSL", testdata.StructMetaTypeDSL},
		{"UnaryRPCAcronymDSL", testdata.UnaryRPCAcronymDSL},
		{"UnaryRPCNoPayloadDSL", testdata.UnaryRPCNoPayloadDSL},
		{"UnaryRPCNoResultDSL", testdata.UnaryRPCNoResultDSL},
		{"UnaryRPCsDSL", testdata.UnaryRPCsDSL},
		{"UnaryRPCWithErrorsDSL", testdata.UnaryRPCWithErrorsDSL},
		{"UnaryRPCWithOverridingErrorsDSL", testdata.UnaryRPCWithOverridingErrorsDSL},
		{"UnionBranchCollisionDSL", testdata.UnionBranchCollisionDSL},
		{"UnionBranchUnionDSL", testdata.UnionBranchUnionDSL},
		{"UnionMessageBranchNameDSL", testdata.UnionMessageBranchNameDSL},
		{"UnionMessageDSL", testdata.UnionMessageDSL},
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			files := compileProtoDescriptors(t, services)
			for _, svc := range root.API.GRPC.Services {
				sd := services.Get(svc.Name())
				for _, e := range sd.Endpoints {
					flags, _ := buildFlags(e)
					for _, f := range flags {
						if f.Name != "message" {
							continue
						}
						require.NotNil(t, e.Request.PayloadMessage, "%s: no request message", e.Method.Name)
						md := findMessageDescriptor(t, files, e.Request.PayloadMessage.Name)
						msg := dynamicpb.NewMessage(md)
						example := unquoteCLIExample(f)
						assert.NoError(t, protojson.Unmarshal([]byte(example), msg), "%s %s example:\n%s", svc.Name(), e.Method.Name, example)
					}
				}
			}
		})
	}
}

// compileProtoDescriptors compiles the proto files of services with protoc
// and returns their descriptors.
func compileProtoDescriptors(t *testing.T, services *ServicesData) *protoregistry.Files {
	t.Helper()
	dir := t.TempDir()
	protoFiles := ProtoFiles("", services)
	args := make([]string, 0, 5+len(protoFiles))
	args = append(args, "--proto_path", dir, "--include_imports", "--descriptor_set_out", filepath.Join(dir, "descriptors.pb"))
	for _, file := range protoFiles {
		sections := file.AllSections()
		require.GreaterOrEqual(t, len(sections), 2)
		name := filepath.Base(file.Path)
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(sectionCode(t, sections[1:]...)), 0o600))
		args = append(args, name)
	}
	command := exec.Command(expr.DefaultProtoc, args...)
	command.Dir = dir
	output, err := command.CombinedOutput()
	require.NoError(t, err, "%s", output)
	raw, err := os.ReadFile(filepath.Join(dir, "descriptors.pb"))
	require.NoError(t, err)
	var set descriptorpb.FileDescriptorSet
	require.NoError(t, proto.Unmarshal(raw, &set))
	files, err := protodesc.NewFiles(&set)
	require.NoError(t, err)
	return files
}

// findMessageDescriptor returns the descriptor of the top-level message name
// in files.
func findMessageDescriptor(t *testing.T, files *protoregistry.Files, name string) protoreflect.MessageDescriptor {
	t.Helper()
	var res protoreflect.MessageDescriptor
	files.RangeFiles(func(fd protoreflect.FileDescriptor) bool {
		if md := fd.Messages().ByName(protoreflect.Name(name)); md != nil {
			res = md
			return false
		}
		return true
	})
	require.NotNil(t, res, "no message %q", name)
	return res
}
