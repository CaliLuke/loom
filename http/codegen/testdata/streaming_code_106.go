package testdata

var BidirectionalStreamingPrimitiveArrayClientStreamRecvCode = `// Recv reads instances of "[]string" from the
// "BidirectionalStreamingPrimitiveArrayMethod" endpoint websocket connection.
func (s *BidirectionalStreamingPrimitiveArrayMethodClientStream) Recv() ([]string, error) {
	var (
		rv   []string
		body []loom.Nullable[string]
		err  error
	)
	err = s.conn.ReadJSON(context.Background(), &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
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
	res := NewBidirectionalStreamingPrimitiveArrayMethodStringOK(body)
	return res, nil
}

// RecvWithContext reads instances of "[]string" from the
// "BidirectionalStreamingPrimitiveArrayMethod" endpoint websocket connection
// with context.
func (s *BidirectionalStreamingPrimitiveArrayMethodClientStream) RecvWithContext(ctx context.Context) ([]string, error) {
	var (
		rv   []string
		body []loom.Nullable[string]
		err  error
	)
	if err := ctx.Err(); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(ctx, &body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
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
	res := NewBidirectionalStreamingPrimitiveArrayMethodStringOK(body)
	return res, nil
}
`
