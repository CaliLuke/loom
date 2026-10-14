package jsonrpc

import (
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	loomhttp "github.com/CaliLuke/loom/http"
)

type candidateFailCloser struct {
	io.Reader
	err error
}

func TestDecodeErrorResponseSelectors(t *testing.T) {
	selected := errors.New("selected")
	for _, tc := range []struct {
		name     string
		data     string
		selector []string
		calls    int
	}{
		{"mapped", `{"error_kind":"left","name":"right"}`, []string{"error_kind"}, 1},
		{"root", `"left"`, []string{}, 1},
		{"missing", `{"name":"left"}`, []string{"error_kind"}, 0},
		{"null", `{"error_kind":null}`, []string{"error_kind"}, 0},
		{"wrong type", `{"error_kind":7}`, []string{"error_kind"}, 0},
		{"unknown", `{"error_kind":"other"}`, []string{"error_kind"}, 0},
		{"malformed", `{"error_kind":`, []string{"error_kind"}, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			candidate := ErrorCandidate{Name: "left", Selector: tc.selector, Decode: func(*http.Response) (error, error) {
				calls++
				return selected, nil
			}}
			got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: []byte(tc.data)}, nil, []ErrorCandidate{candidate})
			require.Equal(t, tc.calls, calls)
			if tc.calls == 1 {
				require.ErrorIs(t, got, selected)
			} else {
				require.NotErrorIs(t, got, selected)
				var invalid *loomhttp.ClientError
				require.ErrorAs(t, got, &invalid)
			}
		})
	}
}

func TestDecodeErrorResponseDifferentSelectorKeys(t *testing.T) {
	left, right := errors.New("left"), errors.New("right")
	for _, tc := range []struct {
		data string
		want error
	}{
		{`{"left_key":"left","name":"right"}`, left},
		{`{"right_key":"right","name":"left"}`, right},
		{`{"left_key":"left","right_key":"right"}`, nil},
	} {
		calls := 0
		candidates := []ErrorCandidate{
			{Name: "left", Selector: []string{"left_key"}, Decode: func(*http.Response) (error, error) {
				calls++
				return left, nil
			}},
			{Name: "right", Selector: []string{"right_key"}, Decode: func(*http.Response) (error, error) {
				calls++
				return right, nil
			}},
		}
		got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: []byte(tc.data)}, nil, candidates)
		if tc.want != nil {
			require.ErrorIs(t, got, tc.want)
			require.Equal(t, 1, calls)
		} else {
			var invalid *loomhttp.ClientError
			require.ErrorAs(t, got, &invalid)
			require.Contains(t, invalid.Message, "ambiguous")
			require.Equal(t, 2, calls)
		}
	}
}

func TestDecodeErrorResponsePrecedence(t *testing.T) {
	selected := errors.New("selected")
	unexpected := errors.New("custom decoder sentinel")
	validation := errors.New("declared minimum")
	success := ErrorCandidate{Name: "left", Decode: func(*http.Response) (error, error) {
		return selected, nil
	}}
	for _, tc := range []struct {
		name      string
		failure   func(*http.Response) (error, error)
		want      error
		ambiguous bool
	}{
		{name: "structured conversion", failure: func(r *http.Response) (error, error) {
			var value int
			err := json.UnmarshalRead(r.Body, &value)
			return nil, loomhttp.ErrDecodingError("svc", "method", err)
		}, want: selected},
		{name: "generated validation", failure: func(*http.Response) (error, error) {
			return nil, loomhttp.ErrValidationError("svc", "method", validation)
		}, want: selected},
		{name: "unexpected after success", failure: func(*http.Response) (error, error) {
			return nil, loomhttp.ErrDecodingError("svc", "method", unexpected)
		}, want: unexpected},
		{name: "application validation-shaped error remains opaque", failure: func(*http.Response) (error, error) {
			return nil, loomhttp.ErrDecodingError("svc", "method", loomhttp.ErrValidationError("app", "codec", unexpected))
		}, want: unexpected},
		{name: "ambiguity", failure: success.Decode, ambiguous: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, reverse := range []bool{false, true} {
				candidates := []ErrorCandidate{success, {Name: "right", Decode: tc.failure}}
				if reverse {
					slices.Reverse(candidates)
				}
				got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: []byte(`"value"`)}, nil, candidates)
				if tc.ambiguous {
					var invalid *loomhttp.ClientError
					require.ErrorAs(t, got, &invalid)
					require.Contains(t, invalid.Message, "ambiguous")
					require.NotErrorIs(t, got, selected)
				} else {
					require.ErrorIs(t, got, tc.want)
				}
			}
		})
	}
	rejected := ErrorCandidate{Name: "left", Decode: func(*http.Response) (error, error) {
		return nil, loomhttp.ErrValidationError("svc", "method", validation)
	}}
	got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: []byte(`{}`)}, nil, []ErrorCandidate{rejected})
	require.ErrorIs(t, got, validation)
}

