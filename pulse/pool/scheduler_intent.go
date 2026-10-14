package pool

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	redis "github.com/CaliLuke/go-redis/v9"
)

type (
	// schedulerIntent retains a requested operation and its immutable first
	// issuance receipt until completion. Complete starts remain responsibility.
	schedulerIntent struct {
		Revision  string `json:"revision"`
		Kind      string `json:"kind"`
		Payload   []byte `json:"payload"`
		CreatedAt string `json:"created_at"`
		Token     string `json:"token"`
		EventID   string `json:"event_id"`
		Complete  bool   `json:"complete"`
	}

	intentPlan struct {
		Start   []intentStart `json:"start"`
		Stop    []string      `json:"stop"`
		StopAll bool          `json:"stop_all"`
		Created string        `json:"created"`
	}

	intentStart struct {
		Key     string `json:"key"`
		Payload []byte `json:"payload"`
	}
)

var (
	// KEYS: producer content/channel, persistent generation. ARGV: plan JSON.
	// The generation key is deliberately independent of rmap Reset/Destroy.
	luaPublishIntent = redis.NewScript(luaOwnerMaps + `
local plan = cjson.decode(ARGV[1])
local starts = {}
local function publish(key, kind, payload)
   local old = redis.call("HGET", KEYS[1], key)
   if old then
      local record = cjson.decode(old)
      -- Repeated pending stops retain their cutoff. An authored Start is a new
      -- command even with identical payload; only reconciliation retries it.
      if kind == "s" and record.kind == "s" then return end
   end
   redis.call("INCR", KEYS[3])
   local revision = redis.call("GET", KEYS[3])
   set_map(KEYS[1], KEYS[2], key, cjson.encode({revision=revision, kind=kind,
      payload=payload, created_at=plan.created, token="", event_id="", complete=false}))
end
for _, job in ipairs(plan.start) do
   starts[job.key] = true
   publish(job.key, "j", job.payload)
end
if plan.stop_all then
   for _, key in ipairs(redis.call("HKEYS", KEYS[1])) do
      if string.sub(key, 1, 1) ~= "=" and not starts[key] then publish(key, "s", "") end
   end
else
   for _, key in ipairs(plan.stop) do publish(key, "s", "") end
end
return 1
`)
	// KEYS: ledger content/channel, epochs, incarnations, cutoffs, payloads,
	// owners, pending content/channel, pool stream.
	// ARGV: key, revision, producer, job bytes, now, until, max length, group,
	// now milliseconds, maximum event age.
	// Revision check, first token, receipt and queue publication share this
	// linearization boundary. A retry may replace delivery, never operation.
	luaIssueIntent = redis.NewScript(luaOwnerMaps + luaDispatchGuard + luaOperation + `
local raw = redis.call("HGET", KEYS[1], ARGV[1])
if not raw then return {0, ""} end
local record = cjson.decode(raw)
if record.revision ~= ARGV[2] or record.complete then return {0, ""} end
local function save()
   set_map(KEYS[1], KEYS[2], ARGV[1], cjson.encode(record))
end
local current = redis.call("HGET", KEYS[4], ARGV[1])
local cutoff = redis.call("HGET", KEYS[5], ARGV[1])
if record.kind == "j" then
   if record.token ~= "" and (record.token ~= current or not token_after(record.token, cutoff)) then
      record.complete = true
      save()
      return {0, ""}
   end
   if current and not token_after(current, cutoff) then
      delete_guard(KEYS[8], KEYS[9], ARGV[1])
   end
   local status, value = admission_status(KEYS[6], KEYS[7], KEYS[8], KEYS[10], ARGV[1], ARGV[5], ARGV[8], tonumber(ARGV[9]), tonumber(ARGV[10]))
   if status == 3 and token_after(current, cutoff) then
      record.token = current
      record.complete = true
      save()
      return {0, ""}
   end
   if status ~= 1 then return {status, value} end
   if record.token == "" then
      record.token = next_token(KEYS[3], ARGV[1])
      redis.call("HSET", KEYS[4], ARGV[1], record.token)
   end
   local data = operation(ARGV[1], record.token, ARGV[3], record.revision, ARGV[4])
   local guard = add_start(KEYS[10], ARGV[7], data, KEYS[8], KEYS[9], ARGV[1], ARGV[6])
   local _, id = parse_guard(guard)
   record.event_id = id
   save()
   return {1, guard}
end
if record.token == "" then
   record.token = next_token(KEYS[3], ARGV[1])
end
if token_after(record.token, cutoff) then
   redis.call("HSET", KEYS[5], ARGV[1], record.token)
end
if record.event_id ~= "" and in_flight(KEYS[10], ARGV[8], record.event_id, tonumber(ARGV[9]), tonumber(ARGV[10])) then
   save()
   return {0, ""}
end
local data = operation(ARGV[1], record.token, ARGV[3], record.revision, "")
record.event_id = add_stop(KEYS[10], ARGV[7], data)
save()
return {0, ""}
`)
	// KEYS: producer content/channel. ARGV: key, revision, token, event kind.
	luaCompleteIntent = redis.NewScript(luaOwnerMaps + `
local raw = redis.call("HGET", KEYS[1], ARGV[1])
if not raw then return 0 end
local record = cjson.decode(raw)
if record.revision ~= ARGV[2] or record.token ~= ARGV[3] or record.kind ~= ARGV[4] then return 0 end
if record.kind == "s" then del_map(KEYS[1], KEYS[2], ARGV[1])
else
   record.complete = true
   set_map(KEYS[1], KEYS[2], ARGV[1], cjson.encode(record))
end
return 1
`)
)

