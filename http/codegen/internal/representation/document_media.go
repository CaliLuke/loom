package representation

import (
	"mime"
	"strings"

	"github.com/CaliLuke/loom/expr"
	"github.com/CaliLuke/loom/http/codegen/internal/transportir"
)

// documentationTargets applies declared media semantics only to independent
// documentation contracts. Runtime targets always retain their emitter-owned
// codec. Only the runtime selector input chooses that codec; a mapped output
// header may name a different media type without selecting another encoder.
func documentationTargets(source *transportir.ValueTarget, mediaTypes []string) map[string]*transportir.ValueTarget {
	if source == nil {
		return nil
	}
	targets := make(map[string]*transportir.ValueTarget, len(mediaTypes))
	for _, media := range mediaTypes {
		target := *source
		target.Codec = expr.ValueCodecRaw
		typ, _, err := mime.ParseMediaType(media)
		if err == nil && (typ == "application/json" || strings.HasSuffix(typ, "+json")) {
			target.Codec = expr.ValueCodecJSON
		}
		target.Plan = expr.ValuePlan{}
		if target.Codec != expr.ValueCodecJSON {
			target.Boundary = "documentation media has no builtin JSON contract"
		}
		targets[media] = &target
	}
	return targets
}
