package jsonrpc

import loom "github.com/CaliLuke/loom/pkg"

type (
	// ErrorData is the default structured JSON-RPC error data emitted for Loom
	// errors. It is intended for machine consumers and carries transport-neutral
	// error characteristics plus optional remediation guidance.
	ErrorData struct {
		// Name is the Loom error name when available.
		Name string `json:"name,omitempty"`
		// ID is the unique Loom service error instance identifier.
		ID string `json:"id,omitempty"`
		// Temporary reports whether the error is temporary.
		Temporary bool `json:"temporary,omitempty"`
		// Timeout reports whether the error is a timeout.
		Timeout bool `json:"timeout,omitempty"`
		// Fault reports whether the error is a server-side fault.
		Fault bool `json:"fault,omitempty"`
		// Remedy contains optional remediation guidance.
		Remedy *loom.ErrorRemedy `json:"remedy,omitempty"`
	}
)

// NewErrorData returns structured JSON-RPC error data for err when Loom error
// metadata is available. It returns nil when err carries no machine-usable
// Loom error information.
//
//nolint:errorlint // The response owner is resolved; do not select an inner contract.
func NewErrorData(err error) any {
	if err == nil {
		return nil
	}

	response := loom.ResolveErrorResponse(err)
	data := &ErrorData{Name: response.Name, Remedy: response.Remedy}
	if serviceError, ok := response.Owner.(*loom.ServiceError); ok {
		data.ID = serviceError.ID
		data.Temporary = serviceError.Temporary
		data.Timeout = serviceError.Timeout
		data.Fault = serviceError.Fault
	}

	if data.Name == "" && data.ID == "" && !data.Temporary && !data.Timeout && !data.Fault && data.Remedy == nil {
		return nil
	}
	return data
}
