package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/CaliLuke/loom/http/codegen/testdata"
	"github.com/stretchr/testify/require"
)

func TestDiscriminatorHintGeneratedWire(t *testing.T) {
	for _, tc := range []struct {
		name   string
		design func()
	}{
		{"hints", testdata.DiscriminatorHintDSL},
		{"no hints", testdata.DiscriminatorNoHintDSL},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := RunHTTPDSL(t, tc.design)
			dir := t.TempDir()
			renderHTTPModule(t, dir, "example.com/discriminatorhint", root)
			require.NoError(t, os.WriteFile(filepath.Join(dir, "integration_test.go"), []byte(discriminatorHintHarness), 0o644))
			runGoCommand(t, dir, "mod", "tidy")
			runGoCommand(t, dir, "test", "./...")
		})
	}
}

const discriminatorHintHarness = `package integration

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"strings"
	"testing"

	hint "example.com/discriminatorhint/gen/hint"
	hintclient "example.com/discriminatorhint/gen/http/hint/client"
	hintserver "example.com/discriminatorhint/gen/http/hint/server"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestTaggedRootWire(t *testing.T) {
	for _, tc := range []struct {
		name, json string
		form url.Values
	}{
		{"flat", "{\"grant_type\":\"authorization_code\",\"data\":{\"code\":\"abc\"}}", url.Values{"grant_type":{"authorization_code"},"code":{"abc"}}},
		{"nested JSON field", "{\"grant_type\":\"nested_code\",\"data\":{\"details\":{\"code\":\"abc\"}}}", url.Values{"grant_type":{"nested_code"},"details":{"{\"code\":\"abc\"}"}}},
		{"nullable JSON member", "{\"grant_type\":\"nullable_record\",\"data\":{\"details\":null}}", url.Values{"grant_type":{"nullable_record"},"details":{"null"}}},
		{"nil JSON array items", "{\"grant_type\":\"records\",\"data\":[]}", url.Values{"grant_type":{"records"}}},
		{"empty JSON array items", "{\"grant_type\":\"records\",\"data\":[]}", url.Values{"grant_type":{"records"}}},
		{"inner array JSON items", "{\"grant_type\":\"matrix\",\"data\":[[1,2],[3]]}", url.Values{"grant_type":{"matrix"},"data":{"[1,2]","[3]"}}},
		{"JSON array items", "{\"grant_type\":\"records\",\"data\":[{\"code\":\"a\"},{\"code\":\"b\"}]}", url.Values{"grant_type":{"records"},"data":{"{\"code\":\"a\"}","{\"code\":\"b\"}"}}},
		{"nested JSON array items", "{\"grant_type\":\"array_record\",\"data\":{\"records\":[{\"code\":\"a\"},{\"code\":\"b\"}]}}", url.Values{"grant_type":{"array_record"},"records":{"{\"code\":\"a\"}","{\"code\":\"b\"}"}}},
		{"JSON map value", "{\"grant_type\":\"labels\",\"data\":{\"region\":\"west\"}}", url.Values{"grant_type":{"labels"},"data":{"{\"region\":\"west\"}"}}},
		{"nullable JSON map", "{\"grant_type\":\"nullable_labels\",\"data\":null}", url.Values{"grant_type":{"nullable_labels"},"data":{"null"}}},
		{"empty JSON map", "{\"grant_type\":\"nullable_labels\",\"data\":{}}", url.Values{"grant_type":{"nullable_labels"},"data":{"{}"}}},
		{"nil scalar array", "{\"grant_type\":\"tags\",\"data\":[]}", url.Values{"grant_type":{"tags"}}},
		{"empty scalar array", "{\"grant_type\":\"tags\",\"data\":[]}", url.Values{"grant_type":{"tags"}}},
		{"repeated scalar values", "{\"grant_type\":\"tags\",\"data\":[\"a\",\"b\"]}", url.Values{"grant_type":{"tags"},"data":{"a","b"}}},
		{"scalar value", "{\"grant_type\":\"text\",\"data\":\"abc\"}", url.Values{"grant_type":{"text"},"data":{"abc"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var value hint.DiscriminatorGrant
			if err := json.Unmarshal([]byte(tc.json), &value); err != nil {
				t.Fatal(err)
			}
			if tc.name == "nil scalar array" {
				value = hint.NewDiscriminatorGrantTags(nil)
			}
			if tc.name == "nil JSON array items" {
				value = hint.NewDiscriminatorGrantRecords(nil)
			}
			decode := hintserver.DecodeFlatRootRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)
			calls := 0
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				calls++
				raw, err := io.ReadAll(r.Body)
				if err != nil {
					t.Error(err)
					http.Error(w, "read", http.StatusInternalServerError)
					return
				}
				form, err := url.ParseQuery(string(raw))
				if err != nil || !reflect.DeepEqual(form, tc.form) {
					t.Errorf("wire = %#v, want %#v (error %v)", form, tc.form, err)
				}
				r.Body = io.NopCloser(bytes.NewReader(raw))
				decoded, err := decode(r)
				if err != nil {
					t.Error(err)
					http.Error(w, "decode", http.StatusBadRequest)
					return
				}
				assertSelectedArray(t, decoded, tc.name)
				got, err := json.Marshal(decoded)
				if err != nil {
					t.Error(err)
					return
				}
				if string(got) != tc.json {
					t.Errorf("decoded = %s, want %s", got, tc.json)
				}
				w.Header().Set("Content-Type", "application/json")
				if _, err := w.Write(got); err != nil {
					t.Error(err)
				}
			}))
			defer srv.Close()
			u, err := url.Parse(srv.URL)
			if err != nil {
				t.Fatal(err)
			}
			client := hintclient.NewClient(u.Scheme, u.Host, srv.Client(), loomhttp.RequestEncoder, loomhttp.ResponseDecoder, false)
			result, err := client.FlatRoot()(context.Background(), &value)
			if err != nil {
				t.Fatal(err)
			}
			assertSelectedArray(t, result.(*hint.DiscriminatorGrant), tc.name)
			if calls != 1 {
				t.Fatalf("calls = %d, want 1", calls)
			}
		})
	}
}
func assertSelectedArray(t *testing.T, value *hint.DiscriminatorGrant, name string) {
	t.Helper()
	if name == "nil JSON array items" || name == "empty JSON array items" {
		array, ok := value.AsRecords()
		if !ok || array == nil || len(array) != 0 {
			t.Errorf("selected empty JSON array = %#v, selected=%v", array, ok)
		}
		return
	}
	if name != "nil scalar array" && name != "empty scalar array" {
		return
	}
	array, ok := value.AsTags()
	if !ok || array == nil || len(array) != 0 {
		t.Errorf("selected empty array = %#v, selected=%v", array, ok)
	}
}

func TestSelectedEmptyArrayConstraints(t *testing.T) {
	decode := hintserver.DecodeFlatRootRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)
	for _, tc := range []struct {
		form string
		valid bool
	}{
		{"grant_type=nonempty_tags", false},
		{"grant_type=nonempty_records", false},
		{"grant_type=nonempty_records&data=%7B%22code%22%3A%22a%22%7D", true},
		{"grant_type=nonempty_tags&data=a", true},
	} {
		request := httptest.NewRequest(http.MethodPost, "/FlatRoot", strings.NewReader(tc.form))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err := decode(request)
		if tc.valid && err != nil {
			t.Error(err)
		}
		if !tc.valid && (err == nil || !strings.Contains(err.Error(), "length")) {
			t.Errorf("expected length rejection after selected empty materialization, got %v", err)
		}
	}
}
func TestTaggedValueFieldRejections(t *testing.T) {
	decode := hintserver.DecodeFlatRootRequest(loomhttp.NewMuxer(), loomhttp.RequestDecoder)
	for _, raw := range []string{
		"grant_type=nullable_labels",
		"grant_type=labels&data=null",
		"grant_type=nullable_labels&data=%7B",
		"grant_type=unknown&data=%7B%7D",
	} {
		request := httptest.NewRequest(http.MethodPost, "/FlatRoot", strings.NewReader(raw))
		request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		_, err := decode(request)
		if err == nil {
			t.Errorf("unexpected acceptance: %s", raw)
			continue
		}
		if status := loomhttp.NewErrorResponse(context.Background(), err).StatusCode(); status != http.StatusBadRequest {
			t.Errorf("rejection status = %d, want 400: %s", status, raw)
		}
	}
}

`
