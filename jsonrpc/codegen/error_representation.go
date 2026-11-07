package codegen

import (
	"fmt"
	"strings"

	httpcodegen "github.com/CaliLuke/loom/http/codegen"
)

// mappedErrorBody emits the source preparation and projection selected by the
// shared response plan, and the identifier for its concrete-owner assertion.
// Callers admit that owner before running the initialization. Normalize err
// rather than res so outer public-message and remedy owners remain authoritative
// in both the envelope and its designed body.
func mappedErrorBody(response *httpcodegen.ResponseData, onError func(string) string) (initialization, value, source string) {
	source = "res"
	var code strings.Builder
	if response.ErrorPayload {
		source = "errorOwner"
		fmt.Fprintf(&code, "res, payloadError := errorOwner.Payload()\nif payloadError != nil {\n%s\n}\n", onError("payloadError"))
	}
	if preparation := response.ProblemSourcePreparation("err"); preparation != "" {
		code.WriteString(preparation)
		code.WriteByte('\n')
	}
	code.WriteString(response.EncodePlan.SourcePreparation(onError, ""))
	value = response.EncodePlan.BodyValues[response.EncodePlan.FirstBody]
	if response.EncodePlan.NeedsProblemSource && !response.ErrorPayload {
		source = "_" // The body reads the resolved problem, not the concrete owner.
	}
	body := response.EncodePlan.FirstBody
	if body.Init != nil {
		args := make([]string, 0, len(body.Init.ServerArgs))
		for _, arg := range body.Init.ServerArgs {
			args = append(args, arg.Ref)
		}
		if body.Init.ServerErrorAware {
			fmt.Fprintf(&code, "data, conversionError := %s(%s)\nif conversionError != nil {\n%s\n}\n", body.Init.Name, strings.Join(args, ", "), onError("conversionError"))
		} else {
			fmt.Fprintf(&code, "data := %s(%s)\n", body.Init.Name, strings.Join(args, ", "))
		}
		value = "data"
	}
	return strings.TrimSuffix(code.String(), "\n"), value, source
}
