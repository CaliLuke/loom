package representation

import (
	"net/textproto"

	"github.com/CaliLuke/loom/codegen/service"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpcodec"
)

func attachResponseTargets(endpoint *transportir.Endpoint, status *transportir.ResponseStatus, source *service.ValueData, schemaOnly bool) {
	codec, boundary := responseCodec(endpoint, status)
	status.BodyValue = ValueTarget(source, status.Body, status.BodyOrigin, codec, schemaOnly)
	if status.BodyValue != nil && boundary != "" {
		status.BodyValue.Boundary = boundary
	}
	status.BodyValues = mediaTargets(status.BodyValue, status.ContentTypes)
	if status.IndependentDocumentBody {
		status.DocumentValue = DocumentTarget(source, status.DocumentBody, status.BodyOrigin, true, schemaOnly)
		status.DocumentValues = documentationTargets(status.DocumentValue, status.ContentTypes)
		return
	}
	status.DocumentValue = ValueTarget(source, status.DocumentBody, status.BodyOrigin, codec, schemaOnly)
	if status.DocumentValue != nil && boundary != "" {
		status.DocumentValue.Boundary = boundary
	}
	// Streaming roots retain their physical message owner and projection handling.
	// Ordinary responses use the complete media set, never an extra JSON root.
	if endpoint.Stream == nil || endpoint.Stream.SSE == nil || endpoint.Stream.HasMixedResults || status.IsError {
		status.DocumentValues = mediaTargets(status.DocumentValue, status.ContentTypes)
	}
}

func responseCodec(endpoint *transportir.Endpoint, status *transportir.ResponseStatus) (expr.ValueCodec, string) {
	if status.BinaryBody || endpoint.Response.SkipBodyEncode || endpoint.Response.FileResponse {
		return expr.ValueCodecRaw, "response body bypasses the built-in encoder"
	}
	if endpoint.Stream != nil && endpoint.Stream.SSE != nil && !endpoint.Stream.HasMixedResults && !status.IsError {
		return expr.ValueCodecJSON, ""
	}
	if status.ContentType != "" {
		selected := httpcodec.ResponseContentType(status.ContentType)
		switch selected.Kind {
		case httpcodec.JSON:
			return expr.ValueCodecJSON, ""
		case httpcodec.Text:
			return expr.ValueCodecText, "response uses the built-in text encoder"
		default:
			return expr.ValueCodecCustom, "response encoder is outside built-in JSON correspondence"
		}
	}
	for _, header := range status.Headers {
		if textproto.CanonicalMIMEHeaderKey(header.HTTPName) == "Content-Type" {
			// Generated code selects encoder(ctx, w) before setting mapped headers.
			// A header enum describes labels, not the Accept input that chose the codec.
			return expr.ValueCodecCustom, "mapped Content-Type does not determine the negotiated response encoder"
		}
	}
	// The default advertised JSON contract describes the default Accept path.
	// It does not claim that other negotiated representations also use JSON.
	return expr.ValueCodecJSON, ""
}

func mediaTargets(source *transportir.ValueTarget, media []string) map[string]*transportir.ValueTarget {
	if source == nil {
		return nil
	}
	targets := make(map[string]*transportir.ValueTarget, len(media))
	for _, name := range media {
		target := *source
		target.Plan = expr.ValuePlan{}
		targets[name] = &target
	}
	return targets
}
