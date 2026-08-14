package openapiv3

import (
	"testing"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/openapi"
	"github.com/CaliLuke/loom/http/codegen/openapi/v3/testdata/dsls"
	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestBuildInfo(t *testing.T) {
	const (
		title        = "test title"
		description  = "test description"
		terms        = "test terms of service"
		version      = "test version"
		contactName  = "test contact name"
		contactEmail = "test contact email"
		contactURL   = "test contact URL"
		licenseName  = "test license name"
		licenseURL   = "test license URL"
	)
	cases := []struct {
		Name           string
		Title          string
		Description    string
		TermsOfService string
		Version        string
		ContactName    string
		ContactEmail   string
		ContactURL     string
		LicenseName    string
		LicenseURL     string
	}{{
		Name:           "simple",
		Title:          title,
		Description:    description,
		TermsOfService: terms,
		Version:        version,
		ContactName:    contactName,
		ContactEmail:   contactEmail,
		ContactURL:     contactURL,
		LicenseName:    licenseName,
		LicenseURL:     licenseURL,
	}, {
		Name:  "empty version",
		Title: title,
	}, {
		Name:    "empty title",
		Version: version,
	}}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			api := &expr.APIExpr{
				Name:           c.Name,
				Title:          c.Title,
				Description:    c.Description,
				TermsOfService: c.TermsOfService,
				Version:        c.Version,
				Contact:        &expr.ContactExpr{Name: contactName, Email: contactEmail, URL: contactURL},
				License:        &expr.LicenseExpr{Name: licenseName, URL: licenseURL},
			}

			info := buildInfo(api)

			expected := c.Title
			if api.Title == "" {
				expected = "Loom API"
			}
			if info.Title != expected {
				t.Errorf("got API title %q, expected %q", info.Title, expected)
			}

			if info.Description != c.Description {
				t.Errorf("got API description %q, expected %q", info.Description, c.Description)
			}

			if info.TermsOfService != c.TermsOfService {
				t.Errorf("got API terms of service %q, expected %q", info.TermsOfService, c.TermsOfService)
			}

			if info.Version != c.Version {
				t.Errorf("got API version %q, expected %q", info.Version, c.Version)
			}
		})
	}
}

func TestBuildServersPreservesHostMetadata(t *testing.T) {
	tests := []struct {
		name              string
		serverDescription string
		hostDescription   string
		wantDescription   string
	}{
		{
			name:              "host description takes precedence",
			serverDescription: "Hosts the device intelligence endpoints.",
			hostDescription:   "Shared staging environment.",
			wantDescription:   "Shared staging environment.",
		},
		{
			name:              "server description is the fallback",
			serverDescription: "Hosts the device intelligence endpoints.",
			wantDescription:   "Hosts the device intelligence endpoints.",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			variables := expr.Object{
				&expr.NamedAttributeExpr{
					Name: "subdomain",
					Attribute: &expr.AttributeExpr{
						Type:         expr.String,
						Description:  "Stage subdomain.",
						DefaultValue: "saas-api.stage",
					},
				},
			}
			servers := buildServers([]*expr.ServerExpr{
				{
					Name:        "device-intelligence",
					Description: test.serverDescription,
					Hosts: []*expr.HostExpr{
						{
							Name:        "stage",
							Description: test.hostDescription,
							URIs:        []expr.URIExpr{"https://{subdomain}.incode.com"},
							Variables:   &expr.AttributeExpr{Type: &variables},
						},
					},
				},
			})

			require.Len(t, servers, 1)
			require.Equal(t, test.wantDescription, servers[0].Description)
			require.Equal(t, "Stage subdomain.", servers[0].Variables["subdomain"].Description)
		})
	}
}

func TestBuildPathsIncludesCORSExtension(t *testing.T) {
	root := expr.RunDSL(t, testdata.ServerCORSPolicyDSL)
	doc := New(root)
	require.NotNil(t, doc)
	path := doc.Paths["/items"]
	require.NotNil(t, path)
	cors, ok := path.Extensions["x-loom-cors"].(map[string]any)
	require.True(t, ok)
	origins, ok := cors["origins"].([]map[string]any)
	require.True(t, ok)
	require.Len(t, origins, 1)
	require.Equal(t, "https://app.example.com", origins[0]["origin"])
	require.Equal(t, true, origins[0]["credentials"])
	require.Equal(t, 600, origins[0]["maxAge"])
}

