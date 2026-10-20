package streaming

import (
	"cmp"
	"context"
	"encoding/json/v2"
	"errors"
	"fmt"
	"regexp"
	"slices"
	"time"

	"github.com/CaliLuke/go-redis/v9"

	"github.com/CaliLuke/loom/pulse/internal/redisop"
	"github.com/CaliLuke/loom/pulse/rmap"
)

type (
	sinkConsumerMap struct {
		stream  *Stream
		replica *rmap.Map
	}

	sinkFilter struct {
		Kind  string `json:"kind"`
		Value string `json:"value"`
	}

	sinkMember struct {
		Name          string `json:"name"`
		RegisteredAt  int64  `json:"registeredAt"`
		LastKeepAlive int64  `json:"lastKeepAlive,omitempty"`
	}

	sinkMembership struct {
		Version int          `json:"version"`
		Filter  *sinkFilter  `json:"filter"`
		Members []sinkMember `json:"members"`
	}

	sinkMemberChange struct {
		add               string
		remove            []string
		keepAliveAt       int64
		retireIfUnchanged map[string]sinkMember
		// restore is authority from an already admitted active registration.
		restore bool
	}
)

func newSinkFilter(topic, pattern string) (sinkFilter, eventFilterFunc, error) {
	if topic != "" {
		return sinkFilter{Kind: "exact", Value: topic}, func(e *Event) bool {
			return e.Topic == topic
		}, nil
	}
	if pattern != "" {
		re, err := regexp.Compile(pattern)
		if err != nil {
			return sinkFilter{}, nil, fmt.Errorf("topic pattern must be a valid regex: %w", err)
		}
		return sinkFilter{Kind: "pattern", Value: pattern}, func(e *Event) bool {
			return re.MatchString(e.Topic)
		}, nil
	}
	return sinkFilter{Kind: "none"}, nil, nil
}

// decodeSinkMembership rejects pre-contract arrays and unknown records. It never
// guesses the filter of a group whose persisted identity cannot be established.
func decodeSinkMembership(raw string) (*sinkMembership, error) {
	var record sinkMembership
	if err := json.Unmarshal([]byte(raw), &record, json.RejectUnknownMembers(true)); err != nil {
		return nil, fmt.Errorf("sink membership requires coordinated migration: %w", err)
	}
	if record.Version != 2 || record.Filter == nil || record.Members == nil {
		return nil, fmt.Errorf("sink membership requires coordinated migration: invalid version, filter or members")
	}
	switch record.Filter.Kind {
	case "none":
		if record.Filter.Value != "" {
			return nil, fmt.Errorf("sink membership requires coordinated migration: invalid unfiltered identity")
		}
	case "exact", "pattern":
		if record.Filter.Value == "" {
			return nil, fmt.Errorf("sink membership requires coordinated migration: empty filter identity")
		}
	default:
		return nil, fmt.Errorf("sink membership requires coordinated migration: unknown filter kind")
	}
	seen := make(map[string]bool, len(record.Members))
	for _, member := range record.Members {
		if member.Name == "" || member.RegisteredAt <= 0 || member.LastKeepAlive < 0 || seen[member.Name] {
			return nil, fmt.Errorf("sink membership requires coordinated migration: invalid consumer names or registration timestamps")
		}
		seen[member.Name] = true
	}
	return &record, nil
}

// readSinkMembership reads Redis rather than a possibly stale local replica.
// All cooperating writers leave this map persistent. Checking expiry before
// publication therefore remains valid until CAS; concurrent legacy writers are
// excluded by the coordinated migration contract.
func readSinkMembership(ctx context.Context, stream *Stream, group string) (*sinkMembership, string, error) {
	key := consumersMapKey(stream)
	ttl, err := redisop.Client(stream.rdb).PTTL(ctx, key).Result()
	if err != nil {
		return nil, "", fmt.Errorf("read sink membership lifetime: %w", err)
	}
	if ttl >= 0 {
		return nil, "", fmt.Errorf("sink membership requires coordinated migration: expiring membership map")
	}
	raw, err := redisop.Client(stream.rdb).HGet(ctx, key, group).Result()
	if errors.Is(err, redis.Nil) {
		return nil, "", nil
	}
	if err != nil {
		return nil, "", fmt.Errorf("read sink membership: %w", err)
	}
	record, err := decodeSinkMembership(raw)
	return record, raw, err
}

