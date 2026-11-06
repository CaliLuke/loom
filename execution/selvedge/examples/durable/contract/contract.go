// Package contract contains the application-owned metadata job shared by binaries.
package contract

import "github.com/CaliLuke/loom/execution"

type (
	// Input identifies the object whose HTTP metadata should be read.
	Input struct {
		// URL is the application's object URL.
		URL string `json:"url"`
	}
	// Metadata is the result of inspecting an object's response headers.
	Metadata struct {
		// Length is the advertised object size, or -1 when unknown.
		Length int64 `json:"length"`
		// ContentType is the advertised media type.
		ContentType string `json:"content_type"`
	}
)

var (
	// Inspect is the durable job registered by the worker and declared by the producer.
	Inspect = execution.Workflow[Input, Metadata]{Name: "example.metadata.inspect"}
	// Read is the activity that performs the application's HTTP request.
	Read = execution.Activity[Input, Metadata]{Name: "example.metadata.read"}
)
