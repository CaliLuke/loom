package framework

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/CaliLuke/loom/jsonrpc/integration_tests/harness"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// executeBatch handles batch request scenarios
func (e *executor) executeBatch(t *testing.T, scenario Scenario) {
	t.Helper()
	e.debugf(t, "batch request size=%d", len(scenario.Batch))

	// Batch requests only work with HTTP
	require.Equal(t, TransportHTTP, scenario.Transport, "Batch requests only supported on HTTP transport")

	ctx := context.Background()
	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	// Build batch request
	batch := make([]any, 0, len(scenario.Batch))
	for _, req := range scenario.Batch {
		method := req.GetMethod(scenario.Method)
		jsonReq := map[string]any{
			"jsonrpc": "2.0",
			"method":  method,
			"params":  req.Params,
		}
		if req.ID != nil {
			jsonReq["id"] = req.ID
		}
		batch = append(batch, jsonReq)
	}

	// Send batch
	batchJSON, err := json.Marshal(batch)
	require.NoError(t, err, "Failed to marshal batch")

	responseJSON, err := client.CallHTTPRaw(ctx, batchJSON)
	require.NoError(t, err, "Batch call failed")

	// Parse batch response
	var responses []json.RawMessage
	err = json.Unmarshal(responseJSON, &responses)
	require.NoError(t, err, "Failed to parse batch response")

	// Validate responses
	require.Len(t, responses, len(scenario.ExpectBatch), "Response count mismatch")

	for i, respJSON := range responses {
		var resp map[string]any
		err := json.Unmarshal(respJSON, &resp)
		require.NoErrorf(t, err, "Failed to parse response %d", i)

		e.validateBatchResponse(t, i, resp, scenario.ExpectBatch[i])
	}
}

// executeRaw handles raw request scenarios
func (e *executor) executeRaw(t *testing.T, scenario Scenario) {
	t.Helper()
	e.debugf(t, "raw request bytes=%d", len(scenario.RawRequest))

	// Raw requests only work with HTTP
	require.Equal(t, TransportHTTP, scenario.Transport, "Raw requests only supported on HTTP transport")

	ctx := context.Background()
	client, err := harness.NewClient(e.serverURL, nil)
	require.NoError(t, err, "Failed to create client")

	// Send raw request
	responseJSON, err := client.CallHTTPRaw(ctx, []byte(scenario.RawRequest))
	if err != nil {
		if scenario.Expect.Error != nil {
			// Expected error
			return
		}
		require.NoError(t, err, "Raw call failed")
	}

	// Parse response
	var resp any
	err = json.Unmarshal(responseJSON, &resp)
	require.NoError(t, err, "Failed to parse response")

	// Validate response
	e.validateRawResponse(t, resp, scenario.Expect)
}

// Validation methods

func (e *executor) validateJSONRPCResponse(t *testing.T, response any, expect Expect) {
	t.Helper()

	respMap, ok := response.(map[string]any)
	require.True(t, ok, "Expected map response, got %T", response)

	// Check ID
	if expect.ID != nil {
		assert.EqualValues(t, expect.ID, respMap["id"], "ID mismatch")
	}

	// Check result or error
	if expect.Error != nil {
		// Expecting error
		errObj, ok := respMap["error"].(map[string]any)
		require.True(t, ok, "Expected error response, got result: %v", respMap["result"])

		e.validateErrorObject(t, errObj, expect.Error)
	} else {
		// Expecting result
		_, hasError := respMap["error"]
		require.False(t, hasError, "Expected result, got error: %v", respMap["error"])

		// Use JSONEq for complex types or EqualValues for simple types
		expectedJSON, errExp := json.Marshal(expect.Result)
		actualJSON, errAct := json.Marshal(respMap["result"])
		if errExp == nil && errAct == nil {
			assert.JSONEq(t, string(expectedJSON), string(actualJSON), "Result mismatch")
		} else {
			assert.EqualValues(t, expect.Result, respMap["result"], "Result mismatch")
		}
	}
}

