package grpc

import (
	"errors"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/emptypb"
)

func TestValidateResponseContract(t *testing.T) {
	contract := ResponseContractCase{
		ID:               "widgets.show.success.0",
		Kind:             ResponseContractSuccess,
		StatusCode:       codes.OK,
		MessageType:      "google.protobuf.Empty",
		RequiredHeaders:  []string{"x-version"},
		RequiredTrailers: []string{"x-checksum"},
	}
	valid := func() *ResponseContractObservation {
		return &ResponseContractObservation{
			Message:  &emptypb.Empty{},
			Headers:  metadata.Pairs("x-version", "1"),
			Trailers: metadata.Pairs("x-checksum", "abc"),
		}
	}

	tests := []struct {
		name    string
		mutate  func(*ResponseContractObservation)
		wantErr string
	}{
		{name: "success"},
		{name: "status", mutate: func(observation *ResponseContractObservation) {
			observation.Error = status.Error(codes.NotFound, "missing")
		}, wantErr: "status is NotFound, want OK"},
		{name: "message type", mutate: func(observation *ResponseContractObservation) {
			observation.Message = nil
		}, wantErr: "message is nil"},
		{name: "header", mutate: func(observation *ResponseContractObservation) {
			observation.Headers = nil
		}, wantErr: `required header "x-version" is missing`},
		{name: "trailer", mutate: func(observation *ResponseContractObservation) {
			observation.Trailers = nil
		}, wantErr: `required trailer "x-checksum" is missing`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation := valid()
			if test.mutate != nil {
				test.mutate(observation)
			}
			err := ValidateResponseContract(observation, contract)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			require.ErrorContains(t, err, test.wantErr)
		})
	}
}

func TestValidateResponseContractErrorDetail(t *testing.T) {
	contract := ResponseContractCase{
		ID:         "widgets.show.error.invalid.3",
		Kind:       ResponseContractError,
		StatusCode: codes.InvalidArgument,
		ErrorName:  "invalid",
		DetailType: "google.protobuf.Empty",
	}
	err := NewStatusError(codes.InvalidArgument, errors.New("invalid"), &emptypb.Empty{})

	require.NoError(t, ValidateResponseContract(&ResponseContractObservation{Error: err}, contract))
	require.ErrorContains(t, ValidateResponseContract(&ResponseContractObservation{
		Error: status.Error(codes.InvalidArgument, "invalid"),
	}, contract), "status detail google.protobuf.Empty is missing")
}

func TestValidateResponseContractServerStreamCompletion(t *testing.T) {
	contract := ResponseContractCase{
		ID:          "events.watch.success.0",
		Kind:        ResponseContractSuccess,
		StatusCode:  codes.OK,
		MessageType: "google.protobuf.Empty",
		Stream: &StreamingResponseContract{
			Direction: "server",
			Terminal:  "eof",
		},
	}

	require.NoError(t, ValidateResponseContract(&ResponseContractObservation{
		Messages:      []proto.Message{&emptypb.Empty{}},
		TerminalError: io.EOF,
	}, contract))
	require.ErrorContains(t, ValidateResponseContract(&ResponseContractObservation{
		Messages:      []proto.Message{&emptypb.Empty{}},
		TerminalError: errors.New("reset"),
	}, contract), "terminal error, want clean EOF")
}