func TestBuildPathsMarksRuntimeCORSWithoutValues(t *testing.T) {
	root := expr.RunDSL(t, testdata.ServerRuntimeCORSPolicyDSL)
	doc := New(root)
	path := doc.Paths["/items"]
	require.NotNil(t, path)
	cors, ok := path.Extensions["x-loom-cors"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, cors["runtime"])
	require.NotContains(t, cors, "origins")
}

type param struct {
	Name        string
	In          string
	Description string
	Style       string
	Required    bool
	Type        typ
}

type requestBody struct {
	Description string
	Type        typ
	Required    bool
}

type response struct {
	Description string
	ContentType string
	Type        typ
	Headers     map[string]param
}

type responses map[string]response

func TestBuildOperation(t *testing.T) {
	const svcName = "test service"
	cases := []struct {
		Name string
		DSL  func()

		ExpectedDescription string
		ExpectedDeprecated  bool
		ExpectedParameters  []param
		ExpectedRequestBody *requestBody
		ExpectedResponses   map[string]response
	}{{
		Name: "desc_only",
		DSL:  dsls.DescOnly(svcName, "desc_only", "desc"),

		ExpectedDescription: "desc",
		ExpectedDeprecated:  false,
		ExpectedResponses:   responses{"204": {Description: "No Content response."}},
	}, {
		Name:               "deprecated_only",
		DSL:                dsls.DeprecatedOnly(svcName, "deprecated_only"),
		ExpectedDeprecated: true,
		ExpectedResponses:  responses{"204": {Description: "No Content response."}},
	}, {
		Name: "request_string_body",
		DSL:  dsls.RequestStringBody(svcName, "request_string_body"),

		ExpectedDeprecated:  false,
		ExpectedRequestBody: &requestBody{"body", tstring, true},
		ExpectedResponses:   responses{"204": {Description: "No Content response."}},
	}, {
		Name: "request_object_body",
		DSL:  dsls.RequestObjectBody(svcName, "request_object_body"),

		ExpectedDeprecated:  false,
		ExpectedRequestBody: &requestBody{"", tobj("name", tstring), true},
		ExpectedResponses:   responses{"204": {Description: "No Content response."}},
	}, {
		Name: "request_optional_object_body",
		DSL:  dsls.RequestOptionalObjectBody(svcName, "request_optional_object_body"),

		ExpectedDeprecated:  false,
		ExpectedRequestBody: &requestBody{"", tobj("name", tstring), false},
		ExpectedResponses:   responses{"204": {Description: "No Content response."}},
	}, {
		Name: "request_streaming_string_body",
		DSL:  dsls.RequestObjectBody(svcName, "request_streaming_string_body"),

		ExpectedDeprecated:  false,
		ExpectedRequestBody: &requestBody{"", tobj("name", tstring), true},
		ExpectedResponses:   responses{"204": {Description: "No Content response."}},
	}, {
		Name: "request_map_params",
		DSL:  dsls.RequestMapParams(svcName, "request_map_params"),

		ExpectedDeprecated: false,
		ExpectedParameters: []param{{Name: "param", In: "query", Description: "Query parameters", Style: "deepObject", Type: tobj()}},
		ExpectedResponses:  responses{"204": {Description: "No Content response."}},
	}, {
		Name: "response_array_of_string",
		DSL:  dsls.ResponseArrayOfString(svcName, "response_array_of_string"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {Description: "OK response.", Type: tobj("result", tobj("children", tarray))}},
	}, {
		Name: "response_recursive_user_type",
		DSL:  dsls.ResponseRecursiveUserType(svcName, "response_recursive_user_type"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {Description: "OK response.", Type: tobj("recursive", tobj())}},
	}, {
		Name: "response_recursive_array_user_type",
		DSL:  dsls.ResponseRecursiveArrayUserType(svcName, "response_recursive_array_user_type"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {Description: "OK response.", Type: tobj("result", tobj("children", tarray))}},
	}, {
		Name: "response_skip_response_body_encode_decode",
		DSL:  dsls.ResponseSkipResponseBodyEncodeDecode(svcName, "response_skip_response_body_encode_decode"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {Description: "OK response.", Type: tbinary, ContentType: "application/json"}},
	}, {
		Name: "response_skip_response_body_encode_decode_openapi_body",
		DSL:  dsls.ResponseSkipResponseBodyEncodeDecodeOpenAPIBody(svcName, "response_skip_response_body_encode_decode_openapi_body"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {Description: "OK response.", Type: tstring, ContentType: "text/html"}},
	}}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			root := codegen.RunDSL(t, c.DSL)

			var bodies *EndpointBodies
			var types map[string]*openapi.Schema
			{
				var bds map[string]map[string]*EndpointBodies
				bds, types = buildBodyTypes(root.API, root.Types, root.ResultTypes)
				if svc, ok := bds[svcName]; ok {
					bodies, ok = svc[c.Name]
					if !ok {
						t.Error("bodies does not contain method details")
						return
					}
				}
			}

			var route *expr.RouteExpr
			if len(root.API.HTTP.Services) == 0 {
				t.Error("no HTTP service created from DSL")
			}
			for _, s := range root.API.HTTP.Services {
				if s.Name() == svcName {
					for _, e := range s.HTTPEndpoints {
						if e.Name() == c.Name {
							route = e.Routes[0]
							break
						}
					}
				}
				if route != nil {
					break
				}
			}
			if route == nil {
				t.Error("could not find route")
				return
			}

			op := buildOperation(c.Name, route, bodies, expr.NewRandom(c.Name), root.API.Meta)

			if op.Description != c.ExpectedDescription {
				t.Errorf("got description %q for method %q, expected %q", op.Description, c.Name, c.ExpectedDescription)
			}
			if len(op.Parameters) != len(c.ExpectedParameters) {
				t.Errorf("got %d parameters, expected %d", len(op.Parameters), len(c.ExpectedParameters))
				return
			}

			if op.Deprecated != c.ExpectedDeprecated {
				t.Errorf("got %t deprecated, expected %t", op.Deprecated, c.ExpectedDeprecated)
			}

			for i, p := range op.Parameters {
				matchesParameter(t, p, types, c.ExpectedParameters[i])
			}
			matchesRequestBody(t, op.RequestBody, types, c.ExpectedRequestBody)
			if len(op.Responses) != len(c.ExpectedResponses) {
				t.Errorf("got %d responses, expected %d", len(op.Responses), len(c.ExpectedResponses))
				return
			}
			for s, r := range op.Responses {
				matchesResponse(t, r, types, c.ExpectedResponses[s])
			}
		})
	}
}

