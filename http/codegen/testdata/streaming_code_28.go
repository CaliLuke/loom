package testdata


var StreamingPayloadResultWithViewsClientStreamSetViewCode = `// SetView sets the view to render the float32 type before sending to the
// "StreamingPayloadResultWithViewsMethod" endpoint websocket connection.
func (s *StreamingPayloadResultWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`


