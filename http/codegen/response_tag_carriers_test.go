package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

const responseTagCarriersHarness = `package tagcarrier_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"testing"

	client "example.com/tagcarriers/gen/http/tagged/client"
	server "example.com/tagcarriers/gen/http/tagged/server"
	svc "example.com/tagcarriers/gen/tagged"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type implementation struct {
	result any
}

func TestTagCarriers(t *testing.T) {
	for _, tc := range []struct {
		name   string
		shape  any
		method func(*client.Client) loom.Endpoint
	}{
		{"ordinary", &svc.OrdinaryResult{}, (*client.Client).Ordinary},
		{"required", &svc.RequiredResult{}, (*client.Client).Required},
		{"alias", &svc.AliasResult{}, (*client.Client).Alias},
		{"required_alias", &svc.RequiredAliasResult{}, (*client.Client).RequiredAlias},
		{"nullable", &svc.NullableResult{}, (*client.Client).Nullable},
		{"named_nullable", &svc.NamedNullableResult{}, (*client.Client).NamedNullable},
		{"renamed", &svc.RenamedResult{}, (*client.Client).Renamed},
		{"mapped", &svc.MappedResult{}, (*client.Client).Mapped},
		{"viewed", &svc.ViewedResult{}, (*client.Client).Viewed},
		{"viewed_alias", &svc.ViewedAliasResult{}, (*client.Client).ViewedAlias},
		{"bodyless", &svc.BodylessResult{}, (*client.Client).Bodyless},
	} {
		for _, state := range []string{"accepted", "other", "absent", "null"} {
			if state == "null" && tc.name != "nullable" && tc.name != "named_nullable" {
				continue
			}
			t.Run(tc.name+"/"+state, func(t *testing.T) {
				value := reflect.New(reflect.TypeOf(tc.shape).Elem())
				value.Elem().FieldByName("Message").SetString("message")
				fieldName := "Kind"
				if tc.name == "renamed" {
					fieldName = "Disposition"
				}
				field := value.Elem().FieldByName(fieldName)
				if state != "absent" {
					setTag(field, state)
				}
				impl := &implementation{result: value.Interface()}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(svc.NewEndpoints(impl), mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				host := httptest.NewServer(mux)
				defer host.Close()
				u, err := url.Parse(host.URL)
				require.NoError(t, err)
				c := client.NewClient(u.Scheme, u.Host, host.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
				response, err := host.Client().Get(host.URL + "/" + tc.name)
				require.NoError(t, err)
				raw, err := io.ReadAll(response.Body)
				require.NoError(t, err)
				require.NoError(t, response.Body.Close())
				wantStatus := http.StatusOK
				if state == "accepted" {
					wantStatus = http.StatusAccepted
				}
				require.Equal(t, wantStatus, response.StatusCode)
				if tc.name == "bodyless" {
					require.Empty(t, raw)
				} else {
					require.Equal(t, "\"message\"\n", string(raw))
				}
				decoded, err := tc.method(c)(context.Background(), nil)
				require.NoError(t, err)
				got := reflect.ValueOf(decoded).Elem().FieldByName(fieldName)
				if state == "accepted" {
					require.Equal(t, "accepted", tagValue(got))
				} else {
					require.Equal(t, "", tagValue(got), "body omitted the tag; fallback must not invent it")
				}
			})
		}
	}
}

func setTag(field reflect.Value, state string) {
	if field.Kind() == reflect.Pointer {
		p := reflect.New(field.Type().Elem())
		p.Elem().SetString(state)
		field.Set(p)
		return
	}
	if field.Kind() == reflect.String {
		field.SetString(state)
		return
	}
	if state == "null" {
		field.Addr().MethodByName("SetNull").Call(nil)
		return
	}
	setter := field.Addr().MethodByName("SetValue")
	setter.Call([]reflect.Value{reflect.ValueOf(state).Convert(setter.Type().In(0))})
}

func tagValue(field reflect.Value) string {
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			return ""
		}
		return field.Elem().String()
	}
	if field.Kind() == reflect.String {
		return field.String()
	}
	values := field.MethodByName("Value").Call(nil)
	if !values[1].Bool() {
		return ""
	}
	return values[0].String()
}

func (s *implementation) Ordinary(context.Context) (*svc.OrdinaryResult, error) {
	return s.result.(*svc.OrdinaryResult), nil
}

func (s *implementation) Required(context.Context) (*svc.RequiredResult, error) {
	return s.result.(*svc.RequiredResult), nil
}

func (s *implementation) Alias(context.Context) (*svc.AliasResult, error) {
	return s.result.(*svc.AliasResult), nil
}

func (s *implementation) RequiredAlias(context.Context) (*svc.RequiredAliasResult, error) {
	return s.result.(*svc.RequiredAliasResult), nil
}

func (s *implementation) Nullable(context.Context) (*svc.NullableResult, error) {
	return s.result.(*svc.NullableResult), nil
}

func (s *implementation) NamedNullable(context.Context) (*svc.NamedNullableResult, error) {
	return s.result.(*svc.NamedNullableResult), nil
}

func (s *implementation) Renamed(context.Context) (*svc.RenamedResult, error) {
	return s.result.(*svc.RenamedResult), nil
}

func (s *implementation) Mapped(context.Context) (*svc.MappedResult, error) {
	return s.result.(*svc.MappedResult), nil
}

func (s *implementation) Viewed(context.Context) (*svc.ViewedResult, error) {
	return s.result.(*svc.ViewedResult), nil
}

func (s *implementation) ViewedAlias(context.Context) (*svc.ViewedAliasResult, error) {
	return s.result.(*svc.ViewedAliasResult), nil
}

func (s *implementation) Bodyless(context.Context) (*svc.BodylessResult, error) {
	return s.result.(*svc.BodylessResult), nil
}
`

// TestResponseTagCarriersGenerated checks legal physical carriers through mounted transports.
func TestResponseTagCarriersGenerated(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResponseTagCarriersDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/tagcarriers", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "tag_test.go"), []byte(responseTagCarriersHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", ".")
}