// changeSinkMembership compares and publishes filter identity and membership as
// one revisioned map value. Empty membership retains identity until Stream.Destroy.
func changeSinkMembership(ctx context.Context, cm *sinkConsumerMap, group string, filter sinkFilter, change sinkMemberChange) ([]string, error) {
	stream := cm.stream
	ctx, cancel := redisop.Context(ctx, stream.rdb)
	defer cancel()
	for {
		record, raw, missing, err := prepareSinkMembership(ctx, stream, group, filter, change)
		if err != nil {
			return nil, err
		}
		if record == nil {
			return nil, nil
		}
		if sinkRetirementChanged(record, change.retireIfUnchanged) {
			return record.memberNames(), nil
		}
		applySinkMemberChange(record, change)
		updated, err := publishSinkMembership(ctx, cm, group, record, raw, missing)
		if err != nil {
			return nil, err
		}
		if updated {
			return record.memberNames(), nil
		}
	}
}

func prepareSinkMembership(ctx context.Context, stream *Stream, group string, filter sinkFilter, change sinkMemberChange) (*sinkMembership, string, bool, error) {
	record, raw, err := readSinkMembership(ctx, stream, group)
	if err != nil {
		return nil, "", false, err
	}
	if record == nil {
		if change.add == "" {
			return nil, "", true, nil
		}
		if !change.restore {
			if err := checkSinkGroupIdentity(ctx, stream, group); err != nil {
				return nil, "", false, err
			}
		}
		return &sinkMembership{Version: 2, Filter: &filter, Members: []sinkMember{}}, raw, true, nil
	}
	if *record.Filter != filter {
		return nil, "", false, fmt.Errorf("sink group %q filter conflicts with its persisted identity", group)
	}
	return record, raw, false, nil
}

func sinkRetirementChanged(record *sinkMembership, expectedMembers map[string]sinkMember) bool {
	for name, expected := range expectedMembers {
		for _, member := range record.Members {
			if member.Name == name && member != expected {
				return true
			}
		}
	}
	return false
}

func applySinkMemberChange(record *sinkMembership, change sinkMemberChange) {
	record.Members = slices.DeleteFunc(record.Members, func(member sinkMember) bool {
		return slices.Contains(change.remove, member.Name)
	})
	if change.add != "" {
		found := false
		for i := range record.Members {
			if record.Members[i].Name != change.add {
				continue
			}
			found = true
			if change.keepAliveAt > record.Members[i].LastKeepAlive {
				record.Members[i].LastKeepAlive = change.keepAliveAt
			}
			break
		}
		if !found {
			record.Members = append(record.Members, sinkMember{
				Name:          change.add,
				RegisteredAt:  time.Now().UnixNano(),
				LastKeepAlive: change.keepAliveAt,
			})
		}
	}
	slices.SortFunc(record.Members, func(a, b sinkMember) int {
		return cmp.Compare(a.Name, b.Name)
	})
}

func publishSinkMembership(ctx context.Context, cm *sinkConsumerMap, group string, record *sinkMembership, raw string, missing bool) (bool, error) {
	encoded, err := json.Marshal(record)
	if err != nil {
		return false, fmt.Errorf("encode sink membership: %w", err)
	}
	if missing {
		return cm.replica.SetIfNotExists(ctx, group, string(encoded))
	}
	_, _, updated, err := cm.replica.TestAndSetEx(ctx, group, raw, string(encoded))
	return updated, err
}

// checkSinkGroupIdentity rejects pre-existing groups without the authoritative
// filter record, while retaining the concurrent supported-creator recheck.
func checkSinkGroupIdentity(ctx context.Context, stream *Stream, group string) error {
	groups, err := redisop.Client(stream.rdb).XInfoGroups(ctx, stream.key).Result()
	if err != nil && !redis.HasErrorPrefix(err, "no such key") {
		return fmt.Errorf("inspect group identity: %w", err)
	}
	for _, existing := range groups {
		if existing.Name != group {
			continue
		}
		// A supported concurrent creator publishes membership first.
		// Re-read before classifying an unknown pre-existing group.
		current, _, err := readSinkMembership(ctx, stream, group)
		if err != nil {
			return err
		}
		if current == nil {
			return fmt.Errorf("sink group %q requires coordinated migration: existing group has no filter identity", group)
		}
	}
	return nil
}

// memberNames exposes the set identity without discarding registration age in storage.
func (m *sinkMembership) memberNames() []string {
	names := make([]string, len(m.Members))
	for i, member := range m.Members {
		names[i] = member.Name
	}
	return names
}
