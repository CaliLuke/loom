package jsonrpc

import (
	"errors"

	goa "goa.design/goa/v3/pkg"
)

type (
	// ErrorData is the default structured JSON-RPC error data emitted for Goa
	// errors. It is intended for machine consumers and carries transport-neutral
	// error characteristics plus optional remediation guidance.
	ErrorData struct {
		// Name is the Goa error name when available.
		Name string `json:"name,omitempty"`
		// ID is the unique Goa service error instance identifier.
		ID string `json:"id,omitempty"`
		// Temporary reports whether the error is temporary.
		Temporary bool `json:"temporary,omitempty"`
		// Timeout reports whether the error is a timeout.
		Timeout bool `json:"timeout,omitempty"`
		// Fault reports whether the error is a server-side fault.
		Fault bool `json:"fault,omitempty"`
		// Remedy contains optional remediation guidance.
		Remedy *goa.ErrorRemedy `json:"remedy,omitempty"`
	}
)

// NewErrorData returns structured JSON-RPC error data for err when Goa error
// metadata is available. It returns nil when err carries no machine-usable
// Goa error information.
func NewErrorData(err error) any {
	if err == nil {
		return nil
	}

	data := &ErrorData{
		Remedy: goa.ExtractErrorRemedy(err),
	}
	var (
		serviceError *goa.ServiceError
		namer        goa.GoaErrorNamer
	)
	if errors.As(err, &serviceError) {
		data.Name = serviceError.Name
		data.ID = serviceError.ID
		data.Temporary = serviceError.Temporary
		data.Timeout = serviceError.Timeout
		data.Fault = serviceError.Fault
	} else if errors.As(err, &namer) {
		data.Name = namer.GoaErrorName()
	}

	if data.Name == "" && data.ID == "" && !data.Temporary && !data.Timeout && !data.Fault && data.Remedy == nil {
		return nil
	}
	return data
}
