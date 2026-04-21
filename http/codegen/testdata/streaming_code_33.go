package testdata


var StreamingPayloadResultCollectionWithViewsServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection with context and closes the connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodServerStream) SendAndCloseWithContext(ctx context.Context, v streamingpayloadresultcollectionwithviewsservice.UsertypeCollection) error {
	defer s.conn.Close()
	res := streamingpayloadresultcollectionwithviewsservice.NewViewedUsertypeCollection(v, s.view)
	var body any
	switch s.view {
	case "tiny":
		body = NewUsertypeResponseTinyCollection(res.Projected)
	case "extended":
		body = NewUsertypeResponseExtendedCollection(res.Projected)
	case "default", "":
		body = NewUsertypeResponseCollection(res.Projected)
	}
	return s.conn.WriteJSON(body)
}

// SendAndClose streams instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection and closes the connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodServerStream) SendAndClose(v streamingpayloadresultcollectionwithviewsservice.UsertypeCollection) error {
	return s.SendAndCloseWithContext(context.Background(), v)
}
`


