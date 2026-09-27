package testdata

var StreamingPayloadResultCollectionWithViewsClientStreamSetViewCode = `// SetView sets the view used to validate
// streamingpayloadresultcollectionwithviewsservice.UsertypeCollection results
// received from the "StreamingPayloadResultCollectionWithViewsMethod" endpoint
// websocket connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
