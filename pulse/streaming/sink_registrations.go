package streaming

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"
)

type sinkRegistration struct {
	stream          *Stream
	startID         string
	recoveryPending bool
}

// reconcileRegistrations restores only currently active registrations. The
// lifecycle mutex must be held throughout: RemoveStream and Close cannot leave a
// late repair behind. A failed command may have committed, so recovery remains
// owned until all steps succeed. Existing groups retain their shared cursor.
func (s *Sink) reconcileRegistrations(ctx context.Context, missingGroup bool) error {
	if s.closing {
		return nil
	}
	if missingGroup {
		for i := range s.streams {
			s.streams[i].recoveryPending = true
		}
	}
	for i := range s.streams {
		registration := &s.streams[i]
		stream := registration.stream
		cm := s.consumersMap[stream.Name]
		members, _ := cm.GetValues(s.Name)
		if !registration.recoveryPending && slices.Contains(members, s.consumer) {
			continue
		}
		registration.recoveryPending = true
		// Publish before group creation so peer removal observes our ownership.
		if _, err := cm.AppendUniqueValues(ctx, s.Name, s.consumer); err != nil {
			return fmt.Errorf("failed to restore membership on stream %s: %w", stream.Name, err)
		}
		if err := stream.createGroup(ctx, s.Name, registration.startID); err != nil {
			return fmt.Errorf("failed to restore group on stream %s: %w", stream.Name, err)
		}
		if err := stream.rdb.XGroupCreateConsumer(ctx, stream.key, s.Name, s.consumer).Err(); err != nil {
			return fmt.Errorf("failed to restore consumer on stream %s: %w", stream.Name, err)
		}
		// This may already be an existing consumer with pending messages. A
		// keepalive failure must not delete it or discard its pending entries.
		if err := s.updateConsumerKeepAlive(ctx, s.consumer); err != nil {
			return fmt.Errorf("failed to restore consumer keep-alive on stream %s: %w", stream.Name, err)
		}
		registration.recoveryPending = false
	}
	return nil
}

// updateConsumerKeepAlive records a confirmed keepalive while the lifecycle lock
// is held, or during private constructor setup before any worker starts.
func (s *Sink) updateConsumerKeepAlive(ctx context.Context, consumer string) error {
	now := time.Now().UnixNano()
	if _, err := s.consumersKeepAliveMap.Set(ctx, consumer, strconv.FormatInt(now, 10)); err != nil {
		return err
	}
	s.lastKeepAlive = now
	return nil
}
