package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	servicedata "github.com/CaliLuke/loom/codegen/service/testdata"
)

const credentialTypesHarness = `package credentials_test

import (
	"context"
	"reflect"
	"testing"

	svc "example.com/credentials/gen/credentials"
	client "example.com/credentials/gen/grpc/credentials/client"
	server "example.com/credentials/gen/grpc/credentials/server"
	"google.golang.org/grpc/metadata"
)

func TestCredentialMetadata(t *testing.T) {
	user, pass, key, token, access := svc.Credential("user"), svc.Credential("pass"), svc.Credential("key"), svc.Credential("token"), svc.Credential("access")
	for _, tc := range []struct {
		name    string
		payload any
		encode  func(context.Context, any, *metadata.MD) (any, error)
		decode  func(context.Context, any, metadata.MD) (any, error)
	}{
		{"required", &svc.RequiredPayload{Login: user, Secret: pass, AccessKey: key, JWTValue: token, OAuthValue: access}, client.EncodeRequiredRequest, server.DecodeRequiredRequest},
		{"optional", &svc.OptionalPayload{Login: &user, Secret: &pass, AccessKey: &key, JWTValue: &token, OAuthValue: &access}, client.EncodeOptionalRequest, server.DecodeOptionalRequest},
		{"absent", &svc.OptionalPayload{}, client.EncodeOptionalRequest, server.DecodeOptionalRequest},
		{"ordinary", &svc.OrdinaryPayload{User: string(user), Pass: string(pass), Key: string(key), Token: string(token), Access: string(access)}, client.EncodeOrdinaryRequest, server.DecodeOrdinaryRequest},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			md := metadata.MD{}
			request, err := tc.encode(ctx, tc.payload, &md)
			if err != nil {
				t.Fatal(err)
			}
			decoded, err := tc.decode(ctx, request, md)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(tc.payload, decoded) {
				t.Errorf("decoded = %#v, want %#v", decoded, tc.payload)
			}
		})
	}
}

func TestCredentialCLI(t *testing.T) {
	required, err := client.BuildRequiredPayload("user", "pass", "key", "token", "access")
	if err != nil {
		t.Fatal(err)
	}
	if required.Login != "user" || required.Secret != "pass" || required.AccessKey != "key" || required.JWTValue != "token" || required.OAuthValue != "access" {
		t.Errorf("required CLI payload = %#v", required)
	}
	optional, err := client.BuildOptionalPayload("", "", "", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if optional.Login != nil || optional.Secret != nil || optional.AccessKey != nil || optional.JWTValue != nil || optional.OAuthValue != nil {
		t.Errorf("omitted CLI values became present: %#v", optional)
	}
}
`

func TestNamedCredentialMetadataGenerated(t *testing.T) {
	root := RunGRPCDSL(t, servicedata.NamedGRPCCredentialFieldsDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/credentials", root)
	for _, file := range ClientCLIFiles("example.com/credentials/gen", CreateGRPCServices(root)) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	require.NoError(t, os.WriteFile(filepath.Join(dir, "credentials_test.go"), []byte(credentialTypesHarness), 0o600))
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-run", "^TestCredential(Metadata|CLI)$", "./...")
}

func TestConstrainedCredentialsGenerated(t *testing.T) {
	root := RunGRPCDSL(t, servicedata.ConstrainedCredentialsDSL)
	dir := t.TempDir()
	renderGRPCResponseContractModule(t, dir, "example.com/credentials", root)
	for _, file := range ClientCLIFiles("example.com/credentials/gen", CreateGRPCServices(root)) {
		_, err := file.Render(dir)
		require.NoError(t, err)
	}
	runGRPCGoCommand(t, dir, "mod", "tidy")
	runGRPCGoCommand(t, dir, "test", "-run", "^$", "./...")
}
