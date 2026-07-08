package testdata

var BidirectionalStreamingNoPayloadClientEndpointCode = `// BidirectionalStreamingNoPayloadMethod returns an endpoint that makes HTTP
// requests to the BidirectionalStreamingNoPayloadService service
// BidirectionalStreamingNoPayloadMethod server.
func (c *Client) BidirectionalStreamingNoPayloadMethod() loom.Endpoint {
	var (
		decodeResponse = DecodeBidirectionalStreamingNoPayloadMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildBidirectionalStreamingNoPayloadMethodRequest(ctx, v)
		if err != nil {
			return nil, err
		}
		conn, resp, err := c.dialer.DialContext(ctx, req.URL.String(), req.Header)
		if err != nil {
			if resp != nil {
				return decodeResponse(resp)
			}
			return nil, loomhttp.ErrRequestError("BidirectionalStreamingNoPayloadService", "BidirectionalStreamingNoPayloadMethod", err)
		}
		if c.configurer.BidirectionalStreamingNoPayloadMethodFn != nil {
			conn = c.configurer.BidirectionalStreamingNoPayloadMethodFn(conn, nil)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		stream := &BidirectionalStreamingNoPayloadMethodClientStream{conn: wsconn}
		return stream, nil
	}
}
`
