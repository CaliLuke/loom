package streaming

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/oklog/ulid/v2"
)

// ensureConsumer replaces a stale consumer only after its replacement is
// registered on every active stream. Failed retirement stays owned so removal
// of a stream can remove every name this sink registered, including names from
// a failed preparation whose rollback also failed.
func (s *Sink) ensureConsumer(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.closing || len(s.streams) == 0 {
		return nil
	}
	if time.Since(time.Unix(0, s.lastKeepAlive)) > 2*s.ackGracePeriod {
		s.logger.Debug("consumer stale, creating new one")
		consumer, err := s.newConsumer(ctx)
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to create new consumer: %w", err))
			return err
		}
		// newConsumer adds the candidate last. It is now current; the old
		// current name becomes retired instead, before attempting cleanup.
		s.retiredConsumers[len(s.retiredConsumers)-1] = s.consumer
		s.consumer = consumer
	}
	if err := s.cleanupRetiredConsumers(ctx); err != nil {
		s.logger.Error(err)
	}
	return nil
}

// newConsumer prepares a replacement on all active streams. The candidate is
// tracked before the first Redis write because even an error reply can mean a
// write succeeded. The caller commits the name while still holding s.lock.
func (s *Sink) newConsumer(ctx context.Context) (consumer string, err error) {
	consumer = ulid.Make().String()
	s.retiredConsumers = append(s.retiredConsumers, consumer)
	previousKeepAlive := s.lastKeepAlive
	defer func() {
		if err != nil {
			s.lastKeepAlive = previousKeepAlive
			err = errors.Join(err, s.cleanupRetiredConsumers(context.WithoutCancel(ctx)))
		}
	}()
	for _, stream := range s.streams {
		cm := s.consumersMap[stream.Name]
		if _, err = cm.AppendUniqueValues(ctx, s.Name, consumer); err != nil {
			return "", fmt.Errorf("failed to register replacement consumer on stream %s: %w", stream.Name, err)
		}
		if err = s.createConsumer(ctx, stream, consumer); err != nil {
			return "", err
		}
	}
	return consumer, nil
}

// cleanupRetiredConsumers forgets a retired name only after every owned map
// confirms its removal. Redis consumers and their pending entries remain for
// the idle-message claim path; deleting those consumers would lose the PEL.
// s.lock must be held.
func (s *Sink) cleanupRetiredConsumers(ctx context.Context) error {
	retained := s.retiredConsumers[:0]
	var result error
	for _, consumer := range s.retiredConsumers {
		var failed bool
		for name, cm := range s.consumersMap {
			if _, _, err := cm.RemoveValues(ctx, s.Name, consumer); err != nil {
				failed = true
				result = errors.Join(result, fmt.Errorf("failed to retire consumer %s on stream %s: %w", consumer, name, err))
			}
		}
		if failed {
			retained = append(retained, consumer)
		}
	}
	s.retiredConsumers = retained
	return result
}
