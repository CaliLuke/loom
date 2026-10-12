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
		ctx, cancel := context.WithCancel(ctx)
		wsconn := loomhttp.NewWebSocketStream(c.configurer.BidirectionalStreamingNoPayloadMethodConfig)
		if err := wsconn.SetConn(ctx, conn, cancel); err != nil {
			cancel()
			return nil, loomhttp.ErrRequestError("BidirectionalStreamingNoPayloadService", "BidirectionalStreamingNoPayloadMethod", err)
		}
		stream := &BidirectionalStreamingNoPayloadMethodClientStream{conn: wsconn}
		return stream, nil
	}
}
`
