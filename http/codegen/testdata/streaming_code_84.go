package testdata


var BidirectionalStreamingResultWithViewsClientStreamSetViewCode = `// SetView sets the view to render the float32 type before sending to the
// "BidirectionalStreamingResultWithViewsMethod" endpoint websocket connection.
func (s *BidirectionalStreamingResultWithViewsMethodClientStream) SetView(view string) {
	s.view = view
}
`


