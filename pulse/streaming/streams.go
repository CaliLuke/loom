package streaming

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/CaliLuke/loom/pulse/internal/keyttl"
	"github.com/CaliLuke/loom/pulse/pulse"
	"github.com/CaliLuke/loom/pulse/rmap"
	"github.com/CaliLuke/loom/pulse/streaming/options"
	redis "github.com/redis/go-redis/v9"
)

type (
	// Stream encapsulates a stream of events.  Events published to a stream
	// can optionally be associated with a topic.  Stream consumers can
	// subscribe to a stream and optionally provide a topic matching
	// criteria. Consumers can be created within a group. Each consumer
	// group receives a unique copy of the stream events.
	Stream struct {
		// Name of the stream.
		Name string
		// MaxLen is the maximum number of events in the stream.
		MaxLen int
		// ttl configures an expiry for the Redis key backing the stream.
		ttl time.Duration
		// ttlSliding controls whether ttl is refreshed on every Add call.
		ttlSliding bool
		// logger is the logger used by the stream.
		logger pulse.Logger
		// rootLogger is the prefix-free logger used to create sink loggers.
		rootLogger pulse.Logger
		// key is the redis key used for the stream.
		key string
		// rdb is the redis connection.
		rdb *redis.Client
	}
)

const (
	// streamKeyPrefix is the prefix used for stream keys.
	streamKeyPrefix = "pulse:stream:"
	// nameKey is the key used to store the event name.
	nameKey = "n"
	// payloadKey is the key used to store the event payload.
	payloadKey = "p"
	// topicKey is the key used to store the event topic.
	topicKey = "t"
)

// addEventScript adds an event to a stream and applies the stream TTL in the
// same script, so a caller never sees an error for an event that was added.
//
// KEYS[1] is the stream key. ARGV holds the MAXLEN (0 for none), "1" to not
// create a missing stream, the TTL arguments of keyttl.Args, then the event
// field/value pairs. The script returns the event id, or nil when the stream
// is missing and must not be created.
var addEventScript = redis.NewScript(keyttl.LuaApply + `
local args = {}
if ARGV[2] == "1" then
   table.insert(args, "NOMKSTREAM")
end
if tonumber(ARGV[1]) > 0 then
   table.insert(args, "MAXLEN")
   table.insert(args, "~")
   table.insert(args, ARGV[1])
end
table.insert(args, "*")
for i = 5, #ARGV do
   table.insert(args, ARGV[i])
end
local id = redis.call("XADD", KEYS[1], unpack(args))
if not id then
   return false
end
apply_ttl(KEYS[1], ARGV[3], ARGV[4])
return id
`)

// createGroupScript creates a consumer group, and the stream if missing, and
// applies the stream TTL in the same script. An existing group is kept.
//
// KEYS[1] is the stream key. ARGV holds the group name, the start id, then
// the TTL arguments of keyttl.Args. The script returns an error reply for
// any failure other than BUSYGROUP.
var createGroupScript = redis.NewScript(keyttl.LuaApply + `
local res = redis.pcall("XGROUP", "CREATE", KEYS[1], ARGV[1], ARGV[2], "MKSTREAM")
if type(res) == "table" and res.err and not string.find(res.err, "BUSYGROUP", 1, true) then
   return res
end
apply_ttl(KEYS[1], ARGV[3], ARGV[4])
return "OK"
`)

// NewStream returns the stream with the given name. All stream instances
// with the same name share the same events.
func NewStream(name string, rdb *redis.Client, opts ...options.Stream) (*Stream, error) {
	if !isValidRedisKeyName(name) {
		return nil, fmt.Errorf("pulse stream: not a valid name %q", name)
	}
	o := options.ParseStreamOptions(opts...)
	if o.TTL < 0 {
		return nil, fmt.Errorf("pulse stream: ttl must be >= 0")
	}
	var logger pulse.Logger
	if o.Logger != nil {
		logger = o.Logger.WithPrefix("stream", name)
	} else {
		logger = pulse.NoopLogger()
	}
	s := &Stream{
		Name:       name,
		MaxLen:     o.MaxLen,
		ttl:        o.TTL,
		ttlSliding: o.TTLSliding,
		logger:     logger,
		rootLogger: o.Logger,
		key:        streamKeyPrefix + name,
		rdb:        rdb,
	}
	return s, nil
}

// NewReader creates a new stream reader. All reader instances get all the
// events in the stream. Events are read starting:
//   - from the last event by default
//   - from the oldest event stored in the stream if the
//     WithReaderStartAtOldest option is used
//   - after the event with the ID provided via WithReaderLastEventID if the
//     event is still in the stream, oldest event otherwise
//   - from the event added on or after the timestamp provided via
//     WithReaderStartAt if still in the stream, oldest event otherwise
func (s *Stream) NewReader(ctx context.Context, opts ...options.Reader) (*Reader, error) {
	reader, err := newReader(ctx, s, opts...)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to create reader: %w", err))
		return nil, err
	}
	s.logger.Info("create reader", "start", reader.startID)
	return reader, nil
}

