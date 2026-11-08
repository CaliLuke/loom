package codegen

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/http/codegen/testdata"
)

func TestInformationalClientHelpersGenerated(t *testing.T) {
	root := RunHTTPDSL(t, informationalClientFixture)
	dir := t.TempDir()
	renderHTTPModule(t, dir, "example.com/selection", root)
	target := filepath.Join(dir, "gen/http/selection/client/informational_test.go")
	require.NoError(t, os.WriteFile(target, []byte(informationalClientHarness), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "gen/http/ctx/client/status_test.go"), []byte(informationalStatusHarness), 0o600))
	runGoCommand(t, dir, "mod", "tidy")
	runGoCommand(t, dir, "test", "-count=1", "./...")
}

const informationalClientHarness = `package client

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptrace"
	"net/textproto"
	"reflect"
	"strings"
	"testing"

	service "example.com/selection/gen/selection"
	loomhttp "github.com/CaliLuke/loom/http"
)

func TestTypedInformationalMetadata(t *testing.T) {
	called := 0
	ctx := WithReadHints(context.Background(), func(_ context.Context, value *service.Hints, decodeErr error) error {
		called++
		if decodeErr != nil {
			t.Errorf("valid decode error=%v", decodeErr)
			return nil
		}
		if value == nil {
			t.Error("missing typed metadata")
			return nil
		}
		if value.Status != 103 || value.Trace != "early" || value.Labels == nil || len(value.Labels) != 0 || value.Metadata["a"] != "one" {
			t.Errorf("metadata=%#v", value)
		}
		if value.Choice == nil {
			t.Error("missing decoded union")
		} else if text, ok := value.Choice.AsString(); !ok || text != "plain" {
			t.Errorf("choice=%#v", value.Choice)
		}
		return nil
	})
	observers := readInformationalObservers(ctx)
	if len(observers) != 4 {
		t.Fatalf("declarations=%d", len(observers))
	}
	headers := http.Header{"X-Trace": {"early"}, "X-Labels": {""}, "X-Metadata": {"a, one"}, "X-Choice": {"plain"}}
	before := headers.Clone()
	if err := observers[0].Observe(ctx, 103, headers); err != nil {
		t.Errorf("observer=%v", err)
	}
	if called != 1 {
		t.Errorf("callbacks=%d", called)
	}
	if !reflect.DeepEqual(before, headers) {
		t.Errorf("headers mutated: %v", headers)
	}
}

func TestInformationalRegistrationOnly(t *testing.T) {
	calls := 0
	original := WithReadHints(context.Background(), func(context.Context, *service.Hints, error) error { calls++; return nil })
	disabled := WithReadHints(original, nil)
	disabled = WithReadProgress(disabled, func(context.Context, *service.Progress, error) error { return nil })
	observers := readInformationalObservers(disabled)
	if len(observers) != 4 {
		t.Fatalf("declarations=%d", len(observers))
	}
	if observers[0].Selector != (loomhttp.StatusSelector{Code: 103}) || observers[0].Observe != nil {
		t.Errorf("nil exact declaration=%#v", observers[0])
	}
	if observers[1].Selector != (loomhttp.StatusSelector{Class: 1}) {
		t.Errorf("class declaration=%#v", observers[1])
	}
	if observers[1].Observe == nil || observers[2].Observe != nil {
		t.Errorf("independent registrations=%#v", observers)
	}
	if calls != 0 {
		t.Error("option registration invoked callback")
	}
	if readInformationalObservers(original)[0].Observe == nil {
		t.Error("derived context changed original registration")
	}
	request, err := http.NewRequest(http.MethodGet, "http://example.test/read", nil)
	if err != nil {
		t.Fatal(err)
	}
	observed, finish := loomhttp.ObserveInformational(request, readInformationalObservers(context.Background()))
	if observed != request {
		t.Error("no-option helper installed runtime tracing")
	}
	response := &http.Response{StatusCode: 200, Body: http.NoBody}
	actual, err := finish(response, nil)
	if err != nil || actual != response {
		t.Errorf("no-option finish=%v %v", actual, err)
	}
}

func TestInformationalDecodeErrorDelivery(t *testing.T) {
	sentinel := errors.New("observer rejected")
	for _, tc := range []struct {
		name          string
		headers       http.Header
		callbackError error
	}{
		{"missing", http.Header{"X-Labels": {""}}, nil},
		{"invalid mapped", http.Header{"X-Trace": {"abc"}, "X-Labels": {""}}, nil},
		{"ambiguous union", http.Header{"X-Trace": {"early"}, "X-Labels": {""}, "X-Choice": {"text, good"}}, nil},
		{"abort", http.Header{"X-Labels": {""}}, sentinel},
	} {
		t.Run(tc.name, func(t *testing.T) {
			called := 0
			ctx := WithReadHints(context.Background(), func(_ context.Context, value *service.Hints, decodeErr error) error {
				called++
				if decodeErr == nil {
					t.Error("decode error was not delivered")
				}
				if value != nil {
					t.Errorf("invalid decode published metadata: %#v", value)
				}
				return tc.callbackError
			})
			request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/read", nil)
			if err != nil {
				t.Fatal(err)
			}
			request, finish := loomhttp.ObserveInformational(request, readInformationalObservers(ctx))
			trace := httptrace.ContextClientTrace(request.Context())
			trace.GotFirstResponseByte()
			hookErr := trace.Got1xxResponse(103, textproto.MIMEHeader(tc.headers))
			if !errors.Is(hookErr, tc.callbackError) {
				t.Errorf("hook=%v callback=%v", hookErr, tc.callbackError)
			}
			response, finishErr := finish(&http.Response{StatusCode: 200, Body: http.NoBody}, nil)
			if tc.callbackError == nil {
				if finishErr != nil || response == nil {
					t.Errorf("ignored decode error aborted: %v", finishErr)
				}
				if response != nil {
					if err := response.Body.Close(); err != nil {
						t.Fatal(err)
					}
				}
			} else if !errors.Is(finishErr, tc.callbackError) {
				t.Errorf("abort identity=%v", finishErr)
			}
			if called != 1 {
				t.Errorf("callbacks=%d", called)
			}
		})
	}
}

func TestExactBeforeClassObserverSelection(t *testing.T) {
	var statuses []int
	ctx := WithReadHints(context.Background(), nil)
	ctx = WithReadProgress(ctx, func(_ context.Context, value *service.Progress, decodeErr error) error {
		if decodeErr != nil {
			t.Errorf("automatic100 or104 rejected: %v", decodeErr)
			return nil
		}
		if value == nil {
			t.Error("missing class metadata")
			return nil
		}
		statuses = append(statuses, value.Status)
		return nil
	})
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://example.test/read", nil)
	if err != nil {
		t.Fatal(err)
	}
	request, finish := loomhttp.ObserveInformational(request, readInformationalObservers(ctx))
	trace := httptrace.ContextClientTrace(request.Context())
	trace.GotFirstResponseByte()
	for _, status := range []int{103, 100, 104} {
		if err := trace.Got1xxResponse(status, textproto.MIMEHeader{"X-Trace": {"valid"}}); err != nil {
			t.Errorf("status=%d error=%v", status, err)
		}
	}
	response, err := finish(&http.Response{StatusCode: 200, Body: http.NoBody}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := response.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(statuses, []int{104}) {
		t.Errorf("exact nil shadow lost: %v", statuses)
	}
}

func TestInformationalHeaderlessMetadata(t *testing.T) {
	called := false
	ctx := WithReadMarker(context.Background(), func(_ context.Context, value *service.Marker, err error) error {
		called = true
		if value == nil || err != nil {
			t.Errorf("headerless metadata=%v %v", value, err)
		}
		return nil
	})
	if err := readInformationalObservers(ctx)[2].Observe(ctx, 102, http.Header{}); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Error("headerless callback missing")
	}
}

func TestDeclaredContinueDecodeError(t *testing.T) {
	for _, abort := range []bool{false, true} {
		sentinel := errors.New("continue rejected")
		calls := 0
		ctx := WithReadContinue(context.Background(), func(_ context.Context, value *service.Continued, decodeErr error) error {
			calls++
			if decodeErr == nil || value != nil {
				t.Errorf("invalid header result=%#v error=%v", value, decodeErr)
			}
			if abort {
				return sentinel
			}
			return nil
		})
		observers := readInformationalObservers(ctx)
		err := observers[3].Observe(ctx, 100, http.Header{"X-Trace": {"bad"}})
		if (err != nil) != abort || abort && !errors.Is(err, sentinel) || calls != 1 {
			t.Errorf("decode error contract: calls=%d error=%v", calls, err)
		}
	}
}

type observedDoer func(*http.Request) (*http.Response, error)

func (d observedDoer) Do(req *http.Request) (*http.Response, error) {
	return d(req)
}

type observedBody struct {
	reader                *strings.Reader
	ctx                   context.Context
	reads                 int
	closes                int
	firstReadContextError error
}

func (b *observedBody) Read(p []byte) (int, error) {
	if b.reads == 0 {
		b.firstReadContextError = b.ctx.Err()
	}
	b.reads++
	return b.reader.Read(p)
}

func (b *observedBody) Close() error {
	b.closes++
	return nil
}

func TestGeneratedInformationalDoBoundary(t *testing.T) {
	callbackFailure := errors.New("callback rejected")
	doFailure := errors.New("do failed")
	for _, tc := range []struct {
		name            string
		registered      bool
		invalidMetadata bool
		callbackError   error
		doError         error
	}{
		{name: "accepted", registered: true},
		{name: "decode error delivered and ignored", registered: true, invalidMetadata: true},
		{name: "callback abort", registered: true, callbackError: callbackFailure},
		{name: "callback and do failure", registered: true, callbackError: callbackFailure, doError: doFailure},
		{name: "no observer"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls, decodes := 0, 0
			ctx := context.Background()
			if tc.registered {
				ctx = WithReadHints(ctx, func(_ context.Context, value *service.Hints, decodeErr error) error {
					calls++
					if (decodeErr != nil) != tc.invalidMetadata {
						t.Errorf("metadata error=%v invalid=%v", decodeErr, tc.invalidMetadata)
					}
					if !tc.invalidMetadata && (value == nil || value.Status != 103) {
						t.Errorf("metadata=%#v", value)
					}
					return tc.callbackError
				})
			}
			body := &observedBody{reader: strings.NewReader("\"final\"")}
			doer := observedDoer(func(req *http.Request) (*http.Response, error) {
				body.ctx = req.Context()
				trace := httptrace.ContextClientTrace(req.Context())
				if tc.registered {
					if trace == nil || trace.Got1xxResponse == nil {
						t.Error("generated Do request lacks informational observation")
					} else {
						trace.GotFirstResponseByte()
						headers := textproto.MIMEHeader{"X-Trace": {"early"}, "X-Labels": {""}}
						if tc.invalidMetadata {
							delete(headers, "X-Trace")
						}
						if err := trace.Got1xxResponse(103, headers); !errors.Is(err, tc.callbackError) {
							t.Errorf("callback result=%v expected=%v", err, tc.callbackError)
						}
					}
				} else if trace != nil || req.Context() != ctx {
					t.Error("unregistered request acquired tracing or a replacement context")
				}
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": {"application/json"}, "X-Trace": {"final"}},
					Body:       body,
				}, tc.doError
			})
			decoder := func(resp *http.Response) loomhttp.Decoder {
				decodes++
				if tc.registered && calls != 1 {
					t.Errorf("final decoder ran before callback: calls=%d", calls)
				}
				return loomhttp.ResponseDecoder(resp)
			}
			client := NewClient("http", "example.test", doer, loomhttp.RequestEncoder, decoder, false)
			result, err := client.Read()(ctx, nil)
			if tc.callbackError != nil || tc.doError != nil {
				if result != nil || !errors.Is(err, tc.callbackError) || !errors.Is(err, tc.doError) && tc.doError != nil {
					t.Errorf("result=%v error=%v", result, err)
				}
				if decodes != 0 || body.reads != 0 {
					t.Errorf("failed observation decoded final body: decodes=%d reads=%d", decodes, body.reads)
				}
			} else {
				value, ok := result.(*service.SelectionResult)
				if err != nil || !ok || value.Message != "final" || value.Trace != "final" {
					t.Errorf("final result=%#v error=%v", result, err)
				}
				if decodes != 1 || body.reads == 0 || body.firstReadContextError != nil {
					t.Errorf("body lifetime: decodes=%d reads=%d canceled=%v", decodes, body.reads, body.firstReadContextError)
				}
			}
			if body.closes != 1 {
				t.Errorf("body closes=%d", body.closes)
			}
			if tc.registered && (calls != 1 || body.ctx.Err() == nil) {
				t.Errorf("observation cleanup: calls=%d context=%v", calls, body.ctx.Err())
			}
		})
	}
}
`

