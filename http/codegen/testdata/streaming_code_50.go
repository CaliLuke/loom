package testdata

var StreamingPayloadPrimitiveArrayClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection and
// reads instances of "[]string" from the connection.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) CloseAndRecv() ([]string, error) {
	var (
		rv   []string
		body []loom.Nullable[string]
		err  error
	)
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(context.Background(), nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingPayloadPrimitiveArrayMethodStringOK(body)
	return res, nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadPrimitiveArrayMethod" endpoint websocket connection and
// reads instances of "[]string" from the connection with context.
func (s *StreamingPayloadPrimitiveArrayMethodClientStream) CloseAndRecvWithContext(ctx context.Context) ([]string, error) {
	var (
		rv   []string
		body []loom.Nullable[string]
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(ctx, nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	for i, e := range body {
		if _, ok := e.Value(); !ok {
			err = loom.MergeErrors(err, loom.InvalidNullElementError("body", i))
		}
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingPayloadPrimitiveArrayMethodStringOK(body)
	return res, nil
}
`
