package generator

import "strings"

const selectedViewCarrierRuntime = `package viewcarrier

import (
	"context"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	httpclient "example.com/viewcarrier/gen/http/views/client"
	httpserver "example.com/viewcarrier/gen/http/views/server"
	rpcclient "example.com/viewcarrier/gen/jsonrpc/viewsrpc/client"
	rpcserver "example.com/viewcarrier/gen/jsonrpc/viewsrpc/server"
	svc "example.com/viewcarrier/gen/views"
	projected "example.com/viewcarrier/gen/views/views"
	rpcsvc "example.com/viewcarrier/gen/viewsrpc"
	loomhttp "github.com/CaliLuke/loom/http"
	loom "github.com/CaliLuke/loom/pkg"
	"github.com/stretchr/testify/require"
)

type (
	carrierCase struct {
		name string
		method string
		view string
		body string
		wire string
		rejected bool
	}
	carrierPresence interface {
		Present() bool
		IsNull() bool
	}
)

// TestSelectedViewRuntime checks service construction, wire output and client reconstruction.
func TestSelectedViewRuntime(t *testing.T) {
	for _, transport := range []string{"http", "jsonrpc"} {
		for _, item := range carrierCases() {
			t.Run(transport+"/"+item.name, func(t *testing.T) {
				construct, decode := carrierFunctions(transport, item.method)
				value, err := callCarrierConstructor(t, construct, item.body, item.view)
				if item.rejected {
					var fault *loom.ServiceError
					require.ErrorAs(t, err, &fault)
					require.True(t, fault.Fault)
					require.Equal(t, "fault", fault.Name)
				} else {
					require.NoError(t, err)
					require.NotNil(t, value)
				}
				calls := 0
				endpoint := func(context.Context, any) (any, error) {
					calls++
					return callCarrierConstructor(t, construct, item.body, item.view)
				}
				mux := loomhttp.NewMuxer()
				request := httptest.NewRequest(http.MethodGet, "/"+item.method, nil)
				if transport == "http" {
					endpoints := &svc.Endpoints{Show: endpoint, Root: endpoint, Children: endpoint, Parents: endpoint}
					httpserver.Mount(mux, httpserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil, nil))
				} else {
					endpoints := &rpcsvc.Endpoints{Show: endpoint, Root: endpoint, Children: endpoint, Parents: endpoint}
					rpcserver.Mount(mux, rpcserver.New(endpoints, mux, loomhttp.RequestDecoder, loomhttp.ResponseEncoder, nil))
					request = httptest.NewRequest(http.MethodPost, "/rpc", strings.NewReader("{\"jsonrpc\":\"2.0\",\"id\":1,\"method\":\""+item.method+"\"}"))
					request.Header.Set("Content-Type", "application/json")
				}
				response := httptest.NewRecorder()
				mux.ServeHTTP(response, request)
				require.Equal(t, 1, calls)
				if item.rejected {
					if transport == "http" {
						require.Equal(t, http.StatusInternalServerError, response.Code)
						var problem loomhttp.ProblemResponse
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &problem))
						require.Equal(t, "fault", problem.Code)
					} else {
						require.Equal(t, http.StatusOK, response.Code)
						var envelope struct {
							Error struct {
								Code int ` + "`json:\"code\"`" + `
							} ` + "`json:\"error\"`" + `
						}
						require.NoError(t, json.Unmarshal(response.Body.Bytes(), &envelope))
						require.Equal(t, -32603, envelope.Error.Code)
					}
					return
				}
				require.Equal(t, http.StatusOK, response.Code)
				wire := response.Body.Bytes()
				if transport == "jsonrpc" {
					var envelope struct {
						Result jsontext.Value ` + "`json:\"result\"`" + `
					}
					require.NoError(t, json.Unmarshal(wire, &envelope))
					wire = envelope.Result
				}
				require.JSONEq(t, item.wire, string(wire))
				decoded, err := decode(response.Result())
				require.NoError(t, err)
				assertCarrierPresence(t, decoded, item)
				var expected any
				require.NoError(t, json.Unmarshal([]byte(item.wire), &expected))
				assertSelectedConcrete(t, reflect.ValueOf(decoded), expected)
			})
		}
	}
}

// TestIntrinsicHelperBoundary distinguishes reusable helper skipping from required call admission.
func TestIntrinsicHelperBoundary(t *testing.T) {
	if "CARRIER" != "intrinsic" {
		return
	}
	for _, state := range []string{"", "null", "{\"text\":\"ok\"}", "{\"text\":\"\"}"} {
		t.Run(state, func(t *testing.T) {
			fn := reflect.ValueOf(projected.ValidateChildView)
			argument := reflect.New(fn.Type().In(0))
			if state != "" {
				require.NoError(t, json.Unmarshal([]byte(state), argument.Interface()))
			}
			result := fn.CallSlice([]reflect.Value{argument.Elem(), reflect.ValueOf([]string{"custom.path"})})
			if state == "{\"text\":\"\"}" {
				require.False(t, result[0].IsNil())
				require.Contains(t, result[0].Interface().(error).Error(), "custom.path.text")
			} else {
				require.True(t, result[0].IsNil())
			}
		})
	}
}

// TestRootEnvelopeAdmission checks required root presence independently of named helpers.
func TestRootEnvelopeAdmission(t *testing.T) {
 if "CARRIER" != "intrinsic" && "CARRIER" != "root-occurrence" {
  return
 }
 for _, view := range []string{"default", "tiny", "unknown"} {
  for _, state := range []string{"", "null", "{\"text\":\"ok\"}", "{\"text\":\"\"}"} {
   t.Run(view+"/"+state, func(t *testing.T) {
    fn := reflect.ValueOf(projected.ValidateChild)
    envelope := reflect.New(fn.Type().In(0).Elem())
    envelope.Elem().FieldByName("View").SetString(view)
    if state != "" {
     require.NoError(t, json.Unmarshal([]byte(state), envelope.Elem().FieldByName("Projected").Addr().Interface()))
    }
    result := fn.CallSlice([]reflect.Value{envelope, reflect.ValueOf([]string{"root.path"})})
    rejected := state == "" || view == "unknown" || state == "{\"text\":\"\"}" && view == "default"
    require.Equal(t, !rejected, result[0].IsNil())
   })
  }
 }
}

func carrierCases() []carrierCase {
	var cases []carrierCase
	nullable := "CARRIER" == "occurrence" || "CARRIER" == "intrinsic"
	intrinsic := "CARRIER" == "intrinsic"
	for _, view := range []string{"default", "tiny", "nested"} {
		for _, field := range []string{"optional", "required"} {
			for _, state := range []string{"absent", "null", "valid", "invalid"} {
				fields := map[string]any{"label": "parent", "required": map[string]any{"text": "ok", "label": "child"}}
				child := map[string]any{"text": "ok", "label": "child"}
				switch state {
				case "absent":
					delete(fields, field)
				case "null":
					fields[field] = nil
				case "valid":
					fields[field] = child
				case "invalid":
					child["text"] = ""
					fields[field] = child
				}
				body, err := json.Marshal(fields)
				if err != nil {
					panic(err)
				}
				rejected := view != "tiny" && (field == "required" && (state == "absent" || state == "null" && !nullable) || view == "default" && state == "invalid")
				wireFields := map[string]any{}
				if view == "tiny" {
					wireFields["label"] = "parent"
				} else {
					if view == "default" {
						wireFields["label"] = "parent"
					}
					for _, name := range []string{"optional", "required"} {
						value, present := fields[name]
						if !present || value == nil && !nullable {
							continue
						}
						if value != nil && view == "nested" {
							value = map[string]any{"label": "child"}
						}
						wireFields[name] = value
					}
				}
				wire, err := json.Marshal(wireFields)
				if err != nil {
					panic(err)
				}
				name := view+"/"+field+"/"+state
				cases = append(cases, carrierCase{name: "show/"+name, method: "show", view: view, body: string(body), wire: string(wire), rejected: rejected})
				cases = append(cases, carrierCase{name: "parents/"+name, method: "parents", view: view, body: "["+string(body)+"]", wire: "["+string(wire)+"]", rejected: rejected})
			}
		}
	}
	for _, view := range []string{"default", "tiny", "unknown"} {
		for _, state := range []string{"absent", "null", "valid", "invalid"} {
			body := "{\"text\":\"ok\",\"label\":\"child\"}"
			switch state {
			case "absent":
				body = ""
			case "null":
				body = "null"
			case "invalid":
				body = "{\"text\":\"\",\"label\":\"child\"}"
			}
			wire := body
			if view == "tiny" && state != "null" {
				wire = "{\"label\":\"child\"}"
			}
			rejected := view == "unknown" || state == "absent" || state == "null" && !intrinsic || state == "invalid" && view == "default"
			cases = append(cases, carrierCase{name: "root/"+view+"/"+state, method: "root", view: view, body: body, wire: wire, rejected: view == "unknown" || state == "absent" || state == "null" && !(intrinsic || "CARRIER" == "root-occurrence") || state == "invalid" && view == "default"})
			collectionBody := "["+body+"]"
			if state == "absent" {
				collectionBody = "absent-element"
			}
			cases = append(cases, carrierCase{name: "children/"+view+"/"+state, method: "children", view: view, body: collectionBody, wire: "["+wire+"]", rejected: rejected})
		}
	}
	for _, method := range []string{"children", "parents"} {
		for _, view := range []string{"default", "tiny"} {
			cases = append(cases, carrierCase{name: method+"/"+view+"/empty", method: method, view: view, body: "[]", wire: "[]"})
		}
	}
 if "CARRIER" == "root-occurrence" {
  for _, view := range []string{"default", "tiny", "unknown"} {
   cases = append(cases,
    carrierCase{name: "children/"+view+"/root-absent", method: "children", view: view, body: "", rejected: true},
    carrierCase{name: "children/"+view+"/root-null", method: "children", view: view, body: "null", wire: "null", rejected: view == "unknown"},
   )
  }
 }
 return cases
}

func carrierFunctions(transport, method string) (any, func(*http.Response) (any, error)) {
	if transport == "http" {
		switch method {
		case "show":
			return svc.NewViewedParent, httpclient.DecodeShowResponse(loomhttp.ResponseDecoder, false)
		case "root":
			return svc.NewViewedChild, httpclient.DecodeRootResponse(loomhttp.ResponseDecoder, false)
		case "children":
			return svc.NewViewedChildCollection, httpclient.DecodeChildrenResponse(loomhttp.ResponseDecoder, false)
		default:
			return svc.NewViewedParentCollection, httpclient.DecodeParentsResponse(loomhttp.ResponseDecoder, false)
		}
	}
	switch method {
	case "show":
		return rpcsvc.NewViewedParent, rpcclient.DecodeShowResponse(loomhttp.ResponseDecoder, false)
	case "root":
		return rpcsvc.NewViewedChild, rpcclient.DecodeRootResponse(loomhttp.ResponseDecoder, false)
	case "children":
		return rpcsvc.NewViewedChildCollection, rpcclient.DecodeChildrenResponse(loomhttp.ResponseDecoder, false)
	default:
		return rpcsvc.NewViewedParentCollection, rpcclient.DecodeParentsResponse(loomhttp.ResponseDecoder, false)
	}
}

func callCarrierConstructor(t *testing.T, constructor any, body, view string) (any, error) {
	t.Helper()
	fn := reflect.ValueOf(constructor)
	argument := reflect.New(fn.Type().In(0))
	if body == "absent-element" {
		if value := argument.Elem().MethodByName("Value"); value.IsValid() {
   concrete := reflect.MakeSlice(value.Type().Out(0), 1, 1)
   argument.MethodByName("SetValue").Call([]reflect.Value{concrete})
  } else {
   argument.Elem().Set(reflect.MakeSlice(fn.Type().In(0), 1, 1))
  }
	} else if body != "" {
		require.NoError(t, json.Unmarshal([]byte(body), argument.Interface()))
	}
	result := fn.Call([]reflect.Value{argument.Elem(), reflect.ValueOf(view)})
	if !result[1].IsNil() {
		return nil, result[1].Interface().(error)
	}
	return result[0].Interface(), nil
}

func assertSelectedConcrete(t *testing.T, actual reflect.Value, expected any) {
	t.Helper()
	if state, ok := actual.Interface().(carrierPresence); ok {
		require.True(t, state.Present())
		if expected == nil {
			require.True(t, state.IsNull())
			return
		}
		require.False(t, state.IsNull())
		value := actual.MethodByName("Value").Call(nil)
		require.True(t, value[1].Bool())
		actual = value[0]
	}
	for actual.Kind() == reflect.Pointer {
		require.False(t, actual.IsNil())
		actual = actual.Elem()
	}
	switch value := expected.(type) {
	case map[string]any:
		for name, field := range value {
			goName := strings.ToUpper(name[:1])+name[1:]
			assertSelectedConcrete(t, actual.FieldByName(goName), field)
		}
	case []any:
		require.Equal(t, len(value), actual.Len())
		for index, element := range value {
			assertSelectedConcrete(t, actual.Index(index), element)
		}
	default:
		require.Equal(t, expected, actual.Interface())
	}
}

func assertCarrierPresence(t *testing.T, decoded any, item carrierCase) {
	t.Helper()
	if item.method == "root" {
		if "CARRIER" == "intrinsic" || "CARRIER" == "root-occurrence" {
			state, ok := decoded.(carrierPresence)
			require.True(t, ok, "%T", decoded)
			require.True(t, state.Present())
			require.Equal(t, item.body == "null", state.IsNull())
		}
		return
	}
	if item.method == "children" {
		values := reflect.ValueOf(decoded)
  if "CARRIER" == "root-occurrence" {
   state, ok := decoded.(carrierPresence)
   require.True(t, ok, "%T", decoded)
   require.True(t, state.Present())
   require.Equal(t, item.body == "null", state.IsNull())
   if state.IsNull() {
    return
   }
   values = values.MethodByName("Value").Call(nil)[0]
  }
  if values.Len() > 0 && "CARRIER" == "intrinsic" {
			state := values.Index(0).Interface().(carrierPresence)
			require.True(t, state.Present())
			require.Equal(t, item.body == "[null]", state.IsNull())
		}
		return
	}
	value := reflect.ValueOf(decoded)
	if item.method == "parents" {
		if value.Len() == 0 {
			return
		}
		value = value.Index(0)
	}
	if "CARRIER" == "ordinary" || "CARRIER" == "root-occurrence" {
		return
	}
	var wire map[string]any
	body := item.wire
	if item.method == "parents" {
		body = strings.TrimSuffix(strings.TrimPrefix(body, "["), "]")
	}
	require.NoError(t, json.Unmarshal([]byte(body), &wire))
	for _, field := range []struct {
		goName string
		wireName string
	}{{"Optional", "optional"}, {"Required", "required"}} {
		state := value.Elem().FieldByName(field.goName).Interface().(carrierPresence)
		wireValue, present := wire[field.wireName]
		require.Equal(t, present, state.Present(), field.goName)
		require.Equal(t, present && wireValue == nil, state.IsNull(), field.goName)
	}
}
`

func selectedViewCarrierHarness(carrier string) string {
	harness := strings.ReplaceAll(selectedViewCarrierRuntime, "CARRIER", carrier)
	if carrier == "root-occurrence" {
		harness = strings.ReplaceAll(harness, "NewViewedChild,", "NewViewedChildNullable,")
		harness = strings.ReplaceAll(harness, "projected.ValidateChild)", "projected.ValidateChildNullable)")
		harness = strings.ReplaceAll(harness, "NewViewedChildCollection,", "NewViewedChildCollectionNullable,")
	}
	return harness
}
