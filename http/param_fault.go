package http

import (
	"errors"

	loom "github.com/CaliLuke/loom/pkg"
)

// NormalizeParamDecodeError reports an error raised while assigning a decoded
// parameter value. A server fault, such as an authored default that cannot be
// materialized, keeps its classification; any other error is a decode_payload
// error carrying err's message.
func NormalizeParamDecodeError(err error) error {
	if fault := serverFault(err); fault != nil {
		return fault
	}
	return loom.DecodePayloadError(err.Error())
}

// serverFault returns the server fault carried by err, or nil.
func serverFault(err error) *loom.ServiceError {
	var serviceErr *loom.ServiceError
	if errors.As(err, &serviceErr) && serviceErr.Fault {
		return serviceErr
	}
	return nil
}
