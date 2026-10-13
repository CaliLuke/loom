package codegen

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/dave/jennifer/jen"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen/cli"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestCLIHelpLiterals(t *testing.T) {
	texts := []string{"ordinary text", "Hello ${missing}", "${name=fallback}", "${default}", "$$", "$${name}", "$5 and trailing $", "quote \" slash \\ newline\nnext", "${unfinished"}
	for _, text := range texts {
		for _, location := range []string{"service", "method", "flag", "default"} {
			t.Run(location+"/"+text, func(t *testing.T) {
				flag := &cli.FlagData{Name: "value", Description: "flag"}
				sub := &cli.SubcommandData{Name: "call", Description: "method", Flags: []*cli.FlagData{flag}}
				command := &cli.CommandData{Name: "sample", Description: "service", Subcommands: []*cli.SubcommandData{sub}}
				expected := text
				switch location {
				case "service":
					command.Description = text
				case "method":
					sub.Description = text
				case "flag":
					flag.Description = text
				case "default":
					flag.Default = text
					expected = strings.TrimSpace(flag.Description + fmt.Sprintf(" (default: %q)", text))
				}
				statement := jen.Empty()
				appendKongCommandLineStruct(statement, []*cli.CommandData{command})
				file, err := parser.ParseFile(token.NewFileSet(), "command.go", "package cli\n"+fmt.Sprintf("%#v", statement), 0)
				require.NoError(t, err)
				definition := file.Decls[0].(*ast.GenDecl).Specs[0].(*ast.TypeSpec).Type.(*ast.StructType)
				value := reflect.New(cliHelpStructType(t, definition)).Interface()
				parsed, err := kong.New(value, kong.Name("literal-help"))
				require.NoError(t, err)
				service := parsed.Model.Children[0]
				method := service.Children[0]
				actual := method.Flags[0].Help
				if location == "service" {
					actual = service.Help
				} else if location == "method" {
					actual = method.Help
				}
				require.Equal(t, expected, actual)
				ctx, err := parsed.Parse([]string{"sample", "call", "--value", text})
				require.NoError(t, err)
				require.Equal(t, "sample call", ctx.Command())
				got := reflect.ValueOf(value).Elem().Field(0).Field(0).Field(0)
				if got.Kind() == reflect.Pointer {
					got = got.Elem()
				}
				require.Equal(t, text, got.String())
			})
		}
	}
}

func TestHTTPCLIHelpLiteralsGenerated(t *testing.T) {
	for _, special := range []bool{false, true} {
		t.Run(strconv.FormatBool(special), func(t *testing.T) {
			root := RunHTTPDSL(t, httpdata.CLIHelpLiteralDSL("http", special))
			dir := t.TempDir()
			if output := os.Getenv("LOOM_CLI_HELP_OUTPUT"); output != "" {
				dir = filepath.Join(output, "http", strconv.FormatBool(special))
				require.NoError(t, os.MkdirAll(dir, 0o700))
			}
			renderHTTPModule(t, dir, "example.com/help", root)
			text := "ordinary text"
			if special {
				text = httpdata.CLIHelpLiteralText
			}
			harness := strings.ReplaceAll(httpdata.CLIHelpLiteralHarness, "EXPECTED_TEXT", strconv.Quote(text))
			require.NoError(t, os.WriteFile(filepath.Join(dir, "gen/http/cli/literalcli/cli_help_test.go"), []byte(harness), 0o600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
			runGoCommand(t, dir, "vet", "./...")
		})
	}
}

func cliHelpStructType(t *testing.T, structure *ast.StructType) reflect.Type {
	t.Helper()
	fields := make([]reflect.StructField, 0, len(structure.Fields.List))
	for _, field := range structure.Fields.List {
		fieldType := reflect.TypeFor[string]()
		switch typ := field.Type.(type) {
		case *ast.StructType:
			fieldType = cliHelpStructType(t, typ)
		case *ast.StarExpr:
			fieldType = reflect.PointerTo(fieldType)
		}
		tag, err := strconv.Unquote(field.Tag.Value)
		require.NoError(t, err)
		fields = append(fields, reflect.StructField{Name: field.Names[0].Name, Type: fieldType, Tag: reflect.StructTag(tag)})
	}
	return reflect.StructOf(fields)
}
