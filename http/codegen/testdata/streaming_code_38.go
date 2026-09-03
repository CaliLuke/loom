package testdata

var StreamingPayloadResultCollectionWithViewsClientStreamSetViewCode = `// SetView sets the view to render the loom.JSONValue type before sending to
// the "StreamingPayloadResultCollectionWithViewsMethod" endpoint websocket
// connection.
func (s *StreamingPayloadResultCollectionWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`
