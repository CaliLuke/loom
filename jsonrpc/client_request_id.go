package jsonrpc

import (
	"context"
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"fmt"
	"net/http"
)

type clientNotificationKey struct{}

// SetClientRequestID encodes a generated client's service ID through its JSON
// codec and records whether the actual request is a notification. Presence is
// resolved by the generated field layout: absence and empty strings notify,
// while an explicit nullable null remains a request. The marker lets response
// handling use this decision without invoking the field codec a second time.
func SetClientRequestID(req *http.Request, message *Request, value any, present bool) error {
	message.ID = nil
	if present {
		encoded, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("encode JSON-RPC ID: %w", err)
		}
		if string(encoded) == "null" {
			message.ID = jsontext.Value("null")
		} else {
			var id string
			if err := json.Unmarshal(encoded, &id); err != nil {
				return fmt.Errorf("encode JSON-RPC String ID: %w", err)
			}
			if id != "" {
				message.ID = id
			}
		}
	}
	*req = *req.WithContext(context.WithValue(req.Context(), clientNotificationKey{}, message.ID == nil))
	return nil
}

// IsClientNotification reports the envelope decision made by SetClientRequestID.
// A request without that marker uses the ordinary response path.
func IsClientNotification(req *http.Request) bool {
	notification, _ := req.Context().Value(clientNotificationKey{}).(bool)
	return notification
}

// MarshalJSONTo preserves explicit null ID tokens while retaining the ordinary
// nil/empty String notification convention. Params preserves supplied empties.
func (r Request) MarshalJSONTo(enc *jsontext.Encoder) error {
	type plainRequest Request
	raw, explicitNull := r.ID.(jsontext.Value)
	if !explicitNull || string(raw) != "null" {
		return json.MarshalEncode(enc, plainRequest(r))
	}
	type envelope struct {
		JSONRPC string         `json:"jsonrpc"`
		Method  string         `json:"method"`
		Params  any            `json:"params,omitzero"`
		ID      jsontext.Value `json:"id"`
		View    ResultView     `json:"loom_view,omitempty"`
	}
	return json.MarshalEncode(enc, envelope{JSONRPC: r.JSONRPC, Method: r.Method, Params: r.Params, ID: raw, View: r.View})
}
