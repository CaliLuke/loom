package streaming

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/CaliLuke/loom/pulse/internal/redisop"
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
	if s.lifetime.Stopping() {
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
		record, _, err := readSinkMembership(ctx, stream, s.Name)
		if err != nil {
			return err
		}
		if record != nil && *record.Filter != s.filterIdentity {
			return fmt.Errorf("sink group %q filter conflicts with its persisted identity", s.Name)
		}
		var members []string
		if record != nil {
			members = record.memberNames()
		}
		if !registration.recoveryPending && slices.Contains(members, s.consumer) {
			continue
		}
		registration.recoveryPending = true
		// Publish before group creation so peer removal observes our ownership.
		if _, err := changeSinkMembership(ctx, cm, s.Name, s.filterIdentity, sinkMemberChange{add: s.consumer, restore: true}); err != nil {
			return fmt.Errorf("failed to restore membership on stream %s: %w", stream.Name, err)
		}
		if err := stream.createGroup(ctx, s.Name, registration.startID); err != nil {
			return fmt.Errorf("failed to restore group on stream %s: %w", stream.Name, err)
		}
		if err := redisop.Client(stream.rdb).XGroupCreateConsumer(ctx, stream.key, s.Name, s.consumer).Err(); err != nil {
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
	for _, registration := range s.streams {
		stream := registration.stream
		cm := s.consumersMap[stream.Name]
		if cm == nil {
			return fmt.Errorf("missing sink membership map for stream %s", stream.Name)
		}
		if _, err := changeSinkMembership(ctx, cm, s.Name, s.filterIdentity, sinkMemberChange{
			add:         consumer,
			restore:     true,
			keepAliveAt: now,
		}); err != nil {
			return fmt.Errorf("failed to publish sink keep-alive on stream %s: %w", stream.Name, err)
		}
	}
	s.lastKeepAlive = now
	return nil
}
