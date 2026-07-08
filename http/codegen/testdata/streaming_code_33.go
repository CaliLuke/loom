package testdata

var StreamingPayloadResultCollectionWithViewsServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection with context and closes the connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodServerStream) SendAndCloseWithContext(ctx context.Context, v streamingpayloadresultcollectionwithviewsservice.UsertypeCollection) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	stopContextWatch := context.AfterFunc(ctx, func() {
		if s.conn == nil {
			return
		}
		if closeErr := s.conn.Close(); closeErr != nil {
			return
		}
	})
	defer stopContextWatch()
	err := func() error {
		defer s.conn.Close()
		res, err := streamingpayloadresultcollectionwithviewsservice.NewViewedUsertypeCollection(v, s.view)
		if err != nil {
			return err
		}
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
	}()
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
	}
	return err
}

// SendAndClose streams instances of
// "streamingpayloadresultcollectionwithviewsservice.UsertypeCollection" to the
// "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection and closes the connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodServerStream) SendAndClose(v streamingpayloadresultcollectionwithviewsservice.UsertypeCollection) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`
