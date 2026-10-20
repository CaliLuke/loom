package streaming

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"strings"
	"time"

	redis "github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/redisop"
)

type sinkClaimStream struct {
	name     string
	key      string
	consumer string
}

// consumerKeepAliveExpired reports whether elapsed exceeds two admitted positive
// grace periods without overflowing their sum. Equality and future timestamps
// remain live, matching the existing strict stale threshold.
func consumerKeepAliveExpired(elapsed, grace time.Duration) bool {
	return elapsed > grace && elapsed-grace > grace
}

// deleteStreamStaleConsumers deletes stale consumers for a specific stream.
// Consumers that still own pending entries are kept because deleting a Redis
// consumer drops its pending entries; the idle message check claims them first,
// and acks those whose events were deleted.
// s.lock must be held.
func (s *Sink) deleteStreamStaleConsumers(ctx context.Context, stream *Stream) error {
	consumers, err := redisop.Client(s.rdb).XInfoConsumers(ctx, stream.key, s.Name).Result()
	if err != nil {
		return fmt.Errorf("failed to get consumers info: %w", err)
	}
	record, _, err := readSinkMembership(ctx, stream, s.Name)
	if err != nil {
		return err
	}
	candidates := make(map[string]sinkMember)
	for _, consumer := range consumers {
		candidates[consumer.Name] = sinkMember{Name: consumer.Name}
	}
	if record != nil {
		for _, member := range record.Members {
			candidates[member.Name] = member
		}
	}
	for _, member := range candidates {
		// Registration protects a join before its first keep-alive. The latest
		// heartbeat shares the membership CAS used by retirement.
		lastActive := member.RegisteredAt
		if member.LastKeepAlive > 0 {
			lastActive = member.LastKeepAlive
		}
		if lastActive > 0 && !consumerKeepAliveExpired(time.Since(time.Unix(0, lastActive)), s.ackGracePeriod) {
			continue
		}
		deleted, err := s.deleteIdleConsumer(ctx, stream.key, member.Name)
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to delete stale consumer: %w", err), "consumer", member.Name)
			continue
		}
		if !deleted {
			// The atomic Redis check retains every pending entry, including
			// entries delivered after our native inventory snapshot.
			continue
		}
		if sinks := s.consumersMap[stream.Name]; sinks != nil {
			if _, err := changeSinkMembership(ctx, sinks, s.Name, s.filterIdentity, sinkMemberChange{
				remove:            []string{member.Name},
				retireIfUnchanged: map[string]sinkMember{member.Name: member},
			}); err != nil {
				s.logger.Error(fmt.Errorf("failed to remove consumer from map: %w", err), "stream", stream.Name, "consumer", member.Name)
				continue
			}
		}
	}
	return nil
}

// deleteIdleConsumer deletes consumer from the sink consumer group of the
// stream stored at key unless it owns pending entries. The check and the
// deletion run atomically so entries delivered to the consumer concurrently
// are never dropped. It returns true if the consumer was deleted.
func (s *Sink) deleteIdleConsumer(ctx context.Context, key, consumer string) (bool, error) {
	deleted, err := deleteIdleConsumerScript.Run(ctx, redisop.Client(s.rdb), []string{key}, s.Name, consumer).Int64()
	if err != nil {
		return false, fmt.Errorf("failed to delete idle consumer %s: %w", consumer, err)
	}
	return deleted == 1, nil
}

// deleteStaleConsumers deletes stale consumers.
// s.lock must be held.
func (s *Sink) deleteStaleConsumers(ctx context.Context) {
	for _, registration := range s.streams {
		stream := registration.stream
		if err := s.deleteStreamStaleConsumers(ctx, stream); err != nil {
			s.logger.Error(fmt.Errorf("failed to delete stale consumers for stream %s: %w", stream.Name, err))
		}
	}
}

