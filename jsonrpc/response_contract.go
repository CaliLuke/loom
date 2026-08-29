package jsonrpc

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
)

type (
	// ResponseContractCaseKind identifies a JSON-RPC response branch.
	ResponseContractCaseKind string

	// ResponseContractCase describes one generated JSON-RPC wire-response branch.
	ResponseContractCase struct {
		// ID is stable while the service contract is unchanged.
		ID string
		// Kind identifies a success, service error, or notification.
		Kind ResponseContractCaseKind
		// ResultType is the designed success result type.
		ResultType string
		// HasResult reports whether the success envelope has a result member.
		HasResult bool
		// ErrorCode is the declared JSON-RPC error code.
		ErrorCode int
		// ErrorName is the declared service error name.
		ErrorName string
		// ErrorDataType is the designed JSON error-data type.
		ErrorDataType string
		// Stream describes a supported streaming terminal contract.
		Stream *StreamingResponseContract
	}

	// StreamingResponseContract describes a selected JSON-RPC stream terminal.
	StreamingResponseContract struct {
		// Transport identifies the streaming wire protocol.
		Transport string
		// Terminal identifies the expected terminal behavior.
		Terminal string
	}

	// ResponseContractEvent is one parsed streaming JSON-RPC event.
	ResponseContractEvent struct {
		// Type is the transport event type.
		Type string
		// Data is the JSON event payload.
		Data jsontext.Value
	}

	// ResponseContractObservation contains values observed from a generated handler.
	ResponseContractObservation struct {
		// Response is the HTTP response for the JSON-RPC request or stream handshake.
		Response *http.Response
		// Events contains parsed server-SSE events in wire order.
		Events []ResponseContractEvent
		// TerminalError is the final server-SSE read error.
		TerminalError error
	}
)

const (
	// ResponseContractSuccess identifies a successful response.
	ResponseContractSuccess ResponseContractCaseKind = "success"
	// ResponseContractError identifies a service error.
	ResponseContractError ResponseContractCaseKind = "error"
	// ResponseContractNotification identifies response suppression for an ID-less request.
	ResponseContractNotification ResponseContractCaseKind = "notification"
)

// ValidateResponseContract validates transport-owned JSON-RPC wire invariants.
func ValidateResponseContract(observation *ResponseContractObservation, contract ResponseContractCase) error {
	prefix := fmt.Sprintf("response contract %q", contract.ID)
	if observation == nil || observation.Response == nil {
		return fmt.Errorf("%s: response is nil", prefix)
	}
	if contract.Stream != nil {
		return validateStreamingResponseContract(prefix, observation, contract)
	}
	body, err := readResponseContractBody(observation.Response)
	if err != nil {
		return fmt.Errorf("%s: read response body: %w", prefix, err)
	}
	if contract.Kind == ResponseContractNotification {
		if len(bytes.TrimSpace(body)) != 0 {
			return fmt.Errorf("%s: notification produced a response body", prefix)
		}
		return nil
	}
	if observation.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: HTTP status is %d, want 200", prefix, observation.Response.StatusCode)
	}
	return validateResponseContractEnvelope(prefix, body, contract)
}

func validateStreamingResponseContract(prefix string, observation *ResponseContractObservation, contract ResponseContractCase) error {
	if observation.Response.StatusCode != http.StatusOK {
		return fmt.Errorf("%s: stream HTTP status is %d, want 200", prefix, observation.Response.StatusCode)
	}
	if observation.TerminalError != nil && !errors.Is(observation.TerminalError, io.EOF) {
		return fmt.Errorf("%s: terminal error, want clean EOF: %w", prefix, observation.TerminalError)
	}
	if contract.Stream.Terminal == "suppressed" {
		for _, event := range observation.Events {
			var envelope map[string]jsontext.Value
			if json.Unmarshal(event.Data, &envelope) == nil && (envelope["result"] != nil || envelope["error"] != nil) {
				return fmt.Errorf("%s: notification stream produced a terminal response", prefix)
			}
		}
		return nil
	}
	if len(observation.Events) == 0 {
		return fmt.Errorf("%s: no server-SSE events were observed", prefix)
	}
	finalEvent := observation.Events[len(observation.Events)-1]
	if finalEvent.Type != "message" {
		return fmt.Errorf("%s: final server-SSE event type is %q, want message", prefix, finalEvent.Type)
	}
	return validateResponseContractEnvelope(prefix, finalEvent.Data, contract)
}

func validateResponseContractEnvelope(prefix string, body []byte, contract ResponseContractCase) error {
	var envelope map[string]jsontext.Value
	if err := json.Unmarshal(body, &envelope); err != nil {
		return fmt.Errorf("%s: decode JSON-RPC response: %w", prefix, err)
	}
	var version string
	if err := json.Unmarshal(envelope["jsonrpc"], &version); err != nil || version != "2.0" {
		return fmt.Errorf("%s: jsonrpc version is %q, want 2.0", prefix, version)
	}
	if _, ok := envelope["id"]; !ok {
		return fmt.Errorf("%s: id member is missing", prefix)
	}
	if contract.Kind == ResponseContractSuccess {
		if envelope["error"] != nil {
			return fmt.Errorf("%s: success contains an error member", prefix)
		}
		if contract.HasResult && envelope["result"] == nil {
			return fmt.Errorf("%s: result member is missing", prefix)
		}
		return nil
	}
	var responseError RawErrorResponse
	if err := json.Unmarshal(envelope["error"], &responseError); err != nil {
		return fmt.Errorf("%s: decode error member: %w", prefix, err)
	}
	if responseError.Code != contract.ErrorCode {
		return fmt.Errorf("%s: error code is %d, want %d", prefix, responseError.Code, contract.ErrorCode)
	}
	if contract.ErrorDataType != "" && len(bytes.TrimSpace(responseError.Data)) == 0 {
		return fmt.Errorf("%s: typed error data %s is missing", prefix, contract.ErrorDataType)
	}
	if contract.ErrorName != "" {
		var data struct {
			Name string `json:"name"`
		}
		if json.Unmarshal(responseError.Data, &data) == nil && data.Name != "" && data.Name != contract.ErrorName {
			return fmt.Errorf("%s: error data name is %q, want %q", prefix, data.Name, contract.ErrorName)
		}
	}
	return nil
}

func readResponseContractBody(response *http.Response) ([]byte, error) {
	if response.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}
	response.Body = io.NopCloser(bytes.NewReader(body))
	return body, nil
}
