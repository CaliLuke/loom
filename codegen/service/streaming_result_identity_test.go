package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CaliLuke/loom/dsl"
	"github.com/CaliLuke/loom/expr"
)

func TestStreamingResultIdentity(t *testing.T) {
	for _, tc := range []struct {
		name     string
		field    string
		optional bool
	}{
		{"distinct", "chunk", false},
		{"shared", "value", false},
		{"optional", "value", true},
	} {
		for _, access := range []string{"read", "write"} {
			t.Run(tc.name+"/"+access, func(t *testing.T) {
				root := runDSL(t, func() {
					audit := dsl.Interceptor("audit", func() {
						fields := func() {
							dsl.Attribute(tc.field)
						}
						if access == "write" {
							dsl.WriteStreamingResult(fields)
						} else {
							dsl.ReadStreamingResult(fields)
						}
					})
					summary := dsl.Type("Summary", func() {
						dsl.Attribute("value", dsl.String)
						dsl.Required("value")
					})
					chunk := dsl.Type("Chunk", func() {
						dsl.Attribute(tc.field, dsl.Int)
						if !tc.optional {
							dsl.Required(tc.field)
						}
					})
					dsl.Service("events", func() {
						dsl.Method("watch", func() {
							dsl.Result(summary)
							dsl.StreamingResult(chunk)
							dsl.ClientInterceptor(audit)
						})
					})
				})
				data := NewServicesData(root).Get("events")
				md := data.Method("watch")
				i := data.ClientInterceptors[0]
				attrs := i.ReadStreamingResult
				if access == "write" {
					attrs = i.WriteStreamingResult
				}
				require.Len(t, attrs, 1)
				assert.Equal(t, "int", attrs[0].TypeRef)
				assert.Equal(t, tc.optional, attrs[0].Pointer)
				assert.Equal(t, "*Chunk", i.Methods[0].StreamingResultRef)
				assert.Equal(t, md.ClientStream.RecvTypeRef, i.Methods[0].StreamingResultRef)
				assert.Equal(t, "*Summary", i.Methods[0].ResultRef)
			})
		}
	}
}

func TestStreamingResultAccessValidation(t *testing.T) {
	for _, tc := range []struct {
		name      string
		field     string
		primitive bool
		ordinary  bool
		want      string
	}{
		{name: "unary-only", field: "summary", want: `streaming result attribute "summary": attribute does not exist`},
		{name: "unknown", field: "missing", want: `streaming result attribute "missing": attribute does not exist`},
		{name: "primitive", field: "chunk", primitive: true, want: "result is not an object"},
		{name: "ordinary-access", field: "summary", ordinary: true, want: "method result is streaming"},
	} {
		for _, access := range []string{"read", "write"} {
			t.Run(tc.name+"/"+access, func(t *testing.T) {
				err := expr.RunInvalidDSL(t, func() {
					audit := dsl.Interceptor("audit", func() {
						fields := func() {
							dsl.Attribute(tc.field)
						}
						switch {
						case tc.ordinary && access == "write":
							dsl.WriteResult(fields)
						case tc.ordinary:
							dsl.ReadResult(fields)
						case access == "write":
							dsl.WriteStreamingResult(fields)
						default:
							dsl.ReadStreamingResult(fields)
						}
					})
					dsl.Service("events", func() {
						dsl.Method("watch", func() {
							dsl.Result(func() {
								dsl.Attribute("summary", dsl.String)
							})
							if tc.primitive {
								dsl.StreamingResult(dsl.String)
							} else {
								dsl.StreamingResult(func() {
									dsl.Attribute("chunk", dsl.Int)
								})
							}
							dsl.ClientInterceptor(audit)
						})
					})
				})
				require.ErrorContains(t, err, tc.want)
			})
		}
	}
}
