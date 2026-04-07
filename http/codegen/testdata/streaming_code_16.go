package testdata


var StreamingPayloadNoPayloadClientEndpointCode = `// StreamingPayloadNoPayloadMethod returns an endpoint that makes HTTP requests
// to the StreamingPayloadNoPayloadService service
// StreamingPayloadNoPayloadMethod server.
func (c *Client) StreamingPayloadNoPayloadMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeStreamingPayloadNoPayloadMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildStreamingPayloadNoPayloadMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("StreamingPayloadNoPayloadService", "StreamingPayloadNoPayloadMethod", err)
		}
		if c.configurer.StreamingPayloadNoPayloadMethodFn != nil {
			conn = c.configurer.StreamingPayloadNoPayloadMethodFn(conn, nil)
		}
		stream := &StreamingPayloadNoPayloadMethodClientStream{conn: conn}
		return stream, nil
	}
}
`


