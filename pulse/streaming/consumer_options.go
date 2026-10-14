package streaming

import "fmt"

// validateConsumerBufferSize admits the channel capacity shared by Reader and
// Sink before either constructor publishes runtime or Redis state.
func validateConsumerBufferSize(size int) error {
	if size < 0 {
		return fmt.Errorf("consumer buffer size must be >= 0, got %d", size)
	}
	return nil
}
