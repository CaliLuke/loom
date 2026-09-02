package codegen

import (
	"fmt"
	"strings"
)

func joinHTTPTemplateSource(source string, partials ...templateSource) string {
	if len(partials) == 0 {
		return source
	}
	defs := make([]string, 0, len(partials)+1)
	for _, partial := range partials {
		defs = append(defs, fmt.Sprintf("{{- define %q }}\n%s\n{{- end }}", "partial_"+partial.name, partial.source))
	}
	defs = append(defs, source)
	return strings.Join(defs, "\n")
}

type templateSource struct {
	name   string
	source string
}

var (
	serverHandlerInitSource = `{{ printf "%s creates a HTTP handler which loads the HTTP request and calls the %q service %q endpoint." .HandlerInit .ServiceName .Method.Name | comment }}
func {{ .HandlerInit }}(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
	{{- if isWebSocketEndpoint . }}
	upgrader loomhttp.Upgrader,
	configurer loomhttp.ConnConfigureFunc,
	{{- end }}
	{{- if or (isWebSocketEndpoint .) (isSSEEndpoint .) }}
	streamWritePolicy ...loomhttp.StreamWritePolicy,
	{{- end }}
) http.Handler {
	{{- if or (isWebSocketEndpoint .) (isSSEEndpoint .) }}
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	{{- end }}
	{{- if (or (mustDecodeRequest .) (not (or .Redirect (isWebSocketEndpoint .) (and (isSSEEndpoint .) (not .HasMixedResults)))) (not .Redirect) .Method.SkipResponseBodyEncodeDecode) }}
	var (
	{{- end }}
		{{- if mustDecodeRequest . }}
		decodeRequest  = {{ .RequestDecoder }}(mux, decoder)
		{{- end }}
		{{- if not (or .Redirect (isWebSocketEndpoint .) (and (isSSEEndpoint .) (not .HasMixedResults))) }}
		encodeResponse = {{ .ResponseEncoder }}(encoder)
		{{- end }}
	{{- if (or (mustDecodeRequest .) (not .Redirect) .Method.SkipResponseBodyEncodeDecode) }}
		encodeError    = {{ if .Errors }}{{ .ErrorEncoder }}{{ else }}loomhttp.ErrorEncoder{{ end }}(encoder, formatter)
		{{- end }}
	{{- if (or (mustDecodeRequest .) (not (or .Redirect (isWebSocketEndpoint .) (and (isSSEEndpoint .) (not .HasMixedResults)))) (not .Redirect) .Method.SkipResponseBodyEncodeDecode) }}
	)
	{{- end }}
	{{- if isPlainUnaryEndpoint . }}
	return loomhttp.NewUnaryHandler(loomhttp.UnaryHandlerSpec[{{ if .Payload.Ref }}{{ .Payload.Ref }}{{ else }}struct{}{{ end }}, {{ if or .Result.Ref .Result.IsAny }}loomhttp.UnaryResult[{{ if .Result.IsAny }}any{{ else }}{{ .Result.Ref }}{{ end }}]{{ else }}struct{}{{ end }}]{
		Service: {{ printf "%q" .ServiceName }},
		Method:  {{ printf "%q" .Method.Name }},
		{{- if .Payload.Ref }}
		Decode: decodeRequest,
		{{- end }}
		Invoke: func(ctx context.Context, {{ if .Payload.Ref }}payload{{ else }}_{{ end }} {{ if .Payload.Ref }}{{ .Payload.Ref }}{{ else }}struct{}{{ end }}) ({{ if or .Result.Ref .Result.IsAny }}loomhttp.UnaryResult[{{ if .Result.IsAny }}any{{ else }}{{ .Result.Ref }}{{ end }}]{{ else }}struct{}{{ end }}, error) {
			{{ if or .Result.Ref .Result.IsAny }}res{{ else }}_{{ end }}, err := endpoint(ctx, {{ if .Payload.Ref }}payload{{ else }}nil{{ end }})
			if err != nil {
				var zero {{ if or .Result.Ref .Result.IsAny }}loomhttp.UnaryResult[{{ if .Result.IsAny }}any{{ else }}{{ .Result.Ref }}{{ end }}]{{ else }}struct{}{{ end }}
				return zero, err
			}
			{{- if or .Result.Ref .Result.IsAny }}
			return loomhttp.UnaryResult[{{ if .Result.IsAny }}any{{ else }}{{ .Result.Ref }}{{ end }}]{Value: res}, nil
			{{- else }}
			return struct{}{}, nil
			{{- end }}
		},
		EncodeResponse: func(ctx context.Context, w http.ResponseWriter, {{ if or .Result.Ref .Result.IsAny }}result{{ else }}_{{ end }} {{ if or .Result.Ref .Result.IsAny }}loomhttp.UnaryResult[{{ if .Result.IsAny }}any{{ else }}{{ .Result.Ref }}{{ end }}]{{ else }}struct{}{{ end }}) error {
			return encodeResponse(ctx, w, {{ if or .Result.Ref .Result.IsAny }}result.Value{{ else }}nil{{ end }})
		},
		EncodeError:   encodeError,
		HandleFailure: errhandler,
	})
	{{- else }}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lifecycle := loomhttp.NewHandlerLifecycle(w, r, {{ printf "%q" .ServiceName }}, {{ printf "%q" .Method.Name }})
		defer lifecycle.End()
		ctx := lifecycle.Context()
		w = lifecycle.Writer()
	{{- if .HasMixedResults }}

		// Content negotiation for mixed results (standard HTTP vs SSE)
		acceptHeader := strings.Join(r.Header.Values("Accept"), ",")
		if strings.Contains(acceptHeader, "text/event-stream") {
			// Handle SSE request
		{{- if mustDecodeRequest . }}
			payload, err := decodeRequest(r)
			if err != nil {
				lifecycle.DecodeFailed(err, encodeError, errhandler)
				return
			}
		{{- else }}
			var err error
		{{- end }}
		{{- if .SSE.RequestIDField }}
			// Set Last-Event-ID header if present
			if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
				ctx = context.WithValue(ctx, loomhttp.LastEventIDKey, lastEventID)
			{{- if .Payload.Ref }}
			{{- if isObject .Payload.Request.PayloadType }}
				{{- if .SSE.RequestIDPointer }}
				payload.{{ .SSE.RequestIDField }} = &lastEventID
				{{- else }}
				payload.{{ .SSE.RequestIDField }} = lastEventID
				{{- end }}
			{{- end }}
			{{- end }}
			}
		{{- end }}
			stream := &{{ .SSE.StructName }}{
				writer: loomhttp.NewSSEStreamWriter(w, r.Context(), loomtransport.TransportHTTP, writePolicy),
			}
			v := &{{ .ServicePkgName }}.{{ .Method.ServerStream.EndpointStruct }}{
				Stream: stream,
			{{- if .Payload.Ref }}
				Payload: payload,
			{{- end }}
			}
			_, err = endpoint(ctx, v)
			if err != nil {
				lifecycle.HandlerFailed(err, stream.started(), encodeError, errhandler)
			}
		} else {
			// Handle standard HTTP request
		{{- if mustDecodeRequest . }}
			payload, err := decodeRequest(r)
			if err != nil {
				lifecycle.DecodeFailed(err, encodeError, errhandler)
				return
			}
		{{- else }}
			var err error
		{{- end }}
		{{- if .Method.SkipRequestBodyEncodeDecode }}
			data := &{{ .ServicePkgName }}.{{ .Method.RequestStruct }}{ {{ if .Payload.Ref }}Payload: payload, {{ end }}Body: r.Body }
			res, err := endpoint(ctx, data)
		{{- else }}
			// Mixed results endpoints always use the generated endpoint input struct.
			// In the standard (non-SSE) mode, Stream discards events and the service
			// must return the synchronous result.
			v := &{{ .ServicePkgName }}.{{ .Method.ServerStream.EndpointStruct }}{
				Stream: &discard{{ .Method.VarName }}ServerStream{},
			{{- if .Payload.Ref }}
				Payload: payload,
			{{- end }}
			}
			res, err := endpoint(ctx, v)
		{{- end }}
			if err != nil {
				lifecycle.HandlerFailed(err, false, encodeError, errhandler)
				return
			}
		{{- if .Method.SkipResponseBodyEncodeDecode }}
			o := res.(*{{ .ServicePkgName }}.{{ .Method.ResponseStruct }})
			lifecycle.WriteRawBody(o.Body, func(ctx context.Context, w http.ResponseWriter) error {
				return encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }})
			}, encodeError, errhandler)
		{{- else }}
			lifecycle.EncodeResponse(func(ctx context.Context, w http.ResponseWriter) error {
				return encodeResponse(ctx, w, res)
			}, encodeError, errhandler)
		{{- end }}
		}
	{{- else }}
		{{- if mustDecodeRequest . }}
			{{ if .Redirect }}_{{ else }}payload{{ end }}, err := decodeRequest(r)
			if err != nil {
				lifecycle.DecodeFailed(err, encodeError, errhandler)
				return
			}
	{{- else if not .Redirect }}
		var err error
	{{- end }}
	{{- if isWebSocketEndpoint . }}
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		v := &{{ .ServicePkgName }}.{{ .Method.ServerStream.EndpointStruct }}{
			Stream: &{{ .ServerWebSocket.VarName }}{
				upgrader: upgrader,
				configurer: configurer,
				cancel: cancel,
				w: w,
				r: r,
				conn: loomhttp.NewWebSocketStream(nil, writePolicy),
			},
		{{- if .Payload.Ref }}
			Payload: payload,
		{{- end }}
		}
		_, err = endpoint(ctx, v)
	{{- else if and (isSSEEndpoint .) (not .HasMixedResults) }}
		{{- if .SSE.RequestIDField }}
		if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
			ctx = context.WithValue(ctx, loomhttp.LastEventIDKey, lastEventID)
			{{- if .Payload.Ref }}
			{{- if isObject .Payload.Request.PayloadType }}
				{{- if .SSE.RequestIDPointer }}
				payload.{{ .SSE.RequestIDField }} = &lastEventID
				{{- else }}
				payload.{{ .SSE.RequestIDField }} = lastEventID
				{{- end }}
			{{- end }}
			{{- end }}
		}
		{{- end }}
		stream := &{{ .SSE.StructName }}{
			writer: loomhttp.NewSSEStreamWriter(w, r.Context(), loomtransport.TransportHTTP, writePolicy),
		}
		v := &{{ .ServicePkgName }}.{{ .Method.ServerStream.EndpointStruct }}{
			Stream: stream,
		{{- if .Payload.Ref }}
			Payload: payload,
		{{- end }}
		}
		_, err = endpoint(ctx, v)
	{{- else if .Method.SkipRequestBodyEncodeDecode }}
		data := &{{ .ServicePkgName }}.{{ .Method.RequestStruct }}{ {{ if .Payload.Ref }}Payload: payload, {{ end }}Body: r.Body }
		res, err := endpoint(ctx, data)
	{{- else if .Redirect }}
		http.Redirect(w, r, "{{ .Redirect.URL }}", {{ .Redirect.StatusCode }})
	{{- else }}
		res, err := endpoint(ctx, {{ if .Payload.Ref }}payload{{ else }}nil{{ end }})
	{{- end }}
		{{- if not .Redirect }}
			if err != nil {
				{{- if isWebSocketEndpoint . }}
			var stream *{{ .ServerWebSocket.VarName }}
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*{{ .ServerWebSocket.VarName }})
			} else {
				stream = v.Stream.(*{{ .ServerWebSocket.VarName }})
			}
				lifecycle.HandlerFailed(err, stream != nil && stream.conn.Conn() != nil, encodeError, errhandler)
				{{- end }}
				{{- if isSSEEndpoint . }}
				lifecycle.HandlerFailed(err, stream.started(), encodeError, errhandler)
				{{- end }}
				{{- if not (or (isWebSocketEndpoint .) (isSSEEndpoint .)) }}
				lifecycle.HandlerFailed(err, false, encodeError, errhandler)
				{{- end }}
				return
			}
		{{- end }}
	{{- if .Method.FileResponse }}
		o := res.(*{{ .ServicePkgName }}.{{ .Method.FileResponseStruct }})
		lifecycle.ServeFile(r, o.File, {{ with (index .Result.Responses 0) }}{{ if .ContentType }}{{ printf "%q" .ContentType }}{{ else }}""{{ end }}{{ else }}""{{ end }}, func(ctx context.Context, w http.ResponseWriter) error {
			return encodeResponse(ctx, w, {{ if .Result.Ref }}o.Result{{ else }}res{{ end }})
		}, encodeError, errhandler)
		return
	{{- else if .Method.SkipResponseBodyEncodeDecode }}
		o := res.(*{{ .ServicePkgName }}.{{ .Method.ResponseStruct }})
		lifecycle.WriteRawBody(o.Body, func(ctx context.Context, w http.ResponseWriter) error {
			return encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }})
		}, encodeError, errhandler)
		return
	{{- end }}
	{{- if not (or .Redirect (isWebSocketEndpoint .) (isSSEEndpoint .) .Method.SkipResponseBodyEncodeDecode .Method.FileResponse) }}
		lifecycle.EncodeResponse(func(ctx context.Context, w http.ResponseWriter) error {
			return encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }})
		}, encodeError, errhandler)
	{{- end }}
	{{- end }}
	})
	{{- end }}
}

{{- if .HasMixedResults }}

// discard{{ .Method.VarName }}ServerStream implements the {{ .SSE.Interface }}
// interface and drops all events. It is used for mixed results endpoints in
// unary (non-SSE) mode so service implementations can use the stream parameter
// without nil checks.
type discard{{ .Method.VarName }}ServerStream struct{}

// {{ .SSE.SendName }} discards the event.
func (s *discard{{ .Method.VarName }}ServerStream) {{ .SSE.SendName }}(v {{ .SSE.EventTypeRef }}) error {
	return nil
}

// {{ .SSE.SendWithContextName }} discards the event.
func (s *discard{{ .Method.VarName }}ServerStream) {{ .SSE.SendWithContextName }}(ctx context.Context, v {{ .SSE.EventTypeRef }}) error {
	return nil
}

// Close is a no-op.
func (s *discard{{ .Method.VarName }}ServerStream) Close() error {
	return nil
}
{{- end }}
`
)
