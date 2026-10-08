package grpc

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/runtime/protoiface"

	loom "github.com/CaliLuke/loom/pkg"
)

type (
	// errorContract is the single resolved owner of a failure's wire response.
	// A join without an explicit owner has only an aggregate code and message.
	errorContract struct {
		err         error
		code        codes.Code
		service     *loom.ServiceError
		status      *status.Status
		mapped      bool
		termination bool
		detail      protoiface.MessageV1
	}

	grpcStatuser interface {
		GRPCStatus() *status.Status
	}
)

// classifyError follows transparent wrappers until an explicit contract or an
// independent join. A direct assertion is essential: errors.As/FromError would
// cross the join and select an arbitrary descendant as the whole contract.
//
//nolint:errorlint // Contract ownership requires direct interfaces, not tree-wide matching.
func classifyError(err error, mapper ErrorMapper) errorContract {
	result := errorContract{err: err, code: codes.Unknown}
	direct := true
	for node := err; node != nil; {
		if named, ok := node.(loom.LoomErrorNamer); ok && mapper != nil {
			mapping, matched, mapErr := mapper(named.LoomErrorName(), err)
			if mapErr != nil {
				return classifyError(mapErr, nil)
			}
			if matched {
				result.code = failureCode(mapping.Code)
				result.mapped = true
				result.detail = mapping.Detail
				return result
			}
		}
		if service, ok := node.(*loom.ServiceError); ok {
			result.service = service
			result.code = serviceErrorCode(service)
			return result
		}
		if provider, ok := node.(grpcStatuser); ok {
			return statusErrorContract(result, provider.GRPCStatus(), direct)
		}
		switch node {
		case context.Canceled:
			result.code = codes.Canceled
			result.termination = true
			return result
		case context.DeadlineExceeded:
			result.code = codes.DeadlineExceeded
			result.termination = true
			return result
		}
		switch wrapped := node.(type) {
		case interface{ Unwrap() []error }:
			var children []error
			for _, child := range wrapped.Unwrap() {
				if child != nil {
					children = append(children, child)
				}
			}
			switch len(children) {
			case 0:
				return result
			case 1:
				node = children[0]
				direct = false
				continue
			}
			result.code = classifyError(children[0], mapper).code
			for _, child := range children[1:] {
				if classifyError(child, mapper).code != result.code {
					result.code = codes.Unknown
					return result
				}
			}
			return result
		case interface{ Unwrap() error }:
			node = wrapped.Unwrap()
			direct = false
		default:
			return result
		}
	}
	return result
}

// statusErrorContract preserves direct status messages; a transparent wrapper
// contributes its complete message without discarding the status details.
func statusErrorContract(result errorContract, st *status.Status, direct bool) errorContract {
	if st == nil {
		return result
	}
	result.code = failureCode(st.Code())
	wire := st.Proto()
	wire.Code = int32(result.code)
	if !direct {
		wire.Message = result.err.Error()
	}
	result.status = status.FromProto(wire)
	return result
}

func encodeError(err error, mapper ErrorMapper) error {
	if err == nil {
		return nil
	}
	contract := classifyError(err, mapper)
	if contract.termination {
		return NewStatusError(contract.code, contract.err)
	}
	if contract.mapped {
		if contract.detail == nil {
			return NewStatusError(contract.code, contract.err)
		}
		return NewStatusError(contract.code, contract.err, contract.detail)
	}
	detail := newErrorResponse(contract.err, contract.service)
	if contract.status != nil {
		if withDetails, err := contract.status.WithDetails(detail); err == nil {
			return withDetails.Err()
		}
		return contract.status.Err()
	}
	return NewStatusError(contract.code, contract.err, detail)
}

func failureCode(code codes.Code) codes.Code {
	if code == codes.OK {
		return codes.Unknown
	}
	return code
}

func serviceErrorCode(err *loom.ServiceError) codes.Code {
	switch err.Name {
	case loom.InvalidFieldType, loom.MissingField, loom.InvalidFormat,
		loom.InvalidLength, loom.InvalidRange, loom.InvalidEnumValue,
		loom.InvalidPattern, loom.DecodePayload, loom.MissingPayload:
		return codes.InvalidArgument
	}
	switch {
	case err.Timeout:
		return codes.DeadlineExceeded
	case err.Fault:
		return codes.Internal
	case err.Temporary:
		return codes.Unavailable
	default:
		return codes.Unknown
	}
}
