package testdata


var StreamingPayloadResultWithViewsServerStreamSetViewCode = `// SetView sets the view to render the
// streamingpayloadresultwithviewsservice.Usertype type before sending to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithViewsMethodServerStream) SetView(view string) {
	s.view = view
}
`


