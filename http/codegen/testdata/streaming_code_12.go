package testdata

var StreamingPayloadClientEndpointCode = `// StreamingPayloadMethod returns an endpoint that makes HTTP requests to the
// StreamingPayloadService service StreamingPayloadMethod server.
func (c *Client) StreamingPayloadMethod() loom.Endpoint {
	var (
		encodeRequest  = EncodeStreamingPayloadMethodRequest(c.encoder)
		decodeResponse = DecodeStreamingPayloadMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingPayloadMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		err = encodeRequest(req, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingPayloadService", "StreamingPayloadMethod", err)
		}
		wsconn := loomhttp.NewWebSocketStream(c.configurer.StreamingPayloadMethodConfig)
		if err := wsconn.SetConn(conn, nil); err != nil {
			return nil, loomhttp.ErrRequestError("StreamingPayloadService", "StreamingPayloadMethod", err)
		}
		stream := &StreamingPayloadMethodClientStream{conn: wsconn}
		return stream, nil
	}
}
`
