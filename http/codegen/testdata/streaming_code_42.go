package testdata


var StreamingPayloadResultCollectionWithExplicitViewClientStreamRecvCode = `// CloseAndRecv stops sending messages to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection and reads instances of
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// from the connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) CloseAndRecv() (streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	var (
		rv   streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection
		body StreamingPayloadResultCollectionWithExplicitViewMethodResponseBody
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
	res := NewStreamingPayloadResultCollectionWithExplicitViewMethodUsertypeCollectionOK(body)
	vres := streamingpayloadresultcollectionwithexplicitviewserviceviews.UsertypeCollection{res, "tiny"}
	if err := streamingpayloadresultcollectionwithexplicitviewserviceviews.ValidateUsertypeCollection(vres); err != nil {
		return rv, loomhttp.ErrValidationError("StreamingPayloadResultCollectionWithExplicitViewService", "StreamingPayloadResultCollectionWithExplicitViewMethod", err)
	}
	return streamingpayloadresultcollectionwithexplicitviewservice.NewUsertypeCollection(vres), nil
}

// CloseAndRecvWithContext stops sending messages to the
// "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint websocket
// connection and reads instances of
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// from the connection with context.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodClientStream) CloseAndRecvWithContext(ctx context.Context) (streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection, error) {
	return s.CloseAndRecv()
}
`