func TestCollapseSchemaAliasesRewritesOperationRefs(t *testing.T) {
	paths := map[string]*PathItem{
		"/invites/redeem": {
			Post: &Operation{
				Responses: map[string]*ResponseRef{
					"201": {
						Value: &Response{
							Content: map[string]*MediaType{
								"application/json": {
									Schema: &openapi.Schema{Ref: "#/components/schemas/RestInvitesRedeemResponseBody"},
								},
							},
						},
					},
				},
			},
		},
	}
	schemas := map[string]*openapi.Schema{
		"RestInvitesRedeemResponseBody": {Ref: "#/components/schemas/RedeemInviteResponse"},
		"RedeemInviteResponse": {
			Type: openapi.Object,
			Properties: map[string]*openapi.Schema{
				"authenticated": {Type: openapi.Boolean},
			},
		},
	}

	collapseSchemaAliases(paths, schemas, reusableComponents{})

	if _, ok := schemas["RestInvitesRedeemResponseBody"]; ok {
		t.Fatal("expected pure-ref alias schema to be removed")
	}
	got := paths["/invites/redeem"].Post.Responses["201"].Value.Content["application/json"].Schema.Ref
	if want := "#/components/schemas/RedeemInviteResponse"; got != want {
		t.Fatalf("got rewritten response ref %q, want %q", got, want)
	}
}

