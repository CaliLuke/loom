package streaming

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	redis "github.com/redis/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
)

// deleteStreamStaleConsumers deletes stale consumers for a specific stream.
// s.lock must be held.
func (s *Sink) deleteStreamStaleConsumers(ctx context.Context, stream *Stream) error {
	// Get all consumers for this group
	consumers, err := s.rdb.XInfoConsumers(ctx, stream.key, s.Name).Result()
	if err != nil {
		return fmt.Errorf("failed to get consumers info: %w", err)
	}

	// Check keep-alive map
	keepAlives := s.consumersKeepAliveMap.Map()
	for _, consumer := range consumers {
		ts, hasKeepAlive := keepAlives[consumer.Name]
		if !hasKeepAlive {
			s.logger.Info("cleaning up consumer with no keep-alive", "consumer", consumer.Name)
			if err := s.rdb.XGroupDelConsumer(ctx, stream.key, s.Name, consumer.Name).Err(); err != nil {
				s.logger.Error(fmt.Errorf("failed to delete consumer with no keep-alive: %w", err), "consumer", consumer.Name)
			}
			continue
		}

		// Check if consumer is stale based on keep-alive timestamp
		keepAliveTs, err := strconv.ParseInt(ts, 10, 64)
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to parse keep-alive timestamp: %w", err), "consumer", consumer.Name, "timestamp", ts)
			continue
		}

		if time.Since(time.Unix(0, keepAliveTs)) > 2*s.ackGracePeriod {
			s.logger.Info("cleaning up stale consumer", "consumer", consumer.Name)
			if err := s.rdb.XGroupDelConsumer(ctx, stream.key, s.Name, consumer.Name).Err(); err != nil {
				s.logger.Error(fmt.Errorf("failed to delete stale consumer: %w", err), "consumer", consumer.Name)
			}
			if _, err := s.consumersKeepAliveMap.Delete(ctx, consumer.Name); err != nil {
				s.logger.Error(fmt.Errorf("failed to delete keep-alive for stale consumer: %w", err), "consumer", consumer.Name)
			}
			if sinks := s.consumersMap[stream.Name]; sinks != nil {
				if _, _, err := sinks.RemoveValues(ctx, s.Name, consumer.Name); err != nil {
					s.logger.Error(fmt.Errorf("failed to remove consumer from map: %w", err), "stream", stream.Name, "consumer", consumer.Name)
				}
			}
		}
	}
	return nil
}

// deleteStaleConsumers deletes stale consumers.
// s.lock must be held.
func (s *Sink) deleteStaleConsumers(ctx context.Context) {
	for _, stream := range s.streams {
		if err := s.deleteStreamStaleConsumers(ctx, stream); err != nil {
			s.logger.Error(fmt.Errorf("failed to delete stale consumers for stream %s: %w", stream.Name, err))
		}
	}
}

// removeStreamConsumer removes the stream consumer from the sink.
func (s *Sink) removeStreamConsumer(ctx context.Context, stream *Stream) error {
	remains, _, err := s.consumersMap[stream.Name].RemoveValues(ctx, s.Name, s.consumer)
	if err != nil {
		return fmt.Errorf("failed to remove consumer %s from replicated map for stream %s: %w", s.consumer, stream.Name, err)
	}
	if len(remains) == 0 {
		if err := s.deleteConsumerGroup(ctx, stream); err != nil {
			return err
		}
	}
	return nil
}

