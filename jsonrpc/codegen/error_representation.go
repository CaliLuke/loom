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
func mappedErrorBody(response *httpcodegen.ResponseData) (initialization, value, source string) {
	source = "res"
	var code strings.Builder
	value = response.SourceVar
	if response.EncodePlan.NeedsProblemSource {
		source = "_" // The body reads the resolved problem, not the concrete owner.
		fmt.Fprintf(&code, "problem := loomhttp.NewProblemResponse(ctx, err, %d, %s, %s)\n", response.Code, response.ProblemTypeOverride, response.ProblemTitleOverride)
	}
	body := response.EncodePlan.FirstBody
	if body.Init != nil {
		args := make([]string, 0, len(body.Init.ServerArgs))
		for _, arg := range body.Init.ServerArgs {
			args = append(args, arg.Ref)
		}
		if body.Init.ServerErrorAware {
			fmt.Fprintf(&code, "data, conversionError := %s(%s)\nif conversionError != nil {\nreturn conversionError\n}\n", body.Init.Name, strings.Join(args, ", "))
		} else {
			fmt.Fprintf(&code, "data := %s(%s)\n", body.Init.Name, strings.Join(args, ", "))
		}
		value = "data"
	} else if response.ResultAttr != "" {
		value += "." + response.ResultAttr
	}
	return strings.TrimSuffix(code.String(), "\n"), value, source
}