// removeStreamConsumer removes the stream consumer from the sink. The boolean
// reports confirmed deregistration, even when subsequent group cleanup fails.
// The caller owns the map until cleanup succeeds or the sink closes.
func (s *Sink) removeStreamConsumer(ctx context.Context, stream *Stream) (bool, error) {
	consumers := append([]string{s.consumer}, s.retiredConsumers...)
	remains, err := changeSinkMembership(ctx, s.consumersMap[stream.Name], s.Name, s.filterIdentity, sinkMemberChange{remove: consumers})
	if err != nil {
		return false, fmt.Errorf("failed to remove consumer %s from replicated map for stream %s: %w", s.consumer, stream.Name, err)
	}
	if len(remains) == 0 {
		return true, s.deleteConsumerGroup(ctx, stream)
	}
	return true, nil
}

// createConsumer creates consumer in the consumer group of the sink for
// stream and records its keep-alive. If the keep-alive update fails, it
// deletes the consumer again, even if ctx is canceled.
func (s *Sink) createConsumer(ctx context.Context, stream *Stream, consumer string) error {
	if err := redisop.Client(stream.rdb).XGroupCreateConsumer(ctx, stream.key, s.Name, consumer).Err(); err != nil {
		return fmt.Errorf("failed to create Redis consumer %s for consumer group %s: %w", consumer, s.Name, err)
	}
	if err := s.updateConsumerKeepAlive(ctx, consumer); err != nil {
		err = fmt.Errorf("failed to set sink keep-alive for new consumer %s: %w", consumer, err)
		if derr := redisop.Client(stream.rdb).XGroupDelConsumer(context.WithoutCancel(ctx), stream.key, s.Name, consumer).Err(); derr != nil {
			err = errors.Join(err, fmt.Errorf("failed to delete consumer %s after failed keep-alive set: %w", consumer, derr))
		}
		return err
	}
	return nil
}

// read reads events from the streams and sends them to the sink channel.
func (s *Sink) read(ctx context.Context) {
	defer s.logger.Debug("read: exiting")
	defer s.wait.Done()
	for {
		if err := s.ensureConsumer(ctx); err != nil {
			if !s.waitBeforeEnsureConsumerRetry() {
				return
			}
			continue
		}
		readCtx, readStreams, consumer, armed := s.armRead(ctx)
		if !armed {
			if s.isClosing() || !s.waitBeforeEnsureConsumerRetry() {
				return
			}
			continue
		}

		s.logger.Debug("reading", "streams", readStreams, "max", s.maxPolled, "block", s.blockDuration)
		client, block := redisop.Read(s.rdb, s.blockDuration)
		streams, err := client.XReadGroup(readCtx, &redis.XReadGroupArgs{
			Group:    s.Name,
			Consumer: consumer,
			Streams:  readStreams,
			Count:    s.maxPolled,
			Block:    block,
			NoAck:    s.noAck,
		}).Result()
		s.clearActiveRead()

		if s.isClosing() {
			// Honor the Close contract and do not forward any more events.
			// Any events in the PEL will be claimed by another consumer.
			return
		}
		if err != nil {
			if redis.HasErrorPrefix(err, "NOGROUP") {
				s.lock.Lock()
				recoveryErr := s.reconcileRegistrations(ctx, true)
				s.lock.Unlock()
				if recoveryErr != nil {
					s.logger.Error(recoveryErr)
				}
				if !s.waitBeforeEnsureConsumerRetry() {
					return
				}
				continue
			}
			if err := handleReadError(err, s.logger, s.donechan); err != nil {
				s.logger.Error(fmt.Errorf("error reading events: %w", err))
			}
			continue
		}
		for _, events := range streams {
			streamName := events.Stream[len(streamKeyPrefix):]
			subscribers, filter := s.snapshotFanOut()
			streamEvents(ctx, streamName, events.Stream, s.Name, events.Messages, filter, subscribers, s.rdb, s.logger, s.donechan)
		}
	}
}

func (s *Sink) armRead(ctx context.Context) (context.Context, []string, string, bool) {
	readCtx, cancel := context.WithCancel(ctx)
	s.lock.Lock()
	defer s.lock.Unlock()
	if s.lifetime.Stopping() || len(s.streams) == 0 {
		cancel()
		return readCtx, nil, "", false
	}
	readStreams := make([]string, 2*len(s.streams))
	for i, registration := range s.streams {
		readStreams[i] = registration.stream.key
		readStreams[len(s.streams)+i] = ">"
	}
	s.readCancel = cancel
	return readCtx, readStreams, s.consumer, true
}

