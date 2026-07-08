package grpc

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	statuspb "google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/anypb"

	loompb "github.com/CaliLuke/loom/grpc/pb"
	loom "github.com/CaliLuke/loom/pkg"
)

// TestNewErrorResponseHistory tests that history is correctly included for merged errors
func TestNewErrorResponseHistory(t *testing.T) {
	// Test simple error - should not have history
	simpleErr := loom.MissingFieldError("username", "body")
	resp := NewErrorResponse(simpleErr)
	assert.Nil(t, resp.History)

	// Test merged error - should have history
	mergedErr := loom.MergeErrors(
		loom.MissingFieldError("username", "body"),
		loom.InvalidFormatError("data", "{invalid}", loom.FormatJSON, fmt.Errorf("invalid JSON")),
	)
	mergedResp := NewErrorResponse(mergedErr)
	assert.NotNil(t, mergedResp.History)
	assert.Len(t, mergedResp.History, 2)
	assert.Equal(t, loom.MissingField, mergedResp.History[0].Name)
	assert.Equal(t, "username", mergedResp.History[0].Field)
	assert.Equal(t, loom.InvalidFormat, mergedResp.History[1].Name)
	assert.Equal(t, "data", mergedResp.History[1].Field)
}

// TestEncodeErrorStatusCodes tests that validation errors get mapped to InvalidArgument
func TestEncodeErrorStatusCodes(t *testing.T) {
	cases := []struct {
		name         string
		err          error
		expectedCode codes.Code
	}{
		{
			name:         "missing_field",
			err:          loom.MissingFieldError("username", "body"),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "invalid_format",
			err:          loom.InvalidFormatError("data", "{invalid}", loom.FormatJSON, fmt.Errorf("invalid JSON")),
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "decode_payload",
			err:          &loom.ServiceError{Name: "decode_payload", Message: "failed to decode"},
			expectedCode: codes.InvalidArgument,
		},
		{
			name:         "timeout",
			err:          &loom.ServiceError{Name: "timeout", Message: "timed out", Timeout: true},
			expectedCode: codes.DeadlineExceeded,
		},
		{
			name:         "fault",
			err:          loom.Fault("internal error"),
			expectedCode: codes.Internal,
		},
		{
			name:         "temporary",
			err:          &loom.ServiceError{Name: "unavailable", Message: "service unavailable", Temporary: true},
			expectedCode: codes.Unavailable,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			encoded := EncodeError(c.err)
			st, ok := status.FromError(encoded)
			assert.True(t, ok)
			assert.Equal(t, c.expectedCode, st.Code())

			// Check that details are included
			details := st.Details()
			assert.Len(t, details, 1)
			_, ok = details[0].(*loompb.ErrorResponse)
			assert.True(t, ok)
		})
	}
}

func TestDecodeErrorSkipsUnrecognizedDetails(t *testing.T) {
	validDetail := &loompb.ErrorResponse{Name: "boom", Msg: "handled"}
	validAny, err := anypb.New(validDetail)
	require.NoError(t, err)

	cases := []struct {
		name    string
		details []*anypb.Any
		want    *loompb.ErrorResponse
	}{
		{
			name: "returns later proto detail",
			details: []*anypb.Any{
				{
					TypeUrl: "type.googleapis.com/loom.test.UnlinkedDetail",
					Value:   []byte{0x08, 0x01},
				},
				validAny,
			},
			want: validDetail,
		},
		{
			name: "returns nil when no proto detail exists",
			details: []*anypb.Any{
				{
					TypeUrl: "type.googleapis.com/loom.test.UnlinkedDetail",
					Value:   []byte{0x08, 0x01},
				},
			},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			st := status.FromProto(&statuspb.Status{
				Code:    int32(codes.Internal),
				Message: "remote failure",
				Details: c.details,
			})

			require.NotPanics(t, func() {
				got := DecodeError(st.Err())
				if c.want == nil {
					assert.Nil(t, got)
					return
				}
				resp, ok := got.(*loompb.ErrorResponse)
				require.True(t, ok)
				assert.Equal(t, c.want.Name, resp.Name)
				assert.Equal(t, c.want.Msg, resp.Msg)
			})
		})
	}
}
