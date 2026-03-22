package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/grpc/codegen/testdata"
)

func TestAnalyzeCollectsMessagesAndProtoImportsWithoutDuplicates(t *testing.T) {
	root := RunGRPCDSL(t, grpcProtoImportDedupDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("ProtoImportDedup")
	require.NotNil(t, svc)

	require.Len(t, svc.ProtoImports, 1)
	require.Equal(t, "example/common.proto", svc.ProtoImports[0])
	require.Equal(t, 1, countMessageByName(svc.Messages, "ExternalPayload"))
}

func TestAnalyzeBuildsRequestCLIArgsFromMessageAndMetadata(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MessageWithMetadataDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("ServiceMessageWithMetadata")
	require.NotNil(t, svc)

	endpoint := svc.Endpoint("MethodMessageWithMetadata")
	require.NotNil(t, endpoint)
	require.Len(t, endpoint.Request.CLIArgs, 2)
	require.Equal(t, "message", endpoint.Request.CLIArgs[0].Name)
	require.Equal(t, "inMetadata", endpoint.Request.CLIArgs[1].Name)
}

func TestAnalyzeBuildsResponseMetadataData(t *testing.T) {
	root := RunGRPCDSL(t, testdata.MessageWithMetadataDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("ServiceMessageWithMetadata")
	require.NotNil(t, svc)

	endpoint := svc.Endpoint("MethodMessageWithMetadata")
	require.NotNil(t, endpoint)
	require.Len(t, endpoint.Response.Headers, 1)
	require.Len(t, endpoint.Response.Trailers, 1)
	require.Equal(t, "Location", endpoint.Response.Headers[0].Name)
	require.Equal(t, "InTrailer", endpoint.Response.Trailers[0].Name)
}

func TestAnalyzePartitionsSecuritySchemesBetweenMessageAndMetadata(t *testing.T) {
	root := RunGRPCDSL(t, grpcSecurityPartitionDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("SecurityPartition")
	require.NotNil(t, svc)

	endpoint := svc.Endpoint("Secure")
	require.NotNil(t, endpoint)
	require.Len(t, endpoint.MessageSchemes, 1)
	require.Len(t, endpoint.MetadataSchemes, 1)
	require.Equal(t, "oauth2", endpoint.MessageSchemes[0].SchemeName)
	require.Equal(t, "jwt", endpoint.MetadataSchemes[0].SchemeName)
}

func TestAnalyzeAttachesStreamDataToStreamingEndpoints(t *testing.T) {
	root := RunGRPCDSL(t, testdata.BidirectionalStreamingRPCWithPayloadDSL)
	services := CreateGRPCServices(root)
	svc := services.Get("ServiceBidirectionalStreamingRPCWithPayload")
	require.NotNil(t, svc)

	endpoint := svc.Endpoint("MethodBidirectionalStreamingRPCWithPayload")
	require.NotNil(t, endpoint)
	require.NotNil(t, endpoint.ServerStream)
	require.NotNil(t, endpoint.ClientStream)
	require.Equal(t, endpoint, endpoint.ServerStream.Endpoint)
	require.Equal(t, endpoint, endpoint.ClientStream.Endpoint)
	require.NotEmpty(t, endpoint.ServerStream.VarName)
	require.NotEmpty(t, endpoint.ClientStream.VarName)
}

func countMessageByName(messages []*service.UserTypeData, name string) int {
	var count int
	for _, msg := range messages {
		if msg.Name == name {
			count++
		}
	}
	return count
}

func grpcProtoImportDedupDSL() {
	var externalPayload = dsl.Type("ExternalPayload", func() {
		dsl.Attribute("wrapped", dsl.String, func() {
			dsl.Meta("struct:field:proto", "wrapped", "example/common.proto", "ExternalPayload", "example/pb")
		})
	})

	dsl.Service("ProtoImportDedup", func() {
		dsl.Method("Show", func() {
			dsl.Payload(func() {
				dsl.Field(1, "first", externalPayload)
				dsl.Field(2, "second", externalPayload)
			})
			dsl.Result(func() {
				dsl.Field(1, "result", externalPayload)
			})
			dsl.GRPC(func() {})
		})
	})
}

func grpcSecurityPartitionDSL() {
	var jwtAuth = dsl.JWTSecurity("jwt")
	var oauth2Auth = dsl.OAuth2Security("oauth2")

	dsl.Service("SecurityPartition", func() {
		dsl.Method("Secure", func() {
			dsl.Security(jwtAuth, oauth2Auth)
			dsl.Payload(func() {
				dsl.TokenField(1, "token", dsl.String)
				dsl.AccessTokenField(2, "oauth_token", dsl.String)
			})
			dsl.GRPC(func() {
				dsl.Metadata(func() {
					dsl.Attribute("token")
				})
				dsl.Message(func() {
					dsl.Attribute("oauth_token")
				})
			})
		})
	})
}
