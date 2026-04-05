package codegen

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/codegen/example"
	ctestdata "github.com/CaliLuke/loom/codegen/example/testdata"
	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestConvertedServerRenderSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerFileServerDSL)
	services := CreateHTTPServices(root)
	files := ServerFiles("gen", services)
	require.NotEmpty(t, files)

	var serverFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("server", "server.go")) {
			serverFile = f
			break
		}
	}
	require.NotNil(t, serverFile)

	serverStruct := codegen.SectionCode(t, serverFile.Section("server-struct")[0])
	require.Contains(t, serverStruct, "type Server struct {")
	require.Contains(t, serverStruct, "[]*MountPoint")

	mountPoint := codegen.SectionCode(t, serverFile.Section("server-mountpoint")[0])
	require.Contains(t, mountPoint, "type MountPoint struct {")
	require.Contains(t, mountPoint, "Method string")
	require.Contains(t, mountPoint, "Pattern string")

	appendFS := codegen.SectionCode(t, serverFile.Section("append-fs")[0])
	require.Contains(t, appendFS, "type appendFS struct {")
	require.Contains(t, appendFS, "func appendPrefix(fsys http.FileSystem, prefix string) http.FileSystem {")
}

func TestConvertedTypeRenderSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadFormBodyUnionDSL)
	services := CreateHTTPServices(root)
	file := serverType("gen", root.API.HTTP.Services[0], services)

	unionSection := codegen.SectionCode(t, file.Section("server-union-type")[0])
	require.Contains(t, unionSection, "type Values struct {")
	require.Contains(t, unionSection, "func (u Values) MarshalFormValues")
	require.Contains(t, unionSection, "func (u *Values) UnmarshalFormValues")
}

func TestConvertedStreamRenderSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.SSEObjectDSL)
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)

	var sseFile *codegen.File
	for _, f := range files {
		if strings.HasSuffix(f.Path, filepath.Join("client", "sse.go")) {
			sseFile = f
			break
		}
	}
	require.NotNil(t, sseFile)

	sseSection := codegen.SectionCode(t, sseFile.Section("client-sse")[0])
	require.Contains(t, sseSection, "type SSEObjectMethodClientStream interface {")
	require.Contains(t, sseSection, "func NewSSEObjectMethodStream(")
	require.Contains(t, sseSection, "func (s *SSEObjectMethodStreamImpl) checkBuffer() ([]byte, bool) {")
}

func TestConvertedCLIRenderSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StreamingMultipleServicesDSL)
	services := CreateHTTPServices(root)
	files := ClientCLIFiles("gen", services)
	require.NotEmpty(t, files)

	parseFile := findFileWithSection(t, files, "parse-endpoint")
	parseSection := codegen.SectionCode(t, parseFile.Section("parse-endpoint")[0])
	require.Contains(t, parseSection, "func ParseEndpoint(")
	require.Contains(t, parseSection, "return endpoint, data, nil")
	require.Contains(t, parseSection, "dialer loomhttp.Dialer")

	pathRoot := RunHTTPDSL(t, testdata.PathMultipleParamsDSL)
	pathServices := CreateHTTPServices(pathRoot)
	pathFile := findFileWithSection(t, PathFiles(pathServices), "path")
	pathSection := codegen.SectionCode(t, pathFile.Section("path")[0])
	require.Contains(t, pathSection, "returns the URL path")
	require.Contains(t, pathSection, "fmt.Sprintf(")
}

func TestConvertedExampleRenderSections(t *testing.T) {
	example.Servers = make(example.ServersData)
	root := codegen.RunDSL(t, ctestdata.ServerHostingMultipleServicesDSL)
	services := NewServicesData(service.NewServicesData(root), root.API.HTTP)

	cliFile := findFileWithSection(t, ExampleCLIFiles("", services), "cli-http-start")
	cliStart := rawSectionCode(t, cliFile.Section("cli-http-start")[0])
	require.Contains(t, cliStart, "func doHTTP(")
	require.Contains(t, cliStart, "doer loomhttp.Doer")

	serverFile := findFileWithSection(t, ExampleServerFiles("", services), "server-http-start")
	serverStart := rawSectionCode(t, serverFile.Section("server-http-start")[0])
	require.Contains(t, serverStart, "func handleHTTPServer(")
	require.Contains(t, serverStart, "errc chan error")
}

func TestConvertedMiscRenderSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadMultipartPrimitiveDSL)
	services := CreateHTTPServices(root)

	clientEncodeFile := findFileWithSuffix(t, ClientFiles("gen", services), filepath.Join("client", "encode_decode.go"))
	requestBuilder := codegen.SectionCode(t, clientEncodeFile.Section("request-builder")[0])
	require.Contains(t, requestBuilder, "http.NewRequest(")
	require.Contains(t, requestBuilder, "loomhttp.ErrInvalidURL(")

	encoderSection := codegen.SectionCode(t, clientEncodeFile.Section("multipart-request-encoder")[0])
	require.Contains(t, encoderSection, "multipart.NewWriter(body)")
	require.Contains(t, encoderSection, "mw.FormDataContentType()")

	serverFile := findFileWithSuffix(t, ServerFiles("gen", services), filepath.Join("server", "server.go"))
	decoderType := codegen.SectionCode(t, serverFile.Section("multipart-request-decoder-type")[0])
	require.Contains(t, decoderType, "type ServiceMultipartPrimitiveMethodMultipartPrimitiveDecoderFunc")
	require.Contains(t, decoderType, "func(*multipart.Reader, *string) error")
}

func findFileWithSection(t *testing.T, files []*codegen.File, sectionName string) *codegen.File {
	t.Helper()
	for _, f := range files {
		if len(f.Section(sectionName)) > 0 {
			return f
		}
	}
	t.Fatalf("missing file with section %q", sectionName)
	return nil
}

func findFileWithSuffix(t *testing.T, files []*codegen.File, suffix string) *codegen.File {
	t.Helper()
	for _, f := range files {
		if strings.HasSuffix(f.Path, suffix) {
			return f
		}
	}
	t.Fatalf("missing file with suffix %q", suffix)
	return nil
}

func rawSectionCode(t *testing.T, section codegen.Section) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, section.Write(&buf))
	return buf.String()
}
