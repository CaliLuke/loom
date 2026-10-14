package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicecodegen "github.com/CaliLuke/loom/codegen/service"
	. "github.com/CaliLuke/loom/dsl"
)

func TestCustomMetadataGenerated(t *testing.T) {
	const module = "example.com/custommetadata"
	root := RunGRPCDSL(t, func() {
		value := Type("Value", func() {
			Field(1, "body", String)
			Attribute("required", String, func() {
				Meta("struct:field:type", "scalar.Token", module+"/scalar")
			})
			Attribute("opaque", String, func() {
				Meta("struct:field:type", "scalar.Opaque", module+"/scalar")
			})
			Attribute("optional", String, func() {
				Meta("struct:field:type", "scalar.Token", module+"/scalar")
			})
			Attribute("defaulted", String, func() {
				Meta("struct:field:type", "scalar.Token", module+"/scalar")
				Default("fallback")
			})
			Attribute("uuid", String, func() {
				Meta("struct:field:type", "uuid.UUID", "github.com/google/uuid")
				Default("550e8400-e29b-41d4-a716-446655440000")
			})
			Attribute("tokens", ArrayOf(String, func() {
				Meta("struct:field:type", "scalar.Token", module+"/scalar")
			}))
			Required("body", "required")
		})
		fields := func() {
			Attribute("required")
			Attribute("optional")
			Attribute("opaque")
			Attribute("defaulted")
			Attribute("uuid")
			Attribute("tokens")
		}
		Service("custommetadata", func() {
			Method("baddefault", func() {
				Payload(func() {
					Attribute("defaulted", String, func() {
						Meta("struct:field:type", "scalar.Token", module+"/scalar")
						Default("bad")
					})
				})
				GRPC(func() {
					Metadata(func() {
						Attribute("defaulted")
					})
				})
			})
			Method("echo", func() {
				Payload(value)
				Result(value)
				GRPC(func() {
					Message(func() {
						Attribute("body")
					})
					Metadata(fields)
					Response(CodeOK, func() {
						Message(func() {
							Attribute("body")
						})
						Headers(fields)
					})
				})
			})
		})
	})
	dir := t.TempDir()
	repository, err := filepath.Abs("../..")
	require.NoError(t, err)
	renderGRPCModule(t, dir, module, root, repository)
	services := servicecodegen.NewServicesData(root)
	for _, service := range root.Services {
		require.NoError(t, servicecodegen.SetUserTypeImports(module+"/gen", services.Get(service.Name)))
	}
	for _, file := range ClientCLIFiles(module+"/gen", NewServicesData(services)) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "scalar"), 0o750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "scalar", "token.go"), []byte(customMetadataToken), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "metadata_test.go"), []byte(customMetadataRuntime), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "vet", "./...")
	runGRPCGoCommand(t, dir, "test", "-race", "-count=1", ".")
}

const customMetadataToken = `package scalar

import "errors"

// ErrRejected identifies authored parse failures.
var ErrRejected = errors.New("token rejected")

// Token stores semantic text independently of native string storage.
type Token struct {
	// Text is the token's semantic value.
	Text string
}

// Opaque deliberately supplies no semantic text capabilities.
type Opaque struct {
	// Number is unrelated physical data.
	Number int
}

// String projects the token's semantic text.
func (t Token) String() string {
	return t.Text
}

// UnmarshalText accepts supplied empty text but rejects a reserved invalid token.
func (t *Token) UnmarshalText(data []byte) error {
	t.Text = "partial mutation"
	if string(data) == "bad" {
		return ErrRejected
	}
	t.Text = string(data)
	return nil
}
`