func (s *Sink) waitBeforeEnsureConsumerRetry() bool {
	delay := min(s.blockDuration, time.Duration(maxJitterMs)*time.Millisecond)
	if delay <= 0 {
		delay = ensureConsumerRetryDelay
	}
	delay = ensureConsumerRetryDelay + time.Duration(rand.Int63n(int64(delay)))
	select {
	case <-time.After(delay):
		return true
	case <-s.donechan:
		return false
	}
}

func (s *Sink) snapshotFanOut() ([]*eventSubscriber, eventFilterFunc) {
	s.lock.Lock()
	defer s.lock.Unlock()
	subscribers := make([]*eventSubscriber, len(s.subscribers))
	copy(subscribers, s.subscribers)
	return subscribers, s.eventFilter
}

func (s *Sink) cancelActiveRead() {
	if s.readCancel != nil {
		s.readCancel()
	}
}

func (s *Sink) clearActiveRead() {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.readCancel = nil
}

// periodicKeepAlive updates this consumer keep-alive every half ack grace period.
func (s *Sink) periodicKeepAlive(ctx context.Context) {
	defer s.wait.Done()
	defer s.logger.Debug("periodicKeepAlive: exiting")
	ticker := time.NewTicker(s.ackGracePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.lock.Lock()
			if s.lifetime.Stopping() {
				s.lock.Unlock()
				return
			}
			if err := s.reconcileRegistrations(ctx, false); err != nil {
				s.logger.Error(err)
			}
			if err := s.updateConsumerKeepAlive(ctx, s.consumer); err != nil {
				s.logger.Error(fmt.Errorf("failed to update sink keep-alive: %w", err))
				s.lock.Unlock()
				continue
			}
			s.lock.Unlock()

		case <-s.donechan:
			return
		}
	}
}

// periodicIdleMessageCheck claims any idle message every check stale period.
// An idle message is one that has not been acked for more than the ack grace period.
// Once all idle messages are claimed, any stale consumer is deleted.
func (s *Sink) periodicIdleMessageCheck(ctx context.Context) {
	defer s.wait.Done()
	defer s.logger.Debug("periodicIdleMessageCheck: exiting")
	checks := s.idleChecks
	var ticker *time.Ticker
	if checks == nil {
		ticker = time.NewTicker(s.idleCheckPeriod)
		checks = ticker.C
		defer ticker.Stop()
	}

	leaseDuration := s.idleCheckPeriod.Milliseconds() - 5
	for {
		select {
		case <-checks:
			now := time.Now().UnixNano() / int64(time.Millisecond)
			newExpiration := now + leaseDuration
			result, err := s.acquireLease.EvalSha(ctx, redisop.Client(s.rdb), s.leaseKeyName, newExpiration, now, leaseDuration).Result()
			if err != nil {
				s.logger.Error(fmt.Errorf("failed to acquire idle message check lease: %w", err))
				continue
			}
			if result != int64(1) {
				// Another sink instance claimed the lease.
				continue
			}
			s.lock.Lock()
			claimStreams := s.snapshotClaimStreams()
			s.lock.Unlock()
			s.claimIdleMessages(ctx, claimStreams)
			s.lock.Lock()
			s.deleteStaleConsumers(ctx)
			s.lock.Unlock()

		case <-s.donechan:
			return
		}
	}
}

func (s *Sink) snapshotClaimStreams() []sinkClaimStream {
	claimStreams := make([]sinkClaimStream, len(s.streams))
	for i, registration := range s.streams {
		stream := registration.stream
		claimStreams[i] = sinkClaimStream{
			name:     stream.Name,
			key:      stream.key,
			consumer: s.consumer,
		}
	}
	return claimStreams
}

