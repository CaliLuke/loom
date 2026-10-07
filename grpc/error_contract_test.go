package grpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/wrapperspb"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
)

type contractStatusError struct {
	err          error
	st           *status.Status
	uncomparable []int
}

func (e contractStatusError) Error() string {
	return fmt.Sprintf("outer contract %v", e.uncomparable)
}

func (e contractStatusError) Unwrap() error {
	return e.err
}

func (e contractStatusError) GRPCStatus() *status.Status {
	return e.st
}

func TestJoinedErrorContract(t *testing.T) {
	for _, tc := range []struct {
		name string
		a, b error
		code codes.Code
	}{
		{"conflicting statuses", status.Error(codes.Canceled, "canceled"), status.Error(codes.Unavailable, "unavailable"), codes.Unknown},
		{"same statuses", status.Error(codes.Unavailable, "one"), status.Error(codes.Unavailable, "two"), codes.Unavailable},
		{"service and plain", loom.TemporaryError("busy", "busy"), errors.New("cleanup failed"), codes.Unknown},
		{"same service traits", loom.TemporaryError("busy", "one"), loom.TemporaryError("overloaded", "two"), codes.Unavailable},
		{"validation and status", loom.MissingFieldError("name", "body"), status.Error(codes.InvalidArgument, "invalid"), codes.InvalidArgument},
		{"nested disagreement", errors.Join(status.Error(codes.Canceled, "one"), status.Error(codes.Unavailable, "two")), status.Error(codes.Canceled, "three"), codes.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, pair := range [][2]error{{tc.a, tc.b}, {tc.b, tc.a}} {
				joined := errors.Join(pair[0], pair[1])
				for _, err := range []error{joined, fmt.Errorf("context: %w", joined)} {
					for _, encoded := range []error{EncodeError(err), EncodeServerError(err, nil)} {
						require.Equal(t, tc.code, status.Code(encoded))
						require.Equal(t, err.Error(), status.Convert(encoded).Message())
						detail := DecodeError(encoded).(*loompb.ErrorResponse)
						assertAggregateResponse(t, err, detail)
					}
					assertAggregateResponse(t, err, NewErrorResponse(err))
				}
			}
		})
	}
}

func TestExplicitOuterErrorContract(t *testing.T) {
	joined := errors.Join(status.Error(codes.Canceled, "one"), status.Error(codes.Unavailable, "two"))
	outer := loom.NewServiceError(joined, "aggregate", false, false, true)
	outer.Message = "complete failure"
	for _, err := range []error{outer, fmt.Errorf("context: %w", outer)} {
		encoded := EncodeServerError(err, nil)
		require.Equal(t, codes.Internal, status.Code(encoded))
		detail := DecodeError(encoded).(*loompb.ErrorResponse)
		require.Equal(t, "aggregate", detail.Name)
		require.Equal(t, "complete failure", detail.Msg)
	}
	for _, tc := range []struct {
		name string
		st   *status.Status
		code codes.Code
	}{
		{"status", status.New(codes.PermissionDenied, "complete denial"), codes.PermissionDenied},
		{"nil status", nil, codes.Unknown},
		{"ok status on error", status.New(codes.OK, "not success"), codes.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeError(contractStatusError{err: joined, st: tc.st})
			require.Error(t, encoded)
			require.Equal(t, tc.code, status.Code(encoded))
		})
	}
}

func TestJoinedDesignedErrorMappings(t *testing.T) {
	mapper := func(name string, err error) (ErrorMapping, bool, error) {
		switch name {
		case "denied":
			return ErrorMapping{Code: codes.PermissionDenied, Detail: wrapperspb.String(err.Error())}, true, nil
		case "broken":
			return ErrorMapping{}, false, errors.New("detail conversion failed")
		}
		return ErrorMapping{}, false, nil
	}
	denied := loom.PermanentError("denied", "denied")
	for _, tc := range []struct {
		name string
		err  error
		code codes.Code
	}{
		{"same mapping", errors.Join(denied, denied), codes.PermissionDenied},
		{"mixed mapping", errors.Join(denied, errors.New("cleanup failed")), codes.Unknown},
		{"conversion failure", errors.Join(denied, loom.PermanentError("broken", "broken")), codes.Unknown},
	} {
		t.Run(tc.name, func(t *testing.T) {
			encoded := EncodeServerError(tc.err, mapper)
			require.Equal(t, tc.code, status.Code(encoded))
			assertAggregateResponse(t, tc.err, DecodeError(encoded).(*loompb.ErrorResponse))
		})
	}
	for _, err := range []error{denied, fmt.Errorf("context: %w", denied), errors.Join(nil, denied)} {
		encoded := EncodeServerError(err, mapper)
		require.Equal(t, codes.PermissionDenied, status.Code(encoded))
		require.IsType(t, &wrapperspb.StringValue{}, DecodeError(encoded))
	}
	outer := loom.NewServiceError(errors.Join(denied, errors.New("cleanup")), "denied", false, false, false)
	require.IsType(t, &wrapperspb.StringValue{}, DecodeError(EncodeServerError(outer, mapper)))
}

func TestSingleErrorContracts(t *testing.T) {
	require.Nil(t, EncodeError(nil))
	require.Nil(t, EncodeServerError(nil, nil))
	require.Nil(t, NewErrorResponse(nil))
	service := loom.TemporaryError("busy", "busy")
	for _, err := range []error{service, fmt.Errorf("wrapped: %w", service), errors.Join(service)} {
		encoded := EncodeError(err)
		require.Equal(t, codes.Unavailable, status.Code(encoded))
		detail := DecodeError(encoded).(*loompb.ErrorResponse)
		require.Equal(t, "busy", detail.Name)
		require.True(t, detail.Temporary)
	}
	st, err := status.New(codes.NotFound, "missing").WithDetails(wrapperspb.String("existing detail"))
	require.NoError(t, err)
	wrapped := fmt.Errorf("wrapped: %w", st.Err())
	encoded := EncodeError(wrapped)
	require.Equal(t, codes.NotFound, status.Code(encoded))
	require.Equal(t, wrapped.Error(), status.Convert(encoded).Message())
	require.IsType(t, &wrapperspb.StringValue{}, DecodeError(encoded))
	validation := loom.MergeErrors(loom.MissingFieldError("one", "body"), loom.MissingFieldError("two", "body"))
	encoded = EncodeError(validation)
	require.Equal(t, codes.InvalidArgument, status.Code(encoded))
	require.Len(t, DecodeError(encoded).(*loompb.ErrorResponse).History, 2)
}

func assertAggregateResponse(t *testing.T, err error, detail *loompb.ErrorResponse) {
	t.Helper()
	require.Equal(t, "fault", detail.Name)
	require.Equal(t, err.Error(), detail.Msg)
	require.True(t, detail.Fault)
	require.False(t, detail.Temporary)
	require.False(t, detail.Timeout)
	require.Empty(t, detail.History)
}
