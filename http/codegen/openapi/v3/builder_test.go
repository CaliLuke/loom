package openapiv3

import (
	"fmt"
	"testing"

	"goa.design/goa/v3/codegen"
	"goa.design/goa/v3/expr"
	"goa.design/goa/v3/http/codegen/openapi"
	"goa.design/goa/v3/http/codegen/openapi/v3/testdata/dsls"
	"goa.design/goa/v3/http/codegen/testdata"
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
				expected = "Goa API"
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
		ExpectedResponses:  responses{"200": {"OK response.", tobj("result", tobj("children", tarray)), nil}},
	}, {
		Name: "response_recursive_user_type",
		DSL:  dsls.ResponseRecursiveUserType(svcName, "response_recursive_user_type"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {"OK response.", tobj("recursive", tobj()), nil}},
	}, {
		Name: "response_recursive_array_user_type",
		DSL:  dsls.ResponseRecursiveArrayUserType(svcName, "response_recursive_array_user_type"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {"OK response.", tobj("result", tobj("children", tarray)), nil}},
	}, {
		Name: "response_skip_response_body_encode_decode",
		DSL:  dsls.ResponseSkipResponseBodyEncodeDecode(svcName, "response_skip_response_body_encode_decode"),

		ExpectedDeprecated: false,
		ExpectedResponses:  responses{"200": {"OK response.", tbinary, nil}},
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

func TestBuildOperationID(t *testing.T) {
	const svcName = "test service"

	cases := []struct {
		Name    string
		Service string
		DSL     func()

		ExpectedOperationIDs []string
	}{
		{
			Name:                 "template_in_method",
			Service:              svcName,
			DSL:                  dsls.OperationIDMethod(svcName, "template_in_method", "{method}"),
			ExpectedOperationIDs: []string{"template_in_method"},
		}, {
			Name:                 "template_in_service",
			Service:              svcName,
			DSL:                  dsls.OperationIDService(svcName, "template_in_service", "{service}"),
			ExpectedOperationIDs: []string{"test_service"},
		}, {
			Name:                 "template_in_api",
			Service:              svcName,
			DSL:                  dsls.OperationIDAPI(svcName, "template_in_api", defaultOperationIDFormat),
			ExpectedOperationIDs: []string{"test_service.template_in_api"},
		}, {
			Name:                 "multiple_routes",
			Service:              svcName,
			DSL:                  dsls.OperationIDMultipleRoutes(svcName, "multiple_routes", defaultOperationIDFormat),
			ExpectedOperationIDs: []string{"test_service.multiple_routes", "test_service.multiple_routes.1"},
		}, {
			Name:                 "multiple_routes_custom_separator",
			Service:              svcName,
			DSL:                  dsls.OperationIDMultipleRoutes(svcName, "multiple_routes_custom_separator", "{service}.{method}(.{routeIndex})"),
			ExpectedOperationIDs: []string{"test_service.multiple_routes_custom_separator", "test_service.multiple_routes_custom_separator.1"},
		}, {
			Name:                 "multiple_routes_custom_separator_without_routeIndex",
			Service:              svcName,
			DSL:                  dsls.OperationIDMultipleRoutes(svcName, "multiple_routes_custom_separator_without_routeIndex", "{service}.{method}"),
			ExpectedOperationIDs: []string{"test_service.multiple_routes_custom_separator_without_route_index", "test_service.multiple_routes_custom_separator_without_route_index#1"},
		}, {
			Name:                 "multiple_routes_no_routeIndex_separator",
			Service:              svcName,
			DSL:                  dsls.OperationIDMultipleRoutes(svcName, "multiple_routes_no_routeIndex_separator", "{service}.{method}({routeIndex})"),
			ExpectedOperationIDs: []string{"test_service.multiple_routes_no_route_index_separator", "test_service.multiple_routes_no_route_index_separator1"},
		}, {
			Name:                 "multiple_routes_long_separator",
			Service:              svcName,
			DSL:                  dsls.OperationIDMultipleRoutes(svcName, "multiple_routes_long_separator", "{service}.{method}(someWordsHere and maybe some spaces{routeIndex})"),
			ExpectedOperationIDs: []string{"test_service.multiple_routes_long_separator", "test_service.multiple_routes_long_separatorsomeWordsHere and maybe some spaces1"},
		}, {
			Name:                 "custom_static_operation_id",
			Service:              svcName,
			DSL:                  dsls.OperationIDMethod(svcName, "custom_static_operation_id", "listThings"),
			ExpectedOperationIDs: []string{"listThings"},
		}, {
			Name:                 "canonicalize_method_and_service_placeholders",
			Service:              "Camel Case Service",
			DSL:                  dsls.OperationIDMethod("Camel Case Service", "OAuth2UserInfo", "{service}.{method}"),
			ExpectedOperationIDs: []string{"camel_case_service.oauth2_user_info"},
		}, {
			Name:                 "canonicalize_file_server_path_placeholders",
			Service:              "File Service",
			DSL:                  dsls.OperationIDService("File Service", "/assets/{*filepath}", "{service}.{method}"),
			ExpectedOperationIDs: []string{"file_service.assets_filepath"},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			api := codegen.RunDSL(t, c.DSL).API

			if len(api.HTTP.Services) == 0 {
				t.Error("no HTTP service created from DSL")
			}

			for _, s := range api.HTTP.Services {
				if s.Name() == c.Service {
					for _, e := range s.HTTPEndpoints {
						for i, r := range e.Routes {
							op := buildOperation(c.Name, r, &EndpointBodies{}, expr.NewRandom(c.Name), api.Meta)

							if len(c.ExpectedOperationIDs) == 0 {
								t.Error("no expected operation IDs")
								return
							}

							if op.OperationID != c.ExpectedOperationIDs[i] {
								t.Errorf("got operation ID %q, expected %q", op.OperationID, c.ExpectedOperationIDs[i])
								return
							}
						}
					}
				}
			}
		})
	}
}

