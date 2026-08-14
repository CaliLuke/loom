package codegen

import (
	"net/http"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	. "github.com/CaliLuke/loom/dsl"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResponseContractCaseContentTypesFollowBodyApplicability(t *testing.T) {
	root := RunHTTPDSL(t, responseContractContentTypeDSL)
	services := CreateHTTPServices(root)
	endpoints := services.Get("files").Endpoints

	for _, endpoint := range endpoints {
		require.Len(t, endpoint.ResponseContractCases, 1)
		switch endpoint.Method.Name {
		case "download_default":
			require.Equal(t, []string{"*/*"}, endpoint.ResponseContractCases[0].ContentTypes)
		case "download_pdf":
			require.Equal(t, []string{"application/pdf"}, endpoint.ResponseContractCases[0].ContentTypes)
		case "no_content":
			require.Empty(t, endpoint.ResponseContractCases[0].ContentTypes)
		default:
			t.Fatalf("unexpected endpoint %q", endpoint.Method.Name)
		}
	}
}

func TestServerResponseContractLimitationsBecomeWarnings(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingResultDSL)
	files := ServerFiles("gen", CreateHTTPServices(root))
	file := findFileWithSuffix(t, files, filepath.Join("server", "server.go"))

	require.Equal(t, []string{
		"response contract omitted for StreamingResultService.StreamingResultMethod: streaming: SSE and WebSocket responses require stream-aware contract scenarios",
	}, file.Warnings)
}

func TestBodylessContentTypeHeaderRemainsARequiredContractHeader(t *testing.T) {
	root := RunHTTPDSL(t, responseContractBodylessContentTypeDSL)
	services := CreateHTTPServices(root)
	endpoint := services.Get("widgets").Endpoints[0]
	require.Len(t, endpoint.ResponseContractCases, 1)
	contract := endpoint.ResponseContractCases[0]
	require.Empty(t, contract.ContentTypes)
	require.Equal(t, []string{"Content-Type"}, contract.RequiredHeaders)

	file := findFileWithSuffix(t, ServerFiles("gen", services), filepath.Join("server", "server.go"))
	generated := codegen.SectionCode(t, file.Section("server-response-contract")[0])
	require.Contains(t, generated, `ID: "widgets.no_content.success.204"`)
	require.Contains(t, generated, `RequiredHeaders: []string{"Content-Type"}`)
	require.NotContains(t, generated, "ContentTypes:")

	runtimeContract := loomhttp.ResponseContractCase{
		ID:              contract.ID,
		Kind:            loomhttp.ResponseContractSuccess,
		StatusCode:      contract.StatusCode,
		RequiredHeaders: contract.RequiredHeaders,
	}
	require.NoError(t, loomhttp.ValidateResponseContract(&http.Response{
		StatusCode: http.StatusNoContent,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
	}, runtimeContract))
	require.ErrorContains(t, loomhttp.ValidateResponseContract(&http.Response{
		StatusCode: http.StatusNoContent,
		Header:     make(http.Header),
	}, runtimeContract), `required header "Content-Type" is missing`)
}

func TestResponseContractLimitationWarningsPreserveAnalysisOrder(t *testing.T) {
	warnings := responseContractLimitationWarnings(&transportir.Endpoint{
		Service:    &transportir.Service{Name: "widgets"},
		MethodName: "show",
	}, []transportir.ResponseContractLimitation{
		{Code: transportir.ResponseContractStreaming, Detail: "streaming detail"},
		{Code: transportir.ResponseContractMultipart, Detail: "multipart detail"},
	})

	require.Equal(t, []string{
		"response contract omitted for widgets.show: streaming: streaming detail",
		"response contract omitted for widgets.show: multipart: multipart detail",
	}, warnings)
}

func responseContractContentTypeDSL() {
	Service("files", func() {
		Method("download_default", func() {
			Result(Empty)
			HTTP(func() {
				GET("/downloads/default")
				FileResponse()
			})
		})
		Method("download_pdf", func() {
			Result(Empty)
			HTTP(func() {
				GET("/downloads/pdf")
				FileResponse()
				Response(func() {
					ContentType("application/pdf")
				})
			})
		})
		Method("no_content", func() {
			Result(Empty)
			HTTP(func() {
				GET("/downloads/empty")
				Response(StatusNoContent)
			})
		})
	})
}

func responseContractBodylessContentTypeDSL() {
	Service("widgets", func() {
		Method("no_content", func() {
			Result(func() {
				Attribute("content_type", String)
				Required("content_type")
			})
			HTTP(func() {
				GET("/widgets")
				Response(StatusNoContent, func() {
					Header("content_type:Content-Type")
				})
			})
		})
	})
}
