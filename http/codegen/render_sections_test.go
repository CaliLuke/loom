package codegen

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
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

func TestServerFileRendersSeparatedDeclarationsAndAttachedFieldDocs(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ServerFileServerDSL)
	services := CreateHTTPServices(root)
	serverFile := findFileWithSuffix(t, ServerFiles("gen", services), filepath.Join("server", "server.go"))

	renderedPath, err := serverFile.Render(t.TempDir())
	require.NoError(t, err)

	source, err := os.ReadFile(renderedPath)
	require.NoError(t, err)
	code := string(source)

	require.NotContains(t, code, "} //")
	require.Contains(t, code, "// Method is the name of the service method served by the mounted HTTP handler.\n\tMethod string")
	require.Contains(t, code, "// Verb is the HTTP method used to match requests to the mounted handler.\n\tVerb string")
	require.Contains(t, code, "// Pattern is the HTTP request path pattern used to match requests to the mounted handler.\n\tPattern string")
}

func TestRenderAppendFSOpenBodySortsMappedFiles(t *testing.T) {
	code := renderAppendFSOpenBody(map[string]string{
		"/z": "/z.json",
		"/y": "/y.json",
		"/x": "/x.json",
		"/w": "/w.json",
		"/v": "/v.json",
		"/u": "/u.json",
		"/t": "/t.json",
		"/s": "/s.json",
	})

	require.Equal(t, "\tswitch name {\n"+
		"\tcase \"/s\":\n\t\tname = \"/s.json\"\n"+
		"\tcase \"/t\":\n\t\tname = \"/t.json\"\n"+
		"\tcase \"/u\":\n\t\tname = \"/u.json\"\n"+
		"\tcase \"/v\":\n\t\tname = \"/v.json\"\n"+
		"\tcase \"/w\":\n\t\tname = \"/w.json\"\n"+
		"\tcase \"/x\":\n\t\tname = \"/x.json\"\n"+
		"\tcase \"/y\":\n\t\tname = \"/y.json\"\n"+
		"\tcase \"/z\":\n\t\tname = \"/z.json\"\n"+
		"\t}\n"+
		"\treturn s.fs.Open(path.Join(s.prefix, name))\n", code)
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
	require.Contains(t, sseSection, "loomhttp.NewSSEStreamReader(resp.Body)")
	require.NotContains(t, sseSection, "func (s *SSEObjectMethodStreamImpl) checkBuffer() ([]byte, bool) {")
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
	require.Contains(t, parseSection, `fmt.Errorf("parse command: %w", err)`)
	require.Contains(t, parseSection, "dialer loomhttp.Dialer")

	pathRoot := RunHTTPDSL(t, testdata.PathMultipleParamsDSL)
	pathServices := CreateHTTPServices(pathRoot)
	pathFile := findFileWithSection(t, PathFiles(pathServices), "path")
	pathSection := codegen.SectionCode(t, pathFile.Section("path")[0])
	require.Contains(t, pathSection, "returns the URL path")
	require.Contains(t, pathSection, "fmt.Sprintf(")
}

func TestConvertedExampleRenderSections(t *testing.T) {
	root := codegen.RunDSL(t, ctestdata.ServerHostingMultipleServicesDSL)
	services := NewServicesData(service.NewServicesData(root), root.API.HTTP)

	cliFile := findFileWithSection(t, ExampleCLIFiles("", services), "cli-http-start")
	cliStart := renderedSectionSource(t, cliFile.Section("cli-http-start")[0])
	require.Contains(t, cliStart, "func doHTTP(")
	require.Contains(t, cliStart, "doer loomhttp.Doer")

	cliEnd := renderedSectionSource(t, cliFile.Section("cli-http-end")[0])
	require.Contains(t, cliEnd, `fmt.Errorf("parse endpoint: %w", err)`)

	serverFile := findFileWithSection(t, ExampleServerFiles("", services), "server-http-start")
	serverStart := renderedSectionSource(t, serverFile.Section("server-http-start")[0])
	require.Contains(t, serverStart, "func handleHTTPServer(")
	require.Contains(t, serverStart, "errc chan error")

	serverEnd := renderedSectionSource(t, serverFile.Section("server-http-end")[0])
	require.Contains(t, serverEnd, "context.WithoutCancel(ctx)")
	require.NotContains(t, serverEnd, "context.WithTimeout(context.Background()")
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

func TestHTTPEncodeDecodeSectionsUseGenericTemplateSections(t *testing.T) {
	root := RunHTTPDSL(t, testdata.PayloadBodyObjectDSL)
	services := CreateHTTPServices(root)
	files := ClientFiles("gen", services)

	encodeFile := findFileWithSection(t, files, "request-encoder")
	requestEncoder := encodeFile.Section("request-encoder")[0]
	require.IsType(t, &codegen.TextTemplateSection{}, requestEncoder)

	serverFiles := ServerFiles("gen", services)
	decodeFile := findFileWithSection(t, serverFiles, "request-decoder")
	requestDecoder := decodeFile.Section("request-decoder")[0]
	require.IsType(t, &codegen.TextTemplateSection{}, requestDecoder)
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

func renderedSectionSource(t *testing.T, section codegen.Section) string {
	t.Helper()
	var buf bytes.Buffer
	require.NoError(t, section.Write(&buf))
	return buf.String()
}