func TestCanonicalOperationIDComponent(t *testing.T) {
	cases := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "service name", input: "test service", expected: "test_service"},
		{name: "camel case", input: "OAuth2UserInfo", expected: "oauth2_user_info"},
		{name: "path-like", input: "/assets/{*filepath}", expected: "assets_filepath"},
		{name: "punctuation-only", input: "{}/*-", expected: "operation"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actual := canonicalOperationIDComponent(tc.input)
			if actual != tc.expected {
				t.Errorf("got canonical operation ID component %q, expected %q", actual, tc.expected)
			}
		})
	}
}

func TestBuildOperationErrorRemedyDescription(t *testing.T) {
	const (
		svcName = "test service"
		metName = "error_remedy"
	)

	root := codegen.RunDSL(t, dsls.ErrorRemedyResponseBodyDSL(svcName, metName))
	bodies, _ := buildBodyTypes(root.API, root.Types, root.ResultTypes)
	endpointBodies := bodies[svcName][metName]

	var route *expr.RouteExpr
	for _, svc := range root.API.HTTP.Services {
		if svc.Name() != svcName {
			continue
		}
		route = svc.Endpoint(metName).Routes[0]
		break
	}
	if route == nil {
		t.Fatal("could not find route")
	}

	op := buildOperation(metName, route, endpointBodies, expr.NewRandom(metName), root.API.Meta)
	resp := op.Responses["400"]
	if resp == nil || resp.Value == nil || resp.Value.Description == nil {
		t.Fatal("missing bad request response description")
	}

	expected := "bad: Bad Request response. Remedy code: bad.fix. Safe message: Retry with a valid request. Retry hint: Correct the payload and retry."
	if *resp.Value.Description != expected {
		t.Errorf("got response description %q, expected %q", *resp.Value.Description, expected)
	}
}

func TestNewDeduplicatesRepeatedParametersIntoComponents(t *testing.T) {
	root := codegen.RunDSL(t, testdata.OpenAPIParameterComponentsDSL)

	spec := New(root)
	if spec == nil {
		t.Fatal("expected OpenAPI spec")
	}
	if spec.Components == nil {
		t.Fatal("expected components")
	}

	widgetIDName := parameterComponentName(&Parameter{Name: "widgetID", In: "path"})
	limitName := parameterComponentName(&Parameter{Name: "limit", In: "query"})

	if len(spec.Components.Parameters) != 2 {
		t.Fatalf("got %d parameter components, expected 2", len(spec.Components.Parameters))
	}
	if spec.Components.Parameters[widgetIDName] == nil || spec.Components.Parameters[widgetIDName].Value == nil {
		t.Fatalf("missing component parameter %q", widgetIDName)
	}
	if spec.Components.Parameters[limitName] == nil || spec.Components.Parameters[limitName].Value == nil {
		t.Fatalf("missing component parameter %q", limitName)
	}

	for _, path := range []string{"/widgets/{widgetID}", "/gadgets/{widgetID}"} {
		op := spec.Paths[path].Get
		if op == nil {
			t.Fatalf("missing GET operation for %s", path)
		}
		if len(op.Parameters) != 2 {
			t.Fatalf("got %d parameters for %s, expected 2", len(op.Parameters), path)
		}
		if op.Parameters[0].Ref != parameterComponentRefPrefix+widgetIDName || op.Parameters[0].Value != nil {
			t.Fatalf("expected first parameter for %s to reference %q, got ref=%q value=%#v", path, parameterComponentRefPrefix+widgetIDName, op.Parameters[0].Ref, op.Parameters[0].Value)
		}
		if op.Parameters[1].Ref != parameterComponentRefPrefix+limitName || op.Parameters[1].Value != nil {
			t.Fatalf("expected second parameter for %s to reference %q, got ref=%q value=%#v", path, parameterComponentRefPrefix+limitName, op.Parameters[1].Ref, op.Parameters[1].Value)
		}
	}
}

