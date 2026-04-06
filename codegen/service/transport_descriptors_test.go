package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/service/testdata"
	dsl "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestBuildPayloadDescriptorUsesResolvedPackage(t *testing.T) {
	root := codegen.RunDSL(t, testdata.PkgPathDSL)
	services := NewServicesData(root)
	svc := services.Get("PkgPathMethod")
	require.NotNil(t, svc)
	method := svc.Method("A")
	require.NotNil(t, method)

	desc := BuildPayloadDescriptor(svc, method, root.Services[0].Methods[0].Payload)
	assert.Equal(t, "foo", desc.Package)
	assert.Equal(t, "foo.Foo", desc.Name)
	assert.Equal(t, "*foo.Foo", desc.Ref)
}

func TestBuildResultDescriptorUsesViewedResultProjection(t *testing.T) {
	root := codegen.RunDSL(t, testdata.MultipleMethodsResultMultipleViewsDSL)
	services := NewServicesData(root)
	svc := services.Get("MultipleMethodsResultMultipleViews")
	require.NotNil(t, svc)
	method := svc.Method("A")
	require.NotNil(t, method)
	require.NotNil(t, method.ViewedResult)

	desc := BuildResultDescriptor(svc, method, root.Services[0].Methods[0].Result)
	require.True(t, desc.UsesViewedResult)
	assert.Equal(t, method.ViewedResult.FullRef, desc.ViewedRef)
	assert.Equal(t, svc.ViewsPkg, desc.Effective.Package)
	assert.Equal(t, method.ViewedResult.FullName, desc.Effective.Name)
	assert.Equal(t, method.ViewedResult.FullRef, desc.Effective.Ref)
	projected := expr.AsObject(method.ViewedResult.Type.Attribute().Type).Attribute("projected")
	require.NotNil(t, projected)
	assert.Same(t, projected, desc.Effective.Attribute)
	assert.Equal(t, expr.DefaultView, desc.View)
}

func TestBuildResultDescriptorPreservesDeclaredResultView(t *testing.T) {
	root := codegen.RunDSL(t, testdata.WithExplicitAndDefaultViewsDSL)
	services := NewServicesData(root)
	svc := services.Get("WithExplicitAndDefaultViews")
	require.NotNil(t, svc)
	method := svc.Method("A")
	require.NotNil(t, method)

	desc := BuildResultDescriptor(svc, method, root.Services[0].Methods[0].Result)
	assert.True(t, desc.UsesViewedResult)
	assert.Equal(t, expr.DefaultView, desc.View)
}

func TestDescribeStream(t *testing.T) {
	cases := []struct {
		name          string
		dsl           func()
		serviceName   string
		methodName    string
		wantKind      expr.StreamKind
		wantStreaming bool
		wantPayload   bool
		wantResult    bool
		wantClient    bool
		wantServer    bool
		wantBidi      bool
	}{
		{
			name:          "server stream",
			dsl:           testdata.StreamingResultWithViewsMethodDSL,
			serviceName:   "StreamingResultWithViewsService",
			methodName:    "StreamingResultWithViewsMethod",
			wantKind:      expr.ServerStreamKind,
			wantStreaming: true,
			wantPayload:   false,
			wantResult:    true,
			wantClient:    false,
			wantServer:    true,
			wantBidi:      false,
		},
		{
			name:          "bidirectional stream",
			dsl:           testdata.BidirectionalStreamingMethodDSL,
			serviceName:   "BidirectionalStreamingService",
			methodName:    "BidirectionalStreamingMethod",
			wantKind:      expr.BidirectionalStreamKind,
			wantStreaming: true,
			wantPayload:   true,
			wantResult:    true,
			wantClient:    true,
			wantServer:    true,
			wantBidi:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, tc.dsl)
			services := NewServicesData(root)
			svc := services.Get(tc.serviceName)
			require.NotNil(t, svc)
			method := svc.Method(tc.methodName)
			require.NotNil(t, method)

			desc := DescribeStream(method)
			assert.Equal(t, tc.wantKind, desc.Kind)
			assert.Equal(t, tc.wantStreaming, desc.IsStreaming)
			assert.Equal(t, tc.wantPayload, desc.HasPayload)
			assert.Equal(t, tc.wantResult, desc.HasResult)
			assert.Equal(t, tc.wantClient, desc.IsClient)
			assert.Equal(t, tc.wantServer, desc.IsServer)
			assert.Equal(t, tc.wantBidi, desc.IsBidirectional)
		})
	}
}

