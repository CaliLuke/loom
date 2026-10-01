package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
)

func TestRetainedUntaggedBytesCLIExampleGeneratedJSONRPC(t *testing.T) {
	const modulePath = "example.com/valuecontractcli"

	root := RunJSONRPCDSL(t, valueContractCLIExampleDSL)
	dir := t.TempDir()
	renderJSONRPCModule(t, dir, modulePath, root)
	renderCodegenFiles(t, dir, ClientCLIFiles(modulePath+"/gen", CreateJSONRPCServices(root)))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "value_contract_cli_test.go"), []byte(valueContractCLIJSONRPCHarness), 0o600))

	runGoJSONRPCTestCommand(t, dir, "mod", "tidy")
	runGoJSONRPCTestCommand(t, dir, "test", "-count=1", ".")
}

func valueContractCLIExampleDSL() {
	API("valuecontractcli", func() {
		JSONRPC(func() {})
		Server("valuecontractcli", func() {
			Services("sender")
			Host("default", func() {
				URI("http://localhost")
			})
		})
	})
	data := Type("Data", func() {
		Attribute("bytes", Bytes)
		Required("bytes")
	})
	other := Type("Other", func() {
		Attribute("count", Int)
		Required("count")
	})
	Service("sender", func() {
		JSONRPC(func() {
			POST("/rpc")
		})
		Method("send", func() {
			Payload(OneOf(data, other), func() {
				Untagged()
				Example(map[string]any{"bytes": "hi"})
			})
			JSONRPC(func() {})
		})
		Method("batch", func() {
			Payload(ArrayOf(Bytes), func() {
				Example([]any{"hi"})
			})
			JSONRPC(func() {})
		})
		Method("suppressed", func() {
			Payload(ArrayOf(Bytes), func() {
				Meta("openapi:example", "false")
			})
			JSONRPC(func() {})
		})
	})
}

const valueContractCLIJSONRPCHarness = `package valuecontractcli

import (
	"bytes"
	"flag"
	"net/http"
	"strings"
	"testing"

	cli "example.com/valuecontractcli/gen/jsonrpc/cli/valuecontractcli"
	sender "example.com/valuecontractcli/gen/sender"
	client "example.com/valuecontractcli/gen/jsonrpc/sender/client"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestAdvertisedExampleBuildsRetainedBranch(t *testing.T) {
	_, err := client.BuildSendPayload("{x}")
	if err == nil {
		t.Fatal("invalid JSON succeeded")
	}
	_, advertised, ok := strings.Cut(err.Error(), "example of valid JSON:\n")
	if !ok {
		t.Fatalf("diagnostic %q has no advertised example", err)
	}
	advertised = strings.Trim(advertised, "'")
	if !strings.Contains(advertised, "\"bytes\": \"aGk=\"") {
		t.Fatalf("advertised example %q does not contain projected bytes", advertised)
	}
	payload, err := client.BuildSendPayload(advertised)
	if err != nil {
		t.Fatalf("advertised example: %v", err)
	}
	if payload.Kind() != sender.DataOrOtherKindData {
		t.Fatalf("branch %q, want Data", payload.Kind())
	}
	data, ok := payload.AsData()
	if !ok || !bytes.Equal(data.Bytes, []byte("hi")) {
		t.Fatalf("data %#v, present %t", data, ok)
	}
}

func TestDirectPayloadAdvertisedExample(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	_, err := parseDirectPayload(t, "batch", "{x}")
	if err == nil {
		t.Fatal("invalid JSON succeeded")
	}
	_, advertised, ok := strings.Cut(err.Error(), "example of valid JSON:\n")
	if !ok {
		t.Fatalf("diagnostic %q has no advertised example", err)
	}
	advertised = strings.Trim(advertised, "'")
	if !strings.Contains(advertised, "aGk=") {
		t.Fatalf("advertised example %q does not contain projected bytes", advertised)
	}
	payload, err := parseDirectPayload(t, "batch", advertised)
	if err != nil {
		t.Fatalf("advertised example: %v", err)
	}
	values, ok := payload.([][]byte)
	if !ok || len(values) != 1 || !bytes.Equal(values[0], []byte("hi")) {
		t.Fatalf("payload %#v", payload)
	}
}

func TestSuppressedDirectPayloadOmitsHint(t *testing.T) {
	oldFlags := flag.CommandLine
	t.Cleanup(func() {
		flag.CommandLine = oldFlags
	})
	_, err := parseDirectPayload(t, "suppressed", "{x}")
	if err == nil {
		t.Fatal("invalid JSON succeeded")
	}
	if strings.Contains(err.Error(), "example of valid JSON") {
		t.Fatalf("suppressed diagnostic advertises an example: %q", err)
	}
}

func parseDirectPayload(t *testing.T, method, raw string) (any, error) {
	t.Helper()
	flag.CommandLine = flag.NewFlagSet("direct-payload", flag.ContinueOnError)
	if err := flag.CommandLine.Parse([]string{"sender", method, "--p=" + raw}); err != nil {
		t.Fatalf("parse flags: %v", err)
	}
	_, payload, err := cli.ParseEndpoint("http", "localhost", http.DefaultClient, loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
	return payload, err
}
`