func TestDecodeErrorResponsePrivateTrials(t *testing.T) {
	source := &http.Response{StatusCode: http.StatusOK, TransferEncoding: []string{"chunked"}, Header: http.Header{"X-Value": {"original"}}, Body: io.NopCloser(strings.NewReader("untouched"))}
	selected := errors.New("selected")
	data := []byte(`{"kind":"left"}`)
	calls := 0
	candidates := []ErrorCandidate{
		{Name: "excluded", Selector: []string{"kind"}, Decode: func(*http.Response) (error, error) {
			t.Error("ineligible decoder invoked")
			return nil, errors.New("ineligible")
		}},
		{Name: "left", Selector: []string{"kind"}, Decode: func(r *http.Response) (error, error) {
			calls++
			r.Header.Set("X-Value", "modified")
			r.TransferEncoding[0] = "modified"
			bytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, data, bytes)
			return selected, nil
		}},
		{Name: "mismatch", Decode: func(r *http.Response) (error, error) {
			calls++
			require.Equal(t, "original", r.Header.Get("X-Value"))
			require.Equal(t, []string{"chunked"}, r.TransferEncoding)
			bytes, err := io.ReadAll(r.Body)
			require.NoError(t, err)
			require.Equal(t, data, bytes)
			return nil, loomhttp.ErrValidationError("svc", "method", errors.New("mismatch"))
		}},
	}
	got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Code: 4001, Data: data}, source, candidates)
	require.ErrorIs(t, got, selected)
	require.Equal(t, 2, calls)
	require.Equal(t, "original", source.Header.Get("X-Value"))
	require.Equal(t, []string{"chunked"}, source.TransferEncoding)
	body, err := io.ReadAll(source.Body)
	require.NoError(t, err)
	require.Equal(t, "untouched", string(body))
	raw := &RawErrorResponse{Code: 999, Data: []byte(`{"name":"raw"}`)}
	require.Same(t, raw, DecodeErrorResponse("svc", "method", raw, source, nil))
}

func TestDecodeErrorResponseCloseFailurePrecedence(t *testing.T) {
	closeFailure := errors.New("close sentinel")
	selected := errors.New("selected")
	for _, reverse := range []bool{false, true} {
		candidates := []ErrorCandidate{
			{Name: "mismatch", Decode: func(r *http.Response) (error, error) {
				r.Body = candidateFailCloser{Reader: strings.NewReader(""), err: closeFailure}
				return nil, loomhttp.ErrValidationError("svc", "method", errors.New("mismatch"))
			}},
			{Name: "match", Decode: func(*http.Response) (error, error) {
				return selected, nil
			}},
		}
		if reverse {
			slices.Reverse(candidates)
		}
		got := DecodeErrorResponse("svc", "method", &RawErrorResponse{Data: []byte(`{}`)}, nil, candidates)
		require.ErrorIs(t, got, closeFailure)
		require.NotErrorIs(t, got, selected)
	}
}

func (r candidateFailCloser) Close() error {
	return r.err
}
