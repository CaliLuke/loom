package testdata

import . "github.com/CaliLuke/loom/dsl"

// StreamAdmissionDSL exercises metadata security before client-stream admission.
func StreamAdmissionDSL() {
	streamAdmissionDSL(false, false)
}

// BidirectionalAdmissionDSL exercises the same contract for bidirectional streams.
func BidirectionalAdmissionDSL() {
	streamAdmissionDSL(true, false)
}

// AuthorizedAdmissionDSL combines admission with complete-payload access checks.
func AuthorizedAdmissionDSL() {
	streamAdmissionDSL(false, true)
}

func streamAdmissionDSL(bidirectional, authorized bool) {
	key := APIKeySecurity("api_key", func() {
	})
	ref := Type("UploadRef", func() {
		Attribute("name", String)
		Required("name")
	})
	access := Authorization("upload.write", ref)
	Service("up", func() {
		Method("upload", func() {
			Security(key)
			Payload(func() {
				APIKey("api_key", "key", String)
				Field(1, "name", String, func() {
					MinLength(1)
				})
				Required("key", "name")
			})
			StreamingPayload(func() {
				Field(1, "chunk", String)
				Required("chunk")
			})
			if bidirectional {
				StreamingResult(String)
			} else {
				Result(String)
			}
			if authorized {
				Authorize(access, func() {
					Bind("name", "name")
				})
			}
			GRPC(func() {
			})
		})
	})
}

// StreamAdmissionControlsDSL exercises secure branches with no blocking initial receive.
func StreamAdmissionControlsDSL() {
	key := APIKeySecurity("api_key", func() {
	})
	Service("controls", func() {
		for _, name := range []string{"Unary", "ServerStreaming", "MetadataOnly"} {
			Method(name, func() {
				Security(key)
				Payload(func() {
					APIKey("api_key", "key", String)
					Required("key")
				})
				if name == "MetadataOnly" {
					StreamingPayload(String)
				}
				if name == "ServerStreaming" {
					StreamingResult(String)
				} else {
					Result(String)
				}
				GRPC(func() {
				})
			})
		}
	})
}
