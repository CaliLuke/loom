package vet

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/expr"
	externalsarif "github.com/owenrumney/go-sarif/v3/pkg/report/v210/sarif"
	"github.com/stretchr/testify/require"
)

func TestAnalyzeHTTPErrorSemantics(t *testing.T) {
	service := &expr.ServiceExpr{Name: "widgets"}
	method := &expr.MethodExpr{Name: "show", Service: service}
	endpoint := &expr.HTTPEndpointExpr{MethodExpr: method}
	httpService := &expr.HTTPServiceExpr{ServiceExpr: service, HTTPEndpoints: []*expr.HTTPEndpointExpr{endpoint}}
	endpoint.Service = httpService
	internal := httpError("internal", 500, nil, nil)
	unavailable := httpError("unavailable", 503, expr.MetaExpr{"loom:error:fault": nil}, nil)
	limited := httpError("limited", 429, expr.MetaExpr{"loom:error:temporary": nil}, nil)
	suppressed := httpError("suppressed", 502, expr.MetaExpr{SuppressionMeta: []string{"all"}}, nil)
	hinted := httpError(
		"hinted",
		503,
		expr.MetaExpr{"loom:error:fault": nil},
		&expr.ErrorRemedyExpr{RetryHint: "Retry after the dependency recovers."},
	)
	endpoint.HTTPErrors = []*expr.HTTPErrorExpr{internal, unavailable, limited, suppressed, hinted}
	duplicateEndpoint := &expr.HTTPEndpointExpr{
		MethodExpr: &expr.MethodExpr{Name: "list", Service: service},
		Service:    httpService,
		HTTPErrors: []*expr.HTTPErrorExpr{internal},
	}
	httpService.HTTPEndpoints = append(httpService.HTTPEndpoints, duplicateEndpoint)
	root := &expr.RootExpr{
		API:      &expr.APIExpr{HTTP: &expr.HTTPExpr{Services: []*expr.HTTPServiceExpr{httpService}}},
		Services: []*expr.ServiceExpr{service},
		Errors:   []*expr.ErrorExpr{internal.ErrorExpr, unavailable.ErrorExpr, limited.ErrorExpr, suppressed.ErrorExpr, hinted.ErrorExpr},
	}

	var report Report
	analyzeHTTPErrorSemantics(root, &report)

	require.Equal(t, []Diagnostic{
		{
			Rule:     RuleServerErrorFault,
			Severity: SeverityError,
			Message:  `HTTP 500 error "internal" must declare Fault()`,
			Location: Location{Path: "api.error.internal"},
		},
		{
			Rule:     RuleRetryableError,
			Severity: SeverityError,
			Message:  "HTTP 503 error \"unavailable\" must declare Temporary() or Remedy(func() {\n    RetryHint(\"...\")\n})",
			Location: Location{Path: "api.error.unavailable"},
		},
	}, report.Diagnostics)
}

func TestAnalyzeAttributeSemantics(t *testing.T) {
	emailType := &expr.UserTypeExpr{
		TypeName: "Email",
		AttributeExpr: &expr.AttributeExpr{
			Type:       expr.String,
			Validation: &expr.ValidationExpr{Format: expr.FormatEmail},
		},
	}
	object := expr.Object{
		&expr.NamedAttributeExpr{Name: "track", Attribute: &expr.AttributeExpr{Type: expr.Int, Description: "1-based track index."}},
		&expr.NamedAttributeExpr{Name: "offset", Attribute: &expr.AttributeExpr{Type: expr.Int, Description: "Zero-based offset."}},
		&expr.NamedAttributeExpr{Name: "confidence", Attribute: &expr.AttributeExpr{Type: expr.Float64, Description: "Normalized confidence."}},
		&expr.NamedAttributeExpr{Name: "callback_url", Attribute: &expr.AttributeExpr{Type: expr.String, Description: "Callback URL."}},
		&expr.NamedAttributeExpr{Name: "validated_email", Attribute: &expr.AttributeExpr{Type: emailType, Description: "Email address."}},
		&expr.NamedAttributeExpr{Name: "denormalized_count", Attribute: &expr.AttributeExpr{Type: expr.Int, Description: "Denormalized count."}},
		&expr.NamedAttributeExpr{Name: "bounded", Attribute: &expr.AttributeExpr{
			Type:        expr.Int,
			Description: "Value from 2 to 4.",
			Validation:  &expr.ValidationExpr{Minimum: new(2.0), Maximum: new(4.0)},
		}},
		&expr.NamedAttributeExpr{Name: "ignored_url", Attribute: &expr.AttributeExpr{
			Type:        expr.String,
			Description: "Optional URL sentinel.",
			Meta:        expr.MetaExpr{SuppressionMeta: []string{RuleStringFormat}},
		}},
	}
	errorObject := expr.Object{
		&expr.NamedAttributeExpr{Name: "callback_url", Attribute: &expr.AttributeExpr{Type: expr.String, Description: "Callback URL."}},
	}
	inlineError := &expr.ErrorExpr{
		Name:          "callback_failed",
		AttributeExpr: &expr.AttributeExpr{Type: &errorObject},
	}
	root := &expr.RootExpr{
		Types: []expr.UserType{
			emailType,
			&expr.UserTypeExpr{
				TypeName:      "Metrics",
				AttributeExpr: &expr.AttributeExpr{Type: &object},
			},
		},
		Errors: []*expr.ErrorExpr{inlineError},
	}

	var report Report
	analyzeAttributeSemantics(root, &report)

	require.ElementsMatch(t, []string{
		RuleDescriptionMinimum + ":type.Metrics.track",
		RuleDescriptionMinimum + ":type.Metrics.offset",
		RuleNormalizedRange + ":type.Metrics.confidence",
		RuleStringFormat + ":type.Metrics.callback_url",
		RuleStringFormat + ":api.error.callback_failed.callback_url",
	}, diagnosticKeys(report.Diagnostics))
}

