package testdata

import "github.com/CaliLuke/loom/dsl"

// CLIHelpLiteralText exercises interpolation, escaped dollars and Go tag quoting.
const CLIHelpLiteralText = "Hello ${name} $$ ${default} \\\n\"quoted\""

// CLIHelpLiteralHarness checks the real generated command type, parsed help and
// payload defaults. Replace EXPECTED_TEXT with a quoted Go string before use.
const CLIHelpLiteralHarness = `package cli

import (
	"fmt"
	"os"
	"testing"

	"github.com/alecthomas/kong"
	"github.com/stretchr/testify/require"

	"example.com/help/gen/sample"
)

func TestGeneratedLiteralHelp(t *testing.T) {
	const text = EXPECTED_TEXT
	parser, err := kong.New(&commandLine{})
	require.NoError(t, err)
	service := parser.Model.Children[0]
	require.Equal(t, "service "+text, service.Help)
	method := service.Children[0]
	require.Equal(t, "method "+text, method.Help)
	require.Contains(t, method.Flags[0].Help, fmt.Sprintf("(default: %q)", text))
	original := os.Args
	t.Cleanup(func() {
		os.Args = original
	})
	for _, tc := range []struct {
		name string
		args []string
		value string
	}{
		{name: "default", args: []string{"sample", "call"}, value: text},
		{name: "explicit", args: []string{"sample", "call", "--value", "${explicit} $$"}, value: "${explicit} $$"},
		{name: "ordinary", args: []string{"sample", "ordinary"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			os.Args = append([]string{"help-client"}, tc.args...)
			endpoint, payload, err := ParseEndpoint("http", "localhost", nil, nil, nil, false)
			require.NoError(t, err)
			require.NotNil(t, endpoint)
			if tc.name == "ordinary" {
				require.Nil(t, payload)
				return
			}
			require.Equal(t, tc.value, payload.(*sample.CallPayload).Value)
		})
	}
}
`

// CLIHelpLiteralDSL combines literal help at every generated Kong tag boundary
// with a payload default and an unrelated ordinary command.
func CLIHelpLiteralDSL(transport string, special bool) func() {
	text := "ordinary text"
	if special {
		text = CLIHelpLiteralText
	}
	return func() {
		dsl.API("help", func() {
			if transport == "jsonrpc" {
				dsl.JSONRPC(func() {
				})
			}
			dsl.Server("literalcli", func() {
				dsl.Services("sample")
				dsl.Host("local", func() {
					dsl.URI("http://localhost:8080")
				})
			})
		})
		dsl.Service("sample", func() {
			dsl.Description("service " + text)
			if transport == "jsonrpc" {
				dsl.JSONRPC(func() {
					dsl.POST("/rpc")
				})
			}
			dsl.Method("call", func() {
				dsl.Description("method " + text)
				dsl.Payload(func() {
					dsl.Attribute("value", dsl.String, "value "+text, func() {
						dsl.Default(text)
					})
				})
				mapping := func() {
					if transport == "http" {
						dsl.GET("/call")
					}
					dsl.Param("value", func() {
						dsl.Description("value " + text)
					})
				}
				if transport == "jsonrpc" {
					dsl.JSONRPC(mapping)
				} else {
					dsl.HTTP(mapping)
				}
			})
			dsl.Method("ordinary", func() {
				if transport == "jsonrpc" {
					dsl.JSONRPC(func() {
					})
				} else {
					dsl.HTTP(func() {
						dsl.GET("/ordinary")
					})
				}
			})
		})
	}
}
