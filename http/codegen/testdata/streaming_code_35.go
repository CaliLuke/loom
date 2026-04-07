package testdata


var StreamingPayloadResultCollectionWithViewsServerStreamSetViewCode = `// SetView sets the view to render the
// streamingpayloadresultcollectionwithviewsservice.UsertypeCollection type
// before sending to the "StreamingPayloadResultCollectionWithViewsMethod"
// endpoint websocket connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`