func TestAnalyzeModuleFindsTypedMuxRoutes(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/service\n\ngo 1.27.0\n\nrequire github.com/CaliLuke/loom v1.2.3\nreplace github.com/CaliLuke/loom => ./.loomstub\n")
	writeTestFile(t, filepath.Join(root, ".loomstub", "go.mod"), "module github.com/CaliLuke/loom\n\ngo 1.27.0\n")
	writeTestFile(t, filepath.Join(root, ".loomstub", "http", "http.go"), `package http

import "net/http"

type Muxer interface {
	Handle(string, string, http.HandlerFunc)
}

type ResolverMuxer interface {
	Muxer
}

type mux struct{}

func (*mux) Handle(string, string, http.HandlerFunc) {}

func NewMuxer() ResolverMuxer { return &mux{} }
`)
	writeTestFile(t, filepath.Join(root, "router.go"), `package service

import (
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

type otherMux struct{}

func (otherMux) Handle(string, string, http.HandlerFunc) {}

func otherRouter() otherMux { return otherMux{} }

func muxHelper() loomhttp.Muxer { return loomhttp.NewMuxer() }

type routeHolder struct {
	Mux loomhttp.Muxer
}

func routes() {
	mux := loomhttp.NewMuxer()
	mux.Handle("GET", "/manual", nil)
	{
		mux := otherRouter()
		mux.Handle("GET", "/not-loom", nil)
	}
	mux.Handle("GET", "/after-shadow", nil)
	var typed loomhttp.Muxer
	typed.Handle("GET", "/typed", nil)
	muxHelper().Handle("GET", "/helper", nil)
	holder := routeHolder{Mux: loomhttp.NewMuxer()}
	holder.Mux.Handle("GET", "/field", nil)
	//loom:vet ignore route-outside-design -- intentional probe
	mux.Handle("GET", "/probe", nil)
}
`)
	writeTestFile(t, filepath.Join(root, "broken.go"), `package service

var _ = missingSymbol
`)
	writeTestFile(t, filepath.Join(root, "generated.go"), `// Code generated by Loom, DO NOT EDIT.
package service

import loomhttp "github.com/CaliLuke/loom/http"

func mount(mux loomhttp.Muxer) { mux.Handle("GET", "/generated", nil) }
`)
	writeTestFile(t, filepath.Join(root, "nested", "go.mod"), "module example.com/nested\n\nrequire github.com/CaliLuke/loom v9.9.9\n")
	writeTestFile(t, filepath.Join(root, "nested", "gen", "loom.json"), `{"loom_version":"v0.0.1"}`)
	writeTestFile(t, filepath.Join(root, "nested", "router.go"), `package nested

import loomhttp "github.com/CaliLuke/loom/http"

func routes() {
	mux := loomhttp.NewMuxer()
	mux.Handle("GET", "/nested", nil)
}
`)

	diagnostics, err := analyzeModule(root)
	require.NoError(t, err)
	require.Len(t, diagnostics, 5)
	routeMessages := make([]string, 0, 5)
	for _, diagnostic := range diagnostics {
		require.Equal(t, RuleRouteOutsideDesign, diagnostic.Rule)
		routeMessages = append(routeMessages, diagnostic.Message)
	}
	require.ElementsMatch(t, []string{
		"route GET /manual is registered directly on a Loom mux; declare and mount it through the design",
		"route GET /after-shadow is registered directly on a Loom mux; declare and mount it through the design",
		"route GET /typed is registered directly on a Loom mux; declare and mount it through the design",
		"route GET /helper is registered directly on a Loom mux; declare and mount it through the design",
		"route GET /field is registered directly on a Loom mux; declare and mount it through the design",
	}, routeMessages)
}

func TestAnalyzeGeneratedVersionsFindsSkew(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.2.3\n")
	writeTestFile(t, filepath.Join(root, "gen", "loom.json"), `{"loom_version":"v1.2.2"}`)

	diagnostics, err := analyzeGeneratedVersions(root)

	require.NoError(t, err)
	require.Equal(t, []string{RuleGeneratedVersionSkew + ":gen/loom.json"}, diagnosticKeys(diagnostics))
}

