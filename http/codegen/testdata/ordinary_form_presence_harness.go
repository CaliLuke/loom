package testdata

const ordinaryFormPresenceHarness = `
func TestOrdinaryFormPresence(t *testing.T) {
	for _, name := range []string{
		"PlainEmpty", "PlainFields", "PlainRequiredFields", "PlainSelectedRequired",
		"PlainOptional", "PlainSelected", "PlainSelectedEmpty",
	} {
		for _, wire := range []string{
			"", "&&", "name=okay", "unknown=okay", "name=x", "name=%ZZ",
			"name=" + strings.Repeat("a", 64),
		} {
			t.Run(name+"/"+wire, func(t *testing.T) {
				calls := 0
				endpoint := func(_ context.Context, raw any) (any, error) {
					calls++
					var fields *svc.OrdinaryFormFields
					switch p := raw.(type) {
					case *svc.OrdinaryFormEmpty:
						require.NotNil(t, p)
					case *svc.OrdinaryFormRequiredFields:
						require.NotNil(t, p)
						if wire == "name=okay" {
							require.Equal(t, "okay", p.Name)
						} else {
							require.Equal(t, "x", p.Name)
						}
					case *svc.PlainSelectedRequiredPayload:
						require.NotNil(t, p)
						require.Equal(t, "kept", *p.Owner)
						fields = p.Value
						require.NotNil(t, fields)
					case *svc.OrdinaryFormFields:
						fields = p
						require.NotNil(t, p)
					case *svc.PlainSelectedPayload:
						require.NotNil(t, p)
						require.Equal(t, "kept", *p.Owner)
						fields = p.Value
						require.Equal(t, wire != "", fields != nil)
					case *svc.PlainSelectedEmptyPayload:
						require.NotNil(t, p)
						require.Equal(t, "kept", *p.Owner)
						require.Equal(t, wire != "", p.Value != nil)
					default:
						t.Errorf("unexpected payload %T", raw)
					}
					if fields != nil {
						if wire == "name=okay" {
							require.NotNil(t, fields.Name)
							require.Equal(t, "okay", *fields.Name)
						} else {
							require.Nil(t, fields.Name)
						}
					}
					return nil, nil
				}
				endpoints := &svc.Endpoints{
					PlainEmpty: endpoint, PlainFields: endpoint, PlainRequiredFields: endpoint,
					PlainSelectedRequired: endpoint, PlainOptional: endpoint,
					PlainSelected: endpoint, PlainSelectedEmpty: endpoint,
				}
				mux := loomhttp.NewMuxer()
				server.Mount(mux, server.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				policy, err := loomhttp.NewRequestBodyPolicy(32)
				require.NoError(t, err)
				req := httptest.NewRequest("POST", "/"+name+"?owner=kept", strings.NewReader(wire))
				req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
				req.ContentLength = -1
				rec := httptest.NewRecorder()
				policy.Handler(mux).ServeHTTP(rec, req)
				status, code, detail := 204, "", ""
				switch {
				case len(wire) > 32:
					status, code, detail = 413, "request_too_large", "request body too large"
				case wire == "name=%ZZ":
					status, code, detail = 400, "decode_payload", "invalid request body"
				case wire == "name=x" && name != "PlainEmpty" && name != "PlainSelectedEmpty" && name != "PlainRequiredFields":
					status, code, detail = 400, "invalid_length", "validation error"
				case name == "PlainRequiredFields" && (wire == "" || wire == "&&" || wire == "unknown=okay"):
					status, code, detail = 400, "missing_field", "Missing required field: name"
				}
				require.Equal(t, status, rec.Code, rec.Body.String())
				if status == 204 {
					require.Equal(t, 1, calls)
					require.Empty(t, rec.Body.String())
				} else {
					require.Zero(t, calls)
					var problem map[string]any
					require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &problem))
					require.Equal(t, code, problem["code"])
					require.Equal(t, detail, problem["detail"])
				}
			})
		}
	}
}

func TestOrdinaryFormClientRoundTrip(t *testing.T) {
	calls := 0
	endpoint := func(_ context.Context, raw any) (any, error) {
		calls++
		require.NotNil(t, raw.(*svc.OrdinaryFormEmpty))
		return nil, nil
	}
	selected := func(_ context.Context, raw any) (any, error) {
		calls++
		value := raw.(*svc.PlainSelectedEmptyPayload)
		require.Nil(t, value.Value)
		require.Equal(t, "kept", *value.Owner)
		return nil, nil
	}
	mux := loomhttp.NewMuxer()
	server.Mount(mux, server.New(&svc.Endpoints{PlainEmpty: endpoint, PlainSelectedEmpty: selected}, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
	owner := "kept"
	for _, tc := range []struct {
		name    string
		payload any
		encode  func(*http.Request, any) error
		sent    bool
	}{
		{"PlainEmpty", &svc.OrdinaryFormEmpty{}, client.EncodePlainEmptyRequest(loomhttp.RequestEncoder), true},
		{"PlainSelectedEmpty", &svc.PlainSelectedEmptyPayload{Value: &svc.OrdinaryFormEmpty{}, Owner: &owner}, client.EncodePlainSelectedEmptyRequest(loomhttp.RequestEncoder), true},
		{"PlainSelectedEmpty", &svc.PlainSelectedEmptyPayload{Owner: &owner}, client.EncodePlainSelectedEmptyRequest(loomhttp.RequestEncoder), false},
	} {
		request := httptest.NewRequest("POST", "/"+tc.name, nil)
		require.NoError(t, tc.encode(request, tc.payload))
		require.Zero(t, request.ContentLength)
		if tc.sent {
			require.Equal(t, "application/x-www-form-urlencoded", request.Header.Get("Content-Type"))
		} else {
			require.Empty(t, request.Header.Get("Content-Type"))
		}
		if request.Body == nil {
			request.Body = http.NoBody
		}
		response := httptest.NewRecorder()
		mux.ServeHTTP(response, request)
		require.Equal(t, 204, response.Code, response.Body.String())
	}
	require.Equal(t, 3, calls)
	request := httptest.NewRequest("POST", "/PlainSelectedEmpty", nil)
	err := client.EncodePlainSelectedEmptyRequest(loomhttp.RequestEncoder)(request, (*svc.PlainSelectedEmptyPayload)(nil))
	require.Error(t, err)
	require.Contains(t, err.Error(), "missing required payload")
	require.Empty(t, request.Header.Get("Content-Type"))
	require.Equal(t, 3, calls)
}
`
