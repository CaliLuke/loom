package jsonrpc

import (
	"bytes"
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	loomhttp "github.com/CaliLuke/loom/http"
)

type (
	// fuzzDispatchMode selects how the fuzz dispatcher answers one request.
	fuzzDispatchMode byte

	// fuzzExpectation is the oracle's view of one JSON-RPC request.
	fuzzExpectation struct {
		respond  bool
		code     Code
		echo     bool
		success  bool
		hasID    bool
		idKind   jsontext.Kind
		idString string
		idRaw    jsontext.Value
	}

	// fuzzDispatcher drives HTTPHandlerSpec callbacks from fuzz input.
	fuzzDispatcher struct {
		modes   []byte
		index   int
		failing *RawRequest
		mode    fuzzDispatchMode
	}
)

const (
	fuzzDispatchEncode fuzzDispatchMode = iota
	fuzzDispatchChunked
	fuzzDispatchFailureText
	fuzzDispatchFailureJSON
	fuzzDispatchSilent
	fuzzDispatchTwice
	fuzzDispatchModeCount
)

// FuzzHTTPHandler checks JSON-RPC 2.0 envelope, batch, and framing invariants
// for arbitrary request bodies and adapter write patterns.
func FuzzHTTPHandler(f *testing.F) {
	seeds := []string{
		``,
		`   `,
		`{`,
		`[`,
		`[]`,
		` [ ] `,
		`[[]]`,
		`[[{"jsonrpc":"2.0","method":"echo","id":1}]]`,
		`null`,
		`1`,
		`"x"`,
		`[1,2,3]`,
		`[null]`,
		`{"jsonrpc":"2.0","method":"echo","id":"one"}`,
		`{"jsonrpc":"2.0","method":"echo","id":1}`,
		`{"jsonrpc":"2.0","method":"echo","id":null}`,
		`{"jsonrpc":"2.0","method":"echo"}`,
		`{"jsonrpc":"2.0","method":"echo","id":{}}`,
		`{"jsonrpc":"2.0","method":"echo","id":[1]}`,
		`{"jsonrpc":"2.0","method":"echo","id":true}`,
		`{"jsonrpc":"2.0","method":"echo","id":123456789012345678901234567890}`,
		`{"jsonrpc":"2.0","method":"echo","id":-1.5e300}`,
		`{"jsonrpc":"2.0","method":"echo","id":"\u00e9\ud83d\ude00"}`,
		`{"jsonrpc":"1.0","method":"echo","id":1}`,
		`{"jsonrpc":2,"method":"echo","id":1}`,
		`{"jsonrpc":"2.0","method":7,"id":1}`,
		`{"jsonrpc":"2.0","method":"","id":1}`,
		`{"jsonrpc":"2.0","id":1}`,
		`{"jsonrpc":"2.0","method":"missing","id":1}`,
		`{"jsonrpc":"2.0","method":"missing"}`,
		`{"jsonrpc":"2.0","method":"echo","id":1,"id":2}`,
		`{"jsonrpc":"2.0","method":"echo","params":[1,{"a":[]}],"id":1}`,
		"\t\r\n [{\"jsonrpc\":\"2.0\",\"method\":\"echo\",\"id\":1}]",
		`[{"jsonrpc":"2.0","method":"echo","id":1},{"jsonrpc":"2.0","method":"echo"},{"jsonrpc":"2.0","method":"echo","id":"two"}]`,
		`[{"jsonrpc":"2.0","method":"echo"},{"jsonrpc":"2.0","method":"echo"}]`,
		`[{"jsonrpc":"2.0","method":"echo","id":1},{"foo":"boo"},{"jsonrpc":"2.0","method":"missing","id":"9"},1]`,
		`[{"jsonrpc":"2.0","method":"echo","id":1},{"jsonrpc":"2.0","method"`,
		`[{"jsonrpc":"2.0","method":"echo","id":1}] trailing`,
		`{"jsonrpc":"2.0","method":"echo","id":1} {}`,
	}
	for _, seed := range seeds {
		for mode := range fuzzDispatchModeCount {
			f.Add([]byte(seed), []byte{byte(mode)})
		}
		f.Add([]byte(seed), []byte{0, 1, 2, 3, 4, 5})
	}

	f.Fuzz(func(t *testing.T, body []byte, modes []byte) {
		dispatcher := &fuzzDispatcher{modes: modes}
		handler := NewHTTPHandler(dispatcher.spec())
		request := httptest.NewRequest(http.MethodPost, "/rpc", bytes.NewReader(body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()

		handler.ServeHTTP(response, request)

		checkFuzzResponse(t, body, modes, response.Body.Bytes())
	})
}

func checkFuzzResponse(t *testing.T, body, modes, output []byte) {
	t.Helper()
	isBatch := fuzzIsBatch(body)
	if !jsontext.Value(body).IsValid() || jsonWhitespace(body) {
		requireFuzzSingleError(t, output, ParseError)
		return
	}
	if !isBatch {
		expectation := fuzzExpect(body)
		if expectation.echo {
			if !fuzzDeterministicSingle(modes) {
				return
			}
			expectation.success = fuzzModeSucceeds(fuzzModeAt(modes, 0))
		}
		if !expectation.respond {
			if len(bytes.TrimSpace(output)) != 0 {
				t.Errorf("notification produced response %q for %q", output, body)
			}
			return
		}
		var envelope map[string]jsontext.Value
		if err := json.Unmarshal(output, &envelope); err != nil {
			t.Errorf("single response %q is not a JSON object: %v", output, err)
			return
		}
		checkFuzzEntry(t, envelope, expectation, body)
		return
	}
	var items []jsontext.Value
	if err := json.Unmarshal(body, &items); err != nil {
		t.Errorf("valid batch %q did not decode as an array: %v", body, err)
		return
	}
	if len(items) == 0 {
		requireFuzzSingleError(t, output, InvalidRequest)
		return
	}
	var expectations []fuzzExpectation
	dispatched := 0
	for _, item := range items {
		expectation := fuzzExpect(item)
		if expectation.echo {
			expectation.success = fuzzModeSucceeds(fuzzModeAt(modes, dispatched))
			dispatched++
		}
		if expectation.respond {
			expectations = append(expectations, expectation)
		}
	}
	if len(expectations) == 0 {
		if len(output) != 0 {
			t.Errorf("all-notification batch produced response %q", output)
		}
		return
	}
	if !jsontext.Value(output).IsValid() {
		t.Errorf("batch response %q is not valid JSON for request %q", output, body)
		return
	}
	var entries []map[string]jsontext.Value
	if err := json.Unmarshal(output, &entries); err != nil {
		t.Errorf("batch response %q is not an array of objects: %v", output, err)
		return
	}
	if len(entries) != len(expectations) {
		t.Errorf("batch response has %d entries, want %d for request %q: %q", len(entries), len(expectations), body, output)
		return
	}
	for i, entry := range entries {
		checkFuzzEntry(t, entry, expectations[i], body)
	}
}

func checkFuzzEntry(t *testing.T, entry map[string]jsontext.Value, expectation fuzzExpectation, body []byte) {
	t.Helper()
	if string(entry["jsonrpc"]) != `"2.0"` {
		t.Errorf("entry %v has jsonrpc %q for request %q", entry, entry["jsonrpc"], body)
	}
	id, ok := entry["id"]
	if !ok {
		t.Errorf("entry %v has no id member for request %q", entry, body)
		return
	}
	checkFuzzID(t, id, expectation, body)
	_, hasResult := entry["result"]
	rawError, hasError := entry["error"]
	if hasResult == hasError {
		t.Errorf("entry %v must contain exactly one of result and error for request %q", entry, body)
		return
	}
	if hasResult != expectation.success {
		t.Errorf("entry %v success is %t, want %t for request %q", entry, hasResult, expectation.success, body)
		return
	}
	if hasResult {
		return
	}
	var responseError RawErrorResponse
	if err := json.Unmarshal(rawError, &responseError); err != nil {
		t.Errorf("entry error %q did not decode: %v", rawError, err)
		return
	}
	want := expectation.code
	if expectation.echo {
		want = InternalError
	}
	if Code(responseError.Code) != want {
		t.Errorf("entry error code %d, want %d for request %q", responseError.Code, want, body)
	}
}

func checkFuzzID(t *testing.T, id jsontext.Value, expectation fuzzExpectation, body []byte) {
	t.Helper()
	kind := expectation.idKind
	if !expectation.hasID {
		kind = 'n'
	}
	switch kind {
	case '"':
		var got string
		if err := json.Unmarshal(id, &got); err != nil || got != expectation.idString {
			t.Errorf("entry id %q, want string %q for request %q", id, expectation.idString, body)
		}
	case '0':
		if !bytes.Equal(id, expectation.idRaw) {
			t.Errorf("entry id %q, want number %q for request %q", id, expectation.idRaw, body)
		}
	default:
		if string(id) != "null" {
			t.Errorf("entry id %q, want null for request %q", id, body)
		}
	}
}

func requireFuzzSingleError(t *testing.T, output []byte, code Code) {
	t.Helper()
	var response RawResponse
	if err := json.Unmarshal(output, &response); err != nil {
		t.Errorf("error response %q is not a JSON object: %v", output, err)
		return
	}
	if response.Error == nil || Code(response.Error.Code) != code {
		t.Errorf("response %q, want error code %d", output, code)
	}
	if response.ID != nil {
		t.Errorf("response %q, want null id", output)
	}
}

// fuzzExpect independently classifies one JSON-RPC request value.
func fuzzExpect(value []byte) fuzzExpectation {
	var object map[string]jsontext.Value
	if err := json.Unmarshal(value, &object); err != nil || object == nil {
		return fuzzExpectation{respond: true, code: InvalidRequest}
	}
	expectation := fuzzExpectation{}
	rawID, hasID := object["id"]
	if hasID {
		expectation.hasID = true
		expectation.idKind = rawID.Kind()
		switch expectation.idKind {
		case '"':
			if err := json.Unmarshal(rawID, &expectation.idString); err != nil {
				return fuzzExpectation{respond: true, code: InvalidRequest}
			}
		case '0':
			expectation.idRaw = rawID
		case 'n':
		default:
			expectation.respond = true
			expectation.code = InvalidRequest
			return expectation
		}
	}
	var version, method string
	versionOK := json.Unmarshal(object["jsonrpc"], &version) == nil && version == "2.0"
	rawMethod, methodPresent := object["method"]
	methodOK := !methodPresent || json.Unmarshal(rawMethod, &method) == nil
	switch {
	case !versionOK || !methodOK || method == "":
		expectation.respond = true
		expectation.code = InvalidRequest
	case method == "echo":
		expectation.respond = hasID
		expectation.echo = true
	case !hasID:
		expectation.respond = false
	default:
		expectation.respond = true
		expectation.code = MethodNotFound
	}
	return expectation
}

func fuzzIsBatch(body []byte) bool {
	for i, b := range body {
		if i >= maxJSONPrefixWhitespace {
			return false
		}
		switch b {
		case ' ', '\t', '\r', '\n':
			continue
		case '[':
			return true
		default:
			return false
		}
	}
	return false
}

func fuzzDeterministicSingle(modes []byte) bool {
	switch fuzzModeAt(modes, 0) {
	case fuzzDispatchEncode, fuzzDispatchChunked, fuzzDispatchFailureJSON:
		return true
	default:
		return false
	}
}

func fuzzModeSucceeds(mode fuzzDispatchMode) bool {
	return mode == fuzzDispatchEncode || mode == fuzzDispatchChunked
}

func fuzzModeAt(modes []byte, index int) fuzzDispatchMode {
	if len(modes) == 0 {
		return fuzzDispatchEncode
	}
	return fuzzDispatchMode(modes[index%len(modes)]) % fuzzDispatchModeCount
}

func jsonWhitespace(data []byte) bool {
	for _, b := range data {
		switch b {
		case ' ', '\t', '\r', '\n':
		default:
			return false
		}
	}
	return true
}

func (d *fuzzDispatcher) spec() HTTPHandlerSpec {
	return HTTPHandlerSpec{
		Service:  "echo",
		Decoder:  loomhttp.RequestDecoder,
		Encoder:  loomhttp.ResponseEncoder,
		Dispatch: d.dispatch,
		HandleFailure: func(ctx context.Context, w http.ResponseWriter, err error) {
			request := d.failing
			d.failing = nil
			if request == nil {
				return
			}
			switch d.mode {
			case fuzzDispatchFailureText:
				http.Error(w, err.Error(), http.StatusInternalServerError)
			case fuzzDispatchFailureJSON:
				response := MakeErrorResponse(request.ID, InternalError, "", nil)
				if encodeErr := loomhttp.ResponseEncoder(ctx, w).Encode(response); encodeErr != nil {
					panic(encodeErr)
				}
			}
		},
	}
}

func (d *fuzzDispatcher) dispatch(ctx context.Context, _ *http.Request, request *RawRequest, w http.ResponseWriter) (bool, error) {
	if request.Method != "echo" {
		return false, nil
	}
	mode := fuzzModeAt(d.modes, d.index)
	d.index++
	response := MakeSuccessResponse(request.ID, map[string]any{"ok": true})
	switch mode {
	case fuzzDispatchEncode:
		return true, loomhttp.ResponseEncoder(ctx, w).Encode(response)
	case fuzzDispatchChunked:
		data, err := json.Marshal(response)
		if err != nil {
			return true, err
		}
		for len(data) > 0 {
			size := min(len(data), 1+len(data)%3)
			if _, err := w.Write(data[:size]); err != nil {
				return true, err
			}
			data = data[size:]
		}
		return true, nil
	case fuzzDispatchFailureText, fuzzDispatchFailureJSON:
		d.failing = request
		d.mode = mode
		return true, errors.New("endpoint failed")
	case fuzzDispatchSilent:
		return true, nil
	default:
		encoder := loomhttp.ResponseEncoder(ctx, w)
		if err := encoder.Encode(response); err != nil {
			return true, err
		}
		return true, encoder.Encode(response)
	}
}
