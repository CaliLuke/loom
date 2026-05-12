package testdata

import (
	. "github.com/CaliLuke/loom/dsl"
)

// GRPCEndpointWithStreamingPayloadInitialRequest declares a gRPC method that
// defines both a one-shot Payload and a StreamingPayload. The Finalize pass
// must keep the ordinary payload fields on the request message instead of
// rewriting them into gRPC metadata.
var GRPCEndpointWithStreamingPayloadInitialRequest = func() {
	var VersionRef = Type("VersionRef", func() {
		OneOf("ref_type", func() {
			Field(1, "version_id", String)
			Field(2, "ref_name", String)
		})
		Required("ref_type")
	})
	var UploadChunk = Type("UploadChunk", func() {
		Field(1, "chunk", Bytes)
		Required("chunk")
	})
	Service("Service", func() {
		Method("Method", func() {
			Payload(func() {
				Field(1, "repository_id", String)
				Field(2, "version_ref", VersionRef)
				Required("repository_id", "version_ref")
			})
			StreamingPayload(UploadChunk)
			GRPC(func() {})
		})
	})
}
