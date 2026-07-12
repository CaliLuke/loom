package dsl_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	. "github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStreamingDSL(t *testing.T) {
	root := expr.RunDSL(t, func() {
		message := Type("Message", func() {
			Attribute("text", String)
			Required("text")
		})

		Service("streams", func() {
			Method("listen", func() {
				StreamingResult(message)
			})
			Method("upload", func() {
				StreamingPayload(func() {
					Description("Upload chunks")
					Attribute("chunk", Bytes)
					Required("chunk")
				})
				Result(func() {
					Attribute("count", Int)
				})
			})
			Method("chat", func() {
				StreamingPayload(message)
				StreamingResult(message)
			})
			Method("described", func() {
				StreamingPayload(String, "lines to convert", func() {
					MinLength(1)
				})
			})
		})
	})

	svc := root.Service("streams")
	require.NotNil(t, svc)

	listen := svc.Method("listen")
	require.NotNil(t, listen)
	require.Equal(t, expr.ServerStreamKind, listen.Stream)
	require.NotNil(t, listen.StreamingResult)
	require.Equal(t, "Message", listen.StreamingResult.Type.Name())
	// Finalize copies StreamingResult to Result when Result is unset.
	require.Same(t, listen.StreamingResult, listen.Result)

	upload := svc.Method("upload")
	require.NotNil(t, upload)
	require.Equal(t, expr.ClientStreamKind, upload.Stream)
	require.NotNil(t, upload.StreamingPayload)
	require.Equal(t, "Upload chunks", upload.StreamingPayload.Description)
	streamObj := expr.AsObject(upload.StreamingPayload.Type)
	require.NotNil(t, streamObj)
	require.NotNil(t, streamObj.Attribute("chunk"))
	resultObj := expr.AsObject(upload.Result.Type)
	require.NotNil(t, resultObj)
	require.NotNil(t, resultObj.Attribute("count"))

	chat := svc.Method("chat")
	require.NotNil(t, chat)
	require.Equal(t, expr.BidirectionalStreamKind, chat.Stream)
	require.Equal(t, "Message", chat.StreamingPayload.Type.Name())
	require.Equal(t, "Message", chat.StreamingResult.Type.Name())

	described := svc.Method("described")
	require.NotNil(t, described)
	require.Equal(t, expr.ClientStreamKind, described.Stream)
	require.Equal(t, "lines to convert", described.StreamingPayload.Description)
	require.Equal(t, expr.String, described.StreamingPayload.Type)
	require.NotNil(t, described.StreamingPayload.Validation)
	require.NotNil(t, described.StreamingPayload.Validation.MinLength)
	require.Equal(t, 1, *described.StreamingPayload.Validation.MinLength)
}

func TestStreamingDSLErrors(t *testing.T) {
	cases := []struct {
		name    string
		dsl     func()
		wantErr string
	}{
		{
			name: "streaming payload at top level",
			dsl: func() {
				StreamingPayload(String)
			},
			wantErr: "invalid use of StreamingPayload",
		},
		{
			name: "streaming result in service",
			dsl: func() {
				Service("svc", func() {
					StreamingResult(String)
				})
			},
			wantErr: "invalid use of StreamingResult",
		},
		{
			name: "invalid streaming payload argument",
			dsl: func() {
				Service("svc", func() {
					Method("upload", func() {
						StreamingPayload(42)
					})
				})
			},
			wantErr: "cannot use 42 (type int) as type type or function",
		},
		{
			name: "too many streaming result arguments",
			dsl: func() {
				Service("svc", func() {
					Method("listen", func() {
						StreamingResult(String, "description", func() {}, func() {})
					})
				})
			},
			wantErr: "too many arguments given to StreamingResult",
		},
		{
			name: "mixed results without server sent events",
			dsl: func() {
				Service("svc", func() {
					Method("mixed", func() {
						Result(func() {
							Attribute("id", String)
						})
						StreamingResult(func() {
							Attribute("tick", String)
						})
						HTTP(func() {
							POST("/mixed")
						})
					})
				})
			},
			wantErr: "Methods with both Result and StreamingResult defined with different types must use ServerSentEvents()",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := expr.RunInvalidDSL(t, tc.dsl)
			require.Error(t, err)
			require.Contains(t, err.Error(), tc.wantErr)
		})
	}
}
