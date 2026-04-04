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
) http.Handler {
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, {{ printf "%q" .Method.Name }})
		ctx = context.WithValue(ctx, loom.ServiceKey, {{ printf "%q" .ServiceName }})
	{{- if .HasMixedResults }}

		// Content negotiation for mixed results (standard HTTP vs SSE)
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/event-stream") {
			// Handle SSE request
		{{- if mustDecodeRequest . }}
			payload, err := decodeRequest(r)
			if err != nil {
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
		{{- else }}
			var err error
		{{- end }}
		{{- if .SSE.RequestIDField }}
			// Set Last-Event-ID header if present
			if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
				ctx = context.WithValue(ctx, "last-event-id", lastEventID)
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
				w: w,
				r: r,
			}
			v := &{{ .ServicePkgName }}.{{ .Method.ServerStream.EndpointStruct }}{
				Stream: stream,
			{{- if .Payload.Ref }}
				Payload: payload,
			{{- end }}
			}
			_, err = endpoint(ctx, v)
			if err != nil {
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
			}
		} else {
			// Handle standard HTTP request
		{{- if mustDecodeRequest . }}
			payload, err := decodeRequest(r)
			if err != nil {
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
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
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
		{{- if .Method.SkipResponseBodyEncodeDecode }}
			o := res.(*{{ .ServicePkgName }}.{{ .Method.ResponseStruct }})
			defer o.Body.Close()
			if wt, ok := o.Body.(io.WriterTo); ok {
				if err := encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }}); err != nil {
					if errhandler != nil {
						errhandler(ctx, w, err)
					}
					return
				}
				n, err := wt.WriteTo(w)
				if err != nil {
					if n == 0 {
						if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
							errhandler(ctx, w, err)
						}
					} else {
						http.NewResponseController(w).Flush()
						panic(http.ErrAbortHandler) // too late to write an error
					}
				}
				return
			}
			// handle immediate read error like a returned error
			buf := bufio.NewReader(o.Body)
			if _, err := buf.Peek(1); err != nil && err != io.EOF {
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if err := encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }}); err != nil {
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if _, err := io.Copy(w, buf); err != nil {
				http.NewResponseController(w).Flush()
				panic(http.ErrAbortHandler)
			}
		{{- else }}
			if err := encodeResponse(ctx, w, res); err != nil {
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
			}
		{{- end }}
		}
	{{- else }}
		{{- if mustDecodeRequest . }}
			{{ if .Redirect }}_{{ else }}payload{{ end }}, err := decodeRequest(r)
			if err != nil {
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
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
			},
		{{- if .Payload.Ref }}
			Payload: payload,
		{{- end }}
		}
		_, err = endpoint(ctx, v)
	{{- else if and (isSSEEndpoint .) (not .HasMixedResults) }}
		{{- if .SSE.RequestIDField }}
		if lastEventID := r.Header.Get("Last-Event-ID"); lastEventID != "" {
			ctx = context.WithValue(ctx, "last-event-id", lastEventID)
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
			w: w,
			r: r,
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
				if stream != nil && stream.conn != nil {
					// Response writer has been hijacked, do not encode the error
					if errhandler != nil {
						errhandler(ctx, w, err)
					}
					return
				}
				{{- end }}
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
		{{- end }}
	{{- if .Method.SkipResponseBodyEncodeDecode }}
		o := res.(*{{ .ServicePkgName }}.{{ .Method.ResponseStruct }})
		defer o.Body.Close()
		if wt, ok := o.Body.(io.WriterTo); ok {
			{{- if not (or .Redirect (isWebSocketEndpoint .)) }}
			if err := encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }}); err != nil {
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			{{- end }}
			n, err := wt.WriteTo(w)
			if err != nil {
				if n == 0 {
					if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
						errhandler(ctx, w, err)
					}
				} else {
					http.NewResponseController(w).Flush()
					panic(http.ErrAbortHandler) // too late to write an error
				}
			}
			return
		}
		// handle immediate read error like a returned error
		buf := bufio.NewReader(o.Body)
		if _, err := buf.Peek(1); err != nil && err != io.EOF {
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
	{{- end }}
	{{- if not (or .Redirect (isWebSocketEndpoint .) (isSSEEndpoint .)) }}
		if err := encodeResponse(ctx, w, {{ if and .Method.SkipResponseBodyEncodeDecode .Result.Ref }}o.Result{{ else }}res{{ end }}); err != nil {
			if errhandler != nil {
				errhandler(ctx, w, err)
			}
			{{- if .Method.SkipResponseBodyEncodeDecode }}
			return
			{{- end }}
		}
	{{- end }}
	{{- if .Method.SkipResponseBodyEncodeDecode }}
		if _, err := io.Copy(w, buf); err != nil {
			http.NewResponseController(w).Flush()
			panic(http.ErrAbortHandler) // too late to write an error
		}
	{{- end }}
	{{- end }}
	})
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

	multipartRequestDecoderSource = joinHTTPTemplateSource(`{{ printf "%s returns a decoder to decode the multipart request for the %q service %q endpoint." .InitName .ServiceName .MethodName | comment }}
func {{ .InitName }}(mux loomhttp.Muxer, {{ .VarName }} {{ .FuncName }}) func(r *http.Request) loomhttp.Decoder {
	return func(r *http.Request) loomhttp.Decoder {
		return loomhttp.EncodingFunc(func(v any) error {
			mr, merr := r.MultipartReader()
			if merr != nil {
				return merr
			}
			p := v.(*{{ .Payload.Ref }})
			if err := {{ .VarName }}(mr, p); err != nil {
				return err
			}
			{{- template "partial_request_elements" .Payload.Request }}
			{{- if .Payload.Request.MustValidate }}
			if err != nil {
				return err
			}
			{{- end }}
			{{- if .Payload.Request.PayloadInit }}
				{{- range .Payload.Request.PayloadInit.ServerArgs }}
					{{- if .FieldName }}
			(*p).{{ .FieldName }} = {{ if and (not .Pointer) .FieldPointer }}&{{ end }}{{ .VarName }}
					{{- end }}
				{{- end }}
			{{- end }}
			return nil
		})
	}
}
`,
		templateSource{name: "request_elements", source: `{{- if or .PathParams .QueryParams .Headers .Cookies }}
{{- if .ServerBody }}{{/* we want a newline only if there was code before */}}
{{ end }}
		var (
		{{- range .PathParams }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .QueryParams }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .Headers }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- range .Cookies }}
			{{ .VarName }} {{ .TypeRef }}
		{{- end }}
		{{- if and .MustValidate (not .ServerBody) }}
			err error
		{{- end }}
		{{- if .Cookies }}
			c *http.Cookie
		{{- end }}
		{{- if .PathParams }}

			params = mux.Vars(r)
		{{- end }}
		)

{{- range .PathParams }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }} = params["{{ .HTTPName }}"]

	{{- else }}
		{
			{{ .VarName }}Raw := params["{{ .HTTPName }}"]
			{{- template "partial_path_conversion" . }}
		}

	{{- end }}
		{{- if .Validate }}
		{{ .Validate }}
		{{- end }}
{{- end }}

{{- $qpVar := "r.URL.Query()" }}
{{- if gt (len .QueryParams) 1 }}
{{- $qpVar = "qp" }}
qp := r.URL.Query()
{{- end }}
{{- range .QueryParams }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
		{{ .VarName }} = {{$qpVar}}.Get("{{ .HTTPName }}")
		if {{ .VarName }} == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}

	{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }}Raw := {{$qpVar}}.Get("{{ .HTTPName }}")
		if {{ .VarName }}Raw != "" {
			{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
		}
		{{- if .DefaultValue }} else {
			{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
		}
		{{- end }}

	{{- else if .StringSlice }}
		{{ .VarName }} = {{$qpVar}}["{{ .HTTPName }}"]
		{{- if .Required }}
		if {{ .VarName }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = []string{
                {{- range $i, $v := .DefaultValue }}
                    {{- if $i }}{{ print ", " }}{{ end }}
                    {{- printf "%q" $v -}}
                {{- end -}} }
		}
		{{- end }}

	{{- else if .Slice }}
	{
		{{ .VarName }}Raw := {{$qpVar}}["{{ .HTTPName }}"]
		{{- if .Required }}
		if {{ .VarName }}Raw == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else if .Map }}
	{
		{{ .VarName }}Raw := {{$qpVar}}
		{{- if .Required }}
		if len({{ .VarName }}Raw) == 0 {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if len({{ .VarName }}Raw) == 0 {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if len({{ .VarName }}Raw) != 0 {
		{{- end }}
		for keyRaw, valRaw := range {{ .VarName }}Raw {
			if strings.HasPrefix(keyRaw, "{{ .HTTPName }}[") {
				{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type .VarName 0) }}
			}
		}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else if .MapQueryParams }}
	{
		{{ .VarName }}Raw := {{$qpVar}}
		{{- if .Required }}
		if len({{ .VarName }}Raw) == 0 {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if len({{ .VarName }}Raw) == 0 {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if len({{ .VarName }}Raw) != 0 {
		{{- end }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = make({{ goTypeRef .Type }})
		}
		for keyRaw, valRaw := range {{ .VarName }}Raw {
			var key {{ goTypeRef .Type.KeyType.Type }}
			{{- if eq .Type.KeyType.Type.Name "string" }}
			key = keyRaw
			{{- else }}
				{{- template "partial_query_type_conversion" (conversionData "key" "query" .Type.KeyType.Type) }}
			{{- end }}
			{{- if eq .Type.ElemType.Type.Name "string" }}
				{{ .VarName }}[key] = valRaw[0]
			{{- else if eq .Type.ElemType.Type.Name "array" }}
				{{- if eq .Type.ElemType.Type.ElemType.Type.Name "string" }}
					{{ .VarName }}[key] = valRaw
				{{- else }}
					var val {{ goTypeRef .Type.ElemType.Type }}
					{
						{{- template "partial_element_slice_conversion" (conversionData "val" "query" .Type.ElemType.Type) }}
					}
					{{ .VarName }}[key] = val
				{{- end }}
			{{- else if eq .Type.ElemType.Type.Name "map" }}
				{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type.ElemType.Type (printf "%s[key]" .VarName) 1) }}
			{{- else }}
				var val{{ .Loop }} {{ goTypeRef .Type.ElemType.Type }}
				{
					val{{ .Loop }}Raw := valRaw[0]
					{{- template "partial_query_type_conversion" (conversionData (printf "val%s" .Loop) "query" .Type.ElemType.Type) }}
				}
				{{ .VarName }}[key] = val{{ .Loop }}
			{{- end }}
		}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else }}
	{
		{{ .VarName }}Raw := {{$qpVar}}.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "query string"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- end }}
		{{- if .Validate }}
		{{ .Validate }}
		{{- end }}
{{- end }}

{{- range .Headers }}
	{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
		{{ .VarName }} = r.Header.Get("{{ .HTTPName }}")
		if {{ .VarName }} == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}

	{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
		{{ .VarName }}Raw := r.Header.Get("{{ .HTTPName }}")
		if {{ .VarName }}Raw != "" {
			{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
		}
		{{- if .DefaultValue }} else {
			{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
		}
		{{- end }}

	{{- else if .StringSlice }}
		{{ .VarName }} = r.Header["{{ .CanonicalName }}"]
		{{- if .Required }}
		if {{ .VarName }} == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }} == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

	{{- else if .Slice }}
	{
		{{ .VarName }}Raw := r.Header["{{ .CanonicalName }}"]
		{{ if .Required }}if {{ .VarName }}Raw == nil {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == nil {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != nil {
		{{- end }}
		{{- template "partial_element_slice_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}

	{{- else }}
	{
		{{ .VarName }}Raw := r.Header.Get("{{ .HTTPName }}")
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "header"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
		{{- if or .DefaultValue (not .Required) }}
		}
		{{- end }}
	}
	{{- end }}
	{{- if .Validate }}
		{{ .Validate }}
	{{- end }}
{{- end }}

{{- range .Cookies }}
		{
			c, cookieErr := r.Cookie("{{ .HTTPName }}")
			if cookieErr != nil {
				if errors.Is(cookieErr, http.ErrNoCookie) {
					{{- if .Required }}
					err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "cookie"))
					{{- end }}
				} else {
					return payload, cookieErr
				}
			}
		{{- if and (or (eq .Type.Name "string") (eq .Type.Name "any")) .Required }}
			if c != nil {
				{{ .VarName }} = c.Value
			}

		{{- else if (or (eq .Type.Name "string") (eq .Type.Name "any")) }}
			var {{ .VarName }}Raw string
			if c != nil {
				{{ .VarName }}Raw = c.Value
			}
			if {{ .VarName }}Raw != "" {
				{{ .VarName }} = {{ if and (eq .Type.Name "string") .Pointer }}&{{ end }}{{ .VarName }}Raw
			}
			{{- if .DefaultValue }} else {
				{{ .VarName }} = {{ if eq .Type.Name "string" }}{{ printf "%q" .DefaultValue }}{{ else }}{{ printf "%#v" .DefaultValue }}{{ end }}
			}
			{{- end }}

		{{- else }}
			var {{ .VarName }}Raw string
			if c != nil {
				{{ .VarName }}Raw = c.Value
		}
		{{- if .Required }}
		if {{ .VarName }}Raw == "" {
			err = loom.MergeErrors(err, loom.MissingFieldError("{{ .Name }}", "cookie"))
		}
		{{- else if .DefaultValue }}
		if {{ .VarName }}Raw == "" {
			{{ .VarName }} = {{ printf "%#v" .DefaultValue }}
		}
		{{- end }}

		{{- if .DefaultValue }}else {
		{{- else if not .Required }}
		if {{ .VarName }}Raw != "" {
		{{- end }}
		{{- template "partial_query_type_conversion" . }}
			{{- if or .DefaultValue (not .Required) }}
			}
			{{- end }}
		{{- end }}
		}
		{{- if .Validate }}
			{{ .Validate }}
		{{- end }}
{{- end }}
{{- end }}`},
		templateSource{name: "slice_item_conversion", source: `		{{- if eq .Type.ElemType.Type.Name "string" }}
			{{ .VarName }}[i] = rv
		{{- else if eq .Type.ElemType.Type.Name "bytes" }}
			{{ .VarName }}[i] = []byte(rv)
		{{- else if eq .Type.ElemType.Type.Name "int" }}
			v, err2 := strconv.ParseInt(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int(v)
		{{- else if eq .Type.ElemType.Type.Name "int32" }}
			v, err2 := strconv.ParseInt(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = int32(v)
		{{- else if eq .Type.ElemType.Type.Name "int64" }}
			v, err2 := strconv.ParseInt(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "uint" }}
			v, err2 := strconv.ParseUint(rv, 10, strconv.IntSize)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint(v)
		{{- else if eq .Type.ElemType.Type.Name "uint32" }}
			v, err2 := strconv.ParseUint(rv, 10, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = uint32(v)
		{{- else if eq .Type.ElemType.Type.Name "uint64" }}
			v, err2 := strconv.ParseUint(rv, 10, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of unsigned integers"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "float32" }}
			v, err2 := strconv.ParseFloat(rv, 32)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = float32(v)
		{{- else if eq .Type.ElemType.Type.Name "float64" }}
			v, err2 := strconv.ParseFloat(rv, 64)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of floats"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "boolean" }}
			v, err2 := strconv.ParseBool(rv)
			if err2 != nil {
				err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "array of booleans"))
			}
			{{ .VarName }}[i] = v
		{{- else if eq .Type.ElemType.Type.Name "any" }}
			{{ .VarName }}[i] = rv
		{{- else }}
			// unsupported slice type {{ .Type.ElemType.Type.Name }} for var {{ .VarName }}
		{{- end }}`},
		templateSource{name: "element_slice_conversion", source: `	{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}Raw))
	for i, rv := range {{ .VarName }}Raw {
		{{- template "partial_slice_item_conversion" . }}
	}`},
		templateSource{name: "query_type_conversion", source: `	{{- if eq .Type.Name "bytes" }}
		{{ .VarName }} = []byte({{.VarName}}Raw)
	{{- else if eq .Type.Name "int" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{slice .TypeRef 1 (len .TypeRef)}}{{ else }}int{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int32" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}int32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}int32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "int64" }}
		v, err2 := strconv.ParseInt({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "int64") (ne .TypeRef "*int64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "uint" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, strconv.IntSize)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint32" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}uint32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}uint32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "uint64" }}
		v, err2 := strconv.ParseUint({{ .VarName }}Raw, 10, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "unsigned integer"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "uint64") (ne .TypeRef "*uint64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "float32" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 32)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{- if .Pointer }}
		pv := {{ if .TypeRef }}{{ slice .TypeRef 1 (len .TypeRef) }}{{ else }}float32{{ end }}(v)
		{{ .VarName }} = &pv
		{{- else }}
		{{ .VarName }} = {{ if .TypeRef }}{{ .TypeRef }}{{ else }}float32{{ end }}(v)
		{{- end }}
	{{- else if eq .Type.Name "float64" }}
		v, err2 := strconv.ParseFloat({{ .VarName }}Raw, 64)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "float"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "float64") (ne .TypeRef "*float64")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else if eq .Type.Name "boolean" }}
		v, err2 := strconv.ParseBool({{ .VarName }}Raw)
		if err2 != nil {
			err = loom.MergeErrors(err, loom.InvalidFieldTypeError({{ printf "%q" .Name }}, {{ .VarName}}Raw, "boolean"))
		}
		{{ if and (ne .TypeRef nil) (and (ne .TypeRef "bool") (ne .TypeRef "*bool")) }}{{ .VarName }} = ({{.TypeRef}})({{ if .Pointer }}&{{ end }}v){{ else }}{{ .VarName }} = {{ if .Pointer }}&{{ end }}v{{ end }}
	{{- else }}
		// unsupported type {{ .Type.Name }} for var {{ .VarName }}
	{{- end }}`},
		templateSource{name: "query_map_conversion", source: `	if {{ .VarName }} == nil {
		{{ .VarName }} = make({{ goTypeRef .Type }})
	}
	var key{{ .Loop }} {{ goTypeRef .Type.KeyType.Type }}
	{
		openIdx := strings.IndexRune(keyRaw, '[')
		closeIdx := strings.IndexRune(keyRaw, ']')
		if openIdx == -1 || closeIdx == -1 || closeIdx <= openIdx {
			err = loom.MergeErrors(err, loom.DecodePayloadError("invalid query string: malformed brackets"))
		} else {
	{{- if eq .Type.KeyType.Type.Name "string" }}
		key{{ .Loop }} = keyRaw[openIdx+1 : closeIdx]
	{{- else }}
		key{{ .Loop }}Raw := keyRaw[openIdx+1 : closeIdx]
		{{- template "partial_query_type_conversion" (conversionData (printf "key%s" .Loop) "query" .Type.KeyType.Type) }}
	{{- end }}
		{{- if gt .Depth 0 }}
			keyRaw = keyRaw[closeIdx+1:]
		{{- end }}
		}
	}
	{{- if eq .Type.ElemType.Type.Name "string" }}
		{{ .VarName }}[key{{ .Loop }}] = valRaw[0]
	{{- else if eq .Type.ElemType.Type.Name "array" }}
		{{- if eq .Type.ElemType.Type.ElemType.Type.Name "string" }}
			{{ .VarName }}[key{{ .Loop }}] = valRaw
		{{- else }}
			var val {{ goTypeRef .Type.ElemType.Type }}
			{
				{{- template "partial_element_slice_conversion" (conversionData "val" "query" .Type.ElemType.Type) }}
			}
			{{ .VarName }}[key{{ .Loop }}] = val
		{{- end }}
	{{- else if eq .Type.ElemType.Type.Name "map" }}
		{{- template "partial_query_map_conversion" (mapQueryDecodeData .Type.ElemType.Type (printf "%s[key%s]" .VarName .Loop) 1) }}
	{{- else }}
		var val{{ .Loop }} {{ goTypeRef .Type.ElemType.Type }}
		{
			val{{ .Loop }}Raw := valRaw[0]
			{{- template "partial_query_type_conversion" (conversionData (printf "val%s" .Loop) "query" .Type.ElemType.Type) }}
		}
		{{ .VarName }}[key{{ .Loop }}] = val{{ .Loop }}
	{{- end }}`},
		templateSource{name: "path_conversion", source: `	{{- if eq .Type.Name "array" }}
		{{ .VarName }}RawSlice := strings.Split({{ .VarName }}Raw, ",")
		{{ .VarName }} = make({{ goTypeRef .Type }}, len({{ .VarName }}RawSlice))
		for i, rv := range {{ .VarName }}RawSlice {
			{{- template "partial_slice_item_conversion" . }}
		}
	{{- else }}
		{{- template "partial_query_type_conversion" . }}
	{{- end }}`},
	)
)
