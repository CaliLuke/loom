package service

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	httpdata "github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestServiceClientMemberIdentity(t *testing.T) {
	for _, tc := range []struct {
		name  string
		names []string
		mixed bool
	}{
		{"ordinary", []string{"watch", "watchStream"}, false},
		{"mixed", []string{"watch"}, true},
		{"authored-collision", []string{"watch", "watchStream"}, true},
		{"reverse-order", []string{"watchStream", "watch"}, true},
		{"occupied-fallback", []string{"watch", "watchStream", "watchStreamStream"}, true},
		{"field-competitor", []string{"watch", "watchStream", "watchStreamEndpoint"}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := codegen.RunDSL(t, func() {
				dsl.Service("members", func() {
					for _, name := range tc.names {
						dsl.Method(name, func() {
							dsl.Payload(dsl.String)
							dsl.Result(dsl.String)
							if tc.mixed && name == "watch" {
								dsl.StreamingResult(dsl.String)
							}
						})
					}
				})
			})
			services := NewServicesData(root)
			data := services.Get("members")
			var source strings.Builder
			for _, section := range ClientFile("example.com/members/gen", root.Services[0], services).AllSections() {
				require.NoError(t, section.Write(&source))
			}
			members, err := httpdata.ClientMemberDeclarations([]string{source.String()})
			require.NoError(t, err)
			expectedCount := 2 * len(tc.names)
			if tc.mixed {
				expectedCount += 2
			}
			require.Len(t, members["Client"], expectedCount, "each authored call and endpoint field plus both mixed roles must be emitted")
			expected := make([]string, 0, expectedCount)
			renamed := services.WithPackageNames(map[string]string{"types/context": "context2"}).Get("members")
			for index, method := range data.Methods {
				expected = append(expected, method.VarName, method.EndpointField)
				require.Equal(t, method.EndpointField, renamed.Methods[index].EndpointField)
				require.Equal(t, method.ClientStreamMethod, renamed.Methods[index].ClientStreamMethod)
				require.Equal(t, method.StreamEndpointField, renamed.Methods[index].StreamEndpointField)
				if method.HasMixedResults {
					require.NotEmpty(t, method.ClientStreamMethod)
					expected = append(expected, method.ClientStreamMethod, method.StreamEndpointField)
				}
			}
			require.ElementsMatch(t, expected, members["Client"], "all emitted roles consume the recorded owner plan")
			seen := make(map[string]bool)
			for _, name := range members["Client"] {
				if seen[name] {
					t.Errorf("Client.%s has duplicate declarations", name)
				}
				seen[name] = true
			}
			for index, name := range tc.names {
				require.Equal(t, name, data.Methods[index].Name)
				require.Equal(t, codegen.Goify(name, true), data.Methods[index].VarName)
				require.Contains(t, members["Client"], data.Methods[index].VarName, "authored method identity must survive allocation")
			}
		})
	}
}
