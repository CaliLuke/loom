package expr

import (
	"strings"

	"github.com/CaliLuke/loom/eval"
	"github.com/CaliLuke/loom/internal/httpcodec"
)

func (e *HTTPEndpointExpr) validateRequestContentType(errors *eval.ValidationErrors) {
	if e.RequestContentType == nil {
		return
	}
	media := *e.RequestContentType
	if media == "" || httpcodec.RequestContentType(media).Kind != httpcodec.JSON {
		errors.Add(e, "request ContentType requires a concrete JSON media type, got %q", media)
	}
	_, rpc := e.MethodExpr.Meta["jsonrpc"]
	if e.FormRequest || e.MultipartRequest || e.SkipRequestBodyEncodeDecode || rpc ||
		(e.MethodExpr.IsStreaming() && e.SSE == nil) || httpRequestBody(e).Type == Empty {
		errors.Add(e, "request ContentType requires a typed JSON request body without form, multipart, raw, JSON-RPC or WebSocket framing")
	}
	for _, binding := range e.ensureWireBindings().bindings {
		if binding.location == "header" && strings.EqualFold(binding.name, "Content-Type") {
			errors.Add(e, "request ContentType cannot coexist with a mapped Content-Type header")
			break
		}
	}
}
