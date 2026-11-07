package testdata

import . "github.com/CaliLuke/loom/dsl"

func untaggedMultipartObjectPresenceMethods() func() {
	basic := BasicAuthSecurity("object_presence_basic")
	return func() {
		for _, tc := range []struct {
			name                                                    string
			requiredMember, bindings, optionalBindings, credentials bool
		}{
			{name: "OptionalObject"},
			{name: "OptionalObjectRequiredMember", requiredMember: true},
			{name: "OptionalObjectBindings", bindings: true},
			{name: "OptionalObjectOptionalBindings", bindings: true, optionalBindings: true},
			{name: "OptionalObjectCredentials", credentials: true},
		} {
			Method(tc.name, func() {
				if tc.credentials {
					Security(basic)
				}
				Payload(func() {
					Attribute("name", String)
					Attribute("count", Int, func() {
						Default(7)
					})
					if tc.requiredMember {
						Required("name")
					}
					if tc.bindings {
						Attribute("query", String)
						Attribute("token", String)
						if !tc.optionalBindings {
							Required("query", "token")
						}
					}
					if tc.credentials {
						Username("user")
						Password("pass")
						Required("user", "pass")
					}
				})
				HTTP(func() {
					POST("/" + tc.name)
					MultipartRequest()
					OptionalRequestBody()
					if tc.bindings {
						Param("query")
						Header("token:X-Token")
					}
					Response(StatusNoContent)
				})
			})
		}
	}
}

const optionalObjectPresenceHarness = `
func TestOptionalObjectSourcePresence(t *testing.T) {
	for _, tc := range []struct {
		name, path, raw, media                    string
		multipart, bindings, credentials, payload bool
		fields                                    [][2]string
		count                                     int
		code, detail                              string
	}{
		{name: "multipart absent", path: "/OptionalObject"},
		{name: "multipart empty", path: "/OptionalObject", multipart: true, payload: true, count: 7},
		{name: "multipart supplied zero", path: "/OptionalObject", multipart: true, fields: [][2]string{{"count", "0"}}, payload: true},
		{name: "multipart malformed", path: "/OptionalObject", raw: "broken", media: "multipart/form-data; boundary=missing", code: "decode_payload", detail: "invalid request body"},
		{name: "member absent body", path: "/OptionalObjectRequiredMember"},
		{name: "member empty body", path: "/OptionalObjectRequiredMember", multipart: true, code: "missing_field", detail: "Missing required field: name"},
		{name: "member supplied", path: "/OptionalObjectRequiredMember", multipart: true, fields: [][2]string{{"name", "value"}}, payload: true, count: 7},
		{name: "bindings absent body", path: "/OptionalObjectBindings?query=value", bindings: true, payload: true},
		{name: "bindings missing", path: "/OptionalObjectBindings", code: "missing_field", detail: "Missing required field: query"},
		{name: "optional bindings absent", path: "/OptionalObjectOptionalBindings", payload: true},
		{name: "credentials absent body", path: "/OptionalObjectCredentials", credentials: true, payload: true},
		{name: "credentials missing", path: "/OptionalObjectCredentials", code: "missing_field", detail: "Missing required field: Authorization"},
		{name: "json absent", path: "/JSONOptionalObject"},
		{name: "json whitespace", path: "/JSONOptionalObject", raw: " \n\t", media: "application/json"},
		{name: "json empty object", path: "/JSONOptionalObject", raw: "{}", media: "application/json", payload: true},
		{name: "json null", path: "/JSONOptionalObject", raw: "null", media: "application/json", code: "decode_payload", detail: "invalid request body"},
		{name: "json malformed", path: "/JSONOptionalObject", raw: "{x}", media: "application/json", code: "decode_payload", detail: "invalid request body"},
		{name: "json truncated", path: "/JSONOptionalObject", raw: "{", media: "application/json", code: "decode_payload", detail: "invalid request body"},
		{name: "required json absent", path: "/JSONRequiredObject", code: "missing_payload", detail: "validation error"},
		{name: "required json null", path: "/JSONRequiredObject", raw: "null", media: "application/json", code: "decode_payload", detail: "invalid request body"},
		{name: "form absent", path: "/FormOptionalObject"},
		{name: "form supplied empty keys", path: "/FormOptionalObject", raw: "&", media: "application/x-www-form-urlencoded", payload: true},
		{name: "form malformed", path: "/FormOptionalObject", raw: "%zz", media: "application/x-www-form-urlencoded", code: "decode_payload", detail: "invalid request body"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			var received any
			endpoint := func(_ context.Context, raw any) (any, error) {
				calls++
				received = raw
				return nil, nil
			}
			endpoints := &svc.Endpoints{
				OptionalObject: endpoint, OptionalObjectRequiredMember: endpoint, OptionalObjectBindings: endpoint,
				OptionalObjectOptionalBindings: endpoint, OptionalObjectCredentials: endpoint,
				JSONOptionalObject: endpoint, JSONRequiredObject: endpoint, FormOptionalObject: endpoint,
			}
			mux := loomhttp.NewMuxer()
			server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
			var input io.Reader
			if tc.raw != "" {
				input = strings.NewReader(tc.raw)
			}
			request := httptest.NewRequest("POST", tc.path, input)
			if tc.multipart {
				request = multipartRequest(t, tc.path, tc.fields)
			}
			if tc.media != "" {
				request.Header.Set("Content-Type", tc.media)
			}
			if tc.bindings {
				request.Header.Set("X-Token", "secret")
			}
			if tc.credentials {
				request.SetBasicAuth("alice", "secret")
			}
			response := httptest.NewRecorder()
			mux.ServeHTTP(response, request)
			if tc.code != "" {
				require.Equal(t, http.StatusBadRequest, response.Code, response.Body.String())
				require.Zero(t, calls)
				require.Nil(t, received)
				var problem loomhttp.ProblemResponse
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
				require.Equal(t, tc.code, problem.Code)
				require.Contains(t, problem.Detail, tc.detail)
				return
			}
			require.Equal(t, http.StatusNoContent, response.Code, response.Body.String())
			require.Equal(t, 1, calls)
			require.Empty(t, response.Body.String())
			value := reflect.ValueOf(received)
			require.Equal(t, reflect.Pointer, value.Kind())
			require.Equal(t, !tc.payload, value.IsNil(), "typed nil must distinguish absence from supplied empty object")
			if !tc.payload {
				return
			}
			object := value.Elem()
			if field := object.FieldByName("Count"); field.IsValid() {
				require.Equal(t, int64(tc.count), field.Int(), "body defaults apply only when supplied")
			}
			if tc.bindings {
				require.Equal(t, "value", object.FieldByName("Query").String())
				require.Equal(t, "secret", object.FieldByName("Token").String())
			}
			if tc.credentials {
				require.Equal(t, "alice", object.FieldByName("User").String())
				require.Equal(t, "secret", object.FieldByName("Pass").String())
			}
		})
	}
}
`