// newConsumer creates a new consumer and registers it in the consumers and
// keep-alive maps.
func (s *Sink) newConsumer(ctx context.Context, stream *Stream) (string, error) {
	consumer := ulid.Make().String()
	if err := stream.rdb.XGroupCreateConsumer(ctx, stream.key, s.Name, consumer).Err(); err != nil {
		return "", fmt.Errorf("failed to create Redis consumer %s for consumer group %s: %w", consumer, s.Name, err)
	}
	if _, err := s.consumersMap[stream.Name].AppendValues(ctx, s.Name, consumer); err != nil {
		if err := stream.rdb.XGroupDelConsumer(ctx, stream.key, s.Name, consumer).Err(); err != nil {
			s.logger.Error(fmt.Errorf("failed to delete consumer %s after failed append: %w", consumer, err))
		}
		return "", fmt.Errorf("failed to append store consumer %s for sink %s: %w", consumer, s.Name, err)
	}
	s.lastKeepAlive = time.Now().UnixNano()
	if _, err := s.consumersKeepAliveMap.Set(ctx, consumer, strconv.FormatInt(s.lastKeepAlive, 10)); err != nil {
		if err := stream.rdb.XGroupDelConsumer(ctx, stream.key, s.Name, consumer).Err(); err != nil {
			s.logger.Error(fmt.Errorf("failed to delete consumer %s after failed keep-alive set: %w", consumer, err))
		}
		return "", fmt.Errorf("failed to set sink keep-alive for new consumer %s: %w", consumer, err)
	}
	return consumer, nil
}

// read reads events from the streams and sends them to the sink channel.
func (s *Sink) read(ctx context.Context) {
	defer s.logger.Debug("read: exiting")
	defer s.wait.Done()
	for {
		if err := s.ensureConsumer(ctx); err != nil {
			time.Sleep(time.Duration(rand.Int63n(int64(s.blockDuration))))
			continue
		}
		s.lock.Lock()
		readStreams := make([]string, len(s.streamCursors))
		copy(readStreams, s.streamCursors)
		s.lock.Unlock()

		s.logger.Debug("reading", "streams", readStreams, "max", s.maxPolled, "block", s.blockDuration)
		streams, err := s.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
			Group:    s.Name,
			Consumer: s.consumer,
			Streams:  readStreams,
			Count:    s.maxPolled,
			Block:    s.blockDuration,
			NoAck:    s.noAck,
		}).Result()

		s.lock.Lock()
		if s.closing {
			s.lock.Unlock()
			// Honor the Close contract and do not forward any more events.
			// Any events in the PEL will be claimed by another consumer.
			return
		}
		if err != nil {
			if err := handleReadError(err, s.logger); err != nil {
				s.logger.Error(fmt.Errorf("error reading events: %w", err))
			}
			s.lock.Unlock()
			continue
		}
		for _, events := range streams {
			streamName := events.Stream[len(streamKeyPrefix):]
			streamEvents(streamName, events.Stream, s.Name, events.Messages, s.eventFilter, s.chans, s.rdb, s.logger)
		}
		s.lock.Unlock()
	}
}

// ensureConsumer ensures that the consumer is still alive.
func (s *Sink) ensureConsumer(ctx context.Context) error {
	s.lock.Lock()
	defer s.lock.Unlock()
	if time.Since(time.Unix(0, s.lastKeepAlive)) > 2*s.ackGracePeriod {
		s.logger.Debug("consumer stale, creating new one")
		var err error
		s.consumer, err = s.newConsumer(ctx, s.streams[0])
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to create new consumer: %w", err))
			return err
		}
	}
	return nil
}

// periodicKeepAlive updates this consumer keep-alive every half ack grace period.
func (s *Sink) periodicKeepAlive() {
	defer s.wait.Done()
	defer s.logger.Debug("periodicKeepAlive: exiting")
	ticker := time.NewTicker(s.ackGracePeriod)
	defer ticker.Stop()

	ctx := context.Background()
	for {
		select {
		case <-ticker.C:
			s.lock.Lock()
			now := time.Now().UnixNano()
			if _, err := s.consumersKeepAliveMap.Set(ctx, s.consumer, strconv.FormatInt(now, 10)); err != nil {
				s.logger.Error(fmt.Errorf("failed to update sink keep-alive: %w", err))
				s.lock.Unlock()
				continue
			}
			s.lastKeepAlive = now
			s.lock.Unlock()

		case <-s.donechan:
			return
		}
	}
}

