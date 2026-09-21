package service

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/internal/designfingerprint"
)

func authorizationTestDSL() {
	ref := dsl.Type("DocumentRef", func() {
		dsl.Attribute("id", dsl.String)
		dsl.Required("id")
	})
	edit := dsl.Authorization("document.edit", ref)
	dsl.Service("documents", func() {
		dsl.StrictAuthorization()
		dsl.Method("update", func() {
			dsl.Payload(func() {
				dsl.Attribute("document_id", dsl.String)
				dsl.Required("document_id")
			})
			dsl.Authorize(edit, func() {
				dsl.Bind("id", "document_id")
			})
			dsl.Result(dsl.String)
		})
		dsl.Method("health", func() {
			dsl.NoAccessCheck("Public health probe")
		})
	})
}

func TestAuthorizationGeneratedContract(t *testing.T) {
	root := codegen.RunDSL(t, authorizationTestDSL)
	services := NewServicesData(root)
	svc := services.Get("documents")
	files := Files("example.com/access/gen", root.Services[0], services, make(map[string][]string))
	require.True(t, svc.HasAccessAuthorizer())
	var source bytes.Buffer
	for _, f := range files {
		if filepath.Base(f.Path) == "authorization.go" {
			for _, section := range f.AllSections()[1:] {
				require.NoError(t, section.Write(&source))
			}
		}
	}
	require.Contains(t, source.String(), "AuthorizeDocumentEdit(context.Context, *DocumentRef) error")
	require.Contains(t, source.String(), "input.ID = p.DocumentID")
	require.Contains(t, source.String(), "security.RequireAuthorizer(access)")
	assertGolden(t, source.String(), "testdata/golden/authorization.go.golden")
	endpoint := EndpointFile("example.com/access/gen", root.Services[0], services)
	generated := codegen.SectionCode(t, endpoint.Section("endpoints-init")[0])
	require.Contains(t, generated, "access AccessAuthorizer")
	generated = codegen.SectionCode(t, endpoint.Section("endpoints-use")[0])
	require.Contains(t, generated, "security.Protect(e.checkUpdate, e.Update, m)")
}

func TestAuthorizationManifestDeterministic(t *testing.T) {
	var previous []byte
	for range 2 {
		path := filepath.Join(t.TempDir(), "manifest.json")
		cmd := exec.Command(os.Args[0], "-test.run=^TestAuthorizationManifestProcess$")
		cmd.Env = append(os.Environ(), "LOOM_AUTHORIZATION_MANIFEST_TEST="+path)
		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "%s", output)
		current, err := os.ReadFile(path)
		require.NoError(t, err)
		if previous != nil {
			require.Equal(t, previous, current)
		}
		previous = current
	}
	require.Contains(t, string(previous), `"payload": "document_id"`)
	require.Contains(t, string(previous), `"exemption": "Public health probe"`)
}

func TestAuthorizationManifestProcess(t *testing.T) {
	path := os.Getenv("LOOM_AUTHORIZATION_MANIFEST_TEST")
	if path == "" {
		t.Skip("subprocess helper")
	}
	root := codegen.RunDSL(t, authorizationTestDSL)
	manifest := authorizationManifest(NewServicesData(root).Get("documents"))
	var content bytes.Buffer
	for _, section := range manifest.AllSections() {
		require.NoError(t, section.Write(&content))
	}
	require.NoError(t, os.WriteFile(path, content.Bytes(), 0o600))
}

func TestAuthorizationFingerprint(t *testing.T) {
	root := codegen.RunDSL(t, authorizationTestDSL)
	digest := func() string {
		d, err := designfingerprint.Digest(root, "gen", "example.com/access/gen", codegen.DesignVersion)
		require.NoError(t, err)
		return d
	}
	before := digest()
	root.Services[0].Methods[0].Authorization.Requirements[0].Bindings[0].Payload = "renamed"
	require.NotEqual(t, before, digest())
}

func TestAuthorizationDoesNotChangeSecurity(t *testing.T) {
	root := expr.RunDSL(t, func() {
		jwt := dsl.JWTSecurity("session")
		dsl.Service("profile", func() {
			dsl.Security(jwt)
			dsl.StrictAuthorization()
			dsl.Method("read", func() {
				dsl.Payload(func() {
					dsl.Token("token", dsl.String)
				})
				dsl.NoAccessCheck("Authenticated caller's own profile")
			})
		})
	})
	m := root.Services[0].Methods[0]
	require.Len(t, m.Requirements, 1)
	require.Equal(t, "session", m.Requirements[0].Schemes[0].SchemeName)
}
