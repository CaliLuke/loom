package http

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateResponseContract(t *testing.T) {
	tests := []struct {
		name     string
		response *http.Response
		contract ResponseContractCase
		wantErr  string
		contains bool
	}{
		{
			name: "success",
			response: &http.Response{
				StatusCode: http.StatusAccepted,
				Header: http.Header{
					"Content-Type": []string{"application/json; charset=utf-8"},
					"X-Version":    []string{""},
					"Set-Cookie":   []string{"widget_session=abc; Path=/"},
				},
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.202",
				Kind:            ResponseContractSuccess,
				StatusCode:      http.StatusAccepted,
				ContentTypes:    []string{"application/json"},
				RequiredHeaders: []string{"X-Version"},
				RequiredCookies: []string{"widget_session"},
			},
		},
		{
			name: "error",
			response: &http.Response{
				StatusCode: http.StatusNotFound,
				Header: http.Header{
					"Content-Type": []string{"application/problem+json"},
					"Loom-Error":   []string{"not_found"},
				},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.error.not_found.404",
				Kind:         ResponseContractError,
				StatusCode:   http.StatusNotFound,
				ErrorName:    "not_found",
				ContentTypes: []string{"application/problem+json"},
			},
		},
		{
			name:     "nil response",
			contract: ResponseContractCase{ID: "widgets.show.success.200"},
			wantErr:  `response contract "widgets.show.success.200": response is nil`,
		},
		{
			name: "status mismatch",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{ID: "widgets.show.success.202", StatusCode: http.StatusAccepted},
			wantErr:  `response contract "widgets.show.success.202": status is 200, want 202`,
		},
		{
			name: "malformed content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json; charset"}},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/json"},
			},
			wantErr:  `response contract "widgets.show.success.200": parse Content-Type "application/json; charset"`,
			contains: true,
		},
		{
			name: "content type mismatch",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"text/plain"}},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/json; profile=widget"},
			},
			wantErr: `response contract "widgets.show.success.200": Content-Type is "text/plain", want one of [application/json]`,
		},
		{
			name: "wildcard content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/pdf"}},
			},
			contract: ResponseContractCase{
				ID:           "files.download.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"*/*"},
			},
		},
		{
			name: "type wildcard content type",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/problem+json"}},
			},
			contract: ResponseContractCase{
				ID:           "errors.show.success.200",
				StatusCode:   http.StatusOK,
				ContentTypes: []string{"application/*"},
			},
		},
		{
			name: "error name mismatch",
			response: &http.Response{
				StatusCode: http.StatusNotFound,
				Header: http.Header{
					"Content-Type": []string{"application/problem+json"},
					"Loom-Error":   []string{"gone"},
				},
			},
			contract: ResponseContractCase{
				ID:           "widgets.show.error.not_found.404",
				Kind:         ResponseContractError,
				StatusCode:   http.StatusNotFound,
				ErrorName:    "not_found",
				ContentTypes: []string{"application/problem+json"},
			},
			wantErr: `response contract "widgets.show.error.not_found.404": Loom-Error is "gone", want "not_found"`,
		},
		{
			name: "success carries error name",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Loom-Error": []string{"not_found"}},
			},
			contract: ResponseContractCase{
				ID:         "widgets.show.success.200",
				Kind:       ResponseContractSuccess,
				StatusCode: http.StatusOK,
			},
			wantErr: `response contract "widgets.show.success.200": Loom-Error is "not_found", want empty`,
		},
		{
			name: "missing required header",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.200",
				StatusCode:      http.StatusOK,
				RequiredHeaders: []string{"X-Version"},
			},
			wantErr: `response contract "widgets.show.success.200": required header "X-Version" is missing`,
		},
		{
			name: "missing required cookie",
			response: &http.Response{
				StatusCode: http.StatusOK,
				Header:     make(http.Header),
			},
			contract: ResponseContractCase{
				ID:              "widgets.show.success.200",
				StatusCode:      http.StatusOK,
				RequiredCookies: []string{"widget_session"},
			},
			wantErr: `response contract "widgets.show.success.200": required cookie "widget_session" is missing`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateResponseContract(test.response, test.contract)
			if test.wantErr == "" {
				require.NoError(t, err)
				return
			}
			if test.contains {
				require.ErrorContains(t, err, test.wantErr)
				return
			}
			require.EqualError(t, err, test.wantErr)
		})
	}
}
