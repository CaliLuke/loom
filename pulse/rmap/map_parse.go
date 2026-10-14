package rmap

import (
	"context"
	"encoding/binary"
	"encoding/json/v2"
	"fmt"
	"strconv"
	"strings"
)

// unpackString reads a length-prefixed string from a buffer using struct.pack
// format "ic0"
func unpackString(data []byte) (string, []byte, error) {
	if len(data) < 4 {
		return "", nil, fmt.Errorf("buffer too short for length")
	}
	length := int(binary.LittleEndian.Uint32(data))
	data = data[4:]
	if len(data) < length {
		return "", nil, fmt.Errorf("buffer too short for string")
	}
	return string(data[:length]), data[length:], nil
}

func parseValues(value string) []string {
	trimmed := strings.TrimSpace(value)
	if strings.HasPrefix(trimmed, "[") {
		var decoded []string
		if err := json.Unmarshal([]byte(trimmed), &decoded); err == nil {
			return decoded
		}
	}
	return strings.Split(value, ",")
}

func parseRevisionString(s string) (uint64, bool) {
	rev, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return rev, true
}

func extractState(content map[string]string) (replicaState, error) {
	state := replicaState{content: content}
	if len(content) == 0 {
		return state, nil
	}
	if content[protocolField] != mapProtocol || content[generationField] == "" {
		return replicaState{}, fmt.Errorf("invalid replicated-map protocol metadata")
	}
	state.generation = content[generationField]
	for field, target := range map[string]*uint64{revField: &state.rev, resetField: &state.reset, deleteField: &state.deleted} {
		value, ok := parseRevisionString(content[field])
		if !ok {
			return replicaState{}, fmt.Errorf("invalid replicated-map metadata %s", field)
		}
		*target = value
	}
	if state.reset > state.rev || state.deleted > state.rev {
		return replicaState{}, fmt.Errorf("invalid replicated-map recovery revisions")
	}
	for key := range content {
		switch key {
		case protocolField, generationField, revField, resetField, deleteField:
			delete(content, key)
		default:
			if strings.Contains(key, "=") {
				return replicaState{}, fmt.Errorf("unknown replicated-map metadata %q", key)
			}
		}
	}
	return state, nil
}

// runSetScript executes the Set Lua script and returns the previous value plus
// the committed revision observed by Redis.
func (sm *Map) runSetScript(ctx context.Context, key, value string) (setResult, error) {
	raw, err := sm.runLuaScript(ctx, "set", sm.setScript, key, value)
	if err != nil {
		return setResult{}, err
	}
	values, ok := raw.([]any)
	if !ok || len(values) != 4 {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set result %T", sm.Name, raw)
	}
	existed, ok := values[0].(int64)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set existence flag %T", sm.Name, values[0])
	}
	prev, ok := values[1].(string)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set previous value %T", sm.Name, values[1])
	}
	revString, ok := values[2].(string)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set revision %T", sm.Name, values[2])
	}
	rev, ok := parseRevisionString(revString)
	if !ok {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set revision %q", sm.Name, revString)
	}
	generation, ok := values[3].(string)
	if !ok || generation == "" {
		return setResult{}, fmt.Errorf("pulse map: %s returned invalid set generation", sm.Name)
	}
	return setResult{prev: prev, existed: existed == 1, rev: rev, generation: generation}, nil
}

// notifySubscribers retains the strongest unread kind for each subscriber.
// The map lock serializes publishers and subscription removal. Receivers may
// consume the pending kind before the take, but cannot fill the emptied slot.
func (sm *Map) notifySubscribers(kind EventKind) {
	for _, c := range sm.chans {
		pending := kind
		select {
		case previous := <-c:
			pending = strongerEventKind(previous, pending)
		default:
		}
		c <- pending
	}
}

// applyMessageLocked applies only the next mutation in the same incarnation.
// Gaps and incarnation changes request a complete snapshot from the sole owner.
func (sm *Map) applyMessageLocked(op string, data []byte) (mapChange, bool, error) {
	var fields []string
	for len(data) > 0 {
		field, rest, err := unpackString(data)
		if err != nil {
			return mapChange{}, false, fmt.Errorf("invalid map envelope: %w", err)
		}
		fields = append(fields, field)
		data = rest
		if len(fields) > 5 {
			return mapChange{}, false, fmt.Errorf("invalid map envelope length")
		}
	}
	want := 3
	switch op {
	case "set":
		want = 5
	case "del":
		want = 4
	case "reset", "destroy":
	default:
		return mapChange{}, false, fmt.Errorf("invalid map operation %q", op)
	}
	if len(fields) != want || fields[0] != mapProtocol || fields[1] == "" {
		return mapChange{}, false, fmt.Errorf("invalid map protocol envelope")
	}
	rev, ok := parseRevisionString(fields[2])
	if !ok || rev == 0 {
		return mapChange{}, false, fmt.Errorf("invalid map envelope revision")
	}
	if want > 3 && (fields[3] == "" || strings.Contains(fields[3], "=")) {
		return mapChange{}, false, fmt.Errorf("invalid map envelope key")
	}
	if fields[1] != sm.generation {
		return mapChange{refresh: true}, false, nil
	}
	if rev <= sm.rev {
		return mapChange{}, false, nil
	}
	if rev != sm.rev+1 {
		return mapChange{refresh: true}, false, nil
	}
	switch op {
	case "set":
		sm.content[fields[3]] = fields[4]
	case "del":
		delete(sm.content, fields[3])
	case "reset", "destroy":
		sm.content = make(map[string]string)
	}
	sm.rev = rev
	return mapChange{kind: parseStoredKind(op), rev: rev}, true, nil
}

func parseStoredKind(kind string) EventKind {
	switch kind {
	case "del":
		return EventDelete
	case "destroy", "reset":
		return EventReset
	default:
		return EventChange
	}
}

func strongerEventKind(current, next EventKind) EventKind {
	if eventPriority(next) > eventPriority(current) {
		return next
	}
	return current
}

func eventPriority(kind EventKind) int {
	switch kind {
	case EventReset:
		return 3
	case EventDelete:
		return 2
	case EventChange:
		return 1
	default:
		return 0
	}
}
