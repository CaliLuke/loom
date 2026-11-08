package openapiimport

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestImportProducerServers(t *testing.T) {
	for _, tc := range []struct{ name, version, servers string }{
		{"named 3.2", "3.2.0", "  - {name: 'test api', url: 'http://localhost:80', description: 'Default server for test api'}\n"},
		{"ordered https", "3.2.0", "  - {name: primary, url: 'https://api.example.test/v1'}\n  - {name: backup, url: 'http://backup.example.test:8080'}\n"},
		{"same name retains order", "3.2.0", "  - {name: shared, url: 'https://one.example.test'}\n  - {name: shared, url: 'https://two.example.test'}\n"},
		{"3.1 unnamed", "3.1.1", "  - {url: 'https://api.example.test/v1'}\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(producerServersSource(tc.version, tc.servers))
			require.NoError(t, err)
			require.Empty(t, diagnostics)
			rendered, err := Render(document, Options{})
			require.NoError(t, err)
			require.Contains(t, string(rendered), "Server(")
			if tc.name == "named 3.2" {
				require.Contains(t, string(rendered), `Server("test api", func()`)
				require.Contains(t, string(rendered), `URI("http://localhost:80")`)
				requireRenderedDesignEvaluates(t, rendered, 1, `
 servers := expr.Root.API.Servers
 if len(servers) != 1 { t.Fatalf("server count: %d", len(servers)) }
 server := servers[0]
 if server.Name != "test api" || len(server.Hosts) != 1 || len(server.Services) != 1 { t.Fatalf("lost server owner: %#v", server) }
 host := server.Hosts[0]
 if host.Description != "Default server for test api" || len(host.URIs) != 1 || string(host.URIs[0]) != "http://localhost:80" { t.Fatalf("changed authored server: %#v", host) }
`)
			}
			if tc.name == "ordered https" {
				require.Less(t, strings.Index(string(rendered), "https://api.example.test/v1"), strings.Index(string(rendered), "http://backup.example.test:8080"))
			}
		})
	}
}

func TestImportProducerServersRejectUnsupported(t *testing.T) {
	for _, tc := range []struct{ name, server string }{
		{"relative", "{name: relative, url: /v1}"},
		{"variables", "{name: templated, url: 'https://{host}', variables: {host: {default: api.example.test}}}"},
		{"unresolved template", "{name: templated, url: 'https://{host}'}"},
		{"userinfo", "{name: private, url: 'https://user:secret@example.test'}"},
		{"query", "{name: query, url: 'https://example.test?x=1'}"},
		{"fragment", "{name: fragment, url: 'https://example.test/#part'}"},
		{"scheme", "{name: other, url: 'ws://example.test'}"},
		{"extension", "{name: extension, url: 'https://example.test', x-private: true}"},
		{"missing name", "{url: 'https://example.test'}"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			document, diagnostics, err := Analyze(producerServersSource("3.2.0", "  - "+tc.server+"\n"))
			require.NoError(t, err)
			require.Contains(t, diagnostics.Error(), "servers")
			_, err = Render(document, Options{})
			require.Error(t, err)
		})
	}
	t.Run("operation override", func(t *testing.T) {
		source := string(producerServersSource("3.2.0", "  - {name: root, url: 'https://example.test'}\n"))
		source = strings.Replace(source, "      operationId: value", "      servers: [{url: 'https://override.example.test'}]\n      operationId: value", 1)
		_, diagnostics, err := Analyze([]byte(source))
		require.NoError(t, err)
		require.Contains(t, diagnostics.Error(), "operation servers")
	})
}

func producerServersSource(version, servers string) []byte {
	return []byte("openapi: " + version + "\ninfo: {title: Servers, version: '1'}\nservers:\n" + servers + "paths:\n  /value:\n    get:\n      operationId: value\n      responses:\n        '200': {description: value, content: {application/json: {schema: {type: string}}}}\n")
}

func TestImportProducerServersPartialOmission(t *testing.T) {
	source := producerServersSource("3.2.0", "  - {name: valid, url: 'https://example.test'}\n  - {name: invalid, url: '/relative'}\n")
	analysis, _, err := AnalyzePartial(Source{Bytes: source}, Selection{}, false)
	require.NoError(t, err)
	require.Empty(t, analysis.Blocked)
	require.Len(t, analysis.Document.Operations, 1)
	requireDiagnosticCode(t, analysis.Omitted, "servers")
	rendered, err := Render(analysis.Document, Options{})
	require.NoError(t, err)
	require.Contains(t, string(rendered), `URI("https://example.test")`)
	require.NotContains(t, string(rendered), "/relative")
}