func TestAnalyzeGeneratedVersionsSkipsLocalReplace(t *testing.T) {
	root := t.TempDir()
	writeTestFile(t, filepath.Join(root, "go.mod"), "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.2.3\nreplace github.com/CaliLuke/loom => ../loom\n")
	writeTestFile(t, filepath.Join(root, "gen", "loom.json"), `{"loom_version":"v1.9.0"}`)

	diagnostics, err := analyzeGeneratedVersions(root)

	require.NoError(t, err)
	require.Empty(t, diagnostics)
}

func TestAnalyzeGeneratedVersionsResolvesSelectedRequirement(t *testing.T) {
	tests := []struct {
		name     string
		goMod    string
		manifest string
	}{
		{
			name:     "pseudo-version is not comparable",
			goMod:    "module example.com/service\n\nrequire github.com/CaliLuke/loom v0.0.0-20260824010101-abcdefabcdef\n",
			manifest: "v1.8.0",
		},
		{
			name:     "unselected version replacement is ignored",
			goMod:    "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.3.0\nreplace github.com/CaliLuke/loom v1.2.0 => ../loom-old\n",
			manifest: "v1.3.0",
		},
		{
			name:     "selected version replacement is used",
			goMod:    "module example.com/service\n\nrequire github.com/CaliLuke/loom v1.3.0\nreplace github.com/CaliLuke/loom v1.3.0 => example.com/loom-fork v1.4.0\n",
			manifest: "v1.4.0",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			writeTestFile(t, filepath.Join(root, "go.mod"), test.goMod)
			writeTestFile(t, filepath.Join(root, "gen", "loom.json"), `{"loom_version":"`+test.manifest+`"}`)

			diagnostics, err := analyzeGeneratedVersions(root)

			require.NoError(t, err)
			require.Empty(t, diagnostics)
		})
	}
}

func TestWriteReportFormats(t *testing.T) {
	report := Report{Diagnostics: []Diagnostic{{
		Rule:     RuleRouteOutsideDesign,
		Severity: SeverityError,
		Message:  "manual route",
		Location: Location{Path: "router.go", Line: 12, Column: 3},
	}}}

	var text bytes.Buffer
	require.NoError(t, WriteReport(&text, report, FormatText))
	require.Equal(t, "router.go:12:3: error[route-outside-design]: manual route\n", text.String())

	var jsonOutput bytes.Buffer
	require.NoError(t, WriteReport(&jsonOutput, report, FormatJSON))
	var decoded Report
	require.NoError(t, json.Unmarshal(jsonOutput.Bytes(), &decoded))
	require.Equal(t, report, decoded)

	var emptyJSON bytes.Buffer
	require.NoError(t, WriteReport(&emptyJSON, Report{}, FormatJSON))
	require.JSONEq(t, `{"diagnostics":[]}`, emptyJSON.String())

	var sarifOutput bytes.Buffer
	require.NoError(t, WriteReport(&sarifOutput, report, FormatSARIF))
	var sarif sarifLog
	require.NoError(t, json.Unmarshal(sarifOutput.Bytes(), &sarif))
	require.Equal(t, "2.1.0", sarif.Version)
	require.Equal(t, "router.go", sarif.Runs[0].Results[0].Locations[0].PhysicalLocation.ArtifactLocation.URI)
	externalSourceReport, err := externalsarif.FromBytes(sarifOutput.Bytes())
	require.NoError(t, err)
	require.NoError(t, externalSourceReport.Validate())

	graphReport := Report{Diagnostics: []Diagnostic{{
		Rule:     RuleServerErrorFault,
		Severity: SeverityError,
		Message:  "missing fault metadata",
		Location: Location{Path: "api.error.internal"},
	}}}
	var graphSARIFOutput bytes.Buffer
	require.NoError(t, WriteReport(&graphSARIFOutput, graphReport, FormatSARIF))
	var graphSARIF sarifLog
	require.NoError(t, json.Unmarshal(graphSARIFOutput.Bytes(), &graphSARIF))
	require.Equal(t, "api.error.internal", graphSARIF.Runs[0].Results[0].Locations[0].LogicalLocations[0].FullyQualifiedName)
	externalGraphReport, err := externalsarif.FromBytes(graphSARIFOutput.Bytes())
	require.NoError(t, err)
	require.NoError(t, externalGraphReport.Validate())
}

func httpError(name string, status int, meta expr.MetaExpr, remedy *expr.ErrorRemedyExpr) *expr.HTTPErrorExpr {
	errorExpr := &expr.ErrorExpr{
		Name: name,
		AttributeExpr: &expr.AttributeExpr{
			Type: expr.ErrorResult,
			Meta: meta,
		},
		Remedy: remedy,
	}
	return &expr.HTTPErrorExpr{
		ErrorExpr: errorExpr,
		Name:      name,
		Response:  &expr.HTTPResponseExpr{StatusCode: status},
	}
}

func diagnosticKeys(diagnostics []Diagnostic) []string {
	keys := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		keys = append(keys, diagnostic.Rule+":"+diagnostic.Location.Path)
	}
	return keys
}

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o750))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
}
