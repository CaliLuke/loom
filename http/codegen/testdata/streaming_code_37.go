package testdata


var StreamingPayloadResultCollectionWithViewsClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection and reads instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" from
// the connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) CloseAndRecv() (streamingpayloadresultcollectionwithviewsservice.UsertypeCollection, error) {
	var (
		rv   streamingpayloadresultcollectionwithviewsservice.UsertypeCollection
		body StreamingPayloadResultCollectionWithViewsMethodResponseBody
		err  error
	)
	defer s.conn.Close()
	// Send a nil payload to the server implying end of message
	if err = s.conn.WriteJSON(nil); err != nil {
		return rv, err
	}
	err = s.conn.ReadJSON(&body)
	if websocket.IsCloseError(err, websocket.CloseNormalClosure) {
		s.conn.Close()
		return rv, io.EOF
	}
	if err != nil {
		return rv, err
	}
	res := NewStreamingPayloadResultCollectionWithViewsMethodUsertypeCollectionOK(body)
	vres := streamingpayloadresultcollectionwithviewsserviceviews.UsertypeCollection{res, s.view}
	if err := streamingpayloadresultcollectionwithviewsserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingPayloadResultCollectionWithViewsService", "StreamingPayloadResultCollectionWithViewsMethod", err)
	}
	return streamingpayloadresultcollectionwithviewsservice.NewUsertypeCollection(vres), nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection and reads instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" from
// the connection with context.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (streamingpayloadresultcollectionwithviewsservice.UsertypeCollection, error) {
	return s.CloseAndRecv()
}
`


