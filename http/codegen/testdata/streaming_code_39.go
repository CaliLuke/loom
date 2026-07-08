package testdata


var StreamingPayloadResultCollectionWithExplicitViewServerStreamSendCode = `// SendAndCloseWithContext streams instances of
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint
// websocket connection with context and closes the connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) SendAndCloseWithContext(ctx context.Context, v streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection) error {
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
		res := streamingpayloadresultcollectionwithexplicitviewservice.NewViewedUsertypeCollection(v, "tiny")
		body := NewUsertypeResponseTinyCollection(res.Projected)
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
// "streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection"
// to the "StreamingPayloadResultCollectionWithExplicitViewMethod" endpoint
// websocket connection and closes the connection.
func (s *StreamingPayloadResultCollectionWithExplicitViewMethodServerStream) SendAndClose(v streamingpayloadresultcollectionwithexplicitviewservice.UsertypeCollection) error {
	return s.SendAndCloseWithContext(s.r.Context(), v)
}
`


