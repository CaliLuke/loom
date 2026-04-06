package cli

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"
)

func TestJSONExampleHandlesEmptyMaps(t *testing.T) {
	require.NotPanics(t, func() {
		require.Equal(t, "{}", jsonExample(map[int]int{}))
	})
}

func TestBuildCommandDataDefaultsDescriptionAndInterceptors(t *testing.T) {
	data := &service.Data{
		Name:               "Storage",
		PkgName:            "storage",
		ClientInterceptors: []*service.InterceptorData{{Name: "Logging"}},
	}

	cmd := BuildCommandData(data)

	require.Equal(t, "storage", cmd.Name)
	require.Equal(t, "storage", cmd.VarName)
	require.Equal(t, `Make requests to the "Storage" service`, cmd.Description)
	require.Equal(t, "storagec", cmd.PkgName)
	require.NotNil(t, cmd.Interceptors)
	require.Equal(t, "storageInter", cmd.Interceptors.VarName)
	require.Equal(t, "storage", cmd.Interceptors.PkgName)
}

func TestBuildSubcommandDataBuildsJSONConversionAndExample(t *testing.T) {
	data := &service.Data{Name: "Storage", PkgName: "storage"}
	method := &service.MethodData{
		Name:               "Create Widget",
		VarName:            "CreateWidget",
		Payload:            "WidgetPayload",
		ClientInterceptors: []string{"Audit"},
	}
	flags := []*FlagData{{
		Name:     "payload",
		FullName: "storageCreateWidgetPayload",
		Type:     "JSON",
		Example:  "{\"name\":\"demo\"}",
	}}

	sub := BuildSubcommandData(data, method, nil, flags)

	require.Equal(t, "create-widget", sub.Name)
	require.Equal(t, "storageCreateWidget", sub.FullName)
	require.Contains(t, sub.Description, `"Create Widget" endpoint`)
	rendered := renderStatement(t, sub.Conversion)
	require.Contains(t, rendered, "var val WidgetPayload")
	require.Contains(t, rendered, "json.Unmarshal")
	require.Contains(t, rendered, "invalid JSON for storageCreateWidgetPayloadFlag")
	require.Equal(t, "storage create-widget --payload {\"name\":\"demo\"}", sub.Example)
	require.NotNil(t, sub.Interceptors)
	require.Equal(t, "storageInter", sub.Interceptors.VarName)
}

func TestFlagsCodeIncludesServiceAndEndpointValidation(t *testing.T) {
	code := FlagsCode([]*CommandData{
		{
			Name:    "storage",
			VarName: "storage",
			Subcommands: []*SubcommandData{
				{Name: "add", FullName: "storageAdd"},
				{Name: "show", FullName: "storageShow"},
			},
		},
	})

	require.Contains(t, code, `flag.NewFlagSet("storage", flag.ContinueOnError)`)
	require.Contains(t, code, `flag.NewFlagSet("add", flag.ExitOnError)`)
	require.Contains(t, code, `flag.CommandLine.Parse(os.Args[1:])`)
	require.Contains(t, code, `return nil, nil, fmt.Errorf("unknown service %q", svcn)`)
	require.Contains(t, code, `return nil, nil, fmt.Errorf("unknown %q endpoint %q", svcn, epn)`)
}

func renderStatement(t *testing.T, stmt *jen.Statement) string {
	t.Helper()
	require.NotNil(t, stmt)
	file := jen.NewFile("cli")
	file.Func().Id("render").Params().Params(jen.Any(), jen.Any(), jen.Error()).Block(stmt)
	var b strings.Builder
	require.NoError(t, file.Render(&b))
	src := b.String()
	marker := "func render() {\n"
	idx := strings.Index(src, marker)
	if idx == -1 {
		return src
	}
	body := src[idx+len(marker):]
	body = strings.TrimSuffix(body, "}\n")
	return body
}
