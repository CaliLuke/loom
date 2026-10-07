package grpc

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	loompb "github.com/CaliLuke/loom/grpc/pb"
)

func TestNewServiceErrorWithCause(t *testing.T) {
	original := status.Error(codes.Canceled, "remote canceled")
	causes := []struct {
		name string
		err  error
	}{
		{"nil", nil},
		{"plain", errors.New("plain failure")},
		{"status", original},
		{"wrapped", fmt.Errorf("wrapped: %w", original)},
	}
	responses := []struct {
		name     string
		response *loompb.ErrorResponse
	}{
		{"empty fields", &loompb.ErrorResponse{}},
		{"received fields", &loompb.ErrorResponse{
			Name: "remote", Id: "wire-id", Msg: "wire message", Timeout: true, Temporary: true, Fault: true,
			History: []*loompb.ErrorField{nil, {Name: "missing", Field: "name", Msg: "missing name"}},
		}},
	}
	for _, c := range causes {
		for _, r := range responses {
			t.Run(c.name+"/"+r.name, func(t *testing.T) {
				got := NewServiceErrorWithCause(r.response, c.err)
				require.Equal(t, r.response.Name, got.Name)
				require.Equal(t, r.response.Id, got.ID)
				require.Equal(t, r.response.Msg, got.Message)
				require.Equal(t, r.response.Timeout, got.Timeout)
				require.Equal(t, r.response.Temporary, got.Temporary)
				require.Equal(t, r.response.Fault, got.Fault)
				require.Equal(t, c.err, errors.Unwrap(got))
				wantCode := codes.Unknown
				if c.err != nil {
					wantCode = status.Code(c.err)
				}
				require.Equal(t, wantCode, status.Code(got))
				history := got.History()
				require.Len(t, history, 1)
				if len(r.response.History) > 0 {
					require.Equal(t, "missing", history[0].Name)
					require.NotNil(t, history[0].Field)
					require.Equal(t, "name", *history[0].Field)
				} else {
					require.Same(t, got, history[0])
				}
			})
		}
	}
}
