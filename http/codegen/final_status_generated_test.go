package codegen

// finalStatusHarness runs in the same maintained generated module as the typed
// handshake specimen, covering ordinary final selection and error ownership.
const finalStatusHarness = `package handshake_test

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	client "example.com/handshake/gen/http/status_selection/client"
	server "example.com/handshake/gen/http/status_selection/server"
	svc "example.com/handshake/gen/status_selection"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestFinalStatusRoundTrip(t *testing.T) {
	for _, status := range []int{201, 202, 299, 303, 500} {
		response := httptest.NewRecorder()
		err := server.EncodeCallResponse(loomhttp.ResponseEncoder)(context.Background(), response, &svc.StatusResult{Status: status, Message: "whole body", Trace: "trace"})
		if err != nil {
			t.Fatal(err)
		}
		if response.Code != status || response.Header().Get("X-Trace") != "trace" || strings.Contains(response.Body.String(), "status") {
			t.Errorf("status/body/headers: %d %s %v", response.Code, response.Body, response.Header())
		}
		value, err := client.DecodeCallResponse(loomhttp.ResponseDecoder, false)(response.Result())
		if err != nil {
			t.Fatal(err)
		}
		got := value.(*svc.StatusResult)
		if got.Status != status || got.Message != "whole body" || got.Trace != "trace" {
			t.Errorf("reconstructed result: %#v", got)
		}
	}
}

func TestFinalStatusRejectsBeforeWrite(t *testing.T) {
	for _, status := range []int{0, 100, 203, 204, 304, 401, 404, 600} {
		response := httptest.NewRecorder()
		err := server.EncodeCallResponse(loomhttp.ResponseEncoder)(context.Background(), response, &svc.StatusResult{Status: status, Message: "body", Trace: "trace"})
		var violation *loomhttp.StatusViolationError
		if !errors.As(err, &violation) || response.Body.Len() != 0 || len(response.Header()) != 0 {
			t.Errorf("status %d published invalid success: err=%v body=%s header=%v", status, err, response.Body, response.Header())
		}
	}
	response := httptest.NewRecorder()
	err := server.EncodeFailResponse(loomhttp.ResponseEncoder)(context.Background(), response, nil)
	var violation *loomhttp.StatusViolationError
	if !errors.As(err, &violation) || response.Body.Len() != 0 || len(response.Header()) != 0 {
		t.Errorf("NoSuccessResponse admitted nil success: %v", err)
	}
}

func TestFinalStatusErrorOwnership(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		valid  bool
	}{
		{"exact", 401, true},
		{"class", 402, true},
		{"other", 499, true},
		{"class", 401, false},
		{"exact", 402, false},
	} {
		response := httptest.NewRecorder()
		failure := &svc.StatusFailure{Name: tc.name, Status: tc.status, Message: "failure"}
		err := server.EncodeCallError(loomhttp.ResponseEncoder, nil)(context.Background(), response, failure)
		if !tc.valid {
			var violation *loomhttp.StatusViolationError
			if !errors.As(err, &violation) || response.Body.Len() != 0 || len(response.Header()) != 0 {
				t.Errorf("error %s/%d escaped selected key: %v", tc.name, tc.status, err)
			}
			continue
		}
		if err != nil || response.Code != tc.status {
			t.Fatalf("encode %s/%d: %v status=%d", tc.name, tc.status, err, response.Code)
		}
		_, err = client.DecodeCallResponse(loomhttp.ResponseDecoder, false)(response.Result())
		var decoded *svc.StatusFailure
		if !errors.As(err, &decoded) || decoded.Name != tc.name || decoded.Status != tc.status || decoded.Message != "failure" {
			t.Errorf("decoded %s/%d: %#v %v", tc.name, tc.status, decoded, err)
		}
	}
}

func TestFinalStatusNativeDecodeConstraint(t *testing.T) {
	response := &http.Response{StatusCode: 203, Header: http.Header{"X-Trace": {"trace"}}, Body: io.NopCloser(strings.NewReader("{\"message\":\"whole body\"}"))}
	value, err := client.DecodeCallResponse(loomhttp.ResponseDecoder, false)(response)
	if err == nil || value != nil {
		t.Errorf("owned status bypassed native enum: %v %v", value, err)
	}
}

func TestSelectedFinalPreparation(t *testing.T) {
	message := "selected"
	for _, tc := range []struct {
		name   string
		value  *svc.BranchResult
		valid  bool
		absent bool
	}{
		{"selected primitive", &svc.BranchResult{Status: 201, Message: &message, Trace: "trace", Session: "session"}, true, false},
		{"selected empty collection", &svc.BranchResult{Status: 202, Entries: []string{}, Trace: "trace", Session: "session"}, true, false},
		{"missing primitive", &svc.BranchResult{Status: 201, Trace: "trace", Session: "session"}, false, true},
		{"missing collection", &svc.BranchResult{Status: 202, Trace: "trace", Session: "session"}, false, true},
		{"native header", &svc.BranchResult{Status: 201, Message: &message, Trace: "x", Session: "session"}, false, false},
		{"mapped header", &svc.BranchResult{Status: 201, Message: &message, Trace: "four", Session: "session"}, false, false},
		{"cookie", &svc.BranchResult{Status: 201, Message: &message, Trace: "trace", Session: "x"}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			response := httptest.NewRecorder()
			err := server.EncodeBranchResponse(loomhttp.ResponseEncoder)(context.Background(), response, tc.value)
			if tc.valid {
				if err != nil || response.Code != tc.value.Status || response.Header().Get("X-Trace") != "trace" || len(response.Result().Cookies()) != 1 {
					t.Fatalf("valid branch: %v %d %v %s", err, response.Code, response.Header(), response.Body)
				}
				return
			}
			var violation *loomhttp.StatusViolationError
			if tc.absent {
				var named interface {
					LoomErrorName() string
				}
				if err == nil || err.Error() != "selected response value is absent" || errors.As(err, &violation) || errors.As(err, &named) {
					t.Errorf("selected body absence category: %T %v", err, err)
				}
			} else if !errors.As(err, &violation) {
				t.Errorf("metadata violation category: %T %v", err, err)
			}
			if response.Body.Len() != 0 || len(response.Header()) != 0 {
				t.Errorf("invalid branch published: %v %d %v %s", err, response.Code, response.Header(), response.Body)
			}
		})
	}
}
`