func matchesParameter(t *testing.T, p *ParameterRef, types map[string]*openapi.Schema, expected param) {
	matchesParameterHeader(t, p, types, expected, "parameter")
}
func matchesParameterHeader(t *testing.T, p *ParameterRef, types map[string]*openapi.Schema, expected param, title string) {
	if p.Value == nil {
		t.Errorf("no value for %s", title)
		return
	}
	if p.Ref != "" {
		t.Errorf("got ref %q for %s %q, expected none", p.Ref, title, p.Value.Name)
	}
	v := p.Value
	if v.Name != expected.Name {
		t.Errorf("got %s name %q, expected %q", title, v.Name, expected.Name)
	}
	if v.In != expected.In {
		t.Errorf("got %s in %q, expected %q", title, v.In, expected.In)
	}
	if v.Description != expected.Description {
		t.Errorf("got %s description %q, expected %q", title, v.Description, expected.Description)
	}
	if v.Style != expected.Style {
		t.Errorf("got %s style %q, expected %q", title, v.Style, expected.Style)
	}
	if v.Required != expected.Required {
		t.Errorf("got %s required %v, expected %v", title, v.Required, expected.Required)
	}
	matchesSchema(t, fmt.Sprintf("%s %q", title, v.Name), v.Schema, types, expected.Type)
	if v.Content != nil {
		t.Errorf("got content %#v, expected none", v.Content)
	}
}

func matchesRequestBody(t *testing.T, b *RequestBodyRef, types map[string]*openapi.Schema, expected *requestBody) {
	if b == nil {
		if expected != nil {
			t.Error("request body is nil")
		}
		return
	}
	if b.Value == nil {
		t.Error("no value for request body")
		return
	}
	if b.Ref != "" {
		t.Errorf("got ref %q for request body, expected none", b.Ref)
	}
	v := b.Value
	if v.Description != expected.Description {
		t.Errorf("got request body description %q, expected %q", v.Description, expected.Description)
	}
	if v.Required != expected.Required {
		t.Errorf("got request body required %v, expected %v", v.Required, expected.Required)
	}
	ct, ok := v.Content["application/json"]
	if !ok {
		t.Error("missing request content, expected application/json")
		return
	}
	matchesSchema(t, "request body", ct.Schema, types, expected.Type)
}

func matchesResponse(t *testing.T, r *ResponseRef, types map[string]*openapi.Schema, expected response) {
	if r.Value == nil {
		t.Error("no value for response")
		return
	}
	if r.Ref != "" {
		t.Errorf("got ref %q for response, expected none", r.Ref)
	}
	v := r.Value
	if v.Description == nil && expected.Description != "" {
		t.Errorf("got no response description, expected %q", expected.Description)
	} else if *v.Description != expected.Description {
		t.Errorf("got response description %q, expected %q", *v.Description, expected.Description)
	}
	if len(v.Headers) != len(expected.Headers) {
		t.Errorf("got %d response header(s), expected %d", len(v.Headers), len(expected.Headers))
		return
	}
	for n, h := range v.Headers {
		exp, ok := expected.Headers[n]
		if !ok {
			t.Errorf("response header %q not expected", n)
		}
		matchesHeader(t, h, types, exp)
	}
	if expected.Type.Type != "" {
		ct, ok := v.Content["application/json"]
		if !ok {
			t.Error("missing response content, expected application/json")
			return
		}
		matchesSchema(t, "response body", ct.Schema, types, expected.Type)
	}
}

func matchesHeader(t *testing.T, h *HeaderRef, types map[string]*openapi.Schema, expected param) {
	if h.Value == nil {
		t.Error("no value for header")
		return
	}
	if h.Ref != "" {
		t.Errorf("got ref %q for header, expected none", h.Ref)
	}
	v := h.Value
	par := &ParameterRef{Value: &Parameter{
		Description:     v.Description,
		Style:           v.Style,
		Explode:         v.Explode,
		AllowEmptyValue: v.AllowEmptyValue,
		AllowReserved:   v.AllowReserved,
		Deprecated:      v.Deprecated,
		Required:        v.Required,
		Schema:          v.Schema,
		Example:         v.Example,
		Examples:        v.Examples,
		Content:         v.Content,
		Extensions:      v.Extensions,
		In:              "header",
	}}
	matchesParameterHeader(t, par, types, expected, "header")
}
