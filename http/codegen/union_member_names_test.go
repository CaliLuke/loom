package codegen

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestUnionMemberNamesGenerated(t *testing.T) {
	for _, test := range []struct {
		name                 string
		collisions, untagged bool
	}{
		{"ordinary", false, false},
		{"tagged", true, false},
		{"untagged", true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			names := []string{"left", "right"}
			if test.collisions {
				names = []string{"Kind", "Kind2", "Validate", "MarshalJSON", "UnmarshalJSON", "MarshalFormValues", "UnmarshalFormValues", "Text", "AsText", "SetText", "a-b", "a_b", "AB2"}
			}
			root := RunHTTPDSL(t, func() {
				choice := Type("Choice", OneOf(func() {
					if test.untagged {
						Untagged()
					}
					for _, name := range names {
						Attribute(name, String, func() {
							Enum(name)
						})
					}
				}))
				Service("members", func() {
					Method("echo", func() {
						Payload(choice)
						Result(choice)
						HTTP(func() {
							POST("/")
						})
					})
				})
			})
			dir := t.TempDir()
			if output := os.Getenv("LOOM_UNION_NAMES_OUTPUT"); output != "" {
				dir = filepath.Join(output, test.name)
			}
			renderHTTPModule(t, dir, "example.com/membernames", root)
			harness := fmt.Sprintf(unionMemberNamesHarness, fmt.Sprintf("%#v", names), test.untagged)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "members_test.go"), []byte(harness), 0600))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

const unionMemberNamesHarness = `package membernames_test

import (
	"context"
	json "encoding/json/v2"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"

	service "example.com/membernames/gen/members"
	client "example.com/membernames/gen/http/members/client"
	server "example.com/membernames/gen/http/members/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestBranches(t *testing.T) {
	calls := 0
	endpoints := &service.Endpoints{Echo: func(_ context.Context, value any) (any, error) {
		calls++
		return value, nil
	}}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	httpServer := httptest.NewServer(mux)
	defer httpServer.Close()
	c := client.NewClient("http", strings.TrimPrefix(httpServer.URL, "http://"), httpServer.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	for _, name := range %s {
		wire := fmt.Sprintf("{\"type\":%%q,\"value\":%%q}", name, name)
		if %t {
			wire = fmt.Sprintf("%%q", name)
		}
		var input service.Choice
		if err := json.Unmarshal([]byte(wire), &input); err != nil {
			t.Fatalf("%%s decode: %%v", name, err)
		}
		if string(input.Kind()) != name {
			t.Fatalf("%%s kind: %%s", name, input.Kind())
		}
		before := calls
		value, err := c.Echo()(context.Background(), &input)
		if err != nil {
			t.Fatalf("%%s roundtrip: %%v", name, err)
		}
		output := value.(*service.Choice)
		got, err := json.Marshal(output)
		if err != nil || string(got) != wire || calls != before+1 {
			t.Fatalf("%%s output %%s, err %%v, calls %%d", name, got, err, calls)
		}
	}
}
`
