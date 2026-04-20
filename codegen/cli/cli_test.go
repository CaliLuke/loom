package cli

import (
	"strings"
	"testing"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
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
		Name:    "Create Widget",
		VarName: "CreateWidget",
		MethodPayloadData: service.MethodPayloadData{
			Payload: "WidgetPayload",
		},
		MethodSecurityData: service.MethodSecurityData{
			ClientInterceptors: []string{"Audit"},
		},
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

func TestConversionCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		from            string
		to              string
		typeName        string
		pointer         bool
		wantDeclErr     bool
		wantCheckErr    bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name:         "pointer int uses local value and assignment",
			from:         "raw",
			to:           "target",
			typeName:     "int",
			pointer:      true,
			wantDeclErr:  true,
			wantCheckErr: true,
			wantContains: []string{
				"var v int64",
				"v, err = strconv.ParseInt(raw, 10, strconv.IntSize)",
				"val := int(v)",
				"target = &val",
			},
		},
		{
			name:         "string conversion is direct",
			from:         "raw",
			to:           "target",
			typeName:     "string",
			pointer:      false,
			wantDeclErr:  false,
			wantCheckErr: false,
			wantContains: []string{
				"target = raw",
			},
			wantNotContains: []string{
				"Parse",
				"json.Unmarshal",
			},
		},
		{
			name:         "json conversion unmarshals",
			from:         "raw",
			to:           "target",
			typeName:     "WidgetPayload",
			pointer:      false,
			wantDeclErr:  true,
			wantCheckErr: true,
			wantContains: []string{
				"err = json.Unmarshal([]byte(raw), &target)",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt, declErr, checkErr := conversionCode(tc.from, tc.to, tc.typeName, tc.pointer)
			require.Equal(t, tc.wantDeclErr, declErr)
			require.Equal(t, tc.wantCheckErr, checkErr)
			rendered := renderStatement(t, stmt)
			for _, want := range tc.wantContains {
				require.Contains(t, rendered, want)
			}
			for _, want := range tc.wantNotContains {
				require.NotContains(t, rendered, want)
			}
		})
	}
}

func TestFieldLoadCode(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name            string
		flag            *FlagData
		argName         string
		argTypeName     string
		validate        string
		defaultValue    any
		payload         expr.DataType
		payloadRef      string
		wantDeclErr     bool
		wantContains    []string
		wantNotContains []string
	}{
		{
			name: "optional string without default uses pointer and guard",
			flag: &FlagData{
				FullName: "nameFlag",
				Required: false,
			},
			argName:      "name",
			argTypeName:  "string",
			payload:      &expr.Object{},
			payloadRef:   "*Payload",
			wantDeclErr:  false,
			wantContains: []string{"if nameFlag != \"\" {", "name = &nameFlag"},
		},
		{
			name: "required int with validation emits conversion and zero return",
			flag: &FlagData{
				FullName: "countFlag",
				Type:     "INT",
				Required: true,
			},
			argName:      "count",
			argTypeName:  "int",
			validate:     "if err = validateCount(count); err != nil {\n}\n",
			payload:      expr.Int,
			payloadRef:   "int",
			wantDeclErr:  true,
			wantContains: []string{"strconv.ParseInt(countFlag, 10, strconv.IntSize)", "var zero int", "return zero, err"},
		},
		{
			name: "json conversion error keeps nil return",
			flag: &FlagData{
				FullName: "payloadFlag",
				Type:     "JSON",
				Example:  "{\"name\":\"demo\"}",
				Required: true,
			},
			argName:      "payload",
			argTypeName:  "WidgetPayload",
			payload:      &expr.Object{},
			payloadRef:   "*WidgetPayload",
			wantDeclErr:  true,
			wantContains: []string{"json.Unmarshal([]byte(payloadFlag), &payload)", "return nil, fmt.Errorf(\"invalid JSON for payload"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			stmt, declErr := FieldLoadCode(tc.flag, tc.argName, tc.argTypeName, tc.validate, tc.defaultValue, tc.payload, tc.payloadRef)
			require.Equal(t, tc.wantDeclErr, declErr)
			rendered := renderStatement(t, stmt)
			for _, want := range tc.wantContains {
				require.Contains(t, rendered, want)
			}
			for _, want := range tc.wantNotContains {
				require.NotContains(t, rendered, want)
			}
		})
	}
}

func TestFlagsCodeStatementBranches(t *testing.T) {
	t.Parallel()

	stmt := FlagsCodeStatement([]*CommandData{
		{
			Name:    "storage",
			VarName: "storage",
			Subcommands: []*SubcommandData{
				{
					Name:     "add",
					FullName: "storageAdd",
					Flags: []*FlagData{
						{Name: "name", FullName: "storageAddName", Description: "item name", Required: true},
						{Name: "count", FullName: "storageAddCount", Description: "item count", Default: 3},
					},
				},
			},
		},
	})

	rendered := renderStatement(t, stmt)
	require.Contains(t, rendered, `flag.NewFlagSet("storage", flag.ContinueOnError)`)
	require.Contains(t, rendered, `flag.NewFlagSet("add", flag.ExitOnError)`)
	require.Contains(t, rendered, `storageAddFlags.String("name", "REQUIRED", "item name")`)
	require.Contains(t, rendered, `storageAddFlags.String("count", "3", "item count")`)
	require.Contains(t, rendered, `storageFlags.Usage = storageUsage`)
	require.Contains(t, rendered, `storageAddFlags.Usage = storageAddUsage`)
	require.Contains(t, rendered, `if flag.NArg() < 2`)
	require.Contains(t, rendered, `svcn = flag.Arg(0)`)
	require.Contains(t, rendered, `epn = svcf.Arg(0)`)
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
