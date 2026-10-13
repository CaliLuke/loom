package testdata

// BasicCLISourcesHarness exercises one logical CLI input across its Basic and
// explicit wire uses. Replace TRANSPORT and MAPPING for the generated specimen.
const BasicCLISourcesHarness = `package sources_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"flag"
	"io"
	"net/http"
	"reflect"
	"testing"

	cli "example.com/sources/gen/TRANSPORT/cli/sources"
	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type captureDoer struct {
	request *http.Request
}

var stopRequest = errors.New("request captured")

func (d *captureDoer) Do(r *http.Request) (*http.Response, error) {
	d.request = r
	return nil, stopRequest
}

func TestCLISourceValues(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	for _, tc := range []struct {
		name string
		flags []string
		user, pass string
		absent, rejected bool
	}{
		{"explicit", []string{"--login=explicit-user", "--passcode=explicit-pass"}, "explicit-user", "explicit-pass", false, false},
		{"omitted", nil, "alice", "secret", "MAPPING" == "optional", "MAPPING" != "optional" && "MAPPING" != "defaulted"},
		{"invalid-user", []string{"--login=x", "--passcode=valid-pass"}, "", "", false, true},
		{"invalid-pass", []string{"--login=valid-user", "--passcode=x"}, "", "", false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := append([]string{"sources", "call"}, tc.flags...)
			switch "MAPPING" {
			case "scalar-body":
				args = append(args, "--body=\"body-user\"")
			case "object-body":
				args = append(args, "--body={\"login\":\"body-user\",\"passcode\":\"body-pass\"}")
			}
			flag.CommandLine = flag.NewFlagSet("sources", flag.ContinueOnError)
			require.NoError(t, flag.CommandLine.Parse(args))
			doer := &captureDoer{}
			endpoint, payload, err := cli.ParseEndpoint("http", "localhost", doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			if tc.rejected {
				require.Error(t, err)
				require.Nil(t, doer.request)
				return
			}
			require.NoError(t, err)
			if tc.absent {
				tc.user, tc.pass = "", ""
			}
			userField := "AuthUser"
			if "MAPPING" == "scalar-body" {
				userField = "Login"
			}
			value := reflect.ValueOf(payload).Elem()
			for name, expected := range map[string]string{userField: tc.user, "AuthPassword": tc.pass} {
				field := value.FieldByName(name)
				require.True(t, field.IsValid(), name)
				if tc.absent {
					require.True(t, field.IsNil(), "%s fabricated presence", name)
					continue
				}
				if field.Kind() == reflect.Pointer {
					require.False(t, field.IsNil(), name)
					field = field.Elem()
				}
				require.Equal(t, "Credential", field.Type().Name())
				require.Equal(t, expected, field.String(), name)
			}
			_, err = endpoint(context.Background(), payload)
			require.ErrorIs(t, err, stopRequest)
			require.NotNil(t, doer.request)
			r := doer.request
			user, pass, present := r.BasicAuth()
			wantBasic := !tc.absent && "MAPPING" != "ordinary-repeated"
			require.Equal(t, wantBasic, present)
			if wantBasic {
				require.Equal(t, tc.user, user)
				require.Equal(t, tc.pass, pass)
			}
			switch "MAPPING" {
			case "query-header", "optional", "defaulted":
				require.Equal(t, tc.user, r.URL.Query().Get("signin"))
				require.Equal(t, !tc.absent, r.URL.Query().Has("signin"))
				require.Equal(t, tc.pass, r.Header.Get("X-Proof"))
			case "repeated", "ordinary-repeated":
				require.Equal(t, tc.user, r.URL.Query().Get("signin"))
				require.Equal(t, tc.user, r.Header.Get("X-Login"))
				cookie, err := r.Cookie("proof")
				require.NoError(t, err)
				require.Equal(t, tc.pass, cookie.Value)
			case "path-cookie":
				require.Equal(t, "/"+tc.user, r.URL.Path)
				cookie, err := r.Cookie("proof")
				require.NoError(t, err)
				require.Equal(t, tc.pass, cookie.Value)
			case "scalar-body", "object-body":
				body, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				require.NoError(t, r.Body.Close())
				var decoded any
				require.NoError(t, json.Unmarshal(body, &decoded))
				if "TRANSPORT" == "jsonrpc" {
					decoded = decoded.(map[string]any)["params"]
				}
				if "MAPPING" == "scalar-body" {
					require.Equal(t, tc.user, decoded)
				} else {
					require.Equal(t, map[string]any{"login": tc.user, "passcode": tc.pass}, decoded)
				}
			}
		})
	}
}
`