const customMetadataRuntime = `package custommetadata_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/metadata"

	svc "example.com/custommetadata/gen/custommetadata"
	client "example.com/custommetadata/gen/grpc/custommetadata/client"
	pb "example.com/custommetadata/gen/grpc/custommetadata/pb"
	server "example.com/custommetadata/gen/grpc/custommetadata/server"
	"example.com/custommetadata/scalar"
)

func TestMetadataDecode(t *testing.T) {
	for _, response := range []bool{false, true} {
		for _, tc := range []struct {
			name      string
			metadata  metadata.MD
			optional  *scalar.Token
			defaulted string
			tokens    []scalar.Token
			invalid   bool
			parser    bool
		}{
			{name: "absence defaults", metadata: metadata.Pairs("required", "present"), defaulted: "fallback"},
			{
				name:     "provided empty",
				metadata: metadata.Pairs("required", "", "optional", "", "defaulted", ""),
				optional: &scalar.Token{},
			},
			{
				name:      "array",
				metadata:  metadata.Pairs("required", "present", "tokens", "one", "tokens", "two"),
				defaulted: "fallback",
				tokens:    []scalar.Token{{Text: "one"}, {Text: "two"}},
			},
			{name: "required absent", metadata: metadata.MD{}, invalid: true},
			{name: "required parser", metadata: metadata.Pairs("required", "bad"), invalid: true, parser: true},
			{
				name:     "optional parser",
				metadata: metadata.Pairs("required", "present", "optional", "bad"),
				invalid:  true,
				parser:   true,
			},
			{
				name:     "defaulted parser",
				metadata: metadata.Pairs("required", "present", "defaulted", "bad"),
				invalid:  true,
				parser:   true,
			},
			{
				name:     "array parser",
				metadata: metadata.Pairs("required", "present", "tokens", "one", "tokens", "bad"),
				invalid:  true,
				parser:   true,
			},
			{
				name:     "opaque parser capability",
				metadata: metadata.Pairs("required", "present", "opaque", "x"),
				invalid:  true,
			},
			{name: "uuid empty is supplied", metadata: metadata.Pairs("required", "present", "uuid", ""), invalid: true},
		} {
			t.Run(tc.name+map[bool]string{true: "/response", false: "/request"}[response], func(t *testing.T) {
				body := "body"
				var got any
				var err error
				if response {
					got, err = client.DecodeEchoResponse(context.Background(), &pb.EchoResponse{Body: &body}, tc.metadata, nil)
				} else {
					got, err = server.DecodeEchoRequest(context.Background(), &pb.EchoRequest{Body: &body}, tc.metadata)
				}
				if tc.invalid {
					require.Error(t, err)
					require.Nil(t, got)
					if tc.parser {
						require.ErrorIs(t, err, scalar.ErrRejected)
					}
					return
				}
				require.NoError(t, err)
				value := got.(*svc.Value)
				require.Equal(t, tc.metadata.Get("required")[0], value.Required.Text)
				require.Equal(t, tc.optional, value.Optional)
				require.Equal(t, tc.defaulted, value.Defaulted.Text)
				require.Equal(t, tc.tokens, value.Tokens)
				require.Equal(t, uuid.MustParse("550e8400-e29b-41d4-a716-446655440000"), value.UUID)
			})
		}
	}
}

func TestMetadataPublication(t *testing.T) {
	empty := scalar.Token{}
	for _, optional := range []*scalar.Token{nil, &empty} {
		value := &svc.Value{
			Body:      "body",
			Required:  scalar.Token{Text: "required"},
			Optional:  optional,
			Defaulted: scalar.Token{},
			UUID:      uuid.Nil,
			Tokens:    []scalar.Token{{Text: "one"}, {Text: "two"}},
		}
		for _, response := range []bool{false, true} {
			hdr, trlr := metadata.MD{}, metadata.MD{}
			var err error
			if response {
				_, err = server.EncodeEchoResponse(context.Background(), value, &hdr, &trlr)
			} else {
				_, err = client.EncodeEchoRequest(context.Background(), value, &hdr)
			}
			require.NoError(t, err)
			require.Equal(t, []string{"required"}, hdr.Get("required"))
			require.Equal(t, []string{""}, hdr.Get("defaulted"))
			require.Equal(t, []string{uuid.Nil.String()}, hdr.Get("uuid"))
			require.Equal(t, []string{"one", "two"}, hdr.Get("tokens"))
			if optional == nil {
				require.Empty(t, hdr.Get("optional"))
			} else {
				require.Equal(t, []string{""}, hdr.Get("optional"))
			}
		}
	}
}

func TestMetadataRejectedCapabilitiesAndDefaults(t *testing.T) {
	decoded, err := server.DecodeBaddefaultRequest(context.Background(), nil, metadata.MD{})
	require.ErrorIs(t, err, scalar.ErrRejected)
	require.Nil(t, decoded)
	value := &svc.Value{Required: scalar.Token{Text: "present"}, Opaque: &scalar.Opaque{Number: 7}}
	for _, response := range []bool{false, true} {
		hdr, trlr := metadata.MD{}, metadata.MD{}
		var encoded any
		if response {
			encoded, err = server.EncodeEchoResponse(context.Background(), value, &hdr, &trlr)
		} else {
			encoded, err = client.EncodeEchoRequest(context.Background(), value, &hdr)
		}
		require.Error(t, err)
		require.Nil(t, encoded)
		require.Empty(t, hdr.Get("opaque"))
	}
}

func TestMetadataCLIConversion(t *testing.T) {
	value, err := client.BuildEchoPayload("{\"body\":\"body\"}", "present", "", "", "fallback", "550e8400-e29b-41d4-a716-446655440000", "[\"one\"]")
	require.NoError(t, err)
	require.Equal(t, "present", value.Required.Text)
	require.Equal(t, []scalar.Token{{Text: "one"}}, value.Tokens)
	value, err = client.BuildEchoPayload("{\"body\":\"body\"}", "bad", "", "", "fallback", "550e8400-e29b-41d4-a716-446655440000", "[]")
	require.ErrorIs(t, err, scalar.ErrRejected)
	require.Nil(t, value)
	rejected, err := client.BuildBaddefaultPayload("bad")
	require.ErrorIs(t, err, scalar.ErrRejected)
	require.Nil(t, rejected)
}
`