func TestBuildErrorDescriptorUsesResolvedPackage(t *testing.T) {
	root := codegen.RunDSL(t, func() {
		var customErr = dsl.Type("CustomErr", func() {
			dsl.Attribute("message", dsl.String)
			dsl.Meta("struct:pkg:path", "errs")
		})
		dsl.Service("ErrorPkgService", func() {
			dsl.Method("Fail", func() {
				dsl.Error("boom", customErr)
			})
		})
	})
	services := NewServicesData(root)
	svc := services.Get("ErrorPkgService")
	require.NotNil(t, svc)
	method := svc.Method("Fail")
	require.NotNil(t, method)
	require.NotEmpty(t, root.Services[0].Methods[0].Errors)

	desc := BuildErrorDescriptor(svc, method, "boom", root.Services[0].Methods[0].Errors[0].AttributeExpr)
	assert.Equal(t, "boom", desc.Name)
	assert.Equal(t, "errs", desc.Type.Package)
	assert.Equal(t, "*errs.CustomErr", desc.Type.Ref)
}

func TestBuildStreamDescriptor(t *testing.T) {
	root := codegen.RunDSL(t, testdata.BidirectionalStreamingResultWithViewsMethodDSL)
	services := NewServicesData(root)
	svc := services.Get("BidirectionalStreamingResultWithViewsService")
	require.NotNil(t, svc)
	method := svc.Method("BidirectionalStreamingResultWithViewsMethod")
	require.NotNil(t, method)
	exprMethod := root.Services[0].Methods[0]

	desc := BuildStreamDescriptor(svc, method, exprMethod.StreamingPayload, exprMethod.StreamingResult)
	assert.True(t, desc.IsBidirectional)
	assert.True(t, desc.HasPayload)
	assert.True(t, desc.HasResult)
	assert.Equal(t, "*bidirectionalstreamingresultwithviewsservice.APayload", desc.Payload.Ref)
	assert.True(t, desc.Result.UsesViewedResult)
	assert.Equal(t, method.ViewedResult.FullRef, desc.Result.ViewedRef)
}

func TestDescribeMethodCapabilities(t *testing.T) {
	t.Run("mixed results", func(t *testing.T) {
		root := codegen.RunDSL(t, testdata.MixedResultsEndpointDSL)
		services := NewServicesData(root)
		svc := services.Get("MixedResultsEndpoint")
		require.NotNil(t, svc)
		method := svc.Method("MixedResultsMethod")
		require.NotNil(t, method)

		desc := DescribeMethodCapabilities(method)
		assert.True(t, desc.HasPayload)
		assert.True(t, desc.HasResult)
		assert.True(t, desc.HasStreamingResult)
		assert.True(t, desc.HasMixedResults)
		assert.True(t, desc.Stream.IsServer)
	})

	t.Run("request and response wrappers", func(t *testing.T) {
		root := codegen.RunDSL(t, func() {
			dsl.Service("WrapperCaps", func() {
				dsl.Method("Upload", func() {
					dsl.Payload(func() {
						dsl.Attribute("name", dsl.String)
					})
					dsl.Result(func() {
						dsl.Attribute("etag", dsl.String)
					})
					dsl.HTTP(func() {
						dsl.POST("/upload")
						dsl.Header("name")
						dsl.SkipRequestBodyEncodeDecode()
						dsl.SkipResponseBodyEncodeDecode()
						dsl.Response(func() {
							dsl.Header("etag")
						})
					})
				})
			})
		})
		services := NewServicesData(root)
		svc := services.Get("WrapperCaps")
		require.NotNil(t, svc)
		method := svc.Method("Upload")
		require.NotNil(t, method)

		desc := DescribeMethodCapabilities(method)
		assert.True(t, desc.HasRequestStruct)
		assert.True(t, desc.HasResponseStruct)
	})
}
