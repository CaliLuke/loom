package testdata

var MixedEndpointsConnConfigurerStructCode = `// ConnConfigurer holds the websocket connection configurer functions for the
// streaming endpoints in "StreamingResultService" service.
type ConnConfigurer struct {
	StreamingResultMethodFn loomhttp.ConnConfigureFunc
}
`

var MixedEndpointsConnConfigurerInitCode = `// NewConnConfigurer initializes the websocket connection configurer function
// with fn for all the streaming endpoints in "StreamingResultService" service.
func NewConnConfigurer(fn loomhttp.ConnConfigureFunc) *ConnConfigurer {
	return &ConnConfigurer{
		StreamingResultMethodFn: fn,
	}
}
`

var StreamingResultServerHandlerInitCode = `// NewStreamingResultMethodHandler creates a HTTP handler which loads the HTTP
// request and calls the "StreamingResultService" service
// "StreamingResultMethod" endpoint.
func NewStreamingResultMethodHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
	upgrader loomhttp.Upgrader,
	configurer loomhttp.ConnConfigureFunc,
	streamWritePolicy ...loomhttp.StreamWritePolicy,
) http.Handler {
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	var (
		decodeRequest = DecodeStreamingResultMethodRequest(mux, decoder)
		encodeError   = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "StreamingResultMethod")
		ctx = context.WithValue(ctx, loom.ServiceKey, "StreamingResultService")
		obs, w := loomtransport.BeginHTTPRequest(ctx, w, "StreamingResultService", "StreamingResultMethod", r)
		defer obs.End()
		payload, err := decodeRequest(r)
		if err != nil {
			obs.Fail(loomtransport.ReasonRequestDecodeFailed)
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		v := &streamingresultservice.StreamingResultMethodEndpointInput{
			Stream: &StreamingResultMethodServerStream{
				upgrader:   upgrader,
				configurer: configurer,
				cancel:     cancel,
				w:          w,
				r:          r,
				conn:       loomhttp.NewWebSocketStream(nil, writePolicy),
			},
			Payload: payload,
		}
		_, err = endpoint(ctx, v)
		if err != nil {
			obs.Fail(loomtransport.ReasonHandlerError)
			var stream *StreamingResultMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*StreamingResultMethodServerStream)
			} else {
				stream = v.Stream.(*StreamingResultMethodServerStream)
			}
			if stream != nil && stream.conn.Conn() != nil {
				// Response writer has been hijacked, do not encode the error
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
	})
}
`

var MixedResultsServerHandlerInitCode = `// NewCreateHandler creates a HTTP handler which loads the HTTP request and
// calls the "MixedResultsService" service "Create" endpoint.
func NewCreateHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
	streamWritePolicy ...loomhttp.StreamWritePolicy,
) http.Handler {
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	var (
		decodeRequest  = DecodeCreateRequest(mux, decoder)
		encodeResponse = EncodeCreateResponse(encoder)
		encodeError    = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "Create")
		ctx = context.WithValue(ctx, loom.ServiceKey, "MixedResultsService")
		obs, w := loomtransport.BeginHTTPRequest(ctx, w, "MixedResultsService", "Create", r)
		defer obs.End()

		// Content negotiation for mixed results (standard HTTP vs SSE)
		acceptHeader := r.Header.Get("Accept")
		if strings.Contains(acceptHeader, "text/event-stream") {
			// Handle SSE request
			payload, err := decodeRequest(r)
			if err != nil {
				obs.Fail(loomtransport.ReasonRequestDecodeFailed)
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			stream := &CreateServerStream{
				writer: loomhttp.NewSSEStreamWriter(w, r.Context(), loomtransport.TransportHTTP, writePolicy),
			}
			v := &mixedresultsservice.CreateEndpointInput{
				Stream:  stream,
				Payload: payload,
			}
			_, err = endpoint(ctx, v)
			if err != nil {
				obs.Fail(loomtransport.ReasonHandlerError)
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
			}
		} else {
			// Handle standard HTTP request
			payload, err := decodeRequest(r)
			if err != nil {
				obs.Fail(loomtransport.ReasonRequestDecodeFailed)
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			// Mixed results endpoints always use the generated endpoint input struct.
			// In the standard (non-SSE) mode, Stream discards events and the service
			// must return the synchronous result.
			v := &mixedresultsservice.CreateEndpointInput{
				Stream:  &discardCreateServerStream{},
				Payload: payload,
			}
			res, err := endpoint(ctx, v)
			if err != nil {
				obs.Fail(loomtransport.ReasonHandlerError)
				if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if err := encodeResponse(ctx, w, res); err != nil {
				obs.Fail(loomtransport.ReasonResponseWriteFailed)
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
			}
		}
	})
}

// discardCreateServerStream implements the mixedresultsservice.CreateServerStream
// interface and drops all events. It is used for mixed results endpoints in
// unary (non-SSE) mode so service implementations can use the stream parameter
// without nil checks.
type discardCreateServerStream struct{}

// Send discards the event.
func (s *discardCreateServerStream) Send(v *mixedresultsservice.Event) error {
	return nil
}

// SendWithContext discards the event.
func (s *discardCreateServerStream) SendWithContext(ctx context.Context, v *mixedresultsservice.Event) error {
	return nil
}

// Close is a no-op.
func (s *discardCreateServerStream) Close() error {
	return nil
}
`