// NewSink creates a new stream sink with the given name. All sink instances
// with the same name share the same stream cursor. Events read through a sink
// are not removed from the stream until they are acked by the client unless the
// WithNoAck option is used. Events are read starting:
//   - from the last event by default
//   - from the oldest event stored in the stream if the WithSinkStartAtOldest
//     option is used
//   - after the event with the ID provided via WithSinkLastEventID if the
//     event is still in the stream, oldest event otherwise
//   - from the event added on or after the timestamp provided via
//     WithSinkStartAt if still in the stream, oldest event otherwise
func (s *Stream) NewSink(ctx context.Context, name string, opts ...options.Sink) (*Sink, error) {
	sink, err := newSink(ctx, name, s, sinkRuntime{idleCheckPeriod: defaultIdleCheckPeriod}, opts...)
	if err != nil {
		s.logger.Error(fmt.Errorf("failed to create sink: %w", err), "sink", name)
		return nil, err
	}
	return sink, nil
}

// Key returns the Redis key of the stream. Scripts that add events to the
// stream atomically with other writes use it.
func (s *Stream) Key() string {
	return s.key
}

// Add appends an event to the stream and returns its ID. If the option
// WithOnlyIfStreamExists is used and the stream does not exist then no event is
// added and the empty string is returned. The stream is created if the option
// is omitted or when NewSink is called.
func (s *Stream) Add(ctx context.Context, name string, payload []byte, opts ...options.AddEvent) (string, error) {
	o := options.ParseAddEventOptions(opts...)
	values := []any{nameKey, name, payloadKey, payload}
	if o.Topic != "" {
		values = append(values, topicKey, o.Topic)
	}
	res, err := s.addEvent(ctx, values, o.OnlyIfStreamExists)
	if err != nil {
		if errors.Is(err, redis.Nil) {
			// Stream does not exist and OnlyIfStreamExists option was used.
			return "", nil
		}
		err = fmt.Errorf("failed to add event: %w", err)
		s.logger.Error(err, "event", name)
		return "", err
	}
	s.logger.Info("add", "event", name, "id", res)
	return res, nil
}

// addEvent adds an event with the given field/value pairs and returns its
// id. With a TTL it runs addEventScript, so the event and the TTL are applied
// atomically; without one it sends XADD. It returns redis.Nil when
// onlyIfExists is set and the stream does not exist.
func (s *Stream) addEvent(ctx context.Context, values []any, onlyIfExists bool) (string, error) {
	if s.ttl <= 0 {
		return s.rdb.XAdd(ctx, &redis.XAddArgs{
			Stream:     s.key,
			Values:     values,
			MaxLen:     int64(s.MaxLen),
			Approx:     true,
			NoMkStream: onlyIfExists,
		}).Result()
	}
	noMkStream := "0"
	if onlyIfExists {
		noMkStream = "1"
	}
	ttlSeconds, ttlSliding := keyttl.Args(s.ttl, s.ttlSliding)
	args := append([]any{s.MaxLen, noMkStream, ttlSeconds, ttlSliding}, values...)
	return addEventScript.Run(ctx, s.rdb, []string{s.key}, args...).Text()
}

// createGroup creates the consumer group name starting after startID, and
// the stream if missing. An existing group is kept. With a TTL it runs
// createGroupScript, so the group and the TTL are applied atomically.
func (s *Stream) createGroup(ctx context.Context, name, startID string) error {
	if s.ttl <= 0 {
		if err := s.rdb.XGroupCreateMkStream(ctx, s.key, name, startID).Err(); err != nil && !isBusyGroupErr(err) {
			return err
		}
		return nil
	}
	ttlSeconds, ttlSliding := keyttl.Args(s.ttl, s.ttlSliding)
	return createGroupScript.Run(ctx, s.rdb, []string{s.key}, name, startID, ttlSeconds, ttlSliding).Err()
}

// Remove removes the events with the given IDs from the stream.
// Note: clients should not need to call this method in normal operation,
// instead they should use the Ack method to acknowledge events.
func (s *Stream) Remove(ctx context.Context, ids ...string) error {
	err := s.rdb.XDel(ctx, s.key, ids...).Err()
	if err != nil {
		err = fmt.Errorf("failed to remove events: %w", err)
		s.logger.Error(err, "events", ids)
		return err
	}
	s.logger.Debug("remove", "events", ids)
	return nil
}

// Destroy deletes the entire stream and all its messages.
func (s *Stream) Destroy(ctx context.Context) error {
	if err := s.rdb.Del(ctx, s.key).Err(); err != nil {
		err := fmt.Errorf("failed to destroy stream: %w", err)
		s.logger.Error(err)
		return err
	}
	if err := s.destroyConsumersMap(ctx); err != nil {
		err := fmt.Errorf("failed to destroy stream sink map: %w", err)
		s.logger.Error(err)
		return err
	}
	s.logger.Info("stream deleted")
	return nil
}

// destroyConsumersMap removes the per-stream sink membership map through the
// rmap destroy protocol so reconnecting replicas observe the same revisioned
// destroy semantics as every other replicated map.
func (s *Stream) destroyConsumersMap(ctx context.Context) error {
	mapName := consumersMapName(s)
	mapKey := fmt.Sprintf("map:%s:content", mapName)
	exists, err := s.rdb.Exists(ctx, mapKey).Result()
	if err != nil {
		return err
	}
	if exists == 0 {
		return nil
	}
	consumers, err := rmap.Join(ctx, mapName, s.rdb, consumersMapOptions(s, s.rootLogger)...)
	if err != nil {
		return err
	}
	defer consumers.Close()
	return consumers.Destroy(ctx)
}

// redisKeyRegex is a regular expression that matches valid Redis keys.
var redisKeyRegex = regexp.MustCompile(`^[^ \0\*\?\[\]]{1,512}$`)

func isValidRedisKeyName(key string) bool {
	return redisKeyRegex.MatchString(key)
}
