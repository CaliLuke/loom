package testdata

// CollectionBodyDefaultsCLIHarness exercises omitted and explicit collection CLI
// flags. Replace TRANSPORT with the generated transport directory.
const CollectionBodyDefaultsCLIHarness = `package collectiondefaults_test

import (
	"flag"
	"net/http"
	"reflect"
	"testing"

	svc "example.com/collectiondefaults/gen/collections"
	cli "example.com/collectiondefaults/gen/TRANSPORT/cli/collection_defaults"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

func TestCollectionBodyFlags(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	for _, tc := range []struct {
		name string
		fallback any
		explicit string
		override any
	}{
		{"array", []string{"a", "b"}, "[]", []string{}},
		{"string-map", map[string]int{"a":1}, "{}", map[string]int{}},
		{"boolean-map", map[bool]string{true:"on"}, "{}", map[bool]string{}},
		{"nested-map", map[string]map[bool]int{"team":{false:9}}, "{\"other\":{\"true\":0}}", map[string]map[bool]int{"other":{true:0}}},
		{"nested-array", []map[bool]bool{{true:false}}, "[{\"false\":true}]", []map[bool]bool{{false:true}}},
		{"named-key", map[svc.FlagKey]string{true:"on"}, "{\"false\":\"off\"}", map[svc.FlagKey]string{false:"off"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, explicit := range []bool{false, true} {
				args := []string{"collections", tc.name}
				expected := tc.fallback
				if explicit {
					args = append(args, "--body="+tc.explicit)
					expected = tc.override
				}
				flag.CommandLine = flag.NewFlagSet("collection-defaults", flag.ContinueOnError)
				require.NoError(t, flag.CommandLine.Parse(args))
				_, payload, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				require.NoError(t, err)
				require.Equal(t, expected, reflect.ValueOf(payload).Elem().FieldByName("Value").Interface())
			}
			for _, invalid := range []string{"", "{"} {
				flag.CommandLine = flag.NewFlagSet("collection-defaults", flag.ContinueOnError)
				require.NoError(t, flag.CommandLine.Parse([]string{"collections", tc.name, "--body="+invalid}))
				_, _, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				require.Error(t, err)
				require.Contains(t, err.Error(), "body")
			}
		})
	}
}
`