var StreamingResultServerStreamSendCode = `// SendWithContext streams instances of "streamingresultservice.UserType" to
// the "StreamingResultMethod" endpoint websocket connection with context.
func (s *StreamingResultMethodServerStream) SendWithContext(ctx context.Context, v *streamingresultservice.UserType) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res := v
		body := NewStreamingResultMethodResponseBody(res)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "streamingresultservice.UserType" to the
// "StreamingResultMethod" endpoint websocket connection.
func (s *StreamingResultMethodServerStream) Send(v *streamingresultservice.UserType) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultServerStreamCloseCode = `// Close closes the "StreamingResultMethod" endpoint websocket connection.
func (s *StreamingResultMethodServerStream) Close() error {
	var err error
	if s.conn == nil {
		return nil
	}
	if err = s.conn.WriteClose("server closing connection"); err != nil {
		return err
	}
	return s.conn.Close()
}
`

var StreamingResultWithViewsServerStreamSendCode = `// SendWithContext streams instances of
// "streamingresultwithviewsservice.Usertype" to the
// "StreamingResultWithViewsMethod" endpoint websocket connection with context.
func (s *StreamingResultWithViewsMethodServerStream) SendWithContext(ctx context.Context, v *streamingresultwithviewsservice.Usertype) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			respHdr := make(http.Header)
			respHdr.Add("loom-view", s.view)
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res, err := streamingresultwithviewsservice.NewViewedUsertype(v, s.view)
		if err != nil {
			return err
		}
		var body any
		switch s.view {
		case "tiny":
			body = NewStreamingResultWithViewsMethodResponseBodyTiny(res.Projected)
		case "extended":
			body = NewStreamingResultWithViewsMethodResponseBodyExtended(res.Projected)
		case "default", "":
			body = NewStreamingResultWithViewsMethodResponseBody(res.Projected)
		}
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "streamingresultwithviewsservice.Usertype" to the
// "StreamingResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultWithViewsMethodServerStream) Send(v *streamingresultwithviewsservice.Usertype) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultWithViewsServerStreamSetViewCode = `// SetView sets the view to render the streamingresultwithviewsservice.Usertype
// type before sending to the "StreamingResultWithViewsMethod" endpoint
// websocket connection.
func (s *StreamingResultWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`

var StreamingResultNoPayloadServerHandlerInitCode = `// NewStreamingResultNoPayloadMethodHandler creates a HTTP handler which loads
// the HTTP request and calls the "StreamingResultNoPayloadService" service
// "StreamingResultNoPayloadMethod" endpoint.
func NewStreamingResultNoPayloadMethodHandler(
	endpoint loom.Endpoint,
	mux loomhttp.Muxer,
	decoder func(*http.Request) loomhttp.Decoder,
	encoder func(context.Context, http.ResponseWriter) loomhttp.Encoder,
	errhandler func(context.Context, http.ResponseWriter, error),
	formatter func(ctx context.Context, err error) loomhttp.Statuser,
	upgrader loomhttp.Upgrader,
	configurer loomhttp.ConnConfigureFunc,
	streamWritePolicy ...loomhttp.StreamWritePolicy,
) http.Handler {
	var writePolicy loomhttp.StreamWritePolicy
	if len(streamWritePolicy) > 0 {
		writePolicy = streamWritePolicy[0]
	}
	var (
		encodeError = loomhttp.ErrorEncoder(encoder, formatter)
	)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := context.WithValue(r.Context(), loomhttp.AcceptTypeKey, r.Header.Get("Accept"))
		ctx = context.WithValue(ctx, loom.MethodKey, "StreamingResultNoPayloadMethod")
		ctx = context.WithValue(ctx, loom.ServiceKey, "StreamingResultNoPayloadService")
		obs, w := loomtransport.BeginHTTPRequest(ctx, w, "StreamingResultNoPayloadService", "StreamingResultNoPayloadMethod", r)
		defer obs.End()
		var err error
		var cancel context.CancelFunc
		ctx, cancel = context.WithCancel(ctx)
		v := &streamingresultnopayloadservice.StreamingResultNoPayloadMethodEndpointInput{
			Stream: &StreamingResultNoPayloadMethodServerStream{
				upgrader:   upgrader,
				configurer: configurer,
				cancel:     cancel,
				w:          w,
				r:          r,
				conn:       loomhttp.NewWebSocketStream(nil, writePolicy),
			},
		}
		_, err = endpoint(ctx, v)
		if err != nil {
			obs.Fail(loomtransport.ReasonHandlerError)
			var stream *StreamingResultNoPayloadMethodServerStream
			if wrapper, ok := v.Stream.(interface{ Unwrap() any }); ok {
				stream = wrapper.Unwrap().(*StreamingResultNoPayloadMethodServerStream)
			} else {
				stream = v.Stream.(*StreamingResultNoPayloadMethodServerStream)
			}
			if stream != nil && stream.conn.Conn() != nil {
				// Response writer has been hijacked, do not encode the error
				if errhandler != nil {
					errhandler(ctx, w, err)
				}
				return
			}
			if err := encodeError(ctx, w, err); err != nil && errhandler != nil {
				errhandler(ctx, w, err)
			}
			return
		}
	})
}
`

var StreamingResultClientEndpointCode = `// StreamingResultMethod returns an endpoint that makes HTTP requests to the
// StreamingResultService service StreamingResultMethod server.
func (c *Client) StreamingResultMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingResultMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingResultMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingResultService", "StreamingResultMethod", err)
		}
		if c.configurer.StreamingResultMethodFn != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			conn = c.configurer.StreamingResultMethodFn(conn, cancel)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if err := wsconn.WriteClose("client closing connection"); err != nil {
					return
				}
				if err := wsconn.Close(); err != nil {
					return
				}
			case <-done:
			}
		}()
		stream := &StreamingResultMethodClientStream{conn: wsconn, done: done}
		return stream, nil
	}
}
`

var StreamingResultWithViewsServerStreamCloseCode = `// Close closes the "StreamingResultWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingResultWithViewsMethodServerStream) Close() error {
	var err error
	if s.conn == nil {
		return nil
	}
	if err = s.conn.WriteClose("server closing connection"); err != nil {
		return err
	}
	return s.conn.Close()
}
`

var StreamingResultClientStreamRecvCode = `// Recv reads instances of "streamingresultservice.UserType" from the
// "StreamingResultMethod" endpoint websocket connection.
func (s *StreamingResultMethodClientStream) Recv() (*streamingresultservice.UserType, error) {
	var (
		rv   *streamingresultservice.UserType
		body StreamingResultMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultMethodUserTypeOK(&body)
	return res, nil
}

// RecvWithContext reads instances of "streamingresultservice.UserType" from
// the "StreamingResultMethod" endpoint websocket connection with context.
func (s *StreamingResultMethodClientStream) RecvWithContext(ctx context.Context) (*streamingresultservice.UserType, error) {
	var (
		rv   *streamingresultservice.UserType
		body StreamingResultMethodResponseBody
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultMethodUserTypeOK(&body)
	return res, nil
}
`

var StreamingResultWithViewsClientEndpointCode = `// StreamingResultWithViewsMethod returns an endpoint that makes HTTP requests
// to the StreamingResultWithViewsService service
// StreamingResultWithViewsMethod server.
func (c *Client) StreamingResultWithViewsMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingResultWithViewsMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingResultWithViewsMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingResultWithViewsService", "StreamingResultWithViewsMethod", err)
		}
		if c.configurer.StreamingResultWithViewsMethodFn != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			conn = c.configurer.StreamingResultWithViewsMethodFn(conn, cancel)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if err := wsconn.WriteClose("client closing connection"); err != nil {
					return
				}
				if err := wsconn.Close(); err != nil {
					return
				}
			case <-done:
			}
		}()
		stream := &StreamingResultWithViewsMethodClientStream{conn: wsconn, done: done}
		view := resp.Header.Get("loom-view")
		stream.SetView(view)
		return stream, nil
	}
}
`

var StreamingResultWithViewsClientStreamRecvCode = `// Recv reads instances of "streamingresultwithviewsservice.Usertype" from the
// "StreamingResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultWithViewsMethodClientStream) Recv() (*streamingresultwithviewsservice.Usertype, error) {
	var (
		rv   *streamingresultwithviewsservice.Usertype
		body StreamingResultWithViewsMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultWithViewsMethodUsertypeOK(&body)
	vres := &streamingresultwithviewsserviceviews.Usertype{res, s.view}
	if err := streamingresultwithviewsserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithViewsService", "StreamingResultWithViewsMethod", err)
	}
	result, err := streamingresultwithviewsservice.NewUsertype(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithViewsService", "StreamingResultWithViewsMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "streamingresultwithviewsservice.Usertype" from the
// "StreamingResultWithViewsMethod" endpoint websocket connection with context.
func (s *StreamingResultWithViewsMethodClientStream) RecvWithContext(ctx context.Context) (*streamingresultwithviewsservice.Usertype, error) {
	var (
		rv   *streamingresultwithviewsservice.Usertype
		body StreamingResultWithViewsMethodResponseBody
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultWithViewsMethodUsertypeOK(&body)
	vres := &streamingresultwithviewsserviceviews.Usertype{res, s.view}
	if err := streamingresultwithviewsserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithViewsService", "StreamingResultWithViewsMethod", err)
	}
	result, err := streamingresultwithviewsservice.NewUsertype(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithViewsService", "StreamingResultWithViewsMethod", err)
	}
	return result, nil
}
`

var StreamingResultWithViewsClientStreamSetViewCode = `// SetView sets the view to render the  type before sending to the
// "StreamingResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`

var StreamingResultWithExplicitViewClientEndpointCode = `// StreamingResultWithExplicitViewMethod returns an endpoint that makes HTTP
// requests to the StreamingResultWithExplicitViewService service
// StreamingResultWithExplicitViewMethod server.
func (c *Client) StreamingResultWithExplicitViewMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingResultWithExplicitViewMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingResultWithExplicitViewMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingResultWithExplicitViewService", "StreamingResultWithExplicitViewMethod", err)
		}
		if c.configurer.StreamingResultWithExplicitViewMethodFn != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			conn = c.configurer.StreamingResultWithExplicitViewMethodFn(conn, cancel)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if err := wsconn.WriteClose("client closing connection"); err != nil {
					return
				}
				if err := wsconn.Close(); err != nil {
					return
				}
			case <-done:
			}
		}()
		stream := &StreamingResultWithExplicitViewMethodClientStream{conn: wsconn, done: done}
		return stream, nil
	}
}
`

var StreamingResultWithExplicitViewClientStreamRecvCode = `// Recv reads instances of "streamingresultwithexplicitviewservice.Usertype"
// from the "StreamingResultWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingResultWithExplicitViewMethodClientStream) Recv() (*streamingresultwithexplicitviewservice.Usertype, error) {
	var (
		rv   *streamingresultwithexplicitviewservice.Usertype
		body StreamingResultWithExplicitViewMethodResponseBodyExtended
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultWithExplicitViewMethodUsertypeOK(&body)
	vres := &streamingresultwithexplicitviewserviceviews.Usertype{res, "extended"}
	if err := streamingresultwithexplicitviewserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithExplicitViewService", "StreamingResultWithExplicitViewMethod", err)
	}
	result, err := streamingresultwithexplicitviewservice.NewUsertype(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithExplicitViewService", "StreamingResultWithExplicitViewMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "streamingresultwithexplicitviewservice.Usertype" from the
// "StreamingResultWithExplicitViewMethod" endpoint websocket connection with
// context.
func (s *StreamingResultWithExplicitViewMethodClientStream) RecvWithContext(ctx context.Context) (*streamingresultwithexplicitviewservice.Usertype, error) {
	var (
		rv   *streamingresultwithexplicitviewservice.Usertype
		body StreamingResultWithExplicitViewMethodResponseBodyExtended
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultWithExplicitViewMethodUsertypeOK(&body)
	vres := &streamingresultwithexplicitviewserviceviews.Usertype{res, "extended"}
	if err := streamingresultwithexplicitviewserviceviews.ValidateUsertype(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithExplicitViewService", "StreamingResultWithExplicitViewMethod", err)
	}
	result, err := streamingresultwithexplicitviewservice.NewUsertype(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultWithExplicitViewService", "StreamingResultWithExplicitViewMethod", err)
	}
	return result, nil
}
`

var StreamingResultWithExplicitViewServerStreamSendCode = `// SendWithContext streams instances of
// "streamingresultwithexplicitviewservice.Usertype" to the
// "StreamingResultWithExplicitViewMethod" endpoint websocket connection with
// context.
func (s *StreamingResultWithExplicitViewMethodServerStream) SendWithContext(ctx context.Context, v *streamingresultwithexplicitviewservice.Usertype) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res, err := streamingresultwithexplicitviewservice.NewViewedUsertype(v, "extended")
		if err != nil {
			return err
		}
		body := NewStreamingResultWithExplicitViewMethodResponseBodyExtended(res.Projected)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "streamingresultwithexplicitviewservice.Usertype"
// to the "StreamingResultWithExplicitViewMethod" endpoint websocket connection.
func (s *StreamingResultWithExplicitViewMethodServerStream) Send(v *streamingresultwithexplicitviewservice.Usertype) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultCollectionWithViewsServerStreamSendCode = `// SendWithContext streams instances of
// "streamingresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingResultCollectionWithViewsMethod" endpoint websocket connection
// with context.
func (s *StreamingResultCollectionWithViewsMethodServerStream) SendWithContext(ctx context.Context, v streamingresultcollectionwithviewsservice.UsertypeCollection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			respHdr := make(http.Header)
			respHdr.Add("loom-view", s.view)
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, respHdr)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res, err := streamingresultcollectionwithviewsservice.NewViewedUsertypeCollection(v, s.view)
		if err != nil {
			return err
		}
		var body any
		switch s.view {
		case "tiny":
			body = NewUsertypeResponseTinyCollection(res.Projected)
		case "extended":
			body = NewUsertypeResponseExtendedCollection(res.Projected)
		case "default", "":
			body = NewUsertypeResponseCollection(res.Projected)
		}
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "streamingresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingResultCollectionWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultCollectionWithViewsMethodServerStream) Send(v streamingresultcollectionwithviewsservice.UsertypeCollection) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultCollectionWithViewsServerStreamSetViewCode = `// SetView sets the view to render the
// streamingresultcollectionwithviewsservice.UsertypeCollection type before
// sending to the "StreamingResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingResultCollectionWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`

var StreamingResultCollectionWithViewsClientStreamRecvCode = `// Recv reads instances of
// "streamingresultcollectionwithviewsservice.UsertypeCollection" from the
// "StreamingResultCollectionWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultCollectionWithViewsMethodClientStream) Recv() (streamingresultcollectionwithviewsservice.UsertypeCollection, error) {
	var (
		rv   streamingresultcollectionwithviewsservice.UsertypeCollection
		body StreamingResultCollectionWithViewsMethodResponseBody
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultCollectionWithViewsMethodUsertypeCollectionOK(body)
	vres := streamingresultcollectionwithviewsserviceviews.UsertypeCollection{res, s.view}
	if err := streamingresultcollectionwithviewsserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithViewsService", "StreamingResultCollectionWithViewsMethod", err)
	}
	result, err := streamingresultcollectionwithviewsservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithViewsService", "StreamingResultCollectionWithViewsMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "streamingresultcollectionwithviewsservice.UsertypeCollection" from the
// "StreamingResultCollectionWithViewsMethod" endpoint websocket connection
// with context.
func (s *StreamingResultCollectionWithViewsMethodClientStream) RecvWithContext(ctx context.Context) (streamingresultcollectionwithviewsservice.UsertypeCollection, error) {
	var (
		rv   streamingresultcollectionwithviewsservice.UsertypeCollection
		body StreamingResultCollectionWithViewsMethodResponseBody
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultCollectionWithViewsMethodUsertypeCollectionOK(body)
	vres := streamingresultcollectionwithviewsserviceviews.UsertypeCollection{res, s.view}
	if err := streamingresultcollectionwithviewsserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithViewsService", "StreamingResultCollectionWithViewsMethod", err)
	}
	result, err := streamingresultcollectionwithviewsservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithViewsService", "StreamingResultCollectionWithViewsMethod", err)
	}
	return result, nil
}
`

var StreamingResultCollectionWithViewsClientStreamSetViewCode = `// SetView sets the view to render the  type before sending to the
// "StreamingResultCollectionWithViewsMethod" endpoint websocket connection.
func (s *StreamingResultCollectionWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`

var StreamingResultCollectionWithExplicitViewServerStreamSendCode = `// SendWithContext streams instances of
// "streamingresultcollectionwithexplicitviewservice.UsertypeCollection" to the
// "StreamingResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingResultCollectionWithExplicitViewMethodServerStream) SendWithContext(ctx context.Context, v streamingresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res, err := streamingresultcollectionwithexplicitviewservice.NewViewedUsertypeCollection(v, "tiny")
		if err != nil {
			return err
		}
		body := NewUsertypeResponseTinyCollection(res.Projected)
		return s.conn.WriteJSON(ctx, body)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of
// "streamingresultcollectionwithexplicitviewservice.UsertypeCollection" to the
// "StreamingResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingResultCollectionWithExplicitViewMethodServerStream) Send(v streamingresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	return s.SendWithContext(s.r.Context(), v)
}
`
var StreamingResultCollectionWithExplicitViewClientEndpointCode = `// StreamingResultCollectionWithExplicitViewMethod returns an endpoint that
// makes HTTP requests to the StreamingResultCollectionWithExplicitViewService
// service StreamingResultCollectionWithExplicitViewMethod server.
func (c *Client) StreamingResultCollectionWithExplicitViewMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingResultCollectionWithExplicitViewMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingResultCollectionWithExplicitViewMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingResultCollectionWithExplicitViewService", "StreamingResultCollectionWithExplicitViewMethod", err)
		}
		if c.configurer.StreamingResultCollectionWithExplicitViewMethodFn != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			conn = c.configurer.StreamingResultCollectionWithExplicitViewMethodFn(conn, cancel)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if err := wsconn.WriteClose("client closing connection"); err != nil {
					return
				}
				if err := wsconn.Close(); err != nil {
					return
				}
			case <-done:
			}
		}()
		stream := &StreamingResultCollectionWithExplicitViewMethodClientStream{conn: wsconn, done: done}
		return stream, nil
	}
}
`

var StreamingResultCollectionWithExplicitViewClientStreamRecvCode = `// Recv reads instances of
// "streamingresultcollectionwithexplicitviewservice.UsertypeCollection" from
// the "StreamingResultCollectionWithExplicitViewMethod" endpoint websocket
// connection.
func (s *StreamingResultCollectionWithExplicitViewMethodClientStream) Recv() (streamingresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var (
		rv   streamingresultcollectionwithexplicitviewservice.UsertypeCollection
		body UsertypeResponseTinyCollection
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultCollectionWithExplicitViewMethodUsertypeCollectionOK(body)
	vres := streamingresultcollectionwithexplicitviewserviceviews.UsertypeCollection{res, "tiny"}
	if err := streamingresultcollectionwithexplicitviewserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithExplicitViewService", "StreamingResultCollectionWithExplicitViewMethod", err)
	}
	result, err := streamingresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithExplicitViewService", "StreamingResultCollectionWithExplicitViewMethod", err)
	}
	return result, nil
}

// RecvWithContext reads instances of
// "streamingresultcollectionwithexplicitviewservice.UsertypeCollection" from
// the "StreamingResultCollectionWithExplicitViewMethod" endpoint websocket
// connection with context.
func (s *StreamingResultCollectionWithExplicitViewMethodClientStream) RecvWithContext(ctx context.Context) (streamingresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var (
		rv   streamingresultcollectionwithexplicitviewservice.UsertypeCollection
		body UsertypeResponseTinyCollection
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingResultCollectionWithExplicitViewMethodUsertypeCollectionOK(body)
	vres := streamingresultcollectionwithexplicitviewserviceviews.UsertypeCollection{res, "tiny"}
	if err := streamingresultcollectionwithexplicitviewserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithExplicitViewService", "StreamingResultCollectionWithExplicitViewMethod", err)
	}
	result, err := streamingresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres)
	if err != nil {
		return rv, loomhttp.ErrValidationError("StreamingResultCollectionWithExplicitViewService", "StreamingResultCollectionWithExplicitViewMethod", err)
	}
	return result, nil
}
`

var StreamingResultPrimitiveServerStreamSendCode = `// SendWithContext streams instances of "string" to the
// "StreamingResultPrimitiveMethod" endpoint websocket connection with context.
func (s *StreamingResultPrimitiveMethodServerStream) SendWithContext(ctx context.Context, v string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res := v
		return s.conn.WriteJSON(ctx, res)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "string" to the "StreamingResultPrimitiveMethod"
// endpoint websocket connection.
func (s *StreamingResultPrimitiveMethodServerStream) Send(v string) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultPrimitiveClientStreamRecvCode = `// Recv reads instances of "string" from the "StreamingResultPrimitiveMethod"
// endpoint websocket connection.
func (s *StreamingResultPrimitiveMethodClientStream) Recv() (string, error) {
	var (
		rv   string
		body string
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// RecvWithContext reads instances of "string" from the
// "StreamingResultPrimitiveMethod" endpoint websocket connection with context.
func (s *StreamingResultPrimitiveMethodClientStream) RecvWithContext(ctx context.Context) (string, error) {
	var (
		rv   string
		body string
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}
`

var StreamingResultPrimitiveArrayServerStreamSendCode = `// SendWithContext streams instances of "[]int32" to the
// "StreamingResultPrimitiveArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveArrayMethodServerStream) SendWithContext(ctx context.Context, v []int32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	err := func() error {
		var err error
		// Upgrade the HTTP connection to a websocket connection only once. Connection
		// upgrade is done here so that authorization logic in the endpoint is executed
		// before calling the actual service method which may call Send().
		s.once.Do(func() {
			var conn *websocket.Conn
			conn, err = s.upgrader.Upgrade(s.w, s.r, nil)
			if err != nil {
				s.upgradeErr = err
				return
			}
			if s.configurer != nil {
				conn = s.configurer(conn, s.cancel)
			}
			s.conn.SetConn(conn)
			if err = ctx.Err(); err != nil {
				if closeErr := s.conn.Close(); closeErr != nil {
					s.upgradeErr = closeErr
					return
				}
				s.upgradeErr = err
				return
			}
		})
		if s.upgradeErr != nil {
			return s.upgradeErr
		}
		res := v
		return s.conn.WriteJSON(ctx, res)
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// Send streams instances of "[]int32" to the
// "StreamingResultPrimitiveArrayMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveArrayMethodServerStream) Send(v []int32) error {
	return s.SendWithContext(s.r.Context(), v)
}
`

var StreamingResultPrimitiveArrayClientStreamRecvCode = `// Recv reads instances of "[]int32" from the
// "StreamingResultPrimitiveArrayMethod" endpoint websocket connection.
func (s *StreamingResultPrimitiveArrayMethodClientStream) Recv() ([]int32, error) {
	var (
		rv   []int32
		body []int32
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}

// RecvWithContext reads instances of "[]int32" from the
// "StreamingResultPrimitiveArrayMethod" endpoint websocket connection with
// context.
func (s *StreamingResultPrimitiveArrayMethodClientStream) RecvWithContext(ctx context.Context) ([]int32, error) {
	var (
		rv   []int32
		body []int32
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.closeOnce.Do(func() {
			if s.done != nil {
				close(s.done)
			}
		})
		if closeErr := s.conn.Close(); closeErr != nil {
			return rv, closeErr
		}
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	return body, nil
}
`