// compareJSONRPCMessages compares two JSON-RPC messages (used for SSE/WebSocket validation)
func (e *executor) compareJSONRPCMessages(t *testing.T, actual, expected map[string]any) {
	t.Helper()

	// Compare jsonrpc version
	if actualVersion, ok := actual["jsonrpc"].(string); ok {
		expectedVersion, _ := expected["jsonrpc"].(string)
		require.Equal(t, expectedVersion, actualVersion, "JSON-RPC version mismatch")
	}

	// Compare method only if explicitly expected
	if _, expHas := expected["method"]; expHas {
		actualMethod, ok := actual["method"].(string)
		require.True(t, ok, "Expected method in response")
		expectedMethod, _ := expected["method"].(string)
		require.Equal(t, expectedMethod, actualMethod, "Method mismatch")
	}

	// Compare params
	if expectedParams, ok := expected["params"]; ok {
		actualParams, ok := actual["params"]
		require.True(t, ok, "Expected params in response")
		e.compareValues(t, actualParams, expectedParams, "params")
	}

	// Compare result
	if expectedResult, ok := expected["result"]; ok {
		actualResult, ok := actual["result"]
		require.True(t, ok, "Expected result in response")
		e.compareValues(t, actualResult, expectedResult, "result")
	}

	// Compare error
	if expectedError, ok := expected["error"]; ok {
		actualError, ok := actual["error"]
		require.True(t, ok, "Expected error in response")
		expectedErrObj, ok := expectedError.(map[string]any)
		if !ok {
			e.compareValues(t, actualError, expectedError, "error")
			return
		}
		actualErrObj, ok := actualError.(map[string]any)
		require.True(t, ok, "Expected object error in response")
		for key, expectedValue := range expectedErrObj {
			actualValue, ok := actualErrObj[key]
			require.Truef(t, ok, "Expected error.%s in response", key)
			e.compareValues(t, actualValue, expectedValue, "error."+key)
		}
	}

	// Compare id
	if expectedID, ok := expected["id"]; ok {
		actualID, ok := actual["id"]
		require.True(t, ok, "Expected id in response")
		e.compareValues(t, actualID, expectedID, "id")
	}
}

func (e *executor) debugf(t *testing.T, format string, args ...any) {
	t.Helper()
	if !e.config.Debug {
		return
	}
	t.Logf(format, args...)
}

func (e *executor) validateBatchResponse(t *testing.T, _ int, response map[string]any, expect Expect) {
	t.Helper()

	// Batch responses are validated the same way
	e.validateJSONRPCResponse(t, response, expect)
}

func (e *executor) validateRawResponse(t *testing.T, response any, expect Expect) {
	t.Helper()

	// Raw responses might not be standard JSON-RPC
	if expect.Error != nil {
		// For raw requests, we might get non-standard errors
		return
	}

	// Try to validate as JSON-RPC response
	if respMap, ok := response.(map[string]any); ok {
		e.validateJSONRPCResponse(t, respMap, expect)
	} else {
		// Just compare directly
		assert.EqualValues(t, expect.Result, response, "Raw response mismatch")
	}
}

func (e *executor) validateError(t *testing.T, err error, expect *ExpectError) {
	t.Helper()

	require.Error(t, err, "Expected error")
	require.NotNil(t, expect, "Expected error expectation")

	errObj, ok := extractExpectedErrorObject(err)
	require.Truef(t, ok, "Failed to parse JSON-RPC error from %q", err.Error())
	e.validateErrorObject(t, errObj, expect)
}

func (e *executor) validateErrorObject(t *testing.T, errObj map[string]any, expect *ExpectError) {
	t.Helper()

	// Check error code
	code, ok := errObj["code"].(float64)
	require.True(t, ok, "Missing or invalid error code")
	assert.EqualValues(t, expect.Code, int(code), "Error code mismatch")

	// Check error message
	msg, ok := errObj["message"].(string)
	require.True(t, ok, "Missing or invalid error message")
	assert.Equal(t, expect.Message, msg, "Error message mismatch")

	// Check error data if expected
	if expect.Data != nil {
		assert.Equal(t, expect.Data, errObj["data"], "Error data mismatch")
	}
}

// compareValues compares two values, handling both simple and complex types
func (e *executor) compareValues(t *testing.T, actual, expected any, path string) {
	t.Helper()

	// Try JSON comparison first for complex types
	expectedJSON, errExp := json.Marshal(expected)
	actualJSON, errAct := json.Marshal(actual)
	if errExp == nil && errAct == nil {
		assert.JSONEq(t, string(expectedJSON), string(actualJSON), "%s mismatch", path)
	} else {
		// Fall back to direct comparison
		assert.EqualValues(t, expected, actual, "%s mismatch", path)
	}
}

func extractExpectedErrorObject(err error) (map[string]any, bool) {
	if err == nil {
		return nil, false
	}
	for _, line := range strings.Split(err.Error(), "\n") {
		errObj, ok := extractExpectedErrorObjectLine(line)
		if ok {
			return errObj, true
		}
	}
	return nil, false
}

func extractExpectedErrorObjectLine(line string) (map[string]any, bool) {
	line = strings.TrimSpace(line)
	for start := strings.Index(line, "{"); start >= 0; {
		var payload map[string]any
		if json.Unmarshal([]byte(strings.TrimSpace(line[start:])), &payload) == nil {
			if errObj, ok := payload["error"].(map[string]any); ok {
				return errObj, true
			}
			if _, ok := payload["code"]; ok {
				if _, ok := payload["message"]; ok {
					return payload, true
				}
			}
		}
		next := strings.Index(line[start+1:], "{")
		if next < 0 {
			break
		}
		start += next + 1
	}
	return nil, false
}
