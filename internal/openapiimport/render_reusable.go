package openapiimport

import (
	"encoding/json/v2"

	"github.com/CaliLuke/loom/internal/openapiref"
)

func reusableMetadata(kind string, chain []openapiref.Object) ([]renderedMetadata, error) {
	if len(chain) == 0 {
		return nil, nil
	}
	value, err := json.Marshal(chain, json.Deterministic(true))
	if err != nil {
		return nil, err
	}
	return []renderedMetadata{{name: "openapi:reference:" + kind, value: string(value)}}, nil
}

func requestBodyMetadata(body RequestBody) ([]renderedMetadata, error) {
	metadata, err := renderedExtensions("requestBody", body.Extensions)
	if err != nil {
		return nil, err
	}
	if body.Description != "" || len(body.referenceChain) > 0 {
		metadata = append(metadata, renderedMetadata{name: "openapi:description:requestBody", value: body.Description})
	}
	references, err := reusableMetadata("requestBody", body.referenceChain)
	if err != nil {
		return nil, err
	}
	return append(metadata, references...), nil
}