func (sched *scheduler) applyPlan(ctx context.Context, plan *JobPlan) {
	if err := sched.publishIntent(ctx, plan); err != nil {
		sched.logger.Error(fmt.Errorf("publish producer intent: %w", err))
		return
	}
	if err := sched.reconcileIntent(ctx); err != nil {
		sched.logger.Error(fmt.Errorf("reconcile producer intent: %w", err))
	}
}

func (sched *scheduler) publishIntent(ctx context.Context, plan *JobPlan) error {
	wire := intentPlan{
		Start:   make([]intentStart, 0, len(plan.Start)),
		Stop:    []string{},
		StopAll: plan.StopAll,
		Created: strconv.FormatInt(time.Now().UnixNano(), 10),
	}
	if !plan.StopAll {
		wire.Stop = append(wire.Stop, plan.Stop...)
	}
	for _, job := range plan.Start {
		if job == nil {
			return fmt.Errorf("nil job in producer plan")
		}
		if err := validateJobKey(job.Key); err != nil {
			return err
		}
		wire.Start = append(wire.Start, intentStart{Key: job.Key, Payload: job.Payload})
	}
	for _, key := range wire.Stop {
		if err := validateJobKey(key); err != nil {
			return err
		}
	}
	data, err := json.Marshal(wire)
	if err != nil {
		return err
	}
	return luaPublishIntent.Run(ctx, sched.node.rdb, []string{
		rmapContentKey(sched.name), rmapUpdateChannel(sched.name), sched.name + ":intent-generation",
	}, data).Err()
}

func (sched *scheduler) reconcileIntent(ctx context.Context) error {
	records, err := sched.node.rdb.HGetAll(ctx, rmapContentKey(sched.name)).Result()
	if err != nil {
		return err
	}
	keys := make([]string, 0, len(records))
	for key := range records {
		if !strings.HasPrefix(key, "=") {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	for _, key := range keys {
		var intent schedulerIntent
		if err := json.Unmarshal([]byte(records[key]), &intent); err != nil {
			return fmt.Errorf("read intent %q: %w", key, err)
		}
		if intent.Complete {
			continue
		}
		guard, err := sched.issueIntent(ctx, key, &intent)
		if err != nil {
			return err
		}
		if guard != "" {
			_, id, err := parsePendingGuard(guard)
			if err != nil {
				return err
			}
			if err := sched.node.awaitDispatch(ctx, key, guard, id); err != nil {
				sched.logger.Error(err, "job", key)
			}
		}
	}
	return nil
}

func (sched *scheduler) issueIntent(ctx context.Context, key string, intent *schedulerIntent) (string, error) {
	now := time.Now()
	created, err := strconv.ParseInt(intent.CreatedAt, 10, 64)
	if err != nil {
		return "", err
	}
	job := marshalJob(&Job{Key: key, Payload: intent.Payload, NodeID: sched.node.ID, CreatedAt: time.Unix(0, created)})
	node := sched.node
	raw, err := luaIssueIntent.Run(ctx, node.rdb, []string{
		rmapContentKey(sched.name), rmapUpdateChannel(sched.name), node.PoolName + ":owner-epochs",
		node.PoolName + ":job-incarnations", node.PoolName + ":job-cutoffs",
		rmapContentKey(jobPayloadMapName(node.PoolName)), node.ownersKey(),
		rmapContentKey(jobPendingMapName(node.PoolName)), rmapUpdateChannel(jobPendingMapName(node.PoolName)), node.poolStream.Key(),
	}, key, intent.Revision, sched.name, job, strconv.FormatInt(now.UnixNano(), 10),
		strconv.FormatInt(now.Add(2*node.ackGracePeriod).UnixNano(), 10), node.poolStream.MaxLen,
		poolSinkName, now.UnixMilli(), pendingEventTTL.Milliseconds()).Result()
	if err != nil {
		return "", err
	}
	status, value, err := parseDispatchClaim(raw)
	if err != nil {
		return "", err
	}
	if status == dispatchClaimed {
		return value, nil
	}
	if status == dispatchMalformedPending {
		return "", fmt.Errorf("malformed dispatch guard for %q", key)
	}
	return "", nil
}

func (node *Node) completeIntent(ctx context.Context, op *poolOperation, kind string) error {
	if op.producer == "" {
		return nil
	}
	return luaCompleteIntent.Run(ctx, node.rdb, []string{
		rmapContentKey(op.producer), rmapUpdateChannel(op.producer),
	}, op.key, op.revision, op.token, kind).Err()
}
