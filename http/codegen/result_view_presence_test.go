package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestResultViewPresenceGeneratedIntegration(t *testing.T) {
	root := RunHTTPDSL(t, testdata.ResultViewPresenceDSL)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/viewpresence", root)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "presence_test.go"), []byte(resultViewPresenceHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "output_test.go"), []byte(resultViewOutputHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "vet", "./...")
	runGoCommand(t, dir, "test", "-race", "-count=1", "./...")
}

const resultViewPresenceHarness = `package presence_test

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	client "example.com/viewpresence/gen/http/presence/client"
	presence "example.com/viewpresence/gen/presence"
	loomhttp "github.com/CaliLuke/loom/http"
)

type responseCase struct {
	name, view, body, missing string
	valid bool
}

func responseCases(t *testing.T) []responseCase {
	t.Helper()
	full := "{\"id\":\"abc\",\"details\":{\"name\":\"Ada\"},\"label\":\"ready\",\"tags\":[\"one\"],\"counts\":{\"one\":1},\"profile\":{\"name\":\"Lin\"}}"
	cases := []responseCase{
		{name: "tiny", view: "tiny", body: "{\"id\":\"abc\"}", valid: true},
		{name: "default", view: "default", body: full, valid: true},
		{name: "implicit_default", body: full, valid: true},
		{name: "unknown", view: "unknown", body: full, missing: "view"},
		{name: "missing_tiny_id", view: "tiny", body: "{}", missing: "id"},
		{name: "missing_default", view: "default", body: "{\"id\":\"abc\"}", missing: "details"},
		{name: "missing_implicit_default", body: "{\"id\":\"abc\"}", missing: "details"},
	}
	for _, field := range []string{"id", "details", "label", "tags", "counts", "profile"} {
		var object map[string]any
		if err := json.Unmarshal([]byte(full), &object); err != nil {
			t.Fatal(err)
		}
		delete(object, field)
		body, err := json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, responseCase{name: "missing_" + field, view: "default", body: string(body), missing: field})
		object[field] = nil
		body, err = json.Marshal(object)
		if err != nil {
			t.Fatal(err)
		}
		cases = append(cases, responseCase{name: "null_" + field, view: "default", body: string(body), missing: field})
	}
	cases = append(cases,
		responseCase{name: "missing_inline_name", view: "default", body: strings.Replace(full, "\"name\":\"Ada\"", "", 1), missing: "name"},
		responseCase{name: "missing_named_name", view: "default", body: strings.Replace(full, "\"name\":\"Lin\"", "", 1), missing: "name"},
	)
	return cases
}

func checkResult(t *testing.T, c responseCase, result *presence.Record, err error) {
	t.Helper()
	if !c.valid {
		var validation *loomhttp.ClientError
		if !errors.As(err, &validation) || validation.Name != "validation_error" || !strings.Contains(validation.Message, c.missing) {
			t.Errorf("error = %v, want validation_error mentioning %q", err, c.missing)
		}
		if result != nil {
			t.Errorf("invalid response returned result: %#v", result)
		}
		return
	}
	if err != nil || result == nil {
		t.Errorf("result=%#v error=%v", result, err)
		return
	}
	if result.ID != "abc" {
		t.Errorf("id=%q", result.ID)
	}
	if c.view == "tiny" {
		if result.Details != nil || result.Profile != nil || result.Label != "" || result.Tags != nil || result.Counts != nil {
			t.Errorf("tiny view populated omitted fields: %#v", result)
		}
		return
	}
	if result.Details == nil || result.Details.Name != "Ada" || result.Profile == nil || result.Profile.Name != "Lin" || result.Label != "ready" || len(result.Tags) != 1 || result.Tags[0] != "one" || result.Counts["one"] != 1 {
		t.Errorf("default view lost values: %#v", result)
	}
}

func TestHTTPViews(t *testing.T) {
	for _, c := range responseCases(t) {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("response conversion panicked: %v", p)
				}
			}()
			header := http.Header{"Content-Type": {"application/json"}}
			if c.view != "" {
				header.Set("Loom-View", c.view)
			}
			resp := &http.Response{StatusCode: http.StatusOK, Header: header, Body: io.NopCloser(strings.NewReader(c.body))}
			raw, err := client.DecodeShowResponse(loomhttp.ResponseDecoder, false)(resp)
			var result *presence.Record
			if raw != nil {
				result = raw.(*presence.Record)
			}
			checkResult(t, c, result, err)
		})
	}
}

func TestProjectionPreservesCollectionPresence(t *testing.T) {
	for _, tags := range [][]string{nil, {}, {"one"}} {
		projected := presence.ProjectRecord(&presence.Record{Tags: tags})
		if !reflect.DeepEqual(projected.Tags, tags) {
			t.Errorf("projected tags=%#v, source=%#v", projected.Tags, tags)
		}
	}
}

func TestWebSocketViews(t *testing.T) {
	for _, c := range responseCases(t) {
		t.Run(c.name, func(t *testing.T) {
			defer func() {
				if p := recover(); p != nil {
					t.Errorf("response conversion panicked: %v", p)
				}
			}()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				upgrader := websocket.Upgrader{}
				conn, err := upgrader.Upgrade(w, r, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer func() {
					if err := conn.Close(); err != nil {
						t.Error(err)
					}
				}()
				if err := conn.WriteMessage(websocket.TextMessage, []byte(c.body)); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			cl := client.NewClient("ws", strings.TrimPrefix(srv.URL, "http://"), srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false, websocket.DefaultDialer, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			raw, err := cl.Watch()(ctx, nil)
			if err != nil {
				t.Fatal(err)
			}
			stream := raw.(*client.WatchClientStream)
			stream.SetView(c.view)
			result, err := stream.Recv()
			checkResult(t, c, result, err)
		})
	}
}
`
