package jsonrpc

import (
	"bytes"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"

	loomhttp "github.com/CaliLuke/loom/http"
	"github.com/CaliLuke/loom/internal/jsondecode"
)

type (
	// ErrorCandidate describes one generated designed-body alternative for a code.
	ErrorCandidate struct {
		// Name is the declared service error identity.
		Name string
		// Selector is the JSON member path of ErrorName. Nil means absent; a nonnil
		// empty path selects the data root. Selection precedes decoding and defaults.
		Selector []string
		// Decode consumes a private response and returns a private candidate.
		// Its second result uses the generated ErrDecodingError/ErrValidationError
		// wrappers, preserving the distinction from application decoder failures.
		Decode func(*http.Response) (error, error)
	}
)

// DecodeErrorResponse reconstructs exactly one mapped error from its designed
// data. Empty candidates retain the raw envelope. Unexpected eligible decoder
// errors take precedence over successes; structured conversion and validation
// failures reject an alternative. Callbacks must keep candidate state private;
// their external side effects cannot be rolled back by this function.
func DecodeErrorResponse(service, method string, raw *RawErrorResponse, response *http.Response, candidates []ErrorCandidate) error {
	if len(candidates) == 0 {
		return raw
	}
	if len(raw.Data) > 0 {
		var value jsontext.Value
		if err := json.Unmarshal(raw.Data, &value); err != nil {
			return loomhttp.ErrDecodingError(service, method, err)
		}
	}
	var selected error
	var failures, unexpected []error
	matches := 0
	eligible := 0
	for _, candidate := range candidates {
		if !matchesErrorSelector(raw.Data, candidate.Selector, candidate.Name) {
			continue
		}
		eligible++
		trial := errorCandidateResponse(response, raw.Data)
		value, err := candidate.Decode(trial)
		if closeErr := trial.Body.Close(); closeErr != nil {
			unexpected = append(unexpected, fmt.Errorf("%s: close candidate body: %w", candidate.Name, closeErr))
		}
		if err != nil {
			failure := fmt.Errorf("%s: %w", candidate.Name, err)
			var validation *loomhttp.ClientError
			generated := errors.As(err, &validation)
			if (generated && validation.Name == "validation_error") || jsondecode.IsMismatch(err) {
				failures = append(failures, failure)
			} else {
				unexpected = append(unexpected, failure)
			}
			continue
		}
		if value == nil {
			unexpected = append(unexpected, fmt.Errorf("%s: decoder returned no service error", candidate.Name))
			continue
		}
		matches++
		selected = value
	}
	if len(unexpected) > 0 {
		return loomhttp.ErrDecodingError(service, method, errors.Join(unexpected...))
	}
	if matches == 1 {
		return selected
	}
	message := "no matching JSON-RPC error body"
	if eligible == 0 {
		message = "unknown JSON-RPC error identity"
	} else if matches > 1 {
		message = "ambiguous JSON-RPC error identity"
	}
	return &loomhttp.ClientError{Name: "invalid_response", Message: message, Service: service, Method: method, Err: errors.Join(failures...)}
}

func matchesErrorSelector(data jsontext.Value, selector []string, name string) bool {
	if selector == nil {
		return true
	}
	for _, member := range selector {
		var object map[string]jsontext.Value
		if err := json.Unmarshal(data, &object); err != nil {
			return false
		}
		value, present := object[member]
		if !present {
			return false
		}
		data = value
	}
	if data.Kind() != '"' {
		return false
	}
	var value string
	return json.Unmarshal(data, &value) == nil && value == name
}

func errorCandidateResponse(source *http.Response, data jsontext.Value) *http.Response {
	response := &http.Response{StatusCode: http.StatusOK}
	if source != nil {
		*response = *source
		response.Header = source.Header.Clone()
		response.Trailer = source.Trailer.Clone()
		response.TransferEncoding = slices.Clone(source.TransferEncoding)
		if source.Request != nil {
			response.Request = source.Request.Clone(source.Request.Context())
		}
	}
	response.Body = io.NopCloser(bytes.NewReader(bytes.Clone(data)))
	return response
}
