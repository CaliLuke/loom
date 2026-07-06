package testdata

const EmptyResultResponseEncoderCode = `// EncodeMethodUnaryRPCNoResultResponse encodes responses from the
// "ServiceUnaryRPCNoResult" service "MethodUnaryRPCNoResult" endpoint.
func EncodeMethodUnaryRPCNoResultResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	resp := NewProtoMethodUnaryRPCNoResultResponse()
	return resp, nil
}
`

const ResultWithViewsResponseEncoderCode = `// EncodeMethodMessageResultTypeWithViewsResponse encodes responses from the
// "ServiceMessageResultTypeWithViews" service
// "MethodMessageResultTypeWithViews" endpoint.
func EncodeMethodMessageResultTypeWithViewsResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	vres, ok := v.(*servicemessageresulttypewithviewsviews.RT)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageResultTypeWithViews", "MethodMessageResultTypeWithViews", "*servicemessageresulttypewithviewsviews.RT", v)
	}
	result := vres.Projected
	(*hdr).Append("loom-view", vres.View)
	resp := NewProtoMethodMessageResultTypeWithViewsResponse(result)
	return resp, nil
}
`

const ResultWithExplicitViewResponseEncoderCode = `// EncodeMethodMessageResultTypeWithExplicitViewResponse encodes responses from
// the "ServiceMessageResultTypeWithExplicitView" service
// "MethodMessageResultTypeWithExplicitView" endpoint.
func EncodeMethodMessageResultTypeWithExplicitViewResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	vres, ok := v.(*servicemessageresulttypewithexplicitviewviews.RT)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageResultTypeWithExplicitView", "MethodMessageResultTypeWithExplicitView", "*servicemessageresulttypewithexplicitviewviews.RT", v)
	}
	result := vres.Projected
	(*hdr).Append("loom-view", vres.View)
	resp := NewProtoMethodMessageResultTypeWithExplicitViewResponse(result)
	return resp, nil
}
`

const ResultArrayResponseEncoderCode = `// EncodeMethodMessageArrayResponse encodes responses from the
// "ServiceMessageArray" service "MethodMessageArray" endpoint.
func EncodeMethodMessageArrayResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	result, ok := v.([]*servicemessagearray.UT)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageArray", "MethodMessageArray", "[]*servicemessagearray.UT", v)
	}
	resp := NewProtoMethodMessageArrayResponse(result)
	return resp, nil
}
`

const ResultPrimitiveResponseEncoderCode = `// EncodeMethodUnaryRPCNoPayloadResponse encodes responses from the
// "ServiceUnaryRPCNoPayload" service "MethodUnaryRPCNoPayload" endpoint.
func EncodeMethodUnaryRPCNoPayloadResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	result, ok := v.(string)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceUnaryRPCNoPayload", "MethodUnaryRPCNoPayload", "string", v)
	}
	resp := NewProtoMethodUnaryRPCNoPayloadResponse(result)
	return resp, nil
}
`

const ResultWithMetadataResponseEncoderCode = `// EncodeMethodMessageWithMetadataResponse encodes responses from the
// "ServiceMessageWithMetadata" service "MethodMessageWithMetadata" endpoint.
func EncodeMethodMessageWithMetadataResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	result, ok := v.(*servicemessagewithmetadata.ResponseUT)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageWithMetadata", "MethodMessageWithMetadata", "*servicemessagewithmetadata.ResponseUT", v)
	}
	resp := NewProtoMethodMessageWithMetadataResponse(result)

	if result.InHeader != nil {
		(*hdr).Append("Location", fmt.Sprintf("%v", *result.InHeader))
	}

	if result.InTrailer != nil {
		(*trlr).Append("InTrailer", fmt.Sprintf("%v", *result.InTrailer))
	}
	return resp, nil
}
`

const ResultWithValidateResponseEncoderCode = `// EncodeMethodMessageWithValidateResponse encodes responses from the
// "ServiceMessageWithValidate" service "MethodMessageWithValidate" endpoint.
func EncodeMethodMessageWithValidateResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	result, ok := v.(*servicemessagewithvalidate.ResponseUT)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageWithValidate", "MethodMessageWithValidate", "*servicemessagewithvalidate.ResponseUT", v)
	}
	resp := NewProtoMethodMessageWithValidateResponse(result)

	if result.InHeader != nil {
		(*hdr).Append("Location", fmt.Sprintf("%v", *result.InHeader))
	}

	if result.InTrailer != nil {
		(*trlr).Append("InTrailer", fmt.Sprintf("%v", *result.InTrailer))
	}
	return resp, nil
}
`

const ResultCollectionResponseEncoderCode = `// EncodeMethodMessageUserTypeWithNestedUserTypesResponse encodes responses
// from the "ServiceMessageUserTypeWithNestedUserTypes" service
// "MethodMessageUserTypeWithNestedUserTypes" endpoint.
func EncodeMethodMessageUserTypeWithNestedUserTypesResponse(ctx context.Context, v any, hdr, trlr *metadata.MD) (any, error) {
	vres, ok := v.(servicemessageusertypewithnestedusertypesviews.RTCollection)
	if !ok {
		return nil, loomgrpc.ErrInvalidType("ServiceMessageUserTypeWithNestedUserTypes", "MethodMessageUserTypeWithNestedUserTypes", "servicemessageusertypewithnestedusertypesviews.RTCollection", v)
	}
	result := vres.Projected
	(*hdr).Append("loom-view", vres.View)
	resp := NewProtoRTCollection(result)
	return resp, nil
}
`
