# TLA+ model of `Sink.AddStream` in `pulse/streaming`

`SinkAddStream.tla` models how instances of one sink add a stream to the sink
and remove it again (issues #481, #508 and #531). Every in-process instance
of a sink has its own consumer. The replicated consumers map of a stream
holds, under the sink name, the consumers of the instances that added the
stream:

- `NewSink` adds the consumer to the map, creates the consumer group and then
  creates the consumer (`XGROUP CREATECONSUMER`), which fails with `NOGROUP`
  when the group is missing. It keeps an existing group (`BUSYGROUP`).
- `Sink.AddStream` adds the consumer to the map and creates the consumer
  group. It keeps an existing group.
- `Sink.RemoveStream` removes the consumer from the map and destroys the group
  when no consumer remains.

Each map update and each group command is one atomic Redis step. The steps of
one call are not atomic together, so the calls of two instances interleave.
Group and consumer creation can fail at any time. The keep-alive update of
`NewSink` fails as the consumer creation does, so the model folds it into that
step.

## Invariants

| Invariant | Meaning |
| --- | --- |
| `NoStaleMember` | The map holds no consumer of an instance that has not added the stream. |
| `AddedHasGroup` | An instance whose `AddStream` or `NewSink` succeeded reads from an existing group. |

## Toggles

| Constant | Meaning |
| --- | --- |
| `CREATE_FIRST` | `TRUE`: `AddStream` creates the group before it adds the consumer to the map. |
| `ROLLBACK` | `TRUE`: `AddStream` removes the consumer from the map again when the group creation fails, and `NewSink` with `NEW_SINK = "map_first"` when the group or the consumer creation fails. |
| `REMOVE` | How `RemoveStream` destroys the group when the map update leaves no consumer. `"split"`: a separate `XGROUP DESTROY` (before #508). `"atomic"`: the map update and the destroy are one step. `"checked"`: a separate script that destroys the group only if the map still holds no consumer under the sink name (`destroyGroupScript` in `sink.go`, the code on `main`). |
| `NEW_SINK` | How `NewSink` orders its steps. `"none"`: no instance calls `NewSink`, the model of `AddStream` and `RemoveStream` only. `"create_first"`: the group, the consumer, then the map (before #531). `"map_first"`: the map, the group, then the consumer, as `AddStream` (the code on `main`). |

## Running TLC

TLC ships in `tla2tools.jar` (tested with TLC 2.19).

```sh
curl -L -o /tmp/tla2tools.jar \
  https://github.com/tlaplus/tlaplus/releases/latest/download/tla2tools.jar
cd pulse/streaming/tla
java -XX:+UseParallelGC -cp /tmp/tla2tools.jar tlc2.TLC \
  -workers 1 -deadlock -metadir /tmp/tlc-states \
  -config cfg/fixed.cfg SinkAddStream.tla
```

`-deadlock` is required: a state with no enabled action is a normal end state.
`-metadir` keeps the `states/` directory out of the source tree. Every
configuration finishes in about a second.

## Results

| Configuration | Result |
| --- | --- |
| `cfg/asis.cfg` | `NoStaleMember` is violated after a 4-state trace: the instance adds its consumer to the map, the group creation fails, and the consumer stays in the map. This is issue #481. |
| `cfg/create_first.cfg` | `AddedHasGroup` is violated after a 9-state trace: instance 2 creates the group (`BUSYGROUP`), instance 1 removes the stream, sees no consumer in the map and destroys the group, then instance 2 adds its consumer and returns with no group. The checked destroy of #508 does not prevent this: the map holds no consumer when it runs. Moving the map update after the group creation is therefore not a fix. |
| `cfg/fixed_nonatomic.cfg` | The fix of #481 with `RemoveStream` before #508 (`REMOVE = "split"`). `NoStaleMember` holds with three instances (424 distinct states). The consumer goes to the map before the group creation, so a concurrent `RemoveStream` sees it and keeps the group, and a failed creation removes it again. `AddedHasGroup` is not checked here because of the `RemoveStream` race below. |
| `cfg/fixed_remove_asis.cfg` | The same design, checking `AddedHasGroup`: it is violated after a 9-state trace, with or without the fix of #481. `RemoveStream` removes the last consumer, another instance adds the stream and finds the existing group, then the first instance destroys the group. The map update and the group destroy of `RemoveStream` are separate calls. This is issue #508. |
| `cfg/fixed_atomic.cfg` | The fix of #481 with `REMOVE = "atomic"`, one candidate fix of #508. Both invariants hold with three instances (189 distinct states). |
| `cfg/newsink_asis.cfg` | `AddStream` and `RemoveStream` with the fixes of #481 and #508, and `NewSink` before #531 (`NEW_SINK = "create_first"`). `AddedHasGroup` is violated after a 10-state trace: instance 1 removes the stream and sees no consumer in the map, instance 2 creates the group (`BUSYGROUP`) and its consumer in `NewSink`, instance 1 destroys the group, then instance 2 adds its consumer to the map and returns with no group. This is issue #531, the race of `cfg/create_first.cfg` in `NewSink`. |
| `cfg/fixed.cfg` | The code on `main`: the fix of #481, the fix of #508 (`REMOVE = "checked"`) and the fix of #531 (`NEW_SINK = "map_first"`). Both invariants hold with three instances (1511 distinct states). Without `NewSink` (`NEW_SINK = "none"`) the same configuration has 340 distinct states. |

The fix of #508 checks the map again instead of making the map update and the
destroy one step. The map update is an rmap script that bumps the map
revision, publishes the change and applies the map TTL. One script that also
destroys the group would have to repeat that protocol outside rmap. The check
needs only the hash key of the map, which holds no field for the sink name
once `RemoveValues` removed its last consumer. An instance that adds the
stream puts its consumer in the map before it creates the group, so the check
sees it and keeps the group. The check reads the map in Redis, not the local
replica, and runs in the same script as `XGROUP DESTROY`.

The fix of #481 does not destroy the group when its rollback leaves no
consumer in the map. A destroy there would reproduce the `RemoveStream` race
above for instances that add the stream at the same time. The group that such
a rollback can leave is the one an earlier instance created. The next
`AddStream` reuses it.

The fix of #531 gives `NewSink` the order of `AddStream`: it adds its new
consumer to the map before it creates the group and the consumer, so a
concurrent `RemoveStream` sees it and keeps the group. When the group
creation, the consumer creation or the keep-alive update fails, `NewSink`
removes the consumer from the map again and keeps the group, as the rollback
of `AddStream` does.
