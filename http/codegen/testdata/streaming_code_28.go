package testdata

var StreamingPayloadResultWithViewsClientStreamSetViewCode = `// SetView sets the view used to validate
// streamingpayloadresultwithviewsservice.Usertype results received from the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
