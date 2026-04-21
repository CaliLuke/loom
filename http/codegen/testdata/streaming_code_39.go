package testdata


var StreamingPayloadResultCollectionWithExplicitViewServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint
// websocket connection with context and closes the connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) SendAndCloseWithContext(ctx context.Context, v streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	defer s.conn.Close()
	res := streamingpayloadresultcollectionwithexplicitviewservice.NewViewedUsertypeCollection(v, "tiny")
	body := NewUsertypeResponseTinyCollection(res.Projected)
	return s.conn.WriteJSON(body)
}

// SendAndClose streams instances of
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint
// websocket connection and closes the connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) SendAndClose(v streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`