// periodicIdleMessageCheck claims any idle message every check stale period.
// An idle message is one that has not been acked for more than the ack grace period.
// Once all idle messages are claimed, any stale consumer is deleted.
func (s *Sink) periodicIdleMessageCheck() {
	defer s.wait.Done()
	defer s.logger.Debug("periodicIdleMessageCheck: exiting")
	ticker := time.NewTicker(checkIdlePeriod)
	defer ticker.Stop()

	leaseDuration := checkIdlePeriod.Milliseconds() - 5
	ctx := context.Background()
	for {
		select {
		case <-ticker.C:
			now := time.Now().UnixNano() / int64(time.Millisecond)
			newExpiration := now + leaseDuration
			result, err := s.acquireLease.EvalSha(ctx, s.rdb, s.leaseKeyName, newExpiration, now, leaseDuration).Result()
			if err != nil {
				s.logger.Error(fmt.Errorf("failed to acquire idle message check lease: %w", err))
				continue
			}
			if result != int64(1) {
				// Another sink instance claimed the lease.
				continue
			}
			s.lock.Lock()
			s.claimIdleMessages(ctx)
			s.deleteStaleConsumers(ctx)
			s.lock.Unlock()

		case <-s.donechan:
			return
		}
	}
}

// claimIdleMessages claims idle messages from the streams.
// s.lock must be held.
func (s *Sink) claimIdleMessages(ctx context.Context) {
	for _, stream := range s.streams {
		args := redis.XAutoClaimArgs{
			Stream:   stream.key,
			Group:    s.Name,
			MinIdle:  s.ackGracePeriod,
			Start:    "0-0",
			Consumer: s.consumer,
		}
		start, err := s.claim(ctx, stream.Name, args)
		if err != nil {
			s.logger.Error(fmt.Errorf("failed to claim idle messages for stream %s: %w", stream.Name, err))
			continue
		}
		for start != "0-0" {
			args.Start = start
			start, err = s.claim(ctx, stream.Name, args)
			if err != nil {
				s.logger.Error(fmt.Errorf("failed to claim idle messages for stream %s: %w", stream.Name, err))
				break
			}
		}
	}
}

// Helper function to claim messages from a stream used by claimIdleMessages.
func (s *Sink) claim(ctx context.Context, streamName string, args redis.XAutoClaimArgs) (string, error) {
	messages, start, err := s.rdb.XAutoClaim(ctx, &args).Result()
	if len(messages) > 0 {
		s.logger.Info("claimed", "stream", streamName, "messages", len(messages))
		streamEvents(streamName, args.Stream, s.Name, messages, s.eventFilter, s.chans, s.rdb, s.logger)
	}
	return start, err
}

// deleteConsumerGroup deletes the consumer group.
func (s *Sink) deleteConsumerGroup(ctx context.Context, stream *Stream) error {
	if err := s.rdb.XGroupDestroy(ctx, stream.key, s.Name).Err(); err != nil {
		return fmt.Errorf("failed to destroy Redis consumer group %q for stream %q: %w", s.Name, stream.Name, err)
	}
	delete(s.consumersMap, stream.Name)
	return nil
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

func consumersMapOptions(stream *Stream, logger pulse.Logger) []rmap.MapOption {
	opts := []rmap.MapOption{
		rmap.WithLogger(logger),
	}
	if stream.ttl > 0 {
		if stream.ttlSliding {
			opts = append(opts, rmap.WithSlidingTTL(stream.ttl))
		} else {
			opts = append(opts, rmap.WithTTL(stream.ttl))
		}
	}
	return opts
}

// sinkKeepAliveMapName is the name of the replicated map that backs a sink keep-alives.
func sinkKeepAliveMapName(sink string) string {
	return fmt.Sprintf("sink:%s:keepalive", sink)
}

// staleLockName is the name of the lock used to check for stale messages.
func staleLockName(sink string) string {
	return fmt.Sprintf("sink:%s:stalelease", sink)
}
