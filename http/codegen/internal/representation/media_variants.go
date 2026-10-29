package representation

import (
	"mime"

	"github.com/CaliLuke/loom/codegen"
	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
	"github.com/CaliLuke/loom/internal/httpcodec"
)

func attachMediaTargets(variants []*transportir.MediaVariant, body *transportir.ValueTarget) {
	if body == nil || body.Source == nil {
		return
	}
	for _, variant := range variants {
		occurrence, err := body.Source.Context.NewOccurrence(variant.Body)
		variant.Value = &transportir.ValueTarget{
			Source: body.Source, Anchor: body.Source,
			Selection:         append(append([]string(nil), body.Selection...), variant.Branch),
			ExampleOccurrence: occurrence, Codec: mediaVariantCodec(variant.ContentType), Error: err,
		}
		if variant.Value.Codec == expr.ValueCodecCustom {
			variant.Value.Boundary = "media variant codec is selected from the concrete Content-Type or supplied factory"
		}
	}
}

func mediaVariantCodec(media string) expr.ValueCodec {
	base, _, err := mime.ParseMediaType(media)
	if err != nil {
		return expr.ValueCodecCustom
	}
	switch base {
	case "application/x-www-form-urlencoded":
		return expr.ValueCodecForm
	case "multipart/form-data":
		return expr.ValueCodecMultipart
	case "application/octet-stream":
		return expr.ValueCodecRaw
	}
	switch httpcodec.RequestContentType(media).Kind {
	case httpcodec.JSON:
		return expr.ValueCodecJSON
	case httpcodec.Text:
		return expr.ValueCodecText
	default:
		return expr.ValueCodecCustom
	}
}

func prepareMediaTargets(variants []*transportir.MediaVariant, context *codegen.AttributeContext) {
	for _, variant := range variants {
		variant.Value = BuildValuePlan(variant.Value, variant.Body, context, expr.ValuePlanSchema)
	}
}
