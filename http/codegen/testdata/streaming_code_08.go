package testdata

var StreamingResultNoPayloadClientEndpointCode = `// StreamingResultNoPayloadMethod returns an endpoint that makes HTTP requests
// to the StreamingResultNoPayloadService service
// StreamingResultNoPayloadMethod server.
func (c *Client) StreamingResultNoPayloadMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingResultNoPayloadMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingResultNoPayloadMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingResultNoPayloadService", "StreamingResultNoPayloadMethod", err)
		}
		if c.configurer.StreamingResultNoPayloadMethodFn != nil {
			var cancel context.CancelFunc
			ctx, cancel = context.WithCancel(ctx)
			conn = c.configurer.StreamingResultNoPayloadMethodFn(conn, cancel)
		}
		done := make(chan struct{})
		go func() {
			select {
			case <-ctx.Done():
				if err := conn.WriteControl(
					websocket.CloseMessage,
					websocket.FormatCloseMessage(websocket.CloseNormalClosure, "client closing connection"),
					time.Now().Add(time.Second),
				); err != nil {
					return
				}
				if err := conn.Close(); err != nil {
					return
				}
			case <-done:
			}
		}()
		stream := &StreamingResultNoPayloadMethodClientStream{conn: conn, done: done}
		return stream, nil
	}
}
`
