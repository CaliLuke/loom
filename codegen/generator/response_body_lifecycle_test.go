package generator

import (
	"testing"

	d "github.com/CaliLuke/loom/dsl"
)

func TestResponseBodyLifecycle(t *testing.T) {
	runDesignHarness(t, "example.com/bodylifecycle", responseBodyLifecycleDSL, responseBodyLifecycleHarness)
}

func responseBodyLifecycleDSL() {
	d.Service("plain", func() {
		d.Method("show", func() {
			d.Result(d.String)
			d.HTTP(func() {
				d.GET("/show")
			})
		})
		for _, name := range []string{"raw", "file"} {
			d.Method(name, func() {
				d.HTTP(func() {
					d.GET("/" + name)
					if name == "raw" {
						d.SkipResponseBodyEncodeDecode()
					} else {
						d.FileResponse()
					}
				})
			})
		}
	})
	d.Service("rpc", func() {
		d.JSONRPC(func() {
			d.POST("/rpc")
		})
		d.Method("show", func() {
			d.Payload(func() {
				d.ID("id")
			})
			d.Result(d.String)
			d.JSONRPC(func() {
			})
		})
	})
}

const responseBodyLifecycleHarness = `package bodylifecycle

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	hc "example.com/bodylifecycle/gen/http/plain/client"
	jc "example.com/bodylifecycle/gen/jsonrpc/rpc/client"
	lh "github.com/CaliLuke/loom/http"
	"github.com/stretchr/testify/require"
)

type trackedBody struct {
	io.Reader
	closes   int
	closeErr error
}

func (b *trackedBody) Close() error {
	b.closes++
	return b.closeErr
}

type brokenReader struct{ err error }

func (r brokenReader) Read([]byte) (int, error) {
	return 0, r.err
}

type doer struct{ resp *http.Response }

func (d doer) Do(*http.Request) (*http.Response, error) {
	return d.resp, nil
}

func TestConsumedBody(t *testing.T) {
	readErr, decodeErr, closeErr := errors.New("read failed"), errors.New("decode failed"), errors.New("close failed")
	for _, restore := range []bool{false, true} {
		for _, rpc := range []bool{false, true} {
			for _, mode := range []string{"ok", "read", "decode", "close", "read-close", "decode-close", "unexpected", "unexpected-read-close"} {
				t.Run(mode, func(t *testing.T) {
					data := "\"ok\""
					if rpc {
						data = "{\"jsonrpc\":\"2.0\",\"result\":\"ok\",\"id\":\"1\"}"
					}
					b := &trackedBody{Reader: strings.NewReader(data)}
					if strings.Contains(mode, "read") {
						b.Reader = io.MultiReader(strings.NewReader(data[:1]), brokenReader{readErr})
					}
					if strings.Contains(mode, "close") {
						b.closeErr = closeErr
					}
					resp := &http.Response{StatusCode: 200, Header: http.Header{"Content-Type": {"application/json"}}, Body: b}
					if strings.HasPrefix(mode, "unexpected") {
						resp.StatusCode = 502
					}
					decoder := lh.ResponseDecoder
					if strings.Contains(mode, "decode") {
						decoder = func(*http.Response) lh.Decoder { return lh.EncodingFunc(func(any) error { return decodeErr }) }
					}
					decode := hc.DecodeShowResponse(decoder, restore)
					if rpc {
						decode = jc.DecodeShowResponse(decoder, restore)
					}
					result, err := decode(resp)
					require.Equal(t, 1, b.closes)
					if mode == "ok" {
						require.NoError(t, err)
						require.NotNil(t, result)
					} else {
						require.Error(t, err)
						require.Nil(t, result)
					}
					if strings.Contains(mode, "read") {
						require.ErrorIs(t, err, readErr)
					}
					if strings.Contains(mode, "decode") {
						require.ErrorIs(t, err, decodeErr)
					}
					if strings.Contains(mode, "close") {
						require.ErrorIs(t, err, closeErr)
					}
					if restore {
						body, readBackErr := io.ReadAll(resp.Body)
						require.NoError(t, readBackErr)
						want := data
						if strings.Contains(mode, "read") {
							want = data[:1]
						}
						require.Equal(t, want, string(body))
					}
				})
			}
		}
	}
}
func TestRawBodyOwnership(t *testing.T) {
	for _, restore := range []bool{false, true} {
		for _, status := range []int{200, 206, 304, 502} {
			for _, file := range []bool{false, true} {
				b := &trackedBody{Reader: strings.NewReader("body"), closeErr: errors.New("close failed")}
				resp := &http.Response{StatusCode: status, Header: http.Header{}, Body: b}
				c := hc.NewClient("http", "example.com", doer{resp}, lh.RequestEncoder, lh.ResponseDecoder, restore)
				endpoint := c.Raw()
				if file {
					endpoint = c.File()
				}
				result, err := endpoint(context.Background(), nil)
				if status == 200 || file && (status == 206 || status == 304) {
					require.NoError(t, err)
					require.NotNil(t, result)
					require.Equal(t, 0, b.closes)
					require.Same(t, b, resp.Body)
					require.ErrorIs(t, resp.Body.Close(), b.closeErr)
				} else {
					require.Error(t, err)
					require.ErrorIs(t, err, b.closeErr)
					require.Equal(t, 1, b.closes)
				}
			}
		}
	}
}
`
