package testdata

var BidirectionalStreamingClientEndpointCode = `// BidirectionalStreamingMethod returns an endpoint that makes HTTP requests to
// the BidirectionalStreamingService service BidirectionalStreamingMethod
// server.
func (c *Client) BidirectionalStreamingMethod() loom.Endpoint {
	var (
		encodeRequest  = EncodeBidirectionalStreamingMethodRequest(c.encoder)
		decodeResponse = DecodeBidirectionalStreamingMethodResponse(c.decoder, c.RestoreResponseBody)
	)
	return func(ctx context.Context, v any) (any, error) {
		req, err := c.BuildBidirectionalStreamingMethodRequest(ctx, v)
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
			return nil, loomhttp.ErrRequestError("BidirectionalStreamingService", "BidirectionalStreamingMethod", err)
		}
		if c.configurer.BidirectionalStreamingMethodFn != nil {
			conn = c.configurer.BidirectionalStreamingMethodFn(conn, nil)
		}
		wsconn := loomhttp.NewWebSocketStream(conn)
		stream := &BidirectionalStreamingMethodClientStream{conn: wsconn}
		return stream, nil
	}
}
`
