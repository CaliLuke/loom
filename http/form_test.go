package http

import (
	"io"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/stretchr/testify/require"
)

type (
	testFormPayload struct {
		ClientID string   `form:"client_id"`
		Scope    []string `form:"scope,omitempty"`
		Active   *bool    `form:"active,omitempty"`
	}

	testGrant struct {
		kind         string
		Code         string
		RefreshToken string
	}
)

func (g testGrant) MarshalFormValues(values url.Values, prefix string) error {
	values.Set(FormChildKey(prefix, "type"), g.kind)
	switch g.kind {
	case "authorization_code":
		values.Set(FormChildKey(prefix, "value"), g.Code)
	case "refresh_token":
		values.Set(FormChildKey(prefix, "value"), g.RefreshToken)
	}
	return nil
}

func (g *testGrant) UnmarshalFormValues(values url.Values, prefix string) error {
	switch values.Get(FormChildKey(prefix, "type")) {
	case "authorization_code":
		var code string
		seen, err := DecodeFormValue(values, FormChildKey(prefix, "value"), &code)
		if err != nil {
			return err
		}
		if seen {
			g.kind = "authorization_code"
			g.Code = code
		}
	case "refresh_token":
		var token string
		seen, err := DecodeFormValue(values, FormChildKey(prefix, "value"), &token)
		if err != nil {
			return err
		}
		if seen {
			g.kind = "refresh_token"
			g.RefreshToken = token
		}
	}
	return nil
}

func TestEncodeFormValues(t *testing.T) {
	active := true
	values, err := EncodeFormValues(testFormPayload{
		ClientID: "client-123",
		Scope:    []string{"openid", "profile"},
		Active:   &active,
	})
	require.NoError(t, err)
	require.Equal(t, "client-123", values.Get("client_id"))
	require.Equal(t, []string{"openid", "profile"}, values["scope"])
	require.Equal(t, "true", values.Get("active"))
}

func TestDecodeFormValues(t *testing.T) {
	var payload testFormPayload
	err := DecodeFormValues(url.Values{
		"client_id": {"client-123"},
		"scope":     {"openid", "profile"},
		"active":    {"true"},
	}, &payload)
	require.NoError(t, err)
	require.Equal(t, "client-123", payload.ClientID)
	require.Equal(t, []string{"openid", "profile"}, payload.Scope)
	require.NotNil(t, payload.Active)
	require.True(t, *payload.Active)
}

func TestDecodeFormValuesInvalidScalar(t *testing.T) {
	var payload struct {
		Count int `form:"count"`
	}
	err := DecodeFormValues(url.Values{"count": {"nope"}}, &payload)
	require.Error(t, err)
}

func TestDecodeFormValuesUnion(t *testing.T) {
	var grant testGrant
	err := DecodeFormValues(url.Values{
		"type":  {"authorization_code"},
		"value": {"abc123"},
	}, &grant)
	require.NoError(t, err)
	require.Equal(t, "authorization_code", grant.kind)
	require.Equal(t, "abc123", grant.Code)
}

func TestSetFormRequest(t *testing.T) {
	req := httptest.NewRequest("POST", "/", nil)
	err := SetFormRequest(req, testGrant{kind: "refresh_token", RefreshToken: "rt-1"})
	require.NoError(t, err)
	require.Equal(t, "application/x-www-form-urlencoded", req.Header.Get("Content-Type"))
	body, err := req.GetBody()
	require.NoError(t, err)
	buf, err := io.ReadAll(body)
	require.NoError(t, err)
	require.Contains(t, string(buf), "type=refresh_token")
	require.Contains(t, string(buf), "value=rt-1")
}
