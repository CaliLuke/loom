package service

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/stretchr/testify/require"
)

func TestFileResponseServiceSections(t *testing.T) {
	method := &MethodData{
		Name:        "download",
		VarName:     "Download",
		Description: "Download serves a file.",
		MethodPayloadData: MethodPayloadData{
			Payload:    "DownloadPayload",
			PayloadRef: "*DownloadPayload",
		},
		MethodResultData: MethodResultData{
			Result:    "DownloadResult",
			ResultRef: "*DownloadResult",
		},
		MethodTransportData: MethodTransportData{
			FileResponse:       true,
			FileResponseStruct: "DownloadFileResponseData",
			ResponseStruct:     "DownloadResponseData",
			EndpointField:      "DownloadEndpoint",
		},
	}

	serviceCode := codegen.SectionCode(t, serviceDefinitionSection(&Data{
		Name:        "files",
		Description: "Files service.",
		Methods:     []*MethodData{method},
	}))
	require.Contains(t, serviceCode, "Download(context.Context, *DownloadPayload) (res *DownloadResult, file *loomhttp.FileResponse, err error)")

	serverCarrierCode := codegen.SectionCode(t, fileResponseStructSection(&EndpointMethodData{MethodData: method}))
	require.Contains(t, serverCarrierCode, "File *loomhttp.FileResponse")
	require.NotContains(t, serverCarrierCode, "Body io.ReadCloser")
	clientCarrierCode := codegen.SectionCode(t, responseBodyStructSection(&EndpointMethodData{MethodData: method}))
	require.Contains(t, clientCarrierCode, "Body io.ReadCloser")
	require.NotContains(t, clientCarrierCode, "File *loomhttp.FileResponse")

	endpointCode := codegen.SectionCode(t, endpointMethodSection(&EndpointMethodData{
		MethodData:     method,
		ServiceName:    "files",
		ServiceVarName: "files",
	}))
	require.Contains(t, endpointCode, "res, file, err := s.Download(ctx, p)")
	require.Contains(t, endpointCode, "File: file")

	clientCode := codegen.SectionCode(t, methodSection(&EndpointMethodData{
		MethodData:    method,
		ClientVarName: "Client",
		ServiceName:   "files",
	}))
	require.Contains(t, clientCode, "Download(ctx context.Context, p *DownloadPayload) (res *DownloadResult, resp io.ReadCloser, err error)")
	require.Contains(t, clientCode, "o := ires.(*DownloadResponseData)")

	capabilities := DescribeMethodCapabilities(method)
	require.True(t, capabilities.IsFileResponse)
	require.True(t, capabilities.HasResponseStruct)
	require.True(t, capabilities.HasFileResponseStruct)

	exampleCode := codegen.SectionCode(t, exampleEndpointSection(&basicEndpointData{
		MethodData:     method,
		ServiceVarName: "files",
		ResultFullRef:  "*files.DownloadResult",
	}))
	require.Contains(t, exampleCode, "file *loomhttp.FileResponse")
	require.Contains(t, exampleCode, `file = &loomhttp.FileResponse{Name: "download", Content: strings.NewReader("download")}`)
}