func TestPruneUnusedComponentSchemasKeepsReachableNestedRefs(t *testing.T) {
	paths := map[string]*PathItem{
		"/sessions": {
			Get: &Operation{
				Responses: map[string]*ResponseRef{
					"200": {
						Value: &Response{
							Content: map[string]*MediaType{
								"application/json": {
									Schema: &openapi.Schema{Ref: "#/components/schemas/SessionEnvelope"},
								},
							},
						},
					},
				},
			},
		},
	}
	schemas := map[string]*openapi.Schema{
		"SessionEnvelope": {
			Type: openapi.Object,
			Properties: map[string]*openapi.Schema{
				"session": {Ref: "#/components/schemas/Session"},
			},
		},
		"Session": {
			Type: openapi.Object,
			Properties: map[string]*openapi.Schema{
				"user": {Ref: "#/components/schemas/User"},
			},
		},
		"User": {
			Type: openapi.Object,
			Properties: map[string]*openapi.Schema{
				"id": {Type: openapi.String},
			},
		},
		"Unused": {
			Type: openapi.Object,
			Properties: map[string]*openapi.Schema{
				"note": {Type: openapi.String},
			},
		},
	}

	pruned := pruneUnusedComponentSchemas(paths, schemas, reusableComponents{})

	if len(pruned) != 3 {
		t.Fatalf("got %d schemas after pruning, want 3", len(pruned))
	}
	if _, ok := pruned["SessionEnvelope"]; !ok {
		t.Fatal("expected root response schema to remain")
	}
	if _, ok := pruned["Session"]; !ok {
		t.Fatal("expected nested schema to remain")
	}
	if _, ok := pruned["User"]; !ok {
		t.Fatal("expected transitive nested schema to remain")
	}
	if _, ok := pruned["Unused"]; ok {
		t.Fatal("expected unreferenced schema to be pruned")
	}
}

func TestNewBuildsReusableContractComponentsAndServiceTags(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIReusableComponentsDSL)

	spec := New(root)
	if spec == nil {
		t.Fatal("expected spec")
	}
	if spec.Components == nil {
		t.Fatal("expected components")
	}
	if len(spec.Components.RequestBodies) == 0 {
		t.Fatal("expected reusable request body components")
	}
	if len(spec.Components.Responses) == 0 {
		t.Fatal("expected reusable response components")
	}
	if len(spec.Components.Headers) == 0 {
		t.Fatal("expected reusable header components")
	}
	if len(spec.Components.Examples) == 0 {
		t.Fatal("expected reusable example components")
	}

	postSignin := spec.Paths["/auth/signin"].Post
	postRefresh := spec.Paths["/auth/refresh"].Post
	postRevoke := spec.Paths["/auth/revoke"].Post
	postSignout := spec.Paths["/auth/signout"].Post
	if postSignin == nil || postRefresh == nil || postRevoke == nil || postSignout == nil {
		t.Fatal("expected auth operations")
	}
	if postSignin.RequestBody == nil || postRefresh.RequestBody == nil {
		t.Fatal("expected reusable request body refs")
	}
	if postSignin.RequestBody.Ref == "" || postRefresh.RequestBody.Ref == "" || postSignin.RequestBody.Ref != postRefresh.RequestBody.Ref {
		t.Fatalf("unexpected request body refs: signin=%#v refresh=%#v", postSignin.RequestBody, postRefresh.RequestBody)
	}
	if postRevoke.Responses["204"] == nil || postSignout.Responses["204"] == nil {
		t.Fatal("expected reusable 204 responses")
	}
	if postRevoke.Responses["204"].Ref == "" || postRevoke.Responses["204"].Ref != postSignout.Responses["204"].Ref {
		t.Fatalf("unexpected 204 response refs: revoke=%#v signout=%#v", postRevoke.Responses["204"], postSignout.Responses["204"])
	}
	if got := postSignin.Tags; len(got) != 1 || got[0] != "Auth" {
		t.Fatalf("unexpected signin tags: %#v", got)
	}
	if got := postRefresh.Tags; len(got) != 1 || got[0] != "Auth" {
		t.Fatalf("unexpected refresh tags: %#v", got)
	}
}

func TestNewAppliesMethodBodyOperationTags(t *testing.T) {
	root := codegen.RunDSL(t, testdata.MethodBodyTagsDSL)

	spec := New(root)
	if spec == nil {
		t.Fatal("expected spec")
	}
	update := spec.Paths["/stats"].Post
	list := spec.Paths["/stats"].Get
	if update == nil || list == nil {
		t.Fatal("expected tagged operations")
	}
	if got := update.Tags; len(got) != 2 || got[0] != "Device" || got[1] != "B2B" {
		t.Errorf("unexpected update tags: %#v", got)
	}
	if got := list.Tags; len(got) != 1 || got[0] != "Internal" {
		t.Errorf("unexpected list tags: %#v", got)
	}
}
