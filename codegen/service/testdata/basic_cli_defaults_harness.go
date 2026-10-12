package testdata

// BasicCLIDefaultsHarness exercises parser defaults, payload representation and
// outbound Basic authentication. Replace TRANSPORT with the transport directory and MULTIPARTARGS with HTTP
// multipart encoder arguments (or nothing for JSON-RPC).
const BasicCLIDefaultsHarness = `package basic_test

import (
	"context"
	"errors"
	"flag"
	"net/http"
	"reflect"
	"testing"

	cli "example.com/basic/gen/TRANSPORT/cli/basic"
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

func TestBasicCLIFlags(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	for _, tc := range []struct {
		service, method string
		user, pass *string
		userField, passField string
	}{
		{"basic", "optional", nil, nil, "AuthUser", "AuthPassword"},
		{"basic", "defaulted", stringPointer("alice"), stringPointer("secret"), "AuthUser", "AuthPassword"},
		{"defaults", "plain", nil, nil, "Login", "Secret"},
		{"defaults", "both", stringPointer("alice"), stringPointer("secret"), "Login", "Secret"},
		{"defaults", "user", stringPointer("alice"), nil, "Login", "Secret"},
		{"defaults", "pass", nil, stringPointer("secret"), "Login", "Secret"},
		{"defaults", "required", stringPointer("alice"), stringPointer("secret"), "Login", "Secret"},
		{"defaults", "empty", stringPointer(""), nil, "Login", "Secret"},
		{"defaults", "inherited", stringPointer("inherited"), stringPointer("inherited"), "Login", "Secret"},
		{"defaults", "override", stringPointer("alice"), stringPointer("inherited"), "Login", "Secret"},
	} {
		t.Run(tc.service+"/"+tc.method, func(t *testing.T) {
			for _, explicit := range []bool{false, true} {
				args := []string{tc.service, tc.method}
				user, pass := tc.user, tc.pass
				if explicit {
					args = append(args, "--login=override-user", "--secret=override-pass")
					user, pass = stringPointer("override-user"), stringPointer("override-pass")
				}
				flag.CommandLine = flag.NewFlagSet("basic-defaults", flag.ContinueOnError)
				require.NoError(t, flag.CommandLine.Parse(args))
				doer := &captureDoer{}
				endpoint, payload, err := cli.ParseEndpoint("http", "localhost", doer, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, falseMULTIPARTARGS)
				require.NoError(t, err)
				value := reflect.ValueOf(payload).Elem()
				for name, expected := range map[string]*string{tc.userField: user, tc.passField: pass} {
					field := value.FieldByName(name)
					if expected == nil {
						require.True(t, field.IsNil(), "%s fabricated presence", name)
						continue
					}
					if field.Kind() == reflect.Pointer {
						require.False(t, field.IsNil(), name)
						field = field.Elem()
					}
					require.Equal(t, *expected, field.String(), name)
				}
				_, err = endpoint(context.Background(), payload)
				require.ErrorIs(t, err, stopRequest)
				require.NotNil(t, doer.request)
				gotUser, gotPass, present := doer.request.BasicAuth()
				require.Equal(t, user != nil || pass != nil, present)
				require.Equal(t, stringValue(user), gotUser)
				require.Equal(t, stringValue(pass), gotPass)
			}
		})
	}
	for _, method := range []string{"user", "pass"} {
		t.Run("required/"+method, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet("basic-defaults", flag.ContinueOnError)
			require.NoError(t, flag.CommandLine.Parse([]string{"basic", method}))
			_, _, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, falseMULTIPARTARGS)
			require.Error(t, err)
		})
	}
	for _, method := range []string{"user", "pass", "defaulted"} {
		t.Run("constraints/"+method, func(t *testing.T) {
			flag.CommandLine = flag.NewFlagSet("basic-defaults", flag.ContinueOnError)
			require.NoError(t, flag.CommandLine.Parse([]string{"basic", method, "--login=x", "--secret=x"}))
			_, _, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, falseMULTIPARTARGS)
			require.Error(t, err)
		})
	}
}

func stringPointer(value string) *string {
	return &value
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
`
