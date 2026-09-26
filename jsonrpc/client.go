package jsonrpc

import (
	"bytes"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/http"

	loomhttp "github.com/CaliLuke/loom/http"
)

// DecodeNotificationResponse checks the HTTP response to a notification, a
// request without an ID, sent by the generated HTTP client of the method
// named method of the service named svc. A server sends no JSON-RPC response
// to a notification, so a 2xx response whose body holds no JSON value is a
// success. A response that holds a JSON-RPC error, such as the Invalid
// Request error of a server that could not read the notification, returns
// that error as a *RawErrorResponse; any other response body is ignored. A
// response status other than 2xx returns loomhttp.ErrInvalidResponse and a
// body that is not a JSON-RPC response returns loomhttp.ErrDecodingError.
// DecodeNotificationResponse reads and closes the body, then replaces it
// with the bytes that it read.
func DecodeNotificationResponse(svc, method string, resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if cerr := resp.Body.Close(); err == nil {
		err = cerr
	}
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return loomhttp.ErrDecodingError(svc, method, fmt.Errorf("read notification response: %w", err))
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return loomhttp.ErrInvalidResponse(svc, method, resp.StatusCode, string(body))
	}
	if len(bytes.TrimSpace(body)) == 0 {
		return nil
	}
	var response RawResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return loomhttp.ErrDecodingError(svc, method, err)
	}
	if response.Error != nil {
		return response.Error
	}
	return nil
}
