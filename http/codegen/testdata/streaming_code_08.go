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
		ctx, cancel := context.WithCancel(ctx)
		wsconn := loomhttp.NewWebSocketStream(c.configurer.StreamingResultNoPayloadMethodConfig)
		if err := wsconn.SetConn(ctx, conn, cancel); err != nil {
			cancel()
			return nil, loomhttp.ErrRequestError("StreamingResultNoPayloadService", "StreamingResultNoPayloadMethod", err)
		}
		stream := &StreamingResultNoPayloadMethodClientStream{conn: wsconn}
		return stream, nil
	}
}
`
