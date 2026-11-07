package codegen

import (
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// requestPayloadPresence records absence from the resolved service carrier,
// before a transport conversion can dereference it. Nil outer envelopes cannot
// supply selected member bodies or separately mapped bindings.
func requestPayloadPresence(request *transportir.Request, payload *expr.AttributeExpr, ref string, credentials bool) (string, bool) {
	if request.Body == nil || request.Body.Type == expr.Empty || request.SkipBodyEncode {
		return "", false
	}
	absent := sourceValueAbsence(payload, "p", strings.HasPrefix(ref, "*"))
	optional := request.BodyOrigin == "" && !request.MustHaveBody && !credentials &&
		len(request.PathParams)+len(request.QueryParams)+len(request.Headers)+len(request.Cookies) == 0
	return absent, optional
}
