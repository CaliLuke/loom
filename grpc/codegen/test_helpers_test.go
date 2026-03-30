package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/testutil"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

type grpcDSLTestCase struct {
	Name string
	DSL  func()
}

var grpcCodecDSLs = []grpcDSLTestCase{
	{Name: "payload-user-type", DSL: testdata.MessageUserTypeWithNestedUserTypesDSL},
	{Name: "payload-array", DSL: testdata.UnaryRPCNoResultDSL},
	{Name: "payload-map", DSL: testdata.MessageMapDSL},
	{Name: "payload-primitive", DSL: testdata.ServerStreamingRPCDSL},
	{Name: "payload-primitive-with-streaming-payload", DSL: testdata.ClientStreamingRPCWithPayloadDSL},
	{Name: "payload-user-type-with-streaming-payload", DSL: testdata.BidirectionalStreamingRPCWithPayloadDSL},
	{Name: "payload-with-metadata", DSL: testdata.MessageWithMetadataDSL},
	{Name: "payload-with-validate", DSL: testdata.MessageWithValidateDSL},
	{Name: "payload-with-security-attributes", DSL: testdata.MessageWithSecurityAttrsDSL},
}

var grpcResultCodecDSLs = []grpcDSLTestCase{
	{Name: "result-with-views", DSL: testdata.MessageResultTypeWithViewsDSL},
	{Name: "result-with-explicit-view", DSL: testdata.MessageResultTypeWithExplicitViewDSL},
	{Name: "result-array", DSL: testdata.MessageArrayDSL},
	{Name: "result-primitive", DSL: testdata.UnaryRPCNoPayloadDSL},
	{Name: "result-with-metadata", DSL: testdata.MessageWithMetadataDSL},
	{Name: "result-with-validate", DSL: testdata.MessageWithValidateDSL},
	{Name: "result-collection", DSL: testdata.MessageResultTypeCollectionDSL},
	{Name: "server-streaming", DSL: testdata.ServerStreamingUserTypeDSL},
	{Name: "server-streaming-result-with-views", DSL: testdata.ServerStreamingResultWithViewsDSL},
	{Name: "client-streaming", DSL: testdata.ClientStreamingRPCDSL},
	{Name: "bidirectional-streaming", DSL: testdata.BidirectionalStreamingRPCDSL},
}

func grpcCodecCases(prefix string, defs []grpcDSLTestCase) []grpcDSLTestCase {
	cases := make([]grpcDSLTestCase, 0, len(defs))
	for _, def := range defs {
		cases = append(cases, grpcDSLTestCase{
			Name: prefix + def.Name,
			DSL:  def.DSL,
		})
	}
	return cases
}

func assertGRPCSectionGolden(
	t *testing.T,
	cases []grpcDSLTestCase,
	files func(*ServicesData) []*codegen.File,
	sectionName, goldenPrefix string,
) {
	t.Helper()
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := RunGRPCDSL(t, c.DSL)
			services := CreateGRPCServices(root)
			fs := files(services)
			require.Len(t, fs, 2)
			sections := fs[1].Section(sectionName)
			require.NotEmpty(t, sections)
			code := codegen.SectionsCode(t, sections)
			testutil.AssertGo(t, "testdata/golden/"+goldenPrefix+c.Name+".go.golden", code)
		})
	}
}
