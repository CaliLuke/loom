package pool

import (
	"bytes"
	"context"
	"fmt"
	"strconv"
	"strings"

	redis "github.com/CaliLuke/go-redis/v9"
)

// poolOperation identifies one admitted operation independently of its delivery
// and of a worker's ownership epoch. Tokens are exact decimal Redis integers.
type poolOperation struct {
	key      string
	token    string
	producer string
	revision string
	data     []byte
}

// luaOperation defines the shared protocol wire and exact token operations.
const luaOperation = `
local function token_after(a, b)
   if not a or a == "" then return false end
   if not b or b == "" then return true end
   if #a ~= #b then return #a > #b end
   return a > b
end
local function next_token(epochs, key)
   redis.call("HINCRBY", epochs, key, 1)
   return redis.call("HGET", epochs, key)
end
local function operation(key, token, producer, revision, data)
   return struct.pack("ic0ic0ic0ic0ic0", #key, key, #token, token,
      #producer, producer, #revision, revision, #data, data)
end
local function add_stop(stream, max_len, data)
   if tonumber(max_len) > 0 then
      return redis.call("XADD", stream, "MAXLEN", "~", max_len, "*", "n", "s", "p", data)
   end
   return redis.call("XADD", stream, "*", "n", "s", "p", data)
end
`

var (
	// KEYS: epochs, cutoffs, stream. ARGV: key, max length.
	luaStopJob = redis.NewScript(luaOperation + `
local token = next_token(KEYS[1], ARGV[1])
redis.call("HSET", KEYS[2], ARGV[1], token)
local data = operation(ARGV[1], token, "", "", "")
return {token, add_stop(KEYS[3], ARGV[2], data)}
`)
	// KEYS: owners, incarnations, cutoffs, payload content/channel.
	// ARGV: key, cutoff. Absence is safe only behind a durable stop fence.
	luaConfirmStop = redis.NewScript(luaOwnerMaps + luaOperation + `
local cutoff = redis.call("HGET", KEYS[3], ARGV[1])
if token_after(ARGV[2], cutoff) then return 0 end
local incarnation = redis.call("HGET", KEYS[2], ARGV[1])
if token_after(incarnation, ARGV[2]) then return 1 end
if redis.call("HEXISTS", KEYS[1], ARGV[1]) == 1 then return 0 end
del_map(KEYS[4], KEYS[5], ARGV[1])
return 1
`)
)

func unmarshalOperation(data []byte) (*poolOperation, error) {
	reader := bytes.NewReader(data)
	fields := make([][]byte, 5)
	for i, name := range []string{"job key", "operation token", "producer", "revision", "operation data"} {
		value, err := readField(reader, name)
		if err != nil {
			return nil, err
		}
		fields[i] = value
	}
	token := string(fields[1])
	value, err := strconv.ParseUint(token, 10, 64)
	if err != nil || value == 0 || strconv.FormatUint(value, 10) != token || reader.Len() != 0 {
		return nil, fmt.Errorf("%w: invalid operation token or trailing data", errMalformedMessage)
	}
	return &poolOperation{
		key:      string(fields[0]),
		token:    token,
		producer: string(fields[2]),
		revision: string(fields[3]),
		data:     fields[4],
	}, nil
}

func tokenAfter(a, b string) bool {
	if a == "" {
		return false
	}
	if len(a) != len(b) {
		return len(a) > len(b)
	}
	return a > b
}

func validateJobKey(key string) error {
	if key == "" {
		return fmt.Errorf("job key cannot be empty")
	}
	if strings.Contains(key, "=") {
		return fmt.Errorf("job key cannot contain '=': %q", key)
	}
	return nil
}

func (node *Node) issueStop(ctx context.Context, key string) (*poolOperation, error) {
	if err := validateJobKey(key); err != nil {
		return nil, err
	}
	values, err := luaStopJob.Run(ctx, node.rdb, []string{
		node.PoolName + ":owner-epochs", node.PoolName + ":job-cutoffs", node.poolStream.Key(),
	}, key, node.poolStream.MaxLen).StringSlice()
	if err != nil {
		return nil, fmt.Errorf("stop publication: %w", err)
	}
	if len(values) != 2 {
		return nil, fmt.Errorf("stop publication: invalid receipt %v", values)
	}
	return &poolOperation{key: key, token: values[0]}, nil
}

func (node *Node) confirmStop(ctx context.Context, op *poolOperation) error {
	confirmed, err := luaConfirmStop.Run(ctx, node.rdb, []string{
		node.ownersKey(), node.PoolName + ":job-incarnations", node.PoolName + ":job-cutoffs",
		rmapContentKey(jobPayloadMapName(node.PoolName)), rmapUpdateChannel(jobPayloadMapName(node.PoolName)),
	}, op.key, op.token).Bool()
	if err != nil {
		return err
	}
	if !confirmed {
		return ErrRequeue
	}
	return nil
}
