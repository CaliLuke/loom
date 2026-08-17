package grpc

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

type (
	// ResponseContractCaseKind identifies a successful result or service error.
	ResponseContractCaseKind string

	// ResponseContractCase describes one generated gRPC wire-response branch.
	ResponseContractCase struct {
		// ID is stable while the service contract is unchanged.
		ID string
		// Kind identifies a successful result or service error.
		Kind ResponseContractCaseKind
		// StatusCode is the declared gRPC status code.
		StatusCode codes.Code
		// MessageType is the protobuf full name of a successful response message.
		MessageType string
		// ErrorName is the declared service error name.
		ErrorName string
		// DetailType is the protobuf full name of the expected status detail.
		DetailType string
		// RequiredHeaders lists required response metadata keys.
		RequiredHeaders []string
		// RequiredTrailers lists required trailer metadata keys.
		RequiredTrailers []string
		// Stream describes a supported streaming completion contract.
		Stream *StreamingResponseContract
	}

	// StreamingResponseContract describes a selected gRPC stream completion.
	StreamingResponseContract struct {
		// Direction is the designed stream direction.
		Direction string
		// Terminal identifies the expected completion behavior.
		Terminal string
	}

	// ResponseContractObservation contains values observed through a generated
	// gRPC client. Unary scenarios set Message and Error. Streaming scenarios set
	// Messages and TerminalError.
	ResponseContractObservation struct {
		// Message is the unary response message.
		Message proto.Message
		// Error is the unary call error.
		Error error
		// Headers contains response header metadata.
		Headers metadata.MD
		// Trailers contains response trailer metadata.
		Trailers metadata.MD
		// Messages contains server-stream messages in receive order.
		Messages []proto.Message
		// TerminalError is the final server-stream receive error.
		TerminalError error
	}
)

const (
	// ResponseContractSuccess identifies a successful response contract case.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies an error response contract case.
	ResponseContractError ResponseContractCaseKind = "error"
)

// ValidateResponseContract validates transport-owned gRPC wire invariants.
func ValidateResponseContract(observation *ResponseContractObservation, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if observation == nil {
		return fmt.Errorf("%s: observation is nil", prefix)
	}
	callErr := observation.Error
	if contract.Stream != nil {
		callErr = observation.TerminalError
		if contract.Kind == ResponseContractSuccess && contract.Stream.Terminal == "eof" && callErr != nil && !errors.Is(callErr, io.EOF) {
			return fmt.Errorf("%s: terminal error, want clean EOF: %w", prefix, callErr)
		}
	}
	actualCode := status.Code(callErr)
	if contract.Stream != nil && errors.Is(callErr, io.EOF) {
		actualCode = codes.OK
	}
	if actualCode != contract.StatusCode {
		return fmt.Errorf("%s: status is %s, want %s", prefix, actualCode, contract.StatusCode)
	}
	if err := validateResponseMetadata(observation.Headers, contract.RequiredHeaders, "header"); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	if err := validateResponseMetadata(observation.Trailers, contract.RequiredTrailers, "trailer"); err != nil {
		return fmt.Errorf("%s: %w", prefix, err)
	}
	if contract.Kind == ResponseContractError {
		if contract.DetailType != "" && !responseContractHasDetail(callErr, contract.DetailType) {
			return fmt.Errorf("%s: status detail %s is missing", prefix, contract.DetailType)
		}
		return nil
	}
	if contract.Stream != nil {
		return validateStreamingResponseContract(prefix, observation, contract)
	}
	return validateResponseContractMessage(prefix, observation.Message, contract.MessageType)
}

func validateStreamingResponseContract(prefix string, observation *ResponseContractObservation, contract ResponseContractCase) error {
	if contract.MessageType != "" && len(observation.Messages) == 0 {
		return fmt.Errorf("%s: no stream messages were observed", prefix)
	}
	for index, message := range observation.Messages {
		if err := validateResponseContractMessage(prefix, message, contract.MessageType); err != nil {
			return fmt.Errorf("%s: stream message %d: %w", prefix, index, err)
		}
	}
	return nil
}

func validateResponseContractMessage(prefix string, message proto.Message, expected string) error {
	if expected == "" {
		return nil
	}
	if message == nil {
		return fmt.Errorf("%s: message is nil", prefix)
	}
	actual := string(message.ProtoReflect().Descriptor().FullName())
	if actual != expected {
		return fmt.Errorf("%s: message type is %s, want %s", prefix, actual, expected)
	}
	return nil
}

func validateResponseMetadata(values metadata.MD, required []string, kind string) error {
	for _, name := range required {
		if len(values.Get(name)) == 0 {
			return fmt.Errorf("required %s %q is missing", kind, name)
		}
	}
	return nil
}

func responseContractHasDetail(err error, expected string) bool {
	if err == nil {
		return false
	}
	for _, detail := range status.Convert(err).Proto().Details {
		if strings.TrimPrefix(detail.TypeUrl, "type.googleapis.com/") == expected {
			return true
		}
	}
	return false
}
