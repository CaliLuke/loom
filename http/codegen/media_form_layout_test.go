package codegen

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestMediaFormFieldsUseBranchLayout(t *testing.T) {
	root := RunHTTPDSL(t, testdata.StyledMediaContractsDSL)
	services := CreateHTTPServices(root)
	checked := 0
	for _, service := range root.API.HTTP.Services {
		for _, endpoint := range services.Get(service.Name()).Endpoints {
			if endpoint.Payload == nil || endpoint.Payload.Request == nil {
				continue
			}
			body := endpoint.Payload.Request.ServerBody
			if body == nil || body.Media == nil {
				continue
			}
			for _, variant := range body.Media.Variants {
				if len(variant.StyledFormFields) == 0 {
					continue
				}
				checked++
				require.False(t, variant.formBody.context.JSONPresence, "form branch must not inherit JSON union presence")
				require.False(t, variant.formBody.context.CollectionElementPresence)
			}
		}
	}
	require.Positive(t, checked)
}