func (s *Sink) claimIdleMessages(ctx context.Context, streams []sinkClaimStream) {
	for _, stream := range streams {
		args := claimArgs{
			stream:   stream.key,
			group:    s.Name,
			consumer: stream.consumer,
			minIdle:  s.ackGracePeriod,
			start:    "0-0",
		}
		start, err := s.claim(ctx, stream.name, args)
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to claim idle messages for stream %s: %w", stream.name, err))
			continue
		}
		for start != claimDone {
			args.start = start
			start, err = s.claim(ctx, stream.name, args)
			if err != nil {
				s.logger.Error(fmt.Errorf("failed to claim idle messages for stream %s: %w", stream.name, err))
				break
			}
		}
	}
}

// claim claims one batch of idle messages for claimIdleMessages, delivers the
// claimed events and returns the start id of the next batch.
func (s *Sink) claim(ctx context.Context, streamName string, args claimArgs) (string, error) {
	messages, deleted, start, err := claimIdle(ctx, s.rdb, args)
	if err != nil {
		return "", err
	}
	if deleted > 0 {
		s.logger.Debug("acked pending entries whose events were deleted", "stream", streamName, "entries", deleted)
	}
	if len(messages) > 0 {
		s.logger.Info("claimed", "stream", streamName, "messages", len(messages))
		subscribers, filter := s.snapshotFanOut()
		streamEvents(ctx, streamName, args.stream, s.Name, messages, filter, subscribers, s.rdb, s.logger, s.donechan)
	}
	return start, nil
}

func (s *Sink) isClosing() bool {
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.lifetime.Stopping()
}

// deleteConsumerGroup destroys the consumer group of the sink for stream
// unless the consumers map of stream holds a consumer of the sink. The check
// and the destroy are one Redis script: the map update of RemoveStream and
// the destroy are separate steps, and another instance may add the stream
// between them.
func (s *Sink) deleteConsumerGroup(ctx context.Context, stream *Stream) error {
	record, raw, err := readSinkMembership(ctx, stream, s.Name)
	if err != nil {
		return err
	}
	if record == nil || len(record.Members) != 0 {
		return nil
	}
	keys := []string{consumersMapKey(stream), stream.key}
	destroyed, err := destroyGroupScript.Run(ctx, redisop.Client(s.rdb), keys, s.Name, raw).Int64()
	if err != nil {
		return fmt.Errorf("failed to destroy Redis consumer group %q for stream %q: %w", s.Name, stream.Name, err)
	}
	if destroyed == 0 {
		s.logger.Debug("consumer group in use by another instance or missing, not destroyed", "stream", stream.Name)
	}
	return nil
}

// rollbackConsumer removes consumer of the sink named sink from the consumers
// map cm of stream after err failed a later step of NewSink, AddStream or a
// consumer replacement, even if ctx is canceled. It returns err joined with
// the removal error, if any. The consumer group stays: another instance of
// the sink may use it (tla/README.md).
func rollbackConsumer(ctx context.Context, cm *sinkConsumerMap, sink, consumer string, filter sinkFilter, err error) error {
	if _, rerr := changeSinkMembership(context.WithoutCancel(ctx), cm, sink, filter, sinkMemberChange{remove: []string{consumer}}); rerr != nil {
		err = errors.Join(err, fmt.Errorf("failed to remove consumer %s from replicated map for stream %s: %w", consumer, cm.stream.Name, rerr))
	}
	return err
}

// isBusyGroupErr returns true if the error is a busy group error.
func isBusyGroupErr(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "BUSYGROUP")
}

// consumersMapName is the name of the replicated map that backs a sink.
func consumersMapName(stream *Stream) string {
	return fmt.Sprintf("stream:%s:sinks", stream.Name)
}

// consumersMapKey is the Redis hash key that holds the content of the
// replicated map named consumersMapName(stream), as rmap.Join names it.
func consumersMapKey(stream *Stream) string {
	return fmt.Sprintf("map:%s:content", consumersMapName(stream))
}

// staleLockName is the name of the lock used to check for stale messages.
func staleLockName(sink string) string {
	return fmt.Sprintf("sink:%s:stalelease", sink)
}
