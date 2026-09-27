package jsonrpc

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
)

// ResultView names the view used to project a streamed result. Its JSON value
// must be a string; explicit null and other token types are invalid. An empty
// value lets generated clients use the method's fixed or default view.
type ResultView string

// UnmarshalJSONFrom decodes a string view name, rejecting non-string metadata.
func (v *ResultView) UnmarshalJSONFrom(dec *jsontext.Decoder) error {
	if dec.PeekKind() != '"' {
		return errors.New("loom_view must be a JSON string")
	}
	return json.UnmarshalDecode(dec, (*string)(v))
}