func informationalClientFixture() {
	testdata.InformationalServerDSL()
	observed := Type("Observed", func() {
		Attribute("status", Int, func() {
			Enum(100, 103)
		})
		Required("status")
	})
	classObserved := Type("ClassObserved", func() {
		Attribute("status", Int, func() {
			Enum(100, 104)
		})
		Required("status")
	})
	Service("ctx", func() {
		Method("reverse", func() {
			HTTP(func() {
				GET("/reverse")
				Informational("class", Status1XX, classObserved, func() {
					StatusField("status")
				})
				Informational("exact", StatusEarlyHints, observed, func() {
					StatusField("status")
				})
				Response(StatusNoContent)
			})
		})
		Method("read", func() {
			HTTP(func() {
				GET("/observed")
				Informational("status", StatusEarlyHints, observed, func() {
					StatusField("status")
				})
				Response(StatusNoContent)
			})
		})
	})
}

const informationalStatusHarness = `package client

import (
 "context"
 "net/http"
 "net/http/httptrace"
 "net/textproto"
 "reflect"
 "testing"

 service "example.com/selection/gen/ctx"
 loomhttp "github.com/CaliLuke/loom/http"
)

func TestObservedNativeStatusConstraint(t *testing.T) {
 for _,tc:=range []struct{status int; valid bool}{{100,true},{103,true},{104,false}} {
  value,err:=decodeReadStatusInformational(tc.status,http.Header{})
  if (err==nil)!=tc.valid {t.Errorf("status=%d value=%#v error=%v",tc.status,value,err)}
  if tc.valid && (value==nil || value.Status!=tc.status) {t.Errorf("actual observed status not retained: %#v",value)}
 }
}
func TestClassBeforeExactObserverSelection(t *testing.T) {
 var statuses []int
 unrelated:=0
 ctx:=WithReadStatus(context.Background(),func(context.Context,*service.Observed,error)error{unrelated++;return nil})
 ctx=WithReverseClass(ctx,func(_ context.Context,value *service.ClassObserved,decodeErr error)error{
  if decodeErr!=nil {t.Errorf("class metadata=%v",decodeErr);return nil}
  if value==nil {t.Error("missing class metadata");return nil}
  statuses=append(statuses,value.Status)
  return nil
 })
 observers:=reverseInformationalObservers(ctx)
 if len(observers)!=2 || observers[0].Selector!=(loomhttp.StatusSelector{Class:1}) || observers[1].Selector!=(loomhttp.StatusSelector{Code:103}) || observers[1].Observe!=nil {t.Fatalf("declarations=%#v",observers)}
 request,err:=http.NewRequestWithContext(ctx,http.MethodGet,"http://example.test/reverse",nil)
 if err!=nil {t.Fatal(err)}
 request,finish:=loomhttp.ObserveInformational(request,observers)
 trace:=httptrace.ContextClientTrace(request.Context())
 trace.GotFirstResponseByte()
 for _,status:=range []int{103,100,104} {
  if err:=trace.Got1xxResponse(status,textproto.MIMEHeader{});err!=nil {t.Errorf("status=%d error=%v",status,err)}
 }
 response,err:=finish(&http.Response{StatusCode:200,Body:http.NoBody},nil)
 if err!=nil {t.Fatal(err)}
 if err:=response.Body.Close();err!=nil {t.Fatal(err)}
 if unrelated!=0 || !reflect.DeepEqual(statuses,[]int{100,104}) {t.Errorf("occurrence identity/shadowing: unrelated=%d statuses=%v",unrelated,statuses)}
}

`
